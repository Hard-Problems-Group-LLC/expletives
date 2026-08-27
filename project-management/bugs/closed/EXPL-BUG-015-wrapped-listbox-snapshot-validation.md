# Bug: Wrapped ListBox Snapshot Failed Matching Client Validation

- ID: `EXPL-BUG-015`
- Status: Resolved
- Priority: High automation reliability
- Reported: 2026-08-13 by `radioradio-ECR-2026-008`
- Confirmed: 2026-08-27T16:32:01-07:00
- Resolved: 2026-08-27T16:41:07-07:00
- Reporter: radioradio maintainers
- Owner: Codex
- Related work: Phase 16 Slice 16.1; `radioradio-ECR-2026-008` revision 1

## Symptom And Impact

The matching automation client rejects a legal server-produced completion
when a word-wrapped ListBox scrolls far enough that its logical current-item
index falls outside the viewport's visual-row interval. The accepted request
has already completed, so this validator defect turns a known completion into
an indeterminate client result even though the application remains healthy.

## Reproduction Or Evidence

On current `feature/table-improvements`, a three-item word-wrapped ListBox
named `workspace.chat.history` with the third item current produced logical
`CurrentIndex` 2, `VisualRowCount` 76, vertical offset 70, and viewport height
2. Its server projection was then rejected by the matching validator with:

`snapshot control "workspace.chat.history" (list_box) details are invalid for its kind`

RadioRadio uses the same word-wrapped ListBox configuration and exact
automation key for its live chat history.

## Root Cause

`ListBoxDetails.CurrentIndex` is a logical item index, while the vertical
viewport offset and height are measured in derived visual rows after wrapping.
The client compares those values as though they shared one coordinate system.
The server cannot provide a valid logical-to-visual mapping in the deliberately
compact details without exposing or duplicating the retained item model.

## Resolution

Matching-client validation now skips the logical-index/visual-row visibility
comparison only when the ListBox uses word or cell wrapping. Unwrapped lists
retain the stricter direct visibility implication because each logical item is
exactly one visual row. All kind, count, identity, digest, status, resource,
enabled-state, and viewport-geometry checks remain fail-closed.

The Collections, Go API, and Automation Protocol specifications now state the
two coordinate domains and the conditional validation boundary explicitly.

## Validation

- A shared server-projection/client-validation matrix covers empty,
  populated, selected, word- and cell-wrapped, scrolled, hidden, disabled,
  loading, and unwrapped-scrolled states with exact compact boundary evidence.
- Repeated live replacements prove retained, moved, removed, selected, hidden,
  and emptied current state remains matching-client valid.
- A real Unix-socket server/client test reproduces the RadioRadio coordinate
  split and accepts the initial, retained-current, moved-current,
  removed-current, and hidden completion snapshots.
- Malformed snapshot tests continue rejecting out-of-range current indices,
  invalid counts, keys, selection implications/digests, resource evidence,
  viewport geometry, and an unwrapped current item outside its viewport.
- The exact repaired worktree passed `make verify`, including vet, ordinary
  tests, PTY integration, the complete race suite, and debug, release, and
  profiling builds.

## History

- 2026-08-27T16:32:01-07:00 — Imported revision 1, reproduced the exact
  matching-client rejection on the current feature branch, and opened Phase
  16 Slice 16.1 for repair.
- 2026-08-27T16:41:07-07:00 — Focused and complete verification passed; the
  target bug and ECR disposition were closed as implemented and verified.
