// Package store keeps flows in DuckDB: the current hour in a table, older
// hours as Parquet files, plus small rollup tables for long time ranges.
package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// Row is one stored flow slice (one flow, one minute).
type Row struct {
	TS        time.Time
	Src, Dst  string
	SPort     uint16
	DPort     uint16
	Proto     uint8
	Client    string
	Server    string
	Cli4      uint32
	Srv4      uint32
	SvcPort   uint16
	App       string
	Bytes     uint64
	Wire      uint64
	Pkts      uint64
	Flows     uint32
	Exporter  string
	SrcType   uint8
	InIf      uint32
	OutIf     uint32
	VLAN      uint16
	Encap     uint8
	Dir       uint8
	PeerCC    string
	PeerASN   uint32
	PeerOrg   string
	Segment   string
	Threat    string
	TCPFlags  uint8
	Mult      float32
	SampKnown bool
	Dup       bool
	CliInt    bool
	SrvInt    bool
}

// Direction values stored in Row.Dir.
const (
	DirOutbound = 1 // internal client, external server
	DirInbound  = 2 // external client, internal server
	DirInternal = 3
	DirTransit  = 4
)

const flowCols = `ts TIMESTAMP, src VARCHAR, dst VARCHAR, sport USMALLINT, dport USMALLINT, proto UTINYINT,
	client VARCHAR, server VARCHAR, cli4 UINTEGER, srv4 UINTEGER, svc_port USMALLINT, app VARCHAR,
	bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UINTEGER, exporter VARCHAR, src_type UTINYINT,
	in_if UINTEGER, out_if UINTEGER, vlan USMALLINT, encap UTINYINT, dir UTINYINT,
	peer_cc VARCHAR, peer_asn UINTEGER, peer_org VARCHAR, segment VARCHAR, threat VARCHAR,
	tcp_flags UTINYINT, mult FLOAT, samp_known BOOLEAN, dup BOOLEAN, cli_int BOOLEAN, srv_int BOOLEAN`

var schema = []string{
	`CREATE TABLE IF NOT EXISTS hot (` + flowCols + `)`,
	`CREATE TABLE IF NOT EXISTS segments (path VARCHAR PRIMARY KEY, min_ts TIMESTAMP, max_ts TIMESTAMP, rows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS r_ts (ts TIMESTAMP, app VARCHAR, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS r_host (ts TIMESTAMP, ip VARCHAR, role UTINYINT, internal BOOLEAN, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS r_dim (ts TIMESTAMP, dim VARCHAR, val VARCHAR, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS ifc (ts TIMESTAMP, exporter VARCHAR, ifindex UINTEGER, speed UBIGINT, in_oct UBIGINT, out_oct UBIGINT, in_pkts UBIGINT, out_pkts UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS meta (k VARCHAR PRIMARY KEY, v VARCHAR)`,
}

// Options configure a Store.
type Options struct {
	Dir            string
	MemoryFraction float64 // share of physical memory DuckDB may use
	Threads        int
	RawDays        int // flow detail retention
	RollupDays     int
}

// Store owns the DuckDB database and the Parquet directory.
type Store struct {
	opt  Options
	conn *duckdb.Connector
	DB   *sql.DB

	wmu     sync.Mutex // guards appenders and sealing
	wconn   driver.Conn
	appHot  *duckdb.Appender
	appTS   *duckdb.Appender
	appHost *duckdb.Appender
	appDim  *duckdb.Appender
	appIfc  *duckdb.Appender

	segMu    sync.RWMutex
	segCache []segment
}

type segment struct {
	path     string
	min, max time.Time
}

// Open creates or opens the store in opt.Dir.
func Open(opt Options) (*Store, error) {
	if opt.MemoryFraction <= 0 {
		opt.MemoryFraction = 0.10
	}
	if opt.Threads <= 0 {
		// Leave a core for collection on bigger machines; collection needs
		// well under one core at the design rate, so two threads is safe.
		opt.Threads = runtime.NumCPU() - 1
		if opt.Threads < 2 {
			opt.Threads = min(2, runtime.NumCPU())
		}
	}
	if opt.RawDays <= 0 {
		opt.RawDays = 30
	}
	if opt.RollupDays <= 0 {
		opt.RollupDays = 400
	}
	if err := os.MkdirAll(filepath.Join(opt.Dir, "raw"), 0o755); err != nil {
		return nil, err
	}
	tmp := filepath.Join(opt.Dir, "tmp")
	os.MkdirAll(tmp, 0o755)
	memMB := int64(float64(totalMemory()) * opt.MemoryFraction / (1 << 20))
	if memMB < 256 {
		memMB = 256
	}
	init := func(ex driver.ExecerContext) error {
		for _, q := range []string{
			fmt.Sprintf("SET memory_limit='%dMB'", memMB),
			fmt.Sprintf("SET threads=%d", opt.Threads),
			fmt.Sprintf("SET temp_directory='%s'", sqlPath(tmp)),
			"SET preserve_insertion_order=false",
		} {
			if _, err := ex.ExecContext(context.Background(), q, nil); err != nil {
				return fmt.Errorf("%s: %w", q, err)
			}
		}
		return nil
	}
	c, err := duckdb.NewConnector(filepath.Join(opt.Dir, "traffic66.duckdb"), init)
	if err != nil {
		return nil, err
	}
	s := &Store{opt: opt, conn: c, DB: sql.OpenDB(c)}
	for _, q := range schema {
		if _, err := s.DB.Exec(q); err != nil {
			return nil, fmt.Errorf("schema: %w", err)
		}
	}
	if err := s.openAppenders(); err != nil {
		return nil, err
	}
	if err := s.loadSegments(); err != nil {
		return nil, err
	}
	log.Printf("store: %s, DuckDB memory limit %d MB, %d threads", opt.Dir, memMB, opt.Threads)
	return s, nil
}

