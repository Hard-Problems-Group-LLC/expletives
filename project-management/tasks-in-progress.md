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
    its exact ACP is staged and awaits an explicitly configured Git identity.
    DataGrid now reuses the private Table model/viewport capability and has
    validated single-line cell editing, Enter/Escape commit/cancel,
    editable-cell Tab traversal, compact redacted automation, a catalog
    fixture, and focused ordinary/concurrent/fuzz tests. Static analysis, the
    full ordinary and race suites, all build modes, and self-check pass. Live
    64x20 frame evidence confirms usable Table/DataGrid body viewports; raw
    key evidence confirms soft edit/commit, hard rejection, cancellation,
    forward Tab traversal, and a KeyDown/KeyPress/KeyUp Shift-Tab chord.

- 2026-07-31 — `EXPL-TASK-032` — Deliver Phase 17 Modals.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T13:00:00-07:00
  - Scope: bounded LIFO ModalPanel/Dialog capture and results, then MessageBox,
    ConfirmDialog, InputDialog, and ProgressDialog with typed automation and
    catalog coverage.
  - Dependencies: verified Phase 16 implementation; ACP publication remains
    administratively blocked only by the missing explicit Git identity.
  - Status: the focused pre-v1 modal contract is recorded in
    [`modals-api-v0.md`](../docs/specifications/modals-api-v0.md). ModalPanel now
    implements a bounded one-shot LIFO lifecycle, explicit nesting, recursive
    minimum and resize geometry, Turbo shadow, scoped focus/input/commands,
    exact restoration, final interrupt/quit results, typed automation, and
    client-side stack validation. Dialog derives that lifecycle with distinct
    theme/kind defaults and dialog-wide default/cancel and mnemonic scopes.
    MessageBox and safe-default ConfirmDialog now close through immutable local
    commands in the same exact completion publication. InputDialog now composes
    the ordinary validated TextField, accepts only valid committed values,
    restores its initial value on cancel, and redacts both value and validator
    alphabet from automation regardless of Password mode. Focused core,
    projection, and hostile-client validation tests pass. ProgressDialog now
    provides atomic copied status/progress updates, a stable operation context,
    a one-shot non-closing cancellation request, disabled-repeat behavior,
    explicit application acknowledgement, failure results, typed compound
    automation, and concurrent/race-tested access. Standard modal compounds
    are complete. The Dialogs menu now opens all four standard compounds with
    stable automation keys and disposes closed one-shot controls for repeatable
    use. Catalog self-check covers every close path. Full vet, ordinary/race,
    all-mode build, and self-check gates pass. Attached raw-key automation
    verified Message OK, Confirm safe-default No, Input invalid rejection and
    cancellation with redaction, and Progress's exact still-active request
    frame before controller acknowledgement; retained character frames were
    visually inspected. The Phase 17 ACP is ready but remains administratively
    queued behind the same missing explicit Git identity as Phase 16.
