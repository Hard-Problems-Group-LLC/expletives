package expletives

import (
	"context"
	"errors"
	"testing"
)

func newModalTestApp(t *testing.T, size Size) *App {
	t.Helper()
	app, err := NewApp(AppOptions{Size: size, Scenario: "modal.test"})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	for _, command := range []CommandID{"outside", "modal.accept"} {
		if err := app.RegisterCommand(CommandDefinition{
			ID: command, Label: string(command), Enabled: true,
		}); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", command, err)
		}
	}
	return app
}

func modalDetailsByKey(t *testing.T, app *App, key string) *ModalPanelDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.ModalPanel
	if details == nil {
		t.Fatalf("control %q has no ModalPanel details", key)
	}
	return details
}

func TestModalPanelLifecycleFocusGeometryAndResult(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 40, Height: 15})
	outside, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "outside",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Command: "outside",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	if err := outside.Focus(); err != nil {
		t.Fatalf("outside.Focus() error = %v", err)
	}
	modal, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.primary",
			Bounds:        Rect{Width: 20, Height: 7},
		},
		Title: "Modal",
	})
	if err != nil {
		t.Fatalf("NewModalPanel() error = %v", err)
	}
	field, err := NewTextField(modal, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.field",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Text: "value",
	})
	if err != nil {
		t.Fatalf("NewTextField() error = %v", err)
	}
	before := controlByKey(t, app.Snapshot(), "modal.primary")
	if before.Visible || before.Details.ModalPanel.Lifecycle != ModalLifecycleInactive {
		t.Fatalf("inactive modal snapshot = %+v", before)
	}
	done := modal.Done()
	if done == nil {
		t.Fatal("ModalPanel.Done() returned nil")
	}
	if err := modal.Show(field); err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if !modal.Active() || app.Focused() != field {
		t.Fatalf("active=%t focused=%v", modal.Active(), app.Focused())
	}
	view := controlByKey(t, app.Snapshot(), "modal.primary")
	if view.Bounds != (Rect{X: 10, Y: 4, Width: 20, Height: 7}) ||
		view.AbsoluteBounds != view.Bounds || !view.Visible {
		t.Fatalf("active modal geometry = %+v", view)
	}
	details := view.Details.ModalPanel
	if details == nil || !details.Active || !details.Top ||
		details.StackIndex != 0 || details.StackDepth != 1 ||
		details.InitialFocus != field.ID() ||
		details.SavedFocus != outside.ID() || details.Degraded {
		t.Fatalf("active modal details = %+v", details)
	}
	shadow, found := app.Snapshot().Frame.Cell(30, 5)
	if !found || shadow.Style != "modal_panel.shadow" || shadow.Owner != modal.ID() {
		t.Fatalf("right shadow cell = %+v, found=%t", shadow, found)
	}
	if err := outside.Focus(); !errors.Is(err, ErrNotFocusable) {
		t.Fatalf("outside.Focus() while modal active error = %v", err)
	}
	result := ModalResult{Reason: ModalAccepted, Action: "modal.accept"}
	if err := modal.Close(result); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case <-done:
	default:
		t.Fatal("Done was not closed")
	}
	if modal.Active() || app.Focused() != outside {
		t.Fatalf("closed active=%t focused=%v", modal.Active(), app.Focused())
	}
	if got, ready := modal.Result(); !ready || got != result {
		t.Fatalf("Result() = %+v, %t", got, ready)
	}
	closed := controlByKey(t, app.Snapshot(), "modal.primary")
	if closed.Visible || closed.Details.ModalPanel.Result == nil ||
		*closed.Details.ModalPanel.Result != result ||
		closed.Details.ModalPanel.Lifecycle != ModalLifecycleClosed {
		t.Fatalf("closed modal snapshot = %+v", closed)
	}
	if err := modal.Show(field); !errors.Is(err, ErrModalState) {
		t.Fatalf("second Show() error = %v", err)
	}
	if err := modal.Close(result); !errors.Is(err, ErrModalState) {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestModalPanelExplicitNestingAndLIFO(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 50, Height: 18})
	outer, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.outer",
			Bounds:        Rect{Width: 28, Height: 10},
		},
		Title: "Outer",
	})
	if err != nil {
		t.Fatalf("NewModalPanel(outer) error = %v", err)
	}
	outerField, err := NewTextField(outer, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.outer.field",
			Bounds:        Rect{X: 1, Y: 1, Width: 10, Height: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewTextField(outer) error = %v", err)
	}
	inner, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.inner",
			Bounds:        Rect{Width: 18, Height: 6},
		},
		Title:       "Inner",
		NestedOwner: outer,
	})
	if err != nil {
		t.Fatalf("NewModalPanel(inner) error = %v", err)
	}
	innerField, err := NewTextField(inner, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.inner.field",
			Bounds:        Rect{X: 1, Y: 1, Width: 10, Height: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewTextField(inner) error = %v", err)
	}
	unowned, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.unowned",
			Bounds:        Rect{Width: 12, Height: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewModalPanel(unowned) error = %v", err)
	}
	if err := outer.Show(outerField); err != nil {
		t.Fatalf("outer.Show() error = %v", err)
	}
	if err := unowned.Show(nil); !errors.Is(err, ErrModalState) {
		t.Fatalf("unowned nested Show() error = %v", err)
	}
	if err := inner.Show(innerField); err != nil {
		t.Fatalf("inner.Show() error = %v", err)
	}
	if app.Focused() != innerField {
		t.Fatalf("nested focus = %v", app.Focused())
	}
	outerDetails := modalDetailsByKey(t, app, "modal.outer")
	innerDetails := modalDetailsByKey(t, app, "modal.inner")
	if outerDetails.Top || outerDetails.StackIndex != 0 ||
		innerDetails.NestedOwner != outer.ID() || !innerDetails.Top ||
		innerDetails.StackIndex != 1 || innerDetails.StackDepth != 2 {
		t.Fatalf("outer=%+v inner=%+v", outerDetails, innerDetails)
	}
	if err := outer.Close(ModalResult{Reason: ModalCancelled}); !errors.Is(err, ErrModalState) {
		t.Fatalf("non-top outer.Close() error = %v", err)
	}
	if err := inner.Close(ModalResult{Reason: ModalBack}); err != nil {
		t.Fatalf("inner.Close() error = %v", err)
	}
	if app.Focused() != outerField || !modalDetailsByKey(t, app, "modal.outer").Top {
		t.Fatalf("focus/top after inner close = %v/%+v", app.Focused(), modalDetailsByKey(t, app, "modal.outer"))
	}
	if err := outer.Close(ModalResult{Reason: ModalCancelled}); err != nil {
		t.Fatalf("outer.Close() error = %v", err)
	}
}

