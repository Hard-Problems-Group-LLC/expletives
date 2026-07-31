# Review Finding: EXPL-REV-001-D-S08-F002

- Finding ID: `EXPL-REV-001-D-S08-F002`
- Reviewer identity: `/root/design_performance`
- Seat and lens: Seat 8 — Performance
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Every successful construction or mutation synchronously renders and
  clones a complete snapshot under the App's exclusive lock; there is no
  construction batch, transaction, invalidation coalescing, or damage
  boundary.
- Consequence: Cost is up to `O(controls × surface)` per publication and
  `O(controls² × surface)` during incremental construction. A legal tree of
  4,096 overlapping 240×120 Panels causes approximately 241,650,892,800 cell
  assignments across mandatory construction publications, before clone and
  garbage-collection costs. Concurrent callers remain blocked throughout.
- Evidence: `docs/specifications/go-api-v0.md:407-417`;
  `panel.go:139-185`; `app.go:200-235`, `app.go:268-353`;
  `types.go:267-283`.
- Recommendation: Add a specified bounded construction or update transaction
  or owner-turn coalescing mechanism, separate state mutation from rendering
  and publication, and establish measured maximum-scale latency and
  allocation budgets. Otherwise lower the advertised geometry and control
  limits to a demonstrated viable envelope.
- Related lenses: Reliability/concurrency; maintainability; testability.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F003`; synchronous
  full rendering and cloning on every mutation is the performance consequence
  of the same absent transaction/invalidation boundary.
- Resolution: pending
- Resolution evidence or operator record: pending
