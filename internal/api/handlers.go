package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/githubflyideas/traffic66/internal/snmp"
	"net/http"
	"net/netip"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/store"
)

var ranges = map[string]time.Duration{
	"15m": 15 * time.Minute, "1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour,
	"7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour,
}

// longRange is the span above which queries are answered from hourly
// summaries. Such ranges start on a whole hour, so that every number on a
// page (totals and charts per minute, Top-N and the other summaries per hour)
// counts exactly the same window; "24 hours" then covers the last 24 whole
// hours plus the current one.
const longRange = 6 * time.Hour

func parseQuery(r *http.Request) (store.Query, error) {
	v := r.URL.Query()
	now := time.Now().UTC()
	q := store.Query{To: now}
	if f, t := v.Get("from"), v.Get("to"); f != "" && t != "" {
		fm, err1 := strconv.ParseInt(f, 10, 64)
		tm, err2 := strconv.ParseInt(t, 10, 64)
		if err1 != nil || err2 != nil || tm <= fm {
			return q, errors.New("bad from/to")
		}
		q.From, q.To = time.UnixMilli(fm).UTC(), time.UnixMilli(tm).UTC()
		if q.To.Sub(q.From) > longRange {
			q.From = q.From.Truncate(time.Hour)
			if t := q.To.Truncate(time.Hour); !t.Equal(q.To) {
				q.To = t.Add(time.Hour)
			}
		}
	} else {
		d, ok := ranges[v.Get("range")]
		if !ok {
			d = 24 * time.Hour
		}
		q.From = now.Add(-d)
		if d > longRange {
			q.From = q.From.Truncate(time.Hour)
		}
	}
	if f := v.Get("f"); f != "" {
		if err := json.Unmarshal([]byte(f), &q.Filters); err != nil {
			return q, fmt.Errorf("bad filters: %w", err)
		}
		if len(q.Filters) > 20 {
			return q, errors.New("too many filters")
		}
	}
	q.IncludeDup = v.Get("dup") == "1"
	return q, nil
}

func shift(q store.Query, d time.Duration) store.Query {
	q.From, q.To = q.From.Add(d), q.To.Add(d)
	return q
}

func limitParam(r *http.Request, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return def
	}
	if n > 1000 {
		n = 1000
	}
	return n
}

// ---------------------------------------------------------------- overview

type slicePart struct {
	Key  string `json:"key"`
	Wire uint64 `json:"wire"`
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	span := q.To.Sub(q.From)
	out := map[string]any{"from": q.From.UnixMilli(), "to": q.To.UnixMilli()}

	tot, err := s.Store.Totals(q)
	if err != nil {
		fail(w, err)
		return
	}
	out["totals"] = tot

	// baseline: same window last week, else the previous window
	basis := "week"
	qb := shift(q, -7*24*time.Hour)
	btot, _ := s.Store.Totals(qb)
	if btot.Wire == 0 {
		basis = "prev"
		qb = shift(q, -span)
		btot, _ = s.Store.Totals(qb)
	}
	out["basis"] = basis
	out["base_totals"] = btot

	nowQ := q
	nowQ.To = time.Now().UTC().Truncate(time.Minute)
	nowQ.From = nowQ.To.Add(-5 * time.Minute)
	if nt, err := s.Store.Totals(nowQ); err == nil {
		out["now_bps"] = float64(nt.Wire) * 8 / 300
	}

	se, err := s.Store.SeriesByApp(q, 8)
	if err != nil {
		fail(w, err)
		return
	}
	out["series"] = se
	peak, peakAt := 0.0, int64(0)
	for i := range se.Times {
		sum := 0.0
		for _, v := range se.Values {
			sum += v[i]
		}
		if sum > peak {
			peak, peakAt = sum, se.Times[i]
		}
	}
	out["peak_bps"], out["peak_at"] = peak, peakAt
	if btot.Wire > 0 {
		if bs, err := s.Store.TotalSeries(qb); err == nil && len(bs.Values) == 1 {
			vals := bs.Values[0]
			if len(vals) > len(se.Times) {
				vals = vals[:len(se.Times)]
			}
			out["baseline"] = vals
		}
	}

	donut := func(dim string, keep int) []slicePart {
		rows, err := s.Store.TopN(q, dim, 20)
		if err != nil {
			return nil
		}
		var parts []slicePart
		var other uint64
		for i, t := range rows {
			if i < keep && t.Key != "" {
				parts = append(parts, slicePart{t.Key, t.Wire})
			} else {
				other += t.Wire
			}
		}
		if other > 0 {
			parts = append(parts, slicePart{"__other__", other})
		}
		return parts
	}
	out["dir"] = donut("dir", 4)
	out["proto"] = donut("proto", 2)
	out["country"] = donut("country", 3)
	if cl, err := s.Store.TopN(q, "client", 10); err == nil {
		out["top_clients"] = s.withBase(cl, qb, "client")
	}
	if sv, err := s.Store.TopN(q, "service", 10); err == nil {
		out["top_services"] = sv
	}
	writeJSON(w, http.StatusOK, out)
}

