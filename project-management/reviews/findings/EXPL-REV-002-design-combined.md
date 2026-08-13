# EXPL-REV-002 — Combined Table Improvements Findings

- Review ID and stage: EXPL-REV-002 Design
- Date: 2026-08-12
- Input integrity identifier:
  `sha256:12f99c58b8ca12a03812c476b32d429367610325c25855b1112a2fb498960ca2`
- Blockers: none
- Related brief:
  [`EXPL-REV-002`](../packets/EXPL-REV-002-table-improvements-design-v1.md)

## Seat 2 — Orthogonality And Design Elegance

Overall recommendation: proceed with five separate layers—canonical schema
and data, derived column presentation, one exclusive selection policy,
independent optional features, and per-instance visual-role mappings. Keep
Table/DataGrid as leaves and DataGrid as the shared table core plus editing.

1. **Major — Independent capabilities and exclusive policies must not share
   one undifferentiated option bag.** The current API already models selection
   as a typed policy while the Columns affordance is independent. A flat list
   makes invalid states easy and grows a collision matrix. Use
   `Features []TableFeature` plus one typed selection style.
2. **Major — The internal Columns affordance cannot truthfully be an ordinary
   Button Control.** A child Button would have its own identity, parent,
   focus, routing, and snapshot contract and would force a leaf-ownership
   break. Define Body/Columns semantic parts under the parent Table identity
   while reusing normal Button presentation and press semantics.
3. **Major — Canonical schema, presentation, and visual geometry must remain
   separate.** Hiding, ordering, wrapping, sorting, and range membership must
   not mutate canonical caller rows/columns. Normalize and publish the derived
   keyed state atomically.

## Seat 4 — Customizability

Overall recommendation: expose orthogonal customization axes, compatible
defaults, stable-keyed presentation, and semantic StyleID/Theme ownership of
colors.

1. **Major — Split feature and selection configuration.** Copy and bound
   features, default omitted selection to current Single behavior, retain
   `RequireSelection`, and reject it with None.
2. **Major — Store visibility, display order, and Clip/Wrap/Hang as exact
   presentation state.** Preserve surviving keys across schema refresh, append
   new keys visible/Clip, retain hidden-column sorting, wrap body only, keep
   existing explicit width policy, and retain single-line DataGrid editing.
3. **Moderate — Customize fixed per-instance visual roles through StyleID.**
   Theme continues to own resolved colors. Use unique catalog role IDs and
   catalog-local named-color DropDowns; a public ColorPicker is separate scope.

## Seat 5 — Client Ease Of Use

Overall recommendation: preserve current behavior when new options are
omitted, keep the controls as leaves, and make the new compound interaction
keyboard complete.

1. **High — Prefer a discoverable typed API.** A feature set plus selection
   enum is easier to construct and migrate than a heterogeneous list. Preserve
   current Single behavior unless the operator explicitly requests clearing
   the sole selection with Space.
2. **High — Make Columns feel like a standard compound control.** Use a
   two-row normal Button appearance, Body/Columns logical Tab parts, Alt-C and
   Enter/Space access, one scrolling/searchable inventory, fixed editing
   controls, atomic OK, and Cancel/Escape. Retain hidden sorting, append new
   keys, wrap body, keep single-line editing, and use stable range endpoints.
3. **Medium — Give the interactive inspector usable minima.** Treat 3:1 as a
   post-minimum grow ratio, label colors “Visual roles,” and isolate changes
   through per-instance role IDs and local DropDowns.

## Seat 8 — Performance

Overall recommendation: proceed only with explicit reflow complexity, cache
invalidation, and bounded evidence. Existing model limits make canonical-model
virtualization unnecessary.

1. **Major — Cache logical-to-visual row geometry.** Key row heights and
   prefix starts by model generation, presentation digest, and width vector;
   use binary search and paint only visible fragments. Do not flatten all
   wrapped lines or remeasure on scrolling, focus, selection, or color change.
2. **Moderate — Keep large-schema editing fixed-tree and batched.** One
   O(columns) draft plus one inventory and fixed controls avoids hundreds of
   child controls. Apply once on OK. Use discrete color commits and one
   Transaction for related live changes.
3. **Moderate — Keep automation compact.** Publish identities, counts,
   endpoints, digests, logical/visual extents, range coordinates, and action
   state. Do not serialize all presentation entries or wrapped rows.

## Seat 9 — Reliability, Concurrency, And Portability

Overall recommendation: use one serialized reducer for options, selection,
presentation, model repair, edit/dialog state, and geometry. Direct methods
delegate to Transaction forms and publish only fully valid states.

1. **Major — Version the private Columns draft.** Ignore row-only revisions,
   deterministically rebase schema changes, and require explicit Reload or
   Cancel after concurrent presentation changes. Never silently overwrite a
   newer presentation.
2. **Major — Make policy/edit/dialog transitions atomic.** Invalid DataGrid
   edits prevent dialog opening; failed changes preserve prior state. Restore
   exact internal focus when eligible and never invoke callbacks under locks.
3. **Major — Specify deterministic tiny-surface behavior.** Use checked
   extents, bounded derived row indices, first-line current visibility,
   visual-distance paging, partial body-before-shadow Button degradation, and
   portable Tab/Enter/Space access. Alt-C is supplemental.

## Convergence And Dissent

All lenses recommended split feature/selection types, compatible Single
default, leaf-owned semantic parts, derived presentation, retained hidden
sorting, body-only wrapping, single-line DataGrid editing, continuous range
endpoints, local color DropDowns, instance role IDs, atomic mutations, cached
geometry, fixed-tree dialogs, and compact automation.

Two lenses favored allowing every column to be hidden when ColumnsAction is
available; two favored rejecting the last hide; performance found no material
difference. Four lenses favored unchanged width allocation; the client lens
favored longest-word auto width for wrapped columns. The facilitator
recommended one visible column for reliability and unchanged widths for
compatibility. Those recommendations were adopted by the operator's
preapproval of `EXPL-PROP-003`.
