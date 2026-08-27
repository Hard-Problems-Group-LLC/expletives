# Imported ECR: Make Server ListBox Snapshots Client-Valid

- Source ID: `radioradio-ECR-2026-008`
- Revision: 1
- Target: Expletives
- Received: 2026-08-27 by direct operator instruction
- Source-file SHA-256: `afdc8359e2893edbbe57787b1c1da7fb393af950a25c106ff201477097cc6c85`
- Sensitivity: reusable automation consistency defect; no traffic content

## Received Request

The matching Expletives automation client rejected a server-produced
completion containing RadioRadio's live `workspace.chat.history` ListBox as
invalid for its kind. A bounded raw-protocol reader accepted the same
completion, and the application remained healthy, isolating the failure to
client validation rather than transport or command execution.

Acceptance requires every legal empty, populated, current, selected, wrapped,
scrolled, hidden, and live-replacement ListBox projection to remain valid to
the matching client. Projection and validation must retain strict shared
bounds without exposing item text or bypassing validation.
