# Menus API v0

- Status: Implemented pre-v1 contract
- Authority: Direct operator instruction to proceed automatically through
  Actions and Menus on 2026-07-30
- Scope: `MenuBar`, immutable popup `Menu` models, `MenuItem`, keyboard popup
  sessions, and `expletives-test` screen navigation

## Public Model

```go
type MenuItemKind string

const (
    MenuItemCommand   MenuItemKind = "command"
    MenuItemSeparator MenuItemKind = "separator"
    MenuItemSubmenu   MenuItemKind = "submenu"
)

type MenuItem struct {
    Key      string
    Kind     MenuItemKind
    Label    string
    Command  CommandID
    Mnemonic Key
    Menu     *Menu
}

type MenuOptions struct {
    Items []MenuItem
}

func NewMenu(MenuOptions) (*Menu, error)
func (m *Menu) Items() []MenuItem

type MenuBarOptions struct {
    PanelOptions
    Items         []MenuItem
    PopupStyle    StyleID
    BorderStyle   StyleID
    FocusedStyle  StyleID
    DisabledStyle StyleID
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

`Menu` is a copy-safe immutable popup model, not a Control and not an
independent event loop. `MenuItem` is a copied immutable descriptor.
`MenuBar` is a non-container leaf Control and the sole visual/session owner.
An App has at most one live MenuBar in v0.

A command item has a required registered `Command`, optional ASCII mnemonic,
and no caller label or child Menu; its current label, enabled/disabled reason,
checked state, and first binding come from the shared `CommandDefinition`.
A separator has no label, command, mnemonic, or child. A submenu has canonical
bounded `Label`, optional mnemonic, and required child `Menu`.

Top-level MenuBar items are submenus. Item keys are bounded, required, and
unique across one complete MenuBar tree. Mnemonics are unique among sibling
items. Aliased Menu instances and cycles are rejected so one attached value
forms one deterministic tree.

Limits are `MaxMenuDepth == 8`, `MaxMenuItemsPerMenu == 64`,
`MaxMenus == 256`, and `MaxMenuItems == 512`.

## Focus And Session

One App owns at most one menu session. Opening saves the currently focused
Button, gives generic focus to the MenuBar, opens one top-level popup, and
selects its first enabled non-separator item. A menu with no eligible item may
open for inspection but has no selection.

Closing the complete session restores the saved Button when it is still live,
visible, and enabled; otherwise ordinary stable-tree automatic focus selects
the first eligible Button. Programmatic Button focus closes an open menu and
selects that Button. Hiding, destroying, or invalidating the MenuBar closes
the session safely.

Changing command state while a menu is open repairs each invalid selection to
the first eligible item in that level, or leaves that level unselected. It
never permits activation of a disabled or unknown command.

## Keyboard Resolution

Menu handling follows overflow/modal handling and precedes non-menu Action
mnemonics and global chords.

When closed:

- exact Alt plus a top-level mnemonic opens that popup (`Alt-F` is the
  required acceptance path);
- F10 opens the first top-level popup; and
- Ctrl-Space is the documented non-Alt/non-function-key fallback.

F10 and Ctrl-Space close an already open session. While open:

- Up/Down move cyclically among enabled non-separator items;
- Home/End select the first/last eligible item;
- Right opens a selected submenu, or switches to the next top-level menu;
- Left closes one nested submenu, or switches to the previous top-level menu;
- Enter opens a selected submenu or activates a selected command;
- an unmodified sibling mnemonic selects and opens/activates that item;
- Alt plus a top-level mnemonic switches directly to that popup; and
- Escape closes one nested submenu, then closes/restores the complete session
  at the root popup.

Selection never activates. Command activation closes and restores focus
before invoking the shared router outside toolkit locks. Button, HotkeyBar,
MenuItem, mnemonic, bound chord, and direct automation routes therefore use
the same command definition, target-independent command identity, outcome,
and correlated snapshot path.

## Rendering And Geometry

MenuBar occupies one ordinary Layout-managed row. It renders every top-level
label in stable order with a non-color selection/open cue. The App paints open
popup menus as a deterministic overlay after the ordinary control tree and
before the existing overflow warning.

Popups use single-line canonical borders, checked markers, submenu arrows,
structured current shortcut text, a non-color focus marker, and explicit
disabled presentation. Popup, border, focused-row, and disabled-row styles
are semantic MenuBar options with Theme defaults.

Popup rectangles clamp to the current App surface. A constrained popup keeps
the selected row visible using a deterministic viewport and top/bottom
continuation markers. Nested popups prefer the right side and fall back left
when needed. Zero/tiny geometry retains semantic open/selection state without
invalid cells. Resize never changes item identity or activates an item.

## Snapshots And Automation

`MenuBarDetails` contains one bounded flattened entry list in stable
depth-first order. Every entry exposes key, parent key, depth, kind, effective
label, command state, mnemonic, current structured chord, selected/open
flags, and child count. It also exposes the ordered open and selected item-key
paths. Flattening prevents recursive wire depth from tracking Menu depth.

Core and automation snapshots deep-copy all slices and chord modifiers.
Automation validates per-menu, aggregate, depth, identity, sibling mnemonic,
command-state, and path consistency bounds. The response-size proof includes
the maximum aggregate Menu contribution.

## Catalog Screens

`expletives-test` retains one persistent MenuBar and one visible
purpose-specific screen under a shared content Panel. Normal stable commands
select Core/Layout, Text/Display, or Actions screens. Screen switching is an
ordinary public transaction that changes Panel visibility; it does not
reparent controls, create a test-only automation operation, or bypass command
policy.

The initial menus are:

- File: Quit;
- View: Core/Layout, Text/Display, Actions, a separator, and a disabled future
  screen item; and
- Actions: Toggle, Reset, and a nested Stacking menu.

The current screen command is checked. Menu commands, direct commands, and
attached automation produce the same screen state and semantic snapshot.

## Verification

Normal Go tests cover construction/copying, invalid item shapes, tree alias
and capacity rejection, command replacement/removal, focus save/restore,
every open/close path, traversal and skipping, mnemonics, F10, Ctrl-Space,
nested popups, checked/disabled/shortcut rendering, clipping/viewports,
resize, transactions, snapshots, concurrency, and response bounds.

The attached demo proof opens View with raw Alt-V, selects Actions, activates
an Action menu item, opens a nested popup, dismisses/restores focus, switches
screens through the shared command path, and exits through ordinary
application quit policy. PTY coverage proves physical Alt-F, F10, arrows,
Enter, Escape, Ctrl-Space, resize, Ctrl-C, and terminal restoration where the
selected decoder profile exposes those keys.

The 2026-07-30 acceptance run passed `make verify`, including ordinary,
Unix-socket, controlling-PTY, full-race, all-mode build, and smoke/self-check
coverage. The response proof measured 37,214,739 JSON bytes and 111,644,217
bytes for three retained maxima, within the 36 MiB line and 128 MiB retained
budgets. The three-second display/command-label fuzz run executed 285,312
cases, and all packages compiled for CGO-free Linux arm64.
