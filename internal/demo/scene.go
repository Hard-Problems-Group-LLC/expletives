// Package demo composes the stable expletives-test public-API fixture.
package demo

import (
	"context"
	"errors"
	"fmt"
	"sync"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const (
	ScenarioID                                    = "toolkit.catalog"
	CommandFixtureToggle     expletives.CommandID = "fixture.toggle"
	CommandPanelRaise        expletives.CommandID = "layout.panel.raise"
	CommandPanelLower        expletives.CommandID = "layout.panel.lower"
	CommandLayerRaise        expletives.CommandID = "layout.layer.raise"
	CommandLayerLower        expletives.CommandID = "layout.layer.lower"
	CommandScenarioReset     expletives.CommandID = "scenario.reset"
	CommandAutomationNotice  expletives.CommandID = "fixture.automation_notice"
	CommandViewHome          expletives.CommandID = "view.home"
	CommandViewText          expletives.CommandID = "view.text"
	CommandViewActions       expletives.CommandID = "view.actions"
	CommandViewMenus         expletives.CommandID = "view.menus"
	CommandViewAbout         expletives.CommandID = "view.about"
	CommandViewPanelsCore    expletives.CommandID = "view.panels.core"
	CommandViewPanelStyles   expletives.CommandID = "view.panels.styles"
	CommandViewLayoutBox     expletives.CommandID = "view.layouts.box"
	CommandViewLayoutGrid    expletives.CommandID = "view.layouts.grid"
	CommandPanelScrollbars   expletives.CommandID = "catalog.panels.scrollbars"
	CommandLayoutAbsolute    expletives.CommandID = "catalog.layouts.absolute"
	CommandStatusBar         expletives.CommandID = "chrome.status.show"
	CommandHeadersShow       expletives.CommandID = "chrome.headers.show"
	CommandHeadersAdd        expletives.CommandID = "chrome.headers.add"
	CommandHeadersRemoveTop  expletives.CommandID = "chrome.headers.remove_highest"
	CommandHeadersRemoveLow  expletives.CommandID = "chrome.headers.remove_lowest"
	CommandFooterGlobalShow  expletives.CommandID = "chrome.footer.global.show"
	CommandFooterScreenShow  expletives.CommandID = "chrome.footer.screen.show"
	CommandFooterFocusShow   expletives.CommandID = "chrome.footer.focus.show"
	CommandSelection         expletives.CommandID = "catalog.controls.selection"
	CommandSelectionChanged  expletives.CommandID = "selection.changed"
	CommandTextInput         expletives.CommandID = "catalog.controls.input"
	CommandTextChanged       expletives.CommandID = "text.changed"
	CommandNumberChanged     expletives.CommandID = "number.changed"
	CommandProgress          expletives.CommandID = "catalog.controls.progress"
	CommandProgressTick      expletives.CommandID = "progress.tick"
	CommandProgressReset     expletives.CommandID = "progress.reset"
	CommandProgressComplete  expletives.CommandID = "progress.complete"
	CommandProgressFail      expletives.CommandID = "progress.fail"
	CommandProgressCancel    expletives.CommandID = "progress.cancel"
	CommandProgressMotion    expletives.CommandID = "progress.reduced_motion"
	CommandNavigation        expletives.CommandID = "catalog.controls.navigation"
	CommandNavigationChanged expletives.CommandID = "navigation.changed"
	CommandScrolling         expletives.CommandID = "catalog.controls.scrolling"
	CommandCollections       expletives.CommandID = "catalog.controls.collections"
	CommandPanelMenu         expletives.CommandID = "catalog.menus.panel"
	CommandContextMenu       expletives.CommandID = "catalog.menus.context"
	CommandDialogMessage     expletives.CommandID = "catalog.dialogs.message"
	CommandDialogConfirm     expletives.CommandID = "catalog.dialogs.confirm"
	CommandDialogInput       expletives.CommandID = "catalog.dialogs.input"
	CommandDialogProgress    expletives.CommandID = "catalog.dialogs.progress"
	CommandUnavailable       expletives.CommandID = "fixture.unavailable"
	CommandAppQuit           expletives.CommandID = "app.quit"
	CommandAppInterrupt      expletives.CommandID = "app.interrupt"
)

var catalogScreens = []struct {
	id    expletives.CommandID
	label string
}{
	{CommandViewHome, "Home"},
	{CommandViewPanelsCore, "Core Panels"},
	{CommandViewPanelStyles, "Visual Styles"},
	{CommandViewLayoutBox, "Box Layout"},
	{CommandViewLayoutGrid, "Grid Layout"},
	{CommandViewText, "Text / Display"},
	{CommandViewActions, "Actions"},
	{CommandSelection, "Selection"},
	{CommandTextInput, "Text / Numeric Input"},
	{CommandProgress, "Progress"},
	{CommandNavigation, "Navigation"},
	{CommandViewMenus, "Menu Bar"},
	{CommandViewAbout, "About"},
}

var (
	rootStyle = expletives.Style{
		ID:         "fixture.root",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x18, 0x48),
	}
	canvasStyle = expletives.Style{
		ID:         "fixture.canvas",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	borderStyle = expletives.Style{
		ID:         "fixture.border",
		Foreground: expletives.RGB(0xFF, 0xFF, 0x55),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	redStyle = expletives.Style{
		ID:         "fixture.red",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0xA8, 0x00, 0x20),
	}
	greenStyle = expletives.Style{
		ID:         "fixture.green",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0x22, 0xC5, 0x5E),
	}
	magentaStyle = expletives.Style{
		ID:         "fixture.magenta",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0xB8, 0x20, 0xD0),
	}
	yellowStyle = expletives.Style{
		ID:         "fixture.yellow",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xE8, 0xC8, 0x20),
	}
	menuStyle = expletives.Style{
		ID:         "menu_bar",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	menuPopupStyle = expletives.Style{
		ID:         "menu.popup",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	menuBorderStyle = expletives.Style{
		ID:         "menu.border",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	menuMnemonicStyle = expletives.Style{
		ID:         "menu.mnemonic",
		Foreground: expletives.RGB(0xAA, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	menuFocusedStyle = expletives.Style{
		ID:         "menu.focused",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0xAA, 0x00),
	}
	menuFocusedMnemonicStyle = expletives.Style{
		ID:         "menu.focused_mnemonic",
		Foreground: expletives.RGB(0xAA, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0xAA, 0x00),
	}
	menuDisabledStyle = expletives.Style{
		ID:         "menu.disabled",
		Foreground: expletives.RGB(0x80, 0x80, 0x80),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	menuFocusedDisabledStyle = expletives.Style{
		ID:         "menu.focused_disabled",
		Foreground: expletives.RGB(0x80, 0x80, 0x80),
		Background: expletives.RGB(0x00, 0xAA, 0x00),
	}
	menuShadowStyle = expletives.Style{
		ID:         "menu.shadow",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	statusStyle = expletives.Style{
		ID:         "status_bar",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	statusShortcutStyle = expletives.Style{
		ID:         "status.shortcut",
		Foreground: expletives.RGB(0xAA, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	statusDisabledStyle = expletives.Style{
		ID:         "status.disabled",
		Foreground: expletives.RGB(0x80, 0x80, 0x80),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
	headerStyle = expletives.Style{
		ID:         "header",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x78, 0x78),
	}
	footerStyle = expletives.Style{
		ID:         "footer",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x78, 0x00, 0x78),
	}
	selectionBaseStyle = expletives.Style{
		ID:         "selection.base",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	selectionMnemonicStyle = expletives.Style{
		ID:         "selection.mnemonic",
		Foreground: expletives.RGB(0xFF, 0x55, 0x55),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	selectionFocusedStyle = expletives.Style{
		ID:         "selection.focused",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0xAA, 0x00),
	}
	selectionFocusedMnemonicStyle = expletives.Style{
		ID:         "selection.focused_mnemonic",
		Foreground: expletives.RGB(0xAA, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0xAA, 0x00),
	}
	selectionDisabledStyle = expletives.Style{
		ID:         "selection.disabled",
		Foreground: expletives.RGB(0x80, 0x80, 0x80),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	textFieldStyle = expletives.Style{
		ID:         "text_field",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x00, 0xAA),
	}
	textInputValidStyle = expletives.Style{
		ID:         "text_input.valid",
		Foreground: expletives.RGB(0x00, 0xFF, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0xAA),
	}
	textInputInvalidStyle = expletives.Style{
		ID:         "text_input.invalid",
		Foreground: expletives.RGB(0xFF, 0xFF, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0xAA),
	}
	textInputInvalidCharacterStyle = expletives.Style{
		ID:         "text_input.invalid_character",
		Foreground: expletives.RGB(0xFF, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0xAA),
	}
	textInputSelectionStyle = expletives.Style{
		ID:         "text_input.selection",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	textInputDisabledStyle = expletives.Style{
		ID:         "text_input.disabled",
		Foreground: expletives.RGB(0x80, 0x80, 0x80),
		Background: expletives.RGB(0x00, 0x00, 0xAA),
	}
	textInputFocusedStyle = expletives.Style{
		ID:         "text_input.focused",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	textInputFocusedValidStyle = expletives.Style{
		ID:         "text_input.focused_valid",
		Foreground: expletives.RGB(0x00, 0xFF, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	textInputFocusedInvalidStyle = expletives.Style{
		ID:         "text_input.focused_invalid",
		Foreground: expletives.RGB(0xFF, 0xFF, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	textInputFocusedInvalidCharacterStyle = expletives.Style{
		ID:         "text_input.focused_invalid_character",
		Foreground: expletives.RGB(0xFF, 0x00, 0x00),
		Background: expletives.RGB(0x00, 0x00, 0x00),
	}
	textInputFocusedSelectionStyle = expletives.Style{
		ID:         "text_input.focused_selection",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xC0, 0xC0, 0xC0),
	}
)

// Scene owns the catalog controls and its small application controller state.
type Scene struct {
	App *expletives.App

	mu                      sync.Mutex
	accent                  *expletives.Panel
	accentControls          []expletives.Control
	layer                   *expletives.BoxLayout
	status                  *expletives.StatusBar
	headers                 []*expletives.Header
	globalHotkeyFooter      *expletives.Footer
	screenHotkeyFooter      *expletives.Footer
	focusGuidanceFooter     *expletives.Footer
	selectionCheckbox       *expletives.Checkbox
	selectionThreeState     *expletives.Checkbox
	selectionRadioGroup     *expletives.RadioGroup
	selectionCycleField     *expletives.CycleField
	selectionSelectField    *expletives.SelectField
	inputPlain              *expletives.TextField
	inputSoft               *expletives.TextField
	inputHard               *expletives.TextField
	inputPassword           *expletives.TextField
	inputNumber             *expletives.NumberField
	inputSpin               *expletives.SpinBox
	inputArea               *expletives.TextArea
	inputViewport           *expletives.ScrollablePanel
	progressBar             *expletives.ProgressBar
	progressIndeterminate   *expletives.ProgressBar
	progressMeter           *expletives.Meter
	progressVerticalMeter   *expletives.Meter
	progressSpinner         *expletives.Spinner
	progressDots            *expletives.ActivityDots
	navigationHorizontal    *expletives.ScrollBar
	navigationVertical      *expletives.ScrollBar
	navigationTabs          *expletives.TabbedPanel
	navigationNotebook      *expletives.Notebook
	screens                 map[expletives.CommandID]*expletives.Panel
	activeScreen            expletives.CommandID
	automationEnabled       bool
	automationNoticeVisible bool
	statusVisible           bool
	headersVisible          bool
	globalFooterVisible     bool
	screenFooterVisible     bool
	focusFooterVisible      bool
	nextHeader              int
	toggled                 bool
	progressTick            uint64
	progressReduced         bool
}

// New constructs the complete fixture through the public toolkit API.
func New(size expletives.Size, automationPath string) (*Scene, error) {
	return NewWithRootConstraints(
		size,
		automationPath,
		expletives.RootConstraints{},
	)
}

// NewWithRootConstraints constructs the fixture with an optional centered
// application-root sizing policy.
func NewWithRootConstraints(
	size expletives.Size,
	automationPath string,
	constraints expletives.RootConstraints,
) (*Scene, error) {
	theme, err := expletives.NewTheme(
		rootStyle,
		canvasStyle,
		borderStyle,
		redStyle,
		greenStyle,
		magentaStyle,
		yellowStyle,
		menuStyle,
		menuPopupStyle,
		menuBorderStyle,
		menuMnemonicStyle,
		menuFocusedStyle,
		menuFocusedMnemonicStyle,
		menuDisabledStyle,
		menuFocusedDisabledStyle,
		menuShadowStyle,
		statusStyle,
		statusShortcutStyle,
		statusDisabledStyle,
		headerStyle,
		footerStyle,
		expletives.Style{
			ID:         "checkbox",
			Foreground: selectionBaseStyle.Foreground,
			Background: selectionBaseStyle.Background,
		},
		expletives.Style{
			ID:         "radio_group",
			Foreground: selectionBaseStyle.Foreground,
			Background: selectionBaseStyle.Background,
		},
		expletives.Style{
			ID:         "radio_button",
			Foreground: selectionBaseStyle.Foreground,
			Background: selectionBaseStyle.Background,
		},
		expletives.Style{
			ID:         "cycle_field",
			Foreground: selectionBaseStyle.Foreground,
			Background: selectionBaseStyle.Background,
		},
		expletives.Style{
			ID:         "select_field",
			Foreground: selectionBaseStyle.Foreground,
			Background: selectionBaseStyle.Background,
		},
		selectionMnemonicStyle,
		selectionFocusedStyle,
		selectionFocusedMnemonicStyle,
		selectionDisabledStyle,
		textFieldStyle,
		textInputValidStyle,
		textInputInvalidStyle,
		textInputInvalidCharacterStyle,
		textInputSelectionStyle,
		textInputDisabledStyle,
		textInputFocusedStyle,
		textInputFocusedValidStyle,
		textInputFocusedInvalidStyle,
		textInputFocusedInvalidCharacterStyle,
		textInputFocusedSelectionStyle,
		expletives.Style{
			ID:         "number_field",
			Foreground: textFieldStyle.Foreground,
			Background: textFieldStyle.Background,
		},
		expletives.Style{
			ID:         "spin_box",
			Foreground: textFieldStyle.Foreground,
			Background: textFieldStyle.Background,
		},
		expletives.Style{
			ID:         "text_area",
			Foreground: textFieldStyle.Foreground,
			Background: textFieldStyle.Background,
		},
		expletives.Style{
			ID:         "progress_bar",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "meter",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "spinner",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "activity_dots",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "progress.fill",
			Foreground: greenStyle.Foreground,
			Background: greenStyle.Background,
		},
		expletives.Style{
			ID:         "progress.text",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: expletives.RGB(0x00, 0x00, 0x00),
			Attributes: expletives.StyleBold,
		},
		expletives.Style{
			ID:         "progress.completed",
			Foreground: greenStyle.Foreground,
			Background: greenStyle.Background,
		},
		expletives.Style{
			ID:         "progress.failed",
			Foreground: redStyle.Foreground,
			Background: redStyle.Background,
		},
		expletives.Style{
			ID:         "progress.cancelled",
			Foreground: yellowStyle.Foreground,
			Background: yellowStyle.Background,
		},
		expletives.Style{
			ID:         "scroll_bar",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollable_panel",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollable_panel.border",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollbar.page",
			Foreground: expletives.RGB(0x80, 0x80, 0x80),
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollbar.arrow",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollbar.thumb",
			Foreground: expletives.RGB(0xC0, 0xC0, 0xC0),
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollbar.focused",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: expletives.RGB(0x00, 0xAA, 0x00),
		},
		expletives.Style{
			ID:         "scrollbar.disabled",
			Foreground: expletives.RGB(0x80, 0x80, 0x80),
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "scrollbar.corner",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "tabbed_panel",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "notebook",
			Foreground: canvasStyle.Foreground,
			Background: canvasStyle.Background,
		},
		expletives.Style{
			ID:         "tabbed_panel.border",
			Foreground: borderStyle.Foreground,
			Background: borderStyle.Background,
		},
		expletives.Style{
			ID:         "notebook.border",
			Foreground: borderStyle.Foreground,
			Background: borderStyle.Background,
		},
		expletives.Style{
			ID:         "tab.normal",
			Foreground: expletives.RGB(0x00, 0x00, 0x00),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
		expletives.Style{
			ID:         "tab.mnemonic",
			Foreground: expletives.RGB(0xAA, 0x00, 0x00),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
		expletives.Style{
			ID:         "tab.selected",
			Foreground: expletives.RGB(0x00, 0x00, 0x00),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
		expletives.Style{
			ID:         "tab.selected_mnemonic",
			Foreground: expletives.RGB(0xAA, 0x00, 0x00),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
		expletives.Style{
			ID:         "tab.focused",
			Foreground: expletives.RGB(0x00, 0x00, 0x00),
			Background: expletives.RGB(0x00, 0xAA, 0x00),
		},
		expletives.Style{
			ID:         "tab.focused_mnemonic",
			Foreground: expletives.RGB(0xAA, 0x00, 0x00),
			Background: expletives.RGB(0x00, 0xAA, 0x00),
		},
		expletives.Style{
			ID:         "tab.disabled",
			Foreground: expletives.RGB(0x80, 0x80, 0x80),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
		expletives.Style{
			ID:         "tab.continuation",
			Foreground: expletives.RGB(0x00, 0x00, 0x00),
			Background: expletives.RGB(0xC0, 0xC0, 0xC0),
		},
	)
	if err != nil {
		return nil, err
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:            size,
		RootConstraints: constraints,
		Theme:           theme,
		RootStyle:       rootStyle.ID,
		Scenario:        ScenarioID,
	})
	if err != nil {
		return nil, err
	}
	automationEnabled := automationPath != ""
	for _, definition := range initialCommandDefinitions(automationEnabled) {
		if err := app.RegisterCommand(definition); err != nil {
			return nil, err
		}
	}
	menuItems, err := catalogMenuItems()
	if err != nil {
		return nil, err
	}

	transaction := app.NewTransaction()
	outer, err := transaction.NewFrame(app.Root(), expletives.FrameOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "fixture",
			Style:         canvasStyle.ID,
		},
		Title:       "expletives Toolkit Catalog",
		BorderStyle: borderStyle.ID,
		BorderForm:  expletives.BorderSingle,
	})
	if err != nil {
		return nil, err
	}
	_, err = transaction.NewMenuBar(app.Root(), expletives.MenuBarOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "menu.main",
			Style:         menuStyle.ID,
		},
		Items:                menuItems,
		PopupStyle:           menuPopupStyle.ID,
		BorderStyle:          menuBorderStyle.ID,
		MnemonicStyle:        menuMnemonicStyle.ID,
		FocusedStyle:         menuFocusedStyle.ID,
		FocusedMnemonicStyle: menuFocusedMnemonicStyle.ID,
		DisabledStyle:        menuDisabledStyle.ID,
		FocusedDisabledStyle: menuFocusedDisabledStyle.ID,
		ShadowStyle:          menuShadowStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	statusBar, err := transaction.NewStatusBar(
		app.Root(),
		expletives.StatusBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "status.main",
				Style:         statusStyle.ID,
			},
			Segments: catalogStatusSegments(
				CommandViewHome,
				automationEnabled,
				automationEnabled,
			),
			ShortcutStyle: statusShortcutStyle.ID,
			DisabledStyle: statusDisabledStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	primaryHeader, err := transaction.NewHeader(
		app.Root(),
		expletives.HeaderOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: "header.primary",
			Style:         headerStyle.ID,
			Hidden:        true,
		}},
	)
	if err != nil {
		return nil, err
	}
	secondaryHeader, err := transaction.NewHeader(
		app.Root(),
		expletives.HeaderOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: "header.secondary",
			Style:         greenStyle.ID,
			Hidden:        true,
		}},
	)
	if err != nil {
		return nil, err
	}
	primaryFooter, err := transaction.NewFooter(
		app.Root(),
		expletives.FooterOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: "footer.hotkeys.global",
			Style:         footerStyle.ID,
		}},
	)
	if err != nil {
		return nil, err
	}
	recentFooter, err := transaction.NewFooter(
		app.Root(),
		expletives.FooterOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: "footer.hotkeys.screen",
			Style:         redStyle.ID,
		}},
	)
	if err != nil {
		return nil, err
	}
	focusFooter, err := transaction.NewFooter(
		app.Root(),
		expletives.FooterOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: "footer.guidance.focus",
			Style:         greenStyle.ID,
		}},
	)
	if err != nil {
		return nil, err
	}
	content, err := transaction.NewPanel(outer, expletives.PanelOptions{
		AutomationKey: "catalog.content",
		Style:         canvasStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	homeScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.home",
		Style:         canvasStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	panelsCoreScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.panels.core",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	panelStylesScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.panels.styles",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	layoutBoxScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.layouts.box",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	layoutGridScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.layouts.grid",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	textScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.text",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	actionsScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.actions",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	selectionScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.selection",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	inputScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.input",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	progressScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.progress",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	navigationScreen, err := transaction.NewPanel(
		content,
		expletives.PanelOptions{
			AutomationKey: "screen.navigation",
			Style:         canvasStyle.ID,
			Hidden:        true,
		},
	)
	if err != nil {
		return nil, err
	}
	menusScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.menus",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	statusScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.status",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	chromeScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.headers_footers",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	aboutScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.about",
		Style:         canvasStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		return nil, err
	}
	menusText, err := transaction.NewStaticText(
		menusScreen,
		expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "menus.overview",
				Style:         canvasStyle.ID,
			},
			Text: "Menu demonstrations\n\n" +
				"Use Alt plus a red mnemonic, F9, arrows, Enter, and Escape. " +
				"Panel-owned and context menus arrive with their required controls.",
			HorizontalAlignment: expletives.TextAlignCenter,
			VerticalAlignment:   expletives.TextAlignCenter,
			Wrap:                expletives.TextWrapWords,
		},
	)
	if err != nil {
		return nil, err
	}
	statusText, err := transaction.NewStaticText(
		statusScreen,
		expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "status.overview",
				Style:         canvasStyle.ID,
			},
			Text: "Status Bar\n\n" +
				"The physical bottom row shows current context and shared " +
				"command hints. Resize the terminal to observe deterministic " +
				"priority and clipping; command labels, bindings, and disabled " +
				"state come from the same Action registry.",
			HorizontalAlignment: expletives.TextAlignCenter,
			VerticalAlignment:   expletives.TextAlignCenter,
			Wrap:                expletives.TextWrapWords,
		},
	)
	if err != nil {
		return nil, err
	}
	chromeText, err := transaction.NewStaticText(
		chromeScreen,
		expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "chrome.overview",
				Style:         canvasStyle.ID,
			},
			Text: "Headers and Footers\n\n" +
				"Two one-row Headers are below the Main Menu. Two Footers are " +
				"above the Status Bar, with the most recent highest. Their " +
				"children use compatible horizontal Box and one-row Grid layouts.",
			HorizontalAlignment: expletives.TextAlignCenter,
			VerticalAlignment:   expletives.TextAlignCenter,
			Wrap:                expletives.TextWrapWords,
		},
	)
	if err != nil {
		return nil, err
	}
	headerProduct, err := transaction.NewLabel(
		primaryHeader,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "header.product", Style: headerStyle.ID,
			},
			Text: "expletives",
		},
	)
	if err != nil {
		return nil, err
	}
	headerSection, err := transaction.NewLabel(
		primaryHeader,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "header.section", Style: headerStyle.ID,
			},
			Text:                "Application Chrome",
			HorizontalAlignment: expletives.TextAlignCenter,
		},
	)
	if err != nil {
		return nil, err
	}
	headerMode, err := transaction.NewLabel(
		primaryHeader,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "header.mode", Style: headerStyle.ID,
			},
			Text:                "BoxLayout",
			HorizontalAlignment: expletives.TextAlignEnd,
		},
	)
	if err != nil {
		return nil, err
	}
	headerGridLeft, err := transaction.NewLabel(
		secondaryHeader,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "header.grid.left", Style: greenStyle.ID,
			},
			Text: "Grid column 1",
		},
	)
	if err != nil {
		return nil, err
	}
	headerGridRight, err := transaction.NewLabel(
		secondaryHeader,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "header.grid.right", Style: greenStyle.ID,
			},
			Text:                "Grid column 2",
			HorizontalAlignment: expletives.TextAlignEnd,
		},
	)
	if err != nil {
		return nil, err
	}
	globalHotkeys, err := transaction.NewHotkeyBar(
		primaryFooter,
		expletives.HotkeyBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "footer.hotkeys.global.items",
				Style:         footerStyle.ID,
			},
			Items: []expletives.HotkeyBarItem{
				{Command: CommandAppQuit},
				{Command: CommandAppInterrupt},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	focusGuidance, err := transaction.NewFocusGuideBar(
		focusFooter,
		expletives.FocusGuideBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "footer.guidance.focus.text",
				Style:         greenStyle.ID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	aboutText, err := transaction.NewStaticText(
		aboutScreen,
		expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "help.about",
				Style:         canvasStyle.ID,
			},
			Text: "expletives\n\n" +
				"A reusable Go character-cell TUI toolkit.\n" +
				"This catalog is its human and automation test application.",
			HorizontalAlignment: expletives.TextAlignCenter,
			VerticalAlignment:   expletives.TextAlignCenter,
			Wrap:                expletives.TextWrapWords,
		},
	)
	if err != nil {
		return nil, err
	}

	red, err := transaction.NewPanel(panelsCoreScreen, expletives.PanelOptions{
		AutomationKey: "panel.red",
		MinimumSize:   expletives.Size{Width: 10, Height: 5},
		Style:         redStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	accent, err := transaction.NewPanel(
		panelsCoreScreen,
		expletives.PanelOptions{
			AutomationKey: "panel.accent",
			MinimumSize:   expletives.Size{Width: 12, Height: 5},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	group, err := transaction.NewGroupBox(
		panelsCoreScreen,
		expletives.GroupBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "group",
				MinimumSize:   expletives.Size{Width: 16, Height: 7},
				Style:         canvasStyle.ID,
			},
			Title:       "Nested Panel",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderDouble,
		},
	)
	if err != nil {
		return nil, err
	}
	nestedPanel, err := transaction.NewPanel(group, expletives.PanelOptions{
		AutomationKey: "panel.nested",
		MinimumSize:   expletives.Size{Width: 12, Height: 3},
		Style:         yellowStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	layerFrame, err := transaction.NewFrame(
		layoutBoxScreen,
		expletives.FrameOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "layers",
				MinimumSize:   expletives.Size{Width: 20, Height: 6},
				Style:         canvasStyle.ID,
			},
			Title:       "Stacked Box Layouts: green over red",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderSingle,
		},
	)
	if err != nil {
		return nil, err
	}
	layerBack, err := transaction.NewPanel(
		layerFrame,
		expletives.PanelOptions{
			AutomationKey: "layer.back",
			MinimumSize:   expletives.Size{Width: 1, Height: 1},
			Style:         redStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	layerFront, err := transaction.NewPanel(
		layerFrame,
		expletives.PanelOptions{
			AutomationKey: "layer.front",
			MinimumSize:   expletives.Size{Width: 1, Height: 1},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	styleSamples := []struct {
		key   string
		title string
		form  expletives.BorderForm
	}{
		{"none", "No Frame", expletives.BorderNone},
		{"single", "Single", expletives.BorderSingle},
		{"double", "Double", expletives.BorderDouble},
		{"shade-light", "Light Shade", expletives.BorderShadeLight},
		{"shade-medium", "Medium Shade", expletives.BorderShadeMedium},
		{"shade-dark", "Dark Shade", expletives.BorderShadeDark},
		{"block", "Full Cell", expletives.BorderBlock},
	}
	styleFrames := make([]expletives.Control, 0, len(styleSamples))
	styleLabels := make([]expletives.Control, 0, len(styleSamples))
	for _, sample := range styleSamples {
		frame, frameErr := transaction.NewFrame(
			panelStylesScreen,
			expletives.FrameOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "panel.style." + sample.key,
					MinimumSize:   expletives.Size{Width: 9, Height: 4},
					Style:         canvasStyle.ID,
				},
				Title:       sample.title,
				BorderStyle: borderStyle.ID,
				BorderForm:  sample.form,
			},
		)
		if frameErr != nil {
			return nil, frameErr
		}
		label, labelErr := transaction.NewStaticText(
			frame,
			expletives.StaticTextOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "panel.style." + sample.key + ".label",
					Style:         canvasStyle.ID,
				},
				Text:                sample.title,
				HorizontalAlignment: expletives.TextAlignCenter,
				VerticalAlignment:   expletives.TextAlignCenter,
				Wrap:                expletives.TextWrapWords,
			},
		)
		if labelErr != nil {
			return nil, labelErr
		}
		styleFrames = append(styleFrames, frame)
		styleLabels = append(styleLabels, label)
	}

	boxArrangement, err := transaction.NewFrame(
		layoutBoxScreen,
		expletives.FrameOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "layout.box.arrangement",
				MinimumSize:   expletives.Size{Width: 20, Height: 5},
				Style:         canvasStyle.ID,
			},
			Title:       "Nested Horizontal / Vertical Box Layouts",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderSingle,
		},
	)
	if err != nil {
		return nil, err
	}
	boxLeft, err := transaction.NewPanel(
		boxArrangement,
		expletives.PanelOptions{
			AutomationKey: "layout.box.left",
			MinimumSize:   expletives.Size{Width: 8, Height: 3},
			Style:         redStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	boxTop, err := transaction.NewPanel(
		boxArrangement,
		expletives.PanelOptions{
			AutomationKey: "layout.box.top",
			MinimumSize:   expletives.Size{Width: 8, Height: 1},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	boxBottom, err := transaction.NewPanel(
		boxArrangement,
		expletives.PanelOptions{
			AutomationKey: "layout.box.bottom",
			MinimumSize:   expletives.Size{Width: 8, Height: 1},
			Style:         magentaStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	gridPanels := make([]expletives.Control, 0, 6)
	gridStyles := []expletives.StyleID{
		redStyle.ID,
		greenStyle.ID,
		magentaStyle.ID,
		yellowStyle.ID,
		redStyle.ID,
		greenStyle.ID,
	}
	for index, style := range gridStyles {
		panel, panelErr := transaction.NewPanel(
			layoutGridScreen,
			expletives.PanelOptions{
				AutomationKey: fmt.Sprintf("layout.grid.cell.%d", index+1),
				MinimumSize:   expletives.Size{Width: 8, Height: 3},
				Style:         style,
			},
		)
		if panelErr != nil {
			return nil, panelErr
		}
		gridPanels = append(gridPanels, panel)
	}

	textRed, err := transaction.NewPanel(
		textScreen,
		expletives.PanelOptions{
			AutomationKey: "display.panel.red",
			MinimumSize:   expletives.Size{Width: 20, Height: 7},
			Style:         redStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	textAccent, err := transaction.NewPanel(
		textScreen,
		expletives.PanelOptions{
			AutomationKey: "display.panel.accent",
			MinimumSize:   expletives.Size{Width: 20, Height: 7},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	label, err := transaction.NewLabel(textRed, expletives.LabelOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "display.label",
			Style:         redStyle.ID,
		},
		Text:                "Label -> accent",
		HorizontalAlignment: expletives.TextAlignCenter,
		VerticalAlignment:   expletives.TextAlignCenter,
		Target:              textAccent,
		Mnemonic:            "a",
	})
	if err != nil {
		return nil, err
	}
	separator, err := transaction.NewSeparator(
		textRed,
		expletives.SeparatorOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "display.separator",
				Style:         redStyle.ID,
			},
			Orientation: expletives.Horizontal,
			Form:        expletives.BorderDouble,
		},
	)
	if err != nil {
		return nil, err
	}
	staticText, err := transaction.NewStaticText(
		textAccent,
		expletives.StaticTextOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "display.static_text",
				Style:         greenStyle.ID,
			},
			Text:                "StaticText wraps words",
			HorizontalAlignment: expletives.TextAlignCenter,
			VerticalAlignment:   expletives.TextAlignCenter,
			Wrap:                expletives.TextWrapWords,
		},
	)
	if err != nil {
		return nil, err
	}
	rule, err := transaction.NewRule(
		textAccent,
		expletives.RuleOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "display.rule",
				Style:         greenStyle.ID,
			},
			Orientation: expletives.Horizontal,
			Form:        expletives.BorderSingle,
			Text:        "Rule",
			Alignment:   expletives.TextAlignCenter,
		},
	)
	if err != nil {
		return nil, err
	}

	actionPanel, err := transaction.NewPanel(
		actionsScreen,
		expletives.PanelOptions{
			AutomationKey: "action.panel",
			MinimumSize:   expletives.Size{Width: 18, Height: 8},
			Style:         yellowStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	actionPreview, err := transaction.NewPanel(
		actionsScreen,
		expletives.PanelOptions{
			AutomationKey: "action.preview",
			MinimumSize:   expletives.Size{Width: 18, Height: 8},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	toggleButton, err := transaction.NewButton(
		actionPanel,
		expletives.ButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.toggle",
				Style:         yellowStyle.ID,
			},
			Command:  CommandFixtureToggle,
			Mnemonic: "g",
			Default:  true,
		},
	)
	if err != nil {
		return nil, err
	}
	resetButton, err := transaction.NewButton(
		actionPanel,
		expletives.ButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.reset",
				Style:         yellowStyle.ID,
			},
			Command:  CommandScenarioReset,
			Mnemonic: "r",
		},
	)
	if err != nil {
		return nil, err
	}
	disabledButton, err := transaction.NewButton(
		actionPanel,
		expletives.ButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.disabled",
				Style:         yellowStyle.ID,
			},
			Command: CommandUnavailable,
		},
	)
	if err != nil {
		return nil, err
	}
	quitButton, err := transaction.NewButton(
		actionPanel,
		expletives.ButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.quit",
				Style:         yellowStyle.ID,
			},
			Command:  CommandAppQuit,
			Mnemonic: "q",
			Cancel:   true,
		},
	)
	if err != nil {
		return nil, err
	}

	checkboxGroup, err := transaction.NewGroupBox(
		selectionScreen,
		expletives.GroupBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.checkboxes",
				MinimumSize:   expletives.Size{Width: 20, Height: 8},
				Style:         canvasStyle.ID,
			},
			Title:       "Checkboxes",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderSingle,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionCheckbox, err := transaction.NewCheckbox(
		checkboxGroup,
		expletives.CheckboxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.checkbox.two_state",
			},
			Label:         "Two state",
			Mnemonic:      "c",
			ChangeCommand: CommandSelectionChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := transaction.SetFocusGuidance(
		selectionCheckbox,
		expletives.FocusGuidance{
			Mode: expletives.FocusGuidanceAppend,
			Text: "Two-state Selection demonstration",
		},
	); err != nil {
		return nil, err
	}
	selectionThreeState, err := transaction.NewCheckbox(
		checkboxGroup,
		expletives.CheckboxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.checkbox.three_state",
			},
			Label:         "Three state",
			Mnemonic:      "s",
			State:         expletives.CheckIndeterminate,
			ThreeState:    true,
			ChangeCommand: CommandSelectionChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionCheckboxDisabled, err := transaction.NewCheckbox(
		checkboxGroup,
		expletives.CheckboxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.checkbox.disabled",
			},
			Label:          "Disabled",
			Disabled:       true,
			DisabledReason: "Demonstration Checkbox is disabled",
		},
	)
	if err != nil {
		return nil, err
	}
	radioBox, err := transaction.NewGroupBox(
		selectionScreen,
		expletives.GroupBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radios",
				MinimumSize:   expletives.Size{Width: 20, Height: 8},
				Style:         canvasStyle.ID,
			},
			Title:       "Radio Group",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderSingle,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionRadioGroup, err := transaction.NewRadioGroup(
		radioBox,
		expletives.RadioGroupOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radio.group",
				MinimumSize:   expletives.Size{Width: 16, Height: 5},
			},
			ChangeCommand: CommandSelectionChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionRadioOne, err := transaction.NewRadioButton(
		selectionRadioGroup,
		expletives.RadioButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radio.one",
			},
			Value: "one", Label: "Option one", Mnemonic: "o", Selected: true,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionRadioDisabled, err := transaction.NewRadioButton(
		selectionRadioGroup,
		expletives.RadioButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radio.disabled",
			},
			Value: "disabled", Label: "Disabled option",
			Disabled:       true,
			DisabledReason: "Demonstration option is disabled",
		},
	)
	if err != nil {
		return nil, err
	}
	selectionRadioTwo, err := transaction.NewRadioButton(
		selectionRadioGroup,
		expletives.RadioButtonOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.radio.two",
			},
			Value: "two", Label: "Option two", Mnemonic: "w",
		},
	)
	if err != nil {
		return nil, err
	}
	choiceGroup, err := transaction.NewGroupBox(
		selectionScreen,
		expletives.GroupBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.choices",
				MinimumSize:   expletives.Size{Width: 26, Height: 8},
				Style:         canvasStyle.ID,
			},
			Title:       "Cycle / Select",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderSingle,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionCycleField, err := transaction.NewCycleField(
		choiceGroup,
		expletives.CycleFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.cycle.primary",
			},
			Label: "Cycle", Mnemonic: "m",
			Options: []expletives.SelectionOption{
				{Value: "alpha", Label: "Alpha"},
				{
					Value: "disabled", Label: "Disabled",
					Disabled:       true,
					DisabledReason: "Demonstration option is disabled",
				},
				{Value: "charlie", Label: "Charlie"},
			},
			Value:         "alpha",
			ChangeCommand: CommandSelectionChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionSelectField, err := transaction.NewSelectField(
		choiceGroup,
		expletives.SelectFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.select.primary",
			},
			Label: "Select", Mnemonic: "l",
			Options: []expletives.SelectionOption{
				{Value: "low", Label: "Low"},
				{Value: "high", Label: "High"},
			},
			Value:         "low",
			ChangeCommand: CommandSelectionChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionEmptyField, err := transaction.NewCycleField(
		choiceGroup,
		expletives.CycleFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "selection.cycle.empty",
			},
			Label: "Empty", Options: []expletives.SelectionOption{},
		},
	)
	if err != nil {
		return nil, err
	}

	inputViewport, err := transaction.NewScrollablePanel(
		inputScreen,
		expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "input.viewport",
					MinimumSize:   expletives.Size{Width: 1, Height: 1},
					Style:         canvasStyle.ID,
				},
				ContentStyle: canvasStyle.ID,
				State: expletives.ViewportState{
					ContentSize: expletives.Size{
						Width: 55, Height: 9,
					},
				},
			},
			BorderForm:    expletives.BorderNone,
			HorizontalBar: expletives.ScrollBarVisibilityAuto,
			VerticalBar:   expletives.ScrollBarVisibilityAuto,
		},
	)
	if err != nil {
		return nil, err
	}
	inputForm, err := transaction.NewPanel(
		inputViewport.Content(),
		expletives.PanelOptions{
			AutomationKey: "input.form",
			MinimumSize:   expletives.Size{Width: 55, Height: 9},
			Style:         canvasStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	type inputFormRow struct {
		key   string
		panel *expletives.Panel
		label *expletives.Label
		field expletives.Control
	}
	inputRows := make([]inputFormRow, 0, 7)
	newInputRow := func(
		key string,
		height int,
		vertical expletives.LayoutSizeHint,
	) (*expletives.Panel, error) {
		return transaction.NewPanel(
			inputForm,
			expletives.PanelOptions{
				AutomationKey: "input.row." + key,
				MinimumSize: expletives.Size{
					Width: 55, Height: height,
				},
				LayoutHints: expletives.LayoutHints{
					Horizontal: expletives.LayoutSizeStretch,
					Vertical:   vertical,
				},
				Style: canvasStyle.ID,
			},
		)
	}
	addInputRow := func(
		key string,
		row *expletives.Panel,
		text string,
		mnemonic expletives.Key,
		field expletives.Control,
	) error {
		label, labelErr := transaction.NewLabel(
			row,
			expletives.LabelOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "input.label." + key,
					MinimumSize: expletives.Size{
						Width: 24, Height: 1,
					},
					Style: canvasStyle.ID,
				},
				Text:                text,
				HorizontalAlignment: expletives.TextAlignStart,
				Target:              field,
				Mnemonic:            mnemonic,
			},
		)
		if labelErr != nil {
			return labelErr
		}
		inputRows = append(inputRows, inputFormRow{
			key: key, panel: row, label: label, field: field,
		})
		return nil
	}
	plainInputRow, err := newInputRow(
		"plain",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	inputPlain, err := transaction.NewTextField(
		plainInputRow,
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.text.plain",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Text:          "Edit me",
			ChangeCommand: CommandTextChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"plain",
		plainInputRow,
		"Plain text:",
		"p",
		inputPlain,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetFocusGuidance(
		inputPlain,
		expletives.FocusGuidance{
			Mode: expletives.FocusGuidanceAppend,
			Text: "This field accepts any supported one-cell character",
		},
	); err != nil {
		return nil, err
	}

	softInputRow, err := newInputRow(
		"soft",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	inputSoft, err := transaction.NewTextField(
		softInputRow,
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.text.soft_whitelist",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Text: "ABC-123",
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationWhitelist,
				Characters:  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-",
			},
			ChangeCommand: CommandTextChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"soft",
		softInputRow,
		"Soft whitelist:",
		"f",
		inputSoft,
	); err != nil {
		return nil, err
	}

	hardInputRow, err := newInputRow(
		"hard",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	inputHard, err := transaction.NewTextField(
		hardInputRow,
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.text.hard_blacklist",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Text: "safe-name.txt",
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationHard,
				Mode:        expletives.TextValidationBlacklist,
				Characters:  `/\:*?"<>|`,
			},
			ChangeCommand: CommandTextChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"hard",
		hardInputRow,
		"Hard blacklist:",
		"b",
		inputHard,
	); err != nil {
		return nil, err
	}

	passwordInputRow, err := newInputRow(
		"password",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	inputPassword, err := transaction.NewTextField(
		passwordInputRow,
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.text.password",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Text:     "secret",
			Password: true,
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationBlacklist,
				Characters:  " ",
			},
			ChangeCommand: CommandTextChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"password",
		passwordInputRow,
		"Password phrase:",
		"h",
		inputPassword,
	); err != nil {
		return nil, err
	}

	numberInputRow, err := newInputRow(
		"number",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	numberMinimum, numberMaximum := 0.0, 20.0
	inputNumber, err := transaction.NewNumberField(
		numberInputRow,
		expletives.NumberFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.number.ranged",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Value: 12.5, Minimum: &numberMinimum, Maximum: &numberMaximum,
			DecimalPlaces: 1, ChangeCommand: CommandNumberChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"number",
		numberInputRow,
		"Number:",
		"u",
		inputNumber,
	); err != nil {
		return nil, err
	}

	spinInputRow, err := newInputRow(
		"spin",
		1,
		expletives.LayoutSizeNatural,
	)
	if err != nil {
		return nil, err
	}
	spinMinimum, spinMaximum := 0.0, 2.0
	inputSpin, err := transaction.NewSpinBox(
		spinInputRow,
		expletives.SpinBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.spin.clamped",
				MinimumSize:   expletives.Size{Width: 30, Height: 1},
			},
			Value: 1, Minimum: &spinMinimum, Maximum: &spinMaximum,
			DecimalPlaces: 1, Step: 0.5,
			ChangeCommand: CommandNumberChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"spin",
		spinInputRow,
		"Spin box:",
		"x",
		inputSpin,
	); err != nil {
		return nil, err
	}

	areaInputRow, err := newInputRow(
		"multiline",
		3,
		expletives.LayoutSizeStretch,
	)
	if err != nil {
		return nil, err
	}
	inputArea, err := transaction.NewTextArea(
		areaInputRow,
		expletives.TextAreaOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "input.text_area.multiline",
				MinimumSize:   expletives.Size{Width: 30, Height: 3},
			},
			Text: "Multiline\ntext area",
			Wrap: expletives.TextWrapWords,
			Validator: &expletives.TextValidator{
				Enforcement: expletives.TextValidationSoft,
				Mode:        expletives.TextValidationBlacklist,
				Characters:  "@",
			},
			ChangeCommand: CommandTextChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := addInputRow(
		"multiline",
		areaInputRow,
		"Multiline value:",
		"v",
		inputArea,
	); err != nil {
		return nil, err
	}

	progressGroups := make([]expletives.Control, 0, 6)
	newProgressGroup := func(key, title string) (*expletives.GroupBox, error) {
		group, groupErr := transaction.NewGroupBox(
			progressScreen,
			expletives.GroupBoxOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "progress.group." + key,
					MinimumSize: expletives.Size{
						Width: 18, Height: 5,
					},
					Style: canvasStyle.ID,
				},
				Title:       title,
				BorderStyle: borderStyle.ID,
				BorderForm:  expletives.BorderSingle,
			},
		)
		if groupErr == nil {
			progressGroups = append(progressGroups, group)
		}
		return group, groupErr
	}
	determinateGroup, err := newProgressGroup(
		"determinate",
		"ProgressBar 42%",
	)
	if err != nil {
		return nil, err
	}
	progressBar, err := transaction.NewProgressBar(
		determinateGroup,
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar.determinate",
			},
			State: expletives.ProgressBarState{
				Current: 42, Total: 100,
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	indeterminateGroup, err := newProgressGroup(
		"indeterminate",
		"Indeterminate",
	)
	if err != nil {
		return nil, err
	}
	progressIndeterminate, err := transaction.NewProgressBar(
		indeterminateGroup,
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar.indeterminate",
			},
			State: expletives.ProgressBarState{
				Indeterminate: true,
				Status:        expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	progressSpinner, err := transaction.NewSpinner(
		indeterminateGroup,
		expletives.SpinnerOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.spinner",
			},
			State: expletives.ActivityState{
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	progressDots, err := transaction.NewActivityDots(
		indeterminateGroup,
		expletives.ActivityDotsOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.activity_dots",
			},
			State: expletives.ActivityState{
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	meterGroup, err := newProgressGroup("meters", "Meters")
	if err != nil {
		return nil, err
	}
	progressMeter, err := transaction.NewMeter(
		meterGroup,
		expletives.MeterOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.meter.horizontal",
			},
			State: expletives.MeterState{
				Value: 65, Minimum: 0, Maximum: 100,
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	progressVerticalMeter, err := transaction.NewMeter(
		meterGroup,
		expletives.MeterOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.meter.vertical",
			},
			State: expletives.MeterState{
				Value: 65, Minimum: 0, Maximum: 100,
				Orientation: expletives.Vertical,
				Status:      expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	terminalGroup, err := newProgressGroup("terminal", "Terminal States")
	if err != nil {
		return nil, err
	}
	progressComplete, err := transaction.NewProgressBar(
		terminalGroup,
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar.completed",
			},
			State: expletives.ProgressBarState{
				Current: 100, Total: 100,
				Status: expletives.ProgressCompleted,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	progressFailed, err := transaction.NewProgressBar(
		terminalGroup,
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar.failed",
			},
			State: expletives.ProgressBarState{
				Current: 60, Total: 100,
				Status: expletives.ProgressFailed,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	progressCancelled, err := transaction.NewProgressBar(
		terminalGroup,
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.bar.cancelled",
			},
			State: expletives.ProgressBarState{
				Current: 25, Total: 100,
				Status: expletives.ProgressCancelled,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	reducedGroup, err := newProgressGroup("reduced", "Reduced Motion")
	if err != nil {
		return nil, err
	}
	reducedSpinner, err := transaction.NewSpinner(
		reducedGroup,
		expletives.SpinnerOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.spinner.reduced",
			},
			State: expletives.ActivityState{
				Tick: 99, ReducedMotion: true,
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	reducedDots, err := transaction.NewActivityDots(
		reducedGroup,
		expletives.ActivityDotsOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress.activity_dots.reduced",
			},
			State: expletives.ActivityState{
				Tick: 99, ReducedMotion: true,
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	actionGroup, err := newProgressGroup("actions", "Actions")
	if err != nil {
		return nil, err
	}
	progressButtons := make([]expletives.Control, 0, 3)
	for _, definition := range []struct {
		key      string
		command  expletives.CommandID
		mnemonic expletives.Key
	}{
		{"tick", CommandProgressTick, "t"},
		{"reset", CommandProgressReset, "e"},
		{"motion", CommandProgressMotion, "n"},
	} {
		button, buttonErr := transaction.NewButton(
			actionGroup,
			expletives.ButtonOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "progress.action." + definition.key,
					Style:         canvasStyle.ID,
				},
				Command:  definition.command,
				Mnemonic: definition.mnemonic,
			},
		)
		if buttonErr != nil {
			return nil, buttonErr
		}
		progressButtons = append(progressButtons, button)
	}

	navigationGroups := make([]expletives.Control, 0, 4)
	newNavigationGroup := func(
		key string,
		title string,
	) (*expletives.GroupBox, error) {
		group, groupErr := transaction.NewGroupBox(
			navigationScreen,
			expletives.GroupBoxOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "navigation.group." + key,
					MinimumSize: expletives.Size{
						Width: 20, Height: 6,
					},
					Style: canvasStyle.ID,
				},
				Title:       title,
				BorderStyle: borderStyle.ID,
				BorderForm:  expletives.BorderSingle,
			},
		)
		if groupErr == nil {
			navigationGroups = append(navigationGroups, group)
		}
		return group, groupErr
	}
	scrollBarGroup, err := newNavigationGroup("scrollbars", "ScrollBar")
	if err != nil {
		return nil, err
	}
	navigationHorizontal, err := transaction.NewScrollBar(
		scrollBarGroup,
		expletives.ScrollBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "navigation.scrollbar.horizontal",
			},
			State: expletives.ScrollBarState{
				ContentSize: 100, ViewportSize: 20, Offset: 40,
			},
			ChangeCommand: CommandNavigationChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	navigationVertical, err := transaction.NewScrollBar(
		scrollBarGroup,
		expletives.ScrollBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "navigation.scrollbar.vertical",
			},
			Orientation: expletives.Vertical,
			State: expletives.ScrollBarState{
				ContentSize: 80, ViewportSize: 16, Offset: 32,
			},
			ChangeCommand: CommandNavigationChanged,
		},
	)
	if err != nil {
		return nil, err
	}

	tabbedGroup, err := newNavigationGroup("tabs", "TabbedPanel")
	if err != nil {
		return nil, err
	}
	navigationTabs, err := transaction.NewTabbedPanel(
		tabbedGroup,
		expletives.TabbedPanelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "navigation.tabs",
				Style:         canvasStyle.ID,
			},
			BorderStyle:   borderStyle.ID,
			BorderForm:    expletives.BorderSingle,
			ChangeCommand: CommandNavigationChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	tabbedDescriptors := make([]expletives.Tab, 3)
	for index, definition := range []struct {
		key      string
		label    string
		mnemonic expletives.Key
	}{
		{"overview", "Overview", "o"},
		{"details", "Details", "d"},
		{"disabled", "Disabled", "i"},
	} {
		page, pageErr := transaction.NewPanel(
			navigationTabs,
			expletives.PanelOptions{
				AutomationKey: "navigation.tabs.page." + definition.key,
				Style:         canvasStyle.ID,
			},
		)
		if pageErr != nil {
			return nil, pageErr
		}
		tabbedDescriptors[index] = expletives.Tab{
			Key:      "navigation.tabs.tab." + definition.key,
			Value:    definition.key,
			Label:    definition.label,
			Mnemonic: definition.mnemonic,
			Page:     page,
		}
		if definition.key == "disabled" {
			tabbedDescriptors[index].Disabled = true
			tabbedDescriptors[index].DisabledReason = "Demonstrates disabled tabs"
		}
		if _, textErr := transaction.NewStaticText(
			page,
			expletives.StaticTextOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "navigation.tabs.text." + definition.key,
					Bounds: expletives.Rect{
						X: 1, Y: 1, Width: 16, Height: 2,
					},
					Style: canvasStyle.ID,
				},
				Text: "TabbedPanel page: " + definition.label,
				Wrap: expletives.TextWrapWords,
			},
		); textErr != nil {
			return nil, textErr
		}
	}
	if err := transaction.SetTabs(
		navigationTabs,
		tabbedDescriptors,
		"overview",
	); err != nil {
		return nil, err
	}

	notebookGroup, err := newNavigationGroup("notebook", "Notebook")
	if err != nil {
		return nil, err
	}
	navigationNotebook, err := transaction.NewNotebook(
		notebookGroup,
		expletives.TabbedPanelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "navigation.notebook",
				Style:         canvasStyle.ID,
			},
			BorderStyle:   borderStyle.ID,
			BorderForm:    expletives.BorderDouble,
			ChangeCommand: CommandNavigationChanged,
		},
	)
	if err != nil {
		return nil, err
	}
	notebookDescriptors := make([]expletives.Tab, 2)
	for index, definition := range []struct {
		key      string
		label    string
		mnemonic expletives.Key
	}{
		{"one", "One", "n"},
		{"two", "Two", "w"},
	} {
		page, pageErr := transaction.NewPanel(
			navigationNotebook,
			expletives.PanelOptions{
				AutomationKey: "navigation.notebook.page." + definition.key,
				Style:         canvasStyle.ID,
			},
		)
		if pageErr != nil {
			return nil, pageErr
		}
		notebookDescriptors[index] = expletives.Tab{
			Key:      "navigation.notebook.tab." + definition.key,
			Value:    definition.key,
			Label:    definition.label,
			Mnemonic: definition.mnemonic,
			Page:     page,
		}
		if _, textErr := transaction.NewStaticText(
			page,
			expletives.StaticTextOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "navigation.notebook.text." + definition.key,
					Bounds: expletives.Rect{
						X: 1, Y: 1, Width: 12, Height: 1,
					},
					Style: canvasStyle.ID,
				},
				Text: "Notebook page " + definition.label,
			},
		); textErr != nil {
			return nil, textErr
		}
	}
	if err := transaction.SetTabs(
		navigationNotebook,
		notebookDescriptors,
		"one",
	); err != nil {
		return nil, err
	}

	hotkeyBar, err := transaction.NewHotkeyBar(
		recentFooter,
		expletives.HotkeyBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.hotkeys",
				Style:         greenStyle.ID,
			},
			Items: []expletives.HotkeyBarItem{
				{Command: CommandFixtureToggle},
				{Command: CommandScenarioReset},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	rootLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.root",
			Insets: expletives.Insets{
				Right: 1, Bottom: 1, Left: 1,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := rootLayout.AddPanel(
		outer,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	outerLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.catalog"},
	)
	if err != nil {
		return nil, err
	}
	if err := outerLayout.AddPanel(
		content,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	screenControls := []struct {
		key     string
		control expletives.Control
	}{
		{"home", homeScreen},
		{"panels.core", panelsCoreScreen},
		{"panels.styles", panelStylesScreen},
		{"layouts.box", layoutBoxScreen},
		{"layouts.grid", layoutGridScreen},
		{"text", textScreen},
		{"actions", actionsScreen},
		{"selection", selectionScreen},
		{"input", inputScreen},
		{"progress", progressScreen},
		{"navigation", navigationScreen},
		{"menus", menusScreen},
		{"status", statusScreen},
		{"headers_footers", chromeScreen},
		{"about", aboutScreen},
	}
	screenLayouts := make([]*expletives.BoxLayout, 0, len(screenControls))
	for _, screen := range screenControls {
		layout, layoutErr := expletives.NewBoxLayout(
			expletives.Vertical,
			expletives.BoxLayoutOptions{
				AutomationKey: "layout.screen." + screen.key,
			},
		)
		if layoutErr != nil {
			return nil, layoutErr
		}
		if err := layout.AddPanel(
			screen.control,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
		screenLayouts = append(screenLayouts, layout)
	}
	menusLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.menus",
			Insets: expletives.Insets{
				Top: 1, Right: 2, Bottom: 1, Left: 2,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := menusLayout.AddPanel(
		menusText,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	statusLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.status",
			Insets: expletives.Insets{
				Top: 1, Right: 2, Bottom: 1, Left: 2,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := statusLayout.AddPanel(
		statusText,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	chromeLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.chrome.overview",
			Insets: expletives.Insets{
				Top: 1, Right: 2, Bottom: 1, Left: 2,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := chromeLayout.AddPanel(
		chromeText,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	primaryHeaderLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.header.primary",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, label := range []expletives.Control{
		headerProduct, headerSection, headerMode,
	} {
		if err := primaryHeaderLayout.AddPanel(
			label,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
	}
	secondaryHeaderLayout, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.header.secondary",
			Columns:       2,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, label := range []expletives.Control{
		headerGridLeft, headerGridRight,
	} {
		if err := secondaryHeaderLayout.AddPanel(
			label,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
	}
	primaryFooterLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.footer.hotkeys.global",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := primaryFooterLayout.AddPanel(
		globalHotkeys,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	recentFooterLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.footer.hotkeys.screen",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := recentFooterLayout.AddPanel(
		hotkeyBar,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	focusFooterLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.footer.guidance.focus",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := focusFooterLayout.AddPanel(
		focusGuidance,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	aboutLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.about",
			Insets: expletives.Insets{
				Top: 1, Right: 2, Bottom: 1, Left: 2,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := aboutLayout.AddPanel(
		aboutText,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}

	coreGrid, err := expletives.NewGridLayout(expletives.GridLayoutOptions{
		AutomationKey: "layout.panels.core",
		Columns:       3,
		HorizontalGap: 1,
	})
	if err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{red, accent, group} {
		if err := coreGrid.AddPanel(
			panel,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	groupLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.group"},
	)
	if err != nil {
		return nil, err
	}
	if err := groupLayout.AddPanel(
		nestedPanel,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	styleGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.panels.styles",
			Columns:       4,
			HorizontalGap: 1,
			VerticalGap:   1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, frame := range styleFrames {
		if err := styleGrid.AddPanel(
			frame,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	styleLabelLayouts := make([]*expletives.BoxLayout, 0, len(styleLabels))
	for index, label := range styleLabels {
		layout, layoutErr := expletives.NewBoxLayout(
			expletives.Vertical,
			expletives.BoxLayoutOptions{
				AutomationKey: fmt.Sprintf(
					"layout.panel.style.%d",
					index+1,
				),
			},
		)
		if layoutErr != nil {
			return nil, layoutErr
		}
		if err := layout.AddPanel(
			label,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
		styleLabelLayouts = append(styleLabelLayouts, layout)
	}
	boxRootLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.box.horizontal",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	boxColumnLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.box.vertical",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := boxRootLayout.AddPanel(
		boxLeft,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := boxRootLayout.AddLayout(
		boxColumnLayout,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{boxTop, boxBottom} {
		if err := boxColumnLayout.AddPanel(
			panel,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
	}
	boxScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.layouts.box",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{boxArrangement, layerFrame} {
		if err := boxScreenLayout.AddPanel(
			panel,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
	}
	gridLayout, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.layouts.grid",
			Columns:       3,
			HorizontalGap: 1,
			VerticalGap:   1,
			Border: expletives.BorderOptions{
				Form:  expletives.BorderShadeDark,
				Style: borderStyle.ID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	for _, panel := range gridPanels {
		if err := gridLayout.AddPanel(
			panel,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	backLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.layer.back",
			Border: expletives.BorderOptions{
				Form:  expletives.BorderShadeLight,
				Style: borderStyle.ID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := backLayout.AddPanel(
		layerBack,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	frontLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{AutomationKey: "layout.layer.front"},
	)
	if err != nil {
		return nil, err
	}
	if err := frontLayout.AddPanel(
		layerFront,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}

	textGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.text.grid",
			Columns:       2,
			HorizontalGap: 1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{textRed, textAccent} {
		if err := textGrid.AddPanel(
			panel,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	textRedLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.display.red"},
	)
	if err != nil {
		return nil, err
	}
	if err := textRedLayout.AddPanel(
		label,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := textRedLayout.AddPanel(
		separator,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	textAccentLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.display.accent"},
	)
	if err != nil {
		return nil, err
	}
	if err := textAccentLayout.AddPanel(
		staticText,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := textAccentLayout.AddPanel(
		rule,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}

	actionsGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.actions.grid",
			Columns:       2,
			HorizontalGap: 1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{actionPanel, actionPreview} {
		if err := actionsGrid.AddPanel(
			panel,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	actionLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.actions"},
	)
	if err != nil {
		return nil, err
	}
	for _, button := range []expletives.Control{
		toggleButton,
		resetButton,
		disabledButton,
		quitButton,
	} {
		if err := actionLayout.AddPanel(
			button,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	selectionGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.selection.grid",
			Columns:       3,
			HorizontalGap: 1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, group := range []expletives.Control{
		checkboxGroup,
		radioBox,
		choiceGroup,
	} {
		if err := selectionGrid.AddPanel(
			group,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	checkboxLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.selection.checkboxes",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, checkbox := range []expletives.Control{
		selectionCheckbox,
		selectionThreeState,
		selectionCheckboxDisabled,
	} {
		if err := checkboxLayout.AddPanel(
			checkbox,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	radioBoxLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.selection.radio.box",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := radioBoxLayout.AddPanel(
		selectionRadioGroup,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	radioGroupLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.selection.radio.group",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, radio := range []expletives.Control{
		selectionRadioOne,
		selectionRadioDisabled,
		selectionRadioTwo,
	} {
		if err := radioGroupLayout.AddPanel(
			radio,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	choiceLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.selection.choices",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, field := range []expletives.Control{
		selectionCycleField,
		selectionSelectField,
		selectionEmptyField,
	} {
		if err := choiceLayout.AddPanel(
			field,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	inputScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.input.screen",
			Insets:        expletives.Insets{Top: 1, Left: 2},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := inputScreenLayout.AddPanel(
		inputViewport,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	inputContentLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.input.viewport.content",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := inputContentLayout.AddPanel(
		inputForm,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	inputFormLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.input.form",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, row := range inputRows {
		if err := inputFormLayout.AddPanel(
			row.panel,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
		rowLayout, layoutErr := expletives.NewBoxLayout(
			expletives.Horizontal,
			expletives.BoxLayoutOptions{
				AutomationKey: "layout.input.row." + row.key,
				Gap:           1,
			},
		)
		if layoutErr != nil {
			return nil, layoutErr
		}
		if err := rowLayout.AddPanel(
			row.label,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
		if err := rowLayout.AddPanel(
			row.field,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
		if err := transaction.SetLayout(row.panel, rowLayout); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(inputForm, inputFormLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		inputViewport.Content(),
		inputContentLayout,
	); err != nil {
		return nil, err
	}
	progressGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.progress.grid",
			Columns:       3,
			HorizontalGap: 1,
			VerticalGap:   1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, group := range progressGroups {
		if err := progressGrid.AddPanel(
			group,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	determinateLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.determinate",
			Insets: expletives.Insets{
				Top: 1, Right: 1, Bottom: 1, Left: 1,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := determinateLayout.AddPanel(
		progressBar,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	indeterminateLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.indeterminate",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, control := range []expletives.Control{
		progressIndeterminate,
		progressSpinner,
		progressDots,
	} {
		if err := indeterminateLayout.AddPanel(
			control,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	meterLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.meters",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := meterLayout.AddPanel(
		progressMeter,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := meterLayout.AddPanel(
		progressVerticalMeter,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	terminalLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.terminal",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, control := range []expletives.Control{
		progressComplete,
		progressFailed,
		progressCancelled,
	} {
		if err := terminalLayout.AddPanel(
			control,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	reducedLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.reduced",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, control := range []expletives.Control{
		reducedSpinner,
		reducedDots,
	} {
		if err := reducedLayout.AddPanel(
			control,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	progressActionLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.progress.actions",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, button := range progressButtons {
		if err := progressActionLayout.AddPanel(
			button,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	navigationGrid, err := expletives.NewGridLayout(
		expletives.GridLayoutOptions{
			AutomationKey: "layout.navigation.grid",
			Columns:       2,
			HorizontalGap: 1,
			VerticalGap:   1,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, group := range navigationGroups {
		if err := navigationGrid.AddPanel(
			group,
			expletives.LayoutItemOptions{},
		); err != nil {
			return nil, err
		}
	}
	navigationScrollBarsLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.navigation.scrollbars",
			Gap:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := navigationScrollBarsLayout.AddPanel(
		navigationHorizontal,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	if err := navigationScrollBarsLayout.AddPanel(
		navigationVertical,
		expletives.LayoutItemOptions{
			Grow:            1,
			HorizontalAlign: expletives.AlignEnd,
		},
	); err != nil {
		return nil, err
	}
	navigationTabsLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.navigation.tabs",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := navigationTabsLayout.AddPanel(
		navigationTabs,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	navigationNotebookLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.navigation.notebook",
		},
	)
	if err != nil {
		return nil, err
	}
	if err := navigationNotebookLayout.AddPanel(
		navigationNotebook,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(app.Root(), rootLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(outer, outerLayout); err != nil {
		return nil, err
	}
	for index, layout := range screenLayouts {
		var layoutErr error
		if index == 0 {
			layoutErr = transaction.SetLayout(content, layout)
		} else {
			layoutErr = transaction.AddLayout(content, layout)
		}
		if layoutErr != nil {
			return nil, layoutErr
		}
	}
	if err := transaction.SetLayout(panelsCoreScreen, coreGrid); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(group, groupLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(panelStylesScreen, styleGrid); err != nil {
		return nil, err
	}
	for index, frame := range styleFrames {
		container, ok := frame.(expletives.Container)
		if !ok {
			return nil, errors.New("style Frame is not a Container")
		}
		if err := transaction.SetLayout(
			container,
			styleLabelLayouts[index],
		); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(
		boxArrangement,
		boxRootLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		layoutBoxScreen,
		boxScreenLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(layerFrame, backLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(layerFrame, frontLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(layoutGridScreen, gridLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(textScreen, textGrid); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(textRed, textRedLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		textAccent,
		textAccentLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		actionsScreen,
		actionsGrid,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		selectionScreen,
		selectionGrid,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		checkboxGroup,
		checkboxLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		radioBox,
		radioBoxLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		selectionRadioGroup,
		radioGroupLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		choiceGroup,
		choiceLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(inputScreen, inputScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(progressScreen, progressGrid); err != nil {
		return nil, err
	}
	for _, entry := range []struct {
		group  expletives.Container
		layout *expletives.BoxLayout
	}{
		{determinateGroup, determinateLayout},
		{indeterminateGroup, indeterminateLayout},
		{meterGroup, meterLayout},
		{terminalGroup, terminalLayout},
		{reducedGroup, reducedLayout},
		{actionGroup, progressActionLayout},
	} {
		if err := transaction.SetLayout(entry.group, entry.layout); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(
		navigationScreen,
		navigationGrid,
	); err != nil {
		return nil, err
	}
	for _, entry := range []struct {
		group  expletives.Container
		layout *expletives.BoxLayout
	}{
		{scrollBarGroup, navigationScrollBarsLayout},
		{tabbedGroup, navigationTabsLayout},
		{notebookGroup, navigationNotebookLayout},
	} {
		if err := transaction.SetLayout(entry.group, entry.layout); err != nil {
			return nil, err
		}
	}
	if err := transaction.SetLayout(menusScreen, menusLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(statusScreen, statusLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(chromeScreen, chromeLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		primaryHeader,
		primaryHeaderLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		secondaryHeader,
		secondaryHeaderLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		primaryFooter,
		primaryFooterLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		recentFooter,
		recentFooterLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(
		focusFooter,
		focusFooterLayout,
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(aboutScreen, aboutLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(actionPanel, actionLayout); err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}

	scene := &Scene{
		App:                     app,
		accent:                  accent,
		layer:                   backLayout,
		status:                  statusBar,
		headers:                 []*expletives.Header{primaryHeader, secondaryHeader},
		globalHotkeyFooter:      primaryFooter,
		screenHotkeyFooter:      recentFooter,
		focusGuidanceFooter:     focusFooter,
		selectionCheckbox:       selectionCheckbox,
		selectionThreeState:     selectionThreeState,
		selectionRadioGroup:     selectionRadioGroup,
		selectionCycleField:     selectionCycleField,
		selectionSelectField:    selectionSelectField,
		inputPlain:              inputPlain,
		inputSoft:               inputSoft,
		inputHard:               inputHard,
		inputPassword:           inputPassword,
		inputNumber:             inputNumber,
		inputSpin:               inputSpin,
		inputArea:               inputArea,
		inputViewport:           inputViewport,
		progressBar:             progressBar,
		progressIndeterminate:   progressIndeterminate,
		progressMeter:           progressMeter,
		progressVerticalMeter:   progressVerticalMeter,
		progressSpinner:         progressSpinner,
		progressDots:            progressDots,
		navigationHorizontal:    navigationHorizontal,
		navigationVertical:      navigationVertical,
		navigationTabs:          navigationTabs,
		navigationNotebook:      navigationNotebook,
		activeScreen:            CommandViewHome,
		automationEnabled:       automationEnabled,
		automationNoticeVisible: automationEnabled,
		statusVisible:           true,
		globalFooterVisible:     true,
		screenFooterVisible:     true,
		focusFooterVisible:      true,
		nextHeader:              1,
		accentControls: []expletives.Control{
			accent,
			textAccent,
			staticText,
			rule,
			actionPreview,
		},
		screens: map[expletives.CommandID]*expletives.Panel{
			CommandViewHome:        homeScreen,
			CommandViewPanelsCore:  panelsCoreScreen,
			CommandViewPanelStyles: panelStylesScreen,
			CommandViewLayoutBox:   layoutBoxScreen,
			CommandViewLayoutGrid:  layoutGridScreen,
			CommandViewText:        textScreen,
			CommandViewActions:     actionsScreen,
			CommandSelection:       selectionScreen,
			CommandTextInput:       inputScreen,
			CommandProgress:        progressScreen,
			CommandNavigation:      navigationScreen,
			CommandViewMenus:       menusScreen,
			CommandViewAbout:       aboutScreen,
		},
	}
	if err := app.SetCommandRouter(scene.routeCommand); err != nil {
		return nil, err
	}
	for _, binding := range []struct {
		chord   expletives.Chord
		command expletives.CommandID
	}{
		{
			chord: expletives.Chord{
				Key:       "r",
				Modifiers: []expletives.Key{expletives.KeyControl},
			},
			command: CommandFixtureToggle,
		},
		{chord: expletives.Chord{Key: "q"}, command: CommandAppQuit},
		{chord: expletives.Chord{Key: expletives.KeyEscape}, command: CommandAppQuit},
		{
			chord: expletives.Chord{
				Key:       "x",
				Modifiers: []expletives.Key{expletives.KeyAlt},
			},
			command: CommandAppQuit,
		},
		{
			chord: expletives.Chord{
				Key:       "c",
				Modifiers: []expletives.Key{expletives.KeyControl},
			},
			command: CommandAppInterrupt,
		},
	} {
		if err := app.BindChord(
			binding.chord,
			expletives.CommandBinding{Command: binding.command},
		); err != nil {
			return nil, err
		}
	}
	if err := scene.syncInputViewportLocked(); err != nil {
		return nil, err
	}
	return scene, nil
}

func initialCommandDefinitions(
	automationEnabled bool,
) []expletives.CommandDefinition {
	definitions := []expletives.CommandDefinition{
		toggleDefinition(false),
		automationNoticeDefinition(automationEnabled, automationEnabled),
		{
			ID: CommandScenarioReset, Label: "Reset",
			Description: "Reset the active demonstration scenario",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandUnavailable, Label: "Disabled",
			Description:    "Demonstrate disabled Action presentation",
			Enabled:        false,
			DisabledReason: "Demonstration command is disabled",
			Automation:     true,
		},
		{
			ID: CommandPanelRaise, Label: "Raise Panel",
			Description: "Raise the accent Panel among Grid Panel peers",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandPanelLower, Label: "Lower Panel",
			Description: "Lower the accent Panel among Grid Panel peers",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandLayerRaise, Label: "Raise Layer",
			Description: "Raise the red Layout layer above the green layer",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandLayerLower, Label: "Lower Layer",
			Description: "Lower the red Layout layer below the green layer",
			Enabled:     true, Automation: true,
		},
		chromeToggleDefinition(
			CommandStatusBar,
			"Status Bar",
			"Show the application Status Bar",
			true,
		),
		chromeToggleDefinition(
			CommandHeadersShow,
			"Show",
			"Show all current application Headers",
			false,
		),
		chromeMutationDefinition(
			CommandHeadersAdd,
			"Add",
			"Add one application Header",
			true,
		),
		chromeMutationDefinition(
			CommandHeadersRemoveTop,
			"Remove Highest",
			"Remove the physically highest application Header",
			true,
		),
		chromeMutationDefinition(
			CommandHeadersRemoveLow,
			"Remove Lowest",
			"Remove the physically lowest application Header",
			true,
		),
		chromeToggleDefinition(
			CommandFooterGlobalShow,
			"Global Hotkeys",
			"Show the lowest global-hotkey Footer",
			true,
		),
		chromeToggleDefinition(
			CommandFooterScreenShow,
			"Screen Hotkeys",
			"Show the current-screen hotkey Footer",
			true,
		),
		chromeToggleDefinition(
			CommandFooterFocusShow,
			"Focus Guidance",
			"Show the focused-control guidance Footer",
			true,
		),
		{
			ID: CommandSelectionChanged, Label: "Selection Changed",
			Description: "Report a user-originated Selection control change",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandTextChanged, Label: "Text Changed",
			Description: "Report a user-originated TextField commit",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandNumberChanged, Label: "Number Changed",
			Description: "Report a user-originated numeric-field commit or step",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandProgressTick, Label: "Tick",
			Description: "Advance every live Progress animation by one absolute tick",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandProgressReset, Label: "Reset",
			Description: "Restore the live Progress examples",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandProgressComplete, Label: "Complete",
			Description: "Move the live Progress examples to completed state",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandProgressFail, Label: "Fail",
			Description: "Move the live Progress examples to failed state",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandProgressCancel, Label: "Cancel",
			Description: "Move the live Progress examples to cancelled state",
			Enabled:     true, Automation: true,
		},
		chromeToggleDefinition(
			CommandProgressMotion,
			"Motion",
			"Use reduced-motion presentation for live Progress examples",
			false,
		),
		{
			ID: CommandNavigationChanged, Label: "Navigation Changed",
			Description: "Report a user-originated ScrollBar or tab selection change",
			Enabled:     true, Automation: true,
		},
		unavailableCatalogDefinition(
			CommandPanelScrollbars,
			"Panel Scroll Bars",
			"Scrolling and Content",
		),
		unavailableCatalogDefinition(
			CommandLayoutAbsolute,
			"Absolute Positioning",
			"future Layout",
		),
		unavailableCatalogDefinition(
			CommandScrolling,
			"Scrolling / Content",
			"Scrolling and Content",
		),
		unavailableCatalogDefinition(
			CommandCollections,
			"Collections",
			"Collections",
		),
		unavailableCatalogDefinition(
			CommandPanelMenu,
			"Panel Menu",
			"panel-owned Menu",
		),
		unavailableCatalogDefinition(
			CommandContextMenu,
			"Context Menu",
			"Collections and context Menu",
		),
		unavailableCatalogDefinition(
			CommandDialogMessage,
			"Message Box",
			"Modal Controls",
		),
		unavailableCatalogDefinition(
			CommandDialogConfirm,
			"Confirm Dialog",
			"Modal Controls",
		),
		unavailableCatalogDefinition(
			CommandDialogInput,
			"Input Dialog",
			"Modal Controls",
		),
		unavailableCatalogDefinition(
			CommandDialogProgress,
			"Progress Dialog",
			"Modal Controls",
		),
		{
			ID: CommandAppQuit, Label: "Quit",
			Description: "Exit the demonstration application",
			Enabled:     true, Automation: true,
		},
		{
			ID: CommandAppInterrupt, Label: "Interrupt",
			Description: "Interrupt the demonstration application",
			Enabled:     true, Automation: true,
		},
	}
	for _, screen := range catalogScreens {
		definitions = append(
			definitions,
			screenDefinition(screen.id, screen.id == CommandViewHome),
		)
	}
	return definitions
}

func unavailableCatalogDefinition(
	id expletives.CommandID,
	label string,
	phase string,
) expletives.CommandDefinition {
	return expletives.CommandDefinition{
		ID: id, Label: label,
		Description:    "Open the " + label + " demonstration",
		Enabled:        false,
		DisabledReason: "Available after the " + phase + " phase",
		Automation:     true,
	}
}

func toggleDefinition(checked bool) expletives.CommandDefinition {
	return expletives.CommandDefinition{
		ID: CommandFixtureToggle, Label: "Toggle",
		Description: "Toggle the accent Panel color",
		Enabled:     true, Checked: checked, Automation: true,
	}
}

func chromeToggleDefinition(
	id expletives.CommandID,
	label string,
	description string,
	checked bool,
) expletives.CommandDefinition {
	return expletives.CommandDefinition{
		ID: id, Label: label, Description: description,
		Enabled: true, Checked: checked, Automation: true,
	}
}

func chromeMutationDefinition(
	id expletives.CommandID,
	label string,
	description string,
	enabled bool,
) expletives.CommandDefinition {
	definition := expletives.CommandDefinition{
		ID: id, Label: label, Description: description,
		Enabled: enabled, Automation: true,
	}
	if !enabled {
		definition.DisabledReason = "No application chrome band is available"
	}
	return definition
}

func automationNoticeDefinition(
	automationEnabled bool,
	checked bool,
) expletives.CommandDefinition {
	definition := expletives.CommandDefinition{
		ID: CommandAutomationNotice, Label: "Automation Notice",
		Description: "Show the unauthenticated automation warning in the Status Bar",
		Enabled:     automationEnabled,
		Checked:     automationEnabled && checked,
		Automation:  true,
	}
	if !automationEnabled {
		definition.DisabledReason = "Automation endpoint is not enabled"
	}
	return definition
}

func screenDefinition(
	id expletives.CommandID,
	checked bool,
) expletives.CommandDefinition {
	label := catalogScreenLabel(id)
	return expletives.CommandDefinition{
		ID: id, Label: label,
		Description: "Show one purpose-specific toolkit catalog screen",
		Enabled:     true, Checked: checked, Automation: true,
	}
}

func catalogScreenLabel(id expletives.CommandID) string {
	for _, screen := range catalogScreens {
		if screen.id == id {
			return screen.label
		}
	}
	return string(id)
}

func catalogStatusSegments(
	screen expletives.CommandID,
	automationEnabled bool,
	automationNoticeVisible bool,
) []expletives.StatusSegment {
	segments := make([]expletives.StatusSegment, 0, 2)
	if automationEnabled && automationNoticeVisible {
		segments = append(segments, expletives.StatusSegment{
			Key:      "automation",
			Text:     "UNAUTHENTICATED AUTOMATION ENABLED",
			Priority: 200,
		})
	}
	return append(segments, expletives.StatusSegment{
		Key: "screen", Text: catalogScreenLabel(screen),
		Priority: 100,
	})
}

func catalogMenuItems() ([]expletives.MenuItem, error) {
	file, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.file.home", Kind: expletives.MenuItemCommand,
				Command: CommandViewHome, Mnemonic: "h",
			},
			{Key: "menu.file.separator", Kind: expletives.MenuItemSeparator},
			{
				Key:     "menu.file.automation_notice",
				Kind:    expletives.MenuItemCommand,
				Command: CommandAutomationNotice, Mnemonic: "a",
			},
			{
				Key:  "menu.file.separator.quit",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.file.quit", Kind: expletives.MenuItemCommand,
				Command: CommandAppQuit, Mnemonic: "q",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	panelStacking, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key:     "menu.stack.panel.raise",
				Kind:    expletives.MenuItemCommand,
				Command: CommandPanelRaise, Mnemonic: "r",
			},
			{
				Key:     "menu.stack.panel.lower",
				Kind:    expletives.MenuItemCommand,
				Command: CommandPanelLower, Mnemonic: "l",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	panels, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.panels.core", Kind: expletives.MenuItemCommand,
				Command: CommandViewPanelsCore, Mnemonic: "c",
			},
			{
				Key: "menu.panels.styles", Kind: expletives.MenuItemCommand,
				Command: CommandViewPanelStyles, Mnemonic: "v",
			},
			{
				Key:  "menu.panels.separator.current",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.panels.stacking", Kind: expletives.MenuItemSubmenu,
				Label: "Stacking", Mnemonic: "t", Menu: panelStacking,
			},
			{
				Key:  "menu.panels.separator.future",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.panels.scrollbars", Kind: expletives.MenuItemCommand,
				Command: CommandPanelScrollbars, Mnemonic: "s",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	layoutStacking, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key:     "menu.stack.layer.raise",
				Kind:    expletives.MenuItemCommand,
				Command: CommandLayerRaise, Mnemonic: "r",
			},
			{
				Key:     "menu.stack.layer.lower",
				Kind:    expletives.MenuItemCommand,
				Command: CommandLayerLower, Mnemonic: "l",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	layouts, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.layouts.box", Kind: expletives.MenuItemCommand,
				Command: CommandViewLayoutBox, Mnemonic: "b",
			},
			{
				Key: "menu.layouts.grid", Kind: expletives.MenuItemCommand,
				Command: CommandViewLayoutGrid, Mnemonic: "g",
			},
			{
				Key:  "menu.layouts.separator.engines",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.layouts.stacking", Kind: expletives.MenuItemSubmenu,
				Label: "Stacking", Mnemonic: "s", Menu: layoutStacking,
			},
			{
				Key:  "menu.layouts.separator.future",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.layouts.absolute", Kind: expletives.MenuItemCommand,
				Command: CommandLayoutAbsolute, Mnemonic: "a",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	headers, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key:     "menu.sections.headers.show",
				Kind:    expletives.MenuItemCommand,
				Command: CommandHeadersShow, Mnemonic: "s",
			},
			{
				Key:     "menu.sections.headers.add",
				Kind:    expletives.MenuItemCommand,
				Command: CommandHeadersAdd, Mnemonic: "a",
			},
			{
				Key:     "menu.sections.headers.remove_highest",
				Kind:    expletives.MenuItemCommand,
				Command: CommandHeadersRemoveTop, Mnemonic: "h",
			},
			{
				Key:     "menu.sections.headers.remove_lowest",
				Kind:    expletives.MenuItemCommand,
				Command: CommandHeadersRemoveLow, Mnemonic: "l",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	footers, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key:     "menu.sections.footers.global",
				Kind:    expletives.MenuItemCommand,
				Command: CommandFooterGlobalShow, Mnemonic: "g",
			},
			{
				Key:     "menu.sections.footers.screen",
				Kind:    expletives.MenuItemCommand,
				Command: CommandFooterScreenShow, Mnemonic: "s",
			},
			{
				Key:     "menu.sections.footers.focus",
				Kind:    expletives.MenuItemCommand,
				Command: CommandFooterFocusShow, Mnemonic: "f",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sections, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.sections.status", Kind: expletives.MenuItemCommand,
				Command: CommandStatusBar, Mnemonic: "s",
			},
			{
				Key:  "menu.sections.separator.chrome",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.sections.headers", Kind: expletives.MenuItemSubmenu,
				Label: "Headers", Mnemonic: "h", Menu: headers,
			},
			{
				Key: "menu.sections.footers", Kind: expletives.MenuItemSubmenu,
				Label: "Footers", Mnemonic: "f", Menu: footers,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	controls, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.controls.text", Kind: expletives.MenuItemCommand,
				Command: CommandViewText, Mnemonic: "t",
			},
			{
				Key: "menu.controls.actions", Kind: expletives.MenuItemCommand,
				Command: CommandViewActions, Mnemonic: "a",
			},
			{
				Key:  "menu.controls.separator.future",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.controls.selection", Kind: expletives.MenuItemCommand,
				Command: CommandSelection, Mnemonic: "e",
			},
			{
				Key: "menu.controls.input", Kind: expletives.MenuItemCommand,
				Command: CommandTextInput, Mnemonic: "n",
			},
			{
				Key: "menu.controls.progress", Kind: expletives.MenuItemCommand,
				Command: CommandProgress, Mnemonic: "p",
			},
			{
				Key: "menu.controls.navigation", Kind: expletives.MenuItemCommand,
				Command: CommandNavigation, Mnemonic: "v",
			},
			{
				Key: "menu.controls.scrolling", Kind: expletives.MenuItemCommand,
				Command: CommandScrolling, Mnemonic: "c",
			},
			{
				Key: "menu.controls.collections", Kind: expletives.MenuItemCommand,
				Command: CommandCollections, Mnemonic: "o",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	menus, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.menus.overview", Kind: expletives.MenuItemCommand,
				Command: CommandViewMenus, Mnemonic: "o",
			},
			{Key: "menu.menus.separator", Kind: expletives.MenuItemSeparator},
			{
				Key: "menu.menus.panel", Kind: expletives.MenuItemCommand,
				Command: CommandPanelMenu, Mnemonic: "p",
			},
			{
				Key: "menu.menus.context", Kind: expletives.MenuItemCommand,
				Command: CommandContextMenu, Mnemonic: "c",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	dialogs, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key: "menu.dialogs.message", Kind: expletives.MenuItemCommand,
				Command: CommandDialogMessage, Mnemonic: "m",
			},
			{
				Key: "menu.dialogs.confirm", Kind: expletives.MenuItemCommand,
				Command: CommandDialogConfirm, Mnemonic: "c",
			},
			{
				Key: "menu.dialogs.input", Kind: expletives.MenuItemCommand,
				Command: CommandDialogInput, Mnemonic: "i",
			},
			{
				Key: "menu.dialogs.progress", Kind: expletives.MenuItemCommand,
				Command: CommandDialogProgress, Mnemonic: "p",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	help, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{{
			Key: "menu.help.about", Kind: expletives.MenuItemCommand,
			Command: CommandViewAbout, Mnemonic: "a",
		}},
	})
	if err != nil {
		return nil, err
	}
	return []expletives.MenuItem{
		{
			Key: "menu.file", Kind: expletives.MenuItemSubmenu,
			Label: "File", Mnemonic: "i", Menu: file,
		},
		{
			Key: "menu.panels", Kind: expletives.MenuItemSubmenu,
			Label: "Panels", Mnemonic: "n", Menu: panels,
		},
		{
			Key: "menu.layouts", Kind: expletives.MenuItemSubmenu,
			Label: "Layouts", Mnemonic: "a", Menu: layouts,
		},
		{
			Key: "menu.controls", Kind: expletives.MenuItemSubmenu,
			Label: "Controls", Mnemonic: "c", Menu: controls,
		},
		{
			Key: "menu.sections", Kind: expletives.MenuItemSubmenu,
			Label: "Sections", Mnemonic: "s", Menu: sections,
		},
		{
			Key: "menu.menus", Kind: expletives.MenuItemSubmenu,
			Label: "Menus", Mnemonic: "m", Menu: menus,
		},
		{
			Key: "menu.dialogs", Kind: expletives.MenuItemSubmenu,
			Label: "Dialogs", Mnemonic: "d", Menu: dialogs,
		},
		{
			Key: "menu.help", Kind: expletives.MenuItemSubmenu,
			Label: "Help", Mnemonic: "p", Menu: help,
			Placement: expletives.MenuBarPlacementEnd,
		},
	}, nil
}

// Resize updates the surface and its root-relative catalog frame.
func (s *Scene) Resize(size expletives.Size) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.App.SetSize(size); err != nil {
		return err
	}
	return s.syncInputViewportLocked()
}

func (s *Scene) syncInputViewportLocked() error {
	if s.inputViewport == nil {
		return nil
	}
	const naturalWidth = 55
	const naturalHeight = 9
	bounds := s.inputViewport.Bounds()
	horizontalVisible := false
	verticalVisible := false
	for range 3 {
		viewportWidth := bounds.Width
		viewportHeight := bounds.Height
		if verticalVisible && viewportWidth > 0 {
			viewportWidth--
		}
		if horizontalVisible && viewportHeight > 0 {
			viewportHeight--
		}
		nextHorizontal := naturalWidth > viewportWidth &&
			bounds.Width > 0 && bounds.Height > 0
		nextVertical := naturalHeight > viewportHeight &&
			bounds.Width > 0 && bounds.Height > 0
		if nextHorizontal == horizontalVisible &&
			nextVertical == verticalVisible {
			break
		}
		horizontalVisible = nextHorizontal
		verticalVisible = nextVertical
	}
	viewportWidth := bounds.Width
	viewportHeight := bounds.Height
	if verticalVisible && viewportWidth > 0 {
		viewportWidth--
	}
	if horizontalVisible && viewportHeight > 0 {
		viewportHeight--
	}
	state := s.inputViewport.State()
	contentSize := expletives.Size{
		Width:  max(naturalWidth, viewportWidth),
		Height: max(naturalHeight, viewportHeight),
	}
	if state.ContentSize == contentSize {
		return nil
	}
	state.ContentSize = contentSize
	return s.inputViewport.SetState(state)
}

// Toggle reports whether the accent panels use the alternate style.
func (s *Scene) Toggle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toggled
}

// ActiveScreen reports the command identity of the visible catalog screen.
func (s *Scene) ActiveScreen() expletives.CommandID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeScreen
}

func (s *Scene) routeCommand(
	ctx context.Context,
	command expletives.Command,
) expletives.CommandResult {
	outcome, err := s.handleCommand(ctx, command)
	if err == nil &&
		outcome == expletives.OutcomeApplied &&
		commandChangesClientGeometry(command.ID) {
		s.mu.Lock()
		err = s.syncInputViewportLocked()
		s.mu.Unlock()
	}
	if err != nil {
		return expletives.CommandResult{
			Outcome: expletives.OutcomeFailed,
			Code:    "fixture_command_failed",
			Message: "the demonstration command failed",
			Cause:   err,
		}
	}
	return expletives.CommandResult{Outcome: outcome}
}

func commandChangesClientGeometry(command expletives.CommandID) bool {
	switch command {
	case CommandStatusBar, CommandHeadersShow, CommandHeadersAdd,
		CommandHeadersRemoveTop, CommandHeadersRemoveLow,
		CommandFooterGlobalShow, CommandFooterScreenShow,
		CommandFooterFocusShow:
		return true
	default:
		return false
	}
}

func (s *Scene) handleCommand(
	_ context.Context,
	command expletives.Command,
) (expletives.Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch command.ID {
	case CommandFixtureToggle:
		s.toggled = !s.toggled
		style := greenStyle.ID
		if s.toggled {
			style = magentaStyle.ID
		}
		transaction := s.App.NewTransaction()
		for _, control := range s.accentControls {
			if err := transaction.SetStyle(control, style); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.App.ReplaceCommand(
			toggleDefinition(s.toggled),
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		return expletives.OutcomeApplied, nil
	case CommandAutomationNotice:
		if !s.automationEnabled {
			return expletives.OutcomeRejected, nil
		}
		s.automationNoticeVisible = !s.automationNoticeVisible
		transaction := s.App.NewTransaction()
		if err := transaction.SetStatusSegments(
			s.status,
			catalogStatusSegments(
				s.activeScreen,
				s.automationEnabled,
				s.automationNoticeVisible,
			),
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.App.ReplaceCommand(automationNoticeDefinition(
			s.automationEnabled,
			s.automationNoticeVisible,
		)); err != nil {
			return expletives.OutcomeFailed, err
		}
		return expletives.OutcomeApplied, nil
	case CommandStatusBar:
		s.statusVisible = !s.statusVisible
		if err := s.status.SetVisible(s.statusVisible); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.App.ReplaceCommand(chromeToggleDefinition(
			CommandStatusBar,
			"Status Bar",
			"Show the application Status Bar",
			s.statusVisible,
		)); err != nil {
			return expletives.OutcomeFailed, err
		}
		return expletives.OutcomeApplied, nil
	case CommandHeadersShow:
		return s.toggleHeadersLocked()
	case CommandFooterGlobalShow, CommandFooterScreenShow,
		CommandFooterFocusShow:
		return s.toggleFooterLocked(command.ID)
	case CommandHeadersAdd:
		return s.addHeaderLocked()
	case CommandHeadersRemoveTop:
		return s.removeHeaderLocked(true)
	case CommandHeadersRemoveLow:
		return s.removeHeaderLocked(false)
	case CommandScenarioReset:
		resetSequence := s.App.Snapshot().Sequence
		changed := s.toggled || s.progressTick != 0 || s.progressReduced ||
			s.navigationHorizontal.State() != (expletives.ScrollBarState{
				ContentSize: 100, ViewportSize: 20, Offset: 40,
			}) ||
			s.navigationVertical.State() != (expletives.ScrollBarState{
				ContentSize: 80, ViewportSize: 16, Offset: 32,
			}) ||
			s.navigationTabs.Selected() != "overview" ||
			s.navigationNotebook.Selected() != "one"
		s.toggled = false
		s.progressTick = 0
		s.progressReduced = false
		transaction := s.App.NewTransaction()
		for _, control := range s.accentControls {
			if err := transaction.SetStyle(
				control,
				greenStyle.ID,
			); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		if err := transaction.SetCheckState(
			s.selectionCheckbox,
			expletives.CheckUnchecked,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetCheckState(
			s.selectionThreeState,
			expletives.CheckIndeterminate,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetRadioValue(
			s.selectionRadioGroup,
			"one",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetChoiceValue(
			s.selectionCycleField,
			"alpha",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetChoiceValue(
			s.selectionSelectField,
			"low",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		for _, input := range []struct {
			field *expletives.TextField
			value string
		}{
			{s.inputPlain, "Edit me"},
			{s.inputSoft, "ABC-123"},
			{s.inputHard, "safe-name.txt"},
			{s.inputPassword, "secret"},
		} {
			if err := transaction.SetText(input.field, input.value); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		if err := transaction.SetNumberValue(s.inputNumber, 12.5); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetNumberValue(s.inputSpin, 1); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetText(
			s.inputArea,
			"Multiline\ntext area",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetProgressBarState(
			s.progressBar,
			expletives.ProgressBarState{
				Current: 42, Total: 100,
				Status: expletives.ProgressRunning,
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetProgressBarState(
			s.progressIndeterminate,
			expletives.ProgressBarState{
				Indeterminate: true,
				Status:        expletives.ProgressRunning,
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		for _, meter := range []*expletives.Meter{
			s.progressMeter,
			s.progressVerticalMeter,
		} {
			orientation := expletives.Horizontal
			if meter == s.progressVerticalMeter {
				orientation = expletives.Vertical
			}
			if err := transaction.SetMeterState(
				meter,
				expletives.MeterState{
					Value: 65, Minimum: 0, Maximum: 100,
					Orientation: orientation,
					Status:      expletives.ProgressRunning,
				},
			); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		for _, activity := range []expletives.Control{
			s.progressSpinner,
			s.progressDots,
		} {
			if err := transaction.SetActivityState(
				activity,
				expletives.ActivityState{
					Status: expletives.ProgressRunning,
				},
			); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		if err := transaction.SetScrollBarState(
			s.navigationHorizontal,
			expletives.ScrollBarState{
				ContentSize: 100, ViewportSize: 20, Offset: 40,
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetScrollBarState(
			s.navigationVertical,
			expletives.ScrollBarState{
				ContentSize: 80, ViewportSize: 16, Offset: 32,
			},
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetSelectedTab(
			s.navigationTabs,
			"overview",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.SetSelectedTab(
			s.navigationNotebook,
			"one",
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return expletives.OutcomeFailed, err
		}
		changed = changed || s.App.Snapshot().Sequence != resetSequence
		if err := s.App.ReplaceCommand(
			toggleDefinition(false),
		); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.App.ReplaceCommand(chromeToggleDefinition(
			CommandProgressMotion,
			"Motion",
			"Use reduced-motion presentation for live Progress examples",
			false,
		)); err != nil {
			return expletives.OutcomeFailed, err
		}
		before := s.App.Snapshot().Sequence
		if err := s.accent.Raise(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.layer.Lower(); err != nil {
			return expletives.OutcomeFailed, err
		}
		changed = changed || s.App.Snapshot().Sequence != before
		if !changed {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandViewHome, CommandViewPanelsCore, CommandViewPanelStyles,
		CommandViewLayoutBox, CommandViewLayoutGrid, CommandViewText,
		CommandViewActions, CommandSelection, CommandTextInput, CommandProgress,
		CommandNavigation, CommandViewMenus, CommandViewAbout:
		return s.switchScreenLocked(command.ID)
	case CommandProgressTick:
		return s.progressTickLocked()
	case CommandProgressReset:
		return s.resetProgressLocked()
	case CommandProgressComplete:
		return s.setProgressStatusLocked(expletives.ProgressCompleted)
	case CommandProgressFail:
		return s.setProgressStatusLocked(expletives.ProgressFailed)
	case CommandProgressCancel:
		return s.setProgressStatusLocked(expletives.ProgressCancelled)
	case CommandProgressMotion:
		return s.toggleProgressMotionLocked()
	case CommandSelectionChanged, CommandTextChanged, CommandNumberChanged,
		CommandNavigationChanged:
		return expletives.OutcomeApplied, nil
	case CommandPanelRaise:
		return s.showAndMutateLocked(
			CommandViewPanelsCore,
			s.accent.Raise,
		)
	case CommandPanelLower:
		return s.showAndMutateLocked(
			CommandViewPanelsCore,
			s.accent.Lower,
		)
	case CommandLayerRaise:
		return s.showAndMutateLocked(
			CommandViewLayoutBox,
			s.layer.Raise,
		)
	case CommandLayerLower:
		return s.showAndMutateLocked(
			CommandViewLayoutBox,
			s.layer.Lower,
		)
	case CommandAppQuit:
		return expletives.OutcomeExited, nil
	case CommandAppInterrupt:
		return expletives.OutcomeInterrupted, nil
	default:
		return expletives.OutcomeRejected, nil
	}
}

func (s *Scene) progressTickLocked() (expletives.Outcome, error) {
	nextTick := s.progressTick + 1
	return s.showAndMutateLocked(CommandProgress, func() error {
		bar := s.progressBar.State()
		if bar.Current < bar.Total {
			bar.Current = min(bar.Total, bar.Current+7)
		}
		bar.Status = expletives.ProgressRunning
		bar.ReducedMotion = s.progressReduced
		indeterminate := expletives.ProgressBarState{
			Indeterminate: true,
			Tick:          nextTick,
			ReducedMotion: s.progressReduced,
			Status:        expletives.ProgressRunning,
		}
		meter := s.progressMeter.State()
		meter.Value += 5
		if meter.Value > meter.Maximum {
			meter.Value = meter.Minimum
		}
		meter.Status = expletives.ProgressRunning
		vertical := s.progressVerticalMeter.State()
		vertical.Value = meter.Value
		vertical.Status = expletives.ProgressRunning
		activity := expletives.ActivityState{
			Tick:          nextTick,
			ReducedMotion: s.progressReduced,
			Status:        expletives.ProgressRunning,
		}
		transaction := s.App.NewTransaction()
		if err := transaction.SetProgressBarState(
			s.progressBar,
			bar,
		); err != nil {
			return err
		}
		if err := transaction.SetProgressBarState(
			s.progressIndeterminate,
			indeterminate,
		); err != nil {
			return err
		}
		if err := transaction.SetMeterState(
			s.progressMeter,
			meter,
		); err != nil {
			return err
		}
		if err := transaction.SetMeterState(
			s.progressVerticalMeter,
			vertical,
		); err != nil {
			return err
		}
		for _, control := range []expletives.Control{
			s.progressSpinner,
			s.progressDots,
		} {
			if err := transaction.SetActivityState(
				control,
				activity,
			); err != nil {
				return err
			}
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return err
		}
		s.progressTick = nextTick
		return nil
	})
}

func (s *Scene) resetProgressLocked() (expletives.Outcome, error) {
	return s.showAndMutateLocked(CommandProgress, func() error {
		transaction := s.App.NewTransaction()
		if err := transaction.SetProgressBarState(
			s.progressBar,
			expletives.ProgressBarState{
				Current: 42, Total: 100,
				Status: expletives.ProgressRunning,
			},
		); err != nil {
			return err
		}
		if err := transaction.SetProgressBarState(
			s.progressIndeterminate,
			expletives.ProgressBarState{
				Indeterminate: true,
				Status:        expletives.ProgressRunning,
			},
		); err != nil {
			return err
		}
		for _, entry := range []struct {
			meter       *expletives.Meter
			orientation expletives.Orientation
		}{
			{s.progressMeter, expletives.Horizontal},
			{s.progressVerticalMeter, expletives.Vertical},
		} {
			if err := transaction.SetMeterState(
				entry.meter,
				expletives.MeterState{
					Value: 65, Minimum: 0, Maximum: 100,
					Orientation: entry.orientation,
					Status:      expletives.ProgressRunning,
				},
			); err != nil {
				return err
			}
		}
		for _, control := range []expletives.Control{
			s.progressSpinner,
			s.progressDots,
		} {
			if err := transaction.SetActivityState(
				control,
				expletives.ActivityState{
					Status: expletives.ProgressRunning,
				},
			); err != nil {
				return err
			}
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return err
		}
		s.progressTick = 0
		s.progressReduced = false
		return s.App.ReplaceCommand(chromeToggleDefinition(
			CommandProgressMotion,
			"Motion",
			"Use reduced-motion presentation for live Progress examples",
			false,
		))
	})
}

func (s *Scene) setProgressStatusLocked(
	status expletives.ProgressStatus,
) (expletives.Outcome, error) {
	return s.showAndMutateLocked(CommandProgress, func() error {
		bar := s.progressBar.State()
		if status == expletives.ProgressCompleted {
			bar.Current = bar.Total
		}
		bar.Status = status
		bar.ReducedMotion = s.progressReduced
		indeterminate := s.progressIndeterminate.State()
		indeterminate.Status = status
		indeterminate.ReducedMotion = s.progressReduced
		meter := s.progressMeter.State()
		meter.Status = status
		vertical := s.progressVerticalMeter.State()
		vertical.Status = status
		activity := expletives.ActivityState{
			Tick:          s.progressTick,
			ReducedMotion: s.progressReduced,
			Status:        status,
		}
		transaction := s.App.NewTransaction()
		if err := transaction.SetProgressBarState(
			s.progressBar,
			bar,
		); err != nil {
			return err
		}
		if err := transaction.SetProgressBarState(
			s.progressIndeterminate,
			indeterminate,
		); err != nil {
			return err
		}
		if err := transaction.SetMeterState(
			s.progressMeter,
			meter,
		); err != nil {
			return err
		}
		if err := transaction.SetMeterState(
			s.progressVerticalMeter,
			vertical,
		); err != nil {
			return err
		}
		for _, control := range []expletives.Control{
			s.progressSpinner,
			s.progressDots,
		} {
			if err := transaction.SetActivityState(
				control,
				activity,
			); err != nil {
				return err
			}
		}
		return transaction.Commit(context.Background())
	})
}

func (s *Scene) toggleProgressMotionLocked() (expletives.Outcome, error) {
	next := !s.progressReduced
	return s.showAndMutateLocked(CommandProgress, func() error {
		indeterminate := s.progressIndeterminate.State()
		indeterminate.ReducedMotion = next
		indeterminate.Tick = s.progressTick
		spinner := s.progressSpinner.State()
		spinner.ReducedMotion = next
		spinner.Tick = s.progressTick
		dots := s.progressDots.State()
		dots.ReducedMotion = next
		dots.Tick = s.progressTick
		transaction := s.App.NewTransaction()
		if err := transaction.SetProgressBarState(
			s.progressIndeterminate,
			indeterminate,
		); err != nil {
			return err
		}
		if err := transaction.SetActivityState(
			s.progressSpinner,
			spinner,
		); err != nil {
			return err
		}
		if err := transaction.SetActivityState(
			s.progressDots,
			dots,
		); err != nil {
			return err
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return err
		}
		s.progressReduced = next
		return s.App.ReplaceCommand(chromeToggleDefinition(
			CommandProgressMotion,
			"Motion",
			"Use reduced-motion presentation for live Progress examples",
			next,
		))
	})
}

func (s *Scene) switchScreenLocked(
	target expletives.CommandID,
) (expletives.Outcome, error) {
	if s.activeScreen == target {
		return expletives.OutcomeNoOp, nil
	}
	previous := s.activeScreen
	transaction := s.App.NewTransaction()
	for command, screen := range s.screens {
		if err := transaction.SetVisible(
			screen,
			command == target,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	if err := transaction.SetStatusSegments(
		s.status,
		catalogStatusSegments(
			target,
			s.automationEnabled,
			s.automationNoticeVisible,
		),
	); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return expletives.OutcomeFailed, err
	}
	s.activeScreen = target
	for _, selection := range []struct {
		id      expletives.CommandID
		checked bool
	}{
		{id: previous, checked: false},
		{id: target, checked: true},
	} {
		if err := s.App.ReplaceCommand(
			screenDefinition(selection.id, selection.checked),
		); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) toggleHeadersLocked() (expletives.Outcome, error) {
	visible := !s.headersVisible
	controls := make([]expletives.Control, len(s.headers))
	for index, header := range s.headers {
		controls[index] = header
	}
	transaction := s.App.NewTransaction()
	for _, control := range controls {
		if err := transaction.SetVisible(control, visible); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return expletives.OutcomeFailed, err
	}
	s.headersVisible = visible
	if err := s.App.ReplaceCommand(chromeToggleDefinition(
		CommandHeadersShow,
		"Show",
		"Show all current application Headers",
		visible,
	)); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) toggleFooterLocked(
	command expletives.CommandID,
) (expletives.Outcome, error) {
	var footer *expletives.Footer
	var visible *bool
	label, description := "", ""
	switch command {
	case CommandFooterGlobalShow:
		footer = s.globalHotkeyFooter
		visible = &s.globalFooterVisible
		label = "Global Hotkeys"
		description = "Show the lowest global-hotkey Footer"
	case CommandFooterScreenShow:
		footer = s.screenHotkeyFooter
		visible = &s.screenFooterVisible
		label = "Screen Hotkeys"
		description = "Show the current-screen hotkey Footer"
	case CommandFooterFocusShow:
		footer = s.focusGuidanceFooter
		visible = &s.focusFooterVisible
		label = "Focus Guidance"
		description = "Show the focused-control guidance Footer"
	default:
		return expletives.OutcomeRejected, nil
	}
	next := !*visible
	if err := footer.SetVisible(next); err != nil {
		return expletives.OutcomeFailed, err
	}
	*visible = next
	if err := s.App.ReplaceCommand(chromeToggleDefinition(
		command,
		label,
		description,
		next,
	)); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) addHeaderLocked() (expletives.Outcome, error) {
	transaction := s.App.NewTransaction()
	var band expletives.Container
	var addedHeader *expletives.Header
	var labelText string
	index := s.nextHeader
	s.nextHeader++
	header, err := transaction.NewHeader(
		s.App.Root(),
		expletives.HeaderOptions{PanelOptions: expletives.PanelOptions{
			AutomationKey: fmt.Sprintf("header.dynamic.%d", index),
			Style:         headerStyle.ID,
			Hidden:        !s.headersVisible,
		}},
	)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	band = header
	addedHeader = header
	labelText = fmt.Sprintf("Added Header %d", index)
	label, err := transaction.NewLabel(
		band,
		expletives.LabelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: band.AutomationKey() + ".label",
				Style:         band.Style(),
			},
			Text:                labelText,
			HorizontalAlignment: expletives.TextAlignCenter,
		},
	)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	layout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout." + band.AutomationKey(),
		},
	)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := layout.AddPanel(
		label,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.SetLayout(band, layout); err != nil {
		return expletives.OutcomeFailed, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return expletives.OutcomeFailed, err
	}
	s.headers = append(s.headers, addedHeader)
	if err := s.replaceHeaderRemoveDefinitionsLocked(true); err != nil {
		return expletives.OutcomeFailed, err
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) removeHeaderLocked(
	highest bool,
) (expletives.Outcome, error) {
	if len(s.headers) == 0 {
		return expletives.OutcomeNoOp, nil
	}
	index := len(s.headers) - 1
	if highest {
		index = 0
	}
	control := s.headers[index]
	if err := control.Destroy(); err != nil {
		return expletives.OutcomeFailed, err
	}
	s.headers = append(s.headers[:index], s.headers[index+1:]...)
	if len(s.headers) == 0 {
		if err := s.replaceHeaderRemoveDefinitionsLocked(false); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	return expletives.OutcomeApplied, nil
}

func (s *Scene) replaceHeaderRemoveDefinitionsLocked(
	enabled bool,
) error {
	for _, definition := range []expletives.CommandDefinition{
		chromeMutationDefinition(
			CommandHeadersRemoveTop,
			"Remove Highest",
			"Remove the physically highest application Header",
			enabled,
		),
		chromeMutationDefinition(
			CommandHeadersRemoveLow,
			"Remove Lowest",
			"Remove the physically lowest application Header",
			enabled,
		),
	} {
		if err := s.App.ReplaceCommand(definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scene) showAndMutateLocked(
	target expletives.CommandID,
	mutate func() error,
) (expletives.Outcome, error) {
	navigation, err := s.switchScreenLocked(target)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	mutation, err := mutationOutcome(s.App, mutate)
	if err != nil {
		return expletives.OutcomeFailed, err
	}
	if navigation == expletives.OutcomeApplied ||
		mutation == expletives.OutcomeApplied {
		return expletives.OutcomeApplied, nil
	}
	return expletives.OutcomeNoOp, nil
}

func mutationOutcome(
	app *expletives.App,
	mutate func() error,
) (expletives.Outcome, error) {
	before := app.Snapshot().Sequence
	if err := mutate(); err != nil {
		return expletives.OutcomeFailed, err
	}
	if app.Snapshot().Sequence == before {
		return expletives.OutcomeNoOp, nil
	}
	return expletives.OutcomeApplied, nil
}

// SelfCheck validates catalog visibility, typed Menu evidence, geometry, and
// Layout common-mode stacking through the public API.
func SelfCheck() error {
	scene, err := New(expletives.Size{Width: 64, Height: 20}, "")
	if err != nil {
		return err
	}
	snapshot := scene.App.Snapshot()
	if snapshot.Version != expletives.SnapshotVersion {
		return errors.New("unexpected snapshot version")
	}
	indexControls := func(
		current expletives.Snapshot,
	) map[string]expletives.ControlSnapshot {
		indexed := make(map[string]expletives.ControlSnapshot)
		for _, control := range current.Controls {
			indexed[control.Key] = control
		}
		return indexed
	}
	controls := indexControls(snapshot)
	for _, key := range []string{
		"menu.main",
		"screen.home",
		"screen.panels.core",
		"screen.panels.styles",
		"screen.layouts.box",
		"screen.layouts.grid",
		"screen.text",
		"screen.actions",
		"screen.selection",
		"screen.input",
		"screen.progress",
		"screen.navigation",
		"screen.menus",
		"screen.status",
		"screen.headers_footers",
		"screen.about",
		"menus.overview",
		"status.main",
		"status.overview",
		"chrome.overview",
		"header.primary",
		"header.secondary",
		"footer.hotkeys.global",
		"footer.hotkeys.screen",
		"footer.guidance.focus",
		"header.product",
		"header.grid.left",
		"footer.hotkeys.global.items",
		"footer.guidance.focus.text",
		"help.about",
		"panel.red",
		"panel.accent",
		"panel.style.none",
		"panel.style.single",
		"panel.style.double",
		"panel.style.shade-light",
		"panel.style.shade-medium",
		"panel.style.shade-dark",
		"panel.style.block",
		"layout.box.left",
		"layout.box.top",
		"layout.box.bottom",
		"layout.grid.cell.1",
		"layout.grid.cell.6",
		"display.label",
		"display.static_text",
		"action.toggle",
		"action.hotkeys",
		"selection.checkbox.two_state",
		"selection.checkbox.three_state",
		"selection.checkbox.disabled",
		"selection.radio.group",
		"selection.radio.one",
		"selection.radio.disabled",
		"selection.radio.two",
		"selection.cycle.primary",
		"selection.select.primary",
		"selection.cycle.empty",
		"input.form",
		"input.row.plain",
		"input.row.soft",
		"input.row.hard",
		"input.row.password",
		"input.row.number",
		"input.row.spin",
		"input.row.multiline",
		"input.label.plain",
		"input.label.soft",
		"input.label.hard",
		"input.label.password",
		"input.label.number",
		"input.label.spin",
		"input.label.multiline",
		"input.text.plain",
		"input.text.soft_whitelist",
		"input.text.hard_blacklist",
		"input.text.password",
		"input.number.ranged",
		"input.spin.clamped",
		"input.text_area.multiline",
		"progress.group.determinate",
		"progress.group.indeterminate",
		"progress.group.meters",
		"progress.group.terminal",
		"progress.group.reduced",
		"progress.group.actions",
		"progress.bar.determinate",
		"progress.bar.indeterminate",
		"progress.meter.horizontal",
		"progress.meter.vertical",
		"progress.spinner",
		"progress.activity_dots",
		"progress.bar.completed",
		"progress.bar.failed",
		"progress.bar.cancelled",
		"progress.spinner.reduced",
		"progress.activity_dots.reduced",
		"progress.action.tick",
		"progress.action.reset",
		"progress.action.motion",
		"navigation.group.scrollbars",
		"navigation.group.tabs",
		"navigation.group.notebook",
		"navigation.scrollbar.horizontal",
		"navigation.scrollbar.vertical",
		"navigation.tabs",
		"navigation.tabs.page.overview",
		"navigation.tabs.page.details",
		"navigation.tabs.page.disabled",
		"navigation.notebook",
		"navigation.notebook.page.one",
		"navigation.notebook.page.two",
		"layer.back",
		"layer.front",
	} {
		if _, exists := controls[key]; !exists {
			return fmt.Errorf("catalog control %q is absent", key)
		}
	}
	menu := controls["menu.main"].Details.MenuBar
	if menu == nil || len(menu.Entries) != 58 ||
		len(menu.OpenPath) != 0 {
		return errors.New("MenuBar typed evidence is incomplete")
	}
	helpAtEnd := false
	expectedRootMnemonics := map[string]expletives.Key{
		"menu.file":     "i",
		"menu.panels":   "n",
		"menu.layouts":  "a",
		"menu.controls": "c",
		"menu.sections": "s",
		"menu.menus":    "m",
		"menu.dialogs":  "d",
		"menu.help":     "p",
	}
	seenRootMnemonics := make(map[string]bool, len(expectedRootMnemonics))
	catalogLabels := map[string]string{
		"menu.file.home":           "Home",
		"menu.panels.core":         "Core Panels",
		"menu.panels.styles":       "Visual Styles",
		"menu.layouts.box":         "Box Layout",
		"menu.layouts.grid":        "Grid Layout",
		"menu.controls.selection":  "Selection",
		"menu.controls.input":      "Text / Numeric Input",
		"menu.controls.progress":   "Progress",
		"menu.controls.navigation": "Navigation",
	}
	seenCatalogLabels := make(map[string]bool, len(catalogLabels))
	homeChecked := false
	automationNoticeDisabled := false
	statusChecked := false
	headersShowChecked := true
	footerToggleCount := 0
	for _, entry := range menu.Entries {
		if entry.Key == "menu.help" &&
			entry.Placement == expletives.MenuBarPlacementEnd {
			helpAtEnd = true
		}
		if mnemonic, ok := expectedRootMnemonics[entry.Key]; ok {
			seenRootMnemonics[entry.Key] = entry.Mnemonic == mnemonic
		}
		if label, ok := catalogLabels[entry.Key]; ok {
			if entry.Label != label {
				return fmt.Errorf(
					"%s label = %q, want %q",
					entry.Key,
					entry.Label,
					label,
				)
			}
			seenCatalogLabels[entry.Key] = true
		}
		if entry.Key == "menu.file.home" {
			homeChecked = entry.Checked
		}
		if entry.Key == "menu.file.automation_notice" {
			automationNoticeDisabled = !entry.Enabled && !entry.Checked
		}
		switch entry.Key {
		case "menu.sections.status":
			statusChecked = entry.Enabled && entry.Checked
		case "menu.sections.headers.show":
			headersShowChecked = entry.Checked
		case "menu.sections.footers.global",
			"menu.sections.footers.screen",
			"menu.sections.footers.focus":
			if entry.Checked {
				footerToggleCount++
			}
		}
	}
	if !helpAtEnd {
		return errors.New("Help menu is not in the end-aligned group")
	}
	if len(seenRootMnemonics) != len(expectedRootMnemonics) {
		return errors.New("catalog root mnemonics are incomplete")
	}
	for key, valid := range seenRootMnemonics {
		if !valid {
			return fmt.Errorf("catalog root mnemonic %q is not collision-audited", key)
		}
	}
	if len(seenCatalogLabels) != len(catalogLabels) || !homeChecked ||
		!automationNoticeDisabled || !statusChecked ||
		headersShowChecked || footerToggleCount != 3 {
		return errors.New("catalog navigation labels are incomplete")
	}
	status := controls["status.main"]
	if status.Details.StatusBar == nil ||
		status.Bounds != (expletives.Rect{
			Y: 19, Width: 64, Height: 1,
		}) ||
		len(status.Details.StatusBar.Segments) != 1 ||
		status.Details.StatusBar.Segments[0].Label != "Home" {
		return errors.New("StatusBar typed evidence is incomplete")
	}
	for column := range snapshot.Frame.Size.Width {
		cell, ok := snapshot.Frame.Cell(column, snapshot.Frame.Size.Height-1)
		if !ok || cell.Owner != status.ID {
			return errors.New("StatusBar does not own the complete bottom row")
		}
	}
	if !controls["screen.home"].Visible ||
		controls["screen.panels.core"].Visible ||
		controls["screen.panels.styles"].Visible ||
		controls["screen.layouts.box"].Visible ||
		controls["screen.layouts.grid"].Visible ||
		controls["screen.text"].Visible ||
		controls["screen.actions"].Visible ||
		controls["screen.selection"].Visible ||
		controls["screen.input"].Visible ||
		controls["screen.progress"].Visible ||
		controls["screen.navigation"].Visible ||
		controls["screen.menus"].Visible ||
		controls["screen.status"].Visible ||
		controls["screen.headers_footers"].Visible ||
		controls["screen.about"].Visible {
		return errors.New("initial catalog screen visibility is invalid")
	}
	home := controls["screen.home"]
	if len(home.Children) != 0 {
		return errors.New("Home screen is not empty")
	}
	homePoint := expletives.Point{
		X: home.AbsoluteBounds.X + home.AbsoluteBounds.Width/2,
		Y: home.AbsoluteBounds.Y + home.AbsoluteBounds.Height/2,
	}
	homeCell, ok := snapshot.Frame.Cell(homePoint.X, homePoint.Y)
	if !ok || homeCell.Owner != home.ID ||
		homeCell.Style != canvasStyle.ID ||
		homeCell.Background != canvasStyle.Background {
		return errors.New("Home screen does not paint the Turbo Vision-blue canvas")
	}
	invoke := func(
		request string,
		command expletives.CommandID,
	) error {
		completion, invokeErr := scene.App.InvokeCommand(
			context.Background(),
			"self-check",
			request,
			command,
			"",
		)
		if invokeErr != nil ||
			completion.Outcome != expletives.OutcomeApplied {
			return fmt.Errorf(
				"%s: %v (%s)",
				command,
				invokeErr,
				completion.Outcome,
			)
		}
		snapshot = scene.App.Snapshot()
		controls = indexControls(snapshot)
		return nil
	}
	if err := invoke("show-panels", CommandViewPanelsCore); err != nil {
		return err
	}
	for _, key := range []string{"panel.red", "panel.accent"} {
		control := controls[key]
		point := expletives.Point{
			X: control.AbsoluteBounds.X + control.AbsoluteBounds.Width/2,
			Y: control.AbsoluteBounds.Y + control.AbsoluteBounds.Height/2,
		}
		cell, ok := snapshot.Frame.Cell(point.X, point.Y)
		if !ok || cell.Owner != control.ID {
			return fmt.Errorf("%q does not paint its arranged interior", key)
		}
	}
	if err := invoke("show-styles", CommandViewPanelStyles); err != nil {
		return err
	}
	for key, form := range map[string]expletives.BorderForm{
		"panel.style.none":         expletives.BorderNone,
		"panel.style.single":       expletives.BorderSingle,
		"panel.style.double":       expletives.BorderDouble,
		"panel.style.shade-light":  expletives.BorderShadeLight,
		"panel.style.shade-medium": expletives.BorderShadeMedium,
		"panel.style.shade-dark":   expletives.BorderShadeDark,
		"panel.style.block":        expletives.BorderBlock,
	} {
		border := controls[key].Details.Border
		if border == nil || border.Form != form || !controls[key].Visible {
			return fmt.Errorf("%q style fixture is incomplete", key)
		}
	}
	if err := invoke("show-grid", CommandViewLayoutGrid); err != nil {
		return err
	}
	gridCell := controls["layout.grid.cell.1"]
	gridPoint := expletives.Point{
		X: gridCell.AbsoluteBounds.X + gridCell.AbsoluteBounds.Width/2,
		Y: gridCell.AbsoluteBounds.Y + gridCell.AbsoluteBounds.Height/2,
	}
	cell, ok := snapshot.Frame.Cell(gridPoint.X, gridPoint.Y)
	if !ok || cell.Owner != gridCell.ID {
		return errors.New("Grid Layout page does not paint its arranged cells")
	}
	if err := invoke("show-box", CommandViewLayoutBox); err != nil {
		return err
	}
	front := controls["layer.front"]
	point := expletives.Point{
		X: front.AbsoluteBounds.X + front.AbsoluteBounds.Width/2,
		Y: front.AbsoluteBounds.Y + front.AbsoluteBounds.Height/2,
	}
	cell, ok = snapshot.Frame.Cell(point.X, point.Y)
	if !ok || cell.Owner != front.ID {
		return errors.New("front Layout does not initially paint above back")
	}
	if err := invoke("raise-layout", CommandLayerRaise); err != nil {
		return err
	}
	cell, ok = snapshot.Frame.Cell(point.X, point.Y)
	if !ok || cell.Owner != controls["layer.back"].ID {
		return errors.New("Layout Raise did not move the complete red layer")
	}
	if err := invoke("show-actions", CommandViewActions); err != nil {
		return err
	}
	if controls["screen.home"].Visible ||
		!controls["screen.actions"].Visible ||
		!controls["action.toggle"].Focused {
		return errors.New("Actions screen did not become visible and focused")
	}
	if err := invoke("show-selection", CommandSelection); err != nil {
		return err
	}
	if !controls["screen.selection"].Visible ||
		!controls["selection.checkbox.two_state"].Focused ||
		controls["selection.checkbox.two_state"].Details.Checkbox == nil ||
		controls["selection.radio.group"].Details.RadioGroup == nil ||
		controls["selection.cycle.primary"].Details.ChoiceField == nil ||
		controls["selection.select.primary"].Kind !=
			expletives.ControlSelectField {
		return errors.New("Selection screen typed evidence is incomplete")
	}
	guide := controls["footer.guidance.focus.text"].Details.FocusGuideBar
	if guide == nil ||
		guide.Target != controls["selection.checkbox.two_state"].ID ||
		guide.TargetKind != expletives.ControlCheckbox ||
		guide.Customization != expletives.FocusGuidanceAppend {
		return errors.New("focused-control guidance did not follow Selection focus")
	}
	selectionCompletion, dispatchErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"selection-space",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeySpace,
		},
	)
	if dispatchErr != nil ||
		selectionCompletion.Command != CommandSelectionChanged {
		return fmt.Errorf(
			"Selection raw Space dispatch = %+v, %v",
			selectionCompletion,
			dispatchErr,
		)
	}
	snapshot = scene.App.Snapshot()
	controls = indexControls(snapshot)
	if controls["selection.checkbox.two_state"].Details.Checkbox.State !=
		expletives.CheckChecked {
		return errors.New("Selection raw Space did not check the Checkbox")
	}
	if err := invoke("show-input", CommandTextInput); err != nil {
		return err
	}
	plainDetails := controls["input.text.plain"].Details.TextField
	softDetails := controls["input.text.soft_whitelist"].Details.TextField
	hardDetails := controls["input.text.hard_blacklist"].Details.TextField
	passwordDetails := controls["input.text.password"].Details.TextField
	numberDetails := controls["input.number.ranged"].Details.NumberField
	spinDetails := controls["input.spin.clamped"].Details.NumberField
	areaDetails := controls["input.text_area.multiline"].Details.TextArea
	if !controls["screen.input"].Visible ||
		!controls["input.text.plain"].Focused ||
		plainDetails == nil || plainDetails.Text != "Edit me" ||
		softDetails == nil || softDetails.Validator == nil ||
		softDetails.Validator.Enforcement != expletives.TextValidationSoft ||
		hardDetails == nil || hardDetails.Validator == nil ||
		hardDetails.Validator.Enforcement != expletives.TextValidationHard ||
		passwordDetails == nil || !passwordDetails.Password ||
		!passwordDetails.Redacted || passwordDetails.Text != "" ||
		numberDetails == nil || numberDetails.Value != 12.5 ||
		numberDetails.Minimum == nil || *numberDetails.Minimum != 0 ||
		numberDetails.Maximum == nil || *numberDetails.Maximum != 20 ||
		numberDetails.Step != 0 ||
		spinDetails == nil || spinDetails.Value != 1 ||
		spinDetails.Step != 0.5 ||
		areaDetails == nil || areaDetails.Text != "Multiline\ntext area" ||
		areaDetails.LineCount != 2 ||
		areaDetails.Wrap != expletives.TextWrapWords {
		return errors.New("Input catalog typed evidence is incomplete")
	}
	if completion, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"input-edit",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyEnter,
		},
	); inputErr != nil ||
		completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf(
			"TextField edit dispatch = %+v, %v; overflows=%+v",
			completion,
			inputErr,
			scene.App.Snapshot().Overflows,
		)
	}
	if completion, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"input-type",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  "!",
		},
	); inputErr != nil ||
		completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf(
			"TextField printable dispatch = %+v, %v; field=%+v overflows=%+v",
			completion,
			inputErr,
			indexControls(scene.App.Snapshot())["input.text.plain"].Details.TextField,
			scene.App.Snapshot().Overflows,
		)
	}
	if completion, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"input-commit",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyEnter,
		},
	); inputErr != nil ||
		completion.Command != CommandTextChanged ||
		completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf("TextField commit dispatch = %+v, %v", completion, inputErr)
	}
	if err := scene.inputArea.Focus(); err != nil {
		return fmt.Errorf("TextArea focus: %w", err)
	}
	if completion, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"area-edit",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyEnter,
		},
	); inputErr != nil ||
		completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf("TextArea edit dispatch = %+v, %v", completion, inputErr)
	}
	if completion, inputErr := scene.App.DispatchTextInput(
		context.Background(),
		"self-check",
		"area-paste",
		expletives.TextInputEvent{
			Kind: expletives.TextInputPaste,
			Text: "\nPasted",
		},
	); inputErr != nil ||
		completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf("TextArea paste dispatch = %+v, %v", completion, inputErr)
	}
	if _, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"area-control-down",
		expletives.KeyEvent{
			Kind: expletives.KeyEventDown,
			Key:  expletives.KeyControl,
		},
	); inputErr != nil {
		return fmt.Errorf("TextArea Ctrl down: %w", inputErr)
	}
	areaCommit, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"area-commit",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyEnter,
		},
	)
	if _, releaseErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"area-control-up",
		expletives.KeyEvent{
			Kind: expletives.KeyEventUp,
			Key:  expletives.KeyControl,
		},
	); releaseErr != nil {
		return fmt.Errorf("TextArea Ctrl up: %w", releaseErr)
	}
	if inputErr != nil ||
		areaCommit.Command != CommandTextChanged ||
		areaCommit.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf("TextArea commit dispatch = %+v, %v", areaCommit, inputErr)
	}
	if err := invoke("show-progress", CommandProgress); err != nil {
		return err
	}
	progressBar := controls["progress.bar.determinate"].Details.Progress
	progressIndeterminate :=
		controls["progress.bar.indeterminate"].Details.Progress
	progressMeter := controls["progress.meter.horizontal"].Details.Progress
	progressVertical := controls["progress.meter.vertical"].Details.Progress
	progressSpinner := controls["progress.spinner"].Details.Progress
	progressDots := controls["progress.activity_dots"].Details.Progress
	progressReduced := controls["progress.spinner.reduced"].Details.Progress
	if !controls["screen.progress"].Visible ||
		!controls["progress.action.tick"].Focused ||
		progressBar == nil || progressBar.Current != 42 ||
		progressBar.Total != 100 ||
		progressBar.Status != expletives.ProgressRunning ||
		progressIndeterminate == nil ||
		!progressIndeterminate.Indeterminate ||
		progressMeter == nil || progressMeter.Value != 65 ||
		progressMeter.Orientation != expletives.Horizontal ||
		progressVertical == nil ||
		progressVertical.Orientation != expletives.Vertical ||
		progressSpinner == nil || progressSpinner.FrameIndex != 0 ||
		progressDots == nil || progressDots.FrameIndex != 0 ||
		progressReduced == nil || !progressReduced.ReducedMotion ||
		progressReduced.Tick != 0 ||
		len(snapshot.Overflows) != 0 {
		return errors.New("Progress catalog typed evidence is incomplete")
	}
	if err := invoke("progress-tick", CommandProgressTick); err != nil {
		return err
	}
	progressBar = controls["progress.bar.determinate"].Details.Progress
	progressIndeterminate =
		controls["progress.bar.indeterminate"].Details.Progress
	progressSpinner = controls["progress.spinner"].Details.Progress
	progressDots = controls["progress.activity_dots"].Details.Progress
	if progressBar.Current != 49 ||
		progressIndeterminate.Tick != 1 ||
		progressSpinner.Tick != 1 || progressSpinner.FrameIndex != 1 ||
		progressDots.Tick != 1 || progressDots.FrameIndex != 1 {
		return errors.New("Progress absolute Tick did not update atomically")
	}
	if err := invoke("progress-motion", CommandProgressMotion); err != nil {
		return err
	}
	if !controls["progress.spinner"].Details.Progress.ReducedMotion ||
		controls["progress.spinner"].Details.Progress.Tick != 0 ||
		!controls["progress.activity_dots"].Details.Progress.ReducedMotion {
		return errors.New("Progress reduced-motion state is not canonical")
	}
	if err := invoke("progress-reset", CommandProgressReset); err != nil {
		return err
	}
	if controls["progress.bar.determinate"].Details.Progress.Current != 42 ||
		controls["progress.spinner"].Details.Progress.Tick != 0 ||
		controls["progress.spinner"].Details.Progress.ReducedMotion {
		return errors.New("Progress Reset did not restore initial state")
	}
	if err := invoke("show-navigation", CommandNavigation); err != nil {
		return err
	}
	horizontal := controls["navigation.scrollbar.horizontal"]
	vertical := controls["navigation.scrollbar.vertical"]
	tabs := controls["navigation.tabs"].Details.TabbedPanel
	notebook := controls["navigation.notebook"].Details.TabbedPanel
	if !controls["screen.navigation"].Visible ||
		!horizontal.Focused ||
		horizontal.Details.ScrollBar == nil ||
		horizontal.Details.ScrollBar.Orientation != expletives.Horizontal ||
		horizontal.Details.ScrollBar.Offset != 40 ||
		horizontal.Details.ScrollBar.MaximumOffset != 80 ||
		horizontal.Details.ScrollBar.TrackSize < 1 ||
		vertical.Details.ScrollBar == nil ||
		vertical.Details.ScrollBar.Orientation != expletives.Vertical ||
		vertical.Details.ScrollBar.Offset != 32 ||
		tabs == nil || tabs.Selected != "overview" ||
		tabs.Current != "overview" || len(tabs.Tabs) != 3 ||
		notebook == nil || notebook.Selected != "one" ||
		notebook.Current != "one" || len(notebook.Tabs) != 2 ||
		!controls["navigation.tabs.page.overview"].Visible ||
		controls["navigation.tabs.page.details"].Visible ||
		!controls["navigation.notebook.page.one"].Visible ||
		controls["navigation.notebook.page.two"].Visible {
		return errors.New("Navigation catalog typed evidence is incomplete")
	}
	scrollCompletion, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-scroll-right",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyRight,
		},
	)
	if inputErr != nil ||
		scrollCompletion.Command != CommandNavigationChanged ||
		scrollCompletion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf(
			"ScrollBar raw Right dispatch = %+v, %v",
			scrollCompletion,
			inputErr,
		)
	}
	snapshot = scene.App.Snapshot()
	controls = indexControls(snapshot)
	if controls["navigation.scrollbar.horizontal"].Details.ScrollBar.Offset !=
		41 {
		return errors.New("ScrollBar raw Right did not advance Offset")
	}
	if err := scene.navigationTabs.Focus(); err != nil {
		return fmt.Errorf("TabbedPanel focus: %w", err)
	}
	tabMove, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-tab-right",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeyRight,
		},
	)
	if inputErr != nil || tabMove.Outcome != expletives.OutcomeApplied ||
		tabMove.Command != "" {
		return fmt.Errorf(
			"TabbedPanel raw Right dispatch = %+v, %v",
			tabMove,
			inputErr,
		)
	}
	snapshot = scene.App.Snapshot()
	controls = indexControls(snapshot)
	tabs = controls["navigation.tabs"].Details.TabbedPanel
	if tabs.Selected != "overview" || tabs.Current != "details" {
		return errors.New("TabbedPanel Right changed selection instead of focus")
	}
	tabSelect, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-tab-select",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  expletives.KeySpace,
		},
	)
	if inputErr != nil ||
		tabSelect.Command != CommandNavigationChanged ||
		tabSelect.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf(
			"TabbedPanel raw Space dispatch = %+v, %v",
			tabSelect,
			inputErr,
		)
	}
	snapshot = scene.App.Snapshot()
	controls = indexControls(snapshot)
	tabs = controls["navigation.tabs"].Details.TabbedPanel
	if tabs.Selected != "details" || tabs.Current != "details" ||
		controls["navigation.tabs.page.overview"].Visible ||
		!controls["navigation.tabs.page.details"].Visible {
		return errors.New("TabbedPanel selection did not switch page visibility")
	}
	if _, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-alt-down",
		expletives.KeyEvent{
			Kind: expletives.KeyEventDown,
			Key:  expletives.KeyAlt,
		},
	); inputErr != nil {
		return fmt.Errorf("Notebook Alt down: %w", inputErr)
	}
	notebookSelect, inputErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-notebook-mnemonic",
		expletives.KeyEvent{
			Kind: expletives.KeyEventPress,
			Key:  "w",
		},
	)
	if _, releaseErr := scene.App.DispatchKey(
		context.Background(),
		"self-check",
		"navigation-alt-up",
		expletives.KeyEvent{
			Kind: expletives.KeyEventUp,
			Key:  expletives.KeyAlt,
		},
	); releaseErr != nil {
		return fmt.Errorf("Notebook Alt up: %w", releaseErr)
	}
	if inputErr != nil ||
		notebookSelect.Command != CommandNavigationChanged ||
		notebookSelect.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf(
			"Notebook raw mnemonic dispatch = %+v, %v",
			notebookSelect,
			inputErr,
		)
	}
	snapshot = scene.App.Snapshot()
	controls = indexControls(snapshot)
	notebook = controls["navigation.notebook"].Details.TabbedPanel
	if notebook.Selected != "two" || notebook.Current != "two" ||
		controls["navigation.notebook.page.one"].Visible ||
		!controls["navigation.notebook.page.two"].Visible {
		return errors.New("Notebook mnemonic did not switch page visibility")
	}
	if err := invoke("hide-status", CommandStatusBar); err != nil {
		return err
	}
	status = controls["status.main"]
	if status.Visible {
		return errors.New("Status Bar toggle did not hide the live bottom row")
	}
	if err := invoke("restore-status", CommandStatusBar); err != nil {
		return err
	}
	if err := invoke("show-headers", CommandHeadersShow); err != nil {
		return err
	}
	for key, want := range map[string]expletives.Rect{
		"header.primary":        {Y: 1, Width: 64, Height: 1},
		"header.secondary":      {Y: 2, Width: 64, Height: 1},
		"footer.hotkeys.global": {Y: 18, Width: 64, Height: 1},
		"footer.hotkeys.screen": {Y: 17, Width: 64, Height: 1},
		"footer.guidance.focus": {Y: 16, Width: 64, Height: 1},
	} {
		if controls[key].Bounds != want || !controls[key].Visible {
			return fmt.Errorf(
				"%s chrome bounds = %+v visible=%t",
				key,
				controls[key].Bounds,
				controls[key].Visible,
			)
		}
	}
	if controls["header.product"].AbsoluteBounds.Height != 1 ||
		controls["header.grid.left"].AbsoluteBounds.Height != 1 ||
		controls["footer.hotkeys.global.items"].AbsoluteBounds.Height != 1 ||
		controls["footer.guidance.focus.text"].AbsoluteBounds.Height != 1 {
		return errors.New("Header/Footer toggle Layout evidence is incomplete")
	}
	if err := invoke("show-home", CommandViewHome); err != nil {
		return err
	}
	if !controls["screen.home"].Visible ||
		controls["screen.actions"].Visible ||
		controls["screen.selection"].Visible ||
		controls["screen.progress"].Visible ||
		controls["screen.navigation"].Visible ||
		controls["screen.status"].Visible ||
		controls["screen.headers_footers"].Visible ||
		!controls["header.primary"].Visible ||
		!controls["footer.hotkeys.screen"].Visible {
		return errors.New("File Home did not restore the empty catalog screen")
	}
	return nil
}
