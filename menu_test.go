package expletives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func pressKey(
	t *testing.T,
	app *App,
	requestID string,
	key Key,
) Completion {
	t.Helper()
	completion, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		requestID,
		KeyEvent{Kind: KeyEventPress, Key: key},
	)
	if err != nil {
		t.Fatalf("DispatchKey(%q) error = %v", key, err)
	}
	return completion
}

func pressChord(
	t *testing.T,
	app *App,
	requestID string,
	modifier Key,
	key Key,
) Completion {
	t.Helper()
	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		requestID+"-down",
		KeyEvent{Kind: KeyEventDown, Key: modifier},
	); err != nil {
		t.Fatalf("DispatchKey(%q down) error = %v", modifier, err)
	}
	completion := pressKey(t, app, requestID, key)
	if _, err := app.DispatchKey(
		context.Background(),
		"keyboard",
		requestID+"-up",
		KeyEvent{Kind: KeyEventUp, Key: modifier},
	); err != nil {
		t.Fatalf("DispatchKey(%q up) error = %v", modifier, err)
	}
	return completion
}

func menuBarByKey(
	t *testing.T,
	snapshot Snapshot,
	key string,
) ControlSnapshot {
	t.Helper()
	control := controlByKey(t, snapshot, key)
	if control.Details.MenuBar == nil {
		t.Fatalf("control %q has no MenuBarDetails", key)
	}
	return control
}

func buildMenuFixture(
	t *testing.T,
) (*App, *Button, *MenuBar, *sync.Mutex, *[]Command) {
	t.Helper()
	app := mustApp(t, Size{Width: 42, Height: 12})
	for _, command := range []struct {
		id      CommandID
		label   string
		enabled bool
	}{
		{"menu.open", "Open", true},
		{"menu.disabled", "Unavailable", false},
		{"menu.nested", "Nested Action", true},
		{"menu.view", "View Screen", true},
		{"button.focus", "Focus", true},
	} {
		registerActionCommand(t, app, command.id, command.label, command.enabled)
	}
	if err := app.BindChord(
		Chord{Key: "o", Modifiers: []Key{KeyControl}},
		CommandBinding{Command: "menu.open"},
	); err != nil {
		t.Fatalf("BindChord() error = %v", err)
	}
	button, err := NewButton(app.Root(), ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "button.focus",
			Bounds:        Rect{X: 2, Y: 8, Width: 14, Height: 1},
		},
		Command: "button.focus",
	})
	if err != nil {
		t.Fatalf("NewButton() error = %v", err)
	}
	nested, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "item.nested.action", Kind: MenuItemCommand,
		Command: "menu.nested", Mnemonic: "n",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(nested) error = %v", err)
	}
	file, err := NewMenu(MenuOptions{Items: []MenuItem{
		{
			Key: "item.open", Kind: MenuItemCommand,
			Command: "menu.open", Mnemonic: "o",
		},
		{Key: "item.separator", Kind: MenuItemSeparator},
		{
			Key: "item.disabled", Kind: MenuItemCommand,
			Command: "menu.disabled", Mnemonic: "d",
		},
		{
			Key: "item.advanced", Kind: MenuItemSubmenu,
			Label: "Advanced", Mnemonic: "a", Menu: nested,
		},
	}})
	if err != nil {
		t.Fatalf("NewMenu(file) error = %v", err)
	}
	view, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "item.view", Kind: MenuItemCommand,
		Command: "menu.view", Mnemonic: "v",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(view) error = %v", err)
	}
	bar, err := NewMenuBar(app.Root(), MenuBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: "menu.main",
		},
		Items: []MenuItem{
			{
				Key: "menu.file", Kind: MenuItemSubmenu,
				Label: "File", Mnemonic: "f", Menu: file,
			},
			{
				Key: "menu.view", Kind: MenuItemSubmenu,
				Label: "View", Mnemonic: "v", Menu: view,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMenuBar() error = %v", err)
	}
	var mu sync.Mutex
	commands := []Command{}
	if err := app.SetCommandRouter(func(
		_ context.Context,
		command Command,
	) CommandResult {
		mu.Lock()
		commands = append(commands, command)
		mu.Unlock()
		return CommandResult{Outcome: OutcomeApplied}
	}); err != nil {
		t.Fatalf("SetCommandRouter() error = %v", err)
	}
	return app, button, bar, &mu, &commands
}

