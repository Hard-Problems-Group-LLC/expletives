# EXPL-REV-001 Design Evidence

- Review ID: `EXPL-REV-001`
- Stage: design
- Record status: Round 1 complete; confirmed findings corrected and verified
- Evidence chair: `/root/design_freeze_chair`
- Freeze time: `2026-07-30T03:42:53-07:00`
- Frozen packet:
  `project-management/reviews/packets/EXPL-REV-001-foundation-design-v1.md`
- Frozen packet SHA-256:
  `246e29b3e5e86dbdb66811db56cba3007a395f0685a5b63cb8b444714500c0b9`
- Manifest integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`

## Freeze Validation

The evidence chair validated the configured project root as this checkout and
the configured framework root as its `FieldManual/` submodule. The packet's
explicit inventory contains 71 unique repository-relative paths, all of which
existed as regular files at freeze. It exactly covered all 38 current Go
source and test files, all 11 files under `docs/specifications/`, and the root
build inputs `.gitignore`, `Makefile`, `go.mod`, and `go.sum`.

The embedded manifest was generated over exactly those 71 paths, sorted
bytewise by path under `LC_ALL=C`. Its integrity identifier is SHA-256 over
the exact manifest lines, including every terminating LF. The packet itself
is excluded from the embedded manifest and was hashed only after its final
edit. It has not been edited since the packet SHA-256 above was computed.

The Git base at freeze was
`df92af4f87af3c6a500a027ddda4a9d1e64f26c8`. A path-bounded
`git status --short --untracked-files=all` over the inventory reported
`.gitignore` and `README.md` as modified and the other 69 inventoried files as
untracked relative to that base. Full repository status also contained
excluded untracked scaffolding, the packet, and other non-inventory records.
The configured FieldManual submodule was
`95702da77b342e95c6b52c5bb0e9f867c61a7662` (`heads/main`) and was excluded
from packet material. `git diff --check` and `git diff --cached --check`
completed with no diagnostics.

## Sponsor-Supplied Validation Evidence

The following is recorded as sponsor-supplied feasibility evidence for the
inventoried bytes. The evidence chair did not independently rerun these
commands or exercises:

- `make clean` followed by `make verify`, including formatting, vet, ordinary,
  integration, and race tests; all three required build modes; and version,
  help, and self-check smoke runs;
- the stated Linux/arm64 command and automation-test cross-compilations and
  Darwin/amd64 terminal-test compilation;
- the stated three-second `FuzzDecodeRequest` smoke run, reported as 109,971
  executions after 139 seeds with no retained project corpus artifact; and
- the stated external closed-loop `expletives-test` and `expletivesctl`
  exercise covering hello, snapshot observation, held Control input,
  accent-state transition, scenario reset, correlated final shutdown, clean
  process exit, and socket cleanup.

These sponsor claims are packet evidence, not independent chair validation,
design approval, or delivery acceptance.

## Convened Participants And Independence

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

All nine proposed isolated reviewer contexts submitted one independent Round 1
record set through their assigned lens. Before review, every seat verified the
frozen packet SHA-256 and manifest integrity identifier recorded above, was
instructed not to read other reviewers' findings, and received no other raw
findings. Raw records were withheld until all nine seats had submitted. The
evidence chair is distinct from the sponsor, operator, and nine reviewers.
Round 1 therefore has complete nine-lens coverage and satisfies the charter's
quorum requirement.

## Round 1 Counts

The 39 immutable raw claims comprise 3 blocker, 35 major, and 1 minor record.
Chair dispositions are 21 `confirmed`, 17 `duplicate`, 1 `already addressed`,
0 `not substantiated`, and 0 `out of scope`.

| Seat | Raw blocker | Raw major | Raw minor | Confirmed | Duplicate | Already addressed |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| S01 Maintainability | 0 | 5 | 0 | 3 | 1 | 1 |
| S02 Orthogonality | 0 | 8 | 1 | 5 | 4 | 0 |
| S03 Extensibility | 1 | 4 | 0 | 4 | 1 | 0 |
| S04 Customizability | 0 | 3 | 0 | 2 | 1 | 0 |
| S05 Client ease of use | 0 | 5 | 0 | 4 | 1 | 0 |
| S06 Testability | 0 | 3 | 0 | 1 | 2 | 0 |
| S07 Security | 0 | 3 | 0 | 0 | 3 | 0 |
| S08 Performance | 1 | 2 | 0 | 1 | 2 | 0 |
| S09 Reliability, concurrency, portability | 1 | 2 | 0 | 1 | 2 | 0 |
| **Total** | **3** | **35** | **1** | **21** | **17** | **1** |

The 21 canonical confirmed records retain their reviewers' raw severities:
3 blocker, 17 major, and 1 minor.

## Duplicate Map

- `EXPL-REV-001-D-S01-F004` -> `EXPL-REV-001-D-S03-F001`
- `EXPL-REV-001-D-S02-F001` -> `EXPL-REV-001-D-S03-F001`
- `EXPL-REV-001-D-S02-F002` -> `EXPL-REV-001-D-S03-F001`
- `EXPL-REV-001-D-S02-F006` -> `EXPL-REV-001-D-S01-F001`
- `EXPL-REV-001-D-S02-F007` -> `EXPL-REV-001-D-S01-F002`
- `EXPL-REV-001-D-S03-F003` -> `EXPL-REV-001-D-S01-F003`
- `EXPL-REV-001-D-S04-F002` -> `EXPL-REV-001-D-S03-F001`
- `EXPL-REV-001-D-S05-F002` -> `EXPL-REV-001-D-S01-F002`
- `EXPL-REV-001-D-S06-F001` -> `EXPL-REV-001-D-S02-F003`
- `EXPL-REV-001-D-S06-F002` -> `EXPL-REV-001-D-S02-F004`
- `EXPL-REV-001-D-S07-F001` -> `EXPL-REV-001-D-S02-F004`
- `EXPL-REV-001-D-S07-F002` -> `EXPL-REV-001-D-S02-F004`
- `EXPL-REV-001-D-S07-F003` -> `EXPL-REV-001-D-S09-F001`
- `EXPL-REV-001-D-S08-F002` -> `EXPL-REV-001-D-S02-F003`
- `EXPL-REV-001-D-S08-F003` -> `EXPL-REV-001-D-S02-F004`
- `EXPL-REV-001-D-S09-F002` -> `EXPL-REV-001-D-S02-F004`
- `EXPL-REV-001-D-S09-F003` -> `EXPL-REV-001-D-S01-F001`

No raw claim, evidence, recommendation, severity, or resolution field was
removed or rewritten when duplicates were mapped.

## Canonical Confirmed Gate Themes

- `EXPL-REV-001-D-S01-F001` — reusable incremental terminal-input decoding;
- `EXPL-REV-001-D-S01-F002` — fixture identity and commands in generic
  automation defaults;
- `EXPL-REV-001-D-S01-F003` — shared fixed snapshot shape across core, wire,
  terminal, and future control detail;
- `EXPL-REV-001-D-S02-F003` — no atomic update/invalidation/publication
  boundary;
- `EXPL-REV-001-D-S02-F004` — incompatible request, completion, and snapshot
  retention authorities and lifetimes;
- `EXPL-REV-001-D-S02-F005` — context-value reentrancy guard and unsupported
  cross-App composition;
- `EXPL-REV-001-D-S02-F008` — copied Panel detached mutation and publication;
- `EXPL-REV-001-D-S02-F009` — exposed queue/cancel surface without usable
  execution semantics;
- `EXPL-REV-001-D-S03-F001` — unresolved public control-extension seam
  (blocker);
- `EXPL-REV-001-D-S03-F002` — no interoperable protocol-version transition;
- `EXPL-REV-001-D-S03-F004` — unresolved owner-safe mutation and marshaling
  contract;
- `EXPL-REV-001-D-S03-F005` — append-only control lifetime and finite control
  capacity;
- `EXPL-REV-001-D-S04-F001` — no coherent App-level semantic style/theme
  resolution model;
- `EXPL-REV-001-D-S04-F003` — fragmented, non-inspectable command and binding
  policy;
- `EXPL-REV-001-D-S05-F001` — discarded handler error diagnostics;
- `EXPL-REV-001-D-S05-F003` — ambiguous and unresolvable command target;
- `EXPL-REV-001-D-S05-F004` — CLI timeout shorter than server execution
  budget;
- `EXPL-REV-001-D-S05-F005` — critical public contracts absent from Go
  documentation;
- `EXPL-REV-001-D-S06-F003` — no full interactive process PTY verification;
- `EXPL-REV-001-D-S08-F001` — unbounded title normalization and response-size
  exposure (blocker); and
- `EXPL-REV-001-D-S09-F001` — non-cancellable dispatch acquisition and
  unbounded handler/shutdown path (blocker).

`EXPL-REV-001-D-S01-F005` is `already addressed`: the project instructions
and frozen packet explicitly establish current-specification precedence and
classify the older Draft/proposal text as design input or provenance.

## Citation Limitations

Three raw records preserve a nonexistent cited path:
`project-management/proposals/expl-prop-2026-02-25-public-api-and-automation.md`
in `EXPL-REV-001-D-S02-F001`, `EXPL-REV-001-D-S02-F003`, and
`EXPL-REV-001-D-S02-F006`. The frozen inventory instead contains
`project-management/proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md`.
The remaining valid frozen citations independently substantiate each
disposition, so the citation defects do not change the outcomes.

`EXPL-REV-001-D-S08-F001` uses the shorthand citation `packet :438-439`;
those lines resolve unambiguously to the frozen packet's bounded-input
acceptance criterion. No broader design research or verification-matrix rerun
was used for chair dispositions.

## Correction Closure Addendum

- Closure time: `2026-07-30T06:00:24-07:00`
- Process authority:
  [`EXPL-DEC-010`](../../decision-log.md#expl-dec-010--reserve-the-panel-for-major-issues)
- Correction design:
  [`EXPL-PROP-002`](../../proposals/under-review/expl-prop-002-foundation-review-corrections.md)

The seven shared workstreams resolved all 3 confirmed blockers, 17 confirmed
majors, and 1 confirmed minor. The original raw findings and duplicate map
remain unchanged above.

| Workstream and findings | Resolution evidence |
| --- | --- |
| Controls, extension, lifetime, snapshot, and theme — S01-F003, S02-F008, S03-F001, S03-F005, S04-F001 | Copy-safe canonical control handles, sealed shallow capabilities, typed detail records, destruction/capacity reclamation, semantic Theme resolution, and external-consumer tests are specified in [`go-api-v0.md`](../../../docs/specifications/go-api-v0.md) and implemented in `panel.go`, `theme.go`, and their tests. |
| Atomic updates, owner, dispatch, and shutdown — S02-F003, S02-F005, S03-F004, S09-F001 | Atomic Transactions, cancellation-aware bounded gates, safe callback boundaries, cross-App dispatch, and orderly final publication are implemented in `transaction.go` and `input.go` with contention, reentrancy, race, timeout, and capacity tests. |
| Automation ownership and evolution — S01-F002, S02-F004, S02-F009, S03-F002, S05-F003, S05-F004 | The automation-owned DTO, one retention authority, strict Hello negotiation, stable key targeting, bounded timeouts, reusable client, and indeterminate reconciliation are specified in [`automation-protocol-v1.md`](../../../docs/specifications/automation-protocol-v1.md) and covered throughout `automation/`. |
| Terminal input and PTY — S01-F001, S06-F003 | The reusable incremental decoder covers fragmented UTF-8, CSI/SS3, Alt/Control chords, function/navigation keys, and paste/discard paths; the tagged controlling-PTY process test verifies fragmentation, resize, Ctrl-C, exit status, and exact terminal restoration. |
| Commands and diagnostics — S04-F003, S05-F001 | The inspectable command registry, structured bindings, stable targets, bounded diagnostics, local-only causes, handler serialization, and failure paths are implemented and tested in `input.go` and `input_test.go`. |
| Public Go documentation — S05-F005 | Every exported root, terminal, and automation identifier now has contract documentation covering defaults, ownership, copying, concurrency, cancellation, trust, and returned-data ownership; public examples and an external-package MVC consumer test compile and pass. |
| Bounded text and response proof — S08-F001 | Border titles normalize to bounded canonical one-cell text, every wire field is bounded and validated, and the incremental encoder rejects at its configured cap. The proven legal maximum is 31,570,899 bytes; four retained maxima total 126,283,596 bytes below 128 MiB. |

Closure verification passed:

- `make verify`: formatting, vet, ordinary tests, the controlling-PTY process
  test, the full race suite, all build modes, and smoke checks;
- `make clean` followed by a fresh `make all`;
- CGO-free Linux arm64 cross-builds for both commands;
- three-second `FuzzDecodeRequest` and `FuzzInputDecoder` smokes; and
- a fresh external `expletivesctl` session covering strict Hello, snapshot
  inspection, raw Control-R, direct reset, correlated final shutdown, clean
  process exit, and socket cleanup.

No confirmed finding requires deferral or operator risk acceptance. This
addendum records technical correction and verification; it does not substitute
for an operator product or scope decision. Under `EXPL-DEC-010`, mechanical
verification of the already-reviewed findings does not trigger a second formal
Panel or separate delivery review.
