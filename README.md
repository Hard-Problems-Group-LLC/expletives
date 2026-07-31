# expletives

`expletives` is a modernized, Turbo Vision-inspired text user interface (TUI)
toolkit for Go, designed around semantic cell frames, a common-controls
library, deterministic testing, and serious automation.

The supported secondary product, `expletives-test`, is a human-runnable
all-controls application and closed-loop diagnostic surface. Its attached
drive-and-observe interface is enabled only with
`--automation <socket-path>`. The supported `expletivesctl` command connects
to that explicitly enabled endpoint.

Core/Containers, Basic Presentation, Basic Automation, Basic Layouts,
Text/Display, Actions, Menus, Status Bar, Headers/Footers, Selection,
Text and Numeric Input, Progress, and Navigation/Chrome are implemented.
Scrolling and Content is implemented through Viewport, ScrollablePanel,
MarkdownView, LogView, and StreamView. Collections is the active phase;
ListBox, TreeView, DropDown, and editable ComboBox now implement
stable-identity bounded models, keyboard behavior, transient popup ownership,
and compact automation.
The public Go package includes non-container `Label`, `StaticText`,
`Separator`, `Rule`, `Button`, `HotkeyBar`, `MenuBar`, and `StatusBar`
controls; `Checkbox`, `RadioButton`/`RadioGroup`, `CycleField`, `SelectField`,
validated/password-safe `TextField`, ranged `NumberField`, and clamped
`SpinBox`, plus wrapped multiline `TextArea`; deterministic `ProgressBar`,
horizontal/vertical `Meter`, `Spinner`, and `ActivityDots`; focusable
horizontal/vertical `ScrollBar`; page-owning `TabbedPanel` and `Notebook`;
generic `Viewport` and framed `ScrollablePanel` containers; read-only
`MarkdownView`; structured bounded `LogView`; inert byte-oriented
`StreamView`; stable-key single/multiple-selection `ListBox`; selection-only
`DropDown`, editable validated `ComboBox`, and hierarchical `TreeView`;
root-owned
one-row `Header` and `Footer` containers; immutable
`Menu` models; Box/Grid and nested Layouts;
independent arrangement and stacking order; structured overflow; optional
root size/aspect constraints; independent Frame/Layout borders; immutable
semantic snapshots; and the CGO-free Linux terminal presenter.
Application surfaces are bounded by aggregate allocated cells rather than a
conventional 240-column ceiling; 1200 by 1200 is a tested geometry. The
`toolkit.catalog` `expletives-test` application exercises the delivered
controls and Layouts through purpose-specific menu screens, while
`expletivesctl` provides the attached closed loop.

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
clean outline. It also presents aligned `Label`, word-wrapped `StaticText`,
double-line `Separator`, titled `Rule`, Action and Menu controls, a persistent
StatusBar, one-row Header/Footer Layout examples, grouped Selection controls,
unrestricted/soft/hard/password TextField examples, and ranged/steppable
numeric input, plus multiline wrapping, selection, bounded paste, determinate
and indeterminate progress, terminal states, and reduced-motion activity.

`ScrollBar` copies bounded content, viewport, and offset state; derives exact
track/thumb geometry; and supports arrow, PageUp/PageDown, Home, and End
navigation with optional user-only change notification. `TabbedPanel` and
`Notebook` copy ordered `Tab` descriptors over direct child Panel pages,
separate tab focus from page selection, and derive visibility for exactly the
selected caller-visible page. The Controls/Navigation catalog screen exercises
both families with raw keys and typed attached-automation evidence. See
[`navigation-chrome-api-v0.md`](docs/specifications/navigation-chrome-api-v0.md).

The Controls/Scrolling / Content screen exercises bounded Markdown parsing,
structured log follow/scrollback, stream truncation and drop accounting,
grouped keyboard focus, and compact typed automation evidence. See
[`scrolling-content-api-v0.md`](docs/specifications/scrolling-content-api-v0.md).

The attached commands `layout.panel.raise`, `layout.panel.lower`,
`layout.layer.raise`, and `layout.layer.lower` exercise the new stack paths.
