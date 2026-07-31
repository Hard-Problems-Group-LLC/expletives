# Review Finding: EXPL-REV-001-D-S04-F003

- Finding ID: EXPL-REV-001-D-S04-F003
- Reviewer identity: `/root/design_customizability`
- Seat and lens: Seat 4 — Customizability: styling, behavior policy, application-defined controls, command and event policy, terminal variation, and safe override or composition mechanisms
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Command customization is split across an opaque append-only chord map, a replace-all handler, and a separate automation inventory, with no shared inspectable policy or scoped override.
- Consequence: An application cannot unbind, replace, compose by scope, or generate help; automation can advertise commands inconsistently; and future menus, edit modes, and interrupt handling must layer around or replace these primitives.
- Evidence:
  - `input.go:101-137`
  - `docs/specifications/go-api-v0.md:484-518`
  - `docs/specifications/automation-protocol-v1.md:871-894`
  - `docs/specifications/automation-protocol-v1.md:321-324`
  - `docs/specifications/automation-protocol-v1.md:560-577`
  - `docs/research/ui-toolkit-lessons.md:40-50`
  - `project-management/open-questions.md:23-26`
- Recommendation: Mark the current facilities as fixture primitives. Before Actions, navigation, modals, or interrupt policy, define an inspectable command and binding registry with scope and precedence, replacement and unbind operations, enabled state, signal and interrupt adaptation, and help and automation projections.
- Related lenses: extensibility, orthogonality, client ease of use, testability, reliability
- Disposition: confirmed
- Disposition rationale: The frozen API has append-only opaque bindings, one
  replace-all handler, and a separately supplied automation inventory, with no
  inspectable shared scope, replacement, or unbind policy.
- Resolution: pending
- Resolution evidence or operator record: pending
