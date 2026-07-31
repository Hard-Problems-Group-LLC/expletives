# Selection API v0

- Status: Directed pre-v1 contract
- Authority: Phase 11 implementation contract, 2026-07-31
- Scope: Checkbox, RadioButton/RadioGroup, and fixed-option
  CycleField/SelectField controls
- Depends on:
  [`actions-api-v0.md`](actions-api-v0.md),
  [`layouts-and-overflow.md`](layouts-and-overflow.md), and
  [`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md)

## Purpose

Selection controls provide bounded, typed values while keeping focus,
selection, and application activation distinct. They use the App's serialized
raw-key path and optional command-router notification so MVC and MVVM
consumers do not need a second event system.

Turbo Vision supplies the appearance and keyboard reference. Win32,
wxWidgets, and Motif supply supporting conventions for checkbox state and
group-owned radio exclusivity. No implementation is copied.

## Shared Option Model

```go
const (
    MaxSelectionOptions = 256
    MaxSelectionItems   = 1024
)

type SelectionOption struct {
    Value          string
    Label          string
    Disabled       bool
    DisabledReason string
}
```

Values are nonempty bounded identifiers and are unique within one field.
Labels are nonempty, single-line, canonical one-cell display text. A disabled
option has a nonempty bounded reason; an enabled option has no disabled
reason. Constructors and mutations copy the complete slice.

An empty option slice is valid. A nonempty selected value must identify an
enabled option. Replacing options while omitting or removing the current value
selects the first enabled option; if none exists, the selected value is empty.
This deterministic repair rule also applies after a selected RadioButton is
destroyed.

`MaxSelectionOptions` bounds one CycleField/SelectField inventory.
`MaxSelectionItems` bounds the App-wide aggregate of RadioButtons and copied
fixed options so typed snapshots and automation responses have an honest
aggregate resource ceiling.

## Checkbox

```go
type CheckState string

const (
    CheckUnchecked     CheckState = "unchecked"
    CheckChecked       CheckState = "checked"
    CheckIndeterminate CheckState = "indeterminate"
)

type CheckboxOptions struct {
    PanelOptions
    Label          string
    Mnemonic       Key
    State          CheckState
    ThreeState     bool
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

type Checkbox struct { /* copy-safe Panel-derived leaf */ }

func NewCheckbox(Container, CheckboxOptions) (*Checkbox, error)
func (t *Transaction) NewCheckbox(
    Container,
    CheckboxOptions,
) (*Checkbox, error)
func (c *Checkbox) State() CheckState
func (c *Checkbox) SetState(CheckState) error
func (t *Transaction) SetCheckState(*Checkbox, CheckState) error
func (c *Checkbox) Focus() error
func (c *Checkbox) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)
```

The empty construction state means unchecked. Indeterminate is rejected
unless `ThreeState` is true. Space or mnemonic activation cycles
unchecked → checked → indeterminate → unchecked for a three-state Checkbox
and toggles unchecked ↔ checked otherwise. Enter retains its ordinary
default-Button meaning.

Turbo Vision markers are `[ ]`, `[X]`, and `[-]`. Focus remains visible
without color alone, and the mnemonic cell uses the Theme's selection
mnemonic role.

## RadioButton And RadioGroup

```go
type RadioGroupOptions struct {
    PanelOptions
    AllowEmpty     bool
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

type RadioButtonOptions struct {
    PanelOptions
    Value          string
    Label          string
    Mnemonic       Key
    Selected       bool
    Disabled       bool
    DisabledReason string
}

type RadioGroup struct { /* copy-safe Panel-derived Container */ }
type RadioButton struct { /* copy-safe Panel-derived leaf */ }

func NewRadioGroup(Container, RadioGroupOptions) (*RadioGroup, error)
func NewRadioButton(*RadioGroup, RadioButtonOptions) (*RadioButton, error)
func (t *Transaction) NewRadioGroup(
    Container,
    RadioGroupOptions,
) (*RadioGroup, error)
func (t *Transaction) NewRadioButton(
    *RadioGroup,
    RadioButtonOptions,
) (*RadioButton, error)
func (g *RadioGroup) Value() string
func (g *RadioGroup) SetValue(string) error
func (t *Transaction) SetRadioValue(*RadioGroup, string) error
func (b *RadioButton) Value() string
func (b *RadioButton) Selected() bool
func (b *RadioButton) Focus() error
func (b *RadioButton) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)
```

A RadioButton must be a direct child of its RadioGroup. Values and mnemonics
are unique among live siblings. At most one construction child is selected.
Unless `AllowEmpty` is true, the group selects its first enabled child when it
would otherwise be empty. An empty or all-disabled group remains empty and is
not focusable.

Tab treats a RadioGroup as one focus group and enters its selected enabled
button, or first enabled button. Arrows move focus spatially among enabled
options without changing the group's selected value. Home and End focus the
first and last enabled option without selecting it. Space or Enter selects
the focused option and therefore deselects the previously selected option.
An option mnemonic focuses and selects its identified option. The markers
are `( )` and `(•)`; physical-terminal fallback may map the one-cell bullet
without changing the intended frame.

The RadioGroup is the state and exclusivity owner. RadioButton snapshots
repeat the effective selected fact for direct inspection, but no button owns
an independent checked bit.

## CycleField And SelectField

The project hierarchy uses `CycleField (SelectField)` for one fixed-option
cycling control. `SelectField` is a public naming variant over the same
behavior, not the popup `DropDown`/`ComboBox` scheduled for Collections.

```go
type CycleFieldOptions struct {
    PanelOptions
    Label          string
    Mnemonic       Key
    Options        []SelectionOption
    Value          string
    Clamp          bool
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

type SelectFieldOptions = CycleFieldOptions

type CycleField struct { /* copy-safe Panel-derived leaf */ }
type SelectField struct { /* same behavior, distinct control kind */ }

func NewCycleField(Container, CycleFieldOptions) (*CycleField, error)
func NewSelectField(Container, SelectFieldOptions) (*SelectField, error)
func (t *Transaction) NewCycleField(
    Container,
    CycleFieldOptions,
) (*CycleField, error)
func (t *Transaction) NewSelectField(
    Container,
    SelectFieldOptions,
) (*SelectField, error)
func (f *CycleField) Options() []SelectionOption
func (f *CycleField) Value() string
func (f *CycleField) SetValue(string) error
func (f *CycleField) SetOptions([]SelectionOption, string) error
func (f *CycleField) Focus() error
```

`SelectField` exposes the corresponding methods.

`[` selects the previous enabled option and `]` selects the next. Cycling
wraps unless `Clamp` is true. Arrows retain their focus-navigation meaning
and never change the field value. Space, Enter, Home, and End likewise do not
change a CycleField/SelectField value. A mnemonic focuses the field without
changing its value. Empty and all-disabled fields are retained and observable
but are not focusable.

The canonical one-row presentation is `Label  ◄ value ►`, with one-cell
ASCII fallback for the arrows at the physical-terminal boundary. A visible
bracketed empty marker is used when no enabled option exists.

## Enabled State, Styles, And Focus

All controls are enabled unless their `Disabled` option is true. A disabled
control or option has a nonempty reason, is visible and typed, cannot receive
focus, and cannot change through user input. Phase 11 construction fixes this
state; a later contract may add dynamic enablement if a demonstrated
application need justifies the API.

The default Theme provides:

- each control-kind base style;
- `selection.mnemonic`;
- `selection.focused`;
- `selection.focused_mnemonic`; and
- `selection.disabled`.

PanelOptions.Style replaces only the base style. The semantic focus marker,
not color alone, distinguishes focus.

The direct parent Container defines a focus group. Tab and Shift-Tab traverse
eligible groups rather than individual controls. Arrows move spatially among
eligible controls within the group; when no in-group target exists, they may
enter another group only when the nearest directional target is unambiguous.
This focus-only movement never changes Checkbox, RadioGroup, CycleField, or
SelectField values. Menus temporarily own focus and restore the prior
eligible control. Visibility, disablement, destruction, and option removal
repair focus deterministically.

## Change Notification And MVC/MVVM Use

`ChangeCommand` is optional. If present, it must name a registered command.
User input and `Activate` first mutate the toolkit selection on the serialized
dispatch path, then invoke the ordinary App command router outside toolkit
locks with the changed control as `Command.Target`. The handler can safely
read the control's typed getter or an App snapshot. It must not receive a
mutable event or unrestricted payload map.

Programmatic setters do not invoke `ChangeCommand`; the caller already owns
that transition. They are thread-safe, publish atomically, and have
Transaction forms for coordinated model-to-view updates. An unchanged
selection produces `OutcomeNoOp` and no change command.

The router result governs the completion outcome but does not roll back a
selection already made by the user. Application validation that can reject a
choice belongs before enablement or in a later explicit editing/commit model,
not in an implicit callback rollback.

## Typed Snapshot And Automation

`ControlDetails` adds exact members for Checkbox, RadioButton, RadioGroup, and
ChoiceField. They expose canonical label/value state, enabled and disabled
facts, mnemonic, change command, three-state/allow-empty/clamp policy, and
copied option records. RadioGroup details include current value and each
direct child's stable value/ID; ChoiceField details include the selected
index and each option's enabled state.

The automation projection copies and validates every field, requires each
member only for its matching control kind, and applies the same per-control
and aggregate resource bounds. Human terminal input, headless raw keys,
attached `KeyDown`/`KeyUp`/`KeyPress`, public `Activate`, and demo semantic
commands converge on the same setters and observable state.

## Acceptance Criteria

- two-state and three-state Checkbox transitions are exact;
- one RadioGroup can never publish two selected RadioButtons;
- radio focus navigation skips disabled options without selecting them;
  Space/Enter selection and post-destruction repair are exact;
- Tab/Shift-Tab cross parent-defined focus groups, while arrows move within
  groups and perform only unambiguous directional crossings;
- `[`/`]` are the only ordinary raw navigation keys that change a
  CycleField/SelectField value;
- empty, singleton, all-disabled, wrapping, and clamped choice fields are
  deterministic;
- focus, selection, and command activation remain distinguishable;
- change commands run outside toolkit locks and are serialized;
- copied options and snapshots cannot alias toolkit state;
- resize and clipping do not change selected values;
- every control is present on purpose-specific `expletives-test` pages and
  reachable through enabled Controls-menu routes; and
- unit, integration, raw automation, race, PTY, and all build-mode checks pass.
