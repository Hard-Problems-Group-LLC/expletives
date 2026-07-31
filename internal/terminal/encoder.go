package terminal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

var ErrInvalidFrame = errors.New("invalid intended frame")

// EncodeSnapshot maps one immutable intended frame to a correctness-first
// xterm-family full-frame update. It never changes canonical snapshot data.
func EncodeSnapshot(snapshot expletives.Snapshot) ([]byte, error) {
	return encodeSnapshot(snapshot, glyphASCII)
}

func encodeSnapshot(snapshot expletives.Snapshot, glyphs glyphMode) ([]byte, error) {
	frame := snapshot.Frame
	width := frame.Size.Width
	height := frame.Size.Height
	if width < 0 || height < 0 {
		return nil, fmt.Errorf(
			"%w: negative size %dx%d",
			ErrInvalidFrame,
			width,
			height,
		)
	}
	if height != 0 && width > int(^uint(0)>>1)/height {
		return nil, fmt.Errorf("%w: cell count overflows int", ErrInvalidFrame)
	}
	cellCount := width * height
	if cellCount > expletives.MaxFrameCells {
		return nil, fmt.Errorf(
			"%w: %d cells exceeds limit %d",
			ErrInvalidFrame,
			cellCount,
			expletives.MaxFrameCells,
		)
	}
	if len(frame.Cells) != cellCount {
		return nil, fmt.Errorf(
			"%w: size %dx%d requires %d cells, got %d",
			ErrInvalidFrame,
			width,
			height,
			cellCount,
			len(frame.Cells),
		)
	}
	const supportedAttributes = expletives.StyleBold |
		expletives.StyleDim |
		expletives.StyleItalic |
		expletives.StyleUnderline |
		expletives.StyleReverse
	for index, cell := range frame.Cells {
		if cell.Attributes&^supportedAttributes != 0 {
			return nil, fmt.Errorf(
				"%w: cell %d has unsupported style attributes %#x",
				ErrInvalidFrame,
				index,
				cell.Attributes,
			)
		}
	}

	// Clearing is part of the reference full-frame path. Absolute addressing
	// avoids newline, scrolling-region, and output-postprocessing semantics.
	output := make([]byte, 0, 32+cellCount*4)
	output = append(output, "\x1b[0m\x1b[2J\x1b[H"...)
	lastForeground := -1
	lastBackground := -1
	var lastAttributes expletives.StyleAttributes
	decGraphics := false

	for y := 0; y < height; y++ {
		output = appendCursorPosition(output, 0, y)
		for x := 0; x < width; x++ {
			cell := frame.Cells[y*width+x]
			character, degraded, useDEC := physicalCharacter(
				cell.Grapheme,
				glyphs,
			)

			foreground := nearestANSI16(
				cell.Foreground.R,
				cell.Foreground.G,
				cell.Foreground.B,
			)
			background := nearestANSI16(
				cell.Background.R,
				cell.Background.G,
				cell.Background.B,
			)
			if degraded {
				// The basic-terminal degradation contract is black on bright
				// yellow. Snapshot colors remain unchanged. Physical
				// attributes are cleared so reverse or dim cannot obscure
				// the required warning presentation.
				foreground = 0
				background = 11
			}
			attributes := cell.Attributes
			if degraded {
				attributes = 0
			}
			if attributes != lastAttributes {
				output = appendFullSGR(
					output,
					attributes,
					foreground,
					background,
				)
				lastAttributes = attributes
				lastForeground = foreground
				lastBackground = background
			} else if foreground != lastForeground || background != lastBackground {
				output = appendSGR(output, foreground, background)
				lastForeground = foreground
				lastBackground = background
			}
			if useDEC != decGraphics {
				if useDEC {
					output = append(output, "\x1b(0"...)
				} else {
					output = append(output, "\x1b(B"...)
				}
				decGraphics = useDEC
			}
			output = append(output, character...)
		}
	}

	if decGraphics {
		output = append(output, "\x1b(B"...)
	}
	output = append(output, "\x1b[0m"...)
	// An explicit cursor move clears xterm's delayed-wrap state after a write
	// to the lower-right cell without changing the user's saved wrap mode.
	output = appendCursorPosition(output, 0, 0)
	if snapshot.Cursor.Visible &&
		snapshot.Cursor.Position.X >= 0 &&
		snapshot.Cursor.Position.Y >= 0 &&
		snapshot.Cursor.Position.X < width &&
		snapshot.Cursor.Position.Y < height {
		output = appendCursorPosition(
			output,
			snapshot.Cursor.Position.X,
			snapshot.Cursor.Position.Y,
		)
		output = append(output, "\x1b[?25h"...)
	} else {
		output = append(output, "\x1b[?25l"...)
	}
	return output, nil
}

