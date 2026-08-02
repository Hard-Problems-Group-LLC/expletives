package expletives

import (
	"context"
	"fmt"
	"sync/atomic"
)

// MessageBoxOptions configures one informational standard Dialog.
type MessageBoxOptions struct {
	DialogOptions
	Message string
}

// MessageBox is a copy-safe informational Dialog compound.
type MessageBox struct {
	Dialog
	messageSurface *ScrollablePanel
	message        *StaticText
	ok             *Button
}

// ConfirmChoice is the typed terminal choice of a ConfirmDialog.
type ConfirmChoice string

const (
	ConfirmChoiceYes    ConfirmChoice = "yes"
	ConfirmChoiceNo     ConfirmChoice = "no"
	ConfirmChoiceCancel ConfirmChoice = "cancel"
)

// ConfirmDialogOptions configures one explicit Yes/No standard Dialog. The
// safe zero-value default choice is No. ShowCancel controls the visible Cancel
// Button; Escape cancels regardless so it never aliases No.
type ConfirmDialogOptions struct {
	DialogOptions
	Message    string
	Default    ConfirmChoice
	ShowCancel bool
}

// ConfirmDialog is a copy-safe Yes/No/optional-Cancel Dialog compound.
type ConfirmDialog struct {
	Dialog
	messageSurface *ScrollablePanel
	message        *StaticText
	yes            *Button
	no             *Button
	cancel         *Button
	initial        *Button
}

// InputDialogOptions configures one validated single-line input Dialog.
type InputDialogOptions struct {
	DialogOptions
	Prompt    string
	Text      string
	Validator *TextValidator
	Password  bool
}

// InputDialog is a copy-safe validated text-entry Dialog compound.
type InputDialog struct {
	Dialog
	promptSurface *ScrollablePanel
	prompt        *StaticText
	field         *TextField
	ok            *Button
	cancel        *Button
}

// ProgressDialogState is one complete copied status/progress update.
type ProgressDialogState struct {
	Status   string
	Progress ProgressBarState
}

// ProgressDialogOptions configures one long-operation Dialog. Cancellable
// adds a Cancel Button whose action requests cancellation without closing.
type ProgressDialogOptions struct {
	DialogOptions
	State       ProgressDialogState
	Cancellable bool
}

// ProgressDialog is a copy-safe status/progress Dialog compound.
type ProgressDialog struct {
	Dialog
	status    *StaticText
	progress  *ProgressBar
	cancel    *Button
	lifecycle *progressDialogLifecycle
}

type progressDialogLifecycle struct {
	ctx             context.Context
	cancel          context.CancelFunc
	cancelRequested atomic.Bool
}

func newProgressDialogLifecycle() *progressDialogLifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &progressDialogLifecycle{ctx: ctx, cancel: cancel}
}

func (l *progressDialogLifecycle) requestCancel() bool {
	if l == nil || !l.cancelRequested.CompareAndSwap(false, true) {
		return false
	}
	l.cancel()
	return true
}

func (l *progressDialogLifecycle) stop() {
	if l != nil {
		l.cancel()
	}
}

// NewMessageBox constructs one inactive informational Dialog atomically.
func NewMessageBox(
	parent Container,
	options MessageBoxOptions,
) (*MessageBox, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	box, err := tx.NewMessageBox(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return box, nil
}

// NewMessageBox records one provisional informational Dialog compound.
func (t *Transaction) NewMessageBox(
	parent Container,
	options MessageBoxOptions,
) (*MessageBox, error) {
	content, err := normalizeDisplayText(options.Message, true)
	if err != nil {
		return nil, err
	}
	prepareStandardDialogSize(t.app, &options.ModalPanelOptions, content, 10)
	dialog, err := t.newDialog(parent, options.DialogOptions, ControlMessageBox)
	if err != nil {
		return nil, err
	}
	setProvisionalModalEscapeCommand(dialog.state, CommandDialogOK)
	messageSurface, message, err := newStandardDialogMessageSurface(
		t,
		dialog,
		content,
		options.Message,
	)
	if err != nil {
		return nil, err
	}
	ok, err := t.NewButton(dialog, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "ok"),
		},
		Command:  CommandDialogOK,
		Mnemonic: "k",
		Default:  true,
	})
	if err != nil {
		return nil, err
	}
	if err := attachStandardDialogLayout(
		t,
		dialog,
		messageSurface,
		[]*Button{ok},
	); err != nil {
		return nil, err
	}
	box := &MessageBox{
		Dialog: *dialog, messageSurface: messageSurface, message: message, ok: ok,
	}
	dialog.state.control = box
	dialog.state.container = box
	return box, nil
}

