# Collections

Phase 16 collection controls use copied, bounded application models and stable
keys. `ListBox`, `DropDown`, `ComboBox`, `TreeView`, `Table`, and `DataGrid`
are implemented. The formal shared contract is
[`specifications/collections-api-v0.md`](specifications/collections-api-v0.md).

## Construct A ListBox

```go
list, err := expletives.NewListBox(
    panel,
    expletives.ListBoxOptions{
        ScrollablePanelOptions: expletives.ScrollablePanelOptions{
            ScrollViewOptions: expletives.ScrollViewOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "jobs.list",
                },
                ChangeCommand: "jobs.selection.changed",
            },
            BorderForm:    expletives.BorderSingle,
            HorizontalBar: expletives.ScrollBarVisibilityAuto,
            VerticalBar:   expletives.ScrollBarVisibilityAuto,
        },
        Items: []expletives.ListItem{
            {Key: "queued", Label: "Queued", Description: "3 jobs"},
            {Key: "running", Label: "Running", Description: "1 job"},
            {
                Key: "archived", Label: "Archived", Disabled: true,
                DisabledReason: "Archive is offline",
            },
        },
        SelectionMode:    expletives.CollectionSelectionSingle,
        RequireSelection: true,
        ActivateCommand:  "jobs.open",
    },
)
if err != nil {
    return err
}
```

Keys are bounded identifiers and must be unique. Labels are required;
descriptions are optional. Enabled rows cannot have a disabled reason, while
disabled rows require one. The toolkit copies all inputs and every getter
returns caller-owned values.

An empty selection mode means single selection. The initial current row is the
first enabled item unless `Current` names another enabled key. Selection is
independent: omit `Selected` for no initial selection, or set
`RequireSelection` to make repair select current whenever enabled rows exist.

## Keyboard And Commands

Up/Down, Page Up/Page Down, Home, and End move current without changing
selection. Disabled rows are skipped. Space selects current in single mode or
toggles it in multiple mode. Enter selects current when necessary and then
activates it. Horizontal arrows scroll long labels and descriptions. Tab and
Shift-Tab retain their application-wide focus-group meaning.

Only a real user selection change routes the embedded `ChangeCommand`.
Enter and `Activate` route `ActivateCommand`; when it is empty, a selection
change may route the change command. Programmatic setters and Transactions are
silent. Routers run outside toolkit locks and may safely inspect `State()`.

## Publish Model State From MVC/MVVM Code

Use `SetItems` when a refreshed model should preserve surviving current and
selected keys. If current vanished, repair starts at its former row, scans
forward, then backward. Missing and newly disabled selections are removed.

Use `Replace` when the controller has an exact model, current, and selection:

```go
if err := list.Replace(rows, currentKey, selectedKeys); err != nil {
    return err
}
```

Use one Transaction to publish several view-model changes in one frame:

```go
tx := app.NewTransaction()
if err := tx.SetListItems(list, rows); err != nil {
    return err
}
if err := tx.SetListStatus(list, expletives.CollectionReady, ""); err != nil {
    return err
}
if err := tx.Commit(ctx); err != nil {
    return err
}
```

Workers may call direct methods concurrently. Transaction builders themselves
remain single-caller objects.

`CollectionLoading` accepts a custom message or defaults to `Loading...`.
`CollectionError` requires a message. Status changes keep the retained model
but loading and error controls are not focusable. Returning to ready restores
ordinary eligibility.

## Collapsed Popup Fields

Use `DropDown` when the value must come from one stable-key item:

```go
priority, err := expletives.NewDropDown(panel, expletives.DropDownOptions{
    PanelOptions: expletives.PanelOptions{
        AutomationKey: "ticket.priority",
    },
    Items: []expletives.ListItem{
        {
            Key: "low", Label: "Low", Indicator: "■",
            IndicatorStyle: "priority.low",
        },
        {Key: "normal", Label: "Normal"},
        {Key: "high", Label: "High"},
    },
    Selected:      "normal",
    PopupRows:     6,
    ChangeCommand: "ticket.priority.changed",
})
```

An optional `Indicator`/`IndicatorStyle` pair adds one independently styled
leading cell while retaining the required textual label. The indicator style
must exist in the App Theme; construction, item replacement, and Theme
replacement reject a missing reference atomically. Current, selected, and
disabled styles continue across the rest of the row, so a named color swatch
can remain accurate without making color the only cue. DropDown repeats the
selected indicator when collapsed. ComboBox keeps the editable field text and
shows indicators only in its choices popup.

