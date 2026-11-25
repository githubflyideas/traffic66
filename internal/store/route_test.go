package store

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// routeFixture is ten days of conversations, one row per conversation
// per hour: IPv4 and IPv6, two exporters, a duplicate copy of one
// conversation, a threat, several VLANs. Every grouped extra (country,
// application, direction) is constant within a conversation, so the
// any_value of a group is the same value on every path.
func routeFixture(base time.Time) []Row {
	type conv struct {
		client, server string
		cli4, srv4     uint32
		port           uint16
		proto          uint8
		app            string
		exp            string
		cc             string
		asn            uint32
		threat         string
		dup            bool
	}
	convs := []conv{
		{"198.51.100.20", "203.0.113.10", 0xC6336414, 0xCB00710A, 443, 6, "HTTPS", "10.0.0.1", "US", 64500, "", false},
		{"198.51.100.21", "203.0.113.10", 0xC6336415, 0xCB00710A, 443, 6, "HTTPS", "10.0.0.1", "US", 64500, "", false},
		{"198.51.100.20", "203.0.113.11", 0xC6336414, 0xCB00710B, 53, 17, "DNS", "10.0.0.1", "JP", 64501, "", false},
		{"2001:db8:2::20", "2001:db8:1::10", 0, 0, 443, 6, "HTTPS", "10.0.0.2", "US", 64500, "", false},
		{"198.51.100.22", "203.0.113.12", 0xC6336416, 0xCB00710C, 25, 6, "SMTP", "10.0.0.2", "DE", 64502, "scanner", false},
		{"198.51.100.20", "203.0.113.10", 0xC6336414, 0xCB00710A, 443, 6, "HTTPS", "10.0.0.2", "US", 64500, "", true},
		{"2001:db8:2::21", "2001:db8:1::10", 0, 0, 53, 17, "DNS", "10.0.0.2", "JP", 64501, "", false},
		{"198.51.100.21", "203.0.113.11", 0xC6336415, 0xCB00710B, 443, 6, "HTTPS", "10.0.0.1", "US", 64500, "", false},
	}
	var rows []Row
	for h := 0; h < 240; h++ {
		for i, c := range convs {
			wire := uint64(1000*(i+1) + (h%24)*10 + h)
			seg := "seg-a"
			if c.exp == "10.0.0.2" {
				seg = "seg-b"
			}
			rows = append(rows, Row{
				TS:  base.Add(time.Duration(h)*time.Hour + time.Duration((i*7)%45)*time.Minute),
				Src: c.client, Dst: c.server, Client: c.client, Server: c.server,
				Cli4: c.cli4, Srv4: c.srv4, SvcPort: c.port, Proto: c.proto, App: c.app,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: wire/100 + 1, Flows: uint32(1 + h%3),
				Exporter: c.exp, InIf: 1, OutIf: 2, Dir: DirOutbound, PeerCC: c.cc,
				PeerASN: c.asn, PeerOrg: "Org", Segment: seg, Encap: uint8(i % 2),
				VLAN: uint16(i % 3), Threat: c.threat, Dup: c.dup, CliInt: true,
			})
		}
	}
	return rows
}

