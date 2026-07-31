package expletives

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
)

func progressDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) ProgressDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.Progress
	if details == nil {
		t.Fatalf("%s has no ProgressDetails", key)
	}
	return *details
}

func TestProgressBarValidationRenderingAndTinyFallback(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 3})
	if _, err := NewProgressBar(app.Root(), ProgressBarOptions{
		State: ProgressBarState{Current: 2, Total: 1},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("NewProgressBar current error = %v, want ErrValidation", err)
	}
	if _, err := NewProgressBar(app.Root(), ProgressBarOptions{
		State: ProgressBarState{
			Current: 1, Total: 2, Status: ProgressCompleted,
		},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("NewProgressBar completion error = %v, want ErrValidation", err)
	}
	if _, err := NewProgressBar(app.Root(), ProgressBarOptions{
		State: ProgressBarState{
			Indeterminate: true,
			TextMode:      ProgressTextPercentage,
		},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("NewProgressBar text error = %v, want ErrValidation", err)
	}
	bar, err := NewProgressBar(app.Root(), ProgressBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "bar",
			Bounds:        Rect{Width: 10, Height: 1},
		},
		State: ProgressBarState{
			Current: 1, Total: 3, Status: ProgressRunning,
		},
	})
	if err != nil {
		t.Fatalf("NewProgressBar() error = %v", err)
	}
	if bar.MinimumSize() != (Size{Width: 8, Height: 1}) {
		t.Fatalf("ProgressBar minimum = %+v", bar.MinimumSize())
	}
	details := progressDetailsByKey(t, app, "bar")
	if details.Current != 1 || details.Total != 3 ||
		details.Status != ProgressRunning ||
		details.TextMode != ProgressTextPercentage ||
		details.Indeterminate || details.FrameIndex != 0 {
		t.Fatalf("ProgressBar details = %#v", details)
	}
	snapshot := app.Snapshot()
	for column := 0; column < 3; column++ {
		cell, _ := snapshot.Frame.Cell(column, 0)
		if cell.Style != "progress.fill" {
			t.Fatalf("fill cell %d style = %q", column, cell.Style)
		}
	}
	if got := rowText(snapshot, 0); got[3:6] != "33%" {
		t.Fatalf("ProgressBar row = %q", got)
	}

	if err := bar.SetBounds(Rect{Y: 1, Width: 1, Height: 1}); err != nil {
		t.Fatalf("SetBounds() error = %v", err)
	}
	if err := bar.SetState(ProgressBarState{
		Current: 1, Total: 2, Status: ProgressRunning,
	}); err != nil {
		t.Fatalf("SetState() error = %v", err)
	}
	cell, _ := app.Snapshot().Frame.Cell(0, 1)
	if cell.Grapheme != ">" || cell.Style != "progress.text" {
		t.Fatalf("tiny ProgressBar cell = %#v", cell)
	}
}

func TestProgressBarIndeterminateReducedMotionAndCoalescing(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 2})
	bar, err := NewProgressBar(app.Root(), ProgressBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "bar",
			Bounds:        Rect{Width: 6, Height: 1},
		},
		State: ProgressBarState{
			Indeterminate: true, Tick: 8, Status: ProgressRunning,
		},
	})
	if err != nil {
		t.Fatalf("NewProgressBar() error = %v", err)
	}
	details := progressDetailsByKey(t, app, "bar")
	if details.Tick != 8 || details.FrameIndex != 2 {
		t.Fatalf("indeterminate details = %#v", details)
	}
	cell, _ := app.Snapshot().Frame.Cell(2, 0)
	if cell.Style != "progress.fill" {
		t.Fatalf("indeterminate highlight = %#v", cell)
	}

	before := app.Snapshot().Sequence
	frozen := ProgressBarState{
		Indeterminate: true, Tick: math.MaxUint64,
		ReducedMotion: true, Status: ProgressRunning,
	}
	if err := bar.SetState(frozen); err != nil {
		t.Fatalf("SetState(reduced) error = %v", err)
	}
	after := app.Snapshot()
	details = progressDetailsByKey(t, app, "bar")
	if details.Tick != 0 || details.FrameIndex != 0 ||
		!details.ReducedMotion {
		t.Fatalf("reduced details = %#v", details)
	}
	if after.Sequence != before+1 {
		t.Fatalf("first reduced update sequence = %d, want %d",
			after.Sequence, before+1)
	}
	if err := bar.SetState(frozen); err != nil {
		t.Fatalf("duplicate SetState() error = %v", err)
	}
	if got := app.Snapshot().Sequence; got != after.Sequence {
		t.Fatalf("frozen duplicate published sequence %d after %d",
			got, after.Sequence)
	}
}

