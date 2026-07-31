package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
)

// PanelOptions configures one ordinary Panel.
type PanelOptions struct {
	// AutomationKey is an optional stable application-selected lookup key.
	AutomationKey string
	// Bounds is parent-client-relative logical geometry. Its zero value is an
	// empty rectangle.
	Bounds Rect
	// MinimumSize is the nonnegative declared logical minimum.
	MinimumSize Size
	// Style is resolved through the App Theme. Empty selects the control-kind
	// default.
	Style StyleID
	// Hidden constructs the control and its descendants without painting them.
	Hidden bool
}

// FrameOptions configures a bordered Frame control.
type FrameOptions struct {
	PanelOptions
	// Title is normalized to bounded one-cell graphemes for painting.
	Title string
	// BorderStyle defaults to "frame.border".
	BorderStyle StyleID
	// BorderForm defaults to BorderSingle. BorderNone removes both decoration
	// and the one-cell client inset.
	BorderForm BorderForm
	// BorderForeground and BorderBackground optionally override the
	// corresponding component of the Theme-resolved BorderStyle.
	BorderForeground *Color
	BorderBackground *Color
}

// GroupBoxOptions configures a bordered GroupBox control.
type GroupBoxOptions struct {
	PanelOptions
	// Title is normalized to bounded one-cell graphemes for painting.
	Title string
	// BorderStyle defaults to "group_box.border".
	BorderStyle StyleID
	// BorderForm defaults to BorderSingle. BorderNone removes both decoration
	// and the one-cell client inset.
	BorderForm BorderForm
	// BorderForeground and BorderBackground optionally override the
	// corresponding component of the Theme-resolved BorderStyle.
	BorderForeground *Color
	BorderBackground *Color
}

// Control is the common read-only identity and geometry capability of a
// toolkit-owned control. The unexported method prevents forged node identity;
// external compound controls can embed an actual toolkit control. All handles
// remain owned by their App and are safe for concurrent use.
type Control interface {
	ID() ControlID
	AutomationKey() string
	Bounds() Rect
	MinimumSize() Size
	Visible() bool
	Style() StyleID

	controlState() *controlState
}

// Container is a Control that may be supplied as a construction parent.
// Future leaf controls implement Control without implementing Container.
// Children returns a caller-owned slice of App-owned handles.
type Container interface {
	Control
	Children() []Control

	containerState() *controlState
}

type containerHandle struct {
	state *controlState
}

// Panel is the foundational container control. It is a copy-safe handle over
// canonical toolkit state: copying a Panel value aliases the same control.
type Panel struct {
	containerHandle
}

// Frame is one bordered copy-safe container handle. Copying a Frame value
// aliases the same canonical toolkit control.
type Frame struct {
	containerHandle
}

// GroupBox is one titled bordered copy-safe container handle. Copying a
// GroupBox value aliases the same canonical toolkit control.
type GroupBox struct {
	containerHandle
}

type controlState struct {
	app *App

	control       Control
	container     Container
	id            ControlID
	automationKey string
	kind          ControlKind
	parent        *controlState
	children      []*controlState
	layoutRoots   []*layoutState
	layout        *layoutState
	bounds        Rect
	minimumSize   Size
	style         StyleID
	visible       bool
	root          bool
	destroyed     bool
	provisional   bool
	aborted       bool
	behavior      controlBehavior
}

type controlBehavior interface {
	clientInset() int
	paintDecoration(
		app *App,
		frame *IntendedFrame,
		state *controlState,
		absolute Rect,
		clip Rect,
	)
	details() ControlDetails
}

type plainBehavior struct{}

func (plainBehavior) clientInset() int {
	return 0
}

func (plainBehavior) paintDecoration(
	*App,
	*IntendedFrame,
	*controlState,
	Rect,
	Rect,
) {
}

func (plainBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		Container: &ContainerDetails{},
	}
}

type borderBehavior struct {
	title              string
	titleCells         []string
	borderStyle        StyleID
	form               BorderForm
	foregroundOverride *Color
	backgroundOverride *Color
}

