package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func tabbedDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) TabbedPanelDetails {
	t.Helper()
	details := controlByKey(t, app.Snapshot(), key).Details.TabbedPanel
	if details == nil {
		t.Fatalf("%s has no TabbedPanelDetails", key)
	}
	return *details
}

func effectivePageVisible(t *testing.T, app *App, page *Panel) bool {
	t.Helper()
	for _, control := range app.Snapshot().Controls {
		if control.ID == page.ID() {
			return control.Visible
		}
	}
	t.Fatalf("page %q missing from snapshot", page.ID())
	return false
}

func newTabbedFixture(
	t *testing.T,
	app *App,
	kind ControlKind,
	key string,
	bounds Rect,
	tabs func(*Transaction, Container) ([]Tab, error),
	selected string,
	changeCommand CommandID,
) (Control, []*Panel) {
	t.Helper()
	transaction := app.NewTransaction()
	var container Container
	var control Control
	var err error
	switch kind {
	case ControlTabbedPanel:
		value, createErr := transaction.NewTabbedPanel(
			app.Root(),
			TabbedPanelOptions{
				PanelOptions: PanelOptions{
					AutomationKey: key,
					Bounds:        bounds,
				},
				ChangeCommand: changeCommand,
			},
		)
		control, container, err = value, value, createErr
	case ControlNotebook:
		value, createErr := transaction.NewNotebook(
			app.Root(),
			TabbedPanelOptions{
				PanelOptions: PanelOptions{
					AutomationKey: key,
					Bounds:        bounds,
				},
				ChangeCommand: changeCommand,
			},
		)
		control, container, err = value, value, createErr
	default:
		t.Fatalf("unsupported fixture kind %q", kind)
	}
	if err != nil {
		t.Fatalf("new tab container error = %v", err)
	}
	descriptors, err := tabs(transaction, container)
	if err != nil {
		t.Fatalf("tab fixture pages error = %v", err)
	}
	if err := transaction.SetTabs(control, descriptors, selected); err != nil {
		t.Fatalf("SetTabs() error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	pages := make([]*Panel, len(descriptors))
	for index := range descriptors {
		pages[index] = descriptors[index].Page
	}
	return control, pages
}

func basicTabs(
	prefix string,
	disabledFirst bool,
) func(*Transaction, Container) ([]Tab, error) {
	return func(transaction *Transaction, container Container) ([]Tab, error) {
		labels := []string{"Alpha", "Beta", "Gamma"}
		mnemonics := []Key{"a", "b", "g"}
		tabs := make([]Tab, len(labels))
		for index := range labels {
			page, err := transaction.NewPanel(
				container,
				PanelOptions{
					AutomationKey: prefix + ".page." +
						string(rune('a'+index)),
					Style: StyleID([]string{
						"back", "nested", "front",
					}[index]),
				},
			)
			if err != nil {
				return nil, err
			}
			tabs[index] = Tab{
				Key:      prefix + ".tab." + string(rune('a'+index)),
				Value:    string(rune('a' + index)),
				Label:    labels[index],
				Mnemonic: mnemonics[index],
				Page:     page,
			}
		}
		if disabledFirst {
			tabs[0].Disabled = true
			tabs[0].DisabledReason = "Unavailable"
		}
		return tabs, nil
	}
}

func TestTabbedPanelAtomicConstructionPagesAndTypedRendering(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 12})
	control, pages := newTabbedFixture(
		t,
		app,
		ControlTabbedPanel,
		"tabs",
		Rect{X: 1, Y: 1, Width: 30, Height: 8},
		basicTabs("tabs", true),
		"",
		"",
	)
	panel := control.(*TabbedPanel)
	if panel.MinimumSize() != (Size{Width: 3, Height: 3}) {
		t.Fatalf("TabbedPanel minimum = %+v", panel.MinimumSize())
	}
	if panel.Selected() != "b" {
		t.Fatalf("default selected = %q, want b", panel.Selected())
	}
	if app.Focused() != panel {
		t.Fatalf("initial focus = %#v, want TabbedPanel", app.Focused())
	}
	for index, page := range pages {
		if got := page.Bounds(); got != (Rect{Width: 28, Height: 6}) {
			t.Fatalf("page %d bounds = %+v", index, got)
		}
		wantVisible := index == 1
		if effectivePageVisible(t, app, page) != wantVisible {
			t.Fatalf("page %d visible=%t, want %t",
				index, effectivePageVisible(t, app, page), wantVisible)
		}
	}
	details := tabbedDetailsByKey(t, app, "tabs")
	if details.Selected != "b" || details.Current != "b" ||
		len(details.Tabs) != 3 ||
		details.Tabs[0].Enabled ||
		details.Tabs[0].DisabledReason != "Unavailable" ||
		!details.Tabs[1].Selected || !details.Tabs[1].Current ||
		details.Tabs[1].Page != pages[1].ID() ||
		details.Tabs[1].PageKey != "tabs.page.b" {
		t.Fatalf("TabbedPanel details = %#v", details)
	}
	controlSnapshot := controlByKey(t, app.Snapshot(), "tabs")
	if controlSnapshot.Details.Container == nil ||
		controlSnapshot.Details.Container.ClientInset != 1 ||
		controlSnapshot.Details.Border == nil ||
		controlSnapshot.Details.Border.Form != BorderSingle {
		t.Fatalf("TabbedPanel container/border = %#v", controlSnapshot.Details)
	}
	selected := details.Tabs[1].Bounds
	cell, _ := app.Snapshot().Frame.Cell(
		controlSnapshot.AbsoluteBounds.X+selected.X,
		controlSnapshot.AbsoluteBounds.Y,
	)
	if cell.Style != "tab.focused" || cell.Grapheme != "└" {
		t.Fatalf("focused selected Tab cell = %#v", cell)
	}
}

