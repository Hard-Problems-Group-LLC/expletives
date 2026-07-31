# Scrolling And Content

The Phase 15 controls share one bounded, thread-safe scrolling model while
serving different kinds of content:

- `Viewport` clips one toolkit-managed Content Panel without a frame or
  integrated scrollbars;
- `ScrollablePanel` adds optional borders and independently configured
  horizontal and vertical scrollbars; and
- `MarkdownView` is a read-only leaf that parses and reflows a deliberately
  small Markdown subset;
- `LogView` retains bounded structured records with severity styling; and
- `StreamView` turns bounded arbitrary byte chunks into inert display lines,
  never a terminal emulator.

The formal contract is
[`specifications/scrolling-content-api-v0.md`](specifications/scrolling-content-api-v0.md).

## Construct A MarkdownView

```go
view, err := expletives.NewMarkdownView(
    app.Root(),
    expletives.MarkdownViewOptions{
        ScrollablePanelOptions: expletives.ScrollablePanelOptions{
            ScrollViewOptions: expletives.ScrollViewOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "help.overview",
                    Bounds: expletives.Rect{
                        X: 2, Y: 2, Width: 72, Height: 20,
                    },
                },
                ChangeCommand: "help.scrolled",
            },
            BorderForm:    expletives.BorderSingle,
            HorizontalBar: expletives.ScrollBarVisibilityAuto,
            VerticalBar:   expletives.ScrollBarVisibilityAuto,
        },
        Markdown: "# Overview\n\nUse **arrows** or `PageDown` to scroll.",
    },
)
if err != nil {
    return err
}
```

The embedded scroll options provide the ordinary Panel identity, style,
visibility, geometry, Layout hints, focus policy, and user-change command.
`ContentAutomationKey`, `ContentStyle`, and `State.ContentSize` are reserved:
MarkdownView derives its private rendered content and rejects callers that
try to supply those values. An initial nonnegative `State.Offset` is allowed
and is clamped after the first layout.

## Supported Markdown

MarkdownView recognizes:

- ATX headings (`#` through `######`), paragraphs, and blank lines;
- ordered and unordered lists without semantic nesting;
- blockquotes, fenced code blocks, and horizontal rules; and
- emphasis, strong emphasis, code spans, and links.

Prose wraps to the viewport. Fenced code does not wrap and can create
horizontal overflow. Links render as `label (destination)` and are not
interactive. HTML and unsupported Markdown syntax remain literal text.
MarkdownView never fetches a URL, reads a file, runs a command, interprets a
terminal sequence, loads a plugin, or performs another source-driven I/O
operation.

Source is normalized to LF-separated, valid one-cell text under
[`Limited-Unicode-Support.md`](Limited-Unicode-Support.md). Unsupported
multi-cell display elements become one `U+FFFD` cell. One view retains at
most `MaxContentBytes`; all Markdown, log, and stream content in one App is
jointly bounded by `MaxContentAggregateBytes`.

## Update, Focus, And Scroll

`SetMarkdown` replaces the complete source synchronously. A worker that needs
cancellation while waiting for the App mutation owner can use `Update`:

```go
if err := view.Update(ctx, newDocument); err != nil {
    return err
}
```

All input is validated and parsed before atomic publication. Public methods
are safe for concurrent callers, and getters return copied values. Use one
Transaction to publish a source and offset change together:

```go
transaction := app.NewTransaction()
if err := transaction.SetMarkdown(view, newDocument); err != nil {
    return err
}
if err := transaction.SetMarkdownOffset(view, expletives.Point{}); err != nil {
    return err
}
if err := transaction.Commit(ctx); err != nil {
    return err
}
```

After `Focus`, arrow keys scroll one configured step, Page Up/Page Down use
the configured or viewport-derived page step, Home moves to the origin, and
End moves to the maximum offset. Movement clamps and never wraps. A
programmatic source or offset change is silent; an actual user-key movement
routes `ChangeCommand`, when configured, after toolkit locks are released.

## Theme And Automation

The ordinary `markdown_view` style paints prose. The semantic style roles
`markdown.heading`, `markdown.emphasis`, `markdown.strong`, `markdown.code`,
`markdown.link`, `markdown.quote`, `markdown.list_marker`, and
`markdown.rule` can be overridden in the App Theme.

