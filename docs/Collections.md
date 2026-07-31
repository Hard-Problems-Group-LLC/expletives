# Collections

Phase 16 collection controls use copied, bounded application models and stable
keys. `ListBox`, `DropDown`, and `ComboBox` are implemented. The formal shared
contract, including the planned TreeView, Table, and DataGrid, is
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
        {Key: "low", Label: "Low"},
        {Key: "normal", Label: "Normal"},
        {Key: "high", Label: "High"},
    },
    Selected:      "normal",
    PopupRows:     6,
    ChangeCommand: "ticket.priority.changed",
})
```

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

## Layout, Theme, And Automation

ListBox offers to stretch on both axes. DropDown and ComboBox offer to stretch
horizontally and remain one row high. A border adds the normal one-cell inset;
integrated scrollbars consume ListBox client cells only when their policies
and content require them. Current is always kept vertically visible when the
viewport has height.

Theme roles are `list_box`, `list_box.border`, `drop_down`,
`drop_down.focused`, `drop_down.disabled`, `drop_down.popup`,
`drop_down.popup_border`, `combo_box`, `combo_box.focused`,
`combo_box.disabled`, `collection.current`,
`collection.selected`, `collection.current_selected`,
`collection.disabled`, `collection.empty`, `collection.loading`, and
`collection.error`. Non-color row markers preserve current and selection
meaning.

Core typed details expose exact bounded state. Automation intentionally omits
retained item models. ListBox replaces status text and disabled reason with
bounded evidence; DropDown exposes compact popup geometry and stable
identities; ComboBox adds the exact bounded editor record. Pull the intended
frame for exact visible labels, markers, and semantic styles. Raw automation
key events exercise the same current, provisional selection, commit/cancel,
editing, scrolling, and activation paths as a terminal user. The catalog page
is reachable at Controls / Collections with stable keys `collections.list`,
`collections.drop-down`, and `collections.combo`.
