package automation

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"runtime"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestMaximumModalStackIsValidAndFitsResponseLine(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 80, Height: 24}, Scenario: "modal.maximum",
	})
	if err != nil {
		t.Fatal(err)
	}
	var owner expletives.Modal
	for index := 0; index < expletives.MaxModalDepth; index++ {
		modal, modalErr := expletives.NewModalPanel(
			app.Root(),
			expletives.ModalPanelOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "modal." + string(rune('0'+index)),
					Bounds:        expletives.Rect{Width: 20 + index, Height: 6 + index},
				},
				NestedOwner: owner,
			},
		)
		if modalErr != nil {
			t.Fatalf("NewModalPanel(%d) error = %v", index, modalErr)
		}
		if modalErr := modal.Show(nil); modalErr != nil {
			t.Fatalf("Show(%d) error = %v", index, modalErr)
		}
		owner = modal
	}
	overflow, err := expletives.NewModalPanel(
		app.Root(),
		expletives.ModalPanelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "modal.overflow",
				Bounds:        expletives.Rect{Width: 20, Height: 6},
			},
			NestedOwner: owner,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := overflow.Show(nil); !errors.Is(err, expletives.ErrModalCapacity) {
		t.Fatalf("ninth Show() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	limits := DefaultLimits()
	if err := validateSnapshot(&projected, limits); err != nil {
		t.Fatalf("maximum modal snapshot rejected: %v", err)
	}
	wire := cloneSnapshot(projected)
	compactFrame(&wire.Frame, false)
	completion := Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     "maximum-modal",
		Operation:     TypeObserve,
		Outcome:       OutcomeNoOp,
		FrameSequence: wire.Sequence,
		Snapshot:      &wire,
	}
	if err := validateCompletion(completion, limits); err != nil {
		t.Fatalf("maximum modal completion rejected: %v", err)
	}
	line, err := encodeLine(completion, limits.ResponseLineBytes)
	if err != nil {
		t.Fatalf("maximum modal completion exceeds response line: %v", err)
	}
	t.Logf("maximum modal completion bytes: %d", len(line))
}

func TestMaximumBoundedCompletionFitsResponseLine(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	completion := maximumElementCompletion(limits)

	validationFixture := maximumValidElementCompletion(limits)
	if err := validateCompletion(validationFixture, limits); err != nil {
		t.Fatalf("maximum element fixture is not protocol-valid: %v", err)
	}

	maximumBytes := maximumCompletionJSONBytes(t, completion, limits)
	t.Logf("maximum bounded completion bytes: %d", maximumBytes)
	if maximumBytes > limits.ResponseLineBytes {
		t.Fatalf(
			"maximum bounded completion needs %d JSON bytes; response limit is %d",
			maximumBytes,
			limits.ResponseLineBytes,
		)
	}
	retainedBytes := maximumBytes * limits.RetainedResults
	if retainedBytes > maxRetainedResponseBytes {
		t.Fatalf(
			"%d retained maximum completions need %d JSON bytes; aggregate budget is %d",
			limits.RetainedResults,
			retainedBytes,
			maxRetainedResponseBytes,
		)
	}
	t.Logf(
		"maximum bounded completion: %d JSON bytes under %d-byte response limit; "+
			"%d retained records: %d bytes under %d-byte aggregate budget",
		maximumBytes,
		limits.ResponseLineBytes,
		limits.RetainedResults,
		retainedBytes,
		maxRetainedResponseBytes,
	)
}

func TestReasonableLargeDesktopFrameCompactsAndDecodes(t *testing.T) {
	const dimension = 1200
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: dimension, Height: dimension},
	})
	if err != nil {
		t.Fatalf("NewApp(1200x1200) error = %v", err)
	}
	core := app.Snapshot()
	if len(core.Frame.Cells) != dimension*dimension {
		t.Fatalf("core cell count = %d", len(core.Frame.Cells))
	}
	projected := snapshotFromCore(core)
	compactFrame(&projected.Frame, false)
	core = expletives.Snapshot{}
	app = nil
	runtime.GC()
	if got := len(projected.Frame.Runs); got != 1 {
		t.Fatalf("uniform 1200x1200 frame run count = %d, want 1", got)
	}
	completion := Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     "large-frame",
		Operation:     TypeObserve,
		Outcome:       OutcomeApplied,
		FrameSequence: projected.Sequence,
		Snapshot:      &projected,
	}
	line, err := encodeLine(completion, DefaultLimits().ResponseLineBytes)
	if err != nil {
		t.Fatalf("encode compact 1200x1200 completion: %v", err)
	}
	var decoded Completion
	if err := json.Unmarshal(line, &decoded); err != nil {
		t.Fatalf("decode compact completion: %v", err)
	}
	if err := validateCompletion(decoded, DefaultLimits()); err != nil {
		t.Fatalf("validate compact completion: %v", err)
	}
	if got := len(decoded.Snapshot.Frame.Cells); got != dimension*dimension {
		t.Fatalf("decoded cell count = %d, want %d", got, dimension*dimension)
	}
}

func TestSnapshotRejectsBorderTitleBeyondBound(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	completion := maximumValidElementCompletion(limits)
	completion.Snapshot.Controls[0].Details.Border.Title = strings.Repeat(
		"x",
		maxBorderTitleBytes+1,
	)

	if err := validateCompletion(completion, limits); err == nil {
		t.Fatal("validateCompletion() error = nil for overlong border title")
	}
}

func TestSnapshotRejectsInvalidCanonicalText(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	tests := []struct {
		name   string
		mutate func(*Completion)
	}{
		{
			name: "noncanonical cell",
			mutate: func(completion *Completion) {
				completion.Snapshot.Frame.Cells[0].Grapheme = "\x00"
			},
		},
		{
			name: "title cell count",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title =
					strings.Repeat("x", maxBorderTitleCells+1)
			},
		},
		{
			name: "title cell bytes",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title =
					"x" + strings.Repeat("\u0301", maxCellGraphemeBytes/2)
			},
		},
		{
			name: "noncanonical title",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title = "\x00"
			},
		},
		{
			name: "snapshot completion message NUL",
			mutate: func(completion *Completion) {
				completion.Snapshot.Completion.Message = "invalid\x00message"
			},
		},
		{
			name: "top-level error message NUL",
			mutate: func(completion *Completion) {
				completion.Error.Message = "invalid\x00message"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			completion := maximumValidElementCompletion(limits)
			test.mutate(&completion)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() error = nil for invalid text")
			}
		})
	}
}

func TestSnapshotRejectsInvalidDisplayControlDetails(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	textTests := []struct {
		name   string
		mutate func(*TextDetails)
	}{
		{
			name: "text bytes",
			mutate: func(details *TextDetails) {
				details.Text = strings.Repeat("x", maxDisplayTextBytes+1)
			},
		},
		{
			name: "wide noncanonical text",
			mutate: func(details *TextDetails) {
				details.Text = "界"
			},
		},
		{
			name: "label newline",
			mutate: func(details *TextDetails) {
				details.Text = "first\nsecond"
			},
		},
		{
			name: "alignment",
			mutate: func(details *TextDetails) {
				details.HorizontalAlignment = "middle"
			},
		},
		{
			name: "wrap",
			mutate: func(details *TextDetails) {
				details.Wrap = "words"
			},
		},
		{
			name: "targetless mnemonic",
			mutate: func(details *TextDetails) {
				details.Target = ""
			},
		},
	}
	for _, test := range textTests {
		test := test
		t.Run("text/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := maximumValidTextCompletion(limits)
			test.mutate(completion.Snapshot.Controls[0].Details.Text)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid TextDetails")
			}
		})
	}

	dividerTests := []struct {
		name   string
		mutate func(*DividerDetails)
	}{
		{
			name: "orientation",
			mutate: func(details *DividerDetails) {
				details.Orientation = 99
			},
		},
		{
			name: "form",
			mutate: func(details *DividerDetails) {
				details.Form = "ornate"
			},
		},
		{
			name: "newline",
			mutate: func(details *DividerDetails) {
				details.Text = "bad\nrule"
			},
		},
		{
			name: "alignment",
			mutate: func(details *DividerDetails) {
				details.Alignment = "middle"
			},
		},
	}
	for _, test := range dividerTests {
		test := test
		t.Run("divider/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := maximumValidElementCompletion(limits)
			control := &completion.Snapshot.Controls[0]
			control.Kind = "rule"
			control.Details = ControlDetails{
				Version: 1,
				Divider: &DividerDetails{
					Orientation: 0,
					Form:        "single",
					Text:        "rule",
					Alignment:   "start",
				},
			}
			test.mutate(control.Details.Divider)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid DividerDetails")
			}
		})
	}
}

func TestSnapshotRejectsInvalidActionControlDetails(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	button := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "button"
		control.Details = ControlDetails{
			Version: 1,
			Action: &ActionDetails{
				Label: "Run", Command: "action.run", Enabled: true,
				Mnemonic: "r", Default: true,
			},
		}
		return completion
	}
	buttonTests := []struct {
		name   string
		mutate func(*ActionDetails)
	}{
		{
			name: "wide label",
			mutate: func(action *ActionDetails) {
				action.Label = "界"
			},
		},
		{
			name: "enabled reason",
			mutate: func(action *ActionDetails) {
				action.DisabledReason = "not valid while enabled"
			},
		},
		{
			name: "disabled without reason",
			mutate: func(action *ActionDetails) {
				action.Enabled = false
			},
		},
		{
			name: "pressed disabled",
			mutate: func(action *ActionDetails) {
				action.Enabled = false
				action.DisabledReason = "disabled"
				action.Pressed = true
			},
		},
		{
			name: "two roles",
			mutate: func(action *ActionDetails) {
				action.Cancel = true
			},
		},
		{
			name: "modifier mnemonic",
			mutate: func(action *ActionDetails) {
				action.Mnemonic = "alt"
			},
		},
	}
	for _, test := range buttonTests {
		test := test
		t.Run("button/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := button()
			test.mutate(completion.Snapshot.Controls[0].Details.Action)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid ActionDetails")
			}
		})
	}

	hotkey := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "hotkey_bar"
		control.Details = ControlDetails{
			Version: 1,
			HotkeyBar: &HotkeyBarDetails{
				Items: []HotkeyBarItemDetails{{
					Label: "Run", Command: "action.run", Enabled: true,
					Chord: &Chord{
						Key: "r", Modifiers: []Key{"control"},
					},
				}},
			},
		}
		return completion
	}
	hotkeyTests := []struct {
		name   string
		mutate func(*HotkeyBarDetails)
	}{
		{
			name: "duplicate command",
			mutate: func(bar *HotkeyBarDetails) {
				bar.Items = append(bar.Items, bar.Items[0])
			},
		},
		{
			name: "modifier key",
			mutate: func(bar *HotkeyBarDetails) {
				bar.Items[0].Chord.Key = "control"
			},
		},
		{
			name: "modifier order",
			mutate: func(bar *HotkeyBarDetails) {
				bar.Items[0].Chord.Modifiers = []Key{"shift", "alt"}
			},
		},
		{
			name: "enabled reason",
			mutate: func(bar *HotkeyBarDetails) {
				bar.Items[0].DisabledReason = "invalid"
			},
		},
	}
	for _, test := range hotkeyTests {
		test := test
		t.Run("hotkey/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := hotkey()
			test.mutate(completion.Snapshot.Controls[0].Details.HotkeyBar)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid HotkeyBarDetails")
			}
		})
	}

	t.Run("aggregate items", func(t *testing.T) {
		t.Parallel()
		completion := hotkey()
		bounded := limits
		bounded.Controls = 1
		item := completion.Snapshot.Controls[0].Details.HotkeyBar.Items[0]
		item.Command = "action.second"
		completion.Snapshot.Controls[0].Details.HotkeyBar.Items =
			append(
				completion.Snapshot.Controls[0].Details.HotkeyBar.Items,
				item,
			)
		if err := validateCompletion(completion, bounded); err == nil {
			t.Fatal("validateCompletion() accepted excess Action items")
		}
	})
}

func TestSnapshotRejectsInvalidSelectionControlDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	validateRejected := func(
		t *testing.T,
		kind ControlKind,
		details ControlDetails,
	) {
		t.Helper()
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = kind
		control.Details = details
		if err := validateCompletion(completion, limits); err == nil {
			t.Fatalf("validateCompletion() accepted invalid %s details", kind)
		}
	}

	t.Run("checkbox indeterminate without policy", func(t *testing.T) {
		validateRejected(t, "checkbox", ControlDetails{
			Version: 1,
			Checkbox: &CheckboxDetails{
				Label: "Check", State: "indeterminate", Enabled: true,
			},
		})
	})
	t.Run("radio group duplicate selection", func(t *testing.T) {
		validateRejected(t, "radio_group", ControlDetails{
			Version:   1,
			Container: &ContainerDetails{},
			RadioGroup: &RadioGroupDetails{
				Value: "one", Enabled: true,
				Options: []RadioOptionDetails{
					{
						Control: "radio.one", Value: "one", Label: "One",
						Selected: true, Enabled: true,
					},
					{
						Control: "radio.two", Value: "two", Label: "Two",
						Selected: true, Enabled: true,
					},
				},
			},
		})
	})
	t.Run("choice selected index mismatch", func(t *testing.T) {
		validateRejected(t, "cycle_field", ControlDetails{
			Version: 1,
			ChoiceField: &ChoiceFieldDetails{
				Label: "Choice", Value: "one", SelectedIndex: 1,
				Enabled: true,
				Options: []SelectionOptionDetails{{
					Value: "one", Label: "One", Enabled: true,
					Selected: true,
				}},
			},
		})
	})
	t.Run("choice duplicate value", func(t *testing.T) {
		validateRejected(t, "select_field", ControlDetails{
			Version: 1,
			ChoiceField: &ChoiceFieldDetails{
				Label: "Choice", Value: "one", SelectedIndex: 0,
				Enabled: true,
				Options: []SelectionOptionDetails{
					{
						Value: "one", Label: "One", Enabled: true,
						Selected: true,
					},
					{Value: "one", Label: "Again", Enabled: true},
				},
			},
		})
	})
	t.Run("kind union mismatch", func(t *testing.T) {
		validateRejected(t, "checkbox", ControlDetails{
			Version: 1,
			RadioButton: &RadioButtonDetails{
				Value: "one", Label: "One", Enabled: true,
			},
		})
	})
}

func TestSnapshotRejectsInvalidFocusGuideBarDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "focus_guide_bar"
		control.Details = ControlDetails{
			Version: 1,
			FocusGuideBar: &FocusGuideBarDetails{
				Target:        "button.run",
				TargetKind:    "button",
				Text:          "Button guidance",
				Customization: "append",
			},
		}
		return completion
	}
	if err := validateCompletion(valid(), limits); err != nil {
		t.Fatalf("valid FocusGuideBar fixture rejected: %v", err)
	}
	for _, kind := range []ControlKind{
		"scroll_bar",
		"tabbed_panel",
		"notebook",
		"viewport",
		"scrollable_panel",
		"markdown_view",
		"log_view",
		"stream_view",
	} {
		completion := valid()
		completion.Snapshot.Controls[0].Details.FocusGuideBar.TargetKind = kind
		if err := validateCompletion(completion, limits); err != nil {
			t.Fatalf("valid %s focus target rejected: %v", kind, err)
		}
	}
	tests := map[string]func(*FocusGuideBarDetails){
		"target kind": func(details *FocusGuideBarDetails) {
			details.TargetKind = "panel"
		},
		"customization": func(details *FocusGuideBarDetails) {
			details.Customization = "replace-ish"
		},
		"orphan kind": func(details *FocusGuideBarDetails) {
			details.Target = ""
		},
		"multiline": func(details *FocusGuideBarDetails) {
			details.Text = "first\nsecond"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			completion := valid()
			mutate(completion.Snapshot.Controls[0].Details.FocusGuideBar)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid focus guidance")
			}
		})
	}
}

func TestSnapshotRejectsInvalidTextFieldDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "text_field"
		control.Details = ControlDetails{
			Version: 1,
			TextField: &TextFieldDetails{
				Text: "abc", Length: 3, MaximumBytes: 8,
				Caret: 2, ViewOffset: 0,
				SelectionStart: 2, SelectionEnd: 2,
				Valid: true, Enabled: true,
				FocusedStyle: "text_input.focused",
				EditingStyle: "text_input.focused",
				ByteStyles: []TextFieldByteStyleDetails{
					{MinimumBytes: 0, Style: "text_input.valid"},
				},
				Validator: &TextValidatorDetails{
					Enforcement: "soft",
					Mode:        "whitelist",
					Characters:  "abc",
				},
			},
		}
		return completion
	}
	if err := validateCompletion(valid(), limits); err != nil {
		t.Fatalf("valid TextField fixture rejected: %v", err)
	}
	tests := map[string]func(*TextFieldDetails){
		"length mismatch": func(details *TextFieldDetails) {
			details.Length++
		},
		"missing maximum bytes": func(details *TextFieldDetails) {
			details.MaximumBytes = 0
		},
		"maximum beyond global bound": func(details *TextFieldDetails) {
			details.MaximumBytes = expletives.MaxTextInputBytes + 1
		},
		"text beyond configured maximum": func(details *TextFieldDetails) {
			details.MaximumBytes = 2
		},
		"caret beyond value": func(details *TextFieldDetails) {
			details.Caret = details.Length + 1
		},
		"reversed selection": func(details *TextFieldDetails) {
			details.SelectionStart = 2
			details.SelectionEnd = 1
		},
		"invalid validity": func(details *TextFieldDetails) {
			details.Text = "abx"
			details.Valid = true
		},
		"hard invalid": func(details *TextFieldDetails) {
			details.Text = "abx"
			details.Valid = false
			details.Validator.Enforcement = "hard"
		},
		"redaction mismatch": func(details *TextFieldDetails) {
			details.Password = true
			details.Redacted = true
		},
		"validator redacted without field": func(details *TextFieldDetails) {
			details.Validator.Characters = ""
			details.Validator.CharactersRedacted = true
		},
		"redacted validator retains characters": func(details *TextFieldDetails) {
			details.Text = ""
			details.Redacted = true
			details.Validator.CharactersRedacted = true
		},
		"duplicate validator": func(details *TextFieldDetails) {
			details.Validator.Characters = "aabc"
		},
		"disabled editing": func(details *TextFieldDetails) {
			details.Enabled = false
			details.DisabledReason = "Disabled"
			details.Editing = true
		},
		"missing focused style": func(details *TextFieldDetails) {
			details.FocusedStyle = ""
		},
		"missing editing style": func(details *TextFieldDetails) {
			details.EditingStyle = ""
		},
		"missing byte styles": func(details *TextFieldDetails) {
			details.ByteStyles = nil
		},
		"decreasing byte threshold": func(details *TextFieldDetails) {
			details.ByteStyles = append(
				details.ByteStyles,
				TextFieldByteStyleDetails{
					MinimumBytes: 0,
					Style:        "text_input.invalid",
				},
			)
		},
		"byte threshold beyond limit": func(details *TextFieldDetails) {
			details.ByteStyles[0].MinimumBytes = expletives.MaxTextInputBytes + 1
		},
		"invalid edit command": func(details *TextFieldDetails) {
			details.EditCommand = "bad command"
		},
		"invalid submit command": func(details *TextFieldDetails) {
			details.SubmitCommand = "bad command"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			completion := valid()
			mutate(completion.Snapshot.Controls[0].Details.TextField)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid TextField details")
			}
		})
	}

	password := valid()
	details := password.Snapshot.Controls[0].Details.TextField
	details.Text = ""
	details.Password = true
	details.Redacted = true
	if err := validateCompletion(password, limits); err != nil {
		t.Fatalf("valid redacted TextField fixture rejected: %v", err)
	}

	compoundSensitive := valid()
	details = compoundSensitive.Snapshot.Controls[0].Details.TextField
	details.Text = ""
	details.Redacted = true
	details.Validator.Characters = ""
	details.Validator.CharactersRedacted = true
	if err := validateCompletion(compoundSensitive, limits); err != nil {
		t.Fatalf("valid compound-sensitive TextField fixture rejected: %v", err)
	}
}

func TestSnapshotRejectsInvalidTextAreaDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "text_area"
		control.Bounds = Rect{Width: 8, Height: 3}
		control.AbsoluteBounds = control.Bounds
		control.Details = ControlDetails{
			Version: 1,
			TextArea: &TextAreaDetails{
				Text: "abc\ndef", Length: 7, LineCount: 2,
				Caret: 7, SelectionStart: 7, SelectionEnd: 7,
				VisualCaretRow: 1, VisualCaretColumn: 3,
				Wrap: "none", Valid: true, Enabled: true,
				Validator: &TextValidatorDetails{
					Enforcement: "soft",
					Mode:        "whitelist",
					Characters:  "abcdef",
				},
			},
		}
		return completion
	}
	if err := validateCompletion(valid(), limits); err != nil {
		t.Fatalf("valid TextArea fixture rejected: %v", err)
	}
	tests := map[string]func(*TextAreaDetails){
		"length mismatch": func(details *TextAreaDetails) {
			details.Length++
		},
		"line mismatch": func(details *TextAreaDetails) {
			details.LineCount++
		},
		"reversed selection": func(details *TextAreaDetails) {
			details.SelectionStart = 5
			details.SelectionEnd = 4
		},
		"wrapped column offset": func(details *TextAreaDetails) {
			details.Wrap = "cells"
			details.ColumnOffset = 1
		},
		"invalid validity": func(details *TextAreaDetails) {
			details.Text = "abc\nx"
			details.Length = 5
			details.Caret = 5
			details.SelectionStart = 5
			details.SelectionEnd = 5
			details.VisualCaretColumn = 1
		},
		"redaction mismatch": func(details *TextAreaDetails) {
			details.Password = true
			details.Redacted = true
		},
		"disabled editing": func(details *TextAreaDetails) {
			details.Enabled = false
			details.DisabledReason = "Disabled"
			details.Editing = true
		},
		"read-only editing": func(details *TextAreaDetails) {
			details.ReadOnly = true
			details.Editing = true
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			completion := valid()
			mutate(completion.Snapshot.Controls[0].Details.TextArea)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid TextArea details")
			}
		})
	}
	password := valid()
	details := password.Snapshot.Controls[0].Details.TextArea
	details.Text = ""
	details.Password = true
	details.Redacted = true
	if err := validateCompletion(password, limits); err != nil {
		t.Fatalf("valid redacted TextArea fixture rejected: %v", err)
	}
}

func TestSnapshotRejectsInvalidProgressDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func(kind ControlKind) Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = kind
		control.Bounds = Rect{Width: 8, Height: 1}
		control.AbsoluteBounds = control.Bounds
		details := &ProgressDetails{
			Status: "running", Orientation: 0,
		}
		switch kind {
		case "progress_bar":
			details.Current = 1
			details.Total = 2
			details.TextMode = "percentage"
		case "meter":
			details.Value = 50
			details.Maximum = 100
		case "spinner":
			details.Indeterminate = true
			details.Tick = 3
			details.TextMode = "none"
			details.FrameIndex = 3
		case "activity_dots":
			details.Indeterminate = true
			details.Tick = 7
			details.TextMode = "none"
			details.FrameIndex = 2
		}
		control.Details = ControlDetails{
			Version:  1,
			Progress: details,
		}
		return completion
	}
	for _, kind := range []ControlKind{
		"progress_bar", "meter", "spinner", "activity_dots",
	} {
		if err := validateCompletion(valid(kind), limits); err != nil {
			t.Fatalf("valid %s fixture rejected: %v", kind, err)
		}
	}
	tests := []struct {
		name   string
		kind   ControlKind
		mutate func(*ProgressDetails)
	}{
		{
			name: "status", kind: "progress_bar",
			mutate: func(details *ProgressDetails) {
				details.Status = "almost"
			},
		},
		{
			name: "bar range", kind: "progress_bar",
			mutate: func(details *ProgressDetails) {
				details.Current = 3
			},
		},
		{
			name: "bar frame", kind: "progress_bar",
			mutate: func(details *ProgressDetails) {
				details.FrameIndex = 1
			},
		},
		{
			name: "meter finite", kind: "meter",
			mutate: func(details *ProgressDetails) {
				details.Value = math.NaN()
			},
		},
		{
			name: "meter range", kind: "meter",
			mutate: func(details *ProgressDetails) {
				details.Minimum = 100
			},
		},
		{
			name: "spinner frame", kind: "spinner",
			mutate: func(details *ProgressDetails) {
				details.FrameIndex = 2
			},
		},
		{
			name: "dots frozen tick", kind: "activity_dots",
			mutate: func(details *ProgressDetails) {
				details.ReducedMotion = true
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			completion := valid(test.kind)
			test.mutate(completion.Snapshot.Controls[0].Details.Progress)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatalf("validateCompletion() accepted invalid %s", test.kind)
			}
		})
	}
}

