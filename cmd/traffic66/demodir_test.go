package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The demo claims only a directory that is new, empty or already its
// own: an installation's data is never opened with the demo's published
// password.
func TestClaimDemoDir(t *testing.T) {
	base := t.TempDir()
	fresh := filepath.Join(base, "fresh")
	if err := claimDemoDir(fresh); err != nil {
		t.Fatalf("new dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fresh, demoMarker)); err != nil {
		t.Fatal("not marked")
	}
	if err := claimDemoDir(fresh); err != nil {
		t.Fatalf("its own dir again: %v", err)
	}
	empty := filepath.Join(base, "empty")
	os.Mkdir(empty, 0o700)
	if err := claimDemoDir(empty); err != nil {
		t.Fatalf("empty dir: %v", err)
	}
	real := filepath.Join(base, "real")
	os.MkdirAll(filepath.Join(real, "raw"), 0o700)
	os.WriteFile(filepath.Join(real, "password"), []byte("x"), 0o600)
	if err := claimDemoDir(real); err == nil {
		t.Fatal("an installation's data directory was claimed")
	}
	if _, err := os.Stat(filepath.Join(real, demoMarker)); err == nil {
		t.Fatal("a refused directory was marked")
	}
}
