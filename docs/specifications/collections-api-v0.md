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

`ListBoxOptions.SelectionMarks` may explicitly hide the normal `[ ]`/`[X]`
marks for list presentations, such as a classic file list, where current and
selection remain visible through semantic row styles. Its zero value shows
the markers for compatibility. This presentation choice does not alter
current, selection, keyboard, result, or automation semantics.

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
    SelectionMarks    CollectionSelectionMarks
    RequireSelection  bool
    Status            CollectionStatus
    StatusMessage     string
    Wrap              TextWrap
    CurrentCommand    CommandID
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
    VisualRowCount  int
    Wrap            TextWrap
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
real user selection change. `CurrentCommand` is routed after keyboard
navigation changes current without also selecting or activating it; this is
useful for inspector panes and previews. `ActivateCommand` is routed on Enter
or public `Activate`; activation has precedence over selection, and selection
has precedence over a coincident current change. Programmatic setters never
route commands.

The zero-value `Wrap` normalizes to `TextWrapNone` and preserves one visual row
per displayed item. `TextWrapWords` and `TextWrapCells` may derive multiple
visual rows from one logical item without manufacturing keys or multiplying
item, current, selection, activation, or retention counts. Word wrapping uses
the same ASCII-space break and canonical one-cell fallback rules as
`StaticText`. When an item has both label and description, the description's
continuation rows begin beneath the first description cell. If that natural
indent would consume the complete narrow viewport, continuation rows degrade
to the two-cell current-marker indent.

Up and Down move current by one enabled logical item, regardless of its visual
height. Page Up and Page Down use the visual viewport height while landing on
an enabled logical item. Home and End move to the first and last enabled item.
Space selects the current single-mode item or toggles it in multiple mode.
Enter selects it if needed and activates it. Vertical offset automatically
keeps the complete current item visible when it fits and at least its leading
visual row visible otherwise. All visual rows of a current, selected,
current-selected, or disabled item use the same semantic row style; the
non-color marker remains on its first visual row.

Wrapped content reflows at settled Panel Client Area width after integrated
vertical-scrollbar visibility converges. Its derived content width never
exceeds that viewport, so an `auto` horizontal bar remains absent. An explicit
`always` policy remains authoritative. Unwrapped content retains horizontal
scrolling for long labels and descriptions. Left and Right are handled,
non-wrapping horizontal movements; at either boundary they are explicit
no-ops, and the offset remains in `[0, MaximumOffset.X]`. In particular, a
wrapped ListBox whose content fits has zero maximum and current horizontal
offset. Tab and Shift-Tab leave the ListBox focus group.

## DropDown And ComboBox

`DropDown` is the selection-only one-row collapsed field. `ComboBox` is the
editable one-row field. They are separate public controls sharing the same
private popup-list model.

Construction always preserves a one-row minimum height. A partial
`PanelOptions.MinimumSize` may override the width or request a greater height,
but a zero omitted height does not collapse either popup field. Leaving the
complete minimum at zero retains ordinary intrinsic automatic sizing.

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

type TreeViewOptions struct {
    ScrollablePanelOptions
    Nodes            []TreeNode
    Current          string
    Selected         []string
    Expanded         []string
    SelectionMode    CollectionSelectionMode
    RequireSelection bool
    Status           CollectionStatus
    StatusMessage    string
    ActivateCommand  CommandID
    ExpandCommand    CommandID
}

type TreeViewState struct {
    Status        CollectionStatus
    StatusMessage string
    Current       string
    CurrentIndex  int
    Selected      []string
    Expanded      []string
    Offset        Point
    NodeCount     int
    VisibleCount  int
    EnabledCount  int
}

