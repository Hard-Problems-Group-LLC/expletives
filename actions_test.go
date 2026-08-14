package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func rowText(snapshot Snapshot, row int) string {
	var text strings.Builder
	for column := range snapshot.Frame.Size.Width {
		cell, _ := snapshot.Frame.Cell(column, row)
		text.WriteString(cell.Grapheme)
	}
	return text.String()
}

func registerActionCommand(
	t *testing.T,
	app *App,
	id CommandID,
	label string,
	enabled bool,
) {
	t.Helper()
	definition := CommandDefinition{
		ID: id, Label: label, Enabled: enabled, Automation: true,
	}
	if !enabled {
		definition.DisabledReason = "Unavailable in this test"
	}
	if err := app.RegisterCommand(definition); err != nil {
		t.Fatalf("RegisterCommand(%q) error = %v", id, err)
	}
}

func actionByKey(
	t *testing.T,
	snapshot Snapshot,
	key string,
) ControlSnapshot {
	t.Helper()
	control := controlByKey(t, snapshot, key)
	if control.Details.Action == nil {
		t.Fatalf("control %q has no ActionDetails", key)
	}
	return control
}

func TestActionConstructionFocusAndRendering(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 5})
	registerActionCommand(t, app, "action.open", "Open", true)
	registerActionCommand(t, app, "action.disabled", "Disabled", false)

	open, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.open",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Command: "action.open",
		Default: true,
	})
	if err != nil {
		t.Fatalf("NewButton(open) error = %v", err)
	}
	disabled, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.disabled",
			Bounds:        Rect{X: 15, Y: 1, Width: 14, Height: 1},
		},
		Command: "action.disabled",
	})
	if err != nil {
		t.Fatalf("NewButton(disabled) error = %v", err)
	}
	if _, ok := any(open).(Container); ok {
		t.Fatal("Button unexpectedly implements Container")
	}
	if got := open.MinimumSize(); got != (Size{Width: 10, Height: 2}) {
		t.Fatalf("Open minimum = %+v, want 10x2", got)
	}
	if got := disabled.MinimumSize(); got != (Size{Width: 12, Height: 2}) {
		t.Fatalf("Disabled minimum = %+v, want 12x2", got)
	}
	if app.Focused() != open {
		t.Fatalf("initial focus = %#v, want first eligible Button", app.Focused())
	}

	snapshot := app.Snapshot()
	openState := actionByKey(t, snapshot, "button.open")
	if !openState.Focused || !openState.Details.Action.Enabled ||
		!openState.Details.Action.Default ||
		openState.Details.Action.Label != "Open" {
		t.Fatalf("open action snapshot = %#v", openState)
	}
	disabledState := actionByKey(t, snapshot, "button.disabled")
	if disabledState.Focused || disabledState.Details.Action.Enabled ||
		disabledState.Details.Action.DisabledReason == "" {
		t.Fatalf("disabled action snapshot = %#v", disabledState)
	}
	if got := rowText(snapshot, 1)[1:13]; got != "    Open    " {
		t.Fatalf("focused one-row Button override = %q", got)
	}
	if got := rowText(snapshot, 1)[15:29]; got != "   Disabled   " {
		t.Fatalf("disabled one-row Button override = %q", got)
	}
	for x := disabledState.AbsoluteBounds.X; x < disabledState.AbsoluteBounds.X+disabledState.AbsoluteBounds.Width; x++ {
		cell, _ := snapshot.Frame.Cell(x, disabledState.AbsoluteBounds.Y)
		if strings.Contains("Disabled", cell.Grapheme) &&
			(cell.Style != "button.disabled" ||
				cell.Foreground == cell.Background) {
			t.Fatalf("disabled Button label cell = %+v", cell)
		}
	}
}

func TestGlobalAltBindingPrecedesLocalControlMnemonic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 3})
	registerActionCommand(t, app, "action.local", "Local", true)
	registerActionCommand(t, app, "app.quit", "Quit", true)
	if _, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.local",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Command:  "action.local",
		Mnemonic: "x",
	}); err != nil {
		t.Fatalf("NewButton(local mnemonic) error = %v", err)
	}
	if err := app.BindChord(
		Chord{Key: "x", Modifiers: []Key{KeyAlt}},
		CommandBinding{Command: "app.quit"},
	); err != nil {
		t.Fatalf("BindChord(Alt-X) error = %v", err)
	}
	var routed []CommandID
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		routed = append(routed, command.ID)
		outcome := OutcomeApplied
		if command.ID == "app.quit" {
			outcome = OutcomeExited
		}
		return CommandResult{Outcome: outcome}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		"global-quit-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	); err != nil {
		t.Fatalf("DispatchKey(Alt down) error = %v", err)
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		"global-quit",
		KeyEvent{Kind: KeyEventPress, Key: "x"},
	)
	if err != nil {
		t.Fatalf("DispatchKey(Alt-X) error = %v", err)
	}
	if completion.Command != "app.quit" ||
		completion.Outcome != OutcomeExited ||
		len(routed) != 1 || routed[0] != "app.quit" {
		t.Fatalf("Alt-X completion=%+v routed=%v", completion, routed)
	}
}

