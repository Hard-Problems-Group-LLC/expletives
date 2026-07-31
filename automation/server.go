package automation

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

// ServerOptions configure one explicitly enabled local automation server.
type ServerOptions struct {
	// App is required and remains caller-owned; Server.Close does not stop it.
	App *expletives.App
	// SocketPath is a required absolute pathname whose immediate parent exists.
	// The operator is responsible for choosing a private, trusted directory.
	SocketPath string
	// Application is a required bounded stable identifier.
	Application string
	// Limits selects DefaultLimits when zero; version 1 accepts no other value.
	Limits Limits
}

// Server owns one Unix listener and its verified filesystem endpoint, but not
// its App. Its exported methods are safe for concurrent use; Serve may run
// once and Close is idempotent. A Server must not be copied after first use.
//
// Server provides no peer authentication or capability authorization. Its
// session ID and socket mode are not credentials.
type Server struct {
	app         *expletives.App
	path        string
	application string
	limits      Limits
	sessionID   string

	listener   *net.UnixListener
	socketInfo os.FileInfo

	mu         sync.Mutex
	controller net.Conn
	processing bool
	requests   map[string]*requestRecord
	retained   []string
	serving    bool
	stopping   bool

	closeOnce sync.Once
	closeErr  error
	wg        sync.WaitGroup
}

type requestRecord struct {
	active bool
	result retainedResult
}

// retainedResult is the single reconciliation authority for a completed
// protocol request. Compact metadata and its exact immutable snapshot enter
// and leave the bounded FIFO atomically.
type retainedResult struct {
	completion RetainedCompletion
	snapshot   SnapshotV1
}

// NewServer validates options and binds the requested Unix-socket path. The
// caller owns Server.Close after every successful call, even when Serve is
// never called. NewServer refuses any pre-existing endpoint and never removes
// one to make room.
func NewServer(options ServerOptions) (*Server, error) {
	if options.App == nil {
		return nil, errors.New("automation: nil App")
	}
	if options.SocketPath == "" {
		return nil, errors.New("automation: empty socket path")
	}
	if !filepath.IsAbs(options.SocketPath) {
		return nil, errors.New("automation: socket path must be absolute")
	}
	if len(options.SocketPath) > 107 {
		return nil, errors.New("automation: socket path exceeds Linux Unix-socket limit")
	}

	limits := options.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if limits != DefaultLimits() {
		return nil, errors.New("automation: protocol version 1 server limits are fixed")
	}
	if err := limits.validate(); err != nil {
		return nil, fmt.Errorf("automation: invalid limits: %w", err)
	}

	application := options.Application
	if !validIdentifier(application, limits.IdentifierBytes) {
		return nil, errors.New("automation: application is required and must be a valid bounded identifier")
	}

	if _, err := automationCommandInventory(options.App, limits); err != nil {
		return nil, err
	}

	sessionID, err := randomID()
	if err != nil {
		return nil, err
	}
	listener, socketInfo, err := listenUnix(options.SocketPath)
	if err != nil {
		return nil, err
	}

	server := &Server{
		app:         options.App,
		path:        options.SocketPath,
		application: application,
		limits:      limits,
		sessionID:   sessionID,
		listener:    listener,
		socketInfo:  socketInfo,
		requests:    make(map[string]*requestRecord),
	}
	return server, nil
}

// SocketPath returns the exact endpoint path owned by the server. The path is
// not a credential and remains available after Close.
func (s *Server) SocketPath() string {
	return s.path
}

