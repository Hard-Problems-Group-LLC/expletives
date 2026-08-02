package expletives

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const (
	// MaxLayouts bounds active Layout objects in one App.
	MaxLayouts = 1024
	// MaxLayoutItems bounds the aggregate number of attached Layout items.
	MaxLayoutItems = MaxControls
	// MaxLayoutDepth bounds a top-level Layout tree, including its root.
	MaxLayoutDepth = 32
	// MaxOverflowHandlers documents the single replaceable App callback slot.
	MaxOverflowHandlers = 1
)

// LayoutID is an App-scoped runtime Layout identity.
type LayoutID string

// LayoutKind identifies one built-in arrangement algorithm.
type LayoutKind string

const (
	LayoutBox  LayoutKind = "box"
	LayoutGrid LayoutKind = "grid"
)

// Orientation selects the BoxLayout main axis.
type Orientation uint8

const (
	Horizontal Orientation = iota
	Vertical
)

// Alignment selects placement within an item's cross-axis or Grid cell.
// AlignDefault consults the item's control-supplied LayoutHints.
type Alignment uint8

const (
	AlignDefault Alignment = iota
	AlignStretch
	AlignStart
	AlignCenter
	AlignEnd
)

// LayoutSizeHint tells a Layout whether an axis normally consumes only its
// natural minimum or may expand into available space. The empty value lets
// the control kind choose its sensible default.
type LayoutSizeHint string

const (
	LayoutSizeDefault LayoutSizeHint = ""
	LayoutSizeNatural LayoutSizeHint = "natural"
	LayoutSizeStretch LayoutSizeHint = "stretch"
)

// LayoutHints are one control's resolved per-axis sizing preferences.
type LayoutHints struct {
	Horizontal       LayoutSizeHint `json:"horizontal"`
	Vertical         LayoutSizeHint `json:"vertical"`
	HorizontalWeight int            `json:"horizontal_weight"`
	VerticalWeight   int            `json:"vertical_weight"`
}

func resolveControlLayoutHints(
	kind ControlKind,
	behavior controlBehavior,
	requested LayoutHints,
) (LayoutHints, error) {
	defaults := defaultControlLayoutHints(kind, behavior)
	resolved := requested
	for _, axis := range []*LayoutSizeHint{
		&resolved.Horizontal,
		&resolved.Vertical,
	} {
		switch *axis {
		case LayoutSizeDefault:
		case LayoutSizeNatural, LayoutSizeStretch:
		default:
			return LayoutHints{}, fmt.Errorf(
				"%w: invalid control layout size hint",
				ErrInvalidLayout,
			)
		}
	}
	if resolved.Horizontal == LayoutSizeDefault {
		resolved.Horizontal = defaults.Horizontal
	}
	if resolved.Vertical == LayoutSizeDefault {
		resolved.Vertical = defaults.Vertical
	}
	if resolved.HorizontalWeight < 0 ||
		resolved.VerticalWeight < 0 ||
		resolved.HorizontalWeight > maxCoordinateMagnitude ||
		resolved.VerticalWeight > maxCoordinateMagnitude {
		return LayoutHints{}, fmt.Errorf(
			"%w: invalid control layout weight",
			ErrInvalidLayout,
		)
	}
	resolved.HorizontalWeight = resolveLayoutAxisWeight(
		resolved.Horizontal,
		resolved.HorizontalWeight,
		defaults.HorizontalWeight,
	)
	resolved.VerticalWeight = resolveLayoutAxisWeight(
		resolved.Vertical,
		resolved.VerticalWeight,
		defaults.VerticalWeight,
	)
	if (resolved.Horizontal == LayoutSizeNatural &&
		requested.HorizontalWeight > 0) ||
		(resolved.Vertical == LayoutSizeNatural &&
			requested.VerticalWeight > 0) {
		return LayoutHints{}, fmt.Errorf(
			"%w: natural layout axis cannot carry growth weight",
			ErrInvalidLayout,
		)
	}
	return resolved, nil
}

func resolveLayoutAxisWeight(
	hint LayoutSizeHint,
	requested int,
	defaultWeight int,
) int {
	if hint == LayoutSizeNatural {
		return 0
	}
	if requested > 0 {
		return requested
	}
	if defaultWeight > 0 {
		return defaultWeight
	}
	return 1
}

