package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
	"github.com/Hard-Problems-Group-LLC/expletives/automation"
	"github.com/Hard-Problems-Group-LLC/expletives/internal/buildinfo"
	"github.com/Hard-Problems-Group-LLC/expletives/internal/demo"
	"github.com/Hard-Problems-Group-LLC/expletives/terminal"
)

const (
	exitOK        = 0
	exitFailure   = 1
	exitInterrupt = 130
)

type options struct {
	automationPath string
	headless       bool
	selfCheck      bool
	version        bool
	width          int
	height         int
	rootMinWidth   int
	rootMinHeight  int
	rootMaxWidth   int
	rootMaxHeight  int
	aspectWidth    int
	aspectHeight   int
}

type automationRun struct {
	server *automation.Server
	cancel context.CancelFunc
	done   chan struct{}

	stopOnce sync.Once
	serveErr error
	stopErr  error
}

type requestCounter struct {
	next atomic.Uint64
}

type terminalKeyState struct {
	control bool
	alt     bool
	meta    bool
	shift   bool
}

func (s *terminalKeyState) consume(event expletives.KeyEvent) os.Signal {
	var held *bool
	switch event.Key {
	case expletives.KeyControl:
		held = &s.control
	case expletives.KeyAlt:
		held = &s.alt
	case expletives.KeyMeta:
		held = &s.meta
	case expletives.KeyShift:
		held = &s.shift
	}
	if held != nil {
		switch event.Kind {
		case expletives.KeyEventDown:
			*held = true
		case expletives.KeyEventUp:
			*held = false
		}
	}
	if event.Kind != expletives.KeyEventPress || !s.control ||
		s.alt || s.meta || s.shift {
		return nil
	}
	switch event.Key {
	case "c":
		return syscall.SIGINT
	case "z":
		return syscall.SIGTSTP
	default:
		return nil
	}
}

func (s *terminalKeyState) reset() {
	*s = terminalKeyState{}
}