func TestActionActivationTraversalPressAndReset(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 4})
	registerActionCommand(t, app, "action.first", "First", true)
	registerActionCommand(t, app, "action.second", "Second", true)
	first, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.first",
			Bounds:        Rect{X: 0, Y: 0, Width: 12, Height: 2},
		},
		Command: "action.first",
	})
	if err != nil {
		t.Fatalf("NewButton(first) error = %v", err)
	}
	second, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.second",
			Bounds:        Rect{X: 13, Y: 0, Width: 13, Height: 2},
		},
		Command: "action.second",
	})
	if err != nil {
		t.Fatalf("NewButton(second) error = %v", err)
	}

	var mu sync.Mutex
	var commands []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		mu.Lock()
		commands = append(commands, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	if _, err := app.DispatchKey(
		context.Background(), "keyboard", "right",
		KeyEvent{Kind: KeyEventPress, Key: KeyRight},
	); err != nil {
		t.Fatalf("DispatchKey(Right) error = %v", err)
	}
	if app.Focused() != second {
		t.Fatalf("focus after Right = %#v, want second", app.Focused())
	}
	down, err := app.DispatchKey(
		context.Background(), "keyboard", "enter-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyEnter},
	)
	if err != nil {
		t.Fatalf("DispatchKey(Enter down) error = %v", err)
	}
	downSnapshot, err := app.SnapshotAt(down.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(down) error = %v", err)
	}
	downButton := actionByKey(t, downSnapshot, "button.second")
	if !downButton.Details.Action.Pressed {
		t.Fatal("Enter KeyDown did not publish pressed state")
	}
	downBounds := downButton.AbsoluteBounds
	left, _ := downSnapshot.Frame.Cell(downBounds.X, downBounds.Y)
	shiftedBody, _ := downSnapshot.Frame.Cell(downBounds.X+2, downBounds.Y)
	right, _ := downSnapshot.Frame.Cell(
		downBounds.X+downBounds.Width-1,
		downBounds.Y,
	)
	bottom, _ := downSnapshot.Frame.Cell(downBounds.X+2, downBounds.Y+1)
	if left.Style != "button.shadow" ||
		shiftedBody.Style != "button.pressed" ||
		right.Grapheme != " " || right.Style != "button.pressed" ||
		bottom.Grapheme != " " || bottom.Style != "button.shadow" {
		t.Fatalf(
			"pressed Button cells = left %+v body %+v right %+v bottom %+v",
			left,
			shiftedBody,
			right,
			bottom,
		)
	}
	up, err := app.DispatchKey(
		context.Background(), "keyboard", "enter-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyEnter},
	)
	if err != nil {
		t.Fatalf("DispatchKey(Enter up) error = %v", err)
	}
	if up.Command != "action.second" {
		t.Fatalf("Enter KeyUp command = %q", up.Command)
	}
	mu.Lock()
	if len(commands) != 1 || commands[0].Target != second.ID() ||
		commands[0].Source != "keyboard" {
		t.Fatalf("routed commands = %#v", commands)
	}
	mu.Unlock()

	if _, err := app.DispatchKey(
		context.Background(), "keyboard", "space-down",
		KeyEvent{Kind: KeyEventDown, Key: KeySpace},
	); err != nil {
		t.Fatalf("DispatchKey(Space down) error = %v", err)
	}
	reset, err := app.ResetInput(
		context.Background(), "keyboard", "reset",
	)
	if err != nil || reset.Outcome != OutcomeApplied {
		t.Fatalf("ResetInput() = %+v, %v", reset, err)
	}
	if actionByKey(t, app.Snapshot(), "button.second").Details.Action.Pressed {
		t.Fatal("ResetInput left a pressed Button")
	}
	mu.Lock()
	if len(commands) != 1 {
		t.Fatalf("ResetInput activated captured Button: %#v", commands)
	}
	mu.Unlock()

	if err := first.Focus(); err != nil {
		t.Fatalf("first.Focus() error = %v", err)
	}
	direct, err := first.Activate(
		context.Background(), "automation", "direct",
	)
	if err != nil || direct.Command != "action.first" {
		t.Fatalf("first.Activate() = %+v, %v", direct, err)
	}
}

