# Review Finding: EXPL-REV-001-D-S08-F001

- Finding ID: `EXPL-REV-001-D-S08-F001`
- Reviewer identity: `/root/design_performance`
- Seat and lens: Seat 8 — Performance
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: blocker
- Claim: Public Frame and GroupBox titles are unbounded, while rendering
  normalizes the complete title before clipping. `Normalize` preallocates a
  string slice with capacity equal to the input byte length.
- Consequence: One valid constructor call can cause arbitrarily large CPU and
  allocation work while holding the App write lock, including process OOM.
  An oversized supported grapheme can also make a snapshot exceed the fixed
  8 MiB response limit after JSON has already been fully allocated. This
  directly violates the packet's bounded-input acceptance criterion.
- Evidence: `panel.go:17-29`, `panel.go:97-104`, `panel.go:139-185`;
  `app.go:382-389`; `internal/display/normalize.go:19-29`;
  `automation/codec.go:399-405`; packet `:438-439`.
- Recommendation: Specify and enforce byte, grapheme-count, and per-cell
  grapheme-byte bounds before construction mutates the App. Normalize and
  cache bounded title cells before taking the App lock, and prove every legal
  maximum snapshot fits the wire limit.
- Related lenses: Security; reliability; client ease of use.
- Disposition: confirmed
- Disposition rationale: Titles have no byte or grapheme bound, construction
  publishes while locked, rendering normalizes the complete string before
  clipping, and encoding allocates before enforcing the response limit. The
  shorthand packet citation resolves to the frozen packet's bounded-input
  acceptance criterion.
- Resolution: pending
- Resolution evidence or operator record: pending