const signalCommandTimeout = 2 * time.Second

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin, stdout, stderr *os.File) int {
	configuration, err := parseOptions(args, stderr)
	if err != nil {
		return exitFailure
	}
	if configuration.version {
		fmt.Fprintf(stdout, "expletives-test %s\n", buildinfo.Current().String())
		return exitOK
	}
	if configuration.selfCheck {
		if err := demo.SelfCheck(); err != nil {
			fmt.Fprintf(stderr, "expletives-test self-check: %v\n", err)
			return exitFailure
		}
		fmt.Fprintln(stdout, "expletives-test self-check: ok")
		return exitOK
	}
	if configuration.headless && configuration.automationPath == "" {
		fmt.Fprintln(
			stderr,
			"expletives-test: --headless requires --automation or --self-check",
		)
		return exitFailure
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 8)
	managedSignals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGWINCH,
	}
	if !configuration.headless {
		managedSignals = append(
			managedSignals,
			syscall.SIGTSTP,
			syscall.SIGCONT,
		)
	}
	signal.Notify(signals, managedSignals...)
	defer signal.Stop(signals)

	var presenter *terminal.Presenter
	size := expletives.Size{
		Width:  configuration.width,
		Height: configuration.height,
	}
	if !configuration.headless {
		presenter, err = terminal.Open(stdin, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "expletives-test: acquire terminal: %v\n", err)
			return exitFailure
		}
		defer func() {
			if closeErr := presenter.Close(); closeErr != nil {
				fmt.Fprintf(stderr, "expletives-test: restore terminal: %v\n", closeErr)
			}
		}()
		geometry, sizeErr := presenter.Size()
		if sizeErr != nil {
			fmt.Fprintf(stderr, "expletives-test: query terminal size: %v\n", sizeErr)
			return exitFailure
		}
		size = boundedSize(geometry.ToSize())
	}

	if configuration.automationPath != "" {
		fmt.Fprintf(
			stderr,
			"WARNING: unauthenticated automation is active at %s; "+
				"any connector can observe and drive this process.\n",
			configuration.automationPath,
		)
	}
	scene, err := demo.NewWithRootConstraints(
		size,
		configuration.automationPath,
		expletives.RootConstraints{
			Minimum: expletives.Size{
				Width: configuration.rootMinWidth, Height: configuration.rootMinHeight,
			},
			Maximum: expletives.Size{
				Width: configuration.rootMaxWidth, Height: configuration.rootMaxHeight,
			},
			AspectRatio: expletives.AspectRatio{
				Width: configuration.aspectWidth, Height: configuration.aspectHeight,
			},
		},
	)
	if err != nil {
		fmt.Fprintf(stderr, "expletives-test: construct fixture: %v\n", err)
		return exitFailure
	}

	var attached *automationRun
	if configuration.automationPath != "" {
		attached, err = startAutomation(ctx, scene.App, configuration.automationPath)
		if err != nil {
			fmt.Fprintf(stderr, "expletives-test: enable automation: %v\n", err)
			return exitFailure
		}
		defer attached.stop()
		fmt.Fprintf(
			stderr,
			"expletives-test: automation ready at %s\n",
			attached.server.SocketPath(),
		)
	}

	counter := &requestCounter{}
	if configuration.headless {
		err = runHeadless(ctx, scene, signals, attached, counter)
	} else {
		err = runInteractive(ctx, scene, presenter, signals, attached, counter)
	}
	if stopErr := attached.stop(); stopErr != nil {
		err = errors.Join(err, stopErr)
	}
	if err != nil {
		fmt.Fprintf(stderr, "expletives-test: %v\n", err)
		return exitFailure
	}
	if completion := scene.App.Snapshot().Completion; completion != nil &&
		completion.Outcome == expletives.OutcomeInterrupted {
		return exitInterrupt
	}
	return exitOK
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var result options
	flags := flag.NewFlagSet("expletives-test", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(
		&result.automationPath,
		"automation",
		"",
		"explicit absolute Unix-socket path for unauthenticated automation",
	)
	flags.BoolVar(&result.headless, "headless", false, "run without a terminal")
	flags.BoolVar(&result.selfCheck, "self-check", false, "run deterministic fixture checks")
	flags.BoolVar(&result.version, "version", false, "print build provenance")
	flags.IntVar(&result.width, "width", 64, "headless frame width")
	flags.IntVar(&result.height, "height", 20, "headless frame height")
	flags.IntVar(&result.rootMinWidth, "root-min-width", 0, "optional root minimum width")
	flags.IntVar(&result.rootMinHeight, "root-min-height", 0, "optional root minimum height")
	flags.IntVar(&result.rootMaxWidth, "root-max-width", 0, "optional root maximum width")
	flags.IntVar(&result.rootMaxHeight, "root-max-height", 0, "optional root maximum height")
	flags.IntVar(&result.aspectWidth, "root-aspect-width", 0, "optional root cell-aspect width term")
	flags.IntVar(&result.aspectHeight, "root-aspect-height", 0, "optional root cell-aspect height term")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	size := expletives.Size{Width: result.width, Height: result.height}
	if size != boundedSize(size) || size.Width <= 0 || size.Height <= 0 {
		fmt.Fprintln(stderr, "expletives-test: width and height exceed supported bounds")
		return options{}, errors.New("invalid geometry")
	}
	values := []int{
		result.rootMinWidth,
		result.rootMinHeight,
		result.rootMaxWidth,
		result.rootMaxHeight,
		result.aspectWidth,
		result.aspectHeight,
	}
	for _, value := range values {
		if value < 0 {
			return options{}, errors.New("root constraints must be nonnegative")
		}
	}
	if (result.aspectWidth == 0) != (result.aspectHeight == 0) {
		return options{}, errors.New("root aspect ratio requires both terms")
	}
	return result, nil
}

