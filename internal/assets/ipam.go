package assets

import (
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// Address kinds. The kind is what the address is for, not what was seen:
// nothing here is set by watching traffic.
const (
	KindStatic   = "static"
	KindDHCP     = "dhcp"     // inside a pool handed out by a DHCP server
	KindReserved = "reserved" // kept free on purpose
	KindGateway  = "gateway"
	KindVIP      = "vip"
)

// MaxSubnetHosts is how many addresses a subnet page will enumerate. A
// /16 has 65,534 of them; listing every one is neither useful nor fast, so
// larger subnets are summarised and searched instead.
const MaxSubnetHosts = 4096

// SaveSubnet inserts or updates a network.
func (s *Store) SaveSubnet(v Subnet) (Subnet, error) {
	v.Name, v.Note = trim(v.Name, 120), trim(v.Note, 2000)
	p, err := canonPrefix(v.Prefix)
	if err != nil {
		return v, err
	}
	v.Prefix = p.String()
	if v.VLAN < 0 || v.VLAN > 4094 {
		return v, fmt.Errorf("%d is not a VLAN id (1 to 4094)", v.VLAN)
	}
	if v.Gateway != "" {
		g, err := canonIP(v.Gateway)
		if err != nil {
			return v, err
		}
		if a, _ := netip.ParseAddr(g); !p.Contains(a) {
			return v, fmt.Errorf("the gateway %s is not inside %s", g, v.Prefix)
		}
		v.Gateway = g
	}
	if err := s.exists("sites", v.SiteID, "site"); err != nil {
		return v, err
	}
	var other int64
	err = s.DB.QueryRow(`SELECT id FROM subnets WHERE prefix = ? AND id <> ?`, v.Prefix, v.ID).Scan(&other)
	if err == nil {
		return v, fmt.Errorf("%s is already recorded", v.Prefix)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return v, err
	}
	tags := tagString(v.Tags)
	v.Tags = tagList(tags)
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		if _, err := s.DB.Exec(`INSERT INTO subnets VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.Prefix, v.Name, v.VLAN, v.SiteID, v.Gateway, tags, v.Note, v.Hidden, now(), now()); err != nil {
			return v, err
		}
	} else {
		r, err := s.DB.Exec(`UPDATE subnets SET prefix = ?, name = ?, vlan = ?, site_id = ?, gateway = ?,
			tags = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
			v.Prefix, v.Name, v.VLAN, v.SiteID, v.Gateway, tags, v.Note, v.Hidden, now(), v.ID)
		if err := affected(r, err); err != nil {
			return v, err
		}
	}
	// addresses find their subnet by containment, so re-home them whenever
	// the networks change; an address inside no recorded network keeps 0
	return v, s.rehome()
}

// Subnets lists networks, narrowest first so that a more specific network
// is offered before the one that contains it.
func (s *Store) Subnets() ([]Subnet, error) {
	rows, err := s.DB.Query(`SELECT id, prefix, name, vlan, site_id, gateway, tags, note, hidden FROM subnets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Subnet{}
	for rows.Next() {
		var v Subnet
		var tags string
		if err := rows.Scan(&v.ID, &v.Prefix, &v.Name, &v.VLAN, &v.SiteID, &v.Gateway, &tags, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		v.Tags = tagList(tags)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortSubnets(out)
	return out, nil
}

// DeleteSubnet removes a network. Its addresses are kept: they were
// assigned to machines, not to the record of the network, and losing them
// because someone re-drew the network is how a ledger stops being trusted.
func (s *Store) DeleteSubnet(id int64) error {
	r, err := s.DB.Exec(`DELETE FROM subnets WHERE id = ?`, id)
	if err := affected(r, err); err != nil {
		return err
	}
	return s.rehome()
}

// SaveAddr assigns an address, or edits an assignment.
func (s *Store) SaveAddr(v Addr) (Addr, error) {
	v.Name, v.Note = trim(v.Name, 120), trim(v.Note, 2000)
	ip, err := canonIP(v.IP)
	if err != nil {
		return v, err
	}
	v.IP = ip
	if v.MAC != "" {
		m := CanonMAC(v.MAC)
		if m == "" {
			return v, fmt.Errorf("%q is not a MAC address", v.MAC)
		}
		v.MAC = m
	}
	switch v.Kind {
	case "", KindStatic, KindDHCP, KindReserved, KindGateway, KindVIP:
	default:
		return v, fmt.Errorf("%q is not an address kind", v.Kind)
	}
	if v.Kind == "" {
		v.Kind = KindStatic
	}
	if err := s.exists("devices", v.DeviceID, "device"); err != nil {
		return v, err
	}
	if err := s.exists("ports", v.PortID, "port"); err != nil {
		return v, err
	}
	var other int64
	var name string
	err = s.DB.QueryRow(`SELECT id, name FROM addrs WHERE ip = ? AND id <> ?`, v.IP, v.ID).Scan(&other, &name)
	if err == nil {
		if name == "" {
			name = "another record"
		}
		return v, fmt.Errorf("%s is already assigned to %s", v.IP, name)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return v, err
	}
	v.SubnetID = s.subnetOf(v.IP)
	tags := tagString(v.Tags)
	v.Tags = tagList(tags)
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		_, err = s.DB.Exec(`INSERT INTO addrs VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.SubnetID, v.IP, v.Name, v.Kind, v.DeviceID, v.PortID, v.MAC, tags, v.Note, v.Hidden, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE addrs SET subnet_id = ?, ip = ?, name = ?, kind = ?, device_id = ?,
		port_id = ?, mac = ?, tags = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
		v.SubnetID, v.IP, v.Name, v.Kind, v.DeviceID, v.PortID, v.MAC, tags, v.Note, v.Hidden, now(), v.ID)
	return v, affected(r, err)
}

