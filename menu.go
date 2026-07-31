package expletives

import (
	"context"
	"fmt"
	"strings"
)

// MenuItemKind identifies one immutable MenuItem role.
type MenuItemKind string

const (
	// MenuItemCommand invokes one shared registered command.
	MenuItemCommand MenuItemKind = "command"
	// MenuItemSeparator is a non-selectable structural divider.
	MenuItemSeparator MenuItemKind = "separator"
	// MenuItemSubmenu opens one immutable child Menu.
	MenuItemSubmenu MenuItemKind = "submenu"
)

// MenuItem is one immutable copied menu descriptor.
type MenuItem struct {
	// Key is stable within one complete MenuBar tree.
	Key string
	// Kind selects command, separator, or submenu behavior.
	Kind MenuItemKind
	// Label is used only by submenu items. Command labels come from the
	// CommandDefinition.
	Label string
	// Command is required only for command items.
	Command CommandID
	// Mnemonic is an optional ASCII sibling access key.
	Mnemonic Key
	// Menu is required only for submenu items.
	Menu *Menu
}

// MenuOptions configures one immutable popup Menu model.
type MenuOptions struct {
	Items []MenuItem
}

// MenuBarOptions configures the persistent menu-session control.
type MenuBarOptions struct {
	PanelOptions
	// Items are top-level submenu descriptors.
	Items []MenuItem
	// PopupStyle defaults to "menu.popup".
	PopupStyle StyleID
	// BorderStyle defaults to "menu.border".
	BorderStyle StyleID
	// MnemonicStyle defaults to "menu.mnemonic".
	MnemonicStyle StyleID
	// FocusedStyle defaults to "menu.focused".
	FocusedStyle StyleID
	// FocusedMnemonicStyle defaults to "menu.focused_mnemonic".
	FocusedMnemonicStyle StyleID
	// DisabledStyle defaults to "menu.disabled".
	DisabledStyle StyleID
	// FocusedDisabledStyle defaults to "menu.focused_disabled".
	FocusedDisabledStyle StyleID
	// ShadowStyle defaults to "menu.shadow".
	ShadowStyle StyleID
}

// Menu is a copy-safe immutable popup model. It is not a Control.
type Menu struct {
	items []MenuItem
}

// MenuBar is a copy-safe, non-container leaf Control.
type MenuBar struct{ controlHandle }

type menuBarBehavior struct {
	items                []MenuItem
	popupStyle           StyleID
	borderStyle          StyleID
	mnemonicStyle        StyleID
	focusedStyle         StyleID
	focusedMnemonicStyle StyleID
	disabledStyle        StyleID
	focusedDisabledStyle StyleID
	shadowStyle          StyleID
}

type menuSession struct {
	bar        *controlState
	rootIndex  int
	menus      []*Menu
	selected   []int
	priorFocus *controlState
}

// NewMenu validates and copies one immutable popup Menu.
func NewMenu(options MenuOptions) (*Menu, error) {
	items, err := normalizeMenuItems(options.Items, false)
	if err != nil {
		return nil, err
	}
	return &Menu{items: items}, nil
}

// Items returns a caller-owned copy of this immutable Menu's direct items.
func (m *Menu) Items() []MenuItem {
	if m == nil {
		return nil
	}
	return append([]MenuItem(nil), m.items...)
}

// NewMenuBar constructs and atomically inserts one MenuBar.
func NewMenuBar(
	parent Container,
	options MenuBarOptions,
) (*MenuBar, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	bar, err := tx.NewMenuBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return bar, nil
}

