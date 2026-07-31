# Review Finding: EXPL-REV-001-D-S06-F003

- Finding ID: EXPL-REV-001-D-S06-F003
- Reviewer identity: `/root/design_testability`
- Seat and lens: Seat 6 — Testability: determinism, observability, state control, automation parity, unit and integration seams, reproducibility, and failure diagnosis
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: The frozen verification lacks coverage across a real human-terminal process boundary.
- Consequence: Fragmented terminal input, Ctrl-C and signal handling, window-size changes, output failure, final snapshot and exit behavior, and terminal restoration are not proven end to end.
- Evidence:
  - `docs/specifications/expletives-test.md:370-392`
  - `docs/specifications/expletives-test.md:397-419`
  - `cmd/expletives-test/main.go:227-275`
  - `cmd/expletives-test/main.go:317-355`
  - `cmd/expletives-test/main.go:385-430`
  - `cmd/expletives-test/main_test.go:15-83`
  - `internal/terminal/presenter_linux_test.go:403`
- Recommendation: Add a real `expletives-test` PTY process harness covering fragmented input, Ctrl-C and signals, `WINCH`, output failure, final snapshot and exit behavior, and terminal restoration.
- Related lenses: reliability, security, client ease of use
- Disposition: confirmed
- Disposition rationale: The cited tests cover an in-process headless loop and
  adapter-level PTY lifecycle, but no real interactive `expletives-test`
  process PTY harness exercises the combined input, signal, resize, output,
  final-state, and restoration path.
- Resolution: pending
- Resolution evidence or operator record: pending