func TestDirectionalFocusUsesNearestPeerGeometry(t *testing.T) {
	t.Parallel()

	t.Run("cross-axis distance", func(t *testing.T) {
		app := mustApp(t, Size{Width: 80, Height: 20})
		for _, id := range []CommandID{"focus.current", "focus.aligned", "focus.offset"} {
			registerActionCommand(t, app, id, string(id), true)
		}
		newGroup := func(key string) *Panel {
			t.Helper()
			group, err := NewPanel(app.Root(), PanelOptions{
				AutomationKey: key,
				Bounds:        Rect{Width: 80, Height: 20},
			})
			if err != nil {
				t.Fatal(err)
			}
			return group
		}
		current, err := NewButton(newGroup("focus.group.current"), ButtonOptions{
			PanelOptions: PanelOptions{Bounds: Rect{X: 30, Y: 8, Width: 10, Height: 1}},
			Command:      "focus.current",
		})
		if err != nil {
			t.Fatal(err)
		}
		aligned, err := NewButton(newGroup("focus.group.aligned"), ButtonOptions{
			PanelOptions: PanelOptions{Bounds: Rect{X: 30, Y: 10, Width: 10, Height: 1}},
			Command:      "focus.aligned",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewButton(newGroup("focus.group.offset"), ButtonOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Y: 9, Width: 4, Height: 1}},
			Command:      "focus.offset",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := current.Focus(); err != nil {
			t.Fatal(err)
		}
		dispatchTextKey(t, app, "nearest-down", KeyDown)
		if app.Focused() != aligned {
			t.Fatalf("Down focus = %#v, want aligned peer", app.Focused())
		}
	})

	t.Run("focusable ancestor is not a peer", func(t *testing.T) {
		app := mustApp(t, Size{Width: 80, Height: 20})
		registerActionCommand(t, app, "focus.child", "Child", true)
		registerActionCommand(t, app, "focus.peer", "Peer", true)
		viewport, err := NewScrollablePanel(app.Root(), ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "focus.viewport",
					Bounds:        Rect{X: 20, Y: 2, Width: 30, Height: 10},
				},
				State: ViewportState{ContentSize: Size{Width: 40, Height: 20}},
			},
			BorderForm: BorderNone,
		})
		if err != nil {
			t.Fatal(err)
		}
		child, err := NewButton(viewport.Content(), ButtonOptions{
			PanelOptions: PanelOptions{Bounds: Rect{X: 10, Y: 3, Width: 8, Height: 1}},
			Command:      "focus.child",
		})
		if err != nil {
			t.Fatal(err)
		}
		peer, err := NewButton(app.Root(), ButtonOptions{
			PanelOptions: PanelOptions{Bounds: Rect{X: 30, Y: 8, Width: 8, Height: 1}},
			Command:      "focus.peer",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := child.Focus(); err != nil {
			t.Fatal(err)
		}
		dispatchTextKey(t, app, "ancestor-down", KeyDown)
		if app.Focused() != peer {
			t.Fatalf("Down focus = %#v, want non-hierarchical peer", app.Focused())
		}
	})
}

