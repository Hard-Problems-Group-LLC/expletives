# Product Goals

Status: Active project direction
Authority: Direct operator requests on 2026-07-24
Last updated: 2026-07-30

## Purpose

This document states what the project is trying to produce and how its
deliverables fit together. Detailed behavior belongs in
[`specifications/`](specifications/README.md), while delivery state and design
decisions belong in [`project-management/`](../project-management/README.md).

## Product Hierarchy

### Primary Product: `expletives`

`expletives` is a reusable Go terminal user-interface toolkit inspired
by Turbo Vision and curses, but designed around modern terminal behavior,
semantic cell frames, accessibility, deterministic testing, and serious
automation.

The library should provide:

- a coherent cell-based rendering and composition model;
- one-cell Unicode rendering with composed one-cell graphemes and deterministic
  `U+FFFD` replacement for unsupported widths;
- conservative basic-terminal presentation that directly emits only definite
  ASCII or known code-page mappings and highlights approximated ASCII output
  in black on yellow;
- a broad common-controls library;
- semantic input, styles, widget identity, and observable state;
- structured menus, accelerators, mnemonics, hotkeys, and configurable
  interrupt handling;
- deterministic headless rendering and interaction;
- thread-safe public behavior for multithreaded consuming applications, with
  deterministic serialized ownership where UI work requires it;
- safe terminal lifecycle and one-owner input/output discipline;
- native automation interfaces that do not depend on parsing ANSI output; and
- a stable, documented Go API suitable for use by other projects.

### Secondary Product: `expletives-test`

`expletives-test` is a supported, human-runnable interactive application that
uses `expletives` as a real consumer. It is not a disposable demo. It must
exercise every public UI element and the important states, interactions,
layouts, degradation paths, and lifecycle behavior of the toolkit.

A normal invocation is a human-driven TUI. An explicit per-process
`--automation <socket-path>` option enables the attached drive-and-observe
interface described in
[`specifications/expletives-test.md`](specifications/expletives-test.md).
Without that option, the application must not create automation listeners,
workers, discovery records, capture paths, or other control-plane artifacts.

The application is the shared closed-loop diagnostic surface for maintainers,
AI collaborators, and human operators:

1. a human can run the application and report an observed problem;
2. a client with access to the operator-enabled endpoint can drive the same
   running UI with raw key lifecycle events or direct semantic commands and
   retrieve the associated intended frames and semantic state;
3. developers can compare those frames with the report, reproduce the
   transition headlessly, and add a regression test; and
4. a normal Go debugger can attach when state, concurrency, or terminal
   adaptation requires source-level diagnosis.

### Supporting Product: `expletivesctl`

`expletivesctl` is the supported command-line client for the explicit
attached-automation endpoint. It uses the reusable Go client package to
observe snapshots, submit raw key lifecycle events and direct semantic
commands, inspect completion, and request orderly exit.

Like every supported executable target, it must build in debug, release, and
profiling modes under `build/<mode>/expletivesctl`.

## Engineering Goals

1. **Correct model first.** Treat the UI as domain and interaction state
   rendered into an intended character-cell frame, not as decorated strings.
2. **Automation-native behavior.** Human input, headless tests, and attached
   automation use the same controller and renderer. Automation can inject raw
   `KeyDown`, `KeyUp`, and `KeyPress` lifecycle events before shortcut
   resolution, as well as direct semantic commands.
3. **Deterministic evidence.** Frames and semantic views are immutable,
   atomic, bounded, and suitable for structural assertions and diagnosis.
4. **Real terminal quality.** Rendering, input, resize, the versioned one-cell
   Unicode width boundary, accessibility, suspension, and teardown are
   verified at the physical terminal boundary as well as headlessly.
5. **Go-community verification.** Use ordinary Go unit and integration tests
   wherever practical, supplemented by headless sessions, protocol and input
   fuzzing, race detection, PTY tests, supported real-terminal checks, and
   attached-automation tests.
6. **Reproducible developer builds.** Every current or future project, test,
   automation, or diagnostic executable builds in debug, release, and
   profiling modes under `build/<mode>/<name>` through the root `Makefile`.
7. **Public-consumer pressure.** `expletives-test` should exercise supported
   public APIs rather than gaining privileged access through accidental
   internals.
