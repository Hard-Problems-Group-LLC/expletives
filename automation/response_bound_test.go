package automation

import (
	"encoding/json"
	"math"
	"runtime"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestMaximumBoundedCompletionFitsResponseLine(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	completion := maximumElementCompletion(limits)

	validationFixture := maximumValidElementCompletion(limits)
	if err := validateCompletion(validationFixture, limits); err != nil {
		t.Fatalf("maximum element fixture is not protocol-valid: %v", err)
	}

	maximumBytes := maximumCompletionJSONBytes(t, completion, limits)
	t.Logf("maximum bounded completion bytes: %d", maximumBytes)
	if maximumBytes > limits.ResponseLineBytes {
		t.Fatalf(
			"maximum bounded completion needs %d JSON bytes; response limit is %d",
			maximumBytes,
			limits.ResponseLineBytes,
		)
	}
	retainedBytes := maximumBytes * limits.RetainedResults
	if retainedBytes > maxRetainedResponseBytes {
		t.Fatalf(
			"%d retained maximum completions need %d JSON bytes; aggregate budget is %d",
			limits.RetainedResults,
			retainedBytes,
			maxRetainedResponseBytes,
		)
	}
	t.Logf(
		"maximum bounded completion: %d JSON bytes under %d-byte response limit; "+
			"%d retained records: %d bytes under %d-byte aggregate budget",
		maximumBytes,
		limits.ResponseLineBytes,
		limits.RetainedResults,
		retainedBytes,
		maxRetainedResponseBytes,
	)
}

func TestReasonableLargeDesktopFrameCompactsAndDecodes(t *testing.T) {
	const dimension = 1200
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: dimension, Height: dimension},
	})
	if err != nil {
		t.Fatalf("NewApp(1200x1200) error = %v", err)
	}
	core := app.Snapshot()
	if len(core.Frame.Cells) != dimension*dimension {
		t.Fatalf("core cell count = %d", len(core.Frame.Cells))
	}
	projected := snapshotFromCore(core)
	compactFrame(&projected.Frame, false)
	core = expletives.Snapshot{}
	app = nil
	runtime.GC()
	if got := len(projected.Frame.Runs); got != 1 {
		t.Fatalf("uniform 1200x1200 frame run count = %d, want 1", got)
	}
	completion := Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     "large-frame",
		Operation:     TypeObserve,
		Outcome:       OutcomeApplied,
		FrameSequence: projected.Sequence,
		Snapshot:      &projected,
	}
	line, err := encodeLine(completion, DefaultLimits().ResponseLineBytes)
	if err != nil {
		t.Fatalf("encode compact 1200x1200 completion: %v", err)
	}
	var decoded Completion
	if err := json.Unmarshal(line, &decoded); err != nil {
		t.Fatalf("decode compact completion: %v", err)
	}
	if err := validateCompletion(decoded, DefaultLimits()); err != nil {
		t.Fatalf("validate compact completion: %v", err)
	}
	if got := len(decoded.Snapshot.Frame.Cells); got != dimension*dimension {
		t.Fatalf("decoded cell count = %d, want %d", got, dimension*dimension)
	}
}

func TestSnapshotRejectsBorderTitleBeyondBound(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	completion := maximumValidElementCompletion(limits)
	completion.Snapshot.Controls[0].Details.Border.Title = strings.Repeat(
		"x",
		maxBorderTitleBytes+1,
	)

	if err := validateCompletion(completion, limits); err == nil {
		t.Fatal("validateCompletion() error = nil for overlong border title")
	}
}

func TestSnapshotRejectsInvalidCanonicalText(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	tests := []struct {
		name   string
		mutate func(*Completion)
	}{
		{
			name: "noncanonical cell",
			mutate: func(completion *Completion) {
				completion.Snapshot.Frame.Cells[0].Grapheme = "\x00"
			},
		},
		{
			name: "title cell count",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title =
					strings.Repeat("x", maxBorderTitleCells+1)
			},
		},
		{
			name: "title cell bytes",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title =
					"x" + strings.Repeat("\u0301", maxCellGraphemeBytes/2)
			},
		},
		{
			name: "noncanonical title",
			mutate: func(completion *Completion) {
				completion.Snapshot.Controls[0].Details.Border.Title = "\x00"
			},
		},
		{
			name: "snapshot completion message NUL",
			mutate: func(completion *Completion) {
				completion.Snapshot.Completion.Message = "invalid\x00message"
			},
		},
		{
			name: "top-level error message NUL",
			mutate: func(completion *Completion) {
				completion.Error.Message = "invalid\x00message"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			completion := maximumValidElementCompletion(limits)
			test.mutate(&completion)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() error = nil for invalid text")
			}
		})
	}
}

