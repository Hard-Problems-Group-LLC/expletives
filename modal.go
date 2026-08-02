package expletives

import (
	"context"
	"fmt"
)

const (
	// MaxModalDepth bounds one App's explicit LIFO modal stack.
	MaxModalDepth      = 8
	modalMinimumWidth  = 4
	modalMinimumHeight = 3
)

// Modal is the common lifecycle and container capability of modal controls.
// The unexported method prevents forged modal identity; external compounds
// can embed a real toolkit modal.
type Modal interface {
	Container
	Show(initialFocus Control) error
	Close(ModalResult) error
	Active() bool
	Result() (ModalResult, bool)
	Done() <-chan struct{}

	modalControlState() *controlState
}

// ModalPanelOptions configures one root-owned ModalPanel. Bounds X and Y must
// be zero; width and height are requested centered body dimensions, with zero
// selecting the recursively measured minimum on that axis.
type ModalPanelOptions struct {
	PanelOptions
	Title            string
	BorderStyle      StyleID
	BorderForm       BorderForm
	BorderForeground *Color
	BorderBackground *Color
	Shadow           ModalShadowPolicy
	ShadowStyle      StyleID
	NestedOwner      Modal
}

// ModalPanel is a copy-safe root-owned modal container handle.
type ModalPanel struct{ containerHandle }

type modalLifecycleState struct {
	done chan struct{}
}

type modalPanelBehavior struct {
	border          borderBehavior
	requested       Size
	shadow          ModalShadowPolicy
	shadowStyle     StyleID
	nestedOwner     *controlState
	lifecycle       ModalLifecycle
	active          bool
	top             bool
	stackIndex      int
	stackDepth      int
	savedFocus      *controlState
	initialFocus    *controlState
	resolvedBounds  Rect
	requiredMinimum Size
	degraded        bool
	result          *ModalResult
	escapeCommand   CommandID
	acceptEditor    *controlState
	inputInitial    normalizedInputText
	inputInitialSet bool
	progress        *progressDialogLifecycle
	progressStatus  *controlState
	progressBar     *controlState
	progressCancel  *controlState
	life            *modalLifecycleState
}

