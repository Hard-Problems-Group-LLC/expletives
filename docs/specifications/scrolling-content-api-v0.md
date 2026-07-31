# Scrolling And Content API v0

- Status: Directed pre-v1 contract
- Authority: Phase 15 roadmap and operator full-automatic direction,
  2026-07-31
- Scope: `ScrollablePanel`, `Viewport`, `MarkdownView`, `LogView`, and
  `StreamView`
- Depends on:
  [`navigation-chrome-api-v0.md`](navigation-chrome-api-v0.md),
  [`layout-api-v0.md`](layout-api-v0.md),
  [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md),
  [`application-architecture.md`](application-architecture.md), and
  [`automation-protocol-v1.md`](automation-protocol-v1.md)

## Design Boundary

Scrolling state is application-observable interaction state over a bounded
logical content surface. It is separate from physical-terminal state,
terminal escape bytes, application domain models, and Layout arrangement.
Every control in this phase uses the App's existing serialized mutation,
focus, raw-key, intended-frame, semantic-style, and typed-snapshot paths.

`ScrollablePanel` and `Viewport` are generic Panel-derived containers.
`MarkdownView`, `LogView`, and `StreamView` are specialized Panel-derived
leaves that reuse the same viewport math and keyboard contract without
exposing an inheritance framework or accepting arbitrary terminal output.

No control starts a goroutine, ticker, decoder worker, channel, or terminal
operation. Application workers submit bounded synchronous updates and retain
ownership of their scheduling, cancellation, retry, and backpressure policy.
This preserves multithreaded MVC/MVVC use without making an application model
inherit toolkit state.

## Shared Viewport State

```go
type ViewportState struct {
    ContentSize Size
    Offset      Point
}

type ScrollBarVisibility string

const (
    ScrollBarVisibilityAuto   ScrollBarVisibility = "auto"
    ScrollBarVisibilityAlways ScrollBarVisibility = "always"
    ScrollBarVisibilityNever  ScrollBarVisibility = "never"
)

type ScrollViewOptions struct {
    PanelOptions
    ContentAutomationKey string
    ContentStyle         StyleID
    State                ViewportState
    ArrowStep            Size
    PageStep             Size
    Disabled             bool
    DisabledReason       string
    ChangeCommand        CommandID
}

type ScrollablePanelOptions struct {
    ScrollViewOptions
    BorderStyle      StyleID
    BorderForm       BorderForm
    BorderForeground *Color
    BorderBackground *Color
    HorizontalBar    ScrollBarVisibility
    VerticalBar      ScrollBarVisibility
}
```

Content width, height, offsets, and steps are checked nonnegative cell counts.
ContentSize may be empty. An empty axis has offset zero. Offset is canonical
for the currently arranged viewport:

```text
MaximumOffset.X = max(0, ContentSize.Width  - ViewportSize.Width)
MaximumOffset.Y = max(0, ContentSize.Height - ViewportSize.Height)
```

Resize, chrome changes, Layout changes, state replacement, visibility repair,
and content replacement re-clamp offsets before publication. ArrowStep zero
defaults componentwise to one. PageStep zero defaults componentwise to the
current viewport axis, with a minimum of one. Explicit steps are positive and
bounded.

Visibility values default to Auto. Never reserves no bar cell. Always reserves
one row or column whenever the arranged base client has that axis. Auto uses a
bounded fixed-point calculation because one bar can make the opposite axis
overflow. When both bars are visible, their lower-right intersection is a
non-actionable corner cell. Tiny zero-, one-, and two-cell arrangements remain
bounded and may leave an empty content viewport.

ScrollBar glyphs, styles, track/thumb ratios, and exact integer rounding reuse
the Phase 14 ScrollBar contract. Integrated bars are decoration and typed
subrecords, not separately focusable child Controls. Applications that need a
separate focus stop use the public `ScrollBar` control.

## Viewport And ScrollablePanel

