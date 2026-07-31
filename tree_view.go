package expletives

import (
	"context"
	"errors"
	"fmt"
)

// TreeNode is one copied stable-identity TreeView node.
type TreeNode struct {
	Key            string
	Label          string
	Disabled       bool
	DisabledReason string
	Expanded       bool
	Children       []TreeNode
}

// TreeViewOptions configures one bounded stable-identity scrolling tree.
//
// ContentAutomationKey, ContentStyle, and State.ContentSize are reserved by
// TreeView and must be empty. A nil Expanded slice uses the Expanded flags in
// Nodes; a non-nil slice is an exact initial expansion set.
type TreeViewOptions struct {
	ScrollablePanelOptions
	Nodes            []TreeNode
	Current          string
	Selected         []string
	Expanded         []string
	SelectionMode    CollectionSelectionMode
	RequireSelection bool
	Status           CollectionStatus
	StatusMessage    string
	ActivateCommand  CommandID
	ExpandCommand    CommandID
}

// TreeViewState is one complete copied semantic and viewport state.
type TreeViewState struct {
	Status        CollectionStatus
	StatusMessage string
	Current       string
	CurrentIndex  int
	Selected      []string
	Expanded      []string
	Offset        Point
	NodeCount     int
	VisibleCount  int
	EnabledCount  int
}

// TreeView is a copy-safe focusable stable-identity collection leaf.
type TreeView struct{ controlHandle }

type normalizedTreeNode struct {
	item       TreeNode
	label      normalizedDisplayText
	parent     int
	depth      int
	children   []int
	subtreeEnd int
}

type treeViewBehavior struct {
	scroll           scrollViewBehavior
	nodes            []normalizedTreeNode
	current          string
	selected         []string
	expanded         []string
	selectionMode    CollectionSelectionMode
	requireSelection bool
	status           CollectionStatus
	statusMessage    normalizedDisplayText
	changeCommand    CommandID
	activateCommand  CommandID
	expandCommand    CommandID
}

type treeRowCell struct {
	grapheme string
	style    StyleID
}

// NewTreeView constructs and atomically inserts a TreeView.
func NewTreeView(parent Container, options TreeViewOptions) (*TreeView, error) {
	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	control, err := transaction.NewTreeView(parent, options)
	if err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}
	return control, nil
}

// NewTreeView records construction of a provisional TreeView.
func (t *Transaction) NewTreeView(
	parent Container,
	options TreeViewOptions,
) (*TreeView, error) {
	behavior, err := newTreeViewBehavior(options)
	if err != nil {
		return nil, err
	}
	behavior = reflowTreeView(behavior, options.Bounds.Size())
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlTreeView,
		behavior,
	)
	if err != nil {
		return nil, err
	}
	control := &TreeView{controlHandle: controlHandle{state: state}}
	state.control = control
	if options.MinimumSize == (Size{}) {
		state.autoMinimum = true
		state.minimumSize = behavior.intrinsicMinimum()
	}
	return control, nil
}

func newTreeViewBehavior(options TreeViewOptions) (treeViewBehavior, error) {
	scrollOptions := options.ScrollViewOptions
	if scrollOptions.ContentAutomationKey != "" ||
		scrollOptions.ContentStyle != "" ||
		scrollOptions.State.ContentSize != (Size{}) {
		return treeViewBehavior{}, fmt.Errorf(
			"%w: TreeView derives its content identity, style, and extent",
			ErrValidation,
		)
	}
	requestedOffset := scrollOptions.State.Offset
	if requestedOffset.X < 0 || requestedOffset.Y < 0 ||
		requestedOffset.X > maxCoordinateMagnitude ||
		requestedOffset.Y > maxCoordinateMagnitude {
		return treeViewBehavior{}, fmt.Errorf(
			"%w: invalid TreeView offset",
			ErrValidation,
		)
	}
	if err := validateOptionalCommand(scrollOptions.ChangeCommand); err != nil {
		return treeViewBehavior{}, err
	}
	if err := validateOptionalCommand(options.ActivateCommand); err != nil {
		return treeViewBehavior{}, err
	}
	if err := validateOptionalCommand(options.ExpandCommand); err != nil {
		return treeViewBehavior{}, err
	}
	scrollOptions.State.Offset = Point{}
	scroll, err := normalizeScrollViewBehavior(scrollViewBehavior{
		state:            scrollOptions.State,
		arrowStep:        scrollOptions.ArrowStep,
		pageStep:         scrollOptions.PageStep,
		disabled:         scrollOptions.Disabled,
		disabledReason:   scrollOptions.DisabledReason,
		horizontalPolicy: options.HorizontalBar,
		verticalPolicy:   options.VerticalBar,
		integratedBars:   true,
	})
	if err != nil {
		return treeViewBehavior{}, err
	}
	border, err := newBorderBehavior(
		"",
		options.BorderStyle,
		ControlTreeView,
		options.BorderForm,
		options.BorderForeground,
		options.BorderBackground,
	)
	if err != nil {
		return treeViewBehavior{}, err
	}
	scroll.border = border
	scroll.state.Offset = requestedOffset
	mode, err := normalizeCollectionSelectionMode(options.SelectionMode)
	if err != nil {
		return treeViewBehavior{}, err
	}
	status, message, err := normalizeCollectionStatus(
		options.Status,
		options.StatusMessage,
	)
	if err != nil {
		return treeViewBehavior{}, err
	}
	nodes, initialExpanded, err := normalizeTreeNodes(options.Nodes)
	if err != nil {
		return treeViewBehavior{}, err
	}
	behavior := treeViewBehavior{
		scroll:           scroll,
		nodes:            nodes,
		selectionMode:    mode,
		requireSelection: options.RequireSelection,
		status:           status,
		statusMessage:    message,
		changeCommand:    scrollOptions.ChangeCommand,
		activateCommand:  options.ActivateCommand,
		expandCommand:    options.ExpandCommand,
	}
	expanded := initialExpanded
	if options.Expanded != nil {
		expanded = options.Expanded
	}
	if err := setExactTreeIdentity(
		&behavior,
		options.Current,
		options.Selected,
		expanded,
	); err != nil {
		return treeViewBehavior{}, err
	}
	return behavior, nil
}

