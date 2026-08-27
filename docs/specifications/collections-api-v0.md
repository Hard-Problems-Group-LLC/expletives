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
    MaxTableFeatures            = 16
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
    Indicator      string
    IndicatorStyle StyleID
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

`Indicator` and `IndicatorStyle` are an optional paired presentation. Both
must be empty or both supplied. The indicator normalizes to exactly one
canonical display cell; its StyleID must exist in the complete staged Theme.
It appears before the textual label with one separating cell and retains its
own semantic style while the surrounding row uses current, selected,
current-selected, disabled, or ordinary styling. ListBox and popup wrapping
retain that one independently styled source cell without changing logical
item identity or navigation. A collapsed DropDown repeats its selected
indicator; ComboBox retains its editable text field while showing indicators
in the choices popup. Textual labels remain required, so an indicator or its
color is never the only representation of item meaning.

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
non-color marker remains on its first visual row. An optional item indicator
is the sole exception: its one cell retains `IndicatorStyle` while every
other cell keeps the logical row style.

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
Optional item indicators contribute two cells—indicator plus separator—to
popup measurement and retain their semantic styles across provisional
current and selection movement.
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

type TableFeature string

const (
    TableFeatureColumns TableFeature = "columns"
)

const (
    CommandTableColumnsOpen      CommandID = "table.columns.open"
    CommandTableColumnsSearch    CommandID = "table.columns.search"
    CommandTableColumnsCurrent   CommandID = "table.columns.current"
    CommandTableColumnsVisible   CommandID = "table.columns.visible"
    CommandTableColumnsWrap      CommandID = "table.columns.wrap"
    CommandTableColumnsMoveLeft  CommandID = "table.columns.move_left"
    CommandTableColumnsMoveRight CommandID = "table.columns.move_right"
    CommandTableColumnsReset     CommandID = "table.columns.reset"
    CommandTableColumnsReload    CommandID = "table.columns.reload"
    CommandTableColumnsOK        CommandID = "table.columns.ok"
    CommandTableColumnsCancel    CommandID = "table.columns.cancel"
)

type TableFocusPart string

const (
    TableFocusBody          TableFocusPart = "body"
    TableFocusColumnsAction TableFocusPart = "columns_action"
)

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

type TableVisualRoles struct {
    Body, Border, Header, CurrentHeader, SortMarker StyleID
    CurrentCell, SelectedRow, CurrentRow StyleID
    CurrentSelectedRow, Disabled, Empty, Loading, Error StyleID
    ScrollbarPage, ScrollbarArrow, ScrollbarThumb StyleID
    ScrollbarFocusedThumb, ScrollbarDisabled, ScrollbarCorner StyleID
    ColumnsBand, ColumnsAction, ColumnsActionDefault StyleID
    ColumnsActionFocused, ColumnsActionPressed StyleID
    ColumnsActionDisabled, ColumnsActionMnemonic, ColumnsActionShadow StyleID
}

type DataGridVisualRoles struct {
    TableVisualRoles
    Editable, FocusedEdit, InvalidEdit StyleID
    InvalidCharacter, TextSelection StyleID
}

type TableOptions struct {
    ScrollablePanelOptions
    Features         []TableFeature
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
    ColumnPresentation []TableColumnPresentation
    VisualRoles      TableVisualRoles
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
    Features           []TableFeature
    CurrentRow         string
    CurrentRowIndex    int
    CurrentColumn      string
    CurrentColumnIndex int
    Selected           []string
    SelectionStyle     TableSelectionStyle
    RequireSelection   bool
    RangeAnchor        string
    RangeExtent        string
    ColumnPresentation []TableColumnPresentation
    VisibleColumnCount int
    VisualRowCount     int
    FocusPart          TableFocusPart
    ColumnsDialogOpen  bool
    FocusMode          TableFocusMode
    SortColumn         string
    SortDirection      SortDirection
    Offset              Point
    RowCount            int
    EnabledCount        int
    ColumnCount         int
    CellCount           int
    ColumnWidths        []int
    VisualRoles         TableVisualRoles
}

type Table struct { /* copy-safe Panel-derived leaf */ }

