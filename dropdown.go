package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// DropDownOptions configures one selection-only collapsed popup field.
type DropDownOptions struct {
	PanelOptions
	Items           []ListItem
	Current         string
	Selected        string
	AllowEmpty      bool
	PopupRows       int
	Disabled        bool
	DisabledReason  string
	ChangeCommand   CommandID
	ActivateCommand CommandID
}

// ComboBoxOptions configures one editable collapsed popup field.
type ComboBoxOptions struct {
	DropDownOptions
	Text      string
	Validator *TextValidator
}

// DropDownState is one complete copied semantic and transient popup state.
type DropDownState struct {
	Current        string
	CurrentIndex   int
	Selected       string
	SelectedIndex  int
	Open           bool
	PopupCurrent   string
	PopupSelection string
	PopupOffset    int
	ItemCount      int
	EnabledCount   int
}

// ComboBoxState combines popup state with the embedded editor state.
type ComboBoxState struct {
	DropDownState
	Text    string
	Editing bool
	Valid   bool
}

// DropDown is a copy-safe focusable selection-only popup leaf.
type DropDown struct{ controlHandle }

// ComboBox is a copy-safe focusable editable popup leaf.
type ComboBox struct{ controlHandle }

type popupCollectionBehavior struct {
	items                 []normalizedListItem
	current               string
	selected              string
	allowEmpty            bool
	popupRows             int
	disabled              bool
	disabledReason        string
	changeCommand         CommandID
	activateCommand       CommandID
	open                  bool
	openingCurrent        string
	openingSelected       string
	popupCurrent          string
	popupSelected         string
	popupSelectionTouched bool
	popupOffset           int
}

type dropDownBehavior struct {
	popup popupCollectionBehavior
}

type comboBoxBehavior struct {
	popup  popupCollectionBehavior
	editor textFieldBehavior
}

