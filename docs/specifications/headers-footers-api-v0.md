# Headers And Footers API v0

- Status: Implemented v0 contract
- Authority: Direct operator instruction on 2026-07-30
- Scope: ordered one-row root chrome containers
- Depends on:
  [`application-chrome-v0.md`](application-chrome-v0.md) and
  [`layout-api-v0.md`](layout-api-v0.md)

## Public API

```go
type HeaderOptions struct { PanelOptions }
type FooterOptions struct { PanelOptions }

type Header struct { /* copy-safe Container handle */ }
type Footer struct { /* copy-safe Container handle */ }

func NewHeader(Container, HeaderOptions) (*Header, error)
func NewFooter(Container, FooterOptions) (*Footer, error)
func (t *Transaction) NewHeader(
    Container,
    HeaderOptions,
) (*Header, error)
func (t *Transaction) NewFooter(
    Container,
    FooterOptions,
) (*Footer, error)
```

Header and Footer embed ordinary Control and Container operations. Their
parent must be exactly `app.Root()`. Caller Bounds and MinimumSize must be
zero; their derived minimum is `(0, 1)`, and later Bounds/MinimumSize setters
reject the operation. They may not themselves be Layout items.

## Ordering And Geometry

Every effectively visible Header and Footer occupies exactly one complete
physical row:

- Headers begin below the visible Main Menu and retain construction order
  from top to bottom.
- Footers end above the visible StatusBar. The oldest Footer is nearest the
  StatusBar; every subsequently constructed Footer is one row higher, so the
  most recent is highest.
- Root constraints never move or narrow either kind.
- Hiding or destroying a band removes it from ordering and returns its row in
  the same atomic publication. Showing it restores construction order.
- Hidden or unallocated bands have empty derived Bounds.

After the Main Menu and distinct StatusBar rows are reserved, Headers consume
remaining rows from the top in construction order. Footers then consume
remaining rows from the bottom in construction order. Excess bands receive
empty Bounds. Thus tiny-surface priority is Main Menu, distinct StatusBar,
Headers, then Footers. At height one, the existing Main Menu-over-StatusBar
paint priority remains unchanged.

## One-Row Layout Compatibility

Header and Footer are ordinary Containers inside their one-row physical
Bounds. They may own horizontal BoxLayout, one-row GridLayout, and nested
Layout trees whose complete measured minimum height is at most one.

Every top-level Layout attachment is validated atomically after all nested
items, child minima, border decoration, insets, and gaps are measured.
Attachment fails with `ErrInvalidLayout` when the resulting minimum height
exceeds one. This rejects multirow Grid content, vertical stacks, top/bottom
insets that require space, and every height-consuming Layout border. Width
overflow retains the ordinary structured Layout overflow behavior.

Multiple compatible top-level Layouts use the existing common-mode stacking
contract. Header/Footer do not introduce another arrangement model.

An attached Layout whose band is hidden retains deterministic arrangement
state but has no actionable overflow episode. Showing the band recalculates
its width against the restored row and starts a new episode if it remains
undersized. A band that is semantically visible but receives no row under the
tiny-surface policy follows the same suppression rule; the separate
minimum-usable-application-geometry policy owns that whole-surface condition.

## Rendering, Snapshot, And Automation

The default semantic styles are `header` and `footer`. Each band fills its
entire allocated row, then paints its ordinary child/Layout tree clipped to
that row. Kind, root parent, children, Bounds, AbsoluteBounds, EffectiveClip,
minimum, style, resolved style, visibility, container detail, child controls,
Layout state, and overflow are exposed by existing typed snapshots and the
automation projection. No unrestricted detail bag is added.

## Acceptance Criteria

- Complete combined chrome follows Main Menu, Headers, root content, Footers,
  StatusBar order at every nonnegative surface size.
- Header construction order and most-recent-highest Footer order are stable
  through hide/show, destroy, resize, and constrained roots.
- Compatible horizontal Box and one-row Grid Layouts arrange children in the
  band; incompatible decorated or multirow trees fail without attachment or
  publication.
- Tiny surfaces allocate deterministically with nonnegative in-surface
  Bounds.
- `expletives-test` provides a Controls/Headers / Footers page and live
  horizontal Box/Grid content.
- Normal Go, socket automation, PTY integration, race, debug, release, and
  profiling verification pass.
