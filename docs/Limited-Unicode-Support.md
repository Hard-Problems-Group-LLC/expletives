# Limited Unicode Support

Status: Directed requirement
Authority: Direct operator clarifications on 2026-07-24
Related decisions: `EXPL-DEC-005`, `EXPL-DEC-006`, and `EXPL-DEC-007` in
[`project-management/decision-log.md`](../project-management/decision-log.md)

## Scope

`expletives` supports displayed Unicode text only when each complete displayed
text element occupies exactly one terminal cell.

This is an intentional product boundary. Multi-cell rendering creates
substantial complexity and ambiguity in cursor movement, selection, editing,
clipping, borders, overlap, hit testing, Layout measurement, snapshots, and
terminal compatibility. The initial toolkit avoids that entire class of
problems.

## What “One Cell” Means

A terminal cell is one addressable column in one row of the intended frame.
Every canonical displayed text element:

- occupies exactly one cell;
- advances a horizontal cursor by exactly one cell;
- can be clipped or selected without reserving a neighboring cell; and
- is represented without a continuation cell.

A supported text element may be:

- one Unicode code point that displays in one cell; or
- one composed grapheme cluster containing several code points that together
  display in one cell, such as a base letter followed by combining marks.

A grapheme cluster is one user-perceived text element. The number of Unicode
code points in the cluster is not the limiting factor; its final terminal
width is.

## Unsupported Display Width

Any grapheme cluster that the project-owned width policy measures as anything
other than exactly one display cell is unsupported as rendered content.
Examples generally include:

- East Asian wide or full-width characters;
- emoji or joined emoji sequences that occupy two or more terminal columns;
- a standalone zero-width combining sequence; and
- any terminal-dependent sequence whose width cannot be classified safely as
  exactly one.

The renderer replaces each unsupported grapheme cluster with exactly one:

```text
� U+FFFD REPLACEMENT CHARACTER
```

`U+FFFD` is the conventional Unicode replacement character. The project width
policy must classify it as one cell, and the intended frame stores it as one
ordinary cell.

One unsupported grapheme cluster produces one replacement cell. The renderer
must never:

- emit half of a wide glyph;
- reserve or mutate a neighboring continuation cell;
- shift later columns by the original terminal width;
- partially paint the original cluster;
- let backend-specific width behavior alter logical geometry; or
- crash, loop, or corrupt a frame because unsupported text was supplied.

Application domain data may retain the original text. The replacement is the
view-layer representation used for measurement, rendering, cursor geometry,
selection, snapshots, and automation.

## Control Characters And Structural Text

Newline, tab, carriage return, escape, and other control characters are not
ordinary displayed grapheme clusters. Each control documents whether it
interprets, rejects, escapes, or replaces them. That policy must still resolve
to deterministic one-cell frame contents and must never replay terminal
control sequences.

Malformed UTF-8 is decoded through the same replacement-character boundary;
it cannot inject terminal bytes or create variable-width frame state.

## Cursor, Editing, And Selection

The one-cell rule is a foundational invariant:

- horizontal cursor movement advances by one displayed element and one cell;
- a supported composed grapheme remains indivisible during cursor movement,
  deletion, selection, and replacement;
- an unsupported grapheme is represented by one replacement cell and occupies
  one position in view geometry;
- selection, hit testing, and automation coordinates use cell indices without
  continuation-cell exceptions; and
- rows with different source text retain identical cursor-step semantics.

An editor may preserve the original unsupported grapheme in its application
value, subject to its documented validation policy, while displaying and
navigating its one-cell replacement representation.

## Layout, Rendering, And Snapshots

Layout measurement counts one cell for every supported or replaced displayed
element. Borders, labels, fields, menus, tables, and other controls do not
contain wide-cell special cases.

The canonical intended frame:

- has one independently addressable value per cell;
- has no width-two lead cells;
- has no continuation cells;
- never depends on a following cell to complete a displayed element; and
- exposes the same one-cell text representation to headless tests and attached
  automation.

The Basic `SnapshotV1` cell record stores that canonical one-cell grapheme,
semantic style, resolved foreground and background colors, and stable owner
identity. It has no width or continuation field. Cursor state and the bounded
typed control tree are snapshot-level facts.

A physical terminal adapter must preserve the intended representation's
one-cell geometry and present its canonical value when the terminal profile can
encode that value safely. When it cannot, the adapter uses the single-cell
physical fallback below. It must not reinterpret preserved application data and
attempt wide rendering behind the renderer's back.

## Basic Terminals And Code Pages

The canonical intended frame and the physical terminal repertoire are separate
contracts. A basic terminal adapter presents each logical cell conservatively:

- printable characters that map cleanly to 7-bit ASCII render directly;
- a character in the upper half of a declared 8-bit code page, or an equivalent
  local terminal repertoire, renders directly only when the adapter can
  determine its exact mapping and one-cell behavior with confidence;
- every other character is replaced physically by the closest reasonable
  single printable 7-bit ASCII character, rendered with black foreground on a
  yellow background; and
