# Foundational Window Tree, Presentation, Automation, And Layouts

- ID: EXPL-PROP-001
- Author: Codex collaboration
- Sponsor: project operator
- Date: 2026-07-24
- Status: Under Review
- Reviewers: project operator and toolkit maintainers
- Affected projects or audiences: `expletives`, `expletives-test`, and future
  consumers of the public Go API
- Related work:
  - [`EXPL-DEC-001`](../../decision-log.md#expl-dec-001--product-test-automation-and-build-direction)
  - [`EXPL-DEC-002`](../../decision-log.md#expl-dec-002--control-delivery-order-and-basic-automation-input)
  - [`EXPL-DEC-003`](../../decision-log.md#expl-dec-003--support-mvc-like-consuming-applications)
  - [`EXPL-DEC-004`](../../decision-log.md#expl-dec-004--root-input-concurrency-layout-and-build-foundations)
  - [`EXPL-DEC-005`](../../decision-log.md#expl-dec-005--limit-displayed-unicode-to-one-cell)
  - [`EXPL-DEC-006`](../../decision-log.md#expl-dec-006--degrade-unicode-conservatively-on-basic-terminals)
  - [`EXPL-DEC-007`](../../decision-log.md#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)
  - [`EXPL-TASK-002`](../../backlog.md)
  - [`docs/product-goals.md`](../../../docs/product-goals.md)
  - [`docs/Limited-Unicode-Support.md`](../../../docs/Limited-Unicode-Support.md)
  - [`docs/specifications/application-architecture.md`](../../../docs/specifications/application-architecture.md)
  - [`docs/specifications/concurrency-and-thread-safety.md`](../../../docs/specifications/concurrency-and-thread-safety.md)
  - [`docs/specifications/control-catalog.md`](../../../docs/specifications/control-catalog.md)
  - [`docs/specifications/expletives-test.md`](../../../docs/specifications/expletives-test.md)
  - [`docs/specifications/layouts-and-overflow.md`](../../../docs/specifications/layouts-and-overflow.md)
  - [`docs/specifications/ui-toolkit-requirements.md`](../../../docs/specifications/ui-toolkit-requirements.md)
  - [`docs/research/ui-toolkit-lessons.md`](../../../docs/research/ui-toolkit-lessons.md)

## Problem Statement

The project needs a small, coherent vertical foundation before it can add the
common controls in catalog order. A control must have a parent at creation
time, but an application also needs one legal root. Panels need deterministic
geometry and presentation before responsive layout exists. The operator needs
to run:

```text
expletives-test --automation /tmp/expletives.sock
```

and immediately inspect and drive the same intended frames a human sees.
BoxLayout and GridLayout must then arrange direct child Panels without
becoming a second control tree.

The older Draft supplies useful design input, but its package names, public
inheritance mechanism, canonical frame, and automation details are not
approved. Implementing an incidental root global, optional-parent controls,
ANSI-based automation, or a layout object that also owns controls would freeze
the wrong boundaries and make later common-control work harder to test.

Consuming applications also need freedom to use multithreaded MVC, MVVC, MVP,
MVU, or a similar application structure. The foundation must make those
structures natural through thread-safe composition without requiring a
toolkit-owned domain model, reflective binding system, or inheritance
hierarchy.

## Goals

- Give every `App` exactly one application-owned special root `Panel`.
- Require every ordinary control to receive a valid container-capable parent
  when it is created.
- Define stable tree ownership, identity, lifetime, geometry, clipping, and
  serialized-owner mutation invariants without global mutable state.
- Support deterministic parent-relative absolute placement before managed
  layouts.
- Present an initial colored-Panel fixture through the real intended-frame and
  terminal-presentation path.
- Provide default-off Basic Automation through the exact
  `--automation <socket-path>` command-line form.
- Let automation inject raw logical `KeyDown`, `KeyUp`, and `KeyPress` events
  before shortcut resolution as well as direct structured semantic commands.
- Correlate every accepted automation request with an explicit outcome and
  immutable frame sequence through a bounded, versioned JSON Lines protocol.
- Supply a supported first-party client path as an `expletivesctl` executable
  plus a reusable Go client package.
- Add instantiated non-control BoxLayout and GridLayout managers that
  deterministically arrange direct child Panels.
- Preserve declared Layout minima at undersized geometries, expose structured
  overflow, and deliver coalesced overflow notification through an optional
  application callback or a safe toolkit fallback.
- Support displayed text elements only when each complete grapheme cluster
  occupies exactly one terminal cell; render every unsupported display element
  as exactly one `U+FFFD REPLACEMENT CHARACTER` (`�`) cell.
- On basic terminals, emit definite ASCII and confidently known code-page
  mappings directly; otherwise present one black-on-yellow ASCII approximation
  or black-on-yellow `?` without mutating the intended frame.
- Keep domain models toolkit-independent and support multithreaded MVC-,
  MVVC-, MVP-, and MVU-style separation through thread-safe commands, events,
  rendering, and owner marshaling.
- Keep the public concurrency contract independent of internal topology.
  Initially prefer one shared serialized UI/layout/render/presentation owner,
  while allowing a backend-specific presentation owner when OS-thread
  affinity or blocking behavior strongly recommends it.
- Build `expletives-test`, `expletivesctl`, and every later executable in
  debug, release, and profiling modes at the required artifact paths.
- Establish ordinary Go, headless, attached-automation, PTY, race, and human
  validation for the foundation.

## Non-Goals

- Implementing controls beyond the foundational `Panel`.
- Implementing the deferred structured-input controls.
- Authenticating or capability-authorizing the initial attached automation
  client.
- Claiming the initial unauthenticated endpoint is suitable for hostile,
  multi-user, shared, or elevated use.
- Accepting arbitrary terminal escape bytes, backend-specific numeric key
  codes, or an unbounded byte stream through Basic Automation.
- Using terminal-output capture as the canonical intended frame.
- Supporting width-two or other multi-cell glyphs, continuation cells,
  half-glyph rendering, wide CJK text, or wide emoji.
- Prescribing MVC, MVVC, MVP, MVU, or another application architecture as the
  only supported structure.
- Requiring application models to embed toolkit types or import the terminal
  backend.
- Adding automatic reflective data binding, string-named property mutation,
  dependency injection, or a global event bus in the foundational phases.
- Adding FlexGridLayout, GridBagLayout, cell spans, docking, anchoring, stack
  layouts, or overlay layout.
- Supporting control reparenting in the foundational phases.
- Resolving the complete public external-control extension model beyond what
  the foundation needs.

## Use Cases

1. An application creates an `App`, obtains `app.Root()`, and creates every
   Panel with that root or another Panel as its parent.
2. Before managed layouts exist, a developer gives several Panels
   parent-relative rectangles and visually confirms their distinct
   backgrounds, nesting, overlap order, and clipping.
3. A human runs `expletives-test` normally and no automation socket, worker,
   discovery record, or capture artifact is created.
4. A developer runs
   `expletives-test --automation /tmp/expletives.sock`, connects a client,
   observes the current atomic snapshot, submits modifier `KeyDown`, letter
   `KeyPress`, and modifier `KeyUp` for the fixture's documented chord, and
   receives each outcome and associated frame sequence.
5. A deterministic test sends a direct semantic command, bypassing shortcut
   lookup but not enabled state, modal scope, or controller policy.
6. A BoxLayout divides the root among colored Panels, then recomputes their
   rectangles when automation or the terminal supplies a resize.
7. A GridLayout places Panels in uniform row-major cells and exposes an honest
   overflow fact when the root is smaller than the declared minima. A
   registered application callback receives one coalesced overflow
   notification after layout; without one, a deterministic dismissible
   toolkit warning can provide the fallback without blocking the event loop or
   automation.
8. A consumer keeps its domain model in a package with no `expletives`
   dependency, uses a controller, presenter, or update function to translate
   model state and structured events, and safely marshals resulting view
   changes across its chosen serialized toolkit owners.

## Constraints And Assumptions

### Verified Constraints

- `expletives` is the primary reusable Go toolkit and `expletives-test` is a
  supported public-API consumer.
- Normal mode must create no attached-automation resources.
- Attached automation is enabled only by an explicit per-process option.
- The initial endpoint may be unauthenticated only for an
  operator-controlled, non-risky context.
- Human input, headless tests, and attached automation must converge on the
  same controller and intended-frame renderer.
- The terminal backend has one owner.
- `app.Root()` is the sole root accessor; there is no package-global root.
- Every ordinary control has a container-capable parent at construction, and
  parent ownership is immutable during the foundational phases.
- Reparenting is deferred to the backlog.
- `KeyPress` is a distinct one-shot raw logical input event and leaves no
  persistent held state.
- BoxLayout and GridLayout initially arrange only direct child Panels;
  layout nesting is expressed by child Panels that own their own Layout.
- Layouts are constructed independently and attached atomically through
  `Panel.SetLayout`.
- Displayed text is limited to complete grapheme clusters whose measured
  terminal width is exactly one cell. A one-cell cluster may contain a base
  character plus combining marks; wide and multi-cell clusters are
  unsupported and each renders as exactly one `U+FFFD REPLACEMENT CHARACTER`
  (`�`) cell.
- `expletivesctl` is a required executable build target, and every executable
  is produced in debug, release, and profiling modes.
- Snapshots atomically pair an immutable intended frame with a bounded typed
  semantic view.
- Each canonical `SnapshotV1` cell record contains its one-cell grapheme,
  semantic style, resolved foreground and background colors, and stable
  owner identity. Cursor state and the bounded typed control tree are
  snapshot-level records. Width and continuation fields do not exist.
- Below-minimum Layout geometry preserves declared logical minima and resulting
  rectangles; effective output is constrained by the intersection of every
  ancestor clip and the application surface. It exposes typed overflow and
  schedules a coalesced post-layout notification. An application may register
  an overflow callback; otherwise the toolkit supplies a deterministic
  nonblocking fallback.
- Each injected request needs a unique request ID, explicit outcome, and
  associated frame sequence. A timeout is not success and a no-op completes
  explicitly.
- Every public control and meaningful state requires ordinary automated
  coverage and an `expletives-test` scenario.

### Assumptions Requiring Review

- The first attached transport is a local pathname-based stream socket.
- JSON Lines is sufficient for the initial bounded request/response protocol.
- The root Panel can begin at `0x0` until a headless session or terminal
  supplies geometry.
- One shared serialized owner is sufficient initially unless the selected
  backend requires a distinct presentation owner.

## Current Context

The first runnable Core/Containers, Basic Presentation, and Basic Automation
slice is now implemented under `EXPL-DEC-008`: the repository has a Go module,
public source, root Makefile, both product commands, the narrow Linux terminal
adapter, and the bounded JSON Lines Unix-socket protocol. The implementation
is evidence for the active foundation design review; it does not retroactively
approve this proposal, Basic Layouts, later controls, or the remaining
extension, terminal-portability, input-policy, and authentication questions.

The comparative research converges on several relevant lessons:

- wxWidgets separates the window parent/child tree from layout membership and
  gives parent windows lifetime responsibility for children.
- Xt and Motif make a parent responsible for final child geometry; their
  recursive geometry negotiation is useful history but is too implicit for
  this deterministic Go design.
- Motif BulletinBoard-style absolute placement is useful for fixed fixtures
  but does not respond to resize.
- Box and grid layout objects should measure and arrange controls without
  becoming focusable, paintable, or independently parented controls.
- Mature event systems put normalized input and structured commands between
  physical terminal bytes and application behavior.

The existing Draft's unexported self-binding mechanism cannot establish the
claimed external extension contract as written. This proposal therefore
defines observable foundation behavior while leaving the broader public
extension decision explicit. Most future controls are conceptually specialized
Panels and should reuse Panel behavior through deliberate Go composition or
embedding rather than a deep or magical inheritance system.

## Proposed Approach

### 1. App-Owned Root And Control Tree

Application construction creates one special root Panel:

```go
app, err := expletives.NewApp(options)
if err != nil {
    return err
}

root := app.Root()
left, err := expletives.NewPanel(root, leftOptions)
```

`Root()` is the approved Go API. It is an inexpensive noun-like accessor, and
the receiver identifies which application owns the root. There is no
package-level `GetRootWindow`, singleton current App, import-time
registration, or other global root lookup.

The root contract is:

- it is created with the App and returned by every call to `app.Root()`;
- it is the only legal parentless control;
- it is a concrete special `Panel`, container-capable from creation;
- it has a reserved stable runtime identity and the semantic role
  `application-root`;
- its parent is always nil;
- its parent-relative and absolute origins are always `(0,0)`;
- it starts at `0x0` before geometry is supplied and then exactly matches the
  current intended-frame geometry;
- callers may style it and add children but may not reparent, collapse,
  manually resize, or independently destroy it; and
- App shutdown publishes any required final snapshot before logically
  destroying the root tree.

Every ordinary control constructor requires a non-nil container-capable
parent. Successful construction atomically inserts the child into the parent
tree. The public type should reject a non-container parent at compile time
where practical. Typed nil, destroyed, cross-App, duplicate, or cyclic
relationships return typed errors rather than partially constructing a
control or panicking.

The public parent parameter is a small toolkit-owned container capability
whose mutation plumbing is not an unrestricted public `AddChild` interface.
The foundational concrete implementation is Panel. Future composite controls
may obtain Panel/container behavior through supported Go composition or
embedding rather than recreating internal tree state.

Tree invariants are:

- every non-root control has exactly one immutable parent;
- every node belongs to the same App as its ancestors;
- runtime control IDs are unique for the App lifetime and are not reused;
- an optional caller-supplied stable automation key is unique within its
  documented scope;
- child enumeration is stable and does not expose a mutable internal slice;
- the tree is acyclic and contains no duplicate child;
- child insertion order supplies the initial deterministic paint order;
- Z-order, focus, and Layout order remain distinct concepts; and
- destroying a parent logically destroys its descendants and removes them
  from rendering, input, layout, focus, automation lookup, and Layout
  references.

Go references may outlive logical destruction. Read-only identity can remain
diagnostic, while later mutation returns `ErrDestroyed`. The foundational
phases do not support reparenting; a later reparenting contract is deferred to
the backlog.

Construction before `Run` is synchronous under the constructing goroutine.
After the App starts, public submission, observation, and marshaling APIs are
safe for concurrent use by application-owned goroutines. Each mutable toolkit
subsystem still has exactly one serialized owner at a time. The initial
recommendation is one shared owner for the event loop, controller/view
mutation, layout, rendering, and terminal presentation. A selected backend
may justify a separate serialized presentation owner, especially for
OS-thread affinity or blocking isolation; such a split uses bounded immutable
messages and preserves event order, request correlation, frame barriers,
cancellation, and shutdown. Internal owner topology is not initially a public
application setting.

Returning `app.Root()` does not authorize unsynchronized cross-owner Panel
mutation. After startup, direct tree and Layout mutation is owner-confined;
other goroutines use the thread-safe App submission or marshaling contract and
observe immutable snapshots.

### 2. Geometry And Absolute Bootstrap Layout

Public geometry uses cell coordinates and checked nonnegative dimensions:

- `Point{X, Y}`;
- `Size{Width, Height}`; and
- `Rect{X, Y, Width, Height}`.

A normal child's requested bounds are relative to its parent's client/content
origin. The parent assigns final bounds. Snapshots expose both final
parent-relative bounds and derived absolute bounds, plus the effective clip
rectangle after intersecting all ancestors.

Before a Panel has a Layout, requested absolute bounds are its bootstrap
placement. Parent resize does not move or resize such children; it only
changes clipping. Negative origins may support deliberate clipping tests, but
negative dimensions and overflowing coordinate arithmetic are rejected.
Zero-sized and fully clipped controls are valid and paint nothing.

Calling a manual bounds setter for a Layout-managed child returns
`ErrLayoutManaged` rather than silently losing the request. The basic Panel
has no frame inset, so its client rectangle is its full bounds. Later framed
containers may define a smaller client area without changing the
parent-relative rule.

### 3. Multithreaded Application-Architecture Composition Boundary

The toolkit supplies mechanisms rather than an application architecture:

```text
toolkit-independent model
    -> application controller, presenter, or update function
    -> structured command/event and view state
    -> thread-safe App submission/marshaling boundary
    -> serialized UI/layout/render owner
    -> serialized terminal presentation
```

Applications may use:

- mutable MVC controllers that translate model notifications into view
  mutations;
- MVVC-style separation with independently scheduled model, view, and
  controller/presenter responsibilities;
- MVP presenters that expose a narrow view interface;
- MVU update functions that return a new model and effects; or
- another explicit application-owned structure.

The foundation must make each practical by providing:

- an explicit App and root rather than ambient current-window state;
- stable control and command identities;
- structured input events and semantic commands;
- thread-safe asynchronous submission and a bounded synchronous call path
  where one is justified;
- documented owner identity and safe inline behavior or an explicit error
  when a synchronous call originates on its target owner, never
  enqueue-and-wait deadlock;
- immutable worker results marshaled between the configured serialized
  owners;
- a public concurrency contract that does not expose or depend on internal
  owner topology; and
- pure layout/rendering from view state plus geometry.

Application domain packages do not need to import the terminal backend or
embed toolkit controls. The foundational phases do not define a required
`Model`, `View`, `Controller`, `Presenter`, or `Update` interface. They also
do not inspect application structs reflectively or mutate string-named
properties. Optional binding or architecture adapters require later use cases
and review.

### 4. Basic Presentation And Colored-Panel Fixture

Basic Presentation introduces the minimum intended-frame and terminal path
needed to prove geometry:

- a fixed-size semantic cell frame;
- a resolved background color and semantic style on every painted cell;
- cell ownership by stable control identity;
- root-to-leaf background fill;
- ancestor clipping;
- deterministic sibling paint order;
- full-frame correctness-first rendering; and
- one terminal adapter that presents the same intended frame observed
  headlessly and through automation.

The displayed-text scope is directed and no longer open: each displayed text
element is one complete grapheme cluster occupying exactly one terminal cell.
A grapheme is one user-perceived text element and may contain multiple Unicode
code points, such as a base letter plus combining marks. Width-two and other
multi-cell glyphs, continuation cells, half-glyph rendering, wide CJK text,
and wide emoji are unsupported; each unsupported display element becomes
exactly one `U+FFFD REPLACEMENT CHARACTER` (`�`) cell. The colored fixture
requires only one-cell space clusters, background styles, owners, geometry,
and cursor-hidden state.

The canonical Basic Presentation cell is one record containing:

- the one-cell grapheme cluster, including the one-cell `U+FFFD`
  representation for unsupported input;
- semantic style;
- resolved foreground and background colors; and
- the stable control owner.

`SnapshotV1` carries cursor position and visibility plus the bounded typed
control tree at snapshot level rather than repeating them per cell. It has no
width or continuation fields and no parallel width or continuation planes.
Optional diagnostic projections remain separate from the canonical snapshot
contract and cannot reopen the one-cell display contract. See
[`Limited-Unicode-Support.md`](../../../docs/Limited-Unicode-Support.md).

The first stable `expletives-test` scenario is
`foundation.absolute-panels`. At an intended geometry of `40x12`, it contains:

| Panel key | Parent | Parent-relative bounds | Semantic background |
| --- | --- | --- | --- |
| `root` | none | `(0,0,40,12)` | `fixture.root` |
| `left` | `root` | `(1,1,12,4)` | `fixture.left` |
| `right` | `root` | `(16,2,10,6)` | `fixture.right` |
| `nested` | `left` | `(2,1,5,2)` | `fixture.nested` |
| `clipped` | `root` | `(36,9,8,4)` | `fixture.clipped` |

The resolved test theme uses visibly distinct backgrounds. Automated
assertions use semantic styles, owners, and rectangles rather than relying on
color perception. This color-only fixture diagnoses foundational composition;
it is not a general accessibility precedent for later interactive controls.

### 5. Basic Automation Command Line And Endpoint

`expletives-test` parses automation as a string-valued option with one
required socket-path argument:

```text
expletives-test --automation /tmp/expletives.sock
```

The exact supplied pathname is the requested endpoint; the application does
not silently replace it with another path. Without the option, the process
creates no listener, automation goroutine, discovery record, capture
directory, or related artifact.

When enabled, startup visibly reports:

- that unauthenticated automation is active;
- the exact socket path;
- that anyone able to connect can observe visible data and drive UI actions;
  and
- that the mode is only for an operator-controlled, non-risky process.

Before binding, the application validates bounds and inspects the path without
following a final symlink. It refuses any existing filesystem object rather
than unlinking it. It creates the socket with the most restrictive practical
permissions, records the identity of the object it created, and removes it on
orderly shutdown only if the path still identifies that socket. A shared
directory such as `/tmp` receives an explicit warning because pathname
permissions are not portable authentication. Authentication, capabilities,
elevated use, and hardened multi-user operation remain deferred.

Basic Automation permits one active injecting controller at a time. Additional
observers may be considered only after ordering, retention, and confidentiality
behavior is defined. Client count, outstanding requests, queue depth, line
length, geometry, text, snapshot size, retained sequences, and timeouts are
bounded with project-owned constants documented before implementation exits
the milestone.

The supported client consists of a reusable Go automation-client package and
the `expletivesctl` command. The command takes an explicit
`--socket <socket-path>`, can perform the handshake, fetch an exact or latest
snapshot, submit each raw key lifecycle event, invoke a stable command, query
a pending result, and request orderly application exit. Machine-readable
results go to stdout and diagnostics to stderr. As a required project
executable, it is built in debug, release, and profiling modes under the same
artifact contract as every other command.

### 6. Versioned JSON Lines Protocol

The local stream carries UTF-8 JSON Lines: exactly one bounded JSON object per
newline-terminated record. Embedded newlines use normal JSON escaping. Blank
lines, malformed JSON, oversized records, unsupported versions, unknown
top-level message types, duplicate active request IDs, and invalid fields
receive structured failures without terminating the server.

Every record includes:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "request-or-response-type"
}
```

The server begins with a bounded `hello` record identifying the selected
version, supported operations, limits, current scenario, and latest frame
sequence. Version 1 operations are:

- `observe` — return the latest or a retained exact immutable snapshot;
- `inject_input` — enqueue a raw logical pre-binding input event;
- `invoke_command` — enqueue a direct stable semantic command;
- `resize` — request bounded headless/test geometry where the active mode
  permits it; and
- `shutdown` or `quit` only through the same enabled application command
  policy as a human action.

Each request supplies a nonempty unique `request_id`. Every accepted request
receives one terminal completion containing:

- the same request ID;
- an explicit outcome such as `applied`, `no_op`, `rejected`, `cancelled`,
  `interrupted`, `exited`, or `failed`;
- the associated monotonic `frame_sequence`; and
- a bounded structured error when applicable.

Acceptance into a queue is not completion. The configured serialized owners
apply the event, lay out, render, atomically publish the intended frame and
semantic view, then complete the matching request. Unrelated human input,
timer activity, resize, or redraw cannot satisfy it. Exit-producing events
publish the final snapshot first. Timeouts report unknown or failed completion
and never cause silent replay.

An `observe` response returns one atomic `SnapshotV1` containing at least:

- frame sequence and geometry;
- immutable intended cells required by Basic Presentation, each containing
  exactly one supported one-cell grapheme cluster—including the one-cell
  `U+FFFD` representation for unsupported input—plus semantic style, resolved
  foreground and background colors, and stable control owner;
- snapshot-level cursor position and visibility;
- the bounded typed control tree with scenario key, control identity, role,
  parent, child order, local bounds, absolute bounds, effective clip,
  visibility, layout-overflow facts, and overflow-notification outcome; and
- final-state indication when applicable.

`SnapshotV1` has no cell-width field, continuation flag, continuation-cell
record, or equivalent parallel plane. Its frame, cursor, typed tree, overflow
state, and notification outcome are immutable and atomic from an observer's
perspective.

No unrestricted application model map is exposed. Optional diagnostics such
as timestamps, redraw counts, backend damage, or export acknowledgements are
separate projections and cannot make an otherwise identical canonical
snapshot nondeterministic.

### 7. Pre-Binding Input And Direct Commands

Basic Automation supports two deliberate injection levels.

The first is raw logical input before binding, mnemonic, accelerator, hotkey,
or focused-control resolution. A conceptual modifier chord is:

```text
{"kind":"key_down","key":"control"}
{"kind":"key_press","key":"r"}
{"kind":"key_up","key":"control"}
```

In this proposal, a “raw” key event means this device-independent key
lifecycle event at the pre-shortcut boundary. It never means raw terminal
bytes.

The versioned kinds are `key_down`, `key_up`, and `key_press`, corresponding
to public concepts `KeyDown`, `KeyUp`, and `KeyPress`. They carry a
backend-independent logical key identity and only the bounded repeat or
committed-text fields approved for that event. Modifier state is produced by
the ordered lifecycle of that automation source rather than an unrelated
global or human-input state. The events do not carry terminal escape
sequences, terminfo numeric values, curses key constants, or backend-specific
codes.

`KeyDown` and `KeyUp` exercise stateful press/release behavior. `KeyPress` is
an approved distinct one-shot raw logical event. It is not expanded into
`KeyDown` plus `KeyUp` and leaves no persistent held-key state. Held-key state
created by `KeyDown` is scoped to the active injecting connection. Disconnect,
explicit input-session reset, and shutdown must reset it deterministically
without leaving a stuck modifier, and the resulting controller state must be
observable.

The second level is a direct structured semantic command with a stable command
ID and, where meaningful, an explicit target control ID or automation key.
Direct commands bypass physical shortcut lookup but still obey enabled state,
focus/modal scope, validation, interrupt policy, and application
authorization. They never call a control method or terminal backend directly.

Input and direct commands enter bounded queues consumed fairly by the
configured serialized owner or owners. Human input, injected input, resize,
and background results cannot starve one another indefinitely. Basic
Automation does not accept arbitrary terminal byte sequences; raw terminal
decoding belongs in PTY and terminal adapter tests.

### 8. BoxLayout And GridLayout

This section summarizes the foundational design. The directed
[`Layouts And Overflow`](../../../docs/specifications/layouts-and-overflow.md)
specification is authoritative for attachment, undersized geometry, overflow
episodes, callback delivery, fallback notification, and automation behavior.

A Layout is an instantiated non-control object. It has no control parent,
identity, focus, input handling, paint method, semantic role, or terminal
access. `NewBoxLayout` and `NewGridLayout` construct independent, unattached
Layout instances. `Panel.SetLayout` is the sole foundational attachment
operation and atomically validates and installs the Layout. A failed
attachment leaves the Panel, Layout, control tree, logical geometry, effective
clips, and current frame unchanged. A parent Panel owns at most one installed
Layout, and one Layout is installed in at most one Panel. The Layout references
only its owning Panel's direct child Panels and does not own their logical
lifetime.

Panels are created with their real Panel parent before being inserted into
that parent's Layout:

```go
left, err := expletives.NewPanel(root, leftOptions)
if err != nil {
    return err
}

box := expletives.NewBoxLayout(expletives.Horizontal, boxOptions)
if err := box.Add(left, itemOptions); err != nil {
    return err
}
if err := root.SetLayout(box); err != nil {
    return err
}
```

Installing or mutating a Layout validates that:

- every item is a direct child Panel of the owning parent Panel;
- each participating child Panel appears exactly once;
- a Layout is installed in at most one parent Panel; and
- destroyed Panels are removed or rejected without dangling Layout
  references.

All attachment checks occur before the owning Panel publishes the new Layout.
Construction and item setup can therefore fail independently from attachment,
while callers never observe a partially installed Layout. Replacement,
detachment, and destruction details must preserve the same atomicity and
single-owner invariant; their exact API names remain a follow-up API-review
detail rather than reopening `Panel.SetLayout` as the attachment boundary.

Layout order determines arrangement. Child-tree order remains the
deterministic paint/Z-order source until an explicit stack contract changes
it. Focus order must eventually follow visual/read order rather than
accidentally depending on allocation addresses.

Layout nesting is expressed through the control tree: a child Panel may own
its own BoxLayout or GridLayout for its direct child Panels. A BoxLayout or
GridLayout is not itself inserted as an item in another Layout. Most later
controls are conceptually specialized Panels and may reuse Panel behavior
through the future approved Go composition/embedding model.

#### BoxLayout

BoxLayout supports:

- horizontal or vertical orientation;
- ordered direct child Panel items;
- nonnegative main-axis gap and per-item insets;
- a nonnegative integer grow weight; and
- cross-axis `start`, `center`, `end`, or `stretch` alignment.

Its measured main-axis minimum is the sum of item outer minima plus gaps. Its
cross-axis minimum is the maximum item outer minimum. Arrange begins at those
minima. Extra main-axis cells are divided by grow weight using integer
arithmetic; remainder cells go to eligible items in stable insertion order.
Without a grow item, unused main-axis space remains at the trailing edge.
Stretch fills the available cross-axis slot after insets; other alignments
retain the item's measured minimum.

#### GridLayout

GridLayout supports:

- row-major item order;
- uniform cells;
- nonnegative horizontal and vertical gaps;
- item insets and the same within-cell alignments as BoxLayout; and
- `Rows` and `Columns`, with at least one positive.

When one dimension is zero, item count derives it from the positive dimension.
When both are positive, adding beyond fixed capacity returns a typed error.
Empty final cells are valid. The cell minimum is the maximum outer minimum
width and height of all items. Grid minimum is uniform cell size times the
derived dimensions plus gaps. Extra cells are distributed deterministically
from left to right and top to bottom.

Flex rows or columns, spans, GridBag constraints, nested Layout items,
spacers, and automatic responsive column-count changes are deferred.

#### Below-Minimum Geometry

The approved policy preserves declared minima in the logical arrangement,
retains the resulting logical rectangles, constrains effective paint, hit
testing, cursor placement, and presentation by the intersection of every
ancestor clip and the application surface, never produces negative rectangles,
and publishes typed overflow state containing the responsible Panel and Layout
identities, required and available sizes, horizontal and vertical overflow,
and a stable incident generation. It does not silently reinterpret minimums as
zero or compress controls below their declared minimum. Later controls may
deliberately declare a separate compact or too-small presentation.

An application may register an `Overflow` callback through the toolkit's
documented notification policy. The callback receives an immutable, bounded
event for a Panel/Layout overflow episode and selects handled or
default-fallback disposition. It is a notification boundary, not a layout
hook: it cannot change the pass that produced the event, and it must request
any UI mutation through the normal thread-safe App submission/marshaling path.
The exact exported handler and registration types remain an API-review detail.

If no application callback is registered, the toolkit uses a sensible
fallback. The initial fallback is a compact toolkit-owned dismissible warning
overlay with an `OK` action where interactive geometry permits. It is ordinary
nonblocking UI state, never a synchronous popup call that waits for a human.
It is clamped to the available root geometry, is excluded from ordinary
application Layout measurement, and cannot itself generate another overflow
incident. At very small geometry it degrades to a bounded high-visibility
indicator; at zero geometry the typed overflow and notification outcome remain
observable even though no warning cells can be painted.

Notification is episode-based rather than frame-based. When an application
handler applies, exactly one callback delivery attempt is queued for each
Panel/Layout pair from transition into overflow until recovery or destruction.
Without a handler, exactly one default-notification attempt is queued. Further
resizes update the structured deficit and geometry but do not create another
callback or warning; a still-queued event may coalesce to the latest state.
Multiple simultaneous episodes remain individually observable, while the
fallback may summarize a bounded group instead of stacking warnings. Recovery
clears the episode latch so a later recurrence can notify again. Dismissing
the fallback acknowledges the current episode and does not immediately reopen
it while the same overflow persists.

### 9. Layout Invalidation And Resize

Internally, layout distinguishes measure, arrange, and paint dirtiness:

- tree, Layout, participation, minimum-size, padding, or content-size changes
  invalidate measure upward;
- absolute placement changes invalidate arrange and paint;
- style and background changes invalidate paint only; and
- resize invalidates measurement when available size can affect content,
  then arrangement and paint.

The configured serialized owners coalesce compatible invalidations, update
root geometry, measure bottom-up, arrange top-down, paint once into the
intended frame, and publish one atomic snapshot. Layout and painting perform
no terminal I/O, time reads, domain mutations, or reentrant callbacks.
Mutations requested during a pass are rejected or scheduled for a later pass
rather than recursing.

Only after the pass has committed its overflow state and released tree,
Layout, rendering, snapshot, and all other internal locks may the toolkit
enqueue the coalesced `Overflow` notification. It never invokes application
code during measure, arrange, or paint. Callback delivery uses a bounded
serialized application-callback context isolated from the layout/render
owner. Long-running work must be posted elsewhere. Callback-driven UI
mutations are later transitions through the App boundary, never recursive
Layout passes.

The disposition path has bounded queueing, cancellation, and completion
behavior. A callback panic, unavailable or blocked dispatcher, shutdown, or
failure to return a disposition within that bound is recorded as a structured
delivery failure and elects the same nonblocking fallback. A stuck callback
cannot block layout or painting, and the toolkit does not create a replacement
goroutine per episode. Headless tests use a controlled scheduler for this
bound so disposition and fallback selection are reproducible rather than
wall-clock races.

Each atomic snapshot exposes both current structured overflow and the
notification outcome for the episode, such as queued for the application
callback, application handled, default requested, coalesced, fallback pending,
fallback visible, acknowledged, delivery failed, or physically suppressed at
zero geometry. Exact exported enum spelling is an API-review detail, but every
state is typed and deterministic. The post-disposition snapshot is linked to
the same overflow episode and records either application handling or the
scheduled default notification.

Automation completion waits only until structured overflow and bounded
notification delivery are published as queued; it does not wait for callback
execution, disposition, or human dismissal. Later disposition and fallback
transitions publish sequenced snapshots linked to the same episode. A valid
event that causes no visible or semantic change still completes as `no_op`
with an associated frame sequence. Headless tests and attached automation run
the same episode/coalescing state machine, can inspect every notification
outcome, and can acknowledge the fallback through the same semantic action as
a human; no special blocking dialog path exists.

## Alternatives Considered

### Package-Global `GetRootWindow`

Rejected. It creates ambient mutable state, prevents independent Apps and
parallel headless tests, obscures ownership, and complicates MVC-style
composition. An App method named `GetRootWindow` would avoid the global but is
less idiomatic and less concise than `Root()`.

### Caller-Created Parentless Top-Level Panels

Rejected for the foundation. Multiple special parentless nodes would require
another hidden desktop or screen owner, weaken constructor invariants, and
make automation and final-frame ownership ambiguous.

### Optional Parent Followed By `Add`

Rejected. It creates observable half-constructed controls and forces every
paint, input, layout, and automation path to handle orphan state.

### Early Reparenting

Deferred to the backlog by operator decision. It expands focus, Layout,
identity, event-routing, and automation semantics before a concrete use case
requires it.

### Layouts As Controls Or Parents

Rejected. It creates competing control and layout trees and confuses lifetime,
focus, event routing, semantic snapshots, and Z-order.

### Absolute Layout Only

Rejected as the ongoing model. It is sufficient for the first colored fixture
but cannot respond coherently to terminal resize or content minima.

### Implement All Layout Managers Initially

Rejected. BoxLayout and uniform GridLayout are sufficient to compose early
catalog screens. Flex, stack, anchor, form, and grid-bag behavior should be
added from demonstrated scenarios.

### ANSI Capture As Automation State

Rejected. It cannot recover semantic styles, control ownership, intended
geometry, or application state and would compete with the intended frame.

### Only Direct Semantic Commands

Rejected. Direct commands are valuable for precision but do not exercise key
normalization, modifier state, mnemonics, accelerators, hotkeys, or shortcut
precedence.

### Arbitrary Escape-Byte Injection

Rejected from Basic Automation. It would expose a parser and terminal-specific
surface at the wrong layer. Raw terminal bytes remain a PTY/input-decoder test
concern.

### Authenticate The First Endpoint

Deferred by operator direction. Default-off enablement, conspicuous warning,
bounds, path safety, and honest cleanup still apply. Authentication and
capability authorization require a later proposal.

### Prescribe One Application-Architecture Framework

Rejected. A reusable toolkit should supply explicit state, event, command,
rendering, and serialized-owner boundaries while letting consumers choose an
application structure appropriate to their domain.

### Automatic Reflective Binding

Rejected from the foundation. It would introduce hidden mutation, runtime
failure, stringly property contracts, and another compatibility surface before
real controls or binding use cases exist.

## Risks And Mitigations

- **The special root leaks exceptions throughout the code.** Keep root-only
  behavior in App construction and guarded root operations; otherwise render
  and traverse it as an ordinary Panel.
- **The parent type freezes the wrong extension model.** Approve the minimum
  container capability before public implementation and keep internal
  mutation plumbing out of a broad producer-owned interface.
- **Immutable parents make a later dynamic use case awkward.** Support
  create/destroy and explicit visibility first; add reparenting only with
  focus, Layout, identity, and event-routing tests through the deferred
  backlog work.
- **Absolute placement appears to be the preferred layout API.** Label it
  bootstrap/manual geometry and move `expletives-test` to Layouts immediately
  after Basic Automation is usable.
- **Terminal colors differ from the intended fixture.** Assert semantic
  styles and resolved intended cells headlessly, then retain a small declared
  real-terminal visual check.
- **JSONL frames become too large or block the UI.** Bound geometry, line and
  response sizes, retained snapshots, queues, clients, and write deadlines;
  serialize outside terminal calls from immutable snapshots.
- **A malformed or slow client starves human input.** Use bounded per-client
  work, one active injector, backpressure, timeouts, and documented fair
  scheduling by the UI owner.
- **Request completion is associated with an unrelated redraw.** Carry a
  request-owned completion token through controller application, layout,
  render, publication, and response.
- **A disconnected controller leaves Alt or another key held.** Scope held
  input state to the active injector and deterministically clear it on
  disconnect, explicit input-session reset, and shutdown with observable
  controller state.
- **“Raw key” is mistaken for arbitrary terminal bytes.** Call the events raw
  logical, pre-binding `KeyDown`, `KeyUp`, and `KeyPress` and explicitly
  reject escape bytes and backend key codes.
- **The unauthenticated socket exposes control or visible data.** Keep it
  default-off, print the risk and exact path, refuse elevated/hardened claims,
  and defer authentication through a separate proposal.
- **The socket path is replaced before cleanup.** Refuse pre-existing paths,
  record the created object identity, and unlink only the verified object
  owned by this process.
- **Layout membership and tree ownership diverge.** Validate direct-child
  Panel membership at installation and mutation, and remove destroyed Panels
  from Layout references.
- **Below-minimum behavior corrupts rectangles.** Preserve nonnegative logical
  minima, clip through ancestors, and expose typed overflow rather than
  hiding it.
- **Overflow callbacks reenter Layout or deadlock internal state.** Publish the
  overflow state before dispatch, invoke immutable events after layout on the
  bounded callback dispatcher with no toolkit locks held, and require
  callbacks to submit later mutations through the normal App boundary.
- **Persistent overflow creates callback or popup storms.** Make exactly one
  handler-delivery or default-notification attempt per Panel/Layout overflow
  episode, coalesce later deficit updates, suppress the fallback after
  acknowledgement while the episode persists, and prevent the toolkit
  fallback from participating in application Layout measurement or producing
  overflow notifications.
- **A callback or warning makes automation hang.** Bound disposition delivery,
  elect the fallback on delivery failure, and never wait for warning dismissal
  when completing the triggering request. Keep callback execution isolated,
  model the fallback as ordinary dismissible UI state, and expose notification
  progress in snapshots.
- **Toolkit architecture contaminates domain models.** Keep model packages
  terminal-independent, use narrow application-owned interfaces, and verify
  multithreaded MVC/MVVC/MVP/MVU examples without reflective binding.
- **Synchronous owner marshaling deadlocks.** Run safely inline when already
  on the target owner or return an explicit error; never enqueue and wait on
  that same owner.
- **A backend-specific presentation owner introduces races or reorders
  completion.** Transfer immutable bounded messages, retain request tokens
  across the owner boundary, and test the selected topology under the race
  detector.
- **The canonical cell grows backend-specific or duplicates snapshot state.**
  Keep each `SnapshotV1` cell to the approved one-cell grapheme, semantic
  style, resolved colors, and owner. Keep cursor, bounded typed tree, overflow,
  and notification outcome at snapshot level, and keep backend diagnostics in
  separate projections.
- **Unsupported text behaves inconsistently across controls.** Define and
  test the required one-cell `U+FFFD` replacement at every public text
  boundary before text-bearing controls ship; never partially render, split,
  or retain half of a wide glyph.
- **A basic or remote terminal's high-bit repertoire is guessed.** Require an
  explicit, queryable, or otherwise proven terminal profile before direct
  high-bit output. Unknown mappings use the deterministic black-on-yellow
  ASCII approximation or `?`, and backend diagnostics expose the degradation.

## Validation

### Documentation And API Review

- Review `Root()`, parent, geometry, Layout, input, command, snapshot, outcome,
  lifecycle, nil, and concurrency contracts before implementation.
- Confirm the proposal does not silently approve the Draft's package names or
  self-binding inheritance mechanism.
- Verify example application model packages import no toolkit or terminal
  package and can be consumed through multithreaded MVC, MVVC, MVP, or MVU
  composition.

### Ordinary Go Unit Tests

- Create multiple independent Apps and verify their roots and IDs do not
  collide.
- Test compile-time container constraints through public API examples, then
  typed-nil parent, cross-App ownership, duplicate child, cycle, destroyed
  parent, forbidden root operations, and immutable parent where applicable.
- Test parent-relative, absolute, nested, negative-origin, zero-size, clipped,
  overlapping, and resized rectangles with checked arithmetic.
- Test intended background, semantic style, owner, clipping, and paint order
  for every fixture cell.
- Test that every canonical cell contains exactly its one-cell grapheme,
  semantic style, resolved foreground and background colors, and stable owner
  identity; verify cursor and bounded typed tree are snapshot-level data and
  that no width or continuation field or plane is serialized.
- Test that a one-cell grapheme containing a base plus combining marks remains
  one displayed cell and one semantic text element.
- Test that width-two and other multi-cell glyphs, wide CJK, and wide emoji
  never create continuation cells, half-glyphs, shifted geometry, or
  nondeterministic output.
- Table-test that measurement, rendering, semantic snapshots, and automation
  report exactly one `U+FFFD` cell for each unsupported display element.
- Table-test ASCII-only, known 8-bit, unknown-code-page, remote-unknown, and
  monochrome terminal profiles. Direct output requires a definite mapping;
  approximations remain one character, use black on yellow when available,
  and do not mutate the intended snapshot.
- Table-test BoxLayout orientation, gap, insets, grow weights, integer
  remainder, cross alignment, direct-child validation, empty content, and
  overflow.
- Table-test GridLayout row-major order, derived dimension, fixed capacity,
  uniform cells, gaps, remainder, alignment, empty final cells, and overflow.
- Test Layout rejection of non-direct children and Layout nesting through
  child Panels that each own their own Layout.
- Test independently constructed Layouts, atomic `Panel.SetLayout`
  installation, single-Panel ownership, and unchanged Panel, Layout, control
  tree, logical geometry, effective clips, and frame after every failed
  attachment validation.
- Test invalidation after content, minimum, style, Layout, and resize changes.
- Test that below-minimum arrangement preserves declared minima and resulting
  logical rectangles, constrains effective output by every ancestor clip and
  the application surface, and publishes required/available sizes, overflow
  amounts, incident generation, and notification outcome.
- Test post-layout callback delivery with no internal locks held, coalescing
  across every later pass in the same episode, deficit updates without repeat
  notification, recovery and recurrence, callback panic and blocked-callback
  fallback, acknowledgement, and a fallback that cannot recursively overflow.
- Test thread-safe post/call cancellation, queue bounds, target-owner inline
  behavior, shutdown, and absence of enqueue-and-wait deadlock.
- Test request/frame ordering with the selected shared-owner topology and any
  backend-specific presentation-owner split.

### Go Integration And Headless Tests

- Build the complete absolute-Panel fixture through supported public APIs in
  an external test package.
- Drive resize and scenario commands through the real controller and assert
  atomic immutable snapshots.
- Exercise a toolkit-independent model with an application-owned controller
  or update function and the public thread-safe owner-marshaling path from
  multiple goroutines.
- Run BoxLayout at `34x8` with horizontal grow weights `1:2:1` and one-cell
  gaps; verify child widths `8,16,8`. Resize to `18x6` and verify `4,8,4`.
- Run GridLayout at `23x9` with three columns, two rows, and one-cell gaps;
  verify `7x4` cells and an empty final cell.
- Shrink below minimum, then grow, and verify recovery without stale cells.
- Run undersized Layouts with an application callback, without a callback, and
  with a blocking callback; verify bounded disposition selects application or
  fallback state, and request completion never waits for warning dismissal.

### Attached Automation Tests

- Assert normal startup creates no endpoint or automation worker.
- Start exactly
  `expletives-test --automation <owned-temporary-socket-path>`, observe the
  warning and exact path, connect, receive `hello`, and retrieve the fixture.
- Test request IDs, duplicate IDs, no-op, rejection, cancellation, unrelated
  redraw, timeout, final exit snapshot, retained and unknown frame sequences.
- Inject `KeyDown`, `KeyUp`, and `KeyPress` through the pre-binding path and
  compare the result with direct semantic commands.
- Drive the foundational fixture's documented modifier chord as modifier
  `KeyDown`, letter `KeyPress`, and modifier `KeyUp`; verify its scene
  change/reset command and exact associated snapshot.
- Exercise an `Alt-F` raw lifecycle sequence once shortcut infrastructure
  exists; ensure arbitrary escape bytes are rejected.
- Test disconnect with held modifiers, fair human/injected scheduling, bounded
  queues, slow clients, malformed/truncated/oversized JSON, wrong versions,
  unknown fields, response write failure, and clean shutdown.
- Test existing regular file, directory, symlink, and socket collisions and
  verify the process removes only its own endpoint.
- Observe structured overflow and every notification outcome, acknowledge the
  toolkit fallback through the same semantic action as a human, and verify the
  triggering request completes with the overflow-and-notification-queued
  snapshot before callback disposition or warning dismissal; then verify later
  disposition and fallback snapshots remain linked to the same episode.

### PTY, Race, And Human Evidence

- Verify terminal initialization, resize, intended-frame presentation, Ctrl-C
  policy placeholder, normal quit, and restoration at the process boundary.
- Run race detection over controller, invalidation, snapshot publication, and
  automation scheduling paths.
- Have a human inspect the colored absolute, BoxLayout, GridLayout, resize, and
  below-minimum scenarios plus the default dismissible overflow warning in
  `expletives-test`.
- On the declared initial terminal, compare intended semantic styles with the
  visibly presented background regions without claiming wider compatibility.

### Build And Artifact Evidence

- Treat `expletives-test` and `expletivesctl` as required executable build
  targets from Basic Automation onward.
- Verify `make build`, `make release`, and `make profiling` each produce every
  executable in the project inventory, including:
  - `build/debug/expletives-test` and `build/debug/expletivesctl`;
  - `build/release/expletives-test` and `build/release/expletivesctl`; and
  - `build/profiling/expletives-test` and
    `build/profiling/expletivesctl`.
- Verify `make all` produces the same complete executable inventory in all
  three modes and that each artifact is executable where the platform uses
  executable mode bits.
- Require later executable/build targets to join all three mode inventories
  rather than being emitted in only one configuration.

## Open Questions

### Critical Operator Decisions

None. On 2026-07-24 the project operator approved the canonical Basic
`SnapshotV1` placement, independent Layout construction with atomic
`Panel.SetLayout` attachment, and preserve/clip/report below-minimum behavior
with a safe application callback and toolkit fallback.

### Follow-Up Design Questions

- What exact exported names represent overflow incidents, callback
  registration, notification outcomes, and acknowledgement, and may a running
  App replace or unregister its callback?
- What exact callback-disposition deadline, cancellation contract, and
  controlled-scheduler hook make a handler that never returns select fallback
  deterministically without unbounded goroutine retention?
- What exact `Panel.SetLayout` replacement, detachment, and Layout-destruction
  APIs preserve atomic attachment and single-Panel ownership?
- What exact quantitative geometry, line, queue, client, retention, and
  timeout bounds does protocol version 1 publish?
- What stable command and logical key identity schema will later cover the
  full common-control catalog?
- Which terminal backend first presents the colored fixture, and which
  terminal constitutes initial human acceptance?
- Should invisible and layout-collapsed be separate states when dynamic
  controls arrive?
- Which versioned Unicode segmentation and width data determines whether a
  complete display element occupies exactly one cell?
- When later layout and paint orders differ, what explicit visual traversal
  order governs focus and accessibility?

## Milestones

### 1. Approve Foundation Contracts

- Owner: project operator and toolkit API reviewer
- Dependencies: `EXPL-TASK-002` and final frame/API review of the approved
  contracts
- Entry criteria: proposal is indexed and reviewers can trace it to the
  directed product requirements
- Exit criteria: the approved SnapshotV1, Layout attachment, below-minimum,
  overflow notification, root, parent, KeyPress, concurrency,
  application-architecture, Layout, and executable decisions are reflected
  consistently

### 2. Core/Container

- Owner: toolkit maintainers
- Dependencies: Milestone 1 and approved Go module/package baseline
- Entry criteria: public naming and parent capability are decided
- Exit criteria:
  - App-owned root and mandatory-parent Panel construction work;
  - tree ownership, identity, destruction, geometry, clipping, and
    invalidation unit tests pass;
  - absolute bootstrap placement is usable headlessly; and
  - toolkit-independent multithreaded MVC/MVVC examples compile through the
    public thread-safe composition and owner-marshaling surface with the
    selected serialized-owner topology

### 3. Basic Presentation

- Owner: rendering maintainers
- Dependencies: Core/container and an approved canonical SnapshotV1
- Entry criteria: deterministic tree and geometry tests pass
- Exit criteria:
  - the `foundation.absolute-panels` intended frame matches the specified
    semantic styles, owners, rectangles, nesting, and clipping;
  - one terminal adapter presents it for a human; and
  - headless and terminal paths originate from the same intended frame

### 4. Basic Automation

- Owner: controller and automation maintainers
- Dependencies: Basic Presentation, approved protocol limits, and explicit
  operator acceptance of the initial unauthenticated trust boundary
- Entry criteria: immutable atomic snapshots can be published in-process
- Exit criteria:
  - normal mode creates no automation resource;
  - `expletives-test --automation <socket-path>` provides bounded JSONL
    version 1;
  - observe, raw logical pre-binding key injection, direct commands, resize,
    correlated completion, no-op, failure, final snapshot, and cleanup pass;
  - distinct one-shot `KeyPress` behavior passes without persistent held
    state;
  - held-key disconnect and fairness behavior pass;
  - arbitrary terminal escape bytes are rejected;
  - the reusable Go client and `expletivesctl` exercise the supported
    protocol; and
  - `expletives-test` and `expletivesctl` are built in debug, release, and
    profiling modes at the required artifact paths.

### 5. Basic Layouts

- Owner: layout maintainers
- Dependencies: Basic Automation and approved Layout attachment/overflow
  decisions
- Entry criteria: attached automation can inspect rectangles and resize the
  foundational fixture
- Exit criteria:
  - BoxLayout and GridLayout measure/arrange contracts pass ordinary tests;
  - each Layout arranges only direct child Panels, and nesting works through
    child Panels with their own Layouts;
  - colored Panel scenarios prove exact placement at normal, changed, and
    below-minimum geometry;
  - snapshots expose deterministic rectangles, overflow facts, incident
    generation, and notification outcomes;
  - application callback, coalescing, acknowledgement, and nonblocking
    toolkit fallback tests pass without recursion or request hangs; and
  - human and attached clients can inspect the same layout transitions.

### 6. Resume Ordered Common Controls

- Owner: common-control maintainers
- Dependencies: Basic Layouts and the approved catalog roadmap
- Entry criteria: the closed-loop Panel/layout foundation is reliable
- Exit criteria: each subsequent non-deferred control category arrives in the
  directed order with ordinary Go tests and an `expletives-test` scenario;
  structured-input controls remain deferred

## Adoption And Rollout

- **Migration:** There is no implementation to migrate. On approval, update
  the directed specifications and roadmap before creating the public Go API.
- **Compatibility:** Treat `Root()`, constructors, geometry, event kinds,
  command IDs, protocol version, SnapshotV1, and Layout behavior as
  interfaces. During pre-v1 development, record deliberate incompatible
  changes rather than maintaining aliases accidentally.
- **Communication:** Document the mandatory-parent rule, App/root ownership,
  serialized ownership, multithreaded MVC/MVVC/MVP/MVU composition examples,
  exact automation warning, socket lifecycle, protocol limits, and Layout
  authority, atomic attachment, overflow callback, and nonblocking fallback in
  package and command documentation.
- **Rollout:** Deliver the milestones in order so Basic Automation is usable
  before managed-Layout and common-control work. Keep each fixture stable once
  an automation client depends on it.
- **Rollback:** Before a public release, revert an unsuccessful phase to the
  last passing intended-frame/headless boundary and revise the proposal.
  Never retain a knowingly ambiguous protocol or global root for temporary
  compatibility.
- **Retirement:** The absolute fixture remains a regression scenario, but
  ordinary application layout moves to Layouts. Any provisional protocol
  version is retired only after clients receive an explicit versioned
  transition.

## Decision Log

- 2026-07-24 — Drafted from direct operator requirements for an app-owned
  root Panel, early presentation and automation, Basic Layouts, raw
  pre-shortcut key events, direct commands, and MVC-compatible application
  composition. Authentication remains deferred.
- 2026-07-24 — Recorded operator approval of `app.Root()` with no
  package-global root; mandatory container-capable and immutable foundational
  parents with reparenting deferred to the backlog; distinct one-shot
  `KeyPress`; thread-safe multithreaded MVC/MVVC consumption with combined or
  split serialized toolkit owners; BoxLayout/GridLayout terminology and
  direct-child-Panel scope with nesting through child Panels; future
  Panel-based control reuse through Go composition/embedding; and
  `expletivesctl` plus every executable in debug, release, and profiling
  builds. SnapshotV1 field and plane design was unresolved at that point.
- 2026-07-24 — Directed Unicode display scope: one complete grapheme cluster
  is supported only when it occupies exactly one terminal cell. A one-cell
  base-plus-combining cluster is allowed. Width-two and other multi-cell
  glyphs, continuation cells, half-glyph rendering, wide CJK, and wide emoji
  are unsupported. Each unsupported display element renders as exactly one
  `U+FFFD REPLACEMENT CHARACTER` (`�`) cell. SnapshotV1 fields and planes were
  open at that point.
- 2026-07-24 — Directed conservative basic-terminal degradation: definite
  7-bit ASCII and confidently known upper-half code-page mappings render
  directly; every other logical cell uses the closest reasonable
  single-character ASCII approximation in black on yellow, or
  black-on-yellow `?` when no reasonable approximation exists. Unknown and
  remote mappings must not be guessed, and the canonical intended frame stays
  unchanged.
- 2026-07-24 — Recorded operator approval of the canonical Basic
  `SnapshotV1`: each cell record contains its canonical one-cell grapheme,
  semantic style, resolved foreground and background colors, and stable owner
  identity; cursor state and the bounded typed control tree live at snapshot
  level; width and continuation fields or planes do not exist. Also approved
  independent Layout construction with atomic `Panel.SetLayout` attachment,
  and below-minimum behavior that preserves minima, clips, and exposes
  structured overflow. An application may register an `Overflow` callback
  delivered after layout with no internal locks held; events are bounded and
  coalesced, callback execution cannot indefinitely block
  layout/render/automation completion, and absent or failed delivery uses the
  nonblocking toolkit fallback, initially a compact dismissible warning
  overlay. Snapshots expose both overflow state and notification outcome.
