# Bug: Disabled Button Labels Had Zero Default Contrast

- ID: `EXPL-BUG-005`
- Status: Resolved
- Priority: High consumer usability
- Reported: 2026-08-07
- Accepted: 2026-08-08T09:47:55-07:00
- Resolved: 2026-08-08T10:31:25-07:00
- Reporter: radioradio ECR 2026-001
- Owner: Codex
- Related work: Phase 19 Slice 19.8; Actions API v0

## Symptom And Impact

The default `button.disabled` style resolved both foreground and background to
`#808080`. Button painting retained the command label, but every label cell
had zero color contrast and appeared blank. Consumers could not distinguish
an unavailable named action from an unlabeled or broken control.

## Root Cause

`DefaultTheme` selected medium gray for both the disabled foreground and the
darkened dialog surface. A standard-dialog regression had institutionalized
the equal pair even though disabled presentation retained correct semantic
state and label content.

## Resolution

The default disabled foreground is now black while the background remains the
medium-gray dialog surface. Ordinary and standard-dialog Buttons preserve the
`button.disabled` semantic role, label, disabled reason, and ineligible focus
and activation state. The maintained catalog uses the same practical
contrast, and the basic terminal projection deterministically maps it to
black on bright black rather than collapsing the colors.

## Validation

Focused Theme, ordinary Button, ProgressDialog, catalog, and ANSI-16 encoder
regressions assert the exact colors, semantic role, retained graphemes, and
nonzero contrast. At 2026-08-08T10:31:25-07:00, `make verify` passed format,
vet, ordinary and Unix-socket tests, PTY integration, the complete race suite,
debug/release/profiling builds, catalog self-checks, and smoke tests.

## History

- 2026-08-08T09:47:55-07:00 — Accepted by direct operator instruction from
  radioradio ECR 2026-001 as Phase 19 Slice 19.8.
- 2026-08-08T10:31:25-07:00 — Repair and complete project verification passed.
