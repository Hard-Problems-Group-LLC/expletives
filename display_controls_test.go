package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

var (
	_ Control = (*Label)(nil)
	_ Control = (*StaticText)(nil)
	_ Control = (*Separator)(nil)
	_ Control = (*Rule)(nil)
)

func TestDisplayControlsAreLeafControlsAndPublishTypedDetails(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 20, Height: 8})
	target := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "target",
		Bounds:        Rect{X: 15, Y: 1, Width: 2, Height: 2},
	})
	label, err := NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "label",
			Bounds:        Rect{X: 1, Y: 1, Width: 10, Height: 3},
		},
		Text:                "e\u0301界",
		HorizontalAlignment: TextAlignCenter,
		VerticalAlignment:   TextAlignEnd,
		Target:              target,
		Mnemonic:            "E",
	})
	if err != nil {
		t.Fatalf("NewLabel() error = %v", err)
	}
	if _, ok := any(label).(Container); ok {
		t.Fatal("Label implements Container")
	}
	if got := label.Text(); got != "e\u0301\uFFFD" {
		t.Fatalf("Label.Text() = %q, want composed cell plus replacement", got)
	}
	if got := label.MinimumSize(); got != (Size{Width: 2, Height: 1}) {
		t.Fatalf("Label minimum = %+v, want 2x1", got)
	}

	snapshot := app.Snapshot()
	details := controlByKey(t, snapshot, "label").Details
	if details.Container != nil || details.Border != nil ||
		details.Divider != nil || details.Text == nil {
		t.Fatalf("Label details union = %+v, want only Text", details)
	}
	if details.Text.Target != target.ID() || details.Text.Mnemonic != "e" ||
		details.Text.Wrap != TextWrapNone {
		t.Fatalf("Label Text details = %+v", *details.Text)
	}
	if got := cellAt(t, snapshot, 5, 3).Grapheme; got != "e\u0301" {
		t.Fatalf("centered composed cell = %q, want e+acute", got)
	}
	if got := cellAt(t, snapshot, 6, 3).Grapheme; got != "\uFFFD" {
		t.Fatalf("wide-character cell = %q, want replacement", got)
	}

	details.Text.Text = "caller mutation"
	if got := controlByKey(t, app.Snapshot(), "label").Details.Text.Text; got != "e\u0301\uFFFD" {
		t.Fatal("Snapshot() exposed mutable TextDetails storage")
	}

	if err := target.Destroy(); err != nil {
		t.Fatalf("target.Destroy() error = %v", err)
	}
	afterDestroy := controlByKey(t, app.Snapshot(), "label").Details.Text
	if afterDestroy.Target != "" || afterDestroy.Mnemonic != "" {
		t.Fatalf("destroyed target association was not cleared: %+v", *afterDestroy)
	}
}

