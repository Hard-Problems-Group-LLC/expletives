# Using Text Validation And Password Fields

`TextField` accepts an optional character validator. A validator always names
both an enforcement policy and whether its character set is an allow-list or
a deny-list.

## Soft Validation

Use soft validation when the user should be allowed to finish typing an
invalid value and correct it afterward:

```go
field, err := expletives.NewTextField(parent, expletives.TextFieldOptions{
    Text: "AB-123",
    Validator: &expletives.TextValidator{
        Enforcement: expletives.TextValidationSoft,
        Mode:        expletives.TextValidationWhitelist,
        Characters:  "ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789",
    },
})
```

A valid value is green. Once any invalid character exists, valid characters
are yellow and the invalid characters are red. `field.Valid()` and typed
snapshot details provide the same state without requiring a caller to infer
meaning from color.

## Hard Validation

Use hard validation when invalid typed characters should have no effect:

```go
field, err := expletives.NewTextField(parent, expletives.TextFieldOptions{
    Validator: &expletives.TextValidator{
        Enforcement: expletives.TextValidationHard,
        Mode:        expletives.TextValidationBlacklist,
        Characters:  "/\\:*?\"<>|",
    },
})
```

Typed or pasted blacklist matches are ignored. They never appear and do not
move the caret. A programmatic `SetText` containing a forbidden character
returns an error instead of silently changing the application's value.

The `Characters` string is a set of complete one-cell input elements, not a
regular expression. A composed character that occupies one cell is one
element. Multi-cell input follows the project's Limited Unicode policy and
becomes one `U+FFFD` replacement element before validation.

## Password Fields

Set `Password: true` to paint one `*` per value element:

```go
field, err := expletives.NewTextField(parent, expletives.TextFieldOptions{
    Password: true,
    Validator: &expletives.TextValidator{
        Enforcement: expletives.TextValidationSoft,
        Mode:        expletives.TextValidationBlacklist,
        Characters:  " ",
    },
})
```

Validation still examines the real text. Invalid positions therefore use the
same soft-validation colors, but the intended frame contains only asterisks.
Core and attached-automation snapshots redact the value and expose only
length, caret, edit state, and validity. The in-process `Text()` accessor
returns the actual value to the owning application.

Password mode does not encrypt application memory or replace responsible
secret handling. Clear secrets promptly and avoid copying them into logs,
commands, diagnostics, or application snapshots.

## Editing

A focused field does not capture bare text until Enter activates edit mode.
This leaves screen hotkeys usable while navigating controls.

- Enter starts editing, then later commits.
- Escape cancels the current edit.
- Left/Right and Home/End move the caret while editing.
- Backspace/Delete remove complete one-cell elements.
- Tab or directional focus movement commits before moving to another group.

Use an optional `ChangeCommand` to notify an MVC/MVVC controller after a user
commit. Programmatic setters do not emit that command.

## Numeric Fields

`NumberField` adds fixed decimal precision and optional inclusive bounds.
`SpinBox` adds a step and uses `[`/`]` outside edit mode:

```go
minimum, maximum := 0.0, 10.0
spin, err := expletives.NewSpinBox(parent, expletives.SpinBoxOptions{
    Value:         2.5,
    Minimum:       &minimum,
    Maximum:       &maximum,
    DecimalPlaces: 1,
    Step:          0.5,
})
```

Numeric options and programmatic values must be finite and already fit the
configured precision. Invalid intermediate text remains editable, but Enter
or Tab will not commit it or leave the field. Escape restores the committed
value. `Value()` exposes committed application state; typed details expose
both that value and the current working text.
