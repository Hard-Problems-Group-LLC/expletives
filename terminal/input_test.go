package terminal

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestInputDecoderPreservesSplitCSI(t *testing.T) {
	t.Parallel()

	input := []byte("\x1b[A")
	want := []expletives.KeyEvent{keyPress(expletives.KeyUp)}
	for split := 1; split < len(input); split++ {
		split := split
		t.Run(string(rune('0'+split)), func(t *testing.T) {
			t.Parallel()
			decoder := mustInputDecoder(t, InputDecoderOptions{
				EscapeTimeout: time.Second,
			})
			now := time.Unix(1, 0)
			var got []expletives.KeyEvent
			got = append(got, decoder.Feed(now, input[:split])...)
			got = append(got, decoder.Feed(now.Add(time.Millisecond), input[split:])...)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("split %d events = %#v, want %#v", split, got, want)
			}
		})
	}
}

func TestInputDecoderHandlesCoalescedInput(t *testing.T) {
	t.Parallel()

	decoder := mustInputDecoder(t, InputDecoderOptions{})
	got := decoder.Feed(time.Unix(1, 0), []byte("a\x1b[B\x12"))
	want := []expletives.KeyEvent{
		keyPress("a"),
		keyPress(expletives.KeyDown),
		{Kind: expletives.KeyEventDown, Key: expletives.KeyControl},
		keyPress("r"),
		{Kind: expletives.KeyEventUp, Key: expletives.KeyControl},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderRecognizesCycleBrackets(t *testing.T) {
	t.Parallel()
	decoder := mustInputDecoder(t, InputDecoderOptions{})
	got := decoder.Feed(time.Unix(1, 0), []byte("[]"))
	want := []expletives.KeyEvent{
		keyPress(expletives.KeyLeftBracket),
		keyPress(expletives.KeyRightBracket),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderRecognizesNavigationSequences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sequence string
		key      expletives.Key
	}{
		{name: "up CSI", sequence: "\x1b[A", key: expletives.KeyUp},
		{name: "down CSI", sequence: "\x1b[B", key: expletives.KeyDown},
		{name: "right CSI", sequence: "\x1b[C", key: expletives.KeyRight},
		{name: "left CSI", sequence: "\x1b[D", key: expletives.KeyLeft},
		{name: "up SS3", sequence: "\x1bOA", key: expletives.KeyUp},
		{name: "home", sequence: "\x1b[1~", key: expletives.KeyHome},
		{name: "end", sequence: "\x1b[4~", key: expletives.KeyEnd},
		{name: "insert", sequence: "\x1b[2~", key: expletives.KeyInsert},
		{name: "delete", sequence: "\x1b[3~", key: expletives.KeyDelete},
		{name: "page up", sequence: "\x1b[5~", key: expletives.KeyPageUp},
		{name: "page down", sequence: "\x1b[6~", key: expletives.KeyPageDown},
		{name: "F1", sequence: "\x1bOP", key: "f1"},
		{name: "F2", sequence: "\x1bOQ", key: "f2"},
		{name: "F3", sequence: "\x1bOR", key: "f3"},
		{name: "F4", sequence: "\x1bOS", key: "f4"},
		{name: "F5", sequence: "\x1b[15~", key: "f5"},
		{name: "F6", sequence: "\x1b[17~", key: "f6"},
		{name: "F7", sequence: "\x1b[18~", key: "f7"},
		{name: "F8", sequence: "\x1b[19~", key: "f8"},
		{name: "F9", sequence: "\x1b[20~", key: "f9"},
		{name: "F10", sequence: "\x1b[21~", key: "f10"},
		{name: "F11", sequence: "\x1b[23~", key: "f11"},
		{name: "F12", sequence: "\x1b[24~", key: "f12"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decoder := mustInputDecoder(t, InputDecoderOptions{})
			got := decoder.Feed(time.Unix(1, 0), []byte(test.sequence))
			want := []expletives.KeyEvent{keyPress(test.key)}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("events = %#v, want %#v", got, want)
			}
		})
	}
}

