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

func dispatchTextChord(
	t *testing.T,
	app *App,
	request string,
	modifier Key,
	key Key,
) Completion {
	t.Helper()
	dispatch := func(suffix string, event KeyEvent) Completion {
		completion, err := app.DispatchKey(
			context.Background(),
			"keyboard",
			request+"-"+suffix,
			event,
		)
		if err != nil {
			t.Fatalf("DispatchKey(%s) error = %v", suffix, err)
		}
		return completion
	}
	dispatch("down", KeyEvent{Kind: KeyEventDown, Key: modifier})
	completion := dispatch("press", KeyEvent{Kind: KeyEventPress, Key: key})
	dispatch("up", KeyEvent{Kind: KeyEventUp, Key: modifier})
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
		if cell.Style != "text_input.focused_valid" {
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
		"text_input.focused_invalid",
		"text_input.focused_invalid",
		"text_input.focused_invalid_character",
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

func TestTextFieldLiveEditCurrentValueAndSubmitCommands(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 3})
	for _, command := range []CommandID{
		"field.changed", "field.edited", "field.submitted",
	} {
		registerActionCommand(t, app, command, string(command), true)
	}
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{Width: 20, Height: 1},
		},
		ChangeCommand: "field.changed",
		EditCommand:   "field.edited",
		SubmitCommand: "field.submitted",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	commands := make([]CommandID, 0, 4)
	values := make([]string, 0, 4)
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		commands = append(commands, command.ID)
		values = append(values, field.CurrentText())
		if command.Target != field.ID() {
			t.Fatalf("command target = %q, want %q", command.Target, field.ID())
		}
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	dispatchTextKey(t, app, "begin", KeyEnter)
	if completion := dispatchTextKey(t, app, "type-a", "a"); completion.Command != "field.edited" || field.Text() != "" ||
		field.CurrentText() != "a" {
		t.Fatalf("live key completion/Text/CurrentText = %#v/%q/%q", completion, field.Text(), field.CurrentText())
	}
	completion, err := app.DispatchTextInput(
		context.Background(),
		"paste",
		"paste-accent",
		TextInputEvent{Kind: TextInputPaste, Text: "é"},
	)
	if err != nil || completion.Command != "field.edited" ||
		field.CurrentText() != "aé" {
		t.Fatalf("paste completion/current = %#v/%q error=%v", completion, field.CurrentText(), err)
	}
	if completion := dispatchTextKey(t, app, "caret-left", KeyLeft); completion.Command != "" {
		t.Fatalf("caret-only completion command = %q", completion.Command)
	}
	if completion := dispatchTextKey(t, app, "cancel", KeyEscape); completion.Command != "field.edited" || field.CurrentText() != "" {
		t.Fatalf("cancel completion/current = %#v/%q", completion, field.CurrentText())
	}
	if err := field.SetText("ok"); err != nil {
		t.Fatalf("SetText() error = %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("programmatic SetText routed commands %v", commands)
	}
	dispatchTextKey(t, app, "begin-submit", KeyEnter)
	dispatchTextKey(t, app, "type-x", "x")
	if completion := dispatchTextKey(t, app, "submit", KeyEnter); completion.Command != "field.submitted" || field.Text() != "okx" ||
		field.Editing() {
		t.Fatalf("submit completion/Text/Editing = %#v/%q/%t", completion, field.Text(), field.Editing())
	}
	wantCommands := []CommandID{
		"field.edited", "field.edited", "field.edited",
		"field.edited", "field.submitted",
	}
	wantValues := []string{"a", "aé", "", "okx", "okx"}
	if strings.Join(commandIDsAsStrings(commands), ",") !=
		strings.Join(commandIDsAsStrings(wantCommands), ",") ||
		strings.Join(values, "|") != strings.Join(wantValues, "|") {
		t.Fatalf("routed commands/values = %v/%v, want %v/%v", commands, values, wantCommands, wantValues)
	}
}

func commandIDsAsStrings(commands []CommandID) []string {
	values := make([]string, len(commands))
	for index, command := range commands {
		values[index] = string(command)
	}
	return values
}

