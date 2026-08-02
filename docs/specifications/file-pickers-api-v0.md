# File And Directory Pickers API v0

- Status: Directed pre-v1 contract
- Scope: roadmap Phase 18
- Authorization: direct operator request and `EXPL-TASK-033`
- Dependencies: Collections API v0 and Modals API v0

## Purpose

Phase 18 supplies three Turbo Vision-style compound dialogs:

- `FilePickerDialog` accepts exactly one file;
- `MultiFilePickerDialog` accepts an explicitly committed set of files; and
- `DirectoryPickerDialog` accepts the directory currently being viewed.

The compounds reuse `Dialog`, `TextField`, `ListBox`, `StaticText`, `Button`,
`Panel`, and `Layout`. They do not add a filesystem dependency to the event
loop, renderer, collection controls, or modal stack.

## Provider Boundary

```go
const MaxFilePickerLocationBytes = MaxTextInputBytes

type FilePickerEntryKind string

const (
    FilePickerFile      FilePickerEntryKind = "file"
    FilePickerDirectory FilePickerEntryKind = "directory"
    FilePickerOther     FilePickerEntryKind = "other"
)

type FilePickerEntry struct {
    Name     string
    Location string
    Kind     FilePickerEntryKind
    Size     uint64
    Modified time.Time
}

type FilePickerListing struct {
    Directory   string
    DisplayPath string
    Parent      string
    Entries     []FilePickerEntry
}

type FilePickerProvider interface {
    List(context.Context, string) (FilePickerListing, error)
    Resolve(context.Context, string, string) (FilePickerEntry, error)
}
```

`Directory`, `Parent`, `Location`, and the directory argument are opaque
provider tokens. They are never interpreted, joined, cleaned, or rendered by
the dialog. A token may contain arbitrary bytes, but must be nonempty where
required and no longer than `MaxFilePickerLocationBytes`.

`DisplayPath` and `Name` are display strings. They must be valid UTF-8 and
obey the project's one-cell display policy after canonical replacement.
`Directory` is nonempty. `Parent` is empty only when the listing has no
navigable parent. Entry locations are nonempty and unique within one listing.
An entry name is nonempty, contains no line separator, and is bounded by
`MaxDisplayTextBytes`. Listings contain no more than `MaxCollectionItems`
entries and their retained display strings and tokens remain within
`MaxCollectionAggregateBytes`.

`List` returns one complete immutable directory observation. `Resolve`
interprets the user-entered display text relative to the supplied directory
and returns one file, directory, or other entry. The dialog validates and
copies every returned value before publication. Invalid provider data fails
closed as a provider-contract error; a normal inaccessible, missing, or
changed path is a recoverable picker error.

Each call is explicit and cancellation-aware. Construction receives a
`context.Context` and performs exactly one `List` request before atomically
constructing the compound. Navigation, refresh, and typed-path resolution
perform at most one provider operation apiece. Provider calls are serialized
per dialog, run outside the App state lock, and publish one atomic UI update
only after the call returns. No renderer, snapshot, paint, focus, resize, or
getter path invokes the provider.

## Filtering And Sorting

```go
type FilePickerFilter struct {
    Key      string
    Label    string
    Patterns []string
}

type FilePickerSortField string

const (
    FilePickerSortName     FilePickerSortField = "name"
    FilePickerSortSize     FilePickerSortField = "size"
    FilePickerSortModified FilePickerSortField = "modified"
)

type FilePickerSort struct {
    Field     FilePickerSortField
    Direction SortDirection
}
```

Filter keys are stable bounded identifiers. Labels and patterns are copied
and bounded. Patterns use Go `path.Match` syntax against the complete entry
`Name`, never against the opaque location. Directories remain visible so the
user can navigate. An empty filter list supplies one implicit `*` filter.
Typing a pattern in the path field applies that transient pattern without a
provider call.

Directories sort before non-directories. The requested name, size, or
modified-time field then determines stable ascending or descending order;
canonical display name and opaque location break ties. Name ascending is the
zero-value default. Filter or sort changes preserve current and selected
locations when they remain visible and otherwise choose the first eligible
entry.

## Constructors And Public Results

```go
type FilePickerDialogOptions struct {
    DialogOptions
    Provider         FilePickerProvider
    InitialDirectory string
    Filters          []FilePickerFilter
    Filter           string
    Sort             FilePickerSort
}

func NewFilePickerDialog(
    context.Context, Container, FilePickerDialogOptions,
) (*FilePickerDialog, error)

func NewMultiFilePickerDialog(
    context.Context, Container, FilePickerDialogOptions,
) (*MultiFilePickerDialog, error)

func NewDirectoryPickerDialog(
    context.Context, Container, FilePickerDialogOptions,
) (*DirectoryPickerDialog, error)
```

