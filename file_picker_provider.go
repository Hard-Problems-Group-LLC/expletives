package expletives

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxFilePickerLocationBytes bounds one opaque provider location token.
	MaxFilePickerLocationBytes = MaxTextInputBytes
	// MaxFilePickerFilters bounds one copied filter inventory.
	MaxFilePickerFilters = MaxSelectionOptions
	// MaxFilePickerPatterns bounds patterns retained across one filter set.
	MaxFilePickerPatterns = MaxSelectionItems
)

// FilePickerEntryKind identifies the provider-observed kind of one entry.
type FilePickerEntryKind string

const (
	FilePickerFile      FilePickerEntryKind = "file"
	FilePickerDirectory FilePickerEntryKind = "directory"
	FilePickerOther     FilePickerEntryKind = "other"
)

// FilePickerEntry is one copied provider entry. Location is an opaque token
// and Name is its bounded display label.
type FilePickerEntry struct {
	Name     string
	Location string
	Kind     FilePickerEntryKind
	Size     uint64
	Modified time.Time
}

// FilePickerListing is one complete copied directory observation.
type FilePickerListing struct {
	Directory   string
	DisplayPath string
	Parent      string
	Entries     []FilePickerEntry
}

// FilePickerProvider supplies bounded directory observations and resolves
// user-entered display paths. Implementations must observe ctx promptly.
type FilePickerProvider interface {
	List(context.Context, string) (FilePickerListing, error)
	Resolve(context.Context, string, string) (FilePickerEntry, error)
}

// FilePickerFilter is one stable copied path.Match filter.
type FilePickerFilter struct {
	Key      string
	Label    string
	Patterns []string
}

// FilePickerSortField selects the stable derived entry order.
type FilePickerSortField string

const (
	FilePickerSortName     FilePickerSortField = "name"
	FilePickerSortSize     FilePickerSortField = "size"
	FilePickerSortModified FilePickerSortField = "modified"
)

// FilePickerSort selects one field and ascending or descending order.
type FilePickerSort struct {
	Field     FilePickerSortField
	Direction SortDirection
}

type normalizedFilePickerEntry struct {
	entry FilePickerEntry
	name  normalizedDisplayText
}

type normalizedFilePickerListing struct {
	listing FilePickerListing
	path    normalizedInputText
	entries []normalizedFilePickerEntry
}

func normalizeFilePickerListing(
	listing FilePickerListing,
) (normalizedFilePickerListing, error) {
	if !validFilePickerLocation(listing.Directory) ||
		(listing.Parent != "" && !validFilePickerLocation(listing.Parent)) {
		return normalizedFilePickerListing{}, fmt.Errorf(
			"%w: invalid file-picker directory token",
			ErrValidation,
		)
	}
	displayPath, err := normalizeInputText(listing.DisplayPath)
	if err != nil || displayPath.text == "" {
		return normalizedFilePickerListing{}, fmt.Errorf(
			"%w: invalid file-picker display path",
			ErrValidation,
		)
	}
	if len(listing.Entries) > MaxCollectionItems {
		return normalizedFilePickerListing{}, fmt.Errorf(
			"%w: file-picker listing exceeds %d entries",
			ErrControlCapacity,
			MaxCollectionItems,
		)
	}
	result := normalizedFilePickerListing{
		listing: FilePickerListing{
			Directory: listing.Directory, DisplayPath: displayPath.text,
			Parent:  listing.Parent,
			Entries: make([]FilePickerEntry, len(listing.Entries)),
		},
		path:    displayPath,
		entries: make([]normalizedFilePickerEntry, len(listing.Entries)),
	}
	locations := make(map[string]bool, len(listing.Entries))
	retained := len(listing.Directory) + len(listing.Parent) + len(displayPath.text)
	for index, entry := range listing.Entries {
		if !validFilePickerLocation(entry.Location) || locations[entry.Location] {
			return normalizedFilePickerListing{}, fmt.Errorf(
				"%w: invalid or duplicate file-picker entry location",
				ErrValidation,
			)
		}
		locations[entry.Location] = true
		if entry.Kind != FilePickerFile && entry.Kind != FilePickerDirectory &&
			entry.Kind != FilePickerOther {
			return normalizedFilePickerListing{}, fmt.Errorf(
				"%w: invalid file-picker entry kind",
				ErrValidation,
			)
		}
		name, normalizeErr := normalizeDisplayText(entry.Name, false)
		if normalizeErr != nil || name.cells == 0 {
			return normalizedFilePickerListing{}, fmt.Errorf(
				"%w: invalid file-picker entry name",
				ErrValidation,
			)
		}
		entry.Name = name.text
		retained += len(entry.Location) + len(entry.Name)
		if retained > MaxCollectionAggregateBytes {
			return normalizedFilePickerListing{}, fmt.Errorf(
				"%w: file-picker listing exceeds retained byte budget",
				ErrControlCapacity,
			)
		}
		result.listing.Entries[index] = entry
		result.entries[index] = normalizedFilePickerEntry{entry: entry, name: name}
	}
	return result, nil
}