// AddrQuery selects assignments.
type AddrQuery struct {
	SubnetID int64
	DeviceID int64
	Tag      string
	Search   string
	Limit    int
}

// Addrs lists assignments in address order.
func (s *Store) Addrs(q AddrQuery) ([]Addr, error) {
	conds, args := []string{"true"}, []any{}
	if q.SubnetID != 0 {
		conds, args = append(conds, "subnet_id = ?"), append(args, q.SubnetID)
	}
	if q.DeviceID != 0 {
		conds, args = append(conds, "device_id = ?"), append(args, q.DeviceID)
	}
	if t := strings.ToLower(strings.TrimSpace(q.Tag)); t != "" {
		conds, args = append(conds, "list_contains(str_split(tags, ','), ?)"), append(args, t)
	}
	if t := strings.TrimSpace(q.Search); t != "" {
		conds = append(conds, `(ip ILIKE ? OR name ILIKE ? OR mac ILIKE ? OR note ILIKE ?)`)
		like := "%" + t + "%"
		args = append(args, like, like, like, like)
	}
	if q.Limit <= 0 || q.Limit > 20000 {
		q.Limit = 5000
	}
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT id, subnet_id, ip, name, kind, device_id, port_id, mac, tags, note, hidden
		FROM addrs WHERE %s ORDER BY ip LIMIT %d`, strings.Join(conds, " AND "), q.Limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Addr{}
	for rows.Next() {
		var v Addr
		var tags string
		if err := rows.Scan(&v.ID, &v.SubnetID, &v.IP, &v.Name, &v.Kind, &v.DeviceID, &v.PortID, &v.MAC, &tags, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		v.Tags = tagList(tags)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortAddrs(out)
	return out, nil
}

// DeleteAddr frees an assignment.
func (s *Store) DeleteAddr(id int64) error {
	r, err := s.DB.Exec(`DELETE FROM addrs WHERE id = ?`, id)
	return affected(r, err)
}

// subnetOf returns the id of the narrowest recorded network containing an
// address, or 0.
func (s *Store) subnetOf(ip string) int64 {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return 0
	}
	subs, err := s.Subnets()
	if err != nil {
		return 0
	}
	for _, sn := range subs { // narrowest first
		if p, err := netip.ParsePrefix(sn.Prefix); err == nil && p.Contains(a) {
			return sn.ID
		}
	}
	return 0
}

// rehome re-reads which network each address falls in. It runs after the
// networks change, which is rare and cheap next to being wrong.
func (s *Store) rehome() error {
	subs, err := s.Subnets()
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(`SELECT id, ip, subnet_id FROM addrs`)
	if err != nil {
		return err
	}
	type move struct {
		id, to int64
	}
	var moves []move
	for rows.Next() {
		var id, cur int64
		var ip string
		if err := rows.Scan(&id, &ip, &cur); err != nil {
			rows.Close()
			return err
		}
		to := int64(0)
		if a, err := netip.ParseAddr(ip); err == nil {
			for _, sn := range subs {
				if p, err := netip.ParsePrefix(sn.Prefix); err == nil && p.Contains(a) {
					to = sn.ID
					break
				}
			}
		}
		if to != cur {
			moves = append(moves, move{id, to})
		}
	}
	rows.Close()
	for _, m := range moves {
		if _, err := s.DB.Exec(`UPDATE addrs SET subnet_id = ? WHERE id = ?`, m.to, m.id); err != nil {
			return err
		}
	}
	return nil
}

// SubnetUse is how full a network is.
type SubnetUse struct {
	Subnet
	Hosts    int64 `json:"hosts"`    // usable addresses, -1 when too large to count
	Assigned int   `json:"assigned"` // addresses written down in it
	Free     int64 `json:"free"`     // -1 when hosts is -1
	Listable bool  `json:"listable"` // small enough to show address by address
}

// Use reports how full each network is.
func (s *Store) Use() ([]SubnetUse, error) {
	subs, err := s.Subnets()
	if err != nil {
		return nil, err
	}
	counts := map[int64]int{}
	rows, err := s.DB.Query(`SELECT subnet_id, count(*) FROM addrs GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			rows.Close()
			return nil, err
		}
		counts[id] = n
	}
	rows.Close()
	out := make([]SubnetUse, 0, len(subs))
	for _, sn := range subs {
		u := SubnetUse{Subnet: sn, Assigned: counts[sn.ID], Hosts: -1, Free: -1}
		if p, err := netip.ParsePrefix(sn.Prefix); err == nil {
			if n, ok := hostCount(p); ok {
				u.Hosts, u.Free = n, max64(0, n-int64(u.Assigned))
				u.Listable = n <= MaxSubnetHosts
			}
		}
		out = append(out, u)
	}
	return out, nil
}

