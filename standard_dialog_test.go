package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestStandardDialogTurboVisionPaletteAndButtonGeometry(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 44, Height: 16}, Scenario: "message.visual",
	})
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewMessageBox(app.Root(), MessageBoxOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "message.visual"},
			Title:        "Notice",
		}},
		Message: "Ready.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := box.Show(nil); err != nil {
		t.Fatal(err)
	}

	snapshot := app.Snapshot()
	modal := controlByKey(t, snapshot, "message.visual")
	message := controlByKey(t, snapshot, "message.visual.message")
	button := controlByKey(t, snapshot, "message.visual.ok")
	if button.AbsoluteBounds.Width < buttonMinimumWidth ||
		button.AbsoluteBounds.Height != 2 ||
		button.Details.Action == nil || button.Details.Action.Mnemonic != "k" {
		t.Fatalf("MessageBox OK geometry/details = %+v", button)
	}

	wantGray := RGB(0xAA, 0xAA, 0xAA)
	wantGreen := RGB(0x00, 0xAA, 0x00)
	assertCell := func(
		x, y int,
		grapheme string,
		style StyleID,
		foreground, background Color,
	) Cell {
		t.Helper()
		cell, found := snapshot.Frame.Cell(x, y)
		if !found || cell.Grapheme != grapheme || cell.Style != style ||
			cell.Foreground != foreground || cell.Background != background {
			t.Fatalf(
				"cell (%d,%d) = %+v, want %q %q %s/%s",
				x, y, cell, grapheme, style, foreground, background,
			)
		}
		return cell
	}

	assertCell(
		modal.AbsoluteBounds.X,
		modal.AbsoluteBounds.Y,
		"╔",
		"message_box.border",
		RGB(0xFF, 0xFF, 0xFF),
		wantGray,
	)
	assertCell(
		modal.AbsoluteBounds.X+2,
		modal.AbsoluteBounds.Y,
		"N",
		"message_box.border",
		RGB(0xFF, 0xFF, 0xFF),
		wantGray,
	)
	assertCell(
		message.AbsoluteBounds.X,
		message.AbsoluteBounds.Y,
		"R",
		"message_box",
		RGB(0x00, 0x00, 0x00),
		wantGray,
	)

	buttonX, buttonY := button.AbsoluteBounds.X, button.AbsoluteBounds.Y
	buttonRight := buttonX + button.AbsoluteBounds.Width - 1
	buttonBottom := buttonY + button.AbsoluteBounds.Height - 1
	assertCell(
		buttonRight, buttonY, "▄", "button.shadow",
		RGB(0x00, 0x00, 0x00), wantGray,
	)
	assertCell(
		buttonX+2, buttonBottom, "▀", "button.shadow",
		RGB(0x00, 0x00, 0x00), wantGray,
	)

	foundO, foundK := false, false
	for x := buttonX + 1; x < buttonRight; x++ {
		cell, _ := snapshot.Frame.Cell(x, buttonY)
		switch cell.Grapheme {
		case "O":
			foundO = true
			if cell.Style != "button.focused" ||
				cell.Foreground != RGB(0xFF, 0xFF, 0xFF) ||
				cell.Background != wantGreen {
				t.Fatalf("focused OK body cell = %+v", cell)
			}
		case "K":
			foundK = true
			if cell.Style != "button.mnemonic" ||
				cell.Foreground != RGB(0xFF, 0xFF, 0x55) ||
				cell.Background != wantGreen {
				t.Fatalf("OK mnemonic cell = %+v", cell)
			}
		}
	}
	if !foundO || !foundK {
		t.Fatalf("OK label cells not found: O=%t K=%t", foundO, foundK)
	}

	assertCell(
		modal.AbsoluteBounds.X+modal.AbsoluteBounds.Width,
		modal.AbsoluteBounds.Y+1,
		" ",
		"message_box.shadow",
		RGB(0x00, 0x00, 0x00),
		RGB(0x00, 0x00, 0x00),
	)
}

