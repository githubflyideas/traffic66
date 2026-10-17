package main

import (
	"reflect"
	"testing"
)

func TestCleanArgs(t *testing.T) {
	got := cleanArgs([]string{"　interfaces'.", "－capture", "“Wi-Fi”", " ", "-listen", ""})
	want := []string{"interfaces", "-capture", "Wi-Fi", "-listen", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if s := nearestCommand("interface"); s != "interfaces" {
		t.Fatalf("nearest %q", s)
	}
	if s := nearestCommand("zzzzzz"); s != "" {
		t.Fatalf("nearest %q", s)
	}
}

// A plain build says so. Only the release workflow sets this, and support
// and upgrade promises only cover what it builds, so the default must not
// quietly claim to be ours.
func TestBuildKindDefaultsToCommunity(t *testing.T) {
	if buildKind != "community" {
		t.Fatalf("buildKind = %q; a build with no -ldflags must say community", buildKind)
	}
}
