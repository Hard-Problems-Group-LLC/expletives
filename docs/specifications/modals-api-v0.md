# Modals API v0

- Status: Directed pre-v1 contract
- Authority: Phase 17 roadmap and operator full-automatic direction,
  2026-07-31
- Scope: `ModalPanel`, `Dialog`, `MessageBox`, `ConfirmDialog`,
  `InputDialog`, and `ProgressDialog`
- Depends on: Actions, grouped focus, Layouts, text validation, progress,
  Collections, immutable snapshots, and Basic Automation v1

## Design Baseline

The public behavior follows Turbo Vision's dialog vocabulary without copying
its implementation. A dialog is a centered, bordered, shadowed window; Escape
requests cancel, Enter invokes its default action, and terminal actions return
an explicit result. Turbo Vision's current source documents `TDialog` Escape,
Enter, `cmOK`, `cmCancel`, `cmYes`, and `cmNo` handling and distinct dialog
palette roles:

- <https://github.com/magiblot/tvision/blob/master/source/tvision/tdialog.cpp>
- <https://github.com/magiblot/tvision/blob/master/include/tvision/dialogs.h>

Expletives deliberately does not reproduce Turbo Vision's nested modal event
loop. Showing or closing a modal is an ordinary serialized toolkit transition.
The application's existing event loop, command router, immutable publication,
and automation completion barriers remain authoritative.

## Modal Stack And Ownership

One App owns one bounded LIFO modal stack. `MaxModalDepth` is 8. Only the top
modal receives keyboard focus, mnemonics, control activation, default-button
Enter, or cancel-button Escape. Lower modals remain painted and observable but
inactive. Ordinary content remains painted below the stack but is outside the
active input scope.

Every ModalPanel is constructed as a direct child of `App.Root()`. Its
`NestedOwner`, when non-nil, is a logical modal owner rather than a control-tree
parent. This keeps every modal centered and clipped against the Application
Client Area while preserving the ordinary Panel parent contract.

The safe default forbids nesting. A modal with no `NestedOwner` can be shown
only when the stack is empty. A modal with a `NestedOwner` can be shown only
when that owner is the current top modal. This explicit owner rule prevents an
unrelated callback from accidentally capturing input over a modal it does not
own. The depth bound, direct-root parent rule, and top-owner rule are validated
atomically.

Only the top modal may close normally. Destroying an active lower modal closes
it and every modal above it, top-down, with an explicit destroyed result.
Application interrupt or quit finalization closes the complete stack top-down
with corresponding interrupt or quit results before publishing the final
snapshot.

## Lifecycle And Results

```go
const MaxModalDepth = 8

type ModalCloseReason string

const (
    ModalAccepted    ModalCloseReason = "accepted"
    ModalCancelled   ModalCloseReason = "cancelled"
    ModalBack        ModalCloseReason = "back"
    ModalInterrupted ModalCloseReason = "interrupted"
    ModalQuit        ModalCloseReason = "quit"
    ModalFailed      ModalCloseReason = "failed"
    ModalDismissed   ModalCloseReason = "dismissed"
    ModalDestroyed   ModalCloseReason = "destroyed"
)

type ModalResult struct {
    Reason ModalCloseReason `json:"reason"`
    Action CommandID        `json:"action,omitempty"`
}

type Modal interface {
    Container
    Show(initialFocus Control) error
    Close(ModalResult) error
    Active() bool
    Result() (ModalResult, bool)
    Done() <-chan struct{}

    // Unexported identity method: external compounds may embed a real modal.
    modalControlState() *controlState
}
```

Back, cancel, interrupt, and quit are never aliases. Accepted records the
dialog action that committed data. Dismissed is an explicit application close
that is neither a user cancel nor navigation back. Destroyed records lifecycle
removal. An invalid reason or action leaves the modal unchanged.

A ModalPanel is one-shot: constructed, shown at most once, and closed at most
once. This avoids stale result channels, ambiguous reuse after model changes,
and races between an old waiter and a new presentation. Applications create a
new modal instance for another interaction.

`Done` returns one stable receive-only channel. It closes exactly once for
every terminal lifecycle result, including destruction and App finalization.
`Result` returns a copied value and a ready flag. Waiting on `Done` never runs
a nested toolkit event loop and is intended for an application worker, model,
controller, or presenter goroutine; a command router must not block waiting
for an action that requires the currently serialized command dispatch to
finish.

## ModalPanel

