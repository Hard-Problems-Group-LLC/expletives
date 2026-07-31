package expletives

import (
	"errors"
	"fmt"
	"testing"
)

func testStyle(id StyleID, background Color) Style {
	return Style{
		ID:         id,
		Foreground: RGB(0xFF, 0xFF, 0xFF),
		Background: background,
	}
}

func mustApp(t *testing.T, size Size) *App {
	t.Helper()
	theme, err := NewTheme(
		testStyle("root", RGB(1, 2, 3)),
		testStyle("panel", RGB(4, 5, 6)),
		testStyle("frame", RGB(0, 0, 0)),
		testStyle("frame.border", RGB(0x40, 0x40, 0x40)),
		testStyle("group_box", RGB(0, 0, 0)),
		testStyle("group_box.border", RGB(0, 0, 0)),
		testStyle("layout.border", RGB(0x30, 0x30, 0x30)),
		testStyle("label", RGB(0, 0, 0)),
		testStyle("static_text", RGB(0, 0, 0)),
		testStyle("separator", RGB(0, 0, 0)),
		testStyle("rule", RGB(0, 0, 0)),
		testStyle("button", RGB(0, 0, 0)),
		testStyle("hotkey_bar", RGB(0, 0, 0)),
		testStyle("focus_guide_bar", RGB(0, 0, 0)),
		testStyle("checkbox", RGB(0, 0, 0)),
		testStyle("radio_group", RGB(0, 0, 0)),
		testStyle("radio_button", RGB(0, 0, 0)),
		testStyle("cycle_field", RGB(0, 0, 0)),
		testStyle("select_field", RGB(0, 0, 0)),
		testStyle("text_field", RGB(0, 0, 0)),
		testStyle("number_field", RGB(0, 0, 0)),
		testStyle("spin_box", RGB(0, 0, 0)),
		testStyle("text_input.valid", RGB(0, 0, 0)),
		testStyle("text_input.invalid", RGB(0, 0, 0)),
		testStyle("text_input.invalid_character", RGB(0, 0, 0)),
		testStyle("text_input.disabled", RGB(0, 0, 0)),
		testStyle("selection.mnemonic", RGB(0, 0, 0)),
		testStyle("selection.focused", RGB(0x20, 0x20, 0x40)),
		testStyle("selection.focused_mnemonic", RGB(0x40, 0x20, 0x40)),
		testStyle("selection.disabled", RGB(0x20, 0x20, 0x20)),
		testStyle("status_bar", RGB(0xC0, 0xC0, 0xC0)),
		testStyle("status.shortcut", RGB(0xC0, 0xC0, 0xC0)),
		testStyle("status.disabled", RGB(0xC0, 0xC0, 0xC0)),
		testStyle("header", RGB(0x10, 0x20, 0x30)),
		testStyle("footer", RGB(0x30, 0x20, 0x10)),
		testStyle("menu_bar", RGB(0, 0, 0)),
		testStyle("menu.popup", RGB(0x10, 0x10, 0x10)),
		testStyle("menu.border", RGB(0x30, 0x30, 0x30)),
		testStyle("menu.mnemonic", RGB(0x40, 0x10, 0x10)),
		testStyle("menu.focused", RGB(0x20, 0x20, 0x40)),
		testStyle("menu.focused_mnemonic", RGB(0x40, 0x20, 0x40)),
		testStyle("menu.disabled", RGB(0x20, 0x20, 0x20)),
		testStyle("menu.focused_disabled", RGB(0x20, 0x20, 0x30)),
		testStyle("menu.shadow", RGB(0, 0, 0)),
		testStyle("before", RGB(1, 1, 1)),
		testStyle("after", RGB(9, 8, 7)),
		testStyle("moving", RGB(7, 8, 9)),
		testStyle("back", RGB(0x80, 0, 0)),
		testStyle("nested", RGB(0, 0x80, 0)),
		testStyle("front", RGB(0, 0, 0x80)),
		testStyle("frame.fill", RGB(0x20, 0x20, 0x20)),
		testStyle("frame.child", RGB(0x60, 0x20, 0x60)),
		testStyle("group.fill", RGB(0x20, 0x40, 0x40)),
		testStyle("group.border", RGB(0x60, 0x60, 0x20)),
		testStyle("child", RGB(9, 8, 7)),
	)
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	app, err := NewApp(AppOptions{
		Size:      size,
		Scenario:  "test.scene",
		Theme:     theme,
		RootStyle: "root",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	return app
}

func mustPanel(t *testing.T, parent Container, options PanelOptions) *Panel {
	t.Helper()
	panel, err := NewPanel(parent, options)
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}
	return panel
}

func controlByKey(t *testing.T, snapshot SnapshotV1, key string) ControlSnapshot {
	t.Helper()
	for _, control := range snapshot.Controls {
		if control.Key == key {
			return control
		}
	}
	t.Fatalf("snapshot has no control with key %q", key)
	return ControlSnapshot{}
}

func cellAt(t *testing.T, snapshot SnapshotV1, x, y int) Cell {
	t.Helper()
	cell, ok := snapshot.Frame.Cell(x, y)
	if !ok {
		t.Fatalf("frame has no cell at (%d,%d)", x, y)
	}
	return cell
}

func TestRootParentAndStableChildOrder(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 8})
	root := app.Root()

	if root.Parent() != nil {
		t.Fatalf("Root().Parent() = %p, want nil", root.Parent())
	}
	if root.ID() != "root" {
		t.Fatalf("Root().ID() = %q, want root", root.ID())
	}
	if got := root.Bounds(); got != (Rect{Width: 12, Height: 8}) {
		t.Fatalf("root bounds = %+v, want surface bounds", got)
	}
	if _, err := NewPanel(nil, PanelOptions{}); !errors.Is(err, ErrInvalidParent) {
		t.Fatalf("NewPanel(nil) error = %v, want ErrInvalidParent", err)
	}
	var zero Panel
	if _, err := NewPanel(&zero, PanelOptions{}); !errors.Is(err, ErrInvalidParent) {
		t.Fatalf("NewPanel(zero Panel) error = %v, want ErrInvalidParent", err)
	}
	copiedRoot := *root
	copiedChild, err := NewPanel(&copiedRoot, PanelOptions{
		AutomationKey: "copied-child",
	})
	if err != nil {
		t.Fatalf("NewPanel(copied root) error = %v", err)
	}
	if copiedChild.Parent() != root {
		t.Fatalf("copied-handle child parent = %p, want canonical root %p",
			copiedChild.Parent(), root)
	}

	first := mustPanel(t, root, PanelOptions{
		AutomationKey: "first",
		Bounds:        Rect{X: 1, Y: 1, Width: 2, Height: 2},
	})
	second := mustPanel(t, root, PanelOptions{
		AutomationKey: "second",
		Bounds:        Rect{X: 4, Y: 1, Width: 2, Height: 2},
	})
	if first.Parent() != root || second.Parent() != root {
		t.Fatal("ordinary Panel parent is not the constructor parent")
	}
	if first.ID() == second.ID() || first.ID() == root.ID() {
		t.Fatalf("control IDs are not unique: root=%q first=%q second=%q",
			root.ID(), first.ID(), second.ID())
	}

	children := root.Children()
	if len(children) != 3 ||
		children[0] != copiedChild ||
		children[1] != first ||
		children[2] != second {
		t.Fatalf("root children = %#v, want stable insertion order", children)
	}
	children[0] = nil
	if got := root.Children(); got[0] != copiedChild {
		t.Fatal("Children() exposed mutable internal storage")
	}

	before := app.Snapshot()
	if _, err := NewPanel(root, PanelOptions{
		AutomationKey: "first",
		Bounds:        Rect{Width: 1, Height: 1},
	}); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("duplicate automation key error = %v, want ErrDuplicateKey", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence || len(after.Controls) != len(before.Controls) {
		t.Fatalf("failed construction changed snapshot: before=%d/%d after=%d/%d",
			before.Sequence, len(before.Controls), after.Sequence, len(after.Controls))
	}

	if err := root.SetBounds(Rect{Width: 1, Height: 1}); err == nil {
		t.Fatal("root SetBounds succeeded")
	}
	if err := root.SetVisible(false); err == nil {
		t.Fatal("root SetVisible(false) succeeded")
	}
}

