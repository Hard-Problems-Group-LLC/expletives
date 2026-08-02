package expletives

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	// CommandFilePickerOpen resolves or accepts the current picker target.
	CommandFilePickerOpen CommandID = "file_picker.open"
	// CommandFilePickerSelect accepts the current directory.
	CommandFilePickerSelect CommandID = "file_picker.select"
	// CommandFilePickerUp navigates to the provider-supplied parent.
	CommandFilePickerUp CommandID = "file_picker.up"
	// CommandFilePickerRefresh reloads the current directory.
	CommandFilePickerRefresh CommandID = "file_picker.refresh"
)

type filePickerMode string

const (
	filePickerSingle    filePickerMode = "single"
	filePickerMultiple  filePickerMode = "multiple"
	filePickerDirectory filePickerMode = "directory"
)

// FilePickerDialogOptions configures each bounded picker specialization.
type FilePickerDialogOptions struct {
	DialogOptions
	Provider         FilePickerProvider
	InitialDirectory string
	Filters          []FilePickerFilter
	Filter           string
	Sort             FilePickerSort
}

// FilePickerState is one copied semantic picker observation. Opaque provider
// locations are deliberately absent.
type FilePickerState struct {
	Mode           string
	Status         CollectionStatus
	DisplayPath    string
	EntryCount     int
	FileCount      int
	DirectoryCount int
	CurrentName    string
	CurrentKind    FilePickerEntryKind
	SelectedCount  int
	Filter         string
	Sort           FilePickerSort
	Error          string
}

// FilePickerDialog is a copy-safe single-file standard Dialog compound.
type FilePickerDialog struct {
	Dialog
	core *filePickerCore
}

// MultiFilePickerDialog is a copy-safe explicitly committed multi-file
// standard Dialog compound.
type MultiFilePickerDialog struct {
	Dialog
	core *filePickerCore
}

// DirectoryPickerDialog is a copy-safe current-directory standard Dialog
// compound.
type DirectoryPickerDialog struct {
	Dialog
	core *filePickerCore
}

type normalizedFilePickerFilter struct {
	value    FilePickerFilter
	label    normalizedDisplayText
	patterns []string
}

type filePickerCore struct {
	dialog   *Dialog
	provider FilePickerProvider
	mode     filePickerMode
	gate     chan struct{}

	mu       sync.RWMutex
	listing  normalizedFilePickerListing
	filters  []normalizedFilePickerFilter
	filter   string
	sort     FilePickerSort
	status   CollectionStatus
	errText  string
	accepted []FilePickerEntry

	pathPanel     *Panel
	pathLabel     *Label
	pathField     *TextField
	mainPanel     *Panel
	listPanel     *Panel
	listLabel     *Label
	list          *ListBox
	buttonPanel   *Panel
	open          *Button
	up            *Button
	refreshButton *Button
	cancel        *Button
	information   *StaticText
	entriesByKey  map[string]FilePickerEntry
}

type filePickerCompound interface {
	filePickerCore() *filePickerCore
}

// NewFilePickerDialog performs one explicit initial provider read and then
// atomically constructs an inactive single-file picker.
func NewFilePickerDialog(
	ctx context.Context,
	parent Container,
	options FilePickerDialogOptions,
) (*FilePickerDialog, error) {
	var result *FilePickerDialog
	compound, err := newFilePickerCompound(ctx, parent, options, filePickerSingle, func(core *filePickerCore) {
		result = &FilePickerDialog{Dialog: *core.dialog, core: core}
		core.dialog.state.control = result
		core.dialog.state.container = result
	})
	if err != nil {
		return nil, err
	}
	_ = compound
	return result, nil
}

// NewMultiFilePickerDialog performs one explicit initial provider read and
// atomically constructs an inactive explicit multi-file picker.
func NewMultiFilePickerDialog(
	ctx context.Context,
	parent Container,
	options FilePickerDialogOptions,
) (*MultiFilePickerDialog, error) {
	var result *MultiFilePickerDialog
	compound, err := newFilePickerCompound(ctx, parent, options, filePickerMultiple, func(core *filePickerCore) {
		result = &MultiFilePickerDialog{Dialog: *core.dialog, core: core}
		core.dialog.state.control = result
		core.dialog.state.container = result
	})
	if err != nil {
		return nil, err
	}
	_ = compound
	return result, nil
}

// NewDirectoryPickerDialog performs one explicit initial provider read and
// atomically constructs an inactive current-directory picker.
func NewDirectoryPickerDialog(
	ctx context.Context,
	parent Container,
	options FilePickerDialogOptions,
) (*DirectoryPickerDialog, error) {
	var result *DirectoryPickerDialog
	compound, err := newFilePickerCompound(ctx, parent, options, filePickerDirectory, func(core *filePickerCore) {
		result = &DirectoryPickerDialog{Dialog: *core.dialog, core: core}
		core.dialog.state.control = result
		core.dialog.state.container = result
	})
	if err != nil {
		return nil, err
	}
	_ = compound
	return result, nil
}

