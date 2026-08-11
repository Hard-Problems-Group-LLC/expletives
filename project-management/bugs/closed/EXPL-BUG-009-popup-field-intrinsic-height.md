# Bug: Width-Only Popup-Field Minimum Can Collapse The One-Row Control

- ID: `EXPL-BUG-009`
- Status: Resolved
- Priority: High
- Reported: 2026-08-11T01:26:22-07:00
- Resolved: 2026-08-11T01:34:02-07:00
- Reporter: radioradio consumer investigation
- Owner: Codex
- Related work: `EXPL-TASK-037`, Phase 20 Slice 20.0,
  [`collections-api-v0.md`](../../../docs/specifications/collections-api-v0.md)

## Symptom And Impact

A `DropDown` constructed with a useful width-only `PanelOptions.MinimumSize`
can receive zero height from a Layout. Its semantic state still reports the
current and selected items, but the closed field paints no label, arrow, or
background cells. The control can appear absent while adjacent actions remain
visible and actionable.

`ComboBox` uses the same construction path and has the same latent defect.

## Reproduction Or Evidence

Construct a populated `DropDown` with `MinimumSize: Size{Width: 24}`, add it
to a BoxLayout, and attach the Layout. The control retains minimum height zero,
receives a zero-row rectangle under its vertical-natural hint, and owns no
framebuffer cells. Supplying `Height: 1` downstream masks the defect.

The affected downstream frame independently proved that this is not a color
collision: the control resolved white foreground on cyan background but had
no owned cells or nonblank glyphs.

## Expected Behavior

The directed collection contract defines `DropDown` and `ComboBox` as
one-row collapsed fields. A construction-time width override must preserve a
minimum height of one row while retaining the caller's width.

## Actual Behavior

The shared leaf constructor derives an intrinsic minimum only when the entire
`MinimumSize` is zero. A width-only minimum therefore disables both-axis
derivation and retains height zero.

## Root Cause

`Transaction.NewDropDown` and `Transaction.NewComboBox` do not reassert their
one-row intrinsic height after the generic leaf constructor applies a partial
caller minimum.

## Resolution

Both popup-field constructors now reassert their intrinsic minimum height
after applying caller options. A width-only minimum therefore retains the
caller width and gains the required one-row height; complete zero minima retain
ordinary automatic intrinsic sizing, and positive caller heights remain
unchanged. The collection contract now states the partial-minimum behavior.

## Validation

- A focused regression proves direct minima, vertical BoxLayout geometry,
  selected-label framebuffer ownership, and foreground/background contrast for
  both `DropDown` and `ComboBox`.
- All-package ordinary Go tests pass.
- `make verify` passes formatting, vet, ordinary tests, Unix-socket and PTY
  integration, the complete race suite, debug/release/profiling builds, and
  smoke checks.

## History

- 2026-08-11T01:26:22-07:00 — Confirmed during downstream live-frame and
  semantic-layout comparison; moved directly to in-progress handling under
  the operator-authorized upstream repair.
- 2026-08-11T01:34:02-07:00 — Implemented the scoped constructor repair,
  documented the contract, and completed focused plus full-project
  verification.
