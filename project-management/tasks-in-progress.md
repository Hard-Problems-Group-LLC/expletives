# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-07-31 — `EXPL-TASK-027` — Deliver Phase 12 Text and Numeric Input.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T01:53:48-07:00
  - Scope: `TextField`, `NumberField`, `SpinBox`, and `TextArea`, beginning
    with the directed validator and password contract.
  - Acceptance:
    [`docs/specifications/text-and-numeric-input-api-v0.md`](../docs/specifications/text-and-numeric-input-api-v0.md).
  - Dependencies: completed Phase 11 focus, Selection, typed snapshot, and
    raw automation input contracts.
  - Blockers: none.
  - Status: TextField, NumberField, and SpinBox are complete with shared
    bounded editing, exact fixed-place numeric policy, typed core/automation
    evidence, catalog coverage, and full `make verify` evidence. TextArea,
    bounded paste/selection behavior, and the Phase ACP gate remain.
