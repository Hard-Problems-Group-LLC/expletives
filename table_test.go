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
	if table.Columns()[0].Header != "Name" || table.Rows()[0].Cells[0].Text != "Bravo" ||
		table.State().Selected[0] != "alpha" || table.State().ColumnWidths[0] == 999 {
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
		len(details.SelectionDigest) != 64 || len(details.ColumnWidthsDigest) != 64 ||
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
