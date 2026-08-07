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
    pre-existing 30-second aggregate budget also failed on the unchanged
    pinned checkout;
    its two-minute harness budget now accommodates observed 70.25- and
    95.93-second race runs without changing independent protocol deadlines or
    assertions.
    At 2026-08-07T14:52:28-07:00 the canonical feature branch passed the
    complete gate again, including a 109.23-second race catalog. The operator
    authorized publication of the permanent feature branch and its non-fast-
    forward merge into `main`; Slice 19.5 is complete. Resume Slice 19.4 after
    the merge. Downstream dependency pinning belongs to the radioradio
    consumer and must use the published immutable revision.
  - TextField consumer repair (started 2026-08-07T15:55:18-07:00):
    radioradio ECR 2026-002 confirmed that commit-only `ChangeCommand` cannot
    keep an outgoing byte counter synchronized while editing and that one
    focused presentation role cannot distinguish selection from active edit.
    Phase 19 Slice 19.6 implements bounded live-edit/current-value,
    selected/editing style, byte-band, and submit contracts under
    `EXPL-BUG-003`. Slice 19.4 is paused while this consumer repair is active;
    tests are headless and synthetic, with no application, serial, or RF use.
    Work paused uncommitted at 2026-08-07T16:05:31-07:00 for a downstream
    operator-requested live-dialog diagnosis. Preserve the feature worktree;
    do not publish or merge it until the downstream dialog repair is resolved.
    The downstream repair passed full fake-only and restarted live acceptance
    at 2026-08-07T16:23:30-07:00. Resume Slice 19.6 in the preserved feature
    worktree; the radio remains outside upstream test scope.
    Slice 19.6 completed at 2026-08-07T16:40:12-07:00. TextField now exposes
    copied bounded live-edit, current-value, explicit submit, distinct
    selected/editing style, and UTF-8 byte-threshold contracts; password
    rendering suppresses the active band. Unit, public-consumer, automation,
    catalog, PTY, race, debug, release, profiling, self-check, and smoke gates
    pass. Publish the permanent feature branch and merge it non-fast-forward,
    then resume Slice 19.4; no serial or RF path was used.