// NewMenuBar records construction of one provisional MenuBar.
func (t *Transaction) NewMenuBar(
	parent Container,
	options MenuBarOptions,
) (*MenuBar, error) {
	if err := t.usable(); err != nil {
		return nil, err
	}
	if parent == nil || parent.containerState() == nil ||
		parent.containerState() != t.app.root.state {
		return nil, fmt.Errorf(
			"%w: MenuBar must be parented directly by App.Root()",
			ErrInvalidParent,
		)
	}
	if options.Bounds != (Rect{}) || options.MinimumSize != (Size{}) {
		return nil, fmt.Errorf(
			"%w: MenuBar geometry is derived from the application surface",
			ErrInvalidGeometry,
		)
	}
	items, err := normalizeMenuItems(options.Items, true)
	if err != nil {
		return nil, err
	}
	popupStyle, err := normalizeStyleID(options.PopupStyle, "menu.popup")
	if err != nil {
		return nil, err
	}
	borderStyle, err := normalizeStyleID(options.BorderStyle, "menu.border")
	if err != nil {
		return nil, err
	}
	mnemonicStyle, err := normalizeStyleID(
		options.MnemonicStyle,
		"menu.mnemonic",
	)
	if err != nil {
		return nil, err
	}
	focusedStyle, err := normalizeStyleID(
		options.FocusedStyle,
		"menu.focused",
	)
	if err != nil {
		return nil, err
	}
	focusedMnemonicStyle, err := normalizeStyleID(
		options.FocusedMnemonicStyle,
		"menu.focused_mnemonic",
	)
	if err != nil {
		return nil, err
	}
	disabledStyle, err := normalizeStyleID(
		options.DisabledStyle,
		"menu.disabled",
	)
	if err != nil {
		return nil, err
	}
	focusedDisabledStyle, err := normalizeStyleID(
		options.FocusedDisabledStyle,
		"menu.focused_disabled",
	)
	if err != nil {
		return nil, err
	}
	shadowStyle, err := normalizeStyleID(options.ShadowStyle, "menu.shadow")
	if err != nil {
		return nil, err
	}
	behavior := menuBarBehavior{
		items: items, popupStyle: popupStyle, borderStyle: borderStyle,
		mnemonicStyle: mnemonicStyle, focusedStyle: focusedStyle,
		focusedMnemonicStyle: focusedMnemonicStyle,
		disabledStyle:        disabledStyle,
		focusedDisabledStyle: focusedDisabledStyle,
		shadowStyle:          shadowStyle,
	}
	options.Bounds = menuBarSurfaceRect(t.app.size)
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlMenuBar,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	bar := &MenuBar{controlHandle: controlHandle{state: state}}
	state.control = bar
	return bar, nil
}

// Items returns a caller-owned copy of the MenuBar's top-level items.
func (b *MenuBar) Items() []MenuItem {
	if b == nil || b.state == nil || b.state.app == nil {
		return nil
	}
	b.state.app.mu.RLock()
	defer b.state.app.mu.RUnlock()
	behavior, ok := b.state.behavior.(menuBarBehavior)
	if !ok || b.state.aborted {
		return nil
	}
	return append([]MenuItem(nil), behavior.items...)
}

// Open opens the first top-level popup and gives focus to this MenuBar.
func (b *MenuBar) Open() error {
	return b.setOpen(true)
}

// Close closes this MenuBar's popup session and restores prior focus.
func (b *MenuBar) Close() error {
	return b.setOpen(false)
}

func (b *MenuBar) setOpen(open bool) error {
	if b == nil || b.state == nil || b.state.app == nil {
		return ErrInvalidControl
	}
	app := b.state.app
	ctx := context.Background()
	if err := app.beginMutation(ctx); err != nil {
		return err
	}
	defer app.endMutation()
	app.mu.Lock()
	defer app.mu.Unlock()
	if err := b.state.mutableLocked(); err != nil {
		return err
	}
	changed := false
	if open {
		changed = app.openMenuBarLocked(b.state, 0)
	} else if app.menu != nil && app.menu.bar == b.state {
		changed = app.closeMenuLocked()
	}
	if changed {
		app.publishLocked(nil)
	}
	return nil
}

