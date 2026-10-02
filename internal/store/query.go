package store

import (
	"database/sql"
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Filter is one condition: Field equals (or, with Neg, does not equal) Value.
type Filter struct {
	Field string `json:"f"`
	Neg   bool   `json:"neg"`
	Value string `json:"v"`
}

// Query selects flow slices in [From, To).
type Query struct {
	From, To   time.Time
	Filters    []Filter
	IncludeDup bool
}

func (q Query) span() time.Duration { return q.To.Sub(q.From) }

// usesRollup reports whether this query can be answered from rollups.
func (q Query) usesRollup() bool { return len(q.Filters) == 0 && q.span() > 6*time.Hour }

var dirNames = map[string]int{"outbound": DirOutbound, "inbound": DirInbound, "internal": DirInternal, "transit": DirTransit}

// DirName is the stored direction's API name.
func DirName(d uint8) string {
	for k, v := range dirNames {
		if v == int(d) {
			return k
		}
	}
	return "unknown"
}

var protoByName = map[string]int{"icmp": 1, "igmp": 2, "ipip": 4, "tcp": 6, "udp": 17, "ipv6": 41, "gre": 47, "esp": 50, "ah": 51, "icmpv6": 58, "ospf": 89, "sctp": 132}

func v4range(p netip.Prefix) (uint32, uint32, bool) {
	if !p.Addr().Is4() {
		return 0, 0, false
	}
	a := p.Masked().Addr().As4()
	lo := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
	hi := lo | (uint32(1)<<(32-p.Bits()) - 1)
	if p.Bits() == 0 {
		hi = 0xFFFFFFFF
	}
	return lo, hi, true
}

// where builds the WHERE clause and its arguments.
func (q Query) where() (string, []any, error) {
	conds := []string{"ts >= ?", "ts < ?"}
	args := []any{q.From.UTC(), q.To.UTC()}
	if !q.IncludeDup {
		conds = append(conds, "NOT dup")
	}
	for _, f := range q.Filters {
		var c string
		v := strings.TrimSpace(f.Value)
		switch f.Field {
		case "ip", "client", "server":
			cols := map[string][]string{"ip": {"client", "server"}, "client": {"client"}, "server": {"server"}}[f.Field]
			n4 := map[string][]string{"ip": {"cli4", "srv4"}, "client": {"cli4"}, "server": {"srv4"}}[f.Field]
			var parts []string
			if p, err := netip.ParsePrefix(v); err == nil && !p.IsSingleIP() {
				lo, hi, ok := v4range(p)
				if !ok {
					return "", nil, fmt.Errorf("IPv6 prefixes are not supported in filters yet: %s", v)
				}
				for _, col := range n4 {
					parts = append(parts, col+" BETWEEN ? AND ?")
					args = append(args, lo, hi)
				}
			} else {
				if a, err := netip.ParseAddr(strings.TrimSuffix(v, "/32")); err == nil {
					v = a.Unmap().String()
				}
				for _, col := range cols {
					parts = append(parts, col+" = ?")
					args = append(args, v)
				}
			}
			c = "(" + strings.Join(parts, " OR ") + ")"
		case "port":
			port := v
			if i := strings.IndexByte(v, '/'); i > 0 {
				port = v[:i]
				pn := protoByName[strings.ToLower(v[i+1:])]
				if pn > 0 {
					c = "(svc_port = ? AND proto = ?)"
					pv, _ := strconv.Atoi(port)
					args = append(args, pv, pn)
					break
				}
			}
			pv, err := strconv.Atoi(port)
			if err != nil {
				return "", nil, fmt.Errorf("bad port %q", v)
			}
			c = "svc_port = ?"
			args = append(args, pv)
		case "app":
			c = "app = ?"
			args = append(args, v)
		case "country":
			if v == "__internal__" {
				c = "dir = 3"
			} else {
				c = "peer_cc = ?"
				args = append(args, strings.ToUpper(v))
			}
		case "asn":
			n, err := strconv.ParseUint(strings.TrimPrefix(strings.ToUpper(v), "AS"), 10, 32)
			if err != nil {
				return "", nil, fmt.Errorf("bad ASN %q", v)
			}
			c = "peer_asn = ?"
			args = append(args, n)
		case "proto":
			n, ok := protoByName[strings.ToLower(v)]
			if !ok {
				x, err := strconv.Atoi(v)
				if err != nil {
					return "", nil, fmt.Errorf("bad protocol %q", v)
				}
				n = x
			}
			c = "proto = ?"
			args = append(args, n)
		case "dir":
			n, ok := dirNames[v]
			if !ok {
				return "", nil, fmt.Errorf("bad direction %q", v)
			}
			c = "dir = ?"
			args = append(args, n)
		case "encap":
			c = "encap = ?"
			n, err := strconv.Atoi(v)
			if err != nil {
				return "", nil, fmt.Errorf("bad encapsulation %q", v)
			}
			args = append(args, n)
		case "exporter":
			c = "exporter = ?"
			args = append(args, v)
		case "segment":
			c = "segment = ?"
			args = append(args, v)
		case "threat":
			c = "threat = ?"
			args = append(args, v)
		case "vlan":
			n, err := strconv.Atoi(v)
			if err != nil {
				return "", nil, fmt.Errorf("bad VLAN %q", v)
			}
			c = "vlan = ?"
			args = append(args, n)
		case "iface":
			// exporter/ifindex, either direction
			i := strings.LastIndexByte(v, '/')
			if i <= 0 {
				return "", nil, fmt.Errorf("bad interface %q", v)
			}
			n, err := strconv.Atoi(v[i+1:])
			if err != nil {
				return "", nil, fmt.Errorf("bad interface %q", v)
			}
			c = "(exporter = ? AND (in_if = ? OR out_if = ?))"
			args = append(args, v[:i], n, n)
		default:
			return "", nil, fmt.Errorf("unknown filter field %q", f.Field)
		}
		if f.Neg {
			c = "NOT " + c
		}
		conds = append(conds, c)
	}
	return strings.Join(conds, " AND "), args, nil
}

// TopRow is one Top-N entry.
type TopRow struct {
	Key   string `json:"key"`
	Key2  string `json:"key2,omitempty"`
	Key3  string `json:"key3,omitempty"`
	Extra string `json:"extra,omitempty"`
	Bytes uint64 `json:"bytes"`
	Wire  uint64 `json:"wire"`
	Pkts  uint64 `json:"pkts"`
	Flows uint64 `json:"flows"`
	Peers uint64 `json:"peers,omitempty"`
	Int   bool   `json:"internal,omitempty"`
}

// dimension definitions: SQL expression(s) and the rollup that serves them.
type dimDef struct {
	expr   string // key expression over flow rows
	expr2  string
	expr3  string
	extra  string // additional grouped column shown next to the key
	rollup string // r_dim dim name, "host:1"/"host:2" for r_host, "" for none
	peers  string // column counted distinct as peers
	intCol string
}

var dims = map[string]dimDef{
	"client":   {expr: "client", rollup: "host:1", peers: "server", intCol: "bool_or(cli_int)"},
	"server":   {expr: "server", extra: "any_value(peer_cc)", rollup: "host:2", peers: "client", intCol: "bool_or(srv_int)"},
	"service":  {expr: "server", expr2: portProtoExpr, extra: "any_value(app)", peers: "client", intCol: "bool_or(srv_int)"},
	"conv":     {expr: "client", expr2: "server", expr3: portProtoExpr, extra: "any_value(peer_cc)"},
	"app":      {expr: "app", rollup: "app"},
	"port":     {expr: portProtoExpr, extra: "any_value(app)", rollup: "port"},
	"country":  {expr: "CASE WHEN dir = 3 THEN '__internal__' ELSE peer_cc END", rollup: "country"},
	"asn":      {expr: "CAST(peer_asn AS VARCHAR)", extra: "any_value(peer_org)", rollup: "asn"},
	"proto":    {expr: "CAST(proto AS VARCHAR)", rollup: "proto"},
	"dir":      {expr: "CAST(dir AS VARCHAR)", rollup: "dir"},
	"exporter": {expr: "exporter", rollup: "exporter"},
	"segment":  {expr: "segment", rollup: "segment"},
	"encap":    {expr: "CAST(encap AS VARCHAR)", rollup: "encap"},
	"threat":   {expr: "threat", expr2: "client", expr3: "server", extra: "concat(any_value(peer_cc), '|', any_value(dir))"},
	"vlan":     {expr: "CAST(vlan AS VARCHAR)"},
}

const portProtoExpr = "concat(svc_port, '/', CASE proto WHEN 6 THEN 'tcp' WHEN 17 THEN 'udp' ELSE CAST(proto AS VARCHAR) END)"

type keyPart struct {
	inner   []string // select expressions in the grouping query
	groups  []string // aliases grouped by
	inner6  []string // same for the IPv6 half (address parts only)
	groups6 []string
	v4cond  string // condition selecting IPv4 rows ("" if not an address)
	outer   string // display expression over the grouped result
}

// keyPartFor maps a dimension expression to fast grouping keys.
func keyPartFor(expr string, i int) keyPart {
	g := "g" + strconv.Itoa(i)
	addr := func(col, num string) keyPart {
		// IPv4 is grouped and printed from its number without reading the
		// address text; IPv6 rows are grouped by text in a separate pass.
		n, t := g+"n", g+"s"
		v4 := "concat(" + n + " >> 24, '.', (" + n + " >> 16) & 255, '.', (" + n + " >> 8) & 255, '.', " + n + " & 255)"
		return keyPart{
			inner:   []string{num + " AS " + n, "NULL::VARCHAR AS " + t},
			groups:  []string{n},
			inner6:  []string{"0::UINTEGER AS " + n, col + " AS " + t},
			groups6: []string{t},
			v4cond:  num + " <> 0",
			outer:   "CASE WHEN " + n + " <> 0 THEN " + v4 + " ELSE " + t + " END",
		}
	}
	switch expr {
	case "client":
		return addr("client", "cli4")
	case "server":
		return addr("server", "srv4")
	case portProtoExpr:
		in := []string{"svc_port AS " + g + "p", "proto AS " + g + "t"}
		gr := []string{g + "p", g + "t"}
		return keyPart{inner: in, groups: gr, inner6: in, groups6: gr,
			outer: "concat(" + g + "p, '/', CASE " + g + "t WHEN 6 THEN 'tcp' WHEN 17 THEN 'udp' ELSE CAST(" + g + "t AS VARCHAR) END)"}
	}
	in := []string{expr + " AS " + g}
	return keyPart{inner: in, groups: []string{g}, inner6: in, groups6: []string{g}, outer: g}
}

// TopN returns the largest groups by wire bytes.
func (s *Store) TopN(q Query, dim string, limit int) ([]TopRow, error) {
	return s.TopNBy(q, dim, limit, "wire")
}

// topOrder maps a ranking measure to the aggregate it orders by, for the
// detail query and the rollup query.
var topOrder = map[string][2]string{
	"wire":  {"x_w", "sum(wire)"},
	"bytes": {"x_b", "sum(bytes)"},
	"pkts":  {"x_p", "sum(pkts)"},
	"flows": {"x_f", "sum(flows)"},
	"peers": {"x_peers", "sum(wire)"}, // rollups hold no peer counts
	// average packet size; smallest first is the interesting end (scans,
	// floods), so callers ask for it ascending
	"avg": {"x_w / greatest(x_p, 1)", "sum(wire) / greatest(sum(pkts), 1)"},
}

// TopNBy returns the groups ranked by one measure: wire, bytes, pkts,
// flows, peers or avg (average wire bytes per packet). asc ranks the
// smallest first.
func (s *Store) TopNBy(q Query, dim string, limit int, by string, asc ...bool) ([]TopRow, error) {
	ord, ok := topOrder[by]
	if !ok {
		return nil, fmt.Errorf("unknown ranking %q", by)
	}
	dir := "DESC"
	if len(asc) > 0 && asc[0] {
		dir = "ASC"
	}
	d, ok := dims[dim]
	if !ok {
		return nil, fmt.Errorf("unknown dimension %q", dim)
	}
	if limit <= 0 || limit > 1000 {
		limit = 66
	}
	if q.usesRollup() && d.rollup != "" {
		return s.topRollup(q, d.rollup, limit, ord[1]+" "+dir)
	}
	where, args, err := q.where()
	if err != nil {
		return nil, err
	}
	// Group by cheap keys (numeric IPv4, port and protocol as integers) and
	// format for display afterwards: grouping on address text is several
	// times slower. IPv4 and IPv6 rows never share a group, so they are
	// aggregated separately and merged.
	var inner, groups, inner6, groups6, outer, v4conds []string
	for i, e := range []string{d.expr, d.expr2, d.expr3} {
		if e == "" {
			outer = append(outer, "'' AS k"+strconv.Itoa(i))
			continue
		}
		kp := keyPartFor(e, i)
		inner = append(inner, kp.inner...)
		groups = append(groups, kp.groups...)
		inner6 = append(inner6, kp.inner6...)
		groups6 = append(groups6, kp.groups6...)
		if kp.v4cond != "" {
			v4conds = append(v4conds, kp.v4cond)
		}
		outer = append(outer, kp.outer+" AS k"+strconv.Itoa(i))
	}
	extra := "''"
	if d.extra != "" {
		extra = "coalesce(CAST(" + d.extra + " AS VARCHAR), '')"
	}
	peers := "0"
	if d.peers != "" && q.span() <= 7*24*time.Hour {
		peers = "approx_count_distinct(" + d.peers + ")"
	}
	intc := "false"
	if d.intCol != "" {
		intc = d.intCol
	}
	cond := where
	if dim == "threat" {
		cond += " AND threat <> ''"
	}
	src := s.Source(q.From, q.To)
	aggs := fmt.Sprintf(`%s AS x_extra, sum(bytes) AS x_b, sum(wire) AS x_w, sum(pkts) AS x_p, sum(flows) AS x_f, %s AS x_peers, %s AS x_int`, extra, peers, intc)
	part := func(sel, grp []string, c string) string {
		return fmt.Sprintf(`(SELECT %s, %s FROM %s WHERE %s GROUP BY %s ORDER BY %s %s, x_w DESC LIMIT %d)`,
			strings.Join(sel, ", "), aggs, src, c, strings.Join(grp, ", "), ord[0], dir, limit)
	}
	body := part(inner, groups, cond)
	qargs := args
	if len(v4conds) > 0 {
		v4 := strings.Join(v4conds, " AND ")
		body = part(inner, groups, "("+cond+") AND "+v4) + " UNION ALL " + part(inner6, groups6, "("+cond+") AND NOT ("+v4+")")
		qargs = append(append([]any{}, args...), args...)
	}
	sqlq := fmt.Sprintf(`SELECT %s, x_extra, x_b, x_w, x_p, x_f, x_peers, x_int FROM (%s) ORDER BY %s %s, x_w DESC LIMIT %d`,
		strings.Join(outer, ", "), body, ord[0], dir, limit)
	rows, err := s.DB.Query(sqlq, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopRow
	for rows.Next() {
		var t TopRow
		var k, k2, k3 sql.NullString
		if err := rows.Scan(&k, &k2, &k3, &t.Extra, &t.Bytes, &t.Wire, &t.Pkts, &t.Flows, &t.Peers, &t.Int); err != nil {
			return nil, err
		}
		t.Key, t.Key2, t.Key3 = k.String, k2.String, k3.String
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) topRollup(q Query, rollup string, limit int, order string) ([]TopRow, error) {
	var sqlq string
	args := []any{q.From.UTC(), q.To.UTC()}
	if strings.HasPrefix(rollup, "host:") {
		role, _ := strconv.Atoi(rollup[5:])
		sqlq = fmt.Sprintf(`SELECT ip, sum(bytes), sum(wire), sum(pkts), sum(flows), bool_or(internal) FROM r_host
			WHERE ts >= ? AND ts < ? AND role = %d GROUP BY 1 ORDER BY %s LIMIT %d`, role, order, limit)
	} else {
		sqlq = fmt.Sprintf(`SELECT val, sum(bytes), sum(wire), sum(pkts), sum(flows), false FROM r_dim
			WHERE ts >= ? AND ts < ? AND dim = ? GROUP BY 1 ORDER BY %s LIMIT %d`, order, limit)
		args = append(args, rollup)
	}
	rows, err := s.DB.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopRow
	for rows.Next() {
		var t TopRow
		if err := rows.Scan(&t.Key, &t.Bytes, &t.Wire, &t.Pkts, &t.Flows, &t.Int); err != nil {
			return nil, err
		}
		if rollup == "port" || rollup == "asn" {
			// value carries "key\textra"
			if i := strings.IndexByte(t.Key, '\t'); i >= 0 {
				t.Key, t.Extra = t.Key[:i], t.Key[i+1:]
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Series is a time series split by one dimension.
type Series struct {
	Step   int64       `json:"step"` // seconds
	Times  []int64     `json:"times"`
	Names  []string    `json:"names"`
	Values [][]float64 `json:"values"` // [series][time] bits per second
}

// StepFor picks a bucket size giving at most ~300 points.
func StepFor(span time.Duration) time.Duration {
	for _, s := range []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 3 * time.Hour, 6 * time.Hour, 24 * time.Hour} {
		if span/s <= 300 {
			return s
		}
	}
	return 24 * time.Hour
}

// SeriesBy returns bits/s over time split by app (top k, rest as "__other__").
func (s *Store) SeriesByApp(q Query, k int) (*Series, error) {
	step := StepFor(q.span())
	q.From = q.From.UTC().Truncate(step)
	var rows *sql.Rows
	var err error
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", int64(step.Seconds()))
	if q.usesRollup() {
		rows, err = s.DB.Query(fmt.Sprintf(`SELECT %s AS b, app, sum(wire) FROM r_ts WHERE ts >= ? AND ts < ? GROUP BY 1,2`, bucket), q.From.UTC(), q.To.UTC())
	} else {
		where, args, werr := q.where()
		if werr != nil {
			return nil, werr
		}
		rows, err = s.DB.Query(fmt.Sprintf(`SELECT %s AS b, app, sum(wire) FROM %s WHERE %s GROUP BY 1,2`, bucket, s.Source(q.From, q.To), where), args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type pt struct {
		t   time.Time
		app string
		v   float64
	}
	var pts []pt
	tot := map[string]float64{}
	for rows.Next() {
		var p pt
		var w uint64
		if err := rows.Scan(&p.t, &p.app, &w); err != nil {
			return nil, err
		}
		p.v = float64(w)
		pts = append(pts, p)
		tot[p.app] += p.v
	}
	names := make([]string, 0, len(tot))
	for n := range tot {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return tot[names[i]] > tot[names[j]] })
	if len(names) > k {
		names = append(names[:k:k], "__other__")
	}
	idx := map[string]int{}
	for i, n := range names {
		idx[n] = i
	}
	start := q.From.UTC().Truncate(step)
	n := int(q.To.Sub(start)/step) + 1
	se := &Series{Step: int64(step.Seconds()), Names: names, Values: make([][]float64, len(names))}
	for i := 0; i < n; i++ {
		se.Times = append(se.Times, start.Add(time.Duration(i)*step).UnixMilli())
	}
	for i := range se.Values {
		se.Values[i] = make([]float64, n)
	}
	for _, p := range pts {
		ti := int(p.t.UTC().Sub(start) / step)
		if ti < 0 || ti >= n {
			continue
		}
		si, ok := idx[p.app]
		if !ok {
			si = idx["__other__"]
		}
		secs := step.Seconds()
		if end := start.Add(time.Duration(ti+1) * step); end.After(q.To) {
			secs = q.To.Sub(start.Add(time.Duration(ti) * step)).Seconds()
			if secs < 1 {
				secs = 1
			}
		}
		se.Values[si][ti] += p.v * 8 / secs
	}
	return se, nil
}

// TotalSeries returns total bits/s over time (one series), used for
// baselines. It shares SeriesByApp's bucketing.
func (s *Store) TotalSeries(q Query) (*Series, error) {
	se, err := s.SeriesByApp(q, 1<<30)
	if err != nil {
		return nil, err
	}
	tot := make([]float64, len(se.Times))
	for _, v := range se.Values {
		for i, x := range v {
			tot[i] += x
		}
	}
	return &Series{Step: se.Step, Times: se.Times, Names: []string{"total"}, Values: [][]float64{tot}}, nil
}

// Totals are headline numbers for a range.
type Totals struct {
	Bytes     uint64 `json:"bytes"`
	Wire      uint64 `json:"wire"`
	Pkts      uint64 `json:"pkts"`
	Flows     uint64 `json:"flows"`
	Hosts     uint64 `json:"hosts"`     // internal addresses seen
	Peers     uint64 `json:"peers"`     // external addresses seen
	Countries uint64 `json:"countries"` // distinct peer countries
}

func (s *Store) Totals(q Query) (Totals, error) {
	var t Totals
	if q.usesRollup() {
		err := s.DB.QueryRow(`SELECT coalesce(sum(bytes),0), coalesce(sum(wire),0), coalesce(sum(pkts),0), coalesce(sum(flows),0) FROM r_ts WHERE ts >= ? AND ts < ?`, q.From.UTC(), q.To.UTC()).
			Scan(&t.Bytes, &t.Wire, &t.Pkts, &t.Flows)
		if err != nil {
			return t, err
		}
		s.DB.QueryRow(`SELECT count(DISTINCT ip) FILTER (WHERE internal), count(DISTINCT ip) FILTER (WHERE NOT internal) FROM r_host WHERE ts >= ? AND ts < ?`, q.From.UTC(), q.To.UTC()).Scan(&t.Hosts, &t.Peers)
		s.DB.QueryRow(`SELECT count(DISTINCT val) FROM r_dim WHERE ts >= ? AND ts < ? AND dim = 'country' AND val NOT IN ('', '__internal__')`, q.From.UTC(), q.To.UTC()).Scan(&t.Countries)
		return t, nil
	}
	where, args, err := q.where()
	if err != nil {
		return t, err
	}
	src := s.Source(q.From, q.To)
	err = s.DB.QueryRow(fmt.Sprintf(`SELECT coalesce(sum(bytes),0), coalesce(sum(wire),0), coalesce(sum(pkts),0), coalesce(sum(flows),0),
		count(DISTINCT CASE WHEN NOT srv_int THEN server WHEN NOT cli_int THEN client END),
		count(DISTINCT nullif(peer_cc, ''))
		FROM %s WHERE %s`, src, where), args...).
		Scan(&t.Bytes, &t.Wire, &t.Pkts, &t.Flows, &t.Peers, &t.Countries)
	if err != nil {
		return t, err
	}
	// internal addresses on either side, each counted once (a host is often
	// a client in one conversation and a server in another)
	err = s.DB.QueryRow(fmt.Sprintf(`SELECT count(*) FROM (SELECT client FROM %s WHERE %s AND cli_int UNION SELECT server FROM %s WHERE %s AND srv_int)`,
		src, where, src, where), append(append([]any{}, args...), args...)...).Scan(&t.Hosts)
	return t, err
}

// Record is one detail row for the flow records view.
type Record struct {
	TS       int64   `json:"ts"`
	Client   string  `json:"client"`
	Server   string  `json:"server"`
	CPort    uint16  `json:"cport"`
	SvcPort  uint16  `json:"port"`
	Proto    uint8   `json:"proto"`
	App      string  `json:"app"`
	CC       string  `json:"cc"`
	ASN      uint32  `json:"asn"`
	Encap    uint8   `json:"encap"`
	Wire     uint64  `json:"wire"`
	Pkts     uint64  `json:"pkts"`
	Exporter string  `json:"exporter"`
	InIf     uint32  `json:"in_if"`
	OutIf    uint32  `json:"out_if"`
	VLAN     uint16  `json:"vlan"`
	Flags    uint8   `json:"tcp_flags"`
	Mult     float32 `json:"mult"`
	Dir      uint8   `json:"dir"`
	Threat   string  `json:"threat,omitempty"`
}

func (s *Store) Records(q Query, limit int) ([]Record, error) {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	where, args, err := q.where()
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(fmt.Sprintf(`SELECT ts, client, server, CASE WHEN client = src THEN sport ELSE dport END, svc_port, proto, app,
		peer_cc, peer_asn, encap, wire, pkts, exporter, in_if, out_if, vlan, tcp_flags, mult, dir, threat
		FROM %s WHERE %s ORDER BY ts DESC, wire DESC LIMIT %d`, s.Source(q.From, q.To), where, limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		var ts time.Time
		if err := rows.Scan(&ts, &r.Client, &r.Server, &r.CPort, &r.SvcPort, &r.Proto, &r.App, &r.CC, &r.ASN, &r.Encap,
			&r.Wire, &r.Pkts, &r.Exporter, &r.InIf, &r.OutIf, &r.VLAN, &r.Flags, &r.Mult, &r.Dir, &r.Threat); err != nil {
			return nil, err
		}
		r.TS = ts.UnixMilli()
		out = append(out, r)
	}
	return out, rows.Err()
}

// SankeyLink is one segment→app or app→country link.
type SankeyLink struct {
	Seg, App, CC string
	Wire         uint64
}

// Sankey returns traffic by segment (or, with byHost, by internal host),
// application and remote country. By host, only the top hosts are kept
// apart; the rest are summed as "__other__".
func (s *Store) Sankey(q Query, byHost bool, hosts int) ([]SankeyLink, error) {
	where, args, err := q.where()
	if err != nil {
		return nil, err
	}
	cc := `CASE WHEN dir = 3 THEN '__internal__' WHEN peer_cc = '' THEN '__unknown__' ELSE peer_cc END`
	sqlq := fmt.Sprintf(`SELECT segment, app, %s, sum(wire) FROM %s WHERE %s GROUP BY 1,2,3`, cc, s.Source(q.From, q.To), where)
	if byHost {
		// the internal side of each conversation: the client, or the server
		// of inbound traffic
		sqlq = fmt.Sprintf(`WITH b AS (SELECT CASE WHEN dir = %d THEN server ELSE client END AS h, app, %s AS cc, sum(wire) AS w
				FROM %s WHERE %s GROUP BY 1,2,3),
			t AS (SELECT h FROM b GROUP BY h ORDER BY sum(w) DESC LIMIT %d)
			SELECT CASE WHEN h IN (SELECT h FROM t) THEN h ELSE '__other__' END, app, cc, sum(w) FROM b GROUP BY 1,2,3`,
			DirInbound, cc, s.Source(q.From, q.To), where, hosts)
	}
	rows, err := s.DB.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SankeyLink
	for rows.Next() {
		var l SankeyLink
		if err := rows.Scan(&l.Seg, &l.App, &l.CC, &l.Wire); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
