package demo

import (
	"context"
	"fmt"
	"strings"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

type catalogPaletteEntry struct {
	key   string
	label string
	color expletives.Color
}

var tableCatalogPalette = []catalogPaletteEntry{
	{key: "black", label: "Black", color: expletives.RGB(0x00, 0x00, 0x00)},
	{key: "navy", label: "Navy", color: expletives.RGB(0x00, 0x00, 0xAA)},
	{key: "green", label: "Green", color: expletives.RGB(0x00, 0xAA, 0x00)},
	{key: "cyan", label: "Cyan", color: expletives.RGB(0x00, 0xAA, 0xAA)},
	{key: "red", label: "Red", color: expletives.RGB(0xAA, 0x00, 0x00)},
	{key: "magenta", label: "Magenta", color: expletives.RGB(0xAA, 0x00, 0xAA)},
	{key: "brown", label: "Brown", color: expletives.RGB(0xAA, 0x55, 0x00)},
	{key: "light-gray", label: "Light Gray", color: expletives.RGB(0xAA, 0xAA, 0xAA)},
	{key: "dark-gray", label: "Dark Gray", color: expletives.RGB(0x55, 0x55, 0x55)},
	{key: "blue", label: "Blue", color: expletives.RGB(0x55, 0x55, 0xFF)},
	{key: "light-green", label: "Light Green", color: expletives.RGB(0x55, 0xFF, 0x55)},
	{key: "light-cyan", label: "Light Cyan", color: expletives.RGB(0x55, 0xFF, 0xFF)},
	{key: "light-red", label: "Light Red", color: expletives.RGB(0xFF, 0x55, 0x55)},
	{key: "light-magenta", label: "Light Magenta", color: expletives.RGB(0xFF, 0x55, 0xFF)},
	{key: "yellow", label: "Yellow", color: expletives.RGB(0xFF, 0xFF, 0x55)},
	{key: "white", label: "White", color: expletives.RGB(0xFF, 0xFF, 0xFF)},
	{key: "canvas-blue", label: "Canvas Blue", color: expletives.RGB(0x00, 0x38, 0x78)},
	{key: "root-blue", label: "Root Blue", color: expletives.RGB(0x00, 0x18, 0x48)},
	{key: "dialog-gray", label: "Dialog Gray", color: expletives.RGB(0x80, 0x80, 0x80)},
}

type catalogRoleDefinition struct {
	label  string
	id     expletives.StyleID
	source expletives.StyleID
}

type catalogColorBinding struct {
	dropdown   *expletives.DropDown
	role       expletives.StyleID
	foreground bool
	initialKey string
}

type tableCatalogControl struct {
	key           string
	screen        *expletives.Panel
	table         *expletives.Table
	grid          *expletives.DataGrid
	notebook      *expletives.Notebook
	options       *expletives.ScrollablePanel
	colors        *expletives.ScrollablePanel
	columns       *expletives.Checkbox
	selection     *expletives.RadioGroup
	require       *expletives.Checkbox
	focus         *expletives.RadioGroup
	bindings      map[expletives.ControlID]catalogColorBinding
	colorBindings []catalogColorBinding
	roles         []catalogRoleDefinition
	styles        map[expletives.StyleID]expletives.ResolvedStyle
}

func (catalog *tableCatalogControl) control() expletives.Control {
	if catalog.grid != nil {
		return catalog.grid
	}
	return catalog.table
}

func (catalog *tableCatalogControl) indexColorBindings() {
	catalog.bindings = make(map[expletives.ControlID]catalogColorBinding, len(catalog.colorBindings))
	for _, binding := range catalog.colorBindings {
		catalog.bindings[binding.dropdown.ID()] = binding
	}
}

func catalogRoleID(prefix, suffix string) expletives.StyleID {
	return expletives.StyleID(prefix + ".role." + suffix)
}

func newCatalogTableRoles(
	theme expletives.Theme,
	prefix string,
	dataGrid bool,
) (expletives.TableVisualRoles, expletives.DataGridVisualRoles, []catalogRoleDefinition, []expletives.Style, error) {
	bodySource, borderSource := expletives.StyleID("table"), expletives.StyleID("table.border")
	if dataGrid {
		bodySource, borderSource = "data_grid", "data_grid.border"
	}
	tableRoles := expletives.TableVisualRoles{
		Body: catalogRoleID(prefix, "body"), Border: catalogRoleID(prefix, "border"),
		Header:             catalogRoleID(prefix, "header"),
		CurrentHeader:      catalogRoleID(prefix, "current-header"),
		SortMarker:         catalogRoleID(prefix, "sort-marker"),
		CurrentCell:        catalogRoleID(prefix, "current-cell"),
		SelectedRow:        catalogRoleID(prefix, "selected-row"),
		CurrentRow:         catalogRoleID(prefix, "current-row"),
		CurrentSelectedRow: catalogRoleID(prefix, "current-selected-row"),
		Disabled:           catalogRoleID(prefix, "disabled"),
		Empty:              catalogRoleID(prefix, "empty"), Loading: catalogRoleID(prefix, "loading"),
		Error:                 catalogRoleID(prefix, "error"),
		ScrollbarPage:         catalogRoleID(prefix, "scrollbar-page"),
		ScrollbarArrow:        catalogRoleID(prefix, "scrollbar-arrow"),
		ScrollbarThumb:        catalogRoleID(prefix, "scrollbar-thumb"),
		ScrollbarFocusedThumb: catalogRoleID(prefix, "scrollbar-focused-thumb"),
		ScrollbarDisabled:     catalogRoleID(prefix, "scrollbar-disabled"),
		ScrollbarCorner:       catalogRoleID(prefix, "scrollbar-corner"),
		ColumnsBand:           catalogRoleID(prefix, "columns-band"),
		ColumnsAction:         catalogRoleID(prefix, "columns-action"),
		ColumnsActionDefault:  catalogRoleID(prefix, "columns-action-default"),
		ColumnsActionFocused:  catalogRoleID(prefix, "columns-action-focused"),
		ColumnsActionPressed:  catalogRoleID(prefix, "columns-action-pressed"),
		ColumnsActionDisabled: catalogRoleID(prefix, "columns-action-disabled"),
		ColumnsActionMnemonic: catalogRoleID(prefix, "columns-action-mnemonic"),
		ColumnsActionShadow:   catalogRoleID(prefix, "columns-action-shadow"),
	}
	definitions := []catalogRoleDefinition{
		{"Body", tableRoles.Body, bodySource}, {"Border", tableRoles.Border, borderSource},
		{"Header", tableRoles.Header, "table.header"},
		{"Current header", tableRoles.CurrentHeader, "table.header_current"},
		{"Sort marker", tableRoles.SortMarker, "table.sort"},
		{"Current cell", tableRoles.CurrentCell, "table.cell_current"},
		{"Selected row", tableRoles.SelectedRow, "table.row_selected"},
		{"Current row", tableRoles.CurrentRow, "collection.current"},
		{"Current selected row", tableRoles.CurrentSelectedRow, "collection.current_selected"},
		{"Disabled", tableRoles.Disabled, "collection.disabled"},
		{"Empty", tableRoles.Empty, "collection.empty"},
		{"Loading", tableRoles.Loading, "collection.loading"},
		{"Error", tableRoles.Error, "collection.error"},
		{"Scroll page", tableRoles.ScrollbarPage, "scrollbar.page"},
		{"Scroll arrow", tableRoles.ScrollbarArrow, "scrollbar.arrow"},
		{"Scroll thumb", tableRoles.ScrollbarThumb, "scrollbar.thumb"},
		{"Focused scroll thumb", tableRoles.ScrollbarFocusedThumb, "scrollbar.focused"},
		{"Disabled scroll bar", tableRoles.ScrollbarDisabled, "scrollbar.disabled"},
		{"Scroll corner", tableRoles.ScrollbarCorner, "scrollbar.corner"},
		{"Columns band", tableRoles.ColumnsBand, bodySource},
		{"Columns action", tableRoles.ColumnsAction, "button"},
		{"Default Columns action", tableRoles.ColumnsActionDefault, "button.default"},
		{"Focused Columns action", tableRoles.ColumnsActionFocused, "button.focused"},
		{"Pressed Columns action", tableRoles.ColumnsActionPressed, "button.pressed"},
		{"Disabled Columns action", tableRoles.ColumnsActionDisabled, "button.disabled"},
		{"Columns mnemonic", tableRoles.ColumnsActionMnemonic, "button.mnemonic"},
		{"Columns shadow", tableRoles.ColumnsActionShadow, "button.shadow"},
	}
	gridRoles := expletives.DataGridVisualRoles{TableVisualRoles: tableRoles}
	if dataGrid {
		gridRoles.Editable = catalogRoleID(prefix, "editable")
		gridRoles.FocusedEdit = catalogRoleID(prefix, "focused-edit")
		gridRoles.InvalidEdit = catalogRoleID(prefix, "invalid-edit")
		gridRoles.InvalidCharacter = catalogRoleID(prefix, "invalid-character")
		gridRoles.TextSelection = catalogRoleID(prefix, "text-selection")
		definitions = append(definitions,
			catalogRoleDefinition{"Editable", gridRoles.Editable, "data_grid.edit"},
			catalogRoleDefinition{"Focused edit", gridRoles.FocusedEdit, "data_grid.edit_focused"},
			catalogRoleDefinition{"Invalid edit", gridRoles.InvalidEdit, "data_grid.edit_invalid"},
			catalogRoleDefinition{"Invalid character", gridRoles.InvalidCharacter, "data_grid.edit_invalid_character"},
			catalogRoleDefinition{"Text selection", gridRoles.TextSelection, "text_input.focused_selection"},
		)
	}
	styles := make([]expletives.Style, 0, len(definitions))
	for _, definition := range definitions {
		resolved, found := theme.Resolve(definition.source)
		if !found {
			return expletives.TableVisualRoles{}, expletives.DataGridVisualRoles{}, nil, nil,
				fmt.Errorf("catalog role source %q is missing", definition.source)
		}
		styles = append(styles, expletives.Style{
			ID: definition.id, Foreground: resolved.Foreground,
			Background: resolved.Background, Attributes: resolved.Attributes,
		})
	}
	return tableRoles, gridRoles, definitions, styles, nil
}

func paletteEntry(key string) (catalogPaletteEntry, bool) {
	for _, entry := range tableCatalogPalette {
		if entry.key == key {
			return entry, true
		}
	}
	return catalogPaletteEntry{}, false
}

func paletteKey(color expletives.Color) string {
	for _, entry := range tableCatalogPalette {
		if entry.color == color {
			return entry.key
		}
	}
	return ""
}

func paletteItems(current expletives.Color) ([]expletives.ListItem, string) {
	items := make([]expletives.ListItem, 0, len(tableCatalogPalette)+1)
	selected := paletteKey(current)
	for _, entry := range tableCatalogPalette {
		items = append(items, expletives.ListItem{Key: entry.key, Label: "■ " + entry.label})
	}
	if selected == "" {
		selected = "original"
		items = append(items, expletives.ListItem{
			Key: "original", Label: "■ Original " + current.String(),
		})
	}
	return items, selected
}

func catalogTableColumns(dataGrid bool) []expletives.Column {
	columns := []expletives.Column{
		{Key: "name", Header: "Name", MinimumWidth: 10, MaximumWidth: 22, Grow: 2, Sortable: true},
		{Key: "summary", Header: "Summary", Width: 18, MaximumWidth: 24, Grow: 3, Sortable: true},
		{Key: "state", Header: "State", Width: 9, Sortable: true},
		{Key: "tests", Header: "Tests", Width: 6, Alignment: expletives.TextAlignEnd, Sortable: true},
		{Key: "owner", Header: "Owner", Width: 12, Sortable: true},
	}
	if dataGrid {
		columns[0].Editable = true
		columns[0].Validator = &expletives.TextValidator{
			Enforcement: expletives.TextValidationSoft,
			Mode:        expletives.TextValidationWhitelist,
			Characters:  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 -",
		}
		columns[2].Editable = true
		columns[2].Validator = &expletives.TextValidator{
			Enforcement: expletives.TextValidationHard,
			Mode:        expletives.TextValidationBlacklist,
			Characters:  "!@#$%^&*",
		}
	}
	return columns
}

func catalogTableRows() []expletives.TableRow {
	rows := []struct {
		key, name, summary, state, tests, owner string
		disabled                                bool
	}{
		{"core", "Core", "Stable control ownership and deterministic intended-frame rendering", "Ready", "148", "Toolkit", false},
		{"terminal", "Terminal", "ANSI projection with code-page fallback and exact clipping", "Active", "62", "Adapter", false},
		{"legacy", "Legacy", "Unavailable compatibility adapter retained as a disabled fixture", "Offline", "0", "Archive", true},
		{"automation", "Automation", "Attached drive and observe through one semantic controller path", "Ready", "91", "Testing", false},
		{"layout", "Layout", "Weighted Box and Grid arrangements with observable overflow episodes", "Ready", "77", "Toolkit", false},
		{"unicode", "Unicode", "One-cell composed text with deterministic replacement for unsupported width", "Review", "54", "Display", false},
		{"menus", "Menus", "Persistent menu bar and bounded nested popup sessions", "Ready", "83", "Actions", false},
		{"dialogs", "Dialogs", "Modal standard dialogs with contextual narrow-surface warnings", "Ready", "68", "Toolkit", false},
		{"streams", "Streams", "Bounded copied content and honest drop accounting without hidden readers", "Active", "45", "Content", false},
		{"release", "Release", "Debug release and profiling artifacts share the public-client gate", "Planned", "39", "Build", false},
	}
	result := make([]expletives.TableRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, expletives.TableRow{
			Key: row.key, Disabled: row.disabled,
			DisabledReason: func() string {
				if row.disabled {
					return "Demonstration row is disabled"
				}
				return ""
			}(),
			Cells: []expletives.TableCell{
				{Column: "name", Text: row.name}, {Column: "summary", Text: row.summary},
				{Column: "state", Text: row.state}, {Column: "tests", Text: row.tests},
				{Column: "owner", Text: row.owner},
			},
		})
	}
	return result
}