func normalizeTreeNodes(
	nodes []TreeNode,
) ([]normalizedTreeNode, []string, error) {
	type treeFrame struct {
		nodes  []TreeNode
		next   int
		parent int
		depth  int
		owner  int
	}
	result := make([]normalizedTreeNode, 0, min(len(nodes), MaxCollectionItems))
	expanded := make([]string, 0)
	seen := make(map[string]bool)
	stack := []treeFrame{{nodes: nodes, parent: -1, owner: -1}}
	for len(stack) > 0 {
		frame := &stack[len(stack)-1]
		if frame.next >= len(frame.nodes) {
			if frame.owner >= 0 {
				result[frame.owner].subtreeEnd = len(result)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if frame.depth >= MaxCollectionDepth {
			return nil, nil, fmt.Errorf(
				"%w: TreeView exceeds depth %d",
				ErrControlCapacity,
				MaxCollectionDepth,
			)
		}
		item := frame.nodes[frame.next]
		frame.next++
		if len(result) >= MaxCollectionItems {
			return nil, nil, fmt.Errorf(
				"%w: TreeView exceeds %d nodes",
				ErrControlCapacity,
				MaxCollectionItems,
			)
		}
		if !validBoundedIdentifier(item.Key) || seen[item.Key] {
			return nil, nil, fmt.Errorf(
				"%w: invalid or duplicate TreeView node key %q",
				ErrValidation,
				item.Key,
			)
		}
		seen[item.Key] = true
		label, err := normalizeDisplayText(item.Label, false)
		if err != nil || label.cells == 0 {
			return nil, nil, fmt.Errorf(
				"%w: invalid TreeView label for %q",
				ErrValidation,
				item.Key,
			)
		}
		reason, err := normalizeDisabledReason(item.Disabled, item.DisabledReason)
		if err != nil {
			return nil, nil, err
		}
		children := item.Children
		if item.Expanded && len(children) == 0 {
			return nil, nil, fmt.Errorf(
				"%w: TreeView leaf %q cannot be expanded",
				ErrValidation,
				item.Key,
			)
		}
		item.Label = label.text
		item.DisabledReason = reason
		item.Children = nil
		index := len(result)
		result = append(result, normalizedTreeNode{
			item: item, label: label, parent: frame.parent, depth: frame.depth,
			subtreeEnd: index + 1,
		})
		if frame.parent >= 0 {
			result[frame.parent].children = append(
				result[frame.parent].children,
				index,
			)
		}
		if item.Expanded {
			expanded = append(expanded, item.Key)
		}
		if len(children) > 0 {
			stack = append(stack, treeFrame{
				nodes: children, parent: index, depth: frame.depth + 1,
				owner: index,
			})
		}
	}
	return result, expanded, nil
}

func setExactTreeIdentity(
	behavior *treeViewBehavior,
	current string,
	selected []string,
	expanded []string,
) error {
	if behavior == nil {
		return ErrInvalidControl
	}
	orderedExpanded, err := normalizeTreeExpansion(behavior.nodes, expanded)
	if err != nil {
		return err
	}
	behavior.expanded = orderedExpanded
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	if current == "" {
		current = firstEnabledTreeKey(behavior.nodes, visible)
	} else {
		index := treeNodeIndex(behavior.nodes, current)
		if index < 0 || behavior.nodes[index].item.Disabled ||
			!treeIndexVisible(visible, index) {
			return fmt.Errorf("%w: invalid TreeView current key", ErrValidation)
		}
	}
	orderedSelected, err := normalizeTreeSelection(
		behavior.nodes,
		behavior.selectionMode,
		selected,
	)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(orderedSelected) == 0 && current != "" {
		orderedSelected = []string{current}
	}
	behavior.current = current
	behavior.selected = orderedSelected
	return nil
}

func normalizeTreeExpansion(
	nodes []normalizedTreeNode,
	expanded []string,
) ([]string, error) {
	requested := make(map[string]bool, len(expanded))
	for _, key := range expanded {
		if !validBoundedIdentifier(key) || requested[key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate TreeView expansion",
				ErrValidation,
			)
		}
		index := treeNodeIndex(nodes, key)
		if index < 0 || len(nodes[index].children) == 0 {
			return nil, fmt.Errorf(
				"%w: TreeView expansion identifies a leaf or missing node",
				ErrValidation,
			)
		}
		requested[key] = true
	}
	result := make([]string, 0, len(expanded))
	for _, node := range nodes {
		if requested[node.item.Key] {
			result = append(result, node.item.Key)
		}
	}
	return result, nil
}

func normalizeTreeSelection(
	nodes []normalizedTreeNode,
	mode CollectionSelectionMode,
	selected []string,
) ([]string, error) {
	if mode == CollectionSelectionSingle && len(selected) > 1 {
		return nil, fmt.Errorf(
			"%w: single-selection TreeView has multiple selected keys",
			ErrValidation,
		)
	}
	requested := make(map[string]bool, len(selected))
	for _, key := range selected {
		if !validBoundedIdentifier(key) || requested[key] {
			return nil, fmt.Errorf(
				"%w: invalid or duplicate TreeView selection",
				ErrValidation,
			)
		}
		index := treeNodeIndex(nodes, key)
		if index < 0 || nodes[index].item.Disabled {
			return nil, fmt.Errorf(
				"%w: TreeView selection identifies an unavailable node",
				ErrValidation,
			)
		}
		requested[key] = true
	}
	result := make([]string, 0, len(selected))
	for _, node := range nodes {
		if requested[node.item.Key] {
			result = append(result, node.item.Key)
		}
	}
	return result, nil
}

func treeNodeIndex(nodes []normalizedTreeNode, key string) int {
	for index := range nodes {
		if nodes[index].item.Key == key {
			return index
		}
	}
	return -1
}

func visibleTreeIndices(
	nodes []normalizedTreeNode,
	expanded []string,
) []int {
	result := make([]int, 0, len(nodes))
	for index := 0; index < len(nodes); {
		result = append(result, index)
		if len(nodes[index].children) > 0 &&
			!listSelectionContains(expanded, nodes[index].item.Key) {
			index = nodes[index].subtreeEnd
			continue
		}
		index++
	}
	return result
}

func treeIndexVisible(visible []int, nodeIndex int) bool {
	for _, index := range visible {
		if index == nodeIndex {
			return true
		}
	}
	return false
}

func treeVisiblePosition(visible []int, nodeIndex int) int {
	for position, index := range visible {
		if index == nodeIndex {
			return position
		}
	}
	return -1
}

func firstEnabledTreeKey(nodes []normalizedTreeNode, visible []int) string {
	for _, index := range visible {
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	return ""
}

func lastEnabledTreeKey(nodes []normalizedTreeNode, visible []int) string {
	for position := len(visible) - 1; position >= 0; position-- {
		index := visible[position]
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	return ""
}

func copyTreeNodes(
	nodes []normalizedTreeNode,
	expanded []string,
) []TreeNode {
	if len(nodes) == 0 {
		return nil
	}
	built := make([]TreeNode, len(nodes))
	for index := len(nodes) - 1; index >= 0; index-- {
		node := nodes[index].item
		node.Expanded = listSelectionContains(expanded, node.Key)
		node.Children = make([]TreeNode, len(nodes[index].children))
		for childIndex, child := range nodes[index].children {
			node.Children[childIndex] = built[child]
		}
		built[index] = node
	}
	result := make([]TreeNode, 0)
	for index := range nodes {
		if nodes[index].parent < 0 {
			result = append(result, built[index])
		}
	}
	return result
}

func cloneTreeViewBehavior(behavior treeViewBehavior) treeViewBehavior {
	cloned := behavior
	cloned.nodes = make([]normalizedTreeNode, len(behavior.nodes))
	for index, node := range behavior.nodes {
		cloned.nodes[index] = node
		cloned.nodes[index].label.lines = cloneTextRows(node.label.lines)
		cloned.nodes[index].children = append([]int(nil), node.children...)
	}
	cloned.selected = append([]string(nil), behavior.selected...)
	cloned.expanded = append([]string(nil), behavior.expanded...)
	cloned.statusMessage.lines = cloneTextRows(behavior.statusMessage.lines)
	return cloned
}

func treeNodeEqual(left, right normalizedTreeNode) bool {
	if left.item.Key != right.item.Key ||
		left.item.Label != right.item.Label ||
		left.item.Disabled != right.item.Disabled ||
		left.item.DisabledReason != right.item.DisabledReason ||
		left.parent != right.parent || left.depth != right.depth ||
		left.subtreeEnd != right.subtreeEnd ||
		len(left.children) != len(right.children) {
		return false
	}
	for index := range left.children {
		if left.children[index] != right.children[index] {
			return false
		}
	}
	return true
}

func treeViewBehaviorEqual(left, right treeViewBehavior) bool {
	if !scrollViewBehaviorEqual(left.scroll, right.scroll) ||
		left.current != right.current ||
		left.selectionMode != right.selectionMode ||
		left.requireSelection != right.requireSelection ||
		left.status != right.status ||
		left.statusMessage.text != right.statusMessage.text ||
		left.changeCommand != right.changeCommand ||
		left.activateCommand != right.activateCommand ||
		left.expandCommand != right.expandCommand ||
		len(left.nodes) != len(right.nodes) ||
		len(left.selected) != len(right.selected) ||
		len(left.expanded) != len(right.expanded) {
		return false
	}
	for index := range left.nodes {
		if !treeNodeEqual(left.nodes[index], right.nodes[index]) {
			return false
		}
	}
	for index := range left.selected {
		if left.selected[index] != right.selected[index] {
			return false
		}
	}
	for index := range left.expanded {
		if left.expanded[index] != right.expanded[index] {
			return false
		}
	}
	return true
}

func (b treeViewBehavior) controlBorder() borderBehavior { return b.scroll.border }

func (b treeViewBehavior) clientInset() int { return b.scroll.clientInset() }

func (b treeViewBehavior) controlClientRect(bounds Rect) Rect {
	return b.scroll.controlClientRect(bounds)
}

func (b treeViewBehavior) intrinsicMinimum() Size {
	return b.scroll.intrinsicMinimum()
}

func (b treeViewBehavior) additionalStyles() []StyleID {
	styles := append([]StyleID{}, b.scroll.additionalStyles()...)
	return append(styles,
		"collection.current",
		"collection.selected",
		"collection.current_selected",
		"collection.disabled",
		"collection.empty",
		"collection.loading",
		"collection.error",
		"tree.guide",
		"tree.branch",
		"tree.expanded",
	)
}

func (b treeViewBehavior) details() ControlDetails {
	details := b.scroll.border.details()
	details.Container = nil
	details.TreeView = &TreeViewDetails{}
	return details
}

func (b treeViewBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	b.scroll.paintDecoration(app, frame, state, absolute, clip)
	geometry := calculateScrollViewGeometry(absolute.Size(), b.scroll)
	viewport := translatedRect(geometry.viewport, absolute.X, absolute.Y)
	visible := viewport.Intersect(clip)
	for y := visible.Y; y < visible.Y+visible.Height; y++ {
		sourceY := b.scroll.state.Offset.Y + y - viewport.Y
		cells, exists := b.displayRow(sourceY, app.focus == state)
		if !exists {
			continue
		}
		for x := visible.X; x < visible.X+visible.Width; x++ {
			sourceX := b.scroll.state.Offset.X + x - viewport.X
			cell := treeRowCell{grapheme: " ", style: StyleID("tree_view")}
			if sourceX >= 0 && sourceX < len(cells) {
				cell = cells[sourceX]
			} else if len(cells) > 0 {
				cell.style = cells[len(cells)-1].style
			}
			app.setCellLocked(
				frame,
				x,
				y,
				cell.grapheme,
				cell.style,
				app.styles[cell.style],
				state.id,
			)
		}
	}
}

func (b treeViewBehavior) displayRow(
	visiblePosition int,
	focused bool,
) ([]treeRowCell, bool) {
	if visiblePosition != 0 &&
		(b.status != CollectionReady || len(b.nodes) == 0) {
		return nil, false
	}
	statusCells := func(prefix string, message normalizedDisplayText, style StyleID) []treeRowCell {
		plain := listStatusCells(prefix, message)
		result := make([]treeRowCell, len(plain))
		for index := range plain {
			result[index] = treeRowCell{grapheme: plain[index], style: style}
		}
		return result
	}
	switch b.status {
	case CollectionLoading:
		return statusCells("[loading] ", b.statusMessage, "collection.loading"), true
	case CollectionError:
		return statusCells("[error] ", b.statusMessage, "collection.error"), true
	}
	if len(b.nodes) == 0 {
		return statusCells("[empty]", normalizedDisplayText{}, "collection.empty"), true
	}
	visible := visibleTreeIndices(b.nodes, b.expanded)
	if visiblePosition < 0 || visiblePosition >= len(visible) {
		return nil, false
	}
	nodeIndex := visible[visiblePosition]
	node := b.nodes[nodeIndex]
	current := node.item.Key == b.current
	selected := listSelectionContains(b.selected, node.item.Key)
	baseStyle := StyleID("tree_view")
	if node.item.Disabled {
		baseStyle = "collection.disabled"
	} else if focused && current && selected {
		baseStyle = "collection.current_selected"
	} else if focused && current {
		baseStyle = "collection.current"
	} else if selected {
		baseStyle = "collection.selected"
	}
	structuralStyle := func(style StyleID) StyleID {
		if baseStyle == "tree_view" {
			return style
		}
		return baseStyle
	}
	result := make([]treeRowCell, 0, node.depth*2+12+node.label.cells)
	appendText := func(text string, style StyleID) {
		for _, grapheme := range text {
			result = append(result, treeRowCell{
				grapheme: string(grapheme), style: style,
			})
		}
	}
	ancestors := make([]int, 0, node.depth)
	for parent := node.parent; parent >= 0; parent = b.nodes[parent].parent {
		ancestors = append(ancestors, parent)
	}
	for index := len(ancestors) - 1; index >= 0; index-- {
		ancestor := ancestors[index]
		if treeNodeHasFollowingSibling(b.nodes, ancestor) {
			appendText("│ ", structuralStyle("tree.guide"))
		} else {
			appendText("  ", structuralStyle("tree.guide"))
		}
	}
	if node.depth > 0 {
		connector := "└─"
		if treeNodeHasFollowingSibling(b.nodes, nodeIndex) {
			connector = "├─"
		}
		appendText(connector, structuralStyle("tree.branch"))
	}
	if len(node.children) == 0 {
		appendText("    ", structuralStyle("tree.branch"))
	} else if listSelectionContains(b.expanded, node.item.Key) {
		appendText("[-] ", structuralStyle("tree.expanded"))
	} else {
		appendText("[+] ", structuralStyle("tree.branch"))
	}
	if focused && current {
		appendText("► ", baseStyle)
	} else {
		appendText("  ", baseStyle)
	}
	if selected {
		appendText("[X] ", baseStyle)
	} else {
		appendText("[ ] ", baseStyle)
	}
	for _, grapheme := range node.label.lines[0] {
		result = append(result, treeRowCell{grapheme: grapheme, style: baseStyle})
	}
	return result, true
}

func treeNodeHasFollowingSibling(nodes []normalizedTreeNode, index int) bool {
	if index < 0 || index >= len(nodes) {
		return false
	}
	parent := nodes[index].parent
	if parent < 0 {
		for next := index + 1; next < len(nodes); next++ {
			if nodes[next].parent < 0 {
				return true
			}
		}
		return false
	}
	children := nodes[parent].children
	for position, child := range children {
		if child == index {
			return position+1 < len(children)
		}
	}
	return false
}

func reflowTreeView(behavior treeViewBehavior, size Size) treeViewBehavior {
	height := 1
	if behavior.status == CollectionReady && len(behavior.nodes) > 0 {
		height = len(visibleTreeIndices(behavior.nodes, behavior.expanded))
	}
	maximum := 0
	for index := 0; index < height; index++ {
		cells, exists := behavior.displayRow(index, true)
		if exists {
			maximum = max(maximum, len(cells))
		}
	}
	behavior.scroll.state.ContentSize = Size{Width: maximum, Height: height}
	geometry := calculateScrollViewGeometry(size, behavior.scroll)
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	if behavior.status == CollectionReady {
		visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
		current := treeVisiblePosition(
			visible,
			treeNodeIndex(behavior.nodes, behavior.current),
		)
		if current >= 0 && geometry.viewport.Height > 0 {
			if current < behavior.scroll.state.Offset.Y {
				behavior.scroll.state.Offset.Y = current
			} else if current >= behavior.scroll.state.Offset.Y+
				geometry.viewport.Height {
				behavior.scroll.state.Offset.Y =
					current - geometry.viewport.Height + 1
			}
		}
	}
	behavior.scroll.state = clampViewportState(behavior.scroll.state, geometry)
	return behavior
}

func reconcileTreeViewLocked(state *controlState) bool {
	if state == nil {
		return false
	}
	behavior, ok := state.behavior.(treeViewBehavior)
	if !ok {
		return false
	}
	next := reflowTreeView(behavior, state.bounds.Size())
	if treeViewBehaviorEqual(behavior, next) {
		return false
	}
	state.behavior = next
	return true
}

func treeViewCanFocus(state *controlState, behavior treeViewBehavior) bool {
	if state == nil || behavior.scroll.disabled ||
		behavior.status != CollectionReady || behavior.current == "" {
		return false
	}
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	index := treeNodeIndex(behavior.nodes, behavior.current)
	return index >= 0 && !behavior.nodes[index].item.Disabled &&
		treeIndexVisible(visible, index)
}

func enabledVisibleTreeCount(behavior treeViewBehavior) int {
	count := 0
	for _, index := range visibleTreeIndices(behavior.nodes, behavior.expanded) {
		if !behavior.nodes[index].item.Disabled {
			count++
		}
	}
	return count
}

func treeViewDetails(bounds Rect, behavior treeViewBehavior) TreeViewDetails {
	viewport := scrollViewDetails(bounds, behavior.scroll)
	viewport.Content = ""
	viewport.ContentKey = ""
	firstSelected, lastSelected := firstAndLastKey(behavior.selected)
	firstExpanded, lastExpanded := firstAndLastKey(behavior.expanded)
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	return TreeViewDetails{
		Status:           behavior.status,
		StatusMessage:    behavior.statusMessage.text,
		NodeCount:        len(behavior.nodes),
		VisibleCount:     len(visible),
		EnabledCount:     enabledVisibleTreeCount(behavior),
		RetainedBytes:    treeViewStorageBytes(behavior),
		Current:          behavior.current,
		CurrentIndex:     treeVisiblePosition(visible, treeNodeIndex(behavior.nodes, behavior.current)),
		SelectionMode:    behavior.selectionMode,
		RequireSelection: behavior.requireSelection,
		SelectedCount:    len(behavior.selected),
		FirstSelected:    firstSelected,
		LastSelected:     lastSelected,
		SelectionDigest:  listSelectionDigest(behavior.selected),
		ExpandedCount:    len(behavior.expanded),
		FirstExpanded:    firstExpanded,
		LastExpanded:     lastExpanded,
		ExpansionDigest:  listSelectionDigest(behavior.expanded),
		Enabled:          !behavior.scroll.disabled,
		DisabledReason:   behavior.scroll.disabledReason,
		ChangeCommand:    behavior.changeCommand,
		ActivateCommand:  behavior.activateCommand,
		ExpandCommand:    behavior.expandCommand,
		Viewport:         viewport,
	}
}

func firstAndLastKey(keys []string) (string, string) {
	if len(keys) == 0 {
		return "", ""
	}
	return keys[0], keys[len(keys)-1]
}

// Nodes returns a caller-owned iterative deep copy of the complete tree.
func (v *TreeView) Nodes() []TreeNode {
	if v == nil || v.state == nil || v.state.app == nil {
		return nil
	}
	v.state.app.mu.RLock()
	defer v.state.app.mu.RUnlock()
	behavior, ok := v.state.behavior.(treeViewBehavior)
	if !ok || v.state.destroyed || v.state.aborted {
		return nil
	}
	return copyTreeNodes(behavior.nodes, behavior.expanded)
}

// State returns a caller-owned copy of the complete semantic state.
func (v *TreeView) State() TreeViewState {
	if v == nil || v.state == nil || v.state.app == nil {
		return TreeViewState{CurrentIndex: -1}
	}
	v.state.app.mu.RLock()
	defer v.state.app.mu.RUnlock()
	behavior, ok := v.state.behavior.(treeViewBehavior)
	if !ok || v.state.destroyed || v.state.aborted {
		return TreeViewState{CurrentIndex: -1}
	}
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	return TreeViewState{
		Status:        behavior.status,
		StatusMessage: behavior.statusMessage.text,
		Current:       behavior.current,
		CurrentIndex:  treeVisiblePosition(visible, treeNodeIndex(behavior.nodes, behavior.current)),
		Selected:      append([]string(nil), behavior.selected...),
		Expanded:      append([]string(nil), behavior.expanded...),
		Offset:        behavior.scroll.state.Offset,
		NodeCount:     len(behavior.nodes),
		VisibleCount:  len(visible),
		EnabledCount:  enabledVisibleTreeCount(behavior),
	}
}

// SetNodes replaces the copied model while preserving surviving identities.
func (v *TreeView) SetNodes(nodes []TreeNode) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeNodes(v, nodes)
	})
}

