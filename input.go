package expletives

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// DefaultDispatchWait bounds waiting for App dispatch serialization and
	// command-handler capacity.
	DefaultDispatchWait = 250 * time.Millisecond
	// DefaultCommandTimeout bounds a router call when the caller supplied no
	// earlier deadline.
	DefaultCommandTimeout = 30 * time.Second
)

// Key is a stable logical key identity independent of terminal key codes.
// Dispatch accepts canonical single-cell printable keys, the named keys below,
// and function-key IDs "f1" through "f12".
type Key string

const (
	// KeyControl is the Control modifier.
	KeyControl Key = "control"
	// KeyAlt is the Alt modifier.
	KeyAlt Key = "alt"
	// KeyShift is the Shift modifier.
	KeyShift Key = "shift"
	// KeyMeta is the Meta modifier.
	KeyMeta Key = "meta"
	// KeySpace is the space key.
	KeySpace Key = "space"
	// KeyEnter is the Enter or Return key.
	KeyEnter Key = "enter"
	// KeyEscape is the Escape key.
	KeyEscape Key = "escape"
	// KeyTab is the Tab key.
	KeyTab Key = "tab"
	// KeyBackspace is the Backspace key.
	KeyBackspace Key = "backspace"
	// KeyUp is the upward navigation key.
	KeyUp Key = "up"
	// KeyDown is the downward navigation key.
	KeyDown Key = "down"
	// KeyLeft is the leftward navigation key.
	KeyLeft Key = "left"
	// KeyRight is the rightward navigation key.
	KeyRight Key = "right"
	// KeyHome is the Home key.
	KeyHome Key = "home"
	// KeyEnd is the End key.
	KeyEnd Key = "end"
	// KeyPageUp is the Page Up key.
	KeyPageUp Key = "page_up"
	// KeyPageDown is the Page Down key.
	KeyPageDown Key = "page_down"
	// KeyInsert is the Insert key.
	KeyInsert Key = "insert"
	// KeyDelete is the Delete key.
	KeyDelete Key = "delete"
	// KeyF2 is function key F2.
	KeyF2 Key = "f2"
	// KeyF4 is function key F4.
	KeyF4 Key = "f4"
	// KeyLeftBracket is the unmodified [ key.
	KeyLeftBracket Key = "["
	// KeyRightBracket is the unmodified ] key.
	KeyRightBracket Key = "]"
	// KeyPlus is the unmodified + key.
	KeyPlus Key = "+"
	// KeyMinus is the unmodified - key.
	KeyMinus Key = "-"
	// KeyAsterisk is the unmodified * key.
	KeyAsterisk Key = "*"
)

// KeyEventKind identifies one raw logical key lifecycle transition.
type KeyEventKind string

const (
	// KeyEventDown marks a key held for subsequent chord resolution.
	KeyEventDown KeyEventKind = "key_down"
	// KeyEventUp releases a held key.
	KeyEventUp KeyEventKind = "key_up"
	// KeyEventPress is a one-shot press that does not change held-key state.
	KeyEventPress KeyEventKind = "key_press"
)

// KeyEvent enters before chord and command resolution.
type KeyEvent struct {
	// Kind is the raw logical lifecycle transition.
	Kind KeyEventKind `json:"kind"`
	// Key is the logical key independent of terminal encoding.
	Key Key `json:"key"`
}

// TextInputKind distinguishes a trusted composition boundary from bounded
// bracketed-paste content. Neither kind participates in command resolution.
type TextInputKind string

const (
	// TextInputCommitted is normalized committed text, such as IME output.
	TextInputCommitted TextInputKind = "committed_text"
	// TextInputPaste is bounded bracketed-paste content.
	TextInputPaste TextInputKind = "paste"
)

// TextInputEvent is one bounded non-key text insertion.
type TextInputEvent struct {
	Kind TextInputKind `json:"kind"`
	Text string        `json:"text"`
}

// CommandID is stable semantic command identity within an App.
type CommandID string

const (
	// CommandOverflowDismiss acknowledges the active toolkit overflow
	// fallback without claiming the underlying geometry recovered.
	CommandOverflowDismiss CommandID = "overflow.dismiss"
	// CommandDialogOK accepts an informational or input standard dialog.
	CommandDialogOK CommandID = "dialog.ok"
	// CommandDialogYes accepts the affirmative ConfirmDialog choice.
	CommandDialogYes CommandID = "dialog.yes"
	// CommandDialogNo accepts the negative ConfirmDialog choice.
	CommandDialogNo CommandID = "dialog.no"
	// CommandDialogCancel cancels a standard dialog without accepting data.
	CommandDialogCancel CommandID = "dialog.cancel"
)

// Command is one resolved application action.
type Command struct {
	// ID selects the registered semantic command.
	ID CommandID
	// Target is an optional App-scoped runtime control identity.
	Target ControlID
	// Source identifies the human, headless, or automation input source.
	Source string
}

// CommandDefinition describes one App-scoped semantic command.
type CommandDefinition struct {
	// ID is the bounded stable semantic identity.
	ID CommandID `json:"id"`
	// Label is canonical one-cell display text. Empty displays ID.
	Label string `json:"label,omitempty"`
	// Description is bounded human-readable help text.
	Description string `json:"description,omitempty"`
	// Enabled permits structured-router resolution and invocation.
	Enabled bool `json:"enabled"`
	// DisabledReason explains an unavailable command.
	DisabledReason string `json:"disabled_reason,omitempty"`
	// Checked is shared presentation state for checkable action surfaces.
	Checked bool `json:"checked,omitempty"`
	// Automation marks the command for attached-automation discovery.
	Automation bool `json:"automation"`
	// ModalPolicy controls un-targeted global use while a modal is active.
	// The zero value is blocked; commands originating from an eligible control
	// in the top modal remain local regardless of this policy.
	ModalPolicy CommandModalPolicy `json:"modal_policy,omitempty"`
}

