package expletives

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func dispatchTextKey(t *testing.T, app *App, request string, key Key) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%q) error = %v", key, err)
	}
	return completion
}

func textFieldDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) TextFieldDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.TextField
	if details == nil {
		t.Fatalf("%s has no TextFieldDetails", key)
	}
	return *details
}

func TestTextFieldValidatorConstructionAndMutation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 3})
	tests := map[string]*TextValidator{
		"missing enforcement": {
			Mode: TextValidationWhitelist, Characters: "abc",
		},
		"missing mode": {
			Enforcement: TextValidationSoft, Characters: "abc",
		},
		"missing characters": {
			Enforcement: TextValidationSoft, Mode: TextValidationWhitelist,
		},
		"control character": {
			Enforcement: TextValidationSoft, Mode: TextValidationBlacklist,
			Characters: "\t",
		},
	}
	for name, validator := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewTextField(app.Root(), TextFieldOptions{
				Validator: validator,
			}); !errors.Is(err, ErrValidation) {
				t.Fatalf("NewTextField() error = %v, want ErrValidation", err)
			}
		})
	}

	validator := &TextValidator{
		Enforcement: TextValidationSoft,
		Mode:        TextValidationWhitelist,
		Characters:  "aabc",
	}
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		Text: "abc", Validator: validator,
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	validator.Characters = "x"
	if got := field.Validator(); got == nil || got.Characters != "abc" {
		t.Fatalf("Validator() = %#v, want copied deduplicated abc", got)
	}
	if err := field.SetValidator(&TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationBlacklist,
		Characters:  "b",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetValidator(hard-invalid) error = %v", err)
	}
	if got := field.Validator(); got == nil ||
		got.Enforcement != TextValidationSoft {
		t.Fatalf("failed SetValidator changed policy to %#v", got)
	}
}

func TestTextFieldSoftValidationPaintingAndEditing(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 2})
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "soft",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		Text: "ab",
		Validator: &TextValidator{
			Enforcement: TextValidationSoft,
			Mode:        TextValidationWhitelist,
			Characters:  "abc",
		},
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	snapshot := app.Snapshot()
	for x := 0; x < 2; x++ {
		cell, _ := snapshot.Frame.Cell(x, 0)
		if cell.Style != "text_input.valid" {
			t.Fatalf("valid cell %d style = %q", x, cell.Style)
		}
	}

	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextKey(t, app, "invalid", "x")
	details := textFieldDetailsByKey(t, app, "soft")
	if details.Text != "abx" || details.Valid || !details.Editing ||
		details.Caret != 3 {
		t.Fatalf("invalid working details = %#v", details)
	}
	if field.Text() != "ab" {
		t.Fatalf("Text() during edit = %q, want committed ab", field.Text())
	}
	snapshot = app.Snapshot()
	for x, want := range []StyleID{
		"text_input.invalid",
		"text_input.invalid",
		"text_input.invalid_character",
	} {
		cell, _ := snapshot.Frame.Cell(x, 0)
		if cell.Style != want {
			t.Fatalf("invalid cell %d style = %q, want %q", x, cell.Style, want)
		}
	}
	if !snapshot.Cursor.Visible || snapshot.Cursor.Position != (Point{X: 3}) {
		t.Fatalf("editing Cursor = %#v", snapshot.Cursor)
	}

	dispatchTextKey(t, app, "left", KeyLeft)
	dispatchTextKey(t, app, "backspace", KeyBackspace)
	details = textFieldDetailsByKey(t, app, "soft")
	if details.Text != "ax" || details.Caret != 1 {
		t.Fatalf("edited details = %#v", details)
	}
	dispatchTextKey(t, app, "cancel", KeyEscape)
	if field.Text() != "ab" || field.Editing() {
		t.Fatalf("cancel left Text=%q Editing=%t", field.Text(), field.Editing())
	}

	dispatchTextKey(t, app, "edit-again", KeyEnter)
	dispatchTextKey(t, app, "type-c", "c")
	dispatchTextKey(t, app, "commit", KeyEnter)
	if field.Text() != "abc" || field.Editing() || !field.Valid() {
		t.Fatalf(
			"commit left Text=%q Editing=%t Valid=%t",
			field.Text(),
			field.Editing(),
			field.Valid(),
		)
	}
}

