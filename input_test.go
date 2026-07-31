package expletives

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func heldFor(snapshot SnapshotV1, source string) []Key {
	for _, input := range snapshot.InputSources {
		if input.Source == source {
			return input.Held
		}
	}
	return nil
}

func TestFunctionKeyValidationIsBounded(t *testing.T) {
	t.Parallel()

	for _, key := range []Key{"f1", "f9", "f10", "f12"} {
		if !validKey(key) {
			t.Errorf("validKey(%q) = false", key)
		}
	}
	for _, key := range []Key{"f0", "f01", "f13", "f100", "f1suffix"} {
		if validKey(key) {
			t.Errorf("validKey(%q) = true", key)
		}
	}
}

func TestCommandRegistryIsInspectableAndControlsRouting(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	for _, definition := range []CommandDefinition{
		{
			ID:          "z.last",
			Description: "Last command",
			Enabled:     true,
			Automation:  false,
		},
		{
			ID:          "a.first",
			Description: "First command",
			Enabled:     false,
			Automation:  true,
		},
	} {
		if err := app.RegisterCommand(definition); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", definition.ID, err)
		}
	}
	if err := app.RegisterCommand(CommandDefinition{
		ID:      "z.last",
		Enabled: true,
	}); !errors.Is(err, ErrDuplicateCommand) {
		t.Fatalf("duplicate RegisterCommand() error = %v", err)
	}
	definitions := app.Commands()
	if len(definitions) != 3 ||
		definitions[0].ID != "a.first" ||
		definitions[1].ID != CommandOverflowDismiss ||
		definitions[2].ID != "z.last" {
		t.Fatalf("Commands() = %#v, want sorted inventory", definitions)
	}
	if err := app.RemoveCommand(CommandOverflowDismiss); !errors.Is(
		err,
		ErrInvalidRequest,
	) {
		t.Fatalf("RemoveCommand(built-in) error = %v, want ErrInvalidRequest", err)
	}

	var calls atomic.Int32
	if err := app.SetCommandRouter(func(
		context.Context,
		Command,
	) CommandResult {
		calls.Add(1)
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	disabled, err := app.InvokeCommand(
		context.Background(),
		"controller",
		"disabled-1",
		"a.first",
		"",
	)
	if err != nil {
		t.Fatalf("InvokeCommand(disabled) error = %v", err)
	}
	if disabled.Outcome != OutcomeRejected ||
		disabled.Code != "command_disabled" ||
		calls.Load() != 0 {
		t.Fatalf("disabled result = %+v calls=%d", disabled, calls.Load())
	}
	if err := app.ReplaceCommand(CommandDefinition{
		ID:          "a.first",
		Description: "First command",
		Enabled:     true,
		Automation:  true,
	}); err != nil {
		t.Fatalf("ReplaceCommand() error = %v", err)
	}
	applied, err := app.InvokeCommand(
		context.Background(),
		"controller",
		"enabled-1",
		"a.first",
		"",
	)
	if err != nil || applied.Outcome != OutcomeApplied || calls.Load() != 1 {
		t.Fatalf("enabled result = %+v error=%v calls=%d",
			applied, err, calls.Load())
	}

	chord := Chord{Key: "a", Modifiers: []Key{KeyAlt}}
	if err := app.BindChord(
		chord,
		CommandBinding{Command: "a.first"},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	if err := app.UnbindChord(chord); err != nil {
		t.Fatalf("UnbindChord() error = %v", err)
	}
	if err := app.RemoveCommand("a.first"); err != nil {
		t.Fatalf("RemoveCommand() error = %v", err)
	}
	unknown, err := app.InvokeCommand(
		context.Background(),
		"controller",
		"unknown-1",
		"a.first",
		"",
	)
	if err != nil || unknown.Code != "command_unknown" || calls.Load() != 1 {
		t.Fatalf("removed command result = %+v error=%v calls=%d",
			unknown, err, calls.Load())
	}
}

func TestCommandRouterFaultsAreBoundedAndPublicDiagnosticsAreSafe(t *testing.T) {
	t.Parallel()

	t.Run("timeout", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		if err := app.RegisterCommand(CommandDefinition{
			ID:      "handler.hang",
			Enabled: true,
		}); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		release := make(chan struct{})
		defer close(release)
		if err := app.SetCommandRouter(func(
			context.Context,
			Command,
		) CommandResult {
			<-release
			return CommandResult{Outcome: OutcomeApplied}
		}); err != nil {
			t.Fatalf("SetCommandRouter() error = %v", err)
		}
		ctx, cancel := context.WithTimeout(
			context.Background(),
			40*time.Millisecond,
		)
		defer cancel()
		completion, err := app.InvokeCommand(
			ctx,
			"controller",
			"hang-1",
			"handler.hang",
			"",
		)
		if err != nil {
			t.Fatalf("InvokeCommand(timeout) error = %v", err)
		}
		if completion.Outcome != OutcomeFailed ||
			completion.Code != "deadline_exceeded" ||
			!errors.Is(completion.Cause, context.DeadlineExceeded) {
			t.Fatalf("timeout completion = %+v", completion)
		}
	})

	t.Run("panic", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		if err := app.RegisterCommand(CommandDefinition{
			ID:      "handler.panic",
			Enabled: true,
		}); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		if err := app.SetCommandRouter(func(
			context.Context,
			Command,
		) CommandResult {
			panic("private panic detail")
		}); err != nil {
			t.Fatalf("SetCommandRouter() error = %v", err)
		}
		completion, err := app.InvokeCommand(
			context.Background(),
			"controller",
			"panic-1",
			"handler.panic",
			"",
		)
		if err != nil {
			t.Fatalf("InvokeCommand(panic) error = %v", err)
		}
		if completion.Code != "handler_panic" ||
			completion.Message != "command handler panicked" ||
			completion.Cause == nil {
			t.Fatalf("panic completion = %+v", completion)
		}
		encoded, err := json.Marshal(completion)
		if err != nil {
			t.Fatalf("Marshal(Completion) error = %v", err)
		}
		if strings.Contains(string(encoded), "private panic detail") {
			t.Fatalf("serialized Completion exposed local Cause: %s", encoded)
		}
	})

	t.Run("invalid public result", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		if err := app.RegisterCommand(CommandDefinition{
			ID:      "handler.invalid",
			Enabled: true,
		}); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		if err := app.SetCommandRouter(func(
			context.Context,
			Command,
		) CommandResult {
			return CommandResult{
				Outcome: OutcomeApplied,
				Message: strings.Repeat("x", MaxPublicMessageBytes+1),
			}
		}); err != nil {
			t.Fatalf("SetCommandRouter() error = %v", err)
		}
		completion, err := app.InvokeCommand(
			context.Background(),
			"controller",
			"invalid-1",
			"handler.invalid",
			"",
		)
		if err != nil {
			t.Fatalf("InvokeCommand(invalid) error = %v", err)
		}
		if completion.Outcome != OutcomeFailed ||
			completion.Code != "invalid_handler_result" {
			t.Fatalf("invalid handler completion = %+v", completion)
		}
	})
}