func TestSnapshotRejectsInvalidScrollBarDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "scroll_bar"
		control.Bounds = Rect{Width: 12, Height: 1}
		control.AbsoluteBounds = control.Bounds
		control.Details = ControlDetails{
			Version: 1,
			ScrollBar: &ScrollBarDetails{
				Orientation:   0,
				ContentSize:   100,
				ViewportSize:  20,
				Offset:        40,
				MaximumOffset: 80,
				ArrowStep:     1,
				PageStep:      20,
				TrackStart:    1,
				TrackSize:     10,
				ThumbStart:    5,
				ThumbSize:     2,
				Enabled:       true,
			},
		}
		return completion
	}
	if err := validateCompletion(valid(), limits); err != nil {
		t.Fatalf("valid ScrollBar fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ScrollBarDetails)
	}{
		{
			name: "orientation",
			mutate: func(details *ScrollBarDetails) {
				details.Orientation = 9
			},
		},
		{
			name: "negative content",
			mutate: func(details *ScrollBarDetails) {
				details.ContentSize = -1
			},
		},
		{
			name: "maximum",
			mutate: func(details *ScrollBarDetails) {
				details.MaximumOffset = 81
			},
		},
		{
			name: "offset",
			mutate: func(details *ScrollBarDetails) {
				details.Offset = 81
			},
		},
		{
			name: "step",
			mutate: func(details *ScrollBarDetails) {
				details.PageStep = 0
			},
		},
		{
			name: "geometry",
			mutate: func(details *ScrollBarDetails) {
				details.ThumbStart = 6
			},
		},
		{
			name: "enabled reason",
			mutate: func(details *ScrollBarDetails) {
				details.DisabledReason = "not disabled"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			completion := valid()
			test.mutate(
				completion.Snapshot.Controls[0].Details.ScrollBar,
			)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid ScrollBar details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidScrollableDetailsAndRelationships(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 30, Height: 12},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewScrollablePanel(
			app.Root(),
			expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{
					PanelOptions: expletives.PanelOptions{
						AutomationKey: "scroll",
						Bounds: expletives.Rect{
							Width: 10, Height: 6,
						},
					},
					State: expletives.ViewportState{
						ContentSize: expletives.Size{
							Width: 20, Height: 10,
						},
						Offset: expletives.Point{X: 2, Y: 3},
					},
				},
				BorderForm: expletives.BorderSingle,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(
		snapshot *SnapshotV1,
	) (*ControlSnapshot, *ControlSnapshot, *ScrollableDetails) {
		var owner, content *ControlSnapshot
		for index := range snapshot.Controls {
			switch snapshot.Controls[index].Key {
			case "scroll":
				owner = &snapshot.Controls[index]
			case "scroll.content":
				content = &snapshot.Controls[index]
			}
		}
		if owner == nil || content == nil ||
			owner.Details.Scrollable == nil {
			t.Fatal("fixture has no scroll owner or Content")
		}
		return owner, content, owner.Details.Scrollable
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid ScrollablePanel fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(
			*SnapshotV1,
			*ControlSnapshot,
			*ControlSnapshot,
			*ScrollableDetails,
		)
	}{
		{
			name: "maximum offset",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				_ *ControlSnapshot,
				details *ScrollableDetails,
			) {
				details.MaximumOffset.X++
			},
		},
		{
			name: "viewport outside owner",
			mutate: func(
				_ *SnapshotV1,
				owner *ControlSnapshot,
				_ *ControlSnapshot,
				details *ScrollableDetails,
			) {
				details.ViewportBounds.Width = owner.Bounds.Width
			},
		},
		{
			name: "visibility without bar",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				_ *ControlSnapshot,
				details *ScrollableDetails,
			) {
				details.HorizontalBar = nil
			},
		},
		{
			name: "nested bar state",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				_ *ControlSnapshot,
				details *ScrollableDetails,
			) {
				details.VerticalBar.Offset++
			},
		},
		{
			name: "invalid policy",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				_ *ControlSnapshot,
				details *ScrollableDetails,
			) {
				details.HorizontalPolicy = "sometimes"
			},
		},
		{
			name: "Content parent",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				content *ControlSnapshot,
				_ *ScrollableDetails,
			) {
				content.Parent = ""
			},
		},
		{
			name: "Content bounds",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				content *ControlSnapshot,
				_ *ScrollableDetails,
			) {
				content.Bounds.X++
			},
		},
		{
			name: "Content key",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				content *ControlSnapshot,
				_ *ScrollableDetails,
			) {
				content.Key = "wrong.content"
			},
		},
		{
			name: "owner child list",
			mutate: func(
				_ *SnapshotV1,
				owner *ControlSnapshot,
				_ *ControlSnapshot,
				_ *ScrollableDetails,
			) {
				owner.Children = nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			owner, content, details := find(&snapshot)
			test.mutate(&snapshot, owner, content, details)
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid Scrollable details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidMarkdownDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 30, Height: 12},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewMarkdownView(
			app.Root(),
			expletives.MarkdownViewOptions{
				ScrollablePanelOptions: expletives.ScrollablePanelOptions{
					ScrollViewOptions: expletives.ScrollViewOptions{
						PanelOptions: expletives.PanelOptions{
							AutomationKey: "markdown",
							Bounds:        expletives.Rect{Width: 14, Height: 6},
						},
					},
					BorderForm:    expletives.BorderSingle,
					HorizontalBar: expletives.ScrollBarVisibilityAuto,
					VerticalBar:   expletives.ScrollBarVisibilityAuto,
				},
				Markdown: "# Title\n\nparagraph words that wrap\n\n---\n\n" +
					"```\n0123456789abcdefghijklmnop\n```",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) (*ControlSnapshot, *MarkdownDetails) {
		for index := range snapshot.Controls {
			control := &snapshot.Controls[index]
			if control.Key == "markdown" && control.Details.Markdown != nil {
				return control, control.Details.Markdown
			}
		}
		t.Fatal("fixture has no Markdown details")
		return nil, nil
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid Markdown fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot, *MarkdownDetails)
	}{
		{
			name: "source bound",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.SourceBytes = expletives.MaxContentBytes + 1
			},
		},
		{
			name: "content metrics",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Viewport.State.ContentSize.Height++
			},
		},
		{
			name: "block kind",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Blocks[0].Kind = "script"
			},
		},
		{
			name: "block ordering",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Blocks[1].SourceLine = details.Blocks[0].SourceLine
			},
		},
		{
			name: "summary truncation",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.SummariesTruncated = false
			},
		},
		{
			name: "maximum offset",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Viewport.MaximumOffset.X++
			},
		},
		{
			name: "viewport geometry",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Viewport.ViewportBounds.X++
			},
		},
		{
			name: "visibility",
			mutate: func(_ *ControlSnapshot, details *MarkdownDetails) {
				details.Viewport.HorizontalVisible = false
			},
		},
		{
			name: "unsupported title",
			mutate: func(control *ControlSnapshot, _ *MarkdownDetails) {
				control.Details.Border.Title = "Title"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			control, details := find(&snapshot)
			test.mutate(control, details)
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid Markdown details")
			}
		})
	}

	aggregateApp, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index <
		expletives.MaxContentAggregateBytes/expletives.MaxContentBytes+1; index++ {
		if _, err := expletives.NewMarkdownView(
			aggregateApp.Root(),
			expletives.MarkdownViewOptions{
				ScrollablePanelOptions: expletives.ScrollablePanelOptions{
					ScrollViewOptions: expletives.ScrollViewOptions{
						PanelOptions: expletives.PanelOptions{
							AutomationKey: "markdown.aggregate." +
								string(rune('a'+index)),
							Bounds: expletives.Rect{Width: 10, Height: 4},
						},
					},
				},
				Markdown: "x",
			},
		); err != nil {
			t.Fatalf("aggregate fixture control %d: %v", index, err)
		}
	}
	aggregate := snapshotFromCore(aggregateApp.Snapshot())
	for index := range aggregate.Controls {
		if details := aggregate.Controls[index].Details.Markdown; details != nil {
			details.SourceBytes = expletives.MaxContentBytes
		}
	}
	if err := validateSnapshot(&aggregate, limits); err == nil {
		t.Fatal("validateSnapshot() accepted aggregate Markdown overflow")
	}
}