func normalizeMenuItems(
	source []MenuItem,
	topLevel bool,
) ([]MenuItem, error) {
	if len(source) == 0 || len(source) > MaxMenuItemsPerMenu {
		return nil, fmt.Errorf(
			"%w: Menu requires 1..%d items",
			ErrInvalidControl,
			MaxMenuItemsPerMenu,
		)
	}
	items := append([]MenuItem(nil), source...)
	keys := make(map[string]bool, len(items))
	mnemonics := make(map[Key]bool, len(items))
	for index := range items {
		item := &items[index]
		if !validBoundedIdentifier(item.Key) || keys[item.Key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate MenuItem key %q",
				ErrInvalidControl,
				item.Key,
			)
		}
		keys[item.Key] = true
		mnemonic, err := normalizeMnemonic(item.Mnemonic)
		if err != nil {
			return nil, err
		}
		item.Mnemonic = mnemonic
		if mnemonic != "" {
			if mnemonics[mnemonic] {
				return nil, fmt.Errorf(
					"%w: duplicate sibling Menu mnemonic %q",
					ErrInvalidControl,
					mnemonic,
				)
			}
			mnemonics[mnemonic] = true
		}
		switch item.Kind {
		case MenuItemCommand:
			if topLevel ||
				!validBoundedIdentifier(string(item.Command)) ||
				item.Label != "" || item.Menu != nil {
				return nil, fmt.Errorf(
					"%w: malformed command MenuItem %q",
					ErrInvalidControl,
					item.Key,
				)
			}
		case MenuItemSeparator:
			if topLevel || item.Label != "" || item.Command != "" ||
				item.Mnemonic != "" || item.Menu != nil {
				return nil, fmt.Errorf(
					"%w: malformed separator MenuItem %q",
					ErrInvalidControl,
					item.Key,
				)
			}
		case MenuItemSubmenu:
			if item.Command != "" || item.Menu == nil {
				return nil, fmt.Errorf(
					"%w: malformed submenu MenuItem %q",
					ErrInvalidControl,
					item.Key,
				)
			}
			label, err := normalizeDisplayText(item.Label, false)
			if err != nil || label.text == "" {
				return nil, fmt.Errorf(
					"%w: invalid submenu label",
					ErrInvalidControl,
				)
			}
			item.Label = label.text
		default:
			return nil, fmt.Errorf(
				"%w: invalid MenuItem kind %q",
				ErrInvalidControl,
				item.Kind,
			)
		}
	}
	return items, nil
}

func (b menuBarBehavior) clientInset() int { return 0 }

func (b menuBarBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	x := absolute.X + 1
	for index, item := range b.items {
		label, _ := normalizeDisplayText(item.Label, false)
		style := state.style
		mnemonicStyle := b.mnemonicStyle
		if app.menu != nil && app.menu.bar == state &&
			app.menu.rootIndex == index {
			style = b.focusedStyle
			mnemonicStyle = b.focusedMnemonicStyle
		}
		cells := append([]string{" "}, label.lines[0]...)
		cells = append(cells, " ")
		app.paintMenuCellsLocked(
			frame,
			clip,
			x,
			absolute.Y,
			cells,
			style,
			state.id,
		)
		app.paintMnemonicLocked(
			frame,
			clip,
			x+1,
			absolute.Y,
			label.lines[0],
			item.Mnemonic,
			mnemonicStyle,
			state.id,
		)
		x += len(cells)
	}
}

func (b menuBarBehavior) details() ControlDetails {
	return ControlDetails{
		Version: ControlDetailsVersion,
		MenuBar: &MenuBarDetails{
			Entries:      []MenuEntryDetails{},
			OpenPath:     []string{},
			SelectedPath: []string{},
		},
	}
}

func (b menuBarBehavior) intrinsicMinimum() Size {
	width := 0
	for _, item := range b.items {
		label, _ := normalizeDisplayText(item.Label, false)
		width += len(label.lines[0]) + 2
	}
	return Size{Width: width, Height: 1}
}

func (b menuBarBehavior) additionalStyles() []StyleID {
	return []StyleID{
		b.popupStyle,
		b.borderStyle,
		b.mnemonicStyle,
		b.focusedStyle,
		b.focusedMnemonicStyle,
		b.disabledStyle,
		b.focusedDisabledStyle,
		b.shadowStyle,
	}
}

func (a *App) menuBarDetailsLocked(
	state *controlState,
	behavior menuBarBehavior,
) MenuBarDetails {
	details := MenuBarDetails{
		Entries:      []MenuEntryDetails{},
		OpenPath:     []string{},
		SelectedPath: []string{},
	}
	selected := make(map[string]bool)
	open := make(map[string]bool)
	if a.menu != nil && a.menu.bar == state {
		root := behavior.items[a.menu.rootIndex]
		details.SelectedPath = append(details.SelectedPath, root.Key)
		selected[root.Key] = true
		if len(a.menu.menus) != 0 {
			details.OpenPath = append(details.OpenPath, root.Key)
			open[root.Key] = true
		}
		for depth, menu := range a.menu.menus {
			index := a.menu.selected[depth]
			if index < 0 || index >= len(menu.items) {
				continue
			}
			key := menu.items[index].Key
			selected[key] = true
			details.SelectedPath = append(details.SelectedPath, key)
			if depth+1 < len(a.menu.menus) {
				open[key] = true
				details.OpenPath = append(details.OpenPath, key)
			}
		}
	}
	var appendMenu func([]MenuItem, string, int)
	appendMenu = func(items []MenuItem, parent string, depth int) {
		for _, item := range items {
			entry := a.menuEntryDetailsLocked(item, parent, depth)
			entry.Selected = selected[item.Key]
			entry.Open = open[item.Key]
			details.Entries = append(details.Entries, entry)
			if item.Kind == MenuItemSubmenu {
				appendMenu(item.Menu.items, item.Key, depth+1)
			}
		}
	}
	appendMenu(behavior.items, "", 0)
	return details
}

