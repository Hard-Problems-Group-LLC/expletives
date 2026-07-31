//go:build linux && integration

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	processPTYWidth       = 80
	processPTYHeight      = 24
	processPTYOutputLimit = 512 * 1024

	// Presenter.Close must emit this mode restoration before the process exits.
	processLeaveTerminal = "\x1b[0m\x1b[?25h\x1b[?1049l"
)

func TestDebugBinaryPTYProcessLifecycle(t *testing.T) {
	binary := debugExpletivesTestBinary(t)

	t.Run("fragmented input resize and clean quit", func(t *testing.T) {
		process := startDebugPTYProcess(t, binary)
		defer process.close()

		ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
		defer cancel()

		waitForInitialFrame(t, ctx, process)
		assertInteractiveTermios(t, process)

		resizeStart := process.output.mark()
		if err := process.pair.setSize(100, 30); err != nil {
			t.Fatalf("resize PTY to 100x30: %v", err)
		}
		waitForFrameHeight(t, ctx, process, resizeStart, 30)

		inputStart := process.output.mark()
		writePTYFragment(t, ctx, process, []byte{0x1b})
		writePTYFragment(t, ctx, process, []byte{'['})
		writePTYFragment(t, ctx, process, []byte{'A'})
		waitForFrameHeight(t, ctx, process, inputStart, 30)

		altStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b, 'F'})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("deliver Alt-F: %v", err)
		}
		waitForFrameHeight(t, ctx, process, altStart, 30)
		escapeStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b})
		waitForFrameHeight(t, ctx, process, escapeStart, 30)

		f10Start := process.output.mark()
		writePTY(
			t,
			process.pair.master,
			[]byte("\x1b[21~\x1b[C\x1b[B\r"),
		)
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("deliver F10/Right/Down/Enter: %v", err)
		}
		waitForFrameHeight(t, ctx, process, f10Start, 30)

		controlSpaceStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("deliver Ctrl-Space: %v", err)
		}
		waitForFrameHeight(t, ctx, process, controlSpaceStart, 30)
		controlSpaceEscapeStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("dismiss Ctrl-Space menu: %v", err)
		}
		waitForFrameHeight(t, ctx, process, controlSpaceEscapeStart, 30)

		nestedStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b, 'L', 's'})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("deliver Alt-L/S: %v", err)
		}
		waitForFrameHeight(t, ctx, process, nestedStart, 30)
		nestedEscapeStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("dismiss nested menu: %v", err)
		}
		waitForFrameHeight(t, ctx, process, nestedEscapeStart, 30)
		rootEscapeStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("dismiss root menu: %v", err)
		}
		waitForFrameHeight(t, ctx, process, rootEscapeStart, 30)

		helpStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{0x1b, 'H', 'a'})
		if err := process.pair.waitInputDrained(ctx); err != nil {
			t.Fatalf("deliver Alt-H/A: %v", err)
		}
		waitForFrameHeight(t, ctx, process, helpStart, 30)

		quitStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{'q'})
		if err := process.wait(ctx); err != nil {
			t.Fatalf(
				"debug expletives-test clean quit: %v; terminal output tail=%q",
				err,
				process.output.tail(4096),
			)
		}
		if err := process.output.waitContains(
			ctx,
			quitStart,
			[]byte(processLeaveTerminal),
		); err != nil {
			t.Fatalf("wait for terminal leave sequence: %v", err)
		}
		assertExactTermiosRestoration(t, process)
	})

	t.Run("terminal Ctrl-C returns interrupt status", func(t *testing.T) {
		process := startDebugPTYProcess(t, binary)
		defer process.close()

		ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
		defer cancel()

		waitForInitialFrame(t, ctx, process)
		assertInteractiveTermios(t, process)

		interruptStart := process.output.mark()
		writePTY(t, process.pair.master, []byte{3})
		err := process.wait(ctx)
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != exitInterrupt {
			t.Fatalf(
				"debug expletives-test Ctrl-C exit = %v, want status %d; "+
					"terminal output tail=%q",
				err,
				exitInterrupt,
				process.output.tail(4096),
			)
		}
		if err := process.output.waitContains(
			ctx,
			interruptStart,
			[]byte(processLeaveTerminal),
		); err != nil {
			t.Fatalf("wait for interrupt terminal leave sequence: %v", err)
		}
		assertExactTermiosRestoration(t, process)
	})
}