// CommandModalPolicy controls an un-targeted command while a modal is active.
type CommandModalPolicy string

const (
	CommandModalDefault CommandModalPolicy = ""
	CommandModalBlocked CommandModalPolicy = "blocked"
	CommandModalAllowed CommandModalPolicy = "allowed"
)

// CommandResult is the structured local result returned by a CommandRouter.
// Code is a bounded identifier, Message is bounded public text, and Cause is
// retained only for local diagnostics. An invalid result is converted to a
// bounded OutcomeFailed result.
type CommandResult struct {
	// Outcome is the actual command result.
	Outcome Outcome `json:"outcome"`
	// Code is an optional stable machine-readable result code.
	Code string `json:"code,omitempty"`
	// Message is optional bounded operator-facing text.
	Message string `json:"message,omitempty"`
	// Cause is never serialized.
	Cause error `json:"-"`
}

// CommandRouter applies application policy outside toolkit state locks. It
// must observe cancellation promptly; a call that outlives its context retains
// one bounded handler slot and may overlap a later invocation.
type CommandRouter func(context.Context, Command) CommandResult

// Outcome is the actual result of one accepted event or command.
type Outcome string

const (
	// OutcomeApplied reports an accepted state change or action.
	OutcomeApplied Outcome = "applied"
	// OutcomeNoOp reports successful handling that changed no state.
	OutcomeNoOp Outcome = "no_op"
	// OutcomeRejected reports a request refused by current policy or state.
	OutcomeRejected Outcome = "rejected"
	// OutcomeCancelled reports caller cancellation.
	OutcomeCancelled Outcome = "cancelled"
	// OutcomeInterrupted reports an accepted interrupt that made the App final.
	OutcomeInterrupted Outcome = "interrupted"
	// OutcomeExited reports an accepted exit that made the App final.
	OutcomeExited Outcome = "exited"
	// OutcomeFailed reports an attempted action that failed.
	OutcomeFailed Outcome = "failed"
)

// Completion correlates one accepted request with its exact snapshot.
type Completion struct {
	// RequestID is the caller-supplied correlation identifier.
	RequestID string `json:"request_id"`
	// Outcome is the actual request result.
	Outcome Outcome `json:"outcome"`
	// Command is populated when chord resolution or direct invocation selected
	// a command.
	Command CommandID `json:"command,omitempty"`
	// FrameSequence identifies the associated immutable snapshot.
	FrameSequence uint64 `json:"frame_sequence"`
	// Code and Message are bounded public diagnostics.
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	// Cause is a local diagnostic and is never serialized.
	Cause error `json:"-"`
}

// Chord combines one non-modifier pressed key with distinct modifiers.
type Chord struct {
	// Key is the pressed non-modifier key.
	Key Key `json:"key"`
	// Modifiers is copied during binding; caller order is insignificant.
	Modifiers []Key `json:"modifiers"`
}

// CommandBinding selects the semantic command resolved by a chord.
type CommandBinding struct {
	// Command is the registered semantic command ID.
	Command CommandID
}

// CommandHandler is the compatibility form of CommandRouter. New applications
// should use SetCommandRouter so they can return bounded public diagnostics.
type CommandHandler func(context.Context, Command) (Outcome, error)

// SetCommandHandler installs a compatibility router. Commands invoked through
// this API need not be registered, but registered commands are still
// inspectable and bindings created afterward are represented in the registry.
// The handler must observe its context promptly.
func (a *App) SetCommandHandler(handler CommandHandler) error {
	if handler == nil {
		return errors.New("expletives: nil command handler")
	}
	return a.setCommandRouter(func(ctx context.Context, command Command) CommandResult {
		outcome, err := handler(ctx, command)
		result := CommandResult{Outcome: outcome, Cause: err}
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				result.Outcome = OutcomeCancelled
				result.Code = "cancelled"
				result.Message = "command cancelled"
			case errors.Is(err, context.DeadlineExceeded):
				result.Outcome = OutcomeFailed
				result.Code = "deadline_exceeded"
				result.Message = "command deadline exceeded"
			default:
				result.Outcome = OutcomeFailed
				result.Code = "handler_error"
				result.Message = "command handler failed"
			}
		}
		return result
	}, true)
}

// SetCommandRouter installs or replaces the App-scoped structured router.
// Replacement affects subsequent resolution, not an invocation in progress.
// The App retains the function until it is replaced or the App becomes final.
func (a *App) SetCommandRouter(router CommandRouter) error {
	if router == nil {
		return errors.New("expletives: nil command router")
	}
	return a.setCommandRouter(router, false)
}

func (a *App) setCommandRouter(router CommandRouter, legacy bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	a.commandRouter = router
	a.legacyRouter = legacy
	return nil
}

// RegisterCommand adds one copied definition to the App-scoped registry.
func (a *App) RegisterCommand(definition CommandDefinition) error {
	normalized, err := normalizeCommandDefinition(definition)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.commands[normalized.ID]; exists {
		return ErrDuplicateCommand
	}
	a.commands[normalized.ID] = normalized
	a.refreshActionPresentationLocked()
	a.publishLocked(nil)
	return nil
}

