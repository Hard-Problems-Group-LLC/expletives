# Bug: Focused TextAreas Matched The Default Application Canvas

- ID: `EXPL-BUG-011`
- Status: Resolved
- Priority: High consumer usability
- Reported: 2026-08-11T10:28:16-07:00
- Resolved: 2026-08-11T10:41:18-07:00
- Reporter: radioradio operator
- Owner: Codex
- Related work: Phase 20 Slice 20.0; Text And Numeric Input API v0

## Symptom And Impact

The default Theme painted `application.root`, focused editable TextAreas, and
focused read-only TextAreas with the same black background. A consumer using
the default canvas could not distinguish focused multiline input or output
from surrounding screen space. A downstream no-black translation exposed the
same semantic collision by mapping every black surface to one blue canvas.

## Reproduction Or Evidence

Resolving `application.root`, `text_input.focused`, and
`text_input.focused_read_only` from the prior `DefaultTheme` returned
`#000000` for all three backgrounds. The directed text-input contract already
required the normal field, focused field, and hosting canvas to use distinct
terminal palette families.

## Root Cause

The initial default text-input palette reused the base black surface for
focused roles. Later TextArea read-only work correctly introduced a distinct
semantic role but copied the same resolved color, leaving semantic evidence
honest while visual focus remained ambiguous.

## Resolution

The default focused editable surface is now black on white. Focused read-only
TextAreas are black on bright yellow. Focused validation foregrounds were
darkened for legibility on white, and focused selection uses white on blue.
All focused backgrounds are distinct from the black canvas; editable focus is
also distinct from the normal teal field and read-only focus.

## Validation

Exact Theme-resolution coverage proves every focused input role differs from
the application canvas and that normal, editable-focused, and read-only-
focused backgrounds remain distinct. A real default-Theme TextArea render
proves the focused read-only semantic and resolved cell colors agree. Focused
ordinary tests passed. At 2026-08-11T10:41:18-07:00, `GOWORK=off make verify`
passed formatting, vet, ordinary and integration tests, race detection, all
debug/release/profiling builds, and smoke checks outside the socket-restricted
sandbox. An initial sandbox run reached the Unix-socket integration tests and
failed only because that sandbox forbids the required socket option.

## History

- 2026-08-11T10:34:45-07:00 — Accepted as a Phase 20 Slice 20.0 consumer
  repair and implementation began on `feature/text-input-focus-contrast`.
- 2026-08-11T10:41:18-07:00 — The complete gate passed and the bug closed.
