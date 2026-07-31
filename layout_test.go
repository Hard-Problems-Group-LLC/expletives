package expletives

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBoxLayoutArrangesAndResizesWithoutChangingOrder(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 6})
	first := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "box.first",
		MinimumSize:   Size{Width: 2, Height: 1},
		Style:         "back",
	})
	second := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "box.second",
		MinimumSize:   Size{Width: 3, Height: 2},
		Style:         "nested",
	})
	third := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "box.third",
		MinimumSize:   Size{Width: 1, Height: 1},
		Style:         "front",
	})
	layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: "box.root",
		Gap:           1,
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(first, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("AddPanel(first) error = %v", err)
	}
	if err := layout.AddPanel(second, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("AddPanel(second) error = %v", err)
	}
	if err := layout.AddPanel(third, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(third) error = %v", err)
	}
	if got, want := layout.MinimumSize(), (Size{Width: 10, Height: 4}); got != want {
		t.Fatalf("detached minimum = %+v, want %+v", got, want)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatalf("SetLayout() error = %v", err)
	}

	if got, want := first.Bounds(), (Rect{X: 1, Y: 1, Width: 7, Height: 4}); got != want {
		t.Fatalf("first bounds = %+v, want %+v", got, want)
	}
	if got, want := second.Bounds(), (Rect{X: 9, Y: 1, Width: 8, Height: 4}); got != want {
		t.Fatalf("second bounds = %+v, want %+v", got, want)
	}
	if got, want := third.Bounds(), (Rect{X: 18, Y: 1, Width: 1, Height: 4}); got != want {
		t.Fatalf("third bounds = %+v, want %+v", got, want)
	}
	if err := second.SetBounds(Rect{Width: 1, Height: 1}); !errors.Is(err, ErrLayoutManaged) {
		t.Fatalf("managed SetBounds() error = %v, want ErrLayoutManaged", err)
	}

	if err := app.SetSize(Size{Width: 10, Height: 6}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	if got, want := first.Bounds(), (Rect{X: 1, Y: 1, Width: 2, Height: 4}); got != want {
		t.Fatalf("resized first bounds = %+v, want %+v", got, want)
	}
	if got, want := second.Bounds(), (Rect{X: 4, Y: 1, Width: 3, Height: 4}); got != want {
		t.Fatalf("resized second bounds = %+v, want %+v", got, want)
	}
	if got, want := third.Bounds(), (Rect{X: 8, Y: 1, Width: 1, Height: 4}); got != want {
		t.Fatalf("resized third bounds = %+v, want %+v", got, want)
	}
	snapshot := app.Snapshot()
	if len(snapshot.Layouts) != 1 || snapshot.Layouts[0].Key != "box.root" {
		t.Fatalf("Layout snapshot = %#v", snapshot.Layouts)
	}
	if got := controlByKey(t, snapshot, "box.second"); got.LayoutIndex != 1 ||
		got.StackIndex != 1 || got.Layout != layout.ID() {
		t.Fatalf("second Layout observation = %#v", got)
	}
	snapshot.Layouts[0].Items[0].Kind = "corrupted"
	if got := app.Snapshot().Layouts[0].Items[0].Kind; got != "panel" {
		t.Fatalf("Layout snapshot exposed mutable item storage: %q", got)
	}
}