func NewTable(Container, TableOptions) (*Table, error)
func (t *Transaction) NewTable(Container, TableOptions) (*Table, error)
func (t *Table) Columns() []Column
func (t *Table) ColumnPresentation() []TableColumnPresentation
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
func (t *Table) ReplaceWithPresentation(
    []Column,
    []TableRow,
    []TableColumnPresentation,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (t *Table) SetCurrent(row string, column string) error
func (t *Table) SetSelection([]string) error
func (t *Table) SetSelectionPolicy(TableSelectionPolicy) error
func (t *Table) SetFeatures([]TableFeature) error
func (t *Table) SetColumnPresentation([]TableColumnPresentation) error
func (t *Table) VisualRoles() TableVisualRoles
func (t *Table) SetVisualRoles(TableVisualRoles) error
func (t *Table) SetFocusMode(TableFocusMode) error
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
func (t *Transaction) ReplaceTableWithPresentation(
    *Table,
    []Column,
    []TableRow,
    []TableColumnPresentation,
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
func (t *Transaction) SetTableFeatures(*Table, []TableFeature) error
func (t *Transaction) SetTableColumnPresentation(
    *Table,
    []TableColumnPresentation,
) error
func (t *Transaction) SetTableVisualRoles(*Table, TableVisualRoles) error
func (t *Transaction) SetTableFocusMode(*Table, TableFocusMode) error
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
on visible columns share spare viewport width in stable presentation order.
The header is always one sticky clipped row and never participates in vertical
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

Column presentation is copied state independent of canonical schema and row
data. Its slice order is display order and contains every canonical column
key exactly once, including hidden entries. Nil means canonical order, all
visible, and Clip. A non-nil exact presentation rejects missing, unknown, or
duplicate keys, unknown wrap values, and a nonempty schema with no visible
column. The zero wrap value normalizes to Clip. State,
`ColumnPresentation()`, core snapshots, and mutations return or retain only
caller-owned copies. `SetColumnPresentation` publishes the complete validated
presentation atomically. Presentation order and visibility govern header/body
painting, width allocation, horizontal geometry, cell navigation, and
DataGrid editable-cell traversal without changing canonical `Columns()` or
`Rows()`.

Visual roles are a copied, fixed semantic-role mapping for one control.
Empty fields normalize to compatible built-in Table roles. Every normalized
StyleID must exist in the Theme selected by the same Transaction, allowing a
Theme and role mapping to change atomically without retaining raw colors in
control state. Role changes are instance local. `SetFocusMode` provides the
same atomic runtime row/cell selection as construction and repairs current
column state without routing application commands.

Clip paints one body line truncated to the derived width. Wrap breaks body
text at word boundaries and falls back to cell boundaries for an overlong
word. Hang uses the same rules with one ASCII-space continuation indent and
degrades to Wrap at width one. Headers never wrap. Each logical row has the
maximum line count of its visible cells; markers occur only on its first
line, while row/cell styles and separators span every visual line. Alignment
applies per line and, for Hang continuations, within the post-indent width.
`VisualRowCount`, vertical content size, offsets, current visibility, and Page
movement use these visual rows without flattening the canonical row model.

Up/Down/Home/End move logical rows; Page movement uses approximately one body
viewport of visual-row distance. In cell mode Left/Right move visible columns; in row
mode they scroll horizontally. Space applies the exact selection style; None
is a handled no-op and Range establishes a one-row interval. Enter activates
the current row/cell without selecting under None. Shift plus row navigation
extends Range, with `[` and `]` as portable previous/next-extent routes.
Ctrl-Home and Ctrl-End move to the first and last enabled row and boundary
column. Disabled rows remain visible but cannot be current, selected, an
endpoint, or activated. Without an eligible internal Columns action, Tab and
Shift-Tab leave the complete Table as one focus group.

The copied `Features` set is bounded by `MaxTableFeatures`, rejects unknown
or duplicate values, and normalizes to declaration order. Its only v0 value
is `TableFeatureColumns`. Enabling it reserves a full-width two-row band
inside the Table's outer bottom edge while keeping the Table one leaf and one
ControlID. The body viewport receives the remaining height. With only one
available band row the action body is allocated without a shadow; with two it
receives body and shadow; with none it is not visible or actionable. Body is
the default semantic focus part. ColumnsAction is retained only while that
feature has visible, enabled geometry; removing or disabling eligibility
repairs it atomically to Body. `SetFeatures` and its Transaction form replace
the complete set, and DataGrid cancels an active editor in the same atomic
mutation.

The raised `Columns...` action is right-justified in the band and reuses the
Button semantic roles, including mnemonic, focused, pressed, disabled, and
shadow presentation. Tab traverses Body to ColumnsAction before leaving the
focus group; Shift-Tab reverses that order. Enter and Space use source-local
press capture, and Alt-C is a supplemental mnemonic. An empty schema does not
remove the recovery action. A disabled or fully clipped action is not
actionable. `ColumnsDialogOpen` reports the exact modal ownership state; while
it is true the underlying action is not enabled or pressed.

Activation opens one toolkit-owned Dialog containing a searchable, bounded
ListBox inventory plus fixed visibility, Clip/Wrap/Hang, Move Left, Move
Right, Reset, OK, Cancel, and conditional Reload controls. Space on the
inventory toggles visibility; Ctrl-Left and Ctrl-Right supplement the move
buttons. Hiding the final visible column is rejected with explanatory dialog
text. All editing occurs in a private copied draft. Reset restores the
constructor presentation repaired to the current schema. Cancel and Escape
discard; OK publishes one complete presentation and then routes at most one
ChangeCommand outside App locks. DataGrid first attempts exactly one active
cell-editor commit; invalid input leaves the editor active and rejects the
dialog, while a successful changed commit routes before the modal opens.

The private draft captures schema and presentation revisions. Row-only model
changes do not conflict. Schema changes deterministically drop removed keys,
retain surviving draft order/visibility/wrap, and append new canonical keys
visible with Clip. An external presentation mutation marks the dialog stale,
disables OK, and exposes Reload; Reload replaces the draft with the latest
presentation, while Cancel discards it. Concurrent Transactions cannot reuse
a presentation revision. Owner hide or destroy, feature removal, modal
invalidation, and App finalization close the dialog and discard its draft.
Close restores ColumnsAction focus only while it remains eligible, otherwise
Body or ordinary App focus repair applies.

`SetRows` preserves surviving row current and selection, column presentation,
the current column, and sort. `SetModel` preserves presentation order,
visibility, and wrap for every surviving key, removes absent keys, and appends
new keys in canonical order as visible/Clip. If repair would leave no visible
column, it makes the first canonical column visible. A hidden or removed
current column repairs from its former presentation index by scanning right,
then left. Sort remains active when its column is merely hidden. `Replace`
preserves compatible surviving presentation while supplying the exact copied
schema, canonical rows, current coordinate, selection, and sort;
`ReplaceWithPresentation` and its Transaction form supply the complete exact
presentation in that same atomic replacement. Current-row
repair starts at the former displayed row index under the resulting sort,
then scans forward and backward.

Core typed details expose exact bounded status text and derived column widths.
Automation omits the retained columns, rows, cells, validators, and exact
status/disabled-reason text. It reports row/column/cell/enabled counts,
retained bytes, logical and visual row counts, current stable row/column and indices, exact selection style,
range endpoints, selection endpoints and digest, sort state, column endpoints
plus exact core presentation, visible-column count/endpoints and a compact
presentation digest for automation, and a digest of all derived widths,
the bounded feature identities, semantic focus part, Columns-action
visibility/enabled/pressed state and allocated bounds, dialog-open state,
optional commands, and the compact body
viewport. The intended frame supplies
exact visible headers, cells, markers, styles, clipping, and sticky-header
evidence.

## DataGrid

```go
type DataGridOptions struct {
    ScrollablePanelOptions
    Features         []TableFeature
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
    ColumnPresentation []TableColumnPresentation
    VisualRoles      DataGridVisualRoles
    SortColumn       string
    SortDirection    SortDirection
    Status           CollectionStatus
    StatusMessage    string
    ActivateCommand  CommandID
    SortCommand      CommandID
}

type DataGridState struct {
    TableState
    VisualRoles DataGridVisualRoles
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
func (g *DataGrid) ColumnPresentation() []TableColumnPresentation
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
func (g *DataGrid) ReplaceWithPresentation(
    []Column,
    []TableRow,
    []TableColumnPresentation,
    currentRow string,
    currentColumn string,
    selected []string,
    sortColumn string,
    sortDirection SortDirection,
) error
func (g *DataGrid) SetCurrent(string, string) error
func (g *DataGrid) SetSelection([]string) error
func (g *DataGrid) SetSelectionPolicy(TableSelectionPolicy) error
func (g *DataGrid) SetFeatures([]TableFeature) error
func (g *DataGrid) SetColumnPresentation([]TableColumnPresentation) error
func (g *DataGrid) VisualRoles() DataGridVisualRoles
func (g *DataGrid) SetVisualRoles(DataGridVisualRoles) error
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
func (t *Transaction) ReplaceDataGridWithPresentation(
    *DataGrid,
    []Column,
    []TableRow,
    []TableColumnPresentation,
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
func (t *Transaction) SetDataGridFeatures(*DataGrid, []TableFeature) error
func (t *Transaction) SetDataGridColumnPresentation(
    *DataGrid,
    []TableColumnPresentation,
) error
func (t *Transaction) SetDataGridVisualRoles(
    *DataGrid,
    DataGridVisualRoles,
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

Its visual-role mapping embeds the complete Table mapping and adds editable,
focused-edit, invalid-edit, invalid-character, and text-selection roles.
Changing the complete mapping cancels an active editor in the same atomic
transition and validates all roles against the staged Theme.

Enter or F2 begins editing the current editable cell. Enter commits, Escape
cancels, and Tab or Shift-Tab commits then moves to the next or previous
editable cell. The field has the same unfocused, focused, invalid-soft, and
invalid-character semantic styles as TextField. Hard validation ignores
disallowed input; soft validation permits it but rejects commit until valid.
Programmatic model, selection-policy, or column-presentation replacement
cancels an active edit before applying stable identity repair. The editor
remains a single horizontally scrolling line on the first visual line of a
wrapped cell. Its committed value continues to determine logical-row height
during editing; a successful commit invalidates and recomputes the affected
wrapped geometry once.

Tab and Shift-Tab keep edit traversal inside the DataGrid while a following or
preceding editable visible cell exists in presentation order, commit the
current valid value, move stable current to that cell, and begin its editor.
At the forward or reverse boundary
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

Every nonempty `ListItem.IndicatorStyle` is an additional caller-selected
Theme reference and is validated with these fixed roles.

Every marker is a canonical intended-frame cell. Physical-terminal fallback
is applied later without changing semantic ownership or style identity.

## Snapshot And Automation Contract

Core details expose exact bounded status text and disabled reason. Automation
details expose compact evidence instead, so a maximum collection payload
cannot inflate the retained wire response. ListBox uses message byte counts,
a SHA-256 status digest, selection cardinality/endpoints/digest, and compact
viewport state. It also exposes the normalized wrap policy and bounded derived
visual-row count while keeping `ItemCount` and `CurrentIndex` logical.
Vertical viewport offsets are visual-row coordinates after wrapping, so a
compact client must not infer wrapped current-item visibility by comparing
them with `CurrentIndex`; the intended frame and exact core snapshot own that
mapping. Unwrapped ListBoxes retain the direct index/viewport visibility
check because each logical item is exactly one visual row.
DropDown uses
item/enabled/retained counts, exact bounded
stable current and selected keys/indices, popup rows/open/bounds/offset and
provisional identities, enabled policy, disabled-reason byte count, and
commands. ComboBox combines that popup record with its exact bounded
TextField-compatible editor record. The intended frame remains the source of
exact visible labels, cells, markers, independently styled indicators, and
resolved colors. Indicator text and StyleID bytes count toward collection
retention; the complete retained item model remains absent from attached
automation.

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
commit. Stable collection-catalog keys are `collections.list`,
`collections.combo`, and `collections.tree`. Table and DataGrid have dedicated
Controls-menu screens with roots `tables.control` and `data-grid.control`.
Their right-hand Notebooks expose scrollable Options and Colors pages. Options
changes the Columns feature, exact selection policy, requirement, and Table
focus mode live. Colors changes one instance-local foreground/background role
through named-palette DropDowns whose textual names have independently styled
one-cell swatches, and an atomic Theme replacement. Both screens
provide exact reset commands and retain a 3:1 post-minimum horizontal split.

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
- `expletives-test` exposes ListBox, TreeView, DropDown, and ComboBox on
  Controls/Collections, and exposes Table and DataGrid on their dedicated
  Controls/Tables and Controls/DataGrid interactive screens.