func NewTreeView(Container, TreeViewOptions) (*TreeView, error)
func (t *Transaction) NewTreeView(
    Container,
    TreeViewOptions,
) (*TreeView, error)
func (v *TreeView) Nodes() []TreeNode
func (v *TreeView) State() TreeViewState
func (v *TreeView) SetNodes([]TreeNode) error
func (v *TreeView) Replace(
    []TreeNode,
    current string,
    selected []string,
    expanded []string,
) error
func (v *TreeView) SetCurrent(string) error
func (v *TreeView) SetSelection([]string) error
func (v *TreeView) SetExpanded([]string) error
func (v *TreeView) SetNodeExpanded(string, bool) error
func (v *TreeView) SetStatus(CollectionStatus, string) error
func (v *TreeView) Focus() error
func (v *TreeView) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func (t *Transaction) SetTreeNodes(*TreeView, []TreeNode) error
func (t *Transaction) ReplaceTree(
    *TreeView,
    []TreeNode,
    current string,
    selected []string,
    expanded []string,
) error
func (t *Transaction) SetTreeCurrent(*TreeView, string) error
func (t *Transaction) SetTreeSelection(*TreeView, []string) error
func (t *Transaction) SetTreeExpanded(*TreeView, []string) error
func (t *Transaction) SetTreeNodeExpanded(
    *TreeView,
    string,
    bool,
) error
func (t *Transaction) SetTreeStatus(
    *TreeView,
    CollectionStatus,
    string,
) error
```

TreeView has ListBox-equivalent status, current, selection mode, required
selection, change command, activation command, and scrolling options. Nodes
are copied iteratively with unique keys across the complete tree, a maximum
depth of `MaxCollectionDepth`, and a maximum total node count of
`MaxCollectionItems`. Only branches may be expanded. A nil `Options.Expanded`
uses the copied `TreeNode.Expanded` flags; a non-nil slice is an exact initial
expansion set. `SetNodes` preserves surviving expansion by stable key and
uses node flags only for new keys. `Replace` and `SetExpanded` apply exact
expansion sets. Every expansion getter and setter uses canonical model
pre-order.

Up and Down move through visible pre-order rows. Right expands a collapsed
branch or moves to its first enabled visible child. Left collapses an expanded
branch or moves to its nearest visible parent. Space changes selection;
Enter activates. `+`, `-`, and `*` expand, collapse, and recursively expand
the current branch within the global bounds. Expansion changes may route a
separate optional `ExpandCommand`; programmatic expansion does not.

Tree guides, branch markers, indentation, current, selection, disabled state,
and horizontal clipping use semantic styles. Expansion and current are
observable by node key and visible-row index. Hidden selections survive a
collapse; when collapse hides current, current moves to its nearest enabled
visible ancestor. `EnabledCount` is the number of enabled visible nodes.
TreeView offers to stretch on both axes and keeps current vertically visible.

Core typed details expose exact state. The automation projection omits the
recursive node model and exact status/disabled-reason text. It reports bounded
node, visible, enabled, selected, expanded, and retained-byte counts; first
and last selected/expanded keys; stable selection and expansion digests;
current visible index; optional commands; and the compact content viewport.

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

type TableSelectionStyle string

const (
    TableSelectionNone     TableSelectionStyle = "none"
    TableSelectionSingle   TableSelectionStyle = "single"
    TableSelectionRange    TableSelectionStyle = "range"
    TableSelectionMultiple TableSelectionStyle = "multiple"
)

type TableSelectionPolicy struct {
    Style    TableSelectionStyle
    Require  bool
    Selected []string
    Anchor   string
    Extent   string
}

type TableOptions struct {
    ScrollablePanelOptions
    Columns          []Column
    Rows             []TableRow
    CurrentRow       string
    CurrentColumn    string
    Selected         []string
    SelectionMode    CollectionSelectionMode
    SelectionStyle   TableSelectionStyle
    RequireSelection bool
    RangeAnchor      string
    RangeExtent      string
    FocusMode        TableFocusMode
    SortColumn       string
    SortDirection    SortDirection
    Status           CollectionStatus
    StatusMessage    string
    ActivateCommand  CommandID
    SortCommand      CommandID
}

type TableState struct {
    Status             CollectionStatus
    StatusMessage      string
    CurrentRow         string
    CurrentRowIndex    int
    CurrentColumn      string
    CurrentColumnIndex int
    Selected           []string
    SelectionStyle     TableSelectionStyle
    RangeAnchor        string
    RangeExtent        string
    FocusMode          TableFocusMode
    SortColumn         string
    SortDirection      SortDirection
    Offset              Point
    RowCount            int
    EnabledCount        int
    ColumnCount         int
    CellCount           int
    ColumnWidths        []int
}

type Table struct { /* copy-safe Panel-derived leaf */ }

func NewTable(Container, TableOptions) (*Table, error)
func (t *Transaction) NewTable(Container, TableOptions) (*Table, error)
func (t *Table) Columns() []Column
func (t *Table) Rows() []TableRow
func (t *Table) State() TableState
func (t *Table) SetRows([]TableRow) error
func (t *Table) SetModel([]Column, []TableRow) error
func (t *Table) Replace(
    []Column,
    []TableRow,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (t *Table) SetCurrent(row string, column string) error
func (t *Table) SetSelection([]string) error
func (t *Table) SetSelectionPolicy(TableSelectionPolicy) error
func (t *Table) SetSort(string, SortDirection) error
func (t *Table) SetStatus(CollectionStatus, string) error
func (t *Table) Focus() error
func (t *Table) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func (t *Transaction) SetTableRows(*Table, []TableRow) error
func (t *Transaction) SetTableModel(
    *Table,
    []Column,
    []TableRow,
) error
func (t *Transaction) ReplaceTable(
    *Table,
    []Column,
    []TableRow,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (t *Transaction) SetTableCurrent(*Table, string, string) error
func (t *Transaction) SetTableSelection(*Table, []string) error
func (t *Transaction) SetTableSelectionPolicy(
    *Table,
    TableSelectionPolicy,
) error
func (t *Transaction) SetTableSort(*Table, string, SortDirection) error
func (t *Transaction) SetTableStatus(
    *Table,
    CollectionStatus,
    string,
) error
```