// NewConfirmDialog constructs one inactive explicit-choice Dialog atomically.
func NewConfirmDialog(
	parent Container,
	options ConfirmDialogOptions,
) (*ConfirmDialog, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	dialog, err := tx.NewConfirmDialog(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return dialog, nil
}

// NewConfirmDialog records one provisional explicit-choice Dialog compound.
func (t *Transaction) NewConfirmDialog(
	parent Container,
	options ConfirmDialogOptions,
) (*ConfirmDialog, error) {
	choice := options.Default
	if choice == "" {
		choice = ConfirmChoiceNo
	}
	if choice != ConfirmChoiceYes && choice != ConfirmChoiceNo {
		return nil, fmt.Errorf("%w: invalid ConfirmDialog default", ErrInvalidControl)
	}
	content, err := normalizeDisplayText(options.Message, true)
	if err != nil {
		return nil, err
	}
	buttonWidth := 10 + 2 + 10
	if options.ShowCancel {
		buttonWidth += 2 + 10
	}
	prepareStandardDialogSize(
		t.app,
		&options.ModalPanelOptions,
		content,
		buttonWidth,
	)
	dialog, err := t.newDialog(parent, options.DialogOptions, ControlConfirmDialog)
	if err != nil {
		return nil, err
	}
	setProvisionalModalEscapeCommand(dialog.state, CommandDialogCancel)
	messageSurface, message, err := newStandardDialogMessageSurface(
		t,
		dialog,
		content,
		options.Message,
	)
	if err != nil {
		return nil, err
	}
	yes, err := t.NewButton(dialog, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "yes"),
		},
		Command:  CommandDialogYes,
		Mnemonic: "y",
		Default:  choice == ConfirmChoiceYes,
	})
	if err != nil {
		return nil, err
	}
	no, err := t.NewButton(dialog, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "no"),
		},
		Command:  CommandDialogNo,
		Mnemonic: "n",
		Default:  choice == ConfirmChoiceNo,
	})
	if err != nil {
		return nil, err
	}
	buttons := []*Button{yes, no}
	var cancel *Button
	if options.ShowCancel {
		cancel, err = t.NewButton(dialog, ButtonOptions{
			PanelOptions: PanelOptions{
				AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "cancel"),
			},
			Command:  CommandDialogCancel,
			Mnemonic: "c",
			Cancel:   true,
		})
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, cancel)
	}
	if err := attachStandardDialogLayout(
		t,
		dialog,
		messageSurface,
		buttons,
	); err != nil {
		return nil, err
	}
	confirm := &ConfirmDialog{
		Dialog: *dialog, messageSurface: messageSurface,
		message: message, yes: yes, no: no, cancel: cancel,
	}
	if choice == ConfirmChoiceYes {
		confirm.initial = yes
	} else {
		confirm.initial = no
	}
	dialog.state.control = confirm
	dialog.state.container = confirm
	return confirm, nil
}

