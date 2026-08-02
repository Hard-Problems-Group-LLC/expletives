package expletives

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type pickerFixtureProvider struct {
	mu       sync.Mutex
	listings map[string]FilePickerListing
	resolved map[string]FilePickerEntry
	errors   map[string]error
	calls    []string
	hook     func()
}

func (p *pickerFixtureProvider) List(
	ctx context.Context,
	directory string,
) (FilePickerListing, error) {
	if err := ctx.Err(); err != nil {
		return FilePickerListing{}, err
	}
	if p.hook != nil {
		p.hook()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, "list:"+directory)
	if err := p.errors[directory]; err != nil {
		return FilePickerListing{}, err
	}
	return cloneFilePickerListing(p.listings[directory]), nil
}

func (p *pickerFixtureProvider) Resolve(
	ctx context.Context,
	directory string,
	input string,
) (FilePickerEntry, error) {
	if err := ctx.Err(); err != nil {
		return FilePickerEntry{}, err
	}
	if p.hook != nil {
		p.hook()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, "resolve:"+directory+":"+input)
	if err := p.errors[directory+":"+input]; err != nil {
		return FilePickerEntry{}, err
	}
	entry, found := p.resolved[directory+":"+input]
	if !found {
		return FilePickerEntry{}, errors.New("not found")
	}
	return entry, nil
}

func standardPickerFixture() *pickerFixtureProvider {
	modified := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	return &pickerFixtureProvider{
		listings: map[string]FilePickerListing{
			"root": {
				Directory: "root", DisplayPath: "/fixture",
				Entries: []FilePickerEntry{
					{Name: "beta.go", Location: "root/beta.go", Kind: FilePickerFile, Size: 20, Modified: modified},
					{Name: "nested", Location: "nested", Kind: FilePickerDirectory, Modified: modified},
					{Name: "alpha.txt", Location: "root/alpha.txt", Kind: FilePickerFile, Size: 10, Modified: modified},
				},
			},
			"nested": {
				Directory: "nested", DisplayPath: "/fixture/nested", Parent: "root",
				Entries: []FilePickerEntry{
					{Name: "inside.txt", Location: "nested/inside.txt", Kind: FilePickerFile, Size: 30, Modified: modified},
				},
			},
		},
		resolved: map[string]FilePickerEntry{
			"root:alpha.txt": {Name: "alpha.txt", Location: "root/alpha.txt", Kind: FilePickerFile, Size: 10, Modified: modified},
			"root:nested":    {Name: "nested", Location: "nested", Kind: FilePickerDirectory, Modified: modified},
		},
		errors: make(map[string]error),
	}
}

func newPickerTestApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp(AppOptions{Size: Size{Width: 80, Height: 25}, Scenario: "picker-test"})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func listKeyByLabel(t *testing.T, list *ListBox, label string) string {
	t.Helper()
	for _, item := range list.Items() {
		if item.Label == label {
			return item.Key
		}
	}
	t.Fatalf("ListBox has no item labeled %q: %#v", label, list.Items())
	return ""
}

