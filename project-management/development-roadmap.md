# Development Roadmap

Use this record for durable phase and workstream planning. It complements the
ordered backlog and the active-task record; it does not replace either.

## Maintenance Rules

- Add only approved or explicitly directed scope.
- Link phases and workstreams to proposals, specifications, and decisions.
- Update status when the source records change.
- Do not put credentials, host identifiers, or other local state here.

## Status Terms

- `planned`
- `active`
- `blocked`
- `complete`
- `deferred`

## Phases

### 1. Foundational Decisions

- Status: `completed`
- Goal: turn the directed product outcomes into coherent public contracts
  before implementation freezes conflicting design models.
- Dependencies: none.
- Includes:
  - reconcile the canonical cell/frame and snapshot representation around the
    directed one-terminal-cell displayed text unit and Limited Unicode scope,
    using the approved minimal Basic `SnapshotV1` cell record and
    snapshot-level cursor and bounded typed tree;
  - choose the Go module/package and public extension model;
  - define seams that let consuming applications keep domain/model state,
    view rendering, and controller transitions separate through
    multithreaded MVC, MVVC, or an equivalently disciplined structure,
    without requiring one framework-owned helper API;
  - define thread-safe public calls, serialized UI/render/presentation owners,
    immutable publication, callback/reentrancy, cancellation, and shutdown;
  - decide the application-scoped root-window and parent/child ownership
    contract;
  - define semantic events, accelerators, hotkeys, configurable interrupts,
    and one-owner event-loop behavior;
  - select the terminal-backend and support boundary;
  - define the initial `--automation <socket-path>` transport, protocol,
    client, bounds, lifecycle, and explicit unauthenticated trust model;
  - define `BoxLayout` and `GridLayout` ownership, Panel-item, and
    measure/arrange contracts around independent construction, atomic
    `Panel.SetLayout` attachment, minima-preserving clipping, observable
    overflow, and safe application notification;
  - define Go/toolchain, Makefile, build-mode, artifact, and verification
    contracts; and
  - use comparative toolkit research as evidence for proposals and decisions.
- `expletives-test` scenario: establish the versioned catalog and state-matrix
  contract that every later phase must extend.
- Normal Go tests: none until the module exists; decisions must identify the
  executable formatting, vet, unit, integration, race, fuzz, and build checks
  that later phases will run.
- Acceptance gate: approved decisions and specifications resolve the active
  foundational questions, including the consumer architecture seams, without
  treating the older Draft as authorization.
