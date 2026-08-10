# Terminal Compatibility v0

- Status: Phase 19 active contract
- Scope: physical terminal profiles, capability policy, lifecycle, and
  verification
- Authorization: `EXPL-TASK-036` and the directed terminal requirements

## Purpose

This contract states exactly where the CGO-free Linux Presenter may run and
what evidence is required before support is broadened. It separates three
things that `$TERM` is often incorrectly treated as proving:

1. a deliberately supported control-sequence profile;
2. a character repertoire selected from the active locale; and
3. a real emulator, multiplexer, and transport combination that has actually
   passed the physical verification matrix.

The Presenter fails closed when the first item is unknown. It never guesses a
high-bit code page or promotes one successful terminal into a generic ANSI,
VT100, curses, ncurses, or terminfo compatibility claim.

## Current Runtime Matrix

| Platform | `$TERM` form | Profile | UTF-8 locale | Non-UTF-8 locale |
| --- | --- | --- | --- | --- |
| Linux | `xterm`, `xterm-*` | `xterm` | canonical supported one-cell graphemes render directly | DEC line art; other non-ASCII cells use highlighted ASCII approximation |
| Linux | `screen`, `screen-*`, `screen.*` | `screen` | canonical supported one-cell graphemes render directly | DEC line art; other non-ASCII cells use highlighted ASCII approximation |
| Linux | `tmux`, `tmux-*`, `tmux.*` | `tmux` | canonical supported one-cell graphemes render directly | DEC line art; other non-ASCII cells use highlighted ASCII approximation |

An active locale is UTF-8 only when the first nonempty `LC_ALL`, `LC_CTYPE`,
or `LANG` value contains `UTF-8` or `UTF8`, case-insensitively. All other
values select the conservative DEC/ASCII path. A non-UTF-8 locale does not
prove an upper-half character mapping.

The current profile contract requires absolute cursor positioning, clearing,
the ANSI 16-color palette and supported SGR attributes, alternate-screen and
cursor modes, bracketed paste, and the requested xterm/Kitty enhanced-keyboard
forms. Unsupported requests must be ignored safely by a conforming terminal.
The correctness-first renderer emits a complete bounded frame and does not
depend on incremental-update state retained by the terminal.

Remote use through SSH, mosh, a serial bridge, or another transport is not a
separate profile. It is within the narrow runtime claim only when the remote
endpoint presents one of the supported `$TERM` forms and the transport
preserves its required byte stream. Unknown remote code pages always use the
highlighted ASCII fallback. Latency, disconnect, and transport-specific
claims require their own recorded matrix evidence.

## Deliberately Unsupported Claims

The following remain unsupported until a later recorded compatibility
extension supplies implementation and evidence:

- `ansi`, `vt100`, `linux`, `dumb`, and unrelated `$TERM` families;
- generic compatibility inferred from emulator names, `COLORTERM`, or a
  single terminfo entry;
- direct upper-half code-page output without an exact proven mapping;
- a monochrome physical warning presentation;
- Windows, Darwin, BSD, and non-POSIX terminal ownership;
- terminal queries whose replies could be confused with user input; and
- arbitrary curses/ncurses ABI or application-window interoperability.

Terminfo is corroborating capability evidence, not an automatic source of
trust. The project-owned reader accepts the ncurses/System V legacy compiled
magic and the ncurses extended-number magic, reads at most 32 KiB, validates
every section and string-table offset, and checks only the stable capability
indices consumed by the static Presenter. It supports directory-tree entries
from the normal `TERMINFO`, user, `TERMINFO_DIRS`, and system search roots;
hashed databases are left to the narrower static profile.

A missing entry retains the static profile. A found malformed entry, wrong
alias, fewer than eight declared colors, or missing clear, absolute cursor,
cursor visibility, alternate-screen, attribute reset, or ANSI foreground/
background capabilities fails closed before any terminal mutation. Parsed
parameter strings are never executed or used for rendering, and the probe
never executes ambient `infocmp`, `tput`, or ncurses code. Evidence cannot
silently widen the supported `$TERM` families.

## Job Control And Signals

Interactive `expletives-test` observes `SIGTSTP` and `SIGCONT` on its terminal
owner loop. `SIGTSTP` performs these operations in order:

1. discard any partial decoder state and reset held/pressed human input;
2. leave the alternate screen, restore cursor and paste/keyboard modes, and
   restore the exact termios captured by `Open`;
3. send the process an uncatchable `SIGSTOP`, after terminal restoration, so
   suspension also works for an orphaned process group; and
4. remain stopped until continued by the operator, shell, or supervisor.

After continuation, the owner reacquires interactive termios and presentation
modes, queries authoritative geometry, resizes the application, and forces a
complete repaint even when the logical frame sequence did not change. A
queued `SIGCONT` notification is an idempotent safety pass. Failure to restore,
stop, reacquire, query geometry, or resize is terminal to the interactive run;
the ordinary deferred close path performs every remaining safe restoration.

