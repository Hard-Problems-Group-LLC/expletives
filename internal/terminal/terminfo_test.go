package terminal

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndValidateTerminfoLegacyAndExtendedNumbers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		magic  uint16
		colors int64
	}{
		{name: "legacy", magic: terminfoLegacyMagic, colors: 8},
		{name: "extended numbers", magic: terminfoExtendedNumberMagic, colors: 256},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entry, err := parseTerminfo(terminfoFixture(
				test.magic,
				"xterm-256color|fixture",
				test.colors,
				nil,
			))
			if err != nil {
				t.Fatalf("parseTerminfo() error = %v", err)
			}
			if entry.colors != test.colors {
				t.Errorf("colors = %d, want %d", entry.colors, test.colors)
			}
			if err := validateTerminfoEntry("xterm-256color", entry); err != nil {
				t.Fatalf("validateTerminfoEntry() error = %v", err)
			}
		})
	}
}

func TestTerminfoValidationRejectsContradictoryEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		term    string
		colors  int64
		missing map[int]bool
		want    string
	}{
		{name: "wrong alias", term: "screen-256color", colors: 256, want: "name is absent"},
		{name: "insufficient colors", term: "xterm-256color", colors: 4, want: "need at least 8"},
		{
			name: "missing cursor address", term: "xterm-256color", colors: 256,
			missing: map[int]bool{terminfoCursorAddressIndex: true},
			want:    "cursor_address",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			entry, err := parseTerminfo(terminfoFixture(
				terminfoExtendedNumberMagic,
				"xterm-256color|fixture",
				test.colors,
				test.missing,
			))
			if err != nil {
				t.Fatal(err)
			}
			err = validateTerminfoEntry(test.term, entry)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParseTerminfoRejectsMalformedAndOversizedSections(t *testing.T) {
	t.Parallel()
	valid := terminfoFixture(
		terminfoLegacyMagic,
		"xterm|fixture",
		8,
		nil,
	)
	tests := []struct {
		name string
		data []byte
	}{
		{name: "short header", data: valid[:11]},
		{name: "unknown magic", data: append([]byte(nil), valid...)},
		{name: "truncated", data: valid[:len(valid)-1]},
		{name: "oversized", data: make([]byte, maxTerminfoBytes+1)},
	}
	binary.LittleEndian.PutUint16(tests[1].data, 0xFFFF)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseTerminfo(test.data); err == nil {
				t.Fatal("parseTerminfo() accepted malformed entry")
			}
		})
	}
}

func TestProbeTerminfoUsesBoundedDirectoryEntry(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "x")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "xterm-256color")
	if err := os.WriteFile(path, terminfoFixture(
		terminfoExtendedNumberMagic,
		"xterm-256color|fixture",
		256,
		nil,
	), 0o600); err != nil {
		t.Fatal(err)
	}
	entry, gotPath, found, err := probeTerminfo(
		"xterm-256color",
		[]string{root},
	)
	if err != nil || !found || gotPath != path {
		t.Fatalf("probeTerminfo() = found %t path %q error %v", found, gotPath, err)
	}
	if err := validateTerminfoEntry("xterm-256color", entry); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, make([]byte, maxTerminfoBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = probeTerminfo("xterm-256color", []string{root})
	if err == nil || !strings.Contains(err.Error(), "outside bounds") {
		t.Fatalf("oversized probe error = %v", err)
	}

	hashed := filepath.Join(root, "terminfo.db")
	if err := os.WriteFile(hashed, []byte("not a directory tree"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, found, err = probeTerminfo("xterm-256color", []string{hashed})
	if err != nil || found {
		t.Fatalf("hashed database probe = found %t error %v", found, err)
	}
}

func TestInstalledTerminfoContradictionFailsClosed(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "x")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "xterm-256color"),
		terminfoFixture(
			terminfoExtendedNumberMagic,
			"xterm-256color|fixture",
			256,
			map[int]bool{terminfoCursorAddressIndex: true},
		),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMINFO", root)
	err := validateInstalledTerminfo("xterm-256color")
	if !errors.Is(err, ErrUnsupportedTerminal) ||
		!strings.Contains(err.Error(), "cursor_address") {
		t.Fatalf("validateInstalledTerminfo() error = %v", err)
	}
}

