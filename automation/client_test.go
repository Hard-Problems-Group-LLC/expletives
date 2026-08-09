package automation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestClientHelloAllowsBoundedAdditiveFields(t *testing.T) {
	t.Parallel()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	limits := DefaultLimits()
	client := &Client{
		conn:   clientConn,
		reader: bufio.NewReaderSize(clientConn, lineReaderBufferBytes),
	}

	serverDone := make(chan error, 1)
	go func() {
		record := struct {
			Hello
			FutureCapability struct {
				Bounded bool `json:"bounded"`
			} `json:"future_capability"`
		}{
			Hello: validTestHello(limits),
		}
		record.SupportedVersions = []int{1, 2}
		record.FutureCapability.Bounded = true
		line, err := encodeLine(record, limits.ResponseLineBytes)
		if err == nil {
			err = writeAll(serverConn, line)
		}
		serverDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.readHello(ctx); err != nil {
		t.Fatalf("readHello() error = %v", err)
	}
	if got := client.Hello().SupportedVersions; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("SupportedVersions = %v, want [1 2]", got)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("write hello error = %v", err)
	}
}

func validTestHello(limits Limits) Hello {
	return Hello{
		Header:              newHeader(TypeHello),
		Application:         "automation-test",
		SessionID:           "session-1",
		Unauthenticated:     true,
		SupportedVersions:   []int{Version},
		Scenario:            "foundation.absolute-panels",
		LatestFrameSequence: 1,
		Operations:          append([]string{}, versionOneOperations[:]...),
		Commands:            []string{},
		Limits:              limits,
	}
}

func TestValidateHelloRequiresExactV1Capabilities(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	tests := []struct {
		name   string
		mutate func(*Hello)
	}{
		{
			name: "authenticated marker",
			mutate: func(hello *Hello) {
				hello.Unauthenticated = false
			},
		},
		{
			name: "empty scenario",
			mutate: func(hello *Hello) {
				hello.Scenario = ""
			},
		},
		{
			name: "invalid scenario",
			mutate: func(hello *Hello) {
				hello.Scenario = "invalid scenario"
			},
		},
		{
			name: "zero latest sequence",
			mutate: func(hello *Hello) {
				hello.LatestFrameSequence = 0
			},
		},
		{
			name: "missing operation",
			mutate: func(hello *Hello) {
				hello.Operations = hello.Operations[:len(hello.Operations)-1]
			},
		},
		{
			name: "unknown operation",
			mutate: func(hello *Hello) {
				hello.Operations[len(hello.Operations)-1] = "future_operation"
			},
		},
		{
			name: "duplicate operation",
			mutate: func(hello *Hello) {
				hello.Operations[len(hello.Operations)-1] = hello.Operations[0]
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			hello := validTestHello(limits)
			test.mutate(&hello)
			if err := validateHello(hello); err == nil {
				t.Fatal("validateHello() error = nil")
			}
		})
	}
}

func TestValidateHelloAcceptsRequiredOperationsInAnyOrder(t *testing.T) {
	t.Parallel()

	hello := validTestHello(DefaultLimits())
	for left, right := 0, len(hello.Operations)-1; left < right; left, right = left+1, right-1 {
		hello.Operations[left], hello.Operations[right] =
			hello.Operations[right], hello.Operations[left]
	}
	if err := validateHello(hello); err != nil {
		t.Fatalf("validateHello() error = %v", err)
	}
}

func TestValidateHelloAcceptsCommandInventoryAtControlBound(t *testing.T) {
	t.Parallel()

	hello := validTestHello(DefaultLimits())
	hello.Commands = make([]string, hello.Limits.Controls)
	for index := range hello.Commands {
		hello.Commands[index] = fmt.Sprintf("command-%04d", index)
	}
	if err := validateHello(hello); err != nil {
		t.Fatalf("validateHello() at command bound error = %v", err)
	}

	hello.Commands = append(hello.Commands, "command-overflow")
	if err := validateHello(hello); err == nil {
		t.Fatal("validateHello() above command bound error = nil")
	}
}

