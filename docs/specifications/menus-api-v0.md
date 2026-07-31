# Menus API v0

- Status: Implemented pre-v1 contract
- Authority: Direct operator instructions through 2026-07-30
- Scope: `MenuBar`, immutable popup `Menu` models, `MenuItem`, keyboard
  sessions, Turbo Vision look-and-feel, and `expletives-test` navigation

Turbo Vision is a visual and interaction reference only. The implementation
is native expletives code and does not copy or depend on Turbo Vision
implementation details.

## Public Model

```go
type MenuItemKind string

const (
    MenuItemCommand   MenuItemKind = "command"
    MenuItemSeparator MenuItemKind = "separator"
    MenuItemSubmenu   MenuItemKind = "submenu"
)

type MenuBarPlacement string

const (
    MenuBarPlacementDefault MenuBarPlacement = ""
    MenuBarPlacementStart   MenuBarPlacement = "start"
    MenuBarPlacementEnd     MenuBarPlacement = "end"
)

type MenuItem struct {
    Key       string
    Kind      MenuItemKind
    Label     string
    Command   CommandID
    Mnemonic  Key
    Placement MenuBarPlacement
    Menu      *Menu
}

type MenuOptions struct {
    Items []MenuItem
}

func NewMenu(MenuOptions) (*Menu, error)
func (m *Menu) Items() []MenuItem

type MenuBarOptions struct {
    PanelOptions
    Items                  []MenuItem
    PopupStyle             StyleID
    BorderStyle            StyleID
    MnemonicStyle          StyleID
    FocusedStyle           StyleID
    FocusedMnemonicStyle   StyleID
    DisabledStyle          StyleID
    FocusedDisabledStyle   StyleID
    ShadowStyle            StyleID
}

func NewMenuBar(Container, MenuBarOptions) (*MenuBar, error)
func (t *Transaction) NewMenuBar(
    Container,
    MenuBarOptions,
) (*MenuBar, error)
func (b *MenuBar) Open() error
func (b *MenuBar) Close() error
func (b *MenuBar) Items() []MenuItem
```

`Menu` is a copy-safe immutable popup model, not a Control or event loop.
`MenuItem` is a copied immutable descriptor. `MenuBar` is a non-container
leaf and the sole visual/session owner. An App permits at most one live
MenuBar in v0.

A command item has a required registered Command, optional ASCII mnemonic,
and no caller label or child Menu. Its label, enabled/disabled reason, checked
state, and first binding come from the current shared `CommandDefinition`. A
separator has no label, command, mnemonic, or child. A submenu has a bounded
Label, optional mnemonic, and required child Menu.

Top-level items are submenus. Their empty Placement canonicalizes to `start`;
`end` places an item in the opposite edge group. Popup items reject any
Placement. Item keys are required and unique across the complete tree.
Mnemonics are unique among siblings. Aliased Menu instances and cycles are
rejected. Limits are `MaxMenuDepth == 8`,
`MaxMenuItemsPerMenu == 64`, `MaxMenus == 256`, and
`MaxMenuItems == 512`.

## Application Chrome

The MenuBar must be constructed directly under `app.Root()`. It is physical
application chrome, not an ordinary root-content or Layout-managed control:

- it occupies physical row 0 from column 0 through the last column;
- Bounds and AbsoluteBounds are derived from the App surface;
- nonzero caller Bounds or MinimumSize are rejected;
- `SetBounds`, `SetMinimumSize`, and Layout membership are rejected;
- a visible MenuBar reserves row 0 from intersecting root content; and
- root constraints never center, narrow, or move it.

Resize, visibility, and destruction update MenuBar and root-content geometry
atomically. The complete edge-row contract is in
[`application-chrome-v0.md`](application-chrome-v0.md).

## Focus And Session State

One App owns at most one menu session. A session saves the focused Button,
gives focus to the MenuBar, and is in one of two states:

- **bar active:** one root label is selected but no popup is open; or
- **popup active:** the selected root popup and zero or more child popups are
  open.

`MenuBar.Open()` opens the first root popup. F10 and Ctrl-Space enter the
bar-active state. Exact Alt plus a root mnemonic opens that root popup
directly.

Opening saves current Button focus. Complete close restores it when still
eligible; otherwise stable-tree focus repair chooses the first eligible
Button. Hiding, destroying, or invalidating the MenuBar closes safely.

Every non-separator item, including a disabled or unknown command, is
selectable so disabled presentation can be inspected. Only a submenu or an
enabled registered command is activatable. Command-state changes retain a
still-structural selection and update its enabled style and reason; they
never activate a disabled or unknown command.

## Keyboard Resolution

Menu handling follows overflow/modal handling and precedes non-menu Action
mnemonics and global chords.

When closed:

- exact Alt plus a root mnemonic opens that popup;
- F10 activates the first root label without opening its popup; and
- Ctrl-Space provides the documented non-Alt/non-function-key fallback with
  the same bar-active behavior.

F10 and Ctrl-Space close an existing session. While only the bar is active:

- Left/Right select the previous/next root cyclically;
- Down or Enter opens the selected root popup;
- an unmodified root mnemonic selects and opens that root; and
- Escape closes and restores focus.

While a popup is active:

- Up/Down move cyclically among non-separator items, including disabled ones;
- Home/End select the first/last non-separator item;
- Right opens a selected submenu, or switches the root popup;
- Left closes one child popup, or switches the root popup;
- Enter opens a selected submenu or activates an enabled command;
- an unmodified sibling mnemonic opens/activates only an activatable item;
- Alt plus a root mnemonic switches directly to that popup; and
- Escape closes one child level, then closes/restores at the root popup.

Selection never activates. Activation closes/restores before the router runs
outside toolkit locks. Button, HotkeyBar, MenuItem, mnemonic, bound chord, and
direct automation routes use the same command definition and correlated
completion path.

## Turbo Vision Look And Feel

The default semantic palette roles are:

| Role | Default presentation |
|---|---|
| MenuBar, popup, and border | black on light gray |
| mnemonic | red on light gray |
| selected item/root | black on green |
| selected mnemonic | red on green |
| disabled item | dark gray on light gray |
| selected disabled item | dark gray on green |
| shadow | black on black |

Top-level labels have one blank cell on each side and no brackets or
synthetic selection marker. Their mnemonic letter alone uses the mnemonic
role. Start-group items are laid out from column 1 toward the right.
End-group items are measured as one block and laid out toward the left from
the last column, preserving declaration order within that group. The last
column is therefore the final end-item padding cell. End items paint after
start items so the explicitly edge-anchored group remains legible when a
surface is narrower than the MenuBar minimum. The rest of the physical row
is filled with MenuBar style.

Popups use a light-gray body, single-line border, red mnemonic letters,
green selected row, disabled roles, right-aligned structured shortcut,
checkmark column, right-pointing submenu indicator, tee-connected separator,
and a two-column-right/one-row-down black shadow. A selected row has no
synthetic `>` marker. Terminal projection may map the canonical check,
triangle, separator, and line glyphs to DEC Special Graphics or ASCII.

## Popup Measurement And Placement

Every popup measures all current effective labels, shortcut text, check
column, submenu indicator, padding, and border. It is sized to fit the widest
entry when the surface permits.

A top-level popup starts at its root label's resolved start- or end-group X
position, then backsets as needed to fit the surface. It does not use a
declaration-order offset that would detach an end-aligned popup from its
label.

A child popup prefers a small right/down cascade from its selected parent
row. If that rectangle would cross the right edge, its X position is backset
left until the complete measured popup fits. This may overlap its parent.
Only a popup wider or taller than the physical surface is clipped.

A vertically constrained popup keeps the selection visible through a
deterministic viewport and top/bottom continuation markers. Zero/tiny
geometry retains semantic open and selection state without invalid cells.
Resize never changes identity or activates an item.

## Snapshots And Automation

`MenuBarDetails` contains a bounded depth-first flat entry list. Entries
expose key, parent, depth, kind, effective label, command state, mnemonic,
top-level placement, structured chord, selected/open flags, and child count.
Nested entries have no placement. Ordered selected and open key paths
distinguish bar-active selection from popup state: bar-active has a root
SelectedPath and an empty OpenPath.

Core and automation snapshots deep-copy all slices and chord modifiers.
Automation validates resource, identity, sibling mnemonic, command-state, and
path bounds. Popup pixels remain ordinary intended-frame cells and therefore
need no menu-specific wire representation.

## Catalog And Verification

`expletives-test` owns one root-level persistent MenuBar and exactly one
visible purpose-specific catalog screen. Start-aligned File, Panels, Layouts,
Controls, Menus, and Dialogs roots organize current and future demonstration
pages; end-aligned Help contains About. Implemented pages and operations are
enabled. Future pages remain visible but disabled with the phase that owns
their implementation. Screen switching uses ordinary public commands and
transactions; it never reparents controls or introduces test-only automation
operations.

Normal Go tests cover construction, immutable copies, invalid trees,
root-chrome ownership, geometry mutation and Layout rejection, surface and
root-constraint resize, row reservation, F10 bar activation, traversal,
disabled selection/nonactivation, mnemonics, nesting, focus restoration,
exact style roles, separators, shadows, measured popup width, child backset,
start/end alignment and popup anchoring, tiny viewports, snapshot paths,
concurrency, and response bounds.

Attached and PTY tests exercise raw Alt-F, F10, Ctrl-Space, arrows, Enter,
Escape, nested popups, commands, resize, Ctrl-C, and clean terminal
restoration through the same logical input path.
