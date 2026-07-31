# Navigation And Chrome API v0

- Status: Directed pre-v1 contract
- Authority: Phase 14 roadmap and operator full-automatic direction,
  2026-07-31
- Scope: `ScrollBar`, `TabbedPanel`, `Notebook`, and `Tab`
- Depends on:
  [`actions-api-v0.md`](actions-api-v0.md),
  [`layout-api-v0.md`](layout-api-v0.md),
  [`application-architecture.md`](application-architecture.md), and
  [`automation-protocol-v1.md`](automation-protocol-v1.md)

## Design Boundary

Navigation controls are copied views of application-owned position and page
state. They reuse the App's serialized mutation, focus, raw-key, semantic
style, intended-frame, and typed-evidence paths. They do not introduce an
event loop, model base class, worker, terminal dependency, or mutable callback
object.

Classic Turbo Vision provides the ScrollBar presentation vocabulary of page
areas, arrows, and an indicator, with separate arrow and page steps. Modern
common-control practice treats a tabbed widget as a tab bar composed with a
stack of owned pages. This contract adopts those observable concepts without
copying either implementation.

## ScrollBar

```go
type ScrollBarState struct {
    ContentSize  int
    ViewportSize int
    Offset       int
}

type ScrollBarOptions struct {
    PanelOptions
    Orientation   Orientation
    State         ScrollBarState
    ArrowStep     int
    PageStep      int
    Disabled      bool
    DisabledReason string
    ChangeCommand CommandID
}

func NewScrollBar(Container, ScrollBarOptions) (*ScrollBar, error)
func (t *Transaction) NewScrollBar(
    Container,
    ScrollBarOptions,
) (*ScrollBar, error)
func (s *ScrollBar) State() ScrollBarState
func (s *ScrollBar) SetState(ScrollBarState) error
func (s *ScrollBar) Update(context.Context, ScrollBarState) error
func (s *ScrollBar) Focus() error
func (t *Transaction) SetScrollBarState(
    *ScrollBar,
    ScrollBarState,
) error
```

All state values are nonnegative checked cell/item counts.
`MaximumOffset = max(0, ContentSize - ViewportSize)` and `Offset` must not
exceed it. Content smaller than or equal to the viewport has offset zero and
a full-track indicator. Zero content is valid. Zero viewport is valid for a
temporarily collapsed consumer.

Orientation defaults Horizontal. ArrowStep defaults to one and must otherwise
be positive. PageStep defaults dynamically to `max(1, ViewportSize)` and must
otherwise be positive. The automatic minimum is 3x1 horizontally and 1x3
vertically.

The track reserves first/last arrow cells when the arranged axis has at least
three cells. The remaining page area is the track. A nonempty partial
viewport has an indicator of at least one cell; otherwise indicator size is
the exact floored viewport/content ratio, capped to the track. Indicator
position is the exact floored offset/maximum-offset ratio over the remaining
travel. Tiny zero-, one-, and two-cell arrangements remain bounded.

When focused, horizontal Left/Right and vertical Up/Down change Offset by
ArrowStep. PageUp/PageDown change it by PageStep. Home/End move to zero or
MaximumOffset. Changes clamp rather than wrap. An arrow that does not match
the orientation remains available to ordinary spatial focus navigation.
Programmatic setters are silent; user changes may route the optional
ChangeCommand after publication and outside toolkit locks.

The canonical Turbo Vision-inspired presentation uses distinct Theme roles
for page areas, arrows, indicator, focused indicator, and disabled state.
Canonical glyphs are one-cell arrows, light shade, and block indicator cells;
physical-terminal projection follows Limited Unicode policy.

`ControlDetails.ScrollBar` contains the complete state and policy plus the
geometry-derived MaximumOffset, TrackStart, TrackSize, ThumbStart, and
ThumbSize. Thumb positions are relative to the control's main axis.

## TabbedPanel, Notebook, And Tab

`TabbedPanel` and `Notebook` are distinct public control kinds over the same
contract. Each is a focusable Panel-derived container with a one-row tab strip
and a stacked page area. `Tab` is a copied descriptor, not another Control or
Layout:

```go
type Tab struct {
    Key            string
    Value          string
    Label          string
    Mnemonic       Key
    Page           *Panel
    Disabled       bool
    DisabledReason string
}

type TabbedPanelOptions struct {
    PanelOptions
    BorderStyle      StyleID
    BorderForm       BorderForm
    BorderForeground *Color
    BorderBackground *Color
    ChangeCommand    CommandID
}

func NewTabbedPanel(Container, TabbedPanelOptions) (*TabbedPanel, error)
func NewNotebook(Container, TabbedPanelOptions) (*Notebook, error)
func (t *Transaction) NewTabbedPanel(
    Container,
    TabbedPanelOptions,
) (*TabbedPanel, error)
func (t *Transaction) NewNotebook(
    Container,
    TabbedPanelOptions,
) (*Notebook, error)

func (p *TabbedPanel) Tabs() []Tab
func (n *Notebook) Tabs() []Tab
func (p *TabbedPanel) Selected() string
func (n *Notebook) Selected() string
func (p *TabbedPanel) SetTabs([]Tab, selected string) error
func (n *Notebook) SetTabs([]Tab, selected string) error
func (p *TabbedPanel) SetSelected(string) error
func (n *Notebook) SetSelected(string) error
func (p *TabbedPanel) Focus() error
func (n *Notebook) Focus() error
func (t *Transaction) SetTabs(Control, []Tab, selected string) error
func (t *Transaction) SetSelectedTab(Control, string) error
```

This two-stage construction supports one atomic public build:

1. create the provisional TabbedPanel/Notebook;
2. create each page Panel with that provisional container as parent;
3. set the ordered copied Tab descriptors referencing those direct pages; and
4. commit the transaction.

Tab Key and Value are stable bounded identifiers unique within the container.
Page references must be unique live direct Panel children of that container.
Labels are nonempty bounded single-line one-cell text. Mnemonics are optional
ASCII label members unique within that tab strip. DisabledReason is required
exactly when Disabled is true.

SetTabs is a complete ordered replacement and therefore supplies insertion,
removal, and reorder without a parallel item API. Removing a Tab descriptor
does not destroy or reparent its page; callers may hide or destroy that page
in the same Transaction. Empty selection deterministically chooses the first
enabled tab, or the first tab if all are disabled. A nonempty programmatic
selection may name a disabled tab so an application can preserve a visible
read-only page. User activation cannot select a disabled tab.

Exactly the selected Tab's page is effectively visible. This derived
visibility is combined with the page's ordinary Visible value and does not
overwrite caller policy. Every managed page fills the container client area;
its own Layout arranges page content normally.

The TabbedPanel/Notebook itself owns one focus stop for the strip. While it is
focused, Left/Right move current tab focus among enabled tabs without changing
selection, Home/End move to the first/last enabled tab, and Space/Enter select
the focused tab. Tab/Shift-Tab retain the project-wide focus-group contract;
leaving a hidden page repairs focus to the selected tab strip rather than an
unrelated hidden control. Alt plus a scoped tab mnemonic focuses the
container and selects that tab. No project default uses Alt-digits,
Ctrl-PageUp, or Ctrl-PageDown because supported host terminals reserve them.

The strip keeps the focused tab visible under clipping and uses explicit
one-cell continuation markers when earlier or later labels are omitted.
Selected, focused, mnemonic, disabled, border, and page-background roles use
the Turbo Vision palette conventions already established by Menus and
Actions. Selection and focus are distinguishable in typed state and by a
non-color-only shape cue.

`ControlDetails.TabbedPanel` contains ordered Tab records with stable key,
value, label, mnemonic, page ControlID/key, enabled state, selected/current
state, and rendered bounds/omission. It also contains selected/current values,
change command, and strip clipping state. Nested records are copied and
bounded.

## Concurrency, Mutation, And Destruction

All getters return copied state and all public methods are safe for concurrent
use. Transaction forms atomically coordinate viewport changes, tab/page model
changes, selected state, focus, page visibility policy, page content, and
other controls.

Destroying a ScrollBar or tab container follows ordinary recursive ownership.
Destroying a page referenced by a retained Tab in the same Transaction repairs
the tab model before publication. Removing the selected Tab chooses the first
enabled remaining Tab, then the first remaining Tab, or empty state.

## Acceptance

- ScrollBar validation, exact track/thumb geometry, empty/full/partial and
  zero/tiny axes, both orientations, clamping, keys, user-only command
  notification, transactions, snapshot copies, concurrency, and resize;
- empty/single/all-disabled Tab sets, selected/current separation, arrows
  without selection, Space/Enter activation, scoped mnemonics, clipping,
  page visibility, SetTabs insertion/removal/reorder, destruction repair,
  focus repair, nested page Layouts, and resize;
- kind-consistent root and automation DTO projection, client validation,
  deep-copy, resource accounting, and response-bound evidence;
- enabled catalog pages exercised through raw keys and direct commands in
  self-check and attached automation; and
- full ordinary, PTY, race, debug/release/profiling, smoke, and live
  closed-loop verification.