func TestEveryStandardDialogUsesRaisedButtons(t *testing.T) {
	type dialogFixture struct {
		buttons map[string]StyleID
		close   func()
	}
	tests := []struct {
		name  string
		build func(*testing.T, *App) dialogFixture
	}{
		{
			name: "MessageBox",
			build: func(t *testing.T, app *App) dialogFixture {
				box, err := NewMessageBox(app.Root(), MessageBoxOptions{
					DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
						PanelOptions: PanelOptions{AutomationKey: "message"},
					}},
					Message: "Ready.",
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := box.Show(nil); err != nil {
					t.Fatal(err)
				}
				return dialogFixture{
					buttons: map[string]StyleID{"message.ok": "button.focused"},
					close: func() {
						_ = box.Close(ModalResult{Reason: ModalAccepted})
					},
				}
			},
		},
		{
			name: "ConfirmDialog",
			build: func(t *testing.T, app *App) dialogFixture {
				dialog, err := NewConfirmDialog(app.Root(), ConfirmDialogOptions{
					DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
						PanelOptions: PanelOptions{AutomationKey: "confirm"},
					}},
					Message: "Continue?", ShowCancel: true,
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := dialog.Show(nil); err != nil {
					t.Fatal(err)
				}
				return dialogFixture{
					buttons: map[string]StyleID{
						"confirm.yes":    "button",
						"confirm.no":     "button.focused",
						"confirm.cancel": "button",
					},
					close: func() {
						_ = dialog.Close(ModalResult{Reason: ModalCancelled})
					},
				}
			},
		},
		{
			name: "InputDialog",
			build: func(t *testing.T, app *App) dialogFixture {
				dialog, err := NewInputDialog(app.Root(), InputDialogOptions{
					DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
						PanelOptions: PanelOptions{AutomationKey: "input"},
					}},
					Prompt: "Name:",
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := dialog.Show(nil); err != nil {
					t.Fatal(err)
				}
				return dialogFixture{
					buttons: map[string]StyleID{
						"input.ok":     "button.default",
						"input.cancel": "button",
					},
					close: func() {
						_ = dialog.Close(ModalResult{Reason: ModalCancelled})
					},
				}
			},
		},
		{
			name: "ProgressDialog",
			build: func(t *testing.T, app *App) dialogFixture {
				dialog, err := NewProgressDialog(app.Root(), ProgressDialogOptions{
					DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
						PanelOptions: PanelOptions{AutomationKey: "progress"},
					}},
					State: ProgressDialogState{
						Status: "Working",
						Progress: ProgressBarState{
							Current: 1, Total: 2, Status: ProgressRunning,
						},
					},
					Cancellable: true,
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := dialog.Show(nil); err != nil {
					t.Fatal(err)
				}
				return dialogFixture{
					buttons: map[string]StyleID{
						"progress.cancel": "button.focused",
					},
					close: func() {
						_ = dialog.Complete(ModalResult{Reason: ModalCancelled})
					},
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app, err := NewApp(AppOptions{
				Size: Size{Width: 60, Height: 20}, Scenario: "dialog.visuals",
			})
			if err != nil {
				t.Fatal(err)
			}
			fixture := test.build(t, app)
			defer fixture.close()
			snapshot := app.Snapshot()
			for key, bodyStyle := range fixture.buttons {
				button := controlByKey(t, snapshot, key)
				bounds := button.AbsoluteBounds
				if bounds.Width < buttonMinimumWidth || bounds.Height != 2 {
					t.Fatalf("%s bounds = %+v", key, bounds)
				}
				body, _ := snapshot.Frame.Cell(bounds.X+1, bounds.Y)
				if body.Style != bodyStyle {
					t.Fatalf("%s body style = %q, want %q", key, body.Style, bodyStyle)
				}
				right, _ := snapshot.Frame.Cell(bounds.X+bounds.Width-1, bounds.Y)
				bottom, _ := snapshot.Frame.Cell(bounds.X+2, bounds.Y+1)
				if right.Grapheme != "▄" || right.Style != "button.shadow" ||
					bottom.Grapheme != "▀" || bottom.Style != "button.shadow" {
					t.Fatalf("%s shadows = right %+v, bottom %+v", key, right, bottom)
				}
				if button.Details.Action == nil || button.Details.Action.Mnemonic == "" {
					t.Fatalf("%s has no mnemonic: %+v", key, button.Details.Action)
				}
				mnemonicFound := false
				for x := bounds.X + 1; x < bounds.X+bounds.Width-1; x++ {
					cell, _ := snapshot.Frame.Cell(x, bounds.Y)
					if cell.Style == "button.mnemonic" {
						mnemonicFound = true
						break
					}
				}
				if !mnemonicFound {
					t.Fatalf("%s did not paint its mnemonic style", key)
				}
			}
		})
	}
}

func TestMessageBoxEnterEscapeAndExactCompletionResult(t *testing.T) {
	for _, key := range []Key{KeyEnter, KeyEscape} {
		t.Run(string(key), func(t *testing.T) {
			app, err := NewApp(AppOptions{
				Size: Size{Width: 40, Height: 14}, Scenario: "message.test",
			})
			if err != nil {
				t.Fatal(err)
			}
			box, err := NewMessageBox(app.Root(), MessageBoxOptions{
				DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
					PanelOptions: PanelOptions{AutomationKey: "message"},
					Title:        "Notice",
				}},
				Message: "The operation completed successfully.",
			})
			if err != nil {
				t.Fatalf("NewMessageBox() error = %v", err)
			}
			if box.MessageControl() == nil || box.OKButton() == nil {
				t.Fatal("MessageBox child accessors returned nil")
			}
			done := box.Done()
			if err := box.Show(nil); err != nil {
				t.Fatalf("Show() error = %v", err)
			}
			if app.Focused() != box.OKButton() {
				t.Fatalf("focus = %v, want OK", app.Focused())
			}
			view := controlByKey(t, app.Snapshot(), "message")
			if view.Kind != ControlMessageBox || view.Style != "message_box" ||
				view.Details.ModalPanel == nil || !view.Details.ModalPanel.Active ||
				view.Bounds.Width < box.OKButton().MinimumSize().Width+4 {
				t.Fatalf("MessageBox snapshot = %+v", view)
			}
			completion, err := app.DispatchKey(
				context.Background(), "test", "close",
				KeyEvent{Kind: KeyEventPress, Key: key},
			)
			if err != nil || completion.Outcome != OutcomeApplied ||
				completion.Command != CommandDialogOK {
				t.Fatalf("DispatchKey() = %+v, %v", completion, err)
			}
			select {
			case <-done:
			default:
				t.Fatal("Done was not closed")
			}
			result, ready := box.Result()
			if !ready || result != (ModalResult{
				Reason: ModalAccepted, Action: CommandDialogOK,
			}) {
				t.Fatalf("Result() = %+v, %t", result, ready)
			}
			snapshot := app.Snapshot()
			if snapshot.Completion == nil ||
				snapshot.Completion.FrameSequence != completion.FrameSequence ||
				snapshot.Completion.Command != CommandDialogOK ||
				controlByKey(t, snapshot, "message").Visible {
				t.Fatalf("associated close snapshot = %+v", snapshot.Completion)
			}
		})
	}
}

