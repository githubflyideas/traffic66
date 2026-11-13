package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PurgeExporter deletes every flow record one device exported: from the
// table of recent rows and from the flow files, which are rewritten without
// them. Its interface summaries and counters go with them, and the overall
// summaries of the hours it had data in are rebuilt from the flow records
// that remain. Summaries older than the flow records (beyond -retention-days)
// cannot be rebuilt and keep its share.
func (s *Store) PurgeExporter(exporter string) (PurgeSize, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	var done PurgeSize
	for _, a := range []interface{ Flush() error }{s.appHot, s.appTS, s.appHost, s.appDim, s.appIfc} {
		if err := a.Flush(); err != nil {
			return done, err
		}
	}
	far := time.Now().Add(24 * time.Hour)
	src := s.Source(time.Time{}, far)
	var n int64
	var first, last, rawStart time.Time
	if err := s.DB.QueryRow(`SELECT count(*), coalesce(min(ts), TIMESTAMP '1970-01-01'), coalesce(max(ts), TIMESTAMP '1970-01-01') FROM `+src+` WHERE exporter = ?`, exporter).Scan(&n, &first, &last); err != nil {
		return done, err
	}
	if err := s.DB.QueryRow(`SELECT coalesce(min(ts), TIMESTAMP '1970-01-01') FROM ` + src).Scan(&rawStart); err != nil {
		return done, err
	}
	defer s.loadSegments()
	if n > 0 {
		res, err := s.DB.Exec(`DELETE FROM hot WHERE exporter = ?`, exporter)
		if err != nil {
			return done, err
		}
		if _, err := s.DB.Exec(`UPDATE hot SET dup = false WHERE rowid IN (SELECT rowid FROM (SELECT rowid, ` + promoteDup + ` AS keep FROM hot WINDOW k AS (PARTITION BY ` + dupKey + `)) WHERE keep = false)`); err != nil {
			return done, err
		}
		hot, _ := res.RowsAffected()
		done.Rows += hot
		s.segMu.RLock()
		segs := append([]segment(nil), s.segCache...)
		s.segMu.RUnlock()
		for _, g := range segs {
			dropped, _, err := s.dropFromSegment(g, exporter)
			if err != nil {
				return done, fmt.Errorf("purge %s: %w", g.path, err)
			}
			done.Rows += dropped
			if dropped > 0 {
				done.Files++
			}
		}
	}
	// What is keyed by the device itself goes for all time.
	for _, q := range []string{
		`DELETE FROM r_ts_if WHERE iface = ? OR starts_with(iface, ? || '/')`,
		`DELETE FROM r_dim_if WHERE iface = ? OR starts_with(iface, ? || '/')`,
		`DELETE FROM r_dim WHERE (dim IN ('exporter', 'obsunk') AND val = ?) OR (dim = 'obsif' AND starts_with(val, ? || '/'))`,
	} {
		if _, err := s.DB.Exec(q, exporter, exporter); err != nil {
			return done, err
		}
	}
	if _, err := s.DB.Exec(`DELETE FROM ifc WHERE exporter = ?`, exporter); err != nil {
		return done, err
	}
	if n > 0 {
		// The overall summaries of the hours the device had data in, rebuilt
		// from what remains; an hour the flow records cover only in part is
		// left as it is.
		from := first.UTC().Truncate(time.Hour)
		if rs := rawStart.UTC(); from.Before(rs) {
			from = rs.Truncate(time.Hour)
			if from.Before(rs) {
				from = from.Add(time.Hour)
			}
		}
		to := last.UTC().Truncate(time.Hour).Add(time.Hour)
		if from.Before(to) {
			if err := s.rebuildSummaries(from, to); err != nil {
				return done, err
			}
		}
	}
	s.dsMu.Lock()
	s.dsTS, s.dsAt = time.Time{}, time.Time{}
	s.dsMu.Unlock()
	return done, nil
}

