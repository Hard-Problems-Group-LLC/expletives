package expletives

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	// SnapshotVersion is the current in-process snapshot schema.
	SnapshotVersion = 1
	// MaxFrameCells bounds one materialized intended frame. The cell budget,
	// rather than an arbitrary axis length, is the allocation safety boundary.
	// It permits geometries such as 1200x1200.
	MaxFrameCells = 4 * 1024 * 1024
	// MaxRetainedFrameCells bounds App-owned exact snapshot history. History is
	// evicted by actual frame cost, so large frames retain fewer generations.
	MaxRetainedFrameCells = 2 * MaxFrameCells
	// MaxSnapshotHistoryRecords bounds non-frame metadata retained when frames
	// are tiny. The aggregate cell budget usually governs large applications.
	MaxSnapshotHistoryRecords = 64
	// SnapshotRetention is the compatibility name for the metadata record cap.
	SnapshotRetention = MaxSnapshotHistoryRecords
	// MaxFrameWidth and MaxFrameHeight are compatibility aliases for the
	// largest possible single axis under MaxFrameCells. New code should check
	// the aggregate cell budget instead.
	MaxFrameWidth  = MaxFrameCells
	MaxFrameHeight = MaxFrameCells
	// MaxControls includes the App-owned root in the active-control limit.
	MaxControls = 4096
	// MaxThemeStyles is the maximum number of definitions in one Theme.
	MaxThemeStyles = 4096
	// MaxTransactionOperations bounds one Transaction's recorded work.
	MaxTransactionOperations = 16384
	// MaxAutomationKeyBytes bounds stable identifiers, including automation
	// keys, scenarios, requests, commands, and result codes.
	MaxAutomationKeyBytes = 64
	// MaxTitleBytes bounds one border title's UTF-8 encoding.
	MaxTitleBytes = 256
	// MaxTitleCells follows the encoded title byte bound; no valid UTF-8 title
	// can contain more display cells than bytes.
	MaxTitleCells = MaxTitleBytes
	// MaxCellBytes bounds one canonical cell grapheme's UTF-8 encoding.
	MaxCellBytes = 64
	// MaxDisplayTextBytes bounds one retained Label, StaticText, or Rule text.
	// It keeps worst-case snapshot and automation evidence within their
	// aggregate resource budgets.
	MaxDisplayTextBytes = 256
	// MaxDisplayTextCells bounds canonical cells, including line separators,
	// in one Text/Display control.
	MaxDisplayTextCells = 256
	// MaxFocusGuidanceApplicationBytes bounds one application's focused-control
	// guidance addition or override. The resolved generic-plus-application
	// guidance remains subject to MaxDisplayTextBytes and MaxDisplayTextCells.
	MaxFocusGuidanceApplicationBytes = 128
	// MaxHotkeyBarItems bounds one HotkeyBar's copied ordered inventory.
	MaxHotkeyBarItems = 64
	// MaxStatusBarSegments bounds one StatusBar's copied ordered inventory.
	MaxStatusBarSegments = 64
	// MaxSelectionOptions bounds one copied CycleField, SelectField, or tab
	// inventory.
	MaxSelectionOptions = 256
	// MaxSelectionItems bounds aggregate RadioButton, copied fixed-option, and
	// copied Tab records across one App and its automation evidence.
	MaxSelectionItems = 1024
	// MaxTextInputBytes and MaxTextInputCells bound one editor value. These are
	// allocation and evidence limits, not conventional terminal geometry caps.
	MaxTextInputBytes = 64 << 10
	MaxTextInputCells = 64 << 10
	// MaxTextValidatorBytes and MaxTextValidatorCells bound one copied
	// whitelist or blacklist.
	MaxTextValidatorBytes = 4 << 10
	MaxTextValidatorCells = 1024
	// MaxTextInputAggregateBytes bounds values plus validator sets retained
	// across one App so immutable snapshots and automation remain bounded.
	MaxTextInputAggregateBytes = 256 << 10
	// MaxActionItems bounds aggregate HotkeyBar items and StatusBar segments
	// across one App.
	MaxActionItems = MaxControls
	// MaxMenuDepth bounds one MenuBar's immutable popup tree.
	MaxMenuDepth = 8
	// MaxMenuItemsPerMenu bounds direct items in one Menu.
	MaxMenuItemsPerMenu = 64
	// MaxMenus bounds aggregate popup Menu models attached to one App.
	MaxMenus = 256
	// MaxMenuItems bounds aggregate MenuItem descriptors attached to one App.
	MaxMenuItems = 512
	// MaxInputSources bounds sources that may hold keys concurrently.
	MaxInputSources = MaxControls
	// MaxHeldKeysPerSource bounds one source's simultaneous held-key state.
	MaxHeldKeysPerSource = 8
	// MaxConcurrentCommandHandlers bounds router calls across an App.
	MaxConcurrentCommandHandlers = 4
	// MaxCommandDescriptionBytes bounds a command description's UTF-8 encoding.
	MaxCommandDescriptionBytes = 256
	// MaxPublicMessageBytes bounds a result message's UTF-8 encoding.
	MaxPublicMessageBytes = 1024
	// MaxContentBytes bounds one retained Markdown, log, or stream source.
	MaxContentBytes = 64 << 10
	// MaxContentAggregateBytes bounds retained content source across one App.
	MaxContentAggregateBytes = 1 << 20
	// MaxContentRecords bounds retained or submitted log and stream records.
	MaxContentRecords = 4096
	// DefaultContentRecordCapacity is the zero-value record budget.
	DefaultContentRecordCapacity = 1024
	// DefaultContentByteCapacity is the zero-value canonical byte budget.
	DefaultContentByteCapacity = MaxContentBytes
	// MaxMarkdownBlocks bounds parsed blocks in one MarkdownView.
	MaxMarkdownBlocks = 4096
	// MaxMarkdownSummaries bounds structural block records in one control
	// snapshot. The first and last records are retained when truncation is
	// necessary so evidence remains useful without inflating worst-case wire
	// responses.
	MaxMarkdownSummaries = 4
	// MaxCollectionItems bounds one copied list, tree, or table row model.
	MaxCollectionItems = 4096
	// MaxCollectionColumns bounds one copied table or data-grid schema.
	MaxCollectionColumns = 256
	// MaxCollectionCells bounds copied table and data-grid cells across an App.
	MaxCollectionCells = 16384
	// MaxCollectionDepth bounds one copied TreeView hierarchy.
	MaxCollectionDepth = 64
	// MaxCollectionAggregateBytes bounds copied collection data across an App.
	MaxCollectionAggregateBytes = 1 << 20
	// DefaultCollectionPopupRows is the zero-value popup row request.
	DefaultCollectionPopupRows = 8
	// MaxCollectionPopupRows bounds one requested transient popup height.
	MaxCollectionPopupRows = 64
)

// Coordinates and Layout arithmetic share the frame allocation scale. This
// is an arithmetic/resource boundary, not a conventional terminal-size cap.
const maxCoordinateMagnitude = MaxFrameCells

// Point is a zero-based logical cell coordinate.
type Point struct {
	// X is the column.
	X int `json:"x"`
	// Y is the row.
	Y int `json:"y"`
}

// Size is a logical width and height in terminal cells.
type Size struct {
	// Width is the number of columns.
	Width int `json:"width"`
	// Height is the number of rows.
	Height int `json:"height"`
}

func (s Size) valid() bool {
	return s.Width >= 0 && s.Height >= 0
}

func validateSize(size Size) error {
	_, err := frameCellCount(size)
	return err
}

