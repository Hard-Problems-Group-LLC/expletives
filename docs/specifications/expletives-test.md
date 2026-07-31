# `expletives-test` Interactive and Automation Requirements

Status: Directed requirements; the first runnable foundation is implemented
Authority: Direct operator requests on 2026-07-24
Related decisions: `EXPL-DEC-001` through `EXPL-DEC-007` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)
Related work: `EXPL-TASK-001` and the follow-up work in
[`project-management/backlog.md`](../../project-management/backlog.md)
Related Layout contract:
[`layouts-and-overflow.md`](layouts-and-overflow.md)

## Purpose

`expletives-test` is a supported secondary product of the `expletives`
project. It is a human-runnable interactive application that exercises the
toolkit as a real consumer and provides a shared closed-loop surface for
visual diagnosis, semantic frame inspection, automated interaction, and
debugger-assisted development.

This specification defines required outcomes. The first runnable foundation's
exact Go and wire contracts are fixed by
[`go-api-v0.md`](go-api-v0.md) and
[`automation-protocol-v1.md`](automation-protocol-v1.md). Later control,
Layout, authentication, terminal-profile, and protocol-version work remains
separate.

## Relationship To `expletives`

The `expletives` library is the primary product. It owns the reusable
controller, layout, rendering, input, widget, snapshot, headless-driver, and
terminal-adapter contracts.

`expletives-test` must:

- consume supported public library APIs rather than depend on accidental
  internals;
- use the same controller, renderer, semantic events, and terminal painter
  that downstream applications use;
- expose every public control and important cross-control behavior through a
  discoverable interactive catalog;
- provide deterministic, non-sensitive fixtures and stable scenario
  identities; and
- evolve with the library rather than being postponed until the control
  library is otherwise considered complete.

A specialized public testing API is acceptable when it is intentionally
designed, documented, and useful to other consumers. A private backdoor that
makes only `expletives-test` observable is not.

## Operating Modes

### Normal Human Mode

Running `expletives-test` without `--automation` starts an ordinary
human-driven TUI.

A normal invocation must not create:

- an automation listener or endpoint;
- an automation server goroutine or worker;
- a discovery or connection record;
- a capture directory or capture artifact; or
- another drive-and-observe control-plane resource.

The absence of attached automation must be testable.

### Attached Automation Mode

Running `expletives-test --automation <socket-path>` explicitly enables a
local drive-and-observe control plane for that process. The application must
use the supplied Unix-socket path and make the active mode and endpoint
location visible to the operator.

For the initial implementation, `--automation <socket-path>` is the opt-in
trust boundary. Application-level peer authentication and capability
authorization are deferred. The mode is therefore intended only for an
operator-controlled, non-risky environment. Anyone able to reach the endpoint
may be able to observe all visible data and perform every action reachable
through the UI. Documentation and startup output must state that consequence
plainly.

The initial endpoint must still:

- use an application-owned local location;
- avoid following attacker-controlled links;
- refuse an existing path rather than deleting an unverified object;
- bound clients, requests, messages, queues, text, geometry, and responses;
- use explicit framing, versions, timeouts, and structured failures;
- close cleanly and remove only the endpoint created by the current process;
  and
- leave no endpoint or related artifact after normal shutdown.

Authentication, capability-specific authorization, hardened multi-user use,
and any elevated-process automation require a separate approved proposal.
They are deliberately not implied by this initial contract.

The supported external client is the reusable Go automation-client package
and its `expletivesctl` command. `expletivesctl` must be able to observe
snapshots, submit each raw key lifecycle event, invoke a direct command, query
completion, and request orderly exit. It is part of the all-modes executable
inventory.

### Headless Test Mode

In-process headless tests are distinct from attached automation. They call
the real controller and intended-frame renderer without a terminal painter,
socket, discovery mechanism, or external controller.

Most behavior should be proven headlessly first. Attached automation exists
to inspect and drive the real interactive process when terminal adaptation,
human observation, scheduling, or an integration defect makes that valuable.

### Root Geometry Options

The current fixture accepts optional cell-based root policy flags:

```text
--root-min-width N
--root-min-height N
--root-max-width N
--root-max-height N
--root-aspect-width N
--root-aspect-height N
```

