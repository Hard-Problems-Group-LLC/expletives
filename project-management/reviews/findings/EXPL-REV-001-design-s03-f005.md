# Review Finding: EXPL-REV-001-D-S03-F005

- Finding ID: `EXPL-REV-001-D-S03-F005`
- Reviewer identity: `/root/design_extensibility`
- Seat and lens: Seat 3 — extensibility
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Controls live for the entire App lifetime, there is no destruction
  contract, and the App has a fixed 4096-control cap.
- Consequence: Dynamic dialogs, tabs, repeatedly rebuilt views, and other
  normal future UI patterns retain dead controls and monotonically consume
  identities until a long-running App cannot create another control.
- Evidence:
  - `types.go` defines the fixed maximum control count.
  - `panel.go` registers each newly constructed Panel in App-owned state.
  - `docs/specifications/go-api-v0.md` defines construction and immutable
    parent ownership but no removal or destruction operation.
  - `app.go` retains registered controls for App lifetime and enforces the
    fixed limit.
  - The future control and application architecture documents include dynamic
    UI patterns for which append-only lifetime is not established as an
    intentional restriction.
- Recommendation: Define logical destruction or subtree removal with
  deterministic identity/tombstone, event, snapshot, and handle behavior. If
  the foundation is intentionally append-only, redesign and document its
  lifetime and limits explicitly before dynamic controls depend on it.
- Related lenses: reliability, resource bounds, client ease of use,
  automation identity
- Disposition: confirmed
- Disposition rationale: Controls are retained for the App lifetime, IDs are
  never reused, construction stops at 4,096 controls, and no removal,
  destruction, or tombstone contract exists.
- Resolution: pending
- Resolution evidence or operator record: pending
