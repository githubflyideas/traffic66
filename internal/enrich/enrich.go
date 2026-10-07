// Package enrich adds names and context to flow endpoints: locality,
// client/server role, service, ASN and country, threat lists and the
// operator's own inventory.
package enrich

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/geo"
)

// ---------------------------------------------------------------- locality

var specialNets = mustPrefixes(
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10",
	"127.0.0.0/8", "169.254.0.0/16", "fc00::/7", "fe80::/10", "::1/128",
)

func mustPrefixes(ss ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParsePrefix(s)
	}
	return out
}

// ---------------------------------------------------------------- inventory

// Network is a named address range from the inventory, optionally with
// the country it is in ("net 10.1.0.0/16 Tokyo office country=JP").
type Network struct {
	Prefix  netip.Prefix
	Name    string
	Country string
}

// Iface is a named interface of an exporter.
type Iface struct {
	Name  string
	Speed uint64 // bits per second, 0 if unknown
}

// Inventory holds operator-provided names. All fields are optional.
type Inventory struct {
	mu        sync.RWMutex
	networks  []Network // longest prefix first
	devices   map[netip.Addr]string
	ifaces    map[string]Iface // "exporter/ifindex"
	hosts     map[netip.Addr]string
	unsampled map[netip.Addr]bool
	snmp      []SNMPTarget
	// autoIfs are interface names learned from the devices (SNMP ifName);
	// names written in the inventory always win.
	autoIfs map[string]Iface
	path    string
	text    string
}

// SNMPTarget is an "snmp" inventory line.
type SNMPTarget struct {
	Exporter  netip.Addr
	Host      netip.AddrPort
	Community string
}

func NewInventory() *Inventory {
	return &Inventory{devices: map[netip.Addr]string{}, ifaces: map[string]Iface{}, hosts: map[netip.Addr]string{}, unsampled: map[netip.Addr]bool{}, autoIfs: map[string]Iface{}}
}

// LoadInventory reads the inventory file. A missing file is not an error.
func LoadInventory(path string) (*Inventory, error) {
	inv := NewInventory()
	inv.path = path
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return inv, nil
	}
	if err != nil {
		return nil, err
	}
	if err := inv.Parse(string(b)); err != nil {
		return nil, err
	}
	return inv, nil
}