// NewModalPanel constructs one inactive ModalPanel as a direct root child.
// Show performs the one-shot modal presentation transition.
func NewModalPanel(
	parent Container,
	options ModalPanelOptions,
) (*ModalPanel, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	modal, err := tx.NewModalPanel(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return modal, nil
}

// NewModalPanel records construction of one provisional inactive ModalPanel.
func (t *Transaction) NewModalPanel(
	parent Container,
	options ModalPanelOptions,
) (*ModalPanel, error) {
	return t.newModalPanel(parent, options, ControlModalPanel)
}

func (t *Transaction) newModalPanel(
	parent Container,
	options ModalPanelOptions,
	kind ControlKind,
) (*ModalPanel, error) {
	if options.Bounds.X != 0 || options.Bounds.Y != 0 {
		return nil, fmt.Errorf(
			"%w: ModalPanel position is resolved by its App",
			ErrInvalidGeometry,
		)
	}
	if options.Hidden {
		return nil, fmt.Errorf(
			"%w: ModalPanel visibility is lifecycle-owned",
			ErrInvalidControl,
		)
	}
	if parent == nil || parent.containerState() == nil ||
		parent.containerState().app == nil ||
		parent.containerState() != parent.containerState().app.root.state {
		return nil, fmt.Errorf(
			"%w: ModalPanel requires App.Root as parent",
			ErrInvalidParent,
		)
	}
	if err := validateRect(options.Bounds); err != nil {
		return nil, err
	}
	shadow, err := normalizeModalShadow(options.Shadow)
	if err != nil {
		return nil, err
	}
	shadowStyle, err := normalizeStyleID(
		options.ShadowStyle,
		StyleID(string(kind)+".shadow"),
	)
	if err != nil {
		return nil, err
	}
	form := options.BorderForm
	if form == BorderDefault {
		form = BorderDouble
	}
	border, err := newBorderBehavior(
		options.Title,
		options.BorderStyle,
		kind,
		form,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return nil, err
	}
	var nestedOwner *controlState
	if options.NestedOwner != nil {
		nestedOwner = options.NestedOwner.modalControlState()
		if nestedOwner == nil || nestedOwner.app != parent.containerState().app {
			return nil, fmt.Errorf("%w: invalid nested modal owner", ErrInvalidControl)
		}
	}
	requested := options.Bounds.Size()
	options.Bounds.X, options.Bounds.Y = 0, 0
	options.MinimumSize.Width = max(options.MinimumSize.Width, modalMinimumWidth)
	options.MinimumSize.Height = max(options.MinimumSize.Height, modalMinimumHeight)
	options.Hidden = true
	behavior := modalPanelBehavior{
		border:      border,
		requested:   requested,
		shadow:      shadow,
		shadowStyle: shadowStyle,
		nestedOwner: nestedOwner,
		lifecycle:   ModalLifecycleInactive,
		stackIndex:  -1,
		life:        &modalLifecycleState{done: make(chan struct{})},
	}
	panel, err := t.newControl(
		parent,
		options.PanelOptions,
		kind,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	modal := &ModalPanel{containerHandle: panel.containerHandle}
	panel.state.control = modal
	panel.state.container = modal
	return modal, nil
}

func isModalKind(kind ControlKind) bool {
	switch kind {
	case ControlModalPanel, ControlDialog, ControlMessageBox, ControlConfirmDialog,
		ControlInputDialog, ControlProgressDialog, ControlFilePickerDialog,
		ControlMultiFilePickerDialog, ControlDirectoryPickerDialog:
		return true
	default:
		return false
	}
}

func normalizeModalShadow(policy ModalShadowPolicy) (ModalShadowPolicy, error) {
	switch policy {
	case ModalShadowDefault, ModalShadowTurbo:
		return ModalShadowTurbo, nil
	case ModalShadowNone:
		return ModalShadowNone, nil
	default:
		return "", fmt.Errorf("%w: invalid ModalPanel shadow", ErrInvalidControl)
	}
}

func (m *ModalPanel) modalControlState() *controlState {
	if m == nil {
		return nil
	}
	return m.state
}

// Show performs the ModalPanel's one allowed presentation transition.
func (m *ModalPanel) Show(initialFocus Control) error {
	state := m.modalControlState()
	if state == nil || state.app == nil {
		return ErrInvalidControl
	}
	app := state.app
	if err := app.beginMutation(context.Background()); err != nil {
		return err
	}
	defer app.endMutation()
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.final {
		return ErrClosed
	}
	behavior, ok := state.behavior.(modalPanelBehavior)
	if !ok || state.provisional || state.aborted {
		return ErrInvalidControl
	}
	if state.destroyed {
		return ErrDestroyed
	}
	if behavior.lifecycle != ModalLifecycleInactive || state.visible {
		return ErrModalState
	}
	if len(app.modals) >= MaxModalDepth {
		return ErrModalCapacity
	}
	if behavior.nestedOwner == nil {
		if len(app.modals) != 0 {
			return fmt.Errorf("%w: unowned nested modal", ErrModalState)
		}
	} else if len(app.modals) == 0 ||
		app.modals[len(app.modals)-1] != behavior.nestedOwner {
		return fmt.Errorf("%w: nested owner is not top modal", ErrModalState)
	}
	var focusState *controlState
	if initialFocus != nil {
		focusState = initialFocus.controlState()
		if !app.modalInitialFocusEligibleLocked(state, focusState) {
			return ErrNotFocusable
		}
	}
	if app.menu != nil {
		app.closeMenuLocked()
	}
	app.cancelOtherPopupCollectionsLocked(nil)
	app.commitOrCancelEditorStateLocked(app.focus)
	behavior.lifecycle = ModalLifecycleActive
	behavior.active = true
	behavior.savedFocus = app.focus
	behavior.initialFocus = focusState
	state.visible = true
	state.behavior = behavior
	app.modals = append(app.modals, state)
	app.focus = nil
	app.clearInvalidPressesLocked()
	arrangeAllLayoutsLocked(app)
	if focusState != nil && app.focusEligibleLocked(focusState) {
		app.focus = focusState
	} else {
		app.ensureFocusLocked()
	}
	app.beginModalEditorLocked(state)
	app.ensureFocusedControlVisibleLocked()
	app.publishLocked(nil)
	return nil
}

// Close records one explicit result and closes the active top ModalPanel.
func (m *ModalPanel) Close(result ModalResult) error {
	if err := validateModalResult(result); err != nil {
		return err
	}
	state := m.modalControlState()
	if state == nil || state.app == nil {
		return ErrInvalidControl
	}
	app := state.app
	if err := app.beginMutation(context.Background()); err != nil {
		return err
	}
	defer app.endMutation()
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.final {
		return ErrClosed
	}
	if state.destroyed {
		return ErrDestroyed
	}
	if len(app.modals) == 0 || app.modals[len(app.modals)-1] != state {
		return ErrModalState
	}
	app.closeModalRangeLocked(len(app.modals)-1, result)
	arrangeAllLayoutsLocked(app)
	app.ensureFocusedControlVisibleLocked()
	app.publishLocked(nil)
	return nil
}

func validateModalResult(result ModalResult) error {
	switch result.Reason {
	case ModalAccepted, ModalCancelled, ModalBack, ModalInterrupted,
		ModalQuit, ModalFailed, ModalDismissed, ModalDestroyed:
	default:
		return fmt.Errorf("%w: invalid Modal result reason", ErrInvalidRequest)
	}
	if result.Action != "" && !validBoundedIdentifier(string(result.Action)) {
		return fmt.Errorf("%w: invalid Modal result action", ErrInvalidRequest)
	}
	return nil
}

// Active reports whether this modal is currently on its App's modal stack.
func (m *ModalPanel) Active() bool {
	state := m.modalControlState()
	if state == nil || state.app == nil {
		return false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(modalPanelBehavior)
	return ok && !state.aborted && !state.destroyed && behavior.active
}

// Result returns a copied terminal result after the one-shot lifecycle ends.
func (m *ModalPanel) Result() (ModalResult, bool) {
	state := m.modalControlState()
	if state == nil || state.app == nil {
		return ModalResult{}, false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(modalPanelBehavior)
	if !ok || behavior.result == nil {
		return ModalResult{}, false
	}
	return *behavior.result, true
}

// Done returns the stable channel closed by the terminal lifecycle transition.
func (m *ModalPanel) Done() <-chan struct{} {
	state := m.modalControlState()
	if state == nil || state.app == nil {
		return nil
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(modalPanelBehavior)
	if !ok || behavior.life == nil {
		return nil
	}
	return behavior.life.done
}

func (b modalPanelBehavior) clientInset() int {
	return b.border.clientInset()
}

func (b modalPanelBehavior) controlBorder() borderBehavior { return b.border }

func (b modalPanelBehavior) additionalStyles() []StyleID {
	if b.shadow == ModalShadowNone {
		return nil
	}
	return []StyleID{b.shadowStyle}
}

func (b modalPanelBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	if b.shadow == ModalShadowTurbo {
		app.paintModalShadowLocked(frame, state, absolute, b.shadowStyle)
	}
	b.border.paintDecoration(app, frame, state, absolute, clip)
}

func (b modalPanelBehavior) details() ControlDetails {
	borderDetails := b.border.details()
	details := ModalPanelDetails{
		Lifecycle:       b.lifecycle,
		Active:          b.active,
		Top:             b.top,
		StackIndex:      b.stackIndex,
		StackDepth:      b.stackDepth,
		RequestedSize:   b.requested,
		ResolvedBounds:  b.resolvedBounds,
		RequiredMinimum: b.requiredMinimum,
		Degraded:        b.degraded,
		Shadow:          b.shadow,
		ShadowStyle:     b.shadowStyle,
	}
	if b.nestedOwner != nil {
		details.NestedOwner = b.nestedOwner.id
	}
	if b.savedFocus != nil {
		details.SavedFocus = b.savedFocus.id
	}
	if b.initialFocus != nil {
		details.InitialFocus = b.initialFocus.id
	}
	if b.result != nil {
		result := *b.result
		details.Result = &result
	}
	borderDetails.ModalPanel = &details
	if b.progress != nil {
		progressDetails := &ProgressDialogDetails{
			Cancellable:     b.progressCancel != nil,
			CancelRequested: b.progress.cancelRequested.Load(),
		}
		if b.progressStatus != nil {
			if status, ok := b.progressStatus.behavior.(textBehavior); ok {
				progressDetails.StatusLength = status.content.cells
			}
		}
		if b.progressBar != nil {
			if progress, ok := b.progressBar.behavior.(progressBehavior); ok &&
				progress.progressBar != nil {
				progressDetails.Progress = *progress.progressBar
			}
		}
		borderDetails.ProgressDialog = progressDetails
	}
	return borderDetails
}

func (a *App) modalInitialFocusEligibleLocked(
	modal *controlState,
	target *controlState,
) bool {
	if modal == nil || target == nil || target.app != a ||
		target.destroyed || target.aborted ||
		!controlWithin(target, modal) || target == modal ||
		!a.focusBehaviorEligibleLocked(target, target.behavior) {
		return false
	}
	for current := target; current != nil && current != modal; current = current.parent {
		if current.destroyed || current.aborted || !current.visible {
			return false
		}
		if managed, selected := a.isManagedTabPageLocked(current); managed && !selected {
			return false
		}
	}
	return true
}

func controlWithin(state, ancestor *controlState) bool {
	for current := state; current != nil; current = current.parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func modalAncestorState(state *controlState) *controlState {
	for current := state; current != nil; current = current.parent {
		if isModalKind(current.kind) {
			return current
		}
	}
	return nil
}

func (a *App) topModalLocked() *controlState {
	if len(a.modals) == 0 {
		return nil
	}
	return a.modals[len(a.modals)-1]
}

func (a *App) inActiveModalScopeLocked(state *controlState) bool {
	top := a.topModalLocked()
	return top == nil || controlWithin(state, top)
}

func (a *App) closeModalRangeLocked(index int, result ModalResult) {
	if index < 0 || index >= len(a.modals) {
		return
	}
	baseBehavior, _ := a.modals[index].behavior.(modalPanelBehavior)
	savedFocus := baseBehavior.savedFocus
	if a.focus != nil && controlWithin(a.focus, a.modals[index]) {
		a.commitOrCancelEditorStateLocked(a.focus)
	}
	for current := len(a.modals) - 1; current >= index; current-- {
		state := a.modals[current]
		behavior, ok := state.behavior.(modalPanelBehavior)
		if !ok || behavior.lifecycle != ModalLifecycleActive {
			continue
		}
		copied := result
		behavior.result = &copied
		behavior.lifecycle = ModalLifecycleClosed
		behavior.active = false
		behavior.top = false
		behavior.stackIndex = -1
		behavior.stackDepth = 0
		state.visible = false
		state.behavior = behavior
		if behavior.progress != nil {
			behavior.progress.stop()
		}
		if behavior.life != nil {
			close(behavior.life.done)
		}
	}
	a.modals = a.modals[:index]
	a.focus = nil
	a.clearInvalidPressesLocked()
	a.reconcileModalsLocked()
	if savedFocus != nil && a.focusEligibleLocked(savedFocus) {
		a.focus = savedFocus
	} else {
		a.ensureFocusLocked()
	}
}

func (a *App) destroyModalStateLocked(state *controlState) {
	behavior, ok := state.behavior.(modalPanelBehavior)
	if !ok || behavior.lifecycle == ModalLifecycleClosed {
		return
	}
	for index, modal := range a.modals {
		if modal == state {
			a.closeModalRangeLocked(index, ModalResult{Reason: ModalDestroyed})
			return
		}
	}
	result := ModalResult{Reason: ModalDestroyed}
	behavior.result = &result
	behavior.lifecycle = ModalLifecycleClosed
	behavior.active = false
	behavior.top = false
	behavior.stackIndex = -1
	behavior.stackDepth = 0
	state.visible = false
	state.behavior = behavior
	if behavior.progress != nil {
		behavior.progress.stop()
	}
	if behavior.life != nil {
		close(behavior.life.done)
	}
}

func abortModalLifecycleLocked(state *controlState) {
	behavior, ok := state.behavior.(modalPanelBehavior)
	if !ok || behavior.lifecycle == ModalLifecycleClosed {
		return
	}
	result := ModalResult{Reason: ModalDestroyed}
	behavior.result = &result
	behavior.lifecycle = ModalLifecycleClosed
	state.behavior = behavior
	if behavior.progress != nil {
		behavior.progress.stop()
	}
	if behavior.life != nil {
		close(behavior.life.done)
	}
}

func (a *App) reconcileModalsLocked() {
	client := a.rootContentRectLocked()
	for index, state := range a.modals {
		behavior, ok := state.behavior.(modalPanelBehavior)
		if !ok || state.destroyed {
			continue
		}
		required := modalRequiredMinimumLocked(state, behavior)
		desired := behavior.requested
		if desired.Width == 0 || desired.Width < required.Width {
			desired.Width = required.Width
		}
		if desired.Height == 0 || desired.Height < required.Height {
			desired.Height = required.Height
		}
		width := min(desired.Width, client.Width)
		height := min(desired.Height, client.Height)
		bounds := Rect{
			X:      max(0, (client.Width-width)/2),
			Y:      max(0, (client.Height-height)/2),
			Width:  max(0, width),
			Height: max(0, height),
		}
		state.bounds = bounds
		behavior.active = true
		behavior.top = index == len(a.modals)-1
		behavior.stackIndex = index
		behavior.stackDepth = len(a.modals)
		behavior.resolvedBounds = bounds
		behavior.requiredMinimum = required
		behavior.degraded = bounds.Width < desired.Width || bounds.Height < desired.Height
		state.behavior = behavior
	}
}

func modalRequiredMinimumLocked(
	state *controlState,
	behavior modalPanelBehavior,
) Size {
	required := state.minimumSize
	inset := behavior.border.clientInset()
	for _, layout := range state.layoutRoots {
		minimum := layout.minimum
		required.Width = max(required.Width, minimum.Width+2*inset)
		required.Height = max(required.Height, minimum.Height+2*inset)
	}
	for _, child := range state.children {
		if child.destroyed || child.layout != nil {
			continue
		}
		required.Width = max(
			required.Width,
			child.bounds.X+max(child.bounds.Width, child.minimumSize.Width)+2*inset,
		)
		required.Height = max(
			required.Height,
			child.bounds.Y+max(child.bounds.Height, child.minimumSize.Height)+2*inset,
		)
	}
	return required
}

func (a *App) paintModalShadowLocked(
	frame *IntendedFrame,
	state *controlState,
	body Rect,
	style StyleID,
) {
	if body.Empty() {
		return
	}
	clip := a.rootContentRectLocked()
	right := Rect{
		X:      body.X + body.Width,
		Y:      body.Y + 1,
		Width:  2,
		Height: body.Height,
	}.Intersect(clip)
	bottom := Rect{
		X:      body.X + 2,
		Y:      body.Y + body.Height,
		Width:  body.Width,
		Height: 1,
	}.Intersect(clip)
	a.fillStyleLocked(frame, right, style, state.id)
	a.fillStyleLocked(frame, bottom, style, state.id)
}