```go
type Viewport struct { /* copy-safe Panel-derived container */ }
type ScrollablePanel struct { /* copy-safe Panel-derived container */ }

func NewViewport(Container, ScrollViewOptions) (*Viewport, error)
func NewScrollablePanel(
    Container,
    ScrollablePanelOptions,
) (*ScrollablePanel, error)
func (t *Transaction) NewViewport(
    Container,
    ScrollViewOptions,
) (*Viewport, error)
func (t *Transaction) NewScrollablePanel(
    Container,
    ScrollablePanelOptions,
) (*ScrollablePanel, error)

func (v *Viewport) Content() *Panel
func (s *ScrollablePanel) Content() *Panel
func (v *Viewport) State() ViewportState
func (s *ScrollablePanel) State() ViewportState
func (v *Viewport) SetState(ViewportState) error
func (s *ScrollablePanel) SetState(ViewportState) error
func (v *Viewport) Update(context.Context, ViewportState) error
func (s *ScrollablePanel) Update(context.Context, ViewportState) error
func (v *Viewport) EnsureVisible(Rect) error
func (s *ScrollablePanel) EnsureVisible(Rect) error
func (v *Viewport) Focus() error
func (s *ScrollablePanel) Focus() error

func (t *Transaction) SetViewportState(Control, ViewportState) error
func (t *Transaction) EnsureViewportVisible(Control, Rect) error
```

Construction atomically creates one toolkit-managed direct child Panel,
returned by Content. Its automation key defaults to
`<owner-automation-key>.content`; an explicit ContentAutomationKey overrides
that derivation. ContentStyle defaults to the ordinary Panel style.

The managed Content Panel is the only direct child and may not be reparented,
destroyed independently, inserted into the owner's Layout, or replaced in v0.
Applications create controls under Content and attach its Layout normally,
including during the same construction Transaction. The scroll container
itself rejects Layout attachment because it owns the physical viewport,
border, and bar geometry.

Content Bounds are logical and derived:

```text
X      = ViewportBounds.X - Offset.X
Y      = ViewportBounds.Y - Offset.Y
Width  = ContentSize.Width
Height = ContentSize.Height
```

The Content Panel and descendants are clipped by ViewportBounds, so no
logical content paints into an optional border, integrated scrollbar,
application chrome, sibling, or ancestor clip. A Layout minimum larger than
ContentSize follows the existing structured Layout-overflow contract; it does
not silently change the application-owned content extent.

Viewport has no border or integrated bars. ScrollablePanel supports the
normal independent border forms and bar policies. Both are focusable when
enabled and at least one axis can move. Left/Right/Up/Down move by the matching
ArrowStep. PageUp/PageDown move vertically by PageStep. Home moves to `(0,0)`;
End moves to MaximumOffset. Movement clamps and never wraps.

Tab/Shift-Tab retain grouped focus traversal. When focus moves to a descendant
of Content, every scrollable ancestor minimally adjusts its offset to keep the
focused descendant visible. EnsureVisible applies the same minimal movement to
one nonempty logical Content rectangle. Programmatic state and ensure-visible
operations are silent. A user-originated offset change may route one optional
ChangeCommand after publication and outside toolkit locks.

`ControlDetails.Scrollable` contains the canonical state, maximum offset,
relative ViewportBounds, enabled/disabled policy, steps, change command,
Content ControlID/key, bar policies, bar visibility, and optional complete
horizontal/vertical `ScrollBarDetails` subrecords. Nested values are copied
and bounded.

## MarkdownView

MarkdownView is a focusable, read-only, vertically reflowing document view:

```go
type MarkdownViewOptions struct {
    ScrollablePanelOptions
    Markdown string
}

type MarkdownView struct { /* copy-safe Panel-derived leaf */ }

func NewMarkdownView(Container, MarkdownViewOptions) (*MarkdownView, error)
func (t *Transaction) NewMarkdownView(
    Container,
    MarkdownViewOptions,
) (*MarkdownView, error)
func (m *MarkdownView) Markdown() string
func (m *MarkdownView) SetMarkdown(string) error
func (m *MarkdownView) Update(context.Context, string) error
func (m *MarkdownView) Offset() Point
func (m *MarkdownView) SetOffset(Point) error
func (m *MarkdownView) Focus() error
func (t *Transaction) SetMarkdown(
    *MarkdownView,
    string,
) error
```

Markdown is copied, LF-normalized, valid UTF-8, one-cell canonical content.
The maximum source is `MaxContentBytes`; aggregate retained source across one
App is `MaxContentAggregateBytes`. The parser is deterministic and has bounded
line, block, inline-delimiter, and nesting work.

The supported v0 block subset is ATX headings, paragraphs, blank lines,
unordered and ordered lists without semantic nesting, blockquotes, fenced
code blocks, and horizontal rules. The inline subset is emphasis, strong
emphasis, code spans, and links. Unsupported HTML is rendered as ordinary
text; no script, image, URL fetch, file read, command, terminal sequence, or
plugin is executed. Link labels and destinations are visible text only in v0.

Ordinary prose word-wraps to viewport width. Fenced code preserves cells and
may scroll horizontally. Resize deterministically recomputes rendered rows and
clamps the offset. Semantic Theme roles distinguish headings, emphasis,
strong text, code, links, quotes, list markers, and rules without making color
the only cue.

`ControlDetails.Markdown` contains source byte/cell counts, block and rendered
row counts, scroll geometry, and bounded structural summaries. It does not
duplicate the complete off-screen source into automation.

## LogView

```go
type LogLevel string

const (
    LogDebug   LogLevel = "debug"
    LogInfo    LogLevel = "info"
    LogWarning LogLevel = "warning"
    LogError   LogLevel = "error"
)

type LogRecord struct {
    Key       string
    Timestamp string
    Level     LogLevel
    Text      string
}

type ContentCapacity struct {
    Records int
    Bytes   int
}

type LogViewOptions struct {
    ScrollablePanelOptions
    Capacity ContentCapacity
    Records  []LogRecord
    Follow   bool
}

type LogViewState struct {
    Follow          bool
    Offset          Point
    RetainedRecords int
    RetainedBytes   int
    DroppedRecords  uint64
    DroppedBytes    uint64
}

func NewLogView(Container, LogViewOptions) (*LogView, error)
func (t *Transaction) NewLogView(
    Container,
    LogViewOptions,
) (*LogView, error)
func (l *LogView) Append(context.Context, []LogRecord) error
func (l *LogView) Replace(context.Context, []LogRecord) error
func (l *LogView) Clear(context.Context) error
func (l *LogView) State() LogViewState
func (l *LogView) SetFollow(bool) error
func (l *LogView) SetOffset(Point) error
func (l *LogView) Focus() error
```

Records are copied and validated before one serialized publication. Keys are
stable bounded identifiers unique among retained records. Timestamp is
optional canonical one-row display text. Level is recognized. Text is
canonical bounded multiline content; raw terminal control traffic is never
interpreted.

Capacity defaults to a named bounded record/byte budget and may be reduced by
the caller. Appending evicts complete oldest records until both budgets hold.
DroppedRecords and DroppedBytes are monotonic until Clear or Replace and count
every eviction or input record that cannot be retained. No loss is silent.

Follow defaults true. While following, append and resize retain the tail.
User movement away from the tail disables Follow; End restores it. While
paused, append preserves the visible logical records as far as retained
capacity permits. Programmatic append/replace/clear/follow changes are silent;
user navigation may route ChangeCommand.

`ControlDetails.LogView` exposes capacity, retained and dropped counts, first
and last retained keys, follow state, and scroll geometry. It does not expose
off-screen log text.

## StreamView

StreamView is a bounded line-oriented byte-stream view, not a terminal
emulator:

```go
type StreamViewOptions struct {
    ScrollablePanelOptions
    Capacity ContentCapacity
    Follow   bool
}

type StreamAppendResult struct {
    InputBytes     int
    CompletedLines int
    DroppedLines   uint64
    DroppedBytes   uint64
}

type StreamView struct { /* copy-safe Panel-derived leaf */ }

func NewStreamView(Container, StreamViewOptions) (*StreamView, error)
func (t *Transaction) NewStreamView(
    Container,
    StreamViewOptions,
) (*StreamView, error)
func (s *StreamView) Append(
    context.Context,
    []byte,
) (StreamAppendResult, error)
func (s *StreamView) Flush(context.Context) error
func (s *StreamView) Clear(context.Context) error
func (s *StreamView) State() LogViewState
func (s *StreamView) SetFollow(bool) error
func (s *StreamView) SetOffset(Point) error
func (s *StreamView) Focus() error
```

Append copies input before returning and processes it synchronously under the
bounded mutation owner. CRLF is one line separator; bare CR and LF are line
separators. Invalid UTF-8 and every unsupported/control display element
become one `U+FFFD` cell. ANSI, DEC, xterm, OSC, DCS, CSI, cursor movement,
carriage-return rewriting, color, hyperlinks, and other terminal semantics
are never executed or emulated.

One bounded partial line is retained across chunks. Flush commits it as a
line. Lines beyond the configured per-line or ring capacity are truncated or
evicted with exact DroppedLines/DroppedBytes accounting and an explicit
visible truncation marker. AppendResult reports the effect of that call;
typed state reports cumulative loss.

StreamView uses the LogView follow/scrollback keyboard contract and
non-wrapping horizontal clipping. It owns no reader, file descriptor,
subprocess, goroutine, channel, or retry loop. A producer that needs an
asynchronous pipe reader owns that worker and calls Append with its own
cancellation and backpressure policy.

`ControlDetails.StreamView` exposes retained/pending byte and line counts,
cumulative drops, follow state, and scroll geometry, without duplicating the
complete off-screen stream.

## Concurrency, Privacy, And Mutation

All getters return independent copies. Public methods are safe for concurrent
use. Bounded append/update calls serialize through the App mutation owner,
observe caller cancellation while waiting, validate complete input before
publication, and publish at most one new snapshot. No application callback
runs while toolkit locks are held.

Content is potentially sensitive. Frames expose visible cells because that is
the existing observe contract; typed automation intentionally exposes counts,
positions, identifiers, and loss state rather than duplicating all retained
off-screen Markdown, log, or stream content. Applications remain responsible
for not displaying secrets in an attached-automation session.

## Acceptance

- Viewport/ScrollablePanel construction, managed-content ownership, Layout
  use, both-axis offsets, Auto/Always/Never fixed-point bars, exact thumb
  geometry, tiny/empty/large content, resize clamp, ensure-visible, focus
  keep-visible, user-only command routing, transactions, concurrency, and
  destruction;
- Markdown normalization, each supported block/inline form, unsupported HTML,
  deterministic wrap/code overflow, resize, semantic styles, bounded parser
  work, no I/O or execution, and typed evidence;
- Log ring capacity, ordered append/replace/clear, record validation, exact
  eviction/drop accounting, follow/scrollback, resize, concurrent producer
  calls, race detection, and bounded-memory benchmarks;
- Stream chunk and UTF-8 boundaries, CR/LF variants, partial flush, very long
  lines, invalid/control/escape input neutralization, exact drop accounting,
  follow/scrollback, and no hidden reader/worker;
- explicit root and automation detail projection, deep-copy, kind and
  relationship validation, and conservative response-bound proof;
- enabled catalog pages driven through raw keys and direct commands in
  self-check and attached automation; and
- full ordinary, PTY, race, debug/release/profiling, smoke, and live
  closed-loop verification.