func TestTextFieldHardValidationAndPasswordRedaction(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	if _, err := NewTextField(app.Root(), TextFieldOptions{
		Text: "bad",
		Validator: &TextValidator{
			Enforcement: TextValidationHard,
			Mode:        TextValidationBlacklist,
			Characters:  "d",
		},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("hard-invalid constructor error = %v", err)
	}
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "password",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		Text: "ace", Password: true,
		Validator: &TextValidator{
			Enforcement: TextValidationHard,
			Mode:        TextValidationWhitelist,
			Characters:  "abcdef",
		},
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	details := textFieldDetailsByKey(t, app, "password")
	if details.Text != "" || !details.Redacted || !details.Password ||
		details.Length != 3 {
		t.Fatalf("password details = %#v", details)
	}
	if got := rowText(app.Snapshot(), 0); !strings.HasPrefix(got, "***") ||
		strings.Contains(got, "ace") {
		t.Fatalf("password rendered row = %q", got)
	}

	dispatchTextKey(t, app, "edit", KeyEnter)
	before := textFieldDetailsByKey(t, app, "password")
	if got := dispatchTextKey(t, app, "reject", "x").Outcome; got != OutcomeNoOp {
		t.Fatalf("hard-invalid Outcome = %q, want no_op", got)
	}
	after := textFieldDetailsByKey(t, app, "password")
	if after.Length != before.Length || after.Caret != before.Caret {
		t.Fatalf("hard-invalid changed details from %#v to %#v", before, after)
	}
	dispatchTextKey(t, app, "accept", "b")
	after = textFieldDetailsByKey(t, app, "password")
	if after.Text != "" || after.Length != 4 || !after.Valid {
		t.Fatalf("accepted password details = %#v", after)
	}
	if got := rowText(app.Snapshot(), 0); !strings.HasPrefix(got, "****") ||
		strings.Contains(got, "aceb") {
		t.Fatalf("edited password row = %q", got)
	}
	dispatchTextKey(t, app, "commit", KeyEnter)
	if field.Text() != "aceb" {
		t.Fatalf("password Text() = %q", field.Text())
	}
	if err := field.SetText("acez"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetText(hard-invalid) error = %v", err)
	}
	if field.Text() != "aceb" {
		t.Fatalf("failed SetText changed password to %q", field.Text())
	}
}

func TestTextFieldCommitNotificationAndTabGroupTraversal(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 3})
	registerActionCommand(t, app, "field.changed", "Changed", true)
	var called atomic.Int32
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		if command.ID != "field.changed" {
			t.Fatalf("command ID = %q", command.ID)
		}
		if command.Target == "" {
			t.Fatal("change command target is empty")
		}
		if got := controlByKey(t, app.Snapshot(), "field").
			Details.TextField.Text; got != "a" {
			t.Fatalf("router observed TextField value %q, want a", got)
		}
		called.Add(1)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	firstGroup, err := NewPanel(app.Root(), PanelOptions{})
	if err != nil {
		t.Fatalf("NewPanel(first) error = %v", err)
	}
	secondGroup, err := NewPanel(app.Root(), PanelOptions{})
	if err != nil {
		t.Fatalf("NewPanel(second) error = %v", err)
	}
	field, err := NewTextField(firstGroup, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{Width: 10, Height: 1},
		},
		ChangeCommand: "field.changed",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	button, err := NewButton(secondGroup, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "next",
			Bounds:        Rect{Y: 1, Width: 10, Height: 1},
		},
		Command: "field.changed",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	if err := field.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextKey(t, app, "type", "a")
	completion := dispatchTextKey(t, app, "tab", KeyTab)
	if completion.Command != "field.changed" ||
		completion.Outcome != OutcomeApplied {
		t.Fatalf("Tab completion = %#v", completion)
	}
	if field.Text() != "a" || field.Editing() || called.Load() != 1 {
		t.Fatalf(
			"Tab commit Text=%q Editing=%t called=%d",
			field.Text(),
			field.Editing(),
			called.Load(),
		)
	}
	if focused := app.Focused(); focused == nil || focused.ID() != button.ID() {
		t.Fatalf("focused after Tab = %#v, want Button", focused)
	}
	if err := field.SetText("b"); err != nil {
		t.Fatalf("SetText() error = %v", err)
	}
	if called.Load() != 1 {
		t.Fatalf("programmatic SetText notified %d times", called.Load())
	}
	if err := field.Focus(); err != nil {
		t.Fatalf("second Focus() error = %v", err)
	}
	dispatchTextKey(t, app, "edit-focus-loss", KeyEnter)
	dispatchTextKey(t, app, "type-focus-loss", "c")
	if err := button.Focus(); err != nil {
		t.Fatalf("Button.Focus() error = %v", err)
	}
	if field.Text() != "bc" || field.Editing() {
		t.Fatalf(
			"programmatic focus loss left Text=%q Editing=%t",
			field.Text(),
			field.Editing(),
		)
	}
	if called.Load() != 1 {
		t.Fatalf("programmatic focus loss notified %d times", called.Load())
	}
}

