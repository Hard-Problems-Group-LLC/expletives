# expletives

`expletives` is a modernized, Turbo Vision-inspired text user interface (TUI)
toolkit for Go, designed around semantic cell frames, a common-controls
library, deterministic testing, and serious automation.

The supported secondary product, `expletives-test`, is a human-runnable
all-controls application and closed-loop diagnostic surface. Its attached
drive-and-observe interface is enabled only with
`--automation <socket-path>`. The supported `expletivesctl` command connects
to that explicitly enabled endpoint.

Core/Containers, Basic Presentation, Basic Automation, and Basic Layouts are
implemented. The public Go package now includes Box/Grid and nested Layouts,
independent arrangement and stacking order, Panel/Layout `Raise` and `Lower`,
structured overflow handling, optional root size/aspect constraints,
independent Frame/Layout border forms, immutable semantic snapshots, and the
CGO-free Linux terminal presenter. Application surfaces are bounded by
aggregate allocated cells rather than a conventional 240-column ceiling;
1200 by 1200 is a tested geometry. The `layouts.basic` `expletives-test`
fixture and `expletivesctl` provide the human and attached closed loop.

Run `make all` to build both commands in debug, release, and profiling modes,
or `make verify` for the complete local verification workflow. See
[Product Goals](docs/product-goals.md), the
[Specifications](docs/specifications/README.md), and the
[Panel of Experts review records](project-management/reviews/README.md).

Run the current interactive fixture with:

```sh
build/debug/expletives-test --automation /tmp/expletives.sock
```

With no root flags, the fixture expands to the complete terminal. For example,
to center a 16:9 root no larger than 160 by 90 cells inside a larger terminal:

```sh
build/debug/expletives-test \
  --root-max-width 160 --root-max-height 90 \
  --root-aspect-width 16 --root-aspect-height 9
```

The fixture includes independent single/double/shade borders on Frames and
Layouts, including an unbordered Frame whose child Layout supplies the one
clean outline.

The attached commands `layout.panel.raise`, `layout.panel.lower`,
`layout.layer.raise`, and `layout.layer.lower` exercise the new stack paths.
