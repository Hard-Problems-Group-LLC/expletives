# Review Finding: EXPL-REV-001-D-S01-F005

- Finding ID: `EXPL-REV-001-D-S01-F005`
- Reviewer identity: `/root/design_maintainability`
- Seat and lens: Seat 1 — maintainability
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Directed documentation is internally inconsistent: the implemented
  Go API and automation protocol specifications select concrete contracts
  while higher-level requirements and proposal text still describe parts of
  those contracts as undecided.
- Consequence: A maintainer or reviewer can follow different project-directed
  documents and reach incompatible conclusions about the intended design.
  This increases orientation and review cost and makes later drift difficult
  to distinguish from intentionally deferred design.
- Evidence:
  - `docs/specifications/go-api-v0.md` specifies the current concrete public
    API and concurrency behavior.
  - `docs/specifications/automation-protocol-v1.md` specifies the current JSON
    Lines encoding and protocol surface.
  - `docs/specifications/ui-toolkit-requirements.md` still marks conflicting
    framebuffer, package, inheritance, and automation models as design input
    that must not be silently resolved.
  - The included foundational proposal and open-question records preserve
    earlier undecided encoding and API language without a uniform precedence
    marker at each conflict.
- Recommendation: Establish one explicit precedence rule for current directed
  specifications, mark superseded proposal and requirement passages as
  historical or unresolved design input, and update cross-references so each
  open question points to the authoritative current contract.
- Related lenses: client ease of use, testability, review efficiency
- Disposition: already addressed
- Disposition rationale: `AGENTS.md` and the frozen packet explicitly make the
  implemented v0/v1 specifications current and controlling while classifying
  the Draft requirements and dated proposal text as design input and
  provenance, addressing the claimed precedence ambiguity.
- Resolution: pending
- Resolution evidence or operator record: pending
