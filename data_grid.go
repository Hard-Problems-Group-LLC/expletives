package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// DataGridOptions configures one bounded editable tabular collection.
//
// DataGrid always uses cell focus. Its ScrollView ChangeCommand is routed for
// user selection changes and committed cell edits; programmatic mutations are
// silent. ContentAutomationKey, ContentStyle, and State.ContentSize are
// reserved and follow Table's contract.
type DataGridOptions struct {
	ScrollablePanelOptions
	Columns          []Column
	Rows             []TableRow
	CurrentRow       string
	CurrentColumn    string
	Selected         []string
	SelectionMode    CollectionSelectionMode
	RequireSelection bool
	SortColumn       string
	SortDirection    SortDirection
	Status           CollectionStatus
	StatusMessage    string
	ActivateCommand  CommandID
	SortCommand      CommandID
}

// DataGridState is one complete copied semantic, viewport, and editor state.
type DataGridState struct {
	TableState
	Editing    bool
	EditRow    string
	EditColumn string
	EditText   string
	EditCaret  int
	EditValid  bool
}

// DataGrid is a copy-safe focusable stable-identity editable table leaf.
type DataGrid struct{ controlHandle }

type dataGridBehavior struct {
	table      tableBehavior
	editor     textFieldBehavior
	editRow    string
	editColumn string
}