func defaultControlLayoutHints(
	kind ControlKind,
	behavior controlBehavior,
) LayoutHints {
	hints := func(horizontal, vertical LayoutSizeHint) LayoutHints {
		result := LayoutHints{Horizontal: horizontal, Vertical: vertical}
		if horizontal == LayoutSizeStretch {
			result.HorizontalWeight = 1
		}
		if vertical == LayoutSizeStretch {
			result.VerticalWeight = 1
		}
		return result
	}
	switch kind {
	case ControlRoot, ControlPanel, ControlFrame, ControlGroupBox,
		ControlModalPanel, ControlDialog, ControlMessageBox, ControlConfirmDialog,
		ControlInputDialog, ControlProgressDialog,
		ControlRadioGroup, ControlStaticText, ControlTextArea,
		ControlTabbedPanel, ControlNotebook, ControlViewport,
		ControlScrollablePanel, ControlMarkdownView:
		return hints(LayoutSizeStretch, LayoutSizeStretch)
	case ControlLogView, ControlStreamView, ControlListBox, ControlTreeView,
		ControlTable, ControlDataGrid:
		return hints(LayoutSizeStretch, LayoutSizeStretch)
	case ControlMenuBar, ControlStatusBar, ControlHeader, ControlFooter,
		ControlHotkeyBar, ControlFocusGuideBar, ControlTextField,
		ControlNumberField, ControlSpinBox, ControlProgressBar,
		ControlDropDown, ControlComboBox:
		return hints(LayoutSizeStretch, LayoutSizeNatural)
	case ControlSeparator, ControlRule:
		if divider, ok := behavior.(dividerBehavior); ok &&
			divider.orientation == Vertical {
			return hints(LayoutSizeNatural, LayoutSizeStretch)
		}
		return hints(LayoutSizeStretch, LayoutSizeNatural)
	case ControlMeter:
		if progress, ok := behavior.(progressBehavior); ok &&
			progress.meter != nil &&
			progress.meter.Orientation == Vertical {
			return hints(LayoutSizeNatural, LayoutSizeStretch)
		}
		return hints(LayoutSizeStretch, LayoutSizeNatural)
	case ControlScrollBar:
		if scrollBar, ok := behavior.(scrollBarBehavior); ok &&
			scrollBar.orientation == Vertical {
			return hints(LayoutSizeNatural, LayoutSizeStretch)
		}
		return hints(LayoutSizeStretch, LayoutSizeNatural)
	default:
		return hints(LayoutSizeNatural, LayoutSizeNatural)
	}
}

// Insets reserves cells at four edges.
type Insets struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// LayoutItemOptions configures one Panel or nested Layout item.
type LayoutItemOptions struct {
	Insets          Insets
	Grow            int
	HorizontalAlign Alignment
	VerticalAlign   Alignment
}

// BoxLayoutOptions configures one linear Layout.
type BoxLayoutOptions struct {
	AutomationKey string
	Gap           int
	Insets        Insets
	Border        BorderOptions
}

// GridLayoutOptions configures one row-major, uniform-cell Layout. At least
// one dimension must be positive; a zero dimension is derived from item count.
type GridLayoutOptions struct {
	AutomationKey string
	Rows          int
	Columns       int
	HorizontalGap int
	VerticalGap   int
	Insets        Insets
	Border        BorderOptions
}

// Layout is a copy-safe arrangement and stacking object with optional border
// decoration. It is not a Control and cannot receive focus or input.
type Layout interface {
	ID() LayoutID
	AutomationKey() string
	Kind() LayoutKind
	Bounds() Rect
	MinimumSize() Size
	Owner() Container
	ParentLayout() Layout
	Raise() error
	Lower() error

	layoutState() *layoutState
}

// BoxLayout arranges items linearly on one axis.
type BoxLayout struct{ state *layoutState }

// GridLayout arranges items in row-major uniform cells.
type GridLayout struct{ state *layoutState }

type layoutItemKind uint8

const (
	layoutPanelItem layoutItemKind = iota + 1
	layoutLayoutItem
)

type layoutItem struct {
	kind    layoutItemKind
	panel   *controlState
	layout  *layoutState
	options LayoutItemOptions
	bounds  Rect
	minimum Size
}

type layoutState struct {
	handle        Layout
	id            LayoutID
	automationKey string
	kind          LayoutKind
	orientation   Orientation
	box           BoxLayoutOptions
	grid          GridLayoutOptions
	border        borderBehavior
	items         []*layoutItem
	stack         []*layoutItem
	app           *App
	owner         *controlState
	parent        *layoutState
	bounds        Rect
	ownerBounds   Rect
	minimum       Size
	rootIndex     int
	attached      bool
	destroyed     bool
}

// Detached Layout builders are independent of Apps, so this narrow lock
// protects their small structural trees until atomic attachment transfers
// ownership to an App's existing mutation lock.
var layoutBuilderMu sync.Mutex

// NewBoxLayout constructs a detached linear Layout.
func NewBoxLayout(
	orientation Orientation,
	options BoxLayoutOptions,
) (*BoxLayout, error) {
	if orientation != Horizontal && orientation != Vertical {
		return nil, fmt.Errorf("%w: invalid Box orientation", ErrInvalidLayout)
	}
	if err := validateLayoutKeyAndGeometry(
		options.AutomationKey,
		options.Gap,
		options.Insets,
	); err != nil {
		return nil, err
	}
	border, err := newLayoutBorder(options.Border)
	if err != nil {
		return nil, err
	}
	state := &layoutState{
		automationKey: options.AutomationKey,
		kind:          LayoutBox,
		orientation:   orientation,
		box:           options,
		border:        border,
	}
	layout := &BoxLayout{state: state}
	state.handle = layout
	return layout, nil
}

