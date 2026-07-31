package expletives

import (
	"context"
	"fmt"
	"sort"
)

// StatusSegment is one copied static-context or command-hint descriptor.
// Exactly one of Text and Command must be set. Higher Priority values are
// retained first when the physical row is too narrow.
type StatusSegment struct {
	Key      string
	Text     string
	Command  CommandID
	Priority int
}

// StatusBarOptions configures the unique bottom-row application chrome.
type StatusBarOptions struct {
	PanelOptions
	Segments []StatusSegment
	// ShortcutStyle defaults to "status.shortcut".
	ShortcutStyle StyleID
	// DisabledStyle defaults to "status.disabled".
	DisabledStyle StyleID
}

// StatusBar is a copy-safe, non-container, non-focusable leaf Control.
type StatusBar struct{ controlHandle }

type normalizedStatusSegment struct {
	descriptor StatusSegment
	content    normalizedDisplayText
}

type statusBarBehavior struct {
	segments      []normalizedStatusSegment
	shortcutStyle StyleID
	disabledStyle StyleID
}

type statusRenderSegment struct {
	details       StatusSegmentDetails
	cells         []string
	shortcutStart int
	shortcutWidth int
}

// NewStatusBar constructs and atomically inserts one StatusBar.
func NewStatusBar(
	parent Container,
	options StatusBarOptions,
) (*StatusBar, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	bar, err := tx.NewStatusBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return bar, nil
}

// NewStatusBar records construction of one provisional StatusBar.
func (t *Transaction) NewStatusBar(
	parent Container,
	options StatusBarOptions,
) (*StatusBar, error) {
	if err := t.usable(); err != nil {
		return nil, err
	}
	if parent == nil || parent.containerState() == nil ||
		parent.containerState() != t.app.root.state {
		return nil, fmt.Errorf(
			"%w: StatusBar must be parented directly by App.Root()",
			ErrInvalidParent,
		)
	}
	if options.Bounds != (Rect{}) || options.MinimumSize != (Size{}) {
		return nil, fmt.Errorf(
			"%w: StatusBar geometry is derived from the application surface",
			ErrInvalidGeometry,
		)
	}
	segments, err := normalizeStatusSegments(options.Segments)
	if err != nil {
		return nil, err
	}
	shortcutStyle, err := normalizeStyleID(
		options.ShortcutStyle,
		"status.shortcut",
	)
	if err != nil {
		return nil, err
	}
	disabledStyle, err := normalizeStyleID(
		options.DisabledStyle,
		"status.disabled",
	)
	if err != nil {
		return nil, err
	}
	behavior := statusBarBehavior{
		segments:      segments,
		shortcutStyle: shortcutStyle,
		disabledStyle: disabledStyle,
	}
	options.Bounds = statusBarSurfaceRect(t.app.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlStatusBar,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	bar := &StatusBar{controlHandle: controlHandle{state: state}}
	state.control = bar
	return bar, nil
}

// Segments returns a caller-owned copy of the current ordered descriptors.
func (b *StatusBar) Segments() []StatusSegment {
	if b == nil || b.state == nil || b.state.app == nil {
		return nil
	}
	b.state.app.mu.RLock()
	defer b.state.app.mu.RUnlock()
	behavior, ok := b.state.behavior.(statusBarBehavior)
	if !ok || b.state.aborted {
		return nil
	}
	return publicStatusSegments(behavior.segments)
}

// SetSegments atomically replaces the complete copied segment inventory.
func (b *StatusBar) SetSegments(segments []StatusSegment) error {
	if b == nil || b.state == nil || b.state.app == nil {
		return ErrInvalidControl
	}
	tx := b.state.app.NewTransaction()
	if err := tx.SetStatusSegments(b, segments); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// SetStatusSegments records replacement of a StatusBar's complete segment
// inventory.
func (t *Transaction) SetStatusSegments(
	bar *StatusBar,
	segments []StatusSegment,
) error {
	if bar == nil {
		return ErrInvalidControl
	}
	state, err := t.control(bar)
	if err != nil {
		return err
	}
	current, ok := state.behavior.(statusBarBehavior)
	if !ok {
		return fmt.Errorf("%w: control is not a StatusBar", ErrInvalidControl)
	}
	normalized, err := normalizeStatusSegments(segments)
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	current.segments = normalized
	t.mutations = append(t.mutations, transactionMutation{
		kind:     mutationStatusSegments,
		state:    state,
		behavior: current,
	})
	return nil
}

func normalizeStatusSegments(
	source []StatusSegment,
) ([]normalizedStatusSegment, error) {
	if len(source) > MaxStatusBarSegments {
		return nil, fmt.Errorf(
			"%w: StatusBar exceeds %d segments",
			ErrControlCapacity,
			MaxStatusBarSegments,
		)
	}
	normalized := make([]normalizedStatusSegment, len(source))
	seen := make(map[string]bool, len(source))
	for index, segment := range source {
		if !validBoundedIdentifier(segment.Key) || seen[segment.Key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate StatusSegment key %q",
				ErrInvalidControl,
				segment.Key,
			)
		}
		seen[segment.Key] = true
		hasText := segment.Text != ""
		hasCommand := segment.Command != ""
		if hasText == hasCommand {
			return nil, fmt.Errorf(
				"%w: StatusSegment %q requires exactly one of Text or Command",
				ErrInvalidControl,
				segment.Key,
			)
		}
		var content normalizedDisplayText
		var err error
		if hasText {
			content, err = normalizeDisplayText(segment.Text, false)
			if err != nil {
				return nil, err
			}
			segment.Text = content.text
		} else if !validBoundedIdentifier(string(segment.Command)) {
			return nil, fmt.Errorf(
				"%w: invalid StatusSegment command",
				ErrInvalidControl,
			)
		}
		normalized[index] = normalizedStatusSegment{
			descriptor: segment,
			content:    content,
		}
	}
	return normalized, nil
}

func publicStatusSegments(
	segments []normalizedStatusSegment,
) []StatusSegment {
	result := make([]StatusSegment, len(segments))
	for index, segment := range segments {
		result[index] = segment.descriptor
	}
	return result
}

func (b statusBarBehavior) clientInset() int { return 0 }

func (b statusBarBehavior) paintDecoration(
	*App,
	*IntendedFrame,
	*controlState,
	Rect,
	Rect,
) {
}

func (b statusBarBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		StatusBar: &StatusBarDetails{Segments: []StatusSegmentDetails{}},
	}
}

