package automation

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
)

// IndeterminateError means the client cannot establish whether an accepted
// or possibly accepted request executed. Callers must not silently retry it.
type IndeterminateError struct {
	// RequestID identifies the operation that requires reconciliation.
	RequestID string
	// Cause is the local transport, framing, or cancellation failure.
	Cause error
}

// Error describes the indeterminate request and its local cause.
func (e *IndeterminateError) Error() string {
	return fmt.Sprintf("request %q has an indeterminate outcome: %v", e.RequestID, e.Cause)
}

// Unwrap returns the local failure that made the result indeterminate.
func (e *IndeterminateError) Unwrap() error {
	return e.Cause
}

// ResponseError is a structured rejection returned by the automation server.
type ResponseError struct {
	// RequestID is present when the server safely associated the rejection.
	RequestID string
	// Code is the stable protocol rejection code.
	Code string
	// Message is bounded operator-facing text.
	Message string
	// Retryable is advisory and never authorizes automatic request replay.
	Retryable bool
}

// Error formats the structured protocol rejection.
func (e *ResponseError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("automation protocol error %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("automation request %q error %s: %s", e.RequestID, e.Code, e.Message)
}

// Client is a sequential version 1 automation controller. Its methods are
// safe for concurrent callers; calls are serialized because version 1 permits
// one outstanding request per connection, with unspecified acquisition order.
// A Client must not be copied after first use.
//
// Operation methods require a non-nil context and a fresh bounded request ID.
// They use the earlier of the context deadline and advertised protocol
// deadlines. A failure after submission may return IndeterminateError and
// closes the connection; callers must reconcile rather than replay. Returned
// completions and snapshots are caller-owned.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader

	requestMu sync.Mutex
	stateMu   sync.Mutex
	hello     Hello
	closed    bool
}

// Dial connects to a version 1 Unix-socket server and validates its Hello
// before returning. The context covers both steps. On success, the caller owns
// Client.Close. Dial authenticates neither endpoint nor peer.
func Dial(ctx context.Context, socketPath string) (*Client, error) {
	if ctx == nil {
		return nil, errors.New("automation: nil context")
	}
	if socketPath == "" {
		return nil, errors.New("automation: empty socket path")
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("automation: connect: %w", err)
	}

	client := &Client{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, lineReaderBufferBytes),
	}
	if err := client.readHello(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

// NewRequestID returns a cryptographically random 32-character hexadecimal
// request identifier. It does not reserve the ID with a server.
func NewRequestID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("automation: generate request ID: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}

// Hello returns a caller-owned copy of the server's initial capability record,
// including independent inventory slices.
func (c *Client) Hello() Hello {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	hello := c.hello
	hello.SupportedVersions = append([]int{}, hello.SupportedVersions...)
	hello.Operations = append([]string{}, hello.Operations...)
	hello.Commands = append([]string{}, hello.Commands...)
	return hello
}

// Close closes the controller connection. The server clears source-local held
// keys when it observes the disconnect. Close is idempotent and interrupts an
// in-flight operation, which will normally become indeterminate.
func (c *Client) Close() error {
	c.stateMu.Lock()

	if c.closed {
		c.stateMu.Unlock()
		return nil
	}
	c.closed = true
	c.stateMu.Unlock()
	return c.conn.Close()
}

// Observe returns the latest snapshot when frameSequence is nil, or requests
// the exact retained sequence otherwise. It does not retain frameSequence.
func (c *Client) Observe(ctx context.Context, requestID string, frameSequence *uint64) (Completion, error) {
	request := observeRequest{
		Header:        newHeader(TypeObserve),
		RequestID:     requestID,
		FrameSequence: frameSequence,
	}
	return c.do(ctx, requestID, TypeObserve, request)
}

// WaitSnapshot waits for a snapshot whose sequence is greater than
// afterSequence. The wait is bounded by ctx and the advertised server maximum.
func (c *Client) WaitSnapshot(ctx context.Context, requestID string, afterSequence uint64) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("automation: nil context")
	}
	timeoutMillis := 0
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return Completion{}, ctx.Err()
		}
		timeoutMillis = int((remaining + time.Millisecond - 1) / time.Millisecond)
		if timeoutMillis > c.hello.Limits.ReadTimeoutMillis {
			timeoutMillis = c.hello.Limits.ReadTimeoutMillis
		}
	}
	request := waitSnapshotRequest{
		Header:        newHeader(TypeWaitSnapshot),
		RequestID:     requestID,
		AfterSequence: afterSequence,
		TimeoutMillis: timeoutMillis,
	}
	return c.do(ctx, requestID, TypeWaitSnapshot, request)
}

