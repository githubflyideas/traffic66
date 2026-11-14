package accuracy

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Ground truth: unsampled exports of known traffic must be stored to the
// byte, IP bytes and link-layer bytes both, whatever tags the frames carry.

func totals(t *testing.T, st *store.Store, exporter string, from, to time.Time) store.Totals {
	t.Helper()
	tot, err := st.Totals(store.Query{From: from, To: to, Filters: []store.Filter{{Field: "exporter", Value: exporter}}})
	if err != nil {
		t.Fatal(err)
	}
	return tot
}

// tagged rebuilds an untagged frame header with extra bytes between the
// MAC addresses and the IP header: VLAN tags, or MPLS labels with their
// own EtherType.
func tagged(hdr []byte, etype uint16, extra []byte) []byte {
	out := append([]byte{}, hdr[:12]...)
	if etype != 0 {
		out = append(out, byte(etype>>8), byte(etype))
		out = append(out, extra...)
		return append(out, hdr[14:]...)
	}
	out = append(out, extra...)
	return append(out, hdr[12:]...)
}

func TestGroundTruthSFlowFrames(t *testing.T) {
	cl, sv := clients[0], servers[0]
	untagged, _ := encode.Frame(cl, sv, 40000, 443, 6, 0, 1000, 0x18)
	dot1q, _ := encode.Frame(cl, sv, 40000, 443, 6, 100, 1000, 0x18)
	qinq := tagged(untagged, 0, []byte{0x88, 0xa8, 0, 10, 0x81, 0x00, 0, 20})
	mpls := tagged(untagged, 0x8847, []byte{0x00, 0x01, 0x00, 0x40, 0x00, 0x01, 0x11, 0x40})
	for _, c := range []struct {
		name string
		hdr  []byte
		l2   int // link-layer header bytes before the IP header
		sw   uint32
	}{
		{"untagged", untagged, 14, 0},
		{"untagged, switch record VLAN 100", untagged, 14, 100},
		{"802.1Q", dot1q, 18, 0},
		{"802.1Q, switch record VLAN 100", dot1q, 18, 100},
		{"QinQ", qinq, 22, 0},
		{"MPLS, two labels", mpls, 22, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			col, p, st := setup(t)
			agent := netip.MustParseAddr("10.0.0.9")
			start := time.Now().UTC().Truncate(time.Minute).Add(-20 * time.Minute)
			const n, ip = 500, 1000
			frame := uint32(c.l2 + ip + 4)
			var batch []encode.SFlowSample
			for i := 0; i < n; i++ {
				batch = append(batch, encode.SFlowSample{Seq: uint32(i + 1), SourceID: 3, Rate: 1, Pool: uint32(i + 1), In: 3, Out: 4, FrameLen: frame, Header: c.hdr, SwitchVLAN: c.sw})
				if len(batch) == 10 {
					col.Handle(encode.SFlow(agent, 0, uint32(i/10+1), 1000, batch, nil), agent, start.Add(time.Duration(i)*time.Second))
					batch = batch[:0]
				}
			}
			p.FlushRows()
			p.FlushRollups()
			tot := totals(t, st, agent.String(), start.Add(-time.Hour), start.Add(time.Hour))
			if tot.Bytes != n*ip || tot.Wire != uint64(n)*uint64(frame) || tot.Pkts != n {
				t.Errorf("IP bytes %d, wire %d, packets %d; want %d, %d, %d", tot.Bytes, tot.Wire, tot.Pkts, n*ip, uint64(n)*uint64(frame), n)
			}
		})
	}
}

func TestGroundTruthNetFlow5(t *testing.T) {
	col, p, st := setup(t)
	exp := netip.MustParseAddr("10.0.0.5")
	start := time.Now().UTC().Truncate(time.Minute).Add(-20 * time.Minute)
	var bytes, pkts uint64
	var recs []encode.V5Record
	for i := 0; i < 30; i++ {
		pk, by := uint32(10+i), uint32((10+i)*(100+7*i))
		bytes += uint64(by)
		pkts += uint64(pk)
		recs = append(recs, encode.V5Record{Src: clients[i%3], Dst: servers[i%2], In: 1, Out: 2, Packets: pk, Bytes: by,
			First: 1000, Last: 20000, SrcPort: uint16(40000 + i), DstPort: 443, Proto: 6})
	}
	col.Handle(encode.V5(60000, uint32(start.Unix()), 1, 0, recs), exp, start)
	p.FlushRows()
	p.FlushRollups()
	tot := totals(t, st, exp.String(), start.Add(-time.Hour), start.Add(time.Hour))
	if tot.Bytes != bytes || tot.Wire != bytes+18*pkts || tot.Pkts != pkts {
		t.Errorf("IP bytes %d, wire %d, packets %d; want %d, %d, %d", tot.Bytes, tot.Wire, tot.Pkts, bytes, bytes+18*pkts, pkts)
	}
}
