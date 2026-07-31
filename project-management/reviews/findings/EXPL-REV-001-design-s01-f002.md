# Review Finding: EXPL-REV-001-D-S01-F002

- Finding ID: `EXPL-REV-001-D-S01-F002`
- Reviewer identity: `/root/design_maintainability`
- Seat and lens: Seat 1 — maintainability
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The reusable automation package depends on `expletives-test`
  identity and fixture command policy.
- Consequence: Changes to the secondary product's scenarios or vocabulary
  require edits to the generic automation library, and another application
  can advertise fixture commands it does not implement merely by accepting
  the package defaults.
- Evidence:
  - `docs/product-goals.md` defines `expletives-test` as a secondary public-API
    consumer, not as policy for the reusable toolkit.
  - `docs/specifications/automation-protocol-v1.md` says the application
    selects its semantic command inventory but also specifies
    `expletives-test`-specific public server defaults.
  - `automation/protocol.go:58-62` exports fixture-specific command constants
    from the generic package.
  - `automation/server.go:88-100` supplies a test-application identity and
    fixture-command defaults.
- Recommendation: Remove fixture identity, command constants, and command
  defaults from the reusable automation package. Require the host to provide
  its identity and advertised inventory, with either a generic empty default
  or validation that makes omitted host policy explicit.
- Related lenses: orthogonality, extensibility, client ease of use
- Disposition: confirmed
- Disposition rationale: The generic package exports fixture commands and
  installs `expletives-test` identity and command defaults despite the
  application-selected inventory contract.
- Resolution: pending
- Resolution evidence or operator record: pending