func (b statusBarBehavior) intrinsicMinimum() Size {
	return Size{Width: 1, Height: 1}
}

func (b statusBarBehavior) additionalStyles() []StyleID {
	return []StyleID{b.shortcutStyle, b.disabledStyle}
}

func statusBarBehaviorEqual(left, right statusBarBehavior) bool {
	if left.shortcutStyle != right.shortcutStyle ||
		left.disabledStyle != right.disabledStyle ||
		len(left.segments) != len(right.segments) {
		return false
	}
	for index := range left.segments {
		if left.segments[index].descriptor != right.segments[index].descriptor {
			return false
		}
	}
	return true
}

func validateStatusCommandsLocked(
	app *App,
	behavior statusBarBehavior,
) error {
	for _, segment := range behavior.segments {
		command := segment.descriptor.Command
		if command == "" {
			continue
		}
		if _, exists := app.commands[command]; !exists {
			return fmt.Errorf(
				"%w: StatusBar command %q is not registered",
				ErrInvalidControl,
				command,
			)
		}
	}
	return nil
}

func statusBarSurfaceRect(size Size) Rect {
	if size.Width <= 0 || size.Height <= 0 {
		return Rect{}
	}
	return Rect{Y: size.Height - 1, Width: size.Width, Height: 1}
}

