# Review Finding: EXPL-REV-001-D-S08-F003

- Finding ID: `EXPL-REV-001-D-S08-F003`
- Reviewer identity: `/root/design_performance`
- Seat and lens: Seat 8 — Performance
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Retention is internally mismatched: each of 256 protocol completion
  records retains a full snapshot even though `query_result` exposes only
  compact metadata, while the much smaller App request-ID ledger never evicts
  and permanently rejects further dispatch after 4,096 accepted operations.
- Consequence: Repeated observations can retain 7,372,800 Cell values and
  1,048,576 ControlSnapshot values in completion records alone at advertised
  maxima, duplicating separate frame caches. Despite that heavyweight
  retention, an otherwise healthy interactive App has a deterministic finite
  input lifetime.
- Evidence: `automation/protocol.go:89-108`,
  `automation/protocol.go:226-260`; `automation/server.go:535-580`,
  `automation/server.go:583-605`, `automation/server.go:625-653`;
  `app.go:189-197`;
  `docs/specifications/automation-protocol-v1.md:624-626`,
  `docs/specifications/automation-protocol-v1.md:746-759`.
- Recommendation: Retain only compact completion metadata plus frame
  sequence, give exact frames one bounded owner or cache, and define a
  coherent expiry or session policy that permits sustained human and
  automation operation without losing the documented reconciliation window.
- Related lenses: Reliability; security/resource exhaustion; testability.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F004`; it combines
  the same heavyweight completion/snapshot retention and finite App request-ID
  lifetime.
- Resolution: pending
- Resolution evidence or operator record: pending
