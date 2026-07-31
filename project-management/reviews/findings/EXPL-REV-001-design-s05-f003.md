# Review Finding: EXPL-REV-001-D-S05-F003

- Finding ID: EXPL-REV-001-D-S05-F003
- Reviewer identity: `/root/design_client_ease`
- Seat and lens: Seat 5 — Client ease of use: discoverability, useful defaults, error quality, lifecycle clarity, API ergonomics, MVC/MVVC fit, and resistance to common misuse
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: A command target is ambiguous between a key and a `ControlID`; the automation bridge casts a string without existence or type verification, and the toolkit exposes no lookup API.
- Consequence: Clients cannot reliably select, resolve, or validate a command target or distinguish a missing, wrong-App, or wrong-kind target.
- Evidence:
  - `docs/specifications/implementation-baseline-v0.md:162-168`
  - `docs/specifications/automation-protocol-v1.md:560-577`
  - `automation/app_bridge.go:34-49`
  - `docs/specifications/go-api-v0.md:516-518`
  - `docs/specifications/go-api-v0.md:827`
  - `types.go:224-236`
  - `app.go:31-32`
- Recommendation: Select one target namespace, expose typed fields or a typed union, have the toolkit resolve and validate targets, and provide safe read lookup with explicit missing, wrong-App, and wrong-kind results.
- Related lenses: orthogonality, extensibility, security, testability, maintainability
- Disposition: confirmed
- Disposition rationale: The wire describes a generic bounded target and the
  baseline calls it a key, while the bridge casts it to `ControlID`; App
  validates only syntax and exposes no public resolution or lookup contract.
- Resolution: pending
- Resolution evidence or operator record: pending
