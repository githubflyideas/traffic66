package assets

import (
	"fmt"
	"strconv"
	"strings"
)

// Importing a switch's own output. An operator runs the command they
// already run, pastes what it printed, and the ledger reads it: there is
// nothing to install on the device, no SNMP community to hand out and no
// credentials to store anywhere.
//
// These outputs are column-formatted, and the columns are what makes them
// readable: a field can hold spaces ("Gig 0/1", "a-full"), so splitting on
// whitespace loses them. The header line is therefore found first and its
// column offsets are used to cut every row. A line that does not fit is
// reported back rather than guessed at — the whole point of a ledger is
// that nothing in it was invented.

// Paste formats.
const (
	FormatNeighbors = "neighbors" // show lldp neighbors, show cdp neighbors
	FormatIfStatus  = "ifstatus"  // show interface status
	FormatARP       = "arp"       // show ip arp, show arp
	FormatMACTable  = "mactable"  // show mac address-table
)

// Change is one thing an import would do, or did.
type Change struct {
	Kind   string `json:"kind"`   // device, port, link, addr
	Action string `json:"action"` // add, update, same
	What   string `json:"what"`   // for the preview, in the operator's terms
	Error  string `json:"error,omitempty"`
}

// Plan is what a paste would do. It is produced without writing anything,
// shown, and only then applied — so the operator sees what a paste of the
// wrong buffer would have done before it does it.
type Plan struct {
	Format  string   `json:"format"`
	Changes []Change `json:"changes"`
	Skipped []string `json:"skipped,omitempty"` // lines that were not understood
	Applied bool     `json:"applied"`
}

// columns are the fields of a header line: where each begins, and the
// name this understands it as ("" for a column it does not know). Every
// field is kept, known or not, because a column nobody recognises is
// still what ends the one before it — which is exactly what goes wrong
// when only the known ones are remembered.
type columns struct {
	at    []int
	names []string
}

// cut returns the value of one column of a row. The last column runs to
// the end of the line, and a value that overran its column (a long
// interface name does) keeps its tail up to the next gap, because the
// header says where fields begin, not where they must stop.
func (c columns) cut(line, name string) string {
	i := -1
	for k, n := range c.names {
		if n == name {
			i = k
			break
		}
	}
	if i < 0 || c.at[i] >= len(line) {
		return ""
	}
	end := len(line)
	if i+1 < len(c.at) && c.at[i+1] < end {
		end = c.at[i+1]
	}
	v := line[c.at[i]:end]
	// a value only overran when the cut falls inside a word; a value that
	// simply fills its column, as a right-aligned one does, has not
	if end < len(line) && line[end] != ' ' && len(v) > 0 && v[len(v)-1] != ' ' {
		rest := line[end:]
		if j := strings.Index(rest, "  "); j >= 0 {
			v += rest[:j]
		} else {
			v += rest
		}
	}
	return strings.TrimSpace(v)
}

// headerAliases maps the column names this understands to the spellings
// the common command outputs use, lowercased.
var headerAliases = map[string][]string{
	"device":   {"device id", "chassis id", "system name", "neighbor", "neighbour", "device"},
	"local":    {"local intf", "local intrfce", "local interface", "local port", "local"},
	"remote":   {"port id", "neighbor port", "neighbour port", "remote port", "port descr"},
	"port":     {"ports", "port", "interface", "intf"},
	"name":     {"name", "description", "descr"},
	"speed":    {"speed"},
	"type":     {"type"},
	"vlan":     {"vlan"},
	"address":  {"address", "ip address", "ip"},
	"hardware": {"hardware addr", "hardware address", "mac address", "mac addr", "mac"},
}

// headerFields cuts a line into its columns: a run of two or more spaces
// separates them, so that "Hardware Addr" and "Age (min)" stay whole.
func headerFields(line string) (at []int, text []string) {
	i := 0
	for i < len(line) {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}
		j := i
		for j < len(line) {
			if line[j] == ' ' && (j+1 >= len(line) || line[j+1] == ' ') {
				break
			}
			j++
		}
		at = append(at, i)
		text = append(text, strings.TrimSpace(line[i:j]))
		i = j
	}
	return at, text
}

// nameOf matches one header field against the aliases, whole.
func nameOf(field string) string {
	f := strings.ToLower(strings.TrimRight(field, ":"))
	for name, aliases := range headerAliases {
		for _, a := range aliases {
			if f == a {
				return name
			}
		}
	}
	return ""
}

