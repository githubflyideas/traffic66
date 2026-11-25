package enrich

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

var A = netip.MustParseAddr

// The inventory lines other than product lines: networks match by the
// longest prefix, a device line carries its place and sampling, an iface
// line written by hand wins over the name learned from the device but
// borrows its speed, and IPv4-mapped addresses find the same entries.
func TestInventoryLines(t *testing.T) {
	inv := NewInventory()
	err := inv.Parse(`# comment
net 10.0.0.0/8 Internal
net 10.1.0.0/16 Branch country=JP
device 192.0.2.1 Core router room=Room_1 rack=A03
device 192.0.2.2 Edge unsampled
device 192.0.2.3 Old switch sampling=512
iface 192.0.2.1 3 ISP uplink speed=1000000000 isp=China_Telecom circuit=SH-12345
iface 192.0.2.1 4 LAN
host 192.168.1.10 File server
snmp 192.0.2.1 public 198.51.100.1:1161
`)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := inv.Network(A("10.1.2.3")); !ok || n.Name != "Branch" {
		t.Errorf("10.1.2.3 in %q, want the longer prefix Branch", n.Name)
	}
	if n, ok := inv.Network(A("::ffff:10.9.0.1")); !ok || n.Name != "Internal" {
		t.Errorf("mapped 10.9.0.1 in %q, want Internal", n.Name)
	}
	if _, ok := inv.Network(A("203.0.113.1")); ok {
		t.Error("203.0.113.1 is in no network")
	}
	if !inv.Internal(A("172.16.0.1")) || !inv.Internal(A("10.1.0.1")) || inv.Internal(A("203.0.113.1")) {
		t.Error("Internal: private and inventory ranges in, public out")
	}
	if got := inv.SegmentCountries(); got["Branch"] != "JP" || len(got) != 1 {
		t.Errorf("segment countries %v", got)
	}
	if s := inv.Segment(A("203.0.113.77")); s != "203.0.113.0/24" {
		t.Errorf("unnamed v4 segment %q", s)
	}
	if s := inv.Segment(A("2001:db8:1:2::1")); s != "2001:db8:1::/48" {
		t.Errorf("unnamed v6 segment %q", s)
	}
	if inv.Device(A("192.0.2.1")) != "Core router" {
		t.Errorf("device name %q", inv.Device(A("192.0.2.1")))
	}
	if p := inv.PlaceOf(A("192.0.2.1")); p.Room != "Room 1" || p.Rack != "A03" {
		t.Errorf("place %+v, want Room 1 / A03 (underscore is a space)", p)
	}
	if !inv.Unsampled()[A("192.0.2.2")] || inv.Unsampled()[A("192.0.2.1")] {
		t.Errorf("unsampled %v", inv.Unsampled())
	}
	if inv.Sampling()[A("192.0.2.3")] != 512 {
		t.Errorf("sampling %v", inv.Sampling())
	}
	if len(inv.Devices()) != 3 {
		t.Errorf("devices %v", inv.Devices())
	}
	up := inv.Iface(A("192.0.2.1"), 3)
	if up.Name != "ISP uplink" || up.Speed != 1e9 || up.ISP == "" || up.Circuit != "SH-12345" {
		t.Errorf("uplink %+v", up)
	}
	if c := inv.Circuits(); len(c) != 1 {
		t.Errorf("circuits %v, want only the uplink", c)
	}
	inv.SetAutoIface(A("192.0.2.1"), 4, "Gi0/4", 1e10)
	inv.SetAutoIface(A("192.0.2.1"), 9, "Gi0/9", 1e9)
	if it := inv.Iface(A("192.0.2.1"), 4); it.Name != "LAN" || it.Speed != 1e10 {
		t.Errorf("hand-named iface %+v, want its own name and the learned speed", it)
	}
	if it := inv.Iface(A("192.0.2.1"), 9); it.Name != "Gi0/9" {
		t.Errorf("learned iface %+v", it)
	}
	if inv.Host(A("::ffff:192.168.1.10")) != "File server" || inv.Hosts()["192.168.1.10"] != "File server" {
		t.Error("host lookup")
	}
	if ts := inv.SNMPTargets(); len(ts) != 1 {
		t.Errorf("snmp targets %+v", ts)
	}
}

// A line that cannot be read is refused with the whole text, so a typo
// never leaves half an inventory in place.
func TestInventoryRefusesBadLines(t *testing.T) {
	inv := NewInventory()
	if err := inv.Parse("host 192.168.1.10 Keep me\n"); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"bad prefix":   "net 10.0.0.0/33 X\n",
		"bad address":  "device 999.0.0.1 X\n",
		"bad ifindex":  "iface 192.0.2.1 x Name\n",
		"unknown kind": "router 192.0.2.1 X\n",
		"bad speed":    "iface 192.0.2.1 3 Up speed=fast\n",
		"bad country":  "net 10.0.0.0/8 X country=JPN\n",
		"bad sampling": "device 192.0.2.1 X sampling=0\n",
	} {
		if err := inv.Parse(text); err == nil {
			t.Errorf("%s: accepted %q", name, text)
		}
	}
	if inv.Host(A("192.168.1.10")) != "Keep me" {
		t.Error("a refused text replaced the inventory")
	}
}

func TestLoadInventory(t *testing.T) {
	dir := t.TempDir()
	inv, err := LoadInventory(filepath.Join(dir, "missing.txt"))
	if err != nil || inv == nil || len(inv.Devices()) != 0 {
		t.Fatalf("missing file: %v, %v", inv, err)
	}
	p := filepath.Join(dir, "inventory.txt")
	os.WriteFile(p, []byte("device 192.0.2.1 R1\n"), 0o600)
	if inv, err := LoadInventory(p); err != nil || inv.Device(A("192.0.2.1")) != "R1" {
		t.Fatalf("load: %v", err)
	}
	os.WriteFile(p, []byte("device nope R1\n"), 0o600)
	if _, err := LoadInventory(p); err == nil {
		t.Error("a broken file loaded")
	}
}

func TestParseSpeed(t *testing.T) {
	for in, want := range map[string]uint64{"1000000000": 1e9, "1G": 1e9, "10g": 1e10, "100M": 1e8, "400k": 4e5, "1T": 1e12} {
		if got, err := parseSpeed(in); err != nil || got != want {
			t.Errorf("parseSpeed(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "0G", "fast", "G", "1.5G", "-1", "99999999999999999999T", "20000000T"} {
		if _, err := parseSpeed(in); err == nil {
			t.Errorf("parseSpeed(%q) accepted", in)
		}
	}
}
