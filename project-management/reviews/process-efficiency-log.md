# Development Process Efficiency Log

- Status: Active process-improvement log
- Opened: 2026-07-30
- Authority: Direct operator request during `EXPL-REV-001`
- Scope: Concrete opportunities to reduce development, review, integration,
  and verification overhead. This is not packet evidence, a Panel finding, a
  chair disposition, or an operator risk decision.

## Observed Opportunities

### 30. Resolve Hidden-View Overflow At The Shared Lifecycle Seam

The first Header/Footer catalog integration attached complete Layouts to
bands that are normally hidden. Their retained zero-sized geometry triggered
the global overflow fallback, so a menu audit appeared to invoke
`overflow.dismiss` instead of the selected entry. A demo-only workaround
would have repeated construction or special-cased every screen transition.

Improvement applied:

- define overflow as actionable only while the Layout owner is effectively
  visible;
- keep arrangement and snapshot geometry deterministic while hidden;
- end the episode on owner or ancestor hiding and begin a fresh episode on
  re-show when space still does not fit;
- suppress one-row chrome Layout overflow when tiny-surface priority denies
  the band any physical row; and
- cover the shared lifecycle once in core tests so later tabs, pages, and
  conditional containers inherit it without fixture-specific cleanup.

### 29. Update Only Changed Catalog Selection Commands

Screen navigation previously replaced every catalog-screen command definition
to recompute checked state, publishing once per screen. Each new page therefore
increased render, snapshot-copy, history, and automation-notification cost
even though exactly two checked values can change.

Improvement applied:

- remember the previous screen;
- uncheck only its command and check only the target command; and
- keep screen visibility plus StatusBar context in the existing single
  control-tree transaction.

This changes command-definition publications per navigation from the number
of catalog pages to exactly two. A future transactional bulk command-state API
may combine those final two publications when broader command-registry
mutation is designed.

### 28. Give Typed Control Details One Extension Checklist

Adding StatusBar state correctly touched the core union, core deep copy,
automation DTO, projection, automation deep copy, client validation,
aggregate bounds, maximum-response proof, and focused invalid-state tests.
Those are real trust-boundary responsibilities, but discovering the same
extension surface by search for the prior control type is slower and risks
missing one copy seam.

Improvement:

- keep a short project-owned checklist for every new typed ControlDetails
  member: core type and clone, renderer population, wire type and projection,
  wire clone, kind validator, aggregate limits, maximum-response proof, and
  valid/invalid/deep-copy tests;
- add a small generic Chord-copy helper when another chord-bearing detail
  arrives, rather than repeating pointer/slice copying again;
- retain explicit kind validators instead of reflection at the untrusted
  automation boundary; and
- calculate unique-control response overhead separately, as now done for
  MenuBar and StatusBar, rather than multiplying a unique large union member
  by every control slot.

Applied immediately: StatusBar painting and typed geometry share one render
plan, MenuBar/StatusBar use one application-chrome predicate, response
accounting treats both unique controls separately, and focused projection,
clone, validity, aggregate, and response-bound tests cover the complete seam.

### 27. Give Closed-Loop Commands A Built-In Compact Projection

The shortcut compatibility check needed only outcome, MenuBar open/selected
paths, and one screen-visible flag. `expletivesctl` emitted the complete
snapshot for each input event, and one omitted projection produced more than
55,000 output tokens. Multi-event `keys` also returns an array while
single-event commands return one object, so an ad hoc query initially assumed
the wrong shape.

Improvement:

- add a built-in concise output mode that normalizes single and multi-event
  results and selects outcome, sequence, final state, menu paths, requested
  control fields, and a bounded frame sample;
- document that mode in the ordinary closed-loop recipe so full snapshots are
  deliberate evidence acquisitions rather than the default diagnostic path;
- keep the full response available for artifact capture and deep inspection;
  and
- consider a future bounded server-side projection only as a versioned
  automation-protocol extension, because client-side filtering reduces local
  output but not transport and parse cost.

Applied immediately: the remainder of `EXPL-TASK-022` piped live results
through a compact normalized projection and recorded only the changed menu
paths and About visibility.

