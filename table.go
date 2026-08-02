package expletives

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Column is one copied stable-identity Table or DataGrid column.
type Column struct {
	Key          string
	Header       string
	Width        int
	MinimumWidth int
	MaximumWidth int
	Grow         int
	Alignment    TextAlignment
	Sortable     bool
	Editable     bool
	Validator    *TextValidator
}

// TableCell is one copied cell associated with a stable column key.
type TableCell struct {
	Column string
	Text   string
}

// TableRow is one copied stable-identity Table or DataGrid row.
type TableRow struct {
	Key            string
	Cells          []TableCell
	Disabled       bool
	DisabledReason string
}

// TableFocusMode selects row-current or row-and-column-current navigation.
type TableFocusMode string

const (
	TableFocusRow  TableFocusMode = "row"
	TableFocusCell TableFocusMode = "cell"
)

// SortDirection selects the derived stable display order of table rows.
type SortDirection string

const (
	SortNone       SortDirection = "none"
	SortAscending  SortDirection = "ascending"
	SortDescending SortDirection = "descending"
)

// TableOptions configures one bounded read-only tabular collection.
//
// ContentAutomationKey, ContentStyle, and State.ContentSize are reserved by
// Table and must be empty. State.Offset is accepted as an initial request and
// clamped while keeping the current row and, in cell mode, column visible.
type TableOptions struct {
	ScrollablePanelOptions
	Columns          []Column
	Rows             []TableRow
	CurrentRow       string
	CurrentColumn    string
	Selected         []string
	SelectionMode    CollectionSelectionMode
	RequireSelection bool
	FocusMode        TableFocusMode
	SortColumn       string
	SortDirection    SortDirection
	Status           CollectionStatus
	StatusMessage    string
	ActivateCommand  CommandID
	SortCommand      CommandID
}

// TableState is one complete copied semantic and viewport state.
type TableState struct {
	Status             CollectionStatus
	StatusMessage      string
	CurrentRow         string
	CurrentRowIndex    int
	CurrentColumn      string
	CurrentColumnIndex int
	Selected           []string
	FocusMode          TableFocusMode
	SortColumn         string
	SortDirection      SortDirection
	Offset             Point
	RowCount           int
	EnabledCount       int
	ColumnCount        int
	CellCount          int
	ColumnWidths       []int
}

// Table is a copy-safe focusable stable-identity read-only table leaf.
type Table struct{ controlHandle }

type normalizedTableColumn struct {
	column    Column
	header    normalizedDisplayText
	validator *normalizedTextValidator
}

type normalizedTableCell struct {
	cell TableCell
	text normalizedDisplayText
}

type normalizedTableRow struct {
	row   TableRow
	cells []normalizedTableCell
}

type tableBehavior struct {
	scroll           scrollViewBehavior
	columns          []normalizedTableColumn
	rows             []normalizedTableRow
	displayOrder     []int
	columnWidths     []int
	currentRow       string
	currentColumn    string
	selected         []string
	selectionMode    CollectionSelectionMode
	requireSelection bool
	focusMode        TableFocusMode
	sortColumn       string
	sortDirection    SortDirection
	status           CollectionStatus
	statusMessage    normalizedDisplayText
	changeCommand    CommandID
	activateCommand  CommandID
	sortCommand      CommandID
}

type tablePaintCell struct {
	grapheme string
	style    StyleID
}

