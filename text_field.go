package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Hard-Problems-Group-LLC/expletives/internal/display"
	"github.com/rivo/uniseg"
)

// TextValidationEnforcement selects whether invalid interactive input remains
// editable or is ignored.
type TextValidationEnforcement string

const (
	// TextValidationSoft retains invalid input and paints its invalid state.
	TextValidationSoft TextValidationEnforcement = "soft"
	// TextValidationHard ignores invalid interactive input.
	TextValidationHard TextValidationEnforcement = "hard"
)

// TextValidationMode selects allow-list or deny-list matching.
type TextValidationMode string

const (
	// TextValidationWhitelist permits only listed one-cell elements.
	TextValidationWhitelist TextValidationMode = "whitelist"
	// TextValidationBlacklist rejects listed one-cell elements.
	TextValidationBlacklist TextValidationMode = "blacklist"
)

// TextValidator is one copied character-set validation policy.
type TextValidator struct {
	Enforcement TextValidationEnforcement
	Mode        TextValidationMode
	Characters  string
}

// TextFieldOptions configures one bounded single-line editor.
type TextFieldOptions struct {
	PanelOptions
	Text           string
	Validator      *TextValidator
	Password       bool
	Disabled       bool
	DisabledReason string
	ChangeCommand  CommandID
}

// TextField is a copy-safe focusable single-line editor leaf.
type TextField struct{ controlHandle }

type normalizedInputText struct {
	text  string
	cells []string
}

type normalizedTextValidator struct {
	value TextValidator
	set   map[string]struct{}
}

type textFieldBehavior struct {
	committed      normalizedInputText
	working        normalizedInputText
	validator      *normalizedTextValidator
	password       bool
	disabled       bool
	disabledReason string
	changeCommand  CommandID
	editing        bool
	caret          int
	viewOffset     int
}

// NewTextField constructs and atomically inserts a TextField.
func NewTextField(
	parent Container,
	options TextFieldOptions,
) (*TextField, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	field, err := tx.NewTextField(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return field, nil
}

// NewTextField records construction of a provisional TextField.
func (t *Transaction) NewTextField(
	parent Container,
	options TextFieldOptions,
) (*TextField, error) {
	value, err := normalizeInputText(options.Text)
	if err != nil {
		return nil, err
	}
	validator, err := normalizeTextValidator(options.Validator)
	if err != nil {
		return nil, err
	}
	if validator != nil &&
		validator.value.Enforcement == TextValidationHard &&
		!textCellsValid(value.cells, validator) {
		return nil, fmt.Errorf(
			"%w: initial TextField value violates hard validation",
			ErrValidation,
		)
	}
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
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlTextField,
		textFieldBehavior{
			committed: value, working: cloneInputText(value),
			validator: validator, password: options.Password,
			disabled: options.Disabled, disabledReason: reason,
			changeCommand: options.ChangeCommand, caret: len(value.cells),
		},
	)
	if err != nil {
		return nil, err
	}
	field := &TextField{controlHandle: controlHandle{state: state}}
	state.control = field
	return field, nil
}

func normalizeInputText(text string) (normalizedInputText, error) {
	if len(text) > MaxTextInputBytes {
		return normalizedInputText{}, fmt.Errorf(
			"%w: input text exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTextInputBytes,
		)
	}
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, display.Replacement)
	}
	for _, current := range text {
		if current == '\r' || current == '\n' || unicode.IsControl(current) {
			return normalizedInputText{}, fmt.Errorf(
				"%w: TextField input contains a control character",
				ErrTextLimit,
			)
		}
	}
	graphemes := uniseg.NewGraphemes(text)
	cells := make([]string, 0, min(len(text), MaxTextInputCells))
	for graphemes.Next() {
		cell := graphemes.Str()
		if graphemes.Width() != 1 || len(cell) > MaxCellBytes {
			cell = display.Replacement
		}
		cells = append(cells, cell)
		if len(cells) > MaxTextInputCells {
			return normalizedInputText{}, fmt.Errorf(
				"%w: input text exceeds %d canonical cells",
				ErrTextLimit,
				MaxTextInputCells,
			)
		}
	}
	normalized := strings.Join(cells, "")
	if len(normalized) > MaxTextInputBytes {
		return normalizedInputText{}, fmt.Errorf(
			"%w: normalized input exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxTextInputBytes,
		)
	}
	return normalizedInputText{text: normalized, cells: cells}, nil
}

