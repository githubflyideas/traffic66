// Package geo reads MaxMind DB files (.mmdb), the format of MaxMind GeoLite2
// and DB-IP Lite databases, following the published format specification.
// Only what traffic66 needs is decoded: country code, AS number and AS
// organisation.
package geo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var metaMarker = []byte("\xab\xcd\xefMaxMind.com")

// Reader is an open .mmdb database held in memory.
type Reader struct {
	buf        []byte
	data       []byte // data section
	nodeCount  uint32
	recordSize uint32
	ipVersion  uint16
	v4Start    uint32 // node reached after the 96 zero bits of an IPv4 address

	Type  string    // database_type, e.g. "GeoLite2-Country"
	Built time.Time // build_epoch

	mu    sync.Mutex
	cache map[uint32]Info
}

// Info is what traffic66 uses from a record.
type Info struct {
	Country string // ISO code
	ASN     uint32
	Org     string
}

// Open reads the file at path.
func Open(path string) (*Reader, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return FromBytes(b)
}

// IsMMDB reports whether b looks like a MaxMind DB.
func IsMMDB(b []byte) bool { return bytes.LastIndex(b, metaMarker) >= 0 }

// FromBytes parses a database held in b.
func FromBytes(b []byte) (*Reader, error) {
	i := bytes.LastIndex(b, metaMarker)
	if i < 0 {
		return nil, errors.New("not a MaxMind DB file (.mmdb)")
	}
	md, _, err := decode(b[i+len(metaMarker):], 0, 0)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}
	m, ok := md.(map[string]any)
	if !ok {
		return nil, errors.New("metadata is not a map")
	}
	r := &Reader{buf: b, cache: map[uint32]Info{}}
	r.nodeCount = uint32(asUint(m["node_count"]))
	r.recordSize = uint32(asUint(m["record_size"]))
	r.ipVersion = uint16(asUint(m["ip_version"]))
	r.Type, _ = m["database_type"].(string)
	if e := asUint(m["build_epoch"]); e > 0 {
		r.Built = time.Unix(int64(e), 0).UTC()
	}
	switch r.recordSize {
	case 24, 28, 32:
	default:
		return nil, fmt.Errorf("unsupported record size %d", r.recordSize)
	}
	if r.ipVersion != 4 && r.ipVersion != 6 {
		return nil, fmt.Errorf("unsupported IP version %d", r.ipVersion)
	}
	treeSize := uint64(r.nodeCount) * uint64(r.recordSize) / 4
	if treeSize+16 > uint64(i) {
		return nil, errors.New("search tree larger than the file")
	}
	r.data = b[treeSize+16 : i]
	if r.ipVersion == 6 {
		node := uint32(0)
		for k := 0; k < 96 && node < r.nodeCount; k++ {
			node = r.record(node, 0)
		}
		r.v4Start = node
	}
	return r, nil
}

// record returns the left (bit 0) or right (bit 1) record of node.
func (r *Reader) record(node uint32, bit int) uint32 {
	t := r.buf
	switch r.recordSize {
	case 24:
		o := node * 6
		if bit == 1 {
			o += 3
		}
		return uint32(t[o])<<16 | uint32(t[o+1])<<8 | uint32(t[o+2])
	case 28:
		o := node * 7
		if bit == 0 {
			return uint32(t[o+3]&0xf0)<<20 | uint32(t[o])<<16 | uint32(t[o+1])<<8 | uint32(t[o+2])
		}
		return uint32(t[o+3]&0x0f)<<24 | uint32(t[o+4])<<16 | uint32(t[o+5])<<8 | uint32(t[o+6])
	default:
		o := node * 8
		if bit == 1 {
			o += 4
		}
		return binary.BigEndian.Uint32(t[o:])
	}
}

// Lookup returns the record for a, or ok=false when the database has none.
func (r *Reader) Lookup(a netip.Addr) (Info, bool) {
	a = a.Unmap()
	var bits []byte
	node := uint32(0)
	if a.Is4() {
		b := a.As4()
		bits = b[:]
		if r.ipVersion == 6 {
			node = r.v4Start
		}
	} else {
		if r.ipVersion == 4 {
			return Info{}, false
		}
		b := a.As16()
		bits = b[:]
	}
	for i := 0; i < len(bits)*8 && node < r.nodeCount; i++ {
		node = r.record(node, int(bits[i/8]>>(7-i%8)&1))
	}
	if node <= r.nodeCount {
		return Info{}, false // node_count means "no data"
	}
	off := node - r.nodeCount - 16
	r.mu.Lock()
	if in, ok := r.cache[off]; ok {
		r.mu.Unlock()
		return in, true
	}
	r.mu.Unlock()
	v, _, err := decode(r.data, int(off), 0)
	if err != nil {
		return Info{}, false
	}
	in := infoFrom(v)
	r.mu.Lock()
	if len(r.cache) > 1<<20 {
		r.cache = map[uint32]Info{}
	}
	r.cache[off] = in
	r.mu.Unlock()
	return in, true
}

func infoFrom(v any) Info {
	m, _ := v.(map[string]any)
	var in Info
	for _, k := range []string{"country", "registered_country", "represented_country"} {
		if c, ok := m[k].(map[string]any); ok {
			if s, ok := c["iso_code"].(string); ok && s != "" {
				in.Country = s
				break
			}
		}
	}
	if in.Country == "" { // ip-location-db and similar flat layouts
		in.Country, _ = m["country_code"].(string)
	}
	in.ASN = uint32(asUint(m["autonomous_system_number"]))
	in.Org, _ = m["autonomous_system_organization"].(string)
	if in.ASN == 0 { // IPinfo Lite: "asn": "AS15169", "as_name": "Google LLC"
		if s, ok := m["asn"].(string); ok {
			n, _ := strconv.ParseUint(strings.TrimPrefix(strings.ToUpper(s), "AS"), 10, 32)
			in.ASN = uint32(n)
			in.Org, _ = m["as_name"].(string)
		} else {
			in.ASN = uint32(asUint(m["asn"]))
		}
	}
	return in
}