// Serve accepts controllers until ctx is cancelled, Close is called, or an
// orderly shutdown request closes the server. Serve may be called once. Before
// returning, it joins all controller workers, closes the Server, and includes
// any endpoint-cleanup failure in its returned error.
func (s *Server) Serve(ctx context.Context) (serveErr error) {
	if ctx == nil {
		return errors.New("automation: nil context")
	}
	s.mu.Lock()
	if s.serving {
		s.mu.Unlock()
		return errors.New("automation: Serve may be called only once")
	}
	if s.stopping {
		s.mu.Unlock()
		return net.ErrClosed
	}
	s.serving = true
	s.mu.Unlock()
	runContext, cancelRun := context.WithCancel(ctx)
	defer func() {
		cancelRun()
		closeErr := s.Close()
		s.wg.Wait()
		serveErr = errors.Join(serveErr, closeErr)
	}()

	stop := context.AfterFunc(runContext, func() {
		_ = s.Close()
	})
	defer stop()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.isStopping() || errors.Is(err, net.ErrClosed) || runContext.Err() != nil {
				return nil
			}
			return fmt.Errorf("automation: accept controller: %w", err)
		}

		if !s.acquireController(conn) {
			_ = s.writeRecord(conn, ProtocolError{
				Header: newHeader(TypeProtocolError),
				Error: Error{
					Code:      "controller_busy",
					Message:   "protocol version 1 permits one controller",
					Retryable: true,
				},
			})
			_ = conn.Close()
			continue
		}

		s.wg.Add(1)
		go s.handleController(runContext, conn)
	}
}

// Close stops acceptance and removes only the socket object created by this
// Server. It closes an idle controller immediately, but lets an accepted
// request that is already processing finish its bounded completion write.
// Close is idempotent and does not wait for that worker; the return of Serve is
// the join point.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.stopping = true
		listener := s.listener
		controller := s.controller
		processing := s.processing
		s.mu.Unlock()

		var errs []error
		if listener != nil {
			if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		// Let an already accepted request finish its bounded completion write.
		// In particular, application teardown may begin as soon as a final
		// snapshot is published, just before this goroutine writes the exact
		// correlated completion to the controller.
		if controller != nil && !processing {
			if err := controller.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		if err := removeOwnedSocket(s.path, s.socketInfo); err != nil {
			errs = append(errs, err)
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}

func (s *Server) handleController(serveCtx context.Context, conn net.Conn) {
	defer s.wg.Done()

	source := "automation:" + s.sessionID
	defer func() {
		s.app.ClearInputSource(source)
		s.releaseController(conn)
		_ = conn.Close()
	}()

	snapshot := s.app.Snapshot()
	commands, err := automationCommandInventory(s.app, s.limits)
	if err != nil {
		_ = s.writeRecord(conn, ProtocolError{
			Header: newHeader(TypeProtocolError),
			Error: Error{
				Code:    "application_inventory_invalid",
				Message: "application command inventory cannot be advertised",
			},
		})
		return
	}
	hello := Hello{
		Header:              newHeader(TypeHello),
		Application:         s.application,
		SessionID:           s.sessionID,
		Unauthenticated:     true,
		SupportedVersions:   []int{Version},
		Scenario:            snapshot.Scenario,
		LatestFrameSequence: snapshot.Sequence,
		Final:               snapshot.Final,
		Operations:          append([]string{}, versionOneOperations[:]...),
		Commands:            commands,
		Limits:              s.limits,
	}
	if err := s.writeRecord(conn, hello); err != nil {
		return
	}

	reader := bufio.NewReaderSize(conn, lineReaderBufferBytes)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(s.limits.readTimeout())); err != nil {
			return
		}
		line, err := readLine(conn, reader, s.limits.RequestLineBytes)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF), errors.Is(err, net.ErrClosed):
				return
			case errors.Is(err, errLineTooLong):
				_ = s.writeRecord(conn, ProtocolError{
					Header: newHeader(TypeProtocolError),
					Error: Error{
						Code:    "line_too_long",
						Message: "request line exceeds configured limit",
					},
				})
				return
			case errors.Is(err, errTruncated):
				_ = s.writeRecord(conn, ProtocolError{
					Header: newHeader(TypeProtocolError),
					Error: Error{
						Code:    "truncated_record",
						Message: "request is not newline terminated",
					},
				})
				return
			default:
				return
			}
		}

		request, err := decodeRequest(line, s.limits)
		if err != nil {
			var failure *decodeError
			if !errors.As(err, &failure) {
				return
			}
			if writeErr := s.writeRecord(conn, ProtocolError{
				Header:    newHeader(TypeProtocolError),
				RequestID: failure.requestID,
				Error: Error{
					Code:      failure.code,
					Message:   failure.message,
					Retryable: failure.retryable,
				},
			}); writeErr != nil {
				return
			}
			continue
		}

		if !s.reserveRequest(request.requestID) {
			if err := s.writeRecord(conn, ProtocolError{
				Header:    newHeader(TypeProtocolError),
				RequestID: request.requestID,
				Error: Error{
					Code:      "duplicate_request_id",
					Message:   "request ID is active or retained",
					Retryable: false,
				},
			}); err != nil {
				return
			}
			continue
		}

		accepted := Accepted{
			Header:    newHeader(TypeAccepted),
			RequestID: request.requestID,
			Operation: request.operation,
		}
		// Establish the drain protection before the accepted record can become
		// visible to the controller. Close must not sever a request after the
		// controller has learned that the server accepted it.
		if !s.beginProcessing(conn) {
			return
		}
		acceptedWriteErr := s.writeRecord(conn, accepted)
		completion := s.executeSafely(serveCtx, source, request)
		s.retainCompletion(completion)
		if acceptedWriteErr != nil {
			s.setProcessing(conn, false)
			return
		}
		completionWriteErr := s.writeRecord(conn, completion)
		stopping := s.setProcessing(conn, false)
		if completionWriteErr != nil {
			return
		}
		if request.operation == TypeShutdown && completion.Outcome == OutcomeExited {
			_ = s.Close()
			return
		}
		if stopping {
			return
		}
	}
}