func normalizeTextValidator(
	validator *TextValidator,
) (*normalizedTextValidator, error) {
	if validator == nil {
		return nil, nil
	}
	value := *validator
	switch value.Enforcement {
	case TextValidationSoft, TextValidationHard:
	default:
		return nil, fmt.Errorf(
			"%w: invalid text validation enforcement %q",
			ErrValidation,
			value.Enforcement,
		)
	}
	switch value.Mode {
	case TextValidationWhitelist, TextValidationBlacklist:
	default:
		return nil, fmt.Errorf(
			"%w: invalid text validation mode %q",
			ErrValidation,
			value.Mode,
		)
	}
	if value.Characters == "" ||
		len(value.Characters) > MaxTextValidatorBytes {
		return nil, fmt.Errorf(
			"%w: validator characters must be nonempty and at most %d bytes",
			ErrValidation,
			MaxTextValidatorBytes,
		)
	}
	characters, err := normalizeInputText(value.Characters)
	if err != nil {
		return nil, fmt.Errorf("%w: validator characters: %v", ErrValidation, err)
	}
	if len(characters.cells) == 0 ||
		len(characters.cells) > MaxTextValidatorCells {
		return nil, fmt.Errorf(
			"%w: validator character set exceeds %d elements",
			ErrValidation,
			MaxTextValidatorCells,
		)
	}
	set := make(map[string]struct{}, len(characters.cells))
	unique := make([]string, 0, len(characters.cells))
	for _, cell := range characters.cells {
		if _, exists := set[cell]; exists {
			continue
		}
		set[cell] = struct{}{}
		unique = append(unique, cell)
	}
	value.Characters = strings.Join(unique, "")
	return &normalizedTextValidator{value: value, set: set}, nil
}

func cloneInputText(value normalizedInputText) normalizedInputText {
	return normalizedInputText{
		text:  value.text,
		cells: append([]string(nil), value.cells...),
	}
}

func cloneTextValidator(
	validator *normalizedTextValidator,
) *normalizedTextValidator {
	if validator == nil {
		return nil
	}
	set := make(map[string]struct{}, len(validator.set))
	for cell := range validator.set {
		set[cell] = struct{}{}
	}
	return &normalizedTextValidator{value: validator.value, set: set}
}

func textCellAllowed(
	cell string,
	validator *normalizedTextValidator,
) bool {
	if validator == nil {
		return true
	}
	_, matched := validator.set[cell]
	if validator.value.Mode == TextValidationWhitelist {
		return matched
	}
	return !matched
}

func textValidationMask(
	cells []string,
	validator *normalizedTextValidator,
) ([]bool, bool) {
	invalid := make([]bool, len(cells))
	valid := true
	for index, cell := range cells {
		invalid[index] = !textCellAllowed(cell, validator)
		if invalid[index] {
			valid = false
		}
	}
	return invalid, valid
}

func textCellsValid(
	cells []string,
	validator *normalizedTextValidator,
) bool {
	_, valid := textValidationMask(cells, validator)
	return valid
}

func (b textFieldBehavior) current() normalizedInputText {
	if b.editing {
		return b.working
	}
	return b.committed
}

func (b textFieldBehavior) clientInset() int { return 0 }

func (b textFieldBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"text_input.valid",
		"text_input.invalid",
		"text_input.invalid_character",
		"text_input.disabled",
	}
}

func (textFieldBehavior) intrinsicMinimum() Size {
	return Size{Width: 8, Height: 1}
}

func (textFieldBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		TextField: &TextFieldDetails{},
	}
}

func (b textFieldBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	current := b.current()
	invalid, valid := textValidationMask(current.cells, b.validator)
	offset := textViewOffset(b.viewOffset, b.caret, len(current.cells), absolute.Width)
	y := absolute.Y + max(0, (absolute.Height-1)/2)
	for column := 0; column < absolute.Width; column++ {
		index := offset + column
		if index >= len(current.cells) {
			break
		}
		cell := current.cells[index]
		if b.password {
			cell = "*"
		}
		style := state.style
		switch {
		case b.disabled:
			style = "text_input.disabled"
		case b.validator == nil:
		case valid:
			style = "text_input.valid"
		case invalid[index]:
			style = "text_input.invalid_character"
		default:
			style = "text_input.invalid"
		}
		app.setClippedCellLocked(
			frame,
			clip,
			absolute.X+column,
			y,
			cell,
			style,
			app.styles[style],
			state.id,
		)
	}
	cursorX := absolute.X + min(max(0, b.caret-offset), max(0, absolute.Width-1))
	if app.focus == state && b.editing && !b.disabled &&
		absolute.Width > 0 &&
		cursorX >= clip.X && cursorX < clip.X+clip.Width &&
		y >= clip.Y && y < clip.Y+clip.Height {
		app.cursor = CursorState{
			Visible: true,
			Position: Point{
				X: cursorX,
				Y: y,
			},
		}
	}
}

