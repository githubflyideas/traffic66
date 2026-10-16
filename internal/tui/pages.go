package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type topRow struct {
	Key   string  `json:"key"`
	Key2  string  `json:"key2"`
	Key3  string  `json:"key3"`
	Extra string  `json:"extra"`
	Wire  float64 `json:"wire"`
	Pkts  float64 `json:"pkts"`
	Flows float64 `json:"flows"`
	Peers float64 `json:"peers"`
}

var protoNames = map[string]string{"1": "ICMP", "6": "TCP", "17": "UDP", "47": "GRE", "50": "ESP", "58": "ICMPv6", "132": "SCTP"}

func (a *app) cc(c string) string {
	switch c {
	case "__internal__":
		return a.t("internal")
	case "__other__":
		return a.t("other")
	case "", "__unknown__":
		return a.t("unknown")
	}
	return c
}

func (a *app) host(ip string) line {
	if n := a.names[ip]; n != "" {
		return line{{"", n + " "}, {dim, ip}}
	}
	return line{{"", ip}}
}

func (a *app) hostLabel(ip string) string {
	if n := a.names[ip]; n != "" {
		return n + " " + ip
	}
	return ip
}

func txt(s string) line        { return line{{"", s}} }
func styled(st, s string) line { return line{{st, s}} }
func ipTarget(a *app, ip string) *target {
	return &target{"ip", ip, a.hostLabel(ip)}
}

func spark(vals []float64, width int, maxv float64) string {
	if len(vals) == 0 || width <= 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	out := make([]rune, 0, width)
	for c := 0; c < width; c++ {
		lo := c * len(vals) / width
		hi := max(lo+1, (c+1)*len(vals)/width)
		v, n := 0.0, 0
		for _, x := range vals[lo:min(hi, len(vals))] {
			if x >= 0 {
				v += x
				n++
			}
		}
		if n == 0 {
			out = append(out, ' ')
			continue
		}
		v /= float64(n)
		i := 0
		if maxv > 0 {
			i = int(math.Round(v / maxv * 7))
		}
		out = append(out, blocks[max(0, min(7, i))])
	}
	return string(out)
}

// measure turns bytes and packets over the range into the chosen unit.
type measure struct {
	unit int
	secs float64
}

func (m measure) val(wire, pkts float64) float64 {
	switch units[m.unit] {
	case "bytes":
		return wire
	case "pps":
		return pkts / m.secs
	}
	return wire * 8 / m.secs
}

func (m measure) fmt(v float64) string {
	switch units[m.unit] {
	case "bytes":
		return fmtBytes(v)
	case "pps":
		return fmtPps(v)
	}
	return fmtBps(v)
}

