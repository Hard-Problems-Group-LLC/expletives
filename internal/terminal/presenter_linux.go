//go:build linux

package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

var (
	ErrClosed          = errors.New("terminal presenter is closed")
	ErrSuspended       = errors.New("terminal presenter is suspended")
	ErrInvalidGeometry = errors.New("invalid terminal geometry")
	ErrNotTerminal     = errors.New("file descriptor is not a terminal")
)

// Geometry is the physical terminal surface. Its fields intentionally match
// expletives.Size so conversion does not obscure the resize boundary.
type Geometry struct {
	Width  int
	Height int
}

// ToSize converts physical terminal geometry to toolkit surface geometry.
func (g Geometry) ToSize() expletives.Size {
	return expletives.Size{Width: g.Width, Height: g.Height}
}

type inputFile interface {
	io.Reader
	Fd() uintptr
}

type outputFile interface {
	io.Writer
	Fd() uintptr
}

type terminalSystem interface {
	getTermios(fd uintptr) (syscall.Termios, error)
	setTermios(fd uintptr, state syscall.Termios) error
	geometry(fd uintptr) (Geometry, error)
	waitReadable(fd uintptr, timeout time.Duration) (bool, error)
}

type linuxSystem struct{}

// Presenter owns one POSIX terminal session. Its methods are safe against
// accidental concurrent calls, but callers should serialize them on the
// application's one terminal owner.
type Presenter struct {
	mu sync.Mutex

	input   inputFile
	output  outputFile
	system  terminalSystem
	profile Profile
	glyphs  glyphMode

	original syscall.Termios
	active   bool
	closed   bool
	closeErr error
}

// Open validates and acquires the supplied terminal streams using TERM from
// the environment. Validation completes before terminal state is changed.
func Open(input, output *os.File) (*Presenter, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: nil input", ErrNotTerminal)
	}
	if output == nil {
		return nil, fmt.Errorf("%w: nil output", ErrNotTerminal)
	}
	term := os.Getenv("TERM")
	profile, err := DetectProfile(term)
	if err != nil {
		return nil, err
	}
	if err := validateInstalledTerminfo(term); err != nil {
		return nil, err
	}
	return openWithGlyphs(
		input,
		output,
		profile,
		glyphModeForLocale(activeLocale()),
		linuxSystem{},
	)
}

func openWith(
	input inputFile,
	output outputFile,
	profile Profile,
	system terminalSystem,
) (*Presenter, error) {
	return openWithGlyphs(input, output, profile, glyphASCII, system)
}

func openWithGlyphs(
	input inputFile,
	output outputFile,
	profile Profile,
	glyphs glyphMode,
	system terminalSystem,
) (*Presenter, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: nil input", ErrNotTerminal)
	}
	if output == nil {
		return nil, fmt.Errorf("%w: nil output", ErrNotTerminal)
	}
	if _, err := DetectProfile(string(profile)); err != nil {
		return nil, err
	}

	original, err := system.getTermios(input.Fd())
	if err != nil {
		return nil, fmt.Errorf("%w: input: %v", ErrNotTerminal, err)
	}
	if _, err := system.getTermios(output.Fd()); err != nil {
		return nil, fmt.Errorf("%w: output: %v", ErrNotTerminal, err)
	}
	if _, err := system.geometry(output.Fd()); err != nil {
		return nil, fmt.Errorf("query initial terminal geometry: %w", err)
	}

	presenter := &Presenter{
		input:    input,
		output:   output,
		system:   system,
		profile:  profile,
		glyphs:   glyphs,
		original: original,
	}
	if err := presenter.acquire(); err != nil {
		return nil, err
	}
	return presenter, nil
}

// Profile returns the immutable terminal profile selected during Open.
func (p *Presenter) Profile() Profile {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.profile
}

// Size queries current physical geometry. A resize signal is only a notice;
// this query is the authoritative geometry read.
func (p *Presenter) Size() (Geometry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return Geometry{}, ErrClosed
	}
	return p.system.geometry(p.output.Fd())
}

// Present emits one correctness-first full frame. The encoder never emits
// application-provided control bytes.
func (p *Presenter) Present(snapshot expletives.Snapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if !p.active {
		return ErrSuspended
	}

	frame, err := encodeSnapshot(snapshot, p.glyphs)
	if err != nil {
		return err
	}
	if err := writeAll(p.output, frame); err != nil {
		return fmt.Errorf("present terminal frame: %w", err)
	}
	return nil
}

