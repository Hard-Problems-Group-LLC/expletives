package automation

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestSnapshotProjectionPreservesJSONAndDeepCopies(t *testing.T) {
	t.Parallel()

	foreground := expletives.RGB(0x11, 0x22, 0x33)
	background := expletives.RGB(0x44, 0x55, 0x66)
	resolved := expletives.ResolvedStyle{
		Foreground: foreground,
		Background: background,
		Attributes: expletives.StyleBold | expletives.StyleUnderline,
	}
	core := expletives.Snapshot{
		Version:  1,
		Sequence: 9,
		Final:    true,
		Scenario: "snapshot.projection",
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 1, Height: 1},
			Cells: []expletives.Cell{{
				Grapheme:   "A",
				Style:      "cell.style",
				Foreground: foreground,
				Background: background,
				Attributes: expletives.StyleItalic,
				Owner:      "root",
			}},
		},
		Cursor: expletives.CursorState{
			Visible:  true,
			Position: expletives.Point{X: 0, Y: 0},
		},
		Controls: []expletives.ControlSnapshot{{
			ID:             "root",
			Key:            "root",
			Kind:           expletives.ControlRoot,
			Children:       []expletives.ControlID{"child"},
			Bounds:         expletives.Rect{Width: 1, Height: 1},
			AbsoluteBounds: expletives.Rect{Width: 1, Height: 1},
			EffectiveClip:  expletives.Rect{Width: 1, Height: 1},
			Minimum:        expletives.Size{Width: 1, Height: 1},
			Layout:         "layout-1",
			LayoutIndex:    0,
			StackIndex:     0,
			Style:          "control.style",
			ResolvedStyle:  resolved,
			Visible:        true,
			Details: expletives.ControlDetails{
				Version: expletives.ControlDetailsVersion,
				Container: &expletives.ContainerDetails{
					ClientInset: 1,
				},
				Border: &expletives.BorderDetails{
					Title:         "Border",
					Form:          expletives.BorderSingle,
					Style:         "border.style",
					ResolvedStyle: resolved,
				},
			},
		}},
		Layouts: []expletives.LayoutSnapshot{{
			ID:          "layout-1",
			Key:         "layout",
			Kind:        expletives.LayoutBox,
			Owner:       "root",
			Bounds:      expletives.Rect{Width: 1, Height: 1},
			OwnerBounds: expletives.Rect{Width: 1, Height: 1},
			Minimum:     expletives.Size{Width: 2, Height: 1},
			Border: &expletives.BorderDetails{
				Form:          expletives.BorderNone,
				Style:         "layout.border",
				ResolvedStyle: expletives.ResolvedStyle{},
			},
			LayoutIndex: 0,
			StackIndex:  0,
			Items: []expletives.LayoutItemSnapshot{{
				Kind:        "panel",
				Panel:       "root",
				Bounds:      expletives.Rect{Width: 1, Height: 1},
				Minimum:     expletives.Size{Width: 2, Height: 1},
				LayoutIndex: 0,
				StackIndex:  0,
			}},
		}},
		InputSources: []expletives.InputSourceSnapshot{{
			Source: "automation:session",
			Held:   []expletives.Key{expletives.KeyControl},
		}},
		Overflows: []expletives.OverflowSnapshot{{
			EpisodeID: "overflow-1",
			Panel:     "root",
			Layout:    "layout-1",
			Available: expletives.Size{Width: 1, Height: 1},
			Required:  expletives.Size{Width: 2, Height: 1},
			Deficit:   expletives.Size{Width: 1},
			State:     "active",
		}},
		Completion: &expletives.Completion{
			RequestID:     "request-1",
			Outcome:       expletives.OutcomeApplied,
			Command:       "scenario.reset",
			FrameSequence: 9,
			Code:          "public_code",
			Message:       "public message",
			Cause:         errors.New("local-only cause"),
		},
	}

	projected := snapshotFromCore(core)
	projectedJSON, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("Marshal(projected) error = %v", err)
	}
	if !bytes.Contains(projectedJSON, []byte(`"runs"`)) ||
		!bytes.Contains(projectedJSON, []byte(`"cells"`)) {
		t.Fatalf("direct projection omits compact or expanded frame view: %s", projectedJSON)
	}
	if bytes.Contains(projectedJSON, []byte("local-only cause")) {
		t.Fatal("projected JSON exposes local-only command cause")
	}

	core.Frame.Cells[0].Grapheme = "Z"
	core.Controls[0].Children[0] = "changed"
	core.Controls[0].Details.Container.ClientInset = 9
	core.Controls[0].Details.Border.Title = "Changed"
	core.Layouts[0].Items[0].Kind = "layout"
	core.InputSources[0].Held[0] = expletives.KeyAlt
	core.Overflows[0].State = "changed"
	core.Completion.Message = "changed"
	afterSourceMutation, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("Marshal(projected after source mutation) error = %v", err)
	}
	if !bytes.Equal(afterSourceMutation, projectedJSON) {
		t.Fatal("projected snapshot aliases core snapshot storage")
	}

	cloned := cloneSnapshot(projected)
	projected.Frame.Cells[0].Grapheme = "Q"
	projected.Controls[0].Children[0] = "mutated"
	projected.Controls[0].Details.Container.ClientInset = 7
	projected.Controls[0].Details.Border.Title = "Mutated"
	projected.Layouts[0].Items[0].Kind = "layout"
	projected.InputSources[0].Held[0] = Key(expletives.KeyShift)
	projected.Overflows[0].State = "mutated"
	projected.Completion.Message = "mutated"
	clonedJSON, err := json.Marshal(cloned)
	if err != nil {
		t.Fatalf("Marshal(cloned) error = %v", err)
	}
	if !bytes.Equal(clonedJSON, projectedJSON) {
		t.Fatal("cloned snapshot aliases projected snapshot storage")
	}
}

func TestExpandFrameRejectsDisagreeingCompactAndExpandedViews(t *testing.T) {
	t.Parallel()

	frame := IntendedFrame{
		Size:  Size{Width: 1, Height: 1},
		Runs:  []CellRun{{Count: 1, Cell: Cell{Grapheme: "A"}}},
		Cells: []Cell{{Grapheme: "B"}},
	}
	if err := expandFrame(&frame, 1, 1); err == nil {
		t.Fatal("expandFrame() accepted contradictory frame representations")
	}
}
