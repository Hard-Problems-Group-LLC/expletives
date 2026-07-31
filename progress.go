package expletives

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strconv"
)

// ProgressStatus identifies application-supplied work state.
type ProgressStatus string

const (
	ProgressIdle      ProgressStatus = "idle"
	ProgressRunning   ProgressStatus = "running"
	ProgressCompleted ProgressStatus = "completed"
	ProgressFailed    ProgressStatus = "failed"
	ProgressCancelled ProgressStatus = "cancelled"
)

// ProgressTextMode selects ProgressBar textual presentation.
type ProgressTextMode string

const (
	ProgressTextDefault    ProgressTextMode = ""
	ProgressTextNone       ProgressTextMode = "none"
	ProgressTextPercentage ProgressTextMode = "percentage"
)

// ProgressBarState is one complete copied ProgressBar update.
type ProgressBarState struct {
	Current       uint64
	Total         uint64
	Indeterminate bool
	Tick          uint64
	ReducedMotion bool
	TextMode      ProgressTextMode
	Status        ProgressStatus
}

// MeterState is one complete copied Meter update.
type MeterState struct {
	Value       float64
	Minimum     float64
	Maximum     float64
	Orientation Orientation
	Status      ProgressStatus
}

// ActivityState is one complete copied Spinner or ActivityDots update.
type ActivityState struct {
	Tick          uint64
	ReducedMotion bool
	Status        ProgressStatus
}

// ProgressBarOptions configures one determinate or indeterminate bar.
type ProgressBarOptions struct {
	PanelOptions
	State ProgressBarState
}

// MeterOptions configures one horizontal or vertical scalar meter.
type MeterOptions struct {
	PanelOptions
	State MeterState
}

// SpinnerOptions configures one single-cell activity spinner.
type SpinnerOptions struct {
	PanelOptions
	State ActivityState
}

// ActivityDotsOptions configures one three-cell activity indicator.
type ActivityDotsOptions struct {
	PanelOptions
	State ActivityState
}

// ProgressBar is a copy-safe non-focusable progress control.
type ProgressBar struct{ controlHandle }

// Meter is a copy-safe non-focusable scalar meter.
type Meter struct{ controlHandle }

// Spinner is a copy-safe non-focusable deterministic activity indicator.
type Spinner struct{ controlHandle }

// ActivityDots is a copy-safe non-focusable deterministic dot indicator.
type ActivityDots struct{ controlHandle }

type progressBehavior struct {
	progressBar  *ProgressBarState
	meter        *MeterState
	activity     *ActivityState
	activityDots bool
}

