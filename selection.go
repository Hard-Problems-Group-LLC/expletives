package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CheckState is the canonical Checkbox selection state.
type CheckState string

const (
	CheckUnchecked     CheckState = "unchecked"
	CheckChecked       CheckState = "checked"
	CheckIndeterminate CheckState = "indeterminate"
)

// SelectionOption is one stable fixed choice in a CycleField or SelectField.
type SelectionOption struct {
	Value          string
	Label          string
	Disabled       bool
	DisabledReason string
}

// CheckboxOptions configures one two-state or three-state Checkbox.
type CheckboxOptions struct {
	PanelOptions
	Label          string
	Mnemonic       Key
	State          CheckState
	ThreeState     bool
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// RadioGroupOptions configures one exclusive-selection container.
type RadioGroupOptions struct {
	PanelOptions
	AllowEmpty     bool
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// RadioButtonOptions configures one option directly owned by a RadioGroup.
type RadioButtonOptions struct {
	PanelOptions
	Value          string
	Label          string
	Mnemonic       Key
	Selected       bool
	Disabled       bool
	DisabledReason string
}

// CycleFieldOptions configures one bounded fixed-option cycling field.
type CycleFieldOptions struct {
	PanelOptions
	Label          string
	Mnemonic       Key
	Options        []SelectionOption
	Value          string
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// SelectFieldOptions is the SelectField naming variant of CycleFieldOptions.
type SelectFieldOptions = CycleFieldOptions

// Checkbox is a copy-safe focusable selection leaf.
type Checkbox struct{ controlHandle }

// RadioGroup is a copy-safe exclusive-selection container.
type RadioGroup struct{ containerHandle }

// RadioButton is a copy-safe focusable option leaf.
type RadioButton struct{ controlHandle }

// CycleField is a copy-safe focusable fixed-option selector.
type CycleField struct{ controlHandle }

// SelectField is the distinct-kind naming variant of CycleField.
type SelectField struct{ controlHandle }

type checkboxBehavior struct {
	label          normalizedDisplayText
	mnemonic       Key
	state          CheckState
	threeState     bool
	disabled       bool
	disabledReason string
	changeCommand  CommandID
}

type radioGroupBehavior struct {
	value          string
	allowEmpty     bool
	disabled       bool
	disabledReason string
	changeCommand  CommandID
}

type radioButtonBehavior struct {
	value           string
	label           normalizedDisplayText
	mnemonic        Key
	initialSelected bool
	disabled        bool
	disabledReason  string
}

type selectionOptionBehavior struct {
	option SelectionOption
	label  normalizedDisplayText
}

type choiceFieldBehavior struct {
	label          normalizedDisplayText
	mnemonic       Key
	options        []selectionOptionBehavior
	value          string
	disabled       bool
	disabledReason string
	changeCommand  CommandID
}

// NewCheckbox constructs and atomically inserts a Checkbox.
func NewCheckbox(parent Container, options CheckboxOptions) (*Checkbox, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	checkbox, err := tx.NewCheckbox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return checkbox, nil
}

// NewRadioGroup constructs and atomically inserts a RadioGroup.
func NewRadioGroup(
	parent Container,
	options RadioGroupOptions,
) (*RadioGroup, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	group, err := tx.NewRadioGroup(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return group, nil
}

// NewRadioButton constructs and atomically inserts a RadioButton.
func NewRadioButton(
	parent *RadioGroup,
	options RadioButtonOptions,
) (*RadioButton, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	button, err := tx.NewRadioButton(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return button, nil
}

// NewCycleField constructs and atomically inserts a CycleField.
func NewCycleField(
	parent Container,
	options CycleFieldOptions,
) (*CycleField, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	field, err := tx.NewCycleField(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return field, nil
}

// NewSelectField constructs and atomically inserts a SelectField.
func NewSelectField(
	parent Container,
	options SelectFieldOptions,
) (*SelectField, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	field, err := tx.NewSelectField(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return field, nil
}

// NewCheckbox records construction of a provisional Checkbox.
func (t *Transaction) NewCheckbox(
	parent Container,
	options CheckboxOptions,
) (*Checkbox, error) {
	label, mnemonic, disabledReason, err := normalizeSelectionPresentation(
		options.Label,
		options.Mnemonic,
		options.Disabled,
		options.DisabledReason,
	)
	if err != nil {
		return nil, err
	}
	state := options.State
	if state == "" {
		state = CheckUnchecked
	}
	if err := validateCheckState(state, options.ThreeState); err != nil {
		return nil, err
	}
	if err := validateOptionalCommand(options.ChangeCommand); err != nil {
		return nil, err
	}
	control, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlCheckbox,
		checkboxBehavior{
			label: label, mnemonic: mnemonic, state: state,
			threeState: options.ThreeState, disabled: options.Disabled,
			disabledReason: disabledReason,
			changeCommand:  options.ChangeCommand,
		},
	)
	if err != nil {
		return nil, err
	}
	checkbox := &Checkbox{controlHandle: controlHandle{state: control}}
	control.control = checkbox
	return checkbox, nil
}

// NewRadioGroup records construction of a provisional RadioGroup.
func (t *Transaction) NewRadioGroup(
	parent Container,
	options RadioGroupOptions,
) (*RadioGroup, error) {
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
	panel, err := t.newControl(
		parent,
		options.PanelOptions,
		ControlRadioGroup,
		radioGroupBehavior{
			allowEmpty: options.AllowEmpty, disabled: options.Disabled,
			disabledReason: reason, changeCommand: options.ChangeCommand,
		},
	)
	if err != nil {
		return nil, err
	}
	group := &RadioGroup{containerHandle: panel.containerHandle}
	panel.state.control = group
	panel.state.container = group
	return group, nil
}

// NewRadioButton records construction of a provisional RadioButton.
func (t *Transaction) NewRadioButton(
	parent *RadioGroup,
	options RadioButtonOptions,
) (*RadioButton, error) {
	if parent == nil || parent.controlState() == nil ||
		parent.controlState().kind != ControlRadioGroup {
		return nil, ErrInvalidParent
	}
	if !validBoundedIdentifier(options.Value) {
		return nil, fmt.Errorf("%w: invalid RadioButton value", ErrInvalidControl)
	}
	label, mnemonic, disabledReason, err := normalizeSelectionPresentation(
		options.Label,
		options.Mnemonic,
		options.Disabled,
		options.DisabledReason,
	)
	if err != nil {
		return nil, err
	}
	control, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlRadioButton,
		radioButtonBehavior{
			value: options.Value, label: label, mnemonic: mnemonic,
			initialSelected: options.Selected, disabled: options.Disabled,
			disabledReason: disabledReason,
		},
	)
	if err != nil {
		return nil, err
	}
	button := &RadioButton{controlHandle: controlHandle{state: control}}
	control.control = button
	return button, nil
}

// NewCycleField records construction of a provisional CycleField.
func (t *Transaction) NewCycleField(
	parent Container,
	options CycleFieldOptions,
) (*CycleField, error) {
	state, err := t.newChoiceField(parent, options, ControlCycleField)
	if err != nil {
		return nil, err
	}
	field := &CycleField{controlHandle: controlHandle{state: state}}
	state.control = field
	return field, nil
}

// NewSelectField records construction of a provisional SelectField.
func (t *Transaction) NewSelectField(
	parent Container,
	options SelectFieldOptions,
) (*SelectField, error) {
	state, err := t.newChoiceField(parent, options, ControlSelectField)
	if err != nil {
		return nil, err
	}
	field := &SelectField{controlHandle: controlHandle{state: state}}
	state.control = field
	return field, nil
}

func (t *Transaction) newChoiceField(
	parent Container,
	options CycleFieldOptions,
	kind ControlKind,
) (*controlState, error) {
	label, mnemonic, disabledReason, err := normalizeSelectionPresentation(
		options.Label,
		options.Mnemonic,
		options.Disabled,
		options.DisabledReason,
	)
	if err != nil {
		return nil, err
	}
	choices, value, err := normalizeSelectionOptions(
		options.Options,
		options.Value,
	)
	if err != nil {
		return nil, err
	}
	if err := validateOptionalCommand(options.ChangeCommand); err != nil {
		return nil, err
	}
	return t.newLeafControl(
		parent,
		options.PanelOptions,
		kind,
		choiceFieldBehavior{
			label: label, mnemonic: mnemonic, options: choices, value: value,
			disabled:       options.Disabled,
			disabledReason: disabledReason,
			changeCommand:  options.ChangeCommand,
		},
	)
}

// SetCheckState records one atomic Checkbox state replacement.
func (t *Transaction) SetCheckState(
	checkbox *Checkbox,
	state CheckState,
) error {
	control, err := t.control(checkbox)
	if err != nil || control.kind != ControlCheckbox {
		return ErrInvalidControl
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:           mutationCheckState,
		state:          control,
		selectionValue: string(state),
	})
	return nil
}

// SetRadioValue records one atomic RadioGroup selection replacement.
func (t *Transaction) SetRadioValue(
	group *RadioGroup,
	value string,
) error {
	control, err := t.control(group)
	if err != nil || control.kind != ControlRadioGroup {
		return ErrInvalidControl
	}
	if value != "" && !validBoundedIdentifier(value) {
		return fmt.Errorf("%w: invalid radio value", ErrInvalidControl)
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:           mutationRadioValue,
		state:          control,
		selectionValue: value,
	})
	return nil
}

// SetChoiceValue records one CycleField or SelectField value replacement.
func (t *Transaction) SetChoiceValue(
	field Control,
	value string,
) error {
	control, err := t.control(field)
	if err != nil ||
		(control.kind != ControlCycleField &&
			control.kind != ControlSelectField) {
		return ErrInvalidControl
	}
	if value != "" && !validBoundedIdentifier(value) {
		return fmt.Errorf("%w: invalid choice value", ErrInvalidControl)
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:           mutationChoiceValue,
		state:          control,
		selectionValue: value,
	})
	return nil
}

// SetChoiceOptions records one copied CycleField or SelectField option-model
// replacement. Empty value applies the deterministic first-enabled repair.
func (t *Transaction) SetChoiceOptions(
	field Control,
	options []SelectionOption,
	value string,
) error {
	control, err := t.control(field)
	if err != nil ||
		(control.kind != ControlCycleField &&
			control.kind != ControlSelectField) {
		return ErrInvalidControl
	}
	normalized, selected, err := normalizeSelectionOptions(options, value)
	if err != nil {
		return err
	}
	if err := t.reserveOperation(); err != nil {
		return err
	}
	copied := make([]SelectionOption, len(normalized))
	for index := range normalized {
		copied[index] = normalized[index].option
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:             mutationChoiceOptions,
		state:            control,
		selectionValue:   selected,
		selectionOptions: copied,
	})
	return nil
}

func normalizeSelectionPresentation(
	label string,
	mnemonic Key,
	disabled bool,
	disabledReason string,
) (normalizedDisplayText, Key, string, error) {
	normalized, err := normalizeDisplayText(label, false)
	if err != nil {
		return normalizedDisplayText{}, "", "", err
	}
	if normalized.cells == 0 {
		return normalizedDisplayText{}, "", "", fmt.Errorf(
			"%w: selection label is empty",
			ErrInvalidControl,
		)
	}
	mnemonic, err = normalizeMnemonic(mnemonic)
	if err != nil {
		return normalizedDisplayText{}, "", "", err
	}
	reason, err := normalizeDisabledReason(disabled, disabledReason)
	if err != nil {
		return normalizedDisplayText{}, "", "", err
	}
	return normalized, mnemonic, reason, nil
}

func normalizeDisabledReason(disabled bool, reason string) (string, error) {
	if len(reason) > MaxCommandDescriptionBytes ||
		strings.ContainsRune(reason, 0) {
		return "", fmt.Errorf("%w: invalid disabled reason", ErrTextLimit)
	}
	if disabled && reason == "" {
		return "Control is disabled", nil
	}
	if !disabled && reason != "" {
		return "", fmt.Errorf(
			"%w: enabled control has a disabled reason",
			ErrInvalidControl,
		)
	}
	return reason, nil
}

func validateOptionalCommand(command CommandID) error {
	if command != "" && !validBoundedIdentifier(string(command)) {
		return fmt.Errorf("%w: invalid change command", ErrInvalidControl)
	}
	return nil
}

func validateCheckState(state CheckState, threeState bool) error {
	switch state {
	case CheckUnchecked, CheckChecked:
		return nil
	case CheckIndeterminate:
		if threeState {
			return nil
		}
	}
	return fmt.Errorf("%w: invalid Checkbox state", ErrInvalidControl)
}

func normalizeSelectionOptions(
	options []SelectionOption,
	requested string,
) ([]selectionOptionBehavior, string, error) {
	if len(options) > MaxSelectionOptions {
		return nil, "", fmt.Errorf(
			"%w: selection field exceeds %d options",
			ErrControlCapacity,
			MaxSelectionOptions,
		)
	}
	normalized := make([]selectionOptionBehavior, len(options))
	seen := make(map[string]bool, len(options))
	selectedEnabled := false
	firstEnabled := ""
	for index, option := range options {
		if !validBoundedIdentifier(option.Value) || seen[option.Value] {
			return nil, "", fmt.Errorf(
				"%w: invalid or duplicate selection value %q",
				ErrInvalidControl,
				option.Value,
			)
		}
		seen[option.Value] = true
		label, _, reason, err := normalizeSelectionPresentation(
			option.Label,
			"",
			option.Disabled,
			option.DisabledReason,
		)
		if err != nil {
			return nil, "", err
		}
		option.Label = label.text
		option.DisabledReason = reason
		normalized[index] = selectionOptionBehavior{
			option: option,
			label:  label,
		}
		if !option.Disabled && firstEnabled == "" {
			firstEnabled = option.Value
		}
		if option.Value == requested && !option.Disabled {
			selectedEnabled = true
		}
	}
	if requested != "" && !seen[requested] {
		requested = ""
	}
	if requested != "" && !selectedEnabled {
		return nil, "", fmt.Errorf(
			"%w: selected option is disabled",
			ErrInvalidControl,
		)
	}
	if requested == "" {
		requested = firstEnabled
	}
	return normalized, requested, nil
}

// State returns the current Checkbox state.
func (c *Checkbox) State() CheckState {
	if c == nil || c.state == nil || c.state.app == nil {
		return ""
	}
	c.state.app.mu.RLock()
	defer c.state.app.mu.RUnlock()
	behavior, ok := c.state.behavior.(checkboxBehavior)
	if !ok || c.state.aborted {
		return ""
	}
	return behavior.state
}

// SetState atomically replaces the Checkbox state without notifying its
// change command.
func (c *Checkbox) SetState(state CheckState) error {
	if c == nil || c.state == nil || c.state.app == nil {
		return ErrInvalidControl
	}
	tx := c.state.app.NewTransaction()
	if err := tx.SetCheckState(c, state); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Value returns the RadioGroup's current stable value, or empty.
func (g *RadioGroup) Value() string {
	if g == nil || g.state == nil || g.state.app == nil {
		return ""
	}
	g.state.app.mu.RLock()
	defer g.state.app.mu.RUnlock()
	behavior, ok := g.state.behavior.(radioGroupBehavior)
	if !ok || g.state.aborted {
		return ""
	}
	return behavior.value
}

// SetValue atomically replaces the RadioGroup value without notifying its
// change command.
func (g *RadioGroup) SetValue(value string) error {
	if g == nil || g.state == nil || g.state.app == nil {
		return ErrInvalidControl
	}
	tx := g.state.app.NewTransaction()
	if err := tx.SetRadioValue(g, value); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Value returns the RadioButton's stable option value.
func (b *RadioButton) Value() string {
	if b == nil || b.state == nil || b.state.app == nil {
		return ""
	}
	b.state.app.mu.RLock()
	defer b.state.app.mu.RUnlock()
	behavior, ok := b.state.behavior.(radioButtonBehavior)
	if !ok || b.state.aborted {
		return ""
	}
	return behavior.value
}

// Selected reports whether this RadioButton is selected by its group.
func (b *RadioButton) Selected() bool {
	if b == nil || b.state == nil || b.state.app == nil {
		return false
	}
	b.state.app.mu.RLock()
	defer b.state.app.mu.RUnlock()
	return b.state.app.radioSelectedLocked(b.state)
}

// Options returns a caller-owned CycleField option slice.
func (f *CycleField) Options() []SelectionOption {
	return choiceOptions(f.controlState())
}

// Options returns a caller-owned SelectField option slice.
func (f *SelectField) Options() []SelectionOption {
	return choiceOptions(f.controlState())
}

func choiceOptions(state *controlState) []SelectionOption {
	if state == nil || state.app == nil {
		return nil
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(choiceFieldBehavior)
	if !ok || state.aborted {
		return nil
	}
	options := make([]SelectionOption, len(behavior.options))
	for index := range behavior.options {
		options[index] = behavior.options[index].option
	}
	return options
}

// Value returns the CycleField's selected stable value.
func (f *CycleField) Value() string { return choiceValue(f.controlState()) }

// Value returns the SelectField's selected stable value.
func (f *SelectField) Value() string { return choiceValue(f.controlState()) }

func choiceValue(state *controlState) string {
	if state == nil || state.app == nil {
		return ""
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(choiceFieldBehavior)
	if !ok || state.aborted {
		return ""
	}
	return behavior.value
}

// SetValue atomically changes the CycleField value without notification.
func (f *CycleField) SetValue(value string) error {
	return setChoiceValue(f, value)
}

// SetValue atomically changes the SelectField value without notification.
func (f *SelectField) SetValue(value string) error {
	return setChoiceValue(f, value)
}

func setChoiceValue(control Control, value string) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	tx := state.app.NewTransaction()
	if err := tx.SetChoiceValue(control, value); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// SetOptions atomically replaces the CycleField option model.
func (f *CycleField) SetOptions(
	options []SelectionOption,
	value string,
) error {
	return setChoiceOptions(f, options, value)
}

// SetOptions atomically replaces the SelectField option model.
func (f *SelectField) SetOptions(
	options []SelectionOption,
	value string,
) error {
	return setChoiceOptions(f, options, value)
}

func setChoiceOptions(
	control Control,
	options []SelectionOption,
	value string,
) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	tx := state.app.NewTransaction()
	if err := tx.SetChoiceOptions(control, options, value); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Focus gives this eligible Checkbox keyboard focus.
func (c *Checkbox) Focus() error { return focusSelectionControl(c) }

// Focus gives this eligible RadioButton keyboard focus.
func (b *RadioButton) Focus() error { return focusSelectionControl(b) }

// Focus gives this eligible CycleField keyboard focus.
func (f *CycleField) Focus() error { return focusSelectionControl(f) }

// Focus gives this eligible SelectField keyboard focus.
func (f *SelectField) Focus() error { return focusSelectionControl(f) }

func focusSelectionControl(control Control) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	tx := state.app.NewTransaction()
	if err := tx.SetFocus(control); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Activate applies one Checkbox user-selection transition.
func (c *Checkbox) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateSelectionHandle(ctx, source, requestID, c)
}

// Activate selects this RadioButton.
func (b *RadioButton) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateSelectionHandle(ctx, source, requestID, b)
}

// Activate advances this CycleField to its next enabled option.
func (f *CycleField) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateSelectionHandle(ctx, source, requestID, f)
}

// Activate advances this SelectField to its next enabled option.
func (f *SelectField) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateSelectionHandle(ctx, source, requestID, f)
}

func activateSelectionHandle(
	ctx context.Context,
	source string,
	requestID string,
	control Control,
) (Completion, error) {
	if control == nil || control.controlState() == nil {
		return Completion{}, ErrInvalidControl
	}
	return control.controlState().app.activateSelection(
		ctx,
		source,
		requestID,
		control.controlState(),
	)
}

type selectionAction uint8

const (
	selectionActivate selectionAction = iota + 1
	selectionPrevious
	selectionNext
	selectionFirst
	selectionLast
)

func (a *App) activateSelection(
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
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	if state == nil || state.app != a || !a.focusEligibleLocked(state) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	command, target, changed := a.applySelectionLocked(
		state,
		selectionActivate,
	)
	result := CommandResult{Outcome: OutcomeNoOp}
	var router CommandRouter
	var execute bool
	if changed {
		result.Outcome = OutcomeApplied
	}
	if command != "" {
		router, result, execute = a.resolveCommandLocked(command, true)
	}
	a.mu.Unlock()

	if execute {
		result = a.callRouter(ctx, router, Command{
			ID: command, Target: target, Source: source,
		})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	return a.associateLocked(requestID, result, command), nil
}

func (a *App) applySelectionLocked(
	state *controlState,
	action selectionAction,
) (CommandID, ControlID, bool) {
	if state == nil || !a.focusEligibleLocked(state) {
		return "", "", false
	}
	switch behavior := state.behavior.(type) {
	case checkboxBehavior:
		if action != selectionActivate {
			return "", "", false
		}
		switch behavior.state {
		case CheckUnchecked:
			behavior.state = CheckChecked
		case CheckChecked:
			if behavior.threeState {
				behavior.state = CheckIndeterminate
			} else {
				behavior.state = CheckUnchecked
			}
		case CheckIndeterminate:
			behavior.state = CheckUnchecked
		}
		state.behavior = behavior
		return behavior.changeCommand, state.id, true
	case radioButtonBehavior:
		if action != selectionActivate {
			return "", "", false
		}
		groupState := state.parent
		if groupState == nil {
			return "", "", false
		}
		group, ok := groupState.behavior.(radioGroupBehavior)
		if !ok || group.disabled {
			return "", "", false
		}
		valueChanged := group.value != behavior.value
		if valueChanged {
			group.value = behavior.value
			groupState.behavior = group
		}
		if !valueChanged {
			return "", groupState.id, false
		}
		return group.changeCommand, groupState.id, true
	case choiceFieldBehavior:
		next := choiceTargetValue(behavior, action)
		if next == behavior.value {
			return "", state.id, false
		}
		behavior.value = next
		state.behavior = behavior
		return behavior.changeCommand, state.id, true
	default:
		return "", "", false
	}
}

func (a *App) radioTargetLocked(
	current *controlState,
	action selectionAction,
) *controlState {
	if current == nil || current.parent == nil {
		return nil
	}
	enabled := make([]*controlState, 0, len(current.parent.children))
	for _, child := range current.parent.children {
		if a.radioButtonEnabledLocked(child) {
			enabled = append(enabled, child)
		}
	}
	if len(enabled) == 0 {
		return nil
	}
	index := 0
	for candidate, child := range enabled {
		if child == current {
			index = candidate
			break
		}
	}
	switch action {
	case selectionFirst:
		return enabled[0]
	case selectionLast:
		return enabled[len(enabled)-1]
	case selectionPrevious:
		return enabled[(index+len(enabled)-1)%len(enabled)]
	case selectionNext:
		return enabled[(index+1)%len(enabled)]
	default:
		return current
	}
}

func (a *App) moveRadioBoundaryFocusLocked(key Key) bool {
	if a.focus == nil || a.focus.kind != ControlRadioButton {
		return false
	}
	action := selectionFirst
	if key == KeyEnd {
		action = selectionLast
	} else if key != KeyHome {
		return false
	}
	target := a.radioTargetLocked(a.focus, action)
	if target == nil || target == a.focus {
		return false
	}
	a.focus = target
	a.clearInvalidPressesLocked()
	return true
}

func choiceTargetValue(
	behavior choiceFieldBehavior,
	action selectionAction,
) string {
	enabled := make([]string, 0, len(behavior.options))
	for _, option := range behavior.options {
		if !option.option.Disabled {
			enabled = append(enabled, option.option.Value)
		}
	}
	if len(enabled) == 0 {
		return ""
	}
	index := 0
	found := false
	for candidate, value := range enabled {
		if value == behavior.value {
			index = candidate
			found = true
			break
		}
	}
	switch action {
	case selectionFirst:
		return enabled[0]
	case selectionLast:
		return enabled[len(enabled)-1]
	case selectionPrevious:
		if !found {
			return enabled[0]
		}
		if index == 0 {
			return enabled[0]
		}
		return enabled[index-1]
	case selectionNext:
		if !found {
			return enabled[0]
		}
		if index == len(enabled)-1 {
			return enabled[index]
		}
		return enabled[index+1]
	case selectionActivate:
		if !found {
			return enabled[0]
		}
		return enabled[(index+1)%len(enabled)]
	default:
		return behavior.value
	}
}

func selectionActionForKey(key Key) (selectionAction, bool) {
	switch key {
	case KeySpace, KeyEnter:
		return selectionActivate, true
	case KeyLeftBracket:
		return selectionPrevious, true
	case KeyRightBracket:
		return selectionNext, true
	default:
		return 0, false
	}
}

func (a *App) selectionKeyActionLocked(
	state *controlState,
	key Key,
) (selectionAction, bool) {
	if !a.focusEligibleLocked(state) {
		return 0, false
	}
	action, recognized := selectionActionForKey(key)
	if !recognized {
		return 0, false
	}
	switch state.behavior.(type) {
	case checkboxBehavior:
		return action, action == selectionActivate && key == KeySpace
	case radioButtonBehavior:
		return action, action == selectionActivate
	case choiceFieldBehavior:
		return action,
			action == selectionActivate ||
				action == selectionPrevious ||
				action == selectionNext
	default:
		return 0, false
	}
}

func (b checkboxBehavior) clientInset() int { return 0 }
func (b checkboxBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"selection.mnemonic",
		"selection.focused",
		"selection.focused_mnemonic",
		"selection.disabled",
	}
}
func (b checkboxBehavior) intrinsicMinimum() Size {
	return Size{Width: b.label.cells + 4, Height: 1}
}
func (b checkboxBehavior) details() ControlDetails {
	return ControlDetails{
		Version:  ControlDetailsVersion,
		Checkbox: &CheckboxDetails{},
	}
}
func (b checkboxBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	marker := " "
	switch b.state {
	case CheckChecked:
		marker = "X"
	case CheckIndeterminate:
		marker = "-"
	}
	cells := []string{"[", marker, "]", " "}
	cells = append(cells, b.label.lines[0]...)
	app.paintSelectionCellsLocked(
		frame,
		state,
		absolute,
		clip,
		cells,
		4,
		b.label.lines[0],
		b.mnemonic,
		!b.disabled,
	)
}

func (b radioGroupBehavior) clientInset() int { return 0 }
func (b radioGroupBehavior) details() ControlDetails {
	return ControlDetails{
		Version:    ControlDetailsVersion,
		Container:  &ContainerDetails{},
		RadioGroup: &RadioGroupDetails{Options: []RadioOptionDetails{}},
	}
}
func (radioGroupBehavior) paintDecoration(
	*App,
	*IntendedFrame,
	*controlState,
	Rect,
	Rect,
) {
}

func (b radioButtonBehavior) clientInset() int { return 0 }
func (b radioButtonBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"selection.mnemonic",
		"selection.focused",
		"selection.focused_mnemonic",
		"selection.disabled",
	}
}
func (b radioButtonBehavior) intrinsicMinimum() Size {
	return Size{Width: b.label.cells + 4, Height: 1}
}
func (b radioButtonBehavior) details() ControlDetails {
	return ControlDetails{
		Version:     ControlDetailsVersion,
		RadioButton: &RadioButtonDetails{},
	}
}
func (b radioButtonBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	marker := " "
	if app.radioSelectedLocked(state) {
		marker = "•"
	}
	cells := []string{"(", marker, ")", " "}
	cells = append(cells, b.label.lines[0]...)
	app.paintSelectionCellsLocked(
		frame,
		state,
		absolute,
		clip,
		cells,
		4,
		b.label.lines[0],
		b.mnemonic,
		app.radioButtonEnabledLocked(state),
	)
}