// load fetches and lays out one page.
func (a *app) load(v view) page {
	var p page
	fail := func(err error) page {
		p.err = err.Error()
		return p
	}
	q := func(extra map[string]string) map[string][]string { return query(v.rng, v.filters, extra) }
	m := measure{v.unit, rangeDur[v.rng].Seconds()}

	switch pageKeys[v.pg] {
	case "top":
		// the header: total rate in and out
		var dir struct{ Rows []topRow }
		if err := a.get("topn", q(map[string]string{"dim": "dir", "limit": "10"}), &dir); err != nil {
			return fail(err)
		}
		var in, out, all topRow
		for _, r := range dir.Rows {
			switch r.Key {
			case "2":
				in = r
			case "1":
				out = r
			}
			all.Wire += r.Wire
			all.Pkts += r.Pkts
		}
		hl := line{{dim, a.t("tui.in") + " "}, {bold + green, m.fmt(m.val(in.Wire, in.Pkts))}, {dim, "   " + a.t("tui.out") + " "}, {bold + blue, m.fmt(m.val(out.Wire, out.Pkts))},
			{dim, "   " + a.t("tui.total") + " "}, {bold, m.fmt(m.val(all.Wire, all.Pkts))}}
		if units[v.unit] != "bytes" {
			hl = append(hl, seg{dim, "   " + a.t("tui.avg", "r", a.t("range."+ranges[v.rng]))})
		}
		p.top = append(p.top, hl)
		tabs := line{{dim, "← → "}}
		for i, k := range dims {
			if i == v.dim {
				tabs = append(tabs, seg{rev + bold, " " + a.t("dim."+k) + " "})
			} else {
				tabs = append(tabs, seg{"", " " + a.t("dim."+k) + " "})
			}
		}
		if v.w < narrow || tabs.width() > v.w {
			tabs = line{{dim, "← "}, {rev + bold, " " + a.t("dim."+dims[v.dim]) + " "}, {dim, fmt.Sprintf(" → %d/%d", v.dim+1, len(dims))}}
		}
		p.top = append(p.top, tabs)

		dm := dims[v.dim]
		by := "wire"
		if units[v.unit] == "pps" {
			by = "pkts"
		}
		var d, prev struct{ Rows []topRow }
		if err := a.get("topn", q(map[string]string{"dim": dm, "limit": "66", "by": by}), &d); err != nil {
			return fail(err)
		}
		// the same window just before, for the change column
		now := time.Now()
		span := rangeDur[v.rng]
		pq := q(map[string]string{"dim": dm, "limit": "1000", "by": by,
			"from": strconv.FormatInt(now.Add(-2*span).UnixMilli(), 10), "to": strconv.FormatInt(now.Add(-span).UnixMilli(), 10)})
		delete(pq, "range")
		a.get("topn", pq, &prev)
		base := map[string]float64{}
		for _, r := range prev.Rows {
			base[r.Key] = m.val(r.Wire, r.Pkts)
		}
		head := a.t("dim." + dm)
		sec := section{cols: []column{{head: "#", width: 3, right: true}, {head: head}, {head: a.t("tui.unit." + units[v.unit]), width: 11, right: true},
			{head: "%", width: 5, right: true}, {head: "", width: 20, optional: true}, {head: "Δ", width: 6, right: true, optional: true}, {head: a.t("col.flows"), width: 8, right: true, optional: true}}}
		tot, maxv := 0.0, 0.0
		for _, r := range d.Rows {
			x := m.val(r.Wire, r.Pkts)
			tot += x
			maxv = math.Max(maxv, x)
		}
		for i, r := range d.Rows {
			var cell line
			var tg *target
			switch dm {
			case "client", "server":
				cell, tg = a.host(r.Key), ipTarget(a, r.Key)
			case "country":
				cell, tg = txt(a.cc(r.Key)), &target{"country", r.Key, a.cc(r.Key)}
			case "asn":
				cell, tg = line{{"", "AS" + r.Key + " "}, {dim, r.Extra}}, &target{"asn", r.Key, "AS" + r.Key}
			case "port":
				cell, tg = line{{"", r.Key + " "}, {dim, r.Extra}}, &target{"port", r.Key, r.Key}
			default:
				cell, tg = txt(r.Key), &target{dm, r.Key, r.Key}
			}
			x := m.val(r.Wire, r.Pkts)
			b, known := base[r.Key]
			dl := delta(x, b, maxv*0.001)
			if !known && len(prev.Rows) >= 1000 {
				dl = "" // beyond the earlier list: unknown, not new
			}
			dst := dim
			switch {
			case dl == "new":
				dl, dst = a.t("tui.new"), yellow
			case dl == ">10×" || strings.HasPrefix(dl, "+"):
				dst = yellow
			}
			sec.rows = append(sec.rows, row{key: r.Key, cells: []line{styled(dim, strconv.Itoa(i+1)), cell, txt(m.fmt(x)),
				styled(dim, fmt.Sprintf("%.1f", x/math.Max(tot, 1e-9)*100)), styled(blue, barCells(x, maxv, 20)), styled(dst, dl), styled(dim, strconv.Itoa(int(r.Flows)))}, target: tg})
		}
		p.sections = []section{sec}

	case "ifaces":
		return a.loadIfaces(v, m, q)

	case "search":
		var d struct {
			Rows []struct {
				TS                                int64 `json:"ts"`
				Client, Server, App, CC, Exporter string
				Port                              int `json:"port"`
				Proto                             int `json:"proto"`
				Wire                              float64
				Pkts                              float64
				Dir                               int
			}
		}
		n := a.pageSize()
		if err := a.get("records", q(map[string]string{"limit": strconv.Itoa(n), "offset": strconv.Itoa(v.offset)}), &d); err != nil {
			return fail(err)
		}
		if v.filters == "" || v.filters == "null" || v.filters == "[]" {
			p.top = append(p.top, styled(dim, a.t("tui.search_hint")))
		}
		sec := section{cols: []column{{head: a.t("rc.time"), width: 8}, {head: a.t("rc.client")}, {head: a.t("rc.server")}, {head: a.t("rc.port"), width: 10},
			{head: a.t("rc.app"), width: 12, optional: true}, {head: a.t("rc.country"), width: 9, optional: true}, {head: a.t("rc.traffic"), width: 9, right: true}, {head: a.t("rc.device"), width: 16, optional: true}}}
		for i, r := range d.Rows {
			pr := protoNames[strconv.Itoa(r.Proto)]
			cc := r.CC
			if r.Dir == 3 {
				cc = a.t("internal")
			}
			amt := fmtBytes(r.Wire)
			if units[v.unit] == "pps" {
				amt = fmt.Sprintf("%.0f pkt", r.Pkts)
			}
			sec.rows = append(sec.rows, row{key: strconv.Itoa(v.offset + i), cells: []line{styled(dim, time.UnixMilli(r.TS).Format("15:04:05")), a.host(r.Client), a.host(r.Server),
				txt(fmt.Sprintf("%d/%s", r.Port, strings.ToLower(pr))), txt(r.App), txt(cc), txt(amt), styled(dim, a.names[r.Exporter]+" "+r.Exporter)}, target: ipTarget(a, r.Server)})
		}
		p.sections = []section{sec}
		first, last := v.offset+1, v.offset+len(d.Rows)
		if len(d.Rows) == 0 {
			first = v.offset
		}
		p.foot = append(p.foot, styled(dim, a.t("tui.records_page", "a", strconv.Itoa(first), "b", strconv.Itoa(last))))
	}
	return p
}

