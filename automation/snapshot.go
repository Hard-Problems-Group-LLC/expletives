package automation

import (
	"crypto/sha256"
	"fmt"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
)

// Snapshot DTO scalar types are automation-owned so the wire contract remains
// independent of the toolkit's in-process representation.
type (
	// StyleID is a bounded semantic style identity.
	StyleID string
	// StyleAttributes carries the version 1 attribute bits: bold, dim, italic,
	// underline, and reverse.
	StyleAttributes uint16
	// Color is a resolved #RRGGBB color string.
	Color string
	// ControlID is an App-scoped runtime control identity.
	ControlID string
	// ControlKind identifies one bounded control behavior.
	ControlKind string
	// InputScopeMode identifies one structural keyboard-input boundary.
	InputScopeMode string
	// LayoutID is an App-scoped runtime Layout identity.
	LayoutID string
	// LayoutKind identifies one bounded arrangement algorithm.
	LayoutKind string
	// Key is a logical key identity independent of terminal bytes.
	Key string
	// TextAlignment selects start, center, or end placement.
	TextAlignment string
	// TextWrap selects no wrapping, word wrapping, or cell wrapping.
	TextWrap string
	// Orientation selects a horizontal or vertical divider axis.
	Orientation uint8
	// MenuItemKind identifies a command, separator, or submenu entry.
	MenuItemKind string
	// MenuBarPlacement identifies a top-level start or end edge group.
	MenuBarPlacement string
)

// Point is a zero-based snapshot cell coordinate.
type Point struct {
	// X is the column.
	X int `json:"x"`
	// Y is the row.
	Y int `json:"y"`
}

// Size is a logical width and height in cells.
type Size struct {
	// Width is the number of columns.
	Width int `json:"width"`
	// Height is the number of rows.
	Height int `json:"height"`
}

