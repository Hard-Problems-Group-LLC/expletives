package expletives_test

import (
	"context"
	"strings"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func TestExternalConsumerObservesLiveTextFieldPresentation(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 16, Height: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []expletives.CommandID{
		"draft.edited", "draft.submitted",
	} {
		if err := app.RegisterCommand(expletives.CommandDefinition{
			ID: command, Label: string(command), Enabled: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	field, err := expletives.NewTextField(
		app.Root(),
		expletives.TextFieldOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "draft",
				Bounds:        expletives.Rect{Width: 12, Height: 1},
			},
			EditCommand:   "draft.edited",
			SubmitCommand: "draft.submitted",
			MaximumBytes:  4,
			FocusedStyle:  "text_input.focused",
			EditingStyle:  "text_input.focused_invalid",
			ByteStyles: []expletives.TextFieldByteStyle{
				{MinimumBytes: 0, Style: "text_input.valid"},
				{MinimumBytes: 4, Style: "text_input.invalid"},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var routed []expletives.CommandID
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command expletives.Command,
	) expletives.CommandResult {
		routed = append(routed, command.ID)
		return expletives.CommandResult{Outcome: expletives.OutcomeApplied}
	}); err != nil {
		t.Fatal(err)
	}
	for request, key := range []expletives.Key{expletives.KeyEnter, "é"} {
		if _, err := app.DispatchKey(
			context.Background(),
			"external-test",
			"text-field-"+string(rune('0'+request)),
			expletives.KeyEvent{Kind: expletives.KeyEventPress, Key: key},
		); err != nil {
			t.Fatal(err)
		}
	}
	if field.Text() != "" || field.CurrentText() != "é" ||
		field.MaximumBytes() != 4 ||
		strings.Join(commandIDs(routed), ",") != "draft.edited" {
		t.Fatalf(
			"Text=%q CurrentText=%q commands=%v",
			field.Text(),
			field.CurrentText(),
			routed,
		)
	}
	completion, err := app.DispatchTextInput(
		context.Background(),
		"external-test",
		"over-maximum",
		expletives.TextInputEvent{Kind: expletives.TextInputCommitted, Text: "éé"},
	)
	if err != nil || completion.Outcome != expletives.OutcomeRejected ||
		field.CurrentText() != "é" {
		t.Fatalf(
			"over-maximum committed text = %+v CurrentText=%q error=%v",
			completion,
			field.CurrentText(),
			err,
		)
	}
}

func commandIDs(values []expletives.CommandID) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}

func TestExternalConsumerConfiguresTableSelectionPolicies(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 36, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	columns := []expletives.Column{{
		Key: "value", Header: "Value", Editable: true,
	}}
	rows := []expletives.TableRow{
		{Key: "one", Cells: []expletives.TableCell{{Column: "value", Text: "One"}}},
		{Key: "disabled", Disabled: true, DisabledReason: "Unavailable"},
		{Key: "three", Cells: []expletives.TableCell{{Column: "value", Text: "Three"}}},
	}
	table, err := expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.table", Bounds: expletives.Rect{Width: 16, Height: 6},
			}},
		},
		Columns: columns, Rows: rows, CurrentRow: "one", CurrentColumn: "value",
		SelectionStyle:   expletives.TableSelectionRange,
		RequireSelection: true, RangeAnchor: "one", RangeExtent: "three",
	})
	if err != nil {
		t.Fatal(err)
	}
	grid, err := expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.grid", Bounds: expletives.Rect{X: 18, Width: 16, Height: 6},
			}},
		},
		Columns: columns, Rows: rows, CurrentRow: "one", CurrentColumn: "value",
		SelectionStyle: expletives.TableSelectionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	state := table.State()
	if state.SelectionStyle != expletives.TableSelectionRange ||
		state.RangeAnchor != "one" || state.RangeExtent != "three" ||
		strings.Join(state.Selected, ",") != "one,three" {
		t.Fatalf("initial external Table state = %+v", state)
	}
	if err := table.Focus(); err != nil {
		t.Fatal(err)
	}
	completion, err := app.DispatchKey(
		context.Background(),
		"external-test",
		"range-contract",
		expletives.KeyEvent{Kind: expletives.KeyEventPress, Key: "["},
	)
	if err != nil || completion.Outcome != expletives.OutcomeApplied {
		t.Fatalf("range contraction = %+v, %v", completion, err)
	}
	state = table.State()
	if state.RangeAnchor != "one" || state.RangeExtent != "one" ||
		strings.Join(state.Selected, ",") != "one" {
		t.Fatalf("contracted external Table state = %+v", state)
	}

	selected := []string{"one", "three"}
	if err := grid.SetSelectionPolicy(expletives.TableSelectionPolicy{
		Style: expletives.TableSelectionMultiple, Selected: selected,
	}); err != nil {
		t.Fatal(err)
	}
	selected[0] = "caller-mutation"
	if got := strings.Join(grid.State().Selected, ","); got != "one,three" {
		t.Fatalf("DataGrid retained caller selection storage: %q", got)
	}

	transaction := app.NewTransaction()
	if err := transaction.SetTableSelectionPolicy(
		table,
		expletives.TableSelectionPolicy{Style: expletives.TableSelectionNone},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetDataGridSelectionPolicy(
		grid,
		expletives.TableSelectionPolicy{
			Style:  expletives.TableSelectionRange,
			Anchor: "one", Extent: "three",
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if table.State().SelectionStyle != expletives.TableSelectionNone ||
		len(table.State().Selected) != 0 ||
		grid.State().SelectionStyle != expletives.TableSelectionRange ||
		strings.Join(grid.State().Selected, ",") != "one,three" {
		t.Fatalf("external atomic policies: table=%+v grid=%+v", table.State(), grid.State())
	}
	for _, control := range app.Snapshot().Controls {
		if control.Key == "external.table" &&
			(control.Details.Table == nil ||
				control.Details.Table.SelectionStyle != expletives.TableSelectionNone) {
			t.Fatalf("external Table details = %+v", control.Details)
		}
		if control.Key == "external.grid" &&
			(control.Details.DataGrid == nil ||
				control.Details.DataGrid.Table.SelectionStyle != expletives.TableSelectionRange ||
				control.Details.DataGrid.Table.RangeAnchor != "one" ||
				control.Details.DataGrid.Table.RangeExtent != "three") {
			t.Fatalf("external DataGrid details = %+v", control.Details)
		}
	}
}

func TestExternalConsumerConfiguresColumnPresentation(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 42, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	columns := []expletives.Column{
		{Key: "name", Header: "Name", Editable: true},
		{Key: "notes", Header: "Notes", Editable: true},
	}
	rows := []expletives.TableRow{{
		Key: "one", Cells: []expletives.TableCell{
			{Column: "name", Text: "One"},
			{Column: "notes", Text: "Long notes"},
		},
	}}
	presentation := []expletives.TableColumnPresentation{
		{Column: "notes", Visible: true, Wrap: expletives.TableColumnHang},
		{Column: "name", Visible: false, Wrap: expletives.TableColumnWrapWords},
	}
	table, err := expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.presentation.table", Bounds: expletives.Rect{Width: 19, Height: 6},
			}},
		},
		Columns: columns, Rows: rows, CurrentColumn: "name",
		ColumnPresentation: presentation,
	})
	if err != nil {
		t.Fatal(err)
	}
	grid, err := expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.presentation.grid", Bounds: expletives.Rect{X: 21, Width: 19, Height: 6},
			}},
		},
		Columns: columns, Rows: rows,
	})
	if err != nil {
		t.Fatal(err)
	}
	presentation[0].Column = "caller-mutation"
	if state := table.State(); state.CurrentColumn != "notes" ||
		state.VisibleColumnCount != 1 || state.ColumnPresentation[0].Column != "notes" {
		t.Fatalf("external Table presentation = %+v", state)
	}

	transaction := app.NewTransaction()
	if err := transaction.SetTableColumnPresentation(table, nil); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetDataGridColumnPresentation(grid, []expletives.TableColumnPresentation{
		{Column: "name", Visible: false},
		{Column: "notes", Visible: true, Wrap: expletives.TableColumnWrapWords},
	}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := table.ColumnPresentation(); len(got) != 2 || got[0].Column != "name" ||
		got[0].Wrap != expletives.TableColumnClip {
		t.Fatalf("external canonical Table presentation = %+v", got)
	}
	if got := grid.ColumnPresentation(); len(got) != 2 || got[0].Visible ||
		got[1].Wrap != expletives.TableColumnWrapWords || grid.State().CurrentColumn != "notes" {
		t.Fatalf("external DataGrid presentation = %+v state=%+v", got, grid.State())
	}
	for _, control := range app.Snapshot().Controls {
		switch control.Key {
		case "external.presentation.table":
			if control.Details.Table == nil ||
				len(control.Details.Table.ColumnPresentation) != 2 ||
				control.Details.Table.VisibleColumnCount != 2 {
				t.Fatalf("external Table details = %+v", control.Details)
			}
		case "external.presentation.grid":
			if control.Details.DataGrid == nil ||
				control.Details.DataGrid.Table.VisibleColumnCount != 1 ||
				len(control.Details.DataGrid.Table.PresentationDigest) != 64 {
				t.Fatalf("external DataGrid details = %+v", control.Details)
			}
		}
	}
}

