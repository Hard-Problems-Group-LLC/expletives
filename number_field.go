package expletives

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const MaxNumberDecimalPlaces = 9

// NumberFieldOptions configures one bounded decimal editor. Minimum and
// Maximum are copied when construction is recorded. DecimalPlaces defaults
// to zero and may be at most MaxNumberDecimalPlaces.
type NumberFieldOptions struct {
	PanelOptions
	Value          float64
	Minimum        *float64
	Maximum        *float64
	DecimalPlaces  int
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// SpinBoxOptions configures one NumberField with bracket-key stepping. A zero
// Step selects one unit at the configured decimal precision.
type SpinBoxOptions struct {
	PanelOptions
	Value          float64
	Minimum        *float64
	Maximum        *float64
	DecimalPlaces  int
	Step           float64
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// NumberField is a copy-safe focusable decimal editor leaf.
type NumberField struct{ controlHandle }

// SpinBox is a copy-safe NumberField specialization with bounded stepping.
type SpinBox struct{ controlHandle }

type numberPolicy struct {
	minimum       float64
	maximum       float64
	hasMinimum    bool
	hasMaximum    bool
	decimalPlaces int
	step          float64
	spin          bool
}

type numberFieldBehavior struct {
	editor textFieldBehavior
	policy numberPolicy
	value  float64
}

// NewNumberField constructs and atomically inserts a NumberField.
func NewNumberField(
	parent Container,
	options NumberFieldOptions,
) (*NumberField, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	field, err := tx.NewNumberField(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return field, nil
}

// NewSpinBox constructs and atomically inserts a SpinBox.
func NewSpinBox(parent Container, options SpinBoxOptions) (*SpinBox, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	spin, err := tx.NewSpinBox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return spin, nil
}

// NewNumberField records construction of a provisional NumberField.
func (t *Transaction) NewNumberField(
	parent Container,
	options NumberFieldOptions,
) (*NumberField, error) {
	behavior, err := newNumberFieldBehavior(
		options.Value,
		options.Minimum,
		options.Maximum,
		options.DecimalPlaces,
		0,
		false,
		options.Disabled,
		options.DisabledReason,
		options.ChangeCommand,
	)
	if err != nil {
		return nil, err
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlNumberField,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	field := &NumberField{controlHandle: controlHandle{state: state}}
	state.control = field
	return field, nil
}

// NewSpinBox records construction of a provisional SpinBox.
func (t *Transaction) NewSpinBox(
	parent Container,
	options SpinBoxOptions,
) (*SpinBox, error) {
	behavior, err := newNumberFieldBehavior(
		options.Value,
		options.Minimum,
		options.Maximum,
		options.DecimalPlaces,
		options.Step,
		true,
		options.Disabled,
		options.DisabledReason,
		options.ChangeCommand,
	)
	if err != nil {
		return nil, err
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlSpinBox,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	spin := &SpinBox{controlHandle: controlHandle{state: state}}
	state.control = spin
	return spin, nil
}

func newNumberFieldBehavior(
	value float64,
	minimum *float64,
	maximum *float64,
	decimalPlaces int,
	step float64,
	spin bool,
	disabled bool,
	disabledReason string,
	changeCommand CommandID,
) (numberFieldBehavior, error) {
	policy, err := normalizeNumberPolicy(
		minimum,
		maximum,
		decimalPlaces,
		step,
		spin,
	)
	if err != nil {
		return numberFieldBehavior{}, err
	}
	reason, err := normalizeDisabledReason(disabled, disabledReason)
	if err != nil {
		return numberFieldBehavior{}, err
	}
	if err := validateOptionalCommand(changeCommand); err != nil {
		return numberFieldBehavior{}, err
	}
	value, text, err := normalizeNumberValue(value, policy)
	if err != nil {
		return numberFieldBehavior{}, err
	}
	normalized, err := normalizeInputText(text)
	if err != nil {
		return numberFieldBehavior{}, err
	}
	characters := "0123456789-"
	if decimalPlaces > 0 {
		characters += "."
	}
	validator, err := normalizeTextValidator(&TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationWhitelist,
		Characters:  characters,
	})
	if err != nil {
		return numberFieldBehavior{}, err
	}
	return numberFieldBehavior{
		editor: textFieldBehavior{
			committed: normalized,
			working:   cloneInputText(normalized),
			validator: validator,
			disabled:  disabled, disabledReason: reason,
			changeCommand: changeCommand, caret: len(normalized.cells),
			selectionAnchor: -1,
		},
		policy: policy,
		value:  value,
	}, nil
}

func normalizeNumberPolicy(
	minimum *float64,
	maximum *float64,
	decimalPlaces int,
	step float64,
	spin bool,
) (numberPolicy, error) {
	if decimalPlaces < 0 || decimalPlaces > MaxNumberDecimalPlaces {
		return numberPolicy{}, fmt.Errorf(
			"%w: decimal places must be from 0 through %d",
			ErrValidation,
			MaxNumberDecimalPlaces,
		)
	}
	policy := numberPolicy{decimalPlaces: decimalPlaces, spin: spin}
	if minimum != nil {
		if !finiteNumber(*minimum) {
			return numberPolicy{}, fmt.Errorf(
				"%w: minimum must be finite",
				ErrValidation,
			)
		}
		policy.minimum = canonicalNumber(*minimum, decimalPlaces)
		if policy.minimum != *minimum {
			return numberPolicy{}, fmt.Errorf(
				"%w: minimum exceeds configured precision",
				ErrValidation,
			)
		}
		policy.hasMinimum = true
	}
	if maximum != nil {
		if !finiteNumber(*maximum) {
			return numberPolicy{}, fmt.Errorf(
				"%w: maximum must be finite",
				ErrValidation,
			)
		}
		policy.maximum = canonicalNumber(*maximum, decimalPlaces)
		if policy.maximum != *maximum {
			return numberPolicy{}, fmt.Errorf(
				"%w: maximum exceeds configured precision",
				ErrValidation,
			)
		}
		policy.hasMaximum = true
	}
	if policy.hasMinimum && policy.hasMaximum &&
		policy.minimum > policy.maximum {
		return numberPolicy{}, fmt.Errorf(
			"%w: minimum exceeds maximum",
			ErrValidation,
		)
	}
	if spin {
		if step == 0 {
			step = 1
		}
		if !finiteNumber(step) || step <= 0 ||
			canonicalNumber(step, decimalPlaces) != step {
			return numberPolicy{}, fmt.Errorf(
				"%w: step must be positive, finite, and fit configured precision",
				ErrValidation,
			)
		}
		policy.step = step
	} else if step != 0 {
		return numberPolicy{}, fmt.Errorf(
			"%w: NumberField cannot define a step",
			ErrValidation,
		)
	}
	return policy, nil
}

func finiteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func canonicalNumber(value float64, decimalPlaces int) float64 {
	factor := math.Pow10(decimalPlaces)
	canonical := math.Round(value*factor) / factor
	if canonical == 0 {
		return 0
	}
	return canonical
}

func formatNumber(value float64, decimalPlaces int) string {
	return strconv.FormatFloat(value, 'f', decimalPlaces, 64)
}

func normalizeNumberValue(
	value float64,
	policy numberPolicy,
) (float64, string, error) {
	if !finiteNumber(value) {
		return 0, "", fmt.Errorf("%w: numeric value must be finite", ErrValidation)
	}
	canonical := canonicalNumber(value, policy.decimalPlaces)
	if canonical != value {
		return 0, "", fmt.Errorf(
			"%w: numeric value exceeds configured precision",
			ErrValidation,
		)
	}
	if policy.hasMinimum && canonical < policy.minimum {
		return 0, "", fmt.Errorf("%w: numeric value is below minimum", ErrValidation)
	}
	if policy.hasMaximum && canonical > policy.maximum {
		return 0, "", fmt.Errorf("%w: numeric value is above maximum", ErrValidation)
	}
	return canonical, formatNumber(canonical, policy.decimalPlaces), nil
}

func parseNumberText(
	text string,
	policy numberPolicy,
) (float64, string, bool) {
	if text == "" || text == "-" || text == "." || text == "-." ||
		strings.HasPrefix(text, "+") || strings.Count(text, ".") > 1 {
		return 0, "Enter a complete decimal number", false
	}
	if point := strings.IndexByte(text, '.'); point >= 0 &&
		len(text)-point-1 > policy.decimalPlaces {
		return 0, "Too many decimal places", false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || !finiteNumber(value) {
		return 0, "Enter a complete decimal number", false
	}
	if canonicalNumber(value, policy.decimalPlaces) != value {
		return 0, "Too many decimal places", false
	}
	if policy.hasMinimum && value < policy.minimum {
		return 0, "Value is below the minimum", false
	}
	if policy.hasMaximum && value > policy.maximum {
		return 0, "Value is above the maximum", false
	}
	return value, "", true
}

func (b numberFieldBehavior) current() normalizedInputText {
	return b.editor.current()
}

func (numberFieldBehavior) clientInset() int { return 0 }

func (b numberFieldBehavior) additionalStyles() []StyleID {
	styles := b.editor.additionalStyles()
	if b.policy.spin {
		styles = append(
			styles,
			"spin_box.button",
			"spin_box.button_focused",
			"spin_box.button_disabled",
		)
	}
	return styles
}

func (b numberFieldBehavior) intrinsicMinimum() Size {
	if b.policy.spin {
		return Size{Width: 10, Height: 1}
	}
	return Size{Width: 8, Height: 1}
}

func (numberFieldBehavior) details() ControlDetails {
	return ControlDetails{
		Version:     ControlDetailsVersion,
		NumberField: &NumberFieldDetails{},
	}
}

func (b numberFieldBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	_, _, valid := parseNumberText(b.current().text, b.policy)
	editorBounds := absolute
	buttonCount := 0
	if b.policy.spin {
		buttonCount = min(2, absolute.Width)
		editorBounds.Width -= buttonCount
	}
	paintTextEditor(
		app,
		frame,
		state,
		editorBounds,
		clip,
		b.editor,
		!valid,
	)
	if buttonCount == 0 {
		return
	}
	style := StyleID("spin_box.button")
	if b.editor.disabled {
		style = "spin_box.button_disabled"
	} else if app.focus == state {
		style = "spin_box.button_focused"
	}
	start := absolute.X + absolute.Width - buttonCount
	if buttonCount == 1 {
		app.setClippedCellLocked(
			frame,
			clip,
			start,
			absolute.Y,
			"↕",
			style,
			app.styles[style],
			state.id,
		)
		return
	}
	for index, glyph := range []string{"▼", "▲"} {
		app.setClippedCellLocked(
			frame,
			clip,
			start+index,
			absolute.Y,
			glyph,
			style,
			app.styles[style],
			state.id,
		)
	}
}

func numberFieldBehaviorForRead(
	state *controlState,
) (numberFieldBehavior, bool) {
	if state == nil || state.app == nil {
		return numberFieldBehavior{}, false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(numberFieldBehavior)
	return behavior, ok && !state.aborted && !state.destroyed
}

// Value returns the committed numeric value.
func (f *NumberField) Value() float64 {
	behavior, ok := numberFieldBehaviorForRead(f.controlState())
	if !ok {
		return 0
	}
	return behavior.value
}

// Value returns the committed numeric value.
func (s *SpinBox) Value() float64 {
	behavior, ok := numberFieldBehaviorForRead(s.controlState())
	if !ok {
		return 0
	}
	return behavior.value
}

// SetValue atomically replaces the committed numeric value without a user
// change notification.
func (f *NumberField) SetValue(value float64) error {
	return setNumberHandleValue(f, value)
}

// SetValue atomically replaces the committed numeric value without a user
// change notification.
func (s *SpinBox) SetValue(value float64) error {
	return setNumberHandleValue(s, value)
}

func setNumberHandleValue(control Control, value float64) error {
	if control == nil || control.controlState() == nil {
		return ErrInvalidControl
	}
	tx := control.controlState().app.NewTransaction()
	if err := tx.SetNumberValue(control, value); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Editing reports whether the NumberField captures printable input.
func (f *NumberField) Editing() bool {
	behavior, ok := numberFieldBehaviorForRead(f.controlState())
	return ok && behavior.editor.editing
}

// Editing reports whether the SpinBox captures printable input.
func (s *SpinBox) Editing() bool {
	behavior, ok := numberFieldBehaviorForRead(s.controlState())
	return ok && behavior.editor.editing
}

// Valid reports whether the current edit or committed value satisfies
// numeric syntax, precision, and range.
func (f *NumberField) Valid() bool {
	return numberHandleValid(f)
}

// Valid reports whether the current edit or committed value satisfies
// numeric syntax, precision, and range.
func (s *SpinBox) Valid() bool {
	return numberHandleValid(s)
}

func numberHandleValid(control Control) bool {
	if control == nil {
		return false
	}
	behavior, ok := numberFieldBehaviorForRead(control.controlState())
	if !ok {
		return false
	}
	_, _, valid := parseNumberText(behavior.current().text, behavior.policy)
	return valid
}

// DecimalPlaces returns the configured fixed display precision.
func (f *NumberField) DecimalPlaces() int {
	return numberHandleDecimalPlaces(f)
}

// DecimalPlaces returns the configured fixed display precision.
func (s *SpinBox) DecimalPlaces() int {
	return numberHandleDecimalPlaces(s)
}

func numberHandleDecimalPlaces(control Control) int {
	if control == nil {
		return 0
	}
	behavior, ok := numberFieldBehaviorForRead(control.controlState())
	if !ok {
		return 0
	}
	return behavior.policy.decimalPlaces
}

// Minimum returns the optional inclusive minimum.
func (f *NumberField) Minimum() (float64, bool) {
	return numberHandleMinimum(f)
}

// Minimum returns the optional inclusive minimum.
func (s *SpinBox) Minimum() (float64, bool) {
	return numberHandleMinimum(s)
}

func numberHandleMinimum(control Control) (float64, bool) {
	if control == nil {
		return 0, false
	}
	behavior, ok := numberFieldBehaviorForRead(control.controlState())
	if !ok {
		return 0, false
	}
	return behavior.policy.minimum, behavior.policy.hasMinimum
}

// Maximum returns the optional inclusive maximum.
func (f *NumberField) Maximum() (float64, bool) {
	return numberHandleMaximum(f)
}

// Maximum returns the optional inclusive maximum.
func (s *SpinBox) Maximum() (float64, bool) {
	return numberHandleMaximum(s)
}

func numberHandleMaximum(control Control) (float64, bool) {
	if control == nil {
		return 0, false
	}
	behavior, ok := numberFieldBehaviorForRead(control.controlState())
	if !ok {
		return 0, false
	}
	return behavior.policy.maximum, behavior.policy.hasMaximum
}

// Focus gives this eligible NumberField keyboard focus.
func (f *NumberField) Focus() error { return focusSelectionControl(f) }

// Focus gives this eligible SpinBox keyboard focus.
func (s *SpinBox) Focus() error { return focusSelectionControl(s) }

// Activate focuses the NumberField and enters edit mode.
func (f *NumberField) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateNumberHandle(ctx, source, requestID, f)
}

// Activate focuses the SpinBox and enters edit mode.
func (s *SpinBox) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activateNumberHandle(ctx, source, requestID, s)
}

func activateNumberHandle(
	ctx context.Context,
	source string,
	requestID string,
	control Control,
) (Completion, error) {
	if control == nil || control.controlState() == nil {
		return Completion{}, ErrInvalidControl
	}
	return control.controlState().app.activateNumberField(
		ctx,
		source,
		requestID,
		control.controlState(),
	)
}

// Step returns the positive configured SpinBox step.
func (s *SpinBox) Step() float64 {
	behavior, ok := numberFieldBehaviorForRead(s.controlState())
	if !ok {
		return 0
	}
	return behavior.policy.step
}

// SetNumberValue records one atomic NumberField or SpinBox value replacement.
func (t *Transaction) SetNumberValue(control Control, value float64) error {
	state, err := t.control(control)
	if err != nil ||
		(state.kind != ControlNumberField && state.kind != ControlSpinBox) {
		return ErrInvalidControl
	}
	behavior, ok := t.recordedControlBehavior(state).(numberFieldBehavior)
	if !ok {
		return ErrInvalidControl
	}
	value, text, err := normalizeNumberValue(value, behavior.policy)
	if err != nil {
		return err
	}
	normalized, err := normalizeInputText(text)
	if err != nil {
		return err
	}
	behavior.value = value
	behavior.editor.committed = normalized
	behavior.editor.working = cloneInputText(normalized)
	behavior.editor.editing = false
	behavior.editor.caret = len(normalized.cells)
	behavior.editor.viewOffset = 0
	behavior.editor.selectionAnchor = -1
	return t.recordNumberFieldBehavior(state, behavior)
}

func (t *Transaction) recordNumberFieldBehavior(
	state *controlState,
	behavior numberFieldBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationNumberField, state: state, behavior: behavior,
	})
	return nil
}

func (a *App) activateNumberField(
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
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	if state == nil || state.app != a || !a.focusEligibleLocked(state) {
		return Completion{}, ErrNotFocusable
	}
	changed := a.focus != state
	if changed {
		if a.commitOrCancelEditorStateLocked(a.focus) {
			changed = true
		}
	}
	a.focus = state
	behavior := state.behavior.(numberFieldBehavior)
	if !behavior.editor.editing {
		behavior.editor.working = cloneInputText(behavior.editor.committed)
		behavior.editor.editing = true
		behavior.editor.caret = len(behavior.editor.working.cells)
		behavior.editor.viewOffset = 0
		behavior.editor.selectionAnchor = -1
		state.behavior = behavior
		changed = true
	}
	outcome := OutcomeNoOp
	if changed {
		outcome = OutcomeApplied
	}
	return a.associateLocked(
		requestID,
		CommandResult{Outcome: outcome},
		"",
	), nil
}

func (a *App) numberFieldDetailsLocked(
	state *controlState,
	behavior numberFieldBehavior,
) NumberFieldDetails {
	current := behavior.current()
	_, invalidReason, valid := parseNumberText(current.text, behavior.policy)
	offset := textViewOffset(
		behavior.editor.viewOffset,
		behavior.editor.caret,
		len(current.cells),
		state.bounds.Width,
	)
	details := NumberFieldDetails{
		Text: current.text, Value: behavior.value,
		Length: len(current.cells), Caret: behavior.editor.caret,
		ViewOffset: offset, Editing: behavior.editor.editing,
		Valid: valid, InvalidReason: invalidReason,
		DecimalPlaces:  behavior.policy.decimalPlaces,
		Enabled:        !behavior.editor.disabled,
		DisabledReason: behavior.editor.disabledReason,
		ChangeCommand:  behavior.editor.changeCommand,
		Step:           behavior.policy.step,
	}
	details.SelectionStart, details.SelectionEnd, _ =
		textSelectionRange(behavior.editor)
	if behavior.policy.hasMinimum {
		minimum := behavior.policy.minimum
		details.Minimum = &minimum
	}
	if behavior.policy.hasMaximum {
		maximum := behavior.policy.maximum
		details.Maximum = &maximum
	}
	return details
}

func (a *App) editorInputLocked(
	state *controlState,
	key Key,
	held map[Key]bool,
) (command CommandID, target ControlID, handled, changed bool) {
	if state == nil {
		return "", "", false, false
	}
	switch behavior := state.behavior.(type) {
	case textFieldBehavior:
		behavior, command, target, handled, changed =
			a.textEditorInputLocked(state, behavior, key, held)
		if changed {
			state.behavior = behavior
		}
		return command, target, handled, changed
	case numberFieldBehavior:
		return a.numberFieldInputLocked(state, behavior, key, held)
	case textAreaBehavior:
		return a.textAreaInputLocked(state, behavior, key, held)
	case comboBoxBehavior:
		return a.comboBoxEditorInputLocked(state, behavior, key, held)
	case dataGridBehavior:
		return a.dataGridEditorInputLocked(state, behavior, key, held)
	default:
		return "", "", false, false
	}
}

func (a *App) numberFieldInputLocked(
	state *controlState,
	behavior numberFieldBehavior,
	key Key,
	held map[Key]bool,
) (command CommandID, target ControlID, handled, changed bool) {
	if state == nil || state != a.focus || !a.focusEligibleLocked(state) {
		return "", "", false, false
	}
	if behavior.policy.spin && !behavior.editor.editing &&
		noHeldModifiers(held) &&
		(key == KeyLeftBracket || key == KeyRightBracket) {
		delta := behavior.policy.step
		if key == KeyLeftBracket {
			delta = -delta
		}
		next := canonicalNumber(
			behavior.value+delta,
			behavior.policy.decimalPlaces,
		)
		if behavior.policy.hasMinimum && next < behavior.policy.minimum {
			next = behavior.policy.minimum
		}
		if behavior.policy.hasMaximum && next > behavior.policy.maximum {
			next = behavior.policy.maximum
		}
		if !finiteNumber(next) || next == behavior.value {
			return "", "", true, false
		}
		normalized, err := normalizeInputText(
			formatNumber(next, behavior.policy.decimalPlaces),
		)
		if err != nil {
			return "", "", true, false
		}
		behavior.value = next
		behavior.editor.committed = normalized
		behavior.editor.working = cloneInputText(normalized)
		behavior.editor.caret = len(normalized.cells)
		behavior.editor.viewOffset = 0
		behavior.editor.selectionAnchor = -1
		state.behavior = behavior
		return behavior.editor.changeCommand, state.id, true, true
	}
	if behavior.editor.editing && key == KeyEnter &&
		textInputModifiers(held) {
		value, _, valid := parseNumberText(
			behavior.editor.working.text,
			behavior.policy,
		)
		if !valid {
			return "", "", true, false
		}
		canonical, err := normalizeInputText(
			formatNumber(value, behavior.policy.decimalPlaces),
		)
		if err != nil {
			return "", "", true, false
		}
		valueChanged := behavior.value != value
		behavior.value = value
		behavior.editor.working = canonical
		behavior.editor.caret = min(
			behavior.editor.caret,
			len(canonical.cells),
		)
		behavior.editor.committed = cloneInputText(canonical)
		behavior.editor.editing = false
		behavior.editor.selectionAnchor = -1
		behavior.editor.viewOffset = textViewOffset(
			behavior.editor.viewOffset,
			behavior.editor.caret,
			len(canonical.cells),
			state.bounds.Width,
		)
		state.behavior = behavior
		if valueChanged {
			return behavior.editor.changeCommand, state.id, true, true
		}
		return "", "", true, true
	}
	editor, command, target, handled, changed :=
		a.textEditorInputLocked(state, behavior.editor, key, held)
	if changed {
		behavior.editor = editor
		state.behavior = behavior
	}
	return command, target, handled, changed
}

func (a *App) commitOrCancelEditorStateLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	switch behavior := state.behavior.(type) {
	case dropDownBehavior:
		return a.cancelPopupCollectionLocked(state)
	case textFieldBehavior:
		_, _, changed := a.commitTextFieldStateLocked(state)
		return changed
	case numberFieldBehavior:
		if !behavior.editor.editing {
			return false
		}
		value, _, valid := parseNumberText(
			behavior.editor.working.text,
			behavior.policy,
		)
		if valid {
			behavior.value = value
			behavior.editor.committed, _ = normalizeInputText(
				formatNumber(value, behavior.policy.decimalPlaces),
			)
		}
		behavior.editor.working = cloneInputText(behavior.editor.committed)
		behavior.editor.editing = false
		behavior.editor.caret = len(behavior.editor.committed.cells)
		behavior.editor.viewOffset = 0
		behavior.editor.selectionAnchor = -1
		state.behavior = behavior
		return true
	case textAreaBehavior:
		if !behavior.editing {
			return false
		}
		behavior.committed = cloneInputText(behavior.working)
		behavior.editing = false
		behavior.selectionAnchor = -1
		behavior.preferredColumn = -1
		state.behavior = behavior
		return true
	case comboBoxBehavior:
		if behavior.popup.open {
			return a.cancelPopupCollectionLocked(state)
		}
		_, _, changed := a.commitComboBoxEditorLocked(state, &behavior)
		if changed {
			state.behavior = behavior
		}
		return changed
	case dataGridBehavior:
		if !behavior.editor.editing {
			return false
		}
		cancelDataGridEditBehavior(&behavior)
		state.behavior = behavior
		return true
	default:
		return false
	}
}

func (a *App) commitFocusedEditorLocked(reverse bool) (
	command CommandID,
	target ControlID,
	changed bool,
	move bool,
) {
	if a.focus == nil {
		return "", "", false, true
	}
	switch behavior := a.focus.behavior.(type) {
	case dropDownBehavior:
		return "", "", a.cancelPopupCollectionLocked(a.focus), true
	case textFieldBehavior:
		command, target, changed = a.commitTextFieldStateLocked(a.focus)
		return command, target, changed, true
	case numberFieldBehavior:
		if !behavior.editor.editing {
			return "", "", false, true
		}
		value, _, valid := parseNumberText(
			behavior.editor.working.text,
			behavior.policy,
		)
		if !valid {
			return "", "", false, false
		}
		valueChanged := behavior.value != value
		behavior.value = value
		behavior.editor.committed, _ = normalizeInputText(
			formatNumber(value, behavior.policy.decimalPlaces),
		)
		behavior.editor.working = cloneInputText(behavior.editor.committed)
		behavior.editor.editing = false
		behavior.editor.caret = min(
			behavior.editor.caret,
			len(behavior.editor.committed.cells),
		)
		behavior.editor.viewOffset = 0
		behavior.editor.selectionAnchor = -1
		a.focus.behavior = behavior
		if valueChanged {
			command = behavior.editor.changeCommand
			target = a.focus.id
		}
		return command, target, true, true
	case textAreaBehavior:
		if !behavior.editing {
			return "", "", false, true
		}
		valueChanged := behavior.committed.text != behavior.working.text
		behavior.committed = cloneInputText(behavior.working)
		behavior.editing = false
		behavior.selectionAnchor = -1
		behavior.preferredColumn = -1
		a.focus.behavior = behavior
		if valueChanged {
			return behavior.changeCommand, a.focus.id, true, true
		}
		return "", "", true, true
	case comboBoxBehavior:
		if behavior.popup.open {
			return "", "", a.cancelPopupCollectionLocked(a.focus), true
		}
		command, target, changed = a.commitComboBoxEditorLocked(
			a.focus,
			&behavior,
		)
		if changed {
			a.focus.behavior = behavior
		}
		return command, target, changed, true
	case dataGridBehavior:
		if !behavior.editor.editing {
			return "", "", false, true
		}
		command, changed, valid := a.commitDataGridEditLocked(a.focus, &behavior)
		if !valid {
			return "", "", false, false
		}
		if row, column, found := nextEditableDataGridCell(behavior, reverse); found {
			behavior.table.currentRow = row
			behavior.table.currentColumn = column
			if !beginDataGridEditBehavior(&behavior) ||
				!a.dataGridWithinBudgetLocked(a.focus, behavior) {
				cancelDataGridEditBehavior(&behavior)
			}
			behavior = reflowDataGrid(behavior, a.focus.bounds.Size())
			a.focus.behavior = behavior
			return command, a.focus.id, true, false
		}
		a.focus.behavior = behavior
		return command, a.focus.id, changed, true
	default:
		return "", "", false, true
	}
}

func numberFieldBehaviorEqual(
	left numberFieldBehavior,
	right numberFieldBehavior,
) bool {
	return left.value == right.value &&
		left.policy == right.policy &&
		controlBehaviorEqual(left.editor, right.editor)
}
