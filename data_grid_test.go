package expletives

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func sampleDataGridColumns() []Column {
	alphanumeric := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return []Column{
		{
			Key: "name", Header: "Name", MinimumWidth: 8, Grow: 1,
			Sortable: true, Editable: true,
			Validator: &TextValidator{
				Enforcement: TextValidationSoft,
				Mode:        TextValidationWhitelist,
				Characters:  alphanumeric,
			},
		},
		{
			Key: "state", Header: "State", Width: 8, Editable: true,
			Validator: &TextValidator{
				Enforcement: TextValidationHard,
				Mode:        TextValidationBlacklist,
				Characters:  "!@#$%",
			},
		},
		{Key: "count", Header: "Count", Width: 5, Alignment: TextAlignEnd},
	}
}

func sampleDataGridRows() []TableRow {
	return []TableRow{
		{Key: "alpha", Cells: []TableCell{
			{Column: "name", Text: "Alpha"},
			{Column: "state", Text: "Ready"},
			{Column: "count", Text: "1"},
		}},
		{Key: "bravo", Cells: []TableCell{
			{Column: "name", Text: "Bravo"},
			{Column: "state", Text: "Idle"},
			{Column: "count", Text: "2"},
		}},
	}
}

func dispatchGridText(t *testing.T, app *App, request, text string) Completion {
	t.Helper()
	completion, err := app.DispatchTextInput(
		context.Background(),
		"data-grid-test",
		request,
		TextInputEvent{Kind: TextInputCommitted, Text: text},
	)
	if err != nil {
		t.Fatalf("DispatchTextInput(%q) error = %v", text, err)
	}
	return completion
}

func TestDataGridRenderingDetailsAndCopiedState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 48, Height: 10})
	columns, rows := sampleDataGridColumns(), sampleDataGridRows()
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "collection.data-grid",
				Bounds:        Rect{X: 1, Y: 1, Width: 38, Height: 7},
			}},
			BorderForm: BorderSingle,
		},
		Columns: columns, Rows: rows, CurrentRow: "alpha", CurrentColumn: "name",
		Selected: []string{"alpha"}, RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	columns[0].Header = "caller mutation"
	rows[0].Cells[0].Text = "caller mutation"
	if grid.Columns()[0].Header != "Name" || grid.Rows()[0].Cells[0].Text != "Alpha" {
		t.Fatal("DataGrid retained caller-owned model storage")
	}
	state := grid.State()
	if state.FocusMode != TableFocusCell || state.CurrentRow != "alpha" ||
		state.CurrentColumn != "name" ||
		state.SelectionStyle != TableSelectionSingle || state.Editing {
		t.Fatalf("initial DataGrid state = %#v", state)
	}
	control := controlByKey(t, app.Snapshot(), "collection.data-grid")
	details := control.Details.DataGrid
	if details == nil || control.Details.Table != nil || details.Table.RowCount != 2 ||
		details.Table.ColumnCount != 3 || details.Table.FocusMode != TableFocusCell ||
		details.Table.SelectionStyle != TableSelectionSingle ||
		details.Table.RangeAnchor != "" || details.Table.RangeExtent != "" ||
		details.Editing || details.Editor != nil || control.Details.Border == nil {
		t.Fatalf("DataGrid details = %#v", control.Details)
	}
	if got := rowText(app.Snapshot(), 2); got == "" {
		t.Fatal("DataGrid sticky header was not painted")
	}
}