// NewDropDown constructs and atomically inserts a DropDown.
func NewDropDown(parent Container, options DropDownOptions) (*DropDown, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewDropDown(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewDropDown records construction of a provisional DropDown.
func (t *Transaction) NewDropDown(
	parent Container,
	options DropDownOptions,
) (*DropDown, error) {
	popup, err := newPopupCollectionBehavior(options)
	if err != nil {
		return nil, err
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlDropDown,
		dropDownBehavior{popup: popup},
	)
	if err != nil {
		return nil, err
	}
	control := &DropDown{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

// NewComboBox constructs and atomically inserts a ComboBox.
func NewComboBox(parent Container, options ComboBoxOptions) (*ComboBox, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewComboBox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewComboBox records construction of a provisional ComboBox.
func (t *Transaction) NewComboBox(
	parent Container,
	options ComboBoxOptions,
) (*ComboBox, error) {
	popup, err := newPopupCollectionBehavior(options.DropDownOptions)
	if err != nil {
		return nil, err
	}
	text := options.Text
	if index := listItemIndex(popup.items, popup.selected); index >= 0 {
		text = popup.items[index].item.Label
	}
	value, err := normalizeInputText(text)
	if err != nil {
		return nil, err
	}
	validator, err := normalizeTextValidator(options.Validator)
	if err != nil {
		return nil, err
	}
	if err := validateComboItemLabels(popup.items, validator); err != nil {
		return nil, err
	}
	if validator != nil &&
		validator.value.Enforcement == TextValidationHard &&
		!textCellsValid(value.cells, validator) {
		return nil, fmt.Errorf(
			"%w: initial ComboBox text violates hard validation",
			ErrValidation,
		)
	}
	editor := textFieldBehavior{
		committed: value, working: cloneInputText(value),
		validator: validator, disabled: popup.disabled,
		disabledReason: popup.disabledReason,
		changeCommand:  popup.changeCommand, caret: len(value.cells),
		selectionAnchor: -1,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlComboBox,
		comboBoxBehavior{popup: popup, editor: editor},
	)
	if err != nil {
		return nil, err
	}
	control := &ComboBox{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

func newPopupCollectionBehavior(
	options DropDownOptions,
) (popupCollectionBehavior, error) {
	items, err := normalizeListItems(options.Items)
	if err != nil {
		return popupCollectionBehavior{}, err
	}
	if options.PopupRows == 0 {
		options.PopupRows = DefaultCollectionPopupRows
	}
	if options.PopupRows < 1 || options.PopupRows > MaxCollectionPopupRows {
		return popupCollectionBehavior{}, fmt.Errorf(
			"%w: popup rows must be between 1 and %d",
			ErrValidation,
			MaxCollectionPopupRows,
		)
	}
	reason, err := normalizeDisabledReason(
		options.Disabled,
		options.DisabledReason,
	)
	if err != nil {
		return popupCollectionBehavior{}, err
	}
	if err := validateOptionalCommand(options.ChangeCommand); err != nil {
		return popupCollectionBehavior{}, err
	}
	if err := validateOptionalCommand(options.ActivateCommand); err != nil {
		return popupCollectionBehavior{}, err
	}
	current, err := exactPopupKey(items, options.Current, "current")
	if err != nil {
		return popupCollectionBehavior{}, err
	}
	selected, err := exactPopupKey(items, options.Selected, "selection")
	if err != nil {
		return popupCollectionBehavior{}, err
	}
	if current == "" {
		current = selected
	}
	if current == "" {
		current = firstEnabledListKey(items)
	}
	if selected == "" && !options.AllowEmpty && current != "" {
		selected = current
	}
	return popupCollectionBehavior{
		items: items, current: current, selected: selected,
		allowEmpty: options.AllowEmpty, popupRows: options.PopupRows,
		disabled: options.Disabled, disabledReason: reason,
		changeCommand:   options.ChangeCommand,
		activateCommand: options.ActivateCommand,
	}, nil
}

func exactPopupKey(
	items []normalizedListItem,
	key string,
	role string,
) (string, error) {
	if key == "" {
		return "", nil
	}
	index := listItemIndex(items, key)
	if index < 0 || items[index].item.Disabled {
		return "", fmt.Errorf(
			"%w: invalid popup %s key",
			ErrValidation,
			role,
		)
	}
	return key, nil
}

func clonePopupCollectionBehavior(
	behavior popupCollectionBehavior,
) popupCollectionBehavior {
	cloned := behavior
	cloned.items = cloneNormalizedListItems(behavior.items)
	return cloned
}

func popupCollectionBehaviorEqual(
	left popupCollectionBehavior,
	right popupCollectionBehavior,
) bool {
	if left.current != right.current || left.selected != right.selected ||
		left.allowEmpty != right.allowEmpty ||
		left.popupRows != right.popupRows || left.disabled != right.disabled ||
		left.disabledReason != right.disabledReason ||
		left.changeCommand != right.changeCommand ||
		left.activateCommand != right.activateCommand ||
		left.open != right.open ||
		left.openingCurrent != right.openingCurrent ||
		left.openingSelected != right.openingSelected ||
		left.popupCurrent != right.popupCurrent ||
		left.popupSelected != right.popupSelected ||
		left.popupSelectionTouched != right.popupSelectionTouched ||
		left.popupOffset != right.popupOffset ||
		len(left.items) != len(right.items) {
		return false
	}
	for index := range left.items {
		if left.items[index].item != right.items[index].item {
			return false
		}
	}
	return true
}

func dropDownBehaviorEqual(left, right dropDownBehavior) bool {
	return popupCollectionBehaviorEqual(left.popup, right.popup)
}

func comboBoxBehaviorEqual(left, right comboBoxBehavior) bool {
	return popupCollectionBehaviorEqual(left.popup, right.popup) &&
		textFieldBehaviorEqual(left.editor, right.editor)
}

func (dropDownBehavior) clientInset() int { return 0 }

func (dropDownBehavior) intrinsicMinimum() Size {
	return Size{Width: 8, Height: 1}
}

func (dropDownBehavior) details() ControlDetails {
	return ControlDetails{
		Version:  ControlDetailsVersion,
		DropDown: &DropDownDetails{},
	}
}

func (b dropDownBehavior) additionalStyles() []StyleID {
	return popupCollectionStyles(false)
}

func (b dropDownBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	paintCollapsedPopupField(app, frame, state, absolute, clip, b.popup, nil)
}

func (comboBoxBehavior) clientInset() int { return 0 }

func (comboBoxBehavior) intrinsicMinimum() Size {
	return Size{Width: 8, Height: 1}
}

func (comboBoxBehavior) details() ControlDetails {
	return ControlDetails{
		Version:  ControlDetailsVersion,
		ComboBox: &ComboBoxDetails{},
	}
}

func (b comboBoxBehavior) additionalStyles() []StyleID {
	return append(popupCollectionStyles(true), b.editor.additionalStyles()...)
}

func (b comboBoxBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	paintCollapsedPopupField(
		app,
		frame,
		state,
		absolute,
		clip,
		b.popup,
		&b.editor,
	)
}

func popupCollectionStyles(combo bool) []StyleID {
	styles := []StyleID{
		"drop_down.popup", "drop_down.popup_border",
		"collection.current", "collection.selected",
		"collection.current_selected", "collection.disabled",
		"collection.empty",
	}
	if combo {
		return append(styles, "combo_box.focused", "combo_box.disabled")
	}
	return append(styles, "drop_down.focused", "drop_down.disabled")
}

func paintCollapsedPopupField(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
	popup popupCollectionBehavior,
	editor *textFieldBehavior,
) {
	content := absolute
	content.Width = max(0, content.Width-2)
	if editor != nil {
		paintTextEditor(app, frame, state, content, clip, *editor, false)
	} else {
		style := state.style
		if popup.disabled {
			style = "drop_down.disabled"
		} else if app.focus == state {
			style = "drop_down.focused"
		}
		app.fillStyleLocked(frame, absolute.Intersect(clip), style, state.id)
		if index := listItemIndex(popup.items, popup.selected); index >= 0 {
			cells := popup.items[index].label.lines[0]
			for column := 0; column < min(len(cells), content.Width); column++ {
				app.setClippedCellLocked(
					frame, clip, content.X+column,
					content.Y+max(0, (content.Height-1)/2),
					cells[column], style, app.styles[style], state.id,
				)
			}
		}
	}
	if absolute.Width <= 0 || absolute.Height <= 0 {
		return
	}
	arrowStyle := StyleID("drop_down")
	if state.kind == ControlComboBox {
		arrowStyle = "combo_box"
	}
	if popup.disabled {
		if state.kind == ControlComboBox {
			arrowStyle = "combo_box.disabled"
		} else {
			arrowStyle = "drop_down.disabled"
		}
	} else if app.focus == state {
		if state.kind == ControlComboBox {
			arrowStyle = "combo_box.focused"
		} else {
			arrowStyle = "drop_down.focused"
		}
	}
	x := absolute.X + absolute.Width - 1
	y := absolute.Y + max(0, (absolute.Height-1)/2)
	app.setClippedCellLocked(
		frame, clip, x, y, "▼", arrowStyle, app.styles[arrowStyle], state.id,
	)
}

func popupCollectionForState(
	state *controlState,
) (popupCollectionBehavior, bool) {
	if state == nil {
		return popupCollectionBehavior{}, false
	}
	switch behavior := state.behavior.(type) {
	case dropDownBehavior:
		return behavior.popup, true
	case comboBoxBehavior:
		return behavior.popup, true
	default:
		return popupCollectionBehavior{}, false
	}
}

func setPopupCollectionForState(
	state *controlState,
	popup popupCollectionBehavior,
) bool {
	if state == nil {
		return false
	}
	switch behavior := state.behavior.(type) {
	case dropDownBehavior:
		behavior.popup = popup
		state.behavior = behavior
		return true
	case comboBoxBehavior:
		behavior.popup = popup
		state.behavior = behavior
		return true
	default:
		return false
	}
}

func clearPopupCollectionTransient(popup *popupCollectionBehavior) {
	if popup == nil {
		return
	}
	popup.open = false
	popup.openingCurrent = ""
	popup.openingSelected = ""
	popup.popupCurrent = ""
	popup.popupSelected = ""
	popup.popupSelectionTouched = false
	popup.popupOffset = 0
}

func popupCollectionStorageBytes(popup popupCollectionBehavior) int {
	total := len(popup.current) + len(popup.selected) +
		len(popup.disabledReason)
	for _, item := range popup.items {
		total += len(item.item.Key) + len(item.item.Label) +
			len(item.item.Description) + len(item.item.DisabledReason)
	}
	return total
}

func comboBoxStorageBytes(behavior comboBoxBehavior) int {
	total := popupCollectionStorageBytes(behavior.popup) +
		len(behavior.editor.committed.text)
	if behavior.editor.editing {
		total += len(behavior.editor.working.text)
	}
	if behavior.editor.validator != nil {
		total += len(behavior.editor.validator.value.Characters)
	}
	return total
}

func validateComboItemLabels(
	items []normalizedListItem,
	validator *normalizedTextValidator,
) error {
	if validator == nil ||
		validator.value.Enforcement != TextValidationHard {
		return nil
	}
	for _, item := range items {
		if item.item.Disabled {
			continue
		}
		if !textCellsValid(item.label.lines[0], validator) {
			return fmt.Errorf(
				"%w: ComboBox item %q violates hard validation",
				ErrValidation,
				item.item.Key,
			)
		}
	}
	return nil
}

func validatePopupCollectionCommands(
	app *App,
	popup popupCollectionBehavior,
	require bool,
	name string,
) error {
	if !require {
		return nil
	}
	for _, command := range []CommandID{
		popup.changeCommand,
		popup.activateCommand,
	} {
		if command == "" {
			continue
		}
		if _, exists := app.commands[command]; !exists {
			return fmt.Errorf(
				"%w: %s command %q is not registered",
				ErrInvalidControl,
				name,
				command,
			)
		}
	}
	return nil
}

func popupCanFocus(state *controlState, popup popupCollectionBehavior) bool {
	return state != nil && !popup.disabled &&
		(state.kind == ControlComboBox || firstEnabledListKey(popup.items) != "")
}

func popupCollectionState(popup popupCollectionBehavior) DropDownState {
	return DropDownState{
		Current: popup.current, CurrentIndex: listItemIndex(popup.items, popup.current),
		Selected:      popup.selected,
		SelectedIndex: listItemIndex(popup.items, popup.selected),
		Open:          popup.open, PopupCurrent: popup.popupCurrent,
		PopupSelection: popup.popupSelected, PopupOffset: popup.popupOffset,
		ItemCount: len(popup.items), EnabledCount: enabledListCount(popup.items),
	}
}

func (app *App) popupDetailsLocked(
	state *controlState,
	popup popupCollectionBehavior,
) DropDownDetails {
	bounds := Rect{}
	if popup.open {
		bounds = app.popupCollectionBoundsLocked(state, popup)
	}
	return DropDownDetails{
		ItemCount: len(popup.items), EnabledCount: enabledListCount(popup.items),
		RetainedBytes: popupCollectionStorageBytes(popup),
		Current:       popup.current, CurrentIndex: listItemIndex(popup.items, popup.current),
		Selected:      popup.selected,
		SelectedIndex: listItemIndex(popup.items, popup.selected),
		AllowEmpty:    popup.allowEmpty, PopupRows: popup.popupRows,
		Open: popup.open, PopupBounds: bounds, PopupOffset: popup.popupOffset,
		PopupCurrent: popup.popupCurrent, PopupSelection: popup.popupSelected,
		Enabled: !popup.disabled, DisabledReason: popup.disabledReason,
		ChangeCommand:   popup.changeCommand,
		ActivateCommand: popup.activateCommand,
	}
}

// Items returns a caller-owned copy of the DropDown model.
func (d *DropDown) Items() []ListItem { return popupItemsForRead(d.controlState()) }

// State returns a copied DropDown state.
func (d *DropDown) State() DropDownState { return popupStateForRead(d.controlState()) }

// Items returns a caller-owned copy of the ComboBox model.
func (c *ComboBox) Items() []ListItem { return popupItemsForRead(c.controlState()) }

// State returns copied ComboBox popup and editor state.
func (c *ComboBox) State() ComboBoxState {
	if c == nil || c.state == nil || c.state.app == nil {
		return ComboBoxState{DropDownState: DropDownState{
			CurrentIndex: -1, SelectedIndex: -1,
		}}
	}
	c.state.app.mu.RLock()
	defer c.state.app.mu.RUnlock()
	behavior, ok := c.state.behavior.(comboBoxBehavior)
	if !ok || c.state.destroyed || c.state.aborted {
		return ComboBoxState{DropDownState: DropDownState{
			CurrentIndex: -1, SelectedIndex: -1,
		}}
	}
	return ComboBoxState{
		DropDownState: popupCollectionState(behavior.popup),
		Text:          behavior.editor.committed.text,
		Editing:       behavior.editor.editing,
		Valid:         textCellsValid(behavior.editor.current().cells, behavior.editor.validator),
	}
}

func popupItemsForRead(state *controlState) []ListItem {
	if state == nil || state.app == nil {
		return nil
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	popup, ok := popupCollectionForState(state)
	if !ok || state.destroyed || state.aborted {
		return nil
	}
	return copyListItems(popup.items)
}

func popupStateForRead(state *controlState) DropDownState {
	if state == nil || state.app == nil {
		return DropDownState{CurrentIndex: -1, SelectedIndex: -1}
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	popup, ok := popupCollectionForState(state)
	if !ok || state.destroyed || state.aborted {
		return DropDownState{CurrentIndex: -1, SelectedIndex: -1}
	}
	return popupCollectionState(popup)
}

// SetItems replaces the copied model while preserving surviving identities.
func (d *DropDown) SetItems(items []ListItem) error {
	return commitPopupMutation(d, func(transaction *Transaction) error {
		return transaction.SetDropDownItems(d, items)
	})
}

// SetSelection changes the exact selected stable key.
func (d *DropDown) SetSelection(selected string) error {
	return commitPopupMutation(d, func(transaction *Transaction) error {
		return transaction.SetDropDownSelection(d, selected)
	})
}

// SetItems replaces the ComboBox model while preserving surviving identities.
func (c *ComboBox) SetItems(items []ListItem) error {
	return commitPopupMutation(c, func(transaction *Transaction) error {
		return transaction.SetDropDownItems(c, items)
	})
}

// SetSelection changes the selected key and copies its label into the editor.
func (c *ComboBox) SetSelection(selected string) error {
	return commitPopupMutation(c, func(transaction *Transaction) error {
		return transaction.SetDropDownSelection(c, selected)
	})
}

// Text returns the committed ComboBox editor value.
func (c *ComboBox) Text() string {
	state := c.State()
	return state.Text
}

// SetText replaces committed ComboBox text and derives exact label selection.
func (c *ComboBox) SetText(value string) error {
	return commitPopupMutation(c, func(transaction *Transaction) error {
		return transaction.SetComboBoxText(c, value)
	})
}

// Validator returns a caller-owned copy of the optional ComboBox validator.
func (c *ComboBox) Validator() *TextValidator {
	if c == nil || c.state == nil || c.state.app == nil {
		return nil
	}
	c.state.app.mu.RLock()
	defer c.state.app.mu.RUnlock()
	behavior, ok := c.state.behavior.(comboBoxBehavior)
	if !ok || behavior.editor.validator == nil || c.state.destroyed || c.state.aborted {
		return nil
	}
	value := behavior.editor.validator.value
	return &value
}

// SetValidator replaces the copied optional ComboBox validator.
func (c *ComboBox) SetValidator(validator *TextValidator) error {
	return commitPopupMutation(c, func(transaction *Transaction) error {
		return transaction.SetComboBoxValidator(c, validator)
	})
}

func commitPopupMutation(
	control Control,
	record func(*Transaction) error,
) error {
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := record(transaction); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Open opens this DropDown's transient popup.
func (d *DropDown) Open() error { return setPopupOpen(d, true) }

// Close cancels this DropDown's transient popup.
func (d *DropDown) Close() error { return setPopupOpen(d, false) }

// Open opens this ComboBox's transient popup, committing an active edit.
func (c *ComboBox) Open() error { return setPopupOpen(c, true) }

// Close cancels this ComboBox's transient popup.
func (c *ComboBox) Close() error { return setPopupOpen(c, false) }

func setPopupOpen(control Control, open bool) error {
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return ErrInvalidControl
	}
	state := control.controlState()
	app := state.app
	if err := app.beginMutation(context.Background()); err != nil {
		return err
	}
	defer app.endMutation()
	app.mu.Lock()
	defer app.mu.Unlock()
	if err := state.mutableLocked(); err != nil {
		return err
	}
	changed := false
	if open {
		changed = app.openPopupCollectionLocked(state)
	} else {
		changed = app.cancelPopupCollectionLocked(state)
	}
	if changed {
		app.publishLocked(nil)
	}
	return nil
}

// Focus gives this eligible DropDown keyboard focus.
func (d *DropDown) Focus() error { return focusSelectionControl(d) }

// Focus gives this eligible ComboBox keyboard focus.
func (c *ComboBox) Focus() error { return focusSelectionControl(c) }

// Activate applies the same open/commit transition as semantic activation.
func (d *DropDown) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activatePopupCollection(ctx, source, requestID, d)
}

// Activate applies the same open/commit transition as semantic activation.
func (c *ComboBox) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	return activatePopupCollection(ctx, source, requestID, c)
}

func activatePopupCollection(
	ctx context.Context,
	source string,
	requestID string,
	control Control,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return Completion{}, ErrInvalidControl
	}
	if !validBoundedIdentifier(source) || !validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	app := control.controlState().app
	if err := app.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer app.endDispatch()
	app.mu.Lock()
	state := control.controlState()
	if app.final {
		app.mu.Unlock()
		return Completion{}, ErrClosed
	}
	popup, ok := popupCollectionForState(state)
	if !ok || !popupCanFocus(state, popup) ||
		!app.inActiveModalScopeLocked(state) {
		app.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	if app.focus != state {
		_ = app.commitOrCancelEditorStateLocked(app.focus)
		app.focus = state
		app.clearInvalidPressesLocked()
	}
	command, target, _, changed := app.popupCollectionKeyLocked(
		state,
		KeyEnter,
		nil,
		true,
	)
	result := CommandResult{Outcome: OutcomeNoOp}
	if changed {
		result.Outcome = OutcomeApplied
	}
	var router CommandRouter
	var execute bool
	if command != "" {
		router, result, execute = app.resolveCommandLocked(command, true)
	}
	app.mu.Unlock()
	if execute {
		result = app.callRouter(ctx, router, Command{
			ID: command, Target: target, Source: source,
		})
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.final {
		return Completion{}, ErrClosed
	}
	return app.associateLocked(requestID, result, command), nil
}

func (a *App) openPopupCollectionLocked(state *controlState) bool {
	popup, ok := popupCollectionForState(state)
	if !ok || !popupCanFocus(state, popup) || !a.effectivelyVisibleLocked(state) {
		return false
	}
	changed := a.cancelOtherPopupCollectionsLocked(state)
	if a.menu != nil {
		changed = a.closeMenuLocked() || changed
	}
	if a.focus != state {
		if a.commitOrCancelEditorStateLocked(a.focus) {
			changed = true
		}
		a.focus = state
		a.clearInvalidPressesLocked()
		changed = true
	}
	if behavior, combo := state.behavior.(comboBoxBehavior); combo &&
		behavior.editor.editing {
		_, _, editorChanged := a.commitComboBoxEditorLocked(state, &behavior)
		if editorChanged {
			state.behavior = behavior
			popup = behavior.popup
			changed = true
		}
	}
	if popup.open {
		return changed
	}
	popup.open = true
	popup.openingCurrent = popup.current
	popup.openingSelected = popup.selected
	popup.popupCurrent = popup.current
	if popup.popupCurrent == "" {
		popup.popupCurrent = firstEnabledListKey(popup.items)
	}
	popup.popupSelected = popup.selected
	popup.popupSelectionTouched = false
	popup.popupOffset = 0
	setPopupCollectionForState(state, popup)
	a.ensurePopupCurrentVisibleLocked(state)
	return true
}

func (a *App) cancelOtherPopupCollectionsLocked(except *controlState) bool {
	changed := false
	for _, state := range a.controlsByID {
		if state == except || state.destroyed {
			continue
		}
		popup, ok := popupCollectionForState(state)
		if !ok || !popup.open {
			continue
		}
		popup.current = popup.openingCurrent
		popup.selected = popup.openingSelected
		clearPopupCollectionTransient(&popup)
		setPopupCollectionForState(state, popup)
		changed = true
	}
	return changed
}

func (a *App) cancelPopupCollectionLocked(state *controlState) bool {
	popup, ok := popupCollectionForState(state)
	if !ok || !popup.open {
		return false
	}
	popup.current = popup.openingCurrent
	popup.selected = popup.openingSelected
	clearPopupCollectionTransient(&popup)
	setPopupCollectionForState(state, popup)
	return true
}

func (a *App) commitPopupCollectionLocked(
	state *controlState,
	activate bool,
) (CommandID, ControlID, bool) {
	popup, ok := popupCollectionForState(state)
	if !ok || !popup.open {
		return "", "", false
	}
	beforeSelected := popup.selected
	popup.current = popup.popupCurrent
	popup.selected = popup.popupSelected
	if popup.selected == "" && !popup.allowEmpty && popup.current != "" {
		popup.selected = popup.current
	}
	clearPopupCollectionTransient(&popup)
	setPopupCollectionForState(state, popup)
	if behavior, combo := state.behavior.(comboBoxBehavior); combo {
		if index := listItemIndex(behavior.popup.items, behavior.popup.selected); index >= 0 {
			value, _ := normalizeInputText(behavior.popup.items[index].item.Label)
			behavior.editor.committed = value
			behavior.editor.working = cloneInputText(value)
			behavior.editor.caret = len(value.cells)
			behavior.editor.viewOffset = 0
			behavior.editor.selectionAnchor = -1
			behavior.editor.editing = false
			state.behavior = behavior
		}
	}
	selectionChanged := beforeSelected != popup.selected
	command := CommandID("")
	if activate && popup.activateCommand != "" {
		command = popup.activateCommand
	} else if selectionChanged {
		command = popup.changeCommand
	}
	return command, state.id, true
}

func (a *App) popupCollectionKeyLocked(
	state *controlState,
	key Key,
	held map[Key]bool,
	semantic bool,
) (CommandID, ControlID, bool, bool) {
	popup, ok := popupCollectionForState(state)
	if !ok || !popupCanFocus(state, popup) || state != a.focus {
		return "", "", false, false
	}
	if held == nil {
		held = map[Key]bool{}
	}
	openRequest := noHeldModifiers(held) &&
		(key == KeySpace || key == KeyEnter || key == KeyF4)
	if state.kind == ControlComboBox && key == KeyEnter && !semantic {
		openRequest = false
	}
	if held[KeyAlt] && !held[KeyControl] && !held[KeyMeta] &&
		key == KeyDown {
		openRequest = true
	}
	if !popup.open {
		if !openRequest {
			return "", "", false, false
		}
		changed := a.openPopupCollectionLocked(state)
		return "", state.id, true, changed
	}
	if noHeldModifiers(held) && key == KeyEscape {
		return "", state.id, true, a.cancelPopupCollectionLocked(state)
	}
	if noHeldModifiers(held) && key == KeyEnter {
		if popup.popupCurrent != "" && !popup.popupSelectionTouched {
			popup.popupSelected = popup.popupCurrent
			setPopupCollectionForState(state, popup)
		}
		command, target, changed := a.commitPopupCollectionLocked(state, true)
		return command, target, true, changed
	}
	if !noHeldModifiers(held) {
		return "", "", false, false
	}
	current := listItemIndex(popup.items, popup.popupCurrent)
	handled := true
	switch key {
	case KeyUp:
		popup.popupCurrent = previousEnabledListKey(popup.items, current)
	case KeyDown:
		popup.popupCurrent = nextEnabledListKey(popup.items, current)
	case KeyPageUp:
		popup.popupCurrent = pageEnabledListKey(
			popup.items,
			current,
			-max(1, popup.popupRows),
		)
	case KeyPageDown:
		popup.popupCurrent = pageEnabledListKey(
			popup.items,
			current,
			max(1, popup.popupRows),
		)
	case KeyHome:
		popup.popupCurrent = firstEnabledListKey(popup.items)
	case KeyEnd:
		popup.popupCurrent = lastEnabledListKey(popup.items)
	case KeySpace:
		popup.popupSelectionTouched = true
		if popup.popupCurrent == popup.popupSelected && popup.allowEmpty {
			popup.popupSelected = ""
		} else {
			popup.popupSelected = popup.popupCurrent
		}
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	before := popupCollectionForComparison(state)
	setPopupCollectionForState(state, popup)
	a.ensurePopupCurrentVisibleLocked(state)
	after := popupCollectionForComparison(state)
	return "", state.id, true, !popupCollectionBehaviorEqual(before, after)
}

func popupCollectionForComparison(state *controlState) popupCollectionBehavior {
	popup, _ := popupCollectionForState(state)
	return clonePopupCollectionBehavior(popup)
}

func (a *App) ensurePopupCurrentVisibleLocked(state *controlState) {
	popup, ok := popupCollectionForState(state)
	if !ok || !popup.open {
		return
	}
	bounds := a.popupCollectionBoundsLocked(state, popup)
	rows := max(0, bounds.Height-2)
	index := listItemIndex(popup.items, popup.popupCurrent)
	maximum := max(0, len(popup.items)-rows)
	if rows <= 0 {
		popup.popupOffset = 0
	} else if index < popup.popupOffset {
		popup.popupOffset = index
	} else if index >= popup.popupOffset+rows {
		popup.popupOffset = index - rows + 1
	}
	popup.popupOffset = min(maximum, max(0, popup.popupOffset))
	setPopupCollectionForState(state, popup)
}

func (a *App) popupCollectionBoundsLocked(
	state *controlState,
	popup popupCollectionBehavior,
) Rect {
	client := a.applicationContentRectLocked()
	if state == nil || client.Empty() {
		return Rect{}
	}
	field := a.controlAbsoluteLocked(state)
	contentRows := min(popup.popupRows, max(1, len(popup.items)))
	height := min(client.Height, contentRows+2)
	if height <= 0 {
		return Rect{}
	}
	width := max(8, field.Width)
	for _, item := range popup.items {
		rowWidth := 6 + item.label.cells
		if item.description.cells > 0 {
			rowWidth += 2 + item.description.cells
		}
		width = max(width, rowWidth+2)
	}
	width = min(client.Width, width)
	x := min(field.X, client.X+client.Width-width)
	x = max(client.X, x)
	below := field.Y + field.Height
	above := field.Y - height
	y := below
	if below+height > client.Y+client.Height && above >= client.Y {
		y = above
	}
	y = min(max(client.Y, y), client.Y+client.Height-height)
	return Rect{X: x, Y: y, Width: width, Height: height}
}

func (a *App) openPopupCollectionStateLocked() *controlState {
	for _, state := range a.controlsByID {
		popup, ok := popupCollectionForState(state)
		if ok && popup.open && !state.destroyed &&
			a.effectivelyVisibleLocked(state) {
			return state
		}
	}
	return nil
}

func (a *App) paintCollectionPopupOverlayLocked(frame *IntendedFrame) {
	state := a.openPopupCollectionStateLocked()
	if state == nil {
		return
	}
	popup, _ := popupCollectionForState(state)
	bounds := a.popupCollectionBoundsLocked(state, popup)
	if bounds.Empty() {
		return
	}
	clip := bounds.Intersect(a.applicationContentRectLocked())
	border := borderBehavior{
		form: BorderSingle, borderStyle: "drop_down.popup_border",
	}
	a.fillStyleLocked(frame, bounds.Intersect(clip), "drop_down.popup", state.id)
	a.paintBorderLocked(frame, bounds, clip, state, border)
	rows := max(0, bounds.Height-2)
	for row := 0; row < rows; row++ {
		index := popup.popupOffset + row
		cells, style, exists := popupCollectionDisplayRow(popup, index)
		if !exists {
			continue
		}
		y := bounds.Y + 1 + row
		rowRect := Rect{X: bounds.X + 1, Y: y, Width: max(0, bounds.Width-2), Height: 1}
		a.fillStyleLocked(frame, rowRect.Intersect(clip), style, state.id)
		for column := 0; column < min(len(cells), rowRect.Width); column++ {
			a.setClippedCellLocked(
				frame, clip, rowRect.X+column, y, cells[column],
				style, a.styles[style], state.id,
			)
		}
	}
}

func popupCollectionDisplayRow(
	popup popupCollectionBehavior,
	index int,
) ([]string, StyleID, bool) {
	if len(popup.items) == 0 {
		if index == 0 {
			return []string{"[", "e", "m", "p", "t", "y", "]"},
				"collection.empty", true
		}
		return nil, "", false
	}
	if index < 0 || index >= len(popup.items) {
		return nil, "", false
	}
	item := popup.items[index]
	current := item.item.Key == popup.popupCurrent
	selected := item.item.Key == popup.popupSelected
	style := StyleID("drop_down.popup")
	if item.item.Disabled {
		style = "collection.disabled"
	} else if current && selected {
		style = "collection.current_selected"
	} else if current {
		style = "collection.current"
	} else if selected {
		style = "collection.selected"
	}
	marker := " "
	if current {
		marker = "►"
	}
	selectedMarker := " "
	if selected {
		selectedMarker = "X"
	}
	cells := []string{marker, " ", "[", selectedMarker, "]", " "}
	cells = append(cells, item.label.lines[0]...)
	if item.description.cells > 0 {
		cells = append(cells, " ", " ")
		cells = append(cells, item.description.lines[0]...)
	}
	return cells, style, true
}

func (t *Transaction) selectedPopupCollection(
	control Control,
) (*controlState, controlBehavior, error) {
	state, err := t.control(control)
	if err != nil {
		return nil, nil, err
	}
	selected := t.selectedControlBehavior(state)
	switch behavior := selected.(type) {
	case dropDownBehavior:
		behavior.popup = clonePopupCollectionBehavior(behavior.popup)
		return state, behavior, nil
	case comboBoxBehavior:
		behavior.popup = clonePopupCollectionBehavior(behavior.popup)
		behavior.editor = cloneTextFieldBehavior(behavior.editor)
		return state, behavior, nil
	default:
		return nil, nil, ErrInvalidControl
	}
}

func cloneTextFieldBehavior(behavior textFieldBehavior) textFieldBehavior {
	cloned := behavior
	cloned.committed = cloneInputText(behavior.committed)
	cloned.working = cloneInputText(behavior.working)
	if behavior.validator != nil {
		validator, _ := normalizeTextValidator(&behavior.validator.value)
		cloned.validator = validator
	}
	return cloned
}

func (t *Transaction) recordPopupCollection(
	state *controlState,
	behavior controlBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationPopupCollection, state: state, behavior: behavior,
	})
	return nil
}

// SetDropDownItems records a copied DropDown or ComboBox model replacement.
func (t *Transaction) SetDropDownItems(
	control Control,
	items []ListItem,
) error {
	state, selected, err := t.selectedPopupCollection(control)
	if err != nil {
		return err
	}
	normalized, err := normalizeListItems(items)
	if err != nil {
		return err
	}
	apply := func(popup *popupCollectionBehavior) {
		oldIndex := listItemIndex(popup.items, popup.current)
		popup.items = normalized
		repairPopupIdentity(popup, oldIndex)
		clearPopupCollectionTransient(popup)
	}
	switch behavior := selected.(type) {
	case dropDownBehavior:
		apply(&behavior.popup)
		return t.recordPopupCollection(state, behavior)
	case comboBoxBehavior:
		if err := validateComboItemLabels(
			normalized,
			behavior.editor.validator,
		); err != nil {
			return err
		}
		apply(&behavior.popup)
		syncComboEditorToSelection(&behavior)
		return t.recordPopupCollection(state, behavior)
	default:
		return ErrInvalidControl
	}
}

// SetDropDownSelection records an exact DropDown or ComboBox selected key.
func (t *Transaction) SetDropDownSelection(
	control Control,
	selectedKey string,
) error {
	state, selected, err := t.selectedPopupCollection(control)
	if err != nil {
		return err
	}
	apply := func(popup *popupCollectionBehavior) error {
		key, err := exactPopupKey(popup.items, selectedKey, "selection")
		if err != nil {
			return err
		}
		if key == "" && !popup.allowEmpty &&
			firstEnabledListKey(popup.items) != "" {
			return fmt.Errorf("%w: popup selection is required", ErrValidation)
		}
		popup.selected = key
		if key != "" {
			popup.current = key
		}
		clearPopupCollectionTransient(popup)
		return nil
	}
	switch behavior := selected.(type) {
	case dropDownBehavior:
		if err := apply(&behavior.popup); err != nil {
			return err
		}
		return t.recordPopupCollection(state, behavior)
	case comboBoxBehavior:
		if err := apply(&behavior.popup); err != nil {
			return err
		}
		syncComboEditorToSelection(&behavior)
		return t.recordPopupCollection(state, behavior)
	default:
		return ErrInvalidControl
	}
}

// SetComboBoxText records one committed editor replacement and exact-label
// selection derivation.
func (t *Transaction) SetComboBoxText(
	control *ComboBox,
	text string,
) error {
	state, selected, err := t.selectedPopupCollection(control)
	if err != nil {
		return err
	}
	behavior, ok := selected.(comboBoxBehavior)
	if !ok {
		return ErrInvalidControl
	}
	value, err := normalizeInputText(text)
	if err != nil {
		return err
	}
	if behavior.editor.validator != nil &&
		behavior.editor.validator.value.Enforcement == TextValidationHard &&
		!textCellsValid(value.cells, behavior.editor.validator) {
		return fmt.Errorf("%w: ComboBox text violates hard validation", ErrValidation)
	}
	behavior.editor.committed = value
	behavior.editor.working = cloneInputText(value)
	behavior.editor.editing = false
	behavior.editor.caret = len(value.cells)
	behavior.editor.viewOffset = 0
	behavior.editor.selectionAnchor = -1
	behavior.popup.selected = popupKeyForLabel(behavior.popup.items, value.text)
	if behavior.popup.selected != "" {
		behavior.popup.current = behavior.popup.selected
	}
	clearPopupCollectionTransient(&behavior.popup)
	return t.recordPopupCollection(state, behavior)
}

// SetComboBoxValidator records one copied optional editor validator.
func (t *Transaction) SetComboBoxValidator(
	control *ComboBox,
	validator *TextValidator,
) error {
	state, selected, err := t.selectedPopupCollection(control)
	if err != nil {
		return err
	}
	behavior, ok := selected.(comboBoxBehavior)
	if !ok {
		return ErrInvalidControl
	}
	normalized, err := normalizeTextValidator(validator)
	if err != nil {
		return err
	}
	if normalized != nil &&
		normalized.value.Enforcement == TextValidationHard &&
		!textCellsValid(behavior.editor.committed.cells, normalized) {
		return fmt.Errorf(
			"%w: committed ComboBox text violates hard validation",
			ErrValidation,
		)
	}
	if err := validateComboItemLabels(behavior.popup.items, normalized); err != nil {
		return err
	}
	behavior.editor.validator = normalized
	behavior.editor.editing = false
	behavior.editor.working = cloneInputText(behavior.editor.committed)
	behavior.editor.selectionAnchor = -1
	clearPopupCollectionTransient(&behavior.popup)
	return t.recordPopupCollection(state, behavior)
}

func repairPopupIdentity(popup *popupCollectionBehavior, oldIndex int) {
	if popup == nil {
		return
	}
	current := listItemIndex(popup.items, popup.current)
	if current < 0 || popup.items[current].item.Disabled {
		popup.current = repairedListCurrent(popup.items, oldIndex)
	}
	selected := listItemIndex(popup.items, popup.selected)
	if selected < 0 || (selected >= 0 && popup.items[selected].item.Disabled) {
		popup.selected = ""
	}
	if popup.selected == "" && !popup.allowEmpty && popup.current != "" {
		popup.selected = popup.current
	}
}

func popupKeyForLabel(items []normalizedListItem, label string) string {
	for _, item := range items {
		if !item.item.Disabled && item.item.Label == label {
			return item.item.Key
		}
	}
	return ""
}

func syncComboEditorToSelection(behavior *comboBoxBehavior) {
	if behavior == nil || behavior.popup.selected == "" {
		return
	}
	index := listItemIndex(behavior.popup.items, behavior.popup.selected)
	if index < 0 {
		return
	}
	value, _ := normalizeInputText(behavior.popup.items[index].item.Label)
	behavior.editor.committed = value
	behavior.editor.working = cloneInputText(value)
	behavior.editor.editing = false
	behavior.editor.caret = len(value.cells)
	behavior.editor.viewOffset = 0
	behavior.editor.selectionAnchor = -1
}

func (a *App) comboBoxEditorInputLocked(
	state *controlState,
	behavior comboBoxBehavior,
	key Key,
	held map[Key]bool,
) (CommandID, ControlID, bool, bool) {
	if behavior.popup.open || behavior.popup.disabled {
		return "", "", false, false
	}
	if key == KeyF2 && noHeldModifiers(held) && !behavior.editor.editing {
		key = KeyEnter
	}
	beforeText := behavior.editor.committed.text
	editor, command, target, handled, changed := a.textEditorInputLocked(
		state,
		behavior.editor,
		key,
		held,
	)
	if !handled {
		return "", "", false, false
	}
	behavior.editor = editor
	if changed && !editor.editing && editor.committed.text != beforeText {
		behavior.popup.selected = popupKeyForLabel(
			behavior.popup.items,
			editor.committed.text,
		)
		if behavior.popup.selected != "" {
			behavior.popup.current = behavior.popup.selected
		}
	}
	state.behavior = behavior
	return command, target, true, changed
}

func (a *App) applyComboBoxTextInputLocked(
	state *controlState,
	behavior comboBoxBehavior,
	text string,
) CommandResult {
	if state == nil || behavior.popup.open || !behavior.editor.editing {
		return CommandResult{Outcome: OutcomeNoOp}
	}
	inserted, err := normalizeInputText(text)
	if err != nil {
		return rejectedTextInput(err)
	}
	inserted.cells = filterTextInputCells(
		inserted.cells,
		behavior.editor.validator,
		false,
	)
	candidate, caret, ok := replaceTextInputCells(
		behavior.editor.working.cells,
		behavior.editor.caret,
		behavior.editor.selectionAnchor,
		inserted.cells,
		behavior.editor.effectiveMaximumBytes(),
	)
	if !ok {
		return textInputCapacityResult()
	}
	if candidate == nil {
		return CommandResult{Outcome: OutcomeNoOp}
	}
	behavior.editor.working.cells = candidate
	behavior.editor.working.text = strings.Join(candidate, "")
	behavior.editor.caret = caret
	behavior.editor.selectionAnchor = -1
	behavior.editor.viewOffset = textViewOffset(
		behavior.editor.viewOffset,
		behavior.editor.caret,
		len(candidate),
		max(0, state.bounds.Width-2),
	)
	state.behavior = behavior
	return CommandResult{Outcome: OutcomeApplied}
}

func (a *App) commitComboBoxEditorLocked(
	state *controlState,
	behavior *comboBoxBehavior,
) (CommandID, ControlID, bool) {
	if state == nil || behavior == nil || !behavior.editor.editing {
		return "", "", false
	}
	changedValue := behavior.editor.committed.text != behavior.editor.working.text
	behavior.editor.committed = cloneInputText(behavior.editor.working)
	behavior.editor.editing = false
	behavior.editor.selectionAnchor = -1
	behavior.popup.selected = popupKeyForLabel(
		behavior.popup.items,
		behavior.editor.committed.text,
	)
	if behavior.popup.selected != "" {
		behavior.popup.current = behavior.popup.selected
	}
	if changedValue {
		return behavior.popup.changeCommand, state.id, true
	}
	return "", "", true
}
