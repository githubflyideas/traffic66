package main

import (
	"os/exec"
	"strings"
	"testing"
)

// Three files had drifted out of gofmt and stayed that way, because
// nothing looked. CI runs vet and the tests, neither of which cares about
// layout, so this is the thing that looks.
//
// It shells out to gofmt rather than reimplementing it: the question is
// what the tool in the contributor's hands says, not what this test
// thinks. If gofmt is not on the path — a packaged Go without the
// toolchain — the test skips rather than failing, because an installation
// that cannot run it cannot act on it either.
//
// It reads files outside its own package, which the test cache does not
// know about, so a local `go test` can report a stale pass after
// something elsewhere is reformatted. Run it with -count=1 to be sure.
// CI already passes -count=1 to the whole suite, so the check there is
// always a fresh one.
func TestTreeIsGofmtClean(t *testing.T) {
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt is not on the path")
	}
	out, err := exec.Command("gofmt", "-l", "../..").CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt: %v: %s", err, out)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Errorf("not gofmt clean; run gofmt -w on:\n%s", files)
	}
}
