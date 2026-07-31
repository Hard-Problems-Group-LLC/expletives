// Package demo composes the stable expletives-test public-API fixture.
package demo

import (
	"context"
	"errors"
	"fmt"
	"sync"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

const (
	ScenarioID                                = "layouts.basic"
	CommandFixtureToggle expletives.CommandID = "fixture.toggle"
	CommandPanelRaise    expletives.CommandID = "layout.panel.raise"
	CommandPanelLower    expletives.CommandID = "layout.panel.lower"
	CommandLayerRaise    expletives.CommandID = "layout.layer.raise"
	CommandLayerLower    expletives.CommandID = "layout.layer.lower"
	CommandScenarioReset expletives.CommandID = "scenario.reset"
	CommandAppQuit       expletives.CommandID = "app.quit"
	CommandAppInterrupt  expletives.CommandID = "app.interrupt"
)

var (
	rootStyle = expletives.Style{
		ID:         "fixture.root",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x18, 0x48),
	}
	canvasStyle = expletives.Style{
		ID:         "fixture.canvas",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	borderStyle = expletives.Style{
		ID:         "fixture.border",
		Foreground: expletives.RGB(0xFF, 0xFF, 0x55),
		Background: expletives.RGB(0x00, 0x38, 0x78),
	}
	redStyle = expletives.Style{
		ID:         "fixture.red",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0xA8, 0x00, 0x20),
	}
	greenStyle = expletives.Style{
		ID:         "fixture.green",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0x22, 0xC5, 0x5E),
	}
	magentaStyle = expletives.Style{
		ID:         "fixture.magenta",
		Foreground: expletives.RGB(0xFF, 0xFF, 0xFF),
		Background: expletives.RGB(0xB8, 0x20, 0xD0),
	}
	yellowStyle = expletives.Style{
		ID:         "fixture.yellow",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xE8, 0xC8, 0x20),
	}
	bannerStyle = expletives.Style{
		ID:         "fixture.automation",
		Foreground: expletives.RGB(0x00, 0x00, 0x00),
		Background: expletives.RGB(0xFF, 0xD7, 0x00),
	}
)

// Scene owns the first colored-Panel fixture and its small controller state.
type Scene struct {
	App *expletives.App

	mu      sync.Mutex
	accent  *expletives.Panel
	layer   *expletives.BoxLayout
	toggled bool
}

// New constructs the complete fixture through the public toolkit API.
func New(size expletives.Size, automationPath string) (*Scene, error) {
	return NewWithRootConstraints(
		size,
		automationPath,
		expletives.RootConstraints{},
	)
}