// Replace atomically replaces the model and exact requested identities.
func (v *TreeView) Replace(
	nodes []TreeNode,
	current string,
	selected []string,
	expanded []string,
) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.ReplaceTree(v, nodes, current, selected, expanded)
	})
}

// SetCurrent changes current by stable enabled visible node key.
func (v *TreeView) SetCurrent(current string) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeCurrent(v, current)
	})
}

// SetSelection changes selection by stable enabled node keys.
func (v *TreeView) SetSelection(selected []string) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeSelection(v, selected)
	})
}

// SetExpanded replaces the exact expanded-branch key set.
func (v *TreeView) SetExpanded(expanded []string) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeExpanded(v, expanded)
	})
}

// SetNodeExpanded expands or collapses one branch by stable key.
func (v *TreeView) SetNodeExpanded(key string, expanded bool) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeNodeExpanded(v, key, expanded)
	})
}

// SetStatus changes ready/loading/error presentation without discarding data.
func (v *TreeView) SetStatus(status CollectionStatus, message string) error {
	return commitTreeMutation(v, func(transaction *Transaction) error {
		return transaction.SetTreeStatus(v, status, message)
	})
}

func commitTreeMutation(
	control *TreeView,
	record func(*Transaction) error,
) error {
	if control == nil || control.state == nil || control.state.app == nil {
		return ErrInvalidControl
	}
	transaction := control.state.app.NewTransaction()
	if err := record(transaction); err != nil {
		return err
	}
	return transaction.Commit(context.Background())
}