// Parse replaces the inventory with the given text. Lines:
//
//	net    10.10.0.0/16   Office LAN
//	device 10.0.0.1       Core router       [unsampled]
//	iface  10.0.0.1 3     Uplink ISP-A 1G   [speed=1000000000]
//	host   10.10.3.27     Finance PC
//	snmp   10.0.0.1 public [192.168.1.1[:161]]
//
// An snmp line polls the device's interface counters with SNMPv2c; the
// optional last field is where to send the queries when the management
// address differs from the address the flows come from.
func (inv *Inventory) Parse(text string) error {
	nets := []Network{}
	devs := map[netip.Addr]string{}
	ifs := map[string]Iface{}
	hosts := map[netip.Addr]string{}
	uns := map[netip.Addr]bool{}
	var snmps []SNMPTarget
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		bad := func(msg string) error { return fmt.Errorf("line %d: %s", n+1, msg) }
		switch strings.ToLower(f[0]) {
		case "net":
			if len(f) < 3 {
				return bad("net needs a prefix and a name")
			}
			p, err := netip.ParsePrefix(f[1])
			if err != nil {
				return bad("bad prefix " + f[1])
			}
			nw := Network{Prefix: p.Masked()}
			var words []string
			for _, w := range f[2:] {
				if k, v, ok := strings.Cut(w, "="); ok && strings.EqualFold(k, "country") {
					if len(v) != 2 {
						return bad("country must be a two-letter code such as JP: " + v)
					}
					nw.Country = strings.ToUpper(v)
					continue
				}
				words = append(words, w)
			}
			if len(words) == 0 {
				return bad("net needs a prefix and a name")
			}
			nw.Name = strings.Join(words, " ")
			nets = append(nets, nw)
		case "device":
			if len(f) < 3 {
				return bad("device needs an address and a name")
			}
			a, err := netip.ParseAddr(f[1])
			if err != nil {
				return bad("bad address " + f[1])
			}
			words := []string{}
			for _, w := range f[2:] {
				if strings.EqualFold(w, "unsampled") {
					uns[a] = true
					continue
				}
				words = append(words, w)
			}
			devs[a] = strings.Join(words, " ")
		case "iface":
			if len(f) < 4 {
				return bad("iface needs an address, an index and a name")
			}
			a, err := netip.ParseAddr(f[1])
			if err != nil {
				return bad("bad address " + f[1])
			}
			idx, err := strconv.ParseUint(f[2], 10, 32)
			if err != nil {
				return bad("bad ifindex " + f[2])
			}
			it := Iface{}
			words := []string{}
			for _, w := range f[3:] {
				if strings.HasPrefix(w, "speed=") {
					it.Speed, _ = strconv.ParseUint(strings.TrimPrefix(w, "speed="), 10, 64)
					continue
				}
				words = append(words, w)
			}
			it.Name = strings.Join(words, " ")
			ifs[fmt.Sprintf("%s/%d", a, idx)] = it
		case "snmp":
			if len(f) < 3 || len(f) > 4 {
				return bad("snmp needs an exporter address, a community and optionally a management address")
			}
			a, err := netip.ParseAddr(f[1])
			if err != nil {
				return bad("bad address " + f[1])
			}
			host := netip.AddrPortFrom(a, 161)
			if len(f) == 4 {
				if ap, err := netip.ParseAddrPort(f[3]); err == nil {
					host = ap
				} else if ha, err := netip.ParseAddr(f[3]); err == nil {
					host = netip.AddrPortFrom(ha, 161)
				} else {
					return bad("bad management address " + f[3])
				}
			}
			snmps = append(snmps, SNMPTarget{Exporter: a, Host: host, Community: f[2]})
		case "host":
			if len(f) < 3 {
				return bad("host needs an address and a name")
			}
			a, err := netip.ParseAddr(f[1])
			if err != nil {
				return bad("bad address " + f[1])
			}
			hosts[a] = strings.Join(f[2:], " ")
		default:
			return bad("unknown keyword " + f[0])
		}
	}
	sort.SliceStable(nets, func(i, j int) bool { return nets[i].Prefix.Bits() > nets[j].Prefix.Bits() })
	inv.mu.Lock()
	inv.networks, inv.devices, inv.ifaces, inv.hosts, inv.unsampled, inv.snmp, inv.text = nets, devs, ifs, hosts, uns, snmps, text
	inv.mu.Unlock()
	return nil
}

// Save validates, applies and writes the inventory text.
func (inv *Inventory) Save(text string) error {
	if err := inv.Parse(text); err != nil {
		return err
	}
	if inv.path == "" {
		return nil
	}
	return os.WriteFile(inv.path, []byte(text), 0o644)
}

func (inv *Inventory) Text() string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.text
}

// Network returns the named network containing a, if any.
func (inv *Inventory) Network(a netip.Addr) (Network, bool) {
	a = a.Unmap()
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	for _, n := range inv.networks {
		if n.Prefix.Contains(a) {
			return n, true
		}
	}
	return Network{}, false
}

// Internal reports whether a belongs to the operator's own address space:
// private/special ranges plus every inventory network.
func (inv *Inventory) Internal(a netip.Addr) bool {
	a = a.Unmap()
	for _, p := range specialNets {
		if p.Contains(a) {
			return true
		}
	}
	_, ok := inv.Network(a)
	return ok
}

func (inv *Inventory) Device(a netip.Addr) string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.devices[a]
}

func (inv *Inventory) Iface(exporter netip.Addr, idx uint32) Iface {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	k := fmt.Sprintf("%s/%d", exporter, idx)
	it, ok := inv.ifaces[k]
	auto := inv.autoIfs[k]
	if !ok {
		return auto
	}
	if it.Speed == 0 {
		it.Speed = auto.Speed
	}
	return it
}

// SetAutoIface records a name and speed learned from the device.
func (inv *Inventory) SetAutoIface(exporter netip.Addr, idx uint32, name string, speed uint64) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if inv.autoIfs == nil {
		inv.autoIfs = map[string]Iface{}
	}
	inv.autoIfs[fmt.Sprintf("%s/%d", exporter, idx)] = Iface{Name: name, Speed: speed}
}

