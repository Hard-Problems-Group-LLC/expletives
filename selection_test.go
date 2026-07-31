package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func dispatchSelectionKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"selection-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func TestFocusGroupsUseTabBetweenAndArrowsWithin(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 50, Height: 8})
	left := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "group.left",
		Bounds:        Rect{X: 1, Y: 1, Width: 18, Height: 4},
	})
	right := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "group.right",
		Bounds:        Rect{X: 25, Y: 1, Width: 18, Height: 4},
	})
	leftFirst, err := NewCheckbox(left, CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "left.first",
			Bounds:        Rect{Width: 14, Height: 1},
		},
		Label: "Left first",
	})
	if err != nil {
		t.Fatalf("NewCheckbox(left first) error = %v", err)
	}
	leftSecond, err := NewCheckbox(left, CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "left.second",
			Bounds:        Rect{Y: 2, Width: 14, Height: 1},
		},
		Label: "Left second",
	})
	if err != nil {
		t.Fatalf("NewCheckbox(left second) error = %v", err)
	}
	rightFirst, err := NewCheckbox(right, CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "right.first",
			Bounds:        Rect{Width: 14, Height: 1},
		},
		Label: "Right first",
	})
	if err != nil {
		t.Fatalf("NewCheckbox(right first) error = %v", err)
	}
	rightSecond, err := NewCheckbox(right, CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "right.second",
			Bounds:        Rect{Y: 2, Width: 14, Height: 1},
		},
		Label: "Right second",
	})
	if err != nil {
		t.Fatalf("NewCheckbox(right second) error = %v", err)
	}

	if app.Focused() != leftFirst {
		t.Fatalf("initial focus = %#v, want left first", app.Focused())
	}
	dispatchSelectionKey(t, app, "within-left", KeyDown)
	if app.Focused() != leftSecond {
		t.Fatalf("Down focus = %#v, want left second", app.Focused())
	}
	dispatchSelectionKey(t, app, "next-group", KeyTab)
	if app.Focused() != rightFirst {
		t.Fatalf("Tab focus = %#v, want right first", app.Focused())
	}

	if _, err := app.DispatchKey(
		context.Background(),
		"selection-test",
		"shift-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyShift},
	); err != nil {
		t.Fatalf("Shift down error = %v", err)
	}
	dispatchSelectionKey(t, app, "previous-group", KeyTab)
	if _, err := app.DispatchKey(
		context.Background(),
		"selection-test",
		"shift-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyShift},
	); err != nil {
		t.Fatalf("Shift up error = %v", err)
	}
	if app.Focused() != leftSecond {
		t.Fatalf("Shift-Tab focus = %#v, want left second", app.Focused())
	}
	dispatchSelectionKey(t, app, "across-right", KeyRight)
	if app.Focused() != rightSecond {
		t.Fatalf("Right cross-group focus = %#v, want right second", app.Focused())
	}
}

