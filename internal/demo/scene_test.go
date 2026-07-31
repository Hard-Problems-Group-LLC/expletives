package demo

import (
	"context"
	"fmt"
	"testing"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

type enabledMenuPath struct {
	key      string
	keys     []expletives.Key
	command  expletives.CommandID
	outcome  expletives.Outcome
	screen   expletives.CommandID
	submenus []string
}

var enabledCatalogMenuPaths = []enabledMenuPath{
	{"menu.file.home", []expletives.Key{"i", "h"}, CommandViewHome, expletives.OutcomeNoOp, CommandViewHome, nil},
	{"menu.file.quit", []expletives.Key{"i", "q"}, CommandAppQuit, expletives.OutcomeExited, "", nil},
	{"menu.panels.core", []expletives.Key{"n", "c"}, CommandViewPanelsCore, expletives.OutcomeApplied, CommandViewPanelsCore, nil},
	{"menu.panels.styles", []expletives.Key{"n", "v"}, CommandViewPanelStyles, expletives.OutcomeApplied, CommandViewPanelStyles, nil},
	{"menu.stack.panel.raise", []expletives.Key{"n", "t", "r"}, CommandPanelRaise, expletives.OutcomeApplied, CommandViewPanelsCore, []string{"menu.panels.stacking"}},
	{"menu.stack.panel.lower", []expletives.Key{"n", "t", "l"}, CommandPanelLower, expletives.OutcomeApplied, CommandViewPanelsCore, []string{"menu.panels.stacking"}},
	{"menu.layouts.box", []expletives.Key{"a", "b"}, CommandViewLayoutBox, expletives.OutcomeApplied, CommandViewLayoutBox, nil},
	{"menu.layouts.grid", []expletives.Key{"a", "g"}, CommandViewLayoutGrid, expletives.OutcomeApplied, CommandViewLayoutGrid, nil},
	{"menu.stack.layer.raise", []expletives.Key{"a", "s", "r"}, CommandLayerRaise, expletives.OutcomeApplied, CommandViewLayoutBox, []string{"menu.layouts.stacking"}},
	{"menu.stack.layer.lower", []expletives.Key{"a", "s", "l"}, CommandLayerLower, expletives.OutcomeApplied, CommandViewLayoutBox, []string{"menu.layouts.stacking"}},
	{"menu.controls.text", []expletives.Key{"c", "t"}, CommandViewText, expletives.OutcomeApplied, CommandViewText, nil},
	{"menu.controls.actions", []expletives.Key{"c", "a"}, CommandViewActions, expletives.OutcomeApplied, CommandViewActions, nil},
	{"menu.menus.overview", []expletives.Key{"m", "o"}, CommandViewMenus, expletives.OutcomeApplied, CommandViewMenus, nil},
	{"menu.help.about", []expletives.Key{"p", "a"}, CommandViewAbout, expletives.OutcomeApplied, CommandViewAbout, nil},
}

type disabledMenuEntry struct {
	key  string
	root expletives.Key
	down int
}

var disabledCatalogMenuEntries = []disabledMenuEntry{
	{"menu.panels.scrollbars", "n", 3},
	{"menu.layouts.absolute", "a", 3},
	{"menu.controls.status", "c", 2},
	{"menu.controls.headers", "c", 3},
	{"menu.controls.selection", "c", 4},
	{"menu.controls.input", "c", 5},
	{"menu.controls.progress", "c", 6},
	{"menu.controls.navigation", "c", 7},
	{"menu.controls.scrolling", "c", 8},
	{"menu.controls.collections", "c", 9},
	{"menu.menus.panel", "m", 1},
	{"menu.menus.context", "m", 2},
	{"menu.dialogs.message", "d", 0},
	{"menu.dialogs.confirm", "d", 1},
	{"menu.dialogs.input", "d", 2},
	{"menu.dialogs.progress", "d", 3},
}

func TestCatalogEveryEnabledMenuPath(t *testing.T) {
	for _, test := range enabledCatalogMenuPaths {
		t.Run(test.key, func(t *testing.T) {
			scene, err := New(expletives.Size{Width: 100, Height: 30}, "")
			if err != nil {
				t.Fatal(err)
			}
			request := 0
			openCatalogMenu(t, scene, test.keys[0], &request)
			for index, key := range test.keys[1:] {
				completion := dispatchCatalogKey(
					t,
					scene,
					expletives.KeyEventPress,
					key,
					&request,
				)
				if index < len(test.keys)-2 {
					if completion.Command != "" ||
						completion.Outcome != expletives.OutcomeApplied {
						t.Fatalf(
							"open submenu %q completion = %+v",
							key,
							completion,
						)
					}
					continue
				}
				if completion.Command != test.command ||
					completion.Outcome != test.outcome {
					t.Fatalf(
						"final completion = %+v, want command=%q outcome=%q",
						completion,
						test.command,
						test.outcome,
					)
				}
			}
			if test.screen != "" && scene.ActiveScreen() != test.screen {
				t.Fatalf(
					"active screen = %q, want %q",
					scene.ActiveScreen(),
					test.screen,
				)
			}
		})
	}
}

func TestCatalogEveryDisabledMenuEntryIsSelectableButInactive(t *testing.T) {
	for _, test := range disabledCatalogMenuEntries {
		t.Run(test.key, func(t *testing.T) {
			scene, err := New(expletives.Size{Width: 100, Height: 30}, "")
			if err != nil {
				t.Fatal(err)
			}
			request := 0
			openCatalogMenu(t, scene, test.root, &request)
			dispatchCatalogKey(
				t,
				scene,
				expletives.KeyEventPress,
				expletives.KeyHome,
				&request,
			)
			for range test.down {
				dispatchCatalogKey(
					t,
					scene,
					expletives.KeyEventPress,
					expletives.KeyDown,
					&request,
				)
			}
			details := catalogMenuDetails(t, scene.App.Snapshot())
			if got := lastPathKey(details.SelectedPath); got != test.key {
				t.Fatalf("selected entry = %q, want %q", got, test.key)
			}
			entry, ok := catalogMenuEntry(details, test.key)
			if !ok || entry.Enabled || entry.DisabledReason == "" {
				t.Fatalf("disabled entry details = %+v", entry)
			}
			completion := dispatchCatalogKey(
				t,
				scene,
				expletives.KeyEventPress,
				expletives.KeyEnter,
				&request,
			)
			if completion.Command != "" ||
				completion.Outcome != expletives.OutcomeNoOp {
				t.Fatalf("disabled activation completion = %+v", completion)
			}
			details = catalogMenuDetails(t, scene.App.Snapshot())
			if got := lastPathKey(details.SelectedPath); got != test.key {
				t.Fatalf(
					"disabled activation moved selection to %q",
					got,
				)
			}
		})
	}
}

func TestCatalogMenuStructureAndSeparatorTraversal(t *testing.T) {
	scene, err := New(expletives.Size{Width: 100, Height: 30}, "")
	if err != nil {
		t.Fatal(err)
	}
	details := catalogMenuDetails(t, scene.App.Snapshot())
	if got := len(details.Entries); got != 47 {
		t.Fatalf("menu entries = %d, want 47", got)
	}

	coveredCommands := make(map[string]bool)
	for _, path := range enabledCatalogMenuPaths {
		coveredCommands[path.key] = true
	}
	for _, entry := range disabledCatalogMenuEntries {
		coveredCommands[entry.key] = true
	}
	separatorKeys := make(map[string]bool)
	submenuKeys := make(map[string]bool)
	for _, entry := range details.Entries {
		switch entry.Kind {
		case expletives.MenuItemCommand:
			if !coveredCommands[entry.Key] {
				t.Errorf("command entry %q has no exhaustive path case", entry.Key)
			}
		case expletives.MenuItemSeparator:
			separatorKeys[entry.Key] = true
			if entry.Enabled || entry.Command != "" ||
				entry.Mnemonic != "" || entry.ChildCount != 0 {
				t.Errorf("separator %q has actionable state: %+v", entry.Key, entry)
			}
		case expletives.MenuItemSubmenu:
			submenuKeys[entry.Key] = true
			if !entry.Enabled || entry.ChildCount == 0 {
				t.Errorf("submenu %q has no sensible child surface", entry.Key)
			}
		default:
			t.Errorf("entry %q has unknown kind %q", entry.Key, entry.Kind)
		}
	}
	if len(coveredCommands) != 30 || len(separatorKeys) != 8 ||
		len(submenuKeys) != 9 {
		t.Fatalf(
			"coverage commands=%d separators=%d submenus=%d",
			len(coveredCommands),
			len(separatorKeys),
			len(submenuKeys),
		)
	}

	traversals := []struct {
		name string
		root expletives.Key
		want []string
	}{
		{"file", "i", []string{"menu.file.home", "menu.file.quit"}},
		{"panels", "n", []string{"menu.panels.core", "menu.panels.styles", "menu.panels.stacking", "menu.panels.scrollbars"}},
		{"layouts", "a", []string{"menu.layouts.box", "menu.layouts.grid", "menu.layouts.stacking", "menu.layouts.absolute"}},
		{"controls", "c", []string{"menu.controls.text", "menu.controls.actions", "menu.controls.status", "menu.controls.headers", "menu.controls.selection", "menu.controls.input", "menu.controls.progress", "menu.controls.navigation", "menu.controls.scrolling", "menu.controls.collections"}},
		{"menus", "m", []string{"menu.menus.overview", "menu.menus.panel", "menu.menus.context"}},
	}
	for _, traversal := range traversals {
		t.Run(traversal.name, func(t *testing.T) {
			current, newErr := New(
				expletives.Size{Width: 100, Height: 30},
				"",
			)
			if newErr != nil {
				t.Fatal(newErr)
			}
			request := 0
			openCatalogMenu(t, current, traversal.root, &request)
			got := []string{
				lastPathKey(
					catalogMenuDetails(t, current.App.Snapshot()).SelectedPath,
				),
			}
			for range len(traversal.want) - 1 {
				dispatchCatalogKey(
					t,
					current,
					expletives.KeyEventPress,
					expletives.KeyDown,
					&request,
				)
				got = append(
					got,
					lastPathKey(
						catalogMenuDetails(
							t,
							current.App.Snapshot(),
						).SelectedPath,
					),
				)
			}
			if fmt.Sprint(got) != fmt.Sprint(traversal.want) {
				t.Fatalf("selectable traversal = %v, want %v", got, traversal.want)
			}
		})
	}
}

func openCatalogMenu(
	t *testing.T,
	scene *Scene,
	root expletives.Key,
	request *int,
) {
	t.Helper()
	dispatchCatalogKey(
		t,
		scene,
		expletives.KeyEventDown,
		expletives.KeyAlt,
		request,
	)
	dispatchCatalogKey(
		t,
		scene,
		expletives.KeyEventPress,
		root,
		request,
	)
	dispatchCatalogKey(
		t,
		scene,
		expletives.KeyEventUp,
		expletives.KeyAlt,
		request,
	)
	details := catalogMenuDetails(t, scene.App.Snapshot())
	if len(details.OpenPath) != 1 {
		t.Fatalf("root %q open path = %v", root, details.OpenPath)
	}
}

func dispatchCatalogKey(
	t *testing.T,
	scene *Scene,
	kind expletives.KeyEventKind,
	key expletives.Key,
	request *int,
) expletives.Completion {
	t.Helper()
	*request++
	completion, err := scene.App.DispatchKey(
		context.Background(),
		"menu-audit",
		fmt.Sprintf("key-%d", *request),
		expletives.KeyEvent{Kind: kind, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%s %s) error = %v", kind, key, err)
	}
	return completion
}

func catalogMenuDetails(
	t *testing.T,
	snapshot expletives.Snapshot,
) *expletives.MenuBarDetails {
	t.Helper()
	for _, control := range snapshot.Controls {
		if control.Key == "menu.main" && control.Details.MenuBar != nil {
			return control.Details.MenuBar
		}
	}
	t.Fatal("menu.main typed details are absent")
	return nil
}

func catalogMenuEntry(
	details *expletives.MenuBarDetails,
	key string,
) (expletives.MenuEntryDetails, bool) {
	for _, entry := range details.Entries {
		if entry.Key == key {
			return entry, true
		}
	}
	return expletives.MenuEntryDetails{}, false
}

func lastPathKey(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[len(path)-1]
}
