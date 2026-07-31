package buildinfo

import (
	"strings"
	"testing"
)

func TestCurrentHasModeAndGoVersion(t *testing.T) {
	got := Current()
	if got.Mode == "" {
		t.Fatal("Current().Mode is empty")
	}
	if got.GoVersion == "" {
		t.Fatal("Current().GoVersion is empty")
	}
	if got.GOOS == "" || got.GOARCH == "" {
		t.Fatalf("Current() target is incomplete: GOOS=%q GOARCH=%q", got.GOOS, got.GOARCH)
	}
}

func TestInfoStringIsBoundedAndSingleLine(t *testing.T) {
	info := Info{
		Mode:          "debug",
		GoVersion:     "go1.test",
		GOOS:          "testos",
		GOARCH:        "testarch",
		ModulePath:    "example.invalid/module",
		ModuleVersion: "v1.2.3",
		Revision:      "abc123",
		RevisionTime:  "2026-07-24T00:00:00Z",
		Modified:      true,
	}

	got := info.String()
	for _, want := range []string{
		"mode=debug",
		"module=example.invalid/module",
		"version=v1.2.3",
		"revision=abc123",
		"revision_time=2026-07-24T00:00:00Z",
		"modified=true",
		"go=go1.test",
		"target=testos/testarch",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Info.String() = %q, want substring %q", got, want)
		}
	}
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("Info.String() = %q, want one line", got)
	}
}
