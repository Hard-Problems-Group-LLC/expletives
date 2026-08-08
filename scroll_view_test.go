package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func scrollViewDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) ScrollableDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.Scrollable
	if details == nil {
		t.Fatalf("%s has no ScrollableDetails", key)
	}
	return *details
}

func dispatchScrollViewKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"scroll-view-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func TestClampViewportStateUsesClosedOffsetInterval(t *testing.T) {
	t.Parallel()
	geometry := scrollViewGeometry{maximumOffset: Point{X: 5, Y: 7}}

	if got := clampViewportState(ViewportState{
		ContentSize: Size{Width: 20, Height: 20},
		Offset:      Point{X: -3, Y: -4},
	}, geometry); got.Offset != (Point{}) {
		t.Fatalf("negative clamp offset = %+v, want origin", got.Offset)
	}
	if got := clampViewportState(ViewportState{
		ContentSize: Size{Width: 20, Height: 20},
		Offset:      Point{X: 9, Y: 11},
	}, geometry); got.Offset != geometry.maximumOffset {
		t.Fatalf(
			"maximum clamp offset = %+v, want %+v",
			got.Offset,
			geometry.maximumOffset,
		)
	}
}

func TestScrollablePanelGeometryManagedContentAndRendering(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 12})
	panel, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "scroll",
					Bounds: Rect{
						X: 2, Y: 1, Width: 10, Height: 6,
					},
				},
				State: ViewportState{
					ContentSize: Size{Width: 20, Height: 10},
					Offset:      Point{X: 2, Y: 3},
				},
			},
			BorderForm: BorderSingle,
		},
	)
	if err != nil {
		t.Fatalf("NewScrollablePanel() error = %v", err)
	}
	content := panel.Content()
	if content == nil ||
		content.AutomationKey() != "scroll.content" ||
		content.Parent() != panel {
		t.Fatalf("managed Content = %#v", content)
	}
	if got, want := content.Bounds(), (Rect{
		X: -2, Y: -3, Width: 20, Height: 10,
	}); got != want {
		t.Fatalf("Content bounds = %+v, want %+v", got, want)
	}

	snapshot := app.Snapshot()
	owner := controlByKey(t, snapshot, "scroll")
	contentSnapshot := controlByKey(t, snapshot, "scroll.content")
	if len(owner.Children) != 1 || owner.Children[0] != content.ID() {
		t.Fatalf("scroll children = %#v", owner.Children)
	}
	if contentSnapshot.Parent != panel.ID() ||
		contentSnapshot.EffectiveClip != (Rect{
			X: 3, Y: 2, Width: 7, Height: 3,
		}) {
		t.Fatalf("Content snapshot = %#v", contentSnapshot)
	}
	details := owner.Details.Scrollable
	if details == nil ||
		details.State != (ViewportState{
			ContentSize: Size{Width: 20, Height: 10},
			Offset:      Point{X: 2, Y: 3},
		}) ||
		details.MaximumOffset != (Point{X: 13, Y: 7}) ||
		details.ViewportBounds != (Rect{
			X: 1, Y: 1, Width: 7, Height: 3,
		}) ||
		!details.HorizontalVisible ||
		!details.VerticalVisible ||
		details.HorizontalBar == nil ||
		details.VerticalBar == nil ||
		details.Content != content.ID() ||
		details.ContentKey != content.AutomationKey() {
		t.Fatalf("Scrollable details = %#v", details)
	}
	if details.HorizontalBar.ContentSize != 20 ||
		details.HorizontalBar.ViewportSize != 7 ||
		details.HorizontalBar.Offset != 2 ||
		details.VerticalBar.ContentSize != 10 ||
		details.VerticalBar.ViewportSize != 3 ||
		details.VerticalBar.Offset != 3 {
		t.Fatalf(
			"integrated bar details horizontal=%#v vertical=%#v",
			details.HorizontalBar,
			details.VerticalBar,
		)
	}
	if got := cellAt(t, snapshot, 2, 1); got.Grapheme != "┌" ||
		got.Owner != panel.ID() {
		t.Fatalf("border corner = %#v", got)
	}
	if got := cellAt(t, snapshot, 3, 5); got.Grapheme != "◄" ||
		got.Owner != panel.ID() {
		t.Fatalf("horizontal bar start = %#v", got)
	}
	if got := cellAt(t, snapshot, 10, 2); got.Grapheme != "▲" ||
		got.Owner != panel.ID() {
		t.Fatalf("vertical bar start = %#v", got)
	}
	if got := cellAt(t, snapshot, 10, 5); got.Style != "scrollbar.corner" ||
		got.Owner != panel.ID() {
		t.Fatalf("scrollbar corner = %#v", got)
	}
}

