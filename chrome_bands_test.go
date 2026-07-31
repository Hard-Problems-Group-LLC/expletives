package expletives

import (
	"context"
	"errors"
	"testing"
)

func newTestMenuBar(t *testing.T, app *App) *MenuBar {
	t.Helper()
	registerActionCommand(t, app, "chrome.noop", "Noop", true)
	menu, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "chrome.noop", Kind: MenuItemCommand, Command: "chrome.noop",
	}}})
	if err != nil {
		t.Fatalf("NewMenu() error = %v", err)
	}
	bar, err := NewMenuBar(app.Root(), MenuBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "menu.chrome"},
		Items: []MenuItem{{
			Key: "chrome.root", Kind: MenuItemSubmenu, Label: "Chrome",
			Mnemonic: "c", Menu: menu,
		}},
	})
	if err != nil {
		t.Fatalf("NewMenuBar() error = %v", err)
	}
	return bar
}

func TestHeaderFooterCombinedOrderingLifecycleAndPainting(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 10})
	newTestMenuBar(t, app)
	if _, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.chrome"},
	}); err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	content, err := NewPanel(app.Root(), PanelOptions{
		AutomationKey: "chrome.content",
		Bounds:        Rect{Width: 20, Height: 10},
	})
	if err != nil {
		t.Fatalf("NewPanel(content) error = %v", err)
	}
	headerOne, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.one"},
	})
	if err != nil {
		t.Fatalf("NewHeader(one) error = %v", err)
	}
	headerTwo, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.two"},
	})
	if err != nil {
		t.Fatalf("NewHeader(two) error = %v", err)
	}
	footerOne, err := NewFooter(app.Root(), FooterOptions{
		PanelOptions: PanelOptions{AutomationKey: "footer.one"},
	})
	if err != nil {
		t.Fatalf("NewFooter(one) error = %v", err)
	}
	footerTwo, err := NewFooter(app.Root(), FooterOptions{
		PanelOptions: PanelOptions{AutomationKey: "footer.two"},
	})
	if err != nil {
		t.Fatalf("NewFooter(two) error = %v", err)
	}
	if _, ok := any(headerOne).(Container); !ok {
		t.Fatal("Header does not implement Container")
	}
	if _, ok := any(footerOne).(Container); !ok {
		t.Fatal("Footer does not implement Container")
	}
	if got := headerOne.MinimumSize(); got != (Size{Height: 1}) {
		t.Fatalf("Header minimum = %+v", got)
	}

	snapshot := app.Snapshot()
	for key, want := range map[string]Rect{
		"header.one": {Y: 1, Width: 20, Height: 1},
		"header.two": {Y: 2, Width: 20, Height: 1},
		"footer.one": {Y: 8, Width: 20, Height: 1},
		"footer.two": {Y: 7, Width: 20, Height: 1},
	} {
		control := controlByKey(t, snapshot, key)
		if control.Bounds != want || control.AbsoluteBounds != want ||
			control.Parent != "root" ||
			control.Details.Container == nil {
			t.Fatalf("%s snapshot = %#v, want bounds %+v", key, control, want)
		}
		cell, ok := snapshot.Frame.Cell(0, want.Y)
		if !ok || cell.Owner != control.ID {
			t.Fatalf("%s does not own first row cell: %#v", key, cell)
		}
	}
	if got := controlByKey(
		t,
		snapshot,
		"chrome.content",
	).EffectiveClip; got != (Rect{Y: 3, Width: 20, Height: 4}) {
		t.Fatalf("content clip = %+v, want y=3 height=4", got)
	}

	if err := headerOne.SetVisible(false); err != nil {
		t.Fatalf("Header.SetVisible(false) error = %v", err)
	}
	snapshot = app.Snapshot()
	if controlByKey(t, snapshot, "header.one").Bounds != (Rect{}) ||
		controlByKey(t, snapshot, "header.two").Bounds.Y != 1 ||
		controlByKey(t, snapshot, "chrome.content").EffectiveClip !=
			(Rect{Y: 2, Width: 20, Height: 5}) {
		t.Fatal("hiding Header did not atomically compact chrome")
	}
	if err := headerOne.SetVisible(true); err != nil {
		t.Fatalf("Header.SetVisible(true) error = %v", err)
	}
	if err := footerOne.Destroy(); err != nil {
		t.Fatalf("Footer.Destroy() error = %v", err)
	}
	snapshot = app.Snapshot()
	if controlByKey(t, snapshot, "footer.two").Bounds.Y != 8 {
		t.Fatal("remaining Footer did not compact next to StatusBar")
	}
	if content.Bounds() != (Rect{Width: 20, Height: 10}) {
		t.Fatal("chrome changed retained logical content bounds")
	}
	if err := footerTwo.SetVisible(false); err != nil {
		t.Fatalf("Footer.SetVisible(false) error = %v", err)
	}
	if controlByKey(
		t,
		app.Snapshot(),
		"chrome.content",
	).EffectiveClip.Height != 6 {
		t.Fatal("hiding final Footer did not return its row")
	}
	_ = headerTwo
}

