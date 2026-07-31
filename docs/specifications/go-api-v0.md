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
  and correlated completions; and
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
- `label`, `static_text`, `separator`, and `rule`.

All default definitions resolve to white foreground on black background.

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
func (b *Button) Focus() error
func (b *Button) Activate(
    context.Context, source, requestID string,
) (Completion, error)
func (b *HotkeyBar) Items() []HotkeyBarItem
func (a *App) Focused() Control
```

Both controls are non-container leaves. Button is focusable and references one
registered command. Its effective label, enabled/disabled reason, and checked
state come from that command definition. Default and cancel are exclusive
roles, and an optional ASCII mnemonic activates through the ordinary router.
HotkeyBar copies an ordered unique command inventory and projects the current
first App binding for each entry; it does not retain a second binding table.

Tab and Shift-Tab traverse eligible Buttons. Plain Enter/Space activates the
focused Button, Enter falls back to the applicable default Button, and Escape
uses the applicable cancel Button. Raw Enter/Space down-up pairs publish and
clear source-local pressed capture; input reset and disconnect clear capture
without activation. An exact Alt mnemonic is resolved before a global Alt
binding. `ControlSnapshot.Focused`, `ActionDetails`, and `HotkeyBarDetails`
make the complete state atomically inspectable.

The complete construction, validation, rendering, focus, routing, snapshot,
automation, and resource-bound contract is in
[`actions-api-v0.md`](actions-api-v0.md).

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
func (t *Transaction) SetSize(Size) error
func (t *Transaction) SetRootConstraints(RootConstraints) error
func (t *Transaction) SetBounds(Control, Rect) error
func (t *Transaction) SetMinimumSize(Control, Size) error
func (t *Transaction) SetStyle(Control, StyleID) error
func (t *Transaction) SetVisible(Control, bool) error
func (t *Transaction) SetText(Control, string) error
func (t *Transaction) SetFocus(Control) error
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

Control bounds are parent-client-relative logical rectangles. A plain Panel's
client rectangle is its full bounds. A decorated Frame or GroupBox has a
one-cell client inset; `BorderNone` has none. A decorated Layout likewise
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
`HotkeyBarDetails` for HotkeyBar. These expose canonical bounded text,
alignment, wrap, Label target/mnemonic, divider orientation/form, generic
focus, command presentation state, pressed/default/cancel roles, and
structured current bindings.

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

The supported keys are lowercase ASCII letters, digits, Control/Alt/Shift/Meta,
Space, Enter, Escape, Tab, Backspace, navigation/editing keys, and F1 through
F12. Terminal escape bytes are not valid logical keys.

`BindChord`, `ReplaceChord`, and `UnbindChord` manage structured bindings.
Chord modifier order is insignificant; modifiers must be unique, and the
pressed key cannot itself be a modifier.

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
measure/arrange rules, nesting, `Panel.Raise`/`Lower`,
`Layout.Raise`/`Lower`, snapshot fields, and overflow callback are specified
in [`layout-api-v0.md`](layout-api-v0.md).

## Deferred And Excluded From v0

This contract does not yet provide:

- Layout replacement, detachment, spacers, and control reparenting;
- reparenting;
- a public custom-paint or arbitrary control factory;
- focus, hit testing, selection, activation, mouse input, paste, or editing;
- mutable titles or border styles;
- menus, mnemonic scopes, or accelerator scopes;
- a public cursor mutator;
- a general UI-owner `Post`/`Call` event loop;
- synchronization of caller-owned models;
- terminal guarantees beyond the terminal-adapter contract; or
- authentication or capability authorization for attached automation.

Later controls and phases are listed in
[`control-catalog.md`](control-catalog.md). Build requirements are in
[`build-and-verification.md`](build-and-verification.md).