func newFilePickerCompound(
	ctx context.Context,
	parent Container,
	options FilePickerDialogOptions,
	mode filePickerMode,
	attach func(*filePickerCore),
) (*filePickerCore, error) {
	if ctx == nil {
		return nil, errors.New("expletives: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.Provider == nil || !validFilePickerLocation(options.InitialDirectory) {
		return nil, fmt.Errorf("%w: picker provider and initial directory are required", ErrValidation)
	}
	filters, activeFilter, err := normalizeFilePickerFilters(options.Filters, options.Filter)
	if err != nil {
		return nil, err
	}
	sortPolicy, err := normalizeFilePickerSort(options.Sort)
	if err != nil {
		return nil, err
	}

	listing, listErr := options.Provider.List(ctx, options.InitialDirectory)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var normalized normalizedFilePickerListing
	if listErr == nil {
		normalized, err = normalizeFilePickerListing(listing)
		if err != nil {
			return nil, fmt.Errorf("file-picker provider contract: %w", err)
		}
	} else {
		display := localPickerDisplay(options.InitialDirectory)
		input, inputErr := normalizeInputText(display)
		if inputErr != nil || input.text == "" {
			display = "[unavailable]"
		}
		normalized, err = normalizeFilePickerListing(FilePickerListing{
			Directory: options.InitialDirectory, DisplayPath: display,
		})
		if err != nil {
			return nil, err
		}
	}

	transaction, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	if options.Bounds.Width == 0 {
		options.Bounds.Width = 49
	}
	if options.Bounds.Height == 0 {
		options.Bounds.Height = 19
	}
	kind, defaultTitle := filePickerKindAndTitle(mode)
	if options.Title == "" {
		options.Title = defaultTitle
	}
	dialog, err := transaction.newDialog(parent, options.DialogOptions, kind)
	if err != nil {
		return nil, err
	}
	setProvisionalModalEscapeCommand(dialog.state, CommandDialogCancel)
	core := &filePickerCore{
		dialog: dialog, provider: options.Provider, mode: mode,
		gate: make(chan struct{}, 1), listing: normalized,
		filters: filters, filter: activeFilter, sort: sortPolicy,
		status: CollectionReady,
	}
	if listErr != nil {
		core.status = CollectionError
		core.errText = boundedFilePickerError(listErr)
	}
	if err := core.build(transaction); err != nil {
		return nil, err
	}
	attach(core)
	if err := transaction.Commit(ctx); err != nil {
		return nil, err
	}
	return core, nil
}

func filePickerKindAndTitle(mode filePickerMode) (ControlKind, string) {
	switch mode {
	case filePickerMultiple:
		return ControlMultiFilePickerDialog, "Open Files"
	case filePickerDirectory:
		return ControlDirectoryPickerDialog, "Select Directory"
	default:
		return ControlFilePickerDialog, "Open File"
	}
}

func (p *filePickerCore) build(transaction *Transaction) error {
	style := StyleID(p.dialog.state.kind)
	view := p.deriveView(nil, nil)
	pathPanel, err := transaction.NewPanel(p.dialog, PanelOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "path-group"),
		Style:         style,
	})
	if err != nil {
		return err
	}
	pathField, err := transaction.NewTextField(pathPanel, TextFieldOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "path"),
		},
		Text: p.listing.path.text, ChangeCommand: CommandFilePickerOpen,
	})
	if err != nil {
		return err
	}
	pathText, pathMnemonic := "File name", Key("f")
	if p.mode == filePickerDirectory {
		pathText, pathMnemonic = "Directory name", Key("n")
	}
	pathLabel, err := transaction.NewLabel(pathPanel, LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "path-label"),
			Style:         style,
		},
		Text: pathText, Target: pathField, Mnemonic: pathMnemonic,
	})
	if err != nil {
		return err
	}
	mainPanel, err := transaction.NewPanel(p.dialog, PanelOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "main"), Style: style,
	})
	if err != nil {
		return err
	}
	listPanel, err := transaction.NewPanel(mainPanel, PanelOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "list-group"), Style: style,
	})
	if err != nil {
		return err
	}
	listOptions := ListBoxOptions{
		ScrollablePanelOptions: ScrollablePanelOptions{
			ScrollViewOptions: ScrollViewOptions{
				PanelOptions: PanelOptions{
					AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "list"),
				},
			},
			BorderForm: BorderSingle,
		},
		Items: view.items, Current: view.current, Selected: view.selected,
		SelectionMode:  CollectionSelectionSingle,
		SelectionMarks: CollectionSelectionMarksHide,
		Status:         p.status, StatusMessage: p.errText,
		ActivateCommand: CommandFilePickerOpen,
	}
	if p.mode == filePickerMultiple {
		listOptions.SelectionMode = CollectionSelectionMultiple
		listOptions.SelectionMarks = CollectionSelectionMarksShow
	}
	list, err := transaction.NewListBox(listPanel, listOptions)
	if err != nil {
		return err
	}
	listText := "Files"
	if p.mode == filePickerDirectory {
		listText = "Directories"
	}
	listLabel, err := transaction.NewLabel(listPanel, LabelOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "list-label"), Style: style,
		},
		Text: listText, Target: list, Mnemonic: "l",
	})
	if err != nil {
		return err
	}
	buttonPanel, err := transaction.NewPanel(mainPanel, PanelOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "buttons"), Style: style,
	})
	if err != nil {
		return err
	}
	openCommand, openMnemonic := CommandFilePickerOpen, Key("o")
	if p.mode == filePickerDirectory {
		openCommand, openMnemonic = CommandFilePickerSelect, Key("s")
	}
	open, err := transaction.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "open")},
		Command:      openCommand, Mnemonic: openMnemonic, Default: true,
	})
	if err != nil {
		return err
	}
	up, err := transaction.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "up")},
		Command:      CommandFilePickerUp, Mnemonic: "u",
	})
	if err != nil {
		return err
	}
	refresh, err := transaction.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "refresh")},
		Command:      CommandFilePickerRefresh, Mnemonic: "r",
	})
	if err != nil {
		return err
	}
	cancel, err := transaction.NewButton(buttonPanel, ButtonOptions{
		PanelOptions: PanelOptions{AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "cancel")},
		Command:      CommandDialogCancel, Mnemonic: "c", Cancel: true,
	})
	if err != nil {
		return err
	}
	information, err := transaction.NewStaticText(p.dialog, StaticTextOptions{
		PanelOptions: PanelOptions{
			AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "information"), Style: style,
		},
		Text: p.initialInformation(view), Wrap: TextWrapWords,
	})
	if err != nil {
		return err
	}
	p.pathPanel, p.pathLabel, p.pathField = pathPanel, pathLabel, pathField
	p.mainPanel, p.listPanel, p.listLabel, p.list = mainPanel, listPanel, listLabel, list
	p.buttonPanel, p.open, p.up = buttonPanel, open, up
	p.refreshButton, p.cancel, p.information = refresh, cancel, information
	p.entriesByKey = view.entriesByKey
	return p.attachLayouts(transaction)
}

