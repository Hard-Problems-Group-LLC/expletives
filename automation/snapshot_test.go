package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestSnapshotProjectionPreservesJSONAndDeepCopies(t *testing.T) {
	t.Parallel()

	foreground := expletives.RGB(0x11, 0x22, 0x33)
	background := expletives.RGB(0x44, 0x55, 0x66)
	resolved := expletives.ResolvedStyle{
		Foreground: foreground,
		Background: background,
		Attributes: expletives.StyleBold | expletives.StyleUnderline,
	}
	core := expletives.Snapshot{
		Version:  1,
		Sequence: 9,
		Final:    true,
		Scenario: "snapshot.projection",
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 1, Height: 1},
			Cells: []expletives.Cell{{
				Grapheme:   "A",
				Style:      "cell.style",
				Foreground: foreground,
				Background: background,
				Attributes: expletives.StyleItalic,
				Owner:      "root",
			}},
		},
		Cursor: expletives.CursorState{
			Visible:  true,
			Position: expletives.Point{X: 0, Y: 0},
		},
		Controls: []expletives.ControlSnapshot{{
			ID:             "root",
			Key:            "root",
			Kind:           expletives.ControlRoot,
			Children:       []expletives.ControlID{"child"},
			Bounds:         expletives.Rect{Width: 1, Height: 1},
			AbsoluteBounds: expletives.Rect{Width: 1, Height: 1},
			EffectiveClip:  expletives.Rect{Width: 1, Height: 1},
			Minimum:        expletives.Size{Width: 1, Height: 1},
			Layout:         "layout-1",
			LayoutIndex:    0,
			StackIndex:     0,
			Style:          "control.style",
			ResolvedStyle:  resolved,
			Visible:        true,
			Focused:        true,
			Details: expletives.ControlDetails{
				Version: expletives.ControlDetailsVersion,
				Container: &expletives.ContainerDetails{
					ClientInset: 1,
					InputScope:  expletives.InputScopeEscaping,
				},
				Border: &expletives.BorderDetails{
					Title:         "Border",
					Form:          expletives.BorderSingle,
					Style:         "border.style",
					ResolvedStyle: resolved,
				},
				Text: &expletives.TextDetails{
					Text:                "Text",
					HorizontalAlignment: expletives.TextAlignCenter,
					VerticalAlignment:   expletives.TextAlignEnd,
					Wrap:                expletives.TextWrapWords,
					Target:              "child",
					Mnemonic:            "t",
				},
				Divider: &expletives.DividerDetails{
					Orientation: expletives.Vertical,
					Form:        expletives.BorderDouble,
					Text:        "Rule",
					Alignment:   expletives.TextAlignStart,
				},
				Action: &expletives.ActionDetails{
					Label: "Run", Command: "action.run", Enabled: true,
					Mnemonic: "r", Pressed: true, Default: true,
				},
				HotkeyBar: &expletives.HotkeyBarDetails{
					Items: []expletives.HotkeyBarItemDetails{{
						Label: "Run", Command: "action.run", Enabled: true,
						Chord: &expletives.Chord{
							Key: "r", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
					}},
				},
				MenuBar: &expletives.MenuBarDetails{
					Entries: []expletives.MenuEntryDetails{{
						Key: "menu.file", Kind: expletives.MenuItemSubmenu,
						Label: "File", Enabled: true, Mnemonic: "f",
						Placement: expletives.MenuBarPlacementEnd,
						Selected:  true, Open: true, ChildCount: 1,
					}, {
						Key: "menu.open", ParentKey: "menu.file", Depth: 1,
						Kind: expletives.MenuItemCommand, Label: "Open",
						Command: "action.open", Enabled: true, Selected: true,
						Chord: &expletives.Chord{
							Key: "o", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
					}},
					OpenPath:     []string{"menu.file"},
					SelectedPath: []string{"menu.file", "menu.open"},
				},
				StatusBar: &expletives.StatusBarDetails{
					Segments: []expletives.StatusSegmentDetails{{
						Key: "status.run", Label: "Run",
						Command: "action.run", Priority: 10, Enabled: true,
						Chord: &expletives.Chord{
							Key: "r", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
						Rendered: true,
						Bounds: expletives.Rect{
							X: 2, Width: 12, Height: 1,
						},
					}},
				},
			},
		}},
		Layouts: []expletives.LayoutSnapshot{{
			ID:          "layout-1",
			Key:         "layout",
			Kind:        expletives.LayoutBox,
			Owner:       "root",
			Bounds:      expletives.Rect{Width: 1, Height: 1},
			OwnerBounds: expletives.Rect{Width: 1, Height: 1},
			Minimum:     expletives.Size{Width: 2, Height: 1},
			Border: &expletives.BorderDetails{
				Form:          expletives.BorderNone,
				Style:         "layout.border",
				ResolvedStyle: expletives.ResolvedStyle{},
			},
			LayoutIndex: 0,
			StackIndex:  0,
			Items: []expletives.LayoutItemSnapshot{{
				Kind:        "panel",
				Panel:       "root",
				Bounds:      expletives.Rect{Width: 1, Height: 1},
				Minimum:     expletives.Size{Width: 2, Height: 1},
				LayoutIndex: 0,
				StackIndex:  0,
			}},
		}},
		InputSources: []expletives.InputSourceSnapshot{{
			Source: "automation:session",
			Held:   []expletives.Key{expletives.KeyControl},
		}},
		Overflows: []expletives.OverflowSnapshot{{
			EpisodeID: "overflow-1",
			Panel:     "root",
			Layout:    "layout-1",
			Available: expletives.Size{Width: 1, Height: 1},
			Required:  expletives.Size{Width: 2, Height: 1},
			Deficit:   expletives.Size{Width: 1},
			State:     "active",
		}},
		Completion: &expletives.Completion{
			RequestID:     "request-1",
			Outcome:       expletives.OutcomeApplied,
			Command:       "scenario.reset",
			FrameSequence: 9,
			Code:          "public_code",
			Message:       "public message",
			Cause:         errors.New("local-only cause"),
		},
	}

	projected := snapshotFromCore(core)
	projectedJSON, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("Marshal(projected) error = %v", err)
	}
	if !bytes.Contains(projectedJSON, []byte(`"runs"`)) ||
		!bytes.Contains(projectedJSON, []byte(`"cells"`)) {
		t.Fatalf("direct projection omits compact or expanded frame view: %s", projectedJSON)
	}
	if bytes.Contains(projectedJSON, []byte("local-only cause")) {
		t.Fatal("projected JSON exposes local-only command cause")
	}
	if got := projected.Controls[0].Details.MenuBar.Entries[0].Placement; got != "end" {
		t.Fatalf("projected MenuBar placement = %q, want end", got)
	}
	if got := projected.Controls[0].Details.Container.InputScope; got != "escaping" {
		t.Fatalf("projected input scope = %q, want escaping", got)
	}
	if got := projected.Controls[0].Details.StatusBar.Segments[0]; got.Key !=
		"status.run" || got.Chord == nil || got.Chord.Key != "r" {
		t.Fatalf("projected StatusBar segment = %#v", got)
	}

	core.Frame.Cells[0].Grapheme = "Z"
	core.Controls[0].Children[0] = "changed"
	core.Controls[0].Details.Container.ClientInset = 9
	core.Controls[0].Details.Container.InputScope = expletives.InputScopeConfined
	core.Controls[0].Details.Border.Title = "Changed"
	core.Controls[0].Details.Text.Text = "Changed"
	core.Controls[0].Details.Divider.Text = "Changed"
	core.Controls[0].Details.Action.Label = "Changed"
	core.Controls[0].Details.HotkeyBar.Items[0].Label = "Changed"
	core.Controls[0].Details.HotkeyBar.Items[0].Chord.Modifiers[0] =
		expletives.KeyAlt
	core.Controls[0].Details.MenuBar.Entries[0].Label = "Changed"
	core.Controls[0].Details.MenuBar.Entries[1].Chord.Modifiers[0] =
		expletives.KeyAlt
	core.Controls[0].Details.MenuBar.OpenPath[0] = "changed"
	core.Controls[0].Details.MenuBar.SelectedPath[0] = "changed"
	core.Controls[0].Details.StatusBar.Segments[0].Label = "Changed"
	core.Controls[0].Details.StatusBar.Segments[0].Chord.Modifiers[0] =
		expletives.KeyAlt
	core.Layouts[0].Items[0].Kind = "layout"
	core.InputSources[0].Held[0] = expletives.KeyAlt
	core.Overflows[0].State = "changed"
	core.Completion.Message = "changed"
	afterSourceMutation, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("Marshal(projected after source mutation) error = %v", err)
	}
	if !bytes.Equal(afterSourceMutation, projectedJSON) {
		t.Fatal("projected snapshot aliases core snapshot storage")
	}

	cloned := cloneSnapshot(projected)
	projected.Frame.Cells[0].Grapheme = "Q"
	projected.Controls[0].Children[0] = "mutated"
	projected.Controls[0].Details.Container.ClientInset = 7
	projected.Controls[0].Details.Container.InputScope = "confined"
	projected.Controls[0].Details.Border.Title = "Mutated"
	projected.Controls[0].Details.Text.Text = "Mutated"
	projected.Controls[0].Details.Divider.Text = "Mutated"
	projected.Controls[0].Details.Action.Label = "Mutated"
	projected.Controls[0].Details.HotkeyBar.Items[0].Label = "Mutated"
	projected.Controls[0].Details.HotkeyBar.Items[0].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
	projected.Controls[0].Details.MenuBar.Entries[0].Label = "Mutated"
	projected.Controls[0].Details.MenuBar.Entries[1].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
	projected.Controls[0].Details.MenuBar.OpenPath[0] = "mutated"
	projected.Controls[0].Details.MenuBar.SelectedPath[0] = "mutated"
	projected.Controls[0].Details.StatusBar.Segments[0].Label = "Mutated"
	projected.Controls[0].Details.StatusBar.Segments[0].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
	projected.Layouts[0].Items[0].Kind = "layout"
	projected.InputSources[0].Held[0] = Key(expletives.KeyShift)
	projected.Overflows[0].State = "mutated"
	projected.Completion.Message = "mutated"
	clonedJSON, err := json.Marshal(cloned)
	if err != nil {
		t.Fatalf("Marshal(cloned) error = %v", err)
	}
	if !bytes.Equal(clonedJSON, projectedJSON) {
		t.Fatal("cloned snapshot aliases projected snapshot storage")
	}
}