func frameCellCount(size Size) (int, error) {
	if !size.valid() {
		return 0, ErrInvalidGeometry
	}
	if size.Height != 0 && size.Width > int(^uint(0)>>1)/size.Height {
		return 0, fmt.Errorf("%w: frame cell count overflows int", ErrInvalidGeometry)
	}
	cells := size.Width * size.Height
	if cells > MaxFrameCells {
		return 0, fmt.Errorf(
			"%w: frame requires %d cells; allocation limit is %d",
			ErrInvalidGeometry,
			cells,
			MaxFrameCells,
		)
	}
	return cells, nil
}

// AspectRatio constrains a root rectangle by a positive terminal-cell width
// to height ratio. Its zero value disables aspect-ratio enforcement.
type AspectRatio struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RootConstraints optionally limit the App-owned root Panel inside the
// physical application surface. Zero minimum axes impose no lower bound; zero
// maximum axes are unbounded; a zero AspectRatio is disabled.
type RootConstraints struct {
	Minimum     Size        `json:"minimum"`
	Maximum     Size        `json:"maximum"`
	AspectRatio AspectRatio `json:"aspect_ratio"`
}

// BorderForm selects the one-cell decoration used by Frame and GroupBox.
// The empty value selects BorderSingle for backward-compatible construction.
type BorderForm string

const (
	BorderDefault     BorderForm = ""
	BorderNone        BorderForm = "none"
	BorderSingle      BorderForm = "single"
	BorderDouble      BorderForm = "double"
	BorderShadeLight  BorderForm = "shade_light"
	BorderShadeMedium BorderForm = "shade_medium"
	BorderShadeDark   BorderForm = "shade_dark"
	BorderBlock       BorderForm = "block"
)

// BorderOptions configures optional Layout decoration. Its zero value selects
// no Layout border; Frame and GroupBox retain their own single-line default.
type BorderOptions struct {
	Form BorderForm
	// Style defaults to "layout.border".
	Style StyleID
	// Foreground and Background independently override Style when non-nil.
	Foreground *Color
	Background *Color
}

// Rect is parent-relative logical cell geometry.
type Rect struct {
	// X and Y locate the top-left cell.
	X int `json:"x"`
	Y int `json:"y"`
	// Width and Height are nonnegative cell counts.
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (r Rect) valid() bool {
	return r.Width >= 0 && r.Height >= 0
}

func validateRect(rect Rect) error {
	if !rect.valid() {
		return ErrInvalidGeometry
	}
	values := []int{rect.X, rect.Y, rect.Width, rect.Height}
	for _, value := range values {
		if value < -maxCoordinateMagnitude || value > maxCoordinateMagnitude {
			return fmt.Errorf("%w: coordinate exceeds checked limit", ErrInvalidGeometry)
		}
	}
	if saturatingAdd(rect.X, rect.Width) > maxCoordinateMagnitude ||
		saturatingAdd(rect.Y, rect.Height) > maxCoordinateMagnitude {
		return fmt.Errorf("%w: extent exceeds checked limit", ErrInvalidGeometry)
	}
	return nil
}

// Empty reports whether r contains no cells.
func (r Rect) Empty() bool {
	return r.Width == 0 || r.Height == 0
}

// Intersect returns the overlap of r and other, which must use the same
// coordinate space.
func (r Rect) Intersect(other Rect) Rect {
	x1 := max(r.X, other.X)
	y1 := max(r.Y, other.Y)
	x2 := min(saturatingAdd(r.X, r.Width), saturatingAdd(other.X, other.Width))
	y2 := min(saturatingAdd(r.Y, r.Height), saturatingAdd(other.Y, other.Height))
	if x2 <= x1 || y2 <= y1 {
		return Rect{X: x1, Y: y1}
	}
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

func saturatingAdd(a, b int) int {
	if b > 0 && a > int(^uint(0)>>1)-b {
		return int(^uint(0) >> 1)
	}
	if b < 0 && a < -int(^uint(0)>>1)-1-b {
		return -int(^uint(0)>>1) - 1
	}
	return a + b
}

// ControlID is an App-scoped runtime identity. It is not stable across Apps
// or process runs; use an automation key for stable lookup.
type ControlID string

// ControlKind identifies the toolkit behavior represented by a snapshot node.
type ControlKind string

const (
	// ControlRoot identifies the sole App-owned root Panel.
	ControlRoot ControlKind = "root"
	// ControlPanel identifies a plain Panel.
	ControlPanel ControlKind = "panel"
	// ControlFrame identifies a bordered Frame.
	ControlFrame ControlKind = "frame"
	// ControlGroupBox identifies a titled bordered GroupBox.
	ControlGroupBox ControlKind = "group_box"
	// ControlLabel identifies a single-line non-container Label.
	ControlLabel ControlKind = "label"
	// ControlStaticText identifies multiline optionally wrapped display text.
	ControlStaticText ControlKind = "static_text"
	// ControlSeparator identifies an untitled structural divider.
	ControlSeparator ControlKind = "separator"
	// ControlRule identifies a titled structural divider.
	ControlRule ControlKind = "rule"
	// ControlButton identifies one focusable command-activation control.
	ControlButton ControlKind = "button"
	// ControlHotkeyBar identifies a non-focusable command-shortcut summary.
	ControlHotkeyBar ControlKind = "hotkey_bar"
	// ControlFocusGuideBar identifies dynamic focused-control guidance.
	ControlFocusGuideBar ControlKind = "focus_guide_bar"
	// ControlMenuBar identifies the persistent popup-menu session owner.
	ControlMenuBar ControlKind = "menu_bar"
	// ControlStatusBar identifies the persistent bottom-row status surface.
	ControlStatusBar ControlKind = "status_bar"
	// ControlHeader identifies one ordered top-edge one-row container.
	ControlHeader ControlKind = "header"
	// ControlFooter identifies one ordered bottom-edge one-row container.
	ControlFooter ControlKind = "footer"
	// ControlCheckbox identifies one two-state or three-state selector.
	ControlCheckbox ControlKind = "checkbox"
	// ControlRadioGroup identifies one exclusive-selection container.
	ControlRadioGroup ControlKind = "radio_group"
	// ControlRadioButton identifies one option owned by a RadioGroup.
	ControlRadioButton ControlKind = "radio_button"
	// ControlCycleField identifies one fixed-option cycling field.
	ControlCycleField ControlKind = "cycle_field"
	// ControlSelectField is the SelectField naming variant of CycleField.
	ControlSelectField ControlKind = "select_field"
	// ControlTextField identifies one focusable single-line editor.
	ControlTextField ControlKind = "text_field"
	// ControlNumberField identifies one focusable bounded decimal editor.
	ControlNumberField ControlKind = "number_field"
	// ControlSpinBox identifies one bracket-steppable NumberField.
	ControlSpinBox ControlKind = "spin_box"
	// ControlTextArea identifies one focusable multiline editor.
	ControlTextArea ControlKind = "text_area"
	// ControlProgressBar identifies one determinate or indeterminate bar.
	ControlProgressBar ControlKind = "progress_bar"
	// ControlMeter identifies one horizontal or vertical scalar meter.
	ControlMeter ControlKind = "meter"
	// ControlSpinner identifies one deterministic single-cell activity mark.
	ControlSpinner ControlKind = "spinner"
	// ControlActivityDots identifies one deterministic dot activity mark.
	ControlActivityDots ControlKind = "activity_dots"
	// ControlScrollBar identifies one focusable viewport-position control.
	ControlScrollBar ControlKind = "scroll_bar"
	// ControlTabbedPanel identifies one focusable stacked-page container.
	ControlTabbedPanel ControlKind = "tabbed_panel"
	// ControlNotebook identifies the Notebook naming variant of TabbedPanel.
	ControlNotebook ControlKind = "notebook"
	// ControlViewport identifies one unframed scrolling container.
	ControlViewport ControlKind = "viewport"
	// ControlScrollablePanel identifies one framed scrolling container with
	// integrated scrollbar decoration.
	ControlScrollablePanel ControlKind = "scrollable_panel"
	// ControlMarkdownView identifies one read-only rendered Markdown leaf.
	ControlMarkdownView ControlKind = "markdown_view"
	// ControlLogView identifies one bounded structured-log leaf.
	ControlLogView ControlKind = "log_view"
	// ControlStreamView identifies one bounded line-oriented stream leaf.
	ControlStreamView ControlKind = "stream_view"
	// ControlListBox identifies one bounded stable-identity item list.
	ControlListBox ControlKind = "list_box"
	// ControlDropDown identifies one selection-only collapsed popup field.
	ControlDropDown ControlKind = "drop_down"
	// ControlComboBox identifies one editable collapsed popup field.
	ControlComboBox ControlKind = "combo_box"
)

// TextAlignment selects placement on one logical control axis. Its empty
// value selects TextAlignStart.
type TextAlignment string

const (
	TextAlignDefault TextAlignment = ""
	TextAlignStart   TextAlignment = "start"
	TextAlignCenter  TextAlignment = "center"
	TextAlignEnd     TextAlignment = "end"
)

// TextWrap selects StaticText line wrapping. Its empty value selects
// TextWrapNone.
type TextWrap string

const (
	TextWrapDefault TextWrap = ""
	TextWrapNone    TextWrap = "none"
	TextWrapWords   TextWrap = "words"
	TextWrapCells   TextWrap = "cells"
)

// StyleID is a stable semantic reference into an App Theme.
type StyleID string

// StyleAttributes is a bounded terminal-independent attribute bit set. Zero
// requests no attributes.
type StyleAttributes uint16

const (
	// StyleBold requests emphasized weight.
	StyleBold StyleAttributes = 1 << iota
	// StyleDim requests reduced intensity.
	StyleDim
	// StyleItalic requests italic presentation.
	StyleItalic
	// StyleUnderline requests underlined presentation.
	StyleUnderline
	// StyleReverse requests swapped foreground and background presentation.
	StyleReverse
)

const supportedStyleAttributes = StyleBold |
	StyleDim |
	StyleItalic |
	StyleUnderline |
	StyleReverse

// Color is a resolved 24-bit color. Its JSON representation is #RRGGBB.
type Color struct {
	// R, G, and B are eight-bit red, green, and blue components.
	R uint8
	G uint8
	B uint8
}

// RGB constructs a resolved 24-bit Color.
func RGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b}
}

