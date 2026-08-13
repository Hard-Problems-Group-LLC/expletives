# EXPL-REV-003 — Table Improvements Roadmap Brief

- Convened: 2026-08-13T10:25:15-07:00
- Sponsor: project operator
- Facilitator: Codex
- Review type: brief roadmap-structure review
- Quorum: three seats
- Selected lenses:
  - Seat 1 — Maintainability
  - Seat 6 — Testability
  - Seat 9 — Reliability, concurrency, and portability
- Response limit: at most three risk-ranked findings per seat, plus any
  blocker

## Decision Under Review

Review whether the approved Table Improvements work is divided into coherent,
correctly ordered, independently verifiable phases and slices in
[`development-roadmap.md`](../../development-roadmap.md#20-table-selection-foundation).
The proposed sequence is:

1. Phase 20 — Table Selection Foundation;
2. Phase 21 — Table Column Presentation And Wrapped Geometry;
3. Phase 22 — Table Columns Action And Editor;
4. Phase 23 — Table Visual Roles And Interactive Catalog;
5. Phase 24 — Table Improvements Integration And Acceptance; and
6. Phase 25 — resumed Release Readiness.

## Scope

- dependency order and placement of work;
- whether phase and slice exits are coherent ACP checkpoints;
- whether verification is scheduled close enough to implementation;
- duplication, gaps, or unnecessary segmentation; and
- whether Phase 24 is integration/acceptance rather than deferred feature
  implementation.

## Non-Goals

- reopen the approved product behavior in `EXPL-PROP-003` or
  `table-improvements-v1.md`;
- review the in-progress selection code;
- conduct a delivery or security review; or
- add new Table/DataGrid scope.

## Alternatives And Principal Risks

- One large Phase 20 would avoid renumbering but obscure usable checkpoints.
- More phases improve boundaries but can create administrative churn or defer
  tests and documentation too long.
- Catalog work before the Columns interaction stabilizes risks rework;
  catalog work too late weakens maintained public-consumer coverage.
- A broad final integration phase can become a hiding place for unfinished
  features unless its entry criteria are explicit.

## Acceptance Criteria

- Every approved behavior has exactly one primary implementation owner.
- Each phase can close with a usable, documented, verified product increment.
- Core/detail/automation/resource tests advance with the behavior they prove.
- Catalog and operator-sensitive acceptance occur at useful points.
- Release Readiness resumes only after the Table Improvements program closes.

## Evidence

- Current uncommitted roadmap draft on `feature/table-improvements`, sections
  20 through 25, as of the convened timestamp.
- [`EXPL-PROP-003`](../../proposals/approved/expl-prop-003-table-and-data-grid-improvements.md#milestones)
- [`Table Improvements v1`](../../../docs/specifications/table-improvements-v1.md)
- `git diff --check` passed immediately before convening.

The Panel is advisory. Findings recommend changes; they do not approve scope,
accept risk, or authorize implementation.
