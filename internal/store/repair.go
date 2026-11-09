package store

import (
	"fmt"
	"log"
	"os"
)

// wrapLimit is far above any real per-minute counter. Values above it are
// negative numbers that wrapped around when converted to unsigned, which
// versions up to 1.6.0 could write when a flow spanned hours (a laptop that
// slept with a connection open). One such value makes every sum overflow
// and the pages fail with "value out of range".
const wrapLimit = uint64(1) << 62

// repairWrapped zeroes wrapped counters in the database and the flow files.
// It runs once per data directory.
func (s *Store) repairWrapped() error {
	const key = "repair_wrapped_v1"
	if s.Meta(key) != "" {
		return nil
	}
	bad := fmt.Sprintf("bytes > %d OR wire > %d OR pkts > %d", wrapLimit, wrapLimit, wrapLimit)
	fixed := int64(0)
	for _, t := range []string{"hot", "r_ts", "r_host", "r_dim"} {
		q := fmt.Sprintf(`UPDATE %s SET bytes = CASE WHEN bytes > %[2]d THEN 0 ELSE bytes END,
			wire = CASE WHEN wire > %[2]d THEN 0 ELSE wire END, pkts = CASE WHEN pkts > %[2]d THEN 0 ELSE pkts END
			WHERE %s`, t, wrapLimit, bad)
		r, err := s.DB.Exec(q)
		if err != nil {
			return fmt.Errorf("repair %s: %w", t, err)
		}
		n, _ := r.RowsAffected()
		fixed += n
	}
	s.segMu.RLock()
	segs := append([]segment(nil), s.segCache...)
	s.segMu.RUnlock()
	for _, g := range segs {
		src := "'" + sqlPath(g.path) + "'"
		var n int64
		if err := s.DB.QueryRow(`SELECT count(*) FROM read_parquet(` + src + `) WHERE ` + bad).Scan(&n); err != nil {
			return fmt.Errorf("repair %s: %w", g.path, err)
		}
		if n == 0 {
			continue
		}
		tmp := g.path + ".repair"
		q := fmt.Sprintf(`COPY (SELECT * REPLACE (
			CASE WHEN bytes > %[1]d THEN 0::UBIGINT ELSE bytes END AS bytes,
			CASE WHEN wire > %[1]d THEN 0::UBIGINT ELSE wire END AS wire,
			CASE WHEN pkts > %[1]d THEN 0::UBIGINT ELSE pkts END AS pkts)
			FROM read_parquet(%s) ORDER BY ts, client) TO '%s' (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)`,
			wrapLimit, src, sqlPath(tmp))
		if _, err := s.DB.Exec(q); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("repair %s: %w", g.path, err)
		}
		if err := os.Rename(tmp, g.path); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("repair %s: %w", g.path, err)
		}
		fixed += n
	}
	if fixed > 0 {
		log.Printf("store: repaired %d records with impossible counters", fixed)
	}
	return s.SetMeta(key, "1")
}