type completionCallResult struct {
	completion Completion
	err        error
}

func completionWithoutDeadlock(
	t *testing.T,
	call func() (Completion, error),
) (Completion, error) {
	t.Helper()
	result := make(chan completionCallResult, 1)
	go func() {
		completion, err := call()
		result <- completionCallResult{completion: completion, err: err}
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case completed := <-result:
		return completed.completion, completed.err
	case <-timer.C:
		t.Fatal("callback operation blocked behind its own dispatch")
		return Completion{}, errors.New("unreachable")
	}
}

func TestCommandHandlerCallbackSafeConfigurationAndCleanup(t *testing.T) {
	t.Run("replace handler", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		var replacementErr error
		var replacementCalls atomic.Int32
		replacement := func(context.Context, Command) (Outcome, error) {
			replacementCalls.Add(1)
			return OutcomeApplied, nil
		}
		if err := app.SetCommandHandler(func(
			context.Context,
			Command,
		) (Outcome, error) {
			replacementErr = app.SetCommandHandler(replacement)
			return OutcomeApplied, nil
		}); err != nil {
			t.Fatalf("SetCommandHandler(initial) error = %v", err)
		}

		if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
			return app.InvokeCommand(
				context.Background(),
				"controller",
				"replace-handler",
				"handler.replace",
				"",
			)
		}); err != nil {
			t.Fatalf("InvokeCommand(replace handler) error = %v", err)
		}
		if replacementErr != nil {
			t.Fatalf("callback SetCommandHandler() error = %v", replacementErr)
		}

		if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
			return app.InvokeCommand(
				context.Background(),
				"controller",
				"replacement-handler",
				"handler.next",
				"",
			)
		}); err != nil {
			t.Fatalf("InvokeCommand(replacement handler) error = %v", err)
		}
		if got := replacementCalls.Load(); got != 1 {
			t.Fatalf("replacement handler calls = %d, want 1", got)
		}
	})

	t.Run("bind chord", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		var bindErr error
		var boundCalls atomic.Int32
		if err := app.SetCommandHandler(func(
			_ context.Context,
			command Command,
		) (Outcome, error) {
			switch command.ID {
			case "binding.install":
				bindErr = app.BindChord(
					Chord{Key: "b", Modifiers: []Key{KeyAlt}},
					CommandBinding{Command: "binding.bound"},
				)
			case "binding.bound":
				boundCalls.Add(1)
			}
			return OutcomeApplied, nil
		}); err != nil {
			t.Fatalf("SetCommandHandler() error = %v", err)
		}

		if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
			return app.InvokeCommand(
				context.Background(),
				"controller",
				"install-binding",
				"binding.install",
				"",
			)
		}); err != nil {
			t.Fatalf("InvokeCommand(install binding) error = %v", err)
		}
		if bindErr != nil {
			t.Fatalf("callback BindChord() error = %v", bindErr)
		}
		if _, err := app.DispatchKey(
			context.Background(),
			"keyboard",
			"alt-down",
			KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
		); err != nil {
			t.Fatalf("DispatchKey(Alt down) error = %v", err)
		}
		if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
			return app.DispatchKey(
				context.Background(),
				"keyboard",
				"bound-press",
				KeyEvent{Kind: KeyEventPress, Key: "b"},
			)
		}); err != nil {
			t.Fatalf("DispatchKey(bound press) error = %v", err)
		}
		if got := boundCalls.Load(); got != 1 {
			t.Fatalf("bound handler calls = %d, want 1", got)
		}
	})

	t.Run("clear input source", func(t *testing.T) {
		app := mustApp(t, Size{Width: 2, Height: 1})
		if _, err := app.DispatchKey(
			context.Background(),
			"controller",
			"hold-alt",
			KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
		); err != nil {
			t.Fatalf("DispatchKey(Alt down) error = %v", err)
		}
		if err := app.SetCommandHandler(func(
			_ context.Context,
			command Command,
		) (Outcome, error) {
			app.ClearInputSource(command.Source)
			return OutcomeApplied, nil
		}); err != nil {
			t.Fatalf("SetCommandHandler() error = %v", err)
		}

		completion, err := completionWithoutDeadlock(t, func() (Completion, error) {
			return app.InvokeCommand(
				context.Background(),
				"controller",
				"clear-source",
				"source.clear",
				"",
			)
		})
		if err != nil {
			t.Fatalf("InvokeCommand(clear source) error = %v", err)
		}
		snapshot, err := app.SnapshotAt(completion.FrameSequence)
		if err != nil {
			t.Fatalf("SnapshotAt(clear source) error = %v", err)
		}
		if got := heldFor(snapshot, "controller"); len(got) != 0 {
			t.Fatalf("held keys after callback cleanup = %#v, want none", got)
		}
	})
}

