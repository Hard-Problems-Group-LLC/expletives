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
	"unicode"
	"unicode/utf8"
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

// TableSelectionStyle selects the exact row-selection behavior of a Table or
// DataGrid. The zero value is accepted only by construction options, where it
// preserves the legacy SelectionMode behavior.
type TableSelectionStyle string

const (
	TableSelectionNone     TableSelectionStyle = "none"
	TableSelectionSingle   TableSelectionStyle = "single"
	TableSelectionRange    TableSelectionStyle = "range"
	TableSelectionMultiple TableSelectionStyle = "multiple"
)

// TableSelectionPolicy is one atomic row-selection configuration and state.
// Selected is copied by constructors and mutations.
type TableSelectionPolicy struct {
	Style    TableSelectionStyle
	Require  bool
	Selected []string
	Anchor   string
	Extent   string
}

// TableFeature selects one optional compatible Table or DataGrid behavior.
type TableFeature string

const (
	TableFeatureColumns TableFeature = "columns"
)

const tableColumnsActionWidth = 14

// TableFocusPart identifies the internally focused semantic portion of one
// Table or DataGrid without introducing a child Control identity.
type TableFocusPart string

const (
	TableFocusBody          TableFocusPart = "body"
	TableFocusColumnsAction TableFocusPart = "columns_action"
)

// TableColumnWrap selects the body-text presentation for one Table or
// DataGrid column. The zero value normalizes to TableColumnClip.
type TableColumnWrap string

const (
	TableColumnClip      TableColumnWrap = "clip"
	TableColumnWrapWords TableColumnWrap = "wrap"
	TableColumnHang      TableColumnWrap = "hang"
)

// TableColumnPresentation is one stable-keyed column presentation entry.
// Slice order is display order and includes hidden columns.
type TableColumnPresentation struct {
	Column  string
	Visible bool
	Wrap    TableColumnWrap
}

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
	Features           []TableFeature
	Columns            []Column
	Rows               []TableRow
	CurrentRow         string
	CurrentColumn      string
	Selected           []string
	SelectionMode      CollectionSelectionMode
	SelectionStyle     TableSelectionStyle
	RequireSelection   bool
	RangeAnchor        string
	RangeExtent        string
	ColumnPresentation []TableColumnPresentation
	FocusMode          TableFocusMode
	SortColumn         string
	SortDirection      SortDirection
	Status             CollectionStatus
	StatusMessage      string
	ActivateCommand    CommandID
	SortCommand        CommandID
}

// TableState is one complete copied semantic and viewport state.
type TableState struct {
	Status             CollectionStatus
	StatusMessage      string
	Features           []TableFeature
	CurrentRow         string
	CurrentRowIndex    int
	CurrentColumn      string
	CurrentColumnIndex int
	Selected           []string
	SelectionStyle     TableSelectionStyle
	RangeAnchor        string
	RangeExtent        string
	ColumnPresentation []TableColumnPresentation
	VisibleColumnCount int
	VisualRowCount     int
	FocusPart          TableFocusPart
	ColumnsDialogOpen  bool
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
	scroll               scrollViewBehavior
	columns              []normalizedTableColumn
	rows                 []normalizedTableRow
	displayOrder         []int
	columnWidths         []int
	rowHeights           []int
	rowStarts            []int
	features             []TableFeature
	focusPart            TableFocusPart
	schemaRevision       uint64
	presentationRevision uint64
	presentationMutation bool
	geometryRevision     uint64
	geometryCachedAt     uint64
	geometryCacheSize    Size
	currentRow           string
	currentColumn        string
	selected             []string
	selectionMode        CollectionSelectionMode
	selectionStyle       TableSelectionStyle
	requireSelection     bool
	rangeAnchor          string
	rangeExtent          string
	columnPresentation   []TableColumnPresentation
	initialPresentation  []TableColumnPresentation
	focusMode            TableFocusMode
	sortColumn           string
	sortDirection        SortDirection
	status               CollectionStatus
	statusMessage        normalizedDisplayText
	changeCommand        CommandID
	activateCommand      CommandID
	sortCommand          CommandID
}

type tablePaintCell struct {
	grapheme string
	style    StyleID
}

type tableWrappedLine struct {
	cells  []string
	indent int
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
	selectionStyle, mode, err := normalizeTableSelectionStyle(
		options.SelectionStyle,
		options.SelectionMode,
	)
	if err != nil {
		return tableBehavior{}, err
	}
	features, err := normalizeTableFeatures(options.Features)
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
	presentation, err := normalizeTableColumnPresentation(
		columns,
		options.ColumnPresentation,
	)
	if err != nil {
		return tableBehavior{}, err
	}
	rows, err := normalizeTableRows(columns, options.Rows)
	if err != nil {
		return tableBehavior{}, err
	}
	behavior := tableBehavior{
		scroll:               scroll,
		columns:              columns,
		rows:                 rows,
		features:             features,
		focusPart:            TableFocusBody,
		schemaRevision:       1,
		presentationRevision: 1,
		selectionMode:        mode,
		selectionStyle:       selectionStyle,
		requireSelection:     options.RequireSelection,
		columnPresentation:   presentation,
		initialPresentation: append(
			[]TableColumnPresentation(nil),
			presentation...,
		),
		geometryRevision: 1,
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
		options.RangeAnchor,
		options.RangeExtent,
	); err != nil {
		return tableBehavior{}, err
	}
	repairTableCurrentColumn(&behavior, behavior.columnPresentation)
	return behavior, nil
}

func normalizeTableFeatures(features []TableFeature) ([]TableFeature, error) {
	if len(features) > MaxTableFeatures {
		return nil, fmt.Errorf(
			"%w: Table exceeds %d features",
			ErrControlCapacity,
			MaxTableFeatures,
		)
	}
	requested := make(map[TableFeature]bool, len(features))
	for _, feature := range features {
		if feature != TableFeatureColumns || requested[feature] {
			return nil, fmt.Errorf(
				"%w: unknown or duplicate Table feature %q",
				ErrValidation,
				feature,
			)
		}
		requested[feature] = true
	}
	result := make([]TableFeature, 0, len(requested))
	for _, feature := range []TableFeature{TableFeatureColumns} {
		if requested[feature] {
			result = append(result, feature)
		}
	}
	return result, nil
}

func tableHasFeature(behavior tableBehavior, feature TableFeature) bool {
	for _, current := range behavior.features {
		if current == feature {
			return true
		}
	}
	return false
}

func tableColumnsBandRows(behavior tableBehavior, size Size) int {
	if !tableHasFeature(behavior, TableFeatureColumns) ||
		size.Width <= 0 || size.Height <= 0 {
		return 0
	}
	return min(2, size.Height)
}

func tableBodySize(behavior tableBehavior, size Size) Size {
	size.Height = max(0, size.Height-tableColumnsBandRows(behavior, size))
	return size
}

func tableBodyRect(behavior tableBehavior, bounds Rect) Rect {
	result := bounds
	result.Height = tableBodySize(behavior, bounds.Size()).Height
	return result
}

func tableColumnsActionBounds(behavior tableBehavior, bounds Rect) Rect {
	rows := tableColumnsBandRows(behavior, bounds.Size())
	if rows == 0 {
		return Rect{}
	}
	return Rect{
		X:      bounds.X + bounds.Width - tableColumnsActionWidth,
		Y:      bounds.Y + bounds.Height - rows,
		Width:  tableColumnsActionWidth,
		Height: rows,
	}
}

func tableColumnsActionVisible(behavior tableBehavior, bounds Rect) bool {
	return !tableColumnsActionBounds(behavior, bounds).Intersect(bounds).Empty()
}

func tableBehaviorFromControl(state *controlState) (tableBehavior, bool) {
	if state == nil {
		return tableBehavior{}, false
	}
	switch behavior := state.behavior.(type) {
	case tableBehavior:
		return behavior, true
	case dataGridBehavior:
		return behavior.table, true
	default:
		return tableBehavior{}, false
	}
}

func setTableBehaviorOnControl(state *controlState, table tableBehavior) bool {
	if state == nil {
		return false
	}
	switch behavior := state.behavior.(type) {
	case tableBehavior:
		state.behavior = table
		return true
	case dataGridBehavior:
		behavior.table = table
		state.behavior = behavior
		return true
	default:
		return false
	}
}

func (a *App) tableColumnsActionEligibleLocked(state *controlState) bool {
	behavior, ok := tableBehaviorFromControl(state)
	if !ok || behavior.scroll.disabled ||
		!tableHasFeature(behavior, TableFeatureColumns) ||
		!a.controlReceivesInputLocked(state) {
		return false
	}
	absolute, found := a.controlAbsoluteBoundsLocked(state)
	if !found {
		return false
	}
	action := tableColumnsActionBounds(behavior, absolute)
	return !action.Intersect(a.effectiveControlClipLocked(state)).Empty()
}

