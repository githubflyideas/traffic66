package store

import (
	"fmt"
	"sort"
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

var ringDims = map[string]string{
	"server":  "server",
	"client":  "client",
	"service": portProtoExpr + " || chr(9) || app",
}

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
	sq := fmt.Sprintf(`WITH p AS (SELECT %s AS i, %s AS o, sum(wire) AS w FROM %s WHERE %s AND %s GROUP BY 1,2),
		ti AS (SELECT i FROM p GROUP BY i ORDER BY sum(w) DESC LIMIT %d)
		SELECT CASE WHEN i IN (SELECT i FROM ti) THEN i ELSE '__other__' END, CASE WHEN i IN (SELECT i FROM ti) THEN o ELSE '__other__' END, CAST(sum(w) AS DOUBLE)
		FROM p GROUP BY 1,2`, ie, oe, s.Source(q.From, q.To), where, cond, nInner)
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
		if err := rows.Scan(&r.Parent, &r.Key, &r.Wire); err != nil {
			return nil, err
		}
		by[r.Parent] = append(by[r.Parent], r)
		tot[r.Parent] += r.Wire
		rs.Total += r.Wire
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
		var rest float64
		for i, o := range os {
			if i < nOuter && o.Key != "__other__" {
				rs.Outer = append(rs.Outer, o)
			} else {
				rest += o.Wire
			}
		}
		if rest > 0 {
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
