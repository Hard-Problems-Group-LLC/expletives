package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func treeViewDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) TreeViewDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.TreeView
	if details == nil {
		t.Fatalf("%s has no TreeViewDetails", key)
	}
	return *details
}

func dispatchTreeViewKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"tree-view-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func sampleTreeNodes() []TreeNode {
	return []TreeNode{
		{
			Key: "root", Label: "Root", Expanded: true,
			Children: []TreeNode{
				{Key: "blocked", Label: "Blocked", Disabled: true,
					DisabledReason: "Fixture policy"},
				{
					Key: "branch", Label: "Branch", Expanded: true,
					Children: []TreeNode{
						{Key: "leaf", Label: "Leaf"},
					},
				},
			},
		},
		{Key: "other", Label: "Other"},
	}
}

func TestTreeViewRenderingDetailsAndCopiedState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 42, Height: 12})
	nodes := sampleTreeNodes()
	tree, err := NewTreeView(app.Root(), TreeViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "collection.tree",
				Bounds:        Rect{X: 1, Y: 1, Width: 30, Height: 8},
			}},
			BorderForm: BorderSingle,
		},
		Nodes: nodes, SelectionMode: CollectionSelectionMultiple,
		Selected: []string{"root", "leaf"},
	})
	if err != nil {
		t.Fatalf("NewTreeView() error = %v", err)
	}
	nodes[0].Label = "caller mutation"
	nodes[0].Children[1].Children[0].Label = "caller leaf mutation"

	state := tree.State()
	if state.Current != "root" || state.CurrentIndex != 0 ||
		state.NodeCount != 5 || state.VisibleCount != 5 ||
		state.EnabledCount != 4 ||
		len(state.Selected) != 2 || len(state.Expanded) != 2 {
		t.Fatalf("TreeView State = %#v", state)
	}
	state.Selected[0] = "caller mutation"
	state.Expanded[0] = "caller mutation"
	retained := tree.Nodes()
	if retained[0].Label != "Root" ||
		retained[0].Children[1].Children[0].Label != "Leaf" ||
		tree.State().Selected[0] != "root" ||
		tree.State().Expanded[0] != "root" {
		t.Fatalf("caller-owned values alias retained state: %#v", retained)
	}
	retained[0].Children[1].Label = "another mutation"
	if got := tree.Nodes()[0].Children[1].Label; got != "Branch" {
		t.Fatalf("Nodes() alias changed retained label to %q", got)
	}

	snapshot := app.Snapshot()
	control := controlByKey(t, snapshot, "collection.tree")
	details := control.Details.TreeView
	if details == nil || details.Status != CollectionReady ||
		details.NodeCount != 5 || details.VisibleCount != 5 ||
		details.EnabledCount != 4 || details.Current != "root" ||
		details.CurrentIndex != 0 ||
		details.SelectionMode != CollectionSelectionMultiple ||
		details.SelectedCount != 2 || details.ExpandedCount != 2 ||
		details.FirstSelected != "root" || details.LastSelected != "leaf" ||
		details.FirstExpanded != "root" || details.LastExpanded != "branch" ||
		len(details.SelectionDigest) != 64 || len(details.ExpansionDigest) != 64 ||
		!details.Enabled || details.Viewport.Content != "" ||
		details.Viewport.ContentKey != "" {
		t.Fatalf("TreeViewDetails = %#v", details)
	}
	if control.Details.Container != nil || control.Details.Border == nil {
		t.Fatalf("TreeView union shape = %#v", control.Details)
	}
	if got := rowText(snapshot, 2); !strings.Contains(got, "[-] ► [X] Root") {
		t.Fatalf("root row = %q", got)
	}
	if got := rowText(snapshot, 4); !strings.Contains(got, "[-]   [ ] Branch") {
		t.Fatalf("branch row = %q", got)
	}
	if got := cellAt(t, snapshot, 12, 3); got.Style != "collection.disabled" {
		t.Fatalf("disabled row style = %#v", got)
	}
}