Space, Enter, F4, and Alt-Down open DropDown choices. Navigation and Space
change only provisional popup state. Enter commits; Escape, Tab, or a focus
transfer cancels. The popup measures its items, backsets at the right edge,
and moves above the field when it cannot fit below. `AllowEmpty` permits a
provisional Space press to clear selection.

Use `ComboBox` for the same stable choices plus custom single-line text:

```go
assignee, err := expletives.NewComboBox(panel, expletives.ComboBoxOptions{
    DropDownOptions: expletives.DropDownOptions{
        PanelOptions: expletives.PanelOptions{
            AutomationKey: "ticket.assignee",
        },
        Items: []expletives.ListItem{
            {Key: "alice", Label: "Alice"},
            {Key: "bob", Label: "Bob"},
        },
        Selected:      "alice",
        ChangeCommand: "ticket.assignee.changed",
    },
    Validator: &expletives.TextValidator{
        Enforcement: expletives.TextValidationSoft,
        Mode:        expletives.TextValidationBlacklist,
        Characters:  "<>|",
    },
})
```

Enter or F2 starts ComboBox editing. Space, F4, or Alt-Down opens choices.
Choosing an item copies its label into the editor. A committed custom value
clears selected identity unless it exactly equals an enabled item label. An
empty ComboBox is still editable. Hard validators reject models containing
enabled labels that they would prohibit, which keeps popup selection and
editor validation consistent.

`Items`, `State`, `Text`, and `Validator` return copied values. `SetItems`
preserves surviving current and selected keys; `SetSelection` copies the
selected label into ComboBox text. `SetText` derives selection by exact label.
Transactions provide `SetDropDownItems`, `SetDropDownSelection`,
`SetComboBoxText`, and `SetComboBoxValidator` for atomic MVC/MVVM updates.
Only one collection popup is open at a time per App, and it uses the ordinary
event loop rather than a nested modal loop.

## Hierarchies With TreeView

TreeView copies a recursive caller model into a bounded internal preorder
representation. Keys are unique across the complete tree, branches and leaves
use the same selection model, and disabled nodes remain visible but cannot
become current, selected, or activated.

```go
tree, err := expletives.NewTreeView(panel, expletives.TreeViewOptions{
    ScrollablePanelOptions: expletives.ScrollablePanelOptions{
        ScrollViewOptions: expletives.ScrollViewOptions{
            PanelOptions: expletives.PanelOptions{
                AutomationKey: "project.tree",
            },
            ChangeCommand: "project.selection.changed",
        },
        BorderForm:  expletives.BorderSingle,
        VerticalBar: expletives.ScrollBarVisibilityAuto,
    },
    Nodes: []expletives.TreeNode{{
        Key: "workspace", Label: "Workspace", Expanded: true,
        Children: []expletives.TreeNode{
            {Key: "docs", Label: "Documentation"},
            {
                Key: "source", Label: "Source",
                Children: []expletives.TreeNode{
                    {Key: "core", Label: "Core toolkit"},
                    {Key: "terminal", Label: "Terminal adapter"},
                },
            },
        },
    }},
    SelectionMode:   expletives.CollectionSelectionMultiple,
    Selected:        []string{"workspace"},
    ActivateCommand: "project.open",
    ExpandCommand:   "project.expansion.changed",
})
```

Up and Down move through enabled visible nodes without changing selection.
Right expands a collapsed branch or enters its first enabled child; Left
collapses an expanded branch or returns to the nearest enabled parent. Space
changes selection and Enter activates. `+` and `-` expand or collapse the
current branch; `*` recursively expands it. Tab and Shift-Tab leave the tree
as one focus group.

Only actual keyboard expansion changes route `ExpandCommand`. Programmatic
`SetExpanded`, `SetNodeExpanded`, and Transaction equivalents are silent.
Collapsed descendants retain selection. If a programmatic collapse hides
current, current moves to the nearest enabled visible ancestor.

`SetNodes` preserves current, selection, and expansion for surviving keys.
New branches use their copied `Expanded` flags; surviving collapsed branches
stay collapsed even if replacement flags differ. Use `Replace` or
`Transaction.ReplaceTree` when a view model owns the exact current, selected,
and expanded key sets. A nil `TreeViewOptions.Expanded` uses node flags, while
a non-nil slice overrides them exactly.

## Read-Only Tables

