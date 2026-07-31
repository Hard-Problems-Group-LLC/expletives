# Text And Display API v0

- Status: Implemented pre-v1 contract
- Authority: Direct operator instruction to proceed automatically in roadmap
  order on 2026-07-30
- Scope: `Label`, `StaticText`, `Separator`, and `Rule`

## Control Model

The four Text/Display types are concrete, copy-safe, non-focusable leaf
controls. Each:

- requires one real `Container` parent at construction;
- implements the sealed `Control` capability but not `Container`;
- participates in Box/Grid Layouts through ordinary Control identity,
  geometry, minima, visibility, style, stacking, and lifetime behavior;
- cannot accept children or own a Layout; and
- is safe for concurrent use under the existing App/Transaction contract.

This is the first executable proof that shared Panel-like window behavior does
not imply that every control is a public container. External compound
controls may still compose/embed real toolkit handles; they cannot forge
toolkit identity.

## Shared Text Values

```go
type TextAlignment string

const (
    TextAlignDefault TextAlignment = ""
    TextAlignStart   TextAlignment = "start"
    TextAlignCenter  TextAlignment = "center"
    TextAlignEnd     TextAlignment = "end"
)

type TextWrap string

const (
    TextWrapDefault TextWrap = ""
    TextWrapNone    TextWrap = "none"
    TextWrapWords   TextWrap = "words"
    TextWrapCells   TextWrap = "cells"
)
```

Empty alignment normalizes to Start. Empty wrapping normalizes to None.
Horizontal Start and End mean left and right. Vertical Start and End mean top
and bottom.

Displayed source text and its normalized form are each limited to
`MaxDisplayTextBytes` (256 UTF-8 bytes), at most `MaxDisplayTextCells` (256)
canonical cells, and `MaxCellBytes` per cell. These named bounds keep
worst-case typed snapshots and automation responses within the version 1
resource proof. A later larger-document or streaming control must use a
bounded content model rather than silently multiplying unbounded strings by
`MaxControls` and retained snapshots.

The project-owned one-cell policy in
[`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md) applies before
measurement or painting. Supported composed graphemes remain one cell.
Unsupported, zero-width, multi-cell, and indeterminate elements become one
`U+FFFD` cell. NUL and non-newline control characters are rejected.
`StaticText` accepts LF and CRLF line breaks; other controls are single-line.

Alignment is calculated from logical control bounds, not the current ancestor
clip, so clipping never reflows or shifts content. If unwrapped content is
wider or taller than the control, Start shows the beginning, Center shows the
middle, and End shows the end. Wrapping uses the logical width before
vertical alignment and clipping.

Word wrapping breaks at ASCII space where possible, removes the break-space
at the new line boundary, and cell-wraps a word longer than the width. Cell
wrapping chunks strictly by canonical cell count. Neither mode splits a
grapheme.

## Label

```go
type LabelOptions struct {
    PanelOptions
    Text                string
    HorizontalAlignment TextAlignment
    VerticalAlignment   TextAlignment
    Target              Control
    Mnemonic            Key
}

func NewLabel(parent Container, options LabelOptions) (*Label, error)
func (t *Transaction) NewLabel(parent Container, options LabelOptions) (*Label, error)
func (l *Label) Text() string
func (l *Label) SetText(text string) error
```

`Label` is single-line and does not wrap. `Target` is optional; `Mnemonic` is
also optional but requires a target. A mnemonic is one lowercase ASCII letter
or digit; uppercase input normalizes to lowercase.
The target must be a live or same-Transaction provisional Control in the same
App. The association is observable but does not focus or activate the target
until the Actions/focus phase defines that event path. Destroying the target
clears the association without destroying the Label.

When `PanelOptions.MinimumSize` is zero, Label derives its minimum from the
normalized text width by one row.

## StaticText

```go
type StaticTextOptions struct {
    PanelOptions
    Text                string
    HorizontalAlignment TextAlignment
    VerticalAlignment   TextAlignment
    Wrap                TextWrap
}

func NewStaticText(parent Container, options StaticTextOptions) (*StaticText, error)
func (t *Transaction) NewStaticText(parent Container, options StaticTextOptions) (*StaticText, error)
func (s *StaticText) Text() string
func (s *StaticText) SetText(text string) error
```

`StaticText` accepts multiple logical lines. None preserves source lines;
Words wraps at spaces with cell fallback; Cells wraps at every logical width.
With an automatic zero minimum, unwrapped text uses its longest line and line
count. Wrapped text uses one cell of width and the source line count, allowing
the Layout to assign a narrower wrapping width. Applications may declare a
different explicit minimum normally.

## Separator And Rule

```go
type SeparatorOptions struct {
    PanelOptions
    Orientation Orientation
    Form        BorderForm
}

type RuleOptions struct {
    PanelOptions
    Orientation Orientation
    Form        BorderForm
    Text        string
    Alignment   TextAlignment
}
```

`Separator` is an untitled divider. `Rule` is the titled form and provides
`Text`/`SetText`. Both accept Horizontal or Vertical orientation and the same
none, single, double, light/medium/dark shade, and full-block forms as other
structural decoration. Empty form means single. Horizontal single/double
rules use `─`/`═`; vertical rules use `│`/`║`; shade forms use their canonical
full-cell glyph.

A Rule places one padding cell before and after nonempty text when space
allows. Horizontal rules lay text left-to-right; vertical rules lay its
graphemes top-to-bottom. Alignment selects placement along that main axis.
The terminal presenter uses the established Unicode, DEC Special Graphics,
or ASCII structural fallback without changing snapshot geometry.

An automatic Separator minimum is one by one. An automatic horizontal Rule
minimum is text width plus two by one; a vertical Rule reverses those axes.

## Mutation, Snapshots, And Automation

`Label.SetText`, `StaticText.SetText`, and `Rule.SetText` use one-operation
Transactions. `Transaction.SetText` supports batching. Text is normalized and
copied before recording; failed validation changes nothing. When a display
control still uses its automatic minimum, text mutation recomputes that
minimum. Any explicit `SetMinimumSize` permanently selects caller-owned
minimum policy for that control.

`ControlDetails` gains mutually meaningful typed `Text` and `Divider`
members. Text details expose canonical text, alignment, wrap, target ID, and
mnemonic. Divider details expose orientation, form, canonical title, and
alignment. Automation projects and validates the same bounded values. Visible
cells remain the authoritative observation of wrapping, alignment, clipping,
and terminal-independent degradation.

## Verification

Required coverage includes:

- empty, narrow, aligned, clipped, word-wrapped, and cell-wrapped cases;
- supported precomposed and composed one-cell text;
- exact one-cell replacement of wide and zero-width input;
- target/mnemonic validation and target destruction;
- horizontal/vertical and every divider form;
- automatic and explicit minima plus atomic text mutation;
- immutable core and automation snapshot details;
- a complete `expletives-test` scene and attached frame inspection; and
- the normal format, vet, unit, integration, PTY, race, all-mode build, and
  smoke gate.
