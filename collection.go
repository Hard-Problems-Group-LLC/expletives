package expletives

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// CollectionStatus is the explicit model state of a collection control.
type CollectionStatus string

const (
	CollectionReady   CollectionStatus = "ready"
	CollectionLoading CollectionStatus = "loading"
	CollectionError   CollectionStatus = "error"
)

// CollectionSelectionMode selects single or multiple stable-key selection.
type CollectionSelectionMode string

const (
	CollectionSelectionSingle   CollectionSelectionMode = "single"
	CollectionSelectionMultiple CollectionSelectionMode = "multiple"
)

// CollectionSelectionMarks controls whether a collection paints explicit
// `[ ]`/`[X]` marks. The zero value preserves the visible-marker default.
type CollectionSelectionMarks string

const (
	CollectionSelectionMarksDefault CollectionSelectionMarks = ""
	CollectionSelectionMarksShow    CollectionSelectionMarks = "show"
	CollectionSelectionMarksHide    CollectionSelectionMarks = "hide"
)

// ListItem is one copied stable-identity ListBox or popup item.
type ListItem struct {
	Key         string
	Label       string
	Description string
	// Indicator is an optional one-cell leading display element painted with
	// IndicatorStyle. Both fields must be empty or both must be supplied.
	Indicator      string
	IndicatorStyle StyleID
	Disabled       bool
	DisabledReason string
}

// ListBoxOptions configures one bounded stable-identity scrolling list.
//
// ContentAutomationKey, ContentStyle, and State.ContentSize are reserved by
// ListBox and must be empty. State.Offset is accepted as an initial request
// and clamped while keeping Current visible.
type ListBoxOptions struct {
	ScrollablePanelOptions
	Items            []ListItem
	Current          string
	Selected         []string
	SelectionMode    CollectionSelectionMode
	SelectionMarks   CollectionSelectionMarks
	RequireSelection bool
	Status           CollectionStatus
	StatusMessage    string
	// Wrap optionally reflows each logical item into one or more visual rows.
	// A label-plus-description row uses the description start as its hanging
	// indent. The zero value preserves the original one-row presentation.
	Wrap            TextWrap
	CurrentCommand  CommandID
	ActivateCommand CommandID
}

// ListBoxState is one complete copied semantic and viewport state.
type ListBoxState struct {
	Status         CollectionStatus
	StatusMessage  string
	Current        string
	CurrentIndex   int
	Selected       []string
	SelectionMarks bool
	Offset         Point
	ItemCount      int
	EnabledCount   int
	VisualRowCount int
	Wrap           TextWrap
}

// ListBox is a copy-safe focusable stable-identity collection leaf.
type ListBox struct{ controlHandle }

type normalizedListItem struct {
	item        ListItem
	label       normalizedDisplayText
	description normalizedDisplayText
	indicator   normalizedDisplayText
}

type listBoxVisualRow struct {
	itemIndex     int
	within        int
	indicatorCell int
	cells         []string
}

type collectionPaintCell struct {
	grapheme string
	style    StyleID
}

type listBoxBehavior struct {
	scroll           scrollViewBehavior
	items            []normalizedListItem
	current          string
	selected         []string
	selectionMode    CollectionSelectionMode
	selectionMarks   bool
	requireSelection bool
	status           CollectionStatus
	statusMessage    normalizedDisplayText
	wrap             TextWrap
	rows             []listBoxVisualRow
	changeCommand    CommandID
	currentCommand   CommandID
	activateCommand  CommandID
}

