# Bug: ListBox Prose Forces Ordinary-Width Horizontal Scrolling

- ID: `EXPL-BUG-002`
- Status: Candidate verified; publication pending
- Priority: High consumer usability
- Reported: 2026-08-07T11:02:54-07:00
- Reporter: radioradio operator via `radioradio-ECR-2026-003`
- Owner: Codex
- Related work: Phase 19 Slice 19.5; Collections API v0

## Symptom And Impact

`ListBox` renders every logical item on exactly one physical row. A prose
description wider than the Panel Client Area therefore forces an automatic
horizontal scrollbar at normal terminal widths. Consumers cannot implement
word-wrapped message histories with hanging indents without fabricating
additional logical keys and breaking navigation, selection, retention, and
automation counts.

## Reproduction Or Evidence

Create a bordered `ListBox` with `HorizontalBar: auto`, a metadata label, and
a description longer than the client width. Core and automation details show
`HorizontalVisible: true`; the frame shows one clipped row and an integrated
horizontal bar. radioradio reproduced this in its live Channel Monitor.

## Expected Behavior

An opt-in word/cell policy derives multiple visual rows from one stable
logical item. Label-plus-description continuation rows align beneath the
description, current/selection styling covers the complete item, resize
reflows deterministically, and ordinary-width wrapped content has no
horizontal overflow. The zero value remains single-line compatible.

## Actual Behavior

The public model and renderer expose no wrapping policy. `displayRow`
concatenates marker, label, separator, and description; `reflowListBox` fixes
content height to logical item count and width to the longest complete row.

## Root Cause

The Phase 16 directed contract intentionally fixed one visual row per item and
retained horizontal scrolling for long content. The later message-monitor
consumer supplied a valid variable-height use case that the v0 contract did
not cover.

## Resolution

The verified local candidate adds `ListBoxOptions.Wrap`, preserves logical
item state, derives bounded visual rows with description-aligned hanging
indents, exposes normalized wrap/visual-row evidence, and extends contract and
regression tests. Publishing that dependency change remains an explicit
operator authorization boundary.

## Validation

At 2026-08-07T11:43:43-07:00 the final Expletives gate passed vet,
ordinary tests, debug/release/profiling builds, PTY integration, and the full
race suite. The complete downstream radioradio fake-only gate also passed.
Compact-width coverage proves that an over-height logical item retains its
leading row without horizontal overflow. The pre-existing 30-second aggregate
deadline for the headless catalog failed on the unchanged pinned checkout as
well, so the harness now permits two minutes while preserving every protocol
deadline and assertion; observed race runs completed in 70.25 and 95.93
seconds.

## History

- 2026-08-07T11:14:29-07:00 — Accepted as Phase 19 Slice 19.5; local
  implementation began without publishing an upstream commit.
- 2026-08-07T11:31:14-07:00 — Complete local verification passed; candidate
  awaits publication authorization and downstream pinning.
- 2026-08-07T11:43:43-07:00 — Compact over-height leading-row correction and
  both final complete project gates passed.
