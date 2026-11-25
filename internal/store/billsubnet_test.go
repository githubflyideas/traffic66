package store

import (
	"net/netip"
	"testing"
	"time"
)

// subnetFixture is two days of flows around two v4 ranges and one
// v6 range, one row per flow per hour, thirteen minutes into the
// hour so the hourly summaries have minutes to fold.
//
// The v4 addresses: x and y in 10.10.0.0/24, z in the /23 but not
// the /24, out beyond both. Per hour:
//
//	out -> x 1000    in to the /24
//	x -> out 9000    out of the /24
//	x -> y   5000    inside the /24
//	y -> x   3000    inside the /24
//	out -> z 7000    in to the /23 only
//	z -> x   2000    in to the /24, inside the /23
//	x -> z   4000    out of the /24, inside the /23
//
// In and Out count every arrival at and departure from the range,
// the inside flows among them; the billed figure counts each flow
// that touches the range once. So the /24 reads in 11000, out
// 21000, billed 24000 (every flow but out -> z, once); the /23
// reads in 22000, out 23000, billed 31000 (every v4 flow, once).
//
// The v6 addresses: a6, b6 and g6 in 2001:db8:1::/48 — g6's string
// ends in '::', a6's and b6's carry it in the middle — u6 outside.
// Per hour:
//
//	u6 -> a6 1100    in
//	a6 -> b6 2000    inside
//	b6 -> g6 3000    inside
//	g6 -> u6 4400    out
//
// so the /48 reads in 6100, out 9400, billed 10500 (all four
// flows, once).
func subnetFixture(base time.Time) []Row {
	const (
		out        = "198.51.100.7"
		x, y, z    = "10.10.0.10", "10.10.0.11", "10.10.1.10"
		u6         = "2001:db8:2::7"
		a6, b6, g6 = "2001:db8:1::10", "2001:db8:1::11", "2001:db8:1:ffff::"
	)
	var rows []Row
	for h := 0; h < 48; h++ {
		ts := base.Add(time.Duration(h)*time.Hour + 13*time.Minute)
		mk := func(src, dst string, wire uint64) Row {
			return Row{TS: ts, Src: src, Dst: dst, Client: src, Server: dst,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: 10, Flows: 1,
				Exporter: "10.0.0.1", Dir: DirTransit}
		}
		rows = append(rows,
			mk(out, x, 1000), mk(x, out, 9000), mk(x, y, 5000), mk(y, x, 3000),
			mk(out, z, 7000), mk(z, x, 2000), mk(x, z, 4000),
			mk(u6, a6, 1100), mk(a6, b6, 2000), mk(b6, g6, 3000), mk(g6, u6, 4400),
		)
	}
	return rows
}

