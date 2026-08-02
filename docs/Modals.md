# Modals

Phase 17 provides a single-loop, thread-safe modal stack. `ModalPanel` and
`Dialog` are ordinary Panel-derived containers: build their child controls and
Layouts normally, then call `Show`. The toolkit never starts a nested event
loop.

## ModalPanel And Dialog

Construct modals as direct children of `app.Root()`. Width and height are the
requested bordered body size; zero derives that axis from the recursive Layout
minimum. Position is always resolved by the App, so X and Y must remain zero.

```go
dialog, err := expletives.NewDialog(app.Root(), expletives.DialogOptions{
    ModalPanelOptions: expletives.ModalPanelOptions{
        PanelOptions: expletives.PanelOptions{
            AutomationKey: "settings.dialog",
            Bounds: expletives.Rect{Width: 44, Height: 12},
        },
        Title: "Settings",
    },
})
if err != nil {
    return err
}

// Add ordinary controls and a Layout before presentation.
if err := dialog.Show(initialField); err != nil {
    return err
}
```

`Show` is one-shot. It captures focus, closes open menus/popups, centers and
clamps the body in the Application Client Area, and limits input to the top
modal. `Close` records an explicit `ModalResult`; `Done` closes exactly once;
`Result` returns a copy. A worker or controller goroutine may wait on `Done`,
but a command callback must not block waiting for input that needs the current
serialized dispatch to finish.

Generic Dialog buttons use application commands and the normal command router.
Default and Cancel roles are unique across the complete dialog subtree. Enter
uses the applicable default Button after the focused control has first refusal;
Escape uses the applicable Cancel Button after editor/popup cancellation.

The default dialog palette is the Turbo Vision gray-dialog palette: white
active border/title and black body text on light gray. Buttons are raised
two-row controls with a green body and a black half-block drop shadow; focused
or pressed text is white, default text is bright cyan, and accelerators are
yellow. The Button shadow is distinct from the larger black dialog shadow.
Override the semantic `message_box`/`confirm_dialog`/`input_dialog`/
`progress_dialog`, `.border`, and `.shadow` styles for dialog surfaces, and
the `button`, `button.default`, `button.focused`, `button.pressed`,
`button.disabled`, `button.mnemonic`, and `button.shadow` styles for Buttons.

## MessageBox

`MessageBox` composes wrapped `StaticText`, an OK Button, and nested BoxLayouts.
Enter and Escape both produce an accepted `dialog.ok` result. No application
command router is required.

```go
box, err := expletives.NewMessageBox(app.Root(), expletives.MessageBoxOptions{
    DialogOptions: expletives.DialogOptions{
        ModalPanelOptions: expletives.ModalPanelOptions{
            PanelOptions: expletives.PanelOptions{
                AutomationKey: "saved.message",
            },
            Title: "Saved",
        },
    },
    Message: "The document was saved successfully.",
})
if err != nil {
    return err
}
return box.Show(nil)
```

## ConfirmDialog

`ConfirmDialog` has explicit Yes and No Buttons and an optional visible Cancel
Button. No is the safe default unless `Default: ConfirmChoiceYes` is deliberate.
Escape always returns Cancel and never aliases No, including when the Cancel
Button is hidden.

```go
confirm, err := expletives.NewConfirmDialog(
    app.Root(),
    expletives.ConfirmDialogOptions{
        DialogOptions: expletives.DialogOptions{
            ModalPanelOptions: expletives.ModalPanelOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "delete.confirm",
                },
                Title: "Confirm Delete",
            },
        },
        Message:    "Delete the selected record?",
        ShowCancel: true,
    },
)
if err != nil {
    return err
}
if err := confirm.Show(nil); err != nil {
    return err
}

go func() {
    <-confirm.Done()
    choice, ready := confirm.Choice()
    if ready {
        controller.HandleDeleteChoice(choice)
    }
}()
```

The toolkit-owned `dialog.ok`, `dialog.yes`, `dialog.no`, and `dialog.cancel`
commands are immutable and valid only for their matching standard-dialog
targets. Their completion snapshot already contains the closed lifecycle and
exact result, which gives automation a reliable barrier.

## InputDialog

