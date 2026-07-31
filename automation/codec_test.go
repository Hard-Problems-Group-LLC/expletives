package automation

import (
	"bufio"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDecodeRequest(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	tests := []struct {
		name      string
		record    string
		operation string
		requestID string
	}{
		{
			name:      "observe latest",
			record:    `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"observe-1"}`,
			operation: TypeObserve,
			requestID: "observe-1",
		},
		{
			name:      "wait",
			record:    `{"protocol":"expletives.automation","version":1,"type":"wait_snapshot","request_id":"wait-1","after_sequence":4,"timeout_ms":1000}`,
			operation: TypeWaitSnapshot,
			requestID: "wait-1",
		},
		{
			name:      "key press",
			record:    `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"key-1","event":{"kind":"key_press","key":"r"}}`,
			operation: TypeInjectInput,
			requestID: "key-1",
		},
		{
			name:      "left bracket key press",
			record:    `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"key-left","event":{"kind":"key_press","key":"["}}`,
			operation: TypeInjectInput,
			requestID: "key-left",
		},
		{
			name:      "right bracket key press",
			record:    `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"key-right","event":{"kind":"key_press","key":"]"}}`,
			operation: TypeInjectInput,
			requestID: "key-right",
		},
		{
			name:      "unicode text key press",
			record:    `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"key-unicode","event":{"kind":"key_press","key":"é"}}`,
			operation: TypeInjectInput,
			requestID: "key-unicode",
		},
		{
			name:      "command",
			record:    `{"protocol":"expletives.automation","version":1,"type":"invoke_command","request_id":"command-1","command":"scenario.reset","target_key":"root"}`,
			operation: TypeInvokeCommand,
			requestID: "command-1",
		},
		{
			name:      "reset",
			record:    `{"protocol":"expletives.automation","version":1,"type":"reset_input","request_id":"reset-1"}`,
			operation: TypeResetInput,
			requestID: "reset-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			decoded, err := decodeRequest([]byte(test.record), limits)
			if err != nil {
				t.Fatalf("decodeRequest() error = %v", err)
			}
			if decoded.operation != test.operation {
				t.Errorf("operation = %q, want %q", decoded.operation, test.operation)
			}
			if decoded.requestID != test.requestID {
				t.Errorf("request ID = %q, want %q", decoded.requestID, test.requestID)
			}
		})
	}
}

func TestDecodeRequestRejectsInvalidRecords(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	tests := []struct {
		name string
		line string
		code string
	}{
		{
			name: "duplicate field",
			line: `{"protocol":"expletives.automation","protocol":"expletives.automation","version":1,"type":"observe","request_id":"r1"}`,
			code: "malformed_json",
		},
		{
			name: "unknown field",
			line: `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"r1","surprise":true}`,
			code: "invalid_request",
		},
		{
			name: "wrong protocol",
			line: `{"protocol":"other","version":1,"type":"observe","request_id":"r1"}`,
			code: "unsupported_protocol",
		},
		{
			name: "wrong version",
			line: `{"protocol":"expletives.automation","version":2,"type":"observe","request_id":"r1"}`,
			code: "unsupported_version",
		},
		{
			name: "invalid request ID",
			line: `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"bad id"}`,
			code: "invalid_request",
		},
		{
			name: "unsupported key",
			line: `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"r1","event":{"kind":"key_press","key":"ab"}}`,
			code: "invalid_request",
		},
		{
			name: "terminal bytes are not a key",
			line: `{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"r1","event":{"kind":"key_press","key":"\u001b[A"}}`,
			code: "invalid_request",
		},
		{
			name: "wait exceeds bound",
			line: `{"protocol":"expletives.automation","version":1,"type":"wait_snapshot","request_id":"r1","after_sequence":1,"timeout_ms":30001}`,
			code: "invalid_request",
		},
		{
			name: "wait omits sequence",
			line: `{"protocol":"expletives.automation","version":1,"type":"wait_snapshot","request_id":"r1"}`,
			code: "invalid_request",
		},
		{
			name: "optional sequence cannot be null",
			line: `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"r1","frame_sequence":null}`,
			code: "invalid_request",
		},
		{
			name: "optional target cannot be null",
			line: `{"protocol":"expletives.automation","version":1,"type":"invoke_command","request_id":"r1","command":"scenario.reset","target_key":null}`,
			code: "invalid_request",
		},
		{
			name: "legacy target field is not accepted",
			line: `{"protocol":"expletives.automation","version":1,"type":"invoke_command","request_id":"r1","command":"scenario.reset","target":"root"}`,
			code: "invalid_request",
		},
		{
			name: "cancel is not a version one operation",
			line: `{"protocol":"expletives.automation","version":1,"type":"cancel","request_id":"r1","target_request_id":"r0"}`,
			code: "unknown_type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeRequest([]byte(test.line), limits)
			var failure *decodeError
			if !errors.As(err, &failure) {
				t.Fatalf("decodeRequest() error = %v, want *decodeError", err)
			}
			if failure.code != test.code {
				t.Errorf("code = %q, want %q", failure.code, test.code)
			}
		})
	}
}

