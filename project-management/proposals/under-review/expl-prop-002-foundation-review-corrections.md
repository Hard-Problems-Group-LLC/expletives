# Foundation Review Corrections

- ID: EXPL-PROP-002
- Author: Codex collaboration
- Sponsor: `/root`
- Date: 2026-07-30
- Status: Under Review; implemented and verified
- Review authority: `EXPL-REV-001`
- Affected products: `expletives`, `expletives-test`, and `expletivesctl`
- Governing evidence:
  [`EXPL-REV-001 Design Evidence`](../../reviews/decisions/EXPL-REV-001-design-evidence.md)

## Purpose

This proposal gives one coherent correction design for the 21 canonical
confirmed findings from `EXPL-REV-001` Round 1. The implementation and
verification are complete. It remains a sponsor mitigation, not operator
approval or risk acceptance.

The correction is deliberately organized as seven workstreams. A shared
implementation and one versioned review addendum will replace 21 independent
fix-and-review cycles.

## Compatibility Boundary

The current API and automation protocol are pre-v1 foundation contracts.
There are no released compatibility consumers in the packet. The correction
may make source and wire changes now, but the resulting contracts must be
fully specified, tested through an external-package consumer, and reviewed
before Basic Layouts or more common controls are added.

## Workstream 1: Canonical Controls And Extension Boundary

### Control identity and capabilities

Every public control becomes a small copy-safe handle over one canonical
toolkit-owned node. Copying a handle aliases the same node; it cannot create a
detached mutable control with the same identity.

The public hierarchy uses two shallow sealed capabilities:

```go
type Control interface {
    ID() ControlID
    AutomationKey() string
    Bounds() Rect
    MinimumSize() Size
    Visible() bool
    Style() StyleID

    controlNode() *controlNode
}

type Container interface {
    Control
    Children() []Control

    containerNode() *controlNode
}
```

The unexported methods prevent unrelated external implementations from
forging toolkit identity. An external compound control may embed or wrap an
actual toolkit control. `Panel`, `Frame`, and `GroupBox` implement
`Container`; future leaf controls implement only `Control`. Constructors
accept `Container`, not `*Panel`.

The root remains the sole parentless node and `App.Root()` continues to return
its special `*Panel`.

### Internal behavior composition

The canonical node owns universal identity, parentage, geometry, visibility,
style reference, lifecycle state, and children. It delegates control-specific
client rectangle, paint, and typed snapshot detail to one internal behavior
descriptor. The renderer traverses canonical nodes and invokes descriptors;
it does not switch on a growing tagged `Panel` union.

`Frame` and `GroupBox` keep their public wrappers, but their title, border, and
client-inset behavior lives with their descriptors.

This slice does not expose arbitrary application callbacks during measure or
paint. External custom painting remains unsupported in the corrected
pre-v1 API. The compatibility boundary for a later custom-control factory is
the canonical node plus behavior descriptor, typed detail, and `Container`
parent capability; adding that factory must not replace the tree or parent
contracts. External compound controls built from public controls are
supported now.

### Lifetime

Non-root controls gain logical recursive destruction. A destroyed handle
returns `ErrDestroyed`; it cannot be used as a parent or mutated. Destruction
removes active IDs and automation keys, publishes one atomic snapshot, and
reduces the concurrent-control count. `MaxControls` becomes a concurrent-tree
limit rather than an App-lifetime limit. The root is destroyed only by App
shutdown.

### Core snapshot and protocol projection

The root package publishes a toolkit `Snapshot`, not an automation wire DTO.
It contains the intended frame, cursor, input-source state, controls,
overflow state, and optional local completion association.

`ControlSnapshot` gains a bounded versioned typed-detail union. The foundation
members are container and border/title detail. Later controls add typed
members without string-keyed property bags.

The automation package owns its `SnapshotV1` projection and explicit
conversion from the core snapshot. The terminal presenter consumes only the
intended frame and cursor. This separates core observation, wire schema, and
physical degradation while retaining the operator-directed atomic snapshot.

### Semantic styles and themes