func topDump(rows []TopRow, peers bool) string {
	var lines []string
	for _, r := range rows {
		p := ""
		if peers {
			p = fmt.Sprintf("|%d", r.Peers)
		}
		lines = append(lines, fmt.Sprintf("%s|%s|%s|%s|%d|%d|%d|%d%s|%v",
			r.Key, r.Key2, r.Key3, r.Extra, r.Bytes, r.Wire, r.Pkts, r.Flows, p, r.Int))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// A range the flow records no longer cover is answered from the
// summaries whole, and the answer is the one the records gave for the
// buckets that cover the range: the same Top in every dimension, the
// same totals, the same series. The records' answers are taken first,
// with a filter that excludes nothing but keeps the summaries out of
// it; then retention moves the records' start past the range, and
// the summaries answer.
func TestRollupRouteMatchesRecords(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.stopFold()
	if err := s.AppendRows(routeFixture(base)); err != nil {
		t.Fatal(err)
	}
	s.wmu.Lock()
	if err := s.rebuildRange(base, base.Add(10*24*time.Hour)); err != nil {
		s.wmu.Unlock()
		t.Fatal(err)
	}
	s.wmu.Unlock()
	// sealed a day at a time, as the pipeline seals an hour at a time:
	// retention lets go of whole files, so the files must be small
	// next to the seven days the records are kept
	for d := 1; d <= 9; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.foldOnce(base.Add(10 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}

	r := Query{From: base.Add(27*time.Hour + 37*time.Minute), To: base.Add(32*time.Hour + 12*time.Minute)}
	ef, et := hourRange(r.From, r.To)
	neg := []Filter{{Field: "country", Value: "ZZ", Neg: true}}
	rNeg := Query{From: r.From, To: r.To, Filters: neg}
	expNeg := Query{From: ef, To: et, Filters: neg}

	dims := []string{"client", "server", "service", "conv", "app", "port", "country", "asn", "proto", "dir", "exporter", "segment", "encap", "threat", "vlan"}
	// the records' answers: over the range itself for the dimensions
	// the summaries read by the minute or exactly, over the covering
	// buckets for the ones they keep by the hour
	wantTop := map[string]string{}
	for _, dim := range dims {
		// every summary is kept by the hour, so the records' answer to
		// compare with is the one over the covering buckets
		rows, err := s.TopNBy(expNeg, dim, 20, "wire")
		if err != nil {
			t.Fatalf("%s: %v", dim, err)
		}
		// the summaries by address report no peers, as they always
		// have, and a server's country is not among them either;
		// services count peers approximately (checked below)
		if dim == "server" {
			for i := range rows {
				rows[i].Extra = ""
			}
		}
		wantTop[dim] = topDump(rows, false)
		if src, _, _ := s.TopSource(r, dim); src != "detail" {
			t.Fatalf("%s: source %q before retention, want detail", dim, src)
		}
	}
	// the top service's peers, approximately (two clients use it)
	svcDetail, err := s.TopNBy(expNeg, "service", 20, "wire")
	if err != nil {
		t.Fatal(err)
	}
	var svcPeers uint64
	for _, row := range svcDetail {
		if row.Key == "203.0.113.10" {
			svcPeers = row.Peers
		}
	}
	if svcPeers == 0 {
		t.Fatal("the records count no peers for the top service")
	}
	totR, err := s.Totals(rNeg)
	if err != nil {
		t.Fatal(err)
	}
	totExp, err := s.Totals(expNeg)
	if err != nil {
		t.Fatal(err)
	}
	seApp, err := s.SeriesByApp(rNeg, 8)
	if err != nil {
		t.Fatal(err)
	}
	// by client the summaries are hourly, so their series covers the
	// covering buckets: the records' answer over those is the one to
	// compare with
	seClient, err := s.SeriesBy(expNeg, "client", "wire", 8)
	if err != nil {
		t.Fatal(err)
	}

	// retention moves the records' start to day 3; the range is day 1
	if err := s.Retain(base.Add(10 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	// the records' start is the first row of the first file that
	// straddles the cut — seals end on the hour, so it sits an hour
	// before the seven days, not on them
	if ds := s.DetailStart(); !ds.Equal(base.Add(71 * time.Hour)) {
		t.Fatalf("records start %v, want %v", ds, base.Add(71*time.Hour))
	}
	for _, dim := range dims {
		src, sf, st := s.TopSource(r, dim)
		if src != "rollup" {
			t.Fatalf("%s: source %q after retention, want rollup", dim, src)
		}
		if !sf.Equal(ef) || !st.Equal(et) {
			t.Fatalf("%s: covered range %v..%v, want %v..%v", dim, sf, st, ef, et)
		}
		rows, err := s.TopNBy(r, dim, 20, "wire")
		if err != nil {
			t.Fatalf("%s: %v", dim, err)
		}
		if got := topDump(rows, false); got != wantTop[dim] {
			t.Errorf("%s:\nsummaries:\n%s\nrecords:\n%s", dim, got, wantTop[dim])
		}
	}
	svcRoll, err := s.TopNBy(r, "service", 20, "wire")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range svcRoll {
		if row.Key == "203.0.113.10" {
			d := int64(row.Peers) - int64(svcPeers)
			if d < -1 || d > 1 {
				t.Errorf("service peers from summaries %d, records counted %d", row.Peers, svcPeers)
			}
		}
	}
	tot, err := s.Totals(r)
	if err != nil {
		t.Fatal(err)
	}
	if tot.Bytes != totR.Bytes || tot.Wire != totR.Wire || tot.Pkts != totR.Pkts || tot.Flows != totR.Flows {
		t.Errorf("totals from summaries %+v, records %+v", tot, totR)
	}
	if tot.Countries != totExp.Countries || tot.Hosts != totExp.Hosts || tot.Peers != totExp.Peers {
		t.Errorf("counts from summaries c%d h%d p%d, records c%d h%d p%d",
			tot.Countries, tot.Hosts, tot.Peers, totExp.Countries, totExp.Hosts, totExp.Peers)
	}
	seApp2, err := s.SeriesByApp(r, 8)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(seApp2.Names) != fmt.Sprint(seApp.Names) || fmt.Sprint(seApp2.Totals) != fmt.Sprint(seApp.Totals) ||
		fmt.Sprint(seApp2.Values) != fmt.Sprint(seApp.Values) {
		t.Errorf("series by app differs:\n%v %v\n%v %v", seApp2.Names, seApp2.Totals, seApp.Names, seApp.Totals)
	}
	seClient2, err := s.SeriesBy(r, "client", "wire", 8)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(seClient2.Names) != fmt.Sprint(seClient.Names) || fmt.Sprint(seClient2.Totals) != fmt.Sprint(seClient.Totals) {
		t.Errorf("series by client differs:\n%v %v\n%v %v", seClient2.Names, seClient2.Totals, seClient.Names, seClient.Totals)
	}
}
