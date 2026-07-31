<!-- FIELDMANUAL_MANAGED_HEADER_START -->
# FieldManual Project Instructions

This project uses FieldManual for shared development, testing, documentation,
review, and project-management practices.

Before substantial work:

1. Read `FieldManual-config.toml` and, when it exists,
   `.local/FieldManual-localconfig.toml`.
2. Confirm the configured `FieldManual_ProjectRoot` and
   `FieldManual_FrameworkRoot` identify the project being changed and the
   FieldManual checkout being used.
3. Read `FieldManual/README.md` and
   `FieldManual/standards-and-practices/core/README.md`.
4. Load only the language, technology, technique, and knack documents that
   apply to the work.
5. Read the project-specific instructions between this header and the managed
   footer.

---

Put project-specific `AGENTS.md` content below this line and above the managed
FieldManual footer.
<!-- FIELDMANUAL_MANAGED_HEADER_END -->

# expletives Project Instructions

## Product Purpose

- `expletives` is the primary product: a reusable Go character-cell TUI
  toolkit with common controls, semantic frames, accessibility, deterministic
  tests, and automation-native architecture.
- `expletives-test` is a supported secondary product and real public-API
  consumer. It must be human runnable, exercise every public UI element and
  important state, and support attached drive/observe only after an explicit
  per-process `--automation <socket-path>` option.
- `expletivesctl` is the supported reusable-client command for attached
  automation.
- Treat [`docs/product-goals.md`](docs/product-goals.md) and the directed
  specifications indexed by
  [`docs/specifications/README.md`](docs/specifications/README.md) as the
  product sources of truth.
- Treat
  [`docs/specifications/ui-toolkit-requirements.md`](docs/specifications/ui-toolkit-requirements.md)
  as design input while it remains Draft. Do not silently resolve its
  conflicting framebuffer, package, inheritance, or automation models in
  code.

## Required Guidance

For substantive Go work, load the applicable documents under
`FieldManual/standards-and-practices/languages/go/`.

For TUI architecture, controls, rendering, or interaction, read:

- `FieldManual/knacks/UI/terminal/terminal-ui-design.knack/terminal-ui-design.overview.knack.md`;
- `FieldManual/knacks/UI/terminal/keyboard-navigation.knack.md`;
- `FieldManual/knacks/UI/terminal/box-drawing-and-shading.knack.md`; and
- `FieldManual/knacks/UI/terminal/turbovision-style-tuis.knack/turbovision-style-tuis.overview.knack.md`.

Before changing headless testing, semantic snapshots, attached automation, or
capture behavior, also read
`FieldManual/knacks/UI/terminal/terminal-ui-design.knack/terminal-ui-design.api.automation.knack.md`.

Before implementing a direct terminal boundary, also load the applicable
POSIX input, ANSI, VT100, xterm, curses, ncurses, and terminfo guidance. Keep
those layers distinct even when a selected Go backend abstracts them.

Use the maintained comparative research under `docs/research/` when selecting
control, event, inheritance/composition, focus, menu, hotkey, threading, or
event-loop patterns. Win32, wxWidgets, Motif, LessTif, curses, ncurses, Turbo
Vision, and other toolkits are design evidence, not compatibility targets.

## Architecture And Automation Boundaries

- Preserve the separation among domain state, interaction/controller state,
  pure intended-frame rendering, the terminal adapter, and the one terminal
  owner.
- Support multithreaded MVC, MVVC, and similar consuming-application
  structures through composition: application models remain
  toolkit-independent, controls and Layouts form the view, and structured
  events/commands can be handled by a controller, presenter, or update
  function. Do not require toolkit model inheritance, package-global
  application state, or reflective two-way binding.
- Toolkit-owned public behavior must be thread-safe. It is acceptable to
  serialize the event loop, renderer, and presenter onto one owner
  thread/goroutine each or a shared owner. Cross-owner access must use
  documented marshaling, ordering, cancellation, reentrancy, and shutdown
  behavior; arbitrary consumer model objects do not become thread-safe
  automatically.
- `app.Root()` is the sole root accessor. The special App-owned root `Panel`
  is the only parentless control; every ordinary control is created with a
  container-capable parent. Foundational parent ownership is immutable, and
  reparenting remains backlog work.
