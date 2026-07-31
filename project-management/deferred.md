# Deferred Tasks

Use this file for intentionally paused work moved from the backlog or
in-progress lists without changing the original task wording.

## Include

- stable ID
- the original request as a Markdown block quote or linked immutable record
- reason for deferral
- deferral timestamp
- the authority that chose to defer it
- the condition or decision that would reactivate it

## Deferred Work

- 2026-07-24 — `EXPL-TASK-006` — Deliver the Structured Input control family.
  - Original request:

    > Alright, we're going to defer the Structured Input controls you listed.

  - Deferred controls:
    - `FormPanel`;
    - `Wizard`; and
    - `StepContainer`.
  - Reason: the active sequence first establishes the ownership, rendering,
    automation, layout, field, navigation, scrolling, collection, and modal
    contracts on which structured multi-control workflows should build.
  - Deferred: 2026-07-24T07:07:26-07:00
  - Authority: project operator
  - Reactivate: after Text and Numeric Input, Navigation and Chrome, Scrolling
    and Content, Collections, and Modal Controls are stable, or when the
    operator explicitly reprioritizes Structured Input.
  - Required next state: restore the family to the roadmap through a reviewed
    proposal that defines composition, validation, navigation, and
    transactional workflow behavior.
  - Related:
    - [`docs/specifications/control-catalog.md`](../docs/specifications/control-catalog.md)
    - [`EXPL-DEC-002`](decision-log.md#expl-dec-002--control-delivery-order-and-basic-automation-input)

- 2026-07-24 — `EXPL-TASK-004` — Add authentication and capability
  authorization to attached automation.
  - Original request:

    > Automation does not have to be authenticated out of the gate. The user
    > simply will not pass `--automation` in risky situations.

    > Authentication beyond that should be moved to a proposal for now, or
    > deferred into the backlog.

  - Reason: the initial development endpoint deliberately uses explicit
    `--automation <socket-path>` as the operator-controlled trust boundary.
    Authentication and capability policy would add design and implementation
    scope that is not required for the first closed-loop development surface.
  - Deferred: 2026-07-24T03:03:22-07:00
  - Authority: project operator
  - Reactivate: before claiming attached automation is suitable for hostile,
    multi-user, remotely reachable, or elevated environments, or when an
    operator explicitly prioritizes hardened authentication.
  - Required next state: prepare an under-review threat-model and protocol
    proposal before implementation.
  - Related:
    - [`docs/specifications/expletives-test.md`](../docs/specifications/expletives-test.md)
    - [`EXPL-DEC-001`](decision-log.md#expl-dec-001--product-test-automation-and-build-direction)
