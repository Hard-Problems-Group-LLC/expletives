# Bug: Automation Client Rejects A Valid Command Inventory

- ID: `EXPL-BUG-008`
- Status: Resolved
- Priority: High
- Reported: 2026-08-09T10:19:22-07:00
- Resolved: 2026-08-09T10:26:55-07:00
- Reporter: radioradio live acceptance
- Owner: Codex
- Related work: Phase 19 consumer repair; automation protocol v1

## Symptom And Impact

`expletivesctl` could not attach to a valid application that advertised more
than 64 automation-enabled commands. It failed the initial Hello with
`automation: hello capability inventory exceeds bounds`, so no safe semantic
observation or control was possible.

## Reproduction Or Evidence

The radioradio application and controller were built from Expletives revision
`098195361588`. The server advertised a registry larger than 64 commands and
the reference client rejected it before returning `Hello`. No UI input or
radio operation occurred.

## Expected Behavior

The reference client accepts a unique, bounded command inventory through the
documented `MaxControls` (4,096) limit and rejects an inventory above it.

## Actual Behavior

`validateHello` applied a private literal limit of 64 even though
`automationCommandInventory` and the protocol specification use
`expletives.MaxControls`.

## Root Cause

The client-side validation literal was not updated when the normative server
inventory bound was established at `MaxControls`.

## Resolution

Client Hello validation now uses `expletives.MaxControls`, matching both the
server's inventory builder and the existing normative protocol. A regression
constructs unique valid command IDs at the exact limit and confirms that the
next entry is rejected.

## Validation

- Focused Hello-validation tests pass.
- `make verify` passes vet, ordinary tests, Unix-socket automation tests, PTY
  lifecycle integration, the complete race suite, and debug, release, and
  profiling builds for both supported commands.

## History

- 2026-08-09T10:19:22-07:00 — Confirmed during a downstream, receive-only
  attached-automation acceptance attempt; isolated a feature worktree without
  modifying the concurrent main checkout.
- 2026-08-09T10:26:55-07:00 — Implemented and completed full verification.