- Use Layout terminology. Instantiated `BoxLayout` and `GridLayout` objects
  are constructed independently, then attached atomically with
  `Panel.SetLayout`; `Panel.AddLayout` adds sibling top-level stacking
  contexts. A Layout tree may contain nested Layouts, but every Panel item
  remains a direct control child of the top-level owner and attachment never
  reparents it. Arrangement order and paint order are independent:
  Panel `Raise`/`Lower` operates among Panel peers in one Layout, and Layout
  `Raise`/`Lower` moves the complete subtree among Layout peers while
  preserving other-kind slots. Most controls should share Panel behavior
  directly or indirectly through the approved Go composition/embedding
  model. Follow
  [`docs/specifications/layout-api-v0.md`](docs/specifications/layout-api-v0.md)
  for the exact contract.
- Default the App-owned root to the complete offered container surface.
  Optional root minimum, maximum, and terminal-cell aspect ratio are one
  atomic root policy; do not reintroduce conventional terminal-size caps.
  Bound materialized frames by named aggregate allocation cost, and preserve
  1200 by 1200 as a tested reasonable geometry. Frame/GroupBox and Layout
  borders are independently configurable so an enclosing bordered Layout may
  contain adjacent unbordered Frames without doubled seams. Follow
  [`docs/specifications/root-sizing-and-borders-v0.md`](docs/specifications/root-sizing-and-borders-v0.md)
  for exact sizing, border, snapshot, and terminal-projection contracts.
- `Label`, `StaticText`, `Separator`, and `Rule` are non-container leaf
  controls. They retain shared Control geometry, style, Layout, stacking,
  lifetime, transaction, and snapshot behavior without acquiring container
  APIs. Follow
  [`docs/specifications/text-and-display-api-v0.md`](docs/specifications/text-and-display-api-v0.md)
  for their exact text, wrapping, alignment, mnemonic-target, divider, and
  one-cell Unicode contracts.
- `Button` and `HotkeyBar` are non-container Action leaves. Command
  definitions are the single source for label, enabled/disabled reason, and
  checked presentation state; Button, HotkeyBar, raw keys, bindings, direct
  commands, and the following Menu controls must converge on the same router.
  Preserve source-local pressed capture, deterministic focus traversal, and
  typed automation evidence as defined in
  [`docs/specifications/actions-api-v0.md`](docs/specifications/actions-api-v0.md).
- `MenuBar` is one persistent non-container leaf and popup-session owner per
  App; it is parented directly by `app.Root()` but occupies the complete
  physical top row as application chrome and may not be placed in a Layout.
  `Menu` and `MenuItem` are copied immutable models rather than Controls or
  nested event loops. Menu commands derive presentation and enabled state
  from the shared command registry, restore prior Button focus exactly when
  possible, and expose a bounded flat semantic tree. Preserve Turbo Vision
  appearance and keyboard behavior (not implementation), including red
  mnemonic letters, green selection, measured/backset child popups,
  start/end top-level groups with conventional right-justified Help, Alt
  top-level mnemonics, F9 bar activation, Ctrl-Space,
  arrow/Home/End/Enter/Escape traversal, nested popup clipping, and
  command-route parity as defined in
  [`docs/specifications/menus-api-v0.md`](docs/specifications/menus-api-v0.md).
- `StatusBar` is one persistent non-container, non-focusable leaf per App. It
  is parented directly by `app.Root()`, occupies the complete physical bottom
  row independently of root constraints, and may not be a Layout item.
  Copied keyed segments are either static context or command-derived hints;
  higher priorities survive narrow widths, while declaration order remains
  the paint order. Preserve shared command-state ownership, atomic segment
  replacement, Turbo Vision palette roles, and typed rendered/omitted/clipped
  evidence as defined in
  [`docs/specifications/status-bar-api-v0.md`](docs/specifications/status-bar-api-v0.md).
  In `expletives-test`, Sections/Status Bar is an independent checked
  visibility toggle, not a catalog-screen selector.
