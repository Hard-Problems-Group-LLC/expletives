package terminal

const (
	// Request disambiguated modified keys from terminals implementing either
	// the Kitty keyboard protocol or xterm modifyOtherKeys. Unsupported control
	// sequences are ignored by conforming terminals. Only atomic key presses
	// are requested; the toolkit synthesizes bounded modifier lifecycle.
	enterTerminal = "\x1b[?1049h\x1b[?2004h\x1b[>1u\x1b[>4;2m\x1b[?25l\x1b[0m\x1b[2J\x1b[H"
	leaveTerminal = "\x1b[0m\x1b[?25h\x1b[>4m\x1b[<u\x1b[?2004l\x1b[?1049l"
)
