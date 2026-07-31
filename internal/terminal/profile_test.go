package terminal

import (
	"errors"
	"testing"
)

func TestDetectProfile(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		want    Profile
		wantErr bool
	}{
		{name: "xterm", term: "xterm", want: ProfileXTerm},
		{name: "xterm color", term: "xterm-256color", want: ProfileXTerm},
		{name: "screen", term: "screen", want: ProfileScreen},
		{name: "screen qualified", term: "screen.xterm-256color", want: ProfileScreen},
		{name: "tmux", term: "tmux", want: ProfileTMux},
		{name: "tmux color", term: "tmux-256color", want: ProfileTMux},
		{name: "empty", term: "", wantErr: true},
		{name: "dumb", term: "dumb", wantErr: true},
		{name: "unclaimed linux console", term: "linux", wantErr: true},
		{name: "substring is not a profile", term: "not-xterm", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectProfile(tt.term)
			if tt.wantErr {
				if !errors.Is(err, ErrUnsupportedTerminal) {
					t.Fatalf("DetectProfile(%q) error = %v, want ErrUnsupportedTerminal", tt.term, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectProfile(%q) error = %v", tt.term, err)
			}
			if got != tt.want {
				t.Errorf("DetectProfile(%q) = %q, want %q", tt.term, got, tt.want)
			}
		})
	}
}

func TestGlyphModeForLocale(t *testing.T) {
	t.Parallel()
	for _, locale := range []string{"en_US.UTF-8", "C.utf8", "UTF8"} {
		if got := glyphModeForLocale(locale); got != glyphUnicode {
			t.Errorf("glyphModeForLocale(%q) = %v, want Unicode", locale, got)
		}
	}
	for _, locale := range []string{"", "C", "POSIX", "en_US.ISO-8859-1"} {
		if got := glyphModeForLocale(locale); got != glyphDEC {
			t.Errorf("glyphModeForLocale(%q) = %v, want DEC", locale, got)
		}
	}
}