func debugExpletivesTestBinary(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate PTY integration test source")
	}
	path := filepath.Clean(filepath.Join(
		filepath.Dir(filename),
		"..",
		"..",
		"build",
		"debug",
		"expletives-test",
	))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf(
			"debug expletives-test artifact is unavailable at %s: %v; "+
				"run make test-integration",
			path,
			err,
		)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatalf("debug expletives-test artifact is not executable: %s", path)
	}
	return path
}

func waitForInitialFrame(
	t *testing.T,
	ctx context.Context,
	process *debugPTYProcess,
) {
	t.Helper()

	if err := process.output.waitContains(
		ctx,
		0,
		[]byte("expletives Toolkit Catalog"),
	); err != nil {
		t.Fatalf("wait for initial debug expletives-test frame: %v", err)
	}
}

func waitForFrameHeight(
	t *testing.T,
	ctx context.Context,
	process *debugPTYProcess,
	start int,
	height int,
) {
	t.Helper()

	rowAddress := []byte(fmt.Sprintf("\x1b[%d;1H", height))
	if err := process.output.waitContains(ctx, start, rowAddress); err != nil {
		t.Fatalf("wait for %d-row frame: %v", height, err)
	}
	if err := process.output.waitContains(ctx, start, []byte("\x1b[?25l")); err != nil {
		t.Fatalf("wait for completed %d-row frame: %v", height, err)
	}
}

func assertInteractiveTermios(t *testing.T, process *debugPTYProcess) {
	t.Helper()

	active, err := process.pair.termios()
	if err != nil {
		t.Fatalf("read active PTY termios: %v", err)
	}
	if active.Lflag&(syscall.ICANON|syscall.ECHO|syscall.ECHONL|syscall.IEXTEN) != 0 {
		t.Errorf("interactive PTY retained canonical/echo flags: Lflag=%#x", active.Lflag)
	}
	if active.Lflag&syscall.ISIG == 0 {
		t.Errorf("interactive PTY disabled signal generation: Lflag=%#x", active.Lflag)
	}
	if active.Cc[syscall.VINTR] != 3 {
		t.Errorf(
			"interactive PTY VINTR = %#x, want Ctrl-C %#x",
			active.Cc[syscall.VINTR],
			byte(3),
		)
	}
}

func assertExactTermiosRestoration(t *testing.T, process *debugPTYProcess) {
	t.Helper()

	restored, err := process.pair.termios()
	if err != nil {
		t.Fatalf("read restored PTY termios: %v", err)
	}
	if !reflect.DeepEqual(restored, process.original) {
		t.Errorf(
			"debug expletives-test did not restore exact PTY termios:\n got: %#v\nwant: %#v",
			restored,
			process.original,
		)
	}
}

type debugPTYProcess struct {
	command  *exec.Cmd
	pair     *processPTYPair
	original syscall.Termios
	output   *boundedPTYOutput
	done     chan struct{}
	waitErr  error
}

func startDebugPTYProcess(t *testing.T, binary string) *debugPTYProcess {
	t.Helper()

	pair, err := openProcessPTY(processPTYWidth, processPTYHeight)
	if err != nil {
		t.Skipf("Linux controlling PTY capability is unavailable: %v", err)
	}

	original, err := pair.termios()
	if err != nil {
		pair.close()
		t.Skipf("Linux controlling PTY termios capability is unavailable: %v", err)
	}
	original.Cc[syscall.VINTR] = 3
	if err := setProcessTermios(pair.slave.Fd(), original); err != nil {
		pair.close()
		t.Skipf("Linux controlling PTY configuration is unavailable: %v", err)
	}
	original, err = pair.termios()
	if err != nil {
		pair.close()
		t.Fatalf("read configured initial PTY termios: %v", err)
	}

	command := exec.Command(binary)
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	command.Stdin = pair.slave
	command.Stdout = pair.slave
	command.Stderr = pair.slave
	command.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}
	if err := command.Start(); err != nil {
		pair.close()
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ENOTTY) {
			t.Skipf("Linux controlling PTY assignment is unavailable: %v", err)
		}
		t.Fatalf("start debug expletives-test under PTY: %v", err)
	}

	process := &debugPTYProcess{
		command:  command,
		pair:     pair,
		original: original,
		output:   newBoundedPTYOutput(pair.master, processPTYOutputLimit),
		done:     make(chan struct{}),
	}
	go func() {
		process.waitErr = command.Wait()
		close(process.done)
	}()
	return process
}

