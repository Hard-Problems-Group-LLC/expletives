# Completed Tasks

Record completed work with newest entries at the top. Use dated bullets and
include concise outcomes, owners, ISO 8601 completion timestamps, validation
evidence, important decisions or risk acceptances, and follow-up records.

- 2026-08-02 — `EXPL-TASK-033` — Deliver Phase 18 file and directory pickers.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-08-02T00:55:00-07:00
  - Completed: 2026-08-02T02:08:22-07:00
  - Outcome:
    - added bounded application-supplied provider contracts and a confined
      local-filesystem adapter with copied entries, sorting, filtering,
      navigation, refresh, validation, and explicit error state;
    - delivered Turbo Vision-style `FilePickerDialog`,
      `MultiFilePickerDialog`, and `DirectoryPickerDialog` compounds with
      typed results, explicit close outcomes, small-screen minima, focus and
      keyboard behavior, and no control-owned filesystem goroutines;
    - added all three deterministic Dialogs-menu demonstrations, typed core
      and privacy-compacted automation evidence, live current-item updates,
      direct public activation seams, and complete self-check coverage; and
    - separated the dialog surface from menu gray: dialogs now render on the
      operator-selected darker `#808080` surface while menus remain
      `#AAAAAA`, with white dialog borders/titles and matching button-shadow
      backing.
  - Verification:
    - `make verify` passed format, vet, unit and integration tests, the full
      race suite, debug/release/profiling builds, and each mode's fail-fast
      smoke/self-check;
    - live attached automation at 100x30 verified all three picker variants,
      zero Layout overflow, keyboard-driven information changes, exact typed
      details, and clean shutdown;
    - exact frame cells proved Menu `#AAAAAA`, dialog body/border/button-shadow
      backgrounds `#808080`, black body text, and white border/title text; and
    - the conservative maximum completion remains 41,391,622 bytes, with
      three retained maxima totaling 124,174,866 bytes below the 128 MiB
      aggregate evidence budget.
  - Contracts:
    - [`File And Directory Pickers API v0`](../docs/specifications/file-pickers-api-v0.md)
    - [`File And Directory Pickers`](../docs/File-Pickers.md)
    - [`Modals API v0`](../docs/specifications/modals-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed without convening the Panel; and
    - made `make smoke` fail fast so an early build mode cannot be masked by a
      later successful command.
  - Follow-up:
    - Phase 19 Terminal Compatibility and Operational Hardening is active.

- 2026-07-31 — `EXPL-TASK-030` — Deliver Phase 15 Scrolling and Content.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T04:35:20-07:00
  - Completed: 2026-07-31T09:34:16-07:00
  - Outcome:
    - added public `Viewport` and `ScrollablePanel` ownership, clipping,
      bounded offset, integrated-scrollbar, focus, Layout-hint, direct
      mutation, and atomic Transaction APIs;
    - added bounded deterministic `MarkdownView`, structured `LogView`, and
      raw-chunk `StreamView` controls with exact retained/pending/drop state,
      follow versus scrollback, stable record keys, semantic styles, and no
      control-owned I/O or goroutines;
    - made accumulative content mutation context-aware and safe for concurrent
      producers while allocating its serialization gate only on LogView and
      StreamView controls;
    - neutralized invalid UTF-8, control/escape traffic, and unsupported-width
      stream input; retained accurate source-byte loss counts and visible
      truncation/drop warnings within one shared bounded content budget;
    - extended core and automation snapshots with compact kind-specific typed
      evidence, deep copies, fail-closed client validation, aggregate content
      accounting, and a conservative maximum-response proof; and
    - enabled the Scrolling / Content catalog with four focus groups,
      deterministic append/follow/reset commands, raw scroll keys, semantic
      frame evidence, and human-readable follow/scrollback fixtures.
  - Verification:
    - `make verify` passed formatting, vet, all unit and Unix-socket tests,
      the controlling-PTY lifecycle, the complete race suite, debug/release/
      profiling builds, and every build mode's smoke/self-check;
    - live attached automation at 80x24 verified four typed content controls,
      visible Markdown/log/stream semantic styling, replacement and
      truncation cells, Page Up leaving follow mode, End resuming it,
      deterministic append/follow behavior, exact reset, and clean shutdown;
    - the conservative maximum completion is 41,391,622 bytes, with three
      retained maxima totaling 124,174,866 bytes below the 128 MiB aggregate
      evidence budget; and
    - the real PTY test sent Alt-X, observed `app.quit`, and confirmed exact
      terminal restoration; a focused regression also proves that the global
      binding cannot be shadowed by a local control mnemonic.
  - Contracts:
    - [`Scrolling and Content API v0`](../docs/specifications/scrolling-content-api-v0.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed without convening the Panel; and
    - recorded bounded-content synchronization and typed-detail maintenance
      opportunities in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 16 Collections is next.

- 2026-07-31 — `EXPL-TASK-029` — Deliver Phase 14 Navigation and Chrome.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T03:47:00-07:00
  - Completed: 2026-07-31T04:32:46-07:00
  - Outcome:
    - added focusable horizontal/vertical `ScrollBar` with copied viewport
      state, exact track/thumb geometry, configurable arrow/page steps,
      clamped keyboard movement, and optional user-only change routing;
    - added distinct `TabbedPanel` and `Notebook` container kinds over copied
      ordered `Tab` descriptors and direct child Panel pages, with complete
      replacement/reorder/removal, destruction repair, nested page Layouts,
      scoped mnemonics, and separate current focus from selection;
    - derived selected-page visibility without overwriting caller visibility,
      made managed pages fill the tab-container client area, and exposed exact
      rendered/omitted/clipped tab-strip evidence;
    - extended Theme, focus guidance, local snapshot, automation projection,
      deep-copy, explicit kind validation, page-relationship validation, and
      response-bound accounting for all Navigation kinds; and
    - enabled Controls/Navigation in `expletives-test` with both ScrollBar
      orientations, disabled-tab and Notebook examples, Scenario Reset, raw
      key self-check, and attached automation coverage.
  - Verification:
    - `make verify` passed vet, ordinary and Unix-socket tests, the
      controlling-PTY lifecycle, the complete race suite, all debug/release/
      profiling builds, and smoke/self-check coverage;
    - the race gate found and then verified the fix for an unlocked base-
      behavior read shared by concurrent read/modify/write Transaction
      setters;
    - a fresh debug build passed live attached inspection at 100x30 with no
      Layout overflow: raw Right advanced Offset from 40 to 41, tab focus and
      selection remained distinct, and page visibility switched exactly; and
    - `git diff --check`, `make fmt-check`, and the FieldManual bootstrap dry
      run passed with zero planned writes.
  - Contracts:
    - [`Navigation and Chrome API v0`](../docs/specifications/navigation-chrome-api-v0.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed without convening the Panel; and
    - recorded contextual focus-kind and diagnostic improvements in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 15 Scrolling and Content is next.

- 2026-07-31 — `EXPL-TASK-028` — Deliver Phase 13 Progress controls.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T03:11:56-07:00
  - Completed: 2026-07-31T03:44:45-07:00
  - Outcome:
    - added non-focusable `ProgressBar`, horizontal/vertical `Meter`,
      `Spinner`, and `ActivityDots` controls with copied complete state,
      public/transaction constructors, atomic setters, and
      cancellation-aware `Update`;
    - implemented determinate/indeterminate rendering, exact integer ratios
      and percentages, narrow ASCII fallbacks, absolute animation ticks,
      reduced-motion canonicalization, terminal states, and semantic styles;
    - kept clocks, tickers, workers, model ordering, cancellation ownership,
      and terminal access outside the controls so multithreaded MVC/MVVC
      applications retain their own work lifecycle;
    - projected exact kind-consistent `ControlDetails.Progress` through the
      root snapshot and versioned automation DTO with trust-boundary
      validation and unchanged aggregate response bounds; and
    - enabled the Controls/Progress catalog page with six grouped
      demonstrations and deterministic Tick, Reset, Complete, Fail, Cancel,
      and Motion command paths.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY lifecycle, the complete race suite,
      debug/release/profiling builds, and every mode's smoke/self-check;
    - a fresh debug build passed direct attached socket inspection at
      100x30 with no Layout overflow, and Tick evidence advanced the bar,
      meters, indeterminate bar, Spinner, and ActivityDots atomically;
    - the conservative maximum completion remains 41,389,979 bytes, with
      three retained maxima totaling 124,169,937 bytes below the 128 MiB
      aggregate budget; and
    - `git diff --check` and the FieldManual bootstrap dry run passed with
      zero planned writes.
  - Contracts:
    - [`Progress API v0`](../docs/specifications/progress-api-v0.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed without convening the Panel; and
    - recorded the default-Theme overlay opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 14 Navigation and Chrome is next.

- 2026-07-31 — `EXPL-TASK-027` — Deliver Phase 12 Text and Numeric Input.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T01:53:48-07:00
  - Completed: 2026-07-31T03:10:36-07:00
  - Outcome:
    - added bounded `TextField`, `NumberField`, `SpinBox`, and multiline
      `TextArea` controls with atomic public/transaction mutation, grouped
      focus behavior, explicit editing, commit/cancel, and user-only change
      commands;
    - implemented soft/hard whitelist/blacklist validation, password masking
      and snapshot redaction, exact fixed-place numeric bounds/steps, shared
      Shift selection and Ctrl-A, multiline navigation/viewports, and
      no/word/cell wrapping;
    - added bounded committed-text and terminal bracketed-paste events that
      enter only an editing control, never command resolution, and discard an
      entire over-limit physical paste rather than exposing a prefix;
    - retained configurable Ctrl-C command handling while every editor is
      active and cleared held-key evidence on terminal outcomes; and
    - enabled the seven-control Input catalog page with raw-key attached
      automation coverage, typed core/wire details, and exact client-side
      trust-boundary validation.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY lifecycle, the complete race suite, debug/release/
      profiling builds, and every mode's smoke/self-check;
    - focused multiline fuzzing completed 83,496 executions without failure;
    - the conservative maximum completion remains 41,389,979 bytes below the
      40 MiB line limit, with three retained maxima totaling 124,169,937 bytes
      below the 128 MiB aggregate budget; and
    - all required executables are current and executable under
      `build/{debug,release,profiling}/`.
  - Contracts:
    - [`Text and Numeric Input API v0`](../docs/specifications/text-and-numeric-input-api-v0.md)
    - [`Text Input Validation and Passwords`](../docs/Text-Input-Validation-and-Passwords.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed without convening the Panel; and
    - recorded shared decoder/catalog verification and integration-test
      budgeting improvements in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 13 Progress controls are next; Structured Input and automation
      authentication remain deferred.

- 2026-07-31 — `EXPL-TASK-026` — Deliver Phase 11 Selection controls.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-31T00:13:13-07:00
  - Completed: 2026-07-31T01:37:02-07:00
  - Outcome:
    - added bounded typed Checkbox, RadioGroup/RadioButton, CycleField, and
      SelectField controls with stable values, disabled/empty cases, atomic
      mutations, outside-lock user-change commands, and exact local and
      automation evidence;
    - defined direct-parent focus groups: Tab/Shift-Tab cross groups, arrows
      move focus spatially without changing values, Radio Space/Enter selects,
      and Cycle/Select `[`/`]` changes values;
    - added dynamic `FocusGuideBar` guidance with per-control application
      append/override text;
    - enabled the Selection catalog page, moved application chrome controls
      into the top-level Sections menu, and formalized Application versus
      Panel Client Area; and
    - removed duplicated phase/slice numbers from Ubersight titles so the
      dashboard renders each project ID once.
  - Verification:
    - ordinary and Unix-socket Go tests, the full race suite, response-bound
      proof, `make all`, and `make smoke` pass;
    - all debug, release, and profiling binaries exist in their required
      mode-specific paths; and
    - a live 100x24 debug automation run proved focus-only Radio arrows,
      Radio Enter selection, grouped Tab/Shift-Tab, `]` Cycle/Select changes,
      typed focus guidance, complete rendering, final shutdown, and socket
      cleanup.
  - Contracts:
    - [`Selection API v0`](../docs/specifications/selection-api-v0.md)
    - [`FocusGuideBar API v0`](../docs/specifications/focus-guide-bar-api-v0.md)
    - [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed without convening the Panel; and
    - recorded the Ubersight numbering and shared focus-navigation efficiency
      improvements in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 12 Text/Numeric Input is next; Structured Input remains deferred
      only where the control catalog explicitly says so.

- 2026-07-31 — `EXPL-TASK-025` — Deliver Phase 10 Headers and Footers.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T23:55:48-07:00
  - Completed: 2026-07-31T00:11:03-07:00
  - Outcome:
    - added root-owned `Header` and `Footer` one-row Containers with derived
      full-physical-width geometry outside root constraints;
    - implemented stable top-down Header ordering and oldest-nearest /
      newest-highest Footer ordering across resize, hide/show, destroy, and
      combined MenuBar/StatusBar geometry;
    - admitted horizontal Box, one-row Grid, and compatible nested Layout
      trees while atomically rejecting complete decorated minimum heights
      above one;
    - defined deterministic tiny-surface allocation with empty Bounds for
      excess bands and no conventional axis cap; and
    - extended Theme defaults, typed control kinds, core snapshots, and the
      validating public automation client without adding an unrestricted
      details bag.
  - Catalog:
    - enabled Controls/Headers / Footers and `screen.headers_footers`;
    - added two live Headers demonstrating horizontal Box and one-row Grid
      arrangements plus two Footers demonstrating construction order; and
    - hid all four bands on every other page, atomically returning their rows.
  - Corrective integration:
    - made Layout overflow actionable only while its owner and ancestors are
      effectively visible;
    - ended hidden-owner episodes while retaining deterministic arranged
      snapshot geometry, and began a new episode after re-show when needed;
    - suppressed false Layout warnings for chrome bands denied a row by the
      tiny-surface policy; and
    - documented and tested the shared lifecycle for later conditional views.
  - Verification:
    - `make verify` passed vet, all ordinary and Unix-socket tests, the
      controlling-PTY lifecycle, the full race suite, and debug, release, and
      profiling builds for `expletives-test` and `expletivesctl`;
    - focused Go tests cover root/geometry/Layout-item rejection, combined
      ordering, full-row painting, row return, constrained roots, tiny
      surfaces, horizontal Box and one-row Grid compatibility, width
      overflow, multirow atomic rejection, and owner/ancestor visibility
      episodes;
    - the exhaustive raw-key catalog audit passes every enabled, disabled,
      separator, root, and nested Menu entry; and
    - a live attached 64x20 debug run used raw Alt-C / `h` and Alt-I / `h`
      navigation to confirm exact Menu/Header/content/Footer/Status row order,
      child Layout geometry, empty hidden Bounds, returned Home rows, zero
      overflows, final shutdown, socket cleanup, and no retained temp path.
  - Contracts:
    - [`Headers and Footers API v0`](../docs/specifications/headers-footers-api-v0.md)
    - [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
    - [`Layouts and Overflow`](../docs/specifications/layouts-and-overflow.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed as a bounded common-control phase without convening the
      Panel; and
    - recorded the shared hidden-view overflow-lifecycle improvement in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 11 Selection is next, followed by the deferred-input boundary and
      Progress; and
    - the minimum-usable-geometry fallback remains backlogged.

- 2026-07-30 — `EXPL-TASK-024` — Deliver Phase 9 Status Bar.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T23:25:38-07:00
  - Completed: 2026-07-30T23:53:29-07:00
  - Outcome:
    - added the unique root-parented, non-container `StatusBar` on the complete
      physical bottom row, outside ordinary root Layouts and constraints;
    - added copied keyed static-context and command-derived segments, atomic
      replacement, dynamic shared command state, and deterministic
      priority/omission/clipping at narrow widths;
    - extended application chrome through resize, visibility, destruction,
      zero/tiny surfaces, and the documented one-row Main Menu priority;
    - added Turbo Vision light-gray, red-shortcut, and disabled palette roles;
      and
    - added bounded typed core and automation details with exact relative
      segment geometry and deep-copy guarantees.
  - Catalog:
    - added global `status.main` context and command hints plus the dedicated
      `screen.status` / `status.overview` page;
    - enabled Controls/Status Bar and included it in the exhaustive raw-key
      menu audit;
    - added collision-audited Alt-X Quit while retaining File/Quit, `q`,
      Escape, and the cancel Button; and
    - reduced screen-navigation command updates from every catalog page to
      only the previous and target checked commands.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY integration test, the full race suite, all
      debug/release/profiling builds, and every mode's smoke/self-check;
    - focused Go tests cover construction, parent/geometry/Layout rejection,
      uniqueness, command replacement/removal, priority, clipping, empty and
      tiny surfaces, row return, constrained roots, atomic rollback, snapshot
      copying, concurrent readers/writers, automation validation, response
      bounds, and catalog routes;
    - the conservative maximum completion remains below the fixed response
      limit at 37,496,731 bytes, with three retained maxima at 112,490,193
      bytes below the 128 MiB aggregate budget; and
    - live attached 100x30 and 18x5 debug runs confirmed full bottom-row
      ownership, Alt-C / `s` navigation, checked menu and contextual state,
      exact ` Home  Alt+X Quit ` narrow rendering, lower-priority omission,
      orderly final snapshots, socket cleanup, and no retained temp paths.
  - Contracts:
    - [`Status Bar API v0`](../docs/specifications/status-bar-api-v0.md)
    - [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
    - [`Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed as a bounded common-control phase without convening the
      Panel;
    - added the
      [`ControlDetails extension checklist`](../docs/Control-Details-Extension-Checklist.md);
      and
    - recorded typed-union and catalog-navigation improvements in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 10 Headers and Footers is next, followed by Selection; and
    - Structured Input and automation authentication remain deferred.

- 2026-07-30 — `EXPL-TASK-023` — Reorganize the interactive catalog and
  exhaustively audit its menus.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T23:07:47-07:00
  - Completed: 2026-07-30T23:24:04-07:00
  - Outcome:
    - made a childless medium-blue Home canvas the startup screen and added
      File/Home followed by the conventional separator before Quit;
    - replaced the combined Panel/Layout aliases with distinct Core Panels,
      Visual Styles, Box Layout, and Grid Layout screens;
    - added visible fixtures for all seven supported Frame forms, nested
      horizontal/vertical Box Layouts, common-mode Layout stacking, and a
      six-Panel Grid;
    - separated Panel and Layout stacking menus and made every mutation route
      navigate to the fixture it changes; and
    - retained stable raw commands, typed screen identity, checked menu state,
      and direct automation observability for every page.
  - Menu audit:
    - raw-key regression coverage traverses all 30 command-bearing menu
      entries, all eight separators, and all nine root or nested submenus;
    - enabled entries reach their exact registered command and visible page;
    - every disabled future entry remains selectable, exposes its phase-owned
      reason, and refuses Enter without dispatch or popup dismissal; and
    - attached debug checks exercised File/Home, nested Panel stacking, and a
      disabled Controls entry through raw KeyDown/KeyUp/KeyPress events.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY integration test, the full race suite, all
      debug/release/profiling builds, and every mode's smoke/self-check;
    - live 100x30 frame inspection confirmed the empty `#003878` Home canvas,
      seven Frame styles, nested Box geometry, stacked Layout geometry, and
      the complete three-by-two Grid; and
    - orderly attached shutdown returned a final snapshot, removed its socket,
      and left no temporary workspace.
  - Contracts:
    - [`expletives-test`](../docs/specifications/expletives-test.md)
    - [`Menus API v0`](../docs/specifications/menus-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
  - Process:
    - completed as a bounded catalog correction without convening the Panel;
    - ACP is authorized without review after the exhaustive audit gate; and
    - recorded the single-source catalog-manifest opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 9 Status Bar is the next active common-control phase, followed by
      Headers/Footers and Selection; and
    - Structured Input remains deferred.

- 2026-07-30 — `EXPL-TASK-022` — Audit terminal-emulator shortcut collisions
  and rebind project defaults.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T22:40:07-07:00
  - Completed: 2026-07-30T22:55:04-07:00
  - Outcome:
    - audited Terminator, Ptyxis, and GNOME Terminal shortcut defaults and
      mapped Ptyxis to current RHEL, Fedora, and Ubuntu desktop releases while
      retaining GNOME Terminal coverage for earlier, upgraded, and customized
      systems;
    - added the project shortcut-compatibility advisory, governing links, and
      `EXPL-DEC-014`, while preserving arbitrary client-selected raw chords;
    - changed MenuBar activation from F10 to F9, retained Ctrl-Space as a
      secondary fallback, and made reserved F10 a tested no-op;
    - changed the catalog roots to Alt-I/Alt-N/Alt-A/Alt-C/Alt-M/Alt-D/Alt-P
      and the Toggle/Reset Button mnemonics to Alt-G/Alt-R; and
    - corrected the controlling-PTY fixture to encode unshifted Alt-letter
      chords as lowercase legacy escape prefixes, avoiding the false
      `ESC P` DCS interpretation.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the controlling-PTY integration test, the full race suite, all
      debug/release/profiling builds, and all-mode smoke/self-checks;
    - focused core, public automation-client, and self-check coverage verifies
      the exact mnemonic catalog, Action mnemonics, F9 activation, F10 no-op,
      and typed red mnemonic cells; and
    - a live attached debug run observed F10 `no_op`, F9 bar activation,
      Alt-I File, Alt-A/Stacking nesting, Alt-P/About navigation, orderly final
      shutdown, socket cleanup, and temporary-workspace removal.
  - Contracts and decision:
    - [`Terminal Shortcut Compatibility Advisory`](../docs/Terminal-Shortcut-Compatibility.md)
    - [`Menus API v0`](../docs/specifications/menus-api-v0.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
    - [`EXPL-DEC-014`](decision-log.md#expl-dec-014--avoid-host-terminal-shortcut-collisions)
  - Process:
    - completed as a bounded compatibility correction without convening the
      Panel; and
    - recorded the compact closed-loop projection opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - future project-selected defaults must follow the advisory;
    - user customizations, desktop/input-method bindings, multiplexers, and
      remote clients still require deployment-local inspection; and
    - a built-in concise `expletivesctl` projection remains a process
      improvement, not part of this compatibility correction.

- 2026-07-30 — `EXPL-TASK-021` — Add end-aligned Help and scalable catalog
  Menu namespaces.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T21:35:00-07:00
  - Completed: 2026-07-30T21:56:10-07:00
  - Outcome:
    - added stable start/end placement for top-level MenuBar items, with
      declaration-order preservation inside each group, deterministic
      overlap painting, and popup anchoring at resolved positions;
    - added conventional right-aligned Help with an Alt-H-accessible About
      page built from existing controls;
    - reorganized the persistent catalog under File, Panels, Layouts,
      Controls, Menus, Dialogs, and Help, with distinct navigation labels and
      sensible separators;
    - routed existing core, text, action, menu, and About demonstrations
      through the catalog while leaving later control, dialog, panel-menu,
      context-menu, scrollbar, and layout demonstrations visible but disabled
      with explicit phase reasons; and
    - projected and validated placement through the typed core snapshot and
      automation protocol without permitting placement on popup entries.
  - Verification:
    - `make verify` passed vet, ordinary and Unix-socket tests, the
      controlling-PTY integration test, the full race suite, and debug,
      release, and profiling builds for both executables;
    - focused tests cover end-group geometry, mnemonic styling, right-edge
      popup anchoring, malformed and nested placement rejection, automation
      projection, and demo self-check labels;
    - the conservative response proof measured 37,255,187 JSON bytes and
      111,765,561 bytes for three retained maxima, below the 36 MiB and
      128 MiB bounds; and
    - a live 80x24 attached run observed the six start-aligned namespaces,
      end-aligned Help, raw Alt-H/A About navigation, grouped Controls entries
      and deferred reasons, final shutdown, and socket cleanup.
  - Contracts:
    - [`Menus API v0`](../docs/specifications/menus-api-v0.md)
    - [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed as a bounded Menu-phase extension without convening the
      Panel; and
    - ACP is authorized without review for this checkpoint.
  - Follow-up:
    - later phases enable their existing catalog entries when the required
      controls and behaviors exist;
    - Status Bar remains the next planned phase, followed by Headers and
      Footers; and
    - Structured Input remains deferred.

- 2026-07-30 — `EXPL-TASK-020` — Correct Menu chrome and Turbo Vision
  look-and-feel; reorder the next chrome phases.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T21:00:00-07:00
  - Completed: 2026-07-30T21:28:43-07:00
  - Outcome:
    - made the root-parented MenuBar immutable full-width physical-row
      application chrome, independent of centered/constrained root geometry
      and excluded from ordinary Layouts;
    - reserved visible top chrome from root content and returned the row
      atomically on hide/destroy;
    - implemented the Turbo Vision look-and-feel through native expletives
      rendering: black/light-gray menus, red mnemonics, green selection,
      disabled roles, single-line borders, tee separators, submenu
      indicators, checkmarks, and black shadows;
    - made F10/Ctrl-Space activate the menu row before Down/Enter opens a
      popup, retained direct Alt-root access, and made disabled entries
      selectable but not activatable;
    - measured popup widths from complete effective entries and backset child
      menus left when their preferred cascade would cross the right edge;
    - updated the demo, terminal glyph fallbacks, core and automation
      validators/tests, and durable Menu/application-chrome contracts; and
    - inserted Status Bar and then Headers/Footers immediately after Menus in
      the catalog and roadmap.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      controlling-PTY integration, the full race suite, all debug/release/
      profiling builds, and all-mode smoke/self-checks;
    - focused tests cover physical edge ownership, constrained roots, content
      reservation, geometry/Layout rejection, palette roles, disabled
      selection, measured child widths, right-edge backsetting, and structured
      terminal fallbacks; and
    - a live attached debug run observed full 80-column row-0 ownership,
      black/red-on-green F10 bar selection with an empty OpenPath, the File
      popup after Down, and a complete untruncated Actions/Stacking child
      popup before orderly final shutdown and socket cleanup.
  - Contracts and decision:
    - [`Application Chrome v0`](../docs/specifications/application-chrome-v0.md)
    - [`Menus API v0`](../docs/specifications/menus-api-v0.md)
    - [`EXPL-DEC-013`](decision-log.md#expl-dec-013--anchor-application-chrome-and-adopt-turbo-vision-menu-look)
  - Process:
    - completed as routine corrective work without convening the Panel;
    - used Turbo Vision only as a visual/behavioral reference; and
    - recorded the early-public-client semantic-state check in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - Phase 9 Status Bar is next, followed by Phase 10 Headers and Footers;
    - `EXPL-TASK-018` retains the recursive minimum-geometry diagnostic; and
    - Structured Input remains deferred.

- 2026-07-30 — `EXPL-TASK-019` — Deliver Menus and purpose-specific catalog
  screen navigation.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T20:19:43-07:00
  - Completed: 2026-07-30T20:52:17-07:00
  - Outcome:
    - delivered copied immutable `Menu` models and `MenuItem` descriptors plus
      one persistent non-container `MenuBar` and popup session per App;
    - reused command definitions for menu label, enabled/disabled reason,
      checked state, and first shortcut while preserving one router and
      correlated completion path across Button, HotkeyBar, menu, mnemonic,
      binding, and direct invocation;
    - delivered exact Alt top-level mnemonics, F10 and Ctrl-Space access,
      arrows/Home/End/Enter/Escape, sibling mnemonics, nested popups,
      deterministic clipping/viewports, and exact eligible Button focus
      restoration without a nested event loop;
    - added bounded flat core and automation MenuBar details, deep-copy
      projection, structural/path validation, and response-bound evidence;
    - converted `expletives-test` to the `toolkit.catalog` scene with one
      persistent File/View/Actions MenuBar and purpose-specific Core/Layout,
      Text/Display, and Actions screens; and
    - retained 28 stable public-API controls, 16 Layouts, 17 menu entries,
      visible checked/disabled/nested states, and no overflow at the live
      80x24 acceptance geometry.
  - Verification:
    - `make verify` passed formatting, vet, ordinary and Unix-socket tests,
      the expanded controlling-PTY menu/resize/lifecycle test, the full race
      suite, every debug/release/profiling build, executable checks, and
      smoke/self-checks;
    - the display/command-label normalization fuzzer executed 285,312 cases
      in three seconds, and the complete package set compiled for CGO-free
      Linux arm64;
    - the conservative response proof measured 37,214,739 JSON bytes and
      111,644,217 bytes for three retained maxima, below the 36 MiB and
      128 MiB bounds; and
    - a live attached debug session observed the closed catalog, opened View
      with raw Alt-V, switched to Actions through a normal menu command,
      observed focus and checked-screen repair, activated Toggle through the
      Actions menu, verified the magenta frame/style transition, received a
      final `app.quit` completion, and confirmed socket cleanup.
  - Physical input evidence:
    - the Linux controlling-PTY test exercised fragmented input, resize,
      physical Alt-F, F10, Right/Down/Enter, Ctrl-Space, nested Alt-A/S,
      staged Escape dismissal, ordinary quit, terminal Ctrl-C, and exact
      terminal-mode restoration.
  - Contracts:
    - [`Menus API v0`](../docs/specifications/menus-api-v0.md)
    - [`Foundational Go API v0`](../docs/specifications/go-api-v0.md)
    - [`Automation Protocol v1`](../docs/specifications/automation-protocol-v1.md)
    - [`expletives-test`](../docs/specifications/expletives-test.md)
  - Process:
    - completed as one routine feature without convening the Panel;
    - ACP is authorized without review for this checkpoint; and
    - recorded a named cross-target compile-check opportunity in the
      [`process efficiency log`](reviews/process-efficiency-log.md).
  - Follow-up:
    - `EXPL-TASK-018` remains the first backlog item for the recursive
      root-minimum `TOO SMALL` diagnostic;
    - Phase 9 Selection is the next planned common-control phase; and
    - Structured Input remains deferred.

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
