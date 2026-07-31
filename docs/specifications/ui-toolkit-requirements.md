# expletives: A Modern Cell-Based TUI Toolkit Inspired by Turbo Vision
# Requirements & Design

Status: Draft

Authority note: this document is design provenance, not implementation
authorization by itself. The directed
[`expletives-test`](expletives-test.md) and
[`build and verification`](build-and-verification.md) requirements,
[`control catalog`](control-catalog.md), and
[`consuming-application architecture`](application-architecture.md), and
[`concurrency and thread safety`](concurrency-and-thread-safety.md) govern
their scopes. In particular, `FormPanel`, `Wizard`, and `StepContainer` are
deferred despite appearing in this historical catalog. The byte-oriented
multi-plane model conflicts with the directed Basic `SnapshotV1` in
[`expletives-test.md`](expletives-test.md), whose cells contain only the
canonical one-cell grapheme, semantic style, resolved foreground/background
colors, and stable owner identity; cursor and bounded typed tree are
snapshot-level facts, with no width or continuation fields. The `panelith`
package names, public inheritance mechanism, terminal backend, and automation
wire encoding remain undecided. Resolve those conflicts through project
decisions before implementation. The directed
[`docs/Limited-Unicode-Support.md`](../Limited-Unicode-Support.md)
specification supersedes all width-two, multi-cell, and continuation-cell
requirements below. Those conflicting passages remain historical design
provenance only and do not authorize implementation or test scope.

Context: distilled from building Go test applications on Bubbletea + Lip
Gloss, where a *string-styling* library was pressed into service as a
*windowing toolkit*. This document specifies what a toolkit must be to
solve those problems permanently, and how it should be packaged for reuse
across Go projects.

The intent is for this toolkit to be heavily inspired by Turbo Vision and
curses, but overridden by modern AI and human automation needs, and general
experience.

The core concepts are an inheritance based common controls library, and a
_multi-plane framebuffer_.

The system renders internally to this muli-plane framebuffer in an
operation referred to as _composition_.  This framebuffer is then rendered
to the host terminal using a method appropriate for the terminal type
detected, in an operation referred to as _presentation_.  The framebuffer
can also be provided to an automation client on demand, written to
disk, or simply copied to a new object in memory, referred to as an _export_
to _automation_, _storage_, or _memory_, the last of which is useful for
unit testing.  Finally, the presentation target can be inspected to build a
framebuffer in reverse; this operation is known as _capture_.

The planes in the framebuffer, some of which are optional, are:
1. a _character plane_,
2. a _style plane_,
3. a _Unicode plane_,
4. an _extended plane_,
5. and a _delta plane_.

Each plane is a "for each row, for each column" two-dimensional array, but
other than the character plane, these are two-dimensional arrays of _structs_
(or struct-like objects).  Automation software requests a frame by indicating
which planes it wants to retrieve; any planes in use that match the request
will be provided; any planes not in use will be explained in the metadata
returned with the frame.

The _character plane_ is 8 bits per cell, in the usual two-dimensional array
that is a typical text mode framebuffer, but with two execeptions.  The
special value 0 indicates 'no character present', which is distinct from space
(ASCII 32), and 255 indicates that a UTF-8 character is present, and the
details are on the _Unicode plane_.

The _style plane_ records the foreground color of the character, the
background color of the cell, the special attributes bold, italic,
underline, and blink (not all of which will be available on all rendering
targets).

The _Unicode plane_ reserves 4 bytes for each character cell to encode a
UTF-8 character (not all characters will use all bytes; unused bytes are
set to 0).  These cells are set fully to zero for characters that are
expressed using only the ASCII character set in the character plane.  If
no character onscreen requires Unicode rendering, frame metadata will
indicate that this plane is not required for the current state of the
framebuffer, notifying automation software not to both retrieving it.

The _extended plane_ includes details useful for debugging:

1. The control ID.  This is a 32-bit serial number of the control that
last drew at this position.  It starts at 0 when the screen is cleared.
2. The _overlap_count_.  This is a 32-bit unsigned integer that is reset
to 0 when a redraw begins.  When anything is drawn to that position,
unless the control being drawn has the same ID as the control ID last drawn,
this value is incremented.  If invalidation, redraw rects, and Z-ordering
are all working properly, this value should rarely be higher than 1.
3. The _redraw_count_.  This is a 32-bit unsigned integer that is reset
to 0 when a redraw begins.  When anything is drawn to that position,
regardless of any other factors, the value is incremented.
4. The _monotonic_timestamp_ that the character was last rendered.  This is
a high-precision timestamp that can express microseconds, but which includes
a UNIX timestamp as a separate member.

The _delta_ plane has an 8-bit bitfield indicating that changes are more
recent than zero or more of the following operations:
Bit 0: Terminal Presentation (the framebuffer has been internally rendered,
       but has not been rendered to the hardware/terminal target).
Bit 1: Export to Automation
Bit 2: Export to Storage
Bit 3: Export to Memory
Bit 4: reserved / available as design needs arise
Bit 5: reserved / available as design needs arise
Bit 6: reserved / available as design needs arise
Bit 7: reserved / available as design needs arise

NOTE: In the above points, "drawn", "changed", or "updated" means a change
to any or all of the character, foreground or background color, or style.

Additionally, the _frame metadata_ structure describes the width and height
of the frame, various terminal emulation data, and the X/Y position of the
cursor.  It also includes the _redraw serial number_ of the frame.  A
boolean indicates whether or not the frame included any UTF-characters; if
not, then the _extended plane_ is not available.

---

## 1. Purpose

A pure-Go, cell-based terminal UI toolkit that gives us a Turbo Vision-class
windowing model: composable panels, a rich common-controls library, correct
backgrounds/overlays/shadows, first-class automation with a **semantic cell
framebuffer**, and advanced keyboard handling — all built on an OO-style
class hierarchy (via a disciplined Go embedding + virtual-dispatch pattern)
so subclassing maximizes reuse and predictability.

It is a standalone module, consumable by any Go project with a normal
`go get`, that TUI applications can build on.

## 2. Problem statement (why a new toolkit)

Lip Gloss styles *strings*; it has no cell grid, no z-order, no compositing.
Every hard bug we hit was that mismatch:

- Nested/joined/overlaid styled strings drop the background on padding and
  border cells (embedded `ESC[0m` resets) → black gaps everywhere; "fixes"
  are manual re-assertions (`padBand`, `fillRegion`, per-line rendering).
- Overlays (dialogs, dropdowns) are hand-spliced with ANSI truncation →
  fragile centering and z-order.
- No cell model → automation must re-parse ANSI to recover per-cell
  attributes.
- No windowing primitives → we hand-roll every control, menu, and dialog.

A cell buffer is the correct model. Curses proves it; we need it in pure Go
(cgo is disallowed for static FactoryOS/edge binaries), with a real controls
library and an automation-native design.

## 3. Design principles (non-negotiable)

1. **Cell-buffer first.** The canonical frame is a 2-D grid of typed cells,
   not a string. All rendering targets the grid. A backend adapter reconciles
   the grid to the terminal with damage-based diffing.
2. **Pure Go, no cgo.** Must build with `CGO_ENABLED=0`. Backend is a pure-Go
   terminal layer (e.g. tcell, or our own vt adapter) — never a libncurses
   binding.
3. **Layered, per the intended-frame model.** Domain state → interaction
   state (widgets) → *pure* render to a cell buffer → backend adapter →
   single event loop. Rendering is a deterministic function of
   `(state, geometry)`; no I/O, no time reads inside paint.
4. **Semantic everything.** Cells and widgets carry semantic tags (roles,
   style names, ids), so tests/automation assert on *meaning*, not pixels or
   raw colors.
5. **Automation is native, not bolted on.** The same seams that drive the UI
   drive tests and the explicitly enabled attached local endpoint. A
   headless in-process driver exists for tests with no terminal.
6. **One owner of the terminal.** Exactly one Screen object reads input and
   writes output. Workers communicate via bounded channels; no widget writes
   to the tty.
7. **OO by discipline.** A single base class, a documented virtual-dispatch
   pattern, deep-but-shallow-in-practice hierarchies. Overriding one method
   changes behavior predictably at every level.