func catalogTablePresentation() []expletives.TableColumnPresentation {
	return []expletives.TableColumnPresentation{
		{Column: "name", Visible: true, Wrap: expletives.TableColumnClip},
		{Column: "summary", Visible: true, Wrap: expletives.TableColumnHang},
		{Column: "state", Visible: true, Wrap: expletives.TableColumnClip},
		{Column: "tests", Visible: true, Wrap: expletives.TableColumnClip},
		{Column: "owner", Visible: true, Wrap: expletives.TableColumnWrapWords},
	}
}

func cloneResolvedStyles(styles []expletives.Style) map[expletives.StyleID]expletives.ResolvedStyle {
	result := make(map[expletives.StyleID]expletives.ResolvedStyle, len(styles))
	for _, style := range styles {
		result[style.ID] = style.Resolved()
	}
	return result
}

func catalogThemeWithStyles(
	theme expletives.Theme,
	replacements map[expletives.StyleID]expletives.ResolvedStyle,
) (expletives.Theme, error) {
	styles := theme.Styles()
	seen := make(map[expletives.StyleID]bool, len(replacements))
	for index := range styles {
		resolved, found := replacements[styles[index].ID]
		if !found {
			continue
		}
		styles[index].Foreground = resolved.Foreground
		styles[index].Background = resolved.Background
		styles[index].Attributes = resolved.Attributes
		seen[styles[index].ID] = true
	}
	for id := range replacements {
		if !seen[id] {
			return expletives.Theme{}, fmt.Errorf("catalog Theme role %q is missing", id)
		}
	}
	return expletives.NewTheme(styles...)
}