func asUint(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case int64:
		if x > 0 {
			return uint64(x)
		}
	case float64:
		if x > 0 {
			return uint64(x)
		}
	}
	return 0
}

// decode reads the value at off in d and returns it and the offset after it.
// depth guards against malformed files that loop through pointers.
func decode(d []byte, off, depth int) (any, int, error) {
	if depth > 32 {
		return nil, 0, errors.New("nesting too deep")
	}
	need := func(n int) error {
		if off+n > len(d) || off+n < off {
			return errors.New("truncated data")
		}
		return nil
	}
	if err := need(1); err != nil {
		return nil, 0, err
	}
	ctrl := d[off]
	off++
	typ := int(ctrl >> 5)
	if typ == 1 { // pointer
		ss, vvv := int(ctrl>>3)&3, uint32(ctrl&7)
		if err := need(ss + 1); err != nil {
			return nil, 0, err
		}
		var p uint32
		switch ss {
		case 0:
			p = vvv<<8 | uint32(d[off])
		case 1:
			p = (vvv<<16 | uint32(d[off])<<8 | uint32(d[off+1])) + 2048
		case 2:
			p = (vvv<<24 | uint32(d[off])<<16 | uint32(d[off+1])<<8 | uint32(d[off+2])) + 526336
		default:
			p = binary.BigEndian.Uint32(d[off:])
		}
		v, _, err := decode(d, int(p), depth+1)
		return v, off + ss + 1, err
	}
	if typ == 0 { // extended type
		if err := need(1); err != nil {
			return nil, 0, err
		}
		typ = 7 + int(d[off])
		off++
	}
	size := int(ctrl & 0x1f)
	if size >= 29 {
		n := size - 28
		if err := need(n); err != nil {
			return nil, 0, err
		}
		switch n {
		case 1:
			size = 29 + int(d[off])
		case 2:
			size = 285 + int(d[off])<<8 + int(d[off+1])
		default:
			size = 65821 + int(d[off])<<16 + int(d[off+1])<<8 + int(d[off+2])
		}
		off += n
	}
	uintN := func() (uint64, error) {
		if err := need(size); err != nil || size > 8 {
			return 0, errors.New("bad integer")
		}
		var v uint64
		for _, c := range d[off : off+size] {
			v = v<<8 | uint64(c)
		}
		return v, nil
	}
	switch typ {
	case 2: // UTF-8 string
		if err := need(size); err != nil {
			return nil, 0, err
		}
		return string(d[off : off+size]), off + size, nil
	case 3: // double
		if err := need(8); err != nil {
			return nil, 0, err
		}
		return math.Float64frombits(binary.BigEndian.Uint64(d[off:])), off + 8, nil
	case 4: // bytes
		if err := need(size); err != nil {
			return nil, 0, err
		}
		return d[off : off+size], off + size, nil
	case 5, 6, 9: // uint16, uint32, uint64
		v, err := uintN()
		return v, off + size, err
	case 8: // int32
		v, err := uintN()
		return int64(int32(uint32(v))), off + size, err
	case 10: // uint128: not needed, skipped
		if err := need(size); err != nil {
			return nil, 0, err
		}
		return nil, off + size, nil
	case 7: // map
		m := make(map[string]any, size)
		for i := 0; i < size; i++ {
			k, next, err := decode(d, off, depth+1)
			if err != nil {
				return nil, 0, err
			}
			ks, ok := k.(string)
			if !ok {
				return nil, 0, errors.New("map key is not a string")
			}
			v, next2, err := decode(d, next, depth+1)
			if err != nil {
				return nil, 0, err
			}
			m[ks] = v
			off = next2
		}
		return m, off, nil
	case 11: // array
		a := make([]any, 0, min(size, 1024))
		for i := 0; i < size; i++ {
			v, next, err := decode(d, off, depth+1)
			if err != nil {
				return nil, 0, err
			}
			a = append(a, v)
			off = next
		}
		return a, off, nil
	case 14: // boolean: the value is the size
		return size != 0, off, nil
	case 15: // float
		if err := need(4); err != nil {
			return nil, 0, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(d[off:]))), off + 4, nil
	}
	return nil, 0, fmt.Errorf("unsupported data type %d", typ)
}

// probes are well-known addresses every country or ASN database covers.
var probes = []netip.Addr{
	netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("9.9.9.9"),
	netip.MustParseAddr("208.67.222.222"), netip.MustParseAddr("114.114.114.114"), netip.MustParseAddr("77.88.8.8"),
	netip.MustParseAddr("2001:4860:4860::8888"), netip.MustParseAddr("2606:4700:4700::1111"),
}

// Kind tells whether a database holds countries ("country"), AS numbers
// ("asn") or both ("both"), from what it answers for a few well-known
// addresses, or else from its type name.
func (r *Reader) Kind() string {
	var cty, as bool
	for _, a := range probes {
		if in, ok := r.Lookup(a); ok {
			cty = cty || in.Country != ""
			as = as || in.ASN != 0
		}
	}
	switch {
	case cty && as:
		return "both"
	case as:
		return "asn"
	case cty:
		return "country"
	}
	t := strings.ToLower(r.Type)
	if strings.Contains(t, "asn") || strings.Contains(t, "-as-") || strings.Contains(t, "isp") {
		return "asn"
	}
	return "country"
}
