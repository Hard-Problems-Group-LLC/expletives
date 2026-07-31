# Review Finding: EXPL-REV-001-D-S07-F003

- Finding ID: `EXPL-REV-001-D-S07-F003`
- Reviewer identity: `/root/design_security`
- Seat and lens: Seat 7 — Security
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The advertised 30-second execution deadline is cooperative and
  cannot bound an application handler's wall-clock execution or server
  teardown.
- Consequence: Invoking an approved but stuck handler can indefinitely hold
  the App dispatch gate and sole controller; `Close` preserves that
  connection and `Serve` waits forever for its worker.
- Evidence: Handlers run synchronously and may ignore cancellation
  (`docs/specifications/go-api-v0.md:520-525`). The dispatch mutex remains
  held across handler execution (`input.go:249-270`). The server waits
  synchronously for execution (`automation/server.go:343-362`), preserves
  processing connections on close (`automation/server.go:221-229`), and joins
  all workers before returning (`automation/server.go:163-167`).
- Recommendation: Define an enforceable teardown boundary, such as bounded
  isolated execution, or explicitly seek operator acceptance of
  cooperative-only, potentially unbounded shutdown. Do not describe the read
  timeout as a wall-clock execution bound.
- Related lenses: Reliability/concurrency; performance; client ease of use.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S09-F001`; both trace an
  uncooperative synchronous handler through the dispatch gate and server join
  to unbounded shutdown.
- Resolution: pending
- Resolution evidence or operator record: pending
