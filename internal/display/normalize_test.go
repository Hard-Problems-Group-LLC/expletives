package display

import (
	"reflect"
	"testing"
)

func TestNormalizeOneCellPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "ascii",
			text: "ABC",
			want: []string{"A", "B", "C"},
		},
		{
			name: "precomposed one cell",
			text: "\u00E9",
			want: []string{"\u00E9"},
		},
		{
			name: "decomposed one cell",
			text: "e\u0301",
			want: []string{"e\u0301"},
		},
		{
			name: "wide CJK once",
			text: "\u754C",
			want: []string{Replacement},
		},
		{
			name: "wide emoji once",
			text: "\U0001F642",
			want: []string{Replacement},
		},
		{
			name: "joined emoji once",
			text: "\U0001F469\u200D\U0001F4BB",
			want: []string{Replacement},
		},
		{
			name: "standalone combining sequence",
			text: "\u0301",
			want: []string{Replacement},
		},
		{
			name: "control",
			text: "\n",
			want: []string{Replacement},
		},
		{
			name: "malformed UTF-8",
			text: string([]byte{0xFF}),
			want: []string{Replacement},
		},
		{
			name: "unsupported element does not shift neighbors",
			text: "A\u754CB",
			want: []string{"A", Replacement, "B"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(test.text); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Normalize(%q) = %#v, want %#v", test.text, got, test.want)
			}
		})
	}
}

func TestNormalizeEmpty(t *testing.T) {
	t.Parallel()
	if got := Normalize(""); len(got) != 0 {
		t.Fatalf("Normalize(empty) = %#v, want no cells", got)
	}
}
