package expletives_test

import (
	"context"
	"fmt"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

func Example() {
	theme, err := expletives.NewTheme(
		expletives.Style{
			ID:         "app",
			Foreground: expletives.RGB(255, 255, 255),
			Background: expletives.RGB(0, 0, 0),
		},
		expletives.Style{
			ID:         "content",
			Foreground: expletives.RGB(255, 255, 255),
			Background: expletives.RGB(0, 40, 90),
		},
	)
	if err != nil {
		panic(err)
	}
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:      expletives.Size{Width: 20, Height: 5},
		Theme:     theme,
		RootStyle: "app",
		Scenario:  "example.mvc",
	})
	if err != nil {
		panic(err)
	}

	update := app.NewTransaction()
	panel, err := update.NewPanel(app.Root(), expletives.PanelOptions{
		AutomationKey: "content",
		Bounds:        expletives.Rect{X: 1, Y: 1, Width: 18, Height: 3},
		Style:         "content",
	})
	if err != nil {
		panic(err)
	}
	if err := update.Commit(context.Background()); err != nil {
		panic(err)
	}

	snapshot := app.Snapshot()
	fmt.Println(len(snapshot.Controls), panel.Style())
	// Output:
	// 2 content
}

func ExampleApp_SetCommandRouter() {
	app, err := expletives.NewApp(expletives.AppOptions{
		Size:     expletives.Size{Width: 1, Height: 1},
		Scenario: "example.commands",
	})
	if err != nil {
		panic(err)
	}
	if err := app.RegisterCommand(expletives.CommandDefinition{
		ID:          "counter.increment",
		Description: "Increment the application counter",
		Enabled:     true,
		Automation:  true,
	}); err != nil {
		panic(err)
	}

	counter := 0 // application model state, not a toolkit type
	if err := app.SetCommandRouter(func(
		ctx context.Context,
		command expletives.Command,
	) expletives.CommandResult {
		if err := ctx.Err(); err != nil {
			return expletives.CommandResult{
				Outcome: expletives.OutcomeCancelled,
				Code:    "cancelled",
				Message: "increment cancelled",
				Cause:   err,
			}
		}
		if command.ID != "counter.increment" {
			return expletives.CommandResult{Outcome: expletives.OutcomeRejected}
		}
		counter++
		return expletives.CommandResult{Outcome: expletives.OutcomeApplied}
	}); err != nil {
		panic(err)
	}
	completion, err := app.InvokeCommand(
		context.Background(),
		"example",
		"increment-1",
		"counter.increment",
		"",
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(counter, completion.Outcome)
	// Output:
	// 1 applied
}