func TestViewportOwnershipLayoutAndDestructionContracts(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	viewport, err := NewViewport(app.Root(), ScrollViewOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "viewport",
			Bounds:        Rect{Width: 8, Height: 4},
		},
		State: ViewportState{
			ContentSize: Size{Width: 12, Height: 6},
		},
	})
	if err != nil {
		t.Fatalf("NewViewport() error = %v", err)
	}
	content := viewport.Content()
	if content == nil {
		t.Fatal("Viewport.Content() = nil")
	}
	details := scrollViewDetailsByKey(t, app, "viewport")
	if details.ViewportBounds != (Rect{Width: 8, Height: 4}) ||
		details.HorizontalPolicy != ScrollBarVisibilityNever ||
		details.VerticalPolicy != ScrollBarVisibilityNever ||
		details.HorizontalVisible ||
		details.VerticalVisible ||
		details.HorizontalBar != nil ||
		details.VerticalBar != nil {
		t.Fatalf("Viewport details = %#v", details)
	}

	if _, err := NewPanel(viewport, PanelOptions{}); !errors.Is(
		err,
		ErrInvalidParent,
	) {
		t.Fatalf("direct child NewPanel() error = %v", err)
	}
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := viewport.SetLayout(layout); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("Viewport.SetLayout() error = %v", err)
	}
	child := mustPanel(t, content, PanelOptions{
		AutomationKey: "viewport.child",
		MinimumSize:   Size{Width: 2, Height: 1},
	})
	contentLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: "viewport.content.layout",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := contentLayout.AddPanel(child, LayoutItemOptions{}); err != nil {
		t.Fatalf("Content Layout AddPanel() error = %v", err)
	}
	if err := content.SetLayout(contentLayout); err != nil {
		t.Fatalf("Content.SetLayout() error = %v", err)
	}
	if err := content.SetBounds(Rect{Width: 1, Height: 1}); err == nil {
		t.Fatal("managed Content SetBounds() succeeded")
	}
	if err := content.SetMinimumSize(Size{Width: 1, Height: 1}); err == nil {
		t.Fatal("managed Content SetMinimumSize() succeeded")
	}
	if err := content.Destroy(); err == nil {
		t.Fatal("managed Content Destroy() succeeded")
	}
	if err := viewport.Destroy(); err != nil {
		t.Fatalf("Viewport.Destroy() error = %v", err)
	}
	if viewport.Content() != nil {
		t.Fatal("destroyed Viewport retained Content")
	}
	if err := child.SetVisible(false); !errors.Is(err, ErrDestroyed) {
		t.Fatalf("destroyed descendant SetVisible() error = %v", err)
	}
}

func TestScrollViewAutoFixedPointPoliciesAndResizeClamping(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	panel, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "fixed-point",
					Bounds:        Rect{Width: 5, Height: 3},
				},
				State: ViewportState{
					ContentSize: Size{Width: 5, Height: 4},
				},
			},
			BorderForm: BorderNone,
		},
	)
	if err != nil {
		t.Fatalf("NewScrollablePanel() error = %v", err)
	}
	details := scrollViewDetailsByKey(t, app, "fixed-point")
	if !details.HorizontalVisible || !details.VerticalVisible ||
		details.ViewportBounds != (Rect{Width: 4, Height: 2}) ||
		details.MaximumOffset != (Point{X: 1, Y: 2}) {
		t.Fatalf("fixed-point details = %#v", details)
	}
	if err := panel.SetState(ViewportState{
		ContentSize: Size{Width: 5, Height: 4},
		Offset:      Point{X: 1, Y: 2},
	}); err != nil {
		t.Fatalf("SetState(maximum) error = %v", err)
	}
	if err := panel.SetBounds(Rect{Width: 12, Height: 7}); err != nil {
		t.Fatalf("SetBounds(larger) error = %v", err)
	}
	if got := panel.State().Offset; got != (Point{}) {
		t.Fatalf("resized offset = %+v, want zero", got)
	}
	details = scrollViewDetailsByKey(t, app, "fixed-point")
	if details.HorizontalVisible || details.VerticalVisible ||
		details.ViewportBounds != (Rect{Width: 12, Height: 7}) ||
		panel.Content().Bounds() != (Rect{Width: 5, Height: 4}) {
		t.Fatalf("resized details = %#v", details)
	}

	always, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "always",
					Bounds:        Rect{Y: 4, Width: 2, Height: 2},
				},
				State: ViewportState{
					ContentSize: Size{Width: 1, Height: 1},
				},
			},
			BorderForm:    BorderNone,
			HorizontalBar: ScrollBarVisibilityAlways,
			VerticalBar:   ScrollBarVisibilityAlways,
		},
	)
	if err != nil {
		t.Fatalf("NewScrollablePanel(always) error = %v", err)
	}
	alwaysDetails := scrollViewDetailsByKey(t, app, "always")
	if !alwaysDetails.HorizontalVisible ||
		!alwaysDetails.VerticalVisible ||
		alwaysDetails.ViewportBounds != (Rect{Width: 1, Height: 1}) ||
		always.State().Offset != (Point{}) {
		t.Fatalf("always details = %#v", alwaysDetails)
	}
}

