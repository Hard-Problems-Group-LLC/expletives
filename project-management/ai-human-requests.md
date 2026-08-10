# AI Human Requests

Use this queue for non-blocking actions that AI agents need humans to perform.

Record each request with concise context, owner or requestor details, and ISO
8601 timestamps.

## Pending Requests

No pending requests.

## Active Requests

No active requests.

## Completed Requests

- 2026-08-10 — `EXPL-REQ-004` — Complete the remaining Phase 19 physical
  terminal rows.
  - Requestor: Codex
  - Owner: project operator
  - Created: 2026-08-02T03:15:00-07:00
  - Completed: 2026-08-10T14:21:00-07:00
  - Outcome: current release `a402cf076358` passed the complete native XTerm,
    tmux, GNU Screen, conservative-locale, and OpenSSH loopback PTY rows,
    including exact-window visuals, terminal-native paste, resize, job
    control, post-resume input, Ctrl-C status 130, and terminal restoration.
  - Prerequisite and boundary: GNU Screen requires `altscreen on`; loopback
    proves the actual OpenSSH TCP/PTY path but not real-network latency or
    disconnect behavior.
  - Safety/privacy: temporary profiles and credentials were isolated under the
    ignored project-local workspace; durable records contain no secrets,
    endpoints, usernames, unrelated captures, or private environment IDs.
  - Evidence:
    [`Terminal Compatibility Verification`](../docs/Terminal-Compatibility-Verification.md).

- 2026-07-24 — `EXPL-REQ-003` — Decide the minimal Basic snapshot,
  Layout-attachment seam, and below-minimum behavior.
  - Requestor: Codex
  - Owner: project operator
  - Created: 2026-07-24T08:59:36-07:00
  - Completed: 2026-07-24T10:44:56-07:00
  - Outcome:
    - approved a Basic `SnapshotV1` cell record containing one one-cell
      grapheme, semantic style, resolved foreground/background colors, and
      stable owner identity, with cursor and bounded typed tree at snapshot
      level and no cell width or continuation fields;
    - approved independent Layout construction followed by atomic
      `Panel.SetLayout` attachment; and
    - directed below-minimum preservation of logical rectangles, effective
      intersection of every ancestor clip and the application surface, and
      observable overflow, with an optional application Overflow callback and
      a sensible fallback such as a dismissible `OK` warning.
  - Safety boundary: callback delivery is queued after layout and outside
    Layout locks, dispatched through bounded cancellation/deadline-aware
    workers, coalesced per overflow episode, observable and dismissible without
    hanging automation, and protected against tiny-terminal recursion. Exact
    handler, dispatcher bounds, and default-visual details remain design work.
  - Related:
    - [`EXPL-DEC-007`](decision-log.md#expl-dec-007--fix-basic-snapshot-layout-attachment-and-overflow-semantics)
    - [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
    - [`project-management/open-questions.md`](open-questions.md)

- 2026-07-24 — `EXPL-REQ-002` — Decide the canonical text unit after the
  grapheme explanation.
  - Requestor: Codex
  - Owner: project operator
  - Created: 2026-07-24T08:40:22-07:00
  - Completed: 2026-07-24
  - Outcome: complete composed grapheme clusters are supported when their
    final measured terminal width is exactly one cell. Every unsupported
    multi-cell, zero-width, indeterminate, or otherwise unrenderable display
    element becomes exactly one `U+FFFD REPLACEMENT CHARACTER` (`�`) cell.
    “Indeterminate” is evaluated by the pinned project policy; a raw East
    Asian Width `Ambiguous` property alone is not a categorical rejection.
  - Related:
    - [`EXPL-DEC-005`](decision-log.md#expl-dec-005--limit-displayed-unicode-to-one-cell)
    - [`docs/Limited-Unicode-Support.md`](../docs/Limited-Unicode-Support.md)
    - [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
    - [`project-management/open-questions.md`](open-questions.md)
- 2026-07-24 — `EXPL-REQ-001` — Review the foundational window,
  presentation, automation, Layout, and consuming-application proposal.
  - Requestor: Codex
  - Owner: project operator
  - Created: 2026-07-24T07:07:26-07:00
  - Completed: 2026-07-24T08:40:22-07:00
  - Outcome:
    - approved `app.Root()`, no global root, container-capable parents, and
      foundational immutable ownership;
    - moved reparenting to the backlog;
    - approved distinct one-shot `KeyPress`;
    - required thread-safe multithreaded MVC/MVVC compatibility;
    - selected Panel-only instantiated `BoxLayout` and `GridLayout`
      terminology and scope; and
    - made `expletivesctl` and every executable part of all three build modes.
  - Follow-up:
    - `EXPL-REQ-002` resolved the displayed Unicode unit; and
    - `EXPL-REQ-003` resolved the minimal Basic snapshot, Layout attachment,
      and below-minimum behavior.
  - Related:
    - [`EXPL-DEC-004`](decision-log.md#expl-dec-004--root-input-concurrency-layout-and-build-foundations)
    - [`EXPL-PROP-001`](proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)
