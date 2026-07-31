# Progress API v0

- Status: Directed pre-v1 contract
- Authority: Phase 13 roadmap and operator full-automatic direction,
  2026-07-31
- Scope: `ProgressBar`, `Meter`, `Spinner`, and `ActivityDots`
- Depends on:
  [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md),
  [`application-architecture.md`](application-architecture.md),
  [`go-api-v0.md`](go-api-v0.md), and
  [`automation-protocol-v1.md`](automation-protocol-v1.md)

## Purpose

Progress controls are bounded view objects for consumer-owned work. They do
not start jobs, read wall time, own a ticker, cancel arbitrary application
goroutines, or mutate a consumer model. A controller copies an immutable
state or absolute animation tick into the UI through the ordinary
owner-serialized transaction boundary.

This keeps MVC, MVVC, MVVM, actor, and worker-pool applications free to choose
their own work and cancellation lifecycles while making every rendered frame
deterministic and automation-replayable.

## Shared Status

```go
type ProgressStatus string

const (
    ProgressIdle      ProgressStatus = "idle"
    ProgressRunning   ProgressStatus = "running"
    ProgressCompleted ProgressStatus = "completed"
    ProgressFailed    ProgressStatus = "failed"
    ProgressCancelled ProgressStatus = "cancelled"
)
```

The empty status normalizes to `ProgressIdle`. Unknown values are rejected.
Completed, failed, and cancelled are presentation facts supplied by the
application; they do not imply that the toolkit owns or has joined a worker.

## Determinate ProgressBar

```go
type ProgressTextMode string

const (
    ProgressTextDefault    ProgressTextMode = ""
    ProgressTextNone       ProgressTextMode = "none"
    ProgressTextPercentage ProgressTextMode = "percentage"
)

type ProgressBarState struct {
    Current       uint64
    Total         uint64
    Indeterminate bool
    Tick          uint64
    ReducedMotion bool
    TextMode      ProgressTextMode
    Status        ProgressStatus
}

type ProgressBarOptions struct {
    PanelOptions
    State ProgressBarState
}

func NewProgressBar(Container, ProgressBarOptions) (*ProgressBar, error)
func (t *Transaction) NewProgressBar(
    Container,
    ProgressBarOptions,
) (*ProgressBar, error)
func (p *ProgressBar) State() ProgressBarState
func (p *ProgressBar) SetState(ProgressBarState) error
func (p *ProgressBar) Update(context.Context, ProgressBarState) error
func (t *Transaction) SetProgressBarState(
    *ProgressBar,
    ProgressBarState,
) error
```

A determinate state requires `Current <= Total`. Zero of zero is valid: it
renders zero percent except when completed, when it renders 100 percent. A
completed state otherwise requires `Current == Total`.

An indeterminate state requires zero `Current` and `Total`. While running, its
highlighted cell is selected by the absolute `Tick`; reduced motion freezes
that highlight and canonicalizes the retained tick to zero. Tick is likewise
zero outside running indeterminate state. The zero `TextMode` selects
percentage text for determinate state and no text for indeterminate state.

The fill count uses exact integer ratio arithmetic and floors partial cells.
Percentage text uses exact integer rounding to the nearest whole percent.
Text is centered when it fits. Geometry too narrow for the percentage uses
one stable ASCII status cell rather than corrupting adjacent content.

## Meter

```go
type MeterState struct {
    Value       float64
    Minimum     float64
    Maximum     float64
    Orientation Orientation
    Status      ProgressStatus
}

type MeterOptions struct {
    PanelOptions
    State MeterState
}

func NewMeter(Container, MeterOptions) (*Meter, error)
func (t *Transaction) NewMeter(Container, MeterOptions) (*Meter, error)
func (m *Meter) State() MeterState
func (m *Meter) SetState(MeterState) error
func (m *Meter) Update(context.Context, MeterState) error
func (t *Transaction) SetMeterState(*Meter, MeterState) error
```

Meter represents a current scalar within a finite inclusive range. If the
entire state is zero-valued, the range normalizes to 0 through 100. Otherwise
`Minimum < Maximum` and `Minimum <= Value <= Maximum` are required. Horizontal
is the zero/default orientation; Vertical is also supported.

Meter fills from left to right or bottom to top. It does not render a numeric
label and does not claim application threshold semantics. Applications may
select a semantic control style or adjacent Label for domain-specific units,
warning bands, and descriptions.

