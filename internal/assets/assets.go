// Package assets is the asset ledger: sites, racks, devices, ports and the
// links between them, the circuits that reach them, and the networks
// written down against a device.
//
// It is deliberately a ledger, not a discovery tool. Nothing here is
// polled or guessed: an operator writes it down, or pastes it in from a
// switch's own output. What traffic66 sees on the wire is matched against
// it, never merged into it, so the ledger stays the record of what was
// meant to be there — which is the only thing it is useful for when the
// two disagree.
//
// It keeps its own database file. The flow store is append-only, huge and
// memory-limited, and deleting traffic data is a routine operation; the
// ledger is small, hand-written and must survive every one of those.
package assets

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// File is the ledger's database file inside the data directory.
const File = "assets.duckdb"

// Store is the asset ledger.
type Store struct {
	conn *duckdb.Connector
	DB   *sql.DB
}

// schema is created on every open. Columns are added by later versions
// with ALTER TABLE in migrate, never by rewriting this list, so that an
// older ledger opens unchanged.
var schema = []string{
	`CREATE SEQUENCE IF NOT EXISTS asset_seq`,

	`CREATE TABLE IF NOT EXISTS sites (id BIGINT PRIMARY KEY, name VARCHAR, address VARCHAR,
		note VARCHAR, hidden BOOLEAN, created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS racks (id BIGINT PRIMARY KEY, site_id BIGINT, name VARCHAR,
		units INTEGER, note VARCHAR, hidden BOOLEAN, created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS devices (id BIGINT PRIMARY KEY, rack_id BIGINT, name VARCHAR,
		role VARCHAR, vendor VARCHAR, model VARCHAR, serial VARCHAR, mgmt_ip VARCHAR,
		pos_u INTEGER, height_u INTEGER, tags VARCHAR, note VARCHAR, hidden BOOLEAN,
		created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS ports (id BIGINT PRIMARY KEY, device_id BIGINT, name VARCHAR,
		ifindex UINTEGER, speed UBIGINT, kind VARCHAR, note VARCHAR,
		created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS links (id BIGINT PRIMARY KEY, a_port BIGINT, b_port BIGINT,
		note VARCHAR, created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS circuits (id BIGINT PRIMARY KEY, name VARCHAR, carrier VARCHAR,
		kind VARCHAR, bandwidth VARCHAR, site_id BIGINT, port_id BIGINT, contract VARCHAR,
		starts VARCHAR, ends VARCHAR, tags VARCHAR, note VARCHAR, hidden BOOLEAN,
		created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS subnets (id BIGINT PRIMARY KEY, prefix VARCHAR, name VARCHAR,
		vlan INTEGER, site_id BIGINT, gateway VARCHAR, tags VARCHAR, note VARCHAR, hidden BOOLEAN,
		created TIMESTAMP, updated TIMESTAMP)`,
	// Addresses were assigned here once, from a page of their own. Both
	// are gone: it was a second place to keep what the system that hands
	// out addresses already knows. The table is still created and left
	// alone, because a ledger that was filled in before this must not lose
	// what somebody typed into it.
	`CREATE TABLE IF NOT EXISTS addrs (id BIGINT PRIMARY KEY, subnet_id BIGINT, ip VARCHAR,
		name VARCHAR, kind VARCHAR, device_id BIGINT, port_id BIGINT, mac VARCHAR,
		tags VARCHAR, note VARCHAR, hidden BOOLEAN, created TIMESTAMP, updated TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS meta (k VARCHAR PRIMARY KEY, v VARCHAR)`,

	`CREATE UNIQUE INDEX IF NOT EXISTS subnets_prefix ON subnets (prefix)`,
	`CREATE INDEX IF NOT EXISTS ports_device ON ports (device_id)`,
	`CREATE INDEX IF NOT EXISTS devices_rack ON devices (rack_id)`,
}

// Open creates or opens the ledger in dir.
func Open(dir string) (*Store, error) {
	init := func(ex driver.ExecerContext) error {
		// The ledger is tiny; it must not take memory from the flow store.
		for _, q := range []string{"SET memory_limit='128MB'", "SET threads=1"} {
			if _, err := ex.ExecContext(context.Background(), q, nil); err != nil {
				return fmt.Errorf("asset ledger setup failed (%s): %w", q, err)
			}
		}
		return nil
	}
	dsn := filepath.Join(dir, File) + "?" + url.Values{
		"autoinstall_known_extensions": {"false"},
		"autoload_known_extensions":    {"false"},
		"home_directory":               {dir},
	}.Encode()
	c, err := duckdb.NewConnector(dsn, init)
	if err != nil {
		return nil, fmt.Errorf("asset ledger: %w", err)
	}
	s := &Store{conn: c, DB: sql.OpenDB(c)}
	// one writer: the ledger is written by people, never in bulk, and a
	// single connection keeps the sequence and the conflict checks simple
	s.DB.SetMaxOpenConns(1)
	for _, q := range schema {
		if _, err := s.DB.Exec(q); err != nil {
			s.Close()
			return nil, fmt.Errorf("asset ledger: %s: %w", q, err)
		}
	}
	// Columns added after a release: a ledger written by an older one is
	// opened as it is and grown here, never rewritten.
	for _, q := range []string{
		`ALTER TABLE subnets ADD COLUMN IF NOT EXISTS device_id BIGINT DEFAULT 0`,
	} {
		if _, err := s.DB.Exec(q); err != nil {
			s.Close()
			return nil, fmt.Errorf("asset ledger: %s: %w", q, err)
		}
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// next takes the next identifier.
func (s *Store) next() (int64, error) {
	var id int64
	err := s.DB.QueryRow(`SELECT nextval('asset_seq')`).Scan(&id)
	return id, err
}

// Counts is how much of each kind the ledger holds, for the quota checks
// and the page header.
type Counts struct {
	Sites    int `json:"sites"`
	Racks    int `json:"racks"`
	Devices  int `json:"devices"`
	Ports    int `json:"ports"`
	Circuits int `json:"circuits"`
	Subnets  int `json:"subnets"`
}

// Counts reports the ledger's size. Hidden records are counted: they are
// kept, so they take up quota.
func (s *Store) Counts() Counts {
	var c Counts
	for _, p := range []struct {
		table string
		into  *int
	}{
		{"sites", &c.Sites}, {"racks", &c.Racks}, {"devices", &c.Devices},
		{"ports", &c.Ports}, {"circuits", &c.Circuits}, {"subnets", &c.Subnets},
	} {
		s.DB.QueryRow(`SELECT count(*) FROM ` + p.table).Scan(p.into)
	}
	return c
}