// NewDataGrid constructs and atomically inserts a DataGrid.
func NewDataGrid(parent Container, options DataGridOptions) (*DataGrid, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewDataGrid(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewDataGrid records construction of a provisional DataGrid.
func (t *Transaction) NewDataGrid(
	parent Container,
	options DataGridOptions,
) (*DataGrid, error) {
	behavior, err := newDataGridBehavior(options)
	if err != nil {
		return nil, err
	}
	behavior = reflowDataGrid(behavior, options.Bounds.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlDataGrid,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &DataGrid{controlHandle: controlHandle{state: state}}
	state.control = control
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func newDataGridBehavior(options DataGridOptions) (dataGridBehavior, error) {
	tableOptions := TableOptions{
		ScrollablePanelOptions: options.ScrollablePanelOptions,
		Columns:                options.Columns,
		Rows:                   options.Rows,
		CurrentRow:             options.CurrentRow,
		CurrentColumn:          options.CurrentColumn,
		Selected:               options.Selected,
		SelectionMode:          options.SelectionMode,
		RequireSelection:       options.RequireSelection,
		FocusMode:              TableFocusCell,
		SortColumn:             options.SortColumn,
		SortDirection:          options.SortDirection,
		Status:                 options.Status,
		StatusMessage:          options.StatusMessage,
		ActivateCommand:        options.ActivateCommand,
		SortCommand:            options.SortCommand,
	}
	if tableOptions.BorderStyle == "" {
		tableOptions.BorderStyle = "data_grid.border"
	}
	table, err := newTableBehavior(tableOptions)
	if err != nil {
		return dataGridBehavior{}, err
	}
	table.focusMode = TableFocusCell
	return dataGridBehavior{
		table:  table,
		editor: textFieldBehavior{selectionAnchor: -1},
	}, nil
}

func cloneDataGridBehavior(behavior dataGridBehavior) dataGridBehavior {
	cloned := behavior
	cloned.table = cloneTableBehavior(behavior.table)
	cloned.editor = cloneTextFieldBehavior(behavior.editor)
	return cloned
}

func dataGridBehaviorEqual(left, right dataGridBehavior) bool {
	return tableBehaviorEqual(left.table, right.table) &&
		textFieldBehaviorEqual(left.editor, right.editor) &&
		left.editRow == right.editRow && left.editColumn == right.editColumn
}

func (b dataGridBehavior) controlBorder() borderBehavior {
	return b.table.controlBorder()
}

func (b dataGridBehavior) clientInset() int { return b.table.clientInset() }

func (b dataGridBehavior) controlClientRect(bounds Rect) Rect {
	return b.table.controlClientRect(bounds)
}

func (b dataGridBehavior) intrinsicMinimum() Size {
	return b.table.intrinsicMinimum()
}

func (b dataGridBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.table.additionalStyles()...)
	return append(styles,
		"data_grid.edit",
		"data_grid.edit_focused",
		"data_grid.edit_invalid",
		"data_grid.edit_invalid_character",
	)
}

func (b dataGridBehavior) details() ControlDetails {
	details := b.table.scroll.border.details()
	details.Container = nil
	details.DataGrid = &DataGridDetails{}
	return details
}

func (b dataGridBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	b.table.scroll.paintDecoration(app, frame, state, absolute, clip)
	geometry := calculateScrollViewGeometry(absolute.Size(), b.table.scroll)
	viewport := translatedRect(geometry.viewport, absolute.X, absolute.Y)
	visible := viewport.Intersect(clip)
	if visible.Width <= 0 || visible.Height <= 0 {
		return
	}
	for y := visible.Y; y < visible.Y+visible.Height; y++ {
		var cells []tablePaintCell
		if y == viewport.Y {
			cells = b.table.headerCells(app.focus == state)
		} else {
			sourceRow := b.table.scroll.state.Offset.Y + y - viewport.Y - 1
			cells = b.bodyCells(sourceRow, app.focus == state)
		}
		for x := visible.X; x < visible.X+visible.Width; x++ {
			sourceX := b.table.scroll.state.Offset.X + x - viewport.X
			cell := tablePaintCell{grapheme: " ", style: "data_grid"}
			if sourceX >= 0 && sourceX < len(cells) {
				cell = cells[sourceX]
			}
			app.setCellLocked(frame, x, y, cell.grapheme, cell.style, app.styles[cell.style], state.id)
		}
	}
	if app.focus == state && b.editor.editing {
		b.paintCursor(app, state, viewport, clip)
	}
}

func (b dataGridBehavior) bodyCells(index int, focused bool) []tablePaintCell {
	cells := b.table.bodyCells(index, focused)
	if b.table.status != CollectionReady || index < 0 {
		return cells
	}
	order := tableDisplayOrder(b.table)
	if index >= len(order) {
		return cells
	}
	row := b.table.rows[order[index]]
	if row.row.Disabled {
		return cells
	}
	start := 4
	for columnIndex, column := range b.table.columns {
		width := b.table.columnWidths[columnIndex]
		if column.column.Editable {
			style := StyleID("data_grid.edit")
			current := row.row.Key == b.table.currentRow &&
				column.column.Key == b.table.currentColumn
			if focused && current {
				style = "data_grid.edit_focused"
			}
			for cellIndex := start; cellIndex < start+width && cellIndex < len(cells); cellIndex++ {
				cells[cellIndex].style = style
			}
			if b.editor.editing && row.row.Key == b.editRow &&
				column.column.Key == b.editColumn {
				b.paintEditorCells(cells, start, width)
			}
		}
		start += width + 1
	}
	return cells
}

func (b dataGridBehavior) paintEditorCells(cells []tablePaintCell, start, width int) {
	current := b.editor.current()
	invalid, valid := textValidationMask(current.cells, b.editor.validator)
	background := StyleID("data_grid.edit_focused")
	if !valid {
		background = "data_grid.edit_invalid"
	}
	for column := 0; column < width && start+column < len(cells); column++ {
		cells[start+column] = tablePaintCell{grapheme: " ", style: background}
		index := b.editor.viewOffset + column
		if index >= len(current.cells) {
			continue
		}
		style := background
		if !valid && invalid[index] {
			style = "data_grid.edit_invalid_character"
		} else if textSelectionContains(b.editor, index) {
			style = "text_input.focused_selection"
		}
		cells[start+column] = tablePaintCell{grapheme: current.cells[index], style: style}
	}
}

func (b dataGridBehavior) paintCursor(
	app *App,
	state *controlState,
	viewport Rect,
	clip Rect,
) {
	row := tableDisplayRowIndex(b.table, b.editRow)
	if row < 0 {
		return
	}
	start, _ := tableColumnInterval(b.table, b.editColumn)
	column := tableColumnIndex(b.table.columns, b.editColumn)
	if column < 0 {
		return
	}
	width := b.table.columnWidths[column]
	x := viewport.X + start - b.table.scroll.state.Offset.X +
		min(max(0, b.editor.caret-b.editor.viewOffset), max(0, width-1))
	y := viewport.Y + 1 + row - b.table.scroll.state.Offset.Y
	if width > 0 && x >= viewport.X && x < viewport.X+viewport.Width &&
		y > viewport.Y && y < viewport.Y+viewport.Height &&
		x >= clip.X && x < clip.X+clip.Width && y >= clip.Y && y < clip.Y+clip.Height {
		app.cursor = CursorState{Visible: true, Position: Point{X: x, Y: y}}
	}
}

func reflowDataGrid(behavior dataGridBehavior, size Size) dataGridBehavior {
	behavior.table = reflowTable(behavior.table, size)
	if behavior.editor.editing {
		column := tableColumnIndex(behavior.table.columns, behavior.editColumn)
		if column < 0 || behavior.editRow != behavior.table.currentRow ||
			behavior.editColumn != behavior.table.currentColumn {
			cancelDataGridEditBehavior(&behavior)
		} else {
			behavior.editor.viewOffset = textViewOffset(
				behavior.editor.viewOffset,
				behavior.editor.caret,
				len(behavior.editor.working.cells),
				behavior.table.columnWidths[column],
			)
		}
	}
	return behavior
}

func reconcileDataGridLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(dataGridBehavior)
	if !ok {
		return false
	}
	next := reflowDataGrid(behavior, state.bounds.Size())
	if dataGridBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func dataGridCanFocus(state *controlState, behavior dataGridBehavior) bool {
	return tableCanFocus(state, behavior.table)
}

func (a *App) dataGridDetailsLocked(
	state *controlState,
	behavior dataGridBehavior,
) DataGridDetails {
	details := DataGridDetails{
		Table:      tableDetails(state.bounds, behavior.table),
		Editing:    behavior.editor.editing,
		EditRow:    behavior.editRow,
		EditColumn: behavior.editColumn,
	}
	details.Table.RetainedBytes = dataGridStorageBytes(behavior)
	if behavior.editor.editing {
		copyState := *state
		if column := tableColumnIndex(behavior.table.columns, behavior.editColumn); column >= 0 {
			copyState.bounds.Width = behavior.table.columnWidths[column]
		}
		editor := a.textFieldDetailsLocked(&copyState, behavior.editor)
		details.Editor = &editor
	}
	return details
}

// Columns returns a caller-owned copy of the complete retained schema.
func (g *DataGrid) Columns() []Column {
	if g == nil || g.state == nil || g.state.app == nil {
		return nil
	}
	g.state.app.mu.RLock()
	defer g.state.app.mu.RUnlock()
	behavior, ok := g.state.behavior.(dataGridBehavior)
	if !ok || g.state.destroyed || g.state.aborted {
		return nil
	}
	return copyTableColumns(behavior.table.columns)
}

// Rows returns a caller-owned copy of the canonical unsorted row model.
func (g *DataGrid) Rows() []TableRow {
	if g == nil || g.state == nil || g.state.app == nil {
		return nil
	}
	g.state.app.mu.RLock()
	defer g.state.app.mu.RUnlock()
	behavior, ok := g.state.behavior.(dataGridBehavior)
	if !ok || g.state.destroyed || g.state.aborted {
		return nil
	}
	return copyTableRows(behavior.table.rows)
}

// State returns a caller-owned copy of complete DataGrid state.
func (g *DataGrid) State() DataGridState {
	if g == nil || g.state == nil || g.state.app == nil {
		return DataGridState{TableState: TableState{CurrentRowIndex: -1, CurrentColumnIndex: -1}}
	}
	g.state.app.mu.RLock()
	defer g.state.app.mu.RUnlock()
	behavior, ok := g.state.behavior.(dataGridBehavior)
	if !ok || g.state.destroyed || g.state.aborted {
		return DataGridState{TableState: TableState{CurrentRowIndex: -1, CurrentColumnIndex: -1}}
	}
	state := tableState(behavior.table)
	result := DataGridState{
		TableState: state,
		Editing:    behavior.editor.editing,
		EditRow:    behavior.editRow,
		EditColumn: behavior.editColumn,
		EditCaret:  behavior.editor.caret,
		EditValid:  true,
	}
	if behavior.editor.editing {
		result.EditText = behavior.editor.working.text
		result.EditValid = textCellsValid(behavior.editor.working.cells, behavior.editor.validator)
	}
	return result
}

func tableState(behavior tableBehavior) TableState {
	return TableState{
		Status: behavior.status, StatusMessage: behavior.statusMessage.text,
		CurrentRow:         behavior.currentRow,
		CurrentRowIndex:    tableDisplayRowIndex(behavior, behavior.currentRow),
		CurrentColumn:      behavior.currentColumn,
		CurrentColumnIndex: tableColumnIndex(behavior.columns, behavior.currentColumn),
		Selected:           append([]string(nil), behavior.selected...), FocusMode: behavior.focusMode,
		SortColumn: behavior.sortColumn, SortDirection: behavior.sortDirection,
		Offset: behavior.scroll.state.Offset, RowCount: len(behavior.rows),
		EnabledCount: enabledTableRowCount(behavior.rows), ColumnCount: len(behavior.columns),
		CellCount: tableCellCount(behavior.rows), ColumnWidths: append([]int(nil), behavior.columnWidths...),
	}
}

// SetRows replaces the canonical row model while preserving surviving state.
func (g *DataGrid) SetRows(rows []TableRow) error {
	return commitDataGridMutation(g, func(tx *Transaction) error { return tx.SetDataGridRows(g, rows) })
}

// SetModel replaces columns and rows while preserving surviving stable state.
func (g *DataGrid) SetModel(columns []Column, rows []TableRow) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.SetDataGridModel(g, columns, rows)
	})
}