8. **Capability detection once, at startup.** Never query the terminal
   (background color, DA, cursor position) while the loop is running — it
   deadlocks against the input reader. (Hard lesson: CE-QUIRK-001.)
9. **Testable, deterministic, side-effect-free construction.** No init-time
   I/O, no hidden goroutines, no global mutable state.
10. **Accessible/degradable.** Monochrome + reduced-decoration modes; never
    convey state by color/shape alone; a plain linear mode for non-TTY output
    and screen readers.

## 4. Packaging & consumption (make it trivial to pull and use)

- **One module, stable path.** e.g. `github.com/Hard-Problems-Group-LLC/expletives`
  (or an HPG namespace). Semantic-versioned, tagged releases; never break
  a tagged API without a major bump.
- **`go.mod` hygiene.** Declared minimum Go (`go` directive = real minimum),
  no `toolchain` pinning imposed on consumers, buildable with
  `CGO_ENABLED=0` on linux/amd64 and linux/arm64. Minimal dependency graph —
  ideally only the terminal backend and a width/grapheme library; no
  transitive markdown/HTTP/logging pulls.
- **No side effects on import.** No `init()` I/O, no global registration, no
  goroutines started at package load. The Screen and event loop are created
  explicitly by the app.
- **Package layout** (subpackages a consumer imports à la carte):
  - `panelith` — root: `Screen`, `App`, `Run`, top-level wiring.
  - `panelith/cell` — `Cell`, `Buffer`, colors, attributes, semantic tags.
  - `panelith/widget` — `Base`, `Panel`, and every control.
  - `panelith/layout` — layout managers (flex/grid/stack/anchor).
  - `panelith/input` — key/mouse decoding, semantic events, key model.
  - `panelith/theme` — semantic style registry, palettes, monochrome map.
  - `panelith/auto` — automation endpoint + headless driver + snapshot types.
  - `panelith/backend` — terminal adapters (tcell-backed, and a headless
    buffer backend for tests).
- **Consumption shape.** Constructors return concrete pointer types that also
  satisfy interfaces; embedding a control's struct gives you a subclass.
  Example intended usage:
  ```go
  app := panelith.New(panelith.Options{Theme: theme.TurboVision()})
  dlg := widget.NewConfirmDialog("Quit?", widget.YesNo).DefaultYes()
  app.Root().Add(myFormPanel)
  app.ShowModal(dlg)
  if err := app.Run(ctx); err != nil { ... }
  ```
- **Docs & examples.** Every exported type documents blocking/concurrency,
  nil/zero behavior, and ownership. A `examples/` tree with runnable programs
  (a form, a dialog, a table, a wizard, an automation-driven test). A
  `doc.go` per package. Godoc-clean.
- **Verification is the consumer's, but shipped harness.** The toolkit ships
  no linters/formatters, but ships the headless driver and cell-assertion
  helpers so consumers can write acceptance tests without a tty.