func TestApplicationClientAreaSectionBoundaries(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 10})
	menu := newTestMenuBar(t, app)
	status, err := NewStatusBar(app.Root(), StatusBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "status.client-area"},
	})
	if err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	header, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.client-area"},
	})
	if err != nil {
		t.Fatalf("NewHeader() error = %v", err)
	}
	footer, err := NewFooter(app.Root(), FooterOptions{
		PanelOptions: PanelOptions{AutomationKey: "footer.client-area"},
	})
	if err != nil {
		t.Fatalf("NewFooter() error = %v", err)
	}

	setVisible := func(
		menuVisible, headerVisible, footerVisible, statusVisible bool,
	) {
		t.Helper()
		if err := menu.SetVisible(menuVisible); err != nil {
			t.Fatalf("MenuBar.SetVisible(%t) error = %v", menuVisible, err)
		}
		if err := header.SetVisible(headerVisible); err != nil {
			t.Fatalf("Header.SetVisible(%t) error = %v", headerVisible, err)
		}
		if err := footer.SetVisible(footerVisible); err != nil {
			t.Fatalf("Footer.SetVisible(%t) error = %v", footerVisible, err)
		}
		if err := status.SetVisible(statusVisible); err != nil {
			t.Fatalf("StatusBar.SetVisible(%t) error = %v", statusVisible, err)
		}
	}
	clientArea := func() Rect {
		t.Helper()
		app.mu.RLock()
		defer app.mu.RUnlock()
		return app.applicationContentRectLocked()
	}
	tests := []struct {
		name                         string
		menu, header, footer, status bool
		want                         Rect
	}{
		{"all sections", true, true, true, true,
			Rect{Y: 2, Width: 20, Height: 6}},
		{"menu and status", true, false, false, true,
			Rect{Y: 1, Width: 20, Height: 8}},
		{"header and footer", false, true, true, false,
			Rect{Y: 1, Width: 20, Height: 8}},
		{"header and status", false, true, false, true,
			Rect{Y: 1, Width: 20, Height: 8}},
		{"menu and footer", true, false, true, false,
			Rect{Y: 1, Width: 20, Height: 8}},
		{"no upper sections", false, false, true, true,
			Rect{Width: 20, Height: 8}},
		{"no lower sections", true, true, false, false,
			Rect{Y: 2, Width: 20, Height: 8}},
		{"no sections", false, false, false, false,
			Rect{Width: 20, Height: 10}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setVisible(test.menu, test.header, test.footer, test.status)
			if got := clientArea(); got != test.want {
				t.Fatalf(
					"Application Client Area = %+v, want %+v",
					got,
					test.want,
				)
			}
		})
	}
}

