# Terminal Compatibility Verification

- Status: Phase 19 physical matrix complete
- Contract: [`Terminal Compatibility v0`](specifications/terminal-compatibility-v0.md)
- Tested product revision: `a402cf0763580328d9ef08d6ee747017ceef7e93`
- Toolchain: Go 1.26.5, Linux amd64
- Evidence completed: 2026-08-10

## Purpose And Claim Boundary

This record separates deterministic PTY/protocol evidence from behavior that
only a real emulator, multiplexer, transport, font, and terminal interaction
path can prove. A successful row applies only to the exact profile recorded
here. It does not promote the toolkit to generic ANSI, curses, ncurses,
emulator, code-page, or remote-transport compatibility.

Do not record hostnames, usernames, addresses, Wi-Fi names, terminal socket
paths, authentication material, or screenshots containing unrelated windows.
Temporary captures belong under `.local/tmp/`. Capture a specifically owned
test window, not the complete desktop.

## Automated Mechanical Evidence

Clean product revision `a402cf076358` passed the complete `make verify` gate,
including ordinary and Unix-socket integration tests, controlling-PTY
lifecycle tests, the race suite, all three build modes, catalog self-checks,
and smoke coverage. The controlling-PTY suite verifies:

- exact interactive termios acquisition and restoration;
- fragmented navigation and enhanced-key input, menus, Ctrl-C, and Ctrl-Z;
- authoritative resize and complete repaint;
- safe `SIGTSTP`/`SIGCONT` leave, stop, raw-mode reacquisition, and repaint;
- post-`fg` input after a job-control shell restores canonical termios; and
- final leave sequences and exact termios restoration on normal and interrupt
  exits.

The ordinary/race suite separately covers bounded input, paste, cleanup-debt
retries, output failures, partial writes, concurrent Presenter serialization,
and terminal-profile/terminfo rejection before mutation. These tests prove
mechanics and byte streams; they do not replace the physical rows below.

The phase-close run completed all four ten-iteration
`make benchmark-terminal` workloads, including the 1200-by-1200 encoder case.
Independent ten-second runs of `FuzzInputDecoder` and `FuzzParseTerminfo`
passed without a failing input. Execution counts and timings are intentionally
omitted because they are host-, scheduler-, and corpus-dependent rather than
compatibility thresholds.

## Completed Physical Matrix

Every row used the release `expletives-test` binary reporting revision
`a402cf076358` with `modified=false`. The exact owned XTerm window was captured
where visual inspection was required. Each row exercised terminal-native
input rather than injecting semantic automation events; attached automation
was used only to inspect the resulting state.

| Row | Emulator / multiplexer | Transport | `$TERM` and locale | Geometry | Result |
| --- | --- | --- | --- | --- | --- |
| Native UTF-8 | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `LANG=en_US.UTF-8` | 120x30, resized to 100x24 | pass |
| tmux UTF-8 | XTerm 366 outside tmux 3.2a; tmux `default-terminal=screen` | local graphical session | `screen`; `LANG=en_US.UTF-8` | 120x29 pane, resized to 100x23 | pass |
| GNU Screen UTF-8 | XTerm 366 outside GNU Screen 4.8.0; `altscreen on` | local graphical session | `screen.xterm-256color`; `LANG=en_US.UTF-8` | 120x30, resized to 100x24 | pass with required Screen setting |
| Conservative locale | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `LC_ALL=C`, `LC_CTYPE=C`, `LANG=C` | 120x30, resized to 100x22 | pass |
| OpenSSH UTF-8 | XTerm 366; no multiplexer | OpenSSH 9.9 TCP loopback with an allocated remote PTY | `xterm-256color`; `LANG=en_US.UTF-8` | 120x30, resized to 100x22 | pass within the stated transport boundary |

The native, tmux, GNU Screen, and OpenSSH rows showed continuous single and
double line art; distinct light, medium, dark, and full-cell shades; correct
menu, chrome, mnemonic, and right-aligned Help presentation; and the accepted
MessageBox palette, button, button shadow, outer shadow, spacing, and clipping.

In every row, the terminal's ordinary paste mechanism inserted the fixed test
text into the focused TextField, and a subsequent resize retained both the
text and focus/edit state. Ctrl-Z returned to a usable canonical/echoing shell
with the caller-visible termios restored exactly. `fg` reacquired raw mode and
forced a complete repaint; Alt-F then opened the File menu immediately. Ctrl-C
while that menu was open followed the configurable interrupt path, exited with
status 130, and restored the caller-visible termios byte for byte.

The conservative-locale content screen additionally proved the specified
projection boundary:

- definite ASCII rendered directly;
- DEC Special Graphics supplied the available closest line art;
- structural shade and block forms used aligned `#` fallbacks;
- unsupported ordinary Unicode used conspicuous black-on-yellow `?` cells;
  and
- no fallback changed row or column alignment.

### GNU Screen Prerequisite

GNU Screen must have alternate-screen preservation enabled with:

```text
altscreen on
```

The tested package's stock configuration left this disabled. In that state,
Screen advertised alternate-screen capabilities through its `$TERM` profile
but did not retain a separate alternate buffer, so application cells remained
behind the returned shell prompt. Enabling `altscreen` and repeating the run
produced a clean restored shell. A Screen profile without this setting is not
a supported clean-teardown configuration.

### OpenSSH Evidence Boundary

The OpenSSH row used the actual OpenSSH client/server transport, TCP framing,
PTY allocation, terminal-size propagation, and terminal byte stream with an
isolated temporary key-only server. Loopback was used deliberately so no
private endpoint or network was placed in scope. This proves the listed SSH
transport path; it does not claim behavior under real-network latency,
disconnects, roaming, mosh, serial bridges, or unrelated remote profiles.

## Phase 19 Result

The required native xterm-family, tmux, GNU Screen, conservative non-UTF-8,
and supported remote-transport rows are complete on one exact release
revision. Together with the automated mechanical, race, resource, benchmark,
and fuzz evidence, this closes the Phase 19 physical-support gate for the
narrow profiles and prerequisites recorded above.

Historical observations from earlier revisions remain useful diagnostic
context but are not needed to support this claim. New emulator, multiplexer,
transport, locale, operating-system, or `$TERM` combinations require a new
row; they must not be inferred from these results.

## Repeatable Operator Procedure

Build the exact revision first:

```text
make build
```

Run `build/debug/expletives-test` in the terminal/profile being evaluated.
Use `--automation <absolute-socket-path>` only when closed-loop semantic
inspection is useful and the unauthenticated local endpoint is acceptable.
Keep the socket in a unique child of `.local/tmp/` and remove only that owned
workspace when finished.

Perform this checklist with the terminal's actual keyboard and paste path:

1. Open Panels -> Visual Styles and inspect every border/shade form and color.
2. Open Dialogs -> Message Box and inspect body, border/title, button, both
   shadows, spacing, and close behavior.
3. Navigate menus with Alt-F, F9, arrows, Enter, mnemonics, and Escape.
4. Edit a TextField with normal text, Backspace, Enter, and terminal paste.
5. Inspect one-cell Unicode and highlighted fallback content.
6. Resize in both axes and check repaint, cursor, chrome, and stale cells.
7. Exercise Ctrl-C under the selected policy; the test app defaults to an
   orderly exit with status 130.
8. Suspend with Ctrl-Z, inspect the shell, resume with `fg`, verify immediate
   input and repaint, then quit and compare the shell's termios and screen.

Record emulator and version, multiplexer and version, transport class,
`$TERM`, locale variables, geometry, toolkit revision, result, prerequisites,
and specific deviations. Do not generalize beyond the exact row.
