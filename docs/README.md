# Project Documentation

Keep durable project documentation here.

- [Product Goals](product-goals.md) defines the primary and secondary
  products and the project's directed outcomes.
- [Limited Unicode Support](Limited-Unicode-Support.md) defines the strict
  one-cell text boundary, `U+FFFD` replacement policy, and conservative
  black-on-yellow ASCII degradation for basic terminals.
- [Terminal Shortcut Compatibility](Terminal-Shortcut-Compatibility.md)
  identifies host-emulator collisions on supported RHEL, Fedora, and Ubuntu
  terminal families and defines the project's safe default-binding policy.
- [ControlDetails Extension Checklist](Control-Details-Extension-Checklist.md)
  keeps core snapshots, automation projection, validation, resource proofs,
  and tests synchronized as new typed controls arrive.
- [Specifications](specifications/README.md) contains behavior, interface,
  data, build, verification, and operational contracts.
- [Go API v0](specifications/go-api-v0.md) records the implemented public
  Core/Presentation, Layout, semantic Theme, Transaction, and command
  contract.
- [Automation Protocol v1](specifications/automation-protocol-v1.md) records
  the implemented local JSON Lines drive-and-observe contract and its
  automation-owned snapshot projection.
- [Consuming-Application Architecture](specifications/application-architecture.md)
  requires support for multithreaded MVC, MVVC, and similar
  model/view/update separations.
- [Concurrency and Thread Safety](specifications/concurrency-and-thread-safety.md)
  defines safe public calls and serialized UI/render/presentation ownership.
- [Control Catalog](specifications/control-catalog.md) defines the common
  controls, delivery phases, and explicitly deferred Structured Input family.
- [Status Bar API v0](specifications/status-bar-api-v0.md) defines root-owned
  bottom-row context and command hints with deterministic narrow behavior.
- [Layouts and Overflow](specifications/layouts-and-overflow.md) defines atomic
  Layout attachment, below-minimum clipping, structured overflow, application
  notification, and deterministic fallback behavior.
- [Foundational Layout API v0](specifications/layout-api-v0.md) fixes the
  implemented Box/Grid, nesting, stacking, snapshot, and Overflow Go surface.
- [`research/`](research/README.md) records non-normative lessons and source
  reviews that inform later proposals and decisions.
- [Panel of Experts reviews](../project-management/reviews/README.md) record
  exceptional major-design reviews and their evidence.

Project-management state belongs under `project-management/`, not here.
Reusable framework guidance belongs in the FieldManual submodule.
