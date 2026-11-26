package store

import (
	"net/netip"
	"testing"
	"time"
)

// The billing figure of a set of addresses: a flow between two addresses
// of the same set is counted once in the total. Adding the per-address
// figures instead would count it twice — once as the sender's out, once
// as the receiver's in — and the bill would charge for it twice.
func TestBillingCountsInsideTrafficOnce(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)
	a, b, user := "203.0.113.10", "203.0.113.11", "198.51.100.7"
	mk := func(src, dst string, wire uint64) Row {
		return Row{TS: ts, Src: src, Dst: dst, Client: src, Server: dst,
			Bytes: wire, Wire: wire, Pkts: 1, Flows: 1, Exporter: "10.0.0.1", Dir: DirTransit}
	}
	rows := []Row{
		mk(user, a, 1000), // in to A from outside
		mk(a, user, 9000), // out of A to outside
		mk(a, b, 5000),    // inside the set: A -> B
		mk(b, user, 2000), // out of B to outside
	}
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	q := Query{From: ts.Add(-time.Minute), To: ts.Add(time.Hour)}
	set := []netip.Addr{netip.MustParseAddr(a), netip.MustParseAddr(b)}
	got, err := s.Billing(q, set)
	if err != nil {
		t.Fatal(err)
	}
	// in: 1000 + 5000 (B's side of the inside flow); out: 9000 + 5000 +
	// 2000; total: every byte once = 1000+9000+5000+2000
	if got.InBytes != 6000 || got.OutBytes != 16000 {
		t.Errorf("in/out = %d/%d, want 6000/16000", got.InBytes, got.OutBytes)
	}
	if got.Bytes != 17000 {
		t.Errorf("total = %d, want 17000 (the inside flow counted once, not twice)", got.Bytes)
	}
	if got.Bytes != got.InBytes+got.OutBytes-5000 {
		t.Errorf("total %d does not reconcile: in + out - inside", got.Bytes)
	}
	// the members' own figures do add up past the total: A carries the
	// inside flow out, B carries it in
	var memberSum uint64
	for _, m := range got.Members {
		memberSum += m.Bytes
	}
	if memberSum != 22000 {
		t.Errorf("member sum = %d, want 22000 (6000+16000, the inside flow on both members)", memberSum)
	}
	// one address alone: no inside traffic, total is in + out
	one, err := s.Billing(q, set[:1])
	if err != nil {
		t.Fatal(err)
	}
	if one.Bytes != 15000 || one.InBytes != 1000 || one.OutBytes != 14000 {
		t.Errorf("single VIP = %d (in %d, out %d), want 15000 (1000, 14000)", one.Bytes, one.InBytes, one.OutBytes)
	}
	// the series tells the same story as the totals
	var tot float64
	for i, v := range got.Total {
		tot += v * float64(got.Step) / 8
		_ = i
	}
	if tot < 16999 || tot > 17001 {
		t.Errorf("series total = %.0f bytes, want 17000", tot)
	}
	if _, err := s.Billing(q, nil); err == nil {
		t.Error("an empty set was accepted")
	}
}

// The days add up to the totals exactly, on both the list and the
// subnet paths, and a day is the viewer's calendar day: a bill and its
// daily breakdown never disagree.
func TestBillingDaysAddUp(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// three days back, at 23:30 UTC, and a row each following day: in
	// UTC+8 the 23:30 rows fall on the next calendar day
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-72*time.Hour + 23*time.Hour + 30*time.Minute)
	vip, user := "203.0.113.10", "198.51.100.7"
	var rows []Row
	for d := 0; d < 3; d++ {
		ts := base.Add(time.Duration(d) * 24 * time.Hour)
		rows = append(rows,
			Row{TS: ts, Src: user, Dst: vip, Client: user, Server: vip, Bytes: 1000, Wire: 1000, Pkts: 1, Flows: 1, Exporter: "10.0.0.1", Dir: DirTransit},
			Row{TS: ts.Add(time.Hour), Src: vip, Dst: user, Client: user, Server: vip, Bytes: 7000, Wire: 7000, Pkts: 1, Flows: 1, Exporter: "10.0.0.1", Dir: DirTransit})
	}
	s.stopFold()
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	// the subnet path reads the hourly summaries: seal the days and fold
	// them in, as the running store does once the minutes settle
	for d := 1; d <= 4; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.foldOnce(time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	check := func(name string, b *Billing) {
		var in, out, tot uint64
		for _, d := range b.Days {
			in, out, tot = in+d.InBytes, out+d.OutBytes, tot+d.Bytes
		}
		if in != b.InBytes || out != b.OutBytes || tot != b.Bytes || b.Bytes != 24000 {
			t.Errorf("%s: days add up to %d/%d/%d, totals %d/%d/%d (want 24000)", name, in, out, tot, b.InBytes, b.OutBytes, b.Bytes)
		}
	}
	for _, zone := range []time.Duration{0, 8 * time.Hour} {
		q := Query{From: base.Add(-2 * time.Hour), To: base.Add(70 * time.Hour), ZoneOffset: zone}
		b, err := s.Billing(q, []netip.Addr{netip.MustParseAddr(vip)})
		if err != nil {
			t.Fatal(err)
		}
		check("list", b)
		sb, err := s.BillingSubnet(q, netip.MustParsePrefix("203.0.113.0/24"))
		if err != nil {
			t.Fatal(err)
		}
		check("subnet", sb)
		// UTC: 23:30 in and 00:30 out land on consecutive days; in UTC+8
		// both are 07:30 and 08:30 of the same day
		first := base.In(time.FixedZone("", int(zone.Seconds()))).Format("2006-01-02")
		for _, d := range b.Days {
			if d.Day == first && zone == 8*time.Hour && d.Bytes != 8000 {
				t.Errorf("UTC+8 %s: %d bytes, want 8000 (in and out the same local day)", d.Day, d.Bytes)
			}
			if d.Day == first && zone == 0 && d.Bytes != 1000 {
				t.Errorf("UTC %s: %d bytes, want 1000 (only the 23:30 inbound)", d.Day, d.Bytes)
			}
		}
	}
}