func (b choiceFieldBehavior) clientInset() int { return 0 }
func (b choiceFieldBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"selection.mnemonic",
		"selection.focused",
		"selection.focused_mnemonic",
		"selection.disabled",
	}
}
func (b choiceFieldBehavior) intrinsicMinimum() Size {
	maximum := 2
	for _, option := range b.options {
		maximum = max(maximum, option.label.cells)
	}
	return Size{Width: b.label.cells + maximum + 7, Height: 1}
}
func (b choiceFieldBehavior) details() ControlDetails {
	return ControlDetails{
		Version:     ControlDetailsVersion,
		ChoiceField: &ChoiceFieldDetails{Options: []SelectionOptionDetails{}},
	}
}
func (b choiceFieldBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	value := []string{"-", "-"}
	for _, option := range b.options {
		if option.option.Value == b.value {
			value = option.label.lines[0]
			break
		}
	}
	cells := append([]string{}, b.label.lines[0]...)
	cells = append(cells, " ", " ", "◄", " ")
	cells = append(cells, value...)
	cells = append(cells, " ", "►")
	app.paintSelectionCellsLocked(
		frame,
		state,
		absolute,
		clip,
		cells,
		0,
		b.label.lines[0],
		b.mnemonic,
		app.choiceFieldEnabledLocked(state),
	)
}

