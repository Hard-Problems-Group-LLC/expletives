package expletives

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func markdownDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) *MarkdownDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.Markdown
	if details == nil {
		t.Fatalf("control %q has no Markdown details", key)
	}
	return details
}

func markdownOwnedRows(
	t *testing.T,
	snapshot Snapshot,
	key string,
) []string {
	t.Helper()
	control := controlByKey(t, snapshot, key)
	rows := make([]string, control.AbsoluteBounds.Height)
	for y := range control.AbsoluteBounds.Height {
		var row strings.Builder
		for x := range control.AbsoluteBounds.Width {
			cell, ok := snapshot.Frame.Cell(
				control.AbsoluteBounds.X+x,
				control.AbsoluteBounds.Y+y,
			)
			if ok && cell.Owner == control.ID {
				row.WriteString(cell.Grapheme)
			} else {
				row.WriteString(" ")
			}
		}
		rows[y] = row.String()
	}
	return rows
}

func TestMarkdownViewNormalizesParsesAndRendersSupportedSubset(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 72, Height: 24})
	source := "# C# Guide ##\r\n\r\n" +
		"Paragraph with *emphasis*, **strong**, `code`, and " +
		"[site](https://example.test).\r\n\r\n" +
		"- first\r\n- second\r\n\r\n" +
		"> quoted text\r\n\r\n---\r\n\r\n" +
		"```go\r\nfmt.Println(\"wide: 界\")\r\n```\r\n\r\n" +
		"<em>literal HTML</em>"
	view, err := NewMarkdownView(app.Root(), MarkdownViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "markdown",
				Bounds:        Rect{Width: 72, Height: 24},
			}},
			BorderForm:    BorderNone,
			HorizontalBar: ScrollBarVisibilityAuto,
			VerticalBar:   ScrollBarVisibilityAuto,
		},
		Markdown: source,
	})
	if err != nil {
		t.Fatalf("NewMarkdownView() error = %v", err)
	}
	control := controlByKey(t, app.Snapshot(), "markdown")
	if control.Kind != ControlMarkdownView || view.AutomationKey() != "markdown" {
		t.Fatalf("Markdown identity = %q/%q", control.Kind, view.AutomationKey())
	}
	if strings.Contains(view.Markdown(), "\r") ||
		!strings.Contains(view.Markdown(), "C# Guide") ||
		!strings.Contains(view.Markdown(), "\uFFFD") {
		t.Fatalf("canonical Markdown = %q", view.Markdown())
	}

	snapshot := app.Snapshot()
	details := markdownDetailsByKey(t, app, "markdown")
	if details.SourceBytes != len(view.Markdown()) ||
		details.SourceCells <= 0 || details.BlockCount != 13 ||
		details.RenderedRows < details.BlockCount ||
		details.Viewport.Content != "" || details.Viewport.ContentKey != "" ||
		details.Viewport.State.ContentSize != (Size{
			Width: details.MaximumLineWidth, Height: details.RenderedRows,
		}) {
		t.Fatalf("Markdown details = %+v", details)
	}
	wantKinds := []string{
		"heading", "blank", "blank", "paragraph",
	}
	for index, want := range wantKinds {
		if details.Blocks[index].Kind != want {
			t.Fatalf("block %d kind = %q, want %q", index, details.Blocks[index].Kind, want)
		}
	}
	if !details.SummariesTruncated || len(details.Blocks) != MaxMarkdownSummaries {
		t.Fatalf("bounded block summaries = %+v", details)
	}
	joined := strings.Join(markdownOwnedRows(t, snapshot, "markdown"), "\n")
	for _, fragment := range []string{
		"C# Guide",
		"site (https://example.test)",
		"• first",
		"│ quoted text",
		"fmt.Println(\"wide: \uFFFD\")",
		"<em>literal HTML</em>",
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("rendered Markdown omitted %q:\n%s", fragment, joined)
		}
	}
	styles := map[StyleID]bool{}
	owner := controlByKey(t, snapshot, "markdown").ID
	for _, cell := range snapshot.Frame.Cells {
		if cell.Owner == owner {
			styles[cell.Style] = true
		}
	}
	for _, style := range []StyleID{
		"markdown.heading", "markdown.emphasis", "markdown.strong",
		"markdown.code", "markdown.link", "markdown.quote",
		"markdown.list_marker", "markdown.rule",
	} {
		if !styles[style] {
			t.Fatalf("rendered frame omitted semantic style %q: %v", style, styles)
		}
	}
}