func TestTextFieldDistinctEditBackgroundsAndUTF8ByteStyles(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 250, Height: 3})
	registerActionCommand(t, app, "field.edited", "Edited", true)
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{Width: 240, Height: 1},
		},
		Text:         strings.Repeat("x", 174),
		EditCommand:  "field.edited",
		FocusedStyle: "test.text.selected",
		EditingStyle: "test.text.editing",
		ByteStyles: []TextFieldByteStyle{
			{MinimumBytes: 0, Style: "test.text.normal"},
			{MinimumBytes: 175, Style: "test.text.warning"},
			{MinimumBytes: 222, Style: "test.text.danger"},
		},
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	optionsByteStyles := []TextFieldByteStyle{{
		MinimumBytes: 0,
		Style:        "test.text.danger",
	}}
	copied, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "copied",
			Bounds:        Rect{Y: 1, Width: 20, Height: 1},
		},
		Text:       "x",
		ByteStyles: optionsByteStyles,
	})
	if err != nil {
		t.Fatalf("NewTextField(copied) error = %v", err)
	}
	optionsByteStyles[0].Style = "test.text.normal"
	copiedDetails := controlByKey(t, app.Snapshot(), "copied").Details.TextField
	if copiedDetails == nil || copiedDetails.ByteStyles[0].Style != "test.text.danger" ||
		copied.CurrentText() != "x" {
		t.Fatalf("TextField presentation policy was not copied: %#v", copiedDetails)
	}
	if err := app.SetCommandRouter(func(_ context.Context, _ Command) CommandResult {
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	assertTextFieldCellStyles(t, app.Snapshot(), field, 174, "test.text.normal", 200, "test.text.selected")
	dispatchTextKey(t, app, "begin", KeyEnter)
	assertTextFieldCellStyles(t, app.Snapshot(), field, 174, "test.text.normal", 200, "test.text.editing")
	dispatchTextKey(t, app, "byte-175", "x")
	assertTextFieldCellStyles(t, app.Snapshot(), field, 175, "test.text.warning", 200, "test.text.editing")

	if err := field.SetText(strings.Repeat("x", 221)); err != nil {
		t.Fatalf("SetText(221) error = %v", err)
	}
	if _, err := field.Activate(context.Background(), "test", "activate-221"); err != nil {
		t.Fatalf("Activate(221) error = %v", err)
	}
	assertTextFieldCellStyles(t, app.Snapshot(), field, 221, "test.text.warning", 230, "test.text.editing")
	dispatchTextKey(t, app, "byte-222", "x")
	assertTextFieldCellStyles(t, app.Snapshot(), field, 222, "test.text.danger", 230, "test.text.editing")

	if err := field.SetText(strings.Repeat("x", 173) + "é"); err != nil {
		t.Fatalf("SetText(multibyte 175) error = %v", err)
	}
	if len(field.CurrentText()) != 175 {
		t.Fatalf("multibyte CurrentText bytes = %d, want 175", len(field.CurrentText()))
	}
	if _, err := field.Activate(context.Background(), "test", "activate-multibyte"); err != nil {
		t.Fatalf("Activate(multibyte) error = %v", err)
	}
	assertTextFieldCellStyles(t, app.Snapshot(), field, 174, "test.text.warning", 200, "test.text.editing")
	if err := field.SetPassword(true); err != nil {
		t.Fatalf("SetPassword(true) error = %v", err)
	}
	cell, _ := app.Snapshot().Frame.Cell(0, 0)
	if cell.Grapheme != "*" || cell.Style == "test.text.warning" ||
		field.CurrentText() != strings.Repeat("x", 173)+"é" {
		t.Fatalf("masked byte-band/current cell = %#v current-bytes=%d", cell, len(field.CurrentText()))
	}
}

func TestTextFieldRejectsInvalidByteStylePolicies(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	tests := map[string][]TextFieldByteStyle{
		"negative": {{MinimumBytes: -1, Style: "text_input.valid"}},
		"beyond input bound": {{
			MinimumBytes: MaxTextInputBytes + 1,
			Style:        "text_input.valid",
		}},
		"duplicate": {
			{MinimumBytes: 1, Style: "text_input.valid"},
			{MinimumBytes: 1, Style: "text_input.invalid"},
		},
		"decreasing": {
			{MinimumBytes: 2, Style: "text_input.valid"},
			{MinimumBytes: 1, Style: "text_input.invalid"},
		},
		"missing style": {{MinimumBytes: 0}},
	}
	tooMany := make([]TextFieldByteStyle, MaxTextFieldByteStyles+1)
	for index := range tooMany {
		tooMany[index] = TextFieldByteStyle{
			MinimumBytes: index,
			Style:        "text_input.valid",
		}
	}
	tests["too many"] = tooMany
	for name, policy := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewTextField(app.Root(), TextFieldOptions{
				ByteStyles: policy,
			}); err == nil {
				t.Fatal("NewTextField() accepted invalid byte-style policy")
			}
		})
	}
}