func (a *App) menuEntryDetailsLocked(
	item MenuItem,
	parent string,
	depth int,
) MenuEntryDetails {
	entry := MenuEntryDetails{
		Key: item.Key, ParentKey: parent, Depth: depth, Kind: item.Kind,
		Mnemonic: item.Mnemonic,
	}
	switch item.Kind {
	case MenuItemCommand:
		definition, exists := a.commands[item.Command]
		entry.Label = effectiveCommandLabel(definition, item.Command).text
		entry.Command = item.Command
		entry.Enabled = exists && definition.Enabled
		entry.DisabledReason = effectiveDisabledReason(definition, exists)
		entry.Checked = definition.Checked
		entry.Chord = a.firstChordLocked(item.Command)
	case MenuItemSubmenu:
		entry.Label = item.Label
		entry.Enabled = true
		entry.ChildCount = len(item.Menu.items)
	}
	return entry
}

func (a *App) openMenuBarLocked(
	state *controlState,
	rootIndex int,
) bool {
	return a.startMenuSessionLocked(state, rootIndex, true)
}

func (a *App) activateMenuBarLocked(
	state *controlState,
	rootIndex int,
) bool {
	return a.startMenuSessionLocked(state, rootIndex, false)
}

func (a *App) startMenuSessionLocked(
	state *controlState,
	rootIndex int,
	openPopup bool,
) bool {
	behavior, ok := state.behavior.(menuBarBehavior)
	if !ok || !a.effectivelyVisibleLocked(state) ||
		rootIndex < 0 || rootIndex >= len(behavior.items) {
		return false
	}
	prior := a.focus
	if a.menu != nil {
		if a.menu.bar == state && a.menu.rootIndex == rootIndex &&
			(len(a.menu.menus) != 0) == openPopup {
			return false
		}
		prior = a.menu.priorFocus
	}
	root := behavior.items[rootIndex]
	a.menu = &menuSession{
		bar: state, rootIndex: rootIndex,
		priorFocus: prior,
	}
	if openPopup {
		a.menu.menus = []*Menu{root.Menu}
		a.menu.selected = []int{a.firstSelectableMenuItemLocked(root.Menu)}
	}
	a.focus = state
	a.clearInvalidPressesLocked()
	return true
}

func (a *App) closeMenuLocked() bool {
	if a.menu == nil {
		return false
	}
	prior := a.menu.priorFocus
	a.menu = nil
	a.focus = nil
	if a.buttonEligibleLocked(prior) {
		a.focus = prior
	} else {
		a.ensureFocusLocked()
	}
	a.clearInvalidPressesLocked()
	return true
}

func (a *App) repairMenuLocked() bool {
	if a.menu == nil {
		return false
	}
	if a.menu.bar == nil ||
		!a.effectivelyVisibleLocked(a.menu.bar) {
		return a.closeMenuLocked()
	}
	changed := false
	for depth, menu := range a.menu.menus {
		index := a.menu.selected[depth]
		if index >= 0 && index < len(menu.items) &&
			menuItemSelectable(menu.items[index]) {
			continue
		}
		a.menu.selected[depth] = a.firstSelectableMenuItemLocked(menu)
		changed = true
	}
	a.focus = a.menu.bar
	return changed
}

func menuItemSelectable(item MenuItem) bool {
	return item.Kind != MenuItemSeparator
}

func (a *App) menuItemActivatableLocked(item MenuItem) bool {
	switch item.Kind {
	case MenuItemSubmenu:
		return item.Menu != nil
	case MenuItemCommand:
		definition, exists := a.commands[item.Command]
		return exists && definition.Enabled
	default:
		return false
	}
}

func (a *App) firstSelectableMenuItemLocked(menu *Menu) int {
	if menu == nil {
		return -1
	}
	for index, item := range menu.items {
		if menuItemSelectable(item) {
			return index
		}
	}
	return -1
}

func (a *App) lastSelectableMenuItemLocked(menu *Menu) int {
	if menu == nil {
		return -1
	}
	for index := len(menu.items) - 1; index >= 0; index-- {
		if menuItemSelectable(menu.items[index]) {
			return index
		}
	}
	return -1
}

