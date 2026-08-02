package expletives

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeFilePickerListingCopiesAndBoundsProviderData(t *testing.T) {
	provided := FilePickerListing{
		Directory: "d", DisplayPath: "/virtual", Parent: "root",
		Entries: []FilePickerEntry{
			{Name: "alpha.txt", Location: "d/alpha", Kind: FilePickerFile, Size: 5},
			{Name: "nested", Location: "d/nested", Kind: FilePickerDirectory},
		},
	}
	normalized, err := normalizeFilePickerListing(provided)
	if err != nil {
		t.Fatal(err)
	}
	provided.Entries[0].Name = "mutated"
	if got := normalized.listing.Entries[0].Name; got != "alpha.txt" {
		t.Fatalf("copied entry name = %q", got)
	}
	if normalized.path.text != "/virtual" || len(normalized.entries) != 2 {
		t.Fatalf("normalized listing = %#v", normalized)
	}

	tests := []FilePickerListing{
		{DisplayPath: "/", Entries: nil},
		{Directory: "d", DisplayPath: "", Entries: nil},
		{Directory: "d", DisplayPath: "/", Entries: []FilePickerEntry{
			{Name: "same", Location: "same", Kind: FilePickerFile},
			{Name: "again", Location: "same", Kind: FilePickerFile},
		}},
		{Directory: "d", DisplayPath: "/", Entries: []FilePickerEntry{
			{Name: "bad", Location: "bad", Kind: "socket"},
		}},
	}
	for index, candidate := range tests {
		if _, err := normalizeFilePickerListing(candidate); !errors.Is(err, ErrValidation) {
			t.Errorf("invalid listing %d error = %v, want ErrValidation", index, err)
		}
	}

	tooMany := make([]FilePickerEntry, MaxCollectionItems+1)
	if _, err := normalizeFilePickerListing(FilePickerListing{
		Directory: "d", DisplayPath: "/", Entries: tooMany,
	}); !errors.Is(err, ErrControlCapacity) {
		t.Fatalf("oversized listing error = %v, want ErrControlCapacity", err)
	}
}

func TestLocalFilePickerProviderListsResolvesAndConfines(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(file, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "pipe")
	if err := os.Symlink(filepath.Join(root, "missing"), other); err != nil {
		t.Fatal(err)
	}

	provider, err := NewLocalFilePickerProvider(LocalFilePickerProviderOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(provider.Root()) {
		t.Fatalf("Root() = %q, want absolute", provider.Root())
	}
	listing, err := provider.List(context.Background(), provider.Root())
	if err != nil {
		t.Fatal(err)
	}
	if listing.Directory != provider.Root() || listing.Parent != "" ||
		listing.DisplayPath != provider.Root() || len(listing.Entries) != 3 {
		t.Fatalf("root listing = %#v", listing)
	}
	kinds := make(map[string]FilePickerEntryKind)
	for _, entry := range listing.Entries {
		kinds[entry.Name] = entry.Kind
	}
	if kinds["alpha.txt"] != FilePickerFile ||
		kinds["directory"] != FilePickerDirectory || kinds["pipe"] != FilePickerOther {
		t.Fatalf("entry kinds = %#v", kinds)
	}

	resolved, err := provider.Resolve(context.Background(), provider.Root(), "alpha.txt")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != FilePickerFile || resolved.Location != file || resolved.Size != 5 {
		t.Fatalf("resolved file = %#v", resolved)
	}
	nested, err := provider.List(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	if nested.Parent != provider.Root() {
		t.Fatalf("nested parent = %q, want %q", nested.Parent, provider.Root())
	}
	if _, err := provider.Resolve(context.Background(), provider.Root(), "../outside"); !errors.Is(err, ErrValidation) {
		t.Fatalf("outside resolve error = %v, want ErrValidation", err)
	}
	if _, err := provider.List(context.Background(), filepath.Dir(provider.Root())); !errors.Is(err, ErrValidation) {
		t.Fatalf("outside list error = %v, want ErrValidation", err)
	}
}

func TestLocalFilePickerProviderValidationAndCancellation(t *testing.T) {
	if _, err := NewLocalFilePickerProvider(LocalFilePickerProviderOptions{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty root error = %v, want ErrValidation", err)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocalFilePickerProvider(LocalFilePickerProviderOptions{Root: file}); !errors.Is(err, ErrValidation) {
		t.Fatalf("file root error = %v, want ErrValidation", err)
	}

	provider, err := NewLocalFilePickerProvider(LocalFilePickerProviderOptions{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.List(ctx, provider.Root()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled List error = %v", err)
	}
	if _, err := provider.Resolve(ctx, provider.Root(), "anything"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Resolve error = %v", err)
	}
	if _, err := provider.Resolve(context.Background(), provider.Root(), strings.Repeat("x", MaxFilePickerLocationBytes+1)); !errors.Is(err, ErrValidation) {
		t.Fatalf("oversized Resolve error = %v, want ErrValidation", err)
	}
}