// NewTable constructs and atomically inserts a Table.
func NewTable(parent Container, options TableOptions) (*Table, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewTable(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewTable records construction of a provisional Table.
func (t *Transaction) NewTable(
	parent Container,
	options TableOptions,
) (*Table, error) {
	behavior, err := newTableBehavior(options)
	if err != nil {
		return nil, err
	}
	behavior = reflowTable(behavior, options.Bounds.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlTable,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &Table{controlHandle: controlHandle{state: state}}
	state.control = control
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func newTableBehavior(options TableOptions) (tableBehavior, error) {
	scrollOptions := options.ScrollViewOptions
	if scrollOptions.ContentAutomationKey != "" ||
		scrollOptions.ContentStyle != "" ||
		scrollOptions.State.ContentSize != (Size{}) {
		return tableBehavior{}, fmt.Errorf(
			"%w: Table derives its content identity, style, and extent",
			ErrValidation,
		)
	}
	requestedOffset := scrollOptions.State.Offset
	if requestedOffset.X < 0 || requestedOffset.Y < 0 ||
		requestedOffset.X > maxCoordinateMagnitude ||
		requestedOffset.Y > maxCoordinateMagnitude {
		return tableBehavior{}, fmt.Errorf("%w: invalid Table offset", ErrValidation)
	}
	for _, command := range []CommandID{
		scrollOptions.ChangeCommand,
		options.ActivateCommand,
		options.SortCommand,
	} {
		if err := validateOptionalCommand(command); err != nil {
			return tableBehavior{}, err
		}
	}
	scrollOptions.State.Offset = Point{}
	scroll, err := normalizeScrollViewBehavior(scrollViewBehavior{
		state:            scrollOptions.State,
		arrowStep:        scrollOptions.ArrowStep,
		pageStep:         scrollOptions.PageStep,
		disabled:         scrollOptions.Disabled,
		disabledReason:   scrollOptions.DisabledReason,
		horizontalPolicy: options.HorizontalBar,
		verticalPolicy:   options.VerticalBar,
		integratedBars:   true,
	})
	if err != nil {
		return tableBehavior{}, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		ControlTable,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return tableBehavior{}, err
	}
	scroll.border = border
	scroll.state.Offset = requestedOffset
	mode, err := normalizeCollectionSelectionMode(options.SelectionMode)
	if err != nil {
		return tableBehavior{}, err
	}
	focusMode, err := normalizeTableFocusMode(options.FocusMode)
	if err != nil {
		return tableBehavior{}, err
	}
	status, message, err := normalizeCollectionStatus(options.Status, options.StatusMessage)
	if err != nil {
		return tableBehavior{}, err
	}
	columns, err := normalizeTableColumns(options.Columns)
	if err != nil {
		return tableBehavior{}, err
	}
	rows, err := normalizeTableRows(columns, options.Rows)
	if err != nil {
		return tableBehavior{}, err
	}
	behavior := tableBehavior{
		scroll:           scroll,
		columns:          columns,
		rows:             rows,
		selectionMode:    mode,
		requireSelection: options.RequireSelection,
		focusMode:        focusMode,
		status:           status,
		statusMessage:    message,
		changeCommand:    scrollOptions.ChangeCommand,
		activateCommand:  options.ActivateCommand,
		sortCommand:      options.SortCommand,
	}
	if err := setExactTableSort(&behavior, options.SortColumn, options.SortDirection); err != nil {
		return tableBehavior{}, err
	}
	if err := setExactTableIdentity(
		&behavior,
		options.CurrentRow,
		options.CurrentColumn,
		options.Selected,
	); err != nil {
		return tableBehavior{}, err
	}
	return behavior, nil
}

func normalizeTableFocusMode(mode TableFocusMode) (TableFocusMode, error) {
	if mode == "" {
		mode = TableFocusRow
	}
	switch mode {
	case TableFocusRow, TableFocusCell:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: invalid Table focus mode %q", ErrValidation, mode)
	}
}

func normalizeTableSortDirection(direction SortDirection) (SortDirection, error) {
	if direction == "" {
		direction = SortNone
	}
	switch direction {
	case SortNone, SortAscending, SortDescending:
		return direction, nil
	default:
		return "", fmt.Errorf("%w: invalid Table sort direction %q", ErrValidation, direction)
	}
}

func normalizeTableColumns(columns []Column) ([]normalizedTableColumn, error) {
	if len(columns) > MaxCollectionColumns {
		return nil, fmt.Errorf("%w: Table exceeds %d columns", ErrControlCapacity, MaxCollectionColumns)
	}
	result := make([]normalizedTableColumn, len(columns))
	seen := make(map[string]bool, len(columns))
	for index, column := range columns {
		if !validBoundedIdentifier(column.Key) || seen[column.Key] {
			return nil, fmt.Errorf("%w: invalid or duplicate Table column key %q", ErrValidation, column.Key)
		}
		seen[column.Key] = true
		header, err := normalizeDisplayText(column.Header, false)
		if err != nil || header.cells == 0 {
			return nil, fmt.Errorf("%w: invalid Table header for %q", ErrValidation, column.Key)
		}
		if column.Width < 0 || column.MinimumWidth < 0 || column.MaximumWidth < 0 ||
			column.Grow < 0 || column.Width > maxCoordinateMagnitude ||
			column.MinimumWidth > maxCoordinateMagnitude ||
			column.MaximumWidth > maxCoordinateMagnitude ||
			column.Grow > maxCoordinateMagnitude ||
			(column.MaximumWidth > 0 && column.MinimumWidth > column.MaximumWidth) {
			return nil, fmt.Errorf("%w: invalid Table width policy for %q", ErrValidation, column.Key)
		}
		alignment, err := normalizeTextAlignment(column.Alignment)
		if err != nil {
			return nil, err
		}
		validator, err := normalizeTextValidator(column.Validator)
		if err != nil {
			return nil, fmt.Errorf("%w: Table column %q validator: %v", ErrValidation, column.Key, err)
		}
		if validator != nil && !column.Editable {
			return nil, fmt.Errorf("%w: Table column %q validator requires Editable", ErrValidation, column.Key)
		}
		column.Header = header.text
		column.Alignment = alignment
		if validator != nil {
			value := validator.value
			column.Validator = &value
		} else {
			column.Validator = nil
		}
		result[index] = normalizedTableColumn{column: column, header: header, validator: validator}
	}
	return result, nil
}

func normalizeTableRows(
	columns []normalizedTableColumn,
	rows []TableRow,
) ([]normalizedTableRow, error) {
	if len(rows) > MaxCollectionItems {
		return nil, fmt.Errorf("%w: Table exceeds %d rows", ErrControlCapacity, MaxCollectionItems)
	}
	columnIndex := make(map[string]int, len(columns))
	for index, column := range columns {
		columnIndex[column.column.Key] = index
	}
	result := make([]normalizedTableRow, len(rows))
	seenRows := make(map[string]bool, len(rows))
	cellCount := 0
	for rowIndex, row := range rows {
		if !validBoundedIdentifier(row.Key) || seenRows[row.Key] {
			return nil, fmt.Errorf("%w: invalid or duplicate Table row key %q", ErrValidation, row.Key)
		}
		seenRows[row.Key] = true
		reason, err := normalizeDisabledReason(row.Disabled, row.DisabledReason)
		if err != nil {
			return nil, err
		}
		if len(row.Cells) > len(columns) {
			return nil, fmt.Errorf("%w: Table row %q has too many cells", ErrValidation, row.Key)
		}
		cellCount += len(row.Cells)
		if cellCount > MaxCollectionCells {
			return nil, fmt.Errorf("%w: Table exceeds %d cells", ErrControlCapacity, MaxCollectionCells)
		}
		seenCells := make(map[string]bool, len(row.Cells))
		cellsByColumn := make(map[int]normalizedTableCell, len(row.Cells))
		for _, cell := range row.Cells {
			columnPosition, found := columnIndex[cell.Column]
			if !found || seenCells[cell.Column] {
				return nil, fmt.Errorf("%w: unknown or duplicate Table cell column %q", ErrValidation, cell.Column)
			}
			seenCells[cell.Column] = true
			text, err := normalizeDisplayText(cell.Text, false)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid Table cell %q/%q", ErrValidation, row.Key, cell.Column)
			}
			validator := columns[columnPosition].validator
			if validator != nil && validator.value.Enforcement == TextValidationHard &&
				!textCellsValid(text.lines[0], validator) {
				return nil, fmt.Errorf("%w: Table cell %q/%q violates hard validation", ErrValidation, row.Key, cell.Column)
			}
			cell.Text = text.text
			cellsByColumn[columnPosition] = normalizedTableCell{cell: cell, text: text}
		}
		ordered := make([]normalizedTableCell, 0, len(cellsByColumn))
		copiedCells := make([]TableCell, 0, len(cellsByColumn))
		for columnPosition := range columns {
			if cell, found := cellsByColumn[columnPosition]; found {
				ordered = append(ordered, cell)
				copiedCells = append(copiedCells, cell.cell)
			}
		}
		row.Cells = copiedCells
		row.DisabledReason = reason
		result[rowIndex] = normalizedTableRow{row: row, cells: ordered}
	}
	return result, nil
}

func setExactTableSort(behavior *tableBehavior, column string, direction SortDirection) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	normalized, err := normalizeTableSortDirection(direction)
	if err != nil {
		return err
	}
	if normalized == SortNone {
		if column != "" {
			return fmt.Errorf("%w: unsorted Table cannot name a sort column", ErrValidation)
		}
		behavior.sortColumn = ""
		behavior.sortDirection = SortNone
		return nil
	}
	index := tableColumnIndex(behavior.columns, column)
	if index < 0 || !behavior.columns[index].column.Sortable {
		return fmt.Errorf("%w: Table sort column is unavailable", ErrValidation)
	}
	behavior.sortColumn = column
	behavior.sortDirection = normalized
	return nil
}

func setExactTableIdentity(
	behavior *tableBehavior,
	currentRow string,
	currentColumn string,
	selected []string,
) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	if currentRow == "" {
		currentRow = firstEnabledTableRowKey(behavior)
	} else if index := tableCanonicalRowIndex(behavior.rows, currentRow); index < 0 ||
		behavior.rows[index].row.Disabled {
		return fmt.Errorf("%w: invalid Table current row", ErrValidation)
	}
	if currentColumn == "" {
		if len(behavior.columns) > 0 {
			currentColumn = behavior.columns[0].column.Key
		}
	} else if tableColumnIndex(behavior.columns, currentColumn) < 0 {
		return fmt.Errorf("%w: invalid Table current column", ErrValidation)
	}
	ordered, err := normalizeTableSelection(behavior, selected)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(ordered) == 0 && currentRow != "" {
		ordered = []string{currentRow}
	}
	behavior.currentRow = currentRow
	behavior.currentColumn = currentColumn
	behavior.selected = ordered
	return nil
}

func normalizeTableSelection(behavior *tableBehavior, selected []string) ([]string, error) {
	if behavior == nil {
		return nil, ErrInvalidControl
	}
	if behavior.selectionMode == CollectionSelectionSingle && len(selected) > 1 {
		return nil, fmt.Errorf("%w: single-selection Table has multiple selected rows", ErrValidation)
	}
	requested := make(map[string]bool, len(selected))
	for _, key := range selected {
		if !validBoundedIdentifier(key) || requested[key] {
			return nil, fmt.Errorf("%w: invalid or duplicate Table selection", ErrValidation)
		}
		index := tableCanonicalRowIndex(behavior.rows, key)
		if index < 0 || behavior.rows[index].row.Disabled {
			return nil, fmt.Errorf("%w: Table selection identifies an unavailable row", ErrValidation)
		}
		requested[key] = true
	}
	result := make([]string, 0, len(selected))
	for _, rowIndex := range tableDisplayOrder(*behavior) {
		key := behavior.rows[rowIndex].row.Key
		if requested[key] {
			result = append(result, key)
		}
	}
	return result, nil
}

func tableCanonicalRowIndex(rows []normalizedTableRow, key string) int {
	for index, row := range rows {
		if row.row.Key == key {
			return index
		}
	}
	return -1
}

func tableColumnIndex(columns []normalizedTableColumn, key string) int {
	for index, column := range columns {
		if column.column.Key == key {
			return index
		}
	}
	return -1
}

func tableCellForColumn(row normalizedTableRow, column string) (normalizedTableCell, bool) {
	for _, cell := range row.cells {
		if cell.cell.Column == column {
			return cell, true
		}
	}
	return normalizedTableCell{}, false
}

func deriveTableDisplayOrder(behavior tableBehavior) []int {
	order := make([]int, len(behavior.rows))
	for index := range order {
		order[index] = index
	}
	if behavior.sortColumn == "" || behavior.sortDirection == SortNone {
		return order
	}
	sort.SliceStable(order, func(left, right int) bool {
		leftCell, _ := tableCellForColumn(behavior.rows[order[left]], behavior.sortColumn)
		rightCell, _ := tableCellForColumn(behavior.rows[order[right]], behavior.sortColumn)
		comparison := strings.Compare(leftCell.text.text, rightCell.text.text)
		if behavior.sortDirection == SortDescending {
			return comparison > 0
		}
		return comparison < 0
	})
	return order
}

func tableDisplayOrder(behavior tableBehavior) []int {
	if len(behavior.displayOrder) == len(behavior.rows) {
		return behavior.displayOrder
	}
	return deriveTableDisplayOrder(behavior)
}

func tableDisplayRowIndex(behavior tableBehavior, key string) int {
	for displayIndex, canonicalIndex := range tableDisplayOrder(behavior) {
		if behavior.rows[canonicalIndex].row.Key == key {
			return displayIndex
		}
	}
	return -1
}

func firstEnabledTableRowKey(behavior *tableBehavior) string {
	if behavior == nil {
		return ""
	}
	for _, canonicalIndex := range tableDisplayOrder(*behavior) {
		if !behavior.rows[canonicalIndex].row.Disabled {
			return behavior.rows[canonicalIndex].row.Key
		}
	}
	return ""
}

func lastEnabledTableRowKey(behavior tableBehavior) string {
	order := tableDisplayOrder(behavior)
	for index := len(order) - 1; index >= 0; index-- {
		row := behavior.rows[order[index]]
		if !row.row.Disabled {
			return row.row.Key
		}
	}
	return ""
}

func copyTableColumns(columns []normalizedTableColumn) []Column {
	result := make([]Column, len(columns))
	for index, column := range columns {
		result[index] = column.column
		if column.validator != nil {
			value := column.validator.value
			result[index].Validator = &value
		}
	}
	return result
}

func copyTableRows(rows []normalizedTableRow) []TableRow {
	result := make([]TableRow, len(rows))
	for index, row := range rows {
		result[index] = row.row
		result[index].Cells = append([]TableCell(nil), row.row.Cells...)
	}
	return result
}

func cloneTableBehavior(behavior tableBehavior) tableBehavior {
	cloned := behavior
	cloned.scroll = behavior.scroll
	cloned.columns = make([]normalizedTableColumn, len(behavior.columns))
	for index, column := range behavior.columns {
		cloned.columns[index] = column
		cloned.columns[index].header.lines = cloneTextRows(column.header.lines)
		cloned.columns[index].validator = cloneTextValidator(column.validator)
		if column.validator != nil {
			value := column.validator.value
			cloned.columns[index].column.Validator = &value
		}
	}
	cloned.rows = make([]normalizedTableRow, len(behavior.rows))
	for index, row := range behavior.rows {
		cloned.rows[index] = row
		cloned.rows[index].row.Cells = append([]TableCell(nil), row.row.Cells...)
		cloned.rows[index].cells = make([]normalizedTableCell, len(row.cells))
		for cellIndex, cell := range row.cells {
			cloned.rows[index].cells[cellIndex] = cell
			cloned.rows[index].cells[cellIndex].text.lines = cloneTextRows(cell.text.lines)
		}
	}
	cloned.displayOrder = append([]int(nil), behavior.displayOrder...)
	cloned.columnWidths = append([]int(nil), behavior.columnWidths...)
	cloned.selected = append([]string(nil), behavior.selected...)
	cloned.statusMessage.lines = cloneTextRows(behavior.statusMessage.lines)
	return cloned
}

func tableBehaviorEqual(left, right tableBehavior) bool {
	if !scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.currentRow != right.currentRow || left.currentColumn != right.currentColumn ||
		left.selectionMode != right.selectionMode || left.requireSelection != right.requireSelection ||
		left.focusMode != right.focusMode || left.sortColumn != right.sortColumn ||
		left.sortDirection != right.sortDirection || left.status != right.status ||
		left.statusMessage.text != right.statusMessage.text || left.changeCommand != right.changeCommand ||
		left.activateCommand != right.activateCommand || left.sortCommand != right.sortCommand ||
		len(left.columns) != len(right.columns) || len(left.rows) != len(right.rows) ||
		len(left.displayOrder) != len(right.displayOrder) ||
		len(left.columnWidths) != len(right.columnWidths) || len(left.selected) != len(right.selected) {
		return false
	}
	for index := range left.columns {
		if !tableColumnEqual(left.columns[index], right.columns[index]) ||
			left.columnWidths[index] != right.columnWidths[index] {
			return false
		}
	}
	for index := range left.rows {
		if !tableRowEqual(left.rows[index], right.rows[index]) {
			return false
		}
	}
	for index := range left.displayOrder {
		if left.displayOrder[index] != right.displayOrder[index] {
			return false
		}
	}
	for index := range left.selected {
		if left.selected[index] != right.selected[index] {
			return false
		}
	}
	return true
}

func tableColumnEqual(left, right normalizedTableColumn) bool {
	leftColumn, rightColumn := left.column, right.column
	leftColumn.Validator, rightColumn.Validator = nil, nil
	if leftColumn != rightColumn || left.header.text != right.header.text ||
		(left.validator == nil) != (right.validator == nil) {
		return false
	}
	return left.validator == nil || left.validator.value == right.validator.value
}

func tableRowEqual(left, right normalizedTableRow) bool {
	if left.row.Key != right.row.Key || left.row.Disabled != right.row.Disabled ||
		left.row.DisabledReason != right.row.DisabledReason || len(left.cells) != len(right.cells) {
		return false
	}
	for index := range left.cells {
		if left.cells[index].cell != right.cells[index].cell {
			return false
		}
	}
	return true
}

func (b tableBehavior) controlBorder() borderBehavior { return b.scroll.border }

func (b tableBehavior) clientInset() int { return b.scroll.clientInset() }

func (b tableBehavior) controlClientRect(bounds Rect) Rect {
	return b.scroll.controlClientRect(bounds)
}

func (b tableBehavior) intrinsicMinimum() Size {
	minimum := b.scroll.intrinsicMinimum()
	minimum.Height++
	return minimum
}

func (b tableBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	return append(styles,
		"table.header",
		"table.header_current",
		"table.sort",
		"table.cell_current",
		"table.row_selected",
		"collection.current",
		"collection.current_selected",
		"collection.disabled",
		"collection.empty",
		"collection.loading",
		"collection.error",
	)
}

func (b tableBehavior) details() ControlDetails {
	details := b.scroll.border.details()
	details.Container = nil
	details.Table = &TableDetails{}
	return details
}

func (b tableBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	b.scroll.paintDecoration(app, frame, state, absolute, clip)
	geometry := calculateScrollViewGeometry(absolute.Size(), b.scroll)
	viewport := translatedRect(geometry.viewport, absolute.X, absolute.Y)
	visible := viewport.Intersect(clip)
	if visible.Width <= 0 || visible.Height <= 0 {
		return
	}
	for y := visible.Y; y < visible.Y+visible.Height; y++ {
		var cells []tablePaintCell
		if y == viewport.Y {
			cells = b.headerCells(app.focus == state)
		} else {
			sourceRow := b.scroll.state.Offset.Y + y - viewport.Y - 1
			cells = b.bodyCells(sourceRow, app.focus == state)
		}
		for x := visible.X; x < visible.X+visible.Width; x++ {
			sourceX := b.scroll.state.Offset.X + x - viewport.X
			cell := tablePaintCell{grapheme: " ", style: "table"}
			if sourceX >= 0 && sourceX < len(cells) {
				cell = cells[sourceX]
			}
			app.setCellLocked(frame, x, y, cell.grapheme, cell.style, app.styles[cell.style], state.id)
		}
	}
}

func (b tableBehavior) headerCells(focused bool) []tablePaintCell {
	if len(b.columns) == 0 {
		return tableTextPaintCells("", "table.header")
	}
	result := tableTextPaintCells("    ", "table.header")
	for columnIndex, column := range b.columns {
		style := StyleID("table.header")
		if focused && b.focusMode == TableFocusCell && column.column.Key == b.currentColumn {
			style = "table.header_current"
		}
		content := append([]string(nil), column.header.lines[0]...)
		content = alignedTableCells(content, b.columnWidths[columnIndex], TextAlignStart)
		if column.column.Key == b.sortColumn && len(content) > 0 {
			marker := "↑"
			if b.sortDirection == SortDescending {
				marker = "↓"
			}
			content[len(content)-1] = marker
		}
		for cellIndex, grapheme := range content {
			cellStyle := style
			if column.column.Key == b.sortColumn && cellIndex == len(content)-1 {
				cellStyle = "table.sort"
			}
			result = append(result, tablePaintCell{grapheme: grapheme, style: cellStyle})
		}
		if columnIndex+1 < len(b.columns) {
			result = append(result, tablePaintCell{grapheme: "│", style: "table.header"})
		}
	}
	return result
}

func (b tableBehavior) bodyCells(index int, focused bool) []tablePaintCell {
	if index != 0 && (b.status != CollectionReady || len(b.rows) == 0) {
		return nil
	}
	switch b.status {
	case CollectionLoading:
		return tableTextPaintCells("[loading] "+b.statusMessage.text, "collection.loading")
	case CollectionError:
		return tableTextPaintCells("[error] "+b.statusMessage.text, "collection.error")
	}
	if len(b.rows) == 0 {
		return tableTextPaintCells("[empty]", "collection.empty")
	}
	order := tableDisplayOrder(b)
	if index < 0 || index >= len(order) {
		return nil
	}
	row := b.rows[order[index]]
	current := row.row.Key == b.currentRow
	selected := listSelectionContains(b.selected, row.row.Key)
	rowStyle := StyleID("table")
	if row.row.Disabled {
		rowStyle = "collection.disabled"
	} else if focused && current && b.focusMode == TableFocusRow && selected {
		rowStyle = "collection.current_selected"
	} else if focused && current && b.focusMode == TableFocusRow {
		rowStyle = "collection.current"
	} else if selected {
		rowStyle = "table.row_selected"
	}
	marker := " "
	if focused && current {
		marker = "►"
	}
	selectedMarker := " "
	if selected {
		selectedMarker = "X"
	}
	result := []tablePaintCell{
		{grapheme: marker, style: rowStyle},
		{grapheme: "[", style: rowStyle},
		{grapheme: selectedMarker, style: rowStyle},
		{grapheme: "]", style: rowStyle},
	}
	for columnIndex, column := range b.columns {
		text := []string{}
		if cell, found := tableCellForColumn(row, column.column.Key); found {
			text = cell.text.lines[0]
		}
		aligned := alignedTableCells(text, b.columnWidths[columnIndex], column.column.Alignment)
		cellStyle := rowStyle
		if !row.row.Disabled && focused && current && b.focusMode == TableFocusCell &&
			column.column.Key == b.currentColumn {
			cellStyle = "table.cell_current"
		}
		for _, grapheme := range aligned {
			result = append(result, tablePaintCell{grapheme: grapheme, style: cellStyle})
		}
		if columnIndex+1 < len(b.columns) {
			result = append(result, tablePaintCell{grapheme: "│", style: rowStyle})
		}
	}
	return result
}

func tableTextPaintCells(text string, style StyleID) []tablePaintCell {
	normalized, err := normalizeDisplayText(text, false)
	if err != nil || len(normalized.lines) == 0 {
		return nil
	}
	result := make([]tablePaintCell, len(normalized.lines[0]))
	for index, grapheme := range normalized.lines[0] {
		result[index] = tablePaintCell{grapheme: grapheme, style: style}
	}
	return result
}

func alignedTableCells(cells []string, width int, alignment TextAlignment) []string {
	if width <= 0 {
		return nil
	}
	if len(cells) > width {
		return append([]string(nil), cells[:width]...)
	}
	result := make([]string, width)
	for index := range result {
		result[index] = " "
	}
	start := 0
	switch alignment {
	case TextAlignCenter:
		start = (width - len(cells)) / 2
	case TextAlignEnd:
		start = width - len(cells)
	}
	copy(result[start:], cells)
	return result
}

func tableBaseColumnWidths(behavior tableBehavior) []int {
	widths := make([]int, len(behavior.columns))
	for index, column := range behavior.columns {
		width := column.header.cells
		for _, row := range behavior.rows {
			if cell, found := tableCellForColumn(row, column.column.Key); found {
				width = max(width, cell.text.cells)
			}
		}
		if column.column.Width > 0 {
			width = column.column.Width
		}
		width = max(1, max(width, column.column.MinimumWidth))
		if column.column.MaximumWidth > 0 {
			width = min(width, column.column.MaximumWidth)
		}
		widths[index] = width
	}
	return widths
}

func tableTotalWidth(widths []int) int {
	if len(widths) == 0 {
		return 1
	}
	total := 4 + len(widths) - 1
	for _, width := range widths {
		total = min(maxCoordinateMagnitude, total+width)
	}
	return total
}

func growTableColumnWidths(columns []normalizedTableColumn, base []int, viewportWidth int) []int {
	result := append([]int(nil), base...)
	remaining := viewportWidth - tableTotalWidth(result)
	for remaining > 0 {
		totalGrow := 0
		for index, column := range columns {
			if column.column.Grow > 0 &&
				(column.column.MaximumWidth == 0 || result[index] < column.column.MaximumWidth) {
				totalGrow += column.column.Grow
			}
		}
		if totalGrow == 0 {
			break
		}
		changed := 0
		for index, column := range columns {
			if remaining == 0 {
				break
			}
			if column.column.Grow == 0 ||
				(column.column.MaximumWidth > 0 && result[index] >= column.column.MaximumWidth) {
				continue
			}
			share := max(1, remaining*column.column.Grow/totalGrow)
			if column.column.MaximumWidth > 0 {
				share = min(share, column.column.MaximumWidth-result[index])
			}
			share = min(share, remaining)
			result[index] += share
			remaining -= share
			changed += share
		}
		if changed == 0 {
			break
		}
	}
	return result
}

func reflowTable(behavior tableBehavior, size Size) tableBehavior {
	behavior.displayOrder = nil
	behavior.displayOrder = deriveTableDisplayOrder(behavior)
	bodyRows := len(behavior.rows)
	if behavior.status != CollectionReady || bodyRows == 0 {
		bodyRows = 1
	}
	baseWidths := tableBaseColumnWidths(behavior)
	behavior.columnWidths = append([]int(nil), baseWidths...)
	behavior.scroll.state.ContentSize = Size{
		Width: tableTotalWidth(behavior.columnWidths), Height: bodyRows + 1,
	}
	for range 3 {
		geometry := calculateScrollViewGeometry(size, behavior.scroll)
		next := growTableColumnWidths(behavior.columns, baseWidths, geometry.viewport.Width)
		nextWidth := tableTotalWidth(next)
		if nextWidth == behavior.scroll.state.ContentSize.Width &&
			len(next) == len(behavior.columnWidths) {
			behavior.columnWidths = next
			break
		}
		behavior.columnWidths = next
		behavior.scroll.state.ContentSize.Width = nextWidth
	}
	geometry := calculateScrollViewGeometry(size, behavior.scroll)
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	if behavior.status == CollectionReady {
		current := tableDisplayRowIndex(behavior, behavior.currentRow)
		bodyHeight := max(0, geometry.viewport.Height-1)
		if current >= 0 && bodyHeight > 0 {
			if current < behavior.scroll.state.Offset.Y {
				behavior.scroll.state.Offset.Y = current
			} else if current >= behavior.scroll.state.Offset.Y+bodyHeight {
				behavior.scroll.state.Offset.Y = current - bodyHeight + 1
			}
		}
		if behavior.focusMode == TableFocusCell && geometry.viewport.Width > 0 {
			start, end := tableColumnInterval(behavior, behavior.currentColumn)
			if start < behavior.scroll.state.Offset.X {
				behavior.scroll.state.Offset.X = start
			} else if end > behavior.scroll.state.Offset.X+geometry.viewport.Width {
				behavior.scroll.state.Offset.X = end - geometry.viewport.Width
			}
		}
	}
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	return behavior
}

func tableColumnInterval(behavior tableBehavior, key string) (int, int) {
	start := 4
	for index, column := range behavior.columns {
		end := start + behavior.columnWidths[index]
		if column.column.Key == key {
			return start, end
		}
		start = end + 1
	}
	return 0, 0
}

func reconcileTableLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(tableBehavior)
	if !ok {
		return false
	}
	next := reflowTable(behavior, state.bounds.Size())
	if tableBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func tableCanFocus(state *controlState, behavior tableBehavior) bool {
	return state != nil && !behavior.scroll.disabled && behavior.status == CollectionReady &&
		behavior.currentRow != "" &&
		(behavior.focusMode == TableFocusRow || behavior.currentColumn != "")
}

func enabledTableRowCount(rows []normalizedTableRow) int {
	count := 0
	for _, row := range rows {
		if !row.row.Disabled {
			count++
		}
	}
	return count
}

func tableCellCount(rows []normalizedTableRow) int {
	count := 0
	for _, row := range rows {
		count += len(row.cells)
	}
	return count
}

func tableDetails(bounds Rect, behavior tableBehavior) TableDetails {
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	firstSelected, lastSelected := "", ""
	if len(behavior.selected) > 0 {
		firstSelected = behavior.selected[0]
		lastSelected = behavior.selected[len(behavior.selected)-1]
	}
	firstColumn, lastColumn := "", ""
	if len(behavior.columns) > 0 {
		firstColumn = behavior.columns[0].column.Key
		lastColumn = behavior.columns[len(behavior.columns)-1].column.Key
	}
	return TableDetails{
		Status: behavior.status, StatusMessage: behavior.statusMessage.text,
		RowCount: len(behavior.rows), EnabledCount: enabledTableRowCount(behavior.rows),
		ColumnCount: len(behavior.columns), CellCount: tableCellCount(behavior.rows),
		RetainedBytes: tableStorageBytes(behavior), CurrentRow: behavior.currentRow,
		CurrentRowIndex:    tableDisplayRowIndex(behavior, behavior.currentRow),
		CurrentColumn:      behavior.currentColumn,
		CurrentColumnIndex: tableColumnIndex(behavior.columns, behavior.currentColumn),
		FocusMode:          behavior.focusMode, SelectionMode: behavior.selectionMode,
		RequireSelection: behavior.requireSelection, SelectedCount: len(behavior.selected),
		FirstSelected: firstSelected, LastSelected: lastSelected,
		SelectionDigest: listSelectionDigest(behavior.selected),
		SortColumn:      behavior.sortColumn, SortDirection: behavior.sortDirection,
		FirstColumn: firstColumn, LastColumn: lastColumn,
		ColumnWidths:       append([]int(nil), behavior.columnWidths...),
		ColumnWidthsDigest: integerSequenceDigest(behavior.columnWidths),
		Enabled:            !behavior.scroll.disabled, DisabledReason: behavior.scroll.disabledReason,
		ChangeCommand: behavior.changeCommand, ActivateCommand: behavior.activateCommand,
		SortCommand: behavior.sortCommand, Viewport: viewport,
	}
}

// Columns returns a caller-owned copy of the complete retained schema.
func (t *Table) Columns() []Column {
	if t == nil || t.state == nil || t.state.app == nil {
		return nil
	}
	t.state.app.mu.RLock()
	defer t.state.app.mu.RUnlock()
	behavior, ok := t.state.behavior.(tableBehavior)
	if !ok || t.state.destroyed || t.state.aborted {
		return nil
	}
	return copyTableColumns(behavior.columns)
}

// Rows returns a caller-owned copy of the canonical unsorted row model.
func (t *Table) Rows() []TableRow {
	if t == nil || t.state == nil || t.state.app == nil {
		return nil
	}
	t.state.app.mu.RLock()
	defer t.state.app.mu.RUnlock()
	behavior, ok := t.state.behavior.(tableBehavior)
	if !ok || t.state.destroyed || t.state.aborted {
		return nil
	}
	return copyTableRows(behavior.rows)
}

// State returns a caller-owned copy of complete table state.
func (t *Table) State() TableState {
	if t == nil || t.state == nil || t.state.app == nil {
		return TableState{CurrentRowIndex: -1, CurrentColumnIndex: -1}
	}
	t.state.app.mu.RLock()
	defer t.state.app.mu.RUnlock()
	behavior, ok := t.state.behavior.(tableBehavior)
	if !ok || t.state.destroyed || t.state.aborted {
		return TableState{CurrentRowIndex: -1, CurrentColumnIndex: -1}
	}
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
func (t *Table) SetRows(rows []TableRow) error {
	return commitTableMutation(t, func(tx *Transaction) error { return tx.SetTableRows(t, rows) })
}

// SetModel replaces columns and rows while preserving surviving stable state.
func (t *Table) SetModel(columns []Column, rows []TableRow) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.SetTableModel(t, columns, rows)
	})
}