// ReplaceCommand replaces one existing command definition.
func (a *App) ReplaceCommand(definition CommandDefinition) error {
	normalized, err := normalizeCommandDefinition(definition)
	if err != nil {
		return err
	}
	if isBuiltInCommand(normalized.ID) {
		return fmt.Errorf("%w: built-in command is immutable", ErrInvalidRequest)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.commands[normalized.ID]; !exists {
		return fmt.Errorf("%w: command %q", ErrInvalidRequest, normalized.ID)
	}
	a.commands[normalized.ID] = normalized
	a.refreshActionPresentationLocked()
	a.publishLocked(nil)
	return nil
}

// RemoveCommand removes a command and every chord currently bound to it.
func (a *App) RemoveCommand(command CommandID) error {
	if !validBoundedIdentifier(string(command)) {
		return ErrInvalidRequest
	}
	if isBuiltInCommand(command) {
		return fmt.Errorf("%w: built-in command is immutable", ErrInvalidRequest)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.commands[command]; !exists {
		return fmt.Errorf("%w: command %q", ErrInvalidRequest, command)
	}
	delete(a.commands, command)
	for chord, bound := range a.bindings {
		if bound == command {
			delete(a.bindings, chord)
		}
	}
	a.refreshActionPresentationLocked()
	a.publishLocked(nil)
	return nil
}

// Commands returns a caller-owned copy of the App command inventory sorted by
// command ID.
func (a *App) Commands() []CommandDefinition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	definitions := make([]CommandDefinition, 0, len(a.commands))
	for _, definition := range a.commands {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].ID < definitions[j].ID
	})
	return definitions
}