func TestModalPanelResizeMinimumAndLifecycleOwnedMutations(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 10, Height: 5})
	modal, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.resize",
			Bounds:        Rect{Width: 5, Height: 3},
		},
		Title: "Resize",
	})
	if err != nil {
		t.Fatalf("NewModalPanel() error = %v", err)
	}
	child, err := NewPanel(modal, PanelOptions{
		AutomationKey: "modal.resize.content",
		MinimumSize:   Size{Width: 18, Height: 6},
	})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(child, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("AddPanel() error = %v", err)
	}
	if err := modal.SetLayout(layout); err != nil {
		t.Fatalf("SetLayout() error = %v", err)
	}
	if err := modal.Show(nil); err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	first := modalDetailsByKey(t, app, "modal.resize")
	if first.RequiredMinimum != (Size{Width: 20, Height: 8}) ||
		first.ResolvedBounds != (Rect{Width: 10, Height: 5}) || !first.Degraded {
		t.Fatalf("small modal details = %+v", first)
	}
	if err := modal.SetBounds(Rect{Width: 30, Height: 10}); !errors.Is(err, ErrModalState) {
		t.Fatalf("SetBounds() error = %v", err)
	}
	if err := modal.SetVisible(false); !errors.Is(err, ErrModalState) {
		t.Fatalf("SetVisible() error = %v", err)
	}
	if err := app.SetSize(Size{Width: 30, Height: 15}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	resized := modalDetailsByKey(t, app, "modal.resize")
	if resized.ResolvedBounds != (Rect{X: 5, Y: 3, Width: 20, Height: 8}) ||
		resized.Degraded {
		t.Fatalf("resized modal details = %+v", resized)
	}
}

func TestDestroyingLowerModalClosesNestedStack(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 40, Height: 12})
	outer, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.destroy.outer",
			Bounds:        Rect{Width: 20, Height: 8},
		},
	})
	if err != nil {
		t.Fatalf("NewModalPanel(outer) error = %v", err)
	}
	inner, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "modal.destroy.inner",
			Bounds:        Rect{Width: 12, Height: 5},
		},
		NestedOwner: outer,
	})
	if err != nil {
		t.Fatalf("NewModalPanel(inner) error = %v", err)
	}
	outerDone, innerDone := outer.Done(), inner.Done()
	if err := outer.Show(nil); err != nil {
		t.Fatalf("outer.Show() error = %v", err)
	}
	if err := inner.Show(nil); err != nil {
		t.Fatalf("inner.Show() error = %v", err)
	}
	if err := outer.Destroy(); err != nil {
		t.Fatalf("outer.Destroy() error = %v", err)
	}
	for name, done := range map[string]<-chan struct{}{
		"outer": outerDone,
		"inner": innerDone,
	} {
		select {
		case <-done:
		default:
			t.Fatalf("%s Done was not closed", name)
		}
	}
	if result, ready := outer.Result(); !ready || result.Reason != ModalDestroyed {
		t.Fatalf("outer Result() = %+v, %t", result, ready)
	}
	if result, ready := inner.Result(); !ready || result.Reason != ModalDestroyed {
		t.Fatalf("inner Result() = %+v, %t", result, ready)
	}
}