func TestMenuConstructionSnapshotCopyAndRendering(t *testing.T) {
	t.Parallel()
	app, button, bar, _, _ := buildMenuFixture(t)
	if app.Focused() != button {
		t.Fatalf("initial focus = %#v, want Button", app.Focused())
	}
	if _, ok := any(bar).(Container); ok {
		t.Fatal("MenuBar unexpectedly implements Container")
	}
	if got := bar.MinimumSize(); got != (Size{Width: 13, Height: 1}) {
		t.Fatalf("MenuBar minimum = %+v, want 13x1", got)
	}
	items := bar.Items()
	items[0].Label = "Changed"
	if got := bar.Items()[0].Label; got != "File" {
		t.Fatalf("MenuBar.Items() exposed storage: %q", got)
	}

	snapshot := app.Snapshot()
	state := menuBarByKey(t, snapshot, "menu.main")
	if state.Focused || len(state.Details.MenuBar.Entries) != 8 ||
		len(state.Details.MenuBar.OpenPath) != 0 {
		t.Fatalf("closed MenuBar snapshot = %#v", state)
	}
	if got := rowText(snapshot, 0)[:12]; got != "  File  View" {
		t.Fatalf("MenuBar row = %q", got)
	}
	if state.Bounds != (Rect{Width: 42, Height: 1}) ||
		state.AbsoluteBounds != (Rect{Width: 42, Height: 1}) ||
		state.EffectiveClip != (Rect{Width: 42, Height: 1}) {
		t.Fatalf("MenuBar physical chrome geometry = %#v", state)
	}
	if got := cellAt(t, snapshot, 2, 0); got.Grapheme != "F" ||
		got.Style != "menu.mnemonic" {
		t.Fatalf("File mnemonic cell = %#v", got)
	}

	pressChord(t, app, "alt-file", KeyAlt, "f")
	snapshot = app.Snapshot()
	state = menuBarByKey(t, snapshot, "menu.main")
	if !state.Focused ||
		len(state.Details.MenuBar.OpenPath) != 1 ||
		state.Details.MenuBar.OpenPath[0] != "menu.file" ||
		len(state.Details.MenuBar.SelectedPath) != 2 ||
		state.Details.MenuBar.SelectedPath[1] != "item.open" {
		t.Fatalf("open MenuBar snapshot = %#v", state.Details.MenuBar)
	}
	foundBorder := false
	foundFocused := false
	for _, cell := range snapshot.Frame.Cells {
		if cell.Owner != state.ID {
			continue
		}
		foundBorder = foundBorder || cell.Grapheme == "┌"
		foundFocused = foundFocused || cell.Style == "menu.focused"
	}
	if !foundBorder || !foundFocused {
		t.Fatalf(
			"popup rendering border=%t focused-style=%t",
			foundBorder,
			foundFocused,
		)
	}
	if got := cellAt(t, snapshot, 2, 0); got.Style != "menu.focused_mnemonic" {
		t.Fatalf("focused File mnemonic cell = %#v", got)
	}
	if got := cellAt(t, snapshot, 2, 2); got.Grapheme == ">" {
		t.Fatalf("popup retained synthetic selection marker: %#v", got)
	}
}

