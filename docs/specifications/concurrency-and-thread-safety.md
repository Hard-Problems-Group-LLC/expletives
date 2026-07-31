# Concurrency And Thread Safety

Status: Directed requirement; exact public API names remain under design
Authority: Direct operator request on 2026-07-24
Related decisions: `EXPL-DEC-004` and `EXPL-DEC-007` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)
Related specification:
[`application-architecture.md`](application-architecture.md)
Layout and overflow contract:
[`layouts-and-overflow.md`](layouts-and-overflow.md)
Related text scope:
[`Limited Unicode Support`](../Limited-Unicode-Support.md)

## Purpose

`expletives` must be safe for applications whose models, controllers,
presenters, view models, services, and workers run concurrently. This
includes multithreaded or multi-goroutine MVC, MVVC, MVVM, MVP, MVU, and
related application structures.

Thread safety is a public behavior contract, not a promise that UI work runs
in parallel. The toolkit may serialize controller transitions, layout,
rendering, and terminal presentation. It may use one logical owner for all of
them or separate owners connected by bounded handoffs.

## Meaning Of Safe Concurrent Use

Every exported type and operation must document its concurrency category,
blocking behavior, cancellation behavior, ownership, and lifetime:

- immutable values are safe to share freely;
- concurrent read operations observe a complete published state;
- mutation operations either marshal to the appropriate owner, synchronize
  internally, or reject safely with a documented error;
- lifecycle operations are safe when they race with work submission and
  observation; and
- no supported public call pattern may cause a data race, corrupt internal
  state, write concurrently to the terminal, send on a closed channel, or
  deadlock merely because calls came from different goroutines.

This requirement does not make unsynchronized aliases safe. When the toolkit
retains or returns slices, maps, pointers, callbacks, models, or option data,
it must copy them, return an immutable view, transfer ownership explicitly,
or document the caller's non-mutation obligation. A mutable value documented
as non-copyable must remain so after first use.

## Logical Owners

The implementation may define these serialized execution contexts:

1. **UI owner** — applies input, commands, focus changes, control-tree
   mutations, invalidation, and lifecycle transitions.
2. **Render owner** — performs layout and produces an intended frame and
   bounded typed control-tree semantic view from an accepted UI state.
3. **Presentation owner** — is the only context that calls the active
   terminal backend.

One goroutine may own all three responsibilities. Separate goroutines may be
used when their handoffs are bounded, ordered, cancellation-aware, and carry
immutable data or explicit ownership transfers. A separate renderer must not
read a control tree while the UI owner mutates it.

An owner is a logical serialization rule, not automatically an operating
system thread. OS-thread affinity applies only where required by a backend or
platform boundary.

## Safe Public API Boundaries

The supported public surface must provide safe ways to:

- post an asynchronous UI operation or typed event;
- synchronously obtain the result of an owner-serialized operation when that
  behavior is appropriate;
- atomically attach an independently constructed Layout through
  `Panel.SetLayout`;
- submit immutable background results;
- observe the latest or a retained immutable snapshot;
- request invalidation or rendering without painting directly;
- cancel request-scoped waits or work;
- initiate shutdown; and
- wait for application termination and its terminal error.

Names such as `Post`, `Call`, `Invoke`, or `Dispatch` are conceptual here,
not approved API names.

Direct control-tree mutation may be convenient during construction. Its
running-state behavior must still be safe and explicit: it may marshal,
return a documented lifecycle or wrong-context error, or be exposed only
through an owner-scoped operation. It must not silently race. Toolkit-private
owner methods should remain unexported rather than making every consumer
reason about internal locks.

Getters must not return mutable internal storage. A concurrently observed
property is either a value from one published state or is documented as a
separate atomic diagnostic; it must not be a torn mixture.

## UI-Owner Marshaling

Owner marshaling must have explicit ordering and completion semantics.

- Posted work enters a bounded queue and reports acceptance, rejection, or
  backpressure.