// NewWithRootConstraints constructs the fixture with an optional centered
// application-root sizing policy.
func NewWithRootConstraints(
	size expletives.Size,
	automationPath string,
	constraints expletives.RootConstraints,
) (*Scene, error) {
	theme, err := expletives.NewTheme(
		rootStyle,
		canvasStyle,
		borderStyle,
		redStyle,
		greenStyle,
		magentaStyle,
		yellowStyle,
		bannerStyle,
	)
	if err != nil {
		return nil, err
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:            size,
		RootConstraints: constraints,
		Theme:           theme,
		RootStyle:       rootStyle.ID,
		Scenario:        ScenarioID,
	})
	if err != nil {
		return nil, err
	}

	transaction := app.NewTransaction()
	outer, err := transaction.NewFrame(app.Root(), expletives.FrameOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "fixture",
			Style:         canvasStyle.ID,
		},
		Title:       "expletives Core / Presentation / Automation",
		BorderStyle: borderStyle.ID,
		BorderForm:  expletives.BorderSingle,
	})
	if err != nil {
		return nil, err
	}
	red, err := transaction.NewPanel(outer, expletives.PanelOptions{
		AutomationKey: "panel.red",
		MinimumSize:   expletives.Size{Width: 10, Height: 5},
		Style:         redStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	accent, err := transaction.NewPanel(
		outer,
		expletives.PanelOptions{
			AutomationKey: "panel.accent",
			MinimumSize:   expletives.Size{Width: 12, Height: 5},
			Style:         greenStyle.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	group, err := transaction.NewGroupBox(
		outer,
		expletives.GroupBoxOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "group",
				MinimumSize:   expletives.Size{Width: 16, Height: 7},
				Style:         canvasStyle.ID,
			},
			Title:       "Nested",
			BorderStyle: borderStyle.ID,
			BorderForm:  expletives.BorderDouble,
		},
	)
	if err != nil {
		return nil, err
	}
	nestedPanel, err := transaction.NewPanel(group, expletives.PanelOptions{
		AutomationKey: "panel.nested",
		MinimumSize:   expletives.Size{Width: 8, Height: 3},
		Style:         yellowStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	layerFrame, err := transaction.NewFrame(outer, expletives.FrameOptions{
		PanelOptions: expletives.PanelOptions{
			AutomationKey: "layers",
			MinimumSize:   expletives.Size{Width: 20, Height: 3},
			Style:         canvasStyle.ID,
		},
		Title:       "Layout stacking: green over red",
		BorderStyle: borderStyle.ID,
		BorderForm:  expletives.BorderNone,
	})
	if err != nil {
		return nil, err
	}
	layerBack, err := transaction.NewPanel(layerFrame, expletives.PanelOptions{
		AutomationKey: "layer.back",
		MinimumSize:   expletives.Size{Width: 1, Height: 1},
		Style:         redStyle.ID,
	})
	if err != nil {
		return nil, err
	}
	layerFront, err := transaction.NewPanel(layerFrame, expletives.PanelOptions{
		AutomationKey: "layer.front",
		MinimumSize:   expletives.Size{Width: 1, Height: 1},
		Style:         greenStyle.ID,
	})
	if err != nil {
		return nil, err
	}

	bannerTitle := "Automation off"
	if automationPath != "" {
		bannerTitle = "UNAUTH AUTOMATION ENABLED: " + automationPath
	}
	banner, err := transaction.NewFrame(
		outer,
		expletives.FrameOptions{
			PanelOptions: expletives.PanelOptions{
				AutomationKey: "automation.status",
				MinimumSize:   expletives.Size{Width: 20, Height: 3},
				Style:         bannerStyle.ID,
			},
			Title:       bannerTitle,
			BorderStyle: bannerStyle.ID,
			BorderForm:  expletives.BorderShadeMedium,
		},
	)
	if err != nil {
		return nil, err
	}

	grid, err := expletives.NewGridLayout(expletives.GridLayoutOptions{
		AutomationKey: "layout.presentation.grid",
		Columns:       3,
		HorizontalGap: 1,
		Border: expletives.BorderOptions{
			Form:  expletives.BorderShadeDark,
			Style: borderStyle.ID,
		},
	})
	if err != nil {
		return nil, err
	}
	rootLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.root",
			Insets: expletives.Insets{
				Top: 1, Right: 1, Bottom: 1, Left: 1,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := rootLayout.AddPanel(
		outer,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	for _, panel := range []expletives.Control{red, accent, group} {
		if err := grid.AddPanel(panel, expletives.LayoutItemOptions{}); err != nil {
			return nil, err
		}
	}
	mainLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.main",
			Gap:           0,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := mainLayout.AddLayout(
		grid,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := mainLayout.AddPanel(layerFrame, expletives.LayoutItemOptions{}); err != nil {
		return nil, err
	}
	if err := mainLayout.AddPanel(banner, expletives.LayoutItemOptions{}); err != nil {
		return nil, err
	}
	nestedLayout, err := expletives.NewBoxLayout(
		expletives.Vertical,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.group",
			Border: expletives.BorderOptions{
				Form:  expletives.BorderBlock,
				Style: borderStyle.ID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := nestedLayout.AddPanel(
		nestedPanel,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	backLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{
			AutomationKey: "layout.layer.back",
			Border: expletives.BorderOptions{
				Form:  expletives.BorderShadeLight,
				Style: borderStyle.ID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := backLayout.AddPanel(
		layerBack,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	frontLayout, err := expletives.NewBoxLayout(
		expletives.Horizontal,
		expletives.BoxLayoutOptions{AutomationKey: "layout.layer.front"},
	)
	if err != nil {
		return nil, err
	}
	if err := frontLayout.AddPanel(
		layerFront,
		expletives.LayoutItemOptions{Grow: 1},
	); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(app.Root(), rootLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(outer, mainLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(group, nestedLayout); err != nil {
		return nil, err
	}
	if err := transaction.SetLayout(layerFrame, backLayout); err != nil {
		return nil, err
	}
	if err := transaction.AddLayout(layerFrame, frontLayout); err != nil {
		return nil, err
	}
	if err := transaction.Commit(context.Background()); err != nil {
		return nil, err
	}

	scene := &Scene{
		App: app, accent: accent, layer: backLayout,
	}
	definitions := []expletives.CommandDefinition{
		{
			ID:          CommandFixtureToggle,
			Description: "Toggle the accent Panel color",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandScenarioReset,
			Description: "Reset the active demonstration scenario",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandPanelRaise,
			Description: "Raise the accent Panel among Grid Panel peers",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandPanelLower,
			Description: "Lower the accent Panel among Grid Panel peers",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandLayerRaise,
			Description: "Raise the red Layout layer above the green layer",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandLayerLower,
			Description: "Lower the red Layout layer below the green layer",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandAppQuit,
			Description: "Exit the demonstration application",
			Enabled:     true,
			Automation:  true,
		},
		{
			ID:          CommandAppInterrupt,
			Description: "Interrupt the demonstration application",
			Enabled:     true,
			Automation:  true,
		},
	}
	for _, definition := range definitions {
		if err := app.RegisterCommand(definition); err != nil {
			return nil, err
		}
	}
	if err := app.SetCommandRouter(scene.routeCommand); err != nil {
		return nil, err
	}
	bindings := []struct {
		chord   expletives.Chord
		command expletives.CommandID
	}{
		{
			chord: expletives.Chord{
				Key:       "r",
				Modifiers: []expletives.Key{expletives.KeyControl},
			},
			command: CommandFixtureToggle,
		},
		{chord: expletives.Chord{Key: "q"}, command: CommandAppQuit},
		{chord: expletives.Chord{Key: expletives.KeyEscape}, command: CommandAppQuit},
		{
			chord: expletives.Chord{
				Key:       "c",
				Modifiers: []expletives.Key{expletives.KeyControl},
			},
			command: CommandAppInterrupt,
		},
	}
	for _, binding := range bindings {
		if err := app.BindChord(
			binding.chord,
			expletives.CommandBinding{Command: binding.command},
		); err != nil {
			return nil, err
		}
	}
	return scene, nil
}

// Resize updates the surface and the root-relative fixture frame.
func (s *Scene) Resize(size expletives.Size) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.App.SetSize(size)
}

// Toggle reports whether the accent panel is in its alternate state.
func (s *Scene) Toggle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toggled
}

func (s *Scene) routeCommand(
	ctx context.Context,
	command expletives.Command,
) expletives.CommandResult {
	outcome, err := s.handleCommand(ctx, command)
	if err != nil {
		return expletives.CommandResult{
			Outcome: expletives.OutcomeFailed,
			Code:    "fixture_command_failed",
			Message: "the demonstration command failed",
			Cause:   err,
		}
	}
	return expletives.CommandResult{Outcome: outcome}
}

func (s *Scene) handleCommand(
	_ context.Context,
	command expletives.Command,
) (expletives.Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch command.ID {
	case CommandFixtureToggle:
		s.toggled = !s.toggled
		style := greenStyle.ID
		if s.toggled {
			style = magentaStyle.ID
		}
		if err := s.accent.SetStyle(style); err != nil {
			return expletives.OutcomeFailed, err
		}
		return expletives.OutcomeApplied, nil
	case CommandScenarioReset:
		changed := s.toggled
		s.toggled = false
		if err := s.accent.SetStyle(greenStyle.ID); err != nil {
			return expletives.OutcomeFailed, err
		}
		before := s.App.Snapshot().Sequence
		if err := s.accent.Raise(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if err := s.layer.Lower(); err != nil {
			return expletives.OutcomeFailed, err
		}
		changed = changed || s.App.Snapshot().Sequence != before
		if !changed {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandPanelRaise:
		before := s.App.Snapshot().Sequence
		if err := s.accent.Raise(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if s.App.Snapshot().Sequence == before {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandPanelLower:
		before := s.App.Snapshot().Sequence
		if err := s.accent.Lower(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if s.App.Snapshot().Sequence == before {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandLayerRaise:
		before := s.App.Snapshot().Sequence
		if err := s.layer.Raise(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if s.App.Snapshot().Sequence == before {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandLayerLower:
		before := s.App.Snapshot().Sequence
		if err := s.layer.Lower(); err != nil {
			return expletives.OutcomeFailed, err
		}
		if s.App.Snapshot().Sequence == before {
			return expletives.OutcomeNoOp, nil
		}
		return expletives.OutcomeApplied, nil
	case CommandAppQuit:
		return expletives.OutcomeExited, nil
	case CommandAppInterrupt:
		return expletives.OutcomeInterrupted, nil
	default:
		return expletives.OutcomeRejected, nil
	}
}

// SelfCheck validates exact Layout geometry, nesting, and stacking.
func SelfCheck() error {
	scene, err := New(expletives.Size{Width: 64, Height: 20}, "")
	if err != nil {
		return err
	}
	snapshot := scene.App.Snapshot()
	if snapshot.Version != expletives.SnapshotVersion {
		return errors.New("unexpected snapshot version")
	}
	if len(snapshot.Controls) != 10 {
		return fmt.Errorf("control count = %d, want 10", len(snapshot.Controls))
	}
	if len(snapshot.Layouts) != 6 {
		return fmt.Errorf("Layout count = %d, want 6", len(snapshot.Layouts))
	}
	checks := []struct {
		x     int
		y     int
		key   string
		color expletives.Color
	}{
		{x: 0, y: 0, key: "root", color: rootStyle.Background},
		{x: 4, y: 4, key: "panel.red", color: redStyle.Background},
		{x: 24, y: 4, key: "panel.accent", color: greenStyle.Background},
		{x: 45, y: 6, key: "panel.nested", color: yellowStyle.Background},
		{x: 4, y: 13, key: "layer.front", color: greenStyle.Background},
	}
	ids := make(map[string]expletives.ControlID)
	for _, control := range snapshot.Controls {
		ids[control.Key] = control.ID
	}
	for _, check := range checks {
		cell, ok := snapshot.Frame.Cell(check.x, check.y)
		if !ok {
			return fmt.Errorf("cell %d,%d is unavailable", check.x, check.y)
		}
		if cell.Owner != ids[check.key] || cell.Background != check.color {
			return fmt.Errorf(
				"cell %d,%d = owner %q background %s, want %q %s",
				check.x,
				check.y,
				cell.Owner,
				cell.Background,
				ids[check.key],
				check.color,
			)
		}
	}
	if err := scene.layer.Raise(); err != nil {
		return err
	}
	snapshot = scene.App.Snapshot()
	cell, ok := snapshot.Frame.Cell(4, 13)
	if !ok || cell.Owner != ids["layer.back"] ||
		cell.Background != redStyle.Background {
		return errors.New("Layout Raise did not move the complete red layer")
	}
	return nil
}