type source struct {
	Exporter, Proto, Name, Sampling, Status string
	LostPct                                 float64 `json:"lost_pct"`
	Issues                                  []string
	Pending                                 int
	ClockSkew                               float64 `json:"clock_skew_ns"`
}

func (a *app) loadIfaces(v view, m measure, q func(map[string]string) map[string][]string) page {
	var p page
	var d struct {
		Ifaces []struct {
			Exporter, Name, Device string
			IfIndex                int     `json:"ifindex"`
			Speed                  float64 `json:"speed"`
			InWire                 float64 `json:"in_wire"`
			OutWire                float64 `json:"out_wire"`
			InDev                  float64 `json:"in_dev"`
			OutDev                 float64 `json:"out_dev"`
			Err                    float64 `json:"stat_err"`
			HasCtr                 bool    `json:"has_counters"`
		}
	}
	if err := a.get("ifaces", q(nil), &d); err != nil {
		p.err = err.Error()
		return p
	}
	var src struct{ Sources []source }
	a.get("sources", nil, &src)
	srcs := map[string][]source{}
	for _, s := range src.Sources {
		srcs[s.Exporter] = append(srcs[s.Exporter], s)
	}
	// rates are in bits or bytes; packets are not counted per interface
	um := m
	if units[m.unit] == "pps" {
		um.unit = 0
	}
	secs := m.secs
	order := []string{}
	byDev := map[string][]int{}
	for i, f := range d.Ifaces {
		if _, ok := byDev[f.Exporter]; !ok {
			order = append(order, f.Exporter)
		}
		byDev[f.Exporter] = append(byDev[f.Exporter], i)
	}
	// devices that send flows but no interface yet still show their sampling
	for _, s := range src.Sources {
		if _, ok := byDev[s.Exporter]; !ok {
			order = append(order, s.Exporter)
			byDev[s.Exporter] = nil
		}
	}
	cols := []column{{head: a.t("field.iface")}, {head: a.t("tui.in"), width: 10, right: true}, {head: a.t("tui.out"), width: 10, right: true},
		{head: a.t("tui.util"), width: 6, right: true}, {head: "", width: 10, optional: true}, {head: a.t("tui.recon"), width: 15, right: true, optional: true}}
	if len(order) == 0 {
		p.top = append(p.top, styled(dim, a.t("empty.first")))
	}
	for _, exp := range order {
		title := line{}
		devName := ""
		for _, i := range byDev[exp] {
			if d.Ifaces[i].Device != "" {
				devName = d.Ifaces[i].Device
				break
			}
		}
		for _, s := range srcs[exp] {
			if devName == "" {
				devName = s.Name
			}
		}
		if devName != "" {
			title = append(title, seg{bold, devName + " "})
		}
		title = append(title, seg{dim, exp})
		for _, s := range srcs[exp] {
			st := green
			switch s.Status {
			case "warn":
				st = yellow
			case "bad":
				st = red
			}
			samp := s.Sampling
			if samp == "" {
				samp = "?"
			}
			title = append(title, seg{dim, "  · " + s.Proto + " · " + a.t("tui.sampling") + " "}, seg{"", samp}, seg{dim, " · " + a.t("tui.loss") + " "}, seg{st, fmt.Sprintf("%.2f%%", s.LostPct)})
			for _, is := range s.Issues {
				title = append(title, seg{st, " ! " + firstSentence(a.t("issue."+is, "n", strconv.Itoa(s.Pending), "p", fmt.Sprintf("%.1f%%", s.LostPct), "s", fmt.Sprintf("%.0f s", math.Abs(s.ClockSkew)/1e9)))})
			}
		}
		sec := section{title: title, cols: cols}
		for _, i := range byDev[exp] {
			f := d.Ifaces[i]
			name := f.Name
			if name == "" {
				name = fmt.Sprintf("#%d", f.IfIndex)
			}
			key := fmt.Sprintf("%s/%d", f.Exporter, f.IfIndex)
			inR, outR := f.InWire*8/secs, f.OutWire*8/secs
			inV, outV := um.val(f.InWire, 0), um.val(f.OutWire, 0)
			util, ust, ubar := "—", dim, ""
			if f.Speed > 0 {
				u := math.Max(inR, outR) / f.Speed
				util, ust = fmt.Sprintf("%.0f%%", u*100), ""
				if u > 0.8 {
					util, ust = util+"!", bold+red
				}
				ubar = barCells(math.Min(u, 1), 1, 10)
			}
			rec, rst := "—", dim
			if f.HasCtr {
				rec, rst = pct(f.InDev)+" "+pct(f.OutDev), green
				if math.Max(math.Abs(f.InDev), math.Abs(f.OutDev)) > math.Max(0.02, 2*f.Err) {
					rst = yellow
				}
			}
			mark := "▸ "
			if key == v.ifc {
				mark = "▾ "
			}
			r := row{key: key, cells: []line{txt(mark + name), styled(green, um.fmt(inV)), styled(blue, um.fmt(outV)), styled(ust, util), styled(ust+blue, ubar), styled(rst, rec)},
				target: &target{"iface", key, name}}
			if key == v.ifc {
				r.detail = a.ifaceDetail(v, um, q, f.Exporter, f.IfIndex)
			}
			sec.rows = append(sec.rows, r)
		}
		p.sections = append(p.sections, sec)
	}
	return p
}

