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
	{"menu.controls.selection", []expletives.Key{"c", "e"}, CommandSelection, expletives.OutcomeApplied, CommandSelection, nil},
	{"menu.controls.input", []expletives.Key{"c", "n"}, CommandTextInput, expletives.OutcomeApplied, CommandTextInput, nil},
	{"menu.sections.status", []expletives.Key{"s", "s"}, CommandStatusBar, expletives.OutcomeApplied, "", nil},
	{"menu.sections.headers.show", []expletives.Key{"s", "h", "s"}, CommandHeadersShow, expletives.OutcomeApplied, "", []string{"menu.sections.headers"}},
	{"menu.sections.headers.add", []expletives.Key{"s", "h", "a"}, CommandHeadersAdd, expletives.OutcomeApplied, "", []string{"menu.sections.headers"}},
	{"menu.sections.headers.remove_highest", []expletives.Key{"s", "h", "h"}, CommandHeadersRemoveTop, expletives.OutcomeApplied, "", []string{"menu.sections.headers"}},
	{"menu.sections.headers.remove_lowest", []expletives.Key{"s", "h", "l"}, CommandHeadersRemoveLow, expletives.OutcomeApplied, "", []string{"menu.sections.headers"}},
	{"menu.sections.footers.global", []expletives.Key{"s", "f", "g"}, CommandFooterGlobalShow, expletives.OutcomeApplied, "", []string{"menu.sections.footers"}},
	{"menu.sections.footers.screen", []expletives.Key{"s", "f", "s"}, CommandFooterScreenShow, expletives.OutcomeApplied, "", []string{"menu.sections.footers"}},
	{"menu.sections.footers.focus", []expletives.Key{"s", "f", "f"}, CommandFooterFocusShow, expletives.OutcomeApplied, "", []string{"menu.sections.footers"}},
	{"menu.menus.overview", []expletives.Key{"m", "o"}, CommandViewMenus, expletives.OutcomeApplied, CommandViewMenus, nil},
	{"menu.help.about", []expletives.Key{"p", "a"}, CommandViewAbout, expletives.OutcomeApplied, CommandViewAbout, nil},
}

type disabledMenuEntry struct {
	key  string
	root expletives.Key
	down int
}

