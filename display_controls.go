package expletives

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
)

// LabelOptions configures one single-line non-container Label.
type LabelOptions struct {
	PanelOptions
	Text                string
	HorizontalAlignment TextAlignment
	VerticalAlignment   TextAlignment
	Target              Control
	Mnemonic            Key
}

// StaticTextOptions configures multiline optionally wrapped display text.
type StaticTextOptions struct {
	PanelOptions
	Text                string
	HorizontalAlignment TextAlignment
	VerticalAlignment   TextAlignment
	Wrap                TextWrap
}

// SeparatorOptions configures one untitled structural divider.
type SeparatorOptions struct {
	PanelOptions
	Orientation Orientation
	Form        BorderForm
}

// RuleOptions configures one titled structural divider.
type RuleOptions struct {
	PanelOptions
	Orientation Orientation
	Form        BorderForm
	Text        string
	Alignment   TextAlignment
}

// Label is a copy-safe non-container control handle.
type Label struct{ controlHandle }

// StaticText is a copy-safe non-container control handle.
type StaticText struct{ controlHandle }

// Separator is a copy-safe non-container control handle.
type Separator struct{ controlHandle }

// Rule is a copy-safe non-container control handle.
type Rule struct{ controlHandle }

type normalizedDisplayText struct {
	text  string
	lines [][]string
	cells int
}

type intrinsicMinimumBehavior interface {
	intrinsicMinimum() Size
}

type mutableTextBehavior interface {
	controlBehavior
	intrinsicMinimumBehavior
	withText(string) (controlBehavior, error)
}

type textBehavior struct {
	content    normalizedDisplayText
	horizontal TextAlignment
	vertical   TextAlignment
	wrap       TextWrap
	target     *controlState
	mnemonic   Key
	multiline  bool
}

type dividerBehavior struct {
	orientation Orientation
	form        BorderForm
	content     normalizedDisplayText
	alignment   TextAlignment
	mutable     bool
}