func TestTreeViewNavigationSelectionExpansionAndCommands(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 34, Height: 9})
	registerActionCommand(t, app, "tree.changed", "Tree changed", true)
	registerActionCommand(t, app, "tree.activate", "Activate node", true)
	registerActionCommand(t, app, "tree.expand", "Expand node", true)
	tree, err := NewTreeView(app.Root(), TreeViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "tree.commands",
					Bounds:        Rect{Width: 26, Height: 5},
				},
				ChangeCommand: "tree.changed",
			},
			BorderForm: BorderSingle,
		},
		Nodes: sampleTreeNodes(), SelectionMode: CollectionSelectionMultiple,
		RequireSelection: true, ActivateCommand: "tree.activate",
		ExpandCommand: "tree.expand",
	})
	if err != nil {
		t.Fatalf("NewTreeView() error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = tree.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	dispatchTreeViewKey(t, app, "tree-down", KeyDown)
	if state := tree.State(); state.Current != "branch" ||
		len(state.Selected) != 1 || state.Selected[0] != "root" {
		t.Fatalf("Down state = %#v", state)
	}
	dispatchTreeViewKey(t, app, "tree-right-child", KeyRight)
	if state := tree.State(); state.Current != "leaf" {
		t.Fatalf("Right to child state = %#v", state)
	}
	dispatchTreeViewKey(t, app, "tree-left-parent", KeyLeft)
	if state := tree.State(); state.Current != "branch" {
		t.Fatalf("Left to parent state = %#v", state)
	}
	collapse := dispatchTreeViewKey(t, app, "tree-collapse", KeyLeft)
	if state := tree.State(); state.VisibleCount != 4 ||
		listSelectionContains(state.Expanded, "branch") ||
		collapse.Command != "tree.expand" {
		t.Fatalf("collapse state=%#v completion=%#v", state, collapse)
	}
	expand := dispatchTreeViewKey(t, app, "tree-expand", KeyPlus)
	if state := tree.State(); state.VisibleCount != 5 ||
		!listSelectionContains(state.Expanded, "branch") ||
		expand.Command != "tree.expand" {
		t.Fatalf("expand state=%#v completion=%#v", state, expand)
	}
	space := dispatchTreeViewKey(t, app, "tree-space", KeySpace)
	if state := tree.State(); len(state.Selected) != 2 ||
		state.Selected[1] != "branch" || space.Command != "tree.changed" {
		t.Fatalf("Space state=%#v completion=%#v", state, space)
	}
	enter := dispatchTreeViewKey(t, app, "tree-enter", KeyEnter)
	if enter.Command != "tree.activate" {
		t.Fatalf("Enter completion = %#v", enter)
	}
	dispatchTreeViewKey(t, app, "tree-home", KeyHome)
	rootCollapse := dispatchTreeViewKey(t, app, "tree-root-collapse", KeyMinus)
	if rootCollapse.Command != "tree.expand" || tree.State().VisibleCount != 2 {
		t.Fatalf("root collapse state=%#v completion=%#v", tree.State(), rootCollapse)
	}
	recursive := dispatchTreeViewKey(t, app, "tree-recursive", KeyAsterisk)
	if recursive.Command != "tree.expand" || tree.State().VisibleCount != 5 {
		t.Fatalf("recursive expansion state=%#v completion=%#v", tree.State(), recursive)
	}

	activation, err := tree.Activate(
		context.Background(),
		"tree-view-test",
		"direct-activate",
	)
	if err != nil || activation.Command != "tree.activate" {
		t.Fatalf("Activate() = %#v, %v", activation, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(routed) != 7 || routed[0].ID != "tree.expand" ||
		routed[1].ID != "tree.expand" || routed[2].ID != "tree.changed" ||
		routed[3].ID != "tree.activate" || routed[4].ID != "tree.expand" ||
		routed[5].ID != "tree.expand" || routed[6].ID != "tree.activate" {
		t.Fatalf("routed commands = %#v", routed)
	}
}

func TestTreeViewStableIdentityRepairExpansionAndStatus(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 34, Height: 9})
	tree, err := NewTreeView(app.Root(), TreeViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "tree.repair",
				Bounds:        Rect{Width: 24, Height: 6},
			}},
		},
		Nodes: sampleTreeNodes(), Current: "leaf", Selected: []string{"leaf"},
		RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewTreeView() error = %v", err)
	}
	if err := tree.SetNodeExpanded("branch", false); err != nil {
		t.Fatalf("SetNodeExpanded(collapse) error = %v", err)
	}
	if state := tree.State(); state.Current != "branch" ||
		state.VisibleCount != 4 || len(state.Selected) != 1 ||
		state.Selected[0] != "leaf" {
		t.Fatalf("collapsed state = %#v", state)
	}
	if err := tree.SetNodes([]TreeNode{
		{Key: "other", Label: "Other moved"},
		{
			Key: "root", Label: "Root moved", Expanded: true,
			Children: []TreeNode{
				{
					Key: "branch", Label: "Branch moved", Expanded: true,
					Children: []TreeNode{{Key: "leaf", Label: "Leaf moved"}},
				},
			},
		},
	}); err != nil {
		t.Fatalf("SetNodes(reorder) error = %v", err)
	}
	if state := tree.State(); state.Current != "branch" ||
		listSelectionContains(state.Expanded, "branch") ||
		!listSelectionContains(state.Expanded, "root") ||
		len(state.Selected) != 1 || state.Selected[0] != "leaf" {
		t.Fatalf("reordered state = %#v", state)
	}
	if err := tree.SetNodes([]TreeNode{
		{Key: "other", Label: "Other"},
		{Key: "root", Label: "Root"},
	}); err != nil {
		t.Fatalf("SetNodes(remove current) error = %v", err)
	}
	if state := tree.State(); state.Current != "root" ||
		len(state.Selected) != 1 || state.Selected[0] != "root" {
		t.Fatalf("repaired state = %#v", state)
	}

	if err := tree.SetStatus(CollectionLoading, "Fetching nodes"); err != nil {
		t.Fatalf("SetStatus(loading) error = %v", err)
	}
	if app.Focused() == tree {
		t.Fatal("loading TreeView retained focus")
	}
	details := treeViewDetailsByKey(t, app, "tree.repair")
	if details.Status != CollectionLoading ||
		details.StatusMessage != "Fetching nodes" ||
		details.Viewport.State.ContentSize.Height != 1 {
		t.Fatalf("loading details = %#v", details)
	}
	if err := tree.SetStatus(CollectionError, ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetStatus(error without message) error = %v", err)
	}
	if err := tree.SetStatus(CollectionReady, ""); err != nil {
		t.Fatalf("SetStatus(ready) error = %v", err)
	}
	if err := tree.Focus(); err != nil {
		t.Fatalf("Focus() after ready error = %v", err)
	}
}