func TestDataGridColumnPresentationCancelsEditorAndCopiesState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 48, Height: 10})
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "grid.presentation", Bounds: Rect{Width: 38, Height: 7}},
		}},
		Columns: sampleDataGridColumns(), Rows: sampleDataGridRows(),
		CurrentRow: "alpha", CurrentColumn: "name",
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	dispatchTableKey(t, app, "grid-presentation-edit", KeyEnter)
	if !grid.State().Editing {
		t.Fatal("DataGrid did not begin editing")
	}
	presentation := []TableColumnPresentation{
		{Column: "state", Visible: true, Wrap: TableColumnHang},
		{Column: "name", Visible: false},
		{Column: "count", Visible: true, Wrap: TableColumnWrapWords},
	}
	if err := grid.SetColumnPresentation(presentation); err != nil {
		t.Fatalf("SetColumnPresentation() error = %v", err)
	}
	presentation[0].Column = "caller-mutation"
	state := grid.State()
	if state.Editing || state.CurrentColumn != "count" || state.VisibleColumnCount != 2 ||
		state.ColumnPresentation[0].Column != "state" {
		t.Fatalf("presentation state = %#v", state)
	}
	copyValue := grid.ColumnPresentation()
	copyValue[0].Column = "getter-mutation"
	if grid.ColumnPresentation()[0].Column != "state" {
		t.Fatal("ColumnPresentation() exposed retained DataGrid storage")
	}

	if err := grid.SetCurrent("alpha", "name"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetCurrent(hidden column) error = %v", err)
	}
	if err := grid.SetCurrent("alpha", "state"); err != nil {
		t.Fatalf("SetCurrent(visible column) error = %v", err)
	}
	dispatchTableKey(t, app, "grid-presentation-edit-again", KeyEnter)
	if !grid.State().Editing {
		t.Fatal("DataGrid did not resume editing")
	}
	before := grid.ColumnPresentation()
	if err := grid.SetColumnPresentation([]TableColumnPresentation{
		{Column: "name"}, {Column: "state"}, {Column: "count"},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid presentation error = %v", err)
	}
	if !grid.State().Editing ||
		!sameTableColumnPresentation(grid.ColumnPresentation(), before) {
		t.Fatal("failed presentation mutation changed DataGrid or cancelled editor")
	}

	tx := app.NewTransaction()
	if err := tx.SetDataGridColumnPresentation(grid, nil); err != nil {
		t.Fatalf("SetDataGridColumnPresentation(nil) error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	state = grid.State()
	if state.Editing || state.VisibleColumnCount != 3 ||
		state.ColumnPresentation[0].Column != "name" ||
		state.ColumnPresentation[0].Wrap != TableColumnClip {
		t.Fatalf("canonical transaction state = %#v", state)
	}
	dispatchTableKey(t, app, "grid-exact-replacement-edit", KeyEnter)
	if !grid.State().Editing {
		t.Fatal("DataGrid did not edit before exact replacement")
	}
	tx = app.NewTransaction()
	if err := tx.ReplaceDataGridWithPresentation(
		grid,
		[]Column{{Key: "new", Header: "New", Editable: true}},
		[]TableRow{{Key: "new-row", Cells: []TableCell{{Column: "new", Text: "Value"}}}},
		[]TableColumnPresentation{{Column: "new", Visible: true, Wrap: TableColumnHang}},
		"new-row",
		"new",
		nil,
		"",
		SortNone,
	); err != nil {
		t.Fatalf("ReplaceDataGridWithPresentation() error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("replacement Commit() error = %v", err)
	}
	if state = grid.State(); state.Editing || state.CurrentColumn != "new" ||
		state.ColumnPresentation[0].Wrap != TableColumnHang {
		t.Fatalf("exact replacement state = %#v", state)
	}
}

func TestDataGridPresentationLimitsEditingToVisibleDisplayOrder(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 46, Height: 10})
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "grid.visible-edit", Bounds: Rect{Width: 36, Height: 7}},
		}},
		Columns: sampleDataGridColumns(), Rows: sampleDataGridRows(),
		CurrentRow: "alpha", CurrentColumn: "name",
		ColumnPresentation: []TableColumnPresentation{
			{Column: "state", Visible: false},
			{Column: "count", Visible: true},
			{Column: "name", Visible: true},
		},
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	if err := grid.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	dispatchTableKey(t, app, "visible-grid-edit", KeyEnter)
	if state := grid.State(); !state.Editing || state.EditColumn != "name" ||
		state.CurrentColumnIndex != 1 {
		t.Fatalf("initial editor state = %#v", state)
	}
	dispatchTableKey(t, app, "visible-grid-tab", KeyTab)
	if state := grid.State(); !state.Editing || state.CurrentRow != "bravo" ||
		state.CurrentColumn != "name" || state.EditColumn != "name" {
		t.Fatalf("visible Tab editor state = %#v", state)
	}
	dispatchTextChord(t, app, "visible-grid-backtab", KeyShift, KeyTab)
	if state := grid.State(); !state.Editing || state.CurrentRow != "alpha" ||
		state.CurrentColumn != "name" || state.EditColumn != "name" {
		t.Fatalf("visible Shift-Tab editor state = %#v", state)
	}
}

