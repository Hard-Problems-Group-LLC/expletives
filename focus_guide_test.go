package expletives

import (
	"context"
	"strings"
	"testing"
)

func TestFocusGuideBarTracksFocusAndCustomization(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 100, Height: 4})
	registerActionCommand(t, app, "action.run", "Run", true)

	button, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.run",
			Bounds:        Rect{Y: 1, Width: 12, Height: 1},
		},
		Command: "action.run",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	checkbox, err := NewCheckbox(app.Root(), CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "check.feature",
			Bounds:        Rect{Y: 2, Width: 20, Height: 1},
		},
		Label: "Feature",
	})
	if err != nil {
		t.Fatalf("NewCheckbox() error = %v", err)
	}
	if _, err := NewFocusGuideBar(app.Root(), FocusGuideBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "guide",
			Bounds:        Rect{Y: 3, Width: 100, Height: 1},
		},
	}); err != nil {
		t.Fatalf("NewFocusGuideBar() error = %v", err)
	}

	guide := controlByKey(t, app.Snapshot(), "guide").Details.FocusGuideBar
	if guide == nil || guide.Target != button.ID() ||
		guide.TargetKind != ControlButton ||
		!strings.Contains(guide.Text, "Enter or Space") {
		t.Fatalf("initial FocusGuideBarDetails = %#v", guide)
	}

	if err := app.SetFocusGuidance(button, FocusGuidance{
		Text: "Runs the current operation",
	}); err != nil {
		t.Fatalf("SetFocusGuidance(append) error = %v", err)
	}
	guide = controlByKey(t, app.Snapshot(), "guide").Details.FocusGuideBar
	if guide.Customization != FocusGuidanceAppend ||
		!strings.Contains(guide.Text, "Enter or Space") ||
		!strings.Contains(guide.Text, "Runs the current operation") {
		t.Fatalf("appended FocusGuideBarDetails = %#v", guide)
	}

	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		"down",
		KeyEvent{Kind: KeyEventPress, Key: KeyDown},
	); err != nil {
		t.Fatalf("DispatchKey(Down) error = %v", err)
	}
	guide = controlByKey(t, app.Snapshot(), "guide").Details.FocusGuideBar
	if guide.Target != checkbox.ID() ||
		guide.TargetKind != ControlCheckbox ||
		guide.Customization != "" ||
		!strings.Contains(guide.Text, "Space changes state") {
		t.Fatalf("Checkbox FocusGuideBarDetails = %#v", guide)
	}

	if err := app.SetFocusGuidance(checkbox, FocusGuidance{
		Mode: FocusGuidanceOverride,
		Text: "Application-owned guidance",
	}); err != nil {
		t.Fatalf("SetFocusGuidance(override) error = %v", err)
	}
	guide = controlByKey(t, app.Snapshot(), "guide").Details.FocusGuideBar
	if guide.Text != "Application-owned guidance" ||
		guide.Customization != FocusGuidanceOverride {
		t.Fatalf("overridden FocusGuideBarDetails = %#v", guide)
	}
	if got := rowText(app.Snapshot(), 3); !strings.HasPrefix(
		got,
		"Application-owned guidance",
	) {
		t.Fatalf("FocusGuideBar rendered row = %q, want left-aligned guidance", got)
	}

	if err := app.ClearFocusGuidance(checkbox); err != nil {
		t.Fatalf("ClearFocusGuidance() error = %v", err)
	}
	guide = controlByKey(t, app.Snapshot(), "guide").Details.FocusGuideBar
	if guide.Customization != "" ||
		!strings.Contains(guide.Text, "Space changes state") {
		t.Fatalf("cleared FocusGuideBarDetails = %#v", guide)
	}
}

func TestFocusGuidanceValidationIsAtomic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 2})
	registerActionCommand(t, app, "action.run", "Run", true)
	button, err := NewButton(app.Root(), ButtonOptions{
		Command: "action.run",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	sequence := app.Snapshot().Sequence
	for name, guidance := range map[string]FocusGuidance{
		"mode": {Mode: "invalid", Text: "Guidance"},
		"size": {
			Text: strings.Repeat("x", MaxFocusGuidanceApplicationBytes+1),
		},
		"line break": {Text: "first\nsecond"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := app.SetFocusGuidance(button, guidance); err == nil {
				t.Fatal("SetFocusGuidance() error = nil")
			}
			if got := app.Snapshot().Sequence; got != sequence {
				t.Fatalf("invalid guidance published sequence %d, want %d", got, sequence)
			}
		})
	}
}

func TestNavigationFocusGuidanceIsSpecific(t *testing.T) {
	t.Parallel()
	for kind, fragment := range map[ControlKind]string{
		ControlScrollBar:   "Page Up or Page Down",
		ControlTabbedPanel: "Space or Enter selects",
		ControlNotebook:    "Space or Enter selects",
	} {
		if guidance := genericFocusGuidance(kind); !strings.Contains(
			guidance,
			fragment,
		) {
			t.Fatalf(
				"genericFocusGuidance(%q) = %q, want fragment %q",
				kind,
				guidance,
				fragment,
			)
		}
	}
}
