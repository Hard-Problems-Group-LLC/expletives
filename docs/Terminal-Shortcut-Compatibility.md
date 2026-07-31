# Terminal Shortcut Compatibility Advisory

Status: project advisory

Applies to: toolkit defaults, `expletives-test`, examples, and acceptance tests

Last reviewed: 2026-07-30

## Purpose

A terminal emulator receives keyboard input before the application attached
to its pseudoterminal. If the emulator, desktop, input method, or remote
client consumes a chord, expletives cannot detect or recover that event.
Project defaults therefore avoid shortcuts used by the common terminal
families on supported RHEL, Fedora, and Ubuntu desktops.

This advisory governs project-selected defaults. The toolkit must continue to
support client-selected raw `KeyDown`, `KeyUp`, and `KeyPress` chords,
including bindings that an application deliberately chooses after accounting
for its deployment environment.

## Audited Terminal Families

The current distribution families are not uniform:

- RHEL 10 replaces GNOME Terminal with Ptyxis. RHEL 9 installations and
  upgraded or customized systems can still use GNOME Terminal.
- Fedora Workstation 41 and later use Ptyxis by default. Earlier releases,
  upgrades, spins, and customized systems can still use GNOME Terminal or
  another emulator.
- Ubuntu 26.04 LTS uses Ptyxis by default, while Ubuntu 24.04 LTS uses GNOME
  Terminal. Upgrades and flavors can retain or select another emulator.
- Terminator is an explicitly supported, commonly selected alternative and
  has its own larger shortcut namespace.

The audit covered upstream documentation and defaults for Terminator 2.1,
Ptyxis 48, and current GNOME Terminal. Local user configuration, desktop
extensions, remote-terminal clients, multiplexers, and input methods can add
collisions that no static project list can exhaust.

## Default Bindings To Avoid

Do not assign these as the sole or preferred project-wide application
shortcuts:

| Chord or family | Host use | Policy |
| --- | --- | --- |
| `F10` | Ptyxis primary menu; usual GNOME/GTK menu accelerator | Reserved |
| `Shift-F10` | Ptyxis terminal popup menu; common context-menu key | Reserved |
| `F11` | Fullscreen in Ptyxis, GNOME Terminal, and Terminator | Reserved |
| `F1` | Help in Terminator and common desktop applications | Reserved |
| `Alt-L` | Terminator Layout Launcher | Reserved |
| `Alt-F`, `Alt-E`, `Alt-V`, `Alt-S`, `Alt-T`, `Alt-H` | Conventional GNOME Terminal menubar mnemonics when its menubar and mnemonics are enabled | Avoid as application defaults |
| `Alt-0` through `Alt-9` | Tab selection in Ptyxis and GNOME Terminal | Reserved |
| `Alt-Arrow` | Pane navigation in Terminator | Reserved |
| `Ctrl-PageUp`, `Ctrl-PageDown` | Tab navigation in Ptyxis and GNOME Terminal | Reserved |
| `Ctrl-Shift-PageUp`, `Ctrl-Shift-PageDown` | Tab movement in Ptyxis and GNOME Terminal | Reserved |
| `Ctrl-+`, `Ctrl--`, `Ctrl-0` | Terminal font zoom | Reserved |
| `Ctrl-Shift-C`, `Ctrl-Shift-V`, `Ctrl-Shift-T`, `Ctrl-Shift-N`, `Ctrl-Shift-W`, `Ctrl-Shift-Q`, `Ctrl-Shift-F` | Common terminal copy, paste, tab, window, close, and search operations | Reserved |

Treat the broader `Ctrl-Shift-<letter>` namespace cautiously: Ptyxis,
GNOME Terminal, and Terminator use many such chords, and the exact set changes
with version and user configuration. Prefer an audited exact binding over a
new default in that family.

Desktop and input-method shortcuts are a separate interception layer. In
particular, do not make `Alt-F2`, `Alt-F4`, `Ctrl-Alt-T`, `Super` chords, or
an input-method-sensitive chord the only route to an essential action.
`Ctrl-Space` remains an expletives menu fallback because it is not claimed by
the audited terminal defaults, but it is secondary to `F9` and may be claimed
by an input method. Menu traversal and exact top-level mnemonics provide
additional routes.

`Ctrl-C` is an intentional exception. It is the terminal interrupt convention,
not an application accelerator. Expletives handles its raw input and signal
paths under the configurable interrupt contract rather than reassigning it.

