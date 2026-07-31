# Review Finding: EXPL-REV-001-D-S02-F008

- Finding ID: `EXPL-REV-001-D-S02-F008`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Copying an exported Panel creates a detached mutable alias that
  retains the same identity and App pointer and can publish unrelated
  snapshots.
- Consequence: A setter can report success and increment the frame sequence
  while the canonical control tree and intended frame remain unchanged. One
  control ID can therefore designate divergent mutable state.
- Evidence:
  - `docs/specifications/go-api-v0.md:157-160` relies only on a caller
    instruction not to copy Panel values.
  - `panel.go:31-48` stores mutable node state directly in the exported
    concrete value.
  - `panel.go:139-152` validates canonical pointers only when a value is used
    as a construction parent.
  - `panel.go:239-313` mutates receiver fields and publishes without
    confirming that the receiver is the canonical node in `app.controls`.
  - `panel_test.go:75-85` covers rejection of a copied parent but not mutation
    through a copied receiver.
  - Reproduction observation: `copy := *panel; copy.SetStyle(...)` mutates
    only the copied fields while calling `copy.app.publishLocked`; traversal
    still reaches the original Panel registered in the App.
- Recommendation: Make public controls opaque, copy-safe handles to canonical
  internal nodes, so copying aliases the same state. If that model is
  deferred, canonical-check every receiver before reads and mutations and add
  an uncopyable diagnostic guard, while recognizing that the guard alone is
  not runtime enforcement.
- Related lenses: identity invariants, Go API ergonomics, snapshot correctness
- Disposition: confirmed
- Disposition rationale: Canonical-pointer validation is construction-only;
  ordinary receivers mutate their own fields and publish through the retained
  App pointer, so a copied Panel can produce an unrelated publication.
- Resolution: pending
- Resolution evidence or operator record: pending
