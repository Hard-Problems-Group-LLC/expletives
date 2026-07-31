package expletives

import "errors"

var (
	// ErrAppStopped reports an operation attempted after an App became final.
	ErrAppStopped = errors.New("expletives: application is stopped")
	// ErrClosed is the compatibility name for ErrAppStopped.
	ErrClosed = ErrAppStopped
	// ErrDuplicateKey reports an App-scoped stable-key collision.
	ErrDuplicateKey = errors.New("expletives: duplicate stable key")
	// ErrDuplicateCommand reports registration of an existing command ID.
	ErrDuplicateCommand = errors.New("expletives: duplicate command")
	// ErrDispatchBusy reports that the bounded dispatch wait elapsed.
	ErrDispatchBusy = errors.New("expletives: dispatch gate is busy")
	// ErrMutationBusy reports that the bounded mutation wait elapsed.
	ErrMutationBusy = errors.New("expletives: mutation gate is busy")
	// ErrTransactionCapacity reports that a transaction reached its operation
	// limit.
	ErrTransactionCapacity = errors.New("expletives: transaction operation capacity reached")
	// ErrControlCapacity reports that an App reached its active-control limit.
	ErrControlCapacity = errors.New("expletives: concurrent control capacity reached")
	// ErrDestroyed reports mutation of a logically destroyed control.
	ErrDestroyed = errors.New("expletives: control is destroyed")
	// ErrInvalidControl reports an invalid, aborted, or foreign control handle.
	ErrInvalidControl = errors.New("expletives: invalid control")
	// ErrInvalidChord reports a malformed, duplicate, or unknown chord binding.
	ErrInvalidChord = errors.New("expletives: invalid chord")
	// ErrInvalidGeometry reports geometry outside the supported bounds.
	ErrInvalidGeometry = errors.New("expletives: invalid geometry")
	// ErrInvalidKeyEvent reports an unsupported key or lifecycle transition.
	ErrInvalidKeyEvent = errors.New("expletives: invalid key event")
	// ErrInvalidParent reports a missing, foreign, or unusable container parent.
	ErrInvalidParent = errors.New("expletives: invalid parent")
	// ErrInvalidRequest reports malformed request or command data.
	ErrInvalidRequest = errors.New("expletives: invalid request")
	// ErrInvalidLayout reports a malformed, foreign, cyclic, or otherwise
	// unusable Layout.
	ErrInvalidLayout = errors.New("expletives: invalid layout")
	// ErrLayoutAttached reports an operation that requires a detached Layout.
	ErrLayoutAttached = errors.New("expletives: layout is already attached")
	// ErrLayoutCapacity reports a Layout or App layout-resource limit.
	ErrLayoutCapacity = errors.New("expletives: layout capacity reached")
	// ErrLayoutManaged reports manual geometry applied to a Layout-managed
	// control.
	ErrLayoutManaged = errors.New("expletives: control geometry is layout-managed")
	// ErrNotLayoutMember reports stacking requested for an unmanaged Panel.
	ErrNotLayoutMember = errors.New("expletives: control is not a layout member")
	// ErrNotFocusable reports focus requested for a non-focusable control.
	ErrNotFocusable = errors.New("expletives: control is not focusable")
	// ErrSnapshotNotRetained reports a sequence outside the retained history.
	ErrSnapshotNotRetained = errors.New("expletives: snapshot is not retained")
	// ErrStyleConflict reports incompatible definitions for one semantic style.
	ErrStyleConflict = errors.New("expletives: semantic style conflict")
	// ErrStyleMissing reports an absent or invalid semantic style definition.
	ErrStyleMissing = errors.New("expletives: semantic style is missing")
	// ErrTextLimit reports text outside the bounded display contract.
	ErrTextLimit = errors.New("expletives: displayed text exceeds a bounded limit")
)
