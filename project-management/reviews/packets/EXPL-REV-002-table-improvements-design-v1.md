# EXPL-REV-002 — Table Improvements Design Brief

- Review ID: EXPL-REV-002
- Stage: Design
- Status: Frozen
- Packet version: 1
- Frozen at: 2026-08-12T17:00:34-07:00
- Scope: Table/DataGrid behavior styles, selection policy, column
  presentation, Columns dialog, wrapping, visual roles, and catalog migration
- Non-goals: implementation, release, unrelated controls, and risk acceptance
- Trigger: direct project-operator request to convene the board
- Sponsor and operator: project operator
- Facilitator: Codex
- Selected quorum: Seats 2, 4, 5, 8, and 9
- Input integrity identifier:
  `sha256:12f99c58b8ca12a03812c476b32d429367610325c25855b1112a2fb498960ca2`
- Related proposal:
  [`EXPL-PROP-003`](../../proposals/approved/expl-prop-003-table-and-data-grid-improvements.md)

## Decision Under Review

Recommend a coherent public and internal design for:

- composable Table/DataGrid behavior options, including mutually exclusive
  selection behavior;
- a full-width `&Columns...` affordance and modal editor;
- per-column visibility, display order, and Clip/Wrap/Hang presentation;
- exactly one of None, Single, continuous Range, or free Multiple selection;
- dedicated interactive Table and DataGrid catalog screens with Options and
  Colors Notebook pages; and
- live per-instance visual-role customization through the existing Theme
  boundary.

## Frozen Alternatives And Questions

1. Unified behavior-style list or independent features plus a selection enum.
2. Compatible Single default or mandatory explicit selection.
3. `RequireSelection` and Single deselection behavior.
4. Two-row normal Button band or a degraded one-row affordance.
5. Internal action, child Button/Container conversion, or wrapper.
6. One focus stop or explicit Body/Columns internal focus parts.
7. Whether all columns may be hidden.
8. Whether hiding the sorted column retains or clears sorting.
9. Placement of newly introduced schema keys in saved presentation.
10. Body-only or header wrapping.
11. Existing or wrap-specific automatic width measurement.
12. Single-line or visually wrapped DataGrid editor.
13. Stable membership or stable continuous endpoints across sorting.
14. Catalog-local color DropDowns or a new public ColorPicker.
15. Shared global roles or per-instance visual-role StyleID overrides.

## Acceptance Criteria

The recommendation must preserve compatible omitted-option behavior, stable
canonical identities, keyboard completeness, leaf ownership, bounded large-
schema behavior, deterministic variable-height geometry, DataGrid validation,
atomic concurrency-safe mutation, compact automation, and a credible live
catalog update path. Operator preferences must remain distinguishable from
design facts.

## Roster And Review Contract

- Seat 2: Orthogonality and design elegance
- Seat 4: Customizability
- Seat 5: Client ease of use
- Seat 8: Performance
- Seat 9: Reliability, concurrency, and portability

Each lens returned one overall recommendation and no more than three
risk-ranked findings plus any blocker. Review was read-only. The collaboration
runtime allowed three reviewer threads; Seats 8 and 9 were therefore conducted
as explicitly separate second-lens passes by the Seat 4 and Seat 2 reviewers.
Their findings remained separately labeled. This limits reviewer independence
but does not change the selected five-lens scope.

## Source Boundary

The reviewed scratch input was ignored and non-normative. Its hash above
identifies the exact review snapshot. The adopted durable result is
`EXPL-PROP-003`; this brief and the combined findings preserve the review
evidence without making the scratch file authoritative.
