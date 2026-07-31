package expletives

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"
)

func numberDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) NumberFieldDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.NumberField
	if details == nil {
		t.Fatalf("%s has no NumberFieldDetails", key)
	}
	return *details
}

func numberPointer(value float64) *float64 { return &value }

func TestNumberFieldConstructionValidationAndCopies(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 4})
	tests := map[string]NumberFieldOptions{
		"NaN": {Value: math.NaN()},
		"precision": {
			Value: 1, DecimalPlaces: MaxNumberDecimalPlaces + 1,
		},
		"value precision": {Value: 1.25, DecimalPlaces: 1},
		"reversed range": {
			Value: 1, Minimum: numberPointer(2), Maximum: numberPointer(1),
		},
		"below range": {Value: 0, Minimum: numberPointer(1)},
		"bound precision": {
			Value: 1, Minimum: numberPointer(0.25), DecimalPlaces: 1,
		},
	}
	for name, options := range tests {
		t.Run(name, func(t *testing.T) {
			options.AutomationKey = "invalid." + name
			if _, err := NewNumberField(app.Root(), options); !errors.Is(err, ErrValidation) {
				t.Fatalf("NewNumberField() error = %v, want ErrValidation", err)
			}
		})
	}
	if _, err := NewSpinBox(app.Root(), SpinBoxOptions{
		Value: 1, Step: -1,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("NewSpinBox(negative step) error = %v", err)
	}

	minimum, maximum := 0.0, 10.0
	field, err := NewNumberField(app.Root(), NumberFieldOptions{
		PanelOptions: PanelOptions{AutomationKey: "number"},
		Value:        2.5, Minimum: &minimum, Maximum: &maximum,
		DecimalPlaces: 1,
	})
	if err != nil {
		t.Fatalf("NewNumberField() error = %v", err)
	}
	minimum, maximum = 4, 5
	if got, ok := field.Minimum(); !ok || got != 0 {
		t.Fatalf("Minimum() = %v, %t", got, ok)
	}
	if got, ok := field.Maximum(); !ok || got != 10 {
		t.Fatalf("Maximum() = %v, %t", got, ok)
	}
	if field.Value() != 2.5 || field.DecimalPlaces() != 1 {
		t.Fatalf(
			"field Value=%v DecimalPlaces=%d",
			field.Value(),
			field.DecimalPlaces(),
		)
	}
	if err := field.SetValue(12); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetValue(out of range) error = %v", err)
	}
	if field.Value() != 2.5 {
		t.Fatalf("failed SetValue changed value to %v", field.Value())
	}
}

func TestNumberFieldEditingCommitInvalidAndTabPolicy(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 4})
	registerActionCommand(t, app, "number.changed", "Changed", true)
	var called atomic.Int32
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		if command.ID != "number.changed" {
			t.Fatalf("command ID = %q", command.ID)
		}
		called.Add(1)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	first, _ := NewPanel(app.Root(), PanelOptions{
		Bounds: Rect{Width: 30, Height: 2},
	})
	second, _ := NewPanel(app.Root(), PanelOptions{
		Bounds: Rect{Y: 2, Width: 30, Height: 2},
	})
	field, err := NewNumberField(first, NumberFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "number",
			Bounds:        Rect{Width: 10, Height: 1},
		},
		Value: 12.5, Minimum: numberPointer(0), Maximum: numberPointer(20),
		DecimalPlaces: 1, ChangeCommand: "number.changed",
	})
	if err != nil {
		t.Fatalf("NewNumberField() error = %v", err)
	}
	button, err := NewButton(second, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "next",
			Bounds:        Rect{Y: 2, Width: 10, Height: 1},
		},
		Command: "number.changed",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	if err := field.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextKey(t, app, "home", KeyHome)
	for index := 0; index < len("12.5"); index++ {
		dispatchTextKey(t, app, "delete-"+string(rune('a'+index)), KeyDelete)
	}
	dispatchTextKey(t, app, "minus", "-")
	details := numberDetailsByKey(t, app, "number")
	if details.Valid || details.InvalidReason == "" || details.Text != "-" {
		t.Fatalf("incomplete numeric details = %#v", details)
	}
	cell, _ := app.Snapshot().Frame.Cell(0, 0)
	if cell.Style != "text_input.focused_invalid" {
		t.Fatalf("incomplete numeric style = %q", cell.Style)
	}
	if completion := dispatchTextKey(t, app, "invalid-enter", KeyEnter); completion.Outcome != OutcomeNoOp || !field.Editing() {
		t.Fatalf("invalid Enter completion = %#v", completion)
	}
	if completion := dispatchTextKey(t, app, "invalid-tab", KeyTab); completion.Outcome != OutcomeNoOp || !field.Editing() {
		t.Fatalf("invalid Tab completion = %#v", completion)
	}
	if focused := app.Focused(); focused == nil || focused.ID() != field.ID() {
		t.Fatalf("invalid Tab moved focus to %#v", focused)
	}
	dispatchTextKey(t, app, "delete-minus", KeyBackspace)
	for index, key := range []Key{"7", ".", "5"} {
		dispatchTextKey(t, app, "type-"+string(rune('a'+index)), key)
	}
	completion := dispatchTextKey(t, app, "commit", KeyEnter)
	if completion.Command != "number.changed" ||
		completion.Outcome != OutcomeApplied ||
		field.Value() != 7.5 || field.Editing() || called.Load() != 1 {
		t.Fatalf(
			"commit=%#v Value=%v Editing=%t called=%d",
			completion,
			field.Value(),
			field.Editing(),
			called.Load(),
		)
	}

	dispatchTextKey(t, app, "edit-again", KeyEnter)
	dispatchTextKey(t, app, "append-hard-reject", "x")
	dispatchTextKey(t, app, "tab-valid", KeyTab)
	if focused := app.Focused(); focused == nil || focused.ID() != button.ID() {
		t.Fatalf("valid Tab focused %#v, want Button", focused)
	}
	if called.Load() != 1 {
		t.Fatalf("unchanged Tab notified %d times", called.Load())
	}
}