- Acceptance into the queue is not completion.
- A synchronous call waits on a request-specific completion and observes its
  caller context.
- Cancellation before execution prevents the operation when that can be
  guaranteed.
- Cancellation or timeout after execution may have begun reports an honest
  completed, failed, cancelled, or indeterminate outcome; it never invents
  success and never silently retries the operation.
- Work rejected because shutdown has begun returns a stable, inspectable
  lifecycle error.

A synchronous owner call made from an owner callback must execute safely
inline when its contract permits that, or fail explicitly with an error such
as "would deadlock." It must never enqueue behind itself and wait. The design
must not depend on an unsupported assumption that Go exposes a general
goroutine identity; an owner-scoped context, dispatcher, transaction, or
equivalent explicit mechanism may carry this authority.

Human input, attached automation, direct commands, timers, resize,
background results, and application posts all converge at documented owner
queues. Their ordering, coalescing, priority, and fairness are product
behavior rather than scheduler accidents.

## Immutable Publication And Snapshots

Published snapshots must be immutable and safe for concurrent readers. One
publication atomically pairs:

- the root-package `Snapshot` intended frame;
- its snapshot-level cursor and bounded typed control tree;
- the frame sequence and final-state marker; and
- any command completion associated with that state.

Each root-package `Snapshot` cell contains only:

- its canonical one-cell grapheme;
- semantic style;
- resolved foreground and background; and
- stable owner identity.

Width and continuation-cell fields are neither needed nor permitted in
`Snapshot`. Cursor and the typed control tree are top-level snapshot data, not
per-cell state. Attached automation explicitly projects this immutable state
into its distinct versioned `automation.SnapshotV1` DTO.

Readers must never observe a new typed control tree with an old frame or
mutable storage that the next render reuses. Publication may use a lock, an
atomic pointer to an immutable object, ownership transfer, or another proven
mechanism.

Serialization for automation or diagnostics occurs from an immutable
published `Snapshot` rather than holding the UI or terminal owner across a
potentially slow write. Retention is bounded, and expiration is reported
honestly.

This specification does not select the exact Go representation, resolved-color
encoding, or wire encoding. It defines their concurrency, atomicity,
ownership, and lifetime properties. Every encoding is nevertheless subject to
the one-cell displayed-text boundary in
[`Limited Unicode Support`](../Limited-Unicode-Support.md).

## Callbacks And Reentrancy

Application callbacks run with a documented execution context. Unless a
specific API says otherwise, control and command callbacks are serialized by
the UI owner.

- Do not invoke application callbacks while holding internal mutexes.
- Do not run an implicit nested event loop to make a callback appear
  synchronous.
- Callback-requested mutations use the supplied owner-scoped facility or are
  queued for a later owner turn.
- Recursive invalidation and duplicate updates may be coalesced, but their
  observable completion rules remain explicit.
- Removing a control or subscription during its callback must not produce a
  use-after-destroy transition or a later callback through a stale alias.
- A slow or blocking application callback can delay UI progress; the toolkit
  documents that consequence rather than starting an unowned goroutine.
- Panic and failure boundaries must preserve lock, owner, terminal, and
  shutdown invariants. The toolkit must not silently treat an unknown panic
  as a successful command.

Callbacks must not be used as an implicit reflective data-binding system.
Typed adapters and generated or handwritten update functions remain valid
consumer choices.

## Layout Attachment And Overflow Delivery

Layouts are independently constructed. `Panel.SetLayout` is a UI-owner
mutation that atomically attaches the complete Layout to that Panel. No
reader, layout pass, or renderer observes a partially validated attachment.
Validation or attachment failure leaves both objects, the control tree,
logical geometry, effective clips, and the current frame unchanged, and
attachment never reparents a control. Replacement and detachment semantics
remain under design. When called during the running lifecycle,
`Panel.SetLayout` follows the same documented marshaling, completion,
cancellation, and shutdown rules as other control-tree mutations.