func (b borderBehavior) clientInset() int {
	if b.form == BorderNone {
		return 0
	}
	return 1
}

func (b borderBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	app.paintBorderLocked(frame, absolute, clip, state, b)
}

func (b borderBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		Container: &ContainerDetails{ClientInset: b.clientInset()},
		Border: &BorderDetails{
			Title:              b.title,
			Form:               b.form,
			Style:              b.borderStyle,
			ForegroundOverride: cloneColor(b.foregroundOverride),
			BackgroundOverride: cloneColor(b.backgroundOverride),
		},
	}
}

// NewPanel constructs and atomically inserts a Panel under parent. It uses a
// one-operation transaction with a DefaultMutationWait bound; use
// Transaction.NewPanel and Commit for cancellation or batching.
func NewPanel(parent Container, options PanelOptions) (*Panel, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	panel, err := tx.NewPanel(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return panel, nil
}

// NewFrame constructs one bordered Frame under parent. It uses a one-operation
// transaction with a DefaultMutationWait bound; use Transaction.NewFrame and
// Commit for cancellation or batching.
func NewFrame(parent Container, options FrameOptions) (*Frame, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	frame, err := tx.NewFrame(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return frame, nil
}

// NewGroupBox constructs one titled bordered GroupBox under parent. It uses a
// one-operation transaction with a DefaultMutationWait bound; use
// Transaction.NewGroupBox and Commit for cancellation or batching.
func NewGroupBox(parent Container, options GroupBoxOptions) (*GroupBox, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	group, err := tx.NewGroupBox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return group, nil
}

func newBorderBehavior(
	title string,
	style StyleID,
	kind ControlKind,
	form BorderForm,
	foregroundOverride *Color,
	backgroundOverride *Color,
) (borderBehavior, error) {
	if len(title) > MaxTitleBytes {
		return borderBehavior{}, fmt.Errorf(
			"%w: title exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTitleBytes,
		)
	}
	cells := display.Normalize(title)
	if len(cells) > MaxTitleCells {
		return borderBehavior{}, fmt.Errorf(
			"%w: title exceeds %d display cells",
			ErrTextLimit,
			MaxTitleCells,
		)
	}
	for _, cell := range cells {
		if len(cell) > MaxCellBytes {
			return borderBehavior{}, fmt.Errorf(
				"%w: one title cell exceeds %d UTF-8 bytes",
				ErrTextLimit,
				MaxCellBytes,
			)
		}
	}
	normalizedTitle := strings.Join(cells, "")
	if len(normalizedTitle) > MaxTitleBytes {
		return borderBehavior{}, fmt.Errorf(
			"%w: normalized title exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTitleBytes,
		)
	}
	normalized, err := normalizeStyleID(
		style,
		StyleID(string(kind)+".border"),
	)
	if err != nil {
		return borderBehavior{}, err
	}
	form, err = normalizeBorderForm(form, BorderSingle)
	if err != nil {
		return borderBehavior{}, err
	}
	return borderBehavior{
		title:              normalizedTitle,
		titleCells:         append([]string(nil), cells...),
		borderStyle:        normalized,
		form:               form,
		foregroundOverride: cloneColor(foregroundOverride),
		backgroundOverride: cloneColor(backgroundOverride),
	}, nil
}

func normalizeBorderForm(form, defaultForm BorderForm) (BorderForm, error) {
	if form == BorderDefault {
		return defaultForm, nil
	}
	switch form {
	case BorderNone, BorderSingle, BorderDouble, BorderShadeLight,
		BorderShadeMedium, BorderShadeDark, BorderBlock:
		return form, nil
	default:
		return "", fmt.Errorf("%w: invalid border form %q", ErrInvalidControl, form)
	}
}

func cloneColor(color *Color) *Color {
	if color == nil {
		return nil
	}
	copied := *color
	return &copied
}

func resolveBorderStyle(base ResolvedStyle, behavior borderBehavior) ResolvedStyle {
	if behavior.foregroundOverride != nil {
		base.Foreground = *behavior.foregroundOverride
	}
	if behavior.backgroundOverride != nil {
		base.Background = *behavior.backgroundOverride
	}
	return base
}

func preparePanel(
	parent Container,
	options PanelOptions,
	kind ControlKind,
	behavior controlBehavior,
) (*Panel, error) {
	if parent == nil {
		return nil, fmt.Errorf(
			"%w: ordinary control requires a parent",
			ErrInvalidParent,
		)
	}
	parentState := parent.containerState()
	if parentState == nil || parentState.app == nil {
		return nil, fmt.Errorf(
			"%w: parent is not owned by an App",
			ErrInvalidParent,
		)
	}
	if err := validateRect(options.Bounds); err != nil {
		return nil, err
	}
	if err := validateMinimumSize(options.MinimumSize); err != nil {
		return nil, err
	}
	if options.AutomationKey != "" &&
		!validBoundedIdentifier(options.AutomationKey) {
		return nil, errors.New("expletives: automation key is invalid or too long")
	}
	style, err := normalizeStyleID(options.Style, StyleID(kind))
	if err != nil {
		return nil, err
	}
	if behavior == nil {
		return nil, errors.New("expletives: nil control behavior")
	}

	state := &controlState{
		app:           parentState.app,
		automationKey: options.AutomationKey,
		kind:          kind,
		parent:        parentState,
		bounds:        options.Bounds,
		minimumSize:   options.MinimumSize,
		style:         style,
		visible:       !options.Hidden,
		provisional:   true,
		behavior:      behavior,
	}
	panel := &Panel{containerHandle: containerHandle{state: state}}
	state.control = panel
	state.container = panel
	return panel, nil
}

func validateMinimumSize(size Size) error {
	if size.Width < 0 || size.Height < 0 ||
		size.Width > maxCoordinateMagnitude ||
		size.Height > maxCoordinateMagnitude {
		return fmt.Errorf("%w: minimum size exceeds checked geometry", ErrInvalidGeometry)
	}
	return nil
}

func (p *containerHandle) controlState() *controlState {
	if p == nil {
		return nil
	}
	return p.state
}

func (p *containerHandle) containerState() *controlState {
	return p.controlState()
}

// ID returns the stable runtime identity. It is empty before a transaction
// commits or after construction aborts, and remains readable after logical
// destruction.
func (p *containerHandle) ID() ControlID {
	if p == nil || p.state == nil || p.state.app == nil {
		return ""
	}
	p.state.app.mu.RLock()
	defer p.state.app.mu.RUnlock()
	if p.state.aborted {
		return ""
	}
	return p.state.id
}

// AutomationKey returns the optional caller-selected stable key. It remains
// readable after logical destruction.
func (p *containerHandle) AutomationKey() string {
	if p == nil || p.state == nil || p.state.app == nil {
		return ""
	}
	p.state.app.mu.RLock()
	defer p.state.app.mu.RUnlock()
	if p.state.aborted {
		return ""
	}
	return p.state.automationKey
}

// Parent returns the immutable App-owned container parent or nil for the App
// root or an aborted handle.
func (p *containerHandle) Parent() Container {
	if p == nil || p.state == nil || p.state.app == nil {
		return nil
	}
	p.state.app.mu.RLock()
	defer p.state.app.mu.RUnlock()
	if p.state.aborted || p.state.parent == nil {
		return nil
	}
	return p.state.parent.container
}

// Bounds returns the current parent-client-relative logical rectangle.
func (p *containerHandle) Bounds() Rect {
	state := p.controlState()
	if state == nil || state.app == nil {
		return Rect{}
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	if state.aborted {
		return Rect{}
	}
	return state.bounds
}

// MinimumSize returns the declared minimum size.
func (p *containerHandle) MinimumSize() Size {
	state := p.controlState()
	if state == nil || state.app == nil {
		return Size{}
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	if state.aborted {
		return Size{}
	}
	return state.minimumSize
}

// Visible reports the control's own active visibility. An invisible ancestor
// may still prevent a visible descendant from painting.
func (p *containerHandle) Visible() bool {
	state := p.controlState()
	if state == nil || state.app == nil {
		return false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	return state.visible && !state.destroyed && !state.aborted
}

// Style returns the semantic style ID.
func (p *containerHandle) Style() StyleID {
	state := p.controlState()
	if state == nil || state.app == nil {
		return ""
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	if state.aborted {
		return ""
	}
	return state.style
}

// Children returns a caller-owned slice of active App-owned handles in stable
// insertion order.
func (p *containerHandle) Children() []Control {
	state := p.controlState()
	if state == nil || state.app == nil {
		return nil
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	if state.destroyed || state.aborted {
		return nil
	}
	children := make([]Control, 0, len(state.children))
	for _, child := range state.children {
		if !child.destroyed {
			children = append(children, child.control)
		}
	}
	return children
}

// SetBounds changes an ordinary control's logical rectangle and publishes one
// complete snapshot when the value changed. It waits at most
// DefaultMutationWait.
func (p *containerHandle) SetBounds(bounds Rect) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	transaction := state.app.NewTransaction()
	if err := transaction.SetBounds(p, bounds); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetStyle changes the semantic style and publishes one complete snapshot when
// the value changed. The ID must exist in the App Theme. SetStyle waits at most
// DefaultMutationWait.
func (p *containerHandle) SetStyle(style StyleID) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	transaction := state.app.NewTransaction()
	if err := transaction.SetStyle(p, style); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetVisible changes painting and effective descendant visibility and
// publishes one complete snapshot when the value changed. It waits at most
// DefaultMutationWait.
func (p *containerHandle) SetVisible(visible bool) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	transaction := state.app.NewTransaction()
	if err := transaction.SetVisible(p, visible); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetMinimumSize changes the declared logical minimum and publishes one
// complete snapshot when the value changed. It waits at most
// DefaultMutationWait.
func (p *containerHandle) SetMinimumSize(size Size) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	transaction := state.app.NewTransaction()
	if err := transaction.SetMinimumSize(p, size); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Destroy logically removes this control and all descendants in one
// publication. The root cannot be destroyed independently of its App. The
// handle remains readable but later mutations return ErrDestroyed. Destroy
// waits at most DefaultMutationWait.
func (p *containerHandle) Destroy() error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	transaction := state.app.NewTransaction()
	if err := transaction.Destroy(p); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

func (p *containerHandle) mutableState() (*controlState, error) {
	if p == nil || p.state == nil || p.state.app == nil {
		return nil, ErrInvalidControl
	}
	return p.state, nil
}

func (s *controlState) mutableLocked() error {
	if s.app.final {
		return ErrClosed
	}
	if s.destroyed {
		return ErrDestroyed
	}
	if s.aborted {
		return ErrInvalidControl
	}
	if canonical, exists := s.app.controlsByID[s.id]; !exists ||
		canonical != s {
		return ErrInvalidControl
	}
	return nil
}

func (a *App) destroyStateLocked(state *controlState) {
	if state.layout != nil {
		removeLayoutPanelLocked(state)
	}
	for _, layout := range state.layoutRoots {
		destroyLayoutTreeLocked(a, layout)
	}
	state.layoutRoots = nil
	for _, child := range state.children {
		a.destroyStateLocked(child)
	}
	state.children = nil
	state.destroyed = true
	delete(a.controlsByID, state.id)
	if state.automationKey != "" {
		delete(a.controlsByKey, state.automationKey)
	}
}

func removeChildState(parent *controlState, target *controlState) {
	if parent == nil {
		return
	}
	for index, child := range parent.children {
		if child != target {
			continue
		}
		copy(parent.children[index:], parent.children[index+1:])
		parent.children[len(parent.children)-1] = nil
		parent.children = parent.children[:len(parent.children)-1]
		return
	}
}
