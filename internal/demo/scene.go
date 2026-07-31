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
	ScenarioID                                  = "toolkit.catalog"
	CommandFixtureToggle   expletives.CommandID = "fixture.toggle"
	CommandPanelRaise      expletives.CommandID = "layout.panel.raise"
	CommandPanelLower      expletives.CommandID = "layout.panel.lower"
	CommandLayerRaise      expletives.CommandID = "layout.layer.raise"
	CommandLayerLower      expletives.CommandID = "layout.layer.lower"
	CommandScenarioReset   expletives.CommandID = "scenario.reset"
	CommandViewCore        expletives.CommandID = "view.core"
	CommandViewText        expletives.CommandID = "view.text"
	CommandViewActions     expletives.CommandID = "view.actions"
	CommandViewMenus       expletives.CommandID = "view.menus"
	CommandViewAbout       expletives.CommandID = "view.about"
	CommandViewPanelsCore  expletives.CommandID = "view.panels.core"
	CommandViewPanelStyles expletives.CommandID = "view.panels.styles"
	CommandViewLayoutBox   expletives.CommandID = "view.layouts.box"
	CommandViewLayoutGrid  expletives.CommandID = "view.layouts.grid"
	CommandPanelScrollbars expletives.CommandID = "catalog.panels.scrollbars"
	CommandLayoutAbsolute  expletives.CommandID = "catalog.layouts.absolute"
	CommandStatusBar       expletives.CommandID = "catalog.controls.status"
	CommandHeadersFooters  expletives.CommandID = "catalog.controls.headers_footers"
	CommandSelection       expletives.CommandID = "catalog.controls.selection"
	CommandTextInput       expletives.CommandID = "catalog.controls.input"
	CommandProgress        expletives.CommandID = "catalog.controls.progress"
	CommandNavigation      expletives.CommandID = "catalog.controls.navigation"
	CommandScrolling       expletives.CommandID = "catalog.controls.scrolling"
	CommandCollections     expletives.CommandID = "catalog.controls.collections"
	CommandPanelMenu       expletives.CommandID = "catalog.menus.panel"
	CommandContextMenu     expletives.CommandID = "catalog.menus.context"
	CommandDialogMessage   expletives.CommandID = "catalog.dialogs.message"
	CommandDialogConfirm   expletives.CommandID = "catalog.dialogs.confirm"
	CommandDialogInput     expletives.CommandID = "catalog.dialogs.input"
	CommandDialogProgress  expletives.CommandID = "catalog.dialogs.progress"
	CommandUnavailable     expletives.CommandID = "fixture.unavailable"
	CommandAppQuit         expletives.CommandID = "app.quit"
	CommandAppInterrupt    expletives.CommandID = "app.interrupt"
)

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
	bannerStyle = expletives.Style{
		ID:         "fixture.automation",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xFF, 0xD7, 0x00),
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
)