// NewGridLayout constructs a detached uniform-cell Grid.
func NewGridLayout(options GridLayoutOptions) (*GridLayout, error) {
	if options.Rows < 0 || options.Columns < 0 ||
		(options.Rows == 0 && options.Columns == 0) {
		return nil, fmt.Errorf("%w: Grid requires a positive dimension", ErrInvalidLayout)
	}
	if options.Rows > MaxLayoutItems || options.Columns > MaxLayoutItems {
		return nil, ErrLayoutCapacity
	}
	if options.Rows > 0 && options.Columns > 0 &&
		options.Rows > MaxLayoutItems/options.Columns {
		return nil, ErrLayoutCapacity
	}
	if err := validateLayoutKeyAndGeometry(
		options.AutomationKey,
		options.HorizontalGap,
		options.Insets,
	); err != nil {
		return nil, err
	}
	if options.VerticalGap < 0 ||
		options.VerticalGap > maxCoordinateMagnitude {
		return nil, fmt.Errorf("%w: negative Grid gap", ErrInvalidLayout)
	}
	border, err := newLayoutBorder(options.Border)
	if err != nil {
		return nil, err
	}
	state := &layoutState{
		automationKey: options.AutomationKey,
		kind:          LayoutGrid,
		grid:          options,
		border:        border,
	}
	layout := &GridLayout{state: state}
	state.handle = layout
	return layout, nil
}

func newLayoutBorder(options BorderOptions) (borderBehavior, error) {
	form, err := normalizeBorderForm(options.Form, BorderNone)
	if err != nil {
		return borderBehavior{}, err
	}
	style, err := normalizeStyleID(options.Style, "layout.border")
	if err != nil {
		return borderBehavior{}, err
	}
	return borderBehavior{
		borderStyle:        style,
		form:               form,
		foregroundOverride: cloneColor(options.Foreground),
		backgroundOverride: cloneColor(options.Background),
	}, nil
}

func validateLayoutKeyAndGeometry(key string, gap int, insets Insets) error {
	if key != "" && !validBoundedIdentifier(key) {
		return fmt.Errorf("%w: invalid automation key", ErrInvalidLayout)
	}
	if gap < 0 || gap > maxCoordinateMagnitude || !insets.valid() {
		return fmt.Errorf("%w: negative gap or inset", ErrInvalidLayout)
	}
	return nil
}

func (i Insets) valid() bool {
	return i.Top >= 0 && i.Right >= 0 && i.Bottom >= 0 && i.Left >= 0 &&
		i.Top <= maxCoordinateMagnitude &&
		i.Right <= maxCoordinateMagnitude &&
		i.Bottom <= maxCoordinateMagnitude &&
		i.Left <= maxCoordinateMagnitude
}

func validateLayoutItemOptions(options LayoutItemOptions) error {
	if options.Grow < 0 || options.Grow > maxCoordinateMagnitude ||
		!options.Insets.valid() ||
		options.HorizontalAlign > AlignEnd ||
		options.VerticalAlign > AlignEnd {
		return fmt.Errorf("%w: invalid item options", ErrInvalidLayout)
	}
	return nil
}

func (b *BoxLayout) layoutState() *layoutState {
	if b == nil {
		return nil
	}
	return b.state
}

func (g *GridLayout) layoutState() *layoutState {
	if g == nil {
		return nil
	}
	return g.state
}

func (b *BoxLayout) ID() LayoutID           { return layoutID(b.layoutState()) }
func (g *GridLayout) ID() LayoutID          { return layoutID(g.layoutState()) }
func (b *BoxLayout) AutomationKey() string  { return layoutKey(b.layoutState()) }
func (g *GridLayout) AutomationKey() string { return layoutKey(g.layoutState()) }
func (b *BoxLayout) Kind() LayoutKind       { return layoutKind(b.layoutState()) }
func (g *GridLayout) Kind() LayoutKind      { return layoutKind(g.layoutState()) }
func (b *BoxLayout) Bounds() Rect           { return layoutBounds(b.layoutState()) }
func (g *GridLayout) Bounds() Rect          { return layoutBounds(g.layoutState()) }
func (b *BoxLayout) MinimumSize() Size      { return layoutMinimum(b.layoutState()) }
func (g *GridLayout) MinimumSize() Size     { return layoutMinimum(g.layoutState()) }
func (b *BoxLayout) Owner() Container       { return layoutOwner(b.layoutState()) }
func (g *GridLayout) Owner() Container      { return layoutOwner(g.layoutState()) }
func (b *BoxLayout) ParentLayout() Layout   { return layoutParent(b.layoutState()) }
func (g *GridLayout) ParentLayout() Layout  { return layoutParent(g.layoutState()) }
func (b *BoxLayout) Raise() error           { return raiseLayout(b.layoutState(), true) }
func (g *GridLayout) Raise() error          { return raiseLayout(g.layoutState(), true) }
func (b *BoxLayout) Lower() error           { return raiseLayout(b.layoutState(), false) }
func (g *GridLayout) Lower() error          { return raiseLayout(g.layoutState(), false) }
func (b *BoxLayout) AddPanel(c Control, o LayoutItemOptions) error {
	return addLayoutPanel(b.layoutState(), c, o)
}
func (g *GridLayout) AddPanel(c Control, o LayoutItemOptions) error {
	return addLayoutPanel(g.layoutState(), c, o)
}
func (b *BoxLayout) AddLayout(l Layout, o LayoutItemOptions) error {
	return addNestedLayout(b.layoutState(), l, o)
}
func (g *GridLayout) AddLayout(l Layout, o LayoutItemOptions) error {
	return addNestedLayout(g.layoutState(), l, o)
}

