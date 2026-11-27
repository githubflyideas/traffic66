package sim

import (
	"math/rand/v2"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

var testNow = time.Date(2026, 11, 27, 12, 0, 0, 0, time.Local)

func TestASNTableCoversExternals(t *testing.T) {
	tab := ASNTable()
	lines := strings.Split(strings.TrimRight(tab, "\n"), "\n")
	if len(lines) != len(externals) {
		t.Fatalf("%d lines for %d externals", len(lines), len(externals))
	}
	for i, ln := range lines {
		f := strings.Split(ln, "\t")
		if len(f) != 5 {
			t.Fatalf("line %d: %d fields: %q", i, len(f), ln)
		}
		lo, err1 := netip.ParseAddr(f[0])
		hi, err2 := netip.ParseAddr(f[1])
		if err1 != nil || err2 != nil || lo.Compare(hi) > 0 {
			t.Errorf("line %d: bad range %q-%q", i, f[0], f[1])
		}
		if f[2] != "64601" && i == 0 {
			t.Errorf("first line ASN = %q, want the backup DC's 64601", f[2])
		}
	}
	// the visitor range is the one wide range
	if !strings.Contains(tab, "198.18.7.0\t198.18.7.255\t64611") {
		t.Errorf("visitor range missing or wrong:\n%s", tab)
	}
}

func TestThreatListNamesTheDemoHosts(t *testing.T) {
	c2, scanners := ThreatList()
	if !strings.Contains(c2, "203.0.113.200") {
		t.Errorf("c2 list: %q", c2)
	}
	if !strings.Contains(scanners, "192.0.2.66") {
		t.Errorf("scanners list: %q", scanners)
	}
}

func TestInventoryNamesDevicesAndAgents(t *testing.T) {
	inv := Inventory()
	for _, want := range []string{
		"device 10.0.0.1 Core router", "device 10.0.0.4 Branch router",
		"host 10.20.0.15 File server", "snmp 10.0.0.1 demo 127.0.0.1:16161",
		"net 10.30.0.0/16 Guest Wi-Fi",
	} {
		if !strings.Contains(inv, want) {
			t.Errorf("inventory missing %q", want)
		}
	}
}

func TestIfAlias(t *testing.T) {
	for _, tc := range []struct {
		dev  netip.Addr
		idx  uint32
		name string
	}{
		{CoreRouter, ifCoreISPA, "Gi0/0/3"},
		{Firewall, ifFwDMZ, "port2"},
		{Branch, ifBranchLAN, "Fa0/2"},
		{netip.MustParseAddr("10.9.9.9"), 7, "if7"},
	} {
		name, _ := IfAlias(tc.dev, tc.idx)
		if name != tc.name {
			t.Errorf("IfAlias(%v, %d) = %q, want %q", tc.dev, tc.idx, name, tc.name)
		}
	}
	if _, alias := IfAlias(CoreRouter, ifCoreISPB); alias != "ISP-B uplink" {
		t.Errorf("core ISP-B alias = %q", alias)
	}
	if _, alias := IfAlias(Switch, 1); alias != "" {
		t.Errorf("switch alias = %q, want none (the switch is not in the table)", alias)
	}
	if len(SNMPAgents) != 3 {
		t.Errorf("SNMPAgents has %d entries, want 3", len(SNMPAgents))
	}
	if _, ok := SNMPAgents[Switch]; ok {
		t.Error("the sFlow switch has no SNMP agent in the demo")
	}
}

func TestModelCurves(t *testing.T) {
	if bell(10.8, 10.8, 2.6) != 1 {
		t.Error("bell at its center is not 1")
	}
	if workday(3) <= 0 || workday(3) >= workday(11) {
		t.Errorf("workday(3) = %v, want a quiet night below the morning peak", workday(3))
	}
	m := newModel(testNow)
	if got := m.incidentMbps(testNow.Add(-6 * time.Hour)); got != 0 {
		t.Errorf("incident before its window = %v", got)
	}
	mid := m.incidentStart.Add(m.incidentEnd.Sub(m.incidentStart) / 2)
	if got := m.incidentMbps(mid); got <= 0 {
		t.Errorf("incident at its middle = %v, want a positive rate", got)
	}
	if got := m.incidentMbps(testNow); got != 0 {
		t.Errorf("incident after its window = %v", got)
	}
}

func TestPoolStaysInPrefix(t *testing.T) {
	p := pool("10.10.0.0/16", 900)
	r := rand.New(rand.NewPCG(7, 7))
	seen := map[netip.Addr]bool{}
	for i := 0; i < 200; i++ {
		a := p(r)
		if !netip.MustParsePrefix("10.10.0.0/16").Contains(a) {
			t.Fatalf("pool address %v outside its prefix", a)
		}
		seen[a] = true
	}
	if len(seen) < 100 {
		t.Errorf("pool produced only %d distinct addresses in 200 draws", len(seen))
	}
}

func TestRecordsAreWellFormedAndDeterministic(t *testing.T) {
	exporters := map[netip.Addr]bool{CoreRouter: true, Switch: true, Firewall: true, Branch: true}
	run := func() (int, uint64) {
		s := New(testNow, 42)
		recs, ctrs := s.Records(testNow, 5*time.Minute, true)
		var bytes uint64
		for _, r := range recs {
			if !exporters[r.Exporter] {
				t.Errorf("record from unknown exporter %v", r.Exporter)
			}
			if r.Bytes == 0 || r.Packets == 0 {
				t.Errorf("empty record: %+v", r)
			}
			if r.End.Before(r.Start) {
				t.Errorf("record ends before it starts: %v .. %v", r.Start, r.End)
			}
			if r.Mult <= 0 {
				t.Errorf("record with Mult %v", r.Mult)
			}
			bytes += r.Bytes
		}
		if len(ctrs) == 0 {
			t.Error("no interface counters alongside the records")
		}
		return len(recs), bytes
	}
	n1, b1 := run()
	if n1 == 0 || b1 == 0 {
		t.Fatalf("no traffic generated: %d records, %d bytes", n1, b1)
	}
	n2, b2 := run()
	if n1 != n2 || b1 != b2 {
		t.Errorf("same seed, different output: (%d, %d) vs (%d, %d)", n1, b1, n2, b2)
	}
}

func TestDeviceCounters(t *testing.T) {
	s := New(testNow, 1)
	if _, ctrs := s.Records(testNow, 5*time.Minute, true); len(ctrs) == 0 {
		t.Fatal("Records brought no counters")
	}
	// the per-device view exists once traffic has flowed through link()
	ctrs := s.DeviceCounters(testNow, CoreRouter)
	if len(ctrs) == 0 {
		t.Fatal("no counters for the core router")
	}
	for _, c := range ctrs {
		if c.Exporter != CoreRouter {
			t.Errorf("counter for %v from the core call", c.Exporter)
		}
		if c.Speed == 0 {
			t.Errorf("counter for if %d has no speed", c.IfIndex)
		}
	}
	if got := ifSpeed(CoreRouter, ifCoreISPB); got != 500e6 {
		t.Errorf("ISP-B speed = %d, want the inventory's 500 Mbit/s", got)
	}
	if got := ifSpeed(Switch, 99); got != 1e9 {
		t.Errorf("default speed = %d", got)
	}
}

func TestBackfillEmits(t *testing.T) {
	s := New(testNow, 3)
	batches := 0
	var recs int
	s.Backfill(testNow.Add(-2*time.Hour), testNow, true, func(rs []flow.Record, cs []flow.IfCounters) {
		batches++
		recs += len(rs)
	})
	if batches == 0 || recs == 0 {
		t.Errorf("backfill over two hours emitted %d batches, %d records", batches, recs)
	}
}