func TestDecodeHelloRequiresPresentNonNullTypedFields(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(validTestHello(DefaultLimits()))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var validFields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &validFields); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	for _, field := range requiredHelloFields {
		field := field
		t.Run(field+"/missing", func(t *testing.T) {
			t.Parallel()

			fields := cloneRawFields(validFields)
			delete(fields, field)
			line, marshalErr := json.Marshal(fields)
			if marshalErr != nil {
				t.Fatalf("json.Marshal() error = %v", marshalErr)
			}
			if _, decodeErr := decodeHelloRecord(line); decodeErr == nil {
				t.Fatal("decodeHelloRecord() error = nil")
			}
		})
		t.Run(field+"/null", func(t *testing.T) {
			t.Parallel()

			fields := cloneRawFields(validFields)
			fields[field] = json.RawMessage("null")
			line, marshalErr := json.Marshal(fields)
			if marshalErr != nil {
				t.Fatalf("json.Marshal() error = %v", marshalErr)
			}
			if _, decodeErr := decodeHelloRecord(line); decodeErr == nil {
				t.Fatal("decodeHelloRecord() error = nil")
			}
		})
	}

	for _, test := range []struct {
		name  string
		field string
		value string
	}{
		{name: "final string", field: "final", value: `"false"`},
		{name: "commands object", field: "commands", value: `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fields := cloneRawFields(validFields)
			fields[test.field] = json.RawMessage(test.value)
			line, marshalErr := json.Marshal(fields)
			if marshalErr != nil {
				t.Fatalf("json.Marshal() error = %v", marshalErr)
			}
			if _, decodeErr := decodeHelloRecord(line); decodeErr == nil {
				t.Fatal("decodeHelloRecord() error = nil")
			}
		})
	}
}

func cloneRawFields(
	fields map[string]json.RawMessage,
) map[string]json.RawMessage {
	cloned := make(map[string]json.RawMessage, len(fields))
	for name, value := range fields {
		cloned[name] = append(json.RawMessage{}, value...)
	}
	return cloned
}

func TestClientOperationalRecordsRemainStrict(t *testing.T) {
	t.Parallel()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	limits := DefaultLimits()
	client := &Client{
		conn:   clientConn,
		reader: bufio.NewReaderSize(clientConn, lineReaderBufferBytes),
		hello:  Hello{Limits: limits},
	}

	go func() {
		_, _ = serverConn.Write([]byte(
			`{"protocol":"expletives.automation","version":1,"type":"accepted",` +
				`"request_id":"request-1","operation":"observe","future":true}` + "\n",
		))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := client.readAccepted(ctx); err == nil {
		t.Fatal("readAccepted() with unknown field error = nil")
	}
}

func TestClientCloseInterruptsInFlightRequest(t *testing.T) {
	t.Parallel()

	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	limits := DefaultLimits()
	client := &Client{
		conn:   clientConn,
		reader: bufio.NewReaderSize(clientConn, lineReaderBufferBytes),
		hello:  Hello{Limits: limits},
	}

	acceptedSent := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		reader := bufio.NewReaderSize(serverConn, limits.RequestLineBytes+1)
		if _, err := readLine(serverConn, reader, limits.RequestLineBytes); err != nil {
			return
		}
		accepted := Accepted{
			Header:    newHeader(TypeAccepted),
			RequestID: "request-1",
			Operation: TypeObserve,
		}
		line, err := encodeLine(accepted, limits.ResponseLineBytes)
		if err != nil {
			return
		}
		if err := writeAll(serverConn, line); err != nil {
			return
		}
		close(acceptedSent)
		_, _ = serverConn.Read(make([]byte, 1))
	}()

	result := make(chan error, 1)
	go func() {
		_, err := client.Observe(context.Background(), "request-1", nil)
		result <- err
	}()

	select {
	case <-acceptedSent:
	case <-time.After(5 * time.Second):
		t.Fatal("fake server did not send acceptance")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Client.Close() error = %v", err)
	}
	select {
	case err := <-result:
		var indeterminate *IndeterminateError
		if !errors.As(err, &indeterminate) {
			t.Fatalf("Observe() error = %v, want IndeterminateError", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Client.Close() did not interrupt in-flight Observe()")
	}
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("fake server did not observe connection close")
	}
}

func TestClientPostWriteAcceptedFailuresAreIndeterminate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response func(Limits) ([]byte, error)
	}{
		{
			name: "mismatched request ID",
			response: func(limits Limits) ([]byte, error) {
				return encodeLine(Accepted{
					Header:    newHeader(TypeAccepted),
					RequestID: "another-request",
					Operation: TypeObserve,
				}, limits.ResponseLineBytes)
			},
		},
		{
			name: "mismatched operation",
			response: func(limits Limits) ([]byte, error) {
				return encodeLine(Accepted{
					Header:    newHeader(TypeAccepted),
					RequestID: "request-1",
					Operation: TypeWaitSnapshot,
				}, limits.ResponseLineBytes)
			},
		},
		{
			name: "malformed accepted",
			response: func(Limits) ([]byte, error) {
				return []byte(
					`{"protocol":"expletives.automation","version":1,` +
						`"type":"accepted","request_id":1,` +
						`"operation":"observe"}` + "\n",
				), nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			clientConn, serverConn := net.Pipe()
			t.Cleanup(func() {
				_ = clientConn.Close()
				_ = serverConn.Close()
			})
			limits := DefaultLimits()
			client := &Client{
				conn:   clientConn,
				reader: bufio.NewReaderSize(clientConn, lineReaderBufferBytes),
				hello:  Hello{Limits: limits},
			}

			serverDone := make(chan error, 1)
			go func() {
				reader := bufio.NewReaderSize(
					serverConn,
					limits.RequestLineBytes+1,
				)
				if _, err := readLine(
					serverConn,
					reader,
					limits.RequestLineBytes,
				); err != nil {
					serverDone <- err
					return
				}
				line, err := test.response(limits)
				if err == nil {
					err = writeAll(serverConn, line)
				}
				serverDone <- err
			}()

			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()
			_, err := client.Observe(ctx, "request-1", nil)
			var indeterminate *IndeterminateError
			if !errors.As(err, &indeterminate) {
				t.Fatalf(
					"Observe() error = %v, want IndeterminateError",
					err,
				)
			}
			if indeterminate.RequestID != "request-1" {
				t.Fatalf(
					"IndeterminateError.RequestID = %q, want request-1",
					indeterminate.RequestID,
				)
			}
			if indeterminate.Cause == nil {
				t.Fatal("IndeterminateError.Cause = nil")
			}

			client.stateMu.Lock()
			closed := client.closed
			client.stateMu.Unlock()
			if !closed {
				t.Fatal("Client remained open after indeterminate acceptance")
			}
			if err := <-serverDone; err != nil {
				t.Fatalf("fake server error = %v", err)
			}
		})
	}
}
