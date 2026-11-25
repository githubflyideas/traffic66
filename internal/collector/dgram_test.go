package collector

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// A datagram the network delivers twice is counted once. A new datagram
// whose samples repeat numbers already seen, as after an agent restarts its
// sample sequence, is new traffic and counted.
func TestRepeatedDatagramAndRestartedSamples(t *testing.T) {
	sink := &countSink{}
	c := New(sink)
	agent := netip.MustParseAddr("192.0.2.44")
	hdr, flen := encode.Frame(netip.MustParseAddr("10.44.0.1"), netip.MustParseAddr("9.9.9.9"), 51000, 443, 6, 0, 1500, 0)
	dg := func(dseq, sseq uint32) []byte {
		return encode.SFlow(agent, 0, dseq, 1000, []encode.SFlowSample{{Seq: sseq, SourceID: 3, Rate: 100, Pool: 100 * sseq, FrameLen: flen, Header: hdr}}, nil)
	}
	now := time.Now()
	for i := uint32(1); i <= 10; i++ {
		b := dg(i, i)
		c.Handle(b, agent, now)
		if i == 5 {
			c.Handle(b, agent, now) // delivered twice
		}
	}
	if sink.records != 10 {
		t.Fatalf("a repeated datagram: %d records, want 10", sink.records)
	}
	for i := uint32(1); i <= 10; i++ { // the agent starts its samples over
		c.Handle(dg(100+i, i), agent, now)
	}
	if sink.records != 20 {
		t.Errorf("samples numbered again in new datagrams: %d records, want 20", sink.records)
	}
	// the agent restarts: datagrams numbered from 1 again, with a new uptime
	for i := uint32(1); i <= 10; i++ {
		c.Handle(encode.SFlow(agent, 0, i, 5, []encode.SFlowSample{{Seq: i, SourceID: 3, Rate: 100, Pool: 100 * i, FrameLen: flen, Header: hdr}}, nil), agent, now)
	}
	if sink.records != 30 {
		t.Errorf("after a restart of the agent: %d records, want 30", sink.records)
	}
}
