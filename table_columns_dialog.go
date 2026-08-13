package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	CommandTableColumnsOpen      CommandID = "table.columns.open"
	CommandTableColumnsSearch    CommandID = "table.columns.search"
	CommandTableColumnsCurrent   CommandID = "table.columns.current"
	CommandTableColumnsVisible   CommandID = "table.columns.visible"
	CommandTableColumnsWrap      CommandID = "table.columns.wrap"
	CommandTableColumnsMoveLeft  CommandID = "table.columns.move_left"
	CommandTableColumnsMoveRight CommandID = "table.columns.move_right"
	CommandTableColumnsReset     CommandID = "table.columns.reset"
	CommandTableColumnsReload    CommandID = "table.columns.reload"
	CommandTableColumnsOK        CommandID = "table.columns.ok"
	CommandTableColumnsCancel    CommandID = "table.columns.cancel"
)

var errTableColumnsStale = errors.New("expletives: column presentation changed externally")

func tableColumnsCommandError(code, message string, err error) CommandResult {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return contextCommandResult(err)
	}
	return publicResult(OutcomeFailed, code, message, err)
}

type tableColumnsDialogCore struct {
	mu sync.Mutex

	owner   *controlState
	dialog  *Dialog
	draft   []TableColumnPresentation
	initial []TableColumnPresentation
	columns []Column

	schemaRevision       uint64
	presentationRevision atomic.Uint64
	stale                atomic.Bool
	selectedIndex        atomic.Int64
	draftLength          atomic.Int64

	search      *TextField
	list        *ListBox
	visible     *Checkbox
	wrap        *RadioGroup
	moveLeft    *Button
	moveRight   *Button
	information *StaticText
	reset       *Button
	reload      *Button
	ok          *Button
	cancel      *Button
}

type tableColumnsLayoutItem struct {
	control Control
	options LayoutItemOptions
}

func addTableColumnsLayoutItems(
	layout *BoxLayout,
	items ...tableColumnsLayoutItem,
) error {
	for _, item := range items {
		if err := layout.AddPanel(item.control, item.options); err != nil {
			return err
		}
	}
	return nil
}

func isTableColumnsCommand(command CommandID) bool {
	switch command {
	case CommandTableColumnsOpen, CommandTableColumnsSearch,
		CommandTableColumnsCurrent, CommandTableColumnsVisible,
		CommandTableColumnsWrap, CommandTableColumnsMoveLeft,
		CommandTableColumnsMoveRight, CommandTableColumnsReset,
		CommandTableColumnsReload, CommandTableColumnsOK,
		CommandTableColumnsCancel:
		return true
	default:
		return false
	}
}

func (a *App) routeTableColumnsCommand(
	ctx context.Context,
	command Command,
) CommandResult {
	if command.ID == CommandTableColumnsOpen {
		return a.openTableColumnsDialog(ctx, command)
	}
	a.mu.RLock()
	state := a.controlsByID[command.Target]
	modal := modalAncestorState(state)
	var core *tableColumnsDialogCore
	for _, candidate := range a.tableColumnsDialogs {
		if candidate != nil && candidate.dialog != nil &&
			candidate.dialog.state == modal {
			core = candidate
			break
		}
	}
	a.mu.RUnlock()
	if core == nil {
		return publicResult(
			OutcomeRejected,
			"columns_dialog_target",
			"command target is not an active Columns dialog",
			ErrInvalidControl,
		)
	}
	return core.handle(ctx, command)
}

