package encode

import (
	"encoding/binary"
	"net/netip"
	"testing"

	"github.com/githubflyideas/traffic66/internal/wire"
)

var (
	ea1 = netip.MustParseAddr("192.0.2.1")
	ea2 = netip.MustParseAddr("198.51.100.2")
	ea6 = netip.MustParseAddr("2001:db8::1")
	eb6 = netip.MustParseAddr("2001:db8::2")
)

func TestUAndA(t *testing.T) {
	if got := U(0x0102, 4); len(got) != 4 || got[2] != 0x01 || got[3] != 0x02 || got[0] != 0 || got[1] != 0 {
		t.Errorf("U(0x0102, 4) = %x", got)
	}
	if got := U(0x1FF, 1); len(got) != 1 || got[0] != 0xFF {
		t.Errorf("U(0x1FF, 1) = %x, want the low byte only", got)
	}
	if got := A(ea1); len(got) != 4 {
		t.Errorf("A(v4) = %d bytes", len(got))
	}
	if got := A(ea6); len(got) != 16 {
		t.Errorf("A(v6) = %d bytes", len(got))
	}
}

func TestFrameV4TCP(t *testing.T) {
	hdr, total := Frame(ea1, ea2, 12345, 443, 6, 0, 100, 0x18)
	if total != 14+100+4 {
		t.Errorf("frame length = %d", total)
	}
	if binary.BigEndian.Uint16(hdr[12:]) != 0x0800 {
		t.Errorf("ethertype = %x", hdr[12:])
	}
	if hdr[14] != 0x45 {
		t.Errorf("IP version/IHL = %x", hdr[14])
	}
	if got := binary.BigEndian.Uint16(hdr[16:]); got != 100 {
		t.Errorf("IP total length = %d", got)
	}
	if hdr[14+9] != 6 {
		t.Errorf("protocol = %d", hdr[14+9])
	}
	if ip := netip.AddrFrom4([4]byte(hdr[26:30])); ip != ea1 {
		t.Errorf("src = %v", ip)
	}
	if ip := netip.AddrFrom4([4]byte(hdr[30:34])); ip != ea2 {
		t.Errorf("dst = %v", ip)
	}
	if binary.BigEndian.Uint16(hdr[34:]) != 12345 || binary.BigEndian.Uint16(hdr[36:]) != 443 {
		t.Errorf("TCP ports = %d/%d", binary.BigEndian.Uint16(hdr[34:]), binary.BigEndian.Uint16(hdr[36:]))
	}
	if hdr[34+13] != 0x18 {
		t.Errorf("TCP flags = %x", hdr[34+13])
	}
}

func TestFrameVLANAndMinimum(t *testing.T) {
	hdr, total := Frame(ea1, ea2, 53, 53, 17, 42, 60, 0)
	if binary.BigEndian.Uint16(hdr[12:]) != 0x8100 || binary.BigEndian.Uint16(hdr[14:]) != 42 {
		t.Errorf("VLAN tag = %x %d", hdr[12:], binary.BigEndian.Uint16(hdr[14:]))
	}
	if binary.BigEndian.Uint16(hdr[16:]) != 0x0800 {
		t.Errorf("inner ethertype = %x", hdr[16:])
	}
	if total != 18+60+4 {
		t.Errorf("tagged frame length = %d", total)
	}
	// UDP length counts the IP payload only
	if got := binary.BigEndian.Uint16(hdr[18+20+4:]); got != 60-20 {
		t.Errorf("UDP length = %d", got)
	}
	_, total = Frame(ea1, ea2, 1, 2, 17, 0, 28, 0)
	if total != 64 {
		t.Errorf("tiny frame length = %d, want the 64-byte minimum", total)
	}
}

func TestFrameV6(t *testing.T) {
	hdr, total := Frame(ea6, eb6, 1000, 2000, 6, 0, 80, 0x02)
	if binary.BigEndian.Uint16(hdr[12:]) != 0x86DD {
		t.Errorf("ethertype = %x", hdr[12:])
	}
	if hdr[14]>>4 != 6 {
		t.Errorf("version nibble = %d", hdr[14]>>4)
	}
	if got := binary.BigEndian.Uint16(hdr[18:]); got != 80-40 {
		t.Errorf("payload length = %d", got)
	}
	if total != 14+80+4 {
		t.Errorf("frame length = %d", total)
	}
}