func automationCommandInventory(
	app *expletives.App,
	limits Limits,
) ([]string, error) {
	definitions := app.Commands()
	commands := make([]string, 0, min(len(definitions), expletives.MaxControls))
	for _, definition := range definitions {
		if !definition.Automation {
			continue
		}
		command := string(definition.ID)
		if !validIdentifier(command, limits.IdentifierBytes) {
			return nil, fmt.Errorf(
				"automation: application command %q is not a valid bounded identifier",
				command,
			)
		}
		if len(commands) == expletives.MaxControls {
			return nil, fmt.Errorf(
				"automation: command inventory exceeds control-bound limit %d",
				expletives.MaxControls,
			)
		}
		commands = append(commands, command)
	}
	if commands == nil {
		commands = []string{}
	}
	return commands, nil
}

func (s *Server) executeSafely(
	serveCtx context.Context,
	source string,
	request decodedRequest,
) (completion Completion) {
	defer func() {
		if recovered := recover(); recovered != nil {
			completion = completionFailure(
				request,
				OutcomeFailed,
				s.app.Snapshot(),
				&Error{
					Code:    "dispatch_panic",
					Message: "request dispatch panicked",
				},
			)
		}
	}()
	return s.execute(serveCtx, source, request)
}

func (s *Server) execute(serveCtx context.Context, source string, request decodedRequest) Completion {
	ctx, cancel := context.WithTimeout(serveCtx, s.limits.readTimeout())
	defer cancel()

	switch request.operation {
	case TypeObserve:
		value := request.value.(observeRequest)
		snapshot, issue := s.observe(value.FrameSequence)
		if issue != nil {
			return completionFailure(request, OutcomeRejected, snapshot, issue)
		}
		return completionSuccess(request, OutcomeApplied, snapshot, nil)

	case TypeWaitSnapshot:
		value := request.value.(waitSnapshotRequest)
		if value.TimeoutMillis > 0 {
			cancel()
			ctx, cancel = context.WithTimeout(serveCtx, time.Duration(value.TimeoutMillis)*time.Millisecond)
			defer cancel()
		}
		snapshot, err := s.app.WaitSnapshot(ctx, value.AfterSequence)
		if err != nil {
			current := s.app.Snapshot()
			issue := &Error{Code: "wait_failed", Message: boundedMessage(err.Error())}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				issue.Code = "wait_timeout"
				issue.Retryable = true
			}
			return completionFailure(request, OutcomeFailed, current, issue)
		}
		return completionSuccess(request, OutcomeApplied, snapshot, nil)

	case TypeInjectInput:
		value := request.value.(injectInputRequest)
		appCompletion, err := dispatchKey(ctx, s.app, source, request.requestID, value.Event)
		return s.fromAppCompletion(request, appCompletion, err)

	case TypeInvokeCommand:
		value := request.value.(invokeCommandRequest)
		appCompletion, err := invokeCommand(
			ctx,
			s.app,
			source,
			request.requestID,
			value.Command,
			value.TargetKey,
		)
		return s.fromAppCompletion(request, appCompletion, err)

	case TypeQueryResult:
		value := request.value.(queryResultRequest)
		result, snapshot, exact := s.query(value.TargetRequestID)
		if !exact {
			// Pending and unknown/expired targets have no retained target
			// evidence, so their query completion carries current state.
			return completionSuccess(
				request,
				OutcomeApplied,
				s.app.Snapshot(),
				&Result{Query: &result},
			)
		}
		// A completed query carries the target's exact retained snapshot and
		// matching frame sequence at top level. This returns metadata plus
		// evidence without serializing a second full snapshot in Result.
		return completionSuccessSnapshot(
			request,
			OutcomeApplied,
			snapshot,
			&Result{Query: &result},
		)

	case TypeResetInput:
		appCompletion, err := resetInput(ctx, s.app, source, request.requestID)
		return s.fromAppCompletion(request, appCompletion, err)

	case TypeShutdown:
		appCompletion, err := invokeCommand(
			ctx,
			s.app,
			source,
			request.requestID,
			"app.quit",
			"",
		)
		return s.fromAppCompletion(request, appCompletion, err)

	default:
		snapshot := s.app.Snapshot()
		return completionFailure(request, OutcomeFailed, snapshot, &Error{
			Code:    "internal_unknown_operation",
			Message: "server accepted an unknown operation",
		})
	}
}

