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

func listBoxDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) ListBoxDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.ListBox
	if details == nil {
		t.Fatalf("%s has no ListBoxDetails", key)
	}
	return *details
}

func dispatchListBoxKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"list-box-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func TestListBoxRenderingDetailsAndCopiedState(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 10})
	items := []ListItem{
		{Key: "alpha", Label: "Alpha", Description: "first"},
		{Key: "blocked", Label: "Blocked", Disabled: true,
			DisabledReason: "Fixture policy"},
		{Key: "charlie", Label: "Charlie"},
	}
	list, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "collection.list",
				Bounds:        Rect{X: 1, Y: 1, Width: 20, Height: 6},
			}},
			BorderForm: BorderSingle,
		},
		Items: items, SelectionMode: CollectionSelectionMultiple,
		Selected: []string{"alpha"},
	})
	if err != nil {
		t.Fatalf("NewListBox() error = %v", err)
	}
	items[0].Label = "caller mutation"

	state := list.State()
	if state.Current != "alpha" || state.CurrentIndex != 0 ||
		len(state.Selected) != 1 || state.Selected[0] != "alpha" ||
		state.ItemCount != 3 || state.EnabledCount != 2 {
		t.Fatalf("ListBox State = %#v", state)
	}
	state.Selected[0] = "caller mutation"
	retained := list.Items()
	if retained[0].Label != "Alpha" || list.State().Selected[0] != "alpha" {
		t.Fatalf("caller-owned values alias retained state: %#v", retained)
	}
	retained[0].Label = "another mutation"
	if got := list.Items()[0].Label; got != "Alpha" {
		t.Fatalf("Items() alias changed retained label to %q", got)
	}

	snapshot := app.Snapshot()
	control := controlByKey(t, snapshot, "collection.list")
	details := control.Details.ListBox
	if details == nil || details.Status != CollectionReady ||
		details.ItemCount != 3 || details.EnabledCount != 2 ||
		details.Current != "alpha" || details.CurrentIndex != 0 ||
		details.SelectionMode != CollectionSelectionMultiple ||
		details.SelectedCount != 1 ||
		details.FirstSelected != "alpha" ||
		details.LastSelected != "alpha" ||
		len(details.SelectionDigest) != 64 || !details.Enabled ||
		details.Viewport.Content != "" || details.Viewport.ContentKey != "" {
		t.Fatalf("ListBoxDetails = %#v", details)
	}
	if control.Details.Container != nil || control.Details.Border == nil {
		t.Fatalf("ListBox union shape = %#v", control.Details)
	}
	if got := rowText(snapshot, 2); !strings.Contains(got, "► [X] Alpha") {
		t.Fatalf("focused selected row = %q", got)
	}
	if got := cellAt(t, snapshot, 2, 3); got.Style != "collection.disabled" {
		t.Fatalf("disabled row style = %#v", got)
	}
}

