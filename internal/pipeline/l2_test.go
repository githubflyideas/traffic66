package pipeline

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/store"
)

func l2Total(t *testing.T, cfg Config, r flow.Record) store.Totals {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30, RollupDays: 400})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	p := New(cfg, st, enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	p.Ingest([]flow.Record{r})
	p.FlushRows()
	p.FlushRollups()
	tot, err := st.Totals(store.Query{From: r.Start.Add(-time.Hour), To: r.End.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	return tot
}

func l2Record(l2 bool, vlan uint16) flow.Record {
	ts := time.Now().UTC().Truncate(time.Minute).Add(-10 * time.Minute)
	return flow.Record{Start: ts, End: ts.Add(30 * time.Second), Src: netip.MustParseAddr("10.1.1.1"), Dst: netip.MustParseAddr("198.51.100.9"),
		SrcPort: 40000, DstPort: 443, Proto: 6, Bytes: 5000, Packets: 10, Mult: 1, SamplingKnown: true, L2: l2, VLAN: vlan,
		Exporter: netip.MustParseAddr("10.0.0.1"), Source: flow.SrcNetFlow5}
}

// -l2-overhead 0 means no overhead, not the default of 18.
func TestL2OverheadZeroIsKept(t *testing.T) {
	if tot := l2Total(t, Config{L2Overhead: 0, L2Given: true}, l2Record(false, 0)); tot.Wire != 5000 {
		t.Fatalf("wire %d, want 5000", tot.Wire)
	}
	if tot := l2Total(t, Config{}, l2Record(false, 0)); tot.Wire != 5180 {
		t.Fatalf("unset: wire %d, want 5180", tot.Wire)
	}
}

// A sampled Ethernet frame with an 802.1Q tag: the tag is link layer.
func TestVLANTagNotCountedAsIP(t *testing.T) {
	if tot := l2Total(t, Config{}, l2Record(true, 100)); tot.Bytes != 5000-10*22 || tot.Wire != 5000 {
		t.Fatalf("bytes %d wire %d, want 4780 and 5000", tot.Bytes, tot.Wire)
	}
	if tot := l2Total(t, Config{}, l2Record(true, 0)); tot.Bytes != 5000-10*18 {
		t.Fatalf("untagged: bytes %d, want 4820", tot.Bytes)
	}
}

// The header length read from the frame wins over the VLAN reported beside
// it (sFlow's switch record) and counts every QinQ tag and MPLS label.
func TestL2HeaderFromFrame(t *testing.T) {
	for _, c := range []struct {
		vlan uint16
		hdr  uint8
		want uint64
	}{{100, 14, 5000 - 10*18}, {100, 18, 5000 - 10*22}, {100, 22, 5000 - 10*26}, {0, 26, 5000 - 10*30}} {
		r := l2Record(true, c.vlan)
		r.L2Hdr = c.hdr
		if tot := l2Total(t, Config{}, r); tot.Bytes != c.want || tot.Wire != 5000 {
			t.Errorf("vlan %d header %d: bytes %d wire %d, want %d and 5000", c.vlan, c.hdr, tot.Bytes, tot.Wire, c.want)
		}
	}
}