// NewProgressBar constructs and atomically inserts a ProgressBar.
func NewProgressBar(
	parent Container,
	options ProgressBarOptions,
) (*ProgressBar, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewProgressBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewMeter constructs and atomically inserts a Meter.
func NewMeter(parent Container, options MeterOptions) (*Meter, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewMeter(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewSpinner constructs and atomically inserts a Spinner.
func NewSpinner(parent Container, options SpinnerOptions) (*Spinner, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewSpinner(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewActivityDots constructs and atomically inserts ActivityDots.
func NewActivityDots(
	parent Container,
	options ActivityDotsOptions,
) (*ActivityDots, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewActivityDots(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewProgressBar records construction of a provisional ProgressBar.
func (t *Transaction) NewProgressBar(
	parent Container,
	options ProgressBarOptions,
) (*ProgressBar, error) {
	progress, err := normalizeProgressBarState(options.State)
	if err != nil {
		return nil, err
	}
	behavior := progressBehavior{progressBar: &progress}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlProgressBar,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &ProgressBar{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

// NewMeter records construction of a provisional Meter.
func (t *Transaction) NewMeter(
	parent Container,
	options MeterOptions,
) (*Meter, error) {
	meter, err := normalizeMeterState(options.State)
	if err != nil {
		return nil, err
	}
	behavior := progressBehavior{meter: &meter}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlMeter,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &Meter{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

// NewSpinner records construction of a provisional Spinner.
func (t *Transaction) NewSpinner(
	parent Container,
	options SpinnerOptions,
) (*Spinner, error) {
	activity, err := normalizeActivityState(options.State)
	if err != nil {
		return nil, err
	}
	behavior := progressBehavior{activity: &activity}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlSpinner,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &Spinner{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

// NewActivityDots records construction of provisional ActivityDots.
func (t *Transaction) NewActivityDots(
	parent Container,
	options ActivityDotsOptions,
) (*ActivityDots, error) {
	activity, err := normalizeActivityState(options.State)
	if err != nil {
		return nil, err
	}
	behavior := progressBehavior{
		activity:     &activity,
		activityDots: true,
	}
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlActivityDots,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &ActivityDots{controlHandle: controlHandle{state: state}}
	state.control = control
	return control, nil
}

func normalizeProgressStatus(status ProgressStatus) (ProgressStatus, error) {
	if status == "" {
		return ProgressIdle, nil
	}
	switch status {
	case ProgressIdle, ProgressRunning, ProgressCompleted, ProgressFailed,
		ProgressCancelled:
		return status, nil
	default:
		return "", fmt.Errorf("%w: invalid progress status", ErrValidation)
	}
}

func normalizeProgressTextMode(
	mode ProgressTextMode,
	indeterminate bool,
) (ProgressTextMode, error) {
	if mode == ProgressTextDefault {
		if indeterminate {
			return ProgressTextNone, nil
		}
		return ProgressTextPercentage, nil
	}
	switch mode {
	case ProgressTextNone, ProgressTextPercentage:
		if indeterminate && mode == ProgressTextPercentage {
			return "", fmt.Errorf(
				"%w: indeterminate ProgressBar cannot show a percentage",
				ErrValidation,
			)
		}
		return mode, nil
	default:
		return "", fmt.Errorf(
			"%w: invalid ProgressBar text mode",
			ErrValidation,
		)
	}
}

func normalizeProgressBarState(
	state ProgressBarState,
) (ProgressBarState, error) {
	status, err := normalizeProgressStatus(state.Status)
	if err != nil {
		return ProgressBarState{}, err
	}
	textMode, err := normalizeProgressTextMode(
		state.TextMode,
		state.Indeterminate,
	)
	if err != nil {
		return ProgressBarState{}, err
	}
	state.Status = status
	state.TextMode = textMode
	if state.Indeterminate {
		if state.Current != 0 || state.Total != 0 {
			return ProgressBarState{}, fmt.Errorf(
				"%w: indeterminate ProgressBar requires zero current and total",
				ErrValidation,
			)
		}
	} else {
		if state.Current > state.Total {
			return ProgressBarState{}, fmt.Errorf(
				"%w: ProgressBar current exceeds total",
				ErrValidation,
			)
		}
		if status == ProgressCompleted && state.Current != state.Total {
			return ProgressBarState{}, fmt.Errorf(
				"%w: completed ProgressBar has remaining work",
				ErrValidation,
			)
		}
	}
	if !state.Indeterminate || state.ReducedMotion ||
		state.Status != ProgressRunning {
		state.Tick = 0
	}
	return state, nil
}

func normalizeMeterState(state MeterState) (MeterState, error) {
	status, err := normalizeProgressStatus(state.Status)
	if err != nil {
		return MeterState{}, err
	}
	state.Status = status
	if state.Value == 0 && state.Minimum == 0 && state.Maximum == 0 {
		state.Maximum = 100
	}
	if math.IsNaN(state.Value) || math.IsInf(state.Value, 0) ||
		math.IsNaN(state.Minimum) || math.IsInf(state.Minimum, 0) ||
		math.IsNaN(state.Maximum) || math.IsInf(state.Maximum, 0) ||
		state.Minimum >= state.Maximum ||
		state.Value < state.Minimum || state.Value > state.Maximum {
		return MeterState{}, fmt.Errorf(
			"%w: invalid Meter range or value",
			ErrValidation,
		)
	}
	if state.Orientation != Horizontal && state.Orientation != Vertical {
		return MeterState{}, fmt.Errorf(
			"%w: invalid Meter orientation",
			ErrValidation,
		)
	}
	return state, nil
}

func normalizeActivityState(state ActivityState) (ActivityState, error) {
	status, err := normalizeProgressStatus(state.Status)
	if err != nil {
		return ActivityState{}, err
	}
	state.Status = status
	if state.ReducedMotion || status != ProgressRunning {
		state.Tick = 0
	}
	return state, nil
}

func (progressBehavior) clientInset() int { return 0 }

func (b progressBehavior) intrinsicMinimum() Size {
	switch {
	case b.progressBar != nil:
		return Size{Width: 8, Height: 1}
	case b.meter != nil:
		if b.meter.Orientation == Vertical {
			return Size{Width: 1, Height: 3}
		}
		return Size{Width: 8, Height: 1}
	case b.activity != nil:
		if b.activityDots {
			return Size{Width: 3, Height: 1}
		}
		return Size{Width: 1, Height: 1}
	default:
		return Size{}
	}
}

func (b progressBehavior) additionalStyles() []StyleID {
	styles := []StyleID{
		"progress.fill",
		"progress.completed",
		"progress.failed",
		"progress.cancelled",
	}
	if b.progressBar != nil {
		styles = append(styles, "progress.text")
	}
	return styles
}

func (progressBehavior) details() ControlDetails {
	return ControlDetails{
		Version:  ControlDetailsVersion,
		Progress: &ProgressDetails{},
	}
}

func (b progressBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	switch {
	case b.progressBar != nil:
		paintProgressBar(app, frame, state, absolute, clip, *b.progressBar)
	case b.meter != nil:
		paintMeter(app, frame, state, absolute, clip, *b.meter)
	case state.kind == ControlSpinner && b.activity != nil:
		paintActivity(
			app,
			frame,
			state,
			absolute,
			clip,
			spinnerCells(*b.activity),
		)
	case state.kind == ControlActivityDots && b.activity != nil:
		paintActivity(
			app,
			frame,
			state,
			absolute,
			clip,
			activityDotCells(*b.activity),
		)
	}
}

func progressFillStyle(status ProgressStatus) StyleID {
	switch status {
	case ProgressCompleted:
		return "progress.completed"
	case ProgressFailed:
		return "progress.failed"
	case ProgressCancelled:
		return "progress.cancelled"
	default:
		return "progress.fill"
	}
}

func paintProgressBar(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
	state ProgressBarState,
) {
	if absolute.Empty() {
		return
	}
	fill := 0
	if state.Indeterminate {
		if state.Status == ProgressRunning {
			position := absolute.Width / 2
			if !state.ReducedMotion {
				position = int(state.Tick % uint64(absolute.Width))
			}
			style := progressFillStyle(state.Status)
			for row := 0; row < absolute.Height; row++ {
				setProgressCell(
					app,
					frame,
					control,
					clip,
					absolute.X+position,
					absolute.Y+row,
					" ",
					style,
				)
			}
			return
		}
		if state.Status != ProgressIdle {
			label := progressNarrowLabel(state.Status)
			setProgressCell(
				app,
				frame,
				control,
				clip,
				absolute.X+absolute.Width/2,
				absolute.Y+absolute.Height/2,
				label,
				progressFillStyle(state.Status),
			)
		}
		return
	}
	if state.Status == ProgressCompleted && state.Total == 0 {
		fill = absolute.Width
	} else if state.Total != 0 {
		fill = int(scaledProgress(
			state.Current,
			state.Total,
			uint64(absolute.Width),
		))
	}
	style := progressFillStyle(state.Status)
	for row := 0; row < absolute.Height; row++ {
		for column := 0; column < fill; column++ {
			setProgressCell(
				app,
				frame,
				control,
				clip,
				absolute.X+column,
				absolute.Y+row,
				" ",
				style,
			)
		}
	}
	if state.TextMode != ProgressTextPercentage {
		return
	}
	label := strconv.FormatUint(progressPercentage(state), 10) + "%"
	if len(label) > absolute.Width {
		label = progressNarrowLabel(state.Status)
	}
	startX := absolute.X + (absolute.Width-len(label))/2
	y := absolute.Y + absolute.Height/2
	for index, cell := range label {
		setProgressCell(
			app,
			frame,
			control,
			clip,
			startX+index,
			y,
			string(cell),
			"progress.text",
		)
	}
}

func paintMeter(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
	state MeterState,
) {
	if absolute.Empty() {
		return
	}
	ratio := (state.Value - state.Minimum) / (state.Maximum - state.Minimum)
	style := progressFillStyle(state.Status)
	if state.Orientation == Vertical {
		fill := int(math.Floor(ratio * float64(absolute.Height)))
		if state.Value == state.Maximum {
			fill = absolute.Height
		}
		for row := absolute.Height - fill; row < absolute.Height; row++ {
			for column := 0; column < absolute.Width; column++ {
				setProgressCell(
					app,
					frame,
					control,
					clip,
					absolute.X+column,
					absolute.Y+row,
					" ",
					style,
				)
			}
		}
		return
	}
	fill := int(math.Floor(ratio * float64(absolute.Width)))
	if state.Value == state.Maximum {
		fill = absolute.Width
	}
	for row := 0; row < absolute.Height; row++ {
		for column := 0; column < fill; column++ {
			setProgressCell(
				app,
				frame,
				control,
				clip,
				absolute.X+column,
				absolute.Y+row,
				" ",
				style,
			)
		}
	}
}

func paintActivity(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	absolute Rect,
	clip Rect,
	cells string,
) {
	if absolute.Empty() || cells == "" {
		return
	}
	startX := absolute.X + max(0, (absolute.Width-len(cells))/2)
	y := absolute.Y + absolute.Height/2
	for index, cell := range cells {
		if index >= absolute.Width {
			break
		}
		setProgressCell(
			app,
			frame,
			control,
			clip,
			startX+index,
			y,
			string(cell),
			progressFillStyle(activityStatusFromBehavior(control.behavior)),
		)
	}
}

func activityStatusFromBehavior(behavior controlBehavior) ProgressStatus {
	progress, ok := behavior.(progressBehavior)
	if !ok || progress.activity == nil {
		return ProgressIdle
	}
	return progress.activity.Status
}

func setProgressCell(
	app *App,
	frame *IntendedFrame,
	control *controlState,
	clip Rect,
	x int,
	y int,
	grapheme string,
	style StyleID,
) {
	app.setClippedCellLocked(
		frame,
		clip,
		x,
		y,
		grapheme,
		style,
		app.styles[style],
		control.id,
	)
}

func spinnerCells(state ActivityState) string {
	switch state.Status {
	case ProgressRunning:
		if state.ReducedMotion {
			return "*"
		}
		return []string{"|", "/", "-", "\\"}[state.Tick%4]
	case ProgressCompleted:
		return "+"
	case ProgressFailed:
		return "!"
	case ProgressCancelled:
		return "x"
	default:
		return ""
	}
}

func activityDotCells(state ActivityState) string {
	switch state.Status {
	case ProgressRunning:
		if state.ReducedMotion {
			return "..."
		}
		return []string{".  ", ".. ", "...", " ..", "  ."}[state.Tick%5]
	case ProgressCompleted:
		return "+++"
	case ProgressFailed:
		return "!!!"
	case ProgressCancelled:
		return "xxx"
	default:
		return ""
	}
}

func scaledProgress(current, total, scale uint64) uint64 {
	if total == 0 || scale == 0 {
		return 0
	}
	if current >= total {
		return scale
	}
	high, low := bits.Mul64(current, scale)
	quotient, _ := bits.Div64(high, low, total)
	return quotient
}

func progressPercentage(state ProgressBarState) uint64 {
	if state.Status == ProgressCompleted && state.Total == 0 {
		return 100
	}
	if state.Total == 0 {
		return 0
	}
	if state.Current >= state.Total {
		return 100
	}
	high, low := bits.Mul64(state.Current, 100)
	quotient, remainder := bits.Div64(high, low, state.Total)
	if remainder >= state.Total-remainder {
		quotient++
	}
	return min(quotient, uint64(100))
}

func progressNarrowLabel(status ProgressStatus) string {
	switch status {
	case ProgressCompleted:
		return "+"
	case ProgressFailed:
		return "!"
	case ProgressCancelled:
		return "x"
	case ProgressRunning:
		return ">"
	default:
		return "-"
	}
}

func progressFrameIndex(
	kind ControlKind,
	bounds Rect,
	behavior progressBehavior,
) int {
	switch kind {
	case ControlProgressBar:
		if behavior.progressBar == nil ||
			!behavior.progressBar.Indeterminate ||
			behavior.progressBar.Status != ProgressRunning ||
			behavior.progressBar.ReducedMotion ||
			bounds.Width <= 0 {
			return 0
		}
		return int(behavior.progressBar.Tick % uint64(bounds.Width))
	case ControlSpinner:
		if behavior.activity != nil &&
			behavior.activity.Status == ProgressRunning &&
			!behavior.activity.ReducedMotion {
			return int(behavior.activity.Tick % 4)
		}
	case ControlActivityDots:
		if behavior.activity != nil &&
			behavior.activity.Status == ProgressRunning &&
			!behavior.activity.ReducedMotion {
			return int(behavior.activity.Tick % 5)
		}
	}
	return 0
}

func progressDetails(
	kind ControlKind,
	bounds Rect,
	behavior progressBehavior,
) ProgressDetails {
	details := ProgressDetails{
		FrameIndex: progressFrameIndex(kind, bounds, behavior),
	}
	switch {
	case behavior.progressBar != nil:
		state := *behavior.progressBar
		details.Status = state.Status
		details.Current = state.Current
		details.Total = state.Total
		details.Indeterminate = state.Indeterminate
		details.Tick = state.Tick
		details.ReducedMotion = state.ReducedMotion
		details.TextMode = state.TextMode
		details.Orientation = Horizontal
	case behavior.meter != nil:
		state := *behavior.meter
		details.Status = state.Status
		details.Value = state.Value
		details.Minimum = state.Minimum
		details.Maximum = state.Maximum
		details.Orientation = state.Orientation
	case behavior.activity != nil:
		state := *behavior.activity
		details.Status = state.Status
		details.Tick = state.Tick
		details.ReducedMotion = state.ReducedMotion
		details.Indeterminate = true
		details.Orientation = Horizontal
		details.TextMode = ProgressTextNone
	}
	return details
}

// State returns the current canonical ProgressBar state.
func (p *ProgressBar) State() ProgressBarState {
	behavior, ok := progressBehaviorForRead(p.controlState())
	if !ok || behavior.progressBar == nil {
		return ProgressBarState{}
	}
	return *behavior.progressBar
}

// State returns the current canonical Meter state.
func (m *Meter) State() MeterState {
	behavior, ok := progressBehaviorForRead(m.controlState())
	if !ok || behavior.meter == nil {
		return MeterState{}
	}
	return *behavior.meter
}

// State returns the current canonical Spinner state.
func (s *Spinner) State() ActivityState {
	behavior, ok := progressBehaviorForRead(s.controlState())
	if !ok || behavior.activity == nil {
		return ActivityState{}
	}
	return *behavior.activity
}

// State returns the current canonical ActivityDots state.
func (a *ActivityDots) State() ActivityState {
	behavior, ok := progressBehaviorForRead(a.controlState())
	if !ok || behavior.activity == nil {
		return ActivityState{}
	}
	return *behavior.activity
}

func progressBehaviorForRead(
	state *controlState,
) (progressBehavior, bool) {
	if state == nil || state.app == nil {
		return progressBehavior{}, false
	}
	state.app.mu.RLock()
	defer state.app.mu.RUnlock()
	behavior, ok := state.behavior.(progressBehavior)
	return behavior, ok && !state.aborted && !state.destroyed
}

// SetState atomically applies one ProgressBar state.
func (p *ProgressBar) SetState(state ProgressBarState) error {
	return p.Update(context.Background(), state)
}

// Update applies one copied ProgressBar state through the bounded mutation
// owner while observing ctx.
func (p *ProgressBar) Update(
	ctx context.Context,
	state ProgressBarState,
) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	control := p.controlState()
	if control == nil || control.app == nil {
		return ErrInvalidControl
	}
	transaction := control.app.NewTransaction()
	if err := transaction.SetProgressBarState(p, state); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// SetState atomically applies one Meter state.
func (m *Meter) SetState(state MeterState) error {
	return m.Update(context.Background(), state)
}

// Update applies one copied Meter state through the bounded mutation owner.
func (m *Meter) Update(ctx context.Context, state MeterState) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	control := m.controlState()
	if control == nil || control.app == nil {
		return ErrInvalidControl
	}
	transaction := control.app.NewTransaction()
	if err := transaction.SetMeterState(m, state); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// SetState atomically applies one Spinner state.
func (s *Spinner) SetState(state ActivityState) error {
	return s.Update(context.Background(), state)
}

// Update applies one copied Spinner state through the bounded mutation owner.
func (s *Spinner) Update(ctx context.Context, state ActivityState) error {
	return updateActivity(ctx, s, state)
}

// SetState atomically applies one ActivityDots state.
func (a *ActivityDots) SetState(state ActivityState) error {
	return a.Update(context.Background(), state)
}

// Update applies one copied ActivityDots state through the bounded mutation
// owner.
func (a *ActivityDots) Update(
	ctx context.Context,
	state ActivityState,
) error {
	return updateActivity(ctx, a, state)
}

func updateActivity(
	ctx context.Context,
	control Control,
	state ActivityState,
) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if control == nil || control.controlState() == nil ||
		control.controlState().app == nil {
		return ErrInvalidControl
	}
	transaction := control.controlState().app.NewTransaction()
	if err := transaction.SetActivityState(control, state); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

// SetProgressBarState records one atomic complete ProgressBar update.
func (t *Transaction) SetProgressBarState(
	control *ProgressBar,
	state ProgressBarState,
) error {
	target, err := t.control(control)
	if err != nil || target.kind != ControlProgressBar {
		return ErrInvalidControl
	}
	normalized, err := normalizeProgressBarState(state)
	if err != nil {
		return err
	}
	return t.recordProgressBehavior(
		target,
		progressBehavior{progressBar: &normalized},
	)
}

// SetMeterState records one atomic complete Meter update.
func (t *Transaction) SetMeterState(
	control *Meter,
	state MeterState,
) error {
	target, err := t.control(control)
	if err != nil || target.kind != ControlMeter {
		return ErrInvalidControl
	}
	normalized, err := normalizeMeterState(state)
	if err != nil {
		return err
	}
	return t.recordProgressBehavior(
		target,
		progressBehavior{meter: &normalized},
	)
}

// SetActivityState records one atomic complete Spinner or ActivityDots update.
func (t *Transaction) SetActivityState(
	control Control,
	state ActivityState,
) error {
	target, err := t.control(control)
	if err != nil ||
		(target.kind != ControlSpinner &&
			target.kind != ControlActivityDots) {
		return ErrInvalidControl
	}
	normalized, err := normalizeActivityState(state)
	if err != nil {
		return err
	}
	return t.recordProgressBehavior(
		target,
		progressBehavior{
			activity:     &normalized,
			activityDots: target.kind == ControlActivityDots,
		},
	)
}

func (t *Transaction) recordProgressBehavior(
	state *controlState,
	behavior progressBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationProgress, state: state, behavior: behavior,
	})
	return nil
}

func progressBehaviorEqual(left, right progressBehavior) bool {
	switch {
	case left.progressBar != nil && right.progressBar != nil:
		return *left.progressBar == *right.progressBar
	case left.meter != nil && right.meter != nil:
		return *left.meter == *right.meter
	case left.activity != nil && right.activity != nil:
		return left.activityDots == right.activityDots &&
			*left.activity == *right.activity
	default:
		return false
	}
}