func (a *App) paintSelectionCellsLocked(
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
	cells []string,
	labelOffset int,
	labelCells []string,
	mnemonic Key,
	enabled bool,
) {
	style := state.style
	mnemonicStyle := StyleID("selection.mnemonic")
	switch {
	case !enabled:
		style = "selection.disabled"
		mnemonicStyle = style
	case a.focus == state:
		style = "selection.focused"
		mnemonicStyle = "selection.focused_mnemonic"
	}
	y := absolute.Y + max(0, (absolute.Height-1)/2)
	for index, cell := range cells {
		a.setClippedCellLocked(
			frame,
			clip,
			absolute.X+index,
			y,
			cell,
			style,
			a.styles[style],
			state.id,
		)
	}
	if mnemonic == "" {
		return
	}
	for index, cell := range labelCells {
		if Key(strings.ToLower(cell)) != mnemonic {
			continue
		}
		a.setClippedCellLocked(
			frame,
			clip,
			absolute.X+labelOffset+index,
			y,
			cell,
			mnemonicStyle,
			a.styles[mnemonicStyle],
			state.id,
		)
		return
	}
}

func cloneSelectionOptionBehaviors(
	options []selectionOptionBehavior,
) []selectionOptionBehavior {
	return append([]selectionOptionBehavior(nil), options...)
}

