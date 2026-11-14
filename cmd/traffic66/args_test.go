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

func TestParsePublicURL(t *testing.T) {
	for v, ok := range map[string]bool{"": true, "https://t66.example.org": true, "https://t66.example.org:8443/": true, "http://10.0.0.5:8080": true,
		"t66.example.org": false, "ftp://x": false, "https://x/traffic66": false, "https://u:p@x": false, "https://": false} {
		if _, err := parsePublicURL(v); (err == nil) != ok {
			t.Errorf("parsePublicURL(%q): %v", v, err)
		}
	}
}
