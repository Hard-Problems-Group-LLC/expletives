# Review Finding: EXPL-REV-001-D-S02-F005

- Finding ID: `EXPL-REV-001-D-S02-F005`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Reentrancy protection is encoded as a hidden context value and
  rejects dispatch to every App, including independent owners.
- Consequence: Correctness depends on callers preserving a context value that
  the type system cannot enforce. Propagation rejects safe cross-App
  composition, while copying or replacing the context bypasses the guard and
  can still deadlock.
- Evidence:
  - `docs/specifications/go-api-v0.md:520-540` requires handlers to propagate
    an App-injected context and rejects nested dispatch.
  - `docs/specifications/go-api-v0.md:625-666` acknowledges that losing the
    value or dispatching and waiting from another goroutine can deadlock, and
    that cross-App synchronous dispatch is unsupported.
  - `input.go:94-99` places the propagation obligation on handler authors.
  - `input.go:332-369` inserts a private owner value and rejects dispatch
    whenever any owner is present rather than testing the target owner.
  - `input_test.go:331-382` codifies rejection when a handler for one App
    calls another App with the propagated context.
  - `project-management/open-questions.md:38-41` still leaves owner topology
    and synchronous versus asynchronous mutation unresolved.
- Recommendation: Resolve Q-011 through an explicit owner/dispatcher
  abstraction. Provide a nonblocking post path, detect same-owner reentry
  independently of user context, permit safe inline execution or return a
  target-specific error, and order cross-App work through the target
  dispatcher. Keep context limited to cancellation and deadline propagation.
- Related lenses: concurrency, reentrancy, controller composition, deadlock
  prevention
- Disposition: confirmed
- Disposition rationale: The guard is a private context value, rejects any
  target App when propagated, and is bypassed by value-dropping contexts or
  goroutine handoff exactly as the specification acknowledges.
- Resolution: pending
- Resolution evidence or operator record: pending