// NewListBox constructs and atomically inserts a ListBox.
func NewListBox(parent Container, options ListBoxOptions) (*ListBox, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewListBox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewListBox records construction of a provisional ListBox.
func (t *Transaction) NewListBox(
	parent Container,
	options ListBoxOptions,
) (*ListBox, error) {
	behavior, err := newListBoxBehavior(options)
	if err != nil {
		return nil, err
	}
	behavior = reflowListBox(behavior, options.Bounds.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlListBox,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &ListBox{controlHandle: controlHandle{state: state}}
	state.control = control
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func newListBoxBehavior(options ListBoxOptions) (listBoxBehavior, error) {
	scrollOptions := options.ScrollViewOptions
	if scrollOptions.ContentAutomationKey != "" ||
		scrollOptions.ContentStyle != "" ||
		scrollOptions.State.ContentSize != (Size{}) {
		return listBoxBehavior{}, fmt.Errorf(
			"%w: ListBox derives its content identity, style, and extent",
			ErrValidation,
		)
	}
	requestedOffset := scrollOptions.State.Offset
	if requestedOffset.X < 0 || requestedOffset.Y < 0 ||
		requestedOffset.X > maxCoordinateMagnitude ||
		requestedOffset.Y > maxCoordinateMagnitude {
		return listBoxBehavior{}, fmt.Errorf(
			"%w: invalid ListBox offset",
			ErrValidation,
		)
	}
	changeCommand := scrollOptions.ChangeCommand
	if err := validateOptionalCommand(changeCommand); err != nil {
		return listBoxBehavior{}, err
	}
	if err := validateOptionalCommand(options.ActivateCommand); err != nil {
		return listBoxBehavior{}, err
	}
	if err := validateOptionalCommand(options.CurrentCommand); err != nil {
		return listBoxBehavior{}, err
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
		return listBoxBehavior{}, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		ControlListBox,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return listBoxBehavior{}, err
	}
	scroll.border = border
	scroll.state.Offset = requestedOffset
	mode, err := normalizeCollectionSelectionMode(options.SelectionMode)
	if err != nil {
		return listBoxBehavior{}, err
	}
	marks, err := normalizeCollectionSelectionMarks(options.SelectionMarks)
	if err != nil {
		return listBoxBehavior{}, err
	}
	status, message, err := normalizeCollectionStatus(
		options.Status,
		options.StatusMessage,
	)
	if err != nil {
		return listBoxBehavior{}, err
	}
	wrap, err := normalizeTextWrap(options.Wrap)
	if err != nil {
		return listBoxBehavior{}, err
	}
	items, err := normalizeListItems(options.Items)
	if err != nil {
		return listBoxBehavior{}, err
	}
	behavior := listBoxBehavior{
		scroll:           scroll,
		items:            items,
		selectionMode:    mode,
		selectionMarks:   marks,
		requireSelection: options.RequireSelection,
		status:           status,
		statusMessage:    message,
		wrap:             wrap,
		changeCommand:    changeCommand,
		currentCommand:   options.CurrentCommand,
		activateCommand:  options.ActivateCommand,
	}
	if err := setExactListIdentity(
		&behavior,
		options.Current,
		options.Selected,
	); err != nil {
		return listBoxBehavior{}, err
	}
	return behavior, nil
}

func normalizeCollectionSelectionMarks(
	marks CollectionSelectionMarks,
) (bool, error) {
	switch marks {
	case CollectionSelectionMarksDefault, CollectionSelectionMarksShow:
		return true, nil
	case CollectionSelectionMarksHide:
		return false, nil
	default:
		return false, fmt.Errorf("%w: invalid collection selection marks", ErrValidation)
	}
}

func normalizeCollectionSelectionMode(
	mode CollectionSelectionMode,
) (CollectionSelectionMode, error) {
	if mode == "" {
		mode = CollectionSelectionSingle
	}
	switch mode {
	case CollectionSelectionSingle, CollectionSelectionMultiple:
		return mode, nil
	default:
		return "", fmt.Errorf(
			"%w: invalid collection selection mode %q",
			ErrValidation,
			mode,
		)
	}
}

func normalizeCollectionStatus(
	status CollectionStatus,
	message string,
) (CollectionStatus, normalizedDisplayText, error) {
	if status == "" {
		status = CollectionReady
	}
	switch status {
	case CollectionReady:
		if message != "" {
			return "", normalizedDisplayText{}, fmt.Errorf(
				"%w: ready collection cannot have a status message",
				ErrValidation,
			)
		}
		return status, normalizedDisplayText{}, nil
	case CollectionLoading:
		if message == "" {
			message = "Loading..."
		}
	case CollectionError:
		if message == "" {
			return "", normalizedDisplayText{}, fmt.Errorf(
				"%w: collection error requires a message",
				ErrValidation,
			)
		}
	default:
		return "", normalizedDisplayText{}, fmt.Errorf(
			"%w: invalid collection status %q",
			ErrValidation,
			status,
		)
	}
	normalized, err := normalizeDisplayText(message, false)
	if err != nil {
		return "", normalizedDisplayText{}, err
	}
	if normalized.cells == 0 {
		return "", normalizedDisplayText{}, fmt.Errorf(
			"%w: collection status message is empty",
			ErrValidation,
		)
	}
	return status, normalized, nil
}

func normalizeListItems(items []ListItem) ([]normalizedListItem, error) {
	if len(items) > MaxCollectionItems {
		return nil, fmt.Errorf(
			"%w: ListBox exceeds %d items",
			ErrControlCapacity,
			MaxCollectionItems,
		)
	}
	result := make([]normalizedListItem, len(items))
	seen := make(map[string]bool, len(items))
	for index, item := range items {
		if !validBoundedIdentifier(item.Key) || seen[item.Key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate ListBox item key %q",
				ErrValidation,
				item.Key,
			)
		}
		seen[item.Key] = true
		label, err := normalizeDisplayText(item.Label, false)
		if err != nil || label.cells == 0 {
			return nil, fmt.Errorf(
				"%w: invalid ListBox label for %q",
				ErrValidation,
				item.Key,
			)
		}
		description, err := normalizeDisplayText(item.Description, false)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: invalid ListBox description for %q",
				ErrValidation,
				item.Key,
			)
		}
		indicator := normalizedDisplayText{}
		if (item.Indicator == "") != (item.IndicatorStyle == "") {
			return nil, fmt.Errorf(
				"%w: collection item %q indicator and style must be supplied together",
				ErrValidation,
				item.Key,
			)
		}
		if item.Indicator != "" {
			indicator, err = normalizeDisplayText(item.Indicator, false)
			if err != nil || indicator.cells != 1 {
				return nil, fmt.Errorf(
					"%w: invalid one-cell collection indicator for %q",
					ErrValidation,
					item.Key,
				)
			}
			item.IndicatorStyle, err = normalizeStyleID(item.IndicatorStyle, "")
			if err != nil {
				return nil, err
			}
		}
		reason, err := normalizeDisabledReason(
			item.Disabled,
			item.DisabledReason,
		)
		if err != nil {
			return nil, err
		}
		item.Label = label.text
		item.Description = description.text
		item.Indicator = indicator.text
		item.DisabledReason = reason
		result[index] = normalizedListItem{
			item: item, label: label, description: description,
			indicator: indicator,
		}
	}
	return result, nil
}

