package sflow

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// Hosts that sample after receive offload report frames over 64 KB, which
// are real; a frame of megabytes is not.
func TestFrameLengthLimit(t *testing.T) {
	agent := netip.MustParseAddr("192.0.2.1")
	hdr, _ := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 40000, 443, 6, 0, 1400, 0x10)
	for _, c := range []struct {
		frame uint32
		ok    bool
	}{{1518, true}, {100_000, true}, {512 << 10, true}, {4 << 20, false}} {
		pkt := encode.SFlow(agent, 0, 1, 1000, []encode.SFlowSample{{Seq: 1, Rate: 100, Pool: 100, In: 1, Out: 2, FrameLen: c.frame, Header: hdr[:64]}}, nil)
		d, err := Decode(pkt, time.Now())
		got := err == nil && len(d.Flows) == 1 && d.Flows[0].Record.Bytes == uint64(c.frame)
		if got != c.ok {
			t.Fatalf("frame %d: accepted %v, want %v (err %v)", c.frame, got, c.ok, err)
		}
	}
}
