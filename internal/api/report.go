package api

import (
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// The report is the one page somebody prints and hands to a customer or a
// manager, so every number on it has to say where it came from. Two
// measurements of the same interface exist here and they do not agree:
//
//   - the device's own interface counters, read over SNMP into five-minute
//     rate buckets. They are what a carrier bills against and they count
//     every byte, sampled or not.
//   - the flows, which are sampled and therefore an estimate, but are the
//     only measurement for an interface no device reports counters for.
//
// Each interface row says which of the two it is (Source), rather than
// mixing them into one number nobody can argue with. Utilisation is only
// given where the interface speed is known: a percentage of an unknown
// capacity is not a figure, it is a guess with a unit.
const (
	srcCounters = "counters" // the device's interface counters
	srcFlows    = "flows"    // sampled flow records
)

// reportIface is one interface's figures over the period.
type reportIface struct {
	Exporter string `json:"exporter"`
	IfIndex  uint32 `json:"ifindex"`
	Name     string `json:"name,omitempty"`
	Device   string `json:"device,omitempty"`
	Speed    uint64 `json:"speed,omitempty"` // bits/s, 0 when unknown
	Source   string `json:"source"`

	InWire  uint64 `json:"in_wire"` // bytes on the wire, from the flows
	OutWire uint64 `json:"out_wire"`

	InAvg   float64 `json:"in_avg"` // bits/s
	OutAvg  float64 `json:"out_avg"`
	InPeak  float64 `json:"in_peak"`
	OutPeak float64 `json:"out_peak"`
	In95    float64 `json:"in95"`
	Out95   float64 `json:"out95"`

	// Util* are per cent of Speed, omitted when the speed is unknown.
	UtilAvg  float64 `json:"util_avg,omitempty"`
	UtilPeak float64 `json:"util_peak,omitempty"`
	Util95   float64 `json:"util95,omitempty"`

	// Coverage is the share of the period's buckets the counters measured,
	// for counter rows: a 95th percentile taken over half the period is not
	// wrong, but it is not the period either.
	Coverage float64 `json:"coverage,omitempty"`
}

// reportOut is the whole report. It is a struct rather than a map so that
// the printable page and the CSV export read the same field names.
type reportOut struct {
	From      int64  `json:"from"`
	To        int64  `json:"to"`
	SpanS     int64  `json:"span_s"`
	Generated int64  `json:"generated"`
	Iface     string `json:"if,omitempty"`

	Totals  store.Totals `json:"totals"`
	AvgBps  float64      `json:"avg_bps"`
	PeakBps float64      `json:"peak_bps"`
	PeakAt  int64        `json:"peak_at,omitempty"`
	P95Bps  float64      `json:"p95_bps"`
	// Step is the width of the buckets the peak and the percentile above
	// were taken over, in seconds. A peak over a one-minute bucket and a
	// peak over an hour are different claims.
	Step int64 `json:"step"`

	Ifaces []reportIface `json:"ifaces"`

	TopHosts     []store.TopRow `json:"top_hosts"`
	TopApps      []store.TopRow `json:"top_apps"`
	TopCountries []store.TopRow `json:"top_countries"`

	Findings map[string]int  `json:"findings"`
	Top      []store.Finding `json:"top_findings,omitempty"`

	Notice string `json:"notice,omitempty"`
}

// percentile returns the 95th percentile of v by nearest rank, which is
// the sample a device's own counters could be checked against rather than
// an interpolation between two of them (see store.Bills).
func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(math.Ceil(p*float64(len(s)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}

// totalSeries sums every application's series into one line of bits/s and
// drops the bucket that is still filling: the last minute of a live range
// is half counted, and a report that takes its peak or its percentile from
// that minute understates both.
func (s *Server) totalSeries(q store.Query) (step int64, times []int64, bps []float64, err error) {
	se, err := s.Store.SeriesByApp(q, 1<<30)
	if err != nil {
		return 0, nil, nil, err
	}
	for i, t := range se.Times {
		if se.SettledTo > 0 && t >= se.SettledTo {
			break
		}
		sum := 0.0
		for _, v := range se.Values {
			sum += v[i]
		}
		times, bps = append(times, t), append(bps, sum)
	}
	return se.Step, times, bps, nil
}

// report answers GET /api/report?from&to&if — the numbers behind the
// printable report: traffic over the period, what each interface carried
// and how full it was, what the top of the period was made of, and what
// the detection rules found while it ran.
func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	only := r.URL.Query().Get("if")
	if only != "" {
		// One interface: every flow figure below is then this interface's,
		// including the top hosts and applications, which is what somebody
		// reporting on one circuit asked for.
		if err := checkIface(only); err != nil {
			fail(w, err)
			return
		}
		q.Filters = append(q.Filters, store.Filter{Field: "iface", Value: only})
	}
	span := q.To.Sub(q.From)
	out := reportOut{
		From: q.From.UnixMilli(), To: q.To.UnixMilli(), SpanS: int64(span.Seconds()),
		Generated: time.Now().UTC().UnixMilli(), Iface: only, Notice: s.notice(),
		Findings: map[string]int{"high": 0, "medium": 0, "low": 0},
	}
	if out.Totals, err = s.Store.Totals(q); err != nil {
		fail(w, err)
		return
	}
	if span > 0 {
		out.AvgBps = float64(out.Totals.Wire) * 8 / span.Seconds()
	}
	step, times, bps, err := s.totalSeries(q)
	if err != nil {
		fail(w, err)
		return
	}
	out.Step = step
	for i, v := range bps {
		if v > out.PeakBps {
			out.PeakBps, out.PeakAt = v, times[i]
		}
	}
	out.P95Bps = percentile(bps, 0.95)

	list, err := s.Store.Interfaces(q)
	if err != nil {
		fail(w, err)
		return
	}
	for _, f := range list {
		k := ifKey(f.Exporter, f.IfIndex)
		if only != "" && k != only {
			continue
		}
		it := reportIface{Exporter: f.Exporter, IfIndex: f.IfIndex, Speed: f.Speed,
			InWire: f.InWire, OutWire: f.OutWire, Source: srcFlows}
		if a, err := netip.ParseAddr(f.Exporter); err == nil && s.Inv != nil {
			ifc := s.Inv.Iface(a, f.IfIndex)
			it.Name, it.Device = ifc.Name, s.Inv.Device(a)
			if it.Speed == 0 {
				it.Speed = ifc.Speed
			}
		}
		if c := s.counterFigures(q, f); c != nil {
			it.Source = srcCounters
			it.InAvg, it.InPeak, it.In95 = c.in.avg, c.in.peak, c.in.p95
			it.OutAvg, it.OutPeak, it.Out95 = c.out.avg, c.out.peak, c.out.p95
			it.Coverage = c.coverage
		} else if span > 0 {
			// No counters: the flows are the only measurement there is.
			// They are sampled, so the row says so through Source.
			it.InAvg = float64(f.InWire) * 8 / span.Seconds()
			it.OutAvg = float64(f.OutWire) * 8 / span.Seconds()
			qi := q
			if only == "" {
				qi.Filters = append(append([]store.Filter{}, q.Filters...), store.Filter{Field: "iface", Value: k})
			}
			if _, _, b, err := s.totalSeries(qi); err == nil {
				for _, v := range b {
					it.InPeak = math.Max(it.InPeak, v)
				}
				it.In95 = percentile(b, 0.95)
				// One series per interface covers both directions; it is
				// reported as the inbound figure with the outbound left
				// empty rather than counted twice.
			}
		}
		if it.Speed > 0 {
			pc := func(v float64) float64 { return v / float64(it.Speed) * 100 }
			it.UtilAvg = pc(math.Max(it.InAvg, it.OutAvg))
			it.UtilPeak = pc(math.Max(it.InPeak, it.OutPeak))
			it.Util95 = pc(math.Max(it.In95, it.Out95))
		}
		out.Ifaces = append(out.Ifaces, it)
		if len(out.Ifaces) >= 50 {
			break
		}
	}

	top := func(dim string) []store.TopRow {
		rows, err := s.Store.TopN(q, dim, 10)
		if err != nil {
			return nil
		}
		return rows
	}
	out.TopHosts, out.TopApps, out.TopCountries = top("client"), top("app"), top("country")

	fs, err := s.Store.Findings(store.FindingQuery{From: q.From, To: q.To, Status: "all", Limit: 1000})
	if err != nil {
		fail(w, err)
		return
	}
	name := map[int]string{3: "high", 2: "medium", 1: "low"}
	for _, f := range fs {
		if n, ok := name[f.Sev]; ok {
			out.Findings[n]++
		}
	}
	if len(fs) > 20 {
		fs = fs[:20]
	}
	out.Top = fs
	writeJSON(w, http.StatusOK, out)
}

// checkIface refuses an interface that is not written as the store's
// filter expects it (exporter/ifindex), here rather than inside a query,
// so the caller is told what is wrong with the request instead of being
// handed a database error.
func checkIface(v string) error {
	e, idx, ok := strings.Cut(v, "/")
	if !ok || e == "" {
		return fmt.Errorf("if must be written exporter/ifindex, not %q", v)
	}
	if _, err := netip.ParseAddr(e); err != nil {
		return fmt.Errorf("if: %q is not an exporter address", e)
	}
	if _, err := strconv.ParseUint(idx, 10, 32); err != nil {
		return fmt.Errorf("if: %q is not an interface number", idx)
	}
	return nil
}

type figures struct{ avg, peak, p95 float64 }

type counterRow struct {
	in, out  figures
	coverage float64
}

// counterFigures is an interface's average, peak and 95th percentile from
// the device's own counters, over the same buckets the interface page
// draws them in; nil when the device reported no counters for it in the
// period. Buckets with no measurement are left out, not counted as zero.
func (s *Server) counterFigures(q store.Query, f store.IfaceInfo) *counterRow {
	if !f.Counters {
		return nil
	}
	rc, err := s.Store.Recon(q, f.Exporter, f.IfIndex)
	if err != nil || !rc.HasCounters {
		return nil
	}
	of := func(v []float64) (figures, int) {
		var got []float64
		sum := 0.0
		for _, x := range v {
			if x >= 0 {
				got = append(got, x)
				sum += x
			}
		}
		if len(got) == 0 {
			return figures{}, 0
		}
		r := figures{avg: sum / float64(len(got)), p95: percentile(got, 0.95)}
		for _, x := range got {
			r.peak = math.Max(r.peak, x)
		}
		return r, len(got)
	}
	in, n := of(rc.InCounter)
	out, _ := of(rc.OutCounter)
	if n == 0 {
		return nil
	}
	return &counterRow{in: in, out: out, coverage: float64(n) / float64(len(rc.InCounter))}
}

func ifKey(exporter string, ifindex uint32) string {
	return exporter + "/" + strconv.FormatUint(uint64(ifindex), 10)
}