func TestStaticTextWrapsWordsAndCellsWithoutSplittingGraphemes(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 12, Height: 10})
	staticText, err := NewStaticText(app.Root(), StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "static",
			Bounds:        Rect{X: 1, Y: 1, Width: 5, Height: 4},
		},
		Text: "one two\r\nabcdef",
		Wrap: TextWrapWords,
	})
	if err != nil {
		t.Fatalf("NewStaticText() error = %v", err)
	}
	if _, ok := any(staticText).(Container); ok {
		t.Fatal("StaticText implements Container")
	}
	if got := staticText.Text(); got != "one two\nabcdef" {
		t.Fatalf("StaticText.Text() = %q", got)
	}
	if got := staticText.MinimumSize(); got != (Size{Width: 1, Height: 2}) {
		t.Fatalf("wrapped StaticText minimum = %+v, want 1x2", got)
	}

	snapshot := app.Snapshot()
	for row, want := range []string{"one", "two", "abcde", "f"} {
		var got strings.Builder
		for column := 0; column < len(want); column++ {
			got.WriteString(cellAt(t, snapshot, 1+column, 1+row).Grapheme)
		}
		if got.String() != want {
			t.Fatalf("row %d = %q, want %q", row, got.String(), want)
		}
	}

	cellWrapped, err := NewStaticText(app.Root(), StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "cells",
			Bounds:        Rect{X: 7, Y: 1, Width: 2, Height: 2},
		},
		Text: "e\u0301AB",
		Wrap: TextWrapCells,
	})
	if err != nil {
		t.Fatalf("NewStaticText(cell wrap) error = %v", err)
	}
	snapshot = app.Snapshot()
	if got := cellAt(t, snapshot, 7, 1).Grapheme; got != "e\u0301" {
		t.Fatalf("wrapped grapheme = %q, want intact composed cell", got)
	}
	if got := cellAt(t, snapshot, 8, 1).Grapheme; got != "A" {
		t.Fatalf("wrapped first row tail = %q, want A", got)
	}
	if got := cellAt(t, snapshot, 7, 2).Grapheme; got != "B" {
		t.Fatalf("wrapped second row = %q, want B", got)
	}
	if _, ok := any(cellWrapped).(Container); ok {
		t.Fatal("cell-wrapped StaticText implements Container")
	}
}

func TestSeparatorAndRuleFormsOrientationsAndTitles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		form BorderForm
		want string
	}{
		{name: "none", form: BorderNone, want: " "},
		{name: "single", form: BorderSingle, want: "─"},
		{name: "double", form: BorderDouble, want: "═"},
		{name: "light shade", form: BorderShadeLight, want: "░"},
		{name: "medium shade", form: BorderShadeMedium, want: "▒"},
		{name: "dark shade", form: BorderShadeDark, want: "▓"},
		{name: "block", form: BorderBlock, want: "█"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app := mustApp(t, Size{Width: 8, Height: 8})
			horizontal, err := NewSeparator(app.Root(), SeparatorOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "horizontal",
					Bounds:        Rect{X: 1, Y: 1, Width: 3, Height: 1},
				},
				Orientation: Horizontal,
				Form:        test.form,
			})
			if err != nil {
				t.Fatalf("NewSeparator(horizontal) error = %v", err)
			}
			vertical, err := NewSeparator(app.Root(), SeparatorOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "vertical",
					Bounds:        Rect{X: 5, Y: 1, Width: 1, Height: 3},
				},
				Orientation: Vertical,
				Form:        test.form,
			})
			if err != nil {
				t.Fatalf("NewSeparator(vertical) error = %v", err)
			}
			if _, ok := any(horizontal).(Container); ok {
				t.Fatal("Separator implements Container")
			}
			snapshot := app.Snapshot()
			if got := cellAt(t, snapshot, 1, 1).Grapheme; got != test.want {
				t.Fatalf("horizontal glyph = %q, want %q", got, test.want)
			}
			wantVertical := test.want
			if test.form == BorderSingle {
				wantVertical = "│"
			} else if test.form == BorderDouble {
				wantVertical = "║"
			}
			if got := cellAt(t, snapshot, 5, 1).Grapheme; got != wantVertical {
				t.Fatalf("vertical glyph = %q, want %q", got, wantVertical)
			}
			if got := horizontal.MinimumSize(); got != (Size{Width: 1, Height: 1}) {
				t.Fatalf("horizontal minimum = %+v", got)
			}
			if got := vertical.MinimumSize(); got != (Size{Width: 1, Height: 1}) {
				t.Fatalf("vertical minimum = %+v", got)
			}
		})
	}

	app := mustApp(t, Size{Width: 10, Height: 10})
	horizontal, err := NewRule(app.Root(), RuleOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "horizontal-rule",
			Bounds:        Rect{X: 1, Y: 1, Width: 7, Height: 1},
		},
		Orientation: Horizontal,
		Form:        BorderDouble,
		Text:        "X",
		Alignment:   TextAlignCenter,
	})
	if err != nil {
		t.Fatalf("NewRule(horizontal) error = %v", err)
	}
	vertical, err := NewRule(app.Root(), RuleOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "vertical-rule",
			Bounds:        Rect{X: 1, Y: 3, Width: 1, Height: 7},
		},
		Orientation: Vertical,
		Form:        BorderSingle,
		Text:        "Y",
		Alignment:   TextAlignEnd,
	})
	if err != nil {
		t.Fatalf("NewRule(vertical) error = %v", err)
	}
	if _, ok := any(horizontal).(Container); ok {
		t.Fatal("Rule implements Container")
	}
	snapshot := app.Snapshot()
	if got := cellAt(t, snapshot, 1, 1).Grapheme; got != "═" {
		t.Fatalf("horizontal rule prefix = %q, want double line", got)
	}
	if got := cellAt(t, snapshot, 4, 1).Grapheme; got != "X" {
		t.Fatalf("horizontal rule title = %q, want X", got)
	}
	if got := cellAt(t, snapshot, 1, 8).Grapheme; got != "Y" {
		t.Fatalf("vertical rule title = %q, want Y", got)
	}
	if got := horizontal.MinimumSize(); got != (Size{Width: 3, Height: 1}) {
		t.Fatalf("horizontal Rule minimum = %+v, want 3x1", got)
	}
	if got := vertical.MinimumSize(); got != (Size{Width: 1, Height: 3}) {
		t.Fatalf("vertical Rule minimum = %+v, want 1x3", got)
	}
}