// Replace atomically replaces the model and exact current, selection, and sort.
func (g *DataGrid) Replace(
	columns []Column,
	rows []TableRow,
	currentRow string,
	currentColumn string,
	selected []string,
	sortColumn string,
	sortDirection SortDirection,
) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.ReplaceDataGrid(
			g, columns, rows, currentRow, currentColumn, selected, sortColumn, sortDirection,
		)
	})
}

// SetCurrent changes current by stable enabled row and known column keys.
func (g *DataGrid) SetCurrent(row, column string) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.SetDataGridCurrent(g, row, column)
	})
}

// SetSelection changes the exact selected stable row keys.
func (g *DataGrid) SetSelection(selected []string) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.SetDataGridSelection(g, selected)
	})
}

// SetSort changes the optional single-column derived stable display order.
func (g *DataGrid) SetSort(column string, direction SortDirection) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.SetDataGridSort(g, column, direction)
	})
}

// SetStatus changes ready/loading/error presentation without discarding data.
func (g *DataGrid) SetStatus(status CollectionStatus, message string) error {
	return commitDataGridMutation(g, func(tx *Transaction) error {
		return tx.SetDataGridStatus(g, status, message)
	})
}

func commitDataGridMutation(control *DataGrid, record func(*Transaction) error) error {
	if control == nil || control.state == nil || control.state.app == nil {
		return ErrInvalidControl
	}
	tx := control.state.app.NewTransaction()
	if err := record(tx); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Focus atomically gives the DataGrid keyboard focus when it is ready.
func (g *DataGrid) Focus() error { return focusSelectionControl(g) }

// Activate begins editing the current editable cell or activates a read-only cell.
func (g *DataGrid) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if g == nil || g.state == nil || g.state.app == nil {
		return Completion{}, ErrInvalidControl
	}
	return g.state.app.activateDataGrid(ctx, source, requestID, g.state)
}