func layoutID(state *layoutState) LayoutID {
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	if state == nil {
		return ""
	}
	return state.id
}

func layoutKey(state *layoutState) string {
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	if state == nil {
		return ""
	}
	return state.automationKey
}

func layoutKind(state *layoutState) LayoutKind {
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	if state == nil {
		return ""
	}
	return state.kind
}

func layoutBounds(state *layoutState) Rect {
	layoutBuilderMu.Lock()
	if state == nil {
		layoutBuilderMu.Unlock()
		return Rect{}
	}
	app := state.app
	if app == nil {
		layoutBuilderMu.Unlock()
		return Rect{}
	}
	layoutBuilderMu.Unlock()
	app.mu.RLock()
	defer app.mu.RUnlock()
	return state.bounds
}

func layoutMinimum(state *layoutState) Size {
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	if state == nil {
		return Size{}
	}
	if state.app != nil {
		state.app.mu.RLock()
		defer state.app.mu.RUnlock()
		return measureLayoutLocked(state)
	}
	return measureDetachedLayoutLocked(state)
}

func measureDetachedLayoutLocked(state *layoutState) Size {
	for _, item := range state.items {
		if item.kind == layoutPanelItem {
			if item.panel == nil || item.panel.app == nil {
				item.minimum = Size{}
				continue
			}
			item.panel.app.mu.RLock()
			item.minimum = item.panel.minimumSize
			item.panel.app.mu.RUnlock()
		} else {
			item.minimum = measureDetachedLayoutLocked(item.layout)
		}
	}
	if state.kind == LayoutGrid {
		state.minimum = measureGridLocked(state)
	} else {
		state.minimum = measureBoxLocked(state)
	}
	state.minimum = addLayoutBorderMinimum(state, state.minimum)
	return state.minimum
}

func layoutOwner(state *layoutState) Container {
	layoutBuilderMu.Lock()
	if state == nil {
		layoutBuilderMu.Unlock()
		return nil
	}
	app := state.app
	if app == nil {
		layoutBuilderMu.Unlock()
		return nil
	}
	layoutBuilderMu.Unlock()
	app.mu.RLock()
	defer app.mu.RUnlock()
	if state.owner == nil || state.destroyed {
		return nil
	}
	return state.owner.container
}

func layoutParent(state *layoutState) Layout {
	layoutBuilderMu.Lock()
	if state == nil {
		layoutBuilderMu.Unlock()
		return nil
	}
	app := state.app
	if app == nil {
		defer layoutBuilderMu.Unlock()
		if state.parent == nil {
			return nil
		}
		return state.parent.handle
	}
	layoutBuilderMu.Unlock()
	app.mu.RLock()
	defer app.mu.RUnlock()
	if state.parent == nil || state.destroyed {
		return nil
	}
	return state.parent.handle
}

func addLayoutPanel(
	state *layoutState,
	control Control,
	options LayoutItemOptions,
) error {
	if state == nil || control == nil || control.controlState() == nil {
		return ErrInvalidLayout
	}
	if err := validateLayoutItemOptions(options); err != nil {
		return err
	}
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	attached, destroyed := layoutLifecycleLocked(state)
	if destroyed {
		return ErrDestroyed
	}
	if attached {
		return ErrLayoutAttached
	}
	if len(state.items) >= MaxLayoutItems {
		return ErrLayoutCapacity
	}
	panel := control.controlState()
	for _, item := range state.items {
		if item.kind == layoutPanelItem && item.panel == panel {
			return fmt.Errorf("%w: duplicate Panel item", ErrInvalidLayout)
		}
	}
	item := &layoutItem{kind: layoutPanelItem, panel: panel, options: options}
	state.items = append(state.items, item)
	state.stack = append(state.stack, item)
	if err := validateGridCapacityLocked(state); err != nil {
		state.items = state.items[:len(state.items)-1]
		state.stack = state.stack[:len(state.stack)-1]
		return err
	}
	return nil
}

func addNestedLayout(
	state *layoutState,
	layout Layout,
	options LayoutItemOptions,
) error {
	if state == nil || layout == nil || layout.layoutState() == nil {
		return ErrInvalidLayout
	}
	if err := validateLayoutItemOptions(options); err != nil {
		return err
	}
	child := layout.layoutState()
	layoutBuilderMu.Lock()
	defer layoutBuilderMu.Unlock()
	stateAttached, stateDestroyed := layoutLifecycleLocked(state)
	childAttached, childDestroyed := layoutLifecycleLocked(child)
	if stateDestroyed || childDestroyed {
		return ErrDestroyed
	}
	if state == child || stateAttached || childAttached || child.parent != nil {
		return fmt.Errorf("%w: unusable nested Layout", ErrInvalidLayout)
	}
	if layoutContainsLocked(child, state) {
		return fmt.Errorf("%w: Layout cycle", ErrInvalidLayout)
	}
	if layoutAncestorDepthLocked(state)+layoutDepthLocked(child) > MaxLayoutDepth {
		return ErrLayoutCapacity
	}
	if len(state.items) >= MaxLayoutItems {
		return ErrLayoutCapacity
	}
	item := &layoutItem{kind: layoutLayoutItem, layout: child, options: options}
	state.items = append(state.items, item)
	state.stack = append(state.stack, item)
	child.parent = state
	if err := validateGridCapacityLocked(state); err != nil {
		child.parent = nil
		state.items = state.items[:len(state.items)-1]
		state.stack = state.stack[:len(state.stack)-1]
		return err
	}
	return nil
}

