# Review Finding: EXPL-REV-001-D-S06-F002

- Finding ID: EXPL-REV-001-D-S06-F002
- Reviewer identity: `/root/design_testability`
- Seat and lens: Seat 6 — Testability: determinism, observability, state control, automation parity, unit and integration seams, reproducibility, and failure diagnosis
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Completion and exact-snapshot retention lifetimes diverge, so a query can prove completion after its associated snapshot has expired.
- Consequence: Tests cannot always retrieve the exact observed state associated with a retained completion, weakening reproducibility and diagnosis.
- Evidence:
  - `automation/protocol.go:87-108`
  - `docs/specifications/automation-protocol-v1.md:624-628`
  - `docs/specifications/automation-protocol-v1.md:767-773`
  - `automation/server.go:583-605`
  - `automation/server.go:625-653`
- Recommendation: Couple each retained completion to a retrievable snapshot or evict them atomically, and add churn tests.
- Related lenses: client ease of use, performance, reliability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F004`; independently
  expiring completion and exact-snapshot histories are part of the same
  mismatched retention authority and lifecycle.
- Resolution: pending
- Resolution evidence or operator record: pending