func TestContainerInputScopeValidationIsKindAware(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	root := ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			InputScope: "confined",
		},
	}
	if !validControlDetails("root", root, 20, 5, limits) {
		t.Fatal("validControlDetails rejected the root's implicit scope")
	}
	root.Container.InputScope = ""
	if validControlDetails("root", root, 20, 5, limits) {
		t.Fatal("validControlDetails accepted a root without its implicit scope")
	}
	panel := ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			InputScope: "escaping",
		},
	}
	if !validControlDetails("panel", panel, 20, 5, limits) {
		t.Fatal("validControlDetails rejected an escaping Panel scope")
	}
	panel.Container.InputScope = "unknown"
	if validControlDetails("panel", panel, 20, 5, limits) {
		t.Fatal("validControlDetails accepted an unknown input scope")
	}
}

func TestSelectionSnapshotProjectionValidationAndDeepCopy(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 40, Height: 8},
		Scenario: "selection.projection",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.RegisterCommand(expletives.CommandDefinition{
		ID: "selection.changed", Label: "Changed", Enabled: true,
		Automation: true,
	}); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}
	if _, err := expletives.NewCheckbox(
		app.Root(),
		expletives.CheckboxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.checkbox",
				Bounds: expletives.Rect{
					Width: 16, Height: 1,
				},
			},
			Label: "Check", State: expletives.CheckIndeterminate,
			ThreeState: true, ChangeCommand: "selection.changed",
		},
	); err != nil {
		t.Fatalf("NewCheckbox() error = %v", err)
	}
	group, err := expletives.NewRadioGroup(
		app.Root(),
		expletives.RadioGroupOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.group",
				Bounds: expletives.Rect{
					Y: 1, Width: 16, Height: 2,
				},
			},
			ChangeCommand: "selection.changed",
		},
	)
	if err != nil {
		t.Fatalf("NewRadioGroup() error = %v", err)
	}
	if _, err := expletives.NewRadioButton(
		group,
		expletives.RadioButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radio",
				Bounds: expletives.Rect{
					Width: 12, Height: 1,
				},
			},
			Value: "one", Label: "One", Selected: true,
		},
	); err != nil {
		t.Fatalf("NewRadioButton() error = %v", err)
	}
	if _, err := expletives.NewCycleField(
		app.Root(),
		expletives.CycleFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.choice",
				Bounds: expletives.Rect{
					Y: 3, Width: 24, Height: 1,
				},
			},
			Label: "Choice", Value: "a",
			Options: []expletives.SelectionOption{
				{Value: "a", Label: "Alpha"},
				{Value: "b", Label: "Beta"},
			},
			ChangeCommand: "selection.changed",
		},
	); err != nil {
		t.Fatalf("NewCycleField() error = %v", err)
	}

	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot(selection) error = %v", err)
	}
	byKey := func(key string) *ControlSnapshot {
		t.Helper()
		for index := range projected.Controls {
			if projected.Controls[index].Key == key {
				return &projected.Controls[index]
			}
		}
		t.Fatalf("projected snapshot has no control %q", key)
		return nil
	}
	if details := byKey("selection.checkbox").Details.Checkbox; details == nil ||
		details.State != "indeterminate" || !details.ThreeState {
		t.Fatalf("projected CheckboxDetails = %#v", details)
	}
	if details := byKey("selection.group").Details.RadioGroup; details == nil ||
		details.Value != "one" || len(details.Options) != 1 ||
		!details.Options[0].Selected {
		t.Fatalf("projected RadioGroupDetails = %#v", details)
	}
	if details := byKey("selection.choice").Details.ChoiceField; details == nil ||
		details.SelectedIndex != 0 || len(details.Options) != 2 {
		t.Fatalf("projected ChoiceFieldDetails = %#v", details)
	}

	cloned := cloneSnapshot(projected)
	byKey("selection.group").Details.RadioGroup.Options[0].Label = "Changed"
	byKey("selection.choice").Details.ChoiceField.Options[0].Label = "Changed"
	var clonedGroup, clonedChoice *ControlSnapshot
	for index := range cloned.Controls {
		switch cloned.Controls[index].Key {
		case "selection.group":
			clonedGroup = &cloned.Controls[index]
		case "selection.choice":
			clonedChoice = &cloned.Controls[index]
		}
	}
	if clonedGroup.Details.RadioGroup.Options[0].Label != "One" ||
		clonedChoice.Details.ChoiceField.Options[0].Label != "Alpha" {
		t.Fatal("cloned selection options alias projected snapshot storage")
	}
}

