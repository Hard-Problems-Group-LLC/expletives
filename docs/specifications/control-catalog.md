# Control Catalog And Delivery Order

Status: Directed requirements; detailed APIs remain under design
Authority: Direct operator request on 2026-07-24
Related decisions: `EXPL-DEC-002`, `EXPL-DEC-004`, `EXPL-DEC-005`,
`EXPL-DEC-006`, and `EXPL-DEC-007` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)
Related roadmap:
[`project-management/development-roadmap.md`](../../project-management/development-roadmap.md)
Related architecture:
[`application-architecture.md`](application-architecture.md)
Related Layout contract:
[`layouts-and-overflow.md`](layouts-and-overflow.md)
Related text scope:
[`Limited Unicode Support`](../Limited-Unicode-Support.md)

## Purpose

This document identifies the required common-control catalog and its delivery
order. The order deliberately establishes presentation, observability, and
Basic Layouts before the toolkit grows a large control surface.

## Parent And Root Invariant

Every non-root control requires a valid parent when it is created. The
application owns one special root `Panel`, which is the only control without a
parent and whose rectangle follows the application surface.

The root is available only through `app.Root()`. There is no package-global
root. Only container-capable controls may accept children.

At minimum, the control tree must reject:

- a nil parent for an ordinary control;
- a parent from another application;
- multiple simultaneous parents;
- parent/child cycles;
- reparenting any control in the initial contract, including the special
  root; and
- paint, hit testing, cursor placement, or physical presentation that escapes
  the control's effective ancestor clip.

A below-minimum Layout may intentionally assign a logical rectangle that
extends beyond its parent's content rectangle. That logical geometry is
retained for inspection; only its effective clip constrains paint, hit
testing, cursor placement, and physical presentation.

Reparenting is deferred. The initial parent selected at construction remains
the control's parent for its lifetime; layout registration never changes
ownership.

Except for explicitly non-control objects such as Layouts and supporting data
models, most common controls are conceptually derived from `Panel`, directly
or indirectly. The exact Go composition, embedding, capability, and extension
model remains a foundational decision; this catalog does not approve the
older Draft's deep inheritance or self-dispatch mechanism.

## Limited Unicode Invariant

The canonical displayed text unit occupies exactly one terminal cell.
One-cell grapheme clusters, including supported base-plus-combining
sequences, are represented as one unit. Width-two or otherwise multi-cell
glyphs and continuation cells are not supported.

Every public text boundary must replace each unsupported display element with
exactly one `U+FFFD REPLACEMENT CHARACTER` (`�`) cell, as defined by
[`Limited Unicode Support`](../Limited-Unicode-Support.md). Unsupported input
must not shift later columns, split borders, corrupt clipping, or make frame
geometry ambiguous. This invariant applies to static text, editing, streamed
content, automation snapshots, and physical presentation.

## Completion Rule

A phase is complete only when:

- its public behavior and ownership contracts are documented;
- its public surface remains thread-safe and usable from a multithreaded
  MVC-, MVVC-, or similarly structured consuming application without placing
  application models inside controls;
- normal Go unit and practical integration tests pass;
- required debug, release, and profiling builds pass;
- every public control and important state in the phase has an
  `expletives-test` scenario;
- headless frame evidence passes; and
- after Basic Automation exists, the same scenarios can be observed and
  driven through the attached socket interface where interaction applies.

Controls from a later phase should not be introduced early merely to make a
demonstration easier. Small private primitives may be introduced when they
are necessary and do not freeze the later public API.

## Delivery Phases

### 1. Core And Containers

Required public surface:

- application/session ownership;
- structured control-event and view-update seams usable by an external
  controller, presenter, update function, or equivalent application layer;
- control identity, parent/child ownership, geometry, visibility, clipping,
  and deterministic child order;
- the special root `Panel`;
- `Panel`;
- `Frame`; and
- `GroupBox`.

