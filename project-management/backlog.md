# Backlog

Use this backlog to track pending work in priority order. Keep the next item
at the top. Use a dated bullet with an indented body for each entry. Include a
stable ID, concise context, requestor or owner details, acceptance criteria,
dependencies and blockers, links to authoritative proposals, bugs, or
specifications, and ISO 8601 timestamps.

## Current Queue

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
    display, action, selection, input, progress, navigation, content,
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