Root snapshots expose typed `ControlDetails.Markdown`: source and rendered
counts, derived scroll geometry, and at most four block summaries. Truncated
details preserve the first two and last two summaries. Attached automation
projects the same meaning through a smaller content-view viewport record and
never duplicates the retained off-screen source.

## Use LogView From A Model Or Controller

```go
logs, err := expletives.NewLogView(
    panel,
    expletives.LogViewOptions{
        ScrollablePanelOptions: expletives.ScrollablePanelOptions{
            ScrollViewOptions: expletives.ScrollViewOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "jobs.log",
                },
                ChangeCommand: "jobs.log.scrolled",
            },
            BorderForm:    expletives.BorderSingle,
            HorizontalBar: expletives.ScrollBarVisibilityAuto,
            VerticalBar:   expletives.ScrollBarVisibilityAuto,
        },
        Capacity: expletives.ContentCapacity{Records: 2_000, Bytes: 512 << 10},
        Follow:   true,
    },
)
if err != nil {
    return err
}

err = logs.Append(ctx, []expletives.LogRecord{{
    Key:       "job.0042.started",
    Timestamp: "14:32:09",
    Level:     expletives.LogInfo,
    Text:      "worker accepted the job",
}})
```

Keys must be unique among records that remain retained. Log text may contain
normalized newlines, but rows do not wrap; horizontal scrolling keeps record
geometry stable. Severity uses `log.debug`, `log.info`, `log.warning`, and
`log.error`; timestamps use `log.timestamp`; and a visible loss summary uses
`content.dropped`.

`Follow` is an explicit bool. Leave it false to start in scrollback, or set it
true to follow the tail. Page Up, Home, or another movement away from the tail
pauses follow; End resumes it. While paused, appends preserve the visible
logical record when that record remains inside the bounded ring.

Use `Replace` for a fresh model snapshot and `Clear` to empty the view. Both
reset cumulative drop counters. When several controls must change in one
frame, use `Transaction.AppendLog`, `ReplaceLog`, `ClearLog`, `SetLogFollow`,
and `SetLogOffset` and commit once. A Transaction builder itself has one
caller; concurrent producers should call the public context-aware methods or
send domain updates through an application-owned controller.

## Display An Inert Byte Stream

```go
stream, err := expletives.NewStreamView(
    panel,
    expletives.StreamViewOptions{
        ScrollablePanelOptions: expletives.ScrollablePanelOptions{
            ScrollViewOptions: expletives.ScrollViewOptions{
                PanelOptions: expletives.PanelOptions{
                    AutomationKey: "build.output",
                },
            },
            BorderForm:    expletives.BorderSingle,
            HorizontalBar: expletives.ScrollBarVisibilityAuto,
            VerticalBar:   expletives.ScrollBarVisibilityAuto,
        },
        Capacity: expletives.ContentCapacity{Records: 4_096, Bytes: 1 << 20},
        Follow:   true,
    },
)
if err != nil {
    return err
}

result, err := stream.Append(ctx, chunk)
if err != nil {
    return err
}
_ = result // per-call input, completed-line, and loss counts
```

CRLF, bare CR, and LF terminate lines even when separators cross Append calls.
An unfinished line remains pending until a separator arrives or `Flush`
commits it. Invalid UTF-8, control bytes, terminal escape traffic, and
unsupported multi-cell elements render as inert replacement cells. Nothing
in the stream can move the cursor, change terminal color, open a link, start a
process, or perform I/O. Overlong content shows `[truncated]` (or the smallest
marker that fits), and `stream.truncated` plus `content.dropped` provide
semantic warning styles.

StreamView does not read from a pipe and does not start a goroutine. An MVC or
MVVC application owns its reader/worker and calls `Append` with its own
cancellation and backpressure policy. Complete lines and the current partial
line share the configured byte budget; `State` and typed automation expose
retained, pending, and cumulative-loss counts without copying off-screen
content.

## Catalog And Automation

The human and automated catalog scenario is Controls/Scrolling / Content.
Its stable keys are `content.markdown`, `content.log-follow`,
`content.log-scrollback`, and `content.stream-drops`; its direct screen command
is `catalog.controls.scrolling`. `content.append` adds deterministic log and
stream fixtures, `content.follow` toggles scrollback following, and real user
movement reports `content.changed`. Tab moves among the four separate focus
groups. Attached automation deliberately exposes bounded metrics and
geometry, while visible frame cells remain available for look-and-feel
inspection.