Table owns a copied canonical row model and derives its displayed order from
one optional stable sortable column. The sticky header remains visible while
rows scroll:

```go
table, err := expletives.NewTable(panel, expletives.TableOptions{
    ScrollablePanelOptions: expletives.ScrollablePanelOptions{
        ScrollViewOptions: expletives.ScrollViewOptions{
            PanelOptions: expletives.PanelOptions{
                AutomationKey: "jobs.table",
            },
            ChangeCommand: "jobs.selection.changed",
        },
        BorderForm:  expletives.BorderSingle,
        VerticalBar: expletives.ScrollBarVisibilityAuto,
    },
    Columns: []expletives.Column{
        {Key: "name", Header: "Name", MinimumWidth: 12, Grow: 2, Sortable: true},
        {Key: "state", Header: "State", Width: 10, Sortable: true},
        {Key: "count", Header: "Count", Width: 6, Alignment: expletives.TextAlignEnd},
    },
    Rows: []expletives.TableRow{
        {Key: "build", Cells: []expletives.TableCell{
            {Column: "name", Text: "Build"},
            {Column: "state", Text: "Running"},
            {Column: "count", Text: "3"},
        }},
    },
    Features: []expletives.TableFeature{expletives.TableFeatureColumns},
    SelectionStyle:  expletives.TableSelectionRange,
    FocusMode:        expletives.TableFocusCell,
    RequireSelection: true,
    RangeAnchor:      "build",
    RangeExtent:      "build",
    ColumnPresentation: []expletives.TableColumnPresentation{
        {Column: "name", Visible: true, Wrap: expletives.TableColumnHang},
        {Column: "state", Visible: true, Wrap: expletives.TableColumnClip},
        {Column: "count", Visible: true, Wrap: expletives.TableColumnClip},
    },
    ActivateCommand:  "jobs.open",
    SortCommand:      "jobs.sort.changed",
})
```

Up/Down, Page Up/Page Down, Home, and End move current by enabled displayed
row. In cell mode Left and Right move the current column; in row mode they
scroll horizontally. Space changes stable row selection, Enter activates,
and Ctrl-Home/Ctrl-End move to the first/last enabled row and boundary column.
Press `S` on a sortable current column to cycle ascending, descending, and
canonical model order. Sorting is stable and never changes `Rows()` order.

Choose exactly one `SelectionStyle`: None, Single, Range, or Multiple. Range
retains stable anchor/extent row keys and derives the continuous enabled
displayed interval; Shift-navigation and `[`/`]` extend it. None leaves
current navigation and activation available without retaining selection.
`SetSelectionPolicy` replaces style, requirement, selected keys, and endpoints
atomically.

`ColumnPresentation` independently owns complete display order, visibility,
and Clip/Wrap/Hang body presentation without changing canonical `Columns()`
or `Rows()`. Wrap breaks words and Hang indents continuations by one space.
`SetColumnPresentation` replaces the exact copied presentation. With
`TableFeatureColumns`, a right-justified `&Columns...` action opens the
keyboard-complete modal editor for these choices.

Use `SetRows` for ordinary controller refreshes. Use `SetModel` when the
column schema changes while surviving identities should be preserved, or
`Replace` when the view model owns exact current, selection, and sort state;
use `ReplaceWithPresentation` when it also owns exact presentation.
All direct methods are concurrency-safe and Transaction forms publish related
view-model changes in one frame. Programmatic updates are silent; only actual
keyboard selection, activation, and sort changes route their respective
commands.

The dedicated catalog Table is `tables.control` under Controls / Tables. Its
Options and Colors Notebook pages exercise live policies, presentation,
per-instance semantic roles, and reset. Its typed automation record is
compact: exact model content stays available through the in-process copied
`Columns` and `Rows` APIs, while automation reports counts, identities,
presentation/width/role digests, sort, Columns state, commands, and viewport.
Pull the frame to inspect exact visible headers, cells, wrapping, markers,
styles, sticky placement, and clipping. At ordinary wide sizes, its flexible
Summary column consumes the remaining viewport width; narrow screens retain
intentional horizontal overflow for scrolling coverage.

## Editable Data Grids

DataGrid uses the same copied row/column model, stable current and selection,
sorting, sticky header, and viewport behavior as Table, but always uses cell
focus and can edit columns marked `Editable`:

```go
grid, err := expletives.NewDataGrid(panel, expletives.DataGridOptions{
    ScrollablePanelOptions: expletives.ScrollablePanelOptions{
        ScrollViewOptions: expletives.ScrollViewOptions{
            PanelOptions: expletives.PanelOptions{
                AutomationKey: "jobs.grid",
            },
            ChangeCommand: "jobs.changed",
        },
        BorderForm:  expletives.BorderSingle,
        VerticalBar: expletives.ScrollBarVisibilityAuto,
    },
    Columns: []expletives.Column{
        {
            Key: "name", Header: "Name", Grow: 2,
            Sortable: true, Editable: true,
            Validator: &expletives.TextValidator{
                Enforcement: expletives.TextValidationSoft,
                Mode:        expletives.TextValidationWhitelist,
                Characters:  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 ",
            },
        },
        {Key: "state", Header: "State", Width: 10, Editable: true},
        {Key: "count", Header: "Count", Width: 6},
    },
    Rows: []expletives.TableRow{{
        Key: "build",
        Cells: []expletives.TableCell{
            {Column: "name", Text: "Build"},
            {Column: "state", Text: "Running"},
            {Column: "count", Text: "3"},
        },
    }},
    CurrentRow: "build", CurrentColumn: "name",
})
```

Enter or F2 begins editing an editable current cell. Enter commits and Escape
cancels. Tab or Shift-Tab commits a valid value and moves directly to the next
or previous editable cell, retaining edit traversal inside the grid; at the
boundary it leaves the DataGrid focus group. Soft validation displays the
whole active value as invalid and its offending cells distinctly, and blocks
commit until corrected. Hard validation ignores disallowed input. Enter on a
read-only cell performs ordinary collection activation.

Every programmatic mutation cancels the active editor and is silent. A user
commit updates the grid's copied model before `ChangeCommand` is invoked
outside the App lock, so an MVC/MVVM controller can call `Rows`/`State`,
persist or reject the edit, and atomically publish a replacement. The same
command also reports row-selection changes. Programmatic role replacement
also cancels the editor and validates the complete embedded Table plus
DataGrid editor-role mapping against the staged Theme. The dedicated catalog
fixture is `data-grid.control` under Controls / DataGrid.

## Layout, Theme, And Automation

ListBox, TreeView, Table, and DataGrid offer to stretch on both axes. DropDown
and ComboBox offer to stretch horizontally and remain one row high. A border
adds the normal one-cell inset; integrated scrollbars consume collection client cells only
when their policies and content require them. Current is always kept
vertically visible when the viewport has height.

Theme roles are `list_box`, `list_box.border`, `tree_view`,
`tree_view.border`, `tree.guide`, `tree.branch`, `tree.expanded`, `table`,
`table.border`, `table.header`, `table.header_current`, `table.sort`,
`table.cell_current`, `table.row_selected`, `data_grid`,
`data_grid.border`, `data_grid.edit`,
`data_grid.edit_focused`, `data_grid.edit_invalid`,
`data_grid.edit_invalid_character`, `drop_down`,
`drop_down.focused`, `drop_down.disabled`, `drop_down.popup`,
`drop_down.popup_border`, `combo_box`, `combo_box.focused`,
`combo_box.disabled`, `collection.current`,
`collection.selected`, `collection.current_selected`,
`collection.disabled`, `collection.empty`, `collection.loading`, and
`collection.error`. Non-color row markers preserve current and selection
meaning.

Table and DataGrid may replace those defaults with a fixed per-instance
`TableVisualRoles` or `DataGridVisualRoles` value. Empty fields normalize to
compatible defaults; nonempty StyleIDs must exist in the Theme selected by
the same Transaction. Theme remains the resolved-color owner, and role
changes never affect an unrelated control instance.

Core typed details expose exact bounded state. Automation intentionally omits
retained item, recursive node, table/grid models, active grid edit text, and
grid validator character sets. ListBox, TreeView, Table, and DataGrid
replace status text and disabled reason with bounded evidence; TreeView also publishes
selection and expansion digests. DropDown exposes compact popup geometry and
stable identities; ComboBox adds the exact bounded editor record. Pull the
intended frame for exact visible labels, markers, and semantic styles. Raw
automation key events exercise the same current, provisional selection,
expansion, commit/cancel, editing, scrolling, Columns dialog, and activation
paths as a terminal user. Controls / Collections retains stable keys
`collections.list`, `collections.tree`, `collections.drop-down`, and
`collections.combo`; Controls / Tables and Controls / DataGrid use
`tables.control` and `data-grid.control` with their own stable Notebook,
Options, and Colors descendants.
