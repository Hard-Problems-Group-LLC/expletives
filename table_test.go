package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func sampleTableColumns() []Column {
	return []Column{
		{Key: "name", Header: "Name", MinimumWidth: 6, Grow: 2, Sortable: true},
		{Key: "state", Header: "State", Width: 8, Sortable: true},
		{Key: "count", Header: "Count", Width: 5, Alignment: TextAlignEnd, Sortable: true},
	}
}

func sampleTableRows() []TableRow {
	return []TableRow{
		{Key: "bravo", Cells: []TableCell{{Column: "name", Text: "Bravo"}, {Column: "state", Text: "Ready"}, {Column: "count", Text: "2"}}},
		{Key: "disabled", Disabled: true, DisabledReason: "Unavailable", Cells: []TableCell{{Column: "name", Text: "Disabled"}, {Column: "state", Text: "Offline"}}},
		{Key: "alpha", Cells: []TableCell{{Column: "name", Text: "Alpha"}, {Column: "state", Text: "Ready"}, {Column: "count", Text: "10"}}},
	}
}

func sameTableColumnPresentation(
	left []TableColumnPresentation,
	right []TableColumnPresentation,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func tableWrappedLineText(line tableWrappedLine) string {
	return strings.Repeat(" ", line.indent) + strings.Join(line.cells, "")
}

func TestTableColumnWrapAlgorithms(t *testing.T) {
	t.Parallel()
	cells := []string{"a", "l", "p", "h", "a", " ", "b", "e", "t", "a"}
	if lines := wrapTableCellLines(cells, 5, TableColumnClip); len(lines) != 1 ||
		tableWrappedLineText(lines[0]) != "alpha beta" {
		t.Fatalf("Clip lines = %#v", lines)
	}
	if lines := wrapTableCellLines(cells, 6, TableColumnWrapWords); len(lines) != 2 ||
		tableWrappedLineText(lines[0]) != "alpha" ||
		tableWrappedLineText(lines[1]) != "beta" {
		t.Fatalf("Wrap lines = %#v", lines)
	}
	if lines := wrapTableCellLines(cells, 6, TableColumnHang); len(lines) != 2 ||
		tableWrappedLineText(lines[0]) != "alpha" ||
		tableWrappedLineText(lines[1]) != " beta" {
		t.Fatalf("Hang lines = %#v", lines)
	}
	if lines := wrapTableCellLines(
		[]string{"a", "b", "c", "d", "e", "f"},
		4,
		TableColumnWrapWords,
	); len(lines) != 2 || tableWrappedLineText(lines[0]) != "abcd" ||
		tableWrappedLineText(lines[1]) != "ef" {
		t.Fatalf("long-word lines = %#v", lines)
	}
	if lines := wrapTableCellLines(
		[]string{"a", "b", "c"},
		1,
		TableColumnHang,
	); len(lines) != 3 || lines[1].indent != 0 ||
		tableWrappedLineText(lines[2]) != "c" {
		t.Fatalf("width-one Hang lines = %#v", lines)
	}
	if got := strings.Join(alignedWrappedTableCells(
		tableWrappedLine{cells: []string{"x", "y"}, indent: 1},
		6,
		TextAlignEnd,
	), ""); got != "    xy" {
		t.Fatalf("aligned Hang continuation = %q", got)
	}
	normalized, err := normalizeDisplayText("e\u0301界", false)
	if err != nil {
		t.Fatal(err)
	}
	lines := wrapTableCellLines(normalized.lines[0], 1, TableColumnWrapWords)
	if len(lines) != 2 || len(lines[0].cells) != 1 || len(lines[1].cells) != 1 ||
		lines[1].cells[0] != "�" {
		t.Fatalf("one-cell Unicode wrap = %#v", lines)
	}
}

func TestTableFeatureValidationCopyBandGeometryAndRepair(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 34, Height: 10})
	tooMany := make([]TableFeature, MaxTableFeatures+1)
	for index := range tooMany {
		tooMany[index] = TableFeatureColumns
	}
	if _, err := NewTable(app.Root(), TableOptions{Features: tooMany}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("feature capacity error = %v", err)
	}
	for name, features := range map[string][]TableFeature{
		"unknown":   {"filters"},
		"duplicate": {TableFeatureColumns, TableFeatureColumns},
	} {
		if _, err := NewTable(app.Root(), TableOptions{Features: features}); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s feature error = %v", name, err)
		}
	}
	features := []TableFeature{TableFeatureColumns}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "table.features", Bounds: Rect{X: 1, Y: 1, Width: 22, Height: 8}},
		}, BorderForm: BorderSingle},
		Features: features,
		Columns:  []Column{{Key: "value", Header: "Value"}},
		Rows:     []TableRow{{Key: "one"}},
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	features[0] = "caller-mutation"
	state := table.State()
	if len(state.Features) != 1 || state.Features[0] != TableFeatureColumns ||
		state.FocusPart != TableFocusBody {
		t.Fatalf("feature state = %#v", state)
	}
	state.Features[0] = "getter-mutation"
	if table.State().Features[0] != TableFeatureColumns {
		t.Fatal("Table State exposed retained feature storage")
	}
	snapshot := app.Snapshot()
	details := controlByKey(t, snapshot, "table.features").Details.Table
	if details == nil || len(details.Features) != 1 ||
		details.Features[0] != TableFeatureColumns ||
		details.FocusPart != TableFocusBody ||
		details.ColumnsActionBounds != (Rect{X: 8, Y: 6, Width: 14, Height: 2}) ||
		!details.ColumnsActionVisible || !details.ColumnsActionEnabled ||
		details.Viewport.ViewportBounds.Y+details.Viewport.ViewportBounds.Height > 7 {
		t.Fatalf("feature details = %#v", details)
	}
	details.Features[0] = "snapshot-mutation"
	if table.State().Features[0] != TableFeatureColumns {
		t.Fatal("snapshot exposed retained feature storage")
	}
	if cell := cellAt(t, snapshot, 1, 6); cell.Grapheme != "└" || cell.Owner != table.ID() {
		t.Fatalf("body bottom border = %#v", cell)
	}
	for y := 7; y <= 8; y++ {
		if cell := cellAt(t, snapshot, 1, y); cell.Grapheme != " " || cell.Owner != table.ID() {
			t.Fatalf("band cell at row %d = %#v", y, cell)
		}
	}
	if table.MinimumSize().Height < 4 {
		t.Fatalf("feature minimum = %+v", table.MinimumSize())
	}

	for name, fixture := range map[string]struct {
		size    Size
		rows    int
		body    int
		visible bool
	}{
		"normal": {Size{Width: 22, Height: 8}, 2, 6, true},
		"one":    {Size{Width: 22, Height: 1}, 1, 0, true},
		"zero":   {Size{Width: 22}, 0, 0, false},
		"narrow": {Size{Width: 3, Height: 2}, 2, 0, true},
	} {
		t.Run(name, func(t *testing.T) {
			behavior := tableBehavior{features: []TableFeature{TableFeatureColumns}}
			if got := tableColumnsBandRows(behavior, fixture.size); got != fixture.rows {
				t.Fatalf("band rows = %d, want %d", got, fixture.rows)
			}
			if got := tableBodySize(behavior, fixture.size).Height; got != fixture.body {
				t.Fatalf("body height = %d, want %d", got, fixture.body)
			}
			if got := tableColumnsActionVisible(
				behavior,
				Rect{Width: fixture.size.Width, Height: fixture.size.Height},
			); got != fixture.visible {
				t.Fatalf("action visible = %v, want %v", got, fixture.visible)
			}
		})
	}

	app.mu.Lock()
	behavior := table.state.behavior.(tableBehavior)
	behavior.focusPart = TableFocusColumnsAction
	table.state.behavior = behavior
	app.mu.Unlock()
	if err := table.SetFeatures(nil); err != nil {
		t.Fatalf("SetFeatures(nil) error = %v", err)
	}
	if state = table.State(); len(state.Features) != 0 || state.FocusPart != TableFocusBody {
		t.Fatalf("feature removal repair = %#v", state)
	}
}

