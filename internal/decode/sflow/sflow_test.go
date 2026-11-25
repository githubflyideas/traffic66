package sflow

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

func TestRoundTrip(t *testing.T) {
	agent := netip.MustParseAddr("10.0.0.2")
	hdr, flen := encode.Frame(netip.MustParseAddr("10.10.3.27"), netip.MustParseAddr("203.0.113.66"), 50522, 443, 6, 0, 1400, 0x10)
	pkt := encode.SFlow(agent, 7, 99, 1000,
		[]encode.SFlowSample{{Seq: 5, SourceID: 3, Rate: 4096, Pool: 400000, In: 3, Out: 9, FrameLen: flen, Header: hdr[:64]}},
		[]encode.SFlowCounter{{Seq: 1, IfIndex: 3, Speed: 1e9, InOctets: 123456789, OutOctets: 987654321, InPkts: 10, OutPkts: 20}})
	now := time.Unix(1700000000, 0)
	d, err := Decode(pkt, now)
	if err != nil {
		t.Fatal(err)
	}
	if d.Agent != agent || d.SubAgent != 7 || d.Seq != 99 || len(d.Flows) != 1 || len(d.Counters) != 1 {
		t.Fatalf("got %+v", d)
	}
	f := d.Flows[0]
	if !f.Decoded || f.Rate != 4096 || f.Pool != 400000 || f.Record.Bytes != uint64(flen) || !f.Record.L2 || f.Record.DstPort != 443 || f.Record.InIf != 3 || f.Record.OutIf != 9 {
		t.Fatalf("flow %+v", f)
	}
	c := d.Counters[0]
	if c.IfIndex != 3 || c.InOctets != 123456789 || c.OutOctets != 987654321 || c.Speed != 1e9 {
		t.Fatalf("counter %+v", c)
	}
}

func TestTruncatedNeverPanics(t *testing.T) {
	hdr, flen := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 1, 2, 17, 0, 100, 0)
	pkt := encode.SFlow(netip.MustParseAddr("10.0.0.9"), 0, 1, 1,
		[]encode.SFlowSample{{Rate: 10, FrameLen: flen, Header: hdr}}, []encode.SFlowCounter{{IfIndex: 1}})
	for i := 0; i < len(pkt); i++ {
		Decode(pkt[:i], time.Now())
	}
}

func FuzzDecode(f *testing.F) {
	hdr, flen := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 1, 2, 17, 0, 100, 0)
	f.Add(encode.SFlow(netip.MustParseAddr("10.0.0.9"), 0, 1, 1, []encode.SFlowSample{{Rate: 10, FrameLen: flen, Header: hdr}}, nil))
	f.Fuzz(func(t *testing.T, b []byte) { Decode(b, time.Now()) })
}

// A switch's extended record names the VLAN of an untagged frame: the
// frame's own header is still 14 bytes.
func TestSwitchVLANIsNotATag(t *testing.T) {
	for _, tc := range []struct {
		tag  uint16
		want uint8
	}{{0, 14}, {100, 18}} {
		hdr, flen := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 1, 443, 6, tc.tag, 1500, 0x10)
		pkt := encode.SFlow(netip.MustParseAddr("10.0.0.9"), 0, 1, 1,
			[]encode.SFlowSample{{Rate: 10, FrameLen: flen, Header: hdr[:64], SwitchVLAN: 100}}, nil)
		d, err := Decode(pkt, time.Now())
		if err != nil || len(d.Flows) != 1 {
			t.Fatal(err)
		}
		r := d.Flows[0].Record
		if r.VLAN != 100 || r.L2Hdr != tc.want {
			t.Errorf("tag %d: VLAN %d L2Hdr %d, want 100 and %d", tc.tag, r.VLAN, r.L2Hdr, tc.want)
		}
	}
}
