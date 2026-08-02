package terminal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	terminfoLegacyMagic         = 0o432
	terminfoExtendedNumberMagic = 0o1036
	maxTerminfoBytes            = 32 * 1024
	maxTerminfoNameBytes        = 128

	terminfoMaxColorsIndex         = 13
	terminfoClearScreenIndex       = 5
	terminfoCursorAddressIndex     = 10
	terminfoCursorInvisibleIndex   = 13
	terminfoCursorNormalIndex      = 16
	terminfoEnterCAModeIndex       = 28
	terminfoExitAttributeModeIndex = 39
	terminfoExitCAModeIndex        = 40
	terminfoSetAForegroundIndex    = 359
	terminfoSetABackgroundIndex    = 360
	terminfoMinimumDeclaredColors  = 8
)

type terminfoEntry struct {
	names   []string
	colors  int64
	strings []bool
}

type requiredTerminfoString struct {
	name  string
	index int
}

var requiredTerminfoStrings = []requiredTerminfoString{
	{name: "clear_screen", index: terminfoClearScreenIndex},
	{name: "cursor_address", index: terminfoCursorAddressIndex},
	{name: "cursor_invisible", index: terminfoCursorInvisibleIndex},
	{name: "cursor_normal", index: terminfoCursorNormalIndex},
	{name: "enter_ca_mode", index: terminfoEnterCAModeIndex},
	{name: "exit_attribute_mode", index: terminfoExitAttributeModeIndex},
	{name: "exit_ca_mode", index: terminfoExitCAModeIndex},
	{name: "set_a_foreground", index: terminfoSetAForegroundIndex},
	{name: "set_a_background", index: terminfoSetABackgroundIndex},
}

// validateInstalledTerminfo treats a found compiled entry as corroborating
// evidence for an already supported static profile. Absence retains the
// static contract; malformed or contradictory evidence fails closed before
// any terminal state is changed.
func validateInstalledTerminfo(term string) error {
	entry, path, found, err := probeTerminfo(term, terminfoSearchRoots())
	if err != nil {
		return fmt.Errorf("%w: TERM=%q terminfo: %v", ErrUnsupportedTerminal, term, err)
	}
	if !found {
		return nil
	}
	if err := validateTerminfoEntry(term, entry); err != nil {
		return fmt.Errorf(
			"%w: TERM=%q terminfo %q: %v",
			ErrUnsupportedTerminal,
			term,
			path,
			err,
		)
	}
	return nil
}

func terminfoSearchRoots() []string {
	if configured := os.Getenv("TERMINFO"); configured != "" {
		return []string{configured}
	}
	roots := make([]string, 0, 8)
	if home := os.Getenv("HOME"); home != "" {
		roots = append(roots, filepath.Join(home, ".terminfo"))
	}
	if configured := os.Getenv("TERMINFO_DIRS"); configured != "" {
		for _, root := range filepath.SplitList(configured) {
			if root == "" {
				roots = append(roots, "/usr/share/terminfo")
				continue
			}
			roots = append(roots, root)
		}
	}
	roots = append(
		roots,
		"/etc/terminfo",
		"/lib/terminfo",
		"/usr/share/terminfo",
	)
	seen := make(map[string]struct{}, len(roots))
	unique := roots[:0]
	for _, root := range roots {
		clean := filepath.Clean(root)
		if clean == "." || clean == string(filepath.Separator) {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}
	return unique
}

func probeTerminfo(
	term string,
	roots []string,
) (terminfoEntry, string, bool, error) {
	if !validTerminfoName(term) {
		return terminfoEntry{}, "", false, errors.New("invalid terminal name")
	}
	directories := []string{
		term[:1],
		strconv.FormatInt(int64(term[0]), 16),
	}
	for _, root := range roots {
		info, err := os.Stat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return terminfoEntry{}, root, false, err
		}
		// Hashed ncurses databases are deliberately not parsed. The existing
		// static profile remains authoritative when no directory-tree entry is
		// available.
		if !info.IsDir() {
			continue
		}
		for _, directory := range directories {
			path := filepath.Join(root, directory, term)
			data, found, err := readBoundedTerminfo(path)
			if err != nil {
				return terminfoEntry{}, path, false, err
			}
			if !found {
				continue
			}
			entry, err := parseTerminfo(data)
			if err != nil {
				return terminfoEntry{}, path, false, err
			}
			return entry, path, true, nil
		}
	}
	return terminfoEntry{}, "", false, nil
}

func validTerminfoName(term string) bool {
	if term == "" || len(term) > maxTerminfoNameBytes {
		return false
	}
	for _, value := range []byte(term) {
		if value >= 'a' && value <= 'z' ||
			value >= 'A' && value <= 'Z' ||
			value >= '0' && value <= '9' ||
			strings.ContainsRune("+._-", rune(value)) {
			continue
		}
		return false
	}
	return true
}