func (a *App) moveMenuSelectionLocked(delta int) bool {
	if a.menu == nil || len(a.menu.menus) == 0 {
		return false
	}
	depth := len(a.menu.menus) - 1
	menu := a.menu.menus[depth]
	if len(menu.items) == 0 {
		return false
	}
	start := a.menu.selected[depth]
	index := start
	if index < 0 {
		if delta < 0 {
			index = 0
		} else {
			index = len(menu.items) - 1
		}
	}
	for range len(menu.items) {
		index = (index + delta + len(menu.items)) % len(menu.items)
		if menuItemSelectable(menu.items[index]) {
			if start == index {
				return false
			}
			a.menu.selected[depth] = index
			return true
		}
	}
	return false
}

func (a *App) switchRootMenuLocked(delta int) bool {
	if a.menu == nil {
		return false
	}
	behavior := a.menu.bar.behavior.(menuBarBehavior)
	index := (a.menu.rootIndex + delta + len(behavior.items)) %
		len(behavior.items)
	return a.startMenuSessionLocked(
		a.menu.bar,
		index,
		len(a.menu.menus) != 0,
	)
}

func (a *App) openSelectedSubmenuLocked() bool {
	if a.menu == nil || len(a.menu.menus) == 0 {
		return false
	}
	depth := len(a.menu.menus) - 1
	index := a.menu.selected[depth]
	if index < 0 {
		return false
	}
	item := a.menu.menus[depth].items[index]
	if item.Kind != MenuItemSubmenu {
		return false
	}
	a.menu.menus = append(a.menu.menus, item.Menu)
	a.menu.selected = append(
		a.menu.selected,
		a.firstSelectableMenuItemLocked(item.Menu),
	)
	return true
}

func (a *App) closeOneMenuLevelLocked() bool {
	if a.menu == nil {
		return false
	}
	if len(a.menu.menus) <= 1 {
		return a.closeMenuLocked()
	}
	a.menu.menus = a.menu.menus[:len(a.menu.menus)-1]
	a.menu.selected = a.menu.selected[:len(a.menu.selected)-1]
	return true
}

func (a *App) menuMnemonicLocked(key Key) (CommandID, bool) {
	if a.menu == nil || len(a.menu.menus) == 0 {
		return "", false
	}
	depth := len(a.menu.menus) - 1
	menu := a.menu.menus[depth]
	for index, item := range menu.items {
		if item.Mnemonic != key || !a.menuItemActivatableLocked(item) {
			continue
		}
		a.menu.selected[depth] = index
		if item.Kind == MenuItemSubmenu {
			a.openSelectedSubmenuLocked()
			return "", true
		}
		command := item.Command
		a.closeMenuLocked()
		return command, true
	}
	return "", false
}

func (a *App) topMenuMnemonicLocked(key Key) (bool, bool) {
	for _, state := range a.controlsByID {
		behavior, ok := state.behavior.(menuBarBehavior)
		if !ok || !a.effectivelyVisibleLocked(state) {
			continue
		}
		for index, item := range behavior.items {
			if item.Mnemonic == key {
				return true, a.openMenuBarLocked(state, index)
			}
		}
	}
	return false, false
}