func (a *App) openTableColumnsDialog(
	ctx context.Context,
	command Command,
) CommandResult {
	a.mu.Lock()
	owner := a.controlsByID[command.Target]
	behavior, ok := tableBehaviorFromControl(owner)
	if !ok || !a.tableColumnsActionEligibleLocked(owner) ||
		behavior.focusPart != TableFocusColumnsAction {
		a.mu.Unlock()
		return publicResult(
			OutcomeRejected,
			"columns_action",
			"Columns action is unavailable",
			ErrNotFocusable,
		)
	}
	if current := a.tableColumnsDialogs[owner]; current != nil {
		a.mu.Unlock()
		return CommandResult{Outcome: OutcomeNoOp}
	}
	changeCommand := CommandID("")
	if grid, gridOK := owner.behavior.(dataGridBehavior); gridOK && grid.editor.editing {
		changeCommand, _, ok = a.commitDataGridEditLocked(owner, &grid)
		if !ok {
			grid.table.focusPart = TableFocusBody
			owner.behavior = grid
			a.mu.Unlock()
			return publicResult(
				OutcomeRejected,
				"validation_failed",
				"Current cell must be valid before editing columns",
				nil,
			)
		}
		owner.behavior = grid
		a.publishLocked(nil)
		behavior = grid.table
	}
	columns := copyTableColumns(behavior.columns)
	draft := append([]TableColumnPresentation(nil), behavior.columnPresentation...)
	initial := append([]TableColumnPresentation(nil), behavior.initialPresentation...)
	schemaRevision := behavior.schemaRevision
	presentationRevision := behavior.presentationRevision
	a.mu.Unlock()

	if changeCommand != "" {
		a.mu.RLock()
		router, result, execute := a.resolveCommandLocked(changeCommand)
		a.mu.RUnlock()
		if execute {
			result = a.callRouter(ctx, router, Command{
				ID: changeCommand, Target: owner.id, Source: command.Source,
			})
		}
		switch result.Outcome {
		case OutcomeRejected, OutcomeFailed, OutcomeInterrupted, OutcomeExited:
			return result
		}
	}

	core := &tableColumnsDialogCore{
		owner:          owner,
		draft:          draft,
		initial:        initial,
		columns:        columns,
		schemaRevision: schemaRevision,
	}
	core.presentationRevision.Store(presentationRevision)
	core.selectedIndex.Store(-1)
	core.draftLength.Store(int64(len(draft)))
	if err := core.build(ctx); err != nil {
		return tableColumnsCommandError(
			"columns_dialog_build",
			"Columns dialog could not be built",
			err,
		)
	}
	a.mu.Lock()
	current, currentOK := tableBehaviorFromControl(owner)
	if a.final || owner.destroyed || owner.aborted ||
		a.tableColumnsDialogs[owner] != nil || !currentOK ||
		current.focusPart != TableFocusColumnsAction ||
		!a.tableColumnsActionEligibleLocked(owner) {
		a.mu.Unlock()
		_ = core.dialog.Destroy()
		return publicResult(
			OutcomeRejected,
			"columns_dialog_state",
			"Columns dialog could not open",
			ErrModalState,
		)
	}
	if current.schemaRevision != core.schemaRevision {
		core.draft = repairTableColumnPresentation(core.draft, current.columns)
		core.initial = repairTableColumnPresentation(core.initial, current.columns)
		core.columns = copyTableColumns(current.columns)
		core.schemaRevision = current.schemaRevision
		core.draftLength.Store(int64(len(core.draft)))
	}
	if current.presentationRevision != core.presentationRevision.Load() {
		core.stale.Store(true)
		core.reload.state.visible = true
		if information, valid := core.information.state.behavior.(textBehavior); valid {
			next, updateErr := information.withText(
				"Column presentation changed externally. Reload or Cancel.",
			)
			if updateErr == nil {
				core.information.state.behavior = next
			}
		}
	}
	a.tableColumnsDialogs[owner] = core
	a.mu.Unlock()
	if err := core.dialog.Show(core.list); err != nil {
		a.mu.Lock()
		delete(a.tableColumnsDialogs, owner)
		a.mu.Unlock()
		_ = core.dialog.Destroy()
		return tableColumnsCommandError(
			"columns_dialog_show",
			"Columns dialog could not open",
			err,
		)
	}
	return CommandResult{Outcome: OutcomeApplied}
}

func (a *App) tableColumnsDialogOpenLocked(owner *controlState) bool {
	return a != nil && owner != nil && a.tableColumnsDialogs[owner] != nil
}

func (a *App) tableColumnsDialogKeyLocked(
	state *controlState,
	key Key,
	held map[Key]bool,
) (CommandID, ControlID, bool) {
	if state == nil {
		return "", "", false
	}
	modal := modalAncestorState(state)
	for _, core := range a.tableColumnsDialogs {
		if core == nil || core.dialog == nil || core.dialog.state != modal {
			continue
		}
		if noHeldModifiers(held) && key == KeySpace && core.list != nil &&
			core.list.state == state {
			return CommandTableColumnsVisible, state.id, true
		}
		if !held[KeyControl] || held[KeyAlt] || held[KeyMeta] || held[KeyShift] ||
			(key != KeyLeft && key != KeyRight) {
			return "", "", false
		}
		command := CommandTableColumnsMoveLeft
		if key == KeyRight {
			command = CommandTableColumnsMoveRight
		}
		return command, state.id, true
	}
	return "", "", false
}

