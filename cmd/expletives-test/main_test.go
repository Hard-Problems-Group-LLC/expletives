//go:build linux

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hard-Problems-Group-LLC/expletives/automation"
	"github.com/Hard-Problems-Group-LLC/expletives/internal/demo"
)

func TestHeadlessAutomationShutdownDeliversFinalCompletion(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "automation.sock")
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = input.Close() })
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = output.Close() })
	stderr, err := os.CreateTemp(t.TempDir(), "stderr-*.log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close() })

	status := make(chan int, 1)
	go func() {
		status <- run(
			[]string{
				"--headless",
				"--automation", socketPath,
				"--width", "68",
				"--height", "22",
				"--root-max-width", "64",
				"--root-max-height", "20",
			},
			input,
			output,
			stderr,
		)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var client *automation.Client
	for client == nil {
		if err := ctx.Err(); err != nil {
			_, _ = stderr.Seek(0, io.SeekStart)
			diagnostic, _ := io.ReadAll(stderr)
			t.Fatalf(
				"automation endpoint did not become ready: %v; stderr=%q",
				err,
				diagnostic,
			)
		}
		client, err = automation.Dial(ctx, socketPath)
		if err != nil {
			time.Sleep(time.Millisecond)
		}
	}
	t.Cleanup(func() { _ = client.Close() })

	observe, err := client.Observe(ctx, "layout-observe", nil)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if observe.Snapshot == nil ||
		observe.Snapshot.Scenario != demo.ScenarioID ||
		len(observe.Snapshot.Layouts) != 10 ||
		len(observe.Snapshot.Overflows) != 0 {
		if observe.Snapshot == nil {
			t.Fatal("initial observation has no snapshot")
		}
		t.Fatalf(
			"initial snapshot scenario=%q controls=%d Layouts=%d overflows=%d",
			observe.Snapshot.Scenario,
			len(observe.Snapshot.Controls),
			len(observe.Snapshot.Layouts),
			len(observe.Snapshot.Overflows),
		)
	}
	var backID, frontID automation.ControlID
	var accentBounds, frontBounds, rootBounds automation.Rect
	displayEvidence := make(map[string]bool)
	actionEvidence := make(map[string]bool)
	for _, control := range observe.Snapshot.Controls {
		switch control.Key {
		case "root":
			rootBounds = control.Bounds
		case "layer.back":
			backID = control.ID
		case "layer.front":
			frontID = control.ID
			frontBounds = control.AbsoluteBounds
		case "panel.accent":
			accentBounds = control.Bounds
		case "display.label":
			displayEvidence[control.Key] =
				control.Kind == "label" &&
					control.Details.Text != nil &&
					control.Details.Text.Target != "" &&
					control.Details.Text.Mnemonic == "a"
		case "display.static_text":
			displayEvidence[control.Key] =
				control.Kind == "static_text" &&
					control.Details.Text != nil &&
					control.Details.Text.Wrap == "words"
		case "display.separator":
			displayEvidence[control.Key] =
				control.Kind == "separator" &&
					control.Details.Divider != nil &&
					control.Details.Divider.Form == "double"
		case "display.rule":
			displayEvidence[control.Key] =
				control.Kind == "rule" &&
					control.Details.Divider != nil &&
					control.Details.Divider.Text == "Rule"
		case "action.toggle":
			actionEvidence[control.Key] =
				control.Kind == "button" &&
					control.Focused &&
					control.Details.Action != nil &&
					control.Details.Action.Command ==
						string(demo.CommandFixtureToggle) &&
					control.Details.Action.Default
		case "action.reset":
			actionEvidence[control.Key] =
				control.Kind == "button" &&
					control.Details.Action != nil &&
					control.Details.Action.Enabled
		case "action.disabled":
			actionEvidence[control.Key] =
				control.Kind == "button" &&
					control.Details.Action != nil &&
					!control.Details.Action.Enabled &&
					control.Details.Action.DisabledReason != ""
		case "action.quit":
			actionEvidence[control.Key] =
				control.Kind == "button" &&
					control.Details.Action != nil &&
					control.Details.Action.Cancel
		case "action.hotkeys":
			actionEvidence[control.Key] =
				control.Kind == "hotkey_bar" &&
					control.Details.HotkeyBar != nil &&
					len(control.Details.HotkeyBar.Items) == 4
		}
	}
	if len(observe.Snapshot.Controls) != 19 ||
		len(displayEvidence) != 4 ||
		len(actionEvidence) != 5 {
		t.Fatalf(
			"catalog evidence: display=%#v action=%#v",
			displayEvidence,
			actionEvidence,
		)
	}
	for key, valid := range displayEvidence {
		if !valid {
			t.Fatalf("display control %q has invalid typed evidence", key)
		}
	}
	for key, valid := range actionEvidence {
		if !valid {
			t.Fatalf("Action control %q has invalid typed evidence", key)
		}
	}
	if rootBounds != (automation.Rect{
		X: 2, Y: 1, Width: 64, Height: 20,
	}) {
		t.Fatalf("centered constrained root bounds = %+v", rootBounds)
	}
	if got, want := len(observe.Snapshot.Frame.Cells),
		observe.Snapshot.Frame.Size.Width*observe.Snapshot.Frame.Size.Height; got != want {
		t.Fatalf("expanded frame cells = %d, want %d", got, want)
	}
	if len(observe.Snapshot.Frame.Runs) == 0 {
		t.Fatal("client observation omitted compact frame runs")
	}
	layerX, layerY := frontBounds.X+2, frontBounds.Y+1
	layerCell := layerY*observe.Snapshot.Frame.Size.Width + layerX
	if backID == "" || frontID == "" ||
		layerX < 0 || layerY < 0 ||
		layerCell >= len(observe.Snapshot.Frame.Cells) ||
		observe.Snapshot.Frame.Cells[layerCell].Owner != frontID {
		t.Fatalf("initial stacking evidence is incomplete")
	}
	raised, err := client.InvokeCommand(
		ctx,
		"layout-raise",
		string(demo.CommandLayerRaise),
		"",
	)
	if err != nil {
		t.Fatalf("InvokeCommand(Layout Raise) error = %v", err)
	}
	if raised.Outcome != automation.OutcomeApplied ||
		raised.Snapshot == nil ||
		raised.Snapshot.Frame.Cells[layerCell].Owner != backID {
		t.Fatalf(
			"Layout Raise outcome=%q snapshot=%t expected_owner=%q",
			raised.Outcome,
			raised.Snapshot != nil,
			backID,
		)
	}
	lowered, err := client.InvokeCommand(
		ctx,
		"panel-lower",
		string(demo.CommandPanelLower),
		"",
	)
	if err != nil {
		t.Fatalf("InvokeCommand(Panel Lower) error = %v", err)
	}
	if lowered.Snapshot == nil {
		t.Fatalf(
			"Panel Lower outcome=%q has no snapshot",
			lowered.Outcome,
		)
	}
	var loweredAccent *automation.ControlSnapshot
	for index := range lowered.Snapshot.Controls {
		if lowered.Snapshot.Controls[index].Key == "panel.accent" {
			loweredAccent = &lowered.Snapshot.Controls[index]
			break
		}
	}
	if lowered.Outcome != automation.OutcomeApplied ||
		loweredAccent == nil ||
		loweredAccent.Bounds != accentBounds ||
		loweredAccent.LayoutIndex != 1 ||
		loweredAccent.StackIndex != 0 {
		t.Fatalf(
			"Panel Lower outcome=%q accent=%+v want bounds=%+v indices=1/0",
			lowered.Outcome,
			loweredAccent,
			accentBounds,
		)
	}

	if _, err := client.InjectInput(
		ctx,
		"action-alt-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt down) error = %v", err)
	}
	toggled, err := client.InjectInput(
		ctx,
		"action-toggle",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "t"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Alt-T) error = %v", err)
	}
	if toggled.Outcome != automation.OutcomeApplied ||
		toggled.Snapshot == nil ||
		toggled.Snapshot.Completion == nil ||
		toggled.Snapshot.Completion.Command !=
			string(demo.CommandFixtureToggle) {
		t.Fatalf("Alt-T completion = %+v", toggled)
	}
	var checked bool
	for _, control := range toggled.Snapshot.Controls {
		if control.Key == "action.toggle" &&
			control.Details.Action != nil {
			checked = control.Details.Action.Checked
		}
	}
	if !checked {
		t.Fatal("Alt-T did not publish checked command state")
	}
	if _, err := client.InjectInput(
		ctx,
		"action-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt up) error = %v", err)
	}

	requestID, err := automation.NewRequestID()
	if err != nil {
		t.Fatal(err)
	}
	completion, err := client.Shutdown(ctx, requestID)
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if completion.Outcome != automation.OutcomeExited ||
		completion.Snapshot == nil ||
		!completion.Snapshot.Final {
		t.Fatalf(
			"shutdown outcome=%q snapshot=%t final=%t",
			completion.Outcome,
			completion.Snapshot != nil,
			completion.Snapshot != nil && completion.Snapshot.Final,
		)
	}

	select {
	case got := <-status:
		if got != exitOK {
			t.Fatalf("expletives-test status = %d", got)
		}
	case <-ctx.Done():
		t.Fatalf("expletives-test did not exit: %v", ctx.Err())
	}
}