func TestTreeViewValidationDepthAndTransactionAtomicity(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 8})
	if _, err := NewTreeView(app.Root(), TreeViewOptions{
		Nodes: []TreeNode{
			{Key: "same", Label: "One"},
			{Key: "parent", Label: "Parent", Children: []TreeNode{
				{Key: "same", Label: "Two"},
			}},
		},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("global duplicate key error = %v", err)
	}
	if _, err := NewTreeView(app.Root(), TreeViewOptions{
		Nodes: []TreeNode{{Key: "leaf", Label: "Leaf", Expanded: true}},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expanded leaf error = %v", err)
	}
	deep := TreeNode{Key: "node-64", Label: "Leaf"}
	for depth := MaxCollectionDepth - 1; depth >= 0; depth-- {
		deep = TreeNode{
			Key: fmt.Sprintf("node-%d", depth), Label: "Branch",
			Children: []TreeNode{deep},
		}
	}
	if _, err := NewTreeView(app.Root(), TreeViewOptions{
		Nodes: []TreeNode{deep},
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("over-depth error = %v", err)
	}
	if _, err := NewTreeView(app.Root(), TreeViewOptions{
		Nodes:         []TreeNode{{Key: "one", Label: "One"}},
		ExpandCommand: "missing.command",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("unregistered expand command error = %v", err)
	}

	tree, err := NewTreeView(app.Root(), TreeViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "tree.atomic",
				Bounds:        Rect{Width: 18, Height: 5},
			}},
		},
		Nodes: []TreeNode{{Key: "one", Label: "One"}},
	})
	if err != nil {
		t.Fatalf("NewTreeView(valid) error = %v", err)
	}
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetTreeNodes(tree, []TreeNode{{Key: "two", Label: "Two"}}); err != nil {
		t.Fatalf("SetTreeNodes() error = %v", err)
	}
	if err := tx.SetTreeCurrent(tree, "missing"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetTreeCurrent(missing) error = %v", err)
	}
	if state := tree.State(); state.Current != "one" ||
		app.Snapshot().Sequence != before {
		t.Fatalf("uncommitted transaction leaked state=%#v", state)
	}
}