The aspect terms must be supplied together. With no root-policy flags, the
root and catalog expand to the complete terminal or headless container.
Maximum/aspect constraints center the catalog within that container; minimum
constraints remain observable when the physical container is too small.
Headless `--width` and `--height` configure the physical surface and are
bounded by aggregate frame cells, not by a conventional terminal axis.

## Interactive Coverage

The application must maintain an inspectable catalog that covers every
public UI element. Each catalog entry needs a stable scenario ID, a concise
purpose, deterministic reset behavior, and the important applicable states.
After the Action controls exist, the following Menu phase adds a persistent
MenuBar that selects one purpose-specific catalog screen at a time. Screen
selection is a normal stable command reachable through menu input and
automation; it must not create a private test-only navigation path.

The coverage inventory must include, as the library gains them:

- panels, frames, labels, buttons, selection controls, fields, progress and
  activity controls, separators, scrollbars, menus, status and hotkey bars,
  tabs, scrollable views, text areas, log views, lists, combo boxes, trees,
  tables, dialogs, and pop-up menus;
- layout managers, clipping, stacking, overlays, shadows, and modal focus
  capture and restoration;
- normal, focused, selected, activated, editing, disabled-with-reason,
  invalid, empty, loading, error, and completed states where meaningful;
- keyboard-only operation, structured menu accelerators such as `Alt-F`,
  component hotkeys, fallback key paths, and inspectable bindings;
- configurable interrupt behavior, including robust Ctrl-C handling in
  ordinary views, editors, modals, long-running operations, and automation;
- supported one-cell grapheme clusters, including combining sequences,
  clipping boundaries, ASCII fallback, monochrome, and reduced-decoration
  behavior;
- deterministic one-cell `U+FFFD` replacement of every unsupported display
  element
  without shifting cells, corrupting geometry, or leaving partial output;
- direct basic-terminal rendering only for definite 7-bit ASCII and known
  code-page mappings, with every other logical cell presented as a
  black-on-yellow single-character ASCII approximation or `?`;
- small and changing geometry, resize during editing and overlays, scroll
  limits, and below-minimum Layout behavior, including structured overflow,
  application handling, warning fallback, and recovery;
- bounded streaming and honest dropped-content reporting; and
- cursor visibility and placement during text entry and non-editing focus.

The first foundation scene visibly distinguishes control and Layout
decoration: a single-line outer Frame, a double-line GroupBox, an unbordered
Frame, independently bordered Grid and nested Layouts, and light, medium,
dark, and full-cell shade forms. The unbordered Frame combined with its
bordered child Layout is the regression fixture for clean shared framing
without doubled seams. The same public-API scene embeds an aligned Label with
an observable mnemonic target, word-wrapped StaticText, a double-line
Separator, and a titled Rule in two child BoxLayouts. Their stable automation
keys are `display.label`, `display.static_text`, `display.separator`, and
`display.rule`.

The Actions catalog embeds default/focused, ordinary, disabled-with-reason,
and cancel Buttons in `panel.nested`, plus a live HotkeyBar in the automation
status Frame. Their stable keys are `action.toggle`, `action.reset`,
`action.disabled`, `action.quit`, and `action.hotkeys`. Raw Alt-T, Tab plus
Enter/Space, the existing Ctrl-R binding, and direct command invocation all
enter the shared command router. The Toggle command's checked presentation
state changes with the same controller transition observed in the accent
Panel.

The authoritative inventory and order are in
[`control-catalog.md`](control-catalog.md). `FormPanel`, `Wizard`, and
`StepContainer` are explicitly deferred Structured Input controls; they do
not count as missing catalog coverage unless that workstream is reactivated.

The application must remain useful to a human while attached automation is
active. Ordering and fairness must prevent either sustained local input or
automation from starving the other indefinitely. A configured interrupt or
emergency path must remain reachable.

## Root `Snapshot` And `automation.SnapshotV1`

The in-process Basic observable state is one immutable, atomic root-package
`Snapshot`. Attached automation receives a distinct, versioned
`automation.SnapshotV1` explicitly projected from that root snapshot. Both
represent the same published UI state. At snapshot level each contains:

