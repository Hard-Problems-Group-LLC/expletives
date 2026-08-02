package expletives

import (
	"context"
	"errors"
	"testing"
)

func newDialogTestApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp(AppOptions{
		Size: Size{Width: 40, Height: 14}, Scenario: "dialog.test",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	for _, command := range []CommandID{
		"outside", "generic.dialog.ok", "generic.dialog.cancel",
	} {
		if err := app.RegisterCommand(CommandDefinition{
			ID: command, Label: string(command), Enabled: true,
		}); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", command, err)
		}
	}
	return app
}

func TestDialogDerivesModalLifecycleAndScopesRolesAndMnemonics(t *testing.T) {
	app := newDialogTestApp(t)
	_, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "outside", Bounds: Rect{X: 1, Y: 1, Width: 14, Height: 1},
		},
		Command: "outside", Mnemonic: "o",
	})
	if err != nil {
		t.Fatalf("NewButton(outside) error = %v", err)
	}
	tx := app.NewTransaction()
	dialog, err := tx.NewDialog(app.Root(), DialogOptions{
		ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "dialog", Bounds: Rect{Width: 24, Height: 8},
			},
			Title: "Dialog",
		},
	})
	if err != nil {
		t.Fatalf("NewDialog() error = %v", err)
	}
	defaultPanel, err := tx.NewPanel(dialog, PanelOptions{
		AutomationKey: "dialog.default-panel",
		Bounds:        Rect{X: 1, Y: 1, Width: 18, Height: 2},
	})
	if err != nil {
		t.Fatalf("NewPanel(default) error = %v", err)
	}
	cancelPanel, err := tx.NewPanel(dialog, PanelOptions{
		AutomationKey: "dialog.cancel-panel",
		Bounds:        Rect{X: 1, Y: 4, Width: 18, Height: 2},
	})
	if err != nil {
		t.Fatalf("NewPanel(cancel) error = %v", err)
	}
	okButton, err := tx.NewButton(defaultPanel, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "dialog.ok", Bounds: Rect{Width: 14, Height: 1},
		},
		Command: "generic.dialog.ok", Mnemonic: "o", Default: true,
	})
	if err != nil {
		t.Fatalf("NewButton(default) error = %v", err)
	}
	_, err = tx.NewButton(cancelPanel, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "dialog.cancel", Bounds: Rect{Width: 14, Height: 1},
		},
		Command: "generic.dialog.cancel", Mnemonic: "c", Cancel: true,
	})
	if err != nil {
		t.Fatalf("NewButton(cancel) error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		routed = append(routed, command)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	if err := dialog.Show(nil); err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if app.Focused() != okButton {
		t.Fatalf("initial focus = %v, want default Button", app.Focused())
	}
	view := controlByKey(t, app.Snapshot(), "dialog")
	if view.Kind != ControlDialog || view.Style != "dialog" ||
		view.Details.ModalPanel == nil || !view.Details.ModalPanel.Active ||
		view.Details.Border == nil || view.Details.Border.Style != "dialog.border" ||
		view.Details.ModalPanel.ShadowStyle != "dialog.shadow" {
		t.Fatalf("Dialog snapshot = %+v", view)
	}
	enter, err := app.DispatchKey(
		context.Background(), "test", "enter",
		KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
	)
	if err != nil || enter.Outcome != OutcomeApplied || len(routed) != 1 ||
		routed[0].ID != "generic.dialog.ok" {
		t.Fatalf("Enter = %+v error=%v routed=%v", enter, err, routed)
	}
	escape, err := app.DispatchKey(
		context.Background(), "test", "escape",
		KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
	)
	if err != nil || escape.Outcome != OutcomeApplied || len(routed) != 2 ||
		routed[1].ID != "generic.dialog.cancel" {
		t.Fatalf("Escape = %+v error=%v routed=%v", escape, err, routed)
	}
	for index, event := range []KeyEvent{
		{Kind: KeyEventDown, Key: KeyAlt},
		{Kind: KeyEventPress, Key: "o"},
		{Kind: KeyEventUp, Key: KeyAlt},
	} {
		completion, dispatchErr := app.DispatchKey(
			context.Background(), "test", "mnemonic-"+string(rune('0'+index)), event,
		)
		if dispatchErr != nil {
			t.Fatalf("DispatchKey(mnemonic %d) error = %v", index, dispatchErr)
		}
		if index == 1 && completion.Outcome != OutcomeApplied {
			t.Fatalf("mnemonic completion = %+v", completion)
		}
	}
	if len(routed) != 3 || routed[2].ID != "generic.dialog.ok" {
		t.Fatalf("modal mnemonic routed = %v", routed)
	}
	if err := dialog.Close(ModalResult{Reason: ModalCancelled}); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestDialogRejectsDuplicateRolesAcrossDescendantPanels(t *testing.T) {
	app := newDialogTestApp(t)
	tx := app.NewTransaction()
	dialog, err := tx.NewDialog(app.Root(), DialogOptions{
		ModalPanelOptions: ModalPanelOptions{PanelOptions: PanelOptions{
			AutomationKey: "dialog.duplicate", Bounds: Rect{Width: 24, Height: 8},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := tx.NewPanel(dialog, PanelOptions{AutomationKey: "first"})
	second, _ := tx.NewPanel(dialog, PanelOptions{AutomationKey: "second"})
	_, err = tx.NewButton(first, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: "first.default"},
		Command:      "generic.dialog.ok", Default: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.NewButton(second, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: "second.default"},
		Command:      "generic.dialog.cancel", Default: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("Commit() duplicate role error = %v", err)
	}
}
