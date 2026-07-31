# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-07-31 — `EXPL-TASK-029` — Deliver Phase 14 Navigation and Chrome.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T03:47:00-07:00
  - Scope: `ScrollBar`, `TabbedPanel`, `Notebook`, and `Tab`, including
    viewport ownership, tab-page visibility, grouped keyboard navigation,
    deterministic mutation, and typed automation.
  - Acceptance: Phase 14 of
    [`development-roadmap.md`](development-roadmap.md).
  - Dependencies: completed Phase 13 Progress, shared grouped focus, Actions,
    Menus, TextArea viewport primitives, Layouts, and automation.
  - Blockers: none.
  - Status: viewport and tab contracts are fixed. ScrollBar library, public
    consumer, and automation tests pass; completing its catalog/build
    checkpoint before TabbedPanel and Notebook.