func TestLongMessageBoxUsesScrollableViewport(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 24, Height: 8}, Scenario: "message.long",
	})
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewMessageBox(app.Root(), MessageBoxOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "long"},
			Title:        "Long Message",
		}},
		Message: "This deliberately long informational message must remain keyboard scrollable on a small application surface instead of disappearing below the modal clipping boundary.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if box.MessageViewport() == nil {
		t.Fatal("MessageViewport() returned nil")
	}
	if err := box.Show(nil); err != nil {
		t.Fatal(err)
	}
	viewport := controlByKey(t, app.Snapshot(), "long.viewport")
	if viewport.Details.Scrollable == nil ||
		!viewport.Details.Scrollable.VerticalVisible ||
		viewport.Details.Scrollable.State.ContentSize.Height <=
			viewport.Details.Scrollable.ViewportBounds.Height {
		t.Fatalf("long-message viewport = %+v", viewport.Details.Scrollable)
	}
	if err := box.MessageViewport().Focus(); err != nil {
		t.Fatalf("MessageViewport.Focus() error = %v", err)
	}
	completion, err := app.DispatchKey(
		context.Background(), "test", "page-down",
		KeyEvent{Kind: KeyEventPress, Key: KeyPageDown},
	)
	if err != nil || completion.Outcome != OutcomeApplied {
		t.Fatalf("PageDown = %+v, %v", completion, err)
	}
	viewport = controlByKey(t, app.Snapshot(), "long.viewport")
	if viewport.Details.Scrollable.State.Offset.Y == 0 {
		t.Fatalf("PageDown did not scroll: %+v", viewport.Details.Scrollable)
	}
}

