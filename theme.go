package expletives

import (
	"context"
	"fmt"
)

// Theme is an immutable mapping from semantic style IDs to resolved styles.
// Its zero value selects DefaultTheme only in AppOptions; use NewTheme,
// DefaultTheme, or App.Theme to obtain a usable standalone value.
type Theme struct {
	styles map[StyleID]ResolvedStyle
}

// NewTheme validates and copies semantic style definitions. It rejects an
// empty set, unsupported attributes, and conflicting duplicate IDs.
func NewTheme(styles ...Style) (Theme, error) {
	if len(styles) == 0 {
		return Theme{}, fmt.Errorf("%w: empty theme", ErrStyleMissing)
	}
	if len(styles) > MaxThemeStyles {
		return Theme{}, fmt.Errorf(
			"%w: theme exceeds %d styles",
			ErrStyleMissing,
			MaxThemeStyles,
		)
	}
	theme := Theme{
		styles: make(map[StyleID]ResolvedStyle, len(styles)),
	}
	for _, candidate := range styles {
		id, err := normalizeStyleID(candidate.ID, "")
		if err != nil {
			return Theme{}, err
		}
		resolved := candidate.Resolved()
		if err := validateResolvedStyle(resolved); err != nil {
			return Theme{}, err
		}
		if existing, found := theme.styles[id]; found &&
			existing != resolved {
			return Theme{}, fmt.Errorf(
				"%w: conflicting definition for %q",
				ErrStyleConflict,
				id,
			)
		}
		theme.styles[id] = resolved
	}
	return theme, nil
}