func TestTextFieldMenuFocusLossCommitsAndHidesCursor(t *testing.T) {
	t.Parallel()
	app, _, _, _, _ := buildMenuFixture(t)
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{X: 2, Y: 6, Width: 12, Height: 1},
		},
		Text: "a",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	if err := field.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextKey(t, app, "type", "b")
	if cursor := app.Snapshot().Cursor; !cursor.Visible {
		t.Fatalf("editing Cursor = %#v", cursor)
	}
	pressKey(t, app, "menu", "f9")
	if field.Text() != "ab" || field.Editing() {
		t.Fatalf(
			"menu focus loss left Text=%q Editing=%t",
			field.Text(),
			field.Editing(),
		)
	}
	if cursor := app.Snapshot().Cursor; cursor.Visible {
		t.Fatalf("Menu overlay left Cursor visible: %#v", cursor)
	}
}

func TestTextFieldLimitedUnicodeAndHorizontalView(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 8, Height: 2})
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "unicode",
			Bounds:        Rect{Width: 3, Height: 1},
		},
		Text: "e\u0301界",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	if field.Text() != "e\u0301�" {
		t.Fatalf("normalized Text() = %q", field.Text())
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	details := textFieldDetailsByKey(t, app, "unicode")
	if details.Length != 2 || details.Caret != 2 || details.ViewOffset != 0 {
		t.Fatalf("Unicode details = %#v", details)
	}
	dispatchTextKey(t, app, "one", "a")
	dispatchTextKey(t, app, "two", "b")
	details = textFieldDetailsByKey(t, app, "unicode")
	if details.Length != 4 || details.ViewOffset != 2 ||
		app.Snapshot().Cursor.Position.X != 2 {
		t.Fatalf(
			"scrolled details=%#v cursor=%#v",
			details,
			app.Snapshot().Cursor,
		)
	}
}

func TestTextFieldSnapshotDeepCopy(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	if _, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{AutomationKey: "field"},
		Text:         "abc",
		Validator: &TextValidator{
			Enforcement: TextValidationSoft,
			Mode:        TextValidationWhitelist,
			Characters:  "abc",
		},
	}); err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	first := app.Snapshot()
	details := controlByKey(t, first, "field").Details.TextField
	details.Text = "mutated"
	details.Validator.Characters = "mutated"
	current := textFieldDetailsByKey(t, app, "field")
	if current.Text != "abc" || current.Validator == nil ||
		current.Validator.Characters != "abc" {
		t.Fatalf("snapshot alias changed details to %#v", current)
	}
}
