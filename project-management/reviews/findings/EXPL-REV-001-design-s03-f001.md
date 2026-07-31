# Review Finding: EXPL-REV-001-D-S03-F001

- Finding ID: `EXPL-REV-001-D-S03-F001`
- Reviewer identity: `/root/design_extensibility`
- Seat and lens: Seat 3 — extensibility
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: blocker
- Claim: The public extension seam remains unresolved while the foundation
  freezes a built-in-only `*Panel` tree.
- Consequence: Continuing into the common-control catalog would turn the
  current closed hierarchy into a compatibility constraint before the project
  has defined how third-party controls obtain parent/container capability,
  participate in rendering and events, expose snapshots, and obey lifetime
  rules.
- Evidence:
  - `project-management/open-questions.md` leaves Q-002, the exact Go
    extension mechanism, unresolved.
  - `project-management/decision-log.md` records the need for an extension seam
    without selecting it.
  - `docs/specifications/control-catalog.md` plans a broad control hierarchy
    and application-defined controls.
  - `docs/specifications/go-api-v0.md` requires concrete `*Panel` parents and
    exposes only the built-in Panel, Frame, and GroupBox construction model.
  - `panel.go` and `app.go` represent and render a closed Panel tree with
    built-in special cases.
- Recommendation: Resolve Q-002 with a minimum extension contract covering
  container and node capability, rendering, events, semantic snapshots, and
  lifetime before adding common controls. If application-defined controls are
  intentionally unsupported in this version, state that explicitly and
  provide a credible migration boundary for introducing them later.
- Related lenses: orthogonality, maintainability, customizability, public API
  compatibility
- Disposition: confirmed
- Disposition rationale: The extension question remains explicitly open while
  the public constructors, tree, snapshots, and renderer expose only the
  built-in Panel model; this conflicts with the packet's future-control
  compatibility criterion.
- Resolution: pending
- Resolution evidence or operator record: pending
