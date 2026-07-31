package expletives

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// AppOptions configures one independent toolkit application.
type AppOptions struct {
	// Size is the initial physical container surface.
	Size Size
	// RootConstraints optionally bound and aspect-constrain the centered root
	// Panel. The zero value makes the root fill Size.
	RootConstraints RootConstraints
	// Theme is copied by NewApp; the zero value selects DefaultTheme.
	Theme Theme
	// RootStyle defaults to "application.root".
	RootStyle StyleID
	// Scenario defaults to "unspecified" and must be a bounded identifier.
	Scenario string
}

// App owns one control tree, input state, renderer, and immutable snapshot
// history. Public methods are safe for concurrent use. An App must not be
// copied after first use.
type App struct {
	mu           sync.RWMutex
	mutationGate chan struct{}
	dispatchGate chan struct{}
	handlerSlots chan struct{}

	size            Size
	rootConstraints RootConstraints
	scenario        string
	root            *Panel
	nextControl     uint64
	nextLayout      uint64
	controlsByID    map[ControlID]*controlState
	controlsByKey   map[string]*controlState
	layoutsByID     map[LayoutID]*layoutState
	layoutsByKey    map[string]*layoutState
	layoutItemCount int
	styles          map[StyleID]ResolvedStyle
	cursor          CursorState
	held            map[string]map[Key]bool
	focus           *controlState
	pressed         map[string]pressedAction
	menu            *menuSession
	bindings        map[string]CommandID
	commandRouter   CommandRouter
	legacyRouter    bool
	commands        map[CommandID]CommandDefinition
	final           bool
	sequence        uint64
	snapshot        Snapshot
	history         map[uint64]Snapshot
	historyOrder    []uint64
	historyCells    int
	changed         chan struct{}
	overflowRunning bool
	overflowGate    chan struct{}
	overflowHandler OverflowHandler
	overflows       map[LayoutID]*overflowRecord
	nextOverflow    uint64
}

// NewApp validates and copies options, creates the sole parentless root Panel,
// and publishes the initial snapshot.
func NewApp(options AppOptions) (*App, error) {
	if err := validateSize(options.Size); err != nil {
		return nil, err
	}
	if err := validateRootConstraints(options.RootConstraints); err != nil {
		return nil, err
	}
	theme := options.Theme
	if len(theme.styles) == 0 {
		theme = DefaultTheme()
	}
	rootStyle, err := normalizeStyleID(options.RootStyle, "application.root")
	if err != nil {
		return nil, err
	}
	if _, found := theme.styles[rootStyle]; !found {
		return nil, fmt.Errorf(
			"%w: theme has no definition for %q",
			ErrStyleMissing,
			rootStyle,
		)
	}
	if options.Scenario == "" {
		options.Scenario = "unspecified"
	}
	if !validBoundedIdentifier(options.Scenario) {
		return nil, errors.New("expletives: scenario ID is invalid or too long")
	}

	app := &App{
		size:            options.Size,
		rootConstraints: options.RootConstraints,
		scenario:        options.Scenario,
		nextControl:     2,
		nextLayout:      1,
		controlsByID:    make(map[ControlID]*controlState),
		controlsByKey:   make(map[string]*controlState),
		layoutsByID:     make(map[LayoutID]*layoutState),
		layoutsByKey:    make(map[string]*layoutState),
		styles:          cloneStyleMap(theme.styles),
		held:            make(map[string]map[Key]bool),
		pressed:         make(map[string]pressedAction),
		bindings:        make(map[string]CommandID),
		commands:        make(map[CommandID]CommandDefinition),
		mutationGate:    make(chan struct{}, 1),
		dispatchGate:    make(chan struct{}, 1),
		handlerSlots:    make(chan struct{}, MaxConcurrentCommandHandlers),
		history:         make(map[uint64]Snapshot),
		changed:         make(chan struct{}),
		overflowGate:    make(chan struct{}, 1),
		overflows:       make(map[LayoutID]*overflowRecord),
	}
	rootState := &controlState{
		app:           app,
		id:            "root",
		automationKey: "root",
		kind:          ControlRoot,
		bounds:        constrainedRootBounds(options.Size, options.RootConstraints),
		minimumSize:   options.RootConstraints.Minimum,
		style:         rootStyle,
		visible:       true,
		root:          true,
		behavior:      containerBehavior{},
	}
	root := &Panel{
		containerHandle: containerHandle{
			controlHandle: controlHandle{state: rootState},
		},
	}
	rootState.control = root
	rootState.container = root
	app.root = root
	app.commands[CommandOverflowDismiss] = CommandDefinition{
		ID:          CommandOverflowDismiss,
		Label:       "OK",
		Description: "Acknowledge the active Layout overflow warning",
		Enabled:     true,
		Automation:  true,
	}
	app.controlsByID[rootState.id] = rootState
	app.controlsByKey[rootState.automationKey] = rootState
	app.publishLocked(nil)
	return app, nil
}