8. **Explicit automation trust boundary.** Automation is a consequential
   control plane and observation surface. It is default-off and enabled only
   by `--automation <socket-path>`. The initial implementation may be
   unauthenticated for operator-controlled, non-risky use; authentication and
   capability authorization are deferred design work.
9. **Learn before specializing.** Use Win32, wxWidgets, Motif, LessTif,
   curses, ncurses, terminfo, Turbo Vision, and other established toolkits as
   evidence about controls, command routing, inheritance/composition, event
   loops, threading, focus, and keyboard behavior. Adopt patterns only after
   translating them deliberately into Go and the terminal domain. The current
   source review is in
   [`research/ui-toolkit-lessons.md`](research/ui-toolkit-lessons.md).
10. **Multithreaded MVC-compatible consumption.** Let applications keep
    toolkit-independent models, express the control tree as a view, and route
    structured commands and events through a controller, presenter, update
    function, or similar MVC, MVVC, or related application layer. Toolkit
    public behavior is thread-safe even when the event loop, renderer, and
    presenter use serialized owners. Do not require a single framework,
    reflective binding, or toolkit model inheritance.
11. **Panel-based Layouts.** Use instantiated `BoxLayout` and `GridLayout`
    objects, attached atomically with `Panel.SetLayout`, to arrange direct
    child Panels relative to a parent Panel. Most controls share Panel behavior
    directly or indirectly through an idiomatic Go composition/embedding
    model. Below minimum, preserve minima and resulting logical rectangles,
    constrain effective output by every ancestor clip and the application
    surface, expose overflow, and dispatch an optional application Overflow
    callback with a deterministic fallback.
12. **Inspectable Basic snapshots.** Each Basic `SnapshotV1` cell contains its
    canonical one-cell grapheme, semantic style, resolved
    foreground/background colors, and stable owner identity. Cursor state and
    the bounded typed control tree are
    snapshot-level facts; width and continuation fields are unnecessary.

## Current State

The first runnable Core/Containers, Basic Presentation, and Basic Automation
vertical slice now exists. It provides the Go module and public package,
`Panel`, `Frame`, and `GroupBox`, deterministic intended frames and immutable
snapshots, the one-cell Unicode boundary, a narrow CGO-free Linux terminal
presenter, a human-runnable colored-Panel fixture, explicit Unix-socket
automation, a reusable client, `expletivesctl`, and the required root
Makefile. Both executables build in all three modes.

The slice has passed ordinary package tests, race detection, vet, focused
fuzzing, PTY and process tests, all-mode builds and self-checks, and a real
headless attached-automation session through raw Control-R, direct reset, and
orderly shutdown. The first-slice correction and verification record is
complete under
[`EXPL-TASK-014`](../project-management/completed-tasks.md). Basic Layouts
were subsequently completed under
[`EXPL-TASK-009`](../project-management/completed-tasks.md); later common
controls remain planned.

The implemented choices are fixed in
[`implementation-baseline-v0.md`](specifications/implementation-baseline-v0.md),
[`go-api-v0.md`](specifications/go-api-v0.md), and
[`automation-protocol-v1.md`](specifications/automation-protocol-v1.md).
Important later choices remain deliberately open:

- the external custom-control extension model beyond the current built-in
  embedding/composition pattern;
- every later common-control phase and Layout replacement, detachment,
  spacers, and reparenting;
- broader terminal, platform, locale, code-page, curses/ncurses, and terminfo
  support beyond the deliberately narrow first Linux profile;
- complete focus, menu, mnemonic, accelerator, hotkey, and configurable
  interrupt policy;
- optional MVC/controller/dispatcher or data-binding helpers that do not
  impose one application architecture;
- later snapshot and protocol extensions, multiple controllers, and capture
  separation;
- authenticated or capability-authorized automation for riskier contexts;
  and
- additional supported targets, CI matrices, pinned analysis/security tools,
  release compatibility policy, and performance baselines.

Do not resolve those questions implicitly in code. Record the decision and
update the governing specification first.

Authentication and capability authorization beyond the initial explicit,
operator-controlled automation opt-in are intentionally deferred work, not an
open prerequisite for Basic Automation.
