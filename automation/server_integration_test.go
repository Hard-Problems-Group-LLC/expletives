//go:build linux

package automation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const (
	testCommandFixtureToggle = "fixture.toggle"
	testCommandScenarioReset = "scenario.reset"
	testCommandAppQuit       = "app.quit"
	testCommandHumanOnly     = "human.only"
	testTargetKey            = "target.panel"
)

func TestServerRequiresApplicationAndDoesNotInventCommands(t *testing.T) {
	t.Parallel()

	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 2, Height: 1},
		Scenario: "automation.empty-inventory",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "automation.sock")
	if _, err := NewServer(ServerOptions{App: app, SocketPath: path}); err == nil {
		t.Fatal("NewServer() without Application error = nil")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("server bound endpoint before rejecting Application: %v", err)
	}

	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
		select {
		case <-serveErrors:
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop during cleanup")
		}
	})

	client := dialTestClient(t, path)
	t.Cleanup(func() { _ = client.Close() })
	hello := client.Hello()
	if len(hello.Commands) != 1 ||
		hello.Commands[0] != string(expletives.CommandOverflowDismiss) {
		t.Fatalf(
			"built-in commands advertised as %#v, want overflow dismissal",
			hello.Commands,
		)
	}
}

func TestServerClientClosedLoop(t *testing.T) {
	t.Parallel()

	app, initialStyle, toggledStyle := newTestApp(t)
	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App:         app,
		SocketPath:  path,
		Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
	})

	client := dialTestClient(t, path)
	hello := client.Hello()
	if hello.Protocol != Protocol || hello.Version != Version {
		t.Fatalf("hello protocol/version = %q/%d", hello.Protocol, hello.Version)
	}
	if len(hello.SupportedVersions) != 1 || hello.SupportedVersions[0] != Version {
		t.Fatalf("hello supported versions = %v, want [%d]", hello.SupportedVersions, Version)
	}
	for _, operation := range hello.Operations {
		if operation == "cancel" {
			t.Fatal("hello advertises removed v1 cancel operation")
		}
	}
	if len(hello.Commands) != 4 {
		t.Fatalf("hello commands = %v, want explicit test inventory", hello.Commands)
	}
	foundOverflowDismiss := false
	for _, command := range hello.Commands {
		if command == testCommandHumanOnly {
			t.Fatal("hello advertises command not exposed to automation")
		}
		if command == string(expletives.CommandOverflowDismiss) {
			foundOverflowDismiss = true
		}
	}
	if !foundOverflowDismiss {
		t.Fatal("hello omits built-in overflow dismissal")
	}
	if hello.Scenario != "foundation.absolute-panels" {
		t.Errorf("hello scenario = %q", hello.Scenario)
	}
	if !hello.Unauthenticated {
		t.Error("hello does not identify unauthenticated mode")
	}

	observed := request(t, func(ctx context.Context) (Completion, error) {
		return client.Observe(ctx, "observe-1", nil)
	})
	if observed.Snapshot == nil || observed.FrameSequence != observed.Snapshot.Sequence {
		t.Fatal("observe completion does not carry its associated snapshot")
	}
	if observed.Snapshot.Frame.Size != (Size{Width: 8, Height: 3}) {
		t.Errorf("initial frame size = %+v", observed.Snapshot.Frame.Size)
	}

	down := request(t, func(ctx context.Context) (Completion, error) {
		return client.InjectInput(ctx, "key-down-1", KeyEvent{Kind: KeyDown, Key: "control"})
	})
	if down.Outcome != OutcomeApplied {
		t.Errorf("key down outcome = %q, want applied", down.Outcome)
	}
	assertHeldKey(t, down.Snapshot, "control", true)

	pressed := request(t, func(ctx context.Context) (Completion, error) {
		return client.InjectInput(ctx, "key-press-1", KeyEvent{Kind: KeyPress, Key: "r"})
	})
	if pressed.Outcome != OutcomeApplied {
		t.Errorf("key press outcome = %q, want applied", pressed.Outcome)
	}
	if got := pressed.Snapshot.Controls[0].Style; got != toggledStyle {
		t.Errorf("root style after chord = %+v, want %+v", got, toggledStyle)
	}
	if pressed.Snapshot.Completion == nil ||
		pressed.Snapshot.Completion.RequestID != "key-press-1" {
		t.Error("key press snapshot omits exact request association")
	}

	up := request(t, func(ctx context.Context) (Completion, error) {
		return client.InjectInput(ctx, "key-up-1", KeyEvent{Kind: KeyUp, Key: "control"})
	})
	assertHeldKey(t, up.Snapshot, "control", false)

	reset := request(t, func(ctx context.Context) (Completion, error) {
		return client.InvokeCommand(ctx, "command-1", testCommandScenarioReset, "")
	})
	if reset.Outcome != OutcomeApplied {
		t.Errorf("reset outcome = %q, want applied", reset.Outcome)
	}
	if got := reset.Snapshot.Controls[0].Style; got != initialStyle {
		t.Errorf("root style after reset = %+v, want %+v", got, initialStyle)
	}
	targeted := request(t, func(ctx context.Context) (Completion, error) {
		return client.InvokeCommand(
			ctx,
			"targeted-command-1",
			testCommandScenarioReset,
			testTargetKey,
		)
	})
	if targeted.Outcome != OutcomeApplied {
		t.Errorf("stable target-key outcome = %q, want applied", targeted.Outcome)
	}
	missingTarget := request(t, func(ctx context.Context) (Completion, error) {
		return client.InvokeCommand(
			ctx,
			"missing-target-1",
			testCommandScenarioReset,
			"missing.panel",
		)
	})
	if missingTarget.Outcome != OutcomeRejected ||
		missingTarget.Error == nil ||
		missingTarget.Error.Code != "target_not_found" {
		t.Errorf(
			"missing target result = outcome %q, error %+v, want rejected/target_not_found",
			missingTarget.Outcome,
			missingTarget.Error,
		)
	}

	exactSequence := pressed.FrameSequence
	exact := request(t, func(ctx context.Context) (Completion, error) {
		return client.Observe(ctx, "observe-exact-1", &exactSequence)
	})
	if exact.FrameSequence != pressed.FrameSequence {
		t.Errorf("exact frame sequence = %d, want %d", exact.FrameSequence, pressed.FrameSequence)
	}
	if got := exact.Snapshot.Controls[0].Style; got != toggledStyle {
		t.Errorf("exact historical root style = %+v, want %+v", got, toggledStyle)
	}

	// Churn the App's independent snapshot history past its current bound.
	// query_result must still return the target snapshot owned by the retained
	// protocol result rather than depending on App.SnapshotAt.
	for index := 0; index < 70; index++ {
		width := 7 + index%2
		if err := app.SetSize(expletives.Size{Width: width, Height: 3}); err != nil {
			t.Fatalf("SetSize() during snapshot churn error = %v", err)
		}
	}

	query := request(t, func(ctx context.Context) (Completion, error) {
		return client.QueryResult(ctx, "query-1", "observe-exact-1")
	})
	if query.Result == nil || query.Result.Query == nil {
		t.Fatal("query completion omits query result")
	}
	if query.Result.Query.Status != "completed" {
		t.Errorf("query status = %q, want completed", query.Result.Query.Status)
	}
	if query.Result.Query.Completion == nil ||
		query.Result.Query.Completion.RequestID != "observe-exact-1" {
		t.Error("query does not return retained exact-observe completion")
	}
	if query.FrameSequence != pressed.FrameSequence ||
		query.Snapshot.Sequence != pressed.FrameSequence ||
		query.Snapshot.Controls[0].Style != toggledStyle {
		t.Errorf(
			"query exact evidence sequence/style = %d/%+v, want %d/%+v",
			query.FrameSequence,
			query.Snapshot.Controls[0].Style,
			pressed.FrameSequence,
			toggledStyle,
		)
	}

	if _, err := client.Observe(testContext(t), "observe-exact-1", nil); err == nil {
		t.Fatal("duplicate request ID error = nil")
	} else {
		var responseError *ResponseError
		if !errors.As(err, &responseError) ||
			responseError.Code != "duplicate_request_id" {
			t.Fatalf("duplicate request error = %v", err)
		}
	}

	resetInput := request(t, func(ctx context.Context) (Completion, error) {
		return client.ResetInput(ctx, "input-reset-1")
	})
	if resetInput.Outcome != OutcomeNoOp {
		t.Errorf("empty input reset outcome = %q, want no_op", resetInput.Outcome)
	}

	after := resetInput.FrameSequence
	waitResult := make(chan Completion, 1)
	waitError := make(chan error, 1)
	waitContext := testContext(t)
	go func() {
		completion, waitErr := client.WaitSnapshot(
			waitContext,
			"wait-1",
			after,
		)
		if waitErr != nil {
			waitError <- waitErr
			return
		}
		waitResult <- completion
	}()
	if err := app.SetSize(expletives.Size{Width: 7, Height: 3}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	select {
	case err := <-waitError:
		t.Fatalf("WaitSnapshot() error = %v", err)
	case waited := <-waitResult:
		if waited.FrameSequence <= after {
			t.Errorf("wait sequence = %d, want > %d", waited.FrameSequence, after)
		}
		if waited.Snapshot.Frame.Size.Width != 7 {
			t.Errorf("wait frame width = %d, want 7", waited.Snapshot.Frame.Size.Width)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitSnapshot() did not complete")
	}

	shutdown := request(t, func(ctx context.Context) (Completion, error) {
		return client.Shutdown(ctx, "shutdown-1")
	})
	if shutdown.Outcome != OutcomeExited || !shutdown.Snapshot.Final {
		t.Errorf("shutdown outcome/final = %q/%t, want exited/true", shutdown.Outcome, shutdown.Snapshot.Final)
	}

	select {
	case err := <-serveErrors:
		if err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve() did not return after shutdown")
	}
	cancelServe()
	_ = client.Close()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("socket remains after shutdown: %v", err)
	}
}