func tableBodyCanFocus(behavior tableBehavior) bool {
	return !behavior.scroll.disabled && behavior.status == CollectionReady &&
		behavior.currentRow != "" &&
		(behavior.focusMode == TableFocusRow || behavior.currentColumn != "")
}

func (a *App) repairFocusedTablePartLocked() bool {
	behavior, ok := tableBehaviorFromControl(a.focus)
	if !ok || behavior.focusPart != TableFocusColumnsAction ||
		a.tableColumnsActionEligibleLocked(a.focus) {
		return false
	}
	behavior.focusPart = TableFocusBody
	setTableBehaviorOnControl(a.focus, behavior)
	a.clearInvalidPressesLocked()
	return true
}

func (a *App) tableColumnsActionFocusedEligibleLocked(state *controlState) bool {
	behavior, ok := tableBehaviorFromControl(state)
	return ok && a.focus == state &&
		behavior.focusPart == TableFocusColumnsAction &&
		a.tableColumnsActionEligibleLocked(state)
}

func (a *App) tableColumnsActionPressedLocked(state *controlState) bool {
	for _, press := range a.pressed {
		if press.control == state && press.part == TableFocusColumnsAction {
			return true
		}
	}
	return false
}

func (a *App) setTableEntryFocusPartLocked(
	state *controlState,
	reverse bool,
) bool {
	behavior, ok := tableBehaviorFromControl(state)
	if !ok {
		return false
	}
	part := TableFocusBody
	if reverse && a.tableColumnsActionEligibleLocked(state) {
		part = TableFocusColumnsAction
	}
	if behavior.focusPart == part {
		return false
	}
	behavior.focusPart = part
	return setTableBehaviorOnControl(state, behavior)
}

func (a *App) moveTableFocusPartLocked(reverse bool) bool {
	behavior, ok := tableBehaviorFromControl(a.focus)
	if !ok || !a.tableColumnsActionEligibleLocked(a.focus) {
		return false
	}
	part := behavior.focusPart
	if (!reverse && part != TableFocusBody) ||
		(reverse && part != TableFocusColumnsAction) {
		return false
	}
	if reverse {
		behavior.focusPart = TableFocusBody
	} else {
		behavior.focusPart = TableFocusColumnsAction
	}
	setTableBehaviorOnControl(a.focus, behavior)
	a.clearInvalidPressesLocked()
	return true
}

func paintTableColumnsBand(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	behavior tableBehavior,
	absolute Rect,
	clip Rect,
) {
	rows := tableColumnsBandRows(behavior, absolute.Size())
	if rows == 0 {
		return
	}
	band := Rect{
		X: absolute.X, Y: absolute.Y + absolute.Height - rows,
		Width: absolute.Width, Height: rows,
	}.Intersect(clip)
	for y := band.Y; y < band.Y+band.Height; y++ {
		for x := band.X; x < band.X+band.Width; x++ {
			app.setCellLocked(
				frame, x, y, " ", state.style, app.styles[state.style], state.id,
			)
		}
	}
}

func paintTableColumnsAction(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	behavior tableBehavior,
	absolute Rect,
	clip Rect,
) {
	action := tableColumnsActionBounds(behavior, absolute)
	if action.Intersect(clip).Empty() {
		return
	}
	enabled := !behavior.scroll.disabled
	focused := enabled && app.focus == state &&
		behavior.focusPart == TableFocusColumnsAction
	pressed := focused && app.tableColumnsActionPressedLocked(state)
	bodyStyle := StyleID("button")
	mnemonicStyle := StyleID("button.mnemonic")
	switch {
	case !enabled:
		bodyStyle, mnemonicStyle = "button.disabled", "button.disabled"
	case pressed:
		bodyStyle = "button.pressed"
	case focused:
		bodyStyle = "button.focused"
	}
	paint := func(x, y int, grapheme string, style StyleID) {
		app.setClippedCellLocked(
			frame, clip, x, y, grapheme, style, app.styles[style], state.id,
		)
	}
	label := []string{"C", "o", "l", "u", "m", "n", "s", ".", ".", "."}
	paintLabel := func(left, right, y int) {
		if right <= left || y < action.Y || y >= action.Y+action.Height {
			return
		}
		start := left + max(0, (right-left-len(label))/2)
		for index, grapheme := range label {
			x := start + index
			if x >= right {
				break
			}
			style := bodyStyle
			if enabled && index == 0 {
				style = mnemonicStyle
			}
			paint(x, y, grapheme, style)
		}
	}
	if action.Height < 2 || action.Width < 3 {
		for y := action.Y; y < action.Y+action.Height; y++ {
			for x := action.X; x < action.X+action.Width; x++ {
				paint(x, y, " ", bodyStyle)
			}
		}
		paintLabel(action.X, action.X+action.Width, action.Y+(action.Height-1)/2)
		return
	}
	left, right := action.X, action.X+action.Width-1
	bottom := action.Y + action.Height - 1
	bodyLeft, bodyRight := left+1, right
	if pressed {
		bodyLeft, bodyRight = left+2, right+1
	}
	for y := action.Y; y < bottom; y++ {
		for x := left; x <= right; x++ {
			paint(x, y, " ", bodyStyle)
		}
		paint(left, y, " ", "button.shadow")
		if pressed {
			paint(left+1, y, " ", "button.shadow")
		} else {
			paint(right, y, "▄", "button.shadow")
		}
	}
	for x := left; x <= right; x++ {
		paint(x, bottom, " ", "button.shadow")
	}
	if !pressed {
		for x := left + 2; x <= right; x++ {
			paint(x, bottom, "▀", "button.shadow")
		}
	}
	paintLabel(bodyLeft, bodyRight, action.Y+action.Height/2-1)
}

func repairTableFocusPart(behavior *tableBehavior, size Size) {
	if behavior == nil {
		return
	}
	if behavior.focusPart == "" {
		behavior.focusPart = TableFocusBody
	}
	localBounds := Rect{Width: size.Width, Height: size.Height}
	if behavior.focusPart == TableFocusColumnsAction &&
		(!tableColumnsActionVisible(*behavior, localBounds) || behavior.scroll.disabled) {
		behavior.focusPart = TableFocusBody
	}
}

func normalizeTableSelectionStyle(
	style TableSelectionStyle,
	legacy CollectionSelectionMode,
) (TableSelectionStyle, CollectionSelectionMode, error) {
	if style == "" {
		mode, err := normalizeCollectionSelectionMode(legacy)
		if err != nil {
			return "", "", err
		}
		if mode == CollectionSelectionMultiple {
			return TableSelectionMultiple, mode, nil
		}
		return TableSelectionSingle, mode, nil
	}
	mode := CollectionSelectionSingle
	switch style {
	case TableSelectionNone, TableSelectionSingle:
	case TableSelectionRange, TableSelectionMultiple:
		mode = CollectionSelectionMultiple
	default:
		return "", "", fmt.Errorf("%w: invalid Table selection style %q", ErrValidation, style)
	}
	if legacy != "" {
		normalizedLegacy, err := normalizeCollectionSelectionMode(legacy)
		if err != nil {
			return "", "", err
		}
		if (style != TableSelectionSingle && style != TableSelectionMultiple) ||
			normalizedLegacy != mode {
			return "", "", fmt.Errorf("%w: conflicting Table selection style and legacy mode", ErrValidation)
		}
	}
	return style, mode, nil
}

