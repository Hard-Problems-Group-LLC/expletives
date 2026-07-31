# Review Packet: Foundation Design

> **Frozen version 1. Do not edit this packet in place.**
> Corrections or later evidence require a versioned addendum or replacement
> packet supplied identically to every reviewer.

- Review ID: `EXPL-REV-001`
- Stage: design
- Status: Frozen; unconvened; reviewer independence and quorum pending
- Scope: Core/Containers, Basic Presentation, Basic Automation, first Linux
  terminal adapter, build contract, and the public/client-facing boundaries of
  the first runnable foundation
- Trigger: new public Go API and hierarchy; automation protocol and trust
  boundary; foundational rendering, input, concurrency, terminal lifecycle,
  portability, and build behavior; multiple supported products
- Sponsor: `/root`, a Codex collaboration context acting on direct operator
  requirements
- Operator: project operator
- Evidence chair: `/root/design_freeze_chair`
- Reviewers: nine proposed isolated contexts listed below; eligibility,
  independence, and quorum pending
- Source base revision: `df92af4f87af3c6a500a027ddda4a9d1e64f26c8`
- Worktree state: dirty; `.gitignore` and `README.md` are modified and the
  remaining inventoried first-slice files are untracked relative to the base
  revision
- Packet version: `v1`
- Frozen at: `2026-07-30T03:42:53-07:00`
- Packet file SHA-256: recorded after freeze in the chair convening record
- Manifest integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`

## Review Purpose

Determine whether the first runnable foundation is coherent and sufficiently
specified to support continued implementation and later common controls
without silently fixing the wrong public, concurrency, automation, rendering,
terminal, or extension boundaries.

This is a design review even though implementation evidence exists. Code and
tests are included so reviewers can challenge feasibility and detect drift
between the written contract and the emerging API. Test passage and existing
code are not substitutes for design approval. A separate delivery review is
required before the implementation is closed.

## Current Implementation State

As of the sponsor reconciliation on 2026-07-30, the current worktree contains
the public Go package, both product commands and their tests, the narrow Linux
terminal adapter, the automation server and reusable client, the root
Makefile, and the formal Go API and automation protocol specifications. The
active task and product records report ordinary, race, vet, fuzz, PTY,
process, all-mode build/self-check, and corrected external closed-loop
exercises.

Those are sponsor-supplied current-state claims. The evidence chair verified
the exact inventoried bytes and the bounded validation evidence recorded
below. A later delivery review must assess the implemented result separately.

Some included proposal, backlog, and completed-work text deliberately
preserves dated pre-implementation context. It is provenance, not a newer
current-state claim. The implemented v0/v1 specifications and this dated
disclosure describe the current slice; reviewers should still report any
material contradiction rather than silently choosing one.

## Scope

The reviewed foundation includes:

- one explicit `App`, its sole special root `Panel`, immutable parent
  ownership, and the first `Panel`, `Frame`, and `GroupBox` hierarchy;
- checked cell geometry, ancestor clipping, deterministic paint order, stable
  identity, one-cell display normalization, intended frames, and immutable
  typed snapshots;
- thread-safe public calls, serialized state ownership, callback/reentrancy
  boundaries needed by the slice, request correlation, and orderly final
  publication;
- raw `KeyDown`, `KeyUp`, and `KeyPress`, held modifiers, structured chords,
  semantic commands, and configurable interrupt/quit separation represented
  by the first fixture policy;
- default-off, explicit, bounded, unauthenticated JSON Lines automation on an
  operator-selected Unix socket, plus the reusable client and
  `expletivesctl`;
- the deliberately narrow CGO-free Linux xterm/screen/tmux-family terminal
  adapter, exact terminal restoration, physical ASCII degradation, resize,
  and signal handling;
- `expletives-test` as a public-API consumer and diagnostic surface; and
- debug, release, and profiling builds for both commands at the required
  artifact paths.

## Non-Goals

This packet does not seek design approval for:

- Basic Layouts, `BoxLayout`, `GridLayout`, or Overflow API details;
- common controls after `Panel`, `Frame`, and `GroupBox`;
- reparenting;
- a complete external custom-control extension model;
- authentication, authorization, hostile multi-user operation, or multiple
  automation controllers;
- arbitrary terminal escape-byte injection;
- general ANSI, VT100, curses, ncurses, terminfo, non-Linux, or terminal
  portability;
- display elements the pinned project policy measures as multi-cell,
  zero-width, indeterminate, or otherwise unsupported;
- broad MVC framework, reflective binding, or toolkit-owned domain models;
- final menu, mnemonic, accelerator, focus, editor, modal, or Ctrl-C policy;
  or
- delivery readiness, release approval, or risk acceptance.

## Applicable Direction And Constraints

- The project-specific `AGENTS.md`, product goals, directed specifications,
  and decision log are controlling within their declared scopes.
- `docs/specifications/ui-toolkit-requirements.md` and comparative research
  are design evidence, not compatibility contracts.
- The current implementation baseline selects Go 1.25, normal verification
  with Go 1.26.5, `github.com/rivo/uniseg` `v0.4.7`, pure-Go Linux products,
  and an intentionally narrow first terminal profile.
- Every displayed element occupies exactly one canonical cell. Supported
  composed clusters are permitted only when the project width policy measures
  them as one cell; unsupported elements become one `U+FFFD` cell.
- Physical terminal degradation is separate from the canonical intended
  frame and must preserve its one-cell geometry.
- Automation is absent by default and may be unauthenticated only for the
  explicit operator-selected non-risky mode. It must not be represented as
  safe for a hostile, shared, or elevated environment.
- Every public toolkit behavior in scope must be thread-safe, though internal
  UI work may be serialized.
- Existing code is concurrent work in a dirty worktree. The manifest, not the
  base Git commit, identifies the exact reviewed bytes.

## Sponsor-Reconciled Explicit File Inventory

The sponsor reconciled these repository-relative paths against the first
runnable slice on 2026-07-30. At freeze, the evidence chair independently
confirmed that all 71 paths exist as regular files and that the inventory
contains all 38 current Go source/test files, all 11 files under
`docs/specifications/`, and the root module and build inputs.

### Governing Instructions, Documentation, Specifications, And Research

```text
AGENTS.md
FieldManual-config.toml
README.md
docs/Limited-Unicode-Support.md
docs/README.md
docs/product-goals.md
docs/research/ui-toolkit-lessons.md
docs/specifications/README.md
docs/specifications/application-architecture.md
docs/specifications/automation-protocol-v1.md
docs/specifications/build-and-verification.md
docs/specifications/concurrency-and-thread-safety.md
docs/specifications/control-catalog.md
docs/specifications/expletives-test.md
docs/specifications/go-api-v0.md
docs/specifications/implementation-baseline-v0.md
docs/specifications/layouts-and-overflow.md
docs/specifications/ui-toolkit-requirements.md
```

### Project-Management Authority, Scope, And State

```text
project-management/ai-human-requests.md
project-management/backlog.md
project-management/completed-tasks.md
project-management/decision-log.md
project-management/deferred.md
project-management/development-roadmap.md
project-management/open-questions.md
project-management/proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md
project-management/reviews/README.md
project-management/reviews/panel-of-experts-charter.md
project-management/tasks-in-progress.md
```

### Module, Build, Public Package, And Public Tests

```text
.gitignore
Makefile
app.go
app_test.go
errors.go
go.mod
go.sum
input.go
input_test.go
panel.go
panel_test.go
public_api_test.go
types.go
```

### Automation Protocol, Server, Client, And Tests

```text
automation/app_bridge.go
automation/client.go
automation/client_test.go
automation/codec.go
automation/codec_test.go
automation/protocol.go
automation/retention_test.go
automation/server.go
automation/server_integration_test.go
automation/socket_test.go
```

### Products And Internal Foundation

```text
cmd/expletives-test/main.go
cmd/expletives-test/main_test.go
cmd/expletivesctl/main.go
cmd/expletivesctl/main_test.go
internal/buildinfo/buildinfo.go
internal/buildinfo/buildinfo_test.go
internal/demo/scene.go
internal/display/normalize.go
internal/display/normalize_test.go
internal/terminal/color.go
internal/terminal/color_test.go
internal/terminal/encoder.go
internal/terminal/encoder_test.go
internal/terminal/presenter_linux.go
internal/terminal/presenter_linux_test.go
internal/terminal/presenter_unsupported.go
internal/terminal/profile.go
internal/terminal/profile_test.go
internal/terminal/sequences.go
```

The inventory deliberately excludes generated `build/` artifacts, `.local/`
state and captured session data, the FieldManual submodule, empty review
scaffolding, and later Layout/control implementation that is outside this
design stage. The Makefile, command sources, build metadata, and `.gitignore`
are the reviewable build inputs; executable artifacts and rerun results belong
in verified evidence and the later delivery packet.

## Canonical SHA-256 Manifest

```text
f42bcbf85571db20520c0023f711e1acfa1eae96bccd86b566ef1c39baa30697  .gitignore
9e5bd2de9f357e1c865d597618d100369155d3ad036233c7fde8139af841251b  AGENTS.md
75b8287482a941acaa6b72d741d095a16cbf7887e2d0be22890e9806fb582b0b  FieldManual-config.toml
13d56758eec59e20a190ed05b343ae5f7ce0507cefbefb12b66efd3c8dbc1691  Makefile
057f6d00aa373851bdac0a85565c6d3e85e86127d1bbf2eed89089151f24f330  README.md
2068201affa73455e206694a9a14cb2f3d3933f882f17e09f2e10afd25399811  app.go
c1fe6b8f826ae9ecec81c44050680ddf75291804aebc42785804c42c7368237e  app_test.go
a6058c747d69fcc456d077118157fe78e54d3094dd9f280676b2c250c718679e  automation/app_bridge.go
126dd3f6f491c7d3cb28ab4940eb3ffe7bd70e70a172aaac7555986ec2fc23d4  automation/client.go
4664328514b0d11bc642606bfed6384494f0a12a7a41517c04c6f0117ff7d363  automation/client_test.go
45f18770220da62be9f464e5fbbe2bb4a43632fe751a91564f9fce9caed0dfc1  automation/codec.go
1327bdde1e81507be621844157e2f42269a606a61dfd7b9d462126315443aaa6  automation/codec_test.go
b35a18abecff6486004fceff0a1495f261a95fa5e438cb6d0f1b0e6273783367  automation/protocol.go
588dd773290b0a67fbfdf0fdbac240028f156f68e2a9f7d096968fd41b4ce198  automation/retention_test.go
cfc0f4152eac94a5316a3eb4108130243e2c388b430aea181b03a864529de48a  automation/server.go
95057e81ff2d060fcb7e73089d89b7d73dba44483202c1e1f0110d2a2dc2eb37  automation/server_integration_test.go
307ed6608d3b404453f3d13c5a098a34fa69c976e56ac4655eaf94b0de999a9e  automation/socket_test.go
41793bb36a127c0ec029c997bb6cf78cda6e6bd97fa50a77449e67070cb859a6  cmd/expletives-test/main.go
69971381992d55301842c52930d6f1c17aea005d1b6f10f240ddee7f6e65d3d8  cmd/expletives-test/main_test.go
7375a1fcd24844366044ca85473ceba5231d5dfb5b07cd7911209757f56775c7  cmd/expletivesctl/main.go
c689f230245916db2a08ec3b0d1aa16ffbfb2a6b7008799fc113dc96b3aeeaae  cmd/expletivesctl/main_test.go
9d65cc589e6168bb600748458a8a7c5d3eb3383eb02a972540dc7a7dc3116e5c  docs/Limited-Unicode-Support.md
8a64dffc69ca1b9815e15ebd5919c39bd4223237282207945ff1e71246a4619f  docs/README.md
7a0184c59bf33f4f9ca144278466a26d4f98d5834e54e8d2a87a644d8edf1fa9  docs/product-goals.md
e001be90d7fd96093d9da0bc52b761dafa3976320b0fc5d3f096f5fcb8055414  docs/research/ui-toolkit-lessons.md
17ff62095c3d1906a22a53587dde7d58a5e4a67c51f189c43edd7af87ba1f939  docs/specifications/README.md
6d0e7a36b49c6ed2bae14a3e6dedf0866e626469e024c0a2d46b728a156f881b  docs/specifications/application-architecture.md
eae1b8ad400d2a57ea8f072922572970022718b2a73dcd737c8b2abd17c13301  docs/specifications/automation-protocol-v1.md
392b7ec20cb83cd5e3993ac4851f8a7cb61227f171a2c5e4efb07a76b274e9e8  docs/specifications/build-and-verification.md
689302093651d54b59c3d71cc6f1f40ad80e77bb027517c6b9c85d8fd6b76f8f  docs/specifications/concurrency-and-thread-safety.md
2d8b9c69f22cfdb3491a22ec0ba38f41703414cf4ce17c93e98dda901cd56c5a  docs/specifications/control-catalog.md
764d01c546afc14a1f3acc3b60b5685b8076e29a65219e503c18e32b59f3534a  docs/specifications/expletives-test.md
fb5b38e6e1a96292a9c6329f508837ab1fb15700cb14a965745d381152ccf5ba  docs/specifications/go-api-v0.md
df3f885ecc1204263202399554a409603fe1d9f826982d3910b115fd9b1fc2cd  docs/specifications/implementation-baseline-v0.md
889fc74bc23045895c68c08e09b9bf493c2083007b54709eb6fc49ec37c15a7f  docs/specifications/layouts-and-overflow.md
e7bafb56b7a57788e33c007fd05f2dfb2fecfc7a22c421022744639154784edf  docs/specifications/ui-toolkit-requirements.md
a8d59939d2b6b2ec3c7089110cc56abb744791cdf8880e65bde971d26504b965  errors.go
ce64cc4663e8eaee0e3d30d779874233e0835ee21b10ba7790df12b2ca0aa827  go.mod
b14bae894b3fa9e9b39c5c850cb044afa3fbd20d5b5104556506baf3acee2b49  go.sum
30c5743dbecba3d96eb4dc78094aadff6a951892145a7ac26e5a4f4fb285dd33  input.go
c6186f677e2e81c0614b97a03afc58e787c01d24209e66780ab8cbf9f8a252cd  input_test.go
2eee42a7a93e623682556ee5e99529c7cc5239bdf6ed30d0435d9bd1cfb48637  internal/buildinfo/buildinfo.go
37dec4dfd4e5fb89869c41a9fef69c73fbc4004e6f7e2e5fc15867ff8e893525  internal/buildinfo/buildinfo_test.go
4c52060731d99b129ece472bb2679b30c2fc8c534fa239b1c23f1cc49f9c6d87  internal/demo/scene.go
8a43d972e392f04fe9b41c8caf4beda63dd3f104d11cebdbde9a16a71adb69ef  internal/display/normalize.go
56940b7b79ebac91b569b3925773d7f3b75f4fcd979b4e8898e794c080e67d4e  internal/display/normalize_test.go
66ca0cac16e5d92b7dfdaea403cdd97f7eec205da21b366db1a46bbabc0be30a  internal/terminal/color.go
b37d0a0fa736ea4c95a08e13fb8110c8f570bffd698bcff6d9e4cd367bc9b24c  internal/terminal/color_test.go
0f931c70c95e22f75793702fbba5c2f86d01a0a2bd9c378888d773c6c3d59e96  internal/terminal/encoder.go
6502b036678dd896e4aa97b67013a846a6dae2664d7d86f3c0157ff125413b07  internal/terminal/encoder_test.go
c789d43462b33a873e098a0563fd031fc5c590db40aebdcea98a5084bd5e362e  internal/terminal/presenter_linux.go
0ff1fe4029802e545a2133c95b90479ca65d5ccf631117009bb67c6b0cc81863  internal/terminal/presenter_linux_test.go
8cf2c3a24d5e5cfde6a998886c29fa175a761638e89af5cb65507c638796e740  internal/terminal/presenter_unsupported.go
700afab0eb460a7c58dd4547096d29582cb3ebfe55c3aabee856f8d95e23a9b2  internal/terminal/profile.go
cb9501a9ea1a66fb78671b035affacd9b582e975b6b1c5eb180e8c12f7327e7e  internal/terminal/profile_test.go
d9a3431fe308a0ed903c8ebb0c396ea7aa17163eabf28aee8af81593525a5431  internal/terminal/sequences.go
2d6c6514857a18a633a97a86817617a2518f2df9e407a916d2a456879a653f1b  panel.go
75e22f36e14ca9340108a88921d95e6708af0d4ab59cd46cb851b01909806c41  panel_test.go
f12f11dab7932a87dee9ed0244b5cbd7d709f78ac58472fe407eaf419ba0e783  project-management/ai-human-requests.md
76ab49f4efb3ab3cba0fd88ea62ddc5b1b40025f270fd478c0f2e79f6aef0bcc  project-management/backlog.md
f6582a8351e89e509c3e2db25a38e6ffd7225d0b051baacc322339ca1663da6e  project-management/completed-tasks.md
42d70f6151563fc91f30f42840e1a3f462ba87230252e34dcaa1e5cffa06ad5a  project-management/decision-log.md
d0bd0825bbca95f95ebcfb8cd10fa165c52f72effed8980125bc674fab8754fa  project-management/deferred.md
a1ae28f6835ba7cafa093c4a6471a157f1c1a0549c8d25701a051ca44bdaaf2b  project-management/development-roadmap.md
7e263105950f58d778d67f9f0ce3dc430e088539d588d677505df065b8dea189  project-management/open-questions.md
cc45e65a8186b25a571bc7f5c4137841538dc94375c611c54212491a824b88a0  project-management/proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md
ad3be155635587a32f7cabac18abf2ddc894ff2e04ede37749dba64048d2df65  project-management/reviews/README.md
a947d8d6df5d032b8b829fa1096ce357c6849928b89b2c525d5509d230289b31  project-management/reviews/panel-of-experts-charter.md
9849ddda0b980950a7500ccbabbfb2a050b4f8e650c95c7140ae7bd656a601d3  project-management/tasks-in-progress.md
e422390d0aa617456be29c3e57b60ec3b03f9df73409dc88020a30459768c556  public_api_test.go
562d35e5156fec13eddf2b642bb39c13e50a92610257314310943d7dd9abcba8  types.go
```

The lines are sorted by path under `LC_ALL=C`. The manifest integrity
identifier above is SHA-256 over the exact manifest lines, including each LF.
The packet file itself is excluded from the embedded manifest; its SHA-256 is
recorded separately in the chair evidence record.

## Dirty-Worktree Disclosure

Immediately before freeze, the evidence chair verified the base revision as
`df92af4f87af3c6a500a027ddda4a9d1e64f26c8`. A path-bounded
`git status --short --untracked-files=all` over the exact 71-file inventory
reported `.gitignore` and `README.md` as modified and every other inventoried
file as untracked relative to that base. The full repository status also
contained excluded untracked scaffolding, this packet, and other non-inventory
records.

The configured FieldManual submodule was
`95702da77b342e95c6b52c5bb0e9f867c61a7662` (`heads/main`); it supplies
read-only guidance and is not packet material. Both `git diff --check` and
`git diff --cached --check` completed successfully with no diagnostics.

## Sponsor-Disclosed Implementation Pressure

The following are sponsor-disclosed design pressures in the current
implementation. They are not Panel findings, severities, dispositions, or
operator decisions. They are stated here so existing code and tests are not
mistaken for silent design approval:

- **Callback reentrancy and cooperative time bounds.** Command handlers run
  synchronously outside the state mutex but while the non-reentrant dispatch
  gate is held. The handler-supplied context marks that scope: when it is
  propagated, any nested App dispatch/reset or blocking snapshot wait returns
  `ErrWouldDeadlock`, including cross-App calls that could form `A -> B` /
  `B -> A` lock cycles. State setters, handler replacement, chord
  registration, source cleanup, and immediate snapshot reads are
  callback-safe. Go has no supported general goroutine identity, so replacing
  the supplied context with a value-dropping context or starting a dispatch
  goroutine and synchronously waiting for it remains unsupported and can
  deadlock. The 30-second default passes a deadline through the context but
  cannot preempt a handler that ignores cancellation, so one handler can hold
  dispatch progress and delay automation/server teardown without a hard
  wall-clock bound.
- **Intermediate publications.** Setters called by a command handler publish
  complete intermediate snapshots before the request method publishes its
  later correlated completion snapshot. A serialized request is therefore not
  an atomic multi-setter view transaction, and observers can see intermediate
  states without the request completion.
- **Control-value and presentation defaults.** A copied constructed `Panel`
  is rejected as a construction parent, but ordinary methods do not validate
  that their receiver is the canonical registered pointer; a copied receiver
  can mutate detached fields and trigger an unrelated App publication.
  Empty style IDs receive semantic defaults while omitted RGB values remain
  zero, so a zero style is black on black. Frame and GroupBox titles are
  immutable and clipped for painting, but constructor input has no byte or
  grapheme bound and each paint normalizes the complete title before
  truncation.
- **Mismatched retention and eventual exhaustion.** The current App keeps 64
  snapshots and up to 4,096 accepted raw-input, direct-command, and
  input-reset request IDs without eviction. The automation server separately
  keeps 32 exact snapshots and 256 protocol completions; each retained server
  completion currently owns its full snapshot even though `query_result`
  returns only a summary. An old completion may therefore outlive separately
  observable exact-snapshot history, protocol eviction does not make an
  App-level ID reusable, and 4,096 App-level requests permanently exhaust
  further input/command/reset acceptance for that App lifetime.
- **Reserved queue and limited fairness.** Version 1 advertises an
  `automation_queue` limit of 32, but exposes one synchronous outstanding
  request and implements no client-visible queue or pipelining. App dispatch
  is mutex-serialized, and no priority or stronger fairness guarantee exists
  among automation, human input, and other App callers beyond arrival at that
  gate.
- **Wait timeout indeterminacy.** `automation.Client.WaitSnapshot` derives its
  server timeout from the caller deadline rounded up to milliseconds while
  the client still enforces the original earlier deadline on transport reads.
  At the boundary, a client can time out before receiving the server's
  retained `wait_timeout` completion. The result must be treated as
  indeterminate and queried when possible, and it does not prove that no later
  snapshot was published around the deadline.
- **Endpoint trust remains operator policy, not authentication.** Version 1
  performs no peer authentication, peer-credential check, capability or
  per-command authorization, or elevated-process refusal. Socket mode `0600`,
  collision refusal, immediate-parent checks, and identity-safe cleanup are
  useful defenses, but the server does not establish safe ownership or mode
  for every ancestor path component. The operator-selected private path and
  non-risky context remain mandatory, and visible semantic/frame data can be
  sensitive.

## Known Questions For Review

1. Does the root/parent model preserve one control-tree node per control and
   permit a clean, idiomatic Go extension model without inheritance
   ambiguity?
2. Are geometry, clipping, normalization, paint order, snapshots, errors, and
   lifetime behavior sufficiently explicit and orthogonal?
3. Does the public API make multithreaded MVC/MVVC-like consumption safe and
   understandable without exposing internal owner topology or creating
   synchronous reentrancy deadlocks?
4. Does one-cell Unicode normalization handle composed text and every
   unsupported element deterministically without cursor or geometry drift?
5. Are raw key lifecycle events, held-state ownership, commands, interrupt,
   no-op, final-frame, duplicate-request, timeout, and cancellation semantics
   complete enough for closed-loop automation?
6. Are JSON decoding, line and collection limits, socket creation and cleanup,
   result retention, one-controller policy, and default-off startup adequate
   for the explicitly trusted initial boundary?
7. Does `expletives-test` apply useful public-consumer pressure, and can a
   client diagnose visual ownership, colors, controls, input, and final state
   without parsing terminal output?
8. Is the terminal adapter's narrow support claim honest and its exact
   restoration, signal, output-bound, and physical-degradation behavior safe?
9. Are performance costs and bounds acceptable for the first geometry limits,
   or does the API freeze an avoidable full-frame, allocation, lock, or
   retention problem?
10. Do ordinary, race, integration, PTY, process, build-mode, and attached
    automation tests cover the design's highest-risk invariants?

## Claimed Design Acceptance Criteria

A design-ready result requires evidence that:

- each public type and operation in the slice has one coherent contract,
  stable identity, bounded inputs, defined failures, and defined lifetime;
- the design remains composition-friendly, thread-safe, automation-native,
  and suitable for future controls and Layouts without claiming those later
  designs are approved;
- human input, headless tests, and attached raw events resolve through the
  same semantic command path;
- canonical intended frames remain independent of physical terminal
  degradation;
- automation is default-off, bounded, observable, request-correlated, and
  explicit about its unauthenticated trust boundary;
- terminal ownership and cleanup have a feasible, testable failure model;
- all supported commands can be built in all three required modes; and
- every confirmed blocker or major finding reaches one of the charter's
  operator-controlled gate resolutions.

## Sponsor-Supplied Validation For Chair Verification

The sponsor ran the following clean validation on 2026-07-30 immediately
before requesting packet freeze. The evidence chair must verify that these
commands apply to the inventoried bytes. Design reviewers may use the results
as feasibility evidence; delivery acceptance remains a separate review.

`make clean` completed, followed by `make verify`, whose implemented workflow
passed:

```text
gofmt check
go vet -mod=readonly ./...
go test -mod=readonly ./...
go test -mod=readonly -tags=integration ./...
CGO_ENABLED=1 go test -mod=readonly -race ./...
all debug, release, and profiling builds
all-mode version, help, and expletives-test self-check smoke runs
```

The six required Linux/amd64 artifacts existed at their exact paths with
executable mode `0755`. Their `--version` records reported Go 1.26.5, target
`linux/amd64`, the correct `debug`, `release`, or `profiling` mode, base
revision `df92af4f87af3c6a500a027ddda4a9d1e64f26c8`, and the expected dirty
worktree marker.

The additional compilation checks passed:

```text
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=readonly \
  -o .local/tmp/review-evidence.7JgrU7/expletives-test-linux-arm64 \
  ./cmd/expletives-test
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=readonly \
  -o .local/tmp/review-evidence.7JgrU7/expletivesctl-linux-arm64 \
  ./cmd/expletivesctl
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go test -mod=readonly -c \
  -o .local/tmp/review-evidence.7JgrU7/automation-linux-arm64.test \
  ./automation
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go test -mod=readonly -c \
  -o .local/tmp/review-evidence.7JgrU7/terminal-darwin-amd64.test \
  ./internal/terminal
