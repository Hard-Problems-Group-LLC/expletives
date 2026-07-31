package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Hard-Problems-Group-LLC/expletives/automation"
)

func TestDefaultDialAndOperationTimeoutsAreSeparate(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	configuration, command, status := parseOptions([]string{"hello"}, &stderr)
	if status != exitOK {
		t.Fatalf("parseOptions() status = %d, stderr = %q", status, stderr.String())
	}
	if command.name != "hello" {
		t.Fatalf("command = %q, want hello", command.name)
	}
	if configuration.dialTimeout != 5*time.Second {
		t.Errorf("dial timeout = %s, want 5s", configuration.dialTimeout)
	}
	if configuration.timeout < 35*time.Second {
		t.Errorf("operation timeout = %s, want at least 35s", configuration.timeout)
	}
}

func TestIndeterminateErrorPrintsExactRecoveryCommand(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	status := reportError(
		&stderr,
		"/tmp/automation socket.sock",
		&automation.IndeterminateError{
			RequestID: "request-1",
			Cause:     errors.New("connection lost"),
		},
	)
	if status != exitIndeterminate {
		t.Fatalf("reportError() status = %d, want %d", status, exitIndeterminate)
	}
	output := stderr.String()
	for _, want := range []string{
		"request ID 'request-1'",
		"query_result recovery",
		"expletivesctl --socket '/tmp/automation socket.sock' result 'request-1'",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("stderr = %q, want substring %q", output, want)
		}
	}
}

func TestVersionAndHelpDoNotRequireSocket(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "version", args: []string{"--version"}, want: "expletivesctl "},
		{name: "help", args: []string{"--help"}, want: "Usage:"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if status := run(test.args, &stdout, &stderr); status != exitOK {
				t.Fatalf("run() status = %d, stderr = %q", status, stderr.String())
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, test.want) {
				t.Fatalf("output = %q, want substring %q", output, test.want)
			}
		})
	}
}

func TestSocketIsExplicit(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if status := run([]string{"hello"}, &stdout, &stderr); status != exitUsage {
		t.Fatalf("run() status = %d, want %d", status, exitUsage)
	}
	if !strings.Contains(stderr.String(), "--socket is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestParseRawKeyEvents(t *testing.T) {
	t.Parallel()

	event, err := parseKeyArgs([]string{"press", "r"})
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != automation.KeyPress || event.Key != "r" {
		t.Fatalf("event = %#v", event)
	}

	events, err := parseKeyTokens([]string{
		"down:control",
		"press:r",
		"up:control",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 ||
		events[0].Kind != automation.KeyDown ||
		events[1].Kind != automation.KeyPress ||
		events[2].Kind != automation.KeyUp {
		t.Fatalf("events = %#v", events)
	}
}

func TestParseRawKeyEventFailures(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		nil,
		{"sideways", "r"},
		{"press"},
		{"press", ""},
	} {
		if _, err := parseKeyArgs(args); err == nil {
			t.Fatalf("parseKeyArgs(%q) unexpectedly succeeded", args)
		}
	}
	for _, args := range [][]string{
		nil,
		{"press"},
		{"sideways:r"},
		{"press:"},
	} {
		if _, err := parseKeyTokens(args); err == nil {
			t.Fatalf("parseKeyTokens(%q) unexpectedly succeeded", args)
		}
	}
}

func TestConfiguredRequestIDsForSequence(t *testing.T) {
	t.Parallel()

	if got, err := indexedRequestID("batch", 1, 3); err != nil || got != "batch-2" {
		t.Fatalf("indexedRequestID() = %q, %v", got, err)
	}
	if got, err := indexedRequestID("only", 0, 1); err != nil || got != "only" {
		t.Fatalf("indexedRequestID() = %q, %v", got, err)
	}
}