- Related:
  - [`docs/product-goals.md`](../docs/product-goals.md)
  - [`docs/specifications/application-architecture.md`](../docs/specifications/application-architecture.md)
  - [`docs/specifications/concurrency-and-thread-safety.md`](../docs/specifications/concurrency-and-thread-safety.md)
  - [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
  - [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
  - [`docs/specifications/layouts-and-overflow.md`](../docs/specifications/layouts-and-overflow.md)
  - [`docs/specifications/build-and-verification.md`](../docs/specifications/build-and-verification.md)
  - [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
  - [`EXPL-DEC-001`](decision-log.md#expl-dec-001--product-test-automation-and-build-direction)
  - [`EXPL-DEC-002`](decision-log.md#expl-dec-002--control-delivery-order-and-basic-automation-input)
  - [`EXPL-DEC-003`](decision-log.md#expl-dec-003--support-mvc-like-consuming-applications)
  - [`EXPL-DEC-004`](decision-log.md#expl-dec-004--root-input-concurrency-layout-and-build-foundations)
  - [`EXPL-DEC-005`](decision-log.md#expl-dec-005--limit-displayed-unicode-to-one-cell)
  - [`EXPL-DEC-006`](decision-log.md#expl-dec-006--degrade-unicode-conservatively-on-basic-terminals)
  - [`EXPL-DEC-007`](decision-log.md#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)

### 2. Core/Container

- Status: `complete`
- Goal: establish the executable project and the ownership, identity,
  geometry, lifecycle, and pure intended-frame behavior of `Panel`, `Frame`,
  and `GroupBox`.
- Dependencies: the Phase 1 module, frame, extension, root-window, ownership,
  and build decisions needed to avoid freezing the wrong public API.
- Includes:
  - the Go module, package and command skeletons, root Makefile, and
    debug/release/profiling artifacts;
  - one special application-owned root `Panel`, the only parentless control;
  - parent-required construction for every other control;
  - a fixed initial parent for each control, with reparenting deferred;
  - the approved Go composition/embedding model through which most controls
    are conceptually `Panel`-derived directly or indirectly;
  - stable control identity, deterministic child order, clipping, stacking,
    visibility, enabled state, and geometry; and
  - pure headless painting with semantic styles and cell ownership, kept
    separate from consumer model state and controller transitions.
- `expletives-test` scenarios: `core.container-tree` and
  `core.frames-and-groups`, plus a noninteractive scenario-list/self-check
  path before physical presentation exists.
- Normal Go tests: root and parent invariants, cycle, cross-application, and
  reparenting rejection, construction/destruction behavior, coordinate
  transforms, clipping,
  stacking, background fill, borders, titles, immutable frames, and an
  external-package public-consumer integration test whose model, controller,
  and view responsibilities remain independently testable.
- Acceptance gate: the three controls are complete headlessly; imports and
  construction perform no I/O or hidden concurrency; all required Make
  targets and mode-specific `expletives-test` artifacts work.
- Completion evidence:
  [`EXPL-TASK-014`](completed-tasks.md).

### 3. Basic Presentation

- Status: `complete`
- Goal: present the Core/container intended frame through a real terminal
  while preserving one-owner lifecycle and teardown.
- Dependencies: Phase 2 and the approved terminal-backend/support decision.
- Includes: minimal capability and semantic-style mapping, damage
  presentation, cursor policy, resize, one terminal-owner event loop, safe
  startup and catchable teardown, minimal quit/interrupt commands, and
  canonical displayed text units that each occupy exactly one terminal cell.
  A basic terminal emits definite ASCII and known code-page mappings directly;
  every other logical cell uses a deterministic single-character ASCII
  approximation, or `?`, in black on yellow.
- `expletives-test` scenarios:
  - `presentation.colored-panels` places multiple Panels with distinct
    backgrounds at fixed, inspectable rectangles;
  - `presentation.nesting-and-clipping` exercises overlap and child clips; and
  - `presentation.resize` shrinks below the useful minimum and recovers.
- Normal Go tests: fake-backend frame/presentation parity, damage, style and
  cursor mapping, supported one-cell grapheme/combining sequences,
  exact one-cell `U+FFFD` replacement of unsupported display elements without
  alignment changes, event ordering, resize and write failures, plus PTY smoke
  tests for startup, quit, Ctrl-C, resize, and terminal restoration.
- Acceptance gate: a human can run the scenario in all three build modes,
  observe the intended panel positions and backgrounds, resize safely, and
  exit without leaving the terminal damaged; a normal run creates no
  automation resource. `expletives-test` performs its application transitions
  outside the terminal painter and demonstrates the supported consumer
  separation rather than depending on toolkit internals.
- Completion evidence:
  [`EXPL-TASK-014`](completed-tasks.md).

### 4. Basic Automation

- Status: `complete`
- Goal: establish the closed-loop observe/drive path before the control
  catalog grows.
- Dependencies: Phase 3 and the approved snapshot, event-outcome, transport,
  protocol, fairness, and endpoint-lifecycle contracts.
- Includes:
  - the in-process headless driver;
  - explicit `expletives-test --automation /tmp/expletives.sock` attached
    operation;
  - atomic immutable frame plus bounded typed semantic-view observation;
  - a root `Snapshot` explicitly projected to a distinct
    `automation.SnapshotV1`, with each cell containing one canonical one-cell
    grapheme, semantic style, resolved foreground/background colors, and
    stable owner identity, with no width or continuation fields;
  - snapshot-level cursor state and bounded typed control tree;
  - stable scenario listing, selection, and reset;
  - unique request IDs, explicit outcomes, and associated frame sequences;
  - a supported client path for observing frames, injecting raw
    `KeyDown`/`KeyUp`/`KeyPress` lifecycle events before shortcut resolution,
    and separately submitting direct semantic commands;
  - the reusable Go client and `expletivesctl` command in all three build
    modes; and
  - bounded messages, clients, queues, requests, responses, and fair
    scheduling with human input.
- `expletives-test` scenario: `automation.closed-loop` observes and validates
  `presentation.colored-panels`, resets it, completes an explicit no-op, and
  receives the final snapshot for an exit-producing request.
- Normal Go tests: default-off startup, protocol encode/decode and malformed
  input, request correlation across unrelated redraws, no-op/rejection/
  cancellation/timeout/exit, queue saturation, fairness, endpoint collision
  and cleanup, process integration, concurrent client/controller/shutdown
  race and deadlock detection, exact `automation.SnapshotV1` schema and
  atomic-pair checks, and protocol fuzzing.
- Acceptance gate: a client can connect to the operator-enabled endpoint,
  inspect exact Panel rectangles, backgrounds, owners, and semantic state,
  inject an event, and receive the correctly correlated frame. Initial
  automation is conspicuously unauthenticated, removes only its own endpoint,
  and remains absent without the flag.
- Completion evidence:
  [`EXPL-TASK-014`](completed-tasks.md).

### 5. Basic Layouts

- Status: `complete`
- Goal: replace bootstrap absolute positioning with deterministic reusable
  Panel layout.
- Dependencies: Phase 4 and the approved Panel-parent, Layout ownership, and
  remaining measure/arrange and Overflow-handler API contracts.
- Includes:
  - independent Layout construction followed by atomic ownership attachment
    through `Panel.SetLayout`, with failed attachment leaving both objects,
    the control tree, logical geometry, effective clips, and current frame
    unchanged;
  - instantiated horizontal and vertical `BoxLayout` objects;
  - instantiated row-major `GridLayout` objects;
  - direct child Panel items and nested Layout items, without control
    reparenting;
  - gaps and insets, plus per-Panel item minimum size, grow weight, expansion,
    and alignment;
  - stable integer remainder allocation;
  - nesting through both child Panels that own Layouts and Layout objects
    inserted in another Layout;
  - one primary and optional sibling top-level Layout contexts;
  - independent arrangement and stacking order, with Panel and Layout
    `Raise`/`Lower` operations;
  - preservation of declared minima when space is insufficient, with
    the resulting logical rectangles retained, presentation constrained by
    every ancestor clip and the application surface, and structured overflow
    exposed to the application, semantic snapshot, and automation;
  - an application-registerable Overflow callback queued after the layout
    result is committed and internal Layout locks are released, coalesced once
    per continuous overflow episode and dispatched with bounded
    cancellation/deadline behavior outside the UI, Layout, render, and
    presentation owners; and
  - one semantically observable default-notification attempt when no handler
    applies or delivery is declined, panics, is saturated, is cancelled,
    times out, or shuts down: a compact dismissible `OK` warning when it fits,
    a bounded high-visibility tiny-terminal indicator, or zero/headless
    semantic evidence. The triggering request completes with overflow and
    notification queued, before callback disposition or dismissal.
- `expletives-test` scenario: `layouts.basic`, using a vertical Box, nested
  three-column Grid, a child Panel-owned Layout, and two top-level Layout
  layers. Stable commands raise/lower a Panel peer and a complete Layout
  layer for correlated frame inspection.
- Normal Go tests: exact table-driven rectangles; empty, hidden, constrained,
  minimum-size, grow, gap, inset, alignment, and remainder cases; rejection
  of non-child Panels, arbitrary non-Panel/non-Layout items, cycles, excess
  depth/capacity, and reparenting; and property/fuzz checks for determinism,
  containment, and
  nonnegative geometry. Here, containment means clipped output remains within
  the intersection of every ancestor clip and the application surface even
  when preserved child minima extend beyond it.
  Include atomic attach failure, minima-preserving clipping, overflow snapshot
  state, post-lock callback ordering, callback reentrancy, episode coalescing
  and recovery, dispatcher saturation, panic, timeout, and a handler that
  ignores cancellation without unbounded goroutines, default dismissal,
  nonblocking automation correlation, and tiny-terminal recursion bounds.
- Acceptance gate: human and attached runs show the expected layout at fixed
  and resized geometries, and automation proves both child rectangles and
  background cells without changing the parent tree. Below-minimum geometry
  remains deterministic and observable; every absent, declined, failed, or
  timed-out application handler yields the directed fallback without blocking
  request completion or recurring without bound. Fixed/stretch spacer objects
  and arbitrary non-Panel/non-Layout items are not part of this phase.
- Completion evidence:
  [`EXPL-TASK-009`](completed-tasks.md).

### 6. Text/Display

- Status: `complete`
- Goal: deliver `Label`, `StaticText`, `Separator`, and `Rule`.
- Dependencies: Phase 5 plus the directed Limited Unicode and approved
  decoration-fallback policies.
- Includes: alignment, wrap, clipping, documented truncation, mnemonic-target
  association where applicable, horizontal/vertical orientation, semantic
  styles, monochrome, reduced decoration, and ASCII fallback.
- `expletives-test` scenarios: `display.text-and-rules`,
  `display.one-cell-unicode`, `display.unsupported-wide`, and
  `display.degradation`.
- Normal Go tests: empty, narrow, aligned, wrapped, clipped, supported
  one-cell grapheme and combining sequences, exact one-cell `U+FFFD`
  replacement of every unsupported display element, post-replacement alignment
  safety, fallback, mnemonic association, separator geometry, and fuzzed
  Limited Unicode rendering cases.
- Acceptance gate: every public display control and important state has
  human, headless, and attached-automation evidence at normal and constrained
  geometries.
- Completion evidence:
  [`EXPL-TASK-016`](completed-tasks.md).

### 7. Actions

- Status: `complete`
- Goal: deliver `Button`, `HotkeyBar`, and `HotkeyBarItem` while completing
  structured command and binding routing.
- Dependencies: Phase 6 and the approved command, focus, mnemonic,
  accelerator, hotkey, and bubbling contracts.
- Includes: focused, pressed, default, cancel, disabled-with-reason, and
  activated states; structured displayed bindings; and non-Alt fallback paths.
- `expletives-test` scenarios: `actions.buttons`, `actions.hotkey-bar`,
  `actions.disabled-command`, and `actions.mnemonic-fallback`.
- Normal Go tests: exactly-once activation, disabled rejection, routing and
  fallback precedence, focus traversal, duplicate binding handling,
  reentrant callback safety, clipping, and parity among keyboard, hotkey, and
  automation activation.
- Acceptance gate: every activation route reaches the same stable command,
  enabled-state check, explicit outcome, and associated frame.
- Completion evidence:
  [`EXPL-TASK-017`](completed-tasks.md).

### 8. Menus

- Status: `complete`
- Goal: deliver `MenuBar`, popup `Menu`, and `MenuItem` immediately after the
  Action controls they invoke, then use them to navigate the growing
  `expletives-test` catalog.
- Dependencies: Phase 7 Button/HotkeyBar activation, command, focus,
  mnemonic, accelerator, and fallback behavior.
- Includes: root-owned full-width physical-row chrome; Turbo Vision
  appearance and keyboard behavior without copying implementation; red
  mnemonic letters, green selection, disabled roles and shadows;
  collision-audited exact Alt popup access; F9 and Ctrl-Space bar activation;
  checked/disabled items;
  measured nested popups with right-edge backsetting; dismissal, clipping,
  resize, focus restoration; start/end top-level placement with conventional
  right-justified Help; scalable File/Panels/Layouts/Controls/Menus/Dialogs/
  Help catalog namespaces; and one active purpose-specific catalog screen.
  The catalog starts on an empty medium-blue Home canvas, exposes Home through
  File, and gives Core Panels, Visual Styles, Box Layout, and Grid Layout
  separate pages. Panel and Layout stacking operations are grouped and routed
  independently.
- `expletives-test` scenarios: `menus.navigation`, `menus.alt-mnemonic`,
  `menus.f9-fallback`, `menus.disabled-command`, and
  `menus.popup-clipping`, and `menus.end-placement`. The persistent MenuBar
  routes current screens through purpose-specific namespaces and holds
  disabled phase-owned future entries so catalog growth does not force every
  demonstration into one cluttered view.
- Normal Go tests: command-route parity, binding precedence and ambiguous Alt,
  root-edge ownership and row reservation, exact palette/mnemonic cells,
  arrow/Home/End navigation, selectable-but-inactive disabled entries,
  separator skipping, measured/backset child popups, nested dismissal, screen
  switching, end-group geometry and popup anchoring, Help/About, clipping/
  resize/focus restoration, and raw automation key lifecycle parity with
  applicable PTY input.
- Acceptance gate: menu, HotkeyBar, Button, mnemonic, direct command, and
  automation routes converge on the same command outcome; the audited exact
  Alt mnemonics and documented fallbacks work; and screen navigation is
  deterministic and observable.
- Completion evidence:
  [`EXPL-TASK-019`](completed-tasks.md) and corrective
  [`EXPL-TASK-020`](completed-tasks.md), extended by
  [`EXPL-TASK-021`](completed-tasks.md) and catalog audit
  `EXPL-TASK-023`.

### 9. Status Bar

- Status: `complete`
- Goal: deliver a root-owned `StatusBar` on the complete physical bottom row.
- Dependencies: Phase 8 application-chrome seam and the shared Action command
  and shortcut model.
- Includes: bounded contextual segments, optional shortcut/action hints,
  narrow-terminal prioritization, exact last-row reservation, root-constraint
  independence, and typed snapshot/automation evidence.
- `expletives-test` scenarios: `status.context`,
  `status.narrow-priority`, and `status.chrome-geometry`.
- Normal Go tests: segment measurement and priority, empty/tiny widths,
  resize, hide/show/destroy row return, constrained roots, command-state
  updates, clipping, and concurrent model updates.
- Acceptance gate: the optional Status Bar owns the first and last cells of
  the last physical row, remains outside ordinary root Layouts, and returns
  its row atomically when hidden or destroyed.
- Contract:
  [`application-chrome-v0.md`](../docs/specifications/application-chrome-v0.md)
  and
  [`status-bar-api-v0.md`](../docs/specifications/status-bar-api-v0.md).
- Completion evidence: [`EXPL-TASK-024`](completed-tasks.md).

### 10. Headers And Footers

- Status: `complete`
- Goal: deliver ordered one-row `Header` and `Footer` chrome containers.
- Dependencies: Phase 9 completes both edge reservations and the shared
  application-chrome geometry seam.
- Includes: Header rows immediately below the Main Menu, Footer rows
  immediately above the Status Bar with most-recent highest ordering,
  root-parent ownership, and atomic validation that an attached Layout tree
  has a complete measured/decorated minimum height of at most one.
- `expletives-test` scenarios: `chrome.headers`,
  `chrome.footers`, `chrome.one-row-layouts`, and
  `chrome.combined-resize`.
- Normal Go tests: stable ordering, add/remove/hide, full-width geometry,
  constrained roots, compatible horizontal Box/Grid Layouts, rejection of
  multirow/inset/gapped/bordered trees, width overflow, tiny surfaces, and
  attached snapshot parity.
- Acceptance gate: every Header/Footer is exactly one physical row, combined
  chrome reserves the canonical edge order atomically, and incompatible
  Layout attachment leaves all state unchanged.
- Completion evidence: [`EXPL-TASK-025`](completed-tasks.md).
- Contract:
  [`application-chrome-v0.md`](../docs/specifications/application-chrome-v0.md)
  and
  [`headers-footers-api-v0.md`](../docs/specifications/headers-footers-api-v0.md).

### 11. Selection

- Status: `complete`
- Goal: deliver `Checkbox`, `RadioButton`, `RadioGroup`, `CycleField`, and
  `SelectField`.
- Dependencies: Phase 7 command/focus behavior and approved stable option and
  selection-model contracts.
- Includes: checked/unchecked and any approved indeterminate state, exclusive
  groups, empty and disabled options, cycling bounds, and distinct focus,
  selection, and activation.
- `expletives-test` scenarios: `selection.checkbox`,
  `selection.radio-group`, and `selection.cycle-select`.
- Normal Go tests: exclusivity, selected-item removal, empty/single/disabled
  options, wrap policy, focus movement, stable values, deterministic change
  callbacks, and resize.
- Acceptance gate: human input, raw automation key lifecycle events, and
  direct semantic commands preserve the same observable selection contract.
- Completion evidence:
  [`EXPL-TASK-026`](completed-tasks.md).

### 12. Text/Numeric Input

- Status: `complete`
- Goal: deliver `TextField`, `NumberField`, `SpinBox`, and `TextArea`.
- Dependencies: Phase 11 plus approved editing, validation, caret, paste, and
  interrupt contracts.
- Includes: committed text distinct from commands, caret and selection over
  supported one-cell grapheme clusters and combining sequences, bounded paste,
  exact one-cell `U+FFFD` replacement of unsupported display elements,
  validation and disabled/invalid reasons, numeric range and step behavior,
  multiline editing, cursor policy, resize, and configurable Ctrl-C behavior.
  `TextArea` uses the internal scroll model later exposed through Phase 15
  controls.
- Directed validator/password addition:
  optional validators require soft or hard enforcement, whitelist or
  blacklist matching, and a nonempty character set. Soft mode retains input
  with green valid state or yellow/red invalid highlighting. Hard mode ignores
  disallowed typed input. Password defaults false, paints `*`, still validates
  the underlying value, and redacts it from snapshot/automation evidence.
- `expletives-test` scenarios: `input.text-field`, `input.numeric`,
  `input.spin`, `input.text-area`, `input.validation`, and
  `input.interrupt-policy`.
- Normal Go tests: edit-operation tables, supported one-cell text-unit
  boundaries, combining sequences, selection replacement, paste and length
  bounds, exact one-cell `U+FFFD` replacement of unsupported display elements
  with alignment preservation, validators, numeric
  parse/overflow/range/step edges, caret visibility, resize, and fuzzed
  Limited Unicode edit sequences.
- Acceptance gate: all editors remain deterministic and recoverable through
  invalid input, paste, resize, focus changes, and every supported Ctrl-C
  policy.
- Completion evidence:
  [`EXPL-TASK-027`](completed-tasks.md).

### 13. Progress

- Status: `complete`
- Goal: deliver `ProgressBar`, `Meter`, `Spinner`, and `ActivityDots`.
- Dependencies: Phase 12 plus approved clock/tick and bounded worker-result
  delivery.
- Includes: determinate and indeterminate state, supplied deterministic ticks,
  reduced motion, textual narrow fallback, completion/error/cancel state, and
  a synchronous cancellation-aware boundary for application-owned worker
  updates. Contextual Status Bar segments were delivered separately in
  Phase 9. Progress controls own no clock, ticker, worker, or terminal access.
- `expletives-test` scenarios: `progress.determinate`,
  `progress.indeterminate`, `progress.terminal-states`, and
  `progress.reduced-motion`.
- Normal Go tests: ratio and rounding boundaries, zero/invalid totals, narrow
  rendering, deterministic animation frames, update coalescing, cancellation,
  concurrent worker-result delivery, race detection, and the no-hidden-worker
  invariant.
- Acceptance gate: progress scenarios stay responsive, bounded, observable,
  deterministic, and cancellation-aware without rendering or terminal access
  from workers.
- Completion evidence:
  [`EXPL-TASK-028`](completed-tasks.md).

### 14. Navigation/Chrome

- Status: `complete`
- Goal: deliver `ScrollBar`, `TabbedPanel`, `Notebook`, and `Tab` on the
  already-complete menu and action foundation.
- Dependencies: Phase 13 plus approved tab focus and viewport contracts.
- Includes: tabs/pages, tab focus and activation, page switching, and
  viewport-derived scrollbar thumbs.
- `expletives-test` scenarios: `navigation.tabs` and
  `navigation.scrollbar`.
- Normal Go tests: tab removal and reorder, focus restoration, page switching,
  and empty/full/tiny scrollbar ranges.
- Acceptance gate: tab and scrollbar behavior remains keyboard complete,
  resize-safe, semantically observable, and reachable from the catalog menu.
- Completion evidence:
  [`EXPL-TASK-029`](completed-tasks.md).

### 15. Scrolling/Content

- Status: `complete`
- Goal: deliver `ScrollablePanel`, `Viewport`, `MarkdownView`, `LogView`, and
  `StreamView`.
- Dependencies: Phase 14 scrolling chrome and the internal scroll model
  established for `TextArea`.
- Includes: bounded offsets, keep-visible behavior, arbitrary clipped
  content, a documented Markdown subset, ring-buffered logs, follow versus
  scrollback, safe streamed text, and honest dropped-content accounting.
  The opening compatibility slice also formalizes per-control horizontal and
  vertical Layout hints/weights and rebuilds Text/Numeric Entry as a compact
  scroll-ready form: single-line fields remain natural-height while multiline
  editors divide the available balance by vertical weight.
- `expletives-test` scenarios: `content.viewport`, `content.markdown`,
  `content.log-follow`, `content.log-scrollback`, and
  `content.stream-drops`.
- Normal Go tests: offset clamping, empty/small/large content, resize,
  Markdown fixtures, control-sequence neutralization, ring wrap/drop counts,
  concurrent producer posting, race detection, and bounded-memory
  benchmarks.
- Acceptance gate: scroll position, follow mode, and drop accounting are
  semantically observable and remain correct under resize and sustained
  bounded input.
- Contract:
  [`scrolling-content-api-v0.md`](../docs/specifications/scrolling-content-api-v0.md).
- Completion evidence:
  [`EXPL-TASK-030`](completed-tasks.md).

### 16. Collections

- Status: `complete`
- Goal: deliver `ListBox`, `ComboBox`, `DropDown`, `TreeView`, `Table`,
  `DataGrid`, `ListItem`, `TreeNode`, and `Column`.
- Dependencies: Phase 15 viewport, scrolling, selection, popup, and editing
  capabilities.
- Includes: stable item identity, empty/loading/error state, single and
  multiple selection, popup selection, tree expansion, row/cell focus,
  sorting, sticky headers, and approved DataGrid editing behavior.
- `expletives-test` scenarios: `collections.list`, `collections.combo`,
  `collections.tree`, `collections.table`, and `collections.data-grid`.
- Normal Go tests: focus and selection across insert/remove/sort, popup
  commit/cancel, tree mutation, table paging/column clipping/sort stability,
  DataGrid validation, large-model bounds, and fuzzed model updates.
- Acceptance gate: each collection preserves focus and selection by stable
  identity, remains keyboard complete, and has deterministic normal, empty,
  large, disabled, and resized evidence.
- Contract:
  [`collections-api-v0.md`](../docs/specifications/collections-api-v0.md).

### 17. Modals

- Status: `complete`
- Goal: deliver `ModalPanel`, `Dialog`, `MessageBox`, `ConfirmDialog`,
  `InputDialog`, and `ProgressDialog`.
- Dependencies: Phase 16 plus approved modal-stack, result, and nested-modal
  policy.
- Includes: centering, stacking, shadows, focus capture/restoration,
  default/cancel buttons, validation, small-screen degradation,
  long-operation cancellation, and distinct back/cancel/interrupt/quit
  outcomes.
- `expletives-test` scenarios: `modal.message`, `modal.confirm`,
  `modal.input`, `modal.progress`, `modal.nested-policy`, and
  `modal.resize-and-focus`.
- Normal Go tests: focus entry/restoration, destructive-default safety,
  Escape/cancel/interrupt distinctions, nested policy, resize, failed
  validation, progress cancellation, exit with a modal open, and z-order/
  clipping.
- Acceptance gate: every modal close path yields an explicit result and
  associated snapshot; focus restoration is exact; Ctrl-C remains distinct
  from cancel, back, and quit.
- Contract:
  [`modals-api-v0.md`](../docs/specifications/modals-api-v0.md).

### 18. File And Directory Pickers

- Status: `complete`
- Goal: deliver a general-purpose Turbo Vision-style `FilePickerDialog` plus
  `MultiFilePickerDialog` and `DirectoryPickerDialog` specializations.
- Dependencies: Phase 16 collection controls and Phase 17 modal/dialog
  lifecycle, focus, validation, and result contracts.
- Includes: application-supplied filesystem/provider models, current-directory
  and path editing, parent navigation, directory and file lists, sorting,
  filtering, typed selection results, inaccessible/missing-path errors,
  single-file activation, explicit multiple-selection commit, directory-only
  selection, resize/small-screen degradation, and safe cancel/interrupt
  behavior. Appearance, focus order, accelerators, default/cancel buttons, and
  keyboard behavior follow Turbo Vision conventions where reasonable.
- `expletives-test` scenarios: `dialogs.file-picker`,
  `dialogs.file-picker-multiple`, `dialogs.directory-picker`,
  `dialogs.file-picker-errors`, and `dialogs.file-picker-resize`.
- Normal Go tests: deterministic provider fixtures, empty/root/parent
  navigation, filtering and sorting, single/multiple/directory result types,
  selection preservation, inaccessible and disappearing entries, validation,
  resize, cancel/interrupt distinctions, and attached automation.
- Acceptance gate: no dialog performs hidden or unbounded filesystem work;
  every close path publishes an explicit typed result and final snapshot; the
  same dialog behavior works against deterministic test providers and an
  explicitly selected local-filesystem adapter.
- Contract:
  [`file-pickers-api-v0.md`](../docs/specifications/file-pickers-api-v0.md).
- Completion evidence:
  [`EXPL-TASK-033`](completed-tasks.md).

### 19. Terminal Compatibility And Operational Hardening

- Status: `complete`
- Goal: validate terminal capabilities, terminfo integration, input parsing,
  resize, signals, suspension/resume, remote transports, physical rendering,
  bounded concurrency, profiling, and failure recovery.
- Dependencies: Phase 18 supplies the complete interaction and control-state
  matrix to exercise at the physical boundary.
- `expletives-test` scenarios: terminal-lab cases for supported one-cell
  Unicode and combining-sequence alignment, exact one-cell `U+FFFD`
  replacement of unsupported display elements, monochrome/color mapping,
  definite direct code-page mappings, conservative black-on-yellow ASCII
  approximation under unknown and remote mappings, cursor placement, paste,
  modified-key fallback, resize, Ctrl-C, suspend/resume, remote transport, and
  output failure. Remote-transport evidence is limited to byte preservation;
  latency and disconnect behavior remain explicit non-claims until separate
  rows exist.
- Normal Go tests: PTY lifecycle and raw-input suites, Limited Unicode
  validation/replacement fuzzing and alignment checks, race and sustained-load
  runs, profiling workloads, failure teardown, and the declared real-terminal/
  transport matrix.
- Directed Slice 19.4 (`complete`): completed the exact-revision physical
  matrix on release `a402cf076358` with native XTerm 366, XTerm plus tmux
  3.2a, XTerm plus GNU Screen 4.8.0 with required `altscreen on`, a
  conservative `C` locale, and an actual OpenSSH 9.9 loopback TCP/PTY
  transport. Each row passed exact-window style/dialog inspection,
  terminal-native paste, authoritative resize, Ctrl-Z/`fg`, immediate
  post-resume keyboard input, Ctrl-C from an open menu with status 130, and
  caller-visible termios restoration. The conservative row additionally
  passed aligned DEC/ASCII and black-on-yellow unsupported-content fallback.
  Loopback proves the actual SSH transport stack and PTY path, not real-
  network latency or disconnect behavior. The completed matrix, narrow claim,
  and repeat procedure are recorded in
  [`Terminal Compatibility Verification`](../docs/Terminal-Compatibility-Verification.md).
- Consumer-driven Slice 19.5 (`complete`): extended `ListBox` with opt-in
  word/cell wrapping
  so one stable logical item may occupy multiple visual rows. Description
  continuations use a hanging indent, current/selection styling covers the
  complete logical item, and compact snapshot/automation evidence reports
  wrap policy and derived visual-row count without retained content.
  The final candidate completed the full verification gate at
  2026-08-07T11:43:43-07:00, including compact over-height leading-row
  behavior. The canonical `feature/listbox-logical-row-wrap` branch repeated
  that complete gate at 2026-08-07T14:52:28-07:00 and is approved for
  permanent origin publication plus a non-fast-forward merge into `main`.
  Consumer pinning must use the resulting immutable revision; no machine-
  local workspace replacement may become part of a consumer commit.
- Consumer-driven Slice 19.6 (`complete`): extended `TextField` with an optional
  live-edit command, process-local current-value access, distinct selected and
  editing semantic styles, and a bounded ordered UTF-8 byte-style policy.
  Preserve the existing commit-only `ChangeCommand`, route every application
  command outside toolkit locks, suppress active byte-band disclosure while a
  password field is masked, and keep the zero-value presentation backward
  compatible. Add the optional submit-command contract requested by the same
  ECR so Enter-to-send can be integrated without a second public-API break.
  Exercise keys, committed input, paste, deletion, cancellation, exact
  multibyte thresholds, whole-entered-text styling, validation, redaction,
  transactions, automation projection, public-consumer compilation, and a
  deterministic `expletives-test` catalog fixture. The complete ordinary,
  PTY, race, build-mode, self-check, and smoke gate passed at
  2026-08-07T16:40:12-07:00; no serial or RF path was used.
- Consumer-driven Slice 19.7 (`complete`): added a zero-compatible, copied
  per-`TextField` canonical UTF-8 byte maximum under `EXPL-BUG-004` and
  radioradio ECR 2026-004. Enforce construction, programmatic replacement,
  printable insertion, selection replacement, committed text, and paste
  atomically. Rejected input changes no value/caret/frame and routes no live
  edit command; Backspace, Left, Delete, and Enter remain available at the
  bound. Extended core/automation evidence, malformed-client validation,
  public-consumer tests, and the maintained catalog with exact ASCII,
  multi-byte, replacement, and zero-value coverage. The full ordinary, PTY,
  race, build-mode, catalog self-check, and smoke gate passed at
  2026-08-07T17:23:42-07:00. Synthetic fixtures only.
- Consumer-driven Slice 19.8 (`complete`): corrected the default
  `button.disabled` palette so retained labels have practical resolved color
  contrast while remaining structurally disabled and visually distinct from
  enabled, focused, default, and pressed Buttons. Added exact Theme, ordinary
  Button, standard-dialog, catalog, and terminal-profile regressions. Intake:
  radioradio ECR 2026-001; resolved target defect `EXPL-BUG-005`. Exact Theme,
  ordinary Button, standard-dialog, catalog, and ANSI-16 regressions pass.
- Consumer-driven Slice 19.9 (`complete`): made the shared viewport clamp
  enforce zero through calculated maximum on both axes. Left/Right boundary
  input remains handled as a state-preserving no-op; construction,
  direct state, reflow, content replacement, resize, local snapshots, and
  attached automation must never publish a negative offset. Intake:
  radioradio ECR 2026-006; resolved target defect `EXPL-BUG-006`. Shared clamp
  and wrapped consumer-equivalent ListBox boundary regressions pass.
- Consumer-driven Slice 19.10 (`complete`): added an immutable, explicitly
  declared non-modal input-scope boundary for container-capable controls.
  Preserved the zero-value App-wide mnemonic and same-parent role behavior;
  declared scopes own descendant mnemonics and default/cancel roles, retain
  direct-parent focus groups, confine Tab by default, and may explicitly
  permit Tab escape. Hidden/destroyed/inactive-modal scopes cannot receive
  input, atomic workspace switches repair focus without an intermediate
  ambiguous resolver, and typed snapshots expose only structural scope mode.
  Intake: radioradio ECR 2026-005; adopted design `EXPL-DEC-015`. Core,
  public-consumer, automation projection/validation/resource proof, modal,
  visibility, clipping, traversal, catalog, and snapshot-copy coverage pass.
  At 2026-08-08T10:31:25-07:00, `make verify` passed the complete ordinary,
  integration, race, build-mode, catalog self-check, and smoke gate for all
  three slices.
- Directed Slice 19.11 (`complete`): made white the default foreground for
  Dialog, ModalPanel, standard-dialog, and file-picker body text on the
  established medium-gray `#808080` surface. Record this as the explicitly
  approved and endorsed permanent Turbo Vision contrast variance in
  `EXPL-DEC-016`; independent Button and shadow roles remain intact. Live
  attached acceptance found and closed `EXPL-BUG-007`: `expletives-test`'s
  complete custom Theme still supplied the prior black body foreground. The
  library and maintained public consumer now have exact Theme and rendered-
  cell coverage. The complete gate and a rebuilt temporary attached instance
  pass, with live `#FFFFFF` on `#808080` at frame 76.
- Directed Slice 19.12 (`complete`): relocated the reserved blank interior row
  below standard-dialog bottom Button sections immediately above those
  Buttons. Dialog height, two-row raised Button bounds and shadows, horizontal
  spacing, focus, and close behavior remain unchanged. Exact geometry coverage
  passes for MessageBox, ConfirmDialog, InputDialog, and cancellable
  ProgressDialog. The complete gate passed, and rebuilt attached frame 76
  retained a 126 by 8 MessageBox with two blank rows above OK and its bottom
  border immediately below. Operator-restarted final acceptance repeated that
  evidence at frame 84 and restored the session at frame 85.
- Directed Slice 19.13 (`complete`): repaired `EXPL-BUG-010`, found during the
  attached native-terminal narrow-resize acceptance. The root-owned overflow
  fallback must word-wrap a plain explanation, place `[OK]` on its own
  following centered row when height permits, retain a bounded high-visibility
  indicator below the explanatory threshold, and preserve the existing
  structured episode, dismissal, color, ownership, and non-recursive behavior.
  Focused and complete automated gates pass. Rebuilt native frame 117 at 12
  by 42 paints centered `Layout`, `overflow`, and `[OK]` rows in the exact
  root-owned black-on-yellow warning style while retaining the active
  structured deficit; the operator accepted the visible result.
- Acceptance gate: the declared PTY and real-terminal matrix passes; teardown
  is reliable on catchable paths; final automation outcomes remain honest;
  and supported performance and resource bounds have evidence.

### 20. Release Readiness

- Status: `in progress`
- Goal: stabilize the public Go API, documentation, examples, compatibility
  policy, release builds, provenance, security review, and downstream
  consumption.
- Dependencies: Phase 19 and completion of every non-deferred catalog phase.
- `expletives-test` scenario: a catalog-completeness check fails whenever an
  exported control or required important state lacks a stable scenario and
  state-matrix entry.
- Normal Go tests: minimum/current Go and supported-target matrices, clean
  module and dependency checks, vulnerability analysis, downstream external
  consumer tests, release smoke tests, debugger/profiler checks, and
  reproducibility evidence.
- Slice 20.0 (`active`): inventory the exported API, non-deferred control
  catalog, specifications, examples, and maintained test surfaces; repair only
  concrete coverage/documentation gaps found by that mechanical audit.
  - Consumer contract repair began at 2026-08-11T01:26:22-07:00 under
    `EXPL-BUG-009`. A downstream Models selector supplied a width-only
    `MinimumSize`; `DropDown` retained the zero height, so a vertical-natural
    Layout allocated no row and the selected label owned no framebuffer
    cells. Preserve the caller's width while enforcing the directed one-row
    construction minimum for both `DropDown` and `ComboBox`, prove geometry
    and painted selected text through a real BoxLayout, then run the complete
    project gate on a permanent feature branch. No terminal, network service,
    or downstream domain operation is in scope.
  - Consumer contract repair completed at 2026-08-11T01:34:02-07:00.
    `DropDown` and `ComboBox` now preserve one row under partial caller
    minima; direct sizing, real Layout geometry, selected-label painting,
    contrast, ordinary/race tests, all build modes, and smoke checks pass.
  - Downstream-approved `TextArea` read-only work began at
    2026-08-11T04:30:22-07:00 under `EXPL-TASK-038`. Add a construction-time
    `ReadOnly` attribute and, because the existing transaction behavior seam
    supports it directly, atomic live `ReadOnly`/`SetReadOnly` operations.
    Read-only areas remain focusable for bounded viewport navigation and
    selection but never enter edit mode or accept mutations. Provide distinct
    normal/focused semantic styles, versioned snapshot/automation evidence,
    public API/specification coverage, and full verification on permanent
    branch `feature/text-area-read-only` before merging to `main`.
  - `EXPL-TASK-038` completed at 2026-08-11T04:45:44-07:00. Construction and
    live atomic read-only state, focusable viewport/selection navigation,
    mutation refusal, normal/focused semantic styles, typed core/automation
    evidence, and normative specifications are complete. Focused tests and
    the full formatting, vet, ordinary, integration, race, all-mode build, and
    smoke gate pass. Permanent feature-branch ACP and merge to `main` follow
    before downstream consumption.
  - Consumer contrast repair began at 2026-08-11T10:34:45-07:00 under
    `EXPL-BUG-011`. The default Theme violates the directed text-input
    contract by giving `application.root` and every focused TextArea role the
    same black background. Give editable focus a white surface, focused
    read-only state a yellow surface, retain legible validation/selection
    colors, and prove both Theme resolution and rendered read-only behavior on
    permanent branch `feature/text-input-focus-contrast` before ACP and merge
    to `main`.
  - `EXPL-BUG-011` closed at 2026-08-11T10:41:18-07:00. Default editable
    focus is black on white, focused read-only TextArea presentation is black
    on yellow, and validation/selection remain legible and distinct. Exact
    Theme and rendered-cell regressions plus the complete project gate pass.
- Slice 20.1 (`planned`): verify supported Go/toolchain/target declarations,
  module hygiene, dependency provenance, and available vulnerability checks.
- Slice 20.2 (`planned`): verify debug, release, and profiling artifacts,
  executable metadata, debugger/profiler entry points, provenance, and
  reproducibility boundaries.
- Slice 20.3 (`planned`): exercise clean external-consumer builds and complete
  the maintainable developer crash-course/reference path without duplicating
  normative specifications.
- Slice 20.4 (`planned`): run the clean-checkout release gate, reconcile all
  non-deferred roadmap/catalog state, and prepare the explicit version/tag
  decision without publishing a tag absent operator authorization.
- Acceptance gate: project-owned verification passes from a clean checkout;
  every required binary exists and is executable at each documented
  `build/<mode>/<name>` path; all non-deferred controls are documented and
  covered; and tagged artifacts match the public contract.

## Explicitly Deferred Control Family

### Structured Input

- Status: `deferred`
- Controls: `FormPanel`, `Wizard`, and `StepContainer`.
- Authority: direct operator request on 2026-07-24.
- Reason: the foundational controls, automation loop, Layouts, fields,
  navigation, scrolling, collections, and modal behavior must mature first.
- Reactivate: only after an explicit operator reprioritization; create a
  dedicated phase with normal Go tests and `expletives-test` catalog coverage
  rather than silently absorbing these controls into another phase.

### Layout And Ownership Extensions

- Status: `deferred`
- Features:
  - fixed spacer and stretch spacer objects;
  - arbitrary non-Panel/non-Layout items; and
  - control reparenting.
- Reason: Basic Layouts now support nested Layout objects while retaining
  immutable control ownership; spacers, foreign item kinds, and reparenting
  need separate contracts.
- Reactivate: only after an explicit ownership and API decision, with normal
  Go tests and `expletives-test` coverage for the approved behavior.

## Phase-wide Verification Rule

From Phase 2 onward, every public control or meaningful state added by a
phase requires ordinary Go unit and public-consumer integration tests plus a
deterministic `expletives-test` scenario. From Phase 4 onward, the same
scenario also requires attached-automation observation and interaction
coverage. Headless, attached, PTY, race, fuzz, profiling, and real-terminal
checks supplement rather than replace normal Go tests.

Across every phase, preserve separable consumer domain/model state,
controller transitions, pure intended-view rendering, terminal adaptation,
and the one terminal owner. The toolkit must support MVC or an equivalently
disciplined application structure without forcing one exact MVC helper API.
`expletives-test` is the maintained external public-API consumer that
demonstrates and tests this separation; it must not gain a private
architecture bypass.

Across every frame, control, editor, stream, snapshot, and terminal adapter,
canonical displayed text units occupy exactly one terminal cell. Supported
one-cell grapheme clusters and combining sequences remain intact. Width-two
or other multi-cell glyphs and continuation cells are not supported. Every
unsupported display element becomes exactly one `U+FFFD` cell under
[`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md) and
must never disturb clipping, alignment, borders, cursor placement, or later
columns. Basic terminals render only definite ASCII and known code-page
mappings directly; every other logical cell uses a black-on-yellow
single-character ASCII approximation or `?`, without mutating the canonical
frame.