// NewInputDialog constructs one inactive validated input Dialog atomically.
func NewInputDialog(
	parent Container,
	options InputDialogOptions,
) (*InputDialog, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	dialog, err := tx.NewInputDialog(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return dialog, nil
}

// NewInputDialog records one provisional validated input Dialog compound.
func (t *Transaction) NewInputDialog(
	parent Container,
	options InputDialogOptions,
) (*InputDialog, error) {
	content, err := normalizeDisplayText(options.Prompt, true)
	if err != nil {
		return nil, err
	}
	autoHeight := options.Bounds.Height == 0
	prepareStandardDialogSize(t.app, &options.ModalPanelOptions, content, 19)
	if autoHeight {
		options.Bounds.Height += 2
	}
	dialog, err := t.newDialog(parent, options.DialogOptions, ControlInputDialog)
	if err != nil {
		return nil, err
	}
	setProvisionalModalEscapeCommand(dialog.state, CommandDialogCancel)
	promptSurface, prompt, err := newStandardDialogMessageSurface(
		t,
		dialog,
		content,
		options.Prompt,
	)
	if err != nil {
		return nil, err
	}
	field, err := t.NewTextField(dialog, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "input"),
		},
		Text: options.Text, Validator: options.Validator, Password: options.Password,
	})
	if err != nil {
		return nil, err
	}
	ok, err := t.NewButton(dialog, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "ok"),
		},
		Command:  CommandDialogOK,
		Mnemonic: "k",
		Default:  true,
	})
	if err != nil {
		return nil, err
	}
	cancel, err := t.NewButton(dialog, ButtonOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "cancel"),
		},
		Command:  CommandDialogCancel,
		Mnemonic: "c",
		Cancel:   true,
	})
	if err != nil {
		return nil, err
	}
	if err := attachInputDialogLayout(
		t,
		dialog,
		promptSurface,
		field,
		[]*Button{ok, cancel},
	); err != nil {
		return nil, err
	}
	behavior := dialog.state.behavior.(modalPanelBehavior)
	behavior.acceptEditor = field.state
	dialog.state.behavior = behavior
	input := &InputDialog{
		Dialog: *dialog, promptSurface: promptSurface, prompt: prompt,
		field: field, ok: ok, cancel: cancel,
	}
	dialog.state.control = input
	dialog.state.container = input
	return input, nil
}

// NewProgressDialog constructs one inactive progress Dialog atomically.
func NewProgressDialog(
	parent Container,
	options ProgressDialogOptions,
) (*ProgressDialog, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	dialog, err := tx.NewProgressDialog(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		dialog.lifecycle.stop()
		return nil, err
	}
	return dialog, nil
}

// NewProgressDialog records one provisional progress Dialog compound.
func (t *Transaction) NewProgressDialog(
	parent Container,
	options ProgressDialogOptions,
) (*ProgressDialog, error) {
	status, err := normalizeDisplayText(options.State.Status, false)
	if err != nil {
		return nil, err
	}
	progress, err := normalizeProgressBarState(options.State.Progress)
	if err != nil {
		return nil, err
	}
	autoHeight := options.Bounds.Height == 0
	buttonWidth := 0
	if options.Cancellable {
		buttonWidth = 10
	}
	prepareStandardDialogSize(t.app, &options.ModalPanelOptions, status, buttonWidth)
	if autoHeight {
		options.Bounds.Height += 2
	}
	dialog, err := t.newDialog(parent, options.DialogOptions, ControlProgressDialog)
	if err != nil {
		return nil, err
	}
	lifecycle := newProgressDialogLifecycle()
	if options.Cancellable {
		setProvisionalModalEscapeCommand(dialog.state, CommandDialogCancel)
	}
	statusControl, err := t.NewStaticText(dialog, StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "status"),
		},
		Text: status.text,
	})
	if err != nil {
		lifecycle.stop()
		return nil, err
	}
	progressControl, err := t.NewProgressBar(dialog, ProgressBarOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "progress"),
		},
		State: progress,
	})
	if err != nil {
		lifecycle.stop()
		return nil, err
	}
	var cancel *Button
	if options.Cancellable {
		cancel, err = t.NewButton(dialog, ButtonOptions{
			PanelOptions: PanelOptions{
				AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "cancel"),
			},
			Command:  CommandDialogCancel,
			Mnemonic: "c",
			Cancel:   true,
		})
		if err != nil {
			lifecycle.stop()
			return nil, err
		}
	}
	if err := attachProgressDialogLayout(
		t, dialog, statusControl, progressControl, cancel,
	); err != nil {
		lifecycle.stop()
		return nil, err
	}
	behavior := dialog.state.behavior.(modalPanelBehavior)
	behavior.progress = lifecycle
	behavior.progressStatus = statusControl.state
	behavior.progressBar = progressControl.state
	if cancel != nil {
		behavior.progressCancel = cancel.state
	}
	dialog.state.behavior = behavior
	result := &ProgressDialog{
		Dialog: *dialog, status: statusControl, progress: progressControl,
		cancel: cancel, lifecycle: lifecycle,
	}
	dialog.state.control = result
	dialog.state.container = result
	return result, nil
}

