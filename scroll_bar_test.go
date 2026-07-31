package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func scrollBarDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) ScrollBarDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.ScrollBar
	if details == nil {
		t.Fatalf("%s has no ScrollBarDetails", key)
	}
	return *details
}

func dispatchScrollBarKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"scrollbar-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func TestScrollBarValidationDefaultsGeometryAndRendering(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	for name, options := range map[string]ScrollBarOptions{
		"orientation": {
			Orientation: Orientation(9),
		},
		"negative content": {
			State: ScrollBarState{ContentSize: -1},
		},
		"offset beyond maximum": {
			State: ScrollBarState{
				ContentSize: 10, ViewportSize: 4, Offset: 7,
			},
		},
		"negative arrow step": {
			ArrowStep: -1,
		},
		"enabled reason": {
			DisabledReason: "not disabled",
		},
	} {
		if _, err := NewScrollBar(
			app.Root(),
			options,
		); !errors.Is(err, ErrValidation) &&
			!errors.Is(err, ErrInvalidControl) {
			t.Fatalf("%s NewScrollBar() error = %v", name, err)
		}
	}

	horizontal, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "horizontal",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		State: ScrollBarState{
			ContentSize: 100, ViewportSize: 20, Offset: 40,
		},
	})
	if err != nil {
		t.Fatalf("NewScrollBar(horizontal) error = %v", err)
	}
	vertical, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "vertical",
			Bounds:        Rect{X: 15, Width: 1, Height: 7},
		},
		Orientation: Vertical,
		State: ScrollBarState{
			ContentSize: 20, ViewportSize: 5, Offset: 15,
		},
		Disabled: true,
	})
	if err != nil {
		t.Fatalf("NewScrollBar(vertical) error = %v", err)
	}
	if horizontal.MinimumSize() != (Size{Width: 3, Height: 1}) ||
		vertical.MinimumSize() != (Size{Width: 1, Height: 3}) {
		t.Fatalf(
			"ScrollBar minima horizontal=%+v vertical=%+v",
			horizontal.MinimumSize(),
			vertical.MinimumSize(),
		)
	}
	details := scrollBarDetailsByKey(t, app, "horizontal")
	if details.MaximumOffset != 80 ||
		details.ArrowStep != 1 ||
		details.PageStep != 20 ||
		details.TrackStart != 1 ||
		details.TrackSize != 10 ||
		details.ThumbStart != 5 ||
		details.ThumbSize != 2 ||
		!details.Enabled {
		t.Fatalf("horizontal details = %#v", details)
	}
	disabled := scrollBarDetailsByKey(t, app, "vertical")
	if disabled.Enabled ||
		disabled.DisabledReason != "Control is disabled" ||
		disabled.TrackStart != 1 ||
		disabled.TrackSize != 5 ||
		disabled.ThumbStart != 5 ||
		disabled.ThumbSize != 1 {
		t.Fatalf("vertical details = %#v", disabled)
	}

	snapshot := app.Snapshot()
	left, _ := snapshot.Frame.Cell(0, 0)
	right, _ := snapshot.Frame.Cell(11, 0)
	thumb, _ := snapshot.Frame.Cell(5, 0)
	up, _ := snapshot.Frame.Cell(15, 0)
	down, _ := snapshot.Frame.Cell(15, 6)
	if left.Grapheme != "◄" || right.Grapheme != "►" ||
		thumb.Grapheme != "▓" ||
		thumb.Style != "scrollbar.focused" ||
		up.Grapheme != "▲" || down.Grapheme != "▼" ||
		up.Style != "scrollbar.disabled" {
		t.Fatalf(
			"ScrollBar cells left=%#v right=%#v thumb=%#v up=%#v down=%#v",
			left,
			right,
			thumb,
			up,
			down,
		)
	}
}

func TestScrollBarTinyAndDegenerateGeometry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		axis  int
		state ScrollBarState
		want  scrollBarGeometry
	}{
		{0, ScrollBarState{}, scrollBarGeometry{}},
		{1, ScrollBarState{}, scrollBarGeometry{
			trackSize: 1, thumbSize: 1,
		}},
		{2, ScrollBarState{
			ContentSize: 10, ViewportSize: 0, Offset: 10,
		}, scrollBarGeometry{
			trackSize: 2, thumbStart: 1, thumbSize: 1,
		}},
		{3, ScrollBarState{
			ContentSize: 10, ViewportSize: 2, Offset: 4,
		}, scrollBarGeometry{
			trackStart: 1, trackSize: 1, thumbStart: 1, thumbSize: 1,
		}},
		{12, ScrollBarState{
			ContentSize: 100, ViewportSize: 20, Offset: 80,
		}, scrollBarGeometry{
			trackStart: 1, trackSize: 10, thumbStart: 9, thumbSize: 2,
		}},
	}
	for _, test := range tests {
		if got := calculateScrollBarGeometry(
			test.axis,
			test.state,
		); got != test.want {
			t.Fatalf(
				"calculateScrollBarGeometry(%d, %#v)=%#v, want %#v",
				test.axis,
				test.state,
				got,
				test.want,
			)
		}
	}
}

