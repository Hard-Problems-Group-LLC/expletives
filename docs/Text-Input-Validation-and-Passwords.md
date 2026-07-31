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
- Shift with movement extends a selection; Ctrl-A selects the complete value.
- Typing, committed text, or paste replaces a selection.
- Backspace/Delete remove a selection or complete one-cell elements.
- Tab or directional focus movement commits before moving to another group.

Ctrl-C remains the application's configurable interrupt chord while editing;
the toolkit does not silently reinterpret it as clipboard copy.

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

## Form Layout

An ordinary text, number, or spin field is one row high and has no implicit
frame. Give it a practical minimum width—usually 10 to 30 cells—and place a
separate left-aligned bound `Label` in the preceding form column. The classic
white-on-blue field background fills its complete arranged width, including
blank cells, against a light-neutral dialog/form surface. Keeping those
physical palette classes distinct avoids both surfaces quantizing to the same
terminal blue.

Single-line fields default to horizontal stretch and natural vertical size.
`TextArea` defaults to stretch in both directions. `BoxLayout` preserves
natural rows and configured gaps, then divides remaining space among
stretch-capable controls by their construction-time axis weights:

```go
area, err := expletives.NewTextArea(parent, expletives.TextAreaOptions{
    PanelOptions: expletives.PanelOptions{
        MinimumSize: expletives.Size{Width: 20, Height: 3},
        LayoutHints: expletives.LayoutHints{
            VerticalWeight: 2,
        },
    },
})
```

Use `LayoutSizeNatural` or `LayoutSizeStretch` in either axis to override a
control kind's default. An explicit Layout-item alignment or positive `Grow`
remains authoritative. Wrapping a one-row field in a one-cell bordered Frame
produces a three-row compound minimum.

## Multiline TextArea

`TextArea` uses the same optional validator and password rules:

```go
area, err := expletives.NewTextArea(parent, expletives.TextAreaOptions{
    Text: "First line\nSecond line",
    Wrap: expletives.TextWrapWords,
    Validator: &expletives.TextValidator{
        Enforcement: expletives.TextValidationSoft,
        Mode:        expletives.TextValidationBlacklist,
        Characters:  "@",
    },
})
```

Enter starts editing and then inserts new lines; Ctrl-Enter commits. Escape
cancels, Tab commits and traverses focus groups, arrows navigate visual rows,
and Shift extends selection. CRLF and CR normalize to LF. The internal
row/column viewport keeps the caret visible with no-wrap, word-wrap, and
cell-wrap policies.

`App.DispatchTextInput` delivers bounded committed text or paste directly to
the focused editing control. The content never enters key, menu, mnemonic, or
command resolution. Candidate values over the 65,536-byte or element limit
are rejected atomically. The terminal adapter enables bracketed paste,
buffers at most that bound, and discards an oversized episode in full.
