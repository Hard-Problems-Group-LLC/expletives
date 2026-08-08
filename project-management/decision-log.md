# Decision Log

Record durable decisions here with newest entries at the top.

For each entry, include:

- date
- decision
- status when relevant
- rationale
- related files, proposals, or backlog items

## EXPL-DEC-015 — Add Explicit Non-Modal Input Scopes

- Date: 2026-08-08
- Status: Directed and adopted
- Authority: Direct operator request accepting radioradio ECR 2026-005

### Decision

Container-capable controls may declare an immutable construction-time input
scope through a typed `PanelOptions` mode. The zero value declares no new
boundary and preserves existing application behavior. A declared scope owns
the mnemonic and default/cancel-role namespace of its descendants, while the
existing direct-parent focus groups remain the units of spatial and Tab entry.

Declared scopes confine Tab and Shift-Tab by default. A distinct explicit mode
permits traversal to escape to another eligible group. Modal scopes remain
implicit, top-of-stack, and confined; they are not replaced by the new
non-modal facility. The App root remains the fallback scope for controls with
no nearer declared or modal boundary.

Input resolution uses committed toolkit visibility, destruction, selected-tab
state, current focus, and modal state. An atomic Transaction that hides one
workspace and shows another publishes no intermediate resolver state. Core and
automation snapshots expose the structural scope mode through bounded typed
container details and no application content.

### Rationale

Mutually exclusive workspaces need conventional repeated mnemonics such as
Save and Cancel without manufacturing modal dialogs or omitting keyboard
access. A typed immutable boundary is smaller and safer than a reflective
binding system, preserves the existing ownership tree, and lets validation,
focus repair, rendering, and automation observe one canonical structure.

### Compatibility And Boundaries

- Applications declaring no scope retain the current App-wide non-modal
  mnemonic validation, same-parent default/cancel validation, and traversal.
- Duplicate mnemonics or default/cancel roles within one declared scope remain
  transaction errors.
- The change does not reparent controls, alter command ownership, make hidden
  or disabled controls eligible, or replace direct-parent focus groups.
- Directional navigation within composite controls and modal input capture are
  unchanged.

### Adoption Evidence

Phase 19 Slice 19.10 implemented `InputScopeNone`, `InputScopeConfined`, and
`InputScopeEscaping` through `PanelOptions`, scope-aware validation and input
resolution, root/modal structural evidence, explicit automation projection
and fail-closed validation, response-bound accounting, public-consumer
coverage, and confined catalog screens. The complete `make verify` gate passed
at 2026-08-08T10:31:25-07:00.

### Related Records

- [`EXPL-TASK-036`](tasks-in-progress.md)
- [`development-roadmap.md`](development-roadmap.md)
- [`actions-api-v0.md`](../docs/specifications/actions-api-v0.md)
- radioradio ECR 2026-005

## EXPL-DEC-014 — Avoid Host-Terminal Shortcut Collisions

- Date: 2026-07-30
- Status: Directed and adopted
- Authority: Direct operator instruction

### Decision

- Treat terminal-emulator, desktop, and input-method interception as a host
  boundary: a TUI cannot recover a chord consumed before it reaches the PTY.
- Avoid audited Terminator, Ptyxis, and GNOME Terminal shortcuts in project
  defaults, examples, and acceptance paths while retaining arbitrary
  client-selected raw chord support.
- Replace the MenuBar's F10 activation with F9 and retain Ctrl-Space as a
  secondary fallback.
- Use Alt-I, Alt-N, Alt-A, Alt-C, Alt-M, Alt-D, and Alt-P for the
  `expletives-test` File, Panels, Layouts, Controls, Menus, Dialogs, and Help
  roots. Use Alt-G and Alt-R for its Toggle and Reset Button mnemonics.
- Require an alternate visible or navigable route for essential behavior and
  verify project defaults through a controlling PTY as well as structured raw
  automation.

### Rationale

Terminator consumes Alt-L for its Layout Launcher. Ptyxis consumes F10 for
its primary menu, and Ptyxis is the current terminal on RHEL 10, Fedora
Workstation 41 and later, and Ubuntu 26.04 LTS. GNOME Terminal remains
relevant on earlier supported releases and customized or upgraded systems;
its optional menubar consumes conventional menu mnemonics including the
former File and Help chords. The replacement set is not claimed by the
audited default terminal bindings.

### Supersession