## Expletives Defaults

The MenuBar activation defaults are:

| Purpose | Default |
| --- | --- |
| Activate the first MenuBar label | `F9` |
| Alternate MenuBar activation | `Ctrl-Space` |
| Open a top-level popup directly | its displayed exact `Alt` mnemonic |

`expletives-test` uses this collision-audited top-level catalog:

| Displayed label | Marked form | Direct chord |
| --- | --- | --- |
| File | `&File` | `Alt-F` |
| Panels | `Pa&nels` | `Alt-N` |
| Layouts | `L&ayouts` | `Alt-A` |
| Controls | `&Controls` | `Alt-C` |
| Sections | `&Sections` | `Alt-S` |
| Menus | `&Menus` | `Alt-M` |
| Dialogs | `&Dialogs` | `Alt-D` |
| Help | `Hel&p` | `Alt-P` |

`Alt-S` for the operator-directed `&Sections` label is an intentional
exception: a visible GNOME Terminal menubar can intercept it. Sections
therefore remains fully reachable through F9 or Ctrl-Space followed by MenuBar
navigation; `Alt-S` is not its only route.

Its essential non-menu routes remain redundant:

| Purpose | Preferred visible hint | Other routes |
| --- | --- | --- |
| Quit | `Alt-X` | File/Quit, `q`, `Escape`, cancel Button |
| Toggle fixture | `Ctrl-R` | Controls/Actions Button |
| Interrupt | `Ctrl-C` | configured signal/input policy |

`Alt-X` is the conventional Turbo Vision exit hint and is not claimed by the
audited terminal defaults. When it is forwarded, its registered global
binding takes precedence over a screen-local control or tab mnemonic; only a
displayed top-level MenuBar mnemonic has earlier Alt-key handling. No
essential operation depends on it being forwarded.

The red Turbo Vision-style mnemonic rendering follows the marked character.
Popup-item mnemonics remain unmodified keys while their popup owns menu
navigation; they are not bare `Alt` chords intercepted by the host.

The toolkit does not reject reserved chords in client-created menus or
bindings. Such a restriction would prevent deployment-specific configuration
and raw-key testing. Examples, automated acceptance scenarios, and product
defaults must use the safe set unless a specification records an intentional
exception and supplies another usable route.

## Selection Checklist

Before adding or changing a default shortcut:

1. Check this advisory and the current upstream defaults for every supported
   terminal family.
2. Check the target desktop, window manager, input method, multiplexer, and
   remote client when the binding is deployment-specific.
3. Provide a visible or navigable alternate route for essential behavior.
4. Test through an attached controlling PTY, not only structured input
   injection. Structured automation proves toolkit routing but cannot prove
   that a real host emulator forwards the bytes.
5. Exercise equivalent raw automation `KeyDown`, `KeyPress`, and `KeyUp`
   events so modifier lifecycle and command parity remain observable.

Terminator users can change or delete a host binding in Preferences →
Keybindings; Backspace deletes the selected binding. That is a useful local
override, but project defaults do not require users to modify their emulator.

## Sources

- [Terminator Layout Launcher defaults](https://gnome-terminator.readthedocs.io/en/latest/layouts.html)
- [Terminator keybinding configuration](https://gnome-terminator.readthedocs.io/en/latest/preferences.html#keybindings)
- [GNOME Terminal shortcut defaults](https://teams.pages.gitlab.gnome.org/Websites/help.gnome.org/gnome-terminal/adv-keyboard-shortcuts.html)
- [GNOME Terminal menubar mnemonics and accelerator](https://help.gnome.org/gnome-terminal/pref-keyboard-access.html)
- [Ptyxis 48.5 shortcut schema](https://sources.debian.org/src/ptyxis/48.5-1~deb13u1/src/org.gnome.Ptyxis.gschema.xml.in)
- [RHEL 10 replacement of GNOME Terminal with Ptyxis](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/10/html/10.0_release_notes/removed-features)
- [Fedora Ptyxis default-terminal QA case](https://fedoraproject.org/wiki/QA%3ATestcase_Ptyxis)
- [Ubuntu 26.04 LTS terminal change](https://documentation.ubuntu.com/release-notes/26.04/summary-for-lts-users/#new-terminal-emulator)
- [Ubuntu default-terminal configuration](https://documentation.ubuntu.com/desktop/en/latest/how-to/change-the-default-terminal/)
