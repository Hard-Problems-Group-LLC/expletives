package expletives

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func dispatchPopupKey(
	t *testing.T,
	app *App,
	request string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"popup-collection-test",
		request,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s) error = %v", key, err)
	}
	return completion
}

func dropDownDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) DropDownDetails {
	t.Helper()
	control := controlByKey(t, app.Snapshot(), key)
	if control.Details.DropDown == nil {
		t.Fatalf("%s has no DropDownDetails", key)
	}
	return *control.Details.DropDown
}

func comboBoxDetailsByKey(
	t *testing.T,
	app *App,
	key string,
) ComboBoxDetails {
	t.Helper()
	control := controlByKey(t, app.Snapshot(), key)
	if control.Details.ComboBox == nil {
		t.Fatalf("%s has no ComboBoxDetails", key)
	}
	return *control.Details.ComboBox
}

func TestDropDownCopiedModelRenderingAndClampedPopup(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	items := []ListItem{
		{Key: "alpha", Label: "Alpha"},
		{Key: "blocked", Label: "Blocked", Disabled: true,
			DisabledReason: "Fixture policy"},
		{Key: "charlie", Label: "A deliberately long Charlie label"},
	}
	dropDown, err := NewDropDown(app.Root(), DropDownOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "drop.copied",
			Bounds:        Rect{X: 25, Y: 5, Width: 5, Height: 1},
		},
		Items: items, PopupRows: 3,
	})
	if err != nil {
		t.Fatalf("NewDropDown() error = %v", err)
	}
	items[0].Label = "caller mutation"
	retained := dropDown.Items()
	if retained[0].Label != "Alpha" {
		t.Fatalf("Items()[0] = %#v", retained[0])
	}
	retained[0].Label = "second mutation"
	if got := dropDown.Items()[0].Label; got != "Alpha" {
		t.Fatalf("caller-owned Items() changed retained label to %q", got)
	}
	state := dropDown.State()
	if state.Current != "alpha" || state.Selected != "alpha" ||
		state.CurrentIndex != 0 || state.SelectedIndex != 0 ||
		state.ItemCount != 3 || state.EnabledCount != 2 {
		t.Fatalf("initial State() = %#v", state)
	}
	snapshot := app.Snapshot()
	control := controlByKey(t, snapshot, "drop.copied")
	if control.Details.DropDown == nil || control.Details.ComboBox != nil ||
		control.Details.DropDown.RetainedBytes == 0 {
		t.Fatalf("DropDown union/details = %#v", control.Details)
	}
	if got := cellAt(t, snapshot, 29, 5); got.Grapheme != "▼" ||
		got.Owner != dropDown.ID() {
		t.Fatalf("collapsed arrow cell = %#v", got)
	}

	if err := dropDown.Open(); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	details := dropDownDetailsByKey(t, app, "drop.copied")
	app.mu.RLock()
	client := app.applicationContentRectLocked()
	app.mu.RUnlock()
	if !details.Open || details.PopupCurrent != "alpha" ||
		details.PopupSelection != "alpha" || details.PopupBounds.Empty() ||
		details.PopupBounds.X < client.X ||
		details.PopupBounds.Y < client.Y ||
		details.PopupBounds.X+details.PopupBounds.Width > client.X+client.Width ||
		details.PopupBounds.Y+details.PopupBounds.Height > client.Y+client.Height ||
		details.PopupBounds.X >= 25 || details.PopupBounds.Y >= 5 {
		t.Fatalf("clamped/backset popup details = %#v, client=%#v", details, client)
	}
	popupCell := cellAt(
		t,
		app.Snapshot(),
		details.PopupBounds.X,
		details.PopupBounds.Y,
	)
	if popupCell.Grapheme != "┌" || popupCell.Owner != dropDown.ID() {
		t.Fatalf("popup border cell = %#v", popupCell)
	}
	if err := dropDown.Close(); err != nil || dropDown.State().Open {
		t.Fatalf("Close() error/state = %v/%#v", err, dropDown.State())
	}
}