### 26. Exercise New Semantic States Through The Public Client Early

The core Menu tests accepted the new F10 bar-active state, but the first live
attached request exposed a duplicated automation-client invariant that still
required every selected root to be open. The full gate would also have caught
it, but only after more unrelated work and output.

Improvement:

- add a focused automation validation fixture whenever a core semantic state
  gains a new legal path shape;
- run one reduced public-client projection immediately after that state first
  works in core tests; and
- keep the reduced query limited to the changed semantic path and a few frame
  cells rather than serializing the complete expanded snapshot.

Applied immediately: bar-active and selected-disabled Menu fixtures now pass
the public-client validator, invalid deeper selections without an open popup
remain rejected, and the live check used a compact menu/row projection.

### 23. Bound Frame-Heavy Failure Diagnostics

One attached-automation assertion formatted an entire completion when a
single overflow-count check failed. Because the public client intentionally
expands frame cells, that one diagnostic produced tens of thousands of tokens
and obscured the actual one-cell minimum mismatch.

Improvement:

- report the small predicate fields that caused a failure rather than the
  complete snapshot or completion;
- reserve full frame serialization for an explicitly acquired diagnostic
  artifact; and
- keep stable keys, counts, outcomes, sequence numbers, and relevant bounds
  in ordinary test failures.

Applied immediately: the `expletives-test` attached integration assertions now
emit concise scenario/count/outcome/geometry evidence instead of expanding
the complete frame on failure.

### 24. Make Multi-Event Controller Output Easier To Query

The Actions closed loop correctly required one persistent controller
connection for a raw KeyDown/KeyUp pair: invoking `expletivesctl key`
separately disconnects between events and therefore clears source-local held
and pressed state by design. The supported `keys` command solves that
lifecycle problem, but it changes the JSON result from one completion object
to an array. A routine projection initially assumed the single-command shape
and had to be corrected.

Improvement:

- retain `keys` as the documented way to send one stateful chord or
  press/release sequence from a single input source;
- add a future concise/final-result output option that selects the last
  completion while preserving the full completion array as the default
  evidence;
- make `expletivesctl` help state explicitly that separate process
  invocations intentionally cannot preserve held or pressed input; and
- keep core disconnect cleanup strict rather than weakening input isolation
  to make shell composition appear stateful.

### 25. Make Cross-Target Compile Checks A Named Target

A direct `GOOS=linux GOARCH=arm64 go test ./... -run '^$'` compiled arm64 test
binaries and then tried to execute them on the amd64 host. Repeating the check
with `-exec=/bin/true` proved the intended compile-only outcome, but the
correct incantation should not need rediscovery at every checkpoint.

Improvement:

- add a named cross-target compile target that pins GOOS, GOARCH, and
  `CGO_ENABLED=0`;
- use `go test -exec=/bin/true ./...` or an equally bounded compile-only
  mechanism that creates no retained binaries;
- include that target in the declared verification contract once the target
  matrix is approved; and
- distinguish compile evidence from runtime support claims.

### 1. Use A Small Contract-First Packet

The first design packet inventories 71 files and asks every reviewer to scan
the whole set. That is reproducible but expensive and encourages repeated
reading of historical and low-signal material.

Improvement:

- make the packet itself the complete contract summary;
- include only the minimum normative specifications, public API, architecture
  diagrams, and representative implementation seams in the primary reading
  set;
- place remaining files in a manifest-backed reference appendix with explicit
  routing by lens; and
- require reviewers to inspect the complete packet but only the referenced
  source needed to substantiate a finding.

### 2. Separate The Manifest From The Narrative Packet

Embedding a large generated manifest inside the Markdown packet made an
administrative freeze cumbersome and exposed a partial-edit failure mode.

Improvement:

- store the canonical manifest in a separate immutable
  `*.sha256` review artifact;
- hash the narrative packet and manifest separately in a small convening
  record;
- provide a project-owned command that validates inventory, sort order,
  hashes, packet hash, base revision, and dirty-state disclosure; and
