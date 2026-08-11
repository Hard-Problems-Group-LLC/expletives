package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TextAreaOptions configures one bounded multiline editor.
type TextAreaOptions struct {
	PanelOptions
	Text           string
	Validator      *TextValidator
	Password       bool
	ReadOnly       bool
	Wrap           TextWrap
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// TextArea is a copy-safe focusable multiline editor or read-only leaf.
type TextArea struct{ controlHandle }

type textAreaBehavior struct {
	committed       normalizedInputText
	working         normalizedInputText
	validator       *normalizedTextValidator
	password        bool
	readOnly        bool
	wrap            TextWrap
	disabled        bool
	disabledReason  string
	changeCommand   CommandID
	editing         bool
	caret           int
	selectionAnchor int
	rowOffset       int
	columnOffset    int
	preferredColumn int
}

type textAreaVisualRow struct {
	start     int
	end       int
	continues bool
}

// NewTextArea constructs and atomically inserts a TextArea.
func NewTextArea(parent Container, options TextAreaOptions) (*TextArea, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	area, err := tx.NewTextArea(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return area, nil
}

// NewTextArea records construction of a provisional TextArea.
func (t *Transaction) NewTextArea(
	parent Container,
	options TextAreaOptions,
) (*TextArea, error) {
	value, err := normalizeTextAreaInput(options.Text)
	if err != nil {
		return nil, err
	}
	validator, err := normalizeTextValidator(options.Validator)
	if err != nil {
		return nil, err
	}
	if validator != nil &&
		validator.value.Enforcement == TextValidationHard &&
		!textAreaCellsValid(value.cells, validator) {
		return nil, fmt.Errorf(
			"%w: initial TextArea value violates hard validation",
			ErrValidation,
		)
	}
	wrap, err := normalizeTextWrap(options.Wrap)
	if err != nil {
		return nil, err
	}
	reason, err := normalizeDisabledReason(
		options.Disabled,
		options.DisabledReason,
	)
	if err != nil {
		return nil, err
	}
	if err := validateOptionalCommand(options.ChangeCommand); err != nil {
		return nil, err
	}
	caret := len(value.cells)
	if options.ReadOnly {
		caret = 0
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlTextArea,
		textAreaBehavior{
			committed:       value,
			working:         cloneInputText(value),
			validator:       validator,
			password:        options.Password,
			readOnly:        options.ReadOnly,
			wrap:            wrap,
			disabled:        options.Disabled,
			disabledReason:  reason,
			changeCommand:   options.ChangeCommand,
			caret:           caret,
			selectionAnchor: -1,
			preferredColumn: -1,
		},
	)
	if err != nil {
		return nil, err
	}
	area := &TextArea{controlHandle: controlHandle{state: state}}
	state.control = area
	return area, nil
}

func normalizeTextAreaInput(text string) (normalizedInputText, error) {
	if len(text) > MaxTextInputBytes {
		return normalizedInputText{}, fmt.Errorf(
			"%w: input text exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTextInputBytes,
		)
	}
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "\uFFFD")
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	sourceLines := strings.Split(text, "\n")
	cells := make([]string, 0, min(len(text), MaxTextInputCells))
	lines := make([]string, len(sourceLines))
	for index, line := range sourceLines {
		normalized, err := normalizeInputText(line)
		if err != nil {
			return normalizedInputText{}, err
		}
		lines[index] = normalized.text
		cells = append(cells, normalized.cells...)
		if index+1 < len(sourceLines) {
			cells = append(cells, "\n")
		}
		if len(cells) > MaxTextInputCells {
			return normalizedInputText{}, fmt.Errorf(
				"%w: input text exceeds %d canonical elements",
				ErrTextLimit,
				MaxTextInputCells,
			)
		}
	}
	normalized := strings.Join(lines, "\n")
	if len(normalized) > MaxTextInputBytes {
		return normalizedInputText{}, fmt.Errorf(
			"%w: normalized input exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTextInputBytes,
		)
	}
	return normalizedInputText{text: normalized, cells: cells}, nil
}

func textAreaCellsValid(
	cells []string,
	validator *normalizedTextValidator,
) bool {
	for _, cell := range cells {
		if cell != "\n" && !textCellAllowed(cell, validator) {
			return false
		}
	}
	return true
}

func textAreaValidationMask(
	cells []string,
	validator *normalizedTextValidator,
) ([]bool, bool) {
	invalid := make([]bool, len(cells))
	valid := true
	for index, cell := range cells {
		if cell == "\n" {
			continue
		}
		invalid[index] = !textCellAllowed(cell, validator)
		if invalid[index] {
			valid = false
		}
	}
	return invalid, valid
}

func (b textAreaBehavior) current() normalizedInputText {
	if b.editing {
		return b.working
	}
	return b.committed
}

func (textAreaBehavior) clientInset() int { return 0 }

func (textAreaBehavior) intrinsicMinimum() Size {
	return Size{Width: 8, Height: 3}
}

func (b textAreaBehavior) additionalStyles() []StyleID {
	styles := []StyleID{
		"text_input.valid",
		"text_input.invalid",
		"text_input.invalid_character",
		"text_input.selection",
		"text_input.disabled",
		"text_input.focused",
		"text_input.focused_valid",
		"text_input.focused_invalid",
		"text_input.focused_invalid_character",
		"text_input.focused_selection",
	}
	if b.readOnly {
		styles = append(
			styles,
			"text_input.read_only",
			"text_input.focused_read_only",
		)
	}
	return styles
}

func (textAreaBehavior) details() ControlDetails {
	return ControlDetails{
		Version:  ControlDetailsVersion,
		TextArea: &TextAreaDetails{},
	}
}

func (b textAreaBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	current := b.current()
	invalid, valid := textAreaValidationMask(current.cells, b.validator)
	backgroundStyle := state.style
	switch {
	case b.disabled:
	case b.readOnly && app.focus == state:
		backgroundStyle = "text_input.focused_read_only"
	case b.readOnly:
		backgroundStyle = "text_input.read_only"
	case app.focus == state:
		backgroundStyle = "text_input.focused"
	}
	app.fillStyleLocked(
		frame,
		absolute.Intersect(clip),
		backgroundStyle,
		state.id,
	)
	rows, caretRow, caretColumn, rowOffset, columnOffset :=
		textAreaViewport(b, absolute.Width, absolute.Height)
	for screenRow := 0; screenRow < absolute.Height; screenRow++ {
		rowIndex := rowOffset + screenRow
		if rowIndex >= len(rows) {
			break
		}
		row := rows[rowIndex]
		first := row.start
		if b.wrap == TextWrapNone {
			first += columnOffset
		}
		for column := 0; column < absolute.Width; column++ {
			index := first + column
			if index >= row.end {
				break
			}
			cell := current.cells[index]
			if b.password {
				cell = "*"
			}
			style := state.style
			switch {
			case b.disabled:
				style = "text_input.disabled"
			case textAreaSelectionContains(b, index):
				style = "text_input.selection"
			case b.readOnly:
				style = "text_input.read_only"
			case b.validator == nil:
			case valid:
				style = "text_input.valid"
			case invalid[index]:
				style = "text_input.invalid_character"
			default:
				style = "text_input.invalid"
			}
			if app.focus == state && !b.disabled {
				if b.readOnly && style == "text_input.read_only" {
					style = "text_input.focused_read_only"
				} else {
					style = focusedTextInputStyle(style)
				}
			}
			app.setClippedCellLocked(
				frame,
				clip,
				absolute.X+column,
				absolute.Y+screenRow,
				cell,
				style,
				app.styles[style],
				state.id,
			)
		}
	}
	cursorX := absolute.X + caretColumn - columnOffset
	cursorY := absolute.Y + caretRow - rowOffset
	if b.wrap != TextWrapNone {
		cursorX = absolute.X + caretColumn
	}
	cursorX = min(cursorX, absolute.X+max(0, absolute.Width-1))
	if app.focus == state && b.editing && !b.disabled && !b.readOnly &&
		absolute.Width > 0 && absolute.Height > 0 &&
		cursorX >= clip.X && cursorX < clip.X+clip.Width &&
		cursorY >= clip.Y && cursorY < clip.Y+clip.Height {
		app.cursor = CursorState{
			Visible:  true,
			Position: Point{X: cursorX, Y: cursorY},
		}
	}
}

func textAreaVisualRows(
	cells []string,
	width int,
	wrap TextWrap,
) []textAreaVisualRow {
	lineCount := 1
	for _, cell := range cells {
		if cell == "\n" {
			lineCount++
		}
	}
	rows := make([]textAreaVisualRow, 0, lineCount)
	lineStart := 0
	for lineStart <= len(cells) {
		lineEnd := lineStart
		for lineEnd < len(cells) && cells[lineEnd] != "\n" {
			lineEnd++
		}
		rows = append(rows, textAreaLineRows(
			cells,
			lineStart,
			lineEnd,
			width,
			wrap,
		)...)
		if lineEnd == len(cells) {
			break
		}
		lineStart = lineEnd + 1
	}
	return rows
}

func textAreaLineRows(
	cells []string,
	start int,
	end int,
	width int,
	wrap TextWrap,
) []textAreaVisualRow {
	if wrap == TextWrapNone || width <= 0 {
		return []textAreaVisualRow{{start: start, end: end}}
	}
	if start == end {
		return []textAreaVisualRow{{start: start, end: end}}
	}
	rows := make([]textAreaVisualRow, 0, 1+(end-start)/max(1, width))
	for current := start; current < end; {
		next := min(end, current+width)
		if wrap == TextWrapWords && next < end {
			for candidate := next; candidate > current; candidate-- {
				if isTextAreaSpace(cells[candidate-1]) {
					next = candidate
					break
				}
			}
		}
		if next <= current {
			next = min(end, current+width)
		}
		rows = append(rows, textAreaVisualRow{
			start: current, end: next, continues: next < end,
		})
		current = next
	}
	return rows
}

func isTextAreaSpace(cell string) bool {
	current, _ := utf8.DecodeRuneInString(cell)
	return unicode.IsSpace(current)
}

func textAreaCaretPosition(
	rows []textAreaVisualRow,
	caret int,
) (rowIndex int, column int) {
	for index, row := range rows {
		if caret < row.end ||
			(caret == row.end && !row.continues) {
			return index, max(0, caret-row.start)
		}
	}
	if len(rows) == 0 {
		return 0, 0
	}
	last := rows[len(rows)-1]
	return len(rows) - 1, max(0, last.end-last.start)
}

func textAreaViewport(
	behavior textAreaBehavior,
	width int,
	height int,
) (
	rows []textAreaVisualRow,
	caretRow int,
	caretColumn int,
	rowOffset int,
	columnOffset int,
) {
	current := behavior.current()
	rows = textAreaVisualRows(current.cells, width, behavior.wrap)
	caretRow, caretColumn = textAreaCaretPosition(rows, behavior.caret)
	rowOffset = max(0, behavior.rowOffset)
	columnOffset = max(0, behavior.columnOffset)
	if behavior.wrap != TextWrapNone {
		columnOffset = 0
	}
	if height > 0 {
		if caretRow < rowOffset {
			rowOffset = caretRow
		}
		if caretRow >= rowOffset+height {
			rowOffset = caretRow - height + 1
		}
		rowOffset = min(rowOffset, max(0, len(rows)-height))
	} else {
		rowOffset = 0
	}
	if behavior.wrap == TextWrapNone && width > 0 {
		if caretColumn < columnOffset {
			columnOffset = caretColumn
		}
		if caretColumn >= columnOffset+width {
			columnOffset = caretColumn - width + 1
		}
		maximum := 0
		for _, row := range rows {
			maximum = max(maximum, row.end-row.start-width+1)
		}
		columnOffset = min(columnOffset, max(0, maximum))
	}
	return rows, caretRow, caretColumn, rowOffset, columnOffset
}

func textAreaSelectionRange(
	behavior textAreaBehavior,
) (start, end int, selected bool) {
	if behavior.selectionAnchor < 0 ||
		behavior.selectionAnchor == behavior.caret {
		return behavior.caret, behavior.caret, false
	}
	return min(behavior.selectionAnchor, behavior.caret),
		max(behavior.selectionAnchor, behavior.caret),
		true
}

func textAreaSelectionContains(behavior textAreaBehavior, index int) bool {
	start, end, selected := textAreaSelectionRange(behavior)
	return selected && index >= start && index < end
}

func moveTextAreaCaret(
	behavior *textAreaBehavior,
	next int,
	extend bool,
) bool {
	next = min(max(0, next), len(behavior.working.cells))
	previousCaret := behavior.caret
	previousAnchor := behavior.selectionAnchor
	if extend {
		if behavior.selectionAnchor < 0 {
			behavior.selectionAnchor = behavior.caret
		}
	} else {
		behavior.selectionAnchor = -1
	}
	behavior.caret = next
	if behavior.selectionAnchor == behavior.caret {
		behavior.selectionAnchor = -1
	}
	return previousCaret != behavior.caret ||
		previousAnchor != behavior.selectionAnchor
}

func deleteTextAreaSelection(behavior *textAreaBehavior) bool {
	start, end, selected := textAreaSelectionRange(*behavior)
	if !selected {
		return false
	}
	behavior.working.cells = append(
		behavior.working.cells[:start],
		behavior.working.cells[end:]...,
	)
	behavior.caret = start
	behavior.selectionAnchor = -1
	return true
}

func rebuildTextAreaWorking(behavior *textAreaBehavior) {
	behavior.working.text = strings.Join(behavior.working.cells, "")
}

func (a *App) textAreaInputLocked(
	state *controlState,
	behavior textAreaBehavior,
	key Key,
	held map[Key]bool,
) (command CommandID, target ControlID, handled, changed bool) {
	if state == nil || state != a.focus || !a.focusEligibleLocked(state) {
		return "", "", false, false
	}
	if behavior.readOnly {
		return a.readOnlyTextAreaInputLocked(state, behavior, key, held)
	}
	if !behavior.editing {
		if key != KeyEnter || !noHeldModifiers(held) {
			return "", "", false, false
		}
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = true
		behavior.caret = len(behavior.working.cells)
		behavior.selectionAnchor = -1
		behavior.rowOffset = 0
		behavior.columnOffset = 0
		behavior.preferredColumn = -1
		state.behavior = behavior
		return "", "", true, true
	}
	if held[KeyAlt] || held[KeyMeta] {
		return "", "", false, false
	}
	if held[KeyControl] && !held[KeyShift] && key == "a" {
		if len(behavior.working.cells) == 0 {
			return "", "", true, false
		}
		behavior.selectionAnchor = 0
		behavior.caret = len(behavior.working.cells)
		behavior.preferredColumn = -1
		state.behavior = behavior
		return "", "", true, true
	}
	if held[KeyControl] && key == KeyEnter && !held[KeyShift] {
		valueChanged := behavior.committed.text != behavior.working.text
		behavior.committed = cloneInputText(behavior.working)
		behavior.editing = false
		behavior.selectionAnchor = -1
		behavior.preferredColumn = -1
		state.behavior = behavior
		if valueChanged {
			return behavior.changeCommand, state.id, true, true
		}
		return "", "", true, true
	}
	if held[KeyControl] && (key == KeyHome || key == KeyEnd) {
		next := 0
		if key == KeyEnd {
			next = len(behavior.working.cells)
		}
		changed = moveTextAreaCaret(&behavior, next, held[KeyShift])
		behavior.preferredColumn = -1
		if changed {
			state.behavior = behavior
		}
		return "", "", true, changed
	}
	if held[KeyControl] {
		return "", "", false, false
	}

	handled = true
	switch key {
	case KeyEnter:
		changed = insertTextAreaCells(&behavior, []string{"\n"})
	case KeyEscape:
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = false
		behavior.caret = len(behavior.committed.cells)
		behavior.selectionAnchor = -1
		behavior.rowOffset = 0
		behavior.columnOffset = 0
		behavior.preferredColumn = -1
		changed = true
	case KeyLeft:
		changed = moveTextAreaCaret(
			&behavior,
			max(0, behavior.caret-1),
			held[KeyShift],
		)
		behavior.preferredColumn = -1
	case KeyRight:
		changed = moveTextAreaCaret(
			&behavior,
			min(len(behavior.working.cells), behavior.caret+1),
			held[KeyShift],
		)
		behavior.preferredColumn = -1
	case KeyHome, KeyEnd:
		rows := textAreaVisualRows(
			behavior.working.cells,
			state.bounds.Width,
			behavior.wrap,
		)
		rowIndex, _ := textAreaCaretPosition(rows, behavior.caret)
		next := rows[rowIndex].start
		if key == KeyEnd {
			next = rows[rowIndex].end
		}
		changed = moveTextAreaCaret(&behavior, next, held[KeyShift])
		behavior.preferredColumn = -1
	case KeyUp, KeyDown, KeyPageUp, KeyPageDown:
		changed = moveTextAreaVertical(
			&behavior,
			key,
			held[KeyShift],
			state.bounds.Width,
			state.bounds.Height,
		)
	case KeyBackspace:
		if deleteTextAreaSelection(&behavior) {
			changed = true
		} else if behavior.caret > 0 {
			index := behavior.caret - 1
			behavior.working.cells = append(
				behavior.working.cells[:index],
				behavior.working.cells[behavior.caret:]...,
			)
			behavior.caret--
			changed = true
		}
		behavior.preferredColumn = -1
	case KeyDelete:
		if deleteTextAreaSelection(&behavior) {
			changed = true
		} else if behavior.caret < len(behavior.working.cells) {
			behavior.working.cells = append(
				behavior.working.cells[:behavior.caret],
				behavior.working.cells[behavior.caret+1:]...,
			)
			changed = true
		}
		behavior.preferredColumn = -1
	default:
		text, printable := printableTextForKey(key, held)
		if !printable {
			return "", "", false, false
		}
		inserted, err := normalizeInputText(text)
		if err != nil || len(inserted.cells) != 1 {
			return "", "", true, false
		}
		cell := inserted.cells[0]
		if behavior.validator != nil &&
			behavior.validator.value.Enforcement == TextValidationHard &&
			!textCellAllowed(cell, behavior.validator) {
			return "", "", true, false
		}
		changed = insertTextAreaCells(&behavior, []string{cell})
		behavior.preferredColumn = -1
	}
	if changed {
		rebuildTextAreaWorking(&behavior)
		_, _, _, behavior.rowOffset, behavior.columnOffset =
			textAreaViewport(
				behavior,
				state.bounds.Width,
				state.bounds.Height,
			)
		state.behavior = behavior
	}
	return command, target, handled, changed
}

func (a *App) readOnlyTextAreaInputLocked(
	state *controlState,
	behavior textAreaBehavior,
	key Key,
	held map[Key]bool,
) (command CommandID, target ControlID, handled, changed bool) {
	if held[KeyAlt] || held[KeyMeta] {
		return "", "", false, false
	}
	if held[KeyControl] && !held[KeyShift] && key == "a" {
		if len(behavior.committed.cells) == 0 {
			return "", "", true, false
		}
		behavior.selectionAnchor = 0
		behavior.caret = len(behavior.committed.cells)
		behavior.preferredColumn = -1
		state.behavior = behavior
		return "", "", true, true
	}
	if held[KeyControl] && (key == KeyHome || key == KeyEnd) {
		next := 0
		if key == KeyEnd {
			next = len(behavior.committed.cells)
		}
		changed = moveTextAreaCaret(&behavior, next, held[KeyShift])
		behavior.preferredColumn = -1
	} else if held[KeyControl] {
		return "", "", false, false
	} else {
		handled = true
		switch key {
		case KeyLeft:
			changed = moveTextAreaCaret(
				&behavior,
				max(0, behavior.caret-1),
				held[KeyShift],
			)
			behavior.preferredColumn = -1
		case KeyRight:
			changed = moveTextAreaCaret(
				&behavior,
				min(len(behavior.committed.cells), behavior.caret+1),
				held[KeyShift],
			)
			behavior.preferredColumn = -1
		case KeyHome, KeyEnd:
			rows := textAreaVisualRows(
				behavior.committed.cells,
				state.bounds.Width,
				behavior.wrap,
			)
			rowIndex, _ := textAreaCaretPosition(rows, behavior.caret)
			next := rows[rowIndex].start
			if key == KeyEnd {
				next = rows[rowIndex].end
			}
			changed = moveTextAreaCaret(&behavior, next, held[KeyShift])
			behavior.preferredColumn = -1
		case KeyUp, KeyDown, KeyPageUp, KeyPageDown:
			changed = moveTextAreaVertical(
				&behavior,
				key,
				held[KeyShift],
				state.bounds.Width,
				state.bounds.Height,
			)
		case KeyEscape:
			if behavior.selectionAnchor >= 0 {
				behavior.selectionAnchor = -1
				changed = true
			}
		default:
			return "", "", true, false
		}
	}
	if changed {
		_, _, _, behavior.rowOffset, behavior.columnOffset =
			textAreaViewport(
				behavior,
				state.bounds.Width,
				state.bounds.Height,
			)
		state.behavior = behavior
	}
	return "", "", true, changed
}

func insertTextAreaCells(
	behavior *textAreaBehavior,
	inserted []string,
) bool {
	start, end, selected := textAreaSelectionRange(*behavior)
	if !selected {
		start, end = behavior.caret, behavior.caret
	}
	candidate := make([]string, 0, len(behavior.working.cells)+len(inserted))
	candidate = append(candidate, behavior.working.cells[:start]...)
	candidate = append(candidate, inserted...)
	candidate = append(candidate, behavior.working.cells[end:]...)
	if len(candidate) > MaxTextInputCells ||
		len(strings.Join(candidate, "")) > MaxTextInputBytes {
		return false
	}
	behavior.working.cells = candidate
	behavior.caret = start + len(inserted)
	behavior.selectionAnchor = -1
	return len(inserted) != 0 || selected
}

func moveTextAreaVertical(
	behavior *textAreaBehavior,
	key Key,
	extend bool,
	width int,
	height int,
) bool {
	rows := textAreaVisualRows(behavior.working.cells, width, behavior.wrap)
	row, column := textAreaCaretPosition(rows, behavior.caret)
	if behavior.preferredColumn < 0 {
		behavior.preferredColumn = column
	}
	delta := 0
	switch key {
	case KeyUp:
		delta = -1
	case KeyDown:
		delta = 1
	case KeyPageUp:
		delta = -max(1, height)
	case KeyPageDown:
		delta = max(1, height)
	}
	nextRow := min(max(0, row+delta), len(rows)-1)
	next := min(
		rows[nextRow].end,
		rows[nextRow].start+behavior.preferredColumn,
	)
	return moveTextAreaCaret(behavior, next, extend)
}

// Text returns the committed application value.
func (a *TextArea) Text() string {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	if !ok {
		return ""
	}
	return behavior.committed.text
}

// SetText atomically replaces the committed value without notification.
func (a *TextArea) SetText(text string) error {
	if a == nil || a.controlState() == nil {
		return ErrInvalidControl
	}
	tx := a.controlState().app.NewTransaction()
	if err := tx.SetText(a, text); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Validator returns a caller-owned copy of the optional validator.
func (a *TextArea) Validator() *TextValidator {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	if !ok || behavior.validator == nil {
		return nil
	}
	value := behavior.validator.value
	return &value
}

// SetValidator atomically replaces the optional validator.
func (a *TextArea) SetValidator(validator *TextValidator) error {
	if a == nil || a.controlState() == nil {
		return ErrInvalidControl
	}
	tx := a.controlState().app.NewTransaction()
	if err := tx.SetTextAreaValidator(a, validator); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Password reports whether content is masked and snapshot-redacted.
func (a *TextArea) Password() bool {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	return ok && behavior.password
}

// SetPassword atomically changes password presentation and redaction.
func (a *TextArea) SetPassword(password bool) error {
	if a == nil || a.controlState() == nil {
		return ErrInvalidControl
	}
	tx := a.controlState().app.NewTransaction()
	if err := tx.SetTextAreaPassword(a, password); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// ReadOnly reports whether user input may navigate but not edit the area.
func (a *TextArea) ReadOnly() bool {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	return ok && behavior.readOnly
}

// SetReadOnly atomically changes user editability while retaining focusability.
func (a *TextArea) SetReadOnly(readOnly bool) error {
	if a == nil || a.controlState() == nil {
		return ErrInvalidControl
	}
	tx := a.controlState().app.NewTransaction()
	if err := tx.SetTextAreaReadOnly(a, readOnly); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Wrap returns the current multiline wrapping policy.
func (a *TextArea) Wrap() TextWrap {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	if !ok {
		return TextWrapNone
	}
	return behavior.wrap
}

// SetWrap atomically changes the multiline wrapping policy.
func (a *TextArea) SetWrap(wrap TextWrap) error {
	if a == nil || a.controlState() == nil {
		return ErrInvalidControl
	}
	tx := a.controlState().app.NewTransaction()
	if err := tx.SetTextAreaWrap(a, wrap); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Editing reports whether the area currently captures text input.
func (a *TextArea) Editing() bool {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	return ok && behavior.editing
}

// Valid reports validation of the current working or committed value.
func (a *TextArea) Valid() bool {
	behavior, ok := textAreaBehaviorForRead(a.controlState())
	return ok && textAreaCellsValid(
		behavior.current().cells,
		behavior.validator,
	)
}

// Focus gives this eligible TextArea keyboard focus.
func (a *TextArea) Focus() error { return focusSelectionControl(a) }

// Activate focuses this eligible TextArea and enters edit mode unless read-only.
func (a *TextArea) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if a == nil || a.controlState() == nil {
		return Completion{}, ErrInvalidControl
	}
	return a.controlState().app.activateTextArea(
		ctx,
		source,
		requestID,
		a.controlState(),
	)
}

func textAreaBehaviorForRead(
	state *controlState,
) (textAreaBehavior, bool) {
	if state == nil || state.app == nil {
		return textAreaBehavior{}, false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(textAreaBehavior)
	return behavior, ok && !state.aborted && !state.destroyed
}

func (t *Transaction) recordTextAreaBehavior(
	state *controlState,
	behavior textAreaBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationTextArea, state: state, behavior: behavior,
	})
	return nil
}

// SetTextAreaValidator records one atomic validator replacement.
func (t *Transaction) SetTextAreaValidator(
	area *TextArea,
	validator *TextValidator,
) error {
	state, err := t.control(area)
	if err != nil || state.kind != ControlTextArea {
		return ErrInvalidControl
	}
	normalized, err := normalizeTextValidator(validator)
	if err != nil {
		return err
	}
	behavior, ok := t.recordedControlBehavior(state).(textAreaBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if normalized != nil &&
		normalized.value.Enforcement == TextValidationHard &&
		!textAreaCellsValid(behavior.committed.cells, normalized) {
		return fmt.Errorf(
			"%w: current TextArea value violates hard validation",
			ErrValidation,
		)
	}
	behavior.validator = normalized
	behavior.working = cloneInputText(behavior.committed)
	behavior.editing = false
	behavior.caret = len(behavior.committed.cells)
	if behavior.readOnly {
		behavior.caret = 0
	}
	behavior.selectionAnchor = -1
	behavior.rowOffset = 0
	behavior.columnOffset = 0
	behavior.preferredColumn = -1
	return t.recordTextAreaBehavior(state, behavior)
}

// SetTextAreaPassword records one atomic password-presentation replacement.
func (t *Transaction) SetTextAreaPassword(
	area *TextArea,
	password bool,
) error {
	state, err := t.control(area)
	if err != nil || state.kind != ControlTextArea {
		return ErrInvalidControl
	}
	behavior, ok := t.recordedControlBehavior(state).(textAreaBehavior)
	if !ok {
		return ErrInvalidControl
	}
	behavior.password = password
	return t.recordTextAreaBehavior(state, behavior)
}

// SetTextAreaReadOnly records one atomic user-editability replacement.
func (t *Transaction) SetTextAreaReadOnly(
	area *TextArea,
	readOnly bool,
) error {
	state, err := t.control(area)
	if err != nil || state.kind != ControlTextArea {
		return ErrInvalidControl
	}
	behavior, ok := t.recordedControlBehavior(state).(textAreaBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if readOnly && behavior.editing {
		behavior.committed = cloneInputText(behavior.working)
		behavior.editing = false
		behavior.selectionAnchor = -1
		behavior.preferredColumn = -1
	}
	behavior.readOnly = readOnly
	behavior.working = cloneInputText(behavior.committed)
	behavior.caret = min(behavior.caret, len(behavior.committed.cells))
	return t.recordTextAreaBehavior(state, behavior)
}

// SetTextAreaWrap records one atomic wrapping-policy replacement.
func (t *Transaction) SetTextAreaWrap(area *TextArea, wrap TextWrap) error {
	state, err := t.control(area)
	if err != nil || state.kind != ControlTextArea {
		return ErrInvalidControl
	}
	normalized, err := normalizeTextWrap(wrap)
	if err != nil {
		return err
	}
	behavior, ok := t.recordedControlBehavior(state).(textAreaBehavior)
	if !ok {
		return ErrInvalidControl
	}
	behavior.wrap = normalized
	behavior.rowOffset = 0
	behavior.columnOffset = 0
	behavior.preferredColumn = -1
	return t.recordTextAreaBehavior(state, behavior)
}

func (a *App) activateTextArea(
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
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	if state == nil || state.app != a || !a.focusEligibleLocked(state) {
		return Completion{}, ErrNotFocusable
	}
	changed := a.focus != state
	if changed && a.commitOrCancelEditorStateLocked(a.focus) {
		changed = true
	}
	a.focus = state
	behavior := state.behavior.(textAreaBehavior)
	if !behavior.editing && !behavior.readOnly {
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = true
		behavior.caret = len(behavior.working.cells)
		behavior.selectionAnchor = -1
		behavior.rowOffset = 0
		behavior.columnOffset = 0
		behavior.preferredColumn = -1
		state.behavior = behavior
		changed = true
	}
	outcome := OutcomeNoOp
	if changed {
		outcome = OutcomeApplied
	}
	return a.associateLocked(
		requestID,
		CommandResult{Outcome: outcome},
		"",
	), nil
}

func (a *App) textAreaDetailsLocked(
	state *controlState,
	behavior textAreaBehavior,
) TextAreaDetails {
	current := behavior.current()
	_, valid := textAreaValidationMask(current.cells, behavior.validator)
	_, caretRow, caretColumn, rowOffset, columnOffset :=
		textAreaViewport(
			behavior,
			state.bounds.Width,
			state.bounds.Height,
		)
	start, end, _ := textAreaSelectionRange(behavior)
	details := TextAreaDetails{
		Length:            len(current.cells),
		LineCount:         1 + strings.Count(current.text, "\n"),
		Caret:             behavior.caret,
		SelectionStart:    start,
		SelectionEnd:      end,
		VisualCaretRow:    caretRow,
		VisualCaretColumn: caretColumn,
		RowOffset:         rowOffset,
		ColumnOffset:      columnOffset,
		Wrap:              behavior.wrap,
		Editing:           behavior.editing,
		Valid:             valid,
		Password:          behavior.password,
		ReadOnly:          behavior.readOnly,
		Redacted:          behavior.password,
		Enabled:           !behavior.disabled,
		DisabledReason:    behavior.disabledReason,
		ChangeCommand:     behavior.changeCommand,
	}
	if !behavior.password {
		details.Text = current.text
	}
	if behavior.validator != nil {
		details.Validator = &TextValidatorDetails{
			Enforcement: behavior.validator.value.Enforcement,
			Mode:        behavior.validator.value.Mode,
			Characters:  behavior.validator.value.Characters,
		}
	}
	return details
}

func textAreaBehaviorEqual(left, right textAreaBehavior) bool {
	return left.committed.text == right.committed.text &&
		left.working.text == right.working.text &&
		textValidatorEqual(left.validator, right.validator) &&
		left.password == right.password &&
		left.readOnly == right.readOnly &&
		left.wrap == right.wrap &&
		left.disabled == right.disabled &&
		left.disabledReason == right.disabledReason &&
		left.changeCommand == right.changeCommand &&
		left.editing == right.editing &&
		left.caret == right.caret &&
		left.selectionAnchor == right.selectionAnchor &&
		left.rowOffset == right.rowOffset &&
		left.columnOffset == right.columnOffset &&
		left.preferredColumn == right.preferredColumn
}
