//go:build !linux

package terminal

import (
	"errors"
	"os"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

var (
	ErrClosed          = errors.New("terminal presenter is closed")
	ErrSuspended       = errors.New("terminal presenter is suspended")
	ErrInvalidGeometry = errors.New("invalid terminal geometry")
	ErrNotTerminal     = errors.New("file descriptor is not a terminal")
)

// Geometry is the physical terminal surface.
type Geometry struct {
	Width  int
	Height int
}

// ToSize converts physical terminal geometry to toolkit surface geometry.
func (g Geometry) ToSize() expletives.Size {
	return expletives.Size{Width: g.Width, Height: g.Height}
}

// Presenter is unavailable until this operating system has an approved
// terminal lifecycle implementation.
type Presenter struct{}

func Open(_, _ *os.File) (*Presenter, error) {
	return nil, errors.Join(ErrUnsupportedTerminal, errors.New("only Linux is supported"))
}

func (*Presenter) Profile() Profile                        { return "" }
func (*Presenter) Size() (Geometry, error)                 { return Geometry{}, ErrUnsupportedTerminal }
func (*Presenter) Present(expletives.Snapshot) error       { return ErrUnsupportedTerminal }
func (*Presenter) ReadReady(time.Duration) ([]byte, error) { return nil, ErrUnsupportedTerminal }
func (*Presenter) Suspend() error                          { return ErrUnsupportedTerminal }
func (*Presenter) Resume() error                           { return ErrUnsupportedTerminal }
func (*Presenter) Close() error                            { return nil }