```

The protocol decoder fuzz smoke run passed:

```text
go test -mod=readonly ./automation -run='^$' \
  -fuzz=FuzzDecodeRequest -fuzztime=3s
```

It executed 109,971 inputs after its 139-seed baseline and retained no
project-owned corpus artifact.

A freshly rebuilt debug `expletives-test` then ran headlessly on an explicit
project-local Unix socket. A real `expletivesctl` process:

1. received version-1 `hello` for
   `foundation.absolute-panels`, initial sequence 7, final false;
2. observed `panel.accent` as owner `control-4`, style `fixture.green`,
   background `#22C55E`;
3. submitted `down:control`, `press:r`, `up:control` on one connection and
   received applied sequences 8, 10, and 11, with Control held only during
   the first two completions and the accent changed to `fixture.magenta`
   background `#B820D0`;
4. invoked `scenario.reset` and received applied sequence 13 with the original
   green style, cell background, and owner restored; and
5. invoked shutdown and received outcome `exited`, sequence 14, a matching
   `app.quit` completion, `final: true`, and no held input.

The process exited with status 0, the socket was absent afterward, and the
operation removed its temporary cross-compile artifacts and owned empty
workspace. The bare `build/debug/expletivesctl` base case was also run: it
printed the supported command surface and returned usage status 2 as
documented.

## Proposed Roster

- Sponsor: `/root`
- Operator: project operator
- Evidence chair: `/root/design_freeze_chair`
- Maintainability: `/root/design_maintainability`
- Orthogonality and design elegance: `/root/design_orthogonality`
- Extensibility: `/root/design_extensibility`
- Customizability: `/root/design_customizability`
- Client ease of use: `/root/design_client_ease`
- Testability: `/root/design_testability`
- Security: `/root/design_security`
- Performance: `/root/design_performance`
- Reliability, concurrency, and portability: `/root/design_reliability`

These reviewer contexts are proposed only. Eligibility, independence, equal
packet access, first-round isolation, quorum, and findings remain pending.

## Addenda

None at freeze. This packet is immutable. All corrections and additional
evidence require a versioned addendum or replacement packet supplied
identically to every reviewer.