func TestV5Packet(t *testing.T) {
	recs := []V5Record{
		{Src: ea1, Dst: ea2, In: 3, Out: 4, Packets: 10, Bytes: 9000, SrcPort: 1111, DstPort: 80, TCPFlags: 0x1B, Proto: 6},
		{Src: ea2, Dst: ea1, Packets: 4, Bytes: 300, Proto: 17},
	}
	p := V5(1000, 2000, 7, 100, recs)
	if len(p) != 24+2*48 {
		t.Fatalf("packet = %d bytes", len(p))
	}
	r := wire.New(p)
	if v, n := r.U16(), r.U16(); v != 5 || n != 2 {
		t.Errorf("header version/count = %d/%d", v, n)
	}
	r.U32() // uptime
	if secs := r.U32(); secs != 2000 {
		t.Errorf("unix secs = %d", secs)
	}
	r.U32() // nanoseconds
	if seq := r.U32(); seq != 7 {
		t.Errorf("sequence = %d", seq)
	}
	r.U8() // engine type
	r.U8() // engine id
	if s := r.U16(); s != 1<<14|100 {
		t.Errorf("sampling = %x, want mode 1, rate 100", s)
	}
	if ip := r.IPv4(); ip != ea1 {
		t.Errorf("record src = %v", ip)
	}
	if ip := r.IPv4(); ip != ea2 {
		t.Errorf("record dst = %v", ip)
	}
	// an unsampled export carries no sampling field
	if p := V5(0, 0, 0, 1, nil); len(p) != 24 || binary.BigEndian.Uint16(p[22:]) != 0 {
		t.Errorf("unsampled header = %x", p)
	}
}

func TestSFlowHeader(t *testing.T) {
	hdr, _ := Frame(ea1, ea2, 1, 2, 6, 0, 60, 0x02)
	p := SFlow(ea1, 0, 9, 123456, []SFlowSample{{Seq: 1, SourceID: 5, Rate: 512, FrameLen: 64, Header: hdr}},
		[]SFlowCounter{{Seq: 2, IfIndex: 7, Speed: 1e9, InOctets: 1000, OutOctets: 2000, InPkts: 10, OutPkts: 20}})
	r := wire.New(p)
	if v := r.U32(); v != 5 {
		t.Fatalf("version = %d", v)
	}
	if at := r.U32(); at != 1 {
		t.Errorf("agent address type = %d, want IPv4", at)
	}
	if ip := r.IPv4(); ip != ea1 {
		t.Errorf("agent = %v", ip)
	}
	r.U32() // sub-agent
	if seq := r.U32(); seq != 9 {
		t.Errorf("sequence = %d", seq)
	}
	if up := r.U32(); up != 123456 {
		t.Errorf("uptime = %d", up)
	}
	if n := r.U32(); n != 2 {
		t.Errorf("sample count = %d, want 1 flow + 1 counter", n)
	}
	// first sample: the type is one packed u32 (enterprise 0, format 1)
	if typ := r.U32(); typ != 1 {
		t.Errorf("first sample type = %d, want 1 (flow sample)", typ)
	}
	r.U32() // sample length
	if seq := r.U32(); seq != 1 {
		t.Errorf("flow sample sequence = %d", seq)
	}
	if src := r.U32(); src != 5 {
		t.Errorf("data source = %d", src)
	}
	if rate := r.U32(); rate != 512 {
		t.Errorf("sampling rate = %d", rate)
	}
	if r.Err() != nil {
		t.Errorf("reader: %v", r.Err())
	}
	// an IPv6 agent is address type 2
	p = SFlow(ea6, 0, 0, 0, nil, nil)
	if binary.BigEndian.Uint32(p[4:]) != 2 {
		t.Errorf("v6 agent type = %d", binary.BigEndian.Uint32(p[4:]))
	}
}

func TestIPFIXMessage(t *testing.T) {
	m := NewIPFIX()
	m.AddTemplate(Template{ID: 257, Fields: []Field{{ID: 8, Len: 4}, {ID: 999, Len: 2, Ent: 12345}}})
	m.AddData(257, []Values{{A(ea1), U(7, 2)}})
	if m.DataRecords() != 1 {
		t.Fatalf("DataRecords = %d", m.DataRecords())
	}
	p := m.IPFIX(555, 0, 77)
	if v := binary.BigEndian.Uint16(p); v != 10 {
		t.Errorf("IPFIX version = %d", v)
	}
	if l := int(binary.BigEndian.Uint16(p[2:])); l != len(p) {
		t.Errorf("IPFIX length field = %d, packet is %d", l, len(p))
	}
	if dom := binary.BigEndian.Uint32(p[12:]); dom != 77 {
		t.Errorf("IPFIX domain = %d", dom)
	}
	// the enterprise field is marked with the high bit and carries its PEN
	found := false
	for i := 0; i+1 < len(p); i++ {
		if binary.BigEndian.Uint16(p[i:]) == 999|0x8000 {
			found = true
		}
	}
	if !found {
		t.Error("enterprise field id not found with the enterprise bit set")
	}
}