`InputDialog` composes a wrapped prompt, one ordinary single-line `TextField`,
and OK/Cancel Buttons. It accepts the complete TextField validator and Password
policy. The editor is focused and active when `Show(nil)` returns. Enter first
commits the editor; a valid value accepts in that same publication, while an
invalid soft-validated value remains open and returns `validation_failed`.
Escape discards the edit, restores the initial value, and cancels.

```go
input, err := expletives.NewInputDialog(
    app.Root(),
    expletives.InputDialogOptions{
        DialogOptions: expletives.DialogOptions{
            ModalPanelOptions: expletives.ModalPanelOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "account.input",
                },
                Title: "Account",
            },
        },
        Prompt: "Account name:",
        Validator: &expletives.TextValidator{
            Enforcement: expletives.TextValidationSoft,
            Mode:        expletives.TextValidationWhitelist,
            Characters:  "abcdefghijklmnopqrstuvwxyz0123456789",
        },
    },
)
if err != nil {
    return err
}
if err := input.Show(nil); err != nil {
    return err
}

go func() {
    <-input.Done()
    if value, accepted := input.Value(); accepted {
        controller.SetAccount(value)
    }
}()
```

Automation can inspect the editor length, validity, enforcement mode, and
whitelist/blacklist mode, but it receives neither the active/accepted value
nor the validator character set. This compound-level privacy rule applies
whether or not Password is enabled.

## ProgressDialog

`ProgressDialog` keeps progress presentation and worker ownership separate.
The application supplies copied status and `ProgressBarState` updates. With
`Cancellable: true`, Cancel or Escape records a one-shot request and cancels
the stable context returned by `Context`, but the dialog stays open until the
worker/controller acknowledges the outcome through `Complete`.

```go
progress, err := expletives.NewProgressDialog(
    app.Root(),
    expletives.ProgressDialogOptions{
        DialogOptions: expletives.DialogOptions{
            ModalPanelOptions: expletives.ModalPanelOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "export.progress",
                },
                Title: "Exporting",
            },
        },
        State: expletives.ProgressDialogState{
            Status: "Starting export",
            Progress: expletives.ProgressBarState{
                Indeterminate: true,
                Status:        expletives.ProgressRunning,
            },
        },
        Cancellable: true,
    },
)
if err != nil {
    return err
}
if err := progress.Show(nil); err != nil {
    return err
}

go func() {
    err := controller.Export(progress.Context(), func(done, total uint64) {
        _ = progress.SetState(expletives.ProgressDialogState{
            Status: "Exporting records",
            Progress: expletives.ProgressBarState{
                Current: done, Total: total,
                Status: expletives.ProgressRunning,
            },
        })
    })
    result := expletives.ModalResult{Reason: expletives.ModalAccepted}
    if progress.CancelRequested() {
        result = expletives.ModalResult{
            Reason: expletives.ModalCancelled,
            Action: expletives.CommandDialogCancel,
        }
    } else if err != nil {
        result.Reason = expletives.ModalFailed
    }
    _ = progress.Complete(result)
}()
```

Repeated cancellation is disabled and rejected. Any terminal close cancels
the context to prevent a worker from outliving the dialog, but only an actual
user request sets `CancelRequested`.

## Nesting, Global Commands, And Automation

Nesting is explicit. Set `NestedOwner` to the current top modal; an unowned
modal cannot open over another modal. The maximum depth is eight. Lower modals
remain visible but cannot receive focus or activation.

Ordinary un-targeted global commands are blocked while a modal is active. Mark
only deliberate escape routes with `ModalPolicy: CommandModalAllowed`, such as
configured interrupt or quit commands. A direct command targeted outside the
top modal remains rejected even if that command is globally allowed.

Snapshots expose `modal_panel`, `dialog`, `message_box`, `confirm_dialog`,
`input_dialog`, or `progress_dialog` kind plus `ModalPanelDetails`: lifecycle,
stack position, focus identities,
requested/resolved/minimum geometry, degraded state, shadow policy, and copied
terminal result. Intended-frame cells remain authoritative for exact borders,
shadows, labels, focus rendering, and clipping.

The full normative contract is in
[`specifications/modals-api-v0.md`](specifications/modals-api-v0.md).
