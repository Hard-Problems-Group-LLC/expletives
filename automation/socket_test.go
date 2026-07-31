//go:build linux

package automation

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestListenUnixRefusesExistingPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "automation.sock")
	const contents = "owned by test"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, _, err := listenUnix(path); err == nil {
		t.Fatal("listenUnix() error = nil, want collision error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != contents {
		t.Errorf("existing path contents = %q, want %q", got, contents)
	}
}

func TestListenUnixModeAndOwnedCleanup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "automation.sock")
	listener, created, err := listenUnix(path)
	if err != nil {
		t.Fatalf("listenUnix() error = %v", err)
	}
	defer listener.Close()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("socket mode = %#o, want 0600", got)
	}
	if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Close() error = %v", err)
	}
	if err := removeOwnedSocket(path, created); err != nil {
		t.Fatalf("removeOwnedSocket() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Lstat() after cleanup error = %v, want not exist", err)
	}
}

func TestRemoveOwnedSocketRefusesReplacement(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "automation.sock")
	listener, created, err := listenUnix(path)
	if err != nil {
		t.Fatalf("listenUnix() error = %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove() socket error = %v", err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatalf("WriteFile() replacement error = %v", err)
	}

	if err := removeOwnedSocket(path, created); err == nil {
		t.Fatal("removeOwnedSocket() error = nil, want identity error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() replacement error = %v", err)
	}
	if string(got) != "replacement" {
		t.Errorf("replacement contents = %q, want replacement", got)
	}
}
