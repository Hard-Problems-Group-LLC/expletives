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
  - Blockers: `EXPL-REQ-004` requires an operator-attached physical terminal,
    GNU Screen, and a real supported remote transport before the Phase 19
    acceptance gate can close.
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
  - TextField maximum-byte consumer repair (started
    2026-08-07T17:09:22-07:00): radioradio ECR 2026-004 requests a hard
    233-byte composer ceiling. Phase 19 Slice 19.7 adds one reusable
    zero-compatible per-field UTF-8 maximum under `EXPL-BUG-004`; excess key
    or committed input is an atomic no-op with no edit command. Backspace,
    Left, Delete, and Enter remain usable at capacity. Update typed evidence,
    automation validation, documentation, public-consumer coverage, and the
    deterministic catalog before the full gate and permanent feature-branch
    publication. Slice 19.4 is paused; fixtures are synthetic only.
  - TextField maximum-byte consumer repair (completed
    2026-08-07T17:23:42-07:00): the effective copied maximum now gates every
    constructor, setter, key, selection replacement, committed-text, and
    paste path atomically while retaining deletion, navigation, cancellation,
    and submission. Core/automation details and the catalog expose the bound;
    malformed-client, public-consumer, ASCII, multi-byte, exact-limit, and
    recovery coverage pass. The complete ordinary, PTY, race, build-mode,
    self-check, and smoke gate is green. Publish and non-fast-forward merge
    the permanent feature branch, then resume Slice 19.4.
  - Consumer ECR intake (completed 2026-08-08T10:31:25-07:00): the operator
    directed target-side handling of radioradio ECRs 2026-001, 2026-005, and
    2026-006. Slice 19.4 is paused while these bounded repairs are active.
    Slice 19.8 corrects the default disabled-Button foreground/background
    collision under `EXPL-BUG-005`. Slice 19.9 establishes the shared closed-
    interval viewport invariant under `EXPL-BUG-006`. Slice 19.10 adds the
    construction-time non-modal input-scope boundary directed by
    `EXPL-DEC-015`, with zero-value compatibility, scope-local mnemonics and
    roles, explicit Tab confinement/escape, atomic visibility/focus repair,
    and typed core/automation evidence. Slices 19.8 through 19.10 are
    complete. `make verify` passed formatting, vet, ordinary and Unix-socket
    tests, PTY integration, the complete race suite, all three build modes,
    catalog self-checks, and smoke tests. The source ECR files and dirty
    radioradio worktree remained read-only and outside target-side mutation.
    Resume Slice 19.4 physical terminal-matrix acquisition.
  - Dialog contrast variance (completed 2026-08-08T11:22:37-07:00): Slice
    19.11 implements the operator-approved and endorsed permanent variance in
    `EXPL-DEC-016`: default Dialog body and static-message text is white rather
    than Turbo Vision black on the established medium-gray `#808080` surface.
    Live attached acceptance reopened the first candidate under
    `EXPL-BUG-007` because the maintained catalog's complete custom Theme
    duplicated the old black body roles. The repaired library and public
    consumer now agree, exact Theme and rendered-cell regressions pass, and
    `make verify` passed the complete gate. A rebuilt temporary attached
    instance proved `#FFFFFF` on `#808080` at frame 76 and shut down cleanly.
    The operator's original active process remains untouched and requires a
    restart to load the repaired executable. Resume Slice 19.4.
  - Standard-dialog button spacing (completed
    2026-08-08T11:46:06-07:00): Slice 19.12 moved the reserved blank interior
    row below bottom Button sections immediately above them while preserving
    dialog dimensions, two-row raised Button geometry, horizontal spacing,
    focus, and shadow roles. Exact MessageBox, ConfirmDialog, InputDialog, and
    cancellable ProgressDialog geometry coverage passes. `make verify` passed
    the complete ordinary, Unix-socket, PTY, race, build-mode, catalog self-
    check, and smoke gate. A rebuilt attached MessageBox retained its 126 by 8
    bounds, moved OK from Y 8 to Y 9, exposed two blank body rows above it,
    and placed the bottom border immediately below it at frame 76. Resume
    Slice 19.4. The operator then restarted and accepted the maintained live
    instance; final ACP inspection at frame 84 reconfirmed the white body,
    revised spacing, and immediate bottom border, and frame 85 restored the
    session.
  - Physical-matrix resync (2026-08-08): current `main` at
    `5e8279c0d54c` retains the full green `make verify` gate from ACP. The
    exact committed revision also completed all four terminal benchmark
    workloads and ten-second bounded fuzz runs for the input decoder and
    terminfo parser without failure. The local agent runner has no controlling
    TTY; tmux 3.4 and an SSH client are present, while xterm, GNU Screen, and
    mosh are absent. The XTerm/tmux/conservative-locale rows from
    `778518ef251d` remain useful historical observations but cannot certify
    the changed current product tree. Slice 19.4 remains active and is waiting
    on the current-revision physical rows in `EXPL-REQ-004`.
  - Current live attached row (2026-08-08): the operator-owned release-mode
    instance reports product revision `5e8279c0d54c` and runs with a
    controlling PTY under Terminator 2.1.5/VTE 0.78.6, the
    `xterm-256color` profile, UTF-8 locale, and 137x19 geometry. Attached
    automation drove the complete canonical style screen, revised MessageBox,
    Alt-F and F9 menu paths, reversible TextField editing, bounded replacement
    cells, and Markdown scrolling with explicit outcomes, then restored clean
    Home frame 110. This closes the current native-profile attached
    application/controller evidence. The operator confirmed that the driven
    frontend rendering and physical keyboard behavior appeared correct.
    Bracketed paste, resize, Ctrl-C, suspend/resume, and teardown remain, along
    with current tmux, conservative-locale, GNU Screen, and remote rows under
    `EXPL-REQ-004`.
  - Evidence ACP and exact verification (2026-08-08): commit
    `098195361588` records the current native attached/operator row and is
    synchronized on `origin/main` with the required author and committer
    identity. A clean post-ACP `make verify` passed the ordinary, Unix-socket,
    PTY lifecycle, race, debug/release/profiling build, catalog self-check, and
    smoke gates. The release `expletives-test` and debug `expletivesctl`
    binaries both report `098195361588` with `modified=false`. The next live
    automatable acquisition is the conservative `C`-locale row in the same
    Terminator profile; user-visible fallback rendering still requires the
    operator-owned instance.
  - Automation command-inventory consumer repair (started
    2026-08-09T10:19:22-07:00): radioradio's live acceptance exposed
    `EXPL-BUG-008`. The server and protocol specification permit up to
    `MaxControls` advertised automation commands, but the reference client
    rejects more than 64 during Hello validation. Align the client with the
    existing normative and server bound, prove exact-bound acceptance and
    overflow rejection, run the project gate, and publish through a permanent
    feature branch before resuming downstream acceptance. No UI command or
    radio operation was issued.
  - Automation command-inventory consumer repair (completed
    2026-08-09T10:26:55-07:00): client Hello validation now uses the same
    4,096-command `MaxControls` ceiling as the server and normative protocol.
    Exact-bound acceptance and one-over rejection are covered. The complete
    vet, ordinary, Unix-socket, PTY, race, debug, release, and profiling gate
    passes. Publish and merge the permanent feature branch, then update the
    downstream module pin and resume receive-only live acceptance.
  - Locale projection comparison (completed 2026-08-10): clean release
    `098195361588` ran at 137x19 under the same Terminator 2.1.5/VTE 0.78.6
    `xterm-256color` profile with conservative `LC_ALL=C` and ordinary
    `LANG=en_US.UTF-8` environments. The operator accepted both physical
    presentations: the conservative row used DEC closest-line art and aligned
    structural `#` fallbacks, while UTF-8 retained distinct single, double,
    light, medium, dark, and full-cell Unicode forms. Canonical attached
    evidence remained distinct in both cases, and the UTF-8 instance returned
    to clean Home frame 85. The locale structural-rendering block is complete;
    current-revision ordinary black-on-yellow content fallback remains open
    because the `C` instance restarted before that fixture, alongside the
    remaining interaction and external-profile rows in `EXPL-REQ-004`.
  - Dialog contrast/spacing branch audit (completed 2026-08-10): fetched and
    pruned every configured remote; this checkout has only `origin`. Commit
    `5e8279c0d54c` contains the complete Slice 19.11 and 19.12 implementation,
    regression tests, maintained-catalog Theme repair, approved variance,
    resolved defect, specifications, and acceptance records, and is reachable
    from local and `origin/main`. No commit outside `main` across local or
    remote branches, linked worktrees, stashes, reflog-only history, or
    unreachable objects changes the relevant Theme, standard-dialog, catalog,
    modal specification, decision, or defect paths. No patch, merge, or
    cherry-pick is missing.