func TestOpenMenuPreservesNonMenuGlobalAltChord(t *testing.T) {
	t.Parallel()
	app, _, _, mu, commands := buildMenuFixture(t)
	registerActionCommand(t, app, "app.quit", "Quit", true)
	if err := app.BindChord(
		Chord{Key: "x", Modifiers: []Key{KeyAlt}},
		CommandBinding{Command: "app.quit"},
	); err != nil {
		t.Fatalf("BindChord(Alt-X) error = %v", err)
	}

	pressChord(t, app, "open-file", KeyAlt, "f")
	if details := menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar; len(details.OpenPath) != 1 {
		t.Fatalf("open menu path = %v, want one root", details.OpenPath)
	}
	completion := pressChord(t, app, "quit", KeyAlt, "x")
	if completion.Command != "app.quit" ||
		completion.Outcome != OutcomeApplied {
		t.Fatalf("Alt-X completion = %+v", completion)
	}
	if details := menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar; len(details.OpenPath) != 0 {
		t.Fatalf("menu remained open after Alt-X: %v", details.OpenPath)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*commands) == 0 || (*commands)[len(*commands)-1].ID != "app.quit" {
		t.Fatalf("routed commands = %+v", *commands)
	}
}

func TestMenuBarEndPlacementRightJustifiesLabelAndAnchorsPopup(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 30, Height: 8})
	registerActionCommand(t, app, "menu.quit", "Quit", true)
	registerActionCommand(t, app, "menu.about", "About", true)
	file, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "file.quit", Kind: MenuItemCommand, Command: "menu.quit",
		Mnemonic: "q",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(file) error = %v", err)
	}
	help, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "help.about", Kind: MenuItemCommand, Command: "menu.about",
		Mnemonic: "a",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(help) error = %v", err)
	}
	bar, err := NewMenuBar(app.Root(), MenuBarOptions{
		PanelOptions: PanelOptions{AutomationKey: "menu.aligned"},
		Items: []MenuItem{
			{
				Key: "root.file", Kind: MenuItemSubmenu,
				Label: "File", Mnemonic: "f", Menu: file,
			},
			{
				Key: "root.help", Kind: MenuItemSubmenu,
				Label: "Help", Mnemonic: "h", Menu: help,
				Placement: MenuBarPlacementEnd,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewMenuBar() error = %v", err)
	}
	if got := bar.MinimumSize(); got != (Size{Width: 14, Height: 1}) {
		t.Fatalf("aligned MenuBar minimum = %+v, want 14x1", got)
	}
	snapshot := app.Snapshot()
	if got := cellAt(t, snapshot, 25, 0); got.Grapheme != "H" ||
		got.Style != "menu.mnemonic" {
		t.Fatalf("right-aligned Help mnemonic = %#v", got)
	}
	details := menuBarByKey(t, snapshot, "menu.aligned").Details.MenuBar
	if details.Entries[0].Placement != MenuBarPlacementStart ||
		details.Entries[2].Placement != MenuBarPlacementEnd {
		t.Fatalf("MenuBar placements = %#v", details.Entries)
	}
	pressChord(t, app, "open-help", KeyAlt, "h")
	snapshot = app.Snapshot()
	if got := cellAt(t, snapshot, 25, 0); got.Style !=
		"menu.focused_mnemonic" {
		t.Fatalf("focused Help mnemonic = %#v", got)
	}
	if got := cellAt(t, snapshot, 18, 1); got.Grapheme != "┌" {
		t.Fatalf("Help popup did not anchor/backset from right label: %#v", got)
	}
}

func TestDefaultThemeUsesTurboVisionMenuPaletteRoles(t *testing.T) {
	t.Parallel()
	theme := DefaultTheme()
	cases := map[StyleID]ResolvedStyle{
		"menu_bar": {
			Foreground: RGB(0x00, 0x00, 0x00),
			Background: RGB(0xAA, 0xAA, 0xAA),
		},
		"menu.mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: RGB(0xAA, 0xAA, 0xAA),
		},
		"menu.focused": {
			Foreground: RGB(0x00, 0x00, 0x00),
			Background: RGB(0x00, 0xAA, 0x00),
		},
		"menu.focused_mnemonic": {
			Foreground: RGB(0xAA, 0x00, 0x00),
			Background: RGB(0x00, 0xAA, 0x00),
		},
	}
	for id, want := range cases {
		if got, found := theme.Resolve(id); !found || got != want {
			t.Errorf("DefaultTheme().Resolve(%q) = %+v, %t; want %+v", id, got, found, want)
		}
	}
}

func TestMenuKeyboardTraversalActivationAndFocusRestore(t *testing.T) {
	t.Parallel()
	app, button, _, mu, commands := buildMenuFixture(t)

	pressChord(t, app, "open-file", KeyAlt, "f")
	pressKey(t, app, "end", KeyEnd)
	details := menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.advanced" {
		t.Fatalf("End selected %q, want item.advanced", got)
	}
	pressKey(t, app, "home", KeyHome)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.open" {
		t.Fatalf("Home selected %q, want item.open", got)
	}
	pressKey(t, app, "disabled-mnemonic", "d")
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.open" {
		t.Fatalf("disabled mnemonic selected %q", got)
	}
	pressKey(t, app, "down-skips", KeyDown)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.disabled" {
		t.Fatalf("Down selected %q, want item.disabled", got)
	}
	pressKey(t, app, "down-next", KeyDown)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.advanced" {
		t.Fatalf("second Down selected %q, want item.advanced", got)
	}
	pressKey(t, app, "right-open", KeyRight)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.OpenPath[len(details.OpenPath)-1]; got != "item.advanced" {
		t.Fatalf("Right open path = %#v", details.OpenPath)
	}
	pressKey(t, app, "left-close", KeyLeft)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if len(details.OpenPath) != 1 {
		t.Fatalf("Left open path = %#v, want root only", details.OpenPath)
	}
	pressKey(t, app, "mnemonic-submenu", "a")
	completion := pressKey(t, app, "activate-nested", KeyEnter)
	if completion.Command != "menu.nested" ||
		completion.Outcome != OutcomeApplied {
		t.Fatalf("nested activation = %+v", completion)
	}
	if app.Focused() != button {
		t.Fatalf("focus after activation = %#v, want prior Button", app.Focused())
	}
	if got := menuBarByKey(t, app.Snapshot(), "menu.main").
		Details.MenuBar.OpenPath; len(got) != 0 {
		t.Fatalf("activation left menu open: %#v", got)
	}
	mu.Lock()
	if len(*commands) != 1 ||
		(*commands)[0].ID != "menu.nested" ||
		(*commands)[0].Target != "" {
		t.Fatalf("menu routed commands = %#v", *commands)
	}
	mu.Unlock()

	pressKey(t, app, "reserved-f10", "f10")
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if len(details.OpenPath) != 0 || len(details.SelectedPath) != 0 ||
		app.Focused() != button {
		t.Fatalf("reserved F10 changed menu state: %#v", details)
	}

	pressKey(t, app, "f9-open", "f9")
	pressChord(t, app, "alt-view-switch", KeyAlt, "v")
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if details.OpenPath[0] != "menu.view" {
		t.Fatalf("Alt-V while open selected %#v", details.OpenPath)
	}
	pressKey(t, app, "escape-close", KeyEscape)
	if app.Focused() != button {
		t.Fatalf("Escape did not restore prior focus")
	}
	pressChord(t, app, "ctrl-space-open", KeyControl, KeySpace)
	details = menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if len(details.OpenPath) != 0 || len(details.SelectedPath) != 1 {
		t.Fatalf("Ctrl-Space did not activate menu bar: %#v", details)
	}
	pressKey(t, app, "down-opens", KeyDown)
	if len(menuBarByKey(t, app.Snapshot(), "menu.main").
		Details.MenuBar.OpenPath) == 0 {
		t.Fatal("Down did not open active menu")
	}
	pressKey(t, app, "f9-close", "f9")
	if app.Focused() != button {
		t.Fatal("F9 toggle did not restore focus")
	}
}