func TestMeterRangeOrientationAndRendering(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 6})
	for name, state := range map[string]MeterState{
		"equal range": {Maximum: 1, Minimum: 1, Value: 1},
		"below":       {Maximum: 1, Value: -1},
		"nan":         {Maximum: 1, Value: math.NaN()},
		"orientation": {Maximum: 1, Orientation: Orientation(9)},
	} {
		if _, err := NewMeter(app.Root(), MeterOptions{
			State: state,
		}); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s Meter error = %v, want ErrValidation", name, err)
		}
	}
	horizontal, err := NewMeter(app.Root(), MeterOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "meter.horizontal",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		State: MeterState{
			Value: 25, Minimum: 0, Maximum: 100,
			Status: ProgressRunning,
		},
	})
	if err != nil {
		t.Fatalf("NewMeter(horizontal) error = %v", err)
	}
	vertical, err := NewMeter(app.Root(), MeterOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "meter.vertical",
			Bounds:        Rect{X: 9, Width: 1, Height: 4},
		},
		State: MeterState{
			Value: 50, Minimum: 0, Maximum: 100,
			Orientation: Vertical, Status: ProgressFailed,
		},
	})
	if err != nil {
		t.Fatalf("NewMeter(vertical) error = %v", err)
	}
	if horizontal.MinimumSize() != (Size{Width: 8, Height: 1}) ||
		vertical.MinimumSize() != (Size{Width: 1, Height: 3}) {
		t.Fatalf("Meter minima horizontal=%+v vertical=%+v",
			horizontal.MinimumSize(), vertical.MinimumSize())
	}
	snapshot := app.Snapshot()
	for column := 0; column < 8; column++ {
		cell, _ := snapshot.Frame.Cell(column, 0)
		want := StyleID("meter")
		if column < 2 {
			want = "progress.fill"
		}
		if cell.Style != want {
			t.Fatalf("horizontal cell %d style = %q, want %q",
				column, cell.Style, want)
		}
	}
	for row := 0; row < 4; row++ {
		cell, _ := snapshot.Frame.Cell(9, row)
		want := StyleID("meter")
		if row >= 2 {
			want = "progress.failed"
		}
		if cell.Style != want {
			t.Fatalf("vertical row %d style = %q, want %q",
				row, cell.Style, want)
		}
	}
}

func TestSpinnerAndActivityDotsFramesAndTerminalStates(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 3})
	spinner, err := NewSpinner(app.Root(), SpinnerOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "spinner",
			Bounds:        Rect{Width: 1, Height: 1},
		},
		State: ActivityState{Tick: 3, Status: ProgressRunning},
	})
	if err != nil {
		t.Fatalf("NewSpinner() error = %v", err)
	}
	dots, err := NewActivityDots(app.Root(), ActivityDotsOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "dots",
			Bounds:        Rect{X: 2, Width: 3, Height: 1},
		},
		State: ActivityState{Tick: 2, Status: ProgressRunning},
	})
	if err != nil {
		t.Fatalf("NewActivityDots() error = %v", err)
	}
	if spinner.MinimumSize() != (Size{Width: 1, Height: 1}) ||
		dots.MinimumSize() != (Size{Width: 3, Height: 1}) {
		t.Fatalf(
			"activity minima spinner=%+v dots=%+v",
			spinner.MinimumSize(),
			dots.MinimumSize(),
		)
	}
	snapshot := app.Snapshot()
	cell, _ := snapshot.Frame.Cell(0, 0)
	if cell.Grapheme != "\\" {
		t.Fatalf("spinner frame = %q", cell.Grapheme)
	}
	if got := rowText(snapshot, 0)[2:5]; got != "..." {
		t.Fatalf("activity dots = %q", got)
	}
	if progressDetailsByKey(t, app, "spinner").FrameIndex != 3 ||
		progressDetailsByKey(t, app, "dots").FrameIndex != 2 {
		t.Fatal("activity frame indices do not match rendering")
	}
	if err := spinner.SetState(ActivityState{
		Tick: 99, ReducedMotion: true, Status: ProgressRunning,
	}); err != nil {
		t.Fatalf("Spinner reduced update error = %v", err)
	}
	if err := dots.SetState(ActivityState{
		Tick: 99, Status: ProgressCancelled,
	}); err != nil {
		t.Fatalf("ActivityDots cancelled update error = %v", err)
	}
	snapshot = app.Snapshot()
	cell, _ = snapshot.Frame.Cell(0, 0)
	if cell.Grapheme != "*" ||
		progressDetailsByKey(t, app, "spinner").Tick != 0 {
		t.Fatalf("reduced Spinner cell=%#v", cell)
	}
	if got := rowText(snapshot, 0)[2:5]; got != "xxx" {
		t.Fatalf("cancelled dots = %q", got)
	}
}

