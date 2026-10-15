package assets

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// The ledger in and out as CSV, because an asset list that cannot leave
// the tool is a hostage, and because most ledgers start life in a
// spreadsheet somebody already keeps.
//
// Export and import are the same columns in the same order, so a file
// exported, edited and imported again updates what it came from rather
// than duplicating it: the id column is what ties a row to its record,
// and a row with an empty id is a new one.

// csvColumns are the columns of each kind, in order. The first is always
// the id.
var csvColumns = map[string][]string{
	"site":    {"id", "name", "address", "note", "hidden"},
	"rack":    {"id", "name", "site", "units", "note", "hidden"},
	"device":  {"id", "name", "role", "vendor", "model", "serial", "mgmt_ip", "rack", "pos_u", "height_u", "tags", "note", "hidden"},
	"port":    {"id", "device", "name", "ifindex", "speed", "kind", "note"},
	"circuit": {"id", "name", "carrier", "kind", "bandwidth", "site", "port", "contract", "starts", "ends", "tags", "note", "hidden"},
	"subnet":  {"id", "prefix", "name", "vlan", "site", "gateway", "tags", "note", "hidden"},
	"addr":    {"id", "ip", "name", "kind", "device", "port", "mac", "tags", "note", "hidden"},
}

// CSVKinds is every kind that can be exported, for the API and the pages.
var CSVKinds = []string{"site", "rack", "device", "port", "circuit", "subnet", "addr"}

// names resolves the references a CSV carries. A site, rack, device or
// port is written out by name rather than by number, because a number
// means nothing in a spreadsheet and a name is what the operator already
// knows; on the way back in, a name that is not there yet is not invented.
type names struct {
	site, rack, device map[int64]string
	port               map[int64]string
}

func (s *Store) names() (names, error) {
	n := names{site: map[int64]string{}, rack: map[int64]string{}, device: map[int64]string{}, port: map[int64]string{}}
	sites, err := s.Sites()
	if err != nil {
		return n, err
	}
	for _, v := range sites {
		n.site[v.ID] = v.Name
	}
	racks, err := s.Racks(0)
	if err != nil {
		return n, err
	}
	for _, v := range racks {
		n.rack[v.ID] = v.Name
	}
	devs, err := s.Devices(DeviceQuery{})
	if err != nil {
		return n, err
	}
	for _, v := range devs {
		n.device[v.ID] = v.Name
	}
	ports, err := s.Ports(0)
	if err != nil {
		return n, err
	}
	for _, v := range ports {
		n.port[v.ID] = n.device[v.DeviceID] + " " + v.Name
	}
	return n, nil
}

func b(v bool) string {
	if v {
		return "yes"
	}
	return ""
}

func num(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func id(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

// ExportCSV writes one kind of record.
func (s *Store) ExportCSV(w io.Writer, kind string) error {
	cols, ok := csvColumns[kind]
	if !ok {
		return fmt.Errorf("%q is not a kind of asset", kind)
	}
	n, err := s.names()
	if err != nil {
		return err
	}
	c := csv.NewWriter(w)
	if err := c.Write(cols); err != nil {
		return err
	}
	var rows [][]string
	switch kind {
	case "site":
		vs, err := s.Sites()
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.Name, v.Address, v.Note, b(v.Hidden)})
		}
	case "rack":
		vs, err := s.Racks(0)
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.Name, n.site[v.SiteID], num(v.Units), v.Note, b(v.Hidden)})
		}
	case "device":
		vs, err := s.Devices(DeviceQuery{})
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.Name, v.Role, v.Vendor, v.Model, v.Serial, v.MgmtIP,
				n.rack[v.RackID], num(v.PosU), num(v.HeightU), strings.Join(v.Tags, " "), v.Note, b(v.Hidden)})
		}
	case "port":
		vs, err := s.Ports(0)
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), n.device[v.DeviceID], v.Name, num(int(v.IfIndex)),
				strconv.FormatUint(v.Speed, 10), v.Kind, v.Note})
		}
	case "circuit":
		vs, err := s.Circuits()
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.Name, v.Carrier, v.Kind, v.Bandwidth, n.site[v.SiteID],
				n.port[v.PortID], v.Contract, v.Starts, v.Ends, strings.Join(v.Tags, " "), v.Note, b(v.Hidden)})
		}
	case "subnet":
		vs, err := s.Subnets()
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.Prefix, v.Name, num(v.VLAN), n.site[v.SiteID],
				v.Gateway, strings.Join(v.Tags, " "), v.Note, b(v.Hidden)})
		}
	case "addr":
		vs, err := s.Addrs(AddrQuery{Limit: 20000})
		if err != nil {
			return err
		}
		for _, v := range vs {
			rows = append(rows, []string{id(v.ID), v.IP, v.Name, v.Kind, n.device[v.DeviceID], n.port[v.PortID],
				v.MAC, strings.Join(v.Tags, " "), v.Note, b(v.Hidden)})
		}
	}
	for _, r := range rows {
		if err := c.Write(r); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

// lookup finds a record by name for an imported reference.
func lookup(m map[int64]string, name string) int64 {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	for k, v := range m {
		if strings.EqualFold(v, name) {
			return k
		}
	}
	return -1 // named, but no such record
}

func yes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "y", "true", "1":
		return true
	}
	return false
}