func textViewOffset(offset, caret, length, width int) int {
	if width <= 0 {
		return 0
	}
	offset = min(max(0, offset), length)
	caret = min(max(0, caret), length)
	if caret < offset {
		offset = caret
	}
	if caret >= offset+width {
		offset = caret - width + 1
	}
	maximum := max(0, length-width+1)
	return min(offset, maximum)
}

// Text returns the committed application value.
func (f *TextField) Text() string {
	behavior, ok := textFieldBehaviorForRead(f.controlState())
	if !ok {
		return ""
	}
	return behavior.committed.text
}

// SetText atomically replaces the committed value without a user-change
// notification and leaves edit mode.
func (f *TextField) SetText(text string) error {
	if f == nil || f.controlState() == nil {
		return ErrInvalidControl
	}
	tx := f.controlState().app.NewTransaction()
	if err := tx.SetText(f, text); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Validator returns a caller-owned copy of the current optional validator.
func (f *TextField) Validator() *TextValidator {
	behavior, ok := textFieldBehaviorForRead(f.controlState())
	if !ok || behavior.validator == nil {
		return nil
	}
	value := behavior.validator.value
	return &value
}

// SetValidator atomically replaces the optional validator.
func (f *TextField) SetValidator(validator *TextValidator) error {
	if f == nil || f.controlState() == nil {
		return ErrInvalidControl
	}
	tx := f.controlState().app.NewTransaction()
	if err := tx.SetTextValidator(f, validator); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Password reports whether display and snapshot values are masked.
func (f *TextField) Password() bool {
	behavior, ok := textFieldBehaviorForRead(f.controlState())
	return ok && behavior.password
}

// SetPassword atomically changes password presentation and snapshot redaction.
func (f *TextField) SetPassword(password bool) error {
	if f == nil || f.controlState() == nil {
		return ErrInvalidControl
	}
	tx := f.controlState().app.NewTransaction()
	if err := tx.SetTextPassword(f, password); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Editing reports whether the field currently captures printable input.
func (f *TextField) Editing() bool {
	behavior, ok := textFieldBehaviorForRead(f.controlState())
	return ok && behavior.editing
}

// Valid reports validation of the current working or committed value.
func (f *TextField) Valid() bool {
	behavior, ok := textFieldBehaviorForRead(f.controlState())
	return ok && textCellsValid(behavior.current().cells, behavior.validator)
}

func textFieldBehaviorForRead(
	state *controlState,
) (textFieldBehavior, bool) {
	if state == nil || state.app == nil {
		return textFieldBehavior{}, false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(textFieldBehavior)
	return behavior, ok && !state.aborted && !state.destroyed
}

// Focus gives this eligible TextField keyboard focus.
func (f *TextField) Focus() error {
	return focusSelectionControl(f)
}

// Activate focuses this eligible TextField and enters edit mode.
func (f *TextField) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if f == nil || f.controlState() == nil {
		return Completion{}, ErrInvalidControl
	}
	return f.controlState().app.activateTextField(
		ctx,
		source,
		requestID,
		f.controlState(),
	)
}

// SetTextValidator records one atomic validator replacement. A hard validator
// cannot be applied to a committed value that it rejects.
func (t *Transaction) SetTextValidator(
	field *TextField,
	validator *TextValidator,
) error {
	state, err := t.control(field)
	if err != nil || state.kind != ControlTextField {
		return ErrInvalidControl
	}
	normalized, err := normalizeTextValidator(validator)
	if err != nil {
		return err
	}
	behavior, ok := t.recordedControlBehavior(state).(textFieldBehavior)
	if !ok {
		return ErrInvalidControl
	}
	if normalized != nil &&
		normalized.value.Enforcement == TextValidationHard &&
		!textCellsValid(behavior.committed.cells, normalized) {
		return fmt.Errorf(
			"%w: current TextField value violates hard validation",
			ErrValidation,
		)
	}
	behavior.validator = normalized
	behavior.working = cloneInputText(behavior.committed)
	behavior.editing = false
	behavior.caret = len(behavior.committed.cells)
	behavior.viewOffset = 0
	return t.recordTextFieldBehavior(state, behavior)
}

// SetTextPassword records one atomic password-presentation replacement.
func (t *Transaction) SetTextPassword(
	field *TextField,
	password bool,
) error {
	state, err := t.control(field)
	if err != nil || state.kind != ControlTextField {
		return ErrInvalidControl
	}
	behavior, ok := t.recordedControlBehavior(state).(textFieldBehavior)
	if !ok {
		return ErrInvalidControl
	}
	behavior.password = password
	return t.recordTextFieldBehavior(state, behavior)
}

func (t *Transaction) recordedControlBehavior(
	state *controlState,
) controlBehavior {
	for index := len(t.mutations) - 1; index >= 0; index-- {
		if t.mutations[index].state == state &&
			t.mutations[index].behavior != nil {
			return t.mutations[index].behavior
		}
	}
	return state.behavior
}

func (t *Transaction) recordTextFieldBehavior(
	state *controlState,
	behavior textFieldBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind:  mutationTextField,
		state: state, behavior: behavior,
	})
	return nil
}

func (a *App) activateTextField(
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
		_, _, committed := a.commitTextFieldStateLocked(a.focus)
		changed = changed || committed
	}
	a.focus = state
	behavior := state.behavior.(textFieldBehavior)
	if !behavior.editing {
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = true
		behavior.caret = len(behavior.working.cells)
		behavior.viewOffset = 0
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

func (a *App) textFieldDetailsLocked(
	state *controlState,
	behavior textFieldBehavior,
) TextFieldDetails {
	current := behavior.current()
	_, valid := textValidationMask(current.cells, behavior.validator)
	offset := textViewOffset(
		behavior.viewOffset,
		behavior.caret,
		len(current.cells),
		state.bounds.Width,
	)
	details := TextFieldDetails{
		Length: len(current.cells), Caret: behavior.caret,
		ViewOffset: offset, Editing: behavior.editing,
		Valid: valid, Password: behavior.password,
		Redacted: behavior.password, Enabled: !behavior.disabled,
		DisabledReason: behavior.disabledReason,
		ChangeCommand:  behavior.changeCommand,
	}
	if !behavior.password {
		details.Text = current.text
	}
	if behavior.validator != nil {
		details.Validator = &TextValidatorDetails{
			Enforcement: behavior.validator.value.Enforcement,
			Mode:        behavior.validator.value.Mode,
			Characters:  behavior.validator.value.Characters,
		}
	}
	return details
}

func (a *App) textFieldInputLocked(
	state *controlState,
	key Key,
	held map[Key]bool,
) (command CommandID, target ControlID, handled, changed bool) {
	if state == nil || state != a.focus || !a.focusEligibleLocked(state) {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(textFieldBehavior)
	if !ok {
		return "", "", false, false
	}
	if !behavior.editing {
		if key != KeyEnter || !noHeldModifiers(held) {
			return "", "", false, false
		}
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = true
		behavior.caret = len(behavior.working.cells)
		behavior.viewOffset = 0
		state.behavior = behavior
		return "", "", true, true
	}
	if !textInputModifiers(held) {
		return "", "", false, false
	}
	handled = true
	switch key {
	case KeyEnter:
		changed = true
		valueChanged := behavior.committed.text != behavior.working.text
		behavior.committed = cloneInputText(behavior.working)
		behavior.editing = false
		if valueChanged {
			command = behavior.changeCommand
			target = state.id
		}
	case KeyEscape:
		behavior.working = cloneInputText(behavior.committed)
		behavior.editing = false
		behavior.caret = len(behavior.committed.cells)
		behavior.viewOffset = 0
		changed = true
	case KeyLeft:
		if behavior.caret > 0 {
			behavior.caret--
			changed = true
		}
	case KeyRight:
		if behavior.caret < len(behavior.working.cells) {
			behavior.caret++
			changed = true
		}
	case KeyHome:
		if behavior.caret != 0 {
			behavior.caret = 0
			changed = true
		}
	case KeyEnd:
		if behavior.caret != len(behavior.working.cells) {
			behavior.caret = len(behavior.working.cells)
			changed = true
		}
	case KeyBackspace:
		if behavior.caret > 0 {
			index := behavior.caret - 1
			behavior.working.cells = append(
				behavior.working.cells[:index],
				behavior.working.cells[behavior.caret:]...,
			)
			behavior.caret--
			changed = true
		}
	case KeyDelete:
		if behavior.caret < len(behavior.working.cells) {
			behavior.working.cells = append(
				behavior.working.cells[:behavior.caret],
				behavior.working.cells[behavior.caret+1:]...,
			)
			changed = true
		}
	default:
		text, printable := printableTextForKey(key, held)
		if !printable {
			handled = false
			return "", "", handled, false
		}
		inserted, err := normalizeInputText(text)
		if err != nil || len(inserted.cells) != 1 {
			return "", "", true, false
		}
		cell := inserted.cells[0]
		if behavior.validator != nil &&
			behavior.validator.value.Enforcement == TextValidationHard &&
			!textCellAllowed(cell, behavior.validator) {
			return "", "", true, false
		}
		candidate := make([]string, 0, len(behavior.working.cells)+1)
		candidate = append(candidate, behavior.working.cells[:behavior.caret]...)
		candidate = append(candidate, cell)
		candidate = append(candidate, behavior.working.cells[behavior.caret:]...)
		candidateText := strings.Join(candidate, "")
		if len(candidateText) > MaxTextInputBytes ||
			len(candidate) > MaxTextInputCells {
			return "", "", true, false
		}
		behavior.working.cells = candidate
		behavior.caret++
		changed = true
	}
	if changed {
		behavior.working.text = strings.Join(behavior.working.cells, "")
		behavior.viewOffset = textViewOffset(
			behavior.viewOffset,
			behavior.caret,
			len(behavior.working.cells),
			state.bounds.Width,
		)
		state.behavior = behavior
	}
	return command, target, handled, changed
}

func (a *App) commitFocusedTextFieldLocked() (
	command CommandID,
	target ControlID,
	changed bool,
) {
	return a.commitTextFieldStateLocked(a.focus)
}

func (a *App) commitTextFieldStateLocked(
	state *controlState,
) (
	command CommandID,
	target ControlID,
	changed bool,
) {
	if state == nil {
		return "", "", false
	}
	behavior, ok := state.behavior.(textFieldBehavior)
	if !ok || !behavior.editing {
		return "", "", false
	}
	valueChanged := behavior.committed.text != behavior.working.text
	behavior.committed = cloneInputText(behavior.working)
	behavior.editing = false
	behavior.viewOffset = textViewOffset(
		behavior.viewOffset,
		behavior.caret,
		len(behavior.working.cells),
		state.bounds.Width,
	)
	state.behavior = behavior
	if valueChanged {
		command = behavior.changeCommand
		target = state.id
	}
	return command, target, true
}

func textInputModifiers(held map[Key]bool) bool {
	return !held[KeyAlt] && !held[KeyControl] && !held[KeyMeta]
}

func printableTextForKey(key Key, held map[Key]bool) (string, bool) {
	if key == KeySpace {
		return " ", true
	}
	if !validPrintableKey(key) {
		return "", false
	}
	text := string(key)
	if held[KeyShift] && len(text) == 1 &&
		text[0] >= 'a' && text[0] <= 'z' {
		text = strings.ToUpper(text)
	}
	return text, true
}

func validPrintableKey(key Key) bool {
	text := string(key)
	if text == "" || len(text) > MaxCellBytes || !utf8.ValidString(text) {
		return false
	}
	for _, current := range text {
		if unicode.IsControl(current) {
			return false
		}
	}
	graphemes := uniseg.NewGraphemes(text)
	if !graphemes.Next() {
		return false
	}
	return !graphemes.Next()
}

func textFieldBehaviorEqual(left, right textFieldBehavior) bool {
	return left.committed.text == right.committed.text &&
		left.working.text == right.working.text &&
		textValidatorEqual(left.validator, right.validator) &&
		left.password == right.password &&
		left.disabled == right.disabled &&
		left.disabledReason == right.disabledReason &&
		left.changeCommand == right.changeCommand &&
		left.editing == right.editing &&
		left.caret == right.caret &&
		left.viewOffset == right.viewOffset
}

func textValidatorEqual(
	left, right *normalizedTextValidator,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.value == right.value
}