// InjectInput submits one raw logical key lifecycle event through the App's
// normal input and command-resolution path.
func (c *Client) InjectInput(ctx context.Context, requestID string, event KeyEvent) (Completion, error) {
	request := injectInputRequest{
		Header:    newHeader(TypeInjectInput),
		RequestID: requestID,
		Event:     event,
	}
	return c.do(ctx, requestID, TypeInjectInput, request)
}

// InvokeCommand submits a stable semantic command and optional stable
// automation target key through normal application command policy.
func (c *Client) InvokeCommand(ctx context.Context, requestID, command, targetKey string) (Completion, error) {
	request := invokeCommandRequest{
		Header:    newHeader(TypeInvokeCommand),
		RequestID: requestID,
		Command:   command,
		TargetKey: targetKey,
	}
	return c.do(ctx, requestID, TypeInvokeCommand, request)
}

// QueryResult queries an active or retained request by ID. requestID must be a
// fresh ID for the query itself; targetRequestID identifies the earlier
// operation.
func (c *Client) QueryResult(ctx context.Context, requestID, targetRequestID string) (Completion, error) {
	request := queryResultRequest{
		Header:          newHeader(TypeQueryResult),
		RequestID:       requestID,
		TargetRequestID: targetRequestID,
	}
	return c.do(ctx, requestID, TypeQueryResult, request)
}

// ResetInput clears held key state for this controller source through the
// App's correlated input path.
func (c *Client) ResetInput(ctx context.Context, requestID string) (Completion, error) {
	request := simpleRequest{
		Header:    newHeader(TypeResetInput),
		RequestID: requestID,
	}
	return c.do(ctx, requestID, TypeResetInput, request)
}

// Shutdown requests orderly application exit through the normal command
// policy. The resulting Outcome reports whether policy actually exited.
func (c *Client) Shutdown(ctx context.Context, requestID string) (Completion, error) {
	request := simpleRequest{
		Header:    newHeader(TypeShutdown),
		RequestID: requestID,
	}
	return c.do(ctx, requestID, TypeShutdown, request)
}

func (c *Client) readHello(ctx context.Context) error {
	limits := DefaultLimits()
	if err := setReadDeadline(c.conn, ctx, limits.readTimeout()); err != nil {
		return err
	}
	stopReadInterrupt := context.AfterFunc(ctx, func() {
		_ = c.conn.SetReadDeadline(time.Now())
	})
	defer stopReadInterrupt()
	line, err := readLine(c.conn, c.reader, limits.ResponseLineBytes)
	if err != nil {
		return fmt.Errorf("automation: read hello: %w", err)
	}
	if !utf8Record(line) {
		return errors.New("automation: hello is not valid UTF-8")
	}
	header, err := responseHeader(line, limits)
	if err != nil {
		return fmt.Errorf("automation: invalid hello envelope: %w", err)
	}
	if header.Type == TypeProtocolError {
		failure, decodeErr := decodeResponseError(line)
		if decodeErr != nil {
			return fmt.Errorf("automation: decode connection rejection: %w", decodeErr)
		}
		return failure
	}
	// Hello is the v1-compatible capability preface. Its line and JSON depth
	// are bounded above, but unknown additive fields are deliberately ignored
	// so a future server can advertise compatible capabilities. Operational
	// records continue to use decodeStrict.
	hello, err := decodeHelloRecord(line)
	if err != nil {
		return err
	}
	c.hello = hello
	return nil
}

