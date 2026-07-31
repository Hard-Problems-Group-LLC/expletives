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
	// MaxHotkeyBarItems bounds one HotkeyBar's copied ordered inventory.
	MaxHotkeyBarItems = 64
	// MaxActionItems bounds aggregate HotkeyBar items across one App.
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
	// ControlMenuBar identifies the persistent popup-menu session owner.
	ControlMenuBar ControlKind = "menu_bar"
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