func TestProgressUpdatesAreAtomicCancellableAndConcurrent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 3})
	bar, err := NewProgressBar(app.Root(), ProgressBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "bar",
			Bounds:        Rect{Width: 10, Height: 1},
		},
		State: ProgressBarState{Total: 100},
	})
	if err != nil {
		t.Fatalf("NewProgressBar() error = %v", err)
	}
	meter, err := NewMeter(app.Root(), MeterOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "meter",
			Bounds:        Rect{Y: 1, Width: 10, Height: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewMeter() error = %v", err)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetProgressBarState(bar, ProgressBarState{
		Current: 50, Total: 100, Status: ProgressRunning,
	}); err != nil {
		t.Fatalf("SetProgressBarState() error = %v", err)
	}
	if err := transaction.SetMeterState(meter, MeterState{
		Value: 50, Minimum: 0, Maximum: 100,
		Status: ProgressRunning,
	}); err != nil {
		t.Fatalf("SetMeterState() error = %v", err)
	}
	before := app.Snapshot().Sequence
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got := app.Snapshot().Sequence; got != before+1 {
		t.Fatalf("batched progress update published %d after %d", got, before)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bar.Update(cancelled, ProgressBarState{
		Current: 75, Total: 100,
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Update() error = %v", err)
	}
	if bar.State().Current != 50 {
		t.Fatalf("cancelled Update changed state to %#v", bar.State())
	}

	const workers = 8
	var wait sync.WaitGroup
	failures := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for attempt := 0; attempt < 20; attempt++ {
				value := uint64((worker*20 + attempt) % 101)
				err := bar.Update(context.Background(), ProgressBarState{
					Current: value, Total: 100, Status: ProgressRunning,
				})
				if err == nil {
					_ = app.Snapshot()
					continue
				}
				if !errors.Is(err, ErrMutationBusy) {
					failures <- err
					return
				}
			}
		}(worker)
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent Update() error = %v", err)
	}
	state := bar.State()
	if state.Current > state.Total {
		t.Fatalf("concurrent final state = %#v", state)
	}
}

func TestProgressSnapshotDetailsAreIndependent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 2})
	if _, err := NewProgressBar(app.Root(), ProgressBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "bar"},
		State:        ProgressBarState{Current: 1, Total: 2},
	}); err != nil {
		t.Fatalf("NewProgressBar() error = %v", err)
	}
	first := app.Snapshot()
	controlByKey(t, first, "bar").Details.Progress.Current = 2
	if got := progressDetailsByKey(t, app, "bar").Current; got != 1 {
		t.Fatalf("snapshot mutation changed canonical current to %d", got)
	}
}

func TestScaledProgressExactBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		current uint64
		total   uint64
		scale   uint64
		want    uint64
	}{
		{0, 0, 10, 0},
		{0, 3, 10, 0},
		{1, 3, 10, 3},
		{2, 3, 10, 6},
		{3, 3, 10, 10},
		{math.MaxUint64 - 1, math.MaxUint64, 4_000_000, 3_999_999},
	}
	for _, test := range tests {
		if got := scaledProgress(
			test.current,
			test.total,
			test.scale,
		); got != test.want {
			t.Fatalf("scaledProgress(%d,%d,%d)=%d, want %d",
				test.current, test.total, test.scale, got, test.want)
		}
	}
}