func (p *filePickerCore) attachLayouts(transaction *Transaction) error {
	pathLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "path-layout"),
	})
	if err != nil {
		return err
	}
	if err := pathLayout.AddPanel(p.pathLabel, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := pathLayout.AddPanel(p.pathField, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := transaction.SetLayout(p.pathPanel, pathLayout); err != nil {
		return err
	}
	listLayout, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "list-layout"),
	})
	if err != nil {
		return err
	}
	if err := listLayout.AddPanel(p.listLabel, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := listLayout.AddPanel(p.list, LayoutItemOptions{Grow: 1}); err != nil {
		return err
	}
	if err := transaction.SetLayout(p.listPanel, listLayout); err != nil {
		return err
	}
	buttons, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "button-layout"), Gap: 1,
	})
	if err != nil {
		return err
	}
	for _, button := range []*Button{p.open, p.up, p.refreshButton, p.cancel} {
		if err := buttons.AddPanel(button, LayoutItemOptions{}); err != nil {
			return err
		}
	}
	if err := transaction.SetLayout(p.buttonPanel, buttons); err != nil {
		return err
	}
	main, err := NewBoxLayout(Horizontal, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "main-layout"), Gap: 1,
	})
	if err != nil {
		return err
	}
	if err := main.AddPanel(p.listPanel, LayoutItemOptions{Grow: 1}); err != nil {
		return err
	}
	if err := main.AddPanel(p.buttonPanel, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := transaction.SetLayout(p.mainPanel, main); err != nil {
		return err
	}
	root, err := NewBoxLayout(Vertical, BoxLayoutOptions{
		AutomationKey: derivedCompoundKey(p.dialog.AutomationKey(), "layout"),
		Insets:        Insets{Top: 1, Right: 1, Bottom: 1, Left: 1}, Gap: 1,
	})
	if err != nil {
		return err
	}
	if err := root.AddPanel(p.pathPanel, LayoutItemOptions{}); err != nil {
		return err
	}
	if err := root.AddPanel(p.mainPanel, LayoutItemOptions{Grow: 1}); err != nil {
		return err
	}
	if err := root.AddPanel(p.information, LayoutItemOptions{}); err != nil {
		return err
	}
	return transaction.SetLayout(p.dialog, root)
}

type filePickerView struct {
	items        []ListItem
	current      string
	selected     []string
	entriesByKey map[string]FilePickerEntry
}

