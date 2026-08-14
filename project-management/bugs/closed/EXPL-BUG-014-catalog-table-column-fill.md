# Bug: Catalog Table Columns Did Not Fill Their Viewport

- ID: `EXPL-BUG-014`
- Status: Resolved
- Priority: Normal catalog presentation
- Reported: 2026-08-13T22:38:00-07:00
- Confirmed: 2026-08-13T22:41:17-07:00
- Closed: 2026-08-13T23:03:31-07:00
- Reporter: project operator
- Owner: Codex
- Related work: Phase 23 Slice 23.8; Table Improvements v1

## Symptom And Impact

The dedicated Table screen leaves a broad unused region to the right of its
last column even though the Table itself fills the left catalog panel. The
DataGrid screen uses the same fixture columns and has the same policy. This
makes the primary demonstrated control look unfinished and reduces the value
of its wide-screen presentation.

## Reproduction Or Evidence

In the operator-started 190 by 40 instance, `tables.control` has a 120-cell
body viewport while its content width stops at 81 cells. The horizontal
maximum offset is zero and the vertical bar is visible, so the 39-cell
difference is unused space rather than horizontal scrolling or clipping.

## Root Cause

The Table implementation already distributes surplus viewport width according
to positive `Column.Grow`. The shared catalog fixture gives Grow only to Name
and Summary, then caps both at 22 and 24 cells. Once both caps are reached no
eligible Grow column remains, so the correct allocator leaves the surplus
unused. This is a demo-model configuration defect, not a Table sizing defect.

## Resolution

The shared catalog fixture retains Name's bounded growth and makes Summary the
unbounded flexible column. Existing Table/DataGrid Grow allocation therefore
consumes all surplus viewport cells at ordinary wide sizes, while Summary's
explicit base width retains narrow-screen wrapping and horizontal overflow.
No library sizing policy changed.

## Validation

- Dedicated Table and DataGrid catalog tests assert equal content/viewport
  width without a horizontal bar at 190 by 40, and content wider than the
  viewport with a visible horizontal bar at 84 by 24.
- A rebuilt attached 190 by 40 snapshot reported 121-cell content and viewport
  widths for both controls, with neither scrollbar needed.
- The exact repaired worktree passed `make verify`, including the complete race
  suite, PTY integration, and every required build mode.

## History

- 2026-08-13T22:41:17-07:00 — Confirmed from the attached semantic viewport
  and the shared catalog column model; queued as Slice 23.8 behind the active
  directional-focus repair.
- 2026-08-13T22:55:00-07:00 — Removed the catalog Summary cap; focused tests
  prove exact wide fill and retained narrow overflow for Table and DataGrid,
  and a rebuilt attached snapshot reports equal 121-cell viewport/content
  widths with no unnecessary bars.
- 2026-08-13T23:03:31-07:00 — Full project verification passed; closed.