// String returns c in uppercase #RRGGBB form.
func (c Color) String() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// MarshalJSON encodes c as a #RRGGBB JSON string.
func (c Color) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// UnmarshalJSON decodes a #RRGGBB JSON string into c.
func (c *Color) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return fmt.Errorf("decode color: %w", err)
	}
	if len(encoded) != 7 || encoded[0] != '#' {
		return fmt.Errorf("decode color %q: expected #RRGGBB", encoded)
	}
	value, err := strconv.ParseUint(encoded[1:], 16, 24)
	if err != nil {
		return fmt.Errorf("decode color %q: %w", encoded, err)
	}
	c.R = uint8(value >> 16)
	c.G = uint8(value >> 8)
	c.B = uint8(value)
	return nil
}

// ResolvedStyle is the visual value supplied by a Theme for a StyleID.
type ResolvedStyle struct {
	// Foreground is the intended text color.
	Foreground Color `json:"foreground"`
	// Background is the intended cell background.
	Background Color `json:"background"`
	// Attributes contains only the supported StyleAttributes bits.
	Attributes StyleAttributes `json:"attributes,omitempty"`
}

// Style is one semantic Theme definition. Controls retain only its ID.
type Style struct {
	// ID is the semantic identifier referenced by controls.
	ID StyleID `json:"id"`
	// Foreground and Background are the resolved intended colors.
	Foreground Color `json:"foreground"`
	Background Color `json:"background"`
	// Attributes contains the terminal-independent presentation requests.
	Attributes StyleAttributes `json:"attributes,omitempty"`
}

// Resolved returns the immutable visual value of this Theme definition.
func (s Style) Resolved() ResolvedStyle {
	return ResolvedStyle{
		Foreground: s.Foreground,
		Background: s.Background,
		Attributes: s.Attributes,
	}
}

func styleDefinition(id StyleID, resolved ResolvedStyle) Style {
	return Style{
		ID:         id,
		Foreground: resolved.Foreground,
		Background: resolved.Background,
		Attributes: resolved.Attributes,
	}
}

func normalizeStyleID(id StyleID, fallback StyleID) (StyleID, error) {
	if id == "" {
		id = fallback
	}
	if !validBoundedIdentifier(string(id)) {
		return "", fmt.Errorf("%w: invalid semantic style ID", ErrStyleMissing)
	}
	return id, nil
}

func validBoundedIdentifier(value string) bool {
	if value == "" || len(value) > MaxAutomationKeyBytes {
		return false
	}
	for _, current := range value {
		if (current >= 'a' && current <= 'z') ||
			(current >= 'A' && current <= 'Z') ||
			(current >= '0' && current <= '9') ||
			strings.ContainsRune("._:-", current) {
			continue
		}
		return false
	}
	return true
}

// Cell is one canonical single-cell intended-frame element.
type Cell struct {
	// Grapheme is exactly one canonical display cell.
	Grapheme string `json:"grapheme"`
	// Style is the semantic style ID used to render this cell.
	Style StyleID `json:"style"`
	// Foreground, Background, and Attributes are the resolved intended style.
	Foreground Color           `json:"foreground"`
	Background Color           `json:"background"`
	Attributes StyleAttributes `json:"attributes,omitempty"`
	// Owner is the control that most recently painted this cell.
	Owner ControlID `json:"owner"`
}

// IntendedFrame is the pure terminal-independent character-cell surface.
type IntendedFrame struct {
	// Size is the frame geometry.
	Size Size `json:"size"`
	// Cells contains Size.Width*Size.Height cells in row-major order.
	Cells []Cell `json:"cells"`
}

// Cell returns the cell at column x and row y. It returns false when the
// coordinate or frame storage is out of bounds.
func (f IntendedFrame) Cell(x, y int) (Cell, bool) {
	if x < 0 || y < 0 || x >= f.Size.Width || y >= f.Size.Height {
		return Cell{}, false
	}
	index := y*f.Size.Width + x
	if index < 0 || index >= len(f.Cells) {
		return Cell{}, false
	}
	return f.Cells[index], true
}

// CursorState describes the intended logical cursor.
type CursorState struct {
	// Visible controls whether the physical presenter should show a cursor.
	Visible bool `json:"visible"`
	// Position is meaningful when Visible is true.
	Position Point `json:"position"`
}

// ControlSnapshot is the bounded semantic observation of one control node.
type ControlSnapshot struct {
	// ID is the App-scoped runtime identity.
	ID ControlID `json:"id"`
	// Key is the optional stable automation key.
	Key string `json:"key,omitempty"`
	// Kind identifies the built-in control behavior.
	Kind ControlKind `json:"kind"`
	// Parent is empty only for the App root.
	Parent ControlID `json:"parent,omitempty"`
	// Children preserves direct-child insertion order.
	Children []ControlID `json:"children,omitempty"`
	// Bounds is parent-client-relative logical geometry.
	Bounds Rect `json:"bounds"`
	// AbsoluteBounds is unclipped App-relative logical geometry.
	AbsoluteBounds Rect `json:"absolute_bounds"`
	// EffectiveClip is the intersection of visibility, ancestors, and surface.
	EffectiveClip Rect `json:"effective_clip"`
	// Minimum is the control's declared logical minimum.
	Minimum Size `json:"minimum"`
	// Layout identifies the nonvisual Layout managing Bounds, when any.
	Layout LayoutID `json:"layout,omitempty"`
	// LayoutIndex and StackIndex expose immutable arrangement order and current
	// peer-kind stack order. They are -1 for unmanaged controls.
	LayoutIndex int `json:"layout_index"`
	StackIndex  int `json:"stack_index"`
	// Style and ResolvedStyle pair semantic and visual state.
	Style         StyleID       `json:"style"`
	ResolvedStyle ResolvedStyle `json:"resolved_style"`
	// Visible reports effective visibility through the ancestor chain.
	Visible bool `json:"visible"`
	// Focused reports that this control owns the App's keyboard focus.
	Focused bool `json:"focused"`
	// Details is the versioned control-specific state.
	Details ControlDetails `json:"details"`
}