func (s *Server) fromAppCompletion(request decodedRequest, appResult appCompletion, err error) Completion {
	if err != nil {
		snapshot := s.app.Snapshot()
		return completionFailure(request, OutcomeFailed, snapshot, &Error{
			Code:    "dispatch_failed",
			Message: "application dispatch failed",
		})
	}

	snapshot, err := s.app.SnapshotAt(appResult.frameSequence)
	if err != nil {
		snapshot = s.app.Snapshot()
		if snapshot.Sequence != appResult.frameSequence {
			return completionFailure(request, OutcomeFailed, snapshot, &Error{
				Code:    "associated_snapshot_unavailable",
				Message: "the exact request-associated snapshot is no longer available",
			})
		}
	}
	completion := completionSuccess(request, appResult.outcome, snapshot, nil)
	if appResult.issue != nil {
		completion.Error = appResult.issue
	}
	return completion
}

func completionSuccess(request decodedRequest, outcome string, snapshot expletives.Snapshot, result *Result) Completion {
	return completionSuccessSnapshot(
		request,
		outcome,
		snapshotFromCore(snapshot),
		result,
	)
}

func completionSuccessSnapshot(
	request decodedRequest,
	outcome string,
	snapshot SnapshotV1,
	result *Result,
) Completion {
	compactFrame(&snapshot.Frame, false)
	return Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     request.requestID,
		Operation:     request.operation,
		Outcome:       outcome,
		FrameSequence: snapshot.Sequence,
		Snapshot:      snapshotPointer(snapshot),
		Result:        result,
	}
}

func completionFailure(request decodedRequest, outcome string, snapshot expletives.Snapshot, issue *Error) Completion {
	completion := completionSuccess(request, outcome, snapshot, nil)
	completion.Error = issue
	return completion
}

func snapshotPointer(snapshot SnapshotV1) *SnapshotV1 {
	return &snapshot
}

func (s *Server) observe(sequence *uint64) (expletives.Snapshot, *Error) {
	current := s.app.Snapshot()
	if sequence == nil || *sequence == current.Sequence {
		return current, nil
	}
	if snapshot, err := s.app.SnapshotAt(*sequence); err == nil {
		return snapshot, nil
	}
	return current, &Error{
		Code:      "snapshot_not_retained",
		Message:   "requested snapshot is not retained",
		Retryable: false,
	}
}

func (s *Server) reserveRequest(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[requestID]; exists {
		return false
	}
	s.requests[requestID] = &requestRecord{active: true}
	return true
}

