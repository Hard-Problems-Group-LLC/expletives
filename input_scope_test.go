package expletives

import (
	"context"
	"errors"
	"testing"
)

func dispatchAltMnemonic(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		request+"-alt-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	); err != nil {
		t.Fatalf("DispatchKey(%s Alt down) error = %v", request, err)
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s Alt-%s) error = %v", request, key, err)
	}
	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		request+"-alt-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyAlt},
	); err != nil {
		t.Fatalf("DispatchKey(%s Alt up) error = %v", request, err)
	}
	return completion
}

func TestNonModalInputScopesOwnActionsTraversalAndAtomicActivation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 50, Height: 12})
	for _, command := range []CommandID{
		"scope.a.save", "scope.a.cancel", "scope.b.save", "scope.b.cancel",
		"scope.root",
	} {
		registerActionCommand(t, app, command, string(command), true)
	}
	tx := app.NewTransaction()
	scopeA, err := tx.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "scope.a",
		Bounds:        Rect{X: 1, Y: 1, Width: 22, Height: 8},
		InputScope:    InputScopeConfined,
	})
	if err != nil {
		t.Fatalf("NewPanel(scope A) error = %v", err)
	}
	scopeB, err := tx.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "scope.b",
		Bounds:        Rect{X: 1, Y: 1, Width: 22, Height: 8},
		Hidden:        true,
		InputScope:    InputScopeConfined,
	})
	if err != nil {
		t.Fatalf("NewPanel(scope B) error = %v", err)
	}
	newScopedButtons := func(
		scope *Panel,
		prefix string,
		y int,
	) (*Button, *Button) {
		t.Helper()
		firstGroup, groupErr := tx.NewPanel(scope, PanelOptions{
			Bounds: Rect{X: 0, Y: y, Width: 20, Height: 2},
		})
		if groupErr != nil {
			t.Fatalf("NewPanel(%s first group) error = %v", prefix, groupErr)
		}
		secondGroup, groupErr := tx.NewPanel(scope, PanelOptions{
			Bounds: Rect{X: 0, Y: y + 3, Width: 20, Height: 2},
		})
		if groupErr != nil {
			t.Fatalf("NewPanel(%s second group) error = %v", prefix, groupErr)
		}
		save, buttonErr := tx.NewButton(firstGroup, ButtonOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "button." + prefix + ".save",
				Bounds:        Rect{Width: 18, Height: 2},
			},
			Command:  CommandID("scope." + prefix + ".save"),
			Mnemonic: "s",
			Default:  true,
		})
		if buttonErr != nil {
			t.Fatalf("NewButton(%s save) error = %v", prefix, buttonErr)
		}
		cancel, buttonErr := tx.NewButton(secondGroup, ButtonOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "button." + prefix + ".cancel",
				Bounds:        Rect{Width: 18, Height: 2},
			},
			Command:  CommandID("scope." + prefix + ".cancel"),
			Mnemonic: "c",
			Cancel:   true,
		})
		if buttonErr != nil {
			t.Fatalf("NewButton(%s cancel) error = %v", prefix, buttonErr)
		}
		return save, cancel
	}
	aSave, aCancel := newScopedButtons(scopeA, "a", 0)
	bSave, _ := newScopedButtons(scopeB, "b", 0)
	rootButton, err := tx.NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.root",
			Bounds:        Rect{X: 28, Y: 1, Width: 18, Height: 2},
		},
		Command: "scope.root",
	})
	if err != nil {
		t.Fatalf("NewButton(root) error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("scope construction Commit() error = %v", err)
	}

	if err := aSave.Focus(); err != nil {
		t.Fatalf("aSave.Focus() error = %v", err)
	}
	if completion := dispatchAltMnemonic(t, app, "scope-a-save", "s"); completion.Command != "scope.a.save" {
		t.Fatalf("scope A mnemonic completion = %+v", completion)
	}
	if _, err := app.DispatchKey(
		context.Background(), "keyboard", "scope-a-tab",
		KeyEvent{Kind: KeyEventPress, Key: KeyTab},
	); err != nil {
		t.Fatalf("DispatchKey(scope A Tab) error = %v", err)
	}
	if app.Focused() != aCancel {
		t.Fatalf("scope A Tab focus = %#v, want cancel", app.Focused())
	}
	if _, err := app.DispatchKey(
		context.Background(), "keyboard", "scope-a-tab-wrap",
		KeyEvent{Kind: KeyEventPress, Key: KeyTab},
	); err != nil {
		t.Fatalf("DispatchKey(scope A wrapped Tab) error = %v", err)
	}
	if app.Focused() != aSave {
		t.Fatalf("confined Tab escaped to %#v, want scope A save", app.Focused())
	}

	toggle := app.NewTransaction()
	if err := toggle.SetVisible(scopeA, false); err != nil {
		t.Fatalf("SetVisible(scope A false) error = %v", err)
	}
	if err := toggle.SetVisible(scopeB, true); err != nil {
		t.Fatalf("SetVisible(scope B true) error = %v", err)
	}
	if err := toggle.Commit(context.Background()); err != nil {
		t.Fatalf("atomic scope toggle Commit() error = %v", err)
	}
	if app.Focused() != bSave {
		t.Fatalf("focus after atomic scope toggle = %#v, want B save", app.Focused())
	}
	if completion := dispatchAltMnemonic(t, app, "scope-b-cancel", "c"); completion.Command != "scope.b.cancel" {
		t.Fatalf("scope B mnemonic completion = %+v", completion)
	}
	if app.Focused() == rootButton {
		t.Fatal("confined scope unexpectedly focused the root Button")
	}

	snapshot := app.Snapshot()
	if got := controlByKey(t, snapshot, "root").Details.Container.InputScope; got != InputScopeConfined {
		t.Fatalf("root input scope = %q", got)
	}
	if got := controlByKey(t, snapshot, "scope.a").Details.Container.InputScope; got != InputScopeConfined {
		t.Fatalf("scope A structural evidence = %q", got)
	}
	for index := range snapshot.Controls {
		if snapshot.Controls[index].Key == "scope.a" {
			snapshot.Controls[index].Details.Container.InputScope = InputScopeNone
		}
	}
	if got := controlByKey(
		t,
		app.Snapshot(),
		"scope.a",
	).Details.Container.InputScope; got != InputScopeConfined {
		t.Fatalf("core snapshot mutation changed input scope to %q", got)
	}
	if got := controlByKey(t, snapshot, "button.a.save").Details.Container; got != nil {
		t.Fatalf("leaf unexpectedly has ContainerDetails = %#v", got)
	}
}

