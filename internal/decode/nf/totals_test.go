package nf

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// totalTmpl carries octetTotalCount/packetTotalCount and no delta counts,
// which is what an exporter that reports running totals sends.
var totalTmpl = encode.Template{ID: 310, Fields: []encode.Field{
	{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
	{ID: ieProtocol, Len: 1}, {ID: ieOctetTotal, Len: 8}, {ID: iePacketTotal, Len: 8},
	{ID: ieStartMs, Len: 8}, {ID: ieEndMs, Len: 8}, {ID: ieInIf, Len: 4},
}}

// noTimeTmpl carries the same totals but no flow times at all.
var noTimeTmpl = encode.Template{ID: 311, Fields: []encode.Field{
	{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
	{ID: ieProtocol, Len: 1}, {ID: ieOctetTotal, Len: 8}, {ID: iePacketTotal, Len: 8},
	{ID: ieInIf, Len: 4},
}}

func totRec(startMs, endMs, bytes, pkts uint64) encode.Values {
	return encode.Values{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2), encode.U(6, 1),
		encode.U(bytes, 8), encode.U(pkts, 8), encode.U(startMs, 8), encode.U(endMs, 8), encode.U(3, 4)}
}

// send one message carrying template 310 and the given records.
func sendTotals(t *testing.T, d *Decoder, seq uint32, at time.Time, recs ...encode.Values) []uint64 {
	t.Helper()
	m := encode.NewIPFIX()
	m.AddTemplate(totalTmpl)
	m.AddData(310, recs)
	res, err := d.Decode(m.IPFIX(uint32(at.Unix()), seq, 1), exp, at)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]uint64, 0, len(res.Records))
	for _, r := range res.Records {
		out = append(out, r.Bytes)
	}
	return out
}

// A flow still running is re-reported on the exporter's active timeout with
// a total that has grown. Only the growth is this report's traffic; adding
// the totals up counts the same bytes again on every report.
func TestOctetTotalIsCumulative(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true} // release records at once
	start := uint64(base.UnixMilli())
	var sum uint64
	// three reports of one flow: 1000, then 2500, then 4000 bytes in all
	for i, tot := range []uint64{1000, 2500, 4000} {
		at := base.Add(time.Duration(i) * time.Minute)
		got := sendTotals(t, d, uint32(i), at, totRec(start, uint64(at.UnixMilli()), tot, (tot/100)*2))
		if len(got) != 1 {
			t.Fatalf("report %d: %d records", i, len(got))
		}
		sum += got[0]
	}
	if sum != 4000 {
		t.Errorf("the flow's three reports added up to %d bytes, want 4000 (its final total)", sum)
	}
}

// An exporter that reports each interval separately gives every report its
// own start time, so nothing is subtracted and the totals are taken whole.
func TestOctetTotalPerIntervalIsNotSubtracted(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true} // release records at once
	var sum uint64
	for i := 0; i < 3; i++ {
		at := base.Add(time.Duration(i) * time.Minute)
		start := uint64(at.Add(-time.Minute).UnixMilli())
		got := sendTotals(t, d, uint32(i), at, totRec(start, uint64(at.UnixMilli()), 1000, 10))
		if len(got) != 1 || got[0] != 1000 {
			t.Fatalf("report %d: %v, want one record of 1000", i, got)
		}
		sum += got[0]
	}
	if sum != 3000 {
		t.Errorf("three separate flows added up to %d bytes, want 3000", sum)
	}
}

func noTimeReport(t *testing.T, d *Decoder, seq uint32, bytes, pkts uint64) uint64 {
	t.Helper()
	m := encode.NewIPFIX()
	m.AddTemplate(noTimeTmpl)
	m.AddData(311, []encode.Values{{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2),
		encode.U(6, 1), encode.U(bytes, 8), encode.U(pkts, 8), encode.U(3, 4)}})
	at := base.Add(time.Duration(seq) * time.Minute)
	res, err := d.Decode(m.IPFIX(uint32(at.Unix()), seq, 1), exp, at)
	if err != nil || len(res.Records) != 1 {
		t.Fatalf("got %d %v", len(res.Records), err)
	}
	return res.Records[0].Bytes
}

