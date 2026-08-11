# Bug: Narrow Overflow Warning Can Present A Contextless OK Action

- ID: `EXPL-BUG-010`
- Status: Resolved
- Priority: High
- Reported: 2026-08-11T10:25:53-07:00
- Resolved: 2026-08-11T11:10:16-07:00
- Reporter: project operator during native-terminal acceptance
- Owner: Codex
- Related work: `EXPL-TASK-037`, Phase 19 Slice 19.13,
  [`layouts-and-overflow.md`](../../../docs/specifications/layouts-and-overflow.md)

## Symptom And Impact

When a surface becomes too narrow for a Layout minimum but remains at least
four cells wide, the default overflow notification paints only `[OK]`. The
action remains operable, but it does not explain what happened or what the
operator is acknowledging.

## Reproduction Or Evidence

Resize the attached native `expletives-test` session to 29 by 42 while the
catalog form requires 39 cells of width. Frame 275 records `overflow-8` in
`default_active` state with a 14-cell horizontal deficit and a centered
black-on-yellow `[OK]` overlay. Physical Enter at frame 276 changes the same
episode to `acknowledged`, proving the action path works while leaving the
prompt's meaning undiscoverable.

## Expected Behavior

The compact fallback word-wraps a plain overflow explanation and places the
`[OK]` action on its own following centered row when height permits. It never
shows an action label without explanatory text.

## Actual Behavior

Every nonzero surface narrower than four cells shows `!`; every surface of
four or more cells shows only `[OK]`, regardless of how much explanatory text
could fit.

## Root Cause

The foundational fallback chose between exactly two fixed strings based only
on whether the four-cell action label fit. It had no adaptive explanatory
message tier or vertical presentation model.

## Resolution

The root-owned fallback now uses the shared deterministic word-wrapping
behavior to form a bounded centered block. It paints `Layout overflow` across
as many whole-word rows as the width requires and places `[OK]` alone on the
following centered row when height permits. When height cannot hold the full
message, it retains the compact `Overflow` explanation; widths below that
word retain the high-visibility `!` indicator. Existing overflow episode,
dismissal, style, ownership, and non-recursive behavior is unchanged.

## Validation

- Focused geometry, wrapping, style, ownership, indicator, dismissal,
  recovery, handler, timeout, and saturation coverage passes.
- The refined `make verify` run passes formatting, vet, ordinary tests,
  Unix-socket and PTY integration, the complete race suite, all three build
  modes, and command smoke checks.
- In the rebuilt native attached session, frame 117 at 12 by 42 has eight
  usable content columns and paints three centered root-owned rows: `Layout`,
  `overflow`, and `[OK]`, all bold black on yellow. The same frame retains
  `overflow-3` in `default_active` state with its exact 42-cell horizontal
  deficit.
- The project operator visually accepted the rebuilt native result as
  excellent before authorizing ACP.

## History

- 2026-08-11T10:25:53-07:00 — Confirmed through the attached native-terminal
  frame and physical Enter dismissal.
- 2026-08-11T10:33:38-07:00 — Accepted as a usability defect under direct
  operator feedback and moved to in-progress repair.
- 2026-08-11T10:40:21-07:00 — Implemented the adaptive explanatory tiers and
  completed focused plus full-project automated verification; paused for one
  rebuilt native-session acceptance run.
- 2026-08-11T10:57:39-07:00 — Native frame 81 proved the first candidate's
  exact text, center placement, black-on-yellow style, root ownership, and
  active structured episode. Operator accepted the improved context and
  directed word wrapping plus a separate `[OK]` line; resumed implementation.
- 2026-08-11T11:01:46-07:00 — Implemented the centered word-wrapped block and
  dedicated action row; focused and complete automated verification pass.
- 2026-08-11T11:10:16-07:00 — Rebuilt native frame 117 and direct operator
  acceptance completed the final validation; resolved.