// Root returns the App-owned special root Panel. The returned copy-safe handle
// remains owned by the App and cannot be destroyed.
func (a *App) Root() *Panel {
	return a.root
}

// ControlByAutomationKey resolves one active stable application-selected key.
// The returned control is a copy-safe handle owned by this App. Invalid,
// absent, provisional, and destroyed controls return false.
func (a *App) ControlByAutomationKey(key string) (Control, bool) {
	if !validBoundedIdentifier(key) {
		return nil, false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	state, found := a.controlsByKey[key]
	if !found || state.destroyed {
		return nil, false
	}
	return state.control, true
}

// Scenario returns the immutable scenario identity.
func (a *App) Scenario() string {
	return a.scenario
}

// Size returns the current application surface.
func (a *App) Size() Size {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.size
}

// RootConstraints returns the current caller-owned root sizing policy.
func (a *App) RootConstraints() RootConstraints {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.rootConstraints
}

// SetRootConstraints changes the centered root sizing policy atomically.
func (a *App) SetRootConstraints(constraints RootConstraints) error {
	transaction := a.NewTransaction()
	if err := transaction.SetRootConstraints(constraints); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// SetSize changes the application surface and root rectangle atomically. A
// changed size publishes one snapshot. SetSize waits at most
// DefaultMutationWait; use Transaction.Commit when cancellation or batching
// is required.
func (a *App) SetSize(size Size) error {
	transaction := a.NewTransaction()
	if err := transaction.SetSize(size); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Snapshot returns a deep caller-owned copy of the current atomic state.
func (a *App) Snapshot() Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneSnapshot(a.snapshot)
}

// SnapshotAt returns a deep copy of an exact retained snapshot. It returns
// ErrSnapshotNotRetained when sequence has expired or never existed.
func (a *App) SnapshotAt(sequence uint64) (Snapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	snapshot, ok := a.history[sequence]
	if !ok {
		return Snapshot{}, ErrSnapshotNotRetained
	}
	return cloneSnapshot(snapshot), nil
}

// WaitSnapshot waits for and returns a deep copy of a current snapshot later
// than after. It returns ctx.Err on cancellation and ErrClosed when the App is
// final with no later snapshot.
func (a *App) WaitSnapshot(
	ctx context.Context,
	after uint64,
) (Snapshot, error) {
	if ctx == nil {
		return Snapshot{}, errors.New("expletives: nil context")
	}
	for {
		a.mu.RLock()
		if a.snapshot.Sequence > after {
			snapshot := cloneSnapshot(a.snapshot)
			a.mu.RUnlock()
			return snapshot, nil
		}
		if a.final {
			a.mu.RUnlock()
			return Snapshot{}, ErrClosed
		}
		changed := a.changed
		a.mu.RUnlock()
		select {
		case <-ctx.Done():
			return Snapshot{}, ctx.Err()
		case <-changed:
		}
	}
}

func (a *App) nextControlIDLocked() ControlID {
	id := ControlID(fmt.Sprintf("control-%d", a.nextControl))
	a.nextControl++
	return id
}

func (a *App) nextLayoutIDLocked() LayoutID {
	id := LayoutID(fmt.Sprintf("layout-%d", a.nextLayout))
	a.nextLayout++
	return id
}

func (a *App) publishLocked(association *Completion) {
	a.sequence++
	snapshot := a.renderLocked()
	snapshot.Sequence = a.sequence
	snapshot.Final = a.final
	if association != nil {
		copied := *association
		snapshot.Completion = &copied
	}
	a.snapshot = snapshot
	a.history[snapshot.Sequence] = snapshot
	a.historyOrder = append(a.historyOrder, snapshot.Sequence)
	a.historyCells += len(snapshot.Frame.Cells)
	for (a.historyCells > MaxRetainedFrameCells ||
		len(a.historyOrder) > MaxSnapshotHistoryRecords) &&
		len(a.historyOrder) > 1 {
		expired := a.historyOrder[0]
		a.historyCells -= len(a.history[expired].Frame.Cells)
		delete(a.history, expired)
		a.historyOrder = a.historyOrder[1:]
	}
	close(a.changed)
	a.changed = make(chan struct{})
}

func (a *App) renderLocked() Snapshot {
	a.cursor = CursorState{}
	cellCount, _ := frameCellCount(a.size)
	frame := IntendedFrame{
		Size:  a.size,
		Cells: make([]Cell, cellCount),
	}
	controls := make([]ControlSnapshot, 0, len(a.controlsByID))
	surface := Rect{Width: a.size.Width, Height: a.size.Height}
	a.fillLocked(&frame, surface, a.root.state.style, a.root.state.id)
	a.paintControlLocked(
		a.root.state,
		Point{},
		surface,
		true,
		&frame,
		&controls,
	)

	sources := make([]string, 0, len(a.held))
	for source := range a.held {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	held := make([]InputSourceSnapshot, 0, len(sources))
	for _, source := range sources {
		keys := make([]Key, 0, len(a.held[source]))
		for key, down := range a.held[source] {
			if down {
				keys = append(keys, key)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		if len(keys) != 0 {
			held = append(held, InputSourceSnapshot{Source: source, Held: keys})
		}
	}

	layouts := a.layoutSnapshotsLocked()
	overflows := a.overflowSnapshotsLocked()
	snapshot := Snapshot{
		Version:      SnapshotVersion,
		Scenario:     a.scenario,
		Final:        a.final,
		Frame:        frame,
		Cursor:       a.cursor,
		InputSources: held,
		Controls:     controls,
		Layouts:      layouts,
		Overflows:    overflows,
	}
	a.paintStatusBarChromeLocked(&snapshot.Frame)
	a.paintMenuBarChromeLocked(&snapshot.Frame)
	a.paintMenuOverlayLocked(&snapshot.Frame)
	a.paintOverflowWarningLocked(&snapshot.Frame, overflows)
	if a.menu != nil || defaultOverflowActive(overflows) {
		snapshot.Cursor = CursorState{}
	}
	return snapshot
}

func (a *App) paintControlLocked(
	state *controlState,
	parentOrigin Point,
	ancestorClip Rect,
	ancestorsVisible bool,
	frame *IntendedFrame,
	controls *[]ControlSnapshot,
) {
	if state.destroyed {
		return
	}
	bounds := state.bounds
	absolute := bounds
	chrome := isApplicationChrome(state.kind)
	postPaintChrome := isPostPaintApplicationChrome(state.kind)
	if chrome {
		switch state.kind {
		case ControlMenuBar:
			bounds = menuBarSurfaceRect(a.size)
		case ControlStatusBar:
			bounds = statusBarSurfaceRect(a.size)
		case ControlHeader, ControlFooter:
			bounds = state.bounds
		}
		absolute = bounds
		ancestorClip = Rect{Width: a.size.Width, Height: a.size.Height}
	} else if !state.root {
		absolute.X += parentOrigin.X
		absolute.Y += parentOrigin.Y
	}
	clip := absolute.Intersect(ancestorClip)
	visible := ancestorsVisible && state.visible
	if !visible {
		clip.Width, clip.Height = 0, 0
	}

	parentID := ControlID("")
	if state.parent != nil {
		parentID = state.parent.id
	}
	details := state.behavior.details()
	if details.Border != nil {
		border := state.behavior.(borderBehavior)
		details.Border.ResolvedStyle = resolveBorderStyle(
			a.styles[details.Border.Style],
			border,
		)
	}
	if behavior, ok := state.behavior.(buttonBehavior); ok {
		action := a.actionDetailsLocked(state, behavior)
		details.Action = &action
	}
	if behavior, ok := state.behavior.(hotkeyBarBehavior); ok {
		details.HotkeyBar = &HotkeyBarDetails{
			Items: a.hotkeyBarItemDetailsLocked(behavior.items),
		}
	}
	if behavior, ok := state.behavior.(menuBarBehavior); ok {
		menuBar := a.menuBarDetailsLocked(state, behavior)
		details.MenuBar = &menuBar
	}
	if behavior, ok := state.behavior.(statusBarBehavior); ok {
		renderWidth := bounds.Width
		if !visible {
			renderWidth = 0
		}
		plans := a.statusBarRenderPlanLocked(behavior, renderWidth)
		statusBar := StatusBarDetails{
			Segments: make([]StatusSegmentDetails, len(plans)),
		}
		for index := range plans {
			statusBar.Segments[index] = plans[index].details
		}
		details.StatusBar = &statusBar
	}
	switch behavior := state.behavior.(type) {
	case checkboxBehavior:
		checkbox := a.checkboxDetailsLocked(behavior)
		details.Checkbox = &checkbox
	case radioGroupBehavior:
		group := a.radioGroupDetailsLocked(state, behavior)
		details.RadioGroup = &group
	case radioButtonBehavior:
		button := a.radioButtonDetailsLocked(state, behavior)
		details.RadioButton = &button
	case choiceFieldBehavior:
		field := a.choiceFieldDetailsLocked(behavior)
		details.ChoiceField = &field
	case focusGuideBarBehavior:
		guide := a.focusGuideBarDetailsLocked()
		details.FocusGuideBar = &guide
	case textFieldBehavior:
		field := a.textFieldDetailsLocked(state, behavior)
		details.TextField = &field
	}
	*controls = append(*controls, ControlSnapshot{
		ID:             state.id,
		Key:            state.automationKey,
		Kind:           state.kind,
		Parent:         parentID,
		Children:       childIDs(state.children),
		Bounds:         bounds,
		AbsoluteBounds: absolute,
		EffectiveClip:  clip,
		Minimum:        state.minimumSize,
		Layout:         layoutIDForControl(state),
		LayoutIndex:    layoutArrangementIndex(state),
		StackIndex:     layoutStackIndex(state),
		Style:          state.style,
		ResolvedStyle:  a.styles[state.style],
		Visible:        visible,
		Focused:        a.focus == state,
		Details:        details,
	})

	if visible && !clip.Empty() && !postPaintChrome {
		a.fillLocked(frame, clip, state.style, state.id)
		state.behavior.paintDecoration(a, frame, state, absolute, clip)
	}

	clientRect := absolute
	if state.root {
		clientRect = a.rootContentRectLocked()
	} else {
		clientInset := state.behavior.clientInset()
		if clientInset != 0 {
			clientRect = insetRect(absolute, clientInset)
		}
	}
	childClip := clip.Intersect(clientRect)
	childOrigin := Point{X: clientRect.X, Y: clientRect.Y}
	if len(state.layoutRoots) == 0 {
		for _, child := range state.children {
			a.paintControlLocked(
				child,
				childOrigin,
				childClip,
				visible,
				frame,
				controls,
			)
		}
		return
	}
	managed := make(map[*controlState]bool)
	for _, layout := range state.layoutRoots {
		a.paintLayoutLocked(
			layout,
			childOrigin,
			childClip,
			visible,
			frame,
			controls,
			managed,
		)
	}
	for _, child := range state.children {
		if managed[child] {
			continue
		}
		a.paintControlLocked(
			child,
			childOrigin,
			childClip,
			visible,
			frame,
			controls,
		)
	}
}

func (a *App) applicationContentRectLocked() Rect {
	return a.applicationChromeContentRectLocked()
}

func (a *App) rootContentRectLocked() Rect {
	return a.root.state.bounds.Intersect(a.applicationContentRectLocked())
}

func (a *App) paintMenuBarChromeLocked(frame *IntendedFrame) {
	state := a.firstMenuBarLocked()
	if state == nil {
		return
	}
	behavior := state.behavior.(menuBarBehavior)
	rect := menuBarSurfaceRect(a.size)
	if rect.Empty() {
		return
	}
	a.fillLocked(frame, rect, state.style, state.id)
	behavior.paintDecoration(a, frame, state, rect, rect)
}

func (a *App) paintLayoutLocked(
	layout *layoutState,
	ownerOrigin Point,
	ancestorClip Rect,
	visible bool,
	frame *IntendedFrame,
	controls *[]ControlSnapshot,
	managed map[*controlState]bool,
) {
	if layout == nil || layout.destroyed {
		return
	}
	absolute := layout.ownerBounds
	absolute.X += ownerOrigin.X
	absolute.Y += ownerOrigin.Y
	a.paintBorderLocked(
		frame,
		absolute,
		absolute.Intersect(ancestorClip),
		layout.owner,
		layout.border,
	)
	for _, item := range layout.stack {
		if item.kind == layoutPanelItem {
			managed[item.panel] = true
			a.paintControlLocked(
				item.panel,
				ownerOrigin,
				ancestorClip,
				visible,
				frame,
				controls,
			)
			continue
		}
		a.paintLayoutLocked(
			item.layout,
			ownerOrigin,
			ancestorClip,
			visible,
			frame,
			controls,
			managed,
		)
	}
}

func insetRect(rect Rect, inset int) Rect {
	width := max(0, rect.Width-inset*2)
	height := max(0, rect.Height-inset*2)
	return Rect{
		X:      rect.X + min(inset, rect.Width),
		Y:      rect.Y + min(inset, rect.Height),
		Width:  width,
		Height: height,
	}
}

func (a *App) fillLocked(
	frame *IntendedFrame,
	rect Rect,
	style StyleID,
	owner ControlID,
) {
	resolved := a.styles[style]
	for y := rect.Y; y < rect.Y+rect.Height; y++ {
		for x := rect.X; x < rect.X+rect.Width; x++ {
			a.setCellLocked(frame, x, y, " ", style, resolved, owner)
		}
	}
}

func (a *App) paintBorderLocked(
	frame *IntendedFrame,
	absolute Rect,
	clip Rect,
	state *controlState,
	behavior borderBehavior,
) {
	if absolute.Width <= 0 || absolute.Height <= 0 ||
		behavior.form == BorderNone {
		return
	}
	style := behavior.borderStyle
	resolved := resolveBorderStyle(a.styles[style], behavior)
	glyphs := glyphsForBorder(behavior.form)
	left := absolute.X
	right := absolute.X + absolute.Width - 1
	top := absolute.Y
	bottom := absolute.Y + absolute.Height - 1
	for x := left; x <= right; x++ {
		a.setClippedCellLocked(
			frame, clip, x, top, glyphs.horizontal, style, resolved, state.id,
		)
		a.setClippedCellLocked(
			frame, clip, x, bottom, glyphs.horizontal, style, resolved, state.id,
		)
	}
	for y := top; y <= bottom; y++ {
		a.setClippedCellLocked(
			frame, clip, left, y, glyphs.vertical, style, resolved, state.id,
		)
		a.setClippedCellLocked(
			frame, clip, right, y, glyphs.vertical, style, resolved, state.id,
		)
	}
	a.setClippedCellLocked(
		frame, clip, left, top, glyphs.topLeft, style, resolved, state.id,
	)
	a.setClippedCellLocked(
		frame, clip, right, top, glyphs.topRight, style, resolved, state.id,
	)
	a.setClippedCellLocked(
		frame, clip, left, bottom, glyphs.bottomLeft, style, resolved, state.id,
	)
	a.setClippedCellLocked(
		frame, clip, right, bottom, glyphs.bottomRight, style, resolved, state.id,
	)

	if absolute.Width < 5 || behavior.title == "" {
		return
	}
	titleCells := behavior.titleCells
	available := absolute.Width - 4
	if len(titleCells) > available {
		titleCells = titleCells[:available]
	}
	for index, grapheme := range titleCells {
		a.setClippedCellLocked(
			frame,
			clip,
			left+2+index,
			top,
			grapheme,
			style,
			resolved,
			state.id,
		)
	}
}

type borderGlyphs struct {
	topLeft, topRight, bottomLeft, bottomRight string
	horizontal, vertical                       string
}

func glyphsForBorder(form BorderForm) borderGlyphs {
	switch form {
	case BorderNone:
		return uniformBorderGlyphs(" ")
	case BorderDouble:
		return borderGlyphs{
			topLeft: "╔", topRight: "╗", bottomLeft: "╚", bottomRight: "╝",
			horizontal: "═", vertical: "║",
		}
	case BorderShadeLight:
		return uniformBorderGlyphs("░")
	case BorderShadeMedium:
		return uniformBorderGlyphs("▒")
	case BorderShadeDark:
		return uniformBorderGlyphs("▓")
	case BorderBlock:
		return uniformBorderGlyphs("█")
	default:
		return borderGlyphs{
			topLeft: "┌", topRight: "┐", bottomLeft: "└", bottomRight: "┘",
			horizontal: "─", vertical: "│",
		}
	}
}

func uniformBorderGlyphs(glyph string) borderGlyphs {
	return borderGlyphs{
		topLeft: glyph, topRight: glyph, bottomLeft: glyph, bottomRight: glyph,
		horizontal: glyph, vertical: glyph,
	}
}

func (a *App) setClippedCellLocked(
	frame *IntendedFrame,
	clip Rect,
	x int,
	y int,
	grapheme string,
	style StyleID,
	resolved ResolvedStyle,
	owner ControlID,
) {
	if x < clip.X || y < clip.Y ||
		x >= clip.X+clip.Width || y >= clip.Y+clip.Height {
		return
	}
	a.setCellLocked(frame, x, y, grapheme, style, resolved, owner)
}

func (a *App) setCellLocked(
	frame *IntendedFrame,
	x int,
	y int,
	grapheme string,
	style StyleID,
	resolved ResolvedStyle,
	owner ControlID,
) {
	if x < 0 || y < 0 ||
		x >= frame.Size.Width || y >= frame.Size.Height {
		return
	}
	frame.Cells[y*frame.Size.Width+x] = Cell{
		Grapheme:   grapheme,
		Style:      style,
		Foreground: resolved.Foreground,
		Background: resolved.Background,
		Attributes: resolved.Attributes,
		Owner:      owner,
	}
}

func (a *App) associateLocked(
	requestID string,
	result CommandResult,
	command CommandID,
) Completion {
	result = normalizeCommandResult(result)
	if result.Outcome == OutcomeExited ||
		result.Outcome == OutcomeInterrupted {
		a.final = true
	}
	completion := Completion{
		RequestID:     requestID,
		Outcome:       result.Outcome,
		Command:       command,
		FrameSequence: a.sequence + 1,
		Code:          result.Code,
		Message:       result.Message,
		Cause:         result.Cause,
	}
	a.publishLocked(&completion)
	return completion
}

func childIDs(children []*controlState) []ControlID {
	ids := make([]ControlID, 0, len(children))
	for _, child := range children {
		if !child.destroyed {
			ids = append(ids, child.id)
		}
	}
	return ids
}
