# Review Finding: EXPL-REV-001-D-S02-F002

- Finding ID: `EXPL-REV-001-D-S02-F002`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Frame and GroupBox embedding is a façade over a closed, tagged Panel
  union rather than a reusable behavior-composition model.
- Consequence: Each new control must either add subtype state and rendering
  branches to Panel/App or introduce a second dispatch mechanism. Concrete
  wrapper identity is discarded during traversal, so the current model does
  not provide a coherent extension seam for the planned control catalog.
- Evidence:
  - `docs/specifications/implementation-baseline-v0.md:53-58` describes
    behavior reuse through embedding and a single node.
  - `docs/specifications/go-api-v0.md:226-256` presents Frame and GroupBox as
    Panel-embedding wrappers.
  - `panel.go:31-48` stores kind-specific inset, title, and border state
    directly on Panel.
  - `panel.go:50-104` leaves the wrappers with only an embedded Panel and
    initializes subtype state through `newPanel`.
  - `panel.go:232-237` exposes children solely as `[]*Panel`.
  - `app.go:268-329` traverses Panels, while `app.go:355-400` paints
    Panel-held border and title state through central special cases.
  - `docs/research/ui-toolkit-lessons.md:100-124` favors shallow capabilities
    and avoiding hidden special-case coupling.
- Recommendation: Select one extensible behavior model now: either a core
  node with composable measure/render/input capabilities, or a sealed internal
  behavior descriptor and dispatch seam. Keep control-specific state and
  behavior with the concrete control while retaining one canonical node
  identity.
- Related lenses: control architecture, rendering, public extension model
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S03-F001`; the tagged
  Panel state, concrete traversal, and central paint branches are the
  implementation form of the same unresolved extension seam.
- Resolution: pending
- Resolution evidence or operator record: pending
