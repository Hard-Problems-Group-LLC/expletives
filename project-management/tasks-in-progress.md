# Tasks In Progress

Track work that is actively underway. Keep this file small and current. Use
the same dated bullet format as the backlog. Add a start timestamp, current
owner, known blockers, and brief status notes.

- 2026-07-30 — `EXPL-TASK-019` — Deliver Menus and purpose-specific catalog
  screen navigation.
  - Requestor: project operator
  - Owner: Codex
  - Started: 2026-07-30T20:19:43-07:00
  - Status: fixing the Menu/MenuBar/MenuItem ownership, interaction, popup,
    focus-restoration, and screen-navigation contract.
  - Acceptance criteria:
    - deliver persistent `MenuBar`, popup `Menu`, and immutable `MenuItem`
      definitions over the shared Action command registry;
    - support Alt mnemonic access including Alt-F, F10 and a documented
      fallback, arrows/Home/End/Enter/Escape, disabled and separator skipping,
      nested popup stacking, clipping, resize, and exact focus restoration;
    - expose bounded typed menu/open-stack/selection state through core and
      automation snapshots;
    - switch `expletives-test` among purpose-specific Core/Layout,
      Text/Display, and Actions screens using normal menu commands;
    - prove Button, HotkeyBar, menu item, mnemonic, binding, and direct
      automation parity; and
    - pass ordinary, attached, PTY, race, fuzz, all-mode build, and smoke
      verification before ACP.
  - Blockers: none.
