package expletives

import (
	"context"
	"errors"
	"fmt"
)

// ViewportState is one complete copied logical content extent and offset.
type ViewportState struct {
	ContentSize Size  `json:"content_size"`
	Offset      Point `json:"offset"`
}

// ScrollBarVisibility selects integrated scrollbar reservation policy.
type ScrollBarVisibility string

const (
	ScrollBarVisibilityAuto   ScrollBarVisibility = "auto"
	ScrollBarVisibilityAlways ScrollBarVisibility = "always"
	ScrollBarVisibilityNever  ScrollBarVisibility = "never"
)

// ScrollViewOptions configures one generic unframed Viewport.
type ScrollViewOptions struct {
	PanelOptions
	ContentAutomationKey string
	ContentStyle         StyleID
	State                ViewportState
	ArrowStep            Size
	PageStep             Size
	Disabled             bool
	DisabledReason       string
	ChangeCommand        CommandID
}

// ScrollablePanelOptions configures one framed viewport with integrated bars.
type ScrollablePanelOptions struct {
	ScrollViewOptions
	BorderStyle      StyleID
	BorderForm       BorderForm
	BorderForeground *Color
	BorderBackground *Color
	HorizontalBar    ScrollBarVisibility
	VerticalBar      ScrollBarVisibility
}

// Viewport is a copy-safe unframed scrolling container.
type Viewport struct{ containerHandle }

// ScrollablePanel is a copy-safe framed scrolling container.
type ScrollablePanel struct{ containerHandle }

type scrollViewBehavior struct {
	border           borderBehavior
	state            ViewportState
	arrowStep        Size
	pageStep         Size
	disabled         bool
	disabledReason   string
	changeCommand    CommandID
	content          *controlState
	horizontalPolicy ScrollBarVisibility
	verticalPolicy   ScrollBarVisibility
	integratedBars   bool
}

type scrollViewGeometry struct {
	viewport          Rect
	horizontalBar     Rect
	verticalBar       Rect
	corner            Rect
	maximumOffset     Point
	horizontalVisible bool
	verticalVisible   bool
}

