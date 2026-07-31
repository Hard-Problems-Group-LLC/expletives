# Text And Numeric Input API v0

- Status: Directed pre-v1 contract
- Authority: Phase 12 operator direction, 2026-07-31
- Scope: `TextField`, `NumberField`, `SpinBox`, and `TextArea`
- Depends on:
  [`Limited-Unicode-Support.md`](../Limited-Unicode-Support.md),
  [`selection-api-v0.md`](selection-api-v0.md),
  [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md), and
  [`automation-protocol-v1.md`](automation-protocol-v1.md)

## Purpose

The input controls provide bounded, character-cell-safe editing without
forcing an application model or validation framework on consuming MVC, MVVC,
or similar applications. Editing state belongs to the view/control. Committed
values and optional `ChangeCommand` notifications provide the controller
boundary.

This contract uses Turbo Vision's explicit editing and cursor idiom while
retaining the project's direct-parent focus groups and raw logical key path.
No source implementation is copied.

## Text Validation

Validation is optional. A configured validator requires all three fields:

```go
type TextValidationEnforcement string

const (
    TextValidationSoft TextValidationEnforcement = "soft"
    TextValidationHard TextValidationEnforcement = "hard"
)

type TextValidationMode string

const (
    TextValidationWhitelist TextValidationMode = "whitelist"
    TextValidationBlacklist TextValidationMode = "blacklist"
)

type TextValidator struct {
    Enforcement TextValidationEnforcement
    Mode        TextValidationMode
    Characters  string
}
```

`Characters` is a nonempty set of canonical one-cell input elements. An
element may be one code point or one composed grapheme cluster that occupies
one cell. Duplicate elements have no additional meaning. Control characters
and line separators are invalid validator entries. An unsupported display
element normalizes to the one-cell `U+FFFD` replacement before matching; an
application must list `U+FFFD` explicitly if it intends to allow that
replacement in a whitelist.

A whitelist accepts only matching elements. A blacklist accepts every element
except matching elements. Omitting the validator is distinct from supplying
an incomplete validator; incomplete or unknown enforcement/mode values are
rejected atomically.

### Soft Enforcement

Soft enforcement accepts every otherwise valid and in-bounds input element.
If the complete current value passes validation, every displayed character is
green. If any element fails:

- every valid displayed character is yellow;
- every invalid displayed character is red; and
- the control's typed state reports invalid.

Color is a required default Theme mapping, not the only state cue. Typed
details expose validity, and password masking described below remains active.
An invalid soft value may be edited and committed; application policy may use
the typed validity fact or the change notification to gate later actions.

### Hard Enforcement

Hard enforcement ignores disallowed interactive input. Rejected typed or
pasted elements never enter the control value, never paint, and never move the
caret. Accepted elements in the field are green.

Constructors and programmatic setters do not silently repair caller data. A
hard-validated programmatic value containing a disallowed element is rejected
atomically. This preserves an honest MVC update boundary while keeping human
typing frictionless.

Length, resource, single-line, and Limited Unicode constraints are independent
of validation. They remain enforced when no validator exists and in both
enforcement modes.

## Password Presentation

Every text-entry control has:

```go
Password bool
```

The default is false. When true, one `*` is painted for each retained input
element instead of the element itself. Validation still examines the real
value. In soft mode, valid masked cells follow the green/yellow whole-value
rule and invalid masked cells are red. Hard-invalid input remains absent.

Password masking is a presentation and observation boundary:

- `Text()` and other direct in-process value accessors return the real value
  to the owning application;
- intended-frame cells contain only `*`, never the real value;
- core and automation typed snapshots omit the real value and expose only
  bounded length, caret, editing, validation, and masking facts;
- completion diagnostics, guidance, logs, and errors never include the real
  value; and
- retained snapshot history cannot recover a prior password.

This is shoulder-surfing and automation-evidence protection, not encrypted
application storage. Consuming applications remain responsible for secret
lifetime, clearing, and domain storage.

## TextField