func TestIndependentAppsHaveIndependentRootsAndKeys(t *testing.T) {
	t.Parallel()
	first := mustApp(t, Size{Width: 2, Height: 2})
	second := mustApp(t, Size{Width: 2, Height: 2})
	if first.Root() == second.Root() {
		t.Fatal("independent Apps share a root")
	}
	if _, err := NewPanel(first.Root(), PanelOptions{
		AutomationKey: "shared",
		Bounds:        Rect{Width: 1, Height: 1},
	}); err != nil {
		t.Fatalf("first App NewPanel() error = %v", err)
	}
	if _, err := NewPanel(second.Root(), PanelOptions{
		AutomationKey: "shared",
		Bounds:        Rect{Width: 1, Height: 1},
	}); err != nil {
		t.Fatalf("automation key should be App-scoped: %v", err)
	}
}

func TestPanelFrameGroupBoxPaintingClippingAndOverlap(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 8})
	root := app.Root()

	back := mustPanel(t, root, PanelOptions{
		AutomationKey: "back",
		Bounds:        Rect{X: 1, Y: 1, Width: 6, Height: 4},
		Style:         "back",
	})
	nested := mustPanel(t, back, PanelOptions{
		AutomationKey: "nested",
		Bounds:        Rect{X: 4, Y: 1, Width: 4, Height: 3},
		Style:         "nested",
	})
	front := mustPanel(t, root, PanelOptions{
		AutomationKey: "front",
		Bounds:        Rect{X: 3, Y: 3, Width: 5, Height: 3},
		Style:         "front",
	})
	frame, err := NewFrame(root, FrameOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "frame",
			Bounds:        Rect{X: 8, Y: 1, Width: 4, Height: 5},
			Style:         "frame.fill",
		},
		BorderStyle: "frame.border",
	})
	if err != nil {
		t.Fatalf("NewFrame() error = %v", err)
	}
	frameChild := mustPanel(t, frame, PanelOptions{
		AutomationKey: "frame-child",
		Bounds:        Rect{X: -1, Y: 0, Width: 4, Height: 3},
		Style:         "frame.child",
	})
	group, err := NewGroupBox(root, GroupBoxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "group",
			Bounds:        Rect{X: 1, Y: 6, Width: 9, Height: 2},
			Style:         "group.fill",
		},
		Title:       "A\u754CB",
		BorderStyle: "group.border",
	})
	if err != nil {
		t.Fatalf("NewGroupBox() error = %v", err)
	}

	snapshot := app.Snapshot()
	wantOrder := []ControlID{
		root.ID(), back.ID(), nested.ID(), front.ID(),
		frame.ID(), frameChild.ID(), group.ID(),
	}
	if len(snapshot.Controls) != len(wantOrder) {
		t.Fatalf("control count = %d, want %d", len(snapshot.Controls), len(wantOrder))
	}
	for index, want := range wantOrder {
		if got := snapshot.Controls[index].ID; got != want {
			t.Fatalf("control[%d].ID = %q, want %q", index, got, want)
		}
	}

	backResolved, found := app.Theme().Resolve(back.Style())
	if !found {
		t.Fatalf("Theme.Resolve(%q) missing", back.Style())
	}
	if got := cellAt(t, snapshot, 2, 2); got.Owner != back.ID() ||
		got.Background != backResolved.Background {
		t.Fatalf("back cell = %+v, want back ownership/style", got)
	}
	if got := cellAt(t, snapshot, 6, 2); got.Owner != nested.ID() {
		t.Fatalf("nested visible cell owner = %q, want %q", got.Owner, nested.ID())
	}
	if got := cellAt(t, snapshot, 5, 3); got.Owner != front.ID() {
		t.Fatalf("overlap owner = %q, want later sibling %q", got.Owner, front.ID())
	}

	nestedView := controlByKey(t, snapshot, "nested")
	if nestedView.AbsoluteBounds != (Rect{X: 5, Y: 2, Width: 4, Height: 3}) {
		t.Fatalf("nested absolute bounds = %+v", nestedView.AbsoluteBounds)
	}
	if nestedView.EffectiveClip != (Rect{X: 5, Y: 2, Width: 2, Height: 3}) {
		t.Fatalf("nested effective clip = %+v, want ancestor-clipped width 2",
			nestedView.EffectiveClip)
	}

	if got := cellAt(t, snapshot, 8, 1); got.Grapheme != "┌" ||
		got.Owner != frame.ID() || got.Style != "frame.border" {
		t.Fatalf("frame corner = %+v", got)
	}
	if got := cellAt(t, snapshot, 8, 2); got.Grapheme != "│" ||
		got.Owner != frame.ID() {
		t.Fatalf("frame border overwritten by clipped child: %+v", got)
	}
	if got := cellAt(t, snapshot, 9, 2); got.Owner != frameChild.ID() {
		t.Fatalf("frame client cell owner = %q, want %q", got.Owner, frameChild.ID())
	}
	childView := controlByKey(t, snapshot, "frame-child")
	if childView.AbsoluteBounds != (Rect{X: 8, Y: 2, Width: 4, Height: 3}) ||
		childView.EffectiveClip != (Rect{X: 9, Y: 2, Width: 2, Height: 3}) {
		t.Fatalf("frame child geometry = abs %+v clip %+v",
			childView.AbsoluteBounds, childView.EffectiveClip)
	}

	titleWant := []string{"A", "\uFFFD", "B"}
	for offset, want := range titleWant {
		got := cellAt(t, snapshot, 3+offset, 6)
		if got.Grapheme != want || got.Owner != group.ID() {
			t.Fatalf("group title cell %d = %+v, want grapheme %q owner %q",
				offset, got, want, group.ID())
		}
	}
}