func (s *Server) retainCompletion(completion Completion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if completion.Snapshot == nil {
		// Every internal completion constructor supplies an exact snapshot.
		// If that invariant is ever broken, do not retain a misleading result.
		delete(s.requests, completion.RequestID)
		return
	}
	record := s.requests[completion.RequestID]
	if record == nil {
		record = &requestRecord{}
		s.requests[completion.RequestID] = record
	}
	record.active = false
	record.result = retainedResult{
		completion: RetainedCompletion{
			RequestID:     completion.RequestID,
			Operation:     completion.Operation,
			Outcome:       completion.Outcome,
			FrameSequence: completion.FrameSequence,
			Error:         cloneProtocolError(completion.Error),
		},
		snapshot: cloneSnapshot(*completion.Snapshot),
	}
	s.retained = append(s.retained, completion.RequestID)
	for len(s.retained) > s.limits.RetainedResults {
		evicted := s.retained[0]
		s.retained = s.retained[1:]
		if retained := s.requests[evicted]; retained != nil && !retained.active {
			delete(s.requests, evicted)
		}
	}
}

func (s *Server) query(targetRequestID string) (QueryResult, SnapshotV1, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.requests[targetRequestID]
	if record == nil {
		return QueryResult{
			TargetRequestID: targetRequestID,
			Status:          "unknown_or_expired",
		}, SnapshotV1{}, false
	}
	if record.active {
		return QueryResult{
			TargetRequestID: targetRequestID,
			Status:          "pending",
		}, SnapshotV1{}, false
	}
	completion := record.result.completion
	completion.Error = cloneProtocolError(completion.Error)
	return QueryResult{
		TargetRequestID: targetRequestID,
		Status:          "completed",
		Completion:      &completion,
	}, cloneSnapshot(record.result.snapshot), true
}

func cloneProtocolError(issue *Error) *Error {
	if issue == nil {
		return nil
	}
	cloned := *issue
	return &cloned
}

func (s *Server) acquireController(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping || s.controller != nil {
		return false
	}
	s.controller = conn
	return true
}

func (s *Server) releaseController(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.controller == conn {
		s.controller = nil
		s.processing = false
	}
}

func (s *Server) beginProcessing(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.controller != conn || s.stopping {
		return false
	}
	s.processing = true
	return true
}

func (s *Server) setProcessing(conn net.Conn, processing bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.controller == conn {
		s.processing = processing
	}
	return s.stopping
}

func (s *Server) isStopping() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopping
}

func (s *Server) writeRecord(conn net.Conn, value any) error {
	line, err := encodeLine(value, s.limits.ResponseLineBytes)
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(s.limits.writeTimeout())); err != nil {
		return err
	}
	return writeAll(conn, line)
}

func randomID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("automation: generate session ID: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}

func listenUnix(path string) (*net.UnixListener, os.FileInfo, error) {
	if _, err := os.Lstat(path); err == nil {
		return nil, nil, fmt.Errorf("automation: socket path already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("automation: inspect socket path: %w", err)
	}

	parentInfo, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, nil, fmt.Errorf("automation: inspect socket parent: %w", err)
	}
	if !parentInfo.IsDir() {
		return nil, nil, errors.New("automation: socket parent is not a direct directory")
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, nil, fmt.Errorf("automation: listen: %w", err)
	}
	listener.SetUnlinkOnClose(false)

	createdInfo, err := os.Lstat(path)
	if err != nil {
		_ = listener.Close()
		return nil, nil, fmt.Errorf("automation: record socket identity: %w", err)
	}
	if createdInfo.Mode()&os.ModeSocket == 0 {
		_ = listener.Close()
		return nil, nil, errors.New("automation: endpoint is not a Unix socket")
	}
	cleanup := func() {
		_ = listener.Close()
		_ = removeOwnedSocket(path, createdInfo)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("automation: set socket permissions: %w", err)
	}
	socketInfo, err := os.Lstat(path)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("automation: record socket identity: %w", err)
	}
	if socketInfo.Mode()&os.ModeSocket == 0 || !os.SameFile(createdInfo, socketInfo) {
		cleanup()
		return nil, nil, errors.New("automation: endpoint identity changed during setup")
	}
	return listener, socketInfo, nil
}

func removeOwnedSocket(path string, created os.FileInfo) error {
	if created == nil {
		return nil
	}
	current, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("automation: inspect endpoint during cleanup: %w", err)
	}
	if current.Mode()&os.ModeSocket == 0 || !os.SameFile(created, current) {
		return errors.New("automation: endpoint identity changed; refusing cleanup")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("automation: remove endpoint: %w", err)
	}
	return nil
}