// nameOfLoose also accepts a field that begins with an alias, for a
// header that carries a unit or a qualifier after the name.
func nameOfLoose(field string) string {
	if n := nameOf(field); n != "" {
		return n
	}
	f := strings.ToLower(field)
	best, bestLen := "", 0
	for name, aliases := range headerAliases {
		for _, a := range aliases {
			if len(a) > bestLen && strings.HasPrefix(f, a+" ") {
				best, bestLen = name, len(a)
			}
		}
	}
	return best
}

// splitField decides whether a header field holding single spaces is one
// column or several. The outputs give no way to tell from the spacing
// alone — "Hardware Addr" is one column and "Speed Type" is two — so a
// field that is not itself a column name is split only when at least two
// of its words are column names.
func splitField(at int, field string) (ats []int, names []string) {
	parts := strings.Split(field, " ")
	if len(parts) < 2 {
		return nil, nil
	}
	var offs []int
	var ns []string
	known, pos := 0, at
	for _, w := range parts {
		n := nameOf(w)
		if n != "" {
			known++
		}
		offs = append(offs, pos)
		ns = append(ns, n)
		pos += len(w) + 1
	}
	if known < 2 {
		return nil, nil
	}
	return offs, ns
}

// readHeader reads a header line. A column name that appears twice is
// kept only the first time, so a second "Type" cannot take the first's
// place.
func readHeader(line string) (columns, bool) {
	at, text := headerFields(line)
	if len(at) < 2 {
		return columns{}, false
	}
	var c columns
	seen := map[string]bool{}
	known := 0
	put := func(at int, n string) {
		if n != "" && seen[n] {
			n = ""
		}
		if n != "" {
			seen[n] = true
			known++
		}
		c.at = append(c.at, at)
		c.names = append(c.names, n)
	}
	for i, f := range text {
		// a field that is itself a column name is never split: "Mac
		// Address" is one column, however many of its words look like one
		if n := nameOf(f); n != "" {
			put(at[i], n)
			continue
		}
		if offs, ns := splitField(at[i], f); offs != nil {
			for k := range offs {
				put(offs[k], ns[k])
			}
			continue
		}
		put(at[i], nameOfLoose(f))
	}
	return c, known >= 2
}