func TestFrameBorderFormsInsetsAndColorOverrides(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 6})
	foreground := RGB(0x12, 0x34, 0x56)
	background := RGB(0x65, 0x43, 0x21)
	forms := []struct {
		form   BorderForm
		corner string
		inset  int
	}{
		{BorderNone, " ", 0},
		{BorderSingle, "┌", 1},
		{BorderDouble, "╔", 1},
		{BorderShadeLight, "░", 1},
		{BorderShadeMedium, "▒", 1},
		{BorderShadeDark, "▓", 1},
		{BorderBlock, "█", 1},
	}
	for index, test := range forms {
		frame, err := NewFrame(app.Root(), FrameOptions{
			PanelOptions: PanelOptions{
				AutomationKey: fmt.Sprintf("border.%d", index),
				Bounds:        Rect{X: index * 4, Width: 4, Height: 3},
			},
			BorderForm:       test.form,
			BorderForeground: &foreground,
			BorderBackground: &background,
		})
		if err != nil {
			t.Fatalf("NewFrame(%s) error = %v", test.form, err)
		}
		snapshot := app.Snapshot()
		cell := cellAt(t, snapshot, index*4, 0)
		if cell.Grapheme != test.corner {
			t.Fatalf("%s corner = %q, want %q", test.form, cell.Grapheme, test.corner)
		}
		view := controlByKey(t, snapshot, fmt.Sprintf("border.%d", index))
		if view.Details.Container.ClientInset != test.inset ||
			view.Details.Border.Form != test.form {
			t.Fatalf(
				"%s inset/form = %d/%q, want %d/%q",
				test.form,
				view.Details.Container.ClientInset,
				view.Details.Border.Form,
				test.inset,
				test.form,
			)
		}
		if test.form != BorderNone &&
			(cell.Foreground != foreground || cell.Background != background) {
			t.Fatalf("%s override colors = %+v/%+v", test.form, cell.Foreground, cell.Background)
		}
		if frame == nil {
			t.Fatal("NewFrame() returned nil")
		}
	}
}

