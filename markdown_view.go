package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
)

// MarkdownViewOptions configures one read-only, bounded Markdown leaf.
//
// ContentAutomationKey, ContentStyle, and State.ContentSize are reserved by
// the underlying scroll implementation and must be empty. State.Offset may be
// supplied as an initial requested position and is clamped after layout.
type MarkdownViewOptions struct {
	ScrollablePanelOptions
	Markdown string
}

// MarkdownView is a copy-safe, read-only Markdown control.
type MarkdownView struct{ controlHandle }

type markdownBlock struct {
	kind        string
	level       int
	sourceLine  int
	sourceLines int
	lines       []string
	marker      string
}

type markdownCell struct {
	grapheme string
	style    StyleID
}

type markdownRenderedBlock struct {
	details MarkdownBlockDetails
	rows    [][]markdownCell
}

type markdownBehavior struct {
	scroll      scrollViewBehavior
	source      string
	sourceCells int
	blocks      []markdownBlock
	rendered    []markdownRenderedBlock
	rows        [][]markdownCell
	maxLine     int
}

// NewMarkdownView constructs and atomically inserts a MarkdownView.
func NewMarkdownView(
	parent Container,
	options MarkdownViewOptions,
) (*MarkdownView, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewMarkdownView(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewMarkdownView records construction of a provisional MarkdownView.
func (t *Transaction) NewMarkdownView(
	parent Container,
	options MarkdownViewOptions,
) (*MarkdownView, error) {
	scrollOptions := options.ScrollViewOptions
	if scrollOptions.ContentAutomationKey != "" ||
		scrollOptions.ContentStyle != "" ||
		scrollOptions.State.ContentSize != (Size{}) {
		return nil, fmt.Errorf(
			"%w: MarkdownView derives its content identity, style, and extent",
			ErrValidation,
		)
	}
	requestedOffset := scrollOptions.State.Offset
	if requestedOffset.X < 0 || requestedOffset.Y < 0 ||
		requestedOffset.X > maxCoordinateMagnitude ||
		requestedOffset.Y > maxCoordinateMagnitude {
		return nil, fmt.Errorf(
			"%w: invalid MarkdownView offset",
			ErrValidation,
		)
	}
	scrollOptions.State.Offset = Point{}
	scroll, err := normalizeScrollViewBehavior(scrollViewBehavior{
		state:            scrollOptions.State,
		arrowStep:        scrollOptions.ArrowStep,
		pageStep:         scrollOptions.PageStep,
		disabled:         scrollOptions.Disabled,
		disabledReason:   scrollOptions.DisabledReason,
		changeCommand:    scrollOptions.ChangeCommand,
		horizontalPolicy: options.HorizontalBar,
		verticalPolicy:   options.VerticalBar,
		integratedBars:   true,
	})
	if err != nil {
		return nil, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		ControlMarkdownView,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return nil, err
	}
	scroll.border = border
	source, sourceCells, err := normalizeMarkdownSource(options.Markdown)
	if err != nil {
		return nil, err
	}
	blocks, err := parseMarkdown(source)
	if err != nil {
		return nil, err
	}
	behavior := markdownBehavior{
		scroll:      scroll,
		source:      source,
		sourceCells: sourceCells,
		blocks:      blocks,
	}
	behavior.scroll.state.Offset = requestedOffset
	behavior = reflowMarkdown(behavior, scrollOptions.Bounds.Size())
	panel, err := t.newControl(
		parent,
		scrollOptions.PanelOptions,
		ControlMarkdownView,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &MarkdownView{controlHandle: panel.controlHandle}
	panel.state.control = control
	if scrollOptions.MinimumSize == (Size{}) {
		panel.state.autoMinimum = true
		panel.state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func normalizeMarkdownSource(source string) (string, int, error) {
	if len(source) > MaxContentBytes {
		return "", 0, fmt.Errorf(
			"%w: Markdown source exceeds %d bytes",
			ErrTextLimit,
			MaxContentBytes,
		)
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	lines := strings.Split(source, "\n")
	cells := 0
	for index := range lines {
		normalized := display.Normalize(lines[index])
		cells += len(normalized)
		lines[index] = strings.Join(normalized, "")
	}
	source = strings.Join(lines, "\n")
	if !utf8.ValidString(source) || len(source) > MaxContentBytes {
		return "", 0, fmt.Errorf(
			"%w: normalized Markdown source exceeds %d bytes",
			ErrTextLimit,
			MaxContentBytes,
		)
	}
	return source, cells, nil
}

func parseMarkdown(source string) ([]markdownBlock, error) {
	lines := strings.Split(source, "\n")
	blocks := make([]markdownBlock, 0, min(len(lines), MaxMarkdownBlocks))
	appendBlock := func(block markdownBlock) error {
		if len(blocks) >= MaxMarkdownBlocks {
			return fmt.Errorf(
				"%w: Markdown exceeds %d blocks",
				ErrTextLimit,
				MaxMarkdownBlocks,
			)
		}
		blocks = append(blocks, block)
		return nil
	}
	for index := 0; index < len(lines); {
		line := lines[index]
		sourceLine := index + 1
		if strings.TrimSpace(line) == "" {
			if err := appendBlock(markdownBlock{
				kind: "blank", sourceLine: sourceLine, sourceLines: 1,
			}); err != nil {
				return nil, err
			}
			index++
			continue
		}
		if marker, ok := markdownFenceMarker(line); ok {
			start := index
			index++
			code := make([]string, 0)
			for index < len(lines) &&
				!markdownFenceCloses(lines[index], marker) {
				code = append(code, lines[index])
				index++
			}
			if index < len(lines) {
				index++
			}
			if err := appendBlock(markdownBlock{
				kind:        "code",
				sourceLine:  sourceLine,
				sourceLines: index - start,
				lines:       code,
			}); err != nil {
				return nil, err
			}
			continue
		}
		if level, text, ok := markdownHeading(line); ok {
			if err := appendBlock(markdownBlock{
				kind: "heading", level: level, sourceLine: sourceLine,
				sourceLines: 1, lines: []string{text},
			}); err != nil {
				return nil, err
			}
			index++
			continue
		}
		if markdownRule(line) {
			if err := appendBlock(markdownBlock{
				kind: "rule", sourceLine: sourceLine, sourceLines: 1,
			}); err != nil {
				return nil, err
			}
			index++
			continue
		}
		if marker, text, ok := markdownListLine(line); ok {
			start := index
			listLines := []string{text}
			index++
			for index < len(lines) {
				nextMarker, nextText, next := markdownListLine(lines[index])
				if !next || orderedListMarker(nextMarker) !=
					orderedListMarker(marker) {
					break
				}
				listLines = append(
					listLines,
					nextMarker+"\x00"+nextText,
				)
				index++
			}
			if err := appendBlock(markdownBlock{
				kind: "list", sourceLine: sourceLine,
				sourceLines: index - start, lines: listLines, marker: marker,
			}); err != nil {
				return nil, err
			}
			continue
		}
		if text, ok := markdownQuoteLine(line); ok {
			start := index
			quoted := []string{text}
			index++
			for index < len(lines) {
				next, matches := markdownQuoteLine(lines[index])
				if !matches {
					break
				}
				quoted = append(quoted, next)
				index++
			}
			if err := appendBlock(markdownBlock{
				kind: "quote", sourceLine: sourceLine,
				sourceLines: index - start, lines: quoted,
			}); err != nil {
				return nil, err
			}
			continue
		}
		start := index
		paragraph := []string{strings.TrimSpace(line)}
		index++
		for index < len(lines) &&
			!markdownStartsBlock(lines[index]) {
			paragraph = append(paragraph, strings.TrimSpace(lines[index]))
			index++
		}
		if err := appendBlock(markdownBlock{
			kind: "paragraph", sourceLine: sourceLine,
			sourceLines: index - start,
			lines:       []string{strings.Join(paragraph, " ")},
		}); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

func markdownFenceMarker(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 {
		return "", false
	}
	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return "", false
	}
	count := 0
	for count < len(trimmed) && trimmed[count] == marker {
		count++
	}
	if count < 3 {
		return "", false
	}
	return trimmed[:count], true
}

func markdownFenceCloses(line, marker string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < len(marker) || trimmed[0] != marker[0] {
		return false
	}
	for index := range trimmed {
		if trimmed[index] != marker[0] {
			return false
		}
	}
	return len(trimmed) >= len(marker)
}

func markdownHeading(line string) (int, string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return 0, "", false
	}
	level := 0
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	text := strings.TrimSpace(trimmed[level:])
	if firstClosing := strings.LastIndex(text, " #"); firstClosing >= 0 {
		closing := text[firstClosing+1:]
		if strings.Trim(closing, "#") == "" {
			text = strings.TrimSpace(text[:firstClosing])
		}
	}
	return level, strings.TrimSpace(text), true
}

func markdownRule(line string) bool {
	trimmed := strings.ReplaceAll(strings.TrimSpace(line), " ", "")
	if len(trimmed) < 3 {
		return false
	}
	marker := trimmed[0]
	if marker != '-' && marker != '_' && marker != '*' {
		return false
	}
	for index := range trimmed {
		if trimmed[index] != marker {
			return false
		}
	}
	return true
}

func markdownListLine(line string) (string, string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 2 {
		return "", "", false
	}
	if strings.ContainsRune("-*+", rune(trimmed[0])) &&
		trimmed[1] == ' ' {
		return trimmed[:1], strings.TrimSpace(trimmed[2:]), true
	}
	digits := 0
	for digits < len(trimmed) &&
		trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits+1 < len(trimmed) &&
		trimmed[digits] == '.' && trimmed[digits+1] == ' ' {
		return trimmed[:digits+1],
			strings.TrimSpace(trimmed[digits+2:]), true
	}
	return "", "", false
}

func orderedListMarker(marker string) bool {
	return len(marker) > 1 && marker[len(marker)-1] == '.'
}

func markdownQuoteLine(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" || trimmed[0] != '>' {
		return "", false
	}
	return strings.TrimSpace(trimmed[1:]), true
}

func markdownStartsBlock(line string) bool {
	if strings.TrimSpace(line) == "" {
		return true
	}
	if _, ok := markdownFenceMarker(line); ok {
		return true
	}
	if _, _, ok := markdownHeading(line); ok {
		return true
	}
	if markdownRule(line) {
		return true
	}
	if _, _, ok := markdownListLine(line); ok {
		return true
	}
	_, ok := markdownQuoteLine(line)
	return ok
}

func reflowMarkdown(
	behavior markdownBehavior,
	size Size,
) markdownBehavior {
	for range 4 {
		geometry := calculateScrollViewGeometry(size, behavior.scroll)
		rendered, rows, maximum := renderMarkdownBlocks(
			behavior.blocks,
			max(1, geometry.viewport.Width),
		)
		nextSize := Size{Width: maximum, Height: len(rows)}
		behavior.rendered = rendered
		behavior.rows = rows
		behavior.maxLine = maximum
		if nextSize == behavior.scroll.state.ContentSize {
			break
		}
		behavior.scroll.state.ContentSize = nextSize
	}
	geometry := calculateScrollViewGeometry(size, behavior.scroll)
	behavior.scroll.state = clampViewportState(
		behavior.scroll.state,
		geometry,
	)
	return behavior
}

func renderMarkdownBlocks(
	blocks []markdownBlock,
	width int,
) ([]markdownRenderedBlock, [][]markdownCell, int) {
	rendered := make([]markdownRenderedBlock, 0, len(blocks))
	all := make([][]markdownCell, 0, len(blocks))
	maximum := 0
	for _, block := range blocks {
		rows := renderMarkdownBlock(block, width)
		record := markdownRenderedBlock{
			details: MarkdownBlockDetails{
				Kind:          block.kind,
				Level:         block.level,
				SourceLine:    block.sourceLine,
				SourceLines:   block.sourceLines,
				RenderedStart: len(all),
				RenderedRows:  len(rows),
			},
			rows: rows,
		}
		rendered = append(rendered, record)
		for _, row := range rows {
			maximum = max(maximum, len(row))
			all = append(all, row)
		}
	}
	return rendered, all, maximum
}

func renderMarkdownBlock(block markdownBlock, width int) [][]markdownCell {
	switch block.kind {
	case "blank":
		return [][]markdownCell{{}}
	case "rule":
		row := make([]markdownCell, max(1, width))
		for index := range row {
			row[index] = markdownCell{"─", "markdown.rule"}
		}
		return [][]markdownCell{row}
	case "code":
		if len(block.lines) == 0 {
			return [][]markdownCell{{}}
		}
		rows := make([][]markdownCell, len(block.lines))
		for index, line := range block.lines {
			rows[index] = markdownLiteralCells(line, "markdown.code")
		}
		return rows
	case "heading":
		return wrapMarkdownCells(
			markdownInlineCells(block.lines[0], "markdown.heading"),
			width,
			nil,
		)
	case "paragraph":
		return wrapMarkdownCells(
			markdownInlineCells(block.lines[0], "markdown_view"),
			width,
			nil,
		)
	case "quote":
		rows := make([][]markdownCell, 0, len(block.lines))
		prefix := []markdownCell{
			{"│", "markdown.quote"}, {" ", "markdown.quote"},
		}
		for _, line := range block.lines {
			content := append(
				append([]markdownCell{}, prefix...),
				markdownInlineCells(line, "markdown.quote")...,
			)
			rows = append(rows, wrapMarkdownCells(content, width, prefix)...)
		}
		return rows
	case "list":
		rows := make([][]markdownCell, 0, len(block.lines))
		for index, encoded := range block.lines {
			marker := block.marker
			text := encoded
			if split := strings.IndexByte(encoded, 0); split >= 0 {
				marker, text = encoded[:split], encoded[split+1:]
			} else if index > 0 {
				marker = block.marker
			}
			if !orderedListMarker(marker) {
				marker = "•"
			}
			prefix := append(
				markdownLiteralCells(marker, "markdown.list_marker"),
				markdownCell{" ", "markdown.list_marker"},
			)
			content := append(
				append([]markdownCell{}, prefix...),
				markdownInlineCells(text, "markdown_view")...,
			)
			continuation := make([]markdownCell, len(prefix))
			for cell := range continuation {
				continuation[cell] = markdownCell{" ", "markdown_view"}
			}
			rows = append(
				rows,
				wrapMarkdownCells(content, width, continuation)...,
			)
		}
		return rows
	default:
		return [][]markdownCell{{}}
	}
}

func markdownLiteralCells(text string, style StyleID) []markdownCell {
	cells := display.Normalize(text)
	result := make([]markdownCell, len(cells))
	for index, cell := range cells {
		result[index] = markdownCell{grapheme: cell, style: style}
	}
	return result
}

func markdownInlineCells(text string, base StyleID) []markdownCell {
	result := make([]markdownCell, 0, len(text))
	linksPossible := true
	for len(text) > 0 {
		if text[0] == '`' {
			if end := strings.IndexByte(text[1:], '`'); end >= 0 {
				result = append(
					result,
					markdownLiteralCells(text[1:end+1], "markdown.code")...,
				)
				text = text[end+2:]
				continue
			}
		}
		if text[0] == '[' && linksPossible {
			closeLabel := strings.Index(text, "](")
			if closeLabel > 0 {
				target := text[closeLabel+2:]
				if closeTarget := strings.IndexByte(
					target,
					')',
				); closeTarget >= 0 {
					result = append(
						result,
						markdownLiteralCells(
							text[1:closeLabel],
							"markdown.link",
						)...,
					)
					result = append(
						result,
						markdownLiteralCells(
							" ("+target[:closeTarget]+")",
							"markdown.link",
						)...,
					)
					text = text[closeLabel+2+closeTarget+1:]
					continue
				}
				// The earliest candidate has no closing parenthesis. No
				// later candidate in this remainder can have one either.
				linksPossible = false
			} else if closeLabel < 0 {
				// Remember this result so a run of literal '[' cells remains
				// linear instead of rescanning the complete suffix per cell.
				linksPossible = false
			}
		}
		if strings.HasPrefix(text, "**") ||
			strings.HasPrefix(text, "__") {
			delimiter := text[:2]
			if end := strings.Index(text[2:], delimiter); end >= 0 {
				result = append(
					result,
					markdownLiteralCells(
						text[2:end+2],
						"markdown.strong",
					)...,
				)
				text = text[end+4:]
				continue
			}
		}
		if text[0] == '*' || text[0] == '_' {
			if end := strings.IndexByte(text[1:], text[0]); end >= 0 {
				result = append(
					result,
					markdownLiteralCells(
						text[1:end+1],
						"markdown.emphasis",
					)...,
				)
				text = text[end+2:]
				continue
			}
		}
		next := len(text)
		for index := 1; index < len(text); index++ {
			switch text[index] {
			case '`', '*', '_':
				next = index
			case '[':
				if linksPossible {
					next = index
				}
			}
			if next != len(text) {
				break
			}
		}
		if next == len(text) {
			result = append(result, markdownLiteralCells(text, base)...)
			break
		}
		result = append(result, markdownLiteralCells(text[:next], base)...)
		text = text[next:]
	}
	return result
}

func wrapMarkdownCells(
	cells []markdownCell,
	width int,
	continuation []markdownCell,
) [][]markdownCell {
	if width < 1 {
		return [][]markdownCell{{}}
	}
	if len(cells) == 0 {
		return [][]markdownCell{{}}
	}
	if len(continuation) >= width {
		continuation = continuation[:max(0, width-1)]
	}
	rows := make([][]markdownCell, 0, 1+len(cells)/width)
	first := true
	for len(cells) > 0 {
		prefix := []markdownCell(nil)
		if !first {
			prefix = continuation
		}
		available := width - len(prefix)
		if len(cells) <= available {
			row := append([]markdownCell{}, prefix...)
			row = append(row, cells...)
			return append(rows, row)
		}
		breakAt := available
		minimumBreak := 0
		if first {
			minimumBreak = len(continuation)
		}
		for index := available - 1; index > minimumBreak; index-- {
			if cells[index].grapheme == " " {
				breakAt = index
				break
			}
		}
		row := append([]markdownCell{}, prefix...)
		row = append(row, cells[:breakAt]...)
		rows = append(rows, row)
		cells = cells[breakAt:]
		for len(cells) > 0 && cells[0].grapheme == " " {
			cells = cells[1:]
		}
		first = false
	}
	return rows
}

func (b markdownBehavior) controlBorder() borderBehavior {
	return b.scroll.border
}

func (b markdownBehavior) clientInset() int {
	return b.scroll.clientInset()
}

func (b markdownBehavior) controlClientRect(bounds Rect) Rect {
	return b.scroll.controlClientRect(bounds)
}

func (b markdownBehavior) intrinsicMinimum() Size {
	return b.scroll.intrinsicMinimum()
}

func (b markdownBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	return append(styles,
		"markdown.heading",
		"markdown.emphasis",
		"markdown.strong",
		"markdown.code",
		"markdown.link",
		"markdown.quote",
		"markdown.list_marker",
		"markdown.rule",
	)
}

func (b markdownBehavior) details() ControlDetails {
	details := b.scroll.border.details()
	details.Container = nil
	details.Markdown = &MarkdownDetails{}
	return details
}

func (b markdownBehavior) paintDecoration(
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
		if sourceY < 0 || sourceY >= len(b.rows) {
			continue
		}
		row := b.rows[sourceY]
		for x := visible.X; x < visible.X+visible.Width; x++ {
			sourceX := b.scroll.state.Offset.X + x - viewport.X
			if sourceX < 0 || sourceX >= len(row) {
				continue
			}
			cell := row[sourceX]
			resolved := app.styles[cell.style]
			app.setCellLocked(
				frame,
				x,
				y,
				cell.grapheme,
				cell.style,
				resolved,
				state.id,
			)
		}
	}
}

func markdownDetails(
	bounds Rect,
	behavior markdownBehavior,
) MarkdownDetails {
	summaries := make([]MarkdownBlockDetails, 0, min(
		len(behavior.rendered),
		MaxMarkdownSummaries,
	))
	if len(behavior.rendered) <= MaxMarkdownSummaries {
		for _, block := range behavior.rendered {
			summaries = append(summaries, block.details)
		}
	} else {
		leading := MaxMarkdownSummaries / 2
		trailing := MaxMarkdownSummaries - leading
		for _, block := range behavior.rendered[:leading] {
			summaries = append(summaries, block.details)
		}
		for _, block := range behavior.rendered[len(behavior.rendered)-trailing:] {
			summaries = append(summaries, block.details)
		}
	}
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	return MarkdownDetails{
		SourceBytes:        len(behavior.source),
		SourceCells:        behavior.sourceCells,
		BlockCount:         len(behavior.blocks),
		RenderedRows:       len(behavior.rows),
		MaximumLineWidth:   behavior.maxLine,
		Viewport:           viewport,
		Blocks:             summaries,
		SummariesTruncated: len(summaries) < len(behavior.rendered),
	}
}

func reconcileMarkdownLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(markdownBehavior)
	if !ok {
		return false
	}
	next := reflowMarkdown(behavior, state.bounds.Size())
	if markdownBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func markdownBehaviorEqual(left, right markdownBehavior) bool {
	if left.source != right.source ||
		left.sourceCells != right.sourceCells ||
		!scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.maxLine != right.maxLine ||
		len(left.blocks) != len(right.blocks) ||
		len(left.rows) != len(right.rows) ||
		len(left.rendered) != len(right.rendered) {
		return false
	}
	for index := range left.blocks {
		if left.blocks[index].kind != right.blocks[index].kind ||
			left.blocks[index].level != right.blocks[index].level ||
			left.blocks[index].sourceLine != right.blocks[index].sourceLine ||
			left.blocks[index].sourceLines != right.blocks[index].sourceLines ||
			left.blocks[index].marker != right.blocks[index].marker ||
			!stringSlicesEqual(
				left.blocks[index].lines,
				right.blocks[index].lines,
			) {
			return false
		}
	}
	for row := range left.rows {
		if len(left.rows[row]) != len(right.rows[row]) {
			return false
		}
		for cell := range left.rows[row] {
			if left.rows[row][cell] != right.rows[row][cell] {
				return false
			}
		}
	}
	return true
}

func stringSlicesEqual(left, right []string) bool {
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

// Markdown returns an independent canonical LF-normalized source copy.
func (v *MarkdownView) Markdown() string {
	if v == nil || v.state == nil {
		return ""
	}
	v.state.app.mu.RLock()
	defer v.state.app.mu.RUnlock()
	behavior, ok := v.state.behavior.(markdownBehavior)
	if !ok || v.state.destroyed || v.state.aborted {
		return ""
	}
	return behavior.source
}

// SetMarkdown replaces the source atomically.
func (v *MarkdownView) SetMarkdown(markdown string) error {
	return v.Update(context.Background(), markdown)
}

// Update replaces the source while observing ctx.
func (v *MarkdownView) Update(ctx context.Context, markdown string) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	transaction := v.state.app.NewTransaction()
	if err := transaction.SetMarkdown(v, markdown); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// Offset returns the current clamped scroll offset.
func (v *MarkdownView) Offset() Point {
	if v == nil || v.state == nil {
		return Point{}
	}
	v.state.app.mu.RLock()
	defer v.state.app.mu.RUnlock()
	behavior, ok := v.state.behavior.(markdownBehavior)
	if !ok || v.state.destroyed || v.state.aborted {
		return Point{}
	}
	return behavior.scroll.state.Offset
}

// SetOffset applies one requested scroll offset.
func (v *MarkdownView) SetOffset(offset Point) error {
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	transaction := v.state.app.NewTransaction()
	if err := transaction.SetMarkdownOffset(v, offset); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus gives this eligible MarkdownView keyboard focus.
func (v *MarkdownView) Focus() error { return focusSelectionControl(v) }

// SetMarkdown records one copied source replacement.
func (t *Transaction) SetMarkdown(
	control *MarkdownView,
	markdown string,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(markdownBehavior)
	if !ok {
		return ErrInvalidControl
	}
	source, cells, err := normalizeMarkdownSource(markdown)
	if err != nil {
		return err
	}
	blocks, err := parseMarkdown(source)
	if err != nil {
		return err
	}
	behavior.source = source
	behavior.sourceCells = cells
	behavior.blocks = blocks
	behavior = reflowMarkdown(behavior, target.bounds.Size())
	return t.recordMarkdownBehavior(target, behavior)
}

// SetMarkdownOffset records one clamped Markdown scroll movement.
func (t *Transaction) SetMarkdownOffset(
	control *MarkdownView,
	offset Point,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(markdownBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if offset.X < 0 || offset.Y < 0 ||
		offset.X > maxCoordinateMagnitude ||
		offset.Y > maxCoordinateMagnitude {
		return fmt.Errorf("%w: invalid MarkdownView offset", ErrValidation)
	}
	geometry := calculateScrollViewGeometry(
		target.bounds.Size(),
		behavior.scroll,
	)
	behavior.scroll.state.Offset = Point{
		X: min(offset.X, geometry.maximumOffset.X),
		Y: min(offset.Y, geometry.maximumOffset.Y),
	}
	return t.recordMarkdownBehavior(target, behavior)
}

func (t *Transaction) recordMarkdownBehavior(
	state *controlState,
	behavior markdownBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationMarkdownView, state: state, behavior: behavior,
	})
	return nil
}

func markdownCanMove(state *controlState, behavior markdownBehavior) bool {
	return scrollViewCanMove(state, behavior.scroll)
}

func (a *App) markdownKeyLocked(
	control *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if control == nil {
		return "", "", false, false
	}
	behavior, ok := control.behavior.(markdownBehavior)
	if !ok {
		return "", "", false, false
	}
	holder := &controlState{
		id:       control.id,
		bounds:   control.bounds,
		behavior: behavior.scroll,
	}
	command, target, handled, changed := a.scrollViewKeyLocked(holder, key)
	if !changed {
		return command, target, handled, false
	}
	behavior.scroll = holder.behavior.(scrollViewBehavior)
	control.behavior = behavior
	return command, target, true, true
}
