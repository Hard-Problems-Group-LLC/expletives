# Phase 23 Table Visual Roles And Interactive Catalog Evidence

- Owner: `EXPL-TASK-039`
- Completed: 2026-08-13T16:32:04-07:00
- Regression repair completed: 2026-08-13T18:28:58-07:00
- Navigation/fill repairs completed: 2026-08-13T23:03:31-07:00
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

## Post-Completion Regression Repair

Operator evaluation exposed `EXPL-BUG-012`: palette `■` glyphs were label
characters and therefore inherited DropDown row-state colors instead of
showing the colors they represented. Slice 23.6 added an optional copied,
one-cell `ListItem` indicator with an independent semantic Theme style,
complete staged-Theme validation, exact frame behavior, and bounded retained
storage. Table/DataGrid catalog palette choices now pair that swatch with the
existing textual color name in both open and collapsed DropDown states.

A rebuilt isolated attached instance at 140 by 40 showed the first eight open
foreground choices as exact `#000000`, `#0000AA`, `#00AA00`, `#00AAAA`,
`#AA0000`, `#AA00AA`, `#AA5500`, and `#AAAAAA` foreground/background swatches.
The adjacent labels retained ordinary, current, and selected row styles.
Keyboard selection of red closed the popup, retained the red swatch in the
field, and resolved `tables.control` Body text to `#AA0000` on `#AAAAAA`.
The attached process returned an `exited` completion with its final snapshot
and removed its owned socket.

The exact repaired worktree passed `make verify` on 2026-08-13, covering vet,
ordinary tests, the PTY lifecycle test, the full race suite, and debug,
release, and profiling builds. Phase 23 is complete again; Phase 24 remains
planned.

## Post-Completion Navigation And Width Repairs

Operator keyboard evaluation exposed `EXPL-BUG-013`: nested focusable
Notebook and ScrollablePanel ancestors could tie an otherwise adjacent
foreground/background color-selector row crossing, and primary-axis-first
ranking could favor a far cross-axis control. Spatial focus now excludes the
focused control's ancestor chain from peer competition and ranks remaining
directional candidates by squared center distance with deterministic
cross/primary tie-breakers. Exhaustive coverage checks all four arrows from
every foreground/background selector on both screens, including the former
row 10/11 collision at zero and maximum scroll offsets.

Operator visual evaluation then exposed `EXPL-BUG-014`: the shared catalog's
only growing columns both had maximum widths, leaving 39 unused cells inside
the wide Table viewport. Summary is now the unbounded flexible demo column.
Exact tests prove Table and DataGrid content fills their wide viewport while
the 84 by 24 layout retains intentional horizontal scrolling.

A rebuilt 190 by 40 attached instance crossed the exact former navigation
boundary and reported equal 121-cell content/viewport widths for Table and
DataGrid, without unnecessary scrollbars. It shut down through the supported
automation command, published a final snapshot, and removed its socket. The
exact repaired worktree passed `make verify` on 2026-08-13, including vet,
ordinary tests, PTY integration, the complete race suite, and debug, release,
and profiling builds. Phase 23 is complete again; Phase 24 remains planned.