func normalizeExplicitTableSelectionStyle(
	style TableSelectionStyle,
) (TableSelectionStyle, CollectionSelectionMode, error) {
	if style == "" {
		return "", "", fmt.Errorf("%w: Table selection policy requires a style", ErrValidation)
	}
	return normalizeTableSelectionStyle(style, "")
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

func normalizeTableColumnWrap(wrap TableColumnWrap) (TableColumnWrap, error) {
	if wrap == "" {
		wrap = TableColumnClip
	}
	switch wrap {
	case TableColumnClip, TableColumnWrapWords, TableColumnHang:
		return wrap, nil
	default:
		return "", fmt.Errorf("%w: invalid Table column wrap %q", ErrValidation, wrap)
	}
}

func normalizeTableColumnPresentation(
	columns []normalizedTableColumn,
	presentation []TableColumnPresentation,
) ([]TableColumnPresentation, error) {
	if presentation == nil {
		result := make([]TableColumnPresentation, len(columns))
		for index, column := range columns {
			result[index] = TableColumnPresentation{
				Column:  column.column.Key,
				Visible: true,
				Wrap:    TableColumnClip,
			}
		}
		return result, nil
	}
	if len(presentation) != len(columns) {
		return nil, fmt.Errorf("%w: Table column presentation must contain every column", ErrValidation)
	}
	known := make(map[string]bool, len(columns))
	for _, column := range columns {
		known[column.column.Key] = true
	}
	result := make([]TableColumnPresentation, len(presentation))
	seen := make(map[string]bool, len(presentation))
	visible := 0
	for index, entry := range presentation {
		if !known[entry.Column] || seen[entry.Column] {
			return nil, fmt.Errorf(
				"%w: unknown or duplicate Table presentation column %q",
				ErrValidation,
				entry.Column,
			)
		}
		wrap, err := normalizeTableColumnWrap(entry.Wrap)
		if err != nil {
			return nil, err
		}
		entry.Wrap = wrap
		result[index] = entry
		seen[entry.Column] = true
		if entry.Visible {
			visible++
		}
	}
	if len(columns) > 0 && visible == 0 {
		return nil, fmt.Errorf("%w: Table requires one visible column", ErrValidation)
	}
	return result, nil
}

func repairTableColumnPresentation(
	previous []TableColumnPresentation,
	columns []normalizedTableColumn,
) []TableColumnPresentation {
	known := make(map[string]bool, len(columns))
	for _, column := range columns {
		known[column.column.Key] = true
	}
	result := make([]TableColumnPresentation, 0, len(columns))
	seen := make(map[string]bool, len(columns))
	visible := 0
	for _, entry := range previous {
		if !known[entry.Column] || seen[entry.Column] {
			continue
		}
		result = append(result, entry)
		seen[entry.Column] = true
		if entry.Visible {
			visible++
		}
	}
	for _, column := range columns {
		if seen[column.column.Key] {
			continue
		}
		result = append(result, TableColumnPresentation{
			Column:  column.column.Key,
			Visible: true,
			Wrap:    TableColumnClip,
		})
		visible++
	}
	if len(columns) > 0 && visible == 0 {
		first := columns[0].column.Key
		for index := range result {
			if result[index].Column == first {
				result[index].Visible = true
				break
			}
		}
	}
	return result
}

func tableSchemaEqual(
	left []normalizedTableColumn,
	right []normalizedTableColumn,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !tableColumnEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

func tablePresentationEqual(
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

func tableColumnPresentationIndex(
	presentation []TableColumnPresentation,
	column string,
) int {
	for index, entry := range presentation {
		if entry.Column == column {
			return index
		}
	}
	return -1
}

func tableColumnIsVisible(behavior tableBehavior, column string) bool {
	index := tableColumnPresentationIndex(behavior.columnPresentation, column)
	return index >= 0 && behavior.columnPresentation[index].Visible
}

func visibleTableColumns(behavior tableBehavior) []int {
	result := make([]int, 0, len(behavior.columnPresentation))
	for _, entry := range behavior.columnPresentation {
		if !entry.Visible {
			continue
		}
		if index := tableColumnIndex(behavior.columns, entry.Column); index >= 0 {
			result = append(result, index)
		}
	}
	return result
}

func visibleTableColumnWidths(behavior tableBehavior) []int {
	indices := visibleTableColumns(behavior)
	result := make([]int, 0, len(indices))
	for _, index := range indices {
		if index >= 0 && index < len(behavior.columnWidths) {
			result = append(result, behavior.columnWidths[index])
		}
	}
	return result
}

func tableVisibleColumnIndex(behavior tableBehavior, key string) int {
	for visibleIndex, canonicalIndex := range visibleTableColumns(behavior) {
		if behavior.columns[canonicalIndex].column.Key == key {
			return visibleIndex
		}
	}
	return -1
}

func repairTableCurrentColumn(
	behavior *tableBehavior,
	previous []TableColumnPresentation,
) {
	if behavior == nil {
		return
	}
	if len(behavior.columns) == 0 {
		behavior.currentColumn = ""
		return
	}
	if tableColumnIsVisible(*behavior, behavior.currentColumn) {
		return
	}
	start := tableColumnPresentationIndex(
		behavior.columnPresentation,
		behavior.currentColumn,
	)
	if start < 0 {
		start = tableColumnPresentationIndex(previous, behavior.currentColumn)
	}
	if start < 0 {
		start = 0
	}
	if start >= len(behavior.columnPresentation) {
		start = len(behavior.columnPresentation) - 1
	}
	for index := start; index < len(behavior.columnPresentation); index++ {
		if behavior.columnPresentation[index].Visible {
			behavior.currentColumn = behavior.columnPresentation[index].Column
			return
		}
	}
	for index := start - 1; index >= 0; index-- {
		if behavior.columnPresentation[index].Visible {
			behavior.currentColumn = behavior.columnPresentation[index].Column
			return
		}
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
	rangeAnchor string,
	rangeExtent string,
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
	behavior.currentRow = currentRow
	behavior.currentColumn = currentColumn
	return setExactTableSelection(behavior, selected, rangeAnchor, rangeExtent)
}

func normalizeTableSelection(behavior *tableBehavior, selected []string) ([]string, error) {
	if behavior == nil {
		return nil, ErrInvalidControl
	}
	if behavior.selectionStyle == TableSelectionNone && len(selected) > 0 {
		return nil, fmt.Errorf("%w: no-selection Table has selected rows", ErrValidation)
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

func setExactTableSelection(
	behavior *tableBehavior,
	selected []string,
	anchor string,
	extent string,
) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	if behavior.selectionStyle == TableSelectionNone {
		if behavior.requireSelection || len(selected) > 0 || anchor != "" || extent != "" {
			return fmt.Errorf("%w: no-selection Table cannot retain selection state", ErrValidation)
		}
		behavior.selected = nil
		behavior.rangeAnchor, behavior.rangeExtent = "", ""
		return nil
	}
	if behavior.selectionStyle != TableSelectionRange {
		if anchor != "" || extent != "" {
			return fmt.Errorf("%w: Table range endpoints require range selection", ErrValidation)
		}
		ordered, err := normalizeTableSelection(behavior, selected)
		if err != nil {
			return err
		}
		if behavior.requireSelection && len(ordered) == 0 && behavior.currentRow != "" {
			ordered = []string{behavior.currentRow}
		}
		behavior.selected = ordered
		behavior.rangeAnchor, behavior.rangeExtent = "", ""
		return nil
	}
	if (anchor == "") != (extent == "") {
		return fmt.Errorf("%w: Table range endpoints must both be present or absent", ErrValidation)
	}
	if anchor != "" {
		derived, err := tableRangeSelection(behavior, anchor, extent)
		if err != nil {
			return err
		}
		if len(selected) > 0 {
			ordered, err := normalizeTableSelection(behavior, selected)
			if err != nil || !sameTableSelection(ordered, derived) {
				return fmt.Errorf("%w: Table range selection does not match its endpoints", ErrValidation)
			}
		}
		behavior.selected = derived
		behavior.rangeAnchor, behavior.rangeExtent = anchor, extent
		return nil
	}
	if len(selected) > 0 {
		ordered, err := normalizeTableSelection(behavior, selected)
		if err != nil {
			return err
		}
		derived, err := tableRangeSelection(behavior, ordered[0], ordered[len(ordered)-1])
		if err != nil || !sameTableSelection(ordered, derived) {
			return fmt.Errorf("%w: Table range selection must be one continuous enabled interval", ErrValidation)
		}
		behavior.selected = derived
		behavior.rangeAnchor, behavior.rangeExtent = ordered[0], ordered[len(ordered)-1]
		return nil
	}
	if behavior.requireSelection && behavior.currentRow != "" {
		behavior.selected = []string{behavior.currentRow}
		behavior.rangeAnchor, behavior.rangeExtent = behavior.currentRow, behavior.currentRow
		return nil
	}
	behavior.selected = nil
	behavior.rangeAnchor, behavior.rangeExtent = "", ""
	return nil
}

func tableRangeSelection(
	behavior *tableBehavior,
	anchor string,
	extent string,
) ([]string, error) {
	if behavior == nil {
		return nil, ErrInvalidControl
	}
	anchorIndex := tableDisplayRowIndex(*behavior, anchor)
	extentIndex := tableDisplayRowIndex(*behavior, extent)
	if anchorIndex < 0 || extentIndex < 0 {
		return nil, fmt.Errorf("%w: Table range endpoint is unavailable", ErrValidation)
	}
	anchorRow := tableCanonicalRowIndex(behavior.rows, anchor)
	extentRow := tableCanonicalRowIndex(behavior.rows, extent)
	if anchorRow < 0 || extentRow < 0 || behavior.rows[anchorRow].row.Disabled ||
		behavior.rows[extentRow].row.Disabled {
		return nil, fmt.Errorf("%w: Table range endpoint is unavailable", ErrValidation)
	}
	first, last := min(anchorIndex, extentIndex), max(anchorIndex, extentIndex)
	order := tableDisplayOrder(*behavior)
	result := make([]string, 0, last-first+1)
	for _, rowIndex := range order[first : last+1] {
		if !behavior.rows[rowIndex].row.Disabled {
			result = append(result, behavior.rows[rowIndex].row.Key)
		}
	}
	return result, nil
}

func sameTableSelection(left, right []string) bool {
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

func refreshTableSelection(behavior *tableBehavior) {
	if behavior == nil {
		return
	}
	if behavior.selectionStyle == TableSelectionRange {
		if behavior.rangeAnchor != "" && behavior.rangeExtent != "" {
			if selected, err := tableRangeSelection(
				behavior,
				behavior.rangeAnchor,
				behavior.rangeExtent,
			); err == nil {
				behavior.selected = selected
				return
			}
		}
		if behavior.requireSelection && behavior.currentRow != "" {
			behavior.selected = []string{behavior.currentRow}
			behavior.rangeAnchor, behavior.rangeExtent = behavior.currentRow, behavior.currentRow
		} else {
			behavior.selected = nil
			behavior.rangeAnchor, behavior.rangeExtent = "", ""
		}
		return
	}
	behavior.rangeAnchor, behavior.rangeExtent = "", ""
	behavior.selected, _ = normalizeTableSelection(behavior, behavior.selected)
	if behavior.requireSelection && len(behavior.selected) == 0 && behavior.currentRow != "" {
		behavior.selected = []string{behavior.currentRow}
	}
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
	cloned.rowHeights = append([]int(nil), behavior.rowHeights...)
	cloned.rowStarts = append([]int(nil), behavior.rowStarts...)
	cloned.features = append([]TableFeature(nil), behavior.features...)
	cloned.selected = append([]string(nil), behavior.selected...)
	cloned.columnPresentation = append(
		[]TableColumnPresentation(nil),
		behavior.columnPresentation...,
	)
	cloned.initialPresentation = append(
		[]TableColumnPresentation(nil),
		behavior.initialPresentation...,
	)
	cloned.statusMessage.lines = cloneTextRows(behavior.statusMessage.lines)
	return cloned
}

func tableBehaviorEqual(left, right tableBehavior) bool {
	if !scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.currentRow != right.currentRow || left.currentColumn != right.currentColumn ||
		left.selectionMode != right.selectionMode || left.selectionStyle != right.selectionStyle ||
		left.requireSelection != right.requireSelection ||
		left.rangeAnchor != right.rangeAnchor || left.rangeExtent != right.rangeExtent ||
		left.focusPart != right.focusPart ||
		left.schemaRevision != right.schemaRevision ||
		left.presentationRevision != right.presentationRevision ||
		left.focusMode != right.focusMode || left.sortColumn != right.sortColumn ||
		left.sortDirection != right.sortDirection || left.status != right.status ||
		left.statusMessage.text != right.statusMessage.text || left.changeCommand != right.changeCommand ||
		left.activateCommand != right.activateCommand || left.sortCommand != right.sortCommand ||
		len(left.columns) != len(right.columns) || len(left.rows) != len(right.rows) ||
		len(left.displayOrder) != len(right.displayOrder) ||
		len(left.columnWidths) != len(right.columnWidths) || len(left.selected) != len(right.selected) ||
		len(left.columnPresentation) != len(right.columnPresentation) ||
		len(left.initialPresentation) != len(right.initialPresentation) ||
		len(left.rowHeights) != len(right.rowHeights) ||
		len(left.rowStarts) != len(right.rowStarts) {
		return false
	}
	if len(left.features) != len(right.features) {
		return false
	}
	for index := range left.features {
		if left.features[index] != right.features[index] {
			return false
		}
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
	for index := range left.columnPresentation {
		if left.columnPresentation[index] != right.columnPresentation[index] {
			return false
		}
	}
	for index := range left.initialPresentation {
		if left.initialPresentation[index] != right.initialPresentation[index] {
			return false
		}
	}
	for index := range left.rowHeights {
		if left.rowHeights[index] != right.rowHeights[index] {
			return false
		}
	}
	for index := range left.rowStarts {
		if left.rowStarts[index] != right.rowStarts[index] {
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
	return b.scroll.controlClientRect(tableBodyRect(b, bounds))
}

func (b tableBehavior) intrinsicMinimum() Size {
	minimum := b.scroll.intrinsicMinimum()
	minimum.Height++
	if tableHasFeature(b, TableFeatureColumns) {
		minimum.Height += 2
	}
	return minimum
}

func (b tableBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	styles = append(styles,
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
	if tableHasFeature(b, TableFeatureColumns) {
		styles = append(styles,
			"button",
			"button.focused",
			"button.pressed",
			"button.disabled",
			"button.mnemonic",
			"button.shadow",
		)
	}
	return styles
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
	paintTableColumnsBand(app, frame, state, b, absolute, clip)
	paintTableColumnsAction(app, frame, state, b, absolute, clip)
	body := tableBodyRect(b, absolute)
	if !body.Empty() {
		b.scroll.paintDecoration(app, frame, state, body, clip)
	}
	geometry := calculateScrollViewGeometry(body.Size(), b.scroll)
	viewport := translatedRect(geometry.viewport, body.X, body.Y)
	visible := viewport.Intersect(clip)
	if visible.Width <= 0 || visible.Height <= 0 {
		return
	}
	for y := visible.Y; y < visible.Y+visible.Height; y++ {
		var cells []tablePaintCell
		bodyFocused := app.focus == state && b.focusPart == TableFocusBody
		if y == viewport.Y {
			cells = b.headerCells(bodyFocused)
		} else {
			sourceRow := b.scroll.state.Offset.Y + y - viewport.Y - 1
			cells = b.bodyCells(sourceRow, bodyFocused)
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
	visibleColumns := visibleTableColumns(b)
	for visibleIndex, columnIndex := range visibleColumns {
		column := b.columns[columnIndex]
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
		if visibleIndex+1 < len(visibleColumns) {
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
	displayIndex, within, ok := tableVisualRowAt(b, index)
	order := tableDisplayOrder(b)
	if !ok || displayIndex < 0 || displayIndex >= len(order) {
		return nil
	}
	row := b.rows[order[displayIndex]]
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
	if within == 0 && focused && current {
		marker = "►"
	}
	selectedMarker := " "
	if within == 0 && selected {
		selectedMarker = "X"
	}
	result := []tablePaintCell{
		{grapheme: marker, style: rowStyle},
		{grapheme: "[", style: rowStyle},
		{grapheme: selectedMarker, style: rowStyle},
		{grapheme: "]", style: rowStyle},
	}
	if b.selectionStyle == TableSelectionNone {
		result = []tablePaintCell{
			{grapheme: marker, style: rowStyle},
			{grapheme: " ", style: rowStyle},
			{grapheme: " ", style: rowStyle},
			{grapheme: " ", style: rowStyle},
		}
	}
	if within > 0 {
		result = []tablePaintCell{
			{grapheme: " ", style: rowStyle},
			{grapheme: " ", style: rowStyle},
			{grapheme: " ", style: rowStyle},
			{grapheme: " ", style: rowStyle},
		}
	}
	visibleColumns := visibleTableColumns(b)
	for visibleIndex, columnIndex := range visibleColumns {
		column := b.columns[columnIndex]
		text := []string{}
		if cell, found := tableCellForColumn(row, column.column.Key); found {
			text = cell.text.lines[0]
		}
		wrap := tableColumnWrapFor(b, column.column.Key)
		lines := wrapTableCellLines(text, b.columnWidths[columnIndex], wrap)
		line := tableWrappedLine{}
		if within >= 0 && within < len(lines) {
			line = lines[within]
		}
		aligned := alignedWrappedTableCells(
			line,
			b.columnWidths[columnIndex],
			column.column.Alignment,
		)
		cellStyle := rowStyle
		if !row.row.Disabled && focused && current && b.focusMode == TableFocusCell &&
			column.column.Key == b.currentColumn {
			cellStyle = "table.cell_current"
		}
		for _, grapheme := range aligned {
			result = append(result, tablePaintCell{grapheme: grapheme, style: cellStyle})
		}
		if visibleIndex+1 < len(visibleColumns) {
			result = append(result, tablePaintCell{grapheme: "│", style: rowStyle})
		}
	}
	return result
}

func tableColumnWrapFor(behavior tableBehavior, key string) TableColumnWrap {
	index := tableColumnPresentationIndex(behavior.columnPresentation, key)
	if index < 0 {
		return TableColumnClip
	}
	return behavior.columnPresentation[index].Wrap
}

func wrapTableCellLines(
	cells []string,
	width int,
	wrap TableColumnWrap,
) []tableWrappedLine {
	if width <= 0 || wrap == TableColumnClip || len(cells) <= width {
		return []tableWrappedLine{{cells: append([]string(nil), cells...)}}
	}
	remaining := append([]string(nil), cells...)
	result := make([]tableWrappedLine, 0, 1+len(cells)/max(1, width))
	first := true
	for len(remaining) > 0 {
		indent := 0
		if !first && wrap == TableColumnHang && width > 1 {
			indent = 1
		}
		available := max(1, width-indent)
		breakAt := min(available, len(remaining))
		if breakAt < len(remaining) {
			for candidate := breakAt - 1; candidate > 0; candidate-- {
				value, _ := utf8.DecodeRuneInString(remaining[candidate])
				if unicode.IsSpace(value) {
					breakAt = candidate
					break
				}
			}
		}
		line := append([]string(nil), remaining[:breakAt]...)
		for len(line) > 0 {
			value, _ := utf8.DecodeRuneInString(line[len(line)-1])
			if !unicode.IsSpace(value) {
				break
			}
			line = line[:len(line)-1]
		}
		result = append(result, tableWrappedLine{cells: line, indent: indent})
		remaining = remaining[breakAt:]
		for len(remaining) > 0 {
			value, _ := utf8.DecodeRuneInString(remaining[0])
			if !unicode.IsSpace(value) {
				break
			}
			remaining = remaining[1:]
		}
		first = false
	}
	if len(result) == 0 {
		return []tableWrappedLine{{}}
	}
	return result
}

func alignedWrappedTableCells(
	line tableWrappedLine,
	width int,
	alignment TextAlignment,
) []string {
	if line.indent <= 0 || width <= 1 {
		return alignedTableCells(line.cells, width, alignment)
	}
	result := []string{" "}
	return append(
		result,
		alignedTableCells(line.cells, width-1, alignment)...,
	)
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

func tableTotalWidth(behavior tableBehavior, widths []int) int {
	visible := visibleTableColumns(behavior)
	if len(visible) == 0 {
		return 1
	}
	total := 4 + len(visible) - 1
	for _, columnIndex := range visible {
		total = min(maxCoordinateMagnitude, total+widths[columnIndex])
	}
	return total
}

func growTableColumnWidths(behavior tableBehavior, base []int, viewportWidth int) []int {
	result := append([]int(nil), base...)
	remaining := viewportWidth - tableTotalWidth(behavior, result)
	visible := visibleTableColumns(behavior)
	for remaining > 0 {
		totalGrow := 0
		for _, index := range visible {
			column := behavior.columns[index]
			if column.column.Grow > 0 &&
				(column.column.MaximumWidth == 0 || result[index] < column.column.MaximumWidth) {
				totalGrow += column.column.Grow
			}
		}
		if totalGrow == 0 {
			break
		}
		changed := 0
		for _, index := range visible {
			column := behavior.columns[index]
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

func deriveTableRowGeometry(behavior *tableBehavior) int {
	if behavior == nil || behavior.status != CollectionReady || len(behavior.rows) == 0 {
		if behavior != nil {
			behavior.rowHeights = nil
			behavior.rowStarts = nil
		}
		return 1
	}
	order := tableDisplayOrder(*behavior)
	heights := make([]int, len(order))
	starts := make([]int, len(order))
	visibleColumns := visibleTableColumns(*behavior)
	wraps := make(map[int]TableColumnWrap, len(visibleColumns))
	for _, columnIndex := range visibleColumns {
		wraps[columnIndex] = tableColumnWrapFor(
			*behavior,
			behavior.columns[columnIndex].column.Key,
		)
	}
	total := 0
	for displayIndex, canonicalIndex := range order {
		starts[displayIndex] = total
		height := 1
		row := behavior.rows[canonicalIndex]
		for _, columnIndex := range visibleColumns {
			column := behavior.columns[columnIndex]
			cells := []string{}
			if cell, found := tableCellForColumn(row, column.column.Key); found {
				cells = cell.text.lines[0]
			}
			height = max(
				height,
				len(wrapTableCellLines(
					cells,
					behavior.columnWidths[columnIndex],
					wraps[columnIndex],
				)),
			)
		}
		height = min(height, maxCoordinateMagnitude-1)
		heights[displayIndex] = height
		total = min(maxCoordinateMagnitude-1, saturatingAdd(total, height))
	}
	behavior.rowHeights = heights
	behavior.rowStarts = starts
	return total
}

func tableVisualRowCount(behavior tableBehavior) int {
	if behavior.status != CollectionReady || len(behavior.rows) == 0 {
		return 1
	}
	if len(behavior.rowHeights) == 0 || len(behavior.rowStarts) == 0 {
		copyValue := behavior
		return deriveTableRowGeometry(&copyValue)
	}
	last := len(behavior.rowHeights) - 1
	return min(
		maxCoordinateMagnitude-1,
		saturatingAdd(behavior.rowStarts[last], behavior.rowHeights[last]),
	)
}

func tableVisualRowAt(
	behavior tableBehavior,
	visualIndex int,
) (displayIndex int, within int, ok bool) {
	if visualIndex < 0 || len(behavior.rowHeights) == 0 ||
		len(behavior.rowStarts) != len(behavior.rowHeights) {
		return -1, 0, false
	}
	index := sort.Search(len(behavior.rowStarts), func(index int) bool {
		return saturatingAdd(
			behavior.rowStarts[index],
			behavior.rowHeights[index],
		) > visualIndex
	})
	if index >= len(behavior.rowStarts) || visualIndex < behavior.rowStarts[index] {
		return -1, 0, false
	}
	return index, visualIndex - behavior.rowStarts[index], true
}

func tableDisplayRowVisualStart(behavior tableBehavior, displayIndex int) int {
	if displayIndex < 0 || displayIndex >= len(behavior.rowStarts) {
		return -1
	}
	return behavior.rowStarts[displayIndex]
}

func invalidateTableGeometry(behavior *tableBehavior) {
	if behavior == nil {
		return
	}
	behavior.geometryRevision++
	if behavior.geometryRevision == 0 {
		behavior.geometryRevision = 1
		behavior.geometryCachedAt = 0
	}
}

func reflowTable(behavior tableBehavior, size Size) tableBehavior {
	repairTableFocusPart(&behavior, size)
	bodySize := tableBodySize(behavior, size)
	if behavior.geometryCachedAt != behavior.geometryRevision ||
		behavior.geometryCacheSize != bodySize {
		behavior.displayOrder = nil
		behavior.displayOrder = deriveTableDisplayOrder(behavior)
		baseWidths := tableBaseColumnWidths(behavior)
		behavior.columnWidths = append([]int(nil), baseWidths...)
		bodyRows := deriveTableRowGeometry(&behavior)
		behavior.scroll.state.ContentSize = Size{
			Width: tableTotalWidth(behavior, behavior.columnWidths), Height: bodyRows + 1,
		}
		for range 4 {
			geometry := calculateScrollViewGeometry(bodySize, behavior.scroll)
			next := growTableColumnWidths(behavior, baseWidths, geometry.viewport.Width)
			nextWidth := tableTotalWidth(behavior, next)
			behavior.columnWidths = next
			nextRows := deriveTableRowGeometry(&behavior)
			nextHeight := nextRows + 1
			if nextWidth == behavior.scroll.state.ContentSize.Width &&
				nextHeight == behavior.scroll.state.ContentSize.Height {
				break
			}
			behavior.scroll.state.ContentSize.Width = nextWidth
			behavior.scroll.state.ContentSize.Height = nextHeight
		}
		behavior.geometryCachedAt = behavior.geometryRevision
		behavior.geometryCacheSize = bodySize
	}
	geometry := calculateScrollViewGeometry(bodySize, behavior.scroll)
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	if behavior.status == CollectionReady {
		current := tableDisplayRowIndex(behavior, behavior.currentRow)
		currentStart := tableDisplayRowVisualStart(behavior, current)
		bodyHeight := max(0, geometry.viewport.Height-1)
		if currentStart >= 0 && bodyHeight > 0 {
			currentHeight := behavior.rowHeights[current]
			if currentStart < behavior.scroll.state.Offset.Y {
				behavior.scroll.state.Offset.Y = currentStart
			} else if currentStart >= behavior.scroll.state.Offset.Y+bodyHeight {
				behavior.scroll.state.Offset.Y = currentStart
			} else if currentHeight <= bodyHeight &&
				currentStart+currentHeight > behavior.scroll.state.Offset.Y+bodyHeight {
				behavior.scroll.state.Offset.Y = currentStart + currentHeight - bodyHeight
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
	for _, index := range visibleTableColumns(behavior) {
		column := behavior.columns[index]
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
		if behavior.geometryRevision != next.geometryRevision ||
			behavior.geometryCachedAt != next.geometryCachedAt ||
			behavior.geometryCacheSize != next.geometryCacheSize {
			state.behavior = next
		}
		return false
	}
	state.behavior = next
	return true
}

func tableCanFocus(state *controlState, behavior tableBehavior) bool {
	if state == nil || behavior.scroll.disabled {
		return false
	}
	action := tableColumnsActionVisible(
		behavior,
		Rect{Width: state.bounds.Width, Height: state.bounds.Height},
	)
	return tableBodyCanFocus(behavior) || action
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

func visibleTableColumnCount(presentation []TableColumnPresentation) int {
	count := 0
	for _, entry := range presentation {
		if entry.Visible {
			count++
		}
	}
	return count
}

func visibleTableColumnRange(
	presentation []TableColumnPresentation,
) (string, string) {
	first, last := "", ""
	for _, entry := range presentation {
		if !entry.Visible {
			continue
		}
		if first == "" {
			first = entry.Column
		}
		last = entry.Column
	}
	return first, last
}

func tableColumnPresentationDigest(presentation []TableColumnPresentation) string {
	hash := sha256.New()
	var length [2]byte
	for _, entry := range presentation {
		binary.BigEndian.PutUint16(length[:], uint16(len(entry.Column)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(entry.Column))
		if entry.Visible {
			_, _ = hash.Write([]byte{1})
		} else {
			_, _ = hash.Write([]byte{0})
		}
		binary.BigEndian.PutUint16(length[:], uint16(len(entry.Wrap)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(entry.Wrap))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func tableDetails(
	bounds Rect,
	behavior tableBehavior,
	actionVisible bool,
	actionPressed bool,
	dialogOpen bool,
) TableDetails {
	viewport := scrollViewDetails(tableBodyRect(behavior, bounds), behavior.scroll)
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
	firstVisibleColumn, lastVisibleColumn := visibleTableColumnRange(
		behavior.columnPresentation,
	)
	localBounds := Rect{Width: bounds.Width, Height: bounds.Height}
	actionBounds := tableColumnsActionBounds(behavior, localBounds)
	return TableDetails{
		Status: behavior.status, StatusMessage: behavior.statusMessage.text,
		Features: append([]TableFeature(nil), behavior.features...),
		RowCount: len(behavior.rows), VisualRowCount: tableVisualRowCount(behavior),
		EnabledCount: enabledTableRowCount(behavior.rows),
		ColumnCount:  len(behavior.columns), CellCount: tableCellCount(behavior.rows),
		RetainedBytes: tableStorageBytes(behavior), CurrentRow: behavior.currentRow,
		CurrentRowIndex:    tableDisplayRowIndex(behavior, behavior.currentRow),
		CurrentColumn:      behavior.currentColumn,
		CurrentColumnIndex: tableVisibleColumnIndex(behavior, behavior.currentColumn),
		FocusMode:          behavior.focusMode, SelectionMode: behavior.selectionMode,
		SelectionStyle: behavior.selectionStyle, RangeAnchor: behavior.rangeAnchor,
		RangeExtent:      behavior.rangeExtent,
		RequireSelection: behavior.requireSelection, SelectedCount: len(behavior.selected),
		FirstSelected: firstSelected, LastSelected: lastSelected,
		SelectionDigest: listSelectionDigest(behavior.selected),
		SortColumn:      behavior.sortColumn, SortDirection: behavior.sortDirection,
		FirstColumn: firstColumn, LastColumn: lastColumn,
		ColumnPresentation: append(
			[]TableColumnPresentation(nil),
			behavior.columnPresentation...,
		),
		VisibleColumnCount:   visibleTableColumnCount(behavior.columnPresentation),
		FirstVisibleColumn:   firstVisibleColumn,
		LastVisibleColumn:    lastVisibleColumn,
		PresentationDigest:   tableColumnPresentationDigest(behavior.columnPresentation),
		FocusPart:            behavior.focusPart,
		ColumnsActionBounds:  actionBounds,
		ColumnsActionVisible: actionVisible,
		ColumnsActionEnabled: actionVisible && !behavior.scroll.disabled && !dialogOpen,
		ColumnsActionPressed: actionPressed,
		ColumnsDialogOpen:    dialogOpen,
		ColumnWidths:         visibleTableColumnWidths(behavior),
		ColumnWidthsDigest:   integerSequenceDigest(visibleTableColumnWidths(behavior)),
		Enabled:              !behavior.scroll.disabled, DisabledReason: behavior.scroll.disabledReason,
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

// ColumnPresentation returns a caller-owned copy in display order, including
// hidden columns.
func (t *Table) ColumnPresentation() []TableColumnPresentation {
	if t == nil || t.state == nil || t.state.app == nil {
		return nil
	}
	t.state.app.mu.RLock()
	defer t.state.app.mu.RUnlock()
	behavior, ok := t.state.behavior.(tableBehavior)
	if !ok || t.state.destroyed || t.state.aborted {
		return nil
	}
	return append([]TableColumnPresentation(nil), behavior.columnPresentation...)
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
		Features:           append([]TableFeature(nil), behavior.features...),
		CurrentRow:         behavior.currentRow,
		CurrentRowIndex:    tableDisplayRowIndex(behavior, behavior.currentRow),
		CurrentColumn:      behavior.currentColumn,
		CurrentColumnIndex: tableVisibleColumnIndex(behavior, behavior.currentColumn),
		Selected:           append([]string(nil), behavior.selected...), FocusMode: behavior.focusMode,
		SelectionStyle: behavior.selectionStyle,
		RangeAnchor:    behavior.rangeAnchor,
		RangeExtent:    behavior.rangeExtent,
		ColumnPresentation: append(
			[]TableColumnPresentation(nil),
			behavior.columnPresentation...,
		),
		VisibleColumnCount: visibleTableColumnCount(behavior.columnPresentation),
		VisualRowCount:     tableVisualRowCount(behavior),
		FocusPart:          behavior.focusPart,
		ColumnsDialogOpen:  t.state.app.tableColumnsDialogOpenLocked(t.state),
		SortColumn:         behavior.sortColumn, SortDirection: behavior.sortDirection,
		Offset: behavior.scroll.state.Offset, RowCount: len(behavior.rows),
		EnabledCount: enabledTableRowCount(behavior.rows), ColumnCount: len(behavior.columns),
		CellCount: tableCellCount(behavior.rows), ColumnWidths: visibleTableColumnWidths(behavior),
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

// ReplaceWithPresentation atomically replaces the model, exact column
// presentation, current coordinate, selection, and sort.
func (t *Table) ReplaceWithPresentation(
	columns []Column,
	rows []TableRow,
	presentation []TableColumnPresentation,
	currentRow string,
	currentColumn string,
	selected []string,
	sortColumn string,
	sortDirection SortDirection,
) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.ReplaceTableWithPresentation(
			t,
			columns,
			rows,
			presentation,
			currentRow,
			currentColumn,
			selected,
			sortColumn,
			sortDirection,
		)
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

// SetSelectionPolicy atomically changes the selection behavior and exact
// retained row selection.
func (t *Table) SetSelectionPolicy(policy TableSelectionPolicy) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.SetTableSelectionPolicy(t, policy)
	})
}

// SetFeatures atomically replaces the complete optional Table feature set.
func (t *Table) SetFeatures(features []TableFeature) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.SetTableFeatures(t, features)
	})
}

// SetColumnPresentation atomically changes the exact display order,
// visibility, and body wrapping policy for every column.
func (t *Table) SetColumnPresentation(
	presentation []TableColumnPresentation,
) error {
	return commitTableMutation(t, func(tx *Transaction) error {
		return tx.SetTableColumnPresentation(t, presentation)
	})
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
	command, target, _, changed := a.tableKeyLocked(state, KeyEnter, false, false)
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
	shift bool,
) (CommandID, ControlID, bool, bool) {
	if state == nil {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(tableBehavior)
	if !ok || !tableCanFocus(state, behavior) {
		return "", "", false, false
	}
	if behavior.focusPart == TableFocusColumnsAction {
		if !control && !shift && (key == KeyEnter || key == KeySpace) {
			return CommandTableColumnsOpen, state.id, true, false
		}
		return "", "", false, false
	}
	before := cloneTableBehavior(behavior)
	handled, selectionChanged, activate, sortChanged := true, false, false, false
	current := tableDisplayRowIndex(behavior, behavior.currentRow)
	geometry := calculateScrollViewGeometry(
		tableBodySize(behavior, state.bounds.Size()),
		behavior.scroll,
	)
	if shift {
		selectionChanged, handled = extendTableRangeSelection(&behavior, key, geometry)
	} else if control && (key == KeyHome || key == KeyEnd) {
		visibleColumns := visibleTableColumns(behavior)
		if key == KeyHome {
			behavior.currentRow = firstEnabledTableRowKey(&behavior)
			if len(visibleColumns) > 0 {
				behavior.currentColumn = behavior.columns[visibleColumns[0]].column.Key
			}
		} else {
			behavior.currentRow = lastEnabledTableRowKey(behavior)
			if len(visibleColumns) > 0 {
				behavior.currentColumn = behavior.columns[visibleColumns[len(visibleColumns)-1]].column.Key
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
				columns := visibleTableColumns(behavior)
				column := tableVisibleColumnIndex(behavior, behavior.currentColumn)
				if column > 0 {
					behavior.currentColumn = behavior.columns[columns[column-1]].column.Key
				}
			} else {
				behavior.scroll.state.Offset.X -= behavior.scroll.arrowStep.Width
			}
		case KeyRight:
			if behavior.focusMode == TableFocusCell {
				columns := visibleTableColumns(behavior)
				column := tableVisibleColumnIndex(behavior, behavior.currentColumn)
				if column >= 0 && column+1 < len(columns) {
					behavior.currentColumn = behavior.columns[columns[column+1]].column.Key
				}
			} else {
				behavior.scroll.state.Offset.X += behavior.scroll.arrowStep.Width
			}
		case KeySpace:
			selectionChanged = selectCurrentTableRow(&behavior, true)
		case KeyEnter:
			selectionChanged = selectCurrentTableRow(&behavior, false)
			activate = true
		case Key("["), Key("]"):
			selectionChanged, handled = stepTableRangeExtent(&behavior, key)
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
	if len(order) == 0 || current < 0 || current >= len(order) || delta == 0 {
		return ""
	}
	currentStart := tableDisplayRowVisualStart(behavior, current)
	if currentStart < 0 {
		return ""
	}
	targetVisual := min(
		max(0, tableVisualRowCount(behavior)-1),
		max(0, currentStart+delta),
	)
	target := sort.Search(len(behavior.rowStarts), func(index int) bool {
		return behavior.rowStarts[index] >= targetVisual
	})
	if delta < 0 {
		if target >= len(behavior.rowStarts) || behavior.rowStarts[target] > targetVisual {
			target--
		}
		if target >= current {
			target = current - 1
		}
		for index := target; index >= 0; index-- {
			if !behavior.rows[order[index]].row.Disabled {
				return behavior.rows[order[index]].row.Key
			}
		}
		return firstEnabledTableRowKey(&behavior)
	}
	if target <= current {
		target = current + 1
	}
	for index := target; index < len(order); index++ {
		if !behavior.rows[order[index]].row.Disabled {
			return behavior.rows[order[index]].row.Key
		}
	}
	return lastEnabledTableRowKey(behavior)
}

func selectCurrentTableRow(behavior *tableBehavior, toggle bool) bool {
	if behavior == nil || behavior.currentRow == "" {
		return false
	}
	if behavior.selectionStyle == TableSelectionNone {
		return false
	}
	if behavior.selectionStyle == TableSelectionRange {
		if !toggle && listSelectionContains(behavior.selected, behavior.currentRow) {
			return false
		}
		changed := len(behavior.selected) != 1 || behavior.selected[0] != behavior.currentRow ||
			behavior.rangeAnchor != behavior.currentRow || behavior.rangeExtent != behavior.currentRow
		behavior.selected = []string{behavior.currentRow}
		behavior.rangeAnchor, behavior.rangeExtent = behavior.currentRow, behavior.currentRow
		return changed
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

func extendTableRangeSelection(
	behavior *tableBehavior,
	key Key,
	geometry scrollViewGeometry,
) (bool, bool) {
	if behavior == nil || behavior.selectionStyle != TableSelectionRange ||
		behavior.currentRow == "" {
		return false, false
	}
	anchor := behavior.rangeAnchor
	if anchor == "" {
		anchor = behavior.currentRow
	}
	current := tableDisplayRowIndex(*behavior, behavior.currentRow)
	next := behavior.currentRow
	switch key {
	case KeyUp:
		next = previousEnabledTableRowKey(*behavior, current)
	case KeyDown:
		next = nextEnabledTableRowKey(*behavior, current)
	case KeyPageUp:
		next = pageEnabledTableRowKey(
			*behavior,
			current,
			-effectiveScrollPageStep(behavior.scroll.pageStep.Height, max(1, geometry.viewport.Height-1)),
		)
	case KeyPageDown:
		next = pageEnabledTableRowKey(
			*behavior,
			current,
			effectiveScrollPageStep(behavior.scroll.pageStep.Height, max(1, geometry.viewport.Height-1)),
		)
	case KeyHome:
		next = firstEnabledTableRowKey(behavior)
	case KeyEnd:
		next = lastEnabledTableRowKey(*behavior)
	default:
		return false, false
	}
	before := append([]string(nil), behavior.selected...)
	oldAnchor, oldExtent := behavior.rangeAnchor, behavior.rangeExtent
	behavior.currentRow = next
	behavior.rangeAnchor, behavior.rangeExtent = anchor, next
	behavior.selected, _ = tableRangeSelection(behavior, anchor, next)
	return !sameTableSelection(before, behavior.selected) || oldAnchor != anchor || oldExtent != next, true
}

func stepTableRangeExtent(behavior *tableBehavior, key Key) (bool, bool) {
	if behavior == nil || behavior.selectionStyle != TableSelectionRange ||
		behavior.currentRow == "" {
		return false, false
	}
	anchor, extent := behavior.rangeAnchor, behavior.rangeExtent
	if anchor == "" {
		anchor, extent = behavior.currentRow, behavior.currentRow
	}
	current := tableDisplayRowIndex(*behavior, extent)
	next := extent
	if key == Key("[") {
		next = previousEnabledTableRowKey(*behavior, current)
	} else if key == Key("]") {
		next = nextEnabledTableRowKey(*behavior, current)
	} else {
		return false, false
	}
	before := append([]string(nil), behavior.selected...)
	oldAnchor, oldExtent := behavior.rangeAnchor, behavior.rangeExtent
	behavior.currentRow = next
	behavior.rangeAnchor, behavior.rangeExtent = anchor, next
	behavior.selected, _ = tableRangeSelection(behavior, anchor, next)
	return !sameTableSelection(before, behavior.selected) || oldAnchor != anchor || oldExtent != next, true
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
	invalidateTableGeometry(behavior)
	refreshTableSelection(behavior)
	return true
}

func tableStorageBytes(behavior tableBehavior) int {
	total := len(behavior.statusMessage.text) + len(behavior.currentRow) +
		len(behavior.currentColumn) + len(behavior.sortColumn) +
		len(behavior.rangeAnchor) + len(behavior.rangeExtent)
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
	for _, entry := range behavior.columnPresentation {
		total += len(entry.Column) + len(entry.Wrap)
	}
	for _, entry := range behavior.initialPresentation {
		total += len(entry.Column) + len(entry.Wrap)
	}
	for _, feature := range behavior.features {
		total += len(feature)
	}
	// Derived integer caches remain bounded by the same aggregate collection
	// budget as their source model. Eight bytes is a conservative per-int
	// accounting on supported targets.
	total += 8 * (len(behavior.displayOrder) + len(behavior.columnWidths) +
		len(behavior.rowHeights) + len(behavior.rowStarts))
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
	basePresentation := behavior.presentationRevision
	if behavior.presentationMutation {
		basePresentation--
	}
	mutation := transactionMutation{
		kind: mutationTable, state: state,
		tablePresentationMutation:     behavior.presentationMutation,
		tableBasePresentationRevision: basePresentation,
	}
	behavior.presentationMutation = false
	mutation.behavior = behavior
	t.mutations = append(t.mutations, mutation)
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
	oldPositions := tableIdentityPositionsFor(behavior)
	behavior.rows = normalized
	behavior.displayOrder = nil
	invalidateTableGeometry(&behavior)
	repairTableIdentity(&behavior, oldPositions)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableModel records a copied schema/model replacement preserving state.
func (t *Transaction) SetTableModel(control *Table, columns []Column, rows []TableRow) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	oldPositions := tableIdentityPositionsFor(behavior)
	previousPresentation := behavior.columnPresentation
	normalizedColumns, err := normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	normalizedRows, err := normalizeTableRows(normalizedColumns, rows)
	if err != nil {
		return err
	}
	schemaChanged := !tableSchemaEqual(behavior.columns, normalizedColumns)
	behavior.columns, behavior.rows = normalizedColumns, normalizedRows
	behavior.columnPresentation = repairTableColumnPresentation(
		previousPresentation,
		behavior.columns,
	)
	if schemaChanged {
		behavior.initialPresentation = repairTableColumnPresentation(
			behavior.initialPresentation,
			behavior.columns,
		)
		behavior.schemaRevision++
	}
	behavior.displayOrder = nil
	invalidateTableGeometry(&behavior)
	if tableColumnIndex(behavior.columns, behavior.sortColumn) < 0 ||
		(behavior.sortColumn != "" && !behavior.columns[tableColumnIndex(behavior.columns, behavior.sortColumn)].column.Sortable) {
		behavior.sortColumn, behavior.sortDirection = "", SortNone
	}
	repairTableCurrentColumn(&behavior, previousPresentation)
	repairTableIdentity(&behavior, oldPositions)
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
	previousPresentation := behavior.columnPresentation
	normalizedColumns, err := normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	schemaChanged := !tableSchemaEqual(behavior.columns, normalizedColumns)
	behavior.columns = normalizedColumns
	behavior.rows, err = normalizeTableRows(behavior.columns, rows)
	if err != nil {
		return err
	}
	behavior.columnPresentation = repairTableColumnPresentation(
		previousPresentation,
		behavior.columns,
	)
	if schemaChanged {
		behavior.initialPresentation = repairTableColumnPresentation(
			behavior.initialPresentation,
			behavior.columns,
		)
		behavior.schemaRevision++
	}
	behavior.displayOrder = nil
	invalidateTableGeometry(&behavior)
	if err := setExactTableSort(&behavior, sortColumn, sortDirection); err != nil {
		return err
	}
	if err := setExactTableIdentity(&behavior, currentRow, currentColumn, selected, "", ""); err != nil {
		return err
	}
	repairTableCurrentColumn(&behavior, previousPresentation)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// ReplaceTableWithPresentation records one exact complete Table replacement,
// including copied column presentation.
func (t *Transaction) ReplaceTableWithPresentation(
	control *Table,
	columns []Column,
	rows []TableRow,
	presentation []TableColumnPresentation,
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
	previousPresentation := behavior.columnPresentation
	normalizedColumns, err := normalizeTableColumns(columns)
	if err != nil {
		return err
	}
	schemaChanged := !tableSchemaEqual(behavior.columns, normalizedColumns)
	behavior.columns = normalizedColumns
	behavior.rows, err = normalizeTableRows(behavior.columns, rows)
	if err != nil {
		return err
	}
	if schemaChanged {
		behavior.initialPresentation = repairTableColumnPresentation(
			behavior.initialPresentation,
			behavior.columns,
		)
		behavior.schemaRevision++
	}
	if !tablePresentationEqual(previousPresentation, behavior.columnPresentation) {
		behavior.presentationRevision++
		behavior.presentationMutation = true
	}
	behavior.columnPresentation, err = normalizeTableColumnPresentation(
		behavior.columns,
		presentation,
	)
	if err != nil {
		return err
	}
	behavior.displayOrder = nil
	invalidateTableGeometry(&behavior)
	if err := setExactTableSort(&behavior, sortColumn, sortDirection); err != nil {
		return err
	}
	if err := setExactTableIdentity(
		&behavior,
		currentRow,
		currentColumn,
		selected,
		"",
		"",
	); err != nil {
		return err
	}
	repairTableCurrentColumn(&behavior, previousPresentation)
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
	} else if tableColumnIndex(behavior.columns, column) < 0 ||
		!tableColumnIsVisible(behavior, column) {
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
	if behavior.requireSelection && len(selected) == 0 && behavior.currentRow != "" {
		return fmt.Errorf("%w: Table selection is required", ErrValidation)
	}
	if err := setExactTableSelection(&behavior, selected, "", ""); err != nil {
		return err
	}
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableSelectionPolicy records one exact atomic selection policy.
func (t *Transaction) SetTableSelectionPolicy(
	control *Table,
	policy TableSelectionPolicy,
) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	style, mode, err := normalizeExplicitTableSelectionStyle(policy.Style)
	if err != nil {
		return err
	}
	behavior.selectionStyle, behavior.selectionMode = style, mode
	behavior.requireSelection = policy.Require
	if err := setExactTableSelection(
		&behavior,
		policy.Selected,
		policy.Anchor,
		policy.Extent,
	); err != nil {
		return err
	}
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableFeatures records one exact copied optional Table feature set.
func (t *Transaction) SetTableFeatures(
	control *Table,
	features []TableFeature,
) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeTableFeatures(features)
	if err != nil {
		return err
	}
	behavior.features = normalized
	invalidateTableGeometry(&behavior)
	repairTableFocusPart(&behavior, target.bounds.Size())
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

// SetTableColumnPresentation records one exact copied column presentation.
func (t *Transaction) SetTableColumnPresentation(
	control *Table,
	presentation []TableColumnPresentation,
) error {
	target, behavior, err := t.selectedTable(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeTableColumnPresentation(
		behavior.columns,
		presentation,
	)
	if err != nil {
		return err
	}
	previous := behavior.columnPresentation
	behavior.columnPresentation = normalized
	if !tablePresentationEqual(previous, normalized) {
		behavior.presentationRevision++
		behavior.presentationMutation = true
	}
	invalidateTableGeometry(&behavior)
	repairTableCurrentColumn(&behavior, previous)
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
	invalidateTableGeometry(&behavior)
	refreshTableSelection(&behavior)
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
	invalidateTableGeometry(&behavior)
	behavior = reflowTable(behavior, target.bounds.Size())
	return t.recordTable(target, behavior)
}

type tableIdentityPositions struct {
	current int
	anchor  int
	extent  int
}

func tableIdentityPositionsFor(behavior tableBehavior) tableIdentityPositions {
	return tableIdentityPositions{
		current: tableDisplayRowIndex(behavior, behavior.currentRow),
		anchor:  tableDisplayRowIndex(behavior, behavior.rangeAnchor),
		extent:  tableDisplayRowIndex(behavior, behavior.rangeExtent),
	}
}

func repairTableIdentity(behavior *tableBehavior, old tableIdentityPositions) {
	if behavior == nil {
		return
	}
	currentIndex := tableCanonicalRowIndex(behavior.rows, behavior.currentRow)
	if currentIndex < 0 || behavior.rows[currentIndex].row.Disabled {
		behavior.currentRow = repairedTableCurrent(*behavior, old.current)
	}
	if behavior.selectionStyle == TableSelectionRange {
		anchorSurvives := enabledTableRowKey(behavior, behavior.rangeAnchor)
		extentSurvives := enabledTableRowKey(behavior, behavior.rangeExtent)
		if !anchorSurvives && !extentSurvives {
			if behavior.requireSelection && behavior.currentRow != "" {
				behavior.rangeAnchor, behavior.rangeExtent = behavior.currentRow, behavior.currentRow
				behavior.selected = []string{behavior.currentRow}
			} else {
				behavior.rangeAnchor, behavior.rangeExtent = "", ""
				behavior.selected = nil
			}
			return
		}
		if !anchorSurvives {
			behavior.rangeAnchor = repairedTableCurrent(*behavior, old.anchor)
			if behavior.rangeAnchor == "" {
				behavior.rangeAnchor = behavior.rangeExtent
			}
		}
		if !extentSurvives {
			behavior.rangeExtent = repairedTableCurrent(*behavior, old.extent)
			if behavior.rangeExtent == "" {
				behavior.rangeExtent = behavior.rangeAnchor
			}
		}
		refreshTableSelection(behavior)
		return
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

func enabledTableRowKey(behavior *tableBehavior, key string) bool {
	if behavior == nil || key == "" {
		return false
	}
	index := tableCanonicalRowIndex(behavior.rows, key)
	return index >= 0 && !behavior.rows[index].row.Disabled
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