func (p *filePickerCore) deriveView(
	currentLocations []string,
	selectedLocations []string,
) filePickerView {
	entries := append([]normalizedFilePickerEntry(nil), p.listing.entries...)
	filter := p.activeFilter()
	visible := entries[:0]
	for _, entry := range entries {
		if p.mode == filePickerDirectory && entry.entry.Kind != FilePickerDirectory {
			continue
		}
		if entry.entry.Kind == FilePickerDirectory || filePickerMatches(entry.entry.Name, filter.patterns) {
			visible = append(visible, entry)
		}
	}
	sort.SliceStable(visible, func(left, right int) bool {
		return filePickerEntryLess(visible[left].entry, visible[right].entry, p.sort)
	})
	currentSet := make(map[string]bool, len(currentLocations))
	for _, location := range currentLocations {
		currentSet[location] = true
	}
	selectedSet := make(map[string]bool, len(selectedLocations))
	for _, location := range selectedLocations {
		selectedSet[location] = true
	}
	view := filePickerView{
		items:        make([]ListItem, 0, len(visible)),
		entriesByKey: make(map[string]FilePickerEntry, len(visible)),
	}
	for _, entry := range visible {
		key := filePickerEntryKey(entry.entry.Location)
		label := entry.entry.Name
		if entry.entry.Kind == FilePickerDirectory {
			label += "/"
		}
		disabled := entry.entry.Kind == FilePickerOther
		view.items = append(view.items, ListItem{
			Key: key, Label: label, Disabled: disabled,
			DisabledReason: map[bool]string{true: "Unsupported entry type"}[disabled],
		})
		view.entriesByKey[key] = entry.entry
		if view.current == "" && currentSet[entry.entry.Location] && !disabled {
			view.current = key
		}
		if selectedSet[entry.entry.Location] && !disabled && entry.entry.Kind == FilePickerFile {
			view.selected = append(view.selected, key)
		}
	}
	return view
}

func (p *filePickerCore) activeFilter() normalizedFilePickerFilter {
	for _, filter := range p.filters {
		if filter.value.Key == p.filter {
			return filter
		}
	}
	return p.filters[0]
}

func filePickerEntryKey(location string) string {
	digest := sha256.Sum256([]byte(location))
	return "fp:" + hex.EncodeToString(digest[:24])
}

func filePickerMatches(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := path.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

func filePickerEntryLess(left, right FilePickerEntry, policy FilePickerSort) bool {
	leftDirectory, rightDirectory := left.Kind == FilePickerDirectory, right.Kind == FilePickerDirectory
	if leftDirectory != rightDirectory {
		return leftDirectory
	}
	comparison := 0
	switch policy.Field {
	case FilePickerSortSize:
		if left.Size < right.Size {
			comparison = -1
		} else if left.Size > right.Size {
			comparison = 1
		}
	case FilePickerSortModified:
		comparison = left.Modified.Compare(right.Modified)
	default:
		comparison = strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	}
	if comparison == 0 {
		comparison = strings.Compare(left.Name, right.Name)
	}
	if comparison == 0 {
		comparison = strings.Compare(left.Location, right.Location)
	}
	if policy.Direction == SortDescending {
		return comparison > 0
	}
	return comparison < 0
}

func normalizeFilePickerFilters(
	filters []FilePickerFilter,
	active string,
) ([]normalizedFilePickerFilter, string, error) {
	if len(filters) == 0 {
		filters = []FilePickerFilter{{Key: "all", Label: "All files", Patterns: []string{"*"}}}
	}
	if len(filters) > MaxFilePickerFilters {
		return nil, "", fmt.Errorf("%w: too many file-picker filters", ErrControlCapacity)
	}
	result := make([]normalizedFilePickerFilter, len(filters))
	seen := make(map[string]bool, len(filters))
	patternCount, retained := 0, 0
	for index, filter := range filters {
		if !validBoundedIdentifier(filter.Key) || seen[filter.Key] || len(filter.Patterns) == 0 {
			return nil, "", fmt.Errorf("%w: invalid file-picker filter", ErrValidation)
		}
		seen[filter.Key] = true
		label, err := normalizeDisplayText(filter.Label, false)
		if err != nil || label.cells == 0 {
			return nil, "", fmt.Errorf("%w: invalid file-picker filter label", ErrValidation)
		}
		patterns := append([]string(nil), filter.Patterns...)
		for _, pattern := range patterns {
			if pattern == "" || len(pattern) > MaxDisplayTextBytes {
				return nil, "", fmt.Errorf("%w: invalid file-picker pattern", ErrValidation)
			}
			if _, err := path.Match(pattern, ""); err != nil {
				return nil, "", fmt.Errorf("%w: invalid file-picker pattern", ErrValidation)
			}
			patternCount++
			retained += len(pattern)
		}
		retained += len(filter.Key) + len(label.text)
		filter.Label = label.text
		filter.Patterns = append([]string(nil), patterns...)
		result[index] = normalizedFilePickerFilter{value: filter, label: label, patterns: patterns}
	}
	if patternCount > MaxFilePickerPatterns || retained > MaxCollectionAggregateBytes {
		return nil, "", fmt.Errorf("%w: file-picker filters exceed capacity", ErrControlCapacity)
	}
	if active == "" {
		active = result[0].value.Key
	}
	if !seen[active] {
		return nil, "", fmt.Errorf("%w: unknown file-picker filter", ErrValidation)
	}
	return result, active, nil
}

func normalizeFilePickerSort(policy FilePickerSort) (FilePickerSort, error) {
	if policy.Field == "" {
		policy.Field = FilePickerSortName
	}
	if policy.Direction == "" || policy.Direction == SortNone {
		policy.Direction = SortAscending
	}
	if policy.Field != FilePickerSortName && policy.Field != FilePickerSortSize &&
		policy.Field != FilePickerSortModified {
		return FilePickerSort{}, fmt.Errorf("%w: invalid file-picker sort field", ErrValidation)
	}
	if policy.Direction != SortAscending && policy.Direction != SortDescending {
		return FilePickerSort{}, fmt.Errorf("%w: invalid file-picker sort direction", ErrValidation)
	}
	return policy, nil
}

func boundedFilePickerError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.ToValidUTF8(err.Error(), "\uFFFD")
	for len(text) > MaxDisplayTextBytes {
		_, width := utf8.DecodeLastRuneInString(text)
		text = text[:len(text)-width]
	}
	if text == "" {
		return "Unable to read this location"
	}
	if normalized, normalizeErr := normalizeDisplayText(text, false); normalizeErr == nil {
		return normalized.text
	}
	return "Unable to read this location"
}

