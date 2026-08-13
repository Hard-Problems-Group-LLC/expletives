# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-08-12 — `EXPL-TASK-039` — Deliver Phases 20–24 Table Improvements.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-08-12T16:26:41-07:00
  - Scope: inventory the existing `Table` and `DataGrid` contracts and
    implementation in detail, turn operator-requested improvements into a
    directed compatible contract, and deliver the five-phase implementation
    sequence across selection, presentation/wrapping, Columns interaction,
    visual roles/catalog, and final integration/acceptance.
  - Dependencies: completed Phase 16 Collections and Phase 19 Terminal
    Compatibility.
  - Blockers: none.
  - Status: Slices 20.0 and 20.1 are complete. `EXPL-PROP-003`,
    `EXPL-DEC-018`, the bounded `EXPL-REV-002` Panel record, and the directed
    Table Improvements v1 contract record the operator-approved design.
    The roadmap now assigns implementation to Phases 20 through 24. Phase 20
    is complete on `feature/table-improvements`: the first coherent delivery
    adds canonical None/Single/Range/Multiple selection policy,
    stable range endpoints, Shift and bracket fallback interaction, exact
    sort/model repair, atomic direct and Transaction mutations, DataGrid edit
    cancellation, typed core/automation evidence, and focused tests.
    Kind-consistent core and compact automation details, public-consumer
    coverage, a reusable catalog selection matrix, explicit catalog policy,
    and atomic reset are complete. The complete `make verify` phase gate
    passed on 2026-08-13, including vet, ordinary and race tests, the PTY
    lifecycle integration test, and all required build modes. Slices 21.0 and
    21.1 are complete: copied presentation state, exact direct/Transaction and
    complete-replacement APIs, stable schema repair, visible-order rendering,
    width, navigation, hidden sorting, DataGrid edit traversal, compact
    automation evidence, reusable catalog checks, and reset behavior have
    focused ordinary/race evidence. Slices 21.2 through 21.5 add exact
    Clip/Wrap/Hang frames and one-cell Unicode handling, cached variable-height
    geometry and visual-distance paging, single-line DataGrid edit/commit
    reflow, fuzzing, compact automation validation, and reusable catalog
    coverage. The complete `make verify` Phase 21 gate passed on 2026-08-13,
    including vet, ordinary and race tests, the PTY lifecycle integration
    test, and every required build mode. Phase 21 is complete. Slice 22.0 is
    active on the bounded compatible feature set and optional Columns action
    band. Later Phases 23 and 24 own per-instance visual roles and dedicated
    catalog screens, then final integration/acceptance.