func TestLayoutBorderOwnsOneInsetWithoutDoubleFramingChildren(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 10, Height: 5})
	foregroundWant := RGB(0x12, 0x34, 0x56)
	backgroundWant := RGB(0x65, 0x43, 0x21)
	foreground, background := foregroundWant, backgroundWant
	left, err := NewFrame(app.Root(), FrameOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "layout-border.left",
			MinimumSize:   Size{Width: 1, Height: 1},
		},
		BorderForm: BorderNone,
	})
	if err != nil {
		t.Fatalf("NewFrame(left) error = %v", err)
	}
	right, err := NewFrame(app.Root(), FrameOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "layout-border.right",
			MinimumSize:   Size{Width: 1, Height: 1},
		},
		BorderForm: BorderNone,
	})
	if err != nil {
		t.Fatalf("NewFrame(right) error = %v", err)
	}
	layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: "layout-border",
		Border: BorderOptions{
			Form:       BorderDouble,
			Foreground: &foreground,
			Background: &background,
		},
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	foreground, background = RGB(0, 0, 0), RGB(0, 0, 0)
	if err := layout.AddPanel(left, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("AddPanel(left) error = %v", err)
	}
	if err := layout.AddPanel(right, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("AddPanel(right) error = %v", err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatalf("SetLayout() error = %v", err)
	}

	if got, want := layout.MinimumSize(), (Size{Width: 4, Height: 3}); got != want {
		t.Fatalf("Layout minimum = %+v, want %+v", got, want)
	}
	if got, want := left.Bounds(), (Rect{X: 1, Y: 1, Width: 4, Height: 3}); got != want {
		t.Fatalf("left bounds = %+v, want %+v", got, want)
	}
	if got, want := right.Bounds(), (Rect{X: 5, Y: 1, Width: 4, Height: 3}); got != want {
		t.Fatalf("right bounds = %+v, want %+v", got, want)
	}
	snapshot := app.Snapshot()
	if got := cellAt(t, snapshot, 0, 0); got.Grapheme != "╔" ||
		got.Owner != app.Root().ID() ||
		got.Foreground != foregroundWant ||
		got.Background != backgroundWant {
		t.Fatalf("Layout border corner = %+v", got)
	}
	if got := cellAt(t, snapshot, 5, 2); got.Grapheme != " " ||
		got.Owner != right.ID() {
		t.Fatalf("adjacent unframed child cell = %+v", got)
	}
	if got := snapshot.Layouts[0].Border; got == nil ||
		got.Form != BorderDouble ||
		got.ForegroundOverride == nil ||
		*got.ForegroundOverride != foregroundWant ||
		got.BackgroundOverride == nil ||
		*got.BackgroundOverride != backgroundWant {
		t.Fatalf("Layout border snapshot = %+v", got)
	}
}

func TestGridAndNestedLayoutUseOwnerRelativePanelBounds(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 11, Height: 5})
	panels := []*Panel{
		mustPanel(t, app.Root(), PanelOptions{AutomationKey: "grid.0", MinimumSize: Size{Width: 1, Height: 1}}),
		mustPanel(t, app.Root(), PanelOptions{AutomationKey: "grid.1", MinimumSize: Size{Width: 1, Height: 1}}),
		mustPanel(t, app.Root(), PanelOptions{AutomationKey: "grid.2", MinimumSize: Size{Width: 1, Height: 1}}),
	}
	grid, err := NewGridLayout(GridLayoutOptions{
		AutomationKey: "grid.root",
		Columns:       2,
		HorizontalGap: 1,
		VerticalGap:   1,
	})
	if err != nil {
		t.Fatalf("NewGridLayout() error = %v", err)
	}
	for _, panel := range panels {
		if err := grid.AddPanel(panel, LayoutItemOptions{}); err != nil {
			t.Fatalf("Grid AddPanel() error = %v", err)
		}
	}
	if err := app.Root().SetLayout(grid); err != nil {
		t.Fatalf("SetLayout(Grid) error = %v", err)
	}
	wants := []Rect{
		{Width: 5, Height: 2},
		{X: 6, Width: 5, Height: 2},
		{Y: 3, Width: 5, Height: 2},
	}
	for index, panel := range panels {
		if got := panel.Bounds(); got != wants[index] {
			t.Fatalf("grid panel %d bounds = %+v, want %+v", index, got, wants[index])
		}
	}

	nestedPanel := mustPanel(t, panels[0], PanelOptions{
		AutomationKey: "nested.panel",
		MinimumSize:   Size{Width: 2, Height: 1},
	})
	nested, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: "nested.layout",
	})
	if err != nil {
		t.Fatalf("NewBoxLayout(nested) error = %v", err)
	}
	if err := nested.AddPanel(nestedPanel, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatalf("nested AddPanel() error = %v", err)
	}
	if err := panels[0].SetLayout(nested); err != nil {
		t.Fatalf("nested SetLayout() error = %v", err)
	}
	if got, want := nestedPanel.Bounds(), (Rect{Width: 5, Height: 2}); got != want {
		t.Fatalf("nested Panel bounds = %+v, want %+v", got, want)
	}
}

