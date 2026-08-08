//go:build linux

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
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

	// This one process test intentionally audits the complete catalog over
	// many correlated snapshot-bearing round trips. Race instrumentation can
	// make the aggregate run substantially slower even though each request
	// remains within the protocol's independent deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	inputEvidence := make(map[string]bool)
	progressEvidence := make(map[string]bool)
	navigationEvidence := make(map[string]bool)
	contentEvidence := make(map[string]bool)
	screenEvidence := make(map[string]bool)
	chromeEvidence := make(map[string]bool)
	menuEvidence := false
	statusEvidence := false
	automationPanel := false
	var statusID automation.ControlID
	helpEnd := false
	expectedRootMnemonics := map[string]automation.Key{
		"menu.file":     "f",
		"menu.panels":   "n",
		"menu.layouts":  "a",
		"menu.controls": "c",
		"menu.sections": "s",
		"menu.menus":    "m",
		"menu.dialogs":  "d",
		"menu.help":     "p",
	}
	rootMnemonicEvidence := make(map[string]bool, len(expectedRootMnemonics))
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
		case "screen.home":
			screenEvidence[control.Key] =
				control.Visible &&
					len(control.Children) == 0 &&
					control.Style == "fixture.canvas" &&
					control.ResolvedStyle.Background == "#003878"
		case "screen.panels.core", "screen.panels.styles",
			"screen.layouts.box", "screen.layouts.grid",
			"screen.text", "screen.actions", "screen.selection", "screen.input",
			"screen.progress", "screen.navigation", "screen.scrolling",
			"screen.menus", "screen.status", "screen.headers_footers",
			"screen.about":
			screenEvidence[control.Key] = !control.Visible
		case "header.primary", "header.secondary":
			chromeEvidence[control.Key] =
				control.Kind == "header" &&
					control.Details.Container != nil &&
					!control.Visible &&
					control.Bounds == (automation.Rect{})
		case "footer.hotkeys.global", "footer.hotkeys.screen",
			"footer.guidance.focus":
			chromeEvidence[control.Key] =
				control.Kind == "footer" &&
					control.Details.Container != nil &&
					control.Visible &&
					control.Bounds.Height == 1
		case "menu.main":
			menuEvidence =
				control.Kind == "menu_bar" &&
					control.Details.MenuBar != nil &&
					len(control.Details.MenuBar.Entries) == 62 &&
					len(control.Details.MenuBar.OpenPath) == 0 &&
					control.Bounds == (automation.Rect{
						Width:  observe.Snapshot.Frame.Size.Width,
						Height: 1,
					}) &&
					control.AbsoluteBounds == (automation.Rect{
						Width:  observe.Snapshot.Frame.Size.Width,
						Height: 1,
					})
			if control.Details.MenuBar != nil {
				for _, entry := range control.Details.MenuBar.Entries {
					if entry.Key == "menu.help" {
						helpEnd = entry.Placement == "end"
					}
					if mnemonic, ok := expectedRootMnemonics[entry.Key]; ok {
						rootMnemonicEvidence[entry.Key] =
							entry.Mnemonic == mnemonic
					}
				}
			}
		case "status.main":
			statusID = control.ID
			statusEvidence =
				control.Kind == "status_bar" &&
					control.Details.StatusBar != nil &&
					len(control.Details.StatusBar.Segments) == 2 &&
					control.Details.StatusBar.Segments[0].Label ==
						"UNAUTHENTICATED AUTOMATION ENABLED" &&
					control.Details.StatusBar.Segments[1].Label == "Home" &&
					control.Bounds == (automation.Rect{
						Y:      observe.Snapshot.Frame.Size.Height - 1,
						Width:  observe.Snapshot.Frame.Size.Width,
						Height: 1,
					})
		case "automation.status":
			automationPanel = true
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
					!control.Focused &&
					control.Details.Action != nil &&
					control.Details.Action.Command ==
						string(demo.CommandFixtureToggle) &&
					control.Details.Action.Default &&
					control.Details.Action.Mnemonic == "g"
		case "action.reset":
			actionEvidence[control.Key] =
				control.Kind == "button" &&
					control.Details.Action != nil &&
					control.Details.Action.Enabled &&
					control.Details.Action.Mnemonic == "r"
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
					len(control.Details.HotkeyBar.Items) == 2
		case "input.text.plain":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Text == "Edit me" &&
					control.Details.TextField.MaximumBytes == 10 &&
					control.Details.TextField.Validator == nil &&
					!control.Details.TextField.Password
		case "input.text.soft_whitelist":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Validator != nil &&
					control.Details.TextField.Validator.Enforcement == "soft" &&
					control.Details.TextField.Validator.Mode == "whitelist"
		case "input.text.hard_whitelist":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Validator != nil &&
					control.Details.TextField.Validator.Enforcement == "hard" &&
					control.Details.TextField.Validator.Mode == "whitelist"
		case "input.text.soft_blacklist":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Validator != nil &&
					control.Details.TextField.Validator.Enforcement == "soft" &&
					control.Details.TextField.Validator.Mode == "blacklist"
		case "input.text.hard_blacklist":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Validator != nil &&
					control.Details.TextField.Validator.Enforcement == "hard" &&
					control.Details.TextField.Validator.Mode == "blacklist"
		case "input.text.password":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Password &&
					control.Details.TextField.Redacted &&
					control.Details.TextField.Text == "" &&
					control.Details.TextField.Length == len("secret")
		case "input.number.ranged":
			inputEvidence[control.Key] =
				control.Kind == "number_field" &&
					control.Details.NumberField != nil &&
					control.Details.NumberField.Value == 12.5 &&
					control.Details.NumberField.Minimum != nil &&
					*control.Details.NumberField.Minimum == 0 &&
					control.Details.NumberField.Maximum != nil &&
					*control.Details.NumberField.Maximum == 20 &&
					control.Details.NumberField.Step == 0
		case "input.spin.clamped":
			inputEvidence[control.Key] =
				control.Kind == "spin_box" &&
					control.Details.NumberField != nil &&
					control.Details.NumberField.Value == 1 &&
					control.Details.NumberField.Step == 0.5
		case "input.text_area.multiline":
			inputEvidence[control.Key] =
				control.Kind == "text_area" &&
					control.Details.TextArea != nil &&
					control.Details.TextArea.Text == "Multiline\ntext area" &&
					control.Details.TextArea.LineCount == 2 &&
					control.Details.TextArea.Wrap == "words"
		case "progress.bar.determinate":
			progressEvidence[control.Key] =
				control.Kind == "progress_bar" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "running" &&
					control.Details.Progress.Current == 42 &&
					control.Details.Progress.Total == 100 &&
					!control.Details.Progress.Indeterminate &&
					control.Details.Progress.TextMode == "percentage"
		case "progress.bar.indeterminate":
			progressEvidence[control.Key] =
				control.Kind == "progress_bar" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "running" &&
					control.Details.Progress.Indeterminate &&
					control.Details.Progress.Tick == 0 &&
					control.Details.Progress.FrameIndex == 0
		case "progress.meter.horizontal":
			progressEvidence[control.Key] =
				control.Kind == "meter" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Value == 65 &&
					control.Details.Progress.Minimum == 0 &&
					control.Details.Progress.Maximum == 100 &&
					control.Details.Progress.Orientation ==
						automation.Orientation(expletives.Horizontal)
		case "progress.meter.vertical":
			progressEvidence[control.Key] =
				control.Kind == "meter" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Value == 65 &&
					control.Details.Progress.Orientation ==
						automation.Orientation(expletives.Vertical)
		case "progress.spinner":
			progressEvidence[control.Key] =
				control.Kind == "spinner" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "running" &&
					control.Details.Progress.FrameIndex == 0
		case "progress.activity_dots":
			progressEvidence[control.Key] =
				control.Kind == "activity_dots" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "running" &&
					control.Details.Progress.FrameIndex == 0
		case "progress.bar.completed":
			progressEvidence[control.Key] =
				control.Kind == "progress_bar" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "completed"
		case "progress.bar.failed":
			progressEvidence[control.Key] =
				control.Kind == "progress_bar" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "failed"
		case "progress.bar.cancelled":
			progressEvidence[control.Key] =
				control.Kind == "progress_bar" &&
					control.Details.Progress != nil &&
					control.Details.Progress.Status == "cancelled"
		case "progress.spinner.reduced":
			progressEvidence[control.Key] =
				control.Kind == "spinner" &&
					control.Details.Progress != nil &&
					control.Details.Progress.ReducedMotion &&
					control.Details.Progress.Tick == 0 &&
					control.Details.Progress.FrameIndex == 0
		case "progress.activity_dots.reduced":
			progressEvidence[control.Key] =
				control.Kind == "activity_dots" &&
					control.Details.Progress != nil &&
					control.Details.Progress.ReducedMotion &&
					control.Details.Progress.Tick == 0 &&
					control.Details.Progress.FrameIndex == 0
		case "navigation.scrollbar.horizontal":
			navigationEvidence[control.Key] =
				control.Kind == "scroll_bar" &&
					control.Details.ScrollBar != nil &&
					control.Details.ScrollBar.Orientation ==
						automation.Orientation(expletives.Horizontal) &&
					control.Details.ScrollBar.ContentSize == 100 &&
					control.Details.ScrollBar.ViewportSize == 20 &&
					control.Details.ScrollBar.Offset == 40 &&
					control.Details.ScrollBar.MaximumOffset == 80
		case "navigation.scrollbar.vertical":
			navigationEvidence[control.Key] =
				control.Kind == "scroll_bar" &&
					control.Details.ScrollBar != nil &&
					control.Details.ScrollBar.Orientation ==
						automation.Orientation(expletives.Vertical) &&
					control.Details.ScrollBar.ContentSize == 80 &&
					control.Details.ScrollBar.ViewportSize == 16 &&
					control.Details.ScrollBar.Offset == 32 &&
					control.Details.ScrollBar.MaximumOffset == 64
		case "navigation.tabs":
			navigationEvidence[control.Key] =
				control.Kind == "tabbed_panel" &&
					control.Details.TabbedPanel != nil &&
					control.Details.TabbedPanel.Selected == "overview" &&
					control.Details.TabbedPanel.Current == "overview" &&
					len(control.Details.TabbedPanel.Tabs) == 3
		case "navigation.notebook":
			navigationEvidence[control.Key] =
				control.Kind == "notebook" &&
					control.Details.TabbedPanel != nil &&
					control.Details.TabbedPanel.Selected == "one" &&
					control.Details.TabbedPanel.Current == "one" &&
					len(control.Details.TabbedPanel.Tabs) == 2
		case "content.markdown":
			contentEvidence[control.Key] =
				control.Kind == "markdown_view" &&
					control.Details.Markdown != nil &&
					control.Details.Markdown.SourceBytes > 0 &&
					control.Details.Markdown.BlockCount >= 8
		case "content.log-follow":
			contentEvidence[control.Key] =
				control.Kind == "log_view" &&
					control.Details.LogView != nil &&
					control.Details.LogView.RetainedRecords == 5 &&
					control.Details.LogView.FirstKey == "startup" &&
					control.Details.LogView.LastKey == "ready" &&
					control.Details.LogView.Follow
		case "content.log-scrollback":
			contentEvidence[control.Key] =
				control.Kind == "log_view" &&
					control.Details.LogView != nil &&
					control.Details.LogView.RetainedRecords == 5 &&
					!control.Details.LogView.Follow
		case "content.stream-drops":
			contentEvidence[control.Key] =
				control.Kind == "stream_view" &&
					control.Details.StreamView != nil &&
					control.Details.StreamView.RetainedLines == 1 &&
					control.Details.StreamView.PendingBytes > 0 &&
					control.Details.StreamView.DroppedLines > 0 &&
					control.Details.StreamView.DroppedBytes > 0 &&
					control.Details.StreamView.Follow
		}
	}
	if len(observe.Snapshot.Frame.Cells) > 2 {
		menuEvidence = menuEvidence &&
			observe.Snapshot.Frame.Cells[0].Style == "menu_bar" &&
			observe.Snapshot.Frame.Cells[2].Grapheme == "F" &&
			observe.Snapshot.Frame.Cells[2].Style == "menu.mnemonic" &&
			observe.Snapshot.Frame.Cells[observe.Snapshot.Frame.Size.Width-2].Grapheme == "p" &&
			observe.Snapshot.Frame.Cells[observe.Snapshot.Frame.Size.Width-2].Style == "menu.mnemonic"
		lastRow := (observe.Snapshot.Frame.Size.Height - 1) *
			observe.Snapshot.Frame.Size.Width
		statusEvidence = statusEvidence &&
			observe.Snapshot.Frame.Cells[lastRow].Owner == statusID &&
			observe.Snapshot.Frame.Cells[lastRow+observe.Snapshot.Frame.Size.Width-1].Owner == statusID
	}
	if !menuEvidence || !statusEvidence || automationPanel || !helpEnd ||
		len(rootMnemonicEvidence) != len(expectedRootMnemonics) ||
		len(displayEvidence) != 4 ||
		len(actionEvidence) != 5 ||
		len(inputEvidence) != 9 ||
		len(progressEvidence) != 11 ||
		len(navigationEvidence) != 4 ||
		len(contentEvidence) != 4 ||
		len(screenEvidence) != 16 ||
		len(chromeEvidence) != 5 {
		t.Fatalf(
			"catalog evidence: menu=%t status=%t screens=%#v chrome=%#v display=%#v action=%#v input=%#v progress=%#v navigation=%#v content=%#v",
			menuEvidence,
			statusEvidence,
			screenEvidence,
			chromeEvidence,
			displayEvidence,
			actionEvidence,
			inputEvidence,
			progressEvidence,
			navigationEvidence,
			contentEvidence,
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
	for key, valid := range inputEvidence {
		if !valid {
			t.Fatalf("TextField %q has invalid typed evidence", key)
		}
	}
	for key, valid := range progressEvidence {
		if !valid {
			t.Fatalf("Progress control %q has invalid typed evidence", key)
		}
	}
	for key, valid := range navigationEvidence {
		if !valid {
			t.Fatalf("Navigation control %q has invalid typed evidence", key)
		}
	}
	for key, valid := range contentEvidence {
		if !valid {
			t.Fatalf("Content control %q has invalid typed evidence", key)
		}
	}
	for key, valid := range rootMnemonicEvidence {
		if !valid {
			t.Fatalf("Menu root %q has a colliding mnemonic", key)
		}
	}
	for key, valid := range screenEvidence {
		if !valid {
			t.Fatalf("screen %q has invalid initial visibility", key)
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
	toggleAutomationNotice := func(
		requestPrefix string,
		wantVisible bool,
	) {
		t.Helper()
		if _, err := client.InjectInput(
			ctx,
			requestPrefix+"-alt-down",
			automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
		); err != nil {
			t.Fatalf("InjectInput(Alt down) error = %v", err)
		}
		if _, err := client.InjectInput(
			ctx,
			requestPrefix+"-file",
			automation.KeyEvent{Kind: automation.KeyPress, Key: "f"},
		); err != nil {
			t.Fatalf("InjectInput(Alt-F) error = %v", err)
		}
		if _, err := client.InjectInput(
			ctx,
			requestPrefix+"-alt-up",
			automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
		); err != nil {
			t.Fatalf("InjectInput(Alt up) error = %v", err)
		}
		completion, err := client.InjectInput(
			ctx,
			requestPrefix+"-toggle",
			automation.KeyEvent{Kind: automation.KeyPress, Key: "a"},
		)
		if err != nil {
			t.Fatalf("InjectInput(File/Automation Notice) error = %v", err)
		}
		if completion.Outcome != automation.OutcomeApplied ||
			completion.Snapshot == nil ||
			completion.Snapshot.Completion == nil ||
			completion.Snapshot.Completion.Command !=
				string(demo.CommandAutomationNotice) {
			t.Fatalf("Automation Notice completion = %+v", completion)
		}
		noticeVisible := false
		menuChecked := false
		menuFound := false
		for _, control := range completion.Snapshot.Controls {
			switch control.Key {
			case "status.main":
				if control.Details.StatusBar == nil {
					t.Fatal("status.main has no StatusBar details")
				}
				for _, segment := range control.Details.StatusBar.Segments {
					if segment.Key == "automation" {
						noticeVisible =
							segment.Label ==
								"UNAUTHENTICATED AUTOMATION ENABLED"
					}
				}
			case "menu.main":
				if control.Details.MenuBar == nil {
					t.Fatal("menu.main has no MenuBar details")
				}
				for _, entry := range control.Details.MenuBar.Entries {
					if entry.Key == "menu.file.automation_notice" {
						menuFound = true
						menuChecked = entry.Checked
					}
				}
			case "automation.status":
				t.Fatal("dedicated automation status panel reappeared")
			}
		}
		if noticeVisible != wantVisible || !menuFound ||
			menuChecked != wantVisible {
			t.Fatalf(
				"Automation Notice visible=%t checked=%t found=%t, want=%t",
				noticeVisible,
				menuChecked,
				menuFound,
				wantVisible,
			)
		}
	}
	toggleAutomationNotice("automation-notice-hide", false)
	toggleAutomationNotice("automation-notice-show", true)

	shownBox, err := client.InvokeCommand(
		ctx,
		"show-box-layout",
		string(demo.CommandViewLayoutBox),
		"",
	)
	if err != nil {
		t.Fatalf("InvokeCommand(Box Layout) error = %v", err)
	}
	if shownBox.Outcome != automation.OutcomeApplied ||
		shownBox.Snapshot == nil {
		t.Fatalf("Box Layout completion = %+v", shownBox)
	}
	layerX, layerY := frontBounds.X+2, frontBounds.Y+1
	layerCell := layerY*shownBox.Snapshot.Frame.Size.Width + layerX
	if backID == "" || frontID == "" ||
		layerX < 0 || layerY < 0 ||
		layerCell >= len(shownBox.Snapshot.Frame.Cells) ||
		shownBox.Snapshot.Frame.Cells[layerCell].Owner != frontID {
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
		"menu-controls-alt-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt down) error = %v", err)
	}
	openedControls, err := client.InjectInput(
		ctx,
		"menu-controls-open",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "c"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Alt-C) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"menu-controls-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt up) error = %v", err)
	}
	var openedControlsMenu *automation.MenuBarDetails
	if openedControls.Snapshot != nil {
		for index := range openedControls.Snapshot.Controls {
			control := &openedControls.Snapshot.Controls[index]
			if control.Key == "menu.main" {
				openedControlsMenu = control.Details.MenuBar
				break
			}
		}
	}
	if openedControlsMenu == nil ||
		len(openedControlsMenu.OpenPath) != 1 ||
		openedControlsMenu.OpenPath[0] != "menu.controls" ||
		len(openedControlsMenu.SelectedPath) != 2 ||
		openedControlsMenu.SelectedPath[1] != "menu.controls.text" {
		t.Fatalf("Alt-C MenuBar state = %#v", openedControlsMenu)
	}
	shownActions, err := client.InjectInput(
		ctx,
		"menu-controls-actions",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "a"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Controls/A) error = %v", err)
	}
	if shownActions.Outcome != automation.OutcomeApplied ||
		shownActions.Snapshot == nil ||
		shownActions.Snapshot.Completion == nil ||
		shownActions.Snapshot.Completion.Command !=
			string(demo.CommandViewActions) {
		t.Fatalf("Controls/Actions completion = %+v", shownActions)
	}
	actionsVisible := false
	actionsFocused := false
	homeHidden := false
	viewChecked := false
	for _, control := range shownActions.Snapshot.Controls {
		switch control.Key {
		case "screen.actions":
			actionsVisible = control.Visible
		case "screen.home":
			homeHidden = !control.Visible
		case "action.toggle":
			actionsFocused = control.Focused
		case "menu.main":
			if control.Details.MenuBar != nil {
				for _, entry := range control.Details.MenuBar.Entries {
					if entry.Key == "menu.controls.actions" {
						viewChecked = entry.Checked
					}
				}
			}
		}
	}
	if !actionsVisible || !actionsFocused || !homeHidden || !viewChecked {
		t.Fatalf(
			"Actions screen visible=%t focused=%t homeHidden=%t checked=%t",
			actionsVisible,
			actionsFocused,
			homeHidden,
			viewChecked,
		)
	}

	selectionScreen, err := client.InvokeCommand(
		ctx,
		"show-selection",
		string(demo.CommandSelection),
		"",
	)
	if err != nil || selectionScreen.Outcome != automation.OutcomeApplied {
		t.Fatalf(
			"InvokeCommand(Selection) = %+v, %v",
			selectionScreen,
			err,
		)
	}
	if selectionScreen.Snapshot == nil ||
		len(selectionScreen.Snapshot.Overflows) == 0 {
		t.Fatal("Selection fixture did not expose its expected narrow-window overflow")
	}
	overflowDismissed, err := client.InjectInput(
		ctx,
		"selection-dismiss-overflow",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "enter"},
	)
	if err != nil || overflowDismissed.Snapshot == nil ||
		overflowDismissed.Snapshot.Completion == nil ||
		overflowDismissed.Snapshot.Completion.Command != "overflow.dismiss" {
		t.Fatalf(
			"Selection overflow dismissal = %+v, %v",
			overflowDismissed,
			err,
		)
	}
	for index := range 2 {
		if _, err := client.InjectInput(
			ctx,
			"selection-tab-"+string(rune('0'+index)),
			automation.KeyEvent{Kind: automation.KeyPress, Key: "tab"},
		); err != nil {
			t.Fatalf("InjectInput(Selection Tab %d) error = %v", index, err)
		}
	}
	choiceValue := func(
		completion automation.Completion,
		key string,
	) string {
		t.Helper()
		if completion.Snapshot == nil {
			t.Fatalf("Selection snapshot is absent for %q", key)
		}
		for _, control := range completion.Snapshot.Controls {
			if control.Key == key && control.Details.ChoiceField != nil {
				return control.Details.ChoiceField.Value
			}
		}
		t.Fatalf("Selection choice %q is absent", key)
		return ""
	}
	cycleNext, err := client.InjectInput(
		ctx,
		"selection-cycle-next",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "]"},
	)
	cycleNextValue := choiceValue(cycleNext, "selection.cycle.primary")
	if err != nil || cycleNext.Outcome != automation.OutcomeApplied ||
		cycleNextValue != "charlie" {
		t.Fatalf(
			"CycleField ] = value:%q completion:%+v, %v",
			cycleNextValue,
			cycleNext,
			err,
		)
	}
	cycleStop, err := client.InjectInput(
		ctx,
		"selection-cycle-stop",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "]"},
	)
	cycleStopValue := choiceValue(cycleStop, "selection.cycle.primary")
	if err != nil || cycleStop.Outcome != automation.OutcomeNoOp ||
		cycleStopValue != "charlie" {
		t.Fatalf(
			"CycleField ] at end = value:%q completion:%+v, %v",
			cycleStopValue,
			cycleStop,
			err,
		)
	}
	cycleWrap, err := client.InjectInput(
		ctx,
		"selection-cycle-space-wrap",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "space"},
	)
	cycleWrapValue := choiceValue(cycleWrap, "selection.cycle.primary")
	if err != nil || cycleWrap.Outcome != automation.OutcomeApplied ||
		cycleWrapValue != "alpha" {
		t.Fatalf(
			"CycleField Space wrap = value:%q completion:%+v, %v",
			cycleWrapValue,
			cycleWrap,
			err,
		)
	}
	cycleEnter, err := client.InjectInput(
		ctx,
		"selection-cycle-enter",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "enter"},
	)
	cycleEnterValue := choiceValue(cycleEnter, "selection.cycle.primary")
	if err != nil || cycleEnter.Outcome != automation.OutcomeApplied ||
		cycleEnterValue != "charlie" {
		command := ""
		focused := ""
		overflows := 0
		if cycleEnter.Snapshot != nil {
			overflows = len(cycleEnter.Snapshot.Overflows)
			if cycleEnter.Snapshot.Completion != nil {
				command = cycleEnter.Snapshot.Completion.Command
			}
			for _, control := range cycleEnter.Snapshot.Controls {
				if control.Focused {
					focused = control.Key
				}
			}
		}
		t.Fatalf(
			"CycleField Enter = value:%q command:%q focused:%q "+
				"overflows:%d completion:%+v, %v",
			cycleEnterValue,
			command,
			focused,
			overflows,
			cycleEnter,
			err,
		)
	}
	actionsRestored, err := client.InvokeCommand(
		ctx,
		"restore-actions",
		string(demo.CommandViewActions),
		"",
	)
	if err != nil || actionsRestored.Outcome != automation.OutcomeApplied {
		t.Fatalf("InvokeCommand(restore Actions) = %+v, %v", actionsRestored, err)
	}

	if _, err := client.InjectInput(
		ctx,
		"menu-layouts-alt-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt down) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"menu-layouts-open",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "a"},
	); err != nil {
		t.Fatalf("InjectInput(Alt-A) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"menu-layouts-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt up) error = %v", err)
	}
	nestedMenu, err := client.InjectInput(
		ctx,
		"menu-layouts-stacking",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "s"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Layouts/S) error = %v", err)
	}
	if nestedMenu.Snapshot == nil {
		t.Fatal("Layouts/Stacking completion has no snapshot")
	}
	var nestedOpen bool
	for _, control := range nestedMenu.Snapshot.Controls {
		if control.Key == "menu.main" &&
			control.Details.MenuBar != nil {
			path := control.Details.MenuBar.OpenPath
			nestedOpen = len(path) == 2 &&
				path[0] == "menu.layouts" &&
				path[1] == "menu.layouts.stacking"
		}
	}
	if !nestedOpen {
		t.Fatal("Layouts/Stacking submenu did not open")
	}
	for index := range 2 {
		if _, err := client.InjectInput(
			ctx,
			"menu-layouts-escape-"+string(rune('0'+index)),
			automation.KeyEvent{Kind: automation.KeyPress, Key: "escape"},
		); err != nil {
			t.Fatalf("InjectInput(Escape) error = %v", err)
		}
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
		automation.KeyEvent{Kind: automation.KeyPress, Key: "g"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Alt-G) error = %v", err)
	}
	if toggled.Outcome != automation.OutcomeApplied ||
		toggled.Snapshot == nil ||
		toggled.Snapshot.Completion == nil ||
		toggled.Snapshot.Completion.Command !=
			string(demo.CommandFixtureToggle) {
		t.Fatalf("Alt-G completion = %+v", toggled)
	}
	var checked bool
	for _, control := range toggled.Snapshot.Controls {
		if control.Key == "action.toggle" &&
			control.Details.Action != nil {
			checked = control.Details.Action.Checked
		}
	}
	if !checked {
		t.Fatal("Alt-G did not publish checked command state")
	}
	if _, err := client.InjectInput(
		ctx,
		"action-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt up) error = %v", err)
	}
	inputScreen, err := client.InvokeCommand(
		ctx,
		"show-input",
		string(demo.CommandTextInput),
		"",
	)
	if err != nil || inputScreen.Outcome != automation.OutcomeApplied {
		t.Fatalf("InvokeCommand(Text Input) = outcome %q, %v",
			inputScreen.Outcome, err)
	}
	for _, event := range []struct {
		request string
		key     string
	}{
		{"input-edit", "enter"},
		{"input-type", "z"},
	} {
		if _, err := client.InjectInput(
			ctx,
			event.request,
			automation.KeyEvent{Kind: automation.KeyPress, Key: event.key},
		); err != nil {
			t.Fatalf("InjectInput(%s) error = %v", event.key, err)
		}
	}
	textCommit, err := client.InjectInput(
		ctx,
		"input-commit",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "enter"},
	)
	if err != nil {
		t.Fatalf("InjectInput(TextField commit) error = %v", err)
	}
	if textCommit.Snapshot == nil ||
		textCommit.Snapshot.Completion == nil ||
		textCommit.Snapshot.Completion.Command !=
			string(demo.CommandTextSubmitted) {
		t.Fatalf("TextField commit outcome=%q snapshot=%t",
			textCommit.Outcome, textCommit.Snapshot != nil)
	}
	var areaFocus automation.Completion
	for index := range 8 {
		areaFocus, err = client.InjectInput(
			ctx,
			"area-tab-"+string(rune('0'+index)),
			automation.KeyEvent{Kind: automation.KeyPress, Key: "tab"},
		)
		if err != nil {
			t.Fatalf("InjectInput(TextArea Tab %d) error = %v", index, err)
		}
	}
	areaFocused := false
	if areaFocus.Snapshot != nil {
		for _, control := range areaFocus.Snapshot.Controls {
			if control.Key == "input.text_area.multiline" {
				areaFocused = control.Focused
			}
		}
	}
	if !areaFocused {
		t.Fatal("raw Tab group traversal did not enter TextArea")
	}
	for _, event := range []struct {
		request string
		key     string
	}{
		{"area-edit", "enter"},
		{"area-type-x", "x"},
		{"area-newline", "enter"},
		{"area-type-y", "y"},
	} {
		if _, err := client.InjectInput(
			ctx,
			event.request,
			automation.KeyEvent{Kind: automation.KeyPress, Key: event.key},
		); err != nil {
			t.Fatalf("InjectInput(TextArea %s) error = %v", event.key, err)
		}
	}
	if _, err := client.InjectInput(
		ctx,
		"area-control-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "control"},
	); err != nil {
		t.Fatalf("InjectInput(TextArea Ctrl down) error = %v", err)
	}
	areaCommit, err := client.InjectInput(
		ctx,
		"area-commit",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "enter"},
	)
	if err != nil {
		t.Fatalf("InjectInput(TextArea commit) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"area-control-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "control"},
	); err != nil {
		t.Fatalf("InjectInput(TextArea Ctrl up) error = %v", err)
	}
	areaValue := ""
	if areaCommit.Snapshot != nil {
		for _, control := range areaCommit.Snapshot.Controls {
			if control.Key == "input.text_area.multiline" &&
				control.Details.TextArea != nil {
				areaValue = control.Details.TextArea.Text
			}
		}
	}
	if areaCommit.Outcome != automation.OutcomeApplied ||
		areaCommit.Snapshot == nil ||
		areaCommit.Snapshot.Completion == nil ||
		areaCommit.Snapshot.Completion.Command !=
			string(demo.CommandTextChanged) ||
		!strings.HasSuffix(areaValue, "x\ny") {
		t.Fatalf(
			"TextArea completion outcome=%q command=%q value=%q",
			areaCommit.Outcome,
			func() string {
				if areaCommit.Snapshot == nil ||
					areaCommit.Snapshot.Completion == nil {
					return ""
				}
				return areaCommit.Snapshot.Completion.Command
			}(),
			areaValue,
		)
	}
	for _, event := range []automation.KeyEvent{
		{Kind: automation.KeyDown, Key: "shift"},
		{Kind: automation.KeyPress, Key: "tab"},
		{Kind: automation.KeyUp, Key: "shift"},
	} {
		if _, err := client.InjectInput(
			ctx,
			"input-spin-focus-"+string(event.Kind),
			event,
		); err != nil {
			t.Fatalf("InjectInput(Shift-Tab %s) error = %v", event.Kind, err)
		}
	}
	var spinCompletion automation.Completion
	spinCompletion, err = client.InjectInput(
		ctx,
		"input-spin-increment",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "]"},
	)
	if err != nil {
		t.Fatalf("InjectInput(SpinBox increment) error = %v", err)
	}
	spinValue := 0.0
	if spinCompletion.Snapshot != nil {
		for _, control := range spinCompletion.Snapshot.Controls {
			if control.Key == "input.spin.clamped" &&
				control.Details.NumberField != nil {
				spinValue = control.Details.NumberField.Value
			}
		}
	}
	if spinCompletion.Outcome != automation.OutcomeApplied ||
		spinCompletion.Snapshot == nil ||
		spinCompletion.Snapshot.Completion == nil ||
		spinCompletion.Snapshot.Completion.Command !=
			string(demo.CommandNumberChanged) ||
		spinValue != 1.5 {
		t.Fatalf(
			"SpinBox completion outcome=%q command=%q value=%v",
			spinCompletion.Outcome,
			func() string {
				if spinCompletion.Snapshot == nil ||
					spinCompletion.Snapshot.Completion == nil {
					return ""
				}
				return spinCompletion.Snapshot.Completion.Command
			}(),
			spinValue,
		)
	}
	progressScreen, err := client.InvokeCommand(
		ctx,
		"show-progress",
		string(demo.CommandProgress),
		"",
	)
	if err != nil || progressScreen.Outcome != automation.OutcomeApplied ||
		progressScreen.Snapshot == nil {
		t.Fatalf(
			"InvokeCommand(Progress) = outcome %q snapshot=%t, %v",
			progressScreen.Outcome,
			progressScreen.Snapshot != nil,
			err,
		)
	}
	progressControl := func(
		snapshot *automation.SnapshotV1,
		key string,
	) *automation.ControlSnapshot {
		t.Helper()
		if snapshot == nil {
			t.Fatalf("Progress snapshot is absent for %q", key)
		}
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == key {
				return &snapshot.Controls[index]
			}
		}
		t.Fatalf("Progress control %q is absent", key)
		return nil
	}
	if !progressControl(progressScreen.Snapshot, "screen.progress").Visible ||
		!progressControl(
			progressScreen.Snapshot,
			"progress.action.tick",
		).Focused {
		t.Fatal("Progress screen did not become visible and focused")
	}
	bar := progressControl(
		progressScreen.Snapshot,
		"progress.bar.determinate",
	).Details.Progress
	if bar == nil || bar.Current != 42 || bar.Total != 100 ||
		bar.Status != "running" {
		t.Fatalf("initial determinate Progress details = %#v", bar)
	}
	progressTick, err := client.InvokeCommand(
		ctx,
		"progress-tick",
		string(demo.CommandProgressTick),
		"",
	)
	if err != nil || progressTick.Outcome != automation.OutcomeApplied ||
		progressTick.Snapshot == nil {
		t.Fatalf("InvokeCommand(Progress Tick) = %+v, %v", progressTick, err)
	}
	bar = progressControl(
		progressTick.Snapshot,
		"progress.bar.determinate",
	).Details.Progress
	spinner := progressControl(
		progressTick.Snapshot,
		"progress.spinner",
	).Details.Progress
	dots := progressControl(
		progressTick.Snapshot,
		"progress.activity_dots",
	).Details.Progress
	if bar == nil || bar.Current != 49 ||
		spinner == nil || spinner.Tick != 1 || spinner.FrameIndex != 1 ||
		dots == nil || dots.Tick != 1 || dots.FrameIndex != 1 {
		t.Fatal("automation Progress Tick evidence is incomplete")
	}
	reducedMotion, err := client.InvokeCommand(
		ctx,
		"progress-motion",
		string(demo.CommandProgressMotion),
		"",
	)
	if err != nil || reducedMotion.Outcome != automation.OutcomeApplied ||
		reducedMotion.Snapshot == nil {
		t.Fatalf(
			"InvokeCommand(Progress Motion) = %+v, %v",
			reducedMotion,
			err,
		)
	}
	spinner = progressControl(
		reducedMotion.Snapshot,
		"progress.spinner",
	).Details.Progress
	if spinner == nil || !spinner.ReducedMotion ||
		spinner.Tick != 0 || spinner.FrameIndex != 0 {
		t.Fatalf("reduced-motion Progress details = %#v", spinner)
	}
	for index, command := range []struct {
		id     expletives.CommandID
		status string
	}{
		{demo.CommandProgressFail, "failed"},
		{demo.CommandProgressCancel, "cancelled"},
		{demo.CommandProgressComplete, "completed"},
	} {
		completion, invokeErr := client.InvokeCommand(
			ctx,
			"progress-terminal-"+string(rune('0'+index)),
			string(command.id),
			"",
		)
		if invokeErr != nil ||
			completion.Outcome != automation.OutcomeApplied ||
			completion.Snapshot == nil {
			t.Fatalf(
				"InvokeCommand(%s) = %+v, %v",
				command.id,
				completion,
				invokeErr,
			)
		}
		bar = progressControl(
			completion.Snapshot,
			"progress.bar.determinate",
		).Details.Progress
		if bar == nil || bar.Status != command.status {
			t.Fatalf("%s Progress status = %#v", command.id, bar)
		}
	}
	progressReset, err := client.InvokeCommand(
		ctx,
		"progress-reset",
		string(demo.CommandProgressReset),
		"",
	)
	if err != nil || progressReset.Outcome != automation.OutcomeApplied ||
		progressReset.Snapshot == nil {
		t.Fatalf("InvokeCommand(Progress Reset) = %+v, %v", progressReset, err)
	}
	bar = progressControl(
		progressReset.Snapshot,
		"progress.bar.determinate",
	).Details.Progress
	spinner = progressControl(
		progressReset.Snapshot,
		"progress.spinner",
	).Details.Progress
	if bar == nil || bar.Current != 42 || bar.Status != "running" ||
		spinner == nil || spinner.Tick != 0 || spinner.ReducedMotion {
		t.Fatal("automation Progress Reset evidence is incomplete")
	}
	navigationScreen, err := client.InvokeCommand(
		ctx,
		"show-navigation",
		string(demo.CommandNavigation),
		"",
	)
	if err != nil ||
		navigationScreen.Outcome != automation.OutcomeApplied ||
		navigationScreen.Snapshot == nil {
		t.Fatalf(
			"InvokeCommand(Navigation) = %+v, %v",
			navigationScreen,
			err,
		)
	}
	horizontal := progressControl(
		navigationScreen.Snapshot,
		"navigation.scrollbar.horizontal",
	)
	tabs := progressControl(
		navigationScreen.Snapshot,
		"navigation.tabs",
	)
	notebook := progressControl(
		navigationScreen.Snapshot,
		"navigation.notebook",
	)
	if !progressControl(
		navigationScreen.Snapshot,
		"screen.navigation",
	).Visible ||
		!horizontal.Focused ||
		horizontal.Details.ScrollBar == nil ||
		horizontal.Details.ScrollBar.Offset != 40 ||
		horizontal.Details.ScrollBar.TrackSize < 1 ||
		tabs.Details.TabbedPanel == nil ||
		tabs.Details.TabbedPanel.Selected != "overview" ||
		tabs.Details.TabbedPanel.Current != "overview" ||
		len(tabs.Details.TabbedPanel.Tabs) != 3 ||
		notebook.Details.TabbedPanel == nil ||
		notebook.Details.TabbedPanel.Selected != "one" ||
		len(notebook.Details.TabbedPanel.Tabs) != 2 {
		t.Fatal("initial attached Navigation evidence is incomplete")
	}
	scrollRight, err := client.InjectInput(
		ctx,
		"navigation-scroll-right",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "right"},
	)
	if err != nil ||
		scrollRight.Outcome != automation.OutcomeApplied ||
		scrollRight.Snapshot == nil ||
		scrollRight.Snapshot.Completion == nil ||
		scrollRight.Snapshot.Completion.Command !=
			string(demo.CommandNavigationChanged) ||
		progressControl(
			scrollRight.Snapshot,
			"navigation.scrollbar.horizontal",
		).Details.ScrollBar.Offset != 41 {
		t.Fatalf("attached ScrollBar Right completion = %+v, %v", scrollRight, err)
	}
	tabFocus, err := client.InjectInput(
		ctx,
		"navigation-focus-tabs",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "tab"},
	)
	if err != nil ||
		tabFocus.Outcome != automation.OutcomeApplied ||
		tabFocus.Snapshot == nil ||
		!progressControl(tabFocus.Snapshot, "navigation.tabs").Focused {
		t.Fatalf("attached TabbedPanel focus completion = %+v, %v", tabFocus, err)
	}
	tabRight, err := client.InjectInput(
		ctx,
		"navigation-tab-right",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "right"},
	)
	if err != nil ||
		tabRight.Outcome != automation.OutcomeApplied ||
		tabRight.Snapshot == nil {
		t.Fatalf("attached TabbedPanel Right completion = %+v, %v", tabRight, err)
	}
	tabDetails := progressControl(
		tabRight.Snapshot,
		"navigation.tabs",
	).Details.TabbedPanel
	if tabDetails == nil || tabDetails.Selected != "overview" ||
		tabDetails.Current != "details" {
		t.Fatalf("attached TabbedPanel focus/selection = %#v", tabDetails)
	}
	tabSpace, err := client.InjectInput(
		ctx,
		"navigation-tab-space",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "space"},
	)
	if err != nil ||
		tabSpace.Outcome != automation.OutcomeApplied ||
		tabSpace.Snapshot == nil ||
		tabSpace.Snapshot.Completion == nil ||
		tabSpace.Snapshot.Completion.Command !=
			string(demo.CommandNavigationChanged) {
		t.Fatalf("attached TabbedPanel Space completion = %+v, %v", tabSpace, err)
	}
	tabDetails = progressControl(
		tabSpace.Snapshot,
		"navigation.tabs",
	).Details.TabbedPanel
	if tabDetails == nil || tabDetails.Selected != "details" ||
		progressControl(
			tabSpace.Snapshot,
			"navigation.tabs.page.overview",
		).Visible ||
		!progressControl(
			tabSpace.Snapshot,
			"navigation.tabs.page.details",
		).Visible {
		t.Fatal("attached TabbedPanel selection did not switch visible page")
	}
	if _, err := client.InjectInput(
		ctx,
		"navigation-alt-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Navigation Alt down) error = %v", err)
	}
	notebookMnemonic, err := client.InjectInput(
		ctx,
		"navigation-notebook-mnemonic",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "w"},
	)
	if _, releaseErr := client.InjectInput(
		ctx,
		"navigation-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); releaseErr != nil {
		t.Fatalf("InjectInput(Navigation Alt up) error = %v", releaseErr)
	}
	if err != nil ||
		notebookMnemonic.Outcome != automation.OutcomeApplied ||
		notebookMnemonic.Snapshot == nil ||
		notebookMnemonic.Snapshot.Completion == nil ||
		notebookMnemonic.Snapshot.Completion.Command !=
			string(demo.CommandNavigationChanged) {
		t.Fatalf(
			"attached Notebook mnemonic completion = %+v, %v",
			notebookMnemonic,
			err,
		)
	}
	notebookDetails := progressControl(
		notebookMnemonic.Snapshot,
		"navigation.notebook",
	).Details.TabbedPanel
	if notebookDetails == nil || notebookDetails.Selected != "two" ||
		notebookDetails.Current != "two" ||
		progressControl(
			notebookMnemonic.Snapshot,
			"navigation.notebook.page.one",
		).Visible ||
		!progressControl(
			notebookMnemonic.Snapshot,
			"navigation.notebook.page.two",
		).Visible {
		t.Fatal("attached Notebook mnemonic did not switch visible page")
	}
	markdownScreen, err := client.InvokeCommand(
		ctx,
		"show-scrolling",
		string(demo.CommandScrolling),
		"",
	)
	if err != nil || markdownScreen.Outcome != automation.OutcomeApplied ||
		markdownScreen.Snapshot == nil {
		t.Fatalf("InvokeCommand(Scrolling) = %+v, %v", markdownScreen, err)
	}
	markdown := progressControl(markdownScreen.Snapshot, "content.markdown")
	markdownDetails := markdown.Details.Markdown
	logFollow := progressControl(
		markdownScreen.Snapshot,
		"content.log-follow",
	).Details.LogView
	logScrollback := progressControl(
		markdownScreen.Snapshot,
		"content.log-scrollback",
	).Details.LogView
	stream := progressControl(
		markdownScreen.Snapshot,
		"content.stream-drops",
	).Details.StreamView
	if !progressControl(markdownScreen.Snapshot, "screen.scrolling").Visible ||
		!markdown.Focused || markdownDetails == nil ||
		markdownDetails.SourceBytes == 0 || markdownDetails.BlockCount < 8 ||
		markdownDetails.RenderedRows < 8 ||
		!markdownDetails.Viewport.HorizontalVisible ||
		!markdownDetails.Viewport.VerticalVisible ||
		len(markdownDetails.Blocks) != expletives.MaxMarkdownSummaries ||
		!markdownDetails.SummariesTruncated ||
		logFollow == nil || logFollow.RetainedRecords != 5 ||
		logFollow.LastKey != "ready" || !logFollow.Follow ||
		logFollow.Viewport.ViewportBounds.Height < 1 ||
		logScrollback == nil || logScrollback.Follow ||
		logScrollback.Viewport.State.Offset.Y != 1 ||
		logScrollback.Viewport.ViewportBounds.Height < 1 ||
		stream == nil || stream.RetainedLines != 1 ||
		stream.PendingBytes == 0 || stream.DroppedLines == 0 ||
		stream.DroppedBytes == 0 || !stream.Follow ||
		stream.Viewport.ViewportBounds.Height < 1 {
		t.Fatalf(
			"attached content evidence: markdown=%+v log-follow=%+v log-scrollback=%+v stream=%+v",
			markdown,
			logFollow,
			logScrollback,
			stream,
		)
	}
	markdownStyles := map[automation.StyleID]bool{}
	for _, cell := range markdownScreen.Snapshot.Frame.Cells {
		if cell.Owner == markdown.ID {
			markdownStyles[cell.Style] = true
		}
	}
	for _, style := range []automation.StyleID{
		"markdown.heading", "markdown.emphasis", "markdown.strong",
		"markdown.code", "markdown.link", "markdown.quote",
	} {
		if !markdownStyles[style] {
			t.Fatalf("attached Markdown frame omitted style %q: %v", style, markdownStyles)
		}
	}
	streamStyles := map[automation.StyleID]bool{}
	streamReplacements := 0
	streamControl := progressControl(
		markdownScreen.Snapshot,
		"content.stream-drops",
	)
	for _, cell := range markdownScreen.Snapshot.Frame.Cells {
		if cell.Owner != streamControl.ID {
			continue
		}
		streamStyles[cell.Style] = true
		if cell.Grapheme == "\uFFFD" {
			streamReplacements++
		}
	}
	for _, style := range []automation.StyleID{
		"stream_view", "stream.truncated", "content.dropped",
	} {
		if !streamStyles[style] {
			t.Fatalf("attached StreamView frame omitted style %q: %v", style, streamStyles)
		}
	}
	if streamReplacements == 0 {
		t.Fatal("attached StreamView frame omitted inert replacement cells")
	}
	markdownEnd, err := client.InjectInput(
		ctx,
		"markdown-end",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "end"},
	)
	if err != nil || markdownEnd.Outcome != automation.OutcomeApplied ||
		markdownEnd.Snapshot == nil ||
		markdownEnd.Snapshot.Completion == nil ||
		markdownEnd.Snapshot.Completion.Command !=
			string(demo.CommandContentChanged) {
		t.Fatalf("attached Markdown End completion = %+v, %v", markdownEnd, err)
	}
	markdownDetails = progressControl(
		markdownEnd.Snapshot,
		"content.markdown",
	).Details.Markdown
	if markdownDetails == nil ||
		markdownDetails.Viewport.State.Offset !=
			markdownDetails.Viewport.MaximumOffset {
		t.Fatalf("attached Markdown End offset = %+v", markdownDetails)
	}
	tabLog, err := client.InjectInput(
		ctx,
		"content-tab-log",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "tab"},
	)
	if err != nil || tabLog.Outcome != automation.OutcomeApplied ||
		tabLog.Snapshot == nil ||
		!progressControl(tabLog.Snapshot, "content.log-follow").Focused {
		t.Fatalf("attached content Tab completion = %+v, %v", tabLog, err)
	}
	logPageUp, err := client.InjectInput(
		ctx,
		"content-log-page-up",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "page_up"},
	)
	if err != nil || logPageUp.Outcome != automation.OutcomeApplied ||
		logPageUp.Snapshot == nil || logPageUp.Snapshot.Completion == nil ||
		logPageUp.Snapshot.Completion.Command !=
			string(demo.CommandContentChanged) ||
		progressControl(
			logPageUp.Snapshot,
			"content.log-follow",
		).Details.LogView.Follow {
		t.Fatalf("attached LogView PageUp completion = %+v, %v", logPageUp, err)
	}
	logEnd, err := client.InjectInput(
		ctx,
		"content-log-end",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "end"},
	)
	if err != nil || logEnd.Outcome != automation.OutcomeApplied ||
		logEnd.Snapshot == nil ||
		!progressControl(
			logEnd.Snapshot,
			"content.log-follow",
		).Details.LogView.Follow {
		t.Fatalf("attached LogView End completion = %+v, %v", logEnd, err)
	}
	contentAppend, err := client.InvokeCommand(
		ctx,
		"content-append",
		string(demo.CommandContentAppend),
		"",
	)
	if err != nil || contentAppend.Outcome != automation.OutcomeApplied ||
		contentAppend.Snapshot == nil {
		t.Fatalf("InvokeCommand(Content Append) = %+v, %v", contentAppend, err)
	}
	logFollow = progressControl(
		contentAppend.Snapshot,
		"content.log-follow",
	).Details.LogView
	logScrollback = progressControl(
		contentAppend.Snapshot,
		"content.log-scrollback",
	).Details.LogView
	stream = progressControl(
		contentAppend.Snapshot,
		"content.stream-drops",
	).Details.StreamView
	if logFollow.LastKey != "live.0001" ||
		logFollow.Viewport.State.Offset != logFollow.Viewport.MaximumOffset ||
		logScrollback.LastKey != "live.0001" || logScrollback.Follow ||
		logScrollback.Viewport.State.Offset.Y != 1 ||
		stream.RetainedLines != 2 {
		t.Fatalf(
			"attached content append: log-follow=%+v log-scrollback=%+v stream=%+v",
			logFollow,
			logScrollback,
			stream,
		)
	}
	contentFollow, err := client.InvokeCommand(
		ctx,
		"content-follow",
		string(demo.CommandContentFollow),
		"",
	)
	if err != nil || contentFollow.Outcome != automation.OutcomeApplied ||
		contentFollow.Snapshot == nil ||
		!progressControl(
			contentFollow.Snapshot,
			"content.log-scrollback",
		).Details.LogView.Follow {
		t.Fatalf("InvokeCommand(Content Follow) = %+v, %v", contentFollow, err)
	}
	contentReset, err := client.InvokeCommand(
		ctx,
		"content-reset",
		string(demo.CommandScenarioReset),
		"",
	)
	if err != nil || contentReset.Outcome != automation.OutcomeApplied ||
		contentReset.Snapshot == nil ||
		progressControl(
			contentReset.Snapshot,
			"content.log-follow",
		).Details.LogView.LastKey != "ready" ||
		progressControl(
			contentReset.Snapshot,
			"content.log-scrollback",
		).Details.LogView.Follow {
		t.Fatalf("InvokeCommand(Content Reset) = %+v, %v", contentReset, err)
	}
	if _, err := client.InjectInput(
		ctx,
		"help-alt-down",
		automation.KeyEvent{Kind: automation.KeyDown, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt down) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"help-open",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "p"},
	); err != nil {
		t.Fatalf("InjectInput(Alt-P) error = %v", err)
	}
	if _, err := client.InjectInput(
		ctx,
		"help-alt-up",
		automation.KeyEvent{Kind: automation.KeyUp, Key: "alt"},
	); err != nil {
		t.Fatalf("InjectInput(Alt up) error = %v", err)
	}
	about, err := client.InjectInput(
		ctx,
		"help-about",
		automation.KeyEvent{Kind: automation.KeyPress, Key: "a"},
	)
	if err != nil {
		t.Fatalf("InjectInput(Help/About) error = %v", err)
	}
	aboutVisible := false
	aboutChecked := false
	if about.Snapshot != nil {
		for _, control := range about.Snapshot.Controls {
			switch control.Key {
			case "screen.about":
				aboutVisible = control.Visible
			case "menu.main":
				if control.Details.MenuBar == nil {
					continue
				}
				for _, entry := range control.Details.MenuBar.Entries {
					if entry.Key == "menu.help.about" {
						aboutChecked = entry.Checked
					}
				}
			}
		}
	}
	if about.Outcome != automation.OutcomeApplied ||
		about.Snapshot == nil ||
		about.Snapshot.Completion == nil ||
		about.Snapshot.Completion.Command != string(demo.CommandViewAbout) ||
		!aboutVisible || !aboutChecked {
		t.Fatalf(
			"Help/About outcome=%q command=%q visible=%t checked=%t",
			about.Outcome,
			func() string {
				if about.Snapshot == nil ||
					about.Snapshot.Completion == nil {
					return ""
				}
				return about.Snapshot.Completion.Command
			}(),
			aboutVisible,
			aboutChecked,
		)
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
