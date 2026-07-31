# Status Bar API v0

- Status: Directed pre-v1 contract
- Authority: Direct operator instruction on 2026-07-30
- Scope: root-owned bottom-row status context and command hints
- Depends on:
  [`application-chrome-v0.md`](application-chrome-v0.md) and
  [`actions-api-v0.md`](actions-api-v0.md)

## Purpose

`StatusBar` provides the conventional Turbo Vision bottom-row surface for
short contextual text and command shortcut hints. It is application chrome,
not a general Layout item or a second command system.

Turbo Vision is the appearance and interaction reference. This specification
does not copy or require its implementation.

## Public API

```go
const MaxStatusBarSegments = 64

type StatusSegment struct {
    Key      string
    Text     string
    Command  CommandID
    Priority int
}

type StatusBarOptions struct {
    PanelOptions
    Segments      []StatusSegment
    ShortcutStyle StyleID
    DisabledStyle StyleID
}

type StatusBar struct { /* copy-safe handle */ }

func NewStatusBar(Container, StatusBarOptions) (*StatusBar, error)
func (t *Transaction) NewStatusBar(
    Container,
    StatusBarOptions,
) (*StatusBar, error)
func (b *StatusBar) Segments() []StatusSegment
func (b *StatusBar) SetSegments([]StatusSegment) error
func (t *Transaction) SetStatusSegments(
    *StatusBar,
    []StatusSegment,
) error
```

`StatusBar` embeds the ordinary Control operations. Bounds and MinimumSize are
derived and their setters reject the operation. Visibility, style, destroy,
and transactional forms retain their ordinary semantics.

## Ownership And Uniqueness

- The parent must be exactly `app.Root()`.
- At most one live StatusBar may exist in an App.
- It is a non-container, non-focusable leaf.
- It cannot be inserted into a Layout.
- Its Bounds are `(0, height-1, width, 1)` on a nonempty surface and an empty
  rectangle at `(0, 0)` otherwise.
- Root constraints never move or narrow it.
- A visible StatusBar reserves the physical last row from root content.
- Resize, hide, show, destroy, and segment changes publish atomically.

## Segment Validation And Ownership

The complete segment slice is copied at construction and mutation. It may be
empty and may contain at most `MaxStatusBarSegments` entries.

Every segment:

- has a nonempty bounded identifier `Key`, unique within the StatusBar;
- supplies exactly one of nonempty single-line `Text` or `Command`;
- uses the toolkit's canonical one-cell Unicode normalization for Text; and
- may use any `int` Priority because priority is compared, not used in
  allocation or geometry arithmetic.

A referenced Command must be registered when its construction or mutation
transaction commits. A command segment dynamically derives its effective
label, enabled state, disabled reason, checked state, and first current
binding from the shared command registry. It therefore stays consistent with
MenuItem, Button, and HotkeyBar without copying presentation state.

## Rendering

The default StatusBar style is `status_bar`: black text on a light-gray
background. `status.shortcut` renders the shortcut portion in red on the same
background. `status.disabled` renders a disabled command hint in gray.
Options may select other Theme style IDs.

The complete row is filled with the StatusBar style, including unused cells.
Segments retain declaration order when rendered:

- static context renders as ` Text `;
- a bound command renders as ` Chord Label `;
- an unbound command renders as ` Label `; and
- disabled command hints use DisabledStyle for the complete segment.

`Chord` is the existing normalized display form of the command's first
binding. Enabled shortcut cells use ShortcutStyle. A Checked command retains
its typed state in snapshots; a later phase may add a visual checked marker
without changing command ownership.

### Narrow-Width Selection

Selection is deterministic and independent of map iteration:

1. consider segments by descending Priority, retaining declaration order for
   ties;
2. retain each complete segment that fits the remaining width;
3. if no segment has yet fit, retain the highest-priority segment clipped to
   the available width; and
4. paint retained segments in original declaration order.

Thus the most important segment always receives any nonempty available row,
while later smaller segments may fill space skipped by a larger lower-priority
segment. Padding is discarded before meaningful content when a clipped
segment is only one or two cells wide. Omitted segments retain typed snapshot
records with `Rendered == false`.

## Tiny Physical Surfaces

Chrome geometry never becomes negative:

- on height 0, MenuBar and StatusBar have empty derived rectangles;
- on height 1 with only StatusBar visible, it owns row 0;
- on height 1 with both MenuBar and StatusBar visible, both retain semantic
  row-0 Bounds but the Main Menu paints last and owns the cells; and
- on height 2 with both visible, Main Menu owns row 0 and StatusBar row 1,
  leaving empty ordinary root content.

This establishes a deterministic priority until the planned recursively
computed `TOO SMALL` presentation is delivered.

## Typed Snapshot And Automation

`ControlDetails.StatusBar` contains:

```go
type StatusSegmentDetails struct {
    Key            string
    Label          string
    Command        CommandID
    Priority       int
    Enabled        bool
    DisabledReason string
    Checked        bool
    Chord          *Chord
    Rendered       bool
    Bounds         Rect
    Clipped        bool
}

type StatusBarDetails struct {
    Segments []StatusSegmentDetails
}
```

`Label` is the canonical static Text or the current effective command label.
`Bounds` is relative to the StatusBar and is empty for an omitted segment.
For static context, Enabled is true and command-only fields are empty.
Snapshots and the automation projection deep-copy the segment slice and
Chord pointers and enforce the same aggregate bounds as construction.

## Concurrency And Failure

All getters return caller-owned copies. Convenience mutation uses the App
mutation gate; transactional mutation can be combined with command, geometry,
visibility, and other control-tree operations. Concurrent render and snapshot
readers observe either the complete old slice or the complete new slice.

Invalid parent, geometry, identifiers, duplicate keys, text, styles,
unregistered commands, capacity, or final-state uniqueness reject the whole
transaction without changing the App.

## Acceptance Criteria

- A visible StatusBar owns the first and last cells of every nonempty physical
  last row unless the documented height-1 Main Menu priority applies.
- Ordinary root Layouts end above its row and reclaim the row atomically on
  hide or destroy.
- Constrained roots do not affect StatusBar geometry.
- Command label, binding, enabled, reason, and checked mutations appear
  without replacing segments.
- Priority, clipping, empty, tiny, resize, and full-width fill behavior are
  deterministic and reflected exactly by typed snapshots and automation.
- `expletives-test` provides `status.context`, `status.narrow-priority`, and
  `status.chrome-geometry` scenarios and a Controls/Status Bar catalog page.
- Normal Go, socket automation, PTY integration, race, debug, release, and
  profiling verification pass.
