# Backlog

Use this backlog to track pending work in priority order. Keep the next item
at the top. Use a dated bullet with an indented body for each entry. Include a
stable ID, concise context, requestor or owner details, acceptance criteria,
dependencies and blockers, links to authoritative proposals, bugs, or
specifications, and ISO 8601 timestamps.

## Current Queue

- 2026-08-12 — `EXPL-TASK-037` — Resume Phase 25 Release Readiness.
  - Requestor: project operator
  - Owner: Codex
  - Created: 2026-08-10T14:21:00-07:00
  - Paused: 2026-08-12T16:26:41-07:00 when Table Improvements was inserted as
    the Table Improvements program, now Phases 20 through 24.
  - Context: Slice 25.0 already completed several concrete downstream and
    native-terminal repairs while its public-surface inventory was underway.
    Resume the remaining API/documentation/example audit, supported-toolchain
    and dependency checks, build/provenance work, external-consumer gate, and
    clean-checkout release decision after Phase 24 closes.
  - Acceptance criteria: the Phase 25 acceptance gate in the development
    roadmap passes; tag creation or public release remains a separate explicit
    operator decision.
  - Dependencies: Phases 20 through 24 Table Improvements and every other non-deferred
    catalog phase.
  - Blockers: the Table Improvements program is active in Phase 20.

- 2026-08-02 — `EXPL-TASK-035` — Build a maintainable developer crash course
  and toolkit reference manual.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-08-02T02:15:00-07:00
  - Context: consuming developers, maintainers, and AI collaborators need a
    concise path from first application to advanced toolkit work, plus a
    dependable reference that can evolve with the public API without becoming
    a second stale specification set.
  - Acceptance criteria:
    - provide a task-oriented crash course covering application startup,
      control ownership, Layouts, actions/events, focus and keyboard behavior,
      styling, modal workflows, concurrency, testing, and automation;
    - provide a public reference organized by package, control family, and
      cross-cutting contract, with runnable examples where practical;
    - define a low-overhead maintenance workflow for both humans and AI,
      including mechanically generated or checked API inventories and links to
      authoritative `docs/specifications/` contracts instead of duplicated
      normative prose;
    - add documentation verification to normal project checks where practical;
      and
    - document how API changes must update examples, reference indexes, and
      compatibility notes.
  - Dependencies: public API and control catalog should be substantially stable
    before the first complete edition; incremental scaffolding may begin sooner.
  - Blockers: none; scheduled behind active Phase 18 picker delivery.

- 2026-08-02 — `EXPL-TASK-034` — Present Help/About as a standard dialog.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-08-02T01:35:00-07:00
  - Context: the catalog About screen predates completed standard-dialog
    support and should now use the same modal look, focus, keyboard, and
    automation behavior as other informational messages.
  - Acceptance criteria:
    - keep `Help` / `About` and its existing command as the public route;
    - replace the dedicated About content screen with a `MessageBox` or
      equivalent informational standard Dialog;
    - preserve Turbo Vision palette, default OK, Escape, focus restoration,
      repeatability, and attached-automation evidence; and
    - update the catalog self-check and menu tests.
  - Dependencies: completed Phase 17 standard dialogs and dialog visual
    correction at `d0a36e4`.
  - Blockers: none; defer until active Phase 18 picker work reaches a suitable
    checkpoint.

- 2026-07-30 — `EXPL-TASK-018` — Render a root-level undersized-geometry
  diagnostic.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-07-30T20:00:00-07:00
  - Context: an application whose offered surface is smaller than its
    recursively determined usable minimum needs an unmistakable diagnostic
    instead of a clipped or misleading ordinary scene.
  - Acceptance criteria:
    - compute the effective minimum recursively from the root, attached
      Layouts, and participating controls using the existing checked geometry
      rules;
    - when either offered dimension is below that effective minimum, render
      `TOO SMALL - MINIMUM GEOMETRY (XxY)` beginning at cell `(0,0)` with
      deterministic clipping on extremely small surfaces;
    - give this diagnostic precedence over ordinary scene rendering without
      changing the retained logical control or Layout geometry;
    - expose the condition and required minimum through typed core and
      automation snapshots; and
    - cover entry, resize while active, recovery, zero geometry, clipping,
      recursive minima, headless automation, and attached-terminal behavior.
  - Dependencies: the existing Layout measurement and root-sizing contracts.
  - Blockers: none; intentionally deferred until the operator selects the
    next work item after the completed Actions and Menus phases.
  - Related:
    - [`docs/specifications/layouts-and-overflow.md`](../docs/specifications/layouts-and-overflow.md)
    - [`docs/specifications/root-sizing-and-borders-v0.md`](../docs/specifications/root-sizing-and-borders-v0.md)
    - [`docs/specifications/application-architecture.md`](../docs/specifications/application-architecture.md)

