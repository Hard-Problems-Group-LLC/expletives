# FocusGuideBar API v0

- Status: Implemented v0 contract
- Authority: Direct operator guidance on contextual Footer content
- Scope: dynamic focused-control guidance and per-instance customization
- Depends on:
  [`actions-api-v0.md`](actions-api-v0.md) and
  [`headers-footers-api-v0.md`](headers-footers-api-v0.md)

## Purpose And Placement

`FocusGuideBar` is a non-container, non-focusable, one-row presentation
control. It resolves its text from the App's current focus during the same
locked render that publishes the control tree. It therefore follows human raw
keys, automation raw keys, menu sessions, programmatic focus, visibility
repair, and destruction without a second application event path.

The standard `expletives-test` placement is the highest of three semantic
Footer rows:

1. lowest: global application hotkeys;
2. middle: current-screen hotkeys; and
3. highest: focused-control guidance and advisories.

Hotkey inventory does not belong in the StatusBar.

## Public API

```go
const MaxFocusGuidanceApplicationBytes = 128

type FocusGuidanceMode string

const (
    FocusGuidanceAppend   FocusGuidanceMode = "append"
    FocusGuidanceOverride FocusGuidanceMode = "override"
)

type FocusGuidance struct {
    Mode FocusGuidanceMode
    Text string
}

type FocusGuideBarOptions struct {
    PanelOptions
}

type FocusGuideBar struct { /* copy-safe non-container leaf */ }

func NewFocusGuideBar(
    Container,
    FocusGuideBarOptions,
) (*FocusGuideBar, error)
func (t *Transaction) NewFocusGuideBar(
    Container,
    FocusGuideBarOptions,
) (*FocusGuideBar, error)
func (a *App) SetFocusGuidance(Control, FocusGuidance) error
func (a *App) ClearFocusGuidance(Control) error
func (t *Transaction) SetFocusGuidance(
    Control,
    FocusGuidance,
) error
```

The empty Mode means append. Empty Text clears a customization. Text is
copied, canonical single-line one-cell display text. Application text is
limited to 128 UTF-8 bytes; the resolved generic-plus-application text must
also fit the ordinary 256-byte and 256-cell display-text limits. Invalid
mode, control, text, or size rejects the complete operation without
publication.

Append preserves toolkit guidance and adds application text after an ASCII
separator. Override uses only application text. Guidance is registered on
the target control, not on the bar, so any live FocusGuideBar reflects the
same current application meaning. Destroying a target destroys its
customization with it.

## Generic Guidance

The toolkit supplies concise generic instructions for every currently
focusable built-in kind:

- Button: Enter/Space activation, grouped Tab traversal, and arrow focus;
- Checkbox: Space state change, grouped Tab traversal, and arrow focus;
- RadioButton: arrow/Home/End focus, Space/Enter selection, and group exit;
- CycleField and SelectField: `[`/`]` value changes and arrow focus; and
- MenuBar: arrow navigation, Enter activation, and Escape dismissal.

No focus resolves to `No focused control`. Generic wording may improve
compatibly, but it must accurately describe the implemented keyboard
contract, remain bounded one-row text, and never replace structured shortcut
or semantic state in snapshots.

## Rendering And Typed Evidence

The default style is `focus_guide_bar`. The complete row is filled by normal
control painting. Current text is left-justified and clipped to Bounds,
matching the other footer guidance layers.

`ControlDetails.FocusGuideBar` contains:

- current focused target ID and kind, or empty values;
- exact resolved canonical rendered text; and
- empty, `append`, or `override` customization state.

The automation package explicitly projects, validates, and deep-copies the
same bounded member. No unrestricted property map or callback output crosses
the protocol.

## Concurrency And Acceptance

Construction and customization use ordinary atomic Transactions. Public
convenience calls are thread-safe and publish at most one new snapshot.
Resolution and painting run under the App's serialized render ownership; no
application callback runs under toolkit locks.

Acceptance requires unit tests for automatic focus tracking, append,
override, clear, invalid atomic rejection, exact rendering, typed local and
automation evidence, and raw-key movement. `expletives-test` must expose the
live control in its Focus Guidance Footer and demonstrate at least one
per-instance application addition.