func TestCheckboxTransitionsMnemonicNotificationAndSnapshot(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 4})
	registerActionCommand(t, app, "selection.checkbox.changed", "Changed", true)
	checkbox, err := NewCheckbox(app.Root(), CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "checkbox.three",
			Bounds:        Rect{X: 1, Y: 1, Width: 20, Height: 1},
		},
		Label: "Choice", Mnemonic: "c", ThreeState: true,
		ChangeCommand: "selection.checkbox.changed",
	})
	if err != nil {
		t.Fatalf("NewCheckbox() error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		// This getter would deadlock if the router ran under the toolkit lock.
		_ = checkbox.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	for index, want := range []CheckState{
		CheckChecked,
		CheckIndeterminate,
		CheckUnchecked,
	} {
		completion := dispatchSelectionKey(
			t,
			app,
			"space-"+string(rune('0'+index)),
			KeySpace,
		)
		if completion.Command != "selection.checkbox.changed" ||
			completion.Outcome != OutcomeApplied ||
			checkbox.State() != want {
			t.Fatalf(
				"Space transition %d = completion:%+v state:%q, want %q",
				index,
				completion,
				checkbox.State(),
				want,
			)
		}
	}

	// A modifier lifecycle requires Down/Press/Up rather than three Presses.
	if _, err := app.DispatchKey(
		context.Background(),
		"selection-mnemonic",
		"mnemonic-alt-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	); err != nil {
		t.Fatalf("Alt down error = %v", err)
	}
	mnemonic, err := app.DispatchKey(
		context.Background(),
		"selection-mnemonic",
		"mnemonic-c",
		KeyEvent{Kind: KeyEventPress, Key: "c"},
	)
	if err != nil {
		t.Fatalf("Alt-C error = %v", err)
	}
	if _, err := app.DispatchKey(
		context.Background(),
		"selection-mnemonic",
		"mnemonic-alt-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyAlt},
	); err != nil {
		t.Fatalf("Alt up error = %v", err)
	}
	if checkbox.State() != CheckChecked ||
		mnemonic.Command != "selection.checkbox.changed" {
		t.Fatalf("Alt-C = completion:%+v state:%q", mnemonic, checkbox.State())
	}

	mu.Lock()
	routedCount := len(routed)
	for _, command := range routed {
		if command.Target != checkbox.ID() {
			t.Fatalf("Checkbox command target = %q, want %q", command.Target, checkbox.ID())
		}
	}
	mu.Unlock()
	if routedCount != 4 {
		t.Fatalf("routed Checkbox changes = %d, want 4", routedCount)
	}
	if err := checkbox.SetState(CheckIndeterminate); err != nil {
		t.Fatalf("SetState(indeterminate) error = %v", err)
	}
	mu.Lock()
	if len(routed) != routedCount {
		t.Fatal("programmatic Checkbox setter invoked ChangeCommand")
	}
	mu.Unlock()

	snapshot := app.Snapshot()
	control := controlByKey(t, snapshot, "checkbox.three")
	if control.Details.Checkbox == nil ||
		control.Details.Checkbox.State != CheckIndeterminate ||
		!control.Details.Checkbox.ThreeState ||
		control.Details.Checkbox.ChangeCommand !=
			"selection.checkbox.changed" {
		t.Fatalf("CheckboxDetails = %#v", control.Details.Checkbox)
	}
	snapshot.Controls[1].Details.Checkbox.State = CheckUnchecked
	if got := controlByKey(
		t,
		app.Snapshot(),
		"checkbox.three",
	).Details.Checkbox.State; got != CheckIndeterminate {
		t.Fatal("caller-mutated CheckboxDetails aliased App state")
	}
	if got := rowText(app.Snapshot(), 1)[1:11]; got != "[-] Choice" {
		t.Fatalf("Checkbox rendering = %q", got)
	}

	if _, err := NewCheckbox(app.Root(), CheckboxOptions{
		Label: "Invalid", State: CheckIndeterminate,
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("two-state indeterminate error = %v", err)
	}
}

func TestRadioGroupExclusivityNavigationRepairAndValidation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 36, Height: 8})
	registerActionCommand(t, app, "selection.radio.changed", "Changed", true)
	group, err := NewRadioGroup(app.Root(), RadioGroupOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "radio.group",
			Bounds:        Rect{X: 1, Y: 1, Width: 24, Height: 4},
		},
		ChangeCommand: "selection.radio.changed",
	})
	if err != nil {
		t.Fatalf("NewRadioGroup() error = %v", err)
	}
	one, err := NewRadioButton(group, RadioButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "radio.one",
			Bounds:        Rect{Width: 16, Height: 1},
		},
		Value: "one", Label: "One", Mnemonic: "o", Selected: true,
	})
	if err != nil {
		t.Fatalf("NewRadioButton(one) error = %v", err)
	}
	if _, err := NewRadioButton(group, RadioButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "radio.disabled",
			Bounds:        Rect{Y: 1, Width: 16, Height: 1},
		},
		Value: "disabled", Label: "Disabled", Mnemonic: "d",
		Disabled: true, DisabledReason: "Not available",
	}); err != nil {
		t.Fatalf("NewRadioButton(disabled) error = %v", err)
	}
	two, err := NewRadioButton(group, RadioButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "radio.two",
			Bounds:        Rect{Y: 2, Width: 16, Height: 1},
		},
		Value: "two", Label: "Two", Mnemonic: "t",
	})
	if err != nil {
		t.Fatalf("NewRadioButton(two) error = %v", err)
	}
	var mu sync.Mutex
	var commands []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = group.Value()
		mu.Lock()
		commands = append(commands, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	if group.Value() != "one" || !one.Selected() || two.Selected() {
		t.Fatalf("initial radio value=%q one=%t two=%t",
			group.Value(), one.Selected(), two.Selected())
	}
	if err := one.Focus(); err != nil {
		t.Fatalf("one.Focus() error = %v", err)
	}
	down := dispatchSelectionKey(t, app, "radio-down", KeyDown)
	if group.Value() != "one" || app.Focused() != two ||
		down.Command != "" {
		t.Fatalf(
			"Down radio value=%q focus=%#v completion=%+v",
			group.Value(),
			app.Focused(),
			down,
		)
	}
	enter := dispatchSelectionKey(t, app, "radio-select", KeyEnter)
	if group.Value() != "two" || app.Focused() != two ||
		enter.Command != "selection.radio.changed" {
		t.Fatalf(
			"Enter radio value=%q focus=%#v completion=%+v",
			group.Value(),
			app.Focused(),
			enter,
		)
	}
	dispatchSelectionKey(t, app, "radio-home", KeyHome)
	if group.Value() != "two" {
		t.Fatalf("Home changed selected value to %q", group.Value())
	}
	if app.Focused() != one {
		t.Fatalf("Home focus = %#v, want first enabled option", app.Focused())
	}
	space := dispatchSelectionKey(t, app, "radio-select-one", KeySpace)
	if group.Value() != "one" ||
		space.Command != "selection.radio.changed" {
		t.Fatalf("Space radio value=%q completion=%+v", group.Value(), space)
	}
	if err := two.Destroy(); err != nil {
		t.Fatalf("Destroy(two) error = %v", err)
	}
	if group.Value() != "one" || !one.Selected() || app.Focused() != one {
		t.Fatalf(
			"destroy repair value=%q selected=%t focus=%#v",
			group.Value(),
			one.Selected(),
			app.Focused(),
		)
	}
	if err := group.SetValue("missing"); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("SetValue(missing) error = %v", err)
	}
	if group.Value() != "one" {
		t.Fatal("rejected radio value changed group")
	}

	details := controlByKey(
		t,
		app.Snapshot(),
		"radio.group",
	).Details.RadioGroup
	if details == nil || details.Value != "one" || len(details.Options) != 2 ||
		!details.Options[0].Selected || details.Options[1].Enabled {
		t.Fatalf("RadioGroupDetails = %#v", details)
	}
	mu.Lock()
	for _, command := range commands {
		if command.Target != group.ID() {
			t.Fatalf("RadioGroup command target = %q, want %q", command.Target, group.ID())
		}
	}
	mu.Unlock()

	if _, err := NewRadioButton(group, RadioButtonOptions{
		Value: "one", Label: "Duplicate", Mnemonic: "x",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate RadioButton value error = %v", err)
	}

	other := mustApp(t, Size{Width: 10, Height: 4})
	tx := other.NewTransaction()
	pendingGroup, err := tx.NewRadioGroup(other.Root(), RadioGroupOptions{})
	if err != nil {
		t.Fatalf("Transaction.NewRadioGroup() error = %v", err)
	}
	if _, err := tx.NewRadioButton(pendingGroup, RadioButtonOptions{
		Value: "a", Label: "A", Selected: true,
	}); err != nil {
		t.Fatalf("Transaction.NewRadioButton(a) error = %v", err)
	}
	if _, err := tx.NewRadioButton(pendingGroup, RadioButtonOptions{
		Value: "b", Label: "B", Selected: true,
	}); err != nil {
		t.Fatalf("Transaction.NewRadioButton(b) error = %v", err)
	}
	if err := tx.Commit(context.Background()); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("multiple selected RadioButtons error = %v", err)
	}
}