func TestServerClientAcceptsWrappedListBoxReplacementSnapshots(t *testing.T) {
	t.Parallel()

	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 30, Height: 10},
		Scenario: "automation.wrapped-list-replacement",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := expletives.NewListBox(
		app.Root(),
		listBoxSnapshotFixtureOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
		select {
		case <-serveErrors:
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop during cleanup")
		}
	})

	client := dialTestClient(t, path)
	t.Cleanup(func() { _ = client.Close() })
	observe := func(requestID string) *ListBoxDetails {
		t.Helper()
		completion := request(t, func(ctx context.Context) (Completion, error) {
			return client.Observe(ctx, requestID, nil)
		})
		return projectedListBoxControl(t, completion.Snapshot).Details.ListBox
	}
	initial := observe("wrapped-list-initial")
	if initial.CurrentIndex != 2 ||
		initial.Viewport.State.Offset.Y <= initial.CurrentIndex {
		t.Fatalf("initial projection did not reproduce coordinate split: %+v", initial)
	}

	items := listBoxSnapshotFixtureItems()
	reordered := []expletives.ListItem{items[2], items[0], items[1]}
	if err := list.SetItems(reordered); err != nil {
		t.Fatal(err)
	}
	if details := observe("wrapped-list-retained"); details.Current != "three" || details.CurrentIndex != 0 {
		t.Fatalf("retained-current completion = %+v", details)
	}

	if err := list.Replace(reordered, "two", []string{"two"}); err != nil {
		t.Fatal(err)
	}
	if details := observe("wrapped-list-moved"); details.Current != "two" || details.CurrentIndex != 2 ||
		details.SelectedCount != 1 {
		t.Fatalf("moved-current completion = %+v", details)
	}

	if err := list.SetItems(reordered[:2]); err != nil {
		t.Fatal(err)
	}
	if details := observe("wrapped-list-removed"); details.Current != "one" || details.CurrentIndex != 1 ||
		details.SelectedCount != 0 {
		t.Fatalf("removed-current completion = %+v", details)
	}

	if err := list.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	completion := request(t, func(ctx context.Context) (Completion, error) {
		return client.Observe(ctx, "wrapped-list-hidden", nil)
	})
	if control := projectedListBoxControl(t, completion.Snapshot); control.Visible {
		t.Fatal("matching client projected hidden ListBox as visible")
	}
}

