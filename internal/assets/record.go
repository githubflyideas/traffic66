package assets

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

// Site is a building or data centre.
type Site struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
	Note    string `json:"note,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
}

// Rack is a cabinet in a site. Units is its height in rack units; devices
// are placed by the unit they start at.
type Rack struct {
	ID     int64  `json:"id"`
	SiteID int64  `json:"site_id"`
	Name   string `json:"name"`
	Units  int    `json:"units"`
	Note   string `json:"note,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
}

// Device is one piece of equipment. RackID 0 means it is not racked, and
// PosU 0 means it is in a rack but its position is not recorded — both are
// ordinary states in a ledger people fill in over time.
type Device struct {
	ID      int64    `json:"id"`
	RackID  int64    `json:"rack_id,omitempty"`
	Name    string   `json:"name"`
	Role    string   `json:"role,omitempty"`
	Vendor  string   `json:"vendor,omitempty"`
	Model   string   `json:"model,omitempty"`
	Serial  string   `json:"serial,omitempty"`
	MgmtIP  string   `json:"mgmt_ip,omitempty"`
	PosU    int      `json:"pos_u,omitempty"`
	HeightU int      `json:"height_u,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Note    string   `json:"note,omitempty"`
	Hidden  bool     `json:"hidden,omitempty"`
}

// Port is a physical port of a device. IfIndex, when known, is what ties
// the ledger to the flow data: it is the same number the exporter puts in
// its records.
type Port struct {
	ID       int64  `json:"id"`
	DeviceID int64  `json:"device_id"`
	Name     string `json:"name"`
	IfIndex  uint32 `json:"ifindex,omitempty"`
	Speed    uint64 `json:"speed,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Link is a cable between two ports.
type Link struct {
	ID    int64  `json:"id"`
	APort int64  `json:"a_port"`
	BPort int64  `json:"b_port"`
	Note  string `json:"note,omitempty"`
}

// Circuit is a line bought from a carrier, landing on a port.
type Circuit struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Carrier   string   `json:"carrier,omitempty"`
	Kind      string   `json:"kind,omitempty"`
	Bandwidth string   `json:"bandwidth,omitempty"`
	SiteID    int64    `json:"site_id,omitempty"`
	PortID    int64    `json:"port_id,omitempty"`
	Contract  string   `json:"contract,omitempty"`
	Starts    string   `json:"starts,omitempty"` // YYYY-MM-DD
	Ends      string   `json:"ends,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Note      string   `json:"note,omitempty"`
	Hidden    bool     `json:"hidden,omitempty"`
}

// Subnet is a network a device carries, written down beside it. It is a
// note, not a source of truth for anything: nothing is looked up in it.
type Subnet struct {
	ID       int64    `json:"id"`
	DeviceID int64    `json:"device_id,omitempty"`
	Prefix   string   `json:"prefix"`
	Name     string   `json:"name,omitempty"`
	VLAN     int      `json:"vlan,omitempty"`
	Gateway  string   `json:"gateway,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Note     string   `json:"note,omitempty"`
	Hidden   bool     `json:"hidden,omitempty"`
}

// CanonTags trims, lowercases, de-duplicates and sorts tags, so that the
// same tag typed three ways is one tag.
func CanonTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range in {
		for _, p := range strings.Split(t, ",") {
			p = strings.ToLower(strings.TrimSpace(p))
			if p == "" || seen[p] || len(out) >= 24 {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func tagString(in []string) string { return strings.Join(CanonTags(in), ",") }

func tagList(s string) []string {
	if s == "" {
		return nil
	}
	return CanonTags(strings.Split(s, ","))
}

// trim cuts a field to a sane length. The ledger is typed in by hand; a
// field longer than this is a paste accident, not a name.
func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = strings.TrimSpace(s[:max])
	}
	return s
}

// CanonMAC reads a MAC address in any of the forms switches print it
// (aa:bb:cc:dd:ee:ff, aa-bb-cc-dd-ee-ff, aabb.ccdd.eeff, aabbccddeeff) and
// returns the colon form. It returns "" when the text is not a MAC, so a
// mistyped one is dropped rather than stored as a fact.
func CanonMAC(s string) string {
	var hex []rune
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
			hex = append(hex, r)
		case r == ':' || r == '-' || r == '.' || r == ' ':
		default:
			return ""
		}
	}
	if len(hex) != 12 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteRune(hex[i])
		b.WriteRune(hex[i+1])
	}
	return b.String()
}

// canonIP returns the address in its canonical form, or an error. Storing
// it canonically is what makes "already assigned" a reliable answer: the
// same address written two ways must be one row.
func canonIP(s string) (string, error) {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%q is not an IP address", s)
	}
	return a.Unmap().String(), nil
}

// canonPrefix returns the network in its canonical form, with the host
// bits cleared, so that 10.0.0.7/24 and 10.0.0.0/24 are the same subnet.
func canonPrefix(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(s))
	if err != nil {
		return p, fmt.Errorf("%q is not a network (expected something like 10.0.0.0/24)", s)
	}
	return p.Masked(), nil
}

func now() time.Time { return time.Now().UTC() }
