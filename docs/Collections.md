# Collections

Phase 16 collection controls use copied, bounded application models and stable
keys. `ListBox` is the first implemented control. The formal shared contract,
including the planned DropDown, ComboBox, TreeView, Table, and DataGrid, is
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

## Layout, Theme, And Automation

ListBox offers to stretch on both axes. A border adds the normal one-cell
inset; integrated scrollbars consume client cells only when their policies and
content require them. Current is always kept vertically visible when the
viewport has height.

Theme roles are `list_box`, `list_box.border`, `collection.current`,
`collection.selected`, `collection.current_selected`,
`collection.disabled`, `collection.empty`, `collection.loading`, and
`collection.error`. Non-color row markers preserve current and selection
meaning.

Core `ControlDetails.ListBox` exposes exact bounded status text plus compact
model/selection/viewport evidence. Automation intentionally omits the item
model and replaces status text and disabled reason with bounded byte-count and
digest evidence. Pull the intended frame for exact visible labels, markers,
and semantic styles. Raw automation key events exercise the same current,
selection, scrolling, and activation paths as a terminal user.