func TestSameAppNestedDispatchAndWaitAreBounded(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *App) error
		want error
	}{
		{
			name: "DispatchKey",
			call: func(ctx context.Context, app *App) error {
				_, err := app.DispatchKey(
					ctx,
					"nested",
					"nested-key",
					KeyEvent{Kind: KeyEventPress, Key: "a"},
				)
				return err
			},
			want: ErrDispatchBusy,
		},
		{
			name: "InvokeCommand",
			call: func(ctx context.Context, app *App) error {
				_, err := app.InvokeCommand(
					ctx,
					"nested",
					"nested-command",
					"nested.invoke",
					"",
				)
				return err
			},
			want: ErrDispatchBusy,
		},
		{
			name: "ResetInput",
			call: func(ctx context.Context, app *App) error {
				_, err := app.ResetInput(ctx, "nested", "nested-reset")
				return err
			},
			want: ErrDispatchBusy,
		},
		{
			name: "WaitSnapshot",
			call: func(ctx context.Context, app *App) error {
				waitContext, cancel := context.WithTimeout(
					ctx,
					50*time.Millisecond,
				)
				defer cancel()
				after := app.Snapshot().Sequence
				_, err := app.WaitSnapshot(waitContext, after)
				return err
			},
			want: context.DeadlineExceeded,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			app := mustApp(t, Size{Width: 2, Height: 1})
			var nestedErr error
			if err := app.SetCommandHandler(func(
				ctx context.Context,
				_ Command,
			) (Outcome, error) {
				childContext, cancel := context.WithCancel(ctx)
				defer cancel()
				nestedErr = test.call(childContext, app)
				return OutcomeApplied, nil
			}); err != nil {
				t.Fatalf("SetCommandHandler() error = %v", err)
			}

			if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
				return app.InvokeCommand(
					context.Background(),
					"controller",
					"outer-command",
					"outer.invoke",
					"",
				)
			}); err != nil {
				t.Fatalf("InvokeCommand(outer) error = %v", err)
			}
			if !errors.Is(nestedErr, test.want) {
				t.Fatalf("nested %s error = %v, want %v",
					test.name, nestedErr, test.want)
			}
		})
	}
}