// DefaultTheme returns an independent copy of the fallback theme used when
// AppOptions supplies no Theme.
func DefaultTheme() Theme {
	white := RGB(0xFF, 0xFF, 0xFF)
	black := RGB(0, 0, 0)
	resolved := ResolvedStyle{
		Foreground: white,
		Background: black,
	}
	menuNormal := ResolvedStyle{
		Foreground: black,
		Background: RGB(0xC0, 0xC0, 0xC0),
	}
	menuSelected := ResolvedStyle{
		Foreground: black,
		Background: RGB(0x00, 0xAA, 0x00),
	}
	inputNormal := ResolvedStyle{
		Foreground: white,
		Background: RGB(0x00, 0x78, 0x78),
	}
	styles := map[StyleID]ResolvedStyle{
		"application.root": resolved,
		"panel":            resolved,
		"frame":            resolved,
		"frame.border":     resolved,
		"group_box":        resolved,
		"group_box.border": resolved,
		"modal_panel":      menuNormal,
		"modal_panel.border": {
			Foreground: white,
			Background: menuNormal.Background,
		},
		"modal_panel.shadow": {
			Foreground: black,
			Background: black,
		},
		"dialog": menuNormal,
		"dialog.border": {
			Foreground: white,
			Background: menuNormal.Background,
		},
		"dialog.shadow": {
			Foreground: black,
			Background: black,
		},
		"message_box": menuNormal,
		"message_box.border": {
			Foreground: white, Background: menuNormal.Background,
		},
		"message_box.shadow": {Foreground: black, Background: black},
		"confirm_dialog":     menuNormal,
		"confirm_dialog.border": {
			Foreground: white, Background: menuNormal.Background,
		},
		"confirm_dialog.shadow": {Foreground: black, Background: black},
		"input_dialog":          menuNormal,
		"input_dialog.border": {
			Foreground: white, Background: menuNormal.Background,
		},
		"input_dialog.shadow": {Foreground: black, Background: black},
		"progress_dialog":     menuNormal,
		"progress_dialog.border": {
			Foreground: white, Background: menuNormal.Background,
		},
		"progress_dialog.shadow": {Foreground: black, Background: black},
		"layout.border":          resolved,
		"label":                  resolved,
		"static_text":            resolved,
		"separator":              resolved,
		"rule":                   resolved,
		"button":                 resolved,
		"hotkey_bar":             resolved,
		"focus_guide_bar":        menuNormal,
		"checkbox":               resolved,
		"radio_group":            resolved,
		"radio_button":           resolved,
		"cycle_field":            resolved,
		"select_field":           resolved,
		"text_field":             inputNormal,
		"number_field":           inputNormal,
		"spin_box":               inputNormal,
		"spin_box.button": {
			Foreground: black,
			Background: RGB(0xC0, 0xC0, 0xC0),
		},
		"spin_box.button_focused": {
			Foreground: black,
			Background: RGB(0x00, 0xAA, 0x00),
		},
		"spin_box.button_disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: RGB(0xC0, 0xC0, 0xC0),
		},
		"text_area":         inputNormal,
		"progress_bar":      resolved,
		"meter":             resolved,
		"spinner":           resolved,
		"activity_dots":     resolved,
		"scroll_bar":        resolved,
		"tabbed_panel":      resolved,
		"notebook":          resolved,
		"viewport":          resolved,
		"scrollable_panel":  resolved,
		"markdown_view":     resolved,
		"log_view":          resolved,
		"stream_view":       resolved,
		"list_box":          menuNormal,
		"tree_view":         menuNormal,
		"table":             menuNormal,
		"data_grid":         menuNormal,
		"drop_down":         inputNormal,
		"combo_box":         inputNormal,
		"drop_down.focused": menuSelected,
		"drop_down.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: inputNormal.Background,
		},
		"combo_box.focused": menuSelected,
		"combo_box.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: inputNormal.Background,
		},
		"drop_down.popup":         menuNormal,
		"drop_down.popup_border":  menuNormal,
		"tabbed_panel.border":     resolved,
		"notebook.border":         resolved,
		"viewport.border":         resolved,
		"scrollable_panel.border": resolved,
		"markdown_view.border":    resolved,
		"log_view.border":         resolved,
		"stream_view.border":      resolved,
		"list_box.border":         menuNormal,
		"tree_view.border":        menuNormal,
		"table.border":            menuNormal,
		"data_grid.border":        menuNormal,
		"table.header":            menuNormal,
		"table.header_current":    menuSelected,
		"table.sort": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"table.cell_current": menuSelected,
		"table.row_selected": {
			Foreground: white,
			Background: RGB(0x00, 0x00, 0xAA),
		},
		"data_grid.edit": inputNormal,
		"data_grid.edit_focused": {
			Foreground: black,
			Background: RGB(0x00, 0xAA, 0xAA),
		},
		"data_grid.edit_invalid": {
			Foreground: black,
			Background: RGB(0xAA, 0xAA, 0x00),
		},
		"data_grid.edit_invalid_character": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: RGB(0xAA, 0xAA, 0x00),
		},
		"tree.guide": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuNormal.Background,
		},
		"tree.branch": {
			Foreground: black,
			Background: menuNormal.Background,
		},
		"tree.expanded": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"collection.current": menuSelected,
		"collection.selected": {
			Foreground: white,
			Background: RGB(0x00, 0x00, 0xAA),
		},
		"collection.current_selected": {
			Foreground: white,
			Background: RGB(0x00, 0xAA, 0x00),
			Attributes: StyleBold,
		},
		"collection.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuNormal.Background,
		},
		"collection.empty": menuNormal,
		"collection.loading": {
			Foreground: RGB(0xFF, 0xFF, 0x00),
			Background: menuNormal.Background,
		},
		"collection.error": {
			Foreground: white,
			Background: RGB(0xAA, 0x00, 0x00),
		},
		"markdown.heading": {
			Foreground: white,
			Background: resolved.Background,
			Attributes: StyleBold,
		},
		"markdown.emphasis": {
			Foreground: white,
			Background: resolved.Background,
			Attributes: StyleItalic,
		},
		"markdown.strong": {
			Foreground: white,
			Background: resolved.Background,
			Attributes: StyleBold,
		},
		"markdown.code": {
			Foreground: RGB(0x00, 0xAA, 0xAA),
			Background: resolved.Background,
		},
		"markdown.link": {
			Foreground: RGB(0x00, 0xAA, 0xAA),
			Background: resolved.Background,
			Attributes: StyleUnderline,
		},
		"markdown.quote": {
			Foreground: RGB(0xC0, 0xC0, 0xC0),
			Background: resolved.Background,
			Attributes: StyleItalic,
		},
		"markdown.list_marker": {
			Foreground: RGB(0xAA, 0xAA, 0x00),
			Background: resolved.Background,
		},
		"markdown.rule": {
			Foreground: RGB(0xC0, 0xC0, 0xC0),
			Background: resolved.Background,
		},
		"log.timestamp": {
			Foreground: RGB(0xC0, 0xC0, 0xC0),
			Background: resolved.Background,
		},
		"log.debug": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: resolved.Background,
		},
		"log.info": resolved,
		"log.warning": {
			Foreground: RGB(0xFF, 0xFF, 0x00),
			Background: resolved.Background,
			Attributes: StyleBold,
		},
		"log.error": {
			Foreground: RGB(0xFF, 0x55, 0x55),
			Background: resolved.Background,
			Attributes: StyleBold,
		},
		"stream.truncated": {
			Foreground: RGB(0x00, 0x00, 0x00),
			Background: RGB(0xFF, 0xFF, 0x00),
			Attributes: StyleBold,
		},
		"content.dropped": {
			Foreground: RGB(0x00, 0x00, 0x00),
			Background: RGB(0xFF, 0xFF, 0x00),
			Attributes: StyleBold,
		},
		"progress.fill": {
			Foreground: black,
			Background: RGB(0x00, 0xAA, 0x00),
		},
		"progress.text": {
			Foreground: white,
			Background: RGB(0x00, 0x00, 0x00),
			Attributes: StyleBold,
		},
		"progress.completed": {
			Foreground: black,
			Background: RGB(0x00, 0xAA, 0x00),
		},
		"progress.failed": {
			Foreground: white,
			Background: RGB(0xAA, 0x00, 0x00),
		},
		"progress.cancelled": {
			Foreground: black,
			Background: RGB(0xAA, 0xAA, 0x00),
		},
		"scrollbar.page": resolved,
		"scrollbar.arrow": {
			Foreground: white,
			Background: RGB(0x00, 0x00, 0xAA),
		},
		"scrollbar.thumb": {
			Foreground: black,
			Background: RGB(0xC0, 0xC0, 0xC0),
		},
		"scrollbar.focused": {
			Foreground: black,
			Background: RGB(0x00, 0xAA, 0x00),
		},
		"scrollbar.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: resolved.Background,
		},
		"scrollbar.corner": resolved,
		"tab.normal":       menuNormal,
		"tab.mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"tab.selected": menuNormal,
		"tab.selected_mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"tab.focused": menuSelected,
		"tab.focused_mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuSelected.Background,
		},
		"tab.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuNormal.Background,
		},
		"tab.continuation": menuNormal,
		"text_input.valid": {
			Foreground: RGB(0x00, 0xAA, 0x00),
			Background: inputNormal.Background,
		},
		"text_input.invalid": {
			Foreground: RGB(0xAA, 0xAA, 0x00),
			Background: inputNormal.Background,
		},
		"text_input.invalid_character": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: inputNormal.Background,
		},
		"text_input.selection": {
			Foreground: resolved.Foreground,
			Background: RGB(0x00, 0x00, 0x00),
		},
		"text_input.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: inputNormal.Background,
		},
		"text_input.focused": {
			Foreground: white,
			Background: black,
		},
		"text_input.focused_valid": {
			Foreground: RGB(0x00, 0xAA, 0x00),
			Background: black,
		},
		"text_input.focused_invalid": {
			Foreground: RGB(0xAA, 0xAA, 0x00),
			Background: black,
		},
		"text_input.focused_invalid_character": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: black,
		},
		"text_input.focused_selection": {
			Foreground: black,
			Background: RGB(0xC0, 0xC0, 0xC0),
		},
		"selection.mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: resolved.Background,
		},
		"selection.focused": menuSelected,
		"selection.focused_mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuSelected.Background,
		},
		"selection.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: resolved.Background,
		},
		"status_bar": menuNormal,
		"status.shortcut": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"status.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuNormal.Background,
		},
		"header":      menuNormal,
		"footer":      menuNormal,
		"menu_bar":    menuNormal,
		"menu.popup":  menuNormal,
		"menu.border": menuNormal,
		"menu.mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuNormal.Background,
		},
		"menu.focused": menuSelected,
		"menu.focused_mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: menuSelected.Background,
		},
		"menu.disabled": ResolvedStyle{
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuNormal.Background,
		},
		"menu.focused_disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: menuSelected.Background,
		},
		"menu.shadow": {
			Foreground: black,
			Background: black,
		},
	}
	return Theme{styles: styles}
}