func TestListBoxNavigationSelectionActivationAndCommands(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 8})
	registerActionCommand(t, app, "list.changed", "List changed", true)
	registerActionCommand(t, app, "list.current", "List current", true)
	registerActionCommand(t, app, "list.activate", "Activate item", true)
	list, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: "list.commands",
					Bounds:        Rect{Width: 16, Height: 4},
				},
				ChangeCommand: "list.changed",
			},
			BorderForm: BorderSingle,
		},
		Items: []ListItem{
			{Key: "one", Label: "One"},
			{Key: "disabled", Label: "Disabled", Disabled: true,
				DisabledReason: "Skip this"},
			{Key: "three", Label: "Three"},
			{Key: "four", Label: "Four"},
		},
		SelectionMode:    CollectionSelectionMultiple,
		RequireSelection: true,
		CurrentCommand:   "list.current",
		ActivateCommand:  "list.activate",
	})
	if err != nil {
		t.Fatalf("NewListBox() error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = list.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	down := dispatchListBoxKey(t, app, "list-down", KeyDown)
	if state := list.State(); state.Current != "three" ||
		len(state.Selected) != 1 || state.Selected[0] != "one" ||
		down.Command != "list.current" {
		t.Fatalf("Down state=%#v completion=%#v", state, down)
	}
	space := dispatchListBoxKey(t, app, "list-space", KeySpace)
	if state := list.State(); len(state.Selected) != 2 ||
		state.Selected[1] != "three" || space.Command != "list.changed" {
		t.Fatalf("Space state=%#v completion=%#v", state, space)
	}
	dispatchListBoxKey(t, app, "list-end", KeyEnd)
	enter := dispatchListBoxKey(t, app, "list-enter", KeyEnter)
	if state := list.State(); state.Current != "four" ||
		len(state.Selected) != 3 || enter.Command != "list.activate" {
		t.Fatalf("Enter state=%#v completion=%#v", state, enter)
	}
	if got := list.State().Offset.Y; got == 0 {
		t.Fatalf("current item was not kept visible; offset=%d", got)
	}

	activation, err := list.Activate(
		context.Background(),
		"list-box-test",
		"direct-activate",
	)
	if err != nil || activation.Command != "list.activate" {
		t.Fatalf("Activate() = %#v, %v", activation, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(routed) != 5 || routed[0].ID != "list.current" ||
		routed[1].ID != "list.changed" || routed[2].ID != "list.current" ||
		routed[3].ID != "list.activate" || routed[4].ID != "list.activate" {
		t.Fatalf("routed commands = %#v", routed)
	}
}

func TestListBoxStableIdentityRepairAndStatus(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	list, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "list.repair",
				Bounds:        Rect{Width: 18, Height: 5},
			}},
		},
		Items: []ListItem{
			{Key: "a", Label: "A"},
			{Key: "b", Label: "B"},
			{Key: "c", Label: "C"},
		},
		Current: "b", Selected: []string{"b"}, RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewListBox() error = %v", err)
	}
	if err := list.SetItems([]ListItem{
		{Key: "c", Label: "C moved"},
		{Key: "b", Label: "B moved"},
		{Key: "a", Label: "A moved"},
	}); err != nil {
		t.Fatalf("SetItems(reorder) error = %v", err)
	}
	if state := list.State(); state.Current != "b" ||
		len(state.Selected) != 1 || state.Selected[0] != "b" ||
		state.CurrentIndex != 1 {
		t.Fatalf("reordered state = %#v", state)
	}
	if err := list.SetItems([]ListItem{
		{Key: "c", Label: "C"},
		{Key: "a", Label: "A"},
	}); err != nil {
		t.Fatalf("SetItems(remove current) error = %v", err)
	}
	if state := list.State(); state.Current != "a" ||
		len(state.Selected) != 1 || state.Selected[0] != "a" {
		t.Fatalf("repaired state = %#v", state)
	}

	if err := list.SetStatus(CollectionLoading, "Fetching rows"); err != nil {
		t.Fatalf("SetStatus(loading) error = %v", err)
	}
	if app.Focused() == list {
		t.Fatal("loading ListBox retained focus")
	}
	details := listBoxDetailsByKey(t, app, "list.repair")
	if details.Status != CollectionLoading ||
		details.StatusMessage != "Fetching rows" ||
		details.Viewport.State.ContentSize.Height != 1 {
		t.Fatalf("loading details = %#v", details)
	}
	if err := list.SetStatus(CollectionError, ""); !errors.Is(
		err,
		ErrValidation,
	) {
		t.Fatalf("SetStatus(error without message) error = %v", err)
	}
	if err := list.SetStatus(CollectionReady, ""); err != nil {
		t.Fatalf("SetStatus(ready) error = %v", err)
	}
	if err := list.Focus(); err != nil {
		t.Fatalf("Focus() after ready error = %v", err)
	}
}