func TestInputDecoderRecognizesShiftTab(t *testing.T) {
	t.Parallel()

	decoder := mustInputDecoder(t, InputDecoderOptions{})
	got := decoder.Feed(time.Unix(1, 0), []byte("\x1b[Z"))
	want := []expletives.KeyEvent{
		{Kind: expletives.KeyEventDown, Key: expletives.KeyShift},
		keyPress(expletives.KeyTab),
		{Kind: expletives.KeyEventUp, Key: expletives.KeyShift},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderEmitsShiftNavigationLifecycle(t *testing.T) {
	t.Parallel()
	decoder := mustInputDecoder(t, InputDecoderOptions{})
	got := decoder.Feed(time.Unix(1, 0), []byte("\x1b[1;2D"))
	want := []expletives.KeyEvent{
		{Kind: expletives.KeyEventDown, Key: expletives.KeyShift},
		keyPress(expletives.KeyLeft),
		{Kind: expletives.KeyEventUp, Key: expletives.KeyShift},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shift-left events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderRecognizesEnhancedCtrlEnter(t *testing.T) {
	t.Parallel()
	want := []expletives.KeyEvent{
		{Kind: expletives.KeyEventDown, Key: expletives.KeyControl},
		keyPress(expletives.KeyEnter),
		{Kind: expletives.KeyEventUp, Key: expletives.KeyControl},
	}
	for _, test := range []struct {
		name     string
		sequence string
	}{
		{name: "Kitty", sequence: "\x1b[13;5u"},
		{name: "modifyOtherKeys", sequence: "\x1b[27;5;13~"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decoder := mustInputDecoder(t, InputDecoderOptions{})
			got := decoder.Feed(time.Unix(1, 0), []byte(test.sequence))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("events = %#v, want %#v", got, want)
			}
		})
	}
}

func TestInputDecoderRecognizesEnhancedAltX(t *testing.T) {
	t.Parallel()
	want := []expletives.KeyEvent{
		{Kind: expletives.KeyEventDown, Key: expletives.KeyAlt},
		keyPress("x"),
		{Kind: expletives.KeyEventUp, Key: expletives.KeyAlt},
	}
	for _, test := range []struct {
		name     string
		sequence string
	}{
		{name: "Kitty", sequence: "\x1b[120;3u"},
		{name: "modifyOtherKeys", sequence: "\x1b[27;3;120~"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decoder := mustInputDecoder(t, InputDecoderOptions{})
			got := decoder.Feed(time.Unix(1, 0), []byte(test.sequence))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("events = %#v, want %#v", got, want)
			}
		})
	}
}

func TestInputDecoderEmitsBoundedPasteInStreamOrder(t *testing.T) {
	t.Parallel()
	decoder := mustInputDecoder(t, InputDecoderOptions{})
	got := decoder.FeedInput(
		time.Unix(1, 0),
		[]byte("a\x1b[200~line 1\n\x1b[A\x1b[201~b"),
	)
	if len(got) != 3 ||
		got[0].KeyEvent == nil || got[0].KeyEvent.Key != "a" ||
		got[1].TextInput == nil ||
		got[1].TextInput.Kind != expletives.TextInputPaste ||
		got[1].TextInput.Text != "line 1\n\x1b[A" ||
		got[2].KeyEvent == nil || got[2].KeyEvent.Key != "b" {
		t.Fatalf("decoded input = %#v", got)
	}
}

func TestInputDecoderDiscardsWholeOversizedPaste(t *testing.T) {
	t.Parallel()
	decoder := mustInputDecoder(t, InputDecoderOptions{})
	start := time.Unix(1, 0)
	if got := decoder.FeedInput(start, []byte(pasteStart)); len(got) != 0 {
		t.Fatalf("paste start events = %#v", got)
	}
	chunk := bytes.Repeat([]byte{'x'}, 4096)
	for written := 0; written <= expletives.MaxTextInputBytes; written += len(chunk) {
		if got := decoder.FeedInput(start, chunk); len(got) != 0 {
			t.Fatalf("paste body events = %#v", got)
		}
	}
	got := decoder.FeedInput(start, append(pasteEnd[:], 'a'))
	if len(got) != 1 || got[0].KeyEvent == nil ||
		got[0].KeyEvent.Key != "a" {
		t.Fatalf("oversized paste tail events = %#v", got)
	}
}

func TestInputDecoderPreservesSplitUTF8PrintableKeys(t *testing.T) {
	t.Parallel()

	now := time.Unix(1, 0)
	for _, encoded := range [][]byte{
		[]byte("é"),
		[]byte("🙂"),
	} {
		for split := 1; split < len(encoded); split++ {
			decoder := mustInputDecoder(t, InputDecoderOptions{})
			if got := decoder.Feed(now, encoded[:split]); len(got) != 0 {
				t.Fatalf("% x split %d first events = %#v, want none",
					encoded, split, got)
			}
			second := append(append([]byte(nil), encoded[split:]...), 'a')
			got := decoder.Feed(now.Add(time.Millisecond), second)
			want := []expletives.KeyEvent{
				keyPress(expletives.Key(string(encoded))),
				keyPress("a"),
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("% x split %d completed events = %#v, want %#v",
					encoded, split, got, want)
			}
		}
	}
}