func TestTerminfoNameRejectsPathAndControlInjection(t *testing.T) {
	t.Parallel()
	for _, term := range []string{
		"xterm-../../etc/passwd",
		"xterm/escape",
		"xterm\nother",
		strings.Repeat("x", maxTerminfoNameBytes+1),
	} {
		if validTerminfoName(term) {
			t.Errorf("validTerminfoName(%q) = true", term)
		}
		if _, err := DetectProfile(term); !errors.Is(err, ErrUnsupportedTerminal) {
			t.Errorf("DetectProfile(%q) error = %v", term, err)
		}
	}
}

func TestInstalledSupportedEntriesWhenAvailable(t *testing.T) {
	for _, term := range []string{
		"xterm",
		"xterm-256color",
		"screen",
		"screen-256color",
		"tmux",
		"tmux-256color",
	} {
		t.Run(term, func(t *testing.T) {
			entry, _, found, err := probeTerminfo(
				term,
				[]string{"/usr/share/terminfo", "/lib/terminfo", "/etc/terminfo"},
			)
			if err != nil {
				t.Fatal(err)
			}
			if !found {
				t.Skip("system terminfo entry is not installed")
			}
			if err := validateTerminfoEntry(term, entry); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func FuzzParseTerminfo(f *testing.F) {
	f.Add(terminfoFixture(
		terminfoLegacyMagic,
		"xterm|fixture",
		8,
		nil,
	))
	f.Add(terminfoFixture(
		terminfoExtendedNumberMagic,
		"tmux-256color|fixture",
		256,
		nil,
	))
	f.Add([]byte("not terminfo"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseTerminfo(data)
	})
}

func terminfoFixture(
	magic uint16,
	names string,
	colors int64,
	missing map[int]bool,
) []byte {
	nameBytes := append([]byte(names), 0)
	numberWidth := 2
	if magic == terminfoExtendedNumberMagic {
		numberWidth = 4
	}
	const numberCount = terminfoMaxColorsIndex + 1
	const stringCount = terminfoSetABackgroundIndex + 1
	numbers := make([]byte, numberCount*numberWidth)
	for index := 0; index < numberCount; index++ {
		if numberWidth == 2 {
			binary.LittleEndian.PutUint16(numbers[index*2:], uint16(0xFFFF))
		} else {
			binary.LittleEndian.PutUint32(numbers[index*4:], uint32(0xFFFFFFFF))
		}
	}
	if numberWidth == 2 {
		binary.LittleEndian.PutUint16(
			numbers[terminfoMaxColorsIndex*2:],
			uint16(colors),
		)
	} else {
		binary.LittleEndian.PutUint32(
			numbers[terminfoMaxColorsIndex*4:],
			uint32(colors),
		)
	}
	offsets := make([]byte, stringCount*2)
	for index := 0; index < stringCount; index++ {
		binary.LittleEndian.PutUint16(offsets[index*2:], uint16(0xFFFF))
	}
	table := make([]byte, 0, len(requiredTerminfoStrings)*2)
	for _, capability := range requiredTerminfoStrings {
		if missing[capability.index] {
			continue
		}
		binary.LittleEndian.PutUint16(
			offsets[capability.index*2:],
			uint16(len(table)),
		)
		table = append(table, 'x', 0)
	}
	header := make([]byte, 12)
	binary.LittleEndian.PutUint16(header[0:], magic)
	values := []int{len(nameBytes), 0, numberCount, stringCount, len(table)}
	for index, value := range values {
		binary.LittleEndian.PutUint16(header[2+index*2:], uint16(value))
	}
	result := append(header, nameBytes...)
	if len(result)%2 != 0 {
		result = append(result, 0)
	}
	result = append(result, numbers...)
	result = append(result, offsets...)
	result = append(result, table...)
	return result
}