func TestListBoxValidationAndTransactionAtomicity(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 6})
	if _, err := NewListBox(app.Root(), ListBoxOptions{
		Items: []ListItem{{Key: "same", Label: "One"},
			{Key: "same", Label: "Two"}},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("duplicate item key error = %v", err)
	}
	if _, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				ChangeCommand: "missing.command",
			},
		},
		Items: []ListItem{{Key: "one", Label: "One"}},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("unregistered change command error = %v", err)
	}
	list, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "list.atomic",
				Bounds:        Rect{Width: 12, Height: 3},
			}},
		},
		Items: []ListItem{{Key: "one", Label: "One"}},
	})
	if err != nil {
		t.Fatalf("NewListBox(valid) error = %v", err)
	}
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetListItems(list, []ListItem{{Key: "two", Label: "Two"}}); err != nil {
		t.Fatalf("SetListItems() error = %v", err)
	}
	if err := tx.SetListCurrent(list, "missing"); !errors.Is(
		err,
		ErrValidation,
	) {
		t.Fatalf("SetListCurrent(missing) error = %v", err)
	}
	if state := list.State(); state.Current != "one" ||
		app.Snapshot().Sequence != before {
		t.Fatalf("uncommitted transaction leaked state=%#v", state)
	}
}

func TestListBoxResourceBoundsAndConcurrentReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 8})
	tooMany := make([]ListItem, MaxCollectionItems+1)
	for index := range tooMany {
		tooMany[index] = ListItem{
			Key: fmt.Sprintf("item-%d", index), Label: "item",
		}
	}
	if _, err := NewListBox(app.Root(), ListBoxOptions{
		Items: tooMany,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("over-capacity item count error = %v", err)
	}

	tooLarge := make([]ListItem, MaxCollectionItems)
	largeLabel := strings.Repeat("x", MaxDisplayTextCells)
	for index := range tooLarge {
		tooLarge[index] = ListItem{
			Key: fmt.Sprintf("large-%d", index), Label: largeLabel,
		}
	}
	if _, err := NewListBox(app.Root(), ListBoxOptions{
		Items: tooLarge,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("over-capacity aggregate bytes error = %v", err)
	}

	list, err := NewListBox(app.Root(), ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{PanelOptions: PanelOptions{
				AutomationKey: "list.concurrent",
				Bounds:        Rect{Width: 16, Height: 4},
			}},
		},
		Items:            []ListItem{{Key: "a", Label: "A"}, {Key: "b", Label: "B"}},
		RequireSelection: true,
	})
	if err != nil {
		t.Fatalf("NewListBox(concurrent) error = %v", err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 8)
	for worker := range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := range 25 {
				first := "a"
				second := "b"
				if (worker+iteration)%2 != 0 {
					first, second = second, first
				}
				if err := list.Replace(
					[]ListItem{
						{Key: first, Label: strings.ToUpper(first)},
						{Key: second, Label: strings.ToUpper(second)},
					},
					first,
					[]string{first},
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
	state := list.State()
	items := list.Items()
	if len(items) != 2 || state.ItemCount != 2 || state.EnabledCount != 2 ||
		state.Current != items[0].Key || len(state.Selected) != 1 ||
		state.Selected[0] != state.Current {
		t.Fatalf("concurrent final state=%#v items=%#v", state, items)
	}
}

func FuzzListBoxItemNormalization(f *testing.F) {
	for _, seed := range []struct {
		label       string
		description string
		reason      string
		disabled    bool
	}{
		{"Alpha", "first", "", false},
		{"e\u0301", "composed", "", false},
		{"wide界", "", "", false},
		{string([]byte{0xff}), "", "", false},
		{"Disabled", "", "policy", true},
	} {
		f.Add(seed.label, seed.description, seed.reason, seed.disabled)
	}
	f.Fuzz(func(
		t *testing.T,
		label string,
		description string,
		reason string,
		disabled bool,
	) {
		if len(label)+len(description)+len(reason) > 4096 {
			t.Skip()
		}
		items, err := normalizeListItems([]ListItem{{
			Key: "item", Label: label, Description: description,
			Disabled: disabled, DisabledReason: reason,
		}})
		if err != nil {
			return
		}
		if len(items) != 1 || items[0].item.Key != "item" ||
			items[0].item.Label == "" ||
			!utf8.ValidString(items[0].item.Label) ||
			strings.Contains(items[0].item.Label, "\n") ||
			strings.Contains(items[0].item.Description, "\n") ||
			(disabled && items[0].item.DisabledReason == "") ||
			(!disabled && items[0].item.DisabledReason != "") {
			t.Fatalf("normalized ListItem = %#v", items[0].item)
		}
	})
}