func TestDisplayTextMutationMinimaTransactionsAndValidation(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 20, Height: 6})
	label, err := NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "label",
			Bounds:        Rect{Width: 8, Height: 1},
		},
		Text: "abc",
	})
	if err != nil {
		t.Fatalf("NewLabel() error = %v", err)
	}
	if err := label.SetText("abcdef"); err != nil {
		t.Fatalf("Label.SetText() error = %v", err)
	}
	if got := label.MinimumSize(); got != (Size{Width: 6, Height: 1}) {
		t.Fatalf("automatic minimum after SetText = %+v", got)
	}
	sequence := app.Snapshot().Sequence
	if err := label.SetText("abcdef"); err != nil {
		t.Fatalf("no-op Label.SetText() error = %v", err)
	}
	if got := app.Snapshot().Sequence; got != sequence {
		t.Fatalf("no-op SetText published sequence %d, want %d", got, sequence)
	}

	tx := app.NewTransaction()
	if err := tx.SetMinimumSize(label, Size{Width: 2, Height: 2}); err != nil {
		t.Fatalf("Transaction.SetMinimumSize() error = %v", err)
	}
	if err := tx.SetText(label, "expanded text"); err != nil {
		t.Fatalf("Transaction.SetText() error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Transaction.Commit() error = %v", err)
	}
	if got := label.MinimumSize(); got != (Size{Width: 2, Height: 2}) {
		t.Fatalf("explicit minimum after SetText = %+v", got)
	}
	if got := label.Text(); got != "expanded text" {
		t.Fatalf("Label.Text() = %q", got)
	}

	before := app.Snapshot()
	if err := label.SetText("invalid\nlabel"); !errors.Is(err, ErrTextLimit) {
		t.Fatalf("newline Label.SetText() error = %v, want ErrTextLimit", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence || label.Text() != "expanded text" {
		t.Fatal("failed text validation changed application state")
	}
	if _, err := NewStaticText(app.Root(), StaticTextOptions{
		Text: strings.Repeat("x", MaxDisplayTextBytes+1),
	}); !errors.Is(err, ErrTextLimit) {
		t.Fatalf("oversized StaticText error = %v, want ErrTextLimit", err)
	}
	if _, err := NewLabel(app.Root(), LabelOptions{
		Text:     "label",
		Mnemonic: "x",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("targetless mnemonic error = %v, want ErrInvalidControl", err)
	}
	if _, err := NewSeparator(app.Root(), SeparatorOptions{
		Orientation: Orientation(99),
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("invalid Separator orientation error = %v", err)
	}
	if err := app.NewTransaction().SetText(label, "unused"); err != nil {
		t.Fatalf("fresh Transaction.SetText() error = %v", err)
	}
}

func TestLabelTargetMustBelongToCommittedTransactionState(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 8, Height: 4})
	targetTransaction := app.NewTransaction()
	target, err := targetTransaction.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "provisional-target",
	})
	if err != nil {
		t.Fatalf("target Transaction.NewPanel() error = %v", err)
	}
	labelTransaction := app.NewTransaction()
	label, err := labelTransaction.NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{AutomationKey: "invalid-label"},
		Text:         "target",
		Target:       target,
	})
	if err != nil {
		t.Fatalf("label Transaction.NewLabel() build error = %v", err)
	}
	if err := labelTransaction.Commit(context.Background()); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("cross-transaction provisional target commit = %v", err)
	}
	if label.ID() != "" {
		t.Fatalf("aborted Label ID = %q, want empty", label.ID())
	}
	if err := targetTransaction.Commit(context.Background()); err != nil {
		t.Fatalf("target Transaction.Commit() error = %v", err)
	}

	transaction := app.NewTransaction()
	sameTarget, err := transaction.NewPanel(app.Root(), PanelOptions{
		AutomationKey: "same-target",
	})
	if err != nil {
		t.Fatalf("same Transaction.NewPanel() error = %v", err)
	}
	sameLabel, err := transaction.NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{AutomationKey: "same-label"},
		Text:         "target",
		Target:       sameTarget,
		Mnemonic:     "t",
	})
	if err != nil {
		t.Fatalf("same Transaction.NewLabel() error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("same Transaction.Commit() error = %v", err)
	}
	if got := controlByKey(t, app.Snapshot(), "same-label").Details.Text.Target; got != sameTarget.ID() {
		t.Fatalf("same-transaction Label target = %q, want %q", got, sameTarget.ID())
	}
	if sameLabel.Parent() != app.Root() {
		t.Fatal("same-transaction Label has the wrong parent")
	}
}