func TestTabbedPanelKeyboardSelectionFocusRepairAndMnemonic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 50, Height: 12})
	registerActionCommand(t, app, "tab.changed", "Changed", true)
	control, pages := newTabbedFixture(
		t,
		app,
		ControlNotebook,
		"notebook",
		Rect{Width: 32, Height: 8},
		func(transaction *Transaction, container Container) ([]Tab, error) {
			tabs, err := basicTabs("notebook", false)(transaction, container)
			if err != nil {
				return nil, err
			}
			tabs[1].Disabled = true
			tabs[1].DisabledReason = "Disabled"
			return tabs, nil
		},
		"a",
		"tab.changed",
	)
	notebook := control.(*Notebook)
	child, err := NewCheckbox(pages[0], CheckboxOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "notebook.page.a.child",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		Label: "Child",
	})
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = notebook.Selected()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatal(err)
	}

	dispatchSelectionKey(t, app, "tab-right", KeyRight)
	details := tabbedDetailsByKey(t, app, "notebook")
	if details.Current != "c" || details.Selected != "a" ||
		!effectivePageVisible(t, app, pages[0]) ||
		effectivePageVisible(t, app, pages[2]) {
		t.Fatalf("Right changed selection or failed skip: %#v", details)
	}
	completion := dispatchSelectionKey(t, app, "tab-select-c", KeySpace)
	if completion.Command != "tab.changed" ||
		notebook.Selected() != "c" ||
		effectivePageVisible(t, app, pages[0]) ||
		!effectivePageVisible(t, app, pages[2]) {
		t.Fatalf("Space selection completion=%+v selected=%q",
			completion, notebook.Selected())
	}
	dispatchSelectionKey(t, app, "tab-left", KeyLeft)
	details = tabbedDetailsByKey(t, app, "notebook")
	if details.Current != "a" || details.Selected != "c" {
		t.Fatalf("Left focus/selection = %#v", details)
	}
	dispatchSelectionKey(t, app, "tab-select-a", KeyEnter)
	if notebook.Selected() != "a" {
		t.Fatalf("Enter selected = %q", notebook.Selected())
	}

	if err := child.Focus(); err != nil {
		t.Fatal(err)
	}
	if app.Focused() != child {
		t.Fatal("page child did not receive focus")
	}
	if err := notebook.SetSelected("c"); err != nil {
		t.Fatal(err)
	}
	if app.Focused() != notebook {
		t.Fatalf("hidden-page focus repaired to %#v, want Notebook", app.Focused())
	}

	if err := notebook.SetSelected("b"); err != nil {
		t.Fatal(err)
	}
	if notebook.Selected() != "b" ||
		!effectivePageVisible(t, app, pages[1]) {
		t.Fatalf("programmatic disabled selection = %q", notebook.Selected())
	}

	for _, event := range []KeyEvent{
		{Kind: KeyEventDown, Key: KeyAlt},
		{Kind: KeyEventPress, Key: "g"},
		{Kind: KeyEventUp, Key: KeyAlt},
	} {
		if _, err := app.DispatchKey(
			context.Background(),
			"tab-mnemonic",
			"tab-mnemonic-"+string(event.Kind),
			event,
		); err != nil {
			t.Fatalf("Alt-G event error = %v", err)
		}
	}
	if notebook.Selected() != "c" || app.Focused() != notebook {
		t.Fatalf("Alt-G selected=%q focus=%#v",
			notebook.Selected(), app.Focused())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(routed) != 3 {
		t.Fatalf("tab change command calls = %d, want 3", len(routed))
	}
}