Supported xterm-family enhanced-key modes may encode physical Ctrl-Z as a
structured Control-plus-`z` sequence instead of the tty's `VSUSP` byte. The
interactive terminal entry point reserves that exact physical chord and
enters the same restore, `SIGSTOP`, resume, and repaint lifecycle as a received
`SIGTSTP`. Ctrl-Alt-Z and Ctrl-Meta-Z are not job-control aliases. Attached
automation continues through the semantic application-input path and does not
silently suspend the process that owns its connection.

Headless mode does not intercept `SIGTSTP` or `SIGCONT`. `SIGWINCH` remains a
notice followed by an authoritative geometry query. `SIGINT` remains the
configurable semantic interrupt path, while `SIGTERM` and `SIGHUP` request an
orderly quit and final snapshot.

## Failure Recovery

Terminal ownership and logical activity are separate state. Once interactive
termios has been changed, the Presenter retains restoration responsibility
until both its leave sequence and original-termios restore have succeeded.
An enter, frame-write, suspend, or resume failure therefore cannot make
`Close` silently skip cleanup. A failed rollback is retried by the enclosing
open/close boundary, and independent output and termios failures are joined so
neither result hides the other.

This contract covers catchable in-process failures. No process can guarantee
cleanup after `SIGKILL`, machine loss, or an equivalent uncatchable boundary;
the controlling shell or terminal remains responsible for those cases.

## Load And Profiling Evidence

`make benchmark-terminal` runs four bounded micro-workloads: 80-by-24 and
1200-by-1200 complete-frame encoding, a 4 KiB coalesced ASCII input read, and
byte-fragmented navigation sequences. The workloads report time, throughput,
bytes, and allocations without imposing host-specific timing thresholds on
ordinary verification.

The ordinary and race suites also drive 512 complete 80-by-24 frames through
one Presenter from eight concurrent callers. The output fake accepts only
bounded partial writes, while geometry and profile reads contend for the same
owner lock; completion and exact termios restoration are required. This is a
deterministic serialization/resource test, not a claim that applications
should paint from multiple threads.

The coalesced ASCII path uses one caller-owned backing key array and one event
array. Escape, control-chord, UTF-8, and bracketed-paste input continues
through the bounded incremental state machine. This optimization does not
alter event order, pointer ownership, escape deadlines, or decoder bounds.

## Verification Matrix

Automated evidence is required at several boundaries:

| Boundary | Required evidence |
| --- | --- |
| Profile and locale | exact accepted/rejected `$TERM` tables and UTF-8/DEC selection tables |
| Encoder | ASCII, Unicode, DEC line art, highlighted degradation, colors, attributes, cursor, bounds, and canonical-frame immutability |
| Presenter | validation-before-mutation, exact termios restore, partial writes, suspend/resume, failed-rollback retry, frame-output failure, read readiness, geometry, and PTY lifecycle |
| Input | fragmented/coalesced CSI and SS3, enhanced modifiers, Escape timeout, paste, malformed/oversized input, reset, and fuzzing |
| Real process | controlling PTY resize, fragmented keys, menus, Ctrl-C, quit, direct `SIGTSTP`, enhanced-key Ctrl-Z, `SIGCONT`, full repaint, and exact final restoration |
| Physical matrix | supported emulator/profile/locale/transport combinations with operator-observed color, line art, cursor, paste, resize, job control, and teardown |

The controlling-PTY test accepts two valid supervisor behaviors: the process
remains stopped with exact restored termios until the test sends `SIGCONT`, or
the runner automatically continues it after the terminal leave transition.
In both cases, terminal re-entry and a complete repaint must follow, and final
exit must restore the original termios exactly.

Continuation handling must tolerate an ordinary job-control shell restoring
its saved canonical termios after the stopped process first wakes. The eager
resume path enters the application modes once; a repeated resume from the
queued `SIGCONT` notification reasserts interactive termios without emitting a
duplicate terminal-entry sequence. Physical input must remain immediately
usable after `fg`.

The initial physical rows to record are a native xterm-family emulator,
GNU Screen, and tmux on Linux in UTF-8 locale. At least one supported remote
transport row and one conservative non-UTF-8 row are required before Phase 19
completion. A row records emulator and version, multiplexer and version,
transport, `$TERM`, locale variables, geometry, result, observed deviations,
and the exact project revision. Secrets, hostnames, addresses, and private
operator/environment identifiers must not be recorded.

## Related Contracts

- [`Terminal Compatibility Verification`](../Terminal-Compatibility-Verification.md)
- [`Limited Unicode Support`](../Limited-Unicode-Support.md)
- [`Terminal Shortcut Compatibility`](../Terminal-Shortcut-Compatibility.md)
- [`Implementation Baseline v0`](implementation-baseline-v0.md)
- [`expletives-test`](expletives-test.md)
- [`Build And Verification`](build-and-verification.md)