// rebuildSummaries recomputes r_ts, r_host and r_dim for [from, to) from the
// flow records, the way the pipeline builds them (see pipeline.rollup):
// duplicates left out, minutes for r_ts and hours for the rest. The rows
// counted per interface (obsif, obsunk) are kept: they are keyed by device.
func (s *Store) rebuildSummaries(from, to time.Time) error {
	src := fmt.Sprintf("(SELECT * FROM %s WHERE ts >= TIMESTAMP '%s' AND ts < TIMESTAMP '%s' AND NOT dup)", s.Source(from, to),
		from.UTC().Format("2006-01-02 15:04:05"), to.UTC().Format("2006-01-02 15:04:05"))
	sums := "sum(bytes)::UBIGINT, sum(wire)::UBIGINT, sum(pkts)::UBIGINT, sum(flows)::UBIGINT"
	proto := "CASE proto WHEN 6 THEN 'tcp' WHEN 17 THEN 'udp' ELSE CAST(proto AS VARCHAR) END"
	dims := `[
		{'d': 'app', 'v': app, 'ok': true},
		{'d': 'port', 'v': CAST(svc_port AS VARCHAR) || '/' || ` + proto + ` || chr(9) || app, 'ok': true},
		{'d': 'country', 'v': CASE WHEN dir = ` + fmt.Sprint(DirInternal) + ` THEN '__internal__' ELSE coalesce(peer_cc, '') END, 'ok': true},
		{'d': 'proto', 'v': CAST(proto AS VARCHAR), 'ok': true},
		{'d': 'dir', 'v': CAST(dir AS VARCHAR), 'ok': true},
		{'d': 'exporter', 'v': exporter, 'ok': true},
		{'d': 'segment', 'v': coalesce(segment, ''), 'ok': true},
		{'d': 'encap', 'v': CAST(encap AS VARCHAR), 'ok': true},
		{'d': 'asn', 'v': CAST(peer_asn AS VARCHAR) || chr(9) || coalesce(peer_org, ''), 'ok': peer_asn <> 0}]`
	win := "ts >= ? AND ts < ?"
	stmts := []struct {
		q    string
		args []any
	}{
		{`DELETE FROM r_ts WHERE ` + win, []any{from, to}},
		{`DELETE FROM r_host WHERE ` + win, []any{from, to}},
		{`DELETE FROM r_dim WHERE ` + win + ` AND dim NOT IN ('obsif', 'obsunk')`, []any{from, to}},
		{`INSERT INTO r_ts SELECT ts, app, ` + sums + ` FROM ` + src + ` GROUP BY ALL`, nil},
		{`INSERT INTO r_host SELECT date_trunc('hour', ts), client, 1, cli_int, ` + sums + ` FROM ` + src + ` GROUP BY ALL`, nil},
		{`INSERT INTO r_host SELECT date_trunc('hour', ts), server, 2, srv_int, ` + sums + ` FROM ` + src + ` GROUP BY ALL`, nil},
		{`INSERT INTO r_dim SELECT date_trunc('hour', ts), x.d, x.v, ` + sums + ` FROM (SELECT *, unnest(list_filter(` + dims + `, x -> x.ok)) AS x FROM ` + src + `) GROUP BY ALL`, nil},
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	for _, st := range stmts {
		if _, err := tx.Exec(st.q, st.args...); err != nil {
			tx.Rollback()
			return fmt.Errorf("rebuild summaries: %w", err)
		}
	}
	return tx.Commit()
}

// dropFromSegment rewrites one hourly file without a device's records, or
// removes it when nothing else is left.
func (s *Store) dropFromSegment(g segment, exporter string) (dropped int64, removed bool, err error) {
	src := "'" + sqlPath(g.path) + "'"
	var total, keep int64
	var mn, mx time.Time
	if err := s.DB.QueryRow(`SELECT count(*), count(*) FILTER (WHERE exporter <> ?), coalesce(min(ts) FILTER (WHERE exporter <> ?), TIMESTAMP '1970-01-01'), coalesce(max(ts) FILTER (WHERE exporter <> ?), TIMESTAMP '1970-01-01') FROM read_parquet(`+src+`)`,
		exporter, exporter, exporter).Scan(&total, &keep, &mn, &mx); err != nil {
		return 0, false, err
	}
	if keep == total {
		return 0, false, nil
	}
	if keep == 0 {
		if err := os.Remove(g.path); err != nil && !os.IsNotExist(err) {
			return 0, false, err
		}
		if _, err := s.DB.Exec(`DELETE FROM segments WHERE path = ?`, g.path); err != nil {
			return 0, false, err
		}
		os.Remove(filepath.Dir(g.path))
		return total, true, nil
	}
	tmp := g.path + ".purge"
	q := fmt.Sprintf(`COPY (SELECT * REPLACE (`+promoteDup+` AS dup) FROM read_parquet(%s) WHERE exporter <> ? WINDOW k AS (PARTITION BY `+dupKey+`) ORDER BY ts, client) TO '%s' (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)`, src, sqlPath(tmp))
	if _, err := s.DB.Exec(q, exporter); err != nil {
		os.Remove(tmp)
		return 0, false, err
	}
	if err := os.Rename(tmp, g.path); err != nil {
		os.Remove(tmp)
		return 0, false, err
	}
	if _, err := s.DB.Exec(`UPDATE segments SET min_ts = ?, max_ts = ?, rows = ? WHERE path = ?`, mn, mx, keep, g.path); err != nil {
		return 0, false, err
	}
	return total - keep, false, nil
}

// StoredExporter is one device with flow records kept.
type StoredExporter struct {
	Exporter string    `json:"exporter"`
	Rows     int64     `json:"rows"`
	Wire     uint64    `json:"wire"`
	Last     time.Time `json:"last"`
}

// StoredExporters lists the devices that have flow records kept, most
// traffic first.
func (s *Store) StoredExporters() ([]StoredExporter, error) {
	src := s.Source(time.Time{}, time.Now().Add(24*time.Hour))
	rows, err := s.DB.Query(`SELECT exporter, count(*), CAST(CASE WHEN sum(wire) > 18446744073709551615::HUGEINT THEN 18446744073709551615::HUGEINT ELSE sum(wire) END AS UBIGINT), max(ts) FROM ` + src + ` GROUP BY exporter ORDER BY 3 DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredExporter
	for rows.Next() {
		var e StoredExporter
		if err := rows.Scan(&e.Exporter, &e.Rows, &e.Wire, &e.Last); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// A flow two devices reported is stored twice, the later copy marked as a
// duplicate and left out of the totals. When the device whose copy counted
// is deleted, the flow would be counted by nobody: one remaining copy of
// each such flow (same minute and key as the pipeline's duplicate check)
// counts instead.
const (
	dupKey     = "ts, src, dst, sport, dport, proto"
	promoteDup = "CASE WHEN dup AND NOT bool_or(NOT dup) OVER k AND row_number() OVER (PARTITION BY " + dupKey + " ORDER BY exporter) = 1 THEN false ELSE dup END"
)
