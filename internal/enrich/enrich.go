// Package enrich adds names and context to flow endpoints: locality,
// client/server role, service, ASN and country, threat lists and the
// operator's own inventory.
package enrich

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// Network is a named address range from the inventory.
type Network struct {
	Prefix netip.Prefix
	Name   string
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
			nets = append(nets, Network{p.Masked(), strings.Join(f[2:], " ")})
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

// ASNDB maps address ranges to ASN, country and organisation, loaded from a
// tab-separated file: range_start range_end as_number country description.
type ASNDB struct {
	mu     sync.RWMutex
	v4, v6 []asRange
}

func NewASNDB() *ASNDB { return &ASNDB{} }

func (db *ASNDB) Size() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.v4) + len(db.v6)
}

// LoadFile reads a .tsv or .tsv.gz file and replaces the table.
func (db *ASNDB) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
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
func (db *ASNDB) Lookup(a netip.Addr) (uint32, string, string) {
	a = a.Unmap()
	db.mu.RLock()
	defer db.mu.RUnlock()
	t := db.v4
	if a.Is6() {
		t = db.v6
	}
	i := sort.Search(len(t), func(i int) bool { return a.Less(t[i].lo) }) - 1
	if i < 0 || t[i].hi.Less(a) {
		return 0, "", ""
	}
	return t[i].asn, t[i].cc, t[i].org
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
				nets = append(nets, Network{p.Masked(), name})
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
