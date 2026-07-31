# Review Finding: EXPL-REV-001-D-S03-F002

- Finding ID: `EXPL-REV-001-D-S03-F002`
- Reviewer identity: `/root/design_extensibility`
- Seat and lens: Seat 3 — extensibility
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Automation protocol v1 has no interoperable version-transition path
  because the server speaks first with a fixed hello, requires an exact
  version, and uses strict message fields.
- Consequence: Even compatible additions can require a flag-day client and
  server upgrade. Once v1 is deployed, a server cannot negotiate a newer
  representation before sending its version-specific hello, and strict
  decoders cannot safely accept additive fields.
- Evidence:
  - `docs/specifications/automation-protocol-v1.md` defines the server-first
    hello, exact protocol-version check, and strict unknown-field behavior.
  - `automation/codec.go` performs strict JSON decoding.
  - `automation/server.go` sends the fixed-version hello before receiving a
    client request.
  - `automation/protocol.go` defines one fixed protocol version and one set of
    message structs.
- Recommendation: Define an explicit transition policy before treating v1 as
  stable: either specify compatible optional additions, add version-range
  negotiation with a version-neutral preface, support dual codecs, or use a
  separate endpoint/version-selection mechanism.
- Related lenses: protocol compatibility, client ease of use, maintainability
- Disposition: confirmed
- Disposition rationale: The server sends a version-specific hello first,
  both peers require the exact version, and strict unknown-field rejection
  makes even optional wire additions incompatible without a defined
  transition mechanism.
- Resolution: pending
- Resolution evidence or operator record: pending
