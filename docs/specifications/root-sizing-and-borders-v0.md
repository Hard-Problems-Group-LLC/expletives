# Root Sizing And Border Contract v0

- Status: Implemented pre-v1 contract
- Authority: Direct operator instructions on 2026-07-30
- Scope: application surfaces, root constraints, Frame/GroupBox/Layout
  borders, snapshot semantics, and terminal projection

## Application Surface And Root

`AppOptions.Size` is the physical character-cell surface offered by the
container, headless host, or terminal presenter. The default
`RootConstraints{}` makes the special root Panel exactly fill that surface.
There is no preferred desktop width and no 240-column policy ceiling.

One materialized intended frame is bounded by the named allocation limit
`MaxFrameCells` rather than by a conventional width or height. The initial
limit is 4,194,304 cells, so 1200 by 1200 is an explicitly tested reasonable
desktop geometry. `MaxFrameWidth` and `MaxFrameHeight` are compatibility names
for the largest possible single axis when the other axis has one cell; new
code must reason about the aggregate cell count.

```go
type AspectRatio struct {
    Width  int
    Height int
}

type RootConstraints struct {
    Minimum     Size
    Maximum     Size
    AspectRatio AspectRatio
}

type AppOptions struct {
    Size            Size
    RootConstraints RootConstraints
    Theme           Theme
    RootStyle       StyleID
    Scenario        string
}
```

Constraint terms use terminal cells:

- a zero minimum axis imposes no requested minimum;
- a zero maximum axis is unbounded;
- an all-zero aspect ratio is disabled;
- otherwise both aspect-ratio terms must be positive; and
- a positive maximum must not be smaller than the corresponding minimum.

The root first takes the complete surface, then applies each positive maximum,
then selects the largest cell rectangle satisfying the aspect ratio within
that result. Fractional aspect results round to the nearest cell. The
resulting root is centered; any cells outside it retain the root semantic
style and are not part of the constrained root content rectangle.

The physical-width **Application Client Area** is independently established
by visible Main Menu/Header and Footer/Status Bar sections as specified by
[`application-chrome-v0.md`](application-chrome-v0.md). Root constraints
intersect that Application Client Area; they never redefine it. A
**Panel Client Area** is local to one Panel and subtracts that Panel's border
and any visible horizontal or vertical scrollbars before arranging children.

A minimum describes required content geometry but cannot enlarge a physical
container. When the offered surface is smaller, the root uses the available
geometry, exposes its declared `Minimum` in snapshots, and lets the existing
Layout overflow contract report insufficient child-layout space. Minimum and
maximum are policy, not a request to allocate an off-screen frame.

`App.RootConstraints` returns the current policy.
`App.SetRootConstraints` changes it atomically. A Transaction can batch
`SetRootConstraints` with `SetSize`, producing one arranged and rendered
publication. Direct `SetMinimumSize` on the special root is rejected so the
minimum cannot drift away from the root policy.

## Resource Limits

Fixed bounds protect checked arithmetic, memory, retained evidence, or the
automation wire; they are not desktop-size policy:

- `MaxFrameCells` bounds one materialized intended frame by aggregate cells;
- `MaxRetainedFrameCells` bounds App-owned retained frame history by actual
  cell cost;
- `MaxSnapshotHistoryRecords` bounds retained non-frame metadata when frames
  are very small; and
- automation separately bounds compact frame runs, response bytes, and
  retained encoded evidence.

Large frames therefore retain fewer historical generations rather than
multiplying the maximum frame allocation by a fixed record count.

## Independent Frame And Layout Borders

Borders are independently available on:

- `Frame` and `GroupBox` controls; and
- `BoxLayout` and `GridLayout` arrangement objects.

A visible Panel border is outside the Panel Client Area and currently removes
one cell from each edge. A future visible vertical scrollbar removes its
column, and a future visible horizontal scrollbar removes its row, after
border deduction. If both are visible, each is deducted exactly once and
their intersection belongs to scrollbar decoration rather than child
content.

This is deliberate. An enclosing Layout may own the visible border while
adjacent child Frames select `BorderNone`, avoiding doubled seams. A bordered
Frame inside a bordered Layout is also valid when two nested outlines are
actually desired. Layout decoration does not turn a Layout into a Control:
it has no control parent, focus, input, or event target and cannot parent a
control. Its owning Panel remains the control parent.

```go
type BorderForm string

const (
    BorderDefault     BorderForm = ""
    BorderNone        BorderForm = "none"
    BorderSingle      BorderForm = "single"
    BorderDouble      BorderForm = "double"
    BorderShadeLight  BorderForm = "shade_light"
    BorderShadeMedium BorderForm = "shade_medium"
    BorderShadeDark   BorderForm = "shade_dark"
    BorderBlock       BorderForm = "block"
)

type BorderOptions struct {
    Form       BorderForm
    Style      StyleID
    Foreground *Color
    Background *Color
}
```

The empty Frame or GroupBox form preserves the historical single-line
default. The empty Layout form means no Layout border. `BorderNone` draws
nothing and reserves no inset. Every other form is one cell thick and adds
two cells to the decorated object's measured width and height.

Frame and GroupBox use `BorderStyle`, `BorderForeground`, and
`BorderBackground` fields. Layouts embed `BorderOptions` in their construction
options. Empty semantic styles select `frame.border`, `group_box.border`, or
`layout.border`. A non-nil foreground or background pointer independently
overrides that component after Theme resolution; the other component and
attributes still come from the semantic style.

The canonical intended-frame glyphs are:

| Form | Canonical glyphs |
| --- | --- |
| `single` | `┌ ─ ┐ │ └ ┘` |
| `double` | `╔ ═ ╗ ║ ╚ ╝` |
| `shade_light` | `░` |
| `shade_medium` | `▒` |
| `shade_dark` | `▓` |
| `block` | `█` |

A Layout border is painted at that Layout's stack position before its
contents. Raising or lowering the Layout moves the border and the complete
Layout subtree together. Layout border cells retain the owning control ID
because a Layout has no `ControlID`; the corresponding `LayoutSnapshot`
identifies which Layout supplied the decoration.

`ControlSnapshot.Details.Border` and `LayoutSnapshot.Border` expose the form,
semantic and resolved styles, title where applicable, and optional color
overrides. The canonical snapshot does not change when a less capable
terminal uses a fallback.

## Terminal Capability Projection

The Linux presenter selects physical glyphs from the supported terminal
profile and active character locale:

- a UTF-8 locale emits the canonical Unicode line and shade characters;
- a supported non-UTF-8 xterm-family profile uses DEC Special Graphics for
  single lines and the closest line representation for double lines; and
- a terminal without either capability uses ASCII `+`, `-`, `|`, and `#`
  fallbacks.

Structured border fallbacks preserve the configured border colors because the
meaning is already known. This is distinct from unknown application text,
which follows the conspicuous black-on-yellow degradation policy in
[`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md).

## Verification

The executable contract is covered by:

- root default-fill, centered maximum/aspect, atomic resize, contradiction,
  and root-minimum tests;
- all border forms, no-border inset, independent foreground/background
  override, and Layout-owned-border seam tests;
- Unicode, DEC Special Graphics, and ASCII terminal encoder tests;
- a 1200 by 1200 allocation and compact automation round trip; and
- the human-runnable `expletives-test` border catalog and attached
  `expletivesctl` observation path.