func (a *App) reconcileTableColumnsDialogsLocked() bool {
	if a == nil || len(a.tableColumnsDialogs) == 0 {
		return false
	}
	changed := false
	const staleMessage = "Column presentation changed externally. Reload or Cancel."
	for owner, core := range a.tableColumnsDialogs {
		if core == nil || core.dialog == nil || core.reload == nil ||
			core.information == nil {
			continue
		}
		behavior, ok := tableBehaviorFromControl(owner)
		featureEnabled := ok && tableHasFeature(behavior, TableFeatureColumns)
		if !ok || owner.destroyed || owner.aborted || !owner.visible ||
			!a.effectivelyVisibleLocked(owner) || !featureEnabled {
			modal := core.dialog.state
			for index, candidate := range a.modals {
				if candidate == modal {
					a.closeModalRangeLocked(index, ModalResult{
						Reason: ModalCancelled,
						Action: CommandTableColumnsCancel,
					})
					changed = true
					break
				}
			}
			removeChildState(core.dialog.state.parent, core.dialog.state)
			a.destroyStateLocked(core.dialog.state)
			delete(a.tableColumnsDialogs, owner)
			continue
		}
		if !ok || behavior.presentationRevision == core.presentationRevision.Load() {
			continue
		}
		core.stale.Store(true)
		if state := core.reload.state; state != nil && !state.visible {
			state.visible = true
			changed = true
		}
		if state := core.information.state; state != nil {
			if current, valid := state.behavior.(textBehavior); valid &&
				current.content.text != staleMessage {
				next, err := current.withText(staleMessage)
				if err == nil {
					state.behavior = next
					if state.autoMinimum {
						state.minimumSize = next.(intrinsicMinimumBehavior).intrinsicMinimum()
					}
					changed = true
				}
			}
		}
	}
	return changed
}

