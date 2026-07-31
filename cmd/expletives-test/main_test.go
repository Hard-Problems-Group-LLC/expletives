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
	screenEvidence := make(map[string]bool)
	chromeEvidence := make(map[string]bool)
	menuEvidence := false
	statusEvidence := false
	automationPanel := false
	var statusID automation.ControlID
	helpEnd := false
	expectedRootMnemonics := map[string]automation.Key{
		"menu.file":     "i",
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
					len(control.Details.MenuBar.Entries) == 58 &&
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
					control.Details.TextField.Validator == nil &&
					!control.Details.TextField.Password
		case "input.text.soft_whitelist":
			inputEvidence[control.Key] =
				control.Kind == "text_field" &&
					control.Details.TextField != nil &&
					control.Details.TextField.Validator != nil &&
					control.Details.TextField.Validator.Enforcement == "soft" &&
					control.Details.TextField.Validator.Mode == "whitelist"
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
		}
	}
	if len(observe.Snapshot.Frame.Cells) > 2 {
		menuEvidence = menuEvidence &&
			observe.Snapshot.Frame.Cells[0].Style == "menu_bar" &&
			observe.Snapshot.Frame.Cells[3].Grapheme == "i" &&
			observe.Snapshot.Frame.Cells[3].Style == "menu.mnemonic" &&
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
		len(inputEvidence) != 4 ||
		len(screenEvidence) != 13 ||
		len(chromeEvidence) != 5 {
		t.Fatalf(
			"catalog evidence: menu=%t status=%t screens=%#v chrome=%#v display=%#v action=%#v input=%#v",
			menuEvidence,
			statusEvidence,
			screenEvidence,
			chromeEvidence,
			displayEvidence,
			actionEvidence,
			inputEvidence,
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
			automation.KeyEvent{Kind: automation.KeyPress, Key: "i"},
		); err != nil {
			t.Fatalf("InjectInput(Alt-I) error = %v", err)
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
