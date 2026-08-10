//go:build linux

package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

type fakeInput struct {
	reader *bytes.Reader
	fd     uintptr
}

func (f *fakeInput) Read(buffer []byte) (int, error) {
	return f.reader.Read(buffer)
}

func (f *fakeInput) Fd() uintptr {
	return f.fd
}

type fakeOutput struct {
	bytes.Buffer
	fd        uintptr
	writeCall int
	failOn    map[int]error
	maxWrite  int
}

func (f *fakeOutput) Fd() uintptr {
	return f.fd
}

func (f *fakeOutput) Write(data []byte) (int, error) {
	f.writeCall++
	if err := f.failOn[f.writeCall]; err != nil {
		return 0, err
	}
	if f.maxWrite > 0 && len(data) > f.maxWrite {
		data = data[:f.maxWrite]
	}
	return f.Buffer.Write(data)
}

type fakeTerminalSystem struct {
	original      syscall.Termios
	geometryValue Geometry
	ready         bool
	waitErr       error

	getCalls  int
	getFailAt int
	getErr    error

	setCalls  []syscall.Termios
	setFailAt int
	setErr    error
}

func (f *fakeTerminalSystem) getTermios(uintptr) (syscall.Termios, error) {
	f.getCalls++
	if f.getCalls == f.getFailAt {
		return syscall.Termios{}, f.getErr
	}
	return f.original, nil
}

func (f *fakeTerminalSystem) setTermios(_ uintptr, state syscall.Termios) error {
	f.setCalls = append(f.setCalls, state)
	if len(f.setCalls) == f.setFailAt {
		return f.setErr
	}
	return nil
}

func (f *fakeTerminalSystem) geometry(uintptr) (Geometry, error) {
	return f.geometryValue, nil
}

func (f *fakeTerminalSystem) waitReadable(uintptr, time.Duration) (bool, error) {
	return f.ready, f.waitErr
}

func testTermios() syscall.Termios {
	state := syscall.Termios{
		Iflag: syscall.ICRNL | syscall.INLCR | syscall.IGNCR |
			syscall.ISTRIP | syscall.IXON | syscall.BRKINT,
		Cflag: syscall.CS7 | syscall.PARENB | syscall.CREAD,
		Lflag: syscall.ICANON | syscall.ECHO | syscall.ECHONL |
			syscall.IEXTEN | syscall.ISIG,
	}
	state.Cc[syscall.VMIN] = 7
	state.Cc[syscall.VTIME] = 9
	return state
}

func newFakeSession() (*fakeInput, *fakeOutput, *fakeTerminalSystem) {
	return &fakeInput{
			reader: bytes.NewReader(nil),
			fd:     10,
		}, &fakeOutput{
			fd:     11,
			failOn: make(map[int]error),
		}, &fakeTerminalSystem{
			original:      testTermios(),
			geometryValue: Geometry{Width: 80, Height: 25},
		}
}

func TestOpenConfiguresAndCloseRestoresExactTermios(t *testing.T) {
	input, output, system := newFakeSession()

	presenter, err := openWith(input, output, ProfileXTerm, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}
	if got := output.String(); got != enterTerminal {
		t.Fatalf("open output = %q, want %q", got, enterTerminal)
	}
	if len(system.setCalls) != 1 {
		t.Fatalf("setTermios calls after open = %d, want 1", len(system.setCalls))
	}

	configured := system.setCalls[0]
	for name, flag := range map[string]uint32{
		"ICANON": syscall.ICANON,
		"ECHO":   syscall.ECHO,
		"ECHONL": syscall.ECHONL,
		"IEXTEN": syscall.IEXTEN,
	} {
		if configured.Lflag&flag != 0 {
			t.Errorf("configured Lflag retains %s", name)
		}
	}
	if configured.Lflag&syscall.ISIG == 0 {
		t.Error("configured Lflag cleared ISIG; Ctrl-C would stop being a signal")
	}
	if configured.Cflag&syscall.CSIZE != syscall.CS8 {
		t.Errorf("configured character size = %#x, want CS8", configured.Cflag&syscall.CSIZE)
	}
	if configured.Cflag&syscall.PARENB != 0 {
		t.Error("configured Cflag retains PARENB")
	}
	if configured.Cc[syscall.VMIN] != 0 || configured.Cc[syscall.VTIME] != 0 {
		t.Errorf(
			"configured VMIN/VTIME = %d/%d, want 0/0",
			configured.Cc[syscall.VMIN],
			configured.Cc[syscall.VTIME],
		)
	}

	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := output.String(); got != enterTerminal+leaveTerminal {
		t.Errorf("lifecycle output = %q, want enter then leave", got)
	}
	if len(system.setCalls) != 2 {
		t.Fatalf("setTermios calls after close = %d, want 2", len(system.setCalls))
	}
	if got := system.setCalls[1]; !reflect.DeepEqual(got, system.original) {
		t.Errorf("restored termios differs:\n got: %#v\nwant: %#v", got, system.original)
	}

	writes := output.writeCall
	sets := len(system.setCalls)
	if err := presenter.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if output.writeCall != writes || len(system.setCalls) != sets {
		t.Error("second Close() performed terminal work")
	}
}

