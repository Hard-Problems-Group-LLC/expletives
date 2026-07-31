package expletives

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultMutationWait bounds waiting to enter an App control-tree mutation.
const DefaultMutationWait = 250 * time.Millisecond

// Transaction records one atomic control-tree update outside App locks.
// Transactions are not safe for concurrent mutation and may be committed
// exactly once. Recorded caller values are copied; returned control handles
// remain provisional until Commit succeeds.
type Transaction struct {
	app         *App
	creates     []*controlState
	mutations   []transactionMutation
	destroys    []*controlState
	layouts     []transactionLayout
	stacks      []transactionStack
	size        *Size
	constraints *RootConstraints
	theme       *Theme
	focus       *controlState
	focusSet    bool
	consumed    bool
}

type transactionLayout struct {
	owner   *controlState
	layout  *layoutState
	primary bool
}

type transactionStack struct {
	control *controlState
	layout  *layoutState
	raise   bool
}

type transactionMutationKind uint8

const (
	mutationBounds transactionMutationKind = iota + 1
	mutationMinimum
	mutationStyle
	mutationVisible
	mutationText
	mutationStatusSegments
	mutationCheckState
	mutationRadioValue
	mutationChoiceValue
	mutationChoiceOptions
	mutationFocusGuidance
	mutationTextField
	mutationNumberField
	mutationTextArea
	mutationProgress
	mutationScrollBar
	mutationTabbedPanel
	mutationScrollView
)

type transactionMutation struct {
	kind     transactionMutationKind
	state    *controlState
	rect     Rect
	size     Size
	style    StyleID
	visible  bool
	text     string
	behavior controlBehavior

	selectionValue   string
	selectionOptions []SelectionOption
	focusGuidance    focusGuidanceConfig
}

// NewTransaction creates an empty App-scoped transaction builder. Building it
// does not acquire App locks or publish state.
func (a *App) NewTransaction() *Transaction {
	return &Transaction{app: a}
}

func transactionForParent(parent Container) (*Transaction, error) {
	if parent == nil || parent.containerState() == nil ||
		parent.containerState().app == nil {
		return nil, fmt.Errorf("%w: invalid construction parent", ErrInvalidParent)
	}
	return parent.containerState().app.NewTransaction(), nil
}

// NewPanel records construction of a Panel. Its handle becomes active when
// Commit succeeds and may parent later controls in this transaction. A failed
// Commit permanently invalidates the provisional handle.
func (t *Transaction) NewPanel(
	parent Container,
	options PanelOptions,
) (*Panel, error) {
	return t.newControl(parent, options, ControlPanel, containerBehavior{})
}

// NewFrame records construction of a bordered Frame. Its provisional handle
// follows the same lifetime rules as NewPanel.
func (t *Transaction) NewFrame(
	parent Container,
	options FrameOptions,
) (*Frame, error) {
	behavior, err := newBorderBehavior(
		options.Title,
		options.BorderStyle,
		ControlFrame,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return nil, err
	}
	panel, err := t.newControl(
		parent,
		options.PanelOptions,
		ControlFrame,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	frame := &Frame{containerHandle: panel.containerHandle}
	panel.state.control = frame
	panel.state.container = frame
	return frame, nil
}

// NewGroupBox records construction of a titled bordered GroupBox. Its
// provisional handle follows the same lifetime rules as NewPanel.
func (t *Transaction) NewGroupBox(
	parent Container,
	options GroupBoxOptions,
) (*GroupBox, error) {
	behavior, err := newBorderBehavior(
		options.Title,
		options.BorderStyle,
		ControlGroupBox,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return nil, err
	}
	panel, err := t.newControl(
		parent,
		options.PanelOptions,
		ControlGroupBox,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	group := &GroupBox{containerHandle: panel.containerHandle}
	panel.state.control = group
	panel.state.container = group
	return group, nil
}

func (t *Transaction) newControl(
	parent Container,
	options PanelOptions,
	kind ControlKind,
	behavior controlBehavior,
) (*Panel, error) {
	if err := t.usable(); err != nil {
		return nil, err
	}
	if err := t.reserveOperation(); err != nil {
		return nil, err
	}
	panel, err := preparePanel(parent, options, kind, behavior)
	if err != nil {
		return nil, err
	}
	if panel.state.app != t.app {
		return nil, fmt.Errorf("%w: parent belongs to another App", ErrInvalidParent)
	}
	t.creates = append(t.creates, panel.state)
	return panel, nil
}

// SetSize records an App surface change.
func (t *Transaction) SetSize(size Size) error {
	if err := t.usable(); err != nil {
		return err
	}
	if t.size == nil {
		if err := t.reserveOperation(); err != nil {
			return err
		}
	}
	if err := validateSize(size); err != nil {
		return err
	}
	copied := size
	t.size = &copied
	return nil
}

// SetRootConstraints records a centered root sizing-policy change.
func (t *Transaction) SetRootConstraints(constraints RootConstraints) error {
	if err := t.usable(); err != nil {
		return err
	}
	if t.constraints == nil {
		if err := t.reserveOperation(); err != nil {
			return err
		}
	}
	if err := validateRootConstraints(constraints); err != nil {
		return err
	}
	copied := constraints
	t.constraints = &copied
	return nil
}

// SetBounds records a control geometry change.
func (t *Transaction) SetBounds(control Control, bounds Rect) error {
	if err := validateRect(bounds); err != nil {
		return err
	}
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if state.root {
		return errors.New("expletives: root bounds follow App root constraints")
	}
	if isApplicationChrome(state.kind) {
		return errors.New(
			"expletives: application chrome bounds follow the application surface",
		)
	}
	if state.managedBy != nil {
		return errors.New("expletives: managed scroll Content bounds are derived")
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:  mutationBounds,
		state: state,
		rect:  bounds,
	})
	return nil
}

// SetMinimumSize records a declared-minimum change.
func (t *Transaction) SetMinimumSize(control Control, size Size) error {
	if err := validateMinimumSize(size); err != nil {
		return err
	}
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if state.root {
		return errors.New("expletives: use SetRootConstraints for the root minimum")
	}
	if isApplicationChrome(state.kind) {
		return errors.New("expletives: application chrome minimum is intrinsic")
	}
	if state.managedBy != nil {
		return errors.New("expletives: managed scroll Content minimum is derived")
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:  mutationMinimum,
		state: state,
		size:  size,
	})
	return nil
}