// Scene owns the catalog controls and its small application controller state.
type Scene struct {
	App *expletives.App

	mu             sync.Mutex
	accent         *expletives.Panel
	accentControls []expletives.Control
	layer          *expletives.BoxLayout
	screens        map[expletives.CommandID]*expletives.Panel
	activeScreen   expletives.CommandID
	toggled        bool
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
		bannerStyle,
		menuStyle,
		menuPopupStyle,
		menuBorderStyle,
		menuMnemonicStyle,
		menuFocusedStyle,
		menuFocusedMnemonicStyle,
		menuDisabledStyle,
		menuFocusedDisabledStyle,
		menuShadowStyle,
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
	for _, definition := range initialCommandDefinitions() {
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
	content, err := transaction.NewPanel(outer, expletives.PanelOptions{
		AutomationKey: "catalog.content",
		Style:         canvasStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	coreScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.core",
		Style:         canvasStyle.ID,
	})
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
	menusScreen, err := transaction.NewPanel(content, expletives.PanelOptions{
		AutomationKey: "screen.menus",
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

	red, err := transaction.NewPanel(coreScreen, expletives.PanelOptions{
		AutomationKey: "panel.red",
		MinimumSize:   expletives.Size{Width: 10, Height: 5},
		Style:         redStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	accent, err := transaction.NewPanel(coreScreen, expletives.PanelOptions{
		AutomationKey: "panel.accent",
		MinimumSize:   expletives.Size{Width: 12, Height: 5},
		Style:         greenStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	group, err := transaction.NewGroupBox(
		coreScreen,
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
		coreScreen,
		expletives.FrameOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "layers",
				MinimumSize:   expletives.Size{Width: 20, Height: 3},
				Style:         canvasStyle.ID,
			},
			Title:       "Layout stacking: green over red",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderNone,
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

	bannerTitle := "Automation off"
	if automationPath != "" {
		bannerTitle = "UNAUTH AUTOMATION ENABLED: " + automationPath
	}
	banner, err := transaction.NewFrame(outer, expletives.FrameOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "automation.status",
			MinimumSize:   expletives.Size{Width: 20, Height: 3},
			Style:         bannerStyle.ID,
		},
		Title:       bannerTitle,
		BorderStyle: bannerStyle.ID,
		BorderForm:  expletives.BorderShadeMedium,
	})
	if err != nil {
		return nil, err
	}
	hotkeyBar, err := transaction.NewHotkeyBar(
		banner,
		expletives.HotkeyBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "action.hotkeys",
				Style:         bannerStyle.ID,
			},
			Items: []expletives.HotkeyBarItem{
				{Command: CommandFixtureToggle},
				{Command: CommandScenarioReset},
				{Command: CommandAppQuit},
				{Command: CommandAppInterrupt},
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
	for _, item := range []struct {
		control expletives.Control
		grow    int
	}{
		{content, 1},
		{banner, 0},
	} {
		if err := outerLayout.AddPanel(
			item.control,
			expletives.LayoutItemOptions{Grow: item.grow},
		); err != nil {
			return nil, err
		}
	}
	coreScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.screen.core"},
	)
	if err != nil {
		return nil, err
	}
	textScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.screen.text"},
	)
	if err != nil {
		return nil, err
	}
	actionsScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.screen.actions"},
	)
	if err != nil {
		return nil, err
	}
	menusScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.screen.menus"},
	)
	if err != nil {
		return nil, err
	}
	aboutScreenLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.screen.about"},
	)
	if err != nil {
		return nil, err
	}
	for layout, screen := range map[*expletives.BoxLayout]expletives.Control{
		coreScreenLayout:    coreScreen,
		textScreenLayout:    textScreen,
		actionsScreenLayout: actionsScreen,
		menusScreenLayout:   menusScreen,
		aboutScreenLayout:   aboutScreen,
	} {
		if err := layout.AddPanel(
			screen,
			expletives.LayoutItemOptions{Grow: 1},
		); err != nil {
			return nil, err
		}
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
		AutomationKey: "layout.core.grid",
		Columns:       3,
		HorizontalGap: 1,
		Border: expletives.BorderOptions{
			Form:  expletives.BorderShadeDark,
			Style: borderStyle.ID,
		},
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
	coreLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{AutomationKey: "layout.core"},
	)
	if err != nil {
		return nil, err
	}
	if err := coreLayout.AddLayout(
		coreGrid,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := coreLayout.AddPanel(
		layerFrame,
		expletives.LayoutItemOptions{},
	); err != nil {
		return nil, err
	}
	groupLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.group",
			Border: expletives.BorderOptions{
				Form:  expletives.BorderBlock,
				Style: borderStyle.ID,
			},
		},
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
	hotkeyLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{AutomationKey: "layout.hotkeys"},
	)
	if err != nil {
		return nil, err
	}
	if err := hotkeyLayout.AddPanel(
		hotkeyBar,
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
	if err := transaction.SetLayout(content, coreScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(content, textScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(content, actionsScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(content, menusScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(content, aboutScreenLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(coreScreen, coreLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(group, groupLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(layerFrame, backLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(layerFrame, frontLayout); err != nil {
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
	if err := transaction.SetLayout(menusScreen, menusLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(aboutScreen, aboutLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(actionPanel, actionLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(banner, hotkeyLayout); err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}

	scene := &Scene{
		App:          app,
		accent:       accent,
		layer:        backLayout,
		activeScreen: CommandViewCore,
		accentControls: []expletives.Control{
			accent,
			textAccent,
			staticText,
			rule,
			actionPreview,
		},
		screens: map[expletives.CommandID]*expletives.Panel{
			CommandViewCore:    coreScreen,
			CommandViewText:    textScreen,
			CommandViewActions: actionsScreen,
			CommandViewMenus:   menusScreen,
			CommandViewAbout:   aboutScreen,
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
	return scene, nil
}

func initialCommandDefinitions() []expletives.CommandDefinition {
	return []expletives.CommandDefinition{
		toggleDefinition(false),
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
		screenDefinition(CommandViewCore, true),
		screenDefinition(CommandViewText, false),
		screenDefinition(CommandViewActions, false),
		screenDefinition(CommandViewMenus, false),
		screenDefinition(CommandViewAbout, false),
		catalogScreenDefinition(
			CommandViewPanelsCore,
			"Core Panels",
			true,
		),
		catalogScreenDefinition(
			CommandViewPanelStyles,
			"Visual Styles",
			true,
		),
		catalogScreenDefinition(
			CommandViewLayoutBox,
			"Box Layout",
			true,
		),
		catalogScreenDefinition(
			CommandViewLayoutGrid,
			"Grid Layout",
			true,
		),
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
			CommandStatusBar,
			"Status Bar",
			"Status Bar",
		),
		unavailableCatalogDefinition(
			CommandHeadersFooters,
			"Headers / Footers",
			"Headers and Footers",
		),
		unavailableCatalogDefinition(
			CommandSelection,
			"Selection",
			"Selection",
		),
		unavailableCatalogDefinition(
			CommandTextInput,
			"Text / Numeric Input",
			"Text and Numeric Input",
		),
		unavailableCatalogDefinition(
			CommandProgress,
			"Progress",
			"Progress",
		),
		unavailableCatalogDefinition(
			CommandNavigation,
			"Navigation",
			"Navigation and Chrome",
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

func screenDefinition(
	id expletives.CommandID,
	checked bool,
) expletives.CommandDefinition {
	labels := map[expletives.CommandID]string{
		CommandViewCore:    "Core / Layout",
		CommandViewText:    "Text / Display",
		CommandViewActions: "Actions",
		CommandViewMenus:   "Menu Bar",
		CommandViewAbout:   "About",
	}
	return expletives.CommandDefinition{
		ID: id, Label: labels[id],
		Description: "Show one purpose-specific toolkit catalog screen",
		Enabled:     true, Checked: checked, Automation: true,
	}
}

func catalogScreenDefinition(
	id expletives.CommandID,
	label string,
	checked bool,
) expletives.CommandDefinition {
	return expletives.CommandDefinition{
		ID: id, Label: label,
		Description: "Show the existing combined core demonstration at the " +
			label + " section",
		Enabled: true, Checked: checked, Automation: true,
	}
}

func catalogMenuItems() ([]expletives.MenuItem, error) {
	file, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{{
			Key: "menu.file.quit", Kind: expletives.MenuItemCommand,
			Command: CommandAppQuit, Mnemonic: "q",
		}},
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
			{Key: "menu.panels.separator", Kind: expletives.MenuItemSeparator},
			{
				Key: "menu.panels.scrollbars", Kind: expletives.MenuItemCommand,
				Command: CommandPanelScrollbars, Mnemonic: "s",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	stacking, err := expletives.NewMenu(expletives.MenuOptions{
		Items: []expletives.MenuItem{
			{
				Key:     "menu.stack.panel.raise",
				Kind:    expletives.MenuItemCommand,
				Command: CommandPanelRaise, Mnemonic: "p",
			},
			{
				Key:     "menu.stack.panel.lower",
				Kind:    expletives.MenuItemCommand,
				Command: CommandPanelLower, Mnemonic: "o",
			},
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
				Label: "Stacking", Mnemonic: "s", Menu: stacking,
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
				Key:  "menu.controls.separator.chrome",
				Kind: expletives.MenuItemSeparator,
			},
			{
				Key: "menu.controls.status", Kind: expletives.MenuItemCommand,
				Command: CommandStatusBar, Mnemonic: "s",
			},
			{
				Key: "menu.controls.headers", Kind: expletives.MenuItemCommand,
				Command: CommandHeadersFooters, Mnemonic: "h",
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
	return s.App.SetSize(size)
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
	case CommandScenarioReset:
		changed := s.toggled
		s.toggled = false
		transaction := s.App.NewTransaction()
		for _, control := range s.accentControls {
			if err := transaction.SetStyle(
				control,
				greenStyle.ID,
			); err != nil {
				return expletives.OutcomeFailed, err
			}
		}
		if err := transaction.Commit(context.Background()); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.App.ReplaceCommand(
			toggleDefinition(false),
		); err != nil {
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
	case CommandViewCore, CommandViewText, CommandViewActions,
		CommandViewMenus, CommandViewAbout:
		return s.switchScreenLocked(command.ID)
	case CommandViewPanelsCore, CommandViewPanelStyles,
		CommandViewLayoutBox, CommandViewLayoutGrid:
		return s.switchScreenLocked(CommandViewCore)
	case CommandPanelRaise:
		return mutationOutcome(s.App, s.accent.Raise)
	case CommandPanelLower:
		return mutationOutcome(s.App, s.accent.Lower)
	case CommandLayerRaise:
		return mutationOutcome(s.App, s.layer.Raise)
	case CommandLayerLower:
		return mutationOutcome(s.App, s.layer.Lower)
	case CommandAppQuit:
		return expletives.OutcomeExited, nil
	case CommandAppInterrupt:
		return expletives.OutcomeInterrupted, nil
	default:
		return expletives.OutcomeRejected, nil
	}
}

func (s *Scene) switchScreenLocked(
	target expletives.CommandID,
) (expletives.Outcome, error) {
	if s.activeScreen == target {
		return expletives.OutcomeNoOp, nil
	}
	transaction := s.App.NewTransaction()
	for command, screen := range s.screens {
		if err := transaction.SetVisible(
			screen,
			command == target,
		); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return expletives.OutcomeFailed, err
	}
	s.activeScreen = target
	for _, command := range []expletives.CommandID{
		CommandViewCore,
		CommandViewText,
		CommandViewActions,
		CommandViewMenus,
		CommandViewAbout,
	} {
		if err := s.App.ReplaceCommand(
			screenDefinition(command, command == target),
		); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	for _, definition := range []struct {
		id    expletives.CommandID
		label string
	}{
		{CommandViewPanelsCore, "Core Panels"},
		{CommandViewPanelStyles, "Visual Styles"},
		{CommandViewLayoutBox, "Box Layout"},
		{CommandViewLayoutGrid, "Grid Layout"},
	} {
		if err := s.App.ReplaceCommand(catalogScreenDefinition(
			definition.id,
			definition.label,
			target == CommandViewCore,
		)); err != nil {
			return expletives.OutcomeFailed, err
		}
	}
	return expletives.OutcomeApplied, nil
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
	controls := make(map[string]expletives.ControlSnapshot)
	for _, control := range snapshot.Controls {
		controls[control.Key] = control
	}
	for _, key := range []string{
		"menu.main",
		"screen.core",
		"screen.text",
		"screen.actions",
		"screen.menus",
		"screen.about",
		"menus.overview",
		"help.about",
		"panel.red",
		"panel.accent",
		"display.label",
		"display.static_text",
		"action.toggle",
		"action.hotkeys",
		"layer.back",
		"layer.front",
	} {
		if _, exists := controls[key]; !exists {
			return fmt.Errorf("catalog control %q is absent", key)
		}
	}
	menu := controls["menu.main"].Details.MenuBar
	if menu == nil || len(menu.Entries) != 43 ||
		len(menu.OpenPath) != 0 {
		return errors.New("MenuBar typed evidence is incomplete")
	}
	helpAtEnd := false
	expectedRootMnemonics := map[string]expletives.Key{
		"menu.file":     "i",
		"menu.panels":   "n",
		"menu.layouts":  "a",
		"menu.controls": "c",
		"menu.menus":    "m",
		"menu.dialogs":  "d",
		"menu.help":     "p",
	}
	seenRootMnemonics := make(map[string]bool, len(expectedRootMnemonics))
	catalogLabels := map[string]string{
		"menu.panels.core":   "Core Panels",
		"menu.panels.styles": "Visual Styles",
		"menu.layouts.box":   "Box Layout",
		"menu.layouts.grid":  "Grid Layout",
	}
	seenCatalogLabels := make(map[string]bool, len(catalogLabels))
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
	if len(seenCatalogLabels) != len(catalogLabels) {
		return errors.New("catalog navigation labels are incomplete")
	}
	if !controls["screen.core"].Visible ||
		controls["screen.text"].Visible ||
		controls["screen.actions"].Visible ||
		controls["screen.menus"].Visible ||
		controls["screen.about"].Visible {
		return errors.New("initial catalog screen visibility is invalid")
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
	front := controls["layer.front"]
	point := expletives.Point{
		X: front.AbsoluteBounds.X + front.AbsoluteBounds.Width/2,
		Y: front.AbsoluteBounds.Y + front.AbsoluteBounds.Height/2,
	}
	cell, ok := snapshot.Frame.Cell(point.X, point.Y)
	if !ok || cell.Owner != front.ID {
		return errors.New("front Layout does not initially paint above back")
	}
	if err := scene.layer.Raise(); err != nil {
		return err
	}
	snapshot = scene.App.Snapshot()
	cell, ok = snapshot.Frame.Cell(point.X, point.Y)
	if !ok || cell.Owner != controls["layer.back"].ID {
		return errors.New("Layout Raise did not move the complete red layer")
	}
	completion, err := scene.App.InvokeCommand(
		context.Background(),
		"self-check",
		"show-actions",
		CommandViewActions,
		"",
	)
	if err != nil || completion.Outcome != expletives.OutcomeApplied {
		return fmt.Errorf("show Actions screen: %v (%s)", err, completion.Outcome)
	}
	snapshot = scene.App.Snapshot()
	controls = make(map[string]expletives.ControlSnapshot)
	for _, control := range snapshot.Controls {
		controls[control.Key] = control
	}
	if controls["screen.core"].Visible ||
		!controls["screen.actions"].Visible ||
		!controls["action.toggle"].Focused {
		return errors.New("Actions screen did not become visible and focused")
	}
	return nil
}