func TestConfirmDialogSafeDefaultTypedChoicesAndCancel(t *testing.T) {
	t.Run("safe default No", func(t *testing.T) {
		app, err := NewApp(AppOptions{
			Size: Size{Width: 50, Height: 16}, Scenario: "confirm.no",
		})
		if err != nil {
			t.Fatal(err)
		}
		dialog, err := NewConfirmDialog(app.Root(), ConfirmDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "confirm"},
				Title:        "Confirm",
			}},
			Message: "Apply the destructive operation?",
		})
		if err != nil {
			t.Fatal(err)
		}
		if dialog.CancelButton() != nil {
			t.Fatal("CancelButton() is non-nil when ShowCancel is false")
		}
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		if app.Focused() != dialog.NoButton() {
			t.Fatalf("focus = %v, want safe default No", app.Focused())
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "no",
			KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
		)
		if err != nil || completion.Command != CommandDialogNo {
			t.Fatalf("Enter = %+v, %v", completion, err)
		}
		if choice, ready := dialog.Choice(); !ready || choice != ConfirmChoiceNo {
			t.Fatalf("Choice() = %q, %t", choice, ready)
		}
		result, _ := dialog.Result()
		if result.Reason != ModalAccepted {
			t.Fatalf("Result() = %+v", result)
		}
	})

	t.Run("Escape remains Cancel without visible button", func(t *testing.T) {
		app, _ := NewApp(AppOptions{
			Size: Size{Width: 40, Height: 12}, Scenario: "confirm.cancel",
		})
		dialog, err := NewConfirmDialog(app.Root(), ConfirmDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "confirm"},
			}},
			Message: "Continue?",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "cancel",
			KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
		)
		if err != nil || completion.Command != CommandDialogCancel {
			t.Fatalf("Escape = %+v, %v", completion, err)
		}
		if choice, ready := dialog.Choice(); !ready || choice != ConfirmChoiceCancel {
			t.Fatalf("Choice() = %q, %t", choice, ready)
		}
		result, _ := dialog.Result()
		if result.Reason != ModalCancelled {
			t.Fatalf("Result() = %+v", result)
		}
	})

	t.Run("explicit Yes and visible Cancel", func(t *testing.T) {
		app, _ := NewApp(AppOptions{
			Size: Size{Width: 50, Height: 14}, Scenario: "confirm.yes",
		})
		dialog, err := NewConfirmDialog(app.Root(), ConfirmDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "confirm"},
			}},
			Message: "Proceed?", Default: ConfirmChoiceYes, ShowCancel: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if dialog.CancelButton() == nil {
			t.Fatal("CancelButton() is nil")
		}
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		if app.Focused() != dialog.YesButton() {
			t.Fatalf("focus = %v, want Yes", app.Focused())
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "cancel",
			KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
		)
		if err != nil || completion.Command != CommandDialogCancel {
			t.Fatalf("Escape = %+v, %v", completion, err)
		}
	})
}

func TestStandardDialogCommandsAreImmutableAndTargetChecked(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 30, Height: 10}, Scenario: "dialog.commands",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RemoveCommand(CommandDialogOK); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("RemoveCommand(dialog.ok) error = %v", err)
	}
	box, err := NewMessageBox(app.Root(), MessageBoxOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "message"},
		}},
		Message: "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := box.Show(nil); err != nil {
		t.Fatal(err)
	}
	completion, err := app.InvokeCommand(
		context.Background(), "test", "wrong", CommandDialogYes, box.ID(),
	)
	if err != nil || completion.Outcome != OutcomeRejected ||
		completion.Code != "dialog_action" || !box.Active() {
		t.Fatalf("wrong dialog action = %+v error=%v active=%t", completion, err, box.Active())
	}
	completion, err = app.InvokeCommand(
		context.Background(), "test", "ok", CommandDialogOK, box.ID(),
	)
	if err != nil || completion.Outcome != OutcomeApplied || box.Active() {
		t.Fatalf("targeted OK = %+v error=%v active=%t", completion, err, box.Active())
	}
}