func TestExternalConsumerConfiguresTableFeatures(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 48, Height: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	features := []expletives.TableFeature{expletives.TableFeatureColumns}
	table, err := expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.features.table",
				Bounds:        expletives.Rect{Width: 22, Height: 7},
			}},
		},
		Features: features,
		Columns:  []expletives.Column{{Key: "value", Header: "Value"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	grid, err := expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.features.grid",
				Bounds:        expletives.Rect{X: 24, Width: 22, Height: 7},
			}},
		},
		Columns: []expletives.Column{{Key: "value", Header: "Value"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	features[0] = "caller-mutation"
	if state := table.State(); len(state.Features) != 1 ||
		state.Features[0] != expletives.TableFeatureColumns ||
		state.FocusPart != expletives.TableFocusBody {
		t.Fatalf("external Table features = %+v", state)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetTableFeatures(table, nil); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetDataGridFeatures(
		grid,
		[]expletives.TableFeature{expletives.TableFeatureColumns},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(table.State().Features) != 0 ||
		len(grid.State().Features) != 1 ||
		grid.State().Features[0] != expletives.TableFeatureColumns {
		t.Fatalf("external feature mutation: table=%+v grid=%+v", table.State(), grid.State())
	}
	for _, control := range app.Snapshot().Controls {
		switch control.Key {
		case "external.features.table":
			if control.Details.Table == nil || control.Details.Table.ColumnsActionVisible {
				t.Fatalf("external Table feature details = %+v", control.Details)
			}
		case "external.features.grid":
			if control.Details.DataGrid == nil ||
				!control.Details.DataGrid.Table.ColumnsActionVisible {
				t.Fatalf("external DataGrid feature details = %+v", control.Details)
			}
		}
	}
}

func TestExternalConsumerConfiguresTableVisualRolesAndFocus(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 42, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	columns := []expletives.Column{{
		Key: "value", Header: "Value", Editable: true,
	}}
	rows := []expletives.TableRow{{
		Key: "one", Cells: []expletives.TableCell{{Column: "value", Text: "One"}},
	}}
	table, err := expletives.NewTable(app.Root(), expletives.TableOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.roles.table", Bounds: expletives.Rect{Width: 19, Height: 6},
			}},
		},
		Columns: columns, Rows: rows,
	})
	if err != nil {
		t.Fatal(err)
	}
	grid, err := expletives.NewDataGrid(app.Root(), expletives.DataGridOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{PanelOptions: expletives.PanelOptions{
				AutomationKey: "external.roles.grid", Bounds: expletives.Rect{X: 21, Width: 19, Height: 6},
			}},
		},
		Columns: columns, Rows: rows,
	})
	if err != nil {
		t.Fatal(err)
	}

	tableRoles := table.VisualRoles()
	tableRoles.Body = "collection.current"
	tableRoles.Header = "collection.current_selected"
	gridRoles := grid.VisualRoles()
	gridRoles.Editable = "text_input.valid"
	gridRoles.FocusedEdit = "text_input.focused"
	transaction := app.NewTransaction()
	if err := transaction.SetTableVisualRoles(table, tableRoles); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetTableFocusMode(table, expletives.TableFocusRow); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetDataGridVisualRoles(grid, gridRoles); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if table.VisualRoles() != tableRoles || table.State().VisualRoles != tableRoles ||
		table.State().FocusMode != expletives.TableFocusRow {
		t.Fatalf("external Table roles/focus = %+v", table.State())
	}
	if grid.VisualRoles() != gridRoles || grid.State().VisualRoles != gridRoles ||
		grid.State().TableState.VisualRoles != gridRoles.TableVisualRoles {
		t.Fatalf("external DataGrid roles = %+v", grid.State())
	}
	for _, control := range app.Snapshot().Controls {
		switch control.Key {
		case "external.roles.table":
			if control.Details.Table == nil ||
				control.Details.Table.VisualRoles != tableRoles ||
				len(control.Details.Table.VisualRolesDigest) != 64 {
				t.Fatalf("external Table role details = %+v", control.Details)
			}
		case "external.roles.grid":
			if control.Details.DataGrid == nil ||
				control.Details.DataGrid.VisualRoles != gridRoles ||
				len(control.Details.DataGrid.VisualRolesDigest) != 64 {
				t.Fatalf("external DataGrid role details = %+v", control.Details)
			}
		}
	}

	updated := table.VisualRoles()
	updated.Body = "collection.current_selected"
	if err := table.SetVisualRoles(updated); err != nil {
		t.Fatal(err)
	}
	if err := table.SetFocusMode(expletives.TableFocusCell); err != nil {
		t.Fatal(err)
	}
	gridUpdated := grid.VisualRoles()
	gridUpdated.InvalidEdit = "text_input.focused_invalid"
	if err := grid.SetVisualRoles(gridUpdated); err != nil {
		t.Fatal(err)
	}
	if table.VisualRoles() != updated || table.State().FocusMode != expletives.TableFocusCell ||
		grid.VisualRoles() != gridUpdated {
		t.Fatalf("external direct roles/focus: table=%+v grid=%+v", table.State(), grid.State())
	}
}