// Show presents the MessageBox with its OK action selected when initialFocus
// is nil.
func (m *MessageBox) Show(initialFocus Control) error {
	if m == nil {
		return ErrInvalidControl
	}
	if initialFocus == nil {
		initialFocus = m.ok
	}
	return m.Dialog.Show(initialFocus)
}

// Show presents the ConfirmDialog, selecting its safe configured default when
// initialFocus is nil.
func (d *ConfirmDialog) Show(initialFocus Control) error {
	if d == nil {
		return ErrInvalidControl
	}
	if initialFocus == nil {
		initialFocus = d.initial
	}
	return d.Dialog.Show(initialFocus)
}

// Show presents the InputDialog with its editor active when initialFocus is
// nil.
func (d *InputDialog) Show(initialFocus Control) error {
	if d == nil {
		return ErrInvalidControl
	}
	if initialFocus == nil {
		initialFocus = d.field
	}
	return d.Dialog.Show(initialFocus)
}

// Show presents the ProgressDialog, selecting Cancel when available.
func (d *ProgressDialog) Show(initialFocus Control) error {
	if d == nil {
		return ErrInvalidControl
	}
	if initialFocus == nil && d.cancel != nil {
		initialFocus = d.cancel
	}
	return d.Dialog.Show(initialFocus)
}

// MessageControl returns the compound's canonical message control.
func (m *MessageBox) MessageControl() *StaticText {
	if m == nil {
		return nil
	}
	return m.message
}

// MessageViewport returns the scrollable message surface.
func (m *MessageBox) MessageViewport() *ScrollablePanel {
	if m == nil {
		return nil
	}
	return m.messageSurface
}

// OKButton returns the compound's canonical default action.
func (m *MessageBox) OKButton() *Button {
	if m == nil {
		return nil
	}
	return m.ok
}

// MessageControl returns the compound's canonical question control.
func (d *ConfirmDialog) MessageControl() *StaticText {
	if d == nil {
		return nil
	}
	return d.message
}

// MessageViewport returns the scrollable question surface.
func (d *ConfirmDialog) MessageViewport() *ScrollablePanel {
	if d == nil {
		return nil
	}
	return d.messageSurface
}

// YesButton returns the affirmative action.
func (d *ConfirmDialog) YesButton() *Button {
	if d == nil {
		return nil
	}
	return d.yes
}

// NoButton returns the negative and safe-default action.
func (d *ConfirmDialog) NoButton() *Button {
	if d == nil {
		return nil
	}
	return d.no
}

// CancelButton returns the optional visible cancel action.
func (d *ConfirmDialog) CancelButton() *Button {
	if d == nil {
		return nil
	}
	return d.cancel
}

// Choice returns the typed terminal choice after the Dialog closes.
func (d *ConfirmDialog) Choice() (ConfirmChoice, bool) {
	if d == nil {
		return "", false
	}
	result, ready := d.Result()
	if !ready {
		return "", false
	}
	switch result.Action {
	case CommandDialogYes:
		return ConfirmChoiceYes, true
	case CommandDialogNo:
		return ConfirmChoiceNo, true
	case CommandDialogCancel:
		return ConfirmChoiceCancel, true
	default:
		return "", false
	}
}

