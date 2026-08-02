# Build and Verification Requirements

Status: Directed requirements; first runnable baseline implemented
Authority: Direct operator requests on 2026-07-24
Related decisions: `EXPL-DEC-001`, `EXPL-DEC-004`, and `EXPL-DEC-008` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)

## Purpose

This document defines the project-owned build surface, artifact layout, and
verification outcomes for `expletives` and its test applications.

The root Makefile and first executable inventory are implemented. This
document remains the contract for every executable added later.

## Root Makefile Interface

The project must provide a root-level `Makefile`. At minimum, these commands
must work from the repository root after documented prerequisites are
available:

```text
make clean
make all
make build
make release
make profiling
```

Their required meanings are:

| Target | Required outcome |
| --- | --- |
| `clean` | Remove only project-owned generated build artifacts beneath the repository-root `build/` directory. |
| `build` | Build every project and test executable in debug mode. |
| `release` | Build every project and test executable in release mode. |
| `profiling` | Build every project and test executable in profiling mode. |
| `all` | Build all three modes and fail if any required artifact cannot be produced. |

Every executable target in the project inventory must participate in
`build`, `release`, and `profiling`. A command may not be debug-only,
release-only, or profiling-only merely because it is primarily a development,
test, automation, or diagnostic tool.

Targets must be non-interactive, deterministic from declared inputs, and
marked phony where appropriate. They must not silently install or upgrade
system-wide or user-wide tools.

The Makefile also exposes `test`, `test-unit`, `test-integration`,
`test-race`, `benchmark-terminal`, `fmt-check`, `vet`, `smoke`, and `verify`.
`verify` runs format checking, vet, ordinary and integration-tagged package
tests, race detection, all three builds, and command smoke/self-checks.
`benchmark-terminal` is deliberately opt-in: it measures coalesced and
fragmented input plus ordinary and 1200-by-1200 frame encoding without making
machine-dependent timing a correctness gate.

## Artifact Layout

Every project or test executable named `<name>` must be written to:

```text
build/debug/<name>
build/release/<name>
build/profiling/<name>
```

The current required executable inventory includes:

```text
build/debug/expletives-test
build/release/expletives-test
build/profiling/expletives-test
build/debug/expletivesctl
build/release/expletivesctl
build/profiling/expletivesctl
```

Any later project command or test application automatically receives the same
three-mode artifact requirement when it joins the supported build inventory.

On targets with Unix executable mode bits, each produced application must be
executable. A build tool that creates the mode correctly needs no redundant
`chmod`; verification must still detect a non-executable artifact.

`build/` is generated output, not durable project state. It must remain
excluded from version control. `make clean` must resolve the exact
repository-root `build/` path and must not remove `.local/`, the repository
root, an unresolved variable, or an arbitrary caller-supplied directory.

## Build Modes

The first baseline uses module
`github.com/Hard-Problems-Group-LLC/expletives`, Go 1.25 as the minimum
language/toolchain line, and Go 1.26.5 as the normal verification toolchain.
Product builds use `CGO_ENABLED=0`, `-mod=readonly`, and `-buildvcs=true`.
Linux amd64 is the initial runtime target; Linux arm64 compilation is checked
separately. Race tests use `CGO_ENABLED=1` because the Go race runtime requires
cgo on the supported host.

### Debug

Debug builds prioritize source-level diagnosis:

- retain the information required by the supported Go debugger;
- define the optimization and inlining policy explicitly;
- preserve useful assertions and diagnostics;
- do not enable attached automation unless the user passes
  `--automation <socket-path>`; and
- remain behaviorally representative except for documented debug-only
  diagnostics.

`make build` produces this mode under `build/debug/`.

The implemented debug command adds `-gcflags='all=-N -l'` and injects
`buildMode=debug` through the linker while retaining source paths.

### Release

Release builds prioritize supported production behavior:

- use the project's declared optimized settings;
- identify source revision, Go toolchain, target, and relevant build mode;
- satisfy the approved cgo and supported-target policy;
- avoid uncontrolled timestamps or workstation paths in artifacts where
  reproducibility matters; and