func (a *App) activateDataGrid(
	ctx context.Context,
	source string,
	requestID string,
	state *controlState,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) || !validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()
	a.mu.Lock()
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	if state == nil || state.app != a {
		a.mu.Unlock()
		return Completion{}, ErrInvalidControl
	}
	behavior, ok := state.behavior.(dataGridBehavior)
	if !ok || !dataGridCanFocus(state, behavior) ||
		!a.inActiveModalScopeLocked(state) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	if a.focus != state {
		_ = a.commitOrCancelEditorStateLocked(a.focus)
		a.focus = state
	}
	command, target, _, changed := a.dataGridKeyLocked(state, KeyEnter, false)
	result := CommandResult{Outcome: OutcomeNoOp}
	var router CommandRouter
	var execute bool
	if changed {
		result.Outcome = OutcomeApplied
	}
	if command != "" {
		router, result, execute = a.resolveCommandLocked(command, true)
	}
	a.mu.Unlock()
	if execute {
		result = a.callRouter(ctx, router, Command{ID: command, Target: target, Source: source})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	return a.associateLocked(requestID, result, command), nil
}

func (a *App) dataGridEditorInputLocked(
	state *controlState,
	behavior dataGridBehavior,
	key Key,
	held map[Key]bool,
) (CommandID, ControlID, bool, bool) {
	if state == nil || state != a.focus || !dataGridCanFocus(state, behavior) {
		return "", "", false, false
	}
	if !behavior.editor.editing {
		if noHeldModifiers(held) && (key == KeyEnter || key == KeyF2) {
			if beginDataGridEditBehavior(&behavior) &&
				a.dataGridWithinBudgetLocked(state, behavior) {
				behavior = reflowDataGrid(behavior, state.bounds.Size())
				state.behavior = behavior
				return "", "", true, true
			}
			return "", "", key == KeyF2, false
		}
		return "", "", false, false
	}
	if held[KeyAlt] || held[KeyMeta] {
		return "", "", false, false
	}
	if key == KeyEnter && noHeldModifiers(held) {
		command, changed, valid := a.commitDataGridEditLocked(state, &behavior)
		if !valid {
			return "", "", true, false
		}
		state.behavior = behavior
		return command, state.id, true, changed
	}
	if key == KeyEscape && noHeldModifiers(held) {
		cancelDataGridEditBehavior(&behavior)
		state.behavior = behavior
		return "", "", true, true
	}
	copyState := *state
	column := tableColumnIndex(behavior.table.columns, behavior.editColumn)
	if column >= 0 {
		copyState.bounds.Width = behavior.table.columnWidths[column]
	}
	editor, _, _, handled, changed := a.textEditorInputLocked(
		&copyState,
		behavior.editor,
		key,
		held,
	)
	if !handled {
		return "", "", false, false
	}
	if changed {
		candidate := behavior
		candidate.editor = editor
		if !a.dataGridWithinBudgetLocked(state, candidate) {
			return "", "", true, false
		}
		behavior = candidate
		state.behavior = behavior
	}
	return "", "", true, changed
}

