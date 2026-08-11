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
  - Consumer contrast repair started 2026-08-11T10:34:45-07:00:
    `EXPL-BUG-011` records that the default root and focused TextArea roles
    share a black background despite the directed distinct-palette contract.
    Repair editable and read-only focused presentation with exact Theme and
    rendered-cell regressions on `feature/text-input-focus-contrast`, then run
    the complete gate before ACP and merge to `main`.
  - Consumer contrast repair completed 2026-08-11T10:41:18-07:00:
    `EXPL-BUG-011` is closed with distinct default editable/read-only focus,
    exact Theme and rendered-cell coverage, and a passing complete gate.
    Permanent feature-branch ACP and merge to `main` follow.
