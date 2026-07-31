// Package display owns expletives' canonical one-cell text policy.
package display

import (
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// PolicyVersion identifies the segmentation and width implementation used by
// this build.
const PolicyVersion = "uniseg-0.4.7-default-width"

const Replacement = "\uFFFD"

// Normalize splits text into extended grapheme clusters and returns one
// canonical cell value for each cluster. Anything not proven to occupy exactly
// one cell becomes one replacement character.
func Normalize(text string) []string {
	graphemes := uniseg.NewGraphemes(text)
	cells := make([]string, 0, len(text))
	for graphemes.Next() {
		cluster := graphemes.Str()
		if !utf8.ValidString(cluster) || graphemes.Width() != 1 {
			cluster = Replacement
		}
		cells = append(cells, cluster)
	}
	return cells
}