```go
type TextFieldOptions struct {
    PanelOptions
    Text           string
    Validator      *TextValidator
    Password       bool
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

type TextField struct { /* copy-safe Panel-derived leaf */ }

func NewTextField(Container, TextFieldOptions) (*TextField, error)
func (t *Transaction) NewTextField(
    Container,
    TextFieldOptions,
) (*TextField, error)
func (f *TextField) Text() string
func (f *TextField) SetText(string) error
func (f *TextField) Validator() *TextValidator
func (f *TextField) SetValidator(*TextValidator) error
func (f *TextField) Password() bool
func (f *TextField) SetPassword(bool) error
func (f *TextField) Editing() bool
func (f *TextField) Valid() bool
func (f *TextField) Focus() error
func (f *TextField) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)
```

A focused field starts outside edit mode so bare letters remain available to
screen bindings. Enter activates editing. While editing:

- printable one-cell input inserts at the caret;
- Left/Right move by one complete displayed element;
- Home/End move to the first/last element;
- holding Shift while using those movement keys extends the selection from
  its fixed anchor, while movement without Shift clears the selection;
- Ctrl-A selects the complete working value;
- printable input and committed text replace a nonempty selection;
- Backspace/Delete remove a nonempty selection, or one complete element when
  no selection exists;
- Enter commits and leaves edit mode;
- Escape restores the pre-edit committed value and leaves edit mode; and
- Tab or focus movement commits and leaves edit mode before group traversal.

Selection endpoints are zero-based canonical-element offsets with an
end-exclusive upper endpoint. They never split a composed one-cell element.
`Ctrl-C` is not claimed as an editor-copy chord: it remains available to the
application's configured interrupt policy in every editor state. A later
clipboard proposal may add copy/cut bindings without weakening that recovery
path.

The logical cursor is visible only for an effectively visible, focused field
in edit mode. Horizontal view offset keeps the caret visible. Focus,
selection, activation, committed value, working value, and validation remain
distinct facts.

User commits publish the value first and then invoke the optional registered
`ChangeCommand` outside toolkit locks. Programmatic setters are silent.

### Form Presentation And Sizing

An ordinary `TextField`, `NumberField`, or `SpinBox` is a one-row field. It
does not draw an implicit control-owned frame; the distinct field background
across its complete arranged width is the primary visual affordance.
Applications normally choose a minimum width between 10 and 30 cells and put
a separate bound `Label` in an adjacent form column. The Label mnemonic
focuses its target field.

These single-line editors default to horizontal stretch and vertical natural
sizing. Their one-row minimum does not grow merely because a vertical Box has
surplus rows. An explicit construction-time `PanelOptions.LayoutHints`
override may change that policy. Composing a field inside a one-cell bordered
Frame yields a three-row compound minimum: border, field, border.

Form Layouts control inter-row whitespace with their ordinary Gap setting.
They must not invent vertical stretching or borders merely to fill a page.
When the complete natural form exceeds its viewport, the containing
ScrollablePanel supplies scrolling rather than shrinking or overlapping
field rows.

## NumberField And SpinBox

```go
const MaxNumberDecimalPlaces = 9

type NumberFieldOptions struct {
    PanelOptions
    Value          float64
    Minimum        *float64
    Maximum        *float64
    DecimalPlaces  int
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

type SpinBoxOptions struct {
    PanelOptions
    Value          float64
    Minimum        *float64
    Maximum        *float64
    DecimalPlaces  int
    Step           float64
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

func NewNumberField(Container, NumberFieldOptions) (*NumberField, error)
func NewSpinBox(Container, SpinBoxOptions) (*SpinBox, error)
func (f *NumberField) Value() float64
func (f *NumberField) SetValue(float64) error
func (s *SpinBox) Value() float64
func (s *SpinBox) SetValue(float64) error
```

Both controls use finite `float64` application values with fixed decimal
presentation from zero through nine places. They are not arbitrary-precision
financial decimal types. Constructors, bounds, steps, and programmatic values
must already fit the configured precision; the toolkit rejects them rather
than silently rounding caller-owned model data. Optional copied minimum and
maximum values are inclusive and must be ordered.

