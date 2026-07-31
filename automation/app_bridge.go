package automation

import (
	"context"
	"errors"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

type appCompletion struct {
	outcome       string
	frameSequence uint64
	issue         *Error
}

func dispatchKey(
	ctx context.Context,
	app *expletives.App,
	source string,
	requestID string,
	event KeyEvent,
) (appCompletion, error) {
	completion, err := app.DispatchKey(
		ctx,
		source,
		requestID,
		expletives.KeyEvent{
			Kind: expletives.KeyEventKind(event.Kind),
			Key:  expletives.Key(event.Key),
		},
	)
	return convertCompletion(completion), err
}

func invokeCommand(
	ctx context.Context,
	app *expletives.App,
	source string,
	requestID string,
	command string,
	targetKey string,
) (appCompletion, error) {
	target := expletives.ControlID("")
	if targetKey != "" {
		control, found := app.ControlByAutomationKey(targetKey)
		if !found {
			return rejectedTarget(app), nil
		}
		target = control.ID()
	}
	completion, err := app.InvokeCommand(
		ctx,
		source,
		requestID,
		expletives.CommandID(command),
		target,
	)
	if errors.Is(err, expletives.ErrInvalidControl) {
		// A target can be destroyed after lookup but before invocation. Map
		// that race to the same stable public rejection as an absent key.
		return rejectedTarget(app), nil
	}
	return convertCompletion(completion), err
}

func rejectedTarget(app *expletives.App) appCompletion {
	snapshot := app.Snapshot()
	return appCompletion{
		outcome:       OutcomeRejected,
		frameSequence: snapshot.Sequence,
		issue: &Error{
			Code:    "target_not_found",
			Message: "automation target key was not found",
		},
	}
}

func resetInput(
	ctx context.Context,
	app *expletives.App,
	source string,
	requestID string,
) (appCompletion, error) {
	completion, err := app.ResetInput(ctx, source, requestID)
	return convertCompletion(completion), err
}

func convertCompletion(completion expletives.Completion) appCompletion {
	converted := appCompletion{
		outcome:       string(completion.Outcome),
		frameSequence: completion.FrameSequence,
	}
	if completion.Code != "" || completion.Message != "" {
		code := completion.Code
		if code == "" {
			code = "application_result"
		}
		converted.issue = &Error{
			Code:    code,
			Message: boundedMessage(completion.Message),
		}
	}
	return converted
}
