package expletives

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"
)

func textAreaDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) TextAreaDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.TextArea
	if details == nil {
		t.Fatalf("%s has no TextAreaDetails", key)
	}
	return *details
}

func TestTextAreaNormalizationValidationAndPassword(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 6})
	if _, err := NewTextArea(app.Root(), TextAreaOptions{
		Text: "bad\nline",
		Validator: &TextValidator{
			Enforcement: TextValidationHard,
			Mode:        TextValidationBlacklist,
			Characters:  "d",
		},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("hard-invalid NewTextArea() error = %v", err)
	}
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "area",
			Bounds:        Rect{Width: 8, Height: 3},
		},
		Text: "a\r\nb\rc", Password: true,
		Validator: &TextValidator{
			Enforcement: TextValidationSoft,
			Mode:        TextValidationWhitelist,
			Characters:  "abc",
		},
	})
	if err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	if got := area.Text(); got != "a\nb\nc" {
		t.Fatalf("normalized Text() = %q", got)
	}
	details := textAreaDetailsByKey(t, app, "area")
	if details.Text != "" || !details.Redacted || details.Length != 5 ||
		details.LineCount != 3 || !details.Valid {
		t.Fatalf("password details = %#v", details)
	}
	owned := app.Snapshot()
	ownedDetails := controlByKey(t, owned, "area").Details.TextArea
	ownedDetails.Validator.Characters = "mutated"
	if current := textAreaDetailsByKey(t, app, "area"); current.Validator == nil ||
		current.Validator.Characters != "abc" {
		t.Fatalf("snapshot mutation changed TextArea details to %#v", current)
	}
	snapshot := app.Snapshot()
	for row := 0; row < 3; row++ {
		if cell, _ := snapshot.Frame.Cell(0, row); cell.Grapheme != "*" {
			t.Fatalf("masked row %d cell = %#v", row, cell)
		}
	}
	if strings.Contains(rowText(snapshot, 0), "a") {
		t.Fatal("password frame exposed content")
	}
}

func TestTextAreaEditingNavigationSelectionAndCommit(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 8})
	registerActionCommand(t, app, "area.changed", "Changed", true)
	var calls atomic.Int32
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		if command.ID != "area.changed" {
			t.Fatalf("command ID = %q", command.ID)
		}
		calls.Add(1)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "area",
			Bounds:        Rect{Width: 4, Height: 2},
		},
		Text: "abcd\nefgh\nijkl",
		Wrap: TextWrapCells, ChangeCommand: "area.changed",
	})
	if err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	details := textAreaDetailsByKey(t, app, "area")
	if !details.Editing || details.VisualCaretRow != 2 ||
		details.RowOffset != 1 {
		t.Fatalf("activated details = %#v", details)
	}
	dispatchTextKey(t, app, "up", KeyUp)
	dispatchTextChord(t, app, "select-up", KeyShift, KeyUp)
	details = textAreaDetailsByKey(t, app, "area")
	if details.Caret != 4 || details.SelectionStart != 4 ||
		details.SelectionEnd != 9 {
		t.Fatalf("vertical selection = %#v", details)
	}
	completion, err := app.DispatchTextInput(
		context.Background(),
		"paste",
		"replace",
		TextInputEvent{Kind: TextInputPaste, Text: "X\r\nY"},
	)
	if err != nil {
		t.Fatalf("DispatchTextInput() error = %v", err)
	}
	if completion.Outcome != OutcomeApplied {
		t.Fatalf("paste completion = %#v", completion)
	}
	details = textAreaDetailsByKey(t, app, "area")
	if details.Text != "abcdX\nY\nijkl" ||
		details.SelectionStart != details.SelectionEnd {
		t.Fatalf("pasted details = %#v", details)
	}
	dispatchTextChord(t, app, "commit", KeyControl, KeyEnter)
	if got := area.Text(); got != "abcdX\nY\nijkl" ||
		area.Editing() || calls.Load() != 1 {
		t.Fatalf(
			"commit Text=%q Editing=%t calls=%d",
			got,
			area.Editing(),
			calls.Load(),
		)
	}
}

func TestTextAreaSoftValidationViewportAndCancel(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 6})
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "area",
			Bounds:        Rect{Width: 3, Height: 2},
		},
		Text: "abc\ndef",
		Validator: &TextValidator{
			Enforcement: TextValidationSoft,
			Mode:        TextValidationWhitelist,
			Characters:  "abcdef",
		},
	})
	if err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	completion, err := app.DispatchTextInput(
		context.Background(),
		"ime",
		"invalid",
		TextInputEvent{Kind: TextInputCommitted, Text: "x"},
	)
	if err != nil || completion.Outcome != OutcomeApplied {
		t.Fatalf("committed text completion=%#v error=%v", completion, err)
	}
	details := textAreaDetailsByKey(t, app, "area")
	if details.Valid || details.ColumnOffset == 0 {
		t.Fatalf("invalid/scrolled details = %#v", details)
	}
	snapshot := app.Snapshot()
	foundInvalid := false
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			cell, _ := snapshot.Frame.Cell(x, y)
			if cell.Style == "text_input.invalid_character" {
				foundInvalid = true
			}
		}
	}
	if !foundInvalid {
		t.Fatal("invalid TextArea cell is not visible in the viewport")
	}
	dispatchTextKey(t, app, "cancel", KeyEscape)
	if area.Text() != "abc\ndef" || area.Editing() {
		t.Fatalf("cancel left Text=%q Editing=%t", area.Text(), area.Editing())
	}
}