Column and row keys are unique. Each row contains at most one cell for a
known column; omitted cells are empty. The App-wide cell count is bounded by
`MaxCollectionCells`. Zero column width measures the maximum header/cell
width; explicit minimum/maximum constraints clamp it; positive Grow values
share spare viewport width in stable column order. Text does not wrap in v0.
The header is always one sticky row and never participates in vertical
scrolling. A column validator requires `Editable`; hard validation rejects
incompatible copied model cells now so the same schema remains valid when the
later DataGrid editor uses it. Table itself never enters edit mode.

Table supports row or cell current focus, one canonical None, Single, Range,
or Multiple stable-row selection policy, and optional single-column stable
sorting. An omitted `SelectionStyle` derives the compatible Single/Multiple
value from legacy `SelectionMode`; explicit disagreement is invalid. None
retains no selection, Range retains stable Anchor/Extent and derives the
continuous enabled displayed interval, and `SetSelectionPolicy` changes the
complete policy atomically. Sorting is stable and compares normalized cell
sequences, then preserves original model order for equal values. Activating a
sortable current column with `S` cycles none → ascending → descending → none.
`SortCommand` is routed only after a real user sort change. Programmatic
`SetSort`, model replacement, and Transactions are silent. Programmatic
replacement never reorders the caller's canonical model; `Rows` always
returns canonical model order while current indices, selection order, the
frame, and automation reflect derived display order.

Up/Down/Page/Home/End move rows. In cell mode Left/Right move columns; in row
mode they scroll horizontally. Space applies the exact selection style; None
is a handled no-op and Range establishes a one-row interval. Enter activates
the current row/cell without selecting under None. Shift plus row navigation
extends Range, with `[` and `]` as portable previous/next-extent routes.
Ctrl-Home and Ctrl-End move to the first and last enabled row and boundary
column. Disabled rows remain visible but cannot be current, selected, an
endpoint, or activated. Tab and Shift-Tab leave the complete Table as one
focus group.

`SetRows` preserves surviving row current and selection, the current column,
and sort. `SetModel` additionally preserves a surviving current column and
sortable sort column; otherwise it repairs to the first column and clears
sort. `Replace` supplies the exact copied schema, canonical rows, current
coordinate, selection, and sort. Current-row repair starts at the former
displayed index under the resulting sort, then scans forward and backward.

Core typed details expose exact bounded status text and derived column widths.
Automation omits the retained columns, rows, cells, validators, and exact
status/disabled-reason text. It reports row/column/cell/enabled counts,
retained bytes, current stable row/column and indices, exact selection style,
range endpoints, selection endpoints and digest, sort state, column endpoints
and a digest of all derived widths,
optional commands, and the compact viewport. The intended frame supplies
exact visible headers, cells, markers, styles, clipping, and sticky-header
evidence.

