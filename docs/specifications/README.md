# Specifications

Keep durable, project-specific behavior and interface contracts here.

Specifications should identify:

- the problem and scope;
- required behavior and explicit non-goals;
- interfaces, data, invariants, and failure behavior;
- compatibility and migration constraints;
- security and operational considerations; and
- acceptance criteria and validation evidence.

Link specifications to the proposal, backlog item, bug, or direct operator
request that authorized them. Update specifications when behavior changes;
do not use them as retrospective decoration.

## Current Documents

- [`../Limited-Unicode-Support.md`](../Limited-Unicode-Support.md) defines the
  governing one-cell Unicode, `U+FFFD` replacement, and conservative
  basic-terminal degradation boundary for every specification below.
- [`../Terminal-Shortcut-Compatibility.md`](../Terminal-Shortcut-Compatibility.md)
  defines the governing host-emulator collision policy for project-selected
  keyboard defaults while preserving client-selected raw chord support.
- [`application-architecture.md`](application-architecture.md) requires the
  toolkit to support multithreaded MVC, MVVC, and similar
  consuming-application structures without forcing toolkit model inheritance
  or one application framework.
- [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md)
  defines safe concurrent public behavior, serialized owner boundaries,
  marshaling, immutable publication, callback/reentrancy, and shutdown rules.
- [`control-catalog.md`](control-catalog.md) defines the directed common-control
  inventory, phase order, parent/root invariant, Basic Presentation,
  Basic Automation, Basic Layouts, and deferred Structured Input family.
- [`layouts-and-overflow.md`](layouts-and-overflow.md) defines atomic
  attachment, logical minima and effective clipping, structured overflow, the
  application callback, coalescing, and deterministic fallback notification.
- [`layout-api-v0.md`](layout-api-v0.md) fixes the implemented Box/Grid,
  nesting, snapshot, transaction, Panel/Layout stacking, and exact Overflow
  Go contracts.
- [`root-sizing-and-borders-v0.md`](root-sizing-and-borders-v0.md) defines
  allocation-based frame sizing, optional root constraints, independent
  Frame/Layout borders, and capability-aware physical glyph projection.
- [`application-chrome-v0.md`](application-chrome-v0.md) fixes physical edge
  ownership and row order for the Main Menu, Headers, Footers, Status Bar,
  and the remaining root-content rectangle.
- [`text-and-display-api-v0.md`](text-and-display-api-v0.md) defines the
  first non-container leaf controls: Label, StaticText, Separator, and Rule.
- [`actions-api-v0.md`](actions-api-v0.md) defines Button, HotkeyBar,
  command-owned presentation state, focus traversal, mnemonics, and unified
  activation used by the following Menu phase.
- [`menus-api-v0.md`](menus-api-v0.md) defines immutable popup Menu models,
  MenuItem descriptors, the persistent MenuBar control, popup keyboard
  sessions, focus restoration, typed automation, and catalog-screen
  navigation.
- [`status-bar-api-v0.md`](status-bar-api-v0.md) defines the unique root-owned
  bottom-row StatusBar, copied contextual and command segments, deterministic
  narrow-width priority, and typed automation evidence.
- [`headers-footers-api-v0.md`](headers-footers-api-v0.md) defines ordered
  root-owned one-row Header/Footer containers and atomic Layout compatibility.
- [`focus-guide-bar-api-v0.md`](focus-guide-bar-api-v0.md) defines dynamic
  focused-control guidance and per-instance append/override customization.
- [`selection-api-v0.md`](selection-api-v0.md) defines Checkbox,
  RadioButton/RadioGroup, and fixed-option CycleField/SelectField behavior.
- [`text-and-numeric-input-api-v0.md`](text-and-numeric-input-api-v0.md)
  defines the Phase 12 editing, validator, password, numeric, and multiline
  input contract.
- [`progress-api-v0.md`](progress-api-v0.md) defines deterministic
  ProgressBar, Meter, Spinner, and ActivityDots state, rendering,
  worker-update, and typed evidence.
- [`navigation-chrome-api-v0.md`](navigation-chrome-api-v0.md) defines honest
  ScrollBar viewport state and TabbedPanel/Notebook page navigation.
- [`scrolling-content-api-v0.md`](scrolling-content-api-v0.md) defines
  managed scrolling containers, Markdown rendering, bounded log/stream
  retention, follow/scrollback, and honest drop accounting.
- [`collections-api-v0.md`](collections-api-v0.md) defines copied bounded
  collection models, stable current/selection identity, popup selection,
  tree expansion, table sorting, DataGrid editing, and compact automation.
- [`modals-api-v0.md`](modals-api-v0.md) defines the Phase 17 modal stack,
  one-shot result lifecycle, focus and command capture, resize/shadow behavior,
  standard dialogs, progress cancellation, and automation evidence.
- [`file-pickers-api-v0.md`](file-pickers-api-v0.md) defines the Phase 18
  bounded provider boundary, typed single/multiple/directory results,
  filtering, sorting, Turbo Vision interaction, concurrency, automation, and
  opt-in local-filesystem adapter.
- [`terminal-compatibility-v0.md`](terminal-compatibility-v0.md) defines the
  Phase 19 physical profile, locale, capability, job-control, unsupported-
  claim, and verification-matrix boundaries.
- [`expletives-test.md`](expletives-test.md) contains directed requirements
  for the secondary interactive product, its human and attached-automation
  modes, frame inspection, event correlation, keyboard/interrupt behavior,
  and closed-loop diagnosis.
- [`build-and-verification.md`](build-and-verification.md) contains directed
  requirements for ordinary Go tests, layered terminal verification, the
  root Makefile, build modes, and artifact paths.
- [`implementation-baseline-v0.md`](implementation-baseline-v0.md) fixes the
  first runnable Go, Unicode, Linux terminal, build-mode, JSON Lines
  automation, semantic-theme, transaction, command, and validation choices
  for Core through Basic Automation.
- [`go-api-v0.md`](go-api-v0.md) defines the implemented public Go contracts
  for App ownership, Control/Container capabilities, copy-safe controls,
  Themes, Transactions, rendering, local snapshots, raw input, commands,
  completion correlation, and final state.
- [`automation-protocol-v1.md`](automation-protocol-v1.md) defines the
  implemented bounded JSON Lines protocol, Unix-socket lifecycle, operations,
  automation-owned snapshot projection, retained results, client behavior,
  and explicit unauthenticated trust boundary.
- [`ui-toolkit-requirements.md`](ui-toolkit-requirements.md) is an older Draft
  design target. It remains useful input, but its conflicting framebuffer,
  package, object, and automation models are not settled contracts.

The directed documents govern their scopes. Unresolved implementation choices
must be decided through project management and then reflected here before
code silently selects one.