// BindChord registers a stable pre-command raw-key binding. It does not retain
// chord.Modifiers.
func (a *App) BindChord(chord Chord, binding CommandBinding) error {
	encoded, err := encodeChord(chord)
	if err != nil {
		return err
	}
	if !validBoundedIdentifier(string(binding.Command)) {
		return fmt.Errorf("%w: invalid command", ErrInvalidChord)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.bindings[encoded]; exists {
		return fmt.Errorf("%w: binding already exists", ErrInvalidChord)
	}
	if _, exists := a.commands[binding.Command]; !exists {
		if !a.legacyRouter && a.commandRouter != nil {
			return fmt.Errorf("%w: command is not registered", ErrInvalidChord)
		}
		a.commands[binding.Command] = CommandDefinition{
			ID:      binding.Command,
			Enabled: true,
		}
	}
	a.bindings[encoded] = binding.Command
	a.publishLocked(nil)
	return nil
}

// ReplaceChord changes an existing binding.
func (a *App) ReplaceChord(chord Chord, binding CommandBinding) error {
	encoded, err := encodeChord(chord)
	if err != nil {
		return err
	}
	if !validBoundedIdentifier(string(binding.Command)) {
		return fmt.Errorf("%w: invalid command", ErrInvalidChord)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.bindings[encoded]; !exists {
		return fmt.Errorf("%w: binding does not exist", ErrInvalidChord)
	}
	if _, exists := a.commands[binding.Command]; !exists {
		return fmt.Errorf("%w: command is not registered", ErrInvalidChord)
	}
	a.bindings[encoded] = binding.Command
	a.publishLocked(nil)
	return nil
}

// UnbindChord removes an existing binding.
func (a *App) UnbindChord(chord Chord) error {
	encoded, err := encodeChord(chord)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return ErrClosed
	}
	if _, exists := a.bindings[encoded]; !exists {
		return fmt.Errorf("%w: binding does not exist", ErrInvalidChord)
	}
	delete(a.bindings, encoded)
	a.publishLocked(nil)
	return nil
}

func validateCommandDefinition(definition CommandDefinition) error {
	_, err := normalizeCommandDefinition(definition)
	return err
}

func normalizeCommandDefinition(
	definition CommandDefinition,
) (CommandDefinition, error) {
	if !validBoundedIdentifier(string(definition.ID)) {
		return CommandDefinition{}, fmt.Errorf(
			"%w: invalid command ID",
			ErrInvalidRequest,
		)
	}
	if len(definition.Description) > MaxCommandDescriptionBytes ||
		!utf8.ValidString(definition.Description) ||
		strings.ContainsRune(definition.Description, 0) {
		return CommandDefinition{}, fmt.Errorf(
			"%w: invalid command description",
			ErrTextLimit,
		)
	}
	if len(definition.DisabledReason) > MaxCommandDescriptionBytes ||
		!utf8.ValidString(definition.DisabledReason) ||
		strings.ContainsRune(definition.DisabledReason, 0) {
		return CommandDefinition{}, fmt.Errorf(
			"%w: invalid disabled reason",
			ErrTextLimit,
		)
	}
	if definition.Enabled && definition.DisabledReason != "" {
		return CommandDefinition{}, fmt.Errorf(
			"%w: enabled command has a disabled reason",
			ErrInvalidRequest,
		)
	}
	if !definition.Enabled && definition.DisabledReason == "" {
		definition.DisabledReason = "Command is disabled"
	}
	if definition.Label != "" {
		label, err := normalizeDisplayText(definition.Label, false)
		if err != nil {
			return CommandDefinition{}, err
		}
		definition.Label = label.text
	}
	switch definition.ModalPolicy {
	case CommandModalDefault, CommandModalBlocked, CommandModalAllowed:
	default:
		return CommandDefinition{}, fmt.Errorf(
			"%w: invalid command modal policy",
			ErrInvalidRequest,
		)
	}
	return definition, nil
}

// DispatchKey serializes and applies one raw logical key event for an isolated
// source. It honors ctx while waiting and during command routing, and returns
// a Completion paired with the exact resulting snapshot for every accepted
// event.
func (a *App) DispatchKey(
	ctx context.Context,
	source string,
	requestID string,
	event KeyEvent,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) ||
		!validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	if !validKey(event.Key) {
		return Completion{}, ErrInvalidKeyEvent
	}
	switch event.Kind {
	case KeyEventDown, KeyEventUp, KeyEventPress:
	default:
		return Completion{}, ErrInvalidKeyEvent
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()

	var command CommandID
	var target ControlID
	result := CommandResult{Outcome: OutcomeNoOp}
	var router CommandRouter
	var execute bool
	a.mu.Lock()
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	held := a.held[source]
	switch event.Kind {
	case KeyEventDown:
		if held[event.Key] {
			result.Outcome = OutcomeNoOp
		} else if len(held) >= MaxHeldKeysPerSource {
			result = publicResult(
				OutcomeRejected,
				"held_key_capacity",
				"held-key capacity reached",
				nil,
			)
		} else if held == nil && len(a.held) >= MaxInputSources {
			result = publicResult(
				OutcomeRejected,
				"input_source_capacity",
				"input-source capacity reached",
				nil,
			)
		} else {
			if held == nil {
				held = make(map[Key]bool)
				a.held[source] = held
			}
			held[event.Key] = true
			result.Outcome = OutcomeApplied
			if (event.Key == KeyEnter || event.Key == KeySpace) &&
				noHeldModifiers(held) &&
				a.buttonEligibleLocked(a.focus) {
				a.pressed[source] = pressedAction{
					control: a.focus,
					key:     event.Key,
				}
			} else if (event.Key == KeyEnter || event.Key == KeySpace) &&
				noHeldModifiers(held) &&
				a.tableColumnsActionFocusedEligibleLocked(a.focus) {
				a.pressed[source] = pressedAction{
					control: a.focus,
					key:     event.Key,
					part:    TableFocusColumnsAction,
				}
			}
		}
	case KeyEventUp:
		press, pressed := a.pressed[source]
		matchedPress := pressed && press.key == event.Key
		if matchedPress {
			delete(a.pressed, source)
		}
		if !held[event.Key] {
			if !matchedPress {
				result.Outcome = OutcomeNoOp
			}
		} else {
			delete(held, event.Key)
			if len(held) == 0 {
				delete(a.held, source)
			}
			result.Outcome = OutcomeApplied
		}
		if matchedPress && press.part == TableFocusColumnsAction &&
			a.tableColumnsActionFocusedEligibleLocked(press.control) {
			command, target = CommandTableColumnsOpen, press.control.id
			router, result, execute = a.resolveCommandLocked(command, true)
		} else if matchedPress && a.buttonEligibleLocked(press.control) {
			behavior := press.control.behavior.(buttonBehavior)
			command = behavior.command
			target = press.control.id
			router, result, execute = a.resolveCommandLocked(command, true)
		}
	case KeyEventPress:
		if (event.Key == KeyEnter || event.Key == KeyEscape) &&
			a.dismissOverflowLocked() {
			command = CommandOverflowDismiss
			result.Outcome = OutcomeApplied
		} else if menuCommand, handled, changed :=
			a.dispatchMenuKeyLocked(event.Key, held); handled {
			if changed {
				result.Outcome = OutcomeApplied
			}
			if menuCommand != "" {
				command = menuCommand
				router, result, execute = a.resolveCommandLocked(command)
			}
		} else if mnemonicKeyEvent(event.Key, held) &&
			a.bindings[chordKey(event.Key, held)] != "" &&
			a.globalCommandMayResolveLocked(
				a.bindings[chordKey(event.Key, held)],
			) {
			// A displayed application accelerator is global. MenuBar
			// mnemonics retain first refusal above, but a local control or
			// tab mnemonic must not shadow a registered Alt chord such as
			// the conventional Alt-X Quit binding.
			command = a.bindings[chordKey(event.Key, held)]
			router, result, execute = a.resolveCommandLocked(command)
		} else if mnemonicKeyEvent(event.Key, held) {
			tabCommand, tabTarget, tabHandled, tabChanged :=
				a.tabMnemonicLocked(event.Key)
			if tabHandled {
				command = tabCommand
				target = tabTarget
				if tabChanged {
					result.Outcome = OutcomeApplied
				}
				if command != "" {
					router, result, execute = a.resolveCommandLocked(command, true)
				}
			} else {
				control, activate := a.mnemonicControlLocked(event.Key)
				if control != nil && control != a.focus {
					_ = a.commitOrCancelEditorStateLocked(a.focus)
				}
				if control != nil && activate {
					if behavior, ok := control.behavior.(buttonBehavior); ok {
						command = behavior.command
						target = control.id
						router, result, execute =
							a.resolveCommandLocked(command, true)
					} else if _, ok := tableBehaviorFromControl(control); ok {
						changed := false
						if a.focus != control {
							a.focus = control
							changed = true
						}
						if a.setTableEntryFocusPartLocked(control, true) {
							changed = true
						}
						a.clearInvalidPressesLocked()
						if changed {
							result.Outcome = OutcomeApplied
						}
						command, target = CommandTableColumnsOpen, control.id
						router, result, execute =
							a.resolveCommandLocked(command, true)
					} else {
						var changed bool
						command, target, changed = a.applySelectionLocked(
							control,
							selectionActivate,
						)
						if changed {
							result.Outcome = OutcomeApplied
						}
						if command != "" {
							router, result, execute =
								a.resolveCommandLocked(command, true)
						}
					}
				} else if control != nil {
					if a.focus != control {
						a.focus = control
						a.clearInvalidPressesLocked()
						result.Outcome = OutcomeApplied
					}
				} else {
					command = a.bindings[chordKey(event.Key, held)]
					if command != "" {
						router, result, execute = a.resolveCommandLocked(command)
					}
				}
			}
		} else if event.Key == KeyTab && tabModifiers(held) {
			result.Outcome = OutcomeNoOp
			var committed, move bool
			command, target, committed, move =
				a.commitFocusedEditorLocked(held[KeyShift])
			if committed {
				result.Outcome = OutcomeApplied
			}
			if move {
				if a.moveTableFocusPartLocked(held[KeyShift]) ||
					a.moveFocusLocked(held[KeyShift]) {
					result.Outcome = OutcomeApplied
				}
			}
			if command != "" {
				router, result, execute = a.resolveCommandLocked(command, true)
			}
		} else {
			textHandled := false
			if !held[KeyAlt] && !held[KeyMeta] {
				var textChanged bool
				command, target, textHandled, textChanged =
					a.editorInputLocked(a.focus, event.Key, held)
				if textHandled {
					result.Outcome = OutcomeNoOp
					if textChanged {
						result.Outcome = OutcomeApplied
					}
					if command != "" {
						router, result, execute =
							a.resolveCommandLocked(command, true)
					}
				}
			}
			if textHandled {
				if command == "" {
					command, target = a.standardDialogEditorCommandLocked(
						event.Key,
						held,
					)
					if command != "" {
						router, result, execute =
							a.resolveCommandLocked(command, true)
					}
				}
				break
			}
			if columnsCommand, columnsTarget, columnsHandled :=
				a.tableColumnsDialogKeyLocked(a.focus, event.Key, held); columnsHandled {
				command, target = columnsCommand, columnsTarget
				router, result, execute = a.resolveCommandLocked(command, true)
				break
			}
			tabCommand := CommandID("")
			tabTarget := ControlID("")
			tabHandled := false
			tabChanged := false
			if noHeldModifiers(held) {
				tabCommand, tabTarget, tabHandled, tabChanged =
					a.tabbedPanelKeyLocked(a.focus, event.Key)
			}
			if tabHandled {
				command = tabCommand
				target = tabTarget
				result.Outcome = OutcomeNoOp
				if tabChanged {
					result.Outcome = OutcomeApplied
				}
				if command != "" {
					router, result, execute =
						a.resolveCommandLocked(command, true)
				}
				break
			}
			scrollCommand := CommandID("")
			scrollTarget := ControlID("")
			scrollHandled := false
			scrollChanged := false
			if noHeldModifiers(held) || held[KeyAlt] {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.popupCollectionKeyLocked(a.focus, event.Key, held, false)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.listBoxKeyLocked(a.focus, event.Key)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.treeViewKeyLocked(a.focus, event.Key)
			}
			controlOnly := held[KeyControl] && !held[KeyAlt] &&
				!held[KeyMeta] && !held[KeyShift]
			shiftOnly := held[KeyShift] && !held[KeyAlt] &&
				!held[KeyMeta] && !held[KeyControl]
			if (noHeldModifiers(held) || controlOnly || shiftOnly) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.tableKeyLocked(a.focus, event.Key, controlOnly, shiftOnly)
			}
			if (noHeldModifiers(held) || controlOnly || shiftOnly) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.dataGridKeyLocked(a.focus, event.Key, controlOnly, shiftOnly)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.logViewKeyLocked(a.focus, event.Key)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.markdownKeyLocked(a.focus, event.Key)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.scrollViewKeyLocked(a.focus, event.Key)
			}
			if noHeldModifiers(held) && !scrollHandled {
				scrollCommand, scrollTarget, scrollHandled, scrollChanged =
					a.scrollBarKeyLocked(a.focus, event.Key)
			}
			if scrollHandled {
				command = scrollCommand
				target = scrollTarget
				result.Outcome = OutcomeNoOp
				if scrollChanged {
					result.Outcome = OutcomeApplied
				}
				if command != "" {
					router, result, execute =
						a.resolveCommandLocked(command, true)
				}
			} else if noHeldModifiers(held) &&
				isDirectionalFocusKey(event.Key) {
				result.Outcome = OutcomeNoOp
				if a.moveDirectionalFocusLocked(event.Key) {
					result.Outcome = OutcomeApplied
				}
			} else if noHeldModifiers(held) &&
				(event.Key == KeyHome || event.Key == KeyEnd) &&
				a.moveRadioBoundaryFocusLocked(event.Key) {
				result.Outcome = OutcomeApplied
			} else if action, handled := a.selectionKeyActionLocked(
				a.focus,
				event.Key,
			); noHeldModifiers(held) && handled {
				var changed bool
				command, target, changed = a.applySelectionLocked(
					a.focus,
					action,
				)
				if changed {
					result.Outcome = OutcomeApplied
				}
				if command != "" {
					router, result, execute =
						a.resolveCommandLocked(command, true)
				}
			} else if noHeldModifiers(held) &&
				(event.Key == KeyEnter || event.Key == KeySpace) {
				control := a.focus
				if !a.buttonEligibleLocked(control) &&
					event.Key == KeyEnter {
					control = a.roleButtonLocked(false)
				}
				if a.buttonEligibleLocked(control) {
					behavior := control.behavior.(buttonBehavior)
					command = behavior.command
					target = control.id
					router, result, execute =
						a.resolveCommandLocked(command, true)
				}
			} else if noHeldModifiers(held) && event.Key == KeyEscape {
				control := a.roleButtonLocked(true)
				if control != nil {
					behavior := control.behavior.(buttonBehavior)
					command = behavior.command
					target = control.id
					router, result, execute =
						a.resolveCommandLocked(command, true)
				} else if modal := a.topModalLocked(); modal != nil {
					behavior, _ := modal.behavior.(modalPanelBehavior)
					if behavior.escapeCommand != "" {
						command = behavior.escapeCommand
						target = modal.id
						router, result, execute =
							a.resolveCommandLocked(command, true)
					}
				}
			} else {
				command = a.bindings[chordKey(event.Key, held)]
				if command != "" {
					router, result, execute =
						a.resolveCommandLocked(command)
				}
			}
		}
	}
	if a.ensureFocusedControlVisibleLocked() &&
		result.Outcome == OutcomeNoOp {
		result.Outcome = OutcomeApplied
	}
	a.mu.Unlock()

	if execute {
		result = a.callRouter(ctx, router, Command{
			ID:     command,
			Target: target,
			Source: source,
		})
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	result = a.applyStandardDialogCommandLocked(command, target, result)
	return a.associateLocked(requestID, result, command), nil
}