func (c *tableColumnsDialogCore) build(ctx context.Context) error {
	if c == nil || c.owner == nil || c.owner.app == nil {
		return ErrInvalidControl
	}
	app := c.owner.app
	tx := app.NewTransaction()
	key := derivedCompoundKey(c.owner.automationKey, "columns-dialog")
	dialog, err := tx.NewDialog(app.Root(), DialogOptions{ModalPanelOptions: ModalPanelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: key,
			Bounds:        Rect{Width: 58, Height: 23},
		},
		Title: "Columns",
	}})
	if err != nil {
		return err
	}
	setProvisionalModalEscapeCommand(dialog.state, CommandTableColumnsCancel)
	style := StyleID(ControlDialog)

	searchPanel, err := tx.NewPanel(dialog, PanelOptions{
		AutomationKey: derivedCompoundKey(key, "search-group"),
		Style:         style,
		MinimumSize:   Size{Height: 2},
	})
	if err != nil {
		return err
	}
	search, err := tx.NewTextField(searchPanel, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "search"),
		},
		EditCommand: CommandTableColumnsSearch,
	})
	if err != nil {
		return err
	}
	searchLabel, err := tx.NewLabel(searchPanel, LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "search-label"), Style: style,
		},
		Text: "Search", Target: search, Mnemonic: "s",
	})
	if err != nil {
		return err
	}

	mainPanel, err := tx.NewPanel(dialog, PanelOptions{
		AutomationKey: derivedCompoundKey(key, "main"), Style: style,
		MinimumSize: Size{Height: 10},
	})
	if err != nil {
		return err
	}
	inventoryPanel, err := tx.NewPanel(mainPanel, PanelOptions{
		AutomationKey: derivedCompoundKey(key, "inventory-group"),
		Style:         style,
		MinimumSize:   Size{Width: 28, Height: 8},
	})
	if err != nil {
		return err
	}
	items, current := c.items("")
	list, err := tx.NewListBox(inventoryPanel, ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: derivedCompoundKey(key, "inventory"),
			}},
			BorderForm: BorderSingle,
		},
		Items: items, Current: current,
		SelectionMarks: CollectionSelectionMarksHide,
		CurrentCommand: CommandTableColumnsCurrent,
	})
	if err != nil {
		return err
	}
	listLabel, err := tx.NewLabel(inventoryPanel, LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "inventory-label"), Style: style,
		},
		Text: "Columns", Target: list, Mnemonic: "o",
	})
	if err != nil {
		return err
	}

	optionsPanel, err := tx.NewPanel(mainPanel, PanelOptions{
		AutomationKey: derivedCompoundKey(key, "options"), Style: style,
		MinimumSize: Size{Width: 16, Height: 11},
	})
	if err != nil {
		return err
	}
	visible, err := tx.NewCheckbox(optionsPanel, CheckboxOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "visible")},
		Label:        "Visible", Mnemonic: "v", State: CheckChecked,
		ChangeCommand: CommandTableColumnsVisible,
	})
	if err != nil {
		return err
	}
	wrap, err := tx.NewRadioGroup(optionsPanel, RadioGroupOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "wrap"), Style: style,
			MinimumSize: Size{Height: 3},
		},
		ChangeCommand: CommandTableColumnsWrap,
	})
	if err != nil {
		return err
	}
	wrapButtons := make([]*RadioButton, 0, 3)
	for _, option := range []struct {
		value, label string
		mnemonic     Key
	}{
		{string(TableColumnClip), "Clip", "p"},
		{string(TableColumnWrapWords), "Wrap", "w"},
		{string(TableColumnHang), "Hang", "h"},
	} {
		button, buttonErr := tx.NewRadioButton(wrap, RadioButtonOptions{
			PanelOptions: PanelOptions{
				AutomationKey: derivedCompoundKey(key, "wrap-"+option.value),
			},
			Value: option.value, Label: option.label, Mnemonic: option.mnemonic,
			Selected: option.value == string(TableColumnClip),
		})
		if buttonErr != nil {
			return buttonErr
		}
		wrapButtons = append(wrapButtons, button)
	}
	moveLeft, err := tx.NewButton(optionsPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "move-left")},
		Command:      CommandTableColumnsMoveLeft, Mnemonic: "l",
	})
	if err != nil {
		return err
	}
	moveRight, err := tx.NewButton(optionsPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "move-right")},
		Command:      CommandTableColumnsMoveRight, Mnemonic: "r",
	})
	if err != nil {
		return err
	}
	information, err := tx.NewStaticText(dialog, StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "information"), Style: style,
			MinimumSize: Size{Height: 1},
		},
		Text: "Choose a column, then change visibility, wrapping, or order.",
		Wrap: TextWrapWords,
	})
	if err != nil {
		return err
	}
	buttonPanel, err := tx.NewPanel(dialog, PanelOptions{
		AutomationKey: derivedCompoundKey(key, "buttons"), Style: style,
		MinimumSize: Size{Height: 2},
	})
	if err != nil {
		return err
	}
	reset, err := tx.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "reset")},
		Command:      CommandTableColumnsReset, Mnemonic: "e",
	})
	if err != nil {
		return err
	}
	reload, err := tx.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(key, "reload"),
			Hidden:        true,
		},
		Command: CommandTableColumnsReload, Mnemonic: "d",
	})
	if err != nil {
		return err
	}
	ok, err := tx.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "ok")},
		Command:      CommandTableColumnsOK, Mnemonic: "k", Default: true,
	})
	if err != nil {
		return err
	}
	cancel, err := tx.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(key, "cancel")},
		Command:      CommandTableColumnsCancel, Mnemonic: "c", Cancel: true,
	})
	if err != nil {
		return err
	}

	searchLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "search-layout"),
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		searchLayout,
		tableColumnsLayoutItem{control: searchLabel},
		tableColumnsLayoutItem{control: search},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(searchPanel, searchLayout); err != nil {
		return err
	}
	inventoryLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "inventory-layout"),
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		inventoryLayout,
		tableColumnsLayoutItem{control: listLabel},
		tableColumnsLayoutItem{control: list, options: LayoutItemOptions{Grow: 1}},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(inventoryPanel, inventoryLayout); err != nil {
		return err
	}
	wrapLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "wrap-layout"),
	})
	if err != nil {
		return err
	}
	for _, button := range wrapButtons {
		if err := wrapLayout.AddPanel(button, LayoutItemOptions{}); err != nil {
			return err
		}
	}
	if err := tx.SetLayout(wrap, wrapLayout); err != nil {
		return err
	}
	optionsLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "options-layout"), Gap: 1,
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		optionsLayout,
		tableColumnsLayoutItem{control: visible},
		tableColumnsLayoutItem{control: wrap},
		tableColumnsLayoutItem{control: moveLeft},
		tableColumnsLayoutItem{control: moveRight},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(optionsPanel, optionsLayout); err != nil {
		return err
	}
	mainLayout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "main-layout"), Gap: 1,
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		mainLayout,
		tableColumnsLayoutItem{
			control: inventoryPanel,
			options: LayoutItemOptions{Grow: 1},
		},
		tableColumnsLayoutItem{control: optionsPanel},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(mainPanel, mainLayout); err != nil {
		return err
	}
	buttonLayout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "button-layout"), Gap: 2,
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		buttonLayout,
		tableColumnsLayoutItem{control: reset},
		tableColumnsLayoutItem{control: reload},
		tableColumnsLayoutItem{control: ok},
		tableColumnsLayoutItem{control: cancel},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(buttonPanel, buttonLayout); err != nil {
		return err
	}
	rootLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(key, "layout"),
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1}, Gap: 1,
	})
	if err != nil {
		return err
	}
	if err := addTableColumnsLayoutItems(
		rootLayout,
		tableColumnsLayoutItem{control: searchPanel},
		tableColumnsLayoutItem{control: mainPanel, options: LayoutItemOptions{Grow: 1}},
		tableColumnsLayoutItem{control: information},
		tableColumnsLayoutItem{control: buttonPanel},
	); err != nil {
		return err
	}
	if err := tx.SetLayout(dialog, rootLayout); err != nil {
		return err
	}

	c.dialog, c.search, c.list, c.visible, c.wrap = dialog, search, list, visible, wrap
	c.moveLeft, c.moveRight, c.information = moveLeft, moveRight, information
	c.reset, c.reload, c.ok, c.cancel = reset, reload, ok, cancel
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return c.sync(ctx, "")
}

