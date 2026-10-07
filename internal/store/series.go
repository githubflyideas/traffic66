package store

import (
	"fmt"
	"sort"
	"time"
)

// Series dimensions: what one chart line stands for. key is the SQL
// expression of the value, cond limits the rows (true for all). rollup, when
// set, answers long ranges from the hourly summaries: the query selects
// (bucket, key, wire, pkts) for the given bucket expression and range.
type seriesDim struct {
	key, cond string
	distinct  string // expression counted for the "how many" number ("" = key)
	rollup    func(bucket string) string
	hourly    bool // the rollup has one row per hour: no finer buckets
}

// peerExpr is the external side of a conversation: the server, or the
// client of inbound traffic.
const peerExpr = "CASE WHEN dir = 2 THEN client ELSE server END"

var seriesDims = map[string]seriesDim{
	"app": {key: "app", cond: "true",
		rollup: func(b string) string {
			return `SELECT ` + b + `, app, sum(wire), sum(pkts) FROM r_ts WHERE ts >= ? AND ts < ? GROUP BY 1,2`
		}},
	"client": {key: "client", cond: "true", hourly: true,
		rollup: func(b string) string {
			return `SELECT ` + b + `, ip, sum(wire), sum(pkts) FROM r_host WHERE ts >= ? AND ts < ? AND role = 1 GROUP BY 1,2`
		}},
	"server": {key: "server", cond: "true", hourly: true,
		rollup: func(b string) string {
			return `SELECT ` + b + `, ip, sum(wire), sum(pkts) FROM r_host WHERE ts >= ? AND ts < ? AND role = 2 GROUP BY 1,2`
		}},
	"service": {key: portProtoExpr + " || chr(9) || app", cond: "svc_port <> 0", hourly: true,
		rollup: func(b string) string {
			return `SELECT ` + b + `, val, sum(wire), sum(pkts) FROM r_dim WHERE ts >= ? AND ts < ? AND dim = 'port' AND NOT starts_with(val, '0/') GROUP BY 1,2`
		}},
	// AS of the sender and of the receiver of each packet, when that side
	// is the external one
	"asn_src": {key: "CAST(peer_asn AS VARCHAR) || chr(9) || peer_org", cond: "peer_asn <> 0 AND src = " + peerExpr},
	"asn_dst": {key: "CAST(peer_asn AS VARCHAR) || chr(9) || peer_org", cond: "peer_asn <> 0 AND dst = " + peerExpr},
	// interfaces of the flow exporters, as "exporter/ifindex"
	"if_in":  {key: "exporter || '/' || in_if", cond: "in_if <> 0"},
	"if_out": {key: "exporter || '/' || out_if", cond: "out_if <> 0"},
}

// SeriesBy returns a measure over time split by a dimension: bits per
// second ("wire") or packets per second ("pkts"), for the top k values; the
// rest is summed as "__other__". Distinct is how many values there were.
func (s *Store) SeriesBy(q Query, by, measure string, k int) (*Series, error) {
	d, ok := seriesDims[by]
	if !ok {
		return nil, fmt.Errorf("unknown series %q", by)
	}
	if measure != "wire" && measure != "pkts" {
		return nil, fmt.Errorf("unknown measure %q", measure)
	}
	if k <= 0 || k > 20 {
		k = 8
	}
	// decide before the start is rounded down to a bucket: rounding makes a
	// 6-hour range a little longer, which must not switch it to the summaries
	rollup := q.usesRollup() && d.rollup != nil
	step := StepFor(q.span())
	if rollup && d.hourly && step < time.Hour {
		// hourly summaries in 5-minute buckets would show one spike an hour
		step = time.Hour
	}
	q.From = q.From.UTC().Truncate(step)
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", int64(step.Seconds()))
	var inner string
	var args []any
	if rollup {
		inner = d.rollup(bucket)
		args = []any{q.From.UTC(), q.To.UTC()}
	} else {
		where, wargs, err := q.where()
		if err != nil {
			return nil, err
		}
		inner = fmt.Sprintf(`SELECT %s, %s, sum(wire), sum(pkts) FROM %s WHERE %s AND %s GROUP BY 1,2`, bucket, d.key, s.Source(q.From, q.To), where, d.cond)
		args = wargs
	}
	m := "w"
	if measure == "pkts" {
		m = "p"
	}
	// keep the top k values apart, everything else as one series, and count
	// the values
	sqlq := fmt.Sprintf(`WITH b AS (%s), b2 AS (SELECT * FROM b AS x(t, k, w, p)),
			top AS (SELECT k FROM b2 GROUP BY k ORDER BY sum(%s) DESC LIMIT %d)
		SELECT t, CASE WHEN k IN (SELECT k FROM top) THEN k ELSE '__other__' END AS kk, CAST(sum(%s) AS DOUBLE), (SELECT count(DISTINCT k) FROM b2)
		FROM b2 GROUP BY 1,2`, inner, m, k, m)
	rows, err := s.DB.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type pt struct {
		t time.Time
		k string
		v float64
	}
	var pts []pt
	tot := map[string]float64{}
	distinct := 0
	for rows.Next() {
		var p pt
		var v float64
		if err := rows.Scan(&p.t, &p.k, &v, &distinct); err != nil {
			return nil, err
		}
		p.v = v
		pts = append(pts, p)
		tot[p.k] += v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tot))
	for n := range tot {
		if n != "__other__" {
			names = append(names, n)
		}
	}
	sort.Slice(names, func(i, j int) bool { return tot[names[i]] > tot[names[j]] })
	if _, ok := tot["__other__"]; ok {
		names = append(names, "__other__")
	}
	idx := map[string]int{}
	for i, n := range names {
		idx[n] = i
	}
	start := q.From
	n := bucketCount(start, q.To, step)
	se := &Series{Step: int64(step.Seconds()), Names: names, Values: make([][]float64, len(names)), Distinct: distinct}
	se.Totals = make([]float64, len(names))
	for i, nm := range names {
		se.Totals[i] = tot[nm]
	}
	for i := 0; i < n; i++ {
		se.Times = append(se.Times, start.Add(time.Duration(i)*step).UnixMilli())
	}
	for i := range se.Values {
		se.Values[i] = make([]float64, n)
	}
	f := 8.0 // bytes to bits
	if measure == "pkts" {
		f = 1
	}
	for _, p := range pts {
		ti := int(p.t.UTC().Sub(start) / step)
		if ti < 0 || ti >= n {
			continue
		}
		secs := step.Seconds()
		if end := start.Add(time.Duration(ti+1) * step); end.After(q.To) {
			secs = max(1, q.To.Sub(start.Add(time.Duration(ti)*step)).Seconds())
		}
		se.Values[idx[p.k]][ti] += p.v * f / secs
	}
	return se, nil
}