func TestInputDecoderEscapeDeadlineAndAlt(t *testing.T) {
	t.Parallel()

	timeout := 20 * time.Millisecond
	start := time.Unix(1, 0)

	t.Run("bare escape waits", func(t *testing.T) {
		decoder := mustInputDecoder(t, InputDecoderOptions{
			EscapeTimeout: timeout,
		})
		if got := decoder.Feed(start, []byte{0x1b}); len(got) != 0 {
			t.Fatalf("Feed(Escape) events = %#v, want none", got)
		}
		deadline, ok := decoder.Deadline()
		if !ok || !deadline.Equal(start.Add(timeout)) {
			t.Fatalf("Deadline() = %v, %t, want %v, true",
				deadline, ok, start.Add(timeout))
		}
		if got := decoder.Flush(deadline.Add(-time.Nanosecond)); len(got) != 0 {
			t.Fatalf("early Flush() events = %#v, want none", got)
		}
		want := []expletives.KeyEvent{keyPress(expletives.KeyEscape)}
		if got := decoder.Flush(deadline); !reflect.DeepEqual(got, want) {
			t.Fatalf("due Flush() events = %#v, want %#v", got, want)
		}
	})

	t.Run("alt prefix arrives before deadline", func(t *testing.T) {
		decoder := mustInputDecoder(t, InputDecoderOptions{
			EscapeTimeout: timeout,
		})
		if got := decoder.Feed(start, []byte{0x1b}); len(got) != 0 {
			t.Fatalf("Feed(Escape) events = %#v, want none", got)
		}
		got := decoder.Feed(start.Add(time.Millisecond), []byte{'F'})
		want := []expletives.KeyEvent{
			{Kind: expletives.KeyEventDown, Key: expletives.KeyAlt},
			keyPress("F"),
			{Kind: expletives.KeyEventUp, Key: expletives.KeyAlt},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Alt-F events = %#v, want %#v", got, want)
		}
	})

	t.Run("late byte is not alt", func(t *testing.T) {
		decoder := mustInputDecoder(t, InputDecoderOptions{
			EscapeTimeout: timeout,
		})
		_ = decoder.Feed(start, []byte{0x1b})
		got := decoder.Feed(start.Add(timeout), []byte{'a'})
		want := []expletives.KeyEvent{
			keyPress(expletives.KeyEscape),
			keyPress("a"),
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("late input events = %#v, want %#v", got, want)
		}
	})
}

func TestInputDecoderDiscardsUnknownAndOverlongSequences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options InputDecoderOptions
		input   []byte
	}{
		{
			name:  "unknown CSI",
			input: []byte("\x1b[999xq"),
		},
		{
			name:  "unknown ESC intermediate",
			input: []byte("\x1b(Bq"),
		},
		{
			name: "overlong CSI",
			options: InputDecoderOptions{
				MaxPendingBytes: 8,
			},
			input: []byte("\x1b[111111111111Aq"),
		},
		{
			name:  "bracketed paste",
			input: []byte("\x1b[200~rq\x1b[201~a"),
		},
		{
			name:  "OSC string",
			input: []byte("\x1b]title-rq\a" + "a"),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decoder := mustInputDecoder(t, test.options)
			got := decoder.Feed(time.Unix(1, 0), test.input)
			want := []expletives.KeyEvent{keyPress("a")}
			if test.name != "bracketed paste" &&
				test.name != "OSC string" {
				want = []expletives.KeyEvent{keyPress("q")}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("events = %#v, want %#v", got, want)
			}
			if len(decoder.pending) > decoder.maxPending {
				t.Fatalf("pending bytes = %d, limit %d",
					len(decoder.pending), decoder.maxPending)
			}
		})
	}
}