// NewViewport constructs a Viewport and its managed Content Panel atomically.
func NewViewport(parent Container, options ScrollViewOptions) (*Viewport, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewViewport(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewScrollablePanel constructs a ScrollablePanel and managed Content Panel
// atomically.
func NewScrollablePanel(
	parent Container,
	options ScrollablePanelOptions,
) (*ScrollablePanel, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewScrollablePanel(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewViewport records construction of a provisional Viewport and Content.
func (t *Transaction) NewViewport(
	parent Container,
	options ScrollViewOptions,
) (*Viewport, error) {
	panel, err := t.newScrollView(
		parent,
		options,
		ControlViewport,
		BorderOptions{Form: BorderNone},
		ScrollBarVisibilityNever,
		ScrollBarVisibilityNever,
		false,
	)
	if err != nil {
		return nil, err
	}
	control := &Viewport{containerHandle: panel.containerHandle}
	panel.state.control = control
	panel.state.container = control
	return control, nil
}

// NewScrollablePanel records construction of a provisional ScrollablePanel
// and Content.
func (t *Transaction) NewScrollablePanel(
	parent Container,
	options ScrollablePanelOptions,
) (*ScrollablePanel, error) {
	panel, err := t.newScrollView(
		parent,
		options.ScrollViewOptions,
		ControlScrollablePanel,
		BorderOptions{
			Form:       options.BorderForm,
			Style:      options.BorderStyle,
			Foreground: options.BorderForeground,
			Background: options.BorderBackground,
		},
		options.HorizontalBar,
		options.VerticalBar,
		true,
	)
	if err != nil {
		return nil, err
	}
	control := &ScrollablePanel{containerHandle: panel.containerHandle}
	panel.state.control = control
	panel.state.container = control
	return control, nil
}

func (t *Transaction) newScrollView(
	parent Container,
	options ScrollViewOptions,
	kind ControlKind,
	borderOptions BorderOptions,
	horizontalPolicy ScrollBarVisibility,
	verticalPolicy ScrollBarVisibility,
	integratedBars bool,
) (_ *Panel, err error) {
	startCreates := len(t.creates)
	defer func() {
		if err != nil {
			t.creates = t.creates[:startCreates]
		}
	}()

	contentKey := options.ContentAutomationKey
	if contentKey == "" && options.AutomationKey != "" {
		contentKey = options.AutomationKey + ".content"
	}
	if contentKey != "" && !validBoundedIdentifier(contentKey) {
		return nil, errors.New(
			"expletives: scroll Content automation key is invalid or too long",
		)
	}
	behavior, err := normalizeScrollViewBehavior(scrollViewBehavior{
		state:            options.State,
		arrowStep:        options.ArrowStep,
		pageStep:         options.PageStep,
		disabled:         options.Disabled,
		disabledReason:   options.DisabledReason,
		changeCommand:    options.ChangeCommand,
		horizontalPolicy: horizontalPolicy,
		verticalPolicy:   verticalPolicy,
		integratedBars:   integratedBars,
	})
	if err != nil {
		return nil, err
	}
	border, err := newBorderBehavior(
		"",
		borderOptions.Style,
		kind,
		borderOptions.Form,
		borderOptions.Foreground,
		borderOptions.Background,
	)
	if err != nil {
		return nil, err
	}
	behavior.border = border
	panel, err := t.newControl(parent, options.PanelOptions, kind, behavior)
	if err != nil {
		return nil, err
	}
	content, err := t.NewPanel(panel, PanelOptions{
		AutomationKey: contentKey,
		Bounds: Rect{
			Width:  behavior.state.ContentSize.Width,
			Height: behavior.state.ContentSize.Height,
		},
		Style: options.ContentStyle,
		LayoutHints: LayoutHints{
			Horizontal: LayoutSizeNatural,
			Vertical:   LayoutSizeNatural,
		},
	})
	if err != nil {
		return nil, err
	}
	content.state.managedBy = panel.state
	behavior.content = content.state
	panel.state.behavior = behavior
	if options.MinimumSize == (Size{}) {
		panel.state.autoMinimum = true
		panel.state.minimumSize = behavior.intrinsicMinimum()
	}
	return panel, nil
}

func normalizeScrollViewBehavior(
	behavior scrollViewBehavior,
) (scrollViewBehavior, error) {
	state, err := normalizeViewportState(behavior.state)
	if err != nil {
		return scrollViewBehavior{}, err
	}
	behavior.state = state
	for _, axis := range []*int{
		&behavior.arrowStep.Width,
		&behavior.arrowStep.Height,
	} {
		if *axis == 0 {
			*axis = 1
		}
	}
	if behavior.arrowStep.Width < 1 ||
		behavior.arrowStep.Height < 1 ||
		behavior.arrowStep.Width > maxCoordinateMagnitude ||
		behavior.arrowStep.Height > maxCoordinateMagnitude ||
		behavior.pageStep.Width < 0 ||
		behavior.pageStep.Height < 0 ||
		behavior.pageStep.Width > maxCoordinateMagnitude ||
		behavior.pageStep.Height > maxCoordinateMagnitude {
		return scrollViewBehavior{}, fmt.Errorf(
			"%w: invalid scroll viewport step",
			ErrValidation,
		)
	}
	behavior.horizontalPolicy, err = normalizeScrollBarVisibility(
		behavior.horizontalPolicy,
	)
	if err != nil {
		return scrollViewBehavior{}, err
	}
	behavior.verticalPolicy, err = normalizeScrollBarVisibility(
		behavior.verticalPolicy,
	)
	if err != nil {
		return scrollViewBehavior{}, err
	}
	reason, err := normalizeDisabledReason(
		behavior.disabled,
		behavior.disabledReason,
	)
	if err != nil {
		return scrollViewBehavior{}, err
	}
	if err := validateOptionalCommand(behavior.changeCommand); err != nil {
		return scrollViewBehavior{}, err
	}
	behavior.disabledReason = reason
	return behavior, nil
}

func normalizeViewportState(state ViewportState) (ViewportState, error) {
	if err := validateSize(state.ContentSize); err != nil {
		return ViewportState{}, fmt.Errorf(
			"%w: invalid viewport content size",
			ErrValidation,
		)
	}
	if state.Offset.X < 0 || state.Offset.Y < 0 ||
		state.Offset.X > maxCoordinateMagnitude ||
		state.Offset.Y > maxCoordinateMagnitude ||
		state.Offset.X > state.ContentSize.Width ||
		state.Offset.Y > state.ContentSize.Height {
		return ViewportState{}, fmt.Errorf(
			"%w: invalid viewport offset",
			ErrValidation,
		)
	}
	if state.ContentSize.Width == 0 {
		state.Offset.X = 0
	}
	if state.ContentSize.Height == 0 {
		state.Offset.Y = 0
	}
	return state, nil
}

func normalizeScrollBarVisibility(
	visibility ScrollBarVisibility,
) (ScrollBarVisibility, error) {
	if visibility == "" {
		visibility = ScrollBarVisibilityAuto
	}
	switch visibility {
	case ScrollBarVisibilityAuto, ScrollBarVisibilityAlways,
		ScrollBarVisibilityNever:
		return visibility, nil
	default:
		return "", fmt.Errorf(
			"%w: invalid integrated scrollbar visibility",
			ErrValidation,
		)
	}
}

func (b scrollViewBehavior) controlBorder() borderBehavior { return b.border }

func (b scrollViewBehavior) clientInset() int {
	if b.border.form == BorderNone {
		return 0
	}
	return 1
}

func (b scrollViewBehavior) intrinsicMinimum() Size {
	if b.border.form == BorderNone {
		return Size{Width: 1, Height: 1}
	}
	return Size{Width: 3, Height: 3}
}

func (b scrollViewBehavior) additionalStyles() []StyleID {
	if !b.integratedBars ||
		(b.horizontalPolicy == ScrollBarVisibilityNever &&
			b.verticalPolicy == ScrollBarVisibilityNever) {
		return nil
	}
	return []StyleID{
		"scrollbar.page",
		"scrollbar.arrow",
		"scrollbar.thumb",
		"scrollbar.focused",
		"scrollbar.disabled",
		"scrollbar.corner",
	}
}

func (b scrollViewBehavior) details() ControlDetails {
	border := b.border.details()
	border.Container = &ContainerDetails{ClientInset: b.clientInset()}
	border.Scrollable = &ScrollableDetails{}
	return border
}

func (b scrollViewBehavior) controlClientRect(bounds Rect) Rect {
	geometry := calculateScrollViewGeometry(bounds.Size(), b)
	viewport := geometry.viewport
	viewport.X += bounds.X
	viewport.Y += bounds.Y
	return viewport
}

func (b scrollViewBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
) {
	b.border.paintDecoration(app, frame, control, absolute, clip)
	geometry := calculateScrollViewGeometry(absolute.Size(), b)
	if geometry.horizontalVisible {
		bar := translatedRect(geometry.horizontalBar, absolute.X, absolute.Y)
		scroll := scrollBarBehavior{
			orientation: Horizontal,
			state: ScrollBarState{
				ContentSize:  b.state.ContentSize.Width,
				ViewportSize: geometry.viewport.Width,
				Offset:       b.state.Offset.X,
			},
			arrowStep:      b.arrowStep.Width,
			pageStep:       effectiveScrollPageStep(b.pageStep.Width, geometry.viewport.Width),
			disabled:       b.disabled,
			disabledReason: b.disabledReason,
			changeCommand:  b.changeCommand,
		}
		scroll.paintDecoration(app, frame, control, bar, clip)
	}
	if geometry.verticalVisible {
		bar := translatedRect(geometry.verticalBar, absolute.X, absolute.Y)
		scroll := scrollBarBehavior{
			orientation: Vertical,
			state: ScrollBarState{
				ContentSize:  b.state.ContentSize.Height,
				ViewportSize: geometry.viewport.Height,
				Offset:       b.state.Offset.Y,
			},
			arrowStep:      b.arrowStep.Height,
			pageStep:       effectiveScrollPageStep(b.pageStep.Height, geometry.viewport.Height),
			disabled:       b.disabled,
			disabledReason: b.disabledReason,
			changeCommand:  b.changeCommand,
		}
		scroll.paintDecoration(app, frame, control, bar, clip)
	}
	if !geometry.corner.Empty() {
		corner := translatedRect(geometry.corner, absolute.X, absolute.Y)
		app.fillStyleLocked(
			frame,
			corner.Intersect(clip),
			"scrollbar.corner",
			control.id,
		)
	}
}

func translatedRect(rect Rect, x, y int) Rect {
	rect.X += x
	rect.Y += y
	return rect
}

func (r Rect) Size() Size {
	return Size{Width: r.Width, Height: r.Height}
}

func (a *App) fillStyleLocked(
	frame *IntendedFrame,
	rect Rect,
	style StyleID,
	owner ControlID,
) {
	resolved := a.styles[style]
	for y := rect.Y; y < rect.Y+rect.Height; y++ {
		for x := rect.X; x < rect.X+rect.Width; x++ {
			a.setCellLocked(frame, x, y, " ", style, resolved, owner)
		}
	}
}

func calculateScrollViewGeometry(
	size Size,
	behavior scrollViewBehavior,
) scrollViewGeometry {
	base := Rect{Width: size.Width, Height: size.Height}
	if behavior.border.form != BorderNone {
		base = insetRect(base, 1)
	}
	horizontal := behavior.integratedBars &&
		behavior.horizontalPolicy == ScrollBarVisibilityAlways &&
		base.Width > 0 && base.Height > 0
	vertical := behavior.integratedBars &&
		behavior.verticalPolicy == ScrollBarVisibilityAlways &&
		base.Width > 0 && base.Height > 0
	for range 3 {
		viewportWidth := max(0, base.Width-boolCell(vertical))
		viewportHeight := max(0, base.Height-boolCell(horizontal))
		nextHorizontal := horizontal
		nextVertical := vertical
		if behavior.integratedBars &&
			behavior.horizontalPolicy == ScrollBarVisibilityAuto {
			nextHorizontal = base.Width > 0 && base.Height > 0 &&
				behavior.state.ContentSize.Width > viewportWidth
		}
		if behavior.horizontalPolicy == ScrollBarVisibilityNever {
			nextHorizontal = false
		}
		if behavior.integratedBars &&
			behavior.verticalPolicy == ScrollBarVisibilityAuto {
			nextVertical = base.Width > 0 && base.Height > 0 &&
				behavior.state.ContentSize.Height > viewportHeight
		}
		if behavior.verticalPolicy == ScrollBarVisibilityNever {
			nextVertical = false
		}
		if nextHorizontal == horizontal && nextVertical == vertical {
			break
		}
		horizontal, vertical = nextHorizontal, nextVertical
	}
	geometry := scrollViewGeometry{
		viewport: Rect{
			X:      base.X,
			Y:      base.Y,
			Width:  max(0, base.Width-boolCell(vertical)),
			Height: max(0, base.Height-boolCell(horizontal)),
		},
		horizontalVisible: horizontal,
		verticalVisible:   vertical,
	}
	if horizontal {
		geometry.horizontalBar = Rect{
			X:      base.X,
			Y:      base.Y + geometry.viewport.Height,
			Width:  geometry.viewport.Width,
			Height: min(1, base.Height),
		}
	}
	if vertical {
		geometry.verticalBar = Rect{
			X:      base.X + geometry.viewport.Width,
			Y:      base.Y,
			Width:  min(1, base.Width),
			Height: geometry.viewport.Height,
		}
	}
	if horizontal && vertical && base.Width > 0 && base.Height > 0 {
		geometry.corner = Rect{
			X:     base.X + geometry.viewport.Width,
			Y:     base.Y + geometry.viewport.Height,
			Width: 1, Height: 1,
		}
	}
	geometry.maximumOffset = Point{
		X: max(0, behavior.state.ContentSize.Width-geometry.viewport.Width),
		Y: max(0, behavior.state.ContentSize.Height-geometry.viewport.Height),
	}
	return geometry
}

func boolCell(value bool) int {
	if value {
		return 1
	}
	return 0
}

func effectiveScrollPageStep(configured, viewport int) int {
	if configured > 0 {
		return configured
	}
	return max(1, viewport)
}

func clampViewportState(
	state ViewportState,
	geometry scrollViewGeometry,
) ViewportState {
	state.Offset.X = min(state.Offset.X, geometry.maximumOffset.X)
	state.Offset.Y = min(state.Offset.Y, geometry.maximumOffset.Y)
	if state.ContentSize.Width == 0 {
		state.Offset.X = 0
	}
	if state.ContentSize.Height == 0 {
		state.Offset.Y = 0
	}
	return state
}

func reconcileScrollViewLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(scrollViewBehavior)
	if !ok || behavior.content == nil || behavior.content.destroyed {
		return false
	}
	geometry := calculateScrollViewGeometry(state.bounds.Size(), behavior)
	next := clampViewportState(behavior.state, geometry)
	changed := next != behavior.state
	behavior.state = next
	state.behavior = behavior
	contentBounds := Rect{
		X:      -next.Offset.X,
		Y:      -next.Offset.Y,
		Width:  next.ContentSize.Width,
		Height: next.ContentSize.Height,
	}
	if behavior.content.bounds != contentBounds {
		behavior.content.bounds = contentBounds
		changed = true
	}
	return changed
}

func scrollViewDetails(
	bounds Rect,
	behavior scrollViewBehavior,
) ScrollableDetails {
	geometry := calculateScrollViewGeometry(bounds.Size(), behavior)
	state := clampViewportState(behavior.state, geometry)
	details := ScrollableDetails{
		State:          state,
		MaximumOffset:  geometry.maximumOffset,
		ViewportBounds: geometry.viewport,
		ArrowStep:      behavior.arrowStep,
		PageStep: Size{
			Width:  effectiveScrollPageStep(behavior.pageStep.Width, geometry.viewport.Width),
			Height: effectiveScrollPageStep(behavior.pageStep.Height, geometry.viewport.Height),
		},
		Enabled:           !behavior.disabled,
		DisabledReason:    behavior.disabledReason,
		ChangeCommand:     behavior.changeCommand,
		HorizontalPolicy:  behavior.horizontalPolicy,
		VerticalPolicy:    behavior.verticalPolicy,
		HorizontalVisible: geometry.horizontalVisible,
		VerticalVisible:   geometry.verticalVisible,
	}
	if behavior.content != nil && !behavior.content.destroyed {
		details.Content = behavior.content.id
		details.ContentKey = behavior.content.automationKey
	}
	if geometry.horizontalVisible {
		bar := scrollBarDetails(geometry.horizontalBar, scrollBarBehavior{
			orientation: Horizontal,
			state: ScrollBarState{
				ContentSize:  state.ContentSize.Width,
				ViewportSize: geometry.viewport.Width,
				Offset:       state.Offset.X,
			},
			arrowStep:      behavior.arrowStep.Width,
			pageStep:       details.PageStep.Width,
			disabled:       behavior.disabled,
			disabledReason: behavior.disabledReason,
			changeCommand:  behavior.changeCommand,
		})
		// Shared enabled policy, reason, and notification identity live once
		// on ScrollableDetails rather than being repeated in integrated bars.
		bar.DisabledReason = ""
		bar.ChangeCommand = ""
		details.HorizontalBar = &bar
	}
	if geometry.verticalVisible {
		bar := scrollBarDetails(geometry.verticalBar, scrollBarBehavior{
			orientation: Vertical,
			state: ScrollBarState{
				ContentSize:  state.ContentSize.Height,
				ViewportSize: geometry.viewport.Height,
				Offset:       state.Offset.Y,
			},
			arrowStep:      behavior.arrowStep.Height,
			pageStep:       details.PageStep.Height,
			disabled:       behavior.disabled,
			disabledReason: behavior.disabledReason,
			changeCommand:  behavior.changeCommand,
		})
		bar.DisabledReason = ""
		bar.ChangeCommand = ""
		details.VerticalBar = &bar
	}
	return details
}

// Content returns the toolkit-managed direct child Panel.
func (v *Viewport) Content() *Panel { return scrollViewContent(v) }

// Content returns the toolkit-managed direct child Panel.
func (s *ScrollablePanel) Content() *Panel { return scrollViewContent(s) }

func scrollViewContent(control Control) *Panel {
	if control == nil || control.controlState() == nil {
		return nil
	}
	state := control.controlState()
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(scrollViewBehavior)
	if !ok || behavior.content == nil || behavior.content.destroyed {
		return nil
	}
	panel, _ := behavior.content.control.(*Panel)
	return panel
}

// State returns an independent canonical viewport state.
func (v *Viewport) State() ViewportState { return scrollViewState(v) }

// State returns an independent canonical viewport state.
func (s *ScrollablePanel) State() ViewportState { return scrollViewState(s) }

func scrollViewState(control Control) ViewportState {
	if control == nil || control.controlState() == nil {
		return ViewportState{}
	}
	state := control.controlState()
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(scrollViewBehavior)
	if !ok || state.destroyed || state.aborted {
		return ViewportState{}
	}
	return behavior.state
}

// SetState applies one complete copied Viewport state.
func (v *Viewport) SetState(state ViewportState) error {
	return v.Update(context.Background(), state)
}

// SetState applies one complete copied ScrollablePanel state.
func (s *ScrollablePanel) SetState(state ViewportState) error {
	return s.Update(context.Background(), state)
}

// Update applies one Viewport state while observing ctx.
func (v *Viewport) Update(ctx context.Context, state ViewportState) error {
	return updateScrollView(ctx, v, state)
}

// Update applies one ScrollablePanel state while observing ctx.
func (s *ScrollablePanel) Update(ctx context.Context, state ViewportState) error {
	return updateScrollView(ctx, s, state)
}

func updateScrollView(
	ctx context.Context,
	control Control,
	state ViewportState,
) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := transaction.SetViewportState(control, state); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// EnsureVisible minimally moves the Viewport to reveal one Content rectangle.
func (v *Viewport) EnsureVisible(rect Rect) error {
	return ensureScrollViewVisible(v, rect)
}

// EnsureVisible minimally moves the ScrollablePanel to reveal one Content
// rectangle.
func (s *ScrollablePanel) EnsureVisible(rect Rect) error {
	return ensureScrollViewVisible(s, rect)
}

func ensureScrollViewVisible(control Control, rect Rect) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := transaction.EnsureViewportVisible(control, rect); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus gives this eligible Viewport keyboard focus.
func (v *Viewport) Focus() error { return focusSelectionControl(v) }

// Focus gives this eligible ScrollablePanel keyboard focus.
func (s *ScrollablePanel) Focus() error { return focusSelectionControl(s) }

// SetViewportState records one atomic complete viewport state replacement.
func (t *Transaction) SetViewportState(
	control Control,
	state ViewportState,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(scrollViewBehavior)
	if !ok {
		return ErrInvalidControl
	}
	normalized, err := normalizeViewportState(state)
	if err != nil {
		return err
	}
	behavior.state = clampViewportState(
		normalized,
		calculateScrollViewGeometry(target.bounds.Size(), behavior),
	)
	return t.recordScrollViewBehavior(target, behavior)
}

// EnsureViewportVisible records one minimal viewport movement.
func (t *Transaction) EnsureViewportVisible(
	control Control,
	rect Rect,
) error {
	target, err := t.control(control)
	if err != nil {
		return err
	}
	behavior, ok := t.selectedControlBehavior(target).(scrollViewBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if err := validateContentRect(rect, behavior.state.ContentSize); err != nil {
		return err
	}
	geometry := calculateScrollViewGeometry(target.bounds.Size(), behavior)
	behavior.state.Offset = ensureRectOffset(
		behavior.state.Offset,
		geometry.viewport.Size(),
		geometry.maximumOffset,
		rect,
	)
	return t.recordScrollViewBehavior(target, behavior)
}

func validateContentRect(rect Rect, content Size) error {
	if err := validateRect(rect); err != nil ||
		rect.Empty() ||
		rect.X < 0 ||
		rect.Y < 0 {
		return fmt.Errorf("%w: invalid viewport Content rectangle", ErrValidation)
	}
	right := int64(rect.X) + int64(rect.Width)
	bottom := int64(rect.Y) + int64(rect.Height)
	if right > int64(content.Width) || bottom > int64(content.Height) {
		return fmt.Errorf(
			"%w: viewport Content rectangle exceeds extent",
			ErrValidation,
		)
	}
	return nil
}

func ensureRectOffset(
	offset Point,
	viewport Size,
	maximum Point,
	rect Rect,
) Point {
	offset.X = ensureAxisOffset(
		offset.X,
		viewport.Width,
		maximum.X,
		rect.X,
		rect.Width,
	)
	offset.Y = ensureAxisOffset(
		offset.Y,
		viewport.Height,
		maximum.Y,
		rect.Y,
		rect.Height,
	)
	return offset
}

func ensureAxisOffset(offset, viewport, maximum, start, length int) int {
	if viewport <= 0 || length >= viewport {
		return min(maximum, max(0, start))
	}
	if start < offset {
		offset = start
	} else if start+length > offset+viewport {
		offset = start + length - viewport
	}
	return min(maximum, max(0, offset))
}

func (t *Transaction) recordScrollViewBehavior(
	state *controlState,
	behavior scrollViewBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationScrollView, state: state, behavior: behavior,
	})
	return nil
}

func scrollViewBehaviorEqual(left, right scrollViewBehavior) bool {
	return left.border.form == right.border.form &&
		left.border.borderStyle == right.border.borderStyle &&
		colorPointersEqual(left.border.foregroundOverride, right.border.foregroundOverride) &&
		colorPointersEqual(left.border.backgroundOverride, right.border.backgroundOverride) &&
		left.state == right.state &&
		left.arrowStep == right.arrowStep &&
		left.pageStep == right.pageStep &&
		left.disabled == right.disabled &&
		left.disabledReason == right.disabledReason &&
		left.changeCommand == right.changeCommand &&
		left.content == right.content &&
		left.horizontalPolicy == right.horizontalPolicy &&
		left.verticalPolicy == right.verticalPolicy &&
		left.integratedBars == right.integratedBars
}

func colorPointersEqual(left, right *Color) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func scrollViewCanMove(state *controlState, behavior scrollViewBehavior) bool {
	if state == nil || behavior.disabled {
		return false
	}
	maximum := calculateScrollViewGeometry(
		state.bounds.Size(),
		behavior,
	).maximumOffset
	return maximum.X > 0 || maximum.Y > 0
}

func (a *App) scrollViewKeyLocked(
	control *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if control == nil {
		return "", "", false, false
	}
	behavior, ok := control.behavior.(scrollViewBehavior)
	if !ok || behavior.disabled {
		return "", "", false, false
	}
	geometry := calculateScrollViewGeometry(control.bounds.Size(), behavior)
	next := behavior.state.Offset
	handled := true
	switch key {
	case KeyLeft:
		next.X -= behavior.arrowStep.Width
	case KeyRight:
		next.X += behavior.arrowStep.Width
	case KeyUp:
		next.Y -= behavior.arrowStep.Height
	case KeyDown:
		next.Y += behavior.arrowStep.Height
	case KeyPageUp:
		next.Y -= effectiveScrollPageStep(
			behavior.pageStep.Height,
			geometry.viewport.Height,
		)
	case KeyPageDown:
		next.Y += effectiveScrollPageStep(
			behavior.pageStep.Height,
			geometry.viewport.Height,
		)
	case KeyHome:
		next = Point{}
	case KeyEnd:
		next = geometry.maximumOffset
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	next.X = min(geometry.maximumOffset.X, max(0, next.X))
	next.Y = min(geometry.maximumOffset.Y, max(0, next.Y))
	if next == behavior.state.Offset {
		return "", "", true, false
	}
	behavior.state.Offset = next
	control.behavior = behavior
	reconcileScrollViewLocked(control)
	return behavior.changeCommand, control.id, true, true
}

func (a *App) ensureFocusedControlVisibleLocked() bool {
	target := a.focus
	if target == nil {
		return false
	}
	changed := false
	for ancestor := target.parent; ancestor != nil; ancestor = ancestor.parent {
		behavior, ok := ancestor.behavior.(scrollViewBehavior)
		if !ok || behavior.content == nil {
			continue
		}
		targetBounds := a.controlAbsoluteLocked(target)
		ownerBounds := a.controlAbsoluteLocked(ancestor)
		geometry := calculateScrollViewGeometry(ancestor.bounds.Size(), behavior)
		viewport := translatedRect(
			geometry.viewport,
			ownerBounds.X,
			ownerBounds.Y,
		)
		next := behavior.state.Offset
		if targetBounds.X < viewport.X {
			next.X += targetBounds.X - viewport.X
		} else if targetBounds.X+targetBounds.Width >
			viewport.X+viewport.Width {
			next.X += targetBounds.X + targetBounds.Width -
				(viewport.X + viewport.Width)
		}
		if targetBounds.Y < viewport.Y {
			next.Y += targetBounds.Y - viewport.Y
		} else if targetBounds.Y+targetBounds.Height >
			viewport.Y+viewport.Height {
			next.Y += targetBounds.Y + targetBounds.Height -
				(viewport.Y + viewport.Height)
		}
		next.X = min(geometry.maximumOffset.X, max(0, next.X))
		next.Y = min(geometry.maximumOffset.Y, max(0, next.Y))
		if next == behavior.state.Offset {
			continue
		}
		behavior.state.Offset = next
		ancestor.behavior = behavior
		reconcileScrollViewLocked(ancestor)
		changed = true
	}
	return changed
}