func TestDataGridWrappedCommittedGeometryAndSingleLineEditor(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 8})
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{AutomationKey: "grid.wrap-edit", Bounds: Rect{Width: 15, Height: 5}},
		}},
		Columns: []Column{
			{Key: "name", Header: "Name", Width: 5, Editable: true, Sortable: true},
			{Key: "secret", Header: "Secret", Width: 5, Editable: true, Sortable: true},
		},
		Rows: []TableRow{
			{Key: "one", Cells: []TableCell{{Column: "name", Text: "alpha beta"}, {Column: "secret", Text: "Zulu"}}},
			{Key: "two", Cells: []TableCell{{Column: "name", Text: "gamma"}, {Column: "secret", Text: "Alpha"}}},
		},
		CurrentRow: "one", CurrentColumn: "name",
		ColumnPresentation: []TableColumnPresentation{
			{Column: "secret", Visible: false},
			{Column: "name", Visible: true, Wrap: TableColumnWrapWords},
		},
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	if err := grid.Focus(); err != nil {
		t.Fatalf("Focus() error = %v", err)
	}
	if state := grid.State(); state.VisualRowCount != 3 {
		t.Fatalf("initial wrapped DataGrid state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-grid-edit", KeyEnter)
	beforeRevision := uint64(0)
	app.mu.RLock()
	beforeRevision = grid.state.behavior.(dataGridBehavior).table.geometryRevision
	app.mu.RUnlock()
	dispatchGridText(t, app, "wrapped-grid-append", " gamma")
	state := grid.State()
	if !state.Editing || state.VisualRowCount != 3 || !app.Snapshot().Cursor.Visible ||
		app.Snapshot().Cursor.Position.Y != 2 {
		t.Fatalf("active single-line wrapped editor state=%#v cursor=%#v", state, app.Snapshot().Cursor)
	}
	app.mu.RLock()
	afterInputRevision := grid.state.behavior.(dataGridBehavior).table.geometryRevision
	app.mu.RUnlock()
	if afterInputRevision != beforeRevision {
		t.Fatalf("uncommitted edit invalidated geometry: %d -> %d", beforeRevision, afterInputRevision)
	}
	dispatchTableKey(t, app, "wrapped-grid-commit", KeyEnter)
	state = grid.State()
	if state.Editing || state.VisualRowCount != 4 ||
		grid.Rows()[0].Cells[0].Text != "alpha beta gamma" {
		t.Fatalf("committed wrapped editor state=%#v rows=%#v", state, grid.Rows())
	}
	app.mu.RLock()
	afterCommitRevision := grid.state.behavior.(dataGridBehavior).table.geometryRevision
	app.mu.RUnlock()
	if afterCommitRevision != beforeRevision+1 {
		t.Fatalf("commit geometry revision = %d, want %d", afterCommitRevision, beforeRevision+1)
	}
	dispatchTableKey(t, app, "wrapped-grid-edit-again", KeyEnter)
	dispatchTableKey(t, app, "wrapped-grid-visible-tab", KeyTab)
	if state = grid.State(); !state.Editing || state.CurrentRow != "two" ||
		state.CurrentColumn != "name" || state.EditColumn != "name" {
		t.Fatalf("wrapped visible Tab state = %#v", state)
	}
	dispatchTableKey(t, app, "wrapped-grid-cancel", KeyEscape)
	if err := grid.SetSort("secret", SortAscending); err != nil {
		t.Fatalf("SetSort(hidden editable) error = %v", err)
	}
	if state = grid.State(); state.SortColumn != "secret" ||
		state.SortDirection != SortAscending || state.CurrentRowIndex != 0 {
		t.Fatalf("hidden sorted DataGrid state = %#v", state)
	}
}