// SNMPTargets returns the devices to poll.
func (inv *Inventory) SNMPTargets() []SNMPTarget {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return append([]SNMPTarget(nil), inv.snmp...)
}

func (inv *Inventory) Host(a netip.Addr) string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.hosts[a.Unmap()]
}

// Hosts returns a copy of all named hosts keyed by address text.
func (inv *Inventory) Hosts() map[string]string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	out := make(map[string]string, len(inv.hosts))
	for a, n := range inv.hosts {
		out[a.String()] = n
	}
	return out
}

func (inv *Inventory) Unsampled() map[netip.Addr]bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	out := map[netip.Addr]bool{}
	for a := range inv.unsampled {
		out[a] = true
	}
	return out
}

// SegmentCountries maps network names to the country set for them.
func (inv *Inventory) SegmentCountries() map[string]string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	out := map[string]string{}
	for _, n := range inv.networks {
		if n.Country != "" {
			out[n.Name] = n.Country
		}
	}
	return out
}

// Segment names the network a client belongs to: the inventory name, or a
// /24 (IPv4) or /48 (IPv6) when unnamed.
func (inv *Inventory) Segment(a netip.Addr) string {
	a = a.Unmap()
	if n, ok := inv.Network(a); ok {
		return n.Name
	}
	bits := 24
	if a.Is6() {
		bits = 48
	}
	p, _ := a.Prefix(bits)
	return p.String()
}

// ---------------------------------------------------------------- ASN/geo

type asRange struct {
	lo, hi netip.Addr
	asn    uint32
	cc     string
	org    string
}

// ASNDB maps addresses to ASN, country and organisation. Sources: a
// tab-separated table (range_start range_end as_number country description,
// as published by iptoasn.com), MaxMind DB files (.mmdb, as GeoLite2, DB-IP
// Lite and IPinfo Lite) installed by the operator, and the DB-IP Lite
// databases built into the program (or a newer download of them). The
// operator's .mmdb files win over the table for what they hold; the DB-IP
// Lite databases answer only what nothing else does.
type ASNDB struct {
	mu      sync.RWMutex
	v4, v6  []asRange
	tsvFile string
	cty, as *geo.Reader
	ctyFile string
	asFile  string
	// fallback: DB-IP Lite, built in or downloaded
	fcty, fas         *geo.Reader
	fctyFile, fasFile string
}

// GeoSource describes one loaded database.
type GeoSource struct {
	Kind     string    `json:"kind"`     // "table", "country", "asn" or "both"
	File     string    `json:"file"`     // file name in the data directory, or "built-in"
	Type     string    `json:"type"`     // database type from the file
	Vendor   string    `json:"vendor"`   // dbip, maxmind, ipinfo, iptoasn or ""
	Built    time.Time `json:"built"`    // build date of an .mmdb
	Entries  int       `json:"entries"`  // ranges in a table
	Fallback bool      `json:"fallback"` // used only where the others have no answer
}

// Vendor guesses who published a database from its type name.
func Vendor(typ string) string {
	t := strings.ToLower(typ)
	switch {
	case strings.Contains(t, "dbip") || strings.Contains(t, "db-ip"):
		return "dbip"
	case strings.Contains(t, "geolite") || strings.Contains(t, "geoip2"):
		return "maxmind"
	case strings.Contains(t, "ipinfo"):
		return "ipinfo"
	}
	return ""
}

// SetMMDB installs an .mmdb reader as the country or ASN database, or both.
func (db *ASNDB) SetMMDB(r *geo.Reader, file string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	k := r.Kind()
	if k == "asn" || k == "both" {
		db.as, db.asFile = r, file
	}
	if k == "country" || k == "both" {
		db.cty, db.ctyFile = r, file
	}
}

// SetFallback installs the DB-IP Lite country and ASN databases; nil keeps
// the one in place.
func (db *ASNDB) SetFallback(cty, as *geo.Reader, ctyFile, asFile string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if cty != nil {
		db.fcty, db.fctyFile = cty, ctyFile
	}
	if as != nil {
		db.fas, db.fasFile = as, asFile
	}
}

// Fallback returns the DB-IP Lite databases in use.
func (db *ASNDB) Fallback() (cty, as *geo.Reader) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.fcty, db.fas
}

