# Terminal Compatibility Verification

- Status: Phase 19 active evidence and operator runbook
- Contract: [`Terminal Compatibility v0`](specifications/terminal-compatibility-v0.md)
- Current product-code revision: `778518ef251d`
- Toolchain: Go 1.26.5, Linux amd64

## Purpose And Claim Boundary

This record separates deterministic PTY/protocol evidence from things that
only a real terminal, multiplexer, transport, font, and operator can prove.
A successful row applies only to the exact profile recorded here. It does not
promote the toolkit to generic ANSI, curses, ncurses, emulator, code-page, or
remote-transport compatibility.

Do not record hostnames, usernames, addresses, Wi-Fi names, terminal socket
paths, or screenshots containing unrelated windows. Temporary captures belong
under `.local/tmp/` and are removed after their observations have been
recorded. Capture a specifically owned test window, not the complete desktop.

## Automated Mechanical Evidence

At the revision above, the controlling-PTY integration suite verifies:

- exact interactive termios acquisition and restoration;
- fragmented navigation input, Alt-F, F9, arrows, Enter, Escape, Ctrl-Space,
  nested menu mnemonics, Alt-X, and Ctrl-C;
- authoritative resize and complete repaint;
- safe `SIGTSTP`/`SIGCONT` leave, stop, reacquire, and repaint behavior;
- an attached automation-only screen mutation producing a later physical PTY
  frame without any terminal input; and
- final leave sequences and exact termios restoration on normal and interrupt
  exits.

The ordinary/race suite separately covers bounded input, paste, cleanup-debt
retries, output failures, partial writes, concurrent Presenter serialization,
and terminal-profile/terminfo rejection before mutation. These tests prove
mechanics and byte streams; they do not replace the visual rows below.

## Acquired Physical Rows

The test binary contained product code from `778518ef251d`. The checkout had
only subsequent verification-test and documentation edits; no product source
used by the binary differed from that revision. Exact-window captures were
inspected and then left only in ignored `.local/tmp/` workspaces.

| Row | Emulator / multiplexer | Transport | `$TERM` and active locale | Geometry | Result |
| --- | --- | --- | --- | --- | --- |
| Native UTF-8 | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `C.UTF-8` | 100x30 | visual pass; human physical-key pass pending |
| Multiplexer UTF-8 | XTerm 366 outside tmux 3.2a; xterm client; tmux `default-terminal=screen` | local graphical session | `screen`; `C.UTF-8` | 100x29 pane plus one tmux status row | visual pass; human physical-key/pass-through pass pending |
| Conservative locale | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `LC_ALL=C`, `LC_CTYPE=C`, `LANG=C` | 100x30 | visual fallback pass; human physical-key pass pending |

Observed native UTF-8 behavior:

- menu labels, red mnemonics, right-aligned Help, status, and Footer rows were
  aligned and colored distinctly;
- single and double borders had continuous corners and edges;
- light, medium, dark, and full-cell structural shades were distinguishable;
- the MessageBox border/title were white, its `#808080` body was visibly
  darker than the `#AAAAAA` menu surface, and its green button and black
  button shadow remained distinct; and
- the dialog, underlying screen, application chrome, and shadows did not
  clip or corrupt each other at 100x30.

The tmux row preserved the same line art, shades, dialog colors, button,
shadow, and clipping behavior inside the 100x29 pane. The tmux-owned status
row remained outside the application's physical surface, as required.

The conservative `C`-locale row showed DEC Special Graphics for single-line
and closest-line double borders. Toolkit-owned shade/block borders used the
documented structural `#` fallback while preserving their configured styles.
Ordinary unsupported Unicode in Markdown/stream fixtures used conspicuous
black-on-yellow one-cell `?` approximations; definite ASCII remained direct.

## Rows Still Required For Phase 19

| Required row | Current state | Completion evidence |
| --- | --- | --- |
| Native xterm-family operator interaction | pending | actual keyboard navigation, paste, resize, Ctrl-C, suspend/continue, and clean teardown on the native UTF-8 row |
| GNU Screen | pending; `screen` is not installed in the current environment | version/profile plus visual and operator-interaction checklist |
| tmux operator interaction | pending | actual keyboard/paste/resize/job-control pass-through on the acquired tmux visual row |
| Supported remote transport | pending | one real SSH, mosh, serial, or other declared transport row preserving the supported profile byte stream |

These rows block only the Phase 19 physical-support claim. They do not erase
the automated, native visual, tmux visual, or conservative-locale evidence
already acquired.

## Operator Procedure

Build the exact revision first:

```text
make build
```

Run `build/debug/expletives-test` in the terminal/profile being evaluated.
Use `--automation <absolute-socket-path>` only when closed-loop semantic
inspection is useful and the unauthenticated local endpoint is acceptable.
Keep the socket in a unique child of `.local/tmp/` and remove only that owned
workspace when finished.

Perform this checklist with the actual terminal keyboard and paste mechanism:

1. Open Panels -> Visual Styles. Verify no-frame, single, double, light,
   medium, dark, and full-cell forms, including all corners and the configured
   colors.
2. Open Dialogs -> Message Box. Verify the dark-gray body is visibly darker
   than the menu, the border/title are white, the button and its shadow are
   distinct, and Enter/Escape close it correctly.
3. Navigate menus with Alt-F, F9, arrows, Enter, sibling mnemonics, and Escape.
   Confirm the host emulator does not consume the documented safe defaults.
4. Open Controls -> Text / Numeric Input and use Tab, Shift-Tab, normal text,
   validation fixtures, Backspace, Enter, and bracketed paste.
5. Open Controls -> Scrolling / Content and inspect one-cell Unicode,
   replacement/fallback highlighting, line alignment, and scroll keys.
6. Resize narrower, wider, shorter, and taller. Verify complete repaint,
   cursor placement, no stale cells, and correct application chrome.
7. Exercise Ctrl-C under the selected application policy. The default test
   app exits with status 130 after an orderly terminal leave.
8. Suspend with Ctrl-Z, inspect the returned shell, resume with `fg`, and
   verify the app reacquires and fully repaints. Quit normally and confirm the
   shell's echo, canonical input, cursor, paste, and alternate-screen state are
   restored.

Record emulator and version, multiplexer and version, transport class,
`$TERM`, locale variables, geometry, toolkit revision, result, and specific
deviations. Do not generalize beyond the exact row.