When available geometry is smaller than a Layout's combined minimum, layout
preserves logical minima and the resulting logical rectangles. Effective
ancestor clips constrain paint, hit testing, cursor placement, and
presentation. Layout then produces an immutable structured overflow fact.
Overflow delivery obeys these rules:

- the fact is transferred to the UI owner only after the layout pass
  completes;
- an application `Overflow` callback or handler never runs inside layout
  evaluation or while an internal mutex is held;
- exactly one callback is queued for each continuous Panel/Layout overflow
  episode; repeated passes and deficit changes update snapshots without
  another callback, while recovery followed by recurrence begins a new
  deliverable episode;
- the handler may accept or decline the notification without suppressing the
  overflow fact from the published snapshot; and
- handler-requested mutations are scheduled for a later owner turn and cannot
  recursively reenter the layout pass.

Overflow callbacks run on a bounded application-callback dispatcher rather
than the Layout, render, presentation, or UI owner. Delivery carries a
documented cancellation/deadline signal. If a handler panics, misses the
disposition deadline, ignores cancellation, or the dispatcher is unavailable
or saturated, the toolkit records delivery failure and selects the default
fallback. It does not create a replacement goroutine for every episode. Thus
an uncooperative application handler may consume only a bounded dispatcher
slot and cannot freeze Layout, painting, input, or automation request
completion.

If no handler is registered, exactly one default-notification attempt is queued
after layout for that episode. If a handler later selects the default
disposition, that episode's one fallback selection is another owner-queued
transition. With sufficient interactive geometry, the toolkit presents one
compact dismissible warning overlay with an `OK` action. The overlay is outside
the failing Layout and does not itself produce another overflow notification.
A tiny terminal uses a bounded high-visibility indicator; zero-sized and
headless execution retain deterministic nonblocking semantic evidence.
Dismissal acknowledges the warning but does not claim that the geometry has
recovered.

The resize, input event, or automation request that caused overflow may
complete once structured overflow is published and bounded notification
delivery is queued. Its associated root `Snapshot` records the notification
as pending or current, and attached automation observes the corresponding
`automation.SnapshotV1`. Completion does not wait for application callback
execution or warning dismissal. Callback disposition and fallback changes
publish later sequenced snapshots, which attached automation can observe
without holding the original request open.

Queue saturation, cancellation, shutdown, or handler failure must have an
explicit observable outcome rather than losing the fact, opening unbounded
overlays, or leaving automation waiting indefinitely. Exact overflow-handler
and report type names remain under design. The complete contract is in
[`layouts-and-overflow.md`](layouts-and-overflow.md).

## Background Work And Consumer Results

Toolkit-owned and consumer-owned workers never mutate controls, focus,
layout, intended-frame state, or the terminal backend directly. They produce
bounded typed results and submit them to the UI owner.

Every toolkit goroutine has:

- a lifecycle owner;
- a stop condition and cancellation path;
- a bounded input or output path;
- an observable error path; and
- a way for shutdown to wait for it.

Results that can become stale should carry stable request, generation, model,
or control identity so the controller can reject them deliberately.
Backpressure, coalescing, and dropped-result reporting must be explicit.
Toolkit code must not hold an internal lock while calling consumer code or
performing an unbounded send.

## Shutdown And Cancellation

Application lifecycle transitions are serialized and observable, for
example: constructed, running, stopping, and stopped. Exact public names
remain under design.

- Concurrent shutdown requests are safe and converge on one shutdown.
- Shutdown initiation is idempotent.
- New work is rejected once the applicable stopping boundary has passed.
- Accepted queued work is either drained or completed with an explicit
  shutdown/cancellation outcome according to the documented policy.
- Blocking calls and queue operations observe cancellation.
- Input-source state, including held automation modifiers, is released or
  cleared.
- Exit-producing work publishes the required final snapshot before its
  completion closes.
- The presentation owner restores terminal state and stops before shutdown
  reports completion.
- All toolkit-owned goroutines are stopped and joined; no application
  callback begins after the documented terminal shutdown point.