func (p *debugPTYProcess) wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.waitErr
	case <-ctx.Done():
		return fmt.Errorf("wait for debug expletives-test: %w", ctx.Err())
	}
}

func (p *debugPTYProcess) close() {
	select {
	case <-p.done:
	default:
		_ = p.command.Process.Kill()
		select {
		case <-p.done:
		case <-time.After(2 * time.Second):
		}
	}
	p.pair.close()
	p.output.wait()
}

type processPTYPair struct {
	master *os.File
	slave  *os.File
}

func openProcessPTY(width, height int) (*processPTYPair, error) {
	masterFD, err := syscall.Open(
		"/dev/ptmx",
		syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open /dev/ptmx: %w", err)
	}
	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	if master == nil {
		_ = syscall.Close(masterFD)
		return nil, errors.New("construct /dev/ptmx file")
	}

	unlock := int32(0)
	if err := processIoctl(
		master.Fd(),
		syscall.TIOCSPTLCK,
		unsafe.Pointer(&unlock),
	); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("unlock PTY: %w", err)
	}
	var number uint32
	if err := processIoctl(
		master.Fd(),
		syscall.TIOCGPTN,
		unsafe.Pointer(&number),
	); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("query PTY number: %w", err)
	}

	slavePath := fmt.Sprintf("/dev/pts/%d", number)
	slave, err := os.OpenFile(
		slavePath,
		os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("open PTY slave %s: %w", slavePath, err)
	}
	pair := &processPTYPair{master: master, slave: slave}
	if err := pair.setSize(width, height); err != nil {
		pair.close()
		return nil, err
	}
	runtime.KeepAlive(&unlock)
	runtime.KeepAlive(&number)
	return pair, nil
}

func (p *processPTYPair) setSize(width, height int) error {
	if width <= 0 || width > int(^uint16(0)) ||
		height <= 0 || height > int(^uint16(0)) {
		return fmt.Errorf("invalid PTY geometry %dx%d", width, height)
	}
	size := struct {
		Rows    uint16
		Columns uint16
		XPixel  uint16
		YPixel  uint16
	}{
		Rows:    uint16(height),
		Columns: uint16(width),
	}
	if err := processIoctl(
		p.slave.Fd(),
		syscall.TIOCSWINSZ,
		unsafe.Pointer(&size),
	); err != nil {
		return fmt.Errorf("set PTY geometry %dx%d: %w", width, height, err)
	}
	runtime.KeepAlive(&size)
	return nil
}

func (p *processPTYPair) termios() (syscall.Termios, error) {
	var state syscall.Termios
	if err := processIoctl(
		p.slave.Fd(),
		syscall.TCGETS,
		unsafe.Pointer(&state),
	); err != nil {
		return syscall.Termios{}, err
	}
	runtime.KeepAlive(&state)
	return state, nil
}

func (p *processPTYPair) waitInputDrained(ctx context.Context) error {
	for {
		var pending int32
		if err := processIoctl(
			p.slave.Fd(),
			syscall.TIOCINQ,
			unsafe.Pointer(&pending),
		); err != nil {
			return fmt.Errorf("query pending PTY input: %w", err)
		}
		runtime.KeepAlive(&pending)
		if pending == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for PTY input consumption: %w", ctx.Err())
		default:
			runtime.Gosched()
		}
	}
}