// hostCount is how many addresses a network can assign. A /31 and a /32
// are whole-prefix cases (point to point, single host); otherwise the
// network and broadcast addresses are not assignable. IPv6 networks are
// counted only when small enough to mean anything.
func hostCount(p netip.Prefix) (int64, bool) {
	bits := p.Addr().BitLen() - p.Bits()
	if bits > 20 {
		return 0, false
	}
	n := int64(1) << bits
	if p.Addr().Is4() && n > 2 {
		n -= 2
	}
	return n, true
}

// FreeAddrs returns the first n addresses of a network that nothing is
// assigned to, skipping the network and broadcast addresses of an IPv4
// network and the gateway.
func (s *Store) FreeAddrs(subnetID int64, n int) ([]string, error) {
	if n <= 0 || n > 256 {
		n = 16
	}
	var prefix, gw string
	if err := s.DB.QueryRow(`SELECT prefix, gateway FROM subnets WHERE id = ?`, subnetID).Scan(&prefix, &gw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	rows, err := s.DB.Query(`SELECT ip FROM addrs WHERE subnet_id = ?`, subnetID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			rows.Close()
			return nil, err
		}
		taken[ip] = true
	}
	rows.Close()
	if gw != "" {
		taken[gw] = true
	}
	out := []string{}
	total, _ := hostCount(p)
	a := p.Addr()
	if p.Addr().Is4() && total > 2 {
		a = a.Next() // the network address is not assignable
	}
	for i := int64(0); i < total && len(out) < n; i++ {
		if !p.Contains(a) {
			break
		}
		if !taken[a.String()] {
			out = append(out, a.String())
		}
		a = a.Next()
	}
	return out, nil
}

// Conflict is an address the ledger and the network disagree about, or
// that the ledger disagrees with itself about.
type Conflict struct {
	IP     string `json:"ip"`
	Kind   string `json:"kind"` // outside, duplicate_mac, unrecorded
	Detail string `json:"detail"`
}

// Conflicts reports what does not add up in the assignments themselves:
// addresses in no recorded network, and one MAC written against several
// addresses. What the network is actually doing is a separate report; this
// one only reads the ledger, so it is the same answer offline.
func (s *Store) Conflicts() ([]Conflict, error) {
	out := []Conflict{}
	rows, err := s.DB.Query(`SELECT ip, name FROM addrs WHERE subnet_id = 0 ORDER BY ip LIMIT 500`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ip, name string
		if err := rows.Scan(&ip, &name); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, Conflict{IP: ip, Kind: "outside", Detail: name})
	}
	rows.Close()
	rows, err = s.DB.Query(`SELECT mac, string_agg(ip, ', ' ORDER BY ip) FROM addrs
		WHERE mac <> '' GROUP BY mac HAVING count(*) > 1 LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var mac, ips string
		if err := rows.Scan(&mac, &ips); err != nil {
			return nil, err
		}
		out = append(out, Conflict{IP: mac, Kind: "duplicate_mac", Detail: ips})
	}
	return out, rows.Err()
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
