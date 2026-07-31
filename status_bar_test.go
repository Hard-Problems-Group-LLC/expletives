package expletives

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func statusBarByKey(
	t *testing.T,
	snapshot Snapshot,
	key string,
) ControlSnapshot {
	t.Helper()
	control := controlByKey(t, snapshot, key)
	if control.Details.StatusBar == nil {
		t.Fatalf("control %q has no StatusBarDetails", key)
	}
	return control
}

func TestStatusBarConstructionRenderingAndCommandProjection(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 5})
	registerActionCommand(t, app, "app.exit", "Exit", true)
	if err := app.BindChord(
		Chord{Key: "x", Modifiers: []Key{KeyAlt}},
		CommandBinding{Command: "app.exit"},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	_, err := NewPanel(app.Root(), PanelOptions{
		AutomationKey: "content",
		Bounds:        Rect{Width: 28, Height: 5},
		Style:         "panel",
	})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.main"},
		Segments: []StatusSegment{
			{Key: "context", Text: "Ready", Priority: 5},
			{Key: "exit", Command: "app.exit", Priority: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	if _, ok := any(bar).(Container); ok {
		t.Fatal("StatusBar unexpectedly implements Container")
	}
	if got := bar.Bounds(); got != (Rect{Y: 4, Width: 28, Height: 1}) {
		t.Fatalf("StatusBar bounds = %+v", got)
	}
	if got := bar.MinimumSize(); got != (Size{Width: 1, Height: 1}) {
		t.Fatalf("StatusBar minimum = %+v", got)
	}
	if got := bar.Segments(); len(got) != 2 ||
		got[0].Text != "Ready" || got[1].Command != "app.exit" {
		t.Fatalf("Segments() = %#v", got)
	}

	snapshot := app.Snapshot()
	state := statusBarByKey(t, snapshot, "status.main")
	if state.Bounds != (Rect{Y: 4, Width: 28, Height: 1}) ||
		state.AbsoluteBounds != state.Bounds ||
		state.Parent != "root" ||
		state.Layout != "" ||
		len(state.Details.StatusBar.Segments) != 2 {
		t.Fatalf("StatusBar snapshot = %#v", state)
	}
	if got := rowText(snapshot, 4); !strings.HasPrefix(
		got,
		" Ready  Alt+X Exit ",
	) {
		t.Fatalf("StatusBar row = %q", got)
	}
	exit := state.Details.StatusBar.Segments[1]
	if !exit.Enabled || exit.Label != "Exit" || exit.Chord == nil ||
		exit.Chord.Key != "x" ||
		len(exit.Chord.Modifiers) != 1 ||
		exit.Chord.Modifiers[0] != KeyAlt ||
		!exit.Rendered || exit.Clipped {
		t.Fatalf("exit segment = %#v", exit)
	}
	shortcutCell, _ := snapshot.Frame.Cell(8, 4)
	if shortcutCell.Style != "status.shortcut" ||
		shortcutCell.Owner != state.ID {
		t.Fatalf("shortcut cell = %#v", shortcutCell)
	}
	contentState := controlByKey(t, snapshot, "content")
	if contentState.EffectiveClip.Height != 4 {
		t.Fatalf("content clip = %+v, want height 4", contentState.EffectiveClip)
	}
	if err := bar.SetVisible(false); err != nil {
		t.Fatalf("SetVisible(false) error = %v", err)
	}
	if got := controlByKey(
		t,
		app.Snapshot(),
		"content",
	).EffectiveClip.Height; got != 5 {
		t.Fatalf("hidden StatusBar content clip height = %d, want 5", got)
	}
	hiddenStatus := statusBarByKey(t, app.Snapshot(), "status.main")
	if hiddenStatus.Details.StatusBar.Segments[0].Rendered ||
		hiddenStatus.Details.StatusBar.Segments[1].Rendered {
		t.Fatal("hidden StatusBar reports rendered segments")
	}
	if err := bar.SetVisible(true); err != nil {
		t.Fatalf("SetVisible(true) error = %v", err)
	}

	updated := CommandDefinition{
		ID: "app.exit", Label: "Leave", Enabled: false,
		DisabledReason: "Not now", Checked: true, Automation: true,
	}
	if err := app.ReplaceCommand(updated); err != nil {
		t.Fatalf("ReplaceCommand() error = %v", err)
	}
	state = statusBarByKey(t, app.Snapshot(), "status.main")
	exit = state.Details.StatusBar.Segments[1]
	if exit.Enabled || exit.Label != "Leave" || !exit.Checked ||
		exit.DisabledReason != "Not now" {
		t.Fatalf("updated exit segment = %#v", exit)
	}
	cell, _ := app.Snapshot().Frame.Cell(8, 4)
	if cell.Style != "status.disabled" {
		t.Fatalf("disabled segment style = %q", cell.Style)
	}
	copied := app.Snapshot()
	copiedStatus := statusBarByKey(t, copied, "status.main")
	copiedStatus.Details.StatusBar.Segments[1].Label = "Mutated"
	copiedStatus.Details.StatusBar.Segments[1].Chord.Modifiers[0] = KeyShift
	fresh := statusBarByKey(t, app.Snapshot(), "status.main")
	if fresh.Details.StatusBar.Segments[1].Label != "Leave" ||
		fresh.Details.StatusBar.Segments[1].Chord.Modifiers[0] != KeyAlt {
		t.Fatal("StatusBar snapshot aliases returned segment or Chord storage")
	}
	if err := app.RemoveCommand("app.exit"); err != nil {
		t.Fatalf("RemoveCommand() error = %v", err)
	}
	exit = statusBarByKey(
		t,
		app.Snapshot(),
		"status.main",
	).Details.StatusBar.Segments[1]
	if exit.Enabled || exit.Label != "app.exit" ||
		exit.DisabledReason == "" || exit.Chord != nil {
		t.Fatalf("removed command segment = %#v", exit)
	}
}

func TestStatusBarPriorityClippingEmptyAndResize(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 8, Height: 2})
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.priority"},
		Segments: []StatusSegment{
			{Key: "low", Text: "Low", Priority: 0},
			{Key: "high", Text: "Important", Priority: 10},
			{Key: "tiny", Text: "X", Priority: -1},
		},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	details := statusBarByKey(
		t,
		app.Snapshot(),
		"status.priority",
	).Details.StatusBar
	if details.Segments[0].Rendered ||
		!details.Segments[1].Rendered ||
		!details.Segments[1].Clipped ||
		details.Segments[1].Bounds != (Rect{Width: 8, Height: 1}) ||
		details.Segments[2].Rendered {
		t.Fatalf("narrow priority details = %#v", details.Segments)
	}
	if got := rowText(app.Snapshot(), 1); got != "Importan" {
		t.Fatalf("clipped row = %q, want %q", got, "Importan")
	}

	if err := app.SetSize(Size{Width: 18, Height: 3}); err != nil {
		t.Fatalf("SetSize(18x3) error = %v", err)
	}
	state := statusBarByKey(t, app.Snapshot(), "status.priority")
	if state.Bounds != (Rect{Y: 2, Width: 18, Height: 1}) {
		t.Fatalf("resized StatusBar bounds = %+v", state.Bounds)
	}
	if !state.Details.StatusBar.Segments[0].Rendered ||
		!state.Details.StatusBar.Segments[1].Rendered ||
		state.Details.StatusBar.Segments[2].Rendered {
		t.Fatalf("resized priority details = %#v", state.Details.StatusBar.Segments)
	}

	if err := bar.SetSegments(nil); err != nil {
		t.Fatalf("SetSegments(nil) error = %v", err)
	}
	state = statusBarByKey(t, app.Snapshot(), "status.priority")
	if len(state.Details.StatusBar.Segments) != 0 ||
		rowText(app.Snapshot(), 2) != strings.Repeat(" ", 18) {
		t.Fatalf("empty StatusBar snapshot = %#v", state)
	}

	if err := app.SetSize(Size{}); err != nil {
		t.Fatalf("SetSize(empty) error = %v", err)
	}
	state = statusBarByKey(t, app.Snapshot(), "status.priority")
	if state.Bounds != (Rect{}) || state.EffectiveClip != (Rect{}) {
		t.Fatalf("empty surface StatusBar = %#v", state)
	}
}