// Replace atomically replaces the model and exact current, selection, and sort.
func (t *Table) Replace(
	columns []Column,
	rows []TableRow,
	currentRow string,
	currentColumn string,
	selected []string,
	sortColumn string,
	sortDirection SortDirection,
) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.ReplaceTable(t, columns, rows, currentRow, currentColumn, selected, sortColumn, sortDirection)
	})
}

// SetCurrent changes current by stable enabled row and known column keys.
func (t *Table) SetCurrent(row, column string) error {
	return commitTableMutation(t, func(tx *Transaction) error { return tx.SetTableCurrent(t, row, column) })
}

// SetSelection changes the exact selected stable row keys.
func (t *Table) SetSelection(selected []string) error {
	return commitTableMutation(t, func(tx *Transaction) error { return tx.SetTableSelection(t, selected) })
}

// SetSort changes the optional single-column derived stable display order.
func (t *Table) SetSort(column string, direction SortDirection) error {
	return commitTableMutation(t, func(tx *Transaction) error { return tx.SetTableSort(t, column, direction) })
}

// SetStatus changes ready/loading/error presentation without discarding data.
func (t *Table) SetStatus(status CollectionStatus, message string) error {
	return commitTableMutation(t, func(tx *Transaction) error { return tx.SetTableStatus(t, status, message) })
}