func TestSnapshotProjectsFocusGuideBarDetails(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 80, Height: 3},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.RegisterCommand(expletives.CommandDefinition{
		ID: "action.run", Label: "Run", Enabled: true, Automation: true,
	}); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}
	button, err := expletives.NewButton(
		app.Root(),
		expletives.ButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "button.run",
			},
			Command: "action.run",
		},
	)
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	if _, err := expletives.NewFocusGuideBar(
		app.Root(),
		expletives.FocusGuideBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "guide",
				Bounds: expletives.Rect{
					Y: 2, Width: 80, Height: 1,
				},
			},
		},
	); err != nil {
		t.Fatalf("NewFocusGuideBar() error = %v", err)
	}
	if err := app.SetFocusGuidance(button, expletives.FocusGuidance{
		Text: "Application addition",
	}); err != nil {
		t.Fatalf("SetFocusGuidance() error = %v", err)
	}

	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var guide *FocusGuideBarDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "guide" {
			guide = projected.Controls[index].Details.FocusGuideBar
			break
		}
	}
	if guide == nil ||
		guide.Target != ControlID(button.ID()) ||
		guide.TargetKind != "button" ||
		guide.Customization != "append" ||
		!strings.Contains(guide.Text, "Application addition") {
		t.Fatalf("projected FocusGuideBarDetails = %#v", guide)
	}
}

func TestSnapshotProjectsAndRedactsTextFieldDetails(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 2},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	for _, command := range []expletives.CommandID{
		"input.edited", "input.submitted",
	} {
		if err := app.RegisterCommand(expletives.CommandDefinition{
			ID: command, Label: string(command), Enabled: true,
			Automation: true,
		}); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", command, err)
		}
	}
	if _, err := expletives.NewTextField(
		app.Root(),
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.password",
				Bounds: expletives.Rect{
					Width: 20, Height: 1,
				},
			},
			Text:          "secret",
			MaximumBytes:  9,
			Password:      true,
			EditCommand:   "input.edited",
			SubmitCommand: "input.submitted",
			FocusedStyle:  "text_input.focused",
			EditingStyle:  "text_input.focused_invalid",
			ByteStyles: []expletives.TextFieldByteStyle{
				{MinimumBytes: 0, Style: "text_input.valid"},
				{MinimumBytes: 5, Style: "text_input.invalid"},
			},
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationBlacklist,
				Characters:  " ",
			},
		},
	); err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TextFieldDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "input.password" {
			details = projected.Controls[index].Details.TextField
			break
		}
	}
	if details == nil || details.Text != "" || details.Length != 6 ||
		details.MaximumBytes != 9 ||
		!details.Password || !details.Redacted || !details.Valid ||
		details.EditCommand != "input.edited" ||
		details.SubmitCommand != "input.submitted" ||
		details.FocusedStyle != "text_input.focused" ||
		details.EditingStyle != "text_input.focused_invalid" ||
		len(details.ByteStyles) != 2 ||
		details.ByteStyles[1].MinimumBytes != 5 ||
		details.ByteStyles[1].Style != "text_input.invalid" ||
		details.Validator == nil ||
		details.Validator.Enforcement != "soft" ||
		details.Validator.Mode != "blacklist" {
		t.Fatalf("projected TextFieldDetails = %#v", details)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(encoded, []byte("secret")) {
		t.Fatal("password value leaked into projected JSON")
	}

	cloned := cloneSnapshot(projected)
	details.Validator.Characters = "mutated"
	details.ByteStyles[1].Style = "mutated"
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "input.password" &&
			(cloned.Controls[index].Details.TextField.Validator.Characters != " " ||
				cloned.Controls[index].Details.TextField.ByteStyles[1].Style !=
					"text_input.invalid") {
			t.Fatal("cloned TextField policy aliases projected storage")
		}
	}
}

func TestSnapshotProjectsNumericFieldDetailsAndCopiesBounds(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 2},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	minimum, maximum := 0.0, 10.0
	if _, err := expletives.NewSpinBox(
		app.Root(),
		expletives.SpinBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.spin",
				Bounds: expletives.Rect{
					Width: 20, Height: 1,
				},
			},
			Value: 2.5, Minimum: &minimum, Maximum: &maximum,
			DecimalPlaces: 1, Step: 0.5,
		},
	); err != nil {
		t.Fatalf("NewSpinBox() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *NumberFieldDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "input.spin" {
			details = projected.Controls[index].Details.NumberField
			break
		}
	}
	if details == nil || details.Text != "2.5" || details.Value != 2.5 ||
		details.DecimalPlaces != 1 || details.Step != 0.5 ||
		details.Minimum == nil || *details.Minimum != 0 ||
		details.Maximum == nil || *details.Maximum != 10 ||
		!details.Valid {
		t.Fatalf("projected NumberFieldDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	*details.Minimum = 99
	*details.Maximum = 100
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "input.spin" {
			copied := cloned.Controls[index].Details.NumberField
			if copied.Minimum == nil || *copied.Minimum != 0 ||
				copied.Maximum == nil || *copied.Maximum != 10 {
				t.Fatal("cloned numeric bounds alias projected storage")
			}
		}
	}
}