func (a *App) dataGridKeyLocked(
	state *controlState,
	key Key,
	control bool,
) (CommandID, ControlID, bool, bool) {
	if state == nil {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(dataGridBehavior)
	if !ok || behavior.editor.editing || !dataGridCanFocus(state, behavior) {
		return "", "", false, false
	}
	if !control && key == KeyEnter {
		column := tableColumnIndex(behavior.table.columns, behavior.table.currentColumn)
		if column >= 0 && behavior.table.columns[column].column.Editable {
			if beginDataGridEditBehavior(&behavior) &&
				a.dataGridWithinBudgetLocked(state, behavior) {
				behavior = reflowDataGrid(behavior, state.bounds.Size())
				state.behavior = behavior
				return "", state.id, true, true
			}
			return "", state.id, true, false
		}
	}
	state.behavior = behavior.table
	command, target, handled, changed := a.tableKeyLocked(state, key, control)
	if table, valid := state.behavior.(tableBehavior); valid {
		behavior.table = table
	}
	state.behavior = behavior
	return command, target, handled, changed
}

func beginDataGridEditBehavior(behavior *dataGridBehavior) bool {
	if behavior == nil || behavior.editor.editing || behavior.table.status != CollectionReady {
		return false
	}
	rowIndex := tableCanonicalRowIndex(behavior.table.rows, behavior.table.currentRow)
	columnIndex := tableColumnIndex(behavior.table.columns, behavior.table.currentColumn)
	if rowIndex < 0 || behavior.table.rows[rowIndex].row.Disabled || columnIndex < 0 ||
		!behavior.table.columns[columnIndex].column.Editable {
		return false
	}
	text := ""
	if cell, found := tableCellForColumn(
		behavior.table.rows[rowIndex],
		behavior.table.currentColumn,
	); found {
		text = cell.cell.Text
	}
	value, err := normalizeInputText(text)
	if err != nil {
		return false
	}
	behavior.editor = textFieldBehavior{
		committed: value, working: cloneInputText(value),
		validator:       cloneTextValidator(behavior.table.columns[columnIndex].validator),
		changeCommand:   behavior.table.changeCommand,
		editing:         true,
		caret:           len(value.cells),
		selectionAnchor: -1,
	}
	behavior.editRow = behavior.table.currentRow
	behavior.editColumn = behavior.table.currentColumn
	return true
}

func cancelDataGridEditBehavior(behavior *dataGridBehavior) bool {
	if behavior == nil || !behavior.editor.editing {
		return false
	}
	behavior.editor = textFieldBehavior{selectionAnchor: -1}
	behavior.editRow = ""
	behavior.editColumn = ""
	return true
}

func (a *App) commitDataGridEditLocked(
	state *controlState,
	behavior *dataGridBehavior,
) (CommandID, bool, bool) {
	if behavior == nil || !behavior.editor.editing {
		return "", false, true
	}
	if !textCellsValid(behavior.editor.working.cells, behavior.editor.validator) {
		return "", false, false
	}
	changed := behavior.editor.committed.text != behavior.editor.working.text
	if changed {
		candidate := cloneDataGridBehavior(*behavior)
		if err := setDataGridCellText(
			&candidate.table,
			candidate.editRow,
			candidate.editColumn,
			candidate.editor.working.text,
		); err != nil {
			return "", false, false
		}
		cancelDataGridEditBehavior(&candidate)
		candidate = reflowDataGrid(candidate, state.bounds.Size())
		if !a.dataGridWithinBudgetLocked(state, candidate) {
			return "", false, false
		}
		*behavior = candidate
		return behavior.table.changeCommand, true, true
	}
	cancelDataGridEditBehavior(behavior)
	*behavior = reflowDataGrid(*behavior, state.bounds.Size())
	return "", true, true
}

func setDataGridCellText(table *tableBehavior, rowKey, columnKey, value string) error {
	rowIndex := tableCanonicalRowIndex(table.rows, rowKey)
	columnIndex := tableColumnIndex(table.columns, columnKey)
	if rowIndex < 0 || columnIndex < 0 || table.rows[rowIndex].row.Disabled ||
		!table.columns[columnIndex].column.Editable {
		return ErrInvalidControl
	}
	normalized, err := normalizeDisplayText(value, false)
	if err != nil {
		return err
	}
	validator := table.columns[columnIndex].validator
	if validator != nil && !textCellsValid(normalized.lines[0], validator) {
		return fmt.Errorf("%w: invalid DataGrid cell value", ErrValidation)
	}
	cells := make(map[string]TableCell, len(table.rows[rowIndex].row.Cells)+1)
	for _, cell := range table.rows[rowIndex].row.Cells {
		cells[cell.Column] = cell
	}
	cells[columnKey] = TableCell{Column: columnKey, Text: normalized.text}
	rebuilt := make([]TableCell, 0, len(cells))
	for _, column := range table.columns {
		if cell, found := cells[column.column.Key]; found {
			rebuilt = append(rebuilt, cell)
		}
	}
	table.rows[rowIndex].row.Cells = rebuilt
	normalizedRows, err := normalizeTableRows(table.columns, copyTableRows(table.rows))
	if err != nil {
		return err
	}
	table.rows = normalizedRows
	table.displayOrder = nil
	table.selected, _ = normalizeTableSelection(table, table.selected)
	return nil
}

func (a *App) applyDataGridTextInputLocked(
	state *controlState,
	behavior dataGridBehavior,
	text string,
) CommandResult {
	if state == nil || !behavior.editor.editing {
		return CommandResult{Outcome: OutcomeNoOp}
	}
	inserted, err := normalizeInputText(text)
	if err != nil {
		return rejectedTextInput(err)
	}
	inserted.cells = filterTextInputCells(inserted.cells, behavior.editor.validator, false)
	candidateCells, caret, ok := replaceTextInputCells(
		behavior.editor.working.cells,
		behavior.editor.caret,
		behavior.editor.selectionAnchor,
		inserted.cells,
	)
	if !ok {
		return textInputCapacityResult()
	}
	if candidateCells == nil {
		return CommandResult{Outcome: OutcomeNoOp}
	}
	candidate := behavior
	candidate.editor.working.cells = candidateCells
	candidate.editor.working.text = strings.Join(candidateCells, "")
	candidate.editor.caret = caret
	candidate.editor.selectionAnchor = -1
	column := tableColumnIndex(candidate.table.columns, candidate.editColumn)
	width := state.bounds.Width
	if column >= 0 {
		width = candidate.table.columnWidths[column]
	}
	candidate.editor.viewOffset = textViewOffset(
		candidate.editor.viewOffset,
		candidate.editor.caret,
		len(candidateCells),
		width,
	)
	if !a.dataGridWithinBudgetLocked(state, candidate) {
		return textInputCapacityResult()
	}
	state.behavior = candidate
	return CommandResult{Outcome: OutcomeApplied}
}

func (a *App) dataGridWithinBudgetLocked(
	target *controlState,
	candidate dataGridBehavior,
) bool {
	collectionBytes, collectionCells := 0, 0
	for _, state := range a.controlsByID {
		if state == nil || state.destroyed || state.aborted {
			continue
		}
		behavior := state.behavior
		if state == target {
			behavior = candidate
		}
		switch value := behavior.(type) {
		case listBoxBehavior:
			collectionBytes += listBoxStorageBytes(value)
		case treeViewBehavior:
			collectionBytes += treeViewStorageBytes(value)
		case tableBehavior:
			collectionBytes += tableStorageBytes(value)
			collectionCells += tableCellCount(value.rows)
		case dataGridBehavior:
			collectionBytes += dataGridStorageBytes(value)
			collectionCells += tableCellCount(value.table.rows)
		case dropDownBehavior:
			collectionBytes += popupCollectionStorageBytes(value.popup)
		case comboBoxBehavior:
			collectionBytes += comboBoxStorageBytes(value)
		}
		if collectionBytes > MaxCollectionAggregateBytes || collectionCells > MaxCollectionCells {
			return false
		}
	}
	return true
}

func dataGridStorageBytes(behavior dataGridBehavior) int {
	total := tableStorageBytes(behavior.table)
	if behavior.editor.editing {
		total += len(behavior.editRow) + len(behavior.editColumn) + len(behavior.editor.working.text)
	}
	return total
}

func nextEditableDataGridCell(
	behavior dataGridBehavior,
	reverse bool,
) (string, string, bool) {
	order := tableDisplayOrder(behavior.table)
	currentRow := tableDisplayRowIndex(behavior.table, behavior.table.currentRow)
	currentColumn := tableColumnIndex(behavior.table.columns, behavior.table.currentColumn)
	if currentRow < 0 || currentColumn < 0 {
		return "", "", false
	}
	totalColumns := len(behavior.table.columns)
	current := currentRow*totalColumns + currentColumn
	delta, end := 1, len(order)*totalColumns
	if reverse {
		delta, end = -1, -1
	}
	for position := current + delta; position != end; position += delta {
		rowPosition := position / totalColumns
		columnPosition := position % totalColumns
		if rowPosition < 0 || rowPosition >= len(order) || columnPosition < 0 {
			break
		}
		row := behavior.table.rows[order[rowPosition]]
		column := behavior.table.columns[columnPosition]
		if !row.row.Disabled && column.column.Editable {
			return row.row.Key, column.column.Key, true
		}
	}
	return "", "", false
}

func (t *Transaction) selectedDataGrid(
	control *DataGrid,
) (*controlState, dataGridBehavior, error) {
	target, err := t.control(control)
	if err != nil {
		return nil, dataGridBehavior{}, err
	}
	behavior, ok := t.selectedControlBehavior(target).(dataGridBehavior)
	if !ok {
		return nil, dataGridBehavior{}, ErrInvalidControl
	}
	return target, cloneDataGridBehavior(behavior), nil
}

func (t *Transaction) recordDataGrid(
	state *controlState,
	behavior dataGridBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationDataGrid, state: state, behavior: behavior,
	})
	return nil
}