func commitTableMutation(control *Table, record func(*Transaction) error) error {
	if control == nil || control.state == nil || control.state.app == nil {
		return ErrInvalidControl
	}
	tx := control.state.app.NewTransaction()
	if err := record(tx); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Focus atomically gives the Table keyboard focus when it is ready and has an enabled row.
func (t *Table) Focus() error { return focusSelectionControl(t) }

// Activate selects current when necessary and invokes the optional activation command.
func (t *Table) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if t == nil || t.state == nil || t.state.app == nil {
		return Completion{}, ErrInvalidControl
	}
	return t.state.app.activateTable(ctx, source, requestID, t.state)
}

func (a *App) activateTable(
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
	behavior, ok := state.behavior.(tableBehavior)
	if !ok || !tableCanFocus(state, behavior) ||
		!a.inActiveModalScopeLocked(state) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	command, target, _, changed := a.tableKeyLocked(state, KeyEnter, false)
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

func (a *App) tableKeyLocked(
	state *controlState,
	key Key,
	control bool,
) (CommandID, ControlID, bool, bool) {
	if state == nil {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(tableBehavior)
	if !ok || !tableCanFocus(state, behavior) {
		return "", "", false, false
	}
	before := cloneTableBehavior(behavior)
	handled, selectionChanged, activate, sortChanged := true, false, false, false
	current := tableDisplayRowIndex(behavior, behavior.currentRow)
	geometry := calculateScrollViewGeometry(state.bounds.Size(), behavior.scroll)
	if control && (key == KeyHome || key == KeyEnd) {
		if key == KeyHome {
			behavior.currentRow = firstEnabledTableRowKey(&behavior)
			if len(behavior.columns) > 0 {
				behavior.currentColumn = behavior.columns[0].column.Key
			}
		} else {
			behavior.currentRow = lastEnabledTableRowKey(behavior)
			if len(behavior.columns) > 0 {
				behavior.currentColumn = behavior.columns[len(behavior.columns)-1].column.Key
			}
		}
	} else if control {
		return "", "", false, false
	} else {
		switch key {
		case KeyUp:
			behavior.currentRow = previousEnabledTableRowKey(behavior, current)
		case KeyDown:
			behavior.currentRow = nextEnabledTableRowKey(behavior, current)
		case KeyPageUp:
			behavior.currentRow = pageEnabledTableRowKey(
				behavior, current,
				-effectiveScrollPageStep(behavior.scroll.pageStep.Height, max(1, geometry.viewport.Height-1)),
			)
		case KeyPageDown:
			behavior.currentRow = pageEnabledTableRowKey(
				behavior, current,
				effectiveScrollPageStep(behavior.scroll.pageStep.Height, max(1, geometry.viewport.Height-1)),
			)
		case KeyHome:
			behavior.currentRow = firstEnabledTableRowKey(&behavior)
		case KeyEnd:
			behavior.currentRow = lastEnabledTableRowKey(behavior)
		case KeyLeft:
			if behavior.focusMode == TableFocusCell {
				column := tableColumnIndex(behavior.columns, behavior.currentColumn)
				if column > 0 {
					behavior.currentColumn = behavior.columns[column-1].column.Key
				}
			} else {
				behavior.scroll.state.Offset.X -= behavior.scroll.arrowStep.Width
			}
		case KeyRight:
			if behavior.focusMode == TableFocusCell {
				column := tableColumnIndex(behavior.columns, behavior.currentColumn)
				if column >= 0 && column+1 < len(behavior.columns) {
					behavior.currentColumn = behavior.columns[column+1].column.Key
				}
			} else {
				behavior.scroll.state.Offset.X += behavior.scroll.arrowStep.Width
			}
		case KeySpace:
			selectionChanged = selectCurrentTableRow(&behavior, true)
		case KeyEnter:
			selectionChanged = selectCurrentTableRow(&behavior, false)
			activate = true
		case Key("s"), Key("S"):
			sortChanged = cycleCurrentTableSort(&behavior)
		default:
			handled = false
		}
	}
	if !handled {
		return "", "", false, false
	}
	behavior = reflowTable(behavior, state.bounds.Size())
	changed := !tableBehaviorEqual(before, behavior)
	if changed {
		state.behavior = behavior
	}
	command := CommandID("")
	if activate && behavior.activateCommand != "" {
		command = behavior.activateCommand
	} else if sortChanged {
		command = behavior.sortCommand
	} else if selectionChanged {
		command = behavior.changeCommand
	}
	return command, state.id, true, changed
}

func previousEnabledTableRowKey(behavior tableBehavior, current int) string {
	order := tableDisplayOrder(behavior)
	for index := current - 1; index >= 0; index-- {
		row := behavior.rows[order[index]]
		if !row.row.Disabled {
			return row.row.Key
		}
	}
	return behavior.currentRow
}

func nextEnabledTableRowKey(behavior tableBehavior, current int) string {
	order := tableDisplayOrder(behavior)
	for index := current + 1; index < len(order); index++ {
		row := behavior.rows[order[index]]
		if !row.row.Disabled {
			return row.row.Key
		}
	}
	return behavior.currentRow
}

func pageEnabledTableRowKey(behavior tableBehavior, current, delta int) string {
	order := tableDisplayOrder(behavior)
	if len(order) == 0 {
		return ""
	}
	target := min(len(order)-1, max(0, current+delta))
	if delta < 0 {
		for index := target; index >= 0; index-- {
			if !behavior.rows[order[index]].row.Disabled {
				return behavior.rows[order[index]].row.Key
			}
		}
		return nextEnabledTableRowKey(behavior, target-1)
	}
	for index := target; index < len(order); index++ {
		if !behavior.rows[order[index]].row.Disabled {
			return behavior.rows[order[index]].row.Key
		}
	}
	return previousEnabledTableRowKey(behavior, target+1)
}

func selectCurrentTableRow(behavior *tableBehavior, toggle bool) bool {
	if behavior == nil || behavior.currentRow == "" {
		return false
	}
	selected := listSelectionContains(behavior.selected, behavior.currentRow)
	if behavior.selectionMode == CollectionSelectionSingle {
		if selected && len(behavior.selected) == 1 {
			return false
		}
		behavior.selected = []string{behavior.currentRow}
		return true
	}
	if selected {
		if !toggle || (behavior.requireSelection && len(behavior.selected) == 1) {
			return false
		}
		result := make([]string, 0, len(behavior.selected)-1)
		for _, key := range behavior.selected {
			if key != behavior.currentRow {
				result = append(result, key)
			}
		}
		behavior.selected = result
		return true
	}
	requested := append(append([]string(nil), behavior.selected...), behavior.currentRow)
	ordered, _ := normalizeTableSelection(behavior, requested)
	behavior.selected = ordered
	return true
}

func cycleCurrentTableSort(behavior *tableBehavior) bool {
	if behavior == nil {
		return false
	}
	index := tableColumnIndex(behavior.columns, behavior.currentColumn)
	if index < 0 || !behavior.columns[index].column.Sortable {
		return false
	}
	switch {
	case behavior.sortColumn != behavior.currentColumn || behavior.sortDirection == SortNone:
		behavior.sortColumn = behavior.currentColumn
		behavior.sortDirection = SortAscending
	case behavior.sortDirection == SortAscending:
		behavior.sortDirection = SortDescending
	default:
		behavior.sortColumn = ""
		behavior.sortDirection = SortNone
	}
	behavior.displayOrder = nil
	behavior.selected, _ = normalizeTableSelection(behavior, behavior.selected)
	return true
}

func tableStorageBytes(behavior tableBehavior) int {
	total := len(behavior.statusMessage.text) + len(behavior.currentRow) +
		len(behavior.currentColumn) + len(behavior.sortColumn)
	for _, column := range behavior.columns {
		total += len(column.column.Key) + len(column.column.Header)
		if column.validator != nil {
			total += len(column.validator.value.Characters)
		}
	}
	for _, row := range behavior.rows {
		total += len(row.row.Key) + len(row.row.DisabledReason)
		for _, cell := range row.cells {
			total += len(cell.cell.Column) + len(cell.cell.Text)
		}
	}
	for _, key := range behavior.selected {
		total += len(key)
	}
	return total
}

func integerSequenceDigest(values []int) string {
	hash := sha256.New()
	var encoded [8]byte
	for _, value := range values {
		binary.BigEndian.PutUint64(encoded[:], uint64(value))
		_, _ = hash.Write(encoded[:])
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (t *Transaction) selectedTable(control *Table) (*controlState, tableBehavior, error) {
	target, err := t.control(control)
	if err != nil {
		return nil, tableBehavior{}, err
	}
	behavior, ok := t.selectedControlBehavior(target).(tableBehavior)
	if !ok {
		return nil, tableBehavior{}, ErrInvalidControl
	}
	return target, cloneTableBehavior(behavior), nil
}

func (t *Transaction) recordTable(state *controlState, behavior tableBehavior) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{kind: mutationTable, state: state, behavior: behavior})
	return nil
}

// SetTableRows records a copied row replacement preserving stable state.
func (t *Transaction) SetTableRows(control *Table, rows []TableRow) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeTableRows(behavior.columns, rows)
	if err != nil {
		return err
	}
	oldIndex := tableDisplayRowIndex(behavior, behavior.currentRow)
	behavior.rows = normalized
	behavior.displayOrder = nil
	repairTableIdentity(&behavior, oldIndex)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableModel records a copied schema/model replacement preserving state.
func (t *Transaction) SetTableModel(control *Table, columns []Column, rows []TableRow) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	oldIndex := tableDisplayRowIndex(behavior, behavior.currentRow)
	normalizedColumns, err := normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	normalizedRows, err := normalizeTableRows(normalizedColumns, rows)
	if err != nil {
		return err
	}
	behavior.columns, behavior.rows = normalizedColumns, normalizedRows
	behavior.displayOrder = nil
	if tableColumnIndex(behavior.columns, behavior.sortColumn) < 0 ||
		(behavior.sortColumn != "" && !behavior.columns[tableColumnIndex(behavior.columns, behavior.sortColumn)].column.Sortable) {
		behavior.sortColumn, behavior.sortDirection = "", SortNone
	}
	if tableColumnIndex(behavior.columns, behavior.currentColumn) < 0 {
		behavior.currentColumn = ""
		if len(behavior.columns) > 0 {
			behavior.currentColumn = behavior.columns[0].column.Key
		}
	}
	repairTableIdentity(&behavior, oldIndex)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// ReplaceTable records one exact complete table replacement.
func (t *Transaction) ReplaceTable(
	control *Table,
	columns []Column,
	rows []TableRow,
	currentRow string,
	currentColumn string,
	selected []string,
	sortColumn string,
	sortDirection SortDirection,
) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	behavior.columns, err = normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	behavior.rows, err = normalizeTableRows(behavior.columns, rows)
	if err != nil {
		return err
	}
	behavior.displayOrder = nil
	if err := setExactTableSort(&behavior, sortColumn, sortDirection); err != nil {
		return err
	}
	if err := setExactTableIdentity(&behavior, currentRow, currentColumn, selected); err != nil {
		return err
	}
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableCurrent records exact stable current row and column keys.
func (t *Transaction) SetTableCurrent(control *Table, row, column string) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	if row == "" {
		if firstEnabledTableRowKey(&behavior) != "" {
			return fmt.Errorf("%w: ready Table requires current row", ErrValidation)
		}
	} else if index := tableCanonicalRowIndex(behavior.rows, row); index < 0 || behavior.rows[index].row.Disabled {
		return fmt.Errorf("%w: invalid Table current row", ErrValidation)
	}
	if column == "" {
		if len(behavior.columns) > 0 {
			return fmt.Errorf("%w: Table requires current column", ErrValidation)
		}
	} else if tableColumnIndex(behavior.columns, column) < 0 {
		return fmt.Errorf("%w: invalid Table current column", ErrValidation)
	}
	behavior.currentRow, behavior.currentColumn = row, column
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableSelection records one exact stable-row selection.
func (t *Transaction) SetTableSelection(control *Table, selected []string) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	ordered, err := normalizeTableSelection(&behavior, selected)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(ordered) == 0 && behavior.currentRow != "" {
		return fmt.Errorf("%w: Table selection is required", ErrValidation)
	}
	behavior.selected = ordered
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableSort records an exact optional single-column sort.
func (t *Transaction) SetTableSort(control *Table, column string, direction SortDirection) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	if err := setExactTableSort(&behavior, column, direction); err != nil {
		return err
	}
	behavior.displayOrder = nil
	behavior.selected, _ = normalizeTableSelection(&behavior, behavior.selected)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableStatus records ready/loading/error presentation state.
func (t *Transaction) SetTableStatus(control *Table, status CollectionStatus, message string) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	status, normalized, err := normalizeCollectionStatus(status, message)
	if err != nil {
		return err
	}
	behavior.status, behavior.statusMessage = status, normalized
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

func repairTableIdentity(behavior *tableBehavior, oldDisplayIndex int) {
	if behavior == nil {
		return
	}
	currentIndex := tableCanonicalRowIndex(behavior.rows, behavior.currentRow)
	if currentIndex < 0 || behavior.rows[currentIndex].row.Disabled {
		behavior.currentRow = repairedTableCurrent(*behavior, oldDisplayIndex)
	}
	requested := make([]string, 0, len(behavior.selected))
	for _, key := range behavior.selected {
		index := tableCanonicalRowIndex(behavior.rows, key)
		if index >= 0 && !behavior.rows[index].row.Disabled {
			requested = append(requested, key)
		}
	}
	behavior.selected, _ = normalizeTableSelection(behavior, requested)
	if behavior.selectionMode == CollectionSelectionSingle && len(behavior.selected) > 1 {
		behavior.selected = behavior.selected[:1]
	}
	if behavior.requireSelection && len(behavior.selected) == 0 && behavior.currentRow != "" {
		behavior.selected = []string{behavior.currentRow}
	}
}

func repairedTableCurrent(behavior tableBehavior, oldDisplayIndex int) string {
	order := tableDisplayOrder(behavior)
	if len(order) == 0 {
		return ""
	}
	oldDisplayIndex = min(len(order)-1, max(0, oldDisplayIndex))
	for index := oldDisplayIndex; index < len(order); index++ {
		if !behavior.rows[order[index]].row.Disabled {
			return behavior.rows[order[index]].row.Key
		}
	}
	for index := oldDisplayIndex - 1; index >= 0; index-- {
		if !behavior.rows[order[index]].row.Disabled {
			return behavior.rows[order[index]].row.Key
		}
	}
	return ""
}