func TestSnapshotProjectsAndRedactsTextAreaDetails(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 5},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if _, err := expletives.NewTextArea(
		app.Root(),
		expletives.TextAreaOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.area",
				Bounds: expletives.Rect{
					Width: 12, Height: 3,
				},
			},
			Text: "secret\nvalue", Password: true, ReadOnly: true,
			Wrap: expletives.TextWrapWords,
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationBlacklist,
				Characters:  " ",
			},
		},
	); err != nil {
		t.Fatalf("NewTextArea() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TextAreaDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "input.area" {
			details = projected.Controls[index].Details.TextArea
			break
		}
	}
	if details == nil || details.Text != "" || details.Length != 12 ||
		details.LineCount != 2 || !details.Password || !details.ReadOnly || !details.Redacted ||
		details.Wrap != "words" || details.Validator == nil {
		t.Fatalf("projected TextAreaDetails = %#v", details)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(encoded, []byte("secret")) ||
		bytes.Contains(encoded, []byte("value")) {
		t.Fatal("TextArea password leaked into projected JSON")
	}
	cloned := cloneSnapshot(projected)
	details.Validator.Characters = "mutated"
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "input.area" &&
			cloned.Controls[index].Details.TextArea.Validator.Characters != " " {
			t.Fatal("cloned TextArea validator aliases projected storage")
		}
	}
}

func TestSnapshotProjectsProgressDetailsAndCopiesState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 3},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if _, err := expletives.NewProgressBar(
		app.Root(),
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar",
				Bounds:        expletives.Rect{Width: 10, Height: 1},
			},
			State: expletives.ProgressBarState{
				Indeterminate: true,
				Tick:          7,
				Status:        expletives.ProgressRunning,
			},
		},
	); err != nil {
		t.Fatalf("NewProgressBar() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *ProgressDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "progress.bar" {
			details = projected.Controls[index].Details.Progress
			break
		}
	}
	if details == nil || details.Status != "running" ||
		!details.Indeterminate || details.Tick != 7 ||
		details.TextMode != "none" || details.FrameIndex != 7 {
		t.Fatalf("projected ProgressDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Tick = 9
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "progress.bar" &&
			cloned.Controls[index].Details.Progress.Tick != 7 {
			t.Fatal("cloned progress state aliases projected storage")
		}
	}
}

func TestSnapshotProjectsScrollBarDetailsAndCopiesState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 3},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if _, err := expletives.NewScrollBar(
		app.Root(),
		expletives.ScrollBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "scroll.bar",
				Bounds:        expletives.Rect{Width: 12, Height: 1},
			},
			State: expletives.ScrollBarState{
				ContentSize: 100, ViewportSize: 20, Offset: 40,
			},
			ArrowStep: 2,
			PageStep:  25,
		},
	); err != nil {
		t.Fatalf("NewScrollBar() error = %v", err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *ScrollBarDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "scroll.bar" {
			details = projected.Controls[index].Details.ScrollBar
			break
		}
	}
	if details == nil ||
		details.ContentSize != 100 ||
		details.ViewportSize != 20 ||
		details.Offset != 40 ||
		details.MaximumOffset != 80 ||
		details.ArrowStep != 2 ||
		details.PageStep != 25 ||
		details.TrackStart != 1 ||
		details.TrackSize != 10 ||
		details.ThumbStart != 5 ||
		details.ThumbSize != 2 ||
		!details.Enabled {
		t.Fatalf("projected ScrollBarDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Offset = 9
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "scroll.bar" &&
			cloned.Controls[index].Details.ScrollBar.Offset != 40 {
			t.Fatal("cloned ScrollBar state aliases projected storage")
		}
	}
}

func TestSnapshotProjectsScrollableDetailsAndCopiesNestedBars(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := expletives.NewScrollablePanel(
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
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var owner, content *ControlSnapshot
	for index := range projected.Controls {
		switch projected.Controls[index].Key {
		case "scroll":
			owner = &projected.Controls[index]
		case "scroll.content":
			content = &projected.Controls[index]
		}
	}
	if owner == nil || content == nil || owner.Details.Scrollable == nil {
		t.Fatalf("projected controls owner=%#v content=%#v", owner, content)
	}
	details := owner.Details.Scrollable
	if details.State != (ViewportState{
		ContentSize: Size{Width: 20, Height: 10},
		Offset:      Point{X: 2, Y: 3},
	}) ||
		details.MaximumOffset != (Point{X: 13, Y: 7}) ||
		details.ViewportBounds != (Rect{
			X: 1, Y: 1, Width: 7, Height: 3,
		}) ||
		details.Content != content.ID ||
		details.ContentKey != content.Key ||
		details.HorizontalBar == nil ||
		details.VerticalBar == nil ||
		content.Bounds != (Rect{
			X: -2, Y: -3, Width: 20, Height: 10,
		}) {
		t.Fatalf("projected Scrollable details = %#v", details)
	}

	cloned := cloneSnapshot(projected)
	var clonedDetails *ScrollableDetails
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "scroll" {
			clonedDetails = cloned.Controls[index].Details.Scrollable
			break
		}
	}
	if clonedDetails == nil {
		t.Fatal("clone has no Scrollable details")
	}
	clonedDetails.HorizontalBar.Offset = 0
	clonedDetails.VerticalBar.Offset = 0
	if details.HorizontalBar.Offset != 2 ||
		details.VerticalBar.Offset != 3 {
		t.Fatal("cloneSnapshot exposed nested ScrollBarDetails storage")
	}
	if panel.Content().AutomationKey() != content.Key {
		t.Fatal("projection changed core managed Content")
	}
}

func TestSnapshotProjectsMarkdownDetailsAndCopiesBlocks(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err := expletives.NewMarkdownView(
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
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var control *ControlSnapshot
	for index := range projected.Controls {
		if projected.Controls[index].Key == "markdown" {
			control = &projected.Controls[index]
			break
		}
	}
	if control == nil || control.Details.Markdown == nil {
		t.Fatalf("projected Markdown control = %#v", control)
	}
	details := control.Details.Markdown
	if control.Kind != "markdown_view" || len(control.Children) != 0 ||
		details.SourceBytes != len(view.Markdown()) ||
		details.BlockCount != 7 || len(details.Blocks) != 4 ||
		!details.SummariesTruncated ||
		details.Viewport.State.ContentSize != (Size{
			Width: details.MaximumLineWidth, Height: details.RenderedRows,
		}) ||
		!details.Viewport.HorizontalVisible ||
		!details.Viewport.VerticalVisible {
		t.Fatalf("projected Markdown details = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	var clonedDetails *MarkdownDetails
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "markdown" {
			clonedDetails = cloned.Controls[index].Details.Markdown
			break
		}
	}
	if clonedDetails == nil {
		t.Fatal("clone has no Markdown details")
	}
	clonedDetails.Blocks[0].Kind = "mutated"
	if details.Blocks[0].Kind == "mutated" {
		t.Fatal("cloneSnapshot exposed Markdown block storage")
	}
}

func TestSnapshotProjectsListBoxDetailsAndCopiesViewport(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 10},
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
			{Key: "one", Label: "One", Description: "wrapped prose for evidence"},
			{Key: "two", Label: "Two"},
			{Key: "three", Label: "Three"},
		},
		SelectionMode: expletives.CollectionSelectionMultiple,
		Selected:      []string{"one", "three"},
		Wrap:          expletives.TextWrapWords,
	})
	if err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *ListBoxDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "list" {
			details = projected.Controls[index].Details.ListBox
			break
		}
	}
	if details == nil || details.Status != "ready" ||
		details.StatusMessageBytes != 0 || details.StatusMessageDigest != "" ||
		details.ItemCount != 3 || details.EnabledCount != 3 ||
		details.VisualRowCount <= details.ItemCount ||
		details.Wrap != TextWrap(expletives.TextWrapWords) ||
		details.RetainedBytes <= 0 || details.Current != "one" ||
		details.CurrentIndex != 0 || details.SelectionMode != "multiple" ||
		details.SelectedCount != 2 || details.FirstSelected != "one" ||
		details.LastSelected != "three" || len(details.SelectionDigest) != 64 ||
		!details.Viewport.VerticalVisible {
		t.Fatalf("projected ListBox details = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	var clonedDetails *ListBoxDetails
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "list" {
			clonedDetails = cloned.Controls[index].Details.ListBox
			break
		}
	}
	if clonedDetails == nil {
		t.Fatal("clone has no ListBox viewport details")
	}
	clonedDetails.Status = "mutated"
	if details.Status == "mutated" {
		t.Fatal("cloneSnapshot exposed ListBoxDetails storage")
	}
}