func (p *filePickerCore) initialInformation(view filePickerView) string {
	if p.errText != "" {
		return p.errText
	}
	if view.current != "" {
		return filePickerEntryInformation(view.entriesByKey[view.current])
	}
	return "No matching entries"
}

func filePickerViewInformation(view filePickerView) string {
	if view.current != "" {
		return filePickerEntryInformation(view.entriesByKey[view.current])
	}
	return "No matching entries"
}

func filePickerEntryInformation(entry FilePickerEntry) string {
	switch entry.Kind {
	case FilePickerDirectory:
		return "Directory: " + entry.Name
	case FilePickerFile:
		return fmt.Sprintf("%s  %d bytes  %s", entry.Name, entry.Size, entry.Modified.UTC().Format("2006-01-02 15:04Z"))
	default:
		return "Unsupported: " + entry.Name
	}
}

func (p *filePickerCore) acquire(ctx context.Context) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	select {
	case p.gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *filePickerCore) release() { <-p.gate }

func (p *filePickerCore) currentLocations() ([]string, []string) {
	state := p.list.State()
	p.mu.RLock()
	defer p.mu.RUnlock()
	current := []string{}
	if entry, found := p.entriesByKey[state.Current]; found {
		current = append(current, entry.Location)
	}
	selected := make([]string, 0, len(state.Selected))
	for _, key := range state.Selected {
		if entry, found := p.entriesByKey[key]; found {
			selected = append(selected, entry.Location)
		}
	}
	return current, selected
}

func (p *filePickerCore) navigate(ctx context.Context, directory string) error {
	if err := p.acquire(ctx); err != nil {
		return err
	}
	defer p.release()
	listing, err := p.provider.List(ctx, directory)
	if err != nil {
		return p.publishError(ctx, err)
	}
	normalized, err := normalizeFilePickerListing(listing)
	if err != nil {
		return fmt.Errorf("file-picker provider contract: %w", err)
	}
	return p.publishListing(ctx, normalized, nil, nil)
}

func (p *filePickerCore) publishListing(
	ctx context.Context,
	listing normalizedFilePickerListing,
	current []string,
	selected []string,
) error {
	p.mu.Lock()
	previous := p.listing
	p.listing = listing
	view := p.deriveView(current, selected)
	p.listing = previous
	p.mu.Unlock()
	transaction := p.dialog.state.app.NewTransaction()
	if err := transaction.SetText(p.pathField, listing.path.text); err != nil {
		return err
	}
	if err := transaction.SetListStatus(p.list, CollectionReady, ""); err != nil {
		return err
	}
	if err := transaction.ReplaceList(p.list, view.items, view.current, view.selected); err != nil {
		return err
	}
	if err := transaction.SetText(p.information, filePickerViewInformation(view)); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	p.mu.Lock()
	p.listing = listing
	p.status, p.errText, p.entriesByKey = CollectionReady, "", view.entriesByKey
	p.mu.Unlock()
	return nil
}

func (p *filePickerCore) publishError(ctx context.Context, providerErr error) error {
	message := boundedFilePickerError(providerErr)
	transaction := p.dialog.state.app.NewTransaction()
	if err := transaction.SetListStatus(p.list, CollectionError, message); err != nil {
		return err
	}
	if err := transaction.SetText(p.information, message); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	p.mu.Lock()
	p.status, p.errText = CollectionError, message
	p.mu.Unlock()
	return providerErr
}

func (p *filePickerCore) handle(
	ctx context.Context,
	command Command,
) CommandResult {
	if p == nil || p.dialog == nil || !p.dialog.Active() {
		return publicResult(OutcomeRejected, "picker_inactive", "file picker is not active", ErrModalState)
	}
	switch command.ID {
	case CommandFilePickerUp:
		p.mu.RLock()
		parent := p.listing.listing.Parent
		p.mu.RUnlock()
		if parent == "" {
			return CommandResult{Outcome: OutcomeNoOp}
		}
		if err := p.navigate(ctx, parent); err != nil {
			return publicResult(OutcomeFailed, "provider_error", boundedFilePickerError(err), err)
		}
		return CommandResult{Outcome: OutcomeApplied}
	case CommandFilePickerRefresh:
		p.mu.RLock()
		directory := p.listing.listing.Directory
		p.mu.RUnlock()
		current, selected := p.currentLocations()
		if err := p.acquire(ctx); err != nil {
			return contextCommandResult(err)
		}
		listing, err := p.provider.List(ctx, directory)
		if err == nil {
			var normalized normalizedFilePickerListing
			normalized, err = normalizeFilePickerListing(listing)
			if err == nil {
				err = p.publishListing(ctx, normalized, current, selected)
			}
		} else {
			err = p.publishError(ctx, err)
		}
		p.release()
		if err != nil {
			return publicResult(OutcomeFailed, "provider_error", boundedFilePickerError(err), err)
		}
		return CommandResult{Outcome: OutcomeApplied}
	case CommandFilePickerOpen, CommandFilePickerSelect:
		return p.handleOpen(ctx, command.ID, command.Target)
	default:
		return publicResult(OutcomeRejected, "picker_command", "unknown file-picker command", ErrInvalidRequest)
	}
}