// Styles returns a caller-owned copy sorted by semantic ID.
func (t Theme) Styles() []Style {
	styles := make([]Style, 0, len(t.styles))
	for id, resolved := range t.styles {
		styles = append(styles, styleDefinition(id, resolved))
	}
	sortStyles(styles)
	return styles
}

// Resolve returns the immutable resolved value for id.
func (t Theme) Resolve(id StyleID) (ResolvedStyle, bool) {
	resolved, found := t.styles[id]
	return resolved, found
}

// Theme returns an independent copy of the App's current semantic theme.
func (a *App) Theme() Theme {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return Theme{styles: cloneStyleMap(a.styles)}
}

// SetTheme atomically replaces every active semantic style. A changed Theme
// publishes one snapshot. The replacement must define every active style ID.
// This convenience method waits at most DefaultMutationWait; use a Transaction
// and Commit with a context when cancellation or batching is required.
func (a *App) SetTheme(theme Theme) error {
	transaction := a.NewTransaction()
	if err := transaction.SetTheme(theme); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

func validateResolvedStyle(style ResolvedStyle) error {
	if style.Attributes&^supportedStyleAttributes != 0 {
		return fmt.Errorf(
			"%w: unsupported style attributes %#x",
			ErrStyleConflict,
			style.Attributes,
		)
	}
	return nil
}

func cloneStyleMap(
	source map[StyleID]ResolvedStyle,
) map[StyleID]ResolvedStyle {
	cloned := make(map[StyleID]ResolvedStyle, len(source))
	for id, style := range source {
		cloned[id] = style
	}
	return cloned
}

func styleMapsEqual(
	left map[StyleID]ResolvedStyle,
	right map[StyleID]ResolvedStyle,
) bool {
	if len(left) != len(right) {
		return false
	}
	for id, style := range left {
		if right[id] != style {
			return false
		}
	}
	return true
}

func sortStyles(styles []Style) {
	for index := 1; index < len(styles); index++ {
		for cursor := index; cursor > 0 &&
			styles[cursor].ID < styles[cursor-1].ID; cursor-- {
			styles[cursor], styles[cursor-1] =
				styles[cursor-1], styles[cursor]
		}
	}
}
