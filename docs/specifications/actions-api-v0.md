# Actions API v0

- Status: Implemented
- Authority: Direct operator instruction to proceed automatically through
  Actions and Menus on 2026-07-30
- Scope: `Button`, `HotkeyBar`, `HotkeyBarItem`, focus, and activation

## Shared Command State

Buttons, HotkeyBar entries, future MenuItems, registered accelerators, raw
keys, and direct automation commands select the same App-scoped `CommandID`
and pass through the same `CommandDefinition` enabled-state check and
`CommandRouter`.

`CommandDefinition` adds canonical bounded `Label`, `DisabledReason`, and
`Checked` values. An empty Label displays the command ID. A disabled command
with no supplied reason receives a bounded fallback reason. Enabled commands
cannot retain a disabled reason. Checked state is carried now for Menu reuse;
Button does not reinterpret it.

Replacing or removing a command immediately republishes any affected Action
presentation. Removing a command leaves an existing Button or HotkeyBar item
as a safely disabled reference; it never bypasses `command_unknown`.

Chord registration remains App-owned through `BindChord`, `ReplaceChord`, and
`UnbindChord`. HotkeyBar discovers the currently bound chord for each item
from that same map instead of retaining a second active-binding table.

## Button

```go
type ButtonOptions struct {
    PanelOptions
    Command  CommandID
    Mnemonic Key
    Default  bool
    Cancel   bool
}

func NewButton(Container, ButtonOptions) (*Button, error)
func (t *Transaction) NewButton(Container, ButtonOptions) (*Button, error)
func (b *Button) Focus() error
func (b *Button) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)
```

Button is a copy-safe, focusable, non-container leaf Control. Command is
required and must be registered when construction commits. Mnemonic is an
optional lowercase ASCII letter or digit; uppercase normalizes. Default and
Cancel are mutually exclusive. At most one surviving Button of each role may
exist under one direct parent.

The effective label and enabled/disabled state come from the referenced
CommandDefinition. The one-row canonical rendering uses stable ASCII
brackets and a one-cell role marker so normal, focused, pressed, disabled,
default, and cancel states remain distinguishable without depending on color.
The semantic Control style still supplies foreground/background/attributes.
The automatic minimum is the normalized command-label width plus five cells
by one row.

`Activate` serializes through normal dispatch, supplies the Button ControlID
as `Command.Target`, performs the ordinary command registry checks, invokes
the ordinary router outside App locks, and returns the exact correlated
Completion. It rejects a hidden, destroyed, or non-Button target.

## HotkeyBar

```go
const (
    MaxHotkeyBarItems = 64
    MaxActionItems    = 4096
)

type HotkeyBarItem struct {
    Command CommandID
}

type HotkeyBarOptions struct {
    PanelOptions
    Items []HotkeyBarItem
}

func NewHotkeyBar(Container, HotkeyBarOptions) (*HotkeyBar, error)
func (t *Transaction) NewHotkeyBar(
    Container,
    HotkeyBarOptions,
) (*HotkeyBar, error)
func (b *HotkeyBar) Items() []HotkeyBarItem
```

HotkeyBar is a copy-safe, non-focusable, non-container leaf. Items are copied,
ordered, unique by CommandID, and limited per control and in aggregate across
the App. Every command must be registered when construction commits.

Each entry renders the command's current first canonical chord and effective
label. Disabled entries remain visible and use parentheses; enabled entries
use surrounding spaces. Clipping removes trailing cells only. Empty and
unbound states remain valid and observable. The automatic minimum is one by
one because command labels and bindings may change independently after
construction and bars normally grow across their Layout axis.

## Focus And Raw-Key Resolution

The App has at most one focused control. `App.Focused` returns its canonical
handle or nil. Button `Focus` and `Transaction.SetFocus` are synchronous
thread-safe mutations. Focus requires a live, effectively visible, enabled,
focusable control. Creating the first eligible Button or invalidating the
current focus selects the first eligible Button in stable control-tree order.

The direct parent Container of a focusable control defines its focus group.
Plain Tab and Shift-Tab traverse eligible groups cyclically in stable
control-tree order rather than visiting every control. Entering a group
chooses its first eligible control when moving forward and its last eligible
control when moving backward; a RadioGroup instead enters through its
selected enabled option when possible.

Plain arrows move focus spatially among eligible controls in the same group.
When no same-group control exists in that direction, focus may enter another
group only when the nearest primary/cross-axis target is unambiguous. Arrow
movement does not imply activation or selection.

A plain Enter or Space activates the focused Button. If no focused Button
handles Enter, the applicable visible default Button is activated. Plain
Escape activates the applicable visible cancel Button.

KeyDown on Enter or Space visually presses the focused Button for that input
source. Matching KeyUp clears the press and activates the captured Button if
it remains eligible. KeyPress activates in one step. Disconnect/reset clears
source-local pressed state without activation. Press capture is bounded by
the existing input-source bound.

An exact Alt plus mnemonic resolves before an ordinary global Alt chord:

- a Button mnemonic activates that Button; and
- a Label mnemonic focuses its target when that target is eligible.

Non-menu Action mnemonics are unique App-wide in this phase. Menu scope and
precedence arrive in the next phase. Other registered chords retain their
existing global command behavior. Text/paste and terminal bytes are not
introduced by this phase.

## Snapshots And Automation

`ControlSnapshot.Focused` exposes generic focus identity. `ActionDetails`
contains effective label, command, enabled flag, disabled reason, mnemonic,
pressed/default/cancel state. `HotkeyBarDetails` contains bounded ordered
items with effective label, command state, and the structured current chord.
All nested slices are deep-copied.

The automation-owned DTO explicitly projects and validates the same typed
union. HotkeyBar items are bounded both per bar and across a snapshot.
Response-line and retained-result proofs include the largest legal Action
control shape and the maximum aggregate action-item contribution.

## Concurrency And MVC

Command definitions and bindings are application presentation/controller
configuration, not domain models. Applications may replace them from any
goroutine through the synchronized App API. Router callbacks remain outside
toolkit locks and may perform ordinary view transactions. Action controls
retain only copied view values and stable command IDs.

Focus, press capture, command state, intended frame, and typed details publish
atomically. No callback runs during focus traversal, layout, or painting.

## Verification

Normal Go tests cover construction, copy safety, leaf capabilities, grouped
Tab/Shift-Tab traversal, spatial arrow traversal and invalidation,
same-parent role uniqueness, every visual state,
label/mnemonic normalization, KeyPress and KeyDown/KeyUp activation,
source-local reset, disabled/unknown rejection, target identity, command
replacement, HotkeyBar binding reflection, clipping, transactions, snapshots,
concurrency, and capacity.

`expletives-test` adds normal/focused/default/cancel/disabled Buttons and a
HotkeyBar. Attached automation activates through grouped navigation and
Enter, Alt mnemonic, a
raw modifier chord, and a direct command, then compares command, target,
outcome, sequence, semantic details, and visible cells. The phase closes only
after ordinary, PTY, race, fuzz, all-mode build, smoke, and live debug
closed-loop checks pass.