func TestTextAreaSettersAreAtomic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 6})
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{AutomationKey: "area"},
		Text:         "abc",
	})
	if err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	if err := area.SetWrap(TextWrapWords); err != nil {
		t.Fatalf("SetWrap() error = %v", err)
	}
	if area.Wrap() != TextWrapWords {
		t.Fatalf("Wrap() = %q", area.Wrap())
	}
	if err := area.SetValidator(&TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationWhitelist,
		Characters:  "ab",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("hard-invalid SetValidator() error = %v", err)
	}
	if area.Validator() != nil {
		t.Fatalf("failed SetValidator installed %#v", area.Validator())
	}
	if err := area.SetText("a\x00b"); !errors.Is(err, ErrTextLimit) {
		t.Fatalf("control SetText() error = %v", err)
	}
	if area.Text() != "abc" {
		t.Fatalf("failed SetText changed value to %q", area.Text())
	}
}

func TestCtrlCRemainsConfiguredInterruptWhileTextAreaEdits(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 5})
	if err := app.RegisterCommand(CommandDefinition{
		ID: "app.interrupt", Label: "Interrupt", Enabled: true,
	}); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}
	if err := app.BindChord(
		Chord{Key: "c", Modifiers: []Key{KeyControl}},
		CommandBinding{Command: "app.interrupt"},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "area",
			Bounds:        Rect{Width: 12, Height: 3},
		},
		Text: "unchanged",
	})
	if err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		if command.ID != "app.interrupt" || !area.Editing() {
			t.Fatalf(
				"interrupt command=%q editing=%t",
				command.ID,
				area.Editing(),
			)
		}
		return CommandResult{Outcome: OutcomeInterrupted}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	dispatchTextKey(t, app, "edit", KeyEnter)
	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		"control-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyControl},
	); err != nil {
		t.Fatalf("Ctrl down error = %v", err)
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		"interrupt",
		KeyEvent{Kind: KeyEventPress, Key: "c"},
	)
	if err != nil {
		t.Fatalf("Ctrl-C error = %v", err)
	}
	snapshot := app.Snapshot()
	if completion.Outcome != OutcomeInterrupted ||
		completion.Command != "app.interrupt" ||
		!snapshot.Final ||
		len(snapshot.InputSources) != 0 ||
		area.Text() != "unchanged" {
		t.Fatalf(
			"Ctrl-C completion=%+v final=%t sources=%+v text=%q",
			completion,
			snapshot.Final,
			snapshot.InputSources,
			area.Text(),
		)
	}
}

func FuzzTextAreaNormalizationAndBoundedEditing(f *testing.F) {
	for _, seed := range []string{
		"",
		"plain ASCII",
		"a\r\nb\rc",
		"e\u0301\nz",
		"wide界\nemoji🙂",
		string([]byte{0xff, 'a', '\n'}),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 {
			t.Skip()
		}
		normalized, err := normalizeTextAreaInput(input)
		if err != nil {
			if !errors.Is(err, ErrTextLimit) {
				t.Fatalf("normalizeTextAreaInput(%q) error = %v", input, err)
			}
			return
		}
		if !utf8.ValidString(normalized.text) {
			t.Fatalf("normalized text is invalid UTF-8: %q", normalized.text)
		}
		if strings.Contains(normalized.text, "\r") {
			t.Fatalf("normalized text retained CR: %q", normalized.text)
		}
		if len(normalized.text) > MaxTextInputBytes ||
			len(normalized.cells) > MaxTextInputCells {
			t.Fatalf(
				"normalized bounds bytes=%d cells=%d",
				len(normalized.text),
				len(normalized.cells),
			)
		}
		for _, cell := range normalized.cells {
			if cell == "\n" {
				continue
			}
			if _, err := normalizeInputText(cell); err != nil {
				t.Fatalf("normalized cell %q is not reusable: %v", cell, err)
			}
		}

		app := mustApp(t, Size{Width: 16, Height: 5})
		area, err := NewTextArea(app.Root(), TextAreaOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "area",
				Bounds:        Rect{Width: 8, Height: 3},
			},
			Text: normalized.text,
			Wrap: TextWrapCells,
		})
		if err != nil {
			t.Fatalf("NewTextArea(%q) error = %v", normalized.text, err)
		}
		dispatchTextKey(t, app, "edit", KeyEnter)
		dispatchTextChord(t, app, "select-all", KeyControl, "a")
		completion, err := app.DispatchTextInput(
			context.Background(),
			"fuzz",
			"replace",
			TextInputEvent{Kind: TextInputPaste, Text: input},
		)
		if err != nil {
			t.Fatalf("DispatchTextInput(%q) error = %v", input, err)
		}
		if completion.Outcome != OutcomeApplied &&
			completion.Outcome != OutcomeNoOp {
			t.Fatalf("paste completion = %#v", completion)
		}
		details := textAreaDetailsByKey(t, app, "area")
		if details.Text != normalized.text ||
			details.Caret < 0 || details.Caret > details.Length ||
			details.SelectionStart != details.Caret ||
			details.SelectionEnd != details.Caret {
			t.Fatalf("edited details = %#v, text=%q", details, area.Text())
		}
	})
}