This phase establishes the logical tree and lifecycle. Physical rendering is
the next phase.

### 2. Basic Presentation

Required public or testable behavior:

- the root-package `Snapshot` intended cell frame needed by the core controls;
- exactly these per-cell fields in that initial snapshot: the canonical
  one-cell grapheme, semantic style, resolved foreground and background, and
  stable owner identity;
- cursor state and a bounded typed control tree at snapshot level rather than
  repeated in cells;
- no width or continuation-cell fields;
- supported one-cell grapheme clusters and combining sequences without
  continuation cells;
- replacement of every display element whose final width cannot be determined
  to be exactly one cell with exactly one `U+FFFD` cell before it can disturb
  geometry;
- deterministic paint traversal, clipping, overlap, and Z-order;
- a correctness-first full-frame presentation path;
- conservative basic-terminal presentation: definite ASCII and known code-page
  mappings render directly, while other logical cells use a deterministic
  single-character ASCII approximation, or `?`, in black on yellow;
- an initial terminal/backend adapter with owned startup and teardown; and
- the first human-runnable `expletives-test`.

The initial acceptance scene contains multiple root and nested Panels with
distinct background colors, fixed rectangles, clipping, and deliberate
overlap. Tests verify that every cell is owned, colored, clipped, and stacked
as intended. A Limited Unicode fixture verifies that supported one-cell text
remains aligned and every unsupported display element becomes exactly one
`U+FFFD` cell without moving later columns.

### 3. Basic Automation

Required invocation:

```text
expletives-test --automation <socket-path>
```

For example:

```text
expletives-test --automation /tmp/expletives.sock
```

This phase supplies the initial unauthenticated local Unix-socket
drive-and-observe path. It must let a client:

- identify the protocol and running session;
- retrieve one atomic immutable `automation.SnapshotV1`, explicitly projected
  from the root `Snapshot` and including its intended frame, top-level cursor
  state, and bounded typed control tree;
- wait for a later frame sequence;
- submit request-correlated `KeyDown`, `KeyUp`, and `KeyPress` events,
  including a modifier chord;
- submit at least one request-correlated test-app semantic command;
- receive the actual outcome and associated frame sequence; and
- request orderly test-app exit and receive the final snapshot.

`KeyPress` is a distinct one-shot raw logical event that leaves no key held.
The supported reusable client package and `expletivesctl` executable provide
the external connection path.

The colored-panel scene is the first automation fixture. A client must be able
to verify panel identity, parentage, rectangle, background style, clipping,
overlap, and final frame cells without parsing terminal escape output.
An application-level test chord must change or reset the scene so Basic
Automation proves held-modifier key routing before menus and buttons exist.

The explicit socket path is the initial trust boundary. Authentication remains
deferred; bounds, framing, collision safety, honest outcomes, and cleanup
remain required.

### 4. Basic Layouts

Required instantiated layout objects:

- `BoxLayout`; and
- `GridLayout`.

Layouts are non-visual objects, not controls and not alternate parents. They
are constructed independently, configured, and then attached to a parent
Panel through `Panel.SetLayout`; `Panel.AddLayout` adds a further top-level
stacking context. The entire attachment is validated before it
mutates either object. A successful attachment becomes visible atomically, and
a failed attachment leaves both objects, the control tree, logical geometry,
effective clips, and the current frame unchanged. Replacement and detachment
semantics remain under design. The attached Layout is then owned by that
Panel.

One Layout tree arranges direct child Panels whose real parent is the owning
Panel and may contain nested Layout objects for arrangement/stacking grouping
with optional border decoration. Attaching
a Layout never reparents a control. Because most
controls are conceptually Panel-derived, later controls can participate
through their Panel behavior without permitting arbitrary non-Panel layout
items.

The initial Layout contract includes:

- horizontal and vertical box arrangement;
- deterministic row-major grid arrangement with configured rows and columns;
- ordered direct-child Panel and nested Layout items;
- gaps and insets, plus per-Panel item minimum size, grow weight, expansion,
  and alignment;
- one primary plus optional sibling top-level Layout stacking contexts;
- nested Layout items and child Panels that own independent Layout trees;
- independent arrangement and paint order, with Panel and Layout `Raise` and
  `Lower` preserving the other peer kind's slots;
- stable remainder allocation;
- relayout after surface resize or relevant child changes; and
- preservation of logical item minima when the available rectangle is smaller
  than the combined minimum, retention of the resulting logical rectangles,
  effective clipping through every ancestor and the application surface, and
  publication of a structured overflow fact.

An application may register an `Overflow` callback or handler. Notification
occurs only after the layout pass, outside internal locks, and is queued and
coalesced to exactly one callback per continuous Panel/Layout overflow episode
so the handler may safely schedule a later view update without causing
recursive layout or notification storms. Deficit changes update snapshots but
do not cause another callback until recovery and recurrence. The structured
overflow remains observable regardless of whether the handler accepts it or
selects the default disposition. A bounded callback dispatcher and documented
cancellation/deadline behavior keep panic, saturation, timeout, shutdown, or
an uncooperative handler off the UI, Layout, render, and presentation owners;
those failures select the default without creating unbounded replacement
goroutines.

When no handler is registered, or the handler selects the default disposition,
the toolkit makes exactly one default-notification attempt for that episode.
Where geometry and presentation mode permit, it is a compact dismissible
warning overlay with an `OK` action. The fallback is outside the failing Layout
and cannot recursively report its own size as another overflow. A tiny terminal
uses a bounded high-visibility indicator; zero-sized and headless runs retain
nonblocking semantic evidence.

An event or automation request that causes overflow may complete once
structured overflow is published and bounded notification delivery is queued.
Its associated snapshot marks notification state as pending or current. It
does not wait for application callback execution or human dismissal; handler
disposition and fallback changes publish later sequenced snapshots. The
complete contract is defined in
[`layouts-and-overflow.md`](layouts-and-overflow.md).

Fixed or stretch spacer objects and arbitrary non-Panel/non-Layout items are
outside the Basic Layouts phase. Gaps, insets, and grow/alignment properties
provide the initial spacing tools.

`expletives-test` must show `BoxLayout` and `GridLayout` arrangements of
distinctly colored direct child Panels. Nested visual arrangements cover both
Layout items and child Panels that own another Layout. Automation exposes
every participating Panel's parent and computed rectangle plus Layout
arrangement and stack indices. A below-minimum scenario must verify
preserved logical minima, clipping, structured overflow, accepted and declined
handler paths, fallback warning behavior, notification coalescing, and
nonblocking automation completion. It also covers panic, dispatcher saturation,
deadline expiry, and a handler that ignores cancellation.

### 5. Text And Display

Required controls:

- `Label`;
- `StaticText`;
- `Separator`; and
- `Rule`.

This phase adds alignment, wrapping where applicable, supported one-cell
Unicode text units, deterministic one-cell `U+FFFD` replacement for
unsupported widths, clipping, semantic text styles, and
ASCII/reduced-decoration fallbacks.

### 6. Actions

Required controls:

- `Button`; and
- `HotkeyBar`.

This phase establishes stable command IDs, activation, enabled/disabled
state, structured hotkey data, focus, and command parity across human input
and automation. The implemented exact contract is
[`actions-api-v0.md`](actions-api-v0.md).

### 7. Menus

Required controls:

- `MenuBar`;
- popup `Menu`; and
- `MenuItem`.

Menus follow Actions immediately because their items invoke the same stable
commands, enabled-state checks, mnemonics, accelerators, and fallback
bindings. This phase also moves `expletives-test` from one accumulating
catalog surface to persistent menu navigation among purpose-specific screens.
It includes `Alt-F`, F10 and another documented fallback, checked/disabled
items, separators, nested popups, dismissal, clipping, resize, and exact focus
restoration.