func TestCloseDrainsAcceptedProcessingCompletion(t *testing.T) {
	t.Parallel()

	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 8, Height: 3},
		Scenario: "automation.close-drain",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	handlerEntered := make(chan struct{})
	var enterOnce sync.Once

	if err := app.SetCommandHandler(func(
		ctx context.Context,
		command expletives.Command,
	) (expletives.Outcome, error) {
		if command.ID != "test.close-drain" {
			return expletives.OutcomeRejected, nil
		}
		enterOnce.Do(func() {
			close(handlerEntered)
		})
		<-ctx.Done()
		return expletives.OutcomeCancelled, ctx.Err()
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
	})

	client := dialTestClient(t, path)
	t.Cleanup(func() {
		_ = client.Close()
	})

	type requestResult struct {
		completion Completion
		err        error
	}
	result := make(chan requestResult, 1)
	requestContext := testContext(t)
	go func() {
		completion, requestErr := client.InvokeCommand(
			requestContext,
			"close-drain-1",
			"test.close-drain",
			"",
		)
		result <- requestResult{completion: completion, err: requestErr}
	}()

	select {
	case <-handlerEntered:
	case <-requestContext.Done():
		t.Fatalf("command handler did not begin: %v", requestContext.Err())
	}

	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket remains after Close(): %v", err)
	}

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("InvokeCommand() error after Close() = %v", got.err)
		}
		if got.completion.Outcome != OutcomeCancelled ||
			got.completion.Error == nil ||
			got.completion.Error.Code != "cancelled" ||
			got.completion.Snapshot == nil ||
			got.completion.Snapshot.Completion == nil ||
			got.completion.Snapshot.Completion.RequestID != "close-drain-1" {
			t.Fatalf("drained completion = %+v", got.completion)
		}
	case <-requestContext.Done():
		t.Fatalf("completion was not drained: %v", requestContext.Err())
	}

	select {
	case err := <-serveErrors:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-requestContext.Done():
		t.Fatalf("Serve() did not join drained request: %v", requestContext.Err())
	}
}

