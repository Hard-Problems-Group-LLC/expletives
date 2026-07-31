# Review Finding: EXPL-REV-001-D-S07-F002

- Finding ID: `EXPL-REV-001-D-S07-F002`
- Reviewer identity: `/root/design_security`
- Seat and lens: Seat 7 — Security
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Any 4,096 accepted input, command, or reset requests permanently
  exhaust the App-wide request-ID store.
- Consequence: A controller can issue unique no-op `reset_input` requests and
  permanently disable later semantic input, commands, and resets, including
  other input sources, for the process lifetime.
- Evidence: The App rejects capacity at 4,096 and never evicts
  (`app.go:189-197`; `types.go:11-18`). Even no-op resets consume IDs
  (`input.go:280-314`). The protocol acknowledges that all subsequent App
  dispatches fail after exhaustion
  (`docs/specifications/automation-protocol-v1.md:751-755`), conflicting with
  the requirement that attached mode remain human-usable and preserve an
  emergency path (`docs/specifications/expletives-test.md:150-153`).
- Recommendation: Align protocol and App deduplication lifetimes using bounded
  eviction or tombstones, or reserve independent capacity for human and
  emergency actions. Add sustained-operation and exhaustion tests.
- Related lenses: Reliability; performance; client ease of use; testability.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F004`; permanent
  4,096-ID App exhaustion is the sustained-operation and denial consequence of
  the same mismatched request-retention lifecycle.
- Resolution: pending
- Resolution evidence or operator record: pending