// PromptControl returns the compound's canonical prompt control.
func (d *InputDialog) PromptControl() *StaticText {
	if d == nil {
		return nil
	}
	return d.prompt
}

// PromptViewport returns the scrollable prompt surface.
func (d *InputDialog) PromptViewport() *ScrollablePanel {
	if d == nil {
		return nil
	}
	return d.promptSurface
}

// Field returns the canonical input editor.
func (d *InputDialog) Field() *TextField {
	if d == nil {
		return nil
	}
	return d.field
}

// OKButton returns the accepting default action.
func (d *InputDialog) OKButton() *Button {
	if d == nil {
		return nil
	}
	return d.ok
}

// CancelButton returns the rejecting cancel action.
func (d *InputDialog) CancelButton() *Button {
	if d == nil {
		return nil
	}
	return d.cancel
}

// Value returns the copied accepted value. Cancelled or otherwise terminal
// dialogs do not expose a value through this result API.
func (d *InputDialog) Value() (string, bool) {
	if d == nil {
		return "", false
	}
	result, ready := d.Result()
	if !ready || result.Reason != ModalAccepted || result.Action != CommandDialogOK {
		return "", false
	}
	return d.field.Text(), true
}

// State returns one atomic copy of the current status and progress state.
func (d *ProgressDialog) State() ProgressDialogState {
	if d == nil || d.state == nil || d.state.app == nil ||
		d.status == nil || d.progress == nil {
		return ProgressDialogState{}
	}
	d.state.app.mu.RLock()
	defer d.state.app.mu.RUnlock()
	status, statusOK := d.status.state.behavior.(textBehavior)
	progress, progressOK := d.progress.state.behavior.(progressBehavior)
	if !statusOK || !progressOK || progress.progressBar == nil ||
		d.state.aborted || d.state.destroyed {
		return ProgressDialogState{}
	}
	return ProgressDialogState{
		Status: status.content.text, Progress: *progress.progressBar,
	}
}

// SetState atomically replaces status and progress through the bounded
// mutation owner.
func (d *ProgressDialog) SetState(state ProgressDialogState) error {
	return d.Update(context.Background(), state)
}