// Rect is logical snapshot geometry.
type Rect struct {
	// X and Y locate the top-left cell.
	X int `json:"x"`
	Y int `json:"y"`
	// Width and Height are nonnegative cell counts.
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ResolvedStyle is the terminal-independent visual value of a StyleID.
type ResolvedStyle struct {
	// Foreground is the intended text color.
	Foreground Color `json:"foreground"`
	// Background is the intended cell background.
	Background Color `json:"background"`
	// Attributes contains only version 1 attribute bits.
	Attributes StyleAttributes `json:"attributes,omitempty"`
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
	// Owner is the runtime control that most recently painted this cell.
	Owner ControlID `json:"owner"`
}

// CellRun is one row-major run of identical canonical cells on the wire.
type CellRun struct {
	Count int  `json:"count"`
	Cell  Cell `json:"cell"`
}

// IntendedFrame is a pure terminal-independent character-cell surface.
type IntendedFrame struct {
	// Size is the frame geometry.
	Size Size `json:"size"`
	// Runs is the compact bounded wire representation.
	Runs []CellRun `json:"runs,omitempty"`
	// Cells is the expanded caller-facing row-major representation. It is
	// populated after decoding. Server completions clear it before wire
	// encoding; expletivesctl output retains it for direct frame inspection.
	Cells []Cell `json:"cells,omitempty"`
}

// CursorState describes the intended logical cursor.
type CursorState struct {
	// Visible controls whether a presenter should show the cursor.
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
	// Kind identifies the control behavior.
	Kind ControlKind `json:"kind"`
	// Parent is empty only for the App root.
	Parent ControlID `json:"parent,omitempty"`
	// Children preserves direct-child insertion order.
	Children []ControlID `json:"children,omitempty"`
	// Bounds is parent-client-relative logical geometry.
	Bounds Rect `json:"bounds"`
	// AbsoluteBounds is unclipped App-relative geometry.
	AbsoluteBounds Rect `json:"absolute_bounds"`
	// EffectiveClip intersects visibility, ancestors, and the App surface.
	EffectiveClip Rect `json:"effective_clip"`
	// Minimum is the declared logical minimum.
	Minimum Size `json:"minimum"`
	// Layout identifies the manager of Bounds, when any.
	Layout      LayoutID `json:"layout,omitempty"`
	LayoutIndex int      `json:"layout_index"`
	StackIndex  int      `json:"stack_index"`
	// Style and ResolvedStyle pair semantic and visual state.
	Style         StyleID       `json:"style"`
	ResolvedStyle ResolvedStyle `json:"resolved_style"`
	// Visible reports effective visibility through the ancestor chain.
	Visible bool `json:"visible"`
	// Focused reports that this control owns keyboard focus.
	Focused bool `json:"focused"`
	// Details is versioned control-specific state.
	Details ControlDetails `json:"details"`
}

// ControlDetails is the versioned typed union for control-specific state.
type ControlDetails struct {
	// Version identifies the union schema.
	Version int `json:"version"`
	// Container is present for controls that can parent children.
	Container *ContainerDetails `json:"container,omitempty"`
	// Border is present for bordered controls.
	Border *BorderDetails `json:"border,omitempty"`
	// Text is present for Label and StaticText controls.
	Text *TextDetails `json:"text,omitempty"`
	// Divider is present for Separator and Rule controls.
	Divider *DividerDetails `json:"divider,omitempty"`
	// Action is present for Button and later activation controls.
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
	// TreeView is present for TreeView.
	TreeView *TreeViewDetails `json:"tree_view,omitempty"`
	// Table is present for Table.
	Table *TableDetails `json:"table,omitempty"`
	// DataGrid is present for DataGrid.
	DataGrid *DataGridDetails `json:"data_grid,omitempty"`
	// DropDown is present for DropDown.
	DropDown *DropDownDetails `json:"drop_down,omitempty"`
	// ComboBox is present for ComboBox.
	ComboBox *ComboBoxDetails `json:"combo_box,omitempty"`
	// ModalPanel is present for ModalPanel and Dialog.
	ModalPanel *ModalPanelDetails `json:"modal_panel,omitempty"`
	// ProgressDialog is present only for ProgressDialog.
	ProgressDialog *ProgressDialogDetails `json:"progress_dialog,omitempty"`
	// FilePicker is present for each file-picker Dialog specialization.
	FilePicker *FilePickerDetails `json:"file_picker,omitempty"`
}

// ContainerDetails describes a container's client area.
type ContainerDetails struct {
	// ClientInset is the number of cells reserved on every edge.
	ClientInset int `json:"client_inset"`
	// InputScope identifies an explicit or toolkit-owned input boundary.
	InputScope InputScopeMode `json:"input_scope,omitempty"`
}

// ModalResultDetails is the bounded terminal result of one modal lifecycle.
type ModalResultDetails struct {
	Reason string `json:"reason"`
	Action string `json:"action,omitempty"`
}

// ModalPanelDetails is the compact modal lifecycle, stack, and geometry
// observation. It contains no retained child-control values.
type ModalPanelDetails struct {
	Lifecycle       string              `json:"lifecycle"`
	Active          bool                `json:"active"`
	Top             bool                `json:"top"`
	StackIndex      int                 `json:"stack_index"`
	StackDepth      int                 `json:"stack_depth"`
	NestedOwner     ControlID           `json:"nested_owner,omitempty"`
	SavedFocus      ControlID           `json:"saved_focus,omitempty"`
	InitialFocus    ControlID           `json:"initial_focus,omitempty"`
	RequestedSize   Size                `json:"requested_size"`
	ResolvedBounds  Rect                `json:"resolved_bounds"`
	RequiredMinimum Size                `json:"required_minimum"`
	Degraded        bool                `json:"degraded"`
	Shadow          string              `json:"shadow"`
	ShadowStyle     StyleID             `json:"shadow_style,omitempty"`
	Result          *ModalResultDetails `json:"result,omitempty"`
}

// ProgressDialogProgressDetails is the compound-level copied ProgressBar
// state without renderer-derived frame geometry.
type ProgressDialogProgressDetails struct {
	Status        string `json:"status"`
	Current       uint64 `json:"current,omitempty"`
	Total         uint64 `json:"total,omitempty"`
	Indeterminate bool   `json:"indeterminate"`
	Tick          uint64 `json:"tick,omitempty"`
	ReducedMotion bool   `json:"reduced_motion"`
	TextMode      string `json:"text_mode"`
}

// ProgressDialogDetails describes status and the one-shot cancellation
// handshake. Exact status text remains on the ordinary StaticText child.
type ProgressDialogDetails struct {
	StatusLength    int                           `json:"status_length"`
	Progress        ProgressDialogProgressDetails `json:"progress"`
	Cancellable     bool                          `json:"cancellable"`
	CancelRequested bool                          `json:"cancel_requested"`
}

// FilePickerDetails is privacy-compacted picker state. Display strings are
// represented by byte counts and SHA-256 evidence; provider tokens and
// accepted results are never projected.
type FilePickerDetails struct {
	Mode              string `json:"mode"`
	Status            string `json:"status"`
	DisplayPathBytes  int    `json:"display_path_bytes"`
	DisplayPathDigest string `json:"display_path_digest,omitempty"`
	EntryCount        int    `json:"entry_count"`
	FileCount         int    `json:"file_count"`
	DirectoryCount    int    `json:"directory_count"`
	CurrentNameBytes  int    `json:"current_name_bytes"`
	CurrentNameDigest string `json:"current_name_digest,omitempty"`
	CurrentKind       string `json:"current_kind,omitempty"`
	SelectedCount     int    `json:"selected_count"`
	Filter            string `json:"filter"`
	SortField         string `json:"sort_field"`
	SortDirection     string `json:"sort_direction"`
	ErrorBytes        int    `json:"error_bytes"`
	ErrorDigest       string `json:"error_digest,omitempty"`
}

// BorderDetails describes one bordered container.
type BorderDetails struct {
	// Title is canonical bounded display text.
	Title string `json:"title,omitempty"`
	// Form identifies the canonical border geometry.
	Form string `json:"form"`
	// Style and ResolvedStyle pair the border's semantic and visual state.
	Style              StyleID       `json:"style"`
	ResolvedStyle      ResolvedStyle `json:"resolved_style"`
	ForegroundOverride *Color        `json:"foreground_override,omitempty"`
	BackgroundOverride *Color        `json:"background_override,omitempty"`
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
	Form        string        `json:"form"`
	Text        string        `json:"text,omitempty"`
	Alignment   TextAlignment `json:"alignment"`
}

// ActionDetails describes one command-backed activation control.
type ActionDetails struct {
	Label          string `json:"label"`
	Command        string `json:"command"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Checked        bool   `json:"checked"`
	Mnemonic       Key    `json:"mnemonic,omitempty"`
	Pressed        bool   `json:"pressed"`
	Default        bool   `json:"default"`
	Cancel         bool   `json:"cancel"`
}

// Chord describes one non-modifier key and its canonical modifiers.
type Chord struct {
	Key       Key   `json:"key"`
	Modifiers []Key `json:"modifiers"`
}

// HotkeyBarItemDetails describes one current command shortcut.
type HotkeyBarItemDetails struct {
	Label          string `json:"label"`
	Command        string `json:"command"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Checked        bool   `json:"checked"`
	Chord          *Chord `json:"chord,omitempty"`
}

// HotkeyBarDetails describes a bounded ordered shortcut inventory.
type HotkeyBarDetails struct {
	Items []HotkeyBarItemDetails `json:"items"`
}

// MenuEntryDetails is one flattened immutable menu entry.
type MenuEntryDetails struct {
	Key            string           `json:"key"`
	ParentKey      string           `json:"parent_key,omitempty"`
	Depth          int              `json:"depth"`
	Kind           MenuItemKind     `json:"kind"`
	Label          string           `json:"label,omitempty"`
	Command        string           `json:"command,omitempty"`
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

// MenuBarDetails is one bounded flattened menu tree and its session paths.
type MenuBarDetails struct {
	Entries      []MenuEntryDetails `json:"entries"`
	OpenPath     []string           `json:"open_path"`
	SelectedPath []string           `json:"selected_path"`
}

// StatusSegmentDetails describes one current contextual or command segment.
type StatusSegmentDetails struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Command        string `json:"command,omitempty"`
	Priority       int    `json:"priority"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Checked        bool   `json:"checked"`
	Chord          *Chord `json:"chord,omitempty"`
	Rendered       bool   `json:"rendered"`
	Bounds         Rect   `json:"bounds"`
	Clipped        bool   `json:"clipped"`
}

// StatusBarDetails describes one bounded ordered bottom-row inventory.
type StatusBarDetails struct {
	Segments []StatusSegmentDetails `json:"segments"`
}

// CheckboxDetails describes one current Checkbox value and policy.
type CheckboxDetails struct {
	Label          string `json:"label"`
	State          string `json:"state"`
	ThreeState     bool   `json:"three_state"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Mnemonic       Key    `json:"mnemonic,omitempty"`
	ChangeCommand  string `json:"change_command,omitempty"`
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

// RadioGroupDetails describes one exclusive selection.
type RadioGroupDetails struct {
	Value          string               `json:"value,omitempty"`
	AllowEmpty     bool                 `json:"allow_empty"`
	Enabled        bool                 `json:"enabled"`
	DisabledReason string               `json:"disabled_reason,omitempty"`
	ChangeCommand  string               `json:"change_command,omitempty"`
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

// ChoiceFieldDetails describes one CycleField or SelectField.
type ChoiceFieldDetails struct {
	Label          string                   `json:"label"`
	Value          string                   `json:"value,omitempty"`
	SelectedIndex  int                      `json:"selected_index"`
	Enabled        bool                     `json:"enabled"`
	DisabledReason string                   `json:"disabled_reason,omitempty"`
	Mnemonic       Key                      `json:"mnemonic,omitempty"`
	ChangeCommand  string                   `json:"change_command,omitempty"`
	Options        []SelectionOptionDetails `json:"options"`
}

// FocusGuideBarDetails describes guidance resolved for current focus.
type FocusGuideBarDetails struct {
	Target        ControlID   `json:"target,omitempty"`
	TargetKind    ControlKind `json:"target_kind,omitempty"`
	Text          string      `json:"text"`
	Customization string      `json:"customization,omitempty"`
}

// TextValidatorDetails describes one copied character-set policy.
type TextValidatorDetails struct {
	Enforcement        string `json:"enforcement"`
	Mode               string `json:"mode"`
	Characters         string `json:"characters"`
	CharactersRedacted bool   `json:"characters_redacted,omitempty"`
}

// TextFieldByteStyleDetails describes one UTF-8 byte threshold and semantic
// style. It exposes policy, never the currently active password threshold.
type TextFieldByteStyleDetails struct {
	MinimumBytes int    `json:"minimum_bytes"`
	Style        string `json:"style"`
}

// TextFieldDetails describes current single-line editing and validation state.
type TextFieldDetails struct {
	Text           string                      `json:"text,omitempty"`
	Length         int                         `json:"length"`
	MaximumBytes   int                         `json:"maximum_bytes"`
	Caret          int                         `json:"caret"`
	SelectionStart int                         `json:"selection_start"`
	SelectionEnd   int                         `json:"selection_end"`
	ViewOffset     int                         `json:"view_offset"`
	Editing        bool                        `json:"editing"`
	Valid          bool                        `json:"valid"`
	Password       bool                        `json:"password"`
	Redacted       bool                        `json:"redacted"`
	Enabled        bool                        `json:"enabled"`
	DisabledReason string                      `json:"disabled_reason,omitempty"`
	ChangeCommand  string                      `json:"change_command,omitempty"`
	EditCommand    string                      `json:"edit_command,omitempty"`
	SubmitCommand  string                      `json:"submit_command,omitempty"`
	FocusedStyle   string                      `json:"focused_style"`
	EditingStyle   string                      `json:"editing_style"`
	ByteStyles     []TextFieldByteStyleDetails `json:"byte_styles"`
	Validator      *TextValidatorDetails       `json:"validator,omitempty"`
}

// NumberFieldDetails describes current decimal editing, range, and step state.
type NumberFieldDetails struct {
	Text           string   `json:"text"`
	Value          float64  `json:"value"`
	Length         int      `json:"length"`
	Caret          int      `json:"caret"`
	SelectionStart int      `json:"selection_start"`
	SelectionEnd   int      `json:"selection_end"`
	ViewOffset     int      `json:"view_offset"`
	Editing        bool     `json:"editing"`
	Valid          bool     `json:"valid"`
	InvalidReason  string   `json:"invalid_reason,omitempty"`
	Minimum        *float64 `json:"minimum,omitempty"`
	Maximum        *float64 `json:"maximum,omitempty"`
	DecimalPlaces  int      `json:"decimal_places"`
	Step           float64  `json:"step,omitempty"`
	Enabled        bool     `json:"enabled"`
	DisabledReason string   `json:"disabled_reason,omitempty"`
	ChangeCommand  string   `json:"change_command,omitempty"`
}

// TextAreaDetails describes current multiline editing and viewport state.
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
	ChangeCommand     string                `json:"change_command,omitempty"`
	Validator         *TextValidatorDetails `json:"validator,omitempty"`
}

// ProgressDetails describes one kind-consistent progress or activity state.
type ProgressDetails struct {
	Status        string      `json:"status"`
	Current       uint64      `json:"current,omitempty"`
	Total         uint64      `json:"total,omitempty"`
	Value         float64     `json:"value,omitempty"`
	Minimum       float64     `json:"minimum,omitempty"`
	Maximum       float64     `json:"maximum,omitempty"`
	Orientation   Orientation `json:"orientation"`
	Indeterminate bool        `json:"indeterminate"`
	Tick          uint64      `json:"tick,omitempty"`
	ReducedMotion bool        `json:"reduced_motion"`
	TextMode      string      `json:"text_mode"`
	FrameIndex    int         `json:"frame_index"`
}

// ScrollBarDetails describes one canonical viewport and rendered thumb.
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
	ChangeCommand  string      `json:"change_command,omitempty"`
}

// ViewportState is one complete logical content extent and offset.
type ViewportState struct {
	ContentSize Size  `json:"content_size"`
	Offset      Point `json:"offset"`
}

// ScrollableDetails describes one generic two-axis content viewport.
type ScrollableDetails struct {
	State             ViewportState     `json:"state"`
	MaximumOffset     Point             `json:"maximum_offset"`
	ViewportBounds    Rect              `json:"viewport_bounds"`
	ArrowStep         Size              `json:"arrow_step"`
	PageStep          Size              `json:"page_step"`
	Enabled           bool              `json:"enabled"`
	DisabledReason    string            `json:"disabled_reason,omitempty"`
	ChangeCommand     string            `json:"change_command,omitempty"`
	Content           ControlID         `json:"content"`
	ContentKey        string            `json:"content_key,omitempty"`
	HorizontalPolicy  string            `json:"horizontal_policy"`
	VerticalPolicy    string            `json:"vertical_policy"`
	HorizontalVisible bool              `json:"horizontal_visible"`
	VerticalVisible   bool              `json:"vertical_visible"`
	HorizontalBar     *ScrollBarDetails `json:"horizontal_bar,omitempty"`
	VerticalBar       *ScrollBarDetails `json:"vertical_bar,omitempty"`
}

// MarkdownBlockDetails is one bounded structural block summary.
type MarkdownBlockDetails struct {
	Kind          string `json:"kind"`
	Level         int    `json:"level,omitempty"`
	SourceLine    int    `json:"source_line"`
	SourceLines   int    `json:"source_lines"`
	RenderedStart int    `json:"rendered_start"`
	RenderedRows  int    `json:"rendered_rows"`
}

// MarkdownDetails describes bounded structure and derived viewport geometry
// without duplicating the retained Markdown source.
type MarkdownDetails struct {
	SourceBytes        int                    `json:"source_bytes"`
	SourceCells        int                    `json:"source_cells"`
	BlockCount         int                    `json:"block_count"`
	RenderedRows       int                    `json:"rendered_rows"`
	MaximumLineWidth   int                    `json:"maximum_line_width"`
	Viewport           ContentViewportDetails `json:"viewport"`
	Blocks             []MarkdownBlockDetails `json:"blocks"`
	SummariesTruncated bool                   `json:"summaries_truncated"`
}

// ContentViewportDetails is bounded scroll geometry for a content leaf. It
// omits generic managed-content and complete integrated-bar records that do
// not exist as independently addressable controls.
type ContentViewportDetails struct {
	State             ViewportState `json:"state"`
	MaximumOffset     Point         `json:"maximum_offset"`
	ViewportBounds    Rect          `json:"viewport_bounds"`
	HorizontalPolicy  string        `json:"horizontal_policy"`
	VerticalPolicy    string        `json:"vertical_policy"`
	HorizontalVisible bool          `json:"horizontal_visible"`
	VerticalVisible   bool          `json:"vertical_visible"`
}

// ContentCapacity is the copied record and canonical-byte retention budget.
type ContentCapacity struct {
	Records int `json:"records"`
	Bytes   int `json:"bytes"`
}

// MarkdownViewportDetails is the compatibility name for the shared bounded
// content-leaf viewport projection.
type MarkdownViewportDetails = ContentViewportDetails

// LogViewDetails describes bounded structured-log retention and viewport
// state without duplicating retained record text.
type LogViewDetails struct {
	Capacity        ContentCapacity        `json:"capacity"`
	RetainedRecords int                    `json:"retained_records"`
	RetainedBytes   int                    `json:"retained_bytes"`
	DroppedRecords  uint64                 `json:"dropped_records"`
	DroppedBytes    uint64                 `json:"dropped_bytes"`
	FirstKey        string                 `json:"first_key,omitempty"`
	LastKey         string                 `json:"last_key,omitempty"`
	Follow          bool                   `json:"follow"`
	Viewport        ContentViewportDetails `json:"viewport"`
}

// StreamViewDetails describes bounded complete and partial stream retention
// without duplicating off-screen stream content.
type StreamViewDetails struct {
	Capacity            ContentCapacity        `json:"capacity"`
	RetainedLines       int                    `json:"retained_lines"`
	RetainedBytes       int                    `json:"retained_bytes"`
	PendingBytes        int                    `json:"pending_bytes"`
	PendingCells        int                    `json:"pending_cells"`
	PendingStorageBytes int                    `json:"pending_storage_bytes"`
	PendingTruncated    bool                   `json:"pending_truncated"`
	DroppedLines        uint64                 `json:"dropped_lines"`
	DroppedBytes        uint64                 `json:"dropped_bytes"`
	Follow              bool                   `json:"follow"`
	Viewport            ContentViewportDetails `json:"viewport"`
}

// ListBoxDetails is a compact stable-identity list observation that omits
// the retained item model.
type ListBoxDetails struct {
	Status              string                 `json:"status"`
	StatusMessageBytes  int                    `json:"status_message_bytes"`
	StatusMessageDigest string                 `json:"status_message_digest,omitempty"`
	ItemCount           int                    `json:"item_count"`
	EnabledCount        int                    `json:"enabled_count"`
	VisualRowCount      int                    `json:"visual_row_count"`
	Wrap                TextWrap               `json:"wrap"`
	RetainedBytes       int                    `json:"retained_bytes"`
	Current             string                 `json:"current,omitempty"`
	CurrentIndex        int                    `json:"current_index"`
	SelectionMode       string                 `json:"selection_mode"`
	SelectionMarks      bool                   `json:"selection_marks"`
	RequireSelection    bool                   `json:"require_selection"`
	SelectedCount       int                    `json:"selected_count"`
	FirstSelected       string                 `json:"first_selected,omitempty"`
	LastSelected        string                 `json:"last_selected,omitempty"`
	SelectionDigest     string                 `json:"selection_digest"`
	Enabled             bool                   `json:"enabled"`
	DisabledReasonBytes int                    `json:"disabled_reason_bytes"`
	ChangeCommand       string                 `json:"change_command,omitempty"`
	CurrentCommand      string                 `json:"current_command,omitempty"`
	ActivateCommand     string                 `json:"activate_command,omitempty"`
	Viewport            ContentViewportDetails `json:"viewport"`
}

// TreeViewDetails is a compact stable-identity hierarchy observation that
// omits the retained recursive node model and exact status/reason text.
type TreeViewDetails struct {
	Status              string                 `json:"status"`
	StatusMessageBytes  int                    `json:"status_message_bytes"`
	StatusMessageDigest string                 `json:"status_message_digest,omitempty"`
	NodeCount           int                    `json:"node_count"`
	VisibleCount        int                    `json:"visible_count"`
	EnabledCount        int                    `json:"enabled_count"`
	RetainedBytes       int                    `json:"retained_bytes"`
	Current             string                 `json:"current,omitempty"`
	CurrentIndex        int                    `json:"current_index"`
	SelectionMode       string                 `json:"selection_mode"`
	RequireSelection    bool                   `json:"require_selection"`
	SelectedCount       int                    `json:"selected_count"`
	FirstSelected       string                 `json:"first_selected,omitempty"`
	LastSelected        string                 `json:"last_selected,omitempty"`
	SelectionDigest     string                 `json:"selection_digest"`
	ExpandedCount       int                    `json:"expanded_count"`
	FirstExpanded       string                 `json:"first_expanded,omitempty"`
	LastExpanded        string                 `json:"last_expanded,omitempty"`
	ExpansionDigest     string                 `json:"expansion_digest"`
	Enabled             bool                   `json:"enabled"`
	DisabledReasonBytes int                    `json:"disabled_reason_bytes"`
	ChangeCommand       string                 `json:"change_command,omitempty"`
	ActivateCommand     string                 `json:"activate_command,omitempty"`
	ExpandCommand       string                 `json:"expand_command,omitempty"`
	Viewport            ContentViewportDetails `json:"viewport"`
}

// TableDetails is a compact stable-identity tabular observation that omits
// the retained column and row models and exact status/reason text.
type TableDetails struct {
	Status              string                 `json:"status"`
	StatusMessageBytes  int                    `json:"status_message_bytes"`
	StatusMessageDigest string                 `json:"status_message_digest,omitempty"`
	RowCount            int                    `json:"row_count"`
	EnabledCount        int                    `json:"enabled_count"`
	ColumnCount         int                    `json:"column_count"`
	CellCount           int                    `json:"cell_count"`
	RetainedBytes       int                    `json:"retained_bytes"`
	CurrentRow          string                 `json:"current_row,omitempty"`
	CurrentRowIndex     int                    `json:"current_row_index"`
	CurrentColumn       string                 `json:"current_column,omitempty"`
	CurrentColumnIndex  int                    `json:"current_column_index"`
	FocusMode           string                 `json:"focus_mode"`
	SelectionMode       string                 `json:"selection_mode"`
	RequireSelection    bool                   `json:"require_selection"`
	SelectedCount       int                    `json:"selected_count"`
	FirstSelected       string                 `json:"first_selected,omitempty"`
	LastSelected        string                 `json:"last_selected,omitempty"`
	SelectionDigest     string                 `json:"selection_digest"`
	SortColumn          string                 `json:"sort_column,omitempty"`
	SortDirection       string                 `json:"sort_direction"`
	FirstColumn         string                 `json:"first_column,omitempty"`
	LastColumn          string                 `json:"last_column,omitempty"`
	ColumnWidthsDigest  string                 `json:"column_widths_digest"`
	Enabled             bool                   `json:"enabled"`
	DisabledReasonBytes int                    `json:"disabled_reason_bytes"`
	ChangeCommand       string                 `json:"change_command,omitempty"`
	ActivateCommand     string                 `json:"activate_command,omitempty"`
	SortCommand         string                 `json:"sort_command,omitempty"`
	Viewport            ContentViewportDetails `json:"viewport"`
}

// DataGridDetails is a compact editable-table observation. It omits retained
// model text, active editor text, and validator character sets.
type DataGridDetails struct {
	Table                 TableDetails `json:"table"`
	Editing               bool         `json:"editing"`
	EditRow               string       `json:"edit_row,omitempty"`
	EditColumn            string       `json:"edit_column,omitempty"`
	EditLength            int          `json:"edit_length"`
	EditCaret             int          `json:"edit_caret"`
	EditViewOffset        int          `json:"edit_view_offset"`
	EditValid             bool         `json:"edit_valid"`
	ValidationEnforcement string       `json:"validation_enforcement,omitempty"`
	ValidationMode        string       `json:"validation_mode,omitempty"`
}

// DropDownDetails is a compact stable-identity popup observation that omits
// the retained item model and exact disabled-reason text.
type DropDownDetails struct {
	ItemCount           int    `json:"item_count"`
	EnabledCount        int    `json:"enabled_count"`
	RetainedBytes       int    `json:"retained_bytes"`
	Current             string `json:"current,omitempty"`
	CurrentIndex        int    `json:"current_index"`
	Selected            string `json:"selected,omitempty"`
	SelectedIndex       int    `json:"selected_index"`
	AllowEmpty          bool   `json:"allow_empty"`
	PopupRows           int    `json:"popup_rows"`
	Open                bool   `json:"open"`
	PopupBounds         Rect   `json:"popup_bounds"`
	PopupOffset         int    `json:"popup_offset"`
	PopupCurrent        string `json:"popup_current,omitempty"`
	PopupSelection      string `json:"popup_selection,omitempty"`
	Enabled             bool   `json:"enabled"`
	DisabledReasonBytes int    `json:"disabled_reason_bytes"`
	ChangeCommand       string `json:"change_command,omitempty"`
	ActivateCommand     string `json:"activate_command,omitempty"`
}

// ComboBoxDetails combines compact popup state with its exact bounded editor
// observation.
type ComboBoxDetails struct {
	Popup  DropDownDetails  `json:"popup"`
	Editor TextFieldDetails `json:"editor"`
}

// TabDetails describes one copied page descriptor and rendered strip state.
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
	ChangeCommand   string       `json:"change_command,omitempty"`
	LeadingOmitted  bool         `json:"leading_omitted"`
	TrailingOmitted bool         `json:"trailing_omitted"`
}