func TestTabbedPanelReplacementReorderRemovalAndDestroyRepair(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 10})
	control, pages := newTabbedFixture(
		t,
		app,
		ControlTabbedPanel,
		"tabs",
		Rect{Width: 30, Height: 8},
		basicTabs("replace", false),
		"b",
		"",
	)
	panel := control.(*TabbedPanel)
	original := panel.Tabs()
	reordered := []Tab{original[2], original[0]}
	transaction := app.NewTransaction()
	if err := transaction.SetTabs(panel, reordered, ""); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetVisible(pages[1], false); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if panel.Selected() != "c" || len(panel.Tabs()) != 2 ||
		panel.Tabs()[0].Value != "c" ||
		effectivePageVisible(t, app, pages[1]) {
		t.Fatalf("replacement state tabs=%#v selected=%q removedVisible=%t",
			panel.Tabs(),
			panel.Selected(),
			effectivePageVisible(t, app, pages[1]),
		)
	}

	transaction = app.NewTransaction()
	if err := transaction.Destroy(pages[2]); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	tabs := panel.Tabs()
	if len(tabs) != 1 || tabs[0].Value != "a" ||
		panel.Selected() != "a" {
		t.Fatalf("destroy repair tabs=%#v selected=%q",
			tabs, panel.Selected())
	}
}