// Focus atomically gives the TreeView keyboard focus when it has a ready,
// enabled visible current node.
func (v *TreeView) Focus() error { return focusSelectionControl(v) }

// Activate selects the current node when necessary and invokes the optional
// activation command through the ordinary command router.
func (v *TreeView) Activate(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if v == nil || v.state == nil || v.state.app == nil {
		return Completion{}, ErrInvalidControl
	}
	return v.state.app.activateTreeView(ctx, source, requestID, v.state)
}

func (a *App) activateTreeView(
	ctx context.Context,
	source string,
	requestID string,
	state *controlState,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) || !validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()

	a.mu.Lock()
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	if state == nil || state.app != a {
		a.mu.Unlock()
		return Completion{}, ErrInvalidControl
	}
	behavior, ok := state.behavior.(treeViewBehavior)
	if !ok || !treeViewCanFocus(state, behavior) {
		a.mu.Unlock()
		return Completion{}, ErrNotFocusable
	}
	command, target, _, changed := a.treeViewKeyLocked(state, KeyEnter)
	result := CommandResult{Outcome: OutcomeNoOp}
	var router CommandRouter
	var execute bool
	if changed {
		result.Outcome = OutcomeApplied
	}
	if command != "" {
		router, result, execute = a.resolveCommandLocked(command)
	}
	a.mu.Unlock()

	if execute {
		result = a.callRouter(ctx, router, Command{
			ID: command, Target: target, Source: source,
		})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	return a.associateLocked(requestID, result, command), nil
}