func TestMenuProgrammaticSessionAndVisibilityRepair(t *testing.T) {
	t.Parallel()
	app, button, bar, _, _ := buildMenuFixture(t)
	copied := *bar
	if err := copied.Open(); err != nil {
		t.Fatalf("copied MenuBar.Open() error = %v", err)
	}
	if app.Focused() != bar {
		t.Fatalf("programmatic open focus = %#v, want MenuBar", app.Focused())
	}
	if err := copied.Close(); err != nil {
		t.Fatalf("copied MenuBar.Close() error = %v", err)
	}
	if app.Focused() != button {
		t.Fatalf("programmatic close focus = %#v, want Button", app.Focused())
	}
	if err := bar.Open(); err != nil {
		t.Fatalf("MenuBar.Open() error = %v", err)
	}
	transaction := app.NewTransaction()
	if err := transaction.SetVisible(bar, false); err != nil {
		t.Fatalf("SetVisible(MenuBar) error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("hide MenuBar Commit() error = %v", err)
	}
	if app.Focused() != button {
		t.Fatalf("hide repair focus = %#v, want Button", app.Focused())
	}
	details := menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if len(details.OpenPath) != 0 {
		t.Fatalf("hidden MenuBar retained session: %#v", details.OpenPath)
	}
}

func TestMenuBarOwnsPhysicalTopRowAndReservesRootContent(t *testing.T) {
	t.Parallel()
	app, button, bar, _, _ := buildMenuFixture(t)
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: "layout.menu-content",
	})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(button, LayoutItemOptions{
		Grow:            1,
		HorizontalAlign: AlignStretch,
	}); err != nil {
		t.Fatalf("AddPanel(button) error = %v", err)
	}
	tx := app.NewTransaction()
	if err := tx.SetLayout(app.Root(), layout); err != nil {
		t.Fatalf("SetLayout(root) error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("layout Commit() error = %v", err)
	}
	if got := button.Bounds(); got != (Rect{Y: 0, Width: 42, Height: 11}) {
		t.Fatalf("root content bounds with MenuBar = %+v", got)
	}
	if got := controlByKey(t, app.Snapshot(), "button.focus").AbsoluteBounds; got != (Rect{Y: 1, Width: 42, Height: 11}) {
		t.Fatalf("root content absolute bounds with MenuBar = %+v", got)
	}
	if err := bar.SetVisible(false); err != nil {
		t.Fatalf("MenuBar.SetVisible(false) error = %v", err)
	}
	if got := button.Bounds(); got != (Rect{Width: 42, Height: 12}) {
		t.Fatalf("root content bounds without MenuBar = %+v", got)
	}

	if err := bar.SetBounds(Rect{Width: 1, Height: 1}); err == nil {
		t.Fatal("MenuBar.SetBounds() succeeded")
	}
	if err := bar.SetMinimumSize(Size{Width: 1, Height: 1}); err == nil {
		t.Fatal("MenuBar.SetMinimumSize() succeeded")
	}
}

