# Review Finding: EXPL-REV-001-D-S02-F009

- Finding ID: `EXPL-REV-001-D-S02-F009`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: minor
- Claim: Protocol v1 exposes queue and cancellation concepts without an
  execution model in which either can operate.
- Consequence: The public surface suggests pipelining and cancellation
  semantics that the single synchronous connection cannot provide, increasing
  protocol complexity without usable capability.
- Evidence:
  - `docs/specifications/automation-protocol-v1.md:273-312` advertises an
    automation queue limit.
  - `docs/specifications/automation-protocol-v1.md:633-657` specifies
    `cancel`, while
    `docs/specifications/automation-protocol-v1.md:1051-1053` excludes
    general cancellation.
  - `automation/server.go:608-622` can only return `no_op`/`too_late` for a
    known target.
  - The server processes one request synchronously per connection, and the
    single-controller rule prevents a second controller from submitting an
    out-of-band cancel while the first request is active.
- Recommendation: Remove `cancel` and the advertised queue limit from
  protocol v1 until requests can be queued or canceled out of band, or define
  and implement an explicit concurrent control channel with bounded ordering
  and cancellation semantics.
- Related lenses: protocol minimality, bounded queues, automation lifecycle
- Disposition: confirmed
- Disposition rationale: Version 1 advertises a reserved queue and exposes
  `cancel`, while its documented single synchronous request model makes every
  known cancellation too late and permits no pipelined cancellation.
- Resolution: pending
- Resolution evidence or operator record: pending
