// Package automation implements the bounded, local drive-and-observe
// protocol for an expletives application.
//
// Version 1 is deliberately unauthenticated. Constructing a Server is an
// explicit per-process trust decision suitable only for an operator-controlled
// Unix-socket path and non-risky application data. A normal application run
// must not construct a Server. Request IDs remain unique while active or
// retained; a client must never automatically replay an indeterminate request.
//
// Version 1 authenticates neither controller nor server and performs no
// capability authorization. Anyone able to connect can observe and drive the
// application with its privileges. The session ID and socket mode are not
// credentials. Do not use this protocol for hostile multi-user, elevated,
// remote, or security-sensitive operation.
package automation

import (
	"fmt"
	"regexp"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const (
	// Protocol is the protocol name carried by every wire record.
	Protocol = "expletives.automation"
	// Version is the only protocol version implemented by this package.
	Version = 1
)

const (
	// TypeHello identifies the server's initial capability record.
	TypeHello = "hello"
	// TypeAccepted identifies a nonterminal request-acceptance record.
	TypeAccepted = "accepted"
	// TypeCompletion identifies an accepted request's terminal response.
	TypeCompletion = "completion"
	// TypeProtocolError identifies a pre-acceptance rejection.
	TypeProtocolError = "protocol_error"

	// TypeObserve identifies latest or exact retained snapshot observation.
	TypeObserve = "observe"
	// TypeWaitSnapshot identifies waiting for a later snapshot.
	TypeWaitSnapshot = "wait_snapshot"
	// TypeInjectInput identifies one raw logical key lifecycle event.
	TypeInjectInput = "inject_input"
	// TypeInvokeCommand identifies direct semantic command invocation.
	TypeInvokeCommand = "invoke_command"
	// TypeQueryResult identifies request-result reconciliation.
	TypeQueryResult = "query_result"
	// TypeResetInput identifies held-key cleanup for the controller.
	TypeResetInput = "reset_input"
	// TypeShutdown identifies orderly exit through application command policy.
	TypeShutdown = "shutdown"
)

var versionOneOperations = [...]string{
	TypeObserve,
	TypeWaitSnapshot,
	TypeInjectInput,
	TypeInvokeCommand,
	TypeQueryResult,
	TypeResetInput,
	TypeShutdown,
}

const (
	// OutcomeApplied reports an accepted state change or action.
	OutcomeApplied = "applied"
	// OutcomeNoOp reports successful handling that changed no state.
	OutcomeNoOp = "no_op"
	// OutcomeRejected reports refusal by current policy or state.
	OutcomeRejected = "rejected"
	// OutcomeCancelled reports cancellation.
	OutcomeCancelled = "cancelled"
	// OutcomeInterrupted reports an accepted interrupt.
	OutcomeInterrupted = "interrupted"
	// OutcomeExited reports orderly application exit.
	OutcomeExited = "exited"
	// OutcomeFailed reports an attempted operation that failed.
	OutcomeFailed = "failed"
)

const (
	// KeyDown marks a logical key held for later chord resolution.
	KeyDown = "key_down"
	// KeyUp releases a held logical key.
	KeyUp = "key_up"
	// KeyPress is a one-shot logical press that leaves no key held.
	KeyPress = "key_press"
)

const (
	maxCellGraphemeBytes    = 64
	maxBorderTitleBytes     = 256
	maxBorderTitleCells     = expletives.MaxTitleCells
	maxDisplayTextBytes     = expletives.MaxDisplayTextBytes
	maxDisplayTextCells     = expletives.MaxDisplayTextCells
	maxErrorMessageBytes    = 256
	maxSnapshotMessageBytes = 1024
	sha256HexBytes          = 64
	lineReaderBufferBytes   = 64 << 10
	// maxRetainedResponseBytes is the aggregate JSON evidence budget for
	// retained v1 completions. Three maximum legal records fit below 128 MiB;
	// ordinary snapshots consume substantially less.
	maxRetainedResponseBytes = 128 << 20
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

// Limits are the advertised version 1 resource bounds for one server. The zero
// value selects DefaultLimits in ServerOptions; any other ServerOptions value
// must equal that fixed default.
type Limits struct {
	// RequestLineBytes and ResponseLineBytes exclude the newline delimiter.
	RequestLineBytes  int `json:"request_line_bytes"`
	ResponseLineBytes int `json:"response_line_bytes"`
	// JSONDepth bounds nested objects and arrays.
	JSONDepth int `json:"json_depth"`
	// RequestIDBytes bounds request correlation identifiers.
	RequestIDBytes int `json:"request_id_bytes"`
	// IdentifierBytes bounds other protocol identifiers.
	IdentifierBytes int `json:"identifier_bytes"`
	// Clients is the simultaneous-controller limit.
	Clients int `json:"clients"`
	// OutstandingPerConnection is the sequential in-flight request limit.
	OutstandingPerConnection int `json:"outstanding_per_connection"`
	// RetainedResults is the completed-request FIFO length.
	RetainedResults int `json:"retained_results"`
	// FrameWidth and FrameHeight bound snapshot geometry in cells.
	FrameWidth  int `json:"frame_width"`
	FrameHeight int `json:"frame_height"`
	// FrameCells bounds a snapshot's row-major cell array.
	FrameCells int `json:"frame_cells"`
	// FrameRuns bounds the compact row-major wire representation.
	FrameRuns int `json:"frame_runs"`
	// Controls bounds controls and aggregate child references.
	Controls int `json:"controls"`
	// Layouts and LayoutItems bound flat Layout observations and their
	// aggregate item references.
	Layouts     int `json:"layouts"`
	LayoutItems int `json:"layout_items"`
	// HeldKeys bounds held keys per input source.
	HeldKeys int `json:"held_keys"`
	// ReadTimeoutMillis bounds idle reads and normal server execution.
	ReadTimeoutMillis int `json:"read_timeout_ms"`
	// WriteTimeoutMillis bounds each response write.
	WriteTimeoutMillis int `json:"write_timeout_ms"`
}

// DefaultLimits returns the fixed version 1 limits. The returned value contains
// no shared mutable state.
func DefaultLimits() Limits {
	return Limits{
		RequestLineBytes:         64 << 10,
		ResponseLineBytes:        40 << 20,
		JSONDepth:                16,
		RequestIDBytes:           64,
		IdentifierBytes:          64,
		Clients:                  1,
		OutstandingPerConnection: 1,
		RetainedResults:          3,
		FrameWidth:               expletives.MaxFrameCells,
		FrameHeight:              expletives.MaxFrameCells,
		FrameCells:               expletives.MaxFrameCells,
		FrameRuns:                16 * 1024,
		Controls:                 expletives.MaxControls,
		Layouts:                  expletives.MaxLayouts,
		LayoutItems:              expletives.MaxLayoutItems,
		HeldKeys:                 expletives.MaxHeldKeysPerSource,
		ReadTimeoutMillis:        30_000,
		WriteTimeoutMillis:       5_000,
	}
}

func (l Limits) validate() error {
	values := map[string]int{
		"request line bytes":         l.RequestLineBytes,
		"response line bytes":        l.ResponseLineBytes,
		"JSON depth":                 l.JSONDepth,
		"request ID bytes":           l.RequestIDBytes,
		"identifier bytes":           l.IdentifierBytes,
		"clients":                    l.Clients,
		"outstanding per connection": l.OutstandingPerConnection,
		"retained results":           l.RetainedResults,
		"frame width":                l.FrameWidth,
		"frame height":               l.FrameHeight,
		"frame cells":                l.FrameCells,
		"frame runs":                 l.FrameRuns,
		"controls":                   l.Controls,
		"layouts":                    l.Layouts,
		"layout items":               l.LayoutItems,
		"held keys":                  l.HeldKeys,
		"read timeout milliseconds":  l.ReadTimeoutMillis,
		"write timeout milliseconds": l.WriteTimeoutMillis,
	}
	for name, value := range values {
		if value <= 0 {
			return fmt.Errorf("%s must be positive", name)
		}
	}
	if l.Clients != 1 || l.OutstandingPerConnection != 1 {
		return fmt.Errorf("protocol version 1 requires one client and one outstanding request")
	}
	maximum := DefaultLimits()
	bounded := map[string]struct {
		value   int
		maximum int
	}{
		"request line bytes":         {l.RequestLineBytes, maximum.RequestLineBytes},
		"response line bytes":        {l.ResponseLineBytes, maximum.ResponseLineBytes},
		"JSON depth":                 {l.JSONDepth, maximum.JSONDepth},
		"request ID bytes":           {l.RequestIDBytes, maximum.RequestIDBytes},
		"identifier bytes":           {l.IdentifierBytes, maximum.IdentifierBytes},
		"retained results":           {l.RetainedResults, maximum.RetainedResults},
		"frame width":                {l.FrameWidth, maximum.FrameWidth},
		"frame height":               {l.FrameHeight, maximum.FrameHeight},
		"frame cells":                {l.FrameCells, maximum.FrameCells},
		"frame runs":                 {l.FrameRuns, maximum.FrameRuns},
		"controls":                   {l.Controls, maximum.Controls},
		"layouts":                    {l.Layouts, maximum.Layouts},
		"layout items":               {l.LayoutItems, maximum.LayoutItems},
		"held keys":                  {l.HeldKeys, maximum.HeldKeys},
		"read timeout milliseconds":  {l.ReadTimeoutMillis, maximum.ReadTimeoutMillis},
		"write timeout milliseconds": {l.WriteTimeoutMillis, maximum.WriteTimeoutMillis},
	}
	for name, bound := range bounded {
		if bound.value > bound.maximum {
			return fmt.Errorf("%s exceeds protocol version 1 maximum %d", name, bound.maximum)
		}
	}
	if l.FrameWidth > l.FrameCells || l.FrameHeight > l.FrameCells {
		return fmt.Errorf("frame dimensions exceed frame cell limit")
	}
	return nil
}

func (l Limits) readTimeout() time.Duration {
	return time.Duration(l.ReadTimeoutMillis) * time.Millisecond
}

func (l Limits) writeTimeout() time.Duration {
	return time.Duration(l.WriteTimeoutMillis) * time.Millisecond
}

// Header is present in every protocol record.
type Header struct {
	// Protocol must equal Protocol.
	Protocol string `json:"protocol"`
	// Version must equal Version for operational version 1 records.
	Version int `json:"version"`
	// Type selects the record shape.
	Type string `json:"type"`
}

func newHeader(messageType string) Header {
	return Header{Protocol: Protocol, Version: Version, Type: messageType}
}

// Hello is the first record written on an accepted controller connection.
type Hello struct {
	Header
	// Application is the required server-selected application identity.
	Application string `json:"application"`
	// SessionID is correlation data, not a credential.
	SessionID string `json:"session_id"`
	// Unauthenticated is true for version 1.
	Unauthenticated bool `json:"unauthenticated"`
	// SupportedVersions is the server's bounded protocol inventory.
	SupportedVersions []int `json:"supported_versions"`
	// Scenario is the App's stable scenario identity.
	Scenario string `json:"scenario"`
	// LatestFrameSequence and Final describe state when Hello was created.
	LatestFrameSequence uint64 `json:"latest_frame_sequence"`
	Final               bool   `json:"final"`
	// Operations and Commands advertise bounded available capabilities.
	Operations []string `json:"operations"`
	Commands   []string `json:"commands"`
	// Limits are the bounds the client validates and enforces.
	Limits Limits `json:"limits"`
}

// Accepted confirms that a request was registered. It is not a terminal
// result.
type Accepted struct {
	Header
	// RequestID is the reserved caller correlation ID.
	RequestID string `json:"request_id"`
	// Operation is the accepted request type.
	Operation string `json:"operation"`
}

// Error describes a bounded protocol or request failure.
type Error struct {
	// Code is the stable machine-readable identifier.
	Code string `json:"code"`
	// Message is bounded operator-facing text.
	Message string `json:"message"`
	// Retryable is advisory and never authorizes automatic request replay.
	Retryable bool `json:"retryable"`
}

// ProtocolError rejects a record before it becomes an accepted request.
type ProtocolError struct {
	Header
	// RequestID is present only when safely recovered from the rejected record.
	RequestID string `json:"request_id,omitempty"`
	Error
}

// Completion is the single terminal response for an accepted request.
// Snapshot is the immutable state associated with FrameSequence.
type Completion struct {
	Header
	// RequestID and Operation match the preceding Accepted record.
	RequestID string `json:"request_id"`
	Operation string `json:"operation"`
	// Outcome is the actual terminal operation result.
	Outcome string `json:"outcome"`
	// FrameSequence identifies Snapshot exactly.
	FrameSequence uint64 `json:"frame_sequence"`
	// Snapshot is the associated bounded observation and is always present.
	Snapshot *SnapshotV1 `json:"snapshot"`
	// Error is an optional bounded diagnostic; Outcome remains authoritative.
	Error *Error `json:"error,omitempty"`
	// Result is present only for operation-specific terminal data.
	Result *Result `json:"result,omitempty"`
}

// Result carries operation-specific bounded terminal data.
type Result struct {
	// Query is present only for a query_result completion.
	Query *QueryResult `json:"query,omitempty"`
}

// QueryResult describes a retained target request.
type QueryResult struct {
	// TargetRequestID is the request being reconciled.
	TargetRequestID string `json:"target_request_id"`
	// Status is pending, completed, or unknown_or_expired.
	Status string `json:"status"`
	// Completion is present only for completed.
	Completion *RetainedCompletion `json:"completion,omitempty"`
}

// RetainedCompletion is a non-recursive terminal result returned by
// query_result. In particular, querying an earlier query cannot create an
// unbounded result chain.
type RetainedCompletion struct {
	// RequestID, Operation, Outcome, and FrameSequence describe the target.
	RequestID     string `json:"request_id"`
	Operation     string `json:"operation"`
	Outcome       string `json:"outcome"`
	FrameSequence uint64 `json:"frame_sequence"`
	// Error is the target's optional bounded diagnostic.
	Error *Error `json:"error,omitempty"`
}

// KeyEvent is a device-independent event before command binding.
type KeyEvent struct {
	// Kind is KeyDown, KeyUp, or KeyPress.
	Kind string `json:"kind"`
	// Key is one supported logical key identity, not terminal bytes.
	Key string `json:"key"`
}

type observeRequest struct {
	Header
	RequestID     string  `json:"request_id"`
	FrameSequence *uint64 `json:"frame_sequence,omitempty"`
}

type waitSnapshotRequest struct {
	Header
	RequestID     string `json:"request_id"`
	AfterSequence uint64 `json:"after_sequence"`
	TimeoutMillis int    `json:"timeout_ms,omitempty"`
}

type injectInputRequest struct {
	Header
	RequestID string   `json:"request_id"`
	Event     KeyEvent `json:"event"`
}

type invokeCommandRequest struct {
	Header
	RequestID string `json:"request_id"`
	Command   string `json:"command"`
	TargetKey string `json:"target_key,omitempty"`
}

type queryResultRequest struct {
	Header
	RequestID       string `json:"request_id"`
	TargetRequestID string `json:"target_request_id"`
}

type simpleRequest struct {
	Header
	RequestID string `json:"request_id"`
}

func validIdentifier(value string, limit int) bool {
	return value != "" && len(value) <= limit && identifierPattern.MatchString(value)
}
