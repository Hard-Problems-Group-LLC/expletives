# Collections API v0

- Status: Directed pre-v1 contract
- Authority: Phase 16 roadmap and operator full-automatic direction,
  2026-07-31
- Scope: `ListBox`, `ComboBox`, `DropDown`, `TreeView`, `Table`, `DataGrid`,
  and their copied data models
- Depends on:
  [`selection-api-v0.md`](selection-api-v0.md),
  [`scrolling-content-api-v0.md`](scrolling-content-api-v0.md),
  [`actions-api-v0.md`](actions-api-v0.md), and
  [`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md)

## Purpose

Collection controls present bounded application-owned data while preserving
current focus, selection, expansion, sort, and scroll state by stable
identity. They are keyboard complete, safe for concurrent callers, suitable
for MVC/MVVM applications, and semantically observable without copying an
unbounded model into automation responses.

The public types are distinct controls. They share private item, selection,
viewport, popup, and editor capabilities; `ComboBox` is not publicly
substitutable for `ListBox`, and `DataGrid` is not exposed as a subclass of
`Table`. This supersedes the deep inheritance sketch in the older Draft.

## Resource And Text Bounds

```go
const (
    MaxCollectionItems          = 4096
    MaxCollectionColumns        = 256
    MaxCollectionCells          = 16384
    MaxCollectionDepth          = 64
    MaxCollectionAggregateBytes = 1 << 20
    DefaultCollectionPopupRows  = 8
    MaxCollectionPopupRows      = 64
)
```

These are allocation and snapshot-copy bounds, not terminal geometry limits.
The aggregate counts every retained key, label, status message, cell,
selection key, and editor value owned by collection controls in one App.
Existing `MaxAutomationKeyBytes`, `MaxDisplayTextBytes`,
`MaxDisplayTextCells`, and single-cell Unicode normalization apply to all
collection keys and text.

Each public replacement copies its complete input before publication. Maps,
recursive caller-owned nodes, and caller slices are never retained by
reference. Rejected replacements leave the prior state and frame unchanged.

## Shared State

```go
type CollectionStatus string

const (
    CollectionReady   CollectionStatus = "ready"
    CollectionLoading CollectionStatus = "loading"
    CollectionError   CollectionStatus = "error"
)

type CollectionSelectionMode string

const (
    CollectionSelectionSingle   CollectionSelectionMode = "single"
    CollectionSelectionMultiple CollectionSelectionMode = "multiple"
)

type ListItem struct {
    Key            string
    Label          string
    Description    string
    Disabled       bool
    DisabledReason string
}
```

The empty status and selection-mode values normalize to `ready` and `single`.
An error status requires a nonempty canonical message. A loading status uses
the supplied message or the toolkit default `Loading...`. Ready with no items
is the normal empty state, not an error.

Item keys are nonempty bounded identifiers and unique within one control.
Labels are nonempty canonical single-line display text. Descriptions are
optional canonical single-line text. Disabled items require a reason;
enabled items cannot carry one. Disabled items remain visible but are skipped
by current-row navigation and cannot be selected or activated.

`current` is the intra-control keyboard focus and is distinct from selection.
Arrow, Page, Home, and End navigation changes current without silently
changing selection. Space applies the control's selection operation. Enter
selects the current item when necessary and performs semantic activation.
This distinction is retained consistently across list, tree, table, and grid
controls.

Selected keys are stored and returned in displayed item order, never caller
order. Single mode accepts at most one selected key. Multiple mode accepts
any bounded set. When `RequireSelection` is true and an enabled item exists,
repair selects the current item if no valid selection remains.

## Stable-Identity Repair

Replacing a model without explicit current or selection overrides preserves
every surviving enabled key. If current was removed or disabled, repair
starts at its former displayed index, scans forward for an enabled item, then
scans backward. If no enabled item exists, current is empty and the control is
not focusable.

Missing or newly disabled selections are removed. Required single selection
then selects repaired current. Expansion is preserved by key for the
surviving TreeView nodes unless an exact replacement explicitly supplies a
new expansion set. Table and DataGrid sorting preserve current and selected
rows by row key, never by transient row index.

The same rules apply to direct methods, Transactions, construction, model
replacement, and removal. There is no pointer-identity fallback.

## ListBox

```go
type ListBoxOptions struct {
    ScrollablePanelOptions
    Items             []ListItem
    Current           string
    Selected          []string
    SelectionMode     CollectionSelectionMode
    RequireSelection  bool
    Status            CollectionStatus
    StatusMessage     string
    ActivateCommand   CommandID
}

type ListBoxState struct {
    Status          CollectionStatus
    StatusMessage   string
    Current         string
    CurrentIndex    int
    Selected        []string
    Offset          Point
    ItemCount       int
    EnabledCount    int
}

type ListBox struct { /* copy-safe Panel-derived leaf */ }

func NewListBox(Container, ListBoxOptions) (*ListBox, error)
func (t *Transaction) NewListBox(
    Container,
    ListBoxOptions,
) (*ListBox, error)
func (l *ListBox) Items() []ListItem
func (l *ListBox) State() ListBoxState
func (l *ListBox) SetItems([]ListItem) error
func (l *ListBox) Replace([]ListItem, string, []string) error
func (l *ListBox) SetCurrent(string) error
func (l *ListBox) SetSelection([]string) error
func (l *ListBox) SetStatus(CollectionStatus, string) error
func (l *ListBox) Focus() error
func (l *ListBox) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func (t *Transaction) SetListItems(*ListBox, []ListItem) error
func (t *Transaction) ReplaceList(
    *ListBox,
    []ListItem,
    current string,
    selected []string,
) error
func (t *Transaction) SetListCurrent(*ListBox, string) error
func (t *Transaction) SetListSelection(*ListBox, []string) error
func (t *Transaction) SetListStatus(
    *ListBox,
    CollectionStatus,
    string,
) error
```

The inherited `ScrollablePanelOptions.ChangeCommand` is routed only after a
real user selection change. `ActivateCommand` is routed on Enter or public
`Activate`; if it is empty and activation changed selection, the change
command is used. Programmatic setters never route commands.

One displayed item occupies one row. Up and Down move current by one enabled
item. Page Up and Page Down move by the current viewport height while landing
on an enabled item. Home and End move to the first and last enabled item.
Space selects the current single-mode item or toggles it in multiple mode.
Enter selects it if needed and activates it. Vertical offset automatically
keeps current visible; horizontal scrolling remains available for long labels
and descriptions. Tab and Shift-Tab leave the ListBox focus group.

## DropDown And ComboBox

`DropDown` is the selection-only one-row collapsed field. `ComboBox` is the
editable one-row field. They are separate public controls sharing the same
private popup-list model.

```go
type DropDownOptions struct {
    PanelOptions
    Items            []ListItem
    Current          string
    Selected         string
    AllowEmpty       bool
    PopupRows        int
    Disabled         bool
    DisabledReason   string
    ChangeCommand    CommandID
    ActivateCommand  CommandID
}

type ComboBoxOptions struct {
    DropDownOptions
    Text       string
    Validator *TextValidator
}

type DropDownState struct {
    Current        string
    CurrentIndex   int
    Selected       string
    SelectedIndex  int
    Open           bool
    PopupCurrent   string
    PopupSelection string
    PopupOffset    int
    ItemCount      int
    EnabledCount   int
}

type ComboBoxState struct {
    DropDownState
    Text    string
    Editing bool
    Valid   bool
}

func NewDropDown(Container, DropDownOptions) (*DropDown, error)
func (t *Transaction) NewDropDown(
    Container,
    DropDownOptions,
) (*DropDown, error)
func (d *DropDown) Items() []ListItem
func (d *DropDown) State() DropDownState
func (d *DropDown) SetItems([]ListItem) error
func (d *DropDown) SetSelection(string) error
func (d *DropDown) Open() error
func (d *DropDown) Close() error
func (d *DropDown) Focus() error
func (d *DropDown) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func NewComboBox(Container, ComboBoxOptions) (*ComboBox, error)
func (t *Transaction) NewComboBox(
    Container,
    ComboBoxOptions,
) (*ComboBox, error)
func (c *ComboBox) Items() []ListItem
func (c *ComboBox) State() ComboBoxState
func (c *ComboBox) SetItems([]ListItem) error
func (c *ComboBox) SetSelection(string) error
func (c *ComboBox) Text() string
func (c *ComboBox) SetText(string) error
func (c *ComboBox) Validator() *TextValidator
func (c *ComboBox) SetValidator(*TextValidator) error
func (c *ComboBox) Open() error
func (c *ComboBox) Close() error
func (c *ComboBox) Focus() error
func (c *ComboBox) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func (t *Transaction) SetDropDownItems(Control, []ListItem) error
func (t *Transaction) SetDropDownSelection(Control, string) error
func (t *Transaction) SetComboBoxText(*ComboBox, string) error
func (t *Transaction) SetComboBoxValidator(
    *ComboBox,
    *TextValidator,
) error
```

DropDown exposes copied `Items`, `State`, `SetItems`, `SetSelection`, `Open`,
`Close`, `Focus`, and `Activate` operations plus atomic Transaction forms.
ComboBox additionally exposes the single-line editor's committed text and
validator. A selected item copies its label into ComboBox text. Committing
custom text clears selected identity unless it exactly matches one enabled
label. Hard validators must accept every enabled popup item label as well as
the committed editor text; a construction or model/policy replacement that
would break that invariant is rejected atomically. Soft validators preserve
TextField's valid/invalid rendering and commit behavior.

Space, Enter, F4, or Alt-Down opens a closed DropDown. For ComboBox, Space,
F4, or Alt-Down opens choices while Enter or F2 starts editor mode. The popup
is sized to its bounded rows and widest item, clamped to the application
client area, and placed below the field when possible or above it otherwise.
Horizontal backset keeps its right edge onscreen. While open, arrows, Page,
Home, End, and Space manipulate provisional popup current/selection; Enter
commits and closes; Escape cancels and restores the opening state. Tab or a
focus transfer cancels a still-open popup before applying ordinary focus-group
navigation. Alt-X and other registered global chords retain normal global
precedence.

DropDown never starts text editing. ComboBox uses the same explicit Enter/F2
edit gate and validator semantics as TextField when its popup is closed.
An empty enabled ComboBox remains focusable and editable even though its
empty choices popup has no current item; an empty DropDown is not focusable.
Only one collection popup may be open per App. Popup ownership does not
introduce a nested event loop or a public modal. Programmatic setters, Open,
and Close are silent; user commits route the optional change or activation
command outside toolkit locks.

## TreeView

```go
type TreeNode struct {
    Key            string
    Label          string
    Disabled       bool
    DisabledReason string
    Expanded       bool
    Children       []TreeNode
}
```

TreeView has ListBox-equivalent status, current, selection mode, required
selection, change command, activation command, and scrolling options. Nodes
are copied iteratively with unique keys across the complete tree, a maximum
depth of `MaxCollectionDepth`, and a maximum total node count of
`MaxCollectionItems`.

Up and Down move through visible pre-order rows. Right expands a collapsed
branch or moves to its first enabled visible child. Left collapses an expanded
branch or moves to its nearest visible parent. Space changes selection;
Enter activates. `+`, `-`, and `*` expand, collapse, and recursively expand
the current branch within the global bounds. Expansion changes may route a
separate optional `ExpandCommand`; programmatic expansion does not.

Tree guides, branch markers, indentation, current, selection, disabled state,
and horizontal clipping use semantic styles. Expansion and current are
observable by node key and visible-row index.

## Table

```go
type Column struct {
    Key          string
    Header       string
    Width        int
    MinimumWidth int
    MaximumWidth int
    Grow         int
    Alignment    TextAlignment
    Sortable     bool
    Editable     bool
    Validator    *TextValidator
}

type TableCell struct {
    Column string
    Text   string
}

type TableRow struct {
    Key            string
    Cells          []TableCell
    Disabled       bool
    DisabledReason string
}

type TableFocusMode string

const (
    TableFocusRow  TableFocusMode = "row"
    TableFocusCell TableFocusMode = "cell"
)

type SortDirection string

const (
    SortNone       SortDirection = "none"
    SortAscending  SortDirection = "ascending"
    SortDescending SortDirection = "descending"
)
```

Column and row keys are unique. Each row contains at most one cell for a
known column; omitted cells are empty. The App-wide cell count is bounded by
`MaxCollectionCells`. Zero column width measures the maximum header/cell
width; explicit minimum/maximum constraints clamp it; positive Grow values
share spare viewport width. Text does not wrap in v0. The header is always
one sticky row and never participates in vertical scrolling.

Table supports row or cell current focus, stable row selection, and optional
single-column stable sorting. Sorting is stable and compares normalized cell
sequences, then preserves original model order for equal values. Activating a
sortable header cycles none → ascending → descending → none. Programmatic
replacement never silently reorders the caller's canonical model; display
order is derived and observable.

Up/Down/Page/Home/End move rows. In cell mode Left/Right move columns; in row
mode they scroll horizontally. Space changes row selection. Enter activates
the current row/cell. Ctrl-Home and Ctrl-End move to the first and last cell.

## DataGrid

DataGrid is the editable tabular control and shares Table's copied column,
row, sort, selection, and viewport semantics privately. It always uses cell
focus. A column is editable only when `Editable` is true. V0 cells are
single-line text editors using the optional copied `TextValidator`; richer
typed editors can be added without changing the row model.

Enter or F2 begins editing the current editable cell. Enter commits, Escape
cancels, and Tab or Shift-Tab commits then moves to the next or previous
editable cell. The field has the same unfocused, focused, invalid-soft, and
invalid-character semantic styles as TextField. Hard validation ignores
disallowed input; soft validation permits it but rejects commit until valid.
Programmatic model replacement cancels an active edit before applying stable
identity repair.

Committed user edits update the DataGrid's copied row model atomically and
route its optional change command with the grid as target. Workers and
application models remain responsible for accepting, persisting, rejecting,
or replacing that value through ordinary MVC/MVVM command handling; no
callback runs while the App lock is held.

## Concurrency And MVC/MVVM Use

All control handles alias App-owned canonical state and are safe for
concurrent use. Direct setters serialize one complete update. A Transaction
can replace several related collections and application state in one frame.
Transactions remain single-caller builders.

Controls do not fetch, page, sort external data, start workers, or retain a
consumer interface. An application model owns those lifecycles and posts
copied bounded results. `loading` and `error` are explicit view states for
that workflow. Commands are invoked outside toolkit locks; the controller can
inspect stable target identity and typed state, update its model, and publish
the next copied view without a second UI event loop.

## Rendering And Theme Roles

Collection controls use Turbo Vision appearance and behavior as the baseline.
They retain non-color indicators for current, selected, expanded, disabled,
loading, empty, error, sorted, and edited state. Default Theme roles include:

- `list_box`, `list_box.border`, `collection.current`,
  `collection.selected`, `collection.current_selected`,
  `collection.disabled`, `collection.empty`, `collection.loading`, and
  `collection.error`;
- `drop_down`, `drop_down.focused`, `drop_down.disabled`,
  `drop_down.popup`, `drop_down.popup_border`, `combo_box`,
  `combo_box.focused`, and `combo_box.disabled`;
- `tree_view`, `tree.guide`, `tree.branch`, and `tree.expanded`;
- `table`, `table.border`, `table.header`, `table.header_current`,
  `table.sort`, `table.cell_current`, and `table.row_selected`; and
- `data_grid.edit`, `data_grid.edit_focused`, and
  `data_grid.edit_invalid`.

Every marker is a canonical intended-frame cell. Physical-terminal fallback
is applied later without changing semantic ownership or style identity.

## Snapshot And Automation Contract

Core details expose exact bounded status text and disabled reason. Automation
details expose compact evidence instead, so a maximum collection payload
cannot inflate the retained wire response. ListBox uses message byte counts,
a SHA-256 status digest, selection cardinality/endpoints/digest, and compact
viewport state. DropDown uses item/enabled/retained counts, exact bounded
stable current and selected keys/indices, popup rows/open/bounds/offset and
provisional identities, enabled policy, disabled-reason byte count, and
commands. ComboBox combines that popup record with its exact bounded
TextField-compatible editor record. The intended frame remains the source of
exact visible labels, cells, markers, and styles.

The selection digest is lowercase hexadecimal SHA-256 over the ordered
sequence of length-prefixed selected keys. It is evidence of the complete
selection without multiplying response size by the collection capacity.
Public `Items`, `Nodes`, `Rows`, and `Selection` methods remain the exact
in-process copied APIs.

Client validation is kind-specific and fail-closed. It checks enums, counts,
key implications, digest shape, status/message rules, viewport geometry,
sort/column/edit implications, and aggregate collection resource counts. New
details follow the project
[`ControlDetails` extension checklist](../Control-Details-Extension-Checklist.md).

Automation may drive the complete raw key lifecycle, including modifier
chords, popup commit/cancel, selection, expansion, sorting, and DataGrid edit
commit. Stable catalog keys are `collections.list`, `collections.combo`,
`collections.tree`, `collections.table`, and `collections.data-grid`.

## Acceptance

Phase 16 is complete when:

- every control preserves current, selection, expansion, and sort by stable
  key across insertion, removal, reorder, resize, and model replacement;
- normal, empty, loading, error, disabled, large, and constrained geometries
  have deterministic typed and frame evidence;
- keyboard behavior is complete without stealing global Alt chords or Tab
  traversal;
- popup commit/cancel and DataGrid edit commit/cancel are exact and leave no
  hidden nested loop or focus leak;
- aggregate models and automation responses remain within proved bounds;
- ordinary, fuzz, concurrent, race, PTY, attached-automation, and all-mode
  build gates pass; and
- `expletives-test` exposes every implemented behavior on purpose-specific
  collection pages reachable from Controls/Collections.