func TestPanelRaiseLowerPreserveArrangementAndOtherKindSlots(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 9, Height: 3})
	first := mustPanel(t, app.Root(), PanelOptions{AutomationKey: "stack.first", MinimumSize: Size{Width: 1, Height: 1}})
	second := mustPanel(t, app.Root(), PanelOptions{AutomationKey: "stack.second", MinimumSize: Size{Width: 1, Height: 1}})
	nestedPanel := mustPanel(t, app.Root(), PanelOptions{AutomationKey: "stack.nested", MinimumSize: Size{Width: 1, Height: 1}})
	nested, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "stack.child"})
	if err := nested.AddPanel(nestedPanel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	layout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "stack.root"})
	if err := layout.AddPanel(first, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := layout.AddLayout(nested, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(second, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	firstBounds, secondBounds := first.Bounds(), second.Bounds()
	before := app.Snapshot().Sequence
	if err := second.Raise(); err != nil {
		t.Fatalf("already-highest Raise() error = %v", err)
	}
	if got := app.Snapshot().Sequence; got != before {
		t.Fatalf("no-op Raise published sequence %d, want %d", got, before)
	}
	if err := first.Raise(); err != nil {
		t.Fatalf("first.Raise() error = %v", err)
	}
	snapshot := app.Snapshot()
	if got := controlByKey(t, snapshot, "stack.first").StackIndex; got != 2 {
		t.Fatalf("raised first stack index = %d, want 2", got)
	}
	if got := controlByKey(t, snapshot, "stack.second").StackIndex; got != 0 {
		t.Fatalf("lowered second stack index = %d, want 0", got)
	}
	childLayout := layoutByKey(t, snapshot, "stack.child")
	rootLayout := layoutByKey(t, snapshot, "stack.root")
	if got := rootLayout.Items[1].Layout; got != childLayout.ID ||
		rootLayout.Items[1].StackIndex != 1 {
		t.Fatalf("nested Layout slot changed: %#v", rootLayout.Items)
	}
	if first.Bounds() != firstBounds || second.Bounds() != secondBounds {
		t.Fatal("Raise changed arrangement geometry")
	}
	if err := first.Lower(); err != nil {
		t.Fatalf("first.Lower() error = %v", err)
	}
	if got := controlByKey(t, app.Snapshot(), "stack.first").StackIndex; got != 0 {
		t.Fatalf("lowered first stack index = %d, want 0", got)
	}
}

func TestTopLevelLayoutRaiseLowerMovesWholePaintSubtree(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	back := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "layer.back", MinimumSize: Size{Width: 1, Height: 1}, Style: "back",
	})
	front := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "layer.front", MinimumSize: Size{Width: 1, Height: 1}, Style: "front",
	})
	backLayout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "layer.back.layout"})
	frontLayout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "layer.front.layout"})
	if err := backLayout.AddPanel(back, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatal(err)
	}
	if err := frontLayout.AddPanel(front, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(backLayout); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().AddLayout(frontLayout); err != nil {
		t.Fatal(err)
	}
	if got := cellAt(t, app.Snapshot(), 0, 0).Owner; got != front.ID() {
		t.Fatalf("initial top painter = %q, want %q", got, front.ID())
	}
	if err := backLayout.Raise(); err != nil {
		t.Fatalf("back Layout Raise() error = %v", err)
	}
	if got := cellAt(t, app.Snapshot(), 0, 0).Owner; got != back.ID() {
		t.Fatalf("raised top painter = %q, want %q", got, back.ID())
	}
	if err := backLayout.Lower(); err != nil {
		t.Fatalf("back Layout Lower() error = %v", err)
	}
	if got := cellAt(t, app.Snapshot(), 0, 0).Owner; got != front.ID() {
		t.Fatalf("lowered top painter = %q, want %q", got, front.ID())
	}
}