// InputSourceSnapshot reports held keys for one isolated input source.
type InputSourceSnapshot struct {
	// Source is the bounded input-source identity.
	Source string `json:"source"`
	// Held is the sorted set of currently held logical keys.
	Held []Key `json:"held"`
}

// OverflowSnapshot describes one bounded Layout overflow observation.
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

// LayoutItemSnapshot describes one Panel or nested Layout item.
type LayoutItemSnapshot struct {
	Kind        string    `json:"kind"`
	Panel       ControlID `json:"panel,omitempty"`
	Layout      LayoutID  `json:"layout,omitempty"`
	Bounds      Rect      `json:"bounds"`
	Minimum     Size      `json:"minimum"`
	LayoutIndex int       `json:"layout_index"`
	StackIndex  int       `json:"stack_index"`
}

// LayoutSnapshot is one bounded nonvisual arrangement observation.
type LayoutSnapshot struct {
	ID          LayoutID             `json:"id"`
	Key         string               `json:"key,omitempty"`
	Kind        LayoutKind           `json:"kind"`
	Owner       ControlID            `json:"owner"`
	Parent      LayoutID             `json:"parent,omitempty"`
	Bounds      Rect                 `json:"bounds"`
	OwnerBounds Rect                 `json:"owner_bounds"`
	Minimum     Size                 `json:"minimum"`
	Border      *BorderDetails       `json:"border"`
	LayoutIndex int                  `json:"layout_index"`
	StackIndex  int                  `json:"stack_index"`
	Items       []LayoutItemSnapshot `json:"items"`
}