func TestTableColumnsActionRenderingTraversalMnemonicAndPressCapture(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 100, Height: 40}})
	if err != nil {
		t.Fatal(err)
	}
	firstParent := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "columns.group", Bounds: Rect{Width: 24, Height: 9},
	})
	secondParent := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "next.group", Bounds: Rect{X: 28, Width: 20, Height: 9},
	})
	first, err := NewTable(firstParent, TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "columns.table", Bounds: Rect{Width: 22, Height: 8},
			}},
			BorderForm: BorderSingle,
		},
		Features: []TableFeature{TableFeatureColumns},
		Columns:  []Column{{Key: "value", Header: "Value"}},
		Rows:     []TableRow{{Key: "one"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTable(secondParent, TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "next.table", Bounds: Rect{Width: 18, Height: 6}},
		}},
		Columns: []Column{{Key: "value", Header: "Value"}},
		Rows:    []TableRow{{Key: "next"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Focus(); err != nil {
		t.Fatal(err)
	}
	if state := first.State(); state.FocusPart != TableFocusBody {
		t.Fatalf("initial focus part = %q", state.FocusPart)
	}
	dispatchTableKey(t, app, "tab-action", KeyTab)
	if app.Focused() != first || first.State().FocusPart != TableFocusColumnsAction {
		t.Fatalf("Tab focus = %#v / %q", app.Focused(), first.State().FocusPart)
	}
	snapshot := app.Snapshot()
	if cell := cellAt(t, snapshot, 10, 6); cell.Grapheme != "C" ||
		cell.Style != "button.mnemonic" || cell.Owner != first.ID() {
		t.Fatalf("focused Columns label = %#v", cell)
	}
	if cell := cellAt(t, snapshot, 11, 6); cell.Grapheme != "o" ||
		cell.Style != "button.focused" {
		t.Fatalf("focused Columns body = %#v", cell)
	}

	down, err := app.DispatchKey(
		context.Background(), "columns-press", "columns-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyEnter},
	)
	if err != nil || down.Outcome != OutcomeApplied {
		t.Fatalf("Enter down = %#v, %v", down, err)
	}
	pressed := controlByKey(t, app.Snapshot(), "columns.table").Details.Table
	if pressed == nil || !pressed.ColumnsActionPressed ||
		cellAt(t, app.Snapshot(), 12, 6).Style != "button.pressed" {
		t.Fatalf("pressed action = %#v", pressed)
	}
	up, err := app.DispatchKey(
		context.Background(), "columns-press", "columns-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyEnter},
	)
	if err != nil || up.Outcome != OutcomeApplied {
		t.Fatalf("Enter up = %#v, cause=%v, err=%v", up, up.Cause, err)
	}
	if details := controlByKey(t, app.Snapshot(), "columns.table").Details.Table; details == nil || details.ColumnsActionPressed {
		t.Fatalf("released action = %#v", details)
	}
	if dialog := controlByKey(t, app.Snapshot(), "columns.table.columns-dialog"); dialog.Kind != ControlDialog || !dialog.Details.ModalPanel.Active {
		t.Fatalf("opened Columns dialog = %#v", dialog)
	}
	if owner := controlByKey(t, app.Snapshot(), "columns.table").Details.Table; owner == nil || !owner.ColumnsDialogOpen || owner.ColumnsActionEnabled ||
		owner.ColumnsActionPressed {
		t.Fatalf("open owner details = %#v", owner)
	}
	if overflows := app.Snapshot().Overflows; len(overflows) != 0 {
		t.Fatalf("Columns dialog overflow = %#v", overflows)
	}
	cancelCompletion := dispatchTableKey(t, app, "cancel-enter-dialog", KeyEscape)
	if app.Focused() != first || first.State().FocusPart != TableFocusColumnsAction {
		t.Fatalf("cancel restore = %#v / %q, completion=%#v cause=%v", app.Focused(), first.State().FocusPart, cancelCompletion, cancelCompletion.Cause)
	}
	if completion := dispatchTableKey(t, app, "activate-space", KeySpace); completion.Outcome != OutcomeApplied {
		t.Fatalf("Space action completion = %#v", completion)
	}
	dispatchTableKey(t, app, "cancel-space-dialog", KeyEscape)

	dispatchTableKey(t, app, "tab-next", KeyTab)
	if app.Focused() != second || second.State().FocusPart != TableFocusBody {
		t.Fatalf("forward exit = %#v / %q", app.Focused(), second.State().FocusPart)
	}
	dispatchTextChord(t, app, "reverse-entry", KeyShift, KeyTab)
	if app.Focused() != first || first.State().FocusPart != TableFocusColumnsAction {
		t.Fatalf("reverse entry = %#v / %q", app.Focused(), first.State().FocusPart)
	}
	dispatchTextChord(t, app, "reverse-body", KeyShift, KeyTab)
	if app.Focused() != first || first.State().FocusPart != TableFocusBody {
		t.Fatalf("reverse internal = %#v / %q", app.Focused(), first.State().FocusPart)
	}
	if err := second.Focus(); err != nil {
		t.Fatal(err)
	}
	dispatchTextChord(t, app, "columns-mnemonic", KeyAlt, Key("c"))
	if first.State().FocusPart != TableFocusColumnsAction ||
		!controlByKey(t, app.Snapshot(), "columns.table.columns-dialog").Details.ModalPanel.Active {
		t.Fatalf("Alt-C action = %#v / %q", app.Focused(), first.State().FocusPart)
	}
	dispatchTableKey(t, app, "cancel-mnemonic-dialog", KeyEscape)
}

func invokeColumnsCommand(
	t *testing.T,
	app *App,
	request string,
	command CommandID,
	target Control,
) Completion {
	t.Helper()
	completion, err := app.InvokeCommand(
		context.Background(), "columns-test", request, command, target.ID(),
	)
	if err != nil {
		t.Fatalf("InvokeCommand(%s) error = %v", command, err)
	}
	return completion
}

func TestTableColumnsDialogDraftSearchEditResetCancelAndApply(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 100, Height: 40}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterCommand(CommandDefinition{
		ID: "table.changed", Label: "Changed", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	routed := 0
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		if command.ID == "table.changed" {
			routed++
		}
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatal(err)
	}
	initial := []TableColumnPresentation{
		{Column: "a", Visible: true, Wrap: TableColumnClip},
		{Column: "b", Visible: true, Wrap: TableColumnWrapWords},
		{Column: "c", Visible: true, Wrap: TableColumnHang},
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions:  PanelOptions{AutomationKey: "editor.table", Bounds: Rect{Width: 36, Height: 10}},
			ChangeCommand: "table.changed",
		}},
		Features: []TableFeature{TableFeatureColumns},
		Columns: []Column{
			{Key: "a", Header: "Alpha"},
			{Key: "b", Header: "Beta"},
			{Key: "c", Header: "Gamma"},
		},
		ColumnPresentation: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	open := func(request string) *tableColumnsDialogCore {
		if err := table.Focus(); err != nil {
			t.Fatal(err)
		}
		dispatchTableKey(t, app, request+"-focus-action", KeyTab)
		if completion := dispatchTableKey(t, app, request+"-open", KeySpace); completion.Outcome != OutcomeApplied {
			t.Fatalf("open completion = %#v", completion)
		}
		app.mu.RLock()
		core := app.tableColumnsDialogs[table.state]
		app.mu.RUnlock()
		if core == nil || !table.State().ColumnsDialogOpen {
			t.Fatalf("Columns dialog core = %#v", core)
		}
		return core
	}

	core := open("draft")
	if err := core.list.SetCurrent("b"); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "select-b", CommandTableColumnsCurrent, core.list)
	if err := core.visible.SetState(CheckUnchecked); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "hide-b", CommandTableColumnsVisible, core.visible)
	if err := core.wrap.SetValue(string(TableColumnHang)); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "hang-b", CommandTableColumnsWrap, core.wrap)
	invokeColumnsCommand(t, app, "move-b-left", CommandTableColumnsMoveLeft, core.moveLeft)
	if got := table.ColumnPresentation(); !sameTableColumnPresentation(got, initial) {
		t.Fatalf("private draft changed Table = %#v", got)
	}
	if len(core.draft) != 3 || core.draft[0].Column != "b" ||
		core.draft[0].Visible || core.draft[0].Wrap != TableColumnHang {
		t.Fatalf("edited draft = %#v", core.draft)
	}
	if err := core.search.SetText("gamma"); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "search-gamma", CommandTableColumnsSearch, core.search)
	if items := core.list.Items(); len(items) != 1 || items[0].Key != "c" {
		t.Fatalf("filtered items = %#v", items)
	}
	invokeColumnsCommand(t, app, "reset", CommandTableColumnsReset, core.reset)
	if !sameTableColumnPresentation(core.draft, initial) {
		t.Fatalf("reset draft = %#v", core.draft)
	}
	invokeColumnsCommand(t, app, "cancel", CommandTableColumnsCancel, core.cancel)
	if table.State().ColumnsDialogOpen || !sameTableColumnPresentation(table.ColumnPresentation(), initial) || routed != 0 {
		t.Fatalf("cancel state = %#v presentation=%#v routed=%d", table.State(), table.ColumnPresentation(), routed)
	}

	core = open("apply")
	if err := core.list.SetCurrent("a"); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "select-a", CommandTableColumnsCurrent, core.list)
	if err := core.visible.SetState(CheckUnchecked); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "hide-a", CommandTableColumnsVisible, core.visible)
	invokeColumnsCommand(t, app, "apply", CommandTableColumnsOK, core.ok)
	want := append([]TableColumnPresentation(nil), initial...)
	want[0].Visible = false
	if table.State().ColumnsDialogOpen || !sameTableColumnPresentation(table.ColumnPresentation(), want) || routed != 1 {
		t.Fatalf("apply state=%#v presentation=%#v routed=%d", table.State(), table.ColumnPresentation(), routed)
	}
}