// NewLabel constructs and atomically inserts a Label.
func NewLabel(parent Container, options LabelOptions) (*Label, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	label, err := tx.NewLabel(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return label, nil
}

// NewStaticText constructs and atomically inserts StaticText.
func NewStaticText(
	parent Container,
	options StaticTextOptions,
) (*StaticText, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	staticText, err := tx.NewStaticText(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return staticText, nil
}

// NewSeparator constructs and atomically inserts a Separator.
func NewSeparator(
	parent Container,
	options SeparatorOptions,
) (*Separator, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	separator, err := tx.NewSeparator(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return separator, nil
}

// NewRule constructs and atomically inserts a Rule.
func NewRule(parent Container, options RuleOptions) (*Rule, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	rule, err := tx.NewRule(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return rule, nil
}

// NewLabel records construction of a provisional Label.
func (t *Transaction) NewLabel(
	parent Container,
	options LabelOptions,
) (*Label, error) {
	horizontal, err := normalizeTextAlignment(options.HorizontalAlignment)
	if err != nil {
		return nil, err
	}
	vertical, err := normalizeTextAlignment(options.VerticalAlignment)
	if err != nil {
		return nil, err
	}
	mnemonic, err := normalizeMnemonic(options.Mnemonic)
	if err != nil {
		return nil, err
	}
	var target *controlState
	if options.Target != nil {
		target, err = t.control(options.Target)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid Label target", ErrInvalidControl)
		}
	}
	if mnemonic != "" && target == nil {
		return nil, fmt.Errorf("%w: Label mnemonic requires a target", ErrInvalidControl)
	}
	content, err := normalizeDisplayText(options.Text, false)
	if err != nil {
		return nil, err
	}
	behavior := textBehavior{
		content:    content,
		horizontal: horizontal,
		vertical:   vertical,
		wrap:       TextWrapNone,
		target:     target,
		mnemonic:   mnemonic,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlLabel,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	label := &Label{controlHandle: controlHandle{state: state}}
	state.control = label
	return label, nil
}

// NewStaticText records construction of provisional StaticText.
func (t *Transaction) NewStaticText(
	parent Container,
	options StaticTextOptions,
) (*StaticText, error) {
	horizontal, err := normalizeTextAlignment(options.HorizontalAlignment)
	if err != nil {
		return nil, err
	}
	vertical, err := normalizeTextAlignment(options.VerticalAlignment)
	if err != nil {
		return nil, err
	}
	wrap, err := normalizeTextWrap(options.Wrap)
	if err != nil {
		return nil, err
	}
	content, err := normalizeDisplayText(options.Text, true)
	if err != nil {
		return nil, err
	}
	behavior := textBehavior{
		content:    content,
		horizontal: horizontal,
		vertical:   vertical,
		wrap:       wrap,
		multiline:  true,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlStaticText,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	staticText := &StaticText{controlHandle: controlHandle{state: state}}
	state.control = staticText
	return staticText, nil
}

// NewSeparator records construction of a provisional Separator.
func (t *Transaction) NewSeparator(
	parent Container,
	options SeparatorOptions,
) (*Separator, error) {
	orientation, err := normalizeDividerOrientation(options.Orientation)
	if err != nil {
		return nil, err
	}
	form, err := normalizeBorderForm(options.Form, BorderSingle)
	if err != nil {
		return nil, err
	}
	behavior := dividerBehavior{
		orientation: orientation,
		form:        form,
		alignment:   TextAlignStart,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlSeparator,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	separator := &Separator{controlHandle: controlHandle{state: state}}
	state.control = separator
	return separator, nil
}

// NewRule records construction of a provisional Rule.
func (t *Transaction) NewRule(
	parent Container,
	options RuleOptions,
) (*Rule, error) {
	orientation, err := normalizeDividerOrientation(options.Orientation)
	if err != nil {
		return nil, err
	}
	form, err := normalizeBorderForm(options.Form, BorderSingle)
	if err != nil {
		return nil, err
	}
	alignment, err := normalizeTextAlignment(options.Alignment)
	if err != nil {
		return nil, err
	}
	content, err := normalizeDisplayText(options.Text, false)
	if err != nil {
		return nil, err
	}
	behavior := dividerBehavior{
		orientation: orientation,
		form:        form,
		content:     content,
		alignment:   alignment,
		mutable:     true,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlRule,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	rule := &Rule{controlHandle: controlHandle{state: state}}
	state.control = rule
	return rule, nil
}

func (t *Transaction) newLeafControl(
	parent Container,
	options PanelOptions,
	kind ControlKind,
	behavior controlBehavior,
) (*controlState, error) {
	panel, err := t.newControl(parent, options, kind, behavior)
	if err != nil {
		return nil, err
	}
	state := panel.controlState()
	state.container = nil
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.(intrinsicMinimumBehavior).intrinsicMinimum()
	}
	return state, nil
}

// Text returns the Label's canonical single-line text.
func (l *Label) Text() string { return controlText(l.controlState()) }

// Text returns StaticText's canonical multiline text.
func (s *StaticText) Text() string { return controlText(s.controlState()) }

// Text returns the Rule's canonical title.
func (r *Rule) Text() string { return controlText(r.controlState()) }

// SetText atomically replaces the Label text.
func (l *Label) SetText(text string) error { return setControlText(l, text) }

// SetText atomically replaces the StaticText text.
func (s *StaticText) SetText(text string) error { return setControlText(s, text) }

// SetText atomically replaces the Rule title.
func (r *Rule) SetText(text string) error { return setControlText(r, text) }

func setControlText(control Control, text string) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	tx := state.app.NewTransaction()
	if err := tx.SetText(control, text); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

func controlText(state *controlState) string {
	if state == nil || state.app == nil {
		return ""
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	if state.aborted {
		return ""
	}
	switch behavior := state.behavior.(type) {
	case textBehavior:
		return behavior.content.text
	case dividerBehavior:
		return behavior.content.text
	default:
		return ""
	}
}

func normalizeDisplayText(
	text string,
	multiline bool,
) (normalizedDisplayText, error) {
	if len(text) > MaxDisplayTextBytes {
		return normalizedDisplayText{}, fmt.Errorf(
			"%w: display text exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxDisplayTextBytes,
		)
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	for _, current := range text {
		if current == '\n' {
			if multiline {
				continue
			}
			return normalizedDisplayText{}, fmt.Errorf(
				"%w: single-line display text contains a line break",
				ErrTextLimit,
			)
		}
		if unicode.IsControl(current) {
			return normalizedDisplayText{}, fmt.Errorf(
				"%w: display text contains a control character",
				ErrTextLimit,
			)
		}
	}

	sourceLines := strings.Split(text, "\n")
	lines := make([][]string, len(sourceLines))
	normalizedLines := make([]string, len(sourceLines))
	cellCount := max(0, len(sourceLines)-1)
	for index, sourceLine := range sourceLines {
		cells := display.Normalize(sourceLine)
		for _, cell := range cells {
			if len(cell) > MaxCellBytes {
				return normalizedDisplayText{}, fmt.Errorf(
					"%w: one display cell exceeds %d UTF-8 bytes",
					ErrTextLimit,
					MaxCellBytes,
				)
			}
		}
		cellCount += len(cells)
		if cellCount > MaxDisplayTextCells {
			return normalizedDisplayText{}, fmt.Errorf(
				"%w: display text exceeds %d canonical cells",
				ErrTextLimit,
				MaxDisplayTextCells,
			)
		}
		lines[index] = append([]string(nil), cells...)
		normalizedLines[index] = strings.Join(cells, "")
	}
	normalized := strings.Join(normalizedLines, "\n")
	if len(normalized) > MaxDisplayTextBytes {
		return normalizedDisplayText{}, fmt.Errorf(
			"%w: normalized display text exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxDisplayTextBytes,
		)
	}
	return normalizedDisplayText{
		text: normalized, lines: lines, cells: cellCount,
	}, nil
}

func normalizeTextAlignment(alignment TextAlignment) (TextAlignment, error) {
	if alignment == TextAlignDefault {
		return TextAlignStart, nil
	}
	switch alignment {
	case TextAlignStart, TextAlignCenter, TextAlignEnd:
		return alignment, nil
	default:
		return "", fmt.Errorf(
			"%w: invalid text alignment %q",
			ErrInvalidControl,
			alignment,
		)
	}
}

func normalizeTextWrap(wrap TextWrap) (TextWrap, error) {
	if wrap == TextWrapDefault {
		return TextWrapNone, nil
	}
	switch wrap {
	case TextWrapNone, TextWrapWords, TextWrapCells:
		return wrap, nil
	default:
		return "", fmt.Errorf(
			"%w: invalid text wrapping %q",
			ErrInvalidControl,
			wrap,
		)
	}
}

func normalizeMnemonic(key Key) (Key, error) {
	value := string(key)
	if len(value) == 1 && value[0] >= 'A' && value[0] <= 'Z' {
		value = strings.ToLower(value)
	}
	if value == "" {
		return "", nil
	}
	if len(value) == 1 &&
		((value[0] >= 'a' && value[0] <= 'z') ||
			(value[0] >= '0' && value[0] <= '9')) {
		return Key(value), nil
	}
	return "", fmt.Errorf(
		"%w: mnemonic must be one ASCII letter or digit",
		ErrInvalidKeyEvent,
	)
}

func normalizeDividerOrientation(orientation Orientation) (Orientation, error) {
	if orientation != Horizontal && orientation != Vertical {
		return 0, fmt.Errorf("%w: invalid divider orientation", ErrInvalidControl)
	}
	return orientation, nil
}

func (b textBehavior) clientInset() int { return 0 }

func (b textBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	rows := b.content.lines
	if b.wrap != TextWrapNone {
		rows = wrapTextRows(rows, absolute.Width, b.wrap)
	}
	app.paintTextRowsLocked(
		frame,
		state,
		absolute,
		clip,
		rows,
		b.horizontal,
		b.vertical,
	)
}

func (b textBehavior) details() ControlDetails {
	var target ControlID
	if b.target != nil && !b.target.destroyed && !b.target.aborted {
		target = b.target.id
	}
	return ControlDetails{
		Version: ControlDetailsVersion,
		Text: &TextDetails{
			Text:                b.content.text,
			HorizontalAlignment: b.horizontal,
			VerticalAlignment:   b.vertical,
			Wrap:                b.wrap,
			Target:              target,
			Mnemonic:            b.mnemonic,
		},
	}
}

func (b textBehavior) intrinsicMinimum() Size {
	height := len(b.content.lines)
	width := 0
	for _, line := range b.content.lines {
		width = max(width, len(line))
	}
	if b.wrap != TextWrapNone && width > 0 {
		width = 1
	}
	return Size{Width: width, Height: height}
}

func (b textBehavior) withText(text string) (controlBehavior, error) {
	content, err := normalizeDisplayText(text, b.multiline)
	if err != nil {
		return nil, err
	}
	b.content = content
	return b, nil
}

func (b dividerBehavior) clientInset() int { return 0 }

func (b dividerBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	app.paintDividerLocked(frame, state, absolute, clip, b)
}

func (b dividerBehavior) details() ControlDetails {
	return ControlDetails{
		Version: ControlDetailsVersion,
		Divider: &DividerDetails{
			Orientation: b.orientation,
			Form:        b.form,
			Text:        b.content.text,
			Alignment:   b.alignment,
		},
	}
}

func (b dividerBehavior) intrinsicMinimum() Size {
	length := 1
	if b.content.cells > 0 {
		length = b.content.cells + 2
	}
	if b.orientation == Vertical {
		return Size{Width: 1, Height: length}
	}
	return Size{Width: length, Height: 1}
}

func (b dividerBehavior) withText(text string) (controlBehavior, error) {
	if !b.mutable {
		return nil, fmt.Errorf("%w: Separator has no text", ErrInvalidControl)
	}
	content, err := normalizeDisplayText(text, false)
	if err != nil {
		return nil, err
	}
	b.content = content
	return b, nil
}

func controlBehaviorEqual(left, right controlBehavior) bool {
	switch leftValue := left.(type) {
	case textBehavior:
		rightValue, ok := right.(textBehavior)
		return ok &&
			leftValue.content.text == rightValue.content.text &&
			leftValue.horizontal == rightValue.horizontal &&
			leftValue.vertical == rightValue.vertical &&
			leftValue.wrap == rightValue.wrap &&
			leftValue.target == rightValue.target &&
			leftValue.mnemonic == rightValue.mnemonic &&
			leftValue.multiline == rightValue.multiline
	case dividerBehavior:
		rightValue, ok := right.(dividerBehavior)
		return ok &&
			leftValue.orientation == rightValue.orientation &&
			leftValue.form == rightValue.form &&
			leftValue.content.text == rightValue.content.text &&
			leftValue.alignment == rightValue.alignment &&
			leftValue.mutable == rightValue.mutable
	case textFieldBehavior:
		rightValue, ok := right.(textFieldBehavior)
		return ok && textFieldBehaviorEqual(leftValue, rightValue)
	case numberFieldBehavior:
		rightValue, ok := right.(numberFieldBehavior)
		return ok && numberFieldBehaviorEqual(leftValue, rightValue)
	case textAreaBehavior:
		rightValue, ok := right.(textAreaBehavior)
		return ok && textAreaBehaviorEqual(leftValue, rightValue)
	case progressBehavior:
		rightValue, ok := right.(progressBehavior)
		return ok && progressBehaviorEqual(leftValue, rightValue)
	case scrollBarBehavior:
		rightValue, ok := right.(scrollBarBehavior)
		return ok && scrollBarBehaviorEqual(leftValue, rightValue)
	default:
		return selectionBehaviorEqual(left, right)
	}
}

func controlBehaviorTarget(behavior controlBehavior) *controlState {
	if text, ok := behavior.(textBehavior); ok {
		return text.target
	}
	return nil
}

func clearControlBehaviorTarget(
	behavior controlBehavior,
	destroyed map[*controlState]bool,
) (controlBehavior, bool) {
	text, ok := behavior.(textBehavior)
	if !ok || text.target == nil || !destroyed[text.target] {
		return behavior, false
	}
	text.target = nil
	text.mnemonic = ""
	return text, true
}

func wrapTextRows(
	lines [][]string,
	width int,
	wrap TextWrap,
) [][]string {
	if width <= 0 {
		return nil
	}
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			rows = append(rows, []string{})
			continue
		}
		remaining := append([]string(nil), line...)
		for len(remaining) > width {
			breakAt := width
			if wrap == TextWrapWords {
				for index := width - 1; index > 0; index-- {
					if remaining[index] == " " {
						breakAt = index
						break
					}
				}
			}
			row := append([]string(nil), remaining[:breakAt]...)
			for len(row) > 0 && row[len(row)-1] == " " {
				row = row[:len(row)-1]
			}
			rows = append(rows, row)
			remaining = remaining[breakAt:]
			if wrap == TextWrapWords {
				for len(remaining) > 0 && remaining[0] == " " {
					remaining = remaining[1:]
				}
			}
		}
		rows = append(rows, append([]string(nil), remaining...))
	}
	return rows
}

func alignedOffset(available, content int, alignment TextAlignment) int {
	difference := available - content
	switch alignment {
	case TextAlignCenter:
		return difference / 2
	case TextAlignEnd:
		return difference
	default:
		return 0
	}
}

func (a *App) paintTextRowsLocked(
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
	rows [][]string,
	horizontal TextAlignment,
	vertical TextAlignment,
) {
	resolved := a.styles[state.style]
	startY := absolute.Y + alignedOffset(absolute.Height, len(rows), vertical)
	for rowIndex, row := range rows {
		y := startY + rowIndex
		startX := absolute.X + alignedOffset(absolute.Width, len(row), horizontal)
		for cellIndex, grapheme := range row {
			a.setClippedCellLocked(
				frame,
				clip,
				startX+cellIndex,
				y,
				grapheme,
				state.style,
				resolved,
				state.id,
			)
		}
	}
}

func (a *App) paintDividerLocked(
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
	behavior dividerBehavior,
) {
	if absolute.Empty() {
		return
	}
	resolved := a.styles[state.style]
	glyphs := glyphsForBorder(behavior.form)
	if behavior.orientation == Horizontal {
		y := absolute.Y + (absolute.Height-1)/2
		for x := absolute.X; x < absolute.X+absolute.Width; x++ {
			a.setClippedCellLocked(
				frame, clip, x, y, glyphs.horizontal,
				state.style, resolved, state.id,
			)
		}
		a.paintRuleTextLocked(
			frame, state, clip, behavior, y, absolute.X, absolute.Width, true,
		)
		return
	}
	x := absolute.X + (absolute.Width-1)/2
	for y := absolute.Y; y < absolute.Y+absolute.Height; y++ {
		a.setClippedCellLocked(
			frame, clip, x, y, glyphs.vertical,
			state.style, resolved, state.id,
		)
	}
	a.paintRuleTextLocked(
		frame, state, clip, behavior, x, absolute.Y, absolute.Height, false,
	)
}

func (a *App) paintRuleTextLocked(
	frame *IntendedFrame,
	state *controlState,
	clip Rect,
	behavior dividerBehavior,
	fixed,
	mainStart,
	mainLength int,
	horizontal bool,
) {
	if len(behavior.content.lines) == 0 ||
		len(behavior.content.lines[0]) == 0 ||
		mainLength <= 0 {
		return
	}
	text := behavior.content.lines[0]
	decoratedLength := len(text) + 2
	start := mainStart + alignedOffset(
		mainLength,
		decoratedLength,
		behavior.alignment,
	)
	resolved := a.styles[state.style]
	for offset := 0; offset < decoratedLength; offset++ {
		grapheme := " "
		if offset > 0 && offset < decoratedLength-1 {
			grapheme = text[offset-1]
		}
		x, y := fixed, start+offset
		if horizontal {
			x, y = start+offset, fixed
		}
		a.setClippedCellLocked(
			frame, clip, x, y, grapheme,
			state.style, resolved, state.id,
		)
	}
}
