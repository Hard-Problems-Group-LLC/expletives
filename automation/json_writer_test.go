package automation

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBoundedJSONEncodingMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	sequence := uint64(42)
	sensitiveText := "<html>&\u2028\u2029\"\\\n" +
		string([]byte{0xff})
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "hello",
			value: validTestHello(limits),
		},
		{
			name: "accepted",
			value: Accepted{
				Header:    newHeader(TypeAccepted),
				RequestID: "request-1",
				Operation: TypeObserve,
			},
		},
		{
			name: "protocol error embedded header and error",
			value: ProtocolError{
				Header: newHeader(TypeProtocolError),
				Error: Error{
					Code:      "invalid_request",
					Message:   sensitiveText,
					Retryable: false,
				},
			},
		},
		{
			name:  "completion full nested shape",
			value: maximumValidElementCompletion(limits),
		},
		{
			name: "completion omitempty",
			value: Completion{
				Header:        newHeader(TypeCompletion),
				RequestID:     "request-1",
				Operation:     TypeObserve,
				Outcome:       OutcomeNoOp,
				FrameSequence: 1,
				Snapshot: &SnapshotV1{
					Version:   1,
					Sequence:  1,
					Scenario:  "scenario",
					Controls:  []ControlSnapshot{},
					Overflows: []OverflowSnapshot{},
				},
			},
		},
		{
			name: "observe request with optional sequence",
			value: observeRequest{
				Header:        newHeader(TypeObserve),
				RequestID:     "request-1",
				FrameSequence: &sequence,
			},
		},
		{
			name: "observe request omitempty",
			value: observeRequest{
				Header:    newHeader(TypeObserve),
				RequestID: "request-1",
			},
		},
		{
			name: "wait snapshot request",
			value: waitSnapshotRequest{
				Header:        newHeader(TypeWaitSnapshot),
				RequestID:     "request-1",
				AfterSequence: 41,
				TimeoutMillis: 1000,
			},
		},
		{
			name: "inject input request",
			value: injectInputRequest{
				Header:    newHeader(TypeInjectInput),
				RequestID: "request-1",
				Event: KeyEvent{
					Kind: KeyPress,
					Key:  "r",
				},
			},
		},
		{
			name: "invoke command request",
			value: invokeCommandRequest{
				Header:    newHeader(TypeInvokeCommand),
				RequestID: "request-1",
				Command:   "scenario.reset",
				TargetKey: "panel.main",
			},
		},
		{
			name: "query result request",
			value: queryResultRequest{
				Header:          newHeader(TypeQueryResult),
				RequestID:       "request-1",
				TargetRequestID: "request-0",
			},
		},
		{
			name: "simple request",
			value: simpleRequest{
				Header:    newHeader(TypeShutdown),
				RequestID: "request-1",
			},
		},
		{
			name: "HTML and Unicode escaping",
			value: struct {
				Text  string `json:"text"`
				Empty string `json:"empty,omitempty"`
			}{
				Text: sensitiveText,
			},
		},
		{
			name: "sorted string map",
			value: map[string]string{
				"z": sensitiveText,
				"a": "first",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			want, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			got, err := encodeLine(test.value, len(want))
			if err != nil {
				t.Fatalf("encodeLine() error = %v", err)
			}
			want = append(want, '\n')
			if !bytes.Equal(got, want) {
				t.Fatalf("encodeLine() = %q, want %q", got, want)
			}
		})
	}
}

func TestBoundedJSONEncodingRejectsOneBeyondWithoutGrowingPastLimit(t *testing.T) {
	t.Parallel()

	value := struct {
		Text string `json:"text"`
	}{
		Text: strings.Repeat("x", 1<<20),
	}
	const limit = 64
	writer := &boundedJSONBuffer{limit: limit}
	err := writeJSONValue(writer, reflect.ValueOf(value), 0)
	if err != errJSONRecordTooLong {
		t.Fatalf("writeJSONValue() error = %v, want errJSONRecordTooLong", err)
	}
	if len(writer.data) > limit || cap(writer.data) > limit {
		t.Fatalf(
			"bounded writer len/cap = %d/%d, want both <= %d",
			len(writer.data),
			cap(writer.data),
			limit,
		)
	}
	if _, err := encodeLine(value, limit); err == nil ||
		err.Error() != "JSON response exceeds 64-byte limit" {
		t.Fatalf("encodeLine() error = %v, want deterministic size error", err)
	}
}

func TestBoundedJSONEncodingRejectsUnsupportedShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{name: "byte slice", value: []byte("not a protocol shape")},
		{name: "custom marshaler", value: testJSONMarshaler("value")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := encodeLine(test.value, 1024); err == nil {
				t.Fatal("encodeLine() error = nil")
			}
		})
	}
}

type testJSONMarshaler string

func (testJSONMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}
