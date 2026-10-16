package enrich

import (
	"net/netip"
	"strings"
	"testing"
)

// A device reports every interface it has; the two or three somebody pays
// a carrier for are marked by hand, as a bare word on the iface line the
// same way default is.
func TestBilledInterfaces(t *testing.T) {
	inv := NewInventory()
	if err := inv.Parse(`
iface 10.0.0.1 3 GigabitEthernet0/1 speed=1000000000 tag=uplink billed
iface 10.0.0.1 4 GigabitEthernet0/2 default
iface 10.0.0.2 7 ISP-B billed
`); err != nil {
		t.Fatal(err)
	}
	b := inv.Billed()
	if len(b) != 2 || !b["10.0.0.1/3"] || !b["10.0.0.2/7"] {
		t.Fatalf("billed = %v, want the two marked interfaces", b)
	}
	one := inv.Iface(netip.MustParseAddr("10.0.0.1"), 3)
	if !one.Billed || one.Tag != "uplink" || one.Speed != 1000000000 || one.Name != "GigabitEthernet0/1" {
		t.Errorf("the mark ate another field: %+v", one)
	}
	if inv.Iface(netip.MustParseAddr("10.0.0.1"), 4).Billed {
		t.Error("an unmarked interface came back billed")
	}
}

// Marking and unmarking go through the same line, and must not lose the
// speed, the name, the tag or the default that line already carried.
func TestSetIfaceKeepsTheRestOfTheLine(t *testing.T) {
	inv := NewInventory()
	if err := inv.Parse("iface 10.0.0.1 3 Gi0/1 speed=1000000000 tag=uplink default\n"); err != nil {
		t.Fatal(err)
	}
	a := netip.MustParseAddr("10.0.0.1")
	if err := inv.SetIface(a, 3, "Gi0/1", "uplink", true, true); err != nil {
		t.Fatal(err)
	}
	got := inv.Iface(a, 3)
	if !got.Billed || !got.Default || got.Tag != "uplink" || got.Speed != 1000000000 {
		t.Fatalf("after marking: %+v", got)
	}
	if n := strings.Count(inv.Text(), "iface 10.0.0.1 3"); n != 1 {
		t.Fatalf("%d lines for one interface, want 1:\n%s", n, inv.Text())
	}
	if err := inv.SetIface(a, 3, "Gi0/1", "uplink", true, false); err != nil {
		t.Fatal(err)
	}
	if inv.Iface(a, 3).Billed {
		t.Error("unmarking did not take")
	}
	if strings.Contains(inv.Text(), "billed") {
		t.Errorf("the word stayed in the file:\n%s", inv.Text())
	}
	if !inv.Iface(a, 3).Default || inv.Iface(a, 3).Speed != 1000000000 {
		t.Errorf("unmarking lost something else: %+v", inv.Iface(a, 3))
	}
}
