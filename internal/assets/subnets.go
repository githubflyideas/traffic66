package assets

import (
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
)

// The networks written down against a device. traffic66 does not assign
// addresses and does not look anything up here: a network recorded on a
// switch is a note about what that switch carries, kept beside its ports,
// and nothing on any chart or page is decided from it. The address
// assignment this ledger used to do, and the page that went with it, were
// removed: they were a second place to keep what belongs in whatever
// system already hands out addresses.

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
	if err := s.exists("devices", v.DeviceID, "device"); err != nil {
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
		if _, err := s.DB.Exec(`INSERT INTO subnets (id, prefix, name, vlan, site_id, gateway, tags, note, hidden, created, updated, device_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.Prefix, v.Name, v.VLAN, int64(0), v.Gateway, tags, v.Note, v.Hidden, now(), now(), v.DeviceID); err != nil {
			return v, err
		}
	} else {
		r, err := s.DB.Exec(`UPDATE subnets SET prefix = ?, name = ?, vlan = ?, device_id = ?, gateway = ?,
			tags = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
			v.Prefix, v.Name, v.VLAN, v.DeviceID, v.Gateway, tags, v.Note, v.Hidden, now(), v.ID)
		if err := affected(r, err); err != nil {
			return v, err
		}
	}
	return v, nil
}

// Subnets lists networks, narrowest first so that a more specific network
// is offered before the one that contains it.
func (s *Store) Subnets() ([]Subnet, error) {
	rows, err := s.DB.Query(`SELECT id, prefix, name, vlan, device_id, gateway, tags, note, hidden FROM subnets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Subnet{}
	for rows.Next() {
		var v Subnet
		var tags string
		if err := rows.Scan(&v.ID, &v.Prefix, &v.Name, &v.VLAN, &v.DeviceID, &v.Gateway, &tags, &v.Note, &v.Hidden); err != nil {
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

// DeleteSubnet removes a network from the ledger.
func (s *Store) DeleteSubnet(id int64) error {
	return affected(s.DB.Exec(`DELETE FROM subnets WHERE id = ?`, id))
}