func (a *App) treeViewKeyLocked(
	state *controlState,
	key Key,
) (CommandID, ControlID, bool, bool) {
	if state == nil {
		return "", "", false, false
	}
	behavior, ok := state.behavior.(treeViewBehavior)
	if !ok || !treeViewCanFocus(state, behavior) {
		return "", "", false, false
	}
	before := cloneTreeViewBehavior(behavior)
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	nodeIndex := treeNodeIndex(behavior.nodes, behavior.current)
	position := treeVisiblePosition(visible, nodeIndex)
	geometry := calculateScrollViewGeometry(state.bounds.Size(), behavior.scroll)
	handled := true
	selectionChanged := false
	expansionChanged := false
	activate := false
	switch key {
	case KeyUp:
		behavior.current = adjacentEnabledTreeKey(behavior.nodes, visible, position, -1)
	case KeyDown:
		behavior.current = adjacentEnabledTreeKey(behavior.nodes, visible, position, 1)
	case KeyPageUp:
		behavior.current = pageEnabledTreeKey(
			behavior.nodes,
			visible,
			position,
			-effectiveScrollPageStep(
				behavior.scroll.pageStep.Height,
				geometry.viewport.Height,
			),
		)
	case KeyPageDown:
		behavior.current = pageEnabledTreeKey(
			behavior.nodes,
			visible,
			position,
			effectiveScrollPageStep(
				behavior.scroll.pageStep.Height,
				geometry.viewport.Height,
			),
		)
	case KeyHome:
		behavior.current = firstEnabledTreeKey(behavior.nodes, visible)
	case KeyEnd:
		behavior.current = lastEnabledTreeKey(behavior.nodes, visible)
	case KeyRight:
		if len(behavior.nodes[nodeIndex].children) > 0 &&
			!listSelectionContains(behavior.expanded, behavior.current) {
			expansionChanged = setTreeBranchExpanded(&behavior, nodeIndex, true)
		} else {
			behavior.current = firstEnabledTreeChildKey(behavior.nodes, nodeIndex)
		}
	case KeyLeft:
		if len(behavior.nodes[nodeIndex].children) > 0 &&
			listSelectionContains(behavior.expanded, behavior.current) {
			expansionChanged = setTreeBranchExpanded(&behavior, nodeIndex, false)
		} else {
			behavior.current = nearestEnabledTreeParentKey(behavior.nodes, nodeIndex)
		}
	case KeyPlus:
		expansionChanged = setTreeBranchExpanded(&behavior, nodeIndex, true)
	case KeyMinus:
		expansionChanged = setTreeBranchExpanded(&behavior, nodeIndex, false)
	case KeyAsterisk:
		expansionChanged = expandTreeBranchRecursive(&behavior, nodeIndex)
	case KeySpace:
		selectionChanged = selectCurrentTreeNode(&behavior, true)
	case KeyEnter:
		selectionChanged = selectCurrentTreeNode(&behavior, false)
		activate = true
	default:
		handled = false
	}
	if !handled {
		return "", "", false, false
	}
	behavior = reflowTreeView(behavior, state.bounds.Size())
	changed := !treeViewBehaviorEqual(before, behavior)
	if changed {
		state.behavior = behavior
	}
	command := CommandID("")
	if activate && behavior.activateCommand != "" {
		command = behavior.activateCommand
	} else if expansionChanged {
		command = behavior.expandCommand
	} else if selectionChanged {
		command = behavior.changeCommand
	}
	return command, state.id, true, changed
}

