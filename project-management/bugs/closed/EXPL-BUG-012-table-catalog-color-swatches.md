# Bug: Table Catalog Color Swatches Used Row Colors

- ID: `EXPL-BUG-012`
- Status: Resolved
- Priority: High catalog usability
- Reported: 2026-08-13T17:48:00-07:00
- Confirmed: 2026-08-13T18:06:15-07:00
- Resolved: 2026-08-13T18:28:58-07:00
- Reporter: project operator
- Owner: Codex
- Related work: Phase 23 Slice 23.6; Table Improvements v1

## Symptom And Impact

Every named color in the Table catalog's foreground/background DropDowns
showed the same black `■` preview under an ordinary row. Current and selected
rows recolored that glyph with the row-state style instead. The visible name
remained correct and committing a choice changed the intended Table role, but
the promised color preview did not identify the represented color.

## Reproduction Or Evidence

Attached automation at frame sequence 122 observed the open
`tables.colors.00.foreground` popup. Ordinary `■` cells resolved through
`drop_down.popup` as black on gray, the selected cell resolved through
`collection.selected` as white on blue, and the current cell resolved through
`collection.current` as black on green. No preview cell used its palette
entry's color.

## Root Cause

Phase 23 encoded `■` as the first character of each `ListItem.Label`.
DropDown correctly paints a complete popup row with its semantic row-state
style, so the glyph had no independent semantic style and could not act as a
color swatch.

## Resolution

Added an optional paired `ListItem.Indicator` and `IndicatorStyle`. The
indicator normalizes to exactly one canonical cell, retains its independent
semantic style in ListBox and popup rows, contributes to exact measurement
and retained-storage bounds, and participates in complete staged-Theme
validation. DropDown repeats the selected indicator when collapsed; ComboBox
keeps its editable collapsed field and shows indicators in its popup.

The catalog now represents each named color with a solid Theme-owned swatch
whose foreground and background equal that color, followed by the unchanged
textual color name. Current, selected, disabled, and ordinary styles continue
to apply to the surrounding row.

## Validation

- Unit and external-consumer tests cover paired-field validation, composed
  one-cell text, deterministic replacement of unsupported wide text,
  ListBox/DropDown rendering, popup row-state independence, exact resolved
  colors, copied models, atomic item/Theme transactions, Theme-removal
  rejection, and retained model behavior.
- Catalog tests cover every palette model and the first eight exact visible
  swatches on both dedicated screens.
- A rebuilt 140 by 40 attached instance showed eight distinct exact-color
  swatches while adjacent labels retained their row-state styles. Selecting
  red produced a red collapsed swatch and red Table Body foreground.
- `make verify` passed on the exact repaired worktree, including vet, all
  ordinary tests, PTY lifecycle integration, the full race suite, and all
  three required build modes.

## History

- 2026-08-13T18:06:15-07:00 — Confirmed from attached semantic state and
  exact intended-frame cells; Phase 23 reopened for Slice 23.6 repair.
- 2026-08-13T18:28:58-07:00 — Repaired, verified through the supported
  attached boundary, and closed after the complete project gate passed.
