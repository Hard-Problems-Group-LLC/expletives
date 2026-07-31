# Review Finding: EXPL-REV-001-D-S04-F002

- Finding ID: EXPL-REV-001-D-S04-F002
- Reviewer identity: `/root/design_customizability`
- Seat and lens: Seat 4 — Customizability: styling, behavior policy, application-defined controls, command and event policy, terminal variation, and safe override or composition mechanisms
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The frozen tree and renderer privilege built-ins and provide no minimum boundary for application-defined controls; external code can wrap a `Panel` but cannot register custom paint, kind, state, events, or a Panel-derived node.
- Consequence: An external control is invisible to the toolkit. Adding an extension seam later changes constructors, the tree, rendering, and snapshots or creates a parallel control model.
- Evidence:
  - `panel.go:31-48`
  - `panel.go:60-104`
  - `app.go:268-329`
  - `docs/specifications/go-api-v0.md:812-823`
  - `project-management/reviews/packets/EXPL-REV-001-foundation-design-v1.md:90-108`
  - `project-management/reviews/packets/EXPL-REV-001-foundation-design-v1.md:405-442`
- Recommendation: Decide the minimum compatibility boundary: either a sealed internal node plus composition adapters for paint, state, events, and container behavior, or explicitly disavow external wrapping as a contract and state that pre-v1 signatures are mutable. Validate the choice with an external-package custom control.
- Related lenses: extensibility, orthogonality, maintainability, client ease of use, testability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S03-F001`; it describes
  the same absence of a compatible application-defined control boundary in
  the closed Panel tree and renderer.
- Resolution: pending
- Resolution evidence or operator record: pending