The implemented exact contract is
[`menus-api-v0.md`](menus-api-v0.md). The public-API catalog now uses its
persistent MenuBar to show exactly one Core/Layout, Text/Display, or Actions
screen while retaining all controls as observable stable nodes.

### 8. Selection

Required controls:

- `Checkbox`;
- `RadioButton`;
- `RadioGroup`;
- `CycleField`; and
- `SelectField`.

Supporting item/selection models should be introduced deliberately and reused
by later collection controls.

### 9. Text And Numeric Input

Required controls:

- `TextField`;
- `NumberField`;
- `SpinBox`; and
- `TextArea`.

This phase includes editing, validation, caret/selection behavior over
supported one-cell grapheme clusters and combining sequences, bounded paste,
deterministic one-cell `U+FFFD` replacement of unsupported display elements,
configurable commit/cancel behavior, and the project's edit-gate policy.
`TextArea` may use a private viewport/offset primitive without publishing the
later `ScrollablePanel` API early.

### 10. Progress And Status

Required controls:

- `ProgressBar`;
- `Meter`;
- `Spinner`;
- `ActivityDots`; and
- `StatusBar`.

Animations and progress updates must use explicit deterministic tick or state
events, honor reduced motion, and remain bounded.

### 11. Navigation And Chrome

Required controls:

- `ScrollBar`;
- `TabbedPanel`; and
- `Notebook`.

This phase adds honest viewport-based scrollbars, tab focus, and page
navigation on top of the already-delivered Action and Menu command routing.

### 12. Scrolling And Content

Required controls:

- `ScrollablePanel`;
- `Viewport`;
- `MarkdownView`;
- `LogView`; and
- `StreamView`.

This phase makes the earlier private viewport/offset behavior a supported
public contract and adds bounded streaming, follow/scrollback, and honest
drop reporting.

### 13. Collections

Required controls:

- `ListBox`;
- `ComboBox`;
- `DropDown`;
- `TreeView`;
- `Table`; and
- `DataGrid`.

Supporting public data includes `ListItem`, `TreeNode`, `Column`, and related
selection/data-source contracts. Composition and capability reuse must be
chosen deliberately; the older Draft's deep inheritance relationships are
not approved by this ordering decision.

### 14. Modal Controls

Required controls:

- `ModalPanel`;
- `Dialog`;
- `MessageBox`;
- `ConfirmDialog`;
- `InputDialog`; and
- `ProgressDialog`.

This phase includes modal capture, focus entry and restoration, stacking,
default-button safety, cancellation, configurable interrupt behavior, and
final-state automation evidence.

## Deferred Structured Input

The following controls are deliberately deferred and are not part of the
active phase sequence:

- `FormPanel`;
- `Wizard`; and
- `StepContainer`.

Their eventual reactivation requires an explicit priority decision after the
underlying input, selection, navigation, scrolling, and modal contracts are
stable.

## Deferred Layout And Ownership Features

The following are deliberately outside Basic Layouts:

- fixed spacer and stretch spacer objects;
- arbitrary non-Panel/non-Layout items; and
- control reparenting.

Nested arrangements are supported both by nested Layout objects and by child
Panels with independently attached Layout trees. Reactivating any deferred
item requires an explicit ownership and API decision plus normal Go and
`expletives-test` coverage.

## Supporting Types

Supporting types should arrive with the first control that needs their public
contract:

- `BoxLayout`, `GridLayout`, their Panel-item options, and the structured
  overflow report with Basic Layouts;
- command and hotkey descriptors with Actions;
- `MenuItem` with Menus;
- `Tab` with Navigation and Chrome;
- item and selection models with Selection or Collections, according to the
  approved ownership design;
- `ListItem`, `TreeNode`, and `Column` with Collections; and
- validation models with Text and Numeric Input.
