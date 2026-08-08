# Bug: TextField Could Not Declare A Smaller UTF-8 Byte Maximum

- ID: `EXPL-BUG-004`
- Status: Resolved
- Priority: High consumer correctness
- Reported: 2026-08-07T17:07:17-07:00
- Resolved: 2026-08-07T17:23:42-07:00
- Owner: Codex
- Related work: Phase 19 Slice 19.7; radioradio ECR 2026-004

## Symptom And Impact

`TextField` used only the library-wide 64-KiB safety ceiling. A consumer with a
smaller protocol payload could not stop excess working input atomically, so it
had to reject later at submit time or destructively repair the editor after a
live-edit notification.

## Expected Behavior

A zero-compatible copied option declares an effective per-field canonical
UTF-8 byte maximum. Every construction, setter, key, selection replacement,
committed text, and paste path enforces it atomically. Capacity rejection
changes no value, caret, frame, selection, or command routing; deletion,
navigation, cancellation, and Enter remain usable for recovery or submission.

## Root Cause

Phase 12 established only global resource limits. Phase 19 Slice 19.6 added
consumer byte-style thresholds, but presentation thresholds deliberately did
not alter input admission.

## Resolution

`TextFieldOptions.MaximumBytes` now accepts zero for the existing
`MaxTextInputBytes` behavior or a narrower positive ceiling. Construction,
`SetText`, transaction replacement, printable keys, selection replacement,
committed text, and paste calculate the complete canonical UTF-8 candidate and
reject an excess value atomically without truncation or an edit command.
`MaximumBytes()` and core/automation `TextFieldDetails` expose the effective
bound, and the automation trust boundary rejects absent, excessive, or
internally inconsistent values.

## Validation

Focused tests cover zero compatibility, invalid construction, setter
atomicity, exact 233-byte acceptance, excess ASCII and multibyte input,
selection replacement, paste, unchanged caret/selection/commands, Left,
Delete, Backspace, and Enter. Public-consumer and malformed-automation tests
cover the exported contract. The maintained `expletives-test` catalog exposes
and self-checks a ten-byte instance. At 2026-08-07T17:23:42-07:00,
`make verify` passed vet, ordinary and race suites, Unix-socket and PTY
integration, debug/release/profiling builds, catalog self-checks, and smoke
tests. All fixtures were synthetic; no serial or RF path was used.

## History

- 2026-08-07T17:07:17-07:00 — Accepted from radioradio ECR 2026-004 as Phase
  19 Slice 19.7 on a permanent feature branch; Slice 19.4 was paused.
- 2026-08-07T17:23:42-07:00 — Full project verification passed and the repair
  was approved for permanent feature-branch publication and non-fast-forward
  merge.
