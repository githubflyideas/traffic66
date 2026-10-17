package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IfacePurge is what deleting one interface's data would remove, or did.
//
// Bytes is what the flow files lose. The database file is not counted:
// most of it is the summaries, which stay, so its share of one interface
// cannot be worked out — PurgeIface reports what was actually freed.
//
// Rows covers the flow records: those are the only thing stored per
// interface. The hourly and daily summaries (and the findings) record
// traffic by application, host and dimension, not by interface, so an
// interface cannot be subtracted from them; Summaries says whether any are
// held for the range, so the page can say so rather than imply the
// interface is gone from everything.
type IfacePurge struct {
	Exporter  string `json:"exporter"`
	IfIndex   uint32 `json:"ifindex"`
	Files     int    `json:"files"` // hourly flow files touched
	Bytes     int64  `json:"bytes"`
	Rows      int64  `json:"rows"`
	Counters  int64  `json:"counters"`  // interface counter samples
	Summaries int64  `json:"summaries"` // summary rows that stay, by hour
}

// ifaceWhere matches the flow records that went in or out of one interface
// of one exporter, optionally only those older than a time.
//
// A flow record names the interface it arrived on and the one it left by,
// so a record matched here is also part of the other interface's traffic:
// deleting it takes those flows out of that interface's view too. There is
// no way around that at the record level, and blanking the interface
// instead would leave traffic that can no longer be attributed, so the
// record goes and the pages say as much.
func ifaceWhere(exporter string, ifindex uint32, before time.Time) (string, []any) {
	w := "exporter = ? AND (in_if = ? OR out_if = ?)"
	args := []any{exporter, ifindex, ifindex}
	if !before.IsZero() {
		w += " AND ts < ?"
		args = append(args, before.UTC())
	}
	return w, args
}

// PurgeSizeIface estimates PurgeIface. A zero time means the interface's
// whole history.
func (s *Store) PurgeSizeIface(exporter string, ifindex uint32, before time.Time) IfacePurge {
	p := IfacePurge{Exporter: exporter, IfIndex: ifindex}
	where, args := ifaceWhere(exporter, ifindex, before)

	var hot int64
	s.DB.QueryRow(`SELECT count(*) FROM hot WHERE `+where, args...).Scan(&hot)
	p.Rows = hot

	for _, c := range s.segmentCounts(where, args) {
		if c.match == 0 {
			continue
		}
		p.Files++
		p.Rows += c.match
		if fi, err := os.Stat(c.path); err == nil && c.total > 0 {
			p.Bytes += fi.Size() * c.match / c.total
		}
	}
	cw, cargs := "exporter = ? AND ifindex = ?", []any{exporter, ifindex}
	if !before.IsZero() {
		cw += " AND ts < ?"
		cargs = append(cargs, before.UTC())
	}
	s.DB.QueryRow(`SELECT count(*) FROM ifc WHERE `+cw, cargs...).Scan(&p.Counters)

	sw, sargs := "true", []any{}
	if !before.IsZero() {
		sw, sargs = "ts < ?", []any{before.UTC()}
	}
	s.DB.QueryRow(`SELECT count(*) FROM r_ts WHERE `+sw, sargs...).Scan(&p.Summaries)
	return p
}