func TestStatusBarChromeReservationVisibilityDestroyAndConstraints(
	t *testing.T,
) {
	t.Parallel()
	app, err := NewApp(AppOptions{
		Size: Size{Width: 30, Height: 8},
		RootConstraints: RootConstraints{
			Minimum: Size{Width: 8, Height: 2},
			Maximum: Size{Width: 12, Height: 4},
		},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	child, err := NewPanel(app.Root(), PanelOptions{
		AutomationKey: "content",
		Bounds:        Rect{Width: 12, Height: 4},
	})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.chrome"},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	snapshot := app.Snapshot()
	root := controlByKey(t, snapshot, "root")
	status := statusBarByKey(t, snapshot, "status.chrome")
	if root.Bounds != (Rect{X: 9, Y: 2, Width: 12, Height: 4}) ||
		status.Bounds != (Rect{Y: 7, Width: 30, Height: 1}) {
		t.Fatalf("constrained chrome root=%+v status=%+v", root.Bounds, status.Bounds)
	}
	if got := controlByKey(t, snapshot, "content").EffectiveClip.Height; got != 4 {
		t.Fatalf("inset root content clip height = %d, want 4", got)
	}
	if err := bar.SetVisible(false); err != nil {
		t.Fatalf("SetVisible(false) error = %v", err)
	}
	status = statusBarByKey(t, app.Snapshot(), "status.chrome")
	if status.Visible || status.Details.StatusBar == nil {
		t.Fatalf("hidden StatusBar = %#v", status)
	}
	if err := bar.SetVisible(true); err != nil {
		t.Fatalf("SetVisible(true) error = %v", err)
	}
	if err := bar.Destroy(); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if _, found := app.ControlByAutomationKey("status.chrome"); found {
		t.Fatal("destroyed StatusBar remains discoverable")
	}
	if got := child.Bounds(); got != (Rect{Width: 12, Height: 4}) {
		t.Fatalf("logical child bounds changed = %+v", got)
	}
}

func TestStatusBarAndMenuTinySurfacePriority(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 1})
	registerActionCommand(t, app, "app.noop", "Noop", true)
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.tiny"},
		Segments: []StatusSegment{{
			Key: "context", Text: "Status", Priority: 1,
		}},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	menu, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "item.noop", Kind: MenuItemCommand, Command: "app.noop",
	}}})
	if err != nil {
		t.Fatalf("NewMenu() error = %v", err)
	}
	menuBar, err := NewMenuBar(app.Root(), MenuBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "menu.tiny"},
		Items: []MenuItem{{
			Key: "menu.file", Kind: MenuItemSubmenu, Label: "File",
			Mnemonic: "f", Menu: menu,
		}},
	})
	if err != nil {
		t.Fatalf("NewMenuBar() error = %v", err)
	}
	snapshot := app.Snapshot()
	status := statusBarByKey(t, snapshot, "status.tiny")
	menuState := menuBarByKey(t, snapshot, "menu.tiny")
	if status.Bounds != (Rect{Width: 12, Height: 1}) ||
		menuState.Bounds != status.Bounds {
		t.Fatalf("tiny bounds status=%+v menu=%+v", status.Bounds, menuState.Bounds)
	}
	for column := range snapshot.Frame.Size.Width {
		cell, _ := snapshot.Frame.Cell(column, 0)
		if cell.Owner != menuState.ID {
			t.Fatalf("cell %d owner = %q, want MenuBar %q", column, cell.Owner, menuState.ID)
		}
	}
	if err := menuBar.Destroy(); err != nil {
		t.Fatalf("Destroy(MenuBar) error = %v", err)
	}
	snapshot = app.Snapshot()
	status = statusBarByKey(t, snapshot, "status.tiny")
	for column := range snapshot.Frame.Size.Width {
		cell, _ := snapshot.Frame.Cell(column, 0)
		if cell.Owner != status.ID {
			t.Fatalf("StatusBar cell %d owner = %q, want %q", column, cell.Owner, status.ID)
		}
	}
	if got := bar.Bounds(); got != (Rect{Width: 12, Height: 1}) {
		t.Fatalf("StatusBar bounds after MenuBar destroy = %+v", got)
	}
}