// ControlDetailsVersion is the current ControlDetails union version.
const ControlDetailsVersion = 1

// ControlDetails is the bounded typed state union for control-specific
// observation. Later controls add typed members rather than string-keyed
// property bags.
type ControlDetails struct {
	// Version identifies the typed union schema.
	Version int `json:"version"`
	// Container is present for controls that can parent children.
	Container *ContainerDetails `json:"container,omitempty"`
	// Border is present for bordered controls.
	Border *BorderDetails `json:"border,omitempty"`
	// Text is present for Label and StaticText.
	Text *TextDetails `json:"text,omitempty"`
	// Divider is present for Separator and Rule.
	Divider *DividerDetails `json:"divider,omitempty"`
	// Action is present for Button and later command-activation controls.
	Action *ActionDetails `json:"action,omitempty"`
	// HotkeyBar is present for HotkeyBar.
	HotkeyBar *HotkeyBarDetails `json:"hotkey_bar,omitempty"`
	// MenuBar is present for MenuBar.
	MenuBar *MenuBarDetails `json:"menu_bar,omitempty"`
	// StatusBar is present for StatusBar.
	StatusBar *StatusBarDetails `json:"status_bar,omitempty"`
	// Checkbox is present for Checkbox.
	Checkbox *CheckboxDetails `json:"checkbox,omitempty"`
	// RadioButton is present for RadioButton.
	RadioButton *RadioButtonDetails `json:"radio_button,omitempty"`
	// RadioGroup is present for RadioGroup.
	RadioGroup *RadioGroupDetails `json:"radio_group,omitempty"`
	// ChoiceField is present for CycleField and SelectField.
	ChoiceField *ChoiceFieldDetails `json:"choice_field,omitempty"`
	// FocusGuideBar is present for FocusGuideBar.
	FocusGuideBar *FocusGuideBarDetails `json:"focus_guide_bar,omitempty"`
	// TextField is present for TextField.
	TextField *TextFieldDetails `json:"text_field,omitempty"`
	// NumberField is present for NumberField and SpinBox.
	NumberField *NumberFieldDetails `json:"number_field,omitempty"`
	// TextArea is present for TextArea.
	TextArea *TextAreaDetails `json:"text_area,omitempty"`
	// Progress is present for ProgressBar, Meter, Spinner, and ActivityDots.
	Progress *ProgressDetails `json:"progress,omitempty"`
	// ScrollBar is present for ScrollBar.
	ScrollBar *ScrollBarDetails `json:"scroll_bar,omitempty"`
	// TabbedPanel is present for TabbedPanel and Notebook.
	TabbedPanel *TabbedPanelDetails `json:"tabbed_panel,omitempty"`
	// Scrollable is present for Viewport and ScrollablePanel.
	Scrollable *ScrollableDetails `json:"scrollable,omitempty"`
	// Markdown is present for MarkdownView.
	Markdown *MarkdownDetails `json:"markdown,omitempty"`
	// LogView is present for LogView.
	LogView *LogViewDetails `json:"log_view,omitempty"`
	// StreamView is present for StreamView.
	StreamView *StreamViewDetails `json:"stream_view,omitempty"`
	// ListBox is present for ListBox.
	ListBox *ListBoxDetails `json:"list_box,omitempty"`
	// DropDown is present for DropDown.
	DropDown *DropDownDetails `json:"drop_down,omitempty"`
	// ComboBox is present for ComboBox.
	ComboBox *ComboBoxDetails `json:"combo_box,omitempty"`
}

// ContainerDetails describes the client-area behavior of a container.
type ContainerDetails struct {
	// ClientInset is the number of cells reserved on every edge.
	ClientInset int `json:"client_inset"`
}

// BorderDetails describes one bordered container.
type BorderDetails struct {
	// Title is the caller-supplied title before cell normalization.
	Title string `json:"title,omitempty"`
	// Form identifies the canonical control-owned border geometry.
	Form BorderForm `json:"form"`
	// Style and ResolvedStyle pair the border's semantic and visual state.
	Style         StyleID       `json:"style"`
	ResolvedStyle ResolvedStyle `json:"resolved_style"`
	// ForegroundOverride and BackgroundOverride are present when construction
	// replaced that component of the Theme-resolved border style.
	ForegroundOverride *Color `json:"foreground_override,omitempty"`
	BackgroundOverride *Color `json:"background_override,omitempty"`
}

// TextDetails describes one canonical Label or StaticText value.
type TextDetails struct {
	Text                string        `json:"text"`
	HorizontalAlignment TextAlignment `json:"horizontal_alignment"`
	VerticalAlignment   TextAlignment `json:"vertical_alignment"`
	Wrap                TextWrap      `json:"wrap"`
	Target              ControlID     `json:"target,omitempty"`
	Mnemonic            Key           `json:"mnemonic,omitempty"`
}

// DividerDetails describes one Separator or Rule.
type DividerDetails struct {
	Orientation Orientation   `json:"orientation"`
	Form        BorderForm    `json:"form"`
	Text        string        `json:"text,omitempty"`
	Alignment   TextAlignment `json:"alignment"`
}

// ActionDetails describes one command-backed activation control.
type ActionDetails struct {
	Label          string    `json:"label"`
	Command        CommandID `json:"command"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
	Checked        bool      `json:"checked"`
	Mnemonic       Key       `json:"mnemonic,omitempty"`
	Pressed        bool      `json:"pressed"`
	Default        bool      `json:"default"`
	Cancel         bool      `json:"cancel"`
}

// HotkeyBarItemDetails describes one command and its current first binding.
type HotkeyBarItemDetails struct {
	Label          string    `json:"label"`
	Command        CommandID `json:"command"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
	Checked        bool      `json:"checked"`
	Chord          *Chord    `json:"chord,omitempty"`
}

// HotkeyBarDetails describes one bounded ordered command summary.
type HotkeyBarDetails struct {
	Items []HotkeyBarItemDetails `json:"items"`
}

// MenuEntryDetails is one flattened immutable MenuItem observation.
type MenuEntryDetails struct {
	Key            string           `json:"key"`
	ParentKey      string           `json:"parent_key,omitempty"`
	Depth          int              `json:"depth"`
	Kind           MenuItemKind     `json:"kind"`
	Label          string           `json:"label,omitempty"`
	Command        CommandID        `json:"command,omitempty"`
	Enabled        bool             `json:"enabled"`
	DisabledReason string           `json:"disabled_reason,omitempty"`
	Checked        bool             `json:"checked"`
	Mnemonic       Key              `json:"mnemonic,omitempty"`
	Placement      MenuBarPlacement `json:"placement,omitempty"`
	Chord          *Chord           `json:"chord,omitempty"`
	Selected       bool             `json:"selected"`
	Open           bool             `json:"open"`
	ChildCount     int              `json:"child_count"`
}

// MenuBarDetails is the bounded flattened state of one MenuBar and session.
type MenuBarDetails struct {
	Entries      []MenuEntryDetails `json:"entries"`
	OpenPath     []string           `json:"open_path"`
	SelectedPath []string           `json:"selected_path"`
}