```go
type ModalShadowPolicy string

const (
    ModalShadowDefault ModalShadowPolicy = ""
    ModalShadowNone    ModalShadowPolicy = "none"
    ModalShadowTurbo   ModalShadowPolicy = "turbo"
)

type ModalPanelOptions struct {
    PanelOptions
    Title            string
    BorderStyle      StyleID
    BorderForm       BorderForm
    BorderForeground *Color
    BorderBackground *Color
    Shadow           ModalShadowPolicy
    ShadowStyle      StyleID
    NestedOwner      Modal
}

type ModalPanel struct { /* copy-safe Panel-derived container */ }

func NewModalPanel(Container, ModalPanelOptions) (*ModalPanel, error)
func (t *Transaction) NewModalPanel(
    Container,
    ModalPanelOptions,
) (*ModalPanel, error)

func (m *ModalPanel) Show(initialFocus Control) error
func (m *ModalPanel) Close(ModalResult) error
func (m *ModalPanel) Active() bool
func (m *ModalPanel) Result() (ModalResult, bool)
func (m *ModalPanel) Done() <-chan struct{}
```

The construction parent must be the owning App's root. `PanelOptions.Bounds.X`
and `.Y` must be zero. Bounds width and height are the requested bordered body
size; zero on either axis requests the recursively measured minimum on that
axis. `PanelOptions.Hidden` must be false because modal visibility is owned by
the lifecycle. The constructor inserts an inactive, non-painted control that
can receive children and a Layout before `Show`.

Modal input scope remains implicit and confined. `PanelOptions.InputScope`
must be zero for a ModalPanel, and descendants of a modal may not declare a
non-modal scope. Every modal container snapshot reports
`ContainerDetails.InputScope == InputScopeConfined`; this structural evidence
does not alter the existing top-modal capture or nested-modal stack.

Border defaults to `BorderDouble`, body style defaults to `modal_panel`, border
style defaults to `modal_panel.border`, and shadow style defaults to
`modal_panel.shadow`. `ModalShadowDefault` selects the Turbo Vision two-column
right and one-row bottom shadow; `ModalShadowNone` explicitly removes it.

`Show` validates the complete stack policy, closes any menu or collection
popup, cancels any unrelated active editor, captures the prior focus, resolves
geometry, pushes the modal, and publishes one atomic snapshot. A non-nil
initial focus must be an eligible descendant. Nil selects the first eligible
focus group; if no descendant is focusable, focus is nil while the modal still
captures input and remains cancellable according to its policy.

`Close` is safe for concurrent use, validates that this modal is the active
top, records the copied result, hides and pops it, closes `Done`, and restores
focus in one publication. The prior exact control is restored when it remains
eligible in the newly active scope. Otherwise focus repairs to the first
eligible group in the newly active modal or ordinary application.

## Geometry, Resize, And Painting

The bordered modal body is centered in the current Application Client Area:
below the lowest Header or Main Menu, above the highest Footer or Status Bar,
and intersected with the constrained root. Its requested size never uses a
fixed terminal-width magic number.

When space is sufficient, the body receives its requested size enlarged as
needed for its recursive Layout minimum. When the client is smaller, the body
is clamped to the available rectangle and reports degraded geometry. Child
Layouts retain their logical minima and ordinary ancestor clipping, so the
snapshot exposes the shortfall rather than silently shrinking every control.
At least the border/title and cancel path remain visible when the physical
surface has enough cells; zero-sized physical surfaces retain semantic state.

Resize recenters every active modal and recomputes degraded state without
changing stack order or focus identity. Each shadow occupies the two columns
to the body's right and one row below, clipped to the Application Client Area.
Stack painting is stable from bottom to top. Application Main Menu and Status
Bar retain their physical edge rows; modal bodies do not displace Headers,
Footers, or Client Area Layouts.

## Input And Command Scope

Modal capture precedes Menu handling and ordinary application input after the
bounded overflow fallback has had first refusal. While a modal is active:

- Tab and Shift-Tab traverse only focus groups inside the top modal;
- arrows, mnemonics, text input, selection, popup, and activation target only
  descendants of the top modal;
- Enter first lets the focused editor/control handle the key, then invokes the
  top modal's enabled default Button;
- Escape first cancels an active editor or popup as that control defines, then
  invokes the top modal's enabled cancel Button or the modal cancel policy;
- a Menu cannot open and an already-open Menu is closed by `Show`; and
- direct commands with an out-of-scope target are rejected.

`CommandDefinition` gains an explicit modal-availability policy. Its zero
value blocks an un-targeted global chord or direct command while a modal is
active. Applications deliberately mark commands such as configured interrupt,
emergency recovery, or Help as modal-available. A command originating from an
eligible control inside the top modal is local and is not blocked merely
because the same command is unavailable as an un-targeted global shortcut.