func TestCommandHandlerContextAllowsImmediateSnapshotRead(t *testing.T) {
	app := mustApp(t, Size{Width: 2, Height: 1})
	var observed SnapshotV1
	var waitErr error
	if err := app.SetCommandHandler(func(
		ctx context.Context,
		_ Command,
	) (Outcome, error) {
		current := app.Snapshot()
		observed, waitErr = app.WaitSnapshot(ctx, current.Sequence-1)
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	if _, err := completionWithoutDeadlock(t, func() (Completion, error) {
		return app.InvokeCommand(
			context.Background(),
			"controller",
			"snapshot-read",
			"snapshot.read",
			"",
		)
	}); err != nil {
		t.Fatalf("InvokeCommand(snapshot read) error = %v", err)
	}
	if waitErr != nil {
		t.Fatalf("immediate WaitSnapshot() error = %v", waitErr)
	}
	if observed.Sequence == 0 {
		t.Fatal("immediate WaitSnapshot() returned an empty snapshot")
	}
}

func TestCommandHandlerContextAllowsCrossAppDispatch(t *testing.T) {
	first := mustApp(t, Size{Width: 2, Height: 1})
	second := mustApp(t, Size{Width: 2, Height: 1})
	var secondCalls atomic.Int32
	if err := second.SetCommandHandler(func(
		_ context.Context,
		_ Command,
	) (Outcome, error) {
		secondCalls.Add(1)
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("second SetCommandHandler() error = %v", err)
	}
	var nestedErr error
	if err := first.SetCommandHandler(func(
		ctx context.Context,
		_ Command,
	) (Outcome, error) {
		_, nestedErr = second.InvokeCommand(
			ctx,
			"first-controller",
			"dispatch-to-second",
			"cycle.second",
			"",
		)
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("first SetCommandHandler() error = %v", err)
	}

	completion, err := completionWithoutDeadlock(t, func() (Completion, error) {
		return first.InvokeCommand(
			context.Background(),
			"controller",
			"start-cycle",
			"cycle.start",
			"",
		)
	})
	if err != nil {
		t.Fatalf("InvokeCommand(start cycle) error = %v", err)
	}
	if completion.Outcome != OutcomeApplied {
		t.Fatalf("start-cycle outcome = %q, want applied", completion.Outcome)
	}
	if nestedErr != nil {
		t.Fatalf("cross-App dispatch error = %v", nestedErr)
	}
	if got := secondCalls.Load(); got != 1 {
		t.Fatalf("second App handler calls = %d, want 1", got)
	}
}

func TestRawChordLifecycleAndCorrelatedCompletion(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 6, Height: 3})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "target",
		Bounds:        Rect{X: 1, Y: 1, Width: 2, Height: 1},
		Style:         "before",
	})

	const commandPaint CommandID = "scene.paint"
	if err := app.BindChord(
		Chord{Key: "a", Modifiers: []Key{KeyControl}},
		CommandBinding{Command: commandPaint},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	if err := app.SetCommandHandler(func(
		ctx context.Context,
		command Command,
	) (Outcome, error) {
		if err := ctx.Err(); err != nil {
			return OutcomeCancelled, err
		}
		if command.ID != commandPaint {
			return OutcomeRejected, nil
		}
		if err := panel.SetStyle("after"); err != nil {
			return OutcomeFailed, err
		}
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	down, err := app.DispatchKey(
		context.Background(),
		"automation.one",
		"key-down",
		KeyEvent{Kind: KeyEventDown, Key: KeyControl},
	)
	if err != nil {
		t.Fatalf("DispatchKey(Control down) error = %v", err)
	}
	if down.Outcome != OutcomeApplied || down.Command != "" {
		t.Fatalf("Control down completion = %+v", down)
	}
	downSnapshot, err := app.SnapshotAt(down.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(down) error = %v", err)
	}
	if downSnapshot.Completion == nil ||
		downSnapshot.Completion.RequestID != "key-down" {
		t.Fatalf("down snapshot completion = %#v", downSnapshot.Completion)
	}
	if got := heldFor(downSnapshot, "automation.one"); len(got) != 1 ||
		got[0] != KeyControl {
		t.Fatalf("held keys after down = %#v, want control", got)
	}

	press, err := app.DispatchKey(
		context.Background(),
		"automation.one",
		"key-press",
		KeyEvent{Kind: KeyEventPress, Key: "a"},
	)
	if err != nil {
		t.Fatalf("DispatchKey(a press) error = %v", err)
	}
	if press.Outcome != OutcomeApplied || press.Command != commandPaint {
		t.Fatalf("a press completion = %+v", press)
	}
	pressSnapshot, err := app.SnapshotAt(press.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(press) error = %v", err)
	}
	if got := heldFor(pressSnapshot, "automation.one"); len(got) != 1 ||
		got[0] != KeyControl {
		t.Fatalf("KeyPress changed held state: %#v", got)
	}
	targetView := controlByKey(t, pressSnapshot, "target")
	if targetView.Style != "after" {
		t.Fatalf("associated snapshot style = %q, want callback mutation", targetView.Style)
	}
	if pressSnapshot.Completion == nil ||
		pressSnapshot.Completion.FrameSequence != pressSnapshot.Sequence {
		t.Fatalf("press snapshot completion = %#v at sequence %d",
			pressSnapshot.Completion, pressSnapshot.Sequence)
	}

	other, err := app.DispatchKey(
		context.Background(),
		"automation.two",
		"other-press",
		KeyEvent{Kind: KeyEventPress, Key: "a"},
	)
	if err != nil {
		t.Fatalf("other source KeyPress error = %v", err)
	}
	if other.Outcome != OutcomeNoOp || other.Command != "" {
		t.Fatalf("source-local modifier state leaked: %+v", other)
	}

	up, err := app.DispatchKey(
		context.Background(),
		"automation.one",
		"key-up",
		KeyEvent{Kind: KeyEventUp, Key: KeyControl},
	)
	if err != nil {
		t.Fatalf("DispatchKey(Control up) error = %v", err)
	}
	if up.Outcome != OutcomeApplied {
		t.Fatalf("Control up outcome = %q, want applied", up.Outcome)
	}
	upSnapshot, err := app.SnapshotAt(up.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(up) error = %v", err)
	}
	if got := heldFor(upSnapshot, "automation.one"); len(got) != 0 {
		t.Fatalf("held keys after up = %#v, want none", got)
	}
}

func TestDirectCommandCarriesTargetAndRunsWithoutStateLock(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 4, Height: 2})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "target",
		Bounds:        Rect{Width: 2, Height: 1},
	})
	seen := make(chan Command, 1)
	if err := app.SetCommandHandler(func(
		ctx context.Context,
		command Command,
	) (Outcome, error) {
		if err := panel.SetBounds(Rect{X: 1, Width: 2, Height: 1}); err != nil {
			return OutcomeFailed, err
		}
		seen <- command
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	completion, err := app.InvokeCommand(
		context.Background(),
		"controller",
		"direct-1",
		"target.move",
		panel.ID(),
	)
	if err != nil {
		t.Fatalf("InvokeCommand() error = %v", err)
	}
	if completion.Outcome != OutcomeApplied ||
		completion.Command != "target.move" {
		t.Fatalf("InvokeCommand completion = %+v", completion)
	}
	command := <-seen
	if command.ID != "target.move" || command.Target != panel.ID() ||
		command.Source != "controller" {
		t.Fatalf("handler command = %+v", command)
	}
	snapshot, err := app.SnapshotAt(completion.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(completion) error = %v", err)
	}
	if got := controlByKey(t, snapshot, "target").Bounds.X; got != 1 {
		t.Fatalf("associated snapshot target X = %d, want 1", got)
	}
}

func TestResetInputIsCorrelatedAndIdempotent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	if _, err := app.DispatchKey(
		context.Background(),
		"client",
		"hold",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	); err != nil {
		t.Fatalf("DispatchKey(Alt down) error = %v", err)
	}

	reset, err := app.ResetInput(context.Background(), "client", "reset-1")
	if err != nil {
		t.Fatalf("ResetInput() error = %v", err)
	}
	if reset.Outcome != OutcomeApplied {
		t.Fatalf("first ResetInput outcome = %q, want applied", reset.Outcome)
	}
	snapshot, err := app.SnapshotAt(reset.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(reset) error = %v", err)
	}
	if got := heldFor(snapshot, "client"); len(got) != 0 {
		t.Fatalf("held keys after reset = %#v", got)
	}
	if snapshot.Completion == nil ||
		snapshot.Completion.RequestID != "reset-1" {
		t.Fatalf("reset snapshot completion = %#v", snapshot.Completion)
	}

	repeat, err := app.ResetInput(context.Background(), "client", "reset-2")
	if err != nil {
		t.Fatalf("second ResetInput() error = %v", err)
	}
	if repeat.Outcome != OutcomeNoOp {
		t.Fatalf("second ResetInput outcome = %q, want no_op", repeat.Outcome)
	}
}