func catalogAutomationKey(parts ...string) string {
	return strings.Join(parts, ".")
}

func commitCatalogTransaction(transaction *expletives.Transaction) error {
	return transaction.Commit(context.Background())
}

func buildTableCatalog(
	transaction *expletives.Transaction,
	screen *expletives.Panel,
	tableRoles expletives.TableVisualRoles,
	gridRoles expletives.DataGridVisualRoles,
	roleDefinitions []catalogRoleDefinition,
	roleStyles []expletives.Style,
	dataGrid bool,
) (*tableCatalogControl, error) {
	key := "tables"
	optionsCommand := CommandTableOptionsChanged
	colorCommand := CommandTableColorChanged
	resetCommand := CommandTableReset
	columnsResetCommand := CommandTableColumnsReset
	if dataGrid {
		key = "data-grid"
		optionsCommand = CommandGridOptionsChanged
		colorCommand = CommandGridColorChanged
		resetCommand = CommandGridReset
		columnsResetCommand = CommandGridColumnsReset
	}
	catalog := &tableCatalogControl{
		key: key, screen: screen,
		bindings: make(map[expletives.ControlID]catalogColorBinding),
		roles:    append([]catalogRoleDefinition(nil), roleDefinitions...),
		styles:   cloneResolvedStyles(roleStyles),
	}

	panelOptions := expletives.PanelOptions{
		AutomationKey: catalogAutomationKey(key, "control"),
		Style: func() expletives.StyleID {
			if dataGrid {
				return gridRoles.Body
			}
			return tableRoles.Body
		}(),
		MinimumSize: expletives.Size{Width: 42, Height: 14},
	}
	common := expletives.ScrollablePanelOptions{
		ScrollViewOptions: expletives.ScrollViewOptions{
			PanelOptions:  panelOptions,
			ChangeCommand: CommandCollectionChanged,
		},
		BorderForm:    expletives.BorderSingle,
		HorizontalBar: expletives.ScrollBarVisibilityAuto,
		VerticalBar:   expletives.ScrollBarVisibilityAuto,
	}
	if dataGrid {
		common.BorderStyle = gridRoles.Border
		grid, err := transaction.NewDataGrid(screen, expletives.DataGridOptions{
			ScrollablePanelOptions: common,
			Features:               []expletives.TableFeature{expletives.TableFeatureColumns},
			Columns:                catalogTableColumns(true), Rows: catalogTableRows(),
			CurrentRow: "core", CurrentColumn: "name", Selected: []string{"core"},
			SelectionStyle:     expletives.TableSelectionMultiple,
			RequireSelection:   true,
			ColumnPresentation: catalogTablePresentation(),
			VisualRoles:        gridRoles,
			ActivateCommand:    CommandCollectionActivate,
			SortCommand:        CommandCollectionSort,
		})
		if err != nil {
			return nil, err
		}
		catalog.grid = grid
	} else {
		common.BorderStyle = tableRoles.Border
		table, err := transaction.NewTable(screen, expletives.TableOptions{
			ScrollablePanelOptions: common,
			Features:               []expletives.TableFeature{expletives.TableFeatureColumns},
			Columns:                catalogTableColumns(false), Rows: catalogTableRows(),
			CurrentRow: "core", CurrentColumn: "name", Selected: []string{"core"},
			SelectionStyle:   expletives.TableSelectionMultiple,
			RequireSelection: true, FocusMode: expletives.TableFocusCell,
			ColumnPresentation: catalogTablePresentation(),
			VisualRoles:        tableRoles,
			ActivateCommand:    CommandCollectionActivate,
			SortCommand:        CommandCollectionSort,
		})
		if err != nil {
			return nil, err
		}
		catalog.table = table
	}
	if err := transaction.SetFocusGuidance(
		catalog.control(),
		expletives.FocusGuidance{
			Mode: expletives.FocusGuidanceAppend,
			Text: "Options and Colors update this instance live; Tab reaches Columns",
		},
	); err != nil {
		return nil, err
	}

	notebook, err := transaction.NewNotebook(screen, expletives.TabbedPanelOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: catalogAutomationKey(key, "notebook"),
			Style:         canvasStyle.ID,
			MinimumSize:   expletives.Size{Width: 34, Height: 14},
		},
		BorderStyle: borderStyle.ID, BorderForm: expletives.BorderDouble,
		ChangeCommand: CommandNavigationChanged,
	})
	if err != nil {
		return nil, err
	}
	catalog.notebook = notebook
	optionsPage, err := transaction.NewPanel(notebook, expletives.PanelOptions{
		AutomationKey: catalogAutomationKey(key, "notebook", "page", "options"),
		Style:         canvasStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	colorsPage, err := transaction.NewPanel(notebook, expletives.PanelOptions{
		AutomationKey: catalogAutomationKey(key, "notebook", "page", "colors"),
		Style:         canvasStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	if err := transaction.SetTabs(notebook, []expletives.Tab{
		{Key: catalogAutomationKey(key, "notebook", "tab", "options"), Value: "options", Label: "Options", Mnemonic: "o", Page: optionsPage},
		{Key: catalogAutomationKey(key, "notebook", "tab", "colors"), Value: "colors", Label: "Colors", Mnemonic: "c", Page: colorsPage},
	}, "options"); err != nil {
		return nil, err
	}

	options, err := transaction.NewScrollablePanel(optionsPage, expletives.ScrollablePanelOptions{
		ScrollViewOptions: expletives.ScrollViewOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: catalogAutomationKey(key, "options"),
				Style:         "scrollable_panel",
			},
			State:        expletives.ViewportState{ContentSize: expletives.Size{Width: 40, Height: 30}},
			ContentStyle: canvasStyle.ID,
		},
		BorderStyle: "scrollable_panel.border", BorderForm: expletives.BorderNone,
		HorizontalBar: expletives.ScrollBarVisibilityAuto,
		VerticalBar:   expletives.ScrollBarVisibilityAuto,
	})
	if err != nil {
		return nil, err
	}
	catalog.options = options

	guidance, err := transaction.NewStaticText(options.Content(), expletives.StaticTextOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: catalogAutomationKey(key, "options", "guidance"),
			Style:         canvasStyle.ID,
			MinimumSize:   expletives.Size{Width: 30, Height: 2},
		},
		Text: "Changes apply immediately to the live control at left.",
		Wrap: expletives.TextWrapWords,
	})
	if err != nil {
		return nil, err
	}
	columns, err := transaction.NewCheckbox(options.Content(), expletives.CheckboxOptions{
		PanelOptions: expletives.PanelOptions{AutomationKey: catalogAutomationKey(key, "options", "columns")},
		Label:        "Show Columns... action", Mnemonic: "h", State: expletives.CheckChecked,
		ChangeCommand: optionsCommand,
	})
	if err != nil {
		return nil, err
	}
	catalog.columns = columns
	selection, err := newCatalogRadioGroup(
		transaction, options.Content(), catalogAutomationKey(key, "options", "selection"),
		"multiple", optionsCommand,
		[]catalogRadioOption{
			{"none", "No selection", "n"}, {"single", "Single selection", "s"},
			{"range", "Continuous range", "g"}, {"multiple", "Free multiple", "m"},
		},
	)
	if err != nil {
		return nil, err
	}
	catalog.selection = selection
	require, err := transaction.NewCheckbox(options.Content(), expletives.CheckboxOptions{
		PanelOptions: expletives.PanelOptions{AutomationKey: catalogAutomationKey(key, "options", "require")},
		Label:        "Require selection", Mnemonic: "r", State: expletives.CheckChecked,
		ChangeCommand: optionsCommand,
	})
	if err != nil {
		return nil, err
	}
	catalog.require = require
	var focusControl expletives.Control
	if dataGrid {
		focusControl, err = transaction.NewStaticText(options.Content(), expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: catalogAutomationKey(key, "options", "focus-fixed"),
				Style:         canvasStyle.ID,
			},
			Text: "Focus mode: Cell (DataGrid invariant)",
		})
	} else {
		catalog.focus, err = newCatalogRadioGroup(
			transaction, options.Content(), catalogAutomationKey(key, "options", "focus"),
			"cell", optionsCommand,
			[]catalogRadioOption{{"row", "Row current", "u"}, {"cell", "Cell current", "e"}},
		)
		focusControl = catalog.focus
	}
	if err != nil {
		return nil, err
	}
	columnsReset, err := transaction.NewButton(options.Content(), expletives.ButtonOptions{
		PanelOptions: expletives.PanelOptions{AutomationKey: catalogAutomationKey(key, "options", "reset-columns")},
		Command:      columnsResetCommand, Mnemonic: "l",
	})
	if err != nil {
		return nil, err
	}
	reset, err := transaction.NewButton(options.Content(), expletives.ButtonOptions{
		PanelOptions: expletives.PanelOptions{AutomationKey: catalogAutomationKey(key, "options", "reset")},
		Command:      resetCommand, Mnemonic: "d",
	})
	if err != nil {
		return nil, err
	}
	optionsLayout, err := expletives.NewBoxLayout(expletives.Vertical, expletives.BoxLayoutOptions{
		AutomationKey: catalogAutomationKey("layout", key, "options", "content"),
		Gap:           1, Insets: expletives.Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		return nil, err
	}
	for _, control := range []expletives.Control{
		guidance, columns, selection, require, focusControl, columnsReset, reset,
	} {
		if err := optionsLayout.AddPanel(control, expletives.LayoutItemOptions{}); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(options.Content(), optionsLayout); err != nil {
		return nil, err
	}

	colorsHeight := len(roleDefinitions) + 3
	colors, err := transaction.NewScrollablePanel(colorsPage, expletives.ScrollablePanelOptions{
		ScrollViewOptions: expletives.ScrollViewOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: catalogAutomationKey(key, "colors"),
				Style:         "scrollable_panel",
			},
			State:        expletives.ViewportState{ContentSize: expletives.Size{Width: 52, Height: colorsHeight}},
			ContentStyle: canvasStyle.ID,
		},
		BorderStyle: "scrollable_panel.border", BorderForm: expletives.BorderNone,
		HorizontalBar: expletives.ScrollBarVisibilityAuto,
		VerticalBar:   expletives.ScrollBarVisibilityAuto,
	})
	if err != nil {
		return nil, err
	}
	catalog.colors = colors
	colorsLayout, err := expletives.NewBoxLayout(expletives.Vertical, expletives.BoxLayoutOptions{
		AutomationKey: catalogAutomationKey("layout", key, "colors", "content"),
	})
	if err != nil {
		return nil, err
	}
	header, err := newCatalogColorRow(
		transaction, colors.Content(), key, "header", "Visual role", nil, nil,
	)
	if err != nil {
		return nil, err
	}
	if err := colorsLayout.AddPanel(header, expletives.LayoutItemOptions{}); err != nil {
		return nil, err
	}
	for index, definition := range roleDefinitions {
		resolved := catalog.styles[definition.id]
		foregroundItems, foregroundKey := paletteItems(resolved.Foreground)
		backgroundItems, backgroundKey := paletteItems(resolved.Background)
		row, foreground, background, rowErr := newCatalogColorSelectionRow(
			transaction, colors.Content(), key, index, definition.label,
			foregroundItems, foregroundKey, backgroundItems, backgroundKey,
			colorCommand,
		)
		if rowErr != nil {
			return nil, rowErr
		}
		catalog.colorBindings = append(catalog.colorBindings, catalogColorBinding{
			dropdown: foreground, role: definition.id, foreground: true,
			initialKey: foregroundKey,
		})
		catalog.colorBindings = append(catalog.colorBindings, catalogColorBinding{
			dropdown: background, role: definition.id, initialKey: backgroundKey,
		})
		if err := colorsLayout.AddPanel(row, expletives.LayoutItemOptions{}); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(colors.Content(), colorsLayout); err != nil {
		return nil, err
	}

	for _, entry := range []struct {
		page *expletives.Panel
		view expletives.Control
		name string
	}{
		{optionsPage, options, "options"}, {colorsPage, colors, "colors"},
	} {
		layout, layoutErr := expletives.NewBoxLayout(expletives.Vertical, expletives.BoxLayoutOptions{
			AutomationKey: catalogAutomationKey("layout", key, "page", entry.name),
		})
		if layoutErr != nil {
			return nil, layoutErr
		}
		if err := layout.AddPanel(entry.view, expletives.LayoutItemOptions{Grow: 1}); err != nil {
			return nil, err
		}
		if err := transaction.SetLayout(entry.page, layout); err != nil {
			return nil, err
		}
	}
	screenLayout, err := expletives.NewBoxLayout(expletives.Horizontal, expletives.BoxLayoutOptions{
		AutomationKey: catalogAutomationKey("layout", key), Gap: 1,
		Insets: expletives.Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		return nil, err
	}
	if err := screenLayout.AddPanel(catalog.control(), expletives.LayoutItemOptions{Grow: 3}); err != nil {
		return nil, err
	}
	if err := screenLayout.AddPanel(notebook, expletives.LayoutItemOptions{Grow: 1}); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(screen, screenLayout); err != nil {
		return nil, err
	}
	return catalog, nil
}

type catalogRadioOption struct {
	value    string
	label    string
	mnemonic expletives.Key
}

func newCatalogRadioGroup(
	transaction *expletives.Transaction,
	parent expletives.Container,
	key string,
	selected string,
	command expletives.CommandID,
	options []catalogRadioOption,
) (*expletives.RadioGroup, error) {
	group, err := transaction.NewRadioGroup(parent, expletives.RadioGroupOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: key,
			MinimumSize:   expletives.Size{Width: 28, Height: len(options)},
		},
		ChangeCommand: command,
	})
	if err != nil {
		return nil, err
	}
	layout, err := expletives.NewBoxLayout(expletives.Vertical, expletives.BoxLayoutOptions{
		AutomationKey: "layout." + key,
	})
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		button, buttonErr := transaction.NewRadioButton(group, expletives.RadioButtonOptions{
			PanelOptions: expletives.PanelOptions{AutomationKey: key + "." + option.value},
			Value:        option.value, Label: option.label, Mnemonic: option.mnemonic,
			Selected: option.value == selected,
		})
		if buttonErr != nil {
			return nil, buttonErr
		}
		if err := layout.AddPanel(button, expletives.LayoutItemOptions{}); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(group, layout); err != nil {
		return nil, err
	}
	return group, nil
}