func TestScrollBarKeyboardClampingFocusAndChangeCommand(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 4})
	registerActionCommand(t, app, "scroll.changed", "Changed", true)
	bar, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "bar",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		State: ScrollBarState{
			ContentSize: 100, ViewportSize: 20, Offset: 10,
		},
		ArrowStep: 3, PageStep: 25,
		ChangeCommand: "scroll.changed",
	})
	if err != nil {
		t.Fatalf("NewScrollBar() error = %v", err)
	}
	other, err := NewCheckbox(app.Root(), CheckboxOptions{
		PanelOptions: PanelOptions{
			Bounds: Rect{Y: 2, Width: 10, Height: 1},
		},
		Label: "Other",
	})
	if err != nil {
		t.Fatalf("NewCheckbox() error = %v", err)
	}
	if app.Focused() != bar {
		t.Fatalf("initial focus = %#v, want ScrollBar", app.Focused())
	}

	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = bar.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	for index, test := range []struct {
		key  Key
		want int
	}{
		{KeyRight, 13},
		{KeyPageDown, 38},
		{KeyEnd, 80},
		{KeyRight, 80},
		{KeyPageUp, 55},
		{KeyHome, 0},
		{KeyLeft, 0},
	} {
		completion := dispatchScrollBarKey(
			t,
			app,
			fmtRequest("scroll", index),
			test.key,
		)
		if bar.State().Offset != test.want {
			t.Fatalf(
				"%s offset=%d, want %d",
				test.key,
				bar.State().Offset,
				test.want,
			)
		}
		wantCommand := CommandID("scroll.changed")
		if index == 3 || index == 6 {
			wantCommand = ""
		}
		if completion.Command != wantCommand {
			t.Fatalf(
				"%s completion command=%q, want %q",
				test.key,
				completion.Command,
				wantCommand,
			)
		}
	}
	mu.Lock()
	if len(routed) != 5 {
		t.Fatalf("change command calls = %d, want 5", len(routed))
	}
	mu.Unlock()

	beforeCalls := len(routed)
	if err := bar.SetState(ScrollBarState{
		ContentSize: 100, ViewportSize: 20, Offset: 12,
	}); err != nil {
		t.Fatalf("programmatic SetState() error = %v", err)
	}
	if len(routed) != beforeCalls {
		t.Fatal("programmatic ScrollBar setter invoked ChangeCommand")
	}

	// The unmatched axis remains available for spatial focus movement.
	dispatchScrollBarKey(t, app, "move-focus-down", KeyDown)
	if app.Focused() != other {
		t.Fatalf("Down focus = %#v, want other Checkbox", app.Focused())
	}
	if err := bar.Focus(); err != nil {
		t.Fatalf("ScrollBar.Focus() error = %v", err)
	}
	if app.Focused() != bar {
		t.Fatalf("Focus() selected %#v", app.Focused())
	}
}

func TestScrollBarUpdatesAreAtomicCancellableAndConcurrent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 3})
	first, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "first"},
		State: ScrollBarState{
			ContentSize: 100, ViewportSize: 10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "second"},
		State: ScrollBarState{
			ContentSize: 100, ViewportSize: 10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetScrollBarState(first, ScrollBarState{
		ContentSize: 100, ViewportSize: 10, Offset: 20,
	}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetScrollBarState(second, ScrollBarState{
		ContentSize: 100, ViewportSize: 10, Offset: 30,
	}); err != nil {
		t.Fatal(err)
	}
	before := app.Snapshot().Sequence
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := app.Snapshot().Sequence; got != before+1 {
		t.Fatalf("batched update sequence=%d after %d", got, before)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := first.Update(cancelled, ScrollBarState{
		ContentSize: 100, ViewportSize: 10, Offset: 40,
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Update() error=%v", err)
	}
	if first.State().Offset != 20 {
		t.Fatalf("cancelled Update changed state to %#v", first.State())
	}

	const workers = 8
	var wait sync.WaitGroup
	failures := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for attempt := 0; attempt < 20; attempt++ {
				err := first.Update(
					context.Background(),
					ScrollBarState{
						ContentSize:  100,
						ViewportSize: 10,
						Offset:       (worker*20 + attempt) % 91,
					},
				)
				if err != nil && !errors.Is(err, ErrMutationBusy) {
					failures <- err
					return
				}
				_ = app.Snapshot()
			}
		}(worker)
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent Update() error=%v", err)
	}
	if state := first.State(); state.Offset < 0 ||
		state.Offset > scrollBarMaximumOffset(state) {
		t.Fatalf("concurrent final state=%#v", state)
	}
}

func TestScrollBarSnapshotDetailsAreIndependent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 2})
	if _, err := NewScrollBar(app.Root(), ScrollBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "bar",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		State: ScrollBarState{
			ContentSize: 20, ViewportSize: 5, Offset: 3,
		},
	}); err != nil {
		t.Fatal(err)
	}
	first := app.Snapshot()
	controlByKey(t, first, "bar").Details.ScrollBar.Offset = 9
	if got := scrollBarDetailsByKey(t, app, "bar").Offset; got != 3 {
		t.Fatalf("snapshot mutation changed canonical offset to %d", got)
	}
}

func fmtRequest(prefix string, index int) string {
	return prefix + "-" + string(rune('a'+index))
}
