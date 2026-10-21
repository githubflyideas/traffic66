package enrich

import (
	"net/netip"
	"strings"
	"testing"
)

func TestSetIface(t *testing.T) {
	inv := NewInventory()
	if err := inv.Save("# names\niface 10.0.0.1 3 ISP uplink speed=1000000000 default\ndevice 10.0.0.1 Core\n"); err != nil {
		t.Fatal(err)
	}
	a := netip.MustParseAddr("10.0.0.1")
	// name and tag a new interface and make it the default
	if err := inv.SetIface(a, 10, "Telecom 100G", "uplink", true); err != nil {
		t.Fatal(err)
	}
	if it := inv.Iface(a, 10); it.Name != "Telecom 100G" || it.Tag != "uplink" || !it.Default {
		t.Fatalf("iface 10 %+v", it)
	}
	if it := inv.Iface(a, 3); it.Default || it.Speed != 1000000000 || it.Name != "ISP uplink" {
		t.Fatalf("iface 3 %+v", it)
	}
	if inv.DefaultIface() != "10.0.0.1/10" {
		t.Fatalf("default %q", inv.DefaultIface())
	}
	// rename keeps the speed and the place of the line
	if err := inv.SetIface(a, 3, "ISP #2", "", false); err != nil {
		t.Fatal(err)
	}
	txt := inv.Text()
	if !strings.Contains(txt, "iface 10.0.0.1 3 ISP #2 speed=1000000000\n") || !strings.HasPrefix(txt, "# names\niface 10.0.0.1 3") {
		t.Fatalf("text %q", txt)
	}
	if it := inv.Iface(a, 3); it.Name != "ISP #2" {
		t.Fatalf("iface 3 %+v", it)
	}
}