func TestSnapshotProjectsTreeViewDetailsAndCopiesViewport(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 36, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = expletives.NewTreeView(app.Root(), expletives.TreeViewOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "tree",
					Bounds:        expletives.Rect{Width: 24, Height: 6},
				},
			},
			BorderForm:  expletives.BorderSingle,
			VerticalBar: expletives.ScrollBarVisibilityAuto,
		},
		Nodes: []expletives.TreeNode{{
			Key: "root", Label: "Root", Expanded: true,
			Children: []expletives.TreeNode{
				{Key: "one", Label: "One"},
				{Key: "two", Label: "Two"},
			},
		}},
		Current: "one", SelectionMode: expletives.CollectionSelectionMultiple,
		Selected: []string{"root", "two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TreeViewDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "tree" {
			details = projected.Controls[index].Details.TreeView
			break
		}
	}
	if details == nil || details.Status != "ready" ||
		details.StatusMessageBytes != 0 || details.StatusMessageDigest != "" ||
		details.NodeCount != 3 || details.VisibleCount != 3 ||
		details.EnabledCount != 3 || details.RetainedBytes <= 0 ||
		details.Current != "one" || details.CurrentIndex != 1 ||
		details.SelectionMode != "multiple" || details.SelectedCount != 2 ||
		details.FirstSelected != "root" || details.LastSelected != "two" ||
		details.ExpandedCount != 1 || details.FirstExpanded != "root" ||
		details.LastExpanded != "root" ||
		len(details.SelectionDigest) != 64 || len(details.ExpansionDigest) != 64 {
		t.Fatalf("projected TreeView details = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	var clonedDetails *TreeViewDetails
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "tree" {
			clonedDetails = cloned.Controls[index].Details.TreeView
			break
		}
	}
	if clonedDetails == nil {
		t.Fatal("clone has no TreeView details")
	}
	clonedDetails.Status = "mutated"
	if details.Status == "mutated" {
		t.Fatal("cloneSnapshot exposed TreeViewDetails storage")
	}
}

func TestSnapshotProjectsTableDetailsAndCopiesState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 44, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "table",
					Bounds:        expletives.Rect{Width: 32, Height: 7},
				},
			},
			BorderForm: expletives.BorderSingle,
		},
		Columns: []expletives.Column{
			{Key: "name", Header: "Name", Grow: 1, Sortable: true},
			{Key: "state", Header: "State", Width: 8},
		},
		Rows: []expletives.TableRow{
			{Key: "two", Cells: []expletives.TableCell{{Column: "name", Text: "Two"}, {Column: "state", Text: "Ready"}}},
			{Key: "one", Cells: []expletives.TableCell{{Column: "name", Text: "One"}, {Column: "state", Text: "Ready"}}},
		},
		CurrentRow: "two", CurrentColumn: "state",
		FocusMode:     expletives.TableFocusCell,
		SelectionMode: expletives.CollectionSelectionMultiple,
		Selected:      []string{"one", "two"},
		SortColumn:    "name", SortDirection: expletives.SortAscending,
		ColumnPresentation: []expletives.TableColumnPresentation{
			{Column: "state", Visible: true, Wrap: expletives.TableColumnHang},
			{Column: "name", Visible: false, Wrap: expletives.TableColumnWrapWords},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TableDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "table" {
			details = projected.Controls[index].Details.Table
			break
		}
	}
	if details == nil || details.Status != "ready" ||
		details.StatusMessageBytes != 0 || details.StatusMessageDigest != "" ||
		details.RowCount != 2 || details.EnabledCount != 2 ||
		details.ColumnCount != 2 || details.CellCount != 4 || details.RetainedBytes <= 0 ||
		details.CurrentRow != "two" || details.CurrentRowIndex != 1 ||
		details.CurrentColumn != "state" || details.CurrentColumnIndex != 0 ||
		details.FocusMode != "cell" || details.SelectionMode != "multiple" ||
		details.SelectionStyle != "multiple" || details.RangeAnchor != "" ||
		details.RangeExtent != "" ||
		details.SelectedCount != 2 || details.FirstSelected != "one" ||
		details.LastSelected != "two" || details.SortColumn != "name" ||
		details.SortDirection != "ascending" || details.VisibleColumnCount != 1 ||
		details.FirstVisibleColumn != "state" || details.LastVisibleColumn != "state" ||
		len(details.SelectionDigest) != 64 || len(details.PresentationDigest) != 64 ||
		len(details.ColumnWidthsDigest) != 64 {
		t.Fatalf("projected Table details = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	var clonedDetails *TableDetails
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "table" {
			clonedDetails = cloned.Controls[index].Details.Table
			break
		}
	}
	if clonedDetails == nil {
		t.Fatal("clone has no Table details")
	}
	clonedDetails.Status = "mutated"
	if details.Status == "mutated" {
		t.Fatal("cloneSnapshot exposed TableDetails storage")
	}
}