func TestDataGridSoftHardEditingCommitCancelAndTab(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 48, Height: 10})
	registerActionCommand(t, app, "grid.changed", "Grid changed", true)
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "grid.edit", Bounds: Rect{Width: 38, Height: 7},
				},
				ChangeCommand: "grid.changed",
			},
			BorderForm: BorderSingle,
		},
		Columns: sampleDataGridColumns(), Rows: sampleDataGridRows(),
		CurrentRow: "alpha", CurrentColumn: "name",
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(_ context.Context, command Command) CommandResult {
		_ = grid.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	dispatchTableKey(t, app, "edit-name", KeyEnter)
	dispatchGridText(t, app, "soft-invalid", "!")
	if state := grid.State(); !state.Editing || state.EditValid || state.EditText != "Alpha!" {
		t.Fatalf("soft-invalid state = %#v", state)
	}
	completion := dispatchTableKey(t, app, "reject-soft", KeyEnter)
	if state := grid.State(); !state.Editing || completion.Outcome != OutcomeNoOp ||
		grid.Rows()[0].Cells[0].Text != "Alpha" {
		t.Fatalf("soft commit state=%#v completion=%#v", state, completion)
	}
	dispatchTableKey(t, app, "remove-invalid", KeyBackspace)
	dispatchGridText(t, app, "append-valid", "X")
	completion = dispatchTableKey(t, app, "commit-name", KeyEnter)
	if state := grid.State(); state.Editing || completion.Command != "grid.changed" ||
		grid.Rows()[0].Cells[0].Text != "AlphaX" {
		t.Fatalf("valid commit state=%#v completion=%#v rows=%#v", state, completion, grid.Rows())
	}

	dispatchTableKey(t, app, "right-state", KeyRight)
	dispatchTableKey(t, app, "edit-state", KeyF2)
	if completion := dispatchGridText(t, app, "hard-reject", "!"); completion.Outcome != OutcomeNoOp {
		t.Fatalf("hard-invalid input completion = %#v", completion)
	}
	if state := grid.State(); state.EditText != "Ready" || !state.EditValid {
		t.Fatalf("hard-invalid state = %#v", state)
	}
	dispatchGridText(t, app, "state-valid", "X")
	dispatchTableKey(t, app, "cancel-state", KeyEscape)
	if state := grid.State(); state.Editing || grid.Rows()[0].Cells[1].Text != "Ready" {
		t.Fatalf("cancel state=%#v rows=%#v", state, grid.Rows())
	}

	dispatchTableKey(t, app, "edit-state-tab", KeyF2)
	dispatchGridText(t, app, "state-tab-value", "Z")
	dispatchTableKey(t, app, "tab-next", KeyTab)
	if state := grid.State(); !state.Editing || state.CurrentRow != "bravo" ||
		state.CurrentColumn != "name" || state.EditText != "Bravo" ||
		grid.Rows()[0].Cells[1].Text != "ReadyZ" {
		t.Fatalf("Tab advance state=%#v rows=%#v", state, grid.Rows())
	}
	dispatchTextChord(t, app, "shift-tab-previous", KeyShift, KeyTab)
	if state := grid.State(); !state.Editing || state.CurrentRow != "alpha" ||
		state.CurrentColumn != "state" || state.EditText != "ReadyZ" {
		t.Fatalf("Shift-Tab reverse state = %#v", state)
	}
	if err := grid.SetRows(sampleDataGridRows()); err != nil {
		t.Fatalf("SetRows() error = %v", err)
	}
	if grid.State().Editing {
		t.Fatal("programmatic model replacement retained active editor")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(routed) != 2 || routed[0].ID != "grid.changed" || routed[1].ID != "grid.changed" {
		t.Fatalf("routed commands = %#v", routed)
	}
}

func TestDataGridSortedEditPreservesStableCurrentAndSelectionOrder(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Width: 24, Height: 6}},
		}},
		Columns: []Column{{Key: "value", Header: "Value", Editable: true, Sortable: true}},
		Rows: []TableRow{
			{Key: "alpha", Cells: []TableCell{{Column: "value", Text: "A"}}},
			{Key: "bravo", Cells: []TableCell{{Column: "value", Text: "B"}}},
		},
		CurrentRow: "alpha", CurrentColumn: "value",
		SelectionMode: CollectionSelectionMultiple,
		Selected:      []string{"alpha", "bravo"},
		SortColumn:    "value", SortDirection: SortAscending,
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	dispatchTableKey(t, app, "sorted-edit", KeyEnter)
	dispatchTextChord(t, app, "sorted-select-all", KeyControl, Key("a"))
	dispatchGridText(t, app, "sorted-replace", "Z")
	dispatchTableKey(t, app, "sorted-commit", KeyEnter)
	state := grid.State()
	if state.CurrentRow != "alpha" || state.CurrentRowIndex != 1 ||
		len(state.Selected) != 2 || state.Selected[0] != "bravo" ||
		state.Selected[1] != "alpha" || grid.Rows()[0].Cells[0].Text != "Z" {
		t.Fatalf("sorted edit state=%#v rows=%#v", state, grid.Rows())
	}
}

