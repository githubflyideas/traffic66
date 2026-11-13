package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// The summaries r_ts, r_host and r_dim are added up in memory and written
// once a minute, while the flow rows are written every second. A stop
// without a shutdown (power cut, kill -9) loses the summaries of up to the
// last minute, and the ranges read from them (over six hours) would miss
// that traffic for good. The pipeline marks a clean stop; after any other
// stop the summaries of the recent hours are rebuilt from the flow rows.

// cleanKey is the meta key the pipeline sets on a clean stop.
const cleanKey = "rollups_clean"

// MarkClean records whether the summaries are complete: true after the
// last summaries were written on shutdown, false while running.
func (s *Store) MarkClean(clean bool) error {
	v := "0"
	if clean {
		v = "1"
	}
	return s.SetMeta(cleanKey, v)
}

// WasClean reports whether the last run stopped cleanly. A data directory
// with no flow rows at all counts as clean.
func (s *Store) WasClean() bool {
	if s.Meta(cleanKey) == "1" {
		return true
	}
	var n int64
	s.DB.QueryRow(`SELECT count(*) FROM (SELECT 1 FROM hot LIMIT 1)`).Scan(&n)
	return n == 0 && len(s.segments()) == 0
}

func (s *Store) segments() []segment {
	s.segMu.RLock()
	defer s.segMu.RUnlock()
	return append([]segment(nil), s.segCache...)
}

// RebuildRecentRollups rebuilds the summaries of every hour that holds a
// row written recently: the rows not yet sealed, and those in a file
// sealed in the last ten minutes. It returns the first hour rebuilt (zero
// when there was nothing to rebuild). The device and interface summaries
// are written in the same transaction as the seal and need no rebuild;
// the sampled-interface marks (obsif, obsunk) cannot be rebuilt from the
// rows and are kept.
func (s *Store) RebuildRecentRollups(now time.Time) (time.Time, error) {
	var from time.Time
	var h sql.NullTime
	s.DB.QueryRow(`SELECT min(ts) FROM hot`).Scan(&h)
	if h.Valid {
		from = h.Time.UTC()
	}
	for _, g := range s.segments() {
		if fi, err := os.Stat(g.path); err == nil && now.Sub(fi.ModTime()) < 10*time.Minute && (from.IsZero() || g.min.Before(from)) {
			from = g.min.UTC()
		}
	}
	if from.IsZero() {
		return from, nil
	}
	from = from.Truncate(time.Hour)
	src := fmt.Sprintf("(SELECT * FROM %s WHERE ts >= ? AND NOT dup)", s.Source(from, now.Add(48*time.Hour)))
	sum := func(c string) string {
		return fmt.Sprintf("sum(CASE WHEN %s > %d THEN 0 ELSE %s END)::UBIGINT", c, wrapLimit, c)
	}
	sums := sum("bytes") + ", " + sum("wire") + ", " + sum("pkts") + ", " + sum("flows")
	var dims string
	for i, d := range viewDims {
		ok := "true"
		if d[0] == "asn" {
			ok = "peer_asn <> 0"
		}
		if i > 0 {
			dims += ", "
		}
		dims += fmt.Sprintf(`{'d': '%s', 'v': %s, 'ok': %s}`, d[0], d[1], ok)
	}
	stmts := []string{
		`DELETE FROM r_ts WHERE ts >= ?`,
		`DELETE FROM r_host WHERE ts >= ?`,
		`DELETE FROM r_dim WHERE ts >= ? AND dim NOT IN ('obsif', 'obsunk')`,
		`INSERT INTO r_ts SELECT ts, app, ` + sums + ` FROM ` + src + ` GROUP BY ALL`,
		`INSERT INTO r_host SELECT date_trunc('hour', ts), client, 1, cli_int, ` + sums + ` FROM ` + src + ` GROUP BY ALL`,
		`INSERT INTO r_host SELECT date_trunc('hour', ts), server, 2, srv_int, ` + sums + ` FROM ` + src + ` GROUP BY ALL`,
		`INSERT INTO r_dim SELECT date_trunc('hour', ts), d.d, d.v, ` + sums + ` FROM (
			SELECT *, unnest(list_filter([` + dims + `], x -> x.ok)) AS d FROM ` + src + `) GROUP BY ALL`,
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.DB.Begin()
	if err != nil {
		return from, err
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q, from); err != nil {
			tx.Rollback()
			return from, fmt.Errorf("rebuild summaries: %w", err)
		}
	}
	return from, tx.Commit()
}