func TestSnapshotProjectsTableRangeSelection(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 28, Height: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "range-table", Bounds: expletives.Rect{Width: 20, Height: 6},
			}},
		},
		Columns: []expletives.Column{{Key: "value", Header: "Value"}},
		Rows: []expletives.TableRow{
			{Key: "one", Cells: []expletives.TableCell{{Column: "value", Text: "One"}}},
			{Key: "disabled", Disabled: true, DisabledReason: "Unavailable"},
			{Key: "three", Cells: []expletives.TableCell{{Column: "value", Text: "Three"}}},
		},
		SelectionStyle:   expletives.TableSelectionRange,
		RequireSelection: true, RangeAnchor: "one", RangeExtent: "three",
	})
	if err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TableDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "range-table" {
			details = projected.Controls[index].Details.Table
			break
		}
	}
	if details == nil || details.SelectionStyle != "range" ||
		details.SelectionMode != "multiple" || !details.RequireSelection ||
		details.RangeAnchor != "one" || details.RangeExtent != "three" ||
		details.SelectedCount != 2 || details.FirstSelected != "one" ||
		details.LastSelected != "three" {
		t.Fatalf("projected range Table details = %#v", details)
	}
}

func TestSnapshotProjectsDataGridEditorWithoutTextOrValidatorSet(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 40, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "data-grid", Bounds: expletives.Rect{Width: 30, Height: 6},
			}},
			BorderForm: expletives.BorderSingle,
		},
		Columns: []expletives.Column{{
			Key: "value", Header: "Value", Editable: true,
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationWhitelist,
				Characters:  "One",
			},
		}},
		Rows: []expletives.TableRow{{
			Key: "row", Cells: []expletives.TableCell{{Column: "value", Text: "One"}},
		}},
		SelectionStyle:   expletives.TableSelectionRange,
		RequireSelection: true, RangeAnchor: "row", RangeExtent: "row",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.DispatchKey(
		context.Background(), "automation-test", "grid-edit",
		expletives.KeyEvent{Kind: expletives.KeyEventPress, Key: expletives.KeyEnter},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DispatchTextInput(
		context.Background(), "automation-test", "grid-type",
		expletives.TextInputEvent{Kind: expletives.TextInputCommitted, Text: "!"},
	); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *DataGridDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "data-grid" {
			details = projected.Controls[index].Details.DataGrid
			break
		}
	}
	if details == nil || !details.Editing || details.EditRow != "row" ||
		details.EditColumn != "value" || details.EditLength != 4 ||
		details.EditCaret != 4 || details.EditValid ||
		details.Table.SelectionStyle != "range" ||
		details.Table.RangeAnchor != "row" || details.Table.RangeExtent != "row" ||
		details.Table.SelectedCount != 1 ||
		details.ValidationEnforcement != "soft" ||
		details.ValidationMode != "whitelist" || details.Table.FocusMode != "cell" {
		t.Fatalf("projected DataGrid details = %#v", details)
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "One!") || strings.Contains(string(encoded), "Characters") {
		t.Fatalf("DataGrid automation leaked edit text or validator set: %s", encoded)
	}
	cloned := cloneSnapshot(projected)
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "data-grid" {
			cloned.Controls[index].Details.DataGrid.EditRow = "mutated"
		}
	}
	if details.EditRow == "mutated" {
		t.Fatal("cloneSnapshot exposed DataGridDetails storage")
	}
}

func TestSnapshotProjectsPopupCollectionDetailsAndCopiesState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 32, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	dropDown, err := expletives.NewDropDown(app.Root(), expletives.DropDownOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "drop",
			Bounds:        expletives.Rect{X: 20, Y: 7, Width: 12, Height: 1},
		},
		Items: []expletives.ListItem{
			{Key: "one", Label: "One"},
			{Key: "two", Label: "A long second item"},
		},
		AllowEmpty: true, Selected: "one", PopupRows: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := dropDown.Open(); err != nil {
		t.Fatal(err)
	}
	_, err = expletives.NewComboBox(app.Root(), expletives.ComboBoxOptions{
		DropDownOptions: expletives.DropDownOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "combo",
				Bounds:        expletives.Rect{Width: 14, Height: 1},
			},
			Items: []expletives.ListItem{
				{Key: "alpha", Label: "Alpha"},
				{Key: "beta", Label: "Beta"},
			},
			Selected: "beta",
		},
		Validator: &expletives.TextValidator{
			Enforcement: expletives.TextValidationSoft,
			Mode:        expletives.TextValidationWhitelist,
			Characters:  "AlphaBet",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var dropDetails *DropDownDetails
	var comboDetails *ComboBoxDetails
	for index := range projected.Controls {
		switch projected.Controls[index].Key {
		case "drop":
			dropDetails = projected.Controls[index].Details.DropDown
		case "combo":
			comboDetails = projected.Controls[index].Details.ComboBox
		}
	}
	if dropDetails == nil || !dropDetails.Open ||
		dropDetails.ItemCount != 2 || dropDetails.EnabledCount != 2 ||
		dropDetails.Selected != "one" || dropDetails.SelectedIndex != 0 ||
		dropDetails.PopupCurrent != "one" ||
		dropDetails.PopupSelection != "one" ||
		dropDetails.PopupBounds.Width <= 12 || dropDetails.RetainedBytes == 0 {
		t.Fatalf("projected DropDown details = %#v", dropDetails)
	}
	if comboDetails == nil || comboDetails.Popup.Open ||
		comboDetails.Popup.Selected != "beta" ||
		comboDetails.Editor.Text != "Beta" ||
		comboDetails.Editor.Validator == nil ||
		comboDetails.Editor.Validator.Enforcement != "soft" {
		t.Fatalf("projected ComboBox details = %#v", comboDetails)
	}

	cloned := cloneSnapshot(projected)
	for index := range cloned.Controls {
		switch cloned.Controls[index].Key {
		case "drop":
			cloned.Controls[index].Details.DropDown.Selected = "mutated"
		case "combo":
			cloned.Controls[index].Details.ComboBox.Editor.Validator.Characters =
				"mutated"
		}
	}
	if dropDetails.Selected == "mutated" ||
		comboDetails.Editor.Validator.Characters == "mutated" {
		t.Fatal("cloneSnapshot exposed popup collection detail storage")
	}
}