This decision supersedes only `EXPL-DEC-013`'s F10 activation default and any
conventional example mnemonic that collides with the host. It retains that
decision's Turbo Vision menu appearance, traversal behavior, Ctrl-Space
fallback, edge ownership, placement, and focus rules.

### Related Records

- [`Terminal Shortcut Compatibility Advisory`](../docs/Terminal-Shortcut-Compatibility.md)
- [`Menus API v0`](../docs/specifications/menus-api-v0.md)
- [`EXPL-TASK-022`](tasks-in-progress.md)

## EXPL-DEC-013 — Anchor Application Chrome And Adopt Turbo Vision Menu Look

- Date: 2026-07-30
- Status: Directed and adopted; activation binding narrowed by
  `EXPL-DEC-014`
- Authority: Direct operator instruction

### Decision

- Treat the Main Menu as root-parented application chrome on the complete
  physical top row, independent of root centering, maximum, or aspect
  constraints and outside ordinary Layout membership.
- Reserve chrome rows from the intersecting root-content rectangle.
- Use Turbo Vision as the Menu appearance and keyboard-behavior reference
  without copying or depending on its implementation.
- Use the classic role distinctions: black on light gray normal Menu,
  red mnemonic letters, black on green selection, selected/ordinary disabled
  roles, single-line popup border, tee separators, and black shadow.
- F10 and Ctrl-Space activate a root label before Down/Enter opens its popup;
  exact Alt mnemonics open a popup directly. Disabled entries remain
  selectable but cannot activate.
- Measure popups from their complete effective entries. Cascade child menus
  with a small offset and backset them left until they fit the right edge
  whenever the physical width permits.
- Deliver Status Bar immediately after Menus, then Headers and Footers. The
  Status Bar is the full physical bottom row; each Header/Footer is exactly
  one row and accepts only a Layout tree compatible with that height.

### Rationale

Global edge chrome must remain predictable when the ordinary application root
is constrained. The Turbo Vision visual and keyboard vocabulary provides the
requested familiar terminal look-and-feel, while semantic style roles retain
theme customizability and native expletives ownership, threading, rendering,
and automation architecture. Measured backsetting prevents nested-menu
content from being truncated merely because the preferred cascade is near an
edge.

### Related Records

