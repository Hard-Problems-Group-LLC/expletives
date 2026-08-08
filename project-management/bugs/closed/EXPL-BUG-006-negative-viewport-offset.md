# Bug: Shared Viewport Clamp Allowed Negative Offsets

- ID: `EXPL-BUG-006`
- Status: Resolved
- Priority: High control-state correctness
- Reported: 2026-08-08
- Accepted: 2026-08-08T09:47:55-07:00
- Resolved: 2026-08-08T10:31:25-07:00
- Reporter: radioradio ECR 2026-006
- Owner: Codex
- Related work: Phase 19 Slice 19.9; Collections and Scrolling APIs

## Symptom And Impact

`ListBox` subtracted its horizontal arrow step on Left Arrow at origin. The
shared `clampViewportState` capped offsets at the calculated maximum but not
at zero, so an internally produced negative offset survived reflow and
shifted wrapped consumer content into invalid blank space.

## Root Cause

Public viewport-state normalization rejected negative caller values, while
the shared post-input/reflow clamp used only `min(offset, maximum)` on each
axis. The generic ScrollView key path performed a separate lower clamp, which
masked the shared invariant defect until ListBox reused the helper directly.

## Resolution

The shared clamp now enforces each axis in the closed interval from zero
through its calculated maximum. Left and Right at a boundary remain handled
no-ops. A consumer-equivalent wrapped multi-select ListBox with no horizontal
overflow retains zero offset, stable current/selection state, hidden
horizontal bar, and an unchanged intended frame across repeated boundary
keys.

## Validation

Focused helper and ListBox regressions cover negative candidates, upper
candidates, repeated Left/Right boundary input, typed viewport evidence, and
frame stability. Existing construction, resize, content replacement,
focus-visible, and automation validation continue to cover the shared
canonical state. At 2026-08-08T10:31:25-07:00, `make verify` passed format,
vet, ordinary and Unix-socket tests, PTY integration, the complete race suite,
debug/release/profiling builds, catalog self-checks, and smoke tests.

## History

- 2026-08-08T09:47:55-07:00 — Accepted by direct operator instruction from
  radioradio ECR 2026-006 as Phase 19 Slice 19.9.
- 2026-08-08T10:31:25-07:00 — Repair and complete project verification passed.