func (a *App) dispatchMenuKeyLocked(
	key Key,
	held map[Key]bool,
) (CommandID, bool, bool) {
	toggle := (key == "f10" && noHeldModifiers(held)) ||
		(key == KeySpace &&
			held[KeyControl] &&
			!held[KeyAlt] &&
			!held[KeyMeta] &&
			!held[KeyShift])
	if toggle {
		if a.menu != nil {
			return "", true, a.closeMenuLocked()
		}
		bar := a.firstMenuBarLocked()
		if bar == nil {
			return "", false, false
		}
		return "", true, a.activateMenuBarLocked(bar, 0)
	}
	if mnemonicKeyEvent(key, held) {
		found, changed := a.topMenuMnemonicLocked(key)
		if found {
			return "", true, changed
		}
		if a.menu == nil {
			return "", false, false
		}
		return "", true, false
	}
	if a.menu == nil {
		return "", false, false
	}
	if !noHeldModifiers(held) {
		return "", true, false
	}
	if len(a.menu.menus) == 0 {
		switch key {
		case KeyLeft:
			return "", true, a.switchRootMenuLocked(-1)
		case KeyRight:
			return "", true, a.switchRootMenuLocked(1)
		case KeyDown, KeyEnter:
			return "", true, a.openMenuBarLocked(
				a.menu.bar,
				a.menu.rootIndex,
			)
		case KeyEscape:
			return "", true, a.closeMenuLocked()
		default:
			if len(key) == 1 {
				behavior := a.menu.bar.behavior.(menuBarBehavior)
				for index, item := range behavior.items {
					if item.Mnemonic == key {
						return "", true, a.openMenuBarLocked(
							a.menu.bar,
							index,
						)
					}
				}
			}
			return "", true, false
		}
	}
	switch key {
	case KeyUp:
		return "", true, a.moveMenuSelectionLocked(-1)
	case KeyDown:
		return "", true, a.moveMenuSelectionLocked(1)
	case KeyHome:
		depth := len(a.menu.menus) - 1
		next := a.firstSelectableMenuItemLocked(a.menu.menus[depth])
		changed := a.menu.selected[depth] != next
		a.menu.selected[depth] = next
		return "", true, changed
	case KeyEnd:
		depth := len(a.menu.menus) - 1
		next := a.lastSelectableMenuItemLocked(a.menu.menus[depth])
		changed := a.menu.selected[depth] != next
		a.menu.selected[depth] = next
		return "", true, changed
	case KeyRight:
		if a.openSelectedSubmenuLocked() {
			return "", true, true
		}
		if len(a.menu.menus) == 1 {
			return "", true, a.switchRootMenuLocked(1)
		}
		return "", true, false
	case KeyLeft:
		if len(a.menu.menus) > 1 {
			return "", true, a.closeOneMenuLevelLocked()
		}
		return "", true, a.switchRootMenuLocked(-1)
	case KeyEscape:
		return "", true, a.closeOneMenuLevelLocked()
	case KeyEnter:
		return a.activateSelectedMenuItemLocked()
	default:
		if len(key) == 1 {
			command, handled := a.menuMnemonicLocked(key)
			if handled {
				return command, true, true
			}
		}
		return "", true, false
	}
}

func (a *App) activateSelectedMenuItemLocked() (
	CommandID,
	bool,
	bool,
) {
	if a.menu == nil || len(a.menu.menus) == 0 {
		return "", false, false
	}
	depth := len(a.menu.menus) - 1
	index := a.menu.selected[depth]
	menu := a.menu.menus[depth]
	if index < 0 || index >= len(menu.items) {
		return "", true, false
	}
	item := menu.items[index]
	if !a.menuItemActivatableLocked(item) {
		return "", true, false
	}
	if item.Kind == MenuItemSubmenu {
		return "", true, a.openSelectedSubmenuLocked()
	}
	command := item.Command
	a.closeMenuLocked()
	return command, true, true
}

func (a *App) firstMenuBarLocked() *controlState {
	var result *controlState
	var visit func(*controlState)
	visit = func(state *controlState) {
		if result != nil {
			return
		}
		if state.kind == ControlMenuBar &&
			a.effectivelyVisibleLocked(state) {
			result = state
			return
		}
		for _, child := range state.children {
			visit(child)
		}
	}
	visit(a.root.state)
	return result
}

func (a *App) paintMenuOverlayLocked(frame *IntendedFrame) {
	if a.menu == nil || len(a.menu.menus) == 0 ||
		frame.Size.Width <= 0 || frame.Size.Height <= 0 {
		return
	}
	behavior := a.menu.bar.behavior.(menuBarBehavior)
	topOffset := 0
	for index := 0; index < a.menu.rootIndex; index++ {
		label, _ := normalizeDisplayText(behavior.items[index].Label, false)
		topOffset += len(label.lines[0]) + 2
	}
	desiredX := 1 + topOffset
	desiredY := 1
	var parentRect Rect
	var parentRow int
	for depth, menu := range a.menu.menus {
		submenu := depth > 0
		rect, start := a.menuPopupGeometryLocked(
			menu,
			a.menu.selected[depth],
			desiredX,
			desiredY,
			submenu,
			parentRect,
		)
		a.paintOneMenuLocked(
			frame,
			a.menu.bar,
			behavior,
			menu,
			a.menu.selected[depth],
			rect,
			start,
		)
		parentRect = rect
		parentRow = rect.Y + 1 + a.menu.selected[depth] - start
		desiredX = rect.X + 2
		desiredY = parentRow + 1
	}
}

