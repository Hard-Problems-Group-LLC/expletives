# Bug: Catalog Theme Overrode The Dialog Contrast Variance

- ID: `EXPL-BUG-007`
- Status: Resolved
- Priority: High acceptance fidelity
- Reported: 2026-08-08T11:13:00-07:00
- Accepted: 2026-08-08T11:15:32-07:00
- Resolved: 2026-08-08T11:22:37-07:00
- Reporter: project operator
- Owner: Codex
- Related work: Phase 19 Slice 19.11; `EXPL-DEC-016`

## Symptom And Impact

The active `expletives-test` MessageBox continued to render black body text on
the medium-gray dialog surface after the library `DefaultTheme` changed that
role to white. The maintained public consumer therefore failed to demonstrate
the directed contrast variance even though the focused library test passed.

## Reproduction Or Evidence

The attached instance at `/home/mheck/expletives-test.sock` exposed a visible
`message_box` StaticText cell with foreground `#000000` and background
`#808080`. PID 1006905 began after the first rebuild, and
`/proc/1006905/exe` and the then-current `build/release/expletives-test` had
the same inode and SHA-256 digest, ruling out a stale process image at the
time of the report.

## Expected Behavior

The maintained catalog and default Theme both render dialog-family body and
static-message text white on `#808080`, as directed by `EXPL-DEC-016`.

## Actual Behavior

The library `DefaultTheme` rendered white, but the catalog rendered black.

## Root Cause

`internal/demo.New` constructs a complete custom Theme and duplicated every
dialog-family body role with `menuPopupStyle.Foreground`, which is black. The
first regression asserted the library Theme and a library-owned headless
MessageBox but did not exercise the required `expletives-test` public
consumer Theme.

## Resolution

The catalog now uses one white `dialogForeground` for every ModalPanel,
Dialog, standard-dialog, and file-picker body role while retaining separate
Button, border, and shadow roles. A public-consumer regression asserts the
complete custom Theme and an exact rendered catalog MessageBox body cell.

## Validation

The focused catalog regression passed. `make verify` then passed formatting,
vet, ordinary and Unix-socket tests, PTY integration, the complete race suite,
debug/release/profiling builds, catalog self-checks, and smoke. A separate
temporary attached instance launched from the rebuilt release returned frame
sequence 76 with the `dialog.message.message` cell styled `message_box`,
foreground `#FFFFFF`, and background `#808080`; it then published a final
shutdown snapshot and its disposable workspace was removed. The operator's
original active process was not terminated and, at that point, required a
restart to load the new executable image. The operator subsequently restarted
the maintained live instance; final ACP acceptance at frame 84 reconfirmed
`#FFFFFF` on `#808080`, and frame 85 closed the inspection dialog cleanly.

## History

- 2026-08-08T11:13:00-07:00 — Operator reported no visible effect and supplied
  the active attached instance.
- 2026-08-08T11:15:32-07:00 — Live snapshot and executable identity isolated
  the catalog custom-Theme override; Slice 19.11 reopened.
- 2026-08-08T11:22:37-07:00 — Focused, complete, and separate live attached
  verification passed; defect closed.
- 2026-08-08T19:14:21-07:00 — Operator restarted and accepted the maintained
  live instance; final attached inspection passed before ACP.