func selectionBehaviorEqual(left, right controlBehavior) bool {
	switch leftValue := left.(type) {
	case checkboxBehavior:
		rightValue, ok := right.(checkboxBehavior)
		return ok && leftValue.label.text == rightValue.label.text &&
			leftValue.mnemonic == rightValue.mnemonic &&
			leftValue.state == rightValue.state &&
			leftValue.threeState == rightValue.threeState &&
			leftValue.disabled == rightValue.disabled &&
			leftValue.disabledReason == rightValue.disabledReason &&
			leftValue.changeCommand == rightValue.changeCommand
	case radioGroupBehavior:
		rightValue, ok := right.(radioGroupBehavior)
		return ok && leftValue == rightValue
	case radioButtonBehavior:
		rightValue, ok := right.(radioButtonBehavior)
		return ok && leftValue.value == rightValue.value &&
			leftValue.label.text == rightValue.label.text &&
			leftValue.mnemonic == rightValue.mnemonic &&
			leftValue.initialSelected == rightValue.initialSelected &&
			leftValue.disabled == rightValue.disabled &&
			leftValue.disabledReason == rightValue.disabledReason
	case choiceFieldBehavior:
		rightValue, ok := right.(choiceFieldBehavior)
		if !ok || leftValue.label.text != rightValue.label.text ||
			leftValue.mnemonic != rightValue.mnemonic ||
			leftValue.value != rightValue.value ||
			leftValue.disabled != rightValue.disabled ||
			leftValue.disabledReason != rightValue.disabledReason ||
			leftValue.changeCommand != rightValue.changeCommand ||
			len(leftValue.options) != len(rightValue.options) {
			return false
		}
		for index := range leftValue.options {
			if leftValue.options[index].option !=
				rightValue.options[index].option {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (a *App) radioSelectedLocked(state *controlState) bool {
	if state == nil || state.kind != ControlRadioButton ||
		state.parent == nil {
		return false
	}
	button, ok := state.behavior.(radioButtonBehavior)
	if !ok {
		return false
	}
	group, ok := state.parent.behavior.(radioGroupBehavior)
	return ok && group.value == button.value
}

func (a *App) radioButtonEnabledLocked(state *controlState) bool {
	if state == nil || state.kind != ControlRadioButton ||
		state.parent == nil || !a.effectivelyVisibleLocked(state) {
		return false
	}
	button, buttonOK := state.behavior.(radioButtonBehavior)
	group, groupOK := state.parent.behavior.(radioGroupBehavior)
	return buttonOK && groupOK && !button.disabled && !group.disabled
}

func (a *App) choiceFieldEnabledLocked(state *controlState) bool {
	if state == nil || !a.effectivelyVisibleLocked(state) {
		return false
	}
	behavior, ok := state.behavior.(choiceFieldBehavior)
	if !ok || behavior.disabled {
		return false
	}
	for _, option := range behavior.options {
		if !option.option.Disabled {
			return true
		}
	}
	return false
}

func choiceSelectedIndex(behavior choiceFieldBehavior) int {
	for index, option := range behavior.options {
		if option.option.Value == behavior.value {
			return index
		}
	}
	return -1
}

func (a *App) checkboxDetailsLocked(
	behavior checkboxBehavior,
) CheckboxDetails {
	return CheckboxDetails{
		Label: behavior.label.text, State: behavior.state,
		ThreeState: behavior.threeState, Enabled: !behavior.disabled,
		DisabledReason: behavior.disabledReason, Mnemonic: behavior.mnemonic,
		ChangeCommand: behavior.changeCommand,
	}
}

func (a *App) radioButtonDetailsLocked(
	state *controlState,
	behavior radioButtonBehavior,
) RadioButtonDetails {
	enabled := !behavior.disabled
	reason := behavior.disabledReason
	if group, ok := state.parent.behavior.(radioGroupBehavior); ok &&
		group.disabled {
		enabled = false
		reason = group.disabledReason
	}
	return RadioButtonDetails{
		Value: behavior.value, Label: behavior.label.text,
		Selected: a.radioSelectedLocked(state), Enabled: enabled,
		DisabledReason: reason, Mnemonic: behavior.mnemonic,
	}
}

func (a *App) radioGroupDetailsLocked(
	state *controlState,
	behavior radioGroupBehavior,
) RadioGroupDetails {
	details := RadioGroupDetails{
		Value: behavior.value, AllowEmpty: behavior.allowEmpty,
		Enabled: !behavior.disabled, DisabledReason: behavior.disabledReason,
		ChangeCommand: behavior.changeCommand,
		Options:       make([]RadioOptionDetails, 0, len(state.children)),
	}
	for _, child := range state.children {
		button, ok := child.behavior.(radioButtonBehavior)
		if !ok || child.destroyed {
			continue
		}
		enabled := !behavior.disabled && !button.disabled
		reason := button.disabledReason
		if behavior.disabled {
			reason = behavior.disabledReason
		}
		details.Options = append(details.Options, RadioOptionDetails{
			Control: child.id, Value: button.value, Label: button.label.text,
			Selected: behavior.value == button.value, Enabled: enabled,
			DisabledReason: reason,
		})
	}
	return details
}

func (a *App) choiceFieldDetailsLocked(
	behavior choiceFieldBehavior,
) ChoiceFieldDetails {
	details := ChoiceFieldDetails{
		Label: behavior.label.text, Value: behavior.value,
		SelectedIndex: choiceSelectedIndex(behavior),
		Enabled:       !behavior.disabled, DisabledReason: behavior.disabledReason,
		Mnemonic: behavior.mnemonic, ChangeCommand: behavior.changeCommand,
		Options: make(
			[]SelectionOptionDetails,
			len(behavior.options),
		),
	}
	for index, option := range behavior.options {
		details.Options[index] = SelectionOptionDetails{
			Value: option.option.Value, Label: option.option.Label,
			Enabled:        !option.option.Disabled,
			DisabledReason: option.option.DisabledReason,
			Selected:       option.option.Value == behavior.value,
		}
	}
	return details
}

func (t *Transaction) selectionMutationBehaviorLocked(
	mutation *transactionMutation,
	base controlBehavior,
	destroyed map[*controlState]bool,
) (controlBehavior, error) {
	switch mutation.kind {
	case mutationCheckState:
		behavior, ok := base.(checkboxBehavior)
		if !ok {
			return nil, ErrInvalidControl
		}
		state := CheckState(mutation.selectionValue)
		if err := validateCheckState(state, behavior.threeState); err != nil {
			return nil, err
		}
		behavior.state = state
		return behavior, nil
	case mutationRadioValue:
		behavior, ok := base.(radioGroupBehavior)
		if !ok {
			return nil, ErrInvalidControl
		}
		value, err := t.validatedRadioValueLocked(
			mutation.state,
			behavior,
			mutation.selectionValue,
			destroyed,
		)
		if err != nil {
			return nil, err
		}
		behavior.value = value
		return behavior, nil
	case mutationChoiceValue:
		behavior, ok := base.(choiceFieldBehavior)
		if !ok {
			return nil, ErrInvalidControl
		}
		value, err := validatedChoiceValue(
			behavior.options,
			mutation.selectionValue,
		)
		if err != nil {
			return nil, err
		}
		behavior.value = value
		return behavior, nil
	case mutationChoiceOptions:
		behavior, ok := base.(choiceFieldBehavior)
		if !ok {
			return nil, ErrInvalidControl
		}
		options, value, err := normalizeSelectionOptions(
			mutation.selectionOptions,
			mutation.selectionValue,
		)
		if err != nil {
			return nil, err
		}
		behavior.options = options
		behavior.value = value
		return behavior, nil
	default:
		return nil, ErrInvalidControl
	}
}

func validatedChoiceValue(
	options []selectionOptionBehavior,
	requested string,
) (string, error) {
	firstEnabled := ""
	for _, option := range options {
		if option.option.Disabled {
			if option.option.Value == requested {
				return "", fmt.Errorf(
					"%w: selected option is disabled",
					ErrInvalidControl,
				)
			}
			continue
		}
		if firstEnabled == "" {
			firstEnabled = option.option.Value
		}
		if option.option.Value == requested {
			return requested, nil
		}
	}
	if requested == "" {
		return firstEnabled, nil
	}
	return "", fmt.Errorf("%w: unknown selection value", ErrInvalidControl)
}

func (t *Transaction) effectiveRadioChildrenLocked(
	group *controlState,
	destroyed map[*controlState]bool,
) []*controlState {
	children := make([]*controlState, 0, len(group.children)+len(t.creates))
	for _, child := range group.children {
		if !destroyed[child] && !child.destroyed {
			children = append(children, child)
		}
	}
	for _, child := range t.creates {
		if child.parent == group && !destroyed[child] {
			children = append(children, child)
		}
	}
	return children
}

func (t *Transaction) validatedRadioValueLocked(
	group *controlState,
	behavior radioGroupBehavior,
	requested string,
	destroyed map[*controlState]bool,
) (string, error) {
	firstEnabled := ""
	for _, child := range t.effectiveRadioChildrenLocked(group, destroyed) {
		button, ok := child.behavior.(radioButtonBehavior)
		if !ok || button.disabled {
			continue
		}
		if firstEnabled == "" {
			firstEnabled = button.value
		}
		if button.value == requested {
			return requested, nil
		}
	}
	if requested == "" {
		if behavior.allowEmpty {
			return "", nil
		}
		return firstEnabled, nil
	}
	return "", fmt.Errorf(
		"%w: unknown or disabled RadioButton value",
		ErrInvalidControl,
	)
}

func (a *App) repairRadioGroupsLocked() bool {
	changed := false
	for _, state := range a.controlsByID {
		if state.destroyed || state.kind != ControlRadioGroup {
			continue
		}
		group := state.behavior.(radioGroupBehavior)
		selectedExists := false
		firstEnabled := ""
		initialSelected := ""
		for _, child := range state.children {
			button, ok := child.behavior.(radioButtonBehavior)
			if !ok || child.destroyed {
				continue
			}
			if button.initialSelected && !button.disabled {
				initialSelected = button.value
			}
			if !button.disabled && firstEnabled == "" {
				firstEnabled = button.value
			}
			if !button.disabled && button.value == group.value {
				selectedExists = true
			}
			if button.initialSelected {
				button.initialSelected = false
				child.behavior = button
			}
		}
		next := group.value
		switch {
		case initialSelected != "":
			next = initialSelected
		case selectedExists:
		case group.allowEmpty && group.value == "":
			next = ""
		default:
			next = firstEnabled
		}
		if group.value != next {
			group.value = next
			state.behavior = group
			changed = true
		}
	}
	return changed
}

var errSelectionNoChange = errors.New("expletives: selection did not change")
