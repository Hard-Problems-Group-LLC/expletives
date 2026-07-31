# Review Finding: EXPL-REV-001-D-S01-F001

- Finding ID: `EXPL-REV-001-D-S01-F001`
- Reviewer identity: `/root/design_maintainability`
- Seat and lens: Seat 1 — maintainability
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Physical terminal-byte decoding lives in `expletives-test` instead of
  at a reusable terminal/input boundary.
- Consequence: Every consuming application that uses the terminal presenter
  must duplicate escape-sequence and control-byte parsing, and the duplicated
  parsers can drift from the toolkit's structured key semantics. A
  read-at-a-time command parser is also the wrong owner for incremental
  sequences that may be split or coalesced by the terminal.
- Evidence:
  - `docs/specifications/expletives-test.md` describes the test application as
    a public-API consumer rather than the owner of reusable terminal behavior.
  - `project-management/decision-log.md` records the terminal adapter and the
    common semantic controller path as architectural boundaries.
  - `docs/research/ui-toolkit-lessons.md` calls for centralized, layered
    terminal-input decoding before command resolution.
  - `internal/terminal/presenter_linux.go:170-202` returns undecoded bytes from
    `ReadReady`.
  - `cmd/expletives-test/main.go:261-270` passes those bytes to command-local
    conversion, and `cmd/expletives-test/main.go:385-430` implements CSI,
    control-byte, and ASCII parsing.
- Recommendation: Move bounded, incremental byte and escape-sequence decoding
  into a reusable terminal/input component that emits toolkit `KeyEvent`
  values. Specify buffering and incomplete-sequence behavior and test
  sequences split across reads as well as multiple sequences in one read.
- Related lenses: testability, client ease of use, reliability, terminal
  portability
- Disposition: confirmed
- Disposition rationale: The cited presenter returns arbitrary byte chunks,
  while the command-local decoder has no retained prefix state; the ownership
  and split/coalesced-sequence claim is directly supported.
- Resolution: pending
- Resolution evidence or operator record: pending
