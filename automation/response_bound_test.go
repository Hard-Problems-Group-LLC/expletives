package automation

import (
	"encoding/json"
	"math"
	"runtime"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

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
				Text: "abc", Length: 3, Caret: 2, ViewOffset: 0,
				SelectionStart: 2, SelectionEnd: 2,
				Valid: true, Enabled: true,
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
		"duplicate validator": func(details *TextFieldDetails) {
			details.Validator.Characters = "aabc"
		},
		"disabled editing": func(details *TextFieldDetails) {
			details.Enabled = false
			details.DisabledReason = "Disabled"
			details.Editing = true
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
	borderControl := controlValue
	borderControl.Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: controlValue.Bounds.X,
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
			Clamp:          true,
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
		Container: &ContainerDetails{},
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
			Length: math.MaxInt, Caret: math.MaxInt,
			ViewOffset: math.MaxInt, Editing: true,
			Valid: true, Password: true, Redacted: true,
			Enabled:        false,
			DisabledReason: strings.Repeat("\x00", maxDisplayTextBytes),
			ChangeCommand:  string(controlValue.ID),
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
	} {
		if len(candidate) > len(control) {
			control = candidate
		}
	}
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
		"bound elements: base=%d run=%d control=%d source=%d overflow=%d layout=%d layout_item=%d menu_item=%d menu_control_overhead=%d status_item=%d status_control_overhead=%d selection_item=%d selection_control_overhead=%d text_input_payload=%d",
		len(base), len(run), len(control), len(source), len(overflow), len(layout), len(layoutItem), len(menuItem), menuControlOverhead, len(statusItem), statusControlOverhead, selectionItemBytes, selectionControlOverhead,
		2*expletives.MaxTextInputAggregateBytes,
	)

	return len(base) +
		(limits.FrameRuns-1)*(len(run)+1) +
		(limits.Controls-1)*(len(control)+1) +
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
		Version:   1,
		Container: &ContainerDetails{ClientInset: 1},
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