func TestOpenValidationDoesNotMutateTerminal(t *testing.T) {
	input, output, system := newFakeSession()
	system.getFailAt = 2
	system.getErr = errors.New("not a tty")

	_, err := openWith(input, output, ProfileXTerm, system)
	if !errors.Is(err, ErrNotTerminal) {
		t.Fatalf("openWith() error = %v, want ErrNotTerminal", err)
	}
	if len(system.setCalls) != 0 {
		t.Errorf("setTermios calls = %d, want 0", len(system.setCalls))
	}
	if output.Len() != 0 {
		t.Errorf("output before validation failure = %q, want empty", output.String())
	}
}

func TestOpenWriteFailureRollsBackTerminal(t *testing.T) {
	input, output, system := newFakeSession()
	enterErr := errors.New("enter failed")
	output.failOn[1] = enterErr

	_, err := openWith(input, output, ProfileXTerm, system)
	if !errors.Is(err, enterErr) {
		t.Fatalf("openWith() error = %v, want enter failure", err)
	}
	if len(system.setCalls) != 2 {
		t.Fatalf("setTermios calls = %d, want configured and restore", len(system.setCalls))
	}
	if got := system.setCalls[1]; !reflect.DeepEqual(got, system.original) {
		t.Errorf("rollback termios differs:\n got: %#v\nwant: %#v", got, system.original)
	}
	if got := output.String(); got != leaveTerminal {
		t.Errorf("rollback output = %q, want leave sequence", got)
	}
}

func TestOpenTermiosFailureDoesNotWrite(t *testing.T) {
	input, output, system := newFakeSession()
	configureErr := errors.New("configure failed")
	system.setFailAt = 1
	system.setErr = configureErr

	_, err := openWith(input, output, ProfileXTerm, system)
	if !errors.Is(err, configureErr) {
		t.Fatalf("openWith() error = %v, want configure failure", err)
	}
	if output.Len() != 0 {
		t.Errorf("output after termios failure = %q, want empty", output.String())
	}
}

func TestOpenRetriesFailedRollbackCleanup(t *testing.T) {
	input, output, system := newFakeSession()
	enterErr := errors.New("enter failed")
	rollbackErr := errors.New("rollback restore failed")
	output.failOn[1] = enterErr
	system.setFailAt = 2
	system.setErr = rollbackErr

	_, err := openWith(input, output, ProfileXTerm, system)
	if !errors.Is(err, enterErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("openWith() error = %v, want enter and rollback failures", err)
	}
	if len(system.setCalls) != 3 {
		t.Fatalf("setTermios calls = %d, want configure, failed rollback, retry", len(system.setCalls))
	}
	if got := system.setCalls[2]; !reflect.DeepEqual(got, system.original) {
		t.Errorf("cleanup retry differs from original")
	}
	if got, want := output.String(), leaveTerminal+leaveTerminal; got != want {
		t.Errorf("cleanup output = %q, want %q", got, want)
	}
}

func TestCloseRestoresTermiosWhenExitWriteFails(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileXTerm, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}
	exitErr := errors.New("exit write failed")
	output.failOn[2] = exitErr

	err = presenter.Close()
	if !errors.Is(err, exitErr) {
		t.Fatalf("Close() error = %v, want exit write failure", err)
	}
	if len(system.setCalls) != 2 {
		t.Fatalf("setTermios calls = %d, want configure and restore", len(system.setCalls))
	}
	if got := system.setCalls[1]; !reflect.DeepEqual(got, system.original) {
		t.Errorf("restored termios differs:\n got: %#v\nwant: %#v", got, system.original)
	}

	writes := output.writeCall
	if secondErr := presenter.Close(); !errors.Is(secondErr, exitErr) {
		t.Fatalf("second Close() error = %v, want stored exit write failure", secondErr)
	}
	if output.writeCall != writes {
		t.Error("second Close() retried terminal output")
	}
}

