# Review Finding: EXPL-REV-001-D-S09-F002

- Finding ID: `EXPL-REV-001-D-S09-F002`
- Reviewer identity: `/root/design_reliability`
- Seat and lens: Seat 9 — Reliability, concurrency, and portability
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Every App permanently exhausts correlated input after 4,096 accepted
  requests.
- Consequence: Ordinary human or automated use deterministically loses all
  further key, command, and reset acceptance; human mode treats
  `ErrRequestCapacity` as fatal. Each key lifecycle event consumes a separate
  ID, so the limit is reachable in a routine interactive session.
- Evidence: `types.go:11-19`, `app.go:189-197`, `input.go:178-180`, and
  `cmd/expletives-test/main.go:265-272`. The contract explicitly confirms no
  eviction and process-lifetime failure at
  `docs/specifications/go-api-v0.md:577-581` and
  `docs/specifications/automation-protocol-v1.md:746-759`.
- Recommendation: Replace lifetime accumulation before freezing the API, such
  as bounded expiry with an explicit duplicate window or per-source monotonic
  request sequencing or high-water marks, and separate uncorrelated physical
  input from automation correlation where appropriate.
- Related lenses: Performance; client ease of use; testability.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F004`; it is the
  ordinary-interactive manifestation of the same non-evicting App request-ID
  lifecycle.
- Resolution: pending
- Resolution evidence or operator record: pending
