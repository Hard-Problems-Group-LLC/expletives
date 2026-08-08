package terminal

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestEncodeSnapshotFullFrameAndCursor(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 2, Height: 1},
			Cells: []expletives.Cell{
				{
					Grapheme:   "A",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0x80),
				},
				{
					Grapheme:   "B",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0x80),
				},
			},
		},
		Cursor: expletives.CursorState{
			Visible:  true,
			Position: expletives.Point{X: 1, Y: 0},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	want := "\x1b[0m\x1b[2J\x1b[H" +
		"\x1b[1;1H\x1b[97;44mAB" +
		"\x1b[0m\x1b[1;1H\x1b[1;2H\x1b[?25h"
	if string(got) != want {
		t.Errorf("EncodeSnapshot() = %q, want %q", got, want)
	}
}

func TestEncodeSnapshotDegradesNonASCIIBlackOnYellow(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 5, Height: 1},
			Cells: []expletives.Cell{
				testCell("é"),
				testCell("e\u0301"),
				testCell("─"),
				testCell("�"),
				testCell("\x1b"),
			},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	encoded := string(got)
	if !strings.Contains(encoded, "\x1b[30;103mee") ||
		!strings.Contains(encoded, "\x1b[97;40m-") ||
		!strings.Contains(encoded, "\x1b[30;103m??") {
		t.Errorf(
			"EncodeSnapshot() = %q, want highlighted text degradation with styled line-art fallback",
			encoded,
		)
	}
	if strings.Contains(encoded, "é") || strings.Contains(encoded, "─") || strings.Contains(encoded, "�") {
		t.Errorf("EncodeSnapshot() emitted a non-ASCII logical glyph: %q", encoded)
	}
}

func TestEncodeSnapshotSelectsUnicodeDECAndASCIILineArt(t *testing.T) {
	t.Parallel()
	glyphs := []string{"┌", "─", "┐", "╔", "═", "╗", "░"}
	cells := make([]expletives.Cell, len(glyphs))
	for index, glyph := range glyphs {
		cells[index] = testCell(glyph)
	}
	snapshot := expletives.Snapshot{Frame: expletives.IntendedFrame{
		Size:  expletives.Size{Width: len(cells), Height: 1},
		Cells: cells,
	}}

	unicodeFrame, err := encodeSnapshot(snapshot, glyphUnicode)
	if err != nil {
		t.Fatalf("Unicode encode error = %v", err)
	}
	if !strings.Contains(string(unicodeFrame), "┌─┐╔═╗░") {
		t.Fatalf("Unicode frame omits canonical line art: %q", unicodeFrame)
	}

	decFrame, err := encodeSnapshot(snapshot, glyphDEC)
	if err != nil {
		t.Fatalf("DEC encode error = %v", err)
	}
	encodedDEC := string(decFrame)
	if !strings.Contains(encodedDEC, "\x1b(0lqklqk\x1b(B#") {
		t.Fatalf("DEC frame omits special-graphics line art: %q", encodedDEC)
	}

	asciiFrame, err := encodeSnapshot(snapshot, glyphASCII)
	if err != nil {
		t.Fatalf("ASCII encode error = %v", err)
	}
	if !strings.Contains(string(asciiFrame), "+-++-+#") {
		t.Fatalf("ASCII frame omits structural fallback: %q", asciiFrame)
	}
}

func TestMenuGlyphsHaveStructuredDECAndASCIIFallbacks(t *testing.T) {
	t.Parallel()
	for glyph, wantASCII := range map[string]string{
		"├": "+",
		"┤": "+",
		"►": ">",
		"√": "*",
	} {
		if got, degraded, dec := physicalCharacter(glyph, glyphASCII); got != wantASCII || degraded || dec {
			t.Errorf(
				"physicalCharacter(%q, ASCII) = %q, %t, %t; want %q, false, false",
				glyph,
				got,
				degraded,
				dec,
				wantASCII,
			)
		}
	}
}

func TestEncodeSnapshotDoesNotMutateCanonicalSnapshot(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 1, Height: 1},
			Cells: []expletives.Cell{
				{
					Grapheme:   "�",
					Style:      "warning.fixture",
					Foreground: expletives.RGB(0x12, 0x34, 0x56),
					Background: expletives.RGB(0x65, 0x43, 0x21),
					Owner:      "fixture",
				},
			},
		},
	}
	before := snapshot
	before.Frame.Cells = append([]expletives.Cell(nil), snapshot.Frame.Cells...)

	if _, err := EncodeSnapshot(snapshot); err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	if !reflect.DeepEqual(snapshot, before) {
		t.Errorf("EncodeSnapshot() mutated canonical snapshot:\n got: %#v\nwant: %#v", snapshot, before)
	}
}

func TestEncodeSnapshotRetainsResolvedColorChanges(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 2, Height: 1},
			Cells: []expletives.Cell{
				{
					Grapheme:   "x",
					Foreground: expletives.RGB(0x80, 0, 0),
					Background: expletives.RGB(0, 0, 0),
				},
				{
					Grapheme:   "y",
					Foreground: expletives.RGB(0, 0xff, 0),
					Background: expletives.RGB(0, 0, 0xff),
				},
			},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	encoded := string(got)
	if !strings.Contains(encoded, "\x1b[31;40mx\x1b[92;104my") {
		t.Errorf("EncodeSnapshot() = %q, want both mapped color runs", encoded)
	}
}