## Spinner And ActivityDots

```go
type ActivityState struct {
    Tick          uint64
    ReducedMotion bool
    Status        ProgressStatus
}

type SpinnerOptions struct {
    PanelOptions
    State ActivityState
}

type ActivityDotsOptions struct {
    PanelOptions
    State ActivityState
}

func NewSpinner(Container, SpinnerOptions) (*Spinner, error)
func NewActivityDots(Container, ActivityDotsOptions) (*ActivityDots, error)
func (s *Spinner) State() ActivityState
func (a *ActivityDots) State() ActivityState
func (s *Spinner) SetState(ActivityState) error
func (a *ActivityDots) SetState(ActivityState) error
func (s *Spinner) Update(context.Context, ActivityState) error
func (a *ActivityDots) Update(context.Context, ActivityState) error
func (t *Transaction) SetActivityState(Control, ActivityState) error
```

Spinner uses the deterministic ASCII frames `|`, `/`, `-`, and `\`.
ActivityDots uses a deterministic three-cell dot sequence. Running reduced
motion uses a stable nonanimated presentation and retains tick zero. Idle and
terminal states are also stable. No internal clock exists.

## Worker And Coalescing Boundary

`Update` is the explicit synchronous worker-result boundary:

- it validates and copies the complete state before mutation;
- it observes the caller context while waiting for the bounded App mutation
  gate;
- it returns cancellation, `ErrMutationBusy`, invalid-state, destroyed, or
  shutdown errors explicitly;
- it publishes the state and intended frame atomically; and
- it never invokes consumer code or accesses a terminal backend.

`SetState` is the ordinary one-operation convenience form with
`DefaultMutationWait`; Transactions batch several controls into one
publication. Applications own generation/staleness policy and should reject a
stale result before submitting it. The toolkit does not infer model ordering
from goroutine scheduling.

An update identical to current canonical state publishes nothing. Ticks that
cannot affect presentation because status is not running, motion is reduced,
or a ProgressBar is determinate canonicalize to zero. Thus common duplicate
and reduced-motion tick streams coalesce without a queue, helper goroutine, or
unbounded retained work.

## Rendering And Styles

The controls are non-focusable Panel-derived leaves. Their intrinsic minima
are 8x1 for ProgressBar, 8x1 or 1x3 for Meter, 1x1 for Spinner, and 3x1 for
ActivityDots. Explicit caller minima still use ordinary Panel rules.

Default Theme roles include the four control bases plus shared fill, text,
completed, failed, and cancelled progress roles; each base style supplies its
empty cells. Terminal presentation uses the existing semantic-style and
Limited Unicode projection. No progress control writes directly to a
terminal.

## Typed Evidence

`ControlDetails.Progress` is present for exactly the four progress kinds. It
contains status, kind-consistent numeric state, orientation, indeterminate
state, canonical tick, reduced-motion and text policy, plus the exact current
animation frame index.

Core snapshots deep-copy the typed member. Automation explicitly projects and
validates it, including finite ordered Meter ranges, ProgressBar totals,
canonical frozen ticks, frame indices, orientation, text modes, and
control-kind consistency. No unrestricted property map is introduced.

## Catalog

The Controls/Progress page contains:

- a partially complete percentage ProgressBar;
- an indeterminate ProgressBar;
- horizontal and vertical Meters;
- an animated Spinner;
- animated ActivityDots; and
- stable completed, failed, cancelled, and reduced-motion examples.

Catalog commands provide deterministic next-tick, reset, complete, fail,
cancel, and reduced-motion transitions. The ordinary contextual Status Bar
identifies the selected page without making the progress controls depend on
StatusBar.

## Acceptance

- exact ratio, rounding, zero-total, range, terminal-state, tiny-geometry, and
  orientation tests;
- deterministic animation and reduced-motion frame tests;
- invalid construction/mutation rejection and atomic transaction rollback;
- duplicate/frozen-update coalescing without hidden goroutines;
- concurrent worker updates, cancellation, shutdown, snapshot reads, and race
  coverage;
- typed core/automation projection, validation, deep-copy, aggregate-bound,
  and maximum-response evidence;
- exhaustive catalog menu, headless self-check, raw attached automation, PTY,
  and debug/release/profiling builds; and
- no toolkit-owned progress worker, clock, or goroutine to leak at shutdown.
