package terminal

import "testing"

func TestNearestANSI16ExactPalette(t *testing.T) {
	for index, color := range ansi16Palette {
		got := nearestANSI16(color.red, color.green, color.blue)
		if got != index {
			t.Errorf(
				"nearestANSI16(%d, %d, %d) = %d, want exact palette index %d",
				color.red,
				color.green,
				color.blue,
				got,
				index,
			)
		}
	}
}

func TestNearestANSI16UsesStableLowerIndexOnTie(t *testing.T) {
	// (64, 0, 0) is equally distant from black and dark red.
	if got := nearestANSI16(64, 0, 0); got != 0 {
		t.Errorf("nearestANSI16(tie) = %d, want stable lower index 0", got)
	}
}

func TestANSIColorCodes(t *testing.T) {
	tests := []struct {
		index      int
		foreground int
		background int
	}{
		{index: 0, foreground: 30, background: 40},
		{index: 7, foreground: 37, background: 47},
		{index: 8, foreground: 90, background: 100},
		{index: 15, foreground: 97, background: 107},
	}
	for _, tt := range tests {
		if got := foregroundCode(tt.index); got != tt.foreground {
			t.Errorf("foregroundCode(%d) = %d, want %d", tt.index, got, tt.foreground)
		}
		if got := backgroundCode(tt.index); got != tt.background {
			t.Errorf("backgroundCode(%d) = %d, want %d", tt.index, got, tt.background)
		}
	}
}