// ifaceDetail is what Enter expands under an interface: its trend in and
// out, and its busiest hosts and applications.
func (a *app) ifaceDetail(v view, m measure, q func(map[string]string) map[string][]string, exp string, idx int) []line {
	var out []line
	ind := "    "
	var rc struct {
		Recon struct {
			HasCounters bool      `json:"has_counters"`
			InFlow      []float64 `json:"in_flow"`
			OutFlow     []float64 `json:"out_flow"`
		}
	}
	sw := max(10, v.w-30)
	if err := a.get("recon", q(map[string]string{"exporter": exp, "ifindex": strconv.Itoa(idx)}), &rc); err == nil {
		maxv := 0.0
		for _, x := range append(append([]float64{}, rc.Recon.InFlow...), rc.Recon.OutFlow...) {
			maxv = math.Max(maxv, x)
		}
		out = append(out, line{{dim, ind + fit(a.t("tui.in"), 8) + " "}, {green, spark(rc.Recon.InFlow, sw, maxv)}, {dim, " " + fmtBps(maxv)}})
		out = append(out, line{{dim, ind + fit(a.t("tui.out"), 8) + " "}, {blue, spark(rc.Recon.OutFlow, sw, maxv)}})
	}
	// the interface filter on top of the current filters
	key := fmt.Sprintf("%s/%d", exp, idx)
	fs := v.filters
	add := fmt.Sprintf(`{"f":"iface","v":%q,"neg":false}`, key)
	if fs == "" || fs == "null" || fs == "[]" {
		fs = "[" + add + "]"
	} else {
		fs = strings.TrimSuffix(fs, "]") + "," + add + "]"
	}
	top := func(dim string) []topRow {
		var d struct{ Rows []topRow }
		a.get("topn", query(v.rng, fs, map[string]string{"dim": dim, "limit": "5"}), &d)
		return d.Rows
	}
	hosts, apps := top("client"), top("app")
	colw := max(20, (v.w-8)/2)
	out = append(out, line{{bold, ind + fit(a.t("tui.top_hosts"), colw-1) + spaces(max(0, colw-strWidth(fit(a.t("tui.top_hosts"), colw-1))))}, {bold, a.t("tui.top_apps")}})
	for i := 0; i < max(len(hosts), len(apps)); i++ {
		l := line{{"", ind}}
		left := ""
		if i < len(hosts) {
			left = fmt.Sprintf("%-10s %s", m.fmt(m.val(hosts[i].Wire, hosts[i].Pkts)), a.hostLabel(hosts[i].Key))
		}
		l = append(l, seg{"", fit(left, colw-1) + spaces(max(0, colw-strWidth(fit(left, colw-1))))})
		if i < len(apps) {
			l = append(l, seg{"", fmt.Sprintf("%-10s %s", m.fmt(m.val(apps[i].Wire, apps[i].Pkts)), apps[i].Key)})
		}
		out = append(out, l)
	}
	return out
}

// firstSentence keeps a long explanation to what fits on a title line.
func firstSentence(s string) string {
	for _, sep := range []string{". ", "。", "। ", "؛ "} {
		if i := strings.Index(s, sep); i > 0 {
			return s[:i+len(strings.TrimSpace(sep))]
		}
	}
	return s
}
