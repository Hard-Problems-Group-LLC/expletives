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
  - Status: slice 19.0 fixed the compatibility matrix and slice 19.2 wired
    and verified `SIGTSTP`/`SIGCONT` through the real terminal owner. Slice
    19.1 capability and terminfo probes is next.
