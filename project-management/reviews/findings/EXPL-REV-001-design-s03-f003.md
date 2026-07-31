# Review Finding: EXPL-REV-001-D-S03-F003

- Finding ID: `EXPL-REV-001-D-S03-F003`
- Reviewer identity: `/root/design_extensibility`
- Seat and lens: Seat 3 — extensibility
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: `SnapshotV1`'s `ControlSnapshot` has no typed per-control extension
  payload for future focus, selection, editing, checked, or other
  control-specific state.
- Consequence: Adding common controls will either force a breaking expansion
  of the universal snapshot shape for every new state, omit automation-visible
  semantics, or introduce unrestricted untyped maps contrary to the bounded
  typed snapshot requirement.
- Evidence:
  - `types.go` defines one fixed `ControlSnapshot` shape shared by the initial
    control kinds.
  - `docs/specifications/go-api-v0.md` makes the bounded control tree part of
    `SnapshotV1`.
  - `docs/specifications/automation-protocol-v1.md` serializes that control
    shape directly in protocol v1.
  - `docs/specifications/control-catalog.md` requires future controls with
    focus, selection, editing, checked, and other observable states.
- Recommendation: Define a bounded tagged and versioned detail union for
  control-specific semantic state, or specify a complete whole-snapshot
  version-transition policy that can add those states without untyped domain
  maps.
- Related lenses: automation testability, protocol evolution,
  customizability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S01-F003`; the missing
  typed control-detail evolution seam is one consequence of the same shared,
  fixed core/wire snapshot shape.
- Resolution: pending
- Resolution evidence or operator record: pending
