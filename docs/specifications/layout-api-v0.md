# Foundational Layout API v0

- Status: Directed implementation contract
- Authority: `EXPL-DEC-007` and `EXPL-DEC-011`
- Scope: Basic `BoxLayout`, `GridLayout`, stacking, and overflow

## Model

A Layout is a copy-safe handle over toolkit-owned arrangement, stacking, and
optional border-decoration state. It is not a Control, cannot receive focus
or input, and never becomes a control parent.

Controls retain their immutable control-tree parent. A Layout tree is a
separate arrangement and stacking tree:

- one owning Panel may attach one primary top-level Layout with
  `Panel.SetLayout`;
- `Panel.AddLayout` may attach another top-level Layout to the same Panel as a
  separate stacking context;
- a Layout may contain Panels and nested Layouts;
- every Panel item must be a direct control child of the owning Panel;
- one Panel may be an item in at most one Layout;
- one Layout may have exactly one Layout parent or one owning Panel; and
- a nested Layout and all of its descendants share the top-level Layout's
  owning Panel.

`SetLayout`, `AddLayout`, and nested `AddLayout` do not reparent Controls.
Top-level Layouts each receive the owning Panel's complete **Panel Client
Area**: its bounds less its border and any visible horizontal or vertical
scrollbars. A top-level root Layout instead receives the constrained root
content rectangle, which is the intersection of root constraints and the
physical-width Application Client Area. This permits independent overlay
contexts while ordinary applications use one top-level Box or Grid.

Frame/GroupBox borders and Layout borders are independent. In particular, an
enclosing Layout may own one outline while adjacent Frames select
`BorderNone`, preventing doubled seams. Layout decoration remains owned by the
Layout's stacking context; it does not change control parentage.

## Bounds And Ordering

Arrangement order and paint/Z-order are independent.

- Insertion order determines Box and Grid measurement and arrangement.
- A separate stack order determines paint traversal.
- `Panel.Raise()` moves a managed Panel to the highest Panel stack slot in its
  Layout.
- `Panel.Lower()` moves it to the lowest Panel stack slot in its Layout.
- `Layout.Raise()` moves a Layout and its complete subtree to the highest
  Layout stack slot under the same parent.
- `Layout.Lower()` moves that subtree to the lowest Layout stack slot.
- Moving one peer kind preserves all stack slots and relative ordering of the
  other peer kind.
- Calling `Raise` or `Lower` at the requested extreme succeeds without
  publishing a redundant snapshot.

Later stack entries paint over earlier entries. Raise/lower never changes
measured minima, arranged rectangles, control parentage, focus, or Layout
membership.

A Panel that is not a Layout item cannot use `Raise` or `Lower`; the operation
returns `ErrNotLayoutMember`. Unmanaged direct children remain legal bootstrap
or overlay controls and paint after attached Layout trees in stable
control-child insertion order.

## Public Types

The root package supplies these bounded scalar types and constants:

```go
type LayoutID string
type LayoutKind string

const (
    LayoutBox  LayoutKind = "box"
    LayoutGrid LayoutKind = "grid"
)

type Orientation uint8

const (
    Horizontal Orientation = iota
    Vertical
)

type Alignment uint8

const (
    AlignStretch Alignment = iota
    AlignStart
    AlignCenter
    AlignEnd
)

type Insets struct {
    Top, Right, Bottom, Left int
}

type LayoutItemOptions struct {
    Insets         Insets
    Grow           int
    HorizontalAlign Alignment
    VerticalAlign   Alignment
}

type BorderOptions struct {
    Form       BorderForm
    Style      StyleID
    Foreground *Color
    Background *Color
}
```

All inset, gap, row, column, and grow values are checked nonnegative integers.
The zero alignment is Stretch. Dimensions and arithmetic remain within the
project's checked geometry bounds.

Resource limits are:

```go
const (
    MaxLayouts          = 1024
    MaxLayoutItems      = MaxControls
    MaxLayoutDepth      = 32
    MaxOverflowHandlers = 1
)
```

`MaxLayoutItems` is an App-wide aggregate across attached Layouts. Layout
depth includes the top-level Layout.

## Layout Interface

```go
type Layout interface {
    ID() LayoutID
    AutomationKey() string
    Kind() LayoutKind
    Bounds() Rect
    MinimumSize() Size
    Owner() Container
    ParentLayout() Layout
    Raise() error
    Lower() error

    layoutState() *layoutState
}
```

The unexported method seals canonical identity. IDs are assigned atomically at
attachment and are not reused. Identity and the automation key remain
readable after the owning Panel is destroyed; mutation then returns
`ErrDestroyed`.

`Bounds` is relative to the parent Layout's content origin, or to the owner
Panel's client origin for a top-level Layout. `MinimumSize` is available while
detached and is recomputed from the immutable builder tree.

## Construction

```go
type BoxLayoutOptions struct {
    AutomationKey string
    Gap           int
    Insets        Insets
    Border        BorderOptions
}

func NewBoxLayout(
    orientation Orientation,
    options BoxLayoutOptions,
) (*BoxLayout, error)

type GridLayoutOptions struct {
    AutomationKey string
    Rows          int
    Columns       int
    HorizontalGap int
    VerticalGap   int
    Insets        Insets
    Border        BorderOptions
}

func NewGridLayout(options GridLayoutOptions) (*GridLayout, error)
```

At least one Grid dimension is positive. When one is zero, item count derives
it from the positive dimension. When both are positive, their product is the
fixed capacity.

Both concrete Layout types provide:

```go
AddPanel(panel Control, options LayoutItemOptions) error
AddLayout(layout Layout, options LayoutItemOptions) error
```

Builder operations are safe for concurrent callers but are rejected after
the Layout tree attaches. `AddLayout` rejects cycles, multiple parents, and
depth or capacity violations before changing either Layout.

The zero Layout border form is `BorderNone`. Every other form reserves one
cell on each edge before Layout Insets and item arrangement. The empty style
selects `"layout.border"`; optional foreground and background pointers
independently override the resolved style. The available forms and terminal
projection are defined in
[`root-sizing-and-borders-v0.md`](root-sizing-and-borders-v0.md).

## Attachment And Mutation

All container handles derived from Panel provide:

```go
SetLayout(layout Layout) error
AddLayout(layout Layout) error
```

`SetLayout` requires no existing top-level Layout. `AddLayout` attaches an
additional top-level stacking context. The complete candidate tree is
validated before any App, Panel, Layout, geometry, or snapshot state changes.
A failed operation leaves the detached Layout reusable.

Transactions provide cancellation-aware atomic forms:

```go
func (t *Transaction) SetLayout(owner Container, layout Layout) error
func (t *Transaction) AddLayout(owner Container, layout Layout) error
func (t *Transaction) Raise(control Control) error
func (t *Transaction) Lower(control Control) error
func (t *Transaction) RaiseLayout(layout Layout) error
func (t *Transaction) LowerLayout(layout Layout) error
```

Manual `SetBounds` on a Layout-managed Panel returns `ErrLayoutManaged`.
Minimum-size, visibility, App-size, destruction, and stacking changes
recompute and publish one complete state. Visibility and Layout participation
are separate: an invisible Panel retains its Layout slot.

## BoxLayout

Box measurement includes an optional two-cell border extent, outer Insets,
item Insets, and stable gaps.

- The main-axis minimum is the sum of item outer minima plus gaps.
- The cross-axis minimum is the maximum item outer minimum.
- When the available main axis exceeds the minimum, extra cells are divided
  by positive Grow weights using integer arithmetic.
- Remainder cells go to eligible items in arrangement order.
- With no grow item, main-axis extra space remains trailing.
- The cross axis uses the corresponding horizontal or vertical Alignment.
- Stretch fills the available slot after item Insets; other alignments retain
  the item's measured minimum.