var requiredHelloFields = [...]string{
	"protocol",
	"version",
	"type",
	"application",
	"session_id",
	"unauthenticated",
	"supported_versions",
	"scenario",
	"latest_frame_sequence",
	"final",
	"operations",
	"commands",
	"limits",
}

func decodeHelloRecord(line []byte) (Hello, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return Hello{}, fmt.Errorf("automation: decode hello fields: %w", err)
	}
	for _, name := range requiredHelloFields {
		value, found := fields[name]
		if !found {
			return Hello{}, fmt.Errorf("automation: hello omits required field %q", name)
		}
		if isJSONNull(value) {
			return Hello{}, fmt.Errorf("automation: hello field %q must not be null", name)
		}
	}

	var hello Hello
	if err := json.Unmarshal(line, &hello); err != nil {
		return Hello{}, fmt.Errorf("automation: decode hello: %w", err)
	}
	if err := validateResponseHeader(hello.Header, TypeHello); err != nil {
		return Hello{}, err
	}
	if err := validateHello(hello); err != nil {
		return Hello{}, err
	}
	return hello, nil
}

func validateHello(hello Hello) error {
	if err := hello.Limits.validate(); err != nil {
		return fmt.Errorf("automation: invalid advertised limits: %w", err)
	}
	if !hello.Unauthenticated {
		return errors.New("automation: protocol version 1 must identify unauthenticated mode")
	}
	if !validIdentifier(hello.SessionID, hello.Limits.IdentifierBytes) ||
		!validIdentifier(hello.Application, hello.Limits.IdentifierBytes) {
		return errors.New("automation: hello omits or invalidates application or session identity")
	}
	if !validIdentifier(hello.Scenario, hello.Limits.IdentifierBytes) {
		return errors.New("automation: hello has an invalid scenario identity")
	}
	if hello.LatestFrameSequence == 0 {
		return errors.New("automation: hello has no current frame sequence")
	}
	if len(hello.SupportedVersions) == 0 || len(hello.SupportedVersions) > 16 {
		return errors.New("automation: hello has an invalid supported-version inventory")
	}
	supported := false
	seenVersions := make(map[int]struct{}, len(hello.SupportedVersions))
	for _, version := range hello.SupportedVersions {
		if version <= 0 || version > 65_535 {
			return errors.New("automation: hello has an invalid supported protocol version")
		}
		if _, exists := seenVersions[version]; exists {
			return errors.New("automation: hello repeats a supported protocol version")
		}
		seenVersions[version] = struct{}{}
		supported = supported || version == Version
	}
	if !supported {
		return errors.New("automation: hello does not advertise protocol version 1")
	}
	if len(hello.Operations) != len(versionOneOperations) ||
		len(hello.Commands) > 64 {
		return errors.New("automation: hello capability inventory exceeds bounds")
	}
	requiredOperations := make(map[string]struct{}, len(versionOneOperations))
	for _, operation := range versionOneOperations {
		requiredOperations[operation] = struct{}{}
	}
	seenOperations := make(map[string]struct{}, len(hello.Operations))
	for _, operation := range hello.Operations {
		if _, required := requiredOperations[operation]; !required {
			return errors.New("automation: hello advertises an unknown operation")
		}
		if _, exists := seenOperations[operation]; exists {
			return errors.New("automation: hello repeats an operation")
		}
		seenOperations[operation] = struct{}{}
	}
	if len(seenOperations) != len(requiredOperations) {
		return errors.New("automation: hello omits a required operation")
	}
	seenCommands := make(map[string]struct{}, len(hello.Commands))
	for _, command := range hello.Commands {
		if !validIdentifier(command, hello.Limits.IdentifierBytes) {
			return errors.New("automation: hello contains an invalid command identifier")
		}
		if _, exists := seenCommands[command]; exists {
			return errors.New("automation: hello repeats a command identifier")
		}
		seenCommands[command] = struct{}{}
	}
	return nil
}