func (c *tableColumnsDialogCore) columnLabel(key string) string {
	for _, column := range c.columns {
		if column.Key == key {
			if column.Header != "" {
				return column.Header
			}
			return key
		}
	}
	return key
}

func (c *tableColumnsDialogCore) items(search string) ([]ListItem, string) {
	needle := strings.ToLower(strings.TrimSpace(search))
	items := make([]ListItem, 0, len(c.draft))
	for _, entry := range c.draft {
		label := c.columnLabel(entry.Column)
		if needle != "" && !strings.Contains(strings.ToLower(label+" "+entry.Column), needle) {
			continue
		}
		items = append(items, ListItem{Key: entry.Column, Label: label})
	}
	current := ""
	if c.list != nil {
		current = c.list.State().Current
	}
	for _, item := range items {
		if item.Key == current {
			return items, current
		}
	}
	if len(items) > 0 {
		current = items[0].Key
	}
	return items, current
}

func (c *tableColumnsDialogCore) selected() (int, bool) {
	if c == nil || c.list == nil {
		return -1, false
	}
	key := c.list.State().Current
	for index, entry := range c.draft {
		if entry.Column == key {
			return index, true
		}
	}
	return -1, false
}

func (c *tableColumnsDialogCore) reconcile() error {
	if c == nil || c.owner == nil || c.owner.app == nil {
		return ErrInvalidControl
	}
	app := c.owner.app
	app.mu.RLock()
	behavior, ok := tableBehaviorFromControl(c.owner)
	if !ok || c.owner.destroyed || c.owner.aborted {
		app.mu.RUnlock()
		return ErrInvalidControl
	}
	if behavior.schemaRevision != c.schemaRevision {
		c.draft = repairTableColumnPresentation(c.draft, behavior.columns)
		c.initial = repairTableColumnPresentation(c.initial, behavior.columns)
		c.columns = copyTableColumns(behavior.columns)
		c.schemaRevision = behavior.schemaRevision
	}
	presentationRevision := behavior.presentationRevision
	app.mu.RUnlock()

	stale := presentationRevision != c.presentationRevision.Load()
	c.stale.Store(stale)
	return nil
}

