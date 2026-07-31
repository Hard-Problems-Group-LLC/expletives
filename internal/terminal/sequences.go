package terminal

const (
	enterTerminal = "\x1b[?1049h\x1b[?25l\x1b[0m\x1b[2J\x1b[H"
	leaveTerminal = "\x1b[0m\x1b[?25h\x1b[?1049l"
)