func TestActionMnemonicsRolesAndInvalidation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 5})
	registerActionCommand(t, app, "action.accept", "Accept", true)
	registerActionCommand(t, app, "action.cancel", "Cancel", true)
	accept, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.accept",
			Bounds:        Rect{X: 0, Y: 0, Width: 13, Height: 1},
		},
		Command:  "action.accept",
		Mnemonic: "A",
		Default:  true,
	})
	if err != nil {
		t.Fatalf("NewButton(accept) error = %v", err)
	}
	cancel, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.cancel",
			Bounds:        Rect{X: 14, Y: 0, Width: 13, Height: 1},
		},
		Command: "action.cancel",
		Cancel:  true,
	})
	if err != nil {
		t.Fatalf("NewButton(cancel) error = %v", err)
	}
	targetLabel, err := NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "label.cancel",
			Bounds:        Rect{X: 0, Y: 2, Width: 8, Height: 1},
		},
		Text:     "Cancel",
		Target:   cancel,
		Mnemonic: "c",
	})
	if err != nil {
		t.Fatalf("NewLabel(target) error = %v", err)
	}
	_ = targetLabel
	var commands []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		commands = append(commands, command)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	key := func(id string, event KeyEvent) Completion {
		t.Helper()
		completion, keyErr := app.DispatchKey(
			context.Background(), "keyboard", id, event,
		)
		if keyErr != nil {
			t.Fatalf("DispatchKey(%s) error = %v", id, keyErr)
		}
		return completion
	}
	key("alt-down", KeyEvent{Kind: KeyEventDown, Key: KeyAlt})
	if got := key("accept-mnemonic", KeyEvent{
		Kind: KeyEventPress, Key: "a",
	}); got.Command != "action.accept" {
		t.Fatalf("Alt-A command = %q", got.Command)
	}
	key("alt-up", KeyEvent{Kind: KeyEventUp, Key: KeyAlt})
	key("right", KeyEvent{Kind: KeyEventPress, Key: KeyRight})
	if app.Focused() != cancel {
		t.Fatalf("focus before Label mnemonic = %#v", app.Focused())
	}
	key("left", KeyEvent{Kind: KeyEventPress, Key: KeyLeft})
	if app.Focused() != accept {
		t.Fatalf("focus after Left = %#v", app.Focused())
	}
	key("alt-down-2", KeyEvent{Kind: KeyEventDown, Key: KeyAlt})
	focusCompletion := key("cancel-focus", KeyEvent{
		Kind: KeyEventPress, Key: "c",
	})
	if focusCompletion.Command != "" || app.Focused() != cancel {
		t.Fatalf("Alt-C completion=%+v focus=%#v", focusCompletion, app.Focused())
	}
	key("alt-up-2", KeyEvent{Kind: KeyEventUp, Key: KeyAlt})
	if got := key("escape", KeyEvent{
		Kind: KeyEventPress, Key: KeyEscape,
	}); got.Command != "action.cancel" {
		t.Fatalf("Escape command = %q", got.Command)
	}

	if err := app.ReplaceCommand(CommandDefinition{
		ID: "action.cancel", Label: "Cancel", Enabled: false,
		DisabledReason: "Not now",
	}); err != nil {
		t.Fatalf("ReplaceCommand(disable cancel) error = %v", err)
	}
	if app.Focused() != accept {
		t.Fatalf("focus after disabling cancel = %#v, want accept", app.Focused())
	}
	if err := app.RemoveCommand("action.accept"); err != nil {
		t.Fatalf("RemoveCommand(accept) error = %v", err)
	}
	if app.Focused() != nil {
		t.Fatalf("focus after removing last eligible command = %#v", app.Focused())
	}
	unknown := actionByKey(t, app.Snapshot(), "button.accept")
	if unknown.Details.Action.Enabled ||
		unknown.Details.Action.DisabledReason != "Command is not registered" {
		t.Fatalf("unknown Button state = %#v", unknown.Details.Action)
	}
	if len(commands) != 2 {
		t.Fatalf("routed commands = %#v, want mnemonic and cancel", commands)
	}
}

