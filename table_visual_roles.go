package expletives

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// TableVisualRoles is the complete fixed set of semantic Theme roles used by
// one Table. Empty fields select the compatible built-in role for that state.
// Values are copied and retain no resolved colors.
type TableVisualRoles struct {
	Body               StyleID `json:"body"`
	Border             StyleID `json:"border"`
	Header             StyleID `json:"header"`
	CurrentHeader      StyleID `json:"current_header"`
	SortMarker         StyleID `json:"sort_marker"`
	CurrentCell        StyleID `json:"current_cell"`
	SelectedRow        StyleID `json:"selected_row"`
	CurrentRow         StyleID `json:"current_row"`
	CurrentSelectedRow StyleID `json:"current_selected_row"`
	Disabled           StyleID `json:"disabled"`
	Empty              StyleID `json:"empty"`
	Loading            StyleID `json:"loading"`
	Error              StyleID `json:"error"`

	ScrollbarPage         StyleID `json:"scrollbar_page"`
	ScrollbarArrow        StyleID `json:"scrollbar_arrow"`
	ScrollbarThumb        StyleID `json:"scrollbar_thumb"`
	ScrollbarFocusedThumb StyleID `json:"scrollbar_focused_thumb"`
	ScrollbarDisabled     StyleID `json:"scrollbar_disabled"`
	ScrollbarCorner       StyleID `json:"scrollbar_corner"`

	ColumnsBand           StyleID `json:"columns_band"`
	ColumnsAction         StyleID `json:"columns_action"`
	ColumnsActionDefault  StyleID `json:"columns_action_default"`
	ColumnsActionFocused  StyleID `json:"columns_action_focused"`
	ColumnsActionPressed  StyleID `json:"columns_action_pressed"`
	ColumnsActionDisabled StyleID `json:"columns_action_disabled"`
	ColumnsActionMnemonic StyleID `json:"columns_action_mnemonic"`
	ColumnsActionShadow   StyleID `json:"columns_action_shadow"`
}

// DataGridVisualRoles extends the Table role set with editable-cell states.
type DataGridVisualRoles struct {
	TableVisualRoles
	Editable         StyleID `json:"editable"`
	FocusedEdit      StyleID `json:"focused_edit"`
	InvalidEdit      StyleID `json:"invalid_edit"`
	InvalidCharacter StyleID `json:"invalid_character"`
	TextSelection    StyleID `json:"text_selection"`
}

type tableRoleDefaultOptions struct {
	body   StyleID
	border StyleID
}

func defaultTableVisualRoles(options tableRoleDefaultOptions) TableVisualRoles {
	body := options.body
	if body == "" {
		body = "table"
	}
	border := options.border
	if border == "" {
		border = "table.border"
	}
	return TableVisualRoles{
		Body: body, Border: border,
		Header: "table.header", CurrentHeader: "table.header_current",
		SortMarker: "table.sort", CurrentCell: "table.cell_current",
		SelectedRow: "table.row_selected", CurrentRow: "collection.current",
		CurrentSelectedRow: "collection.current_selected",
		Disabled:           "collection.disabled", Empty: "collection.empty",
		Loading: "collection.loading", Error: "collection.error",
		ScrollbarPage: "scrollbar.page", ScrollbarArrow: "scrollbar.arrow",
		ScrollbarThumb:        "scrollbar.thumb",
		ScrollbarFocusedThumb: "scrollbar.focused",
		ScrollbarDisabled:     "scrollbar.disabled",
		ScrollbarCorner:       "scrollbar.corner",
		ColumnsBand:           body, ColumnsAction: "button",
		ColumnsActionDefault:  "button.default",
		ColumnsActionFocused:  "button.focused",
		ColumnsActionPressed:  "button.pressed",
		ColumnsActionDisabled: "button.disabled",
		ColumnsActionMnemonic: "button.mnemonic",
		ColumnsActionShadow:   "button.shadow",
	}
}

func defaultDataGridVisualRoles(body, border StyleID) DataGridVisualRoles {
	if body == "" {
		body = "data_grid"
	}
	if border == "" {
		border = "data_grid.border"
	}
	return DataGridVisualRoles{
		TableVisualRoles: defaultTableVisualRoles(tableRoleDefaultOptions{
			body: body, border: border,
		}),
		Editable: "data_grid.edit", FocusedEdit: "data_grid.edit_focused",
		InvalidEdit:      "data_grid.edit_invalid",
		InvalidCharacter: "data_grid.edit_invalid_character",
		TextSelection:    "text_input.focused_selection",
	}
}

