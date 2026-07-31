# Review Finding: EXPL-REV-001-D-S01-F004

- Finding ID: `EXPL-REV-001-D-S01-F004`
- Reviewer identity: `/root/design_maintainability`
- Seat and lens: Seat 1 — maintainability
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The current `*Panel` tree and renderer are closed over built-in kinds,
  while the internal control/capability seam needed for later controls remains
  unresolved.
- Consequence: Each new control will require edits to the central Panel state,
  traversal, and rendering switches, increasing regression scope and making
  control behavior harder to understand and test in isolation. The same
  closed shape also leaves no coherent route for application-defined controls.
- Evidence:
  - `docs/specifications/control-catalog.md` distinguishes shared control and
    container behavior while leaving the exact Go capability model open.
  - `project-management/open-questions.md` and
    `project-management/decision-log.md` leave the foundational extension
    mechanism unresolved.
  - `panel.go` stores kind-specific Frame and GroupBox state in Panel and
    exposes children only as `[]*Panel`.
  - `app.go` traverses that concrete tree and renders kind-specific border and
    title behavior through central special cases.
- Recommendation: Define a small internal node and capability seam before
  adding common controls. Keep canonical identity and ownership in the node,
  place control-specific state and behavior behind explicit capabilities, and
  state how built-in and application-defined controls participate.
- Related lenses: orthogonality, extensibility, customizability, testability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S03-F001`; both identify
  the unresolved extension contract behind the closed built-in Panel tree and
  renderer.
- Resolution: pending
- Resolution evidence or operator record: pending