func TestSuspendResumeAndClose(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileTMux, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}

	if err := presenter.Suspend(); err != nil {
		t.Fatalf("Suspend() error = %v", err)
	}
	if err := presenter.Suspend(); err != nil {
		t.Fatalf("second Suspend() error = %v", err)
	}
	if err := presenter.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if err := presenter.Resume(); err != nil {
		t.Fatalf("second Resume() error = %v", err)
	}
	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	wantOutput := enterTerminal + leaveTerminal + enterTerminal + leaveTerminal
	if got := output.String(); got != wantOutput {
		t.Errorf("lifecycle output = %q, want %q", got, wantOutput)
	}
	if len(system.setCalls) != 5 {
		t.Fatalf(
			"setTermios calls = %d, want configure/restore/configure/reassert/restore",
			len(system.setCalls),
		)
	}
	for _, index := range []int{1, 4} {
		if !reflect.DeepEqual(system.setCalls[index], system.original) {
			t.Errorf("restore call %d differs from original", index)
		}
	}
	configured := makeInteractive(system.original)
	for _, index := range []int{0, 2, 3} {
		if !reflect.DeepEqual(system.setCalls[index], configured) {
			t.Errorf("interactive call %d differs from configured state", index)
		}
	}
}

func TestActiveResumeReportsTermiosReassertionFailure(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileTMux, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}

	reassertErr := errors.New("termios reassertion failed")
	system.setFailAt = 2
	system.setErr = reassertErr
	if err := presenter.Resume(); !errors.Is(err, reassertErr) {
		t.Fatalf("active Resume() error = %v, want %v", err, reassertErr)
	}
	if got := output.String(); got != enterTerminal {
		t.Errorf("active Resume() output = %q, want no duplicate entry", got)
	}
	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() after reassertion failure = %v", err)
	}
}

func TestSuspendFailureLeavesCleanupForCloseRetry(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileTMux, system)
	if err != nil {
		t.Fatal(err)
	}
	leaveErr := errors.New("suspend leave failed")
	output.failOn[2] = leaveErr
	if err := presenter.Suspend(); !errors.Is(err, leaveErr) {
		t.Fatalf("Suspend() error = %v", err)
	}
	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() cleanup retry error = %v", err)
	}
	if len(system.setCalls) != 3 {
		t.Fatalf("setTermios calls = %d, want configure, suspend restore, close retry", len(system.setCalls))
	}
	if got, want := output.String(), enterTerminal+leaveTerminal; got != want {
		t.Errorf("terminal output = %q, want %q", got, want)
	}
}

func TestPresentFailureStillAllowsCloseRestoration(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileXTerm, system)
	if err != nil {
		t.Fatal(err)
	}
	presentErr := errors.New("frame write failed")
	output.failOn[2] = presentErr
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{Size: expletives.Size{Width: 1, Height: 1}, Cells: []expletives.Cell{{
			Grapheme: "x",
		}}},
	}
	if err := presenter.Present(snapshot); !errors.Is(err, presentErr) {
		t.Fatalf("Present() error = %v", err)
	}
	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() after Present failure = %v", err)
	}
	if len(system.setCalls) != 2 || !reflect.DeepEqual(system.setCalls[1], system.original) {
		t.Fatalf("Present failure did not restore original termios")
	}
}