// Update atomically replaces status and progress while observing ctx.
func (d *ProgressDialog) Update(
	ctx context.Context,
	state ProgressDialogState,
) error {
	if ctx == nil {
		return fmt.Errorf("expletives: nil context")
	}
	if d == nil || d.state == nil || d.state.app == nil ||
		d.status == nil || d.progress == nil {
		return ErrInvalidControl
	}
	tx := d.state.app.NewTransaction()
	if err := tx.SetText(d.status, state.Status); err != nil {
		return err
	}
	if err := tx.SetProgressBarState(d.progress, state.Progress); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Context returns the stable dialog-owned operation context. It is cancelled
// when the user requests cancellation or the dialog reaches any terminal
// lifecycle state.
func (d *ProgressDialog) Context() context.Context {
	if d == nil || d.lifecycle == nil {
		return nil
	}
	return d.lifecycle.ctx
}

// CancelRequested reports whether the user made the one-shot cancellation
// request. Terminal closure alone does not set it.
func (d *ProgressDialog) CancelRequested() bool {
	return d != nil && d.lifecycle != nil &&
		d.lifecycle.cancelRequested.Load()
}

// Complete acknowledges application completion with an explicit terminal
// result and closes the dialog. It is equivalent to Close but names the
// ProgressDialog handshake.
func (d *ProgressDialog) Complete(result ModalResult) error {
	if d == nil {
		return ErrInvalidControl
	}
	return d.Close(result)
}

// StatusControl returns the ordinary status StaticText child.
func (d *ProgressDialog) StatusControl() *StaticText {
	if d == nil {
		return nil
	}
	return d.status
}

// ProgressControl returns the ordinary ProgressBar child.
func (d *ProgressDialog) ProgressControl() *ProgressBar {
	if d == nil {
		return nil
	}
	return d.progress
}

// CancelButton returns the optional cancellation-request Button.
func (d *ProgressDialog) CancelButton() *Button {
	if d == nil {
		return nil
	}
	return d.cancel
}

func setProvisionalModalEscapeCommand(state *controlState, command CommandID) {
	behavior := state.behavior.(modalPanelBehavior)
	behavior.escapeCommand = command
	state.behavior = behavior
}

func derivedCompoundKey(base, suffix string) string {
	if base == "" {
		return ""
	}
	candidate := base + "." + suffix
	if !validBoundedIdentifier(candidate) {
		return ""
	}
	return candidate
}

func prepareStandardDialogSize(
	app *App,
	options *ModalPanelOptions,
	content normalizedDisplayText,
	buttonWidth int,
) {
	if options.Bounds.Width == 0 {
		longest := 1
		for _, line := range content.lines {
			longest = max(longest, len(line))
		}
		available := app.Size().Width
		if available > 0 {
			longest = min(longest, max(1, available-4))
		}
		options.Bounds.Width = max(buttonWidth+4, longest+4)
	}
	if options.Bounds.Height == 0 {
		contentWidth := max(1, options.Bounds.Width-4)
		rows := 0
		for _, line := range content.lines {
			rows += max(1, (len(line)+contentWidth-1)/contentWidth)
		}
		options.Bounds.Height = rows + 7
	}
}

func newStandardDialogMessageSurface(
	t *Transaction,
	dialog *Dialog,
	content normalizedDisplayText,
	text string,
) (*ScrollablePanel, *StaticText, error) {
	behavior := dialog.state.behavior.(modalPanelBehavior)
	contentWidth := max(1, behavior.requested.Width-4)
	rows := 0
	for _, line := range content.lines {
		rows += max(1, (len(line)+contentWidth-1)/contentWidth)
	}
	style := StyleID(dialog.state.kind)
	surface, err := t.NewScrollablePanel(dialog, ScrollablePanelOptions{
		ScrollViewOptions: ScrollViewOptions{
			PanelOptions: PanelOptions{
				AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "viewport"),
				Style:         style,
			},
			ContentStyle: style,
			State: ViewportState{
				ContentSize: Size{Width: contentWidth, Height: rows},
			},
		},
		BorderForm:    BorderNone,
		HorizontalBar: ScrollBarVisibilityNever,
		VerticalBar:   ScrollBarVisibilityAuto,
	})
	if err != nil {
		return nil, nil, err
	}
	message, err := t.NewStaticText(surface.Content(), StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "message"),
			Style:         style,
		},
		Text: text,
		Wrap: TextWrapWords,
	})
	if err != nil {
		return nil, nil, err
	}
	layout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "message-layout"),
	})
	if err != nil {
		return nil, nil, err
	}
	if err := layout.AddPanel(message, LayoutItemOptions{Grow: 1}); err != nil {
		return nil, nil, err
	}
	if err := t.SetLayout(surface.Content(), layout); err != nil {
		return nil, nil, err
	}
	return surface, message, nil
}

func attachStandardDialogLayout(
	t *Transaction,
	dialog *Dialog,
	message Control,
	buttons []*Button,
) error {
	root, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "layout"),
		Gap:           1,
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		return err
	}
	row, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "buttons"),
		Gap:           2,
	})
	if err != nil {
		return err
	}
	if err := root.AddPanel(message, LayoutItemOptions{Grow: 1}); err != nil {
		return err
	}
	for _, button := range buttons {
		if err := row.AddPanel(button, LayoutItemOptions{}); err != nil {
			return err
		}
	}
	if err := root.AddLayout(row, LayoutItemOptions{
		HorizontalAlign: AlignCenter,
	}); err != nil {
		return err
	}
	return t.SetLayout(dialog, root)
}

