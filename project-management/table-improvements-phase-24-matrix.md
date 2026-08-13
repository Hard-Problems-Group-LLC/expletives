# Phase 24 Table Improvements Integration Matrix

- Owner: `EXPL-TASK-039`
- Recorded: 2026-08-13
- Generator seed: `230024`
- Status: approved input to Phase 24, not execution evidence

This is the deterministic pairwise covering array required by Phase 23 Slice
23.5. It covers every valid pair among the listed factor values in 17 rows.
The row order is stable. A future regeneration must enumerate the factor
values in the order below, reject `selection=none` with `require=required`,
use Python-compatible MT19937 seed `230024`, sample at most 12,000 candidates
per greedy iteration, and choose the first candidate with the largest number
of uncovered pairs. The recorded array, rather than a regenerated result, is
authoritative for Phase 24.

## Factor Meanings

- `control`: the public Table or DataGrid API.
- `feature`: no optional feature, or the Columns feature and action band.
- `selection`: the exact None, Single, Range, or Multiple policy.
- `require`: optional or required nonempty selection.
- `presentation`: canonical/all-visible, reordered/all-visible, or one hidden
  non-final-visible column.
- `wrap`: the representative changed column uses Clip, Wrap, or Hang.
- `roles`: built-in roles, or unique per-instance roles introduced together
  with their Theme styles.
- `sort`: no sort or stable ascending/descending sort.
- `geometry`: `wide` is 140 by 40, `narrow` is 80 by 24, and `restored` means
  resize 140 by 40 to 80 by 24 and back to 140 by 40.
- `mutation`: direct public setters applied in declared axis order, or one
  Transaction carrying every changed axis and any required Theme.

All 17 rows expect `OutcomeApplied`, no active overflow episode at settled
geometry, canonical caller columns/rows unchanged, exact policy and
presentation state, deterministic current/selection repair, matching core and
automation digests, and no command from a programmatic mutation. A custom
role must paint only the target instance and resolve through the active
Theme. A hidden sort column remains active. Restored rows must reproduce the
original wide frame and semantic state except for monotonically increasing
snapshot sequence.

| ID | Control | Feature | Selection | Require | Presentation | Wrap | Roles | Sort | Geometry | Mutation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| P24-01 | Table | Columns | Multiple | Optional | Reordered | Clip | Custom | None | Narrow | Direct |
| P24-02 | DataGrid | Off | Range | Required | Reordered | Wrap | Default | Ascending | Wide | Transaction |
| P24-03 | Table | Off | Single | Optional | Canonical | Hang | Default | Descending | Restored | Direct |
| P24-04 | DataGrid | Columns | None | Optional | Hidden | Hang | Custom | Ascending | Restored | Transaction |
| P24-05 | Table | Columns | Range | Required | Hidden | Wrap | Custom | Descending | Wide | Direct |
| P24-06 | DataGrid | Off | Single | Required | Canonical | Clip | Default | None | Narrow | Transaction |
| P24-07 | DataGrid | Off | Multiple | Required | Hidden | Clip | Default | Descending | Restored | Transaction |
| P24-08 | Table | Off | None | Optional | Canonical | Wrap | Custom | None | Wide | Direct |
| P24-09 | Table | Columns | Range | Required | Canonical | Hang | Default | Ascending | Narrow | Direct |
| P24-10 | Table | Columns | Single | Optional | Hidden | Wrap | Custom | Ascending | Narrow | Transaction |
| P24-11 | DataGrid | Off | Range | Optional | Reordered | Hang | Custom | None | Restored | Direct |
| P24-12 | Table | Columns | None | Optional | Reordered | Clip | Default | Descending | Narrow | Direct |
| P24-13 | DataGrid | Off | Multiple | Required | Canonical | Clip | Custom | Ascending | Wide | Transaction |
| P24-14 | DataGrid | Off | Single | Required | Reordered | Hang | Custom | Ascending | Wide | Direct |
| P24-15 | DataGrid | Columns | Multiple | Optional | Hidden | Wrap | Default | None | Restored | Direct |
| P24-16 | DataGrid | Columns | Range | Optional | Canonical | Clip | Custom | None | Narrow | Transaction |
| P24-17 | DataGrid | Columns | Multiple | Optional | Canonical | Hang | Custom | Descending | Restored | Direct |

## Targeted Matrices Outside The Covering Array

The pairwise array deliberately keeps modal and editor lifecycle out of its
base factors; treating inapplicable lifecycle values as ordinary levels would
manufacture meaningless Table/edit and feature-off/dialog cases. Phase 24
must run these deterministic targeted matrices in addition:

1. Range repair: all 3 sort values (`none`, `ascending`, `descending`) by all
   3 model transitions (`row reorder`, `remove one endpoint`, `replace both
   endpoints`), for Table and DataGrid. Expected results are exact stable-key
   interval rederivation, forward-then-backward endpoint repair, and required
   current/current fallback only when neither endpoint survives.
2. Wrapped DataGrid editing: Wrap and Hang by `valid commit`, `soft-invalid
   refusal`, `hard-invalid character filtering`, and `read-only activation`,
   at wide, narrow, and restored geometry. Expected results are one-line
   private editing, no secret/validator-set disclosure, one geometry refresh
   on successful commit, retained editor on soft-invalid refusal, ignored hard
   input, and explicit no-op on read-only activation.
3. Columns dialog: Table and DataGrid by `apply`, `cancel`, `schema rebase`,
   and `stale presentation then reload`. Expected results are one atomic
   apply/route, draft discard on cancel, stable-key rebase, and disabled OK
   until explicit Reload for a presentation conflict.

## Invalid And Excluded Combinations

- `selection=none` with `require=required` is excluded from the valid array
  and must be tested as `ErrValidation` with no publication.
- Unknown or duplicate features, an incomplete/duplicate/unknown
  presentation, an unknown wrap value, and hiding the final visible column
  must each fail atomically.
- A visual role absent from the staged Theme must fail with `ErrStyleMissing`;
  a role added by the same Transaction is valid.
- Dialog lifecycle values are excluded when the Columns feature is off. The
  supplementary Alt-C route must then be an explicit no-op without a modal.
- Editor lifecycle values other than idle are excluded for Table. DataGrid
  programmatic policy, presentation, model, feature, and role changes must
  cancel an active editor in the same publication.
- Clip is excluded from the targeted wrapped-edit matrix because its editing
  behavior is the established baseline and remains covered in the pairwise
  array; Wrap and Hang receive the cross-axis exhaustive treatment.
- Unsupported terminal width and encoding behavior is excluded from this
  semantic matrix and remains governed by the completed Phase 19 physical
  terminal matrix. Phase 24 still verifies canonical one-cell frames and the
  attached automation path.
