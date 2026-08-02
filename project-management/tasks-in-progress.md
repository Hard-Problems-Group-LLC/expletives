# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-08-02 — `EXPL-TASK-033` — Deliver Phase 18 file and directory pickers.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-08-02T00:55:00-07:00
  - Scope: bounded provider and local adapter; Turbo Vision-style single-file,
    explicit multiple-file, and directory-only compound dialogs; typed
    results; deterministic tests; catalog pages; attached automation.
  - Contract:
    [`file-pickers-api-v0.md`](../docs/specifications/file-pickers-api-v0.md).
  - Dependencies: completed Phase 16 Collections and Phase 17 Modals at
    `f0a63d2`, plus the standard-dialog visual correction at `d0a36e4`.
  - Blockers: none.
  - Status: slice 18.0 fixes provider, result, I/O ownership, filtering,
    sorting, interaction, automation, and local-adapter contracts before the
    core compound implementation.
