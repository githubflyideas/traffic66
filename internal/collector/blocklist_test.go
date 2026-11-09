package collector

import (
	"context"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

type countSink struct {
	records  int
	counters int
}

func (s *countSink) Submit(r []flow.Record)             { s.records += len(r) }
func (s *countSink) SubmitCounters(c []flow.IfCounters) { s.counters += len(c) }

// A block survives a restart, because the next start reads the same file.
func TestBlocklistSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBlocklist(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.List()) != 0 {
		t.Fatalf("a fresh installation blocks %d sources", len(b.List()))
	}
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := b.Add(netip.MustParseAddr("192.0.2.7"), "test rig left running", at); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, BlockFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "192.0.2.7") || !strings.Contains(string(raw), "test rig left running") {
		t.Fatalf("the file does not hold the block:\n%s", raw)
	}

	again, err := NewBlocklist(dir) // the restart
	if err != nil {
		t.Fatal(err)
	}
	got := again.List()
	if len(got) != 1 {
		t.Fatalf("after a restart %d sources are blocked, want 1", len(got))
	}
	if got[0].Address != "192.0.2.7" || got[0].Note != "test rig left running" || !got[0].Since.Equal(at) {
		t.Errorf("the block came back as %+v", got[0])
	}
	if !again.Has(netip.MustParseAddr("192.0.2.7")) {
		t.Error("the restored block does not match its own address")
	}
	if again.Has(netip.MustParseAddr("192.0.2.8")) {
		t.Error("an address that was never blocked matches")
	}
}

// Unblocking is the way back from a mis-click, and says whether there was
// anything to undo.
func TestBlocklistRemove(t *testing.T) {
	dir := t.TempDir()
	b, _ := NewBlocklist(dir)
	a := netip.MustParseAddr("198.51.100.9")
	b.Add(a, "", time.Now())
	if had, err := b.Remove(a); err != nil || !had {
		t.Fatalf("remove: had=%v err=%v", had, err)
	}
	if b.Has(a) {
		t.Error("still blocked after being unblocked")
	}
	if had, _ := b.Remove(a); had {
		t.Error("removing an address that is not blocked reports that it was")
	}
	again, _ := NewBlocklist(dir)
	if len(again.List()) != 0 {
		t.Errorf("the unblock did not reach the file: %+v", again.List())
	}
}

// A blocked source's datagrams are dropped where it matters: before the
// decoder, so nothing is parsed, nothing is stored and nothing reaches the
// rules. This sends real datagrams at a real socket rather than calling
// Handle, because the drop is in the receive loop and that is the thing
// worth testing.
func TestBlockedDatagramsNeverReachTheDecoder(t *testing.T) {
	sink := &countSink{}
	c := New(sink)
	b, err := NewBlocklist(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c.Block = b

	l, err := c.Listen("127.0.0.1:0", "sflow")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Serve(ctx, l)

	conn, err := net.Dial("udp", l.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Something the decoder will refuse either way: what is being measured
	// is whether it was offered the datagram at all, which Undecoded says.
	junk := []byte{9, 9, 9, 9, 1, 2, 3, 4}
	send := func() {
		if _, err := conn.Write(junk); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(what string, get func() uint64, want uint64) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if get() >= want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("%s reached %d, want %d", what, get(), want)
	}

	send()
	wait("packets", l.Packets.Load, 1)
	wait("undecoded", l.Undecoded.Load, 1)
	if n := l.Dropped.Load(); n != 0 {
		t.Fatalf("nothing is blocked yet, but %d datagrams were dropped", n)
	}

	// now block the sender and send the same thing again
	from := netip.MustParseAddr("127.0.0.1")
	if err := b.Add(from, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	before := l.Undecoded.Load()
	send()
	wait("dropped", l.Dropped.Load, 1)
	wait("packets", l.Packets.Load, 2)
	if after := l.Undecoded.Load(); after != before {
		t.Errorf("a blocked datagram reached the decoder: undecoded went %d -> %d", before, after)
	}
	if got := b.List()[0].Dropped; got != 1 {
		t.Errorf("the blocklist counted %d drops, want 1", got)
	}

	// and unblocking lets it through to the decoder again
	if _, err := b.Remove(from); err != nil {
		t.Fatal(err)
	}
	send()
	wait("undecoded", l.Undecoded.Load, before+1)
	if n := l.Dropped.Load(); n != 1 {
		t.Errorf("dropped is %d after unblocking; the drop should not have happened again", n)
	}
	if sink.records != 0 {
		t.Errorf("the junk datagram produced %d records", sink.records)
	}
}