- `Header` and `Footer` are root-owned one-row Containers with derived,
  full-physical-width geometry outside root constraints. Headers retain
  construction order below the Main Menu; Footers retain construction order
  upward from the Status Bar, so the newest is highest. They accept only
  Layout trees whose complete measured minimum height is at most one. Hidden
  or tiny-surface-unallocated bands have empty Bounds and no actionable
  Layout-overflow episode. Preserve their exact ownership, ordering, geometry,
  validation, rendering, snapshot, and automation contract from
  [`docs/specifications/headers-footers-api-v0.md`](docs/specifications/headers-footers-api-v0.md).
  The test catalog exposes a top-level `&Sections` menu with Status Bar plus
  separate Sections/Headers and Sections/Footers
  submenus. Headers owns a checked Show toggle plus Add, Remove Highest, and
  Remove Lowest. Footers owns independent checked visibility toggles for the
  fixed Global Hotkeys, Screen Hotkeys, and Focus Guidance roles.
- Do not put hotkey inventory in the StatusBar. Footer guidance is ordered
  physically from lowest to highest as global application hotkeys,
  current-screen hotkeys, then focused-control-type hotkeys/advisories. Allow
  applications to append to or override the generic focused-control guidance
  for one instance. All three guidance layers are left-justified. Preserve
  the dynamic `FocusGuideBar` and typed evidence contract in
  [`docs/specifications/focus-guide-bar-api-v0.md`](docs/specifications/focus-guide-bar-api-v0.md).
- Follow
  [`docs/Terminal-Shortcut-Compatibility.md`](docs/Terminal-Shortcut-Compatibility.md)
  for toolkit, demo, example, and acceptance-test defaults. Host terminal
  bindings consume input before the application can recover it: do not make a
  reserved or conditionally intercepted chord the preferred or only route to
  essential behavior. Preserve client-configurable raw chord support.
- Application chrome is anchored to physical terminal edges in this order:
  Main Menu, Headers, root content, Footers, Status Bar. Main Menu and Status
  Bar are full-width edge rows; every Header/Footer is exactly one row and
  accepts only a Layout tree compatible with that height. The complete
  initial chrome sequence is delivered; follow
  [`docs/specifications/application-chrome-v0.md`](docs/specifications/application-chrome-v0.md).
- Use “Application Client Area” for the complete physical-width rectangle
  below the lowest visible Header (or Main Menu, or row 0) and above the
  highest visible Footer (or Status Bar, or the physical bottom edge). Use
  “Panel Client Area” for a Panel's rectangle less its border and any visible
  horizontal or vertical scrollbars. Root size constraints may intersect the
  Application Client Area for root content, but do not redefine it.
- Follow
  [`docs/specifications/layouts-and-overflow.md`](docs/specifications/layouts-and-overflow.md)
  for logical versus clipped geometry, attachment, overflow episodes,
  notification, and fallback behavior.
- When Layout space is below the combined minimum, preserve logical minima,
  retain the resulting logical rectangles, constrain effective output by the
  intersection of every ancestor clip and the application surface, and publish
  structured overflow state. Queue an application-registered Overflow callback
  only after the Layout pass and outside internal locks on a bounded callback
  dispatcher with documented cancellation/deadline behavior. Never create an
  unbounded replacement goroutine for a stuck handler. Make exactly one
  handler-delivery attempt, or one default-notification attempt when no handler
  applies, per continuous Panel/Layout overflow episode. An absent, declined,
  failed, or timed-out handler uses a deterministic, observable, non-recursive
  fallback that never blocks headless or attached automation.
- Follow [`docs/Limited-Unicode-Support.md`](docs/Limited-Unicode-Support.md).
  Canonical displayed text elements occupy exactly one terminal cell.
  One-cell composed grapheme clusters are allowed. The pinned project width
  policy, rather than a raw Unicode East Asian Width property alone, decides
  the cell count: a cluster measured as exactly one cell is supported; every
  multi-cell, zero-width, indeterminate, or otherwise unsupported cluster
  renders as one `U+FFFD REPLACEMENT CHARACTER` (`�`). Do not add width-two
  or continuation cells. Basic terminal adapters render definite 7-bit ASCII
  and confidently known code-page mappings directly; otherwise they use a
  deterministic single-character ASCII approximation, or `?`, in black on
  yellow. Unknown and remote code-page mappings must not be guessed.
- Human input, headless tests, and attached automation must use the same
  semantic controller path. Automation workers never call the terminal
  backend directly.