func (c *Client) do(ctx context.Context, requestID, operation string, request any) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("automation: nil context")
	}

	c.requestMu.Lock()
	defer c.requestMu.Unlock()

	c.stateMu.Lock()
	closed := c.closed
	c.stateMu.Unlock()
	if closed {
		return Completion{}, net.ErrClosed
	}
	if err := validateRequestID(requestID, c.hello.Limits); err != nil {
		return Completion{}, fmt.Errorf("automation: %w", err)
	}

	line, err := encodeLine(request, c.hello.Limits.RequestLineBytes)
	if err != nil {
		return Completion{}, err
	}
	if err := setWriteDeadline(c.conn, ctx, c.hello.Limits.writeTimeout()); err != nil {
		return Completion{}, err
	}
	stopWriteInterrupt := context.AfterFunc(ctx, func() {
		_ = c.conn.SetWriteDeadline(time.Now())
	})
	if err := writeAll(c.conn, line); err != nil {
		stopWriteInterrupt()
		c.closeAfterError()
		return Completion{}, &IndeterminateError{RequestID: requestID, Cause: err}
	}
	stopWriteInterrupt()

	postWriteFailure := func(cause error) (Completion, error) {
		c.closeAfterError()
		return Completion{}, &IndeterminateError{
			RequestID: requestID,
			Cause:     cause,
		}
	}

	accepted, protocolFailure, err := c.readAccepted(ctx)
	if err != nil {
		return postWriteFailure(err)
	}
	if protocolFailure != nil {
		if protocolFailure.RequestID == requestID {
			return Completion{}, protocolFailure
		}
		return postWriteFailure(fmt.Errorf(
			"automation: uncorrelated protocol rejection for request %q",
			requestID,
		))
	}
	if accepted.RequestID != requestID || accepted.Operation != operation {
		return postWriteFailure(fmt.Errorf(
			"automation: mismatched acceptance for request %q",
			requestID,
		))
	}

	completion, protocolFailure, err := c.readCompletion(ctx)
	if err != nil {
		return postWriteFailure(err)
	}
	if protocolFailure != nil {
		return postWriteFailure(fmt.Errorf(
			"automation: accepted request received protocol error: %w",
			protocolFailure,
		))
	}
	if completion.RequestID != requestID || completion.Operation != operation {
		return postWriteFailure(fmt.Errorf(
			"automation: mismatched completion for request %q",
			requestID,
		))
	}
	if completion.Snapshot == nil {
		return postWriteFailure(fmt.Errorf(
			"automation: completion for request %q omits snapshot",
			requestID,
		))
	}
	if completion.Snapshot.Sequence != completion.FrameSequence {
		return postWriteFailure(fmt.Errorf(
			"automation: completion snapshot sequence mismatch for request %q",
			requestID,
		))
	}
	if err := validateCompletion(completion, c.hello.Limits); err != nil {
		return postWriteFailure(fmt.Errorf(
			"automation: invalid completion for request %q: %w",
			requestID,
			err,
		))
	}
	return completion, nil
}

func (c *Client) closeAfterError() {
	c.stateMu.Lock()
	alreadyClosed := c.closed
	c.closed = true
	c.stateMu.Unlock()
	if !alreadyClosed {
		_ = c.conn.Close()
	}
}

func (c *Client) readAccepted(ctx context.Context) (Accepted, *ResponseError, error) {
	line, err := c.readResponseLine(ctx)
	if err != nil {
		return Accepted{}, nil, err
	}
	header, err := responseHeader(line, c.hello.Limits)
	if err != nil {
		return Accepted{}, nil, err
	}
	switch header.Type {
	case TypeAccepted:
		var accepted Accepted
		if err := decodeStrict(line, &accepted); err != nil {
			return Accepted{}, nil, err
		}
		return accepted, nil, nil
	case TypeProtocolError:
		failure, err := decodeResponseError(line)
		return Accepted{}, failure, err
	default:
		return Accepted{}, nil, fmt.Errorf("expected accepted or protocol_error, got %q", header.Type)
	}
}