// DispatchTextInput serializes one bounded committed-text or paste event.
// Text enters only the focused enabled editor in edit mode and is never
// interpreted as keys, mnemonics, bindings, menu input, or commands.
func (a *App) DispatchTextInput(
	ctx context.Context,
	source string,
	requestID string,
	event TextInputEvent,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) ||
		!validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	switch event.Kind {
	case TextInputCommitted, TextInputPaste:
	default:
		return Completion{}, ErrInvalidRequest
	}
	if len(event.Text) > MaxTextInputBytes {
		return Completion{}, ErrTextLimit
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()

	var command CommandID
	var target ControlID
	var router CommandRouter
	var execute bool
	a.mu.Lock()
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	state := a.focus
	before := ""
	if behavior, ok := stateBehaviorAsTextField(state); ok && behavior.editing {
		before = behavior.current().text
	}
	result := a.applyTextInputLocked(event.Text)
	if behavior, ok := stateBehaviorAsTextField(state); ok &&
		behavior.editing && before != behavior.current().text {
		command = behavior.editCommand
		target = state.id
		if command != "" {
			router, result, execute = a.resolveCommandLocked(command, true)
		}
	}
	a.mu.Unlock()
	if execute {
		result = a.callRouter(ctx, router, Command{
			ID: command, Target: target, Source: source,
		})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	return a.associateLocked(requestID, result, command), nil
}

func stateBehaviorAsTextField(
	state *controlState,
) (textFieldBehavior, bool) {
	if state == nil {
		return textFieldBehavior{}, false
	}
	behavior, ok := state.behavior.(textFieldBehavior)
	return behavior, ok
}

func (a *App) applyTextInputLocked(text string) CommandResult {
	state := a.focus
	if state == nil || !a.focusEligibleLocked(state) {
		return CommandResult{Outcome: OutcomeNoOp}
	}
	switch behavior := state.behavior.(type) {
	case textFieldBehavior:
		if !behavior.editing {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		inserted, err := normalizeInputText(text)
		if err != nil {
			return rejectedTextInput(err)
		}
		inserted.cells = filterTextInputCells(
			inserted.cells,
			behavior.validator,
			false,
		)
		candidate, caret, ok := replaceTextInputCells(
			behavior.working.cells,
			behavior.caret,
			behavior.selectionAnchor,
			inserted.cells,
			behavior.effectiveMaximumBytes(),
		)
		if !ok {
			return textInputCapacityResult()
		}
		if candidate == nil {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		behavior.working.cells = candidate
		behavior.working.text = strings.Join(candidate, "")
		behavior.caret = caret
		behavior.selectionAnchor = -1
		behavior.viewOffset = textViewOffset(
			behavior.viewOffset,
			behavior.caret,
			len(candidate),
			state.bounds.Width,
		)
		state.behavior = behavior
		return CommandResult{Outcome: OutcomeApplied}
	case comboBoxBehavior:
		return a.applyComboBoxTextInputLocked(state, behavior, text)
	case dataGridBehavior:
		return a.applyDataGridTextInputLocked(state, behavior, text)
	case numberFieldBehavior:
		if !behavior.editor.editing {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		inserted, err := normalizeInputText(text)
		if err != nil {
			return rejectedTextInput(err)
		}
		inserted.cells = filterTextInputCells(
			inserted.cells,
			behavior.editor.validator,
			false,
		)
		candidate, caret, ok := replaceTextInputCells(
			behavior.editor.working.cells,
			behavior.editor.caret,
			behavior.editor.selectionAnchor,
			inserted.cells,
			behavior.editor.effectiveMaximumBytes(),
		)
		if !ok {
			return textInputCapacityResult()
		}
		if candidate == nil {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		behavior.editor.working.cells = candidate
		behavior.editor.working.text = strings.Join(candidate, "")
		behavior.editor.caret = caret
		behavior.editor.selectionAnchor = -1
		behavior.editor.viewOffset = textViewOffset(
			behavior.editor.viewOffset,
			behavior.editor.caret,
			len(candidate),
			state.bounds.Width,
		)
		state.behavior = behavior
		return CommandResult{Outcome: OutcomeApplied}
	case textAreaBehavior:
		if !behavior.editing {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		inserted, err := normalizeTextAreaInput(text)
		if err != nil {
			return rejectedTextInput(err)
		}
		inserted.cells = filterTextInputCells(
			inserted.cells,
			behavior.validator,
			true,
		)
		candidate, caret, ok := replaceTextInputCells(
			behavior.working.cells,
			behavior.caret,
			behavior.selectionAnchor,
			inserted.cells,
			MaxTextInputBytes,
		)
		if !ok {
			return textInputCapacityResult()
		}
		if candidate == nil {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		behavior.working.cells = candidate
		rebuildTextAreaWorking(&behavior)
		behavior.caret = caret
		behavior.selectionAnchor = -1
		behavior.preferredColumn = -1
		_, _, _, behavior.rowOffset, behavior.columnOffset =
			textAreaViewport(
				behavior,
				state.bounds.Width,
				state.bounds.Height,
			)
		state.behavior = behavior
		return CommandResult{Outcome: OutcomeApplied}
	default:
		return CommandResult{Outcome: OutcomeNoOp}
	}
}

func filterTextInputCells(
	cells []string,
	validator *normalizedTextValidator,
	allowNewline bool,
) []string {
	if validator == nil ||
		validator.value.Enforcement != TextValidationHard {
		return cells
	}
	filtered := make([]string, 0, len(cells))
	for _, cell := range cells {
		if (allowNewline && cell == "\n") ||
			textCellAllowed(cell, validator) {
			filtered = append(filtered, cell)
		}
	}
	return filtered
}

func replaceTextInputCells(
	current []string,
	caret int,
	anchor int,
	inserted []string,
	maximumBytes int,
) (candidate []string, nextCaret int, ok bool) {
	start, end := caret, caret
	if anchor >= 0 && anchor != caret {
		start, end = min(anchor, caret), max(anchor, caret)
	}
	if len(inserted) == 0 && start == end {
		return nil, caret, true
	}
	candidate = make([]string, 0, len(current)-(end-start)+len(inserted))
	candidate = append(candidate, current[:start]...)
	candidate = append(candidate, inserted...)
	candidate = append(candidate, current[end:]...)
	if maximumBytes <= 0 || maximumBytes > MaxTextInputBytes {
		maximumBytes = MaxTextInputBytes
	}
	if len(candidate) > MaxTextInputCells ||
		len(strings.Join(candidate, "")) > maximumBytes {
		return nil, caret, false
	}
	return candidate, start + len(inserted), true
}

func rejectedTextInput(err error) CommandResult {
	return publicResult(
		OutcomeRejected,
		"invalid_text",
		"text input is invalid for the focused editor",
		err,
	)
}

func textInputCapacityResult() CommandResult {
	return publicResult(
		OutcomeRejected,
		"text_capacity",
		"text input would exceed the editor capacity",
		ErrTextLimit,
	)
}

// InvokeCommand serializes one stable direct semantic command. It honors ctx
// while waiting and during routing. A nonempty target must name an active
// control in this App.
func (a *App) InvokeCommand(
	ctx context.Context,
	source string,
	requestID string,
	command CommandID,
	target ControlID,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) ||
		!validBoundedIdentifier(requestID) ||
		!validBoundedIdentifier(string(command)) {
		return Completion{}, ErrInvalidRequest
	}
	if target != "" && !validBoundedIdentifier(string(target)) {
		return Completion{}, ErrInvalidRequest
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()

	a.mu.Lock()
	if a.final {
		a.mu.Unlock()
		return Completion{}, ErrClosed
	}
	modalLocal := false
	if target != "" {
		state, exists := a.controlsByID[target]
		if !exists || state.destroyed {
			a.mu.Unlock()
			return Completion{}, fmt.Errorf(
				"%w: command target %q",
				ErrInvalidControl,
				target,
			)
		}
		if top := a.topModalLocked(); top != nil && !controlWithin(state, top) {
			completion := a.associateLocked(
				requestID,
				publicResult(
					OutcomeRejected,
					"modal_scope",
					"command target is outside the active modal",
					nil,
				),
				command,
			)
			a.mu.Unlock()
			return completion, nil
		}
		modalLocal = true
	}
	router, result, execute := a.resolveCommandLocked(command, modalLocal)
	a.mu.Unlock()
	if execute {
		result = a.callRouter(ctx, router, Command{
			ID:     command,
			Target: target,
			Source: source,
		})
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	result = a.applyStandardDialogCommandLocked(command, target, result)
	return a.associateLocked(requestID, result, command), nil
}

// ResetInput serializes and clears held keys for a source, then publishes an
// associated completion. It honors ctx while waiting.
func (a *App) ResetInput(
	ctx context.Context,
	source string,
	requestID string,
) (Completion, error) {
	if ctx == nil {
		return Completion{}, errors.New("expletives: nil context")
	}
	if !validBoundedIdentifier(source) ||
		!validBoundedIdentifier(requestID) {
		return Completion{}, ErrInvalidRequest
	}
	if err := a.beginDispatch(ctx); err != nil {
		return Completion{}, err
	}
	defer a.endDispatch()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final {
		return Completion{}, ErrClosed
	}
	outcome := OutcomeNoOp
	if len(a.held[source]) != 0 {
		delete(a.held, source)
		outcome = OutcomeApplied
	}
	if _, exists := a.pressed[source]; exists {
		delete(a.pressed, source)
		outcome = OutcomeApplied
	}
	return a.associateLocked(
		requestID,
		CommandResult{Outcome: outcome},
		"",
	), nil
}

// ClearInputSource clears disconnected source state without a request. It is
// callback-safe and may publish an intermediate uncorrelated snapshot. It
// returns immediately after acquiring the App state lock.
func (a *App) ClearInputSource(source string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, pressed := a.pressed[source]
	if len(a.held[source]) == 0 && !pressed {
		delete(a.held, source)
		return
	}
	delete(a.held, source)
	delete(a.pressed, source)
	if !a.final {
		a.publishLocked(nil)
	}
}

func noHeldModifiers(held map[Key]bool) bool {
	return !held[KeyAlt] && !held[KeyControl] &&
		!held[KeyMeta] && !held[KeyShift]
}

func tabModifiers(held map[Key]bool) bool {
	return !held[KeyAlt] && !held[KeyControl] && !held[KeyMeta]
}

func mnemonicKeyEvent(key Key, held map[Key]bool) bool {
	return held[KeyAlt] && !held[KeyControl] &&
		!held[KeyMeta] && !held[KeyShift] &&
		len(key) == 1
}

func (a *App) resolveCommandLocked(
	command CommandID,
	modalLocal ...bool,
) (CommandRouter, CommandResult, bool) {
	definition, registered := a.commands[command]
	if command == CommandOverflowDismiss && registered {
		outcome := OutcomeNoOp
		if a.dismissOverflowLocked() {
			outcome = OutcomeApplied
		}
		return nil, CommandResult{Outcome: outcome}, false
	}
	if !a.legacyRouter {
		if !registered {
			return nil, publicResult(
				OutcomeRejected,
				"command_unknown",
				"command is not registered",
				nil,
			), false
		}
		if !definition.Enabled {
			return nil, publicResult(
				OutcomeRejected,
				"command_disabled",
				effectiveDisabledReason(definition, true),
				nil,
			), false
		}
	}
	local := len(modalLocal) != 0 && modalLocal[0]
	if a.topModalLocked() != nil && !local &&
		definition.ModalPolicy != CommandModalAllowed {
		return nil, publicResult(
			OutcomeRejected,
			"modal_scope",
			"command is blocked while a modal is active",
			nil,
		), false
	}
	if isStandardDialogCommand(command) {
		return nil, CommandResult{Outcome: OutcomeApplied}, false
	}
	if isTableColumnsCommand(command) {
		return a.routeTableColumnsCommand, CommandResult{}, true
	}
	if isFilePickerCommand(command) {
		return a.routeFilePickerCommand, CommandResult{}, true
	}
	if a.commandRouter == nil {
		return nil, publicResult(
			OutcomeRejected,
			"command_unhandled",
			"no command router is installed",
			nil,
		), false
	}
	return a.commandRouter, CommandResult{}, true
}

func isBuiltInCommand(command CommandID) bool {
	return command == CommandOverflowDismiss || isStandardDialogCommand(command) ||
		isFilePickerCommand(command) || isTableColumnsCommand(command)
}

func isStandardDialogCommand(command CommandID) bool {
	switch command {
	case CommandDialogOK, CommandDialogYes, CommandDialogNo, CommandDialogCancel:
		return true
	default:
		return false
	}
}

func (a *App) globalCommandMayResolveLocked(command CommandID) bool {
	if a.topModalLocked() == nil {
		return true
	}
	definition, registered := a.commands[command]
	return registered && definition.ModalPolicy == CommandModalAllowed
}

func (a *App) callRouter(
	ctx context.Context,
	router CommandRouter,
	command Command,
) CommandResult {
	if router == nil {
		return publicResult(
			OutcomeRejected,
			"command_unhandled",
			"no command router is installed",
			nil,
		)
	}

	callContext, cancel := context.WithTimeout(ctx, DefaultCommandTimeout)
	defer cancel()

	slotTimer := time.NewTimer(DefaultDispatchWait)
	defer slotTimer.Stop()
	select {
	case a.handlerSlots <- struct{}{}:
	case <-callContext.Done():
		return contextCommandResult(callContext.Err())
	case <-slotTimer.C:
		return publicResult(
			OutcomeFailed,
			"handler_capacity",
			"command handler capacity reached",
			ErrDispatchBusy,
		)
	}

	completed := make(chan CommandResult, 1)
	go func() {
		result := CommandResult{}
		defer func() {
			if recovered := recover(); recovered != nil {
				result = publicResult(
					OutcomeFailed,
					"handler_panic",
					"command handler panicked",
					fmt.Errorf(
						"expletives: command router panic: %v",
						recovered,
					),
				)
			}
			completed <- result
			<-a.handlerSlots
		}()
		result = router(callContext, command)
	}()

	select {
	case result := <-completed:
		return normalizeCommandResult(result)
	case <-callContext.Done():
		return contextCommandResult(callContext.Err())
	}
}

func (a *App) beginDispatch(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(DefaultDispatchWait)
	defer timer.Stop()
	select {
	case a.dispatchGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-a.dispatchGate
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrDispatchBusy
	}
}

func (a *App) endDispatch() {
	<-a.dispatchGate
}

func contextCommandResult(err error) CommandResult {
	if errors.Is(err, context.Canceled) {
		return publicResult(
			OutcomeCancelled,
			"cancelled",
			"command cancelled",
			err,
		)
	}
	return publicResult(
		OutcomeFailed,
		"deadline_exceeded",
		"command deadline exceeded",
		err,
	)
}

func publicResult(
	outcome Outcome,
	code string,
	message string,
	cause error,
) CommandResult {
	return CommandResult{
		Outcome: outcome,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func normalizeCommandResult(result CommandResult) CommandResult {
	if !validOutcome(result.Outcome) ||
		(result.Code != "" && !validBoundedIdentifier(result.Code)) ||
		len(result.Message) > MaxPublicMessageBytes ||
		!utf8.ValidString(result.Message) ||
		strings.ContainsRune(result.Message, 0) {
		return publicResult(
			OutcomeFailed,
			"invalid_handler_result",
			"command handler returned an invalid result",
			result.Cause,
		)
	}
	return result
}

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomeApplied, OutcomeNoOp, OutcomeRejected, OutcomeCancelled,
		OutcomeInterrupted, OutcomeExited, OutcomeFailed:
		return true
	default:
		return false
	}
}

func encodeChord(chord Chord) (string, error) {
	if !validKey(chord.Key) || isModifier(chord.Key) {
		return "", ErrInvalidChord
	}
	modifiers := make([]Key, len(chord.Modifiers))
	copy(modifiers, chord.Modifiers)
	seen := make(map[Key]bool)
	for _, modifier := range modifiers {
		if !isModifier(modifier) || seen[modifier] {
			return "", ErrInvalidChord
		}
		seen[modifier] = true
	}
	sort.Slice(modifiers, func(i, j int) bool {
		return modifiers[i] < modifiers[j]
	})
	parts := make([]string, 0, len(modifiers)+1)
	for _, modifier := range modifiers {
		parts = append(parts, string(modifier))
	}
	parts = append(parts, string(chord.Key))
	return strings.Join(parts, "+"), nil
}

func chordKey(key Key, held map[Key]bool) string {
	modifiers := make([]Key, 0, 4)
	for _, modifier := range []Key{KeyAlt, KeyControl, KeyMeta, KeyShift} {
		if held[modifier] {
			modifiers = append(modifiers, modifier)
		}
	}
	encoded, err := encodeChord(Chord{Key: key, Modifiers: modifiers})
	if err != nil {
		return ""
	}
	return encoded
}

func isModifier(key Key) bool {
	switch key {
	case KeyControl, KeyAlt, KeyShift, KeyMeta:
		return true
	default:
		return false
	}
}

func validKey(key Key) bool {
	if validPrintableKey(key) {
		return true
	}
	switch key {
	case KeyControl, KeyAlt, KeyShift, KeyMeta,
		KeySpace, KeyEnter, KeyEscape, KeyTab, KeyBackspace,
		KeyUp, KeyDown, KeyLeft, KeyRight, KeyHome, KeyEnd,
		KeyPageUp, KeyPageDown, KeyInsert, KeyDelete:
		return true
	}
	switch key {
	case "f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12":
		return true
	}
	return false
}