func TestChoiceFieldNavigationClampOptionsAndFocusTraversal(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 50, Height: 8})
	registerActionCommand(t, app, "selection.choice.changed", "Changed", true)
	checkbox, err := NewCheckbox(app.Root(), CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "choice.before",
			Bounds:        Rect{Y: 0, Width: 16, Height: 1},
		},
		Label: "Before",
	})
	if err != nil {
		t.Fatalf("NewCheckbox(before) error = %v", err)
	}
	field, err := NewCycleField(app.Root(), CycleFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "choice.field",
			Bounds:        Rect{Y: 1, Width: 30, Height: 1},
		},
		Label: "Mode", Mnemonic: "m",
		Options: []SelectionOption{
			{Value: "a", Label: "Alpha"},
			{
				Value: "b", Label: "Beta", Disabled: true,
				DisabledReason: "Unavailable",
			},
			{Value: "c", Label: "Charlie"},
		},
		Value: "a", ChangeCommand: "selection.choice.changed",
	})
	if err != nil {
		t.Fatalf("NewCycleField() error = %v", err)
	}
	clamped, err := NewSelectField(app.Root(), SelectFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "choice.clamped",
			Bounds:        Rect{Y: 2, Width: 30, Height: 1},
		},
		Label: "Clamp", Clamp: true,
		Options: []SelectionOption{
			{Value: "x", Label: "X"},
			{Value: "y", Label: "Y"},
		},
		Value: "x",
	})
	if err != nil {
		t.Fatalf("NewSelectField() error = %v", err)
	}
	empty, err := NewCycleField(app.Root(), CycleFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "choice.empty",
			Bounds:        Rect{Y: 3, Width: 20, Height: 1},
		},
		Label: "Empty",
	})
	if err != nil {
		t.Fatalf("NewCycleField(empty) error = %v", err)
	}
	var calls int
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = field.Value()
		if command.Target != field.ID() {
			t.Errorf("choice command target = %q, want %q", command.Target, field.ID())
		}
		calls++
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	if app.Focused() != checkbox {
		t.Fatalf("initial focus = %#v, want Checkbox", app.Focused())
	}
	dispatchSelectionKey(t, app, "choice-down", KeyDown)
	if app.Focused() != field {
		t.Fatalf("Down focus = %#v, want CycleField", app.Focused())
	}
	dispatchSelectionKey(t, app, "choice-next", KeyRightBracket)
	if field.Value() != "c" {
		t.Fatalf("] selected %q, want c", field.Value())
	}
	dispatchSelectionKey(t, app, "choice-wrap", KeyRightBracket)
	if field.Value() != "a" {
		t.Fatalf("] wrapped to %q, want a", field.Value())
	}
	dispatchSelectionKey(t, app, "choice-previous", KeyLeftBracket)
	if field.Value() != "c" {
		t.Fatalf("[ selected %q, want c", field.Value())
	}
	if calls != 3 {
		t.Fatalf("choice ChangeCommand calls = %d, want 3", calls)
	}

	if err := clamped.Focus(); err != nil {
		t.Fatalf("clamped.Focus() error = %v", err)
	}
	dispatchSelectionKey(t, app, "clamp-previous", KeyLeftBracket)
	if clamped.Value() != "x" {
		t.Fatalf("clamped previous selected %q", clamped.Value())
	}
	dispatchSelectionKey(t, app, "clamp-next", KeyRightBracket)
	dispatchSelectionKey(t, app, "clamp-past-end", KeyRightBracket)
	if clamped.Value() != "y" {
		t.Fatalf("clamped next selected %q", clamped.Value())
	}
	if err := empty.Focus(); !errors.Is(err, ErrNotFocusable) {
		t.Fatalf("empty.Focus() error = %v", err)
	}

	replacement := []SelectionOption{
		{Value: "new", Label: "New"},
		{Value: "later", Label: "Later"},
	}
	if err := field.SetOptions(replacement, ""); err != nil {
		t.Fatalf("SetOptions() error = %v", err)
	}
	replacement[0].Label = "Aliased"
	if field.Value() != "new" || field.Options()[0].Label != "New" {
		t.Fatal("choice options were not copied or repaired to first enabled")
	}
	if err := field.SetValue("missing"); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("SetValue(missing) error = %v", err)
	}

	snapshot := app.Snapshot()
	control := controlByKey(t, snapshot, "choice.field")
	if control.Kind != ControlCycleField ||
		control.Details.ChoiceField == nil ||
		control.Details.ChoiceField.Value != "new" ||
		control.Details.ChoiceField.SelectedIndex != 0 ||
		len(control.Details.ChoiceField.Options) != 2 {
		t.Fatalf("ChoiceFieldDetails = %#v", control.Details.ChoiceField)
	}
	if controlByKey(t, snapshot, "choice.clamped").Kind != ControlSelectField {
		t.Fatal("SelectField did not retain its distinct control kind")
	}
}