func TestMarkdownViewReflowsProsePreservesCodeAndClampsResize(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 18, Height: 8})
	view, err := NewMarkdownView(app.Root(), MarkdownViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "markdown.reflow",
				Bounds:        Rect{Width: 18, Height: 8},
			}},
			BorderForm:    BorderSingle,
			HorizontalBar: ScrollBarVisibilityAuto,
			VerticalBar:   ScrollBarVisibilityAuto,
		},
		Markdown: "words wrap across a narrow viewport without horizontal overflow\n\n" +
			"```\n0123456789abcdefghijklmnop\n```\n\nlast line",
	})
	if err != nil {
		t.Fatal(err)
	}
	details := markdownDetailsByKey(t, app, "markdown.reflow")
	if !details.Viewport.HorizontalVisible || !details.Viewport.VerticalVisible ||
		details.MaximumLineWidth != 26 || details.RenderedRows < 7 {
		t.Fatalf("reflow details = %+v", details)
	}
	if err := view.SetOffset(Point{X: 100, Y: 100}); err != nil {
		t.Fatal(err)
	}
	if got := view.Offset(); got != details.Viewport.MaximumOffset {
		t.Fatalf("clamped offset = %+v, want %+v", got, details.Viewport.MaximumOffset)
	}
	if err := view.SetBounds(Rect{Width: 10, Height: 5}); err != nil {
		t.Fatal(err)
	}
	resized := markdownDetailsByKey(t, app, "markdown.reflow")
	if resized.RenderedRows <= details.RenderedRows ||
		view.Offset().X > resized.Viewport.MaximumOffset.X ||
		view.Offset().Y > resized.Viewport.MaximumOffset.Y {
		t.Fatalf("resized Markdown details = %+v offset=%+v", resized, view.Offset())
	}
}

func TestMarkdownViewKeyboardRoutingAndProgrammaticSilence(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 5})
	registerActionCommand(t, app, "markdown.changed", "Markdown moved", true)
	var mu sync.Mutex
	commands := []Command{}
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		mu.Lock()
		commands = append(commands, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatal(err)
	}
	view, err := NewMarkdownView(app.Root(), MarkdownViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "markdown.keys",
					Bounds:        Rect{Width: 20, Height: 5},
				},
				ChangeCommand: "markdown.changed",
			},
			BorderForm:  BorderSingle,
			VerticalBar: ScrollBarVisibilityAuto,
		},
		Markdown: strings.Repeat("line\n", 20),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := view.Focus(); err != nil {
		t.Fatal(err)
	}
	completion := pressKey(t, app, "markdown-end", KeyEnd)
	if completion.Command != "markdown.changed" ||
		completion.Outcome != OutcomeApplied ||
		view.Offset() != markdownDetailsByKey(t, app, "markdown.keys").Viewport.MaximumOffset {
		t.Fatalf("End completion=%+v offset=%+v", completion, view.Offset())
	}
	mu.Lock()
	commandCount := len(commands)
	mu.Unlock()
	if err := view.SetOffset(Point{}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if len(commands) != commandCount {
		t.Fatalf("programmatic SetOffset routed commands: %+v", commands)
	}
	mu.Unlock()
	completion = pressKey(t, app, "markdown-down", KeyDown)
	if completion.Command != "markdown.changed" || view.Offset().Y != 1 {
		t.Fatalf("Down completion=%+v offset=%+v", completion, view.Offset())
	}
}

func TestMarkdownViewUpdateTransactionsBoundsAndCopies(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	view, err := NewMarkdownView(app.Root(), MarkdownViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "markdown.update",
				Bounds:        Rect{Width: 32, Height: 8},
			}},
		},
		Markdown: "initial",
	})
	if err != nil {
		t.Fatal(err)
	}
	sequence := app.Snapshot().Sequence
	if err := view.Update(nil, "nope"); err == nil {
		t.Fatal("Update(nil) error = nil")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := view.Update(cancelled, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Update() error = %v", err)
	}
	if app.Snapshot().Sequence != sequence || view.Markdown() != "initial" {
		t.Fatal("rejected update mutated MarkdownView")
	}
	if err := view.SetMarkdown("next\rline"); err != nil {
		t.Fatal(err)
	}
	if view.Markdown() != "next\nline" || app.Snapshot().Sequence != sequence+1 {
		t.Fatalf("updated source=%q sequence=%d", view.Markdown(), app.Snapshot().Sequence)
	}
	if err := view.SetMarkdown(strings.Repeat("x", MaxContentBytes+1)); err == nil {
		t.Fatal("oversized Markdown update error = nil")
	}
	if err := view.SetMarkdown(strings.Repeat("\n", MaxMarkdownBlocks)); err == nil {
		t.Fatal("over-blocked Markdown update error = nil")
	}
	first := app.Snapshot()
	details := controlByKey(t, first, "markdown.update").Details.Markdown
	details.Blocks[0].Kind = "mutated"
	if got := markdownDetailsByKey(t, app, "markdown.update").Blocks[0].Kind; got == "mutated" {
		t.Fatal("Markdown block summaries alias caller snapshot")
	}
}

