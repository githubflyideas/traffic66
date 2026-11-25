package collector

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// What a blocked source has sent since it was blocked is exactly what
// somebody wants to see after a restart — a count that starts again from
// zero says the block is new, which it is not.
func TestDropCountsSurviveARestart(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBlocklist(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := netip.MustParseAddr("192.0.2.7")
	if err := b.Add(a, "test rig left running", time.Now()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2600; i++ {
		b.Has(a)
	}
	if err := b.Flush(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, BlockFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "dropped=2600") {
		t.Errorf("the file does not carry the count:\n%s", raw)
	}

	again, err := NewBlocklist(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := again.List()
	if len(list) != 1 {
		t.Fatalf("%d entries after reload, want 1", len(list))
	}
	if list[0].Dropped != 2600 {
		t.Errorf("Dropped = %d after reload, want 2600", list[0].Dropped)
	}
	if list[0].Note != "test rig left running" {
		t.Errorf("note = %q, want it kept beside the count", list[0].Note)
	}
	// Flushing with nothing new must not rewrite the file.
	st, _ := os.Stat(filepath.Join(dir, BlockFile))
	if err := again.Flush(); err != nil {
		t.Fatal(err)
	}
	st2, _ := os.Stat(filepath.Join(dir, BlockFile))
	if !st.ModTime().Equal(st2.ModTime()) {
		t.Error("Flush rewrote the file although no datagram was dropped")
	}
}

// An address that can never be the source of a datagram is not worth a
// line in the file.
func TestBlocklistRejectsAddressesThatCannotSend(t *testing.T) {
	b, err := NewBlocklist(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"0.0.0.0", "::", "224.0.0.1", "ff02::1"} {
		if err := b.Add(netip.MustParseAddr(s), "", time.Now()); err == nil {
			t.Errorf("%s was accepted", s)
		}
	}
	if n := len(b.List()); n != 0 {
		t.Errorf("%d entries, want none", n)
	}
}
