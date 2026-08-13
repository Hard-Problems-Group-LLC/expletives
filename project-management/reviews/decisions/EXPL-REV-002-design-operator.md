# EXPL-REV-002 — Table Improvements Operator Decision

- Date: 2026-08-12
- Decision type: operator decision and Panel closure
- Review ID and stage: EXPL-REV-002 Design
- Operator: project operator
- Facilitator: Codex
- Related packet:
  [`EXPL-REV-002`](../packets/EXPL-REV-002-table-improvements-design-v1.md)
- Packet integrity identifier:
  `sha256:12f99c58b8ca12a03812c476b32d429367610325c25855b1112a2fb498960ca2`
- Related findings:
  [combined findings](../findings/EXPL-REV-002-design-combined.md)
- Decision: approved through preapproved `EXPL-PROP-003`; implement on
  `feature/table-improvements`

## Quorum And Rounds

The operator requested a formal Board review. One design round used five
selected lenses: orthogonality/design elegance, customizability, client ease
of use, performance, and reliability/concurrency/portability. No follow-up
round is required because no blocker remains and the operator authorized the
recommended proposal and implementation.

The collaboration runtime allowed three reviewer threads. Performance and
reliability were separately labeled second-lens passes by two completed
reviewers rather than fully independent people/agents. The combined record
preserves that limitation rather than overstating independence.

## Finding Totals

- Blocker: 0
- Major/high: 8
- Moderate/medium: 4

The findings primarily constrain API separation, leaf ownership, state-layer
separation, variable-height caching, dialog revision conflicts, atomic
transitions, small geometry, per-instance Theme roles, and bounded automation.

## Required Resolutions

`EXPL-PROP-003` incorporates every converged major/high recommendation. It
resolves split advice by requiring at least one visible column for a nonempty
schema and retaining the current width allocation for wrapped columns.

Implementation must verify those resolutions. A material departure requires
an explicit operator decision; ordinary identifier refinement and mechanical
delivery do not require reconvening the Panel.

## Deferrals Or Accepted Risk

- A public reusable ColorPicker is explicitly outside this proposal and may be
  proposed separately.
- Allowing a zero-visible-column presentation and a new preferred/automatic
  wrapped-width policy are not accepted behavior in this phase.
- No confirmed blocker or major finding is deferred or accepted as risk.

## Rationale

The proposal satisfies the requested functionality while preserving the
existing canonical model, leaf-control architecture, Theme boundary,
thread-safe Transaction model, and compact automation contract. The selected
tie-breaks favor recoverability, compatibility, and deterministic resource
behavior. The operator's direct instruction to draft the result as a
preapproved proposal and begin is the approval authority; Panel headcount is
not approval.
