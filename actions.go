package expletives

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const buttonMinimumWidth = 10

// ButtonOptions configures one focusable command-activation Button.
type ButtonOptions struct {
	PanelOptions
	Command  CommandID
	Mnemonic Key
	Default  bool
	Cancel   bool
}

// HotkeyBarItem identifies one command shown by a HotkeyBar.
type HotkeyBarItem struct {
	Command CommandID
}

// HotkeyBarOptions configures one ordered command-shortcut summary.
type HotkeyBarOptions struct {
	PanelOptions
	Items []HotkeyBarItem
}

// Button is a copy-safe focusable non-container control handle.
type Button struct{ controlHandle }

// HotkeyBar is a copy-safe non-focusable non-container control handle.
type HotkeyBar struct{ controlHandle }

type buttonBehavior struct {
	command  CommandID
	mnemonic Key
	default_ bool
	cancel   bool
}

type hotkeyBarBehavior struct {
	items []HotkeyBarItem
}

type pressedAction struct {
	control *controlState
	key     Key
}

// NewButton constructs and atomically inserts a Button.
func NewButton(parent Container, options ButtonOptions) (*Button, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	button, err := tx.NewButton(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return button, nil
}

// NewHotkeyBar constructs and atomically inserts a HotkeyBar.
func NewHotkeyBar(
	parent Container,
	options HotkeyBarOptions,
) (*HotkeyBar, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	bar, err := tx.NewHotkeyBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return bar, nil
}

// NewButton records construction of a provisional Button.
func (t *Transaction) NewButton(
	parent Container,
	options ButtonOptions,
) (*Button, error) {
	if !validBoundedIdentifier(string(options.Command)) {
		return nil, fmt.Errorf("%w: invalid Button command", ErrInvalidControl)
	}
	mnemonic, err := normalizeMnemonic(options.Mnemonic)
	if err != nil {
		return nil, err
	}
	if options.Default && options.Cancel {
		return nil, fmt.Errorf(
			"%w: Button cannot be both default and cancel",
			ErrInvalidControl,
		)
	}
	behavior := buttonBehavior{
		command: options.Command, mnemonic: mnemonic,
		default_: options.Default, cancel: options.Cancel,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlButton,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	button := &Button{controlHandle: controlHandle{state: state}}
	state.control = button
	return button, nil
}

// NewHotkeyBar records construction of a provisional HotkeyBar.
func (t *Transaction) NewHotkeyBar(
	parent Container,
	options HotkeyBarOptions,
) (*HotkeyBar, error) {
	if len(options.Items) > MaxHotkeyBarItems {
		return nil, fmt.Errorf(
			"%w: HotkeyBar exceeds %d items",
			ErrControlCapacity,
			MaxHotkeyBarItems,
		)
	}
	items := append([]HotkeyBarItem(nil), options.Items...)
	seen := make(map[CommandID]bool, len(items))
	for _, item := range items {
		if !validBoundedIdentifier(string(item.Command)) {
			return nil, fmt.Errorf(
				"%w: invalid HotkeyBar command",
				ErrInvalidControl,
			)
		}
		if seen[item.Command] {
			return nil, fmt.Errorf(
				"%w: duplicate HotkeyBar command %q",
				ErrInvalidControl,
				item.Command,
			)
		}
		seen[item.Command] = true
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlHotkeyBar,
		hotkeyBarBehavior{items: items},
	)
	if err != nil {
		return nil, err
	}
	bar := &HotkeyBar{controlHandle: controlHandle{state: state}}
	state.control = bar
	return bar, nil
}

// Items returns a caller-owned copy of the HotkeyBar's ordered inventory.
func (b *HotkeyBar) Items() []HotkeyBarItem {
	if b == nil || b.state == nil || b.state.app == nil {
		return nil
	}
	b.state.app.mu.RLock()
	defer b.state.app.mu.RUnlock()
	behavior, ok := b.state.behavior.(hotkeyBarBehavior)
	if !ok || b.state.aborted {
		return nil
	}
	return append([]HotkeyBarItem(nil), behavior.items...)
}

// Focus synchronously gives this eligible Button keyboard focus.
func (b *Button) Focus() error {
	if b == nil || b.state == nil || b.state.app == nil {
		return ErrInvalidControl
	}
	tx := b.state.app.NewTransaction()
	if err := tx.SetFocus(b); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Activate invokes this eligible Button through the App command router.
func (b *Button) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if b == nil || b.state == nil || b.state.app == nil {
		return Completion{}, ErrInvalidControl
	}
	return b.state.app.activateButton(ctx, source, requestID, b.state)
}

// Focused returns the currently focused control, or nil.
func (a *App) Focused() Control {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.focus == nil || a.focus.destroyed || a.focus.aborted {
		return nil
	}
	return a.focus.control
}

func (b buttonBehavior) clientInset() int { return 0 }

func (b buttonBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"button.default",
		"button.focused",
		"button.pressed",
		"button.disabled",
		"button.mnemonic",
		"button.shadow",
	}
}

func (b buttonBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	definition := app.commands[b.command]
	label := effectiveCommandLabel(definition, b.command)
	details := app.actionDetailsLocked(state, b)
	bodyStyle := state.style
	mnemonicStyle := StyleID("button.mnemonic")
	switch {
	case !details.Enabled:
		bodyStyle = "button.disabled"
		mnemonicStyle = bodyStyle
	case app.pressedAnyLocked(state):
		bodyStyle = "button.pressed"
	case app.focus == state:
		bodyStyle = "button.focused"
	case b.default_:
		bodyStyle = "button.default"
	}
	if absolute.Empty() {
		return
	}

	paint := func(x, y int, grapheme string, style StyleID) {
		app.setClippedCellLocked(
			frame, clip, x, y, grapheme, style, app.styles[style], state.id,
		)
	}
	paintLabel := func(left, right, y int) {
		if right <= left || y < absolute.Y || y >= absolute.Y+absolute.Height {
			return
		}
		cells := label.lines[0]
		start := left + max(0, (right-left-len(cells))/2)
		for index, grapheme := range cells {
			x := start + index
			if x >= right {
				break
			}
			style := bodyStyle
			if details.Enabled && b.mnemonic != "" &&
				Key(strings.ToLower(grapheme)) == b.mnemonic {
				style = mnemonicStyle
			}
			paint(x, y, grapheme, style)
		}
	}

	// A caller may explicitly constrain a Button below its natural two-row
	// Turbo Vision geometry. Keep that degraded override usable, but do not
	// pretend that it has room for the raised-control shadow.
	if absolute.Height < 2 || absolute.Width < 3 {
		for y := absolute.Y; y < absolute.Y+absolute.Height; y++ {
			for x := absolute.X; x < absolute.X+absolute.Width; x++ {
				paint(x, y, " ", bodyStyle)
			}
		}
		paintLabel(
			absolute.X,
			absolute.X+absolute.Width,
			absolute.Y+(absolute.Height-1)/2,
		)
		return
	}

	pressed := app.pressedAnyLocked(state)
	left := absolute.X
	right := absolute.X + absolute.Width - 1
	bottom := absolute.Y + absolute.Height - 1
	bodyLeft, bodyRight := left+1, right
	if pressed {
		bodyLeft, bodyRight = left+2, right+1
	}
	for y := absolute.Y; y < bottom; y++ {
		for x := left; x <= right; x++ {
			paint(x, y, " ", bodyStyle)
		}
		paint(left, y, " ", "button.shadow")
		if pressed {
			paint(left+1, y, " ", "button.shadow")
		} else {
			shadow := "█"
			if y == absolute.Y {
				shadow = "▄"
			}
			paint(right, y, shadow, "button.shadow")
		}
	}
	for x := left; x <= right; x++ {
		paint(x, bottom, " ", "button.shadow")
	}
	if !pressed {
		for x := left + 2; x <= right; x++ {
			paint(x, bottom, "▀", "button.shadow")
		}
	}
	paintLabel(
		bodyLeft,
		bodyRight,
		absolute.Y+absolute.Height/2-1,
	)
}

func (b buttonBehavior) details() ControlDetails {
	return ControlDetails{
		Version: ControlDetailsVersion,
		Action: &ActionDetails{
			Command:  b.command,
			Mnemonic: b.mnemonic,
			Default:  b.default_,
			Cancel:   b.cancel,
		},
	}
}

func (b buttonBehavior) intrinsicMinimum() Size {
	// Command labels are App-owned and may change independently. Commit
	// replaces this conservative value with the current effective label.
	return Size{Width: buttonMinimumWidth, Height: 2}
}

func buttonMinimumForLabel(label normalizedDisplayText) Size {
	return Size{
		Width:  max(buttonMinimumWidth, len(label.lines[0])+4),
		Height: 2,
	}
}

func (b hotkeyBarBehavior) clientInset() int { return 0 }

func (b hotkeyBarBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	cells := make([]string, 0, absolute.Width)
	for _, item := range app.hotkeyBarItemDetailsLocked(b.items) {
		segment := []string{" "}
		if !item.Enabled {
			segment[0] = "("
		}
		if item.Chord != nil {
			segment = append(segment, displayChord(*item.Chord)...)
			segment = append(segment, " ")
		}
		label, _ := normalizeDisplayText(item.Label, false)
		segment = append(segment, label.lines[0]...)
		if item.Enabled {
			segment = append(segment, " ")
		} else {
			segment = append(segment, ")")
		}
		cells = append(cells, segment...)
	}
	app.paintTextRowsLocked(
		frame,
		state,
		absolute,
		clip,
		[][]string{cells},
		TextAlignStart,
		TextAlignCenter,
	)
}

func (b hotkeyBarBehavior) details() ControlDetails {
	return ControlDetails{
		Version:   ControlDetailsVersion,
		HotkeyBar: &HotkeyBarDetails{Items: []HotkeyBarItemDetails{}},
	}
}

func (b hotkeyBarBehavior) intrinsicMinimum() Size {
	return Size{Width: 1, Height: 1}
}

func effectiveCommandLabel(
	definition CommandDefinition,
	command CommandID,
) normalizedDisplayText {
	text := definition.Label
	if text == "" {
		text = string(command)
	}
	normalized, err := normalizeDisplayText(text, false)
	if err != nil {
		// Registered definitions have already passed normalization; a valid
		// command ID is always safe as a defensive fallback.
		normalized, _ = normalizeDisplayText(string(command), false)
	}
	return normalized
}

func effectiveDisabledReason(definition CommandDefinition, exists bool) string {
	if !exists {
		return "Command is not registered"
	}
	if definition.Enabled {
		return ""
	}
	if definition.DisabledReason != "" {
		return definition.DisabledReason
	}
	return "Command is disabled"
}

func (a *App) actionDetailsLocked(
	state *controlState,
	behavior buttonBehavior,
) ActionDetails {
	definition, exists := a.commands[behavior.command]
	enabled := exists && definition.Enabled
	disabledReason := effectiveDisabledReason(definition, exists)
	if reason := a.localButtonDisabledReasonLocked(state, behavior); reason != "" {
		enabled = false
		disabledReason = reason
	}
	return ActionDetails{
		Label:          effectiveCommandLabel(definition, behavior.command).text,
		Command:        behavior.command,
		Enabled:        enabled,
		DisabledReason: disabledReason,
		Checked:        definition.Checked,
		Mnemonic:       behavior.mnemonic,
		Pressed:        a.pressedAnyLocked(state),
		Default:        behavior.default_,
		Cancel:         behavior.cancel,
	}
}

func (a *App) localButtonDisabledReasonLocked(
	state *controlState,
	behavior buttonBehavior,
) string {
	if behavior.command != CommandDialogCancel {
		return ""
	}
	modal := modalAncestorState(state)
	if modal == nil || modal.kind != ControlProgressDialog {
		return ""
	}
	modalBehavior, ok := modal.behavior.(modalPanelBehavior)
	if !ok || modalBehavior.progress == nil || modalBehavior.progressCancel == nil {
		return "Cancellation is unavailable"
	}
	if modalBehavior.progress.cancelRequested.Load() {
		return "Cancellation requested"
	}
	return ""
}

func (a *App) hotkeyBarItemDetailsLocked(
	items []HotkeyBarItem,
) []HotkeyBarItemDetails {
	details := make([]HotkeyBarItemDetails, 0, len(items))
	for _, item := range items {
		definition, exists := a.commands[item.Command]
		details = append(details, HotkeyBarItemDetails{
			Label:          effectiveCommandLabel(definition, item.Command).text,
			Command:        item.Command,
			Enabled:        exists && definition.Enabled,
			DisabledReason: effectiveDisabledReason(definition, exists),
			Checked:        definition.Checked,
			Chord:          a.firstChordLocked(item.Command),
		})
	}
	return details
}

func (a *App) firstChordLocked(command CommandID) *Chord {
	candidates := make([]string, 0)
	for encoded, bound := range a.bindings {
		if bound == command {
			candidates = append(candidates, encoded)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Strings(candidates)
	chord := decodeChord(candidates[0])
	return &chord
}

func decodeChord(encoded string) Chord {
	parts := strings.Split(encoded, "+")
	chord := Chord{Key: Key(parts[len(parts)-1])}
	for _, part := range parts[:len(parts)-1] {
		chord.Modifiers = append(chord.Modifiers, Key(part))
	}
	return chord
}

func displayChord(chord Chord) []string {
	names := make([]string, 0, len(chord.Modifiers)+1)
	for _, modifier := range chord.Modifiers {
		switch modifier {
		case KeyControl:
			names = append(names, "Ctrl")
		case KeyAlt:
			names = append(names, "Alt")
		case KeyShift:
			names = append(names, "Shift")
		case KeyMeta:
			names = append(names, "Meta")
		}
	}
	key := string(chord.Key)
	if len(key) == 1 {
		key = strings.ToUpper(key)
	} else if key != "" {
		key = strings.ToUpper(key[:1]) + key[1:]
	}
	names = append(names, key)
	cells := make([]string, 0)
	for index, name := range names {
		if index != 0 {
			cells = append(cells, "+")
		}
		for _, current := range name {
			cells = append(cells, string(current))
		}
	}
	return cells
}

func (a *App) pressedAnyLocked(state *controlState) bool {
	for _, press := range a.pressed {
		if press.control == state {
			return true
		}
	}
	return false
}

func (a *App) effectivelyVisibleLocked(state *controlState) bool {
	for current := state; current != nil; current = current.parent {
		if current.destroyed || current.aborted || !current.visible {
			return false
		}
		if managed, selected := a.isManagedTabPageLocked(current); managed && !selected {
			return false
		}
	}
	return true
}

func (a *App) effectiveControlClipLocked(state *controlState) Rect {
	if state == nil {
		return Rect{}
	}
	path := make([]*controlState, 0)
	for current := state; current != nil; current = current.parent {
		path = append(path, current)
	}
	if len(path) == 0 || path[len(path)-1] != a.root.state {
		return Rect{}
	}
	surface := Rect{Width: a.size.Width, Height: a.size.Height}
	parentOrigin := Point{}
	ancestorClip := surface
	ancestorsVisible := true
	for index := len(path) - 1; index >= 0; index-- {
		current := path[index]
		if current.destroyed || current.aborted {
			return Rect{}
		}
		bounds := current.bounds
		absolute := bounds
		if isApplicationChrome(current.kind) {
			switch current.kind {
			case ControlMenuBar:
				bounds = menuBarSurfaceRect(a.size)
			case ControlStatusBar:
				bounds = statusBarSurfaceRect(a.size)
			}
			absolute = bounds
			ancestorClip = surface
		} else if !current.root {
			absolute.X += parentOrigin.X
			absolute.Y += parentOrigin.Y
		}
		clip := absolute.Intersect(ancestorClip)
		visible := ancestorsVisible && current.visible
		if managed, selected := a.isManagedTabPageLocked(current); managed && !selected {
			visible = false
		}
		if !visible {
			return Rect{}
		}
		if current == state {
			return clip
		}
		client := absolute
		if current.root {
			client = a.rootContentRectLocked()
		} else {
			client = controlClientRect(current, absolute)
		}
		ancestorClip = clip.Intersect(client)
		parentOrigin = Point{X: client.X, Y: client.Y}
		ancestorsVisible = visible
	}
	return Rect{}
}

func (a *App) controlReceivesInputLocked(state *controlState) bool {
	if !a.effectivelyVisibleLocked(state) ||
		!a.inActiveModalScopeLocked(state) {
		return false
	}
	if scope := declaredInputScopeState(state); scope != nil &&
		a.effectiveControlClipLocked(scope).Empty() {
		return false
	}
	return true
}

func (a *App) activeInputScopeLocked() (*controlState, InputScopeMode) {
	if modal := a.topModalLocked(); modal != nil {
		return modal, InputScopeConfined
	}
	if scope := declaredInputScopeState(a.focus); scope != nil &&
		a.controlReceivesInputLocked(scope) {
		return scope, scope.inputScope
	}
	return a.root.state, InputScopeConfined
}

func (a *App) inInputScopeLocked(
	state *controlState,
	scope *controlState,
) bool {
	if state == nil || scope == nil || !controlWithin(state, scope) {
		return false
	}
	if scope == a.topModalLocked() {
		return true
	}
	return inputScopeOwnerState(state, a.root.state) == scope
}

func (a *App) inActiveInputScopeLocked(state *controlState) bool {
	scope, _ := a.activeInputScopeLocked()
	return a.inInputScopeLocked(state, scope)
}

func (a *App) buttonEligibleLocked(state *controlState) bool {
	if state == nil || state.kind != ControlButton ||
		!a.controlReceivesInputLocked(state) {
		return false
	}
	behavior, ok := state.behavior.(buttonBehavior)
	if !ok {
		return false
	}
	definition, exists := a.commands[behavior.command]
	return exists && definition.Enabled &&
		a.localButtonDisabledReasonLocked(state, behavior) == ""
}

func (a *App) focusEligibleLocked(state *controlState) bool {
	if state == nil || !a.controlReceivesInputLocked(state) {
		return false
	}
	switch behavior := state.behavior.(type) {
	case buttonBehavior:
		definition, exists := a.commands[behavior.command]
		return exists && definition.Enabled &&
			a.localButtonDisabledReasonLocked(state, behavior) == ""
	case checkboxBehavior:
		return !behavior.disabled
	case radioButtonBehavior:
		return a.radioButtonEnabledLocked(state)
	case choiceFieldBehavior:
		return a.choiceFieldEnabledLocked(state)
	case textFieldBehavior:
		return !behavior.disabled
	case numberFieldBehavior:
		return !behavior.editor.disabled
	case textAreaBehavior:
		return !behavior.disabled
	case scrollBarBehavior:
		return !behavior.disabled
	case tabbedPanelBehavior:
		return tabbedPanelEnabled(behavior)
	case scrollViewBehavior:
		return scrollViewCanMove(state, behavior)
	case markdownBehavior:
		return markdownCanMove(state, behavior)
	case logViewBehavior:
		return logCanMove(state, behavior)
	case listBoxBehavior:
		return listBoxCanFocus(state, behavior)
	case treeViewBehavior:
		return treeViewCanFocus(state, behavior)
	case tableBehavior:
		return tableCanFocus(state, behavior)
	case dataGridBehavior:
		return dataGridCanFocus(state, behavior)
	case dropDownBehavior:
		return popupCanFocus(state, behavior.popup)
	case comboBoxBehavior:
		return popupCanFocus(state, behavior.popup)
	default:
		return false
	}
}

func (a *App) focusBehaviorEligibleLocked(
	state *controlState,
	behavior controlBehavior,
) bool {
	switch behavior := behavior.(type) {
	case buttonBehavior:
		definition, exists := a.commands[behavior.command]
		return exists && definition.Enabled &&
			a.localButtonDisabledReasonLocked(state, behavior) == ""
	case checkboxBehavior:
		return !behavior.disabled
	case radioButtonBehavior:
		if behavior.disabled || state == nil || state.parent == nil {
			return false
		}
		group, ok := state.parent.behavior.(radioGroupBehavior)
		return ok && !group.disabled
	case choiceFieldBehavior:
		if behavior.disabled {
			return false
		}
		for _, option := range behavior.options {
			if !option.option.Disabled {
				return true
			}
		}
	case textFieldBehavior:
		return !behavior.disabled
	case numberFieldBehavior:
		return !behavior.editor.disabled
	case textAreaBehavior:
		return !behavior.disabled
	case scrollBarBehavior:
		return !behavior.disabled
	case tabbedPanelBehavior:
		return tabbedPanelEnabled(behavior)
	case scrollViewBehavior:
		return scrollViewCanMove(state, behavior)
	case markdownBehavior:
		return markdownCanMove(state, behavior)
	case logViewBehavior:
		return logCanMove(state, behavior)
	case listBoxBehavior:
		return listBoxCanFocus(state, behavior)
	case treeViewBehavior:
		return treeViewCanFocus(state, behavior)
	case tableBehavior:
		return tableCanFocus(state, behavior)
	case dataGridBehavior:
		return dataGridCanFocus(state, behavior)
	case dropDownBehavior:
		return popupCanFocus(state, behavior.popup)
	case comboBoxBehavior:
		return popupCanFocus(state, behavior.popup)
	}
	return false
}

func (a *App) focusableControlsLocked(global bool) []*controlState {
	groups := a.focusGroupsLocked(global)
	controls := make([]*controlState, 0, len(groups))
	for _, group := range groups {
		if target := a.tabEntryForFocusGroupLocked(group, false); target != nil {
			controls = append(controls, target)
		}
	}
	return controls
}

func (a *App) allFocusableControlsLocked(global bool) []*controlState {
	controls := make([]*controlState, 0)
	scope, mode := a.activeInputScopeLocked()
	var visit func(*controlState)
	visit = func(state *controlState) {
		if a.focusEligibleLocked(state) &&
			(global || mode == InputScopeEscaping || a.inInputScopeLocked(state, scope)) {
			controls = append(controls, state)
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	start := a.root.state
	if modal := a.topModalLocked(); modal != nil {
		start = modal
	}
	visit(start)
	return controls
}

type focusGroup struct {
	owner    *controlState
	controls []*controlState
}

func (a *App) focusGroupsLocked(global bool) []focusGroup {
	groups := make([]focusGroup, 0)
	indexes := make(map[*controlState]int)
	for _, control := range a.allFocusableControlsLocked(global) {
		owner := control.parent
		index, found := indexes[owner]
		if !found {
			index = len(groups)
			indexes[owner] = index
			groups = append(groups, focusGroup{owner: owner})
		}
		groups[index].controls = append(groups[index].controls, control)
	}
	return groups
}

func (a *App) tabEntryForFocusGroupLocked(
	group focusGroup,
	reverse bool,
) *controlState {
	if len(group.controls) == 0 {
		return nil
	}
	if group.owner != nil && group.owner.kind == ControlRadioGroup {
		for _, control := range group.controls {
			if a.radioSelectedLocked(control) {
				return control
			}
		}
	}
	if reverse {
		return group.controls[len(group.controls)-1]
	}
	return group.controls[0]
}

func (a *App) ensureFocusLocked() bool {
	if a.menu != nil {
		if a.topModalLocked() != nil {
			a.closeMenuLocked()
		} else {
			if a.menu.bar != nil &&
				a.effectivelyVisibleLocked(a.menu.bar) {
				changed := a.focus != a.menu.bar
				a.focus = a.menu.bar
				return changed
			}
			a.closeMenuLocked()
		}
	}
	if a.focusEligibleLocked(a.focus) {
		return false
	}
	previous := a.focus
	committed := a.commitOrCancelEditorStateLocked(previous)
	a.focus = nil
	controls := a.focusableControlsLocked(true)
	if len(controls) != 0 {
		a.focus = controls[0]
	}
	if previous != a.focus {
		a.clearInvalidPressesLocked()
		return true
	}
	return committed
}

func (a *App) moveFocusLocked(reverse bool) bool {
	groups := a.focusGroupsLocked(false)
	if len(groups) == 0 {
		if a.focus == nil {
			return false
		}
		a.focus = nil
		a.clearInvalidPressesLocked()
		return true
	}
	index := -1
	currentOwner := (*controlState)(nil)
	if a.focus != nil {
		currentOwner = a.focus.parent
	}
	if len(groups) == 1 && groups[0].owner == currentOwner {
		return false
	}
	for current, group := range groups {
		if group.owner == currentOwner {
			index = current
			break
		}
	}
	if reverse {
		if index <= 0 {
			index = len(groups) - 1
		} else {
			index--
		}
	} else {
		index = (index + 1) % len(groups)
	}
	target := a.tabEntryForFocusGroupLocked(groups[index], reverse)
	if target == nil || a.focus == target {
		return false
	}
	a.focus = target
	a.clearInvalidPressesLocked()
	return true
}

func isDirectionalFocusKey(key Key) bool {
	switch key {
	case KeyLeft, KeyRight, KeyUp, KeyDown:
		return true
	default:
		return false
	}
}

func (a *App) moveDirectionalFocusLocked(key Key) bool {
	if !isDirectionalFocusKey(key) || !a.focusEligibleLocked(a.focus) {
		return false
	}
	controls := a.allFocusableControlsLocked(false)
	sameGroup := make([]*controlState, 0)
	otherGroups := make([]*controlState, 0)
	for _, control := range controls {
		if control == a.focus {
			continue
		}
		if control.parent == a.focus.parent {
			sameGroup = append(sameGroup, control)
		} else {
			otherGroups = append(otherGroups, control)
		}
	}
	target, _ := a.directionalFocusTargetLocked(
		a.focus,
		sameGroup,
		key,
		false,
	)
	if target == nil {
		target, _ = a.directionalFocusTargetLocked(
			a.focus,
			otherGroups,
			key,
			true,
		)
	}
	if target == nil {
		return false
	}
	a.focus = target
	a.clearInvalidPressesLocked()
	return true
}

func (a *App) directionalFocusTargetLocked(
	current *controlState,
	candidates []*controlState,
	key Key,
	requireUnique bool,
) (*controlState, bool) {
	currentBounds, found := a.snapshotControlBoundsLocked(current)
	if !found {
		return nil, false
	}
	currentX := 2*currentBounds.X + currentBounds.Width
	currentY := 2*currentBounds.Y + currentBounds.Height
	var best *controlState
	bestPrimary, bestCross := 0, 0
	tied := false
	for _, candidate := range candidates {
		bounds, exists := a.snapshotControlBoundsLocked(candidate)
		if !exists {
			continue
		}
		x := 2*bounds.X + bounds.Width
		y := 2*bounds.Y + bounds.Height
		primary, cross := 0, 0
		switch key {
		case KeyLeft:
			primary, cross = currentX-x, abs(currentY-y)
		case KeyRight:
			primary, cross = x-currentX, abs(currentY-y)
		case KeyUp:
			primary, cross = currentY-y, abs(currentX-x)
		case KeyDown:
			primary, cross = y-currentY, abs(currentX-x)
		}
		if primary <= 0 {
			continue
		}
		if best == nil || primary < bestPrimary ||
			(primary == bestPrimary && cross < bestCross) {
			best = candidate
			bestPrimary, bestCross = primary, cross
			tied = false
		} else if primary == bestPrimary && cross == bestCross {
			tied = true
		}
	}
	if requireUnique && tied {
		return nil, false
	}
	return best, best != nil
}

func (a *App) snapshotControlBoundsLocked(
	state *controlState,
) (Rect, bool) {
	if state == nil {
		return Rect{}, false
	}
	for _, control := range a.snapshot.Controls {
		if control.ID == state.id {
			return control.AbsoluteBounds, true
		}
	}
	return Rect{}, false
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (a *App) roleButtonLocked(cancel bool) *controlState {
	scope, _ := a.activeInputScopeLocked()
	for _, state := range a.allFocusableControlsLocked(true) {
		if !a.inInputScopeLocked(state, scope) {
			continue
		}
		behavior, ok := state.behavior.(buttonBehavior)
		if !ok {
			continue
		}
		if (!cancel && behavior.default_) || (cancel && behavior.cancel) {
			return state
		}
	}
	return nil
}

func (a *App) mnemonicControlLocked(key Key) (*controlState, bool) {
	var found *controlState
	activate := false
	scope, _ := a.activeInputScopeLocked()
	var visit func(*controlState)
	visit = func(state *controlState) {
		if found != nil || !a.controlReceivesInputLocked(state) ||
			!a.inInputScopeLocked(state, scope) {
			return
		}
		switch behavior := state.behavior.(type) {
		case buttonBehavior:
			if behavior.mnemonic == key && a.buttonEligibleLocked(state) {
				found = state
				activate = true
				return
			}
		case checkboxBehavior:
			if behavior.mnemonic == key && a.focusEligibleLocked(state) {
				found = state
				activate = true
				return
			}
		case radioButtonBehavior:
			if behavior.mnemonic == key && a.focusEligibleLocked(state) {
				found = state
				activate = true
				return
			}
		case choiceFieldBehavior:
			if behavior.mnemonic == key && a.focusEligibleLocked(state) {
				found = state
				return
			}
		case textBehavior:
			if behavior.mnemonic == key &&
				a.focusEligibleLocked(behavior.target) {
				found = behavior.target
				return
			}
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	visit(scope)
	return found, activate
}

func (a *App) clearInvalidPressesLocked() bool {
	changed := false
	for source, press := range a.pressed {
		if !a.buttonEligibleLocked(press.control) || press.control != a.focus {
			delete(a.pressed, source)
			changed = true
		}
	}
	return changed
}

func (a *App) refreshActionPresentationLocked() {
	minimumChanged := false
	for _, state := range a.controlsByID {
		behavior, ok := state.behavior.(buttonBehavior)
		if !ok || !state.autoMinimum || state.destroyed {
			continue
		}
		minimum := buttonMinimumForLabel(effectiveCommandLabel(
			a.commands[behavior.command],
			behavior.command,
		))
		if state.minimumSize != minimum {
			state.minimumSize = minimum
			minimumChanged = true
		}
	}
	a.repairMenuLocked()
	a.ensureFocusLocked()
	a.clearInvalidPressesLocked()
	if minimumChanged {
		arrangeAllLayoutsLocked(a)
	}
}

func (a *App) activateButton(
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
	if state == nil || state.app != a || !a.buttonEligibleLocked(state) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	behavior := state.behavior.(buttonBehavior)
	router, result, execute := a.resolveCommandLocked(behavior.command, true)
	target := state.id
	a.mu.Unlock()
	if execute {
		result = a.callRouter(ctx, router, Command{
			ID: behavior.command, Target: target, Source: source,
		})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	result = a.applyStandardDialogCommandLocked(behavior.command, target, result)
	return a.associateLocked(requestID, result, behavior.command), nil
}
