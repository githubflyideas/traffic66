package geo

import (
	"net/netip"
	"testing"
)

// The test databases were made with the mmdb-writer Python package and
// checked against MaxMind's maxminddb reader.
func TestLookup(t *testing.T) {
	c, err := Open("testdata/country.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	a, err := Open("testdata/asn.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	v4, err := Open("testdata/country4.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind() != "country" || a.Kind() != "asn" || c.Type != "GeoLite2-Country" {
		t.Fatalf("kinds %q %q type %q", c.Kind(), a.Kind(), c.Type)
	}
	for _, tc := range []struct {
		r    *Reader
		ip   string
		ok   bool
		want Info
	}{
		{c, "1.0.0.5", true, Info{Country: "AU"}},
		{c, "::ffff:1.0.0.5", true, Info{Country: "AU"}},
		{c, "8.8.8.8", true, Info{Country: "US"}},
		{c, "203.0.113.7", true, Info{Country: "RU"}}, // registered_country only
		{c, "2001:db8::1", true, Info{Country: "JP"}},
		{c, "9.9.9.9", false, Info{}},
		{a, "8.8.8.8", true, Info{ASN: 15169, Org: "Google LLC"}},
		{a, "2001:db8:1::5", true, Info{ASN: 64500, Org: "Example Net"}},
		{a, "2001:db8:2::5", false, Info{}},
		{v4, "192.0.2.9", true, Info{Country: "DE"}},
		{v4, "2001:db8::1", false, Info{}},
	} {
		got, ok := tc.r.Lookup(netip.MustParseAddr(tc.ip))
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s in %s: %+v %v, want %+v %v", tc.ip, tc.r.Type, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRecordSizes(t *testing.T) {
	// one node holding left=0x0ABCDEF / right=0x0123456 in each layout
	for _, tc := range []struct {
		size uint32
		node []byte
	}{
		{24, []byte{0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56}},
		{28, []byte{0xAB, 0xCD, 0xEF, 0x00, 0x12, 0x34, 0x56}},
		{32, []byte{0x00, 0xAB, 0xCD, 0xEF, 0x00, 0x12, 0x34, 0x56}},
	} {
		r := &Reader{buf: tc.node, recordSize: tc.size}
		if l, rr := r.record(0, 0), r.record(0, 1); l != 0xABCDEF || rr != 0x123456 {
			t.Errorf("size %d: %x %x", tc.size, l, rr)
		}
	}
	r := &Reader{buf: []byte{0xAB, 0xCD, 0xEF, 0x5A, 0x12, 0x34, 0x56}, recordSize: 28}
	if l, rr := r.record(0, 0), r.record(0, 1); l != 0x5ABCDEF || rr != 0xA123456 {
		t.Errorf("28-bit high nibbles: %x %x", l, rr)
	}
}

func TestRejectsGarbage(t *testing.T) {
	if _, err := FromBytes([]byte("hello")); err == nil {
		t.Fatal("garbage accepted")
	}
	b := append([]byte{0, 0, 0}, metaMarker...)
	b = append(b, 0xe0) // empty map: no node_count, record size 0
	if _, err := FromBytes(b); err == nil {
		t.Fatal("metadata without record size accepted")
	}
}

// IPinfo Lite keeps country and ASN in one database, the ASN as "AS15169".
func TestCombined(t *testing.T) {
	r, err := Open("testdata/ipinfo.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	if k := r.Kind(); k != "both" {
		t.Fatalf("kind %q", k)
	}
	if in, ok := r.Lookup(netip.MustParseAddr("8.8.8.8")); !ok || in != (Info{Country: "US", ASN: 15169, Org: "Google LLC"}) {
		t.Fatalf("%+v %v", in, ok)
	}
}

func TestBuiltin(t *testing.T) {
	c, a, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind() != "country" || a.Kind() != "asn" {
		t.Fatalf("kinds %q %q", c.Kind(), a.Kind())
	}
	if in, _ := c.Lookup(netip.MustParseAddr("8.8.8.8")); in.Country != "US" {
		t.Errorf("8.8.8.8 country %+v", in)
	}
	if in, _ := a.Lookup(netip.MustParseAddr("2606:4700:4700::1111")); in.ASN != 13335 {
		t.Errorf("Cloudflare ASN %+v", in)
	}
}