func setExactListIdentity(
	behavior *listBoxBehavior,
	current string,
	selected []string,
) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	if current == "" {
		current = firstEnabledListKey(behavior.items)
	} else if index := listItemIndex(behavior.items, current); index < 0 ||
		behavior.items[index].item.Disabled {
		return fmt.Errorf("%w: invalid ListBox current key", ErrValidation)
	}
	ordered, err := normalizeListSelection(
		behavior.items,
		behavior.selectionMode,
		selected,
	)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(ordered) == 0 && current != "" {
		ordered = []string{current}
	}
	behavior.current = current
	behavior.selected = ordered
	return nil
}

func normalizeListSelection(
	items []normalizedListItem,
	mode CollectionSelectionMode,
	selected []string,
) ([]string, error) {
	if mode == CollectionSelectionSingle && len(selected) > 1 {
		return nil, fmt.Errorf(
			"%w: single-selection ListBox has multiple selected keys",
			ErrValidation,
		)
	}
	requested := make(map[string]bool, len(selected))
	for _, key := range selected {
		if !validBoundedIdentifier(key) || requested[key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate ListBox selection",
				ErrValidation,
			)
		}
		index := listItemIndex(items, key)
		if index < 0 || items[index].item.Disabled {
			return nil, fmt.Errorf(
				"%w: ListBox selection identifies an unavailable item",
				ErrValidation,
			)
		}
		requested[key] = true
	}
	result := make([]string, 0, len(selected))
	for _, item := range items {
		if requested[item.item.Key] {
			result = append(result, item.item.Key)
		}
	}
	return result, nil
}

func firstEnabledListKey(items []normalizedListItem) string {
	for _, item := range items {
		if !item.item.Disabled {
			return item.item.Key
		}
	}
	return ""
}

func listItemIndex(items []normalizedListItem, key string) int {
	for index, item := range items {
		if item.item.Key == key {
			return index
		}
	}
	return -1
}

func listSelectionContains(selected []string, key string) bool {
	for _, candidate := range selected {
		if candidate == key {
			return true
		}
	}
	return false
}

func copyListItems(items []normalizedListItem) []ListItem {
	result := make([]ListItem, len(items))
	for index := range items {
		result[index] = items[index].item
	}
	return result
}

func cloneListBoxBehavior(behavior listBoxBehavior) listBoxBehavior {
	cloned := behavior
	cloned.items = cloneNormalizedListItems(behavior.items)
	cloned.rows = make([]listBoxVisualRow, len(behavior.rows))
	for index, row := range behavior.rows {
		cloned.rows[index] = row
		cloned.rows[index].cells = append([]string(nil), row.cells...)
	}
	cloned.selected = append([]string(nil), behavior.selected...)
	cloned.statusMessage.lines = cloneTextRows(behavior.statusMessage.lines)
	return cloned
}

func cloneNormalizedListItems(items []normalizedListItem) []normalizedListItem {
	cloned := make([]normalizedListItem, len(items))
	for index, item := range items {
		cloned[index] = item
		cloned[index].label.lines = cloneTextRows(item.label.lines)
		cloned[index].description.lines = cloneTextRows(item.description.lines)
		cloned[index].indicator.lines = cloneTextRows(item.indicator.lines)
	}
	return cloned
}

func cloneTextRows(rows [][]string) [][]string {
	result := make([][]string, len(rows))
	for index := range rows {
		result[index] = append([]string(nil), rows[index]...)
	}
	return result
}

func listBoxBehaviorEqual(left, right listBoxBehavior) bool {
	if !scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.current != right.current ||
		left.selectionMode != right.selectionMode ||
		left.selectionMarks != right.selectionMarks ||
		left.requireSelection != right.requireSelection ||
		left.status != right.status ||
		left.statusMessage.text != right.statusMessage.text ||
		left.wrap != right.wrap ||
		left.changeCommand != right.changeCommand ||
		left.currentCommand != right.currentCommand ||
		left.activateCommand != right.activateCommand ||
		len(left.items) != len(right.items) ||
		len(left.selected) != len(right.selected) ||
		len(left.rows) != len(right.rows) {
		return false
	}
	for index := range left.items {
		if left.items[index].item != right.items[index].item {
			return false
		}
	}
	for index := range left.selected {
		if left.selected[index] != right.selected[index] {
			return false
		}
	}
	for index := range left.rows {
		if left.rows[index].itemIndex != right.rows[index].itemIndex ||
			left.rows[index].within != right.rows[index].within ||
			left.rows[index].indicatorCell != right.rows[index].indicatorCell ||
			!stringSlicesEqual(left.rows[index].cells, right.rows[index].cells) {
			return false
		}
	}
	return true
}

func (b listBoxBehavior) controlBorder() borderBehavior {
	return b.scroll.border
}

func (b listBoxBehavior) clientInset() int { return b.scroll.clientInset() }

func (b listBoxBehavior) controlClientRect(bounds Rect) Rect {
	return b.scroll.controlClientRect(bounds)
}

