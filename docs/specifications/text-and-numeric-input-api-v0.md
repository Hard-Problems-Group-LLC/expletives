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
- Backspace/Delete remove one complete element;
- Enter commits and leaves edit mode;
- Escape restores the pre-edit committed value and leaves edit mode; and
- Tab or focus movement commits and leaves edit mode before group traversal.

The logical cursor is visible only for an effectively visible, focused field
in edit mode. Horizontal view offset keeps the caret visible. Focus,
selection, activation, committed value, working value, and validation remain
distinct facts.

User commits publish the value first and then invoke the optional registered
`ChangeCommand` outside toolkit locks. Programmatic setters are silent.

## NumberField And SpinBox

`NumberField` is a decimal numeric specialization of the single-line editor.
Its invariant numeric validator composes with range and precision rules rather
than weakening the text validator contract. Empty/intermediate edit strings
may exist while editing; commit either produces one bounded numeric value or
retains editing with an invalid fact according to the configured policy.

`SpinBox` adds step, minimum, maximum, and `[`/`]` decrement/increment behavior
to `NumberField`. Arrow keys retain focus-navigation meaning outside edit mode
and caret-navigation meaning inside edit mode.

Exact numeric types, overflow behavior, commit policy, and typed evidence are
specified before those controls are exposed.

## TextArea

`TextArea` extends the same editor and validator/password rules to explicit
line separators. Validator `Characters` describes input elements, not newline;
the multiline control admits newline structurally through Enter. It uses a
private bounded viewport until Phase 15 publishes the reusable scrolling
controls.

Exact multiline selection, wrapping, offset, and commit behavior are specified
before `TextArea` is exposed.

## Typed Evidence And Security

`ControlDetails.TextField` is a versioned typed member. For ordinary fields it
contains the canonical committed or current edit value, value length, caret,
view offset, editing, valid, password, disabled reason, change command, and a
copied validator summary. For password fields the value member is always
empty and a redacted flag is true.

No unrestricted details map is introduced. Automation projection explicitly
copies and validates the member, enforces kind consistency and aggregate
bounds, and rejects a password detail containing a value.

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
- raw human/headless/automation key equivalence;
- edit activation, commit, cancel, focus loss, and disabled behavior;
- ordinary Go, race, headless automation, attached automation, and debug,
  release, and profiling builds; and
- bounded-response proof after the new typed details are added.