func layoutLifecycleLocked(state *layoutState) (attached, destroyed bool) {
	if state == nil {
		return false, false
	}
	if state.app == nil {
		return state.attached, state.destroyed
	}
	state.app.mu.RLock()
	attached, destroyed = state.attached, state.destroyed
	state.app.mu.RUnlock()
	return attached, destroyed
}

func layoutAncestorDepthLocked(state *layoutState) int {
	depth := 1
	for state.parent != nil {
		depth++
		state = state.parent
	}
	return depth
}

func validateGridCapacityLocked(state *layoutState) error {
	if state.kind != LayoutGrid ||
		state.grid.Rows == 0 || state.grid.Columns == 0 {
		return nil
	}
	if len(state.items) > state.grid.Rows*state.grid.Columns {
		return ErrLayoutCapacity
	}
	return nil
}

func layoutContainsLocked(root, target *layoutState) bool {
	if root == target {
		return true
	}
	for _, item := range root.items {
		if item.kind == layoutLayoutItem &&
			layoutContainsLocked(item.layout, target) {
			return true
		}
	}
	return false
}

func layoutDepthLocked(state *layoutState) int {
	depth := 1
	for _, item := range state.items {
		if item.kind == layoutLayoutItem {
			depth = max(depth, 1+layoutDepthLocked(item.layout))
		}
	}
	return depth
}

// SetLayout atomically attaches the first top-level Layout to this container.
func (p *containerHandle) SetLayout(layout Layout) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	tx := state.app.NewTransaction()
	if err := tx.SetLayout(p, layout); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// AddLayout atomically attaches an additional top-level stacking context.
func (p *containerHandle) AddLayout(layout Layout) error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	tx := state.app.NewTransaction()
	if err := tx.AddLayout(p, layout); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Raise moves this Panel to the highest Panel stack slot in its Layout.
func (p *controlHandle) Raise() error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	tx := state.app.NewTransaction()
	if err := tx.Raise(p); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// Lower moves this Panel to the lowest Panel stack slot in its Layout.
