# Review Finding: EXPL-REV-001-D-S02-F001

- Finding ID: `EXPL-REV-001-D-S02-F001`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The public model conflates universal control identity with container
  capability by requiring `*Panel` as every parent.
- Consequence: A future leaf control that shares Panel behavior will also
  appear structurally usable as a parent. The implementation must either admit
  invalid child relationships or accumulate runtime kind checks and
  exceptions. This freezes the unresolved extension model into constructors
  and layout APIs.
- Evidence:
  - `docs/specifications/control-catalog.md:23-56` distinguishes
    container-capable controls while leaving the exact composition and
    capability model unresolved.
  - `project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md:279-290`
    calls for a small toolkit-owned container capability and compile-time
    rejection where practical.
  - `docs/specifications/go-api-v0.md:164-170` requires `parent *Panel` for
    ordinary constructors.
  - `panel.go:31-58` makes Panel the foundational container and embeds it in
    Frame and GroupBox.
  - `project-management/open-questions.md:10-13` and
    `project-management/decision-log.md:379-384` leave the exact Go extension
    mechanism open.
- Recommendation: Resolve Q-002 before freezing common-control constructors.
  Separate the universal control node or handle from an explicit container
  capability, and accept that capability as the parent. Define how container
  controls expose it and leaf controls do not.
- Related lenses: public API stability, hierarchy invariants, layout
  extensibility
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S03-F001`. The cited
  `project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md`
  does not exist in the frozen inventory, but the remaining cited specification,
  implementation, and open-question evidence substantiates the same claim.
- Resolution: pending
- Resolution evidence or operator record: pending
