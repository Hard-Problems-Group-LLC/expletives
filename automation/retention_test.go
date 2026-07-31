package automation

import (
	"context"
	"fmt"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestQueryWithoutRetainedEvidenceUsesCurrentSnapshot(t *testing.T) {
	t.Parallel()

	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 2, Height: 1},
		Scenario: "automation.query-current",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	server := &Server{
		app:      app,
		limits:   DefaultLimits(),
		requests: make(map[string]*requestRecord),
	}
	if !server.reserveRequest("pending-target") {
		t.Fatal("reserveRequest(pending-target) = false")
	}
	current := app.Snapshot()
	for _, test := range []struct {
		name   string
		target string
		status string
	}{
		{name: "pending", target: "pending-target", status: "pending"},
		{name: "unknown", target: "unknown-target", status: "unknown_or_expired"},
	} {
		t.Run(test.name, func(t *testing.T) {
			completion := server.execute(
				context.Background(),
				"automation:test",
				decodedRequest{
					operation: TypeQueryResult,
					requestID: "query-" + test.name,
					value: queryResultRequest{
						RequestID:       "query-" + test.name,
						TargetRequestID: test.target,
					},
				},
			)
			if completion.FrameSequence != current.Sequence ||
				completion.Snapshot == nil ||
				completion.Snapshot.Sequence != current.Sequence {
				t.Errorf(
					"query frame/snapshot sequence = %d/%v, want current %d",
					completion.FrameSequence,
					completion.Snapshot,
					current.Sequence,
				)
			}
			if completion.Result == nil ||
				completion.Result.Query == nil ||
				completion.Result.Query.Status != test.status {
				t.Errorf("query result = %+v, want status %q", completion.Result, test.status)
			}
		})
	}
}

func TestResultAndSnapshotRetentionExpireAtomically(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	limits.RetainedResults = 3
	server := &Server{
		limits:   limits,
		requests: make(map[string]*requestRecord),
	}
	for index := 0; index < 5; index++ {
		requestID := fmt.Sprintf("request-%d", index)
		if !server.reserveRequest(requestID) {
			t.Fatalf("reserveRequest(%q) = false", requestID)
		}
		snapshot := SnapshotV1{
			Version:  1,
			Sequence: uint64(index + 1),
		}
		server.retainCompletion(Completion{
			RequestID:     requestID,
			Operation:     TypeObserve,
			Outcome:       OutcomeApplied,
			FrameSequence: snapshot.Sequence,
			Snapshot:      &snapshot,
		})
	}

	if got := len(server.requests); got != 3 {
		t.Fatalf("retained request count = %d, want 3", got)
	}
	for _, expired := range []string{"request-0", "request-1"} {
		result, _, exact := server.query(expired)
		if result.Status != "unknown_or_expired" || exact {
			t.Errorf("query(%q) status = %q, want unknown_or_expired", expired, result.Status)
		}
	}
	for index, retained := range []string{"request-2", "request-3", "request-4"} {
		result, snapshot, exact := server.query(retained)
		if result.Status != "completed" || !exact {
			t.Errorf("query(%q) status = %q, want completed", retained, result.Status)
		}
		wantSequence := uint64(index + 3)
		if result.Completion == nil ||
			result.Completion.FrameSequence != wantSequence ||
			snapshot.Sequence != wantSequence {
			t.Errorf(
				"query(%q) metadata/snapshot sequences = %+v/%d, want %d",
				retained,
				result.Completion,
				snapshot.Sequence,
				wantSequence,
			)
		}
	}
}

func TestActiveCompletionIsNeverEvicted(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	limits.RetainedResults = 1
	server := &Server{
		limits:   limits,
		requests: make(map[string]*requestRecord),
	}
	if !server.reserveRequest("active") {
		t.Fatal("reserveRequest(active) = false")
	}
	for _, requestID := range []string{"complete-1", "complete-2"} {
		if !server.reserveRequest(requestID) {
			t.Fatalf("reserveRequest(%q) = false", requestID)
		}
		snapshot := SnapshotV1{Version: 1, Sequence: 1}
		server.retainCompletion(Completion{
			RequestID:     requestID,
			Operation:     TypeObserve,
			Outcome:       OutcomeApplied,
			FrameSequence: snapshot.Sequence,
			Snapshot:      &snapshot,
		})
	}
	result, _, exact := server.query("active")
	if result.Status != "pending" || exact {
		t.Errorf("active status = %q, want pending", result.Status)
	}
}