func TestSnapshotRejectsInvalidDisplayControlDetails(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	textTests := []struct {
		name   string
		mutate func(*TextDetails)
	}{
		{
			name: "text bytes",
			mutate: func(details *TextDetails) {
				details.Text = strings.Repeat("x", maxDisplayTextBytes+1)
			},
		},
		{
			name: "wide noncanonical text",
			mutate: func(details *TextDetails) {
				details.Text = "界"
			},
		},
		{
			name: "label newline",
			mutate: func(details *TextDetails) {
				details.Text = "first\nsecond"
			},
		},
		{
			name: "alignment",
			mutate: func(details *TextDetails) {
				details.HorizontalAlignment = "middle"
			},
		},
		{
			name: "wrap",
			mutate: func(details *TextDetails) {
				details.Wrap = "words"
			},
		},
		{
			name: "targetless mnemonic",
			mutate: func(details *TextDetails) {
				details.Target = ""
			},
		},
	}
	for _, test := range textTests {
		test := test
		t.Run("text/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := maximumValidTextCompletion(limits)
			test.mutate(completion.Snapshot.Controls[0].Details.Text)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid TextDetails")
			}
		})
	}

	dividerTests := []struct {
		name   string
		mutate func(*DividerDetails)
	}{
		{
			name: "orientation",
			mutate: func(details *DividerDetails) {
				details.Orientation = 99
			},
		},
		{
			name: "form",
			mutate: func(details *DividerDetails) {
				details.Form = "ornate"
			},
		},
		{
			name: "newline",
			mutate: func(details *DividerDetails) {
				details.Text = "bad\nrule"
			},
		},
		{
			name: "alignment",
			mutate: func(details *DividerDetails) {
				details.Alignment = "middle"
			},
		},
	}
	for _, test := range dividerTests {
		test := test
		t.Run("divider/"+test.name, func(t *testing.T) {
			t.Parallel()
			completion := maximumValidElementCompletion(limits)
			control := &completion.Snapshot.Controls[0]
			control.Kind = "rule"
			control.Details = ControlDetails{
				Version: 1,
				Divider: &DividerDetails{
					Orientation: 0,
					Form:        "single",
					Text:        "rule",
					Alignment:   "start",
				},
			}
			test.mutate(control.Details.Divider)
			if err := validateCompletion(completion, limits); err == nil {
				t.Fatal("validateCompletion() accepted invalid DividerDetails")
			}
		})
	}
}

func TestSnapshotRejectsAggregateChildReferencesBeyondBound(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	limits.Controls = 1
	completion := maximumValidElementCompletion(limits)
	completion.Snapshot.Controls[0].Children = append(
		completion.Snapshot.Controls[0].Children,
		ControlID("second"),
	)

	if err := validateCompletion(completion, limits); err == nil {
		t.Fatal("validateCompletion() error = nil for excessive child references")
	}
}

