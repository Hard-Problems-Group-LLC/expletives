# Foundational Go API v0

- Status: Implemented pre-v1 contract
- Authority: Direct operator instructions on 2026-07-24 and 2026-07-25

Related decisions:
[`EXPL-DEC-008`](../../project-management/decision-log.md#expl-dec-008--select-the-first-runnable-go-terminal-and-automation-baseline)
and
[`EXPL-PROP-002`](../../project-management/proposals/under-review/expl-prop-002-foundation-review-corrections.md)

Implementation reviewed: 2026-07-30

## Purpose And Scope

This document defines the implemented public Go contract for the first
Core/Containers, Basic Presentation, raw-input, command, atomic-update, and
Basic Layout foundation in the root `expletives` package. It covers:

- independent `App` instances and their special root Panels;
- sealed `Control` and `Container` capabilities;
- copy-safe `Panel`, `Frame`, and `GroupBox` handles;
- non-container `Label`, `StaticText`, `Separator`, and `Rule` handles;
- semantic styles, immutable Themes, and resolved intended-frame cells;
- bounded atomic `Transaction` updates and recursive destruction;
- immutable local snapshots;
- Box/Grid measurement and arrangement, nested Layouts, Panel/Layout stacking,
  and overflow delivery;
- raw key lifecycle events, the command registry, structured command results,
  correlated completions, and Menu popup sessions; and
- typed Selection controls, focused-control guidance, bounded validated and
  password-safe text/numeric input, and multiline editing;
- deterministic ProgressBar, Meter, Spinner, and ActivityDots state,
  rendering, worker-update, and typed evidence; and
- concurrency, dispatch, callback, and final-state behavior.

The import path is:

```go
import "github.com/Hard-Problems-Group-LLC/expletives"
```

This remains a pre-v1 API. Changes require an explicit design decision,
updated specifications, migration notes when callers are affected, and
corresponding tests.

The root package owns local application state and observations. The
`automation` package owns its separate versioned wire DTO and projection; the
root `Snapshot` type is not the automation protocol schema.

## Value Types And Limits

`Point`, `Size`, and `Rect` use integer terminal cells. Sizes and rectangle
dimensions are nonnegative. Negative child origins are valid for deliberate
clipping. Coordinates, extents, and Layout arithmetic use checked geometry.
App surfaces are bounded by aggregate allocated cells, not an arbitrary
desktop width or height. Layout measurement interprets declared sizes as
minima, preserves them when clipped, and rejects a combined Layout minimum
beyond checked geometry.

The implemented exported limits are:

| Constant | Value | Meaning |
| --- | ---: | --- |
| `SnapshotVersion` | 1 | Current root-package snapshot revision |
| `MaxFrameCells` | 4,194,304 | Maximum cells in one materialized intended frame |
| `MaxRetainedFrameCells` | 8,388,608 | Aggregate App-owned retained frame cells |
| `MaxSnapshotHistoryRecords` | 64 | Retained metadata-record cap for tiny frames |
| `SnapshotRetention` | 64 | Compatibility name for the metadata-record cap |
| `MaxFrameWidth` | 4,194,304 | Compatibility single-axis bound; aggregate cells govern |
| `MaxFrameHeight` | 4,194,304 | Compatibility single-axis bound; aggregate cells govern |
| `MaxControls` | 4,096 | Maximum concurrently active nodes, including the root |
| `MaxLayouts` | 1,024 | Maximum active Layout objects in one App |
| `MaxLayoutItems` | 4,096 | Maximum aggregate attached Layout items |
| `MaxLayoutDepth` | 32 | Maximum nested Layout depth |
| `MaxOverflowHandlers` | 1 | Replaceable App overflow-handler slots |
| `MaxThemeStyles` | 4,096 | Maximum semantic definitions in one Theme |
| `MaxTransactionOperations` | 16,384 | Maximum recorded operations in one Transaction |
| `MaxAutomationKeyBytes` | 64 | Identifier and semantic-style-ID byte limit |
| `MaxTitleBytes` | 256 | Maximum UTF-8 bytes in a Frame or GroupBox title |
| `MaxTitleCells` | 256 | Maximum normalized cells in a title |
| `MaxCellBytes` | 64 | Maximum UTF-8 bytes in one retained canonical cell |
| `MaxDisplayTextBytes` | 256 | Maximum retained UTF-8 bytes for one display-control text |
| `MaxDisplayTextCells` | 256 | Maximum canonical cells, including line separators, for one display-control text |
| `MaxFocusGuidanceApplicationBytes` | 128 | Maximum application append/override guidance bytes |
| `MaxTextInputBytes` | 65,536 | Maximum retained UTF-8 bytes in one editor value |
| `MaxTextInputCells` | 65,536 | Maximum canonical one-cell elements in one editor value |
| `MaxTextValidatorBytes` | 4,096 | Maximum copied validator-set UTF-8 bytes |
| `MaxTextValidatorCells` | 1,024 | Maximum copied validator-set elements |
| `MaxTextInputAggregateBytes` | 262,144 | Maximum editor values and validators retained by one App |
| `MaxNumberDecimalPlaces` | 9 | Maximum fixed decimal places in NumberField/SpinBox |
| `MaxHotkeyBarItems` | 64 | Maximum entries in one HotkeyBar |
| `MaxActionItems` | 4,096 | Maximum aggregate HotkeyBar entries in one App |
| `MaxSelectionOptions` | 256 | Maximum copied options or Tabs in one control |
| `MaxSelectionItems` | 1,024 | Maximum aggregate RadioButton, fixed-option, and Tab records |
| `MaxMenuDepth` | 8 | Maximum immutable popup Menu tree depth |
| `MaxMenuItemsPerMenu` | 64 | Maximum direct entries in one Menu |
| `MaxMenus` | 256 | Maximum popup Menu models in one MenuBar tree |
| `MaxMenuItems` | 512 | Maximum aggregate MenuItem descriptors in one App |
| `MaxInputSources` | 4,096 | Maximum sources that may hold keys concurrently |
| `MaxHeldKeysPerSource` | 8 | Maximum simultaneously held keys for one source |
| `MaxConcurrentCommandHandlers` | 4 | Maximum live router callbacks per App |
| `MaxCommandDescriptionBytes` | 256 | Maximum command-description bytes |
| `MaxPublicMessageBytes` | 1,024 | Maximum structured command-result message bytes |

`Rect.Empty`, `Rect.Intersect`, and `IntendedFrame.Cell` operate on these
checked values. `ControlID`, `ControlKind`, `StyleID`, `Key`, `KeyEventKind`,
`CommandID`, and `Outcome` are distinct string-backed domain types.

Stable identifiers use 1 through 64 ASCII bytes from letters, digits, `.`,
`_`, `:`, and `-`.

## Application Construction And Theme

```go
type AppOptions struct {
    Size            Size
    RootConstraints RootConstraints
    Theme           Theme
    RootStyle       StyleID
    Scenario        string
}
```

`NewApp` validates the complete initial state before returning an App. A zero
Theme selects `DefaultTheme`; an empty `RootStyle` selects
`"application.root"`; and an empty Scenario selects `"unspecified"`. The
selected root style must exist in the Theme, and the scenario must be a stable
identifier. Zero root constraints make the root fill the complete surface.
Optional minimum, maximum, and terminal-cell aspect-ratio policy is specified
in
[`root-sizing-and-borders-v0.md`](root-sizing-and-borders-v0.md).

Callers retain and share the returned `*App`; an `App` value must not be
copied after first use.

`DefaultTheme` defines:

- `application.root`;
- `panel`;
- `frame` and `frame.border`;
- `group_box` and `group_box.border`;
- `layout.border`; and
- `label`, `static_text`, `separator`, `rule`, `button`, `hotkey_bar`,
  `focus_guide_bar`, and `menu_bar`;
- `checkbox`, `radio_group`, `radio_button`, `cycle_field`, `select_field`,
  `text_field`, `number_field`, `spin_box`, and `text_area`;
- `progress_bar`, `meter`, `spinner`, and `activity_dots`;
- `progress.fill`, `progress.text`, `progress.completed`, `progress.failed`,
  and `progress.cancelled`; and
- the Menu, Status Bar, Selection, and text-input variant roles listed by
  `DefaultTheme().Styles()`.

Base content roles resolve to white foreground on black background. Chrome,
focus, validation, and progress variants carry their documented semantic
colors.

```go
type StyleAttributes uint16

const (
    StyleBold StyleAttributes = 1 << iota
    StyleDim
    StyleItalic
    StyleUnderline
    StyleReverse
)

type ResolvedStyle struct {
    Foreground Color
    Background Color
    Attributes StyleAttributes
}

type Style struct {
    ID         StyleID
    Foreground Color
    Background Color
    Attributes StyleAttributes
}
```

Controls retain only a semantic `StyleID`. A Theme maps each ID to one
`ResolvedStyle`; controls never retain an independent RGB copy. `NewTheme`
requires one through `MaxThemeStyles` valid definitions, copies them, rejects
unsupported attributes, and rejects conflicting duplicate IDs. Identical
duplicate definitions are harmless.

`Theme.Styles` returns a sorted copy. `Theme.Resolve` reports the immutable
resolved value for an ID. `App.Theme` returns a copy of the current Theme.
`App.SetTheme` is a single-operation Transaction: every style referenced by
the final active tree and its decorations must exist in the replacement, and
one successful replacement produces at most one publication.

`RGB` constructs a resolved 24-bit `Color`. Its string and JSON form is
uppercase `#RRGGBB`; JSON decoding accepts either hexadecimal case.

## Root, Control, And Container

Each App owns exactly one parentless root returned by `App.Root()`. The root:

- has runtime ID and automation key `"root"`;
- implements `Container`;
- matches the App surface by default and otherwise uses its centered
  `RootConstraints`;
- cannot be hidden, independently destroyed, or assigned ordinary bounds; and
- is the only parentless control.

There is no package-global App or root. Apps have independent trees, themes,
input state, commands, snapshot histories, and sequence spaces.

The shallow sealed capabilities are:

```go
type Control interface {
    ID() ControlID
    AutomationKey() string
    Bounds() Rect
    MinimumSize() Size
    Visible() bool
    Style() StyleID

    // unexported identity method
}

type Container interface {
    Control
    Children() []Control

    // unexported container method
}
```

The unexported methods prevent fabricated toolkit identity. `Panel`, `Frame`,
and `GroupBox` implement `Container`. `Label`, `StaticText`, `Separator`, and
`Rule` implement only `Control`. External compound controls may embed a real
toolkit control but cannot forge a node.

The concrete public types are small handles over one canonical internal node.
Copying a constructed `Panel`, `Frame`, or `GroupBox` value aliases the same
node and is supported. Their embedded storage is unexported, so copying cannot
replace the handle's node. Zero and fabricated values remain invalid.

`Parent` returns the original concrete `Container`, and `Children` returns a
new `[]Control` containing the original concrete child handles in stable
insertion order. Mutating that slice does not mutate the tree.

`App.ControlByAutomationKey` resolves an active control by its stable key.
Nonempty automation keys are immutable, App-scoped, and unique among the final
active tree. A key becomes available again after destruction, including for a
replacement created in the same atomic Transaction.

Runtime control IDs are monotonically allocated per App and are not reused.
Their spelling is opaque except for `"root"`.

## Control Construction And Lifetime

```go
func NewPanel(parent Container, options PanelOptions) (*Panel, error)
func NewFrame(parent Container, options FrameOptions) (*Frame, error)
func NewGroupBox(parent Container, options GroupBoxOptions) (*GroupBox, error)

type PanelOptions struct {
    AutomationKey string
    Bounds        Rect
    MinimumSize   Size
    Style         StyleID
    Hidden        bool
}

type FrameOptions struct {
    PanelOptions
    Title            string
    BorderStyle      StyleID
    BorderForm       BorderForm
    BorderForeground *Color
    BorderBackground *Color
}

type GroupBoxOptions struct {
    PanelOptions
    Title            string
    BorderStyle      StyleID
    BorderForm       BorderForm
    BorderForeground *Color
    BorderBackground *Color
}
```

Every ordinary control requires a valid `Container` in the same App.
Parentage is immutable, cycles cannot be introduced, and reparenting remains
unavailable.

Empty style IDs select the control-kind defaults. Empty border style IDs
select `"frame.border"` or `"group_box.border"`. Every selected ID must exist
in the final Theme.

The empty Frame/GroupBox border form selects a one-cell single-line border.
`BorderNone` removes the decoration and client inset. Single, double, light,
medium, dark, and full-block forms are available. Non-nil foreground and
background pointers independently override the corresponding component of
the resolved border style. Layouts offer the same border forms independently,
including the common enclosing-Layout plus unbordered-adjacent-Frames case.

A title is validated and normalized before mutation. It is limited to
`MaxTitleBytes`, `MaxTitleCells`, and `MaxCellBytes` per normalized cell.
Unsupported display elements become one `U+FFFD` cell under
[`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md).

Each concrete container handle exposes these common mutations:

```go
SetBounds(Rect) error
SetMinimumSize(Size) error
SetStyle(StyleID) error
SetVisible(bool) error
Destroy() error
```

They are promoted as public methods by each concrete handle and execute as
single-operation Transactions. An unchanged assignment succeeds without a
publication.

`Destroy` recursively removes a non-root control and all descendants from the
active indexes and parent child order. It publishes once. A destroyed handle's
ID remains readable, `Visible` is false, `Children` is empty, and mutations
return `ErrDestroyed`. The concurrent-control count and stable keys are
released.

## Text And Display Controls

The first leaf-control family is:

```go
func NewLabel(Container, LabelOptions) (*Label, error)
func NewStaticText(Container, StaticTextOptions) (*StaticText, error)
func NewSeparator(Container, SeparatorOptions) (*Separator, error)
func NewRule(Container, RuleOptions) (*Rule, error)

func (l *Label) Text() string
func (l *Label) SetText(string) error
func (s *StaticText) Text() string
func (s *StaticText) SetText(string) error
func (r *Rule) Text() string
func (r *Rule) SetText(string) error
```

All four types retain common Control geometry, minimum, visibility, style,
Layout membership, stacking, parentage, and destruction behavior without
exposing `Children`, `SetLayout`, or `AddLayout`.

`Label` is single-line and supports horizontal/vertical alignment plus an
observable target and lowercase ASCII mnemonic association. Alt plus that
mnemonic focuses an eligible target through the Actions resolver.
`StaticText` supports LF/CRLF
lines and none, word, or cell wrapping. `Separator` is an untitled horizontal
or vertical divider; `Rule` adds aligned text. Dividers support none, single,
double, shade, and block forms.

Text normalizes through the one-cell display policy before measurement and is
bounded by `MaxDisplayTextBytes`, `MaxDisplayTextCells`, and
`MaxCellBytes`. A zero construction minimum selects intrinsic automatic
measurement; `SetText` updates that minimum until the application explicitly
calls `SetMinimumSize`. Destroying a Label target clears both target and
mnemonic without destroying the Label. The complete options, wrapping,
clipping, truncation, minima, snapshot, and automation behavior is defined in
[`text-and-display-api-v0.md`](text-and-display-api-v0.md).

## Actions

The first command-backed controls are:

```go
func NewButton(Container, ButtonOptions) (*Button, error)
func NewHotkeyBar(Container, HotkeyBarOptions) (*HotkeyBar, error)
func NewFocusGuideBar(Container, FocusGuideBarOptions) (*FocusGuideBar, error)
func (b *Button) Focus() error
func (b *Button) Activate(
    context.Context, source, requestID string,
) (Completion, error)
func (b *HotkeyBar) Items() []HotkeyBarItem
func (a *App) Focused() Control
func (a *App) SetFocusGuidance(Control, FocusGuidance) error
func (a *App) ClearFocusGuidance(Control) error
```

Both controls are non-container leaves. Button is focusable and references one
registered command. Its effective label, enabled/disabled reason, and checked
state come from that command definition. Default and cancel are exclusive
roles, and an optional ASCII mnemonic activates through the ordinary router.
HotkeyBar copies an ordered unique command inventory and projects the current
first App binding for each entry; it does not retain a second binding table.

Each direct parent Container defines a focus group. Tab and Shift-Tab cross
groups; arrows move spatially between controls within a group and cross to a
different group only for an unambiguous directional target. Plain
Enter/Space activates the focused Button, Enter falls back to the applicable
default Button, and Escape uses the applicable cancel Button. Raw Enter/Space
down-up pairs publish and clear source-local pressed capture; input reset and
disconnect clear capture without activation. An exact Alt mnemonic is
resolved before a global Alt binding. `ControlSnapshot.Focused`,
`ActionDetails`, and `HotkeyBarDetails` make the complete state atomically
inspectable.

The complete construction, validation, rendering, focus, routing, snapshot,
automation, and resource-bound contract is in
[`actions-api-v0.md`](actions-api-v0.md).

`FocusGuideBar` is the non-focusable presentation companion to this focus
model. It resolves generic keyboard guidance from current focus during render,
then optionally appends or overrides bounded application guidance registered
on that control. See
[`focus-guide-bar-api-v0.md`](focus-guide-bar-api-v0.md).

## Menus

The first popup-menu surface is:

```go
func NewMenu(MenuOptions) (*Menu, error)
func (m *Menu) Items() []MenuItem
func NewMenuBar(Container, MenuBarOptions) (*MenuBar, error)
func (b *MenuBar) Open() error
func (b *MenuBar) Close() error
func (b *MenuBar) Items() []MenuItem
```

`Menu` is a copied immutable model, not a Control or an event loop.
`MenuItem` is a copied command, separator, or submenu descriptor. `MenuBar`
is a non-container leaf parented directly by `app.Root()`; version 0 permits
one live MenuBar and popup session per App. It is physical application chrome
at row 0, not an ordinary Layout item, and its full-surface Bounds are
derived rather than caller-set. Top-level MenuItems select a start or end edge
group; declaration order is stable within each, and right-justified Help is
an end item. Command items derive label, enabled/disabled reason, checked
state, and first binding from the same `CommandDefinition` used by Button and
HotkeyBar.

Exact Alt top-level mnemonics open popups. F9 and Ctrl-Space activate the
menu row; Down or Enter opens the selected root. Arrow, Home, End, Enter,
Escape, and unmodified sibling mnemonics traverse without activation
surprises, and disabled items remain selectable but not activatable. Opening
saves prior Button focus; complete close or command activation restores it
when still eligible. Turbo Vision palette roles distinguish normal,
mnemonic, selected, selected-mnemonic, disabled, selected-disabled, and
shadow cells. Measured child popups backset left when their preferred cascade
would overflow. `MenuBarDetails` exposes a bounded depth-first flat tree plus
distinct open and selected key paths.

The complete ownership, validation, focus, rendering, keyboard, snapshot,
automation, and catalog-screen contract is in
[`menus-api-v0.md`](menus-api-v0.md).

## Status Bar

```go
func NewStatusBar(Container, StatusBarOptions) (*StatusBar, error)
func (b *StatusBar) Segments() []StatusSegment
func (b *StatusBar) SetSegments([]StatusSegment) error
```

`StatusBar` is the unique non-container leaf parented directly by
`app.Root()`. Its derived full-width Bounds occupy the physical last row,
independently of root constraints, and it cannot be a Layout item. A copied
bounded segment is either canonical static context or a command reference.
Command segments dynamically derive label, enabled/disabled reason, checked
state, and first binding from the shared command registry.

Higher segment Priority values are retained first when narrow; ties preserve
declaration order, retained segments paint in declaration order, and the
highest-priority segment receives a clipped representation when no complete
segment fits. `StatusBarDetails` exposes every segment, including effective
command state, chord, rendered relative Bounds, and clipping/omission state.

The exact public, geometry, painting, mutation, concurrency, snapshot, and
automation contract is
[`status-bar-api-v0.md`](status-bar-api-v0.md).

## Headers And Footers

```go
func NewHeader(Container, HeaderOptions) (*Header, error)
func NewFooter(Container, FooterOptions) (*Footer, error)
```

`Header` and `Footer` are root-owned one-row Containers with derived
full-physical-width Bounds. Headers retain construction order immediately
below the visible MenuBar. Footers retain construction order upward from the
visible StatusBar, making the most recently constructed Footer highest. They
remain outside root constraints and may not themselves become Layout items.

Each band may own ordinary Layout trees, but attachment is atomic and accepts
only a complete measured minimum height of at most one. This admits horizontal
BoxLayouts and one-row GridLayouts while rejecting vertical stacks, multirow
Grids, and height-consuming borders, gaps, or insets. Hidden bands and bands
denied a row by tiny-surface priority retain deterministic inspectable
geometry without an actionable overflow episode.

The exact public, ordering, tiny-surface, Layout, rendering, snapshot, and
automation contract is
[`headers-footers-api-v0.md`](headers-footers-api-v0.md).

## Selection Controls

```go
func NewCheckbox(Container, CheckboxOptions) (*Checkbox, error)
func NewRadioGroup(Container, RadioGroupOptions) (*RadioGroup, error)
func NewRadioButton(*RadioGroup, RadioButtonOptions) (*RadioButton, error)
func NewCycleField(Container, CycleFieldOptions) (*CycleField, error)
func NewSelectField(Container, SelectFieldOptions) (*SelectField, error)
```

Checkbox has typed two-state/three-state values. RadioGroup owns exactly one
stable child value unless its policy or lack of enabled children permits an
empty value. CycleField and SelectField copy bounded stable option records;
the latter is a distinct control kind and naming variant, not a popup.

Radio arrows and Home/End move focus without changing selection; Space or
Enter selects the focused RadioButton. CycleField/SelectField use `[` for the
previous enabled value and `]` for the next, stopping at the ends; Space,
Enter, and `Activate` advance and wrap. Arrows remain focus navigation.

User changes use serialized raw input and optionally invoke a registered
ChangeCommand outside toolkit locks after publishing the new typed value.
Programmatic setters and Transaction mutations do not emit that user
notification. Snapshot and automation details are exact typed members, not
unrestricted maps. The complete contract is
[`selection-api-v0.md`](selection-api-v0.md).

## TextField

```go
func NewTextField(Container, TextFieldOptions) (*TextField, error)
func (f *TextField) Text() string
func (f *TextField) SetText(string) error
func (f *TextField) Validator() *TextValidator
func (f *TextField) SetValidator(*TextValidator) error
func (f *TextField) SetPassword(bool) error
func (f *TextField) Focus() error
func (f *TextField) Activate(
    context.Context, source string, requestID string,
) (Completion, error)
```

TextField is a bounded single-line editor over canonical one-cell elements.
Enter starts and commits editing; Escape cancels; caret and delete keys operate
on complete elements. Tab commits before direct-parent focus-group traversal.
Programmatic focus loss commits silently, while a user Enter/Tab commit may
route the optional `ChangeCommand` after the value is published.

TextField, NumberField, and SpinBox default to horizontal stretch and natural
one-row vertical sizing. They draw no implicit frame and use their distinct
background across the complete arranged width. TextArea defaults to stretch
on both axes. Every `PanelOptions` accepts construction-time `LayoutHints`;
every `Control` exposes its resolved immutable hints through `LayoutHints()`.

The optional copied `TextValidator` requires soft or hard enforcement,
whitelist or blacklist mode, and a nonempty character set. Soft-invalid input
remains editable and paints the complete valid portion yellow with invalid
characters red. Hard-invalid input is ignored. Password mode paints `*`,
retains validation over the actual value, and redacts the value from core and
automation snapshots.

The exact public, validation, editing, Limited Unicode, snapshot, and
automation contract is
[`text-and-numeric-input-api-v0.md`](text-and-numeric-input-api-v0.md).

`NumberField` and `SpinBox` reuse the bounded editor with finite fixed-place
numeric values, optional inclusive bounds, typed invalid-intermediate state,
and atomic `SetValue`. SpinBox adds positive fixed-place `Step` and clamped
`[`/`]` changes outside edit mode. Invalid Enter/Tab commits retain edit
focus. Their complete surface and typed `ControlDetails.NumberField` evidence
are defined by the same input contract.

`TextArea` provides multiline editing over the same validator, password, and
Limited Unicode policies. Enter inserts LF while editing, Ctrl-Enter commits,
and Tab commits before group traversal. `TextWrapNone`, `TextWrapWords`, and
`TextWrapCells` select its private bounded viewport behavior. Shift movement,
selection replacement/deletion, Ctrl-A, visual-row navigation, and typed
`ControlDetails.TextArea` evidence are defined by the same input contract.

## Progress Controls

```go
func NewProgressBar(Container, ProgressBarOptions) (*ProgressBar, error)
func NewMeter(Container, MeterOptions) (*Meter, error)
func NewSpinner(Container, SpinnerOptions) (*Spinner, error)
func NewActivityDots(Container, ActivityDotsOptions) (*ActivityDots, error)

func (p *ProgressBar) State() ProgressBarState
func (p *ProgressBar) SetState(ProgressBarState) error
func (p *ProgressBar) Update(context.Context, ProgressBarState) error
func (m *Meter) State() MeterState
func (m *Meter) SetState(MeterState) error
func (m *Meter) Update(context.Context, MeterState) error
func (s *Spinner) State() ActivityState
func (s *Spinner) SetState(ActivityState) error
func (s *Spinner) Update(context.Context, ActivityState) error
func (a *ActivityDots) State() ActivityState
func (a *ActivityDots) SetState(ActivityState) error
func (a *ActivityDots) Update(context.Context, ActivityState) error
```

Progress controls are non-focusable Panel-derived leaves. They copy complete
application-owned state and never start a clock or worker. Determinate and
indeterminate bars, finite horizontal/vertical meters, absolute-tick spinner
and dot animation, stable terminal states, reduced-motion canonicalization,
exact ratio rendering, and `ControlDetails.Progress` evidence are fixed by
[`progress-api-v0.md`](progress-api-v0.md).

`Update` is the cancellation-aware synchronous boundary for worker results;
Transactions atomically publish a related group of states. Identical
canonical updates publish nothing.

## Navigation And Chrome

```go
func NewScrollBar(Container, ScrollBarOptions) (*ScrollBar, error)
func (s *ScrollBar) State() ScrollBarState
func (s *ScrollBar) SetState(ScrollBarState) error
func (s *ScrollBar) Update(context.Context, ScrollBarState) error
func (s *ScrollBar) Focus() error
func NewTabbedPanel(Container, TabbedPanelOptions) (*TabbedPanel, error)
func NewNotebook(Container, TabbedPanelOptions) (*Notebook, error)
func (p *TabbedPanel) Tabs() []Tab
func (n *Notebook) Tabs() []Tab
func (p *TabbedPanel) Selected() string
func (n *Notebook) Selected() string
func (p *TabbedPanel) SetTabs([]Tab, string) error
func (n *Notebook) SetTabs([]Tab, string) error
func (p *TabbedPanel) SetSelected(string) error
func (n *Notebook) SetSelected(string) error
func (p *TabbedPanel) Focus() error
func (n *Notebook) Focus() error
```

`ScrollBar` is a focusable Panel-derived leaf over copied nonnegative
`ContentSize`, `ViewportSize`, and `Offset` state. It derives
`MaximumOffset`, track, and thumb geometry; clamps matching arrow, page,
Home, and End navigation; leaves an orientation-mismatched arrow available
to spatial focus; and routes an optional ChangeCommand only for user changes.
The complete contract, including Theme roles and typed evidence, is
[`navigation-chrome-api-v0.md`](navigation-chrome-api-v0.md).

`TabbedPanel` and `Notebook` are distinct focusable container kinds over the
same copied `Tab` model. Each Tab names one unique live direct child Panel
page. The strip is one focus stop: Left/Right and Home/End move current tab
focus without selecting, Space/Enter selects, and a scoped Alt mnemonic
focuses and selects. Exactly the selected caller-visible page is effectively
visible and fills the container client area. `SetTabs` performs complete
ordered insertion/removal/reorder, and page destruction repairs the copied
model before publication.

## Scrolling And Content

```go
func NewViewport(Container, ScrollViewOptions) (*Viewport, error)
func NewScrollablePanel(
    Container,
    ScrollablePanelOptions,
) (*ScrollablePanel, error)
func (v *Viewport) Content() *Panel
func (s *ScrollablePanel) Content() *Panel
func (v *Viewport) State() ViewportState
func (s *ScrollablePanel) State() ViewportState
func (v *Viewport) SetState(ViewportState) error
func (s *ScrollablePanel) SetState(ViewportState) error
func (v *Viewport) Update(context.Context, ViewportState) error
func (s *ScrollablePanel) Update(context.Context, ViewportState) error
func (v *Viewport) EnsureVisible(Rect) error
func (s *ScrollablePanel) EnsureVisible(Rect) error
func (v *Viewport) Focus() error
func (s *ScrollablePanel) Focus() error
```

Both controls atomically create one toolkit-managed direct Content Panel.
Applications create children under `Content()` and attach Layouts to it; the
outer control rejects application Layouts and derives Content bounds from the
copied content extent and offset. `Viewport` is unframed with no integrated
bars. `ScrollablePanel` supports the normal border forms plus independently
configured Auto, Always, or Never horizontal and vertical integrated bars.

Offsets clamp after state and geometry changes. Arrow, page, Home, and End
keys move a focused eligible viewport without wrapping. `EnsureVisible` and
focused-descendant repair make the smallest required offset change.
Programmatic changes are silent; direct user scrolling may route the optional
change command outside toolkit locks. `ControlDetails.Scrollable` exposes the
complete canonical state, viewport, managed Content relationship, policies,
and integrated bar subrecords. The exact contract is
[`scrolling-content-api-v0.md`](scrolling-content-api-v0.md).

`MarkdownView` is the Panel-derived read-only document leaf built on that
scroll model. It supports the contract's bounded Markdown subset, visible
link destinations, deterministic prose reflow, non-wrapping fenced code,
thread-safe source replacement, and typed structural evidence without I/O or
link activation. Its complete surface is defined by the same Phase 15
contract.

## Atomic Transactions

```go
func (a *App) NewTransaction() *Transaction

func (t *Transaction) NewPanel(Container, PanelOptions) (*Panel, error)
func (t *Transaction) NewFrame(Container, FrameOptions) (*Frame, error)
func (t *Transaction) NewGroupBox(Container, GroupBoxOptions) (*GroupBox, error)
func (t *Transaction) NewLabel(Container, LabelOptions) (*Label, error)
func (t *Transaction) NewStaticText(Container, StaticTextOptions) (*StaticText, error)
func (t *Transaction) NewSeparator(Container, SeparatorOptions) (*Separator, error)
func (t *Transaction) NewRule(Container, RuleOptions) (*Rule, error)
func (t *Transaction) NewButton(Container, ButtonOptions) (*Button, error)
func (t *Transaction) NewHotkeyBar(Container, HotkeyBarOptions) (*HotkeyBar, error)
func (t *Transaction) NewFocusGuideBar(Container, FocusGuideBarOptions) (*FocusGuideBar, error)
func (t *Transaction) NewMenuBar(Container, MenuBarOptions) (*MenuBar, error)
func (t *Transaction) NewStatusBar(Container, StatusBarOptions) (*StatusBar, error)
func (t *Transaction) NewHeader(Container, HeaderOptions) (*Header, error)
func (t *Transaction) NewCheckbox(Container, CheckboxOptions) (*Checkbox, error)
func (t *Transaction) NewRadioGroup(Container, RadioGroupOptions) (*RadioGroup, error)
func (t *Transaction) NewRadioButton(*RadioGroup, RadioButtonOptions) (*RadioButton, error)
func (t *Transaction) NewCycleField(Container, CycleFieldOptions) (*CycleField, error)
func (t *Transaction) NewSelectField(Container, SelectFieldOptions) (*SelectField, error)
func (t *Transaction) NewTextField(Container, TextFieldOptions) (*TextField, error)
func (t *Transaction) NewNumberField(Container, NumberFieldOptions) (*NumberField, error)
func (t *Transaction) NewSpinBox(Container, SpinBoxOptions) (*SpinBox, error)
func (t *Transaction) NewTextArea(Container, TextAreaOptions) (*TextArea, error)
func (t *Transaction) NewProgressBar(Container, ProgressBarOptions) (*ProgressBar, error)
func (t *Transaction) NewMeter(Container, MeterOptions) (*Meter, error)
func (t *Transaction) NewSpinner(Container, SpinnerOptions) (*Spinner, error)
func (t *Transaction) NewActivityDots(Container, ActivityDotsOptions) (*ActivityDots, error)
func (t *Transaction) NewScrollBar(Container, ScrollBarOptions) (*ScrollBar, error)
func (t *Transaction) NewTabbedPanel(Container, TabbedPanelOptions) (*TabbedPanel, error)
func (t *Transaction) NewNotebook(Container, TabbedPanelOptions) (*Notebook, error)
func (t *Transaction) NewViewport(Container, ScrollViewOptions) (*Viewport, error)
func (t *Transaction) NewScrollablePanel(Container, ScrollablePanelOptions) (*ScrollablePanel, error)
func (t *Transaction) NewMarkdownView(Container, MarkdownViewOptions) (*MarkdownView, error)
func (t *Transaction) NewFooter(Container, FooterOptions) (*Footer, error)
func (t *Transaction) SetSize(Size) error
func (t *Transaction) SetRootConstraints(RootConstraints) error
func (t *Transaction) SetBounds(Control, Rect) error
func (t *Transaction) SetMinimumSize(Control, Size) error
func (t *Transaction) SetStyle(Control, StyleID) error
func (t *Transaction) SetVisible(Control, bool) error
func (t *Transaction) SetText(Control, string) error
func (t *Transaction) SetTextValidator(*TextField, *TextValidator) error
func (t *Transaction) SetTextPassword(*TextField, bool) error
func (t *Transaction) SetNumberValue(Control, float64) error
func (t *Transaction) SetTextAreaValidator(*TextArea, *TextValidator) error
func (t *Transaction) SetTextAreaPassword(*TextArea, bool) error
func (t *Transaction) SetTextAreaWrap(*TextArea, TextWrap) error
func (t *Transaction) SetProgressBarState(*ProgressBar, ProgressBarState) error
func (t *Transaction) SetMeterState(*Meter, MeterState) error
func (t *Transaction) SetActivityState(Control, ActivityState) error
func (t *Transaction) SetScrollBarState(*ScrollBar, ScrollBarState) error
func (t *Transaction) SetTabs(Control, []Tab, selected string) error
func (t *Transaction) SetSelectedTab(Control, string) error
func (t *Transaction) SetViewportState(Control, ViewportState) error
func (t *Transaction) EnsureViewportVisible(Control, Rect) error
func (t *Transaction) SetMarkdown(*MarkdownView, string) error
func (t *Transaction) SetMarkdownOffset(*MarkdownView, Point) error
func (t *Transaction) SetStatusSegments(*StatusBar, []StatusSegment) error
func (t *Transaction) SetFocus(Control) error
func (t *Transaction) SetFocusGuidance(Control, FocusGuidance) error
func (t *Transaction) Destroy(Control) error
func (t *Transaction) SetTheme(Theme) error
func (t *Transaction) Commit(context.Context) error
```

A Transaction is App-scoped, is not safe for concurrent builder mutation, and
is consumed by its single Commit attempt. At most
`MaxTransactionOperations` operations may be recorded. Repeated `SetSize` and
`SetTheme` calls replace their earlier recorded value and consume one
operation slot each; other recorded operations count individually.

Provisional controls may parent later provisional controls in the same
Transaction. They become active only after a successful Commit. A failed or
cancelled Commit aborts them: identity/property getters return zero values and
mutation returns `ErrInvalidControl`.

Commit:

1. observes caller cancellation while waiting for the App mutation gate;
2. waits no longer than `DefaultMutationWait` before `ErrMutationBusy`;
3. validates all current preconditions and the complete final state;
4. validates capacity after final destruction and creation;
5. validates key uniqueness and Theme references on only surviving nodes,
   using last recorded style mutations;
6. rejects mutation of a control destroyed by the same Transaction and
   creation under a destroyed parent;
7. applies all operations or none;
8. invokes no application callback while holding toolkit state; and
9. renders and publishes once when the final state changed.

`App.SetSize`, `App.SetRootConstraints`, `App.SetTheme`, ordinary
constructors, control setters, and `Destroy` are convenience
single-operation Transactions.

## Geometry, Painting, And Intended Cells

Control bounds are parent-client-relative logical rectangles. The
**Application Client Area** is the complete physical-width rectangle between
visible Main Menu/Headers and visible Footers/Status Bar. The constrained root
content rectangle is its intersection with the root constraints.

A **Panel Client Area** is the Panel's bounds after subtracting its border and
any visible horizontal or vertical scrollbars. A plain, unscrolled Panel's
client rectangle is its full bounds. A decorated Frame or GroupBox has a
one-cell client inset; `BorderNone` has none. Scrollbar deductions become
operative with the public scrollbar phase. A decorated Layout likewise
reserves one cell on every edge before its own Insets and item arrangement.
Logical geometry is retained outside ancestor or surface bounds.

Rendering performs stable depth-first traversal. Unmanaged children use
control insertion order; managed children use Layout stack order while
`Children` continues to expose control-tree insertion order:

1. calculate absolute bounds and the intersection of every ancestor client
   clip and the App surface;
2. fill the visible node with its Theme-resolved semantic style;
3. paint its control border and normalized title, if any;
4. paint each attached Layout border and its complete stack subtree in Layout
   stack order; and
5. paint unmanaged child subtrees in insertion order.

Later siblings paint over earlier siblings. Children of bordered containers
cannot paint over the one-cell border.

Each intended `Cell` contains:

- exactly one canonical grapheme;
- its semantic `StyleID`;
- resolved foreground, background, and terminal-independent attributes; and
- the runtime `ControlID` that last painted it.

Physical terminal degradation is a presenter projection and never mutates the
intended frame.

## Local Snapshot Contract

`App.Snapshot` returns a deep copy of the current atomic root-package
`Snapshot`. `App.SnapshotAt` returns a retained exact sequence, and
`App.WaitSnapshot` waits for a later sequence, caller cancellation, or final
state.

The temporary pre-v1 `type SnapshotV1 = Snapshot` alias exists only for local
source compatibility. It is not the automation protocol DTO.

The local snapshot contains sequence/finality, scenario, intended frame,
cursor, typed control tree, Layout tree, held input sources, overflow state,
and an optional local `Completion`. Control and Layout records expose separate
arrangement and current stack indices. A `ControlSnapshot` contains semantic
`StyleID` and its Theme-resolved `ResolvedStyle`; border detail does the same.
The typed details union contains `TextDetails` for Label/StaticText and
`DividerDetails` for Separator/Rule, `ActionDetails` for Button, and
`HotkeyBarDetails` for HotkeyBar, `FocusGuideBarDetails` for FocusGuideBar,
flat `MenuBarDetails` for MenuBar, and `StatusBarDetails` for StatusBar.
Selection, Text/Numeric Input, Progress, and Navigation controls add their
kind-consistent `CheckboxDetails`, `RadioButtonDetails`,
`RadioGroupDetails`, `ChoiceFieldDetails`, `TextFieldDetails`,
`NumberFieldDetails`, `TextAreaDetails`, `ProgressDetails`,
`ScrollBarDetails`, `TabbedPanelDetails`, `ScrollableDetails`, and
`MarkdownDetails` members.
These expose canonical bounded text,
alignment, wrap, Label target/mnemonic, divider orientation/form, generic
focus, command presentation state, pressed/default/cancel roles, and
structured current bindings. Menu details additionally expose immutable entry
identity and parent/depth, kind, effective command state, child counts,
selected/open flags, and ordered session paths. Status details expose copied
segment identity and priority, effective command state and chord, and exact
rendered/omitted/clipped geometry.

Snapshot storage is independent, including frame cells, child IDs, Layout
items, held keys, typed detail, overflow records, and completion. History is
evicted when actual retained frame cells exceed `MaxRetainedFrameCells` or
when tiny frames exceed `MaxSnapshotHistoryRecords`; at least the current
snapshot remains. Unknown, future, and expired sequences return
`ErrSnapshotNotRetained`.

Every accepted core input, command, or correlated reset publishes a completion
snapshot, including `no_op`, rejection, cancellation, and failure. A
Transaction used by a router can make one atomic view publication before the
later correlated completion publication.

The root package validates a request ID as correlation metadata but owns no
request-ID ledger, duplicate detection, or completion-result retention.
Submitting the same ID again is therefore not rejected by App. A protocol or
other boundary that requires session uniqueness owns that policy.

The versioned `automation.SnapshotV1` is a distinct automation-owned DTO
populated by explicit projection from the local `Snapshot`. Root snapshot
fields are not imported as the wire contract.

## Raw Input And Chords

`KeyEventDown`, `KeyEventUp`, and `KeyEventPress` enter before chord and
command resolution. `KeyPress` is one-shot and never leaves a key held.
Held state is isolated by source; at most `MaxInputSources` sources may hold
keys concurrently and at most `MaxHeldKeysPerSource` keys may be held per
source. A one-shot press does not allocate held-source state, and releasing
the last key reclaims that source. Capacity rejection is an explicit
correlated completion. Only held modifiers participate in chord matching.

The supported keys are lowercase ASCII letters, digits, `[` and `]`,
Control/Alt/Shift/Meta, Space, Enter, Escape, Tab, Backspace,
navigation/editing keys, and F1 through F12. Terminal escape bytes are not
valid logical keys.

`BindChord`, `ReplaceChord`, and `UnbindChord` manage structured bindings.
Chord modifier order is insignificant; modifiers must be unique, and the
pressed key cannot itself be a modifier.

`TextInputCommitted` and `TextInputPaste` instead enter through
`DispatchTextInput`. They are bounded non-key text events delivered only to a
focused enabled editor in edit mode. They never participate in command,
mnemonic, accelerator, menu, or binding resolution. Hard validators filter
disallowed elements, selections are replaced atomically, single-line editors
reject line separators, and over-capacity candidates are rejected without
partial insertion.

## Command Registry And Structured Results

```go
type CommandDefinition struct {
    ID             CommandID
    Label          string
    Description    string
    Enabled        bool
    DisabledReason string
    Checked        bool
    Automation     bool
}

type CommandResult struct {
    Outcome Outcome
    Code    string
    Message string
    Cause   error
}

type CommandRouter func(context.Context, Command) CommandResult
```

`RegisterCommand`, `ReplaceCommand`, `RemoveCommand`, and `Commands` own the
App-scoped command inventory. `Commands` is a deterministic copy. Labels are
canonical bounded one-cell display text; an empty label displays the command
ID. Descriptions and disabled reasons must be valid UTF-8 without NUL and fit
`MaxCommandDescriptionBytes`. An enabled command cannot retain a disabled
reason; a disabled command receives a sensible reason when none is supplied.
Replacing or removing a command republishes affected Action presentation and
repairs focus. Removing a command removes its current chord bindings while
existing controls retain a safely disabled unknown-command reference.

Every App includes the immutable, enabled, automation-visible
`CommandOverflowDismiss` (`"overflow.dismiss"`). Direct invocation, or
`Enter`/`Escape` while the fallback is active, acknowledges the warning
without clearing the overflow fact or calling the application router.

The structured router installed by `SetCommandRouter` executes only registered
and enabled commands. `Automation` is capability metadata used by the
automation server when it builds its advertised command inventory; it does
not independently authorize a local direct invocation.

`SetCommandHandler` remains a compatibility adapter for the former
`func(context.Context, Command) (Outcome, error)` shape. New callers use
`SetCommandRouter`. Legacy handler errors are mapped to stable public codes and
messages.

A `CommandResult` must contain a known Outcome, an empty or valid bounded Code,
and a valid UTF-8 Message without NUL of at most `MaxPublicMessageBytes`.
Invalid results become `failed` with `invalid_handler_result`. `Cause` is for
local diagnostics only and is excluded from JSON and automation projection.
Raw local error text is never made public automatically.

`Completion` contains request ID, outcome, resolved command, exact frame
sequence, bounded public code/message, and local-only cause.

The correlated methods are:

```go
func (a *App) DispatchKey(
    context.Context, source, requestID string, event KeyEvent,
) (Completion, error)

func (a *App) DispatchTextInput(
    context.Context, source, requestID string, event TextInputEvent,
) (Completion, error)

func (a *App) InvokeCommand(
    context.Context, source, requestID string,
    command CommandID, target ControlID,
) (Completion, error)

func (a *App) ResetInput(
    context.Context, source, requestID string,
) (Completion, error)
```

A direct nonempty target must identify an active control. Automation resolves
its stable `target_key` to this runtime ID at the boundary.

`ClearInputSource` is uncorrelated disconnect cleanup. It publishes only when
held or pressed-capture state actually changed and the App is not final.

## Concurrency, Dispatch, And Callbacks

Public App and control methods are safe for concurrent use. State mutation,
rendering, and immutable publication are protected separately from command
callback execution.

Each App has a bounded dispatch gate. Waiting observes caller cancellation and
is limited by `DefaultDispatchWait`; saturation returns `ErrDispatchBusy`.
There is no hidden context value or goroutine-identity convention.

The router runs outside App state locks on a bounded callback executor. A
caller deadline is preserved; otherwise `DefaultCommandTimeout` applies.
Router panics become a stable failed result. A router that ignores cancellation
may retain one of `MaxConcurrentCommandHandlers` slots, but the request returns
at its deadline and releases the dispatch gate. Saturated callback capacity
returns a structured `handler_capacity` failure.

A router may use getters, Transactions, constructors, and setters. Nested
dispatch to the same App is bounded and normally returns `ErrDispatchBusy`.
Dispatch to another App uses that App's independent gate and is supported;
cyclic or contended cross-App dispatch resolves through cancellation or the
bounded busy result rather than an indefinite lock cycle.

The toolkit does not synchronize caller-owned model values. Applications must
select their own model ownership or synchronization strategy.

## Final State And Errors

`OutcomeExited` and `OutcomeInterrupted` make the correlated snapshot final.
Afterward, snapshot reads remain available, but ordinary mutation, command
configuration, and correlated dispatch return `ErrClosed`. Input-source
cleanup may discard held state without replacing the final snapshot.

The root package owns no terminal or socket resource and exposes no App
`Close` in this slice.

Callers branch with `errors.Is` over:

| Error | Meaning |
| --- | --- |
| `ErrClosed` / `ErrAppStopped` | App is final |
| `ErrDuplicateKey` | Duplicate active automation key |
| `ErrDuplicateCommand` | Command already registered |
| `ErrDispatchBusy` | Dispatch or callback capacity wait expired |
| `ErrMutationBusy` | Mutation-gate wait expired |
| `ErrTransactionCapacity` | Transaction operation bound reached |
| `ErrControlCapacity` | Final active tree exceeds `MaxControls` |
| `ErrDestroyed` | Mutation targets a destroyed control |
| `ErrInvalidControl` | Fabricated, aborted, foreign, or inactive control |
| `ErrInvalidParent` | Nil, foreign, destroyed, or invalid parent |
| `ErrInvalidChord` | Invalid, missing, or duplicate binding |
| `ErrInvalidGeometry` | Invalid surface or rectangle |
| `ErrInvalidKeyEvent` | Unsupported key or event kind |
| `ErrInvalidRequest` | Invalid request, command, source, or target |
| `ErrSnapshotNotRetained` | Exact local sequence is unavailable |
| `ErrStyleConflict` | Theme definition or attributes conflict |
| `ErrStyleMissing` | Theme or referenced semantic style is missing |
| `ErrTextLimit` | Bounded title, display text, description, or message validation failed |
| `ErrValidation` | Invalid control policy/value state or hard-invalid programmatic text |

Context-aware methods reject nil contexts and may return
`context.Canceled` or `context.DeadlineExceeded`.

## Example

```go
theme, err := expletives.NewTheme(
    expletives.Style{
        ID: "application.root",
        Foreground: expletives.RGB(0xff, 0xff, 0xff),
        Background: expletives.RGB(0, 0, 0x20),
    },
    expletives.Style{
        ID: "status.inactive",
        Foreground: expletives.RGB(0xff, 0xff, 0xff),
        Background: expletives.RGB(0x40, 0, 0),
    },
    expletives.Style{
        ID: "status.active",
        Foreground: expletives.RGB(0, 0, 0),
        Background: expletives.RGB(0xff, 0xff, 0),
        Attributes: expletives.StyleBold,
    },
)
if err != nil {
    log.Fatal(err)
}

app, err := expletives.NewApp(expletives.AppOptions{
    Size: expletives.Size{Width: 40, Height: 10},
    Theme: theme,
    Scenario: "example.mvc",
})
if err != nil {
    log.Fatal(err)
}

view, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
    AutomationKey: "status",
    Bounds: expletives.Rect{X: 2, Y: 2, Width: 12, Height: 3},
    Style: "status.inactive",
})
if err != nil {
    log.Fatal(err)
}

