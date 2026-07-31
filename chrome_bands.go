package expletives

import (
	"context"
	"fmt"
)

// HeaderOptions configures one root-owned one-row Header container.
type HeaderOptions struct{ PanelOptions }

// FooterOptions configures one root-owned one-row Footer container.
type FooterOptions struct{ PanelOptions }

// Header is a copy-safe one-row top-edge Container.
type Header struct{ containerHandle }

// Footer is a copy-safe one-row bottom-edge Container.
type Footer struct{ containerHandle }

// NewHeader constructs and atomically inserts one Header.
func NewHeader(parent Container, options HeaderOptions) (*Header, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	header, err := tx.NewHeader(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return header, nil
}

// NewFooter constructs and atomically inserts one Footer.
func NewFooter(parent Container, options FooterOptions) (*Footer, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	footer, err := tx.NewFooter(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return footer, nil
}

// NewHeader records construction of one provisional Header.
func (t *Transaction) NewHeader(
	parent Container,
	options HeaderOptions,
) (*Header, error) {
	panel, err := t.newApplicationChromeBand(
		parent,
		options.PanelOptions,
		ControlHeader,
	)
	if err != nil {
		return nil, err
	}
	header := &Header{containerHandle: panel.containerHandle}
	panel.state.control = header
	panel.state.container = header
	return header, nil
}

// NewFooter records construction of one provisional Footer.
func (t *Transaction) NewFooter(
	parent Container,
	options FooterOptions,
) (*Footer, error) {
	panel, err := t.newApplicationChromeBand(
		parent,
		options.PanelOptions,
		ControlFooter,
	)
	if err != nil {
		return nil, err
	}
	footer := &Footer{containerHandle: panel.containerHandle}
	panel.state.control = footer
	panel.state.container = footer
	return footer, nil
}

func (t *Transaction) newApplicationChromeBand(
	parent Container,
	options PanelOptions,
	kind ControlKind,
) (*Panel, error) {
	if err := t.usable(); err != nil {
		return nil, err
	}
	if parent == nil || parent.containerState() == nil ||
		parent.containerState() != t.app.root.state {
		return nil, fmt.Errorf(
			"%w: %s must be parented directly by App.Root()",
			ErrInvalidParent,
			kind,
		)
	}
	if options.Bounds != (Rect{}) || options.MinimumSize != (Size{}) {
		return nil, fmt.Errorf(
			"%w: %s geometry is derived from the application surface",
			ErrInvalidGeometry,
			kind,
		)
	}
	panel, err := t.newControl(parent, options, kind, containerBehavior{})
	if err != nil {
		return nil, err
	}
	panel.state.minimumSize = Size{Height: 1}
	panel.state.autoMinimum = true
	return panel, nil
}

func (a *App) updateApplicationChromeBoundsLocked(size Size) bool {
	changed := false
	setBounds := func(state *controlState, bounds Rect) {
		if state.bounds != bounds {
			state.bounds = bounds
			changed = true
		}
	}
	menuBounds := menuBarSurfaceRect(size)
	statusBounds := statusBarSurfaceRect(size)
	for _, state := range a.root.state.children {
		if state.destroyed {
			continue
		}
		switch state.kind {
		case ControlMenuBar:
			setBounds(state, menuBounds)
		case ControlStatusBar:
			setBounds(state, statusBounds)
		case ControlHeader, ControlFooter:
			setBounds(state, Rect{})
		}
	}

	top, bottom := a.applicationChromeFixedEdgesLocked(size)
	for _, state := range a.root.state.children {
		if state.kind != ControlHeader ||
			state.destroyed ||
			!a.effectivelyVisibleLocked(state) {
			continue
		}
		if top < bottom {
			if size.Width > 0 {
				setBounds(state, Rect{
					Y: top, Width: size.Width, Height: 1,
				})
			}
			top++
		}
	}
	for _, state := range a.root.state.children {
		if state.kind != ControlFooter ||
			state.destroyed ||
			!a.effectivelyVisibleLocked(state) {
			continue
		}
		if top < bottom {
			bottom--
			if size.Width > 0 {
				setBounds(state, Rect{
					Y: bottom, Width: size.Width, Height: 1,
				})
			}
		}
	}
	return changed
}

func (a *App) applicationChromeFixedEdgesLocked(size Size) (int, int) {
	top, bottom := 0, size.Height
	if a.firstMenuBarLocked() != nil && bottom > 0 {
		top = 1
	}
	if a.firstStatusBarLocked() != nil && bottom > top {
		bottom--
	}
	return top, bottom
}

func (a *App) applicationChromeContentRectLocked() Rect {
	top, bottom := a.applicationChromeFixedEdgesLocked(a.size)
	for _, state := range a.root.state.children {
		if state.kind == ControlHeader &&
			!state.destroyed &&
			a.effectivelyVisibleLocked(state) &&
			top < bottom {
			top++
		}
	}
	for _, state := range a.root.state.children {
		if state.kind == ControlFooter &&
			!state.destroyed &&
			a.effectivelyVisibleLocked(state) &&
			top < bottom {
			bottom--
		}
	}
	return Rect{
		Y: top, Width: a.size.Width, Height: max(0, bottom-top),
	}
}
