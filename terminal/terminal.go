// Package terminal provides the reusable physical-terminal boundary for
// expletives applications.
//
// Presenter owns the terminal lifecycle and physical frame projection.
// InputDecoder incrementally converts terminal bytes into logical toolkit key
// and bounded bracketed-paste events. Applications should keep both on their
// single terminal-owner goroutine and forward decoded events through
// expletives.App.DispatchKey or DispatchTextInput. The initial direct
// Presenter implementation supports Linux and a deliberately narrow xterm,
// Screen, and tmux profile set.
package terminal

import (
	"os"

	internalterminal "github.com/Hard-Problems-Group-LLC/expletives/internal/terminal"
)

// Presenter owns one physical terminal session acquired by Open. Its methods
// synchronize accidental concurrent calls, but applications should serialize
// Size, Present, ReadReady, Suspend, Resume, and Close on one terminal owner.
// Size queries authoritative geometry; Present synchronously projects a
// snapshot without retaining it; and ReadReady waits for at most its timeout
// and reports a timeout as an empty slice with no error. Suspend restores
// shell-facing state, Resume reacquires presentation state, and Close
// idempotently restores the state captured by Open.
type Presenter = internalterminal.Presenter

// Geometry is the current physical terminal width and height in cells. Its
// ToSize method converts it to an expletives.Size.
type Geometry = internalterminal.Geometry

// Profile identifies a deliberately supported physical-terminal contract.
// Callers should use the declared constants rather than inventing values.
type Profile = internalterminal.Profile

const (
	// ProfileXTerm selects the supported xterm-family projection.
	ProfileXTerm = internalterminal.ProfileXTerm
	// ProfileScreen selects the supported GNU Screen-family projection.
	ProfileScreen = internalterminal.ProfileScreen
	// ProfileTMux selects the supported tmux-family projection.
	ProfileTMux = internalterminal.ProfileTMux
)

var (
	// ErrClosed reports use after a Presenter has closed.
	ErrClosed = internalterminal.ErrClosed
	// ErrSuspended reports an operation that requires an active Presenter.
	ErrSuspended = internalterminal.ErrSuspended
	// ErrInvalidGeometry reports unusable terminal dimensions.
	ErrInvalidGeometry = internalterminal.ErrInvalidGeometry
	// ErrNotTerminal reports a stream that is not a usable terminal.
	ErrNotTerminal = internalterminal.ErrNotTerminal
	// ErrUnsupportedTerminal reports a terminal outside the supported profiles
	// or an operating system without an approved terminal implementation.
	ErrUnsupportedTerminal = internalterminal.ErrUnsupportedTerminal
)

// Open validates and acquires input and output using the current TERM value.
// When a bounded compiled terminfo directory entry is available, Open also
// requires it to corroborate the capabilities consumed by the already-narrow
// static profile. Missing terminfo retains that static contract; malformed or
// contradictory evidence fails closed before terminal mutation. The files
// must remain open for the Presenter lifetime; Open and Close do not transfer
// or close file ownership. The caller must call Close to restore the acquired
// terminal state.
func Open(input, output *os.File) (*Presenter, error) {
	return internalterminal.Open(input, output)
}

// DetectProfile maps a TERM value to the deliberately narrow supported
// profiles. It returns ErrUnsupportedTerminal rather than guessing
// capabilities from unrelated environment values.
func DetectProfile(term string) (Profile, error) {
	return internalterminal.DetectProfile(term)
}
