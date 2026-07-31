# ControlDetails Extension Checklist

Use this checklist when a new control adds a typed `ControlDetails` member.
It keeps the local snapshot and untrusted automation projection synchronized
without weakening explicit kind validation.

## Core State

- Add the control kind, public detail types, and union pointer.
- Populate details from the same measured/rendered model used for painting.
- Deep-copy every slice, pointer, and nested modifier collection in
  `cloneSnapshot`.
- Test semantic state, exact geometry, and mutation of returned copies.

## Automation Projection

- Add the wire detail types and union pointer.
- Project every field explicitly from core types.
- Deep-copy every wire slice, pointer, and nested modifier collection.
- Add projection and clone-alias tests.

## Untrusted Client Validation

- Require the detail member only for its exact control kind and reject it for
  every other kind.
- Validate identifiers, canonical text, enums, chords, state implications,
  geometry, ordering, uniqueness, and empty/nonempty invariants.
- Enforce per-control and aggregate resource limits with checked arithmetic.
- Add one valid fixture plus focused invalid fixtures for each implication.

## Response And Integration Bounds

- Add the largest single nested item to the maximum-response proof.
- If the control is unique, account for its one-control overhead separately
  rather than multiplying it by every control slot.
- Recalculate the retained-response aggregate and update the automation
  specification.
- Exercise the member through `expletives-test`, socket automation, and a
  concise live projection using its stable automation key.

## Cross-Control Registries

- If the new kind is focusable, add accurate generic FocusGuideBar text,
  register the kind in automation focus-target validation, and exercise the
  resulting contextual Footer through attached automation.
- Audit every private exhaustive kind switch that supplies focus,
  container/border behavior, geometry, rendering, input dispatch, destruction
  repair, and snapshot validation. Add a regression test at each affected
  cross-control boundary.

Reflection is not a substitute for these explicit trust-boundary checks.
Shared copy helpers are appropriate only when they preserve the same bounded
typed contract.