Editing admits digits, one leading minus when the complete value is parsed,
and a decimal point only when `DecimalPlaces` is nonzero. Intermediate text
such as empty, `-`, or an incomplete/out-of-range value may exist while
editing and is painted in the invalid style with a typed reason. Enter and
Tab refuse an invalid commit and keep focus/edit mode. Escape restores the
committed value. A forced programmatic or screen focus loss commits a valid
edit and cancels an invalid edit. A successful commit publishes a canonical
fixed-place string and the new numeric value before routing `ChangeCommand`.

`SpinBox` adds a positive finite step that fits the configured precision. A
zero option selects step `1`. Outside edit mode, `[` decrements and `]`
increments, clamping to an optional bound; a step already at the bound is a
no-op. Inside edit mode those characters are rejected by the numeric
character policy, so arrows retain caret meaning and bracket stepping cannot
silently replace a working edit. Arithmetic overflow without an applicable
bound is a no-op.

`ControlDetails.NumberField` is present for both kinds. It contains current
text, committed numeric value, length, caret/view position, edit and validity
state, invalid reason, copied optional bounds, decimal places, optional step,
disabled reason, and change command. `Step` is zero for NumberField and
positive for SpinBox. Core and wire snapshots deep-copy bound pointers and
the client revalidates syntax, precision, range, kind consistency, and finite
values.

## TextArea

```go
type TextAreaOptions struct {
    PanelOptions
    Text           string
    Validator      *TextValidator
    Password       bool
    Wrap           TextWrap
    Disabled       bool
    DisabledReason string
    ChangeCommand  CommandID
}

func NewTextArea(Container, TextAreaOptions) (*TextArea, error)
func (t *Transaction) NewTextArea(
    Container,
    TextAreaOptions,
) (*TextArea, error)
func (a *TextArea) Text() string
func (a *TextArea) SetText(string) error
func (a *TextArea) Validator() *TextValidator
func (a *TextArea) SetValidator(*TextValidator) error
func (a *TextArea) Password() bool
func (a *TextArea) SetPassword(bool) error
func (a *TextArea) Wrap() TextWrap
func (a *TextArea) SetWrap(TextWrap) error
func (a *TextArea) Editing() bool
func (a *TextArea) Valid() bool
func (a *TextArea) Focus() error
func (a *TextArea) Activate(
    context.Context,
    source string,
    requestID string,
) (Completion, error)
```

`TextArea` extends the validator/password rules to explicit line separators.
CRLF and CR input normalize to LF. Other control characters are rejected.
Validator `Characters` describes input elements, not newline; newline is
admitted structurally and is never colored as a validator failure.

The zero `Wrap` value selects `TextWrapNone`. `TextWrapNone`,
`TextWrapCells`, and `TextWrapWords` have the same meanings as StaticText.
Unwrapped areas maintain both row and column viewport offsets. Wrapped areas
keep column offset zero and maintain a visual-row offset. Resizing clamps the
effective offsets and keeps the editing caret visible. The private viewport
is not a public scrollbar model; Phase 15 may replace its implementation
without changing this editor contract.

Outside edit mode Enter activates editing. Inside edit mode:

- Enter inserts one LF at the caret;
- Ctrl-Enter commits and leaves edit mode;
- Tab commits and moves between focus groups;
- Escape restores the pre-edit value and leaves edit mode;
- Left/Right move by one canonical element, including across LF;
- Up/Down preserve the preferred visual column where possible;
- Home/End move to the beginning/end of the current visual row;
- Ctrl-Home/Ctrl-End move to the beginning/end of the complete value;
- PageUp/PageDown move by one visible page; and
- Shift extends selection for every movement operation.

The same selection replacement, deletion, Ctrl-A, validation, password
redaction, focus-loss commit, and user-only `ChangeCommand` rules as
`TextField` apply. Selection endpoints and caret offsets count canonical
elements including each LF as one structural element. `TextArea` has an
intrinsic minimum of 8 by 3 cells.