func (c *tableColumnsDialogCore) reloadLatest() error {
	if c == nil || c.owner == nil || c.owner.app == nil {
		return ErrInvalidControl
	}
	app := c.owner.app
	app.mu.RLock()
	behavior, ok := tableBehaviorFromControl(c.owner)
	if !ok || c.owner.destroyed || c.owner.aborted {
		app.mu.RUnlock()
		return ErrInvalidControl
	}
	c.draft = append([]TableColumnPresentation(nil), behavior.columnPresentation...)
	c.initial = append([]TableColumnPresentation(nil), behavior.initialPresentation...)
	c.columns = copyTableColumns(behavior.columns)
	c.schemaRevision = behavior.schemaRevision
	c.presentationRevision.Store(behavior.presentationRevision)
	app.mu.RUnlock()
	c.stale.Store(false)
	return nil
}

func (c *tableColumnsDialogCore) sync(ctx context.Context, message string) error {
	items, current := c.items(c.search.Text())
	tx := c.owner.app.NewTransaction()
	if err := tx.ReplaceList(c.list, items, current, nil); err != nil {
		return err
	}
	index := -1
	for candidate, entry := range c.draft {
		if entry.Column == current {
			index = candidate
			break
		}
	}
	if index >= 0 {
		state := CheckUnchecked
		if c.draft[index].Visible {
			state = CheckChecked
		}
		if err := tx.SetCheckState(c.visible, state); err != nil {
			return err
		}
		if err := tx.SetRadioValue(c.wrap, string(c.draft[index].Wrap)); err != nil {
			return err
		}
	}
	c.selectedIndex.Store(int64(index))
	c.draftLength.Store(int64(len(c.draft)))
	if message == "" {
		if index >= 0 {
			message = fmt.Sprintf("%d of %d columns shown", visibleTableColumnCount(c.draft), len(c.draft))
		} else {
			message = "No columns match the search"
		}
	}
	if err := tx.SetText(c.information, message); err != nil {
		return err
	}
	if err := tx.SetVisible(c.reload, c.stale.Load()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (c *tableColumnsDialogCore) handle(
	ctx context.Context,
	command Command,
) CommandResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.reconcile(); err != nil {
		return tableColumnsCommandError("columns_dialog_state", "Columns dialog is unavailable", err)
	}
	message := ""
	if c.stale.Load() {
		message = "Column presentation changed externally. Reload or Cancel."
	}
	switch command.ID {
	case CommandTableColumnsSearch, CommandTableColumnsCurrent:
	case CommandTableColumnsVisible:
		index, ok := c.selected()
		if !ok {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		visible := c.visible.State() == CheckChecked
		if c.list != nil && command.Target == c.list.state.id {
			visible = !c.draft[index].Visible
		}
		if !visible && c.draft[index].Visible && visibleTableColumnCount(c.draft) == 1 {
			message = "At least one column must remain visible."
		} else {
			c.draft[index].Visible = visible
		}
	case CommandTableColumnsWrap:
		index, ok := c.selected()
		if !ok {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		c.draft[index].Wrap = TableColumnWrap(c.wrap.Value())
	case CommandTableColumnsMoveLeft, CommandTableColumnsMoveRight:
		index, ok := c.selected()
		if !ok {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		next := index - 1
		if command.ID == CommandTableColumnsMoveRight {
			next = index + 1
		}
		if next < 0 || next >= len(c.draft) {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		c.draft[index], c.draft[next] = c.draft[next], c.draft[index]
	case CommandTableColumnsReset:
		c.draft = append([]TableColumnPresentation(nil), c.initial...)
	case CommandTableColumnsReload:
		if err := c.reloadLatest(); err != nil {
			return tableColumnsCommandError(
				"columns_dialog_reload",
				"The latest column presentation could not be loaded",
				err,
			)
		}
		message = "Latest column presentation loaded."
	case CommandTableColumnsOK:
		if c.stale.Load() {
			return publicResult(
				OutcomeRejected,
				"columns_dialog_stale",
				"Column presentation changed externally; Reload or Cancel",
				nil,
			)
		}
		if err := c.apply(ctx, command.Source); err != nil {
			if errors.Is(err, errTableColumnsStale) {
				c.stale.Store(true)
				_ = c.sync(ctx, "Column presentation changed externally. Reload or Cancel.")
				return publicResult(
					OutcomeRejected,
					"columns_dialog_stale",
					"Column presentation changed externally; Reload or Cancel",
					nil,
				)
			}
			return tableColumnsCommandError(
				"columns_dialog_apply",
				"Column presentation could not be applied",
				err,
			)
		}
		return CommandResult{Outcome: OutcomeApplied}
	case CommandTableColumnsCancel:
		if err := c.close(ModalCancelled, command.ID); err != nil {
			return tableColumnsCommandError(
				"columns_dialog_close",
				"Columns dialog could not close",
				err,
			)
		}
		return CommandResult{Outcome: OutcomeApplied}
	default:
		return publicResult(
			OutcomeRejected, "columns_dialog_command",
			"unknown Columns dialog command", ErrInvalidRequest,
		)
	}
	if err := c.sync(ctx, message); err != nil {
		return tableColumnsCommandError(
			"columns_dialog_update",
			"Columns dialog could not update",
			err,
		)
	}
	return CommandResult{Outcome: OutcomeApplied}
}

func (c *tableColumnsDialogCore) apply(ctx context.Context, source string) error {
	app := c.owner.app
	if err := app.beginMutation(ctx); err != nil {
		return err
	}
	app.mu.Lock()
	if err := c.owner.mutableLocked(); err != nil {
		app.mu.Unlock()
		app.endMutation()
		return err
	}
	behavior, ok := tableBehaviorFromControl(c.owner)
	if !ok {
		app.mu.Unlock()
		app.endMutation()
		return ErrInvalidControl
	}
	if behavior.schemaRevision != c.schemaRevision {
		c.draft = repairTableColumnPresentation(c.draft, behavior.columns)
		c.initial = repairTableColumnPresentation(c.initial, behavior.columns)
		c.columns = copyTableColumns(behavior.columns)
		c.schemaRevision = behavior.schemaRevision
	}
	if behavior.presentationRevision != c.presentationRevision.Load() {
		c.stale.Store(true)
		app.mu.Unlock()
		app.endMutation()
		return errTableColumnsStale
	}
	normalized, err := normalizeTableColumnPresentation(behavior.columns, c.draft)
	if err != nil {
		app.mu.Unlock()
		app.endMutation()
		return err
	}
	changed := !tablePresentationEqual(behavior.columnPresentation, normalized)
	changeCommand := behavior.changeCommand
	if changed {
		previous := behavior.columnPresentation
		behavior.columnPresentation = normalized
		behavior.presentationRevision++
		c.presentationRevision.Store(behavior.presentationRevision)
		invalidateTableGeometry(&behavior)
		repairTableCurrentColumn(&behavior, previous)
		switch current := c.owner.behavior.(type) {
		case tableBehavior:
			behavior = reflowTable(behavior, c.owner.bounds.Size())
			c.owner.behavior = behavior
		case dataGridBehavior:
			cancelDataGridEditBehavior(&current)
			current.table = behavior
			current = reflowDataGrid(current, c.owner.bounds.Size())
			c.owner.behavior = current
		default:
			app.mu.Unlock()
			app.endMutation()
			return ErrInvalidControl
		}
		if c.owner.autoMinimum {
			c.owner.minimumSize = behavior.intrinsicMinimum()
		}
		arrangeAllLayoutsLocked(app)
		app.ensureFocusLocked()
		app.ensureFocusedControlVisibleLocked()
		app.publishLocked(nil)
	}
	app.mu.Unlock()
	app.endMutation()

	if err := c.close(ModalAccepted, CommandTableColumnsOK); err != nil {
		return err
	}
	app.mu.RLock()
	behavior, ok = tableBehaviorFromControl(c.owner)
	if !ok {
		app.mu.RUnlock()
		return ErrInvalidControl
	}
	var router CommandRouter
	var result CommandResult
	var execute bool
	if changeCommand != "" {
		router, result, execute = app.resolveCommandLocked(changeCommand)
	}
	app.mu.RUnlock()
	if execute {
		result = app.callRouter(ctx, router, Command{
			ID: changeCommand, Target: c.owner.id, Source: source,
		})
		if result.Outcome == OutcomeFailed || result.Outcome == OutcomeRejected {
			return fmt.Errorf("%w: Table ChangeCommand was not accepted", ErrInvalidRequest)
		}
	}
	return nil
}

func (c *tableColumnsDialogCore) close(reason ModalCloseReason, action CommandID) error {
	if c == nil || c.dialog == nil || c.owner == nil || c.owner.app == nil {
		return ErrInvalidControl
	}
	app := c.owner.app
	if err := c.dialog.Close(ModalResult{Reason: reason, Action: action}); err != nil {
		return err
	}
	app.mu.Lock()
	delete(app.tableColumnsDialogs, c.owner)
	app.mu.Unlock()
	return c.dialog.Destroy()
}