func TestPresenterSustainedConcurrentCallsRemainSerialized(t *testing.T) {
	input, output, system := newFakeSession()
	output.maxWrite = 97
	presenter, err := openWith(input, output, ProfileXTerm, system)
	if err != nil {
		t.Fatal(err)
	}

	const (
		workers         = 8
		framesPerWorker = 64
	)
	cells := make([]expletives.Cell, 80*24)
	for index := range cells {
		cells[index] = expletives.Cell{Grapheme: "x"}
	}
	snapshot := expletives.Snapshot{Frame: expletives.IntendedFrame{
		Size:  expletives.Size{Width: 80, Height: 24},
		Cells: cells,
	}}

	errorsSeen := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range framesPerWorker {
				if err := presenter.Present(snapshot); err != nil {
					errorsSeen <- err
					return
				}
				if geometry, err := presenter.Size(); err != nil {
					errorsSeen <- err
					return
				} else if geometry != system.geometryValue {
					errorsSeen <- fmt.Errorf("geometry = %+v", geometry)
					return
				}
				if profile := presenter.Profile(); profile != ProfileXTerm {
					errorsSeen <- fmt.Errorf("profile = %q", profile)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent Presenter operation: %v", err)
	}

	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if output.writeCall <= workers*framesPerWorker {
		t.Fatalf("partial-write calls = %d, want more than frame count", output.writeCall)
	}
	if got := system.setCalls[len(system.setCalls)-1]; !reflect.DeepEqual(got, system.original) {
		t.Fatal("sustained presentation did not restore original termios")
	}
}

func TestResumeWriteFailureRollsBackToOriginalState(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileTMux, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}
	if err := presenter.Suspend(); err != nil {
		t.Fatalf("Suspend() error = %v", err)
	}

	resumeErr := errors.New("resume write failed")
	output.failOn[3] = resumeErr
	err = presenter.Resume()
	if !errors.Is(err, resumeErr) {
		t.Fatalf("Resume() error = %v, want resume write failure", err)
	}
	if len(system.setCalls) != 4 {
		t.Fatalf("setTermios calls = %d, want configure/restore/configure/rollback", len(system.setCalls))
	}
	if got := system.setCalls[3]; !reflect.DeepEqual(got, system.original) {
		t.Errorf("resume rollback differs:\n got: %#v\nwant: %#v", got, system.original)
	}
	if err := presenter.Close(); err != nil {
		t.Fatalf("Close() after failed Resume error = %v", err)
	}
}