- publish only after the command completes atomically.

### 3. Automate Freeze And Convening

The chair manually reconciled inventory, packet placeholders, roster,
timestamps, hashes, and the convening record. Interrupted work left a
temporarily inconsistent pre-distribution packet that required recovery.

Improvement:

- add a deterministic `review freeze` helper that writes a new version rather
  than editing a Draft in place;
- validate every required placeholder and refuse partial Frozen state;
- generate the chair record from the same transaction; and
- add a `review verify` command reviewers can run in one step.

### 4. Bound Reviewer Work And Output

Unbounded instructions to review the whole implementation led to long turns
and extensive overlapping findings.

Improvement:

- give each seat a short risk-ranked routing guide;
- set a default maximum of three material findings, with an explicit exception
  only for additional blockers;
- time-box source exploration after packet and manifest verification;
- ask for concise claim/evidence/remedy records rather than repeated contract
  summaries; and
- require the chair to identify likely duplicates before requesting any
  follow-up analysis.

### 5. Avoid Repeating Expensive Validation

The sponsor, specification auditors, and chair can otherwise rerun the same
test matrix even though design review needs feasibility rather than delivery
proof.

Improvement:

- retain one machine-readable validation record containing command, toolchain,
  target, start/end time, exit status, and output digest;
- let the chair verify its binding to the manifested bytes;
- reserve independent full reruns for the delivery review or a disputed
  claim; and
- keep design review focused on contracts and architectural feasibility.

### 6. Make Independence Cheap To Enforce

First-round independence currently depends on careful sequencing and delaying
all shared finding files until nine long-running contexts finish.

Improvement:

- give each reviewer an isolated output spool unavailable to other seats;
- publish all spools atomically only after quorum submissions exist;
- let the orchestrator enforce read restrictions and record exposure
  automatically; and
- support bounded concurrent waves without completed contexts consuming
  active review capacity.

### 7. Use Structured Findings

Large free-form Markdown responses require mechanical copying and make
validation, duplicate mapping, counting, and addendum routing harder.

Improvement:

- collect findings in a small versioned JSON or TOML schema;
- validate IDs, seat, stage, packet hash, severity, citations, and required
  fields automatically;
- render durable Markdown from the validated records; and
- give the evidence chair a generated duplicate/disposition worksheet.

### 8. Keep Design And Delivery Reviews Proportional

Separate design and delivery gates are valuable, but repeating nine broad
full-tree reviews would double overhead without doubling assurance.

Improvement:

- keep the nine independent design lenses for foundational architecture;
- make delivery reviewers inspect a manifest-bound diff, confirmed design
  finding resolutions, and risk-ranked verification evidence;
- route unchanged design areas by reference instead of rereading them; and
- document a lean exemption for mechanical fixes already covered exactly by
  confirmed design resolutions.

### 9. Release Completed Reviewer Capacity Explicitly

The review runner currently limits the active thread tree, and completed
review contexts can temporarily occupy capacity needed by a new independent
seat. That turned a nominally parallel nine-seat review into avoidable waves
and made reviewer lifecycle management part of the critical path.

Improvement:

- distinguish active-concurrency limits from retained-result limits;
- persist a completed result to the isolated spool and retire its execution
  context immediately;
- let the orchestrator start the next independent identity without discarding
  the persisted result; and
- expose capacity and lifecycle state directly so scheduling does not depend
  on failed spawn attempts.

### 10. Route Duplicate-Prone Themes Before Review

The first five seats returned 27 findings. Several independently converge on
the control extension seam, fixture coupling in reusable automation, terminal
input ownership, and snapshot/request lifecycle. Independent convergence is
useful evidence, but every full narrative still carries copying and
adjudication cost.

Improvement:

- retain independent voting-free review, but provide a narrow question set per
  lens;
- accept a compact `supports concern <topic>` record when a seat independently
  discovers an already anticipated architectural question;
- reserve a full finding narrative for new evidence, a different consequence,
  or a materially different remedy; and
- have tooling expand compact support records into the chair's convergence
  table without erasing reviewer independence.

