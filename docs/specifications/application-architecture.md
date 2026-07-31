# Consuming-Application Architecture

Status: Directed requirement; helper APIs remain under design
Authority: Direct operator request on 2026-07-24
Related decisions: `EXPL-DEC-003`, `EXPL-DEC-004`, and `EXPL-DEC-007` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)
Concurrency contract:
[`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md)
Layout and overflow contract:
[`layouts-and-overflow.md`](layouts-and-overflow.md)

## Purpose

Consuming projects must be able to organize an `expletives` application using
Model-View-Controller (MVC) or a closely related separation such as MVVC,
MVVM, MVP, MVU, a presentation model, or another explicit unidirectional
architecture.

The toolkit must make that separation natural without requiring one named
application framework. Its public behavior must remain safe when consumer
models, controllers, presenters, view models, and background services use
multiple goroutines or operating-system threads.

## Required Separation

### Model

Application domain state belongs to the consuming application:

- a model may be any appropriate Go type and need not implement or embed a
  toolkit base type;
- model behavior must be testable without a terminal, control tree, or
  automation endpoint;
- the toolkit must not require domain state to live in controls or a
  package-global application singleton; and
- exposing a view or automation snapshot must not implicitly expose the
  application's unrestricted domain model.

The consuming application owns synchronization of its model. The toolkit
does not make an arbitrary model thread safe, infer its locks, or inspect it
through mandatory reflection. State crossing into the view must use a copy,
an immutable value, or an explicit ownership contract.

### View

The `expletives` application, root `Panel`, descendant controls, layout, and
intended-frame rendering form the view boundary:

- controls display explicit application state and emit typed interaction
  events or structured commands;
- Layouts are independently constructed view objects and are attached
  atomically to a parent through `Panel.SetLayout`;
- rendering and layout must not mutate unrelated domain state;
- a view can be constructed and exercised through public APIs in a headless
  test; and
- application-specific view adapters may translate model state into control
  state without becoming mandatory toolkit base classes.

### Controller Or Update Logic

Consuming applications must be able to put application decisions outside the
controls:

- a controller, update function, presenter, or equivalent receives structured
  commands and relevant control events;
- it applies application rules to the model and schedules resulting view
  state on the authoritative UI owner;
- background workers return bounded typed results rather than mutating
  controls or using the terminal backend directly; and
- human input, raw automation key events, direct automation commands, and
  headless tests converge on the same command and application-rule paths.

The toolkit may provide optional dispatcher, adapter, or binding helpers, but
ordinary use must not require reflection-driven two-way binding, a reactive
runtime, or inheritance from a toolkit model/controller hierarchy.

## Layout Overflow As An Application Event

When a Layout's available rectangle is smaller than its combined logical
minimum, the Layout preserves those minima and resulting logical rectangles.
Paint, hit testing, cursor placement, and presentation remain constrained by
the effective ancestor clip. The Layout publishes a structured overflow fact.
This condition is part of view state; it must not silently mutate or compress
application model data.

A consuming application may register an `Overflow` callback or handler at the
toolkit's documented application or view boundary. The handler can accept the
notification—for example, by scheduling a controller or view response—or
decline it. The exact handler type and registration API remain under design.
The callback runs after the layout pass on a bounded application-callback
dispatcher, never inside layout evaluation, on the UI owner, or while toolkit
locks are held. It receives documented cancellation/deadline signaling;
dispatcher saturation, panic, cancellation, or a missed deadline selects the
default fallback without spawning unbounded replacement goroutines.
Callback-requested changes are therefore a later owner turn rather than
recursive layout.

When the application supplies no handler, or the handler declines, the toolkit
uses a deterministic sensible fallback. With sufficient interactive geometry,
that is a compact dismissible warning overlay with an `OK` action. A tiny
terminal uses a bounded high-visibility indicator; zero-sized and headless
execution retains nonblocking semantic evidence. The event that caused layout
may complete once structured overflow is published and bounded notification
delivery is queued. Its associated snapshot records notification state as
pending or current; callback disposition and fallback changes publish later
sequenced snapshots. Completion waits for neither callback execution nor human
dismissal. The fallback never turns overflow into an implicit
application-model decision.

Continuous overflow for one Panel/Layout pair is one episode with exactly one
callback delivery, even when later passes change the deficit. Recovery then
recurrence begins another episode. The fallback overlay is outside the failing
Layout and cannot recursively trigger the same notification. The active
overflow and fallback state remain observable in the immutable snapshot so an
MVC-, MVVC-, or similarly structured controller and attached automation can
react without a private view backdoor. The full logical-rectangle,
effective-clipping, notification, and fallback contract is in
[`layouts-and-overflow.md`](layouts-and-overflow.md).

## Ownership And Concurrency

An explicit application instance owns its root and UI-owner event loop. This
supports multiple isolated applications in tests and avoids global state that
would entangle a controller with one process-wide view.

Before the loop starts, the consuming application may construct its view
synchronously. Concurrent use of public toolkit values must nevertheless
have documented, safe behavior. While the loop runs, model results that
affect controls must be posted or called through the documented UI-owner
mechanism. Direct running-state mutation must marshal, synchronize, or reject
with a documented error rather than race.

The toolkit may serialize input/controller work, rendering, and presentation
on one shared owner goroutine. It may instead use a UI owner, render owner,
and presentation owner separately, provided each stage has one authoritative
owner and separate stages exchange bounded immutable data or explicit
ownership transfers. A backend that requires operating-system-thread
affinity may pin its presentation owner without pinning consumer model or
controller work.

Owner marshaling defines ordering, queue bounds, completion, cancellation,
shutdown, and error behavior. A synchronous call made from an owner callback
must run safely inline or fail explicitly; it must never enqueue behind
itself and wait. Application callbacks do not run while toolkit mutexes are
held and do not create implicit nested event loops.

Atomic immutable root-package `Snapshot` values are the supported concurrent
observation surface. Their cells contain only a canonical one-cell grapheme,
semantic style, resolved foreground and background, and stable owner identity.
Cursor state and the bounded typed control tree are snapshot-level data; there
are no width or continuation-cell fields. Attached automation explicitly
projects this state into its distinct `automation.SnapshotV1` DTO. The
complete concurrency contract is in
[`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md).

