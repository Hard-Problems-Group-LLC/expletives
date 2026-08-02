# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-07-31 — `EXPL-TASK-031` — Deliver Phase 16 Collections.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T09:37:00-07:00
  - Scope: `ListBox`, `ComboBox`, `DropDown`, `TreeView`, `Table`, and
    `DataGrid`, plus copied `ListItem`, `TreeNode`, and `Column` data,
    stable-identity repair, keyboard-complete focus/selection, bounded
    mutation, rendering, and typed automation.
  - Acceptance: Phase 16 of
    [`development-roadmap.md`](development-roadmap.md) and the directed
    Collections API contract created in its opening slice.
  - Dependencies: completed Phase 15 content viewport and scrolling,
    Selection, Actions, Menus, grouped focus, Layouts, and automation.
  - Blockers: none.
  - Status: ListBox, DropDown/ComboBox, and TreeView are verified pushed ACPs
    through `cf8fd19`. Table now has its copied canonical column/row model,
    stable row/cell current and selection, derived stable sorting, sticky
    header, keyboard behavior, compact typed automation, catalog fixture,
    documentation, focused tests, static analysis, all build modes, and
    self-check. Full ordinary/race and live raw-key/frame verification passed;
    the Table ACP is the remaining checkpoint before the DataGrid slice.