func TestActionConstructionValidationAndHotkeyReflection(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 60, Height: 3})
	registerActionCommand(t, app, "action.one", "One", true)
	registerActionCommand(t, app, "action.two", "Two", false)

	items := []HotkeyBarItem{{Command: "action.one"}, {Command: "action.two"}}
	bar, err := NewHotkeyBar(app.Root(), HotkeyBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "hotkeys",
			Bounds:        Rect{Width: 60, Height: 1},
		},
		Items: items,
	})
	if err != nil {
		t.Fatalf("NewHotkeyBar() error = %v", err)
	}
	items[0].Command = "changed"
	if got := bar.Items()[0].Command; got != "action.one" {
		t.Fatalf("HotkeyBar retained caller slice: first command = %q", got)
	}
	if _, ok := any(bar).(Container); ok {
		t.Fatal("HotkeyBar unexpectedly implements Container")
	}
	if err := app.BindChord(
		Chord{Key: "o", Modifiers: []Key{KeyControl}},
		CommandBinding{Command: "action.one"},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	state := controlByKey(t, app.Snapshot(), "hotkeys")
	if state.Details.HotkeyBar == nil ||
		len(state.Details.HotkeyBar.Items) != 2 ||
		state.Details.HotkeyBar.Items[0].Chord == nil ||
		state.Details.HotkeyBar.Items[0].Chord.Key != "o" ||
		state.Details.HotkeyBar.Items[1].Enabled {
		t.Fatalf("HotkeyBar details = %#v", state.Details.HotkeyBar)
	}
	if got := rowText(app.Snapshot(), 0)[:21]; got !=
		" Ctrl+O One (Two)    " {
		t.Fatalf("HotkeyBar row prefix = %q", got)
	}

	if _, err := NewButton(app.Root(), ButtonOptions{
		Command: "missing",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("NewButton(missing) error = %v", err)
	}
	if _, err := NewButton(app.Root(), ButtonOptions{
		Command: "action.one", Default: true, Cancel: true,
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("NewButton(default+cancel) error = %v", err)
	}
	if _, err := NewHotkeyBar(app.Root(), HotkeyBarOptions{
		Items: []HotkeyBarItem{
			{Command: "action.one"},
			{Command: "action.one"},
		},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("NewHotkeyBar(duplicate) error = %v", err)
	}
}

func TestActionRoleAndMnemonicUniquenessIsAtomic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	registerActionCommand(t, app, "action.a", "A", true)
	registerActionCommand(t, app, "action.b", "B", true)
	if _, err := NewButton(app.Root(), ButtonOptions{
		Command: "action.a", Mnemonic: "a", Default: true,
	}); err != nil {
		t.Fatalf("NewButton(first) error = %v", err)
	}
	before := app.Snapshot()
	if _, err := NewButton(app.Root(), ButtonOptions{
		Command: "action.b", Mnemonic: "a",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate mnemonic error = %v", err)
	}
	if _, err := NewButton(app.Root(), ButtonOptions{
		Command: "action.b", Default: true,
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate default error = %v", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence ||
		len(after.Controls) != len(before.Controls) {
		t.Fatalf(
			"failed Action construction published state: before=%d/%d after=%d/%d",
			before.Sequence,
			len(before.Controls),
			after.Sequence,
			len(after.Controls),
		)
	}
}

func TestCommandPresentationReplacementUpdatesActionGeometry(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 2})
	registerActionCommand(t, app, "action.rename", "Go", true)
	button, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.rename",
			Bounds:        Rect{Width: 20, Height: 1},
		},
		Command: "action.rename",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	before := app.Snapshot().Sequence
	if err := app.ReplaceCommand(CommandDefinition{
		ID: "action.rename", Label: "Go Further", Enabled: true,
		Checked: true,
	}); err != nil {
		t.Fatalf("ReplaceCommand() error = %v", err)
	}
	if got := button.MinimumSize(); got != (Size{Width: 14, Height: 2}) {
		t.Fatalf("renamed Button minimum = %+v, want 14x2", got)
	}
	state := actionByKey(t, app.Snapshot(), "button.rename")
	if state.Details.Action.Label != "Go Further" ||
		!state.Details.Action.Checked ||
		app.Snapshot().Sequence <= before {
		t.Fatalf("renamed Action state = %#v", state.Details.Action)
	}
	if err := app.ReplaceCommand(CommandDefinition{
		ID:             "action.rename",
		Label:          "No",
		Enabled:        false,
		DisabledReason: "Policy",
	}); err != nil {
		t.Fatalf("ReplaceCommand(disable) error = %v", err)
	}
	if app.Focused() != nil {
		t.Fatalf("disabled focused Button retained focus: %#v", app.Focused())
	}
	if err := app.ReplaceCommand(CommandDefinition{
		ID: "action.rename", Label: "No", Enabled: true,
		DisabledReason: "stale",
	}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("enabled command with disabled reason error = %v", err)
	}
}

func TestHotkeyBarAggregateCapacityIsAtomic(t *testing.T) {
	app := mustApp(t, Size{Width: 4, Height: 1})
	items := make([]HotkeyBarItem, MaxHotkeyBarItems)
	for index := range items {
		id := CommandID(fmt.Sprintf("action.%02d", index))
		registerActionCommand(t, app, id, string(id), true)
		items[index] = HotkeyBarItem{Command: id}
	}
	for index := 0; index < MaxActionItems/MaxHotkeyBarItems; index++ {
		if _, err := NewHotkeyBar(app.Root(), HotkeyBarOptions{
			Items: items,
		}); err != nil {
			t.Fatalf("NewHotkeyBar(%d) error = %v", index, err)
		}
	}
	before := app.Snapshot()
	if _, err := NewHotkeyBar(app.Root(), HotkeyBarOptions{
		Items: items,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("excess HotkeyBar error = %v", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence ||
		len(after.Controls) != len(before.Controls) {
		t.Fatal("excess HotkeyBar changed App state")
	}
}