func subnetStore(t *testing.T) (*Store, time.Time) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := Open(Options{Dir: t.TempDir(), Threads: 2, RawDays: 7, RollupDays: 180})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	s.stopFold()
	if err := s.AppendRows(subnetFixture(base)); err != nil {
		t.Fatal(err)
	}
	for d := 1; d <= 2; d++ {
		if err := s.Seal(base.Add(time.Duration(d) * 24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.foldOnce(base.Add(48 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	return s, base
}

// memberMap flattens a member split for comparison between answers
// whose members ride in different orders.
func memberMap(b *Billing) map[string]MemberRow {
	out := map[string]MemberRow{}
	for _, m := range b.Members {
		out[m.Addr] = m
	}
	return out
}

func TestBillingSubnetFigures(t *testing.T) {
	s, base := subnetStore(t)
	q := Query{From: base, To: base.Add(48 * time.Hour)}
	const H = 48 // hours in the fixture

	cases := []struct {
		prefix string
		set    []netip.Addr
		bytes  uint64
	}{
		{"10.10.0.0/24", []netip.Addr{netip.MustParseAddr("10.10.0.10"), netip.MustParseAddr("10.10.0.11")}, 24000 * H},
		{"10.10.0.0/23", []netip.Addr{netip.MustParseAddr("10.10.0.10"), netip.MustParseAddr("10.10.0.11"), netip.MustParseAddr("10.10.1.10")}, 31000 * H},
		{"2001:db8:1::/48", []netip.Addr{netip.MustParseAddr("2001:db8:1::10"), netip.MustParseAddr("2001:db8:1::11"), netip.MustParseAddr("2001:db8:1:ffff::")}, 10500 * H},
	}
	for _, c := range cases {
		got, err := s.BillingSubnet(q, netip.MustParsePrefix(c.prefix))
		if err != nil {
			t.Fatalf("%s: %v", c.prefix, err)
		}
		if got.Bytes != c.bytes {
			t.Errorf("%s: bytes = %d, want %d", c.prefix, got.Bytes, c.bytes)
		}
		// the same set asked as a product line over the flow records
		// must read the same figures, inside traffic counted once
		s.billReady.Store(false)
		want, err := s.Billing(q, c.set)
		s.billReady.Store(true)
		if err != nil {
			t.Fatalf("%s as a set: %v", c.prefix, err)
		}
		if got.InBytes != want.InBytes || got.OutBytes != want.OutBytes || got.Bytes != want.Bytes {
			t.Errorf("%s: subnet in/out/bytes = %d/%d/%d, set reads %d/%d/%d",
				c.prefix, got.InBytes, got.OutBytes, got.Bytes, want.InBytes, want.OutBytes, want.Bytes)
		}
		gm, wm := memberMap(got), memberMap(want)
		if len(gm) != len(wm) {
			t.Fatalf("%s: %d members, set has %d", c.prefix, len(gm), len(wm))
		}
		for a, w := range wm {
			if gm[a] != w {
				t.Errorf("%s: member %s = %+v, set reads %+v", c.prefix, a, gm[a], w)
			}
		}
		if got.MembersTotal != len(wm) {
			t.Errorf("%s: members_total = %d, want %d", c.prefix, got.MembersTotal, len(wm))
		}
	}

	// an unmasked prefix asks for the range its address stands in
	a, err := s.BillingSubnet(q, netip.MustParsePrefix("10.10.0.10/24"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.BillingSubnet(q, netip.MustParsePrefix("10.10.0.0/24"))
	if err != nil {
		t.Fatal(err)
	}
	if a.Bytes != b.Bytes || a.InBytes != b.InBytes || a.OutBytes != b.OutBytes {
		t.Errorf("unmasked /24 = %d/%d/%d, masked reads %d/%d/%d",
			a.InBytes, a.OutBytes, a.Bytes, b.InBytes, b.OutBytes, b.Bytes)
	}

	// the whole of each family: every flow is inside traffic, so the
	// billed figure is the family's whole wire, counted once — and
	// no flow of the other family leaks in
	v4, err := s.BillingSubnet(q, netip.MustParsePrefix("0.0.0.0/0"))
	if err != nil {
		t.Fatal(err)
	}
	if v4.Bytes != 31000*H || v4.InBytes != 31000*H || v4.OutBytes != 31000*H {
		t.Errorf("0.0.0.0/0: in/out/bytes = %d/%d/%d, want %d throughout", v4.InBytes, v4.OutBytes, v4.Bytes, 31000*H)
	}
	v6, err := s.BillingSubnet(q, netip.MustParsePrefix("::/0"))
	if err != nil {
		t.Fatal(err)
	}
	if v6.Bytes != 10500*H {
		t.Errorf("::/0: bytes = %d, want %d", v6.Bytes, 10500*H)
	}

	// a range with no traffic in it reads zero, with no members
	none, err := s.BillingSubnet(q, netip.MustParsePrefix("192.0.2.0/24"))
	if err != nil {
		t.Fatal(err)
	}
	if none.Bytes != 0 || len(none.Members) != 0 || none.MembersTotal != 0 {
		t.Errorf("empty range: bytes = %d, members = %d/%d, want all zero", none.Bytes, len(none.Members), none.MembersTotal)
	}
}

// The member split of a wide range is capped, largest first, and the
// answer says how many addresses there were.
func TestBillingSubnetMemberCap(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// 300 addresses of 10.99.0.0/23, each receiving its own index in
	// bytes from outside, straight into the summaries
	for i := 1; i <= 300; i++ {
		dst := netip.AddrFrom4([4]byte{10, 99, byte(i >> 8), byte(i)}).String()
		if _, err := s.DB.Exec(`INSERT INTO r_bill VALUES (?, false, '198.51.100.7', ?, ?)`,
			ts, dst, uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	b, err := s.BillingSubnet(Query{From: ts, To: ts.Add(time.Hour)}, netip.MustParsePrefix("10.99.0.0/23"))
	if err != nil {
		t.Fatal(err)
	}
	if b.MembersTotal != 300 {
		t.Errorf("members_total = %d, want 300", b.MembersTotal)
	}
	if len(b.Members) != subnetMemberCap {
		t.Fatalf("members = %d, want the cap %d", len(b.Members), subnetMemberCap)
	}
	if b.Members[0].InBytes != 300 || b.Members[len(b.Members)-1].InBytes != 300-subnetMemberCap+1 {
		t.Errorf("split runs %d..%d, want the largest 300..%d",
			b.Members[0].InBytes, b.Members[len(b.Members)-1].InBytes, 300-subnetMemberCap+1)
	}
	if b.InBytes != 300*301/2 {
		t.Errorf("in bytes = %d, want %d (the cap trims the split, never the total)", b.InBytes, 300*301/2)
	}
}
