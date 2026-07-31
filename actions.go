package expletives

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

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

func (b buttonBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	definition, exists := app.commands[b.command]
	label := effectiveCommandLabel(definition, b.command)
	enabled := exists && definition.Enabled
	left, right := "[", "]"
	switch {
	case !enabled:
		left, right = "(", ")"
	case app.pressedAnyLocked(state):
		left, right = "{", "}"
	case app.focus == state:
		left, right = "<", ">"
	}
	marker := " "
	if b.default_ {
		marker = "*"
	} else if b.cancel {
		marker = "/"
	}
	cells := []string{left, marker, " "}
	cells = append(cells, label.lines[0]...)
	cells = append(cells, " ", right)
	app.paintTextRowsLocked(
		frame,
		state,
		absolute,
		clip,
		[][]string{cells},
		TextAlignCenter,
		TextAlignCenter,
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
	return Size{Width: 5, Height: 1}
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
	return ActionDetails{
		Label:          effectiveCommandLabel(definition, behavior.command).text,
		Command:        behavior.command,
		Enabled:        exists && definition.Enabled,
		DisabledReason: effectiveDisabledReason(definition, exists),
		Checked:        definition.Checked,
		Mnemonic:       behavior.mnemonic,
		Pressed:        a.pressedAnyLocked(state),
		Default:        behavior.default_,
		Cancel:         behavior.cancel,
	}
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
	}
	return true
}

func (a *App) buttonEligibleLocked(state *controlState) bool {
	if state == nil || state.kind != ControlButton ||
		!a.effectivelyVisibleLocked(state) {
		return false
	}
	behavior, ok := state.behavior.(buttonBehavior)
	if !ok {
		return false
	}
	definition, exists := a.commands[behavior.command]
	return exists && definition.Enabled
}

func (a *App) focusableButtonsLocked() []*controlState {
	buttons := make([]*controlState, 0)
	var visit func(*controlState)
	visit = func(state *controlState) {
		if a.buttonEligibleLocked(state) {
			buttons = append(buttons, state)
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	visit(a.root.state)
	return buttons
}

func (a *App) ensureFocusLocked() bool {
	if a.buttonEligibleLocked(a.focus) {
		return false
	}
	previous := a.focus
	a.focus = nil
	buttons := a.focusableButtonsLocked()
	if len(buttons) != 0 {
		a.focus = buttons[0]
	}
	if previous != a.focus {
		a.clearInvalidPressesLocked()
		return true
	}
	return false
}

func (a *App) moveFocusLocked(reverse bool) bool {
	buttons := a.focusableButtonsLocked()
	if len(buttons) == 0 {
		if a.focus == nil {
			return false
		}
		a.focus = nil
		a.clearInvalidPressesLocked()
		return true
	}
	index := -1
	for current, button := range buttons {
		if button == a.focus {
			index = current
			break
		}
	}
	if reverse {
		if index <= 0 {
			index = len(buttons) - 1
		} else {
			index--
		}
	} else {
		index = (index + 1) % len(buttons)
	}
	if a.focus == buttons[index] {
		return false
	}
	a.focus = buttons[index]
	a.clearInvalidPressesLocked()
	return true
}

func (a *App) roleButtonLocked(cancel bool) *controlState {
	for _, state := range a.focusableButtonsLocked() {
		behavior := state.behavior.(buttonBehavior)
		if (!cancel && behavior.default_) || (cancel && behavior.cancel) {
			return state
		}
	}
	return nil
}

func (a *App) mnemonicControlLocked(key Key) (*controlState, bool) {
	var found *controlState
	activate := false
	var visit func(*controlState)
	visit = func(state *controlState) {
		if found != nil || !a.effectivelyVisibleLocked(state) {
			return
		}
		switch behavior := state.behavior.(type) {
		case buttonBehavior:
			if behavior.mnemonic == key && a.buttonEligibleLocked(state) {
				found = state
				activate = true
				return
			}
		case textBehavior:
			if behavior.mnemonic == key &&
				a.buttonEligibleLocked(behavior.target) {
				found = behavior.target
				return
			}
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	visit(a.root.state)
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
		minimum := Size{
			Width: len(effectiveCommandLabel(
				a.commands[behavior.command],
				behavior.command,
			).lines[0]) + 5,
			Height: 1,
		}
		if state.minimumSize != minimum {
			state.minimumSize = minimum
			minimumChanged = true
		}
	}
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
	router, result, execute := a.resolveCommandLocked(behavior.command)
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
	return a.associateLocked(requestID, result, behavior.command), nil
}