func TestStatusBarValidationAtomicMutationAndUniqueness(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 3})
	registerActionCommand(t, app, "app.valid", "Valid", true)
	parent, err := NewPanel(app.Root(), PanelOptions{
		AutomationKey: "ordinary.parent",
	})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	if _, err := NewStatusBar(parent, StatusBarOptions{}); err == nil {
		t.Fatal("non-root StatusBar parent unexpectedly succeeded")
	}
	if _, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{
			Bounds: Rect{Width: 1, Height: 1},
		},
	}); err == nil {
		t.Fatal("caller-set StatusBar bounds unexpectedly succeeded")
	}
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.valid"},
		Segments: []StatusSegment{{
			Key: "valid", Command: "app.valid",
		}},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	if _, err := NewStatusBar(app.Root(), StatusBarOptions{}); err == nil {
		t.Fatal("second StatusBar unexpectedly succeeded")
	}
	if err := bar.SetBounds(Rect{Width: 1, Height: 1}); err == nil {
		t.Fatal("SetBounds(StatusBar) unexpectedly succeeded")
	}
	if err := bar.SetMinimumSize(Size{Width: 1, Height: 1}); err == nil {
		t.Fatal("SetMinimumSize(StatusBar) unexpectedly succeeded")
	}
	if err := bar.SetSegments(
		make([]StatusSegment, MaxStatusBarSegments+1),
	); err == nil {
		t.Fatal("oversized StatusBar inventory unexpectedly succeeded")
	}
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(bar, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(StatusBar) builder error = %v", err)
	}
	layoutTx := app.NewTransaction()
	if err := layoutTx.SetLayout(app.Root(), layout); err != nil {
		t.Fatalf("SetLayout() builder error = %v", err)
	}
	if err := layoutTx.Commit(context.Background()); !errors.Is(
		err,
		ErrInvalidLayout,
	) {
		t.Fatalf("StatusBar Layout commit error = %v", err)
	}
	for name, segments := range map[string][]StatusSegment{
		"missing key": {{Text: "text"}},
		"duplicate key": {
			{Key: "same", Text: "one"},
			{Key: "same", Text: "two"},
		},
		"neither value": {{Key: "empty"}},
		"both values": {{
			Key: "both", Text: "text", Command: "app.valid",
		}},
		"unregistered": {{Key: "bad", Command: "app.missing"}},
	} {
		t.Run(name, func(t *testing.T) {
			before := app.Snapshot().Sequence
			if err := bar.SetSegments(segments); err == nil {
				t.Fatal("invalid SetSegments() unexpectedly succeeded")
			}
			if app.Snapshot().Sequence != before {
				t.Fatal("invalid mutation published a snapshot")
			}
			if got := bar.Segments(); len(got) != 1 || got[0].Key != "valid" {
				t.Fatalf("invalid mutation changed segments = %#v", got)
			}
		})
	}

	tx := app.NewTransaction()
	if err := tx.SetStatusSegments(bar, []StatusSegment{{
		Key: "replacement", Text: "Replacement",
	}}); err != nil {
		t.Fatalf("SetStatusSegments() error = %v", err)
	}
	if err := tx.SetSize(Size{Width: 24, Height: 4}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	before := app.Snapshot().Sequence
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if app.Snapshot().Sequence != before+1 ||
		app.Size() != (Size{Width: 24, Height: 4}) ||
		bar.Segments()[0].Key != "replacement" {
		t.Fatalf("atomic StatusBar transaction did not publish complete state")
	}
}

func TestStatusBarConcurrentReadersAndWriters(t *testing.T) {
	app := mustApp(t, Size{Width: 30, Height: 4})
	bar, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.concurrent"},
		Segments:     []StatusSegment{{Key: "initial", Text: "Initial"}},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	var wait sync.WaitGroup
	errs := make(chan error, 2)
	wait.Add(2)
	go func() {
		defer wait.Done()
		for index := 0; index < 100; index++ {
			key := "even"
			if index%2 != 0 {
				key = "odd"
			}
			if err := bar.SetSegments([]StatusSegment{{
				Key: key, Text: strings.ToUpper(key),
			}}); err != nil {
				errs <- err
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		for range 100 {
			snapshot := app.Snapshot()
			state := statusBarByKey(t, snapshot, "status.concurrent")
			if len(state.Details.StatusBar.Segments) != 1 {
				errs <- errors.New("observed partial StatusBar inventory")
				return
			}
		}
	}()
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