func TestTableColumnsDialogRejectsHidingFinalVisibleColumn(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 80, Height: 30}})
	if err != nil {
		t.Fatal(err)
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "last.table", Bounds: Rect{Width: 30, Height: 8}},
		}},
		Features: []TableFeature{TableFeatureColumns},
		Columns:  []Column{{Key: "only", Header: "Only"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := table.Focus(); err != nil {
		t.Fatal(err)
	}
	dispatchTableKey(t, app, "last-action", KeyTab)
	dispatchTableKey(t, app, "last-open", KeySpace)
	app.mu.RLock()
	core := app.tableColumnsDialogs[table.state]
	app.mu.RUnlock()
	if core == nil {
		t.Fatal("Columns dialog did not open")
	}
	if err := core.visible.SetState(CheckUnchecked); err != nil {
		t.Fatal(err)
	}
	completion := invokeColumnsCommand(
		t, app, "hide-last", CommandTableColumnsVisible, core.visible,
	)
	if completion.Outcome != OutcomeApplied || !core.draft[0].Visible ||
		core.visible.State() != CheckChecked ||
		!strings.Contains(controlByKey(t, app.Snapshot(), "last.table.columns-dialog.information").Details.Text.Text, "At least one") {
		t.Fatalf("last-visible rejection completion=%#v draft=%#v checkbox=%q", completion, core.draft, core.visible.State())
	}
	invokeColumnsCommand(t, app, "last-cancel", CommandTableColumnsCancel, core.cancel)
}

func TestTableColumnsDialogRevisionRebaseConflictReloadAndShortcuts(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 90, Height: 35}})
	if err != nil {
		t.Fatal(err)
	}
	initial := []TableColumnPresentation{
		{Column: "a", Visible: true, Wrap: TableColumnClip},
		{Column: "b", Visible: true, Wrap: TableColumnClip},
		{Column: "c", Visible: true, Wrap: TableColumnClip},
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "revisions.table",
				Bounds:        Rect{Width: 36, Height: 10},
			},
		}},
		Features: []TableFeature{TableFeatureColumns},
		Columns: []Column{
			{Key: "a", Header: "Alpha"},
			{Key: "b", Header: "Beta"},
			{Key: "c", Header: "Gamma"},
		},
		ColumnPresentation: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	open := func(request string) *tableColumnsDialogCore {
		t.Helper()
		if err := table.Focus(); err != nil {
			t.Fatal(err)
		}
		if table.State().FocusPart == TableFocusBody {
			dispatchTableKey(t, app, request+"-action", KeyTab)
		}
		if completion := dispatchTableKey(t, app, request+"-open", KeySpace); completion.Outcome != OutcomeApplied {
			t.Fatalf("open completion = %#v", completion)
		}
		app.mu.RLock()
		core := app.tableColumnsDialogs[table.state]
		app.mu.RUnlock()
		if core == nil {
			t.Fatal("Columns dialog did not open")
		}
		return core
	}

	core := open("rebase")
	if err := core.list.SetCurrent("b"); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "rebase-select-b", CommandTableColumnsCurrent, core.list)
	if err := core.visible.SetState(CheckUnchecked); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "rebase-hide-b", CommandTableColumnsVisible, core.visible)
	if err := core.wrap.SetValue(string(TableColumnHang)); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "rebase-hang-b", CommandTableColumnsWrap, core.wrap)
	invokeColumnsCommand(t, app, "rebase-move-b", CommandTableColumnsMoveLeft, core.moveLeft)
	beforeRowsRevision := core.schemaRevision
	if err := table.SetRows([]TableRow{{Key: "row"}}); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "rebase-row-only", CommandTableColumnsCurrent, core.list)
	if core.schemaRevision != beforeRowsRevision || core.stale.Load() {
		t.Fatalf("row-only revision state = schema %d stale %t", core.schemaRevision, core.stale.Load())
	}
	if err := table.SetModel([]Column{
		{Key: "a", Header: "Alpha"},
		{Key: "b", Header: "Beta"},
		{Key: "d", Header: "Delta"},
	}, nil); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "rebase-schema", CommandTableColumnsCurrent, core.list)
	wantRebased := []TableColumnPresentation{
		{Column: "b", Visible: false, Wrap: TableColumnHang},
		{Column: "a", Visible: true, Wrap: TableColumnClip},
		{Column: "d", Visible: true, Wrap: TableColumnClip},
	}
	if core.stale.Load() || core.reload.Visible() ||
		!sameTableColumnPresentation(core.draft, wantRebased) {
		t.Fatalf("schema rebase = %#v stale=%t reload=%t", core.draft, core.stale.Load(), core.reload.Visible())
	}
	if completion := invokeColumnsCommand(
		t, app, "rebase-apply", CommandTableColumnsOK, core.ok,
	); completion.Outcome != OutcomeApplied ||
		!sameTableColumnPresentation(table.ColumnPresentation(), wantRebased) {
		t.Fatalf("rebased apply = %#v presentation=%#v", completion, table.ColumnPresentation())
	}

	core = open("conflict")
	external := []TableColumnPresentation{
		{Column: "d", Visible: true, Wrap: TableColumnWrapWords},
		{Column: "a", Visible: true, Wrap: TableColumnClip},
		{Column: "b", Visible: true, Wrap: TableColumnHang},
	}
	if err := table.SetColumnPresentation(external); err != nil {
		t.Fatal(err)
	}
	okDetails := controlByKey(
		t, app.Snapshot(), "revisions.table.columns-dialog.ok",
	).Details.Action
	if !core.stale.Load() || !core.reload.Visible() || okDetails == nil ||
		okDetails.Enabled || !strings.Contains(okDetails.DisabledReason, "Reload") {
		t.Fatalf("stale UI = stale %t reload %t OK %#v", core.stale.Load(), core.reload.Visible(), okDetails)
	}
	if completion := invokeColumnsCommand(
		t, app, "conflict-ok", CommandTableColumnsOK, core.ok,
	); completion.Outcome != OutcomeRejected || completion.Code != "columns_dialog_stale" ||
		!sameTableColumnPresentation(table.ColumnPresentation(), external) {
		t.Fatalf("stale OK = %#v presentation=%#v", completion, table.ColumnPresentation())
	}
	if completion := invokeColumnsCommand(
		t, app, "conflict-reload", CommandTableColumnsReload, core.reload,
	); completion.Outcome != OutcomeApplied || core.stale.Load() || core.reload.Visible() ||
		!sameTableColumnPresentation(core.draft, external) {
		t.Fatalf("Reload = %#v draft=%#v stale=%t reload=%t", completion, core.draft, core.stale.Load(), core.reload.Visible())
	}
	if err := core.list.SetCurrent("d"); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "conflict-select-d", CommandTableColumnsCurrent, core.list)
	if completion := dispatchTableKey(t, app, "conflict-space-toggle", KeySpace); completion.Outcome != OutcomeApplied || core.draft[0].Visible {
		t.Fatalf("inventory Space = %#v draft=%#v", completion, core.draft)
	}
	if completion := dispatchTextChord(t, app, "conflict-move-right", KeyControl, KeyRight); completion.Outcome != OutcomeApplied || core.draft[1].Column != "d" {
		t.Fatalf("Ctrl-Right = %#v draft=%#v", completion, core.draft)
	}
	wantReloaded := []TableColumnPresentation{
		{Column: "a", Visible: true, Wrap: TableColumnClip},
		{Column: "d", Visible: false, Wrap: TableColumnWrapWords},
		{Column: "b", Visible: true, Wrap: TableColumnHang},
	}
	if completion := invokeColumnsCommand(
		t, app, "conflict-apply", CommandTableColumnsOK, core.ok,
	); completion.Outcome != OutcomeApplied ||
		!sameTableColumnPresentation(table.ColumnPresentation(), wantReloaded) {
		t.Fatalf("reloaded apply = %#v presentation=%#v", completion, table.ColumnPresentation())
	}

	core = open("transaction-conflict")
	first := app.NewTransaction()
	second := app.NewTransaction()
	firstPresentation := append([]TableColumnPresentation(nil), wantReloaded...)
	firstPresentation[0].Wrap = TableColumnHang
	secondPresentation := append([]TableColumnPresentation(nil), wantReloaded...)
	secondPresentation[0].Wrap = TableColumnWrapWords
	if err := first.SetTableColumnPresentation(table, firstPresentation); err != nil {
		t.Fatal(err)
	}
	if err := second.SetTableColumnPresentation(table, secondPresentation); err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	invokeColumnsCommand(t, app, "transaction-reload", CommandTableColumnsReload, core.reload)
	if err := second.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !core.stale.Load() || !core.reload.Visible() {
		t.Fatalf("stale transaction was not detected: stale=%t reload=%t", core.stale.Load(), core.reload.Visible())
	}
	invokeColumnsCommand(t, app, "transaction-cancel", CommandTableColumnsCancel, core.cancel)
}

func TestTableColumnsDialogOwnerLifecycleInvalidation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		change func(*testing.T, *Table)
	}{
		{name: "hide", change: func(t *testing.T, table *Table) {
			t.Helper()
			if err := table.SetVisible(false); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "remove feature", change: func(t *testing.T, table *Table) {
			t.Helper()
			if err := table.SetFeatures(nil); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "destroy", change: func(t *testing.T, table *Table) {
			t.Helper()
			if err := table.Destroy(); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app, err := NewApp(AppOptions{Size: Size{Width: 70, Height: 25}})
			if err != nil {
				t.Fatal(err)
			}
			table, err := NewTable(app.Root(), TableOptions{
				ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
					PanelOptions: PanelOptions{
						AutomationKey: "lifecycle.table",
						Bounds:        Rect{Width: 30, Height: 8},
					},
				}},
				Features: []TableFeature{TableFeatureColumns},
				Columns:  []Column{{Key: "value", Header: "Value"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := table.Focus(); err != nil {
				t.Fatal(err)
			}
			dispatchTableKey(t, app, "lifecycle-action", KeyTab)
			dispatchTableKey(t, app, "lifecycle-open", KeySpace)
			app.mu.RLock()
			core := app.tableColumnsDialogs[table.state]
			app.mu.RUnlock()
			if core == nil {
				t.Fatal("Columns dialog did not open")
			}
			test.change(t, table)
			app.mu.RLock()
			retained := app.tableColumnsDialogs[table.state]
			app.mu.RUnlock()
			if retained != nil || !core.dialog.state.destroyed || core.dialog.Visible() {
				t.Fatalf("invalidated dialog retained=%#v destroyed=%t visible=%t", retained, core.dialog.state.destroyed, core.dialog.Visible())
			}
			if test.name == "remove feature" && table.State().FocusPart != TableFocusBody {
				t.Fatalf("feature-removal focus part = %q", table.State().FocusPart)
			}
		})
	}
}

func TestTableColumnsDialogAppFinalizationDiscardsDraft(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 70, Height: 25}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterCommand(CommandDefinition{
		ID: "app.quit", Label: "Quit", Enabled: true,
		ModalPolicy: CommandModalAllowed,
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.SetCommandRouter(func(context.Context, Command) CommandResult {
		return CommandResult{Outcome: OutcomeExited}
	}); err != nil {
		t.Fatal(err)
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "final.table",
				Bounds:        Rect{Width: 30, Height: 8},
			},
		}},
		Features: []TableFeature{TableFeatureColumns},
		Columns:  []Column{{Key: "value", Header: "Value"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := table.Focus(); err != nil {
		t.Fatal(err)
	}
	dispatchTableKey(t, app, "final-action", KeyTab)
	dispatchTableKey(t, app, "final-open", KeySpace)
	completion, err := app.InvokeCommand(
		context.Background(), "columns-test", "final-quit", "app.quit", "",
	)
	if err != nil || completion.Outcome != OutcomeExited {
		t.Fatalf("quit completion = %#v, %v", completion, err)
	}
	app.mu.RLock()
	retained := app.tableColumnsDialogs[table.state]
	app.mu.RUnlock()
	details := controlByKey(t, app.Snapshot(), "final.table").Details.Table
	if retained != nil || details == nil || details.ColumnsDialogOpen ||
		!app.Snapshot().Final {
		t.Fatalf("final dialog state retained=%#v details=%#v final=%t", retained, details, app.Snapshot().Final)
	}
}

func TestTableColumnsDialogMaximumSchemaUsesFixedControlTree(t *testing.T) {
	t.Parallel()
	app, err := NewApp(AppOptions{Size: Size{Width: 100, Height: 40}})
	if err != nil {
		t.Fatal(err)
	}
	columns := make([]Column, MaxCollectionColumns)
	for index := range columns {
		key := fmt.Sprintf("column-%03d", index)
		columns[index] = Column{Key: key, Header: "Column " + key}
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "maximum.table",
				Bounds:        Rect{Width: 60, Height: 12},
			},
		}},
		Features: []TableFeature{TableFeatureColumns},
		Columns:  columns,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := len(app.controlsByID)
	if err := table.Focus(); err != nil {
		t.Fatal(err)
	}
	dispatchTableKey(t, app, "maximum-action", KeyTab)
	dispatchTableKey(t, app, "maximum-open", KeySpace)
	app.mu.RLock()
	core := app.tableColumnsDialogs[table.state]
	after := len(app.controlsByID)
	app.mu.RUnlock()
	if core == nil {
		t.Fatal("maximum-schema Columns dialog did not open")
	}
	if after-before > 24 || len(core.list.Items()) != MaxCollectionColumns {
		t.Fatalf("fixed tree: controls=%d inventory=%d", after-before, len(core.list.Items()))
	}
	if err := core.search.SetText("column-255"); err != nil {
		t.Fatal(err)
	}
	if completion := invokeColumnsCommand(
		t, app, "maximum-search", CommandTableColumnsSearch, core.search,
	); completion.Outcome != OutcomeApplied {
		t.Fatalf("maximum search completion = %#v", completion)
	}
	items := core.list.Items()
	if len(items) != 1 || items[0].Key != "column-255" {
		t.Fatalf("maximum filtered inventory = %#v", items)
	}
	invokeColumnsCommand(t, app, "maximum-cancel", CommandTableColumnsCancel, core.cancel)
}

func TestTableColumnsActionDisabledAndOneRowDegradation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 3})
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "columns.disabled", Bounds: Rect{Width: 14, Height: 1}},
			Disabled:     true, DisabledReason: "Unavailable",
		}},
		Features: []TableFeature{TableFeatureColumns},
	})
	if err != nil {
		t.Fatal(err)
	}
	details := controlByKey(t, app.Snapshot(), "columns.disabled").Details.Table
	if details == nil || !details.ColumnsActionVisible || details.ColumnsActionEnabled ||
		details.ColumnsActionBounds != (Rect{Width: 14, Height: 1}) {
		t.Fatalf("disabled one-row details = %#v", details)
	}
	if cell := cellAt(t, app.Snapshot(), 2, 0); cell.Grapheme != "C" ||
		cell.Style != "button.disabled" || cell.Owner != table.ID() {
		t.Fatalf("disabled one-row label = %#v", cell)
	}
}