- Automation may inject raw key lifecycle events `KeyDown`, `KeyUp`, and
  `KeyPress` before mnemonic, accelerator, hotkey, and command resolution.
  This must support held modifiers and multi-key chords without accepting an
  arbitrary terminal escape byte stream. Direct semantic commands remain a
  separate supported path.
- `KeyPress` is a distinct one-shot raw logical event and leaves no key held.
  Stateful chords use ordered modifier `KeyDown`, non-modifier `KeyPress`,
  and modifier `KeyUp` events.
- Snapshots must be immutable and atomically pair the intended frame with a
  bounded, typed semantic view. Basic `SnapshotV1` cells contain the canonical
  one-cell grapheme, semantic style, resolved foreground/background colors,
  and stable owner identity; cursor and the bounded typed control tree are
  snapshot-level facts. Do not add width or continuation fields or expose
  unrestricted domain maps merely for test convenience.
- When extending the typed detail union, follow
  [`docs/Control-Details-Extension-Checklist.md`](docs/Control-Details-Extension-Checklist.md)
  so core cloning, automation projection/cloning, explicit kind validation,
  resource proofs, and tests advance together.
- Every injected automation event or command needs a unique request ID, an
  explicit outcome, and its associated frame sequence. No-ops complete
  explicitly; unrelated redraws do not count; timeouts never mean success;
  exit-producing events publish a final snapshot.
- A normal `expletives-test` run creates no automation endpoint or artifact.
  The initial `--automation <socket-path>` mode may be unauthenticated because
  the operator is choosing a trusted, non-risky context. Do not claim it is
  safe for hostile, multi-user, or elevated use. Authentication and
  capability authorization require a later approved proposal. When attached
  automation is active, show `UNAUTHENTICATED AUTOMATION ENABLED` as a
  default-visible StatusBar segment, never as a dedicated content Panel.
  Provide a checked File-menu command that hides and restores that segment
  without disabling the endpoint, and test both transitions.
- Keep automation queues and protocol inputs bounded, define fairness with
  human input, and clean up only endpoint resources owned by the current
  process.
- Keep focus, selection, activation, back, cancel, interrupt, and quit
  distinct. Ctrl-C handling must be robust, complete, configurable, and
  tested across views, editors, modals, long-running work, and automation.
- Preserve the implemented bounded Selection contract in
  [`docs/specifications/selection-api-v0.md`](docs/specifications/selection-api-v0.md):
  group-owned radio exclusivity, stable values, disabled-option skipping,
  wrap/clamp policies, direct-parent focus groups, Tab/Shift-Tab group
  traversal, focus-only spatial arrows, Radio Space/Enter selection,
  Cycle/Select `[`/`]` changes, serialized user changes, programmatic-setter
  silence, outside-lock ChangeCommand routing, and exact typed
  snapshot/automation evidence.
- Preserve the directed Text and Numeric Input contract in
  [`docs/specifications/text-and-numeric-input-api-v0.md`](docs/specifications/text-and-numeric-input-api-v0.md).
  Optional text validators require soft or hard enforcement, whitelist or
  blacklist matching, and a nonempty matching-character set. Soft mode
  accepts input with green valid text or yellow valid/red invalid text when
  invalid. Hard mode ignores disallowed typed input. Password defaults false,
  paints `*`, still validates the real value, and never exposes that value in
  frames, snapshots, automation, diagnostics, or logs. NumberField and
  SpinBox use finite fixed-place `float64` values, reject caller values that
  require rounding, retain invalid intermediate edits, refuse invalid
  Enter/Tab commits, and use `[`/`]` for clamped SpinBox stepping outside edit
  mode.
- Represent menu accelerators, mnemonics, hotkeys, and bindings as structured
  data. Provide fallback paths when Alt or modified keys are unavailable.
- Preserve the completed Actions, Menus, Status Bar, Headers/Footers, and
  Selection sequence. Use the persistent
  `expletives-test` MenuBar to navigate purpose-specific catalog screens as
  the public control set grows, rather than crowding every demonstration onto
  one surface.

## Testing And Builds

- Add normal Go unit and integration tests wherever practical. Specialized
  headless, `expletives-test`, attached-automation, PTY, race, fuzz, profiling,
  and real-terminal tests supplement rather than replace them.
- Every public control or meaningful state added to the library requires
  corresponding automated coverage and an `expletives-test` catalog scenario.