func TestScrollViewKeyboardEnsureVisibleAndChangeCommand(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 9})
	registerActionCommand(t, app, "viewport.changed", "Changed", true)
	panel, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "keys",
					Bounds:        Rect{Width: 10, Height: 6},
				},
				State: ViewportState{
					ContentSize: Size{Width: 20, Height: 10},
					Offset:      Point{X: 2, Y: 3},
				},
				ArrowStep:     Size{Width: 2, Height: 2},
				ChangeCommand: "viewport.changed",
			},
			BorderForm: BorderSingle,
		},
	)
	if err != nil {
		t.Fatalf("NewScrollablePanel() error = %v", err)
	}
	if app.Focused() != panel {
		t.Fatalf("initial focus = %#v, want ScrollablePanel", app.Focused())
	}

	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = panel.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	tests := []struct {
		key  Key
		want Point
	}{
		{KeyLeft, Point{X: 0, Y: 3}},
		{KeyPageDown, Point{X: 0, Y: 6}},
		{KeyEnd, Point{X: 13, Y: 7}},
		{KeyRight, Point{X: 13, Y: 7}},
		{KeyHome, Point{}},
	}
	for index, test := range tests {
		completion := dispatchScrollViewKey(
			t,
			app,
			fmtRequest("viewport-key", index),
			test.key,
		)
		if got := panel.State().Offset; got != test.want {
			t.Fatalf("%s offset = %+v, want %+v", test.key, got, test.want)
		}
		wantCommand := CommandID("viewport.changed")
		if index == 3 {
			wantCommand = ""
		}
		if completion.Command != wantCommand {
			t.Fatalf(
				"%s command = %q, want %q",
				test.key,
				completion.Command,
				wantCommand,
			)
		}
	}
	mu.Lock()
	if len(routed) != 4 {
		t.Fatalf("routed changes = %d, want 4", len(routed))
	}
	before := len(routed)
	mu.Unlock()

	if err := panel.EnsureVisible(Rect{
		X: 15, Y: 8, Width: 2, Height: 2,
	}); err != nil {
		t.Fatalf("EnsureVisible() error = %v", err)
	}
	if got, want := panel.State().Offset, (Point{X: 10, Y: 7}); got != want {
		t.Fatalf("EnsureVisible offset = %+v, want %+v", got, want)
	}
	if err := panel.SetState(ViewportState{
		ContentSize: Size{Width: 20, Height: 10},
		Offset:      Point{X: 1, Y: 1},
	}); err != nil {
		t.Fatalf("programmatic SetState() error = %v", err)
	}
	for name, rect := range map[string]Rect{
		"negative": {X: -1, Width: 1, Height: 1},
		"empty":    {Width: 0, Height: 1},
		"outside":  {X: 19, Width: 2, Height: 1},
	} {
		if err := panel.EnsureVisible(rect); err == nil {
			t.Fatalf("EnsureVisible(%s) succeeded", name)
		}
	}
	if got := panel.State().Offset; got != (Point{X: 1, Y: 1}) {
		t.Fatalf("invalid EnsureVisible changed offset to %+v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(routed) != before {
		t.Fatal("programmatic viewport operations invoked ChangeCommand")
	}
}