Controls store a semantic `StyleID`, not independently resolved RGB values.
An App-owned immutable `Theme` maps IDs to `ResolvedStyle` values. A resolved
style includes foreground, background, and bounded terminal-independent
attributes. The App supplies documented defaults.

Construction rejects missing semantic styles. Replacing the Theme is one
atomic transaction and one publication. Snapshots retain both the semantic ID
and the resolved cell colors/attributes required for exact observation.
Terminal capability degradation is a presenter projection and never mutates
the intended frame.

## Workstream 2: Atomic Updates, Dispatch, And Shutdown

### Transactions

An App creates transaction builders outside its state lock. A transaction can
create controls, mutate geometry/visibility/style/minima, destroy controls,
and replace the theme. Provisional controls may parent later provisional
controls in the same transaction.

Commit:

- accepts a context;
- validates the complete operation set and all current preconditions before
  mutation;
- invokes no application callback while holding toolkit state;
- applies all operations or none;
- renders and publishes exactly once; and
- consumes the transaction so it cannot be replayed.

Existing constructors and setters remain convenience single-operation
transactions. The fixture uses one construction transaction and one
transaction per logical resize or command transition.

### Synchronous public calls

Contextless getters and convenience setters are thread-safe synchronous
calls. They may wait only for bounded toolkit-owned critical sections and
never call application code. Context-aware transaction commit, snapshot wait,
input dispatch, and command dispatch observe cancellation at their documented
queue boundaries.

### Dispatch gate and callbacks

The non-cancellable dispatch mutex and hidden cross-App context-value guard
are removed. Each App uses a bounded cancellation-aware dispatch gate.
Contention returns a typed busy/capacity result or observes caller
cancellation; it cannot wait forever.

Application command handlers execute outside state and presentation owners on
a bounded callback executor. The toolkit waits only to a documented deadline.
A timed-out or cancellation-ignoring callback may consume one bounded
executor slot, but it cannot retain the dispatch gate, prevent terminal
restoration, or make toolkit shutdown wait forever. App operations reject
after shutdown.

The handler returns a structured `CommandResult` containing outcome, bounded
public code/message, and an optional local-only cause. Cancellation and
deadline outcomes have specified mappings. Raw local error strings are never
sent to automation unless the application explicitly supplies the public
message.

## Workstream 3: Automation Ownership And Evolution

### Host-owned identity and command inventory

The reusable server requires a nonempty host application identity. It derives
the current command inventory from the App command registry. Fixture command
IDs and defaults move into `internal/demo` or `cmd/expletives-test`.

### One request/result authority

The App no longer permanently retains external request IDs. It treats a
request ID as a correlation label. The automation session is the sole
duplicate-detection and reconciliation authority.

Accepted IDs are unique while active or retained. One bounded FIFO record
owns compact completion metadata and the exact immutable associated projected
snapshot. Metadata and snapshot expire atomically. Querying a retained result
returns both. Expiry permits later ID reuse, but clients must never
automatically replay an indeterminate request.

The default retained-result count is selected from a tested aggregate memory
budget. Server limits advertise that one value rather than incompatible App,
completion, and snapshot counts.

### Command targets

Automation command targets are explicitly stable automation keys. The bridge
resolves the key through a read-only App lookup, rejects missing or destroyed
targets, and passes the canonical `ControlID` to application policy.

### Version transition

The v1 hello carries a supported-version list. Hello decoding permits bounded
unknown additive fields; operational records remain strict. A future server
can continue sending the v1-compatible preface, advertise multiple versions,
and accept an explicit version-selection record before operational requests.
Current servers advertise only v1. Dropping v1 requires a new endpoint policy
or an explicit compatibility decision.

### Minimal v1 surface and client timeouts

The unusable v1 cancel operation and automation-queue advertisement are
removed until an out-of-band cancellation path exists. The client separates
dial and operation deadlines. Its default operation budget is at least the
server execution budget plus response margin, and indeterminate diagnostics
print the request ID and exact recovery command.

## Workstream 4: Terminal Input And Process Verification

A reusable public terminal adapter owns a bounded incremental decoder. It:

- retains partial UTF-8 and escape-sequence prefixes across reads;
- handles coalesced records;
- uses an explicit short Escape-disambiguation deadline;
- emits only logical `KeyDown`, `KeyUp`, and `KeyPress` events;
- handles unknown sequences deterministically without interpreting arbitrary
  bytes as commands; and
- exposes reset/flush behavior for disconnect and shutdown.

`expletives-test` forwards decoded logical events and contains no escape-byte
grammar.

A Linux PTY process test launches the actual debug `expletives-test` binary
and covers fragmented input, resize, Ctrl-C and termination signals, final
automation state where applicable, exit status, and exact terminal
restoration. The Makefile builds the required binary before that test.

## Workstream 5: Commands And Bindings

The App owns one inspectable command registry. A command definition supplies:

- stable ID and bounded help text;
- enabled state;
- public automation visibility; and
- routing to the application command handler.

Bindings refer to registry entries. The Basic registry supports inspect,
replace, and unbind operations and one documented application scope. Later
focus, modal, menu, mnemonic, and editing scopes extend the registry instead
of creating parallel tables. Automation hello and future visible help project
from this same authority.

Ctrl-C remains a distinct future policy decision, but the registry reserves
interrupt, cancel, back, and quit as separate semantic commands rather than
hard-coding one signal meaning.

## Workstream 6: Go Documentation And Examples

The root package gains package documentation and external-package examples.
Every exported option and method documents:

- zero/default behavior;
- ownership and lifetime;
- copying;
- synchronous, blocking, and cancellation behavior;
- callback context and failure behavior;
- publication and transaction semantics; and
- inspectable error values.

Examples cover construction, atomic MVC-style updates, command routing,
snapshots, automation activation, and orderly shutdown.

## Workstream 7: Bounded Text And Response Proof

The corrected contract defines:

- maximum UTF-8 bytes per displayed one-cell grapheme;
- maximum title bytes and title cells;
- maximum controls and frame geometry;
- maximum projected snapshot bytes; and
- maximum response bytes and retained aggregate bytes.

Titles are validated and normalized outside the App lock, then cached as
bounded cells. A response encoder stops at its configured limit rather than
first allocating an arbitrarily oversized JSON result.

Tests construct synthetic legal maxima and prove that every maximum projected
snapshot fits the advertised response bound. One-beyond-limit tests reject
before mutating the App.

## Finding Map

| Workstream | Canonical findings |
| --- | --- |
| Controls, extension, lifetime, snapshot, theme | S01-F003, S02-F008, S03-F001, S03-F005, S04-F001 |
| Atomic updates, owner, dispatch, shutdown | S02-F003, S02-F005, S03-F004, S09-F001 |
| Automation ownership and evolution | S01-F002, S02-F004, S02-F009, S03-F002, S05-F003, S05-F004 |
| Terminal input and PTY | S01-F001, S06-F003 |
| Commands and diagnostics | S04-F003, S05-F001 |
| Go documentation | S05-F005 |
| Bounded text/response proof | S08-F001 |

## Verification

The shared correction must pass:

- formatting, vet, ordinary, integration, and race tests;
- focused transaction atomicity and observer-race tests;
- copied-handle, destroyed-handle, and concurrent-capacity tests;
- callback timeout, saturation, panic, dispatch contention, and shutdown
  tests;
- retention churn and indeterminate reconciliation tests;
- decoder split/coalescing/fuzz tests and the actual-process PTY test;
- legal-maximum snapshot and one-beyond-limit tests;
- all debug, release, and profiling builds and smoke checks;
- cross-compilation checks already required by the foundation; and
- a fresh attached closed-loop exercise with `expletivesctl`.

## Review And Completion

The seven workstreams, formal specifications, and verification evidence are
complete. One concise closure addendum records their resolution in
[`EXPL-REV-001 Design Evidence`](../../reviews/decisions/EXPL-REV-001-design-evidence.md).
Under `EXPL-DEC-010`, this mechanical correction cycle does not require a
replacement manifest, reviewer reconvening, or separate delivery Panel.
