# File And Directory Pickers

File pickers are standard compound dialogs. Applications provide the model;
the toolkit provides bounded navigation, filtering, sorting, modal behavior,
Turbo Vision presentation, and typed results.

```go
provider, err := expletives.NewLocalFilePickerProvider(
    expletives.LocalFilePickerProviderOptions{Root: workspace},
)
if err != nil {
    return err
}

picker, err := expletives.NewFilePickerDialog(
    ctx,
    app.Root(),
    expletives.FilePickerDialogOptions{
        DialogOptions: expletives.DialogOptions{
            ModalPanelOptions: expletives.ModalPanelOptions{
                AutomationKey: "open.source",
                Title:         "Open Source File",
            },
        },
        Provider:         provider,
        InitialDirectory: provider.Root(),
        Filters: []expletives.FilePickerFilter{
            {Key: "go", Label: "Go source", Patterns: []string{"*.go"}},
            {Key: "all", Label: "All files", Patterns: []string{"*"}},
        },
        Filter: "go",
    },
)
if err != nil {
    return err
}
if err := picker.Show(nil); err != nil {
    return err
}

select {
case <-picker.Done():
    entry, accepted := picker.Selection()
    if accepted {
        // Open entry.Location using application policy.
    }
case <-ctx.Done():
    return ctx.Err()
}
```

Use `MultiFilePickerDialog.Selections` when the user must explicitly commit a
set, and `DirectoryPickerDialog.Selection` when the result is the directory
currently displayed. Never infer acceptance from the list's current row;
inspect the typed result only after `Done` closes.

For virtual, remote, archive, permission-filtered, or test data, implement
`FilePickerProvider`. Provider methods may block on their own services, but
must honor context cancellation. They are called outside toolkit locks and
serialized per picker. Return complete copied listings; do not mutate a
previously returned slice.

The local adapter is convenient, not a sandbox. Give it the narrowest useful
root and apply application authorization again before opening a returned
location. For deterministic tests, use an in-memory provider and assert both
the typed result and the associated semantic frame.

See [File And Directory Pickers API v0](specifications/file-pickers-api-v0.md)
for exact provider validation, keyboard behavior, result, concurrency,
privacy, and automation requirements.