func TestInputDecoderTimedOutCSIDiscardsLateFinal(t *testing.T) {
	t.Parallel()

	timeout := 20 * time.Millisecond
	start := time.Unix(1, 0)
	decoder := mustInputDecoder(t, InputDecoderOptions{
		EscapeTimeout: timeout,
	})
	if got := decoder.Feed(start, []byte("\x1b[")); len(got) != 0 {
		t.Fatalf("partial CSI events = %#v, want none", got)
	}
	if got := decoder.Flush(start.Add(timeout)); len(got) != 0 {
		t.Fatalf("expired CSI events = %#v, want none", got)
	}
	got := decoder.Feed(start.Add(timeout+time.Millisecond), []byte("Aq"))
	want := []expletives.KeyEvent{keyPress("q")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("late CSI final events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderTimedOutPastePrefixStillDiscardsPaste(t *testing.T) {
	t.Parallel()

	timeout := 20 * time.Millisecond
	start := time.Unix(1, 0)
	decoder := mustInputDecoder(t, InputDecoderOptions{
		EscapeTimeout: timeout,
	})
	if got := decoder.Feed(start, []byte("\x1b[20")); len(got) != 0 {
		t.Fatalf("partial paste prefix events = %#v, want none", got)
	}
	if got := decoder.Flush(start.Add(timeout)); len(got) != 0 {
		t.Fatalf("expired paste prefix events = %#v, want none", got)
	}
	got := decoder.Feed(
		start.Add(timeout+time.Millisecond),
		[]byte("0~rq\x1b[201~a"),
	)
	want := []expletives.KeyEvent{keyPress("a")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("late paste events = %#v, want %#v", got, want)
	}
}

func TestInputDecoderResetDropsPendingInput(t *testing.T) {
	t.Parallel()

	decoder := mustInputDecoder(t, InputDecoderOptions{})
	if got := decoder.Feed(time.Unix(1, 0), []byte("\x1b[")); len(got) != 0 {
		t.Fatalf("partial CSI events = %#v, want none", got)
	}
	decoder.Reset()
	if _, ok := decoder.Deadline(); ok {
		t.Fatal("Deadline() remained active after Reset()")
	}
	if len(decoder.pending) != 0 || decoder.state != stateGround {
		t.Fatalf("state after Reset() = %d, pending %d",
			decoder.state, len(decoder.pending))
	}
}

func TestInputDecoderRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	tests := []InputDecoderOptions{
		{EscapeTimeout: -time.Nanosecond},
		{MaxPendingBytes: len(pasteStart) - 1},
		{MaxPendingBytes: MaxPendingBytesLimit + 1},
	}
	for _, options := range tests {
		if _, err := NewInputDecoder(options); !errors.Is(
			err,
			ErrInvalidInputDecoderOptions,
		) {
			t.Errorf("NewInputDecoder(%+v) error = %v, want %v",
				options, err, ErrInvalidInputDecoderOptions)
		}
	}
}

func FuzzInputDecoder(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		[]byte("abc"),
		[]byte("\x1b[A"),
		[]byte("\x1b[200~rq\x1b[201~"),
		{0xc3, 0xa9},
		[]byte("\x1b[111111111111111111A"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		decoder := mustInputDecoder(t, InputDecoderOptions{
			EscapeTimeout:   time.Millisecond,
			MaxPendingBytes: 8,
		})
		now := time.Unix(1, 0)
		for _, value := range input {
			events := decoder.Feed(now, []byte{value})
			assertLogicalEvents(t, events)
			if len(decoder.pending) > decoder.maxPending {
				t.Fatalf("pending bytes = %d, limit %d",
					len(decoder.pending), decoder.maxPending)
			}
			now = now.Add(time.Microsecond)
		}
		assertLogicalEvents(t, decoder.Flush(now.Add(time.Second)))
		decoder.Reset()
		if len(decoder.pending) != 0 {
			t.Fatalf("pending bytes after Reset() = %d", len(decoder.pending))
		}
	})
}

func mustInputDecoder(
	t testing.TB,
	options InputDecoderOptions,
) *InputDecoder {
	t.Helper()
	decoder, err := NewInputDecoder(options)
	if err != nil {
		t.Fatalf("NewInputDecoder() error = %v", err)
	}
	return decoder
}

func assertLogicalEvents(t testing.TB, events []expletives.KeyEvent) {
	t.Helper()
	for _, event := range events {
		switch event.Kind {
		case expletives.KeyEventDown,
			expletives.KeyEventUp,
			expletives.KeyEventPress:
		default:
			t.Fatalf("event kind = %q", event.Kind)
		}
		if event.Key == "" {
			t.Fatal("event has empty key")
		}
	}
}