func (c *Client) readCompletion(ctx context.Context) (Completion, *ResponseError, error) {
	line, err := c.readResponseLine(ctx)
	if err != nil {
		return Completion{}, nil, err
	}
	header, err := responseHeader(line, c.hello.Limits)
	if err != nil {
		return Completion{}, nil, err
	}
	switch header.Type {
	case TypeCompletion:
		var completion Completion
		if err := decodeStrict(line, &completion); err != nil {
			return Completion{}, nil, err
		}
		return completion, nil, nil
	case TypeProtocolError:
		failure, err := decodeResponseError(line)
		return Completion{}, failure, err
	default:
		return Completion{}, nil, fmt.Errorf("expected completion, got %q", header.Type)
	}
}

func (c *Client) readResponseLine(ctx context.Context) ([]byte, error) {
	if err := setReadDeadline(c.conn, ctx, c.hello.Limits.readTimeout()); err != nil {
		return nil, err
	}
	stopReadInterrupt := context.AfterFunc(ctx, func() {
		_ = c.conn.SetReadDeadline(time.Now())
	})
	defer stopReadInterrupt()
	line, err := readLine(c.conn, c.reader, c.hello.Limits.ResponseLineBytes)
	if err != nil {
		return nil, err
	}
	if !utf8Record(line) {
		return nil, errors.New("response is not valid UTF-8")
	}
	return line, nil
}

func responseHeader(line []byte, limits Limits) (Header, error) {
	if err := validateJSONStructure(line, limits.JSONDepth); err != nil {
		return Header{}, err
	}
	var header Header
	if err := json.Unmarshal(line, &header); err != nil {
		return Header{}, err
	}
	if header.Protocol != Protocol || header.Version != Version {
		return Header{}, errors.New("response uses unsupported protocol or version")
	}
	return header, nil
}

func decodeResponseError(line []byte) (*ResponseError, error) {
	var failure ProtocolError
	if err := decodeStrict(line, &failure); err != nil {
		return nil, err
	}
	return &ResponseError{
		RequestID: failure.RequestID,
		Code:      failure.Code,
		Message:   failure.Message,
		Retryable: failure.Retryable,
	}, nil
}

func validateResponseHeader(header Header, expectedType string) error {
	if header.Protocol != Protocol {
		return errors.New("automation: unsupported protocol")
	}
	if header.Version != Version {
		return errors.New("automation: unsupported protocol version")
	}
	if header.Type != expectedType {
		return fmt.Errorf("automation: expected %q, got %q", expectedType, header.Type)
	}
	return nil
}

func validateCompletion(completion Completion, limits Limits) error {
	if !validIdentifier(completion.RequestID, limits.RequestIDBytes) ||
		!validIdentifier(completion.Operation, limits.IdentifierBytes) {
		return errors.New("completion has invalid identity metadata")
	}
	if !validOutcome(completion.Outcome) {
		return fmt.Errorf("unknown outcome %q", completion.Outcome)
	}
	if completion.FrameSequence == 0 {
		return errors.New("frame sequence must be positive")
	}
	snapshot := completion.Snapshot
	if snapshot == nil {
		return errors.New("snapshot is required")
	}
	if err := validateSnapshot(snapshot, limits); err != nil {
		return err
	}
	if completion.Error != nil &&
		!validCompletionError(completion.Error, limits) {
		return errors.New("completion error exceeds advertised bounds")
	}
	if completion.Operation == TypeQueryResult {
		if completion.Result == nil || completion.Result.Query == nil {
			return errors.New("query_result completion omits query result")
		}
	} else if completion.Result != nil {
		return errors.New("non-query completion includes operation-specific result")
	}
	if completion.Result != nil {
		query := completion.Result.Query
		if !validIdentifier(query.TargetRequestID, limits.RequestIDBytes) {
			return errors.New("query result has an invalid target request ID")
		}
		switch query.Status {
		case "pending", "completed", "unknown_or_expired":
		default:
			return fmt.Errorf("unknown query status %q", query.Status)
		}
		if query.Status == "completed" {
			if query.Completion == nil {
				return errors.New("completed query omits retained completion")
			}
			if !validIdentifier(query.Completion.RequestID, limits.RequestIDBytes) ||
				!validIdentifier(query.Completion.Operation, limits.IdentifierBytes) {
				return errors.New("retained completion has invalid identity metadata")
			}
			if !validOutcome(query.Completion.Outcome) {
				return errors.New("retained completion has unknown outcome")
			}
			if query.Completion.FrameSequence == 0 {
				return errors.New("retained completion has invalid frame sequence")
			}
			if query.Completion.FrameSequence != completion.FrameSequence {
				return errors.New("query result does not carry the exact retained snapshot")
			}
			if query.Completion.Error != nil &&
				!validCompletionError(query.Completion.Error, limits) {
				return errors.New("retained completion error exceeds advertised bounds")
			}
		} else if query.Completion != nil {
			return errors.New("non-completed query includes a retained completion")
		}
	}
	return nil
}