func TestRequestIDsAreCorrelationLabelsAcrossSubmissionKinds(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	first, err := app.DispatchKey(
		context.Background(),
		"client",
		"request-1",
		KeyEvent{Kind: KeyEventPress, Key: "a"},
	)
	if err != nil {
		t.Fatalf("first request error = %v", err)
	}
	second, err := app.InvokeCommand(
		context.Background(),
		"client",
		"request-1",
		"app.noop",
		"",
	)
	if err != nil {
		t.Fatalf("reused correlation label error = %v", err)
	}
	if second.RequestID != first.RequestID ||
		second.FrameSequence <= first.FrameSequence {
		t.Fatalf("reused correlation completions = first:%+v second:%+v",
			first, second)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := app.ResetInput(ctx, "client", "not-consumed"); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("cancelled request error = %v, want context.Canceled", err)
	}
	if _, err := app.ResetInput(
		context.Background(),
		"client",
		"not-consumed",
	); err != nil {
		t.Fatalf("pre-accept cancellation consumed request ID: %v", err)
	}
}

func TestConcurrentRepeatedCorrelationLabelsRemainIndependent(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	var calls atomic.Int32
	if err := app.SetCommandHandler(func(
		context.Context,
		Command,
	) (Outcome, error) {
		calls.Add(1)
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := app.InvokeCommand(
				context.Background(),
				"client",
				"same-request",
				"do.work",
				"",
			)
			errs <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		default:
			t.Fatalf("concurrent request error = %v", err)
		}
	}
	if successes != 2 || calls.Load() != 2 {
		t.Fatalf("successes=%d handler calls=%d, want 2/2",
			successes, calls.Load())
	}
}

