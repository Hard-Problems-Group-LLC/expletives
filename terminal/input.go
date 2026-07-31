package terminal

import (
	"errors"
	"time"
	"unicode/utf8"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const (
	// DefaultEscapeTimeout is the default time allowed for bytes following an
	// ambiguous Escape byte.
	DefaultEscapeTimeout = 35 * time.Millisecond
	// DefaultMaxPendingBytes bounds an incomplete UTF-8 or terminal sequence.
	DefaultMaxPendingBytes = 64
	// MaxPendingBytesLimit is the largest pending-byte limit accepted by
	// NewInputDecoder.
	MaxPendingBytesLimit = 4096
)

var (
	// ErrInvalidInputDecoderOptions reports an invalid decoder bound or
	// deadline.
	ErrInvalidInputDecoderOptions = errors.New("invalid terminal input decoder options")
)

// InputDecoderOptions configures an InputDecoder. Zero fields select the
// documented defaults.
type InputDecoderOptions struct {
	// EscapeTimeout is the maximum ambiguity interval for a bare Escape byte
	// or an incomplete escape sequence. It must not be negative.
	EscapeTimeout time.Duration
	// MaxPendingBytes bounds retained input between Feed calls. Values from 6
	// through MaxPendingBytesLimit are accepted; zero selects the default.
	MaxPendingBytes int
}

type decoderState uint8

const (
	stateGround decoderState = iota
	stateUTF8
	stateEscape
	stateAltUTF8
	stateEscapeIntermediate
	stateCSI
	stateSS3
	stateDiscardCSI
	stateDiscardEscape
	stateDiscardString
	statePaste
)

var pasteEnd = [...]byte{0x1b, '[', '2', '0', '1', '~'}

const pasteStart = "\x1b[200~"

// InputDecoder incrementally converts physical terminal input into logical
// key lifecycle events.
//
// The zero value is not initialized; construct an InputDecoder with
// NewInputDecoder.
//
// InputDecoder retains incomplete UTF-8, CSI, and SS3 prefixes across Feed
// calls. Bare Escape is emitted only after EscapeTimeout. Unknown control
// sequences, valid non-ASCII text, invalid UTF-8, and bracketed-paste content
// are discarded because the current toolkit key vocabulary cannot represent
// committed text. InputDecoder is intentionally a small supported-key parser,
// not a general terminal-protocol implementation.
//
// InputDecoder is not safe for concurrent use. It belongs to the one terminal
// input owner. Feed and Flush times must come from one nondecreasing clock;
// Deadline returns a value in that same clock domain.
type InputDecoder struct {
	escapeTimeout time.Duration
	maxPending    int

	state           decoderState
	pending         []byte
	deadline        time.Time
	pasteEndMatch   int
	stringSawEscape bool
	stringEndsAtBEL bool
}

// NewInputDecoder validates and copies options and constructs a bounded
// incremental terminal input decoder. It acquires no external resources.
func NewInputDecoder(options InputDecoderOptions) (*InputDecoder, error) {
	escapeTimeout := options.EscapeTimeout
	if escapeTimeout == 0 {
		escapeTimeout = DefaultEscapeTimeout
	}
	if escapeTimeout < 0 {
		return nil, ErrInvalidInputDecoderOptions
	}

	maxPending := options.MaxPendingBytes
	if maxPending == 0 {
		maxPending = DefaultMaxPendingBytes
	}
	if maxPending < len(pasteStart) || maxPending > MaxPendingBytesLimit {
		return nil, ErrInvalidInputDecoderOptions
	}

	return &InputDecoder{
		escapeTimeout: escapeTimeout,
		maxPending:    maxPending,
		pending:       make([]byte, 0, min(maxPending, DefaultMaxPendingBytes)),
	}, nil
}

// Feed consumes one arbitrary input chunk at now and returns every complete
// logical event in byte-stream order. Feed does not retain the caller's input
// slice after it returns, and the returned slice is caller-owned. If an Escape
// deadline elapsed before now, that prefix is resolved before the new bytes
// are consumed.
func (d *InputDecoder) Feed(now time.Time, input []byte) []expletives.KeyEvent {
	events := d.flushExpired(now)
	for _, value := range input {
		events = d.consume(events, now, value)
	}
	return events
}

// Deadline returns the deadline for the currently ambiguous Escape prefix.
// The second result is false when no timer is required. Deadline does not
// resolve an already elapsed prefix; call Flush or Feed.
func (d *InputDecoder) Deadline() (time.Time, bool) {
	if d.deadline.IsZero() {
		return time.Time{}, false
	}
	return d.deadline, true
}

// Flush resolves an Escape prefix whose deadline is at or before now.
// A bare Escape emits KeyEscape; incomplete control sequences are discarded.
// Prefixes whose deadline has not elapsed remain pending.
func (d *InputDecoder) Flush(now time.Time) []expletives.KeyEvent {
	return d.flushExpired(now)
}

// Reset discards every retained prefix and unknown-sequence recovery state.
// It emits no events and is suitable for disconnect, suspension, and
// shutdown boundaries. It preserves the configured bounds and timeout.
func (d *InputDecoder) Reset() {
	d.state = stateGround
	d.pending = d.pending[:0]
	d.deadline = time.Time{}
	d.pasteEndMatch = 0
	d.stringSawEscape = false
	d.stringEndsAtBEL = false
}

func (d *InputDecoder) flushExpired(now time.Time) []expletives.KeyEvent {
	if d.deadline.IsZero() || now.Before(d.deadline) {
		return nil
	}

	switch d.state {
	case stateEscape:
		d.Reset()
		return []expletives.KeyEvent{keyPress(expletives.KeyEscape)}
	case stateCSI, stateSS3:
		d.state = stateDiscardCSI
		d.deadline = time.Time{}
	case stateEscapeIntermediate:
		d.state = stateDiscardEscape
		d.deadline = time.Time{}
	case stateAltUTF8:
		d.Reset()
	default:
		d.deadline = time.Time{}
	}
	return nil
}

func (d *InputDecoder) consume(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	switch d.state {
	case stateGround:
		return d.consumeGround(events, now, value)
	case stateUTF8:
		return d.consumeUTF8(events, now, value)
	case stateEscape:
		return d.consumeEscape(events, now, value)
	case stateAltUTF8:
		return d.consumeUTF8(events, now, value)
	case stateEscapeIntermediate:
		return d.consumeEscapeIntermediate(events, now, value)
	case stateCSI, stateSS3:
		return d.consumeControlSequence(events, now, value)
	case stateDiscardCSI:
		return d.consumeDiscardCSI(events, now, value)
	case stateDiscardEscape:
		return d.consumeDiscardEscape(events, now, value)
	case stateDiscardString:
		return d.consumeDiscardString(events, value)
	case statePaste:
		return d.consumePaste(events, value)
	default:
		d.Reset()
		return events
	}
}

func (d *InputDecoder) consumeGround(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	switch {
	case value == 0x1b:
		d.beginEscape(now)
		return events
	case value < utf8.RuneSelf:
		return appendASCII(events, value)
	case !utf8.RuneStart(value):
		return events
	default:
		d.state = stateUTF8
		d.pending = append(d.pending[:0], value)
		return d.finishUTF8(events, now)
	}
}

func (d *InputDecoder) consumeEscape(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	switch value {
	case '[':
		d.state = stateCSI
		d.pending = append(d.pending, value)
		return events
	case 'O':
		d.state = stateSS3
		d.pending = append(d.pending, value)
		return events
	case ']':
		d.state = stateDiscardString
		d.pending = d.pending[:0]
		d.deadline = time.Time{}
		d.stringSawEscape = false
		d.stringEndsAtBEL = true
		return events
	case 'P', '^', '_', 'X':
		d.state = stateDiscardString
		d.pending = d.pending[:0]
		d.deadline = time.Time{}
		d.stringSawEscape = false
		d.stringEndsAtBEL = false
		return events
	case 0x1b:
		d.Reset()
		return appendAlt(events, []expletives.KeyEvent{
			keyPress(expletives.KeyEscape),
		})
	}

	if value >= 0x20 && value <= 0x2f {
		d.state = stateEscapeIntermediate
		d.pending = append(d.pending, value)
		return events
	}
	if value >= utf8.RuneSelf {
		if !utf8.RuneStart(value) {
			d.Reset()
			return events
		}
		d.state = stateAltUTF8
		d.pending = append(d.pending[:0], value)
		return d.finishUTF8(events, now)
	}

	decoded := appendASCII(nil, value)
	d.Reset()
	if len(decoded) == 0 {
		return events
	}
	return append(events, appendAlt(nil, decoded)...)
}

func (d *InputDecoder) consumeEscapeIntermediate(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	if len(d.pending) >= d.maxPending {
		d.state = stateDiscardEscape
		d.pending = d.pending[:0]
		d.deadline = time.Time{}
		return d.consumeDiscardEscape(events, now, value)
	}
	d.pending = append(d.pending, value)
	if value >= 0x30 && value <= 0x7e {
		d.Reset()
		return events
	}
	if value >= 0x20 && value <= 0x2f {
		return events
	}
	d.Reset()
	if value == 0x1b {
		d.beginEscape(now)
	}
	return events
}

func (d *InputDecoder) consumeUTF8(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	if len(d.pending) >= d.maxPending {
		d.Reset()
		return d.consumeGround(events, now, value)
	}
	d.pending = append(d.pending, value)
	return d.finishUTF8(events, now)
}

func (d *InputDecoder) finishUTF8(
	events []expletives.KeyEvent,
	now time.Time,
) []expletives.KeyEvent {
	if !utf8.FullRune(d.pending) {
		return events
	}

	_, size := utf8.DecodeRune(d.pending)
	remainder := append([]byte(nil), d.pending[size:]...)
	d.Reset()
	for _, value := range remainder {
		events = d.consume(events, now, value)
	}
	return events
}

func (d *InputDecoder) consumeControlSequence(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	if len(d.pending) >= d.maxPending {
		d.state = stateDiscardCSI
		d.pending = d.pending[:0]
		d.deadline = time.Time{}
		return d.consumeDiscardCSI(events, now, value)
	}
	d.pending = append(d.pending, value)

	if value >= 0x40 && value <= 0x7e {
		sequence := string(d.pending)
		d.Reset()
		if sequence == pasteStart {
			d.state = statePaste
			return events
		}
		if key, ok := keyForSequence(sequence); ok {
			return append(events, keyEvents(key)...)
		}
		return events
	}
	if value >= 0x20 && value <= 0x3f {
		return events
	}

	d.Reset()
	if value == 0x1b {
		d.beginEscape(now)
	}
	return events
}

func (d *InputDecoder) consumeDiscardCSI(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	if len(d.pending) != 0 {
		if len(d.pending) < d.maxPending {
			d.pending = append(d.pending, value)
		} else {
			d.pending = d.pending[:0]
		}
	}
	if value >= 0x40 && value <= 0x7e {
		paste := string(d.pending) == pasteStart
		d.Reset()
		if paste {
			d.state = statePaste
		}
		return events
	}
	if value == 0x1b {
		d.beginEscape(now)
	}
	return events
}

func (d *InputDecoder) consumeDiscardEscape(
	events []expletives.KeyEvent,
	now time.Time,
	value byte,
) []expletives.KeyEvent {
	if value >= 0x30 && value <= 0x7e {
		d.Reset()
		return events
	}
	if value == 0x1b {
		d.beginEscape(now)
	}
	return events
}

func (d *InputDecoder) consumeDiscardString(
	events []expletives.KeyEvent,
	value byte,
) []expletives.KeyEvent {
	if (d.stringEndsAtBEL && value == 0x07) ||
		(d.stringSawEscape && value == '\\') {
		d.Reset()
		return events
	}
	d.stringSawEscape = value == 0x1b
	return events
}

func (d *InputDecoder) consumePaste(
	events []expletives.KeyEvent,
	value byte,
) []expletives.KeyEvent {
	if value == pasteEnd[d.pasteEndMatch] {
		d.pasteEndMatch++
		if d.pasteEndMatch == len(pasteEnd) {
			d.Reset()
		}
		return events
	}
	if value == pasteEnd[0] {
		d.pasteEndMatch = 1
	} else {
		d.pasteEndMatch = 0
	}
	return events
}

func (d *InputDecoder) beginEscape(now time.Time) {
	d.state = stateEscape
	d.pending = append(d.pending[:0], 0x1b)
	d.deadline = now.Add(d.escapeTimeout)
	d.pasteEndMatch = 0
	d.stringSawEscape = false
	d.stringEndsAtBEL = false
}

func keyForSequence(sequence string) (expletives.Key, bool) {
	switch sequence {
	case "\x1b[A", "\x1bOA":
		return expletives.KeyUp, true
	case "\x1b[B", "\x1bOB":
		return expletives.KeyDown, true
	case "\x1b[C", "\x1bOC":
		return expletives.KeyRight, true
	case "\x1b[D", "\x1bOD":
		return expletives.KeyLeft, true
	case "\x1b[H", "\x1bOH", "\x1b[1~", "\x1b[7~":
		return expletives.KeyHome, true
	case "\x1b[F", "\x1bOF", "\x1b[4~", "\x1b[8~":
		return expletives.KeyEnd, true
	case "\x1b[2~":
		return expletives.KeyInsert, true
	case "\x1b[3~":
		return expletives.KeyDelete, true
	case "\x1b[5~":
		return expletives.KeyPageUp, true
	case "\x1b[6~":
		return expletives.KeyPageDown, true
	case "\x1b[Z":
		return expletives.KeyTab, true
	case "\x1bOP":
		return "f1", true
	case "\x1bOQ":
		return "f2", true
	case "\x1bOR":
		return "f3", true
	case "\x1bOS":
		return "f4", true
	case "\x1b[15~":
		return "f5", true
	case "\x1b[17~":
		return "f6", true
	case "\x1b[18~":
		return "f7", true
	case "\x1b[19~":
		return "f8", true
	case "\x1b[20~":
		return "f9", true
	case "\x1b[21~":
		return "f10", true
	case "\x1b[23~":
		return "f11", true
	case "\x1b[24~":
		return "f12", true
	default:
		return "", false
	}
}

func keyEvents(key expletives.Key) []expletives.KeyEvent {
	if key == expletives.KeyTab {
		return []expletives.KeyEvent{
			{Kind: expletives.KeyEventDown, Key: expletives.KeyShift},
			keyPress(expletives.KeyTab),
			{Kind: expletives.KeyEventUp, Key: expletives.KeyShift},
		}
	}
	return []expletives.KeyEvent{keyPress(key)}
}

func appendASCII(
	events []expletives.KeyEvent,
	value byte,
) []expletives.KeyEvent {
	switch {
	case value >= 0x01 && value <= 0x1a &&
		value != '\b' &&
		value != '\t' &&
		value != '\n' &&
		value != '\r':
		return appendControl(events, expletives.Key('a'+value-1))
	case value >= 'A' && value <= 'Z':
		return append(events, keyPress(expletives.Key(value-'A'+'a')))
	case value >= 'a' && value <= 'z':
		return append(events, keyPress(expletives.Key(value)))
	case value >= '0' && value <= '9':
		return append(events, keyPress(expletives.Key(value)))
	}

	switch value {
	case 0:
		return appendControl(events, expletives.KeySpace)
	case ' ':
		return append(events, keyPress(expletives.KeySpace))
	case '\r', '\n':
		return append(events, keyPress(expletives.KeyEnter))
	case '\t':
		return append(events, keyPress(expletives.KeyTab))
	case 0x7f, '\b':
		return append(events, keyPress(expletives.KeyBackspace))
	default:
		return events
	}
}

func appendControl(
	events []expletives.KeyEvent,
	key expletives.Key,
) []expletives.KeyEvent {
	return append(
		events,
		expletives.KeyEvent{
			Kind: expletives.KeyEventDown,
			Key:  expletives.KeyControl,
		},
		keyPress(key),
		expletives.KeyEvent{
			Kind: expletives.KeyEventUp,
			Key:  expletives.KeyControl,
		},
	)
}

func appendAlt(
	events []expletives.KeyEvent,
	inside []expletives.KeyEvent,
) []expletives.KeyEvent {
	events = append(events, expletives.KeyEvent{
		Kind: expletives.KeyEventDown,
		Key:  expletives.KeyAlt,
	})
	events = append(events, inside...)
	return append(events, expletives.KeyEvent{
		Kind: expletives.KeyEventUp,
		Key:  expletives.KeyAlt,
	})
}

func keyPress(key expletives.Key) expletives.KeyEvent {
	return expletives.KeyEvent{
		Kind: expletives.KeyEventPress,
		Key:  key,
	}
}