func TestSnapshotProjectsLogAndStreamDetailsAndCopiesState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 40, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	logView, err := expletives.NewLogView(app.Root(), expletives.LogViewOptions{
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
			{Key: "b", Level: expletives.LogWarning, Text: "beta"},
			{Key: "c", Level: expletives.LogError, Text: "gamma"},
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
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var logDetails *LogViewDetails
	var streamDetails *StreamViewDetails
	for index := range projected.Controls {
		switch projected.Controls[index].Key {
		case "log":
			logDetails = projected.Controls[index].Details.LogView
		case "stream":
			streamDetails = projected.Controls[index].Details.StreamView
		}
	}
	if logDetails == nil || logDetails.RetainedRecords != 2 ||
		logDetails.DroppedRecords != 1 || logDetails.FirstKey != "b" ||
		logDetails.LastKey != "c" || logDetails.RetainedBytes !=
		logView.State().RetainedBytes || logDetails.Viewport.State.Offset !=
		(pointFromCore(logView.State().Offset)) {
		t.Fatalf("projected LogView details = %#v", logDetails)
	}
	if streamDetails == nil || streamDetails.PendingBytes != 1 ||
		!streamDetails.PendingTruncated || streamDetails.DroppedLines != 1 ||
		streamDetails.DroppedBytes != 19 ||
		streamDetails.PendingStorageBytes != 12 ||
		streamDetails.RetainedLines != 0 ||
		streamDetails.PendingBytes != stream.State().PendingBytes {
		t.Fatalf("projected StreamView details = %#v", streamDetails)
	}
	cloned := cloneSnapshot(projected)
	for index := range cloned.Controls {
		switch cloned.Controls[index].Key {
		case "log":
			cloned.Controls[index].Details.LogView.Capacity.Records = 99
		case "stream":
			cloned.Controls[index].Details.StreamView.Follow = true
		}
	}
	if logDetails.Capacity.Records == 99 || streamDetails.Follow {
		t.Fatal("cloneSnapshot exposed LogView or StreamView detail storage")
	}
}

func TestSnapshotProjectsTabbedPanelDetailsAndCopiesTabs(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	panel, err := transaction.NewTabbedPanel(
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
		"second",
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *TabbedPanelDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "tabs" {
			details = projected.Controls[index].Details.TabbedPanel
			break
		}
	}
	if details == nil ||
		details.Selected != "second" ||
		details.Current != "second" ||
		len(details.Tabs) != 2 ||
		details.Tabs[0].PageKey != "page.first" ||
		details.Tabs[1].Page != ControlID(second.ID()) ||
		!details.Tabs[1].Selected ||
		!details.Tabs[1].Current {
		t.Fatalf("projected TabbedPanelDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Tabs[0].Label = "mutated"
	details.Selected = "mutated"
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "tabs" {
			copied := cloned.Controls[index].Details.TabbedPanel
			if copied.Tabs[0].Label != "First" ||
				copied.Selected != "second" {
				t.Fatal("cloned Tab details alias projected storage")
			}
		}
	}
}

func TestSnapshotProjectsClosedDialogResultAndDeepCopies(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	modal, err := expletives.NewDialog(
		app.Root(),
		expletives.DialogOptions{ModalPanelOptions: expletives.ModalPanelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "modal", Bounds: expletives.Rect{Width: 16, Height: 5},
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := modal.Show(nil); err != nil {
		t.Fatal(err)
	}
	if err := modal.Close(expletives.ModalResult{
		Reason: expletives.ModalAccepted,
		Action: "dialog.ok",
	}); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}
	var details *ModalPanelDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "modal" {
			if projected.Controls[index].Kind != "dialog" {
				t.Fatalf("projected kind = %q", projected.Controls[index].Kind)
			}
			details = projected.Controls[index].Details.ModalPanel
			break
		}
	}
	if details == nil || details.Lifecycle != "closed" || details.Active ||
		details.Result == nil || details.Result.Reason != "accepted" ||
		details.Result.Action != "dialog.ok" {
		t.Fatalf("projected ModalPanelDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Result.Action = "mutated"
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "modal" &&
			cloned.Controls[index].Details.ModalPanel.Result.Action != "dialog.ok" {
			t.Fatal("cloneSnapshot exposed ModalPanel result storage")
		}
	}
}

func TestSnapshotValidatesStandardDialogKinds(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 50, Height: 16},
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := expletives.NewMessageBox(
		app.Root(),
		expletives.MessageBoxOptions{
			DialogOptions: expletives.DialogOptions{
				ModalPanelOptions: expletives.ModalPanelOptions{
					PanelOptions: expletives.PanelOptions{AutomationKey: "message"},
				},
			},
			Message: "Complete.",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	confirm, err := expletives.NewConfirmDialog(
		app.Root(),
		expletives.ConfirmDialogOptions{
			DialogOptions: expletives.DialogOptions{
				ModalPanelOptions: expletives.ModalPanelOptions{
					PanelOptions: expletives.PanelOptions{AutomationKey: "confirm"},
				},
			},
			Message: "Continue?", ShowCancel: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		key  string
		kind ControlKind
		show func() error
		stop func() error
	}{
		{
			key: "message", kind: "message_box", show: func() error { return message.Show(nil) },
			stop: func() error {
				return message.Close(expletives.ModalResult{Reason: expletives.ModalDismissed})
			},
		},
		{
			key: "confirm", kind: "confirm_dialog", show: func() error { return confirm.Show(nil) },
			stop: func() error {
				return confirm.Close(expletives.ModalResult{Reason: expletives.ModalDismissed})
			},
		},
	} {
		if err := fixture.show(); err != nil {
			t.Fatalf("Show(%s) error = %v", fixture.key, err)
		}
		projected := snapshotFromCore(app.Snapshot())
		if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
			t.Fatalf("validateSnapshot(%s) error = %v", fixture.key, err)
		}
		found := false
		for index := range projected.Controls {
			control := projected.Controls[index]
			if control.Key == fixture.key {
				found = control.Kind == fixture.kind &&
					control.Details.ModalPanel != nil &&
					control.Details.ModalPanel.Active
				break
			}
		}
		if !found {
			t.Fatalf("projected %s kind/details missing", fixture.key)
		}
		if err := fixture.stop(); err != nil {
			t.Fatalf("Close(%s) error = %v", fixture.key, err)
		}
	}
}

func TestInputDialogSnapshotRedactsValueAndValidatorCharacters(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 50, Height: 16},
	})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := expletives.NewInputDialog(
		app.Root(),
		expletives.InputDialogOptions{
			DialogOptions: expletives.DialogOptions{
				ModalPanelOptions: expletives.ModalPanelOptions{
					PanelOptions: expletives.PanelOptions{AutomationKey: "input"},
				},
			},
			Prompt: "Enter a private value:",
			Text:   "private",
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationWhitelist,
				Characters:  "abcdefghijklmnopqrstuvwxyz",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := dialog.Show(nil); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot(input dialog) error = %v", err)
	}
	var field *TextFieldDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "input.input" {
			field = projected.Controls[index].Details.TextField
			break
		}
	}
	if field == nil {
		t.Fatal("projected input field details missing")
	}
	if field.Text != "" || !field.Redacted || field.Password ||
		field.Length != len("private") || field.Validator == nil ||
		field.Validator.Characters != "" ||
		!field.Validator.CharactersRedacted ||
		field.Validator.Enforcement != "soft" ||
		field.Validator.Mode != "whitelist" {
		t.Fatalf("projected sensitive TextFieldDetails = %#v", field)
	}
}

func TestProgressDialogSnapshotTracksCancellationRequest(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 50, Height: 16},
	})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := expletives.NewProgressDialog(
		app.Root(),
		expletives.ProgressDialogOptions{
			DialogOptions: expletives.DialogOptions{
				ModalPanelOptions: expletives.ModalPanelOptions{
					PanelOptions: expletives.PanelOptions{AutomationKey: "progress"},
				},
			},
			State: expletives.ProgressDialogState{
				Status: "Working",
				Progress: expletives.ProgressBarState{
					Indeterminate: true,
					Tick:          7,
					Status:        expletives.ProgressRunning,
				},
			},
			Cancellable: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := dialog.Show(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DispatchKey(
		context.Background(), "test", "cancel",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyEscape,
		},
	); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot(progress dialog) error = %v", err)
	}
	var details *ProgressDialogDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "progress" {
			control := projected.Controls[index]
			if control.Kind != "progress_dialog" ||
				control.Details.ModalPanel == nil ||
				!control.Details.ModalPanel.Active {
				t.Fatalf("projected progress root = %#v", control)
			}
			details = control.Details.ProgressDialog
			break
		}
	}
	if details == nil || details.StatusLength != len("Working") ||
		!details.Cancellable || !details.CancelRequested ||
		!details.Progress.Indeterminate || details.Progress.Tick != 7 ||
		details.Progress.Status != "running" {
		t.Fatalf("projected ProgressDialogDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Cancellable = false
	if err := validateSnapshot(&projected, DefaultLimits()); err == nil {
		t.Fatal("validateSnapshot accepted requested but non-cancellable dialog")
	}
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "progress" &&
			!cloned.Controls[index].Details.ProgressDialog.Cancellable {
			t.Fatal("cloneSnapshot exposed ProgressDialogDetails storage")
		}
	}
}

