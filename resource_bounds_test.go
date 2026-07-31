package expletives

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestTransactionCommitObservesCancellationAtMutationGate(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 2, Height: 1})
	app.mutationGate <- struct{}{}

	transaction := app.NewTransaction()
	panel, err := transaction.NewPanel(app.Root(), PanelOptions{
		Bounds: Rect{Width: 1, Height: 1},
	})
	if err != nil {
		<-app.mutationGate
		t.Fatalf("Transaction.NewPanel() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = transaction.Commit(ctx)
	<-app.mutationGate

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Transaction.Commit() error = %v, want deadline exceeded", err)
	}
	if panel.ID() != "" || panel.Visible() {
		t.Fatalf("cancelled provisional control remains active: ID=%q visible=%v",
			panel.ID(), panel.Visible())
	}
	if err := app.SetSize(Size{Width: 3, Height: 1}); err != nil {
		t.Fatalf("mutation after cancelled waiter error = %v", err)
	}
}

func TestCommandHandlerCapacityIsBounded(t *testing.T) {
	t.Parallel()

	app := mustApp(t, Size{Width: 2, Height: 1})
	if err := app.RegisterCommand(CommandDefinition{
		ID:      "handler.capacity",
		Enabled: true,
	}); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}
	if err := app.SetCommandRouter(func(
		context.Context,
		Command,
	) CommandResult {
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	for range MaxConcurrentCommandHandlers {
		app.handlerSlots <- struct{}{}
	}

	completion, err := app.InvokeCommand(
		context.Background(),
		"test",
		"capacity-1",
		"handler.capacity",
		"",
	)
	for range MaxConcurrentCommandHandlers {
		<-app.handlerSlots
	}
	if err != nil {
		t.Fatalf("InvokeCommand() error = %v", err)
	}
	if completion.Outcome != OutcomeFailed ||
		completion.Code != "handler_capacity" ||
		!errors.Is(completion.Cause, ErrDispatchBusy) {
		t.Fatalf("capacity completion = %+v", completion)
	}
}

func TestConcurrentControlCapacityIsReclaimed(t *testing.T) {
	app := mustApp(t, Size{Width: 1, Height: 1})
	transaction := app.NewTransaction()
	var victim *Panel
	for index := 1; index < MaxControls; index++ {
		panel, err := transaction.NewPanel(app.Root(), PanelOptions{})
		if err != nil {
			t.Fatalf("Transaction.NewPanel(%d) error = %v", index, err)
		}
		if victim == nil {
			victim = panel
		}
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("capacity Commit() error = %v", err)
	}
	if got := len(app.Snapshot().Controls); got != MaxControls {
		t.Fatalf("control count = %d, want %d", got, MaxControls)
	}

	if _, err := NewPanel(app.Root(), PanelOptions{}); !errors.Is(
		err,
		ErrControlCapacity,
	) {
		t.Fatalf("one-beyond NewPanel() error = %v, want ErrControlCapacity", err)
	}

	reclaim := app.NewTransaction()
	if err := reclaim.Destroy(victim); err != nil {
		t.Fatalf("Transaction.Destroy() error = %v", err)
	}
	replacement, err := reclaim.NewPanel(app.Root(), PanelOptions{})
	if err != nil {
		t.Fatalf("Transaction.NewPanel(replacement) error = %v", err)
	}
	if err := reclaim.Commit(context.Background()); err != nil {
		t.Fatalf("replacement Commit() error = %v", err)
	}
	if replacement.ID() == "" || replacement.ID() == victim.ID() {
		t.Fatalf("replacement ID = %q, destroyed ID = %q",
			replacement.ID(), victim.ID())
	}
	if got := len(app.Snapshot().Controls); got != MaxControls {
		t.Fatalf("reclaimed control count = %d, want %d", got, MaxControls)
	}
}

func TestHeldInputSourceCapacityIsBoundedAndReclaimed(t *testing.T) {
	app := mustApp(t, Size{Width: 1, Height: 1})

	if _, err := app.DispatchKey(
		context.Background(),
		"one-shot",
		"press-1",
		KeyEvent{Kind: KeyEventPress, Key: "a"},
	); err != nil {
		t.Fatalf("DispatchKey(one-shot) error = %v", err)
	}
	if got := len(app.held); got != 0 {
		t.Fatalf("one-shot input retained %d empty sources, want none", got)
	}

	for index := range MaxInputSources {
		source := fmt.Sprintf("source-%d", index)
		app.held[source] = map[Key]bool{KeyControl: true}
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"one-beyond",
		"source-capacity-1",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	)
	if err != nil {
		t.Fatalf("DispatchKey(one beyond) error = %v", err)
	}
	if completion.Outcome != OutcomeRejected ||
		completion.Code != "input_source_capacity" ||
		len(app.held) != MaxInputSources {
		t.Fatalf("one-beyond completion = %+v, sources = %d",
			completion, len(app.held))
	}

	if _, err := app.DispatchKey(
		context.Background(),
		"source-0",
		"release-source",
		KeyEvent{Kind: KeyEventUp, Key: KeyControl},
	); err != nil {
		t.Fatalf("DispatchKey(release source) error = %v", err)
	}
	if _, found := app.held["source-0"]; found {
		t.Fatal("empty held-key source was not reclaimed")
	}
	completion, err = app.DispatchKey(
		context.Background(),
		"replacement-source",
		"source-capacity-2",
		KeyEvent{Kind: KeyEventDown, Key: KeyAlt},
	)
	if err != nil {
		t.Fatalf("DispatchKey(replacement source) error = %v", err)
	}
	if completion.Outcome != OutcomeApplied ||
		len(app.held) != MaxInputSources {
		t.Fatalf("replacement completion = %+v, sources = %d",
			completion, len(app.held))
	}
}
