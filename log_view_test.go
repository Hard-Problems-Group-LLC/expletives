package expletives

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func logDetailsByKey(t *testing.T, app *App, key string) *LogViewDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.LogView
	if details == nil {
		t.Fatalf("control %q has no LogView details", key)
	}
	return details
}

func streamDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) *StreamViewDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.StreamView
	if details == nil {
		t.Fatalf("control %q has no StreamView details", key)
	}
	return details
}

func TestLogViewNormalizesRendersAndExposesBoundedDetails(t *testing.T) {
	app := mustApp(t, Size{Width: 40, Height: 8})
	view, err := NewLogView(app.Root(), LogViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "log.primary",
				Bounds:        Rect{Width: 40, Height: 8},
			}},
			BorderForm:    BorderSingle,
			HorizontalBar: ScrollBarVisibilityAuto,
			VerticalBar:   ScrollBarVisibilityAuto,
		},
		Capacity: ContentCapacity{Records: 8, Bytes: 1024},
		Records: []LogRecord{
			{Key: "first", Timestamp: "12:00", Level: LogDebug, Text: "debug"},
			{Key: "second", Level: LogWarning, Text: "line one\nwide 界 and escape \x1b"},
			{Key: "third", Level: LogError, Text: "failure"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	state := view.State()
	if state.Follow || state.RetainedRecords != 3 || state.DroppedRecords != 0 ||
		state.PendingBytes != 0 {
		t.Fatalf("initial LogView state = %+v", state)
	}
	details := logDetailsByKey(t, app, "log.primary")
	if details.Capacity != (ContentCapacity{Records: 8, Bytes: 1024}) ||
		details.FirstKey != "first" || details.LastKey != "third" ||
		details.RetainedRecords != 3 || details.RetainedBytes != state.RetainedBytes ||
		details.Viewport.Content != "" || details.Viewport.ContentKey != "" {
		t.Fatalf("LogView details = %+v", details)
	}
	details.FirstKey = "caller-mutated"
	if details.Viewport.HorizontalBar != nil {
		details.Viewport.HorizontalBar.Offset = 999
	}
	freshDetails := logDetailsByKey(t, app, "log.primary")
	if freshDetails.FirstKey != "first" ||
		(freshDetails.Viewport.HorizontalBar != nil &&
			freshDetails.Viewport.HorizontalBar.Offset == 999) {
		t.Fatal("LogView snapshot details exposed retained mutable storage")
	}
	control := controlByKey(t, app.Snapshot(), "log.primary")
	if control.Kind != ControlLogView || len(control.Children) != 0 {
		t.Fatalf("LogView snapshot = %+v", control)
	}
	styles := map[StyleID]bool{}
	replacements := 0
	snapshot := app.Snapshot()
	for y := control.AbsoluteBounds.Y; y <
		control.AbsoluteBounds.Y+control.AbsoluteBounds.Height; y++ {
		for x := control.AbsoluteBounds.X; x <
			control.AbsoluteBounds.X+control.AbsoluteBounds.Width; x++ {
			cell, ok := snapshot.Frame.Cell(x, y)
			if !ok || cell.Owner != control.ID {
				continue
			}
			styles[cell.Style] = true
			if cell.Grapheme == "�" {
				replacements++
			}
		}
	}
	for _, style := range []StyleID{
		"log.timestamp", "log.debug", "log.warning", "log.error",
	} {
		if !styles[style] {
			t.Fatalf("rendered LogView styles = %+v, missing %q", styles, style)
		}
	}
	if replacements != 2 {
		t.Fatalf("replacement cells = %d, want wide and escape replacements", replacements)
	}
}

func TestLogViewCapacityDropsAreExactAndMutationsAreAtomic(t *testing.T) {
	app := mustApp(t, Size{Width: 30, Height: 6})
	view, err := NewLogView(app.Root(), LogViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "log.capacity",
				Bounds:        Rect{Width: 30, Height: 6},
			}},
		},
		Capacity: ContentCapacity{Records: 2, Bytes: 64},
		Records: []LogRecord{
			{Key: "a", Level: LogInfo, Text: "A"},
			{Key: "b", Level: LogInfo, Text: "B"},
			{Key: "c", Level: LogInfo, Text: "C"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	state := view.State()
	if state.RetainedRecords != 2 || state.DroppedRecords != 1 ||
		state.DroppedBytes != 6 {
		t.Fatalf("capacity state = %+v, want one 6-byte record dropped", state)
	}
	if details := logDetailsByKey(t, app, "log.capacity"); details.FirstKey != "b" || details.LastKey != "c" {
		t.Fatalf("capacity keys = %+v", details)
	}
	sequence := app.Snapshot().Sequence
	if err := view.Append(context.Background(), []LogRecord{
		{Key: "c", Level: LogInfo, Text: "duplicate"},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("duplicate Append error = %v", err)
	}
	if app.Snapshot().Sequence != sequence || view.State() != state {
		t.Fatal("rejected duplicate append published state")
	}
	if err := view.Replace(context.Background(), []LogRecord{
		{Key: "replacement", Level: LogDebug, Text: "new"},
	}); err != nil {
		t.Fatal(err)
	}
	state = view.State()
	if state.RetainedRecords != 1 || state.DroppedRecords != 0 ||
		state.DroppedBytes != 0 {
		t.Fatalf("replacement state = %+v", state)
	}
	if err := view.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
	if state = view.State(); state.RetainedRecords != 0 ||
		state.RetainedBytes != 0 || state.DroppedRecords != 0 ||
		state.Offset != (Point{}) {
		t.Fatalf("cleared state = %+v", state)
	}
}

func TestLogViewFollowScrollbackAndUserOnlyNotification(t *testing.T) {
	app := mustApp(t, Size{Width: 24, Height: 5})
	registerActionCommand(t, app, "log.changed", "Log changed", true)
	if err := app.SetCommandRouter(func(
		context.Context,
		Command,
	) CommandResult {
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatal(err)
	}
	view, err := NewLogView(app.Root(), LogViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "log.follow",
					Bounds:        Rect{Width: 24, Height: 5},
				},
				ChangeCommand: "log.changed",
			},
			VerticalBar: ScrollBarVisibilityAuto,
		},
		Capacity: ContentCapacity{Records: 32, Bytes: 2048},
		Follow:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := make([]LogRecord, 12)
	for index := range records {
		records[index] = LogRecord{
			Key: fmt.Sprintf("row.%02d", index), Level: LogInfo,
			Text: fmt.Sprintf("row %02d", index),
		}
	}
	if err := view.Append(context.Background(), records); err != nil {
		t.Fatal(err)
	}
	if state := view.State(); !state.Follow || state.Offset.Y == 0 {
		t.Fatalf("following append state = %+v", state)
	}
	if err := view.Focus(); err != nil {
		t.Fatal(err)
	}
	up := pressKey(t, app, "log-up", KeyUp)
	paused := view.State()
	if up.Command != "log.changed" || up.Outcome != OutcomeApplied ||
		paused.Follow {
		t.Fatalf("Up completion=%+v state=%+v", up, paused)
	}
	if err := view.Append(context.Background(), []LogRecord{
		{Key: "row.12", Level: LogInfo, Text: "row 12"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := view.State(); got.Follow || got.Offset.Y != paused.Offset.Y {
		t.Fatalf("paused append moved anchor: before=%+v after=%+v", paused, got)
	}
	end := pressKey(t, app, "log-end", KeyEnd)
	if got := view.State(); end.Command != "log.changed" || !got.Follow ||
		got.Offset != logDetailsByKey(t, app, "log.follow").Viewport.MaximumOffset {
		t.Fatalf("End completion=%+v state=%+v", end, got)
	}
	sequence := app.Snapshot().Sequence
	if err := view.SetOffset(Point{}); err != nil {
		t.Fatal(err)
	}
	if app.Snapshot().Sequence <= sequence || view.State().Follow {
		t.Fatalf("programmatic SetOffset state = %+v", view.State())
	}
}

func TestStreamViewChunkBoundariesNeutralizationAndFlush(t *testing.T) {
	app := mustApp(t, Size{Width: 40, Height: 8})
	view, err := NewStreamView(app.Root(), StreamViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "stream.chunks",
				Bounds:        Rect{Width: 40, Height: 8},
			}},
			BorderForm:  BorderSingle,
			VerticalBar: ScrollBarVisibilityAuto,
		},
		Capacity: ContentCapacity{Records: 16, Bytes: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := view.Append(context.Background(), []byte("one\r"))
	if err != nil || result.CompletedLines != 1 {
		t.Fatalf("first Append result=%+v err=%v", result, err)
	}
	result, err = view.Append(context.Background(), []byte("\ntwo\nthree e\xcc"))
	if err != nil || result.CompletedLines != 1 || result.InputBytes != 13 {
		t.Fatalf("second Append result=%+v err=%v", result, err)
	}
	result, err = view.Append(
		context.Background(),
		[]byte{0x81, ' ', 0x1b, ' ', 0xe7, 0x95, 0x8c},
	)
	if err != nil || result.CompletedLines != 0 {
		t.Fatalf("third Append result=%+v err=%v", result, err)
	}
	state := view.State()
	if state.RetainedRecords != 2 || state.PendingBytes == 0 ||
		state.DroppedRecords != 0 {
		t.Fatalf("chunked stream state = %+v", state)
	}
	if err := view.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	state = view.State()
	if state.RetainedRecords != 3 || state.PendingBytes != 0 {
		t.Fatalf("flushed stream state = %+v", state)
	}
	details := streamDetailsByKey(t, app, "stream.chunks")
	if details.RetainedLines != 3 || details.PendingBytes != 0 ||
		details.Viewport.Content != "" || details.Viewport.ContentKey != "" {
		t.Fatalf("StreamView details = %+v", details)
	}
	details.RetainedLines = 999
	if fresh := streamDetailsByKey(t, app, "stream.chunks"); fresh.RetainedLines != 3 {
		t.Fatal("StreamView snapshot details exposed retained mutable storage")
	}
	text := ""
	for _, row := range markdownOwnedRows(t, app.Snapshot(), "stream.chunks") {
		text += row + "\n"
	}
	if !strings.Contains(text, "one") || !strings.Contains(text, "two") ||
		!strings.Contains(text, "three é � �") {
		t.Fatalf("rendered stream text = %q", text)
	}
}

func TestStreamViewTruncationEvictionAndDropAccounting(t *testing.T) {
	app := mustApp(t, Size{Width: 30, Height: 6})
	view, err := NewStreamView(app.Root(), StreamViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "stream.drops",
				Bounds:        Rect{Width: 30, Height: 6},
			}},
		},
		Capacity: ContentCapacity{Records: 2, Bytes: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := view.Append(
		context.Background(),
		[]byte("abcdefghijklmnopqrst\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.CompletedLines != 1 || result.DroppedLines != 1 ||
		result.DroppedBytes != 19 {
		t.Fatalf("truncating Append result = %+v", result)
	}
	details := streamDetailsByKey(t, app, "stream.drops")
	if details.DroppedLines != 1 || details.DroppedBytes != 19 ||
		details.RetainedBytes != 12 {
		t.Fatalf("truncated details = %+v", details)
	}
	text := ""
	for _, row := range markdownOwnedRows(t, app.Snapshot(), "stream.drops") {
		text += row
	}
	if !strings.Contains(text, "[truncated]") {
		t.Fatalf("truncation marker not visible in %q", text)
	}
	result, err = view.Append(context.Background(), []byte("z\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.DroppedLines != 0 || result.DroppedBytes != 1 {
		t.Fatalf("eviction of already-truncated line result = %+v", result)
	}
	state := view.State()
	if state.DroppedRecords != 1 || state.DroppedBytes != 20 ||
		state.RetainedRecords != 1 {
		t.Fatalf("post-eviction stream state = %+v", state)
	}
}

func TestLogAndStreamConcurrentUpdatesRemainAtomic(t *testing.T) {
	app := mustApp(t, Size{Width: 60, Height: 12})
	logView, err := NewLogView(app.Root(), LogViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "log.concurrent",
				Bounds:        Rect{Width: 30, Height: 6},
			}},
		},
		Capacity: ContentCapacity{Records: 64, Bytes: 4096},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := NewStreamView(app.Root(), StreamViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "stream.concurrent",
				Bounds:        Rect{X: 30, Width: 30, Height: 6},
			}},
		},
		Capacity: ContentCapacity{Records: 64, Bytes: 4096},
	})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 16
	errorsByWorker := make(chan error, workers*2)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		index := index
		wait.Add(2)
		go func() {
			defer wait.Done()
			errorsByWorker <- logView.Append(context.Background(), []LogRecord{{
				Key:   fmt.Sprintf("worker.%02d", index),
				Level: LogInfo,
				Text:  fmt.Sprintf("value %02d", index),
			}})
		}()
		go func() {
			defer wait.Done()
			_, err := stream.Append(
				context.Background(),
				[]byte(fmt.Sprintf("stream %02d\n", index)),
			)
			errorsByWorker <- err
		}()
	}
	wait.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		if err != nil {
			t.Fatalf("concurrent update error = %v", err)
		}
	}
	if state := logView.State(); state.RetainedRecords != workers {
		t.Fatalf("concurrent LogView state = %+v", state)
	}
	if state := stream.State(); state.RetainedRecords != workers {
		t.Fatalf("concurrent StreamView state = %+v", state)
	}
}