func TestCommandHandlersAreSerialized(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	entered := make(chan CommandID, 2)
	release := make(chan struct{}, 2)
	if err := app.SetCommandHandler(func(
		_ context.Context,
		command Command,
	) (Outcome, error) {
		entered <- command.ID
		<-release
		return OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	results := make(chan error, 2)
	go func() {
		_, err := app.InvokeCommand(
			context.Background(),
			"client.one",
			"serialized-1",
			"work.one",
			"",
		)
		results <- err
	}()
	if got := <-entered; got != "work.one" {
		t.Fatalf("first entered command = %q", got)
	}

	secondStarted := make(chan struct{})
	go func() {
		close(secondStarted)
		_, err := app.InvokeCommand(
			context.Background(),
			"client.two",
			"serialized-2",
			"work.two",
			"",
		)
		results <- err
	}()
	<-secondStarted
	select {
	case got := <-entered:
		t.Fatalf("second handler %q entered before first returned", got)
	default:
	}

	release <- struct{}{}
	if got := <-entered; got != "work.two" {
		t.Fatalf("second entered command = %q", got)
	}
	release <- struct{}{}
	for index := 0; index < 2; index++ {
		if err := <-results; err != nil {
			t.Fatalf("serialized InvokeCommand error = %v", err)
		}
	}
}

func TestExitPublishesFinalAssociatedSnapshot(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 2, Height: 1})
	panel := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "panel",
		Bounds:        Rect{Width: 1, Height: 1},
	})
	if err := app.SetCommandHandler(func(
		context.Context,
		Command,
	) (Outcome, error) {
		return OutcomeExited, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}
	before := app.Snapshot().Sequence
	completion, err := app.InvokeCommand(
		context.Background(),
		"client",
		"quit",
		"app.quit",
		"",
	)
	if err != nil {
		t.Fatalf("InvokeCommand(quit) error = %v", err)
	}
	if completion.Outcome != OutcomeExited || completion.FrameSequence <= before {
		t.Fatalf("quit completion = %+v", completion)
	}
	final, err := app.SnapshotAt(completion.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt(final) error = %v", err)
	}
	if !final.Final || final.Completion == nil ||
		final.Completion.RequestID != "quit" {
		t.Fatalf("final snapshot = Final:%v Completion:%#v",
			final.Final, final.Completion)
	}
	if _, err := app.WaitSnapshot(
		context.Background(),
		final.Sequence,
	); !errors.Is(err, ErrClosed) {
		t.Fatalf("WaitSnapshot(after final) error = %v, want ErrClosed", err)
	}
	if _, err := app.DispatchKey(
		context.Background(),
		"client",
		"after-final",
		KeyEvent{Kind: KeyEventPress, Key: "a"},
	); !errors.Is(err, ErrClosed) {
		t.Fatalf("post-final DispatchKey error = %v, want ErrClosed", err)
	}
	if err := panel.SetVisible(false); !errors.Is(err, ErrClosed) {
		t.Fatalf("post-final Panel mutation error = %v, want ErrClosed", err)
	}
}