// SetText records a canonical text change for a Label, StaticText, Rule, or
// TextField, or TextArea. An editor replacement is a silent committed-value
// change that leaves edit mode.
func (t *Transaction) SetText(control Control, text string) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if state.kind == ControlTextField {
		value, err := normalizeInputText(text)
		if err != nil {
			return err
		}
		behavior, ok := t.recordedControlBehavior(state).(textFieldBehavior)
		if !ok {
			return ErrInvalidControl
		}
		if behavior.validator != nil &&
			behavior.validator.value.Enforcement == TextValidationHard &&
			!textCellsValid(value.cells, behavior.validator) {
			return fmt.Errorf(
				"%w: TextField value violates hard validation",
				ErrValidation,
			)
		}
		behavior.committed = value
		behavior.working = cloneInputText(value)
		behavior.editing = false
		behavior.caret = len(value.cells)
		behavior.viewOffset = 0
		behavior.selectionAnchor = -1
		return t.recordTextFieldBehavior(state, behavior)
	}
	if state.kind == ControlTextArea {
		value, err := normalizeTextAreaInput(text)
		if err != nil {
			return err
		}
		behavior, ok := t.recordedControlBehavior(state).(textAreaBehavior)
		if !ok {
			return ErrInvalidControl
		}
		if behavior.validator != nil &&
			behavior.validator.value.Enforcement == TextValidationHard &&
			!textAreaCellsValid(value.cells, behavior.validator) {
			return fmt.Errorf(
				"%w: TextArea value violates hard validation",
				ErrValidation,
			)
		}
		behavior.committed = value
		behavior.working = cloneInputText(value)
		behavior.editing = false
		behavior.caret = len(value.cells)
		behavior.selectionAnchor = -1
		behavior.rowOffset = 0
		behavior.columnOffset = 0
		behavior.preferredColumn = -1
		return t.recordTextAreaBehavior(state, behavior)
	}
	multiline := false
	switch state.kind {
	case ControlLabel, ControlRule:
	case ControlStaticText:
		multiline = true
	default:
		return fmt.Errorf("%w: control has no mutable text", ErrInvalidControl)
	}
	normalized, err := normalizeDisplayText(text, multiline)
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:  mutationText,
		state: state,
		text:  normalized.text,
	})
	return nil
}

// SetStyle records a semantic style-ID change.
func (t *Transaction) SetStyle(control Control, style StyleID) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeStyleID(style, StyleID(state.kind))
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:  mutationStyle,
		state: state,
		style: normalized,
	})
	return nil
}

// SetVisible records a visibility change.
func (t *Transaction) SetVisible(control Control, visible bool) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if state.root && !visible {
		return errors.New("expletives: root cannot be hidden")
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:    mutationVisible,
		state:   state,
		visible: visible,
	})
	return nil
}

// SetFocus records keyboard focus for one focusable control. Eligibility is
// revalidated atomically at Commit.
func (t *Transaction) SetFocus(control Control) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	switch state.kind {
	case ControlButton, ControlCheckbox, ControlRadioButton,
		ControlCycleField, ControlSelectField, ControlTextField,
		ControlNumberField, ControlSpinBox, ControlTextArea,
		ControlScrollBar, ControlTabbedPanel, ControlNotebook,
		ControlViewport, ControlScrollablePanel:
	default:
		return ErrNotFocusable
	}
	if !t.focusSet {
		if err := t.reserveOperation(); err != nil {
			return err
		}
	}
	t.focus = state
	t.focusSet = true
	return nil
}

// SetFocusGuidance records application-specific guidance for a control.
// Append retains the toolkit's generic control guidance; Override replaces it.
// An empty Text clears a prior customization.
func (t *Transaction) SetFocusGuidance(
	control Control,
	guidance FocusGuidance,
) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	config, err := normalizeFocusGuidance(state.kind, guidance)
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:          mutationFocusGuidance,
		state:         state,
		focusGuidance: config,
	})
	return nil
}

// Destroy records recursive logical destruction of a non-root control.
func (t *Transaction) Destroy(control Control) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if state.root {
		return errors.New("expletives: root cannot be destroyed")
	}
	if state.managedBy != nil {
		return errors.New(
			"expletives: managed scroll Content cannot be destroyed independently",
		)
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.destroys = append(t.destroys, state)
	return nil
}

// SetLayout records atomic attachment of an owner's first top-level Layout.
func (t *Transaction) SetLayout(owner Container, layout Layout) error {
	return t.attachLayout(owner, layout, true)
}

// AddLayout records atomic attachment of an additional top-level Layout.
func (t *Transaction) AddLayout(owner Container, layout Layout) error {
	return t.attachLayout(owner, layout, false)
}