// withBase adds the baseline wire bytes per key as the Extra field.
func (s *Server) withBase(rows []store.TopRow, qb store.Query, dim string) []store.TopRow {
	if len(rows) == 0 {
		return rows
	}
	var ips []string
	for _, r := range rows {
		ips = append(ips, r.Key)
	}
	base := s.hostWire(qb, ips)
	for i := range rows {
		rows[i].Extra = strconv.FormatUint(base[rows[i].Key], 10)
	}
	return rows
}

func (s *Server) hostWire(q store.Query, ips []string) map[string]uint64 {
	out := map[string]uint64{}
	for _, ip := range ips {
		qq := q
		qq.Filters = append(append([]store.Filter{}, q.Filters...), store.Filter{Field: "ip", Value: ip})
		if t, err := s.Store.Totals(qq); err == nil {
			out[ip] = t.Wire
		}
	}
	return out
}

// ---------------------------------------------------------------- lists

func (s *Server) topn(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	dim, by := r.URL.Query().Get("dim"), r.URL.Query().Get("by")
	if by == "" {
		by = "wire"
	}
	rows, err := s.Store.TopNBy(q, dim, limitParam(r, 66), by, r.URL.Query().Get("asc") == "1")
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dim": dim, "rows": rows, "rollup": len(q.Filters) == 0 && q.To.Sub(q.From) > 6*time.Hour})
}

func (s *Server) sankey(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	mode := r.URL.Query().Get("by")
	if mode == "" {
		mode = "host"
	}
	// 8 hosts, clients or servers (the colours there are); 6 applications,
	// services and countries
	n0, n2 := 8, 6
	if mode == "conv" {
		n2 = 8
	}
	links, err := s.Store.Sankey(q, mode, n0, n2)
	if err != nil {
		fail(w, err)
		return
	}
	// fold everything past the largest nodes of each column into "__other__"
	top := func(get func(store.SankeyLink) string, n ...int) map[string]bool {
		t := map[string]uint64{}
		for _, l := range links {
			t[get(l)] += l.Wire
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return t[keys[i]] > t[keys[j]] })
		m := map[string]bool{}
		limit := 6
		if len(n) > 0 {
			limit = n[0]
		}
		for i, k := range keys {
			if i < limit && k != "__other__" {
				m[k] = true
			}
		}
		return m
	}
	first := n0
	ks := top(func(l store.SankeyLink) string { return l.Seg }, first)
	ka := top(func(l store.SankeyLink) string { return l.App })
	kc := top(func(l store.SankeyLink) string { return l.CC }, n2)
	fold := func(m map[string]bool, k string) string {
		if m[k] {
			return k
		}
		return "__other__"
	}
	type link struct {
		S, T string
		V    uint64
	}
	l1, l2 := map[[2]string]uint64{}, map[[2]string]uint64{}
	for _, l := range links {
		a, b, c := fold(ks, l.Seg), fold(ka, l.App), fold(kc, l.CC)
		l1[[2]string{a, b}] += l.Wire
		l2[[2]string{b, c}] += l.Wire
	}
	flat := func(m map[[2]string]uint64) []link {
		var out []link
		for k, v := range m {
			out = append(out, link{k[0], k[1], v})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].V > out[j].V })
		return out
	}
	writeJSON(w, http.StatusOK, map[string]any{"by": mode, "seg_app": flat(l1), "app_cc": flat(l2)})
}

func (s *Server) records(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	rows, err := s.Store.Records(q, limitParam(r, 200))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (s *Server) threats(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	rows, err := s.Store.TopN(q, "threat", limitParam(r, 66))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows, "lists": s.Thr.Lists()})
}

