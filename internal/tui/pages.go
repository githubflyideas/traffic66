package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type topRow struct {
	Key   string  `json:"key"`
	Key2  string  `json:"key2"`
	Key3  string  `json:"key3"`
	Extra string  `json:"extra"`
	Wire  float64 `json:"wire"`
	Flows float64 `json:"flows"`
	Peers float64 `json:"peers"`
}

var protoNames = map[string]string{"1": "ICMP", "6": "TCP", "17": "UDP", "47": "GRE", "50": "ESP", "58": "ICMPv6", "132": "SCTP"}
var encapNames = []string{"", "GRE", "IPIP", "6in4", "IPv6-in-IPv6", "VXLAN", "GENEVE", "MPLS"}
var dirNames = map[string]string{"1": "outbound", "2": "inbound", "3": "internal", "4": "transit"}

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

// chart draws a column chart of total bits/s, height rows tall, with the
// baseline marked by dots.
func chart(total, base []float64, width, height int) []line {
	if len(total) == 0 || width < 10 {
		return nil
	}
	cols := make([]float64, width)
	bcols := make([]float64, width)
	maxv := 0.0
	for c := 0; c < width; c++ {
		lo := c * len(total) / width
		hi := max(lo+1, (c+1)*len(total)/width)
		s, n, bs := 0.0, 0, 0.0
		for i := lo; i < min(hi, len(total)); i++ {
			s += total[i]
			n++
			if i < len(base) {
				bs += base[i]
			}
		}
		if n > 0 {
			cols[c], bcols[c] = s/float64(n), bs/float64(n)
		}
		maxv = math.Max(maxv, math.Max(cols[c], bcols[c]))
	}
	if maxv == 0 {
		return nil
	}
	eighth := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	var out []line
	for r := height - 1; r >= 0; r-- {
		l := line{}
		for c := 0; c < width; c++ {
			v := cols[c] / maxv * float64(height)
			b := bcols[c] / maxv * float64(height)
			cell := v - float64(r)
			switch {
			case cell >= 1:
				l = append(l, seg{blue, "█"})
			case cell > 0:
				l = append(l, seg{blue, eighth[int(math.Round(cell*8))]})
			case b > float64(r) && b <= float64(r+1):
				l = append(l, seg{dim, "·"})
			default:
				l = append(l, seg{"", " "})
			}
		}
		if r == height-1 {
			l = append(l, seg{dim, " " + fmtBps(maxv)})
		}
		out = append(out, l)
	}
	return out
}

