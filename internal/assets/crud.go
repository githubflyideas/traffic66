package assets

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// MaxUnits is the tallest rack the ledger will record. Racks taller than
// this are a typo, and the rack diagram has to draw whatever is stored.
const MaxUnits = 60

// DefaultUnits is the height a rack gets when none is given. Most
// cabinets in a room are the same height and 46U is the common one; any
// other height is a number away.
const DefaultUnits = 46

// ErrNotFound is returned when an identifier names nothing.
var ErrNotFound = errors.New("no such record")

// InUse says a record cannot be deleted because others point at it, and
// names them, so the message can say what to do first.
type InUse struct{ What string }

func (e InUse) Error() string { return "still in use by " + e.What }

// ---------------------------------------------------------------- sites

// SaveSite inserts a site (ID 0) or updates one.
func (s *Store) SaveSite(v Site) (Site, error) {
	v.Name = trim(v.Name, 120)
	v.Address, v.Note = trim(v.Address, 300), trim(v.Note, 2000)
	if v.Name == "" {
		return v, errors.New("the site needs a name")
	}
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		_, err = s.DB.Exec(`INSERT INTO sites VALUES (?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.Name, v.Address, v.Note, v.Hidden, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE sites SET name = ?, address = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
		v.Name, v.Address, v.Note, v.Hidden, now(), v.ID)
	return v, affected(r, err)
}

// Sites lists sites by name.
func (s *Store) Sites() ([]Site, error) {
	rows, err := s.DB.Query(`SELECT id, name, address, note, hidden FROM sites ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Site{}
	for rows.Next() {
		var v Site
		if err := rows.Scan(&v.ID, &v.Name, &v.Address, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteSite removes a site that nothing points at.
func (s *Store) DeleteSite(id int64) error {
	for _, p := range []struct{ table, what string }{
		{"racks", "racks"}, {"subnets", "subnets"}, {"circuits", "circuits"},
	} {
		var n int
		s.DB.QueryRow(`SELECT count(*) FROM `+p.table+` WHERE site_id = ?`, id).Scan(&n)
		if n > 0 {
			return InUse{fmt.Sprintf("%d %s", n, p.what)}
		}
	}
	r, err := s.DB.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return affected(r, err)
}

// ---------------------------------------------------------------- racks

// SaveRack inserts or updates a rack.
func (s *Store) SaveRack(v Rack) (Rack, error) {
	v.Name, v.Note = trim(v.Name, 120), trim(v.Note, 2000)
	if v.Name == "" {
		return v, errors.New("the rack needs a name")
	}
	if v.Units <= 0 {
		v.Units = DefaultUnits
	}
	if v.Units > MaxUnits {
		return v, fmt.Errorf("a rack of %d units is more than the %d this records", v.Units, MaxUnits)
	}
	if err := s.exists("sites", v.SiteID, "site"); err != nil {
		return v, err
	}
	// a rack cannot shrink below the devices already placed in it
	if v.ID != 0 {
		var top sql.NullInt64
		s.DB.QueryRow(`SELECT max(pos_u + greatest(height_u, 1) - 1) FROM devices WHERE rack_id = ? AND pos_u > 0`, v.ID).Scan(&top)
		if top.Valid && int(top.Int64) > v.Units {
			return v, fmt.Errorf("a device reaches unit %d; empty it before making the rack %d units", top.Int64, v.Units)
		}
	}
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		_, err = s.DB.Exec(`INSERT INTO racks VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.SiteID, v.Name, v.Units, v.Note, v.Hidden, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE racks SET site_id = ?, name = ?, units = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
		v.SiteID, v.Name, v.Units, v.Note, v.Hidden, now(), v.ID)
	return v, affected(r, err)
}

// Racks lists racks, of one site when siteID is not 0.
func (s *Store) Racks(siteID int64) ([]Rack, error) {
	q := `SELECT id, site_id, name, units, note, hidden FROM racks`
	var args []any
	if siteID != 0 {
		q += ` WHERE site_id = ?`
		args = append(args, siteID)
	}
	rows, err := s.DB.Query(q+` ORDER BY name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rack{}
	for rows.Next() {
		var v Rack
		if err := rows.Scan(&v.ID, &v.SiteID, &v.Name, &v.Units, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteRack removes a rack. The devices in it are kept and unracked:
// equipment outlives the cabinet it stood in, and deleting a rack must
// never quietly delete the record of what was in it.
func (s *Store) DeleteRack(id int64) error {
	if _, err := s.DB.Exec(`UPDATE devices SET rack_id = 0, pos_u = 0, updated = ? WHERE rack_id = ?`, now(), id); err != nil {
		return err
	}
	r, err := s.DB.Exec(`DELETE FROM racks WHERE id = ?`, id)
	return affected(r, err)
}

// -------------------------------------------------------------- devices

// SaveDevice inserts or updates a device, checking that it fits where it
// is placed and that nothing else is already there.
func (s *Store) SaveDevice(v Device) (Device, error) {
	v.Name = trim(v.Name, 120)
	v.Role, v.Vendor, v.Model = trim(v.Role, 60), trim(v.Vendor, 60), trim(v.Model, 60)
	v.Serial, v.Note = trim(v.Serial, 80), trim(v.Note, 2000)
	if v.Name == "" {
		return v, errors.New("the device needs a name")
	}
	if v.MgmtIP != "" {
		ip, err := canonIP(v.MgmtIP)
		if err != nil {
			return v, err
		}
		v.MgmtIP = ip
	}
	if v.HeightU <= 0 {
		v.HeightU = 1
	}
	if err := s.exists("racks", v.RackID, "rack"); err != nil {
		return v, err
	}
	if v.RackID == 0 {
		v.PosU = 0 // a position means nothing without a rack
	}
	if err := s.fits(v); err != nil {
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
		_, err = s.DB.Exec(`INSERT INTO devices VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.RackID, v.Name, v.Role, v.Vendor, v.Model, v.Serial, v.MgmtIP,
			v.PosU, v.HeightU, tags, v.Note, v.Hidden, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE devices SET rack_id = ?, name = ?, role = ?, vendor = ?, model = ?,
		serial = ?, mgmt_ip = ?, pos_u = ?, height_u = ?, tags = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
		v.RackID, v.Name, v.Role, v.Vendor, v.Model, v.Serial, v.MgmtIP,
		v.PosU, v.HeightU, tags, v.Note, v.Hidden, now(), v.ID)
	return v, affected(r, err)
}

// fits checks a device's place in its rack: inside the rack, and not on
// top of another device.
func (s *Store) fits(v Device) error {
	if v.RackID == 0 || v.PosU == 0 {
		return nil
	}
	var units int
	if err := s.DB.QueryRow(`SELECT units FROM racks WHERE id = ?`, v.RackID).Scan(&units); err != nil {
		return ErrNotFound
	}
	if v.PosU < 1 || v.PosU+v.HeightU-1 > units {
		return fmt.Errorf("unit %d to %d is outside a rack of %d units", v.PosU, v.PosU+v.HeightU-1, units)
	}
	var name string
	err := s.DB.QueryRow(`SELECT name FROM devices WHERE rack_id = ? AND id <> ? AND pos_u > 0
		AND pos_u <= ? AND pos_u + greatest(height_u, 1) - 1 >= ? LIMIT 1`,
		v.RackID, v.ID, v.PosU+v.HeightU-1, v.PosU).Scan(&name)
	if err == nil {
		return fmt.Errorf("%s is already in units %d to %d", name, v.PosU, v.PosU+v.HeightU-1)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

// DeviceQuery selects devices.
type DeviceQuery struct {
	RackID int64
	SiteID int64
	Tag    string
	Search string
}

// Devices lists devices, newest placement first by rack then unit.
func (s *Store) Devices(q DeviceQuery) ([]Device, error) {
	conds, args := []string{"true"}, []any{}
	if q.RackID != 0 {
		conds, args = append(conds, "d.rack_id = ?"), append(args, q.RackID)
	}
	if q.SiteID != 0 {
		conds, args = append(conds, "d.rack_id IN (SELECT id FROM racks WHERE site_id = ?)"), append(args, q.SiteID)
	}
	if t := strings.ToLower(strings.TrimSpace(q.Tag)); t != "" {
		conds, args = append(conds, "list_contains(str_split(d.tags, ','), ?)"), append(args, t)
	}
	if t := strings.TrimSpace(q.Search); t != "" {
		conds = append(conds, `(d.name ILIKE ? OR d.model ILIKE ? OR d.serial ILIKE ? OR d.mgmt_ip ILIKE ?)`)
		like := "%" + t + "%"
		args = append(args, like, like, like, like)
	}
	rows, err := s.DB.Query(`SELECT d.id, d.rack_id, d.name, d.role, d.vendor, d.model, d.serial,
		d.mgmt_ip, d.pos_u, d.height_u, d.tags, d.note, d.hidden FROM devices d
		WHERE `+strings.Join(conds, " AND ")+` ORDER BY d.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Device{}
	for rows.Next() {
		var v Device
		var tags string
		if err := rows.Scan(&v.ID, &v.RackID, &v.Name, &v.Role, &v.Vendor, &v.Model, &v.Serial,
			&v.MgmtIP, &v.PosU, &v.HeightU, &tags, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		v.Tags = tagList(tags)
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteDevice removes a device with its ports and their links. Addresses
// assigned to it are kept and unassigned: an address stays allocated when
// the box behind it is replaced, which is exactly when it must not be
// quietly freed for someone else.
func (s *Store) DeleteDevice(id int64) error {
	if _, err := s.DB.Exec(`DELETE FROM links WHERE a_port IN (SELECT id FROM ports WHERE device_id = ?)
		OR b_port IN (SELECT id FROM ports WHERE device_id = ?)`, id, id); err != nil {
		return err
	}
	// The networks written against a device are a note about that device,
	// so they go when it does. (Cables and ports go too; a circuit stays,
	// because it was bought from a carrier and outlives the box it lands
	// on, and only loses the port it landed on.)
	if _, err := s.DB.Exec(`DELETE FROM subnets WHERE device_id = ?`, id); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`UPDATE circuits SET port_id = 0, updated = ? WHERE port_id IN (SELECT id FROM ports WHERE device_id = ?)`, now(), id); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`DELETE FROM ports WHERE device_id = ?`, id); err != nil {
		return err
	}
	r, err := s.DB.Exec(`DELETE FROM devices WHERE id = ?`, id)
	return affected(r, err)
}

// ---------------------------------------------------------------- ports

// SavePort inserts or updates a port of a device.
func (s *Store) SavePort(v Port) (Port, error) {
	v.Name, v.Kind, v.Note = trim(v.Name, 80), trim(v.Kind, 40), trim(v.Note, 500)
	if v.Name == "" {
		return v, errors.New("the port needs a name")
	}
	if v.DeviceID == 0 {
		return v, errors.New("the port needs a device")
	}
	if err := s.exists("devices", v.DeviceID, "device"); err != nil {
		return v, err
	}
	var other int64
	err := s.DB.QueryRow(`SELECT id FROM ports WHERE device_id = ? AND lower(name) = lower(?) AND id <> ?`,
		v.DeviceID, v.Name, v.ID).Scan(&other)
	if err == nil {
		return v, fmt.Errorf("the device already has a port called %s", v.Name)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return v, err
	}
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		_, err = s.DB.Exec(`INSERT INTO ports VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.DeviceID, v.Name, v.IfIndex, v.Speed, v.Kind, v.Note, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE ports SET device_id = ?, name = ?, ifindex = ?, speed = ?, kind = ?, note = ?, updated = ? WHERE id = ?`,
		v.DeviceID, v.Name, v.IfIndex, v.Speed, v.Kind, v.Note, now(), v.ID)
	return v, affected(r, err)
}

// Ports lists a device's ports, or all of them when deviceID is 0.
func (s *Store) Ports(deviceID int64) ([]Port, error) {
	q := `SELECT id, device_id, name, ifindex, speed, kind, note FROM ports`
	var args []any
	if deviceID != 0 {
		q += ` WHERE device_id = ?`
		args = append(args, deviceID)
	}
	rows, err := s.DB.Query(q+` ORDER BY device_id, ifindex, name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Port{}
	for rows.Next() {
		var v Port
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.Name, &v.IfIndex, &v.Speed, &v.Kind, &v.Note); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeletePort removes a port with its link, and lets go of what pointed at it.
func (s *Store) DeletePort(id int64) error {
	if _, err := s.DB.Exec(`DELETE FROM links WHERE a_port = ? OR b_port = ?`, id, id); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`UPDATE addrs SET port_id = 0, updated = ? WHERE port_id = ?`, now(), id); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`UPDATE circuits SET port_id = 0, updated = ? WHERE port_id = ?`, now(), id); err != nil {
		return err
	}
	r, err := s.DB.Exec(`DELETE FROM ports WHERE id = ?`, id)
	return affected(r, err)
}

// ---------------------------------------------------------------- links

// SaveLink records a cable between two ports. A port has one cable: the
// second one replaces the first, because that is what happened in the room.
func (s *Store) SaveLink(v Link) (Link, error) {
	v.Note = trim(v.Note, 500)
	if v.APort == 0 || v.BPort == 0 || v.APort == v.BPort {
		return v, errors.New("a link needs two different ports")
	}
	for _, p := range []int64{v.APort, v.BPort} {
		if err := s.exists("ports", p, "port"); err != nil {
			return v, err
		}
	}
	if _, err := s.DB.Exec(`DELETE FROM links WHERE id <> ? AND (a_port IN (?, ?) OR b_port IN (?, ?))`,
		v.ID, v.APort, v.BPort, v.APort, v.BPort); err != nil {
		return v, err
	}
	if v.ID == 0 {
		id, err := s.next()
		if err != nil {
			return v, err
		}
		v.ID = id
		_, err = s.DB.Exec(`INSERT INTO links VALUES (?, ?, ?, ?, ?, ?)`, v.ID, v.APort, v.BPort, v.Note, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE links SET a_port = ?, b_port = ?, note = ?, updated = ? WHERE id = ?`,
		v.APort, v.BPort, v.Note, now(), v.ID)
	return v, affected(r, err)
}

// Links lists every cable.
func (s *Store) Links() ([]Link, error) {
	rows, err := s.DB.Query(`SELECT id, a_port, b_port, note FROM links ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		var v Link
		if err := rows.Scan(&v.ID, &v.APort, &v.BPort, &v.Note); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteLink removes a cable.
func (s *Store) DeleteLink(id int64) error {
	r, err := s.DB.Exec(`DELETE FROM links WHERE id = ?`, id)
	return affected(r, err)
}

// ------------------------------------------------------------- circuits

// SaveCircuit inserts or updates a carrier line.
func (s *Store) SaveCircuit(v Circuit) (Circuit, error) {
	v.Name, v.Carrier = trim(v.Name, 120), trim(v.Carrier, 120)
	v.Kind, v.Bandwidth = trim(v.Kind, 60), trim(v.Bandwidth, 40)
	v.Contract, v.Note = trim(v.Contract, 120), trim(v.Note, 2000)
	if v.Name == "" {
		return v, errors.New("the circuit needs a name")
	}
	for _, d := range []*string{&v.Starts, &v.Ends} {
		if *d = trim(*d, 10); *d != "" && !isDay(*d) {
			return v, fmt.Errorf("%q is not a date (expected YYYY-MM-DD)", *d)
		}
	}
	if err := s.exists("sites", v.SiteID, "site"); err != nil {
		return v, err
	}
	if err := s.exists("ports", v.PortID, "port"); err != nil {
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
		_, err = s.DB.Exec(`INSERT INTO circuits VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, v.Name, v.Carrier, v.Kind, v.Bandwidth, v.SiteID, v.PortID,
			v.Contract, v.Starts, v.Ends, tags, v.Note, v.Hidden, now(), now())
		return v, err
	}
	r, err := s.DB.Exec(`UPDATE circuits SET name = ?, carrier = ?, kind = ?, bandwidth = ?, site_id = ?,
		port_id = ?, contract = ?, starts = ?, ends = ?, tags = ?, note = ?, hidden = ?, updated = ? WHERE id = ?`,
		v.Name, v.Carrier, v.Kind, v.Bandwidth, v.SiteID, v.PortID,
		v.Contract, v.Starts, v.Ends, tags, v.Note, v.Hidden, now(), v.ID)
	return v, affected(r, err)
}

// Circuits lists carrier lines by name.
func (s *Store) Circuits() ([]Circuit, error) {
	rows, err := s.DB.Query(`SELECT id, name, carrier, kind, bandwidth, site_id, port_id,
		contract, starts, ends, tags, note, hidden FROM circuits ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Circuit{}
	for rows.Next() {
		var v Circuit
		var tags string
		if err := rows.Scan(&v.ID, &v.Name, &v.Carrier, &v.Kind, &v.Bandwidth, &v.SiteID, &v.PortID,
			&v.Contract, &v.Starts, &v.Ends, &tags, &v.Note, &v.Hidden); err != nil {
			return nil, err
		}
		v.Tags = tagList(tags)
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteCircuit removes a carrier line.
func (s *Store) DeleteCircuit(id int64) error {
	r, err := s.DB.Exec(`DELETE FROM circuits WHERE id = ?`, id)
	return affected(r, err)
}

// ------------------------------------------------------------- helpers

// exists checks an optional reference: 0 means none, anything else must be
// a record of that table.
func (s *Store) exists(table string, id int64, what string) error {
	if id == 0 {
		return nil
	}
	var n int
	if err := s.DB.QueryRow(`SELECT count(*) FROM `+table+` WHERE id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no such %s", what)
	}
	return nil
}

// affected turns an update that matched nothing into ErrNotFound, so that
// editing a record someone else deleted says so instead of succeeding.
func affected(r sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, err := r.RowsAffected(); err == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// isDay reports whether s is a YYYY-MM-DD date.
func isDay(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