- [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
- [`Menus API v0`](../docs/specifications/menus-api-v0.md)
- [`EXPL-TASK-020`](completed-tasks.md)

## EXPL-DEC-012 — Use Allocation-Based Sizing And Independent Borders

- Date: 2026-07-30
- Status: Directed and adopted
- Authority: Direct operator instruction

### Decision

- Remove the 240 by 120 application policy cap. Bound materialized frames by
  a named aggregate cell-allocation budget; 1200 by 1200 is an explicitly
  supported and tested reasonable geometry.
- Make the root fill its offered surface by default, with optional minimum,
  maximum, and terminal-cell aspect-ratio constraints.
- Let Frame/GroupBox controls and Layouts independently select none, single,
  double, light/medium/dark shade, or full-cell borders, with semantic style
  plus optional foreground/background overrides.
- A Layout border belongs to the Layout's stack subtree and consumes a
  one-cell interior inset. Adjacent Frames may select no border so an
  enclosing Layout produces one outline without doubled seams.
- Preserve canonical Unicode line/shade semantics in intended snapshots.
  Project those glyphs to Unicode, DEC Special Graphics, or ASCII according
  to the terminal and locale capability profile.
- Scale retained snapshots by actual cell cost and use a bounded compact run
  representation on the automation wire while preserving expanded cells in
  the public client observation.

### Rationale

Desktop terminal surfaces can be far larger than conventional interactive
terminal sizes. Aggregate allocation and encoded-evidence bounds protect real
resources without turning a historical display size into toolkit policy.
Independent control and Layout decoration supports nested outlines and clean
shared framing while retaining the separate control and arrangement trees.

### Supersession

This decision narrows `EXPL-DEC-011`'s word “nonvisual”: Layouts remain
non-Control arrangement objects with no focus or input behavior, but may now
paint their explicitly configured border decoration.

### Related Records

- [`Root Sizing And Border Contract`](../docs/specifications/root-sizing-and-borders-v0.md)
- [`Foundational Layout API`](../docs/specifications/layout-api-v0.md)
- [`EXPL-TASK-015`](tasks-in-progress.md)

## EXPL-DEC-011 — Define Layout Nesting And Stacking

- Date: 2026-07-30
- Status: Directed and adopted; decoration wording narrowed by `EXPL-DEC-012`
- Authority: Direct operator instruction

### Decision

- Keep Layouts outside the Control tree and separate from immutable control
  parentage. `EXPL-DEC-012` later permits optional Layout border decoration.
- Permit a Layout to contain Panels and nested Layouts. A top-level Layout is
  attached atomically to its owning Panel; a nested Layout has another Layout
  as its arrangement and stacking parent.
- Keep arrangement order separate from paint/Z-order.
- `Panel.Raise` and `Panel.Lower` move the Panel to the top or bottom,
  respectively, among Panel peers in the same Layout.
- `Layout.Raise` and `Layout.Lower` move the complete Layout subtree to the
  top or bottom among Layout peers with the same parent.
- Raising or lowering an already-extreme item is a successful no-op and does
  not publish a redundant snapshot.
- Moving one peer kind preserves the relative order and stack slots of the
  other peer kind. Layout contents therefore move as one deterministic
  stacking context without changing measured or arranged rectangles.

### Rationale

This provides wxSizer-like nested composition while preserving the separate
control tree, and supplies the stacking contexts needed by future absolute or
overlay Layouts without coupling focus, control ownership, or arrangement
order to Z-order.

### Related Records

- [`Layouts and Overflow`](../docs/specifications/layouts-and-overflow.md)
- [`EXPL-TASK-009`](completed-tasks.md)

## EXPL-DEC-010 — Reserve The Panel For Major Issues

- Date: 2026-07-30
- Status: Directed and adopted
- Authority: Direct operator instruction to reduce overhead and administrative
  complexity

### Decision

- Make ordinary implementation, localized bug fixes, refactoring, tests,
  documentation, and diagnostics use normal verification and focused peer
  review.
- Do not treat a specialist consultation or sub-agent review as a formal Panel
  unless it is explicitly convened.
- Reserve the formal Panel for genuinely major, cross-cutting,
  hard-to-reverse decisions that ordinary design work cannot settle.
- Select a proportional odd subset of three, five, or nine relevant expert
  lenses. Reserve all nine for foundational or product-wide architecture.
- Default to one lean brief, one bounded findings record, one round, targeted
  follow-up, and one closure record. Do not automatically require separate
  design and delivery Panels, an independent chair, a custom manifest, or one
  file per finding.
- Close the already-completed `EXPL-REV-001` correction cycle with shared
  implementation evidence and a concise chair addendum rather than another
  nine-seat review.

### Rationale

The comprehensive initial foundation review found important design problems,
but its 39 raw findings included 17 duplicates and required substantial
packet, relay, and adjudication work. That cost was acceptable for the initial
architecture freeze and is disproportionate for routine delivery. Confidence
should come primarily from clear contracts, automated tests, closed-loop
automation, and focused review.

### Related Records

- [`Panel Of Experts Charter`](reviews/panel-of-experts-charter.md)
- [`Development process efficiency log`](reviews/process-efficiency-log.md)
- [`EXPL-REV-001 design evidence`](reviews/decisions/EXPL-REV-001-design-evidence.md)

## EXPL-DEC-009 — Adopt A Nine-Seat Panel Of Experts

- Date: 2026-07-25
- Status: Directed and adopted
- Authority: Direct operator instruction to form and document the board and
  its operating standards

### Decision

- Establish a standing, risk-triggered Panel of Experts with nine distinct
  review seats:
  maintainability; orthogonality and design/inheritance elegance;
  extensibility; customizability; client ease of use; testability; security;
  performance; and reliability/concurrency/portability.
- Use a separate, distinct, non-voting evidence chair to freeze packets,
  validate evidence, preserve findings and dissent, control dispositions, and
  maintain the review record.
- Keep the Panel advisory. There is no majority approval or risk acceptance.
  Scope, approval, deferral, rejection, and accepted risk remain with the
  operator.
- Use risk triggers, separate design and delivery reviews, immutable versioned
  packets, reproducible SHA-256 manifests, independent first-round findings,
  no more than three rounds per stage, all-nine-seat lens coverage, recorded
  recusal and replacement, controlled severities and dispositions, and
  evidence-backed gate resolution.

### Rationale

The toolkit's public hierarchy, concurrency model, terminal boundary, and
automation control plane create coupled risks that benefit from independent
specialist scrutiny. Nine seats preserve all eight requested lenses while
adding reliability, concurrency, and portability without creating an
even-sized board. A separate evidence chair keeps record quality and evidence
adjudication independent of the expert lenses.

The absence of voting is deliberate: an odd membership avoids a structurally
even board, but a headcount still cannot waive a supported safety or
correctness issue.

### Decision Boundary

This decision adopts the review process and authorizes convening it when the
charter triggers apply. It does not approve the current foundation design,
complete its implementation, dispose of any future finding, or accept risk.

### Related Records

- [`Panel Of Experts Charter`](reviews/panel-of-experts-charter.md)
- [`EXPL-REV-001 foundation design packet`](reviews/packets/EXPL-REV-001-foundation-design-v1.md)
- [`EXPL-TASK-014`](tasks-in-progress.md)

## EXPL-DEC-008 — Select The First Runnable Go, Terminal, And Automation Baseline

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator instruction to proceed with the runnable demo

### Decision

- Use module path `github.com/Hard-Problems-Group-LLC/expletives`, root package
  `expletives`, Go 1.25 as the minimum line, and Go 1.26.5 as the initial
  normal verification toolchain.
- Build pure-Go Linux product artifacts with `CGO_ENABLED=0`; use cgo only for
  test-only race-detector execution.
- Pin `github.com/rivo/uniseg` `v0.4.7` behind the project-owned one-cell
  normalization boundary.
- Implement the first physical backend as an internal CGO-free Linux adapter
  for tested xterm-, screen-, and tmux-family profiles. It owns termios,
  alternate-screen, cursor, input, presentation, resize, and catchable
  teardown. It does not claim generic ANSI, curses, ncurses, terminfo, or
  cross-platform support.
- Build both `expletives-test` and `expletivesctl` in debug, release, and
  profiling modes. Debug disables optimization/inlining; release and profiling
  retain symbols, use normal optimization and `-trimpath`, and publish mode
  metadata. Profiling starts no listener.
- Use a bounded, strict JSON Lines version-1 protocol over the exact explicit
  Unix-socket path. Permit one controller and sequential requests. Support
  snapshots, later-sequence waits, raw key lifecycle events, direct commands,
  retained-result queries, honest cancellation, source reset, and orderly
  shutdown.
- Bind source-local Control-down, `r`-press, Control-up to
  `fixture.toggle`. Direct commands are `fixture.toggle`, `scenario.reset`,
  and `app.quit`.

### Rationale

These choices create a useful closed loop without pretending that the first
raw Linux presenter solves terminal portability. Matching the repository
origin avoids an artificial module rename. A pinned grapheme implementation
is safer than a temporary rune-per-cell model. One controller and sequential,
bounded requests keep the unauthenticated first protocol deterministic while
still exercising the intended input and command paths.

### Decision Boundary

This decision covers Core/Containers, Basic Presentation, and Basic
Automation only. Basic Layouts, Overflow API details, general terminal
support, other operating systems, a public external-control extension model,
authentication, and later protocol expansion remain separate work.

### Related Records

- [`First Runnable Implementation Baseline`](../docs/specifications/implementation-baseline-v0.md)
- [`Build And Verification`](../docs/specifications/build-and-verification.md)
- [`expletives-test`](../docs/specifications/expletives-test.md)
- [`EXPL-TASK-014`](tasks-in-progress.md)

## EXPL-DEC-007 — Fix Basic Snapshot, Layout Attachment, And Overflow Semantics

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator answers to the remaining foundational questions

### Decision

- The minimal Basic Automation `SnapshotV1` cell record contains:
  - one canonical one-cell grapheme after Limited Unicode normalization;
  - its semantic style;
  - its resolved foreground and background colors; and
  - its stable owner identity.
- Cursor state and the bounded typed control tree are snapshot-level data.
  Basic `SnapshotV1` cell records have no width or continuation fields.
- A Layout is constructed independently, then attached atomically with
  `Panel.SetLayout(layout)`. Consumers do not provide the owner Panel to the
  Layout constructor. A failed attachment must not expose a partially
  installed Layout or disturb either object, the control tree, logical
  geometry, effective clips, or the current frame.
- When available geometry is below declared Layout minima, the Layout
  preserves those minima and resulting logical rectangles, constrains
  effective paint, hit testing, cursor placement, and presentation by every
  ancestor clip and the application surface, and exposes a structured overflow
  fact to application code, semantic observation, and automation.
- An application may register an Overflow callback. It is queued only after
  the applicable layout result has been committed and internal Layout locks
  have been released; it is never called inline from measure/arrange or while
  such locks are held.
- Callback execution uses a bounded application dispatcher with documented
  cancellation/deadline behavior rather than a Layout, render, presentation,
  or UI owner. Panic, saturation, shutdown, or failure to return a disposition
  within the bound records delivery failure and selects the default fallback;
  the toolkit does not create unbounded replacement goroutines.
- When an application handler applies, exactly one bounded delivery attempt is
  made per continuous Panel/Layout overflow episode. Without an applicable
  handler, exactly one default-notification attempt is made. Deficit changes
  update snapshots without another attempt. Recovery ends the episode, so a
  later transition into overflow notifies again.
- An absent, declined, failed, cancelled, timed-out, or unavailable handler
  selects the default fallback. Interactive geometry gets one compact
  dismissible application-level warning with an `OK` action. A tiny terminal
  gets a bounded high-visibility indicator; zero-sized and headless execution
  retains nonblocking semantic evidence.
- The request that exposes overflow may complete once the structured fact and
  bounded notification-queued state are published. It waits for neither
  callback execution, disposition, nor `OK`. Callback and fallback changes
  publish later sequenced snapshots linked to the same overflow episode.
- A tiny-terminal and reentrancy guard prevents the fallback surface, or
  layout changes made in response to overflow, from recursively producing
  unbounded callbacks, warnings, or layout passes.

### Rationale

The selected cell record is sufficient to inspect exact intended text,
semantics, colors, and control ownership without reviving width-two or
continuation-cell machinery. Snapshot-level cursor and tree data avoid
duplicating global or structural facts in every cell.

Independent construction plus atomic `Panel.SetLayout` keeps Layout setup
composable while giving Panel ownership one explicit commit point. Preserving
minima makes constraints honest; clipping and structured overflow reporting
keep constrained geometry deterministic and observable. Deferred, coalesced
notification avoids lock inversion and callback storms, while an
asynchronous, observable fallback remains useful to humans without hanging
closed-loop automation.

### Decision Boundary

The exact Overflow registration scope, handler signature, payload, return or
disposition values, deadline duration, dispatcher capacity, cancellation and
test-scheduler APIs, exact warning text and style, smallest visible indicator,
simultaneous-episode aggregation limit, dismissal command name, and diagnostic
field names remain design work. Those details cannot replace the directed
compact `OK` warning, tiny-terminal indicator, zero/headless semantic fallback,
or the queued, lock-free, bounded-dispatch, episode-coalesced,
automation-safe, and recursion-bounded behavior above.

The project-owned Unicode segmentation/width-data version, optional
post-Basic snapshot evolution, and Layout replacement/detachment details
remain open. `EXPL-DEC-011` and `layout-api-v0.md` later resolved nested
Layout stacking, manual-overlay coexistence, and the exact Overflow API.

### Related Records

- [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
- [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
- [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
- [`docs/specifications/layouts-and-overflow.md`](../docs/specifications/layouts-and-overflow.md)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`EXPL-Q-001`](open-questions.md#active-questions)
- [`EXPL-Q-010`](open-questions.md#resolved-questions)
- [`EXPL-Q-012`](open-questions.md#resolved-questions)
- [`EXPL-TASK-013`](completed-tasks.md)

## EXPL-DEC-006 — Degrade Unicode Conservatively On Basic Terminals

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator clarification

### Decision

- A physical terminal adapter renders printable characters that map cleanly to
  7-bit ASCII directly.
- Characters in the upper half of an 8-bit code page, or an equivalent local
  terminal repertoire, render directly only when their exact mapping and
  one-cell behavior are definitely known.
- Every other logical cell is physically approximated by the closest
  reasonable single printable ASCII character in black on yellow. When no
  reasonable approximation exists, the adapter uses black-on-yellow `?`.
- Uncertain configuration, including a code page that cannot be established
  across a remote connection, uses the highlighted ASCII fallback rather than
  a guessed high-bit mapping.
- Physical degradation does not mutate application data or the canonical
  intended frame. A terminal lacking black and yellow uses a documented,
  observable high-visibility monochrome fallback.

### Rationale

Basic and remote terminals do not always expose enough information to prove
which high-bit repertoire is active. Conservative highlighted substitution
avoids mojibake and unsafe control-byte guesses while keeping every logical
element one cell and making loss of fidelity visible.

### Decision Boundary

The versioned approximation table, terminal-profile API, capability evidence,
encoding mechanism, monochrome warning style, and backend diagnostic fields
remain design work. `$TERM` or locale alone is not proof of an exact code-page
mapping.

### Related Records

- [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
- [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`EXPL-Q-003`](open-questions.md#active-questions)

## EXPL-DEC-005 — Limit Displayed Unicode To One Cell

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator clarification

### Decision

- The toolkit only supports displayed Unicode grapheme clusters whose final
  measured terminal width is exactly one cell.
- A composed grapheme containing multiple Unicode code points is supported
  when the complete displayed element fits in one cell.
- Multi-cell glyph rendering is not supported. The canonical frame has no
  width-two lead cells, continuation cells, partial glyphs, or neighboring
  cell reservations.
- Each unsupported multi-cell, zero-width, ambiguous-width, or otherwise
  unrenderable grapheme cluster is displayed as one
  `U+FFFD REPLACEMENT CHARACTER` (`�`) occupying one cell.
- View measurement, cursor movement, selection, clipping, Layout geometry,
  snapshots, automation, and terminal presentation all use that one-cell
  representation. Application data may retain the original text.

### Rationale

One displayed element per cell keeps horizontal cursor movement, selection,
editing, hit testing, borders, clipping, and Layout measurement consistent
from row to row. Multi-cell glyphs would introduce continuation ownership,
partial clipping, cursor-position, overlap, and terminal-width disagreements
that are intentionally outside the initial toolkit.

Using the conventional replacement character preserves deterministic
geometry and makes unsupported text visible without attempting partial or
backend-dependent rendering.

### Decision Boundary

The project still must select the remaining canonical frame fields and the
versioned Unicode segmentation/width data. General UTF-8 acceptance does not
imply wide-glyph presentation support.

### Later Clarification

`EXPL-DEC-006` distinguishes canonical and physical presentation. Measurement,
view state, snapshots, automation, and the terminal adapter's logical input
retain the one-cell `U+FFFD` representation. A physical adapter preserves that
one-cell geometry but may substitute a highlighted single-cell ASCII
approximation or `?` when the terminal cannot definitely encode the canonical
value.

`EXPL-DEC-007` resolves the minimal Basic snapshot fields. Each cell stores
one normalized one-cell grapheme, semantic style, resolved foreground and
background colors, and owner identity, with no width or continuation fields.

The original phrase “ambiguous-width” refers to a cluster the selected,
versioned project policy cannot deterministically classify as exactly one
cell. It is not a categorical rejection based only on the Unicode East Asian
Width `Ambiguous` property. A complete cluster with that property is supported
when the pinned policy deterministically measures it as one cell.
Cursor state and the bounded typed control tree remain at snapshot level.

### Related Records

- [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
- [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
- [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
- [`EXPL-DEC-006`](#expl-dec-006--degrade-unicode-conservatively-on-basic-terminals)
- [`EXPL-DEC-007`](#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`EXPL-Q-001`](open-questions.md#active-questions)

## EXPL-DEC-004 — Root, Input, Concurrency, Layout, And Build Foundations

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator answers to `EXPL-PROP-001`

### Decision

- `app.Root()` is the sole root accessor. There is no package-global root.
  The App-owned special root `Panel` is the only parentless control.
- Every ordinary control constructor requires a valid container-capable
  parent. Parent ownership is immutable during the foundational phases;
  reparenting is separate backlog work.
- `KeyPress` is a distinct one-shot raw logical event at the pre-shortcut
  boundary. It does not leave held-key state. Stateful chords use ordered
  modifier `KeyDown`, non-modifier `KeyPress`, and modifier `KeyUp` events.
- The toolkit must fit well to very well into most MVC, MVVC, and similar
  consuming-application architectures, including multithreaded ones.
  Toolkit-owned public behavior must be thread-safe.
- Thread safety does not require concurrent mutation of internal UI state.
  The implementation may serialize the event loop, renderer, and presentation
  logic onto one owner thread/goroutine each or a shared owner when required
  or strongly recommended. Cross-owner calls must use documented safe
  marshaling, ordering, cancellation, and shutdown behavior.
- Use **Layout** as the project term. The initial instantiated layout objects
  are `BoxLayout` and `GridLayout`.
- Basic Layouts only need to arrange direct child `Panel` objects relative to
  their parent `Panel`. Nesting is achieved through Panels that own their own
  Layouts. Most controls are expected to derive directly or indirectly from
  Panel behavior through the approved Go composition/embedding model.
- `expletivesctl` is a supported executable and automation client. Every
  current or future executable/build target must be produced in debug,
  release, and profiling modes under `build/<mode>/<name>`.

### Rationale

An explicit App root supports multiple isolated applications and
model/controller instances. Raw key lifecycle state tests the same shortcut
resolver as physical input. Thread-safe public boundaries allow worker-heavy
and multithreaded application designs while a serialized UI pipeline preserves
deterministic state, rendering, and terminal ownership.

“Layout” describes measurement and arrangement directly and avoids making a
wxWidgets-specific term part of the public vocabulary. Restricting the first
Layouts to Panel relationships keeps the foundational API small while still
covering most controls through their shared Panel basis.

Building every command in every mode prevents diagnostic or automation tools
from silently diverging from the executable matrix.

### Decision Boundary

The exact Go mechanism by which controls derive or compose Panel behavior
remains part of the public extension-model decision. Thread safety covers
toolkit-owned state and documented toolkit calls; it cannot make arbitrary
consumer-owned model objects safe without their cooperation.

The canonical cell/frame representation remains open. In particular, the
operator requested clarification of “grapheme” before approving the proposed
grapheme-capable `SnapshotV1`.

### Later Clarification

`EXPL-DEC-005` resolves the displayed-text portion of this boundary: complete
grapheme clusters are supported only when they occupy exactly one cell, and
every unsupported display element renders as exactly one `U+FFFD` cell. The
remaining canonical frame fields and versioned Unicode segmentation/width
data were still open at that point. `EXPL-DEC-007` subsequently fixes the
minimal Basic snapshot fields; only the versioned Unicode-data selection and
possible post-Basic evolution remain open.

### Related Records

- [`docs/specifications/application-architecture.md`](../docs/specifications/application-architecture.md)
- [`docs/specifications/concurrency-and-thread-safety.md`](../docs/specifications/concurrency-and-thread-safety.md)
- [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
- [`docs/specifications/build-and-verification.md`](../docs/specifications/build-and-verification.md)
- [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
- [`EXPL-DEC-005`](#expl-dec-005--limit-displayed-unicode-to-one-cell)
- [`EXPL-DEC-007`](#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`EXPL-TASK-012`](backlog.md)

## EXPL-DEC-003 — Support MVC-Like Consuming Applications

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator request

### Decision

The public toolkit architecture must support consuming projects organized as
MVC or a similar explicit separation. Application models remain independently
testable and need not inherit from toolkit types. Controls and layouts form
the view, while structured commands and events can be routed to a controller,
presenter, update function, or equivalent application layer.

The requirement is architectural compatibility, not a mandate for one named
framework. Foundational design must avoid package-global roots, hidden domain
state in controls, terminal I/O from application models, and unsynchronized
control mutation from background workers.

`expletives-test` will demonstrate the chosen separation as an external
public-API consumer.

### Rationale

Separating domain state, view state, and application decisions improves
ordinary Go testing, enables headless composition, and lets physical input and
automation share application behavior without exposing a private model
backdoor.

### Decision Boundary

This decision does not yet approve a public `Controller` interface, reactive
runtime, automatic two-way data binding, mandatory package layout, or
inheritance hierarchy. The foundational proposal recommends neutral
composition and command-routing primitives first.

### Related Records

- [`docs/specifications/application-architecture.md`](../docs/specifications/application-architecture.md)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`project-management/development-roadmap.md`](development-roadmap.md)
- [`EXPL-DEC-004`](#expl-dec-004--root-input-concurrency-layout-and-build-foundations)

## EXPL-DEC-002 — Control Delivery Order And Basic Automation Input

- Date: 2026-07-24
- Status: Directed; Layout terminology and scope superseded by `EXPL-DEC-004`
- Authority: Direct operator requests

### Decision

- Deliver the foundational control work in this order:
  1. Core and Containers;
  2. Basic Presentation;
  3. Basic Automation; and
  4. Basic Sizers.
- Basic Presentation must first prove multiple root and nested `Panel`
  controls with different background colors, fixed placement, clipping, and
  overlap.
- Basic Automation must support the explicit invocation
  `expletives-test --automation <socket-path>`, including the concrete
  development form
  `expletives-test --automation /tmp/expletives.sock`.
- Automation may submit raw key lifecycle events named `KeyDown`, `KeyUp`,
  and `KeyPress`. They enter before mnemonic, accelerator, hotkey, and command
  resolution so tests can hold and release modifier keys and form chords.
  Direct semantic command submission remains a separate supported path.
- Basic Sizers must include `BoxSizer` and `GridSizer` for inserting and
  arranging controls after they have been created with their actual parent.
- After Basic Sizers, deliver the remaining common-control categories in
  catalog order: Text and Display, Actions, Selection, Text and Numeric
  Input, Progress and Status, Navigation and Chrome, Scrolling and Content,
  Collections, and Modal Controls.
- `FormPanel`, `Wizard`, and `StepContainer` are deferred together as
  Structured Input controls.
- Every ordinary control is created with another control as its parent. The
  root exception, parent capability, exact accessor name, sizer ownership,
  and undersized-layout behavior must be settled through the foundational
  proposal before implementation freezes the public API.

### Rationale

Presentation first gives the container tree a physical, human-visible proof.
Automation immediately afterward creates the closed-loop observation and
input surface that will test every later phase. Sizers then replace bootstrap
absolute placement before the public control catalog grows.

Raw key lifecycle injection is required in addition to direct commands:
command-only automation cannot prove the real modifier-state, mnemonic,
accelerator, hotkey, edit-gate, and configurable interrupt paths.

### Decision Boundary

This decision does not approve a package-global root, the exact root accessor
name, a final Go parent interface, the wire schema, arbitrary terminal
escape-byte injection, the terminal backend, or hardened automation
authentication. Recommended answers are under review in `EXPL-PROP-001`.

### Later Clarification

`EXPL-DEC-004` approves `app.Root()`, the one-shot `KeyPress` meaning,
thread-safe multithreaded application support, and the all-modes
`expletivesctl` target. It also replaces the earlier “Sizer” terminology and
scope with Panel-only instantiated `BoxLayout` and `GridLayout` objects.

### Related Records

- [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
- [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
- [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
- [`project-management/development-roadmap.md`](development-roadmap.md)
- [`project-management/deferred.md`](deferred.md)

## EXPL-DEC-001 — Product, Test, Automation, And Build Direction

- Date: 2026-07-24
- Status: Directed
- Authority: Direct operator requests

### Decision

- `expletives` is the primary product: a reusable Go TUI toolkit.
- `expletives-test` is a supported secondary product and public-API consumer,
  not a disposable demo. It is human runnable and must exercise every public
  UI element and important state.
- A normal `expletives-test` run is human driven. The attached drive-and-
  observe interface exists only after an explicit per-process
  `--automation <socket-path>` option.
- Initial attached automation may be unauthenticated. Passing
  `--automation <socket-path>` is the operator's trust decision for a
  controlled, non-risky environment. Authentication and capability
  authorization are deferred to a later proposal or backlog item.
- Closed-loop development must support request-correlated raw key lifecycle
  events and semantic commands, immutable intended-frame and semantic-state
  inspection, human problem reports, and ordinary Go debugger attachment.
- Use normal Go unit and integration tests wherever practical, supplemented
  by headless, interactive-catalog, attached-automation, PTY, race, fuzz,
  profiling, and real-terminal verification.
- Every project and test application must build in debug, release, and
  profiling modes at `build/<mode>/<name>`.
- A root Makefile must provide working `clean`, `all`, `build`, `release`, and
  `profiling` targets at minimum.
- Ctrl-C behavior must be robust, complete, configurable, and distinct from
  back, cancel, and quit. Menu accelerators, mnemonics, hotkeys, and keyboard
  fallbacks are first-class control requirements.
- Win32, wxWidgets, Motif, LessTif, curses, ncurses, terminfo, Turbo Vision,
  and other established toolkits are required research context. They are not
  compatibility or dependency decisions.

### Rationale

The interactive application creates a shared, observable reproduction surface
throughout toolkit development. Conventional Go tests prove ordinary package
behavior; semantic headless and attached automation shorten the
report-inspect-drive-debug-regress loop; physical terminal tests retain
evidence for behavior that intended frames cannot prove.

Deferring authentication keeps the first local development control plane
small. Default-off activation and conspicuous risk documentation remain
mandatory because drive and observation are consequential.

### Decision Boundary

This decision establishes product outcomes and build/test interfaces. It does
not approve the older draft's framebuffer model, `panelith` package names,
public inheritance pattern, terminal backend, module/cgo policy, automation
wire protocol, build flags, or terminal support matrix. Those require
separate decisions and updated specifications.

### Related Records

- [`docs/product-goals.md`](../docs/product-goals.md)
- [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
- [`docs/specifications/build-and-verification.md`](../docs/specifications/build-and-verification.md)
- [`docs/research/ui-toolkit-lessons.md`](../docs/research/ui-toolkit-lessons.md)
- [`project-management/development-roadmap.md`](development-roadmap.md)
- [`project-management/backlog.md`](backlog.md)
- [`project-management/open-questions.md`](open-questions.md)
