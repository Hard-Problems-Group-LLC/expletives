# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-08-10 — `EXPL-TASK-037` — Deliver Phase 20 release readiness.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-08-10T14:21:00-07:00
  - Scope: stabilize and audit the public Go API, maintained documentation and
    examples, compatibility policy, dependency/security evidence, all-mode
    release artifacts and provenance, debugger/profiler paths, downstream
    consumption, and clean-checkout release gates.
  - Dependencies: completed Phase 19 terminal compatibility and every other
    non-deferred catalog phase.
  - Blockers: none for Slice 20.0. Tag creation or public release remains an
    explicit operator decision and is not implied by this task.
  - Status: Slice 20.0 mechanical public-surface, catalog, specification,
    example, and test inventory is active. Slices 20.1 through 20.4 are
    planned in the development roadmap and Ubersight.
  - Consumer repair (started 2026-08-11T01:26:22-07:00): radioradio exposed
    `EXPL-BUG-009`. A width-only `MinimumSize` suppresses automatic minimum
    derivation and leaves a documented one-row `DropDown` at zero height in a
    vertical-natural Layout. On permanent branch
    `feature/dropdown-intrinsic-height`, enforce the construction-time
    one-row minimum for `DropDown` and `ComboBox`, retain caller width, add a
    layout/frame regression, update the collection contract, and run the full
    gate before ACP. No physical terminal or downstream domain operation is
    required.
  - Consumer repair completed 2026-08-11T01:34:02-07:00: `EXPL-BUG-009` is
    closed with documented one-row partial-minimum behavior and full gate
    evidence. Permanent feature-branch ACP and merge to `main` remain.
  - Native-terminal acceptance repair started
    2026-08-11T10:33:38-07:00: `EXPL-BUG-010` records that the foundational
    Layout-overflow fallback can show a contextless `[OK]` action. The initial
    Phase 19 Slice 19.13 candidate will add a clear explanation without
    changing overflow state or input semantics; later acceptance notes record
    the directed presentation refinement.
  - Native-terminal acceptance repair automated gate passed
    2026-08-11T10:40:21-07:00. Focused width/style/dismissal coverage and the
    complete `make verify` suite pass; the rebuilt native attached display is
    the only remaining acceptance row.
  - Native frame 81 showed the first repaired candidate at 29 by 42 with the
    correct centered `Layout overflow [OK]` text and exact warning style. The
    operator directed a final presentation refinement: word-wrap the
    explanation and put `[OK]` on its own line.
  - Refined candidate automated gate passed 2026-08-11T11:01:46-07:00,
    including focused wrap/geometry coverage and complete `make verify`.
    Rebuilt attached native acceptance remains.
  - Native-terminal acceptance repair completed
    2026-08-11T11:10:16-07:00. Frame 117 at 12 by 42 proves the centered
    word-wrapped explanation, separate action row, exact warning style/root
    ownership, and retained structured overflow; the operator accepted the
    result. `EXPL-BUG-010` and Phase 19 Slice 19.13 are closed.
