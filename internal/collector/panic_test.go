package collector

import (
	"net/netip"
	"strings"
	"testing"
	"time"
)

// One packet that makes a decoder panic must not take the listener, and
// with it the whole analyser, down. Anyone who can reach the port can send
// it, so this is the difference between a logged bug and an outage.
func TestOnePacketCannotKillTheListener(t *testing.T) {
	// A collector with no agent map: recording the sender panics, the way
	// a decoder bug on a malformed packet would.
	c := &Collector{}
	l := &Listener{Addr: "127.0.0.1:0"}
	pkt := []byte{0, 0, 0, 5, 1, 2, 3, 4} // sFlow version 5
	if ok := c.handleSafely(l, pkt, netip.MustParseAddr("10.0.0.1"), time.Now()); ok {
		t.Error("handleSafely reported success although the decode panicked")
	}
	if n := l.Panics.Load(); n != 1 {
		t.Errorf("Panics = %d, want 1", n)
	}
	msg, _ := l.LastErr.Load().(string)
	if !strings.Contains(msg, "panic decoding 8 bytes from 10.0.0.1") {
		t.Errorf("LastErr = %q, want the sender and the length in it", msg)
	}
	// and the next packet is still handled
	if ok := c.handleSafely(l, pkt, netip.MustParseAddr("10.0.0.2"), time.Now()); ok {
		t.Error("the second packet reported success")
	}
	if n := l.Panics.Load(); n != 2 {
		t.Errorf("Panics = %d after two bad packets, want 2", n)
	}
}