func TestTabbedPanelValidationAllDisabledClippingAndSnapshotCopies(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 10})
	transaction := app.NewTransaction()
	panel, err := transaction.NewTabbedPanel(
		app.Root(),
		TabbedPanelOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "tabs",
				Bounds:        Rect{Width: 9, Height: 6},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := transaction.NewPanel(panel, PanelOptions{
		AutomationKey: "page.first",
	})
	second, _ := transaction.NewPanel(panel, PanelOptions{
		AutomationKey: "page.second",
	})
	third, _ := transaction.NewPanel(panel, PanelOptions{
		AutomationKey: "page.third",
	})
	tabs := []Tab{
		{
			Key: "first", Value: "first", Label: "FirstLong",
			Mnemonic: "f", Page: first,
			Disabled: true, DisabledReason: "Disabled",
		},
		{
			Key: "second", Value: "second", Label: "SecondLong",
			Mnemonic: "s", Page: second,
			Disabled: true, DisabledReason: "Disabled",
		},
		{
			Key: "third", Value: "third", Label: "ThirdLong",
			Mnemonic: "t", Page: third,
			Disabled: true, DisabledReason: "Disabled",
		},
	}
	if err := transaction.SetTabs(panel, tabs, "second"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	details := tabbedDetailsByKey(t, app, "tabs")
	if details.Selected != "second" || details.Current != "first" ||
		details.LeadingOmitted || !details.TrailingOmitted ||
		details.Tabs[0].Omitted || details.Tabs[0].Bounds.Width == 0 {
		t.Fatalf("all-disabled clipped details = %#v", details)
	}
	if err := panel.Focus(); !errors.Is(err, ErrNotFocusable) {
		t.Fatalf("all-disabled Focus() error = %v", err)
	}

	firstSnapshot := app.Snapshot()
	firstDetails := controlByKey(
		t,
		firstSnapshot,
		"tabs",
	).Details.TabbedPanel
	firstDetails.Tabs[0].Label = "mutated"
	firstDetails.Selected = "mutated"
	current := tabbedDetailsByKey(t, app, "tabs")
	if current.Tabs[0].Label != "FirstLong" ||
		current.Selected != "second" {
		t.Fatal("snapshot mutation changed canonical Tab state")
	}
}

func TestTabbedPanelRejectsInvalidDescriptorsAndPageLayoutConflict(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 10})
	transaction := app.NewTransaction()
	panel, err := transaction.NewTabbedPanel(
		app.Root(),
		TabbedPanelOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := transaction.NewPanel(panel, PanelOptions{})
	second, _ := transaction.NewPanel(panel, PanelOptions{})
	outside, _ := transaction.NewPanel(app.Root(), PanelOptions{})
	base := []Tab{
		{Key: "a", Value: "a", Label: "Alpha", Mnemonic: "a", Page: first},
		{Key: "b", Value: "b", Label: "Beta", Mnemonic: "b", Page: second},
	}
	tests := []struct {
		name string
		tabs []Tab
	}{
		{
			name: "duplicate key",
			tabs: []Tab{
				base[0],
				{Key: "a", Value: "b", Label: "Beta", Page: second},
			},
		},
		{
			name: "duplicate value",
			tabs: []Tab{
				base[0],
				{Key: "b", Value: "a", Label: "Beta", Page: second},
			},
		},
		{
			name: "duplicate page",
			tabs: []Tab{
				base[0],
				{Key: "b", Value: "b", Label: "Beta", Page: first},
			},
		},
		{
			name: "outside page",
			tabs: []Tab{
				{Key: "a", Value: "a", Label: "Alpha", Page: outside},
			},
		},
		{
			name: "mnemonic missing from label",
			tabs: []Tab{
				{Key: "a", Value: "a", Label: "Alpha", Mnemonic: "z", Page: first},
			},
		},
	}
	for _, test := range tests {
		if err := transaction.SetTabs(
			panel,
			test.tabs,
			"",
		); err == nil {
			t.Fatalf("%s SetTabs() accepted invalid descriptors", test.name)
		}
	}
	if err := transaction.SetTabs(panel, base, "missing"); err == nil {
		t.Fatal("SetTabs() accepted missing selected value")
	}

	layout, err := NewBoxLayout(Horizontal, BoxLayoutOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(first, LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetLayout(panel, layout); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetTabs(panel, base, "a"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err == nil {
		t.Fatal("Commit() accepted a Layout-managed Tab page")
	}
}

func TestTabRenderPlanKeepsCurrentVisibleAtTinyWidths(t *testing.T) {
	t.Parallel()
	makeEntry := func(value, label string) tabEntry {
		normalized, err := normalizeDisplayText(label, false)
		if err != nil {
			t.Fatal(err)
		}
		return tabEntry{
			tab:   Tab{Value: value, Label: label},
			label: normalized,
		}
	}
	behavior := tabbedPanelBehavior{
		tabs: []tabEntry{
			makeEntry("a", "Alpha"),
			makeEntry("b", "Beta"),
			makeEntry("c", "Gamma"),
		},
		selected: "b",
		current:  "b",
	}
	for _, width := range []int{1, 2, 4, 8} {
		plan := buildTabRenderPlan(width, behavior)
		if plan.entries[1].omitted ||
			plan.entries[1].bounds.Width < 1 ||
			plan.entries[1].bounds.X < 0 ||
			plan.entries[1].bounds.X+
				plan.entries[1].bounds.Width > width ||
			!plan.leadingOmitted ||
			!plan.trailingOmitted {
			t.Fatalf("width %d plan = %#v", width, plan)
		}
	}
}

func TestTabbedPanelConcurrentSelectionAndNestedPageLayout(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 40, Height: 12})
	control, pages := newTabbedFixture(
		t,
		app,
		ControlNotebook,
		"notebook",
		Rect{Width: 30, Height: 8},
		basicTabs("concurrent", false),
		"a",
		"",
	)
	notebook := control.(*Notebook)
	child, err := NewPanel(pages[0], PanelOptions{
		AutomationKey: "nested.child",
		MinimumSize:   Size{Width: 2, Height: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	layout, err := NewBoxLayout(
		Horizontal,
		BoxLayoutOptions{AutomationKey: "nested.layout"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(child, LayoutItemOptions{Grow: 1}); err != nil {
		t.Fatal(err)
	}
	if err := pages[0].SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	if child.Bounds() != (Rect{Width: 28, Height: 6}) {
		t.Fatalf("nested page child bounds = %+v", child.Bounds())
	}

	const workers = 8
	var wait sync.WaitGroup
	failures := make(chan error, workers)
	values := []string{"a", "b", "c"}
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for attempt := 0; attempt < 20; attempt++ {
				err := notebook.SetSelected(
					values[(worker+attempt)%len(values)],
				)
				if err != nil && !errors.Is(err, ErrMutationBusy) {
					failures <- err
					return
				}
				_ = app.Snapshot()
			}
		}(worker)
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent SetSelected() error = %v", err)
	}
	found := false
	for _, tab := range notebook.Tabs() {
		if tab.Value == notebook.Selected() {
			found = true
		}
	}
	if !found {
		t.Fatalf("concurrent final selected = %q", notebook.Selected())
	}
}
