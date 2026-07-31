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
	styles := map[StyleID]ResolvedStyle{
		"application.root": resolved,
		"panel":            resolved,
		"frame":            resolved,
		"frame.border":     resolved,
		"group_box":        resolved,
		"group_box.border": resolved,
		"layout.border":    resolved,
		"label":            resolved,
		"static_text":      resolved,
		"separator":        resolved,
		"rule":             resolved,
		"button":           resolved,
		"hotkey_bar":       resolved,
		"focus_guide_bar":  menuNormal,
		"checkbox":         resolved,
		"radio_group":      resolved,
		"radio_button":     resolved,
		"cycle_field":      resolved,
		"select_field":     resolved,
		"text_field":       resolved,
		"number_field":     resolved,
		"spin_box":         resolved,
		"text_area":        resolved,
		"progress_bar":     resolved,
		"meter":            resolved,
		"spinner":          resolved,
		"activity_dots":    resolved,
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
		"text_input.valid": {
			Foreground: RGB(0x00, 0xAA, 0x00),
			Background: resolved.Background,
		},
		"text_input.invalid": {
			Foreground: RGB(0xAA, 0xAA, 0x00),
			Background: resolved.Background,
		},
		"text_input.invalid_character": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: resolved.Background,
		},
		"text_input.selection": {
			Foreground: resolved.Foreground,
			Background: RGB(0x00, 0x00, 0x00),
		},
		"text_input.disabled": {
			Foreground: RGB(0x80, 0x80, 0x80),
			Background: resolved.Background,
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
