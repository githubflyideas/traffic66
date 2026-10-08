package enrich

import (
	"net/netip"
	"testing"

	"github.com/githubflyideas/traffic66/internal/flow"
)

// Exported TCP records carry the flags of all their packets: both
// directions of a normal connection read SYN+ACK. Both must agree on who
// the client is.
func TestRoleMergedFlags(t *testing.T) {
	pc, srv := netip.MustParseAddr("10.0.0.5"), netip.MustParseAddr("20.27.177.113")
	up := flow.Record{Src: pc, Dst: srv, SrcPort: 52114, DstPort: 443, Proto: 6, TCPFlags: 0x1b, Packets: 40}
	down := flow.Record{Src: srv, Dst: pc, SrcPort: 443, DstPort: 52114, Proto: 6, TCPFlags: 0x1b, Packets: 60}
	if !Role(&up) || Role(&down) {
		t.Fatalf("up=%v down=%v, want the PC as client both ways", Role(&up), Role(&down))
	}
	// both ports unknown: the opener seen in the first packet decides
	a := flow.Record{Src: pc, Dst: srv, SrcPort: 40000, DstPort: 50000, Proto: 6, TCPFlags: 0x1b, Packets: 9, Opener: 1}
	b := flow.Record{Src: srv, Dst: pc, SrcPort: 50000, DstPort: 40000, Proto: 6, TCPFlags: 0x1b, Packets: 9, Opener: -1}
	if !Role(&a) || Role(&b) {
		t.Fatalf("opener ignored: a=%v b=%v", Role(&a), Role(&b))
	}
	// a single sampled packet still reads its flags
	syn := flow.Record{Src: pc, Dst: srv, SrcPort: 50000, DstPort: 40000, Proto: 6, TCPFlags: 0x02, Packets: 1}
	if !Role(&syn) {
		t.Fatal("a lone SYN opens the connection")
	}
}