func TestTableColumnsActionEmptyModelAndFullClipEligibility(t *testing.T) {
	t.Parallel()
	t.Run("empty model", func(t *testing.T) {
		app := mustApp(t, Size{Width: 24, Height: 5})
		table, err := NewTable(app.Root(), TableOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{AutomationKey: "columns.empty", Bounds: Rect{Width: 22, Height: 4}},
			}},
			Features: []TableFeature{TableFeatureColumns},
		})
		if err != nil {
			t.Fatal(err)
		}
		if app.Focused() != table || table.State().FocusPart != TableFocusBody {
			t.Fatalf("empty initial focus = %#v / %q", app.Focused(), table.State().FocusPart)
		}
		dispatchTableKey(t, app, "empty-action", KeyTab)
		if app.Focused() != table || table.State().FocusPart != TableFocusColumnsAction {
			t.Fatalf("empty action focus = %#v / %q", app.Focused(), table.State().FocusPart)
		}
	})

	t.Run("fully clipped", func(t *testing.T) {
		app := mustApp(t, Size{Width: 48, Height: 8})
		clippedParent := mustPanel(t, app.Root(), PanelOptions{
			AutomationKey: "clipped.group", Bounds: Rect{Width: 24, Height: 5},
		})
		nextParent := mustPanel(t, app.Root(), PanelOptions{
			AutomationKey: "visible.group", Bounds: Rect{X: 28, Width: 18, Height: 6},
		})
		clipped, err := NewTable(clippedParent, TableOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{AutomationKey: "columns.clipped", Bounds: Rect{Width: 22, Height: 8}},
			}},
			Features: []TableFeature{TableFeatureColumns},
			Columns:  []Column{{Key: "value", Header: "Value"}},
			Rows:     []TableRow{{Key: "one"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		next, err := NewTable(nextParent, TableOptions{
			ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{AutomationKey: "visible.table", Bounds: Rect{Width: 16, Height: 5}},
			}},
			Columns: []Column{{Key: "value", Header: "Value"}},
			Rows:    []TableRow{{Key: "next"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := clipped.Focus(); err != nil {
			t.Fatal(err)
		}
		details := controlByKey(t, app.Snapshot(), "columns.clipped").Details.Table
		if details == nil || details.ColumnsActionVisible || details.ColumnsActionEnabled {
			t.Fatalf("clipped action details = %#v", details)
		}
		dispatchTableKey(t, app, "clipped-tab", KeyTab)
		if app.Focused() != next {
			t.Fatalf("clipped action retained traversal: focus = %#v", app.Focused())
		}
	})
}

func dispatchTableKey(t *testing.T, app *App, request string, key Key) Completion {
	t.Helper()
	completion, err := app.DispatchKey(context.Background(), "table-test", request, KeyEvent{Kind: KeyEventPress, Key: key})
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func dispatchTableControlKey(t *testing.T, app *App, request string, key Key) Completion {
	t.Helper()
	for index, event := range []KeyEvent{
		{Kind: KeyEventDown, Key: KeyControl},
		{Kind: KeyEventPress, Key: key},
		{Kind: KeyEventUp, Key: KeyControl},
	} {
		completion, err := app.DispatchKey(context.Background(), "table-test", fmt.Sprintf("%s-%d", request, index), event)
		if err != nil {
			t.Fatalf("DispatchKey(%s) error = %v", key, err)
		}
		if index == 1 {
			return completion
		}
	}
	return Completion{}
}

func TestTableRenderingDetailsAndCopiedState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 52, Height: 12})
	columns, rows := sampleTableColumns(), sampleTableRows()
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "collection.table",
				Bounds:        Rect{X: 1, Y: 1, Width: 42, Height: 8},
			}},
			BorderForm: BorderSingle,
		},
		Columns: columns, Rows: rows, CurrentRow: "bravo", CurrentColumn: "name",
		Selected: []string{"alpha"}, FocusMode: TableFocusCell,
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	columns[0].Header = "caller mutation"
	rows[0].Cells[0].Text = "caller mutation"
	retainedColumns, retainedRows := table.Columns(), table.Rows()
	if retainedColumns[0].Header != "Name" || retainedRows[0].Cells[0].Text != "Bravo" {
		t.Fatalf("construction retained caller storage: columns=%#v rows=%#v", retainedColumns, retainedRows)
	}
	retainedColumns[0].Header = "getter mutation"
	retainedRows[0].Cells[0].Text = "getter mutation"
	state := table.State()
	state.Selected[0] = "getter mutation"
	state.ColumnWidths[0] = 999
	state.ColumnPresentation[0].Column = "getter mutation"
	if table.Columns()[0].Header != "Name" || table.Rows()[0].Cells[0].Text != "Bravo" ||
		table.State().Selected[0] != "alpha" || table.State().ColumnWidths[0] == 999 ||
		table.State().ColumnPresentation[0].Column != "name" {
		t.Fatal("Table getter exposed retained mutable storage")
	}

	control := controlByKey(t, app.Snapshot(), "collection.table")
	details := control.Details.Table
	if details == nil || details.Status != CollectionReady || details.RowCount != 3 ||
		details.EnabledCount != 2 || details.ColumnCount != 3 || details.CellCount != 8 ||
		details.CurrentRow != "bravo" || details.CurrentRowIndex != 0 ||
		details.CurrentColumn != "name" || details.CurrentColumnIndex != 0 ||
		details.FocusMode != TableFocusCell ||
		details.SelectionStyle != TableSelectionSingle ||
		details.RangeAnchor != "" || details.RangeExtent != "" ||
		details.SelectedCount != 1 ||
		details.FirstSelected != "alpha" || details.LastSelected != "alpha" ||
		details.VisibleColumnCount != 3 || details.FirstVisibleColumn != "name" ||
		details.LastVisibleColumn != "count" || len(details.ColumnPresentation) != 3 ||
		len(details.SelectionDigest) != 64 || len(details.PresentationDigest) != 64 ||
		len(details.ColumnWidthsDigest) != 64 ||
		len(details.ColumnWidths) != 3 || !details.Enabled ||
		details.Viewport.Content != "" || details.Viewport.ContentKey != "" {
		t.Fatalf("TableDetails = %#v", details)
	}
	if control.Details.Container != nil || control.Details.Border == nil {
		t.Fatalf("Table union shape = %#v", control.Details)
	}
	if got := rowText(app.Snapshot(), 2); !strings.Contains(got, "Name") || !strings.Contains(got, "State") {
		t.Fatalf("sticky header row = %q", got)
	}
	if got := rowText(app.Snapshot(), 3); !strings.Contains(got, "►[ ]Bravo") {
		t.Fatalf("first table row = %q", got)
	}
}

