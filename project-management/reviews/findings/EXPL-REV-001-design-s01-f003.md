# Review Finding: EXPL-REV-001-D-S01-F003

- Finding ID: `EXPL-REV-001-D-S01-F003`
- Reviewer identity: `/root/design_maintainability`
- Seat and lens: Seat 1 — maintainability
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: `SnapshotV1` conflates the toolkit's core observation model, the
  automation wire DTO, and the terminal presentation input.
- Consequence: A change needed by any one consumer can force coordinated
  changes in all three layers. Future control-specific state either bloats the
  universal core type or pushes the design toward unrestricted maps, while
  protocol versioning and terminal presentation remain coupled to internal
  representation.
- Evidence:
  - `types.go` defines the public `SnapshotV1`, frame, cell, cursor, and
    control-observation types together.
  - `docs/specifications/go-api-v0.md` makes `SnapshotV1` the core immutable
    observation contract.
  - `docs/specifications/automation-protocol-v1.md` places that snapshot shape
    directly on the versioned JSON wire.
  - The terminal presenter consumes the same snapshot/frame representation
    used for automation observation.
- Recommendation: Separate the core immutable snapshot from an explicit
  automation protocol projection DTO and from the terminal presenter's
  frame/cursor input. Define bounded, typed, versioned control-detail
  projection rather than making every layer evolve with the same struct.
- Related lenses: orthogonality, extensibility, protocol compatibility,
  terminal architecture
- Disposition: confirmed
- Disposition rationale: One public `SnapshotV1` shape is used for core
  observation, direct wire serialization, and terminal presentation, and the
  fixed control record has no typed control-specific detail seam.
- Resolution: pending
- Resolution evidence or operator record: pending
