package packet

import (
	"net/netip"
	"testing"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/flow"
)

func TestEthernetVLANTCP(t *testing.T) {
	src, dst := netip.MustParseAddr("10.1.2.3"), netip.MustParseAddr("198.51.100.7")
	hdr, flen := encode.Frame(src, dst, 50000, 443, 6, 120, 1500, 0x18)
	var r flow.Record
	if !Ethernet(hdr, &r) {
		t.Fatal("not parsed")
	}
	if r.Src != src || r.Dst != dst || r.SrcPort != 50000 || r.DstPort != 443 || r.Proto != 6 || r.VLAN != 120 || r.TCPFlags != 0x18 {
		t.Fatalf("got %+v", r)
	}
	if flen != 14+4+1500+4 {
		t.Fatalf("frame len %d", flen)
	}
}

func TestEncapDetection(t *testing.T) {
	a, b := netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2")
	cases := []struct {
		proto uint8
		dport uint16
		want  flow.Encap
	}{
		{47, 0, flow.EncapGRE}, {4, 0, flow.EncapIPIP}, {41, 0, flow.Encap6in4},
		{17, 4789, flow.EncapVXLAN}, {17, 6081, flow.EncapGENEVE}, {17, 53, flow.EncapNone},
	}
	for _, c := range cases {
		hdr, _ := encode.Frame(a, b, 1000, c.dport, c.proto, 0, 200, 0)
		var r flow.Record
		Ethernet(hdr, &r)
		if r.Encap != c.want {
			t.Errorf("proto %d port %d: got %v want %v", c.proto, c.dport, r.Encap, c.want)
		}
	}
}

func TestMPLS(t *testing.T) {
	// dst, src MAC, MPLS unicast, two labels (second bottom-of-stack), IPv4 UDP.
	ip, _ := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 1, 2, 17, 0, 100, 0)
	frame := append([]byte{}, ip[:12]...)
	frame = append(frame, 0x88, 0x47, 0x00, 0x01, 0x00, 0x40, 0x00, 0x02, 0x01, 0x40)
	frame = append(frame, ip[14:]...)
	var r flow.Record
	if !Ethernet(frame, &r) || r.Encap != flow.EncapMPLS || r.DstPort != 2 {
		t.Fatalf("got %+v", r)
	}
}

func TestIPv6FragmentAndExt(t *testing.T) {
	b := make([]byte, 40+8+8+4)
	b[0] = 0x60
	b[6] = 0 // hop-by-hop
	copy(b[8:24], netip.MustParseAddr("2001:db8::1").AsSlice())
	copy(b[24:40], netip.MustParseAddr("2001:db8::2").AsSlice())
	b[40] = 44 // next: fragment
	b[41] = 0
	b[48] = 17 // fragment header next: UDP, offset 0
	b[56], b[57], b[58], b[59] = 0x13, 0x88, 0x00, 0x35
	var r flow.Record
	if !IPv6(b, &r) || r.Proto != 17 || r.SrcPort != 5000 || r.DstPort != 53 {
		t.Fatalf("got %+v", r)
	}
}

func TestTruncatedNeverPanics(t *testing.T) {
	hdr, _ := encode.Frame(netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.0.2"), 1, 2, 6, 5, 100, 0)
	for i := 0; i < len(hdr); i++ {
		var r flow.Record
		Ethernet(hdr[:i], &r)
	}
}
