# Terminal Compatibility Verification

- Status: Phase 19 active evidence and operator runbook
- Contract: [`Terminal Compatibility v0`](specifications/terminal-compatibility-v0.md)
- Current checkout and automated-evidence revision: `098195361588`
- Historical physical-row baseline revision: `778518ef251d`
- Toolchain: Go 1.26.5, Linux amd64
- Evidence refreshed: 2026-08-08

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

At the current automated-evidence revision above, the controlling-PTY
integration suite verifies:

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

The clean source tree committed as `098195361588` passed the complete `make
verify` gate after publication, including ordinary, Unix-socket integration,
PTY lifecycle, race, all-mode build, catalog self-check, and smoke coverage.
That commit changes only this evidence and its project-management records, so
compiled product source is identical to `5e8279c0d54c`. On that code-equivalent
parent, `make benchmark-terminal` completed all four ten-iteration workloads,
including the 1200-by-1200 encoder case. Ten-second bounded runs of
`FuzzInputDecoder` and `FuzzParseTerminfo` also passed without a failing input
after 64,065 and 608 executions respectively. Those counts are host- and
corpus-dependent and are not compatibility thresholds.

## Acquired Physical And Live Attached Evidence

The rows in this section are retained as a historical baseline. Their test
binary contained product code from `778518ef251d`; at acquisition time, the
checkout had only subsequent verification-test and documentation edits.
Product and catalog source have changed since then, so these rows do not
certify the current revision. Exact-window captures were inspected and then
left only in ignored `.local/tmp/` workspaces.

| Row | Evidence revision | Emulator / multiplexer | Transport | `$TERM` and active locale | Geometry | Result |
| --- | --- | --- | --- | --- | --- | --- |
| Native UTF-8 | `778518ef251d` | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `C.UTF-8` | 100x30 | historical visual pass; human physical-key pass pending |
| Multiplexer UTF-8 | `778518ef251d` | XTerm 366 outside tmux 3.2a; xterm client; tmux `default-terminal=screen` | local graphical session | `screen`; `C.UTF-8` | 100x29 pane plus one tmux status row | historical visual pass; human physical-key/pass-through pass pending |
| Conservative locale | `778518ef251d` | XTerm 366; no multiplexer | local graphical session | `xterm-256color`; `LC_ALL=C`, `LC_CTYPE=C`, `LANG=C` | 100x30 | historical visual fallback pass; human physical-key pass pending |

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

### Current Live Attached Row

On 2026-08-08, a release-mode live instance reported product revision
`5e8279c0d54c` and owned a controlling PTY. Its VCS modified marker covered
only the three documentation/project-management records later committed as
`098195361588`; no compiled product source differed from either revision.

| Emulator / multiplexer | Transport | `$TERM` and active locale | Geometry | Result |
| --- | --- | --- | --- | --- |
| Terminator 2.1.5 with VTE 0.78.6; no multiplexer | local graphical session | `xterm-256color`; `LANG=en_US.UTF-8`; `COLORTERM=truecolor` | 137x19 | current-revision live attached canonical/controller, operator rendering, and physical-key interaction pass; paste/resize/job-control/teardown pending |

Attached automation drove frames 73 through 110 through the same semantic
controller used by human input:

- Visual Styles exposed complete no-frame, single, double, light, medium,
  dark, and full-cell canonical line art without clipping at 137x19.
- MessageBox resolved its body and static text as `#FFFFFF` on `#808080`, its
  double border as white on the same body, and its focused Button as black on
  green. Its 126-by-8 bounds, two blank rows above the Button, shadow, and
  immediate bottom border remained intact.
- Raw logical Alt-F, F9, Right, Down, and Escape events opened, traversed, and
  closed the expected menu paths with explicit applied outcomes.
- Tab reached the plain TextField. Enter, one character, Backspace, and Escape
  exercised then cancelled a reversible edit, restoring `Edit me` outside edit
  mode.
- Scrolling/Content retained one-cell alignment, rendered fifteen canonical
  `U+FFFD` cells in the bounded StreamView fixture, moved MarkdownView from
  offset `(0,0)` to `(0,8)` with Page Down, and returned it with Home.
- The instance was restored to Home at frame 110 with no open menu, active
  modal, held input, or focused control.

This evidence proves the live current-revision application/controller and
canonical intended-frame path in a real xterm-profile process. The operator
observed the driven sequence and confirmed that rendering and physical keyboard
interaction appeared correct. This closes those portions of the native row,
including the earlier current-revision acceptance of the two dialog changes.
Attached automation still cannot prove paste, resize, job-control signals, or
final teardown through the emulator and shell; those remain explicitly
pending.

### Current Locale Projection Comparison

On 2026-08-10, clean release revision `098195361588` ran twice under the same
Terminator 2.1.5/VTE 0.78.6, no-multiplexer, `xterm-256color`, 137x19 local
profile:

- With `LC_ALL=C`, `LC_CTYPE=C`, and `LANG=C`, the operator observed and
  accepted the conservative structural projection: DEC closest-line art made
  the requested single and double forms appear as the available single-line
  repertoire, while light, medium, dark, and full-cell borders used aligned
  `#` glyphs with their configured presentation retained.
- After restart with the ordinary `LANG=en_US.UTF-8` environment, the operator
  observed and accepted distinct single, double, `░`, `▒`, `▓`, and `█`
  rendering on the same Visual Styles screen.
- Attached automation confirmed that both instances continued to request the
  exact distinct canonical border forms. The UTF-8 instance was restored to
  Home at frame 85 with no open menu, modal, held input, or focus.

The `C`-locale instance was restarted before its ordinary unsupported-text
fixture could be displayed. Current-revision physical confirmation of the
black-on-yellow ASCII/`?` content fallback therefore remains open; the
structural comparison itself is complete.

## Rows Still Required For Phase 19

| Required row | Current state | Completion evidence |
| --- | --- | --- |
| Native xterm-family, current revision | live attached canonical/controller, current-revision frontend rendering, and code-equivalent physical-key interaction pass on Terminator/VTE; completion pending | bracketed paste, resize, Ctrl-C, suspend/continue, and clean teardown |
| GNU Screen, current revision | pending; `screen` is not installed in the local agent environment | version/profile plus visual and operator-interaction checklist |
| tmux, current revision | pending refresh | complete visual checklist plus actual keyboard/paste/resize/job-control pass-through |
| Conservative non-UTF-8 locale, current revision | structural fallback rendering pass on Terminator/VTE; completion pending | ordinary black-on-yellow content fallback plus physical key/paste, resize, Ctrl-C, suspend/continue, and clean teardown |
| Supported remote transport, current revision | pending | one real SSH, mosh, serial, or other declared transport row preserving the supported profile byte stream |

These rows block only the Phase 19 physical-support claim. They do not erase
the current automated evidence or the historical native, tmux, and
conservative-locale observations already acquired.

The 2026-08-08 local agent command runner had no controlling TTY. It had tmux
3.4 and an SSH client, but no xterm, GNU Screen, or mosh installation. The
operator-owned live instance supplied the current native-profile attached
evidence above, but a nested PTY, tmux server, or SSH client-version check from
the agent runner would only duplicate mechanical evidence; none can substitute
for the remaining operator-observed physical behavior.

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
