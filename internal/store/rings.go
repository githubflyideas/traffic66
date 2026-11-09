package store

import (
	"fmt"
	"sort"
	"strings"
)

// Ring is one segment of a two-ring chart: an inner value, or an outer value
// belonging to an inner one.
type Ring struct {
	Parent string  `json:"p,omitempty"` // inner key of an outer segment
	Key    string  `json:"k"`
	Wire   float64 `json:"v"`
}

// RingSet is the data of a two-ring (sunburst) chart.
type RingSet struct {
	Inner []Ring  `json:"inner"`
	Outer []Ring  `json:"outer"`
	Total float64 `json:"total"`
}

// ringDims are the dimensions of the rings, each as a number and a text
// for the IPv4 rows and for the others, and how the two are shown. An
// address is its number where it has one: reading the address text of
// every row takes longer than everything else the query reads.
var ringDims = map[string]struct{ num, text, num6, text6, show string }{
	"server":  {"srv4", "NULL::VARCHAR", "0::UINTEGER", "server", addrText},
	"client":  {"cli4", "NULL::VARCHAR", "0::UINTEGER", "client", addrText},
	"service": {"svc_port::UINTEGER * 256 + proto", "app", "svc_port::UINTEGER * 256 + proto", "app", serviceText},
}

// shown from {n} (the number) and {s} (the text) of a ring value
const (
	addrText    = "CASE WHEN {n} <> 0 THEN concat({n} >> 24, '.', ({n} >> 16) & 255, '.', ({n} >> 8) & 255, '.', {n} & 255) ELSE {s} END"
	serviceText = "concat({n} >> 8, '/', CASE {n} & 255 WHEN 6 THEN 'tcp' WHEN 17 THEN 'udp' ELSE CAST({n} & 255 AS VARCHAR) END, chr(9), {s})"
)

// Rings splits the traffic by an inner dimension (its top nInner values,
// the rest as "__other__") and each inner value by an outer dimension (its
// top nOuter values, the rest as "__other__").
func (s *Store) Rings(q Query, inner, outer string, nInner, nOuter int) (*RingSet, error) {
	ie, ok1 := ringDims[inner]
	oe, ok2 := ringDims[outer]
	if !ok1 || !ok2 || inner == outer {
		return nil, fmt.Errorf("unknown ring pair %s/%s", inner, outer)
	}
	cond := "true"
	if inner == "service" || outer == "service" {
		cond = "svc_port <> 0"
	}
	where, args, err := q.where()
	if err != nil {
		return nil, err
	}
	// The rows are read again by each step rather than kept, and the
	// pairs are added up only for the top inner values: adding up every
	// pair first (each of a week's clients with each server it reached)
	// is what ran out of memory. The database keeps each inner value's
	// top outer values and their total; the rest is the difference. A
	// flow's two ends are the same family, so cli4 <> 0 tells the IPv4
	// rows for either end.
	src := s.Source(q.From, q.To)
	show := func(d struct{ num, text, num6, text6, show string }, n, t string) string {
		return strings.NewReplacer("{n}", n, "{s}", t).Replace(d.show)
	}
	sq := fmt.Sprintf(`WITH x AS NOT MATERIALIZED (
			SELECT %[1]s AS in_, %[2]s AS is_, %[3]s AS on_, %[4]s AS os_, wire AS w FROM %[5]s WHERE (%[6]s) AND %[7]s AND cli4 <> 0
			UNION ALL SELECT %[8]s, %[9]s, %[10]s, %[11]s, wire FROM %[5]s WHERE (%[6]s) AND %[7]s AND cli4 = 0),
		ti AS (SELECT in_, is_, true AS hit FROM x GROUP BY ALL ORDER BY sum(w) DESC LIMIT %[12]d),
		p AS (SELECT CASE WHEN ti.hit THEN x.in_ END AS pn, CASE WHEN ti.hit THEN x.is_ END AS ps,
				CASE WHEN ti.hit THEN x.on_ END AS qn, CASE WHEN ti.hit THEN x.os_ END AS qs, sum(x.w) AS w
			FROM x LEFT JOIN ti ON x.in_ = ti.in_ AND coalesce(x.is_, '') = coalesce(ti.is_, '') GROUP BY ALL)
		SELECT CASE WHEN pn IS NULL THEN '__other__' ELSE %[13]s END, CASE WHEN qn IS NULL THEN '__other__' ELSE %[14]s END,
			CAST(w AS DOUBLE), CAST(sum(w) OVER (PARTITION BY pn, ps) AS DOUBLE) FROM p
		QUALIFY row_number() OVER (PARTITION BY pn, ps ORDER BY w DESC) <= %[15]d`,
		ie.num, ie.text, oe.num, oe.text, src, where, cond, ie.num6, ie.text6, oe.num6, oe.text6, nInner,
		show(ie, "pn", "ps"), show(oe, "qn", "qs"), nOuter+1)
	args = append(append([]any{}, args...), args...)
	rows, err := s.DB.Query(sq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	by := map[string][]Ring{}
	tot := map[string]float64{}
	rs := &RingSet{Inner: []Ring{}, Outer: []Ring{}}
	for rows.Next() {
		var r Ring
		var all float64
		if err := rows.Scan(&r.Parent, &r.Key, &r.Wire, &all); err != nil {
			return nil, err
		}
		by[r.Parent] = append(by[r.Parent], r)
		if _, seen := tot[r.Parent]; !seen {
			tot[r.Parent] = all
			rs.Total += all
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(tot))
	for k := range tot {
		keys = append(keys, k)
	}
	// biggest first, Other last
	sort.Slice(keys, func(a, b int) bool {
		if (keys[a] == "__other__") != (keys[b] == "__other__") {
			return keys[b] == "__other__"
		}
		return tot[keys[a]] > tot[keys[b]]
	})
	for _, k := range keys {
		rs.Inner = append(rs.Inner, Ring{Key: k, Wire: tot[k]})
		os := by[k]
		sort.Slice(os, func(a, b int) bool { return os[a].Wire > os[b].Wire })
		// what is not among the outer values kept goes to Other: those
		// the database left out and an outer value called Other itself
		rest := tot[k]
		for i, o := range os {
			if i < nOuter && o.Key != "__other__" {
				rs.Outer = append(rs.Outer, o)
				rest -= o.Wire
			}
		}
		if rest > 0.5 {
			rs.Outer = append(rs.Outer, Ring{Parent: k, Key: "__other__", Wire: rest})
		}
	}
	return rs, nil
}

// SegmentCountries is traffic between internal segments and remote
// countries (both directions), for lines on the world map.
type SegmentCountry struct {
	Segment string  `json:"seg"`
	Country string  `json:"cc"`
	Wire    float64 `json:"v"`
}

func (s *Store) SegmentCountries(q Query) ([]SegmentCountry, error) {
	where, args, err := q.where()
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT segment, peer_cc, CAST(sum(wire) AS DOUBLE) FROM %s WHERE %s AND dir IN (1, 2) AND peer_cc <> '' GROUP BY 1,2 ORDER BY 3 DESC LIMIT 2000`,
		s.Source(q.From, q.To), where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SegmentCountry{}
	for rows.Next() {
		var x SegmentCountry
		if err := rows.Scan(&x.Segment, &x.Country, &x.Wire); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