func TestExternalConsumerDeclaresAndObservesInputScope(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 20, Height: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: "workspace",
		Bounds:        expletives.Rect{Width: 20, Height: 5},
		InputScope:    expletives.InputScopeEscaping,
	}); err != nil {
		t.Fatal(err)
	}
	snapshot := app.Snapshot()
	var observed expletives.InputScopeMode
	for _, control := range snapshot.Controls {
		if control.Key == "workspace" && control.Details.Container != nil {
			observed = control.Details.Container.InputScope
		}
	}
	if observed != expletives.InputScopeEscaping {
		t.Fatalf("workspace input scope = %q", observed)
	}
}

// counterModel deliberately contains no toolkit types.
type counterModel struct {
	value int
}

func updateCounter(model counterModel, command string) counterModel {
	if command == "counter.increment" {
		model.value++
	}
	return model
}

func TestExternalConsumerKeepsModelControllerAndViewSeparate(t *testing.T) {
	t.Parallel()
	theme, err := expletives.NewTheme(
		expletives.Style{
			ID:         "application",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: expletives.RGB(0, 0, 0),
		},
		expletives.Style{
			ID:         "counter.zero",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: expletives.RGB(0x10, 0x10, 0x10),
		},
		expletives.Style{
			ID:         "counter.nonzero",
			Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
			Background: expletives.RGB(0, 0x40, 0),
		},
	)
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:      expletives.Size{Width: 8, Height: 3},
		Scenario:  "external.mvc",
		Theme:     theme,
		RootStyle: "application",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	view, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: "counter-view",
		Bounds:        expletives.Rect{X: 1, Y: 1, Width: 4, Height: 1},
		Style:         "counter.zero",
	})
	if err != nil {
		t.Fatalf("NewPanel() error = %v", err)
	}

	model := counterModel{}
	if err := app.SetCommandHandler(func(
		_ context.Context,
		command expletives.Command,
	) (expletives.Outcome, error) {
		model = updateCounter(model, string(command.ID))
		if model.value == 0 {
			return expletives.OutcomeNoOp, nil
		}
		if err := view.SetStyle("counter.nonzero"); err != nil {
			return expletives.OutcomeFailed, err
		}
		return expletives.OutcomeApplied, nil
	}); err != nil {
		t.Fatalf("SetCommandHandler() error = %v", err)
	}

	completion, err := app.InvokeCommand(
		context.Background(),
		"external-controller",
		"increment-1",
		"counter.increment",
		view.ID(),
	)
	if err != nil {
		t.Fatalf("InvokeCommand() error = %v", err)
	}
	if completion.Outcome != expletives.OutcomeApplied || model.value != 1 {
		t.Fatalf("completion=%+v model=%+v", completion, model)
	}
	snapshot, err := app.SnapshotAt(completion.FrameSequence)
	if err != nil {
		t.Fatalf("SnapshotAt() error = %v", err)
	}
	if snapshot.Scenario != "external.mvc" {
		t.Fatalf("snapshot scenario = %q", snapshot.Scenario)
	}
	if len(snapshot.Controls) != 2 ||
		snapshot.Controls[1].Key != "counter-view" ||
		snapshot.Controls[1].Style != "counter.nonzero" {
		t.Fatalf("external view snapshot = %#v", snapshot.Controls)
	}
}

