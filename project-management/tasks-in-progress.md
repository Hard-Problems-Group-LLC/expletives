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
  - Status: shared contracts are fixed. ListBox, DropDown, and ComboBox now
    have stable-key repair, keyboard-complete selection/editing, transient
    popup ownership, compact typed automation, catalog/self-check coverage,
    full ordinary/race verification, debug builds, live attached evidence, and
    pushed ACPs. TreeView now has its copied flat preorder model, stable
    expansion/current/selection repair, keyboard behavior, compact typed
    automation, catalog fixture, full ordinary/race/static/self-check gates,
    and live raw-key/frame evidence. ACP is the remaining checkpoint before
    Table.
