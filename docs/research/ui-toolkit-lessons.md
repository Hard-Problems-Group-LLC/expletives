# Comparative UI Toolkit Lessons

Status: Research; non-normative
Reviewed: 2026-07-24

## Purpose

This review collects transferable lessons from established GUI and terminal
UI systems. It informs later proposals for `expletives`; it does not make
those systems dependencies or compatibility targets.

The review covers Win32 common controls, wxWidgets, Xt/Motif, LessTif, modern
Turbo Vision, tcell, tview, Bubble Tea, curses, ncurses, and terminfo. The
FieldManual terminal, keyboard, automation, and Turbo Vision knacks remain the
project's maintained general guidance.

## Convergent Lessons

### Commands, Controls, And Event Routing

Several mature systems converge on a stable command between physical input
and business behavior:

- Win32 uses `WM_COMMAND` to converge menu, accelerator, and control
  activation, with `WM_NOTIFY` for richer control-specific notifications.
- wxWidgets command events can propagate through parent scopes.
- Turbo Vision routes events through groups, views, and command phases.
- curses menu and form drivers translate keys into control-owned requests.

A promising `expletives` model would preserve these stages:

```text
terminal bytes or backend event
    -> normalized Key, Text, Paste, Mouse, Resize, or Interrupt event
    -> context-sensitive binding, mnemonic, and accelerator resolution
    -> stable CommandID or widget semantic event
    -> routed handler and explicit outcome
```

Menu items, buttons, accelerators, hotkeys, and automation should converge on
the same stable command identity. Enabled, disabled, checked, help, and label
state should come from one command definition rather than drift across
display and handler tables.

Routing needs more information than `Handle(event) bool` can express. A
proposal should consider explicit phases such as application filtering,
modal capture, ancestor preprocessing, focused target, command bubbling,
application fallback, and unhandled/default processing. Events should be
immutable enough for deterministic replay and diagnostics, and results should
distinguish handled, continue, command, cancellation, and failure.

### Keyboard Layers

Keep these concepts distinct:

1. physical or backend key identity and modifiers;
2. committed Unicode text and input-method output;
3. bounded paste;
4. menu mnemonics or access keys such as `Alt-F`;
5. scope-bound accelerators such as `Ctrl-S`;
6. focus traversal and component navigation;
7. application commands; and
8. interrupt and process-control policy.

Do not reduce all input to commands before a focused editor or modal scope can
interpret it. Conversely, widgets should not receive raw terminal bytes or
backend-specific numeric constants.

Automation supports raw key lifecycle events (`KeyDown`, `KeyUp`, and
`KeyPress`) to exercise held modifiers, chords, binding, and menu resolution,
as well as direct semantic commands for precise deterministic control. These
events use stable toolkit key identities and source-local state rather than
terminal-library integers. PTY tests remain the right layer for raw escape
and signal bytes.

Essential actions need fallbacks when Alt, function keys, or modified
sequences are unavailable. Displayed shortcut text and active bindings must
come from the same structured data. Disabled commands must remain disabled
through menus, hotkeys, pointer activation, and automation alike.

### Ctrl-C And Interrupt Policy

Ctrl-C is not just another accelerator. Depending on terminal and operating
system modes, it may become a process signal before the input decoder sees a
byte, or it may arrive as ordinary input.

The project should explicitly design and test policies such as:

- native process behavior;
- graceful interrupt or cancellation;
- application delivery as a key or command;
- ignore or pass-through where meaningful; and
- optional repeated-interrupt escalation.

Signal handling should cancel or enqueue an immutable event to the UI owner;
it should not mutate widgets or call the terminal backend asynchronously.
Catchable interrupt paths must publish honest automation outcomes and restore
terminal state. Signal-preserving and raw-input modes need separate tests.

### Composition, Inheritance, And Control Reuse

Win32 subclassing, wxWidgets class hierarchies, Xt/Motif widget classes,
LessTif compatibility work, and tview's self-binding base pattern all
demonstrate that inheritance can reuse control behavior, but also expose
lifetime, chaining, initialization, and hidden-coupling costs.

The current Draft's unexported `init(self Widget)` cannot support the claimed
external implementation and re-binding model as written. Its
`ComboBox is-a ListBox` relationship also overstates substitutability; mature
toolkits commonly share item, selection, text-entry, and popup behavior
without making a combo box a list box.

A proposal should compare:

- a small public widget contract plus package-private tree plumbing;
- shallow embedding for shared default behavior;
- composable capabilities such as item model, selection model, text entry,
  popup list, scrolling, validation, and focusability; and
- deliberate factories or binding lifecycle when external extension needs
  virtual dispatch.

Avoid deep hierarchies, broad producer-owned interfaces, stringly resources,
and override rules that require every downstream constructor to repair hidden
self pointers.

### Event Loop And Concurrency

Win32, wxWidgets, Turbo Vision, tview, Bubble Tea, and curses all reinforce
one authoritative UI execution context.

- The UI owner mutates the widget tree, focus, layout, intended frame,
  terminal modes, and the backend's presented-frame estimate.
- Workers post immutable results through bounded queues.
- Async `Post` and synchronous `Call`/`CallAndFrame` behavior need explicit
  contracts.
- A synchronous call from inside the UI owner must run safely inline or fail
  clearly; enqueue-and-wait can deadlock.
- Queue overflow, coalescing, ordering, cancellation, and shutdown are product
  behavior, not incidental implementation details.
- Application callbacks should not run while internal locks are held.
- Nested event pumps or generic “yield” mechanisms introduce reentrancy and
  should not be the normal design.

Automation completion requires a rendered-frame barrier. Queue acceptance
alone, as in many async frameworks, is not evidence that a command's effect
was applied and rendered.

### Layout And Geometry

Containers should own final child geometry. Controls report minimum,
preferred, and constrained sizes; deterministic measure and arrange passes
then assign rectangles.

Xt/Motif geometry negotiation is useful history but should not be copied
literally. The Go design should avoid unbounded negotiation, cycles, implicit
storage aliasing, and layout callbacks that mutate unrelated state. Test zero,
tiny, large, and changing dimensions; changing labels and visibility;
deterministic one-cell replacement of unsupported text; clipping; modal layers;
and cyclic or unsatisfiable constraints.

### Curses, ncurses, And terminfo Context

Curses separates application windows, a virtual desired screen, and its
estimate of the physical screen. `wnoutrefresh` stages updates and `doupdate`
performs one reconciliation. Panels add explicit depth ordering; pads provide
a larger surface viewed through a viewport. These are strong precedents for:

- composing one authoritative intended frame;
- keeping backend damage and last-presented state private to the painter;
- staging all layers before one bounded flush;
- invalidating physical state after partial writes, suspension, or external
  terminal ownership; and
- separating Z-order from focus and presentation.

Terminfo is capability data selected by `$TERM`; it is not screen
virtualization. A pure-Go backend may consume terminfo without linking
ncurses. It must deliberately handle search paths, typed capabilities,
parameter expansion, padding, extended entries, fallback data, and stale or
incorrect terminal descriptions. Capability resolution belongs outside the
paint loop in an immutable backend profile.

ncurses threading remains deliberately coarse, reinforcing the single terminal
owner. Its wide-character `cchar_t` model is historical context, not a modern
grapheme contract. `expletives` deliberately supports complete grapheme
clusters only when they resolve to exactly one cell, renders each unsupported
display element as one `U+FFFD` cell, and has no continuation cells. Explicit
clipping, semantic styles, a project-owned width policy, and real-terminal
verification remain necessary.

### Rendering, Testing, And Automation

Keep one authority for each kind of state:

- semantic intended frame and widget view;
- backend-private damage and last-presented frame;
- per-observer automation sequence;
- optional bounded painter trace; and
- genuinely physical terminal evidence.

If a backend such as tcell already diffs a logical and physical screen,
decide whether `expletives` or the backend owns that reconciliation. Two
independent damage models create ambiguous failure and performance behavior.

Test at distinct seams:

1. pure widget, layout, command-routing, and semantic-cell tests;
2. deterministic headless sessions;
3. backend simulation or recording painter tests;
4. `expletives-test` catalog and attached automation;
5. PTY lifecycle/input/output tests; and
6. a supported real-terminal matrix.

The test environment should inject geometry, capability profiles, clocks,
signals, scheduler events, and fixtures. A correctness-first full-render
reference path remains valuable even if normal presentation uses damage
diffing.

## Lessons Not To Copy Blindly

