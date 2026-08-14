package demo

import (
	"context"
	"fmt"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func focusedCatalogControlKey(snapshot expletives.Snapshot) string {
	for _, control := range snapshot.Controls {
		if control.Focused {
			return control.Key
		}
	}
	return ""
}

func expectedCatalogColorDirection(
	prefix string,
	row int,
	rows int,
	axis string,
	key expletives.Key,
) string {
	start := fmt.Sprintf("%s.colors.%02d.%s", prefix, row, axis)
	switch key {
	case expletives.KeyLeft:
		if axis == "background" {
			return fmt.Sprintf("%s.colors.%02d.foreground", prefix, row)
		}
		return prefix + ".control"
	case expletives.KeyRight:
		if axis == "foreground" {
			return fmt.Sprintf("%s.colors.%02d.background", prefix, row)
		}
	case expletives.KeyUp:
		if row > 0 {
			return fmt.Sprintf("%s.colors.%02d.%s", prefix, row-1, axis)
		}
	case expletives.KeyDown:
		if row+1 < rows {
			return fmt.Sprintf("%s.colors.%02d.%s", prefix, row+1, axis)
		}
	}
	return start
}

func dispatchCatalogColorDirection(
	t *testing.T,
	scene *Scene,
	prefix string,
	row int,
	rows int,
	axis string,
	dropdown *expletives.DropDown,
	key expletives.Key,
	requestPrefix string,
	focus bool,
) {
	t.Helper()
	if focus {
		if err := dropdown.Focus(); err != nil {
			t.Fatalf("Focus(%s row %d %s): %v", prefix, row, axis, err)
		}
	}
	start := fmt.Sprintf("%s.colors.%02d.%s", prefix, row, axis)
	if got := focusedCatalogControlKey(scene.App.Snapshot()); got != start {
		t.Fatalf("focus before %s = %q, want %q", key, got, start)
	}
	completion, err := scene.App.DispatchKey(
		context.Background(),
		"catalog-direction-test",
		fmt.Sprintf("%s-%s-%02d-%s-%s", requestPrefix, prefix, row, axis, key),
		expletives.KeyEvent{Kind: expletives.KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s row %d %s, %s): %v", prefix, row, axis, key, err)
	}
	got := focusedCatalogControlKey(scene.App.Snapshot())
	want := expectedCatalogColorDirection(prefix, row, rows, axis, key)
	if got != want {
		t.Fatalf("%s.colors.%02d.%s %s focus = %q, want %q (outcome=%s)",
			prefix, row, axis, key, got, want, completion.Outcome)
	}
}

func catalogScrollableMaximumOffset(
	t *testing.T,
	snapshot expletives.Snapshot,
	control expletives.ControlID,
) expletives.Point {
	t.Helper()
	for _, candidate := range snapshot.Controls {
		if candidate.ID == control && candidate.Details.Scrollable != nil {
			return candidate.Details.Scrollable.MaximumOffset
		}
	}
	t.Fatalf("ScrollablePanel %s details are absent", control)
	return expletives.Point{}
}

func TestDedicatedTableCatalogColorDirectionalNavigation(t *testing.T) {
	t.Parallel()
	scene, err := New(expletives.Size{Width: 190, Height: 40}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		command expletives.CommandID
		catalog *tableCatalogControl
		prefix  string
	}{
		{"Table", CommandTables, scene.tableCatalog, "tables"},
		{"DataGrid", CommandDataGrid, scene.dataGridCatalog, "data-grid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			completion, invokeErr := scene.App.InvokeCommand(
				context.Background(), "catalog-direction-test",
				"show-direction-"+test.prefix, test.command, "",
			)
			if invokeErr != nil || completion.Outcome != expletives.OutcomeApplied {
				t.Fatalf("show %s = %+v, %v", test.name, completion, invokeErr)
			}
			if err := test.catalog.notebook.SetSelected("colors"); err != nil {
				t.Fatal(err)
			}
			rows := len(test.catalog.colorBindings) / 2
			if rows == 0 || len(test.catalog.colorBindings) != 2*rows {
				t.Fatalf("%s color bindings = %d", test.name, len(test.catalog.colorBindings))
			}

			// Exercise all four directions from every foreground/background cell.
			for index, binding := range test.catalog.colorBindings {
				row := index / 2
				axis := "foreground"
				if !binding.foreground {
					axis = "background"
				}
				for _, key := range []expletives.Key{
					expletives.KeyLeft,
					expletives.KeyRight,
					expletives.KeyUp,
					expletives.KeyDown,
				} {
					dispatchCatalogColorDirection(
						t, scene, test.prefix, row, rows, axis,
						binding.dropdown, key, "complete", true,
					)
				}
			}

			// At offset zero and at the maximum offset, the nested Notebook and
			// ScrollablePanel centers fall between a distinct pair of rows. Those
			// ancestors must not make the aligned vertical crossing ambiguous.
			state := test.catalog.colors.State()
			maximum := catalogScrollableMaximumOffset(
				t, scene.App.Snapshot(), test.catalog.colors.ID(),
			)
			for _, offset := range []int{0, maximum.Y} {
				lower := 10 + offset
				upper := lower + 1
				if upper >= rows {
					t.Fatalf("%s offset %d regression rows exceed %d", test.name, offset, rows)
				}
				for axisIndex, axis := range []string{"foreground", "background"} {
					for _, crossing := range []struct {
						row int
						key expletives.Key
					}{
						{lower, expletives.KeyDown},
						{upper, expletives.KeyUp},
					} {
						binding := test.catalog.colorBindings[2*crossing.row+axisIndex]
						if err := binding.dropdown.Focus(); err != nil {
							t.Fatalf("Focus(offset %d row %d %s): %v",
								offset, crossing.row, axis, err)
						}
						state.Offset = expletives.Point{Y: offset}
						if err := test.catalog.colors.SetState(state); err != nil {
							t.Fatalf("SetState(offset %d): %v", offset, err)
						}
						dispatchCatalogColorDirection(
							t, scene, test.prefix, crossing.row, rows, axis,
							binding.dropdown, crossing.key,
							fmt.Sprintf("offset-%d", offset), false,
						)
						if got := test.catalog.colors.State().Offset.Y; got != offset {
							t.Fatalf("%s offset after row %d %s = %d, want %d",
								test.name, crossing.row, crossing.key, got, offset)
						}
					}
				}
			}
		})
	}
}
