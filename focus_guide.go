package expletives

import (
	"context"
	"fmt"
)

// FocusGuidanceMode selects how application guidance combines with the
// toolkit's generic focused-control guidance.
type FocusGuidanceMode string

const (
	// FocusGuidanceAppend preserves generic guidance and appends Text.
	FocusGuidanceAppend FocusGuidanceMode = "append"
	// FocusGuidanceOverride replaces generic guidance with Text.
	FocusGuidanceOverride FocusGuidanceMode = "override"
)

// FocusGuidance is optional application guidance for one focusable control.
// Empty Text clears a prior customization.
type FocusGuidance struct {
	Mode FocusGuidanceMode
	Text string
}

// FocusGuideBarOptions configures one dynamic one-row guidance control.
type FocusGuideBarOptions struct {
	PanelOptions
}

// FocusGuideBar is a copy-safe, non-focusable leaf that renders guidance for
// the App's currently focused control.
type FocusGuideBar struct{ controlHandle }

type focusGuidanceConfig struct {
	mode    FocusGuidanceMode
	content normalizedDisplayText
	set     bool
}

type focusGuideBarBehavior struct{}

// NewFocusGuideBar constructs and atomically inserts a FocusGuideBar.
func NewFocusGuideBar(
	parent Container,
	options FocusGuideBarOptions,
) (*FocusGuideBar, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	bar, err := tx.NewFocusGuideBar(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return bar, nil
}

// NewFocusGuideBar records construction of a provisional FocusGuideBar.
func (t *Transaction) NewFocusGuideBar(
	parent Container,
	options FocusGuideBarOptions,
) (*FocusGuideBar, error) {
	state, err := t.newLeafControl(
		parent,
		options.PanelOptions,
		ControlFocusGuideBar,
		focusGuideBarBehavior{},
	)
	if err != nil {
		return nil, err
	}
	bar := &FocusGuideBar{controlHandle: controlHandle{state: state}}
	state.control = bar
	return bar, nil
}

// SetFocusGuidance atomically registers application-specific guidance for a
// control. Empty Text clears a prior customization.
func (a *App) SetFocusGuidance(
	control Control,
	guidance FocusGuidance,
) error {
	tx := a.NewTransaction()
	if err := tx.SetFocusGuidance(control, guidance); err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

// ClearFocusGuidance atomically removes application guidance for a control.
func (a *App) ClearFocusGuidance(control Control) error {
	return a.SetFocusGuidance(control, FocusGuidance{})
}

func normalizeFocusGuidance(
	kind ControlKind,
	guidance FocusGuidance,
) (focusGuidanceConfig, error) {
	if guidance.Text == "" {
		return focusGuidanceConfig{}, nil
	}
	switch guidance.Mode {
	case "", FocusGuidanceAppend:
		guidance.Mode = FocusGuidanceAppend
	case FocusGuidanceOverride:
	default:
		return focusGuidanceConfig{}, fmt.Errorf(
			"%w: invalid focus guidance mode %q",
			ErrInvalidControl,
			guidance.Mode,
		)
	}
	if len(guidance.Text) > MaxFocusGuidanceApplicationBytes {
		return focusGuidanceConfig{}, fmt.Errorf(
			"%w: focus guidance exceeds %d UTF-8 bytes",
			ErrTextLimit,
			MaxFocusGuidanceApplicationBytes,
		)
	}
	content, err := normalizeDisplayText(guidance.Text, false)
	if err != nil {
		return focusGuidanceConfig{}, err
	}
	config := focusGuidanceConfig{
		mode: guidance.Mode, content: content, set: true,
	}
	if _, err := resolvedFocusGuidance(kind, config); err != nil {
		return focusGuidanceConfig{}, err
	}
	return config, nil
}

func focusGuidanceConfigEqual(left, right focusGuidanceConfig) bool {
	return left.set == right.set &&
		left.mode == right.mode &&
		left.content.text == right.content.text
}

func genericFocusGuidance(kind ControlKind) string {
	switch kind {
	case ControlButton:
		return "Button: Enter or Space activates; Tab or Shift+Tab moves focus"
	case ControlCheckbox:
		return "Checkbox: Space changes state; Tab or Shift+Tab moves focus"
	case ControlRadioButton:
		return "Radio: arrows move focus; Space or Enter selects; Tab leaves group"
	case ControlCycleField:
		return "Cycle: [/] stop; Space/Enter next (wraps)"
	case ControlSelectField:
		return "Select: [/] stop; Space/Enter next (wraps)"
	case ControlTextField:
		return "Text field: Enter edits/commits; Esc cancels; arrows move caret"
	case ControlNumberField:
		return "Number field: Enter edits/commits; Esc cancels; arrows move caret"
	case ControlSpinBox:
		return "Spin box: [ decrements; ] increments; Enter edits"
	case ControlTextArea:
		return "Text area: Enter edits/newline; Ctrl-Enter commits; Esc cancels"
	case ControlMenuBar:
		return "Menu: arrows navigate; Enter activates; Esc closes"
	case ControlScrollBar:
		return "Scroll bar: arrows move; Page Up or Page Down pages; Home or End jumps"
	case ControlTabbedPanel, ControlNotebook:
		return "Tabs: Left or Right moves focus; Space or Enter selects; Tab leaves group"
	case ControlViewport, ControlScrollablePanel:
		return "Viewport: arrows scroll; Page Up or Page Down pages; Home or End jumps"
	case ControlMarkdownView:
		return "Markdown: arrows scroll; Page Up or Page Down pages; Home or End jumps"
	case ControlLogView, ControlStreamView:
		return "Content: arrows scroll; End resumes follow; Home or Page Up pauses"
	case ControlListBox:
		return "List: arrows move current; Space selects; Enter activates; Tab leaves"
	case ControlTreeView:
		return "Tree: arrows navigate; Space selects; Enter activates; +, -, * expand"
	case ControlDropDown:
		return "Drop-down: Space, Enter, F4, or Alt-Down opens; Escape cancels"
	case ControlComboBox:
		return "Combo: Enter or F2 edits; Space, F4, or Alt-Down opens choices"
	default:
		return "No focused control"
	}
}

func resolvedFocusGuidance(
	kind ControlKind,
	config focusGuidanceConfig,
) (normalizedDisplayText, error) {
	text := genericFocusGuidance(kind)
	if config.set {
		switch config.mode {
		case FocusGuidanceOverride:
			text = config.content.text
		case FocusGuidanceAppend:
			text += " | " + config.content.text
		}
	}
	return normalizeDisplayText(text, false)
}

func (a *App) focusGuideBarDetailsLocked() FocusGuideBarDetails {
	details := FocusGuideBarDetails{}
	kind := ControlKind("")
	config := focusGuidanceConfig{}
	if a.focus != nil && !a.focus.destroyed && !a.focus.aborted {
		details.Target = a.focus.id
		details.TargetKind = a.focus.kind
		kind = a.focus.kind
		config = a.focus.focusGuidance
		if config.set {
			details.Customization = config.mode
		}
	}
	content, _ := resolvedFocusGuidance(kind, config)
	details.Text = content.text
	return details
}

func (focusGuideBarBehavior) clientInset() int { return 0 }

func (focusGuideBarBehavior) paintDecoration(
	app *App,
	frame *IntendedFrame,
	state *controlState,
	absolute Rect,
	clip Rect,
) {
	details := app.focusGuideBarDetailsLocked()
	content, _ := normalizeDisplayText(details.Text, false)
	app.paintTextRowsLocked(
		frame,
		state,
		absolute,
		clip,
		content.lines,
		TextAlignStart,
		TextAlignStart,
	)
}

func (focusGuideBarBehavior) details() ControlDetails {
	return ControlDetails{
		Version:       ControlDetailsVersion,
		FocusGuideBar: &FocusGuideBarDetails{},
	}
}

func (focusGuideBarBehavior) intrinsicMinimum() Size {
	return Size{Width: 1, Height: 1}
}
