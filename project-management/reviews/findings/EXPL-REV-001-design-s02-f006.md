# Review Finding: EXPL-REV-001-D-S02-F006

- Finding ID: `EXPL-REV-001-D-S02-F006`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Physical terminal input decoding is implemented in the
  `expletives-test` command instead of the terminal adapter or another
  reusable input boundary.
- Consequence: Applications must duplicate terminal byte and escape-sequence
  parsing, and human/automation parity depends on application-specific
  translation rather than one toolkit-owned normalization path.
- Evidence:
  - `project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md:229-262`
    assigns terminal decoding to the adapter and keeps widgets independent of
    raw bytes.
  - `docs/research/ui-toolkit-lessons.md:30-38` and
    `docs/research/ui-toolkit-lessons.md:52-74` describe a layered,
    centralized keyboard pipeline.
  - `internal/terminal/presenter_linux.go:170-202` exposes raw `[]byte` from
    `ReadReady`.
  - `cmd/expletives-test/main.go:261-270` reads those bytes and invokes
    command-local conversion.
  - `cmd/expletives-test/main.go:385-430` implements CSI, control-byte, and
    ASCII decoding in the secondary product.
- Recommendation: Put bounded stateful terminal decoding behind the reusable
  adapter/input seam and have the composition root forward normalized
  `KeyEvent` values into App. Keep terminal ownership and signal handling in
  the composition root, but remove escape-byte grammar from the demo command.
- Related lenses: terminal boundary, input normalization, reusable toolkit
  architecture
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S01-F001`. The cited
  `project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md`
  does not exist in the frozen inventory, but the remaining code and research
  citations substantiate the same terminal-decoder ownership gap.
- Resolution: pending
- Resolution evidence or operator record: pending