func TestTextFieldMaximumBytesRejectsAtomicallyAndRetainsRecoveryKeys(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 250, Height: 3})
	for _, command := range []CommandID{"field.edited", "field.submitted"} {
		registerActionCommand(t, app, command, string(command), true)
	}
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "field",
			Bounds:        Rect{Width: 240, Height: 1},
		},
		Text:          strings.Repeat("x", 232),
		MaximumBytes:  233,
		EditCommand:   "field.edited",
		SubmitCommand: "field.submitted",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	var commands []CommandID
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		commands = append(commands, command.ID)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	if field.MaximumBytes() != 233 ||
		textFieldDetailsByKey(t, app, "field").MaximumBytes != 233 {
		t.Fatalf(
			"MaximumBytes accessor/details = %d/%d",
			field.MaximumBytes(),
			textFieldDetailsByKey(t, app, "field").MaximumBytes,
		)
	}
	if err := field.SetText(strings.Repeat("y", 234)); !errors.Is(err, ErrTextLimit) || field.Text() != strings.Repeat("x", 232) {
		t.Fatalf("SetText(over limit) error/Text = %v/%d bytes", err, len(field.Text()))
	}

	dispatchTextKey(t, app, "begin", KeyEnter)
	before := textFieldDetailsByKey(t, app, "field")
	if completion := dispatchTextKey(t, app, "reject-multibyte", "é"); completion.Outcome != OutcomeNoOp || completion.Command != "" {
		t.Fatalf("over-limit key completion = %+v", completion)
	}
	after := textFieldDetailsByKey(t, app, "field")
	if after.Text != before.Text || after.Caret != before.Caret || len(commands) != 0 {
		t.Fatalf("rejected key changed details/commands = %#v/%#v/%v", before, after, commands)
	}
	if completion := dispatchTextKey(t, app, "fill", "x"); completion.Command != "field.edited" || len(field.CurrentText()) != 233 {
		t.Fatalf("exact-limit key completion/current = %+v/%d", completion, len(field.CurrentText()))
	}
	before = textFieldDetailsByKey(t, app, "field")
	if completion := dispatchTextKey(t, app, "reject-extra", "y"); completion.Outcome != OutcomeNoOp || completion.Command != "" {
		t.Fatalf("extra key completion = %+v", completion)
	}
	after = textFieldDetailsByKey(t, app, "field")
	if after.Text != before.Text || after.Caret != before.Caret || len(commands) != 1 {
		t.Fatalf("extra key changed details/commands = %#v/%#v/%v", before, after, commands)
	}

	dispatchTextKey(t, app, "left-at-limit", KeyLeft)
	if details := textFieldDetailsByKey(t, app, "field"); details.Caret != 232 {
		t.Fatalf("Left at limit caret = %d, want 232", details.Caret)
	}
	dispatchTextKey(t, app, "delete-at-limit", KeyDelete)
	if len(field.CurrentText()) != 232 {
		t.Fatalf("Delete at limit left %d bytes, want 232", len(field.CurrentText()))
	}
	dispatchTextKey(t, app, "backspace-after-delete", KeyBackspace)
	if len(field.CurrentText()) != 231 {
		t.Fatalf("Backspace left %d bytes, want 231", len(field.CurrentText()))
	}
	dispatchTextKey(t, app, "refill-multibyte", "é")
	if len(field.CurrentText()) != 233 {
		t.Fatalf("multibyte refill left %d bytes, want 233", len(field.CurrentText()))
	}
	if completion := dispatchTextKey(t, app, "submit-at-limit", KeyEnter); completion.Command != "field.submitted" || field.Editing() || len(field.Text()) != 233 {
		t.Fatalf("submit-at-limit completion/editing/text = %+v/%t/%d", completion, field.Editing(), len(field.Text()))
	}

	if err := field.SetText(strings.Repeat("x", 232)); err != nil {
		t.Fatal(err)
	}
	dispatchTextKey(t, app, "begin-paste", KeyEnter)
	commandCount := len(commands)
	completion, err := app.DispatchTextInput(
		context.Background(),
		"paste",
		"paste-over-limit",
		TextInputEvent{Kind: TextInputPaste, Text: "é"},
	)
	if err != nil || completion.Outcome != OutcomeRejected ||
		completion.Code != "text_capacity" || len(field.CurrentText()) != 232 ||
		len(commands) != commandCount {
		t.Fatalf(
			"over-limit paste completion/current/commands = %+v/%d/%v error=%v",
			completion,
			len(field.CurrentText()),
			commands,
			err,
		)
	}
	dispatchTextKey(t, app, "fill-before-selection", "x")
	dispatchTextChord(t, app, "select-one", KeyShift, KeyLeft)
	before = textFieldDetailsByKey(t, app, "field")
	completion, err = app.DispatchTextInput(
		context.Background(),
		"paste",
		"replace-one-over-limit",
		TextInputEvent{Kind: TextInputPaste, Text: "é"},
	)
	after = textFieldDetailsByKey(t, app, "field")
	if err != nil || completion.Outcome != OutcomeRejected ||
		completion.Code != "text_capacity" || after.Text != before.Text ||
		after.Caret != before.Caret ||
		after.SelectionStart != before.SelectionStart ||
		after.SelectionEnd != before.SelectionEnd {
		t.Fatalf(
			"over-limit selection replacement = %+v before=%#v after=%#v error=%v",
			completion,
			before,
			after,
			err,
		)
	}
	dispatchTextChord(t, app, "select-two", KeyShift, KeyLeft)
	completion, err = app.DispatchTextInput(
		context.Background(),
		"paste",
		"replace-two-at-limit",
		TextInputEvent{Kind: TextInputPaste, Text: "é"},
	)
	if err != nil || completion.Outcome != OutcomeApplied ||
		completion.Command != "field.edited" || len(field.CurrentText()) != 233 {
		t.Fatalf(
			"exact-limit selection replacement = %+v current=%d error=%v",
			completion,
			len(field.CurrentText()),
			err,
		)
	}
}