func validateSnapshot(snapshot *SnapshotV1, limits Limits) error {
	if snapshot == nil {
		return errors.New("snapshot is required")
	}
	if snapshot.Version != 1 {
		return fmt.Errorf("unsupported SnapshotV1 version %d", snapshot.Version)
	}
	if snapshot.Sequence == 0 {
		return errors.New("snapshot sequence must be positive")
	}
	if !validIdentifier(snapshot.Scenario, limits.IdentifierBytes) {
		return errors.New("snapshot scenario is not a bounded identifier")
	}
	width := snapshot.Frame.Size.Width
	height := snapshot.Frame.Size.Height
	if width < 0 || height < 0 ||
		width > limits.FrameWidth || height > limits.FrameHeight {
		return errors.New("snapshot geometry exceeds advertised bounds")
	}
	if height != 0 && width > limits.FrameCells/height {
		return errors.New("snapshot cell count exceeds advertised bound")
	}
	cellCount := width * height
	if cellCount > limits.FrameCells {
		return errors.New("snapshot cell count exceeds advertised bound")
	}
	if err := expandFrame(&snapshot.Frame, cellCount, limits.FrameRuns); err != nil {
		return err
	}
	for _, cell := range snapshot.Frame.Cells {
		if len(cell.Grapheme) > maxCellGraphemeBytes ||
			!canonicalCell(cell.Grapheme) {
			return errors.New("snapshot cell grapheme exceeds bounds")
		}
		if !validIdentifier(string(cell.Style), limits.IdentifierBytes) ||
			!validIdentifier(string(cell.Owner), limits.IdentifierBytes) ||
			!validColor(cell.Foreground) ||
			!validColor(cell.Background) ||
			!validAttributes(cell.Attributes) {
			return errors.New("snapshot cell has invalid bounded style or owner data")
		}
	}
	if snapshot.Cursor.Visible &&
		(snapshot.Cursor.Position.X < 0 ||
			snapshot.Cursor.Position.Y < 0 ||
			snapshot.Cursor.Position.X >= width ||
			snapshot.Cursor.Position.Y >= height) {
		return errors.New("visible snapshot cursor is outside the frame")
	}
	if len(snapshot.Controls) > limits.Controls {
		return errors.New("snapshot control count exceeds advertised bound")
	}
	childCount := 0
	for _, control := range snapshot.Controls {
		if !validIdentifier(string(control.ID), limits.IdentifierBytes) ||
			(control.Key != "" && !validIdentifier(control.Key, limits.IdentifierBytes)) ||
			!validIdentifier(string(control.Kind), limits.IdentifierBytes) ||
			(control.Parent != "" &&
				!validIdentifier(string(control.Parent), limits.IdentifierBytes)) ||
			!validIdentifier(string(control.Style), limits.IdentifierBytes) ||
			!validResolvedStyle(control.ResolvedStyle) ||
			control.Bounds.Width < 0 ||
			control.Bounds.Height < 0 ||
			control.AbsoluteBounds.Width < 0 ||
			control.AbsoluteBounds.Height < 0 ||
			control.EffectiveClip.Width < 0 ||
			control.EffectiveClip.Height < 0 ||
			control.Minimum.Width < 0 ||
			control.Minimum.Height < 0 ||
			(control.Layout != "" &&
				(!validIdentifier(string(control.Layout), limits.IdentifierBytes) ||
					control.LayoutIndex < 0 ||
					control.StackIndex < 0 ||
					control.LayoutIndex >= limits.LayoutItems ||
					control.StackIndex >= limits.LayoutItems)) ||
			(control.Layout == "" &&
				(control.LayoutIndex != -1 || control.StackIndex != -1)) {
			return errors.New("snapshot control has invalid bounded identity, geometry, or style")
		}
		if len(control.Children) > limits.Controls-childCount {
			return errors.New("snapshot child-reference count exceeds advertised bound")
		}
		childCount += len(control.Children)
		for _, child := range control.Children {
			if !validIdentifier(string(child), limits.IdentifierBytes) {
				return errors.New("snapshot control child has an invalid identity")
			}
		}
		if control.Details.Version != 1 {
			return errors.New("snapshot control details use an unsupported version")
		}
		if control.Details.Border != nil {
			if !validBorderDetails(control.Details.Border, limits) {
				return errors.New("snapshot border details exceed bounds")
			}
		}
	}
	if len(snapshot.Layouts) > limits.Layouts {
		return errors.New("snapshot Layout count exceeds advertised bound")
	}
	layoutItemCount := 0
	for _, layout := range snapshot.Layouts {
		if !validIdentifier(string(layout.ID), limits.IdentifierBytes) ||
			(layout.Key != "" &&
				!validIdentifier(layout.Key, limits.IdentifierBytes)) ||
			!validIdentifier(string(layout.Kind), limits.IdentifierBytes) ||
			!validIdentifier(string(layout.Owner), limits.IdentifierBytes) ||
			(layout.Parent != "" &&
				!validIdentifier(string(layout.Parent), limits.IdentifierBytes)) ||
			layout.Bounds.Width < 0 || layout.Bounds.Height < 0 ||
			layout.OwnerBounds.Width < 0 || layout.OwnerBounds.Height < 0 ||
			layout.Minimum.Width < 0 || layout.Minimum.Height < 0 ||
			layout.LayoutIndex < 0 || layout.StackIndex < 0 ||
			!validBorderDetails(layout.Border, limits) {
			return errors.New("snapshot Layout has invalid bounded data")
		}
		if len(layout.Items) > limits.LayoutItems-layoutItemCount {
			return errors.New("snapshot Layout item count exceeds advertised bound")
		}
		layoutItemCount += len(layout.Items)
		stackIndices := make(map[int]bool, len(layout.Items))
		for itemIndex, item := range layout.Items {
			if (item.Kind != "panel" && item.Kind != "layout") ||
				(item.Kind == "panel" &&
					(!validIdentifier(string(item.Panel), limits.IdentifierBytes) ||
						item.Layout != "")) ||
				(item.Kind == "layout" &&
					(!validIdentifier(string(item.Layout), limits.IdentifierBytes) ||
						item.Panel != "")) ||
				item.Bounds.Width < 0 || item.Bounds.Height < 0 ||
				item.Minimum.Width < 0 || item.Minimum.Height < 0 ||
				item.LayoutIndex != itemIndex ||
				item.StackIndex < 0 ||
				item.StackIndex >= len(layout.Items) ||
				stackIndices[item.StackIndex] {
				return errors.New("snapshot Layout item has invalid bounded data")
			}
			stackIndices[item.StackIndex] = true
		}
	}
	if len(snapshot.InputSources) > limits.Controls {
		return errors.New("snapshot input-source count exceeds advertised bound")
	}
	for _, source := range snapshot.InputSources {
		if !validIdentifier(source.Source, limits.IdentifierBytes) {
			return errors.New("snapshot input source is not a bounded identifier")
		}
		if len(source.Held) > limits.HeldKeys {
			return errors.New("snapshot held-key count exceeds advertised bound")
		}
		for _, key := range source.Held {
			if !validLogicalKey(string(key)) {
				return errors.New("snapshot held-key identity is invalid")
			}
		}
	}
	if len(snapshot.Overflows) > limits.Controls {
		return errors.New("snapshot overflow count exceeds advertised bound")
	}
	for _, overflow := range snapshot.Overflows {
		if !validIdentifier(overflow.EpisodeID, limits.IdentifierBytes) ||
			!validIdentifier(string(overflow.Panel), limits.IdentifierBytes) ||
			!validIdentifier(string(overflow.Layout), limits.IdentifierBytes) ||
			!validIdentifier(overflow.State, limits.IdentifierBytes) ||
			overflow.Available.Width < 0 ||
			overflow.Available.Height < 0 ||
			overflow.Required.Width < 0 ||
			overflow.Required.Height < 0 ||
			overflow.Deficit.Width < 0 ||
			overflow.Deficit.Height < 0 {
			return errors.New("snapshot overflow has invalid bounded data")
		}
	}
	if snapshot.Completion != nil {
		association := snapshot.Completion
		if !validIdentifier(association.RequestID, limits.RequestIDBytes) ||
			!validOutcome(association.Outcome) ||
			(association.Command != "" &&
				!validIdentifier(association.Command, limits.IdentifierBytes)) ||
			(association.Code != "" &&
				!validIdentifier(association.Code, limits.IdentifierBytes)) ||
			len(association.Message) > maxSnapshotMessageBytes ||
			!utf8.ValidString(association.Message) ||
			strings.ContainsRune(association.Message, 0) ||
			association.FrameSequence != snapshot.Sequence {
			return errors.New("snapshot completion association has invalid bounded data")
		}
	}
	return nil
}

