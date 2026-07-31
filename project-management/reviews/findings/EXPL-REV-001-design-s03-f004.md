# Review Finding: EXPL-REV-001-D-S03-F004

- Finding ID: `EXPL-REV-001-D-S03-F004`
- Reviewer identity: `/root/design_extensibility`
- Seat and lens: Seat 3 — extensibility
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Contextless synchronous mutators conflict with the required future
  owner-marshaled API for a running multithreaded application.
- Consequence: Moving from the current lock-based slice to an owner
  goroutine can make existing calls block, deadlock during callbacks, or need
  cancellation that their signatures cannot express. Retrofitting posting and
  owner-only rules later would create ambiguous compatibility behavior.
- Evidence:
  - `docs/specifications/go-api-v0.md` exposes synchronous contextless App and
    Panel mutators and documents the current mutex-serialized implementation.
  - `docs/specifications/concurrency-and-thread-safety.md` requires documented
    marshaling, ordering, cancellation, reentrancy, and shutdown behavior for
    cross-owner work.
  - `project-management/open-questions.md` leaves the long-term owner topology
    and synchronous versus asynchronous mutation model unresolved.
  - `panel.go` implements public setters as synchronous mutation and immediate
    publication.
- Recommendation: Classify APIs now as construction-only, owner-safe,
  synchronously marshaled, asynchronously posted, or rejected during running
  state. Reserve or introduce a context-aware submission path before client
  code depends on every mutator remaining contextless and synchronous.
- Related lenses: concurrency, client ease of use, maintainability, MVC/MVVC
  integration
- Disposition: confirmed
- Disposition rationale: The concurrency specification requires explicit
  owner-safe post/call categories and cancellation semantics, while current
  mutators are contextless synchronous calls and the long-term owner model is
  still open.
- Resolution: pending
- Resolution evidence or operator record: pending
