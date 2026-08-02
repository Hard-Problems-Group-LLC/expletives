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
  - Status: slices 19.0 through 19.3 fixed the compatibility matrix, bounded
    compiled-terminfo corroboration, verified `SIGTSTP`/`SIGCONT` terminal
    ownership, cleanup-debt recovery, profiling workloads through 1200 by
    1200, allocation-reduced coalesced input, and 512-frame concurrent
    Presenter serialization under the race detector. Slice 19.4 physical
    terminal-matrix acquisition is active: native XTerm 366 UTF-8, XTerm plus
    tmux 3.2a (`TERM=screen`), and conservative XTerm `C`-locale visual rows
    pass. Actual-operator keyboard/paste/job-control confirmation, GNU Screen,
    and one real remote-transport row remain under `EXPL-REQ-004` before the
    phase-completion ACP.