func TestFrameRejectsUnknownBorderForm(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	if _, err := NewFrame(app.Root(), FrameOptions{
		BorderForm: "ornate",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("NewFrame(unknown border) error = %v, want ErrInvalidControl", err)
	}
}

func TestHiddenParentClipsDescendantsAndKeepsLogicalGeometry(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 6, Height: 4})
	parent := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "parent",
		Bounds:        Rect{X: 1, Y: 1, Width: 4, Height: 2},
	})
	child := mustPanel(t, parent, PanelOptions{
		AutomationKey: "child",
		Bounds:        Rect{X: 1, Y: 0, Width: 2, Height: 2},
		Style:         "child",
	})
	if err := parent.SetVisible(false); err != nil {
		t.Fatalf("SetVisible(false) error = %v", err)
	}

	snapshot := app.Snapshot()
	childView := controlByKey(t, snapshot, "child")
	if childView.Visible {
		t.Fatal("child of hidden parent is semantically visible")
	}
	if childView.AbsoluteBounds != (Rect{X: 2, Y: 1, Width: 2, Height: 2}) {
		t.Fatalf("hidden child lost logical geometry: %+v", childView.AbsoluteBounds)
	}
	if !childView.EffectiveClip.Empty() {
		t.Fatalf("hidden child clip = %+v, want empty", childView.EffectiveClip)
	}
	if got := cellAt(t, snapshot, 2, 1); got.Owner == child.ID() {
		t.Fatal("hidden descendant painted a cell")
	}

	if err := child.SetBounds(Rect{Width: -1, Height: 1}); !errors.Is(err, ErrInvalidGeometry) {
		t.Fatalf("negative SetBounds error = %v, want ErrInvalidGeometry", err)
	}
}