// SetDataGridRows records a copied row replacement preserving stable state.
func (t *Transaction) SetDataGridRows(control *DataGrid, rows []TableRow) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	normalized, err := normalizeTableRows(behavior.table.columns, rows)
	if err != nil {
		return err
	}
	oldIndex := tableDisplayRowIndex(behavior.table, behavior.table.currentRow)
	behavior.table.rows = normalized
	behavior.table.displayOrder = nil
	repairTableIdentity(&behavior.table, oldIndex)
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// SetDataGridModel records a copied schema/model replacement preserving state.
func (t *Transaction) SetDataGridModel(
	control *DataGrid,
	columns []Column,
	rows []TableRow,
) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	oldIndex := tableDisplayRowIndex(behavior.table, behavior.table.currentRow)
	normalizedColumns, err := normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	normalizedRows, err := normalizeTableRows(normalizedColumns, rows)
	if err != nil {
		return err
	}
	behavior.table.columns, behavior.table.rows = normalizedColumns, normalizedRows
	behavior.table.displayOrder = nil
	if index := tableColumnIndex(behavior.table.columns, behavior.table.sortColumn); index < 0 ||
		(behavior.table.sortColumn != "" && !behavior.table.columns[index].column.Sortable) {
		behavior.table.sortColumn, behavior.table.sortDirection = "", SortNone
	}
	if tableColumnIndex(behavior.table.columns, behavior.table.currentColumn) < 0 {
		behavior.table.currentColumn = ""
		if len(behavior.table.columns) > 0 {
			behavior.table.currentColumn = behavior.table.columns[0].column.Key
		}
	}
	repairTableIdentity(&behavior.table, oldIndex)
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// ReplaceDataGrid records one exact complete DataGrid replacement.
func (t *Transaction) ReplaceDataGrid(
	control *DataGrid,
	columns []Column,
	rows []TableRow,
	currentRow string,
	currentColumn string,
	selected []string,
	sortColumn string,
	sortDirection SortDirection,
) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	behavior.table.columns, err = normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	behavior.table.rows, err = normalizeTableRows(behavior.table.columns, rows)
	if err != nil {
		return err
	}
	behavior.table.displayOrder = nil
	if err := setExactTableSort(&behavior.table, sortColumn, sortDirection); err != nil {
		return err
	}
	if err := setExactTableIdentity(
		&behavior.table,
		currentRow,
		currentColumn,
		selected,
	); err != nil {
		return err
	}
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// SetDataGridCurrent records exact stable current row and column keys.
func (t *Transaction) SetDataGridCurrent(control *DataGrid, row, column string) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	if row == "" {
		if firstEnabledTableRowKey(&behavior.table) != "" {
			return fmt.Errorf("%w: ready DataGrid requires current row", ErrValidation)
		}
	} else if index := tableCanonicalRowIndex(behavior.table.rows, row); index < 0 ||
		behavior.table.rows[index].row.Disabled {
		return fmt.Errorf("%w: invalid DataGrid current row", ErrValidation)
	}
	if column == "" {
		if len(behavior.table.columns) > 0 {
			return fmt.Errorf("%w: DataGrid requires current column", ErrValidation)
		}
	} else if tableColumnIndex(behavior.table.columns, column) < 0 {
		return fmt.Errorf("%w: invalid DataGrid current column", ErrValidation)
	}
	behavior.table.currentRow, behavior.table.currentColumn = row, column
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// SetDataGridSelection records one exact stable-row selection.
func (t *Transaction) SetDataGridSelection(control *DataGrid, selected []string) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	ordered, err := normalizeTableSelection(&behavior.table, selected)
	if err != nil {
		return err
	}
	if behavior.table.requireSelection && len(ordered) == 0 && behavior.table.currentRow != "" {
		return fmt.Errorf("%w: DataGrid selection is required", ErrValidation)
	}
	behavior.table.selected = ordered
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// SetDataGridSort records an exact optional single-column sort.
func (t *Transaction) SetDataGridSort(
	control *DataGrid,
	column string,
	direction SortDirection,
) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	if err := setExactTableSort(&behavior.table, column, direction); err != nil {
		return err
	}
	behavior.table.displayOrder = nil
	behavior.table.selected, _ = normalizeTableSelection(
		&behavior.table,
		behavior.table.selected,
	)
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}

// SetDataGridStatus records ready/loading/error presentation state.
func (t *Transaction) SetDataGridStatus(
	control *DataGrid,
	status CollectionStatus,
	message string,
) error {
	target, behavior, err := t.selectedDataGrid(control)
	if err != nil {
		return err
	}
	cancelDataGridEditBehavior(&behavior)
	status, normalized, err := normalizeCollectionStatus(status, message)
	if err != nil {
		return err
	}
	behavior.table.status, behavior.table.statusMessage = status, normalized
	behavior = reflowDataGrid(behavior, target.bounds.Size())
	return t.recordDataGrid(target, behavior)
}
