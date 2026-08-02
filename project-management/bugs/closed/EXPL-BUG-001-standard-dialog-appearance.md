# Bug: Standard Dialog Appearance Diverged From Turbo Vision

- ID: `EXPL-BUG-001`
- Status: Closed
- Priority: High
- Reported: 2026-08-02
- Reporter: project operator
- Owner: Codex
- Related work: Phase 17 Modals

## Symptom And Impact

MessageBox and the other standard dialogs used correct compound structure but
incorrect dialog Button colors, shape, spacing, and presentation. Buttons also
lacked their own raised shadow, while the independent dialog shadow was
already correct.

## Reproduction Or Evidence

Open any entry under the catalog's Dialogs menu and compare its Button face,
mnemonic, focus/default state, and shadow to the Turbo Vision gray-dialog
palette and standard 10-by-2 Button geometry.

## Expected Behavior

Standard dialogs use white active borders and titles on light gray, black
message text on light gray, and raised green Buttons with state-specific text,
yellow mnemonics, and black half-block right/bottom shadows.

## Actual Behavior

Buttons were one-row ASCII placeholders using bracket-like state markers and
the general screen palette. Their intrinsic size and dialog-row spacing did
not reserve the Turbo Vision Button shadow.

## Root Cause

The original Actions phase implemented Button semantics and input behavior
before the standard-dialog visual contract was applied. Later modal tests
covered lifecycle and outcomes but did not assert exact Button frame cells.

## Resolution

The shared Button renderer now implements the Turbo Vision gray-dialog
normal, default, focused, pressed, disabled, mnemonic, and shadow styles. Its
natural geometry is at least 10 columns by 2 rows; standard dialogs use the
canonical two-cell inter-button spacing and K/Y/N/C mnemonics. Explicit
one-row caller geometry remains a documented degraded override.

## Validation

- exact MessageBox border, title, message, Button, and dialog-shadow cells;
- raised Button geometry and mnemonic styling in MessageBox, ConfirmDialog,
  InputDialog, and ProgressDialog;
- raw KeyDown pressed-state and Progress cancellation-disabled frames;
- full ordinary, integration, race, static-analysis, build-mode, and catalog
  self-check gates.

## History

- 2026-08-02 — reported, corrected in the shared renderer, verified through
  attached automation, and closed.