// PurgeIface deletes one interface's flow records and counter samples,
// either all of them or only those older than a time. Flow files that hold
// nothing else are deleted; the rest are rewritten without those records.
//
// The summaries are left alone: see IfacePurge. This is deliberate — an
// interface's share of an hourly total is not recorded, so removing it
// would mean either dropping other interfaces' traffic with it or
// inventing a number.
func (s *Store) PurgeIface(exporter string, ifindex uint32, before time.Time) (IfacePurge, error) {
	done := IfacePurge{Exporter: exporter, IfIndex: ifindex}
	if exporter == "" || ifindex == 0 {
		return done, fmt.Errorf("store: no interface given")
	}
	where, args := ifaceWhere(exporter, ifindex, before)

	for _, c := range s.segmentCounts(where, args) {
		if c.match == 0 {
			continue
		}
		fi, _ := os.Stat(c.path)
		size := int64(0)
		if fi != nil {
			size = fi.Size()
		}
		if c.match >= c.total {
			if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
				log.Printf("store: purge %s: %v", c.path, err)
				continue
			}
			s.DB.Exec(`DELETE FROM segments WHERE path = ?`, c.path)
			os.Remove(filepath.Dir(c.path)) // the day's directory, once empty
			done.Files++
			done.Rows += c.total
			done.Bytes += size
			continue
		}
		n, err := s.rewriteSegment(c.path, where, args)
		if err != nil {
			log.Printf("store: purge %s: %v", c.path, err)
			continue
		}
		done.Files++
		done.Rows += n
		if fi2, err := os.Stat(c.path); err == nil {
			done.Bytes += max(0, size-fi2.Size())
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
	if err := s.appIfc.Flush(); err != nil {
		return done, err
	}
	if r, err := s.DB.Exec(`DELETE FROM hot WHERE `+where, args...); err == nil {
		n, _ := r.RowsAffected()
		done.Rows += n
	} else {
		return done, err
	}
	cw, cargs := "exporter = ? AND ifindex = ?", []any{exporter, ifindex}
	if !before.IsZero() {
		cw += " AND ts < ?"
		cargs = append(cargs, before.UTC())
	}
	if r, err := s.DB.Exec(`DELETE FROM ifc WHERE `+cw, cargs...); err == nil {
		n, _ := r.RowsAffected()
		done.Counters += n
	} else {
		return done, err
	}
	s.DB.Exec(`CHECKPOINT`)
	log.Printf("store: purged %d flow records and %d counter samples of %s/%d", done.Rows, done.Counters, exporter, ifindex)
	return done, nil
}

// segCount is how much of one flow file a condition matches.
type segCount struct {
	path         string
	match, total int64
}

// segmentCounts counts the rows matching a condition in every flow file,
// in one pass. A file DuckDB reports under a path the segment table does
// not know is ignored rather than guessed at.
func (s *Store) segmentCounts(where string, args []any) []segCount {
	s.segMu.RLock()
	files := make([]string, 0, len(s.segCache))
	known := make(map[string]string, len(s.segCache))
	for _, g := range s.segCache {
		files = append(files, "'"+sqlPath(g.path)+"'")
		known[sqlPath(g.path)] = g.path
	}
	s.segMu.RUnlock()
	if len(files) == 0 {
		return nil
	}
	q := fmt.Sprintf(`SELECT filename, count(*) FILTER (WHERE %s), count(*)
		FROM read_parquet([%s], filename = true) GROUP BY 1`, where, strings.Join(files, ","))
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		log.Printf("store: counting flow files: %v", err)
		return nil
	}
	defer rows.Close()
	var out []segCount
	for rows.Next() {
		var name string
		var c segCount
		if err := rows.Scan(&name, &c.match, &c.total); err != nil {
			log.Printf("store: counting flow files: %v", err)
			return out
		}
		p, ok := known[sqlPath(name)]
		if !ok {
			continue
		}
		c.path = p
		out = append(out, c)
	}
	return out
}

// rewriteSegment rewrites a flow file without the rows the drop condition
// matches and returns how many it dropped. The file's recorded range and
// row count are updated, because dropping rows from the middle of a file
// can move either end.
func (s *Store) rewriteSegment(path, drop string, args []any) (int64, error) {
	var total, kept int64
	var mn, mx sql.NullTime
	src := "'" + sqlPath(path) + "'"
	// the condition is evaluated once, in the subquery, so its parameters
	// are bound once however many aggregates read the result
	q := fmt.Sprintf(`SELECT count(*), count(*) FILTER (WHERE keep), min(ts) FILTER (WHERE keep), max(ts) FILTER (WHERE keep)
		FROM (SELECT ts, NOT (%s) AS keep FROM read_parquet(%s))`, drop, src)
	if err := s.DB.QueryRow(q, args...).Scan(&total, &kept, &mn, &mx); err != nil {
		return 0, err
	}
	tmp := path + ".trim"
	cp := fmt.Sprintf(`COPY (SELECT * FROM read_parquet(%s) WHERE NOT (%s) ORDER BY ts, client) TO '%s' (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)`,
		src, drop, sqlPath(tmp))
	if _, err := s.DB.Exec(cp, args...); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	if _, err := s.DB.Exec(`UPDATE segments SET min_ts = ?, max_ts = ?, rows = ? WHERE path = ?`, mn.Time, mx.Time, kept, path); err != nil {
		return 0, err
	}
	return total - kept, nil
}