func TestModalCommandScopeAndExplicitGlobalPolicy(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 40, Height: 12})
	for _, definition := range []CommandDefinition{
		{ID: "global.blocked", Enabled: true},
		{
			ID: "global.allowed", Enabled: true,
			ModalPolicy: CommandModalAllowed,
		},
	} {
		if err := app.RegisterCommand(definition); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", definition.ID, err)
		}
	}
	if err := app.RegisterCommand(CommandDefinition{
		ID: "invalid.policy", Enabled: true,
		ModalPolicy: CommandModalPolicy("invalid"),
	}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid modal policy error = %v", err)
	}
	outside, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "scope.outside",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Command: "outside",
	})
	if err != nil {
		t.Fatalf("NewButton(outside) error = %v", err)
	}
	modal, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "scope.modal",
			Bounds:        Rect{Width: 20, Height: 7},
		},
	})
	if err != nil {
		t.Fatalf("NewModalPanel() error = %v", err)
	}
	inside, err := NewButton(modal, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "scope.inside",
			Bounds:        Rect{X: 1, Y: 1, Width: 12, Height: 1},
		},
		Command: "modal.accept",
	})
	if err != nil {
		t.Fatalf("NewButton(inside) error = %v", err)
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
	if err := modal.Show(inside); err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	blocked, err := app.InvokeCommand(
		context.Background(), "test", "blocked", "global.blocked", "",
	)
	if err != nil || blocked.Outcome != OutcomeRejected ||
		blocked.Code != "modal_scope" || len(routed) != 0 {
		t.Fatalf("blocked = %+v error=%v routed=%v", blocked, err, routed)
	}
	targetedOutside, err := app.InvokeCommand(
		context.Background(), "test", "outside", "global.allowed", outside.ID(),
	)
	if err != nil || targetedOutside.Outcome != OutcomeRejected ||
		targetedOutside.Code != "modal_scope" || len(routed) != 0 {
		t.Fatalf(
			"targeted outside = %+v error=%v routed=%v",
			targetedOutside,
			err,
			routed,
		)
	}
	allowed, err := app.InvokeCommand(
		context.Background(), "test", "allowed", "global.allowed", "",
	)
	if err != nil || allowed.Outcome != OutcomeApplied || len(routed) != 1 ||
		routed[0].ID != "global.allowed" || routed[0].Target != "" {
		t.Fatalf("allowed = %+v error=%v routed=%v", allowed, err, routed)
	}
	local, err := inside.Activate(context.Background(), "test", "local")
	if err != nil || local.Outcome != OutcomeApplied || len(routed) != 2 ||
		routed[1].ID != "modal.accept" || routed[1].Target != inside.ID() {
		t.Fatalf("local = %+v error=%v routed=%v", local, err, routed)
	}
}

func TestAllowedInterruptClosesModalAndApp(t *testing.T) {
	app := newModalTestApp(t, Size{Width: 30, Height: 10})
	if err := app.RegisterCommand(CommandDefinition{
		ID: "app.interrupt", Enabled: true,
		ModalPolicy: CommandModalAllowed,
	}); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}
	modal, err := NewModalPanel(app.Root(), ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "interrupt.modal",
			Bounds:        Rect{Width: 16, Height: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewModalPanel() error = %v", err)
	}
	if err := modal.Show(nil); err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if err := app.SetCommandRouter(func(
		context.Context,
		Command,
	) CommandResult {
		return CommandResult{Outcome: OutcomeInterrupted}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	completion, err := app.InvokeCommand(
		context.Background(), "test", "interrupt", "app.interrupt", "",
	)
	if err != nil || completion.Outcome != OutcomeInterrupted {
		t.Fatalf("InvokeCommand() = %+v, %v", completion, err)
	}
	result, ready := modal.Result()
	if !ready || result.Reason != ModalInterrupted ||
		result.Action != "app.interrupt" || modal.Active() {
		t.Fatalf("modal result = %+v ready=%t active=%t", result, ready, modal.Active())
	}
	if snapshot := app.Snapshot(); !snapshot.Final {
		t.Fatalf("final snapshot = %+v", snapshot)
	}
}