func (p *processPTYPair) close() {
	_ = p.slave.Close()
	_ = p.master.Close()
}

func setProcessTermios(fd uintptr, state syscall.Termios) error {
	if err := processIoctl(fd, syscall.TCSETS, unsafe.Pointer(&state)); err != nil {
		return err
	}
	runtime.KeepAlive(&state)
	return nil
}

func processIoctl(fd uintptr, request uintptr, pointer unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		request,
		uintptr(pointer),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func writePTY(t *testing.T, file *os.File, data []byte) {
	t.Helper()

	for len(data) > 0 {
		written, err := file.Write(data)
		if err != nil {
			t.Fatalf("write PTY input %q: %v", data, err)
		}
		if written == 0 {
			t.Fatalf("write PTY input %q made no progress", data)
		}
		data = data[written:]
	}
}

func writePTYFragment(
	t *testing.T,
	ctx context.Context,
	process *debugPTYProcess,
	data []byte,
) {
	t.Helper()

	writePTY(t, process.pair.master, data)
	if err := process.pair.waitInputDrained(ctx); err != nil {
		t.Fatalf("deliver fragmented PTY input %q: %v", data, err)
	}
}

type boundedPTYOutput struct {
	mu       sync.Mutex
	data     []byte
	limit    int
	overflow bool
	readErr  error
	changed  chan struct{}
	done     chan struct{}
}

func newBoundedPTYOutput(file *os.File, limit int) *boundedPTYOutput {
	output := &boundedPTYOutput{
		data:    make([]byte, 0, min(limit, 32*1024)),
		limit:   limit,
		changed: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	go output.read(file)
	return output
}

func (o *boundedPTYOutput) read(file *os.File) {
	defer close(o.done)

	buffer := make([]byte, 4096)
	for {
		count, err := file.Read(buffer)
		if count > 0 {
			o.mu.Lock()
			remaining := o.limit - len(o.data)
			if remaining > 0 {
				o.data = append(o.data, buffer[:min(count, remaining)]...)
			}
			if count > remaining {
				o.overflow = true
			}
			o.mu.Unlock()
			o.signalChange()
		}
		if err != nil {
			o.mu.Lock()
			if !errors.Is(err, os.ErrClosed) &&
				!errors.Is(err, syscall.EIO) &&
				!errors.Is(err, syscall.EBADF) {
				o.readErr = err
			}
			o.mu.Unlock()
			o.signalChange()
			return
		}
	}
}

func (o *boundedPTYOutput) signalChange() {
	select {
	case o.changed <- struct{}{}:
	default:
	}
}

func (o *boundedPTYOutput) mark() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.data)
}

func (o *boundedPTYOutput) waitContains(
	ctx context.Context,
	start int,
	needle []byte,
) error {
	for {
		o.mu.Lock()
		if start < 0 || start > len(o.data) {
			o.mu.Unlock()
			return fmt.Errorf("invalid output offset %d", start)
		}
		found := bytes.Contains(o.data[start:], needle)
		overflow := o.overflow
		readErr := o.readErr
		tail := append([]byte(nil), boundedTail(o.data, 4096)...)
		o.mu.Unlock()

		if found {
			return nil
		}
		if overflow {
			return fmt.Errorf(
				"PTY output exceeded %d bytes before %q; tail=%q",
				o.limit,
				needle,
				tail,
			)
		}
		if readErr != nil {
			return fmt.Errorf(
				"read PTY output before %q: %w; tail=%q",
				needle,
				readErr,
				tail,
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"observe %q: %w; terminal output tail=%q",
				needle,
				ctx.Err(),
				tail,
			)
		case <-o.changed:
		case <-o.done:
		}
	}
}

func (o *boundedPTYOutput) tail(limit int) []byte {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]byte(nil), boundedTail(o.data, limit)...)
}

func (o *boundedPTYOutput) wait() {
	<-o.done
}

func boundedTail(data []byte, limit int) []byte {
	if len(data) <= limit {
		return data
	}
	return data[len(data)-limit:]
}