func TestDisconnectClearsHeldKeysAndAllowsReconnect(t *testing.T) {
	t.Parallel()

	app, _, _ := newTestApp(t)
	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
		select {
		case <-serveErrors:
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop during cleanup")
		}
	})

	first := dialTestClient(t, path)
	down := request(t, func(ctx context.Context) (Completion, error) {
		return first.InjectInput(ctx, "disconnect-down", KeyEvent{Kind: KeyDown, Key: "alt"})
	})
	assertHeldKey(t, down.Snapshot, "alt", true)
	after := down.FrameSequence
	if err := first.Close(); err != nil {
		t.Fatalf("first client Close() error = %v", err)
	}

	cleared, err := app.WaitSnapshot(testContext(t), after)
	if err != nil {
		t.Fatalf("WaitSnapshot() after disconnect error = %v", err)
	}
	if len(cleared.InputSources) != 0 {
		t.Errorf("held input after disconnect = %+v, want empty", cleared.InputSources)
	}

	second := dialTestClient(t, path)
	observed := request(t, func(ctx context.Context) (Completion, error) {
		return second.Observe(ctx, "reconnect-observe", nil)
	})
	if len(observed.Snapshot.InputSources) != 0 {
		t.Errorf("reconnected snapshot held input = %+v, want empty", observed.Snapshot.InputSources)
	}
	retained := request(t, func(ctx context.Context) (Completion, error) {
		return second.QueryResult(ctx, "reconnect-query", "disconnect-down")
	})
	if retained.Result == nil || retained.Result.Query == nil ||
		retained.Result.Query.Status != "completed" ||
		retained.Result.Query.Completion == nil {
		t.Fatalf("retained result after reconnect = %+v", retained.Result)
	}
	if retained.Result.Query.Completion.FrameSequence != retained.FrameSequence {
		t.Errorf(
			"retained metadata/snapshot sequences = %d/%d",
			retained.Result.Query.Completion.FrameSequence,
			retained.FrameSequence,
		)
	}
	assertHeldKey(t, retained.Snapshot, "alt", true)
	if err := second.Close(); err != nil {
		t.Fatalf("second client Close() error = %v", err)
	}
}

func TestSecondControllerIsRejected(t *testing.T) {
	t.Parallel()

	app, _, _ := newTestApp(t)
	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
		select {
		case <-serveErrors:
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop during cleanup")
		}
	})

	first := dialTestClient(t, path)
	defer first.Close()
	_, err = Dial(testContext(t), path)
	var responseError *ResponseError
	if !errors.As(err, &responseError) {
		t.Fatalf("second Dial() error = %v, want *ResponseError", err)
	}
	if responseError.Code != "controller_busy" {
		t.Errorf("second Dial() code = %q, want controller_busy", responseError.Code)
	}
}

