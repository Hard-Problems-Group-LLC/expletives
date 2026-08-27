package automation

import (
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const listBoxSnapshotFixtureKey = "workspace.chat.history"

func listBoxSnapshotFixtureItems() []expletives.ListItem {
	return []expletives.ListItem{
		{
			Key: "one", Label: "One",
			Description: "alpha beta gamma delta epsilon zeta eta theta",
		},
		{
			Key: "two", Label: "Two",
			Description: "iota kappa lambda mu nu xi omicron pi",
		},
		{
			Key: "three", Label: "Three",
			Description: "rho sigma tau upsilon phi chi psi omega",
		},
	}
}

func listBoxSnapshotFixtureOptions() expletives.ListBoxOptions {
	return expletives.ListBoxOptions{
		ScrollablePanelOptions: expletives.ScrollablePanelOptions{
			ScrollViewOptions: expletives.ScrollViewOptions{
				PanelOptions: expletives.PanelOptions{
					AutomationKey: listBoxSnapshotFixtureKey,
					Bounds:        expletives.Rect{Width: 15, Height: 4},
				},
			},
			BorderForm:  expletives.BorderSingle,
			VerticalBar: expletives.ScrollBarVisibilityAlways,
		},
		Items:         listBoxSnapshotFixtureItems(),
		Current:       "three",
		Selected:      []string{"one", "three"},
		SelectionMode: expletives.CollectionSelectionMultiple,
		Wrap:          expletives.TextWrapWords,
	}
}

func projectedListBoxControl(
	t *testing.T,
	snapshot *SnapshotV1,
) *ControlSnapshot {
	t.Helper()
	if snapshot == nil {
		t.Fatal("snapshot is nil")
	}
	for index := range snapshot.Controls {
		control := &snapshot.Controls[index]
		if control.Key == listBoxSnapshotFixtureKey {
			if control.Details.ListBox == nil {
				t.Fatal("ListBox fixture has no ListBox details")
			}
			return control
		}
	}
	t.Fatalf("snapshot has no %q control", listBoxSnapshotFixtureKey)
	return nil
}

func validateProjectedListBox(
	t *testing.T,
	app *expletives.App,
) *ControlSnapshot {
	t.Helper()
	projected := snapshotFromCore(app.Snapshot())
	if err := validateSnapshot(&projected, DefaultLimits()); err != nil {
		t.Fatalf("matching client rejected server projection: %v", err)
	}
	return projectedListBoxControl(t, &projected)
}

func TestSnapshotAcceptsLegalListBoxStateMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		options        expletives.ListBoxOptions
		itemCount      int
		enabled        int
		current        string
		currentAt      int
		selected       int
		wrapped        bool
		visible        bool
		controlEnabled bool
	}{
		{
			name: "empty",
			options: expletives.ListBoxOptions{
				ScrollablePanelOptions: expletives.ScrollablePanelOptions{
					ScrollViewOptions: expletives.ScrollViewOptions{
						PanelOptions: expletives.PanelOptions{
							AutomationKey: listBoxSnapshotFixtureKey,
							Bounds:        expletives.Rect{Width: 15, Height: 4},
						},
					},
					BorderForm: expletives.BorderSingle,
				},
			},
			currentAt:      -1,
			visible:        true,
			controlEnabled: true,
		},
		{
			name:      "populated selected and scrolled",
			options:   listBoxSnapshotFixtureOptions(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, wrapped: true, visible: true, controlEnabled: true,
		},
		{
			name: "cell wrapped and scrolled",
			options: func() expletives.ListBoxOptions {
				options := listBoxSnapshotFixtureOptions()
				options.Wrap = expletives.TextWrapCells
				return options
			}(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, wrapped: true, visible: true, controlEnabled: true,
		},
		{
			name: "unwrapped and scrolled",
			options: func() expletives.ListBoxOptions {
				options := listBoxSnapshotFixtureOptions()
				options.Wrap = expletives.TextWrapNone
				return options
			}(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, visible: true, controlEnabled: true,
		},
		{
			name: "hidden wrapped and scrolled",
			options: func() expletives.ListBoxOptions {
				options := listBoxSnapshotFixtureOptions()
				options.PanelOptions.Hidden = true
				return options
			}(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, wrapped: true, controlEnabled: true,
		},
		{
			name: "disabled wrapped and scrolled",
			options: func() expletives.ListBoxOptions {
				options := listBoxSnapshotFixtureOptions()
				options.Disabled = true
				options.DisabledReason = "history unavailable"
				return options
			}(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, wrapped: true, visible: true,
		},
		{
			name: "loading with retained model",
			options: func() expletives.ListBoxOptions {
				options := listBoxSnapshotFixtureOptions()
				options.Status = expletives.CollectionLoading
				options.StatusMessage = "Refreshing history"
				return options
			}(),
			itemCount: 3, enabled: 3, current: "three", currentAt: 2,
			selected: 2, visible: true, controlEnabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app, err := expletives.NewApp(expletives.AppOptions{
				Size: expletives.Size{Width: 30, Height: 10},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := expletives.NewListBox(app.Root(), test.options); err != nil {
				t.Fatal(err)
			}
			control := validateProjectedListBox(t, app)
			details := control.Details.ListBox
			if details.ItemCount != test.itemCount ||
				details.EnabledCount != test.enabled ||
				details.Current != test.current ||
				details.CurrentIndex != test.currentAt ||
				details.SelectedCount != test.selected ||
				details.Enabled != test.controlEnabled ||
				control.Visible != test.visible {
				t.Fatalf("projected ListBox boundary evidence = control %+v details %+v", control, details)
			}
			if test.itemCount > 0 && details.RetainedBytes == 0 {
				t.Fatal("populated ListBox has zero retained bytes")
			}
			if test.wrapped {
				if details.VisualRowCount <= details.ItemCount ||
					details.Viewport.State.Offset.Y <= details.CurrentIndex {
					t.Fatalf("fixture did not separate logical and visual coordinates: %+v", details)
				}
			}
		})
	}
}

func TestSnapshotAcceptsRepeatedWrappedListBoxReplacement(t *testing.T) {
	t.Parallel()
	app, err := expletives.NewApp(expletives.AppOptions{
		Size: expletives.Size{Width: 30, Height: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := expletives.NewListBox(app.Root(), listBoxSnapshotFixtureOptions())
	if err != nil {
		t.Fatal(err)
	}
	assertState := func(current string, currentAt, items, selected int) {
		t.Helper()
		details := validateProjectedListBox(t, app).Details.ListBox
		if details.Current != current || details.CurrentIndex != currentAt ||
			details.ItemCount != items || details.SelectedCount != selected {
			t.Fatalf("replacement projection = %+v", details)
		}
	}
	assertState("three", 2, 3, 2)

	items := listBoxSnapshotFixtureItems()
	reordered := []expletives.ListItem{items[2], items[0], items[1]}
	if err := list.SetItems(reordered); err != nil {
		t.Fatal(err)
	}
	assertState("three", 0, 3, 2)

	if err := list.Replace(reordered, "two", []string{"two"}); err != nil {
		t.Fatal(err)
	}
	assertState("two", 2, 3, 1)

	if err := list.SetItems(reordered[:2]); err != nil {
		t.Fatal(err)
	}
	assertState("one", 1, 2, 0)

	if err := list.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if control := validateProjectedListBox(t, app); control.Visible {
		t.Fatal("hidden replacement ListBox projected as visible")
	}

	if err := list.Replace(nil, "", nil); err != nil {
		t.Fatal(err)
	}
	assertState("", -1, 0, 0)
}
