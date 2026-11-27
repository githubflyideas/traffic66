package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWantsHelp(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"-data", "x"}, false},
		{[]string{"-h"}, true},
		{[]string{"--help"}, true},
		{[]string{"-data", "x", "-help"}, true},
		{[]string{"help"}, false}, // the word alone is a command, not a flag
	} {
		if got := wantsHelp(tc.args); got != tc.want {
			t.Errorf("wantsHelp(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestPrintHelp(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "version", nil)
	out := buf.String()
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "traffic66 version") {
		t.Errorf("help without flags:\n%s", out)
	}
	if strings.Contains(out, "Flags:") {
		t.Errorf("no flag set, but a Flags section appeared:\n%s", out)
	}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.String("addr", ":8518", "listen address")
	buf.Reset()
	printHelp(&buf, "serve", fs)
	out = buf.String()
	if !strings.Contains(out, "Flags:") || !strings.Contains(out, "-addr") {
		t.Errorf("help with flags:\n%s", out)
	}
	// withHelp wires -h to the same text
	fs2 := withHelp(flag.NewFlagSet("version", flag.ContinueOnError), "version")
	if fs2.Usage == nil {
		t.Error("withHelp left Usage unset")
	}
	if fs2.Name() != "version" {
		t.Errorf("withHelp renamed the set to %q", fs2.Name())
	}
}

func TestMultiFlag(t *testing.T) {
	var m multiFlag
	if m.String() != "" {
		t.Errorf("empty multiFlag = %q", m.String())
	}
	m.Set("a=1")
	m.Set("b=2")
	if m.String() != "a=1,b=2" {
		t.Errorf("multiFlag = %q", m.String())
	}
}

func TestRandomHexAndReadablePassword(t *testing.T) {
	h := randomHex(16)
	if len(h) != 32 {
		t.Errorf("randomHex(16) = %d chars", len(h))
	}
	for _, c := range h {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("randomHex produced %q", h)
		}
	}
	if randomHex(16) == h {
		t.Error("two randomHex draws are identical")
	}
	pw := readablePassword()
	if len(pw) != 12 {
		t.Errorf("readablePassword = %d chars", len(pw))
	}
	// the set has no look-alikes: no 0/O/1/l/I/o
	for _, c := range pw {
		if strings.ContainsRune("01lIoO", c) {
			t.Fatalf("readablePassword %q contains look-alike %q", pw, c)
		}
		if !strings.ContainsRune("abcdefghjkmnpqrstuvwxyz23456789", c) {
			t.Fatalf("readablePassword %q contains %q outside its set", pw, c)
		}
	}
}