func (b listBoxBehavior) intrinsicMinimum() Size {
	return b.scroll.intrinsicMinimum()
}

func (b listBoxBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	styles = append(styles,
		"collection.current",
		"collection.selected",
		"collection.current_selected",
		"collection.disabled",
		"collection.empty",
		"collection.loading",
		"collection.error",
	)
	return append(styles, listItemIndicatorStyles(b.items)...)
}

func listItemIndicatorStyles(items []normalizedListItem) []StyleID {
	styles := make([]StyleID, 0)
	seen := make(map[StyleID]bool)
	for _, item := range items {
		style := item.item.IndicatorStyle
		if style == "" || seen[style] {
			continue
		}
		seen[style] = true
		styles = append(styles, style)
	}
	return styles
}

func (b listBoxBehavior) details() ControlDetails {
	details := b.scroll.border.details()
	details.Container = nil
	details.ListBox = &ListBoxDetails{}
	return details
}

func (b listBoxBehavior) paintDecoration(
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
	for y := visible.Y; y < visible.Y+visible.Height; y++ {
		sourceY := b.scroll.state.Offset.Y + y - viewport.Y
		cells, rowStyle, exists := b.displayRow(sourceY, app.focus == state)
		if !exists {
			continue
		}
		for x := visible.X; x < visible.X+visible.Width; x++ {
			sourceX := b.scroll.state.Offset.X + x - viewport.X
			cell := collectionPaintCell{grapheme: " ", style: rowStyle}
			if sourceX >= 0 && sourceX < len(cells) {
				cell = cells[sourceX]
			}
			app.setCellLocked(
				frame,
				x,
				y,
				cell.grapheme,
				cell.style,
				app.styles[cell.style],
				state.id,
			)
		}
	}
}

func (b listBoxBehavior) displayRow(
	index int,
	focused bool,
) ([]collectionPaintCell, StyleID, bool) {
	if index != 0 && (b.status != CollectionReady || len(b.items) == 0) {
		return nil, "", false
	}
	switch b.status {
	case CollectionLoading:
		return uniformCollectionPaintCells(
			listStatusCells("[loading] ", b.statusMessage),
			"collection.loading",
		), "collection.loading", true
	case CollectionError:
		return uniformCollectionPaintCells(
			listStatusCells("[error] ", b.statusMessage),
			"collection.error",
		), "collection.error", true
	}
	if len(b.items) == 0 {
		return uniformCollectionPaintCells(
			[]string{"[", "e", "m", "p", "t", "y", "]"},
			"collection.empty",
		), "collection.empty", true
	}
	if index < 0 || index >= len(b.rows) {
		return nil, "", false
	}
	row := b.rows[index]
	if row.itemIndex < 0 || row.itemIndex >= len(b.items) {
		return nil, "", false
	}
	item := b.items[row.itemIndex]
	current := item.item.Key == b.current
	selected := listSelectionContains(b.selected, item.item.Key)
	style := StyleID("list_box")
	if item.item.Disabled {
		style = "collection.disabled"
	} else if focused && current && selected {
		style = "collection.current_selected"
	} else if focused && current {
		style = "collection.current"
	} else if selected {
		style = "collection.selected"
	}
	marker := " "
	if focused && current {
		marker = "►"
	}
	selectedMarker := " "
	if selected {
		selectedMarker = "X"
	}
	rowCells := append([]string(nil), row.cells...)
	if len(rowCells) > 0 && row.within == 0 {
		rowCells[0] = marker
		if b.selectionMarks && len(rowCells) > 3 {
			rowCells[3] = selectedMarker
		}
	}
	cells := uniformCollectionPaintCells(rowCells, style)
	if row.indicatorCell >= 0 && row.indicatorCell < len(cells) {
		cells[row.indicatorCell].style = item.item.IndicatorStyle
	}
	return cells, style, true
}

func uniformCollectionPaintCells(
	cells []string,
	style StyleID,
) []collectionPaintCell {
	result := make([]collectionPaintCell, len(cells))
	for index, grapheme := range cells {
		result[index] = collectionPaintCell{grapheme: grapheme, style: style}
	}
	return result
}

func listItemCells(
	item normalizedListItem,
	selectionMarks bool,
) ([]string, int, int) {
	cells := []string{" ", " "}
	if selectionMarks {
		cells = append(cells, "[", " ", "]", " ")
	}
	indicatorCell := -1
	if item.indicator.cells == 1 {
		indicatorCell = len(cells)
		cells = append(cells, item.indicator.lines[0][0], " ")
	}
	cells = append(cells, item.label.lines[0]...)
	indent := len(cells)
	if item.description.cells > 0 {
		cells = append(cells, " ", " ")
		indent = len(cells)
		cells = append(cells, item.description.lines[0]...)
	}
	return cells, indent, indicatorCell
}

type wrappedListItemCells struct {
	cells         []string
	indicatorCell int
}