func TestTreeViewResourceBoundsAndConcurrentReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 26, Height: 8})
	tooMany := make([]TreeNode, MaxCollectionItems+1)
	for index := range tooMany {
		tooMany[index] = TreeNode{
			Key: fmt.Sprintf("node-%d", index), Label: "Node",
		}
	}
	if _, err := NewTreeView(app.Root(), TreeViewOptions{
		Nodes: tooMany,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("over-capacity node count error = %v", err)
	}

	tree, err := NewTreeView(app.Root(), TreeViewOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "tree.concurrent",
				Bounds:        Rect{Width: 18, Height: 5},
			}},
		},
		Nodes: []TreeNode{
			{Key: "root", Label: "Root", Expanded: true, Children: []TreeNode{
				{Key: "a", Label: "A"}, {Key: "b", Label: "B"},
			}},
		},
		Current: "a", Selected: []string{"a"}, RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewTreeView(concurrent) error = %v", err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 6)
	for worker := range 6 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := range 20 {
				first, second := "a", "b"
				if (worker+iteration)%2 != 0 {
					first, second = second, first
				}
				if err := tree.Replace(
					[]TreeNode{{
						Key: "root", Label: "Root", Children: []TreeNode{
							{Key: first, Label: strings.ToUpper(first)},
							{Key: second, Label: strings.ToUpper(second)},
						},
					}},
					first,
					[]string{first},
					[]string{"root"},
				); err != nil {
					errorsSeen <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent Replace() error = %v", err)
	}
	state := tree.State()
	nodes := tree.Nodes()
	if len(nodes) != 1 || len(nodes[0].Children) != 2 ||
		state.NodeCount != 3 || state.VisibleCount != 3 ||
		state.Current != nodes[0].Children[0].Key ||
		len(state.Selected) != 1 || state.Selected[0] != state.Current {
		t.Fatalf("concurrent final state=%#v nodes=%#v", state, nodes)
	}
}

func FuzzTreeNodeNormalization(f *testing.F) {
	for _, seed := range []struct {
		label    string
		reason   string
		disabled bool
	}{
		{"Node", "", false},
		{"e\u0301", "", false},
		{"wide界", "", false},
		{string([]byte{0xff}), "", false},
		{"Disabled", "policy", true},
	} {
		f.Add(seed.label, seed.reason, seed.disabled)
	}
	f.Fuzz(func(t *testing.T, label string, reason string, disabled bool) {
		if len(label)+len(reason) > 4096 {
			t.Skip()
		}
		nodes, expanded, err := normalizeTreeNodes([]TreeNode{{
			Key: "node", Label: label,
			Disabled: disabled, DisabledReason: reason,
		}})
		if err != nil {
			return
		}
		if len(nodes) != 1 || len(expanded) != 0 ||
			nodes[0].item.Key != "node" || nodes[0].item.Label == "" ||
			!utf8.ValidString(nodes[0].item.Label) ||
			strings.Contains(nodes[0].item.Label, "\n") ||
			(disabled && nodes[0].item.DisabledReason == "") ||
			(!disabled && nodes[0].item.DisabledReason != "") {
			t.Fatalf("normalized TreeNode = %#v", nodes[0].item)
		}
	})
}