func TestPopupFieldsPreserveOneRowMinimumWithWidthOverride(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 2})
	dropDown, err := NewDropDown(app.Root(), DropDownOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "popup-minimum.dropdown",
			MinimumSize:   Size{Width: 12},
		},
		Items:    []ListItem{{Key: "alpha", Label: "Alpha"}},
		Selected: "alpha",
	})
	if err != nil {
		t.Fatalf("NewDropDown() error = %v", err)
	}
	comboBox, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "popup-minimum.combo",
				MinimumSize:   Size{Width: 14},
			},
			Items:    []ListItem{{Key: "bravo", Label: "Bravo"}},
			Selected: "bravo",
		},
	})
	if err != nil {
		t.Fatalf("NewComboBox() error = %v", err)
	}
	if got, want := dropDown.MinimumSize(), (Size{Width: 12, Height: 1}); got != want {
		t.Fatalf("DropDown minimum = %+v, want %+v", got, want)
	}
	if got, want := comboBox.MinimumSize(), (Size{Width: 14, Height: 1}); got != want {
		t.Fatalf("ComboBox minimum = %+v, want %+v", got, want)
	}

	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: "popup-minimum.layout",
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(dropDown, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(dropDown) error = %v", err)
	}
	if err := layout.AddPanel(comboBox, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(comboBox) error = %v", err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatalf("SetLayout() error = %v", err)
	}
	if got := dropDown.Bounds(); got.Height != 1 {
		t.Fatalf("DropDown bounds = %+v, want one row", got)
	}
	if got := comboBox.Bounds(); got.Height != 1 {
		t.Fatalf("ComboBox bounds = %+v, want one row", got)
	}

	snapshot := app.Snapshot()
	for _, want := range []struct {
		x, y     int
		grapheme string
		owner    ControlID
	}{
		{x: 0, y: 0, grapheme: "A", owner: dropDown.ID()},
		{x: 0, y: 1, grapheme: "B", owner: comboBox.ID()},
	} {
		got := cellAt(t, snapshot, want.x, want.y)
		if got.Grapheme != want.grapheme || got.Owner != want.owner {
			t.Fatalf("painted popup cell at (%d,%d) = %+v", want.x, want.y, got)
		}
		if got.Foreground == got.Background {
			t.Fatalf("popup cell at (%d,%d) has no color contrast: %+v", want.x, want.y, got)
		}
	}
}

func TestDropDownPopupNavigationRollbackActivationAndTab(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 8})
	registerActionCommand(t, app, "drop.changed", "Changed", true)
	registerActionCommand(t, app, "drop.activate", "Activate", true)
	first, err := NewDropDown(app.Root(), DropDownOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "drop.first",
			Bounds:        Rect{Width: 12, Height: 1},
		},
		Items: []ListItem{
			{Key: "one", Label: "One"},
			{Key: "disabled", Label: "Disabled", Disabled: true,
				DisabledReason: "Skip"},
			{Key: "three", Label: "Three"},
			{Key: "four", Label: "Four"},
		},
		AllowEmpty:    true,
		Selected:      "one",
		ChangeCommand: "drop.changed", ActivateCommand: "drop.activate",
	})
	if err != nil {
		t.Fatalf("NewDropDown(first) error = %v", err)
	}
	second, err := NewDropDown(app.Root(), DropDownOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "drop.second",
			Bounds:        Rect{Y: 2, Width: 12, Height: 1},
		},
		Items: []ListItem{{Key: "other", Label: "Other"}},
	})
	if err != nil {
		t.Fatalf("NewDropDown(second) error = %v", err)
	}
	var mu sync.Mutex
	var routed []Command
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		_ = first.State()
		mu.Lock()
		routed = append(routed, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}

	dispatchPopupKey(t, app, "open", KeySpace)
	dispatchPopupKey(t, app, "down", KeyDown)
	if state := first.State(); state.PopupCurrent != "three" ||
		state.PopupSelection != "one" || state.Selected != "one" {
		t.Fatalf("provisional navigation State() = %#v", state)
	}
	dispatchPopupKey(t, app, "select", KeySpace)
	dispatchPopupKey(t, app, "cancel", KeyEscape)
	if state := first.State(); state.Open || state.Current != "one" ||
		state.Selected != "one" {
		t.Fatalf("Escape rollback State() = %#v", state)
	}

	dispatchPopupKey(t, app, "reopen", KeyF4)
	dispatchPopupKey(t, app, "end", KeyEnd)
	completion := dispatchPopupKey(t, app, "commit", KeyEnter)
	if state := first.State(); state.Open || state.Current != "four" ||
		state.Selected != "four" || completion.Command != "drop.activate" {
		t.Fatalf("committed State()=%#v completion=%#v", state, completion)
	}
	mu.Lock()
	if len(routed) != 1 || routed[0].ID != "drop.activate" ||
		routed[0].Target != first.ID() {
		t.Fatalf("routed commands = %#v", routed)
	}
	mu.Unlock()
	if err := first.Open(); err != nil {
		t.Fatalf("first.Open(clear) error = %v", err)
	}
	dispatchPopupKey(t, app, "clear-selection", KeySpace)
	dispatchPopupKey(t, app, "commit-empty", KeyEnter)
	if state := first.State(); state.Selected != "" || state.SelectedIndex != -1 {
		t.Fatalf("AllowEmpty Space/Enter State() = %#v", state)
	}

	if err := first.Open(); err != nil {
		t.Fatalf("first.Open() error = %v", err)
	}
	if err := second.Open(); err != nil {
		t.Fatalf("second.Open() error = %v", err)
	}
	if first.State().Open || !second.State().Open {
		t.Fatalf("single-popup invariant first=%#v second=%#v", first.State(), second.State())
	}
	dispatchPopupKey(t, app, "tab", KeyTab)
	if second.State().Open {
		t.Fatalf("Tab popup/focus = %#v/%T", second.State(), app.Focused())
	}
}

