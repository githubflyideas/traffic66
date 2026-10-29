package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Summaries by device and by interface.
//
// The summaries in r_ts and r_dim are of all the traffic. Narrowed
// to one device or one interface, which is how most of the pages are read,
// a range of days used to be answered from the flow records themselves: a
// week is tens of millions of rows, read two or three times over for one
// chart. These tables hold the same summaries once per device and once per
// interface, so those ranges are read from them instead.
//
// There is no such table for addresses: a client is seldom seen more than
// a few times in an hour, so an hourly summary by address and interface
// has more rows than the flow records it sums. Questions about addresses
// are read from the flow records, in queries written to stay within the
// memory there is (query.go, series.go, rings.go).
//
// iface is the exporter for a device ("10.0.0.1") or "exporter/ifindex"
// for an interface. side says how the traffic used the interface:
// sideIn came in through it, sideOut went out through it, sideBoth came in
// and went out through it (in and out the same interface). For a device
// side is sideDevice. A flow row is counted once per side, so an interface
// filter adds up sides 1 to 3 without counting anything twice.
//
// They are written when the flow records are sealed into a file, from
// exactly the rows sealed, in the same transaction that takes them out of
// the hot table; the rows not sealed yet are read from the hot table, which
// is small. They count a device's own traffic: rows marked as duplicates of
// another device's report are included, as they are for any query that
// keeps to one device.
const (
	sideDevice = 0
	sideIn     = 1
	sideOut    = 2
	sideBoth   = 3
)