func TestDecodeRequestRejectsExcessiveDepth(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	nested := strings.Repeat(`{"x":`, limits.JSONDepth+1) +
		`0` +
		strings.Repeat(`}`, limits.JSONDepth+1)
	line := `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"r1","extra":` +
		nested +
		`}`

	_, err := decodeRequest([]byte(line), limits)
	var failure *decodeError
	if !errors.As(err, &failure) {
		t.Fatalf("decodeRequest() error = %v, want *decodeError", err)
	}
	if failure.code != "malformed_json" {
		t.Errorf("code = %q, want malformed_json", failure.code)
	}
}

func TestDecodeRequestRejectsOversizedLine(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	record := []byte(strings.Repeat("x", limits.RequestLineBytes+1))
	_, err := decodeRequest(record, limits)
	var failure *decodeError
	if !errors.As(err, &failure) {
		t.Fatalf("decodeRequest() error = %v, want *decodeError", err)
	}
	if failure.code != "line_too_long" {
		t.Errorf("code = %q, want line_too_long", failure.code)
	}
}

func TestEncodeLineBound(t *testing.T) {
	t.Parallel()

	if _, err := encodeLine(map[string]string{"value": strings.Repeat("x", 128)}, 32); err == nil {
		t.Fatal("encodeLine() error = nil, want response-size error")
	}
}

func TestBoundedMessageProducesValidNULFreeText(t *testing.T) {
	t.Parallel()

	got := boundedMessage("  invalid\x00\xffmessage  ")
	if !utf8.ValidString(got) {
		t.Fatalf("boundedMessage() = %q, want valid UTF-8", got)
	}
	if strings.ContainsRune(got, 0) {
		t.Fatalf("boundedMessage() = %q, want NUL-free text", got)
	}
	if len(got) > maxErrorMessageBytes {
		t.Fatalf(
			"boundedMessage() length = %d, want <= %d",
			len(got),
			maxErrorMessageBytes,
		)
	}
}

func TestReadLineAcrossSmallReaderBuffer(t *testing.T) {
	t.Parallel()

	const maxBytes = 256
	record := strings.Repeat("x", maxBytes)
	reader := bufio.NewReaderSize(strings.NewReader(record+"\n"), 16)
	got, err := readLine(nil, reader, maxBytes)
	if err != nil {
		t.Fatalf("readLine() error = %v", err)
	}
	if string(got) != record {
		t.Fatalf("readLine() length = %d, want %d", len(got), len(record))
	}
}

func TestReadLineRejectsOverlongRecordAcrossSmallReaderBuffer(t *testing.T) {
	t.Parallel()

	const maxBytes = 256
	reader := bufio.NewReaderSize(
		strings.NewReader(strings.Repeat("x", maxBytes+1)+"\n"),
		16,
	)
	if _, err := readLine(nil, reader, maxBytes); !errors.Is(err, errLineTooLong) {
		t.Fatalf("readLine() error = %v, want errLineTooLong", err)
	}
}

func TestLimitsRejectVersionOneExpansion(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	limits.RetainedResults++
	if err := limits.validate(); err == nil {
		t.Fatal("Limits.validate() error = nil after increasing retained completion cap")
	}
}

func TestNewRequestID(t *testing.T) {
	t.Parallel()

	first, err := NewRequestID()
	if err != nil {
		t.Fatalf("NewRequestID() error = %v", err)
	}
	second, err := NewRequestID()
	if err != nil {
		t.Fatalf("NewRequestID() second error = %v", err)
	}
	if first == second {
		t.Fatalf("NewRequestID() repeated %q", first)
	}
	if !validIdentifier(first, DefaultLimits().RequestIDBytes) {
		t.Errorf("request ID %q is not valid", first)
	}
}

func FuzzDecodeRequest(f *testing.F) {
	limits := DefaultLimits()
	for _, seed := range []string{
		`{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"r1"}`,
		`{"protocol":"expletives.automation","version":1,"type":"inject_input","request_id":"r2","event":{"kind":"key_down","key":"control"}}`,
		`{"protocol":"expletives.automation","version":1,"type":"invoke_command","request_id":"r3","command":"scenario.reset"}`,
		`{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"r4","request_id":"duplicate"}`,
		``,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, record []byte) {
		if len(record) > limits.RequestLineBytes+1 {
			return
		}
		decoded, err := decodeRequest(record, limits)
		if err != nil {
			return
		}
		if decoded.operation == "" {
			t.Error("accepted decoded request has an empty operation")
		}
		if err := validateRequestID(decoded.requestID, limits); err != nil {
			t.Errorf("accepted decoded request has invalid request ID: %v", err)
		}
	})
}
