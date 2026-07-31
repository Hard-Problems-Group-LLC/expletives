# Review Finding: EXPL-REV-001-D-S09-F003

- Finding ID: `EXPL-REV-001-D-S09-F003`
- Reviewer identity: `/root/design_reliability`
- Seat and lens: Seat 9 — Reliability, concurrency, and portability
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Physical terminal escape decoding treats arbitrary read chunks as
  complete input records.
- Consequence: A supported arrow sequence split across reads can become
  Escape plus unrelated printable input; longer or unknown sequences can
  likewise trigger unintended commands. Behavior therefore varies with PTY,
  SSH, tmux, and scheduler chunking.
- Evidence: `internal/terminal/presenter_linux.go:194-202` returns arbitrary
  chunks. `cmd/expletives-test/main.go:261-265` decodes each independently,
  while `cmd/expletives-test/main.go:385-429` recognizes an arrow only when
  all three bytes are adjacent and retains no partial-prefix state. This
  undermines the xterm/screen/tmux support claim in
  `docs/specifications/implementation-baseline-v0.md:99-105`.
- Recommendation: Use a bounded incremental decoder owned by the terminal
  loop, with retained prefix state, explicit Escape disambiguation timeout,
  deterministic unknown-sequence handling, and tests across every split and
  coalescing boundary.
- Related lenses: Testability; client ease of use; security.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S01-F001`; arbitrary
  read chunking plus stateless command-local escape decoding is the same
  terminal-input boundary defect.
- Resolution: pending
- Resolution evidence or operator record: pending