func (p *controlHandle) Lower() error {
	state, err := p.mutableState()
	if err != nil {
		return err
	}
	tx := state.app.NewTransaction()
	if err := tx.Lower(p); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

func raiseLayout(state *layoutState, raise bool) error {
	layoutBuilderMu.Lock()
	if state == nil {
		layoutBuilderMu.Unlock()
		return ErrInvalidLayout
	}
	app := state.app
	if app == nil {
		layoutBuilderMu.Unlock()
		return ErrInvalidLayout
	}
	handle := state.handle
	layoutBuilderMu.Unlock()
	app.mu.RLock()
	destroyed, attached := state.destroyed, state.attached
	app.mu.RUnlock()
	if destroyed {
		return ErrDestroyed
	}
	if !attached {
		return ErrInvalidLayout
	}
	tx := app.NewTransaction()
	var err error
	if raise {
		err = tx.RaiseLayout(handle)
	} else {
		err = tx.LowerLayout(handle)
	}
	if err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

func measureLayoutLocked(state *layoutState) Size {
	for _, item := range state.items {
		if item.kind == layoutPanelItem {
			item.minimum = item.panel.minimumSize
		} else {
			item.minimum = measureLayoutLocked(item.layout)
		}
	}
	if state.kind == LayoutGrid {
		state.minimum = measureGridLocked(state)
	} else {
		state.minimum = measureBoxLocked(state)
	}
	state.minimum = addLayoutBorderMinimum(state, state.minimum)
	return state.minimum
}

func addLayoutBorderMinimum(state *layoutState, size Size) Size {
	if state.border.form == BorderNone {
		return size
	}
	return Size{
		Width:  checkedLayoutSum(size.Width, 2),
		Height: checkedLayoutSum(size.Height, 2),
	}
}

func measureBoxLocked(state *layoutState) Size {
	main, cross := 0, 0
	for index, item := range state.items {
		if index > 0 {
			main = checkedLayoutSum(main, state.box.Gap)
		}
		outerW := checkedLayoutSum(
			item.minimum.Width,
			item.options.Insets.Left,
			item.options.Insets.Right,
		)
		outerH := checkedLayoutSum(
			item.minimum.Height,
			item.options.Insets.Top,
			item.options.Insets.Bottom,
		)
		if state.orientation == Horizontal {
			main = checkedLayoutSum(main, outerW)
			cross = max(cross, outerH)
		} else {
			main = checkedLayoutSum(main, outerH)
			cross = max(cross, outerW)
		}
	}
	if state.orientation == Horizontal {
		return Size{
			Width: checkedLayoutSum(
				main, state.box.Insets.Left, state.box.Insets.Right,
			),
			Height: checkedLayoutSum(
				cross, state.box.Insets.Top, state.box.Insets.Bottom,
			),
		}
	}
	return Size{
		Width: checkedLayoutSum(
			cross, state.box.Insets.Left, state.box.Insets.Right,
		),
		Height: checkedLayoutSum(
			main, state.box.Insets.Top, state.box.Insets.Bottom,
		),
	}
}

func gridDimensionsLocked(state *layoutState) (rows, columns int) {
	rows, columns = state.grid.Rows, state.grid.Columns
	count := len(state.items)
	if rows == 0 {
		rows = max(1, (count+columns-1)/columns)
	}
	if columns == 0 {
		columns = max(1, (count+rows-1)/rows)
	}
	return rows, columns
}

func measureGridLocked(state *layoutState) Size {
	cellW, cellH := 0, 0
	for _, item := range state.items {
		cellW = max(cellW, checkedLayoutSum(
			item.minimum.Width,
			item.options.Insets.Left,
			item.options.Insets.Right,
		))
		cellH = max(cellH, checkedLayoutSum(
			item.minimum.Height,
			item.options.Insets.Top,
			item.options.Insets.Bottom,
		))
	}
	rows, columns := gridDimensionsLocked(state)
	return Size{
		Width: checkedLayoutSum(
			state.grid.Insets.Left,
			state.grid.Insets.Right,
			checkedLayoutProduct(columns, cellW),
			checkedLayoutProduct(max(0, columns-1), state.grid.HorizontalGap),
		),
		Height: checkedLayoutSum(
			state.grid.Insets.Top,
			state.grid.Insets.Bottom,
			checkedLayoutProduct(rows, cellH),
			checkedLayoutProduct(max(0, rows-1), state.grid.VerticalGap),
		),
	}
}

func checkedLayoutSum(values ...int) int {
	const overflow = maxCoordinateMagnitude + 1
	total := int64(0)
	for _, value := range values {
		total += int64(value)
		if total > overflow {
			return overflow
		}
	}
	return int(total)
}

func checkedLayoutProduct(left, right int) int {
	const overflow = maxCoordinateMagnitude + 1
	product := int64(left) * int64(right)
	if product > overflow {
		return overflow
	}
	return int(product)
}

func arrangeAllLayoutsLocked(app *App) {
	for _, state := range app.layoutsByID {
		if state.parent == nil && !state.destroyed {
			measureLayoutLocked(state)
		}
	}
	app.reconcileModalsLocked()
	arrangeControlLayoutsLocked(app, app.root.state)
	app.reconcileOverflowsLocked()
}

func arrangeControlLayoutsLocked(app *App, owner *controlState) {
	if owner == nil || owner.destroyed {
		return
	}
	reconcileScrollViewLocked(owner)
	reconcileMarkdownLocked(owner)
	reconcileLogViewLocked(owner)
	reconcileListBoxLocked(owner)
	reconcileTreeViewLocked(owner)
	reconcileTableLocked(owner)
	reconcileDataGridLocked(owner)
	for _, root := range owner.layoutRoots {
		client := controlClientSize(owner)
		if owner.root {
			content := app.rootContentRectLocked()
			client = Size{Width: content.Width, Height: content.Height}
		}
		arrangeLayoutLocked(
			root,
			Rect{Width: client.Width, Height: client.Height},
			Point{},
		)
	}
	if behavior, ok := owner.behavior.(tabbedPanelBehavior); ok {
		client := controlClientSize(owner)
		for _, tab := range behavior.tabs {
			if tab.page == nil || tab.page.destroyed {
				continue
			}
			tab.page.bounds = Rect{
				Width: client.Width, Height: client.Height,
			}
		}
	}
	for _, child := range owner.children {
		arrangeControlLayoutsLocked(app, child)
	}
}

func controlClientSize(state *controlState) Size {
	client := controlClientRect(
		state,
		Rect{Width: state.bounds.Width, Height: state.bounds.Height},
	)
	return Size{
		Width:  client.Width,
		Height: client.Height,
	}
}

func arrangeLayoutLocked(state *layoutState, bounds Rect, ownerOrigin Point) {
	state.bounds = bounds
	state.ownerBounds = Rect{
		X: ownerOrigin.X + bounds.X, Y: ownerOrigin.Y + bounds.Y,
		Width: bounds.Width, Height: bounds.Height,
	}
	if state.kind == LayoutGrid {
		arrangeGridLocked(state)
	} else {
		arrangeBoxLocked(state)
	}
	for _, item := range state.items {
		if item.kind == layoutPanelItem {
			item.panel.bounds = Rect{
				X:     item.bounds.X + state.ownerBounds.X,
				Y:     item.bounds.Y + state.ownerBounds.Y,
				Width: item.bounds.Width, Height: item.bounds.Height,
			}
		} else {
			arrangeLayoutLocked(
				item.layout,
				item.bounds,
				Point{X: state.ownerBounds.X, Y: state.ownerBounds.Y},
			)
		}
	}
}

func arrangeBoxLocked(state *layoutState) {
	content := layoutBorderClientRect(state)
	content = insetBy(
		content,
		state.box.Insets,
	)
	mainAvailable, crossAvailable := content.Width, content.Height
	if state.orientation == Vertical {
		mainAvailable, crossAvailable = content.Height, content.Width
	}
	minMain := 0
	totalGrow := int64(0)
	explicitGrow := false
	for index, item := range state.items {
		if index > 0 {
			minMain += state.box.Gap
		}
		if state.orientation == Horizontal {
			minMain += item.minimum.Width + item.options.Insets.Left + item.options.Insets.Right
		} else {
			minMain += item.minimum.Height + item.options.Insets.Top + item.options.Insets.Bottom
		}
		totalGrow += int64(item.options.Grow)
		explicitGrow = explicitGrow || item.options.Grow > 0
	}
	if !explicitGrow {
		totalGrow = 0
		for _, item := range state.items {
			totalGrow += int64(layoutItemWeight(
				item,
				state.orientation == Horizontal,
			))
		}
	}
	extra := max(0, mainAvailable-minMain)
	growth := make([]int, len(state.items))
	if totalGrow > 0 {
		assigned := 0
		for index, item := range state.items {
			weight := item.options.Grow
			if !explicitGrow {
				weight = layoutItemWeight(
					item,
					state.orientation == Horizontal,
				)
			}
			if weight > 0 {
				growth[index] = int(
					int64(extra) * int64(weight) / totalGrow,
				)
				assigned += growth[index]
			}
		}
		remainder := extra - assigned
		for index, item := range state.items {
			if remainder == 0 {
				break
			}
			weight := item.options.Grow
			if !explicitGrow {
				weight = layoutItemWeight(
					item,
					state.orientation == Horizontal,
				)
			}
			if weight > 0 {
				growth[index]++
				remainder--
			}
		}
	}
	cursor := 0
	for index, item := range state.items {
		if index > 0 {
			cursor += state.box.Gap
		}
		base := item.minimum.Width + item.options.Insets.Left + item.options.Insets.Right
		crossMin := item.minimum.Height
		crossBefore, crossAfter := item.options.Insets.Top, item.options.Insets.Bottom
		align := item.options.VerticalAlign
		if state.orientation == Vertical {
			base = item.minimum.Height + item.options.Insets.Top + item.options.Insets.Bottom
			crossMin = item.minimum.Width
			crossBefore, crossAfter = item.options.Insets.Left, item.options.Insets.Right
			align = item.options.HorizontalAlign
		}
		align = effectiveItemAlignment(
			item,
			align,
			state.orientation == Vertical,
		)
		slotMain := base + growth[index]
		crossPos, crossSize := alignedSpan(
			crossAvailable, crossMin, crossBefore, crossAfter, align,
		)
		if state.orientation == Horizontal {
			item.bounds = Rect{
				X: content.X + cursor + item.options.Insets.Left,
				Y: content.Y + crossPos,
				Width: max(item.minimum.Width,
					slotMain-item.options.Insets.Left-item.options.Insets.Right),
				Height: crossSize,
			}
		} else {
			item.bounds = Rect{
				X:     content.X + crossPos,
				Y:     content.Y + cursor + item.options.Insets.Top,
				Width: crossSize,
				Height: max(item.minimum.Height,
					slotMain-item.options.Insets.Top-item.options.Insets.Bottom),
			}
		}
		cursor += slotMain
	}
}

func arrangeGridLocked(state *layoutState) {
	content := layoutBorderClientRect(state)
	content = insetBy(
		content,
		state.grid.Insets,
	)
	rows, columns := gridDimensionsLocked(state)
	cellMinW, cellMinH := 0, 0
	for _, item := range state.items {
		cellMinW = max(cellMinW, item.minimum.Width+
			item.options.Insets.Left+item.options.Insets.Right)
		cellMinH = max(cellMinH, item.minimum.Height+
			item.options.Insets.Top+item.options.Insets.Bottom)
	}
	availableW := max(0, content.Width-max(0, columns-1)*state.grid.HorizontalGap)
	availableH := max(0, content.Height-max(0, rows-1)*state.grid.VerticalGap)
	widths := distributedSpans(columns, cellMinW, availableW)
	heights := distributedSpans(rows, cellMinH, availableH)
	xs, ys := make([]int, columns), make([]int, rows)
	for column := 1; column < columns; column++ {
		xs[column] = xs[column-1] + widths[column-1] + state.grid.HorizontalGap
	}
	for row := 1; row < rows; row++ {
		ys[row] = ys[row-1] + heights[row-1] + state.grid.VerticalGap
	}
	for index, item := range state.items {
		row, column := index/columns, index%columns
		x, width := alignedSpan(
			widths[column],
			item.minimum.Width,
			item.options.Insets.Left,
			item.options.Insets.Right,
			effectiveItemAlignment(
				item,
				item.options.HorizontalAlign,
				true,
			),
		)
		y, height := alignedSpan(
			heights[row],
			item.minimum.Height,
			item.options.Insets.Top,
			item.options.Insets.Bottom,
			effectiveItemAlignment(
				item,
				item.options.VerticalAlign,
				false,
			),
		)
		item.bounds = Rect{
			X:     content.X + xs[column] + x,
			Y:     content.Y + ys[row] + y,
			Width: width, Height: height,
		}
	}
}

func layoutBorderClientRect(state *layoutState) Rect {
	rect := Rect{Width: state.bounds.Width, Height: state.bounds.Height}
	if state.border.form == BorderNone {
		return rect
	}
	return insetRect(rect, 1)
}

func insetBy(rect Rect, insets Insets) Rect {
	return Rect{
		X:      rect.X + min(rect.Width, insets.Left),
		Y:      rect.Y + min(rect.Height, insets.Top),
		Width:  max(0, rect.Width-insets.Left-insets.Right),
		Height: max(0, rect.Height-insets.Top-insets.Bottom),
	}
}

func alignedSpan(
	available, minimum, before, after int,
	alignment Alignment,
) (position, size int) {
	inner := max(0, available-before-after)
	size = minimum
	if alignment == AlignStretch {
		size = max(minimum, inner)
	}
	position = before
	if alignment == AlignCenter {
		position = before + max(0, (inner-size)/2)
	} else if alignment == AlignEnd {
		position = max(before, available-after-size)
	}
	return position, size
}

func layoutItemHints(item *layoutItem) LayoutHints {
	if item != nil && item.kind == layoutPanelItem && item.panel != nil {
		return item.panel.layoutHints
	}
	return LayoutHints{
		Horizontal:       LayoutSizeStretch,
		Vertical:         LayoutSizeStretch,
		HorizontalWeight: 1,
		VerticalWeight:   1,
	}
}

func layoutItemStretches(item *layoutItem, horizontal bool) bool {
	hints := layoutItemHints(item)
	if horizontal {
		return hints.Horizontal == LayoutSizeStretch
	}
	return hints.Vertical == LayoutSizeStretch
}

func layoutItemWeight(item *layoutItem, horizontal bool) int {
	hints := layoutItemHints(item)
	if horizontal {
		return hints.HorizontalWeight
	}
	return hints.VerticalWeight
}

func effectiveItemAlignment(
	item *layoutItem,
	alignment Alignment,
	horizontal bool,
) Alignment {
	if alignment != AlignDefault {
		return alignment
	}
	if layoutItemStretches(item, horizontal) {
		return AlignStretch
	}
	return AlignStart
}

func distributedSpans(count, minimum, available int) []int {
	spans := make([]int, count)
	if count == 0 {
		return spans
	}
	total := max(available, count*minimum)
	each, remainder := total/count, total%count
	for index := range spans {
		spans[index] = each
		if index < remainder {
			spans[index]++
		}
	}
	return spans
}

func reorderPeerItems(stack []*layoutItem, target *layoutItem, raise bool) bool {
	kind := target.kind
	indices := make([]int, 0, len(stack))
	values := make([]*layoutItem, 0, len(stack))
	current := -1
	for index, item := range stack {
		if item.kind != kind {
			continue
		}
		if item == target {
			current = len(values)
		}
		indices = append(indices, index)
		values = append(values, item)
	}
	if current < 0 || len(values) < 2 ||
		(raise && current == len(values)-1) || (!raise && current == 0) {
		return false
	}
	copy(values[current:], values[current+1:])
	if raise {
		values[len(values)-1] = target
	} else {
		copy(values[1:current+1], values[:current])
		values[0] = target
	}
	for index, slot := range indices {
		stack[slot] = values[index]
	}
	return true
}

func removeLayoutPanelLocked(state *controlState) {
	layout := state.layout
	if layout == nil {
		return
	}
	for index, item := range layout.items {
		if item.kind == layoutPanelItem && item.panel == state {
			layout.items = append(layout.items[:index], layout.items[index+1:]...)
			break
		}
	}
	for index, item := range layout.stack {
		if item.kind == layoutPanelItem && item.panel == state {
			layout.stack = append(layout.stack[:index], layout.stack[index+1:]...)
			break
		}
	}
	state.layout = nil
	layout.app.layoutItemCount--
}

func destroyLayoutTreeLocked(app *App, state *layoutState) {
	for _, item := range state.items {
		if item.kind == layoutPanelItem {
			item.panel.layout = nil
		} else {
			destroyLayoutTreeLocked(app, item.layout)
		}
	}
	delete(app.layoutsByID, state.id)
	delete(app.layoutsByKey, state.automationKey)
	delete(app.overflows, state.id)
	app.layoutItemCount -= len(state.items)
	state.destroyed = true
	state.attached = false
	state.app = app
	state.owner = nil
}

func flattenLayoutPanelsLocked(state *layoutState, result *[]*controlState) {
	for _, item := range state.stack {
		if item.kind == layoutPanelItem {
			if !item.panel.destroyed {
				*result = append(*result, item.panel)
			}
		} else {
			flattenLayoutPanelsLocked(item.layout, result)
		}
	}
}

// Internal compile-time checks for the sealed public interface.
var (
	_ Layout = (*BoxLayout)(nil)
	_ Layout = (*GridLayout)(nil)
	_        = errors.Is
)
