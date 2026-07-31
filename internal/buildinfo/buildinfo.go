// Package buildinfo reports the provenance embedded by the Go tool and the
// project build mode selected by the root Makefile.
package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
)

// buildMode is set with the linker's -X option. The default is intentionally
// useful for go run and for tests that do not pass through the Makefile.
var buildMode = "development"

// Info is the bounded, non-sensitive build provenance exposed by commands.
type Info struct {
	Mode          string
	GoVersion     string
	GOOS          string
	GOARCH        string
	ModulePath    string
	ModuleVersion string
	Revision      string
	RevisionTime  string
	Modified      bool
}

// Current returns the build provenance available to the running executable.
func Current() Info {
	result := Info{
		Mode:      buildMode,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}

	build, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}

	result.ModulePath = build.Main.Path
	result.ModuleVersion = build.Main.Version
	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			result.Revision = setting.Value
		case "vcs.time":
			result.RevisionTime = setting.Value
		case "vcs.modified":
			result.Modified, _ = strconv.ParseBool(setting.Value)
		}
	}
	return result
}

// String returns a stable, single-line representation suitable for --version.
func (i Info) String() string {
	moduleVersion := i.ModuleVersion
	if moduleVersion == "" {
		moduleVersion = "(unknown)"
	}
	revision := i.Revision
	if revision == "" {
		revision = "(unknown)"
	}
	revisionTime := i.RevisionTime
	if revisionTime == "" {
		revisionTime = "(unknown)"
	}

	return fmt.Sprintf(
		"mode=%s module=%s version=%s revision=%s revision_time=%s modified=%t go=%s target=%s/%s",
		i.Mode,
		i.ModulePath,
		moduleVersion,
		revision,
		revisionTime,
		i.Modified,
		i.GoVersion,
		i.GOOS,
		i.GOARCH,
	)
}