func validBorderForm(form string) bool {
	switch form {
	case "none", "single", "double", "shade_light", "shade_medium",
		"shade_dark", "block":
		return true
	default:
		return false
	}
}

func validBorderDetails(border *BorderDetails, limits Limits) bool {
	return border != nil &&
		len(border.Title) <= maxBorderTitleBytes &&
		canonicalTitle(border.Title) &&
		validBorderForm(border.Form) &&
		validIdentifier(string(border.Style), limits.IdentifierBytes) &&
		validResolvedStyle(border.ResolvedStyle) &&
		(border.ForegroundOverride == nil ||
			validColor(*border.ForegroundOverride)) &&
		(border.BackgroundOverride == nil ||
			validColor(*border.BackgroundOverride))
}

func validCompletionError(issue *Error, limits Limits) bool {
	return validIdentifier(issue.Code, limits.IdentifierBytes) &&
		len(issue.Message) <= maxErrorMessageBytes &&
		utf8.ValidString(issue.Message) &&
		!strings.ContainsRune(issue.Message, 0)
}

func canonicalCell(value string) bool {
	cells := display.Normalize(value)
	return len(cells) == 1 && cells[0] == value
}

func canonicalTitle(title string) bool {
	cells := display.Normalize(title)
	if len(cells) > maxBorderTitleCells {
		return false
	}
	for _, cell := range cells {
		if len(cell) > maxCellGraphemeBytes {
			return false
		}
	}
	return strings.Join(cells, "") == title
}