func (t *Transaction) attachLayout(
	owner Container,
	layout Layout,
	primary bool,
) error {
	if err := t.usable(); err != nil {
		return err
	}
	if owner == nil || owner.containerState() == nil ||
		owner.containerState().app != t.app {
		return ErrInvalidParent
	}
	switch owner.containerState().kind {
	case ControlViewport, ControlScrollablePanel:
		return fmt.Errorf(
			"%w: scroll container reserves Layout ownership for its Content",
			ErrInvalidLayout,
		)
	}
	if layout == nil || layout.layoutState() == nil {
		return ErrInvalidLayout
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.layouts = append(t.layouts, transactionLayout{
		owner: owner.containerState(), layout: layout.layoutState(), primary: primary,
	})
	return nil
}

// Raise records moving a managed Panel to its Layout's highest Panel slot.
func (t *Transaction) Raise(control Control) error {
	return t.stackControl(control, true)
}

// Lower records moving a managed Panel to its Layout's lowest Panel slot.
func (t *Transaction) Lower(control Control) error {
	return t.stackControl(control, false)
}

func (t *Transaction) stackControl(control Control, raise bool) error {
	state, err := t.control(control)
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.stacks = append(t.stacks, transactionStack{control: state, raise: raise})
	return nil
}

// RaiseLayout records moving a Layout subtree to its highest Layout slot.
func (t *Transaction) RaiseLayout(layout Layout) error {
	return t.stackLayout(layout, true)
}

// LowerLayout records moving a Layout subtree to its lowest Layout slot.
func (t *Transaction) LowerLayout(layout Layout) error {
	return t.stackLayout(layout, false)
}

func (t *Transaction) stackLayout(layout Layout, raise bool) error {
	if err := t.usable(); err != nil {
		return err
	}
	if layout == nil || layout.layoutState() == nil {
		return ErrInvalidLayout
	}
	state := layout.layoutState()
	layoutBuilderMu.Lock()
	app := state.app
	layoutBuilderMu.Unlock()
	if app != t.app {
		return ErrInvalidLayout
	}
	app.mu.RLock()
	valid := state.attached && !state.destroyed
	app.mu.RUnlock()
	if !valid {
		return ErrInvalidLayout
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.stacks = append(t.stacks, transactionStack{layout: state, raise: raise})
	return nil
}

// SetTheme copies and records an atomic semantic-theme replacement.
func (t *Transaction) SetTheme(theme Theme) error {
	if err := t.usable(); err != nil {
		return err
	}
	if len(theme.styles) == 0 {
		return fmt.Errorf("%w: empty theme", ErrStyleMissing)
	}
	if len(theme.styles) > MaxThemeStyles {
		return fmt.Errorf(
			"%w: theme exceeds %d styles",
			ErrStyleMissing,
			MaxThemeStyles,
		)
	}
	if t.theme == nil {
		if err := t.reserveOperation(); err != nil {
			return err
		}
	}
	copied := Theme{styles: cloneStyleMap(theme.styles)}
	t.theme = &copied
	return nil
}

// Commit consumes the transaction on every attempt. It waits for the App
// mutation gate until ctx is cancelled or DefaultMutationWait elapses, then
// validates and applies all operations atomically and renders at most once.
// On failure it publishes nothing and permanently invalidates provisional
// handles. No application callback runs under App locks.
func (t *Transaction) Commit(ctx context.Context) (resultErr error) {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if err := t.usable(); err != nil {
		return err
	}
	t.consumed = true
	committed := false
	defer func() {
		if !committed {
			t.abortProvisional()
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.app.beginMutation(ctx); err != nil {
		return err
	}
	defer t.app.endMutation()

	if len(t.layouts) != 0 {
		layoutBuilderMu.Lock()
		defer layoutBuilderMu.Unlock()
	}
	t.app.mu.Lock()
	defer t.app.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.app.final {
		return ErrClosed
	}

	provisional := make(map[*controlState]bool, len(t.creates))
	for _, state := range t.creates {
		if state.app != t.app || !state.provisional ||
			state.destroyed || state.aborted {
			return ErrInvalidControl
		}
		if err := t.validateParentLocked(state.parent, provisional); err != nil {
			return err
		}
		provisional[state] = true
	}

	for _, mutation := range t.mutations {
		if err := t.validateTargetLocked(mutation.state, provisional); err != nil {
			return err
		}
	}
	for _, stack := range t.stacks {
		if stack.control != nil {
			if err := t.validateTargetLocked(stack.control, provisional); err != nil {
				return err
			}
			if stack.control.layout == nil {
				return ErrNotLayoutMember
			}
			continue
		}
		if stack.layout == nil || stack.layout.app != t.app ||
			!stack.layout.attached || stack.layout.destroyed {
			return ErrInvalidLayout
		}
	}
	destroyed := make(map[*controlState]bool)
	for _, state := range t.destroys {
		if err := t.validateTargetLocked(state, provisional); err != nil {
			return err
		}
		markDestroyedStates(state, destroyed)
	}
	if t.focusSet {
		if err := t.validateTargetLocked(t.focus, provisional); err != nil {
			return err
		}
		if destroyed[t.focus] {
			return ErrDestroyed
		}
		if !t.app.focusBehaviorEligibleLocked(
			t.focus,
			t.focus.behavior,
		) {
			return ErrNotFocusable
		}
		plannedVisibility := make(map[*controlState]bool)
		for _, mutation := range t.mutations {
			if mutation.kind == mutationVisible {
				plannedVisibility[mutation.state] = mutation.visible
			}
		}
		for current := t.focus; current != nil; current = current.parent {
			visible := current.visible
			if planned, found := plannedVisibility[current]; found {
				visible = planned
			}
			if !visible || destroyed[current] {
				return ErrNotFocusable
			}
		}
	}
	for _, state := range t.creates {
		if destroyed[state.parent] {
			return fmt.Errorf("%w: transaction destroys a construction parent", ErrInvalidParent)
		}
	}
	for _, mutation := range t.mutations {
		if destroyed[mutation.state] {
			return fmt.Errorf("%w: transaction mutates a destroyed control", ErrDestroyed)
		}
	}
	stagedMutationBehaviors := make(map[*controlState]controlBehavior)
	for index := range t.mutations {
		mutation := &t.mutations[index]
		base := mutation.state.behavior
		if staged, found := stagedMutationBehaviors[mutation.state]; found {
			base = staged
		}
		switch mutation.kind {
		case mutationText:
			mutable, ok := base.(mutableTextBehavior)
			if !ok {
				return fmt.Errorf(
					"%w: control has no mutable text",
					ErrInvalidControl,
				)
			}
			behavior, err := mutable.withText(mutation.text)
			if err != nil {
				return err
			}
			mutation.behavior = behavior
			stagedMutationBehaviors[mutation.state] = behavior
		case mutationTextField, mutationNumberField, mutationTextArea,
			mutationProgress, mutationScrollBar, mutationTabbedPanel,
			mutationScrollView:
			stagedMutationBehaviors[mutation.state] = mutation.behavior
		case mutationCheckState, mutationRadioValue,
			mutationChoiceValue, mutationChoiceOptions:
			behavior, err := t.selectionMutationBehaviorLocked(
				mutation,
				base,
				destroyed,
			)
			if err != nil {
				return err
			}
			mutation.behavior = behavior
			stagedMutationBehaviors[mutation.state] = behavior
		}
	}
	for _, state := range t.creates {
		target := controlBehaviorTarget(state.behavior)
		if target == nil {
			continue
		}
		if err := t.validateTargetLocked(target, provisional); err != nil {
			return fmt.Errorf("%w: invalid Label target", err)
		}
	}
	for _, mutation := range t.mutations {
		target := controlBehaviorTarget(mutation.behavior)
		if target == nil {
			continue
		}
		if err := t.validateTargetLocked(target, provisional); err != nil {
			return fmt.Errorf("%w: invalid Label target", err)
		}
	}
	for _, state := range t.creates {
		behavior, ok := state.behavior.(buttonBehavior)
		if !ok || !state.autoMinimum || destroyed[state] {
			continue
		}
		if definition, exists := t.app.commands[behavior.command]; exists {
			state.minimumSize = Size{
				Width: len(effectiveCommandLabel(
					definition,
					behavior.command,
				).lines[0]) + 5,
				Height: 1,
			}
		}
	}

	plannedLayouts := make(map[*layoutState]bool)
	plannedPanels := make(map[*controlState]bool)
	plannedKeys := make(map[string]bool)
	plannedRoots := make(map[*controlState]int)
	layoutCount := len(t.app.layoutsByID)
	layoutItems := t.app.layoutItemCount
	for key := range t.app.layoutsByKey {
		if key != "" {
			plannedKeys[key] = true
		}
	}
	for _, state := range t.app.controlsByID {
		if destroyed[state] || state.automationKey == "" {
			continue
		}
		if plannedKeys[state.automationKey] {
			return fmt.Errorf(
				"%w: automation key %q",
				ErrDuplicateKey,
				state.automationKey,
			)
		}
		plannedKeys[state.automationKey] = true
	}
	for _, state := range t.creates {
		if destroyed[state] || state.automationKey == "" {
			continue
		}
		if plannedKeys[state.automationKey] {
			return fmt.Errorf(
				"%w: automation key %q",
				ErrDuplicateKey,
				state.automationKey,
			)
		}
		plannedKeys[state.automationKey] = true
	}
	for _, attachment := range t.layouts {
		if err := t.validateTargetLocked(attachment.owner, provisional); err != nil {
			return ErrInvalidParent
		}
		if destroyed[attachment.owner] {
			return ErrDestroyed
		}
		if attachment.primary &&
			len(attachment.owner.layoutRoots)+plannedRoots[attachment.owner] != 0 {
			return fmt.Errorf("%w: owner already has a Layout", ErrLayoutAttached)
		}
		if !attachment.primary &&
			len(attachment.owner.layoutRoots)+plannedRoots[attachment.owner] == 0 {
			return fmt.Errorf(
				"%w: AddLayout requires an existing primary Layout",
				ErrInvalidLayout,
			)
		}
		addedLayouts, addedItems, err := t.validateLayoutTreeLocked(
			attachment.owner,
			attachment.layout,
			1,
			provisional,
			destroyed,
			plannedLayouts,
			plannedPanels,
			plannedKeys,
		)
		if err != nil {
			return err
		}
		layoutCount += addedLayouts
		layoutItems += addedItems
		if layoutCount > MaxLayouts || layoutItems > MaxLayoutItems {
			return ErrLayoutCapacity
		}
		plannedRoots[attachment.owner]++
	}
	for _, mutation := range t.mutations {
		if mutation.kind == mutationBounds &&
			(mutation.state.layout != nil || plannedPanels[mutation.state]) {
			return ErrLayoutManaged
		}
	}

	stagedStyles := cloneStyleMap(t.app.styles)
	if t.theme != nil {
		stagedStyles = cloneStyleMap(t.theme.styles)
	}
	if len(stagedStyles) == 0 || len(stagedStyles) > MaxThemeStyles {
		return fmt.Errorf("%w: invalid theme size", ErrStyleMissing)
	}
	for _, style := range stagedStyles {
		if err := validateResolvedStyle(style); err != nil {
			return err
		}
	}

	selectedStyles := make(map[*controlState]StyleID)
	for _, mutation := range t.mutations {
		if mutation.kind == mutationStyle {
			selectedStyles[mutation.state] = mutation.style
		}
	}
	selectedBehaviors := make(map[*controlState]controlBehavior)
	for _, mutation := range t.mutations {
		switch mutation.kind {
		case mutationTextField, mutationNumberField, mutationTextArea,
			mutationProgress, mutationScrollBar, mutationTabbedPanel,
			mutationScrollView,
			mutationStatusSegments,
			mutationCheckState, mutationRadioValue,
			mutationChoiceValue, mutationChoiceOptions:
			selectedBehaviors[mutation.state] = mutation.behavior
		}
	}
	effectiveBehavior := func(state *controlState) controlBehavior {
		if selected, found := selectedBehaviors[state]; found {
			return selected
		}
		return state.behavior
	}
	validateStyleReferences := func(state *controlState) error {
		style := state.style
		if selected, found := selectedStyles[state]; found {
			style = selected
		}
		if _, found := stagedStyles[style]; !found {
			return fmt.Errorf(
				"%w: theme has no definition for %q",
				ErrStyleMissing,
				style,
			)
		}
		behavior := effectiveBehavior(state)
		if provider, ok := behavior.(controlBorderProvider); ok {
			border := provider.controlBorder()
			if _, found := stagedStyles[border.borderStyle]; !found {
				return fmt.Errorf(
					"%w: theme has no definition for %q",
					ErrStyleMissing,
					border.borderStyle,
				)
			}
		}
		if styled, ok := behavior.(interface {
			additionalStyles() []StyleID
		}); ok {
			for _, additional := range styled.additionalStyles() {
				if _, found := stagedStyles[additional]; !found {
					return fmt.Errorf(
						"%w: theme has no definition for %q",
						ErrStyleMissing,
						additional,
					)
				}
			}
		}
		return nil
	}
	validateLayoutStyle := func(state *layoutState) error {
		if state.border.form == BorderNone {
			return nil
		}
		if _, found := stagedStyles[state.border.borderStyle]; !found {
			return fmt.Errorf(
				"%w: theme has no definition for %q",
				ErrStyleMissing,
				state.border.borderStyle,
			)
		}
		return nil
	}
	for _, state := range t.app.layoutsByID {
		if !state.destroyed {
			if err := validateLayoutStyle(state); err != nil {
				return err
			}
		}
	}
	for state := range plannedLayouts {
		if err := validateLayoutStyle(state); err != nil {
			return err
		}
	}

	activeCount := 0
	actionItems := 0
	selectionItems := 0
	textInputBytes := 0
	menuBars := 0
	statusBars := 0
	mnemonics := make(map[Key]*controlState)
	defaults := make(map[*controlState]*controlState)
	cancels := make(map[*controlState]*controlState)
	radioValues := make(map[*controlState]map[string]*controlState)
	radioSelections := make(map[*controlState]*controlState)
	validateChangeCommand := func(
		command CommandID,
		requireCommand bool,
		controlName string,
	) error {
		if command == "" || !requireCommand {
			return nil
		}
		if _, exists := t.app.commands[command]; !exists {
			return fmt.Errorf(
				"%w: %s change command %q is not registered",
				ErrInvalidControl,
				controlName,
				command,
			)
		}
		return nil
	}
	validateMnemonic := func(state *controlState, mnemonic Key) error {
		if mnemonic == "" {
			return nil
		}
		if existing := mnemonics[mnemonic]; existing != nil {
			return fmt.Errorf(
				"%w: duplicate Action mnemonic %q",
				ErrInvalidControl,
				mnemonic,
			)
		}
		mnemonics[mnemonic] = state
		return nil
	}
	validateAction := func(
		state *controlState,
		behavior controlBehavior,
		requireCommand bool,
	) error {
		switch behavior := behavior.(type) {
		case buttonBehavior:
			if requireCommand {
				if _, exists := t.app.commands[behavior.command]; !exists {
					return fmt.Errorf(
						"%w: Button command %q is not registered",
						ErrInvalidControl,
						behavior.command,
					)
				}
			}
			if err := validateMnemonic(state, behavior.mnemonic); err != nil {
				return err
			}
			if behavior.default_ {
				if defaults[state.parent] != nil {
					return fmt.Errorf(
						"%w: duplicate default Button under one parent",
						ErrInvalidControl,
					)
				}
				defaults[state.parent] = state
			}
			if behavior.cancel {
				if cancels[state.parent] != nil {
					return fmt.Errorf(
						"%w: duplicate cancel Button under one parent",
						ErrInvalidControl,
					)
				}
				cancels[state.parent] = state
			}
		case hotkeyBarBehavior:
			actionItems += len(behavior.items)
			if requireCommand {
				for _, item := range behavior.items {
					if _, exists := t.app.commands[item.Command]; !exists {
						return fmt.Errorf(
							"%w: HotkeyBar command %q is not registered",
							ErrInvalidControl,
							item.Command,
						)
					}
				}
			}
		case menuBarBehavior:
			menuBars++
			if menuBars > 1 {
				return fmt.Errorf(
					"%w: v0 supports one MenuBar per App",
					ErrInvalidControl,
				)
			}
			if err := validateMenuBarTreeLocked(
				t.app,
				behavior,
				requireCommand,
			); err != nil {
				return err
			}
		case statusBarBehavior:
			statusBars++
			if statusBars > 1 {
				return fmt.Errorf(
					"%w: v0 supports one StatusBar per App",
					ErrInvalidControl,
				)
			}
			actionItems += len(behavior.segments)
			if requireCommand {
				if err := validateStatusCommandsLocked(t.app, behavior); err != nil {
					return err
				}
			}
		case textBehavior:
			if behavior.mnemonic != "" && !destroyed[behavior.target] {
				if err := validateMnemonic(state, behavior.mnemonic); err != nil {
					return err
				}
			}
		case checkboxBehavior:
			if err := validateMnemonic(state, behavior.mnemonic); err != nil {
				return err
			}
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"Checkbox",
			); err != nil {
				return err
			}
		case radioGroupBehavior:
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"RadioGroup",
			); err != nil {
				return err
			}
		case radioButtonBehavior:
			selectionItems++
			if state.parent == nil ||
				state.parent.kind != ControlRadioGroup {
				return fmt.Errorf(
					"%w: RadioButton requires a RadioGroup parent",
					ErrInvalidParent,
				)
			}
			values := radioValues[state.parent]
			if values == nil {
				values = make(map[string]*controlState)
				radioValues[state.parent] = values
			}
			if values[behavior.value] != nil {
				return fmt.Errorf(
					"%w: duplicate RadioButton value %q",
					ErrInvalidControl,
					behavior.value,
				)
			}
			values[behavior.value] = state
			if behavior.initialSelected {
				if radioSelections[state.parent] != nil {
					return fmt.Errorf(
						"%w: multiple initially selected RadioButtons",
						ErrInvalidControl,
					)
				}
				radioSelections[state.parent] = state
			}
			if err := validateMnemonic(state, behavior.mnemonic); err != nil {
				return err
			}
		case choiceFieldBehavior:
			selectionItems += len(behavior.options)
			if err := validateMnemonic(state, behavior.mnemonic); err != nil {
				return err
			}
		case textFieldBehavior:
			textInputBytes += len(behavior.committed.text)
			if behavior.editing {
				textInputBytes += len(behavior.working.text)
			}
			if behavior.validator != nil {
				textInputBytes += len(behavior.validator.value.Characters)
			}
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"TextField",
			); err != nil {
				return err
			}
		case numberFieldBehavior:
			textInputBytes += len(behavior.editor.committed.text)
			if behavior.editor.editing {
				textInputBytes += len(behavior.editor.working.text)
			}
			if behavior.editor.validator != nil {
				textInputBytes += len(
					behavior.editor.validator.value.Characters,
				)
			}
			if err := validateChangeCommand(
				behavior.editor.changeCommand,
				requireCommand,
				"numeric field",
			); err != nil {
				return err
			}
			if _, _, err := normalizeNumberValue(
				behavior.value,
				behavior.policy,
			); err != nil {
				return err
			}
		case textAreaBehavior:
			textInputBytes += len(behavior.committed.text)
			if behavior.editing {
				textInputBytes += len(behavior.working.text)
			}
			if behavior.validator != nil {
				textInputBytes += len(behavior.validator.value.Characters)
			}
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"text area",
			); err != nil {
				return err
			}
		case scrollBarBehavior:
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"ScrollBar",
			); err != nil {
				return err
			}
		case tabbedPanelBehavior:
			selectionItems += len(behavior.tabs)
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"tab container",
			); err != nil {
				return err
			}
			for _, tab := range behavior.tabs {
				if destroyed[tab.page] {
					continue
				}
				if err := t.validateTargetLocked(
					tab.page,
					provisional,
				); err != nil ||
					tab.page.parent != state ||
					tab.page.kind != ControlPanel ||
					tab.page.layout != nil ||
					plannedPanels[tab.page] {
					return fmt.Errorf(
						"%w: invalid Tab page",
						ErrInvalidParent,
					)
				}
			}
		case scrollViewBehavior:
			if err := validateChangeCommand(
				behavior.changeCommand,
				requireCommand,
				"scroll container",
			); err != nil {
				return err
			}
			content := behavior.content
			if content == nil ||
				destroyed[content] ||
				t.validateTargetLocked(content, provisional) != nil ||
				content.parent != state ||
				content.kind != ControlPanel ||
				content.managedBy != state ||
				content.layout != nil ||
				plannedPanels[content] {
				return fmt.Errorf(
					"%w: invalid managed scroll Content",
					ErrInvalidParent,
				)
			}
			directChildren := 0
			for _, child := range state.children {
				if !destroyed[child] {
					directChildren++
					if child != content {
						return fmt.Errorf(
							"%w: unexpected scroll-container child",
							ErrInvalidParent,
						)
					}
				}
			}
			for _, child := range t.creates {
				if child.parent == state && !destroyed[child] {
					directChildren++
					if child != content {
						return fmt.Errorf(
							"%w: unexpected scroll-container construction child",
							ErrInvalidParent,
						)
					}
				}
			}
			if directChildren != 1 {
				return fmt.Errorf(
					"%w: scroll container requires exactly one Content child",
					ErrInvalidParent,
				)
			}
		}
		return nil
	}
	for _, state := range t.app.controlsByID {
		if destroyed[state] {
			continue
		}
		activeCount++
		if err := validateStyleReferences(state); err != nil {
			return err
		}
		_, behaviorMutated := selectedBehaviors[state]
		if err := validateAction(
			state,
			effectiveBehavior(state),
			behaviorMutated,
		); err != nil {
			return err
		}
	}
	for _, state := range t.creates {
		if destroyed[state] {
			continue
		}
		activeCount++
		if err := validateStyleReferences(state); err != nil {
			return err
		}
		if err := validateAction(
			state,
			effectiveBehavior(state),
			true,
		); err != nil {
			return err
		}
	}
	if activeCount > MaxControls {
		return ErrControlCapacity
	}
	if actionItems > MaxActionItems {
		return ErrControlCapacity
	}
	if selectionItems > MaxSelectionItems {
		return ErrControlCapacity
	}
	if textInputBytes > MaxTextInputAggregateBytes {
		return fmt.Errorf(
			"%w: aggregate text input exceeds %d bytes",
			ErrControlCapacity,
			MaxTextInputAggregateBytes,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	changed := false
	for _, state := range t.app.controlsByID {
		if destroyed[state] {
			continue
		}
		if behavior, cleared := clearControlBehaviorTarget(
			state.behavior,
			destroyed,
		); cleared {
			state.behavior = behavior
			changed = true
		}
	}
	for _, state := range t.creates {
		if destroyed[state] {
			continue
		}
		if behavior, cleared := clearControlBehaviorTarget(
			state.behavior,
			destroyed,
		); cleared {
			state.behavior = behavior
		}
	}
	for index := range t.mutations {
		if behavior, cleared := clearControlBehaviorTarget(
			t.mutations[index].behavior,
			destroyed,
		); cleared {
			t.mutations[index].behavior = behavior
		}
	}
	for _, state := range uniqueTopLevelDestroys(t.destroys, destroyed) {
		if state.id == "" || state.destroyed {
			continue
		}
		removeChildState(state.parent, state)
		t.app.destroyStateLocked(state)
		changed = true
	}
	for _, state := range t.creates {
		if destroyed[state] {
			state.provisional = false
			state.destroyed = true
			continue
		}
		state.id = t.app.nextControlIDLocked()
		state.provisional = false
		state.parent.children = append(state.parent.children, state)
		t.app.controlsByID[state.id] = state
		if state.automationKey != "" {
			t.app.controlsByKey[state.automationKey] = state
		}
		if behavior, ok := state.behavior.(buttonBehavior); ok &&
			state.autoMinimum {
			state.minimumSize = Size{
				Width: len(effectiveCommandLabel(
					t.app.commands[behavior.command],
					behavior.command,
				).lines[0]) + 5,
				Height: 1,
			}
		}
		changed = true
	}
	for _, attachment := range t.layouts {
		t.app.attachLayoutTreeLocked(
			attachment.owner,
			nil,
			attachment.layout,
		)
		attachment.owner.layoutRoots = append(
			attachment.owner.layoutRoots,
			attachment.layout,
		)
		changed = true
	}
	targetSize := t.app.size
	if t.size != nil {
		targetSize = *t.size
	}
	targetConstraints := t.app.rootConstraints
	if t.constraints != nil {
		targetConstraints = *t.constraints
	}
	targetRootBounds := constrainedRootBounds(targetSize, targetConstraints)
	if t.app.size != targetSize ||
		t.app.rootConstraints != targetConstraints ||
		t.app.root.state.bounds != targetRootBounds {
		t.app.size = targetSize
		t.app.rootConstraints = targetConstraints
		t.app.root.state.bounds = targetRootBounds
		t.app.root.state.minimumSize = targetConstraints.Minimum
		changed = true
	}
	for _, mutation := range t.mutations {
		switch mutation.kind {
		case mutationBounds:
			if mutation.state.bounds != mutation.rect {
				mutation.state.bounds = mutation.rect
				changed = true
			}
		case mutationMinimum:
			if mutation.state.minimumSize != mutation.size {
				mutation.state.minimumSize = mutation.size
				changed = true
			}
			mutation.state.autoMinimum = false
		case mutationStyle:
			if mutation.state.style != mutation.style {
				mutation.state.style = mutation.style
				changed = true
			}
		case mutationVisible:
			if mutation.state.visible != mutation.visible {
				mutation.state.visible = mutation.visible
				changed = true
			}
		case mutationText:
			if !controlBehaviorEqual(mutation.state.behavior, mutation.behavior) {
				mutation.state.behavior = mutation.behavior
				if mutation.state.autoMinimum {
					mutation.state.minimumSize =
						mutation.behavior.(intrinsicMinimumBehavior).intrinsicMinimum()
				}
				changed = true
			}
		case mutationStatusSegments:
			current := mutation.state.behavior.(statusBarBehavior)
			next := mutation.behavior.(statusBarBehavior)
			if !statusBarBehaviorEqual(current, next) {
				mutation.state.behavior = next
				changed = true
			}
		case mutationCheckState, mutationRadioValue,
			mutationChoiceValue, mutationChoiceOptions:
			if !selectionBehaviorEqual(
				mutation.state.behavior,
				mutation.behavior,
			) {
				mutation.state.behavior = mutation.behavior
				if mutation.state.autoMinimum {
					mutation.state.minimumSize =
						mutation.behavior.(intrinsicMinimumBehavior).
							intrinsicMinimum()
				}
				changed = true
			}
		case mutationFocusGuidance:
			if !focusGuidanceConfigEqual(
				mutation.state.focusGuidance,
				mutation.focusGuidance,
			) {
				mutation.state.focusGuidance = mutation.focusGuidance
				changed = true
			}
		case mutationTextField, mutationNumberField, mutationTextArea,
			mutationProgress, mutationScrollBar, mutationTabbedPanel,
			mutationScrollView:
			if !controlBehaviorEqual(
				mutation.state.behavior,
				mutation.behavior,
			) {
				mutation.state.behavior = mutation.behavior
				if mutation.state.autoMinimum {
					mutation.state.minimumSize =
						mutation.behavior.(intrinsicMinimumBehavior).
							intrinsicMinimum()
				}
				changed = true
			}
		}
	}
	if t.app.repairRadioGroupsLocked() {
		changed = true
	}
	if t.app.repairTabbedPanelsLocked(destroyed) {
		changed = true
	}
	if t.app.updateApplicationChromeBoundsLocked(targetSize) {
		changed = true
	}
	for _, stack := range t.stacks {
		if stack.control != nil {
			item := layoutItemForPanelLocked(stack.control.layout, stack.control)
			if item != nil &&
				reorderPeerItems(stack.control.layout.stack, item, stack.raise) {
				changed = true
			}
			continue
		}
		if reorderLayoutLocked(stack.layout, stack.raise) {
			changed = true
		}
	}
	if t.theme != nil && !styleMapsEqual(t.app.styles, stagedStyles) {
		t.app.styles = stagedStyles
		changed = true
	}
	if t.focusSet {
		if t.app.menu != nil {
			t.app.closeMenuLocked()
			changed = true
		}
		if t.app.focus != t.focus {
			if t.app.commitOrCancelEditorStateLocked(t.app.focus) {
				changed = true
			}
			t.app.focus = t.focus
			t.app.clearInvalidPressesLocked()
			changed = true
		}
	}
	if t.app.repairMenuLocked() {
		changed = true
	}
	if t.app.ensureFocusLocked() {
		changed = true
	}
	if changed {
		arrangeAllLayoutsLocked(t.app)
		t.app.ensureFocusLocked()
		t.app.ensureFocusedControlVisibleLocked()
		t.app.publishLocked(nil)
	}
	committed = true
	return nil
}

func (t *Transaction) usable() error {
	if t == nil || t.app == nil {
		return ErrInvalidControl
	}
	if t.consumed {
		return errors.New("expletives: transaction is already consumed")
	}
	return nil
}

func (t *Transaction) reserveOperation() error {
	count := len(t.creates) + len(t.mutations) + len(t.destroys) +
		len(t.layouts) + len(t.stacks)
	if t.size != nil {
		count++
	}
	if t.constraints != nil {
		count++
	}
	if t.theme != nil {
		count++
	}
	if t.focusSet {
		count++
	}
	if count >= MaxTransactionOperations {
		return ErrTransactionCapacity
	}
	return nil
}

func (t *Transaction) validateLayoutTreeLocked(
	owner *controlState,
	state *layoutState,
	depth int,
	provisional map[*controlState]bool,
	destroyed map[*controlState]bool,
	plannedLayouts map[*layoutState]bool,
	plannedPanels map[*controlState]bool,
	plannedKeys map[string]bool,
) (layoutCount, itemCount int, err error) {
	if state == nil || state.attached || state.destroyed ||
		plannedLayouts[state] || depth > MaxLayoutDepth {
		return 0, 0, fmt.Errorf("%w: unusable Layout tree", ErrInvalidLayout)
	}
	if depth == 1 && state.parent != nil {
		return 0, 0, fmt.Errorf("%w: top-level Layout has a parent", ErrInvalidLayout)
	}
	if depth == 1 {
		minimum := measureLayoutLocked(state)
		if minimum.Width > maxCoordinateMagnitude ||
			minimum.Height > maxCoordinateMagnitude {
			return 0, 0, fmt.Errorf(
				"%w: combined Layout minimum exceeds checked geometry",
				ErrInvalidGeometry,
			)
		}
		if (owner.kind == ControlHeader || owner.kind == ControlFooter) &&
			minimum.Height > 1 {
			return 0, 0, fmt.Errorf(
				"%w: one-row application chrome Layout minimum height is %d",
				ErrInvalidLayout,
				minimum.Height,
			)
		}
	}
	if state.automationKey != "" && plannedKeys[state.automationKey] {
		return 0, 0, fmt.Errorf(
			"%w: Layout automation key %q",
			ErrDuplicateKey,
			state.automationKey,
		)
	}
	plannedLayouts[state] = true
	if state.automationKey != "" {
		plannedKeys[state.automationKey] = true
	}
	layoutCount = 1
	itemCount = len(state.items)
	if err := validateGridCapacityLocked(state); err != nil {
		return 0, 0, err
	}
	for _, item := range state.items {
		switch item.kind {
		case layoutPanelItem:
			panel := item.panel
			if panel == nil || panel.app != t.app || panel.parent != owner ||
				panel.layout != nil || plannedPanels[panel] ||
				destroyed[panel] || panel.managedBy != nil {
				return 0, 0, fmt.Errorf(
					"%w: Panel item is not an available direct child",
					ErrInvalidLayout,
				)
			}
			if isApplicationChrome(panel.kind) {
				return 0, 0, fmt.Errorf(
					"%w: application chrome cannot be a Layout item",
					ErrInvalidLayout,
				)
			}
			if err := t.validateTargetLocked(panel, provisional); err != nil {
				return 0, 0, err
			}
			plannedPanels[panel] = true
		case layoutLayoutItem:
			if item.layout == nil || item.layout.parent != state {
				return 0, 0, fmt.Errorf(
					"%w: inconsistent nested Layout",
					ErrInvalidLayout,
				)
			}
			childLayouts, childItems, childErr := t.validateLayoutTreeLocked(
				owner,
				item.layout,
				depth+1,
				provisional,
				destroyed,
				plannedLayouts,
				plannedPanels,
				plannedKeys,
			)
			if childErr != nil {
				return 0, 0, childErr
			}
			layoutCount += childLayouts
			itemCount += childItems
		default:
			return 0, 0, ErrInvalidLayout
		}
	}
	return layoutCount, itemCount, nil
}

func (a *App) attachLayoutTreeLocked(
	owner *controlState,
	parent *layoutState,
	state *layoutState,
) {
	state.id = a.nextLayoutIDLocked()
	state.app = a
	state.owner = owner
	state.parent = parent
	state.attached = true
	if parent == nil {
		state.rootIndex = len(owner.layoutRoots)
	}
	a.layoutsByID[state.id] = state
	if state.automationKey != "" {
		a.layoutsByKey[state.automationKey] = state
	}
	a.layoutItemCount += len(state.items)
	for _, item := range state.items {
		if item.kind == layoutPanelItem {
			item.panel.layout = state
		} else {
			a.attachLayoutTreeLocked(owner, state, item.layout)
		}
	}
}

func layoutItemForPanelLocked(
	layout *layoutState,
	panel *controlState,
) *layoutItem {
	for _, item := range layout.items {
		if item.kind == layoutPanelItem && item.panel == panel {
			return item
		}
	}
	return nil
}

func reorderLayoutLocked(state *layoutState, raise bool) bool {
	if state.parent != nil {
		for _, item := range state.parent.items {
			if item.kind == layoutLayoutItem && item.layout == state {
				return reorderPeerItems(state.parent.stack, item, raise)
			}
		}
		return false
	}
	roots := state.owner.layoutRoots
	index := -1
	for candidate, root := range roots {
		if root == state {
			index = candidate
			break
		}
	}
	if index < 0 || len(roots) < 2 ||
		(raise && index == len(roots)-1) || (!raise && index == 0) {
		return false
	}
	if raise {
		copy(roots[index:], roots[index+1:])
		roots[len(roots)-1] = state
	} else {
		copy(roots[1:index+1], roots[:index])
		roots[0] = state
	}
	return true
}

func (t *Transaction) abortProvisional() {
	if t == nil || t.app == nil {
		return
	}
	t.app.mu.Lock()
	defer t.app.mu.Unlock()
	for _, state := range t.creates {
		if !state.provisional {
			continue
		}
		state.provisional = false
		state.aborted = true
	}
}

func (a *App) beginMutation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(DefaultMutationWait)
	defer timer.Stop()
	select {
	case a.mutationGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-a.mutationGate
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrMutationBusy
	}
}