func TestSnapshotRejectsInvalidLogAndStreamDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 40, Height: 12},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewLogView(app.Root(), expletives.LogViewOptions{
			ScrollablePanelOptions: expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{
					PanelOptions: expletives.PanelOptions{
						AutomationKey: "log",
						Bounds:        expletives.Rect{Width: 18, Height: 6},
					},
				},
				BorderForm:    expletives.BorderSingle,
				HorizontalBar: expletives.ScrollBarVisibilityAuto,
				VerticalBar:   expletives.ScrollBarVisibilityAuto,
			},
			Capacity: expletives.ContentCapacity{Records: 2, Bytes: 64},
			Records: []expletives.LogRecord{
				{Key: "a", Level: expletives.LogInfo, Text: "alpha"},
				{Key: "b", Level: expletives.LogInfo, Text: "beta"},
				{Key: "c", Level: expletives.LogInfo, Text: "gamma"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		stream, err := expletives.NewStreamView(
			app.Root(),
			expletives.StreamViewOptions{
				ScrollablePanelOptions: expletives.ScrollablePanelOptions{
					ScrollViewOptions: expletives.ScrollViewOptions{
						PanelOptions: expletives.PanelOptions{
							AutomationKey: "stream",
							Bounds: expletives.Rect{
								X: 20, Width: 18, Height: 6,
							},
						},
					},
					BorderForm:    expletives.BorderSingle,
					HorizontalBar: expletives.ScrollBarVisibilityAuto,
					VerticalBar:   expletives.ScrollBarVisibilityAuto,
				},
				Capacity: expletives.ContentCapacity{Records: 2, Bytes: 12},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := stream.Append(
			context.Background(),
			[]byte("abcdefghijklmnopqrst"),
		); err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) (*ControlSnapshot, *ControlSnapshot) {
		var logControl *ControlSnapshot
		var streamControl *ControlSnapshot
		for index := range snapshot.Controls {
			switch snapshot.Controls[index].Key {
			case "log":
				logControl = &snapshot.Controls[index]
			case "stream":
				streamControl = &snapshot.Controls[index]
			}
		}
		if logControl == nil || streamControl == nil {
			t.Fatal("fixture has no LogView or StreamView details")
		}
		return logControl, streamControl
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid content fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot, *ControlSnapshot)
	}{
		{
			name: "log capacity",
			mutate: func(logControl, _ *ControlSnapshot) {
				logControl.Details.LogView.Capacity.Records = 0
			},
		},
		{
			name: "log retained bytes",
			mutate: func(logControl, _ *ControlSnapshot) {
				logControl.Details.LogView.RetainedBytes = 65
			},
		},
		{
			name: "log retained key",
			mutate: func(logControl, _ *ControlSnapshot) {
				logControl.Details.LogView.FirstKey = ""
			},
		},
		{
			name: "log viewport rows",
			mutate: func(logControl, _ *ControlSnapshot) {
				logControl.Details.LogView.Viewport.State.ContentSize.Height = 1
			},
		},
		{
			name: "stream pending storage",
			mutate: func(_, streamControl *ControlSnapshot) {
				streamControl.Details.StreamView.PendingStorageBytes = 13
			},
		},
		{
			name: "stream pending cells",
			mutate: func(_, streamControl *ControlSnapshot) {
				streamControl.Details.StreamView.PendingCells = 0
			},
		},
		{
			name: "stream drop consistency",
			mutate: func(_, streamControl *ControlSnapshot) {
				streamControl.Details.StreamView.DroppedLines = 0
			},
		},
		{
			name: "stream viewport rows",
			mutate: func(_, streamControl *ControlSnapshot) {
				streamControl.Details.StreamView.Viewport.State.ContentSize.Height++
			},
		},
		{
			name: "detail union",
			mutate: func(logControl, streamControl *ControlSnapshot) {
				logControl.Details.StreamView = streamControl.Details.StreamView
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			logControl, streamControl := find(&snapshot)
			test.mutate(logControl, streamControl)
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid content details")
			}
		})
	}

	aggregateApp, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 20, Height: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index <
		expletives.MaxContentAggregateBytes/expletives.MaxContentBytes+1; index++ {
		if _, err := expletives.NewLogView(
			aggregateApp.Root(),
			expletives.LogViewOptions{
				ScrollablePanelOptions: expletives.ScrollablePanelOptions{
					ScrollViewOptions: expletives.ScrollViewOptions{
						PanelOptions: expletives.PanelOptions{
							AutomationKey: "log.aggregate." +
								string(rune('a'+index)),
							Bounds: expletives.Rect{Width: 10, Height: 4},
						},
					},
				},
				Records: []expletives.LogRecord{{
					Key: "record", Level: expletives.LogInfo, Text: "x",
				}},
			},
		); err != nil {
			t.Fatalf("aggregate fixture control %d: %v", index, err)
		}
	}
	aggregate := snapshotFromCore(aggregateApp.Snapshot())
	for index := range aggregate.Controls {
		if details := aggregate.Controls[index].Details.LogView; details != nil {
			details.RetainedBytes = expletives.MaxContentBytes
		}
	}
	if err := validateSnapshot(&aggregate, limits); err == nil {
		t.Fatal("validateSnapshot() accepted aggregate LogView overflow")
	}
}

func TestSnapshotRejectsInvalidListBoxDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 24, Height: 8},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewListBox(app.Root(), expletives.ListBoxOptions{
			ScrollablePanelOptions: expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{
					PanelOptions: expletives.PanelOptions{
						AutomationKey: "list",
						Bounds:        expletives.Rect{Width: 15, Height: 4},
					},
				},
				BorderForm:  expletives.BorderSingle,
				VerticalBar: expletives.ScrollBarVisibilityAuto,
			},
			Items: []expletives.ListItem{
				{Key: "one", Label: "One"},
				{Key: "two", Label: "Two"},
				{Key: "three", Label: "Three"},
			},
			SelectionMode:    expletives.CollectionSelectionMultiple,
			RequireSelection: true,
			Selected:         []string{"one", "three"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "list" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no ListBox details")
		return nil
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid ListBox fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"status", func(control *ControlSnapshot) {
			control.Details.ListBox.Status = "unknown"
		}},
		{"ready message", func(control *ControlSnapshot) {
			control.Details.ListBox.StatusMessageBytes = 1
		}},
		{"item count", func(control *ControlSnapshot) {
			control.Details.ListBox.ItemCount = expletives.MaxCollectionItems + 1
		}},
		{"enabled count", func(control *ControlSnapshot) {
			control.Details.ListBox.EnabledCount = 4
		}},
		{"visual row count", func(control *ControlSnapshot) {
			control.Details.ListBox.VisualRowCount = -1
		}},
		{"wrap", func(control *ControlSnapshot) {
			control.Details.ListBox.Wrap = "paragraphs"
		}},
		{"retained bytes", func(control *ControlSnapshot) {
			control.Details.ListBox.RetainedBytes = -1
		}},
		{"current index", func(control *ControlSnapshot) {
			control.Details.ListBox.CurrentIndex = 3
		}},
		{"selection mode", func(control *ControlSnapshot) {
			control.Details.ListBox.SelectionMode = "range"
		}},
		{"required selection", func(control *ControlSnapshot) {
			control.Details.ListBox.SelectedCount = 0
			control.Details.ListBox.FirstSelected = ""
			control.Details.ListBox.LastSelected = ""
			control.Details.ListBox.SelectionDigest =
				"e3b0c44298fc1c149afbf4c8996fb924" +
					"27ae41e4649b934ca495991b7852b855"
		}},
		{"selection endpoints", func(control *ControlSnapshot) {
			control.Details.ListBox.LastSelected = "one"
		}},
		{"selection digest", func(control *ControlSnapshot) {
			control.Details.ListBox.SelectionDigest = strings.Repeat("G", 64)
		}},
		{"viewport state", func(control *ControlSnapshot) {
			control.Details.ListBox.Viewport.State.ContentSize.Height++
		}},
		{"viewport policy", func(control *ControlSnapshot) {
			control.Details.ListBox.Viewport.HorizontalPolicy = "sometimes"
		}},
		{"disabled reason", func(control *ControlSnapshot) {
			control.Details.ListBox.Enabled = false
		}},
		{"detail union", func(control *ControlSnapshot) {
			control.Details.StreamView = &StreamViewDetails{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			test.mutate(find(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid ListBox details")
			}
		})
	}

	aggregate := valid()
	first := find(&aggregate)
	copyControl := *first
	copyControl.ID = "list-copy-id"
	copyControl.Key = "list-copy"
	copyControl.AbsoluteBounds.X = 16
	copyControl.Details.ListBox.RetainedBytes =
		expletives.MaxCollectionAggregateBytes/2 + 1
	first.Details.ListBox.RetainedBytes =
		expletives.MaxCollectionAggregateBytes/2 + 1
	aggregate.Controls = append(aggregate.Controls, copyControl)
	root := &aggregate.Controls[0]
	root.Children = append(root.Children, copyControl.ID)
	if err := validateSnapshot(&aggregate, limits); err == nil {
		t.Fatal("validateSnapshot() accepted excessive aggregate collection bytes")
	}
}

func TestSnapshotRejectsInvalidTreeViewDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 28, Height: 9},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewTreeView(app.Root(), expletives.TreeViewOptions{
			ScrollablePanelOptions: expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{
					PanelOptions: expletives.PanelOptions{
						AutomationKey: "tree",
						Bounds:        expletives.Rect{Width: 20, Height: 5},
					},
				},
				BorderForm: expletives.BorderSingle,
			},
			Nodes: []expletives.TreeNode{{
				Key: "root", Label: "Root", Expanded: true,
				Children: []expletives.TreeNode{
					{Key: "one", Label: "One"},
					{Key: "two", Label: "Two"},
				},
			}},
			SelectionMode:    expletives.CollectionSelectionMultiple,
			RequireSelection: true,
			Selected:         []string{"root", "two"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "tree" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no TreeView details")
		return nil
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid TreeView fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"status", func(control *ControlSnapshot) {
			control.Details.TreeView.Status = "unknown"
		}},
		{"node count", func(control *ControlSnapshot) {
			control.Details.TreeView.NodeCount = expletives.MaxCollectionItems + 1
		}},
		{"visible count", func(control *ControlSnapshot) {
			control.Details.TreeView.VisibleCount = 4
		}},
		{"enabled count", func(control *ControlSnapshot) {
			control.Details.TreeView.EnabledCount = 4
		}},
		{"retained bytes", func(control *ControlSnapshot) {
			control.Details.TreeView.RetainedBytes = -1
		}},
		{"current index", func(control *ControlSnapshot) {
			control.Details.TreeView.CurrentIndex = 3
		}},
		{"selection mode", func(control *ControlSnapshot) {
			control.Details.TreeView.SelectionMode = "range"
		}},
		{"selection endpoints", func(control *ControlSnapshot) {
			control.Details.TreeView.LastSelected = "root"
		}},
		{"expansion endpoints", func(control *ControlSnapshot) {
			control.Details.TreeView.FirstExpanded = "missing"
		}},
		{"expansion digest", func(control *ControlSnapshot) {
			control.Details.TreeView.ExpansionDigest = strings.Repeat("G", 64)
		}},
		{"viewport state", func(control *ControlSnapshot) {
			control.Details.TreeView.Viewport.State.ContentSize.Height++
		}},
		{"invalid command", func(control *ControlSnapshot) {
			control.Details.TreeView.ExpandCommand = "bad command"
		}},
		{"detail union", func(control *ControlSnapshot) {
			control.Details.ListBox = &ListBoxDetails{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			test.mutate(find(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid TreeView details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidTableDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 36, Height: 10},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewTable(app.Root(), expletives.TableOptions{
			ScrollablePanelOptions: expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
					AutomationKey: "table", Bounds: expletives.Rect{Width: 28, Height: 6},
				}},
				BorderForm: expletives.BorderSingle,
			},
			Columns: []expletives.Column{
				{Key: "name", Header: "Name", Sortable: true},
				{Key: "state", Header: "State"},
			},
			Rows: []expletives.TableRow{
				{Key: "one", Cells: []expletives.TableCell{{Column: "name", Text: "One"}}},
				{Key: "two", Cells: []expletives.TableCell{{Column: "name", Text: "Two"}}},
			},
			SelectionMode: expletives.CollectionSelectionMultiple,
			Selected:      []string{"one", "two"}, RequireSelection: true,
			FocusMode: expletives.TableFocusCell,
		})
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "table" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no Table details")
		return nil
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid Table fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"status", func(control *ControlSnapshot) { control.Details.Table.Status = "unknown" }},
		{"row count", func(control *ControlSnapshot) { control.Details.Table.RowCount = expletives.MaxCollectionItems + 1 }},
		{"visual row count", func(control *ControlSnapshot) {
			control.Details.Table.VisualRowCount = expletives.MaxFrameCells
		}},
		{"column count", func(control *ControlSnapshot) {
			control.Details.Table.ColumnCount = expletives.MaxCollectionColumns + 1
		}},
		{"visible column count", func(control *ControlSnapshot) {
			control.Details.Table.VisibleColumnCount = control.Details.Table.ColumnCount + 1
		}},
		{"visible column range", func(control *ControlSnapshot) {
			control.Details.Table.FirstVisibleColumn = ""
		}},
		{"cell count", func(control *ControlSnapshot) { control.Details.Table.CellCount = expletives.MaxCollectionCells + 1 }},
		{"current row", func(control *ControlSnapshot) { control.Details.Table.CurrentRowIndex = 2 }},
		{"current column", func(control *ControlSnapshot) { control.Details.Table.CurrentColumnIndex = 2 }},
		{"focus mode", func(control *ControlSnapshot) { control.Details.Table.FocusMode = "header" }},
		{"selection style", func(control *ControlSnapshot) { control.Details.Table.SelectionStyle = "free" }},
		{"multiple range endpoint", func(control *ControlSnapshot) { control.Details.Table.RangeAnchor = "one" }},
		{"selection digest", func(control *ControlSnapshot) { control.Details.Table.SelectionDigest = strings.Repeat("G", 64) }},
		{"presentation digest", func(control *ControlSnapshot) {
			control.Details.Table.PresentationDigest = strings.Repeat("G", 64)
		}},
		{"width digest", func(control *ControlSnapshot) { control.Details.Table.ColumnWidthsDigest = strings.Repeat("G", 64) }},
		{"sort implication", func(control *ControlSnapshot) { control.Details.Table.SortColumn = "name" }},
		{"viewport state", func(control *ControlSnapshot) { control.Details.Table.Viewport.State.ContentSize.Height++ }},
		{"invalid command", func(control *ControlSnapshot) { control.Details.Table.SortCommand = "bad command" }},
		{"detail union", func(control *ControlSnapshot) { control.Details.TreeView = &TreeViewDetails{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			test.mutate(find(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid Table details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidDataGridDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 36, Height: 10},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
			ScrollablePanelOptions: expletives.ScrollablePanelOptions{
				ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
					AutomationKey: "data-grid", Bounds: expletives.Rect{Width: 28, Height: 6},
				}},
				BorderForm: expletives.BorderSingle,
			},
			Columns: []expletives.Column{{Key: "value", Header: "Value", Editable: true}},
			Rows: []expletives.TableRow{{
				Key: "row", Cells: []expletives.TableCell{{Column: "value", Text: "One"}},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "data-grid" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no DataGrid details")
		return nil
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid DataGrid fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"focus mode", func(control *ControlSnapshot) {
			control.Details.DataGrid.Table.FocusMode = "row"
		}},
		{"idle edit row", func(control *ControlSnapshot) {
			control.Details.DataGrid.EditRow = "row"
		}},
		{"editor without editing", func(control *ControlSnapshot) {
			control.Details.DataGrid.EditLength = 1
		}},
		{"detail union", func(control *ControlSnapshot) {
			control.Details.Table = &TableDetails{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			test.mutate(find(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid DataGrid details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidModalPanelDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 36, Height: 12},
		})
		if err != nil {
			t.Fatal(err)
		}
		outside, err := expletives.NewTextField(
			app.Root(),
			expletives.TextFieldOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "outside", Bounds: expletives.Rect{Width: 10, Height: 1},
			}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := outside.Focus(); err != nil {
			t.Fatal(err)
		}
		modal, err := expletives.NewModalPanel(
			app.Root(),
			expletives.ModalPanelOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "modal", Bounds: expletives.Rect{Width: 20, Height: 7},
				},
				Title: "Modal",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		inside, err := expletives.NewTextField(
			modal,
			expletives.TextFieldOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "inside",
				Bounds:        expletives.Rect{X: 1, Y: 1, Width: 10, Height: 1},
			}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := modal.Show(inside); err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	find := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "modal" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no ModalPanel details")
		return nil
	}
	snapshot := valid()
	modal := find(&snapshot)
	if modal.Details.ModalPanel == nil ||
		modal.Details.ModalPanel.SavedFocus == "" ||
		modal.Details.ModalPanel.InitialFocus == "" {
		t.Fatalf("projected ModalPanel details = %+v", modal.Details.ModalPanel)
	}
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid ModalPanel fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"lifecycle", func(control *ControlSnapshot) {
			control.Details.ModalPanel.Lifecycle = "unknown"
		}},
		{"visibility", func(control *ControlSnapshot) { control.Visible = false }},
		{"stack index", func(control *ControlSnapshot) {
			control.Details.ModalPanel.StackIndex = 1
		}},
		{"stack depth", func(control *ControlSnapshot) {
			control.Details.ModalPanel.StackDepth = 2
		}},
		{"base owner", func(control *ControlSnapshot) {
			control.Details.ModalPanel.NestedOwner = "control-1"
		}},
		{"shadow", func(control *ControlSnapshot) {
			control.Details.ModalPanel.Shadow = "unknown"
		}},
		{"active result", func(control *ControlSnapshot) {
			control.Details.ModalPanel.Result = &ModalResultDetails{Reason: "accepted"}
		}},
		{"resolved geometry", func(control *ControlSnapshot) {
			control.Details.ModalPanel.ResolvedBounds.Width++
		}},
		{"detail union", func(control *ControlSnapshot) {
			control.Details.DataGrid = &DataGridDetails{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			test.mutate(find(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid ModalPanel details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidPopupCollectionDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	validDropDown := func(open bool) SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 24, Height: 8},
		})
		if err != nil {
			t.Fatal(err)
		}
		dropDown, err := expletives.NewDropDown(
			app.Root(),
			expletives.DropDownOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "drop",
					Bounds: expletives.Rect{
						X: 10, Y: 5, Width: 12, Height: 1,
					},
				},
				Items: []expletives.ListItem{
					{Key: "one", Label: "One"},
					{Key: "two", Label: "Two"},
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if open {
			if err := dropDown.Open(); err != nil {
				t.Fatal(err)
			}
		}
		return snapshotFromCore(app.Snapshot())
	}
	findDropDown := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "drop" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no DropDown details")
		return nil
	}
	if snapshot := validDropDown(false); validateSnapshot(&snapshot, limits) != nil {
		t.Fatal("valid closed DropDown fixture rejected")
	}
	if snapshot := validDropDown(true); validateSnapshot(&snapshot, limits) != nil {
		t.Fatal("valid open DropDown fixture rejected")
	}
	tests := []struct {
		name   string
		open   bool
		mutate func(*ControlSnapshot)
	}{
		{"item count", false, func(control *ControlSnapshot) {
			control.Details.DropDown.ItemCount = expletives.MaxCollectionItems + 1
		}},
		{"enabled count", false, func(control *ControlSnapshot) {
			control.Details.DropDown.EnabledCount = 3
		}},
		{"retained bytes", false, func(control *ControlSnapshot) {
			control.Details.DropDown.RetainedBytes = -1
		}},
		{"current index", false, func(control *ControlSnapshot) {
			control.Details.DropDown.CurrentIndex = 2
		}},
		{"required selection", false, func(control *ControlSnapshot) {
			control.Details.DropDown.Selected = ""
			control.Details.DropDown.SelectedIndex = -1
		}},
		{"popup rows", false, func(control *ControlSnapshot) {
			control.Details.DropDown.PopupRows = expletives.MaxCollectionPopupRows + 1
		}},
		{"closed transient state", false, func(control *ControlSnapshot) {
			control.Details.DropDown.PopupCurrent = "one"
		}},
		{"open bounds", true, func(control *ControlSnapshot) {
			control.Details.DropDown.PopupBounds.Width = 0
		}},
		{"open disabled", true, func(control *ControlSnapshot) {
			control.Details.DropDown.Enabled = false
			control.Details.DropDown.DisabledReasonBytes = 1
		}},
		{"invalid command", false, func(control *ControlSnapshot) {
			control.Details.DropDown.ChangeCommand = "bad command"
		}},
		{"detail union", false, func(control *ControlSnapshot) {
			control.Details.ListBox = &ListBoxDetails{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validDropDown(test.open)
			test.mutate(findDropDown(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid DropDown details")
			}
		})
	}

	validCombo := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 24, Height: 8},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = expletives.NewComboBox(app.Root(), expletives.ComboBoxOptions{
			DropDownOptions: expletives.DropDownOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "combo",
					Bounds:        expletives.Rect{Width: 12, Height: 1},
				},
				Items: []expletives.ListItem{{Key: "one", Label: "One"}},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	findCombo := func(snapshot *SnapshotV1) *ControlSnapshot {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "combo" {
				return &snapshot.Controls[index]
			}
		}
		t.Fatal("fixture has no ComboBox details")
		return nil
	}
	comboTests := []struct {
		name   string
		mutate func(*ControlSnapshot)
	}{
		{"editor enabled mismatch", func(control *ControlSnapshot) {
			control.Details.ComboBox.Editor.Enabled = false
			control.Details.ComboBox.Editor.DisabledReason = "disabled"
		}},
		{"editor command mismatch", func(control *ControlSnapshot) {
			control.Details.ComboBox.Editor.ChangeCommand = "different"
		}},
		{"password editor", func(control *ControlSnapshot) {
			control.Details.ComboBox.Editor.Password = true
			control.Details.ComboBox.Editor.Redacted = true
			control.Details.ComboBox.Editor.Text = ""
		}},
		{"popup union", func(control *ControlSnapshot) {
			popup := control.Details.ComboBox.Popup
			control.Details.DropDown = &popup
		}},
	}
	for _, test := range comboTests {
		t.Run("combo "+test.name, func(t *testing.T) {
			snapshot := validCombo()
			if err := validateSnapshot(&snapshot, limits); err != nil {
				t.Fatalf("valid ComboBox fixture rejected: %v", err)
			}
			test.mutate(findCombo(&snapshot))
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid ComboBox details")
			}
		})
	}

	aggregate := validDropDown(false)
	first := findDropDown(&aggregate)
	copyControl := *first
	copyControl.ID = "drop-copy-id"
	copyControl.Key = "drop-copy"
	copyControl.AbsoluteBounds.X = 12
	copyControl.Details.DropDown.RetainedBytes =
		expletives.MaxCollectionAggregateBytes/2 + 1
	first.Details.DropDown.RetainedBytes =
		expletives.MaxCollectionAggregateBytes/2 + 1
	aggregate.Controls = append(aggregate.Controls, copyControl)
	root := &aggregate.Controls[0]
	root.Children = append(root.Children, copyControl.ID)
	if err := validateSnapshot(&aggregate, limits); err == nil {
		t.Fatal("validateSnapshot() accepted popup collection aggregate overflow")
	}
}

func TestSnapshotRejectsInvalidTabbedPanelDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() SnapshotV1 {
		app, err := expletives.NewApp(expletives.AppOptions{
			Size: expletives.Size{Width: 30, Height: 10},
		})
		if err != nil {
			t.Fatal(err)
		}
		transaction := app.NewTransaction()
		panel, err := transaction.NewNotebook(
			app.Root(),
			expletives.TabbedPanelOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "tabs",
					Bounds: expletives.Rect{
						Width: 20, Height: 8,
					},
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		first, _ := transaction.NewPanel(
			panel,
			expletives.PanelOptions{AutomationKey: "page.first"},
		)
		second, _ := transaction.NewPanel(
			panel,
			expletives.PanelOptions{AutomationKey: "page.second"},
		)
		if err := transaction.SetTabs(
			panel,
			[]expletives.Tab{
				{
					Key: "first", Value: "first", Label: "First",
					Mnemonic: "f", Page: first,
				},
				{
					Key: "second", Value: "second", Label: "Second",
					Mnemonic: "s", Page: second,
				},
			},
			"first",
		); err != nil {
			t.Fatal(err)
		}
		if err := transaction.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		return snapshotFromCore(app.Snapshot())
	}
	snapshot := valid()
	if err := validateSnapshot(&snapshot, limits); err != nil {
		t.Fatalf("valid TabbedPanel fixture rejected: %v", err)
	}
	find := func(snapshot *SnapshotV1) (*ControlSnapshot, *TabbedPanelDetails) {
		for index := range snapshot.Controls {
			if snapshot.Controls[index].Key == "tabs" {
				return &snapshot.Controls[index],
					snapshot.Controls[index].Details.TabbedPanel
			}
		}
		t.Fatal("fixture has no tabs control")
		return nil, nil
	}
	tests := []struct {
		name   string
		mutate func(*SnapshotV1, *ControlSnapshot, *TabbedPanelDetails)
	}{
		{
			name: "selected",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				details.Selected = "missing"
			},
		},
		{
			name: "duplicate key",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				details.Tabs[1].Key = details.Tabs[0].Key
			},
		},
		{
			name: "mnemonic",
			mutate: func(
				_ *SnapshotV1,
				_ *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				details.Tabs[0].Mnemonic = "z"
			},
		},
		{
			name: "bounds",
			mutate: func(
				_ *SnapshotV1,
				control *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				details.Tabs[0].Bounds.Width = control.Bounds.Width + 1
			},
		},
		{
			name: "page relationship",
			mutate: func(
				snapshot *SnapshotV1,
				_ *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				for index := range snapshot.Controls {
					if snapshot.Controls[index].ID ==
						details.Tabs[0].Page {
						snapshot.Controls[index].Parent = ""
					}
				}
			},
		},
		{
			name: "nonselected visible",
			mutate: func(
				snapshot *SnapshotV1,
				_ *ControlSnapshot,
				details *TabbedPanelDetails,
			) {
				for index := range snapshot.Controls {
					if snapshot.Controls[index].ID ==
						details.Tabs[1].Page {
						snapshot.Controls[index].Visible = true
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid()
			control, details := find(&snapshot)
			test.mutate(&snapshot, control, details)
			if err := validateSnapshot(&snapshot, limits); err == nil {
				t.Fatal("validateSnapshot() accepted invalid TabbedPanel details")
			}
		})
	}
}

func TestSnapshotRejectsInvalidNumberFieldDetails(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	valid := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "number_field"
		control.Details = ControlDetails{
			Version: 1,
			NumberField: &NumberFieldDetails{
				Text: "1.5", Value: 1.5, Length: 3, Caret: 3,
				SelectionStart: 3, SelectionEnd: 3,
				Valid: true, Minimum: float64Pointer(0),
				Maximum: float64Pointer(2), DecimalPlaces: 1,
				Enabled: true,
			},
		}
		return completion
	}
	if err := validateCompletion(valid(), limits); err != nil {
		t.Fatalf("valid NumberField fixture rejected: %v", err)
	}
	tests := map[string]func(*NumberFieldDetails){
		"length mismatch": func(details *NumberFieldDetails) {
			details.Length++
		},
		"nonfinite value": func(details *NumberFieldDetails) {
			details.Value = math.NaN()
		},
		"invalid precision": func(details *NumberFieldDetails) {
			details.DecimalPlaces = expletives.MaxNumberDecimalPlaces + 1
		},
		"reversed range": func(details *NumberFieldDetails) {
			*details.Minimum = 3
		},
		"noncanonical committed text": func(details *NumberFieldDetails) {
			details.Text = "01.5"
			details.Length = 4
			details.Caret = 4
		},
		"unexpected step": func(details *NumberFieldDetails) {
			details.Step = 0.5
		},
		"invalid reason mismatch": func(details *NumberFieldDetails) {
			details.Editing = true
			details.Text = "-"
			details.Length = 1
			details.Caret = 1
			details.Valid = false
		},
		"disabled editing": func(details *NumberFieldDetails) {
			details.Enabled = false
			details.DisabledReason = "Disabled"
			details.Editing = true
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			completion := valid()
			mutate(completion.Snapshot.Controls[0].Details.NumberField)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid numeric details")
			}
		})
	}

	spin := valid()
	spin.Snapshot.Controls[0].Kind = "spin_box"
	spin.Snapshot.Controls[0].Details.NumberField.Step = 0.5
	if err := validateCompletion(spin, limits); err != nil {
		t.Fatalf("valid SpinBox fixture rejected: %v", err)
	}
	spin.Snapshot.Controls[0].Details.NumberField.Step = 0
	if err := validateCompletion(spin, limits); err == nil {
		t.Fatal("validateCompletion() accepted zero-step SpinBox")
	}
}

func TestSnapshotRejectsInvalidMenuBarDetails(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	menuBar := func() Completion {
		completion := maximumValidElementCompletion(limits)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "menu_bar"
		control.Details = ControlDetails{
			Version: 1,
			MenuBar: &MenuBarDetails{
				Entries: []MenuEntryDetails{
					{
						Key: "menu.file", Kind: "submenu", Label: "File",
						Enabled: true, Mnemonic: "f", Selected: true,
						Open: true, ChildCount: 2,
					},
					{
						Key: "item.open", ParentKey: "menu.file", Depth: 1,
						Kind: "command", Label: "Open",
						Command: "action.open", Enabled: true, Selected: true,
						Mnemonic: "o",
						Chord: &Chord{
							Key: "o", Modifiers: []Key{"control"},
						},
					},
					{
						Key: "item.separator", ParentKey: "menu.file",
						Depth: 1, Kind: "separator",
					},
				},
				OpenPath:     []string{"menu.file"},
				SelectedPath: []string{"menu.file", "item.open"},
			},
		}
		return completion
	}
	if err := validateCompletion(menuBar(), limits); err != nil {
		t.Fatalf("valid MenuBar fixture rejected: %v", err)
	}
	barActive := menuBar()
	activeDetails := barActive.Snapshot.Controls[0].Details.MenuBar
	activeDetails.OpenPath = nil
	activeDetails.SelectedPath = []string{"menu.file"}
	activeDetails.Entries[0].Open = false
	activeDetails.Entries[1].Selected = false
	if err := validateCompletion(barActive, limits); err != nil {
		t.Fatalf("valid bar-active MenuBar fixture rejected: %v", err)
	}
	disabledSelected := menuBar()
	disabledEntry := &disabledSelected.Snapshot.Controls[0].
		Details.MenuBar.Entries[1]
	disabledEntry.Enabled = false
	disabledEntry.DisabledReason = "Unavailable"
	if err := validateCompletion(disabledSelected, limits); err != nil {
		t.Fatalf("valid selected-disabled MenuBar fixture rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*MenuBarDetails)
	}{
		{
			name: "duplicate key",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[2].Key = "item.open"
			},
		},
		{
			name: "wrong parent",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[1].ParentKey = "missing"
			},
		},
		{
			name: "wrong depth",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[1].Depth = 2
			},
		},
		{
			name: "invalid root placement",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[0].Placement = "middle"
			},
		},
		{
			name: "nested placement",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[1].Placement = "end"
			},
		},
		{
			name: "child count",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[0].ChildCount = 1
			},
		},
		{
			name: "selected path",
			mutate: func(bar *MenuBarDetails) {
				bar.SelectedPath[1] = "item.separator"
			},
		},
		{
			name: "selected unopened descendant",
			mutate: func(bar *MenuBarDetails) {
				bar.OpenPath = nil
				bar.Entries[0].Open = false
			},
		},
		{
			name: "separator state",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[2].Enabled = true
			},
		},
		{
			name: "command disabled reason",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[1].Enabled = false
			},
		},
		{
			name: "duplicate sibling mnemonic",
			mutate: func(bar *MenuBarDetails) {
				bar.Entries[2].Kind = "command"
				bar.Entries[2].Label = "Other"
				bar.Entries[2].Command = "action.other"
				bar.Entries[2].Enabled = true
				bar.Entries[2].Mnemonic = "o"
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			completion := menuBar()
			test.mutate(completion.Snapshot.Controls[0].Details.MenuBar)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid MenuBarDetails")
			}
		})
	}
}