```go
type CommandModalPolicy string

const (
    CommandModalDefault CommandModalPolicy = ""
    CommandModalBlocked CommandModalPolicy = "blocked"
    CommandModalAllowed CommandModalPolicy = "allowed"
)
```

Ctrl-C remains a configurable semantic command, not a hard-coded alias for
Escape. If the configured modal-available interrupt command returns
`OutcomeInterrupted`, every open modal records `ModalInterrupted` and the App
publishes its normal final interrupt snapshot. Likewise an allowed quit
command returning `OutcomeExited` records `ModalQuit`. A progress dialog may
also interpret an application-selected command as a cancellation request
without claiming that the App was interrupted.

## Dialog

`Dialog` is a distinct copy-safe ModalPanel-derived container with `dialog`,
`dialog.border`, and `dialog.shadow` theme defaults and automatic scoped
default/cancel Button resolution. Its snapshot kind is `dialog` and its
ModalPanel lifecycle appears in `ModalPanelDetails`. It does not add a second
control tree or event loop.

```go
type DialogOptions struct { ModalPanelOptions }
type Dialog struct { /* copy-safe ModalPanel-derived container */ }

func NewDialog(Container, DialogOptions) (*Dialog, error)
func (t *Transaction) NewDialog(Container, DialogOptions) (*Dialog, error)
```

A Dialog may contain arbitrary ordinary controls and Layouts. At most one
descendant Button may have `Default: true` and at most one may have
`Cancel: true`; the transaction rejects ambiguous roles across the complete
Dialog subtree. Action mnemonics are likewise unique within one modal input
scope, while the same mnemonic may be reused by ordinary application content
or a different inactive modal.
Potentially destructive actions do not become the default implicitly. Client
commands decide application policy and close a generic Dialog explicitly.

### Turbo Vision Appearance

The default gray-dialog presentation uses the original Turbo Vision visual
vocabulary while deliberately keeping the dialog surface distinct from the
lighter Menu palette: active double-line border characters, title, body, and
static message text are white on medium gray, and the outer modal shadow
remains black. Standard-dialog Buttons use the two-row raised Button contract
from [`actions-api-v0.md`](actions-api-v0.md): green body, state-specific text,
yellow mnemonic, and a black right/bottom half-block shadow over the dialog's
medium-gray body. Button shadows are part of each Button's Bounds and are
independent of the already-separate outer modal shadow.

The project-selected default true-color dialog surface is `#808080`; the Menu
surface remains `#AAAAAA`. This darker dialog gray is an authorized permanent
palette choice made to preserve practical contrast in terminal renderers. By
direct operator direction, white dialog body and static-message text on that
surface is an explicitly approved and endorsed permanent variance from Turbo
Vision's black body text. The dialog body, border background, disabled
buttons, and raised-button shadow background use the same gray surface unless
a Theme overrides their semantic styles. Border and title foreground remains
white because the darker gray is rendered distinctly; the dark-blue fallback
variance is therefore not used. Disabled Button text remains governed by its
distinct `button.disabled` role.