func TestDataGridSelectionPolicyCancelsEditor(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Width: 24, Height: 6}},
		}},
		Columns: []Column{{Key: "value", Header: "Value", Editable: true, Sortable: true}},
		Rows: []TableRow{
			{Key: "alpha", Cells: []TableCell{{Column: "value", Text: "A"}}},
			{Key: "bravo", Cells: []TableCell{{Column: "value", Text: "B"}}},
		},
		CurrentRow: "alpha", CurrentColumn: "value",
		SortColumn: "value", SortDirection: SortAscending,
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	dispatchTableKey(t, app, "begin-policy-edit", KeyEnter)
	if !grid.State().Editing {
		t.Fatal("Enter did not begin DataGrid edit")
	}
	if err := grid.SetSelectionPolicy(TableSelectionPolicy{
		Style: TableSelectionRange, Require: true,
		Anchor: "alpha", Extent: "bravo",
	}); err != nil {
		t.Fatalf("SetSelectionPolicy(range) error = %v", err)
	}
	state := grid.State()
	if state.Editing || state.SelectionStyle != TableSelectionRange ||
		state.RangeAnchor != "alpha" || state.RangeExtent != "bravo" ||
		!sameTableSelection(state.Selected, []string{"alpha", "bravo"}) {
		t.Fatalf("DataGrid range policy state = %#v", state)
	}
	dispatchTableKey(t, app, "begin-range-edit", KeyEnter)
	dispatchTextChord(t, app, "select-range-edit", KeyControl, Key("a"))
	dispatchGridText(t, app, "replace-range-edit", "Z")
	dispatchTableKey(t, app, "commit-range-edit", KeyEnter)
	if state = grid.State(); state.CurrentRow != "alpha" || state.CurrentRowIndex != 1 ||
		state.RangeAnchor != "alpha" || state.RangeExtent != "bravo" ||
		!sameTableSelection(state.Selected, []string{"bravo", "alpha"}) {
		t.Fatalf("sorted committed range state = %#v", state)
	}
}

func TestDataGridValidationAndTransactionAtomicity(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 36, Height: 8})
	if _, err := NewDataGrid(app.Root(), DataGridOptions{
		Columns: []Column{{Key: "value", Header: "Value", Editable: true, Validator: &TextValidator{
			Enforcement: TextValidationHard,
			Mode:        TextValidationWhitelist,
			Characters:  "a",
		}}},
		Rows: []TableRow{{Key: "row", Cells: []TableCell{{Column: "value", Text: "b"}}}},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("hard-invalid model error = %v", err)
	}
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		Columns: []Column{{Key: "value", Header: "Value", Editable: true}},
		Rows:    []TableRow{{Key: "row", Cells: []TableCell{{Column: "value", Text: "one"}}}},
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetDataGridRows(grid, []TableRow{{
		Key: "next", Cells: []TableCell{{Column: "value", Text: "two"}},
	}}); err != nil {
		t.Fatalf("SetDataGridRows() error = %v", err)
	}
	if err := tx.SetDataGridCurrent(grid, "missing", "value"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetDataGridCurrent(missing) error = %v", err)
	}
	if grid.Rows()[0].Key != "row" || app.Snapshot().Sequence != before {
		t.Fatal("failed DataGrid transaction leaked staged state")
	}
}

func TestDataGridConcurrentModelReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 32, Height: 8})
	columns := []Column{{Key: "value", Header: "Value", Editable: true, Sortable: true}}
	grid, err := NewDataGrid(app.Root(), DataGridOptions{
		Columns: columns,
		Rows: []TableRow{
			{Key: "a", Cells: []TableCell{{Column: "value", Text: "A"}}},
			{Key: "b", Cells: []TableCell{{Column: "value", Text: "B"}}},
		},
		CurrentRow: "a", CurrentColumn: "value", Selected: []string{"a"},
		RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewDataGrid() error = %v", err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 4)
	for worker := range 4 {
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
				if err := grid.Replace(
					columns, rows, first, "value", []string{first}, "", SortNone,
				); err != nil {
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
	state, rows := grid.State(), grid.Rows()
	if len(rows) != 2 || state.CurrentRow != rows[0].Key ||
		len(state.Selected) != 1 || state.Selected[0] != state.CurrentRow || state.Editing {
		t.Fatalf("concurrent final state=%#v rows=%#v", state, rows)
	}
}

func FuzzDataGridHardValidatedModel(f *testing.F) {
	for _, seed := range []string{"Ready", "blocked!", "e\u0301", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > MaxTextInputBytes {
			t.Skip()
		}
		columns, err := normalizeTableColumns([]Column{{
			Key: "value", Header: "Value", Editable: true,
			Validator: &TextValidator{
				Enforcement: TextValidationHard,
				Mode:        TextValidationBlacklist,
				Characters:  "!",
			},
		}})
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
			strings.Contains(rows[0].cells[0].cell.Text, "!") {
			t.Fatalf("hard-validated DataGrid row = %#v", rows)
		}
	})
}