func TestComboBoxEditingSelectionAndValidation(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 28, Height: 8})
	registerActionCommand(t, app, "combo.changed", "Changed", true)
	validator := &TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationWhitelist,
		Characters:  "AlphaBetaxy ",
	}
	combo, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{
			PanelOptions: PanelOptions{
				AutomationKey: "combo.edit",
				Bounds:        Rect{Width: 16, Height: 1},
			},
			Items: []ListItem{
				{Key: "alpha", Label: "Alpha"},
				{Key: "beta", Label: "Beta"},
			},
			Selected: "alpha", ChangeCommand: "combo.changed",
		},
		Validator: validator,
	})
	if err != nil {
		t.Fatalf("NewComboBox() error = %v", err)
	}
	validator.Characters = "z"
	if state := combo.State(); state.Text != "Alpha" ||
		state.Selected != "alpha" || state.Editing || !state.Valid {
		t.Fatalf("initial ComboBox State() = %#v", state)
	}

	dispatchPopupKey(t, app, "edit", KeyEnter)
	dispatchPopupKey(t, app, "type-x", "x")
	if state := combo.State(); !state.Editing || state.Text != "Alpha" {
		t.Fatalf("working edit State() = %#v", state)
	}
	completion := dispatchPopupKey(t, app, "commit-edit", KeyEnter)
	if state := combo.State(); state.Editing || state.Text != "Alphax" ||
		state.Selected != "" || completion.Command != "combo.changed" {
		t.Fatalf("committed edit State()=%#v completion=%#v", state, completion)
	}

	dispatchPopupKey(t, app, "open", KeySpace)
	dispatchPopupKey(t, app, "next", KeyDown)
	completion = dispatchPopupKey(t, app, "choose", KeyEnter)
	if state := combo.State(); state.Open || state.Selected != "beta" ||
		state.Text != "Beta" || completion.Command != "combo.changed" {
		t.Fatalf("selected item State()=%#v completion=%#v", state, completion)
	}
	dispatchPopupKey(t, app, "edit-f2", KeyF2)
	dispatchPopupKey(t, app, "type-y", "y")
	dispatchPopupKey(t, app, "cancel-edit", KeyEscape)
	if state := combo.State(); state.Editing || state.Text != "Beta" ||
		state.Selected != "beta" {
		t.Fatalf("cancelled edit State() = %#v", state)
	}

	details := comboBoxDetailsByKey(t, app, "combo.edit")
	if details.Popup.Selected != "beta" || details.Editor.Text != "Beta" ||
		details.Popup.RetainedBytes <= len(details.Editor.Text) ||
		details.Editor.Editing {
		t.Fatalf("ComboBoxDetails = %#v", details)
	}
	if err := combo.SetText("Alpha"); err != nil ||
		combo.State().Selected != "alpha" {
		t.Fatalf("SetText(exact label) error/state = %v/%#v", err, combo.State())
	}
	if err := combo.SetText("z"); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetText(hard-invalid) error = %v", err)
	}
}