func TestNestedLayoutRaisePreservesPanelSlotAndArrangement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 6, Height: 1})
	left := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "nested.left", MinimumSize: Size{Width: 1, Height: 1},
	})
	middle := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "nested.middle", MinimumSize: Size{Width: 1, Height: 1},
	})
	right := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "nested.right", MinimumSize: Size{Width: 1, Height: 1},
	})
	leftLayout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "nested.left.layout"})
	rightLayout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "nested.right.layout"})
	parent, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "nested.parent"})
	if err := leftLayout.AddPanel(left, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := rightLayout.AddPanel(right, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddLayout(leftLayout, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddPanel(middle, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddLayout(rightLayout, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(parent); err != nil {
		t.Fatal(err)
	}
	if got, want := left.Bounds(), (Rect{Width: 2, Height: 1}); got != want {
		t.Fatalf("left nested bounds = %+v, want %+v", got, want)
	}
	if got, want := right.Bounds(), (Rect{X: 4, Width: 2, Height: 1}); got != want {
		t.Fatalf("right nested bounds = %+v, want %+v", got, want)
	}
	leftBounds, rightBounds := left.Bounds(), right.Bounds()
	if err := leftLayout.Raise(); err != nil {
		t.Fatalf("nested Layout Raise() error = %v", err)
	}
	snapshot := app.Snapshot()
	if got := layoutByKey(t, snapshot, "nested.left.layout"); got.LayoutIndex != 0 ||
		got.StackIndex != 2 {
		t.Fatalf("raised nested Layout = %#v", got)
	}
	if got := layoutByKey(t, snapshot, "nested.right.layout"); got.LayoutIndex != 2 ||
		got.StackIndex != 0 {
		t.Fatalf("lower nested Layout = %#v", got)
	}
	if got := controlByKey(t, snapshot, "nested.middle"); got.StackIndex != 1 {
		t.Fatalf("Panel slot moved to %d, want 1", got.StackIndex)
	}
	if left.Bounds() != leftBounds || right.Bounds() != rightBounds {
		t.Fatal("nested Layout Raise changed arrangement")
	}
}

func TestControlLayoutHintDefaultsAndConstructionOverrides(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 10})
	field, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "hints.field",
			MinimumSize:   Size{Width: 10, Height: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	area, err := NewTextArea(app.Root(), TextAreaOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "hints.area",
			MinimumSize:   Size{Width: 10, Height: 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	label, err := NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "hints.label",
			MinimumSize:   Size{Width: 10, Height: 1},
		},
		Text: "Label",
	})
	if err != nil {
		t.Fatal(err)
	}
	overridden, err := NewTextField(app.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "hints.overridden",
			MinimumSize:   Size{Width: 10, Height: 1},
			LayoutHints: LayoutHints{
				Horizontal:     LayoutSizeNatural,
				Vertical:       LayoutSizeStretch,
				VerticalWeight: 3,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := field.LayoutHints(), (LayoutHints{
		Horizontal:       LayoutSizeStretch,
		Vertical:         LayoutSizeNatural,
		HorizontalWeight: 1,
	}); got != want {
		t.Fatalf("TextField LayoutHints = %+v, want %+v", got, want)
	}
	if got, want := area.LayoutHints(), (LayoutHints{
		Horizontal:       LayoutSizeStretch,
		Vertical:         LayoutSizeStretch,
		HorizontalWeight: 1,
		VerticalWeight:   1,
	}); got != want {
		t.Fatalf("TextArea LayoutHints = %+v, want %+v", got, want)
	}
	if got, want := label.LayoutHints(), (LayoutHints{
		Horizontal: LayoutSizeNatural,
		Vertical:   LayoutSizeNatural,
	}); got != want {
		t.Fatalf("Label LayoutHints = %+v, want %+v", got, want)
	}
	if got, want := overridden.LayoutHints(), (LayoutHints{
		Horizontal:     LayoutSizeNatural,
		Vertical:       LayoutSizeStretch,
		VerticalWeight: 3,
	}); got != want {
		t.Fatalf("overridden TextField LayoutHints = %+v, want %+v", got, want)
	}

	for _, options := range []PanelOptions{
		{LayoutHints: LayoutHints{Horizontal: LayoutSizeHint("invalid")}},
		{LayoutHints: LayoutHints{HorizontalWeight: -1}},
		{LayoutHints: LayoutHints{
			Horizontal:       LayoutSizeNatural,
			HorizontalWeight: 1,
		}},
	} {
		if _, createErr := NewPanel(app.Root(), options); !errors.Is(
			createErr,
			ErrInvalidLayout,
		) {
			t.Fatalf("NewPanel(%+v) error = %v, want ErrInvalidLayout", options, createErr)
		}
	}
}

func TestBoxLayoutUsesControlHintsAndWeights(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 20})
	newField := func(key string) *TextField {
		t.Helper()
		field, err := NewTextField(app.Root(), TextFieldOptions{
			PanelOptions: PanelOptions{
				AutomationKey: key,
				MinimumSize:   Size{Width: 10, Height: 1},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return field
	}
	newArea := func(key string, weight int) *TextArea {
		t.Helper()
		area, err := NewTextArea(app.Root(), TextAreaOptions{
			PanelOptions: PanelOptions{
				AutomationKey: key,
				MinimumSize:   Size{Width: 10, Height: 3},
				LayoutHints: LayoutHints{
					VerticalWeight: weight,
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return area
	}
	firstField := newField("weights.field.first")
	secondField := newField("weights.field.second")
	firstArea := newArea("weights.area.first", 1)
	secondArea := newArea("weights.area.second", 3)
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: "weights.layout",
		Gap:           1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, control := range []Control{
		firstField,
		secondField,
		firstArea,
		secondArea,
	} {
		if err := layout.AddPanel(control, LayoutItemOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	wants := []Rect{
		{Width: 40, Height: 1},
		{Y: 2, Width: 40, Height: 1},
		{Y: 4, Width: 40, Height: 6},
		{Y: 11, Width: 40, Height: 9},
	}
	for index, control := range []Control{
		firstField,
		secondField,
		firstArea,
		secondArea,
	} {
		if got := control.Bounds(); got != wants[index] {
			t.Fatalf("control %d bounds = %+v, want %+v", index, got, wants[index])
		}
	}
}

func TestBoxAndGridDefaultAlignmentConsultControlHints(t *testing.T) {
	t.Parallel()
	boxApp := mustApp(t, Size{Width: 40, Height: 3})
	label, err := NewLabel(boxApp.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			MinimumSize: Size{Width: 10, Height: 1},
		},
		Text: "Label",
	})
	if err != nil {
		t.Fatal(err)
	}
	field, err := NewTextField(boxApp.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{
			MinimumSize: Size{Width: 10, Height: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewBoxLayout(Horizontal, BoxLayoutOptions{Gap: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := box.AddPanel(label, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := box.AddPanel(field, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := boxApp.Root().SetLayout(box); err != nil {
		t.Fatal(err)
	}
	if got, want := label.Bounds(), (Rect{Width: 10, Height: 1}); got != want {
		t.Fatalf("Box Label bounds = %+v, want %+v", got, want)
	}
	if got, want := field.Bounds(), (Rect{X: 11, Width: 29, Height: 1}); got != want {
		t.Fatalf("Box TextField bounds = %+v, want %+v", got, want)
	}

	gridApp := mustApp(t, Size{Width: 20, Height: 5})
	gridLabel, err := NewLabel(gridApp.Root(), LabelOptions{
		PanelOptions: PanelOptions{MinimumSize: Size{Width: 5, Height: 1}},
		Text:         "Label",
	})
	if err != nil {
		t.Fatal(err)
	}
	gridField, err := NewTextField(gridApp.Root(), TextFieldOptions{
		PanelOptions: PanelOptions{MinimumSize: Size{Width: 5, Height: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	grid, err := NewGridLayout(GridLayoutOptions{Columns: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := grid.AddPanel(gridLabel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := grid.AddPanel(gridField, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := gridApp.Root().SetLayout(grid); err != nil {
		t.Fatal(err)
	}
	if got, want := gridLabel.Bounds(), (Rect{Width: 5, Height: 1}); got != want {
		t.Fatalf("Grid Label bounds = %+v, want %+v", got, want)
	}
	if got, want := gridField.Bounds(), (Rect{
		X: 10, Width: 10, Height: 1,
	}); got != want {
		t.Fatalf("Grid TextField bounds = %+v, want %+v", got, want)
	}
}

func TestLayoutAttachmentIsAtomicAndDestroyRemovesMembership(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 8, Height: 2})
	other := mustApp(t, Size{Width: 8, Height: 2})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "atomic.panel", MinimumSize: Size{Width: 2, Height: 1},
	})
	layout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "atomic.layout"})
	if err := layout.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	before := app.Snapshot()
	if err := other.Root().SetLayout(layout); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("cross-App SetLayout() error = %v, want ErrInvalidLayout", err)
	}
	if got := app.Snapshot().Sequence; got != before.Sequence || layout.ID() != "" {
		t.Fatal("failed attachment changed App or detached Layout")
	}
	collision, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: "atomic.panel",
	})
	if err := collision.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(collision); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("cross-kind key collision error = %v, want ErrDuplicateKey", err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	if err := panel.Destroy(); err != nil {
		t.Fatalf("managed Panel Destroy() error = %v", err)
	}
	snapshot := app.Snapshot()
	if len(snapshot.Layouts) != 1 || len(snapshot.Layouts[0].Items) != 0 {
		t.Fatalf("destroy left dangling Layout item: %#v", snapshot.Layouts)
	}
}

func TestDetachedLayoutRejectsDuplicateCycleDepthAndGridCapacity(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	panel := mustPanel(t, app.Root(), PanelOptions{})
	box, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err := box.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := box.AddPanel(panel, LayoutItemOptions{}); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("duplicate AddPanel() error = %v, want ErrInvalidLayout", err)
	}

	child, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err := box.AddLayout(child, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := child.AddLayout(box, LayoutItemOptions{}); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("cyclic AddLayout() error = %v, want ErrInvalidLayout", err)
	}

	parent := child
	for depth := 2; depth < MaxLayoutDepth; depth++ {
		next, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{})
		if err := parent.AddLayout(next, LayoutItemOptions{}); err != nil {
			t.Fatalf("AddLayout depth %d error = %v", depth+1, err)
		}
		parent = next
	}
	tooDeep, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err := parent.AddLayout(tooDeep, LayoutItemOptions{}); !errors.Is(
		err,
		ErrLayoutCapacity,
	) {
		t.Fatalf("depth overflow error = %v, want ErrLayoutCapacity", err)
	}

	grid, err := NewGridLayout(GridLayoutOptions{Rows: 1, Columns: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := grid.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	other := mustPanel(t, app.Root(), PanelOptions{})
	if err := grid.AddPanel(other, LayoutItemOptions{}); !errors.Is(
		err,
		ErrLayoutCapacity,
	) {
		t.Fatalf("fixed Grid overflow error = %v, want ErrLayoutCapacity", err)
	}
}

func TestOverflowFallbackDismissRecoveryAndHandler(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "overflow.panel",
		MinimumSize:   Size{Width: 4, Height: 2},
	})
	layout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{AutomationKey: "overflow.layout"})
	if err := layout.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	waitOverflowState(t, app, "default_active")
	overflow := app.Snapshot().Overflows[0]
	if overflow.Layout != layout.ID() ||
		overflow.Deficit != (Size{Width: 2, Height: 1}) {
		t.Fatalf("overflow = %#v", overflow)
	}
	warning := cellAt(t, app.Snapshot(), 0, 0)
	if warning.Grapheme != "!" ||
		warning.Foreground != (RGB(0, 0, 0)) ||
		warning.Background != (RGB(0xFF, 0xFF, 0)) {
		t.Fatalf("tiny overflow warning = %#v", warning)
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"human",
		"overflow-dismiss",
		KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
	)
	if err != nil {
		t.Fatalf("overflow Enter dismissal error = %v", err)
	}
	if completion.Outcome != OutcomeApplied ||
		completion.Command != CommandOverflowDismiss {
		t.Fatalf("overflow Enter completion = %+v", completion)
	}
	if got := app.Snapshot().Overflows[0].State; got != "acknowledged" {
		t.Fatalf("dismissed state = %q", got)
	}
	if got := cellAt(t, app.Snapshot(), 0, 0).Owner; got != panel.ID() {
		t.Fatalf("dismissal left warning cell owner %q, want %q", got, panel.ID())
	}
	sequence := app.Snapshot().Sequence
	if err := app.DismissOverflow(); err != nil {
		t.Fatalf("already-dismissed DismissOverflow() error = %v", err)
	}
	if got := app.Snapshot().Sequence; got != sequence {
		t.Fatalf("no-op DismissOverflow published %d, want %d", got, sequence)
	}
	if err := app.SetSize(Size{Width: 5, Height: 3}); err != nil {
		t.Fatal(err)
	}
	if got := len(app.Snapshot().Overflows); got != 0 {
		t.Fatalf("recovery retained %d overflow records", got)
	}

	handled := make(chan OverflowEvent, 1)
	if err := app.SetOverflowHandler(func(
		_ context.Context,
		event OverflowEvent,
	) OverflowDisposition {
		handled <- event
		return OverflowHandled
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetSize(Size{Width: 1, Height: 1}); err != nil {
		t.Fatal(err)
	}
	waitOverflowState(t, app, "application_handled")
	select {
	case event := <-handled:
		if event.Overflow.Layout != layout.ID() {
			t.Fatalf("handler Layout = %q, want %q", event.Overflow.Layout, layout.ID())
		}
	default:
		t.Fatal("handled state published without handler observation")
	}
}

func TestOverflowEpisodeFollowsEffectiveOwnerVisibility(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	ancestor := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "visibility.ancestor",
		Bounds:        Rect{Width: 2, Height: 1},
	})
	owner := mustPanel(t, ancestor, PanelOptions{
		AutomationKey: "visibility.owner",
		Bounds:        Rect{Width: 2, Height: 1},
	})
	child := mustPanel(t, owner, PanelOptions{
		AutomationKey: "visibility.child",
		MinimumSize:   Size{Width: 4, Height: 1},
	})
	layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: "visibility.layout",
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(child, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel() error = %v", err)
	}
	if err := owner.SetLayout(layout); err != nil {
		t.Fatalf("SetLayout() error = %v", err)
	}
	if got := len(app.Snapshot().Overflows); got != 1 {
		t.Fatalf("visible owner overflows = %d, want 1", got)
	}
	firstEpisode := app.Snapshot().Overflows[0].EpisodeID

	if err := owner.SetVisible(false); err != nil {
		t.Fatalf("SetVisible(false) error = %v", err)
	}
	if got := len(app.Snapshot().Overflows); got != 0 {
		t.Fatalf("hidden owner overflows = %d, want 0", got)
	}
	if err := owner.SetVisible(true); err != nil {
		t.Fatalf("SetVisible(true) error = %v", err)
	}
	snapshot := app.Snapshot()
	if got := len(snapshot.Overflows); got != 1 {
		t.Fatalf("reshown owner overflows = %d, want 1", got)
	}
	if snapshot.Overflows[0].EpisodeID == firstEpisode {
		t.Fatalf(
			"reshown overflow retained episode %q, want a new episode",
			firstEpisode,
		)
	}

	if err := ancestor.SetVisible(false); err != nil {
		t.Fatalf("ancestor SetVisible(false) error = %v", err)
	}
	if got := len(app.Snapshot().Overflows); got != 0 {
		t.Fatalf("hidden ancestor overflows = %d, want 0", got)
	}
}

func TestOverflowPanicAndStuckHandlerUseBoundedFallback(t *testing.T) {
	app := mustApp(t, Size{Width: 1, Height: 1})
	panel := mustPanel(t, app.Root(), PanelOptions{
		MinimumSize: Size{Width: 2, Height: 1},
	})
	layout, _ := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err := layout.AddPanel(panel, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetOverflowHandler(func(
		context.Context,
		OverflowEvent,
	) OverflowDisposition {
		panic("test panic")
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	waitOverflowState(t, app, "default_active")

	if err := app.SetSize(Size{Width: 2, Height: 1}); err != nil {
		t.Fatal(err)
	}
	block := make(chan struct{})
	invoked := make(chan struct{}, 2)
	if err := app.SetOverflowHandler(func(
		context.Context,
		OverflowEvent,
	) OverflowDisposition {
		invoked <- struct{}{}
		<-block
		return OverflowHandled
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetSize(Size{Width: 1, Height: 1}); err != nil {
		t.Fatal(err)
	}
	waitOverflowState(t, app, "default_active")
	select {
	case <-invoked:
	default:
		t.Fatal("stuck handler was not invoked")
	}

	if err := app.SetSize(Size{Width: 2, Height: 1}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetSize(Size{Width: 1, Height: 1}); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	waitOverflowState(t, app, "default_active")
	if elapsed := time.Since(start); elapsed >= DefaultOverflowHandlerTimeout {
		t.Fatalf("saturated fallback waited %v for a stuck handler", elapsed)
	}
	select {
	case <-invoked:
		t.Fatal("saturated dispatcher started a second handler")
	default:
	}
	close(block)
}

func layoutByKey(t *testing.T, snapshot Snapshot, key string) LayoutSnapshot {
	t.Helper()
	for _, layout := range snapshot.Layouts {
		if layout.Key == key {
			return layout
		}
	}
	t.Fatalf("snapshot has no Layout with key %q", key)
	return LayoutSnapshot{}
}

func waitOverflowState(t *testing.T, app *App, state string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := app.Snapshot()
		if len(snapshot.Overflows) == 1 &&
			snapshot.Overflows[0].State == state {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("overflow never reached %q: %#v", state, app.Snapshot().Overflows)
}

func FuzzBoxLayoutDeterminism(f *testing.F) {
	f.Add(byte(3), byte(2), byte(1), byte(1))
	f.Add(byte(8), byte(9), byte(3), byte(7))
	f.Fuzz(func(
		t *testing.T,
		countByte, minimumByte, gapByte, extraByte byte,
	) {
		count := 1 + int(countByte%8)
		minimum := 1 + int(minimumByte%10)
		gap := int(gapByte % 4)
		extra := int(extraByte % 32)
		width := count*minimum + (count-1)*gap + extra

		app := mustApp(t, Size{Width: width, Height: 1})
		tx := app.NewTransaction()
		layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{Gap: gap})
		if err != nil {
			t.Fatal(err)
		}
		for index := 0; index < count; index++ {
			panel, createErr := tx.NewPanel(app.Root(), PanelOptions{
				MinimumSize: Size{Width: minimum, Height: 1},
			})
			if createErr != nil {
				t.Fatal(createErr)
			}
			if addErr := layout.AddPanel(panel, LayoutItemOptions{
				Grow: 1 + index%3,
			}); addErr != nil {
				t.Fatal(addErr)
			}
		}
		if err := tx.SetLayout(app.Root(), layout); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		first := app.Snapshot()
		if len(first.Layouts) != 1 || len(first.Layouts[0].Items) != count {
			t.Fatalf("Layout snapshot cardinality = %#v", first.Layouts)
		}
		cursor := 0
		for index, item := range first.Layouts[0].Items {
			if item.LayoutIndex != index || item.Bounds.X != cursor ||
				item.Bounds.Width < minimum || item.Bounds.Height < 1 {
				t.Fatalf("item %d = %#v, cursor %d", index, item, cursor)
			}
			cursor += item.Bounds.Width
			if index+1 < count {
				cursor += gap
			}
		}
		if cursor != width {
			t.Fatalf("arranged extent = %d, want %d", cursor, width)
		}
		sequence := first.Sequence
		if err := app.SetSize(Size{Width: width, Height: 1}); err != nil {
			t.Fatal(err)
		}
		if got := app.Snapshot().Sequence; got != sequence {
			t.Fatalf("no-op resize published %d, want %d", got, sequence)
		}
	})
}
