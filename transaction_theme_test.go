package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTransactionPublishesOneAtomicConstructionAndResize(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 12, Height: 6})
	before := app.Snapshot()

	transaction := app.NewTransaction()
	frame, err := transaction.NewFrame(app.Root(), FrameOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "atomic.frame",
			Bounds:        Rect{X: 1, Y: 1, Width: 8, Height: 4},
			Style:         "frame.fill",
		},
		Title:       "Atomic",
		BorderStyle: "frame.border",
	})
	if err != nil {
		t.Fatalf("Transaction.NewFrame() error = %v", err)
	}
	child, err := transaction.NewPanel(frame, PanelOptions{
		AutomationKey: "atomic.child",
		Bounds:        Rect{X: 1, Y: 1, Width: 3, Height: 1},
		Style:         "child",
	})
	if err != nil {
		t.Fatalf("Transaction.NewPanel() error = %v", err)
	}
	if frame.ID() != "" || child.ID() != "" {
		t.Fatal("provisional controls exposed runtime IDs before commit")
	}
	if got := len(app.Snapshot().Controls); got != 1 {
		t.Fatalf("pre-commit control count = %d, want root only", got)
	}

	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("Transaction.Commit() error = %v", err)
	}
	constructed := app.Snapshot()
	if constructed.Sequence != before.Sequence+1 {
		t.Fatalf("construction sequence = %d, want %d",
			constructed.Sequence, before.Sequence+1)
	}
	if len(constructed.Controls) != 3 {
		t.Fatalf("constructed controls = %d, want 3", len(constructed.Controls))
	}
	children := app.Root().Children()
	if len(children) != 1 {
		t.Fatalf("root children = %d, want 1", len(children))
	}
	if _, ok := children[0].(*Frame); !ok {
		t.Fatalf("root child type = %T, want *Frame", children[0])
	}
	if child.Parent() != frame {
		t.Fatalf("child parent = %T %v, want canonical Frame", child.Parent(), child.Parent())
	}
	if err := transaction.Commit(context.Background()); err == nil {
		t.Fatal("second Transaction.Commit() error = nil")
	}

	resize := app.NewTransaction()
	if err := resize.SetSize(Size{Width: 16, Height: 8}); err != nil {
		t.Fatalf("Transaction.SetSize() error = %v", err)
	}
	if err := resize.SetBounds(
		frame,
		Rect{X: 2, Y: 1, Width: 12, Height: 6},
	); err != nil {
		t.Fatalf("Transaction.SetBounds() error = %v", err)
	}
	if err := resize.Commit(context.Background()); err != nil {
		t.Fatalf("resize Commit() error = %v", err)
	}
	resized := app.Snapshot()
	if resized.Sequence != constructed.Sequence+1 {
		t.Fatalf("resize sequence = %d, want %d",
			resized.Sequence, constructed.Sequence+1)
	}
	if resized.Frame.Size != (Size{Width: 16, Height: 8}) ||
		controlByKey(t, resized, "atomic.frame").Bounds !=
			(Rect{X: 2, Y: 1, Width: 12, Height: 6}) {
		t.Fatalf("atomic resize snapshot = size:%+v frame:%+v",
			resized.Frame.Size,
			controlByKey(t, resized, "atomic.frame").Bounds)
	}
}