- when no reasonably close single-cell ASCII approximation exists, the adapter
  presents `?` with black foreground on a yellow background.

The approximation must remain one cell. It cannot expand one logical element
into several ASCII characters. The project owns and versions the approximation
table so that equivalent terminal profiles produce deterministic output.

“Definitely” is a strict requirement. The adapter must not infer an exact
high-bit mapping merely from `$TERM`, a process locale, a terminal family name,
or an unverified default code page. If the application cannot query or
otherwise establish the effective terminal and code-page configuration,
especially across a remote connection, it uses the highlighted ASCII
approximation instead of guessing.

Black-on-yellow is the required warning presentation when those colors are
available. A terminal that cannot express that color pair uses the documented
highest-visibility monochrome fallback and reports the presentation
degradation in backend diagnostics; it must not silently make the substituted
character indistinguishable from direct rendering.

This physical fallback does not mutate the canonical intended frame or
application data. For example, an unsupported multi-cell source element first
becomes one logical `U+FFFD` cell under the display-width rule. An ASCII-only
terminal then presents that logical cell as a black-on-yellow `?`. Headless
snapshots and attached automation continue to observe the canonical logical
cell; backend presentation evidence may additionally expose the physical
substitution.

### Toolkit Structural Glyphs

Known toolkit-owned border glyphs are a narrower capability projection, not
unknown application text. The intended frame retains canonical single-line,
double-line, or shade/block border characters. The terminal adapter selects
the best one-cell structural repertoire it knows the detected profile can
support:

- canonical Unicode line and shade glyphs in a UTF-8 locale;
- DEC Special Graphics line glyphs for a supported xterm-family terminal
  without UTF-8; or
- ASCII `+`, `-`, `|`, and `#` only when neither richer representation is
  safely available.

This structural substitution preserves the border's configured foreground,
background, and attributes; otherwise an explicit border-color override would
be lost merely because its shape used a terminal capability fallback. An
unrecognized Unicode value supplied as ordinary content still uses the
black-on-yellow approximation policy above.

## Width Policy

The project must own and version the Unicode segmentation and width policy
used to decide whether a grapheme cluster fits in one cell. Results cannot
depend silently on the developer workstation, locale, font, terminal emulator,
or an ambient C library version.

Support is based on the pinned project's deterministic measured width, not on
a raw Unicode East Asian Width property in isolation. A cluster whose
code-point data includes an `Ambiguous` property is still supported when the
pinned policy deterministically measures the complete cluster as exactly one
cell. A cluster measured as zero, two or more cells, or not classifiable with
the required determinism is unsupported and renders as `U+FFFD`.

For the first runnable baseline, the pinned policy is the project wrapper
around `github.com/rivo/uniseg` `v0.4.7`. Both precomposed one-cell text such
as `é` and an equivalent base-plus-combining-mark grapheme are accepted when
that policy measures the complete grapheme as one cell. A later change to the
dependency, width policy, or multi-cell support boundary requires an explicit
proposal, updated frame and cursor contracts, and new compatibility evidence.

## Verification

Ordinary Go, headless, attached-automation, and terminal-boundary tests must
cover:

- ASCII and ordinary one-cell Unicode;
- precomposed and decomposed one-cell accented text;
- one-cell combining grapheme editing and selection;
- wide and full-width input replaced by one `U+FFFD`;
- wide emoji and joined sequences replaced once per grapheme cluster;
- standalone zero-width sequences replaced safely;
- representative East Asian Width `Ambiguous` cases accepted when the pinned
  policy measures them as one cell, and replaced when it cannot establish
  exactly one cell;
- malformed UTF-8 and untrusted streamed text;
- clipping at the first and last column;
- cursor movement across rows containing replacements;
- borders, alignment, overlap, and Layout geometry after replacement;
- immutable snapshots with no continuation-cell state; and
- direct printable 7-bit ASCII presentation on a basic terminal;
- direct upper-half code-page presentation only under a known exact profile;
- conservative black-on-yellow single-character ASCII approximation for
  unknown, unavailable, and remote code-page mappings;
- black-on-yellow `?` when no reasonable single-cell approximation exists;
- Unicode, DEC Special Graphics, and ASCII projection of known toolkit border
  forms while preserving their configured style;
- a documented high-visibility fallback when the terminal cannot express
  black on yellow; and
- consistency between intended-frame, headless, automation, and terminal
  presentation paths.

Tests must assert that unsupported text never shifts a later cell, changes a
Panel rectangle, creates a partial glyph, or causes row-dependent cursor-step
behavior.

## Non-Goals

The initial toolkit does not support:

- two-column or wider glyphs;
- continuation-cell representations;
- terminals dynamically disagreeing about the width of canonical frame text;
- cursor movement by physical columns inside a multi-cell glyph; or
- preserving visual fidelity for unsupported wide text.

Supporting those behaviors later is possible only through an explicitly
approved compatibility and frame-model expansion. It is not an implicit
requirement of general UTF-8 input support.