func (a *App) firstStatusBarLocked() *controlState {
	var result *controlState
	var visit func(*controlState)
	visit = func(state *controlState) {
		if result != nil {
			return
		}
		if state.kind == ControlStatusBar &&
			a.effectivelyVisibleLocked(state) {
			result = state
			return
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	visit(a.root.state)
	return result
}

func (a *App) statusBarRenderPlanLocked(
	behavior statusBarBehavior,
	width int,
) []statusRenderSegment {
	plans := make([]statusRenderSegment, len(behavior.segments))
	for index, segment := range behavior.segments {
		descriptor := segment.descriptor
		plan := &plans[index]
		plan.details = StatusSegmentDetails{
			Key:      descriptor.Key,
			Priority: descriptor.Priority,
			Enabled:  true,
		}
		if descriptor.Command == "" {
			plan.details.Label = descriptor.Text
			plan.cells = append(plan.cells, " ")
			plan.cells = append(plan.cells, segment.content.lines[0]...)
			plan.cells = append(plan.cells, " ")
			continue
		}
		definition, exists := a.commands[descriptor.Command]
		label := effectiveCommandLabel(definition, descriptor.Command)
		chord := a.firstChordLocked(descriptor.Command)
		plan.details.Label = label.text
		plan.details.Command = descriptor.Command
		plan.details.Enabled = exists && definition.Enabled
		plan.details.DisabledReason =
			effectiveDisabledReason(definition, exists)
		plan.details.Checked = definition.Checked
		plan.details.Chord = chord
		plan.cells = append(plan.cells, " ")
		if chord != nil {
			plan.shortcutStart = len(plan.cells)
			chordCells := displayChord(*chord)
			plan.shortcutWidth = len(chordCells)
			plan.cells = append(plan.cells, chordCells...)
			plan.cells = append(plan.cells, " ")
		}
		plan.cells = append(plan.cells, label.lines[0]...)
		plan.cells = append(plan.cells, " ")
	}
	if width <= 0 || len(plans) == 0 {
		return plans
	}

	priorityOrder := make([]int, len(plans))
	for index := range priorityOrder {
		priorityOrder[index] = index
	}
	sort.SliceStable(priorityOrder, func(left, right int) bool {
		return behavior.segments[priorityOrder[left]].descriptor.Priority >
			behavior.segments[priorityOrder[right]].descriptor.Priority
	})
	selected := make([]bool, len(plans))
	remaining := width
	selectedCount := 0
	for _, index := range priorityOrder {
		desired := len(plans[index].cells)
		if desired <= remaining {
			selected[index] = true
			remaining -= desired
			selectedCount++
			continue
		}
		if selectedCount == 0 && remaining > 0 {
			selected[index] = true
			plans[index].details.Clipped = true
			plans[index].cells = clippedStatusCells(&plans[index], remaining)
			remaining = 0
		}
	}
	x := 0
	for index := range plans {
		if !selected[index] {
			plans[index].cells = nil
			continue
		}
		plans[index].details.Rendered = true
		plans[index].details.Bounds = Rect{
			X: x, Width: len(plans[index].cells), Height: 1,
		}
		x += len(plans[index].cells)
	}
	return plans
}

func clippedStatusCells(plan *statusRenderSegment, width int) []string {
	if width <= 0 {
		return nil
	}
	cells := plan.cells
	if len(cells) > width && len(cells) != 0 && cells[0] == " " {
		cells = cells[1:]
		plan.shortcutStart--
	}
	if len(cells) > width {
		cells = cells[:width]
	}
	if plan.shortcutStart < 0 {
		plan.shortcutWidth += plan.shortcutStart
		plan.shortcutStart = 0
	}
	if plan.shortcutStart >= len(cells) {
		plan.shortcutWidth = 0
	} else if plan.shortcutStart+plan.shortcutWidth > len(cells) {
		plan.shortcutWidth = len(cells) - plan.shortcutStart
	}
	return append([]string(nil), cells...)
}

func (a *App) paintStatusBarChromeLocked(frame *IntendedFrame) {
	state := a.firstStatusBarLocked()
	if state == nil {
		return
	}
	behavior := state.behavior.(statusBarBehavior)
	rect := statusBarSurfaceRect(a.size)
	if rect.Empty() {
		return
	}
	a.fillLocked(frame, rect, state.style, state.id)
	for _, segment := range a.statusBarRenderPlanLocked(behavior, rect.Width) {
		if !segment.details.Rendered {
			continue
		}
		style := state.style
		if !segment.details.Enabled {
			style = behavior.disabledStyle
		}
		x := rect.X + segment.details.Bounds.X
		a.paintMenuCellsLocked(
			frame,
			rect,
			x,
			rect.Y,
			segment.cells,
			style,
			state.id,
		)
		if segment.details.Enabled && segment.shortcutWidth > 0 {
			start := segment.shortcutStart
			end := start + segment.shortcutWidth
			a.paintMenuCellsLocked(
				frame,
				rect,
				x+start,
				rect.Y,
				segment.cells[start:end],
				behavior.shortcutStyle,
				state.id,
			)
		}
	}
}