func TestMenuBarRejectsOrdinaryParentAndLayoutMembership(t *testing.T) {
	t.Parallel()
	app, _, bar, _, _ := buildMenuFixture(t)
	parent := mustPanel(t, app.Root(), PanelOptions{
		AutomationKey: "ordinary.parent",
	})
	items := bar.Items()
	if _, err := NewMenuBar(parent, MenuBarOptions{
		Items: items,
	}); !errors.Is(err, ErrInvalidParent) {
		t.Fatalf("ordinary-parent MenuBar error = %v", err)
	}
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{})
	if err != nil {
		t.Fatalf("NewBoxLayout() error = %v", err)
	}
	if err := layout.AddPanel(bar, LayoutItemOptions{}); err != nil {
		t.Fatalf("AddPanel(MenuBar) staging error = %v", err)
	}
	tx := app.NewTransaction()
	if err := tx.SetLayout(app.Root(), layout); err != nil {
		t.Fatalf("SetLayout(MenuBar layout) staging error = %v", err)
	}
	if err := tx.Commit(context.Background()); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("MenuBar Layout Commit() error = %v", err)
	}
}

func TestMenuBarIgnoresRootConstraintsAndTracksSurfaceResize(t *testing.T) {
	t.Parallel()
	app, _, _, _, _ := buildMenuFixture(t)
	if err := app.SetRootConstraints(RootConstraints{
		Maximum: Size{Width: 20, Height: 8},
	}); err != nil {
		t.Fatalf("SetRootConstraints() error = %v", err)
	}
	snapshot := app.Snapshot()
	root := controlByKey(t, snapshot, "root")
	bar := menuBarByKey(t, snapshot, "menu.main")
	if root.Bounds != (Rect{X: 11, Y: 2, Width: 20, Height: 8}) ||
		bar.AbsoluteBounds != (Rect{Width: 42, Height: 1}) {
		t.Fatalf("constrained root/menu geometry = root:%+v menu:%+v", root.Bounds, bar.AbsoluteBounds)
	}
	if err := app.SetSize(Size{Width: 60, Height: 16}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	bar = menuBarByKey(t, app.Snapshot(), "menu.main")
	if bar.Bounds != (Rect{Width: 60, Height: 1}) ||
		bar.AbsoluteBounds != bar.Bounds {
		t.Fatalf("resized MenuBar geometry = %+v", bar)
	}
}

func TestMenuCommandInvalidationRetainsDisabledSelection(t *testing.T) {
	t.Parallel()
	app, _, _, _, _ := buildMenuFixture(t)
	pressChord(t, app, "open-file", KeyAlt, "f")
	if err := app.RemoveCommand("menu.open"); err != nil {
		t.Fatalf("RemoveCommand() error = %v", err)
	}
	details := menuBarByKey(t, app.Snapshot(), "menu.main").Details.MenuBar
	if got := details.SelectedPath[len(details.SelectedPath)-1]; got != "item.open" {
		t.Fatalf("retained selection = %q, want item.open", got)
	}
	for _, entry := range details.Entries {
		if entry.Key == "item.open" &&
			(entry.Enabled || entry.DisabledReason == "") {
			t.Fatalf("removed command entry = %#v", entry)
		}
	}
}

func TestMenuValidationIsAtomic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	registerActionCommand(t, app, "menu.action", "Action", true)
	if _, err := NewMenu(MenuOptions{Items: []MenuItem{
		{
			Key: "one", Kind: MenuItemCommand,
			Command: "menu.action", Mnemonic: "x",
		},
		{
			Key: "two", Kind: MenuItemCommand,
			Command: "menu.action", Mnemonic: "x",
		},
	}}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("duplicate sibling mnemonic error = %v", err)
	}
	if _, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "placed.popup", Kind: MenuItemCommand,
		Command: "menu.action", Placement: MenuBarPlacementEnd,
	}}}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("popup placement error = %v", err)
	}

	child, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "child", Kind: MenuItemCommand, Command: "menu.action",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(child) error = %v", err)
	}
	before := app.Snapshot()
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: []MenuItem{
			{
				Key: "first", Kind: MenuItemSubmenu,
				Label: "First", Menu: child,
			},
			{
				Key: "second", Kind: MenuItemSubmenu,
				Label: "Second", Menu: child,
			},
		},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("aliased Menu tree error = %v", err)
	}
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: []MenuItem{{
			Key: "placed.invalid", Kind: MenuItemSubmenu,
			Label: "Placed", Menu: child, Placement: "middle",
		}},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("invalid top-level placement error = %v", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence ||
		len(after.Controls) != len(before.Controls) {
		t.Fatal("failed aliased MenuBar construction changed App")
	}

	missing, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "missing", Kind: MenuItemCommand, Command: "missing.command",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(missing) error = %v", err)
	}
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: []MenuItem{{
			Key: "missing.root", Kind: MenuItemSubmenu,
			Label: "Missing", Menu: missing,
		}},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("missing command error = %v", err)
	}

	valid, err := NewMenu(MenuOptions{Items: []MenuItem{{
		Key: "valid", Kind: MenuItemCommand, Command: "menu.action",
	}}})
	if err != nil {
		t.Fatalf("NewMenu(valid) error = %v", err)
	}
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: []MenuItem{{
			Key: "valid.root", Kind: MenuItemSubmenu,
			Label: "Valid", Menu: valid,
		}},
	}); err != nil {
		t.Fatalf("NewMenuBar(valid) error = %v", err)
	}
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: []MenuItem{{
			Key: "other.root", Kind: MenuItemSubmenu,
			Label: "Other", Menu: valid,
		}},
	}); !errors.Is(err, ErrInvalidControl) {
		t.Fatalf("second MenuBar error = %v", err)
	}
}