### 11. Enforce The Exploration Budget, Not Just The Output Limit

The three-finding cap reduced returned material, but it did not by itself stop
reviewers from continuing broad exploration before choosing those three
findings. One bounded seat had to be interrupted and instructed to finalize
from evidence already gathered.

Improvement:

- give each seat an explicit elapsed-time or inspected-file budget in addition
  to the finding cap;
- have the runner issue a warning and then automatically close exploration
  while preserving time for structured output;
- record packet-verification time separately from substantive review time;
  and
- treat a concise no-finding submission as a successful bounded result, not
  an invitation to search until something is found.

### 12. Eliminate Manual Submission Relay And Chair Re-entry

Completed reviewer responses were available to the orchestrating context but
not reliably available to later transcription contexts. Publishing the 39
records therefore required manually relaying summarized fields. The chair
then had to reread those Markdown records and reconstruct an overlap matrix.

Improvement:

- persist each validated structured submission directly from the reviewer
  turn, while keeping it hidden from other active seats;
- publish the nine spools as a single atomic operation when quorum arrives;
- generate Markdown finding records without a separate transcription turn;
- compute exact and candidate-semantic duplicate groups from structured
  fields before the chair starts; and
- give the chair one worksheet containing citation existence checks, packet
  membership, duplicate candidates, and empty disposition fields.

The chair must still judge whether evidence supports a claim. Tooling should
remove byte copying, field validation, missing-path discovery, and repeated
record parsing from that judgment.

### 13. Keep Routine Feature Work In One Task And One Verification Loop

The root-sizing and border work was substantial but followed already-directed
architecture. A formal Panel would have added scheduling and record overhead
without supplying missing authority.

Improvement:

- use one active task, one durable behavior specification, focused unit and
  integration tests, and one final verification record;
- publish Ubersight only at phase transitions;
- reserve specialist review for a specific unresolved risk; and
- do not create a proposal merely to restate a direct, reversible operator
  decision.

### 14. Give Large Evidence A Compact Transport Without Sacrificing Inspection

Expanding every automation frame on the wire made a reasonable 1200 by 1200
desktop conflict with response and retention budgets. Removing the
human-friendly indexed cell view would instead slow closed-loop diagnosis.

Improvement:

- retain canonical expanded cells in the core and client API;
- run-length encode complete cells only across the socket boundary;
- expand and validate once in the reusable client so every consumer shares
  the safety logic; and
- keep one executable worst-case response proof rather than manually
  recalculating the bound in each review.

### 15. Reduce Specification Drift With One New Contract And Targeted Repairs

The old fixed frame limits appeared in the Go API, implementation baseline,
automation protocol, demo requirements, and project instructions. Updating
the same narrative in many places is slow and creates inconsistent authority.

Improvement:

- put the complete sizing/border behavior in one focused specification;
- keep other documents to short summaries and links where possible;
- use a narrow stale-value search as a release check; and
- treat historical review packets as immutable evidence rather than updating
  them to describe current behavior.

## Actions During This Review

- Preserve the already-started nine-seat independent Round 1 so its process is
  not silently changed midstream.
- For remaining seats, route attention by lens and cap ordinary output at
  three material findings, allowing extra records only for blockers.
- Do not rerun the full verification matrix during each design review.
- Publish raw findings only after all nine independent submissions are
  complete.
- Have the chair validate citations and collapse work through duplicate maps
  before requesting sponsor mitigations.
- Record live counts and elapsed review waves so the final process proposal
  can compare the bounded seats with the first five unbounded seats.
- Enforce conclusion at the remaining seats' exploration boundary; do not let
  the output cap merely hide unbounded reading time.

## Live Results

Round 1 reached all nine independent seats with 39 raw findings:

- the first five, reviewed under the original broad instructions, returned 27
  findings, or 5.4 per seat;
- the final four, capped at three ordinary material findings, returned 12
  findings, or 3.0 per seat; and
- the cap therefore reduced raw record volume per seat by about 44 percent.