func wrapListItemCells(
	cells []string,
	width int,
	wrap TextWrap,
	indent int,
	indicatorCell int,
) []wrappedListItemCells {
	if wrap == TextWrapNone || width <= 0 || len(cells) <= width {
		return []wrappedListItemCells{{
			cells: append([]string(nil), cells...), indicatorCell: indicatorCell,
		}}
	}
	continuation := indent
	if continuation >= width {
		continuation = min(2, max(0, width-1))
	}
	rows := make([]wrappedListItemCells, 0, 1+len(cells)/max(1, width-continuation))
	remaining := append([]string(nil), cells...)
	sourceOffset := 0
	first := true
	for len(remaining) > 0 {
		prefix := 0
		if !first {
			prefix = continuation
		}
		available := max(1, width-prefix)
		breakAt := min(available, len(remaining))
		if wrap == TextWrapWords && breakAt < len(remaining) {
			for index := breakAt - 1; index > 0; index-- {
				if remaining[index] == " " {
					breakAt = index
					break
				}
			}
		}
		line := append([]string(nil), remaining[:breakAt]...)
		lineIndicator := -1
		if indicatorCell >= sourceOffset && indicatorCell < sourceOffset+breakAt {
			lineIndicator = prefix + indicatorCell - sourceOffset
		}
		for len(line) > 0 && line[len(line)-1] == " " {
			line = line[:len(line)-1]
		}
		if prefix > 0 {
			line = append(make([]string, prefix), line...)
			for index := range prefix {
				line[index] = " "
			}
		}
		rows = append(rows, wrappedListItemCells{
			cells: line, indicatorCell: lineIndicator,
		})
		remaining = remaining[breakAt:]
		sourceOffset += breakAt
		if wrap == TextWrapWords {
			for len(remaining) > 0 && remaining[0] == " " {
				remaining = remaining[1:]
				sourceOffset++
			}
		}
		first = false
	}
	return rows
}

func listBoxRows(behavior listBoxBehavior, width int) []listBoxVisualRow {
	rows := make([]listBoxVisualRow, 0, len(behavior.items))
	for itemIndex, item := range behavior.items {
		cells, indent, indicatorCell := listItemCells(item, behavior.selectionMarks)
		wrapped := wrapListItemCells(
			cells, width, behavior.wrap, indent, indicatorCell,
		)
		for within, line := range wrapped {
			rows = append(rows, listBoxVisualRow{
				itemIndex: itemIndex, within: within,
				indicatorCell: line.indicatorCell, cells: line.cells,
			})
		}
	}
	return rows
}

func listStatusCells(
	prefix string,
	message normalizedDisplayText,
) []string {
	cells := make([]string, 0, len(prefix)+message.cells)
	for _, value := range prefix {
		cells = append(cells, string(value))
	}
	if len(message.lines) > 0 {
		cells = append(cells, message.lines[0]...)
	}
	return cells
}

func reflowListBox(
	behavior listBoxBehavior,
	size Size,
) listBoxBehavior {
	behavior.rows = nil
	if behavior.status == CollectionReady && len(behavior.items) > 0 {
		probe := behavior.scroll
		probe.state.ContentSize = Size{Height: len(behavior.items)}
		geometry := calculateScrollViewGeometry(size, probe)
		for range 3 {
			behavior.rows = listBoxRows(behavior, geometry.viewport.Width)
			maximum := 0
			for _, row := range behavior.rows {
				maximum = max(maximum, len(row.cells))
			}
			behavior.scroll.state.ContentSize = Size{Width: maximum, Height: len(behavior.rows)}
			next := calculateScrollViewGeometry(size, behavior.scroll)
			if next.viewport.Width == geometry.viewport.Width &&
				next.viewport.Height == geometry.viewport.Height {
				break
			}
			geometry = next
		}
	} else {
		behavior.scroll.state.ContentSize = Size{Width: 0, Height: 1}
		cells, _, exists := behavior.displayRow(0, true)
		if exists {
			behavior.scroll.state.ContentSize.Width = len(cells)
		}
	}
	geometry := calculateScrollViewGeometry(size, behavior.scroll)
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	if behavior.status == CollectionReady {
		current := listItemIndex(behavior.items, behavior.current)
		if current >= 0 && geometry.viewport.Height > 0 {
			first, last := listBoxItemVisualBounds(behavior.rows, current)
			if last-first+1 > geometry.viewport.Height {
				behavior.scroll.state.Offset.Y = first
			} else if first < behavior.scroll.state.Offset.Y {
				behavior.scroll.state.Offset.Y = first
			} else if last >= behavior.scroll.state.Offset.Y+geometry.viewport.Height {
				behavior.scroll.state.Offset.Y = last - geometry.viewport.Height + 1
			}
		}
	}
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	return behavior
}

func listBoxItemVisualBounds(rows []listBoxVisualRow, item int) (int, int) {
	first := -1
	last := -1
	for index, row := range rows {
		if row.itemIndex != item {
			continue
		}
		if first < 0 {
			first = index
		}
		last = index
	}
	return max(first, 0), max(last, 0)
}

func reconcileListBoxLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(listBoxBehavior)
	if !ok {
		return false
	}
	next := reflowListBox(behavior, state.bounds.Size())
	if listBoxBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func listBoxCanFocus(state *controlState, behavior listBoxBehavior) bool {
	return state != nil && !behavior.scroll.disabled &&
		behavior.status == CollectionReady && behavior.current != ""
}

func enabledListCount(items []normalizedListItem) int {
	count := 0
	for _, item := range items {
		if !item.item.Disabled {
			count++
		}
	}
	return count
}