func sqlPath(p string) string { return strings.ReplaceAll(filepath.ToSlash(p), "'", "''") }

func (s *Store) openAppenders() error {
	var err error
	s.wconn, err = s.conn.Connect(context.Background())
	if err != nil {
		return err
	}
	open := func(t string) *duckdb.Appender {
		if err != nil {
			return nil
		}
		var a *duckdb.Appender
		a, err = duckdb.NewAppenderFromConn(s.wconn, "", t)
		return a
	}
	s.appHot, s.appTS, s.appHost, s.appDim, s.appIfc = open("hot"), open("r_ts"), open("r_host"), open("r_dim"), open("ifc")
	return err
}

// AppendRows writes flow slices.
func (s *Store) AppendRows(rows []Row) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for i := range rows {
		r := &rows[i]
		if err := s.appHot.AppendRow(r.TS.UTC(), r.Src, r.Dst, r.SPort, r.DPort, r.Proto, r.Client, r.Server, r.Cli4, r.Srv4,
			r.SvcPort, r.App, r.Bytes, r.Wire, r.Pkts, r.Flows, r.Exporter, r.SrcType, r.InIf, r.OutIf, r.VLAN, r.Encap, r.Dir,
			r.PeerCC, r.PeerASN, r.PeerOrg, r.Segment, r.Threat, r.TCPFlags, r.Mult, r.SampKnown, r.Dup, r.CliInt, r.SrvInt); err != nil {
			return err
		}
	}
	return s.appHot.Flush()
}

// Counters is one rollup accumulator.
type Counters struct{ Bytes, Wire, Pkts, Flows uint64 }

func (c *Counters) Add(o Counters) {
	c.Bytes += o.Bytes
	c.Wire += o.Wire
	c.Pkts += o.Pkts
	c.Flows += o.Flows
}

type TSKey struct {
	TS  time.Time
	App string
}
type HostKey struct {
	TS       time.Time
	IP       string
	Role     uint8
	Internal bool
}
type DimKey struct {
	TS       time.Time
	Dim, Val string
}

// AppendRollups writes partial rollup rows; queries sum duplicates.
func (s *Store) AppendRollups(ts map[TSKey]*Counters, host map[HostKey]*Counters, dim map[DimKey]*Counters) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for k, c := range ts {
		if err := s.appTS.AppendRow(k.TS.UTC(), k.App, c.Bytes, c.Wire, c.Pkts, c.Flows); err != nil {
			return err
		}
	}
	for k, c := range host {
		if err := s.appHost.AppendRow(k.TS.UTC(), k.IP, k.Role, k.Internal, c.Bytes, c.Wire, c.Pkts, c.Flows); err != nil {
			return err
		}
	}
	for k, c := range dim {
		if err := s.appDim.AppendRow(k.TS.UTC(), k.Dim, k.Val, c.Bytes, c.Wire, c.Pkts, c.Flows); err != nil {
			return err
		}
	}
	for _, a := range []*duckdb.Appender{s.appTS, s.appHost, s.appDim} {
		if err := a.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// AppendCounter stores one interface counter snapshot.
func (s *Store) AppendCounter(ts time.Time, exporter string, ifindex uint32, speed, inOct, outOct, inPkts, outPkts uint64) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.appIfc.AppendRow(ts.UTC(), exporter, ifindex, speed, inOct, outOct, inPkts, outPkts); err != nil {
		return err
	}
	return s.appIfc.Flush()
}