func (s *Server) ifaces(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	list, err := s.Store.Interfaces(q)
	if err != nil {
		fail(w, err)
		return
	}
	type item struct {
		store.IfaceInfo
		Name   string  `json:"name"`
		Device string  `json:"device"`
		InDev  float64 `json:"in_dev"`
		OutDev float64 `json:"out_dev"`
		Err    float64 `json:"stat_err"`
		HasCtr bool    `json:"has_counters"`
	}
	var out []item
	for i, f := range list {
		if i >= 50 {
			break
		}
		it := item{IfaceInfo: f}
		if a, err := netip.ParseAddr(f.Exporter); err == nil {
			it.Device = s.Inv.Device(a)
			ifc := s.Inv.Iface(a, f.IfIndex)
			it.Name = ifc.Name
			if it.Speed == 0 {
				it.Speed = ifc.Speed
			}
		}
		if f.Counters {
			if rc, err := s.Store.Recon(q, f.Exporter, f.IfIndex); err == nil && rc.HasCounters {
				it.InDev, it.OutDev, it.Err, it.HasCtr = rc.InDev, rc.OutDev, rc.InStatErr, true
			}
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ifaces": out})
}

func (s *Server) recon(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	exp := r.URL.Query().Get("exporter")
	idx, err := strconv.ParseUint(r.URL.Query().Get("ifindex"), 10, 32)
	if err != nil || exp == "" {
		fail(w, errors.New("exporter and ifindex required"))
		return
	}
	rc, err := s.Store.Recon(q, exp, uint32(idx))
	if err != nil {
		fail(w, err)
		return
	}
	out := map[string]any{"recon": rc}
	if rc.InPkts > 0 && rc.HasCounters {
		out["gap_per_pkt"] = (float64(rc.InCtrBytes) - float64(rc.InFlowBytes)) / float64(rc.InPkts)
	}
	for _, src := range s.Col.Sources() {
		if src.Exporter == exp {
			out["source"] = src
			break
		}
	}
	if a, err := netip.ParseAddr(exp); err == nil {
		out["name"] = s.Inv.Iface(a, uint32(idx)).Name
		out["device"] = s.Inv.Device(a)
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------- sources

type sourceOut struct {
	collector.SourceInfo
	Name   string   `json:"name"`
	Status string   `json:"status"` // ok, warn, bad
	Issues []string `json:"issues"`
	RecPS  float64  `json:"rec_per_sec"`
}

var srcRate = struct {
	last map[string]uint64
	at   time.Time
	ps   map[string]float64
}{last: map[string]uint64{}, ps: map[string]float64{}}

func srcKey(si collector.SourceInfo) string {
	return si.Exporter + "|" + si.Proto + "|" + strconv.Itoa(int(si.Domain))
}

// sampleSources updates per-exporter record rates; called every few seconds.
func (s *Server) sampleSources(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt := now.Sub(srcRate.at).Seconds()
	for _, si := range s.Col.Sources() {
		k := srcKey(si)
		if last, ok := srcRate.last[k]; ok && dt > 0.5 {
			r := float64(si.Records-last) / dt
			if old, ok := srcRate.ps[k]; ok {
				r = old*0.6 + r*0.4
			}
			srcRate.ps[k] = r
		}
		srcRate.last[k] = si.Records
	}
	srcRate.at = now
}

func (s *Server) sources(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	s.mu.Lock()
	var out []sourceOut
	for _, si := range s.Col.Sources() {
		o := sourceOut{SourceInfo: si, Status: "ok"}
		o.RecPS = srcRate.ps[srcKey(si)]
		if a, err := netip.ParseAddr(si.Exporter); err == nil {
			o.Name = s.Inv.Device(a)
		}
		issue := func(level, code string) {
			o.Issues = append(o.Issues, code)
			if level == "bad" || o.Status == "ok" {
				o.Status = level
			}
		}
		if now.Sub(si.LastSeen) > 2*time.Minute {
			issue("bad", "silent")
		}
		switch si.SamplingState {
		case "waiting":
			issue("warn", "waiting_sampling")
		case "assumed1":
			issue("warn", "assumed_unsampled")
		}
		if si.LostPct >= 1 {
			issue("warn", "loss")
		}
		if si.NoTemplate > 0 && si.Templates == 0 {
			issue("warn", "no_template")
		}
		if si.ClockSkew > 2*time.Minute || si.ClockSkew < -2*time.Minute {
			issue("warn", "clock_skew")
		}
		if si.DecodeErrors > 0 && si.DecodeErrors*100 > si.Packets {
			issue("warn", "decode_errors")
		}
		out = append(out, o)
	}
	s.mu.Unlock()
	var ls []map[string]any
	for _, l := range s.Col.Listeners {
		ls = append(ls, map[string]any{"addr": l.Addr, "proto": l.Proto, "packets": l.Packets.Load(), "undecoded": l.Undecoded.Load(), "rcvbuf": l.RcvBuf})
	}
	res := map[string]any{"sources": out, "listeners": ls}
	if s.Capture != nil {
		res["captures"] = s.Capture()
	}
	if s.SNMP != nil {
		type snmpOut struct {
			snmp.Status
			Name string `json:"name,omitempty"`
		}
		var so []snmpOut
		for _, st := range s.SNMP() {
			o := snmpOut{Status: st}
			if a, err := netip.ParseAddr(st.Exporter); err == nil {
				o.Name = s.Inv.Device(a)
			}
			so = append(so, o)
		}
		res["snmp"] = so
	}
	writeJSON(w, http.StatusOK, res)
}

// ---------------------------------------------------------------- misc

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	u := s.Store.Usage()
	s.rates.mu.Lock()
	rec, rows := s.rates.recPerSec, s.rates.rowRate
	s.rates.mu.Unlock()
	// Disk: what is used and free, and, once there is a day of data, how much
	// the kept days of detail will need at the current rate. Retention
	// deletes older data, so the disk does not fill as long as that fits.
	free := diskFree(s.DataDir)
	need := int64(-1)
	if !u.Oldest.IsZero() {
		age := time.Since(u.Oldest).Hours() / 24
		if age >= 1 && u.DiskBytes > 0 {
			need = int64(float64(u.DiskBytes) / age * float64(s.Store.RetentionDays()))
		}
	}
	warn := 0
	for _, si := range s.Col.Sources() {
		if si.SamplingState == "waiting" || si.LostPct >= 1 || time.Since(si.LastSeen) > 2*time.Minute {
			warn++
		}
	}
	if s.SNMP != nil {
		for _, st := range s.SNMP() {
			if !st.OK {
				warn++
			}
		}
	}
	names := map[string]string{}
	for ip, n := range s.Inv.Hosts() {
		names[ip] = n
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version": s.Version, "demo": s.Demo, "uptime": int64(time.Since(s.Started).Seconds()),
		"records_per_sec": rec, "rows_per_sec": rows, "dropped": s.Pipe.Dropped.Load(), "dup_rows": s.Pipe.DupRows.Load(), "dedup_full": s.Pipe.DedupFull.Load(),
		"write_errors": s.Pipe.WriteErrs.Load(), "last_error": s.Pipe.LastError(),
		"disk_bytes": u.DiskBytes, "disk_free": free, "disk_need": need, "retention_days": s.Store.RetentionDays(), "oldest": u.Oldest.UnixMilli(),
		"hot_rows": u.HotRows, "segments": u.Segments, "segment_rows": u.SegmentRows,
		"dns_upstream": s.DNS.Upstream(), "dns_queries": s.DNS.Queries, "asn_ranges": s.ASN.Size(), "threat_lists": s.Thr.Lists(),
		"source_warnings": warn, "findings_open": s.Store.OpenFindings(2, time.Now().Add(-24*time.Hour)), "hosts": names, "skewed": s.Col.Skewed.Load(), "go": runtime.Version(),
	})
}

func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	var in struct{ IPs []string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, err)
		return
	}
	if len(in.IPs) > 500 {
		in.IPs = in.IPs[:500]
	}
	out := map[string]string{}
	var ask []string
	for _, ip := range in.IPs {
		a, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		if n := s.Inv.Host(a); n != "" {
			out[ip] = n
			continue
		}
		ask = append(ask, ip)
	}
	for ip, n := range s.DNS.Lookup(ask, 800*time.Millisecond) {
		out[ip] = n
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getInventory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"text": s.Inv.Text()})
}

func (s *Server) putInventory(w http.ResponseWriter, r *http.Request) {
	var in struct{ Text string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		fail(w, err)
		return
	}
	if err := s.Inv.Save(strings.ReplaceAll(in.Text, "\r\n", "\n")); err != nil {
		fail(w, err)
		return
	}
	s.Col.NF.SetUnsampled(s.Inv.Unsampled())
	writeJSON(w, http.StatusOK, map[string]string{"ok": "saved"})
}
