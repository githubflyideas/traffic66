package assets

import (
	"testing"
)

// counts summarises a plan for the assertions below.
func counts(p Plan) map[string]int {
	m := map[string]int{}
	for _, c := range p.Changes {
		m[c.Kind+"/"+c.Action]++
		if c.Error != "" {
			m["errors"]++
		}
	}
	m["skipped"] = len(p.Skipped)
	return m
}

// show lldp neighbors: the local port, the neighbour and its port. The
// neighbour and the cable are recorded; a previewed paste writes nothing.
func TestPasteLLDP(t *testing.T) {
	s := open(t)
	me, err := s.SaveDevice(Device{Name: "core-sw1"})
	if err != nil {
		t.Fatal(err)
	}
	text := `core-sw1# show lldp neighbors

Capability codes:
    (R) Router, (B) Bridge, (T) Telephone

Device ID           Local Intf     Hold-time  Capability      Port ID
edge-sw1.lan        Gi0/1          120        B,R             Gi1/0/24
fw-01               Gi0/2          120        R               ge-0/0/3

Total entries displayed: 2
`
	p, err := s.Paste(PasteOptions{Text: text, DeviceID: me.ID})
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != FormatNeighbors {
		t.Fatalf("read as %q", p.Format)
	}
	c := counts(p)
	if c["link/add"] != 2 || c["device/add"] != 2 || c["port/add"] != 4 || c["errors"] != 0 {
		t.Fatalf("preview: %v\n%+v", c, p.Changes)
	}
	if n, _ := s.Devices(DeviceQuery{}); len(n) != 1 {
		t.Fatalf("a previewed paste wrote %d devices", len(n))
	}

	p, err = s.Paste(PasteOptions{Text: text, DeviceID: me.ID, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if c := counts(p); c["errors"] != 0 {
		t.Fatalf("applied: %v\n%+v", c, p.Changes)
	}
	ds, _ := s.Devices(DeviceQuery{})
	if len(ds) != 3 {
		t.Fatalf("devices: %+v", ds)
	}
	ls, _ := s.Links()
	if len(ls) != 2 {
		t.Fatalf("links: %+v", ls)
	}
	// running it again changes nothing that matters: the same ports and
	// the same cables are found rather than doubled
	if _, err := s.Paste(PasteOptions{Text: text, DeviceID: me.ID, Apply: true}); err != nil {
		t.Fatal(err)
	}
	ds, _ = s.Devices(DeviceQuery{})
	ls, _ = s.Links()
	ps, _ := s.Ports(0)
	if len(ds) != 3 || len(ls) != 2 || len(ps) != 4 {
		t.Fatalf("after a second paste: %d devices, %d links, %d ports", len(ds), len(ls), len(ps))
	}
}

// show cdp neighbors spells its columns differently and puts a space
// inside the interface name, which is why the columns are read rather
// than the whitespace.
func TestPasteCDP(t *testing.T) {
	s := open(t)
	me, _ := s.SaveDevice(Device{Name: "core-sw1"})
	text := `Device ID        Local Intrfce     Holdtme    Capability  Platform  Port ID
edge-sw1         Gig 0/1            150         S I       WS-C2960  Gig 1/0/24
`
	p, err := s.Paste(PasteOptions{Text: text, DeviceID: me.ID, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if c := counts(p); c["link/add"] != 1 || c["errors"] != 0 || c["skipped"] != 0 {
		t.Fatalf("%v\n%+v\nskipped %v", c, p.Changes, p.Skipped)
	}
	ps, _ := s.Ports(me.ID)
	if len(ps) != 1 || ps[0].Name != "Gig 0/1" {
		t.Fatalf("ports: %+v", ps)
	}
	// and the same port written short is recognised as the same one
	short := `Device ID        Local Intrfce     Holdtme    Capability  Platform  Port ID
other-sw         Gi0/1              150         S I       WS-C2960  Gi1/0/1
`
	if _, err := s.Paste(PasteOptions{Text: short, DeviceID: me.ID, Apply: true}); err != nil {
		t.Fatal(err)
	}
	if ps, _ := s.Ports(me.ID); len(ps) != 1 {
		t.Fatalf("the same port was recorded twice: %+v", ps)
	}
}

// show interface status fills in the descriptions and speeds of a
// device's own ports.
func TestPasteIfStatus(t *testing.T) {
	s := open(t)
	me, _ := s.SaveDevice(Device{Name: "core-sw1"})
	// as the switch prints it: Duplex and Speed right-aligned in their
	// columns, Type left-aligned, one space between Speed and Type
	text := `Port      Name               Status       Vlan       Duplex  Speed Type
Gi0/1     uplink to isp      connected    trunk      a-full  a-1000 10/100/1000BaseTX
Gi0/2                        notconnect   10           auto   auto 10/100/1000BaseTX
Te1/1     storage            connected    20           full    10G SFP-10GBase-SR
`
	p, err := s.Paste(PasteOptions{Text: text, DeviceID: me.ID, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != FormatIfStatus {
		t.Fatalf("read as %q", p.Format)
	}
	if c := counts(p); c["port/add"] != 3 || c["errors"] != 0 {
		t.Fatalf("%v\n%+v", c, p.Changes)
	}
	ps, _ := s.Ports(me.ID)
	by := map[string]Port{}
	for _, x := range ps {
		by[x.Name] = x
	}
	if by["Gi0/1"].Speed != 1_000_000_000 {
		t.Fatalf("Gi0/1 speed %d", by["Gi0/1"].Speed)
	}
	if by["Te1/1"].Speed != 10_000_000_000 {
		t.Fatalf("Te1/1 speed %d", by["Te1/1"].Speed)
	}
	if by["Gi0/2"].Speed != 0 { // "auto" is not a speed; nothing is invented
		t.Fatalf("Gi0/2 speed %d", by["Gi0/2"].Speed)
	}
	if by["Gi0/1"].Note != "uplink to isp" {
		t.Fatalf("Gi0/1 note %q", by["Gi0/1"].Note)
	}
}

// A paste that is not one of these outputs says so, and one that needs to
// know which device it came from refuses rather than guessing.
func TestPasteRefuses(t *testing.T) {
	s := open(t)
	if _, err := s.Paste(PasteOptions{Text: "hello\nworld\n"}); err == nil {
		t.Fatal("rubbish was accepted")
	}
	text := `Device ID           Local Intf     Hold-time  Capability      Port ID
edge-sw1            Gi0/1          120        B,R             Gi1/0/24
`
	if _, err := s.Paste(PasteOptions{Text: text}); err == nil {
		t.Fatal("neighbours were imported without saying which device they were seen from")
	}
	if _, err := s.Paste(PasteOptions{Text: text, DeviceID: 999}); err == nil {
		t.Fatal("a device that does not exist was accepted")
	}
}

// Interface names are compared as the vendors write them.
func TestSamePort(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"Gi0/1", "Gig 0/1", true},
		{"Gi0/1", "GigabitEthernet0/1", true},
		{"gi0/1", "Gi0/1", true},
		{"Gi0/1", "Gi0/2", false},
		{"Gi0/1", "Te0/1", false},
		{"ge-0/0/3", "ge-0/0/3", true},
		{"Eth1/1", "Ethernet1/1", true},
	} {
		if got := samePort(c.a, c.b); got != c.want {
			t.Fatalf("samePort(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

// Speeds are read as the interface tables print them, and anything that
// is not a speed stays unknown.
func TestParseSpeed(t *testing.T) {
	for in, want := range map[string]uint64{
		"1000": 1_000_000_000, "a-1000": 1_000_000_000, "10G": 10_000_000_000,
		"a-10G": 10_000_000_000, "100": 100_000_000, "auto": 0, "": 0, "unknown": 0,
	} {
		if got := parseSpeed(in); got != want {
			t.Fatalf("parseSpeed(%q) = %d, want %d", in, got, want)
		}
	}
}