func TestTableColumnPresentationValidationCopyMutationAndRepair(t *testing.T) {
	t.Parallel()
	columns, rows := sampleTableColumns(), sampleTableRows()
	invalid := map[string][]TableColumnPresentation{
		"empty": {},
		"missing": {
			{Column: "name", Visible: true},
			{Column: "state", Visible: true},
		},
		"unknown": {
			{Column: "name", Visible: true},
			{Column: "state", Visible: true},
			{Column: "missing", Visible: true},
		},
		"duplicate": {
			{Column: "name", Visible: true},
			{Column: "state", Visible: true},
			{Column: "state", Visible: true},
		},
		"wrap": {
			{Column: "name", Visible: true},
			{Column: "state", Visible: true, Wrap: "fold"},
			{Column: "count", Visible: true},
		},
		"hidden": {
			{Column: "name"},
			{Column: "state"},
			{Column: "count"},
		},
	}
	for name, presentation := range invalid {
		name, presentation := name, presentation
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			app := mustApp(t, Size{Width: 40, Height: 9})
			if _, err := NewTable(app.Root(), TableOptions{
				Columns: columns, Rows: rows, ColumnPresentation: presentation,
			}); !errors.Is(err, ErrValidation) {
				t.Fatalf("NewTable() error = %v", err)
			}
		})
	}
	emptyApp := mustApp(t, Size{Width: 20, Height: 6})
	if _, err := NewTable(emptyApp.Root(), TableOptions{
		ColumnPresentation: []TableColumnPresentation{},
	}); err != nil {
		t.Fatalf("empty-schema presentation error = %v", err)
	}

	app := mustApp(t, Size{Width: 44, Height: 10})
	presentation := []TableColumnPresentation{
		{Column: "state", Visible: true, Wrap: TableColumnHang},
		{Column: "name", Visible: false},
		{Column: "count", Visible: true, Wrap: TableColumnWrapWords},
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "table.presentation", Bounds: Rect{Width: 34, Height: 7}},
		}},
		Columns: columns, Rows: rows, CurrentColumn: "name",
		ColumnPresentation: presentation,
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	presentation[0].Column = "caller-mutation"
	want := []TableColumnPresentation{
		{Column: "state", Visible: true, Wrap: TableColumnHang},
		{Column: "name", Visible: false, Wrap: TableColumnClip},
		{Column: "count", Visible: true, Wrap: TableColumnWrapWords},
	}
	if state := table.State(); state.CurrentColumn != "count" ||
		state.VisibleColumnCount != 2 ||
		!sameTableColumnPresentation(state.ColumnPresentation, want) {
		t.Fatalf("initial presentation state = %#v", state)
	}
	copyValue := table.ColumnPresentation()
	copyValue[0].Column = "getter-mutation"
	if sameTableColumnPresentation(copyValue, table.ColumnPresentation()) {
		t.Fatal("ColumnPresentation() exposed retained slice storage")
	}
	details := controlByKey(t, app.Snapshot(), "table.presentation").Details.Table
	if details == nil || details.VisibleColumnCount != 2 ||
		details.FirstVisibleColumn != "state" || details.LastVisibleColumn != "count" ||
		len(details.PresentationDigest) != 64 ||
		!sameTableColumnPresentation(details.ColumnPresentation, want) {
		t.Fatalf("presentation details = %#v", details)
	}
	details.ColumnPresentation[0].Column = "snapshot-mutation"
	if table.ColumnPresentation()[0].Column != "state" {
		t.Fatal("snapshot exposed retained presentation storage")
	}

	before := table.ColumnPresentation()
	if err := table.SetColumnPresentation([]TableColumnPresentation{
		{Column: "name", Visible: false},
		{Column: "state", Visible: false},
		{Column: "count", Visible: false},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetColumnPresentation(all hidden) error = %v", err)
	}
	if !sameTableColumnPresentation(table.ColumnPresentation(), before) {
		t.Fatal("failed presentation mutation changed state")
	}

	tx := app.NewTransaction()
	staged := []TableColumnPresentation{
		{Column: "count", Visible: true, Wrap: TableColumnHang},
		{Column: "name", Visible: false},
		{Column: "state", Visible: true, Wrap: TableColumnWrapWords},
	}
	if err := tx.SetTableColumnPresentation(table, staged); err != nil {
		t.Fatalf("SetTableColumnPresentation() error = %v", err)
	}
	staged[0].Column = "caller-mutation"
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got := table.ColumnPresentation(); got[0].Column != "count" {
		t.Fatalf("transaction retained caller storage: %#v", got)
	}

	newColumns := []Column{
		{Key: "state", Header: "State"},
		{Key: "fresh", Header: "Fresh"},
		{Key: "count", Header: "Count"},
	}
	if err := table.SetModel(newColumns, []TableRow{{Key: "new-row"}}); err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}
	want = []TableColumnPresentation{
		{Column: "count", Visible: true, Wrap: TableColumnHang},
		{Column: "state", Visible: true, Wrap: TableColumnWrapWords},
		{Column: "fresh", Visible: true, Wrap: TableColumnClip},
	}
	if got := table.ColumnPresentation(); !sameTableColumnPresentation(got, want) {
		t.Fatalf("repaired presentation = %#v, want %#v", got, want)
	}
	if err := table.SetColumnPresentation(nil); err != nil {
		t.Fatalf("SetColumnPresentation(nil) error = %v", err)
	}
	want = []TableColumnPresentation{
		{Column: "state", Visible: true, Wrap: TableColumnClip},
		{Column: "fresh", Visible: true, Wrap: TableColumnClip},
		{Column: "count", Visible: true, Wrap: TableColumnClip},
	}
	if got := table.ColumnPresentation(); !sameTableColumnPresentation(got, want) {
		t.Fatalf("canonical presentation = %#v, want %#v", got, want)
	}
	if err := table.SetColumnPresentation([]TableColumnPresentation{
		{Column: "state", Visible: true},
		{Column: "fresh", Visible: false, Wrap: TableColumnHang},
		{Column: "count", Visible: false, Wrap: TableColumnWrapWords},
	}); err != nil {
		t.Fatalf("SetColumnPresentation(repair fixture) error = %v", err)
	}
	if err := table.SetModel(
		[]Column{{Key: "fresh", Header: "Fresh"}, {Key: "count", Header: "Count"}},
		[]TableRow{{Key: "final-row"}},
	); err != nil {
		t.Fatalf("SetModel(remove sole visible column) error = %v", err)
	}
	want = []TableColumnPresentation{
		{Column: "fresh", Visible: true, Wrap: TableColumnHang},
		{Column: "count", Visible: false, Wrap: TableColumnWrapWords},
	}
	if got := table.ColumnPresentation(); !sameTableColumnPresentation(got, want) ||
		table.State().CurrentColumn != "fresh" {
		t.Fatalf("no-visible repair presentation=%#v state=%#v", got, table.State())
	}
	if err := table.ReplaceWithPresentation(
		[]Column{{Key: "left", Header: "Left"}, {Key: "right", Header: "Right"}},
		[]TableRow{{Key: "replacement"}},
		[]TableColumnPresentation{
			{Column: "right", Visible: true, Wrap: TableColumnHang},
			{Column: "left", Visible: false},
		},
		"replacement",
		"left",
		nil,
		"",
		SortNone,
	); err != nil {
		t.Fatalf("ReplaceWithPresentation() error = %v", err)
	}
	if state := table.State(); state.CurrentColumn != "right" ||
		state.CurrentColumnIndex != 0 || state.VisibleColumnCount != 1 ||
		state.ColumnPresentation[0].Wrap != TableColumnHang {
		t.Fatalf("exact replacement state = %#v", state)
	}
}

func TestTableColumnPresentationControlsRenderingNavigationAndHiddenSort(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 34, Height: 9})
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "table.visible-columns", Bounds: Rect{Width: 22, Height: 7}},
		}},
		Columns: []Column{
			{Key: "name", Header: "Name", Width: 5},
			{Key: "state", Header: "State", Width: 5, Sortable: true},
			{Key: "count", Header: "Count", Width: 5},
		},
		Rows: []TableRow{
			{Key: "one", Cells: []TableCell{{Column: "name", Text: "One"}, {Column: "state", Text: "Zulu"}, {Column: "count", Text: "1"}}},
			{Key: "two", Cells: []TableCell{{Column: "name", Text: "Two"}, {Column: "state", Text: "Alpha"}, {Column: "count", Text: "2"}}},
		},
		CurrentRow: "one", CurrentColumn: "name", FocusMode: TableFocusCell,
		ColumnPresentation: []TableColumnPresentation{
			{Column: "count", Visible: true},
			{Column: "state", Visible: false},
			{Column: "name", Visible: true},
		},
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	if err := table.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	header := rowText(app.Snapshot(), 1)
	if got := string([]rune(header)[1:21]); got != "    Count│Name      " {
		t.Fatalf("exact presentation header = %q", got)
	}
	if countAt, nameAt := strings.Index(header, "Count"), strings.Index(header, "Name"); countAt < 0 || nameAt <= countAt || strings.Contains(header, "State") {
		t.Fatalf("presentation header = %q", header)
	}
	firstRow := rowText(app.Snapshot(), 2)
	if got := string([]rune(firstRow)[1:21]); got != "►[ ]1    │One       " {
		t.Fatalf("exact presentation row = %q", got)
	}
	if countAt, nameAt := strings.Index(firstRow, "1"), strings.Index(firstRow, "One"); countAt < 0 || nameAt <= countAt || strings.Contains(firstRow, "Zulu") {
		t.Fatalf("presentation row = %q", firstRow)
	}
	state := table.State()
	if state.CurrentColumnIndex != 1 || len(state.ColumnWidths) != 2 ||
		state.Offset.X != 0 {
		t.Fatalf("initial visible state = %#v", state)
	}
	dispatchTableKey(t, app, "visible-left", KeyLeft)
	if state = table.State(); state.CurrentColumn != "count" || state.CurrentColumnIndex != 0 {
		t.Fatalf("Left visible state = %#v", state)
	}
	dispatchTableKey(t, app, "visible-right", KeyRight)
	if state = table.State(); state.CurrentColumn != "name" || state.CurrentColumnIndex != 1 {
		t.Fatalf("Right visible state = %#v", state)
	}
	dispatchTableControlKey(t, app, "visible-home", KeyHome)
	if state = table.State(); state.CurrentColumn != "count" || state.CurrentRow != "one" {
		t.Fatalf("Ctrl-Home visible state = %#v", state)
	}
	dispatchTableControlKey(t, app, "visible-end", KeyEnd)
	if state = table.State(); state.CurrentColumn != "name" || state.CurrentRow != "two" {
		t.Fatalf("Ctrl-End visible state = %#v", state)
	}
	if err := table.SetSort("state", SortAscending); err != nil {
		t.Fatalf("SetSort(hidden) error = %v", err)
	}
	if state = table.State(); state.SortColumn != "state" || state.CurrentRowIndex != 0 {
		t.Fatalf("hidden sort state = %#v", state)
	}
	if firstRow = rowText(app.Snapshot(), 2); !strings.Contains(firstRow, "2") ||
		!strings.Contains(firstRow, "Two") || strings.Contains(firstRow, "Alpha") {
		t.Fatalf("hidden sorted row = %q", firstRow)
	}
}