func TestSnapshotRejectsInvalidStatusBarDetails(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	statusBar := func() Completion {
		completion := maximumValidElementCompletion(limits)
		cell := completion.Snapshot.Frame.Cells[0]
		completion.Snapshot.Frame.Size = Size{Width: 19, Height: 1}
		completion.Snapshot.Frame.Cells = make([]Cell, 19)
		for index := range completion.Snapshot.Frame.Cells {
			completion.Snapshot.Frame.Cells[index] = cell
		}
		compactFrame(&completion.Snapshot.Frame, true)
		control := &completion.Snapshot.Controls[0]
		control.Kind = "status_bar"
		control.Bounds = Rect{Width: 19, Height: 1}
		control.AbsoluteBounds = control.Bounds
		control.Details = ControlDetails{
			Version: 1,
			StatusBar: &StatusBarDetails{
				Segments: []StatusSegmentDetails{
					{
						Key: "status.context", Label: "Ready",
						Priority: 5, Enabled: true, Rendered: true,
						Bounds: Rect{Width: 7, Height: 1},
					},
					{
						Key: "status.exit", Label: "Exit",
						Command: "app.exit", Priority: 10, Enabled: true,
						Chord: &Chord{
							Key: "x", Modifiers: []Key{"alt"},
						},
						Rendered: true,
						Bounds:   Rect{X: 7, Width: 12, Height: 1},
					},
					{
						Key: "status.omitted", Label: "Omitted",
						Priority: -1, Enabled: true,
					},
				},
			},
		}
		return completion
	}
	if err := validateCompletion(statusBar(), limits); err != nil {
		t.Fatalf("valid StatusBar fixture rejected: %v", err)
	}
	disabled := statusBar()
	disabledSegment := &disabled.Snapshot.Controls[0].
		Details.StatusBar.Segments[1]
	disabledSegment.Enabled = false
	disabledSegment.DisabledReason = "Unavailable"
	if err := validateCompletion(disabled, limits); err != nil {
		t.Fatalf("valid disabled StatusBar fixture rejected: %v", err)
	}
	clipped := statusBar()
	clippedSegment := &clipped.Snapshot.Controls[0].
		Details.StatusBar.Segments[1]
	clippedSegment.Clipped = true
	if err := validateCompletion(clipped, limits); err != nil {
		t.Fatalf("valid clipped StatusBar fixture rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*StatusBarDetails)
	}{
		{
			name: "duplicate key",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].Key = bar.Segments[0].Key
			},
		},
		{
			name: "empty label",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[0].Label = ""
			},
		},
		{
			name: "invalid command",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].Command = "bad command"
			},
		},
		{
			name: "enabled reason",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].DisabledReason = "Contradiction"
			},
		},
		{
			name: "disabled without reason",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].Enabled = false
			},
		},
		{
			name: "static command state",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[0].Checked = true
			},
		},
		{
			name: "omitted bounds",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[2].Bounds.Width = 1
			},
		},
		{
			name: "rendered empty",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[0].Bounds.Width = 0
			},
		},
		{
			name: "render gap",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].Bounds.X++
			},
		},
		{
			name: "invalid chord",
			mutate: func(bar *StatusBarDetails) {
				bar.Segments[1].Chord.Modifiers = []Key{"alt", "alt"}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			completion := statusBar()
			test.mutate(completion.Snapshot.Controls[0].Details.StatusBar)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal(
					"validateCompletion() accepted invalid StatusBarDetails",
				)
			}
		})
	}

	t.Run("multiple bars", func(t *testing.T) {
		t.Parallel()
		completion := statusBar()
		duplicate := completion.Snapshot.Controls[0]
		duplicate.ID = "control-second"
		duplicate.Key = "status.second"
		completion.Snapshot.Controls = append(
			completion.Snapshot.Controls,
			duplicate,
		)
		if err := validateCompletion(completion, limits); err == nil {
			t.Fatal("validateCompletion() accepted multiple StatusBars")
		}
	})
}

