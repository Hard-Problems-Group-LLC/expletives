# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-08-02 — `EXPL-TASK-036` — Deliver Phase 19 terminal compatibility and
  operational hardening.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-08-02T02:08:22-07:00
  - Scope: terminal capability and terminfo integration; physical input,
    resize, signals, suspend/resume, and teardown; remote/fallback behavior;
    bounded concurrency, sustained-load, failure-recovery, and profiling
    evidence across the declared terminal matrix.
  - Dependencies: completed Phase 18 interaction/control-state matrix.
  - Blockers: none.
  - Status: slices 19.0 through 19.2 fixed the compatibility matrix, bounded
    compiled-terminfo corroboration, and verified `SIGTSTP`/`SIGCONT` terminal
    ownership. Slice 19.3 now has cleanup-debt recovery tests, bounded renderer
    workloads through 1200 by 1200, and an allocation-reduced coalesced-input
    path; the full verification gate is active.