The bounded seats still required active conclusion management: three were
interrupted at the exploration boundary and told to format only evidence
already gathered. The cap reduced downstream transcription and adjudication
volume, but an enforceable exploration deadline is still needed to reduce
review latency.

The final four did not merely omit known severe issues: they independently
reported two additional blocker claims, covering unbounded title processing
and cancellation-unbounded shutdown. The bounded approach therefore retained
material risk discovery in this sample. One review is not enough evidence to
claim equivalent coverage generally.

Chair adjudication classified 21 findings as canonical and confirmed, 17 as
duplicates, and 1 as already addressed. Duplicate mapping therefore removed
about 44 percent of the raw records from the remediation decision surface
without deleting their independent evidence. The remaining 21 canonical
records cluster into seven implementation/design workstreams:

1. control node, extension, lifetime, copy safety, snapshot detail, and theme;
2. atomic updates, owner marshaling, dispatch reentrancy, and shutdown;
3. automation identity, retention, versioning, targets, timeout, and cancel
   semantics;
4. reusable incremental terminal input and process-level PTY verification;
5. command registry and bounded diagnostic results;
6. public Go documentation and examples; and
7. bounded displayed text and maximum-response proof.

Using those seven workstreams for one shared mitigation addendum avoids
treating 21 convergent symptoms as 21 unrelated design projects.

## Follow-Up

### 13. Make The Formal Panel Exceptional

The initial charter made broad categories automatic triggers and normally
required separate design and delivery reviews. That would turn ordinary
implementation and localized bug fixing into administrative review projects.

Operator direction on 2026-07-30 changed the default:

- normal work uses proportionate tests and focused peer review;
- a formal Panel is reserved for genuinely major, cross-cutting,
  hard-to-reverse issues;
- a convened Panel selects an odd three, five, or nine relevant lenses rather
  than automatically using all nine;
- one review round, one combined findings record, and targeted follow-up are
  the defaults; and
- `EXPL-REV-001` closes with implementation evidence and a concise chair
  addendum, without an automatic second nine-seat review.

This policy change is now reflected in the charter and `AGENTS.md`.

### 14. Reduce Repeated Trust-Boundary JSON Passes

The automation codec currently token-validates bounded JSON, decodes a field
map or header, and then decodes the concrete record. This preserves duplicate,
depth, unknown-field, and size checks, but costs three or four full passes and
additional allocations.

Improvement:

- retain the current correct bounded validation while protocol work is still
  changing;
- remove redundant validations when the same invariant is already established
  at one boundary—the current correction removed one extra hello validation;
  and
- later evaluate one bounded scanner plus typed decode pipeline that preserves
  all existing trust-boundary checks with fewer passes.

### 15. Give Parallel Workstreams Stable Integration Windows

The terminal and automation workstreams owned disjoint packages, but both
depended on a root API that was temporarily uncompilable during refactoring.
They spent time polling, reporting the same compile blocker, and rerunning
tests after brief green windows.

Improvement:

- land the smallest compiling public type/API seam before deeper internals;
- announce explicit stable integration points rather than having workers poll;
- keep root-breaking edits short and use compile-only checks before inviting
  downstream verification; and
- for longer incompatible migrations, use isolated worktrees or stage the
  compatibility adapter first.

### 16. Separate Sandbox-Denied Checks From Ordinary Fast Tests

Unix-socket integration tests repeatedly reached the same managed-sandbox
`setsockopt: operation not permitted` denial. Re-running them in ordinary test
bundles adds latency without new evidence.

Improvement:

- keep pure unit and compile checks in the fast unprivileged lane;
- group permission-dependent Unix-socket and PTY tests behind one declared
  verification target;
- run that target once with the required approved permissions after the tree
  is stable; and
- preserve stderr in harness failures so a sandbox denial is immediately
  distinguishable from a product timeout.

### 17. Rebuild Process-Test Artifacts As Prerequisites

The first real-process PTY runs exercised an older debug binary even though
the source-level decoder tests were current. That produced a plausible but
misleading input failure and repeated diagnosis.

Improvement:

- make every process integration target depend on the exact artifact it
  executes;
- rebuild that artifact through the normal mode target before starting the
  test;