func TestSnapshotRejectsAggregateChildReferencesBeyondBound(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	limits.Controls = 1
	completion := maximumValidElementCompletion(limits)
	completion.Snapshot.Controls[0].Children = append(
		completion.Snapshot.Controls[0].Children,
		ControlID("second"),
	)

	if err := validateCompletion(completion, limits); err == nil {
		t.Fatal("validateCompletion() error = nil for excessive child references")
	}
}

// maximumCompletionJSONBytes computes the exact encoded size of a completion
// containing the maximum number of copies of each bounded element. The run
// and title strings deliberately use the maximum six-byte JSON expansion for
// every permitted input byte. That is a conservative overestimate for
// canonical one-cell text, and therefore remains an upper bound.
// Every repeated array already has one element in completion, so each
// additional element contributes one comma plus its independently marshaled
// size. The aggregate child-reference validation permits at most one such
// maximal one-child control per control slot.
func maximumCompletionJSONBytes(
	t *testing.T,
	completion Completion,
	limits Limits,
) int {
	t.Helper()

	wireCompletion := completion
	wireSnapshot := cloneSnapshot(*completion.Snapshot)
	compactFrame(&wireSnapshot.Frame, false)
	wireCompletion.Snapshot = &wireSnapshot
	base := mustMarshal(t, wireCompletion)
	run := mustMarshal(t, completion.Snapshot.Frame.Runs[0])
	controlValue := completion.Snapshot.Controls[0]
	control := mustMarshal(t, controlValue)
	baseControlBytes := len(control)
	borderControl := controlValue
	borderControl.Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: controlValue.Bounds.X,
			InputScope:  "escaping",
		},
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          "single",
			Style:         controlValue.Style,
			ResolvedStyle: controlValue.ResolvedStyle,
		},
	}
	dividerControl := controlValue
	dividerControl.Details = ControlDetails{
		Version: 1,
		Divider: &DividerDetails{
			Orientation: Orientation(math.MaxUint8),
			Form:        string(controlValue.ID),
			Text:        strings.Repeat("\x00", maxDisplayTextBytes),
			Alignment:   TextAlignment(controlValue.ID),
		},
	}
	actionControl := controlValue
	actionControl.Details = ControlDetails{
		Version: 1,
		Action: &ActionDetails{
			Label:          strings.Repeat("\x00", maxDisplayTextBytes),
			Command:        string(controlValue.ID),
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			Checked:        true,
			Mnemonic:       Key(controlValue.ID),
			Pressed:        true,
			Default:        true,
			Cancel:         true,
		},
	}
	hotkeyControl := controlValue
	hotkeyControl.Details = ControlDetails{
		Version: 1,
		HotkeyBar: &HotkeyBarDetails{
			Items: []HotkeyBarItemDetails{{
				Label:          strings.Repeat("\x00", maxDisplayTextBytes),
				Command:        string(controlValue.ID),
				Enabled:        false,
				DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
				Checked:        true,
				Chord: &Chord{
					Key: Key(controlValue.ID),
					Modifiers: []Key{
						Key(controlValue.ID),
						Key(controlValue.ID),
						Key(controlValue.ID),
						Key(controlValue.ID),
					},
				},
			}},
		},
	}
	statusSegment := StatusSegmentDetails{
		Key:            string(controlValue.ID),
		Label:          strings.Repeat("\x00", maxDisplayTextBytes),
		Command:        string(controlValue.ID),
		Priority:       math.MaxInt,
		Enabled:        false,
		DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
		Checked:        true,
		Chord: &Chord{
			Key: Key(controlValue.ID),
			Modifiers: []Key{
				Key(controlValue.ID),
				Key(controlValue.ID),
				Key(controlValue.ID),
				Key(controlValue.ID),
			},
		},
		Rendered: true,
		Bounds: Rect{
			X:      math.MaxInt,
			Y:      math.MaxInt,
			Width:  math.MaxInt,
			Height: math.MaxInt,
		},
		Clipped: true,
	}
	statusControl := controlValue
	statusControl.Details = ControlDetails{
		Version: 1,
		StatusBar: &StatusBarDetails{
			Segments: []StatusSegmentDetails{statusSegment},
		},
	}
	menuEntry := MenuEntryDetails{
		Key:            string(controlValue.ID),
		ParentKey:      string(controlValue.ID),
		Depth:          math.MaxInt,
		Kind:           MenuItemKind(controlValue.ID),
		Label:          strings.Repeat("\x00", maxDisplayTextBytes),
		Command:        string(controlValue.ID),
		Enabled:        false,
		DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
		Checked:        true,
		Mnemonic:       Key(controlValue.ID),
		Placement:      MenuBarPlacement(controlValue.ID),
		Chord: &Chord{
			Key: Key(controlValue.ID),
			Modifiers: []Key{
				Key(controlValue.ID),
				Key(controlValue.ID),
				Key(controlValue.ID),
				Key(controlValue.ID),
			},
		},
		Selected:   true,
		Open:       true,
		ChildCount: math.MaxInt,
	}
	menuControl := controlValue
	menuControl.Details = ControlDetails{
		Version: 1,
		MenuBar: &MenuBarDetails{
			Entries: []MenuEntryDetails{menuEntry},
			OpenPath: []string{
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
			},
			SelectedPath: []string{
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID), string(controlValue.ID),
				string(controlValue.ID),
			},
		},
	}
	selectionOption := SelectionOptionDetails{
		Value:          string(controlValue.ID),
		Label:          strings.Repeat("\x00", maxDisplayTextBytes),
		Enabled:        false,
		DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
		Selected:       true,
	}
	choiceControl := controlValue
	choiceControl.Details = ControlDetails{
		Version: 1,
		ChoiceField: &ChoiceFieldDetails{
			Label:          strings.Repeat("\x00", maxDisplayTextBytes),
			Value:          string(controlValue.ID),
			SelectedIndex:  math.MaxInt,
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			Mnemonic:       Key(controlValue.ID),
			ChangeCommand:  string(controlValue.ID),
			Options:        []SelectionOptionDetails{},
		},
	}
	radioOption := RadioOptionDetails{
		Control:        controlValue.ID,
		Value:          string(controlValue.ID),
		Label:          strings.Repeat("\x00", maxDisplayTextBytes),
		Selected:       true,
		Enabled:        false,
		DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
	}
	radioControl := controlValue
	radioControl.Details = ControlDetails{
		Version:   1,
		Container: &ContainerDetails{InputScope: "escaping"},
		RadioGroup: &RadioGroupDetails{
			Value:          string(controlValue.ID),
			AllowEmpty:     true,
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			ChangeCommand:  string(controlValue.ID),
			Options:        []RadioOptionDetails{},
		},
	}
	focusGuideControl := controlValue
	focusGuideControl.Details = ControlDetails{
		Version: 1,
		FocusGuideBar: &FocusGuideBarDetails{
			Target:        controlValue.ID,
			TargetKind:    ControlKind(controlValue.ID),
			Text:          strings.Repeat("\x00", maxDisplayTextBytes),
			Customization: string(controlValue.ID),
		},
	}
	textFieldControl := controlValue
	textFieldControl.Details = ControlDetails{
		Version: 1,
		TextField: &TextFieldDetails{
			Length: math.MaxInt, MaximumBytes: math.MaxInt,
			Caret:      math.MaxInt,
			ViewOffset: math.MaxInt, Editing: true,
			Valid: true, Password: true, Redacted: true,
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			ChangeCommand:  string(controlValue.ID),
			EditCommand:    string(controlValue.ID),
			SubmitCommand:  string(controlValue.ID),
			FocusedStyle:   string(controlValue.ID),
			EditingStyle:   string(controlValue.ID),
			ByteStyles: []TextFieldByteStyleDetails{{
				MinimumBytes: math.MaxInt,
				Style:        string(controlValue.ID),
			}},
			Validator: &TextValidatorDetails{
				Enforcement: string(controlValue.ID),
				Mode:        string(controlValue.ID),
			},
		},
	}
	numberFieldControl := controlValue
	numberFieldControl.Details = ControlDetails{
		Version: 1,
		NumberField: &NumberFieldDetails{
			Value: math.MaxFloat64, Length: math.MaxInt,
			Caret: math.MaxInt, ViewOffset: math.MaxInt,
			Editing: true, Valid: false,
			InvalidReason: strings.Repeat("<", maxDisplayTextBytes),
			Minimum:       float64Pointer(-math.MaxFloat64),
			Maximum:       float64Pointer(math.MaxFloat64),
			DecimalPlaces: math.MaxInt,
			Step:          math.MaxFloat64,
			Enabled:       true,
			ChangeCommand: string(controlValue.ID),
		},
	}
	textAreaControl := controlValue
	textAreaControl.Details = ControlDetails{
		Version: 1,
		TextArea: &TextAreaDetails{
			Length: math.MaxInt, LineCount: math.MaxInt,
			Caret:          math.MaxInt,
			SelectionStart: math.MaxInt, SelectionEnd: math.MaxInt,
			VisualCaretRow: math.MaxInt, VisualCaretColumn: math.MaxInt,
			RowOffset: math.MaxInt, ColumnOffset: math.MaxInt,
			Wrap:    TextWrap(controlValue.ID),
			Editing: true, Valid: true, Password: true, Redacted: true,
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			ChangeCommand:  string(controlValue.ID),
			Validator: &TextValidatorDetails{
				Enforcement: string(controlValue.ID),
				Mode:        string(controlValue.ID),
			},
		},
	}
	progressControl := controlValue
	progressControl.Details = ControlDetails{
		Version: 1,
		Progress: &ProgressDetails{
			Status:        string(controlValue.ID),
			Current:       ^uint64(0),
			Total:         ^uint64(0),
			Value:         math.MaxFloat64,
			Minimum:       -math.MaxFloat64,
			Maximum:       math.MaxFloat64,
			Orientation:   Orientation(255),
			Indeterminate: true,
			Tick:          ^uint64(0),
			ReducedMotion: true,
			TextMode:      string(controlValue.ID),
			FrameIndex:    math.MaxInt,
		},
	}
	maximumScrollBar := &ScrollBarDetails{
		Orientation:   Orientation(255),
		ContentSize:   math.MaxInt,
		ViewportSize:  math.MaxInt,
		Offset:        math.MaxInt,
		MaximumOffset: math.MaxInt,
		ArrowStep:     math.MaxInt,
		PageStep:      math.MaxInt,
		TrackStart:    math.MaxInt,
		TrackSize:     math.MaxInt,
		ThumbStart:    math.MaxInt,
		ThumbSize:     math.MaxInt,
		Enabled:       false,
	}
	scrollableControl := controlValue
	scrollableControl.Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: math.MaxInt,
			InputScope:  "escaping",
		},
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          string(controlValue.ID),
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		Scrollable: &ScrollableDetails{
			State: ViewportState{
				ContentSize: Size{Width: math.MaxInt, Height: math.MaxInt},
				Offset:      Point{X: math.MaxInt, Y: math.MaxInt},
			},
			MaximumOffset: Point{X: math.MaxInt, Y: math.MaxInt},
			ViewportBounds: Rect{
				X: math.MaxInt, Y: math.MaxInt,
				Width: math.MaxInt, Height: math.MaxInt,
			},
			ArrowStep:         Size{Width: math.MaxInt, Height: math.MaxInt},
			PageStep:          Size{Width: math.MaxInt, Height: math.MaxInt},
			DisabledReason:    strings.Repeat("\x00", maxDisplayTextBytes),
			ChangeCommand:     string(controlValue.ID),
			Content:           controlValue.ID,
			ContentKey:        string(controlValue.ID),
			HorizontalPolicy:  string(controlValue.ID),
			VerticalPolicy:    string(controlValue.ID),
			HorizontalVisible: true,
			VerticalVisible:   true,
			HorizontalBar:     maximumScrollBar,
			VerticalBar:       maximumScrollBar,
		},
	}
	markdownControl := controlValue
	markdownBlocks := make(
		[]MarkdownBlockDetails,
		expletives.MaxMarkdownSummaries,
	)
	for index := range markdownBlocks {
		markdownBlocks[index] = MarkdownBlockDetails{
			Kind:          "paragraph",
			SourceLine:    math.MaxInt,
			SourceLines:   math.MaxInt,
			RenderedStart: math.MaxInt,
			RenderedRows:  math.MaxInt,
		}
	}
	markdownViewport := MarkdownViewportDetails{
		State: ViewportState{
			ContentSize: Size{Width: math.MaxInt, Height: math.MaxInt},
			Offset:      Point{X: math.MaxInt, Y: math.MaxInt},
		},
		MaximumOffset: Point{X: math.MaxInt, Y: math.MaxInt},
		ViewportBounds: Rect{
			X: math.MaxInt, Y: math.MaxInt,
			Width: math.MaxInt, Height: math.MaxInt,
		},
		HorizontalPolicy:  string(controlValue.ID),
		VerticalPolicy:    string(controlValue.ID),
		HorizontalVisible: true,
		VerticalVisible:   true,
	}
	markdownControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          string(controlValue.ID),
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		Markdown: &MarkdownDetails{
			SourceBytes:        math.MaxInt,
			SourceCells:        math.MaxInt,
			BlockCount:         math.MaxInt,
			RenderedRows:       math.MaxInt,
			MaximumLineWidth:   math.MaxInt,
			Viewport:           markdownViewport,
			Blocks:             markdownBlocks,
			SummariesTruncated: true,
		},
	}
	logControl := controlValue
	logControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          string(controlValue.ID),
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		LogView: &LogViewDetails{
			Capacity:        ContentCapacity{Records: math.MaxInt, Bytes: math.MaxInt},
			RetainedRecords: math.MaxInt,
			RetainedBytes:   math.MaxInt,
			DroppedRecords:  math.MaxUint64,
			DroppedBytes:    math.MaxUint64,
			FirstKey:        string(controlValue.ID),
			LastKey:         string(controlValue.ID),
			Follow:          true,
			Viewport:        markdownViewport,
		},
	}
	streamControl := controlValue
	streamControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          string(controlValue.ID),
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		StreamView: &StreamViewDetails{
			Capacity:            ContentCapacity{Records: math.MaxInt, Bytes: math.MaxInt},
			RetainedLines:       math.MaxInt,
			RetainedBytes:       math.MaxInt,
			PendingBytes:        math.MaxInt,
			PendingCells:        math.MaxInt,
			PendingStorageBytes: math.MaxInt,
			PendingTruncated:    true,
			DroppedLines:        math.MaxUint64,
			DroppedBytes:        math.MaxUint64,
			Follow:              true,
			Viewport:            markdownViewport,
		},
	}
	listBoxControl := controlValue
	listBoxControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Form:          "single",
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		ListBox: &ListBoxDetails{
			Status:              "error",
			StatusMessageBytes:  math.MaxInt,
			StatusMessageDigest: strings.Repeat("f", sha256HexBytes),
			ItemCount:           math.MaxInt,
			EnabledCount:        math.MaxInt,
			RetainedBytes:       math.MaxInt,
			Current:             string(controlValue.ID),
			CurrentIndex:        math.MaxInt,
			SelectionMode:       "multiple",
			RequireSelection:    true,
			SelectedCount:       math.MaxInt,
			FirstSelected:       string(controlValue.ID),
			LastSelected:        string(controlValue.ID),
			SelectionDigest:     strings.Repeat("f", sha256HexBytes),
			DisabledReasonBytes: math.MaxInt,
			ChangeCommand:       string(controlValue.ID),
			CurrentCommand:      string(controlValue.ID),
			ActivateCommand:     string(controlValue.ID),
			Viewport:            markdownViewport,
		},
	}
	treeViewControl := controlValue
	treeViewControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Form:          "single",
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		TreeView: &TreeViewDetails{
			Status:              "error",
			StatusMessageBytes:  math.MaxInt,
			StatusMessageDigest: strings.Repeat("f", sha256HexBytes),
			NodeCount:           math.MaxInt,
			VisibleCount:        math.MaxInt,
			EnabledCount:        math.MaxInt,
			RetainedBytes:       math.MaxInt,
			Current:             string(controlValue.ID),
			CurrentIndex:        math.MaxInt,
			SelectionMode:       "multiple",
			RequireSelection:    true,
			SelectedCount:       math.MaxInt,
			FirstSelected:       string(controlValue.ID),
			LastSelected:        string(controlValue.ID),
			SelectionDigest:     strings.Repeat("f", sha256HexBytes),
			ExpandedCount:       math.MaxInt,
			FirstExpanded:       string(controlValue.ID),
			LastExpanded:        string(controlValue.ID),
			ExpansionDigest:     strings.Repeat("f", sha256HexBytes),
			DisabledReasonBytes: math.MaxInt,
			ChangeCommand:       string(controlValue.ID),
			ActivateCommand:     string(controlValue.ID),
			ExpandCommand:       string(controlValue.ID),
			Viewport:            markdownViewport,
		},
	}
	tableControl := controlValue
	tableControl.Details = ControlDetails{
		Version: 1,
		Border: &BorderDetails{
			Form:          "single",
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		Table: &TableDetails{
			Status:              "error",
			StatusMessageBytes:  math.MaxInt,
			StatusMessageDigest: strings.Repeat("f", sha256HexBytes),
			RowCount:            math.MaxInt,
			VisualRowCount:      math.MaxInt,
			EnabledCount:        math.MaxInt,
			ColumnCount:         math.MaxInt,
			CellCount:           math.MaxInt,
			RetainedBytes:       math.MaxInt,
			CurrentRow:          string(controlValue.ID),
			CurrentRowIndex:     math.MaxInt,
			CurrentColumn:       string(controlValue.ID),
			CurrentColumnIndex:  math.MaxInt,
			FocusMode:           "cell",
			SelectionMode:       "multiple",
			RequireSelection:    true,
			SelectedCount:       math.MaxInt,
			FirstSelected:       string(controlValue.ID),
			LastSelected:        string(controlValue.ID),
			SelectionDigest:     strings.Repeat("f", sha256HexBytes),
			SortColumn:          string(controlValue.ID),
			SortDirection:       "descending",
			FirstColumn:         string(controlValue.ID),
			LastColumn:          string(controlValue.ID),
			VisibleColumnCount:  math.MaxInt,
			FirstVisibleColumn:  string(controlValue.ID),
			LastVisibleColumn:   string(controlValue.ID),
			PresentationDigest:  strings.Repeat("f", sha256HexBytes),
			ColumnWidthsDigest:  strings.Repeat("f", sha256HexBytes),
			DisabledReasonBytes: math.MaxInt,
			ChangeCommand:       string(controlValue.ID),
			ActivateCommand:     string(controlValue.ID),
			SortCommand:         string(controlValue.ID),
			Viewport:            markdownViewport,
		},
	}
	dataGridControl := controlValue
	dataGridControl.Details = ControlDetails{
		Version: 1,
		Border:  tableControl.Details.Border,
		DataGrid: &DataGridDetails{
			Table:                 *tableControl.Details.Table,
			Editing:               true,
			EditRow:               string(controlValue.ID),
			EditColumn:            string(controlValue.ID),
			EditLength:            math.MaxInt,
			EditCaret:             math.MaxInt,
			EditViewOffset:        math.MaxInt,
			EditValid:             true,
			ValidationEnforcement: "hard",
			ValidationMode:        "blacklist",
		},
	}
	comboBoxControl := controlValue
	comboBoxControl.Details = ControlDetails{
		Version: 1,
		ComboBox: &ComboBoxDetails{
			Popup: DropDownDetails{
				ItemCount:           math.MaxInt,
				EnabledCount:        math.MaxInt,
				RetainedBytes:       math.MaxInt,
				Current:             string(controlValue.ID),
				CurrentIndex:        math.MaxInt,
				Selected:            string(controlValue.ID),
				SelectedIndex:       math.MaxInt,
				PopupRows:           math.MaxInt,
				PopupBounds:         controlValue.AbsoluteBounds,
				PopupOffset:         math.MaxInt,
				PopupCurrent:        string(controlValue.ID),
				PopupSelection:      string(controlValue.ID),
				DisabledReasonBytes: math.MaxInt,
				ChangeCommand:       string(controlValue.ID),
				ActivateCommand:     string(controlValue.ID),
			},
			Editor: *textFieldControl.Details.TextField,
		},
	}
	modalControl := controlValue
	modalControl.Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: math.MaxInt,
			InputScope:  "confined",
		},
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          "double",
			Style:         StyleID(controlValue.ID),
			ResolvedStyle: controlValue.ResolvedStyle,
		},
		ModalPanel: &ModalPanelDetails{
			Lifecycle:       "closed",
			StackIndex:      math.MaxInt,
			StackDepth:      math.MaxInt,
			NestedOwner:     controlValue.ID,
			SavedFocus:      controlValue.ID,
			InitialFocus:    controlValue.ID,
			RequestedSize:   Size{Width: math.MaxInt, Height: math.MaxInt},
			ResolvedBounds:  controlValue.Bounds,
			RequiredMinimum: Size{Width: math.MaxInt, Height: math.MaxInt},
			Degraded:        true,
			Shadow:          "turbo",
			ShadowStyle:     StyleID(controlValue.ID),
			Result: &ModalResultDetails{
				Reason: string(controlValue.ID), Action: string(controlValue.ID),
			},
		},
	}
	pickerControl := modalControl
	pickerControl.Kind = "file_picker_dialog"
	pickerControl.Details.FilePicker = &FilePickerDetails{
		Mode: "single", Status: "error",
		DisplayPathBytes:  math.MaxInt,
		DisplayPathDigest: strings.Repeat("f", sha256HexBytes),
		EntryCount:        math.MaxInt, FileCount: math.MaxInt,
		DirectoryCount:    math.MaxInt,
		CurrentNameBytes:  math.MaxInt,
		CurrentNameDigest: strings.Repeat("f", sha256HexBytes),
		CurrentKind:       "directory", SelectedCount: math.MaxInt,
		Filter: string(controlValue.ID), SortField: "modified",
		SortDirection: "descending", ErrorBytes: math.MaxInt,
		ErrorDigest: strings.Repeat("f", sha256HexBytes),
	}
	for _, candidate := range [][]byte{
		mustMarshal(t, borderControl),
		mustMarshal(t, dividerControl),
		mustMarshal(t, actionControl),
		mustMarshal(t, hotkeyControl),
		mustMarshal(t, focusGuideControl),
		mustMarshal(t, textFieldControl),
		mustMarshal(t, numberFieldControl),
		mustMarshal(t, textAreaControl),
		mustMarshal(t, progressControl),
		mustMarshal(t, markdownControl),
		mustMarshal(t, logControl),
		mustMarshal(t, streamControl),
		mustMarshal(t, listBoxControl),
		mustMarshal(t, treeViewControl),
		mustMarshal(t, tableControl),
		mustMarshal(t, dataGridControl),
		mustMarshal(t, comboBoxControl),
		mustMarshal(t, modalControl),
		mustMarshal(t, pickerControl),
	} {
		if len(candidate) > len(control) {
			control = candidate
		}
	}
	ordinaryControlBytes := len(control)
	scrollableControlBytes := len(mustMarshal(t, scrollableControl))
	contentControl := controlValue
	contentControl.Kind = "panel"
	contentControl.Details = ControlDetails{
		Version:   1,
		Container: &ContainerDetails{InputScope: "escaping"},
	}
	contentControlBytes := len(mustMarshal(t, contentControl))
	ordinaryControlsBytes := limits.Controls * ordinaryControlBytes
	scrollPairCount := limits.Controls / 2
	scrollControlsBytes := scrollPairCount *
		(scrollableControlBytes + contentControlBytes)
	if limits.Controls%2 != 0 {
		scrollControlsBytes += ordinaryControlBytes
	}
	controlArrayBytes := max(ordinaryControlsBytes, scrollControlsBytes)
	if limits.Controls > 0 {
		controlArrayBytes += limits.Controls - 1
	}
	controlExpansionBytes := max(0, controlArrayBytes-baseControlBytes)
	source := mustMarshal(t, completion.Snapshot.InputSources[0])
	overflow := mustMarshal(t, completion.Snapshot.Overflows[0])
	layout := mustMarshal(t, completion.Snapshot.Layouts[0])
	layoutItem := mustMarshal(t, completion.Snapshot.Layouts[0].Items[0])
	menuItem := mustMarshal(t, menuEntry)
	statusItem := mustMarshal(t, statusSegment)
	menuControlBytes := mustMarshal(t, menuControl)
	menuControlOverhead := max(0, len(menuControlBytes)-len(control))
	statusControlBytes := mustMarshal(t, statusControl)
	statusControlOverhead := max(0, len(statusControlBytes)-len(control))
	choiceItem := mustMarshal(t, selectionOption)
	radioItem := mustMarshal(t, radioOption)
	selectionItemBytes := max(len(choiceItem), len(radioItem))
	selectionControlBytes := max(
		len(mustMarshal(t, choiceControl)),
		len(mustMarshal(t, radioControl)),
	)
	selectionControlOverhead := max(
		0,
		selectionControlBytes-len(control),
	)
	t.Logf(
		"bound elements: base=%d run=%d ordinary_control=%d scroll_owner=%d scroll_content=%d control_array=%d source=%d overflow=%d layout=%d layout_item=%d menu_item=%d menu_control_overhead=%d status_item=%d status_control_overhead=%d selection_item=%d selection_control_overhead=%d text_input_payload=%d",
		len(base), len(run), ordinaryControlBytes, scrollableControlBytes,
		contentControlBytes, controlArrayBytes, len(source), len(overflow),
		len(layout), len(layoutItem), len(menuItem), menuControlOverhead,
		len(statusItem), statusControlOverhead, selectionItemBytes,
		selectionControlOverhead,
		2*expletives.MaxTextInputAggregateBytes,
	)

	return len(base) +
		(limits.FrameRuns-1)*(len(run)+1) +
		controlExpansionBytes +
		(limits.Controls-1)*(len(source)+1) +
		(limits.Controls-1)*(len(overflow)+1) +
		(limits.Layouts-1)*(len(layout)+1) +
		(limits.LayoutItems-limits.Layouts)*(len(layoutItem)+1) +
		menuControlOverhead +
		(expletives.MaxMenuItems-1)*(len(menuItem)+1) +
		statusControlOverhead +
		(expletives.MaxStatusBarSegments-1)*(len(statusItem)+1) +
		expletives.MaxSelectionItems*(selectionItemBytes+1) +
		expletives.MaxSelectionItems*selectionControlOverhead +
		2*expletives.MaxTextInputAggregateBytes
}

