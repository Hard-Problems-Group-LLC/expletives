package expletives

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
	"github.com/rivo/uniseg"
)

// LogLevel identifies one recognized log-record severity.
type LogLevel string

const (
	LogDebug   LogLevel = "debug"
	LogInfo    LogLevel = "info"
	LogWarning LogLevel = "warning"
	LogError   LogLevel = "error"
)

// LogRecord is one copied, stable, bounded application record.
type LogRecord struct {
	Key       string
	Timestamp string
	Level     LogLevel
	Text      string
}

// ContentCapacity bounds retained complete records and canonical content
// bytes. Zero components select their named defaults.
type ContentCapacity struct {
	Records int `json:"records"`
	Bytes   int `json:"bytes"`
}

// LogViewOptions configures one bounded structured-log leaf.
type LogViewOptions struct {
	ScrollablePanelOptions
	Capacity ContentCapacity
	Records  []LogRecord
	Follow   bool
}

// StreamViewOptions configures one bounded line-oriented byte-stream leaf.
type StreamViewOptions struct {
	ScrollablePanelOptions
	Capacity ContentCapacity
	Follow   bool
}

// LogViewState is copied public retention and scrollback state. For
// StreamView, RetainedRecords and DroppedRecords count completed lines.
type LogViewState struct {
	Follow          bool
	Offset          Point
	RetainedRecords int
	RetainedBytes   int
	PendingBytes    int
	DroppedRecords  uint64
	DroppedBytes    uint64
}

// StreamAppendResult describes the effect of one committed Append call.
type StreamAppendResult struct {
	InputBytes     int
	CompletedLines int
	DroppedLines   uint64
	DroppedBytes   uint64
}

// LogView is a copy-safe, read-only structured-log control.
type LogView struct{ controlHandle }

// StreamView is a copy-safe, read-only line-oriented byte-stream control.
type StreamView struct{ controlHandle }

type retainedLogRecord struct {
	key          string
	timestamp    []markdownCell
	level        LogLevel
	lines        [][]markdownCell
	storageBytes int
	sourceBytes  int
	lossCounted  bool
}

type logViewBehavior struct {
	scroll             scrollViewBehavior
	capacity           ContentCapacity
	records            []retainedLogRecord
	follow             bool
	droppedRecords     uint64
	droppedBytes       uint64
	rows               [][]markdownCell
	rowKeys            []string
	rowWithinRecord    []int
	maxLine            int
	stream             bool
	pending            []byte
	pendingCells       []markdownCell
	pendingTruncated   bool
	pendingLossCounted bool
	skipNextLF         bool
	nextLine           uint64
}

type logAnchor struct {
	key    string
	within int
	valid  bool
}

type streamElement struct {
	cell        string
	sourceBytes int
}