- deep control hierarchies with obscure override chaining;
- mutable events cleared to indicate handling;
- stringly typed resources or actions without validation;
- direct worker mutation of UI state;
- blocking cross-thread message sends;
- callbacks invoked while internal locks are held;
- nested main loops used as routine control flow;
- implicit focus-driven Z-order;
- global handlers that swallow text or navigation;
- fixed-size character containers presented as Unicode grapheme support; and
- treating one successful emulator, terminfo entry, or visual snapshot as a
  portability guarantee.

LessTif's release history is particularly useful negative evidence: geometry,
focus traversal, accelerators, grabs, and menus generated repeated
compatibility regressions. The lesson is to build feature-focused automated
coverage and a comprehensive interactive catalog, not to recreate Motif.

## Questions For Proposals

This research should inform, but does not answer:

- the public widget and extension model;
- capture/target/bubble routing and structured handler results;
- the normalized input and stable command schemas;
- mnemonic, accelerator, hotkey, and interrupt precedence;
- sync/async UI-call and rendered-frame-barrier contracts;
- the canonical semantic frame and backend damage owner;
- the terminal capability strategy and fallback support; and
- the exact `expletives-test` catalog completeness mechanism.

These questions are tracked in
[`project-management/open-questions.md`](../../project-management/open-questions.md).

## Primary Sources

### Win32

- [Messages and message queues](https://learn.microsoft.com/en-us/windows/win32/winmsg/about-messages-and-message-queues)
- [`WM_COMMAND`](https://learn.microsoft.com/en-us/windows/win32/menurc/wm-command)
- [`WM_NOTIFY`](https://learn.microsoft.com/en-us/windows/win32/controls/wm-notify)
- [Subclassing controls](https://learn.microsoft.com/en-us/windows/win32/controls/subclassing-overview)
- [Keyboard accelerators](https://learn.microsoft.com/en-us/windows/win32/menurc/about-keyboard-accelerators)
- [Menus and access keys](https://learn.microsoft.com/en-us/windows/win32/menurc/about-menus)
- [`SendMessage` threading behavior](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendmessage)
- [Console control handlers](https://learn.microsoft.com/en-us/windows/console/setconsolectrlhandler)

### wxWidgets

- [Event architecture](https://docs.wxwidgets.org/3.2/overview_events.html)
- [Threading overview](https://docs.wxwidgets.org/3.2/overview_thread.html)
- [`wxEvtHandler`](https://docs.wxwidgets.org/3.2/classwx_evt_handler.html)
- [`wxAcceleratorTable`](https://docs.wxwidgets.org/3.2/classwx_accelerator_table.html)
- [Mnemonic tutorial](https://wxwidgets.org/docs/tutorials/using-mnemonics/)
- [`wxComboBox`](https://docs.wxwidgets.org/3.2/classwx_combo_box.html)

### Xt, Motif, And LessTif

- [X Toolkit Intrinsics specification](https://www.x.org/releases/current/doc/libXt/intrinsics.pdf)
- [Motif resource and inheritance documentation](https://docs.oracle.com/cd/E19205-01/819-3700/Resources_1.html)
- [Motif widget reference](https://docs.oracle.com/cd/E19422-01/819-3700/WidgetReference_1.html)
- [Official Motif source](https://sourceforge.net/p/motif/code/ci/master/tree/)
- [LessTif project](https://lesstif.sourceforge.net/)
- [Inside LessTif](https://lesstif.sourceforge.net/InsideLessTif/index.html)
- [LessTif release and regression history](https://lesstif.sourceforge.net/ReleaseNotes.html)

### Terminal Toolkits And Protocol Context

- [Modern Turbo Vision](https://github.com/magiblot/tvision)
- [tcell](https://github.com/gdamore/tcell)
- [tview](https://github.com/rivo/tview)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [ncurses refresh model](https://invisible-island.net/ncurses/man/curs_refresh.3x.html)
- [ncurses windows](https://invisible-island.net/ncurses/man/curs_window.3x.html)
- [ncurses panels](https://invisible-island.net/ncurses/man/panel.3x.html)
- [terminfo](https://invisible-island.net/ncurses/man/terminfo.5.html)
- [ncurses input decoding](https://invisible-island.net/ncurses/man/curs_getch.3x.html)
- [ncurses resize handling](https://invisible-island.net/ncurses/man/resizeterm.3x.html)
- [ncurses threading constraints](https://invisible-island.net/ncurses/man/curs_threads.3x.html)
- [Go signal handling](https://pkg.go.dev/os/signal)