func (p *filePickerCore) handleOpen(
	ctx context.Context,
	action CommandID,
	target ControlID,
) CommandResult {
	if target == p.pathField.ID() {
		return p.resolveTyped(ctx, p.pathField.Text())
	}
	if target == p.open.ID() {
		switch p.mode {
		case filePickerMultiple:
			entries := p.selectedEntries()
			if len(entries) == 0 {
				return publicResult(OutcomeRejected, "selection_required", "select at least one file", ErrValidation)
			}
			return p.accept(action, entries)
		case filePickerDirectory:
			p.mu.RLock()
			entry := FilePickerEntry{
				Name: p.listing.path.text, Location: p.listing.listing.Directory,
				Kind: FilePickerDirectory,
			}
			p.mu.RUnlock()
			return p.accept(action, []FilePickerEntry{entry})
		}
	}
	entry, found := p.currentEntry()
	if !found {
		return publicResult(OutcomeRejected, "selection_required", "choose an entry", ErrValidation)
	}
	switch entry.Kind {
	case FilePickerDirectory:
		if err := p.navigate(ctx, entry.Location); err != nil {
			return publicResult(OutcomeFailed, "provider_error", boundedFilePickerError(err), err)
		}
		return CommandResult{Outcome: OutcomeApplied}
	case FilePickerFile:
		if p.mode == filePickerDirectory {
			return publicResult(OutcomeRejected, "directory_required", "choose a directory", ErrValidation)
		}
		if p.mode == filePickerMultiple {
			return CommandResult{Outcome: OutcomeApplied}
		}
		return p.accept(CommandFilePickerOpen, []FilePickerEntry{entry})
	default:
		return publicResult(OutcomeRejected, "entry_unsupported", "entry cannot be opened", ErrValidation)
	}
}

func (p *filePickerCore) resolveTyped(ctx context.Context, input string) CommandResult {
	p.mu.RLock()
	directory := p.listing.listing.Directory
	p.mu.RUnlock()
	entry, err := p.provider.Resolve(ctx, directory, input)
	if err != nil {
		_ = p.publishError(ctx, err)
		return publicResult(OutcomeFailed, "provider_error", boundedFilePickerError(err), err)
	}
	entry, err = normalizeFilePickerResolvedEntry(entry)
	if err != nil {
		return publicResult(OutcomeFailed, "provider_contract", "provider returned an invalid entry", err)
	}
	if entry.Kind == FilePickerDirectory {
		if err := p.navigate(ctx, entry.Location); err != nil {
			return publicResult(OutcomeFailed, "provider_error", boundedFilePickerError(err), err)
		}
		return CommandResult{Outcome: OutcomeApplied}
	}
	if entry.Kind != FilePickerFile || p.mode == filePickerDirectory {
		return publicResult(OutcomeRejected, "entry_unsupported", "entry cannot be selected", ErrValidation)
	}
	if p.mode == filePickerMultiple {
		return publicResult(OutcomeRejected, "explicit_commit", "select files from the list and use Open", ErrValidation)
	}
	return p.accept(CommandFilePickerOpen, []FilePickerEntry{entry})
}

func (p *filePickerCore) currentEntry() (FilePickerEntry, bool) {
	state := p.list.State()
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, found := p.entriesByKey[state.Current]
	return cloneFilePickerEntry(entry), found
}

func (p *filePickerCore) selectedEntries() []FilePickerEntry {
	state := p.list.State()
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]FilePickerEntry, 0, len(state.Selected))
	for _, key := range state.Selected {
		entry, found := p.entriesByKey[key]
		if found && entry.Kind == FilePickerFile {
			result = append(result, cloneFilePickerEntry(entry))
		}
	}
	return result
}

func (p *filePickerCore) accept(
	action CommandID,
	entries []FilePickerEntry,
) CommandResult {
	p.mu.Lock()
	p.accepted = append([]FilePickerEntry(nil), entries...)
	p.mu.Unlock()
	if err := p.dialog.Close(ModalResult{Reason: ModalAccepted, Action: action}); err != nil {
		return publicResult(OutcomeFailed, "picker_close", "file picker could not close", err)
	}
	return CommandResult{Outcome: OutcomeApplied}
}