func TestScrollViewFocusKeepsDescendantVisible(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 9})
	registerActionCommand(t, app, "near", "Near", true)
	registerActionCommand(t, app, "far", "Far", true)
	panel, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "focus-scroll",
					Bounds:        Rect{Width: 10, Height: 6},
				},
				State: ViewportState{
					ContentSize: Size{Width: 20, Height: 12},
				},
			},
			BorderForm: BorderSingle,
		},
	)
	if err != nil {
		t.Fatalf("NewScrollablePanel() error = %v", err)
	}
	near, err := NewButton(panel.Content(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "near",
			Bounds:        Rect{Width: 4, Height: 1},
		},
		Command: "near",
	})
	if err != nil {
		t.Fatal(err)
	}
	far, err := NewButton(panel.Content(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "far",
			Bounds: Rect{
				X: 15, Y: 9, Width: 3, Height: 1,
			},
		},
		Command: "far",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := far.Focus(); err != nil {
		t.Fatalf("far.Focus() error = %v", err)
	}
	if app.Focused() != far {
		t.Fatalf("focus = %#v, want far button", app.Focused())
	}
	if got, want := panel.State().Offset, (Point{X: 11, Y: 7}); got != want {
		t.Fatalf("focused descendant offset = %+v, want %+v", got, want)
	}
	if err := near.Focus(); err != nil {
		t.Fatalf("near.Focus() error = %v", err)
	}
	if got := panel.State().Offset; got != (Point{}) {
		t.Fatalf("return-to-near offset = %+v", got)
	}
}

func TestScrollViewValidationAndAtomicConstruction(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	tests := map[string]ScrollablePanelOptions{
		"negative content": {
			ScrollViewOptions: ScrollViewOptions{
				State: ViewportState{
					ContentSize: Size{Width: -1},
				},
			},
		},
		"offset beyond content": {
			ScrollViewOptions: ScrollViewOptions{
				State: ViewportState{
					ContentSize: Size{Width: 4, Height: 4},
					Offset:      Point{X: 5},
				},
			},
		},
		"negative arrow step": {
			ScrollViewOptions: ScrollViewOptions{
				ArrowStep: Size{Width: -1},
			},
		},
		"negative page step": {
			ScrollViewOptions: ScrollViewOptions{
				PageStep: Size{Height: -1},
			},
		},
		"enabled reason": {
			ScrollViewOptions: ScrollViewOptions{
				DisabledReason: "not disabled",
			},
		},
		"bad policy": {
			HorizontalBar: ScrollBarVisibility("sometimes"),
		},
	}
	before := len(app.Snapshot().Controls)
	for name, options := range tests {
		options.PanelOptions.AutomationKey = "invalid." + name
		if _, err := NewScrollablePanel(app.Root(), options); err == nil {
			t.Fatalf("%s NewScrollablePanel() succeeded", name)
		}
		if got := len(app.Snapshot().Controls); got != before {
			t.Fatalf(
				"%s published partial construction: controls=%d, want %d",
				name,
				got,
				before,
			)
		}
	}
}

func TestScrollViewUpdatesAreCancellableAndConcurrent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	panel, err := NewScrollablePanel(
		app.Root(),
		ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "concurrent",
					Bounds:        Rect{Width: 10, Height: 6},
				},
				State: ViewportState{
					ContentSize: Size{Width: 40, Height: 20},
				},
			},
			BorderForm: BorderSingle,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := panel.Update(cancelled, ViewportState{
		ContentSize: Size{Width: 40, Height: 20},
		Offset:      Point{X: 1, Y: 1},
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Update() error = %v", err)
	}
	if panel.State().Offset != (Point{}) {
		t.Fatal("cancelled Update() changed state")
	}
	if err := panel.Update(nil, ViewportState{}); err == nil {
		t.Fatal("nil-context Update() succeeded")
	}

	const workers = 8
	var wait sync.WaitGroup
	errs := make(chan error, workers)
	for index := range workers {
		wait.Add(1)
		go func(offset int) {
			defer wait.Done()
			errs <- panel.SetState(ViewportState{
				ContentSize: Size{Width: 40, Height: 20},
				Offset: Point{
					X: offset,
					Y: offset,
				},
			})
		}(index + 1)
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent SetState() error = %v", err)
		}
	}
	state := panel.State()
	if state.Offset.X < 1 || state.Offset.X > workers ||
		state.Offset.Y < 1 || state.Offset.Y > workers {
		t.Fatalf("concurrent final state = %+v", state)
	}
}
