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
    complete with the bounded copied feature set, optional owned action band,
    reduced body viewport, semantic focus-part repair, DataGrid edit
    cancellation, and core/compact automation evidence. Slices 22.1 through
    22.4 complete the raised Columns action, fixed-tree modal editor, private
    draft/search/visibility/wrap/order controls, atomic revision-aware apply,
    schema rebase, explicit stale Reload/Cancel handling, DataGrid pre-open
    commit, owner/modal lifecycle cleanup, focus restoration, and compact
    typed evidence. Slice 22.5 completes fixed-tree maximum-schema evidence,
    compact snapshot validation and response bounds, focused race tests,
    catalog self-check, and an attached drive/observe edit/apply/cancel/
    shutdown matrix through `expletivesctl`. The complete `make verify` Phase
    22 gate passed on 2026-08-13, including vet, ordinary and race tests, the
    PTY lifecycle integration test, every required build mode, and smoke/
    self-checks. On 2026-08-13 the operator directed automatic delivery through
    Phase 23 because its dedicated screens are the practical evaluation
    surface for the Columns behavior. The attached Phase 22 matrix closes its
    behavior gate and Phase 22 is complete. Phase 23 is now complete: fixed
    per-instance visual roles, staged-Theme validation, exact core and compact
    automation evidence, dedicated Table/DataGrid routes, 3:1 live-control/
    Notebook screens, scrollable Options and Colors pages, named palette
    changes, comprehensive fixtures, and atomic reset are delivered. The
    exact final worktree passed `make verify`; attached automation passed the
    wide and 84-by-24 layouts, live Columns toggle, Columns modal focus/close,
    representative DataGrid Body color change, instance isolation, reset, and
    final-snapshot shutdown. The operator's automatic-delivery direction
    closed Phase 23's delegated acceptance. On 2026-08-13, attached human
    evaluation then exposed `EXPL-BUG-012`: palette `■` glyphs inherited
    DropDown row styles instead of representing their named colors. Completed
    Slice 23.6 added one-cell Theme-owned collection indicators, catalog
    swatches, exact frame coverage, and attached regression evidence; the full
    gate passed and Phase 23 was complete again. Subsequent attached keyboard
    evaluation exposed `EXPL-BUG-013`: at particular Colors viewport offsets,
    focusable Notebook and ScrollablePanel ancestors tie as cross-group
    spatial candidates and block the otherwise adjacent row transition.
    Phase 23 reopened for bounded Slice 23.7 repair; exhaustive in-process and
    rebuilt attached checks now pass. Operator inspection then confirmed
    `EXPL-BUG-014`: the catalog's capped Grow columns stop at 81 cells inside
    a 120-cell Table viewport. Completed Slice 23.8 makes Summary the unbounded
    flexible catalog column and proves exact wide fill plus retained narrow
    overflow for both controls. The rebuilt attached checks and complete
    `make verify` gate passed for both regression repairs; Phase 23 is complete
    again. The deterministic 17-row Phase 24 covering array and targeted
    Range/edit/dialog matrices are
    recorded in `table-improvements-phase-24-matrix.md`. Phase 24 remains
    planned for final cross-axis integration and acceptance.
