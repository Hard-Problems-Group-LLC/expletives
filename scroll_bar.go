package expletives

import (
	"context"
	"errors"
	"fmt"
)

// ScrollBarState is one complete copied viewport update.
type ScrollBarState struct {
	ContentSize  int
	ViewportSize int
	Offset       int
}

// ScrollBarOptions configures one focusable viewport-position control.
type ScrollBarOptions struct {
	PanelOptions
	Orientation    Orientation
	State          ScrollBarState
	ArrowStep      int
	PageStep       int
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// ScrollBar is a copy-safe focusable viewport-position control.
type ScrollBar struct{ controlHandle }

type scrollBarBehavior struct {
	orientation    Orientation
	state          ScrollBarState
	arrowStep      int
	pageStep       int
	disabled       bool
	disabledReason string
	changeCommand  CommandID
}

type scrollBarGeometry struct {
	trackStart int
	trackSize  int
	thumbStart int
	thumbSize  int
}

// NewScrollBar constructs and atomically inserts a ScrollBar.
func NewScrollBar(
	parent Container,
	options ScrollBarOptions,
) (*ScrollBar, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewScrollBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewScrollBar records construction of a provisional ScrollBar.
func (t *Transaction) NewScrollBar(
	parent Container,
	options ScrollBarOptions,
) (*ScrollBar, error) {
	behavior, err := normalizeScrollBarBehavior(scrollBarBehavior{
		orientation:    options.Orientation,
		state:          options.State,
		arrowStep:      options.ArrowStep,
		pageStep:       options.PageStep,
		disabled:       options.Disabled,
		disabledReason: options.DisabledReason,
		changeCommand:  options.ChangeCommand,
	})
	if err != nil {
		return nil, err
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlScrollBar,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &ScrollBar{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

func normalizeScrollBarBehavior(
	behavior scrollBarBehavior,
) (scrollBarBehavior, error) {
	switch behavior.orientation {
	case Horizontal, Vertical:
	default:
		return scrollBarBehavior{}, fmt.Errorf(
			"%w: invalid ScrollBar orientation",
			ErrValidation,
		)
	}
	state, err := normalizeScrollBarState(behavior.state)
	if err != nil {
		return scrollBarBehavior{}, err
	}
	if behavior.arrowStep == 0 {
		behavior.arrowStep = 1
	}
	if behavior.pageStep == 0 {
		behavior.pageStep = max(1, state.ViewportSize)
	}
	if behavior.arrowStep < 1 ||
		behavior.arrowStep > maxCoordinateMagnitude ||
		behavior.pageStep < 1 ||
		behavior.pageStep > maxCoordinateMagnitude {
		return scrollBarBehavior{}, fmt.Errorf(
			"%w: invalid ScrollBar step",
			ErrValidation,
		)
	}
	reason, err := normalizeDisabledReason(
		behavior.disabled,
		behavior.disabledReason,
	)
	if err != nil {
		return scrollBarBehavior{}, err
	}
	if err := validateOptionalCommand(behavior.changeCommand); err != nil {
		return scrollBarBehavior{}, err
	}
	behavior.state = state
	behavior.disabledReason = reason
	return behavior, nil
}

func normalizeScrollBarState(state ScrollBarState) (ScrollBarState, error) {
	if state.ContentSize < 0 || state.ContentSize > maxCoordinateMagnitude ||
		state.ViewportSize < 0 ||
		state.ViewportSize > maxCoordinateMagnitude ||
		state.Offset < 0 || state.Offset > maxCoordinateMagnitude {
		return ScrollBarState{}, fmt.Errorf(
			"%w: invalid ScrollBar viewport state",
			ErrValidation,
		)
	}
	maximum := scrollBarMaximumOffset(state)
	if state.Offset > maximum {
		return ScrollBarState{}, fmt.Errorf(
			"%w: ScrollBar offset exceeds maximum",
			ErrValidation,
		)
	}
	if maximum == 0 {
		state.Offset = 0
	}
	return state, nil
}

func scrollBarMaximumOffset(state ScrollBarState) int {
	return max(0, state.ContentSize-state.ViewportSize)
}

func (scrollBarBehavior) clientInset() int { return 0 }

func (b scrollBarBehavior) intrinsicMinimum() Size {
	if b.orientation == Vertical {
		return Size{Width: 1, Height: 3}
	}
	return Size{Width: 3, Height: 1}
}

func (scrollBarBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"scrollbar.page",
		"scrollbar.arrow",
		"scrollbar.thumb",
		"scrollbar.focused",
		"scrollbar.disabled",
	}
}

func (scrollBarBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		ScrollBar: &ScrollBarDetails{},
	}
}

func (b scrollBarBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
) {
	axis := absolute.Width
	if b.orientation == Vertical {
		axis = absolute.Height
	}
	geometry := calculateScrollBarGeometry(axis, b.state)
	baseStyle := StyleID("scrollbar.page")
	arrowStyle := StyleID("scrollbar.arrow")
	thumbStyle := StyleID("scrollbar.thumb")
	if b.disabled {
		baseStyle = "scrollbar.disabled"
		arrowStyle = baseStyle
		thumbStyle = baseStyle
	} else if app.focus == control {
		thumbStyle = "scrollbar.focused"
	}

	for position := 0; position < axis; position++ {
		glyph := "░"
		style := baseStyle
		if axis >= 3 && position == 0 {
			glyph = "◄"
			if b.orientation == Vertical {
				glyph = "▲"
			}
			style = arrowStyle
		} else if axis >= 3 && position == axis-1 {
			glyph = "►"
			if b.orientation == Vertical {
				glyph = "▼"
			}
			style = arrowStyle
		} else if position >= geometry.thumbStart &&
			position < geometry.thumbStart+geometry.thumbSize {
			glyph = "█"
			if !b.disabled && app.focus == control {
				glyph = "▓"
			}
			style = thumbStyle
		}
		x, y := absolute.X+position, absolute.Y
		if b.orientation == Vertical {
			x, y = absolute.X, absolute.Y+position
		}
		app.setClippedCellLocked(
			frame,
			clip,
			x,
			y,
			glyph,
			style,
			app.styles[style],
			control.id,
		)
	}
}

func calculateScrollBarGeometry(
	axis int,
	state ScrollBarState,
) scrollBarGeometry {
	if axis <= 0 {
		return scrollBarGeometry{}
	}
	geometry := scrollBarGeometry{trackSize: axis}
	if axis >= 3 {
		geometry.trackStart = 1
		geometry.trackSize = axis - 2
	}
	if geometry.trackSize <= 0 {
		return geometry
	}
	maximum := scrollBarMaximumOffset(state)
	switch {
	case maximum == 0:
		geometry.thumbSize = geometry.trackSize
	case state.ViewportSize == 0:
		geometry.thumbSize = 1
	default:
		geometry.thumbSize = max(
			1,
			geometry.trackSize*state.ViewportSize/state.ContentSize,
		)
		geometry.thumbSize = min(geometry.thumbSize, geometry.trackSize)
	}
	travel := geometry.trackSize - geometry.thumbSize
	geometry.thumbStart = geometry.trackStart
	if travel > 0 && maximum > 0 {
		geometry.thumbStart += state.Offset * travel / maximum
	}
	return geometry
}

func scrollBarDetails(
	bounds Rect,
	behavior scrollBarBehavior,
) ScrollBarDetails {
	axis := bounds.Width
	if behavior.orientation == Vertical {
		axis = bounds.Height
	}
	geometry := calculateScrollBarGeometry(axis, behavior.state)
	return ScrollBarDetails{
		Orientation:    behavior.orientation,
		ContentSize:    behavior.state.ContentSize,
		ViewportSize:   behavior.state.ViewportSize,
		Offset:         behavior.state.Offset,
		MaximumOffset:  scrollBarMaximumOffset(behavior.state),
		ArrowStep:      behavior.arrowStep,
		PageStep:       behavior.pageStep,
		TrackStart:     geometry.trackStart,
		TrackSize:      geometry.trackSize,
		ThumbStart:     geometry.thumbStart,
		ThumbSize:      geometry.thumbSize,
		Enabled:        !behavior.disabled,
		DisabledReason: behavior.disabledReason,
		ChangeCommand:  behavior.changeCommand,
	}
}

// State returns the current canonical ScrollBar viewport state.
func (s *ScrollBar) State() ScrollBarState {
	if s == nil || s.state == nil || s.state.app == nil {
		return ScrollBarState{}
	}
	s.state.app.mu.RLock()
	defer s.state.app.mu.RUnlock()
	if s.state.aborted || s.state.destroyed {
		return ScrollBarState{}
	}
	behavior, ok := s.state.behavior.(scrollBarBehavior)
	if !ok {
		return ScrollBarState{}
	}
	return behavior.state
}

// SetState atomically applies one complete ScrollBar viewport state.
func (s *ScrollBar) SetState(state ScrollBarState) error {
	return s.Update(context.Background(), state)
}

// Update applies one copied ScrollBar state while observing ctx.
func (s *ScrollBar) Update(
	ctx context.Context,
	state ScrollBarState,
) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if s == nil || s.state == nil || s.state.app == nil {
		return ErrInvalidControl
	}
	transaction := s.state.app.NewTransaction()
	if err := transaction.SetScrollBarState(s, state); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// Focus gives this eligible ScrollBar keyboard focus.
func (s *ScrollBar) Focus() error { return focusSelectionControl(s) }

// SetScrollBarState records one atomic complete viewport update.
func (t *Transaction) SetScrollBarState(
	control *ScrollBar,
	state ScrollBarState,
) error {
	target, err := t.control(control)
	if err != nil || target.kind != ControlScrollBar {
		return ErrInvalidControl
	}
	normalized, err := normalizeScrollBarState(state)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(scrollBarBehavior)
	if !ok {
		return ErrInvalidControl
	}
	behavior.state = normalized
	return t.recordScrollBarBehavior(target, behavior)
}

func (t *Transaction) recordScrollBarBehavior(
	state *controlState,
	behavior scrollBarBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationScrollBar, state: state, behavior: behavior,
	})
	return nil
}

func (t *Transaction) selectedControlBehavior(
	state *controlState,
) controlBehavior {
	return t.recordedControlBehavior(state)
}

func scrollBarBehaviorEqual(left, right scrollBarBehavior) bool {
	return left == right
}

func (a *App) scrollBarKeyLocked(
	control *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if control == nil {
		return "", "", false, false
	}
	behavior, ok := control.behavior.(scrollBarBehavior)
	if !ok || behavior.disabled {
		return "", "", false, false
	}
	next := behavior.state.Offset
	handled := true
	switch key {
	case KeyLeft:
		if behavior.orientation != Horizontal {
			handled = false
		} else {
			next -= behavior.arrowStep
		}
	case KeyRight:
		if behavior.orientation != Horizontal {
			handled = false
		} else {
			next += behavior.arrowStep
		}
	case KeyUp:
		if behavior.orientation != Vertical {
			handled = false
		} else {
			next -= behavior.arrowStep
		}
	case KeyDown:
		if behavior.orientation != Vertical {
			handled = false
		} else {
			next += behavior.arrowStep
		}
	case KeyPageUp:
		next -= behavior.pageStep
	case KeyPageDown:
		next += behavior.pageStep
	case KeyHome:
		next = 0
	case KeyEnd:
		next = scrollBarMaximumOffset(behavior.state)
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	next = max(0, min(next, scrollBarMaximumOffset(behavior.state)))
	if next == behavior.state.Offset {
		return "", "", true, false
	}
	behavior.state.Offset = next
	control.behavior = behavior
	return behavior.changeCommand, control.id, true, true
}
