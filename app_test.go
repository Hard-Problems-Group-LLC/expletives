package expletives

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func TestColorJSONUsesUppercaseHex(t *testing.T) {
	t.Parallel()
	color := RGB(0x0A, 0xBC, 0x01)
	encoded, err := json.Marshal(color)
	if err != nil {
		t.Fatalf("Marshal(Color) error = %v", err)
	}
	if got, want := string(encoded), `"#0ABC01"`; got != want {
		t.Fatalf("Marshal(Color) = %s, want %s", got, want)
	}

	var decoded Color
	if err := json.Unmarshal([]byte(`"#0abc01"`), &decoded); err != nil {
		t.Fatalf("Unmarshal(Color) error = %v", err)
	}
	if decoded != color {
		t.Fatalf("decoded Color = %+v, want %+v", decoded, color)
	}
}

func TestSnapshotAndHistoryAreDeepCopies(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 3})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "panel",
		Bounds:        Rect{X: 1, Y: 1, Width: 2, Height: 1},
		Style:         "panel",
	})
	snapshot := app.Snapshot()
	sequence := snapshot.Sequence

	snapshot.Frame.Cells[0].Grapheme = "X"
	snapshot.Frame.Cells[0].Owner = panel.ID()
	snapshot.Controls[0].Children[0] = "corrupted"
	snapshot.Controls[0].Style = "corrupted"
	snapshot.Controls[0].ResolvedStyle.Background = RGB(0, 0, 0)
	if len(snapshot.Layouts) == 0 {
		snapshot.Layouts = append(snapshot.Layouts, LayoutSnapshot{
			ID: "caller-owned",
			Items: []LayoutItemSnapshot{{
				Kind: "panel",
			}},
		})
	}
	if len(snapshot.Overflows) == 0 {
		snapshot.Overflows = append(snapshot.Overflows, OverflowSnapshot{
			EpisodeID: "caller-owned",
		})
	}

	current := app.Snapshot()
	if got := current.Frame.Cells[0].Grapheme; got != " " {
		t.Fatalf("snapshot mutation changed published frame: %q", got)
	}
	if got := current.Controls[0].Children[0]; got != panel.ID() {
		t.Fatalf("snapshot mutation changed published children: %q", got)
	}
	if got := current.Controls[0].Style; got == "corrupted" {
		t.Fatal("snapshot mutation changed published control style")
	}
	if len(current.Layouts) != 0 {
		t.Fatalf("snapshot mutation changed published Layout state: %#v", current.Layouts)
	}
	if len(current.Overflows) != 0 {
		t.Fatalf("snapshot mutation changed published overflow state: %#v", current.Overflows)
	}

	retained, err := app.SnapshotAt(sequence)
	if err != nil {
		t.Fatalf("SnapshotAt(%d) error = %v", sequence, err)
	}
	retained.Frame.Cells[0].Grapheme = "Y"
	again, err := app.SnapshotAt(sequence)
	if err != nil {
		t.Fatalf("second SnapshotAt(%d) error = %v", sequence, err)
	}
	if again.Frame.Cells[0].Grapheme != " " {
		t.Fatal("SnapshotAt exposed mutable retained storage")
	}
}

func TestSnapshotClonePreservesRequiredEmptyOverflowArray(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 1, Height: 1})

	for name, snapshot := range map[string]SnapshotV1{
		"current":  app.Snapshot(),
		"retained": mustSnapshotAt(t, app, app.Snapshot().Sequence),
	} {
		encoded, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatalf("%s Marshal(SnapshotV1) error = %v", name, err)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &object); err != nil {
			t.Fatalf("%s Unmarshal(snapshot object) error = %v", name, err)
		}
		if got, want := string(object["overflows"]), "[]"; got != want {
			t.Fatalf("%s overflows JSON = %s, want %s", name, got, want)
		}
	}
}

func TestSnapshotHistoryIsBoundedAndExact(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 3})
	first := app.Snapshot().Sequence

	for index := 0; index < SnapshotRetention+4; index++ {
		width := 4
		if index%2 == 0 {
			width = 5
		}
		if err := app.SetSize(Size{Width: width, Height: 3}); err != nil {
			t.Fatalf("SetSize iteration %d error = %v", index, err)
		}
	}

	if _, err := app.SnapshotAt(first); !errors.Is(err, ErrSnapshotNotRetained) {
		t.Fatalf("SnapshotAt(expired) error = %v, want ErrSnapshotNotRetained", err)
	}
	latest := app.Snapshot()
	exact, err := app.SnapshotAt(latest.Sequence)
	if err != nil {
		t.Fatalf("SnapshotAt(latest) error = %v", err)
	}
	if exact.Sequence != latest.Sequence || exact.Frame.Size != latest.Frame.Size {
		t.Fatalf("exact snapshot = seq %d size %+v, want seq %d size %+v",
			exact.Sequence, exact.Frame.Size, latest.Sequence, latest.Frame.Size)
	}
}

func mustSnapshotAt(t *testing.T, app *App, sequence uint64) SnapshotV1 {
	t.Helper()
	snapshot, err := app.SnapshotAt(sequence)
	if err != nil {
		t.Fatalf("SnapshotAt(%d) error = %v", sequence, err)
	}
	return snapshot
}

func TestWaitSnapshotReturnsLaterPublication(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 3, Height: 2})
	after := app.Snapshot().Sequence
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(ready)
		snapshot, err := app.WaitSnapshot(context.Background(), after)
		if err == nil && snapshot.Sequence <= after {
			err = errors.New("WaitSnapshot returned a non-later sequence")
		}
		done <- err
	}()
	<-ready
	if err := app.SetSize(Size{Width: 4, Height: 2}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("WaitSnapshot() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := app.WaitSnapshot(ctx, app.Snapshot().Sequence); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("cancelled WaitSnapshot() error = %v, want context.Canceled", err)
	}
}

func TestConcurrentSnapshotReadersObserveCompleteFrames(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 10, Height: 5})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "moving",
		Bounds:        Rect{X: 1, Y: 1, Width: 3, Height: 2},
		Style:         "moving",
	})

	const readers = 8
	const iterations = 100
	start := make(chan struct{})
	var wait sync.WaitGroup
	errs := make(chan error, readers)
	for reader := 0; reader < readers; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			for iteration := 0; iteration < iterations; iteration++ {
				snapshot := app.Snapshot()
				wantCells := snapshot.Frame.Size.Width * snapshot.Frame.Size.Height
				if len(snapshot.Frame.Cells) != wantCells {
					errs <- errors.New("snapshot contains a torn frame")
					return
				}
				if len(snapshot.Controls) != 2 {
					errs <- errors.New("snapshot contains a torn control tree")
					return
				}
			}
		}()
	}
	close(start)
	for iteration := 0; iteration < iterations; iteration++ {
		x := iteration % 6
		if err := panel.SetBounds(Rect{X: x, Y: 1, Width: 3, Height: 2}); err != nil {
			t.Fatalf("SetBounds iteration %d error = %v", iteration, err)
		}
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