func normalizeFilePickerResolvedEntry(entry FilePickerEntry) (FilePickerEntry, error) {
	listing, err := normalizeFilePickerListing(FilePickerListing{
		Directory: "resolved", DisplayPath: "resolved", Entries: []FilePickerEntry{entry},
	})
	if err != nil {
		return FilePickerEntry{}, err
	}
	return listing.listing.Entries[0], nil
}

func validFilePickerLocation(location string) bool {
	return location != "" && len(location) <= MaxFilePickerLocationBytes
}

func cloneFilePickerEntry(entry FilePickerEntry) FilePickerEntry { return entry }

func cloneFilePickerListing(listing FilePickerListing) FilePickerListing {
	cloned := listing
	cloned.Entries = append([]FilePickerEntry(nil), listing.Entries...)
	return cloned
}

// LocalFilePickerProviderOptions configures the opt-in local filesystem
// adapter. Root is required and establishes its navigation boundary.
type LocalFilePickerProviderOptions struct {
	Root string
}

// LocalFilePickerProvider adapts a caller-selected local directory tree to
// FilePickerProvider. It is a navigation policy, not a security sandbox.
type LocalFilePickerProvider struct {
	root string
}

// NewLocalFilePickerProvider validates and resolves one local root.
func NewLocalFilePickerProvider(
	options LocalFilePickerProviderOptions,
) (*LocalFilePickerProvider, error) {
	if options.Root == "" || len(options.Root) > MaxFilePickerLocationBytes {
		return nil, fmt.Errorf("%w: local file-picker root is required", ErrValidation)
	}
	absolute, err := filepath.Abs(options.Root)
	if err != nil {
		return nil, fmt.Errorf("local file-picker root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("local file-picker root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("local file-picker root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: local file-picker root is not a directory", ErrValidation)
	}
	if len(resolved) > MaxFilePickerLocationBytes {
		return nil, fmt.Errorf("%w: local file-picker root is too long", ErrValidation)
	}
	return &LocalFilePickerProvider{root: filepath.Clean(resolved)}, nil
}

// Root returns the immutable absolute root location token.
func (p *LocalFilePickerProvider) Root() string {
	if p == nil {
		return ""
	}
	return p.root
}

// List returns one deterministic local directory observation.
func (p *LocalFilePickerProvider) List(
	ctx context.Context,
	directory string,
) (FilePickerListing, error) {
	if ctx == nil {
		return FilePickerListing{}, fmt.Errorf("expletives: nil context")
	}
	if err := ctx.Err(); err != nil {
		return FilePickerListing{}, err
	}
	clean, resolved, err := p.resolveDirectory(directory)
	if err != nil {
		return FilePickerListing{}, err
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return FilePickerListing{}, err
	}
	listing := FilePickerListing{
		Directory: clean, DisplayPath: localPickerDisplay(clean),
		Entries: make([]FilePickerEntry, 0, min(len(entries), MaxCollectionItems)),
	}
	if clean != p.root {
		listing.Parent = filepath.Dir(clean)
	}
	if len(entries) > MaxCollectionItems {
		return FilePickerListing{}, fmt.Errorf(
			"%w: directory exceeds %d entries",
			ErrControlCapacity,
			MaxCollectionItems,
		)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return FilePickerListing{}, err
		}
		location := filepath.Join(clean, entry.Name())
		pickerEntry := FilePickerEntry{
			Name: localPickerDisplay(entry.Name()), Location: location,
			Kind: FilePickerOther,
		}
		resolvedEntry, resolveErr := filepath.EvalSymlinks(location)
		if resolveErr == nil && localPickerWithin(p.root, resolvedEntry) {
			if info, infoErr := os.Stat(location); infoErr == nil {
				pickerEntry.Size = uint64(max(int64(0), info.Size()))
				pickerEntry.Modified = info.ModTime()
				switch {
				case info.IsDir():
					pickerEntry.Kind = FilePickerDirectory
				case info.Mode().IsRegular():
					pickerEntry.Kind = FilePickerFile
				}
			}
		}
		listing.Entries = append(listing.Entries, pickerEntry)
	}
	if err := ctx.Err(); err != nil {
		return FilePickerListing{}, err
	}
	return listing, nil
}

