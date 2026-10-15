package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The default version went stale once: the source said 0.1.0-dev while the
// program was at 2.0.0, so anything built without the release workflow —
// including a contributor's own build, and the version in a bug report —
// named a release two majors behind. This keeps the two in step.
func TestDevVersionMatchesChangelog(t *testing.T) {
	b, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## ([0-9]+\.[0-9]+\.[0-9]+)\s*$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no version heading in CHANGELOG.md")
	}
	if got := string(m[1]); got != devVersion {
		t.Errorf("devVersion is %q, newest CHANGELOG.md heading is %q: update devVersion in main.go", devVersion, got)
	}
}

func TestBuildVersionSaysDev(t *testing.T) {
	v := buildVersion()
	if !strings.HasPrefix(v, devVersion+"-dev") {
		t.Errorf("buildVersion() = %q, want it to start with %q", v, devVersion+"-dev")
	}
}

// A build whose version was set at link time keeps it; the fallback only
// fills in an empty one.
func TestVersionSet(t *testing.T) {
	if version == "" {
		t.Error("version is empty; init should have filled it in")
	}
}
