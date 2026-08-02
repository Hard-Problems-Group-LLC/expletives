# First Runnable Implementation Baseline

Status: Directed implementation baseline
Authority: Direct operator instruction to proceed on 2026-07-24
Related decision:
[`EXPL-DEC-008`](../../project-management/decision-log.md#expl-dec-008--select-the-first-runnable-go-terminal-and-automation-baseline)

Related correction:
[`EXPL-PROP-002`](../../project-management/proposals/under-review/expl-prop-002-foundation-review-corrections.md)

## Scope

This specification selects the minimum concrete choices needed to build and
run the Core/Containers, Basic Presentation, and Basic Automation vertical
slice. It does not approve Basic Layouts or the later common-control phases.

The first runnable products are:

- the public `expletives` Go package;
- the human-runnable `expletives-test` colored-Panel fixture; and
- the reusable automation client package plus `expletivesctl`.

## Go And Build Baseline

- The module path is
  `github.com/Hard-Problems-Group-LLC/expletives`, matching the repository
  origin.
- The root package is `expletives`.
- Product commands live under `cmd/expletives-test` and
  `cmd/expletivesctl`.
- The minimum language/toolchain line is Go 1.25. The normal verified
  toolchain for this slice is Go 1.26.5.
- Product binaries are pure Go with `CGO_ENABLED=0`.
- Initial runtime support is Linux on amd64. Linux arm64 must compile; runtime
  evidence may follow when an owned arm64 environment is available.
- Race-detector verification is test-only and may use the race runtime's cgo
  requirement without making product artifacts cgo-dependent.

All modes build both commands:

```text
build/debug/expletives-test
build/debug/expletivesctl
build/release/expletives-test
build/release/expletivesctl
build/profiling/expletives-test
build/profiling/expletivesctl
```

Debug disables optimization and inlining and retains source paths. Release
uses ordinary Go optimization, `-trimpath`, symbols, VCS build information,
and explicit mode metadata. Profiling uses ordinary representative
optimization, `-trimpath`, symbols, and explicit mode metadata; it enables no
listener or collection endpoint by itself.

## Core And Snapshot Baseline

One `App` owns one special root `Panel`. `Control` is the sealed common
identity/geometry capability; `Container` is the sealed child-parent
capability. Every ordinary control is created with a `Container` parent.
`Panel`, `Frame`, and `GroupBox` are distinct concrete copy-safe handles over
one canonical node each. Copying a constructed handle aliases that node;
zero or fabricated handles remain invalid. `Parent` and `Children` retain the
actual concrete wrappers.

Runtime control IDs are monotonically allocated per App and never reused.
Optional automation keys are bounded and unique in the final active tree.
Parent ownership is immutable. Recursive destruction removes a subtree from
active indexes, releases its keys and concurrent-control capacity, and
publishes once.

Controls store semantic `StyleID` values. An immutable App `Theme` maps those
IDs to `ResolvedStyle` values containing foreground, background, and bounded
terminal-independent attributes. Construction and mutation reject a style
missing from the final Theme. Theme replacement is atomic.

An App-scoped bounded `Transaction` can create, mutate, destroy, resize, and
replace the Theme. Provisional controls can parent later controls in the same
Transaction. Commit validates the complete final tree—including capacity,
keys, surviving style references, and destruction conflicts—then applies all
operations or none and publishes at most once. Convenience constructors and
setters use the same transaction path. Mutation-gate waits observe context
cancellation and have a finite busy bound.

Geometry uses checked integer cell rectangles; negative child origins are
allowed for deliberate clipping, while dimensions are nonnegative. Intended
frames are limited by the named 4,194,304 aggregate-cell allocation budget
rather than a conventional desktop axis. Titles are limited to 256 UTF-8
bytes, 256 normalized cells, and 64 bytes per canonical cell.

Rendering paints the root and then descendants in stable stack/insertion
order. Later siblings paint over earlier siblings. Effective clipping
intersects the control rectangle, every ancestor client clip, and the
application surface. Frames, GroupBoxes, and Layouts independently select
optional border forms; each enabled border contributes a one-cell client
inset. This permits a bordered Layout to contain adjacent unbordered Frames
without doubled seams.

The first root-package local `Snapshot` contains:

- a monotonic sequence and final marker;
- one row-major intended frame;
- cursor state;
- a bounded typed pre-order control view;
- source-local held-key state needed to diagnose raw chords; and
- the request completion associated with a submitted event or command.

Each cell contains exactly its canonical one-cell grapheme, semantic style,
resolved foreground/background/attributes, and stable owner ID. There is no
width or continuation field. The automation package owns a distinct
`automation.SnapshotV1` DTO and explicitly projects the root-package
`Snapshot` into it; root Go types are not the wire schema.

The App retains exact local snapshots by aggregate frame-cell cost, with a
64-record metadata cap for tiny frames, but no request-ID ledger or result
cache. Protocol-session request uniqueness and retained results belong to the
automation server.

Held-key storage exists only for sources with at least one key down. It is
bounded to 4,096 concurrent sources and eight keys per source; one-shot
presses allocate no source map, and releasing the last key reclaims the
source.

## Command Baseline

Each App owns a registry of `CommandDefinition` values. Definitions carry a
bounded ID and description plus `Enabled` and `Automation` capability flags.
The registry supports register, replace, remove, deterministic inspection,
structured chord binding, replacement, and unbinding.

`SetCommandRouter` installs the application policy callback. It returns a
`CommandResult` with an outcome, stable public code, public message of at most
1,024 UTF-8 bytes without NUL, and a local-only cause. Raw local error text is
never exposed automatically. The former `SetCommandHandler` shape remains a
compatibility adapter.

Router callbacks run outside App state locks on a four-slot bounded executor.
They receive the earlier caller deadline or a 30-second default. A stuck
callback can retain one slot but cannot retain the dispatch gate indefinitely.
Each App has an independent cancellation-aware dispatch gate with a finite
busy bound; cross-App dispatch is supported and cyclic contention terminates
through cancellation or a busy result rather than a hidden context-value
guard.

## Unicode Baseline

The project pins `github.com/rivo/uniseg` version `v0.4.7` as the initial
segmentation and display-width data/algorithm dependency. All text-bearing
rendering passes through one project-owned normalization boundary:

1. segment a complete grapheme cluster;
2. retain it only when its measured width is exactly one cell and it is not a
   terminal control; and
3. otherwise emit exactly one `U+FFFD REPLACEMENT CHARACTER` cell.

Pinning the dependency versions the first width policy. The project wrapper,
one-cell acceptance rule, tests, and snapshot schema remain authoritative.
Changing the dependency or policy requires recorded compatibility review.

## Initial Terminal Profile

The first physical adapter is a small internal, standard-library-only,
CGO-free Linux terminal boundary. It deliberately supports interactive
xterm-, screen-, and tmux-family `$TERM` profiles that are exercised by the
project. It is not a claim of generic ANSI, VT100, curses, ncurses, terminfo,
or other-platform support.

Before changing terminal state, the adapter verifies terminal descriptors,
the supported profile, any available bounded compiled terminfo evidence, and
geometry. Missing terminfo retains the narrow static profile; found malformed
or contradictory evidence fails closed without executing its strings or an
ambient helper. The adapter saves the exact termios state, enters
noncanonical/no-echo mode while retaining signal generation, enters the
alternate screen, enables bracketed paste, hides the cursor, and uses one
owner for input and output. Every catchable exit restores style, cursor,
bracketed-paste mode, alternate-screen state, and exact termios state in
reverse order.

The correctness-first presenter emits a complete bounded frame with
ECMA-48 cursor/style operations and the explicitly supported xterm private
screen/cursor modes. It maps resolved RGB values to the fixed terminal
palette. Printable 7-bit ASCII is emitted directly. Every non-ASCII logical
cell uses a project-owned single-ASCII approximation in black on yellow, or a
black-on-yellow `?`, without changing the canonical snapshot.

`SIGWINCH` produces a resize transition. `SIGINT`, `SIGTERM`, and `SIGHUP`
produce orderly semantic shutdown paths and final snapshot publication.
The demo's default Ctrl-C policy is interrupt-and-exit; injected Ctrl-C uses
the same command policy. Interactive `SIGTSTP` restores terminal and input
state before a guaranteed self-stop; `SIGCONT` reacquires modes and geometry
and forces a complete repaint. Broader terminal portability remains Phase 19
work governed by
[`terminal-compatibility-v0.md`](terminal-compatibility-v0.md).

Headless and self-check modes initialize no terminal and exist for normal Go
and process integration tests.

The public `terminal.InputDecoder` owns bounded, stateful translation from
terminal bytes to ordered logical key and bracketed-paste values. Keys enter
`App.DispatchKey`; complete paste enters `App.DispatchTextInput` and cannot
enter command resolution. The key-only compatibility `Feed` method continues
to discard paste. Human key input then enters the same App dispatch path as
attached automation. Application commands do not parse escape sequences
themselves.

## Basic Automation Protocol

The transport is an unauthenticated Unix stream socket created only by:

```text
expletives-test --automation <socket-path>
```

The server refuses every pre-existing path, creates the socket with mode
`0600`, remembers its created object identity, and removes only that same
socket during orderly cleanup. The operator receives a conspicuous warning
that anyone able to connect can observe and drive the application.

The wire format is UTF-8 JSON Lines. Every record includes:

```json
{"protocol":"expletives.automation","version":1,"type":"..."}
```

Version 1 permits one connected controller and one sequential request in
flight. It fixes a 64 KiB request-line limit, a 40 MiB response-line limit,
4,194,304 aggregate frame cells, 16,384 compact frame runs, 4,096 controls and
aggregate child references, 1,024 Layouts, 4,096 aggregate Layout items,
eight held keys per source, and three retained results under a 128 MiB
aggregate encoded-evidence budget, with finite read/write deadlines, strict
operational records, and bounded identifiers. A server `hello` advertises
`supported_versions`, exact limits, session ID, scenario, operations,
App-registered automation commands, and current frame sequence. The session
ID is not a credential.

Every request ID is unique while active or retained by that server session.
The single `RetainedResults` FIFO stores completion metadata and its exact
immutable automation snapshot together and evicts them together. A terminal
completion reports
the actual `applied`, `no_op`, `rejected`, `cancelled`, `interrupted`,
`exited`, or `failed` outcome and its associated frame sequence. A client-side
timeout is indeterminate and is never automatically retried.

Version 1 operations are:

- `observe`, for the latest or a retained exact snapshot;
- `wait_snapshot`, for a sequence later than a supplied sequence;
- `inject_input`, with exactly `key_down`, `key_up`, or `key_press`;
- `invoke_command`, with a stable command ID and optional `target_key`;
- `query_result`, for an active or retained result;
- `reset_input`, which clears source-local held state; and
- `shutdown`, which uses the normal `app.quit` policy and publishes a final
  snapshot.

The stable key set covers ASCII letters/digits, `[`/`]`, modifiers, common
navigation/editing keys, and F1 through F12. `KeyPress` is one-shot. Held state
belongs to one connection and is cleared on disconnect. The fixture binds
Control-down, `r`-press, Control-up to `fixture.toggle`.

The reusable automation server has no fixture command defaults. Its
`Application` identity is required, and its command inventory is derived from
the host App registry entries whose `Automation` flag is true.
`expletives-test` registers `fixture.toggle`, `scenario.reset`, and
`app.quit`. Direct commands and raw-key resolution enter the same App command
router.

## Initial Command-Line Surfaces

`expletives-test` supports:

- normal human mode;
- `--automation <socket-path>`;
- `--headless` for owned process tests;
- bounded `--width` and `--height`;
- `--self-check`; and
- `--version`.

`expletivesctl --socket <path>` supports `hello`, `snapshot`/`observe`,
`wait`, `key`, `keys`, `command`, `result`, `reset-input`, and `shutdown`.
The `keys` form keeps one connection open for the complete modifier chord.
The reference CLI separates a five-second connection-and-hello deadline from
a 35-second operation deadline, leaving the server's 30-second execution
budget plus its response margin.

## Acceptance

Before inviting a human launch, verification must include:

- ordinary Core, renderer, Unicode, input, terminal-encoder, protocol, client,
  and process integration tests;
- concurrent snapshot/resize/input coverage under the race detector;
- `go vet`;
- `make clean`, `make all`, and exact executable checks for all six
  artifacts;
- `expletives-test --self-check` in every mode; and
- one real attached headless session observed, toggled through the raw
  Control-R lifecycle, reset directly, and shut down through
  `expletivesctl`.
