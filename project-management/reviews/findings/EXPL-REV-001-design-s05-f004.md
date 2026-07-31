# Review Finding: EXPL-REV-001-D-S05-F004

- Finding ID: EXPL-REV-001-D-S05-F004
- Reviewer identity: `/root/design_client_ease`
- Seat and lens: Seat 5 — Client ease of use: discoverability, useful defaults, error quality, lifecycle clarity, API ergonomics, MVC/MVVC fit, and resistance to common misuse
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: `expletivesctl` uses a ten-second whole-operation default that is shorter than the server's thirty-second execution budget.
- Consequence: The client can time out while a valid server operation is still executing, leaving its result uncertain.
- Evidence:
  - `docs/specifications/automation-protocol-v1.md:917-920`
  - `docs/specifications/automation-protocol-v1.md:718-725`
  - `docs/specifications/automation-protocol-v1.md:761-778`
  - `docs/specifications/automation-protocol-v1.md:967-975`
  - `docs/specifications/automation-protocol-v1.md:960-965`
- Recommendation: Separate dial and operation timeouts, derive the operation budget from `hello` plus a margin, budget key sequences appropriately, and print the request ID and a recovery command when the outcome is uncertain.
- Related lenses: reliability, testability, maintainability
- Disposition: confirmed
- Disposition rationale: The documented CLI default is one 10-second
  dial-plus-operation deadline, while normal server execution may receive 30
  seconds and client timeout is explicitly indeterminate.
- Resolution: pending
- Resolution evidence or operator record: pending
