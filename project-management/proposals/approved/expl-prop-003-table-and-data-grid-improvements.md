# Table And DataGrid Improvements

- ID: EXPL-PROP-003
- Author: Codex
- Sponsor and decision authority: project operator
- Date: 2026-08-12
- Status: Approved before implementation
- Reviewers: operator-convened five-lens `EXPL-REV-002` Panel of Experts
- Affected projects or audiences: `expletives`, `expletives-test`,
  `expletivesctl`, and consuming Go applications
- Related work:
  - [`EXPL-TASK-039`](../../tasks-in-progress.md)
  - [Phases 20–24 Table Improvements](../../development-roadmap.md#20-table-selection-foundation)
  - [`EXPL-REV-002`](../../reviews/packets/EXPL-REV-002-table-improvements-design-v1.md)
  - [Collections API v0](../../../docs/specifications/collections-api-v0.md)
  - [Limited Unicode Support](../../../docs/Limited-Unicode-Support.md)

## Problem Statement

The current `Table` and `DataGrid` controls provide stable copied schemas and
rows, row or cell current state, row selection, one-column sorting, sticky
headers, integrated scrolling, deterministic frames, typed observation, and,
for DataGrid, validated single-line cell editing. They do not let an
application compose optional tabular capabilities, let a user choose visible
columns or their order, wrap individual columns, select a continuous range,
disable selection entirely, or customize visual roles for one instance.

The combined Collections catalog screen also no longer has enough room to
demonstrate the controls as interactive public-API products. Continuing to add
static fixtures there would obscure behavior, make visual evaluation harder,
and diverge from the intended control-focused shape of `expletives-test`.

## Goals

- Add bounded, composable optional Table/DataGrid features without overloading
  the existing visual `Style`/`StyleID` terminology.
- Select exactly one Table row-selection policy: None, Single, Range, or
  Multiple, with compatible zero-value behavior.
- Add stable-keyed per-column presentation for visibility, display order, and
  Clip, Wrap, or one-space hanging-indent wrapping.
- Offer an optional full-width Columns command band and a keyboard-complete
  modal editor for large column schemas.
- Preserve canonical caller schema/data, stable identities, derived sorting,
  DataGrid validation, leaf-control ownership, thread safety, deterministic
  rendering, and bounded automation.
- Allow compatible feature, selection, presentation, and per-instance visual
  role changes atomically while a control is live.
- Give Table and DataGrid separate interactive catalog screens with Options
  and Colors Notebook pages.
- Keep the complete result usable without color and without reliance on
  terminal-specific modified-key reporting.

## Non-Goals

- Do not turn Table or DataGrid into a Container or expose their internal
  affordances as independently parented public Controls.
- Do not mutate canonical `Column` order or `TableRow` data when presentation,
  sorting, or selection changes.
- Do not add multi-column sorting, arbitrary comparators, typed cell values,
  multiline cell editing, column resizing by pointer, or mouse-only behavior.
- Do not introduce a public reusable ColorPicker in this phase. The catalog
  will compose existing DropDowns over a bounded named palette.
- Do not make raw RGB values part of Table/DataGrid state; Theme remains the
  owner of resolved colors.
- Do not expose retained cell text, validator sets, complete presentation
  arrays, or flattened wrapped output through attached automation.
- Do not change terminal encoding, Unicode-width, or physical painter policy.

## Use Cases

1. An application enables the Columns feature, lets an operator hide rarely
   used columns, moves an important column left, selects Hang for a narrative
   field, and applies the complete draft atomically.
2. A read-only Table disables selection while retaining current-row/cell
   navigation, sorting, and activation.
3. A results viewer uses a continuous range: arrows move current, Shift plus
   navigation extends the stable anchor/extent range, and sorting rederives a
   continuous displayed interval.
4. A DataGrid uses free multi-selection, wrapped display for one column, and
   validated single-line editing. Hidden columns retain their values and
   validators, and edit traversal skips them.
5. A consumer stores a user column presentation, refreshes a schema, preserves
   surviving keys, and receives new columns visibly at the end in canonical
   order.
6. A developer opens either dedicated catalog screen, changes features and
   selection policy on the Options page, changes foreground/background choices
   for Visual roles on the Colors page, and observes the live control without
   affecting unrelated catalog controls.
7. Headless or attached automation proves the same keyboard paths, semantic
   state, dialog behavior, compact digests, and exact visible frame as a human
   terminal session.

## Constraints And Assumptions

### Verified Constraints

- `Table` and `DataGrid` are non-container leaf controls. DataGrid privately
  reuses the Table model, rendering, order, widths, navigation, and viewport.
- `Column.Key` and `TableRow.Key` are stable bounded identities. `Columns()`
  and `Rows()` return caller-owned copies in canonical order.
- Current bounds are 256 columns, 4,096 rows, 16,384 aggregate Table/DataGrid
  cells, and 1 MiB aggregate copied collection text per App.
- Current zero `CollectionSelectionMode` means Single. Existing callers must
  retain that behavior when new fields are omitted.
- Current sorting is a derived stable lexicographic order over complete
  normalized cell values and never changes canonical row order.
- Current cell/header text is normalized to one-cell displayed elements and is
  single-line. Unsupported elements become the one-cell replacement glyph.
- Table/DataGrid direct mutations are concurrency-safe wrappers over
  Transactions. Programmatic mutations are silent.
- A normal raised Button needs two physical rows including its shadow. It has
  established focus, mnemonic, press-capture, command, semantic-style, and
  automation contracts.
- The Theme maps semantic `StyleID`s to resolved colors and attributes and can
  be replaced atomically in a Transaction.
- Attached automation intentionally projects compact typed evidence rather
  than retained collection models or secrets.

### Assumptions Adopted By This Proposal

- “Style” in the feature request means behavior or presentation choice. The
  public API uses `Feature`, `SelectionStyle`, `ColumnPresentation`, and
  `VisualRoles` so it cannot be confused with visual `StyleID`.
- A nonempty schema always retains at least one visible column. An empty schema
  remains valid.
- Wrap/Hang line breaking occurs after existing Width/MinimumWidth/
  MaximumWidth/Grow allocation. Callers constrain a wrapped column when they
  want wrapping to engage.
- Headers remain one clipped sticky row; wrapping applies only to body cells.
- DataGrid keeps a single-line active editor even when the committed cell is
  displayed as wrapped.
- The 3:1 catalog split is a post-minimum grow ratio, not a guarantee that the
  Notebook receives one literal quarter of every surface.

## Current Context

`table.go` owns the public schema, copied model, stable derived display order,
width allocation, rendering, row/cell focus, row selection, sorting,
Transactions, and typed details. `data_grid.go` embeds that private behavior
and adds a `textFieldBehavior` editor, validation, commit/cancel, and editable
cell traversal. The normative baseline is the Table and DataGrid portion of
[`collections-api-v0.md`](../../../docs/specifications/collections-api-v0.md).

The combined catalog fixture lives under Controls / Collections with stable
keys `collections.table` and `collections.data-grid`. It proves core
navigation, sorting, editing, validation, typed state, and reset behavior, but
does not provide live option or palette editing.

The operator first directed the desired shape in a scratch requirements pass,
then explicitly convened `EXPL-REV-002`. The Panel found no technical blocker
and converged on the approach below. The operator authorized this proposal as
preapproved and directed implementation on permanent branch
`feature/table-improvements` on 2026-08-12.

## Proposed Approach

### 1. Separate Features From Exclusive Selection Policy

Use one copied bounded feature set for independent capabilities and one typed
selection policy for the mutually exclusive category:

```go
type TableFeature string

const (
    TableFeatureColumns TableFeature = "columns"
)

type TableSelectionStyle string

const (
    TableSelectionNone     TableSelectionStyle = "none"
    TableSelectionSingle   TableSelectionStyle = "single"
    TableSelectionRange    TableSelectionStyle = "range"
    TableSelectionMultiple TableSelectionStyle = "multiple"
)
```

`TableOptions` and `DataGridOptions` gain `Features []TableFeature` and
`SelectionStyle TableSelectionStyle`. Unknown and duplicate features are
validation errors. The zero selection style derives from the existing
`SelectionMode`: empty or Single becomes Single, and Multiple becomes
Multiple. If both old and new fields are supplied, Single/Single and
Multiple/Multiple agree; every disagreement is a validation error. The legacy
field remains for source compatibility until a future major-version decision.

`RequireSelection` remains independent. It is invalid with None. Existing
Single semantics remain: Space selects current but does not clear the sole
selected row. None always retains an empty selection and Enter activates
without selecting. Multiple retains the existing free stable-key set.

Range retains stable anchor and extent row keys and derives every enabled row
between them in current displayed order. Shift-Up/Down/PageUp/PageDown/Home/End
extends the extent. Space establishes a one-row range at current. Unmodified
navigation moves current without destroying an existing range. When sorting
changes, surviving endpoints remain stable and membership is rederived so the
range remains visually continuous. Model replacement repairs a missing
endpoint from its former displayed position, scanning forward then backward.

Construction and exact replacements validate the supplied selection against
the active policy. Programmatic policy changes use one exact mutation that
also supplies `RequireSelection` and any required range endpoints/selection,
so no invalid intermediate frame is published.

### 2. Keep Column Presentation Separate From Schema

Add copied keyed presentation state:

```go
type TableColumnWrap string

const (
    TableColumnClip TableColumnWrap = "clip"
    TableColumnWrapWords TableColumnWrap = "wrap"
    TableColumnHang TableColumnWrap = "hang"
)

type TableColumnPresentation struct {
    Column  string
    Visible bool
    Wrap    TableColumnWrap
}
```

The slice order is display order and contains every canonical schema key
exactly once, including hidden columns. Nil at construction means canonical
order, all visible, Clip. A non-nil exact presentation rejects unknown,
missing, or duplicate keys, unknown wrap values, and a nonempty schema with no
visible column.

`Columns()` continues to return canonical schema order. State and an explicit
`ColumnPresentation()` getter return caller-owned presentation copies. Exact
direct and Transaction setters validate and publish the complete presentation
in one frame.

`SetRows` preserves presentation. `SetModel` retains the order, visibility,
and wrap of every surviving key; removed keys disappear; new keys append in
canonical schema order as visible/Clip. A removed and later reintroduced key
is new. `Replace` accepts exact presentation in a new companion API while the
existing signature retains the compatible default/preservation contract.

If the current column is hidden or removed, repair it to the visible column at
the same presentation index, then scan right and left. If the active sort
column is hidden, sorting remains active and the Columns dialog identifies it
as sorted. Presentation never changes the canonical data or sort value.

### 3. Add Clip, Wrap, And Hang Rendering

Clip retains current behavior. Wrap breaks normalized body text at word
boundaries within the derived visible column width and falls back to cell
boundaries for an overlong word. Hang uses the same algorithm and begins every
continuation line with one ASCII space. At width one, Hang degrades to Wrap.

One logical displayed row occupies the maximum rendered line count of its
visible cells, minimum one. Shorter cells are padded. Current, selected,
current-selected, disabled, editable, and validation backgrounds cover every
visual line. The `►[X]` marker is present on the first visual line only;
continuations use four spaces. Column separators span every visual line.

Each line honors the column alignment. For Hang continuation lines, the
indent consumes the first cell and alignment applies within the remainder.
Headers remain one clipped sticky row. Sorting compares complete cell text.

Cache one visual height and prefix start for each displayed logical row,
keyed by model generation, presentation digest, and width vector. Use prefix
sums and binary search for viewport-to-row lookup. Do not allocate or retain a
flattened array of every wrapped line; paint only visible fragments. Height-
only resize, focus, selection, scrolling, and color changes do not invalidate
the wrap geometry. Width-vector or cell-content changes may perform one
bounded full-model measurement. Use checked/saturating extent arithmetic.

Up/Down move logical enabled rows. Page keys choose the enabled row nearest
one viewport height of visual distance away. Keeping current visible shows its
first visual line and as much of the row as fits without offset oscillation.

DataGrid displays committed text with the same wrapping. Its active editor
remains the current single-line horizontally scrolling editor, painted on the
first visual line while the row retains its derived height. A successful
commit invalidates and reflows the affected geometry once.

### 4. Add The Columns Action As A Semantic Internal Part

When `TableFeatureColumns` is active, Table/DataGrid reserve a full-width
two-physical-row band immediately below the tabular viewport. The band is
inside the control's existing outer bounds and has the same width. It contains
a right-justified normal raised Button affordance labeled exactly
`&Columns...`.

Table/DataGrid remain leaf controls. The action is not an independently
parented `Button` and has no separate `ControlID`; it is a typed semantic part
under the parent identity. It reuses Button label normalization, styles,
mnemonic rendering, default pressed capture, enabled/disabled presentation,
and activation behavior. Snapshot/automation evidence identifies the focused
part and action state without pretending it is another Control.

The parent retains one App focus identity and one ordinary focus group, with
two internal parts: Body and ColumnsAction. Tab from Body moves to the action;
Tab again leaves the control. Shift-Tab reverses the order. Enter or Space on
the action opens the dialog. Alt-C is a supplemental mnemonic route; it is not
the only route. The action remains reachable for an empty row model but is not
actionable when fully clipped or when the complete control is disabled.

At a one-row band allocation, paint the Button body without its shadow. At two
rows, paint the normal body and shadow. At zero allocation it is not visible or
actionable. The tabular viewport receives the remaining height and follows
the normal small-surface/overflow contract.

Activating Columns from an active DataGrid edit attempts exactly one valid
commit. Invalid input retains edit focus and does not open the dialog. A valid
commit publishes before the dialog opens and routes at most one change command
outside App locks.

### 5. Use A Revisioned Atomic Columns Dialog

The toolkit-owned modal Dialog uses a scrolling/searchable single inventory
of column keys rather than one child-control tree per column. The selected
inventory row shows and edits that column's visibility and one of Clip, Wrap,
or Hang. Space toggles visibility. Move Left and Move Right move one display
position and disable at their corresponding boundary. Ctrl-Left/Ctrl-Right
are supplemental shortcuts. Reset restores the constructor-supplied
presentation, or canonical/all-visible/Clip when none was supplied. OK is the
default action; Cancel is the Escape action.

The dialog edits an O(columns) private draft. No table presentation changes
while it is open. OK validates, copies, derives, and publishes the complete
presentation once. Cancel/Escape discards it. Attempting to hide the last
visible column is rejected with a useful dialog message and leaves the draft
open.

The draft captures schema and presentation revisions. Row-only changes do not
conflict. A concurrent schema change rebases by dropping removed keys,
preserving surviving draft entries, and appending new keys visible/Clip in
canonical order. A concurrent external presentation change marks the draft
stale and offers Reload or Cancel; OK cannot silently overwrite it.

Closing restores ColumnsAction focus when it remains eligible, otherwise Body,
otherwise ordinary App focus repair. Destroying/hiding the owner, removing the
feature, closing the App, or modal-scope invalidation cancels the dialog and
discards its draft. No callback or command router runs under an App lock.

### 6. Add Per-Instance Visual Roles

Introduce fixed typed Table visual-role configuration whose fields are
semantic `StyleID`s. DataGrid extends it with edit roles. Empty fields use the
existing default theme roles. The exact set includes at least:

- body and border;
- header, current header, and sort marker;
- current cell, selected row, current row, and current-selected row;
- disabled, empty, loading, and error states;
- scrollbar page, arrows, thumb, focused thumb, and disabled state;
- Columns command band;
- Columns action normal, default, focused, pressed, disabled, mnemonic, and
  shadow roles; and
- DataGrid editable, focused edit, invalid edit, invalid character, and text
  selection roles.

Construction copies and validates the role IDs. Complete direct and
Transaction replacements publish atomically. Theme remains the only owner of
resolved RGB and attributes.

The catalog creates screen-specific semantic IDs. Its Colors page changes
their foreground/background definitions through a Theme replacement, leaving
attributes intact and unrelated controls unchanged.

### 7. Build Dedicated Interactive Catalog Screens

Add two commands and Controls-menu entries:

- `&Tables` routes to the Table screen;
- `Data&Grid` routes to the DataGrid screen.

Adjust conflicting Controls-menu mnemonics; prefer `Text / &Display` for the
existing text route so `T` is available to Tables and `G` to DataGrid. Remove
the Table/DataGrid fixtures from the combined Collections screen while
retaining ListBox, TreeView, DropDown, and ComboBox there.

Each new screen contains a horizontal BoxLayout with Table/DataGrid at grow 3
and a Notebook at grow 1 after explicit usable minima. Narrow surfaces follow
normal logical-minimum, clipping, and overflow rules; the Notebook is not
squeezed below a useful field width merely to maintain a literal percentage.

The Notebook has:

- **Options**: a ScrollablePanel containing feature Checkboxes, a selection-
  style RadioGroup, `Require selection`, Table focus-mode options, and later
  compatible controls. Changes use exact atomic live mutations and the panel
  mirrors normalized state after success or rejection.
- **Colors**: a ScrollablePanel with columns `Visual role`, `Foreground`, and
  `Background`. Foreground/background are catalog-local DropDown compositions
  over a bounded named palette, with visible names/swatches. Discrete choice
  commits update the instance-specific Theme definitions atomically; there is
  no hover preview.

Fixtures include long cells, explicit wrapped widths, enough rows/columns for
both scroll axes, a disabled row, sorting, every selection style, and editable
and read-only DataGrid cells with soft/hard validation.

### 8. Preserve Atomic Mutation And Concurrent Repair

Every direct method delegates to a Transaction form. Exact feature, selection,
presentation, visual-role, model, edit, dialog, and geometry transitions clone
current state, validate all cross-field invariants, normalize/repair stable
identities, derive geometry, check aggregate resources, and only then publish
one immutable state/frame.

Failed mutations leave the prior editor, dialog draft, frame sequence, and
semantic state intact. Programmatic mutations remain silent. User selection,
sort, edit commit, Columns apply, and other routed actions emit at most their
documented command after publication and outside locks.

Feature removal while ColumnsAction or its dialog is active performs the
directed focus/dialog repair in the same transaction. Changing selection
policy repairs selection and endpoints in the same transition. Programmatic
model/presentation/policy changes cancel an active DataGrid editor, preserving
the current v0 model/status-mutation rule unless the exact operation is the
user's successful pre-dialog commit.

### 9. Extend Typed Observation Conservatively

Core details expose exact bounded state needed by in-process tests, including
features, selection policy/endpoints, copied column presentation, logical and
visual row counts, focused internal part, action/dialog state, visual-role
IDs, and viewport geometry.

Attached automation remains compact. It exposes feature and selection
identities, counts, first/last visible keys, presentation/width/style digests,
logical/visual row counts, range anchor/extent, ColumnsAction focus/enabled/
pressed/open state, compact dialog collection state, and the existing editor
evidence without edit text. It does not copy schema, cells, full presentation,
validator sets, or flattened wrap lines. The intended frame proves exact
visible content, wrapping, style roles, clipping, separators, caret, and
button/dialog geometry.

Every injected event or command retains unique request correlation, explicit
outcome, and an associated frame sequence. Human, headless, and attached input
use the same semantic controller paths.

## Alternatives Considered

### One Undifferentiated Behavior-Style List

Rejected. Independent features and one exclusive policy vary on different
axes. A single list makes invalid states easy to construct, obscures runtime
setters/state, and turns future additions into a pairwise conflict matrix.

### Change Table/DataGrid Into Containers With A Child Button

Rejected. It breaks the established leaf-control ownership, focus, snapshot,
and Layout contract. The affordance is one semantic part of a compound leaf.

### Add A Separate Compound Wrapper

Rejected. The requested capability belongs to Table/DataGrid and must be
available through their stable public identities. A wrapper would fragment the
API and catalog behavior.

### Permit Zero Visible Columns

Deferred. It maximizes customization when ColumnsAction remains present, but
creates presentation validity that depends on feature eligibility and leaves
poor recovery after feature removal. The initial invariant requires one
visible column for a nonempty schema.

### Clear Sorting When Its Column Is Hidden

Rejected. Visibility is presentation and must not silently mutate sort state.
The dialog discloses the hidden sorted column instead.

### Make Wrap Change Automatic Width Measurement

Rejected for this phase. Longest-word or preferred-width measurement could
make Wrap immediately visible, but couples line breaking to width policy and
changes existing layout. Explicit Width/MaximumWidth already gives callers a
deterministic bound.

### Wrap Headers Or Use A Multiline DataGrid Editor

Rejected. Either introduces a second vertical geometry/editor contract beyond
the requested body presentation. Headers remain a stable one-row landmark and
cell values remain single-line text.

### Preserve Arbitrary Range Membership Across Sort

Rejected. That behavior is free Multiple selection, not a continuous range.
Range preserves endpoints and rederives the displayed interval.

### Add A Public ColorPicker Now

Deferred. It is a separate reusable-control contract requiring its own API,
snapshots, automation, tests, and catalog scenario. Existing DropDowns satisfy
the Table catalog need without expanding the Table Improvements program.

### Edit Shared Global Theme Roles

Rejected. It would recolor unrelated instances and catalog chrome. Per-instance
role IDs preserve the semantic Theme boundary and isolate the demonstration.

## Risks And Mitigations

- **Variable-height reflow cost:** cache row heights/prefix starts; invalidate
  only on model/presentation/width changes; paint visible fragments; benchmark
  representative narrow large models before retaining complexity.
- **Stale dialog overwrites concurrent changes:** use schema/presentation
  revisions, deterministic schema rebase, and explicit Reload/Cancel for
  presentation conflicts.
- **Invalid transient selection/editor states:** expose exact compound
  Transaction mutations and publish only after full normalization.
- **Focus traps or inaccessible internal action:** retain one global focus
  group with explicit Body/Columns parts, portable Tab/Shift-Tab and
  Enter/Space routes, supplemental mnemonic, and exact focus restoration.
- **Tiny-surface ambiguity:** allocate remaining table viewport first-class,
  degrade action body/shadow deterministically, and make fully clipped parts
  non-actionable.
- **Snapshot/automation growth:** expose compact counts/endpoints/digests and
  use the intended frame for visible truth.
- **Theme leakage:** use instance-specific StyleIDs; keep a fixed role set and
  reject missing/invalid identifiers before publication.
- **Compatibility:** new fields have compatible zero behavior; legacy
  SelectionMode remains accepted; canonical getters/order and commands remain
  stable; new exact companion APIs avoid changing existing signatures.
- **Catalog crowding:** move the controls to dedicated screens and treat 3:1 as
  a ratio after minima.

## Validation

### Normal Go Tests

- feature-set copying/bounds/unknown/duplicate validation and live atomic
  replacement;
- every selection style, compatible legacy migration, conflict rejection,
  required-selection rules, disabled rows, range extension, sorting, model
  repair, and command routing;
- exact presentation validation, getter copying, hidden/current/sort repair,
  schema preservation/rebase, new-column placement, and transaction rollback;
- Clip/Wrap/Hang exact frames for empty, narrow, width-one, long-word, aligned,
  selected/current/disabled, Unicode one-cell, scrolled, paged, and resized
  states;
- cached geometry invalidation and bounded arithmetic, with representative
  benchmarks/allocations where measurement justifies the cache;
- ColumnsAction rendering, mnemonic, internal focus traversal, press capture,
  disabled/empty/tiny-surface behavior, dialog focus entry/restoration, and
  modal lifecycle;
- large-schema dialog search/toggle/reorder/reset/apply/cancel, final-visible
  rejection, revision rebase/conflict/reload, and atomic publication;
- DataGrid pre-dialog commit/refusal, hidden-cell traversal, wrapped display,
  single-line editor, validation, commit/cancel, and concurrent replacement;
- per-instance visual-role normalization, missing IDs, live replacement,
  Theme transaction isolation, and exact semantic frame roles;
- typed core detail copying, automation projection/cloning/validation,
  response bounds, request outcomes, and non-disclosure; and
- fuzzed presentation/range/wrap/model normalization plus concurrent direct and
  Transaction replacement under the race detector.

### Catalog And Integration

- stable dedicated Table and DataGrid routes, adjusted menu mnemonics, and
  removal from the combined Collections fixture;
- 3:1 post-minimum Layout allocation, Notebook Options/Colors, scrollability,
  live normalized updates, named color choices, reset, and screen switching;
- complete `expletives-test --self-check` transitions for every important
  feature, selection, presentation, dialog, editing, and visual-role state;
- attached automation drives the same keyboard paths and observes the exact
  frame/typed state; and
- operator acceptance of presentation-sensitive wide, narrow, wrap, hanging
  indent, Columns band/dialog, and color-role states.

### Complete Project Gate

Run formatting, vet, ordinary tests, integration, race, all debug/release/
profiling builds, and smoke checks. Keep generated `build/` artifacts ignored.

## Open Questions

No design blocker remains. Exact exported identifier names, command IDs,
visual-role field names, dialog copy, and geometry constants may be refined
during specification-first implementation if they preserve this approved
behavior. Any material change to the adopted invariants returns to the project
operator; it does not require reconvening unaffected Panel seats.

## Milestones

The development roadmap is the durable owner of implementation status and
slice detail. Its delivery sequence is:

1. **Phase 20 — Table Selection Foundation:** compatible four-style selection,
   stable endpoints, interaction/repair, typed evidence, catalog state, and a
   verified checkpoint.
2. **Phase 21 — Table Column Presentation And Wrapped Geometry:** keyed
   visibility/order, Clip/Wrap/Hang, variable-height viewport behavior, and
   DataGrid integration.
3. **Phase 22 — Table Columns Action And Editor:** optional semantic action
   band, internal focus, revisioned modal draft, conflict handling, and
   DataGrid pre-open commit behavior.
4. **Phase 23 — Table Visual Roles And Interactive Catalog:** fixed
   per-instance roles, dedicated Table/DataGrid screens, live Options and
   Colors pages, self-check, and attached automation.
5. **Phase 24 — Table Improvements Integration And Acceptance:** cross-axis,
   concurrency, resource, documentation, external-consumer, full-gate, and
   operator acceptance evidence.

Each phase exits through a coherent verified ACP checkpoint. Phase 24 prepares
the branch for final merge; Release Readiness resumes as Phase 25 only after
the Table Improvements acceptance gate closes. See
[`development-roadmap.md`](../../development-roadmap.md#20-table-selection-foundation)
for exact slices, dependencies, and current state.

## Adoption And Rollout

This is a compatible additive rollout on `feature/table-improvements`.
Omitted new options preserve current behavior. Existing canonical getters,
selection mode values, model setters, sorting, commands, and automation fields
remain supported; new exact companion operations and additive typed details
carry the expanded state.

The branch will use scoped ACP checkpoints at the verified exit of Phases 20
through 23, then a final Phase 24 ACP after the complete gate and required
operator acceptance. Merge to `main` only after the Phase 24 acceptance gate.
There is no data migration or external service rollout. Reverting the feature
branch before merge restores the prior Phase 16 contract; after merge, new
features can be omitted to obtain compatible presentation.

## Decision Log

- 2026-08-12 — The project operator directed the enhancement set and required
  scratch specification work before code or normative documentation changes.
- 2026-08-12 — The operator explicitly convened a formal Panel. The five
  selected lenses found no technical blocker and recommended independent
  features plus typed selection, leaf-owned semantic action parts, derived
  keyed presentation, cached variable-height geometry, per-instance visual
  roles, and catalog-local color DropDowns.
- 2026-08-12 — The facilitator resolved split advice by requiring one visible
  column for a nonempty schema and retaining existing width allocation.
- 2026-08-12 — The project operator authorized a permanent feature branch,
  directed this proposal to be treated as preapproved, and directed
  implementation to begin. Status is therefore Approved without a second
  proposal-review wait. The Panel remains advisory and did not itself approve
  scope or accept risk.