func newCatalogColorDropDown(
	transaction *expletives.Transaction,
	parent expletives.Container,
	key string,
	index int,
	axis string,
	items []expletives.ListItem,
	selected string,
	command expletives.CommandID,
) (*expletives.DropDown, error) {
	return transaction.NewDropDown(parent, expletives.DropDownOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: fmt.Sprintf("%s.colors.%02d.%s", key, index, axis),
			Style:         "drop_down",
			MinimumSize:   expletives.Size{Width: 15, Height: 1},
		},
		Items: items, Current: selected, Selected: selected, PopupRows: 8,
		ChangeCommand: command,
	})
}

func newCatalogColorSelectionRow(
	transaction *expletives.Transaction,
	parent expletives.Container,
	key string,
	index int,
	label string,
	foregroundItems []expletives.ListItem,
	foregroundKey string,
	backgroundItems []expletives.ListItem,
	backgroundKey string,
	command expletives.CommandID,
) (*expletives.Panel, *expletives.DropDown, *expletives.DropDown, error) {
	rowKey := fmt.Sprintf("%02d", index)
	row, err := transaction.NewPanel(parent, expletives.PanelOptions{
		AutomationKey: catalogAutomationKey(key, "colors", "row", rowKey),
		Style:         canvasStyle.ID,
		MinimumSize:   expletives.Size{Width: 50, Height: 1},
	})
	if err != nil {
		return nil, nil, nil, err
	}
	labelControl, err := transaction.NewStaticText(row, expletives.StaticTextOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: catalogAutomationKey(key, "colors", "label", rowKey),
			Style:         canvasStyle.ID,
			MinimumSize:   expletives.Size{Width: 20, Height: 1},
		},
		Text: label,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	foreground, err := newCatalogColorDropDown(
		transaction, row, key, index, "foreground", foregroundItems,
		foregroundKey, command,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	background, err := newCatalogColorDropDown(
		transaction, row, key, index, "background", backgroundItems,
		backgroundKey, command,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	layout, err := catalogColorRowLayout(key, rowKey, labelControl, foreground, background)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := transaction.SetLayout(row, layout); err != nil {
		return nil, nil, nil, err
	}
	return row, foreground, background, nil
}

func newCatalogColorRow(
	transaction *expletives.Transaction,
	parent expletives.Container,
	key string,
	rowKey string,
	label string,
	foreground *expletives.DropDown,
	background *expletives.DropDown,
) (*expletives.Panel, error) {
	row, err := transaction.NewPanel(parent, expletives.PanelOptions{
		AutomationKey: catalogAutomationKey(key, "colors", "row", rowKey),
		Style:         canvasStyle.ID,
		MinimumSize:   expletives.Size{Width: 50, Height: 1},
	})
	if err != nil {
		return nil, err
	}
	labelControl, err := transaction.NewStaticText(row, expletives.StaticTextOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: catalogAutomationKey(key, "colors", "label", rowKey),
			Style:         canvasStyle.ID,
			MinimumSize:   expletives.Size{Width: 20, Height: 1},
		},
		Text: label,
	})
	if err != nil {
		return nil, err
	}
	if foreground == nil || background == nil {
		foregroundLabel, labelErr := transaction.NewStaticText(row, expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: catalogAutomationKey(key, "colors", "foreground", "header"),
				Style:         canvasStyle.ID, MinimumSize: expletives.Size{Width: 15, Height: 1},
			},
			Text: "Foreground",
		})
		if labelErr != nil {
			return nil, labelErr
		}
		backgroundLabel, labelErr := transaction.NewStaticText(row, expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: catalogAutomationKey(key, "colors", "background", "header"),
				Style:         canvasStyle.ID, MinimumSize: expletives.Size{Width: 15, Height: 1},
			},
			Text: "Background",
		})
		if labelErr != nil {
			return nil, labelErr
		}
		layout, layoutErr := catalogColorRowLayout(key, rowKey, labelControl, foregroundLabel, backgroundLabel)
		if layoutErr != nil {
			return nil, layoutErr
		}
		if err := transaction.SetLayout(row, layout); err != nil {
			return nil, err
		}
		return row, nil
	}
	layout, err := catalogColorRowLayout(key, rowKey, labelControl, foreground, background)
	if err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(row, layout); err != nil {
		return nil, err
	}
	return row, nil
}

