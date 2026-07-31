# Review Finding: EXPL-REV-001-D-S04-F001

- Finding ID: EXPL-REV-001-D-S04-F001
- Reviewer identity: `/root/design_customizability`
- Seat and lens: Seat 4 — Customizability: styling, behavior policy, application-defined controls, command and event policy, terminal variation, and safe override or composition mechanisms
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: `Style` conflates semantic identity with concrete RGB and lacks an App theme, palette, inheritance, precedence, and degradation model.
- Consequence: The same semantic ID may conflict; themes and accessibility changes cannot be applied atomically; per-control mutations produce intermediate snapshots; and monochrome attributes cannot express non-color intent.
- Evidence:
  - `types.go:115-176`
  - `docs/specifications/go-api-v0.md:326-357`
  - `panel.go:225-275`
  - `app.go:306-317`
  - `app.go:431-436`
  - `internal/terminal/encoder.go:62-89`
  - `docs/specifications/ui-toolkit-requirements.md:571-580`
  - `docs/specifications/expletives-test.md:126-143`
- Recommendation: Decide an App semantic-style resolver, including registry defaults and missing-style semantics, content and border roles, override precedence, atomic replacement, terminal mapping, and non-color intent; otherwise mark the current design fixture-only and migration-prone.
- Related lenses: extensibility, client ease of use, testability, reliability and portability
- Disposition: confirmed
- Disposition rationale: Public Style values carry both semantic IDs and
  concrete colors directly; no App resolver, inheritance, palette replacement,
  or non-color intent contract exists in the cited surface.
- Resolution: pending
- Resolution evidence or operator record: pending