func TestFilePickerDialogFiltersSortsNavigatesAndAccepts(t *testing.T) {
	app := newPickerTestApp(t)
	provider := standardPickerFixture()
	picker, err := NewFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "picker.single"},
		}},
		Provider: provider, InitialDirectory: "root",
		Filters: []FilePickerFilter{
			{Key: "text", Label: "Text", Patterns: []string{"*.txt"}},
			{Key: "all", Label: "All", Patterns: []string{"*"}},
		},
		Filter: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	items := picker.List().Items()
	if len(items) != 2 || items[0].Label != "nested/" || items[1].Label != "alpha.txt" ||
		picker.List().State().SelectionMarks {
		t.Fatalf("initial picker list/state = %#v / %#v", items, picker.List().State())
	}
	state := picker.State()
	if state.Mode != "single" || state.DisplayPath != "/fixture" ||
		state.EntryCount != 3 || state.FileCount != 2 || state.DirectoryCount != 1 ||
		state.Filter != "text" || state.Sort.Field != FilePickerSortName {
		t.Fatalf("initial picker State() = %#v", state)
	}
	if err := picker.Show(nil); err != nil {
		t.Fatal(err)
	}
	if !controlByKey(t, app.Snapshot(), "picker.single.list").Focused {
		t.Fatal("picker did not initially focus its ready list")
	}
	completion, err := picker.List().Activate(context.Background(), "test", "open-directory")
	if err != nil || completion.Outcome != OutcomeApplied || !picker.Active() {
		t.Fatalf("directory activation = %+v, %v active=%t", completion, err, picker.Active())
	}
	if picker.State().DisplayPath != "/fixture/nested" || picker.List().Items()[0].Label != "inside.txt" {
		t.Fatalf("nested state/items = %#v / %#v", picker.State(), picker.List().Items())
	}
	completion, err = picker.UpButton().Activate(context.Background(), "test", "go-up")
	if err != nil || completion.Outcome != OutcomeApplied || picker.State().DisplayPath != "/fixture" {
		t.Fatalf("Up activation = %+v, %v state=%#v", completion, err, picker.State())
	}
	if err := picker.List().SetCurrent(listKeyByLabel(t, picker.List(), "alpha.txt")); err != nil {
		t.Fatal(err)
	}
	completion, err = picker.List().Activate(context.Background(), "test", "accept-file")
	if err != nil || completion.Outcome != OutcomeApplied || picker.Active() {
		t.Fatalf("file activation = %+v, %v active=%t", completion, err, picker.Active())
	}
	selection, accepted := picker.Selection()
	if !accepted || selection.Location != "root/alpha.txt" || selection.Kind != FilePickerFile {
		t.Fatalf("Selection() = %#v, %t", selection, accepted)
	}
}

func TestMultiAndDirectoryPickerResultsAreDistinct(t *testing.T) {
	t.Run("multiple requires explicit commit", func(t *testing.T) {
		app := newPickerTestApp(t)
		picker, err := NewMultiFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "picker.multiple"},
			}},
			Provider: standardPickerFixture(), InitialDirectory: "root",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := picker.Show(nil); err != nil {
			t.Fatal(err)
		}
		completion, err := picker.OpenButton().Activate(context.Background(), "test", "empty-open")
		if err != nil || completion.Outcome != OutcomeRejected || !picker.Active() {
			t.Fatalf("empty multi Open = %+v, %v active=%t", completion, err, picker.Active())
		}
		keys := []string{
			listKeyByLabel(t, picker.List(), "alpha.txt"),
			listKeyByLabel(t, picker.List(), "beta.go"),
		}
		if err := picker.List().SetSelection(keys); err != nil {
			t.Fatal(err)
		}
		completion, err = picker.OpenButton().Activate(context.Background(), "test", "commit-open")
		if err != nil || completion.Outcome != OutcomeApplied || picker.Active() {
			t.Fatalf("multi Open = %+v, %v active=%t", completion, err, picker.Active())
		}
		selections, accepted := picker.Selections()
		if !accepted || len(selections) != 2 || selections[0].Name != "alpha.txt" || selections[1].Name != "beta.go" {
			t.Fatalf("Selections() = %#v, %t", selections, accepted)
		}
	})

	t.Run("directory accepts current listing", func(t *testing.T) {
		app := newPickerTestApp(t)
		picker, err := NewDirectoryPickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
			DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
				PanelOptions: PanelOptions{AutomationKey: "picker.directory"},
			}},
			Provider: standardPickerFixture(), InitialDirectory: "root",
		})
		if err != nil {
			t.Fatal(err)
		}
		if items := picker.List().Items(); len(items) != 1 || items[0].Label != "nested/" {
			t.Fatalf("directory items = %#v", items)
		}
		if err := picker.Show(nil); err != nil {
			t.Fatal(err)
		}
		completion, err := picker.SelectButton().Activate(context.Background(), "test", "select-directory")
		if err != nil || completion.Outcome != OutcomeApplied || picker.Active() {
			t.Fatalf("directory Select = %+v, %v active=%t", completion, err, picker.Active())
		}
		selection, accepted := picker.Selection()
		if !accepted || selection.Kind != FilePickerDirectory ||
			selection.Location != "root" || selection.Name != "/fixture" {
			t.Fatalf("directory Selection() = %#v, %t", selection, accepted)
		}
	})
}

