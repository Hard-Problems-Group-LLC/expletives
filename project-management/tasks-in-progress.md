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
  - Consumer repair (started 2026-08-07T11:14:29-07:00): radioradio exposed
    an ordinary-width horizontal-overflow failure in the single-line
    `ListBox` row contract. Slice 19.5 adds opt-in word/cell reflow with
    description-aligned hanging indents while preserving one logical stable
    item for current, selection, activation, retention, and automation.
    The local candidate and downstream integration pass the focused tests. At
    2026-08-07T11:31:14-07:00 the complete Expletives gate also passed vet,
    ordinary and race suites, headless automation, PTY lifecycle, and debug,
    release, and profiling builds. A final compact-width correction keeps the
    leading row of an over-height logical item visible and passed the repeated
    complete gate at 2026-08-07T11:43:43-07:00. The catalog race test's
    pre-existing
    30-second aggregate budget also failed on the unchanged pinned checkout;
    its two-minute harness budget now accommodates observed 70.25- and
    95.93-second race runs without changing independent protocol deadlines or
    assertions.
    Publication and downstream dependency pinning remain pending explicit
    authorization; Slice 19.4 terminal-matrix evidence is paused, not
    discarded.