// SnapshotCompletion is the bounded public command association embedded in a
// snapshot. It deliberately has no local-only Cause field.
type SnapshotCompletion struct {
	// RequestID is the associated request correlation ID.
	RequestID string `json:"request_id"`
	// Outcome is the associated terminal result.
	Outcome string `json:"outcome"`
	// Command is the optional resolved semantic command.
	Command string `json:"command,omitempty"`
	// FrameSequence equals the containing snapshot sequence.
	FrameSequence uint64 `json:"frame_sequence"`
	// Code and Message are bounded public diagnostics.
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// SnapshotV1 is the automation-owned version 1 observation DTO. Client
// completions contain a newly decoded, caller-owned object graph; mutating it
// cannot affect server or App state.
type SnapshotV1 struct {
	// Version is the snapshot schema version and equals 1.
	Version int `json:"version"`
	// Sequence is the exact associated App frame sequence.
	Sequence uint64 `json:"sequence"`
	// Final reports that the observed App no longer accepts ordinary work.
	Final bool `json:"final"`
	// Scenario is the stable App scenario identity.
	Scenario string `json:"scenario,omitempty"`
	// Frame and semantic fields belong to the same atomic observation.
	Frame        IntendedFrame         `json:"frame"`
	Cursor       CursorState           `json:"cursor"`
	Controls     []ControlSnapshot     `json:"controls"`
	Layouts      []LayoutSnapshot      `json:"layouts"`
	InputSources []InputSourceSnapshot `json:"input_sources,omitempty"`
	// Overflows is a bounded required array.
	Overflows []OverflowSnapshot `json:"overflows"`
	// Completion is the optional sanitized App request association.
	Completion *SnapshotCompletion `json:"completion,omitempty"`
}

func snapshotFromCore(snapshot expletives.Snapshot) SnapshotV1 {
	controlsByID := make(map[expletives.ControlID]expletives.ControlSnapshot,
		len(snapshot.Controls))
	for _, control := range snapshot.Controls {
		controlsByID[control.ID] = control
	}
	sensitiveInput := make(map[expletives.ControlID]bool)
	for _, control := range snapshot.Controls {
		ancestor := control
		for depth := 0; depth <= len(snapshot.Controls); depth++ {
			if ancestor.Kind == expletives.ControlInputDialog {
				sensitiveInput[control.ID] = true
				break
			}
			if ancestor.Parent == "" {
				break
			}
			parent, ok := controlsByID[ancestor.Parent]
			if !ok {
				break
			}
			ancestor = parent
		}
	}
	projected := SnapshotV1{
		Version:  snapshot.Version,
		Sequence: snapshot.Sequence,
		Final:    snapshot.Final,
		Scenario: snapshot.Scenario,
		Frame: IntendedFrame{
			Size: Size{
				Width:  snapshot.Frame.Size.Width,
				Height: snapshot.Frame.Size.Height,
			},
			Cells: make([]Cell, len(snapshot.Frame.Cells)),
		},
		Cursor: CursorState{
			Visible: snapshot.Cursor.Visible,
			Position: Point{
				X: snapshot.Cursor.Position.X,
				Y: snapshot.Cursor.Position.Y,
			},
		},
		Controls:     make([]ControlSnapshot, len(snapshot.Controls)),
		Layouts:      make([]LayoutSnapshot, len(snapshot.Layouts)),
		InputSources: make([]InputSourceSnapshot, len(snapshot.InputSources)),
		Overflows:    make([]OverflowSnapshot, len(snapshot.Overflows)),
	}
	for index, cell := range snapshot.Frame.Cells {
		projected.Frame.Cells[index] = Cell{
			Grapheme:   cell.Grapheme,
			Style:      StyleID(cell.Style),
			Foreground: colorFromCore(cell.Foreground),
			Background: colorFromCore(cell.Background),
			Attributes: StyleAttributes(cell.Attributes),
			Owner:      ControlID(cell.Owner),
		}
	}
	compactFrame(&projected.Frame, true)
	for index, control := range snapshot.Controls {
		projectedControl := ControlSnapshot{
			ID:             ControlID(control.ID),
			Key:            control.Key,
			Kind:           ControlKind(control.Kind),
			Parent:         ControlID(control.Parent),
			Children:       make([]ControlID, len(control.Children)),
			Bounds:         rectFromCore(control.Bounds),
			AbsoluteBounds: rectFromCore(control.AbsoluteBounds),
			EffectiveClip:  rectFromCore(control.EffectiveClip),
			Minimum:        sizeFromCore(control.Minimum),
			Layout:         LayoutID(control.Layout),
			LayoutIndex:    control.LayoutIndex,
			StackIndex:     control.StackIndex,
			Style:          StyleID(control.Style),
			ResolvedStyle:  resolvedStyleFromCore(control.ResolvedStyle),
			Visible:        control.Visible,
			Focused:        control.Focused,
			Details: ControlDetails{
				Version: control.Details.Version,
			},
		}
		for childIndex, child := range control.Children {
			projectedControl.Children[childIndex] = ControlID(child)
		}
		if control.Details.Container != nil {
			projectedControl.Details.Container = &ContainerDetails{
				ClientInset: control.Details.Container.ClientInset,
				InputScope: InputScopeMode(
					control.Details.Container.InputScope,
				),
			}
		}
		if control.Details.Border != nil {
			var foregroundOverride, backgroundOverride *Color
			if color := control.Details.Border.ForegroundOverride; color != nil {
				projected := colorFromCore(*color)
				foregroundOverride = &projected
			}
			if color := control.Details.Border.BackgroundOverride; color != nil {
				projected := colorFromCore(*color)
				backgroundOverride = &projected
			}
			projectedControl.Details.Border = &BorderDetails{
				Title:              control.Details.Border.Title,
				Form:               string(control.Details.Border.Form),
				Style:              StyleID(control.Details.Border.Style),
				ResolvedStyle:      resolvedStyleFromCore(control.Details.Border.ResolvedStyle),
				ForegroundOverride: foregroundOverride,
				BackgroundOverride: backgroundOverride,
			}
		}
		if control.Details.Text != nil {
			projectedControl.Details.Text = &TextDetails{
				Text: control.Details.Text.Text,
				HorizontalAlignment: TextAlignment(
					control.Details.Text.HorizontalAlignment,
				),
				VerticalAlignment: TextAlignment(
					control.Details.Text.VerticalAlignment,
				),
				Wrap:     TextWrap(control.Details.Text.Wrap),
				Target:   ControlID(control.Details.Text.Target),
				Mnemonic: Key(control.Details.Text.Mnemonic),
			}
		}
		if control.Details.Divider != nil {
			projectedControl.Details.Divider = &DividerDetails{
				Orientation: Orientation(control.Details.Divider.Orientation),
				Form:        string(control.Details.Divider.Form),
				Text:        control.Details.Divider.Text,
				Alignment: TextAlignment(
					control.Details.Divider.Alignment,
				),
			}
		}
		if control.Details.Action != nil {
			projectedControl.Details.Action = &ActionDetails{
				Label:          control.Details.Action.Label,
				Command:        string(control.Details.Action.Command),
				Enabled:        control.Details.Action.Enabled,
				DisabledReason: control.Details.Action.DisabledReason,
				Checked:        control.Details.Action.Checked,
				Mnemonic:       Key(control.Details.Action.Mnemonic),
				Pressed:        control.Details.Action.Pressed,
				Default:        control.Details.Action.Default,
				Cancel:         control.Details.Action.Cancel,
			}
		}
		if control.Details.HotkeyBar != nil {
			items := control.Details.HotkeyBar.Items
			projectedControl.Details.HotkeyBar = &HotkeyBarDetails{
				Items: make([]HotkeyBarItemDetails, len(items)),
			}
			for itemIndex, item := range items {
				projectedItem := HotkeyBarItemDetails{
					Label:          item.Label,
					Command:        string(item.Command),
					Enabled:        item.Enabled,
					DisabledReason: item.DisabledReason,
					Checked:        item.Checked,
				}
				if item.Chord != nil {
					projectedItem.Chord = &Chord{
						Key: Key(item.Chord.Key),
						Modifiers: make(
							[]Key,
							len(item.Chord.Modifiers),
						),
					}
					for modifierIndex, modifier := range item.Chord.Modifiers {
						projectedItem.Chord.Modifiers[modifierIndex] =
							Key(modifier)
					}
				}
				projectedControl.Details.HotkeyBar.Items[itemIndex] =
					projectedItem
			}
		}
		if control.Details.MenuBar != nil {
			menuBar := control.Details.MenuBar
			projectedControl.Details.MenuBar = &MenuBarDetails{
				Entries: make(
					[]MenuEntryDetails,
					len(menuBar.Entries),
				),
				OpenPath: append([]string{}, menuBar.OpenPath...),
				SelectedPath: append(
					[]string{},
					menuBar.SelectedPath...,
				),
			}
			for entryIndex, entry := range menuBar.Entries {
				projectedEntry := MenuEntryDetails{
					Key:            entry.Key,
					ParentKey:      entry.ParentKey,
					Depth:          entry.Depth,
					Kind:           MenuItemKind(entry.Kind),
					Label:          entry.Label,
					Command:        string(entry.Command),
					Enabled:        entry.Enabled,
					DisabledReason: entry.DisabledReason,
					Checked:        entry.Checked,
					Mnemonic:       Key(entry.Mnemonic),
					Placement:      MenuBarPlacement(entry.Placement),
					Selected:       entry.Selected,
					Open:           entry.Open,
					ChildCount:     entry.ChildCount,
				}
				if entry.Chord != nil {
					projectedEntry.Chord = &Chord{
						Key: Key(entry.Chord.Key),
						Modifiers: make(
							[]Key,
							len(entry.Chord.Modifiers),
						),
					}
					for modifierIndex, modifier := range entry.Chord.Modifiers {
						projectedEntry.Chord.Modifiers[modifierIndex] =
							Key(modifier)
					}
				}
				projectedControl.Details.MenuBar.Entries[entryIndex] =
					projectedEntry
			}
		}
		if control.Details.StatusBar != nil {
			statusBar := control.Details.StatusBar
			projectedControl.Details.StatusBar = &StatusBarDetails{
				Segments: make(
					[]StatusSegmentDetails,
					len(statusBar.Segments),
				),
			}
			for segmentIndex, segment := range statusBar.Segments {
				projectedSegment := StatusSegmentDetails{
					Key:            segment.Key,
					Label:          segment.Label,
					Command:        string(segment.Command),
					Priority:       segment.Priority,
					Enabled:        segment.Enabled,
					DisabledReason: segment.DisabledReason,
					Checked:        segment.Checked,
					Rendered:       segment.Rendered,
					Bounds:         rectFromCore(segment.Bounds),
					Clipped:        segment.Clipped,
				}
				if segment.Chord != nil {
					projectedSegment.Chord = &Chord{
						Key: Key(segment.Chord.Key),
						Modifiers: make(
							[]Key,
							len(segment.Chord.Modifiers),
						),
					}
					for modifierIndex, modifier := range segment.Chord.Modifiers {
						projectedSegment.Chord.Modifiers[modifierIndex] =
							Key(modifier)
					}
				}
				projectedControl.Details.StatusBar.Segments[segmentIndex] =
					projectedSegment
			}
		}
		if details := control.Details.Checkbox; details != nil {
			projectedControl.Details.Checkbox = &CheckboxDetails{
				Label: details.Label, State: string(details.State),
				ThreeState: details.ThreeState, Enabled: details.Enabled,
				DisabledReason: details.DisabledReason,
				Mnemonic:       Key(details.Mnemonic),
				ChangeCommand:  string(details.ChangeCommand),
			}
		}
		if details := control.Details.RadioButton; details != nil {
			projectedControl.Details.RadioButton = &RadioButtonDetails{
				Value: details.Value, Label: details.Label,
				Selected: details.Selected, Enabled: details.Enabled,
				DisabledReason: details.DisabledReason,
				Mnemonic:       Key(details.Mnemonic),
			}
		}
		if details := control.Details.RadioGroup; details != nil {
			group := &RadioGroupDetails{
				Value: details.Value, AllowEmpty: details.AllowEmpty,
				Enabled:        details.Enabled,
				DisabledReason: details.DisabledReason,
				ChangeCommand:  string(details.ChangeCommand),
				Options: make(
					[]RadioOptionDetails,
					len(details.Options),
				),
			}
			for optionIndex, option := range details.Options {
				group.Options[optionIndex] = RadioOptionDetails{
					Control: ControlID(option.Control), Value: option.Value,
					Label: option.Label, Selected: option.Selected,
					Enabled:        option.Enabled,
					DisabledReason: option.DisabledReason,
				}
			}
			projectedControl.Details.RadioGroup = group
		}
		if details := control.Details.ChoiceField; details != nil {
			field := &ChoiceFieldDetails{
				Label: details.Label, Value: details.Value,
				SelectedIndex:  details.SelectedIndex,
				Enabled:        details.Enabled,
				DisabledReason: details.DisabledReason,
				Mnemonic:       Key(details.Mnemonic),
				ChangeCommand:  string(details.ChangeCommand),
				Options: make(
					[]SelectionOptionDetails,
					len(details.Options),
				),
			}
			for optionIndex, option := range details.Options {
				field.Options[optionIndex] = SelectionOptionDetails{
					Value: option.Value, Label: option.Label,
					Enabled:        option.Enabled,
					DisabledReason: option.DisabledReason,
					Selected:       option.Selected,
				}
			}
			projectedControl.Details.ChoiceField = field
		}
		if details := control.Details.FocusGuideBar; details != nil {
			projectedControl.Details.FocusGuideBar = &FocusGuideBarDetails{
				Target:        ControlID(details.Target),
				TargetKind:    ControlKind(details.TargetKind),
				Text:          details.Text,
				Customization: string(details.Customization),
			}
		}
		if details := control.Details.TextField; details != nil {
			field := textFieldDetailsFromCore(details)
			if sensitiveInput[control.ID] {
				field.Text = ""
				field.Redacted = true
				if field.Validator != nil {
					field.Validator.Characters = ""
					field.Validator.CharactersRedacted = true
				}
			}
			projectedControl.Details.TextField = &field
		}
		if details := control.Details.NumberField; details != nil {
			projectedControl.Details.NumberField = &NumberFieldDetails{
				Text: details.Text, Value: details.Value,
				Length: details.Length, Caret: details.Caret,
				SelectionStart: details.SelectionStart,
				SelectionEnd:   details.SelectionEnd,
				ViewOffset:     details.ViewOffset, Editing: details.Editing,
				Valid: details.Valid, InvalidReason: details.InvalidReason,
				Minimum:       cloneFloat64(details.Minimum),
				Maximum:       cloneFloat64(details.Maximum),
				DecimalPlaces: details.DecimalPlaces, Step: details.Step,
				Enabled:        details.Enabled,
				DisabledReason: details.DisabledReason,
				ChangeCommand:  string(details.ChangeCommand),
			}
		}
		if details := control.Details.TextArea; details != nil {
			area := &TextAreaDetails{
				Text: details.Text, Length: details.Length,
				LineCount: details.LineCount, Caret: details.Caret,
				SelectionStart:    details.SelectionStart,
				SelectionEnd:      details.SelectionEnd,
				VisualCaretRow:    details.VisualCaretRow,
				VisualCaretColumn: details.VisualCaretColumn,
				RowOffset:         details.RowOffset,
				ColumnOffset:      details.ColumnOffset,
				Wrap:              TextWrap(details.Wrap),
				Editing:           details.Editing, Valid: details.Valid,
				Password: details.Password, Redacted: details.Redacted,
				Enabled:        details.Enabled,
				DisabledReason: details.DisabledReason,
				ChangeCommand:  string(details.ChangeCommand),
			}
			if details.Validator != nil {
				area.Validator = &TextValidatorDetails{
					Enforcement: string(details.Validator.Enforcement),
					Mode:        string(details.Validator.Mode),
					Characters:  details.Validator.Characters,
				}
			}
			projectedControl.Details.TextArea = area
		}
		if details := control.Details.Progress; details != nil {
			projectedControl.Details.Progress = &ProgressDetails{
				Status:        string(details.Status),
				Current:       details.Current,
				Total:         details.Total,
				Value:         details.Value,
				Minimum:       details.Minimum,
				Maximum:       details.Maximum,
				Orientation:   Orientation(details.Orientation),
				Indeterminate: details.Indeterminate,
				Tick:          details.Tick,
				ReducedMotion: details.ReducedMotion,
				TextMode:      string(details.TextMode),
				FrameIndex:    details.FrameIndex,
			}
		}
		if details := control.Details.ScrollBar; details != nil {
			projectedControl.Details.ScrollBar = &ScrollBarDetails{
				Orientation:    Orientation(details.Orientation),
				ContentSize:    details.ContentSize,
				ViewportSize:   details.ViewportSize,
				Offset:         details.Offset,
				MaximumOffset:  details.MaximumOffset,
				ArrowStep:      details.ArrowStep,
				PageStep:       details.PageStep,
				TrackStart:     details.TrackStart,
				TrackSize:      details.TrackSize,
				ThumbStart:     details.ThumbStart,
				ThumbSize:      details.ThumbSize,
				Enabled:        details.Enabled,
				DisabledReason: details.DisabledReason,
				ChangeCommand:  string(details.ChangeCommand),
			}
		}
		if details := control.Details.TabbedPanel; details != nil {
			tabs := make([]TabDetails, len(details.Tabs))
			for tabIndex := range details.Tabs {
				tab := details.Tabs[tabIndex]
				tabs[tabIndex] = TabDetails{
					Key:            tab.Key,
					Value:          tab.Value,
					Label:          tab.Label,
					Mnemonic:       Key(tab.Mnemonic),
					Page:           ControlID(tab.Page),
					PageKey:        tab.PageKey,
					Enabled:        tab.Enabled,
					DisabledReason: tab.DisabledReason,
					Selected:       tab.Selected,
					Current:        tab.Current,
					Bounds:         rectFromCore(tab.Bounds),
					Omitted:        tab.Omitted,
					Clipped:        tab.Clipped,
				}
			}
			projectedControl.Details.TabbedPanel = &TabbedPanelDetails{
				Tabs:            tabs,
				Selected:        details.Selected,
				Current:         details.Current,
				ChangeCommand:   string(details.ChangeCommand),
				LeadingOmitted:  details.LeadingOmitted,
				TrailingOmitted: details.TrailingOmitted,
			}
		}
		if details := control.Details.Scrollable; details != nil {
			projectedControl.Details.Scrollable =
				scrollableDetailsFromCore(details)
		}
		if details := control.Details.Markdown; details != nil {
			blocks := make([]MarkdownBlockDetails, len(details.Blocks))
			for blockIndex, block := range details.Blocks {
				blocks[blockIndex] = MarkdownBlockDetails{
					Kind:          block.Kind,
					Level:         block.Level,
					SourceLine:    block.SourceLine,
					SourceLines:   block.SourceLines,
					RenderedStart: block.RenderedStart,
					RenderedRows:  block.RenderedRows,
				}
			}
			projectedControl.Details.Markdown = &MarkdownDetails{
				SourceBytes:        details.SourceBytes,
				SourceCells:        details.SourceCells,
				BlockCount:         details.BlockCount,
				RenderedRows:       details.RenderedRows,
				MaximumLineWidth:   details.MaximumLineWidth,
				Viewport:           contentViewportDetailsFromCore(&details.Viewport),
				Blocks:             blocks,
				SummariesTruncated: details.SummariesTruncated,
			}
		}
		if details := control.Details.LogView; details != nil {
			projectedControl.Details.LogView = &LogViewDetails{
				Capacity: ContentCapacity{
					Records: details.Capacity.Records,
					Bytes:   details.Capacity.Bytes,
				},
				RetainedRecords: details.RetainedRecords,
				RetainedBytes:   details.RetainedBytes,
				DroppedRecords:  details.DroppedRecords,
				DroppedBytes:    details.DroppedBytes,
				FirstKey:        details.FirstKey,
				LastKey:         details.LastKey,
				Follow:          details.Follow,
				Viewport:        contentViewportDetailsFromCore(&details.Viewport),
			}
		}
		if details := control.Details.StreamView; details != nil {
			projectedControl.Details.StreamView = &StreamViewDetails{
				Capacity: ContentCapacity{
					Records: details.Capacity.Records,
					Bytes:   details.Capacity.Bytes,
				},
				RetainedLines:       details.RetainedLines,
				RetainedBytes:       details.RetainedBytes,
				PendingBytes:        details.PendingBytes,
				PendingCells:        details.PendingCells,
				PendingStorageBytes: details.PendingStorageBytes,
				PendingTruncated:    details.PendingTruncated,
				DroppedLines:        details.DroppedLines,
				DroppedBytes:        details.DroppedBytes,
				Follow:              details.Follow,
				Viewport: contentViewportDetailsFromCore(
					&details.Viewport,
				),
			}
		}
		if details := control.Details.ListBox; details != nil {
			statusBytes, statusDigest := compactTextEvidence(details.StatusMessage)
			projectedControl.Details.ListBox = &ListBoxDetails{
				Status:              string(details.Status),
				StatusMessageBytes:  statusBytes,
				StatusMessageDigest: statusDigest,
				ItemCount:           details.ItemCount,
				EnabledCount:        details.EnabledCount,
				VisualRowCount:      details.VisualRowCount,
				Wrap:                TextWrap(details.Wrap),
				RetainedBytes:       details.RetainedBytes,
				Current:             details.Current,
				CurrentIndex:        details.CurrentIndex,
				SelectionMode:       string(details.SelectionMode),
				SelectionMarks:      details.SelectionMarks,
				RequireSelection:    details.RequireSelection,
				SelectedCount:       details.SelectedCount,
				FirstSelected:       details.FirstSelected,
				LastSelected:        details.LastSelected,
				SelectionDigest:     details.SelectionDigest,
				Enabled:             details.Enabled,
				DisabledReasonBytes: len(details.DisabledReason),
				ChangeCommand:       string(details.ChangeCommand),
				CurrentCommand:      string(details.CurrentCommand),
				ActivateCommand:     string(details.ActivateCommand),
				Viewport: contentViewportDetailsFromCore(
					&details.Viewport,
				),
			}
		}
		if details := control.Details.TreeView; details != nil {
			statusBytes, statusDigest := compactTextEvidence(details.StatusMessage)
			projectedControl.Details.TreeView = &TreeViewDetails{
				Status:              string(details.Status),
				StatusMessageBytes:  statusBytes,
				StatusMessageDigest: statusDigest,
				NodeCount:           details.NodeCount,
				VisibleCount:        details.VisibleCount,
				EnabledCount:        details.EnabledCount,
				RetainedBytes:       details.RetainedBytes,
				Current:             details.Current,
				CurrentIndex:        details.CurrentIndex,
				SelectionMode:       string(details.SelectionMode),
				RequireSelection:    details.RequireSelection,
				SelectedCount:       details.SelectedCount,
				FirstSelected:       details.FirstSelected,
				LastSelected:        details.LastSelected,
				SelectionDigest:     details.SelectionDigest,
				ExpandedCount:       details.ExpandedCount,
				FirstExpanded:       details.FirstExpanded,
				LastExpanded:        details.LastExpanded,
				ExpansionDigest:     details.ExpansionDigest,
				Enabled:             details.Enabled,
				DisabledReasonBytes: len(details.DisabledReason),
				ChangeCommand:       string(details.ChangeCommand),
				ActivateCommand:     string(details.ActivateCommand),
				ExpandCommand:       string(details.ExpandCommand),
				Viewport: contentViewportDetailsFromCore(
					&details.Viewport,
				),
			}
		}
		if details := control.Details.Table; details != nil {
			statusBytes, statusDigest := compactTextEvidence(details.StatusMessage)
			projectedControl.Details.Table = &TableDetails{
				Status:              string(details.Status),
				StatusMessageBytes:  statusBytes,
				StatusMessageDigest: statusDigest,
				RowCount:            details.RowCount,
				EnabledCount:        details.EnabledCount,
				ColumnCount:         details.ColumnCount,
				CellCount:           details.CellCount,
				RetainedBytes:       details.RetainedBytes,
				CurrentRow:          details.CurrentRow,
				CurrentRowIndex:     details.CurrentRowIndex,
				CurrentColumn:       details.CurrentColumn,
				CurrentColumnIndex:  details.CurrentColumnIndex,
				FocusMode:           string(details.FocusMode),
				SelectionMode:       string(details.SelectionMode),
				RequireSelection:    details.RequireSelection,
				SelectedCount:       details.SelectedCount,
				FirstSelected:       details.FirstSelected,
				LastSelected:        details.LastSelected,
				SelectionDigest:     details.SelectionDigest,
				SortColumn:          details.SortColumn,
				SortDirection:       string(details.SortDirection),
				FirstColumn:         details.FirstColumn,
				LastColumn:          details.LastColumn,
				ColumnWidthsDigest:  details.ColumnWidthsDigest,
				Enabled:             details.Enabled,
				DisabledReasonBytes: len(details.DisabledReason),
				ChangeCommand:       string(details.ChangeCommand),
				ActivateCommand:     string(details.ActivateCommand),
				SortCommand:         string(details.SortCommand),
				Viewport: contentViewportDetailsFromCore(
					&details.Viewport,
				),
			}
		}
		if details := control.Details.DataGrid; details != nil {
			table := details.Table
			statusBytes, statusDigest := compactTextEvidence(table.StatusMessage)
			projected := &DataGridDetails{
				Table: TableDetails{
					Status:              string(table.Status),
					StatusMessageBytes:  statusBytes,
					StatusMessageDigest: statusDigest,
					RowCount:            table.RowCount,
					EnabledCount:        table.EnabledCount,
					ColumnCount:         table.ColumnCount,
					CellCount:           table.CellCount,
					RetainedBytes:       table.RetainedBytes,
					CurrentRow:          table.CurrentRow,
					CurrentRowIndex:     table.CurrentRowIndex,
					CurrentColumn:       table.CurrentColumn,
					CurrentColumnIndex:  table.CurrentColumnIndex,
					FocusMode:           string(table.FocusMode),
					SelectionMode:       string(table.SelectionMode),
					RequireSelection:    table.RequireSelection,
					SelectedCount:       table.SelectedCount,
					FirstSelected:       table.FirstSelected,
					LastSelected:        table.LastSelected,
					SelectionDigest:     table.SelectionDigest,
					SortColumn:          table.SortColumn,
					SortDirection:       string(table.SortDirection),
					FirstColumn:         table.FirstColumn,
					LastColumn:          table.LastColumn,
					ColumnWidthsDigest:  table.ColumnWidthsDigest,
					Enabled:             table.Enabled,
					DisabledReasonBytes: len(table.DisabledReason),
					ChangeCommand:       string(table.ChangeCommand),
					ActivateCommand:     string(table.ActivateCommand),
					SortCommand:         string(table.SortCommand),
					Viewport: contentViewportDetailsFromCore(
						&table.Viewport,
					),
				},
				Editing:    details.Editing,
				EditRow:    details.EditRow,
				EditColumn: details.EditColumn,
			}
			if details.Editor != nil {
				projected.EditLength = details.Editor.Length
				projected.EditCaret = details.Editor.Caret
				projected.EditViewOffset = details.Editor.ViewOffset
				projected.EditValid = details.Editor.Valid
				if details.Editor.Validator != nil {
					projected.ValidationEnforcement = string(
						details.Editor.Validator.Enforcement,
					)
					projected.ValidationMode = string(details.Editor.Validator.Mode)
				}
			}
			projectedControl.Details.DataGrid = projected
		}
		if details := control.Details.DropDown; details != nil {
			dropDown := dropDownDetailsFromCore(details)
			projectedControl.Details.DropDown = &dropDown
		}
		if details := control.Details.ComboBox; details != nil {
			projectedControl.Details.ComboBox = &ComboBoxDetails{
				Popup:  dropDownDetailsFromCore(&details.Popup),
				Editor: textFieldDetailsFromCore(&details.Editor),
			}
		}
		if details := control.Details.ModalPanel; details != nil {
			projected := &ModalPanelDetails{
				Lifecycle:       string(details.Lifecycle),
				Active:          details.Active,
				Top:             details.Top,
				StackIndex:      details.StackIndex,
				StackDepth:      details.StackDepth,
				NestedOwner:     ControlID(details.NestedOwner),
				SavedFocus:      ControlID(details.SavedFocus),
				InitialFocus:    ControlID(details.InitialFocus),
				RequestedSize:   sizeFromCore(details.RequestedSize),
				ResolvedBounds:  rectFromCore(details.ResolvedBounds),
				RequiredMinimum: sizeFromCore(details.RequiredMinimum),
				Degraded:        details.Degraded,
				Shadow:          string(details.Shadow),
				ShadowStyle:     StyleID(details.ShadowStyle),
			}
			if details.Result != nil {
				projected.Result = &ModalResultDetails{
					Reason: string(details.Result.Reason),
					Action: string(details.Result.Action),
				}
			}
			projectedControl.Details.ModalPanel = projected
		}
		if details := control.Details.ProgressDialog; details != nil {
			projectedControl.Details.ProgressDialog = &ProgressDialogDetails{
				StatusLength: details.StatusLength,
				Progress: ProgressDialogProgressDetails{
					Status:        string(details.Progress.Status),
					Current:       details.Progress.Current,
					Total:         details.Progress.Total,
					Indeterminate: details.Progress.Indeterminate,
					Tick:          details.Progress.Tick,
					ReducedMotion: details.Progress.ReducedMotion,
					TextMode:      string(details.Progress.TextMode),
				},
				Cancellable:     details.Cancellable,
				CancelRequested: details.CancelRequested,
			}
		}
		if details := control.Details.FilePicker; details != nil {
			pathBytes, pathDigest := compactTextEvidence(details.DisplayPath)
			nameBytes, nameDigest := compactTextEvidence(details.CurrentName)
			errorBytes, errorDigest := compactTextEvidence(details.Error)
			projectedControl.Details.FilePicker = &FilePickerDetails{
				Mode: details.Mode, Status: string(details.Status),
				DisplayPathBytes: pathBytes, DisplayPathDigest: pathDigest,
				EntryCount: details.EntryCount, FileCount: details.FileCount,
				DirectoryCount:   details.DirectoryCount,
				CurrentNameBytes: nameBytes, CurrentNameDigest: nameDigest,
				CurrentKind:   string(details.CurrentKind),
				SelectedCount: details.SelectedCount, Filter: details.Filter,
				SortField:     string(details.SortField),
				SortDirection: string(details.SortDirection),
				ErrorBytes:    errorBytes, ErrorDigest: errorDigest,
			}
		}
		projected.Controls[index] = projectedControl
	}
	for index, layout := range snapshot.Layouts {
		projectedLayout := LayoutSnapshot{
			ID:          LayoutID(layout.ID),
			Key:         layout.Key,
			Kind:        LayoutKind(layout.Kind),
			Owner:       ControlID(layout.Owner),
			Parent:      LayoutID(layout.Parent),
			Bounds:      rectFromCore(layout.Bounds),
			OwnerBounds: rectFromCore(layout.OwnerBounds),
			Minimum:     sizeFromCore(layout.Minimum),
			LayoutIndex: layout.LayoutIndex,
			StackIndex:  layout.StackIndex,
			Items:       make([]LayoutItemSnapshot, len(layout.Items)),
		}
		if layout.Border != nil {
			projectedLayout.Border = borderDetailsFromCore(layout.Border)
		}
		for itemIndex, item := range layout.Items {
			projectedLayout.Items[itemIndex] = LayoutItemSnapshot{
				Kind:        item.Kind,
				Panel:       ControlID(item.Panel),
				Layout:      LayoutID(item.Layout),
				Bounds:      rectFromCore(item.Bounds),
				Minimum:     sizeFromCore(item.Minimum),
				LayoutIndex: item.LayoutIndex,
				StackIndex:  item.StackIndex,
			}
		}
		projected.Layouts[index] = projectedLayout
	}
	for index, source := range snapshot.InputSources {
		projectedSource := InputSourceSnapshot{
			Source: source.Source,
			Held:   make([]Key, len(source.Held)),
		}
		for keyIndex, key := range source.Held {
			projectedSource.Held[keyIndex] = Key(key)
		}
		projected.InputSources[index] = projectedSource
	}
	for index, overflow := range snapshot.Overflows {
		projected.Overflows[index] = OverflowSnapshot{
			EpisodeID: overflow.EpisodeID,
			Panel:     ControlID(overflow.Panel),
			Layout:    LayoutID(overflow.Layout),
			Available: sizeFromCore(overflow.Available),
			Required:  sizeFromCore(overflow.Required),
			Deficit:   sizeFromCore(overflow.Deficit),
			State:     overflow.State,
		}
	}
	if snapshot.Completion != nil {
		projected.Completion = &SnapshotCompletion{
			RequestID:     snapshot.Completion.RequestID,
			Outcome:       string(snapshot.Completion.Outcome),
			Command:       string(snapshot.Completion.Command),
			FrameSequence: snapshot.Completion.FrameSequence,
			Code:          snapshot.Completion.Code,
			Message:       snapshot.Completion.Message,
		}
	}
	return projected
}

func compactTextEvidence(value string) (int, string) {
	if value == "" {
		return 0, ""
	}
	hash := sha256.Sum256([]byte(value))
	return len(value), fmt.Sprintf("%x", hash)
}

func textFieldDetailsFromCore(
	details *expletives.TextFieldDetails,
) TextFieldDetails {
	field := TextFieldDetails{
		Text: details.Text, Length: details.Length,
		MaximumBytes:   details.MaximumBytes,
		Caret:          details.Caret,
		SelectionStart: details.SelectionStart,
		SelectionEnd:   details.SelectionEnd,
		ViewOffset:     details.ViewOffset,
		Editing:        details.Editing, Valid: details.Valid,
		Password: details.Password, Redacted: details.Redacted,
		Enabled:        details.Enabled,
		DisabledReason: details.DisabledReason,
		ChangeCommand:  string(details.ChangeCommand),
		EditCommand:    string(details.EditCommand),
		SubmitCommand:  string(details.SubmitCommand),
		FocusedStyle:   string(details.FocusedStyle),
		EditingStyle:   string(details.EditingStyle),
		ByteStyles: make(
			[]TextFieldByteStyleDetails,
			len(details.ByteStyles),
		),
	}
	for index, band := range details.ByteStyles {
		field.ByteStyles[index] = TextFieldByteStyleDetails{
			MinimumBytes: band.MinimumBytes,
			Style:        string(band.Style),
		}
	}
	if details.Validator != nil {
		field.Validator = &TextValidatorDetails{
			Enforcement: string(details.Validator.Enforcement),
			Mode:        string(details.Validator.Mode),
			Characters:  details.Validator.Characters,
		}
	}
	return field
}

func dropDownDetailsFromCore(
	details *expletives.DropDownDetails,
) DropDownDetails {
	return DropDownDetails{
		ItemCount: details.ItemCount, EnabledCount: details.EnabledCount,
		RetainedBytes: details.RetainedBytes,
		Current:       details.Current, CurrentIndex: details.CurrentIndex,
		Selected: details.Selected, SelectedIndex: details.SelectedIndex,
		AllowEmpty: details.AllowEmpty, PopupRows: details.PopupRows,
		Open: details.Open, PopupBounds: rectFromCore(details.PopupBounds),
		PopupOffset: details.PopupOffset, PopupCurrent: details.PopupCurrent,
		PopupSelection:      details.PopupSelection,
		Enabled:             details.Enabled,
		DisabledReasonBytes: len(details.DisabledReason),
		ChangeCommand:       string(details.ChangeCommand),
		ActivateCommand:     string(details.ActivateCommand),
	}
}

func scrollableDetailsFromCore(
	details *expletives.ScrollableDetails,
) *ScrollableDetails {
	if details == nil {
		return nil
	}
	scrollable := &ScrollableDetails{
		State: ViewportState{
			ContentSize: sizeFromCore(details.State.ContentSize),
			Offset:      pointFromCore(details.State.Offset),
		},
		MaximumOffset:     pointFromCore(details.MaximumOffset),
		ViewportBounds:    rectFromCore(details.ViewportBounds),
		ArrowStep:         sizeFromCore(details.ArrowStep),
		PageStep:          sizeFromCore(details.PageStep),
		Enabled:           details.Enabled,
		DisabledReason:    details.DisabledReason,
		ChangeCommand:     string(details.ChangeCommand),
		Content:           ControlID(details.Content),
		ContentKey:        details.ContentKey,
		HorizontalPolicy:  string(details.HorizontalPolicy),
		VerticalPolicy:    string(details.VerticalPolicy),
		HorizontalVisible: details.HorizontalVisible,
		VerticalVisible:   details.VerticalVisible,
	}
	if details.HorizontalBar != nil {
		scrollable.HorizontalBar =
			scrollBarDetailsFromCore(details.HorizontalBar)
	}
	if details.VerticalBar != nil {
		scrollable.VerticalBar =
			scrollBarDetailsFromCore(details.VerticalBar)
	}
	return scrollable
}

func contentViewportDetailsFromCore(
	details *expletives.ScrollableDetails,
) ContentViewportDetails {
	if details == nil {
		return ContentViewportDetails{}
	}
	return ContentViewportDetails{
		State: ViewportState{
			ContentSize: sizeFromCore(details.State.ContentSize),
			Offset:      pointFromCore(details.State.Offset),
		},
		MaximumOffset:     pointFromCore(details.MaximumOffset),
		ViewportBounds:    rectFromCore(details.ViewportBounds),
		HorizontalPolicy:  string(details.HorizontalPolicy),
		VerticalPolicy:    string(details.VerticalPolicy),
		HorizontalVisible: details.HorizontalVisible,
		VerticalVisible:   details.VerticalVisible,
	}
}

func cloneSnapshot(snapshot SnapshotV1) SnapshotV1 {
	cloned := snapshot
	cloned.Frame.Cells = append([]Cell{}, snapshot.Frame.Cells...)
	cloned.Frame.Runs = append([]CellRun{}, snapshot.Frame.Runs...)
	cloned.Controls = append([]ControlSnapshot{}, snapshot.Controls...)
	for index := range cloned.Controls {
		cloned.Controls[index].Children = append(
			[]ControlID{},
			snapshot.Controls[index].Children...,
		)
		if snapshot.Controls[index].Details.Container != nil {
			container := *snapshot.Controls[index].Details.Container
			cloned.Controls[index].Details.Container = &container
		}
		if snapshot.Controls[index].Details.Border != nil {
			border := *snapshot.Controls[index].Details.Border
			if border.ForegroundOverride != nil {
				color := *border.ForegroundOverride
				border.ForegroundOverride = &color
			}
			if border.BackgroundOverride != nil {
				color := *border.BackgroundOverride
				border.BackgroundOverride = &color
			}
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
				[]HotkeyBarItemDetails{},
				snapshot.Controls[index].Details.HotkeyBar.Items...,
			)
			for itemIndex := range hotkeyBar.Items {
				if hotkeyBar.Items[itemIndex].Chord != nil {
					chord := *hotkeyBar.Items[itemIndex].Chord
					chord.Modifiers = append(
						[]Key{},
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
				[]MenuEntryDetails{},
				snapshot.Controls[index].Details.MenuBar.Entries...,
			)
			for entryIndex := range menuBar.Entries {
				if menuBar.Entries[entryIndex].Chord != nil {
					chord := *menuBar.Entries[entryIndex].Chord
					chord.Modifiers = append(
						[]Key{},
						menuBar.Entries[entryIndex].Chord.Modifiers...,
					)
					menuBar.Entries[entryIndex].Chord = &chord
				}
			}
			menuBar.OpenPath = append(
				[]string{},
				snapshot.Controls[index].Details.MenuBar.OpenPath...,
			)
			menuBar.SelectedPath = append(
				[]string{},
				snapshot.Controls[index].Details.MenuBar.SelectedPath...,
			)
			cloned.Controls[index].Details.MenuBar = &menuBar
		}
		if snapshot.Controls[index].Details.StatusBar != nil {
			statusBar := *snapshot.Controls[index].Details.StatusBar
			statusBar.Segments = append(
				[]StatusSegmentDetails{},
				snapshot.Controls[index].Details.StatusBar.Segments...,
			)
			for segmentIndex := range statusBar.Segments {
				if statusBar.Segments[segmentIndex].Chord != nil {
					chord := *statusBar.Segments[segmentIndex].Chord
					chord.Modifiers = append(
						[]Key{},
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
				[]RadioOptionDetails{},
				snapshot.Controls[index].Details.RadioGroup.Options...,
			)
			cloned.Controls[index].Details.RadioGroup = &group
		}
		if snapshot.Controls[index].Details.ChoiceField != nil {
			field := *snapshot.Controls[index].Details.ChoiceField
			field.Options = append(
				[]SelectionOptionDetails{},
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
			field.ByteStyles = append(
				[]TextFieldByteStyleDetails{},
				snapshot.Controls[index].Details.TextField.ByteStyles...,
			)
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
			cloned.Controls[index].Details.Markdown = &markdown
		}
		if snapshot.Controls[index].Details.LogView != nil {
			logView := *snapshot.Controls[index].Details.LogView
			cloned.Controls[index].Details.LogView = &logView
		}
		if snapshot.Controls[index].Details.StreamView != nil {
			streamView := *snapshot.Controls[index].Details.StreamView
			cloned.Controls[index].Details.StreamView = &streamView
		}
		if snapshot.Controls[index].Details.ListBox != nil {
			listBox := *snapshot.Controls[index].Details.ListBox
			cloned.Controls[index].Details.ListBox = &listBox
		}
		if snapshot.Controls[index].Details.TreeView != nil {
			treeView := *snapshot.Controls[index].Details.TreeView
			cloned.Controls[index].Details.TreeView = &treeView
		}
		if snapshot.Controls[index].Details.Table != nil {
			table := *snapshot.Controls[index].Details.Table
			cloned.Controls[index].Details.Table = &table
		}
		if snapshot.Controls[index].Details.DataGrid != nil {
			dataGrid := *snapshot.Controls[index].Details.DataGrid
			cloned.Controls[index].Details.DataGrid = &dataGrid
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
		if snapshot.Controls[index].Details.ModalPanel != nil {
			modal := *snapshot.Controls[index].Details.ModalPanel
			if modal.Result != nil {
				result := *modal.Result
				modal.Result = &result
			}
			cloned.Controls[index].Details.ModalPanel = &modal
		}
		if snapshot.Controls[index].Details.ProgressDialog != nil {
			progress := *snapshot.Controls[index].Details.ProgressDialog
			cloned.Controls[index].Details.ProgressDialog = &progress
		}
		if snapshot.Controls[index].Details.FilePicker != nil {
			picker := *snapshot.Controls[index].Details.FilePicker
			cloned.Controls[index].Details.FilePicker = &picker
		}
	}
	cloned.Layouts = append([]LayoutSnapshot{}, snapshot.Layouts...)
	for index := range cloned.Layouts {
		cloned.Layouts[index].Items = append(
			[]LayoutItemSnapshot{},
			snapshot.Layouts[index].Items...,
		)
		if snapshot.Layouts[index].Border != nil {
			border := *snapshot.Layouts[index].Border
			if border.ForegroundOverride != nil {
				color := *border.ForegroundOverride
				border.ForegroundOverride = &color
			}
			if border.BackgroundOverride != nil {
				color := *border.BackgroundOverride
				border.BackgroundOverride = &color
			}
			cloned.Layouts[index].Border = &border
		}
	}
	cloned.InputSources = append([]InputSourceSnapshot{}, snapshot.InputSources...)
	for index := range cloned.InputSources {
		cloned.InputSources[index].Held = append([]Key{}, snapshot.InputSources[index].Held...)
	}
	cloned.Overflows = append([]OverflowSnapshot{}, snapshot.Overflows...)
	if snapshot.Completion != nil {
		completion := *snapshot.Completion
		cloned.Completion = &completion
	}
	return cloned
}

func compactFrame(frame *IntendedFrame, keepCells bool) {
	if frame == nil {
		return
	}
	if len(frame.Cells) == 0 {
		return
	}
	frame.Runs = frame.Runs[:0]
	for _, cell := range frame.Cells {
		last := len(frame.Runs) - 1
		if last >= 0 && frame.Runs[last].Cell == cell {
			frame.Runs[last].Count++
			continue
		}
		frame.Runs = append(frame.Runs, CellRun{Count: 1, Cell: cell})
	}
	if !keepCells {
		frame.Cells = nil
	}
}

func expandFrame(frame *IntendedFrame, cellCount, maxRuns int) error {
	if frame == nil {
		return nil
	}
	if len(frame.Runs) > maxRuns {
		return fmt.Errorf("snapshot frame run count exceeds advertised bound")
	}
	if len(frame.Cells) == cellCount {
		if len(frame.Runs) == 0 {
			compactFrame(frame, true)
			return nil
		}
		offset := 0
		for _, run := range frame.Runs {
			if run.Count <= 0 || run.Count > cellCount-offset {
				return fmt.Errorf("snapshot frame run exceeds bounded geometry")
			}
			for index := range run.Count {
				if frame.Cells[offset+index] != run.Cell {
					return fmt.Errorf(
						"snapshot frame runs disagree with expanded cells",
					)
				}
			}
			offset += run.Count
		}
		if offset != cellCount {
			return fmt.Errorf("snapshot frame runs do not match bounded geometry")
		}
		return nil
	}
	if len(frame.Cells) != 0 {
		return fmt.Errorf("snapshot cell array does not match bounded geometry")
	}
	frame.Cells = make([]Cell, 0, cellCount)
	for _, run := range frame.Runs {
		if run.Count <= 0 || run.Count > cellCount-len(frame.Cells) {
			return fmt.Errorf("snapshot frame run exceeds bounded geometry")
		}
		for range run.Count {
			frame.Cells = append(frame.Cells, run.Cell)
		}
	}
	if len(frame.Cells) != cellCount {
		return fmt.Errorf("snapshot frame runs do not match bounded geometry")
	}
	return nil
}

func borderDetailsFromCore(border *expletives.BorderDetails) *BorderDetails {
	if border == nil {
		return nil
	}
	var foregroundOverride, backgroundOverride *Color
	if border.ForegroundOverride != nil {
		projected := colorFromCore(*border.ForegroundOverride)
		foregroundOverride = &projected
	}
	if border.BackgroundOverride != nil {
		projected := colorFromCore(*border.BackgroundOverride)
		backgroundOverride = &projected
	}
	return &BorderDetails{
		Title:              border.Title,
		Form:               string(border.Form),
		Style:              StyleID(border.Style),
		ResolvedStyle:      resolvedStyleFromCore(border.ResolvedStyle),
		ForegroundOverride: foregroundOverride,
		BackgroundOverride: backgroundOverride,
	}
}

func colorFromCore(color expletives.Color) Color {
	return Color(color.String())
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func resolvedStyleFromCore(style expletives.ResolvedStyle) ResolvedStyle {
	return ResolvedStyle{
		Foreground: colorFromCore(style.Foreground),
		Background: colorFromCore(style.Background),
		Attributes: StyleAttributes(style.Attributes),
	}
}

func sizeFromCore(size expletives.Size) Size {
	return Size{Width: size.Width, Height: size.Height}
}

func pointFromCore(point expletives.Point) Point {
	return Point{X: point.X, Y: point.Y}
}

func rectFromCore(rect expletives.Rect) Rect {
	return Rect{
		X:      rect.X,
		Y:      rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
}

func scrollBarDetailsFromCore(
	details *expletives.ScrollBarDetails,
) *ScrollBarDetails {
	if details == nil {
		return nil
	}
	return &ScrollBarDetails{
		Orientation:    Orientation(details.Orientation),
		ContentSize:    details.ContentSize,
		ViewportSize:   details.ViewportSize,
		Offset:         details.Offset,
		MaximumOffset:  details.MaximumOffset,
		ArrowStep:      details.ArrowStep,
		PageStep:       details.PageStep,
		TrackStart:     details.TrackStart,
		TrackSize:      details.TrackSize,
		ThumbStart:     details.ThumbStart,
		ThumbSize:      details.ThumbSize,
		Enabled:        details.Enabled,
		DisabledReason: details.DisabledReason,
		ChangeCommand:  string(details.ChangeCommand),
	}
}
