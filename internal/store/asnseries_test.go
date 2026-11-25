package store

import (
	"testing"
	"time"
)

// The two network charts are drawn from asn_src and asn_dst. Traffic whose
// peer has no AS number has to count, as AS 0: an installation without a
// good AS database has minutes that carry nothing else, and leaving those
// minutes out drew the chart as a run of blocks with gaps between them
// while the AS table under it showed the same traffic as "unknown".
func TestASSeriesCountsUnknownAS(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	to := time.Now().UTC().Truncate(time.Minute)
	from := to.Add(-30 * time.Minute)
	// Every minute carries traffic. Only every third minute has a peer with
	// an AS number; the rest is traffic the AS database could not place.
	var rows []Row
	for i := 0; i < 30; i++ {
		ts := from.Add(time.Duration(i)*time.Minute + 10*time.Second)
		r := Row{TS: ts, Src: "203.0.113.9", Dst: "10.1.1.5", Client: "10.1.1.5", Server: "203.0.113.9",
			Cli4: 0x0A010105, Srv4: 0xCB007109, Dir: DirOutbound, App: "HTTPS", SvcPort: 443, Proto: 6,
			Wire: 1_000_000, Bytes: 1_000_000, Pkts: 800, Flows: 1, Exporter: "10.0.0.1", InIf: 1, CliInt: true}
		if i%3 == 0 {
			r.PeerASN, r.PeerOrg = 64501, "Example Net"
		}
		// the other half of the same conversation, so both charts have data
		b := r
		b.Src, b.Dst = r.Dst, r.Src
		b.Wire, b.Bytes, b.Pkts = r.Wire/4, r.Bytes/4, r.Pkts/4
		rows = append(rows, r, b)
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	q := Query{From: from, To: to}
	for _, by := range []string{"asn_src", "asn_dst"} {
		se, err := s.SeriesBy(q, by, "wire", 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(se.Names) == 0 {
			t.Fatalf("%s: no series", by)
		}
		var unknown, known bool
		for _, n := range se.Names {
			switch {
			case n == "0\t":
				unknown = true
			case n == "64501\tExample Net":
				known = true
			}
		}
		if !unknown || !known {
			t.Fatalf("%s: series %q, want both the placed AS and AS 0", by, se.Names)
		}
		// no bucket of a minute that carried traffic may be empty
		for i := range se.Times {
			sum := 0.0
			for _, v := range se.Values {
				sum += v[i]
			}
			if sum == 0 {
				t.Errorf("%s: bucket %d of %d is empty although that minute carried traffic", by, i, len(se.Times))
			}
		}
	}
}