var disabledCatalogMenuEntries = []disabledMenuEntry{
	{"menu.file.automation_notice", "i", 1},
	{"menu.panels.scrollbars", "n", 3},
	{"menu.layouts.absolute", "a", 3},
	{"menu.controls.progress", "c", 4},
	{"menu.controls.navigation", "c", 5},
	{"menu.controls.scrolling", "c", 6},
	{"menu.controls.collections", "c", 7},
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
	if got := len(details.Entries); got != 58 {
		t.Fatalf("menu entries = %d, want 58", got)
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
	if len(coveredCommands) != 37 || len(separatorKeys) != 9 ||
		len(submenuKeys) != 12 {
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
		{"file", "i", []string{"menu.file.home", "menu.file.automation_notice", "menu.file.quit"}},
		{"panels", "n", []string{"menu.panels.core", "menu.panels.styles", "menu.panels.stacking", "menu.panels.scrollbars"}},
		{"layouts", "a", []string{"menu.layouts.box", "menu.layouts.grid", "menu.layouts.stacking", "menu.layouts.absolute"}},
		{"controls", "c", []string{"menu.controls.text", "menu.controls.actions", "menu.controls.selection", "menu.controls.input", "menu.controls.progress", "menu.controls.navigation", "menu.controls.scrolling", "menu.controls.collections"}},
		{"sections", "s", []string{"menu.sections.status", "menu.sections.headers", "menu.sections.footers"}},
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

func TestApplicationChromeMenuCommandsAreIndependentTogglesAndMutations(
	t *testing.T,
) {
	scene, err := New(expletives.Size{Width: 80, Height: 24}, "")
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(id expletives.CommandID) expletives.Snapshot {
		t.Helper()
		completion, invokeErr := scene.App.InvokeCommand(
			context.Background(),
			"chrome-test",
			"request-"+string(id),
			id,
			"",
		)
		if invokeErr != nil || completion.Outcome != expletives.OutcomeApplied {
			t.Fatalf("InvokeCommand(%q) = %+v, %v", id, completion, invokeErr)
		}
		return scene.App.Snapshot()
	}
	control := func(
		snapshot expletives.Snapshot,
		key string,
	) expletives.ControlSnapshot {
		t.Helper()
		for _, current := range snapshot.Controls {
			if current.Key == key {
				return current
			}
		}
		t.Fatalf("control %q is absent", key)
		return expletives.ControlSnapshot{}
	}
	menuChecked := func(
		snapshot expletives.Snapshot,
		key string,
	) bool {
		t.Helper()
		entry, ok := catalogMenuEntry(catalogMenuDetails(t, snapshot), key)
		if !ok {
			t.Fatalf("menu entry %q is absent", key)
		}
		return entry.Checked
	}

	initial := scene.App.Snapshot()
	if !control(initial, "status.main").Visible ||
		!menuChecked(initial, "menu.sections.status") ||
		control(initial, "header.primary").Visible ||
		!control(initial, "footer.hotkeys.global").Visible ||
		!control(initial, "footer.hotkeys.screen").Visible ||
		!control(initial, "footer.guidance.focus").Visible ||
		menuChecked(initial, "menu.sections.headers.show") ||
		!menuChecked(initial, "menu.sections.footers.global") ||
		!menuChecked(initial, "menu.sections.footers.screen") ||
		!menuChecked(initial, "menu.sections.footers.focus") {
		t.Fatal("initial independent application-chrome state is invalid")
	}
	hiddenStatus := invoke(CommandStatusBar)
	if control(hiddenStatus, "status.main").Visible ||
		menuChecked(hiddenStatus, "menu.sections.status") {
		t.Fatal("Status Bar toggle did not hide and uncheck")
	}
	shownStatus := invoke(CommandStatusBar)
	if !control(shownStatus, "status.main").Visible ||
		!menuChecked(shownStatus, "menu.sections.status") {
		t.Fatal("Status Bar toggle did not restore and check")
	}

	shownHeaders := invoke(CommandHeadersShow)
	if !control(shownHeaders, "header.primary").Visible ||
		!menuChecked(shownHeaders, "menu.sections.headers.show") {
		t.Fatal("Header Show toggle did not act independently")
	}
	addedHeader := invoke(CommandHeadersAdd)
	if !control(addedHeader, "header.dynamic.1").Visible {
		t.Fatal("Add did not create a visible Header under the checked policy")
	}
	invoke(CommandHeadersRemoveTop)
	afterHeaderRemoval := scene.App.Snapshot()
	if _, found := scene.App.ControlByAutomationKey("header.primary"); found {
		t.Fatal("Remove Highest did not remove the physically highest Header")
	}
	if !control(afterHeaderRemoval, "header.secondary").Visible {
		t.Fatal("Header removal damaged the remaining visible family")
	}
	for _, test := range []struct {
		command expletives.CommandID
		key     string
		menu    string
	}{
		{CommandFooterGlobalShow, "footer.hotkeys.global", "menu.sections.footers.global"},
		{CommandFooterScreenShow, "footer.hotkeys.screen", "menu.sections.footers.screen"},
		{CommandFooterFocusShow, "footer.guidance.focus", "menu.sections.footers.focus"},
	} {
		snapshot := invoke(test.command)
		if control(snapshot, test.key).Visible ||
			menuChecked(snapshot, test.menu) {
			t.Fatalf("%q Footer toggle did not hide and uncheck", test.key)
		}
	}
}

func TestSelectionCatalogRawKeyboardAndReset(t *testing.T) {
	scene, err := New(expletives.Size{Width: 100, Height: 30}, "")
	if err != nil {
		t.Fatal(err)
	}
	open, err := scene.App.InvokeCommand(
		context.Background(),
		"selection-audit",
		"open-selection",
		CommandSelection,
		"",
	)
	if err != nil || open.Outcome != expletives.OutcomeApplied {
		t.Fatalf("open Selection = %+v, %v", open, err)
	}
	control := func(key string) expletives.ControlSnapshot {
		t.Helper()
		for _, current := range scene.App.Snapshot().Controls {
			if current.Key == key {
				return current
			}
		}
		t.Fatalf("control %q is absent", key)
		return expletives.ControlSnapshot{}
	}
	dispatch := func(request string, key expletives.Key) expletives.Completion {
		t.Helper()
		completion, dispatchErr := scene.App.DispatchKey(
			context.Background(),
			"selection-audit",
			request,
			expletives.KeyEvent{
				Kind: expletives.KeyEventPress,
				Key:  key,
			},
		)
		if dispatchErr != nil {
			t.Fatalf("DispatchKey(%s) error = %v", key, dispatchErr)
		}
		return completion
	}
	if !control("selection.checkbox.two_state").Focused {
		t.Fatal("Selection screen did not focus its first eligible control")
	}
	if completion := dispatch("check", expletives.KeySpace); completion.Command != CommandSelectionChanged ||
		control(
			"selection.checkbox.two_state",
		).Details.Checkbox.State != expletives.CheckChecked {
		t.Fatalf("Checkbox Space completion = %+v", completion)
	}
	dispatch("down-three-state", expletives.KeyDown)
	dispatch("cycle-three-state", expletives.KeySpace)
	if got := control(
		"selection.checkbox.three_state",
	).Details.Checkbox.State; got != expletives.CheckUnchecked {
		t.Fatalf("three-state Checkbox after Space = %q", got)
	}
	dispatch("tab-radio", expletives.KeyTab)
	if !control("selection.radio.one").Focused {
		t.Fatal("Tab did not treat RadioGroup as one selected-option stop")
	}
	dispatch("radio-next", expletives.KeyDown)
	if group := control(
		"selection.radio.group",
	).Details.RadioGroup; group.Value != "one" ||
		!control("selection.radio.two").Focused {
		t.Fatalf(
			"RadioGroup Down value = %q, option-two focus=%t",
			group.Value,
			control("selection.radio.two").Focused,
		)
	}
	dispatch("radio-select", expletives.KeyEnter)
	if group := control(
		"selection.radio.group",
	).Details.RadioGroup; group.Value != "two" {
		t.Fatalf("RadioGroup Enter value = %q", group.Value)
	}
	dispatch("tab-cycle", expletives.KeyTab)
	dispatch("cycle-next", expletives.KeyRightBracket)
	if field := control(
		"selection.cycle.wrap",
	).Details.ChoiceField; field.Value != "charlie" {
		t.Fatalf("CycleField Right value = %q", field.Value)
	}
	dispatch("down-select", expletives.KeyDown)
	dispatch("select-next", expletives.KeyRightBracket)
	if field := control(
		"selection.select.clamp",
	).Details.ChoiceField; field.Value != "high" {
		t.Fatalf("SelectField End value = %q", field.Value)
	}

	reset, err := scene.App.InvokeCommand(
		context.Background(),
		"selection-audit",
		"reset-selection",
		CommandScenarioReset,
		"",
	)
	if err != nil || reset.Outcome != expletives.OutcomeApplied {
		t.Fatalf("Selection reset = %+v, %v", reset, err)
	}
	if control(
		"selection.checkbox.two_state",
	).Details.Checkbox.State != expletives.CheckUnchecked ||
		control(
			"selection.checkbox.three_state",
		).Details.Checkbox.State != expletives.CheckIndeterminate ||
		control("selection.radio.group").Details.RadioGroup.Value != "one" ||
		control("selection.cycle.wrap").Details.ChoiceField.Value != "alpha" ||
		control("selection.select.clamp").Details.ChoiceField.Value != "low" {
		t.Fatal("Scenario Reset did not restore Selection initial values")
	}
}

func TestTextInputCatalogValidationPasswordAndReset(t *testing.T) {
	scene, err := New(expletives.Size{Width: 100, Height: 30}, "")
	if err != nil {
		t.Fatal(err)
	}
	open, err := scene.App.InvokeCommand(
		context.Background(),
		"input-audit",
		"open-input",
		CommandTextInput,
		"",
	)
	if err != nil || open.Outcome != expletives.OutcomeApplied {
		t.Fatalf("open Input = %+v, %v", open, err)
	}
	control := func(key string) expletives.ControlSnapshot {
		t.Helper()
		for _, current := range scene.App.Snapshot().Controls {
			if current.Key == key {
				return current
			}
		}
		t.Fatalf("control %q is absent", key)
		return expletives.ControlSnapshot{}
	}
	dispatch := func(request string, key expletives.Key) expletives.Completion {
		t.Helper()
		completion, dispatchErr := scene.App.DispatchKey(
			context.Background(),
			"input-audit",
			request,
			expletives.KeyEvent{
				Kind: expletives.KeyEventPress,
				Key:  key,
			},
		)
		if dispatchErr != nil {
			t.Fatalf("DispatchKey(%s) error = %v", key, dispatchErr)
		}
		return completion
	}
	if !control("input.text.plain").Focused {
		t.Fatal("Input screen did not focus its first TextField")
	}
	dispatch("plain-edit", expletives.KeyEnter)
	dispatch("plain-type", "!")
	if completion := dispatch("plain-commit", expletives.KeyEnter); completion.Command != CommandTextChanged {
		t.Fatalf("plain commit = %+v", completion)
	}
	if details := control("input.text.plain").Details.TextField; details == nil || details.Text != "Edit me!" || details.Editing {
		t.Fatalf("plain details after commit = %#v", details)
	}

	dispatch("tab-soft", expletives.KeyTab)
	if !control("input.text.soft_whitelist").Focused {
		t.Fatal("Tab did not cross to the Soft Whitelist group")
	}
	dispatch("soft-edit", expletives.KeyEnter)
	dispatch("soft-invalid", "x")
	if details := control("input.text.soft_whitelist").Details.TextField; details == nil || details.Valid || details.Text != "ABC-123x" {
		t.Fatalf("soft-invalid details = %#v", details)
	}
	dispatch("soft-commit", expletives.KeyEnter)

	dispatch("tab-hard", expletives.KeyTab)
	dispatch("hard-edit", expletives.KeyEnter)
	before := control("input.text.hard_blacklist").Details.TextField
	if completion := dispatch("hard-reject", ":"); completion.Outcome != expletives.OutcomeNoOp {
		t.Fatalf("hard-invalid completion = %+v", completion)
	}
	after := control("input.text.hard_blacklist").Details.TextField
	if before == nil || after == nil || before.Length != after.Length {
		t.Fatalf("hard-invalid changed details from %#v to %#v", before, after)
	}
	dispatch("hard-cancel", expletives.KeyEscape)

	dispatch("tab-password", expletives.KeyTab)
	password := control("input.text.password").Details.TextField
	if password == nil || !password.Password || !password.Redacted ||
		password.Text != "" || password.Length != len("secret") {
		t.Fatalf("password details = %#v", password)
	}
	dispatch("tab-number", expletives.KeyTab)
	number := control("input.number.ranged").Details.NumberField
	if number == nil || !control("input.number.ranged").Focused ||
		number.Value != 12.5 || number.DecimalPlaces != 1 ||
		number.Minimum == nil || *number.Minimum != 0 ||
		number.Maximum == nil || *number.Maximum != 20 {
		t.Fatalf("NumberField details = %#v", number)
	}
	dispatch("tab-spin", expletives.KeyTab)
	if completion := dispatch("spin-increment", expletives.KeyRightBracket); completion.Command != CommandNumberChanged {
		t.Fatalf("SpinBox increment = %+v", completion)
	}
	spin := control("input.spin.clamped").Details.NumberField
	if spin == nil || spin.Value != 1.5 || spin.Step != 0.5 {
		t.Fatalf("SpinBox details = %#v", spin)
	}

	reset, err := scene.App.InvokeCommand(
		context.Background(),
		"input-audit",
		"reset-input",
		CommandScenarioReset,
		"",
	)
	if err != nil || reset.Outcome != expletives.OutcomeApplied {
		t.Fatalf("Input reset = %+v, %v", reset, err)
	}
	if details := control("input.text.plain").Details.TextField; details == nil || details.Text != "Edit me" {
		t.Fatalf("Scenario Reset left plain details = %#v", details)
	}
	if details := control("input.text.soft_whitelist").Details.TextField; details == nil || details.Text != "ABC-123" || !details.Valid {
		t.Fatalf("Scenario Reset left soft details = %#v", details)
	}
	if details := control("input.number.ranged").Details.NumberField; details == nil || details.Value != 12.5 {
		t.Fatalf("Scenario Reset left NumberField details = %#v", details)
	}
	if details := control("input.spin.clamped").Details.NumberField; details == nil || details.Value != 1 {
		t.Fatalf("Scenario Reset left SpinBox details = %#v", details)
	}
}

func TestAutomationNoticeDefaultsToStatusBarAndMenuToggles(t *testing.T) {
	scene, err := New(
		expletives.Size{Width: 100, Height: 30},
		"/tmp/expletives-test.sock",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertNotice := func(wantVisible bool, wantChecked bool) {
		t.Helper()
		snapshot := scene.App.Snapshot()
		var noticeVisible bool
		var hotkeysParent expletives.ControlID
		var screenFooterID expletives.ControlID
		for _, control := range snapshot.Controls {
			switch control.Key {
			case "automation.status":
				t.Fatal("dedicated automation status panel still exists")
			case "footer.hotkeys.screen":
				screenFooterID = control.ID
			case "action.hotkeys":
				hotkeysParent = control.Parent
			case "status.main":
				if control.Details.StatusBar == nil {
					t.Fatal("status.main has no StatusBar details")
				}
				for _, segment := range control.Details.StatusBar.Segments {
					if segment.Key == "automation" {
						noticeVisible =
							segment.Label == "UNAUTHENTICATED AUTOMATION ENABLED"
					}
				}
			}
		}
		if noticeVisible != wantVisible {
			t.Fatalf(
				"automation notice visible = %t, want %t",
				noticeVisible,
				wantVisible,
			)
		}
		if screenFooterID == "" || hotkeysParent != screenFooterID {
			t.Fatalf(
				"HotkeyBar parent = %q, screen Footer = %q",
				hotkeysParent,
				screenFooterID,
			)
		}
		entry, ok := catalogMenuEntry(
			catalogMenuDetails(t, snapshot),
			"menu.file.automation_notice",
		)
		if !ok || !entry.Enabled || entry.Checked != wantChecked {
			t.Fatalf("automation notice Menu entry = %+v", entry)
		}
	}

	assertNotice(true, true)
	request := 0
	for _, wantVisible := range []bool{false, true} {
		openCatalogMenu(t, scene, "i", &request)
		completion := dispatchCatalogKey(
			t,
			scene,
			expletives.KeyEventPress,
			"a",
			&request,
		)
		if completion.Command != CommandAutomationNotice ||
			completion.Outcome != expletives.OutcomeApplied {
			t.Fatalf("automation notice toggle completion = %+v", completion)
		}
		assertNotice(wantVisible, wantVisible)
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