func catalogColorRowLayout(
	key string,
	rowKey string,
	label expletives.Control,
	foreground expletives.Control,
	background expletives.Control,
) (*expletives.BoxLayout, error) {
	layout, err := expletives.NewBoxLayout(expletives.Horizontal, expletives.BoxLayoutOptions{
		AutomationKey: catalogAutomationKey("layout", key, "colors", "row", rowKey),
		Gap:           1,
	})
	if err != nil {
		return nil, err
	}
	for _, entry := range []struct {
		control expletives.Control
		grow    int
	}{{label, 2}, {foreground, 1}, {background, 1}} {
		if err := layout.AddPanel(entry.control, expletives.LayoutItemOptions{Grow: entry.grow}); err != nil {
			return nil, err
		}
	}
	return layout, nil
}

func (s *Scene) applyTableCatalogOptionsLocked(
	catalog *tableCatalogControl,
	target expletives.ControlID,
) (expletives.Outcome, error) {
	if catalog == nil {
		return expletives.OutcomeFailed, errorsNewCatalog("missing catalog")
	}
	var mutation func() error
	switch target {
	case catalog.columns.ID():
		features := []expletives.TableFeature(nil)
		if catalog.columns.State() == expletives.CheckChecked {
			features = []expletives.TableFeature{expletives.TableFeatureColumns}
		}
		if catalog.grid != nil {
			mutation = func() error { return catalog.grid.SetFeatures(features) }
		} else {
			mutation = func() error { return catalog.table.SetFeatures(features) }
		}
	case catalog.selection.ID(), catalog.require.ID():
		style := expletives.TableSelectionStyle(catalog.selection.Value())
		require := catalog.require.State() == expletives.CheckChecked
		if style == expletives.TableSelectionNone && require {
			if target == catalog.selection.ID() {
				require = false
			} else {
				if err := s.syncTableCatalogOptionsLocked(catalog); err != nil {
					return expletives.OutcomeFailed, err
				}
				return expletives.OutcomeRejected, nil
			}
		}
		state := catalogTableState(catalog)
		selected := append([]string(nil), state.Selected...)
		anchor, extent := "", ""
		switch style {
		case expletives.TableSelectionNone:
			selected = nil
		case expletives.TableSelectionSingle:
			if len(selected) > 1 {
				selected = selected[:1]
			}
		case expletives.TableSelectionRange:
			if len(selected) == 0 && state.CurrentRow != "" {
				selected = []string{state.CurrentRow}
			}
			if len(selected) > 0 {
				selected = []string{selected[0]}
				anchor, extent = selected[0], selected[0]
			}
		case expletives.TableSelectionMultiple:
		}
		if require && len(selected) == 0 && state.CurrentRow != "" {
			selected = []string{state.CurrentRow}
			if style == expletives.TableSelectionRange {
				anchor, extent = state.CurrentRow, state.CurrentRow
			}
		}
		policy := expletives.TableSelectionPolicy{
			Style: style, Require: require, Selected: selected,
			Anchor: anchor, Extent: extent,
		}
		if catalog.grid != nil {
			mutation = func() error { return catalog.grid.SetSelectionPolicy(policy) }
		} else {
			mutation = func() error { return catalog.table.SetSelectionPolicy(policy) }
		}
	default:
		if catalog.focus != nil && target == catalog.focus.ID() {
			mode := expletives.TableFocusMode(catalog.focus.Value())
			mutation = func() error { return catalog.table.SetFocusMode(mode) }
		}
	}
	if mutation == nil {
		return expletives.OutcomeRejected, nil
	}
	if err := mutation(); err != nil {
		if syncErr := s.syncTableCatalogOptionsLocked(catalog); syncErr != nil {
			return expletives.OutcomeFailed, syncErr
		}
		return expletives.OutcomeFailed, err
	}
	if err := s.syncTableCatalogOptionsLocked(catalog); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func catalogTableState(catalog *tableCatalogControl) expletives.TableState {
	if catalog.grid != nil {
		return catalog.grid.State().TableState
	}
	return catalog.table.State()
}

func (s *Scene) syncTableCatalogOptionsLocked(catalog *tableCatalogControl) error {
	state := catalogTableState(catalog)
	transaction := s.App.NewTransaction()
	columnsState := expletives.CheckUnchecked
	for _, feature := range state.Features {
		if feature == expletives.TableFeatureColumns {
			columnsState = expletives.CheckChecked
		}
	}
	if err := transaction.SetCheckState(catalog.columns, columnsState); err != nil {
		return err
	}
	if err := transaction.SetRadioValue(catalog.selection, string(state.SelectionStyle)); err != nil {
		return err
	}
	requireState := expletives.CheckUnchecked
	if state.SelectionStyle != expletives.TableSelectionNone && state.RequireSelection {
		requireState = expletives.CheckChecked
	}
	if err := transaction.SetCheckState(catalog.require, requireState); err != nil {
		return err
	}
	if catalog.focus != nil {
		if err := transaction.SetRadioValue(catalog.focus, string(state.FocusMode)); err != nil {
			return err
		}
	}
	return commitCatalogTransaction(transaction)
}

func (s *Scene) applyTableCatalogColorLocked(
	catalog *tableCatalogControl,
	target expletives.ControlID,
) (expletives.Outcome, error) {
	binding, found := catalog.bindings[target]
	if !found {
		return expletives.OutcomeRejected, nil
	}
	selected := binding.dropdown.State().Selected
	var color expletives.Color
	if entry, paletteFound := paletteEntry(selected); paletteFound {
		color = entry.color
	} else if selected == "original" {
		original, originalFound := catalog.styles[binding.role]
		if !originalFound {
			return expletives.OutcomeFailed, errorsNewCatalog("missing original role")
		}
		if binding.foreground {
			color = original.Foreground
		} else {
			color = original.Background
		}
	} else {
		return expletives.OutcomeRejected, nil
	}
	currentTheme := s.App.Theme()
	resolved, found := currentTheme.Resolve(binding.role)
	if !found {
		return expletives.OutcomeFailed, errorsNewCatalog("active role is missing")
	}
	if binding.foreground {
		resolved.Foreground = color
	} else {
		resolved.Background = color
	}
	nextTheme, err := catalogThemeWithStyles(
		currentTheme,
		map[expletives.StyleID]expletives.ResolvedStyle{binding.role: resolved},
	)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	transaction := s.App.NewTransaction()
	if err := transaction.SetTheme(nextTheme); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := commitCatalogTransaction(transaction); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) resetTableCatalogColumnsLocked(
	catalog *tableCatalogControl,
) (expletives.Outcome, error) {
	var err error
	if catalog.grid != nil {
		err = catalog.grid.SetColumnPresentation(catalogTablePresentation())
	} else {
		err = catalog.table.SetColumnPresentation(catalogTablePresentation())
	}
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) resetTableCatalogLocked(
	catalog *tableCatalogControl,
) (expletives.Outcome, error) {
	transaction := s.App.NewTransaction()
	if catalog.grid != nil {
		if err := transaction.SetDataGridSelectionPolicy(
			catalog.grid,
			expletives.TableSelectionPolicy{
				Style: expletives.TableSelectionMultiple, Require: true,
				Selected: []string{"core"},
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.ReplaceDataGridWithPresentation(
			catalog.grid, catalogTableColumns(true), catalogTableRows(),
			catalogTablePresentation(), "core", "name", []string{"core"},
			"", expletives.SortNone,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetDataGridFeatures(
			catalog.grid, []expletives.TableFeature{expletives.TableFeatureColumns},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
	} else {
		if err := transaction.SetTableSelectionPolicy(
			catalog.table,
			expletives.TableSelectionPolicy{
				Style: expletives.TableSelectionMultiple, Require: true,
				Selected: []string{"core"},
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.ReplaceTableWithPresentation(
			catalog.table, catalogTableColumns(false), catalogTableRows(),
			catalogTablePresentation(), "core", "name", []string{"core"},
			"", expletives.SortNone,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetTableFeatures(
			catalog.table, []expletives.TableFeature{expletives.TableFeatureColumns},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetTableFocusMode(catalog.table, expletives.TableFocusCell); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	resetTheme, err := catalogThemeWithStyles(s.App.Theme(), catalog.styles)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.SetTheme(resetTheme); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.SetCheckState(catalog.columns, expletives.CheckChecked); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.SetRadioValue(catalog.selection, "multiple"); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.SetCheckState(catalog.require, expletives.CheckChecked); err != nil {
		return expletives.OutcomeFailed, err
	}
	if catalog.focus != nil {
		if err := transaction.SetRadioValue(catalog.focus, "cell"); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	if err := transaction.SetSelectedTab(catalog.notebook, "options"); err != nil {
		return expletives.OutcomeFailed, err
	}
	for _, binding := range catalog.bindings {
		if err := transaction.SetDropDownSelection(binding.dropdown, binding.initialKey); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	if err := commitCatalogTransaction(transaction); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func errorsNewCatalog(message string) error {
	return fmt.Errorf("table catalog: %s", message)
}