func TestInputDialogValidationAcceptanceAndCancellation(t *testing.T) {
	validator := &TextValidator{
		Enforcement: TextValidationSoft,
		Mode:        TextValidationWhitelist,
		Characters:  "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
	newDialog := func(t *testing.T, password bool) (*App, *InputDialog) {
		t.Helper()
		app, err := NewApp(AppOptions{
			Size: Size{Width: 44, Height: 14}, Scenario: "input.dialog",
		})
		if err != nil {
			t.Fatal(err)
		}
		dialog, err := NewInputDialog(app.Root(), InputDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "input"},
				Title:        "Input",
			}},
			Prompt: "Enter an alphanumeric value:",
			Text:   "seed", Validator: validator, Password: password,
		})
		if err != nil {
			t.Fatal(err)
		}
		return app, dialog
	}

	t.Run("valid Enter accepts exact local value", func(t *testing.T) {
		app, dialog := newDialog(t, false)
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		if app.Focused() != dialog.Field() || !dialog.Field().Editing() {
			t.Fatalf("focus/editing = %v/%t", app.Focused(), dialog.Field().Editing())
		}
		if _, err := app.DispatchTextInput(
			context.Background(), "test", "text",
			TextInputEvent{Kind: TextInputCommitted, Text: "42"},
		); err != nil {
			t.Fatal(err)
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "accept",
			KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
		)
		if err != nil || completion.Outcome != OutcomeApplied ||
			completion.Command != CommandDialogOK || dialog.Active() {
			t.Fatalf("Enter = %+v error=%v active=%t", completion, err, dialog.Active())
		}
		if value, ready := dialog.Value(); !ready || value != "seed42" {
			t.Fatalf("Value() = %q, %t", value, ready)
		}
	})

	t.Run("soft invalid rejects and resumes editing", func(t *testing.T) {
		app, dialog := newDialog(t, false)
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		if _, err := app.DispatchTextInput(
			context.Background(), "test", "invalid-text",
			TextInputEvent{Kind: TextInputCommitted, Text: "!"},
		); err != nil {
			t.Fatal(err)
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "invalid-accept",
			KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
		)
		if err != nil || completion.Outcome != OutcomeRejected ||
			completion.Code != "validation_failed" || !dialog.Active() ||
			!dialog.Field().Editing() || dialog.Field().Valid() {
			t.Fatalf(
				"invalid Enter = %+v error=%v active/editing/valid=%t/%t/%t",
				completion, err, dialog.Active(), dialog.Field().Editing(), dialog.Field().Valid(),
			)
		}
		if _, err := app.DispatchKey(
			context.Background(), "test", "remove-invalid",
			KeyEvent{Kind: KeyEventPress, Key: KeyBackspace},
		); err != nil {
			t.Fatal(err)
		}
		completion, err = app.DispatchKey(
			context.Background(), "test", "valid-accept",
			KeyEvent{Kind: KeyEventPress, Key: KeyEnter},
		)
		if err != nil || completion.Outcome != OutcomeApplied || dialog.Active() {
			t.Fatalf("corrected Enter = %+v error=%v", completion, err)
		}
	})

	t.Run("Escape discards working value", func(t *testing.T) {
		app, dialog := newDialog(t, true)
		if err := dialog.Show(nil); err != nil {
			t.Fatal(err)
		}
		if _, err := app.DispatchTextInput(
			context.Background(), "test", "secret",
			TextInputEvent{Kind: TextInputCommitted, Text: "secret"},
		); err != nil {
			t.Fatal(err)
		}
		completion, err := app.DispatchKey(
			context.Background(), "test", "cancel",
			KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
		)
		if err != nil || completion.Command != CommandDialogCancel || dialog.Active() {
			t.Fatalf("Escape = %+v error=%v", completion, err)
		}
		if value, ready := dialog.Value(); ready || value != "" {
			t.Fatalf("cancelled Value() = %q, %t", value, ready)
		}
		if got := dialog.Field().Text(); got != "seed" {
			t.Fatalf("cancelled field text = %q", got)
		}
	})
}