func listSelectionDigest(selected []string) string {
	hash := sha256.New()
	var length [2]byte
	for _, key := range selected {
		binary.BigEndian.PutUint16(length[:], uint16(len(key)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(key))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func listBoxDetails(bounds Rect, behavior listBoxBehavior) ListBoxDetails {
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	firstSelected := ""
	lastSelected := ""
	if len(behavior.selected) > 0 {
		firstSelected = behavior.selected[0]
		lastSelected = behavior.selected[len(behavior.selected)-1]
	}
	return ListBoxDetails{
		Status:           behavior.status,
		StatusMessage:    behavior.statusMessage.text,
		ItemCount:        len(behavior.items),
		EnabledCount:     enabledListCount(behavior.items),
		VisualRowCount:   len(behavior.rows),
		Wrap:             behavior.wrap,
		RetainedBytes:    listBoxStorageBytes(behavior),
		Current:          behavior.current,
		CurrentIndex:     listItemIndex(behavior.items, behavior.current),
		SelectionMode:    behavior.selectionMode,
		SelectionMarks:   behavior.selectionMarks,
		RequireSelection: behavior.requireSelection,
		SelectedCount:    len(behavior.selected),
		FirstSelected:    firstSelected,
		LastSelected:     lastSelected,
		SelectionDigest:  listSelectionDigest(behavior.selected),
		Enabled:          !behavior.scroll.disabled,
		DisabledReason:   behavior.scroll.disabledReason,
		ChangeCommand:    behavior.changeCommand,
		CurrentCommand:   behavior.currentCommand,
		ActivateCommand:  behavior.activateCommand,
		Viewport:         viewport,
	}
}

// Items returns a caller-owned copy of the complete retained model.
func (l *ListBox) Items() []ListItem {
	if l == nil || l.state == nil || l.state.app == nil {
		return nil
	}
	l.state.app.mu.RLock()
	defer l.state.app.mu.RUnlock()
	behavior, ok := l.state.behavior.(listBoxBehavior)
	if !ok || l.state.destroyed || l.state.aborted {
		return nil
	}
	return copyListItems(behavior.items)
}

// State returns a caller-owned copy of the complete semantic state.
func (l *ListBox) State() ListBoxState {
	if l == nil || l.state == nil || l.state.app == nil {
		return ListBoxState{CurrentIndex: -1}
	}
	l.state.app.mu.RLock()
	defer l.state.app.mu.RUnlock()
	behavior, ok := l.state.behavior.(listBoxBehavior)
	if !ok || l.state.destroyed || l.state.aborted {
		return ListBoxState{CurrentIndex: -1}
	}
	return ListBoxState{
		Status:         behavior.status,
		StatusMessage:  behavior.statusMessage.text,
		Current:        behavior.current,
		CurrentIndex:   listItemIndex(behavior.items, behavior.current),
		Selected:       append([]string(nil), behavior.selected...),
		SelectionMarks: behavior.selectionMarks,
		Offset:         behavior.scroll.state.Offset,
		ItemCount:      len(behavior.items),
		EnabledCount:   enabledListCount(behavior.items),
		VisualRowCount: len(behavior.rows),
		Wrap:           behavior.wrap,
	}
}

// SetItems replaces the copied model while preserving surviving identities.
func (l *ListBox) SetItems(items []ListItem) error {
	return commitListMutation(l, func(transaction *Transaction) error {
		return transaction.SetListItems(l, items)
	})
}

// Replace atomically replaces the model and exact requested identities.
func (l *ListBox) Replace(
	items []ListItem,
	current string,
	selected []string,
) error {
	return commitListMutation(l, func(transaction *Transaction) error {
		return transaction.ReplaceList(l, items, current, selected)
	})
}

// SetCurrent changes current by stable enabled item key.
func (l *ListBox) SetCurrent(current string) error {
	return commitListMutation(l, func(transaction *Transaction) error {
		return transaction.SetListCurrent(l, current)
	})
}

// SetSelection changes selection by stable enabled item keys.
func (l *ListBox) SetSelection(selected []string) error {
	return commitListMutation(l, func(transaction *Transaction) error {
		return transaction.SetListSelection(l, selected)
	})
}

// SetStatus changes ready/loading/error presentation without discarding data.
func (l *ListBox) SetStatus(status CollectionStatus, message string) error {
	return commitListMutation(l, func(transaction *Transaction) error {
		return transaction.SetListStatus(l, status, message)
	})
}

func commitListMutation(
	control *ListBox,
	record func(*Transaction) error,
) error {
	if control == nil || control.state == nil || control.state.app == nil {
		return ErrInvalidControl
	}
	transaction := control.state.app.NewTransaction()
	if err := record(transaction); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus atomically gives the ListBox keyboard focus when it has a ready,
// enabled current item.
func (l *ListBox) Focus() error { return focusSelectionControl(l) }

// Activate selects the current item when necessary and invokes the optional
// activation command through the ordinary command router.
func (l *ListBox) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if l == nil || l.state == nil || l.state.app == nil {
		return Completion{}, ErrInvalidControl
	}
	return l.state.app.activateListBox(ctx, source, requestID, l.state)
}

func (a *App) activateListBox(
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
	behavior, ok := state.behavior.(listBoxBehavior)
	if !ok || !listBoxCanFocus(state, behavior) ||
		!a.inActiveModalScopeLocked(state) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	command, target, _, changed := a.listBoxKeyLocked(state, KeyEnter)
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
		result = a.callRouter(ctx, router, Command{
			ID: command, Target: target, Source: source,
		})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	return a.associateLocked(requestID, result, command), nil
}

func (a *App) listBoxKeyLocked(
	state *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if state == nil {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(listBoxBehavior)
	if !ok || !listBoxCanFocus(state, behavior) {
		return "", "", false, false
	}
	before := cloneListBoxBehavior(behavior)
	handled := true
	selectionChanged := false
	activate := false
	current := listItemIndex(behavior.items, behavior.current)
	geometry := calculateScrollViewGeometry(state.bounds.Size(), behavior.scroll)
	switch key {
	case KeyUp:
		behavior.current = previousEnabledListKey(behavior.items, current)
	case KeyDown:
		behavior.current = nextEnabledListKey(behavior.items, current)
	case KeyPageUp:
		behavior.current = pageEnabledListKeyByVisualRows(
			behavior,
			current,
			-effectiveScrollPageStep(
				behavior.scroll.pageStep.Height,
				geometry.viewport.Height,
			),
		)
	case KeyPageDown:
		behavior.current = pageEnabledListKeyByVisualRows(
			behavior,
			current,
			effectiveScrollPageStep(
				behavior.scroll.pageStep.Height,
				geometry.viewport.Height,
			),
		)
	case KeyHome:
		behavior.current = firstEnabledListKey(behavior.items)
	case KeyEnd:
		behavior.current = lastEnabledListKey(behavior.items)
	case KeyLeft:
		behavior.scroll.state.Offset.X -= behavior.scroll.arrowStep.Width
	case KeyRight:
		behavior.scroll.state.Offset.X += behavior.scroll.arrowStep.Width
	case KeySpace:
		selectionChanged = selectCurrentListItem(&behavior, true)
	case KeyEnter:
		selectionChanged = selectCurrentListItem(&behavior, false)
		activate = true
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	behavior = reflowListBox(behavior, state.bounds.Size())
	changed := !listBoxBehaviorEqual(before, behavior)
	if changed {
		state.behavior = behavior
	}
	currentChanged := before.current != behavior.current
	command := CommandID("")
	if activate && behavior.activateCommand != "" {
		command = behavior.activateCommand
	} else if selectionChanged {
		command = behavior.changeCommand
	} else if currentChanged {
		command = behavior.currentCommand
	}
	return command, state.id, true, changed
}

func previousEnabledListKey(items []normalizedListItem, current int) string {
	for index := current - 1; index >= 0; index-- {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	if current >= 0 && current < len(items) {
		return items[current].item.Key
	}
	return firstEnabledListKey(items)
}

func nextEnabledListKey(items []normalizedListItem, current int) string {
	for index := current + 1; index < len(items); index++ {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	if current >= 0 && current < len(items) {
		return items[current].item.Key
	}
	return firstEnabledListKey(items)
}

func lastEnabledListKey(items []normalizedListItem) string {
	for index := len(items) - 1; index >= 0; index-- {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	return ""
}

func pageEnabledListKey(
	items []normalizedListItem,
	current int,
	delta int,
) string {
	if len(items) == 0 {
		return ""
	}
	target := min(len(items)-1, max(0, current+delta))
	if delta < 0 {
		for index := target; index >= 0; index-- {
			if !items[index].item.Disabled {
				return items[index].item.Key
			}
		}
		return nextEnabledListKey(items, target-1)
	}
	for index := target; index < len(items); index++ {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	return previousEnabledListKey(items, target+1)
}

func pageEnabledListKeyByVisualRows(
	behavior listBoxBehavior,
	current int,
	delta int,
) string {
	if current < 0 || current >= len(behavior.items) || len(behavior.rows) == 0 {
		return firstEnabledListKey(behavior.items)
	}
	first, _ := listBoxItemVisualBounds(behavior.rows, current)
	targetRow := min(len(behavior.rows)-1, max(0, first+delta))
	targetItem := behavior.rows[targetRow].itemIndex
	logicalDelta := targetItem - current
	if logicalDelta == 0 {
		if delta < 0 {
			logicalDelta = -1
		} else if delta > 0 {
			logicalDelta = 1
		}
	}
	return pageEnabledListKey(behavior.items, current, logicalDelta)
}

func selectCurrentListItem(behavior *listBoxBehavior, toggle bool) bool {
	if behavior == nil || behavior.current == "" {
		return false
	}
	selected := listSelectionContains(behavior.selected, behavior.current)
	if behavior.selectionMode == CollectionSelectionSingle {
		if selected && len(behavior.selected) == 1 {
			return false
		}
		behavior.selected = []string{behavior.current}
		return true
	}
	if selected {
		if !toggle || (behavior.requireSelection && len(behavior.selected) == 1) {
			return false
		}
		result := make([]string, 0, len(behavior.selected)-1)
		for _, key := range behavior.selected {
			if key != behavior.current {
				result = append(result, key)
			}
		}
		behavior.selected = result
		return true
	}
	requested := append(append([]string(nil), behavior.selected...), behavior.current)
	ordered, _ := normalizeListSelection(
		behavior.items,
		behavior.selectionMode,
		requested,
	)
	behavior.selected = ordered
	return true
}

func listBoxStorageBytes(behavior listBoxBehavior) int {
	total := len(behavior.statusMessage.text) + len(behavior.current)
	for _, item := range behavior.items {
		total += len(item.item.Key) + len(item.item.Label) +
			len(item.item.Description) + len(item.item.Indicator) +
			len(item.item.IndicatorStyle) + len(item.item.DisabledReason)
	}
	for _, key := range behavior.selected {
		total += len(key)
	}
	return total
}

func (t *Transaction) selectedListBox(
	control *ListBox,
) (*controlState, listBoxBehavior, error) {
	target, err := t.control(control)
	if err != nil {
		return nil, listBoxBehavior{}, err
	}
	behavior, ok := t.selectedControlBehavior(target).(listBoxBehavior)
	if !ok {
		return nil, listBoxBehavior{}, ErrInvalidControl
	}
	return target, cloneListBoxBehavior(behavior), nil
}

func (t *Transaction) recordListBox(
	state *controlState,
	behavior listBoxBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationListBox, state: state, behavior: behavior,
	})
	return nil
}

// SetListItems records a copied model replacement preserving stable state.
func (t *Transaction) SetListItems(
	control *ListBox,
	items []ListItem,
) error {
	target, behavior, err := t.selectedListBox(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeListItems(items)
	if err != nil {
		return err
	}
	oldIndex := listItemIndex(behavior.items, behavior.current)
	behavior.items = normalized
	repairListIdentity(&behavior, oldIndex)
	behavior = reflowListBox(behavior, target.bounds.Size())
	return t.recordListBox(target, behavior)
}

// ReplaceList records one exact copied model/current/selection replacement.
func (t *Transaction) ReplaceList(
	control *ListBox,
	items []ListItem,
	current string,
	selected []string,
) error {
	target, behavior, err := t.selectedListBox(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeListItems(items)
	if err != nil {
		return err
	}
	behavior.items = normalized
	if err := setExactListIdentity(&behavior, current, selected); err != nil {
		return err
	}
	behavior = reflowListBox(behavior, target.bounds.Size())
	return t.recordListBox(target, behavior)
}

// SetListCurrent records one stable enabled current key.
func (t *Transaction) SetListCurrent(
	control *ListBox,
	current string,
) error {
	target, behavior, err := t.selectedListBox(control)
	if err != nil {
		return err
	}
	if current == "" {
		if firstEnabledListKey(behavior.items) != "" {
			return fmt.Errorf("%w: ready ListBox requires current", ErrValidation)
		}
	} else {
		index := listItemIndex(behavior.items, current)
		if index < 0 || behavior.items[index].item.Disabled {
			return fmt.Errorf("%w: invalid ListBox current key", ErrValidation)
		}
	}
	behavior.current = current
	behavior = reflowListBox(behavior, target.bounds.Size())
	return t.recordListBox(target, behavior)
}

// SetListSelection records one exact stable-key selection.
func (t *Transaction) SetListSelection(
	control *ListBox,
	selected []string,
) error {
	target, behavior, err := t.selectedListBox(control)
	if err != nil {
		return err
	}
	ordered, err := normalizeListSelection(
		behavior.items,
		behavior.selectionMode,
		selected,
	)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(ordered) == 0 && behavior.current != "" {
		return fmt.Errorf("%w: ListBox selection is required", ErrValidation)
	}
	behavior.selected = ordered
	behavior = reflowListBox(behavior, target.bounds.Size())
	return t.recordListBox(target, behavior)
}

// SetListStatus records ready/loading/error presentation state.
func (t *Transaction) SetListStatus(
	control *ListBox,
	status CollectionStatus,
	message string,
) error {
	target, behavior, err := t.selectedListBox(control)
	if err != nil {
		return err
	}
	status, normalized, err := normalizeCollectionStatus(status, message)
	if err != nil {
		return err
	}
	behavior.status = status
	behavior.statusMessage = normalized
	behavior = reflowListBox(behavior, target.bounds.Size())
	return t.recordListBox(target, behavior)
}

func repairListIdentity(behavior *listBoxBehavior, oldIndex int) {
	if behavior == nil {
		return
	}
	currentIndex := listItemIndex(behavior.items, behavior.current)
	if currentIndex < 0 || behavior.items[currentIndex].item.Disabled {
		behavior.current = repairedListCurrent(behavior.items, oldIndex)
	}
	selected := make([]string, 0, len(behavior.selected))
	for _, item := range behavior.items {
		if !item.item.Disabled &&
			listSelectionContains(behavior.selected, item.item.Key) {
			selected = append(selected, item.item.Key)
		}
	}
	if behavior.selectionMode == CollectionSelectionSingle && len(selected) > 1 {
		selected = selected[:1]
	}
	if behavior.requireSelection && len(selected) == 0 && behavior.current != "" {
		selected = []string{behavior.current}
	}
	behavior.selected = selected
}

func repairedListCurrent(items []normalizedListItem, oldIndex int) string {
	if len(items) == 0 {
		return ""
	}
	oldIndex = min(len(items)-1, max(0, oldIndex))
	for index := oldIndex; index < len(items); index++ {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	for index := oldIndex - 1; index >= 0; index-- {
		if !items[index].item.Disabled {
			return items[index].item.Key
		}
	}
	return ""
}