var viewSchema = []string{
	`CREATE TABLE IF NOT EXISTS r_ts_if (ts TIMESTAMP, iface VARCHAR, side UTINYINT, app VARCHAR, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS r_dim_if (ts TIMESTAMP, iface VARCHAR, side UTINYINT, dim VARCHAR, val VARCHAR, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
}

var viewTables = []string{"r_ts_if", "r_dim_if"}

// viewsDone is the meta key set once the summaries cover every flow file.
const viewsDone = "views_v1"

// tsStep is the bucket of r_ts_if: the finest step a range long enough
// for the summaries is drawn with.
const tsStep = "INTERVAL 5 MINUTE"

// The dimensions of r_dim_if, as r_dim has them.
var viewDims = [][2]string{
	{"app", "app"},
	{"port", portProtoExpr + " || chr(9) || app"},
	{"country", "CASE WHEN dir = 3 THEN '__internal__' ELSE peer_cc END"},
	{"proto", "CAST(proto AS VARCHAR)"},
	{"dir", "CAST(dir AS VARCHAR)"},
	{"exporter", "exporter"},
	{"segment", "segment"},
	{"encap", "CAST(encap AS VARCHAR)"},
	{"asn", "CAST(peer_asn AS VARCHAR) || chr(9) || peer_org"},
}

// viewInserts are the statements that add the summaries of the flow rows
// in src (a table or subquery) to the tables. Each reads src once: a row
// is repeated for the device, the interface it came in through and the
// one it went out through (and for r_dim once more for each dimension)
// with unnest, rather than src being read again for each of them.
func viewInserts(src string) []string {
	// a counter that wrapped below zero (written by 1.6.0 and earlier) is
	// left out rather than allowed to overflow the sum
	ok := func(c string) string {
		return fmt.Sprintf("sum(CASE WHEN %s > %d THEN 0 ELSE %s END)::UBIGINT", c, wrapLimit, c)
	}
	sums := ok("bytes") + ", " + ok("wire") + ", " + ok("pkts") + ", " + ok("flows")
	v := `(SELECT *, unnest(list_filter([
			{'f': exporter, 's': 0::UTINYINT, 'ok': true},
			{'f': exporter || '/' || in_if, 's': (CASE WHEN in_if = out_if THEN 3 ELSE 1 END)::UTINYINT, 'ok': in_if <> 0},
			{'f': exporter || '/' || out_if, 's': 2::UTINYINT, 'ok': out_if <> 0 AND out_if <> in_if}], x -> x.ok)) AS v FROM ` + src + `)`
	var dims []string
	for _, d := range viewDims {
		okd := "true"
		if d[0] == "asn" {
			okd = "peer_asn <> 0"
		}
		dims = append(dims, fmt.Sprintf(`{'d': '%s', 'v': %s, 'ok': %s}`, d[0], d[1], okd))
	}
	return []string{
		`INSERT INTO r_ts_if SELECT time_bucket(` + tsStep + `, ts), v.f, v.s, app, ` + sums + ` FROM ` + v + ` GROUP BY ALL`,
		`INSERT INTO r_dim_if SELECT date_trunc('hour', ts), v.f, v.s, d.d, d.v, ` + sums + ` FROM (
			SELECT *, unnest(list_filter([` + strings.Join(dims, ", ") + `], x -> x.ok)) AS d FROM ` + v + `) GROUP BY ALL`,
	}
}

// view is which summaries answer a query: all traffic (key "") or one
// device or interface and the sides of it.
type view struct {
	key   string
	sides []int
}

// view works out the view a query's filters describe. ok is false when
// they narrow it some other way, and only the flow records can answer.
func (q Query) view() (view, bool) {
	if len(q.Filters) == 0 {
		return view{}, true
	}
	var key string
	sides := []int{sideIn, sideOut, sideBoth}
	device := ""
	for _, f := range q.Filters {
		if f.Neg {
			return view{}, false
		}
		var want []int
		switch f.Field {
		case "exporter":
			if device != "" && device != f.Value {
				return view{}, false
			}
			device = f.Value
			continue
		case "iface":
			want = []int{sideIn, sideOut, sideBoth}
		case "iface_in":
			want = []int{sideIn, sideBoth}
		case "iface_out":
			want = []int{sideOut, sideBoth}
		default:
			return view{}, false
		}
		if key != "" && key != f.Value {
			return view{}, false
		}
		key = f.Value
		sides = slices.DeleteFunc(sides, func(x int) bool { return !slices.Contains(want, x) })
	}
	switch {
	case key == "":
		return view{key: device, sides: []int{sideDevice}}, true
	case device != "" && !strings.HasPrefix(key, device+"/"):
		return view{}, false
	case len(sides) == 0:
		return view{}, false
	}
	return view{key: key, sides: sides}, true
}

// rollupOK reports whether the summaries answer q. hosts: the answer needs
// the summaries by address, which only all traffic has.
func (s *Store) rollupOK(q Query, hosts bool) bool {
	if q.span() <= 6*time.Hour {
		return false
	}
	v, ok := q.view()
	return ok && (v.key == "" || (!hosts && s.viewsReady.Load()))
}

// rollups replaces the placeholders {r_ts}, {r_host} and {r_dim} in sqlq
// with the tables or subqueries that hold q's summaries, and returns the
// arguments those need, which come before any of sqlq's own.
func (s *Store) rollups(q Query, sqlq string) (string, []any, error) {
	v, _ := q.view()
	var args []any
	for _, t := range []string{"r_ts", "r_host", "r_dim"} {
		ph := "{" + t + "}"
		if !strings.Contains(sqlq, ph) {
			continue
		}
		if v.key == "" {
			sqlq = strings.ReplaceAll(sqlq, ph, t)
			continue
		}
		src, a, err := s.viewSource(q, v, t)
		if err != nil {
			return "", nil, err
		}
		for range strings.Count(sqlq, ph) {
			args = append(args, a...)
		}
		sqlq = strings.ReplaceAll(sqlq, ph, src)
	}
	return sqlq, args, nil
}

// viewSource is one summary table for a view, shaped like the table of all
// traffic it stands in for: the sealed rows from the view's table and the
// rows still in the hot table, summarised on the fly.
func (s *Store) viewSource(q Query, v view, table string) (string, []any, error) {
	where, wargs, err := q.where()
	if err != nil {
		return "", nil, err
	}
	sides := make([]string, len(v.sides))
	for i, x := range v.sides {
		sides[i] = strconv.Itoa(x)
	}
	sel := fmt.Sprintf(`FROM %s_if WHERE iface = ? AND side IN (%s)`, table, strings.Join(sides, ","))
	args := []any{v.key}
	var hot []string
	switch table {
	case "r_ts":
		sel = "SELECT ts, app, bytes, wire, pkts, flows " + sel
		hot = []string{"SELECT ts, app, bytes, wire, pkts, flows FROM hot WHERE " + where}
	case "r_dim":
		sel = "SELECT ts, dim, val, bytes, wire, pkts, flows " + sel
		for _, d := range viewDims {
			c := where
			if d[0] == "asn" {
				c += " AND peer_asn <> 0"
			}
			hot = append(hot, fmt.Sprintf("SELECT ts, '%s', %s, bytes, wire, pkts, flows FROM hot WHERE %s", d[0], d[1], c))
		}
	}
	for range hot {
		args = append(args, wargs...)
	}
	return "(" + sel + " UNION ALL " + strings.Join(hot, " UNION ALL ") + ")", args, nil
}

// sealViews adds the summaries of the hot rows before boundary, in the
// transaction that deletes them from the hot table.
func sealViews(tx *sql.Tx, boundary time.Time) error {
	src := fmt.Sprintf("(SELECT * FROM hot WHERE ts < TIMESTAMP '%s')", boundary.UTC().Format("2006-01-02 15:04:05.999999"))
	for _, q := range viewInserts(src) {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("summaries: %w", err)
		}
	}
	return nil
}

// buildViews summarises the flow files sealed before the summaries
// existed, once per data directory, a file at a time so the memory it
// needs is a file's worth. Until it is done the pages read those ranges
// from the flow records as before.
func (s *Store) buildViews(ctx context.Context) {
	defer func() {
		s.vmu.Lock()
		s.vStop = nil
		s.vmu.Unlock()
		s.bg.Done()
	}()
	// Start over: an earlier run may have stopped half way. The files
	// sealed from here on are summarised by the seal itself, so the list
	// is taken, and the tables emptied, with sealing held off.
	s.wmu.Lock()
	for _, t := range viewTables {
		if _, err := s.DB.Exec(`DELETE FROM ` + t); err != nil {
			s.wmu.Unlock()
			log.Printf("store: summaries: %v", err)
			return
		}
	}
	s.segMu.RLock()
	segs := append([]segment(nil), s.segCache...)
	s.segMu.RUnlock()
	s.wmu.Unlock()
	t0 := time.Now()
	for i, g := range segs {
		if ctx.Err() != nil {
			return
		}
		src := "(SELECT * FROM read_parquet('" + sqlPath(g.path) + "'))"
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("store: summaries: %v", err)
			return
		}
		for _, q := range viewInserts(src) {
			if _, err = tx.ExecContext(ctx, q); err != nil {
				break
			}
		}
		if err != nil {
			tx.Rollback()
			if ctx.Err() == nil {
				log.Printf("store: summaries of %s: %v", g.path, err)
			}
			return
		}
		if err := tx.Commit(); err != nil {
			log.Printf("store: summaries: %v", err)
			return
		}
		if (i+1)%50 == 0 {
			log.Printf("store: summarising flow files by device and interface: %d of %d", i+1, len(segs))
		}
	}
	if err := s.SetMeta(viewsDone, time.Now().UTC().Format(time.RFC3339)); err != nil {
		log.Printf("store: summaries: %v", err)
		return
	}
	s.viewsReady.Store(true)
	if len(segs) > 0 {
		log.Printf("store: summarised %d flow files by device and interface in %s", len(segs), time.Since(t0).Round(time.Second))
	}
}

// startViews builds the summaries of the flow files in the background.
func (s *Store) startViews() {
	ctx, stop := context.WithCancel(context.Background())
	s.vmu.Lock()
	s.vStop = stop
	s.vmu.Unlock()
	s.bg.Add(1)
	go s.buildViews(ctx)
}

// stopViews stops a build in progress, waits for it, and reports whether
// there was one.
func (s *Store) stopViews() bool {
	s.vmu.Lock()
	stop := s.vStop
	s.vmu.Unlock()
	if stop == nil {
		return false
	}
	stop()
	s.bg.Wait()
	return true
}
