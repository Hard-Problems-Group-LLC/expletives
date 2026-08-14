# Bug: Color Selector Directional Focus Was Viewport-Dependent

- ID: `EXPL-BUG-013`
- Status: Resolved
- Priority: High catalog keyboard usability
- Reported: 2026-08-13T18:36:00-07:00
- Confirmed: 2026-08-13T19:06:07-07:00
- Closed: 2026-08-13T23:03:31-07:00
- Reporter: project operator
- Owner: Codex
- Related work: Phase 23 Slice 23.7; Actions spatial focus; Table Improvements v1

## Symptom And Impact

Arrow focus between the foreground/background selectors can stop at an
otherwise ordinary adjacent-row boundary. The exact failure depends on the
Colors ScrollablePanel offset, making the two-column matrix feel unreliable
even though its Layout geometry is correct.

## Reproduction Or Evidence

An exhaustive in-process matrix over every four-way transition found that, at
viewport offset zero, Down from row 10 to row 11 and Up from row 11 to row 10
were no-ops in both columns on both the Table and DataGrid screens. The
operator-started attached instance reproduced Down from
`tables.colors.10.foreground` as `no_op` at frame sequence 508 while focus
remained on row 10. At offset two, the same row-11 Up transition succeeded.

## Root Cause

Generic spatial focus first searches same-parent peers, then other groups for
one unambiguous directional target. The color cells in different row Panels
require that cross-group search. Their focusable Notebook and ScrollablePanel
ancestors were also admitted as peer candidates. At the failing offset both
ancestor rectangles tied and blocked the adjacent cell. Excluding only those
ancestors exposed a second defect: primary-axis-first ranking could prefer a
slightly nearer but far cross-axis control to the visually adjacent aligned
cell. Scrolling moved the cell rectangles relative to those candidates and
changed the result.

## Resolution

Spatial focus now excludes the focused control and every one of its ancestors
from peer competition. Remaining directional candidates are ranked by squared
center distance, then cross-axis and primary-axis distance; an exact
cross-group tie remains a no-op. Same-parent preference, input-scope
confinement, and ordinary non-hierarchical movement remain intact. The Actions
and Table Improvements specifications record that contract.

## Validation

- Focused unit coverage proves both ancestor exclusion and that an aligned
  neighbor beats a slightly nearer but far cross-axis distractor.
- The catalog test exercises Left, Right, Up, and Down from every foreground
  and background selector on both Table and DataGrid. It also pins the former
  row 10/11 collision at offsets zero and maximum and proves the viewport does
  not move.
- A rebuilt attached 190 by 40 instance crossed the exact row 10/11 boundary
  in both directions and retained the expected foreground/background crossing.
- The exact repaired worktree passed `make verify`, including the complete race
  suite, PTY integration, and every required build mode.

## History

- 2026-08-13T19:06:07-07:00 — Confirmed through the exhaustive catalog matrix
  and exact attached frame sequence 508; Phase 23 reopened for Slice 23.7.
- 2026-08-13T22:55:00-07:00 — Excluded ancestors from peer competition,
  ranked remaining candidates by center distance, and passed exhaustive
  every-cell/direction tests plus the exact rebuilt attached crossing.
- 2026-08-13T23:03:31-07:00 — Full project verification passed; closed.