func TestMenuAggregateCapacityIsAtomic(t *testing.T) {
	t.Parallel()
	app := mustApp(t, Size{Width: 20, Height: 8})
	registerActionCommand(t, app, "menu.action", "Action", true)
	roots := make([]MenuItem, 9)
	for rootIndex := range roots {
		items := make([]MenuItem, MaxMenuItemsPerMenu)
		for itemIndex := range items {
			items[itemIndex] = MenuItem{
				Key:     fmt.Sprintf("item.%d.%d", rootIndex, itemIndex),
				Kind:    MenuItemCommand,
				Command: "menu.action",
			}
		}
		menu, err := NewMenu(MenuOptions{Items: items})
		if err != nil {
			t.Fatalf("NewMenu(%d) error = %v", rootIndex, err)
		}
		roots[rootIndex] = MenuItem{
			Key:  fmt.Sprintf("root.%d", rootIndex),
			Kind: MenuItemSubmenu, Label: "Root", Menu: menu,
		}
	}
	before := app.Snapshot()
	if _, err := NewMenuBar(app.Root(), MenuBarOptions{
		Items: roots,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("aggregate Menu capacity error = %v", err)
	}
	after := app.Snapshot()
	if after.Sequence != before.Sequence ||
		len(after.Controls) != len(before.Controls) {
		t.Fatal("excess MenuBar changed App state")
	}
}

func TestMenuPopupClipsAcrossTinyResize(t *testing.T) {
	t.Parallel()
	app, _, _, _, _ := buildMenuFixture(t)
	pressChord(t, app, "open-file", KeyAlt, "f")
	for _, size := range []Size{
		{Width: 3, Height: 2},
		{Width: 1, Height: 1},
		{Width: 0, Height: 0},
		{Width: 42, Height: 12},
	} {
		if err := app.SetSize(size); err != nil {
			t.Fatalf("SetSize(%+v) error = %v", size, err)
		}
		snapshot := app.Snapshot()
		if snapshot.Frame.Size != size ||
			len(snapshot.Frame.Cells) != size.Width*size.Height {
			t.Fatalf("resized frame = %+v/%d", snapshot.Frame.Size, len(snapshot.Frame.Cells))
		}
		if len(menuBarByKey(t, snapshot, "menu.main").
			Details.MenuBar.OpenPath) == 0 {
			t.Fatalf("SetSize(%+v) closed menu session", size)
		}
	}
}

func TestChildMenuMeasuresEntriesAndBacksetsFromRightEdge(t *testing.T) {
	t.Parallel()
	app, _, _, _, _ := buildMenuFixture(t)
	if err := app.SetSize(Size{Width: 22, Height: 12}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	pressChord(t, app, "open-file", KeyAlt, "f")
	pressKey(t, app, "select-advanced", KeyEnd)
	pressKey(t, app, "open-advanced", KeyRight)
	snapshot := app.Snapshot()
	found := false
	for y := 0; y < snapshot.Frame.Size.Height; y++ {
		if strings.Contains(rowText(snapshot, y), "Nested Action") {
			found = true
			break
		}
	}
	if !found {
		rows := make([]string, snapshot.Frame.Size.Height)
		for y := range rows {
			rows[y] = rowText(snapshot, y)
		}
		t.Fatalf(
			"measured/backset child menu clipped its entry:\n%s",
			strings.Join(rows, "\n"),
		)
	}
}