// has reports whether the table carries all of these columns.
func (c columns) has(names ...string) bool {
	for _, n := range names {
		found := false
		for _, h := range c.names {
			if h == n {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// guessFormat names the table from the columns it carries.
func guessFormat(c columns) string {
	switch {
	case c.has("device", "local", "remote"):
		return FormatNeighbors
	case c.has("address", "hardware"):
		return FormatARP
	case c.has("hardware", "port"):
		return FormatMACTable
	case c.has("port", "speed"), c.has("port", "vlan"), c.has("port", "name"):
		return FormatIfStatus
	}
	return ""
}

// table finds the header of the first table in the text and returns it
// with the rows under it.
func table(text string) (columns, []string, bool) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, l := range lines {
		c, ok := readHeader(l)
		if !ok || guessFormat(c) == "" {
			continue
		}
		var rows []string
		for _, r := range lines[i+1:] {
			if strings.TrimSpace(r) == "" {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(r), "---") || strings.HasPrefix(strings.TrimSpace(r), "===") {
				continue
			}
			rows = append(rows, strings.TrimRight(r, " \t"))
		}
		return c, rows, true
	}
	return columns{}, nil, false
}

// samePort reports whether two interface names are the same port written
// two ways: "Gi0/1", "Gig 0/1" and "GigabitEthernet0/1" all name one port,
// because the vendors abbreviate their own names in their own output.
func samePort(a, b string) bool {
	na, ta := splitIface(a)
	nb, tb := splitIface(b)
	if ta != tb || ta == "" {
		return strings.EqualFold(strings.ReplaceAll(a, " ", ""), strings.ReplaceAll(b, " ", ""))
	}
	return strings.HasPrefix(na, nb) || strings.HasPrefix(nb, na)
}

// splitIface cuts an interface name into its alphabetic prefix,
// lowercased and without spaces, and its numeric tail ("Gig 0/1" ->
// "gig", "0/1").
func splitIface(s string) (name, tail string) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || s[i] == ' ' || s[i] == '-') {
		i++
	}
	return strings.ToLower(strings.ReplaceAll(s[:i], " ", "")), strings.ReplaceAll(s[i:], " ", "")
}

// parseSpeed reads the speed column of an interface table: "1000",
// "a-1000" (auto-negotiated), "10G", "a-10G". Anything else is 0, because
// a speed the ledger made up is worse than no speed.
func parseSpeed(s string) uint64 {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "a-")
	mult := uint64(1_000_000) // the column is megabits unless it says otherwise
	switch {
	case strings.HasSuffix(s, "gb/s"), strings.HasSuffix(s, "g"):
		s, mult = strings.TrimSuffix(strings.TrimSuffix(s, "gb/s"), "g"), 1_000_000_000
	case strings.HasSuffix(s, "mb/s"), strings.HasSuffix(s, "m"):
		s = strings.TrimSuffix(strings.TrimSuffix(s, "mb/s"), "m")
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || n <= 0 {
		return 0
	}
	return uint64(n * float64(mult))
}

// PasteOptions say what a paste is about.
type PasteOptions struct {
	Text     string
	DeviceID int64 // the device the command was run on; needed by all but ARP
	Apply    bool
}

// Paste reads a pasted command output. With Apply false nothing is
// written and the plan is only described.
func (s *Store) Paste(o PasteOptions) (Plan, error) {
	c, rows, ok := table(o.Text)
	if !ok {
		return Plan{}, fmt.Errorf("this does not look like the output of show lldp/cdp neighbors, show interface status, show ip arp or show mac address-table")
	}
	p := Plan{Format: guessFormat(c), Applied: o.Apply}
	var err error
	switch p.Format {
	case FormatNeighbors:
		err = s.pasteNeighbors(&p, c, rows, o)
	case FormatIfStatus:
		err = s.pasteIfStatus(&p, c, rows, o)
	case FormatARP:
		err = s.pasteARP(&p, c, rows, o)
	case FormatMACTable:
		err = s.pasteMACTable(&p, c, rows, o)
	}
	return p, err
}

// needDevice checks that the operator said which device the output came
// from. The command does not say so itself, and guessing would hang
// someone else's ports off the wrong box.
func (s *Store) needDevice(id int64) (Device, error) {
	if id == 0 {
		return Device{}, fmt.Errorf("say which device this was run on")
	}
	ds, err := s.Devices(DeviceQuery{})
	if err != nil {
		return Device{}, err
	}
	for _, d := range ds {
		if d.ID == id {
			return d, nil
		}
	}
	return Device{}, ErrNotFound
}

// port finds a device's port by name, however the vendor spelled it, and
// adds it when add is set.
func (s *Store) port(deviceID int64, name string, add bool, p *Plan) (Port, bool, error) {
	ps, err := s.Ports(deviceID)
	if err != nil {
		return Port{}, false, err
	}
	for _, x := range ps {
		if samePort(x.Name, name) {
			return x, false, nil
		}
	}
	if !add {
		return Port{}, false, nil
	}
	np := Port{DeviceID: deviceID, Name: name}
	if p.Applied {
		np, err = s.SavePort(np)
		if err != nil {
			return Port{}, false, err
		}
	}
	return np, true, nil
}

// device finds a device by name, or adds one.
func (s *Store) device(name string, p *Plan) (Device, bool, error) {
	ds, err := s.Devices(DeviceQuery{})
	if err != nil {
		return Device{}, false, err
	}
	// a neighbour often reports its fully qualified name
	short := strings.SplitN(name, ".", 2)[0]
	for _, d := range ds {
		if strings.EqualFold(d.Name, name) || strings.EqualFold(d.Name, short) {
			return d, false, nil
		}
	}
	nd := Device{Name: name}
	if p.Applied {
		nd, err = s.SaveDevice(nd)
		if err != nil {
			return Device{}, false, err
		}
	}
	return nd, true, nil
}

func (p *Plan) add(kind, action, what string, err error) {
	c := Change{Kind: kind, Action: action, What: what}
	if err != nil {
		c.Action, c.Error = "error", err.Error()
	}
	p.Changes = append(p.Changes, c)
}

// pasteNeighbors reads show lldp/cdp neighbors: the local port, the
// device at the other end and its port. It records the neighbour and the
// cable, which is the part nobody keeps up to date by hand.
func (s *Store) pasteNeighbors(p *Plan, c columns, rows []string, o PasteOptions) error {
	me, err := s.needDevice(o.DeviceID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		local, peer, rport := c.cut(r, "local"), c.cut(r, "device"), c.cut(r, "remote")
		if local == "" || peer == "" || rport == "" {
			p.Skipped = append(p.Skipped, strings.TrimSpace(r))
			continue
		}
		lp, added, err := s.port(me.ID, local, true, p)
		if err != nil {
			p.add("port", "add", me.Name+" "+local, err)
			continue
		}
		if added {
			p.add("port", "add", me.Name+" "+local, nil)
		}
		pd, added, err := s.device(peer, p)
		if err != nil {
			p.add("device", "add", peer, err)
			continue
		}
		if added {
			p.add("device", "add", peer, nil)
		}
		pp, added, err := s.port(pd.ID, rport, true, p)
		if err != nil {
			p.add("port", "add", peer+" "+rport, err)
			continue
		}
		if added {
			p.add("port", "add", peer+" "+rport, nil)
		}
		what := fmt.Sprintf("%s %s — %s %s", me.Name, local, peer, rport)
		if p.Applied {
			if _, err := s.SaveLink(Link{APort: lp.ID, BPort: pp.ID}); err != nil {
				p.add("link", "add", what, err)
				continue
			}
		}
		p.add("link", "add", what, nil)
	}
	return nil
}

// pasteIfStatus reads show interface status: the device's own ports, with
// the description and the speed it is running at.
func (s *Store) pasteIfStatus(p *Plan, c columns, rows []string, o PasteOptions) error {
	me, err := s.needDevice(o.DeviceID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		name := c.cut(r, "port")
		if name == "" {
			p.Skipped = append(p.Skipped, strings.TrimSpace(r))
			continue
		}
		cur, added, err := s.port(me.ID, name, true, p)
		if err != nil {
			p.add("port", "add", me.Name+" "+name, err)
			continue
		}
		cur.Note, cur.Kind = c.cut(r, "name"), c.cut(r, "type")
		if sp := parseSpeed(c.cut(r, "speed")); sp > 0 {
			cur.Speed = sp
		}
		act := "update"
		if added {
			act = "add"
		}
		if p.Applied {
			if _, err := s.SavePort(cur); err != nil {
				p.add("port", act, me.Name+" "+name, err)
				continue
			}
		}
		p.add("port", act, me.Name+" "+name, nil)
	}
	return nil
}

// pasteARP reads show ip arp: which address answered from which MAC. It
// is the half of the picture the ledger cannot work out on its own, and
// the half a mac-address-table paste needs to be worth anything.
func (s *Store) pasteARP(p *Plan, c columns, rows []string, o PasteOptions) error {
	have, err := s.Addrs(AddrQuery{Limit: 20000})
	if err != nil {
		return err
	}
	by := make(map[string]Addr, len(have))
	for _, a := range have {
		by[a.IP] = a
	}
	for _, r := range rows {
		ip, err := canonIP(c.cut(r, "address"))
		mac := CanonMAC(c.cut(r, "hardware"))
		if err != nil || mac == "" {
			p.Skipped = append(p.Skipped, strings.TrimSpace(r))
			continue
		}
		cur, known := by[ip]
		if known && cur.MAC == mac {
			p.add("addr", "same", ip+" "+mac, nil)
			continue
		}
		cur.IP, cur.MAC = ip, mac
		act := "update"
		if !known {
			act, cur.Kind = "add", KindStatic
		}
		if p.Applied {
			if _, err := s.SaveAddr(cur); err != nil {
				p.add("addr", act, ip+" "+mac, err)
				continue
			}
		}
		p.add("addr", act, ip+" "+mac, nil)
	}
	return nil
}

// pasteMACTable reads show mac address-table: which MAC is on which port.
// On its own that is a list of MACs; after an ARP paste it is what finally
// says which port a given address is plugged into, which is the question
// the ledger exists to answer and the one nobody can answer from memory.
func (s *Store) pasteMACTable(p *Plan, c columns, rows []string, o PasteOptions) error {
	me, err := s.needDevice(o.DeviceID)
	if err != nil {
		return err
	}
	have, err := s.Addrs(AddrQuery{Limit: 20000})
	if err != nil {
		return err
	}
	byMAC := map[string][]Addr{}
	for _, a := range have {
		if a.MAC != "" {
			byMAC[a.MAC] = append(byMAC[a.MAC], a)
		}
	}
	for _, r := range rows {
		mac, pname := CanonMAC(c.cut(r, "hardware")), c.cut(r, "port")
		if mac == "" || pname == "" {
			p.Skipped = append(p.Skipped, strings.TrimSpace(r))
			continue
		}
		as := byMAC[mac]
		if len(as) == 0 {
			p.add("addr", "same", mac+" — "+pname, fmt.Errorf("no address recorded for this MAC; paste the ARP table first"))
			continue
		}
		pt, added, err := s.port(me.ID, pname, true, p)
		if err != nil {
			p.add("port", "add", me.Name+" "+pname, err)
			continue
		}
		if added {
			p.add("port", "add", me.Name+" "+pname, nil)
		}
		for _, a := range as {
			if a.DeviceID == me.ID && a.PortID == pt.ID {
				p.add("addr", "same", a.IP+" — "+me.Name+" "+pname, nil)
				continue
			}
			a.DeviceID, a.PortID = me.ID, pt.ID
			if p.Applied {
				if _, err := s.SaveAddr(a); err != nil {
					p.add("addr", "update", a.IP+" — "+me.Name+" "+pname, err)
					continue
				}
			}
			p.add("addr", "update", a.IP+" — "+me.Name+" "+pname, nil)
		}
	}
	return nil
}