// Sources lists the loaded databases.
func (db *ASNDB) Sources() []GeoSource {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var out []GeoSource
	if n := len(db.v4) + len(db.v6); n > 0 {
		out = append(out, GeoSource{Kind: "table", File: db.tsvFile, Type: "IP to ASN table", Vendor: "iptoasn", Entries: n})
	}
	src := func(kind string, r *geo.Reader, file string, fb bool) GeoSource {
		v := Vendor(r.Type)
		if fb {
			v = "dbip"
		}
		return GeoSource{Kind: kind, File: file, Type: r.Type, Vendor: v, Built: r.Built, Fallback: fb}
	}
	if db.cty != nil && db.cty == db.as {
		out = append(out, src("both", db.cty, db.ctyFile, false))
	} else {
		if db.cty != nil {
			out = append(out, src("country", db.cty, db.ctyFile, false))
		}
		if db.as != nil {
			out = append(out, src("asn", db.as, db.asFile, false))
		}
	}
	if db.fcty != nil {
		out = append(out, src("country", db.fcty, db.fctyFile, true))
	}
	if db.fas != nil {
		out = append(out, src("asn", db.fas, db.fasFile, true))
	}
	return out
}

// Loaded reports whether any database is loaded.
func (db *ASNDB) Loaded() bool { return len(db.Sources()) > 0 }

func NewASNDB() *ASNDB { return &ASNDB{} }

func (db *ASNDB) Size() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.v4) + len(db.v6)
}

// LoadTable parses a TSV table without installing it, for validation.
func LoadTable(r io.Reader) (*ASNDB, error) {
	t := NewASNDB()
	if err := t.Load(r); err != nil {
		return nil, err
	}
	if t.Size() == 0 {
		return nil, errors.New("no address ranges found; expected tab-separated lines: first address, last address, AS number, country, name")
	}
	return t, nil
}

// TakeTable installs the ranges of another ASNDB as this one's table.
func (db *ASNDB) TakeTable(o *ASNDB, file string) {
	o.mu.RLock()
	v4, v6 := o.v4, o.v6
	o.mu.RUnlock()
	db.mu.Lock()
	db.v4, db.v6, db.tsvFile = v4, v6, file
	db.mu.Unlock()
}

// LoadFile reads a .tsv or .tsv.gz file and replaces the table.
func (db *ASNDB) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	defer func() { db.mu.Lock(); db.tsvFile = filepath.Base(path); db.mu.Unlock() }()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}
	return db.Load(r)
}

// Load reads TSV rows from r and replaces the table.
func (db *ASNDB) Load(r io.Reader) error {
	var v4, v6 []asRange
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) < 5 {
			continue
		}
		lo, err1 := netip.ParseAddr(f[0])
		hi, err2 := netip.ParseAddr(f[1])
		asn, err3 := strconv.ParseUint(f[2], 10, 32)
		if err1 != nil || err2 != nil || err3 != nil || asn == 0 {
			continue
		}
		cc := f[3]
		if cc == "None" {
			cc = ""
		}
		e := asRange{lo.Unmap(), hi.Unmap(), uint32(asn), cc, f[4]}
		if e.lo.Is4() {
			v4 = append(v4, e)
		} else {
			v6 = append(v6, e)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	byLo := func(s []asRange) { sort.Slice(s, func(i, j int) bool { return s[i].lo.Less(s[j].lo) }) }
	byLo(v4)
	byLo(v6)
	db.mu.Lock()
	db.v4, db.v6 = v4, v6
	db.mu.Unlock()
	return nil
}

// Lookup returns the ASN, country code and organisation for a.
func (db *ASNDB) Lookup(a netip.Addr) (asn uint32, cc, org string) {
	a = a.Unmap()
	db.mu.RLock()
	defer db.mu.RUnlock()
	t := db.v4
	if a.Is6() {
		t = db.v6
	}
	if i := sort.Search(len(t), func(i int) bool { return a.Less(t[i].lo) }) - 1; i >= 0 && !t[i].hi.Less(a) {
		asn, cc, org = t[i].asn, t[i].cc, t[i].org
	}
	if db.cty != nil {
		if in, ok := db.cty.Lookup(a); ok && in.Country != "" {
			cc = in.Country
		}
	}
	if db.as != nil {
		if in, ok := db.as.Lookup(a); ok && in.ASN != 0 {
			asn, org = in.ASN, in.Org
		}
	}
	if cc == "" && db.fcty != nil {
		if in, ok := db.fcty.Lookup(a); ok {
			cc = in.Country
		}
	}
	if asn == 0 && db.fas != nil {
		if in, ok := db.fas.Lookup(a); ok && in.ASN != 0 {
			asn, org = in.ASN, in.Org
		}
	}
	return asn, cc, org
}