func TestProgressDialogCancellationRequestAndAcknowledgement(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 50, Height: 16}, Scenario: "progress.dialog",
	})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := NewProgressDialog(app.Root(), ProgressDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "progress"},
			Title:        "Working",
		}},
		State: ProgressDialogState{
			Status: "Starting",
			Progress: ProgressBarState{
				Indeterminate: true, Tick: 3, Status: ProgressRunning,
			},
		},
		Cancellable: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if controlByKey(t, app.Snapshot(), "progress").Kind != ControlProgressDialog ||
		dialog.CancelButton() == nil ||
		dialog.StatusControl() == nil || dialog.ProgressControl() == nil {
		t.Fatalf("progress compound cancel/status/progress = %v/%v/%v",
			dialog.CancelButton(), dialog.StatusControl(), dialog.ProgressControl())
	}
	if state := dialog.State(); state.Status != "Starting" ||
		!state.Progress.Indeterminate || state.Progress.Tick != 3 {
		t.Fatalf("initial State() = %+v", state)
	}
	operationContext := dialog.Context()
	if operationContext == nil {
		t.Fatal("Context() is nil")
	}
	if err := dialog.Show(nil); err != nil {
		t.Fatal(err)
	}
	if app.Focused() != dialog.CancelButton() {
		t.Fatalf("initial focus = %v", app.Focused())
	}

	completion, err := app.DispatchKey(
		context.Background(), "test", "cancel-request",
		KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
	)
	if err != nil || completion.Outcome != OutcomeApplied ||
		completion.Command != CommandDialogCancel || !dialog.Active() ||
		!dialog.CancelRequested() {
		t.Fatalf("Escape = %+v error=%v active/requested=%t/%t",
			completion, err, dialog.Active(), dialog.CancelRequested())
	}
	select {
	case <-operationContext.Done():
	default:
		t.Fatal("operation context remained active after cancellation request")
	}
	snapshot := app.Snapshot()
	cancelView := controlByKey(t, snapshot, "progress.cancel")
	cancelDetails := cancelView.Details.Action
	progressDetails := controlByKey(t, snapshot, "progress").Details.ProgressDialog
	if cancelDetails == nil || cancelDetails.Enabled ||
		cancelDetails.DisabledReason != "Cancellation requested" ||
		progressDetails == nil || !progressDetails.CancelRequested ||
		!progressDetails.Cancellable {
		t.Fatalf("cancel/progress details = %#v / %#v", cancelDetails, progressDetails)
	}
	disabledBody, _ := snapshot.Frame.Cell(
		cancelView.AbsoluteBounds.X+1,
		cancelView.AbsoluteBounds.Y,
	)
	disabledShadow, _ := snapshot.Frame.Cell(
		cancelView.AbsoluteBounds.X+cancelView.AbsoluteBounds.Width-1,
		cancelView.AbsoluteBounds.Y,
	)
	if disabledBody.Style != "button.disabled" ||
		disabledBody.Foreground != RGB(0x80, 0x80, 0x80) ||
		disabledBody.Background != RGB(0xAA, 0xAA, 0xAA) ||
		disabledShadow.Style != "button.shadow" ||
		disabledShadow.Grapheme != "▄" {
		t.Fatalf(
			"disabled cancel body/shadow = %+v / %+v",
			disabledBody,
			disabledShadow,
		)
	}

	repeated, err := app.InvokeCommand(
		context.Background(), "test", "repeat-cancel",
		CommandDialogCancel, dialog.ID(),
	)
	if err != nil || repeated.Outcome != OutcomeRejected ||
		repeated.Code != "cancel_requested" || !dialog.Active() {
		t.Fatalf("repeated cancellation = %+v error=%v", repeated, err)
	}
	if err := dialog.Update(context.Background(), ProgressDialogState{
		Status: "Stopping safely",
		Progress: ProgressBarState{
			Indeterminate: true, Tick: 4, Status: ProgressRunning,
		},
	}); err != nil {
		t.Fatalf("Update() after request error = %v", err)
	}
	if err := dialog.Complete(ModalResult{
		Reason: ModalCancelled, Action: CommandDialogCancel,
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if dialog.Active() {
		t.Fatal("acknowledged ProgressDialog remained active")
	}
	if result, ready := dialog.Result(); !ready ||
		result.Reason != ModalCancelled || result.Action != CommandDialogCancel {
		t.Fatalf("Result() = %+v, %t", result, ready)
	}
}

func TestProgressDialogNormalCompletionAndNonCancellableEscape(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 46, Height: 14}, Scenario: "progress.complete",
	})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := NewProgressDialog(app.Root(), ProgressDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "progress"},
		}},
		State: ProgressDialogState{
			Status: "Halfway",
			Progress: ProgressBarState{
				Current: 1, Total: 2, Status: ProgressRunning,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dialog.CancelButton() != nil || dialog.CancelRequested() {
		t.Fatalf("non-cancellable dialog cancel state = %v/%t",
			dialog.CancelButton(), dialog.CancelRequested())
	}
	operationContext := dialog.Context()
	if err := dialog.Show(nil); err != nil {
		t.Fatal(err)
	}
	completion, err := app.DispatchKey(
		context.Background(), "test", "escape",
		KeyEvent{Kind: KeyEventPress, Key: KeyEscape},
	)
	if err != nil || completion.Outcome != OutcomeNoOp ||
		completion.Command != "" || !dialog.Active() {
		t.Fatalf("non-cancellable Escape = %+v error=%v", completion, err)
	}
	if err := dialog.SetState(ProgressDialogState{
		Status: "Complete",
		Progress: ProgressBarState{
			Current: 2, Total: 2, Status: ProgressCompleted,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := dialog.Complete(ModalResult{Reason: ModalAccepted}); err != nil {
		t.Fatal(err)
	}
	if dialog.CancelRequested() {
		t.Fatal("normal completion set CancelRequested")
	}
	select {
	case <-operationContext.Done():
	default:
		t.Fatal("operation context remained active after terminal completion")
	}
	if err := dialog.SetState(ProgressDialogState{Status: "late"}); !errors.Is(err, ErrModalState) {
		t.Fatalf("closed SetState() error = %v", err)
	}
	if err := validateModalResult(ModalResult{Reason: ModalFailed}); err != nil {
		t.Fatalf("ModalFailed validation error = %v", err)
	}
}

func TestProgressDialogConcurrentUpdatesAndCompletion(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 48, Height: 15}, Scenario: "progress.concurrent",
	})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := NewProgressDialog(app.Root(), ProgressDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "progress"},
		}},
		State: ProgressDialogState{
			Status:   "Starting",
			Progress: ProgressBarState{Total: 8, Status: ProgressRunning},
		},
		Cancellable: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := dialog.Show(nil); err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 9)
	for index := 1; index <= 8; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			if updateErr := dialog.Update(context.Background(), ProgressDialogState{
				Status: "Working",
				Progress: ProgressBarState{
					Current: uint64(index), Total: 8, Status: ProgressRunning,
				},
			}); updateErr != nil {
				errorsSeen <- updateErr
			}
			_ = dialog.State()
			_ = dialog.CancelRequested()
			if dialog.Context() == nil {
				errorsSeen <- errors.New("nil progress context")
			}
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for updateErr := range errorsSeen {
		t.Fatalf("concurrent Update() error = %v", updateErr)
	}

	start := make(chan struct{})
	updateResult := make(chan error, 1)
	completeResult := make(chan error, 1)
	go func() {
		<-start
		updateResult <- dialog.SetState(ProgressDialogState{
			Status: "Finishing",
			Progress: ProgressBarState{
				Current: 8, Total: 8, Status: ProgressCompleted,
			},
		})
	}()
	go func() {
		<-start
		completeResult <- dialog.Complete(ModalResult{Reason: ModalAccepted})
	}()
	close(start)
	if completeErr := <-completeResult; completeErr != nil {
		t.Fatalf("concurrent Complete() error = %v", completeErr)
	}
	if updateErr := <-updateResult; updateErr != nil &&
		!errors.Is(updateErr, ErrModalState) {
		t.Fatalf("concurrent terminal Update() error = %v", updateErr)
	}
	if result, ready := dialog.Result(); !ready || result.Reason != ModalAccepted {
		t.Fatalf("concurrent Result() = %+v, %t", result, ready)
	}
}