func TestTextFieldMaximumBytesConstructionAndZeroValue(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	field, err := NewTextField(app.Root(), TextFieldOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if field.MaximumBytes() != MaxTextInputBytes {
		t.Fatalf("zero-value MaximumBytes = %d, want %d", field.MaximumBytes(), MaxTextInputBytes)
	}
	for name, options := range map[string]TextFieldOptions{
		"negative maximum":             {MaximumBytes: -1},
		"maximum beyond global":        {MaximumBytes: MaxTextInputBytes + 1},
		"initial value beyond maximum": {Text: "é", MaximumBytes: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewTextField(app.Root(), options); err == nil {
				t.Fatal("NewTextField() accepted invalid maximum/value")
			}
		})
	}
}

func assertTextFieldCellStyles(
	t *testing.T,
	snapshot SnapshotV1,
	field *TextField,
	textCells int,
	textStyle StyleID,
	emptyColumn int,
	emptyStyle StyleID,
) {
	t.Helper()
	for column := 0; column < textCells; column++ {
		cell, found := snapshot.Frame.Cell(column, 0)
		if !found || cell.Owner != field.ID() || cell.Style != textStyle {
			t.Fatalf("text cell %d = %#v found=%t, want owner %q style %q", column, cell, found, field.ID(), textStyle)
		}
	}
	cell, found := snapshot.Frame.Cell(emptyColumn, 0)
	if !found || cell.Owner != field.ID() || cell.Style != emptyStyle {
		t.Fatalf("empty cell %d = %#v found=%t, want owner %q style %q", emptyColumn, cell, found, field.ID(), emptyStyle)
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

func TestTextFieldSelectionAndBoundedTextInput(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 2})
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "selection",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		Text: "abcd",
		Validator: &TextValidator{
			Enforcement: TextValidationHard,
			Mode:        TextValidationBlacklist,
			Characters:  "x",
		},
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	dispatchTextChord(t, app, "select-d", KeyShift, KeyLeft)
	dispatchTextChord(t, app, "select-c", KeyShift, KeyLeft)
	details := textFieldDetailsByKey(t, app, "selection")
	if details.SelectionStart != 2 || details.SelectionEnd != 4 {
		t.Fatalf("selection details = %#v", details)
	}
	if cell, _ := app.Snapshot().Frame.Cell(2, 0); cell.Style !=
		"text_input.focused_selection" {
		t.Fatalf("selected cell style = %q", cell.Style)
	}
	completion, err := app.DispatchTextInput(
		context.Background(),
		"paste",
		"replace-selection",
		TextInputEvent{Kind: TextInputPaste, Text: "xy"},
	)
	if err != nil {
		t.Fatalf("DispatchTextInput() error = %v", err)
	}
	if completion.Outcome != OutcomeApplied {
		t.Fatalf("paste Outcome = %q", completion.Outcome)
	}
	details = textFieldDetailsByKey(t, app, "selection")
	if details.Text != "aby" || details.Caret != 3 ||
		details.SelectionStart != 3 || details.SelectionEnd != 3 {
		t.Fatalf("hard-filtered replacement = %#v", details)
	}
	dispatchTextChord(t, app, "select-all", KeyControl, "a")
	details = textFieldDetailsByKey(t, app, "selection")
	if details.SelectionStart != 0 || details.SelectionEnd != 3 {
		t.Fatalf("Ctrl-A selection = %#v", details)
	}
	dispatchTextKey(t, app, "delete", KeyBackspace)
	dispatchTextKey(t, app, "commit", KeyEnter)
	if field.Text() != "" {
		t.Fatalf("selection deletion committed %q", field.Text())
	}

	if _, err := app.DispatchTextInput(
		context.Background(),
		"paste",
		"multiline-rejected",
		TextInputEvent{Kind: TextInputPaste, Text: "a\nb"},
	); err != nil {
		t.Fatalf("invalid focused-state paste returned transport error: %v", err)
	}
}