// maximumCompletionJSONBytes computes the exact encoded size of a completion
// containing the maximum number of copies of each bounded element. The run
// and title strings deliberately use the maximum six-byte JSON expansion for
// every permitted input byte. That is a conservative overestimate for
// canonical one-cell text, and therefore remains an upper bound.
// Every repeated array already has one element in completion, so each
// additional element contributes one comma plus its independently marshaled
// size. The aggregate child-reference validation permits at most one such
// maximal one-child control per control slot.
func maximumCompletionJSONBytes(
	t *testing.T,
	completion Completion,
	limits Limits,
) int {
	t.Helper()

	wireCompletion := completion
	wireSnapshot := cloneSnapshot(*completion.Snapshot)
	compactFrame(&wireSnapshot.Frame, false)
	wireCompletion.Snapshot = &wireSnapshot
	base := mustMarshal(t, wireCompletion)
	run := mustMarshal(t, completion.Snapshot.Frame.Runs[0])
	controlValue := completion.Snapshot.Controls[0]
	control := mustMarshal(t, controlValue)
	borderControl := controlValue
	borderControl.Details = ControlDetails{
		Version: 1,
		Container: &ContainerDetails{
			ClientInset: controlValue.Bounds.X,
		},
		Border: &BorderDetails{
			Title:         strings.Repeat("\x00", maxBorderTitleBytes),
			Form:          "single",
			Style:         controlValue.Style,
			ResolvedStyle: controlValue.ResolvedStyle,
		},
	}
	dividerControl := controlValue
	dividerControl.Details = ControlDetails{
		Version: 1,
		Divider: &DividerDetails{
			Orientation: Orientation(math.MaxUint8),
			Form:        string(controlValue.ID),
			Text:        strings.Repeat("\x00", maxDisplayTextBytes),
			Alignment:   TextAlignment(controlValue.ID),
		},
	}
	for _, candidate := range [][]byte{
		mustMarshal(t, borderControl),
		mustMarshal(t, dividerControl),
	} {
		if len(candidate) > len(control) {
			control = candidate
		}
	}
	source := mustMarshal(t, completion.Snapshot.InputSources[0])
	overflow := mustMarshal(t, completion.Snapshot.Overflows[0])
	layout := mustMarshal(t, completion.Snapshot.Layouts[0])
	layoutItem := mustMarshal(t, completion.Snapshot.Layouts[0].Items[0])
	t.Logf(
		"bound elements: base=%d run=%d control=%d source=%d overflow=%d layout=%d layout_item=%d",
		len(base), len(run), len(control), len(source), len(overflow), len(layout), len(layoutItem),
	)

	return len(base) +
		(limits.FrameRuns-1)*(len(run)+1) +
		(limits.Controls-1)*(len(control)+1) +
		(limits.Controls-1)*(len(source)+1) +
		(limits.Controls-1)*(len(overflow)+1) +
		(limits.Layouts-1)*(len(layout)+1) +
		(limits.LayoutItems-limits.Layouts)*(len(layoutItem)+1)
}

func maximumElementCompletion(limits Limits) Completion {
	identifier := strings.Repeat("x", limits.IdentifierBytes)
	requestID := strings.Repeat("r", limits.RequestIDBytes)
	escapedCell := strings.Repeat("\x00", maxCellGraphemeBytes)
	escapedDisplayText := strings.Repeat("\x00", maxDisplayTextBytes)
	escapedError := strings.Repeat("<", maxErrorMessageBytes)
	escapedMessage := strings.Repeat("<", maxSnapshotMessageBytes)
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	sequence := uint64(math.MaxUint64)
	resolved := ResolvedStyle{
		Foreground: "#FFFFFF",
		Background: "#FFFFFF",
		Attributes: 31,
	}

	snapshot := SnapshotV1{
		Version:  1,
		Sequence: sequence,
		Final:    false,
		Scenario: identifier,
		Frame: IntendedFrame{
			Size: Size{
				Width:  limits.FrameWidth,
				Height: limits.FrameHeight,
			},
			Cells: []Cell{{
				Grapheme:   escapedCell,
				Style:      StyleID(identifier),
				Foreground: "#FFFFFF",
				Background: "#FFFFFF",
				Attributes: 31,
				Owner:      ControlID(identifier),
			}},
		},
		Cursor: CursorState{
			Visible: false,
			Position: Point{
				X: minInt,
				Y: minInt,
			},
		},
		Controls: []ControlSnapshot{{
			ID:       ControlID(identifier),
			Key:      identifier,
			Kind:     ControlKind(identifier),
			Parent:   ControlID(identifier),
			Children: []ControlID{ControlID(identifier)},
			Bounds: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			AbsoluteBounds: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			EffectiveClip: Rect{
				X:      minInt,
				Y:      minInt,
				Width:  maxInt,
				Height: maxInt,
			},
			Minimum: Size{
				Width:  maxInt,
				Height: maxInt,
			},
			Layout:        LayoutID(identifier),
			LayoutIndex:   maxInt,
			StackIndex:    maxInt,
			Style:         StyleID(identifier),
			ResolvedStyle: resolved,
			Visible:       false,
			Details: ControlDetails{
				Version: 1,
				Text: &TextDetails{
					Text:                escapedDisplayText,
					HorizontalAlignment: TextAlignment(identifier),
					VerticalAlignment:   TextAlignment(identifier),
					Wrap:                TextWrap(identifier),
					Target:              ControlID(identifier),
					Mnemonic:            Key(identifier),
				},
			},
		}},
		Layouts: []LayoutSnapshot{{
			ID:          LayoutID(identifier),
			Key:         identifier,
			Kind:        LayoutKind(identifier),
			Owner:       ControlID(identifier),
			Parent:      LayoutID(identifier),
			Bounds:      Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
			OwnerBounds: Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
			Minimum:     Size{Width: maxInt, Height: maxInt},
			Border: &BorderDetails{
				Form:          "single",
				Style:         StyleID(identifier),
				ResolvedStyle: resolved,
			},
			LayoutIndex: maxInt,
			StackIndex:  maxInt,
			Items: []LayoutItemSnapshot{{
				Kind:        "panel",
				Panel:       ControlID(identifier),
				Bounds:      Rect{X: minInt, Y: minInt, Width: maxInt, Height: maxInt},
				Minimum:     Size{Width: maxInt, Height: maxInt},
				LayoutIndex: maxInt,
				StackIndex:  maxInt,
			}},
		}},
		InputSources: []InputSourceSnapshot{{
			Source: identifier,
			Held: []Key{
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
				"backspace",
			},
		}},
		Overflows: []OverflowSnapshot{{
			EpisodeID: identifier,
			Panel:     ControlID(identifier),
			Layout:    LayoutID(identifier),
			Available: Size{Width: maxInt, Height: maxInt},
			Required:  Size{Width: maxInt, Height: maxInt},
			Deficit:   Size{Width: maxInt, Height: maxInt},
			State:     identifier,
		}},
		Completion: &SnapshotCompletion{
			RequestID:     requestID,
			Outcome:       OutcomeInterrupted,
			Command:       identifier,
			FrameSequence: sequence,
			Code:          identifier,
			Message:       escapedMessage,
		},
	}

	compactFrame(&snapshot.Frame, true)
	return Completion{
		Header:        newHeader(TypeCompletion),
		RequestID:     requestID,
		Operation:     TypeQueryResult,
		Outcome:       OutcomeInterrupted,
		FrameSequence: sequence,
		Snapshot:      &snapshot,
		Error: &Error{
			Code:      identifier,
			Message:   escapedError,
			Retryable: false,
		},
		Result: &Result{
			Query: &QueryResult{
				TargetRequestID: requestID,
				Status:          "completed",
				Completion: &RetainedCompletion{
					RequestID:     requestID,
					Operation:     identifier,
					Outcome:       OutcomeInterrupted,
					FrameSequence: sequence,
					Error: &Error{
						Code:      identifier,
						Message:   escapedError,
						Retryable: false,
					},
				},
			},
		},
	}
}