// Seal moves every hot row older than the current hour (with a grace
// period for late records) into a Parquet file.
func (s *Store) Seal(now time.Time) error {
	boundary := now.UTC().Add(-5 * time.Minute).Truncate(time.Hour)
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.appHot.Flush(); err != nil {
		return err
	}
	var n int64
	var mn, mx sql.NullTime
	if err := s.DB.QueryRow(`SELECT count(*), min(ts), max(ts) FROM hot WHERE ts < ?`, boundary).Scan(&n, &mn, &mx); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	name := fmt.Sprintf("%s_%d.parquet", mx.Time.UTC().Format("2006-01-02T15"), now.UnixNano())
	path := filepath.Join(s.opt.Dir, "raw", mx.Time.UTC().Format("2006-01-02"), name)
	os.MkdirAll(filepath.Dir(path), 0o755)
	q := fmt.Sprintf(`COPY (SELECT * FROM hot WHERE ts < ? ORDER BY ts, client) TO '%s' (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)`, sqlPath(path))
	if _, err := s.DB.Exec(q, boundary); err != nil {
		return fmt.Errorf("seal: %w", err)
	}
	if _, err := s.DB.Exec(`INSERT INTO segments VALUES (?, ?, ?, ?)`, path, mn.Time, mx.Time, n); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`DELETE FROM hot WHERE ts < ?`, boundary); err != nil {
		return err
	}
	s.DB.Exec(`CHECKPOINT`)
	log.Printf("store: sealed %d rows into %s", n, filepath.Base(path))
	return s.loadSegments()
}

func (s *Store) loadSegments() error {
	rows, err := s.DB.Query(`SELECT path, min_ts, max_ts FROM segments ORDER BY min_ts`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var segs []segment
	for rows.Next() {
		var g segment
		if err := rows.Scan(&g.path, &g.min, &g.max); err != nil {
			return err
		}
		if _, err := os.Stat(g.path); err == nil {
			segs = append(segs, g)
		}
	}
	s.segMu.Lock()
	s.segCache = segs
	s.segMu.Unlock()
	return rows.Err()
}

// Retain deletes detail and rollups past their retention.
func (s *Store) Retain(now time.Time) error {
	rawCut := now.UTC().AddDate(0, 0, -s.opt.RawDays)
	rollCut := now.UTC().AddDate(0, 0, -s.opt.RollupDays)
	s.segMu.RLock()
	var old []segment
	for _, g := range s.segCache {
		if g.max.Before(rawCut) {
			old = append(old, g)
		}
	}
	s.segMu.RUnlock()
	for _, g := range old {
		os.Remove(g.path)
		s.DB.Exec(`DELETE FROM segments WHERE path = ?`, g.path)
	}
	if len(old) > 0 {
		log.Printf("store: removed %d expired segments", len(old))
		s.loadSegments()
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for _, t := range []string{"r_ts", "r_host", "r_dim"} {
		s.DB.Exec(`DELETE FROM `+t+` WHERE ts < ?`, rollCut)
	}
	s.DB.Exec(`DELETE FROM ifc WHERE ts < ?`, rawCut)
	s.DB.Exec(`DELETE FROM hot WHERE ts < ?`, rawCut)
	return nil
}

// Source returns a SQL relation over all flow rows overlapping [from, to).
func (s *Store) Source(from, to time.Time) string {
	s.segMu.RLock()
	var files []string
	for _, g := range s.segCache {
		if g.max.Before(from.UTC()) || !g.min.Before(to.UTC()) {
			continue
		}
		files = append(files, "'"+sqlPath(g.path)+"'")
	}
	s.segMu.RUnlock()
	if len(files) == 0 {
		return "hot"
	}
	return fmt.Sprintf("(SELECT * FROM hot UNION ALL BY NAME SELECT * FROM read_parquet([%s]))", strings.Join(files, ","))
}

// Usage reports stored rows and bytes on disk.
type Usage struct {
	HotRows     int64
	Segments    int
	SegmentRows int64
	DiskBytes   int64
	Oldest      time.Time
}

func (s *Store) Usage() Usage {
	var u Usage
	s.DB.QueryRow(`SELECT count(*) FROM hot`).Scan(&u.HotRows)
	var mn sql.NullTime
	s.DB.QueryRow(`SELECT count(*), coalesce(sum(rows),0), min(min_ts) FROM segments`).Scan(&u.Segments, &u.SegmentRows, &mn)
	if mn.Valid {
		u.Oldest = mn.Time
	} else {
		var h sql.NullTime
		s.DB.QueryRow(`SELECT min(ts) FROM hot`).Scan(&h)
		u.Oldest = h.Time
	}
	filepath.Walk(s.opt.Dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			u.DiskBytes += info.Size()
		}
		return nil
	})
	return u
}

// Close flushes and closes everything.
func (s *Store) Close() error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for _, a := range []*duckdb.Appender{s.appHot, s.appTS, s.appHost, s.appDim, s.appIfc} {
		if a != nil {
			a.Close()
		}
	}
	if s.wconn != nil {
		s.wconn.Close()
	}
	s.DB.Close()
	return s.conn.Close()
}
