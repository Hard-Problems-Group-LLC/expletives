package terminal

import (
	"errors"
	"fmt"
	"strings"
)

type glyphMode uint8

const (
	glyphASCII glyphMode = iota
	glyphDEC
	glyphUnicode
)

// Profile identifies the deliberately narrow terminal contract implemented by
// the initial dependency-free presenter.
type Profile string

const (
	ProfileXTerm  Profile = "xterm"
	ProfileScreen Profile = "screen"
	ProfileTMux   Profile = "tmux"
)

var ErrUnsupportedTerminal = errors.New("unsupported terminal")

// DetectProfile recognizes only terminal descriptions covered by the initial
// presenter. It deliberately does not infer capabilities from COLORTERM,
// emulator names, or other unverified environment values.
func DetectProfile(term string) (Profile, error) {
	switch {
	case term == "xterm" || strings.HasPrefix(term, "xterm-"):
		return ProfileXTerm, nil
	case term == "screen" || strings.HasPrefix(term, "screen-") ||
		strings.HasPrefix(term, "screen."):
		return ProfileScreen, nil
	case term == "tmux" || strings.HasPrefix(term, "tmux-") ||
		strings.HasPrefix(term, "tmux."):
		return ProfileTMux, nil
	default:
		return "", fmt.Errorf("%w: TERM=%q", ErrUnsupportedTerminal, term)
	}
}

func glyphModeForLocale(locale string) glyphMode {
	normalized := strings.ToLower(locale)
	if strings.Contains(normalized, "utf-8") ||
		strings.Contains(normalized, "utf8") {
		return glyphUnicode
	}
	return glyphDEC
}