1. intended-frame geometry and its cell matrix;
2. cursor state;
3. a bounded typed control tree with stable control and scenario identity,
   parentage, computed rectangles, focus, selection, modes, enabled actions,
   validation, Layout overflow, and other allowlisted meaning needed for
   diagnosis;
4. the monotonic frame sequence and final-state marker; and
5. any request-completion association published with that state.

Each intended-frame cell contains only these logical values:

- the canonical one-cell grapheme;
- semantic style;
- resolved foreground and background; and
- stable owner identity.

Cells do not carry a width, continuation marker, cursor, or unrestricted
application data. Cursor and the bounded typed control tree occur once at
snapshot level. The root and wire representations are specified separately;
their explicit projection preserves this semantic schema without sharing
mutable storage or Go type aliases.

The frame and control tree must describe one atomic UI state. The control tree
uses typed, versioned, allowlisted fields rather than an unrestricted
domain-data dump.

The attached protocol carries large frames as bounded row-major cell runs.
The supported client validates and expands them, so `expletivesctl` retains
the familiar indexable `snapshot.frame.cells` view while also exposing
`snapshot.frame.runs`.

Unicode behavior and acceptance scope are governed by
[`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md). Combining
sequences that satisfy that policy remain one cell. Every unsupported display
element must become exactly one `U+FFFD` replacement cell before it can corrupt
frame geometry.

The incompatible byte-oriented multi-plane and width/continuation-cell models
in [`ui-toolkit-requirements.md`](ui-toolkit-requirements.md) do not govern
the root `Snapshot` or `automation.SnapshotV1`. A later automation snapshot
version requires an explicit compatible decision rather than silently
changing version 1.

Each published state has a monotonic frame sequence. Sequence numbers support
observation and change detection; they do not replace command request IDs.
Diagnostic timing, redraw counts, backend damage, export acknowledgements,
and physical-presentation evidence must not make a logically identical
canonical frame nondeterministic.

## Layout Attachment And Overflow Scenarios

`expletives-test` must construct `BoxLayout` and `GridLayout` independently
and attach them through `Panel.SetLayout` and `Panel.AddLayout`. The
`layouts.basic` fixture uses a vertical Box containing a nested three-column
Grid, a child Panel with its own Layout, and two top-level Layout layers.
`layout.panel.raise`, `layout.panel.lower`, `layout.layer.raise`, and
`layout.layer.lower` let a human or attached client verify that arrangement
rectangles remain fixed while Panel peers or complete Layout subtrees change
stack order. A scenario must also demonstrate a successful atomic initial
attachment and show that every rejected attachment leaves the Panel, Layout
objects, logical geometry, effective clips, and published frame unchanged.
Replacement and detachment are not assumed.

A separate deterministic scenario constrains a parent Panel below the combined
minimum of its Layout items. It must demonstrate all of these outcomes:

- logical minima and resulting logical Layout rectangles are preserved, while
  paint, hit testing, cursor placement, and presentation obey the effective
  ancestor clip;
- the snapshot's typed control tree exposes structured overflow;
- a registered application `Overflow` callback or handler can accept the
  notification;
- a declining handler and an absent handler both select the same deterministic
  fallback;
- the fallback is a compact dismissible warning overlay with an `OK` action
  when there is enough room;
- a tiny terminal uses a bounded high-visibility indicator, while zero-sized
  and headless modes retain nonblocking semantic evidence;
- repeated layout passes coalesce one continuing overflow episode instead of
  recursively opening warnings or delivering another callback when only the
  deficit changes;
- callback panic, dispatcher saturation, deadline expiry, and a handler that
  ignores cancellation select the observable fallback without blocking the UI
  owner or creating unbounded workers; and
- resolution clears the active overflow episode so a later recurrence can be
  reported.

Overflow notification and fallback presentation occur after the layout pass
and outside internal locks. The warning overlay is outside the failing Layout
and cannot create a recursive overflow report. A resize, key event, or direct
automation command that causes overflow may complete once structured overflow
is published and bounded notification delivery is queued. Its associated
root `Snapshot` marks notification state as pending or current, and attached
automation observes the corresponding `automation.SnapshotV1`; completion
does not wait for callback execution or `OK`. Handler disposition and fallback
changes publish later sequenced snapshots. The detailed state, coalescing,
and fallback contract is in
[`layouts-and-overflow.md`](layouts-and-overflow.md).

## Semantic Event Submission And Completion

Attached and headless automation never depend on terminal-library numeric key
codes or an unbounded stream of terminal escape bytes. They support three
correlated submission levels:

- raw key lifecycle events `KeyDown`, `KeyUp`, and `KeyPress`, using the
  toolkit's stable key identity and source-local modifier state;
- normalized committed text, bounded paste, resize, and other non-key input
  events; and
- direct stable commands or widget-semantic actions for precise controller
  tests and diagnosis.

`KeyDown` and `KeyUp` maintain pressed-key state for their automation source,
including modifier keys. `KeyPress` is a distinct one-shot raw logical event
that does not leave a key held. Ordered down/press/up sequences must be able
to create chords such as `Alt-F` and `Ctrl-S` and enter the same
context-sensitive mnemonic, accelerator, hotkey, and command resolver used by
physical input.

Pressed-key state is isolated by input source: automation cannot release a
human-held key or another client's virtual key. Disconnect, session reset,
and shutdown must clear or synthesize release for that source so modifiers
cannot remain stuck. Exact key identity, left/right modifier, repeat, and text
fields require a versioned schema.

The Basic Automation acceptance fixture must bind one documented
application-level modifier chord to a deterministic colored-Panel scene
change or reset. Its test sends modifier `KeyDown`, non-modifier `KeyPress`,
and modifier `KeyUp`, then verifies both the resolved command outcome and its
associated snapshot. This exercises the raw-key path before menu controls
exist.

Direct commands must still obey enabled state, modal scope, validation, and
other application rules. Text, paste, menu accelerators, hotkeys, navigation,
activation, resize, interrupt, and transport failure remain distinct event
classes. Arbitrary escape-stream and signal-byte behavior belongs in PTY
tests.

The conceptual contract is:

```text
submit(request_id, raw_key_or_normalized_input_or_semantic_command)
    -> completion(request_id, outcome, frame_sequence)

observe(frame_sequence)
    -> immutable automation.SnapshotV1
```

For every accepted request:

- the request ID is unique within its documented lifetime;
- the one authoritative UI owner applies the event in defined order;
- the completion reports the actual outcome and associated frame sequence;
- a valid no-op completes explicitly even when the frame is unchanged;
- rejection, cancellation, interruption, exit, and failure are distinct
  outcomes;
- an unrelated timer, resize, background result, human key, or redraw cannot
  satisfy the request;
- a timeout reports failure or indeterminate state, never success; and
- an event that exits the UI publishes and associates the final snapshot
  before the request is closed.

The protocol must define duplicate IDs, retries, cancellation, late
completion, result retention, shutdown with requests in flight, and debugger
pauses. In particular, pausing under a debugger may cause a client timeout,
but it must not turn an unknown outcome into success or cause the event to be
silently replayed.

Queues are bounded. Ordering and fairness among human input, injected input,
resize, clock/tick events, and background results are documented and tested.
No automation worker may call the terminal backend directly.

All concurrent server, client, controller, snapshot, and shutdown behavior is
governed by
[`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md).