func TestSpinBoxBracketStepClampAndEditMode(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	registerActionCommand(t, app, "spin.changed", "Changed", true)
	if err := app.SetCommandRouter(func(
		_ context.Context,
		_ Command,
	) CommandResult {
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	spin, err := NewSpinBox(app.Root(), SpinBoxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "spin",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		Value: 1, Minimum: numberPointer(0), Maximum: numberPointer(2),
		DecimalPlaces: 1, Step: 0.5, ChangeCommand: "spin.changed",
	})
	if err != nil {
		t.Fatalf("NewSpinBox() error = %v", err)
	}
	if spin.Step() != 0.5 {
		t.Fatalf("Step() = %v", spin.Step())
	}
	for index, want := range []float64{1.5, 2, 2} {
		completion := dispatchTextKey(
			t,
			app,
			"increment-"+string(rune('a'+index)),
			KeyRightBracket,
		)
		wantOutcome := OutcomeApplied
		if index == 2 {
			wantOutcome = OutcomeNoOp
		}
		if spin.Value() != want || completion.Outcome != wantOutcome {
			t.Fatalf(
				"increment %d Value=%v completion=%#v",
				index,
				spin.Value(),
				completion,
			)
		}
	}
	dispatchTextKey(t, app, "decrement", KeyLeftBracket)
	if spin.Value() != 1.5 {
		t.Fatalf("decrement Value=%v", spin.Value())
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	before := numberDetailsByKey(t, app, "spin")
	if completion := dispatchTextKey(t, app, "bracket-edit", KeyRightBracket); completion.Outcome != OutcomeNoOp {
		t.Fatalf("editing bracket completion = %#v", completion)
	}
	after := numberDetailsByKey(t, app, "spin")
	if before.Text != after.Text || before.Caret != after.Caret {
		t.Fatalf("editing bracket changed details from %#v to %#v", before, after)
	}
}

func TestNumberFieldInvalidForcedFocusLossCancels(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 3})
	first, _ := NewPanel(app.Root(), PanelOptions{})
	second, _ := NewPanel(app.Root(), PanelOptions{})
	field, err := NewNumberField(first, NumberFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "number",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		Value: 4,
	})
	if err != nil {
		t.Fatalf("NewNumberField() error = %v", err)
	}
	registerActionCommand(t, app, "next", "Next", true)
	button, err := NewButton(second, ButtonOptions{
		Command: "next",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	if err := field.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextKey(t, app, "home", KeyHome)
	dispatchTextKey(t, app, "delete", KeyDelete)
	dispatchTextKey(t, app, "minus", "-")
	if field.Valid() {
		t.Fatal("incomplete edit unexpectedly valid")
	}
	if err := button.Focus(); err != nil {
		t.Fatalf("Button.Focus() error = %v", err)
	}
	details := numberDetailsByKey(t, app, "number")
	if field.Value() != 4 || details.Text != "4" ||
		details.Editing || !details.Valid {
		t.Fatalf("forced focus loss details = %#v Value=%v", details, field.Value())
	}
}

func TestNumberFieldSnapshotDeepCopy(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	if _, err := NewSpinBox(app.Root(), SpinBoxOptions{
		PanelOptions: PanelOptions{AutomationKey: "spin"},
		Value:        2,
		Minimum:      numberPointer(0),
		Maximum:      numberPointer(4),
		Step:         1,
	}); err != nil {
		t.Fatalf("NewSpinBox() error = %v", err)
	}
	first := app.Snapshot()
	details := controlByKey(t, first, "spin").Details.NumberField
	*details.Minimum = 99
	*details.Maximum = 100
	current := numberDetailsByKey(t, app, "spin")
	if current.Minimum == nil || *current.Minimum != 0 ||
		current.Maximum == nil || *current.Maximum != 4 {
		t.Fatalf("snapshot alias changed details to %#v", current)
	}
}