func readBoundedTerminfo(path string) ([]byte, bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errors.New("entry is not a regular file")
	}
	if info.Size() < 12 || info.Size() > maxTerminfoBytes {
		return nil, false, fmt.Errorf("entry size %d is outside bounds", info.Size())
	}
	data, err := io.ReadAll(io.LimitReader(file, maxTerminfoBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > maxTerminfoBytes {
		return nil, false, errors.New("entry exceeds size bound")
	}
	return data, true, nil
}

func parseTerminfo(data []byte) (terminfoEntry, error) {
	if len(data) < 12 || len(data) > maxTerminfoBytes {
		return terminfoEntry{}, errors.New("invalid compiled entry size")
	}
	magic := binary.LittleEndian.Uint16(data[0:2])
	numberWidth := 2
	switch magic {
	case terminfoLegacyMagic:
	case terminfoExtendedNumberMagic:
		numberWidth = 4
	default:
		return terminfoEntry{}, fmt.Errorf("unknown magic %#o", magic)
	}
	counts := make([]int, 5)
	for index := range counts {
		value := int(int16(binary.LittleEndian.Uint16(
			data[2+index*2 : 4+index*2],
		)))
		if value < 0 {
			return terminfoEntry{}, errors.New("negative section count")
		}
		counts[index] = value
	}
	namesSize, booleans, numbers, stringCount, tableSize :=
		counts[0], counts[1], counts[2], counts[3], counts[4]
	if namesSize == 0 || namesSize > maxTerminfoNameBytes {
		return terminfoEntry{}, errors.New("invalid names section")
	}

	offset := 12
	namesBytes, next, err := terminfoSection(data, offset, namesSize)
	if err != nil {
		return terminfoEntry{}, err
	}
	offset = next
	if namesBytes[len(namesBytes)-1] != 0 {
		return terminfoEntry{}, errors.New("unterminated names section")
	}
	names := strings.Split(string(namesBytes[:len(namesBytes)-1]), "|")
	for _, name := range names {
		if name == "" || strings.IndexByte(name, 0) >= 0 {
			return terminfoEntry{}, errors.New("invalid terminal alias")
		}
	}

	_, offset, err = terminfoSection(data, offset, booleans)
	if err != nil {
		return terminfoEntry{}, err
	}
	if offset%2 != 0 {
		offset++
		if offset > len(data) {
			return terminfoEntry{}, errors.New("missing numeric alignment byte")
		}
	}
	numberBytes, offset, err := terminfoSection(data, offset, numbers*numberWidth)
	if err != nil {
		return terminfoEntry{}, err
	}
	stringOffsets, offset, err := terminfoSection(data, offset, stringCount*2)
	if err != nil {
		return terminfoEntry{}, err
	}
	stringTable, _, err := terminfoSection(data, offset, tableSize)
	if err != nil {
		return terminfoEntry{}, err
	}

	colors := int64(-1)
	if numbers > terminfoMaxColorsIndex {
		start := terminfoMaxColorsIndex * numberWidth
		if numberWidth == 2 {
			colors = int64(int16(binary.LittleEndian.Uint16(numberBytes[start:])))
		} else {
			colors = int64(int32(binary.LittleEndian.Uint32(numberBytes[start:])))
		}
	}
	present := make([]bool, stringCount)
	hasTerminator := make([]bool, len(stringTable))
	foundTerminator := false
	for index := len(stringTable) - 1; index >= 0; index-- {
		if stringTable[index] == 0 {
			foundTerminator = true
		}
		hasTerminator[index] = foundTerminator
	}
	for index := 0; index < stringCount; index++ {
		value := int(int16(binary.LittleEndian.Uint16(stringOffsets[index*2:])))
		if value == -1 || value == -2 {
			continue
		}
		if value < 0 || value >= len(stringTable) {
			return terminfoEntry{}, errors.New("invalid string-table offset")
		}
		if !hasTerminator[value] {
			return terminfoEntry{}, errors.New("unterminated capability string")
		}
		present[index] = true
	}
	return terminfoEntry{names: names, colors: colors, strings: present}, nil
}

func terminfoSection(
	data []byte,
	offset int,
	size int,
) ([]byte, int, error) {
	if offset < 0 || size < 0 || offset > len(data) || size > len(data)-offset {
		return nil, offset, errors.New("truncated compiled entry")
	}
	return data[offset : offset+size], offset + size, nil
}

func validateTerminfoEntry(term string, entry terminfoEntry) error {
	alias := false
	for _, name := range entry.names {
		if name == term {
			alias = true
			break
		}
	}
	if !alias {
		return errors.New("terminal name is absent from entry")
	}
	if entry.colors < terminfoMinimumDeclaredColors {
		return fmt.Errorf("declares %d colors, need at least %d", entry.colors, terminfoMinimumDeclaredColors)
	}
	missing := make([]string, 0, len(requiredTerminfoStrings))
	for _, capability := range requiredTerminfoStrings {
		if capability.index >= len(entry.strings) || !entry.strings[capability.index] {
			missing = append(missing, capability.name)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing required capabilities: %s", strings.Join(missing, ", "))
	}
	return nil
}