// Without a start time a flow re-reported with a growing running total is
// counted by its growth, not whole on every report.
func TestOctetTotalWithoutStartTime(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true} // release records at once
	var sum uint64
	for i, tot := range []uint64{1000, 2500, 4000} {
		sum += noTimeReport(t, d, uint32(i), tot, tot/100)
	}
	if sum != 4000 {
		t.Errorf("three reports added up to %d bytes, want 4000 (the final total)", sum)
	}
}

// Without a start time a total that shrank is a reset or a new flow with the
// same identity, and is taken whole.
func TestOctetTotalWithoutStartTimeReset(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true}
	if got := noTimeReport(t, d, 0, 5000, 50); got != 5000 {
		t.Fatalf("first report %d", got)
	}
	if got := noTimeReport(t, d, 1, 800, 8); got != 800 {
		t.Fatalf("after a reset %d, want 800", got)
	}
	if got := noTimeReport(t, d, 2, 1300, 13); got != 500 {
		t.Fatalf("after the reset the growth is %d, want 500", got)
	}
}

// A total that shrank means the exporter restarted its metering or reused
// the flow's identity; the difference would be meaningless.
func TestOctetTotalGoingBackwards(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true} // release records at once
	start := uint64(base.UnixMilli())
	got := sendTotals(t, d, 0, base, totRec(start, uint64(base.UnixMilli()), 5000, 50))
	if len(got) != 1 || got[0] != 5000 {
		t.Fatalf("first report %v", got)
	}
	at := base.Add(time.Minute)
	got = sendTotals(t, d, 1, at, totRec(start, uint64(at.UnixMilli()), 800, 8))
	if len(got) != 1 || got[0] != 800 {
		t.Fatalf("after a reset %v, want the total taken whole (800)", got)
	}
}

// Two different flows reported in the same message keep their own totals.
func TestOctetTotalPerFlow(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true} // release records at once
	start := uint64(base.UnixMilli())
	a := totRec(start, uint64(base.UnixMilli()), 1000, 10)
	b := totRec(start, uint64(base.UnixMilli()), 2000, 20)
	b[3] = encode.U(8080, 2) // a different destination port: another flow
	if got := sendTotals(t, d, 0, base, a, b); len(got) != 2 || got[0] != 1000 || got[1] != 2000 {
		t.Fatalf("first reports %v", got)
	}
	at := base.Add(time.Minute)
	a = totRec(start, uint64(at.UnixMilli()), 1500, 15)
	b = totRec(start, uint64(at.UnixMilli()), 2200, 22)
	b[3] = encode.U(8080, 2)
	if got := sendTotals(t, d, 1, at, a, b); len(got) != 2 || got[0] != 500 || got[1] != 200 {
		t.Fatalf("second reports %v, want [500 200]", got)
	}
}

// The table is bounded: it rotates rather than growing without end, and a
// flow still being reported keeps its total across a rotation.
func TestTotalsRotation(t *testing.T) {
	var tt totals
	k := totKey{sport: 1}
	if b, _ := tt.delta(k, 1000, 10, base); b != 1000 {
		t.Fatalf("first = %d", b)
	}
	// fill the current generation, forcing a rotation
	for i := 0; i < maxTotals; i++ {
		tt.delta(totKey{sport: uint16(i % 65535), dport: uint16(i / 65535), proto: 17}, 1, 1, base)
	}
	if b, _ := tt.delta(k, 1500, 15, base); b != 500 {
		t.Errorf("after a rotation = %d, want 500: the previous generation is still read", b)
	}
	if len(tt.cur) > maxTotals+2 {
		t.Errorf("current generation holds %d entries, cap is %d", len(tt.cur), maxTotals)
	}
	// an exporter too slow to fill the table still rotates on time
	tt.delta(k, 2000, 20, base.Add(totalsIdle+time.Minute))
	if len(tt.cur) != 1 {
		t.Errorf("after an idle rotation the current generation holds %d, want 1", len(tt.cur))
	}
}
