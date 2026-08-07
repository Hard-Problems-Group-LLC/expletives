# Bug: TextField Could Not Report Or Present Live Working-Value State

- ID: `EXPL-BUG-003`
- Status: Resolved
- Priority: High consumer usability
- Reported: 2026-08-07T15:55:18-07:00
- Resolved: 2026-08-07T16:40:12-07:00
- Reporter: radioradio operator via `radioradio-ECR-2026-002`
- Owner: Codex
- Related work: Phase 19 Slice 19.6; Text/Numeric Input API v0

## Symptom And Impact

`TextField.ChangeCommand` is intentionally emitted only for a changed commit.
There was no command for an accepted working-value edit or paste, so a
consumer could not update byte counts, validation-dependent actions, or
related status as the operator typed. Focused-but-not-editing and active
editing shared the same fixed presentation roles, and no bounded UTF-8 byte
policy could restyle the whole entered value near an external payload limit.
Enter also had no optional submit command after the second-stage edit commit.

## Reproduction Or Evidence

Bind a `ChangeCommand`, focus a TextField, press Enter, and type one character.
The working value and snapshot changed, but no command reached the
application; committed-text and paste events likewise had no command-routing
path. Both selected and editing states painted `text_input.focused*`. The
radioradio Chat composer consequently remained at `0/233 bytes` until commit
and could not show its requested selected/editing and whole-message warning
states.

## Expected Behavior

An opt-in bounded contract reports interactive current-value transitions
outside toolkit locks, provides owning-process access to the current value,
distinguishes selected from editing backgrounds, and applies an ordered UTF-8
byte threshold style to every entered cell. Password snapshots remain
redacted and do not reveal the active byte band. An optional submit command
routes once after Enter commits an actively edited field. Zero-value behavior
remains compatible.

## Root Cause

The Phase 12 input contract covered committed application values, validation,
and editing mechanics, but did not include a live consumer projection or
consumer-specific presentation thresholds. The later constrained-message
composer supplied those requirements.

## Resolution

`TextFieldOptions` now provides optional `EditCommand` and `SubmitCommand`,
distinct `FocusedStyle` and `EditingStyle` roles, and a copied ordered policy
of at most 16 `TextFieldByteStyle` entries. `CurrentText` exposes the working
value to the owning process. Interactive key and committed-text changes route
live notifications outside toolkit locks; caret-only, rejected, and
programmatic transitions remain silent. Enter can route one explicit submit
after commit. Thresholds use canonical UTF-8 byte counts, style the complete
entered value, and are suppressed while Password presentation is active.
Core and automation snapshots carry validated copied policy without exposing
password text or an active password threshold.

## Validation

Exact 174/175 and 221/222 boundaries, multi-byte classification, whole-value
styling, selected/editing backgrounds, key and paste notification,
cancellation, silent transitions, submission, password suppression,
construction bounds, deep-copy behavior, malformed automation rejection, and
public-consumer compilation have focused tests. The maintained
`expletives-test` catalog demonstrates and self-checks the states and command
transitions. At 2026-08-07T16:40:12-07:00, `make verify` passed vet, ordinary
and race suites, attached Unix-socket and PTY integration, debug/release/
profiling builds, self-checks, and smoke tests. All fixtures were synthetic;
no application serial or RF path was used.

## History

- 2026-08-07T15:55:18-07:00 — Accepted from radioradio ECR 2026-002 as Phase
  19 Slice 19.6; implementation began on an auditable feature branch.
- 2026-08-07T16:05:31-07:00 — Paused uncommitted for a downstream live-dialog
  diagnosis; the worktree was preserved.
- 2026-08-07T16:23:30-07:00 — Resumed after the downstream repair passed its
  complete fake-only and live acceptance gates.
- 2026-08-07T16:40:12-07:00 — Complete project verification passed and the
  repair was approved for permanent feature-branch publication and merge.