func validResolvedStyle(style ResolvedStyle) bool {
	return validColor(style.Foreground) &&
		validColor(style.Background) &&
		validAttributes(style.Attributes)
}

func validAttributes(attributes StyleAttributes) bool {
	const supported StyleAttributes = 1 | 2 | 4 | 8 | 16
	return attributes&^supported == 0
}

func validColor(color Color) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, digit := range color[1:] {
		if (digit >= '0' && digit <= '9') ||
			(digit >= 'a' && digit <= 'f') ||
			(digit >= 'A' && digit <= 'F') {
			continue
		}
		return false
	}
	return true
}

func validOutcome(outcome string) bool {
	switch outcome {
	case OutcomeApplied, OutcomeNoOp, OutcomeRejected, OutcomeCancelled,
		OutcomeInterrupted, OutcomeExited, OutcomeFailed:
		return true
	default:
		return false
	}
}

func setReadDeadline(conn net.Conn, ctx context.Context, fallback time.Duration) error {
	deadline := time.Now().Add(fallback)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return conn.SetReadDeadline(deadline)
}

func setWriteDeadline(conn net.Conn, ctx context.Context, fallback time.Duration) error {
	deadline := time.Now().Add(fallback)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return conn.SetWriteDeadline(deadline)
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

func utf8Record(line []byte) bool {
	return utf8.Valid(line)
}