func (p *filePickerCore) state() FilePickerState {
	if p == nil || p.list == nil {
		return FilePickerState{}
	}
	listState := p.list.State()
	p.mu.RLock()
	defer p.mu.RUnlock()
	state := FilePickerState{
		Mode: string(p.mode), Status: p.status, DisplayPath: p.listing.path.text,
		EntryCount: len(p.listing.entries), SelectedCount: len(listState.Selected),
		Filter: p.filter, Sort: p.sort, Error: p.errText,
	}
	for _, entry := range p.listing.entries {
		if entry.entry.Kind == FilePickerDirectory {
			state.DirectoryCount++
		} else if entry.entry.Kind == FilePickerFile {
			state.FileCount++
		}
	}
	if entry, found := p.entriesByKey[listState.Current]; found {
		state.CurrentName, state.CurrentKind = entry.Name, entry.Kind
	}
	return state
}

func (p *filePickerCore) refresh(ctx context.Context) error {
	result := p.handle(ctx, Command{ID: CommandFilePickerRefresh, Target: p.dialog.ID(), Source: "api"})
	return commandResultError(result)
}

func (p *filePickerCore) setFilter(ctx context.Context, key string) error {
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if err := p.acquire(ctx); err != nil {
		return err
	}
	defer p.release()
	current, selected := p.currentLocations()
	p.mu.Lock()
	found := false
	for _, filter := range p.filters {
		if filter.value.Key == key {
			found = true
			break
		}
	}
	if !found {
		p.mu.Unlock()
		return fmt.Errorf("%w: unknown file-picker filter", ErrValidation)
	}
	p.filter = key
	view := p.deriveView(current, selected)
	p.mu.Unlock()
	transaction := p.dialog.state.app.NewTransaction()
	if err := transaction.SetListStatus(p.list, CollectionReady, ""); err != nil {
		return err
	}
	if err := transaction.ReplaceList(p.list, view.items, view.current, view.selected); err != nil {
		return err
	}
	if err := transaction.SetText(p.information, filePickerViewInformation(view)); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	p.mu.Lock()
	p.entriesByKey, p.status, p.errText = view.entriesByKey, CollectionReady, ""
	p.mu.Unlock()
	return nil
}

func (p *filePickerCore) setSort(ctx context.Context, policy FilePickerSort) error {
	policy, err := normalizeFilePickerSort(policy)
	if err != nil {
		return err
	}
	if ctx == nil {
		return errors.New("expletives: nil context")
	}
	if err := p.acquire(ctx); err != nil {
		return err
	}
	defer p.release()
	current, selected := p.currentLocations()
	p.mu.Lock()
	p.sort = policy
	view := p.deriveView(current, selected)
	p.mu.Unlock()
	if err := p.list.Replace(view.items, view.current, view.selected); err != nil {
		return err
	}
	p.mu.Lock()
	p.entriesByKey = view.entriesByKey
	p.mu.Unlock()
	return nil
}

func commandResultError(result CommandResult) error {
	if result.Outcome == OutcomeApplied || result.Outcome == OutcomeNoOp {
		return nil
	}
	if result.Cause != nil {
		return result.Cause
	}
	return fmt.Errorf("%w: %s", ErrInvalidRequest, result.Message)
}

func (d *FilePickerDialog) filePickerCore() *filePickerCore {
	if d == nil {
		return nil
	}
	return d.core
}
func (d *MultiFilePickerDialog) filePickerCore() *filePickerCore {
	if d == nil {
		return nil
	}
	return d.core
}
func (d *DirectoryPickerDialog) filePickerCore() *filePickerCore {
	if d == nil {
		return nil
	}
	return d.core
}

// Show presents a picker with the file list initially focused when possible.
func (d *FilePickerDialog) Show(initialFocus Control) error {
	return showFilePicker(d.core, initialFocus)
}
func (d *MultiFilePickerDialog) Show(initialFocus Control) error {
	return showFilePicker(d.core, initialFocus)
}
func (d *DirectoryPickerDialog) Show(initialFocus Control) error {
	return showFilePicker(d.core, initialFocus)
}

func showFilePicker(core *filePickerCore, initialFocus Control) error {
	if core == nil {
		return ErrInvalidControl
	}
	if initialFocus == nil {
		if core.list.State().Current != "" {
			initialFocus = core.list
		} else {
			initialFocus = core.pathField
		}
	}
	return core.dialog.Show(initialFocus)
}

// State returns one copied semantic state without provider location tokens.
func (d *FilePickerDialog) State() FilePickerState      { return d.core.state() }
func (d *MultiFilePickerDialog) State() FilePickerState { return d.core.state() }
func (d *DirectoryPickerDialog) State() FilePickerState { return d.core.state() }

// Refresh explicitly reloads the current directory.
func (d *FilePickerDialog) Refresh(ctx context.Context) error      { return d.core.refresh(ctx) }
func (d *MultiFilePickerDialog) Refresh(ctx context.Context) error { return d.core.refresh(ctx) }
func (d *DirectoryPickerDialog) Refresh(ctx context.Context) error { return d.core.refresh(ctx) }

// Navigate explicitly loads one opaque provider directory token.
func (d *FilePickerDialog) Navigate(ctx context.Context, location string) error {
	return d.core.navigate(ctx, location)
}
func (d *MultiFilePickerDialog) Navigate(ctx context.Context, location string) error {
	return d.core.navigate(ctx, location)
}
func (d *DirectoryPickerDialog) Navigate(ctx context.Context, location string) error {
	return d.core.navigate(ctx, location)
}