Closing a listener, queue, or subscription is performed by its owner.
Receivers do not close shared producer channels as an ad hoc cancellation
mechanism.

## Backend OS-Thread Affinity

If a terminal backend, GUI bridge, platform API, or test adapter requires one
operating system thread, the presentation owner may lock its goroutine to
that thread. Initialization, all backend calls, failure cleanup, and teardown
then occur on the required thread.

This affinity is an adapter concern:

- controls, controllers, consumer models, and snapshots do not acquire
  OS-thread affinity;
- other goroutines communicate with the adapter only through bounded owner
  handoffs;
- callbacks are not moved onto the presentation thread unless their public
  contract explicitly says so; and
- a backend without affinity requirements must not impose needless process-
  wide thread pinning.

The presentation owner remains unique even when the backend itself claims to
be thread safe.

## Consumer Model Boundary

The toolkit does not own or synchronize an arbitrary consumer model.
Applications choose an appropriate model strategy, such as:

- immutable state replaced by an update function;
- a consumer-owned mutex or actor;
- a database or service boundary;
- a controller-owned mutable model; or
- another explicitly synchronized architecture.

The toolkit guarantees that its documented callbacks, events, marshaling,
and snapshots can be composed with those strategies. It does not inspect
fields reflectively, infer locks, make arbitrary domain objects atomic, or
require a toolkit model superclass. If a model value crosses into the view,
the consumer and toolkit must establish a copy, immutable value, or explicit
ownership contract.

## Verification

Ordinary and external-package tests must cover:

- concurrent posts, synchronous calls, property reads, and snapshot reads;
- concurrent `Panel.SetLayout` attachment, failed validation, and observation
  without partial attachment;
- calls racing with cancellation and shutdown;
- owner-context synchronous calls without enqueue-and-wait deadlock;
- callbacks that post, invalidate, remove controls, or request shutdown;
- overflow handlers that accept, decline, post view changes, fail, or are
  absent, including coalescing and recovery of an overflow episode;
- interactive warning, tiny-terminal, and headless overflow fallbacks without
  recursive layout, overlay storms, or blocked automation completion;
- queue saturation, backpressure, fairness, and stale background results;
- rendering and snapshot publication while other goroutines observe;
- two or more independent application instances running in parallel;
- final snapshot publication and complete goroutine teardown;
- backend initialization, calls, and teardown on one owner, including a
  platform-specific affinity test where required; and
- consumer examples using concurrent MVC, MVVC, or related update logic
  without reflective binding.

Run the representative concurrent suites with Go's race detector. Add
bounded deadlock and goroutine-leak tests that synchronize on explicit
events rather than arbitrary sleeps. Stress scheduling and cancellation
orders, retain reproducible seeds or traces, and test with a recording
backend before relying on a real terminal.

The race detector supplements ownership reasoning; one clean run is not
proof that unexecuted paths are safe.

## Acceptance Criteria

- Supported public operations can be called from documented concurrent
  contexts without races, terminal corruption, or scheduler-dependent
  behavior.
- All UI mutations reach one authoritative UI owner.
- Rendering and presentation use one owner each or a documented shared owner,
  with immutable or ownership-transferring handoffs between separate stages.
- Concurrent readers receive atomic immutable snapshots.
- `Panel.SetLayout` atomically attaches an independently constructed Layout
  and leaves the prior attachment intact on failure.
- Owner callbacks cannot trigger enqueue-and-wait deadlock or nested event
  pumps.
- Layout overflow is published after layout, outside internal locks, with
  coalesced application handling and a deterministic nonblocking fallback.
- Background results, cancellation, and shutdown have bounded, explicit
  outcomes.
- A backend can require OS-thread affinity without imposing it on consumer
  models.
- Concurrent MVC, MVVC, and related consumers are supported through neutral
  typed events, commands, marshaling, and observation primitives.
- No reflective binder, reactive runtime, application model type, or
  unapproved snapshot encoding is made mandatory by this contract.