func TestExternalConsumerBuildsAndStacksLayouts(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 8, Height: 2},
		Scenario: "external.layouts",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: "first",
		MinimumSize:   expletives.Size{Width: 2, Height: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := expletives.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: "second",
		MinimumSize:   expletives.Size{Width: 2, Height: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	layout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{AutomationKey: "main", Gap: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(
		first,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(
		second,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		t.Fatal(err)
	}
	if err := app.Root().SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	firstBounds := first.Bounds()
	if err := first.Raise(); err != nil {
		t.Fatal(err)
	}
	snapshot := app.Snapshot()
	if first.Bounds() != firstBounds ||
		len(snapshot.Layouts) != 1 ||
		snapshot.Layouts[0].Items[0].LayoutIndex != 0 ||
		snapshot.Layouts[0].Items[0].StackIndex != 1 {
		t.Fatalf("external Layout snapshot = %#v", snapshot.Layouts)
	}
}

func TestExternalConsumerPublishesCopiedProgressState(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 24, Height: 4},
		Scenario: "external.progress",
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	bar, err := transaction.NewProgressBar(
		app.Root(),
		expletives.ProgressBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "progress",
				Bounds:        expletives.Rect{Width: 12, Height: 1},
			},
			State: expletives.ProgressBarState{
				Current: 1,
				Total:   4,
				Status:  expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	spinner, err := transaction.NewSpinner(
		app.Root(),
		expletives.SpinnerOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "spinner",
				Bounds: expletives.Rect{
					X: 13, Width: 1, Height: 1,
				},
			},
			State: expletives.ActivityState{
				Status: expletives.ProgressRunning,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := bar.Update(
		context.Background(),
		expletives.ProgressBarState{
			Current: 2,
			Total:   4,
			Status:  expletives.ProgressRunning,
		},
	); err != nil {
		t.Fatal(err)
	}
	transaction = app.NewTransaction()
	if err := transaction.SetProgressBarState(
		bar,
		expletives.ProgressBarState{
			Current: 4,
			Total:   4,
			Status:  expletives.ProgressCompleted,
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetActivityState(
		spinner,
		expletives.ActivityState{
			Tick:   3,
			Status: expletives.ProgressRunning,
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	snapshot := app.Snapshot()
	var barDetails, spinnerDetails *expletives.ProgressDetails
	for index := range snapshot.Controls {
		switch snapshot.Controls[index].Key {
		case "progress":
			barDetails = snapshot.Controls[index].Details.Progress
		case "spinner":
			spinnerDetails = snapshot.Controls[index].Details.Progress
		}
	}
	if barDetails == nil ||
		barDetails.Status != expletives.ProgressCompleted ||
		barDetails.Current != 4 || barDetails.Total != 4 ||
		spinnerDetails == nil ||
		spinnerDetails.Status != expletives.ProgressRunning ||
		spinnerDetails.Tick != 3 || spinnerDetails.FrameIndex != 3 {
		t.Fatalf(
			"external Progress snapshot bar=%#v spinner=%#v",
			barDetails,
			spinnerDetails,
		)
	}
}

func TestExternalConsumerBuildsScrollableContent(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 20, Height: 8},
		Scenario: "external.scrollable",
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	scrollable, err := transaction.NewScrollablePanel(
		app.Root(),
		expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: "scroll",
					Bounds: expletives.Rect{
						Width: 10, Height: 6,
					},
				},
				State: expletives.ViewportState{
					ContentSize: expletives.Size{
						Width: 20, Height: 10,
					},
				},
			},
			BorderForm: expletives.BorderSingle,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	child, err := transaction.NewPanel(
		scrollable.Content(),
		expletives.PanelOptions{
			AutomationKey: "scroll.child",
			MinimumSize:   expletives.Size{Width: 4, Height: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "scroll.content.layout",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.AddPanel(child, expletives.LayoutItemOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.SetLayout(scrollable.Content(), layout); err != nil {
		t.Fatal(err)
	}
	if err := transaction.EnsureViewportVisible(
		scrollable,
		expletives.Rect{X: 15, Y: 8, Width: 2, Height: 2},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if scrollable.Content().Parent() != scrollable ||
		scrollable.State().Offset != (expletives.Point{X: 10, Y: 7}) {
		t.Fatalf(
			"external ScrollablePanel Content=%#v state=%+v",
			scrollable.Content(),
			scrollable.State(),
		)
	}
}

func TestExternalConsumerBuildsAndUpdatesScrollBar(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 24, Height: 4},
		Scenario: "external.scrollbar",
	})
	if err != nil {
		t.Fatal(err)
	}
	scrollBar, err := expletives.NewScrollBar(
		app.Root(),
		expletives.ScrollBarOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "scrollbar",
				Bounds:        expletives.Rect{Width: 12, Height: 1},
			},
			State: expletives.ScrollBarState{
				ContentSize:  80,
				ViewportSize: 20,
				Offset:       10,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := scrollBar.Update(
		context.Background(),
		expletives.ScrollBarState{
			ContentSize:  80,
			ViewportSize: 20,
			Offset:       30,
		},
	); err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetScrollBarState(
		scrollBar,
		expletives.ScrollBarState{
			ContentSize:  80,
			ViewportSize: 20,
			Offset:       60,
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if state := scrollBar.State(); state.Offset != 60 {
		t.Fatalf("ScrollBar.State() = %#v", state)
	}
	var details *expletives.ScrollBarDetails
	for index := range app.Snapshot().Controls {
		control := app.Snapshot().Controls[index]
		if control.Key == "scrollbar" {
			details = control.Details.ScrollBar
			break
		}
	}
	if details == nil ||
		details.Offset != 60 ||
		details.MaximumOffset != 60 ||
		details.ThumbStart != 9 {
		t.Fatalf("external ScrollBar details = %#v", details)
	}
}

func TestExternalConsumerBuildsTabbedPanelAtomically(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 30, Height: 10},
		Scenario: "external.tabs",
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := app.NewTransaction()
	panel, err := transaction.NewTabbedPanel(
		app.Root(),
		expletives.TabbedPanelOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "tabs",
				Bounds: expletives.Rect{
					Width: 20, Height: 8,
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := transaction.NewPanel(
		panel,
		expletives.PanelOptions{AutomationKey: "page.first"},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transaction.NewPanel(
		panel,
		expletives.PanelOptions{AutomationKey: "page.second"},
	)
	if err != nil {
		t.Fatal(err)
	}
	tabs := []expletives.Tab{
		{
			Key: "first", Value: "first", Label: "First",
			Mnemonic: "f", Page: first,
		},
		{
			Key: "second", Value: "second", Label: "Second",
			Mnemonic: "s", Page: second,
		},
	}
	if err := transaction.SetTabs(panel, tabs, "first"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if panel.Selected() != "first" || len(panel.Tabs()) != 2 {
		t.Fatalf("initial external tabs = %#v, %q",
			panel.Tabs(), panel.Selected())
	}
	if err := panel.SetSelected("second"); err != nil {
		t.Fatal(err)
	}
	if panel.Selected() != "second" {
		t.Fatalf("selected external tab = %q", panel.Selected())
	}
	var details *expletives.TabbedPanelDetails
	for _, control := range app.Snapshot().Controls {
		if control.Key == "tabs" {
			details = control.Details.TabbedPanel
		}
	}
	if details == nil ||
		details.Selected != "second" ||
		len(details.Tabs) != 2 ||
		details.Tabs[1].Page != second.ID() {
		t.Fatalf("external TabbedPanel details = %#v", details)
	}
}