When available space is below minimum, all item minima are preserved and may
extend beyond the available rectangle.

## GridLayout

Grid uses row-major arrangement and uniform measured cells. Its optional
border adds two cells to each measured axis and is removed before Grid Insets
and cell arrangement.

- Cell minimum is the maximum outer item minimum on each axis.
- Grid minimum is uniform cell size times derived rows/columns plus gaps and
  outer Insets.
- Extra columns and rows are distributed left-to-right and top-to-bottom.
- Each item uses HorizontalAlign and VerticalAlign inside its cell after item
  Insets.
- Empty fixed-capacity cells are valid.

When available space is below minimum, uniform cell minima are preserved.

## Snapshot Contract

`Snapshot.Layouts` is a required bounded array. Each `LayoutSnapshot`
contains:

- stable ID, automation key, kind, owner Panel, and optional parent Layout;
- parent-relative bounds, owner-client-relative bounds, and measured minimum;
- border details including the selected form, semantic/resolved style, and
  optional color overrides (`BorderNone` is explicit);
- immutable attachment/arrangement index and current Layout-peer stack index;
- ordered item records with Panel or Layout identity;
- each item's immutable arrangement index and current stack index; and
- the item's arranged bounds and measured minimum.

`ControlSnapshot.Layout` identifies the containing Layout when managed.
`ControlSnapshot.LayoutIndex` and `StackIndex` expose arrangement and stack
positions without changing `Children`, which remains control-tree insertion
order.

Automation explicitly projects these records into automation-owned version 1
DTOs and validates the same collection, identity, geometry, and ordering
bounds.

## Overflow

The exact callback contract is:

```go
type OverflowDisposition uint8

const (
    OverflowUseDefault OverflowDisposition = iota
    OverflowHandled
)

type OverflowEvent struct {
    Overflow OverflowSnapshot
}

type OverflowHandler func(context.Context, OverflowEvent) OverflowDisposition

func (a *App) SetOverflowHandler(handler OverflowHandler) error
func (a *App) DismissOverflow() error
```

One App-owned dispatcher starts only when an episode is pending and exits when
the bounded queue drains. It retains at most one handler invocation that
ignores cancellation. Each delivery uses the 250 ms cancellation context and
never runs under App, Layout, render, presentation, or terminal locks.

One Layout has at most one current overflow record. An episode begins on the
fit-to-overflow transition, updates without another callback, clears on
recovery or destruction, and increments its episode generation on recurrence.
The triggering mutation publishes `pending` before handler disposition.

An absent, default-requesting, panicking, saturated, timed-out, cancelled, or
shut-down handler selects `default_active`. Interactive rendering shows one
private black-on-yellow application overlay with `[OK]` when it fits, a `!`
indicator when it does not, and semantic evidence at zero geometry.
`DismissOverflow` changes the notification to `acknowledged` without falsely
clearing the underlying overflow. `Enter` or `Escape` while the fallback is
active, and the built-in automation-visible `overflow.dismiss` command, use
the same dismissal transition.

## Verification

Required coverage includes:

- exact Box/Grid minima and rectangles, stable remainder distribution,
  alignment, nesting, clipping, and resize;
- atomic attachment failure, cross-App, duplicate Panel, cycle, depth,
  capacity, and fixed-Grid rejection;
- arrangement-order stability across Panel and Layout raise/lower;
- subtree-common stacking and no-op publication behavior;
- destroyed items and owners without dangling membership;
- overflow entry, update, recovery, recurrence, handler disposition, panic,
  timeout, saturation, default overlay, dismissal, and shutdown bounds;
- immutable snapshot and automation projection/validation;
- race tests, bounded layout fuzz/property checks, and public external-package
  examples; and
- human and attached `expletives-test` Box, Grid, nested, resize, and stacking
  scenarios.