if err := app.RegisterCommand(expletives.CommandDefinition{
    ID: "status.toggle", Enabled: true, Automation: true,
}); err != nil {
    log.Fatal(err)
}

active := false
if err := app.SetCommandRouter(func(
    ctx context.Context,
    command expletives.Command,
) expletives.CommandResult {
    active = !active
    style := expletives.StyleID("status.inactive")
    if active {
        style = "status.active"
    }
    tx := app.NewTransaction()
    if err := tx.SetStyle(view, style); err == nil {
        err = tx.Commit(ctx)
    }
    if err != nil {
        return expletives.CommandResult{
            Outcome: expletives.OutcomeFailed,
            Code: "view_update_failed",
            Message: "status view could not be updated",
            Cause: err,
        }
    }
    return expletives.CommandResult{Outcome: expletives.OutcomeApplied}
}); err != nil {
    log.Fatal(err)
}
```

The application owns synchronization for `active` if other goroutines access
it.

## Layout API

The exact implemented constructors, attachment/transaction methods,
per-axis natural/stretch hints and weights, measure/arrange rules, nesting,
`Panel.Raise`/`Lower`,
`Layout.Raise`/`Lower`, snapshot fields, and overflow callback are specified
in [`layout-api-v0.md`](layout-api-v0.md).

## Deferred And Excluded From v0

This contract does not yet provide:

- Layout replacement, detachment, spacers, and control reparenting;
- reparenting;
- a public custom-paint or arbitrary control factory;
- hit testing or mouse input;
- mutable titles or border styles;
- panel-owned/context menus or application-extensible mnemonic scopes;
- a public cursor mutator;
- a general UI-owner `Post`/`Call` event loop;
- synchronization of caller-owned models;
- terminal guarantees beyond the terminal-adapter contract; or
- authentication or capability authorization for attached automation.

Later controls and phases are listed in
[`control-catalog.md`](control-catalog.md). Build requirements are in
[`build-and-verification.md`](build-and-verification.md).
