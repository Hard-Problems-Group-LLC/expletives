# Table Improvements v1

- Status: Directed, approved, and in implementation
- Authority: direct operator instruction, `EXPL-PROP-003`, and
  `EXPL-DEC-018`, 2026-08-12
- Scope: compatible extensions to `Table`, `DataGrid`, their shared private
  behavior, typed observation, automation, and `expletives-test`
- Extends and, where explicit, supersedes only the Table/DataGrid portions of
  [`collections-api-v0.md`](collections-api-v0.md)
- Depends on:
  [`actions-api-v0.md`](actions-api-v0.md),
  [`modals-api-v0.md`](modals-api-v0.md),
  [`navigation-chrome-api-v0.md`](navigation-chrome-api-v0.md),
  [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md),
  [`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md), and
  [`Terminal-Shortcut-Compatibility.md`](../Terminal-Shortcut-Compatibility.md)
- Proposal:
  [`EXPL-PROP-003`](../../project-management/proposals/approved/expl-prop-003-table-and-data-grid-improvements.md)

## Purpose

Table and DataGrid add orthogonal optional features, four exact row-selection
styles, stable-keyed column presentation, per-column Clip/Wrap/Hang behavior,
an optional Columns action/dialog, and fixed per-instance semantic visual
roles. The controls remain bounded, copied, thread-safe, deterministic,
keyboard complete, automation-native leaves. DataGrid continues to reuse the
shared Table behavior privately and adds only validated cell editing.

## Compatibility

Omitting every new option preserves the current Table/DataGrid behavior:

- selection is Single unless legacy `SelectionMode` requests Multiple;
- `RequireSelection`, current/selection repair, sorting, activation, copied
  getters, commands, and canonical schema/row order retain their v0 meaning;
- all columns are visible in canonical order with Clip;
- the Columns command band is absent;
- existing default semantic style roles are used; and
- existing direct methods and Transaction forms remain supported.

`CollectionSelectionMode` remains in Table/DataGrid options for source
compatibility. The new Table-only policy is authoritative when explicitly
supplied. Explicit old/new Single or Multiple values may agree; any conflict
is `ErrValidation`. Range and None require the legacy field to be empty.

## Public Types

```go
const MaxTableFeatures = 16

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

type TableSelectionPolicy struct {
    Style     TableSelectionStyle
    Require   bool
    Selected  []string
    Anchor    string
    Extent    string
}

type TableColumnWrap string

const (
    TableColumnClip      TableColumnWrap = "clip"
    TableColumnWrapWords TableColumnWrap = "wrap"
    TableColumnHang      TableColumnWrap = "hang"
)

type TableColumnPresentation struct {
    Column  string
    Visible bool
    Wrap    TableColumnWrap
}

type TableFocusPart string

const (
    TableFocusBody          TableFocusPart = "body"
    TableFocusColumnsAction TableFocusPart = "columns_action"
)
```

Feature collections are copied, bounded by `MaxTableFeatures`, reject unknown
or duplicate values, and normalize to declaration order. The only v1 feature
is `TableFeatureColumns`.

The zero `TableSelectionStyle` is accepted only as an options compatibility
input and normalizes to Single or legacy Multiple. Stored state, getters,
snapshots, and automation always contain one nonempty canonical value.

The zero `TableColumnWrap` normalizes to Clip. Exact presentation rejects
unknown, missing, or duplicate column keys, unknown wrap values, and a
nonempty schema with no visible column. A nil presentation means canonical
column order, all visible, Clip. A non-nil empty presentation is valid only
for an empty schema.

`TableFocusPart` is normalized state. Body is the default. ColumnsAction is
valid only while the Columns feature is enabled and the action has allocated,
actionable geometry.

## Option And State Extensions

`TableOptions` and `DataGridOptions` add:

```go
Features           []TableFeature
SelectionStyle     TableSelectionStyle
RangeAnchor        string
RangeExtent        string
ColumnPresentation []TableColumnPresentation
VisualRoles        TableVisualRoles // DataGridVisualRoles for DataGrid
```

`TableState` adds:

```go
Features              []TableFeature
SelectionStyle        TableSelectionStyle
RequireSelection      bool
RangeAnchor           string
RangeExtent           string
ColumnPresentation    []TableColumnPresentation
VisibleColumnCount    int
VisualRowCount        int
FocusPart             TableFocusPart
ColumnsDialogOpen     bool
VisualRoles           TableVisualRoles
```

`DataGridState` embeds the extended TableState, adds its exact
`DataGridVisualRoles`, and retains its current editor fields.

Every slice returned from state or a getter is caller owned. Constructors,
direct mutations, Transactions, state getters, core snapshots, and automation
projection never retain or expose caller slice storage.

## Exact Runtime Configuration

The controls provide direct and Transaction forms for each coherent axis:

```go
func (t *Table) SetFeatures([]TableFeature) error
func (t *Table) SetSelectionPolicy(TableSelectionPolicy) error
func (t *Table) SetColumnPresentation([]TableColumnPresentation) error
func (t *Table) SetVisualRoles(TableVisualRoles) error
func (t *Table) SetFocusMode(TableFocusMode) error

func (g *DataGrid) SetFeatures([]TableFeature) error
func (g *DataGrid) SetSelectionPolicy(TableSelectionPolicy) error
func (g *DataGrid) SetColumnPresentation([]TableColumnPresentation) error
func (g *DataGrid) SetVisualRoles(DataGridVisualRoles) error

func (tx *Transaction) SetTableFeatures(*Table, []TableFeature) error
func (tx *Transaction) SetTableSelectionPolicy(
    *Table, TableSelectionPolicy,
) error
func (tx *Transaction) SetTableColumnPresentation(
    *Table, []TableColumnPresentation,
) error
func (tx *Transaction) SetTableVisualRoles(*Table, TableVisualRoles) error
func (tx *Transaction) SetTableFocusMode(*Table, TableFocusMode) error

func (tx *Transaction) SetDataGridFeatures(*DataGrid, []TableFeature) error
func (tx *Transaction) SetDataGridSelectionPolicy(
    *DataGrid, TableSelectionPolicy,
) error
func (tx *Transaction) SetDataGridColumnPresentation(
    *DataGrid, []TableColumnPresentation,
) error
func (tx *Transaction) SetDataGridVisualRoles(
    *DataGrid, DataGridVisualRoles,
) error
```

`Table.ReplaceWithPresentation` and
`Transaction.ReplaceTableWithPresentation`, plus the corresponding DataGrid
forms, add the exact presentation slice to the existing complete replacement
arguments. The existing `Replace` forms preserve compatible surviving
presentation.

Direct methods delegate to Transactions. Multiple Transaction operations may
atomically combine model, policy, presentation, role, and other control
changes. Each operation selects the previously staged behavior, so later
operations see earlier ones in the same Transaction. Failed validation records
nothing and committed failure publishes no partial frame.

Programmatic mutations are silent. User selection changes, DataGrid commits,
and Columns-dialog apply use the existing `ChangeCommand` and route at most
once after successful publication and outside internal locks.

Programmatic policy, presentation, or model changes cancel an active DataGrid
editor in the same atomic state transition. The one exception is a successful
user edit commit immediately before opening the Columns dialog.

## Selection Policy

Current row/cell, selection, and activation remain distinct. Disabled rows
remain visible but can never be current, selected, an endpoint, or activated.
Selected keys remain in current displayed order.

### None

- Retained selection, Anchor, and Extent are empty.
- `RequireSelection` and nonempty selected/endpoints are validation errors.
- Space is handled as an explicit no-op.
- Enter activates current without selecting it.
- The row marker region retains its four-cell width but paints no brackets or
  selection mark; current still uses its non-color arrow.

### Single

- Zero or one enabled stable row key may be selected.
- Anchor and Extent are empty.
- Space selects current and never clears the sole selected row.
- Enter selects current when necessary, then activates.
- `RequireSelection` selects repaired current whenever an enabled row exists
  and no valid selection survives.

### Multiple

- Any bounded set of enabled stable row keys may be selected.
- Anchor and Extent are empty.
- Space toggles current, except `RequireSelection` prevents clearing the last
  selected row.
- Enter adds current when necessary, then activates.
- Sorting/replacement preserve surviving membership by stable key and reorder
  the returned set into new displayed order.

### Range

- Anchor and Extent are both empty, or both identify enabled stable rows.
- Selected is exactly every enabled row in the inclusive displayed interval
  between Anchor and Extent. Disabled rows inside the interval are skipped.
- At construction or exact replacement, callers may supply endpoints and an
  empty Selected value, or an exact matching Selected value. Without
  endpoints, a nonempty Selected value must be one contiguous enabled interval
  and normalizes to its first/last displayed keys. Any mismatch or
  discontinuity is `ErrValidation`.
- `RequireSelection` establishes Anchor=Extent=current when necessary.
- Space establishes a one-row range at current.
- Shift-Up/Down/PageUp/PageDown/Home/End moves current and Extent while
  preserving Anchor. Modifier reporting is not the only route: `[` and `]`
  move Extent one enabled displayed row backward or forward without wrapping.
- Unmodified row navigation moves current without changing the retained range.
- Enter establishes a one-row range only when current is outside the retained
  interval, then activates.
- Sorting preserves surviving Anchor/Extent keys and rederives the continuous
  interval in the new display order.
- Model replacement repairs each missing/disabled endpoint from its former
  displayed position, scanning forward then backward. If neither endpoint can
  survive and an enabled row exists, required Range becomes current/current;
  optional Range becomes empty.

`SetSelectionPolicy` supplies the exact Style, Require, Selected, Anchor, and
Extent together. It never publishes a transient old-style/new-selection
combination.

## Column Presentation And Repair

Presentation is independent of canonical `Column` and `TableRow` values. The
presentation slice order is display order and contains hidden entries. Hiding
does not discard cell text, editability, validators, width policy, or sorting.
Unhiding returns the column to its retained display position.

`SetRows` preserves presentation. `SetModel` preserves every surviving keyed
entry and appends new schema keys in canonical order as visible/Clip. Removed
keys disappear; reintroduced keys are new. If a preserved presentation would
have no visible column after schema repair, the first canonical column becomes
visible.

Changing or repairing presentation preserves current column when visible.
Otherwise it starts at the old presentation index, scans right for a visible
column, then left. DataGrid edit traversal considers editable visible columns
only. Active sorting remains valid while its column is hidden and is disclosed
by the Columns dialog and typed state.

`Columns()` remains canonical. `ColumnPresentation()` and State return
presentation copies.

## Body Wrapping And Geometry

Column widths continue to derive from existing Width, MinimumWidth,
MaximumWidth, Grow, header, and complete cell-text rules. Wrap does not change
width allocation. The catalog uses explicit widths/maximums to demonstrate
wrapping.

- Clip paints the first column-width cells.
- Wrap performs word wrapping, falling back to cell-boundary breaks for a word
  longer than the width.
- Hang is Wrap with one ASCII-space prefix on every continuation line. At
  width one it degrades to Wrap.

Only body cells wrap. Headers remain one sticky clipped row. Sorting uses
complete normalized cell text. Each logical row height is the maximum line
count among visible cells, minimum one. Short cells pad to that height.
Current/selection/disabled/edit styles span the whole visual row. The four-cell
row marker appears only on the first line; continuations are blank. Separators
span the row height. Each line honors column alignment; Hang aligns within the
post-indent remainder.

The shared behavior caches logical displayed row heights and prefix starts by
model generation, presentation digest, and derived width vector. It uses
binary search for visual-offset lookup and paints only visible fragments. It
never materializes all wrapped lines. Checked/saturating arithmetic bounds
content height.

Up/Down and Home/End operate on logical enabled rows. Page movement chooses an
enabled row approximately one body viewport of visual distance away. Current
visibility anchors its first visual line and shows as much of an overheight row
as fits without resize/navigation oscillation.

DataGrid's active editor remains single-line and horizontally scrolling. It is
painted on the first visual line of the cell while the logical row retains its
committed derived height. Successful commit invalidates affected geometry once.

## Columns Action And Dialog

With `TableFeatureColumns`, the control reserves a full-width two-row bottom
band inside its outer bounds. A normal raised `&Columns...` affordance is
right-justified in that band. The table viewport occupies the remaining rows.
One available band row paints the body without shadow; two paint body and
shadow; zero paints nothing and is not actionable.

Table/DataGrid remain leaves with one ControlID and two semantic focus parts:
Body and ColumnsAction. Tab moves Body→ColumnsAction→next focus group;
Shift-Tab reverses. Enter or Space activates the action. Alt-C is supplemental.
Empty row/schema states do not remove the recovery action. A disabled control
or fully clipped action is not actionable.

Activation opens a toolkit-owned modal Dialog. A DataGrid edit first attempts
one commit; invalid input prevents the dialog. The dialog contains one bounded
scrolling/searchable column inventory and fixed controls for visibility,
Clip/Wrap/Hang, Move Left, Move Right, Reset, OK, Cancel, and Reload when
needed. Space toggles visibility; Ctrl-Left/Right supplement the move buttons.
Attempting to hide the final visible column rejects with a useful message.

The dialog edits a private draft. Reset restores constructor presentation, or
canonical/all-visible/Clip when omitted. OK applies once through ChangeCommand;
Cancel/Escape discards. The draft captures schema and presentation revisions.
Row-only changes do not conflict. Schema changes rebase by removing missing
keys, preserving surviving draft state, and appending new keys visible/Clip.
External presentation changes mark the draft stale; OK is unavailable until
Reload or Cancel.

Close restores ColumnsAction focus if still eligible, otherwise Body, otherwise
normal App repair. Owner hide/destroy, feature removal, modal invalidation,
shutdown, or finalization discards the draft. No router/callback runs under an
App lock.

## Visual Roles

`TableVisualRoles` is a fixed struct of semantic StyleIDs for body, border,
header, current header, sort marker, current cell, selected row, current row,
current-selected row, disabled, empty, loading, error, scrollbar states,
Columns band, and Columns action normal/default/focused/pressed/disabled/
mnemonic/shadow roles. `DataGridVisualRoles` embeds that value and adds
editable, focused-edit, invalid-edit, invalid-character, and text-selection
roles.

Every empty field uses the existing default semantic role. Nonempty IDs are
validated and must exist in the staged Theme. Roles are copied values and may
be replaced atomically with a Theme in one Transaction. Raw colors never enter
Table/DataGrid state.

## Typed Snapshot And Automation Evidence

Core Table details add exact copied features, selection style/endpoints,
presentation, visible/visual counts, focus part, Columns action/dialog state,
and exact visual-role IDs. DataGrid retains kind-consistent nested Table detail
plus its existing editor detail. Snapshot cloning deep-copies every slice.

Automation adds only compact bounded evidence:

- feature count/identities or digest;
- exact selection style and range endpoints;
- total/visible column counts, first/last visible keys, and presentation,
  width, and visual-role digests;
- logical/visual row counts and compact viewport state;
- focused part and Columns action enabled/visible/pressed/dialog-open state;
- compact standard-dialog/inventory state; and
- existing non-disclosing DataGrid edit coordinates, length, caret, validity,
  and validator mode.

Automation never exposes retained schema/cells, complete presentation arrays,
validator character sets, editor text, or flattened wrapped content. The
intended frame proves exact visible text, wrapping, styles, clipping, row
markers, separators, caret, action, and dialog geometry. Validation rejects
wrong-kind members, invalid enums/identifiers/digests, impossible counts or
implications, and aggregate response overflow.

## Dedicated Catalog Screens

Controls / `&Tables` and Controls / `Data&Grid` route to separate screens.
The existing Text / Display route uses `Text / &Display` or another unique
mnemonic so `T` and `G` remain available. The combined Collections screen
retains ListBox, TreeView, DropDown, and ComboBox only.

Each dedicated screen has a horizontal 3:1 grow split after usable minima:

- left: the live control and Columns band;
- right: a Notebook with Options and Colors.

Options uses a ScrollablePanel with feature Checkboxes, selection-style
RadioButtons, `Require selection`, and Table focus-mode controls. Every change
uses one atomic public mutation and the controls mirror normalized state.

Colors uses a ScrollablePanel labeled `Visual role`, `Foreground`, and
`Background`. Catalog-local DropDowns choose from a bounded named palette.
The screen owns unique semantic role IDs and atomically replaces their Theme
definitions on discrete commits, preserving attributes and unrelated controls.
No public ColorPicker is added by this phase.

The fixtures cover long/wrapped text, both scroll axes, disabled rows, sorting,
all selection styles, Columns editing, and DataGrid editable/read-only soft/
hard validation. Self-check and attached automation exercise every important
transition and reset state.

## Concurrency, Failure, And Resource Rules

All state transitions serialize through the App/Transaction owner. A
candidate clones current state, validates cross-field invariants, normalizes
and repairs identities, derives geometry, checks collection/theme/transaction
bounds, then publishes once. Failure leaves state, editor, dialog draft,
snapshot sequence, and frame unchanged.

Feature, policy, presentation, model, dialog, and resize changes never start a
goroutine. The Columns dialog uses a fixed control tree and O(columns) copied
draft. Exact automation presentation requests contain no more than
`MaxCollectionColumns` entries. Color commits are discrete; visual roles are a
fixed set, never per-row/cell/column Theme IDs.

## Acceptance Criteria

- New types and zero values are compatible, copied, bounded, documented, and
  consistent across Table and DataGrid.
- Every feature/selection/presentation combination either normalizes to the
  exact directed state or returns a stable validation/capacity error without
  mutation.
- Selection None/Single/Range/Multiple, disabled/required states, sort and
  model repair, portable range keys, command routing, and concurrency pass
  normal and race tests.
- Clip/Wrap/Hang exact frames, visual geometry caching, narrow/short/large
  surfaces, alignment, paging, resize, Unicode one-cell policy, and DataGrid
  editing pass focused tests and representative benchmarks where useful.
- Columns action/dialog focus, mnemonic, press capture, revision conflicts,
  large schemas, final-visible rejection, apply/cancel/reset/reload, and tiny
  geometry are deterministic and observable.
- Per-instance visual roles and catalog color changes do not affect unrelated
  controls and remain Theme-owned.
- Core snapshot cloning, automation projection/cloning/validation/bounds,
  catalog self-check, socket drive/observe, and non-disclosure advance
  together.
- The project formatting, vet, ordinary, integration, race, all-mode build,
  and smoke gate passes, followed by operator acceptance of presentation-
  sensitive states.