func TestEscapingAndClippedInputScopes(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 8})
	registerActionCommand(t, app, "scope.root", "Root", true)
	registerActionCommand(t, app, "scope.escape", "Escape", true)
	registerActionCommand(t, app, "scope.clipped", "Clipped", true)
	rootButton, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{X: 1, Y: 1, Width: 12, Height: 2}},
		Command:      "scope.root",
	})
	if err != nil {
		t.Fatalf("NewButton(root) error = %v", err)
	}
	escaping, err := NewPanel(app.Root(), PanelOptions{
		Bounds:     Rect{X: 15, Y: 1, Width: 15, Height: 4},
		InputScope: InputScopeEscaping,
	})
	if err != nil {
		t.Fatalf("NewPanel(escaping) error = %v", err)
	}
	escapeButton, err := NewButton(escaping, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Width: 12, Height: 2}},
		Command:      "scope.escape",
	})
	if err != nil {
		t.Fatalf("NewButton(escaping) error = %v", err)
	}
	if err := escapeButton.Focus(); err != nil {
		t.Fatalf("escapeButton.Focus() error = %v", err)
	}
	if _, err := app.DispatchKey(
		context.Background(), "keyboard", "escape-scope-tab",
		KeyEvent{Kind: KeyEventPress, Key: KeyTab},
	); err != nil {
		t.Fatalf("DispatchKey(escaping Tab) error = %v", err)
	}
	if app.Focused() != rootButton {
		t.Fatalf("escaping Tab focus = %#v, want root Button", app.Focused())
	}

	clipped, err := NewPanel(app.Root(), PanelOptions{
		Bounds:     Rect{X: 80, Y: 1, Width: 15, Height: 4},
		InputScope: InputScopeConfined,
	})
	if err != nil {
		t.Fatalf("NewPanel(clipped) error = %v", err)
	}
	clippedButton, err := NewButton(clipped, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Width: 12, Height: 2}},
		Command:      "scope.clipped",
		Mnemonic:     "z",
	})
	if err != nil {
		t.Fatalf("NewButton(clipped) error = %v", err)
	}
	if err := clippedButton.Focus(); err != nil {
		t.Fatalf("clippedButton.Focus() error = %v", err)
	}
	if app.Focused() == clippedButton {
		t.Fatal("clipped input scope retained focus")
	}
	if completion := dispatchAltMnemonic(t, app, "clipped-scope", "z"); completion.Command != "" {
		t.Fatalf("clipped scope mnemonic completion = %+v", completion)
	}
}

