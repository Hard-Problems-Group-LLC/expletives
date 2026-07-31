# Application Chrome v0

- Status: Directed pre-v1 contract
- Authority: Direct operator instruction on 2026-07-30
- Scope: physical edge rows reserved for the Main Menu, Headers, Footers, and
  Status Bar

## Purpose

Application chrome is anchored to the physical terminal client area, not to a
centered, maximum-sized, aspect-constrained, or otherwise inset application
root rectangle. It establishes a predictable edge frame around the ordinary
root content and prevents catalog controls from treating global chrome as
ordinary Layout items.

Turbo Vision is the reference for Menu appearance and keyboard behavior only.
This contract does not import, copy, or require any Turbo Vision
implementation.

## Canonical Row Order

When present, chrome consumes physical rows in this order:

1. the Main Menu occupies row 0, from column 0 through the last column;
2. zero or more Header rows occupy consecutive rows immediately below the
   Main Menu, or begin at row 0 when no Main Menu exists;
3. ordinary root content receives the remaining middle rectangle;
4. zero or more Footer rows occupy consecutive rows immediately above the
   Status Bar, with the most recently added Footer highest in the Footer
   stack; and
5. the Status Bar occupies the last physical row, from column 0 through the
   last column.

If the surface is too short for every requested row, chrome retains stable
semantic identity and uses a deterministic priority and clipping policy
defined by the implementing phase. Geometry is always nonnegative and never
materializes cells outside the physical surface.

The Main Menu and Status Bar are each optional and unique per App. Headers
and Footers are optional ordered collections. Every individual Header and
Footer is exactly one row high.

Main Menu root items may form start- and end-aligned groups within row 0.
Both remain part of the same MenuBar and keyboard sequence. This supports the
conventional right-justified Help menu without creating another chrome
control or changing row ownership.

## Ownership And Geometry

Every chrome control is constructed with `app.Root()` as its immutable
control parent, preserving the project-wide parent requirement. Chrome is not
an ordinary child of the root content rectangle:

- its physical Bounds are derived from the App surface;
- it may not be inserted into an ordinary Layout;
- callers may not set its Bounds or declared minimum directly;
- physical resize updates its derived Bounds atomically; and
- its snapshot Bounds and AbsoluteBounds use App-surface coordinates.

The root content rectangle is the intersection of the constrained root
rectangle and the physical surface remaining after visible chrome rows are
reserved. A centered root that does not overlap an edge row loses no
additional space. A root that reaches an occupied edge begins or ends beside
that chrome. Hiding or destroying chrome returns its row to ordinary content
in the same atomic layout publication.

## One-Row Header And Footer Layouts

Header and Footer controls are one-row containers. They may own only Layout
trees whose complete measured and decorated minimum height fits one row.
Expected compatible cases include a horizontal `BoxLayout` and a one-row
`GridLayout` with no height-consuming border. Attachment rejects, atomically:

- a Layout whose measured minimum height exceeds one;
- vertical gaps, top/bottom insets, or decoration that require more than one
  row;
- a Grid requiring more than one row; and
- any nested Layout tree whose combined one-row constraint cannot hold.

This validation concerns height. Ordinary width overflow remains governed by
the Layout overflow contract.

## Delivery Sequence

- Menus deliver the Main Menu and the initial reusable root-content
  reservation seam.
- Status Bar is the immediately following phase.
- Headers and Footers follow Status Bar.

Later chrome phases extend the same physical-edge calculation rather than
introducing a second coordinate model.

## Acceptance Criteria

- At every nonempty width, a visible Main Menu owns the first and last cells
  of physical row 0.
- A constrained root never moves or narrows the Main Menu or Status Bar.
- Root Layouts begin below visible top chrome and end above visible bottom
  chrome.
- Resize, hide, show, and destroy publish chrome and root-content geometry
  atomically.
- Header/Footer Layout validation rejects incompatible trees without partial
  attachment.
- Snapshots and attached automation expose exact chrome Bounds, ownership,
  style, visibility, and content displacement.
