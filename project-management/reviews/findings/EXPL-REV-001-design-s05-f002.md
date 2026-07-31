# Review Finding: EXPL-REV-001-D-S05-F002

- Finding ID: EXPL-REV-001-D-S05-F002
- Reviewer identity: `/root/design_client_ease`
- Seat and lens: Seat 5 — Client ease of use: discoverability, useful defaults, error quality, lifecycle clarity, API ergonomics, MVC/MVVC fit, and resistance to common misuse
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Reusable automation zero-value defaults falsely advertise application identity `expletives-test` and fixture commands.
- Consequence: A host using the reusable automation package can advertise a secondary product and commands it does not implement.
- Evidence:
  - `docs/specifications/automation-protocol-v1.md:871-894`
  - `docs/specifications/automation-protocol-v1.md:293-324`
  - `docs/specifications/automation-protocol-v1.md:1040-1041`
  - `automation/protocol.go:58-62`
- Recommendation: Move the defaults to the command, require application identity, and define omitted commands to mean none or an explicit inventory.
- Related lenses: orthogonality, maintainability, extensibility, testability
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S01-F002`; it identifies
  the same reusable-server default identity and fixture-command policy.
- Resolution: pending
- Resolution evidence or operator record: pending