func TestTableWrappedRowsExactFrameAndVisualGeometry(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 9})
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "table.wrap", Bounds: Rect{Width: 18, Height: 6}},
		}},
		Columns: []Column{
			{Key: "notes", Header: "Notes", Width: 6},
			{Key: "count", Header: "N", Width: 3},
		},
		Rows: []TableRow{
			{Key: "one", Cells: []TableCell{{Column: "notes", Text: "alpha beta"}, {Column: "count", Text: "7"}}},
			{Key: "two", Cells: []TableCell{{Column: "notes", Text: "gamma"}, {Column: "count", Text: "8"}}},
		},
		CurrentRow: "one", CurrentColumn: "notes", Selected: []string{"one"},
		FocusMode: TableFocusCell,
		ColumnPresentation: []TableColumnPresentation{
			{Column: "notes", Visible: true, Wrap: TableColumnWrapWords},
			{Column: "count", Visible: true},
		},
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	if err := table.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	for row, want := range map[int]string{
		1: "    Notes │N    ",
		2: "►[X]alpha │7    ",
		3: "    beta  │     ",
		4: " [ ]gamma │8    ",
	} {
		got := string([]rune(rowText(app.Snapshot(), row))[1:17])
		if got != want {
			t.Fatalf("frame row %d = %q, want %q", row, got, want)
		}
	}
	state := table.State()
	if state.RowCount != 2 || state.VisualRowCount != 3 ||
		state.Offset.Y != 0 {
		t.Fatalf("wrapped state = %#v", state)
	}
	details := controlByKey(t, app.Snapshot(), "table.wrap").Details.Table
	if details == nil || details.VisualRowCount != 3 ||
		details.Viewport.State.ContentSize.Height != 4 {
		t.Fatalf("wrapped details = %#v", details)
	}
	snapshot := app.Snapshot()
	for x := 1; x < 5; x++ {
		cell, _ := snapshot.Frame.Cell(x, 3)
		if cell.Style != "table.row_selected" {
			t.Fatalf("continuation marker style at %d = %q", x, cell.Style)
		}
	}
	for x := 5; x < 11; x++ {
		cell, _ := snapshot.Frame.Cell(x, 3)
		if cell.Style != "table.cell_current" {
			t.Fatalf("continuation current-cell style at %d = %q", x, cell.Style)
		}
	}
	dispatchTableKey(t, app, "wrapped-down", KeyDown)
	if state = table.State(); state.CurrentRow != "two" || state.Offset.Y != 0 {
		t.Fatalf("wrapped Down state = %#v", state)
	}
	if err := table.SetColumnPresentation([]TableColumnPresentation{
		{Column: "notes", Visible: true, Wrap: TableColumnHang},
		{Column: "count", Visible: true},
	}); err != nil {
		t.Fatalf("SetColumnPresentation(Hang) error = %v", err)
	}
	if got := string([]rune(rowText(app.Snapshot(), 3))[1:17]); got != "     beta │     " {
		t.Fatalf("Hang continuation frame = %q", got)
	}
}

func TestTableWrappedNavigationUsesVisualDistanceAndOffsets(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 26, Height: 8})
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Width: 16, Height: 5}},
		}},
		Columns: []Column{{Key: "notes", Header: "Notes", Width: 5}},
		Rows: []TableRow{
			{Key: "one", Cells: []TableCell{{Column: "notes", Text: "one two three four"}}},
			{Key: "two", Cells: []TableCell{{Column: "notes", Text: "two"}}},
			{Key: "three", Cells: []TableCell{{Column: "notes", Text: "three"}}},
		},
		CurrentRow: "one", CurrentColumn: "notes",
		ColumnPresentation: []TableColumnPresentation{{
			Column: "notes", Visible: true, Wrap: TableColumnWrapWords,
		}},
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	if state := table.State(); state.VisualRowCount != 6 || state.Offset.Y != 0 {
		t.Fatalf("initial wrapped navigation state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-offset-down", KeyDown)
	if state := table.State(); state.CurrentRow != "two" || state.Offset.Y != 4 {
		t.Fatalf("Down wrapped navigation state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-offset-up", KeyUp)
	if state := table.State(); state.CurrentRow != "one" || state.Offset.Y != 0 {
		t.Fatalf("Up wrapped navigation state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-page-down", KeyPageDown)
	if state := table.State(); state.CurrentRow != "two" || state.Offset.Y != 4 {
		t.Fatalf("PageDown wrapped navigation state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-page-down-last", KeyPageDown)
	if state := table.State(); state.CurrentRow != "three" || state.Offset.Y != 4 {
		t.Fatalf("PageDown last wrapped state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-page-up", KeyPageUp)
	if state := table.State(); state.CurrentRow != "one" || state.Offset.Y != 0 {
		t.Fatalf("PageUp wrapped navigation state = %#v", state)
	}
}

func TestTableWrappedGeometryCacheInvalidation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 8})
	table, err := NewTable(app.Root(), TableOptions{
		Columns: []Column{{Key: "value", Header: "Value", Width: 5, Sortable: true}},
		Rows: []TableRow{
			{Key: "one", Cells: []TableCell{{Column: "value", Text: "alpha beta"}}},
			{Key: "two", Cells: []TableCell{{Column: "value", Text: "gamma"}}},
		},
		CurrentRow: "one", CurrentColumn: "value",
		ColumnPresentation: []TableColumnPresentation{{
			Column: "value", Visible: true, Wrap: TableColumnWrapWords,
		}},
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	revisions := func() (uint64, uint64) {
		app.mu.RLock()
		defer app.mu.RUnlock()
		behavior := table.state.behavior.(tableBehavior)
		return behavior.geometryRevision, behavior.geometryCachedAt
	}
	initial, cached := revisions()
	if initial == 0 || cached != initial {
		t.Fatalf("initial geometry revisions = %d/%d", initial, cached)
	}
	if err := table.SetSelection([]string{"one"}); err != nil {
		t.Fatalf("SetSelection() error = %v", err)
	}
	if revision, at := revisions(); revision != initial || at != initial {
		t.Fatalf("selection invalidated geometry = %d/%d", revision, at)
	}
	if err := table.SetCurrent("two", "value"); err != nil {
		t.Fatalf("SetCurrent() error = %v", err)
	}
	if revision, at := revisions(); revision != initial || at != initial {
		t.Fatalf("navigation invalidated geometry = %d/%d", revision, at)
	}
	if err := table.SetColumnPresentation([]TableColumnPresentation{{
		Column: "value", Visible: true, Wrap: TableColumnHang,
	}}); err != nil {
		t.Fatalf("SetColumnPresentation() error = %v", err)
	}
	presentationRevision, at := revisions()
	if presentationRevision <= initial || at != presentationRevision {
		t.Fatalf("presentation geometry revisions = %d/%d", presentationRevision, at)
	}
	if err := table.SetSort("value", SortAscending); err != nil {
		t.Fatalf("SetSort() error = %v", err)
	}
	if revision, at := revisions(); revision <= presentationRevision || at != revision {
		t.Fatalf("sort geometry revisions = %d/%d", revision, at)
	}
}

func TestTableNavigationSelectionSortingAndCommands(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 46, Height: 10})
	for _, command := range []CommandID{"table.changed", "table.activate", "table.sort"} {
		registerActionCommand(t, app, command, string(command), true)
	}
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions:  PanelOptions{AutomationKey: "table.commands", Bounds: Rect{Width: 34, Height: 7}},
				ChangeCommand: "table.changed",
			},
			BorderForm: BorderSingle,
		},
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionMode: CollectionSelectionMultiple, RequireSelection: true,
		FocusMode: TableFocusCell, ActivateCommand: "table.activate", SortCommand: "table.sort",
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		_ = table.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	dispatchTableKey(t, app, "right", KeyRight)
	if state := table.State(); state.CurrentColumn != "state" {
		t.Fatalf("Right state = %#v", state)
	}
	dispatchTableKey(t, app, "down", KeyDown)
	if state := table.State(); state.CurrentRow != "alpha" || len(state.Selected) != 1 || state.Selected[0] != "bravo" {
		t.Fatalf("Down state = %#v", state)
	}
	space := dispatchTableKey(t, app, "space", KeySpace)
	if state := table.State(); len(state.Selected) != 2 || state.Selected[1] != "alpha" || space.Command != "table.changed" {
		t.Fatalf("Space state=%#v completion=%#v", state, space)
	}
	enter := dispatchTableKey(t, app, "enter", KeyEnter)
	if enter.Command != "table.activate" {
		t.Fatalf("Enter completion = %#v", enter)
	}

	dispatchTableKey(t, app, "left", KeyLeft)
	if state := table.State(); state.CurrentColumn != "name" {
		t.Fatalf("Left state = %#v", state)
	}
	sortUp := dispatchTableKey(t, app, "sort-up", Key("s"))
	if state := table.State(); state.SortColumn != "name" || state.SortDirection != SortAscending ||
		state.CurrentRow != "alpha" || state.CurrentRowIndex != 0 || sortUp.Command != "table.sort" {
		t.Fatalf("ascending state=%#v completion=%#v", state, sortUp)
	}
	dispatchTableKey(t, app, "sort-down", Key("s"))
	if state := table.State(); state.SortDirection != SortDescending || state.CurrentRowIndex != 2 ||
		state.Selected[0] != "bravo" || state.Selected[1] != "alpha" {
		t.Fatalf("descending state = %#v", state)
	}
	dispatchTableKey(t, app, "sort-none", Key("s"))
	if state := table.State(); state.SortColumn != "" || state.SortDirection != SortNone {
		t.Fatalf("unsorted state = %#v", state)
	}

	dispatchTableControlKey(t, app, "ctrl-home", KeyHome)
	if state := table.State(); state.CurrentRow != "bravo" || state.CurrentColumn != "name" {
		t.Fatalf("Ctrl-Home state = %#v", state)
	}
	dispatchTableControlKey(t, app, "ctrl-end", KeyEnd)
	if state := table.State(); state.CurrentRow != "alpha" || state.CurrentColumn != "count" {
		t.Fatalf("Ctrl-End state = %#v", state)
	}

	activation, err := table.Activate(context.Background(), "table-test", "direct-activate")
	if err != nil || activation.Command != "table.activate" {
		t.Fatalf("Activate() = %#v, %v", activation, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(routed) != 6 || routed[0].ID != "table.changed" || routed[1].ID != "table.activate" ||
		routed[2].ID != "table.sort" || routed[3].ID != "table.sort" ||
		routed[4].ID != "table.sort" || routed[5].ID != "table.activate" {
		t.Fatalf("routed commands = %#v", routed)
	}
}

func TestTableSelectionPolicyValidationAndRangeInteraction(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 46, Height: 10})
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionStyle: TableSelectionSingle,
		SelectionMode:  CollectionSelectionMultiple,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("conflicting legacy selection error = %v", err)
	}
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionStyle: TableSelectionNone, RequireSelection: true,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("required no-selection error = %v", err)
	}
	registerActionCommand(t, app, "range.changed", "Range changed", true)
	registerActionCommand(t, app, "range.activate", "Activate range row", true)
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions:  PanelOptions{AutomationKey: "table.range", Bounds: Rect{Width: 34, Height: 7}},
			ChangeCommand: "range.changed",
		}},
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionStyle: TableSelectionRange, RequireSelection: true,
		CurrentRow: "bravo", CurrentColumn: "name", ActivateCommand: "range.activate",
	})
	if err != nil {
		t.Fatalf("NewTable(range) error = %v", err)
	}
	if err := table.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	state := table.State()
	if state.SelectionStyle != TableSelectionRange || state.RangeAnchor != "bravo" ||
		state.RangeExtent != "bravo" || !sameTableSelection(state.Selected, []string{"bravo"}) {
		t.Fatalf("initial range state = %#v", state)
	}
	completion := dispatchTextChord(t, app, "range-extend", KeyShift, KeyDown)
	state = table.State()
	if completion.Command != "range.changed" || state.CurrentRow != "alpha" ||
		state.RangeAnchor != "bravo" || state.RangeExtent != "alpha" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("extended range state=%#v completion=%#v", state, completion)
	}
	dispatchTableKey(t, app, "range-current-only", KeyUp)
	if state = table.State(); state.CurrentRow != "bravo" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("unmodified navigation state = %#v", state)
	}
	dispatchTableKey(t, app, "range-contract", Key("["))
	if state = table.State(); state.RangeExtent != "bravo" ||
		!sameTableSelection(state.Selected, []string{"bravo"}) {
		t.Fatalf("contracted range state = %#v", state)
	}
	dispatchTableKey(t, app, "range-expand", Key("]"))
	dispatchTableKey(t, app, "range-sort", Key("s"))
	if state = table.State(); state.SortDirection != SortAscending ||
		state.RangeAnchor != "bravo" || state.RangeExtent != "alpha" ||
		!sameTableSelection(state.Selected, []string{"alpha", "bravo"}) {
		t.Fatalf("sorted range state = %#v", state)
	}
	if err := table.SetRows([]TableRow{{
		Key: "charlie", Cells: []TableCell{{Column: "name", Text: "Charlie"}},
	}}); err != nil {
		t.Fatalf("SetRows(repair range) error = %v", err)
	}
	if state = table.State(); state.CurrentRow != "charlie" ||
		state.RangeAnchor != "charlie" || state.RangeExtent != "charlie" ||
		!sameTableSelection(state.Selected, []string{"charlie"}) {
		t.Fatalf("repaired required range state = %#v", state)
	}
	if err := table.SetSelectionPolicy(TableSelectionPolicy{Style: TableSelectionNone}); err != nil {
		t.Fatalf("SetSelectionPolicy(none) error = %v", err)
	}
	completion = dispatchTableKey(t, app, "none-space", KeySpace)
	if state = table.State(); state.SelectionStyle != TableSelectionNone ||
		len(state.Selected) != 0 || state.RangeAnchor != "" || state.RangeExtent != "" ||
		completion.Outcome != OutcomeNoOp {
		t.Fatalf("no-selection Space state=%#v completion=%#v", state, completion)
	}
	control := controlByKey(t, app.Snapshot(), "table.range")
	if got := rowText(app.Snapshot(), control.AbsoluteBounds.Y+2); !strings.Contains(got, "►   Charlie") {
		t.Fatalf("no-selection marker row = %q", got)
	}
	completion = dispatchTableKey(t, app, "none-enter", KeyEnter)
	if state = table.State(); completion.Command != "range.activate" || len(state.Selected) != 0 {
		t.Fatalf("no-selection Enter state=%#v completion=%#v", state, completion)
	}
}