- have failures print the artifact path, embedded build mode, and captured
  process stderr; and
- keep source-only unit tests separate from artifact/process tests so their
  evidence cannot be confused.

### 18. Keep One Durable Owner For Status

The first foundation slice repeated its current state in the active-task
record, backlog, roadmap, product goals, root README, proposal, review
records, and Ubersight. Even small state changes then created a wide
administrative edit surface and stale, contradictory summaries.

Improvement:

- keep task status in one active or completed task record;
- keep behavior in specifications, durable choices in the decision log, and
  defects in the bug tracker;
- make roadmap, README, proposal, and review text link to those records
  instead of mirroring volatile progress prose;
- update durable state only at meaningful transitions or closure; and
- publish Ubersight only when a phase, blocker, or verification state
  materially changes, not after every command.

### 19. Do Not Rerun The Full Suite To Select One Tagged Test

The integration target enabled the PTY build tag across `./...`, which reran
all ordinary package tests before executing the one tagged real-process test.

Improvement applied:

- keep the ordinary suite in `test-unit`; and
- make `test-integration` rebuild the exact debug artifact and select only the
  tagged PTY lifecycle test in its owning package.

### 20. Clean Cross-Build Artifacts At The End Of Their Check

The initial review verification left four disposable cross-build outputs
using 21 MiB under `.local/tmp` after their evidence had been recorded.

Improvement applied:

- cross-build into one uniquely owned temporary directory;
- retain the command result, not duplicate binaries, as evidence; and
- remove only those exact outputs and their empty directory immediately after
  the check.

### 21. Project Large Snapshots Before Inspecting Them

The first live Layout observation printed the complete cell array and
truncated the useful Layout tail. A compact `jq` projection immediately made
geometry, stack indices, overflow state, and selected frame cells readable.

Improvement applied:

- query the ordinary immutable snapshot, then project only the semantic
  records and diagnostic cells relevant to the current hypothesis;
- preserve the full protocol path instead of adding a diagnostic-only server
  operation;
- use stable automation keys in projections so runtime IDs remain evidence,
  not selection inputs; and
- reserve full frame output for capture artifacts or cases that genuinely
  require every cell.

The Text/Display checkpoint repeated this mistake at the command line, which
shows that caller discipline alone is too fragile. A later focused
`expletivesctl` improvement should offer a concise summary or field-selection
mode while retaining full JSON as the compatibility default.

### 22. Arrange Owner Layouts In Control-Tree Dependency Order

A simultaneous transaction attached Layouts to a managed parent and its child
containers. Iterating an App map arranged a child-owner Layout before its
parent Layout had assigned the child's bounds, producing transient zero-sized
geometry. The attached snapshot exposed the issue immediately.

Improvement applied:

- measure all Layout roots first;
- arrange from the App root through control-tree order so parent geometry is
  committed before child-owner Layouts consume it;
- keep one transaction and publication rather than adding retry passes; and
- retain a same-transaction nested fixture so this dependency cannot regress.

### 23. Derive Catalog Audits From One Menu Manifest

The exhaustive catalog audit deliberately enumerates every command path and
disabled entry, but its test table currently repeats keys and mnemonics already
declared by the demo Menu tree. That is fast enough now, yet each new control
phase would otherwise require synchronized edits in construction, path tests,
and documentation.

Improvement:

- introduce one private immutable catalog descriptor when the next page is
  enabled;
- derive Menu models and exhaustive audit cases from that descriptor while
  keeping behavior contracts and prose independently reviewed;
- retain explicit assertions for expected counts, placement, grouping, and
  outcomes so generation cannot hide accidental catalog growth; and
- do not broaden the public toolkit API merely to reduce test-application
  duplication.

## Adopted Going-Forward Policy

The charter simplification is the immediate process correction. Do not build
a review-freeze tool merely to automate a workflow that should usually not
run. The single-owner status policy prevents ordinary implementation from
becoming a documentation synchronization exercise. If future major reviews
show repeated freeze/verification cost, propose the smallest helper justified
by those actual reviews.