func TestMalformedRecordDoesNotStopControllerLoop(t *testing.T) {
	t.Parallel()

	app, _, _ := newTestApp(t)
	path := filepath.Join(t.TempDir(), "automation.sock")
	server, err := NewServer(ServerOptions{
		App: app, SocketPath: path, Application: "automation-test",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	serveContext, cancelServe := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(serveContext)
	}()
	t.Cleanup(func() {
		cancelServe()
		_ = server.Close()
		select {
		case <-serveErrors:
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop during cleanup")
		}
	})

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	if _, err := reader.ReadBytes('\n'); err != nil {
		t.Fatalf("read hello error = %v", err)
	}

	malformed := `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"bad-1","unknown":true}` + "\n"
	if _, err := conn.Write([]byte(malformed)); err != nil {
		t.Fatalf("write malformed request error = %v", err)
	}
	failureLine, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read protocol error = %v", err)
	}
	var failure ProtocolError
	if err := json.Unmarshal(failureLine, &failure); err != nil {
		t.Fatalf("decode protocol error = %v", err)
	}
	if failure.Code != "invalid_request" {
		t.Errorf("protocol error code = %q, want invalid_request", failure.Code)
	}

	valid := `{"protocol":"expletives.automation","version":1,"type":"observe","request_id":"good-1"}` + "\n"
	if _, err := conn.Write([]byte(valid)); err != nil {
		t.Fatalf("write valid request error = %v", err)
	}
	if _, err := reader.ReadBytes('\n'); err != nil {
		t.Fatalf("read accepted after error = %v", err)
	}
	completionLine, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read completion after error = %v", err)
	}
	var completion Completion
	if err := json.Unmarshal(completionLine, &completion); err != nil {
		t.Fatalf("decode completion = %v", err)
	}
	if completion.RequestID != "good-1" || completion.Snapshot == nil {
		t.Errorf("completion after malformed request = %+v", completion)
	}
}

func newTestApp(t *testing.T) (*expletives.App, StyleID, StyleID) {
	t.Helper()

	initialStyle := expletives.Style{
		ID:         "fixture.root",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x10, 0x20, 0x30),
	}
	toggledStyle := expletives.Style{
		ID:         "fixture.toggled",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xE0, 0xD0, 0x20),
	}
	theme, err := expletives.NewTheme(initialStyle, toggledStyle)
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:      expletives.Size{Width: 8, Height: 3},
		Theme:     theme,
		RootStyle: initialStyle.ID,
		Scenario:  "foundation.absolute-panels",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	for _, definition := range []expletives.CommandDefinition{
		{ID: testCommandFixtureToggle, Enabled: true, Automation: true},
		{ID: testCommandScenarioReset, Enabled: true, Automation: true},
		{ID: testCommandAppQuit, Enabled: true, Automation: true},
		{ID: testCommandHumanOnly, Enabled: true, Automation: false},
	} {
		if err := app.RegisterCommand(definition); err != nil {
			t.Fatalf("RegisterCommand(%q) error = %v", definition.ID, err)
		}
	}
	targetPanel, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: testTargetKey,
		Style:         initialStyle.ID,
		Hidden:        true,
	})
	if err != nil {
		t.Fatalf("NewPanel(target) error = %v", err)
	}
	if err := app.BindChord(
		expletives.Chord{
			Key:       expletives.Key("r"),
			Modifiers: []expletives.Key{expletives.KeyControl},
		},
		expletives.CommandBinding{Command: expletives.CommandID(testCommandFixtureToggle)},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}

	var handlerMutex sync.Mutex
	toggled := false
	if err := app.SetCommandHandler(func(_ context.Context, command expletives.Command) (expletives.Outcome, error) {
		handlerMutex.Lock()
		defer handlerMutex.Unlock()
		switch string(command.ID) {
		case testCommandFixtureToggle:
			toggled = !toggled
			style := initialStyle.ID
			if toggled {
				style = toggledStyle.ID
			}
			if err := app.Root().SetStyle(style); err != nil {
				return expletives.OutcomeFailed, err
			}
			return expletives.OutcomeApplied, nil
		case testCommandScenarioReset:
			if command.Target != "" && command.Target != targetPanel.ID() {
				return expletives.OutcomeRejected, nil
			}
			toggled = false
			if err := app.Root().SetStyle(initialStyle.ID); err != nil {
				return expletives.OutcomeFailed, err
			}
			return expletives.OutcomeApplied, nil
		case testCommandAppQuit:
			return expletives.OutcomeExited, nil
		default:
			return expletives.OutcomeRejected, nil
		}
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}
	return app, StyleID(initialStyle.ID), StyleID(toggledStyle.ID)
}

func request(
	t *testing.T,
	call func(context.Context) (Completion, error),
) Completion {
	t.Helper()
	completion, err := call(testContext(t))
	if err != nil {
		t.Fatalf("automation request error = %v", err)
	}
	if completion.Snapshot == nil {
		t.Fatal("automation completion omits snapshot")
	}
	return completion
}

func dialTestClient(t *testing.T, path string) *Client {
	t.Helper()
	client, err := Dial(testContext(t), path)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	return client
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func assertHeldKey(t *testing.T, snapshot *SnapshotV1, key string, want bool) {
	t.Helper()
	found := false
	for _, source := range snapshot.InputSources {
		for _, held := range source.Held {
			if string(held) == key {
				found = true
			}
		}
	}
	if found != want {
		t.Errorf("held key %q = %t, want %t", key, found, want)
	}
}