func TestMarkdownViewConcurrentUpdatesRemainAtomic(t *testing.T) {
	app := mustApp(t, Size{Width: 40, Height: 8})
	view, err := NewMarkdownView(app.Root(), MarkdownViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "markdown.concurrent",
				Bounds:        Rect{Width: 40, Height: 8},
			}},
		},
		Markdown: "initial",
	})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 8
	errorsByWorker := make(chan error, workers)
	var wait sync.WaitGroup
	for index := range workers {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsByWorker <- view.Update(
				context.Background(),
				"# worker\n\nvalue "+string(rune('a'+index)),
			)
		}(index)
	}
	wait.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		if err != nil {
			t.Fatalf("concurrent Update() error = %v", err)
		}
	}
	if details := markdownDetailsByKey(t, app, "markdown.concurrent"); details.BlockCount != 3 || details.SourceBytes != len(view.Markdown()) {
		t.Fatalf("final atomic Markdown details = %+v source=%q", details, view.Markdown())
	}
}

func TestMarkdownViewRejectsReservedOptionsAndAggregateOverflow(t *testing.T) {
	app := mustApp(t, Size{Width: 256, Height: 8})
	for name, mutate := range map[string]func(*MarkdownViewOptions){
		"content key": func(options *MarkdownViewOptions) {
			options.ContentAutomationKey = "reserved.content"
		},
		"content style": func(options *MarkdownViewOptions) {
			options.ContentStyle = "panel"
		},
		"content extent": func(options *MarkdownViewOptions) {
			options.State.ContentSize = Size{Width: 1, Height: 1}
		},
		"negative offset": func(options *MarkdownViewOptions) {
			options.State.Offset = Point{X: -1}
		},
	} {
		t.Run(name, func(t *testing.T) {
			options := MarkdownViewOptions{
				ScrollablePanelOptions: ScrollablePanelOptions{
					ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
						AutomationKey: "markdown.invalid." + strings.ReplaceAll(name, " ", "_"),
						Bounds:        Rect{Width: 10, Height: 4},
					}},
				},
				Markdown: "text",
			}
			mutate(&options)
			sequence := app.Snapshot().Sequence
			if _, err := NewMarkdownView(app.Root(), options); err == nil {
				t.Fatal("NewMarkdownView() error = nil")
			}
			if app.Snapshot().Sequence != sequence {
				t.Fatal("rejected Markdown construction published state")
			}
		})
	}

	source := strings.Repeat("x", MaxContentBytes)
	transaction := app.NewTransaction()
	for index := 0; index < MaxContentAggregateBytes/MaxContentBytes; index++ {
		if _, err := transaction.NewMarkdownView(
			app.Root(),
			MarkdownViewOptions{
				ScrollablePanelOptions: ScrollablePanelOptions{
					ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
						AutomationKey: "markdown.capacity." + string(rune('a'+index)),
						Bounds:        Rect{Width: 256, Height: 8},
					}},
				},
				Markdown: source,
			},
		); err != nil {
			t.Fatalf("NewMarkdownView(%d) error = %v", index, err)
		}
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("commit exact aggregate capacity: %v", err)
	}
	sequence := app.Snapshot().Sequence
	if _, err := NewMarkdownView(
		app.Root(),
		MarkdownViewOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{
				ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
					AutomationKey: "markdown.capacity.overflow",
					Bounds:        Rect{Width: 256, Height: 8},
				}},
			},
			Markdown: "x",
		},
	); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("aggregate overflow error = %v, want ErrControlCapacity", err)
	}
	if app.Snapshot().Sequence != sequence {
		t.Fatal("aggregate overflow published state")
	}
}

func TestMarkdownViewLiteralLinkMarkersAndNarrowWrapPreserveContent(t *testing.T) {
	source := strings.Repeat("[", 1024)
	cells := markdownInlineCells(source, "markdown_view")
	if len(cells) != len(source) {
		t.Fatalf("literal marker cells = %d, want %d", len(cells), len(source))
	}
	for index, cell := range cells {
		if cell.grapheme != "[" || cell.style != "markdown_view" {
			t.Fatalf("literal marker cell %d = %+v", index, cell)
		}
	}

	prefix := []markdownCell{{grapheme: "•", style: "markdown.list_marker"}, {grapheme: " ", style: "markdown.list_marker"}}
	rows := wrapMarkdownCells(append(append([]markdownCell{}, prefix...), cells...), 3, prefix)
	if len(rows) != len(source) {
		t.Fatalf("narrow wrapped rows = %d, want %d", len(rows), len(source))
	}
	for index, row := range rows {
		if len(row) != 3 || row[2].grapheme != "[" {
			t.Fatalf("narrow wrapped row %d = %+v", index, row)
		}
	}
}

func BenchmarkMarkdownViewMaximumLiteralLinkMarkers(b *testing.B) {
	source := "- " + strings.Repeat("[x", (MaxContentBytes-2)/2)
	b.ReportAllocs()
	for b.Loop() {
		blocks, err := parseMarkdown(source)
		if err != nil {
			b.Fatal(err)
		}
		behavior := reflowMarkdown(
			markdownBehavior{source: source, sourceCells: len(source), blocks: blocks},
			Size{Width: 80, Height: 24},
		)
		if len(behavior.rows) == 0 {
			b.Fatal("maximum source rendered no rows")
		}
	}
}