func TestFilePickerRefreshErrorsFilterSortCancelAndOutsideLock(t *testing.T) {
	app := newPickerTestApp(t)
	provider := standardPickerFixture()
	provider.hook = func() { _ = app.Snapshot() }
	picker, err := NewFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "picker.operations"},
		}},
		Provider: provider, InitialDirectory: "root",
		Filters: []FilePickerFilter{
			{Key: "text", Label: "Text", Patterns: []string{"*.txt"}},
			{Key: "all", Label: "All", Patterns: []string{"*"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := picker.SetFilter(context.Background(), "text"); err != nil {
		t.Fatal(err)
	}
	if len(picker.List().Items()) != 2 {
		t.Fatalf("filtered items = %#v", picker.List().Items())
	}
	if err := picker.SetSort(context.Background(), FilePickerSort{
		Field: FilePickerSortSize, Direction: SortDescending,
	}); err != nil {
		t.Fatal(err)
	}
	if picker.State().Sort.Field != FilePickerSortSize {
		t.Fatalf("sorted state = %#v", picker.State())
	}
	if err := picker.Show(nil); err != nil {
		t.Fatal(err)
	}
	provider.mu.Lock()
	provider.errors["root"] = errors.New("permission denied")
	provider.mu.Unlock()
	completion, err := picker.RefreshButton().Activate(context.Background(), "test", "refresh-error")
	if err != nil || completion.Outcome != OutcomeFailed || picker.State().Status != CollectionError ||
		picker.State().Error == "" || !picker.Active() {
		t.Fatalf("failed Refresh = %+v, %v state=%#v", completion, err, picker.State())
	}
	provider.mu.Lock()
	delete(provider.errors, "root")
	provider.mu.Unlock()
	completion, err = picker.RefreshButton().Activate(context.Background(), "test", "refresh-recover")
	if err != nil || completion.Outcome != OutcomeApplied || picker.State().Status != CollectionReady ||
		picker.State().Error != "" {
		t.Fatalf("recovered Refresh = %+v, %v state=%#v", completion, err, picker.State())
	}
	completion, err = picker.CancelButton().Activate(context.Background(), "test", "cancel")
	if err != nil || completion.Outcome != OutcomeApplied || picker.Active() {
		t.Fatalf("Cancel = %+v, %v active=%t", completion, err, picker.Active())
	}
	if _, accepted := picker.Selection(); accepted {
		t.Fatal("cancelled picker exposed an accepted selection")
	}
}

func TestFilePickerValidationAndInitialOperationalError(t *testing.T) {
	app := newPickerTestApp(t)
	if _, err := NewFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing provider error = %v, want ErrValidation", err)
	}
	provider := standardPickerFixture()
	if _, err := NewFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
		Provider: provider, InitialDirectory: "root",
		Filters: []FilePickerFilter{{Key: "broken", Label: "Broken", Patterns: []string{"["}}},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad filter error = %v, want ErrValidation", err)
	}

	provider.errors["missing"] = errors.New("missing directory")
	picker, err := NewFilePickerDialog(context.Background(), app.Root(), FilePickerDialogOptions{
		DialogOptions: DialogOptions{ModalPanelOptions: ModalPanelOptions{
			PanelOptions: PanelOptions{AutomationKey: "picker.error"},
		}},
		Provider: provider, InitialDirectory: "missing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if state := picker.State(); state.Status != CollectionError || state.Error == "" {
		t.Fatalf("initial error State() = %#v", state)
	}
	if err := picker.Show(nil); err != nil {
		t.Fatal(err)
	}
	if !controlByKey(t, app.Snapshot(), "picker.error.path").Focused {
		t.Fatal("error picker did not focus its usable path field")
	}
}