func TestPopupCollectionValidationRepairTransactionsAndEmptyCombo(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 24, Height: 7})
	if _, err := NewDropDown(app.Root(), DropDownOptions{
		Items: []ListItem{{Key: "same", Label: "One"},
			{Key: "same", Label: "Two"}},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("duplicate key error = %v", err)
	}
	if _, err := NewDropDown(app.Root(), DropDownOptions{
		Items:         []ListItem{{Key: "one", Label: "One"}},
		ChangeCommand: "missing.command",
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("unregistered command error = %v", err)
	}
	hardDigits := &TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationWhitelist,
		Characters:  "0123456789",
	}
	if _, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{
			Items: []ListItem{{Key: "word", Label: "Word"}},
		},
		Validator: hardDigits,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("hard-invalid item label error = %v", err)
	}

	empty, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{PanelOptions: PanelOptions{
			AutomationKey: "combo.empty",
			Bounds:        Rect{Width: 10, Height: 1},
		}},
		Text: "",
	})
	if err != nil {
		t.Fatalf("NewComboBox(empty) error = %v", err)
	}
	if err := empty.Focus(); err != nil {
		t.Fatalf("empty ComboBox Focus() error = %v", err)
	}
	dispatchPopupKey(t, app, "empty-edit", KeyEnter)
	if !empty.State().Editing {
		t.Fatalf("empty ComboBox did not enter edit mode: %#v", empty.State())
	}
	dispatchPopupKey(t, app, "empty-cancel", KeyEscape)

	dropDown, err := NewDropDown(app.Root(), DropDownOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "drop.repair",
			Bounds:        Rect{Y: 2, Width: 12, Height: 1},
		},
		Items: []ListItem{
			{Key: "a", Label: "A"},
			{Key: "b", Label: "B"},
			{Key: "c", Label: "C"},
		},
		Current: "b", Selected: "b",
	})
	if err != nil {
		t.Fatalf("NewDropDown(repair) error = %v", err)
	}
	tx := app.NewTransaction()
	if err := tx.SetDropDownItems(dropDown, []ListItem{
		{Key: "c", Label: "C moved"},
		{Key: "a", Label: "A moved"},
	}); err != nil {
		t.Fatalf("SetDropDownItems() error = %v", err)
	}
	if state := dropDown.State(); state.Current != "b" {
		t.Fatalf("uncommitted transaction leaked State() = %#v", state)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if state := dropDown.State(); state.Current != "a" ||
		state.Selected != "a" || state.CurrentIndex != 1 {
		t.Fatalf("repaired State() = %#v", state)
	}

	combo, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Y: 4, Width: 12, Height: 1}},
			Items:        []ListItem{{Key: "one", Label: "1"}},
		},
		Validator: hardDigits,
	})
	if err != nil {
		t.Fatalf("NewComboBox(digits) error = %v", err)
	}
	if err := combo.SetItems([]ListItem{{Key: "bad", Label: "bad"}}); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetItems(hard-invalid) error = %v", err)
	}
	if got := combo.Items()[0].Key; got != "one" {
		t.Fatalf("failed SetItems changed model to %q", got)
	}
	if err := combo.SetValidator(&TextValidator{
		Enforcement: TextValidationHard,
		Mode:        TextValidationBlacklist,
		Characters:  "1",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("SetValidator(item-invalid) error = %v", err)
	}
}

func TestComboBoxConcurrentModelAndTextReplacement(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 4})
	combo, err := NewComboBox(app.Root(), ComboBoxOptions{
		DropDownOptions: DropDownOptions{
			PanelOptions: PanelOptions{Bounds: Rect{Width: 12, Height: 1}},
			Items: []ListItem{
				{Key: "a", Label: "A"},
				{Key: "b", Label: "B"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewComboBox() error = %v", err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 8)
	for worker := range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := range 20 {
				if (worker+iteration)%2 == 0 {
					if err := combo.SetText("A"); err != nil {
						errorsSeen <- err
						return
					}
					continue
				}
				if err := combo.SetItems([]ListItem{
					{Key: "b", Label: "B"},
					{Key: "a", Label: "A"},
				}); err != nil {
					errorsSeen <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent replacement error = %v", err)
	}
	state := combo.State()
	if len(combo.Items()) != 2 || state.ItemCount != 2 ||
		state.EnabledCount != 2 || strings.TrimSpace(state.Text) == "" {
		t.Fatalf("concurrent final State() = %#v", state)
	}
}