// Resolve interprets a typed absolute or current-directory-relative path.
func (p *LocalFilePickerProvider) Resolve(
	ctx context.Context,
	directory string,
	input string,
) (FilePickerEntry, error) {
	if ctx == nil {
		return FilePickerEntry{}, fmt.Errorf("expletives: nil context")
	}
	if err := ctx.Err(); err != nil {
		return FilePickerEntry{}, err
	}
	if input == "" || len(input) > MaxFilePickerLocationBytes {
		return FilePickerEntry{}, fmt.Errorf("%w: file-picker path is empty or too long", ErrValidation)
	}
	cleanDirectory, _, err := p.resolveDirectory(directory)
	if err != nil {
		return FilePickerEntry{}, err
	}
	candidate := input
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(cleanDirectory, candidate)
	}
	candidate = filepath.Clean(candidate)
	if !localPickerWithin(p.root, candidate) {
		return FilePickerEntry{}, fmt.Errorf("%w: path is outside the picker root", ErrValidation)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return FilePickerEntry{}, err
	}
	if !localPickerWithin(p.root, resolved) {
		return FilePickerEntry{}, fmt.Errorf("%w: path resolves outside the picker root", ErrValidation)
	}
	info, err := os.Stat(candidate)
	if err != nil {
		return FilePickerEntry{}, err
	}
	entry := FilePickerEntry{
		Name: localPickerDisplay(filepath.Base(candidate)), Location: candidate,
		Kind: FilePickerOther, Size: uint64(max(int64(0), info.Size())),
		Modified: info.ModTime(),
	}
	if info.IsDir() {
		entry.Kind = FilePickerDirectory
	} else if info.Mode().IsRegular() {
		entry.Kind = FilePickerFile
	}
	return entry, nil
}

func (p *LocalFilePickerProvider) resolveDirectory(
	directory string,
) (string, string, error) {
	if p == nil || p.root == "" || !validFilePickerLocation(directory) {
		return "", "", fmt.Errorf("%w: invalid local file-picker location", ErrValidation)
	}
	clean := filepath.Clean(directory)
	if !filepath.IsAbs(clean) || !localPickerWithin(p.root, clean) {
		return "", "", fmt.Errorf("%w: directory is outside the picker root", ErrValidation)
	}
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", "", err
	}
	if !localPickerWithin(p.root, resolved) {
		return "", "", fmt.Errorf("%w: directory resolves outside the picker root", ErrValidation)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("%w: location is not a directory", ErrValidation)
	}
	return clean, resolved, nil
}

func localPickerWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func localPickerDisplay(value string) string {
	if utf8.ValidString(value) {
		return value
	}
	return strings.ToValidUTF8(value, "\uFFFD")
}