func TestFailedTransactionIsAtomicAndAbortsProvisionalHandles(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 6, Height: 3})
	before := app.Snapshot()

	transaction := app.NewTransaction()
	panel, err := transaction.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "missing-style",
		Bounds:        Rect{Width: 2, Height: 1},
		Style:         "theme.missing",
	})
	if err != nil {
		t.Fatalf("Transaction.NewPanel() error = %v", err)
	}
	if err := transaction.Commit(context.Background()); !errors.Is(
		err,
		ErrStyleMissing,
	) {
		t.Fatalf("Commit() error = %v, want ErrStyleMissing", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence ||
		len(after.Controls) != len(before.Controls) {
		t.Fatalf("failed transaction published state: before=%d/%d after=%d/%d",
			before.Sequence, len(before.Controls),
			after.Sequence, len(after.Controls))
	}
	if panel.ID() != "" || panel.Visible() {
		t.Fatalf("aborted provisional handle remains live: ID=%q Visible=%v",
			panel.ID(), panel.Visible())
	}
	if err := panel.SetBounds(Rect{Width: 1, Height: 1}); !errors.Is(
		err,
		ErrInvalidControl,
	) {
		t.Fatalf("aborted handle SetBounds() error = %v, want ErrInvalidControl", err)
	}
}

func TestThemeReplacementResolvesSemanticStylesAtomically(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "accent",
		Bounds:        Rect{X: 1, Width: 2, Height: 1},
		Style:         "before",
	})
	before := app.Snapshot()

	replacement, err := NewTheme(
		testStyle("root", RGB(8, 8, 8)),
		testStyle("before", RGB(20, 30, 40)),
	)
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	if err := app.SetTheme(replacement); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence+1 || panel.Style() != "before" {
		t.Fatalf("theme replacement sequence/style = %d/%q",
			after.Sequence, panel.Style())
	}
	view := controlByKey(t, after, "accent")
	if view.ResolvedStyle.Background != RGB(20, 30, 40) {
		t.Fatalf("resolved control style = %+v", view.ResolvedStyle)
	}
	cell := cellAt(t, after, 1, 0)
	if cell.Style != "before" || cell.Background != RGB(20, 30, 40) {
		t.Fatalf("theme-resolved cell = %+v", cell)
	}

	finalTheme, err := NewTheme(
		testStyle("root", RGB(9, 9, 9)),
		testStyle("alternate", RGB(40, 50, 60)),
	)
	if err != nil {
		t.Fatalf("NewTheme(final) error = %v", err)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetStyle(panel, "alternate"); err != nil {
		t.Fatalf("Transaction.SetStyle() error = %v", err)
	}
	if err := transaction.SetTheme(finalTheme); err != nil {
		t.Fatalf("Transaction.SetTheme() error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("style-plus-theme Commit() error = %v", err)
	}
	final := app.Snapshot()
	if final.Sequence != after.Sequence+1 ||
		controlByKey(t, final, "accent").Style != "alternate" ||
		cellAt(t, final, 1, 0).Background != RGB(40, 50, 60) {
		t.Fatalf("style-plus-theme final snapshot = %#v",
			controlByKey(t, final, "accent"))
	}
}

func TestDestroyAndSameKeyReplacementUseFinalTreeState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	old := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "replace",
		Bounds:        Rect{Width: 1, Height: 1},
		Style:         "before",
	})

	transaction := app.NewTransaction()
	if err := transaction.Destroy(old); err != nil {
		t.Fatalf("Transaction.Destroy() error = %v", err)
	}
	replacement, err := transaction.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "replace",
		Bounds:        Rect{X: 1, Width: 1, Height: 1},
		Style:         "after",
	})
	if err != nil {
		t.Fatalf("Transaction.NewPanel(replacement) error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("replacement Commit() error = %v", err)
	}
	resolved, found := app.ControlByAutomationKey("replace")
	if !found {
		t.Fatal("replacement lookup did not find the reused key")
	}
	if resolved.ID() != replacement.ID() {
		t.Fatalf("replacement lookup = %T/%q, want %q",
			resolved, resolved.ID(), replacement.ID())
	}
	if err := old.SetVisible(false); !errors.Is(err, ErrDestroyed) {
		t.Fatalf("destroyed handle mutation error = %v, want ErrDestroyed", err)
	}
}

func TestBorderTitlesAreRejectedBeforeNormalizationCanExceedBounds(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "source bytes",
			title: strings.Repeat("x", MaxTitleBytes+1),
		},
		{
			name:  "display cells",
			title: strings.Repeat("x", MaxTitleCells+1),
		},
		{
			name: "one cell bytes",
			title: "e" + strings.Repeat(
				"\u0301",
				MaxCellBytes/len("\u0301"),
			),
		},
	}
	for index, test := range tests {
		before := app.Snapshot().Sequence
		_, err := NewFrame(app.Root(), FrameOptions{
			PanelOptions: PanelOptions{
				AutomationKey: fmt.Sprintf("oversized-title-%d", index),
				Bounds:        Rect{Width: 4, Height: 2},
				Style:         "frame.fill",
			},
			Title:       test.title,
			BorderStyle: "frame.border",
		})
		if !errors.Is(err, ErrTextLimit) {
			t.Errorf("%s NewFrame() error = %v, want ErrTextLimit",
				test.name, err)
		}
		if got := app.Snapshot().Sequence; got != before {
			t.Errorf("%s published sequence %d, want %d",
				test.name, got, before)
		}
	}
}

func TestBorderTitleSnapshotStoresBoundedCanonicalText(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 8, Height: 3})
	frame, err := NewFrame(app.Root(), FrameOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "canonical-title",
			Bounds:        Rect{Width: 8, Height: 3},
			Style:         "frame.fill",
		},
		Title:       string([]byte{'A', 0xff}) + "界B",
		BorderStyle: "frame.border",
	})
	if err != nil {
		t.Fatalf("NewFrame(canonical title) error = %v", err)
	}
	control := controlByKey(t, app.Snapshot(), frame.AutomationKey())
	if control.Details.Border == nil {
		t.Fatal("Frame snapshot omits border details")
	}
	title := control.Details.Border.Title
	if !utf8.ValidString(title) || title != "A��B" ||
		len(title) > MaxTitleBytes {
		t.Fatalf("canonical title = %q (%d bytes)", title, len(title))
	}
}
