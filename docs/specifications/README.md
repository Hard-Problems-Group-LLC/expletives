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
- [`text-and-display-api-v0.md`](text-and-display-api-v0.md) defines the
  first non-container leaf controls: Label, StaticText, Separator, and Rule.
- [`actions-api-v0.md`](actions-api-v0.md) defines Button, HotkeyBar,
  command-owned presentation state, focus traversal, mnemonics, and unified
  activation used by the following Menu phase.
- [`menus-api-v0.md`](menus-api-v0.md) defines immutable popup Menu models,
  MenuItem descriptors, the persistent MenuBar control, popup keyboard
  sessions, focus restoration, typed automation, and catalog-screen
  navigation.
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