func TestQuoteArg(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"plain", "plain"},
		{"/tmp/data dir", `"/tmp/data dir"`},
		{`say "hi"`, `"say "hi""`},
		{"it's", `"it's"`},
		{"", ""},
	} {
		if got := quoteArg(tc.in); got != tc.want {
			t.Errorf("quoteArg(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEditDistanceAndNearest(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"serve", "serve", 0},
		{"serve", "serv", 1},
		{"serve", "srve", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
		{"abc", "", 3},
	} {
		if got := editDistance(tc.a, tc.b); got != tc.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
	if got := nearestCommand("serv"); got != "serve" {
		t.Errorf("nearestCommand(serv) = %q", got)
	}
	if got := nearestCommand("zzzzzz"); got != "" {
		t.Errorf("nearestCommand(zzzzzz) = %q, want no suggestion", got)
	}
}

func TestServeFlagsCheck(t *testing.T) {
	ok := serveFlags{retention: 30, mem: 0.5, l2: 0, hold: 0, dnsRate: 20, listen: "sflow=:6343"}
	if err := ok.check(); err != nil {
		t.Fatalf("valid flags refused: %v", err)
	}
	bad := []struct {
		name   string
		mutate func(*serveFlags)
		want   string
	}{
		{"retention low", func(f *serveFlags) { f.retention = 0 }, "-retention-days"},
		{"retention high", func(f *serveFlags) { f.retention = 3651 }, "-retention-days"},
		{"memory zero", func(f *serveFlags) { f.mem = 0 }, "-memory"},
		{"memory high", func(f *serveFlags) { f.mem = 0.9 }, "-memory"},
		{"l2 high", func(f *serveFlags) { f.l2 = 65 }, "-l2-overhead"},
		{"hold high", func(f *serveFlags) { f.hold = 2 * time.Hour }, "-sampling-wait"},
		{"dns rate", func(f *serveFlags) { f.dnsRate = 0 }, "-dns-rate"},
		{"dns ttl", func(f *serveFlags) { f.dnsTTL = -time.Second }, "-dns-cache"},
		{"listen no port", func(f *serveFlags) { f.listen = "sflow=localhost" }, "-listen"},
		{"listen empty name", func(f *serveFlags) { f.listen = "=:6343" }, "-listen"},
		{"threat missing", func(f *serveFlags) { f.threats = multiFlag{"list=/nonexistent-xyz.txt"} }, "-threat"},
		{"public url", func(f *serveFlags) { f.publicURL = "ftp://example.org" }, "-public-url"},
	}
	for _, tc := range bad {
		f := ok
		f.threats = nil
		tc.mutate(&f)
		err := f.check()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: check() = %v, want an error naming %s", tc.name, err, tc.want)
		}
	}
	// -no-dns lifts the rate requirement; a bare address is a udp listener
	f := ok
	f.noDNS, f.dnsRate, f.listen = true, 0, ":6343, nf=:2055"
	if err := f.check(); err != nil {
		t.Errorf("no-dns + bare listen refused: %v", err)
	}
}

func TestPrepareDemo(t *testing.T) {
	dir := t.TempDir()
	prepareDemo(dir)
	for _, name := range []string{"inventory.txt", "asn.tsv", "threats/c2.txt", "threats/scanner.txt"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(b) == 0 {
			t.Errorf("demo file %s: err %v, %d bytes", name, err, len(b))
		}
	}
	// a second run keeps the user's edits
	p := filepath.Join(dir, "inventory.txt")
	os.WriteFile(p, []byte("mine"), 0o644)
	prepareDemo(dir)
	if b, _ := os.ReadFile(p); string(b) != "mine" {
		t.Errorf("prepareDemo overwrote an existing file: %q", b)
	}
}

func TestLoginChecker(t *testing.T) {
	defer func() { fixedPassword, firstStart = false, false }()

	// a password given at start wins and is the only login
	dir := t.TempDir()
	fixedPassword = false
	c := loginChecker(dir, "admin", "s3cret", false)
	if !fixedPassword {
		t.Error("a flag password did not mark the run as fixed-password")
	}
	if !c.Check("admin", "s3cret") || c.Check("admin", "wrong") || c.Check("other", "s3cret") {
		t.Error("the fixed checker does not honour exactly the given login")
	}

	// a fresh directory gets the default password and a first-start mark
	dir2 := t.TempDir()
	firstStart = false
	c2 := loginChecker(dir2, "admin", "", false)
	if !firstStart {
		t.Error("a fresh directory was not marked as a first start")
	}
	if !c2.Check("admin", "traffic66") {
		t.Error("the default password does not sign in on a first start")
	}
	// and a second start over the same directory is not a first start
	firstStart = false
	loginChecker(dir2, "admin", "", false)
	if firstStart {
		t.Error("an existing password file still counted as a first start")
	}
}

func TestDisplayAddr(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"[::]:6343", "127.0.0.1:6343"},
		{"0.0.0.0:6343", "127.0.0.1:6343"},
		{"127.0.0.1:6343", "127.0.0.1:6343"},
		{"192.168.1.2:8518", "192.168.1.2:8518"},
	} {
		a := &fakeAddr{s: tc.in}
		if got := displayAddr(a); got != tc.want {
			t.Errorf("displayAddr(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

type fakeAddr struct{ s string }

func (a *fakeAddr) Network() string { return "tcp" }
func (a *fakeAddr) String() string  { return a.s }