func TestReadReadyTimeoutAndData(t *testing.T) {
	input, output, system := newFakeSession()
	presenter, err := openWith(input, output, ProfileScreen, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}
	t.Cleanup(func() {
		_ = presenter.Close()
	})

	got, err := presenter.ReadReady(0)
	if err != nil {
		t.Fatalf("ReadReady(timeout) error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("ReadReady(timeout) = %#v, want non-nil empty slice", got)
	}

	input.reader = bytes.NewReader([]byte("q"))
	system.ready = true
	got, err = presenter.ReadReady(time.Second)
	if err != nil {
		t.Fatalf("ReadReady(data) error = %v", err)
	}
	if string(got) != "q" {
		t.Errorf("ReadReady(data) = %q, want q", got)
	}
}

func TestSizeAndToSize(t *testing.T) {
	input, output, system := newFakeSession()
	system.geometryValue = Geometry{Width: 132, Height: 43}
	presenter, err := openWith(input, output, ProfileXTerm, system)
	if err != nil {
		t.Fatalf("openWith() error = %v", err)
	}
	t.Cleanup(func() {
		_ = presenter.Close()
	})

	got, err := presenter.Size()
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if got != system.geometryValue {
		t.Errorf("Size() = %+v, want %+v", got, system.geometryValue)
	}
	size := got.ToSize()
	if size.Width != 132 || size.Height != 43 {
		t.Errorf("ToSize() = %+v, want 132x43", size)
	}
}

func TestWriteAllHandlesPartialWrites(t *testing.T) {
	output := &fakeOutput{
		failOn:   make(map[int]error),
		maxWrite: 2,
	}
	if err := writeAll(output, []byte("abcdef")); err != nil {
		t.Fatalf("writeAll() error = %v", err)
	}
	if got := output.String(); got != "abcdef" {
		t.Errorf("writeAll() output = %q, want abcdef", got)
	}
}

func TestWriteAllRejectsNoProgress(t *testing.T) {
	err := writeAll(zeroWriter{}, []byte("x"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("writeAll() error = %v, want io.ErrShortWrite", err)
	}
}

func TestLinuxPTYLifecycleRestoresExactTermios(t *testing.T) {
	master, slave := openTestPTY(t, Geometry{Width: 80, Height: 24})
	t.Cleanup(func() {
		_ = master.Close()
		_ = slave.Close()
	})
	t.Setenv("TERM", "xterm-256color")

	system := linuxSystem{}
	before, err := system.getTermios(slave.Fd())
	if err != nil {
		t.Fatalf("get initial PTY termios: %v", err)
	}
	presenter, err := Open(slave, slave)
	if err != nil {
		t.Fatalf("Open(PTY) error = %v", err)
	}

	during, err := system.getTermios(slave.Fd())
	if err != nil {
		t.Fatalf("get active PTY termios: %v", err)
	}
	if during.Lflag&syscall.ICANON != 0 || during.Lflag&syscall.ECHO != 0 {
		t.Errorf("active PTY remains canonical or echoing: Lflag=%#x", during.Lflag)
	}
	if during.Lflag&syscall.ISIG == 0 {
		t.Errorf("active PTY lost ISIG: Lflag=%#x", during.Lflag)
	}

	if err := presenter.Close(); err != nil {
		t.Fatalf("Close(PTY) error = %v", err)
	}
	after, err := system.getTermios(slave.Fd())
	if err != nil {
		t.Fatalf("get restored PTY termios: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Errorf("PTY termios was not restored exactly:\n got: %#v\nwant: %#v", after, before)
	}
}

func TestLinuxPTYReadWaitHonorsTimeout(t *testing.T) {
	master, slave := openTestPTY(t, Geometry{Width: 80, Height: 24})
	t.Cleanup(func() {
		_ = master.Close()
		_ = slave.Close()
	})

	started := time.Now()
	ready, err := (linuxSystem{}).waitReadable(slave.Fd(), 40*time.Millisecond)
	if err != nil {
		t.Fatalf("waitReadable() error = %v", err)
	}
	if ready {
		t.Fatal("idle PTY reported readable")
	}
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond ||
		elapsed > 250*time.Millisecond {
		t.Fatalf("waitReadable() elapsed = %s, want a bounded 40ms wait", elapsed)
	}
}

func TestOpenTerminfoContradictionDoesNotMutatePTY(t *testing.T) {
	master, slave := openTestPTY(t, Geometry{Width: 80, Height: 24})
	t.Cleanup(func() {
		_ = master.Close()
		_ = slave.Close()
	})
	root := t.TempDir()
	directory := filepath.Join(root, "x")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "xterm-256color"),
		terminfoFixture(
			terminfoExtendedNumberMagic,
			"xterm-256color|fixture",
			256,
			map[int]bool{terminfoCursorAddressIndex: true},
		),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERMINFO", root)

	system := linuxSystem{}
	before, err := system.getTermios(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	_, err = Open(slave, slave)
	if !errors.Is(err, ErrUnsupportedTerminal) ||
		!strings.Contains(err.Error(), "cursor_address") {
		t.Fatalf("Open() error = %v", err)
	}
	after, err := system.getTermios(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("contradictory terminfo mutated PTY termios:\n got: %#v\nwant: %#v", after, before)
	}
}

func openTestPTY(t *testing.T, geometry Geometry) (*os.File, *os.File) {
	t.Helper()

	masterFD, err := syscall.Open(
		"/dev/ptmx",
		syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		t.Skipf("open /dev/ptmx: %v", err)
	}
	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	if master == nil {
		_ = syscall.Close(masterFD)
		t.Fatal("os.NewFile(/dev/ptmx) returned nil")
	}

	unlock := int32(0)
	if err := testIoctl(master.Fd(), syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		_ = master.Close()
		t.Skipf("unlock PTY: %v", err)
	}
	var number uint32
	if err := testIoctl(master.Fd(), syscall.TIOCGPTN, unsafe.Pointer(&number)); err != nil {
		_ = master.Close()
		t.Skipf("query PTY number: %v", err)
	}
	slavePath := fmt.Sprintf("/dev/pts/%d", number)
	slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		t.Skipf("open PTY slave %s: %v", slavePath, err)
	}

	size := struct {
		Rows    uint16
		Columns uint16
		XPixel  uint16
		YPixel  uint16
	}{
		Rows:    uint16(geometry.Height),
		Columns: uint16(geometry.Width),
	}
	if err := testIoctl(slave.Fd(), syscall.TIOCSWINSZ, unsafe.Pointer(&size)); err != nil {
		_ = slave.Close()
		_ = master.Close()
		t.Skipf("set PTY geometry: %v", err)
	}
	runtime.KeepAlive(&unlock)
	runtime.KeepAlive(&number)
	runtime.KeepAlive(&size)
	return master, slave
}

func testIoctl(fd uintptr, request uintptr, pointer unsafe.Pointer) error {
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

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) {
	return 0, nil
}
