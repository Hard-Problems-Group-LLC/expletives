package expletives

import (
	"context"
	"fmt"
	"strings"
)

// Tab is one copied page descriptor. Page must be a unique direct Panel child
// of the TabbedPanel or Notebook that receives the descriptor.
type Tab struct {
	Key            string
	Value          string
	Label          string
	Mnemonic       Key
	Page           *Panel
	Disabled       bool
	DisabledReason string
}

// TabbedPanelOptions configures one stacked-page container and its strip.
type TabbedPanelOptions struct {
	PanelOptions
	BorderStyle      StyleID
	BorderForm       BorderForm
	BorderForeground *Color
	BorderBackground *Color
	ChangeCommand    CommandID
}

// TabbedPanel is a copy-safe focusable stacked-page container.
type TabbedPanel struct{ containerHandle }

// Notebook is the Notebook naming variant of TabbedPanel.
type Notebook struct{ containerHandle }

type tabEntry struct {
	tab   Tab
	label normalizedDisplayText
	page  *controlState
}

type tabbedPanelBehavior struct {
	border        borderBehavior
	tabs          []tabEntry
	selected      string
	current       string
	changeCommand CommandID
}

type tabRenderEntry struct {
	bounds  Rect
	omitted bool
	clipped bool
	cells   []string
}

type tabRenderPlan struct {
	entries         []tabRenderEntry
	leadingOmitted  bool
	trailingOmitted bool
}