func TestTableRangeExactSelectionValidation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 9})
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionStyle: TableSelectionRange,
		Selected:       []string{"bravo", "alpha"},
		RangeAnchor:    "bravo", RangeExtent: "bravo",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("mismatched range error = %v", err)
	}
	table, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
		SelectionStyle: TableSelectionRange,
		RangeAnchor:    "alpha", RangeExtent: "bravo",
	})
	if err != nil {
		t.Fatalf("NewTable(exact endpoints) error = %v", err)
	}
	state := table.State()
	if state.RangeAnchor != "alpha" || state.RangeExtent != "bravo" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("exact endpoint state = %#v", state)
	}
	if err := table.SetSelectionPolicy(TableSelectionPolicy{
		Style: TableSelectionMultiple, Selected: []string{"alpha", "bravo"},
	}); err != nil {
		t.Fatalf("SetSelectionPolicy(multiple) error = %v", err)
	}
	if state = table.State(); state.SelectionStyle != TableSelectionMultiple ||
		state.RangeAnchor != "" || state.RangeExtent != "" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("multiple policy state = %#v", state)
	}
	tx := app.NewTransaction()
	if err := tx.SetTableSelectionPolicy(table, TableSelectionPolicy{
		Style: TableSelectionNone,
	}); err != nil {
		t.Fatalf("staged None policy error = %v", err)
	}
	if err := tx.SetTableSelectionPolicy(table, TableSelectionPolicy{
		Style: TableSelectionRange, Anchor: "bravo", Extent: "alpha",
	}); err != nil {
		t.Fatalf("staged Range policy error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("policy transaction Commit() error = %v", err)
	}
	if state = table.State(); state.SelectionStyle != TableSelectionRange ||
		state.RangeAnchor != "bravo" || state.RangeExtent != "alpha" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("transaction range policy state = %#v", state)
	}
}

func TestTableStableIdentityRepairModelAndStatus(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 9})
	table, err := NewTable(app.Root(), TableOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
			AutomationKey: "table.repair", Bounds: Rect{Width: 30, Height: 6},
		}}},
		Columns: sampleTableColumns(), Rows: sampleTableRows(), CurrentRow: "alpha",
		CurrentColumn: "state", Selected: []string{"alpha"}, RequireSelection: true,
		SortColumn: "name", SortDirection: SortAscending,
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	if err := table.SetRows([]TableRow{
		{Key: "charlie", Cells: []TableCell{{Column: "name", Text: "Charlie"}}},
		{Key: "alpha", Cells: []TableCell{{Column: "name", Text: "Alpha changed"}}},
	}); err != nil {
		t.Fatalf("SetRows() error = %v", err)
	}
	if state := table.State(); state.CurrentRow != "alpha" || state.CurrentRowIndex != 0 ||
		len(state.Selected) != 1 || state.Selected[0] != "alpha" || state.SortColumn != "name" {
		t.Fatalf("preserved state = %#v", state)
	}
	if err := table.SetRows([]TableRow{{Key: "charlie", Cells: []TableCell{{Column: "name", Text: "Charlie"}}}}); err != nil {
		t.Fatalf("SetRows(remove current) error = %v", err)
	}
	if state := table.State(); state.CurrentRow != "charlie" || len(state.Selected) != 1 || state.Selected[0] != "charlie" {
		t.Fatalf("repaired state = %#v", state)
	}
	columns := []Column{{Key: "new", Header: "New"}}
	if err := table.SetModel(columns, []TableRow{{Key: "charlie", Cells: []TableCell{{Column: "new", Text: "Value"}}}}); err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}
	if state := table.State(); state.CurrentColumn != "new" || state.SortColumn != "" || state.SortDirection != SortNone {
		t.Fatalf("schema repair state = %#v", state)
	}
	if err := table.SetStatus(CollectionLoading, "Fetching rows"); err != nil {
		t.Fatalf("SetStatus(loading) error = %v", err)
	}
	if app.Focused() == table {
		t.Fatal("loading Table retained focus")
	}
	details := controlByKey(t, app.Snapshot(), "table.repair").Details.Table
	if details == nil || details.Status != CollectionLoading || details.StatusMessage != "Fetching rows" ||
		details.Viewport.State.ContentSize.Height != 2 {
		t.Fatalf("loading details = %#v", details)
	}
	if err := table.SetStatus(CollectionError, ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetStatus(error without message) error = %v", err)
	}
	if err := table.SetStatus(CollectionReady, ""); err != nil {
		t.Fatalf("SetStatus(ready) error = %v", err)
	}
}