// StatusSegmentDetails describes one current contextual or command segment.
type StatusSegmentDetails struct {
	Key            string    `json:"key"`
	Label          string    `json:"label"`
	Command        CommandID `json:"command,omitempty"`
	Priority       int       `json:"priority"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
	Checked        bool      `json:"checked"`
	Chord          *Chord    `json:"chord,omitempty"`
	Rendered       bool      `json:"rendered"`
	Bounds         Rect      `json:"bounds"`
	Clipped        bool      `json:"clipped"`
}

// StatusBarDetails describes one bounded ordered bottom-row status inventory.
type StatusBarDetails struct {
	Segments []StatusSegmentDetails `json:"segments"`
}

// CheckboxDetails describes one current Checkbox value and policy.
type CheckboxDetails struct {
	Label          string     `json:"label"`
	State          CheckState `json:"state"`
	ThreeState     bool       `json:"three_state"`
	Enabled        bool       `json:"enabled"`
	DisabledReason string     `json:"disabled_reason,omitempty"`
	Mnemonic       Key        `json:"mnemonic,omitempty"`
	ChangeCommand  CommandID  `json:"change_command,omitempty"`
}

// RadioButtonDetails describes one option owned by a RadioGroup.
type RadioButtonDetails struct {
	Value          string `json:"value"`
	Label          string `json:"label"`
	Selected       bool   `json:"selected"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Mnemonic       Key    `json:"mnemonic,omitempty"`
}

// RadioOptionDetails is one direct RadioButton summary in group order.
type RadioOptionDetails struct {
	Control        ControlID `json:"control"`
	Value          string    `json:"value"`
	Label          string    `json:"label"`
	Selected       bool      `json:"selected"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
}

// RadioGroupDetails describes one exclusive selection and its direct options.
type RadioGroupDetails struct {
	Value          string               `json:"value,omitempty"`
	AllowEmpty     bool                 `json:"allow_empty"`
	Enabled        bool                 `json:"enabled"`
	DisabledReason string               `json:"disabled_reason,omitempty"`
	ChangeCommand  CommandID            `json:"change_command,omitempty"`
	Options        []RadioOptionDetails `json:"options"`
}

// SelectionOptionDetails is one copied fixed choice.
type SelectionOptionDetails struct {
	Value          string `json:"value"`
	Label          string `json:"label"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Selected       bool   `json:"selected"`
}

// ChoiceFieldDetails describes one CycleField or SelectField value.
type ChoiceFieldDetails struct {
	Label          string                   `json:"label"`
	Value          string                   `json:"value,omitempty"`
	SelectedIndex  int                      `json:"selected_index"`
	Enabled        bool                     `json:"enabled"`
	DisabledReason string                   `json:"disabled_reason,omitempty"`
	Mnemonic       Key                      `json:"mnemonic,omitempty"`
	ChangeCommand  CommandID                `json:"change_command,omitempty"`
	Options        []SelectionOptionDetails `json:"options"`
}

// FocusGuideBarDetails describes the guidance resolved for current focus.
type FocusGuideBarDetails struct {
	// Target and TargetKind identify the focused control, when any.
	Target     ControlID   `json:"target,omitempty"`
	TargetKind ControlKind `json:"target_kind,omitempty"`
	// Text is the exact canonical one-row guidance currently rendered.
	Text string `json:"text"`
	// Customization reports append or override when application guidance is
	// registered for Target.
	Customization FocusGuidanceMode `json:"customization,omitempty"`
}

// TextValidatorDetails describes one copied TextField character-set policy.
type TextValidatorDetails struct {
	Enforcement TextValidationEnforcement `json:"enforcement"`
	Mode        TextValidationMode        `json:"mode"`
	Characters  string                    `json:"characters"`
}

// TextFieldDetails describes current single-line editing and validation state.
// Text is always empty when Redacted is true.
type TextFieldDetails struct {
	Text           string                `json:"text,omitempty"`
	Length         int                   `json:"length"`
	Caret          int                   `json:"caret"`
	SelectionStart int                   `json:"selection_start"`
	SelectionEnd   int                   `json:"selection_end"`
	ViewOffset     int                   `json:"view_offset"`
	Editing        bool                  `json:"editing"`
	Valid          bool                  `json:"valid"`
	Password       bool                  `json:"password"`
	Redacted       bool                  `json:"redacted"`
	Enabled        bool                  `json:"enabled"`
	DisabledReason string                `json:"disabled_reason,omitempty"`
	ChangeCommand  CommandID             `json:"change_command,omitempty"`
	Validator      *TextValidatorDetails `json:"validator,omitempty"`
}

