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
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
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
	actionItemCount := 0
	menuItemCount := 0
	menuBarCount := 0
	statusBarCount := 0
	selectionItemCount := 0
	textInputBytes := 0
	contentBytes := 0
	collectionBytes := 0
	controlsByID := make(map[ControlID]*ControlSnapshot, len(snapshot.Controls))
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
		if controlsByID[control.ID] != nil {
			return errors.New("snapshot control identity is duplicated")
		}
		controlCopy := control
		controlsByID[control.ID] = &controlCopy
		if len(control.Children) > limits.Controls-childCount {
			return errors.New("snapshot child-reference count exceeds advertised bound")
		}
		childCount += len(control.Children)
		for _, child := range control.Children {
			if !validIdentifier(string(child), limits.IdentifierBytes) {
				return errors.New("snapshot control child has an invalid identity")
			}
		}
		if control.Kind == "status_bar" {
			expected := Rect{}
			if width > 0 && height > 0 {
				expected = Rect{
					Y: height - 1, Width: width, Height: 1,
				}
			}
			if control.Bounds != expected ||
				control.AbsoluteBounds != expected {
				return errors.New(
					"snapshot StatusBar does not have derived surface bounds",
				)
			}
		}
		if !validControlDetails(
			control.Kind,
			control.Details,
			control.Bounds.Width,
			control.Bounds.Height,
			limits,
		) {
			return fmt.Errorf(
				"snapshot control %q (%s) details are invalid for its kind",
				control.Key,
				control.Kind,
			)
		}
		if control.Details.HotkeyBar != nil {
			if len(control.Details.HotkeyBar.Items) >
				expletives.MaxHotkeyBarItems ||
				len(control.Details.HotkeyBar.Items) >
					limits.Controls-actionItemCount {
				return errors.New(
					"snapshot Action item count exceeds advertised bound",
				)
			}
			actionItemCount += len(control.Details.HotkeyBar.Items)
		}
		if control.Details.MenuBar != nil {
			menuBarCount++
			if menuBarCount > 1 ||
				len(control.Details.MenuBar.Entries) >
					expletives.MaxMenuItems-menuItemCount {
				return errors.New(
					"snapshot MenuBar count or item count exceeds bound",
				)
			}
			menuItemCount += len(control.Details.MenuBar.Entries)
		}
		if control.Details.RadioGroup != nil {
			selectionItemCount += len(control.Details.RadioGroup.Options)
		}
		if control.Details.ChoiceField != nil {
			selectionItemCount += len(control.Details.ChoiceField.Options)
		}
		if control.Details.TabbedPanel != nil {
			selectionItemCount += len(control.Details.TabbedPanel.Tabs)
		}
		if selectionItemCount > expletives.MaxSelectionItems {
			return errors.New(
				"snapshot Selection item count exceeds advertised bound",
			)
		}
		if control.Details.TextField != nil {
			textInputBytes += len(control.Details.TextField.Text)
			if control.Details.TextField.Validator != nil {
				textInputBytes += len(
					control.Details.TextField.Validator.Characters,
				)
			}
			if textInputBytes > expletives.MaxTextInputAggregateBytes {
				return errors.New(
					"snapshot TextField data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.NumberField != nil {
			textInputBytes += len(control.Details.NumberField.Text)
			if textInputBytes > expletives.MaxTextInputAggregateBytes {
				return errors.New(
					"snapshot numeric input data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.TextArea != nil {
			textInputBytes += len(control.Details.TextArea.Text)
			if control.Details.TextArea.Validator != nil {
				textInputBytes += len(
					control.Details.TextArea.Validator.Characters,
				)
			}
			if textInputBytes > expletives.MaxTextInputAggregateBytes {
				return errors.New(
					"snapshot TextArea data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.Markdown != nil {
			contentBytes += control.Details.Markdown.SourceBytes
			if contentBytes > expletives.MaxContentAggregateBytes {
				return errors.New(
					"snapshot Markdown data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.LogView != nil {
			contentBytes += control.Details.LogView.RetainedBytes
			if contentBytes > expletives.MaxContentAggregateBytes {
				return errors.New(
					"snapshot LogView data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.StreamView != nil {
			contentBytes += control.Details.StreamView.RetainedBytes +
				control.Details.StreamView.PendingStorageBytes
			if contentBytes > expletives.MaxContentAggregateBytes {
				return errors.New(
					"snapshot StreamView data exceeds advertised aggregate bound",
				)
			}
		}
		if control.Details.ListBox != nil {
			retained := control.Details.ListBox.RetainedBytes
			if retained > expletives.MaxCollectionAggregateBytes-collectionBytes {
				return errors.New(
					"snapshot collection data exceeds advertised aggregate bound",
				)
			}
			collectionBytes += retained
		}
		if control.Details.StatusBar != nil {
			statusBarCount++
			if statusBarCount > 1 ||
				len(control.Details.StatusBar.Segments) >
					expletives.MaxStatusBarSegments ||
				len(control.Details.StatusBar.Segments) >
					limits.Controls-actionItemCount {
				return errors.New(
					"snapshot StatusBar count or segment count exceeds bound",
				)
			}
			actionItemCount += len(control.Details.StatusBar.Segments)
		}
	}
	for _, control := range snapshot.Controls {
		if control.Details.TabbedPanel == nil {
			continue
		}
		for _, tab := range control.Details.TabbedPanel.Tabs {
			page := controlsByID[tab.Page]
			if page == nil ||
				page.Parent != control.ID ||
				page.Kind != "panel" ||
				page.Layout != "" ||
				page.Bounds != (Rect{
					Width:  max(0, control.Bounds.Width-2),
					Height: max(0, control.Bounds.Height-2),
				}) ||
				(!tab.Selected && page.Visible) {
				return errors.New(
					"snapshot Tab page relationship is invalid",
				)
			}
		}
	}
	for _, control := range snapshot.Controls {
		scrollable := control.Details.Scrollable
		if scrollable == nil {
			continue
		}
		content := controlsByID[scrollable.Content]
		if content == nil ||
			content.Parent != control.ID ||
			content.Kind != "panel" ||
			content.Layout != "" ||
			content.Key != scrollable.ContentKey ||
			content.Bounds != (Rect{
				X:      -scrollable.State.Offset.X,
				Y:      -scrollable.State.Offset.Y,
				Width:  scrollable.State.ContentSize.Width,
				Height: scrollable.State.ContentSize.Height,
			}) ||
			len(control.Children) != 1 ||
			control.Children[0] != content.ID {
			return errors.New(
				"snapshot scroll Content relationship is invalid",
			)
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

func validControlDetails(
	kind ControlKind,
	details ControlDetails,
	controlWidth int,
	controlHeight int,
	limits Limits,
) bool {
	if details.Version != 1 {
		return false
	}
	specialMembers := 0
	if details.Checkbox != nil {
		specialMembers++
	}
	if details.RadioButton != nil {
		specialMembers++
	}
	if details.RadioGroup != nil {
		specialMembers++
	}
	if details.ChoiceField != nil {
		specialMembers++
	}
	if details.FocusGuideBar != nil {
		specialMembers++
	}
	if details.TextField != nil {
		specialMembers++
	}
	if details.NumberField != nil {
		specialMembers++
	}
	if details.TextArea != nil {
		specialMembers++
	}
	if details.Progress != nil {
		specialMembers++
	}
	if details.ScrollBar != nil {
		specialMembers++
	}
	if details.TabbedPanel != nil {
		specialMembers++
	}
	if details.Scrollable != nil {
		specialMembers++
	}
	if details.Markdown != nil {
		specialMembers++
	}
	if details.LogView != nil {
		specialMembers++
	}
	if details.StreamView != nil {
		specialMembers++
	}
	if details.ListBox != nil {
		specialMembers++
	}
	switch kind {
	case "root", "panel", "header", "footer":
		return specialMembers == 0 &&
			details.Container != nil &&
			details.Container.ClientInset == 0 &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "frame", "group_box":
		return specialMembers == 0 &&
			details.Container != nil &&
			validBorderDetails(details.Border, limits) &&
			((details.Border.Form == "none" &&
				details.Container.ClientInset == 0) ||
				(details.Border.Form != "none" &&
					details.Container.ClientInset == 1)) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "label":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			validTextDetails(details.Text, limits, false) &&
			details.Text.Wrap == "none" &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "static_text":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			validTextDetails(details.Text, limits, true) &&
			details.Text.Target == "" &&
			details.Text.Mnemonic == "" &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "separator":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			validDividerDetails(details.Divider) &&
			details.Divider.Text == "" &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "rule":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			validDividerDetails(details.Divider) &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "button":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			validActionDetails(details.Action, limits) &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "hotkey_bar":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			validHotkeyBarDetails(details.HotkeyBar, limits) &&
			details.MenuBar == nil &&
			details.StatusBar == nil
	case "menu_bar":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			validMenuBarDetails(details.MenuBar, limits) &&
			details.StatusBar == nil
	case "status_bar":
		return specialMembers == 0 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			validStatusBarDetails(details.StatusBar, controlWidth, limits)
	case "checkbox":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validCheckboxDetails(details.Checkbox, limits)
	case "radio_button":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validRadioButtonDetails(details.RadioButton, limits)
	case "radio_group":
		return specialMembers == 1 &&
			details.Container != nil &&
			details.Container.ClientInset == 0 &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validRadioGroupDetails(details.RadioGroup, limits)
	case "cycle_field", "select_field":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validChoiceFieldDetails(details.ChoiceField, limits)
	case "focus_guide_bar":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validFocusGuideBarDetails(details.FocusGuideBar, limits)
	case "text_field":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validTextFieldDetails(details.TextField, limits)
	case "number_field", "spin_box":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validNumberFieldDetails(
				details.NumberField,
				kind == "spin_box",
				limits,
			)
	case "text_area":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validTextAreaDetails(details.TextArea, controlWidth, limits)
	case "progress_bar", "meter", "spinner", "activity_dots":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validProgressDetails(
				kind,
				details.Progress,
				controlWidth,
			)
	case "scroll_bar":
		return specialMembers == 1 &&
			details.Container == nil &&
			details.Border == nil &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validScrollBarDetails(
				details.ScrollBar,
				controlWidth,
				controlHeight,
				limits,
			)
	case "tabbed_panel", "notebook":
		return specialMembers == 1 &&
			details.Container != nil &&
			details.Container.ClientInset == 1 &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validTabbedPanelDetails(
				details.TabbedPanel,
				controlWidth,
				limits,
			)
	case "viewport", "scrollable_panel":
		return specialMembers == 1 &&
			details.Container != nil &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validScrollableDetails(
				kind,
				details.Container,
				details.Border,
				details.Scrollable,
				controlWidth,
				controlHeight,
				limits,
			)
	case "markdown_view":
		return specialMembers == 1 &&
			details.Container == nil &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validMarkdownDetails(
				details.Markdown,
				details.Border,
				controlWidth,
				controlHeight,
			)
	case "log_view":
		return specialMembers == 1 &&
			details.Container == nil &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validLogViewDetails(
				details.LogView,
				details.Border,
				controlWidth,
				controlHeight,
				limits,
			)
	case "stream_view":
		return specialMembers == 1 &&
			details.Container == nil &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validStreamViewDetails(
				details.StreamView,
				details.Border,
				controlWidth,
				controlHeight,
			)
	case "list_box":
		return specialMembers == 1 &&
			details.Container == nil &&
			validBorderDetails(details.Border, limits) &&
			details.Text == nil &&
			details.Divider == nil &&
			details.Action == nil &&
			details.HotkeyBar == nil &&
			details.MenuBar == nil &&
			details.StatusBar == nil &&
			validListBoxDetails(
				details.ListBox,
				details.Border,
				controlWidth,
				controlHeight,
				limits,
			)
	default:
		return false
	}
}

func validListBoxDetails(
	details *ListBoxDetails,
	border *BorderDetails,
	controlWidth int,
	controlHeight int,
	limits Limits,
) bool {
	if details == nil || border == nil || border.Title != "" ||
		details.ItemCount < 0 ||
		details.ItemCount > expletives.MaxCollectionItems ||
		details.EnabledCount < 0 ||
		details.EnabledCount > details.ItemCount ||
		details.RetainedBytes < 0 ||
		details.RetainedBytes > expletives.MaxCollectionAggregateBytes ||
		((details.ItemCount > 0 || details.StatusMessageBytes > 0) &&
			details.RetainedBytes == 0) ||
		details.SelectedCount < 0 ||
		details.SelectedCount > details.EnabledCount ||
		details.DisabledReasonBytes < 0 ||
		details.DisabledReasonBytes > expletives.MaxCommandDescriptionBytes ||
		(details.Enabled && details.DisabledReasonBytes != 0) ||
		(!details.Enabled && details.DisabledReasonBytes == 0) ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) ||
		(details.ActivateCommand != "" &&
			!validIdentifier(details.ActivateCommand, limits.IdentifierBytes)) ||
		!validContentViewportDetails(
			&details.Viewport,
			border,
			controlWidth,
			controlHeight,
		) {
		return false
	}
	switch details.Status {
	case "ready":
		if details.StatusMessageBytes != 0 || details.StatusMessageDigest != "" ||
			details.Viewport.State.ContentSize.Height != max(1, details.ItemCount) {
			return false
		}
	case "loading", "error":
		if details.StatusMessageBytes < 1 ||
			details.StatusMessageBytes > expletives.MaxDisplayTextBytes ||
			!validLowerSHA256(details.StatusMessageDigest) ||
			details.Viewport.State.ContentSize.Height != 1 {
			return false
		}
	default:
		return false
	}
	if details.SelectionMode != "single" &&
		details.SelectionMode != "multiple" {
		return false
	}
	if details.SelectionMode == "single" && details.SelectedCount > 1 {
		return false
	}
	if details.RequireSelection && details.EnabledCount > 0 &&
		details.SelectedCount == 0 {
		return false
	}
	if details.EnabledCount == 0 {
		if details.Current != "" || details.CurrentIndex != -1 {
			return false
		}
	} else if !validIdentifier(details.Current, limits.IdentifierBytes) ||
		details.CurrentIndex < 0 || details.CurrentIndex >= details.ItemCount {
		return false
	}
	if details.SelectedCount == 0 {
		if details.FirstSelected != "" || details.LastSelected != "" {
			return false
		}
	} else if !validIdentifier(details.FirstSelected, limits.IdentifierBytes) ||
		!validIdentifier(details.LastSelected, limits.IdentifierBytes) ||
		(details.SelectedCount == 1 &&
			details.FirstSelected != details.LastSelected) ||
		(details.SelectedCount > 1 &&
			details.FirstSelected == details.LastSelected) {
		return false
	}
	if !validLowerSHA256(details.SelectionDigest) {
		return false
	}
	if details.SelectedCount == 0 && details.SelectionDigest !=
		"e3b0c44298fc1c149afbf4c8996fb924"+
			"27ae41e4649b934ca495991b7852b855" {
		return false
	}
	if details.Status == "ready" && details.CurrentIndex >= 0 &&
		details.Viewport.ViewportBounds.Height > 0 &&
		(details.CurrentIndex < details.Viewport.State.Offset.Y ||
			details.CurrentIndex >= details.Viewport.State.Offset.Y+
				details.Viewport.ViewportBounds.Height) {
		return false
	}
	return details.Viewport.State.ContentSize.Width >= 1 &&
		details.Viewport.State.ContentSize.Width <=
			2*expletives.MaxDisplayTextCells+8
}

func validLowerSHA256(value string) bool {
	if len(value) != sha256HexBytes || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validMarkdownDetails(
	details *MarkdownDetails,
	border *BorderDetails,
	controlWidth int,
	controlHeight int,
) bool {
	if details == nil || border == nil || border.Title != "" ||
		details.SourceBytes < 0 ||
		details.SourceBytes > expletives.MaxContentBytes ||
		details.SourceCells < 0 ||
		details.SourceCells > details.SourceBytes ||
		details.BlockCount < 1 ||
		details.BlockCount > expletives.MaxMarkdownBlocks ||
		details.RenderedRows < 1 ||
		details.RenderedRows >
			expletives.MaxContentBytes+expletives.MaxMarkdownBlocks ||
		details.MaximumLineWidth < 0 ||
		details.MaximumLineWidth > expletives.MaxContentBytes ||
		len(details.Blocks) > expletives.MaxMarkdownSummaries ||
		len(details.Blocks) > details.BlockCount ||
		(details.SummariesTruncated &&
			(len(details.Blocks) != expletives.MaxMarkdownSummaries ||
				details.BlockCount <= len(details.Blocks))) ||
		(!details.SummariesTruncated &&
			len(details.Blocks) != details.BlockCount) ||
		details.Viewport.State.ContentSize != (Size{
			Width: details.MaximumLineWidth, Height: details.RenderedRows,
		}) ||
		details.Viewport.State.Offset.X < 0 ||
		details.Viewport.State.Offset.Y < 0 ||
		!validScrollVisibility(details.Viewport.HorizontalPolicy) ||
		!validScrollVisibility(details.Viewport.VerticalPolicy) {
		return false
	}
	previousSourceLine := 0
	previousRenderedStart := -1
	for _, block := range details.Blocks {
		if !validMarkdownBlockKind(block.Kind) ||
			block.SourceLine < 1 ||
			block.SourceLines < 1 ||
			block.RenderedStart < 0 ||
			block.RenderedRows < 1 ||
			block.RenderedStart+block.RenderedRows > details.RenderedRows ||
			block.SourceLine <= previousSourceLine ||
			block.RenderedStart <= previousRenderedStart ||
			(block.Kind == "heading" &&
				(block.Level < 1 || block.Level > 6)) ||
			(block.Kind != "heading" && block.Level != 0) {
			return false
		}
		previousSourceLine = block.SourceLine
		previousRenderedStart = block.RenderedStart
	}
	return validContentViewportDetails(
		&details.Viewport,
		border,
		controlWidth,
		controlHeight,
	)
}

func validLogViewDetails(
	details *LogViewDetails,
	border *BorderDetails,
	controlWidth int,
	controlHeight int,
	limits Limits,
) bool {
	if details == nil || border == nil || border.Title != "" ||
		!validContentCapacity(details.Capacity) ||
		details.RetainedRecords < 0 ||
		details.RetainedRecords > details.Capacity.Records ||
		details.RetainedBytes < 0 ||
		details.RetainedBytes > details.Capacity.Bytes ||
		(details.RetainedRecords == 0 &&
			(details.RetainedBytes != 0 || details.FirstKey != "" ||
				details.LastKey != "")) ||
		(details.RetainedRecords > 0 &&
			(!validIdentifier(details.FirstKey, limits.IdentifierBytes) ||
				!validIdentifier(details.LastKey, limits.IdentifierBytes))) ||
		(details.RetainedRecords > 1 && details.FirstKey == details.LastKey) ||
		(details.DroppedBytes > 0 && details.DroppedRecords == 0) ||
		details.Viewport.State.ContentSize.Width < 0 ||
		details.Viewport.State.ContentSize.Width >
			expletives.MaxContentBytes+2*expletives.MaxDisplayTextBytes ||
		details.Viewport.State.ContentSize.Height < details.RetainedRecords ||
		details.Viewport.State.ContentSize.Height >
			expletives.MaxContentBytes+expletives.MaxContentRecords+1 ||
		(details.RetainedRecords == 0 && details.DroppedRecords == 0 &&
			details.Viewport.State.ContentSize.Height != 0) {
		return false
	}
	if details.DroppedRecords > 0 &&
		details.Viewport.State.ContentSize.Height < details.RetainedRecords+1 {
		return false
	}
	return validContentViewportDetails(
		&details.Viewport,
		border,
		controlWidth,
		controlHeight,
	)
}

func validStreamViewDetails(
	details *StreamViewDetails,
	border *BorderDetails,
	controlWidth int,
	controlHeight int,
) bool {
	if details == nil || border == nil || border.Title != "" ||
		!validContentCapacity(details.Capacity) ||
		details.RetainedLines < 0 ||
		details.RetainedLines > details.Capacity.Records ||
		details.RetainedBytes < 0 ||
		details.PendingBytes < 0 ||
		details.PendingCells < 0 ||
		details.PendingStorageBytes < 0 ||
		details.PendingCells > details.PendingStorageBytes ||
		details.RetainedBytes+details.PendingStorageBytes >
			details.Capacity.Bytes ||
		details.PendingBytes > details.Capacity.Bytes ||
		(details.PendingTruncated && details.DroppedLines == 0) ||
		(details.DroppedBytes > 0 && details.DroppedLines == 0) {
		return false
	}
	pendingVisible := details.PendingBytes > 0 || details.PendingTruncated
	dropVisible := details.DroppedLines > 0 || details.DroppedBytes > 0
	if pendingVisible !=
		(details.PendingCells > 0 && details.PendingStorageBytes > 0) {
		return false
	}
	expectedRows := details.RetainedLines + boolInt(pendingVisible) +
		boolInt(dropVisible)
	if details.Viewport.State.ContentSize.Height != expectedRows ||
		details.Viewport.State.ContentSize.Width < 0 ||
		details.Viewport.State.ContentSize.Width >
			max(expletives.MaxContentBytes, expletives.MaxDisplayTextBytes) {
		return false
	}
	return validContentViewportDetails(
		&details.Viewport,
		border,
		controlWidth,
		controlHeight,
	)
}

func validContentCapacity(capacity ContentCapacity) bool {
	return capacity.Records >= 1 &&
		capacity.Records <= expletives.MaxContentRecords &&
		capacity.Bytes >= 1 && capacity.Bytes <= expletives.MaxContentBytes
}

func validContentViewportDetails(
	details *ContentViewportDetails,
	border *BorderDetails,
	controlWidth int,
	controlHeight int,
) bool {
	if details == nil || border == nil ||
		details.State.ContentSize.Width < 0 ||
		details.State.ContentSize.Height < 0 ||
		details.State.Offset.X < 0 || details.State.Offset.Y < 0 ||
		!validScrollVisibility(details.HorizontalPolicy) ||
		!validScrollVisibility(details.VerticalPolicy) {
		return false
	}
	inset := 0
	if border.Form != "none" {
		inset = 1
	}
	baseWidth := max(0, controlWidth-2*inset)
	baseHeight := max(0, controlHeight-2*inset)
	horizontal := details.HorizontalPolicy == "always" &&
		baseWidth > 0 && baseHeight > 0
	vertical := details.VerticalPolicy == "always" &&
		baseWidth > 0 && baseHeight > 0
	for range 3 {
		viewportWidth := max(0, baseWidth-boolInt(vertical))
		viewportHeight := max(0, baseHeight-boolInt(horizontal))
		nextHorizontal := horizontal
		nextVertical := vertical
		if details.HorizontalPolicy == "auto" {
			nextHorizontal = baseWidth > 0 && baseHeight > 0 &&
				details.State.ContentSize.Width > viewportWidth
		}
		if details.HorizontalPolicy == "never" {
			nextHorizontal = false
		}
		if details.VerticalPolicy == "auto" {
			nextVertical = baseWidth > 0 && baseHeight > 0 &&
				details.State.ContentSize.Height > viewportHeight
		}
		if details.VerticalPolicy == "never" {
			nextVertical = false
		}
		if nextHorizontal == horizontal && nextVertical == vertical {
			break
		}
		horizontal, vertical = nextHorizontal, nextVertical
	}
	expectedBounds := Rect{
		X: inset,
		Y: inset,
		Width: max(
			0,
			baseWidth-boolInt(vertical),
		),
		Height: max(
			0,
			baseHeight-boolInt(horizontal),
		),
	}
	maximum := Point{
		X: max(0, details.State.ContentSize.Width-expectedBounds.Width),
		Y: max(0, details.State.ContentSize.Height-expectedBounds.Height),
	}
	return details.ViewportBounds == expectedBounds &&
		details.MaximumOffset == maximum &&
		details.State.Offset.X <= maximum.X &&
		details.State.Offset.Y <= maximum.Y &&
		details.HorizontalVisible == horizontal &&
		details.VerticalVisible == vertical
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func validMarkdownBlockKind(kind string) bool {
	switch kind {
	case "blank", "code", "heading", "list", "paragraph", "quote", "rule":
		return true
	default:
		return false
	}
}

func validScrollableDetails(
	kind ControlKind,
	container *ContainerDetails,
	border *BorderDetails,
	details *ScrollableDetails,
	controlWidth int,
	controlHeight int,
	limits Limits,
) bool {
	if details == nil ||
		!validIdentifier(string(details.Content), limits.IdentifierBytes) ||
		(details.ContentKey != "" &&
			!validIdentifier(details.ContentKey, limits.IdentifierBytes)) ||
		details.State.ContentSize.Width < 0 ||
		details.State.ContentSize.Height < 0 ||
		details.State.Offset.X < 0 ||
		details.State.Offset.Y < 0 ||
		details.ArrowStep.Width < 1 ||
		details.ArrowStep.Height < 1 ||
		details.PageStep.Width < 1 ||
		details.PageStep.Height < 1 ||
		details.ViewportBounds.X < 0 ||
		details.ViewportBounds.Y < 0 ||
		details.ViewportBounds.Width < 0 ||
		details.ViewportBounds.Height < 0 ||
		details.ViewportBounds.X+details.ViewportBounds.Width > controlWidth ||
		details.ViewportBounds.Y+details.ViewportBounds.Height > controlHeight ||
		len(details.DisabledReason) > expletives.MaxCommandDescriptionBytes ||
		(!details.Enabled && details.DisabledReason == "") ||
		(details.Enabled && details.DisabledReason != "") ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) ||
		!validScrollVisibility(details.HorizontalPolicy) ||
		!validScrollVisibility(details.VerticalPolicy) {
		return false
	}
	if kind == "viewport" {
		if container.ClientInset != 0 ||
			border.Form != "none" ||
			details.HorizontalPolicy != "never" ||
			details.VerticalPolicy != "never" ||
			details.HorizontalVisible ||
			details.VerticalVisible {
			return false
		}
	} else {
		wantInset := 0
		if border.Form != "none" {
			wantInset = 1
		}
		if container == nil || container.ClientInset != wantInset {
			return false
		}
	}
	maximum := Point{
		X: max(
			0,
			details.State.ContentSize.Width-details.ViewportBounds.Width,
		),
		Y: max(
			0,
			details.State.ContentSize.Height-details.ViewportBounds.Height,
		),
	}
	if details.MaximumOffset != maximum ||
		details.State.Offset.X > maximum.X ||
		details.State.Offset.Y > maximum.Y ||
		(details.HorizontalVisible != (details.HorizontalBar != nil)) ||
		(details.VerticalVisible != (details.VerticalBar != nil)) {
		return false
	}
	if details.HorizontalBar != nil &&
		(details.HorizontalBar.DisabledReason != "" ||
			details.HorizontalBar.ChangeCommand != "" ||
			!validIntegratedScrollBarDetails(
				details.HorizontalBar,
				details,
				details.ViewportBounds.Width,
				1,
				limits,
			) ||
			details.HorizontalBar.Orientation !=
				Orientation(expletives.Horizontal) ||
			details.HorizontalBar.ContentSize != details.State.ContentSize.Width ||
			details.HorizontalBar.ViewportSize != details.ViewportBounds.Width ||
			details.HorizontalBar.Offset != details.State.Offset.X) {
		return false
	}
	if details.VerticalBar != nil &&
		(details.VerticalBar.DisabledReason != "" ||
			details.VerticalBar.ChangeCommand != "" ||
			!validIntegratedScrollBarDetails(
				details.VerticalBar,
				details,
				1,
				details.ViewportBounds.Height,
				limits,
			) ||
			details.VerticalBar.Orientation !=
				Orientation(expletives.Vertical) ||
			details.VerticalBar.ContentSize != details.State.ContentSize.Height ||
			details.VerticalBar.ViewportSize != details.ViewportBounds.Height ||
			details.VerticalBar.Offset != details.State.Offset.Y) {
		return false
	}
	return true
}

func validIntegratedScrollBarDetails(
	bar *ScrollBarDetails,
	parent *ScrollableDetails,
	width int,
	height int,
	limits Limits,
) bool {
	if bar == nil || parent == nil {
		return false
	}
	complete := *bar
	complete.DisabledReason = parent.DisabledReason
	complete.ChangeCommand = parent.ChangeCommand
	return validScrollBarDetails(&complete, width, height, limits)
}

func validScrollVisibility(value string) bool {
	switch value {
	case "auto", "always", "never":
		return true
	default:
		return false
	}
}

func validTabbedPanelDetails(
	details *TabbedPanelDetails,
	controlWidth int,
	limits Limits,
) bool {
	if details == nil ||
		len(details.Tabs) > expletives.MaxSelectionOptions ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) {
		return false
	}
	if len(details.Tabs) == 0 {
		return details.Selected == "" &&
			details.Current == "" &&
			!details.LeadingOmitted &&
			!details.TrailingOmitted
	}
	keys := make(map[string]bool, len(details.Tabs))
	values := make(map[string]bool, len(details.Tabs))
	pages := make(map[ControlID]bool, len(details.Tabs))
	mnemonics := make(map[Key]bool, len(details.Tabs))
	selectedCount := 0
	currentCount := 0
	enabledCount := 0
	renderedCount := 0
	lastRight := 0
	for index := range details.Tabs {
		tab := details.Tabs[index]
		if !validIdentifier(tab.Key, limits.IdentifierBytes) ||
			keys[tab.Key] ||
			!validIdentifier(tab.Value, limits.IdentifierBytes) ||
			values[tab.Value] ||
			!validIdentifier(string(tab.Page), limits.IdentifierBytes) ||
			pages[tab.Page] ||
			(tab.PageKey != "" &&
				!validIdentifier(tab.PageKey, limits.IdentifierBytes)) ||
			!canonicalDisplayText(tab.Label, false) ||
			tab.Label == "" ||
			len(tab.DisabledReason) > expletives.MaxCommandDescriptionBytes ||
			(tab.Enabled && tab.DisabledReason != "") ||
			(!tab.Enabled && tab.DisabledReason == "") ||
			tab.Bounds.X < 0 ||
			tab.Bounds.Y != 0 ||
			tab.Bounds.Width < 0 ||
			tab.Bounds.Height < 0 ||
			tab.Bounds.X+tab.Bounds.Width > controlWidth {
			return false
		}
		if tab.Mnemonic != "" {
			if !validTabMnemonic(tab.Mnemonic, tab.Label) ||
				mnemonics[tab.Mnemonic] {
				return false
			}
			mnemonics[tab.Mnemonic] = true
		}
		if tab.Omitted {
			if tab.Bounds != (Rect{}) || tab.Clipped {
				return false
			}
		} else {
			labelCells := len(display.Normalize(tab.Label))
			if tab.Bounds.Height != 1 ||
				tab.Bounds.Width < 1 ||
				tab.Bounds.X < lastRight ||
				(!tab.Clipped &&
					tab.Bounds.Width != labelCells+2) ||
				(tab.Clipped &&
					tab.Bounds.Width >= labelCells+2) {
				return false
			}
			lastRight = tab.Bounds.X + tab.Bounds.Width
			renderedCount++
		}
		if tab.Selected {
			selectedCount++
			if tab.Value != details.Selected {
				return false
			}
		}
		if tab.Current {
			currentCount++
			if tab.Value != details.Current {
				return false
			}
		}
		if tab.Enabled {
			enabledCount++
		}
		keys[tab.Key] = true
		values[tab.Value] = true
		pages[tab.Page] = true
	}
	selectedIndex := -1
	currentIndex := -1
	for index := range details.Tabs {
		if details.Tabs[index].Value == details.Selected {
			selectedIndex = index
		}
		if details.Tabs[index].Value == details.Current {
			currentIndex = index
		}
	}
	if selectedCount != 1 || currentCount != 1 ||
		selectedIndex < 0 || currentIndex < 0 ||
		(enabledCount > 0 && !details.Tabs[currentIndex].Enabled) ||
		(enabledCount == 0 && currentIndex != 0) {
		return false
	}
	if controlWidth <= 0 {
		return renderedCount == 0
	}
	return renderedCount > 0
}

func validTabMnemonic(mnemonic Key, label string) bool {
	value := string(mnemonic)
	if len(value) != 1 ||
		!((value[0] >= 'a' && value[0] <= 'z') ||
			(value[0] >= '0' && value[0] <= '9')) {
		return false
	}
	for _, cell := range display.Normalize(label) {
		if strings.ToLower(cell) == value {
			return true
		}
	}
	return false
}

func validScrollBarDetails(
	details *ScrollBarDetails,
	controlWidth int,
	controlHeight int,
	limits Limits,
) bool {
	if details == nil ||
		(details.Orientation != Orientation(expletives.Horizontal) &&
			details.Orientation != Orientation(expletives.Vertical)) ||
		details.ContentSize < 0 ||
		details.ContentSize > expletives.MaxFrameCells ||
		details.ViewportSize < 0 ||
		details.ViewportSize > expletives.MaxFrameCells ||
		details.Offset < 0 ||
		details.ArrowStep < 1 ||
		details.ArrowStep > expletives.MaxFrameCells ||
		details.PageStep < 1 ||
		details.PageStep > expletives.MaxFrameCells ||
		len(details.DisabledReason) > expletives.MaxCommandDescriptionBytes ||
		(!details.Enabled && details.DisabledReason == "") ||
		(details.Enabled && details.DisabledReason != "") ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) {
		return false
	}
	maximum := details.ContentSize - details.ViewportSize
	if maximum < 0 {
		maximum = 0
	}
	if details.MaximumOffset != maximum || details.Offset > maximum {
		return false
	}
	axis := controlWidth
	if details.Orientation == Orientation(expletives.Vertical) {
		axis = controlHeight
	}
	core := expletives.ScrollBarState{
		ContentSize:  details.ContentSize,
		ViewportSize: details.ViewportSize,
		Offset:       details.Offset,
	}
	trackStart := 0
	trackSize := axis
	if axis >= 3 {
		trackStart = 1
		trackSize = axis - 2
	}
	thumbSize := 0
	if trackSize > 0 {
		switch {
		case maximum == 0:
			thumbSize = trackSize
		case core.ViewportSize == 0:
			thumbSize = 1
		default:
			thumbSize = max(
				1,
				trackSize*core.ViewportSize/core.ContentSize,
			)
			thumbSize = min(thumbSize, trackSize)
		}
	}
	thumbStart := trackStart
	travel := trackSize - thumbSize
	if travel > 0 && maximum > 0 {
		thumbStart += core.Offset * travel / maximum
	}
	return details.TrackStart == trackStart &&
		details.TrackSize == trackSize &&
		details.ThumbStart == thumbStart &&
		details.ThumbSize == thumbSize
}

func validProgressDetails(
	kind ControlKind,
	details *ProgressDetails,
	controlWidth int,
) bool {
	if details == nil || details.FrameIndex < 0 {
		return false
	}
	switch details.Status {
	case "idle", "running", "completed", "failed", "cancelled":
	default:
		return false
	}
	switch kind {
	case "progress_bar":
		if details.Orientation != Orientation(expletives.Horizontal) ||
			details.Value != 0 || details.Minimum != 0 ||
			details.Maximum != 0 ||
			(details.TextMode != "none" &&
				details.TextMode != "percentage") {
			return false
		}
		if details.Indeterminate {
			if details.Current != 0 || details.Total != 0 ||
				details.TextMode != "none" {
				return false
			}
		} else if details.Current > details.Total ||
			(details.Status == "completed" &&
				details.Current != details.Total) ||
			details.Tick != 0 || details.FrameIndex != 0 {
			return false
		}
		if details.ReducedMotion || details.Status != "running" {
			return details.Tick == 0 && details.FrameIndex == 0
		}
		if controlWidth <= 0 {
			return details.FrameIndex == 0
		}
		return details.FrameIndex ==
			int(details.Tick%uint64(controlWidth))
	case "meter":
		return !details.Indeterminate &&
			details.Current == 0 && details.Total == 0 &&
			details.Tick == 0 && !details.ReducedMotion &&
			details.TextMode == "" && details.FrameIndex == 0 &&
			(details.Orientation == Orientation(expletives.Horizontal) ||
				details.Orientation == Orientation(expletives.Vertical)) &&
			finiteWireNumber(details.Value) &&
			finiteWireNumber(details.Minimum) &&
			finiteWireNumber(details.Maximum) &&
			details.Minimum < details.Maximum &&
			details.Value >= details.Minimum &&
			details.Value <= details.Maximum
	case "spinner", "activity_dots":
		if !details.Indeterminate ||
			details.Current != 0 || details.Total != 0 ||
			details.Value != 0 || details.Minimum != 0 ||
			details.Maximum != 0 ||
			details.Orientation != Orientation(expletives.Horizontal) ||
			details.TextMode != "none" {
			return false
		}
		if details.ReducedMotion || details.Status != "running" {
			return details.Tick == 0 && details.FrameIndex == 0
		}
		frameCount := uint64(4)
		if kind == "activity_dots" {
			frameCount = 5
		}
		return details.FrameIndex == int(details.Tick%frameCount)
	default:
		return false
	}
}

func validNumberFieldDetails(
	details *NumberFieldDetails,
	spin bool,
	limits Limits,
) bool {
	if details == nil ||
		details.Length < 0 ||
		details.Length > expletives.MaxTextInputCells ||
		details.Caret < 0 ||
		details.Caret > details.Length ||
		details.SelectionStart < 0 ||
		details.SelectionStart > details.SelectionEnd ||
		details.SelectionEnd > details.Length ||
		!validEditorSelection(
			details.Caret,
			details.SelectionStart,
			details.SelectionEnd,
		) ||
		details.ViewOffset < 0 ||
		details.ViewOffset > details.Length ||
		details.DecimalPlaces < 0 ||
		details.DecimalPlaces > expletives.MaxNumberDecimalPlaces ||
		!finiteWireNumber(details.Value) ||
		(!details.Enabled && details.Editing) ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) ||
		len(details.InvalidReason) > maxDisplayTextBytes ||
		!utf8.ValidString(details.InvalidReason) ||
		strings.ContainsRune(details.InvalidReason, 0) {
		return false
	}
	cells, ok := canonicalInputCells(details.Text)
	if !ok || len(cells) != details.Length {
		return false
	}
	if details.Minimum != nil {
		if !finiteWireNumber(*details.Minimum) ||
			canonicalWireNumber(
				*details.Minimum,
				details.DecimalPlaces,
			) != *details.Minimum {
			return false
		}
	}
	if details.Maximum != nil {
		if !finiteWireNumber(*details.Maximum) ||
			canonicalWireNumber(
				*details.Maximum,
				details.DecimalPlaces,
			) != *details.Maximum {
			return false
		}
	}
	if details.Minimum != nil && details.Maximum != nil &&
		*details.Minimum > *details.Maximum {
		return false
	}
	if canonicalWireNumber(details.Value, details.DecimalPlaces) !=
		details.Value ||
		(details.Minimum != nil && details.Value < *details.Minimum) ||
		(details.Maximum != nil && details.Value > *details.Maximum) {
		return false
	}
	if spin {
		if !finiteWireNumber(details.Step) || details.Step <= 0 ||
			canonicalWireNumber(details.Step, details.DecimalPlaces) !=
				details.Step {
			return false
		}
	} else if details.Step != 0 {
		return false
	}
	if !details.Editing {
		return details.Valid &&
			details.InvalidReason == "" &&
			details.Text == formatWireNumber(
				details.Value,
				details.DecimalPlaces,
			)
	}
	_, reason, valid := parseWireNumberText(
		details.Text,
		details.DecimalPlaces,
		details.Minimum,
		details.Maximum,
	)
	return details.Valid == valid &&
		details.InvalidReason == reason
}

func finiteWireNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func canonicalWireNumber(value float64, decimalPlaces int) float64 {
	factor := math.Pow10(decimalPlaces)
	canonical := math.Round(value*factor) / factor
	if canonical == 0 {
		return 0
	}
	return canonical
}

func formatWireNumber(value float64, decimalPlaces int) string {
	return strconv.FormatFloat(value, 'f', decimalPlaces, 64)
}

func parseWireNumberText(
	text string,
	decimalPlaces int,
	minimum *float64,
	maximum *float64,
) (float64, string, bool) {
	if text == "" || text == "-" || text == "." || text == "-." ||
		strings.HasPrefix(text, "+") || strings.Count(text, ".") > 1 {
		return 0, "Enter a complete decimal number", false
	}
	if point := strings.IndexByte(text, '.'); point >= 0 &&
		len(text)-point-1 > decimalPlaces {
		return 0, "Too many decimal places", false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || !finiteWireNumber(value) {
		return 0, "Enter a complete decimal number", false
	}
	if canonicalWireNumber(value, decimalPlaces) != value {
		return 0, "Too many decimal places", false
	}
	if minimum != nil && value < *minimum {
		return 0, "Value is below the minimum", false
	}
	if maximum != nil && value > *maximum {
		return 0, "Value is above the maximum", false
	}
	return value, "", true
}

func validTextFieldDetails(
	details *TextFieldDetails,
	limits Limits,
) bool {
	if details == nil ||
		details.Length < 0 ||
		details.Length > expletives.MaxTextInputCells ||
		details.Caret < 0 ||
		details.Caret > details.Length ||
		details.SelectionStart < 0 ||
		details.SelectionStart > details.SelectionEnd ||
		details.SelectionEnd > details.Length ||
		!validEditorSelection(
			details.Caret,
			details.SelectionStart,
			details.SelectionEnd,
		) ||
		details.ViewOffset < 0 ||
		details.ViewOffset > details.Length ||
		details.Password != details.Redacted ||
		(!details.Enabled && details.Editing) ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) {
		return false
	}
	if details.Redacted {
		if details.Text != "" {
			return false
		}
	} else {
		cells, ok := canonicalInputCells(details.Text)
		if !ok || len(cells) != details.Length {
			return false
		}
	}
	if details.Validator == nil {
		return details.Valid
	}
	validatorCells, ok := canonicalInputCells(details.Validator.Characters)
	if !ok || len(validatorCells) == 0 ||
		len(details.Validator.Characters) > expletives.MaxTextValidatorBytes ||
		len(validatorCells) > expletives.MaxTextValidatorCells {
		return false
	}
	switch details.Validator.Enforcement {
	case "soft", "hard":
	default:
		return false
	}
	switch details.Validator.Mode {
	case "whitelist", "blacklist":
	default:
		return false
	}
	set := make(map[string]bool, len(validatorCells))
	for _, cell := range validatorCells {
		if set[cell] {
			return false
		}
		set[cell] = true
	}
	if details.Redacted {
		return details.Validator.Enforcement != "hard" || details.Valid
	}
	valueCells, _ := canonicalInputCells(details.Text)
	valid := true
	for _, cell := range valueCells {
		matched := set[cell]
		allowed := matched
		if details.Validator.Mode == "blacklist" {
			allowed = !matched
		}
		valid = valid && allowed
	}
	return details.Valid == valid &&
		(details.Validator.Enforcement != "hard" || valid)
}

func validTextAreaDetails(
	details *TextAreaDetails,
	controlWidth int,
	limits Limits,
) bool {
	if details == nil ||
		details.Length < 0 ||
		details.Length > expletives.MaxTextInputCells ||
		details.LineCount < 1 ||
		details.LineCount > details.Length+1 ||
		details.Caret < 0 ||
		details.Caret > details.Length ||
		details.SelectionStart < 0 ||
		details.SelectionStart > details.SelectionEnd ||
		details.SelectionEnd > details.Length ||
		!validEditorSelection(
			details.Caret,
			details.SelectionStart,
			details.SelectionEnd,
		) ||
		details.VisualCaretRow < 0 ||
		details.VisualCaretRow > details.Length ||
		details.VisualCaretColumn < 0 ||
		details.VisualCaretColumn > details.Length ||
		details.RowOffset < 0 ||
		details.RowOffset > details.VisualCaretRow ||
		details.ColumnOffset < 0 ||
		details.ColumnOffset > details.VisualCaretColumn ||
		details.Password != details.Redacted ||
		(!details.Enabled && details.Editing) ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		(details.ChangeCommand != "" &&
			!validIdentifier(details.ChangeCommand, limits.IdentifierBytes)) {
		return false
	}
	switch details.Wrap {
	case "none":
	case "words", "cells":
		if details.ColumnOffset != 0 ||
			(controlWidth > 0 &&
				details.VisualCaretColumn > controlWidth) {
			return false
		}
	default:
		return false
	}
	var valueCells []string
	if details.Redacted {
		if details.Text != "" {
			return false
		}
	} else {
		var ok bool
		valueCells, ok = canonicalTextAreaCells(details.Text)
		if !ok || len(valueCells) != details.Length ||
			1+strings.Count(details.Text, "\n") != details.LineCount {
			return false
		}
	}
	if details.Validator == nil {
		return details.Valid
	}
	validatorCells, ok := canonicalInputCells(details.Validator.Characters)
	if !ok || len(validatorCells) == 0 ||
		len(details.Validator.Characters) > expletives.MaxTextValidatorBytes ||
		len(validatorCells) > expletives.MaxTextValidatorCells {
		return false
	}
	switch details.Validator.Enforcement {
	case "soft", "hard":
	default:
		return false
	}
	switch details.Validator.Mode {
	case "whitelist", "blacklist":
	default:
		return false
	}
	set := make(map[string]bool, len(validatorCells))
	for _, cell := range validatorCells {
		if set[cell] {
			return false
		}
		set[cell] = true
	}
	if details.Redacted {
		return details.Validator.Enforcement != "hard" || details.Valid
	}
	valid := true
	for _, cell := range valueCells {
		if cell == "\n" {
			continue
		}
		matched := set[cell]
		allowed := matched
		if details.Validator.Mode == "blacklist" {
			allowed = !matched
		}
		valid = valid && allowed
	}
	return details.Valid == valid &&
		(details.Validator.Enforcement != "hard" || valid)
}

func validEditorSelection(caret, start, end int) bool {
	if start == end {
		return caret == start
	}
	return caret == start || caret == end
}

func canonicalTextAreaCells(text string) ([]string, bool) {
	if len(text) > expletives.MaxTextInputBytes ||
		!utf8.ValidString(text) ||
		strings.ContainsRune(text, '\r') {
		return nil, false
	}
	lines := strings.Split(text, "\n")
	cells := make([]string, 0, len(text))
	for index, line := range lines {
		lineCells, ok := canonicalInputCells(line)
		if !ok {
			return nil, false
		}
		cells = append(cells, lineCells...)
		if index+1 < len(lines) {
			cells = append(cells, "\n")
		}
	}
	if len(cells) > expletives.MaxTextInputCells ||
		strings.Join(cells, "") != text {
		return nil, false
	}
	return cells, true
}

func canonicalInputCells(text string) ([]string, bool) {
	if len(text) > expletives.MaxTextInputBytes ||
		!utf8.ValidString(text) {
		return nil, false
	}
	for _, current := range text {
		if current == '\r' || current == '\n' || unicode.IsControl(current) {
			return nil, false
		}
	}
	cells := display.Normalize(text)
	if len(cells) > expletives.MaxTextInputCells ||
		strings.Join(cells, "") != text {
		return nil, false
	}
	return cells, true
}

func validFocusGuideBarDetails(
	details *FocusGuideBarDetails,
	limits Limits,
) bool {
	if details == nil ||
		!canonicalDisplayText(details.Text, false) ||
		len(details.Text) > maxDisplayTextBytes {
		return false
	}
	switch details.Customization {
	case "", "append", "override":
	default:
		return false
	}
	if details.Target == "" {
		return details.TargetKind == "" && details.Customization == ""
	}
	return validIdentifier(string(details.Target), limits.IdentifierBytes) &&
		validFocusTargetKind(details.TargetKind)
}

func validFocusTargetKind(kind ControlKind) bool {
	switch kind {
	case "button", "checkbox", "radio_button", "cycle_field",
		"select_field", "text_field", "number_field", "spin_box",
		"text_area", "menu_bar", "scroll_bar", "tabbed_panel", "notebook",
		"viewport", "scrollable_panel", "markdown_view", "log_view",
		"stream_view", "list_box":
		return true
	default:
		return false
	}
}

func validSelectionReason(enabled bool, reason string) bool {
	return len(reason) <= maxDisplayTextBytes &&
		utf8.ValidString(reason) &&
		!strings.ContainsRune(reason, 0) &&
		((enabled && reason == "") || (!enabled && reason != ""))
}

func validOptionalMnemonic(mnemonic Key) bool {
	return mnemonic == "" ||
		(len(mnemonic) == 1 && validLogicalKey(string(mnemonic)))
}

func validOptionalCommand(command string, limits Limits) bool {
	return command == "" ||
		validIdentifier(command, limits.IdentifierBytes)
}

func validCheckboxDetails(
	details *CheckboxDetails,
	limits Limits,
) bool {
	if details == nil ||
		!canonicalDisplayText(details.Label, false) ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		!validOptionalMnemonic(details.Mnemonic) ||
		!validOptionalCommand(details.ChangeCommand, limits) {
		return false
	}
	switch details.State {
	case "unchecked", "checked":
		return true
	case "indeterminate":
		return details.ThreeState
	default:
		return false
	}
}

func validRadioButtonDetails(
	details *RadioButtonDetails,
	limits Limits,
) bool {
	return details != nil &&
		validIdentifier(details.Value, limits.IdentifierBytes) &&
		canonicalDisplayText(details.Label, false) &&
		validSelectionReason(details.Enabled, details.DisabledReason) &&
		validOptionalMnemonic(details.Mnemonic)
}

func validRadioGroupDetails(
	details *RadioGroupDetails,
	limits Limits,
) bool {
	if details == nil ||
		len(details.Options) > expletives.MaxSelectionOptions ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		!validOptionalCommand(details.ChangeCommand, limits) {
		return false
	}
	values := make(map[string]bool, len(details.Options))
	controls := make(map[ControlID]bool, len(details.Options))
	selected := ""
	enabledCount := 0
	for _, option := range details.Options {
		if !validIdentifier(string(option.Control), limits.IdentifierBytes) ||
			controls[option.Control] ||
			!validIdentifier(option.Value, limits.IdentifierBytes) ||
			values[option.Value] ||
			!canonicalDisplayText(option.Label, false) ||
			!validSelectionReason(option.Enabled, option.DisabledReason) {
			return false
		}
		if option.Enabled {
			enabledCount++
		}
		controls[option.Control] = true
		values[option.Value] = true
		if option.Selected {
			if selected != "" {
				return false
			}
			selected = option.Value
		}
	}
	if details.Value == "" {
		return selected == "" && (details.AllowEmpty || enabledCount == 0)
	}
	return validIdentifier(details.Value, limits.IdentifierBytes) &&
		selected == details.Value
}

func validChoiceFieldDetails(
	details *ChoiceFieldDetails,
	limits Limits,
) bool {
	if details == nil ||
		!canonicalDisplayText(details.Label, false) ||
		len(details.Options) > expletives.MaxSelectionOptions ||
		!validSelectionReason(details.Enabled, details.DisabledReason) ||
		!validOptionalMnemonic(details.Mnemonic) ||
		!validOptionalCommand(details.ChangeCommand, limits) {
		return false
	}
	values := make(map[string]bool, len(details.Options))
	selectedIndex := -1
	for index, option := range details.Options {
		if !validIdentifier(option.Value, limits.IdentifierBytes) ||
			values[option.Value] ||
			!canonicalDisplayText(option.Label, false) ||
			!validSelectionReason(option.Enabled, option.DisabledReason) ||
			(option.Selected && !option.Enabled) {
			return false
		}
		values[option.Value] = true
		if option.Selected {
			if selectedIndex >= 0 {
				return false
			}
			selectedIndex = index
		}
	}
	if details.Value == "" {
		return details.SelectedIndex == -1 && selectedIndex == -1
	}
	return validIdentifier(details.Value, limits.IdentifierBytes) &&
		details.SelectedIndex == selectedIndex &&
		selectedIndex >= 0 &&
		details.Options[selectedIndex].Value == details.Value
}

func validActionDetails(action *ActionDetails, limits Limits) bool {
	if action == nil ||
		!canonicalDisplayText(action.Label, false) ||
		!validIdentifier(action.Command, limits.IdentifierBytes) ||
		len(action.DisabledReason) > maxDisplayTextBytes ||
		!utf8.ValidString(action.DisabledReason) ||
		strings.ContainsRune(action.DisabledReason, 0) ||
		(action.Enabled && action.DisabledReason != "") ||
		(!action.Enabled && action.DisabledReason == "") ||
		(action.Pressed && !action.Enabled) ||
		(action.Default && action.Cancel) {
		return false
	}
	return action.Mnemonic == "" ||
		(len(action.Mnemonic) == 1 &&
			validLogicalKey(string(action.Mnemonic)))
}

func validHotkeyBarDetails(
	bar *HotkeyBarDetails,
	limits Limits,
) bool {
	if bar == nil || len(bar.Items) > expletives.MaxHotkeyBarItems {
		return false
	}
	seen := make(map[string]bool, len(bar.Items))
	for _, item := range bar.Items {
		if !canonicalDisplayText(item.Label, false) ||
			!validIdentifier(item.Command, limits.IdentifierBytes) ||
			seen[item.Command] ||
			len(item.DisabledReason) > maxDisplayTextBytes ||
			!utf8.ValidString(item.DisabledReason) ||
			strings.ContainsRune(item.DisabledReason, 0) ||
			(item.Enabled && item.DisabledReason != "") ||
			(!item.Enabled && item.DisabledReason == "") ||
			(item.Chord != nil && !validChord(*item.Chord)) {
			return false
		}
		seen[item.Command] = true
	}
	return true
}

func validStatusBarDetails(
	bar *StatusBarDetails,
	controlWidth int,
	limits Limits,
) bool {
	if bar == nil || controlWidth < 0 ||
		len(bar.Segments) > expletives.MaxStatusBarSegments {
		return false
	}
	seen := make(map[string]bool, len(bar.Segments))
	nextX := 0
	for _, segment := range bar.Segments {
		if !validIdentifier(segment.Key, limits.IdentifierBytes) ||
			seen[segment.Key] ||
			segment.Label == "" ||
			!canonicalDisplayText(segment.Label, false) ||
			(segment.Command != "" &&
				!validIdentifier(segment.Command, limits.IdentifierBytes)) ||
			len(segment.DisabledReason) > maxDisplayTextBytes ||
			!utf8.ValidString(segment.DisabledReason) ||
			strings.ContainsRune(segment.DisabledReason, 0) ||
			(segment.Enabled && segment.DisabledReason != "") ||
			(!segment.Enabled && segment.DisabledReason == "") ||
			(segment.Command == "" &&
				(!segment.Enabled ||
					segment.DisabledReason != "" ||
					segment.Checked ||
					segment.Chord != nil)) ||
			(segment.Chord != nil && !validChord(*segment.Chord)) ||
			segment.Bounds.X < 0 ||
			segment.Bounds.Y != 0 ||
			segment.Bounds.Width < 0 ||
			segment.Bounds.Height < 0 ||
			(!segment.Rendered &&
				(segment.Bounds != (Rect{}) || segment.Clipped)) ||
			(segment.Rendered &&
				(segment.Bounds.X != nextX ||
					segment.Bounds.Height != 1 ||
					segment.Bounds.Width <= 0 ||
					nextX > controlWidth ||
					segment.Bounds.Width > controlWidth-nextX)) {
			return false
		}
		if segment.Rendered {
			nextX += segment.Bounds.Width
		}
		seen[segment.Key] = true
	}
	return true
}

func validMenuBarDetails(
	bar *MenuBarDetails,
	limits Limits,
) bool {
	if bar == nil || len(bar.Entries) == 0 ||
		len(bar.Entries) > expletives.MaxMenuItems ||
		len(bar.OpenPath) > expletives.MaxMenuDepth ||
		len(bar.SelectedPath) > expletives.MaxMenuDepth+1 {
		return false
	}
	entries := make(map[string]MenuEntryDetails, len(bar.Entries))
	children := make(map[string]int)
	siblingMnemonics := make(map[string]map[Key]bool)
	menuCount := 0
	rootCount := 0
	for _, entry := range bar.Entries {
		if !validIdentifier(entry.Key, limits.IdentifierBytes) ||
			(entry.ParentKey != "" &&
				!validIdentifier(entry.ParentKey, limits.IdentifierBytes)) ||
			entry.Depth < 0 || entry.Depth > expletives.MaxMenuDepth ||
			!canonicalDisplayText(entry.Label, false) ||
			len(entry.DisabledReason) > maxDisplayTextBytes ||
			!utf8.ValidString(entry.DisabledReason) ||
			strings.ContainsRune(entry.DisabledReason, 0) ||
			entry.ChildCount < 0 ||
			entry.ChildCount > expletives.MaxMenuItemsPerMenu {
			return false
		}
		if _, exists := entries[entry.Key]; exists {
			return false
		}
		if entry.Depth == 0 {
			if entry.ParentKey != "" ||
				(entry.Placement != "" &&
					entry.Placement != "start" &&
					entry.Placement != "end") {
				return false
			}
			rootCount++
			if rootCount > expletives.MaxMenuItemsPerMenu {
				return false
			}
		} else {
			parent, exists := entries[entry.ParentKey]
			if !exists || parent.Kind != "submenu" ||
				parent.Depth+1 != entry.Depth ||
				entry.Placement != "" {
				return false
			}
			children[entry.ParentKey]++
		}
		if entry.Mnemonic != "" {
			if len(entry.Mnemonic) != 1 ||
				!validLogicalKey(string(entry.Mnemonic)) {
				return false
			}
			seen := siblingMnemonics[entry.ParentKey]
			if seen == nil {
				seen = make(map[Key]bool)
				siblingMnemonics[entry.ParentKey] = seen
			}
			if seen[entry.Mnemonic] {
				return false
			}
			seen[entry.Mnemonic] = true
		}
		switch entry.Kind {
		case "command":
			if entry.Depth == 0 ||
				entry.Label == "" ||
				!validIdentifier(entry.Command, limits.IdentifierBytes) ||
				entry.ChildCount != 0 ||
				(entry.Enabled && entry.DisabledReason != "") ||
				(!entry.Enabled && entry.DisabledReason == "") ||
				(entry.Chord != nil && !validChord(*entry.Chord)) ||
				entry.Open {
				return false
			}
		case "separator":
			if entry.Depth == 0 || entry.Label != "" ||
				entry.Command != "" || entry.Enabled ||
				entry.DisabledReason != "" || entry.Checked ||
				entry.Mnemonic != "" || entry.Chord != nil ||
				entry.Selected || entry.Open || entry.ChildCount != 0 {
				return false
			}
		case "submenu":
			menuCount++
			if entry.Label == "" || entry.Command != "" ||
				!entry.Enabled || entry.DisabledReason != "" ||
				entry.Checked || entry.Chord != nil ||
				entry.ChildCount == 0 {
				return false
			}
		default:
			return false
		}
		entries[entry.Key] = entry
	}
	if menuCount > expletives.MaxMenus {
		return false
	}
	for key, entry := range entries {
		if entry.Kind == "submenu" &&
			children[key] != entry.ChildCount {
			return false
		}
	}
	if !validMenuPath(bar.OpenPath, entries, true, limits) ||
		!validMenuPath(bar.SelectedPath, entries, false, limits) ||
		len(bar.OpenPath) > len(bar.SelectedPath) ||
		len(bar.SelectedPath) > len(bar.OpenPath)+1 ||
		(len(bar.OpenPath) == 0 && len(bar.SelectedPath) > 1) {
		return false
	}
	for index, key := range bar.OpenPath {
		if bar.SelectedPath[index] != key {
			return false
		}
	}
	selected := make(map[string]bool, len(bar.SelectedPath))
	for _, key := range bar.SelectedPath {
		selected[key] = true
	}
	open := make(map[string]bool, len(bar.OpenPath))
	for _, key := range bar.OpenPath {
		open[key] = true
	}
	for key, entry := range entries {
		if entry.Selected != selected[key] ||
			entry.Open != open[key] {
			return false
		}
	}
	return true
}

func validMenuPath(
	path []string,
	entries map[string]MenuEntryDetails,
	requireSubmenu bool,
	limits Limits,
) bool {
	for index, key := range path {
		if !validIdentifier(key, limits.IdentifierBytes) {
			return false
		}
		entry, exists := entries[key]
		if !exists || (requireSubmenu && entry.Kind != "submenu") {
			return false
		}
		if index == 0 {
			if entry.Depth != 0 {
				return false
			}
		} else if entry.ParentKey != path[index-1] {
			return false
		}
	}
	return true
}

func validChord(chord Chord) bool {
	if !validLogicalKey(string(chord.Key)) ||
		isLogicalModifier(chord.Key) ||
		len(chord.Modifiers) > 4 {
		return false
	}
	seen := make(map[Key]bool, len(chord.Modifiers))
	previous := Key("")
	for _, modifier := range chord.Modifiers {
		if !isLogicalModifier(modifier) || seen[modifier] ||
			(previous != "" && modifier < previous) {
			return false
		}
		seen[modifier] = true
		previous = modifier
	}
	return true
}

func isLogicalModifier(key Key) bool {
	switch key {
	case "alt", "control", "meta", "shift":
		return true
	default:
		return false
	}
}

func validTextDetails(
	text *TextDetails,
	limits Limits,
	multiline bool,
) bool {
	if text == nil ||
		!canonicalDisplayText(text.Text, multiline) ||
		!validTextAlignment(text.HorizontalAlignment) ||
		!validTextAlignment(text.VerticalAlignment) ||
		!validTextWrap(text.Wrap) ||
		(text.Target != "" &&
			!validIdentifier(string(text.Target), limits.IdentifierBytes)) {
		return false
	}
	if text.Mnemonic == "" {
		return true
	}
	return text.Target != "" &&
		len(text.Mnemonic) == 1 &&
		validLogicalKey(string(text.Mnemonic))
}

func validDividerDetails(divider *DividerDetails) bool {
	return divider != nil &&
		(divider.Orientation == Orientation(expletives.Horizontal) ||
			divider.Orientation == Orientation(expletives.Vertical)) &&
		validBorderForm(divider.Form) &&
		canonicalDisplayText(divider.Text, false) &&
		validTextAlignment(divider.Alignment)
}

func validTextAlignment(alignment TextAlignment) bool {
	switch alignment {
	case "start", "center", "end":
		return true
	default:
		return false
	}
}

func validTextWrap(wrap TextWrap) bool {
	switch wrap {
	case "none", "words", "cells":
		return true
	default:
		return false
	}
}

func canonicalDisplayText(value string, multiline bool) bool {
	if len(value) > maxDisplayTextBytes || !utf8.ValidString(value) {
		return false
	}
	lines := strings.Split(value, "\n")
	cellCount := len(lines) - 1
	for _, current := range value {
		if current == '\n' {
			if multiline {
				continue
			}
			return false
		}
		if unicode.IsControl(current) {
			return false
		}
	}
	for _, line := range lines {
		cells := display.Normalize(line)
		cellCount += len(cells)
		if cellCount > maxDisplayTextCells ||
			strings.Join(cells, "") != line {
			return false
		}
		for _, cell := range cells {
			if len(cell) > maxCellGraphemeBytes {
				return false
			}
		}
	}
	return true
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