## Ctrl-C, Interrupts, And Keyboard Commands

Ctrl-C handling must be robust, complete, and configurable. It is not
hard-coded universally as quit, cancel, or emergency exit.

The application and library must:

- distinguish interrupt, cancel, back, and quit as semantic operations;
- define a safe default interrupt policy;
- allow applications to configure or handle Ctrl-C without losing a
  documented operator recovery path;
- make behavior explicit in editors, modals, menus, long operations, and
  automation-driven sessions;
- preserve terminal restoration and final-state publication on catchable
  interrupt paths; and
- test Ctrl-C as a terminal signal, a decoded input event where applicable,
  an injected semantic interrupt, and a concurrent event during other work.

Menu accelerators, mnemonics, global hotkeys, and control-local bindings are
structured data. Essential actions need a fallback when Alt or modified keys
are unavailable or ambiguous. Text entry and paste must not be reinterpreted
as command shortcuts.

## Observation, Capture, And Debugging

Intended semantic observation is the primary diagnosis surface. It does not
prove what a physical terminal displayed.

Evidence must distinguish:

1. the intended semantic frame and widget view;
2. bounded backend or presentation traces; and
3. physical terminal or emulator evidence.

Reverse capture of terminal output cannot reconstruct semantic widget state,
style IDs, or original grapheme ownership and must not become a competing
source of truth.