Standard MessageBox and InputDialog OK Buttons use K as their mnemonic, while
Yes, No, and Cancel use Y, N, and C where present. Buttons in a horizontal
standard-dialog row retain two cells between their bounded shadow rectangles.
Bottom button sections place two blank dialog-body rows above the two-row
Button and place its bottom shadow immediately above the dialog's bottom
border; they do not retain an additional blank row below the Button. Moving
that reserved row does not change the dialog's requested or measured height.
The source-level compatibility references are Turbo Vision's
[`TDialog` palette map](https://github.com/magiblot/tvision/blob/master/include/tvision/dialogs.h),
[`TButton` renderer](https://github.com/magiblot/tvision/blob/master/source/tvision/tbutton.cpp),
and [`MessageBox` composition](https://github.com/magiblot/tvision/blob/master/source/tvision/msgbox.cpp).

## Standard Dialogs

The standard dialog constructors compose ordinary public controls and Layouts
inside a Dialog. They use the same model, focus, validation, command, and
snapshot paths as consuming applications; they are not private renderer-only
overlays.

### MessageBox

MessageBox displays bounded wrapped StaticText and an OK Button. OK is both the
default and cancel-safe terminal action. Enter and Escape therefore return an
accepted `dialog.ok` action rather than pretending an informational message
was rejected. Its visible accelerator is K, matching Turbo Vision's `O~K~`
label. Long messages use a ScrollablePanel when required.

```go
const CommandDialogOK CommandID = "dialog.ok"

type MessageBoxOptions struct {
    DialogOptions
    Message string
}

type MessageBox struct { /* Dialog-derived standard compound */ }

func NewMessageBox(Container, MessageBoxOptions) (*MessageBox, error)
func (t *Transaction) NewMessageBox(
    Container,
    MessageBoxOptions,
) (*MessageBox, error)
func (m *MessageBox) MessageControl() *StaticText
func (m *MessageBox) MessageViewport() *ScrollablePanel
func (m *MessageBox) OKButton() *Button
```

### ConfirmDialog

ConfirmDialog exposes explicit Yes, No, and optional Cancel choices. The
caller selects the default; no destructive Yes action is selected by default.
If no default is specified, No is the safe default. Escape returns cancelled,
not No. A typed result distinguishes Yes, No, and Cancel without parsing a
label.

```go
const (
    CommandDialogYes    CommandID = "dialog.yes"
    CommandDialogNo     CommandID = "dialog.no"
    CommandDialogCancel CommandID = "dialog.cancel"
)

type ConfirmChoice string

const (
    ConfirmChoiceYes    ConfirmChoice = "yes"
    ConfirmChoiceNo     ConfirmChoice = "no"
    ConfirmChoiceCancel ConfirmChoice = "cancel"
)

type ConfirmDialogOptions struct {
    DialogOptions
    Message    string
    Default    ConfirmChoice
    ShowCancel bool
}

func NewConfirmDialog(Container, ConfirmDialogOptions) (*ConfirmDialog, error)
func (t *Transaction) NewConfirmDialog(
    Container,
    ConfirmDialogOptions,
) (*ConfirmDialog, error)
func (d *ConfirmDialog) Show(initialFocus Control) error
func (d *ConfirmDialog) Choice() (ConfirmChoice, bool)
func (d *ConfirmDialog) MessageControl() *StaticText
func (d *ConfirmDialog) MessageViewport() *ScrollablePanel
func (d *ConfirmDialog) YesButton() *Button
func (d *ConfirmDialog) NoButton() *Button
func (d *ConfirmDialog) CancelButton() *Button
```

The four `dialog.*` commands are immutable toolkit commands. They bypass the
application router only for a matching top standard-dialog target and close in
the same correlated snapshot publication. Un-targeted use, a target outside
the top modal, or an action invalid for that standard dialog is rejected.

### InputDialog

InputDialog composes StaticText/Label and one TextField. It accepts the same
optional soft or hard validator and Password policy as TextField. Initial
focus is the editor. Enter commits through the enabled default action only
when the current value is valid; Escape cancels without publishing the value.
The local typed result returns a copied accepted value. Basic Automation
reports length and validation state but never publishes password text,
validator character sets, or the active value.

```go
type InputDialogOptions struct {
    DialogOptions
    Prompt    string
    Text      string
    Validator *TextValidator
    Password  bool
}

type InputDialog struct { /* Dialog-derived standard compound */ }

func NewInputDialog(Container, InputDialogOptions) (*InputDialog, error)
func (t *Transaction) NewInputDialog(
    Container,
    InputDialogOptions,
) (*InputDialog, error)
func (d *InputDialog) Show(initialFocus Control) error
func (d *InputDialog) Value() (string, bool)
func (d *InputDialog) PromptControl() *StaticText
func (d *InputDialog) PromptViewport() *ScrollablePanel
func (d *InputDialog) Field() *TextField
func (d *InputDialog) OKButton() *Button
func (d *InputDialog) CancelButton() *Button
```

`Value` returns the accepted copied value only after a `dialog.ok` terminal
result. Cancel, dismissal, destruction, interruption, and quit do not expose a
value. Cancel restores the TextField's construction-time value so an in-process
snapshot after closure cannot retain a discarded edit. Automation additionally
redacts the TextField value and validator characters throughout the compound's
lifetime, including after acceptance and regardless of Password mode. It
retains length, caret, validity, enforcement, and whitelist/blacklist mode so
closed-loop clients can still diagnose interaction and validation behavior.

### ProgressDialog

ProgressDialog composes a ProgressBar, bounded status text, and
an optional Cancel Button. User cancellation records an observable
cancellation request, disables repeated cancellation, and cancels one stable
dialog-owned context. It does not declare the background operation stopped or
close the dialog until application code acknowledges completion with an
explicit result. Updates are thread-safe, copied, bounded, and use ordinary
toolkit mutations. Interrupt, cancellation request, operation failure,
successful completion, dialog close, and App quit remain distinct states.

```go
type ProgressDialogState struct {
    Status   string
    Progress ProgressBarState
}

type ProgressDialogOptions struct {
    DialogOptions
    State       ProgressDialogState
    Cancellable bool
}

type ProgressDialog struct { /* Dialog-derived standard compound */ }

func NewProgressDialog(
    Container,
    ProgressDialogOptions,
) (*ProgressDialog, error)
func (t *Transaction) NewProgressDialog(
    Container,
    ProgressDialogOptions,
) (*ProgressDialog, error)
func (d *ProgressDialog) Show(initialFocus Control) error
func (d *ProgressDialog) State() ProgressDialogState
func (d *ProgressDialog) SetState(ProgressDialogState) error
func (d *ProgressDialog) Update(
    context.Context,
    ProgressDialogState,
) error
func (d *ProgressDialog) Context() context.Context
func (d *ProgressDialog) CancelRequested() bool
func (d *ProgressDialog) Complete(ModalResult) error
func (d *ProgressDialog) StatusControl() *StaticText
func (d *ProgressDialog) ProgressControl() *ProgressBar
func (d *ProgressDialog) CancelButton() *Button
```

`State`, `SetState`, and `Update` treat bounded one-line status plus the
complete canonical `ProgressBarState` as one copied application-facing state.
The two child-control mutations publish atomically. Updates may configure an
inactive dialog and continue after a cancellation request, allowing the
application to report orderly shutdown, but a transaction targeting a closed
modal is rejected with `ErrModalState`.

`Context` returns one stable context for the dialog lifetime. A first visible
Cancel activation or Escape request atomically sets `CancelRequested`, cancels
that context, disables the Button, repairs focus, and publishes an applied
`dialog.cancel` completion without closing. Repetition is rejected as
`cancel_requested`. Terminal close also cancels the context to prevent worker
lifetime leakage but does not set `CancelRequested`. The application calls
`Complete` only when it has acknowledged the worker outcome, using
`ModalAccepted`, `ModalFailed`, or `ModalCancelled` as appropriate. A
non-cancellable dialog has no Cancel Button or Escape action.

## Typed Details And Automation

Core `ModalPanelDetails` exposes:

- lifecycle (`inactive`, `active`, or `closed`), top/active flags, stack index
  and depth;
- nested owner, saved focus, and initial focus identities when present;
- requested size, resolved body bounds, degraded flag, recursive minimum,
  shadow policy/style, and border details; and
- the exact terminal `ModalResult`, when ready.

Dialog specialization details add bounded semantic state. ConfirmDialog
exposes its choice set and safe default. InputDialog exposes editor length,
caret, validity, enforcement, mode, and password flag without text or
validator characters. ProgressDialog exposes progress state, cancellation
availability/request state, and bounded status-message length without a
background callback or context object.

Automation projects ModalPanel lifecycle, stack identities, focus identities,
geometry, shadow policy, and terminal result into an exact kind-consistent DTO.
It validates stack ordering and implications and deep-copies the optional
result. Standard-dialog DTOs and their redaction rules arrive with those
controls. Maximum legal modal records remain part of the response-line proof.
Intended
frames remain the authority for exact visible borders, shadows, title,
mnemonics, button roles, clipping, and semantic styles.

Standard automation commands open each `expletives-test` dialog page and
activate every close path. Raw key tests cover Enter, Escape, Tab/Shift-Tab,
mnemonics, configured Ctrl-C lifecycles, nested capture, resize, and focus
restoration. Every close completion carries the snapshot containing the
explicit result before a later catalog transition can obscure it.

## Verification Gate

Phase 17 is complete only when tests cover:

- root-only construction, one-shot lifecycle, invalid results, direct-root
  geometry, shadows, resize, tiny/zero surfaces, and recursive minima;
- exact LIFO nesting, forbidden accidental nesting, depth capacity,
  lower-modal destruction, and stable stack painting;
- focus capture, scoped traversal/mnemonics/default/cancel, editor and popup
  precedence, focus restoration, and out-of-scope direct command rejection;
- distinct accepted, cancel, back, interrupt, quit, dismiss, destroy, failed
  validation, and progress-cancellation-request evidence;
- concurrent show/close/result/update calls, `Done` lifetime, race detection,
  and absence of nested event loops or callbacks under toolkit locks;
- kind-consistent core and automation details, redaction, deep-copy,
  aggregate bounds, maximum response size, and final snapshot association;
- all six modal controls in `expletives-test`, including deterministic normal,
  nested, validation, progress, resized, and small-screen scenarios; and
- debug, release, and profiling builds plus ordinary, race, headless,
  attached-automation, and controlling-PTY verification.
