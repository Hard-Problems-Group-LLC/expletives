# EXPL-REV-003 — Combined Roadmap Findings

- Review: brief Table Improvements roadmap-structure Panel
- Packet:
  [`EXPL-REV-003 v1`](../packets/EXPL-REV-003-table-roadmap-brief-v1.md)
- Panel: Seat 1 Maintainability, Seat 6 Testability, Seat 9 Reliability /
  concurrency / portability
- Round completed: 2026-08-13
- Verdict: favorable with bounded roadmap corrections
- Blockers: none
- Disposition: all four recommendations accepted by the project operator and
  applied to the roadmap on 2026-08-13

## Combined Findings

| Rank | Finding | Evidence and consequence | Recommendation |
| --- | --- | --- | --- |
| Major | Verification and documentation are too back-loaded within Phases 21–23. | Implementing slices establish representation, geometry, modal-lifecycle, and public-API choices, while later slices currently concentrate automation, race/resource proof, catalog matrices, and documentation reconciliation. Late failures could force structural rework and leave interim ACP checkpoints incompletely documented. | Require every implementation slice to include its focused ordinary, frame, Transaction, automation-bound, resource/race, and phase-owned documentation work as applicable. Reserve aggregate evidence slices for cross-slice matrices, attached end-to-end coverage, and the complete gate. |
| Major | Phase 24 needs a strict entry and ownership rule. | “Close gaps” across every axis could let missing primary behavior escape the acceptance gates of Phases 20–23 and turn integration into an unpredictable catch-all. | Require every earlier phase gate to close before Phase 24 starts. Phase 24 may repair integration defects exposed by cross-axis testing; missing primary behavior or documentation returns to its owning phase. Narrow Slice 24.2 to consistency/index/compatibility audit and external-consumer proof. |
| Moderate | The final pairwise audit is not yet objectively reproducible. | The listed axes have no recorded factors, representative values, exclusions, or stable generation rule, so coverage could become subjective or unexpectedly expensive. | Before Phase 24, record a deterministic pairwise matrix or seeded covering-array fixture with expected outcomes and exclusions; keep targeted exhaustive tests for critical combinations such as Range with sort/model repair and Wrap/Hang with DataGrid edit/resize. |
| Minor | Early catalog behavior should survive the Phase 23 screen migration. | Phase 20 adds a maintained selection matrix, then Phase 23 moves Table/DataGrid out of Collections. Screen-coupled fixtures would create avoidable rewrite churn. | Keep fixture construction, transitions, and self-check expectations in reusable scenario helpers; Phase 23 should replace routing/layout while extending the same behavior matrix. |

## Consensus

All three seats found the phase order coherent, the checkpoints useful, and
Phase 25 correctly dependent on completion of the Table Improvements program.
No product-design change or additional phase is recommended.