func float64Pointer(value float64) *float64 {
	return &value
}

func maximumElementCompletion(limits Limits) Completion {
	identifier := strings.Repeat("x", limits.IdentifierBytes)
	requestID := strings.Repeat("r", limits.RequestIDBytes)
	escapedCell := strings.Repeat("\x00", maxCellGraphemeBytes)
	escapedDisplayText := strings.Repeat("\x00", maxDisplayTextBytes)
	escapedError := strings.Repeat("<", maxErrorMessageBytes)
	escapedMessage := strings.Repeat("<", maxSnapshotMessageBytes)
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	sequence := uint64(math.MaxUint64)
	resolved := ResolvedStyle{
		Foreground: "#FFFFFF",
		Background: "#FFFFFF",
		Attributes: 31,
	}

	snapshot := SnapshotV1{
		Version:  1,
		Sequence: sequence,
		Final:    false,
		Scenario: identifier,
		Frame: IntendedFrame{
			Size: Size{
				Width:  limits.FrameWidth,
				Height: limits.FrameHeight,
			},
			Cells: []Cell{{
				Grapheme:   escapedCell,
				Style:      StyleID(identifier),
				Foreground: "#FFFFFF",
				Background: "#FFFFFF",
				Attributes: 31,
				Owner:      ControlID(identifier),
			}},
		},
		Cursor: CursorState{
			Visible: false,
			Position: Point{
				X: minInt,
				Y: minInt,
			},
		},
		Controls: []ControlSnapshot{{
			ID:       ControlID(identifier),
			Key:      identifier,
			Kind:     ControlKind(identifier),
			Parent:   ControlID(identifier),
			Children: []ControlID{ControlID(identifier)},
			Bounds: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			AbsoluteBounds: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			EffectiveClip: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			Minimum: Size{
				Width:  maxInt,
				Height: maxInt,
			},
			Layout:        LayoutID(identifier),
			LayoutIndex:   maxInt,
			StackIndex:    maxInt,
			Style:         StyleID(identifier),
			ResolvedStyle: resolved,
			Visible:       false,
			Details: ControlDetails{
				Version: 1,
				Text: &TextDetails{
					Text:                escapedDisplayText,
					HorizontalAlignment: TextAlignment(identifier),
					VerticalAlignment:   TextAlignment(identifier),
					Wrap:                TextWrap(identifier),
					Target:              ControlID(identifier),
					Mnemonic:            Key(identifier),
				},
			},
		}},
		Layouts: []LayoutSnapshot{{
			ID:          LayoutID(identifier),
			Key:         identifier,
			Kind:        LayoutKind(identifier),
			Owner:       ControlID(identifier),
			Parent:      LayoutID(identifier),
			Bounds:      Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
			OwnerBounds: Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
			Minimum:     Size{Width: maxInt, Height: maxInt},
			Border: &BorderDetails{
				Form:          "single",
				Style:         StyleID(identifier),
				ResolvedStyle: resolved,
			},
			LayoutIndex: maxInt,
			StackIndex:  maxInt,
			Items: []LayoutItemSnapshot{{
				Kind:        "panel",
				Panel:       ControlID(identifier),
				Bounds:      Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
				Minimum:     Size{Width: maxInt, Height: maxInt},
				LayoutIndex: maxInt,
				StackIndex:  maxInt,
			}},
		}},
		InputSources: []InputSourceSnapshot{{
			Source: identifier,
			Held: []Key{
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
			},
		}},
		Overflows: []OverflowSnapshot{{
			EpisodeID: identifier,
			Panel:     ControlID(identifier),
			Layout:    LayoutID(identifier),
			Available: Size{Width: maxInt, Height: maxInt},
			Required:  Size{Width: maxInt, Height: maxInt},
			Deficit:   Size{Width: maxInt, Height: maxInt},
			State:     identifier,
		}},
		Completion: &SnapshotCompletion{
			RequestID:     requestID,
			Outcome:       OutcomeInterrupted,
			Command:       identifier,
			FrameSequence: sequence,
			Code:          identifier,
			Message:       escapedMessage,
		},
	}

	compactFrame(&snapshot.Frame, true)
	return Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     requestID,
		Operation:     TypeQueryResult,
		Outcome:       OutcomeInterrupted,
		FrameSequence: sequence,
		Snapshot:      &snapshot,
		Error: &Error{
			Code:      identifier,
			Message:   escapedError,
			Retryable: false,
		},
		Result: &Result{
			Query: &QueryResult{
				TargetRequestID: requestID,
				Status:          "completed",
				Completion: &RetainedCompletion{
					RequestID:     requestID,
					Operation:     identifier,
					Outcome:       OutcomeInterrupted,
					FrameSequence: sequence,
					Error: &Error{
						Code:      identifier,
						Message:   escapedError,
						Retryable: false,
					},
				},
			},
		},
	}
}