func TestContentUpdateCancellationWhileQueuedPublishesNothing(t *testing.T) {
	app := mustApp(t, Size{Width: 20, Height: 5})
	view, err := NewLogView(app.Root(), LogViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "log.cancel",
				Bounds:        Rect{Width: 20, Height: 5},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	view.state.contentGate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sequence := app.Snapshot().Sequence
	err = view.Append(ctx, []LogRecord{{
		Key: "cancelled", Level: LogInfo, Text: "not published",
	}})
	<-view.state.contentGate
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("queued cancelled Append error = %v", err)
	}
	if app.Snapshot().Sequence != sequence || view.State().RetainedRecords != 0 {
		t.Fatal("queued cancelled Append published state")
	}
}

func TestLogAndStreamShareExactAggregateContentBound(t *testing.T) {
	app := mustApp(t, Size{Width: 20, Height: 5})
	transaction := app.NewTransaction()
	text := strings.Repeat("x", MaxContentBytes-len("r")-len(LogInfo))
	for index := 0; index < MaxContentAggregateBytes/MaxContentBytes; index++ {
		if _, err := transaction.NewLogView(app.Root(), LogViewOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{
				ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
					AutomationKey: "log.aggregate." + string(rune('a'+index)),
					Bounds:        Rect{Width: 20, Height: 5},
				}},
			},
			Records: []LogRecord{{Key: "r", Level: LogInfo, Text: text}},
		}); err != nil {
			t.Fatalf("NewLogView(%d) error = %v", index, err)
		}
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("commit exact aggregate capacity: %v", err)
	}
	stream, err := NewStreamView(app.Root(), StreamViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "stream.aggregate",
				Bounds:        Rect{Width: 20, Height: 5},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sequence := app.Snapshot().Sequence
	if _, err := stream.Append(context.Background(), []byte("x")); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("aggregate overflow error = %v, want ErrControlCapacity", err)
	}
	if app.Snapshot().Sequence != sequence || stream.State().PendingBytes != 0 {
		t.Fatal("aggregate overflow published stream state")
	}
}