func (a *App) menuPopupGeometryLocked(
	menu *Menu,
	selected int,
	desiredX int,
	desiredY int,
	submenu bool,
	parent Rect,
) (Rect, int) {
	width := 8
	for _, item := range menu.items {
		width = max(width, a.menuItemDisplayWidthLocked(item)+7)
	}
	width = min(width, a.size.Width)
	height := min(len(menu.items)+2, a.size.Height)
	start := 0
	rows := max(0, height-2)
	if rows > 0 && selected >= rows {
		start = selected - rows + 1
	}
	if start > max(0, len(menu.items)-rows) {
		start = max(0, len(menu.items)-rows)
	}
	x := desiredX
	if submenu && x+width > a.size.Width {
		x = a.size.Width - width
	}
	x = max(0, min(x, a.size.Width-width))
	y := max(0, min(desiredY, a.size.Height-height))
	return Rect{X: x, Y: y, Width: width, Height: height}, start
}

func (a *App) menuItemDisplayWidthLocked(item MenuItem) int {
	entry := a.menuEntryDetailsLocked(item, "", 0)
	width := len(normalizeCells(entry.Label))
	if entry.Chord != nil {
		width += len(displayChord(*entry.Chord))
	} else if entry.Kind == MenuItemSubmenu {
		width++
	}
	return width
}

func (a *App) paintOneMenuLocked(
	frame *IntendedFrame,
	state *controlState,
	behavior menuBarBehavior,
	menu *Menu,
	selected int,
	rect Rect,
	start int,
) {
	if rect.Empty() {
		return
	}
	surface := Rect{Width: frame.Size.Width, Height: frame.Size.Height}
	a.fillLocked(
		frame,
		Rect{
			X:      rect.X + rect.Width,
			Y:      rect.Y + 1,
			Width:  2,
			Height: max(0, rect.Height),
		}.Intersect(surface),
		behavior.shadowStyle,
		state.id,
	)
	a.fillLocked(
		frame,
		Rect{
			X:      rect.X + 2,
			Y:      rect.Y + rect.Height,
			Width:  max(0, rect.Width),
			Height: 1,
		}.Intersect(surface),
		behavior.shadowStyle,
		state.id,
	)
	a.fillLocked(frame, rect, behavior.popupStyle, state.id)
	a.paintBorderLocked(
		frame,
		rect,
		rect,
		state,
		borderBehavior{
			borderStyle: behavior.borderStyle,
			form:        BorderSingle,
		},
	)
	rows := max(0, rect.Height-2)
	end := min(len(menu.items), start+rows)
	for index := start; index < end; index++ {
		y := rect.Y + 1 + index - start
		item := menu.items[index]
		if item.Kind == MenuItemSeparator {
			for x := rect.X; x < rect.X+rect.Width; x++ {
				glyph := "─"
				if x == rect.X {
					glyph = "├"
				} else if x == rect.X+rect.Width-1 {
					glyph = "┤"
				}
				a.setClippedCellLocked(
					frame, rect, x, y, glyph,
					behavior.borderStyle,
					a.styles[behavior.borderStyle],
					state.id,
				)
			}
			continue
		}
		entry := a.menuEntryDetailsLocked(item, "", 0)
		style := behavior.popupStyle
		mnemonicStyle := behavior.mnemonicStyle
		if index == selected {
			style = behavior.focusedStyle
			mnemonicStyle = behavior.focusedMnemonicStyle
		}
		if !entry.Enabled {
			style = behavior.disabledStyle
			mnemonicStyle = behavior.disabledStyle
			if index == selected {
				style = behavior.focusedDisabledStyle
				mnemonicStyle = behavior.focusedDisabledStyle
			}
		}
		check := " "
		if entry.Checked {
			check = "√"
		}
		label := normalizeCells(entry.Label)
		available := max(0, rect.Width-2)
		cells := make([]string, available)
		for index := range cells {
			cells[index] = " "
		}
		if available > 0 {
			cells[0] = check
		}
		labelStart := 2
		for offset, cell := range label {
			if labelStart+offset >= available {
				break
			}
			cells[labelStart+offset] = cell
		}
		trailer := []string{}
		if entry.Kind == MenuItemSubmenu {
			trailer = []string{"►"}
		} else if entry.Chord != nil {
			trailer = displayChord(*entry.Chord)
		}
		trailerStart := available - len(trailer) - 1
		if trailerStart < labelStart+len(label)+1 {
			trailerStart = labelStart + len(label) + 1
		}
		for offset, cell := range trailer {
			if trailerStart+offset >= 0 && trailerStart+offset < available {
				cells[trailerStart+offset] = cell
			}
		}
		a.paintMenuCellsLocked(
			frame,
			rect,
			rect.X+1,
			y,
			cells,
			style,
			state.id,
		)
		a.paintMnemonicLocked(
			frame,
			rect,
			rect.X+1+labelStart,
			y,
			label,
			item.Mnemonic,
			mnemonicStyle,
			state.id,
		)
	}
	if start > 0 && rect.Width > 2 {
		a.setClippedCellLocked(
			frame, rect, rect.X+rect.Width-2, rect.Y, "^",
			behavior.borderStyle, a.styles[behavior.borderStyle], state.id,
		)
	}
	if end < len(menu.items) && rect.Width > 2 {
		a.setClippedCellLocked(
			frame, rect, rect.X+rect.Width-2, rect.Y+rect.Height-1, "v",
			behavior.borderStyle, a.styles[behavior.borderStyle], state.id,
		)
	}
}