func maximumValidElementCompletion(limits Limits) Completion {
	completion := maximumElementCompletion(limits)
	completion.Snapshot = snapshotPointer(cloneSnapshot(*completion.Snapshot))
	completion.Snapshot.Frame.Size = Size{Width: 1, Height: 1}
	completion.Snapshot.Frame.Cells[0].Grapheme = "<"
	compactFrame(&completion.Snapshot.Frame, true)
	completion.Snapshot.Controls[0].Kind = "frame"
	completion.Snapshot.Controls[0].Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: 1,
			InputScope:  "escaping",
		},
		Border: &BorderDetails{
			Title: maximumCanonicalTitle(),
			Form:  "single",
			Style: completion.Snapshot.Controls[0].Style,
			ResolvedStyle: completion.Snapshot.Controls[0].
				ResolvedStyle,
		},
	}
	completion.Snapshot.Controls[0].LayoutIndex = limits.LayoutItems - 1
	completion.Snapshot.Controls[0].StackIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].LayoutIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].StackIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].Items[0].LayoutIndex = 0
	completion.Snapshot.Layouts[0].Items[0].StackIndex = 0
	return completion
}

func maximumValidTextCompletion(limits Limits) Completion {
	completion := maximumValidElementCompletion(limits)
	control := &completion.Snapshot.Controls[0]
	control.Kind = "label"
	control.Details = ControlDetails{
		Version: 1,
		Text: &TextDetails{
			Text:                maximumCanonicalDisplayText(),
			HorizontalAlignment: "center",
			VerticalAlignment:   "end",
			Wrap:                "none",
			Target:              control.ID,
			Mnemonic:            "x",
		},
	}
	return completion
}

func maximumCanonicalTitle() string {
	return strings.Repeat("<", maxBorderTitleCells) +
		strings.Repeat("\u0301", (maxBorderTitleBytes-maxBorderTitleCells)/2)
}

func maximumCanonicalDisplayText() string {
	return strings.Repeat("<", maxDisplayTextCells)
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}