func TestHeaderFooterTinySurfaceAllocation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 6})
	newTestMenuBar(t, app)
	if _, err := NewStatusBar(app.Root(), StatusBarOptions{}); err != nil {
		t.Fatalf("NewStatusBar() error = %v", err)
	}
	for _, key := range []string{"one", "two"} {
		if _, err := NewHeader(app.Root(), HeaderOptions{
			PanelOptions: PanelOptions{AutomationKey: "header." + key},
		}); err != nil {
			t.Fatalf("NewHeader(%s) error = %v", key, err)
		}
	}
	for _, key := range []string{"one", "two"} {
		if _, err := NewFooter(app.Root(), FooterOptions{
			PanelOptions: PanelOptions{AutomationKey: "footer." + key},
		}); err != nil {
			t.Fatalf("NewFooter(%s) error = %v", key, err)
		}
	}
	assertBounds := func(size Size, wants map[string]Rect) {
		t.Helper()
		if err := app.SetSize(size); err != nil {
			t.Fatalf("SetSize(%+v) error = %v", size, err)
		}
		snapshot := app.Snapshot()
		for key, want := range wants {
			if got := controlByKey(t, snapshot, key).Bounds; got != want {
				t.Fatalf("%+v %s bounds = %+v, want %+v", size, key, got, want)
			}
		}
	}
	assertBounds(Size{Width: 4, Height: 6}, map[string]Rect{
		"header.one": {Y: 1, Width: 4, Height: 1},
		"header.two": {Y: 2, Width: 4, Height: 1},
		"footer.one": {Y: 4, Width: 4, Height: 1},
		"footer.two": {Y: 3, Width: 4, Height: 1},
	})
	assertBounds(Size{Width: 4, Height: 3}, map[string]Rect{
		"header.one": {Y: 1, Width: 4, Height: 1},
		"header.two": {},
		"footer.one": {},
		"footer.two": {},
	})
	assertBounds(Size{Width: 4, Height: 2}, map[string]Rect{
		"header.one": {},
		"header.two": {},
		"footer.one": {},
		"footer.two": {},
	})
	assertBounds(Size{Width: 4, Height: 1}, map[string]Rect{
		"header.one": {},
		"header.two": {},
		"footer.one": {},
		"footer.two": {},
	})
	if got := len(app.Snapshot().Overflows); got != 0 {
		t.Fatalf("unallocated chrome bands produced %d overflows", got)
	}
}