func TestInputScopeRolesRouteWithinActiveBoundary(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 40, Height: 10}})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	for _, command := range []CommandID{
		"scope.default", "scope.cancel", "root.default",
	} {
		registerActionCommand(t, app, command, string(command), true)
	}
	scope, err := NewPanel(app.Root(), PanelOptions{
		Bounds:     Rect{Width: 24, Height: 8},
		InputScope: InputScopeConfined,
	})
	if err != nil {
		t.Fatalf("NewPanel(scope) error = %v", err)
	}
	scroll, err := NewScrollablePanel(scope, ScrollablePanelOptions{
		ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Width: 10, Height: 3}},
			State: ViewportState{
				ContentSize: Size{Width: 20, Height: 6},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewScrollablePanel() error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Y: 4, Width: 10, Height: 2}},
		Command:      "scope.default",
		Default:      true,
	}); err != nil {
		t.Fatalf("NewButton(scope default) error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{X: 11, Y: 4, Width: 10, Height: 2}},
		Command:      "scope.cancel",
		Cancel:       true,
	}); err != nil {
		t.Fatalf("NewButton(scope cancel) error = %v", err)
	}
	if _, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{X: 27, Width: 12, Height: 2}},
		Command:      "root.default",
		Default:      true,
	}); err != nil {
		t.Fatalf("NewButton(root default) error = %v", err)
	}
	if err := scroll.Focus(); err != nil {
		t.Fatalf("scroll.Focus() error = %v", err)
	}
	enter, err := app.DispatchKey(
		context.Background(), "keyboard", "scope-default",
		KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
	)
	if err != nil || enter.Command != "scope.default" {
		t.Fatalf("scoped Enter completion = %+v, %v", enter, err)
	}
	escape, err := app.DispatchKey(
		context.Background(), "keyboard", "scope-cancel",
		KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
	)
	if err != nil || escape.Command != "scope.cancel" {
		t.Fatalf("scoped Escape completion = %+v, %v", escape, err)
	}
}

func TestInputScopeValidationPreservesModalAndLocalUniqueness(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	registerActionCommand(t, app, "scope.one", "One", true)
	registerActionCommand(t, app, "scope.two", "Two", true)
	scope, err := NewPanel(app.Root(), PanelOptions{
		Bounds:     Rect{Width: 20, Height: 6},
		InputScope: InputScopeConfined,
	})
	if err != nil {
		t.Fatalf("NewPanel(scope) error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Width: 10, Height: 2}},
		Command:      "scope.one",
		Mnemonic:     "x",
		Default:      true,
	}); err != nil {
		t.Fatalf("NewButton(first) error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Y: 3, Width: 10, Height: 2}},
		Command:      "scope.two",
		Mnemonic:     "x",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate scoped mnemonic error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Y: 3, Width: 10, Height: 2}},
		Command:      "scope.two",
		Default:      true,
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate scoped default error = %v", err)
	}
	if _, err := NewButton(scope, ButtonOptions{
		PanelOptions: PanelOptions{
			Bounds:     Rect{Y: 3, Width: 10, Height: 2},
			InputScope: InputScopeConfined,
		},
		Command: "scope.two",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("leaf input scope error = %v", err)
	}
	if _, err := NewPanel(app.Root(), PanelOptions{
		InputScope: InputScopeMode("invalid"),
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("invalid input scope mode error = %v", err)
	}
	if _, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{InputScope: InputScopeConfined},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("explicit modal input scope error = %v", err)
	}
	modalApp, err := NewApp(AppOptions{
		Size: Size{Width: 30, Height: 8},
	})
	if err != nil {
		t.Fatalf("NewApp(modal) error = %v", err)
	}
	modal, err := NewModalPanel(modalApp.Root(), ModalPanelOptions{})
	if err != nil {
		t.Fatalf("NewModalPanel() error = %v", err)
	}
	if _, err := NewPanel(modal, PanelOptions{
		InputScope: InputScopeConfined,
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("nested non-modal scope error = %v", err)
	}
}