func TestFilePickerSnapshotCompactsDisplayStringsAndValidatesKind(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(root+"/example.txt", []byte("example"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root+"/nested", 0o700); err != nil {
		t.Fatal(err)
	}
	provider, err := expletives.NewLocalFilePickerProvider(
		expletives.LocalFilePickerProviderOptions{Root: root},
	)
	if err != nil {
		t.Fatal(err)
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 80, Height: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	picker, err := expletives.NewFilePickerDialog(
		context.Background(), app.Root(), expletives.FilePickerDialogOptions{
			DialogOptions: expletives.DialogOptions{
				ModalPanelOptions: expletives.ModalPanelOptions{
					PanelOptions: expletives.PanelOptions{AutomationKey: "picker"},
				},
			},
			Provider: provider, InitialDirectory: provider.Root(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := picker.Show(nil); err != nil {
		t.Fatal(err)
	}
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("validateSnapshot(file picker) error = %v", err)
	}
	var details *FilePickerDetails
	for index := range projected.Controls {
		if projected.Controls[index].Key == "picker" {
			control := projected.Controls[index]
			if control.Kind != "file_picker_dialog" ||
				control.Details.ModalPanel == nil ||
				!control.Details.ModalPanel.Active {
				t.Fatalf("projected picker root = %#v", control)
			}
			details = control.Details.FilePicker
			break
		}
	}
	if details == nil || details.Mode != "single" || details.Status != "ready" ||
		details.DisplayPathBytes != len(root) ||
		!validLowerSHA256(details.DisplayPathDigest) ||
		details.EntryCount != 2 || details.FileCount != 1 ||
		details.DirectoryCount != 1 || details.Filter != "all" ||
		details.SortField != "name" || details.SortDirection != "ascending" ||
		details.ErrorBytes != 0 || details.ErrorDigest != "" {
		t.Fatalf("projected FilePickerDetails = %#v", details)
	}
	cloned := cloneSnapshot(projected)
	details.Mode = "directory"
	if err := validateSnapshot(&projected, DefaultLimits()); err == nil {
		t.Fatal("validateSnapshot accepted FilePickerDetails kind/mode mismatch")
	}
	for index := range cloned.Controls {
		if cloned.Controls[index].Key == "picker" &&
			cloned.Controls[index].Details.FilePicker.Mode != "single" {
			t.Fatal("cloneSnapshot exposed FilePickerDetails storage")
		}
	}
}

func TestExpandFrameRejectsDisagreeingCompactAndExpandedViews(t *testing.T) {
	t.Parallel()

	frame := IntendedFrame{
		Size:  Size{Width: 1, Height: 1},
		Runs:  []CellRun{{Count: 1, Cell: Cell{Grapheme: "A"}}},
		Cells: []Cell{{Grapheme: "B"}},
	}
	if err := expandFrame(&frame, 1, 1); err == nil {
		t.Fatal("expandFrame() accepted contradictory frame representations")
	}
}