// NumberFieldDetails describes current decimal editing, range, and step
// state. Step is positive for SpinBox and zero for NumberField.
type NumberFieldDetails struct {
	Text           string    `json:"text"`
	Value          float64   `json:"value"`
	Length         int       `json:"length"`
	Caret          int       `json:"caret"`
	SelectionStart int       `json:"selection_start"`
	SelectionEnd   int       `json:"selection_end"`
	ViewOffset     int       `json:"view_offset"`
	Editing        bool      `json:"editing"`
	Valid          bool      `json:"valid"`
	InvalidReason  string    `json:"invalid_reason,omitempty"`
	Minimum        *float64  `json:"minimum,omitempty"`
	Maximum        *float64  `json:"maximum,omitempty"`
	DecimalPlaces  int       `json:"decimal_places"`
	Step           float64   `json:"step,omitempty"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
	ChangeCommand  CommandID `json:"change_command,omitempty"`
}

// TextAreaDetails describes current multiline editing, selection, wrapping,
// and private viewport state. Text is empty when Redacted is true.
type TextAreaDetails struct {
	Text              string                `json:"text,omitempty"`
	Length            int                   `json:"length"`
	LineCount         int                   `json:"line_count"`
	Caret             int                   `json:"caret"`
	SelectionStart    int                   `json:"selection_start"`
	SelectionEnd      int                   `json:"selection_end"`
	VisualCaretRow    int                   `json:"visual_caret_row"`
	VisualCaretColumn int                   `json:"visual_caret_column"`
	RowOffset         int                   `json:"row_offset"`
	ColumnOffset      int                   `json:"column_offset"`
	Wrap              TextWrap              `json:"wrap"`
	Editing           bool                  `json:"editing"`
	Valid             bool                  `json:"valid"`
	Password          bool                  `json:"password"`
	Redacted          bool                  `json:"redacted"`
	Enabled           bool                  `json:"enabled"`
	DisabledReason    string                `json:"disabled_reason,omitempty"`
	ChangeCommand     CommandID             `json:"change_command,omitempty"`
	Validator         *TextValidatorDetails `json:"validator,omitempty"`
}

// ProgressDetails describes one kind-consistent progress or activity state.
type ProgressDetails struct {
	Status        ProgressStatus   `json:"status"`
	Current       uint64           `json:"current,omitempty"`
	Total         uint64           `json:"total,omitempty"`
	Value         float64          `json:"value,omitempty"`
	Minimum       float64          `json:"minimum,omitempty"`
	Maximum       float64          `json:"maximum,omitempty"`
	Orientation   Orientation      `json:"orientation"`
	Indeterminate bool             `json:"indeterminate"`
	Tick          uint64           `json:"tick,omitempty"`
	ReducedMotion bool             `json:"reduced_motion"`
	TextMode      ProgressTextMode `json:"text_mode"`
	FrameIndex    int              `json:"frame_index"`
}

// ScrollBarDetails describes one canonical viewport and rendered thumb.
// TrackStart and ThumbStart are relative to the control's orientation axis.
type ScrollBarDetails struct {
	Orientation    Orientation `json:"orientation"`
	ContentSize    int         `json:"content_size"`
	ViewportSize   int         `json:"viewport_size"`
	Offset         int         `json:"offset"`
	MaximumOffset  int         `json:"maximum_offset"`
	ArrowStep      int         `json:"arrow_step"`
	PageStep       int         `json:"page_step"`
	TrackStart     int         `json:"track_start"`
	TrackSize      int         `json:"track_size"`
	ThumbStart     int         `json:"thumb_start"`
	ThumbSize      int         `json:"thumb_size"`
	Enabled        bool        `json:"enabled"`
	DisabledReason string      `json:"disabled_reason,omitempty"`
	ChangeCommand  CommandID   `json:"change_command,omitempty"`
}

// ScrollableDetails describes one generic two-axis logical content viewport.
// ViewportBounds and integrated bar bounds are relative to the owner control.
type ScrollableDetails struct {
	State             ViewportState       `json:"state"`
	MaximumOffset     Point               `json:"maximum_offset"`
	ViewportBounds    Rect                `json:"viewport_bounds"`
	ArrowStep         Size                `json:"arrow_step"`
	PageStep          Size                `json:"page_step"`
	Enabled           bool                `json:"enabled"`
	DisabledReason    string              `json:"disabled_reason,omitempty"`
	ChangeCommand     CommandID           `json:"change_command,omitempty"`
	Content           ControlID           `json:"content"`
	ContentKey        string              `json:"content_key,omitempty"`
	HorizontalPolicy  ScrollBarVisibility `json:"horizontal_policy"`
	VerticalPolicy    ScrollBarVisibility `json:"vertical_policy"`
	HorizontalVisible bool                `json:"horizontal_visible"`
	VerticalVisible   bool                `json:"vertical_visible"`
	HorizontalBar     *ScrollBarDetails   `json:"horizontal_bar,omitempty"`
	VerticalBar       *ScrollBarDetails   `json:"vertical_bar,omitempty"`
}

// MarkdownBlockDetails is one bounded structural Markdown block summary.
type MarkdownBlockDetails struct {
	Kind          string `json:"kind"`
	Level         int    `json:"level,omitempty"`
	SourceLine    int    `json:"source_line"`
	SourceLines   int    `json:"source_lines"`
	RenderedStart int    `json:"rendered_start"`
	RenderedRows  int    `json:"rendered_rows"`
}

// MarkdownDetails describes bounded source structure and derived viewport
// geometry without duplicating the retained source.
type MarkdownDetails struct {
	SourceBytes        int                    `json:"source_bytes"`
	SourceCells        int                    `json:"source_cells"`
	BlockCount         int                    `json:"block_count"`
	RenderedRows       int                    `json:"rendered_rows"`
	MaximumLineWidth   int                    `json:"maximum_line_width"`
	Viewport           ScrollableDetails      `json:"viewport"`
	Blocks             []MarkdownBlockDetails `json:"blocks"`
	SummariesTruncated bool                   `json:"summaries_truncated"`
}

// LogViewDetails describes bounded structured-log retention and viewport
// state without duplicating retained record text.
type LogViewDetails struct {
	Capacity        ContentCapacity   `json:"capacity"`
	RetainedRecords int               `json:"retained_records"`
	RetainedBytes   int               `json:"retained_bytes"`
	DroppedRecords  uint64            `json:"dropped_records"`
	DroppedBytes    uint64            `json:"dropped_bytes"`
	FirstKey        string            `json:"first_key,omitempty"`
	LastKey         string            `json:"last_key,omitempty"`
	Follow          bool              `json:"follow"`
	Viewport        ScrollableDetails `json:"viewport"`
}

// StreamViewDetails describes bounded line and partial-stream retention
// without duplicating complete off-screen stream content.
type StreamViewDetails struct {
	Capacity            ContentCapacity   `json:"capacity"`
	RetainedLines       int               `json:"retained_lines"`
	RetainedBytes       int               `json:"retained_bytes"`
	PendingBytes        int               `json:"pending_bytes"`
	PendingCells        int               `json:"pending_cells"`
	PendingStorageBytes int               `json:"pending_storage_bytes"`
	PendingTruncated    bool              `json:"pending_truncated"`
	DroppedLines        uint64            `json:"dropped_lines"`
	DroppedBytes        uint64            `json:"dropped_bytes"`
	Follow              bool              `json:"follow"`
	Viewport            ScrollableDetails `json:"viewport"`
}

// ListBoxDetails describes compact stable-identity list state without
// duplicating the retained item model.
type ListBoxDetails struct {
	Status           CollectionStatus        `json:"status"`
	StatusMessage    string                  `json:"status_message,omitempty"`
	ItemCount        int                     `json:"item_count"`
	EnabledCount     int                     `json:"enabled_count"`
	RetainedBytes    int                     `json:"retained_bytes"`
	Current          string                  `json:"current,omitempty"`
	CurrentIndex     int                     `json:"current_index"`
	SelectionMode    CollectionSelectionMode `json:"selection_mode"`
	RequireSelection bool                    `json:"require_selection"`
	SelectedCount    int                     `json:"selected_count"`
	FirstSelected    string                  `json:"first_selected,omitempty"`
	LastSelected     string                  `json:"last_selected,omitempty"`
	SelectionDigest  string                  `json:"selection_digest"`
	Enabled          bool                    `json:"enabled"`
	DisabledReason   string                  `json:"disabled_reason,omitempty"`
	ChangeCommand    CommandID               `json:"change_command,omitempty"`
	ActivateCommand  CommandID               `json:"activate_command,omitempty"`
	Viewport         ScrollableDetails       `json:"viewport"`
}

// DropDownDetails describes one collapsed field and its optional transient
// popup without duplicating the retained item model.
type DropDownDetails struct {
	ItemCount       int       `json:"item_count"`
	EnabledCount    int       `json:"enabled_count"`
	RetainedBytes   int       `json:"retained_bytes"`
	Current         string    `json:"current,omitempty"`
	CurrentIndex    int       `json:"current_index"`
	Selected        string    `json:"selected,omitempty"`
	SelectedIndex   int       `json:"selected_index"`
	AllowEmpty      bool      `json:"allow_empty"`
	PopupRows       int       `json:"popup_rows"`
	Open            bool      `json:"open"`
	PopupBounds     Rect      `json:"popup_bounds"`
	PopupOffset     int       `json:"popup_offset"`
	PopupCurrent    string    `json:"popup_current,omitempty"`
	PopupSelection  string    `json:"popup_selection,omitempty"`
	Enabled         bool      `json:"enabled"`
	DisabledReason  string    `json:"disabled_reason,omitempty"`
	ChangeCommand   CommandID `json:"change_command,omitempty"`
	ActivateCommand CommandID `json:"activate_command,omitempty"`
}

// ComboBoxDetails combines popup selection with the exact embedded editor
// state used for rendering and input.
type ComboBoxDetails struct {
	Popup  DropDownDetails  `json:"popup"`
	Editor TextFieldDetails `json:"editor"`
}

// TabDetails describes one copied page descriptor and its rendered strip
// state. Bounds is relative to the owning tab container.
type TabDetails struct {
	Key            string    `json:"key"`
	Value          string    `json:"value"`
	Label          string    `json:"label"`
	Mnemonic       Key       `json:"mnemonic,omitempty"`
	Page           ControlID `json:"page"`
	PageKey        string    `json:"page_key,omitempty"`
	Enabled        bool      `json:"enabled"`
	DisabledReason string    `json:"disabled_reason,omitempty"`
	Selected       bool      `json:"selected"`
	Current        bool      `json:"current"`
	Bounds         Rect      `json:"bounds"`
	Omitted        bool      `json:"omitted"`
	Clipped        bool      `json:"clipped"`
}

// TabbedPanelDetails describes one ordered tab strip and page selection.
type TabbedPanelDetails struct {
	Tabs            []TabDetails `json:"tabs"`
	Selected        string       `json:"selected,omitempty"`
	Current         string       `json:"current,omitempty"`
	ChangeCommand   CommandID    `json:"change_command,omitempty"`
	LeadingOmitted  bool         `json:"leading_omitted"`
	TrailingOmitted bool         `json:"trailing_omitted"`
}

// InputSourceSnapshot reports held logical keys for one isolated source.
type InputSourceSnapshot struct {
	// Source is the caller-selected input-source ID.
	Source string `json:"source"`
	// Held is a sorted copy of currently held keys.
	Held []Key `json:"held"`
}

// OverflowSnapshot reserves the typed snapshot surface used by Basic Layouts.
// The root/container slice publishes an empty collection.
type OverflowSnapshot struct {
	// EpisodeID identifies one continuous overflow episode.
	EpisodeID string `json:"episode_id"`
	// Panel identifies the overflowing container.
	Panel ControlID `json:"panel"`
	// Layout identifies the overflowing Layout.
	Layout LayoutID `json:"layout"`
	// Available and Required describe the layout-space comparison.
	Available Size `json:"available"`
	Required  Size `json:"required"`
	// Deficit is the missing width and height.
	Deficit Size `json:"deficit"`
	// State is the bounded overflow lifecycle state.
	State string `json:"state"`
}

// LayoutItemSnapshot describes one ordered Panel or nested Layout item.
type LayoutItemSnapshot struct {
	Kind        string    `json:"kind"`
	Panel       ControlID `json:"panel,omitempty"`
	Layout      LayoutID  `json:"layout,omitempty"`
	Bounds      Rect      `json:"bounds"`
	Minimum     Size      `json:"minimum"`
	LayoutIndex int       `json:"layout_index"`
	StackIndex  int       `json:"stack_index"`
}

// LayoutSnapshot is the bounded semantic observation of one Layout.
type LayoutSnapshot struct {
	ID          LayoutID       `json:"id"`
	Key         string         `json:"key,omitempty"`
	Kind        LayoutKind     `json:"kind"`
	Owner       ControlID      `json:"owner"`
	Parent      LayoutID       `json:"parent,omitempty"`
	Bounds      Rect           `json:"bounds"`
	OwnerBounds Rect           `json:"owner_bounds"`
	Minimum     Size           `json:"minimum"`
	Border      *BorderDetails `json:"border"`
	// LayoutIndex is fixed attachment/arrangement order; StackIndex is the
	// current position among Layout peers with the same parent.
	LayoutIndex int                  `json:"layout_index"`
	StackIndex  int                  `json:"stack_index"`
	Items       []LayoutItemSnapshot `json:"items"`
}

// Snapshot is one atomic toolkit observation. App snapshot methods return deep
// copies, so caller mutation cannot affect the App or retained history.
// Automation projects Snapshot into its own versioned wire DTO.
type Snapshot struct {
	// Version identifies the in-process schema.
	Version int `json:"version"`
	// Sequence increases for every published App state.
	Sequence uint64 `json:"sequence"`
	// Final reports that no later App state will be accepted.
	Final bool `json:"final"`
	// Scenario is the immutable application scenario ID.
	Scenario string `json:"scenario,omitempty"`
	// Frame and semantic fields belong to the same atomic publication.
	Frame    IntendedFrame     `json:"frame"`
	Cursor   CursorState       `json:"cursor"`
	Controls []ControlSnapshot `json:"controls"`
	// Layouts is a required array in deterministic Layout-ID order.
	Layouts      []LayoutSnapshot      `json:"layouts"`
	InputSources []InputSourceSnapshot `json:"input_sources,omitempty"`
	// Overflows is present as an empty slice until Basic Layouts publishes one.
	Overflows []OverflowSnapshot `json:"overflows"`
	// Completion is present when this publication correlates an input request.
	Completion *Completion `json:"completion,omitempty"`
}

// SnapshotV1 is retained as a source-compatibility alias during the pre-v1
// transition. New code should use Snapshot.
type SnapshotV1 = Snapshot

func cloneSnapshot(snapshot Snapshot) Snapshot {
	cloned := snapshot
	cloned.Frame.Cells = append([]Cell(nil), snapshot.Frame.Cells...)
	cloned.Controls = append([]ControlSnapshot(nil), snapshot.Controls...)
	for index := range cloned.Controls {
		cloned.Controls[index].Children = append(
			[]ControlID(nil),
			snapshot.Controls[index].Children...,
		)
		if snapshot.Controls[index].Details.Container != nil {
			container := *snapshot.Controls[index].Details.Container
			cloned.Controls[index].Details.Container = &container
		}
		if snapshot.Controls[index].Details.Border != nil {
			border := *snapshot.Controls[index].Details.Border
			border.ForegroundOverride = cloneColor(border.ForegroundOverride)
			border.BackgroundOverride = cloneColor(border.BackgroundOverride)
			cloned.Controls[index].Details.Border = &border
		}
		if snapshot.Controls[index].Details.Text != nil {
			text := *snapshot.Controls[index].Details.Text
			cloned.Controls[index].Details.Text = &text
		}
		if snapshot.Controls[index].Details.Divider != nil {
			divider := *snapshot.Controls[index].Details.Divider
			cloned.Controls[index].Details.Divider = &divider
		}
		if snapshot.Controls[index].Details.Action != nil {
			action := *snapshot.Controls[index].Details.Action
			cloned.Controls[index].Details.Action = &action
		}
		if snapshot.Controls[index].Details.HotkeyBar != nil {
			hotkeyBar := *snapshot.Controls[index].Details.HotkeyBar
			hotkeyBar.Items = append(
				[]HotkeyBarItemDetails(nil),
				snapshot.Controls[index].Details.HotkeyBar.Items...,
			)
			for itemIndex := range hotkeyBar.Items {
				if hotkeyBar.Items[itemIndex].Chord != nil {
					chord := *hotkeyBar.Items[itemIndex].Chord
					chord.Modifiers = append(
						[]Key(nil),
						hotkeyBar.Items[itemIndex].Chord.Modifiers...,
					)
					hotkeyBar.Items[itemIndex].Chord = &chord
				}
			}
			cloned.Controls[index].Details.HotkeyBar = &hotkeyBar
		}
		if snapshot.Controls[index].Details.MenuBar != nil {
			menuBar := *snapshot.Controls[index].Details.MenuBar
			menuBar.Entries = append(
				[]MenuEntryDetails(nil),
				snapshot.Controls[index].Details.MenuBar.Entries...,
			)
			for entryIndex := range menuBar.Entries {
				if menuBar.Entries[entryIndex].Chord != nil {
					chord := *menuBar.Entries[entryIndex].Chord
					chord.Modifiers = append(
						[]Key(nil),
						menuBar.Entries[entryIndex].Chord.Modifiers...,
					)
					menuBar.Entries[entryIndex].Chord = &chord
				}
			}
			menuBar.OpenPath = append(
				[]string(nil),
				snapshot.Controls[index].Details.MenuBar.OpenPath...,
			)
			menuBar.SelectedPath = append(
				[]string(nil),
				snapshot.Controls[index].Details.MenuBar.SelectedPath...,
			)
			cloned.Controls[index].Details.MenuBar = &menuBar
		}
		if snapshot.Controls[index].Details.StatusBar != nil {
			statusBar := *snapshot.Controls[index].Details.StatusBar
			statusBar.Segments = append(
				[]StatusSegmentDetails(nil),
				snapshot.Controls[index].Details.StatusBar.Segments...,
			)
			for segmentIndex := range statusBar.Segments {
				if statusBar.Segments[segmentIndex].Chord != nil {
					chord := *statusBar.Segments[segmentIndex].Chord
					chord.Modifiers = append(
						[]Key(nil),
						statusBar.Segments[segmentIndex].Chord.Modifiers...,
					)
					statusBar.Segments[segmentIndex].Chord = &chord
				}
			}
			cloned.Controls[index].Details.StatusBar = &statusBar
		}
		if snapshot.Controls[index].Details.Checkbox != nil {
			checkbox := *snapshot.Controls[index].Details.Checkbox
			cloned.Controls[index].Details.Checkbox = &checkbox
		}
		if snapshot.Controls[index].Details.RadioButton != nil {
			button := *snapshot.Controls[index].Details.RadioButton
			cloned.Controls[index].Details.RadioButton = &button
		}
		if snapshot.Controls[index].Details.RadioGroup != nil {
			group := *snapshot.Controls[index].Details.RadioGroup
			group.Options = append(
				[]RadioOptionDetails(nil),
				snapshot.Controls[index].Details.RadioGroup.Options...,
			)
			cloned.Controls[index].Details.RadioGroup = &group
		}
		if snapshot.Controls[index].Details.ChoiceField != nil {
			field := *snapshot.Controls[index].Details.ChoiceField
			field.Options = append(
				[]SelectionOptionDetails(nil),
				snapshot.Controls[index].Details.ChoiceField.Options...,
			)
			cloned.Controls[index].Details.ChoiceField = &field
		}
		if snapshot.Controls[index].Details.FocusGuideBar != nil {
			guide := *snapshot.Controls[index].Details.FocusGuideBar
			cloned.Controls[index].Details.FocusGuideBar = &guide
		}
		if snapshot.Controls[index].Details.TextField != nil {
			field := *snapshot.Controls[index].Details.TextField
			if field.Validator != nil {
				validator := *field.Validator
				field.Validator = &validator
			}
			cloned.Controls[index].Details.TextField = &field
		}
		if snapshot.Controls[index].Details.NumberField != nil {
			field := *snapshot.Controls[index].Details.NumberField
			field.Minimum = cloneFloat64(field.Minimum)
			field.Maximum = cloneFloat64(field.Maximum)
			cloned.Controls[index].Details.NumberField = &field
		}
		if snapshot.Controls[index].Details.TextArea != nil {
			area := *snapshot.Controls[index].Details.TextArea
			if area.Validator != nil {
				validator := *area.Validator
				area.Validator = &validator
			}
			cloned.Controls[index].Details.TextArea = &area
		}
		if snapshot.Controls[index].Details.Progress != nil {
			progress := *snapshot.Controls[index].Details.Progress
			cloned.Controls[index].Details.Progress = &progress
		}
		if snapshot.Controls[index].Details.ScrollBar != nil {
			scrollBar := *snapshot.Controls[index].Details.ScrollBar
			cloned.Controls[index].Details.ScrollBar = &scrollBar
		}
		if snapshot.Controls[index].Details.TabbedPanel != nil {
			tabbedPanel := *snapshot.Controls[index].Details.TabbedPanel
			tabbedPanel.Tabs = append(
				[]TabDetails(nil),
				snapshot.Controls[index].Details.TabbedPanel.Tabs...,
			)
			cloned.Controls[index].Details.TabbedPanel = &tabbedPanel
		}
		if snapshot.Controls[index].Details.Scrollable != nil {
			scrollable := *snapshot.Controls[index].Details.Scrollable
			if scrollable.HorizontalBar != nil {
				bar := *scrollable.HorizontalBar
				scrollable.HorizontalBar = &bar
			}
			if scrollable.VerticalBar != nil {
				bar := *scrollable.VerticalBar
				scrollable.VerticalBar = &bar
			}
			cloned.Controls[index].Details.Scrollable = &scrollable
		}
		if snapshot.Controls[index].Details.Markdown != nil {
			markdown := *snapshot.Controls[index].Details.Markdown
			markdown.Blocks = append(
				[]MarkdownBlockDetails(nil),
				markdown.Blocks...,
			)
			if markdown.Viewport.HorizontalBar != nil {
				bar := *markdown.Viewport.HorizontalBar
				markdown.Viewport.HorizontalBar = &bar
			}
			if markdown.Viewport.VerticalBar != nil {
				bar := *markdown.Viewport.VerticalBar
				markdown.Viewport.VerticalBar = &bar
			}
			cloned.Controls[index].Details.Markdown = &markdown
		}
		if snapshot.Controls[index].Details.LogView != nil {
			logView := *snapshot.Controls[index].Details.LogView
			if logView.Viewport.HorizontalBar != nil {
				bar := *logView.Viewport.HorizontalBar
				logView.Viewport.HorizontalBar = &bar
			}
			if logView.Viewport.VerticalBar != nil {
				bar := *logView.Viewport.VerticalBar
				logView.Viewport.VerticalBar = &bar
			}
			cloned.Controls[index].Details.LogView = &logView
		}
		if snapshot.Controls[index].Details.StreamView != nil {
			streamView := *snapshot.Controls[index].Details.StreamView
			if streamView.Viewport.HorizontalBar != nil {
				bar := *streamView.Viewport.HorizontalBar
				streamView.Viewport.HorizontalBar = &bar
			}
			if streamView.Viewport.VerticalBar != nil {
				bar := *streamView.Viewport.VerticalBar
				streamView.Viewport.VerticalBar = &bar
			}
			cloned.Controls[index].Details.StreamView = &streamView
		}
		if snapshot.Controls[index].Details.ListBox != nil {
			listBox := *snapshot.Controls[index].Details.ListBox
			if listBox.Viewport.HorizontalBar != nil {
				bar := *listBox.Viewport.HorizontalBar
				listBox.Viewport.HorizontalBar = &bar
			}
			if listBox.Viewport.VerticalBar != nil {
				bar := *listBox.Viewport.VerticalBar
				listBox.Viewport.VerticalBar = &bar
			}
			cloned.Controls[index].Details.ListBox = &listBox
		}
		if snapshot.Controls[index].Details.DropDown != nil {
			dropDown := *snapshot.Controls[index].Details.DropDown
			cloned.Controls[index].Details.DropDown = &dropDown
		}
		if snapshot.Controls[index].Details.ComboBox != nil {
			comboBox := *snapshot.Controls[index].Details.ComboBox
			if comboBox.Editor.Validator != nil {
				validator := *comboBox.Editor.Validator
				comboBox.Editor.Validator = &validator
			}
			cloned.Controls[index].Details.ComboBox = &comboBox
		}
	}
	cloned.Layouts = append([]LayoutSnapshot{}, snapshot.Layouts...)
	for index := range cloned.Layouts {
		cloned.Layouts[index].Items = append(
			[]LayoutItemSnapshot(nil),
			snapshot.Layouts[index].Items...,
		)
		if snapshot.Layouts[index].Border != nil {
			border := *snapshot.Layouts[index].Border
			border.ForegroundOverride = cloneColor(border.ForegroundOverride)
			border.BackgroundOverride = cloneColor(border.BackgroundOverride)
			cloned.Layouts[index].Border = &border
		}
	}
	cloned.InputSources = append([]InputSourceSnapshot(nil), snapshot.InputSources...)
	for index := range cloned.InputSources {
		cloned.InputSources[index].Held = append([]Key(nil), snapshot.InputSources[index].Held...)
	}
	// Overflows is a required JSON array. Preserve an empty, non-nil slice so
	// cloned snapshots encode it as [] rather than null.
	cloned.Overflows = append([]OverflowSnapshot{}, snapshot.Overflows...)
	if snapshot.Completion != nil {
		completion := *snapshot.Completion
		cloned.Completion = &completion
	}
	return cloned
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
