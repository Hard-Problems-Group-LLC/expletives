package expletives

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// DefaultOverflowHandlerTimeout bounds one application disposition attempt.
const DefaultOverflowHandlerTimeout = 250 * time.Millisecond

// OverflowDisposition selects application handling or the toolkit fallback.
type OverflowDisposition uint8

const (
	OverflowUseDefault OverflowDisposition = iota
	OverflowHandled
)

// OverflowEvent is the immutable callback value for one overflow episode.
type OverflowEvent struct {
	Overflow OverflowSnapshot
}

// OverflowHandler runs outside toolkit locks with a bounded context.
type OverflowHandler func(context.Context, OverflowEvent) OverflowDisposition

type overflowRecord struct {
	snapshot OverflowSnapshot
	queued   bool
}

// SetOverflowHandler replaces the optional App callback. Nil restores the
// deterministic toolkit fallback.
func (a *App) SetOverflowHandler(handler OverflowHandler) error {
	if a == nil {
		return ErrInvalidControl
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	a.overflowHandler = handler
	a.signalOverflowLocked()
	return nil
}

// DismissOverflow acknowledges all currently active fallback notifications.
// It does not clear the underlying geometry fact.
func (a *App) DismissOverflow() error {
	if a == nil {
		return ErrInvalidControl
	}
	if err := a.beginMutation(context.Background()); err != nil {
		return err
	}
	defer a.endMutation()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	changed := a.dismissOverflowLocked()
	if changed {
		a.publishLocked(nil)
	}
	return nil
}

func (a *App) dismissOverflowLocked() bool {
	changed := false
	for _, record := range a.overflows {
		if record.snapshot.State == "default_active" {
			record.snapshot.State = "acknowledged"
			changed = true
		}
	}
	return changed
}

func (a *App) reconcileOverflowsLocked() {
	active := make(map[LayoutID]bool)
	for id, layout := range a.layoutsByID {
		if layout.destroyed {
			continue
		}
		available := Size{Width: layout.bounds.Width, Height: layout.bounds.Height}
		required := layout.minimum
		if available.Width >= required.Width && available.Height >= required.Height {
			continue
		}
		active[id] = true
		deficit := Size{
			Width:  max(0, required.Width-available.Width),
			Height: max(0, required.Height-available.Height),
		}
		if record, found := a.overflows[id]; found {
			record.snapshot.Available = available
			record.snapshot.Required = required
			record.snapshot.Deficit = deficit
			continue
		}
		a.nextOverflow++
		a.overflows[id] = &overflowRecord{snapshot: OverflowSnapshot{
			EpisodeID: fmt.Sprintf("overflow-%d", a.nextOverflow),
			Panel:     layout.owner.id,
			Layout:    id,
			Available: available,
			Required:  required,
			Deficit:   deficit,
			State:     "pending",
		}}
	}
	for id := range a.overflows {
		if !active[id] {
			delete(a.overflows, id)
		}
	}
	a.signalOverflowLocked()
}

func (a *App) overflowSnapshotsLocked() []OverflowSnapshot {
	ids := make([]string, 0, len(a.overflows))
	for id := range a.overflows {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	snapshots := make([]OverflowSnapshot, 0, len(ids))
	for _, encoded := range ids {
		snapshots = append(snapshots, a.overflows[LayoutID(encoded)].snapshot)
	}
	return snapshots
}

func (a *App) signalOverflowLocked() {
	if a.overflowRunning || a.final {
		return
	}
	for _, record := range a.overflows {
		if record.snapshot.State == "pending" && !record.queued {
			a.overflowRunning = true
			go a.runOverflowDispatcher()
			return
		}
	}
}

func (a *App) runOverflowDispatcher() {
	for {
		a.mu.Lock()
		if a.final {
			a.overflowRunning = false
			a.mu.Unlock()
			return
		}
		var selected *overflowRecord
		for _, record := range a.overflows {
			if record.snapshot.State == "pending" && !record.queued {
				selected = record
				break
			}
		}
		if selected == nil {
			a.overflowRunning = false
			a.mu.Unlock()
			return
		}
		selected.queued = true
		event := OverflowEvent{Overflow: selected.snapshot}
		handler := a.overflowHandler
		a.mu.Unlock()

		state := a.deliverOverflow(handler, event)

		a.mu.Lock()
		current := a.overflows[event.Overflow.Layout]
		if current != nil &&
			current.snapshot.EpisodeID == event.Overflow.EpisodeID &&
			current.snapshot.State == "pending" {
			current.snapshot.State = state
			a.publishLocked(nil)
		}
		a.mu.Unlock()
	}
}

func (a *App) deliverOverflow(
	handler OverflowHandler,
	event OverflowEvent,
) string {
	if handler == nil {
		return "default_active"
	}
	select {
	case a.overflowGate <- struct{}{}:
	default:
		return "default_active"
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		DefaultOverflowHandlerTimeout,
	)
	result := make(chan OverflowDisposition, 1)
	go func() {
		defer func() {
			<-a.overflowGate
			if recover() != nil {
				select {
				case result <- OverflowUseDefault:
				default:
				}
			}
		}()
		result <- handler(ctx, event)
	}()
	defer cancel()
	select {
	case disposition := <-result:
		if disposition == OverflowHandled {
			return "application_handled"
		}
		return "default_active"
	case <-ctx.Done():
		return "default_active"
	}
}

func (a *App) paintOverflowWarningLocked(
	frame *IntendedFrame,
	overflows []OverflowSnapshot,
) {
	active := false
	for _, overflow := range overflows {
		if overflow.State == "default_active" {
			active = true
			break
		}
	}
	if !active || frame.Size.Width == 0 || frame.Size.Height == 0 {
		return
	}
	text := []string{"!"}
	if frame.Size.Width >= 4 {
		text = []string{"[", "O", "K", "]"}
	}
	x := max(0, (frame.Size.Width-len(text))/2)
	y := max(0, frame.Size.Height/2)
	style := StyleID("_overflow.warning")
	resolved := ResolvedStyle{
		Foreground: RGB(0, 0, 0),
		Background: RGB(0xFF, 0xFF, 0),
		Attributes: StyleBold,
	}
	for index, grapheme := range text {
		a.setCellLocked(
			frame,
			x+index,
			y,
			grapheme,
			style,
			resolved,
			a.root.state.id,
		)
	}
}
