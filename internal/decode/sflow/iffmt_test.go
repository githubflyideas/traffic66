package sflow

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// An interface field in a flow sample is a format and a value together.
// Only format 0 is an ifIndex: 1 means the device discarded the packet and
// the value is its reason code, 2 means several interfaces and the value
// is how many. Reading all three as an ifIndex put reason codes and
// interface counts into the per-interface views and into the bill.
func TestInterfaceFormatIsNotAnIfIndex(t *testing.T) {
	agent := netip.MustParseAddr("10.0.0.2")
	hdr, flen := encode.Frame(netip.MustParseAddr("10.10.3.27"), netip.MustParseAddr("203.0.113.66"), 50522, 443, 6, 0, 1400, 0x10)
	const (
		discarded = 1 << 30 // format 1: the value is a drop reason
		several   = 2 << 30 // format 2: the value is how many interfaces
	)
	for _, tc := range []struct {
		name            string
		in, out         uint32
		wantIn, wantOut uint32
	}{
		{"both are interfaces", 3, 9, 3, 9},
		{"dropped by an ACL", 3, discarded | 300, 3, 0},
		{"dropped for congestion", 3, discarded | 2, 3, 0},
		{"flooded to several ports", 3, several | 4, 3, 0},
		{"ingress unknown too", several | 2, 9, 0, 9},
	} {
		pkt := encode.SFlow(agent, 7, 99, 1000,
			[]encode.SFlowSample{{Seq: 5, SourceID: 3, Rate: 4096, Pool: 400000, In: tc.in, Out: tc.out, FrameLen: flen, Header: hdr[:64]}}, nil)
		d, err := Decode(pkt, time.Unix(1700000000, 0))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(d.Flows) != 1 {
			t.Fatalf("%s: %d flows", tc.name, len(d.Flows))
		}
		r := d.Flows[0].Record
		if r.InIf != tc.wantIn || r.OutIf != tc.wantOut {
			t.Errorf("%s: in=%d out=%d, want in=%d out=%d", tc.name, r.InIf, r.OutIf, tc.wantIn, tc.wantOut)
		}
		// the packet is still counted: it crossed the ingress interface
		// whatever the device did with it afterwards
		if r.Bytes != uint64(flen) || r.Packets != 1 {
			t.Errorf("%s: the sample stopped being counted: %d bytes %d packets", tc.name, r.Bytes, r.Packets)
		}
	}
}