func maximumValidElementCompletion(limits Limits) Completion {
	completion := maximumElementCompletion(limits)
	completion.Snapshot = snapshotPointer(cloneSnapshot(*completion.Snapshot))
	completion.Snapshot.Frame.Size = Size{Width: 1, Height: 1}
	completion.Snapshot.Frame.Cells[0].Grapheme = "<"
	compactFrame(&completion.Snapshot.Frame, true)
	completion.Snapshot.Controls[0].Kind = "frame"
	completion.Snapshot.Controls[0].Details = ControlDetails{
		Version:   1,
		Container: &ContainerDetails{ClientInset: 1},
		Border: &BorderDetails{
			Title: maximumCanonicalTitle(),
			Form:  "single",
			Style: completion.Snapshot.Controls[0].Style,
			ResolvedStyle: completion.Snapshot.Controls[0].
				ResolvedStyle,
		},
	}
	completion.Snapshot.Controls[0].LayoutIndex = limits.LayoutItems - 1
	completion.Snapshot.Controls[0].StackIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].LayoutIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].StackIndex = limits.LayoutItems - 1
	completion.Snapshot.Layouts[0].Items[0].LayoutIndex = 0
	completion.Snapshot.Layouts[0].Items[0].StackIndex = 0
	return completion
}

func maximumValidTextCompletion(limits Limits) Completion {
	completion := maximumValidElementCompletion(limits)
	control := &completion.Snapshot.Controls[0]
	control.Kind = "label"
	control.Details = ControlDetails{
		Version: 1,
		Text: &TextDetails{
			Text:                maximumCanonicalDisplayText(),
			HorizontalAlignment: "center",
			VerticalAlignment:   "end",
			Wrap:                "none",
			Target:              control.ID,
			Mnemonic:            "x",
		},
	}
	return completion
}

func maximumCanonicalTitle() string {
	return strings.Repeat("<", maxBorderTitleCells) +
		strings.Repeat("\u0301", (maxBorderTitleBytes-maxBorderTitleCells)/2)
}

func maximumCanonicalDisplayText() string {
	return strings.Repeat("<", maxDisplayTextCells)
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}
