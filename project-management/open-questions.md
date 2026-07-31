# Open Questions

Use this file for unresolved questions that need user input, later research,
or follow-up.

## Pending Questions

## Active Questions

- `EXPL-Q-002` — What Go module/package naming and public control-extension
  model lets most controls derive directly or indirectly from shared Panel
  behavior through idiomatic composition/embedding, without copying
  unsuitable inheritance mechanics from another language?
- `EXPL-Q-003` — Which backend and capability strategy should follow the
  deliberately narrow first Linux xterm/screen/tmux adapter, and what broader
  terminals, platforms, locales, code-page profiles, remote transports,
  curses/ncurses/terminfo integrations, and proven high-bit mappings will be
  supported?
- `EXPL-Q-004` — Which additions should follow the selected bounded Basic
  Automation JSON Lines protocol, including multiple controllers, post-v1 key
  identity, retained-history expansion, capture separation, and authenticated
  or capability-authorized transports?
- `EXPL-Q-005` — What structured command-routing model will unify menus,
  `Alt` accelerators, mnemonics, global and control-local hotkeys, editing
  modes, physical input, injected events, and configurable Ctrl-C/interrupt
  policy?
- `EXPL-Q-006` — After the selected Go 1.25 minimum, Go 1.26.5 normal
  toolchain, module path, Linux CGO-free product baseline, and three build
  modes, what additional supported targets, executables, CI matrices, pinned
  analysis/security tools, and release compatibility policy are required?
- `EXPL-Q-007` — What versioned catalog and state matrix defines “every UI
  element” for `expletives-test`, and how will completeness be checked as the
  public API grows?
- `EXPL-Q-011` — Which public mutations are synchronous thread-safe calls,
  which are asynchronous posts, what callback/reentrancy guarantees apply,
  and should the event loop, renderer, and presenter share one owner or use
  separately serialized owners?

## Resolved Questions

- 2026-07-30 — `EXPL-DEC-011` and `layout-api-v0.md` resolved
  `EXPL-Q-010`: unmanaged direct children may coexist with managed Layout
  trees, retain manual bounds, and paint after the attached Layout contexts
  in stable control insertion order. Managed Panels use Layout stack order.
- 2026-07-30 — `layout-api-v0.md` resolved `EXPL-Q-012` with the App-scoped
  `OverflowHandler`, one bounded handler slot, a 250 ms cancellation context,
  handled/default dispositions, observable episode states, black-on-yellow
  `[OK]`/`!` fallback, and `App.DismissOverflow`.
- 2026-07-24 — `EXPL-DEC-008` selected the first runnable implementation:
  module `github.com/Hard-Problems-Group-LLC/expletives`, Go 1.25 minimum and
  Go 1.26.5 normal toolchain, CGO-free Linux products, a deliberately narrow
  owned xterm/screen/tmux-family adapter, all-mode builds, and the bounded
  Unix-socket JSON Lines Basic Automation protocol. Broader follow-up remains
  in `EXPL-Q-003`, `EXPL-Q-004`, and `EXPL-Q-006`.
- 2026-07-24 — The initial project-owned Unicode policy pins
  `github.com/rivo/uniseg` `v0.4.7` for grapheme segmentation and display
  width, then accepts only complete clusters measured as exactly one cell.
  Dependency or policy changes require compatibility review. See
  `EXPL-DEC-008`.
- 2026-07-24 — The operator approved the minimal Basic `SnapshotV1` cell
  record: one canonical one-cell grapheme, semantic style, resolved foreground
  and background colors, and stable owner identity. Cursor state and the
  bounded typed control tree are snapshot-level data; Basic cells have no
  width or continuation fields. Versioned Unicode data remains in
  `EXPL-Q-001`. See `EXPL-DEC-007`.
- 2026-07-24 — The operator approved independently constructed Layouts
  attached atomically through `Panel.SetLayout`. See `EXPL-DEC-007`.
- 2026-07-24 — The operator directed that below-minimum Layouts preserve
  minima and resulting logical rectangles, constrain effective output by every
  ancestor clip and the application surface, and expose structured overflow.
  Applications may register an Overflow callback, with a sensible observable
  and dismissible fallback when no handler applies. Callback and fallback
  mechanics are constrained by `EXPL-DEC-007`; exact API and visual details
  were completed on 2026-07-30 in `layout-api-v0.md`.
- 2026-07-24 — The operator directed conservative basic-terminal Unicode
  degradation: definite 7-bit ASCII and confidently known upper-half code-page
  mappings render directly; other logical cells use a black-on-yellow
  single-character ASCII approximation, or a black-on-yellow `?` when none is
  reasonable. Unknown and remote code-page mappings are not guessed. See
  `EXPL-DEC-006`; backend/profile details stay in `EXPL-Q-003`.
- 2026-07-24 — The operator directed that complete composed grapheme clusters
  are supported when their final measured terminal width is exactly one cell.
  Every element the pinned policy measures as multi-cell, zero-width,
  indeterminate, or otherwise unsupported renders as exactly one
  `U+FFFD REPLACEMENT CHARACTER` (`�`) cell. A raw East Asian Width
  `Ambiguous` property alone does not reject a cluster measured as one cell.
  See `EXPL-DEC-005`. `EXPL-DEC-007` later fixed the minimal Basic snapshot
  fields; versioned Unicode-data details stay in `EXPL-Q-001`.
- 2026-07-24 — The operator directed that Basic Automation use
  `expletives-test --automation <socket-path>` and may provide raw
  `KeyDown`, `KeyUp`, and `KeyPress` lifecycle events before shortcut
  resolution, including modifier chords. Arbitrary terminal escape-byte
  injection is not implied. Remaining wire details stay in `EXPL-Q-004`.
- 2026-07-24 — The operator approved `app.Root()` as the sole root accessor,
  no package-global root, and container-capable parents for ordinary controls.
  Foundational parents remain immutable; reparenting moved to
  `EXPL-TASK-012`.
- 2026-07-24 — The operator approved `KeyPress` as a distinct one-shot raw
  logical event without persistent held state.
- 2026-07-24 — The operator approved neutral composition and update
  primitives rather than a mandatory MVC framework, while requiring the
  toolkit to fit multithreaded MVC, MVVC, and related applications and to be
  thread-safe at its documented public boundaries.
- 2026-07-24 — The operator directed the term Layout and initial instantiated
  `BoxLayout` and `GridLayout` objects that arrange direct child Panels
  relative to a parent Panel. The exact Go Panel-derivation model remains in
  `EXPL-Q-002`.
- 2026-07-24 — `expletivesctl` is a supported client/build target, and every
  executable target must build in debug, release, and profiling modes.
