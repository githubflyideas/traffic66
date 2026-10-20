package assets

import (
	"errors"
	"testing"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// A rack holds devices at the unit they start at, and refuses a device
// that would stand where another already does or hang out of the rack.
func TestRackPlacement(t *testing.T) {
	s := open(t)
	site, err := s.SaveSite(Site{Name: "Site A"})
	if err != nil {
		t.Fatal(err)
	}
	rack, err := s.SaveRack(Rack{SiteID: site.ID, Name: "R1", Units: 12})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveDevice(Device{RackID: rack.ID, Name: "sw1", PosU: 5, HeightU: 2}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		why    string
		d      Device
		reject bool
	}{
		{"on top of sw1", Device{RackID: rack.ID, Name: "sw2", PosU: 6, HeightU: 1}, true},
		{"overlapping sw1 from below", Device{RackID: rack.ID, Name: "sw3", PosU: 4, HeightU: 3}, true},
		{"out of the rack", Device{RackID: rack.ID, Name: "sw4", PosU: 12, HeightU: 2}, true},
		{"below sw1", Device{RackID: rack.ID, Name: "sw5", PosU: 3, HeightU: 2}, false},
		{"racked but unplaced", Device{RackID: rack.ID, Name: "sw6"}, false},
		{"not racked", Device{Name: "spare"}, false},
	} {
		_, err := s.SaveDevice(c.d)
		if c.reject && err == nil {
			t.Fatalf("%s: accepted", c.why)
		}
		if !c.reject && err != nil {
			t.Fatalf("%s: %v", c.why, err)
		}
	}
	// the rack cannot shrink below what stands in it
	rack.Units = 4
	if _, err := s.SaveRack(rack); err == nil {
		t.Fatal("the rack shrank under a device")
	}
}

// A rack with no height given gets the common one, and a rack taller
// than anything real is refused.
func TestRackDefaultHeight(t *testing.T) {
	s := open(t)
	site, _ := s.SaveSite(Site{Name: "Site A"})
	r, err := s.SaveRack(Rack{SiteID: site.ID, Name: "R1"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Units != DefaultUnits {
		t.Fatalf("a rack with no height given is %dU, want %dU", r.Units, DefaultUnits)
	}
	// and a height that was asked for is kept
	r2, err := s.SaveRack(Rack{SiteID: site.ID, Name: "R2", Units: 12})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Units != 12 {
		t.Fatalf("a 12U rack is %dU", r2.Units)
	}
	if _, err := s.SaveRack(Rack{SiteID: site.ID, Name: "R3", Units: MaxUnits + 1}); err == nil {
		t.Fatal("a rack taller than MaxUnits was accepted")
	}
}

// Deleting a rack keeps the equipment that was in it; deleting a site is
// refused while anything points at it.
func TestDeleteKeepsEquipment(t *testing.T) {
	s := open(t)
	site, _ := s.SaveSite(Site{Name: "Site A"})
	rack, _ := s.SaveRack(Rack{SiteID: site.ID, Name: "R1", Units: 42})
	dev, err := s.SaveDevice(Device{RackID: rack.ID, Name: "sw1", PosU: 1})
	if err != nil {
		t.Fatal(err)
	}
	var inUse InUse
	if err := s.DeleteSite(site.ID); !errors.As(err, &inUse) {
		t.Fatalf("deleting a site with a rack: %v", err)
	}
	if err := s.DeleteRack(rack.ID); err != nil {
		t.Fatal(err)
	}
	ds, err := s.Devices(DeviceQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].ID != dev.ID {
		t.Fatalf("the device went with the rack: %+v", ds)
	}
	if ds[0].RackID != 0 || ds[0].PosU != 0 {
		t.Fatalf("the device still claims a place: %+v", ds[0])
	}
	if err := s.DeleteSite(site.ID); err != nil {
		t.Fatal(err)
	}
}

// Deleting a device takes its ports and their cables, and lets go of the
// addresses without freeing them.
func TestDeleteDevice(t *testing.T) {
	s := open(t)
	d1, _ := s.SaveDevice(Device{Name: "sw1"})
	d2, _ := s.SaveDevice(Device{Name: "sw2"})
	p1, err := s.SavePort(Port{DeviceID: d1.ID, Name: "Gi0/1", IfIndex: 1})
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := s.SavePort(Port{DeviceID: d2.ID, Name: "Gi0/1", IfIndex: 1})
	if _, err := s.SaveLink(Link{APort: p1.ID, BPort: p2.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveSubnet(Subnet{Prefix: "10.0.0.0/24", DeviceID: d1.ID, Name: "mgmt"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteDevice(d1.ID); err != nil {
		t.Fatal(err)
	}
	if ls, _ := s.Links(); len(ls) != 0 {
		t.Fatalf("the cable outlived the device: %+v", ls)
	}
	if ps, _ := s.Ports(0); len(ps) != 1 || ps[0].ID != p2.ID {
		t.Fatalf("ports left: %+v", ps)
	}
	// the networks written against it are a note about the device, and
	// they go when it does
	if ns, _ := s.Subnets(); len(ns) != 0 {
		t.Fatalf("the network outlived the device: %+v", ns)
	}
}

// A port carries one cable: a second one replaces the first.
func TestOneCablePerPort(t *testing.T) {
	s := open(t)
	d, _ := s.SaveDevice(Device{Name: "sw1"})
	var ps []Port
	for _, n := range []string{"a", "b", "c"} {
		p, err := s.SavePort(Port{DeviceID: d.ID, Name: n})
		if err != nil {
			t.Fatal(err)
		}
		ps = append(ps, p)
	}
	if _, err := s.SaveLink(Link{APort: ps[0].ID, BPort: ps[1].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveLink(Link{APort: ps[0].ID, BPort: ps[2].ID}); err != nil {
		t.Fatal(err)
	}
	ls, _ := s.Links()
	if len(ls) != 1 || ls[0].BPort != ps[2].ID {
		t.Fatalf("links: %+v", ls)
	}
	if _, err := s.SaveLink(Link{APort: ps[0].ID, BPort: ps[0].ID}); err == nil {
		t.Fatal("a port was cabled to itself")
	}
	// a device may not have two ports of the same name
	if _, err := s.SavePort(Port{DeviceID: d.ID, Name: "A"}); err == nil {
		t.Fatal("a duplicate port name was accepted")
	}
}

// A network is stored with its host bits cleared, its gateway must be
// inside it, and the same network cannot be recorded twice.
func TestSubnetCanonical(t *testing.T) {
	s := open(t)
	dev, err := s.SaveDevice(Device{Name: "sw1"})
	if err != nil {
		t.Fatal(err)
	}
	sn, err := s.SaveSubnet(Subnet{Prefix: "192.168.5.37/24", Gateway: "192.168.5.1", DeviceID: dev.ID})
	if err != nil {
		t.Fatal(err)
	}
	if sn.Prefix != "192.168.5.0/24" {
		t.Fatalf("prefix stored as %q", sn.Prefix)
	}
	if _, err := s.SaveSubnet(Subnet{Prefix: "192.168.5.0/24"}); err == nil {
		t.Fatal("the same network was recorded twice")
	}
	if _, err := s.SaveSubnet(Subnet{Prefix: "10.0.0.0/8", Gateway: "192.168.5.1"}); err == nil {
		t.Fatal("a gateway outside the network was accepted")
	}
	if _, err := s.SaveSubnet(Subnet{Prefix: "not a network"}); err == nil {
		t.Fatal("rubbish was accepted as a network")
	}
	if _, err := s.SaveSubnet(Subnet{Prefix: "172.16.0.0/16", DeviceID: 999}); err == nil {
		t.Fatal("a network was written against a device that does not exist")
	}
	if sn.DeviceID != dev.ID {
		t.Fatalf("the network is not on the device: %+v", sn)
	}
}

// Tags are one tag however they are typed, and filter.
func TestTags(t *testing.T) {
	s := open(t)
	d, err := s.SaveDevice(Device{Name: "sw1", Tags: []string{" Core ", "core", "PROD,edge"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Tags) != 3 || d.Tags[0] != "core" || d.Tags[1] != "edge" || d.Tags[2] != "prod" {
		t.Fatalf("tags: %v", d.Tags)
	}
	if _, err := s.SaveDevice(Device{Name: "sw2", Tags: []string{"edge"}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Devices(DeviceQuery{Tag: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "sw1" {
		t.Fatalf("filtered by tag: %+v", got)
	}
}

// Editing something that is gone says so instead of succeeding quietly.
func TestMissing(t *testing.T) {
	s := open(t)
	if _, err := s.SaveSite(Site{ID: 999, Name: "ghost"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("editing a missing site: %v", err)
	}
	if err := s.DeleteDevice(999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleting a missing device: %v", err)
	}
	if _, err := s.SaveDevice(Device{Name: "x", RackID: 999}); err == nil {
		t.Fatal("a device went into a rack that does not exist")
	}
}

// MAC addresses are read in every form a switch prints them, and anything
// else is refused rather than stored as a fact.
func TestCanonMAC(t *testing.T) {
	for _, in := range []string{"aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF", "aabb.ccdd.eeff", "AABBCCDDEEFF"} {
		if got := CanonMAC(in); got != "aa:bb:cc:dd:ee:ff" {
			t.Fatalf("%q read as %q", in, got)
		}
	}
	for _, in := range []string{"", "aa:bb:cc:dd:ee", "zz:bb:cc:dd:ee:ff", "10.0.0.1", "aabbccddeeff00"} {
		if got := CanonMAC(in); got != "" {
			t.Fatalf("%q read as %q", in, got)
		}
	}
}
