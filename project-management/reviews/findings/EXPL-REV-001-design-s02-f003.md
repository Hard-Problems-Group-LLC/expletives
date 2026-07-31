# Review Finding: EXPL-REV-001-D-S02-F003

- Finding ID: `EXPL-REV-001-D-S02-F003`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: State mutation, full rendering, snapshot retention, and observer
  publication are fused into every setter, with no coherent update or
  invalidation boundary.
- Consequence: Multi-control updates expose partially applied UI states,
  generate redundant full renders and history entries, and make request
  correlation attach only to the final publication while observers can see
  preceding intermediate frames. Layout and common-control work would deepen
  this coupling.
- Evidence:
  - `docs/specifications/go-api-v0.md:308-324` requires every successful
    setter to publish a complete snapshot.
  - `docs/specifications/go-api-v0.md:407-417` explicitly permits intermediate
    snapshots during multi-setter handler work.
  - `docs/specifications/go-api-v0.md:625-640` places mutation, input,
    rendering, and snapshot history behind one mutex and performs rendering
    synchronously.
  - `panel.go:239-313` mutates state and immediately invokes `publishLocked`.
  - `app.go:200-219` renders, retains, and wakes observers as one operation.
  - `internal/demo/scene.go:193-201` resizes the App and outer Panel through
    two separately publishing calls.
  - `project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md:760-776`
    instead anticipates coalesced invalidation followed by one
    measure/arrange/paint/publication cycle.
  - `docs/specifications/go-api-v0.md:812-826` defers the transaction or
    owner-call API needed to express that cycle.
- Recommendation: Define a minimal owner-scoped update transaction or render
  barrier before layouts and common controls expand: accept related mutations,
  coalesce invalidation, and publish exactly one resulting frame and
  completion. Keep individual setters as convenience operations routed
  through that seam.
- Related lenses: concurrency, atomic snapshots, layout, automation
  correlation, performance
- Disposition: confirmed
- Disposition rationale: The contract and implementation require immediate
  full publication from each mutation and explicitly permit intermediate
  handler snapshots, with no transaction or invalidation boundary. The cited
  proposal path does not exist, but the other frozen citations are sufficient.
- Resolution: pending
- Resolution evidence or operator record: pending
