# Review Finding: EXPL-REV-001-D-S05-F001

- Finding ID: EXPL-REV-001-D-S05-F001
- Reviewer identity: `/root/design_client_ease`
- Seat and lens: Seat 5 — Client ease of use: discoverability, useful defaults, error quality, lifecycle clarity, API ergonomics, MVC/MVVC fit, and resistance to common misuse
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: A handler can return an error, but dispatch discards that error and retains only `OutcomeFailed`.
- Consequence: `Completion.Message` is empty, so clients receive no diagnosis, and cancellation is mis-mapped.
- Evidence:
  - `input.go:94-99`
  - `input.go:332-357`
  - `input.go:74-81`
  - `docs/specifications/go-api-v0.md:542-545`
  - `docs/specifications/go-api-v0.md:599-602`
  - `automation/app_bridge.go:62-72`
- Recommendation: Define a stable failure result with a local cause and a bounded public code and message. Map cancellation and deadline outcomes distinctly without exposing raw remote errors.
- Related lenses: reliability, testability, security, maintainability
- Disposition: confirmed
- Disposition rationale: `callHandler` converts every handler error to
  `OutcomeFailed`, returns no cause, and leaves the reserved Completion message
  empty, so the automation bridge has no diagnostic to project.
- Resolution: pending
- Resolution evidence or operator record: pending
