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
