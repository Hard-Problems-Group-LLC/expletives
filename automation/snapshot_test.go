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
			Focused:        true,
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
				Text: &expletives.TextDetails{
					Text:                "Text",
					HorizontalAlignment: expletives.TextAlignCenter,
					VerticalAlignment:   expletives.TextAlignEnd,
					Wrap:                expletives.TextWrapWords,
					Target:              "child",
					Mnemonic:            "t",
				},
				Divider: &expletives.DividerDetails{
					Orientation: expletives.Vertical,
					Form:        expletives.BorderDouble,
					Text:        "Rule",
					Alignment:   expletives.TextAlignStart,
				},
				Action: &expletives.ActionDetails{
					Label: "Run", Command: "action.run", Enabled: true,
					Mnemonic: "r", Pressed: true, Default: true,
				},
				HotkeyBar: &expletives.HotkeyBarDetails{
					Items: []expletives.HotkeyBarItemDetails{{
						Label: "Run", Command: "action.run", Enabled: true,
						Chord: &expletives.Chord{
							Key: "r", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
					}},
				},
				MenuBar: &expletives.MenuBarDetails{
					Entries: []expletives.MenuEntryDetails{{
						Key: "menu.file", Kind: expletives.MenuItemSubmenu,
						Label: "File", Enabled: true, Mnemonic: "f",
						Placement: expletives.MenuBarPlacementEnd,
						Selected:  true, Open: true, ChildCount: 1,
					}, {
						Key: "menu.open", ParentKey: "menu.file", Depth: 1,
						Kind: expletives.MenuItemCommand, Label: "Open",
						Command: "action.open", Enabled: true, Selected: true,
						Chord: &expletives.Chord{
							Key: "o", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
					}},
					OpenPath:     []string{"menu.file"},
					SelectedPath: []string{"menu.file", "menu.open"},
				},
				StatusBar: &expletives.StatusBarDetails{
					Segments: []expletives.StatusSegmentDetails{{
						Key: "status.run", Label: "Run",
						Command: "action.run", Priority: 10, Enabled: true,
						Chord: &expletives.Chord{
							Key: "r", Modifiers: []expletives.Key{
								expletives.KeyControl,
							},
						},
						Rendered: true,
						Bounds: expletives.Rect{
							X: 2, Width: 12, Height: 1,
						},
					}},
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
	if got := projected.Controls[0].Details.MenuBar.Entries[0].Placement; got != "end" {
		t.Fatalf("projected MenuBar placement = %q, want end", got)
	}
	if got := projected.Controls[0].Details.StatusBar.Segments[0]; got.Key !=
		"status.run" || got.Chord == nil || got.Chord.Key != "r" {
		t.Fatalf("projected StatusBar segment = %#v", got)
	}

	core.Frame.Cells[0].Grapheme = "Z"
	core.Controls[0].Children[0] = "changed"
	core.Controls[0].Details.Container.ClientInset = 9
	core.Controls[0].Details.Border.Title = "Changed"
	core.Controls[0].Details.Text.Text = "Changed"
	core.Controls[0].Details.Divider.Text = "Changed"
	core.Controls[0].Details.Action.Label = "Changed"
	core.Controls[0].Details.HotkeyBar.Items[0].Label = "Changed"
	core.Controls[0].Details.HotkeyBar.Items[0].Chord.Modifiers[0] =
		expletives.KeyAlt
	core.Controls[0].Details.MenuBar.Entries[0].Label = "Changed"
	core.Controls[0].Details.MenuBar.Entries[1].Chord.Modifiers[0] =
		expletives.KeyAlt
	core.Controls[0].Details.MenuBar.OpenPath[0] = "changed"
	core.Controls[0].Details.MenuBar.SelectedPath[0] = "changed"
	core.Controls[0].Details.StatusBar.Segments[0].Label = "Changed"
	core.Controls[0].Details.StatusBar.Segments[0].Chord.Modifiers[0] =
		expletives.KeyAlt
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
	projected.Controls[0].Details.Text.Text = "Mutated"
	projected.Controls[0].Details.Divider.Text = "Mutated"
	projected.Controls[0].Details.Action.Label = "Mutated"
	projected.Controls[0].Details.HotkeyBar.Items[0].Label = "Mutated"
	projected.Controls[0].Details.HotkeyBar.Items[0].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
	projected.Controls[0].Details.MenuBar.Entries[0].Label = "Mutated"
	projected.Controls[0].Details.MenuBar.Entries[1].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
	projected.Controls[0].Details.MenuBar.OpenPath[0] = "mutated"
	projected.Controls[0].Details.MenuBar.SelectedPath[0] = "mutated"
	projected.Controls[0].Details.StatusBar.Segments[0].Label = "Mutated"
	projected.Controls[0].Details.StatusBar.Segments[0].Chord.Modifiers[0] =
		Key(expletives.KeyShift)
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