func menuBarSurfaceRect(size Size) Rect {
	return Rect{Width: size.Width, Height: min(1, size.Height)}
}

func normalizeCells(text string) []string {
	normalized, _ := normalizeDisplayText(text, false)
	if len(normalized.lines) == 0 {
		return nil
	}
	return normalized.lines[0]
}

func (a *App) paintMenuCellsLocked(
	frame *IntendedFrame,
	clip Rect,
	x int,
	y int,
	cells []string,
	style StyleID,
	owner ControlID,
) {
	resolved := a.styles[style]
	for index, cell := range cells {
		a.setClippedCellLocked(
			frame, clip, x+index, y, cell, style, resolved, owner,
		)
	}
}

func (a *App) paintMnemonicLocked(
	frame *IntendedFrame,
	clip Rect,
	x int,
	y int,
	cells []string,
	mnemonic Key,
	style StyleID,
	owner ControlID,
) {
	if mnemonic == "" {
		return
	}
	for index, cell := range cells {
		if Key(strings.ToLower(cell)) != mnemonic {
			continue
		}
		a.setClippedCellLocked(
			frame,
			clip,
			x+index,
			y,
			cell,
			style,
			a.styles[style],
			owner,
		)
		return
	}
}

func (a *App) controlAbsoluteLocked(state *controlState) Rect {
	if state == nil {
		return Rect{}
	}
	rect := state.bounds
	for parent := state.parent; parent != nil; parent = parent.parent {
		inset := parent.behavior.clientInset()
		rect.X += parent.bounds.X + min(inset, parent.bounds.Width)
		rect.Y += parent.bounds.Y + min(inset, parent.bounds.Height)
	}
	return rect
}

// Used only while Transaction holds the App lock.
func validateMenuBarTreeLocked(
	app *App,
	behavior menuBarBehavior,
	requireCommands bool,
) error {
	seenMenus := make(map[*Menu]bool)
	seenKeys := make(map[string]bool)
	menuCount := 0
	itemCount := len(behavior.items)
	var visit func(*Menu, int) error
	visit = func(menu *Menu, depth int) error {
		if menu == nil || seenMenus[menu] || depth > MaxMenuDepth {
			return fmt.Errorf(
				"%w: cyclic, aliased, or over-depth Menu tree",
				ErrInvalidControl,
			)
		}
		seenMenus[menu] = true
		menuCount++
		itemCount += len(menu.items)
		if menuCount > MaxMenus || itemCount > MaxMenuItems {
			return ErrControlCapacity
		}
		for _, item := range menu.items {
			if seenKeys[item.Key] {
				return fmt.Errorf(
					"%w: duplicate MenuItem key %q",
					ErrInvalidControl,
					item.Key,
				)
			}
			seenKeys[item.Key] = true
			if item.Kind == MenuItemCommand && requireCommands {
				if _, exists := app.commands[item.Command]; !exists {
					return fmt.Errorf(
						"%w: Menu command %q is not registered",
						ErrInvalidControl,
						item.Command,
					)
				}
			}
			if item.Kind == MenuItemSubmenu {
				if err := visit(item.Menu, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, root := range behavior.items {
		if seenKeys[root.Key] {
			return fmt.Errorf(
				"%w: duplicate MenuItem key %q",
				ErrInvalidControl,
				root.Key,
			)
		}
		seenKeys[root.Key] = true
		if err := visit(root.Menu, 1); err != nil {
			return err
		}
	}
	return nil
}