// load fetches and lays out one page.
func (a *app) load(pg, rng, di int, filters []filter, ifc string) page {
	var p page
	fail := func(err error) page {
		p.err = err.Error()
		return p
	}
	q := func(extra map[string]string) map[string][]string { return a.query(rng, filters, extra) }
	a.mu.Lock()
	w := a.w
	a.mu.Unlock()

	switch pageKeys[pg] {
	case "overview":
		var d struct {
			Totals struct{ Wire, Hosts, Peers, Countries float64 }
			Base   struct{ Wire float64 } `json:"base_totals"`
			Basis  string
			Now    float64 `json:"now_bps"`
			Series struct {
				Values [][]float64
			}
			Baseline []float64
			Movers   []struct {
				IP     string  `json:"ip"`
				Delta  float64 `json:"delta_bps"`
				Peer   string
				Port   string
				CC     string
				Threat string
			}
			TopClients []topRow `json:"top_clients"`
		}
		if err := a.get("overview", q(nil), &d); err != nil {
			return fail(err)
		}
		vs := ""
		if d.Base.Wire > 0 {
			k := "kpi.vs_prev"
			if d.Basis == "week" {
				k = "kpi.vs_week"
			}
			vs = a.t(k, "p", pct((d.Totals.Wire-d.Base.Wire)/d.Base.Wire))
		}
		p.top = append(p.top, line{{dim, a.t("kpi.now") + " "}, {bold, fmtBps(d.Now)}, {dim, "  " + vs + "   │ "},
			{dim, a.t("kpi.total", "r", a.t("range."+ranges[rng])) + " "}, {bold, fmtBytes(d.Totals.Wire)}, {dim, "   │ " + a.t("kpi.hosts") + " "},
			{bold, strconv.Itoa(int(d.Totals.Hosts))}, {dim, "   │ " + a.t("kpi.peers") + " "}, {bold, strconv.Itoa(int(d.Totals.Peers))}})
		var total []float64
		for _, s := range d.Series.Values {
			for i, v := range s {
				if i >= len(total) {
					total = append(total, 0)
				}
				total[i] += v
			}
		}
		p.top = append(p.top, chart(total, d.Baseline, w-12, 7)...)
		p.top = append(p.top, txt(""))
		mv := section{title: a.t("ov.movers"), cols: []column{{head: a.t("field.ip"), width: 34}, {head: "", width: 12, right: true}, {head: ""}}}
		for _, m := range d.Movers {
			reason := ""
			if m.Peer != "" {
				reason = a.t("ov.mover_reason", "peer", a.hostLabel(m.Peer), "port", m.Port, "cc", a.cc(m.CC))
			}
			r := row{cells: []line{a.host(m.IP), styled(red, "+"+fmtBps(m.Delta)), txt(reason)}, target: ipTarget(a, m.IP)}
			if m.Threat != "" {
				r.cells[2] = append(r.cells[2], seg{red, "  [" + m.Threat + "]"})
			}
			mv.rows = append(mv.rows, r)
		}
		tc := section{title: a.t("ov.top_clients"), cols: []column{{head: a.t("col.client"), width: 34}, {head: a.t("col.traffic"), width: 10, right: true}, {head: ""}, {head: a.t("col.vs_week"), width: 10, right: true}}}
		maxw := 1.0
		for _, r := range d.TopClients {
			maxw = math.Max(maxw, r.Wire)
		}
		for _, r := range d.TopClients {
			b, _ := strconv.ParseFloat(r.Extra, 64)
			ch := a.t("ov.mover_new")
			if b > 0 {
				ch = pct((r.Wire - b) / b)
			}
			tc.rows = append(tc.rows, row{cells: []line{a.host(r.Key), txt(fmtBytes(r.Wire)), styled(blue, barCells(r.Wire, maxw, 30)), txt(ch)}, target: ipTarget(a, r.Key)})
		}
		p.sections = []section{mv, tc}

	case "topn":
		var d struct{ Rows []topRow }
		if err := a.get("topn", q(map[string]string{"dim": dims[di], "limit": "66"}), &d); err != nil {
			return fail(err)
		}
		tabs := line{{dim, "← → "}}
		for i, k := range dims {
			if i == di {
				tabs = append(tabs, seg{rev, " " + a.t("dim."+k) + " "})
			} else {
				tabs = append(tabs, seg{"", " " + a.t("dim."+k) + " "})
			}
		}
		p.top = append(p.top, tabs, txt(""))
		sec := section{title: a.t("topn.title", "n", "66"), cols: []column{{head: "#", width: 3, right: true}, {head: a.t("dim." + dims[di])}, {head: a.t("col.traffic"), width: 10, right: true}, {head: "%", width: 6, right: true}, {head: "", width: 24}, {head: a.t("col.flows"), width: 9, right: true}}}
		tot, maxw := 0.0, 1.0
		for _, r := range d.Rows {
			tot += r.Wire
			maxw = math.Max(maxw, r.Wire)
		}
		for i, r := range d.Rows {
			var cell line
			var tg *target
			switch dims[di] {
			case "client", "server", "exporter":
				cell, tg = a.host(r.Key), ipTarget(a, r.Key)
				if dims[di] == "exporter" {
					tg.field = "exporter"
				}
			case "conv":
				cell = append(append(a.host(r.Key), seg{dim, " → "}), a.host(r.Key2)...)
				cell = append(cell, seg{dim, " " + r.Key3})
				tg = ipTarget(a, r.Key)
			case "country":
				cell, tg = txt(a.cc(r.Key)), &target{"country", r.Key, a.cc(r.Key)}
			case "asn":
				cell, tg = line{{"", "AS" + r.Key + " "}, {dim, r.Extra}}, &target{"asn", r.Key, "AS" + r.Key}
			case "port":
				cell, tg = line{{"", r.Key + " "}, {dim, r.Extra}}, &target{"port", r.Key, r.Key}
			case "encap":
				n, _ := strconv.Atoi(r.Key)
				lab := a.t("encap.none")
				if n > 0 && n < len(encapNames) {
					lab = encapNames[n]
				}
				cell, tg = txt(lab), &target{"encap", r.Key, lab}
			case "vlan":
				lab := r.Key
				if r.Key == "0" {
					lab = a.t("vlan.none")
				}
				cell, tg = txt(lab), &target{"vlan", r.Key, lab}
			default:
				cell, tg = txt(r.Key), &target{dims[di], r.Key, r.Key}
			}
			sec.rows = append(sec.rows, row{cells: []line{styled(dim, strconv.Itoa(i+1)), cell, txt(fmtBytes(r.Wire)),
				styled(dim, fmt.Sprintf("%.1f", r.Wire/math.Max(1, tot)*100)), styled(blue, barCells(r.Wire, maxw, 24)), txt(strconv.Itoa(int(r.Flows)))}, target: tg})
		}
		p.sections = []section{sec}

	case "sankey":
		var d struct {
			SegApp []struct {
				S, T string
				V    float64
			} `json:"seg_app"`
			AppCC []struct {
				S, T string
				V    float64
			} `json:"app_cc"`
		}
		if err := a.get("sankey", q(nil), &d); err != nil {
			return fail(err)
		}
		lab := func(s string) string {
			if s == "__other__" {
				return a.t("other")
			}
			return s
		}
		s1 := section{title: a.t("field.segment") + " → " + a.t("field.app"), cols: []column{{head: a.t("field.segment"), width: 22}, {head: a.t("field.app"), width: 18}, {head: a.t("col.traffic"), width: 10, right: true}, {head: ""}}}
		m1 := 1.0
		for _, l := range d.SegApp {
			m1 = math.Max(m1, l.V)
		}
		for _, l := range d.SegApp {
			s1.rows = append(s1.rows, row{cells: []line{txt(lab(l.S)), txt(lab(l.T)), txt(fmtBytes(l.V)), styled(blue, barCells(l.V, m1, 30))}, target: &target{"segment", l.S, l.S}})
		}
		s2 := section{title: a.t("field.app") + " → " + a.t("field.country"), cols: []column{{head: a.t("field.app"), width: 22}, {head: a.t("field.country"), width: 18}, {head: a.t("col.traffic"), width: 10, right: true}, {head: ""}}}
		m2 := 1.0
		for _, l := range d.AppCC {
			m2 = math.Max(m2, l.V)
		}
		for _, l := range d.AppCC {
			s2.rows = append(s2.rows, row{cells: []line{txt(lab(l.S)), txt(a.cc(l.T)), txt(fmtBytes(l.V)), styled(blue, barCells(l.V, m2, 30))}, target: &target{"app", l.S, l.S}})
		}
		p.sections = []section{s1, s2}

	case "geo":
		var c, as struct{ Rows []topRow }
		if err := a.get("topn", q(map[string]string{"dim": "country", "limit": "66"}), &c); err != nil {
			return fail(err)
		}
		a.get("topn", q(map[string]string{"dim": "asn", "limit": "66"}), &as)
		s1 := section{title: a.t("geo.countries"), cols: []column{{head: a.t("col.country"), width: 26}, {head: a.t("col.traffic"), width: 10, right: true}, {head: ""}}}
		m := 1.0
		for _, r := range c.Rows {
			m = math.Max(m, r.Wire)
		}
		for _, r := range c.Rows {
			if r.Key == "__internal__" {
				continue
			}
			s1.rows = append(s1.rows, row{cells: []line{txt(a.cc(r.Key)), txt(fmtBytes(r.Wire)), styled(blue, barCells(r.Wire, m, 30))}, target: &target{"country", r.Key, a.cc(r.Key)}})
		}
		s2 := section{title: a.t("geo.as"), cols: []column{{head: a.t("col.asn"), width: 10}, {head: a.t("col.org"), width: 28}, {head: a.t("col.traffic"), width: 10, right: true}, {head: ""}}}
		m = 1.0
		for _, r := range as.Rows {
			m = math.Max(m, r.Wire)
		}
		for _, r := range as.Rows {
			s2.rows = append(s2.rows, row{cells: []line{txt("AS" + r.Key), txt(r.Extra), txt(fmtBytes(r.Wire)), styled(blue, barCells(r.Wire, m, 24))}, target: &target{"asn", r.Key, "AS" + r.Key}})
		}
		p.sections = []section{s1, s2}

	case "threats":
		var d struct {
			Rows  []topRow
			Lists map[string]int
		}
		if err := a.get("threats", q(map[string]string{"limit": "66"}), &d); err != nil {
			return fail(err)
		}
		if len(d.Lists) == 0 {
			p.top = append(p.top, styled(yellow, a.t("thr.no_lists")))
		}
		sec := section{title: a.t("thr.table"), cols: []column{{head: a.t("col.list"), width: 12}, {head: a.t("col.client"), width: 30}, {head: a.t("col.server"), width: 30}, {head: a.t("col.direction"), width: 10}, {head: a.t("col.traffic"), width: 10, right: true}}}
		for _, r := range d.Rows {
			cc, dir, _ := strings.Cut(r.Extra, "|")
			st := yellow
			if dir == "1" {
				st = red
			}
			_ = cc
			sec.rows = append(sec.rows, row{cells: []line{styled(st, r.Key), a.host(r.Key2), a.host(r.Key3), styled(st, a.t("dir."+dir)), styled(st, fmtBytes(r.Wire))}, target: ipTarget(a, r.Key2)})
		}
		p.sections = []section{sec}

	case "records":
		var d struct {
			Rows []struct {
				Client, Server, App, CC, Exporter string
				Port                              int `json:"port"`
				Proto                             int `json:"proto"`
				Wire                              float64
				Encap                             int
				Dir                               int
			}
		}
		if err := a.get("records", q(map[string]string{"limit": "200"}), &d); err != nil {
			return fail(err)
		}
		sec := section{title: a.t("rec.title"), cols: []column{{head: a.t("rc.client"), width: 28}, {head: a.t("rc.server"), width: 28}, {head: a.t("rc.port"), width: 10}, {head: a.t("rc.app"), width: 14}, {head: a.t("rc.country"), width: 8}, {head: a.t("rc.encap"), width: 8}, {head: a.t("rc.traffic"), width: 10, right: true}, {head: a.t("rc.device")}}}
		for _, r := range d.Rows {
			pr := protoNames[strconv.Itoa(r.Proto)]
			enc := ""
			if r.Encap > 0 && r.Encap < len(encapNames) {
				enc = encapNames[r.Encap]
			}
			cc := r.CC
			if r.Dir == 3 {
				cc = a.t("internal")
			}
			sec.rows = append(sec.rows, row{cells: []line{a.host(r.Client), a.host(r.Server), txt(fmt.Sprintf("%d/%s", r.Port, strings.ToLower(pr))), txt(r.App), txt(cc), txt(enc), txt(fmtBytes(r.Wire)), a.host(r.Exporter)}, target: ipTarget(a, r.Server)})
		}
		p.sections = []section{sec}

	case "ifaces":
		var d struct {
			Ifaces []struct {
				Exporter, Name, Device string
				IfIndex                int     `json:"ifindex"`
				InDev                  float64 `json:"in_dev"`
				OutDev                 float64 `json:"out_dev"`
				Err                    float64 `json:"stat_err"`
				HasCtr                 bool    `json:"has_counters"`
			}
		}
		if err := a.get("ifaces", q(nil), &d); err != nil {
			return fail(err)
		}
		sec := section{title: a.t("if.title"), cols: []column{{head: a.t("if.title"), width: 28}, {head: a.t("col.device"), width: 26}, {head: a.t("if.in"), width: 9, right: true}, {head: a.t("if.out"), width: 9, right: true}, {head: ""}}}
		selected := ifc
		for i, f := range d.Ifaces {
			name := f.Name
			if name == "" {
				name = fmt.Sprintf("#%d", f.IfIndex)
			}
			key := fmt.Sprintf("%s/%d", f.Exporter, f.IfIndex)
			if selected == "" && i == 0 {
				selected = key
			}
			in, out, st := "—", "—", dim
			if f.HasCtr {
				in, out = pct(f.InDev), pct(f.OutDev)
				st = green
				if math.Max(math.Abs(f.InDev), math.Abs(f.OutDev)) > math.Max(0.02, 2*f.Err) {
					st = yellow
				}
			}
			mark := "  "
			if key == selected {
				mark = "● "
			}
			dev := f.Device
			if dev == "" {
				dev = f.Exporter
			}
			sec.rows = append(sec.rows, row{cells: []line{txt(mark + name), styled(dim, dev), styled(st, in), styled(st, out), txt("")}, target: &target{"iface", key, name}})
		}
		p.sections = []section{sec}
		if exp, idx, ok := strings.Cut(selected, "/"); ok {
			var rc struct {
				Recon struct {
					HasCounters bool      `json:"has_counters"`
					InDev       float64   `json:"in_dev"`
					InErr       float64   `json:"in_stat_err"`
					InCounter   []float64 `json:"in_counter"`
					InFlow      []float64 `json:"in_flow"`
				}
				Name, Device string
			}
			if err := a.get("recon", q(map[string]string{"exporter": exp, "ifindex": idx}), &rc); err == nil {
				r := rc.Recon
				maxv := 0.0
				for _, v := range append(append([]float64{}, r.InCounter...), r.InFlow...) {
					maxv = math.Max(maxv, v)
				}
				title := rc.Name
				if title == "" {
					title = "#" + idx
				}
				p.top = append(p.top, line{{bold, title}, {dim, " · " + rc.Device + " · " + a.t("if.in")}})
				p.top = append(p.top, line{{blue, fit(a.t("if.flow"), 16)}, {blue, spark(r.InFlow, w-20, maxv)}})
				if r.HasCounters {
					p.top = append(p.top, line{{yellow, fit(a.t("if.counter"), 16)}, {yellow, spark(r.InCounter, w-20, maxv)}})
					verdict := a.t("if.ok", "dev", pct(r.InDev), "err", fmt.Sprintf("±%.1f%%", r.InErr*100))
					st := green
					if math.Abs(r.InDev) > math.Max(0.02, 2*r.InErr) {
						st = yellow
						k := "if.high"
						if r.InDev < 0 {
							k = "if.low"
						}
						verdict = a.t(k, "dev", fmt.Sprintf("%.1f%%", math.Abs(r.InDev)*100), "causes", a.t("cause.unsampled_iface"))
					}
					p.top = append(p.top, styled(st, verdict))
				} else {
					p.top = append(p.top, styled(dim, a.t("if.no_counters")))
				}
				p.top = append(p.top, styled(dim, "Enter: "+a.t("if.title")), txt(""))
			}
		}

	case "sources":
		var d struct {
			Sources []struct {
				Exporter, Proto, Name, Sampling, Status string
				SamplingState                           string  `json:"sampling_state"`
				RecPS                                   float64 `json:"rec_per_sec"`
				LostPct                                 float64 `json:"lost_pct"`
				Pending                                 int
				Issues                                  []string
				ClockSkew                               float64 `json:"clock_skew_ns"`
			}
		}
		if err := a.get("sources", q(nil), &d); err != nil {
			return fail(err)
		}
		sec := section{title: a.t("src.title"), cols: []column{{head: a.t("col.device"), width: 30}, {head: a.t("col.proto"), width: 11}, {head: a.t("col.rate"), width: 9, right: true}, {head: a.t("col.sampling"), width: 14}, {head: a.t("col.lost"), width: 8, right: true}, {head: a.t("col.status")}}}
		for _, s := range d.Sources {
			st := green
			switch s.Status {
			case "warn":
				st = yellow
			case "bad":
				st = red
			}
			name := line{{"", s.Exporter}}
			if s.Name != "" {
				name = line{{"", s.Name + " "}, {dim, s.Exporter}}
			}
			samp := s.Sampling
			if samp == "" {
				samp = "?"
			}
			r := row{cells: []line{name, txt(s.Proto), txt(fmt.Sprintf("%.1f", s.RecPS)), txt(samp), txt(fmt.Sprintf("%.2f%%", s.LostPct)), styled(st, a.t("st."+s.Status))},
				target: &target{"exporter", s.Exporter, s.Exporter}}
			for _, is := range s.Issues {
				r.detail = append(r.detail, styled(st, "  "+a.t("issue."+is, "n", strconv.Itoa(s.Pending), "p", fmt.Sprintf("%.1f%%", s.LostPct), "s", fmt.Sprintf("%.0f s", math.Abs(s.ClockSkew)/1e9))))
			}
			sec.rows = append(sec.rows, r)
		}
		p.sections = []section{sec}
		if len(d.Sources) == 0 {
			p.top = append(p.top, styled(dim, a.t("empty.first")))
		}
	}
	return p
}