`ControlDetails.TextArea` contains current text (or an empty redacted value),
element length, logical-line count, caret and selection offsets, visual
caret row/column, row/column viewport offsets, wrap, edit/valid/password
state, disabled reason, change command, and copied validator summary.

Unlike a single-line editor, `TextArea` defaults to stretch on both axes. In a
BoxLayout with no explicit positive item `Grow`, all remaining main-axis
space is divided among stretch-capable controls according to the applicable
construction-time horizontal or vertical weights. Natural controls retain
their minima and Layout Gap remains independent. Applications may override a
TextArea axis to natural or assign a different positive weight through
`PanelOptions.LayoutHints`.

## Committed Text And Paste Events

```go
type TextInputKind string

const (
    TextInputCommitted TextInputKind = "committed_text"
    TextInputPaste     TextInputKind = "paste"
)

type TextInputEvent struct {
    Kind TextInputKind
    Text string
}

func (a *App) DispatchTextInput(
    context.Context,
    source string,
    requestID string,
    event TextInputEvent,
) (Completion, error)
```

Committed text represents an IME or another trusted text-composition
boundary. Paste represents bracketed-paste content. Both are bounded to
`MaxTextInputBytes`, enter only the focused enabled editor while it is in edit
mode, replace its current selection, and produce an exact associated
completion. They never enter mnemonic, accelerator, hotkey, menu, or command
resolution.

Input is normalized to canonical one-cell elements before insertion.
TextField, NumberField, and SpinBox reject an event containing line
separators. TextArea normalizes line separators as described above. A hard
validator filters rejected non-newline elements without residue; if nothing
remains, the event is a handled no-op. Candidate values exceeding the
per-control byte or element bound are rejected atomically rather than
truncated. Numeric controls accept a text event only when every retained
element satisfies their numeric character policy; parse/range validation
still occurs at commit.

The terminal adapter emits bracketed paste through this semantic path and
retains at most `MaxTextInputBytes` of one paste. If the physical paste
exceeds the bound, it discards the complete episode and reports no partial
text. Escape/control sequences inside a bounded paste remain inert text and
are rejected or normalized by the target editor; they are never replayed
through the terminal decoder.

## Typed Evidence And Security

`ControlDetails.TextField`, `ControlDetails.NumberField`, and
`ControlDetails.TextArea` are versioned typed members. TextField and TextArea
details expose canonical committed or current edit text, length, caret,
selection, viewport, editing, validity, password, disabled reason, change
command, and the applicable copied policy summaries. Number details expose
the corresponding edit state, exact numeric policy, and committed value. A
password member always has empty text and an asserted redacted flag.

No unrestricted details map is introduced. Core snapshots deep-copy every
pointer-bearing detail. Automation projection explicitly copies and validates
the applicable member, enforces control-kind consistency, canonical text,
selection/viewport/range invariants, finite numeric values, and aggregate
bounds, and rejects any password detail containing text.

## Acceptance

Verification includes:

- missing, invalid, whitelist, and blacklist validators;
- every soft valid/invalid color transition, including multiple invalid
  characters;
- hard input rejection without value, frame, or caret residue;
- constructor and setter atomic rejection in hard mode;
- password masking in frames and password redaction in every retained core and
  automation snapshot;
- one-cell composed input, `U+FFFD` replacement, deletion, caret motion, and
  horizontal clipping;
- Shift selection, selection replacement/deletion, Ctrl-A, and Ctrl-C
  interrupt availability;
- multiline CR/LF normalization, visual-row navigation, wrapping, viewport
  clamping, and resize;
- compact label/field form geometry, one-row single-line controls, two-axis
  TextArea stretching, and weighted division between multiple TextAreas;
- bounded committed-text and bracketed-paste delivery with no command
  interpretation or partial over-limit insertion;
- raw human/headless/automation key equivalence;
- edit activation, commit, cancel, focus loss, and disabled behavior;
- ordinary Go, race, headless automation, attached automation, and debug,
  release, and profiling builds; and
- bounded-response proof after the new typed details are added.