func TestTableValidationBoundsAndTransactionAtomicity(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	for name, options := range map[string]TableOptions{
		"duplicate column":       {Columns: []Column{{Key: "same", Header: "One"}, {Key: "same", Header: "Two"}}},
		"unknown cell":           {Columns: []Column{{Key: "known", Header: "Known"}}, Rows: []TableRow{{Key: "row", Cells: []TableCell{{Column: "missing", Text: "X"}}}}},
		"duplicate cell":         {Columns: []Column{{Key: "known", Header: "Known"}}, Rows: []TableRow{{Key: "row", Cells: []TableCell{{Column: "known", Text: "X"}, {Column: "known", Text: "Y"}}}}},
		"unsortable":             {Columns: []Column{{Key: "plain", Header: "Plain"}}, SortColumn: "plain", SortDirection: SortAscending},
		"validator without edit": {Columns: []Column{{Key: "value", Header: "Value", Validator: &TextValidator{Enforcement: TextValidationHard, Mode: TextValidationWhitelist, Characters: "x"}}}},
	} {
		options.ScrollablePanelOptions.ScrollViewOptions.PanelOptions.Bounds = Rect{Width: 20, Height: 5}
		if _, err := NewTable(app.Root(), options); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s error = %v", name, err)
		}
	}
	tooManyColumns := make([]Column, MaxCollectionColumns+1)
	for index := range tooManyColumns {
		tooManyColumns[index] = Column{Key: fmt.Sprintf("c-%d", index), Header: "Column"}
	}
	if _, err := NewTable(app.Root(), TableOptions{Columns: tooManyColumns}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("over-capacity columns error = %v", err)
	}
	aggregateColumns := make([]Column, MaxCollectionColumns)
	for index := range aggregateColumns {
		aggregateColumns[index] = Column{Key: fmt.Sprintf("aggregate-%d", index), Header: "C"}
	}
	aggregateRows := make([]TableRow, MaxCollectionCells/MaxCollectionColumns/2+1)
	for rowIndex := range aggregateRows {
		aggregateRows[rowIndex].Key = fmt.Sprintf("aggregate-row-%d", rowIndex)
		aggregateRows[rowIndex].Cells = make([]TableCell, len(aggregateColumns))
		for columnIndex, column := range aggregateColumns {
			aggregateRows[rowIndex].Cells[columnIndex] = TableCell{Column: column.Key}
		}
	}
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: aggregateColumns,
		Rows:    aggregateRows,
	}); err != nil {
		t.Fatalf("first aggregate Table error = %v", err)
	}
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: aggregateColumns,
		Rows:    aggregateRows,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("aggregate cell capacity error = %v", err)
	}
	if _, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(), SortCommand: "missing.command",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("unregistered sort command error = %v", err)
	}

	table, err := NewTable(app.Root(), TableOptions{
		Columns: []Column{{Key: "value", Header: "Value"}},
		Rows:    []TableRow{{Key: "one", Cells: []TableCell{{Column: "value", Text: "One"}}}},
	})
	if err != nil {
		t.Fatalf("NewTable(valid) error = %v", err)
	}
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetTableRows(table, []TableRow{{Key: "two", Cells: []TableCell{{Column: "value", Text: "Two"}}}}); err != nil {
		t.Fatalf("SetTableRows() error = %v", err)
	}
	if err := tx.SetTableCurrent(table, "missing", "value"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetTableCurrent(missing) error = %v", err)
	}
	if state := table.State(); state.CurrentRow != "one" || app.Snapshot().Sequence != before {
		t.Fatalf("uncommitted transaction leaked state=%#v", state)
	}
}

func TestTableConcurrentReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	columns := []Column{{Key: "value", Header: "Value", Sortable: true}}
	table, err := NewTable(app.Root(), TableOptions{
		Columns:    columns,
		Rows:       []TableRow{{Key: "a", Cells: []TableCell{{Column: "value", Text: "A"}}}, {Key: "b", Cells: []TableCell{{Column: "value", Text: "B"}}}},
		CurrentRow: "a", CurrentColumn: "value", Selected: []string{"a"}, RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 6)
	for worker := range 6 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := range 20 {
				first, second := "a", "b"
				if (worker+iteration)%2 != 0 {
					first, second = second, first
				}
				rows := []TableRow{
					{Key: first, Cells: []TableCell{{Column: "value", Text: strings.ToUpper(first)}}},
					{Key: second, Cells: []TableCell{{Column: "value", Text: strings.ToUpper(second)}}},
				}
				if err := table.Replace(columns, rows, first, "value", []string{first}, "", SortNone); err != nil {
					errorsSeen <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent Replace() error = %v", err)
	}
	state, rows := table.State(), table.Rows()
	if len(rows) != 2 || state.RowCount != 2 || state.CurrentRow != rows[0].Key ||
		len(state.Selected) != 1 || state.Selected[0] != state.CurrentRow {
		t.Fatalf("concurrent final state=%#v rows=%#v", state, rows)
	}
}

func TestTableConcurrentSelectionPolicyReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 7})
	table, err := NewTable(app.Root(), TableOptions{
		Columns: []Column{{Key: "value", Header: "Value"}},
		Rows: []TableRow{
			{Key: "a", Cells: []TableCell{{Column: "value", Text: "A"}}},
			{Key: "b", Cells: []TableCell{{Column: "value", Text: "B"}}},
		},
		CurrentRow: "a", CurrentColumn: "value",
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	policies := []TableSelectionPolicy{
		{Style: TableSelectionNone},
		{Style: TableSelectionSingle, Selected: []string{"a"}},
		{Style: TableSelectionRange, Anchor: "a", Extent: "b"},
		{Style: TableSelectionMultiple, Selected: []string{"a", "b"}},
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, len(policies))
	for worker, policy := range policies {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 40 {
				if err := table.SetSelectionPolicy(policy); err != nil {
					errorsSeen <- fmt.Errorf("worker %d: %w", worker, err)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	state := table.State()
	switch state.SelectionStyle {
	case TableSelectionNone:
		if len(state.Selected) != 0 || state.RangeAnchor != "" || state.RangeExtent != "" {
			t.Fatalf("final None state = %#v", state)
		}
	case TableSelectionSingle:
		if !sameTableSelection(state.Selected, []string{"a"}) ||
			state.RangeAnchor != "" || state.RangeExtent != "" {
			t.Fatalf("final Single state = %#v", state)
		}
	case TableSelectionRange:
		if !sameTableSelection(state.Selected, []string{"a", "b"}) ||
			state.RangeAnchor != "a" || state.RangeExtent != "b" {
			t.Fatalf("final Range state = %#v", state)
		}
	case TableSelectionMultiple:
		if !sameTableSelection(state.Selected, []string{"a", "b"}) ||
			state.RangeAnchor != "" || state.RangeExtent != "" {
			t.Fatalf("final Multiple state = %#v", state)
		}
	default:
		t.Fatalf("final selection style = %q", state.SelectionStyle)
	}
}

func TestTableConcurrentColumnPresentationReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	table, err := NewTable(app.Root(), TableOptions{
		Columns: sampleTableColumns(), Rows: sampleTableRows(),
	})
	if err != nil {
		t.Fatalf("NewTable() error = %v", err)
	}
	presentations := [][]TableColumnPresentation{
		{
			{Column: "name", Visible: true},
			{Column: "state", Visible: true, Wrap: TableColumnWrapWords},
			{Column: "count", Visible: false, Wrap: TableColumnHang},
		},
		{
			{Column: "count", Visible: true, Wrap: TableColumnHang},
			{Column: "name", Visible: false},
			{Column: "state", Visible: true},
		},
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, len(presentations))
	for worker, presentation := range presentations {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 40 {
				if err := table.SetColumnPresentation(presentation); err != nil {
					errorsSeen <- fmt.Errorf("worker %d: %w", worker, err)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	state := table.State()
	if state.VisibleColumnCount < 1 || len(state.ColumnPresentation) != 3 ||
		!tableColumnIsVisible(tableBehavior{columnPresentation: state.ColumnPresentation}, state.CurrentColumn) {
		t.Fatalf("final presentation state = %#v", state)
	}
}

func FuzzTableCellNormalization(f *testing.F) {
	for _, seed := range []string{"value", "e\u0301", "wide界", string([]byte{0xff}), ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 {
			t.Skip()
		}
		columns, err := normalizeTableColumns([]Column{{Key: "value", Header: "Value"}})
		if err != nil {
			t.Fatal(err)
		}
		rows, err := normalizeTableRows(columns, []TableRow{{
			Key: "row", Cells: []TableCell{{Column: "value", Text: value}},
		}})
		if err != nil {
			return
		}
		if len(rows) != 1 || len(rows[0].cells) != 1 ||
			!utf8.ValidString(rows[0].cells[0].cell.Text) ||
			strings.Contains(rows[0].cells[0].cell.Text, "\n") {
			t.Fatalf("normalized Table cell = %#v", rows)
		}
	})
}

func FuzzTableColumnPresentationNormalization(f *testing.F) {
	f.Add("name", "state", "count", true, false, true, "clip", "wrap", "hang")
	f.Fuzz(func(
		t *testing.T,
		first, second, third string,
		firstVisible, secondVisible, thirdVisible bool,
		firstWrap, secondWrap, thirdWrap string,
	) {
		columns, err := normalizeTableColumns(sampleTableColumns())
		if err != nil {
			t.Fatal(err)
		}
		presentation, err := normalizeTableColumnPresentation(
			columns,
			[]TableColumnPresentation{
				{Column: first, Visible: firstVisible, Wrap: TableColumnWrap(firstWrap)},
				{Column: second, Visible: secondVisible, Wrap: TableColumnWrap(secondWrap)},
				{Column: third, Visible: thirdVisible, Wrap: TableColumnWrap(thirdWrap)},
			},
		)
		if err != nil {
			return
		}
		if len(presentation) != len(columns) || visibleTableColumnCount(presentation) < 1 {
			t.Fatalf("normalized presentation = %#v", presentation)
		}
		seen := map[string]bool{}
		for _, entry := range presentation {
			if seen[entry.Column] ||
				(entry.Wrap != TableColumnClip && entry.Wrap != TableColumnWrapWords && entry.Wrap != TableColumnHang) {
				t.Fatalf("normalized presentation = %#v", presentation)
			}
			seen[entry.Column] = true
		}
	})
}

func FuzzTableColumnWrapping(f *testing.F) {
	for _, seed := range []struct {
		text  string
		width int
		wrap  string
	}{
		{"alpha beta", 6, "wrap"},
		{"abcdefgh", 3, "hang"},
		{"e\u0301界", 1, "wrap"},
		{"   ", 2, "hang"},
	} {
		f.Add(seed.text, seed.width, seed.wrap)
	}
	f.Fuzz(func(t *testing.T, text string, width int, wrapValue string) {
		if len(text) > 4096 || width < 1 || width > 256 {
			t.Skip()
		}
		wrap, err := normalizeTableColumnWrap(TableColumnWrap(wrapValue))
		if err != nil || wrap == TableColumnClip {
			return
		}
		normalized, err := normalizeDisplayText(text, false)
		if err != nil {
			return
		}
		lines := wrapTableCellLines(normalized.lines[0], width, wrap)
		if len(lines) < 1 || len(lines) > max(1, len(normalized.lines[0])) {
			t.Fatalf("wrapped line count = %d", len(lines))
		}
		for index, line := range lines {
			if line.indent < 0 || line.indent >= width ||
				line.indent+len(line.cells) > width ||
				(wrap == TableColumnHang && width > 1 && index > 0 && line.indent != 1) ||
				(wrap == TableColumnWrapWords && line.indent != 0) {
				t.Fatalf("wrapped line %d = %#v", index, line)
			}
		}
	})
}