func adjacentEnabledTreeKey(
	nodes []normalizedTreeNode,
	visible []int,
	position int,
	delta int,
) string {
	for next := position + delta; next >= 0 && next < len(visible); next += delta {
		index := visible[next]
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	if position >= 0 && position < len(visible) {
		return nodes[visible[position]].item.Key
	}
	return firstEnabledTreeKey(nodes, visible)
}

func pageEnabledTreeKey(
	nodes []normalizedTreeNode,
	visible []int,
	position int,
	delta int,
) string {
	if len(visible) == 0 {
		return ""
	}
	target := min(len(visible)-1, max(0, position+delta))
	if delta < 0 {
		for next := target; next >= 0; next-- {
			index := visible[next]
			if !nodes[index].item.Disabled {
				return nodes[index].item.Key
			}
		}
		return adjacentEnabledTreeKey(nodes, visible, target-1, 1)
	}
	for next := target; next < len(visible); next++ {
		index := visible[next]
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	return adjacentEnabledTreeKey(nodes, visible, target+1, -1)
}

func firstEnabledTreeChildKey(nodes []normalizedTreeNode, parent int) string {
	if parent < 0 || parent >= len(nodes) {
		return ""
	}
	for _, child := range nodes[parent].children {
		if !nodes[child].item.Disabled {
			return nodes[child].item.Key
		}
	}
	return nodes[parent].item.Key
}

func nearestEnabledTreeParentKey(nodes []normalizedTreeNode, index int) string {
	if index < 0 || index >= len(nodes) {
		return ""
	}
	for parent := nodes[index].parent; parent >= 0; parent = nodes[parent].parent {
		if !nodes[parent].item.Disabled {
			return nodes[parent].item.Key
		}
	}
	return nodes[index].item.Key
}

func setTreeBranchExpanded(
	behavior *treeViewBehavior,
	index int,
	expanded bool,
) bool {
	if behavior == nil || index < 0 || index >= len(behavior.nodes) ||
		len(behavior.nodes[index].children) == 0 {
		return false
	}
	key := behavior.nodes[index].item.Key
	already := listSelectionContains(behavior.expanded, key)
	if already == expanded {
		return false
	}
	requested := append([]string(nil), behavior.expanded...)
	if expanded {
		requested = append(requested, key)
	} else {
		requested = removeKey(requested, key)
	}
	ordered, _ := normalizeTreeExpansion(behavior.nodes, requested)
	behavior.expanded = ordered
	return true
}

func expandTreeBranchRecursive(behavior *treeViewBehavior, index int) bool {
	if behavior == nil || index < 0 || index >= len(behavior.nodes) ||
		len(behavior.nodes[index].children) == 0 {
		return false
	}
	requested := append([]string(nil), behavior.expanded...)
	changed := false
	for next := index; next < behavior.nodes[index].subtreeEnd; next++ {
		if len(behavior.nodes[next].children) == 0 ||
			listSelectionContains(requested, behavior.nodes[next].item.Key) {
			continue
		}
		requested = append(requested, behavior.nodes[next].item.Key)
		changed = true
	}
	if changed {
		behavior.expanded, _ = normalizeTreeExpansion(behavior.nodes, requested)
	}
	return changed
}

func removeKey(keys []string, key string) []string {
	result := make([]string, 0, len(keys))
	for _, candidate := range keys {
		if candidate != key {
			result = append(result, candidate)
		}
	}
	return result
}

func selectCurrentTreeNode(behavior *treeViewBehavior, toggle bool) bool {
	if behavior == nil || behavior.current == "" {
		return false
	}
	selected := listSelectionContains(behavior.selected, behavior.current)
	if behavior.selectionMode == CollectionSelectionSingle {
		if selected && len(behavior.selected) == 1 {
			return false
		}
		behavior.selected = []string{behavior.current}
		return true
	}
	if selected {
		if !toggle || (behavior.requireSelection && len(behavior.selected) == 1) {
			return false
		}
		behavior.selected = removeKey(behavior.selected, behavior.current)
		return true
	}
	requested := append(append([]string(nil), behavior.selected...), behavior.current)
	behavior.selected, _ = normalizeTreeSelection(
		behavior.nodes,
		behavior.selectionMode,
		requested,
	)
	return true
}

func treeViewStorageBytes(behavior treeViewBehavior) int {
	total := len(behavior.statusMessage.text) + len(behavior.current)
	for _, node := range behavior.nodes {
		total += len(node.item.Key) + len(node.item.Label) +
			len(node.item.DisabledReason)
	}
	for _, key := range behavior.selected {
		total += len(key)
	}
	for _, key := range behavior.expanded {
		total += len(key)
	}
	return total
}

func (t *Transaction) selectedTreeView(
	control *TreeView,
) (*controlState, treeViewBehavior, error) {
	target, err := t.control(control)
	if err != nil {
		return nil, treeViewBehavior{}, err
	}
	behavior, ok := t.selectedControlBehavior(target).(treeViewBehavior)
	if !ok {
		return nil, treeViewBehavior{}, ErrInvalidControl
	}
	return target, cloneTreeViewBehavior(behavior), nil
}

func (t *Transaction) recordTreeView(
	state *controlState,
	behavior treeViewBehavior,
) error {
	if err := t.reserveOperation(); err != nil {
		return err
	}
	t.mutations = append(t.mutations, transactionMutation{
		kind: mutationTreeView, state: state, behavior: behavior,
	})
	return nil
}

// SetTreeNodes records a copied model replacement preserving stable state.
func (t *Transaction) SetTreeNodes(
	control *TreeView,
	nodes []TreeNode,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	normalized, defaults, err := normalizeTreeNodes(nodes)
	if err != nil {
		return err
	}
	oldVisible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	oldPosition := treeVisiblePosition(
		oldVisible,
		treeNodeIndex(behavior.nodes, behavior.current),
	)
	oldKeys := make(map[string]bool, len(behavior.nodes))
	for _, node := range behavior.nodes {
		oldKeys[node.item.Key] = true
	}
	requestedExpanded := make([]string, 0)
	for _, node := range normalized {
		if len(node.children) == 0 {
			continue
		}
		if oldKeys[node.item.Key] {
			if listSelectionContains(behavior.expanded, node.item.Key) {
				requestedExpanded = append(requestedExpanded, node.item.Key)
			}
		} else if listSelectionContains(defaults, node.item.Key) {
			requestedExpanded = append(requestedExpanded, node.item.Key)
		}
	}
	behavior.nodes = normalized
	behavior.expanded = requestedExpanded
	repairTreeIdentity(&behavior, oldPosition)
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// ReplaceTree records one exact model/current/selection/expansion replacement.
func (t *Transaction) ReplaceTree(
	control *TreeView,
	nodes []TreeNode,
	current string,
	selected []string,
	expanded []string,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	normalized, _, err := normalizeTreeNodes(nodes)
	if err != nil {
		return err
	}
	behavior.nodes = normalized
	if err := setExactTreeIdentity(&behavior, current, selected, expanded); err != nil {
		return err
	}
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// SetTreeCurrent records one stable enabled visible current key.
func (t *Transaction) SetTreeCurrent(
	control *TreeView,
	current string,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	if current == "" {
		if firstEnabledTreeKey(behavior.nodes, visible) != "" {
			return fmt.Errorf("%w: ready TreeView requires current", ErrValidation)
		}
	} else {
		index := treeNodeIndex(behavior.nodes, current)
		if index < 0 || behavior.nodes[index].item.Disabled ||
			!treeIndexVisible(visible, index) {
			return fmt.Errorf("%w: invalid TreeView current key", ErrValidation)
		}
	}
	behavior.current = current
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// SetTreeSelection records one exact stable-key selection.
func (t *Transaction) SetTreeSelection(
	control *TreeView,
	selected []string,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	ordered, err := normalizeTreeSelection(
		behavior.nodes,
		behavior.selectionMode,
		selected,
	)
	if err != nil {
		return err
	}
	if behavior.requireSelection && len(ordered) == 0 && behavior.current != "" {
		return fmt.Errorf("%w: TreeView selection is required", ErrValidation)
	}
	behavior.selected = ordered
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// SetTreeExpanded records one exact expanded-branch key set.
func (t *Transaction) SetTreeExpanded(
	control *TreeView,
	expanded []string,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	oldVisible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	oldPosition := treeVisiblePosition(
		oldVisible,
		treeNodeIndex(behavior.nodes, behavior.current),
	)
	ordered, err := normalizeTreeExpansion(behavior.nodes, expanded)
	if err != nil {
		return err
	}
	behavior.expanded = ordered
	repairTreeCurrent(&behavior, oldPosition)
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// SetTreeNodeExpanded records expansion or collapse of one branch key.
func (t *Transaction) SetTreeNodeExpanded(
	control *TreeView,
	key string,
	expanded bool,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	index := treeNodeIndex(behavior.nodes, key)
	if index < 0 || len(behavior.nodes[index].children) == 0 {
		return fmt.Errorf("%w: invalid TreeView branch key", ErrValidation)
	}
	oldVisible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	oldPosition := treeVisiblePosition(
		oldVisible,
		treeNodeIndex(behavior.nodes, behavior.current),
	)
	setTreeBranchExpanded(&behavior, index, expanded)
	repairTreeCurrent(&behavior, oldPosition)
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

// SetTreeStatus records ready/loading/error presentation state.
func (t *Transaction) SetTreeStatus(
	control *TreeView,
	status CollectionStatus,
	message string,
) error {
	target, behavior, err := t.selectedTreeView(control)
	if err != nil {
		return err
	}
	status, normalized, err := normalizeCollectionStatus(status, message)
	if err != nil {
		return err
	}
	behavior.status = status
	behavior.statusMessage = normalized
	behavior = reflowTreeView(behavior, target.bounds.Size())
	return t.recordTreeView(target, behavior)
}

func repairTreeIdentity(behavior *treeViewBehavior, oldPosition int) {
	if behavior == nil {
		return
	}
	repairTreeCurrent(behavior, oldPosition)
	selected := make([]string, 0, len(behavior.selected))
	for _, node := range behavior.nodes {
		if !node.item.Disabled &&
			listSelectionContains(behavior.selected, node.item.Key) {
			selected = append(selected, node.item.Key)
		}
	}
	if behavior.selectionMode == CollectionSelectionSingle && len(selected) > 1 {
		selected = selected[:1]
	}
	if behavior.requireSelection && len(selected) == 0 && behavior.current != "" {
		selected = []string{behavior.current}
	}
	behavior.selected = selected
}

func repairTreeCurrent(behavior *treeViewBehavior, oldPosition int) {
	if behavior == nil {
		return
	}
	visible := visibleTreeIndices(behavior.nodes, behavior.expanded)
	current := treeNodeIndex(behavior.nodes, behavior.current)
	if current >= 0 && !behavior.nodes[current].item.Disabled &&
		treeIndexVisible(visible, current) {
		return
	}
	if current >= 0 {
		for parent := behavior.nodes[current].parent; parent >= 0; parent = behavior.nodes[parent].parent {
			if !behavior.nodes[parent].item.Disabled &&
				treeIndexVisible(visible, parent) {
				behavior.current = behavior.nodes[parent].item.Key
				return
			}
		}
	}
	behavior.current = repairedTreeCurrent(behavior.nodes, visible, oldPosition)
}

func repairedTreeCurrent(
	nodes []normalizedTreeNode,
	visible []int,
	oldPosition int,
) string {
	if len(visible) == 0 {
		return ""
	}
	oldPosition = min(len(visible)-1, max(0, oldPosition))
	for position := oldPosition; position < len(visible); position++ {
		index := visible[position]
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	for position := oldPosition - 1; position >= 0; position-- {
		index := visible[position]
		if !nodes[index].item.Disabled {
			return nodes[index].item.Key
		}
	}
	return ""
}