// ---------------------------------------------------------------- threats

// Threats holds named lists of addresses and prefixes.
type Threats struct {
	mu    sync.RWMutex
	exact map[netip.Addr]string
	nets  []Network
	lists map[string]int
}

func NewThreats() *Threats { return &Threats{exact: map[netip.Addr]string{}, lists: map[string]int{}} }

// Add loads one list; lines hold an address or prefix, '#' and ';' start
// comments.
func (t *Threats) Add(name string, r io.Reader) (int, error) {
	exact := map[netip.Addr]string{}
	var nets []Network
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexAny(line, "#;"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		field := strings.Fields(line)[0]
		if p, err := netip.ParsePrefix(field); err == nil {
			if p.IsSingleIP() {
				exact[p.Addr().Unmap()] = name
			} else {
				nets = append(nets, Network{Prefix: p.Masked(), Name: name})
			}
			n++
			continue
		}
		if a, err := netip.ParseAddr(field); err == nil {
			exact[a.Unmap()] = name
			n++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for a := range t.exact {
		if t.exact[a] == name {
			delete(t.exact, a)
		}
	}
	keep := t.nets[:0]
	for _, x := range t.nets {
		if x.Name != name {
			keep = append(keep, x)
		}
	}
	t.nets = append(keep, nets...)
	for a, v := range exact {
		t.exact[a] = v
	}
	t.lists[name] = n
	return n, nil
}

func (t *Threats) Match(a netip.Addr) string {
	a = a.Unmap()
	t.mu.RLock()
	defer t.mu.RUnlock()
	if n, ok := t.exact[a]; ok {
		return n
	}
	for _, x := range t.nets {
		if x.Prefix.Contains(a) {
			return x.Name
		}
	}
	return ""
}

func (t *Threats) Lists() map[string]int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := map[string]int{}
	for k, v := range t.lists {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------- geo files

// MMDBFiles are the operator's databases in the data directory, in loading
// order: a later file replaces what an earlier one holds.
var MMDBFiles = []string{"both.mmdb", "country.mmdb", "asn.mmdb"}

// LoadMMDBs (re)loads the operator's .mmdb files from dir; databases whose
// file is gone are dropped.
func (db *ASNDB) LoadMMDBs(dir string) []error {
	db.mu.Lock()
	db.cty, db.as, db.ctyFile, db.asFile = nil, nil, "", ""
	db.mu.Unlock()
	var errs []error
	for _, name := range MMDBFiles {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		r, err := geo.Open(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			continue
		}
		db.SetMMDB(r, name)
	}
	return errs
}

// DB-IP Lite files downloaded into the data directory.
const (
	DBIPCountryFile = "dbip-country.mmdb"
	DBIPASNFile     = "dbip-asn.mmdb"
)

// LoadDBIP installs the DB-IP Lite databases: a download in dir when it is
// newer than the built-in one, else the built-in one.
func (db *ASNDB) LoadDBIP(dir string) error {
	bc, ba, err := geo.Builtin()
	if err != nil {
		return fmt.Errorf("built-in country and ASN data: %w", err)
	}
	pick := func(b *geo.Reader, name string) (*geo.Reader, string) {
		if dir != "" {
			if r, err := geo.Open(filepath.Join(dir, name)); err == nil && r.Built.After(b.Built) {
				return r, name
			}
		}
		return b, "built-in"
	}
	c, cf := pick(bc, DBIPCountryFile)
	a, af := pick(ba, DBIPASNFile)
	db.SetFallback(c, a, cf, af)
	return nil
}

// TableFile is the file the table was loaded from.
func (db *ASNDB) TableFile() string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if len(db.v4)+len(db.v6) == 0 {
		return ""
	}
	return filepath.Base(db.tsvFile)
}