func TestEncodeSnapshotKeepsDefaultDisabledButtonLabelContrasting(t *testing.T) {
	t.Parallel()
	disabled, found := expletives.DefaultTheme().Resolve("button.disabled")
	if !found {
		t.Fatal("DefaultTheme has no button.disabled style")
	}
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 1, Height: 1},
			Cells: []expletives.Cell{{
				Grapheme:   "S",
				Style:      "button.disabled",
				Foreground: disabled.Foreground,
				Background: disabled.Background,
			}},
		},
	}
	encoded, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot(disabled Button) error = %v", err)
	}
	if !strings.Contains(string(encoded), "\x1b[30;100mS") {
		t.Fatalf(
			"EncodeSnapshot(disabled Button) = %q, want black on bright-black label",
			encoded,
		)
	}
}

func TestEncodeSnapshotProjectsResolvedAttributes(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 3, Height: 1},
			Cells: []expletives.Cell{
				{
					Grapheme:   "a",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0),
					Attributes: expletives.StyleBold | expletives.StyleUnderline,
				},
				{
					Grapheme:   "b",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0),
					Attributes: expletives.StyleBold | expletives.StyleUnderline,
				},
				{
					Grapheme:   "c",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0),
				},
			},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	encoded := string(got)
	if !strings.Contains(encoded, "\x1b[0;1;4;97;40mab\x1b[0;97;40mc") {
		t.Errorf("EncodeSnapshot() = %q, want bounded attribute runs", encoded)
	}
}

func TestEncodeSnapshotDegradationClearsResolvedAttributes(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size: expletives.Size{Width: 2, Height: 1},
			Cells: []expletives.Cell{
				{
					Grapheme:   "a",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0),
					Attributes: expletives.StyleReverse,
				},
				{
					Grapheme:   "é",
					Foreground: expletives.RGB(0xff, 0xff, 0xff),
					Background: expletives.RGB(0, 0, 0),
					Attributes: expletives.StyleReverse,
				},
			},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	encoded := string(got)
	if !strings.Contains(encoded, "\x1b[0;7;97;40ma\x1b[0;30;103me") {
		t.Errorf("EncodeSnapshot() = %q, want exact black-on-yellow fallback", encoded)
	}
}

func TestEncodeSnapshotHidesOutOfBoundsCursor(t *testing.T) {
	snapshot := expletives.Snapshot{
		Frame: expletives.IntendedFrame{
			Size:  expletives.Size{Width: 1, Height: 1},
			Cells: []expletives.Cell{testCell("x")},
		},
		Cursor: expletives.CursorState{
			Visible:  true,
			Position: expletives.Point{X: 1, Y: 0},
		},
	}

	got, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot() error = %v", err)
	}
	if !strings.HasSuffix(string(got), "\x1b[?25l") {
		t.Errorf("EncodeSnapshot() = %q, want hidden cursor suffix", got)
	}
}

func TestEncodeSnapshotRejectsInvalidFrame(t *testing.T) {
	tests := []struct {
		name  string
		frame expletives.IntendedFrame
	}{
		{
			name: "negative width",
			frame: expletives.IntendedFrame{
				Size: expletives.Size{Width: -1, Height: 1},
			},
		},
		{
			name: "cell count mismatch",
			frame: expletives.IntendedFrame{
				Size: expletives.Size{Width: 2, Height: 1},
				Cells: []expletives.Cell{
					testCell("x"),
				},
			},
		},
		{
			name: "bounded cell count",
			frame: expletives.IntendedFrame{
				Size: expletives.Size{
					Width:  expletives.MaxFrameCells + 1,
					Height: 1,
				},
			},
		},
		{
			name: "unsupported attributes",
			frame: expletives.IntendedFrame{
				Size: expletives.Size{Width: 1, Height: 1},
				Cells: []expletives.Cell{
					{
						Grapheme:   "x",
						Foreground: expletives.RGB(0xff, 0xff, 0xff),
						Background: expletives.RGB(0, 0, 0),
						Attributes: 1 << 15,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeSnapshot(expletives.Snapshot{Frame: tt.frame})
			if !errors.Is(err, ErrInvalidFrame) {
				t.Fatalf("EncodeSnapshot() error = %v, want ErrInvalidFrame", err)
			}
		})
	}
}

func BenchmarkTerminalEncodeSnapshot80x24(b *testing.B) {
	benchmarkTerminalEncodeSnapshot(b, 80, 24)
}

func BenchmarkTerminalEncodeSnapshot1200x1200(b *testing.B) {
	benchmarkTerminalEncodeSnapshot(b, 1200, 1200)
}

func benchmarkTerminalEncodeSnapshot(b *testing.B, width, height int) {
	b.Helper()
	cell := expletives.Cell{
		Grapheme:   "x",
		Style:      "benchmark",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0, 0, 0),
	}
	cells := make([]expletives.Cell, width*height)
	for index := range cells {
		cells[index] = cell
	}
	snapshot := expletives.Snapshot{Frame: expletives.IntendedFrame{
		Size:  expletives.Size{Width: width, Height: height},
		Cells: cells,
	}}
	b.ReportAllocs()
	b.SetBytes(int64(len(cells)))
	b.ResetTimer()
	for range b.N {
		if _, err := EncodeSnapshot(snapshot); err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(snapshot)
}

func testCell(grapheme string) expletives.Cell {
	return expletives.Cell{
		Grapheme:   grapheme,
		Foreground: expletives.RGB(0xff, 0xff, 0xff),
		Background: expletives.RGB(0, 0, 0),
	}
}