// SetFilter changes the active copied filter without provider I/O.
func (d *FilePickerDialog) SetFilter(ctx context.Context, key string) error {
	return d.core.setFilter(ctx, key)
}
func (d *MultiFilePickerDialog) SetFilter(ctx context.Context, key string) error {
	return d.core.setFilter(ctx, key)
}
func (d *DirectoryPickerDialog) SetFilter(ctx context.Context, key string) error {
	return d.core.setFilter(ctx, key)
}

// SetSort changes the stable derived entry order without provider I/O.
func (d *FilePickerDialog) SetSort(ctx context.Context, policy FilePickerSort) error {
	return d.core.setSort(ctx, policy)
}
func (d *MultiFilePickerDialog) SetSort(ctx context.Context, policy FilePickerSort) error {
	return d.core.setSort(ctx, policy)
}
func (d *DirectoryPickerDialog) SetSort(ctx context.Context, policy FilePickerSort) error {
	return d.core.setSort(ctx, policy)
}

// Selection returns the accepted single file.
func (d *FilePickerDialog) Selection() (FilePickerEntry, bool) {
	return singleFilePickerSelection(d.core)
}

// Selections returns the explicitly accepted files in display order.
func (d *MultiFilePickerDialog) Selections() ([]FilePickerEntry, bool) {
	if d == nil || d.core == nil {
		return nil, false
	}
	result, ready := d.Result()
	if !ready || result.Reason != ModalAccepted || result.Action != CommandFilePickerOpen {
		return nil, false
	}
	d.core.mu.RLock()
	defer d.core.mu.RUnlock()
	if len(d.core.accepted) == 0 {
		return nil, false
	}
	return append([]FilePickerEntry(nil), d.core.accepted...), true
}

// Selection returns the accepted current directory entry.
func (d *DirectoryPickerDialog) Selection() (FilePickerEntry, bool) {
	return singleFilePickerSelection(d.core)
}

func singleFilePickerSelection(core *filePickerCore) (FilePickerEntry, bool) {
	if core == nil {
		return FilePickerEntry{}, false
	}
	result, ready := core.dialog.Result()
	if !ready || result.Reason != ModalAccepted ||
		(result.Action != CommandFilePickerOpen && result.Action != CommandFilePickerSelect) {
		return FilePickerEntry{}, false
	}
	core.mu.RLock()
	defer core.mu.RUnlock()
	if len(core.accepted) != 1 {
		return FilePickerEntry{}, false
	}
	return cloneFilePickerEntry(core.accepted[0]), true
}

// Canonical child accessors support ordinary composition and tests.
func (d *FilePickerDialog) PathField() *TextField       { return d.core.pathField }
func (d *MultiFilePickerDialog) PathField() *TextField  { return d.core.pathField }
func (d *DirectoryPickerDialog) PathField() *TextField  { return d.core.pathField }
func (d *FilePickerDialog) List() *ListBox              { return d.core.list }
func (d *MultiFilePickerDialog) List() *ListBox         { return d.core.list }
func (d *DirectoryPickerDialog) List() *ListBox         { return d.core.list }
func (d *FilePickerDialog) OpenButton() *Button         { return d.core.open }
func (d *MultiFilePickerDialog) OpenButton() *Button    { return d.core.open }
func (d *DirectoryPickerDialog) SelectButton() *Button  { return d.core.open }
func (d *FilePickerDialog) UpButton() *Button           { return d.core.up }
func (d *MultiFilePickerDialog) UpButton() *Button      { return d.core.up }
func (d *DirectoryPickerDialog) UpButton() *Button      { return d.core.up }
func (d *FilePickerDialog) RefreshButton() *Button      { return d.core.refreshButton }
func (d *MultiFilePickerDialog) RefreshButton() *Button { return d.core.refreshButton }
func (d *DirectoryPickerDialog) RefreshButton() *Button { return d.core.refreshButton }
func (d *FilePickerDialog) CancelButton() *Button       { return d.core.cancel }
func (d *MultiFilePickerDialog) CancelButton() *Button  { return d.core.cancel }
func (d *DirectoryPickerDialog) CancelButton() *Button  { return d.core.cancel }

func (a *App) routeFilePickerCommand(ctx context.Context, command Command) CommandResult {
	a.mu.RLock()
	state := a.controlsByID[command.Target]
	modal := modalAncestorState(state)
	if modal == nil {
		modal = state
	}
	var core *filePickerCore
	if modal != nil {
		if compound, ok := modal.control.(filePickerCompound); ok {
			core = compound.filePickerCore()
		}
	}
	a.mu.RUnlock()
	if core == nil {
		return publicResult(OutcomeRejected, "picker_target", "command target is not a file picker", ErrInvalidControl)
	}
	return core.handle(ctx, command)
}

func isFilePickerCommand(command CommandID) bool {
	switch command {
	case CommandFilePickerOpen, CommandFilePickerSelect, CommandFilePickerUp,
		CommandFilePickerRefresh:
		return true
	default:
		return false
	}
}