Live observation and persisted capture are separate operations. Persisted
captures, if added, require explicit enablement, bounded size, a project-owned
location, restrictive permissions, redaction, retention, and cleanup rules.
Deterministic test fixtures must not contain credentials or private operator
data.

The debug build of `expletives-test` must support normal Go debugger use.
Debugger attachment does not implicitly enable automation, bypass the
`--automation <socket-path>` boundary, or relax protocol outcome rules.

## Verification

Use layered evidence:

1. ordinary Go unit tests for controller transitions, layout, cells,
   controls, configuration, and failure behavior;
2. ordinary Go integration tests for real owned package wiring and public
   consumer behavior;
3. complete deterministic headless sessions through the same controller and
   renderer;
4. automated `expletives-test` catalog and state-matrix coverage;
5. attached `--automation <socket-path>` tests for correlation, fairness,
   bounds, malformed input, cleanup, final snapshots, and nonblocking overflow
   fallback;
6. PTY tests for process startup, terminal modes, raw input decoding, resize,
   signals, output, and teardown;
7. a small supported real-terminal matrix for one-cell grapheme and combining
   behavior, `U+FFFD` replacement of every unsupported display element,
   basic-terminal code-page and highlighted ASCII degradation, color, cursor,
   paste, suspend/continue, and visual fidelity; and
8. race, fuzz, and profiling work focused on concurrent scheduling, terminal
   input, supported one-cell Unicode and all unsupported display categories,
   snapshot/protocol parsing, and bounded-resource behavior.

The build and test entry points are governed by
[`build-and-verification.md`](build-and-verification.md).

## Acceptance Criteria

- A human can navigate a stable catalog and exercise every shipped public
  control and documented important state.
- A normal run creates no automation control-plane resource.
- `--automation <socket-path>` visibly enables the initial unauthenticated
  local endpoint with an explicit risk warning and safe lifecycle.
- A client with access to the operator-enabled endpoint can observe atomic
  immutable `automation.SnapshotV1` values, submit raw key lifecycle events
  or direct semantic commands, and associate every completion with the
  correct frame.
- No-ops, rejection, cancellation, timeout, interruption, debugger pauses,
  and exit are represented honestly.
- Human and injected input remain bounded and fairly serviced by one terminal
  owner.
- In the canonical frame, supported one-cell grapheme and combining fixtures
  render deterministically, and every unsupported display element becomes
  exactly one `U+FFFD` cell without shifting cells or corrupting layout.
- In physical basic-terminal output, definite ASCII and known code-page
  mappings render directly; other cells use a black-on-yellow
  single-character ASCII approximation or a black-on-yellow `?` when no
  reasonable approximation exists.
- Known toolkit border forms use the best supported Unicode, DEC Special
  Graphics, or ASCII structural glyphs while preserving their configured
  border style.
- The default root fills the offered surface; optional centered
  minimum/maximum/aspect policy works, and a 1200 by 1200 headless frame can
  be allocated, compacted for automation, decoded, and inspected.
- Ctrl-C and other interrupt behavior is configurable, documented, and tested
  across relevant modes.
- A below-minimum Layout preserves logical minima and the resulting logical
  rectangles while effective ancestor clips constrain paint, hit testing,
  cursor placement, and presentation. It publishes structured overflow, and
  its application-handling, compact warning, tiny-terminal, and headless
  fallback paths are deterministic, coalesced, dismissible where applicable,
  and cannot hang automation.
- A reported visual or interaction defect can be reproduced, inspected,
  debugged, and converted into a narrow regression test using this surface.

## Non-Goals

- Making attached automation safe for arbitrary hostile or multi-user
  environments in the initial implementation.
- Replacing ordinary Go unit and integration tests with one interactive app.
- Claiming semantic observation alone proves terminal-emulator fidelity.
- Requiring a debugger or attached endpoint for deterministic headless tests.
- Treating visual resemblance as source or binary compatibility with Turbo
  Vision, Win32, wxWidgets, curses, or another toolkit.
