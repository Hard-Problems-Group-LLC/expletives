# Review Finding: EXPL-REV-001-D-S02-F007

- Finding ID: `EXPL-REV-001-D-S02-F007`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The generic public automation package embeds `expletives-test`
  identity and fixture command policy.
- Consequence: A reusable library server created with omitted options
  advertises a secondary-product identity and commands it may not implement.
  Fixture vocabulary changes consequently require changes in the generic
  automation package.
- Evidence:
  - `docs/product-goals.md:14-76` identifies expletives as the reusable
    primary product and `expletives-test` as a secondary consumer.
  - `docs/specifications/automation-protocol-v1.md:321-324` says command
    inventory is application-selected.
  - `docs/specifications/automation-protocol-v1.md:871-894` nevertheless
    defaults the public server to `expletives-test` and fixture commands.
  - `automation/protocol.go:58-62` exports fixture-specific command constants
    from the generic package.
  - `automation/server.go:88-100` installs test-application and
    fixture-command defaults.
  - `automation/server.go:466-475` also hardcodes the quit-command binding.
- Recommendation: Require host-supplied application identity and command
  inventory, or default to a generic identity with no application commands.
  Move fixture constants and defaults into `internal/demo` or
  `cmd/expletives-test`; make lifecycle shutdown an explicit toolkit semantic
  or configured host command.
- Related lenses: package boundaries, secondary-product isolation, protocol
  extensibility
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S01-F002`; it cites the
  same generic-package fixture identity and command defaults.
- Resolution: pending
- Resolution evidence or operator record: pending