func activeLocale() string {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

// ReadReady waits up to timeout for terminal input and returns one bounded
// chunk. A timeout is reported as an empty slice and a nil error. Exactly one
// terminal owner should call this method.
func (p *Presenter) ReadReady(timeout time.Duration) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, ErrClosed
	}
	if !p.active {
		return nil, ErrSuspended
	}
	if timeout < 0 {
		return nil, fmt.Errorf("wait for terminal input: negative timeout %s", timeout)
	}

	ready, err := p.system.waitReadable(p.input.Fd(), timeout)
	if err != nil {
		return nil, fmt.Errorf("wait for terminal input: %w", err)
	}
	if !ready {
		return []byte{}, nil
	}

	buffer := make([]byte, 256)
	n, err := p.input.Read(buffer)
	if n > 0 {
		return buffer[:n], nil
	}
	if err != nil {
		return nil, fmt.Errorf("read terminal input: %w", err)
	}
	return []byte{}, nil
}

// Suspend restores shell-facing terminal state without permanently closing
// the presenter. It is the safe boundary before a catchable job-control stop.
func (p *Presenter) Suspend() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if !p.active {
		return nil
	}

	err := errors.Join(
		wrapWriteError("leave terminal for suspension", writeAll(p.output, []byte(leaveTerminal))),
		wrapStateError("restore terminal for suspension", p.system.setTermios(p.input.Fd(), p.original)),
	)
	p.active = false
	return err
}

// Resume reacquires terminal modes after continuation. If reacquisition fails,
// it rolls back to the exact state captured by Open.
func (p *Presenter) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if p.active {
		return nil
	}
	return p.acquire()
}

// Close restores the exact original termios and exits all modes enabled by
// Open. It is idempotent and returns the first close result on later calls.
func (p *Presenter) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return p.closeErr
	}
	p.closed = true
	if !p.active {
		return nil
	}

	p.closeErr = errors.Join(
		wrapWriteError("leave terminal", writeAll(p.output, []byte(leaveTerminal))),
		wrapStateError("restore terminal", p.system.setTermios(p.input.Fd(), p.original)),
	)
	p.active = false
	return p.closeErr
}

func (p *Presenter) acquire() error {
	configured := makeInteractive(p.original)
	if err := p.system.setTermios(p.input.Fd(), configured); err != nil {
		return fmt.Errorf("configure terminal input: %w", err)
	}
	if err := writeAll(p.output, []byte(enterTerminal)); err != nil {
		rollbackErr := errors.Join(
			wrapWriteError("rollback terminal presentation", writeAll(p.output, []byte(leaveTerminal))),
			wrapStateError("rollback terminal input", p.system.setTermios(p.input.Fd(), p.original)),
		)
		return errors.Join(fmt.Errorf("enter terminal presentation: %w", err), rollbackErr)
	}
	p.active = true
	return nil
}

func makeInteractive(original syscall.Termios) syscall.Termios {
	configured := original
	configured.Iflag &^= syscall.ICRNL |
		syscall.INLCR |
		syscall.IGNCR |
		syscall.ISTRIP |
		syscall.IXON
	configured.Cflag &^= syscall.CSIZE | syscall.PARENB
	configured.Cflag |= syscall.CS8
	configured.Lflag &^= syscall.ICANON |
		syscall.ECHO |
		syscall.ECHONL |
		syscall.IEXTEN
	configured.Lflag |= syscall.ISIG
	configured.Cc[syscall.VMIN] = 0
	configured.Cc[syscall.VTIME] = 0
	return configured
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func wrapWriteError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func wrapStateError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (linuxSystem) getTermios(fd uintptr) (syscall.Termios, error) {
	var state syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&state)),
	)
	runtime.KeepAlive(&state)
	if errno != 0 {
		return syscall.Termios{}, errno
	}
	return state, nil
}

func (linuxSystem) setTermios(fd uintptr, state syscall.Termios) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&state)),
	)
	runtime.KeepAlive(&state)
	if errno != 0 {
		return errno
	}
	return nil
}

func (linuxSystem) geometry(fd uintptr) (Geometry, error) {
	var size struct {
		Rows    uint16
		Columns uint16
		XPixel  uint16
		YPixel  uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&size)),
	)
	runtime.KeepAlive(&size)
	if errno != 0 {
		return Geometry{}, errno
	}
	result := Geometry{
		Width:  int(size.Columns),
		Height: int(size.Rows),
	}
	if result.Width <= 0 || result.Height <= 0 {
		return Geometry{}, fmt.Errorf(
			"%w: width=%d height=%d",
			ErrInvalidGeometry,
			result.Width,
			result.Height,
		)
	}
	return result, nil
}

func (linuxSystem) waitReadable(fd uintptr, timeout time.Duration) (bool, error) {
	var readSet syscall.FdSet
	bitCount := uintptr(len(readSet.Bits) * 64)
	if fd >= bitCount {
		return false, fmt.Errorf("terminal descriptor %d exceeds select limit %d", fd, bitCount)
	}
	readSet.Bits[fd/64] |= int64(uint64(1) << uint(fd%64))

	timeval := syscall.NsecToTimeval(timeout.Nanoseconds())
	ready, err := syscall.Select(int(fd)+1, &readSet, nil, nil, &timeval)
	if errors.Is(err, syscall.EINTR) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ready > 0, nil
}