## Commands, Raw Keys, And Automation

Raw `KeyDown`, `KeyUp`, and `KeyPress` automation events enter before
mnemonic, accelerator, hotkey, and command resolution. Once resolved, the
same structured command reaches the controller or equivalent update logic
that physical input uses. `KeyPress` is a distinct one-shot raw logical event
that leaves no key held; stateful chords use `KeyDown` and `KeyUp` around it.

Direct automation commands may enter at that structured boundary, but they
must still obey the same enabled state, modal scope, validation, and
application rules. Automation is not permission to invoke private model
methods or mutate arbitrary domain fields.

## `expletives-test` As An Architecture Example

`expletives-test` must be a real public-API consumer that demonstrates the
separation:

- deterministic scenario data and catalog state are its model;
- toolkit controls and layouts are its view;
- scenario selection, reset, interaction, and quit behavior pass through its
  controller or equivalent update layer; and
- normal Go tests can exercise its model and update behavior independently,
  then verify the composed view headlessly and through attached automation.

This application is both a control catalog and an executable example of a
maintainable consuming-project structure.

## Non-Goals

This requirement does not yet:

- mandate classic MVC terminology or one package layout;
- approve a public `Controller` interface, data-binding DSL, reactive runtime,
  or state container;
- require controls to know application model types;
- synchronize arbitrary consumer-owned model state;
- require controller, renderer, and presenter work to run concurrently rather
  than on serialized owners;
- make every control event remotely invocable; or
- permit automation snapshots to dump unrestricted model state.

Those helpers may be proposed after the foundational event, command,
ownership, and update contracts are settled.

## Acceptance Criteria

- A small external package can keep its model free of `expletives` types,
  construct a view from public APIs, and route a control command through
  separate application logic.
- The same application logic can be tested without initializing a terminal.
- Physical keys, raw automation key lifecycle events, and direct automation
  commands reach the same applicable command rules.
- Background work cannot mutate controls outside the UI-owner path.
- Consumer model and service goroutines can safely post typed results and
  observe immutable snapshots while the application runs.
- A Layout can be independently constructed and attached atomically through
  `Panel.SetLayout` without changing control parentage.
- A below-minimum Layout preserves logical minima and resulting logical
  rectangles, constrains effective output by every ancestor clip and the
  application surface, and exposes structured overflow; application handling
  and toolkit fallback run after layout without locks, recursive warning
  storms, or waits that hang headless or attached automation.
- A synchronous UI-owner call from an owner callback cannot deadlock by
  enqueueing behind itself.
- Two independent headless application instances can run in parallel without
  sharing a global root, controller, model, or key state.
- Representative concurrent application paths pass race and bounded deadlock
  tests.
- `expletives-test` documents and tests its chosen MVC-like package
  boundaries.
