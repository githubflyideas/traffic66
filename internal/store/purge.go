package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// PurgeSize is what deleting the data before a time would remove.
type PurgeSize struct {
	Files int   `json:"files"` // hourly flow files
	Bytes int64 `json:"bytes"` // their size on disk
	Rows  int64 `json:"rows"`  // flow records, in files and not yet in files
}

// PurgeSizeBefore estimates PurgeBefore(before); a zero time means all data.
func (s *Store) PurgeSizeBefore(before time.Time) PurgeSize {
	var p PurgeSize
	s.segMu.RLock()
	for _, g := range s.segCache {
		if before.IsZero() || g.max.Before(before) {
			p.Files++
			if fi, err := os.Stat(g.path); err == nil {
				p.Bytes += fi.Size()
			}
		}
	}
	s.segMu.RUnlock()
	var n sql.NullInt64
	if before.IsZero() {
		s.DB.QueryRow(`SELECT coalesce(sum(rows),0) FROM segments`).Scan(&n)
	} else {
		s.DB.QueryRow(`SELECT coalesce(sum(rows),0) FROM segments WHERE max_ts < ?`, before.UTC()).Scan(&n)
		// files reaching past the time lose only their older records
		// (their size shrinks in proportion)
		s.segMu.RLock()
		var cut []segment
		for _, g := range s.segCache {
			if g.min.Before(before) && !g.max.Before(before) {
				cut = append(cut, g)
			}
		}
		s.segMu.RUnlock()
		for _, g := range cut {
			var m, all int64
			s.DB.QueryRow(`SELECT count(*) FILTER (WHERE ts < ?), count(*) FROM read_parquet('`+sqlPath(g.path)+`')`, before.UTC()).Scan(&m, &all)
			n.Int64 += m
			if fi, err := os.Stat(g.path); err == nil && all > 0 {
				p.Bytes += fi.Size() * m / all
			}
		}
	}
	p.Rows = n.Int64
	var h int64
	if before.IsZero() {
		s.DB.QueryRow(`SELECT count(*) FROM hot`).Scan(&h)
	} else {
		s.DB.QueryRow(`SELECT count(*) FROM hot WHERE ts < ?`, before.UTC()).Scan(&h)
	}
	p.Rows += h
	return p
}

// PurgeBefore deletes everything recorded before a time: flow records,
// hourly and daily summaries, interface counters and findings last seen
// before it. A zero time deletes all data, including what the detection
// rules have learned. Flow files entirely before the time are deleted; a
// file that reaches past it is rewritten without the older records.
func (s *Store) PurgeBefore(before time.Time) (PurgeSize, error) {
	all := before.IsZero()
	var done PurgeSize
	s.segMu.RLock()
	var old []segment
	for _, g := range s.segCache {
		if all || g.max.Before(before) {
			old = append(old, g)
		}
	}
	s.segMu.RUnlock()
	for _, g := range old {
		fi, _ := os.Stat(g.path)
		var rows int64
		s.DB.QueryRow(`SELECT rows FROM segments WHERE path = ?`, g.path).Scan(&rows)
		if err := os.Remove(g.path); err != nil && !os.IsNotExist(err) {
			log.Printf("store: purge %s: %v", g.path, err)
			continue
		}
		s.DB.Exec(`DELETE FROM segments WHERE path = ?`, g.path)
		os.Remove(filepath.Dir(g.path)) // the day's directory, once empty
		done.Files++
		done.Rows += rows
		if fi != nil {
			done.Bytes += fi.Size()
		}
	}
	if !all {
		s.segMu.RLock()
		var cut []segment
		for _, g := range s.segCache {
			if g.min.Before(before) && !g.max.Before(before) {
				cut = append(cut, g)
			}
		}
		s.segMu.RUnlock()
		for _, g := range cut {
			n, err := s.trimSegment(g, before)
			if err != nil {
				log.Printf("store: purge %s: %v", g.path, err)
				continue
			}
			done.Rows += n
		}
	}
	if err := s.loadSegments(); err != nil {
		return done, err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.appHot.Flush(); err != nil {
		return done, err
	}
	cond, args := "ts < ?", []any{before.UTC()}
	if all {
		cond, args = "true", nil
	}
	if r, err := s.DB.Exec(`DELETE FROM hot WHERE `+cond, args...); err == nil {
		n, _ := r.RowsAffected()
		done.Rows += n
	} else {
		return done, err
	}
	for _, t := range []string{"r_ts", "r_host", "r_dim", "ifc"} {
		if _, err := s.DB.Exec(`DELETE FROM `+t+` WHERE `+cond, args...); err != nil {
			return done, err
		}
	}
	if all {
		for _, q := range []string{`DELETE FROM findings`, `DELETE FROM seen`, `DELETE FROM meta WHERE k = 'detect_last'`} {
			if _, err := s.DB.Exec(q); err != nil {
				return done, err
			}
		}
	} else if _, err := s.DB.Exec(`DELETE FROM findings WHERE last_ts < ?`, before.UTC()); err != nil {
		return done, err
	}
	s.DB.Exec(`CHECKPOINT`)
	log.Printf("store: purged %d files, %d flow records (before %v)", done.Files, done.Rows, before)
	return done, nil
}

// trimSegment rewrites a flow file without its records before a time and
// returns how many it dropped.
func (s *Store) trimSegment(g segment, before time.Time) (int64, error) {
	var total, keep int64
	var mn sql.NullTime
	src := "'" + sqlPath(g.path) + "'"
	if err := s.DB.QueryRow(`SELECT count(*), count(*) FILTER (WHERE ts >= ?), min(ts) FILTER (WHERE ts >= ?) FROM read_parquet(`+src+`)`, before.UTC(), before.UTC()).Scan(&total, &keep, &mn); err != nil {
		return 0, err
	}
	tmp := g.path + ".trim"
	q := fmt.Sprintf(`COPY (SELECT * FROM read_parquet(%s) WHERE ts >= ? ORDER BY ts, client) TO '%s' (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)`, src, sqlPath(tmp))
	if _, err := s.DB.Exec(q, before.UTC()); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	if err := os.Rename(tmp, g.path); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	if _, err := s.DB.Exec(`UPDATE segments SET min_ts = ?, rows = ? WHERE path = ?`, mn.Time, keep, g.path); err != nil {
		return 0, err
	}
	return total - keep, nil
}
