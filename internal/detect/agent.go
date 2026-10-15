package detect

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// These four rules are about work done by a program rather than by a
// person at a keyboard, which is what an agent driving a network looks
// like from the flow data. None of them claims to identify what the
// program is; each says that the shape of the traffic is one a person
// does not produce.
//
// They are deliberately narrower than the rules they sit beside. A scan is
// a scan whoever runs it, and the scan rule already reports it; these fire
// only on the part that is characteristic — the speed, the regularity, the
// sequence — so that a finding of one of these kinds is worth reading on
// its own.

// fastScan: reconnaissance at a speed and breadth a person does not drive
// by hand. The scan rule already reports a sweep of many hosts on one port
// and a sweep of many ports on one host. What this adds is both at once
// inside a single minute: a source that in one minute touched many hosts
// AND many ports is working through a list, not looking at something.
func (w *window) fastScan() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT src, date_trunc('minute', ts) AS m,
			count(DISTINCT dst) AS nd, count(DISTINCT dport) AS np,
			sum(pkts), sum(wire), max(mult),
			array_to_string(list_slice(list(DISTINCT dst), 1, 6), ',')
		FROM %s WHERE %s AND proto = 6
		GROUP BY src, m
		HAVING count(DISTINCT dst) >= %d AND count(DISTINCT dport) >= %d
		   AND sum(wire) <= sum(pkts) * %g
		ORDER BY nd * np DESC LIMIT 50`,
		w.src, w.base(), c.FastScanTargets, c.FastScanPorts, c.ProbeMaxAvg)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("fastscan rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var src, ex string
		var m time.Time
		var nd, np int
		var pk, wire uint64
		var mult float64
		if err := rows.Scan(&src, &m, &nd, &np, &pk, &wire, &mult, &ex); err != nil {
			return nil, err
		}
		sev := 2
		if w.internal(src) {
			sev = 3
		}
		out = append(out, store.Finding{Kind: KindFastScan, Sev: sev, Src: src,
			First: m, Last: m.Add(time.Minute),
			Ev: map[string]any{"targets": float64(nd), "ports": float64(np), "pkts": float64(pk),
				"avg": float64(wire) / float64(max(pk, 1)), "sampling": mult, "examples": split(ex)}})
	}
	return out, rows.Err()
}

// beacon: one client talking to one service at a steady cadence, in small
// exchanges. A person's traffic to a service is bursty and irregular —
// they load a page, read it, load another. A program that checks in on a
// timer is regular, and the regularity is the whole signal: the rule asks
// for little traffic, evenly spaced, over enough of the window to be a
// pattern rather than a coincidence.
//
// Rows are per minute, so cadences faster than a minute are not visible
// here and are not claimed to be. What is visible is a check-in every
// minute, every two, every five.
func (w *window) beacon() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`SELECT client, server, svc_port, proto,
			list(DISTINCT date_trunc('minute', ts) ORDER BY date_trunc('minute', ts)) AS mins,
			sum(wire), sum(pkts), max(mult), any_value(peer_org)
		FROM %s WHERE %s AND NOT srv_int AND proto IN (6, 17)
		GROUP BY client, server, svc_port, proto
		HAVING count(DISTINCT date_trunc('minute', ts)) >= %d
		   AND sum(wire) <= %g * count(DISTINCT date_trunc('minute', ts))
		LIMIT 200`,
		w.src, w.base(), c.BeaconMinutes, c.BeaconMaxBytesPerMin)
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("beacon rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var cl, sv, org string
		var port, proto int
		// The driver hands a list back as []any, so the times come out
		// one at a time rather than as a []time.Time.
		var raw []any
		var wire, pk uint64
		var mult float64
		if err := rows.Scan(&cl, &sv, &port, &proto, &raw, &wire, &pk, &mult, &org); err != nil {
			return nil, err
		}
		mins := make([]time.Time, 0, len(raw))
		for _, v := range raw {
			if t, ok := v.(time.Time); ok {
				mins = append(mins, t.UTC())
			}
		}
		period, jitter, ok := cadence(mins)
		if !ok || jitter > c.BeaconJitter {
			continue
		}
		out = append(out, store.Finding{Kind: KindBeacon, Sev: 2, Src: cl, Dst: sv,
			Port: portLabel(port, proto), First: mins[0], Last: mins[len(mins)-1].Add(time.Minute),
			Ev: map[string]any{"period_s": period.Seconds(), "jitter": jitter, "checkins": float64(len(mins)),
				"bytes": float64(wire), "pkts": float64(pk), "sampling": mult, "org": org}})
	}
	return out, rows.Err()
}

// cadence measures how evenly spaced a set of minutes is. It returns the
// typical gap and the jitter: the mean absolute difference from that gap,
// as a fraction of it. Zero jitter is a perfect metronome. It needs at
// least three points, because two are evenly spaced by definition.
func cadence(mins []time.Time) (time.Duration, float64, bool) {
	if len(mins) < 3 {
		return 0, 0, false
	}
	sort.Slice(mins, func(i, j int) bool { return mins[i].Before(mins[j]) })
	gaps := make([]float64, 0, len(mins)-1)
	for i := 1; i < len(mins); i++ {
		g := mins[i].Sub(mins[i-1]).Seconds()
		if g <= 0 {
			continue
		}
		gaps = append(gaps, g)
	}
	if len(gaps) < 2 {
		return 0, 0, false
	}
	var sum float64
	for _, g := range gaps {
		sum += g
	}
	mean := sum / float64(len(gaps))
	if mean <= 0 {
		return 0, 0, false
	}
	var dev float64
	for _, g := range gaps {
		dev += math.Abs(g - mean)
	}
	return time.Duration(mean * float64(time.Second)), dev / float64(len(gaps)) / mean, true
}

// chain: A reaches B on an administration service, and then B reaches C on
// one, inside the same window. One hop is lateral movement and the lateral
// rule reports it; the second hop is the thing a person does over hours
// and a program does in seconds. The rule reports the middle host, because
// that is the one that was taken and then used.
func (w *window) chain() ([]store.Finding, error) {
	c := w.d.Cfg
	q := fmt.Sprintf(`WITH hops AS (
			SELECT client AS a, server AS b, svc_port AS port, min(ts) AS first, max(ts) AS last
			FROM %s WHERE %s AND proto = 6 AND cli_int AND srv_int AND svc_port IN (%s)
			GROUP BY client, server, svc_port
		)
		SELECT i.a, i.b, o.b, i.port, o.port, i.first, o.last,
			array_to_string(list_slice(list(DISTINCT o.b), 1, 6), ',')
		FROM hops i JOIN hops o ON i.b = o.a
		WHERE o.b <> i.a AND o.first >= i.first AND o.first <= i.first + INTERVAL '%d seconds'
		GROUP BY i.a, i.b, o.b, i.port, o.port, i.first, o.last
		LIMIT 50`,
		w.src, w.base(), intList(adminPorts), int(c.ChainWithin.Seconds()))
	rows, err := w.d.St.DB.Query(q, w.args()...)
	if err != nil {
		return nil, fmt.Errorf("chain rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var a, b, cc, ex string
		var inPort, outPort int
		var first, last time.Time
		if err := rows.Scan(&a, &b, &cc, &inPort, &outPort, &first, &last, &ex); err != nil {
			return nil, err
		}
		out = append(out, store.Finding{Kind: KindChain, Sev: 3, Src: b, Dst: cc,
			Port: portLabel(outPort, 6), First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"from": a, "via": b, "to": cc,
				"in_port": portLabel(inPort, 6), "out_port": portLabel(outPort, 6), "examples": split(ex)}})
	}
	return out, rows.Err()
}

// llm: an internal host sending its traffic to a machine-learning API.
// This is not a fault and is not reported as one: it is reported because
// an agent that acts on a network is usually driven by one, and because
// what leaves for one is prompt text, which is the part of this that an
// administrator may need to know about before it leaves.
//
// It matches on the organisation the address belongs to, from the same
// lookup the rest of the pages use, against a list that is a starting
// point and is meant to be edited: Cfg.LLMOrgs replaces it entirely.
func (w *window) llm() ([]store.Finding, error) {
	orgs := w.d.Cfg.LLMOrgs
	if len(orgs) == 0 {
		return nil, nil
	}
	pats := make([]string, len(orgs))
	args := w.args()
	for i, o := range orgs {
		pats[i] = "lower(peer_org) LIKE ?"
		args = append(args, "%"+strings.ToLower(o)+"%")
	}
	q := fmt.Sprintf(`SELECT client, server, any_value(peer_org), svc_port,
			sum(bytes), sum(pkts), count(DISTINCT date_trunc('minute', ts)), min(ts), max(ts)
		FROM %s WHERE %s AND NOT srv_int AND dir = %d AND (%s)
		GROUP BY client, server, svc_port
		HAVING sum(bytes) >= %g
		ORDER BY sum(bytes) DESC LIMIT 50`,
		w.src, w.base(), store.DirOutbound, strings.Join(pats, " OR "), w.d.Cfg.LLMBytes)
	rows, err := w.d.St.DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("llm rule: %w", err)
	}
	defer rows.Close()
	var out []store.Finding
	for rows.Next() {
		var cl, sv, org string
		var port, mins int
		var by, pk uint64
		var first, last time.Time
		if err := rows.Scan(&cl, &sv, &org, &port, &by, &pk, &mins, &first, &last); err != nil {
			return nil, err
		}
		out = append(out, store.Finding{Kind: KindLLM, Sev: 1, Src: cl, Dst: sv,
			Port: portLabel(port, 6), First: first, Last: last.Add(time.Minute),
			Ev: map[string]any{"org": org, "bytes": float64(by), "pkts": float64(pk), "minutes": float64(mins)}})
	}
	return out, rows.Err()
}