func normalizeTableVisualRoles(
	roles TableVisualRoles,
	defaults TableVisualRoles,
) (TableVisualRoles, error) {
	values := []*StyleID{
		&roles.Body, &roles.Border, &roles.Header, &roles.CurrentHeader,
		&roles.SortMarker, &roles.CurrentCell, &roles.SelectedRow,
		&roles.CurrentRow, &roles.CurrentSelectedRow, &roles.Disabled,
		&roles.Empty, &roles.Loading, &roles.Error, &roles.ScrollbarPage,
		&roles.ScrollbarArrow, &roles.ScrollbarThumb,
		&roles.ScrollbarFocusedThumb, &roles.ScrollbarDisabled,
		&roles.ScrollbarCorner, &roles.ColumnsBand, &roles.ColumnsAction,
		&roles.ColumnsActionDefault, &roles.ColumnsActionFocused,
		&roles.ColumnsActionPressed, &roles.ColumnsActionDisabled,
		&roles.ColumnsActionMnemonic, &roles.ColumnsActionShadow,
	}
	defaultValues := tableVisualRoleIDs(defaults)
	for index, value := range values {
		normalized, err := normalizeStyleID(*value, defaultValues[index])
		if err != nil {
			return TableVisualRoles{}, fmt.Errorf("Table visual role: %w", err)
		}
		*value = normalized
	}
	return roles, nil
}

func normalizeDataGridVisualRoles(
	roles DataGridVisualRoles,
	defaults DataGridVisualRoles,
) (DataGridVisualRoles, error) {
	var err error
	roles.TableVisualRoles, err = normalizeTableVisualRoles(
		roles.TableVisualRoles,
		defaults.TableVisualRoles,
	)
	if err != nil {
		return DataGridVisualRoles{}, err
	}
	values := []*StyleID{
		&roles.Editable, &roles.FocusedEdit, &roles.InvalidEdit,
		&roles.InvalidCharacter, &roles.TextSelection,
	}
	defaultValues := []StyleID{
		defaults.Editable, defaults.FocusedEdit, defaults.InvalidEdit,
		defaults.InvalidCharacter, defaults.TextSelection,
	}
	for index, value := range values {
		normalized, normalizeErr := normalizeStyleID(*value, defaultValues[index])
		if normalizeErr != nil {
			return DataGridVisualRoles{}, fmt.Errorf("DataGrid visual role: %w", normalizeErr)
		}
		*value = normalized
	}
	return roles, nil
}

func tableVisualRoleIDs(roles TableVisualRoles) []StyleID {
	return []StyleID{
		roles.Body, roles.Border, roles.Header, roles.CurrentHeader,
		roles.SortMarker, roles.CurrentCell, roles.SelectedRow,
		roles.CurrentRow, roles.CurrentSelectedRow, roles.Disabled,
		roles.Empty, roles.Loading, roles.Error, roles.ScrollbarPage,
		roles.ScrollbarArrow, roles.ScrollbarThumb,
		roles.ScrollbarFocusedThumb, roles.ScrollbarDisabled,
		roles.ScrollbarCorner, roles.ColumnsBand, roles.ColumnsAction,
		roles.ColumnsActionDefault, roles.ColumnsActionFocused,
		roles.ColumnsActionPressed, roles.ColumnsActionDisabled,
		roles.ColumnsActionMnemonic, roles.ColumnsActionShadow,
	}
}

func dataGridVisualRoleIDs(roles DataGridVisualRoles) []StyleID {
	return append(tableVisualRoleIDs(roles.TableVisualRoles),
		roles.Editable, roles.FocusedEdit, roles.InvalidEdit,
		roles.InvalidCharacter, roles.TextSelection,
	)
}

func visualRoleDigest(roles []StyleID) string {
	hash := sha256.New()
	var length [2]byte
	for _, role := range roles {
		binary.BigEndian.PutUint16(length[:], uint16(len(role)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(role))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func tableVisualRoleDigest(roles TableVisualRoles) string {
	return visualRoleDigest(tableVisualRoleIDs(roles))
}

func dataGridVisualRoleDigest(roles DataGridVisualRoles) string {
	return visualRoleDigest(dataGridVisualRoleIDs(roles))
}

func scrollBarRolesFromTable(roles TableVisualRoles) scrollBarVisualRoles {
	return scrollBarVisualRoles{
		page: roles.ScrollbarPage, arrow: roles.ScrollbarArrow,
		thumb: roles.ScrollbarThumb, focused: roles.ScrollbarFocusedThumb,
		disabled: roles.ScrollbarDisabled, corner: roles.ScrollbarCorner,
	}
}