func intOf(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// bom is the byte order mark a spreadsheet puts at the start of a file it
// saved as UTF-8; it would otherwise become part of the first column name.
const bom = "\uFEFF"

// ImportCSV reads one kind of record. A row carrying an id updates that
// record; a row without one adds. Rows are read in order and a row that
// cannot be read stops nothing: it is reported with its line number, and
// the rest of the file still lands, because a spreadsheet with one bad
// line in it is the normal case, not an exception.
func (s *Store) ImportCSV(r io.Reader, kind string, apply bool) (Plan, error) {
	cols, ok := csvColumns[kind]
	if !ok {
		return Plan{}, fmt.Errorf("%q is not a kind of asset", kind)
	}
	n, err := s.names()
	if err != nil {
		return Plan{}, err
	}
	rd := csv.NewReader(r)
	rd.FieldsPerRecord = -1
	rd.TrimLeadingSpace = true
	head, err := rd.Read()
	if err != nil {
		return Plan{}, fmt.Errorf("the file has no header row")
	}
	at := map[string]int{}
	for i, h := range head {
		at[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, bom)))] = i
	}
	for _, want := range cols[1:2] { // the first real column is required
		if _, ok := at[want]; !ok {
			return Plan{}, fmt.Errorf("the file has no %q column; export this kind first to see the columns", want)
		}
	}
	p := Plan{Format: "csv:" + kind, Applied: apply}
	line := 1
	for {
		rec, err := rd.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			p.Skipped = append(p.Skipped, fmt.Sprintf("line %d: %v", line, err))
			continue
		}
		get := func(name string) string {
			i, ok := at[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if strings.TrimSpace(strings.Join(rec, "")) == "" {
			continue
		}
		rid, _ := strconv.ParseInt(get("id"), 10, 64)
		act := "add"
		if rid != 0 {
			act = "update"
		}
		ref := func(m map[int64]string, col string) (int64, error) {
			v := lookup(m, get(col))
			if v < 0 {
				return 0, fmt.Errorf("line %d: no %s called %q", line, col, get(col))
			}
			return v, nil
		}
		var what string
		var werr error
		switch kind {
		case "site":
			v := Site{ID: rid, Name: get("name"), Address: get("address"), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.Name
			if apply {
				_, werr = s.SaveSite(v)
			}
		case "rack":
			site, err := ref(n.site, "site")
			if err != nil {
				p.add("rack", act, get("name"), err)
				continue
			}
			v := Rack{ID: rid, Name: get("name"), SiteID: site, Units: intOf(get("units")), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.Name
			if apply {
				_, werr = s.SaveRack(v)
			}
		case "device":
			rack, err := ref(n.rack, "rack")
			if err != nil {
				p.add("device", act, get("name"), err)
				continue
			}
			v := Device{ID: rid, Name: get("name"), Role: get("role"), Vendor: get("vendor"), Model: get("model"),
				Serial: get("serial"), MgmtIP: get("mgmt_ip"), RackID: rack, PosU: intOf(get("pos_u")),
				HeightU: intOf(get("height_u")), Tags: strings.Fields(get("tags")), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.Name
			if apply {
				_, werr = s.SaveDevice(v)
			}
		case "port":
			dev, err := ref(n.device, "device")
			if err != nil {
				p.add("port", act, get("name"), err)
				continue
			}
			sp, _ := strconv.ParseUint(get("speed"), 10, 64)
			v := Port{ID: rid, DeviceID: dev, Name: get("name"), IfIndex: uint32(intOf(get("ifindex"))),
				Speed: sp, Kind: get("kind"), Note: get("note")}
			what = get("device") + " " + v.Name
			if apply {
				_, werr = s.SavePort(v)
			}
		case "circuit":
			site, err := ref(n.site, "site")
			if err != nil {
				p.add("circuit", act, get("name"), err)
				continue
			}
			port, err := ref(n.port, "port")
			if err != nil {
				p.add("circuit", act, get("name"), err)
				continue
			}
			v := Circuit{ID: rid, Name: get("name"), Carrier: get("carrier"), Kind: get("kind"), Bandwidth: get("bandwidth"),
				SiteID: site, PortID: port, Contract: get("contract"), Starts: get("starts"), Ends: get("ends"),
				Tags: strings.Fields(get("tags")), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.Name
			if apply {
				_, werr = s.SaveCircuit(v)
			}
		case "subnet":
			site, err := ref(n.site, "site")
			if err != nil {
				p.add("subnet", act, get("prefix"), err)
				continue
			}
			v := Subnet{ID: rid, Prefix: get("prefix"), Name: get("name"), VLAN: intOf(get("vlan")), SiteID: site,
				Gateway: get("gateway"), Tags: strings.Fields(get("tags")), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.Prefix
			if apply {
				_, werr = s.SaveSubnet(v)
			}
		case "addr":
			dev, err := ref(n.device, "device")
			if err != nil {
				p.add("addr", act, get("ip"), err)
				continue
			}
			port, err := ref(n.port, "port")
			if err != nil {
				p.add("addr", act, get("ip"), err)
				continue
			}
			v := Addr{ID: rid, IP: get("ip"), Name: get("name"), Kind: get("kind"), DeviceID: dev, PortID: port,
				MAC: get("mac"), Tags: strings.Fields(get("tags")), Note: get("note"), Hidden: yes(get("hidden"))}
			what = v.IP
			if apply {
				_, werr = s.SaveAddr(v)
			}
		}
		if werr != nil {
			werr = fmt.Errorf("line %d: %v", line, werr)
		}
		p.add(kind, act, what, werr)
	}
	return p, nil
}
