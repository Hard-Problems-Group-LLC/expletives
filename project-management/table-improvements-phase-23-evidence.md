# Phase 23 Table Visual Roles And Interactive Catalog Evidence

- Owner: `EXPL-TASK-039`
- Completed: 2026-08-13T16:32:04-07:00
- Branch: `feature/table-improvements`
- Baseline checkpoint: `33506edca7f2` (completed Phase 22)
- Status: complete; Phase 24 remains planned

## Delivered Contract

- Added fixed copied `TableVisualRoles` and `DataGridVisualRoles` values with
  compatible default normalization, staged-Theme membership validation,
  exact direct and Transaction replacement, instance-local painting, exact
  core details, and compact automation digests.
- Added live atomic Table focus-mode replacement so the catalog can exercise
  its already-public construction choice without rebuilding the control.
- Routed Table-owned integrated scrollbars, status states, headers, body
  states, Columns band/action, and DataGrid editor states through the selected
  per-instance semantic roles while preserving existing defaults.
- Moved Table and DataGrid out of Collections into unique Controls/Tables and
  Controls/DataGrid routes. Each screen owns a post-minimum 3:1 control/
  Notebook Layout, scrollable Options and Colors pages, unique role IDs,
  comprehensive model fixtures, and atomic reset paths.
- Options exercises Columns, all four selection styles, required selection,
  Table row/cell focus, and presentation/demo reset. Colors provides bounded
  named foreground/background DropDowns for every role, preserves
  attributes/unrelated Theme styles, and applies each discrete choice
  atomically.
- Updated the Collections, Go API, automation, and directed Table
  Improvements specifications plus public-consumer-only API coverage.

## Verification

The exact final worktree passed `make verify` on 2026-08-13. This includes:

- formatting and `go vet`;
- all ordinary library, external-consumer, automation, catalog, command, and
  terminal tests;
- the debug-binary PTY lifecycle integration test;
- the complete race suite, including the 213-second end-to-end attached
  catalog audit;
- debug, release, and profiling builds of `expletives-test` and
  `expletivesctl`; and
- every build mode's version, self-check, and command smoke tests.

The attached audit's aggregate context is five minutes because its many
snapshot-bearing round trips now include both complete Colors-page control
trees under race instrumentation. Independent protocol request deadlines are
unchanged.

Focused regression coverage additionally proves role normalization and
Theme ownership, transaction rollback, DataGrid editor cancellation, compact
automation validation and response bounds, menu routes/mnemonics, live
Options synchronization and incompatible-policy rejection, instance-local
color mutation, exact reset, 3:1 post-minimum growth, and the 84 by 24
minimum-useful layout.

## Attached Drive And Observe

The rebuilt debug binaries were exercised through `expletivesctl` using the
supported attached automation boundary.

- At 140 by 40, DataGrid occupied 85 by 30 and its Notebook 48 by 30 with no
  overflow. Options retained a 40 by 30 logical canvas and only the expected
  vertical scrollbar.
- Keyboard traversal selected the Colors Notebook page. Its 52 by 35 logical
  canvas exposed both scroll axes without Layout overflow.
- The first foreground DropDown changed the DataGrid Body role from black to
  navy. The intended frame resolved the unique Body role to `#0000AA` on
  `#AAAAAA`, while the separate Table and DataGrid role digests remained
  distinct. Reset restored black, Multiple/required selection, the Columns
  feature, five visible columns, and the Options page.
- The Table Options UI toggled the Columns feature off and back on through raw
  keyboard input. The Table Columns action then opened its 58 by 23 modal,
  focused the bounded inventory, reported dialog-open state, and restored
  ColumnsAction focus on Escape without overflow.
- At 84 by 24, both dedicated screens settled without overflow. Each live
  control occupied 43 by 14, each Notebook occupied 34 by 14, the Columns
  action remained visible, and the 40 by 30 Options canvas exposed bounded
  horizontal and vertical scrolling.
- Both temporary attached processes returned an `exited` completion with an
  associated final snapshot and removed their owned sockets.

During this pass the wide DataGrid screen exposed a five-cell Options-content
deficit caused by its fixed focus explanation. Increasing the shared logical
Options width from 34 to 40 repaired the issue and the new narrow/wide tests
prevent recurrence.

## Acceptance And Follow-Up

The operator explicitly directed automatic delivery through Phase 23 because
these dedicated screens are the evaluation surface. The automated wide,
narrow, representative-color, Options, modal, reset, and complete-gate
evidence therefore closes the delegated Phase 23 acceptance without claiming
a separate human visual session.

Phase 24 must execute the recorded deterministic covering array and targeted
matrices in
[`table-improvements-phase-24-matrix.md`](table-improvements-phase-24-matrix.md).
No Phase 23 blocker or non-deferred defect remains.