func attachInputDialogLayout(
	t *Transaction,
	dialog *Dialog,
	prompt Control,
	field *TextField,
	buttons []*Button,
) error {
	root, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "layout"),
		Gap:           1,
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		return err
	}
	row, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "buttons"),
		Gap:           2,
	})
	if err != nil {
		return err
	}
	if err := root.AddPanel(prompt, LayoutItemOptions{Grow: 1}); err != nil {
		return err
	}
	if err := root.AddPanel(field, LayoutItemOptions{}); err != nil {
		return err
	}
	for _, button := range buttons {
		if err := row.AddPanel(button, LayoutItemOptions{}); err != nil {
			return err
		}
	}
	if err := root.AddLayout(row, LayoutItemOptions{
		HorizontalAlign: AlignCenter,
	}); err != nil {
		return err
	}
	return t.SetLayout(dialog, root)
}

func attachProgressDialogLayout(
	t *Transaction,
	dialog *Dialog,
	status *StaticText,
	progress *ProgressBar,
	cancel *Button,
) error {
	root, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(dialog.AutomationKey(), "layout"),
		Gap:           1,
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
	})
	if err != nil {
		return err
	}
	if err := root.AddPanel(status, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := root.AddPanel(progress, LayoutItemOptions{}); err != nil {
		return err
	}
	if cancel != nil {
		if err := root.AddPanel(cancel, LayoutItemOptions{
			HorizontalAlign: AlignCenter,
		}); err != nil {
			return err
		}
	}
	return t.SetLayout(dialog, root)
}

func (a *App) applyStandardDialogCommandLocked(
	command CommandID,
	target ControlID,
	result CommandResult,
) CommandResult {
	if !isStandardDialogCommand(command) || result.Outcome != OutcomeApplied {
		return result
	}
	state := a.controlsByID[target]
	modal := modalAncestorState(state)
	if state == nil || modal == nil || modal != a.topModalLocked() {
		return publicResult(
			OutcomeRejected,
			"dialog_target",
			"standard dialog command target is not in the top modal",
			nil,
		)
	}
	reason := ModalAccepted
	valid := standardDialogCommandValidForKind(modal.kind, command)
	switch modal.kind {
	case ControlMessageBox:
	case ControlConfirmDialog:
		if command == CommandDialogCancel {
			reason = ModalCancelled
		}
	case ControlInputDialog:
		if command == CommandDialogCancel {
			reason = ModalCancelled
			a.cancelInputDialogLocked(modal)
		} else if command == CommandDialogOK &&
			!a.acceptInputDialogLocked(modal) {
			return publicResult(
				OutcomeRejected,
				"validation_failed",
				"input value does not satisfy validation",
				nil,
			)
		}
	case ControlProgressDialog:
		if command == CommandDialogCancel {
			behavior, ok := modal.behavior.(modalPanelBehavior)
			if !ok || behavior.progress == nil || behavior.progressCancel == nil {
				return publicResult(
					OutcomeRejected,
					"dialog_action",
					"progress cancellation is unavailable",
					nil,
				)
			}
			if !behavior.progress.requestCancel() {
				return publicResult(
					OutcomeRejected,
					"cancel_requested",
					"progress cancellation was already requested",
					nil,
				)
			}
			a.focus = nil
			a.ensureFocusLocked()
			return result
		}
	case ControlFilePickerDialog, ControlMultiFilePickerDialog,
		ControlDirectoryPickerDialog:
		if command == CommandDialogCancel {
			reason = ModalCancelled
		}
	}
	if !valid {
		return publicResult(
			OutcomeRejected,
			"dialog_action",
			"standard dialog command is invalid for the target",
			nil,
		)
	}
	a.closeModalRangeLocked(len(a.modals)-1, ModalResult{
		Reason: reason,
		Action: command,
	})
	arrangeAllLayoutsLocked(a)
	a.ensureFocusedControlVisibleLocked()
	return result
}