`Provider` and a nonempty `InitialDirectory` are required. There are no
transaction constructors: provider I/O must complete before the compound's
single atomic control-tree construction. The constructors return only after
the initial provider request and compound commit complete. Caller
cancellation leaves no provisional controls.

All three types embed the ordinary one-shot `Dialog` lifecycle and expose
their canonical path field, list, information text, and action buttons for
ordinary application composition and tests. They also provide thread-safe
copied state, `Refresh`, `Navigate`, `SetFilter`, and `SetSort` methods.

An accepted `FilePickerDialog` exposes one copied file entry. An accepted
`MultiFilePickerDialog` exposes a copied nonempty slice ordered by current
display order, regardless of selection order. An accepted
`DirectoryPickerDialog` exposes the current listing's directory token and
display path. Cancelled, interrupted, exited, failed, or still-active dialogs
do not expose an accepted value.

## Local Filesystem Adapter

```go
type LocalFilePickerProviderOptions struct {
    Root string
}

func NewLocalFilePickerProvider(
    LocalFilePickerProviderOptions,
) (*LocalFilePickerProvider, error)
```

The local adapter is opt-in. `Root` is required, resolved to an absolute
directory at construction, and becomes the adapter's initial/root location.
The adapter uses `os.ReadDir`, cancellation checks, deterministic copied
metadata, and path resolution confined to its configured navigation root.
It sanitizes invalid filename bytes for display while retaining the original
path bytes in opaque location tokens.

This is a picker-navigation policy, not a security sandbox. Filesystem races,
mount changes, hard links, symlink replacement, and the caller's subsequent
open operation remain outside its authority. Applications handling hostile
paths need an OS-enforced sandbox or their own provider.

## Visual And Keyboard Contract

The normal dialog is centered, double-framed, gray, and shadowed. Its
preferred geometry follows the classic Turbo Vision file dialog: a path
label and one-row input at the top; a scrollable file list on the left; a
right-side vertical stack of raised green buttons; and a compact information
area below. The dialog may grow with the terminal and degrades by clipping
and scrolling rather than by fixed conventional terminal maxima.

Directories render with a trailing `/`. Button, dialog, list, text-field,
mnemonic, border, and shadow colors come from semantic styles and follow the
project's Turbo Vision baseline. Mnemonic letters use the shared action
renderer.

- Tab and Shift-Tab move among the path, list, and button focus groups.
- Up/Down, Home/End, and Page Up/Page Down move list current state.
- Space toggles a file in the multiple picker and only changes current in the
  other variants.
- Enter activates the current list row. It navigates into directories,
  accepts a file in the single picker, and toggles a file without accepting
  in the multiple picker.
- The multiple picker closes successfully only through its explicit Open
  action with a nonempty selection.
- The directory picker Enter action navigates; its Select action accepts the
  directory currently being viewed.
- Enter commits path editing; a subsequent Enter resolves the committed
  path, navigates a directory, applies a wildcard, or accepts an eligible
  single file.
- Alt-O activates Open, Alt-U activates Up, Alt-C cancels, and Escape cancels.
- F5 refreshes the current listing without changing modal meaning.
- Ctrl-C remains the App's configured interrupt, never picker Cancel.

Provider failures remain within the same dialog as bounded error state. They
do not open a recursive MessageBox. A successful later operation clears the
error. A disappeared current or selected entry is repaired by stable location
at the next provider publication and can never be returned after it is absent.

## Automation And Privacy

Each specialization has its own control kind. Its typed semantic details
contain mode, provider status, display path, entry/file/directory counts,
current entry kind and display name, selected count, active filter and sort,
and recoverable error state. Opaque locations and accepted result tokens are
not projected through attached automation. Ordinary child controls remain
individually discoverable by derived automation keys.

Automation drives the exact raw logical key path used by a human, including
KeyDown/KeyUp/KeyPress modifier chords. Direct semantic picker commands use
the same internal controller. Every provider-triggering command retains one
request completion and its post-operation frame sequence; timeout never means
success.

## Acceptance

- deterministic providers cover root/parent/empty navigation, filtering,
  sorting, preservation, disappearance, invalid provider data, and errors;
- single, multiple, and directory results are typed and accept only their
  documented values;
- cancel, interrupt, quit, provider failure, and small geometry remain
  distinct and observable;
- concurrent public calls are race-free and provider calls never occur under
  the App state lock;
- the opt-in local adapter is tested in a uniquely owned temporary root;
- `expletives-test` exposes all three useful dialogs plus error and resize
  cases; and
- ordinary, race, all-build-mode, self-check, and attached automation gates
  pass before the phase ACP.
