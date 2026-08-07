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
		strings.Join(commandIDs(routed), ",") != "draft.edited" {
		t.Fatalf(
			"Text=%q CurrentText=%q commands=%v",
			field.Text(),
			field.CurrentText(),
			routed,
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