func TestStreamCanonicalPendingStorageUsesAggregateContentBound(t *testing.T) {
	app := mustApp(t, Size{Width: 20, Height: 5})
	chunk := bytes.Repeat([]byte{0xff}, MaxContentBytes/3)
	const exactViews = MaxContentAggregateBytes / (MaxContentBytes - 1)
	for index := 0; index < exactViews+1; index++ {
		view, err := NewStreamView(app.Root(), StreamViewOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{
				ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
					AutomationKey: fmt.Sprintf("stream.canonical.%02d", index),
					Bounds:        Rect{Width: 20, Height: 5},
				}},
			},
		})
		if err != nil {
			t.Fatalf("NewStreamView(%d) error = %v", index, err)
		}
		sequence := app.Snapshot().Sequence
		_, err = view.Append(context.Background(), chunk)
		if index < exactViews {
			if err != nil {
				t.Fatalf("Append(%d) error = %v", index, err)
			}
			continue
		}
		if !errors.Is(err, ErrControlCapacity) {
			t.Fatalf("overflow Append error = %v, want ErrControlCapacity", err)
		}
		if app.Snapshot().Sequence != sequence ||
			view.State().PendingBytes != 0 {
			t.Fatal("canonical aggregate overflow published stream state")
		}
	}
}

func BenchmarkStreamViewBoundedAppend(b *testing.B) {
	chunk := []byte(strings.Repeat("line data ", 4096) + "\n")
	for b.Loop() {
		app, err := NewApp(AppOptions{Size: Size{Width: 80, Height: 24}})
		if err != nil {
			b.Fatal(err)
		}
		view, err := NewStreamView(app.Root(), StreamViewOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{
				ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
					AutomationKey: "stream.benchmark",
					Bounds:        Rect{Width: 80, Height: 24},
				}},
			},
		})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := view.Append(context.Background(), chunk); err != nil {
			b.Fatal(err)
		}
	}
}