func standardDialogCommandValidForKind(
	kind ControlKind,
	command CommandID,
) bool {
	switch kind {
	case ControlMessageBox:
		return command == CommandDialogOK
	case ControlConfirmDialog:
		return command == CommandDialogYes || command == CommandDialogNo ||
			command == CommandDialogCancel
	case ControlInputDialog:
		return command == CommandDialogOK || command == CommandDialogCancel
	case ControlProgressDialog:
		return command == CommandDialogCancel
	case ControlFilePickerDialog, ControlMultiFilePickerDialog,
		ControlDirectoryPickerDialog:
		return command == CommandDialogCancel
	default:
		return false
	}
}

func (a *App) beginModalEditorLocked(modal *controlState) {
	behavior, ok := modal.behavior.(modalPanelBehavior)
	if !ok || behavior.acceptEditor == nil || a.focus != behavior.acceptEditor {
		return
	}
	editor, ok := behavior.acceptEditor.behavior.(textFieldBehavior)
	if !ok || editor.disabled || editor.editing {
		return
	}
	editor.working = cloneInputText(editor.committed)
	editor.editing = true
	editor.caret = len(editor.working.cells)
	editor.viewOffset = 0
	editor.selectionAnchor = -1
	behavior.acceptEditor.behavior = editor
	behavior.inputInitial = cloneInputText(editor.committed)
	behavior.inputInitialSet = true
	modal.behavior = behavior
}

func (a *App) standardDialogEditorCommandLocked(
	key Key,
	held map[Key]bool,
) (CommandID, ControlID) {
	if !noHeldModifiers(held) {
		return "", ""
	}
	modal := a.topModalLocked()
	if modal == nil || modal.kind != ControlInputDialog {
		return "", ""
	}
	behavior, _ := modal.behavior.(modalPanelBehavior)
	if behavior.acceptEditor == nil || a.focus != behavior.acceptEditor {
		return "", ""
	}
	if key == KeyEscape {
		return CommandDialogCancel, modal.id
	}
	if key != KeyEnter {
		return "", ""
	}
	editor, ok := behavior.acceptEditor.behavior.(textFieldBehavior)
	if !ok || editor.editing {
		return "", ""
	}
	return CommandDialogOK, modal.id
}

func (a *App) acceptInputDialogLocked(modal *controlState) bool {
	behavior, ok := modal.behavior.(modalPanelBehavior)
	if !ok || behavior.acceptEditor == nil {
		return false
	}
	editor, ok := behavior.acceptEditor.behavior.(textFieldBehavior)
	if !ok {
		return false
	}
	current := editor.current()
	if !textCellsValid(current.cells, editor.validator) {
		editor.working = cloneInputText(current)
		editor.editing = true
		editor.caret = len(editor.working.cells)
		editor.viewOffset = 0
		editor.selectionAnchor = -1
		behavior.acceptEditor.behavior = editor
		return false
	}
	editor.committed = cloneInputText(current)
	editor.working = cloneInputText(current)
	editor.editing = false
	editor.selectionAnchor = -1
	behavior.acceptEditor.behavior = editor
	return true
}

func (a *App) cancelInputDialogLocked(modal *controlState) {
	behavior, ok := modal.behavior.(modalPanelBehavior)
	if !ok || behavior.acceptEditor == nil || !behavior.inputInitialSet {
		return
	}
	editor, ok := behavior.acceptEditor.behavior.(textFieldBehavior)
	if !ok {
		return
	}
	editor.committed = cloneInputText(behavior.inputInitial)
	editor.working = cloneInputText(behavior.inputInitial)
	editor.editing = false
	editor.caret = len(editor.committed.cells)
	editor.viewOffset = 0
	editor.selectionAnchor = -1
	behavior.acceptEditor.behavior = editor
}
