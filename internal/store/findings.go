package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Finding is one thing the detection rules found: who (Src) did what (Kind)
// to whom (Dst, Port), from First to Last, with the numbers that show it.
type Finding struct {
	ID     int64          `json:"id"`
	Kind   string         `json:"kind"`
	Sev    int            `json:"sev"` // 1 low, 2 medium, 3 high
	Src    string         `json:"src"`
	Dst    string         `json:"dst"`
	Port   string         `json:"port"`
	First  time.Time      `json:"first"`
	Last   time.Time      `json:"last"`
	Hits   int            `json:"hits"` // detection runs that saw it
	Ev     map[string]any `json:"ev"`
	Status string         `json:"status"` // open, done, false
}

// Finding statuses.
const (
	FindingOpen  = "open"
	FindingDone  = "done"  // dealt with; a recurrence opens a new finding
	FindingFalse = "false" // not a problem; never reported again
)

// SaveFinding records a finding. An open finding with the same kind, source,
// target and port whose last sighting is less than gap before f.First is
// extended instead of opening a new one, so a scan that goes on for an hour is
// one finding. A finding marked as false is never reported again.
func (s *Store) SaveFinding(f Finding, gap time.Duration) (id int64, created bool, err error) {
	var n int
	if err = s.DB.QueryRow(`SELECT count(*) FROM findings WHERE kind = ? AND src = ? AND dst = ? AND port = ? AND status = ?`,
		f.Kind, f.Src, f.Dst, f.Port, FindingFalse).Scan(&n); err != nil {
		return 0, false, err
	}
	if n > 0 {
		return 0, false, nil
	}
	var (
		first, last time.Time
		hits, sev   int
		evs         string
	)
	err = s.DB.QueryRow(`SELECT id, first_ts, last_ts, hits, sev, ev FROM findings
		WHERE kind = ? AND src = ? AND dst = ? AND port = ? AND status = ? AND last_ts >= ?
		ORDER BY last_ts DESC LIMIT 1`, f.Kind, f.Src, f.Dst, f.Port, FindingOpen, f.First.UTC().Add(-gap)).Scan(&id, &first, &last, &hits, &sev, &evs)
	now := time.Now().UTC()
	if errors.Is(err, sql.ErrNoRows) {
		b, _ := json.Marshal(f.Ev)
		err = s.DB.QueryRow(`INSERT INTO findings VALUES (nextval('finding_seq'), ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?) RETURNING id`,
			f.Kind, f.Sev, f.Src, f.Dst, f.Port, f.First.UTC(), f.Last.UTC(), string(b), FindingOpen, now).Scan(&id)
		return id, err == nil, err
	}
	if err != nil {
		return 0, false, err
	}
	var old map[string]any
	json.Unmarshal([]byte(evs), &old)
	b, _ := json.Marshal(MergeEvidence(old, f.Ev))
	if f.First.Before(first) {
		first = f.First
	}
	if f.Last.After(last) {
		last = f.Last
	}
	_, err = s.DB.Exec(`UPDATE findings SET first_ts = ?, last_ts = ?, hits = ?, sev = ?, ev = ?, updated = ? WHERE id = ?`,
		first.UTC(), last.UTC(), hits+1, max(sev, f.Sev), string(b), now, id)
	return id, false, err
}

// MergeEvidence combines two sightings of one finding: numbers keep the
// larger value, lists are joined (at most 12 entries), anything else takes
// the newer value.
func MergeEvidence(old, nw map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range old {
		out[k] = v
	}
	for k, v := range nw {
		switch x := v.(type) {
		case float64, int, int64, uint64:
			a, b := num(out[k]), num(x)
			if b > a {
				out[k] = b
			} else {
				out[k] = a
			}
		case []string:
			out[k] = union(out[k], x)
		case []any:
			var ss []string
			for _, e := range x {
				ss = append(ss, fmt.Sprint(e))
			}
			out[k] = union(out[k], ss)
		default:
			out[k] = v
		}
	}
	return out
}

func num(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	}
	return 0
}

func union(old any, add []string) []string {
	seen := map[string]bool{}
	var out []string
	put := func(s string) {
		if !seen[s] && len(out) < 12 {
			seen[s] = true
			out = append(out, s)
		}
	}
	switch x := old.(type) {
	case []string:
		for _, s := range x {
			put(s)
		}
	case []any:
		for _, s := range x {
			put(fmt.Sprint(s))
		}
	}
	for _, s := range add {
		put(s)
	}
	sort.Strings(out)
	return out
}

// FindingQuery selects findings seen in [From, To).
type FindingQuery struct {
	From, To time.Time
	Status   string // "" or "all": any; otherwise that status
	IP       string // involving this address
	Limit    int
}

// Findings returns findings, most severe and most recent first.
func (s *Store) Findings(q FindingQuery) ([]Finding, error) {
	conds := []string{"last_ts >= ?", "first_ts < ?"}
	args := []any{q.From.UTC(), q.To.UTC()}
	if q.Status != "" && q.Status != "all" {
		conds = append(conds, "status = ?")
		args = append(args, q.Status)
	}
	if q.IP != "" {
		conds = append(conds, `(src = ? OR dst = ? OR contains(ev, ?))`)
		args = append(args, q.IP, q.IP, `"`+q.IP+`"`)
	}
	if q.Limit <= 0 || q.Limit > 1000 {
		q.Limit = 500
	}
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT id, kind, sev, src, dst, port, first_ts, last_ts, hits, ev, status FROM findings
		WHERE %s ORDER BY status = 'open' DESC, sev DESC, last_ts DESC LIMIT %d`, strings.Join(conds, " AND "), q.Limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Finding{}
	for rows.Next() {
		var f Finding
		var ev string
		if err := rows.Scan(&f.ID, &f.Kind, &f.Sev, &f.Src, &f.Dst, &f.Port, &f.First, &f.Last, &f.Hits, &ev, &f.Status); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ev), &f.Ev)
		out = append(out, f)
	}
	return out, rows.Err()
}

// SetFindingStatus marks findings as open, done or false.
func (s *Store) SetFindingStatus(ids []int64, status string) error {
	if status != FindingOpen && status != FindingDone && status != FindingFalse {
		return fmt.Errorf("unknown status %q", status)
	}
	for _, id := range ids {
		if _, err := s.DB.Exec(`UPDATE findings SET status = ?, updated = ? WHERE id = ?`, status, time.Now().UTC(), id); err != nil {
			return err
		}
	}
	return nil
}

// Meta reads a value from the meta table ("" when absent).
func (s *Store) Meta(k string) string {
	var v string
	s.DB.QueryRow(`SELECT v FROM meta WHERE k = ?`, k).Scan(&v)
	return v
}

// SetMeta stores a value in the meta table.
func (s *Store) SetMeta(k, v string) error {
	_, err := s.DB.Exec(`INSERT INTO meta VALUES (?, ?) ON CONFLICT (k) DO UPDATE SET v = excluded.v`, k, v)
	return err
}

// OpenFindings counts open findings of at least severity sev seen since.
func (s *Store) OpenFindings(sev int, since time.Time) int {
	var n int
	s.DB.QueryRow(`SELECT count(*) FROM findings WHERE status = ? AND sev >= ? AND last_ts >= ?`, FindingOpen, sev, since.UTC()).Scan(&n)
	return n
}