func (a *App) endMutation() {
	<-a.mutationGate
}

func (t *Transaction) control(control Control) (*controlState, error) {
	if err := t.usable(); err != nil {
		return nil, err
	}
	if control == nil || control.controlState() == nil ||
		control.controlState().app != t.app {
		return nil, ErrInvalidControl
	}
	return control.controlState(), nil
}

func (t *Transaction) validateParentLocked(
	parent *controlState,
	provisional map[*controlState]bool,
) error {
	if parent == nil || parent.app != t.app || parent.destroyed {
		return ErrInvalidParent
	}
	if parent.provisional {
		if !provisional[parent] {
			return fmt.Errorf(
				"%w: provisional parent must be created earlier in the transaction",
				ErrInvalidParent,
			)
		}
		return nil
	}
	if canonical, found := t.app.controlsByID[parent.id]; !found ||
		canonical != parent {
		return ErrInvalidParent
	}
	return nil
}

func (t *Transaction) validateTargetLocked(
	state *controlState,
	provisional map[*controlState]bool,
) error {
	if state == nil || state.app != t.app || state.aborted {
		return ErrInvalidControl
	}
	if state.destroyed {
		return ErrDestroyed
	}
	if state.provisional {
		if !provisional[state] {
			return ErrInvalidControl
		}
		return nil
	}
	if canonical, found := t.app.controlsByID[state.id]; !found ||
		canonical != state {
		return ErrInvalidControl
	}
	return nil
}

func markDestroyedStates(state *controlState, marked map[*controlState]bool) {
	if state == nil || marked[state] {
		return
	}
	marked[state] = true
	for _, child := range state.children {
		markDestroyedStates(child, marked)
	}
}

func uniqueTopLevelDestroys(
	requested []*controlState,
	all map[*controlState]bool,
) []*controlState {
	unique := make([]*controlState, 0, len(requested))
	seen := make(map[*controlState]bool)
	for _, state := range requested {
		if seen[state] {
			continue
		}
		if state.parent != nil && all[state.parent] {
			continue
		}
		seen[state] = true
		unique = append(unique, state)
	}
	return unique
}
