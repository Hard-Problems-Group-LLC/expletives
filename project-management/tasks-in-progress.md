# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-07-31 — `EXPL-TASK-030` — Deliver Phase 15 Scrolling and Content.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T04:35:20-07:00
  - Scope: `ScrollablePanel`, `Viewport`, `MarkdownView`, `LogView`, and
    `StreamView`, including exact viewport ownership, bounded parsing and
    retention, follow/scrollback, honest drop accounting, and typed
    automation.
  - Acceptance: Phase 15 of
    [`development-roadmap.md`](development-roadmap.md) and
    [`scrolling-content-api-v0.md`](../docs/specifications/scrolling-content-api-v0.md).
  - Dependencies: completed Phase 14 ScrollBar/tab navigation, Layouts,
    grouped focus, Limited Unicode, and automation.
  - Blockers: none.
  - Status: directed contracts and the opening compatibility slice are
    complete: Cycle semantics, per-axis Layout hints/weights, and the compact
    Text/Numeric form pass the complete verification gate. Viewport and
    ScrollablePanel implementation is active.