func startAutomation(
	parent context.Context,
	app *expletives.App,
	path string,
) (*automationRun, error) {
	server, err := automation.NewServer(automation.ServerOptions{
		App:         app,
		SocketPath:  path,
		Application: "expletives-test",
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	run := &automationRun{
		server: server,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go func() {
		run.serveErr = server.Serve(ctx)
		close(run.done)
	}()
	return run, nil
}

func runInteractive(
	ctx context.Context,
	scene *demo.Scene,
	presenter *terminal.Presenter,
	signals <-chan os.Signal,
	attached *automationRun,
	counter *requestCounter,
) error {
	decoder, err := terminal.NewInputDecoder(terminal.InputDecoderOptions{})
	if err != nil {
		return fmt.Errorf("construct terminal input decoder: %w", err)
	}
	defer decoder.Reset()

	var presented uint64
	var keyState terminalKeyState
	for {
		snapshot := scene.App.Snapshot()
		if snapshot.Sequence != presented {
			if err := presenter.Present(snapshot); err != nil {
				return err
			}
			presented = snapshot.Sequence
		}
		if snapshot.Final {
			return nil
		}
		if err := automationFailure(attached); err != nil {
			return err
		}
		for {
			select {
			case received := <-signals:
				if received == syscall.SIGTSTP || received == syscall.SIGCONT {
					keyState.reset()
				}
				repaint, err := handleInteractiveSignal(
					ctx,
					scene,
					presenter,
					decoder,
					received,
					counter,
				)
				if err != nil {
					return err
				}
				if repaint {
					presented = 0
				}
			default:
				goto input
			}
		}
	input:
		wait := 40 * time.Millisecond
		if deadline, ok := decoder.Deadline(); ok {
			untilDeadline := time.Until(deadline)
			if untilDeadline < 0 {
				untilDeadline = 0
			}
			wait = min(wait, untilDeadline)
		}
		data, err := presenter.ReadReady(wait)
		if err != nil {
			return err
		}
		now := time.Now()
		var events []terminal.InputEvent
		if len(data) == 0 {
			events = decoder.FlushInput(now)
		} else {
			events = decoder.FeedInput(now, data)
		}
		for _, event := range events {
			requestID := counter.id("human")
			if event.KeyEvent != nil {
				// XTerm's enhanced-key modes encode Ctrl-C and Ctrl-Z as
				// structured chords, so the terminal driver cannot turn them
				// into SIGINT and SIGTSTP. Reserve those exact physical chords
				// for the same configurable interrupt and safe job-control
				// lifecycles as the corresponding terminal-driver signals.
				if received := keyState.consume(*event.KeyEvent); received != nil {
					keyState.reset()
					repaint, err := handleInteractiveSignal(
						ctx,
						scene,
						presenter,
						decoder,
						received,
						counter,
					)
					if err != nil {
						return err
					}
					if repaint {
						presented = 0
					}
					continue
				}
				if _, err := scene.App.DispatchKey(
					ctx,
					"terminal",
					requestID,
					*event.KeyEvent,
				); err != nil && !errors.Is(err, expletives.ErrClosed) {
					return err
				}
			}
			if event.TextInput != nil {
				if _, err := scene.App.DispatchTextInput(
					ctx,
					"terminal",
					requestID,
					*event.TextInput,
				); err != nil && !errors.Is(err, expletives.ErrClosed) {
					return err
				}
			}
		}
	}
}

func handleInteractiveSignal(
	ctx context.Context,
	scene *demo.Scene,
	presenter *terminal.Presenter,
	decoder *terminal.InputDecoder,
	received os.Signal,
	counter *requestCounter,
) (bool, error) {
	switch received {
	case syscall.SIGTSTP:
		decoder.Reset()
		resetContext, cancel := context.WithTimeout(
			ctx,
			signalCommandTimeout,
		)
		defer cancel()
		_, err := scene.App.ResetInput(
			resetContext,
			"terminal",
			counter.id("signal-suspend-reset"),
		)
		if errors.Is(err, expletives.ErrClosed) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("reset terminal input for suspension: %w", err)
		}
		if err := presenter.Suspend(); err != nil {
			return false, fmt.Errorf("suspend terminal presentation: %w", err)
		}

		// The notification handler has consumed the catchable SIGTSTP. Use an
		// uncatchable self-stop only after restoring the terminal; unlike a
		// re-raised SIGTSTP, SIGSTOP remains effective for orphaned process
		// groups such as a directly launched controlling-PTY session.
		stopErr := syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
		if stopErr != nil {
			resumeErr := presenter.Resume()
			return false, errors.Join(
				fmt.Errorf("stop process for suspension: %w", stopErr),
				resumeErr,
			)
		}
		// Execution resumes here only after SIGCONT. Reacquire immediately;
		// the queued SIGCONT notification is an idempotent safety net.
		if err := resumeInteractive(scene, presenter); err != nil {
			return false, err
		}
		return true, nil
	case syscall.SIGCONT:
		if err := resumeInteractive(scene, presenter); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, handleSignal(ctx, scene, presenter, received, counter)
	}
}

func resumeInteractive(
	scene *demo.Scene,
	presenter *terminal.Presenter,
) error {
	if err := presenter.Resume(); err != nil {
		return fmt.Errorf("resume terminal presentation: %w", err)
	}
	geometry, err := presenter.Size()
	if err != nil {
		return fmt.Errorf("query resumed terminal size: %w", err)
	}
	if err := scene.Resize(boundedSize(geometry.ToSize())); err != nil {
		return fmt.Errorf("resize resumed application: %w", err)
	}
	return nil
}

func runHeadless(
	ctx context.Context,
	scene *demo.Scene,
	signals <-chan os.Signal,
	attached *automationRun,
	counter *requestCounter,
) error {
	sequence := scene.App.Snapshot().Sequence
	for {
		snapshot := scene.App.Snapshot()
		if snapshot.Final {
			return nil
		}
		if err := automationFailure(attached); err != nil {
			return err
		}
		select {
		case received := <-signals:
			if err := handleSignal(ctx, scene, nil, received, counter); err != nil {
				return err
			}
			continue
		default:
		}
		waitContext, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		updated, err := scene.App.WaitSnapshot(waitContext, sequence)
		cancel()
		if err == nil {
			sequence = updated.Sequence
			continue
		}
		if !errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, expletives.ErrClosed) {
			return err
		}
	}
}

func handleSignal(
	ctx context.Context,
	scene *demo.Scene,
	presenter *terminal.Presenter,
	received os.Signal,
	counter *requestCounter,
) error {
	switch received {
	case syscall.SIGWINCH:
		if presenter == nil {
			return nil
		}
		geometry, err := presenter.Size()
		if err != nil {
			return err
		}
		return scene.Resize(boundedSize(geometry.ToSize()))
	case syscall.SIGINT:
		signalContext, cancel := context.WithTimeout(
			ctx,
			signalCommandTimeout,
		)
		defer cancel()
		completion, err := scene.App.InvokeCommand(
			signalContext,
			"signal",
			counter.id("signal"),
			demo.CommandAppInterrupt,
			"",
		)
		if errors.Is(err, expletives.ErrClosed) {
			return nil
		}
		if err != nil {
			return err
		}
		if completion.Outcome != expletives.OutcomeInterrupted {
			return fmt.Errorf(
				"SIGINT command completed as %q",
				completion.Outcome,
			)
		}
		return nil
	case syscall.SIGTERM, syscall.SIGHUP:
		signalContext, cancel := context.WithTimeout(
			ctx,
			signalCommandTimeout,
		)
		defer cancel()
		completion, err := scene.App.InvokeCommand(
			signalContext,
			"signal",
			counter.id("signal"),
			demo.CommandAppQuit,
			"",
		)
		if errors.Is(err, expletives.ErrClosed) {
			return nil
		}
		if err != nil {
			return err
		}
		if completion.Outcome != expletives.OutcomeExited {
			return fmt.Errorf(
				"termination command completed as %q",
				completion.Outcome,
			)
		}
		return nil
	default:
		return nil
	}
}

func automationFailure(attached *automationRun) error {
	if attached == nil {
		return nil
	}
	select {
	case <-attached.done:
		if attached.serveErr != nil {
			return fmt.Errorf("automation server: %w", attached.serveErr)
		}
		return errors.New("automation server stopped before application completion")
	default:
		return nil
	}
}

func (a *automationRun) stop() error {
	if a == nil {
		return nil
	}
	a.stopOnce.Do(func() {
		a.cancel()
		closeErr := a.server.Close()
		<-a.done
		a.stopErr = errors.Join(closeErr, a.serveErr)
	})
	return a.stopErr
}

func boundedSize(size expletives.Size) expletives.Size {
	return expletives.Size{
		Width:  max(size.Width, 0),
		Height: max(size.Height, 0),
	}
}

func (c *requestCounter) id(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, c.next.Add(1))
}