func TestHeaderFooterLayoutCompatibilityAndAtomicRejection(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 4})
	header, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.layout"},
	})
	if err != nil {
		t.Fatalf("NewHeader() error = %v", err)
	}
	left, err := NewLabel(header, LabelOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.left"},
		Text:         "Left",
	})
	if err != nil {
		t.Fatalf("NewLabel(left) error = %v", err)
	}
	right, err := NewLabel(header, LabelOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.right"},
		Text:         "Right",
	})
	if err != nil {
		t.Fatalf("NewLabel(right) error = %v", err)
	}
	box, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: "layout.header",
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	for _, label := range []Control{left, right} {
		if err := box.AddPanel(label, LayoutItemOptions{Grow: 1}); err != nil {
			t.Fatalf("AddPanel() error = %v", err)
		}
	}
	if err := header.SetLayout(box); err != nil {
		t.Fatalf("Header.SetLayout(horizontal) error = %v", err)
	}
	snapshot := app.Snapshot()
	if controlByKey(t, snapshot, "header.left").Bounds.Height != 1 ||
		controlByKey(t, snapshot, "header.right").Bounds.Height != 1 {
		t.Fatal("compatible Header Layout did not arrange one-row children")
	}

	footer, err := NewFooter(app.Root(), FooterOptions{
		PanelOptions: PanelOptions{AutomationKey: "footer.grid"},
	})
	if err != nil {
		t.Fatalf("NewFooter() error = %v", err)
	}
	first, err := NewPanel(footer, PanelOptions{
		AutomationKey: "footer.first", MinimumSize: Size{Width: 8, Height: 1},
	})
	if err != nil {
		t.Fatalf("NewPanel(first) error = %v", err)
	}
	second, err := NewPanel(footer, PanelOptions{
		AutomationKey: "footer.second", MinimumSize: Size{Width: 8, Height: 1},
	})
	if err != nil {
		t.Fatalf("NewPanel(second) error = %v", err)
	}
	grid, err := NewGridLayout(GridLayoutOptions{
		AutomationKey: "layout.footer", Columns: 2,
	})
	if err != nil {
		t.Fatalf("NewGridLayout() error = %v", err)
	}
	if err := grid.AddPanel(first, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(first) error = %v", err)
	}
	if err := grid.AddPanel(second, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(second) error = %v", err)
	}
	if err := footer.SetLayout(grid); err != nil {
		t.Fatalf("Footer.SetLayout(grid) error = %v", err)
	}
	if len(app.Snapshot().Overflows) == 0 {
		t.Fatal("one-row Footer width overflow is not observable")
	}
	waitOverflowState(t, app, "default_active")

	bad, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.bad"},
	})
	if err != nil {
		t.Fatalf("NewHeader(bad) error = %v", err)
	}
	top, err := NewPanel(bad, PanelOptions{
		AutomationKey: "bad.top", MinimumSize: Size{Height: 1},
	})
	if err != nil {
		t.Fatalf("NewPanel(top) error = %v", err)
	}
	bottom, err := NewPanel(bad, PanelOptions{
		AutomationKey: "bad.bottom", MinimumSize: Size{Height: 1},
	})
	if err != nil {
		t.Fatalf("NewPanel(bottom) error = %v", err)
	}
	vertical, err := NewBoxLayout(Vertical, BoxLayoutOptions{})
	if err != nil {
		t.Fatalf("NewBoxLayout(vertical) error = %v", err)
	}
	_ = vertical.AddPanel(top, LayoutItemOptions{})
	_ = vertical.AddPanel(bottom, LayoutItemOptions{})
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetLayout(bad, vertical); err != nil {
		t.Fatalf("SetLayout() builder error = %v", err)
	}
	if err := tx.Commit(context.Background()); !errors.Is(
		err,
		ErrInvalidLayout,
	) {
		t.Fatalf("multirow Header Layout error = %v", err)
	}
	if app.Snapshot().Sequence != before {
		t.Fatal("rejected Header Layout published partial state")
	}
	for _, layout := range app.Snapshot().Layouts {
		if layout.Owner == bad.ID() {
			t.Fatal("rejected Header Layout became attached")
		}
	}
}

func TestHeaderFooterValidationAndConstrainedRoot(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{
		Size: Size{Width: 30, Height: 8},
		RootConstraints: RootConstraints{
			Maximum: Size{Width: 12, Height: 4},
		},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	parent, err := NewPanel(app.Root(), PanelOptions{})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	if _, err := NewHeader(parent, HeaderOptions{}); err == nil {
		t.Fatal("non-root Header parent unexpectedly succeeded")
	}
	if _, err := NewFooter(app.Root(), FooterOptions{
		PanelOptions: PanelOptions{Bounds: Rect{Width: 1, Height: 1}},
	}); err == nil {
		t.Fatal("caller-set Footer geometry unexpectedly succeeded")
	}
	header, err := NewHeader(app.Root(), HeaderOptions{
		PanelOptions: PanelOptions{AutomationKey: "header.constrained"},
	})
	if err != nil {
		t.Fatalf("NewHeader() error = %v", err)
	}
	if header.Bounds() != (Rect{Width: 30, Height: 1}) {
		t.Fatalf("constrained Header bounds = %+v", header.Bounds())
	}
	if err := header.SetBounds(Rect{Width: 1, Height: 1}); err == nil {
		t.Fatal("Header.SetBounds() unexpectedly succeeded")
	}
	if err := header.SetMinimumSize(Size{Height: 1}); err == nil {
		t.Fatal("Header.SetMinimumSize() unexpectedly succeeded")
	}
	layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(header, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(Header) builder error = %v", err)
	}
	tx := app.NewTransaction()
	if err := tx.SetLayout(app.Root(), layout); err != nil {
		t.Fatalf("SetLayout(root) builder error = %v", err)
	}
	if err := tx.Commit(context.Background()); !errors.Is(
		err,
		ErrInvalidLayout,
	) {
		t.Fatalf("Header as Layout item error = %v", err)
	}
}