// NewTabbedPanel constructs and atomically inserts a TabbedPanel.
func NewTabbedPanel(
	parent Container,
	options TabbedPanelOptions,
) (*TabbedPanel, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewTabbedPanel(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewNotebook constructs and atomically inserts a Notebook.
func NewNotebook(
	parent Container,
	options TabbedPanelOptions,
) (*Notebook, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewNotebook(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewTabbedPanel records construction of a provisional TabbedPanel.
func (t *Transaction) NewTabbedPanel(
	parent Container,
	options TabbedPanelOptions,
) (*TabbedPanel, error) {
	panel, err := t.newTabbedPanel(parent, options, ControlTabbedPanel)
	if err != nil {
		return nil, err
	}
	control := &TabbedPanel{containerHandle: panel.containerHandle}
	panel.state.control = control
	panel.state.container = control
	return control, nil
}

// NewNotebook records construction of a provisional Notebook.
func (t *Transaction) NewNotebook(
	parent Container,
	options TabbedPanelOptions,
) (*Notebook, error) {
	panel, err := t.newTabbedPanel(parent, options, ControlNotebook)
	if err != nil {
		return nil, err
	}
	control := &Notebook{containerHandle: panel.containerHandle}
	panel.state.control = control
	panel.state.container = control
	return control, nil
}

func (t *Transaction) newTabbedPanel(
	parent Container,
	options TabbedPanelOptions,
	kind ControlKind,
) (*Panel, error) {
	if err := validateOptionalCommand(options.ChangeCommand); err != nil {
		return nil, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		kind,
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
		kind,
		tabbedPanelBehavior{
			border:        border,
			tabs:          []tabEntry{},
			changeCommand: options.ChangeCommand,
		},
	)
	if err != nil {
		return nil, err
	}
	if options.MinimumSize == (Size{}) {
		panel.state.autoMinimum = true
		panel.state.minimumSize =
			panel.state.behavior.(intrinsicMinimumBehavior).intrinsicMinimum()
	}
	return panel, nil
}

func (b tabbedPanelBehavior) controlBorder() borderBehavior {
	return b.border
}

func (tabbedPanelBehavior) clientInset() int {
	// The strip occupies the top edge. A uniform one-cell inset keeps the
	// public Container contract simple and leaves a bounded page rectangle
	// even when the optional border is disabled.
	return 1
}

func (tabbedPanelBehavior) intrinsicMinimum() Size {
	return Size{Width: 3, Height: 3}
}

func (tabbedPanelBehavior) additionalStyles() []StyleID {
	return []StyleID{
		"tab.normal",
		"tab.mnemonic",
		"tab.selected",
		"tab.selected_mnemonic",
		"tab.focused",
		"tab.focused_mnemonic",
		"tab.disabled",
		"tab.continuation",
	}
}

func (b tabbedPanelBehavior) details() ControlDetails {
	border := b.border.details()
	border.Container = &ContainerDetails{ClientInset: b.clientInset()}
	border.TabbedPanel = &TabbedPanelDetails{Tabs: []TabDetails{}}
	return border
}

func (b tabbedPanelBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
) {
	b.border.paintDecoration(app, frame, control, absolute, clip)
	plan := buildTabRenderPlan(absolute.Width, b)
	if plan.leadingOmitted && absolute.Width > 0 &&
		!tabPlanOccupiesColumn(plan, 0) {
		app.setClippedCellLocked(
			frame,
			clip,
			absolute.X,
			absolute.Y,
			"◄",
			"tab.continuation",
			app.styles["tab.continuation"],
			control.id,
		)
	}
	if plan.trailingOmitted && absolute.Width > 0 &&
		!tabPlanOccupiesColumn(plan, absolute.Width-1) {
		app.setClippedCellLocked(
			frame,
			clip,
			absolute.X+absolute.Width-1,
			absolute.Y,
			"►",
			"tab.continuation",
			app.styles["tab.continuation"],
			control.id,
		)
	}
	for index, entry := range plan.entries {
		if entry.omitted || entry.bounds.Empty() {
			continue
		}
		tab := b.tabs[index]
		style, mnemonicStyle := tabStyles(
			control,
			app.focus == control,
			tab.tab.Disabled,
			tab.tab.Value == b.selected,
			tab.tab.Value == b.current,
		)
		x := absolute.X + entry.bounds.X
		app.paintMenuCellsLocked(
			frame,
			clip,
			x,
			absolute.Y,
			entry.cells,
			style,
			control.id,
		)
		mnemonicOffset := tabMnemonicCellOffset(tab)
		if mnemonicOffset >= 0 && mnemonicOffset < len(entry.cells) {
			app.setClippedCellLocked(
				frame,
				clip,
				x+mnemonicOffset,
				absolute.Y,
				entry.cells[mnemonicOffset],
				mnemonicStyle,
				app.styles[mnemonicStyle],
				control.id,
			)
		}
	}
}

func tabPlanOccupiesColumn(plan tabRenderPlan, column int) bool {
	for _, entry := range plan.entries {
		if !entry.omitted &&
			column >= entry.bounds.X &&
			column < entry.bounds.X+entry.bounds.Width {
			return true
		}
	}
	return false
}

func tabStyles(
	_ *controlState,
	focused bool,
	disabled bool,
	selected bool,
	current bool,
) (StyleID, StyleID) {
	switch {
	case disabled:
		return "tab.disabled", "tab.disabled"
	case focused && current:
		return "tab.focused", "tab.focused_mnemonic"
	case selected:
		return "tab.selected", "tab.selected_mnemonic"
	default:
		return "tab.normal", "tab.mnemonic"
	}
}

func tabCells(tab tabEntry, selected bool, focused bool) []string {
	left, right := "[", "]"
	switch {
	case tab.tab.Disabled:
		left, right = "(", ")"
	case selected:
		left, right = "└", "┘"
	case focused:
		left, right = "<", ">"
	}
	cells := make([]string, 0, len(tab.label.lines[0])+2)
	cells = append(cells, left)
	cells = append(cells, tab.label.lines[0]...)
	cells = append(cells, right)
	return cells
}

func tabMnemonicCellOffset(tab tabEntry) int {
	if tab.tab.Mnemonic == "" {
		return -1
	}
	for index, cell := range tab.label.lines[0] {
		if Key(strings.ToLower(cell)) == tab.tab.Mnemonic {
			return index + 1
		}
	}
	return -1
}

func buildTabRenderPlan(
	width int,
	behavior tabbedPanelBehavior,
) tabRenderPlan {
	plan := tabRenderPlan{
		entries: make([]tabRenderEntry, len(behavior.tabs)),
	}
	if width <= 0 || len(behavior.tabs) == 0 {
		for index := range plan.entries {
			plan.entries[index].omitted = true
		}
		return plan
	}
	current := tabIndexByValue(behavior.tabs, behavior.current)
	if current < 0 {
		current = 0
	}
	total := 0
	for index, tab := range behavior.tabs {
		total += len(tabCells(
			tab,
			tab.tab.Value == behavior.selected,
			tab.tab.Value == behavior.current,
		))
		plan.entries[index].omitted = true
	}
	start := 0
	if total > width {
		start = current
	}
	plan.leadingOmitted = start > 0
	cursor := 0
	if plan.leadingOmitted && width >= 2 {
		cursor = 1
	}
	for index := start; index < len(behavior.tabs); index++ {
		cells := tabCells(
			behavior.tabs[index],
			behavior.tabs[index].tab.Value == behavior.selected,
			behavior.tabs[index].tab.Value == behavior.current,
		)
		remaining := width - cursor
		if remaining <= 0 {
			plan.trailingOmitted = true
			break
		}
		reserveTrailing := 0
		if index < len(behavior.tabs)-1 && remaining >= 2 {
			reserveTrailing = 1
		}
		count := min(len(cells), remaining-reserveTrailing)
		if count <= 0 {
			plan.trailingOmitted = true
			break
		}
		plan.entries[index] = tabRenderEntry{
			bounds:  Rect{X: cursor, Width: count, Height: 1},
			clipped: count < len(cells),
			cells:   append([]string(nil), cells[:count]...),
		}
		cursor += count
		if count < len(cells) {
			plan.trailingOmitted = true
			break
		}
		if index < len(behavior.tabs)-1 && cursor >= width {
			plan.trailingOmitted = true
			break
		}
	}
	for index := 0; index < start; index++ {
		plan.entries[index].omitted = true
	}
	lastRendered := -1
	for index := range plan.entries {
		if !plan.entries[index].omitted {
			lastRendered = index
		}
	}
	if lastRendered < len(behavior.tabs)-1 {
		plan.trailingOmitted = true
	}
	return plan
}

func tabIndexByValue(tabs []tabEntry, value string) int {
	for index := range tabs {
		if tabs[index].tab.Value == value {
			return index
		}
	}
	return -1
}

func tabIndexByMnemonic(tabs []tabEntry, key Key) int {
	for index := range tabs {
		if tabs[index].tab.Mnemonic == key {
			return index
		}
	}
	return -1
}

func firstEnabledTab(tabs []tabEntry) int {
	for index := range tabs {
		if !tabs[index].tab.Disabled {
			return index
		}
	}
	if len(tabs) != 0 {
		return 0
	}
	return -1
}

func tabbedPanelEnabled(behavior tabbedPanelBehavior) bool {
	for _, tab := range behavior.tabs {
		if !tab.tab.Disabled {
			return true
		}
	}
	return false
}

func normalizeTabs(
	transaction *Transaction,
	owner *controlState,
	tabs []Tab,
	selected string,
	current string,
) ([]tabEntry, string, string, error) {
	if len(tabs) > MaxSelectionOptions {
		return nil, "", "", fmt.Errorf(
			"%w: tab count exceeds %d",
			ErrControlCapacity,
			MaxSelectionOptions,
		)
	}
	normalized := make([]tabEntry, len(tabs))
	keys := make(map[string]bool, len(tabs))
	values := make(map[string]bool, len(tabs))
	pages := make(map[*controlState]bool, len(tabs))
	mnemonics := make(map[Key]bool, len(tabs))
	for index := range tabs {
		tab := tabs[index]
		if !validBoundedIdentifier(tab.Key) || keys[tab.Key] ||
			!validBoundedIdentifier(tab.Value) || values[tab.Value] {
			return nil, "", "", fmt.Errorf(
				"%w: invalid or duplicate Tab identity",
				ErrInvalidControl,
			)
		}
		page, err := transaction.control(tab.Page)
		if err != nil || page.kind != ControlPanel || page.parent != owner ||
			page.layout != nil || pages[page] {
			return nil, "", "", fmt.Errorf(
				"%w: Tab page is not a unique direct Panel child",
				ErrInvalidParent,
			)
		}
		label, mnemonic, reason, err := normalizeSelectionPresentation(
			tab.Label,
			tab.Mnemonic,
			tab.Disabled,
			tab.DisabledReason,
		)
		if err != nil {
			return nil, "", "", err
		}
		if mnemonic != "" {
			if mnemonics[mnemonic] ||
				!displayTextContainsMnemonic(label, mnemonic) {
				return nil, "", "", fmt.Errorf(
					"%w: invalid or duplicate Tab mnemonic %q",
					ErrInvalidControl,
					mnemonic,
				)
			}
			mnemonics[mnemonic] = true
		}
		keys[tab.Key] = true
		values[tab.Value] = true
		pages[page] = true
		tab.Label = label.text
		tab.Mnemonic = mnemonic
		tab.DisabledReason = reason
		normalized[index] = tabEntry{tab: tab, label: label, page: page}
	}
	if len(normalized) == 0 {
		if selected != "" {
			return nil, "", "", fmt.Errorf(
				"%w: selected Tab does not exist",
				ErrInvalidControl,
			)
		}
		return normalized, "", "", nil
	}
	selectedIndex := tabIndexByValue(normalized, selected)
	if selected == "" {
		selectedIndex = firstEnabledTab(normalized)
		selected = normalized[selectedIndex].tab.Value
	} else if selectedIndex < 0 {
		return nil, "", "", fmt.Errorf(
			"%w: selected Tab does not exist",
			ErrInvalidControl,
		)
	}
	currentIndex := tabIndexByValue(normalized, current)
	if currentIndex < 0 || normalized[currentIndex].tab.Disabled {
		if !normalized[selectedIndex].tab.Disabled {
			currentIndex = selectedIndex
		} else {
			currentIndex = firstEnabledTab(normalized)
		}
	}
	current = normalized[currentIndex].tab.Value
	return normalized, selected, current, nil
}

func displayTextContainsMnemonic(
	label normalizedDisplayText,
	mnemonic Key,
) bool {
	for _, cell := range label.lines[0] {
		if Key(strings.ToLower(cell)) == mnemonic {
			return true
		}
	}
	return false
}

// Tabs returns an independent copy of the ordered Tab descriptors.
func (p *TabbedPanel) Tabs() []Tab { return tabsForRead(p.controlState()) }

// Tabs returns an independent copy of the ordered Tab descriptors.
func (n *Notebook) Tabs() []Tab { return tabsForRead(n.controlState()) }

func tabsForRead(state *controlState) []Tab {
	if state == nil || state.app == nil {
		return nil
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(tabbedPanelBehavior)
	if !ok || state.aborted || state.destroyed {
		return nil
	}
	tabs := make([]Tab, len(behavior.tabs))
	for index := range behavior.tabs {
		tabs[index] = behavior.tabs[index].tab
		page, _ := behavior.tabs[index].page.control.(*Panel)
		tabs[index].Page = page
	}
	return tabs
}

// Selected returns the selected page value.
func (p *TabbedPanel) Selected() string {
	return selectedTabForRead(p.controlState())
}

// Selected returns the selected page value.
func (n *Notebook) Selected() string {
	return selectedTabForRead(n.controlState())
}

func selectedTabForRead(state *controlState) string {
	if state == nil || state.app == nil {
		return ""
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(tabbedPanelBehavior)
	if !ok || state.aborted || state.destroyed {
		return ""
	}
	return behavior.selected
}

// SetTabs atomically replaces the ordered Tab model.
func (p *TabbedPanel) SetTabs(tabs []Tab, selected string) error {
	return setTabs(p, tabs, selected)
}

// SetTabs atomically replaces the ordered Tab model.
func (n *Notebook) SetTabs(tabs []Tab, selected string) error {
	return setTabs(n, tabs, selected)
}

func setTabs(control Control, tabs []Tab, selected string) error {
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := transaction.SetTabs(control, tabs, selected); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetSelected atomically changes the selected page.
func (p *TabbedPanel) SetSelected(value string) error {
	return setSelectedTab(p, value)
}

// SetSelected atomically changes the selected page.
func (n *Notebook) SetSelected(value string) error {
	return setSelectedTab(n, value)
}

func setSelectedTab(control Control, value string) error {
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := transaction.SetSelectedTab(control, value); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus gives this eligible TabbedPanel keyboard focus.
func (p *TabbedPanel) Focus() error { return focusSelectionControl(p) }

// Focus gives this eligible Notebook keyboard focus.
func (n *Notebook) Focus() error { return focusSelectionControl(n) }

// SetTabs records one complete ordered Tab replacement.
func (t *Transaction) SetTabs(
	control Control,
	tabs []Tab,
	selected string,
) error {
	target, err := t.control(control)
	if err != nil ||
		(target.kind != ControlTabbedPanel &&
			target.kind != ControlNotebook) {
		return ErrInvalidControl
	}
	behavior, ok := t.selectedControlBehavior(target).(tabbedPanelBehavior)
	if !ok {
		return ErrInvalidControl
	}
	normalized, selected, current, err := normalizeTabs(
		t,
		target,
		tabs,
		selected,
		behavior.current,
	)
	if err != nil {
		return err
	}
	behavior.tabs = normalized
	behavior.selected = selected
	behavior.current = current
	return t.recordTabbedPanelBehavior(target, behavior)
}

// SetSelectedTab records one selected-page change.
func (t *Transaction) SetSelectedTab(
	control Control,
	value string,
) error {
	target, err := t.control(control)
	if err != nil ||
		(target.kind != ControlTabbedPanel &&
			target.kind != ControlNotebook) {
		return ErrInvalidControl
	}
	behavior, ok := t.selectedControlBehavior(target).(tabbedPanelBehavior)
	if !ok || tabIndexByValue(behavior.tabs, value) < 0 {
		return fmt.Errorf(
			"%w: selected Tab does not exist",
			ErrInvalidControl,
		)
	}
	behavior.selected = value
	if index := tabIndexByValue(behavior.tabs, value); index >= 0 && !behavior.tabs[index].tab.Disabled {
		behavior.current = value
	}
	return t.recordTabbedPanelBehavior(target, behavior)
}

func (t *Transaction) recordTabbedPanelBehavior(
	state *controlState,
	behavior tabbedPanelBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationTabbedPanel, state: state, behavior: behavior,
	})
	return nil
}

func tabbedPanelBehaviorEqual(
	left tabbedPanelBehavior,
	right tabbedPanelBehavior,
) bool {
	if !borderBehaviorEqual(left.border, right.border) ||
		left.selected != right.selected ||
		left.current != right.current ||
		left.changeCommand != right.changeCommand ||
		len(left.tabs) != len(right.tabs) {
		return false
	}
	for index := range left.tabs {
		if left.tabs[index].tab != right.tabs[index].tab ||
			left.tabs[index].label.text != right.tabs[index].label.text ||
			left.tabs[index].page != right.tabs[index].page {
			return false
		}
	}
	return true
}

func borderBehaviorEqual(left borderBehavior, right borderBehavior) bool {
	return left.title == right.title &&
		left.borderStyle == right.borderStyle &&
		left.form == right.form &&
		colorsEqual(left.foregroundOverride, right.foregroundOverride) &&
		colorsEqual(left.backgroundOverride, right.backgroundOverride)
}

func colorsEqual(left *Color, right *Color) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func repairTabbedPanelBehavior(
	behavior tabbedPanelBehavior,
	destroyed map[*controlState]bool,
) (tabbedPanelBehavior, bool) {
	kept := make([]tabEntry, 0, len(behavior.tabs))
	for _, tab := range behavior.tabs {
		if destroyed[tab.page] || tab.page == nil || tab.page.destroyed {
			continue
		}
		kept = append(kept, tab)
	}
	if len(kept) == len(behavior.tabs) {
		return behavior, false
	}
	behavior.tabs = kept
	if len(kept) == 0 {
		behavior.selected = ""
		behavior.current = ""
		return behavior, true
	}
	if tabIndexByValue(kept, behavior.selected) < 0 {
		index := firstEnabledTab(kept)
		behavior.selected = kept[index].tab.Value
	}
	current := tabIndexByValue(kept, behavior.current)
	if current < 0 || kept[current].tab.Disabled {
		index := tabIndexByValue(kept, behavior.selected)
		if index < 0 || kept[index].tab.Disabled {
			index = firstEnabledTab(kept)
		}
		behavior.current = kept[index].tab.Value
	}
	return behavior, true
}

func (a *App) repairTabbedPanelsLocked(
	destroyed map[*controlState]bool,
) bool {
	changed := false
	for _, state := range a.controlsByID {
		behavior, ok := state.behavior.(tabbedPanelBehavior)
		if !ok || state.destroyed {
			continue
		}
		repaired, repairedState := repairTabbedPanelBehavior(
			behavior,
			destroyed,
		)
		if repairedState {
			state.behavior = repaired
			changed = true
		}
	}
	return changed
}

func tabbedPanelDetails(
	bounds Rect,
	behavior tabbedPanelBehavior,
) TabbedPanelDetails {
	plan := buildTabRenderPlan(bounds.Width, behavior)
	details := TabbedPanelDetails{
		Tabs:            make([]TabDetails, len(behavior.tabs)),
		Selected:        behavior.selected,
		Current:         behavior.current,
		ChangeCommand:   behavior.changeCommand,
		LeadingOmitted:  plan.leadingOmitted,
		TrailingOmitted: plan.trailingOmitted,
	}
	for index := range behavior.tabs {
		tab := behavior.tabs[index]
		details.Tabs[index] = TabDetails{
			Key:            tab.tab.Key,
			Value:          tab.tab.Value,
			Label:          tab.tab.Label,
			Mnemonic:       tab.tab.Mnemonic,
			Page:           tab.page.id,
			PageKey:        tab.page.automationKey,
			Enabled:        !tab.tab.Disabled,
			DisabledReason: tab.tab.DisabledReason,
			Selected:       tab.tab.Value == behavior.selected,
			Current:        tab.tab.Value == behavior.current,
			Bounds:         plan.entries[index].bounds,
			Omitted:        plan.entries[index].omitted,
			Clipped:        plan.entries[index].clipped,
		}
	}
	return details
}

func (a *App) tabbedPanelKeyLocked(
	control *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if control == nil {
		return "", "", false, false
	}
	behavior, ok := control.behavior.(tabbedPanelBehavior)
	if !ok {
		return "", "", false, false
	}
	current := tabIndexByValue(behavior.tabs, behavior.current)
	handled := true
	switch key {
	case KeyLeft:
		current = adjacentEnabledTab(behavior.tabs, current, -1)
	case KeyRight:
		current = adjacentEnabledTab(behavior.tabs, current, 1)
	case KeyHome:
		current = firstEnabledTab(behavior.tabs)
		if current >= 0 && behavior.tabs[current].tab.Disabled {
			current = -1
		}
	case KeyEnd:
		current = lastEnabledTab(behavior.tabs)
	case KeySpace, KeyEnter:
		if current < 0 || behavior.tabs[current].tab.Disabled ||
			behavior.tabs[current].tab.Value == behavior.selected {
			return "", "", true, false
		}
		behavior.selected = behavior.tabs[current].tab.Value
		control.behavior = behavior
		return behavior.changeCommand, control.id, true, true
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	if current < 0 || behavior.tabs[current].tab.Value == behavior.current {
		return "", "", true, false
	}
	behavior.current = behavior.tabs[current].tab.Value
	control.behavior = behavior
	return "", control.id, true, true
}

func adjacentEnabledTab(tabs []tabEntry, current int, direction int) int {
	for index := current + direction; index >= 0 && index < len(tabs); index += direction {
		if !tabs[index].tab.Disabled {
			return index
		}
	}
	return current
}

func lastEnabledTab(tabs []tabEntry) int {
	for index := len(tabs) - 1; index >= 0; index-- {
		if !tabs[index].tab.Disabled {
			return index
		}
	}
	return -1
}

func (a *App) tabMnemonicLocked(
	key Key,
) (CommandID, ControlID, bool, bool) {
	var candidates []*controlState
	for _, state := range a.controlsByID {
		behavior, ok := state.behavior.(tabbedPanelBehavior)
		if !ok || !a.controlReceivesInputLocked(state) ||
			!a.inActiveInputScopeLocked(state) {
			continue
		}
		index := tabIndexByMnemonic(behavior.tabs, key)
		if index < 0 {
			continue
		}
		if focusWithinControl(a.focus, state) {
			return a.activateTabMnemonicLocked(state, index)
		}
		candidates = append(candidates, state)
	}
	if len(candidates) != 1 {
		return "", "", false, false
	}
	behavior := candidates[0].behavior.(tabbedPanelBehavior)
	return a.activateTabMnemonicLocked(
		candidates[0],
		tabIndexByMnemonic(behavior.tabs, key),
	)
}

func focusWithinControl(focus *controlState, owner *controlState) bool {
	for current := focus; current != nil; current = current.parent {
		if current == owner {
			return true
		}
	}
	return false
}

func (a *App) activateTabMnemonicLocked(
	control *controlState,
	index int,
) (CommandID, ControlID, bool, bool) {
	behavior := control.behavior.(tabbedPanelBehavior)
	if index < 0 || index >= len(behavior.tabs) ||
		behavior.tabs[index].tab.Disabled {
		return "", "", true, false
	}
	changed := false
	selectedChanged := behavior.selected != behavior.tabs[index].tab.Value
	if selectedChanged {
		behavior.selected = behavior.tabs[index].tab.Value
		changed = true
	}
	if behavior.current != behavior.tabs[index].tab.Value {
		behavior.current = behavior.tabs[index].tab.Value
		changed = true
	}
	if a.focus != control {
		_ = a.commitOrCancelEditorStateLocked(a.focus)
		a.focus = control
		a.clearInvalidPressesLocked()
		changed = true
	}
	control.behavior = behavior
	command := CommandID("")
	if selectedChanged {
		command = behavior.changeCommand
	}
	return command, control.id, true, changed
}

func (a *App) selectedTabPageLocked(owner *controlState) *controlState {
	behavior, ok := owner.behavior.(tabbedPanelBehavior)
	if !ok {
		return nil
	}
	index := tabIndexByValue(behavior.tabs, behavior.selected)
	if index < 0 {
		return nil
	}
	return behavior.tabs[index].page
}

func (a *App) isManagedTabPageLocked(state *controlState) (bool, bool) {
	if state == nil || state.parent == nil {
		return false, false
	}
	behavior, ok := state.parent.behavior.(tabbedPanelBehavior)
	if !ok {
		return false, false
	}
	for _, tab := range behavior.tabs {
		if tab.page == state {
			return true, tab.tab.Value == behavior.selected
		}
	}
	return false, false
}