// NewLogView constructs and atomically inserts a LogView.
func NewLogView(parent Container, options LogViewOptions) (*LogView, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewLogView(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewStreamView constructs and atomically inserts a StreamView.
func NewStreamView(
	parent Container,
	options StreamViewOptions,
) (*StreamView, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewStreamView(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewLogView records construction of a provisional LogView.
func (t *Transaction) NewLogView(
	parent Container,
	options LogViewOptions,
) (*LogView, error) {
	behavior, err := newLogViewBehavior(
		options.ScrollablePanelOptions,
		options.Capacity,
		options.Follow,
		false,
		ControlLogView,
	)
	if err != nil {
		return nil, err
	}
	records, err := normalizeLogRecords(options.Records)
	if err != nil {
		return nil, err
	}
	behavior, err = appendRetainedRecords(behavior, records)
	if err != nil {
		return nil, err
	}
	behavior = reflowLogView(behavior, options.Bounds.Size())
	if behavior.follow {
		behavior.scroll.state.Offset.Y = logMaximumOffset(
			options.Bounds.Size(),
			behavior,
		).Y
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlLogView,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &LogView{controlHandle: controlHandle{state: state}}
	state.control = control
	state.contentGate = make(chan struct{}, 1)
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

// NewStreamView records construction of a provisional StreamView.
func (t *Transaction) NewStreamView(
	parent Container,
	options StreamViewOptions,
) (*StreamView, error) {
	behavior, err := newLogViewBehavior(
		options.ScrollablePanelOptions,
		options.Capacity,
		options.Follow,
		true,
		ControlStreamView,
	)
	if err != nil {
		return nil, err
	}
	behavior = reflowLogView(behavior, options.Bounds.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlStreamView,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &StreamView{controlHandle: controlHandle{state: state}}
	state.control = control
	state.contentGate = make(chan struct{}, 1)
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func newLogViewBehavior(
	options ScrollablePanelOptions,
	capacity ContentCapacity,
	follow bool,
	stream bool,
	kind ControlKind,
) (logViewBehavior, error) {
	scrollOptions := options.ScrollViewOptions
	if scrollOptions.ContentAutomationKey != "" ||
		scrollOptions.ContentStyle != "" ||
		scrollOptions.State.ContentSize != (Size{}) {
		return logViewBehavior{}, fmt.Errorf(
			"%w: %s derives its content identity, style, and extent",
			ErrValidation,
			kind,
		)
	}
	requestedOffset := scrollOptions.State.Offset
	if requestedOffset.X < 0 || requestedOffset.Y < 0 ||
		requestedOffset.X > maxCoordinateMagnitude ||
		requestedOffset.Y > maxCoordinateMagnitude {
		return logViewBehavior{}, fmt.Errorf(
			"%w: invalid %s offset",
			ErrValidation,
			kind,
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
		return logViewBehavior{}, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		kind,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return logViewBehavior{}, err
	}
	scroll.border = border
	scroll.state.Offset = requestedOffset
	capacity, err = normalizeContentCapacity(capacity)
	if err != nil {
		return logViewBehavior{}, err
	}
	return logViewBehavior{
		scroll: scroll, capacity: capacity, follow: follow, stream: stream,
	}, nil
}

func normalizeContentCapacity(
	capacity ContentCapacity,
) (ContentCapacity, error) {
	if capacity.Records < 0 || capacity.Bytes < 0 {
		return ContentCapacity{}, fmt.Errorf(
			"%w: negative content capacity",
			ErrValidation,
		)
	}
	if capacity.Records == 0 {
		capacity.Records = DefaultContentRecordCapacity
	}
	if capacity.Bytes == 0 {
		capacity.Bytes = DefaultContentByteCapacity
	}
	if capacity.Records > MaxContentRecords ||
		capacity.Bytes > MaxContentBytes {
		return ContentCapacity{}, fmt.Errorf(
			"%w: content capacity exceeds records=%d bytes=%d",
			ErrControlCapacity,
			MaxContentRecords,
			MaxContentBytes,
		)
	}
	return capacity, nil
}

func validLogLevel(level LogLevel) bool {
	switch level {
	case LogDebug, LogInfo, LogWarning, LogError:
		return true
	default:
		return false
	}
}

func normalizeLogRecords(
	records []LogRecord,
) ([]retainedLogRecord, error) {
	if len(records) > MaxContentRecords {
		return nil, fmt.Errorf(
			"%w: log update exceeds %d records",
			ErrControlCapacity,
			MaxContentRecords,
		)
	}
	result := make([]retainedLogRecord, 0, len(records))
	inputBytes := 0
	for _, record := range records {
		inputBytes += len(record.Key) + len(record.Timestamp) +
			len(record.Level) + len(record.Text)
		if inputBytes > MaxContentBytes {
			return nil, fmt.Errorf(
				"%w: log update exceeds %d input bytes",
				ErrTextLimit,
				MaxContentBytes,
			)
		}
		if !validBoundedIdentifier(record.Key) || !validLogLevel(record.Level) {
			return nil, fmt.Errorf(
				"%w: invalid log key or level",
				ErrValidation,
			)
		}
		timestamp, err := normalizeDisplayText(record.Timestamp, false)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid log timestamp", err)
		}
		lines, canonical, err := normalizeLogText(record.Text)
		if err != nil {
			return nil, err
		}
		timestampCells := make([]markdownCell, len(timestamp.lines[0]))
		for index, cell := range timestamp.lines[0] {
			timestampCells[index] = markdownCell{
				grapheme: cell,
				style:    "log.timestamp",
			}
		}
		storageBytes := len(record.Key) + len(record.Level) +
			len(timestamp.text) + len(canonical)
		result = append(result, retainedLogRecord{
			key:          record.Key,
			timestamp:    timestampCells,
			level:        record.Level,
			lines:        lines,
			storageBytes: storageBytes,
			sourceBytes:  storageBytes,
		})
	}
	return result, nil
}

func normalizeLogText(
	text string,
) ([][]markdownCell, string, error) {
	if len(text) > MaxContentBytes {
		return nil, "", fmt.Errorf(
			"%w: log record exceeds %d bytes",
			ErrTextLimit,
			MaxContentBytes,
		)
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	parts := strings.Split(text, "\n")
	lines := make([][]markdownCell, len(parts))
	canonicalParts := make([]string, len(parts))
	for lineIndex, part := range parts {
		cells := display.Normalize(part)
		line := make([]markdownCell, len(cells))
		for index, cell := range cells {
			line[index] = markdownCell{grapheme: cell, style: "log_view"}
		}
		lines[lineIndex] = line
		canonicalParts[lineIndex] = strings.Join(cells, "")
	}
	canonical := strings.Join(canonicalParts, "\n")
	if !utf8.ValidString(canonical) || len(canonical) > MaxContentBytes {
		return nil, "", fmt.Errorf(
			"%w: normalized log record exceeds %d bytes",
			ErrTextLimit,
			MaxContentBytes,
		)
	}
	return lines, canonical, nil
}

func appendRetainedRecords(
	behavior logViewBehavior,
	records []retainedLogRecord,
) (logViewBehavior, error) {
	behavior = cloneLogViewBehavior(behavior)
	behavior.records = append(behavior.records, records...)
	for logRetainedStorage(behavior.records) > behavior.capacity.Bytes ||
		len(behavior.records) > behavior.capacity.Records {
		oldest := behavior.records[0]
		behavior.records = behavior.records[1:]
		var err error
		behavior.droppedRecords, behavior.droppedBytes, err = addContentDrop(
			behavior.droppedRecords,
			behavior.droppedBytes,
			oldest,
		)
		if err != nil {
			return logViewBehavior{}, err
		}
	}
	seen := make(map[string]bool, len(behavior.records))
	for _, record := range behavior.records {
		if seen[record.key] {
			return logViewBehavior{}, fmt.Errorf(
				"%w: duplicate retained log key %q",
				ErrValidation,
				record.key,
			)
		}
		seen[record.key] = true
	}
	return behavior, nil
}

func addContentDrop(
	records uint64,
	bytes uint64,
	record retainedLogRecord,
) (uint64, uint64, error) {
	if !record.lossCounted {
		if records == math.MaxUint64 {
			return 0, 0, ErrControlCapacity
		}
		records++
	}
	if uint64(record.sourceBytes) > math.MaxUint64-bytes {
		return 0, 0, ErrControlCapacity
	}
	return records, bytes + uint64(record.sourceBytes), nil
}

func logRetainedStorage(records []retainedLogRecord) int {
	total := 0
	for _, record := range records {
		total += record.storageBytes
	}
	return total
}

func (b logViewBehavior) controlBorder() borderBehavior {
	return b.scroll.border
}

func (b logViewBehavior) clientInset() int { return b.scroll.clientInset() }

func (b logViewBehavior) controlClientRect(bounds Rect) Rect {
	return b.scroll.controlClientRect(bounds)
}

func (b logViewBehavior) intrinsicMinimum() Size {
	return b.scroll.intrinsicMinimum()
}

func (b logViewBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	if b.stream {
		return append(styles, "stream.truncated", "content.dropped")
	}
	return append(styles,
		"log.timestamp",
		"log.debug",
		"log.info",
		"log.warning",
		"log.error",
		"content.dropped",
	)
}

func (b logViewBehavior) details() ControlDetails {
	details := b.scroll.border.details()
	details.Container = nil
	if b.stream {
		details.StreamView = &StreamViewDetails{}
	} else {
		details.LogView = &LogViewDetails{}
	}
	return details
}

func (b logViewBehavior) paintDecoration(
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

func reflowLogView(behavior logViewBehavior, size Size) logViewBehavior {
	rows := make([][]markdownCell, 0, len(behavior.records)+1)
	keys := make([]string, 0, len(behavior.records)+1)
	within := make([]int, 0, len(behavior.records)+1)
	maximum := 0
	appendRow := func(row []markdownCell, key string, rowWithin int) {
		rows = append(rows, row)
		keys = append(keys, key)
		within = append(within, rowWithin)
		maximum = max(maximum, len(row))
	}
	if behavior.droppedRecords > 0 || behavior.droppedBytes > 0 {
		unit := "records"
		if behavior.stream {
			unit = "lines"
		}
		text := fmt.Sprintf(
			"[dropped %d %s / %d bytes]",
			behavior.droppedRecords,
			unit,
			behavior.droppedBytes,
		)
		appendRow(markdownLiteralCells(text, "content.dropped"), "", 0)
	}
	for _, record := range behavior.records {
		if behavior.stream {
			row := cloneMarkdownCells(record.lines[0])
			appendRow(row, record.key, 0)
			continue
		}
		prefix := logRecordPrefix(record)
		continuation := make([]markdownCell, len(prefix))
		for index := range continuation {
			continuation[index] = markdownCell{
				grapheme: " ", style: "log_view",
			}
		}
		for lineIndex, line := range record.lines {
			row := cloneMarkdownCells(continuation)
			if lineIndex == 0 {
				row = cloneMarkdownCells(prefix)
			}
			row = append(row, styledLogLine(line, record.level)...)
			appendRow(row, record.key, lineIndex)
		}
	}
	if behavior.stream &&
		(len(behavior.pendingCells) > 0 || behavior.pendingTruncated) {
		appendRow(
			cloneMarkdownCells(behavior.pendingCells),
			"stream.pending",
			0,
		)
	}
	behavior.rows = rows
	behavior.rowKeys = keys
	behavior.rowWithinRecord = within
	behavior.maxLine = maximum
	behavior.scroll.state.ContentSize = Size{Width: maximum, Height: len(rows)}
	geometry := calculateScrollViewGeometry(size, behavior.scroll)
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	if behavior.follow {
		behavior.scroll.state.Offset.Y = geometry.maximumOffset.Y
	}
	return behavior
}

func logRecordPrefix(record retainedLogRecord) []markdownCell {
	prefix := cloneMarkdownCells(record.timestamp)
	if len(prefix) > 0 {
		prefix = append(prefix, markdownCell{" ", "log.timestamp"})
	}
	level := strings.ToUpper(string(record.level))
	return append(prefix, markdownLiteralCells("["+level+"] ", logLevelStyle(record.level))...)
}

func logLevelStyle(level LogLevel) StyleID {
	return StyleID("log." + string(level))
}

func styledLogLine(line []markdownCell, level LogLevel) []markdownCell {
	result := cloneMarkdownCells(line)
	style := logLevelStyle(level)
	for index := range result {
		result[index].style = style
	}
	return result
}

func cloneMarkdownCells(cells []markdownCell) []markdownCell {
	return append([]markdownCell(nil), cells...)
}

func logMaximumOffset(size Size, behavior logViewBehavior) Point {
	return calculateScrollViewGeometry(size, behavior.scroll).maximumOffset
}

func captureLogAnchor(behavior logViewBehavior) logAnchor {
	row := behavior.scroll.state.Offset.Y
	if row < 0 || row >= len(behavior.rowKeys) {
		return logAnchor{}
	}
	if behavior.rowKeys[row] == "" {
		row++
		if row >= len(behavior.rowKeys) {
			return logAnchor{}
		}
	}
	return logAnchor{
		key: behavior.rowKeys[row], within: behavior.rowWithinRecord[row], valid: true,
	}
}

func restoreLogAnchor(
	behavior *logViewBehavior,
	anchor logAnchor,
	size Size,
) {
	if behavior == nil || !anchor.valid {
		return
	}
	for index, key := range behavior.rowKeys {
		if key == anchor.key && behavior.rowWithinRecord[index] == anchor.within {
			behavior.scroll.state.Offset.Y = index
			geometry := calculateScrollViewGeometry(size, behavior.scroll)
			behavior.scroll.state.Offset.Y = min(
				behavior.scroll.state.Offset.Y,
				geometry.maximumOffset.Y,
			)
			return
		}
	}
	behavior.scroll.state.Offset.Y = 0
}

func cloneRetainedRecord(record retainedLogRecord) retainedLogRecord {
	cloned := record
	cloned.timestamp = cloneMarkdownCells(record.timestamp)
	cloned.lines = make([][]markdownCell, len(record.lines))
	for index := range record.lines {
		cloned.lines[index] = cloneMarkdownCells(record.lines[index])
	}
	return cloned
}

func cloneLogViewBehavior(behavior logViewBehavior) logViewBehavior {
	cloned := behavior
	cloned.records = make([]retainedLogRecord, len(behavior.records))
	for index, record := range behavior.records {
		cloned.records[index] = cloneRetainedRecord(record)
	}
	cloned.rows = make([][]markdownCell, len(behavior.rows))
	for index := range behavior.rows {
		cloned.rows[index] = cloneMarkdownCells(behavior.rows[index])
	}
	cloned.rowKeys = append([]string(nil), behavior.rowKeys...)
	cloned.rowWithinRecord = append([]int(nil), behavior.rowWithinRecord...)
	cloned.pending = append([]byte(nil), behavior.pending...)
	cloned.pendingCells = cloneMarkdownCells(behavior.pendingCells)
	return cloned
}

func logViewBehaviorEqual(left, right logViewBehavior) bool {
	if !scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.capacity != right.capacity || left.follow != right.follow ||
		left.droppedRecords != right.droppedRecords ||
		left.droppedBytes != right.droppedBytes ||
		left.maxLine != right.maxLine || left.stream != right.stream ||
		left.pendingTruncated != right.pendingTruncated ||
		left.pendingLossCounted != right.pendingLossCounted ||
		left.skipNextLF != right.skipNextLF ||
		left.nextLine != right.nextLine ||
		!bytesEqual(left.pending, right.pending) ||
		len(left.records) != len(right.records) ||
		len(left.rows) != len(right.rows) {
		return false
	}
	for index := range left.records {
		if !retainedLogRecordEqual(left.records[index], right.records[index]) {
			return false
		}
	}
	for index := range left.rows {
		if !markdownCellSlicesEqual(left.rows[index], right.rows[index]) {
			return false
		}
	}
	return true
}

func bytesEqual(left, right []byte) bool {
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

func retainedLogRecordEqual(left, right retainedLogRecord) bool {
	if left.key != right.key || left.level != right.level ||
		left.storageBytes != right.storageBytes ||
		left.sourceBytes != right.sourceBytes ||
		left.lossCounted != right.lossCounted ||
		!markdownCellSlicesEqual(left.timestamp, right.timestamp) ||
		len(left.lines) != len(right.lines) {
		return false
	}
	for index := range left.lines {
		if !markdownCellSlicesEqual(left.lines[index], right.lines[index]) {
			return false
		}
	}
	return true
}

func markdownCellSlicesEqual(left, right []markdownCell) bool {
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

func logViewState(behavior logViewBehavior) LogViewState {
	return LogViewState{
		Follow:          behavior.follow,
		Offset:          behavior.scroll.state.Offset,
		RetainedRecords: len(behavior.records),
		RetainedBytes:   logRetainedStorage(behavior.records),
		PendingBytes:    len(behavior.pending),
		DroppedRecords:  behavior.droppedRecords,
		DroppedBytes:    behavior.droppedBytes,
	}
}

// State returns a copied LogView state.
func (v *LogView) State() LogViewState { return stateForLogControl(v) }

// State returns a copied StreamView state.
func (v *StreamView) State() LogViewState { return stateForLogControl(v) }

func stateForLogControl(control Control) LogViewState {
	if control == nil || control.controlState() == nil {
		return LogViewState{}
	}
	state := control.controlState()
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(logViewBehavior)
	if !ok || state.destroyed || state.aborted {
		return LogViewState{}
	}
	return logViewState(behavior)
}

func lockControlContent(ctx context.Context, state *controlState) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if state == nil || state.contentGate == nil {
		return ErrInvalidControl
	}
	select {
	case state.contentGate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func unlockControlContent(state *controlState) {
	if state != nil && state.contentGate != nil {
		<-state.contentGate
	}
}

// Append validates, copies, and appends structured records atomically.
func (v *LogView) Append(ctx context.Context, records []LogRecord) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	if err := transaction.AppendLog(v, records); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// Replace atomically replaces retained records and resets drop counters.
func (v *LogView) Replace(ctx context.Context, records []LogRecord) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	if err := transaction.ReplaceLog(v, records); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// Clear removes retained LogView records and resets drop counters.
func (v *LogView) Clear(ctx context.Context) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	if err := transaction.ClearLog(v); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// SetFollow changes LogView tail-follow state without routing ChangeCommand.
func (v *LogView) SetFollow(follow bool) error {
	return setLogControlFollow(v, follow)
}

// SetFollow changes StreamView tail-follow state without routing
// ChangeCommand.
func (v *StreamView) SetFollow(follow bool) error {
	return setLogControlFollow(v, follow)
}

func setLogControlFollow(control Control, follow bool) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	if err := lockControlContent(context.Background(), state); err != nil {
		return err
	}
	defer unlockControlContent(state)
	transaction := state.app.NewTransaction()
	if err := transaction.SetLogFollow(control, follow); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetOffset applies one clamped LogView scroll offset.
func (v *LogView) SetOffset(offset Point) error {
	return setLogControlOffset(v, offset)
}

// SetOffset applies one clamped StreamView scroll offset.
func (v *StreamView) SetOffset(offset Point) error {
	return setLogControlOffset(v, offset)
}

func setLogControlOffset(control Control, offset Point) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	if err := lockControlContent(context.Background(), state); err != nil {
		return err
	}
	defer unlockControlContent(state)
	transaction := state.app.NewTransaction()
	if err := transaction.SetLogOffset(control, offset); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus gives this eligible LogView keyboard focus.
func (v *LogView) Focus() error { return focusSelectionControl(v) }

// Focus gives this eligible StreamView keyboard focus.
func (v *StreamView) Focus() error { return focusSelectionControl(v) }

// AppendLog records one copied structured-log append.
func (t *Transaction) AppendLog(
	control *LogView,
	records []LogRecord,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || behavior.stream {
		return ErrInvalidControl
	}
	normalized, err := normalizeLogRecords(records)
	if err != nil {
		return err
	}
	anchor := captureLogAnchor(behavior)
	behavior, err = appendRetainedRecords(behavior, normalized)
	if err != nil {
		return err
	}
	behavior = reflowLogView(behavior, target.bounds.Size())
	if !behavior.follow {
		restoreLogAnchor(&behavior, anchor, target.bounds.Size())
		behavior.scroll.state = clampViewportState(
			behavior.scroll.state,
			calculateScrollViewGeometry(target.bounds.Size(), behavior.scroll),
		)
	}
	return t.recordLogViewBehavior(target, behavior)
}

// ReplaceLog records one complete LogView replacement and counter reset.
func (t *Transaction) ReplaceLog(
	control *LogView,
	records []LogRecord,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || behavior.stream {
		return ErrInvalidControl
	}
	normalized, err := normalizeLogRecords(records)
	if err != nil {
		return err
	}
	behavior = cloneLogViewBehavior(behavior)
	behavior.records = nil
	behavior.droppedRecords = 0
	behavior.droppedBytes = 0
	behavior, err = appendRetainedRecords(behavior, normalized)
	if err != nil {
		return err
	}
	behavior = reflowLogView(behavior, target.bounds.Size())
	return t.recordLogViewBehavior(target, behavior)
}

// ClearLog records a complete LogView clear and counter reset.
func (t *Transaction) ClearLog(control *LogView) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || behavior.stream {
		return ErrInvalidControl
	}
	behavior = cloneLogViewBehavior(behavior)
	behavior.records = nil
	behavior.droppedRecords = 0
	behavior.droppedBytes = 0
	behavior.scroll.state.Offset = Point{}
	behavior = reflowLogView(behavior, target.bounds.Size())
	return t.recordLogViewBehavior(target, behavior)
}

// SetLogFollow records one LogView or StreamView follow-state change.
func (t *Transaction) SetLogFollow(control Control, follow bool) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok {
		return ErrInvalidControl
	}
	behavior = cloneLogViewBehavior(behavior)
	behavior.follow = follow
	behavior = reflowLogView(behavior, target.bounds.Size())
	return t.recordLogViewBehavior(target, behavior)
}

// SetLogOffset records one LogView or StreamView offset change.
func (t *Transaction) SetLogOffset(control Control, offset Point) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if offset.X < 0 || offset.Y < 0 ||
		offset.X > maxCoordinateMagnitude ||
		offset.Y > maxCoordinateMagnitude {
		return fmt.Errorf("%w: invalid content-view offset", ErrValidation)
	}
	behavior = cloneLogViewBehavior(behavior)
	geometry := calculateScrollViewGeometry(target.bounds.Size(), behavior.scroll)
	behavior.scroll.state.Offset = Point{
		X: min(offset.X, geometry.maximumOffset.X),
		Y: min(offset.Y, geometry.maximumOffset.Y),
	}
	if behavior.scroll.state.Offset.Y < geometry.maximumOffset.Y {
		behavior.follow = false
	}
	return t.recordLogViewBehavior(target, behavior)
}

func (t *Transaction) recordLogViewBehavior(
	state *controlState,
	behavior logViewBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationLogView, state: state, behavior: behavior,
	})
	return nil
}

// Append copies one bounded stream chunk and records its complete synchronous
// state transition.
func (v *StreamView) Append(
	ctx context.Context,
	chunk []byte,
) (StreamAppendResult, error) {
	if ctx == nil {
		return StreamAppendResult{}, errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return StreamAppendResult{}, ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return StreamAppendResult{}, err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	result, err := transaction.AppendStream(v, chunk)
	if err != nil {
		return StreamAppendResult{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return StreamAppendResult{}, err
	}
	return result, nil
}

// Flush commits one retained partial stream line, if present.
func (v *StreamView) Flush(ctx context.Context) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	if err := transaction.FlushStream(v); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// Clear removes complete and partial stream content and resets drop counters.
func (v *StreamView) Clear(ctx context.Context) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if v == nil || v.state == nil {
		return ErrInvalidControl
	}
	if err := lockControlContent(ctx, v.state); err != nil {
		return err
	}
	defer unlockControlContent(v.state)
	transaction := v.state.app.NewTransaction()
	if err := transaction.ClearStream(v); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// AppendStream records one bounded copied stream chunk and returns its
// provisional effect. The result becomes authoritative only if Commit
// succeeds.
func (t *Transaction) AppendStream(
	control *StreamView,
	chunk []byte,
) (StreamAppendResult, error) {
	if len(chunk) > MaxContentBytes {
		return StreamAppendResult{}, fmt.Errorf(
			"%w: stream append exceeds %d bytes",
			ErrTextLimit,
			MaxContentBytes,
		)
	}
	target, err := t.control(control)
	if err != nil {
		return StreamAppendResult{}, err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || !behavior.stream {
		return StreamAppendResult{}, ErrInvalidControl
	}
	behavior = cloneLogViewBehavior(behavior)
	anchor := captureLogAnchor(behavior)
	beforeRecords := behavior.droppedRecords
	beforeBytes := behavior.droppedBytes
	completed := 0
	for _, value := range append([]byte(nil), chunk...) {
		if behavior.skipNextLF {
			behavior.skipNextLF = false
			if value == '\n' {
				continue
			}
		}
		if value == '\r' || value == '\n' {
			if err := finalizeStreamPending(&behavior); err != nil {
				return StreamAppendResult{}, err
			}
			if err := commitStreamPending(&behavior); err != nil {
				return StreamAppendResult{}, err
			}
			completed++
			if value == '\r' {
				behavior.skipNextLF = true
			}
			continue
		}
		if behavior.pendingTruncated {
			if err := addPendingDrop(&behavior, 1); err != nil {
				return StreamAppendResult{}, err
			}
			continue
		}
		if len(behavior.pending) >= behavior.capacity.Bytes {
			if err := markPendingTruncated(&behavior); err != nil {
				return StreamAppendResult{}, err
			}
			if err := addPendingDrop(&behavior, 1); err != nil {
				return StreamAppendResult{}, err
			}
			continue
		}
		behavior.pending = append(behavior.pending, value)
	}
	if err := finalizeStreamPending(&behavior); err != nil {
		return StreamAppendResult{}, err
	}
	if err := enforceStreamCapacity(&behavior); err != nil {
		return StreamAppendResult{}, err
	}
	behavior = reflowLogView(behavior, target.bounds.Size())
	if !behavior.follow {
		restoreLogAnchor(&behavior, anchor, target.bounds.Size())
		behavior.scroll.state = clampViewportState(
			behavior.scroll.state,
			calculateScrollViewGeometry(target.bounds.Size(), behavior.scroll),
		)
	}
	if err := t.recordLogViewBehavior(target, behavior); err != nil {
		return StreamAppendResult{}, err
	}
	return StreamAppendResult{
		InputBytes:     len(chunk),
		CompletedLines: completed,
		DroppedLines:   behavior.droppedRecords - beforeRecords,
		DroppedBytes:   behavior.droppedBytes - beforeBytes,
	}, nil
}

// FlushStream records one partial-line commit.
func (t *Transaction) FlushStream(control *StreamView) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || !behavior.stream {
		return ErrInvalidControl
	}
	if len(behavior.pending) == 0 && !behavior.pendingTruncated {
		return nil
	}
	behavior = cloneLogViewBehavior(behavior)
	anchor := captureLogAnchor(behavior)
	if err := finalizeStreamPending(&behavior); err != nil {
		return err
	}
	if err := commitStreamPending(&behavior); err != nil {
		return err
	}
	if err := enforceStreamCapacity(&behavior); err != nil {
		return err
	}
	behavior = reflowLogView(behavior, target.bounds.Size())
	if !behavior.follow {
		restoreLogAnchor(&behavior, anchor, target.bounds.Size())
	}
	return t.recordLogViewBehavior(target, behavior)
}

// ClearStream records a complete stream clear and counter reset.
func (t *Transaction) ClearStream(control *StreamView) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(logViewBehavior)
	if !ok || !behavior.stream {
		return ErrInvalidControl
	}
	behavior = cloneLogViewBehavior(behavior)
	behavior.records = nil
	behavior.pending = nil
	behavior.pendingCells = nil
	behavior.pendingTruncated = false
	behavior.pendingLossCounted = false
	behavior.skipNextLF = false
	behavior.droppedRecords = 0
	behavior.droppedBytes = 0
	behavior.scroll.state.Offset = Point{}
	behavior = reflowLogView(behavior, target.bounds.Size())
	return t.recordLogViewBehavior(target, behavior)
}

func finalizeStreamPending(behavior *logViewBehavior) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	elements := streamElements(behavior.pending)
	marker := streamTruncationMarker(behavior.capacity.Bytes)
	markerBytes := 0
	if behavior.pendingTruncated {
		markerBytes = len(marker)
	}
	kept := 0
	storage := 0
	for kept < len(elements) {
		cellBytes := len(elements[kept].cell)
		if storage+cellBytes+markerBytes > behavior.capacity.Bytes {
			break
		}
		storage += cellBytes
		kept++
	}
	if kept < len(elements) {
		if err := markPendingTruncated(behavior); err != nil {
			return err
		}
		marker = streamTruncationMarker(behavior.capacity.Bytes)
		markerBytes = len(marker)
		for kept > 0 && storage+markerBytes > behavior.capacity.Bytes {
			kept--
			storage -= len(elements[kept].cell)
		}
		dropped := 0
		for _, element := range elements[kept:] {
			dropped += element.sourceBytes
		}
		if err := addPendingDrop(behavior, dropped); err != nil {
			return err
		}
		retained := 0
		for _, element := range elements[:kept] {
			retained += element.sourceBytes
		}
		behavior.pending = append([]byte(nil), behavior.pending[:retained]...)
		elements = elements[:kept]
	}
	cells := make([]markdownCell, 0, len(elements)+len(marker))
	for _, element := range elements {
		cells = append(cells, markdownCell{element.cell, "stream_view"})
	}
	if behavior.pendingTruncated {
		cells = append(cells, markdownLiteralCells(marker, "stream.truncated")...)
	}
	behavior.pendingCells = cells
	return nil
}

func streamElements(raw []byte) []streamElement {
	graphemes := uniseg.NewGraphemes(string(raw))
	result := make([]streamElement, 0, len(raw))
	for graphemes.Next() {
		cluster := graphemes.Str()
		cell := cluster
		if !utf8.ValidString(cluster) || graphemes.Width() != 1 {
			cell = display.Replacement
		}
		result = append(result, streamElement{
			cell: cell, sourceBytes: len(cluster),
		})
	}
	return result
}

func streamTruncationMarker(capacity int) string {
	for _, marker := range []string{"[truncated]", "!"} {
		if len(marker) <= capacity {
			return marker
		}
	}
	return ""
}

func markPendingTruncated(behavior *logViewBehavior) error {
	if behavior.pendingTruncated {
		return nil
	}
	behavior.pendingTruncated = true
	if behavior.pendingLossCounted {
		return nil
	}
	if behavior.droppedRecords == math.MaxUint64 {
		return ErrControlCapacity
	}
	behavior.droppedRecords++
	behavior.pendingLossCounted = true
	return nil
}

func addPendingDrop(behavior *logViewBehavior, dropped int) error {
	if dropped < 0 || uint64(dropped) > math.MaxUint64-behavior.droppedBytes {
		return ErrControlCapacity
	}
	behavior.droppedBytes += uint64(dropped)
	return nil
}

func commitStreamPending(behavior *logViewBehavior) error {
	if behavior.nextLine == math.MaxUint64 {
		return ErrControlCapacity
	}
	record := retainedLogRecord{
		key:          fmt.Sprintf("stream.%016x", behavior.nextLine),
		level:        LogInfo,
		lines:        [][]markdownCell{cloneMarkdownCells(behavior.pendingCells)},
		storageBytes: markdownCellsBytes(behavior.pendingCells),
		sourceBytes:  len(behavior.pending),
		lossCounted:  behavior.pendingLossCounted,
	}
	behavior.nextLine++
	behavior.records = append(behavior.records, record)
	behavior.pending = nil
	behavior.pendingCells = nil
	behavior.pendingTruncated = false
	behavior.pendingLossCounted = false
	return enforceStreamCapacity(behavior)
}

func markdownCellsBytes(cells []markdownCell) int {
	total := 0
	for _, cell := range cells {
		total += len(cell.grapheme)
	}
	return total
}

func enforceStreamCapacity(behavior *logViewBehavior) error {
	pendingBytes := markdownCellsBytes(behavior.pendingCells)
	for len(behavior.records) > behavior.capacity.Records ||
		logRetainedStorage(behavior.records)+pendingBytes > behavior.capacity.Bytes {
		if len(behavior.records) == 0 {
			break
		}
		oldest := behavior.records[0]
		behavior.records = behavior.records[1:]
		var err error
		behavior.droppedRecords, behavior.droppedBytes, err = addContentDrop(
			behavior.droppedRecords,
			behavior.droppedBytes,
			oldest,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func logCanMove(state *controlState, behavior logViewBehavior) bool {
	return scrollViewCanMove(state, behavior.scroll)
}

func (a *App) logViewKeyLocked(
	control *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if control == nil {
		return "", "", false, false
	}
	behavior, ok := control.behavior.(logViewBehavior)
	if !ok {
		return "", "", false, false
	}
	beforeFollow := behavior.follow
	holder := &controlState{
		id: control.id, bounds: control.bounds, behavior: behavior.scroll,
	}
	command, target, handled, changed := a.scrollViewKeyLocked(holder, key)
	if !handled {
		return "", "", false, false
	}
	behavior.scroll = holder.behavior.(scrollViewBehavior)
	maximum := calculateScrollViewGeometry(
		control.bounds.Size(),
		behavior.scroll,
	).maximumOffset
	if key == KeyEnd {
		behavior.follow = true
	} else if behavior.scroll.state.Offset.Y < maximum.Y &&
		(key == KeyUp || key == KeyDown || key == KeyPageUp ||
			key == KeyPageDown || key == KeyHome) {
		behavior.follow = false
	}
	if behavior.follow != beforeFollow {
		changed = true
		command = behavior.scroll.changeCommand
		target = control.id
	}
	if changed {
		control.behavior = behavior
	}
	return command, target, true, changed
}

func reconcileLogViewLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(logViewBehavior)
	if !ok {
		return false
	}
	next := reflowLogView(behavior, state.bounds.Size())
	if logViewBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func logDetails(bounds Rect, behavior logViewBehavior) LogViewDetails {
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	first := ""
	last := ""
	if len(behavior.records) > 0 {
		first = behavior.records[0].key
		last = behavior.records[len(behavior.records)-1].key
	}
	return LogViewDetails{
		Capacity:        behavior.capacity,
		RetainedRecords: len(behavior.records),
		RetainedBytes:   logRetainedStorage(behavior.records),
		DroppedRecords:  behavior.droppedRecords,
		DroppedBytes:    behavior.droppedBytes,
		FirstKey:        first,
		LastKey:         last,
		Follow:          behavior.follow,
		Viewport:        viewport,
	}
}

func streamDetails(bounds Rect, behavior logViewBehavior) StreamViewDetails {
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	return StreamViewDetails{
		Capacity:            behavior.capacity,
		RetainedLines:       len(behavior.records),
		RetainedBytes:       logRetainedStorage(behavior.records),
		PendingBytes:        len(behavior.pending),
		PendingCells:        len(behavior.pendingCells),
		PendingStorageBytes: markdownCellsBytes(behavior.pendingCells),
		PendingTruncated:    behavior.pendingTruncated,
		DroppedLines:        behavior.droppedRecords,
		DroppedBytes:        behavior.droppedBytes,
		Follow:              behavior.follow,
		Viewport:            viewport,
	}
}
