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

// Where a device stands and which carrier an interface goes out to are
// written on their own lines, and editing one thing on a line keeps the
// rest of it.
func TestPlacesAndCircuits(t *testing.T) {
	inv := NewInventory()
	if err := inv.Save("device 10.0.0.1 Core sampling=1000\niface 10.0.0.1 3 Uplink speed=1000000000 tag=wan default\n"); err != nil {
		t.Fatal(err)
	}
	a := netip.MustParseAddr("10.0.0.1")
	if err := inv.SetDevice(a, "Core router", "Room 1", "A03"); err != nil {
		t.Fatal(err)
	}
	if p := inv.PlaceOf(a); p.Room != "Room 1" || p.Rack != "A03" || inv.Device(a) != "Core router" {
		t.Fatalf("place %+v name %q", p, inv.Device(a))
	}
	if inv.Sampling()[a] != 1000 {
		t.Fatal("the sampling rate on the device line was lost")
	}
	if err := inv.SetCircuit(a, 3, "Telecom uplink", "China Telecom", "SH-12345"); err != nil {
		t.Fatal(err)
	}
	it := inv.Iface(a, 3)
	if it.ISP != "China Telecom" || it.Circuit != "SH-12345" || it.Name != "Telecom uplink" {
		t.Fatalf("circuit %+v", it)
	}
	if it.Speed != 1000000000 || it.Tag != "wan" || !it.Default {
		t.Fatalf("the rest of the line was lost: %+v", it)
	}
	// renaming from the interface page keeps the carrier
	if err := inv.SetIface(a, 3, "WAN", "wan", true); err != nil {
		t.Fatal(err)
	}
	if it := inv.Iface(a, 3); it.ISP != "China Telecom" || it.Circuit != "SH-12345" {
		t.Fatalf("renaming dropped the carrier: %+v", it)
	}
	if c := inv.Circuits(); len(c) != 1 {
		t.Fatalf("circuits %v", c)
	}
	want := "device 10.0.0.1 Core router room=Room_1 rack=A03 sampling=1000\n" +
		"iface 10.0.0.1 3 WAN speed=1000000000 tag=wan isp=China_Telecom circuit=SH-12345 default\n"
	if got := inv.Text(); got != want {
		t.Fatalf("text\n%s\nwant\n%s", got, want)
	}
}
