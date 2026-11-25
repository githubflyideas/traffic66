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
