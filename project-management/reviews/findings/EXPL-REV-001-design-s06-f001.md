# Review Finding: EXPL-REV-001-D-S06-F001

- Finding ID: EXPL-REV-001-D-S06-F001
- Reviewer identity: `/root/design_testability`
- Seat and lens: Seat 6 — Testability: determinism, observability, state control, automation parity, unit and integration seams, reproducibility, and failure diagnosis
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: There is no owner-atomic update operation, so one logical transition exposes intermediate states; `Scene.Resize` publishes the surface change and then the outer-bounds change separately, making observation timing-sensitive.
- Consequence: A test or automation observer can capture a partially applied logical transition instead of one correlated snapshot.
- Evidence:
  - `docs/specifications/go-api-v0.md:407-421`
  - `internal/demo/scene.go:193-200`
  - `project-management/reviews/packets/EXPL-REV-001-foundation-design-v1.md:360-364`
- Recommendation: Add a batch, commit, or controller-transition mechanism that produces one correlated snapshot, plus a test with a racing observer.
- Related lenses: orthogonality, client ease of use, reliability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F003`; `Scene.Resize`
  is a concrete instance of the same missing atomic update/publication
  boundary.
- Resolution: pending
- Resolution evidence or operator record: pending