## DataGrid

```go
type DataGridOptions struct {
    ScrollablePanelOptions
    Columns          []Column
    Rows             []TableRow
    CurrentRow       string
    CurrentColumn    string
    Selected         []string
    SelectionMode    CollectionSelectionMode
    SelectionStyle   TableSelectionStyle
    RequireSelection bool
    RangeAnchor      string
    RangeExtent      string
    SortColumn       string
    SortDirection    SortDirection
    Status           CollectionStatus
    StatusMessage    string
    ActivateCommand  CommandID
    SortCommand      CommandID
}

type DataGridState struct {
    TableState
    Editing    bool
    EditRow    string
    EditColumn string
    EditText   string
    EditCaret  int
    EditValid  bool
}

type DataGrid struct { /* copy-safe Panel-derived leaf */ }

func NewDataGrid(Container, DataGridOptions) (*DataGrid, error)
func (t *Transaction) NewDataGrid(
    Container,
    DataGridOptions,
) (*DataGrid, error)
func (g *DataGrid) Columns() []Column
func (g *DataGrid) Rows() []TableRow
func (g *DataGrid) State() DataGridState
func (g *DataGrid) SetRows([]TableRow) error
func (g *DataGrid) SetModel([]Column, []TableRow) error
func (g *DataGrid) Replace(
    []Column,
    []TableRow,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (g *DataGrid) SetCurrent(string, string) error
func (g *DataGrid) SetSelection([]string) error
func (g *DataGrid) SetSelectionPolicy(TableSelectionPolicy) error
func (g *DataGrid) SetSort(string, SortDirection) error
func (g *DataGrid) SetStatus(CollectionStatus, string) error
func (g *DataGrid) Focus() error
func (g *DataGrid) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)

func (t *Transaction) SetDataGridRows(*DataGrid, []TableRow) error
func (t *Transaction) SetDataGridModel(
    *DataGrid,
    []Column,
    []TableRow,
) error
func (t *Transaction) ReplaceDataGrid(
    *DataGrid,
    []Column,
    []TableRow,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (t *Transaction) SetDataGridCurrent(*DataGrid, string, string) error
func (t *Transaction) SetDataGridSelection(*DataGrid, []string) error
func (t *Transaction) SetDataGridSelectionPolicy(
    *DataGrid,
    TableSelectionPolicy,
) error
func (t *Transaction) SetDataGridSort(
    *DataGrid,
    string,
    SortDirection,
) error
func (t *Transaction) SetDataGridStatus(
    *DataGrid,
    CollectionStatus,
    string,
) error
```

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

Tab and Shift-Tab keep edit traversal inside the DataGrid while a following or
preceding editable cell exists, commit the current valid value, move stable
current to that cell, and begin its editor. At the forward or reverse boundary
they commit and leave the DataGrid focus group. A soft-invalid value blocks
both Enter and Tab commit and retains the visible editor. Generic focus loss,
menu entry, and every programmatic DataGrid mutation cancel an active edit;
they never publish a partially edited value. Enter on a read-only current cell
uses Table activation behavior.

Committed user edits update the DataGrid's copied row model atomically and
rederive sorted display and selection order without changing stable current.
The `ScrollablePanelOptions.ScrollViewOptions.ChangeCommand` is the optional
user-change notification for both row-selection changes and committed cell
edits; the grid is the target in either case. Workers and
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
viewport state. It also exposes the normalized wrap policy and bounded derived
visual-row count while keeping `ItemCount` logical. DropDown uses
item/enabled/retained counts, exact bounded
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
- wrapped logical rows preserve identity, navigation, hanging indentation,
  style, resize reflow, and ordinary-width horizontal-bar suppression;
- keyboard behavior is complete without stealing global Alt chords or Tab
  traversal;
- popup commit/cancel and DataGrid edit commit/cancel are exact and leave no
  hidden nested loop or focus leak;
- aggregate models and automation responses remain within proved bounds;
- ordinary, fuzz, concurrent, race, PTY, attached-automation, and all-mode
  build gates pass; and
- `expletives-test` exposes every implemented behavior on purpose-specific
  collection pages reachable from Controls/Collections.
