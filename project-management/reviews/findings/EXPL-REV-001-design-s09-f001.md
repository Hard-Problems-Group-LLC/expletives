# Review Finding: EXPL-REV-001-D-S09-F001

- Finding ID: `EXPL-REV-001-D-S09-F001`
- Reviewer identity: `/root/design_reliability`
- Seat and lens: Seat 9 — Reliability, concurrency, and portability
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: blocker
- Claim: Dispatch acquisition and command-handler execution are not
  cancellation-bounded, so orderly shutdown can hang indefinitely.
- Consequence: A blocked or cancellation-ignoring handler holds `dispatchMu`;
  later calls cannot observe their expired contexts while waiting. Automation
  teardown then waits forever for its controller worker, and
  `expletives-test` can block before deferred terminal restoration.
- Evidence: `input.go:165-169`, `input.go:249-253`, and
  `input.go:340-347` use a non-cancellable mutex and synchronous handler call.
  `automation/server.go:163-167`, `automation/server.go:202-229`, and
  `cmd/expletives-test/main.go:372-380` join that potentially stuck path.
  This conflicts with
  `docs/specifications/concurrency-and-thread-safety.md:274-296`, which
  requires cancellation-observing blocking calls, terminal restoration, and
  joined toolkit goroutines.
- Recommendation: Introduce a cancellation-aware bounded dispatch or owner
  queue and make terminal restoration independent of application callback
  completion. Explicitly decide how uncooperative callbacks are isolated or
  how the shutdown guarantee is narrowed.
- Related lenses: Security; testability; client ease of use.
- Disposition: confirmed
- Disposition rationale: Dispatch waits on a non-cancellable mutex and runs
  handlers synchronously; server and command teardown then join that path with
  no enforceable bound, contrary to the cited cancellation and restoration
  requirements.
- Resolution: pending
- Resolution evidence or operator record: pending