- **License:** its own repo, MIT (matching HPG's other tooling); it is a
  reusable library, not product-proprietary.

## 5. Architecture overview

```
            ┌─────────────────────────────────────────────┐
 App/domain │ crate model, business logic (no UI imports)  │
            └───────────────┬─────────────────────────────┘
                            │ semantic events / data binding
            ┌───────────────▼─────────────────────────────┐
 Widgets    │ Panel tree: interaction state + pure paint   │
            └───────────────┬─────────────────────────────┘
                            │ paint(buffer, rect)
            ┌───────────────▼─────────────────────────────┐
 Cell frame │ Buffer: [][]Cell  (the intended frame)       │
            └───────────────┬─────────────────────────────┘
                 diff/damage │        ▲ snapshot (automation/tests)
            ┌───────────────▼────────┴────────────────────┐
 Backend    │ terminal adapter (tcell / headless)          │
            └───────────────┬─────────────────────────────┘
                            │ single event loop (input+resize+paint)
            ┌───────────────▼─────────────────────────────┐
 Input      │ raw key/mouse → semantic events (controller) │
            └─────────────────────────────────────────────┘
```

The App owns the loop. The terminal adapter decodes terminal input into
stable input events before the controller resolves mnemonics, accelerators,
hotkeys, bindings, and commands; widgets never see raw escape bytes.
Painting is a pure walk of the panel tree into the cell buffer; the backend
diffs and flushes. Automation reads the buffer + the widget tree and has two
deliberate injection points in the same controller path:

- raw key-lifecycle events (`KeyDown`, `KeyUp`, and `KeyPress`) using stable
  key identities, before shortcut and command resolution; and
- direct semantic commands after resolution for precise control.

Here, "raw key" means an unresolved logical key event with source-local
pressed-key and modifier state. It never means an arbitrary terminal escape
byte stream or a backend-specific numeric key code.

## 6. The cell buffer & semantic framebuffer

`cell.Cell` (value type, cheap to copy):

- one canonical grapheme string that the project width policy measures as
  exactly one cell, including supported base-plus-combining sequences;
- no width or continuation-cell member; unsupported zero-width, multi-cell,
  or indeterminate elements normalize to one `U+FFFD` cell;
- `FG, BG cell.Color` — resolved color (ANSI-16 / 256 / truecolor, or a
  named palette ref that the theme resolves).
- `Attrs cell.Attr` — bit flags: Bold, Faint, Italic, Underline, Reverse,
  Blink, Strike.
- `Style theme.StyleID` — the **semantic** style name applied ("canvas",
  "hotkey", "focused-row", "disabled", …). This is what tests assert on.
- `Owner widget.ID` — which widget painted this cell (for per-control
  snapshots and hit-testing).

`cell.Buffer`:

- Fixed `W×H`, addressable `At(x,y)`, `Set(x,y, Cell)`, `Fill(rect, Cell)`.
- `SubBuffer(rect)` / clip regions so a widget paints only within its rect.
- **Blit(dst, srcRect, at, z)** for overlays: composite a sub-buffer at a
  position with z-order — this is how dialogs/dropdowns/shadows compose,
  correctly, with no ANSI splicing.
- **Diff(prev) []Damage** for minimal terminal updates.
- **Snapshot()** → an immutable, serializable framebuffer (below).

**Semantic framebuffer** (the automation/read-out format — the thing we said
we always want): a struct, never a raw ANSI string:

```go
type Snapshot struct {
    W, H  int
    Cells [][]CellView      // each: Grapheme string, Width, FG, BG, Attrs, Style, Owner
    Cursor Cursor           // pos, visible, shape
    Tree  WidgetNode        // the widget hierarchy: id, role, rect, state, text, children
}
```

Consumers/tests never parse escape codes. "Is every cell backed by a
background?" is `cell.BG != Default`. "Is the focused row styled focused?" is
`cell.Style == "focused-row"`. Golden tests diff `Snapshot` structurally.

## 7. Object model & inheritance in Go

Go has no classes or virtual methods. We get inheritance-with-override via a
**shared base struct + a self-interface (template-method) pattern**. This is
the core of the "everything descends from Panel" requirement.

### 7.1 The `Widget` interface and `Base`

```go
type Widget interface {
    // identity & tree
    ID() ID
    Parent() Widget
    Children() []Widget
    // geometry
    Rect() Rect
    SetRect(Rect)
    Measure(avail Size) Size          // preferred size
    // lifecycle
    init(self Widget)                 // wires virtual dispatch (called once)
    // paint & input (the overridable "virtuals")
    Paint(b *cell.Buffer)             // template: frame + PaintContent
    PaintContent(b *cell.Buffer)      // override point for content
    Handle(ev Event) bool             // override point for input
    Focusable() bool
    // semantic
    Role() string
    State() map[string]any            // for the snapshot tree
}
```

`Base` implements the *default* of every method and is embedded by every
widget. The trick: `Base` holds `self Widget` (the outermost concrete
object). Any base algorithm that must reach an override calls through
`self`, not through its own method set:

```go
type Base struct {
    self     Widget      // the outermost object; set by init()
    parent   Widget
    rect     Rect
    style    theme.StyleID
    focused, disabled bool
    // ...
}

func (b *Base) init(self Widget) { b.self = self }

// Template method: draws the frame, then dispatches content to the subclass.
func (b *Base) Paint(buf *cell.Buffer) {
    b.paintBackground(buf)     // fill rect with the resolved background
    b.self.PaintContent(buf)   // <-- virtual dispatch: subclass override runs
    b.paintChildren(buf)
}
func (b *Base) PaintContent(buf *cell.Buffer) {} // default: empty
```

Every constructor wires `self` exactly once:

```go
func NewListBox() *ListBox {
    lb := &ListBox{}
    lb.init(lb)   // sets Base.self = lb through the embedding chain
    return lb
}
```

Because `self` points at the outermost object, an override *at any level* of
the embedding chain is reached. `ComboBox` embeds `ListBox` embeds
`ScrollablePanel` embeds `Panel` embeds `Base`; `Base.Paint` calls
`self.PaintContent`, and if `ComboBox` overrides `PaintContent`, that runs;
otherwise it falls through embedding to `ListBox`, then `ScrollablePanel`,
etc. This gives predictable, classical virtual dispatch.

### 7.2 Subclassing rules (documented, enforced by lint/examples)

- Every concrete widget **must** call `x.init(x)` in its constructor.
- Overrides call the embedded parent explicitly for "super" behavior:
  `cb.ListBox.PaintContent(buf)` then add the combo arrow.
- The overridable set ("virtuals") is fixed and documented: `PaintContent`,
  `Handle`, `Measure`, `Layout`, `Focusable`, `OnFocus/OnBlur`, `Role`,
  `State`. Base algorithms only ever dispatch to these through `self`.
- Non-virtual base helpers (geometry, background fill, child management) are
  not meant to be overridden and are lowercase/unexported where possible.

### 7.3 Composition where inheritance is wrong

Prefer embedding for "is-a" (ComboBox is-a ListBox). Use fields/interfaces
for "has-a" (a ScrollablePanel *has* a ScrollBar; a Dialog *has* a Button
row). Selection models, validators, and data sources are injected
interfaces, not base methods, so they can be swapped without subclassing.

## 8. Control hierarchy & catalog

`Base` (embedded by all) → `Panel` is the root concrete class. Indentation =
"descends from / embeds".

```
Panel                         rect, background, border(optional), children, focus
├─ Frame / GroupBox           Panel + title + border (labeled container)
├─ Label / StaticText         non-focusable text; alignment; wrap
├─ Button                     hotkey, activate, Turbo-Vision drop shadow
├─ Checkbox                   bool; [x]/[ ]; label
├─ RadioButton / RadioGroup   exclusive selection within a group
├─ CycleField (SelectField)   fixed options, ◄ value ► cycling
├─ TextField                  single-line editor, edit-gate, validators
│  └─ NumberField / SpinBox   numeric constraints, step, min/max
├─ ProgressBar / Meter        determinate/indeterminate; textual fallback
├─ Spinner / ActivityDots     animation tick; honors reduced-motion
├─ Separator / Rule           horizontal/vertical divider
├─ ScrollBar                  component; track/thumb from viewport ratio
├─ MenuBar                    top accelerators; audited fallback focus
├─ StatusBar                  bottom band; contextual segments
├─ HotkeyBar                  ordered items: Label | Hotkey | Comment
├─ TabbedPanel / Notebook     tab strip + page container (Diagnostics tabs)
├─ FormPanel                  labeled field rows + focus traversal + submit
├─ Wizard / StepContainer     ordered steps, nav contract, gating
├─ ScrollablePanel            Panel + content>viewport, offset, scrollbars
│  ├─ Viewport                arbitrary scrollable content
│  │  └─ MarkdownView         markdown → cells (help viewer)
│  ├─ TextArea                multi-line editor, wrap, caret, selection
│  ├─ LogView / StreamView    bounded ring buffer, follow/scrollback, drops
│  ├─ ListBox                 items, selection model, keyboard nav
│  │  └─ ComboBox / DropDown  collapsed value + popup = ScrollablePanel of entries
│  ├─ TreeView                node model, expand/collapse, guides
│  └─ Table / DataGrid        columns, row/cell focus, sort, sticky header
└─ ModalPanel / Dialog        Panel + title bar + shadow + centering + input capture
   ├─ MessageBox / ConfirmDialog   message + button row (Yes/No/Quit…)
   ├─ InputDialog                  prompt + field(s)
   ├─ ProgressDialog               long-op progress + cancel
   └─ Menu (popup)                 ScrollablePanel of MenuItems (dropdown body)
```

`MenuItem`, `ListItem`, `TreeNode`, `Column`, `Tab`, `Button`,
`HotkeyBarItem` are the small data/leaf types those containers own.

### ComboBox worked example (the requested shape)

`ComboBox` embeds `ListBox` (so it inherits the item model, selection, and
keyboard nav). Overrides:

- `PaintContent`: when collapsed, draw only the selected item + a `▾` arrow
  in a one-row field; when expanded, also blit the popup.
- The popup is a `ScrollablePanel` (actually a `Menu`/`ListBox` in a
  `ModalPanel`) full of entry rows, opened below/above the field with
  z-order and its own scrollbar — reusing ListBox rendering for the entries.
- `Handle`: `Activate`/`Space` toggles the popup; while open, arrow/Enter/Esc
  are delegated to the embedded ListBox; selection collapses the popup.
- `Measure`: one row collapsed; popup height clamped to available space.

## 9. Unusual / beyond-standard behaviors required

Beyond the textbook behavior of each control, we specifically need:

1. **Edit-gated text fields.** A focused single-line field does **not**
   capture printable input until activated with `[Enter]`; before that, bare
   letters remain screen hotkeys. `[Enter]`/`[Esc]` leave edit mode. (In a
   modal, fields may auto-activate since no hotkeys compete.) This is a
   first-class field state, not a hack. (Our CE-DEC-008.)
2. **Input and semantic-command boundary.** Controls consume semantic events
   (`activate`, `toggle`, `move-up`, `text`, `paste`, `back`, `cancel`, …),
   never terminal escape bytes or backend-specific numeric key codes. The
   controller accepts raw `KeyDown`, `KeyUp`, and `KeyPress` events expressed
   with stable key identities, maintains source-local pressed-key and
   modifier state, then resolves mnemonics, accelerators, hotkeys, bindings,
   and commands. Automation and headless tests may enter there to exercise
   that resolution or submit a direct semantic command for precise control.
3. **Structured hotkeys.** A hotkey is data (`key`, `label`, `enabled`,
   `event`), rendered as `[K]abel` by parsing structure — never by scanning a
   display string for brackets. Disabled hotkeys stay visible with a textual
   cue. The key glyph is styled distinctly (e.g., red).
4. **Focus-aware hotkey bar.** A bar that shows the keys valid for the
   *currently focused control type and its state* (e.g., a TextField shows
   "[Enter] to edit … [^] load default"; once editing, a comment "type, then
   [Enter] to commit or [Esc] to discard"). Requires controls to publish
   their contextual key affordances.
5. **Four-way key semantics distinction.** Back, Cancel, Interrupt, and Quit
   are distinct operations, never conflated. `Ctrl-C` has a robust,
   documented, configurable interrupt policy that remains recoverable inside
   editors, modals, long-running work, and automation.
6. **Turbo Vision chrome.** Drop shadows (half-block shading below/right) on
   dialogs and buttons; row-focus bar + distinct focused-cell in tables;
   honest scrollbars sized from viewport ratio; stable menu/status bands;
   graceful degradation of decoration at small geometry (never corrupt the
   frame).
7. **Modal capture + z-order.** A modal owns all input and composites above
   siblings; opening/closing restores focus exactly.
8. **Resize at any time.** Every control re-lays-out on any resize; overlays
   re-center; scroll offsets re-clamp; nothing panics or bleeds past its
   rect. (Hard requirement.)
9. **Streaming/bounded content.** `LogView` ingests high-volume subprocess
   output via bounded channels with backpressure and honest drop-counting;
   follow-tail vs. scrollback; raw bytes clipped, never wrapped into bands.
10. **One-cell grapheme correctness** in rendering and text editing:
    supported composed one-cell clusters remain indivisible, while every
    unsupported cluster becomes exactly one `U+FFFD` cell.
11. **Disposition/gating.** Controls can be disabled *with a reason* surfaced
    to the user and to automation state.
12. **Validation hooks.** Fields carry validators; invalid state is a visible
    control state and a snapshot fact, and can gate form submission.
13. **Deterministic paint.** Given identical state+geometry, identical cell
    buffer — required for golden-snapshot tests.

## 10. Input & advanced keyboard handling

- **Raw→semantic decoding in one place** (`panelith/input`), handling the
  terminal encoding boundary: Enter as CR/LF/keypad; `Esc` ambiguity (bare
  key vs Alt-prefix vs sequence start) via a bounded, configurable delay with
  incomplete-input retention; multiple `Home`/`End`/`PgUp` encodings;
  `Shift-Tab` via `key_btab`/`CSI Z`; bracketed paste as **bounded text, not
  commands**; mouse (SGR) and focus events as negotiated modes.
- **Enhanced-keyboard protocols.** Detect and, if present, use
  `modifyOtherKeys` / the Kitty keyboard protocol for unambiguous modified
  keys; degrade cleanly when absent. Never make essential navigation depend
  on a single modified key.
- **Raw key lifecycle.** `KeyDown` and `KeyUp` maintain pressed-key state for
  their input source, including modifier keys. `KeyPress` is an atomic
  convenience that does not leave a key held. Modifier chords such as
  `Alt-I` and `Ctrl-S` enter mnemonic, accelerator, hotkey, and binding
  resolution with the same source-local state whether they came from the
  terminal adapter, headless tests, or attached automation. Disconnect,
  cancellation, failure, and shutdown clear or synthesize release for that
  source so a modifier cannot remain stuck.
- **Accelerators & hotkeys.** Global hotkeys; menu `ALT`+letter with
  collision-audited F9 and Ctrl-Space fallbacks; per-control hotkeys;
  edit-gate suppression while a field captures text. Project-selected
  defaults follow
  [`../Terminal-Shortcut-Compatibility.md`](../Terminal-Shortcut-Compatibility.md).
- **Multi-stroke sequences** (future-proofing): a pluggable key-sequence
  resolver (e.g. `Ctrl-X Ctrl-S`) with timeout. This is distinct from the
  required simultaneous modifier chords represented by the raw key
  lifecycle.
- **Bindings are data**: inspectable and, for long-lived tools, configurable;
  a help/keymap view is generated from the same binding tables the bars use.
- **IME / composed input** treated as committed text events.
- **Paste safety**: pasted control sequences are never interpreted as key
  commands; OSC/DCS in streamed content is treated as untrusted and not
  replayed.

## 11. Focus, selection, activation

Three distinct, separately-modeled states:

- **Focus** — the widget receiving navigation/command events; visible focus
  indicator that never relies on color alone; Tab/Shift-Tab across
  focusables in visual order; arrows within a composite; focus stays visible
  after scroll/resize; modal entry moves focus in, close restores it.
- **Selection** — marked item(s); may follow focus only when single-select
  and preview is cheap; kept independent for multi-select or when moving must
  not discard the selection set.
- **Activation** — open/commit/confirm; `Enter` (and control-specific keys);
  destructive activation must be explicit and distinguishable from
  navigation; a default button must not make an accidental `Enter`
  destructive.

## 12. Layout

- Managers in `panelith/layout`: `Flex` (row/col, grow/shrink), `Grid`,
  `Stack` (z-layers for overlays), `Anchor` (dock N/S/E/W/fill), `Absolute`.
- `Measure(avail) → preferred`, then a two-pass arrange; containers assign
  child rects. Layout is deterministic and re-run on resize.
- Bands (menu/status/hotkey bars) are anchored; the body is fill; dialogs are
  centered by a Stack/overlay layer with computed offsets (both axes).

## 13. Styling / theming

- **Semantic style registry**: named styles ("canvas", "menu-bar",
  "menu-accel", "focused-row", "hotkey", "disabled", "dialog", "error", …)
  → concrete (fg, bg, attrs). Widgets reference names; concrete colors live
  only in the theme.
- **Theme = one object**, swappable; ships a Turbo Vision palette and a
  **monochrome/reduced-decoration** map (reverse/underline/weight + textual
  cues; never color-only meaning).
- **Cells carry the style name** (§6) so snapshots assert semantic styling.
- Backgrounds are a cell property, so composition/overlays/shadows never
  produce "black gaps."

## 14. Automation & introspection

- **In-process headless driver** (no socket, for tests): submit raw
  `KeyDown`, `KeyUp`, and `KeyPress` events before shortcut resolution or
  direct semantic commands after resolution, then read `Snapshot`;
  correlated completion means an injected event's effect is applied and
  repainted before the call returns, no-ops complete explicitly, and
  timeouts report failure rather than success.
- **Opt-in attached endpoint** (default-deny, per
  [`expletives-test.md`](expletives-test.md)): a normal run exposes nothing;
  the explicit per-process `--automation <socket-path>` option opens an
  app-owned, bounded local control plane. Initial operator-controlled use may
  be unauthenticated; authentication and capability authorization are deferred
  work. `observe` returns an atomic semantic cell framebuffer + bounded widget
  view. `inject` accepts request-correlated raw `KeyDown`, `KeyUp`, and
  `KeyPress` events using stable key identities, including held modifiers and
  modifier chords, before shortcut resolution; it also accepts direct
  semantic commands. It does not accept arbitrary terminal escape bytes or
  terminal-library numeric key codes. Exit publishes a final snapshot.
- **Widget-tree queries**: find by id/role/text; assert state (focused,
  selected, disabled+reason, value, validity). This is what makes UI
  acceptance tests robust and readable.
- Captures are sensitive (may contain secrets in streamed output) — redaction
  and retention are part of the contract.

## 15. Testing & acceptance

- Ship cell-assertion helpers: `AssertNoDefaultBackground`, `AssertRowText`,
  `AssertStyleAt`, `AssertCentered(widget)`, `AssertFocused(role)`.
- Golden `Snapshot` tests (structural diff, not byte diff).
- A resize-sequence harness (shrink below min, grow back) asserting no panic
  and consistent regions in every state (screens, overlays, modals).
- Headless driver drives full flows (open dialog → default button → assert
  effect) with zero terminal dependence.
- `expletives-test` exercises every public control and important state as a
  human-runnable public-API consumer, with optional attached automation for
  closed-loop diagnosis.
- Ordinary Go unit and integration tests remain the baseline. PTY,
  real-terminal, race, fuzz, and attached-automation checks cover the
  boundaries those tests cannot.

## 16. Accessibility & degradation

- Monochrome and reduced-decoration modes are first-class, not afterthoughts.
- No meaning by color/shape/animation alone; every state has a textual cue.
- A **plain/linear output mode** when stdout is not a tty (CI, pipes, screen
  readers): render the semantic tree as readable text instead of a
  full-screen frame.
- Reduced-motion honored by spinners/progress.

## 17. Lifecycle, threading, resize, terminal safety

- One goroutine owns the tty (input+paint). Widgets never touch it; workers
  post messages/events through bounded queues.
- Terminal modes (alt-screen, paste, mouse, cursor) restored on **every**
  exit path including panic; job-control stop/continue handled; capability
  detection only at startup.
- Resize is a coalesced geometry event that re-lays-out everything.
- Deterministic teardown; automation endpoint (if any) closes and unlinks its
  own socket, unblocking connected clients.

## 18. Non-goals

- Not a general application framework, DB layer, or async runtime.
- No cgo, no bundled fonts, no image protocols (initially).
- Not a Bubbletea/Lip Gloss shim — a clean cell-based toolkit. (Glamour-style
  markdown may be used *internally* by `MarkdownView` by rendering into
  cells, not by adopting a string-styling model.)
- Ships no verifiers/linters; consumers own their toolchain.

## 19. Open questions

1. Backend: adopt **tcell** as the cell/screen/input backend (fast path,
   mature, pure-Go, handles wide chars + damage) vs. write a thin vt adapter
   ourselves. Recommendation: start on tcell behind our own `backend`
   interface so it can be swapped.
2. Do we adopt any **tview** controls as a reference/bootstrap, or build the
   whole hierarchy fresh on the cell backend for a consistent OO model?
   (tview is *not* subclass-friendly the way this design wants.)
3. Enhanced-keyboard protocol support scope for v1 (Kitty/modifyOtherKeys).
4. Truecolor vs. 256 vs. 16 policy and the palette mapping for FactoryOS
   target terminals.

---

*This is a design target, not a committed plan. It captures what "solved all
our problems" looks like so we can evaluate build-vs-adopt (tcell / tview /
own) against a concrete bar before writing code.*
