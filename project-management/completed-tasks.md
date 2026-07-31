# Completed Tasks

Record completed work with newest entries at the top. Use dated bullets and
include concise outcomes, owners, ISO 8601 completion timestamps, validation
evidence, important decisions or risk acceptances, and follow-up records.

- 2026-07-30 — `EXPL-TASK-017` — Deliver Actions and the shared activation
  foundation required by Menus.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T19:50:00-07:00
  - Completed: 2026-07-30T20:19:43-07:00
  - Outcome:
    - delivered copy-safe non-container `Button` and `HotkeyBar` controls,
      copied ordered `HotkeyBarItem` values, automatic Button minima, and
      bounded aggregate Action resources;
    - made canonical label, enabled/disabled reason, and checked state one
      App command definition reused by controls, bindings, raw keys, direct
      commands, and the following Menu phase;
    - delivered deterministic Button focus, Tab/Shift-Tab traversal,
      Enter/Space pressed capture and activation, default/cancel roles, Label
      and Button Alt mnemonics, focus repair, and source-local reset;
    - added generic focus plus bounded typed Action and HotkeyBar details to
      core and explicitly projected automation snapshots; and
    - expanded the public-API demo to 19 controls and ten Layouts with focused,
      ordinary, disabled, default, cancel, checked, mnemonic, and structured
      shortcut evidence.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY lifecycle test, the full race suite, every debug,
      release, and profiling build, executable checks, and smoke/self-checks;
    - the display/command-label normalization fuzzer executed 323,473 cases
      in three seconds, and the complete package set compiled for CGO-free
      Linux arm64;
    - the conservative response proof measured 35,182,459 JSON bytes and
      105,547,377 bytes for three retained maxima, below the 36 MiB and
      128 MiB bounds; and
    - an attached debug session observed all Action states with no overflow,
      raw pressed capture, Tab/Enter and Alt mnemonic routing, checked/view
      updates, exact correlated completions, final shutdown, and socket
      cleanup.
  - Contracts:
    - [`Actions API v0`](../docs/specifications/actions-api-v0.md)
    - [`Foundational Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed as one routine feature without convening the Panel;
    - used one contract, one implementation loop, one live closed loop, and
      one full verification gate; and
    - recorded the multi-event controller-output opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 8 Menus is active immediately and reuses this command, focus,
      mnemonic, raw-key, snapshot, and automation foundation.

- 2026-07-30 — `EXPL-TASK-016` — Deliver the Text and Display control phase.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T19:12:00-07:00
  - Completed: 2026-07-30T19:47:44-07:00
  - Outcome:
    - delivered copy-safe, non-container `Label`, `StaticText`, `Separator`,
      and `Rule` controls with ordinary Control geometry, style, Layout,
      stacking, transaction, lifetime, and thread-safety behavior;
    - delivered deterministic start/center/end alignment, none/word/cell
      wrapping, horizontal/vertical divider forms, automatic intrinsic
      minima, mutable text, and observable Label target/mnemonic association;
    - applied the one-cell Unicode policy before measurement and painting,
      preserving composed cells and replacing unsupported widths with one
      `U+FFFD` cell;
    - added kind-consistent typed core and automation details with deep-copy
      projection and client validation; and
    - added all four controls to the public-API demo through child BoxLayouts,
      with stable attached-automation evidence and synchronized accent-state
      changes.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY lifecycle test, the full race suite, every debug,
      release, and profiling build, executable checks, and smoke/self-checks;
    - the display normalization fuzzer executed 180,425 cases in three
      seconds, and the complete package set cross-compiled for CGO-free Linux
      arm64;
    - the conservative response proof measured 28,388,838 JSON bytes and
      85,166,514 bytes for three retained maxima, below the 36 MiB and
      128 MiB bounds; and
    - an attached debug session observed 14 controls, eight Layouts, all four
      typed display-control records, no overflow, a correlated final
      shutdown, and owned-socket cleanup.
  - Contracts:
    - [`Text And Display API v0`](../docs/specifications/text-and-display-api-v0.md)
    - [`Foundational Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed as one routine feature task without convening the Panel; and
    - bounded frame-heavy integration diagnostics and recorded the remaining
      concise-controller-output opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - mnemonic activation proceeds with Actions;
    - Menus now follow Actions immediately and will provide
      purpose-specific `expletives-test` screen navigation.

- 2026-07-30 — `EXPL-TASK-015` — Deliver scalable root geometry and
  independent control/Layout borders.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T18:23:06-07:00
  - Completed: 2026-07-30T19:08:43-07:00
  - Outcome:
    - removed the 240 by 120 policy cap in favor of a 4,194,304 aggregate
      frame-cell allocation bound and cost-based snapshot retention;
    - made the root fill the offered surface by default and added atomic
      optional minimum, maximum, and terminal-cell aspect-ratio constraints;
    - added independent none, single, double, light/medium/dark shade, and
      full-block borders to Frame/GroupBox controls and Layouts, including
      semantic styles and optional per-component color overrides;
    - made Layout borders participate in measurement, arrangement inset,
      snapshots, automation projection, and Layout subtree stacking, allowing
      adjacent unbordered Frames inside one clean bordered Layout;
    - added Unicode, DEC Special Graphics, and ASCII physical border
      projection without changing canonical snapshots; and
    - added compact bounded cell runs to the automation wire while preserving
      the expanded `snapshot.frame.cells` view in the reusable client and
      `expletivesctl`.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY lifecycle test, the full race suite, all debug,
      release, and profiling builds, executable checks, and smoke/self-checks;
    - a 1200 by 1200 App frame allocated 1,440,000 canonical cells, compacted
      to one wire run, decoded, validated, and re-expanded exactly;
    - the worst-case response proof measured 27,647,462 JSON bytes and
      82,942,386 bytes for three retained maxima, below the 36 MiB and
      128 MiB bounds;
    - attached `expletivesctl` inspection observed a centered 64 by 20 root
      inside a 68 by 22 physical surface, 1,496 expanded cells, 255 compact
      runs, every selected control/Layout border form, no overflow, final
      shutdown, and owned-socket cleanup; and
    - the post-verification strengthened Layout color-override regression and
      complete Go suite passed.
  - Decisions and contracts:
    - [`EXPL-DEC-012`](decision-log.md#expl-dec-012--use-allocation-based-sizing-and-independent-borders)
    - [`Root Sizing And Border Contract v0`](../docs/specifications/root-sizing-and-borders-v0.md)
    - [`Foundational Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed as one routine feature task without convening the Panel;
    - recorded compact-transport, single-verification-loop, and
      specification-drift improvements in the
      [`process efficiency log`](reviews/process-efficiency-log.md).

- 2026-07-30 — `EXPL-TASK-009` — Deliver Basic Layouts and deterministic
  stacking.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T17:30:27-07:00
  - Completed: 2026-07-30T18:04:47-07:00
  - Outcome:
    - delivered independently constructed horizontal/vertical `BoxLayout` and
      row-major `GridLayout` objects, atomic transaction attachment, nested
      Layout trees, stable minima/grow/alignment geometry, clipping, resize,
      destruction cleanup, and immutable Layout snapshots;
    - delivered Panel `Raise`/`Lower` among Panel peers and Layout
      `Raise`/`Lower` among Layout peers, preserving arrangement, control
      parentage, and the other peer kind's stack slots;
    - delivered bounded observable overflow episodes, one replaceable
      cancellation-aware application handler, panic/timeout/saturation
      fallback, black-on-yellow compact/tiny warning presentation,
      acknowledgement, recovery, and recurrence;
    - projected Layouts and stack indices into the validated automation-owned
      DTO, with a new conservative 36 MiB response and three-result/128 MiB
      aggregate proof; and
    - replaced the demo's absolute fixture with the `layouts.basic` public-API
      consumer using Box, Grid, nesting, child-owned Layout, and visible
      top-level Layout stacking.
  - Verification:
    - `make verify` passed formatting, vet, all ordinary and Unix-socket
      tests, the controlling-PTY test, the full race suite, all three build
      modes, executable checks, and smoke/self-check runs;
    - the bounded Box property fuzzer executed 48,960 cases in two seconds;
    - the response proof measured a 34,681,823-byte legal maximum and
      104,045,469 bytes for three retained maxima, below their fixed bounds;
      and
    - attached `expletivesctl` observation verified five correctly sized
      Layouts, no overflow, fixed Grid rectangles across Panel lowering, a
      green-to-red frame-cell owner/color change after raising the red Layout,
      correlated shutdown, and socket cleanup.
  - Decisions and contracts:
    - [`EXPL-DEC-011`](decision-log.md#expl-dec-011--define-layout-nesting-and-stacking)
    - [`Foundational Layout API v0`](../docs/specifications/layout-api-v0.md)
  - Follow-up:
    - Layout replacement, detachment, spacers, arbitrary foreign item kinds,
      and control reparenting remain deferred; and
    - the next common-control phase remains under
      [`EXPL-TASK-010`](backlog.md).

- 2026-07-30 — `EXPL-TASK-014` — Deliver the first runnable
  Core/Presentation/Automation vertical slice.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-24T11:17:00-07:00
  - Completed: 2026-07-30T06:00:24-07:00
  - Outcome:
    - delivered the public Go toolkit foundation with App ownership, copy-safe
      `Panel`, `Frame`, and `GroupBox` handles, semantic themes, atomic
      transactions and snapshots, bounded command/input routing, and the
      one-cell Unicode contract;
    - delivered the CGO-free Linux terminal presenter, reusable incremental
      input decoder, human-runnable colored-Panel `expletives-test`, explicit
      bounded automation server/client, and `expletivesctl`;
    - reconciled the public Go and automation contracts and resolved every
      confirmed `EXPL-REV-001` finding without a second formal Panel; and
    - closed the implementation scope previously tracked as
      `EXPL-TASK-003`, `EXPL-TASK-007`, and `EXPL-TASK-008`.
  - Verification:
    - `make verify` passed formatting, vet, ordinary tests, the controlling-PTY
      process test, the full race suite, all three build modes, and smoke
      checks;
    - `make clean` and a fresh `make all` passed, leaving both executables at
      every required `build/<mode>/<name>` path;
    - both commands cross-built with `CGO_ENABLED=0` for Linux arm64;
    - three-second decoder fuzz smokes passed 87,230 automation and 126,476
      terminal executions; and
    - a fresh external session verified strict Hello negotiation, green Panel
      geometry, a raw Control-R chord to magenta, direct reset to green,
      correlated final shutdown, process exit, and socket cleanup.
  - Decisions and evidence:
    - [`EXPL-DEC-008`](decision-log.md#expl-dec-008--select-the-first-runnable-go-terminal-and-automation-baseline)
    - [`EXPL-DEC-010`](decision-log.md#expl-dec-010--reserve-the-panel-for-major-issues)
    - [`EXPL-REV-001 closure`](reviews/decisions/EXPL-REV-001-design-evidence.md)
  - Follow-up:
    - Basic Layouts were subsequently completed under
      [`EXPL-TASK-009`](#expl-task-009--deliver-basic-layouts-and-deterministic-stacking);
      and
    - broader interrupt, menu, focus, terminal-profile, and extension choices
      remain under [`EXPL-TASK-002`](backlog.md).

- 2026-07-24 — `EXPL-TASK-013` — Reconcile the approved Basic snapshot,
  Layout attachment, and overflow-notification contracts.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-24T10:44:56-07:00
  - Completed: 2026-07-24T11:09:37-07:00
  - Outcome:
    - fixed the Basic `SnapshotV1` cell record as one canonical one-cell
      grapheme, semantic style, resolved foreground and background colors, and
      stable owner identity, with cursor and the bounded typed control tree at
      snapshot level and no width or continuation fields;
    - specified independent Layout construction and atomic
      `Panel.SetLayout`, including a fully unchanged Panel, Layout, control
      tree, logical geometry, effective clips, and frame after failed
      attachment;
    - distinguished preserved logical Layout rectangles from effective output
      clipped by every ancestor and the application surface;
    - defined observable overflow episodes, exactly one applicable handler
      delivery attempt or initial default-notification attempt per episode,
      and deficit updates without callback storms;
    - defined bounded, cancellation/deadline-aware callback dispatch outside
      Layout, rendering, presentation, UI, and internal locks, with no
      unbounded replacement goroutines;
    - defined the nonblocking fallback ladder: compact dismissible `OK`
      warning, tiny-terminal high-visibility indicator, or zero/headless
      semantic evidence; and
    - made the overflow-producing request complete after the structured fact
      and notification-queued snapshot, with callback, disposition, fallback,
      and acknowledgement represented by later sequenced snapshots.
  - Validation:
    - an independent targeted audit passed all approved SnapshotV1, attachment,
      clipping, callback-attempt, fallback, and automation-completion
      consistency checks;
    - checked 55 project-owned Markdown files for balanced fences, valid
      relative link targets, and valid heading anchors;
    - checked project-owned documentation for trailing whitespace;
    - ran `git diff --check`; and
    - ran `python3 FieldManual/bootstrap.py --project-root . --dry-run`, which
      passed with zero planned writes.
    - No Go module, Go source, or Makefile exists yet, so implementation builds
      and Go tests were not applicable to this documentation task.
  - Decision:
    - [`EXPL-DEC-007`](decision-log.md#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)
  - Follow-up:
    - exact Overflow API names, bounds, fallback presentation, and dismissal
      were subsequently completed under
      [`EXPL-TASK-009`](#expl-task-009--deliver-basic-layouts-and-deterministic-stacking);
      and
    - [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
      has no remaining critical operator decision.

- 2026-07-24 — `EXPL-TASK-011` — Incorporate the foundational operator
  decisions, multithreaded application requirement, and Limited Unicode
  clarifications.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-24 (exact start time not recorded)
  - Completed: 2026-07-24T08:59:36-07:00
  - Outcome:
    - approved the App-owned `app.Root()` Panel, mandatory container-capable
      construction parents, immutable foundational ownership, and deferred
      reparenting;
    - specified one-shot raw logical `KeyPress`, modifier lifecycle chords,
      thread-safe multithreaded MVC/MVVC consumption, and serialized toolkit
      owner boundaries;
    - replaced active Sizer terminology with instantiated Panel-only
      `BoxLayout` and `GridLayout` objects;
    - made `expletivesctl` and every current or future executable part of the
      debug, release, and profiling inventories;
    - limited the canonical display model to complete grapheme clusters that
      occupy exactly one cell and required one logical `U+FFFD` cell for every
      unsupported display element;
    - specified conservative basic-terminal presentation: direct definite
      ASCII or known code-page mappings, otherwise one black-on-yellow ASCII
      approximation or `?`, without mutating the canonical frame; and
    - reconciled the roadmap, proposal, specifications, decisions, open
      questions, human requests, and project instructions.
  - Validation:
    - independently audited all active Unicode contracts and their distinction
      between canonical and physical presentation;
    - checked 54 project-owned Markdown files for relative link targets and
      heading anchors;
    - checked project-owned Markdown for trailing whitespace and balanced code
      fences;
    - ran `git diff --check`; and
    - ran `python3 FieldManual/bootstrap.py --project-root . --dry-run`, which
      passed with zero planned writes.
    - No Go module, Go source, or Makefile exists yet, so implementation builds
      and Go tests were not applicable to this documentation task.
  - Decisions:
    - [`EXPL-DEC-004`](decision-log.md#expl-dec-004--root-input-concurrency-layout-and-build-foundations)
    - [`EXPL-DEC-005`](decision-log.md#expl-dec-005--limit-displayed-unicode-to-one-cell)
    - [`EXPL-DEC-006`](decision-log.md#expl-dec-006--degrade-unicode-conservatively-on-basic-terminals)
  - Follow-up:
    - [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
      retains the remaining critical foundation choices.
    - [`EXPL-TASK-012`](backlog.md) holds reparenting for later design.

- 2026-07-24 — `EXPL-TASK-005` — Design the phased common-control delivery,
  foundational automation/layout slice, and MVC-compatible application
  boundary.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-24 (exact start time not recorded)
  - Completed: 2026-07-24T07:22:06-07:00
  - Directed scope:

    > After Core/container, add Basic Presentation, Basic Automation, and
    > Basic Sizers; defer Structured Input; make every ordinary control take a
    > parent; and use the colored-Panel scene as the first closed-loop fixture.

    > Automation is allowed to provide `KeyDown`, `KeyUp`, and `KeyPress` so
    > modifier chords can be created.

    > Support MVC or a similar application structure for consuming projects.

  - Outcome:
    - defined the complete directed control catalog and exact delivery order;
    - specified app-owned root/parent invariants, the initial colored-Panel
      presentation scene, raw key lifecycle automation over
      `--automation <socket-path>`, and non-control BoxSizer/GridSizer layout;
    - recorded Structured Input as deferred work;
    - specified toolkit-independent models, control-tree views, structured
      controller/update paths, and UI-owner marshaling for MVC-like consumers;
    - created under-review `EXPL-PROP-001` with alternatives, risks,
      validation, milestones, and critical decisions;
    - split the implementation queue into Core/container, Basic Presentation,
      Basic Automation, Basic Sizers, and the remaining ordered controls; and
    - recorded directed decisions `EXPL-DEC-002` and `EXPL-DEC-003`.
  - Validation:
    - audited all project documentation for raw-key, automation invocation,
      Structured Input, root, sizer, and MVC consistency;
    - checked trailing whitespace and fenced-block balance;
    - verified relative Markdown files and heading anchors;
    - ran `git diff --check`; and
    - ran `python3 FieldManual/bootstrap.py --project-root . --dry-run`, which
      passed with zero planned writes.
  - Follow-up:
    - [`EXPL-REQ-001`](ai-human-requests.md) requests operator decisions on
      the critical proposal questions.
    - [`EXPL-TASK-002`](backlog.md) remains the active foundational-contract
      backlog item; implementation starts with `EXPL-TASK-003` after its
      prerequisites are approved.

- 2026-07-24 — `EXPL-TASK-001` — Orient to the project and establish its
  durable product/documentation baseline.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-24 (exact start time not recorded)
  - Completed: 2026-07-24T03:19:34-07:00
  - Original request and clarifications:

    > Orient yourself to the project, FieldManual, and Ubersight; begin
    > publishing updates; update `AGENTS.md` and create documentation needed
    > to track the work and state the goals.

    Subsequent direct clarifications established `expletives-test`, attached
    automation, ordinary Go tests, build modes and paths, the root Makefile
    interface, keyboard/interrupt requirements, comparative toolkit research,
    curses/terminfo context, and initial unauthenticated automation.

  - Outcome:
    - added project-specific `AGENTS.md` instructions;
    - recorded primary/secondary products and engineering goals;
    - specified `expletives-test`, attached automation, closed-loop diagnosis,
      configurable Ctrl-C, ordinary Go tests, build modes, artifact paths, and
      Make targets;
    - preserved the older toolkit document as Draft design input with explicit
      unresolved conflicts;
    - recorded comparative Win32, wxWidgets, Motif, LessTif, Turbo Vision,
      tcell/tview/Bubble Tea, curses, ncurses, and terminfo lessons;
    - added the roadmap, active questions, implementation backlog, decision
      `EXPL-DEC-001`, and deferred authentication work `EXPL-TASK-004`;
    - excluded generated `build/` artifacts from version control; and
    - published meaningful Ubersight transitions through the installed atomic
      writer.
  - Validation:
    - confirmed project and FieldManual roots and clean submodule revision;
    - read the required core, Go, terminal, Turbo Vision, automation, and
      Ubersight guidance;
    - checked all changed documentation for trailing whitespace and balanced
      fenced blocks;
    - verified relative Markdown link targets;
    - ran `git diff --check`; and
    - ran `python3 FieldManual/bootstrap.py --project-root . --dry-run`, which
      passed with zero planned writes.
  - Decisions and risk:
    - [`EXPL-DEC-001`](decision-log.md#expl-dec-001--product-test-automation-and-build-direction)
      records the directed product/build/test baseline.
    - Initial `--automation <socket-path>` may be unauthenticated only in an
      operator-controlled, non-risky context. Hardened authentication and
      capability authorization are intentionally deferred under
      [`EXPL-TASK-004`](deferred.md).
  - Follow-up:
    - [`EXPL-TASK-002`](backlog.md) reconciles foundational contracts.
    - [`EXPL-TASK-003`](backlog.md) implements the first Go/Makefile/test
      vertical slice.