- Test one-cell composed text and `U+FFFD` replacement of every unsupported
  display element without cursor, clipping, border, Panel, or Layout
  displacement. Test known and unknown basic-terminal code-page profiles and
  highlighted ASCII degradation separately from the canonical frame.
- The root Makefile and artifact contract are defined in
  [`docs/specifications/build-and-verification.md`](docs/specifications/build-and-verification.md).
  Do not claim those commands work until they have been implemented and run.
- Every current or future project, test, automation, and diagnostic executable
  must build in debug, release, and profiling modes at
  `build/<mode>/<name>`. This includes `expletives-test` and
  `expletivesctl`.
- Keep generated `build/` artifacts out of version control.

## Panel Of Experts

- The standing nine-lens
  [`Panel Of Experts Charter`](project-management/reviews/panel-of-experts-charter.md)
  is an exceptional mechanism, not the default implementation or bug-fix
  workflow. Do not convene it for routine features, localized fixes,
  refactoring, tests, documentation, diagnostics, or mechanical resolution of
  an already-reviewed finding.
- Use normal verification and focused peer review by default. Asking a
  specialist or sub-agent for a bounded review does not constitute a Panel.
- Convene a formal Panel only on operator request or for a genuinely major,
  cross-cutting, hard-to-reverse decision that ordinary design work cannot
  resolve. Select a proportional odd subset of three, five, or nine relevant
  seats; all nine are reserved for foundational or product-wide architecture.
- Use one lean brief, bounded findings, targeted follow-up, and one closure
  record. Separate design and delivery Panels, per-finding files, independent
  chairs, and SHA-256 manifests are not automatic requirements.
- Review headcounts do not approve work or accept risk. Confirmed blockers and
  majors still require a verified fix or an explicit operator decision.

## Work Tracking And Ubersight

- Record direct work, decisions, open questions, validation, and deferrals in
  `project-management/` using one primary task state at a time.
- Keep one durable owner for each fact: task status in the active or completed
  task record, behavior in specifications, durable choices in the decision
  log, and defects in the bug tracker. Link to that fact instead of copying
  and synchronizing narrative status across the README, roadmap, backlog,
  review records, and proposals.
- Update durable project-management state at meaningful transitions, when a
  blocker or decision changes, and at closure. Routine edit/test iterations
  need no administrative entry.
- For extended work, publish concise Ubersight transitions with the installed
  writer and the guidance in
  `FieldManual/knacks/software-engineering/development-observability/ubersight.knack.md`.
- Publish Ubersight when a phase, blocker, or verification state materially
  changes. Do not treat it as a per-command activity feed.
- Prefix Ubersight phase IDs/titles with their roadmap phase number and
  current-phase slice IDs/titles with both phase and slice numbers so an
  operator can correlate numeric references at a glance.
- Use an explicit collision-resistant project context; `expletives-main` is
  the context for this checkout unless local policy specifies another.
- Ubersight is a projection, not durable history. Do not publish sensitive
  content, and never stage `.local/ubersight/`.

<!-- FIELDMANUAL_MANAGED_FOOTER_START -->
---

## Field Manual Managed Guidance

- Treat the configured project root as the boundary for project-owned files,
  local state, and project-management records.
- Treat the configured framework root as read-only guidance when it is a
  submodule. Do not write project state into the FieldManual subtree.
- Use `project-management/` for the backlog, active and completed work, bugs,
  proposals, reviews, decisions, open questions, and bounded human requests.
- Keep durable behavior and interface contracts in `docs/specifications/`.
- Keep checkout-local state and policy inputs under `.local/`, and keep the
  entire `.local/` tree ignored by version control.
- Put disposable test workspaces and temporary project artifacts in uniquely
  named children of the project root's `.local/tmp/` by default. Clean up
  only paths created or explicitly acquired by the current operation.
- Put cross-project change requests under `ECRs/<target-project>/`. Use the
  ready `ECRs/FieldManual/` tree for FieldManual requests and copy
  `ECRs/_target-template/` for another target.
- FieldManual supplies prose standards, not language-specific verification
  programs. Use the consuming project's declared build, test, lint, security,
  and release commands.
- Project-specific instructions may specialize FieldManual defaults. Record
  intentional exceptions explicitly instead of allowing silent drift.
<!-- FIELDMANUAL_MANAGED_FOOTER_END -->