- pass release-level tests and startup smoke checks.

The implemented release command uses ordinary Go optimization, `-trimpath`,
retained symbols and VCS build information, and injected
`buildMode=release`.

### Profiling

Profiling builds support representative performance investigation:

- retain the symbols and metadata needed by the supported profiling tools;
- document any difference from release optimization or instrumentation;
- keep workload behavior representative enough for the question being
  measured;
- do not expose a profiling listener, automation endpoint, or sensitive
  diagnostic surface merely because this mode was selected; and
- remain safe to run with explicit, protected profiling collection.

Profiling evidence must record the build mode, toolchain, target, workload,
and relevant configuration.

The implemented profiling command deliberately uses the same ordinary
optimization, `-trimpath`, symbols, and VCS information as release, with
`buildMode=profiling`. It adds no listener or instrumentation by itself so
standard Go profiling tools can observe a representative binary.

The reproducible terminal micro-workload is:

```text
make benchmark-terminal
```

It reports allocation counts and processed-byte throughput. Results are
diagnostic evidence tied to the recorded host and revision, not a portable
latency guarantee.

## Normal Go Tests

Use ordinary Go community testing practices wherever practical.

- Keep unit tests in `*_test.go` files beside the packages they exercise.
- Use external test packages when validating the supported consumer-facing
  API, and same-package tests only when access to internals materially
  improves a bounded test.
- Prefer the standard `testing` package unless another dependency earns its
  cost.
- Use deterministic table-driven tests where cases share a clear shape.
- Test observable contracts, error behavior, ownership, cancellation, and
  cleanup rather than internal call choreography.
- Use real owned implementations and small fakes before broad mocks.
- Keep tests independent of order, ambient credentials, a developer home
  directory, public network services, and arbitrary sleeps.
- Run integration tests through real project composition and encoding paths
  where practical.
- Run race detection for concurrent controller, event-loop, snapshot, and
  automation paths on a supported target.
- Fuzz structurally varied and untrusted boundaries, including terminal input,
  Unicode/grapheme handling, snapshot decoding, and the attached protocol.

The baseline workflow must eventually declare commands for formatting,
module consistency, `go vet`, selected static analysis, unit and integration
tests, builds, race detection, vulnerability analysis, supported versions,
and supported target coverage.

## Layered UI Verification

Normal Go tests are necessary but not sufficient for a terminal toolkit. The
verification contract also includes:

1. direct controller and pure-renderer tests;
2. deterministic in-process headless sessions;
3. structural semantic-frame and widget-view assertions;
4. `expletives-test` catalog coverage;
5. opt-in attached automation tests;
6. PTY process-boundary and terminal-lifecycle tests; and
7. a small real-terminal and remote-transport matrix for declared support.

These layers prove different things. A headless snapshot cannot establish
physical emulator fidelity, and a screenshot cannot establish controller or
semantic-state correctness.

See [`expletives-test.md`](expletives-test.md) for the interactive and
drive-and-observe contract.

## Build Verification

Automated checks must confirm:

- all required Make targets exit successfully from a clean checkout with the
  declared prerequisites;
- every executable in the declared inventory is present in all three modes,
  with no target silently omitted from one mode;
- each expected artifact exists at exactly the required path;
- artifacts are executable on platforms that require an executable mode bit;
- debug artifacts are usable with the supported debugger;
- release artifacts identify their build provenance and pass smoke tests;
- profiling artifacts work with the supported profiling workflow;
- `make all` produces the same required artifacts as the three individual
  mode targets;
- `make clean` removes generated `build/` contents without touching unrelated
  paths; and
- rebuilding after `make clean` succeeds.

## Remaining Decisions

The first Makefile baseline is implemented. Later project decisions still
need to select:

- additional supported `GOOS` and `GOARCH` runtime targets;
- the local-versus-CI target and real-terminal matrices;
- pinned lint, vulnerability, benchmark, and profiling tools;
- release compatibility and reproducibility policy beyond the current Go/VCS
  provenance; and
- any future executable inventory, which automatically joins all three build
  modes.