func appendFullSGR(
	output []byte,
	attributes expletives.StyleAttributes,
	foreground,
	background int,
) []byte {
	output = append(output, "\x1b[0"...)
	codes := []struct {
		attribute expletives.StyleAttributes
		code      byte
	}{
		{expletives.StyleBold, '1'},
		{expletives.StyleDim, '2'},
		{expletives.StyleItalic, '3'},
		{expletives.StyleUnderline, '4'},
		{expletives.StyleReverse, '7'},
	}
	for _, candidate := range codes {
		if attributes&candidate.attribute != 0 {
			output = append(output, ';', candidate.code)
		}
	}
	output = append(output, ';')
	output = strconv.AppendInt(output, int64(foregroundCode(foreground)), 10)
	output = append(output, ';')
	output = strconv.AppendInt(output, int64(backgroundCode(background)), 10)
	output = append(output, 'm')
	return output
}

func appendCursorPosition(output []byte, x, y int) []byte {
	output = append(output, "\x1b["...)
	output = strconv.AppendInt(output, int64(y+1), 10)
	output = append(output, ';')
	output = strconv.AppendInt(output, int64(x+1), 10)
	output = append(output, 'H')
	return output
}

func appendSGR(output []byte, foreground, background int) []byte {
	output = append(output, "\x1b["...)
	output = strconv.AppendInt(output, int64(foregroundCode(foreground)), 10)
	output = append(output, ';')
	output = strconv.AppendInt(output, int64(backgroundCode(background)), 10)
	output = append(output, 'm')
	return output
}

func physicalCharacter(
	grapheme string,
	mode glyphMode,
) (character string, degraded, decGraphics bool) {
	if len(grapheme) == 1 && grapheme[0] >= 0x20 && grapheme[0] <= 0x7e {
		return grapheme, false, false
	}
	if mode == glyphUnicode && grapheme != "" {
		return grapheme, false, false
	}

	runes := []rune(grapheme)
	if len(runes) == 0 {
		return "?", true, false
	}
	if character, dec, ok := lineArtFallback(runes[0], mode); ok {
		return string(character), false, dec
	}
	if runes[0] >= 0x20 && runes[0] <= 0x7e && onlyCombining(runes[1:]) {
		return string(runes[0]), true, false
	}
	if approximation, ok := approximateRune(runes[0]); ok {
		return string(approximation), true, false
	}
	return "?", true, false
}

func lineArtFallback(value rune, mode glyphMode) (byte, bool, bool) {
	var ascii, dec byte
	switch value {
	case '─', '━', '═':
		ascii, dec = '-', 'q'
	case '│', '┃', '║':
		ascii, dec = '|', 'x'
	case '┌', '╔':
		ascii, dec = '+', 'l'
	case '┐', '╗':
		ascii, dec = '+', 'k'
	case '└', '╚':
		ascii, dec = '+', 'm'
	case '┘', '╝':
		ascii, dec = '+', 'j'
	case '░', '▒', '▓', '█':
		return '#', false, true
	default:
		return 0, false, false
	}
	if mode == glyphDEC {
		return dec, true, true
	}
	return ascii, false, true
}

func onlyCombining(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, value := range runes {
		if !unicode.Is(unicode.M, value) {
			return false
		}
	}
	return true
}

func approximateRune(value rune) (byte, bool) {
	switch {
	case strings.ContainsRune("ÀÁÂÃÄÅ", value):
		return 'A', true
	case strings.ContainsRune("àáâãäå", value):
		return 'a', true
	case value == 'Ç':
		return 'C', true
	case value == 'ç':
		return 'c', true
	case strings.ContainsRune("ÈÉÊË", value):
		return 'E', true
	case strings.ContainsRune("èéêë", value):
		return 'e', true
	case strings.ContainsRune("ÌÍÎÏ", value):
		return 'I', true
	case strings.ContainsRune("ìíîï", value):
		return 'i', true
	case value == 'Ñ':
		return 'N', true
	case value == 'ñ':
		return 'n', true
	case strings.ContainsRune("ÒÓÔÕÖØ", value):
		return 'O', true
	case strings.ContainsRune("òóôõöø", value):
		return 'o', true
	case strings.ContainsRune("ÙÚÛÜ", value):
		return 'U', true
	case strings.ContainsRune("ùúûü", value):
		return 'u', true
	case strings.ContainsRune("ÝŸ", value):
		return 'Y', true
	case strings.ContainsRune("ýÿ", value):
		return 'y', true
	case strings.ContainsRune("─━═‐‑‒–—―−", value):
		return '-', true
	case strings.ContainsRune("│┃║", value):
		return '|', true
	case value >= '\u2500' && value <= '\u257f':
		return '+', true
	case value >= '\u2580' && value <= '\u259f':
		return '#', true
	case strings.ContainsRune("‘’‚‛", value):
		return '\'', true
	case strings.ContainsRune("“”„‟", value):
		return '"', true
	case strings.ContainsRune("•◦∙⋅", value):
		return '*', true
	case value == '\u2026':
		return '.', true
	case value == '\u00a0':
		return ' ', true
	case value == utf8.RuneError:
		return '?', true
	default:
		return 0, false
	}
}