- 2026-07-24 — `EXPL-TASK-002` — Reconcile foundational toolkit contracts.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-07-24T03:03:22-07:00
  - Context: the directed product goals are clear, while the older Draft
    contains conflicting frame, package, inheritance, and automation models.
  - Acceptance criteria:
    - finish the remaining canonical-frame extensions beyond the approved
      Basic Snapshot fields, public Go API/extension model,
      module/package names, terminal backend/support boundary, semantic input
      and configurable interrupt model, and initial automation protocol;
    - use the comparative UI-toolkit and curses/terminfo research as evidence;
    - specify stable identities, bounds, event outcomes, lifecycle, and
      validation without promoting research analogy into policy; and
    - update the durable specifications after decisions are approved.
  - Dependencies: none
  - Blockers: remaining designated decisions have not yet been made; the Basic
    Snapshot, Layout attachment, and below-minimum overflow behavior are no
    longer blocked.
  - Related:
    - [`EXPL-DEC-001`](decision-log.md#expl-dec-001--product-test-automation-and-build-direction)
    - [`docs/specifications/ui-toolkit-requirements.md`](../docs/specifications/ui-toolkit-requirements.md)
    - [`docs/research/ui-toolkit-lessons.md`](../docs/research/ui-toolkit-lessons.md)
    - [`project-management/open-questions.md`](open-questions.md)

- 2026-07-24 — `EXPL-TASK-010` — Deliver the remaining non-deferred common
  controls in roadmap order.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-07-24T07:07:26-07:00
  - Context: after layout and automation, the toolkit needs the directed
    display, action, menu, selection, input, progress, navigation, content,
    collection, and modal families.
  - Acceptance criteria:
    - implement one roadmap phase at a time in the directed order;
    - extend the public control/state catalog and MVC-compatible
      `expletives-test` consumer with every public control;
    - add normal Go unit and integration coverage wherever practical;
    - add headless and attached-automation evidence for every phase and PTY or
      real-terminal evidence where physical behavior matters; and
    - keep Structured Input excluded until `EXPL-TASK-006` is reactivated.
  - Dependencies: Basic Layouts are complete; each later phase still requires
    its phase-specific approved contract.
  - Blockers: phase-specific open questions and proposals.
  - Related:
    - [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
    - [`project-management/development-roadmap.md`](development-roadmap.md)
    - [`EXPL-TASK-006`](deferred.md)

- 2026-07-24 — `EXPL-TASK-012` — Design and implement safe control
  reparenting.
  - Requestor: project operator
  - Owner: unassigned
  - Created: 2026-07-24T08:34:21-07:00
  - Context: reparenting may be useful, but the foundational control tree will
    use immutable parent ownership until focus, Layout, event-routing,
    lifetime, and automation behavior can be designed together.
  - Acceptance criteria:
    - prepare a reviewable proposal defining atomic reparenting, allowed
      parent capabilities, cross-App rejection, lifecycle, focus/capture,
      Layout membership, Z-order, invalidation, event ordering, and snapshot
      outcomes;
    - preserve stable control identity where approved and reject failed moves
      without partially mutating either tree;
    - provide ordinary Go unit/integration, race, headless, and attached
      automation tests; and
    - update the public ownership specification before exposing the API.
  - Dependencies: stable Core/container, focus, Layout, event-routing, and
    automation contracts.
  - Blockers: no immediate blocker; intentionally lower priority than the
    foundational phases.
  - Related:
    - [`EXPL-DEC-004`](decision-log.md#expl-dec-004--root-input-concurrency-layout-and-build-foundations)
    - [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