func TestDisplayControlConcurrentReadsAndMutations(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 20, Height: 4})
	label, err := NewLabel(app.Root(), LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "concurrent-label",
			Bounds:        Rect{Width: 20, Height: 1},
		},
		Text: "initial",
	})
	if err != nil {
		t.Fatalf("NewLabel() error = %v", err)
	}

	const workers = 8
	const iterations = 24
	failures := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := range workers {
		worker := worker
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := range iterations {
				text := fmt.Sprintf("worker-%d-%d", worker, iteration)
				if err := label.SetText(text); err != nil {
					failures <- err
					return
				}
				_ = label.Text()
				_ = label.MinimumSize()
				_ = app.Snapshot()
			}
		}()
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent Label operation error = %v", err)
	}
	if label.Text() == "" {
		t.Fatal("concurrent Label mutations left empty text")
	}
}

func FuzzNormalizeDisplayText(f *testing.F) {
	for _, seed := range []string{
		"",
		"ASCII",
		"e\u0301",
		"界",
		"first\r\nsecond",
		"\U0001F469\u200D\U0001F4BB",
	} {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, text string, multiline bool) {
		normalized, err := normalizeDisplayText(text, multiline)
		if err != nil {
			return
		}
		repeated, err := normalizeDisplayText(normalized.text, multiline)
		if err != nil {
			t.Fatalf("normalized text was rejected: %v", err)
		}
		if repeated.text != normalized.text ||
			repeated.cells != normalized.cells {
			t.Fatalf(
				"normalization is not idempotent: first=%q/%d second=%q/%d",
				normalized.text,
				normalized.cells,
				repeated.text,
				repeated.cells,
			)
		}
	})
}
