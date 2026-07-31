package automation

import (
	"errors"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestConvertCompletionProjectsOnlyPublicDiagnostics(t *testing.T) {
	t.Parallel()

	converted := convertCompletion(expletives.Completion{
		Outcome:       expletives.OutcomeFailed,
		FrameSequence: 7,
		Code:          "public_code",
		Message:       "safe public message",
		Cause:         errors.New("local secret cause"),
	})
	if converted.outcome != OutcomeFailed || converted.frameSequence != 7 {
		t.Fatalf("converted completion = %+v", converted)
	}
	if converted.issue == nil ||
		converted.issue.Code != "public_code" ||
		converted.issue.Message != "safe public message" {
		t.Fatalf("projected issue = %+v", converted.issue)
	}
}
