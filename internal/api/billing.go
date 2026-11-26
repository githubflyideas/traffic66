package api

import (
	"encoding/csv"
	"errors"
	"math"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// Traffic statistics: the traffic of one VIP, or of every VIP of one
// product line, with inbound and outbound added together — the figure a
// business is billed from. It is a registered feature as a whole: an
// unregistered installation is answered not_registered for every
// object, one VIP as much as a group or a subnet, before anything is
// resolved or counted.
//
// The gate is the registration state itself, not a licence feature name:
// the Features list in the licence package is signed as a whole and
// shared with the issuing repository, so a new name is not added for
// this. The aggregation is store.Billing — or store.BillingSubnet for
// a subnet — the one counting behind the statistics page, its CSV and
// the read-only accounts' page alike.
//
// A read-only account (readonly.go) sees this and nothing else, and
// only its own bound object, over a window of at most fourteen days,
// at most sixty queries a minute. There is no way to these figures
// without signing in: the platform links' key-in-the-URL mechanism is
// for embedding a chart in another platform, and it stays separate.

// roMaxSpan is the widest window a read-only account may be asked for.
// Checking a bill does not need the archive: fourteen days reaches back
// past any single invoice period a customer disputes, and a wider ask
// is refused as bad_range rather than quietly cut.
const roMaxSpan = 14 * 24 * time.Hour

// roPerMinute is the query limit of one read-only account, counted per
// account and per client address both.
const roPerMinute = 60

// registered reports whether this installation is registered.
func (s *Server) registered() bool {
	// the demo shows every page: its data is simulated, so nothing it
	// shows can be put to use, and the registered pages are what an
	// evaluator most wants to see
	return s.Demo || (s.License != nil && s.License.State().Registered())
}

// paidOnly wraps a route that belongs to the registered pages: traffic
// statistics, interface check, findings and the inventory. Unregistered
// it is answered not_registered with the feature's name, before the
// handler runs.
func (s *Server) paidOnly(feature string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.registeredOnly(w, feature) {
			h(w, r)
		}
	}
}

// registeredOnly answers whether this installation is registered, and
// writes the not_registered answer when it is not — the same answer
// allowed() writes for a feature-gated call.
func (s *Server) registeredOnly(w http.ResponseWriter, feature string) bool {
	if s.registered() {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_registered", "feature": feature})
	return false
}

// billingObject is the selection a billing query is about: one VIP,
// one product line with all of its VIPs, or one subnet.
type billingObject struct {
	Kind   string // "vip", "product" or "subnet"
	Name   string // the VIP's address, the product line's name, or the subnet's masked CIDR
	Addrs  []netip.Addr
	Prefix netip.Prefix // the subnet, when Kind is "subnet"
}

// resolveBillingObject turns the subnet / product / vip parameters
// into the selection, reading the product lines from the inventory as
// it stands now — a line edited a minute ago bills by its new
// membership. A subnet is masked: 10.0.0.9/24 asks for 10.0.0.0/24,
// the range the address stands in.
func (s *Server) resolveBillingObject(v map[string]string) (billingObject, error) {
	if sn := v["subnet"]; sn != "" {
		p, err := netip.ParsePrefix(sn)
		if err != nil {
			return billingObject{}, errors.New("bad_subnet")
		}
		p = p.Masked()
		return billingObject{Kind: "subnet", Name: p.String(), Prefix: p}, nil
	}
	if p := v["product"]; p != "" {
		prod, ok := s.Inv.Product(p)
		if !ok {
			return billingObject{}, errors.New("unknown_product")
		}
		return billingObject{Kind: "product", Name: prod.Name, Addrs: prod.VIPs}, nil
	}
	if a := v["vip"]; a != "" {
		addrs, err := parseVIPs(a)
		if err != nil {
			return billingObject{}, err
		}
		names := make([]string, len(addrs))
		for i, x := range addrs {
			names[i] = x.String()
		}
		return billingObject{Kind: "vip", Name: strings.Join(names, ","), Addrs: addrs}, nil
	}
	return billingObject{}, errors.New("no_object")
}

// billingOut is the answer of a billing query: the object's figures over
// the window, and the marks that say where the numbers came from.
type billingOut struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	VIPs       []string `json:"vips"`
	VIPsTotal  int      `json:"vips_total,omitempty"` // the subnet's addresses, when the list is capped
	From       int64    `json:"from"`
	To         int64    `json:"to"`
	Registered bool     `json:"registered"`
	Notice     string   `json:"notice,omitempty"`
	*store.Billing
	AvgBps  float64 `json:"avg_bps"`
	PeakBps float64 `json:"peak_bps"`
	PeakAt  int64   `json:"peak_at,omitempty"`
	// P95Bps is the 95th percentile of the total rate over the settled
	// buckets, the figure bandwidth is commonly billed on; P95Step is the
	// bucket it was taken over, in seconds, because a 95th percentile is
	// only comparable between bills taken over the same bucket width.
	P95Bps  float64 `json:"p95_bps"`
	P95Step int64   `json:"p95_step,omitempty"`
}

// billingFor runs the one aggregation behind the statistics page, its
// CSV and the read-only page. The settled window is what the headline
// figures describe: AvgBps is the settled total over the settled span.
func (s *Server) billingFor(obj billingObject, q store.Query) (*billingOut, error) {
	var b *store.Billing
	var err error
	if obj.Kind == "subnet" {
		b, err = s.Store.BillingSubnet(q, obj.Prefix)
	} else {
		b, err = s.Store.Billing(q, obj.Addrs)
	}
	if err != nil {
		return nil, err
	}
	out := &billingOut{Kind: obj.Kind, Name: obj.Name, Billing: b,
		From: q.From.UnixMilli(), To: q.To.UnixMilli(),
		Registered: s.registered(),
		Notice:     s.notice()}
	if obj.Kind == "subnet" {
		// the addresses are the ones the traffic names, not an input:
		// the member split's, largest first, capped like it is
		for _, m := range b.Members {
			out.VIPs = append(out.VIPs, m.Addr)
		}
		out.VIPsTotal = b.MembersTotal
	} else {
		for _, a := range obj.Addrs {
			out.VIPs = append(out.VIPs, a.String())
		}
	}
	settledEnd := q.To
	if b.SettledTo > 0 {
		settledEnd = time.UnixMilli(b.SettledTo).UTC()
	}
	if span := settledEnd.Sub(q.From).Seconds(); span > 0 {
		out.AvgBps = float64(b.Bytes) * 8 / span
	}
	// peak and 95th percentile over the settled buckets only:
	// a bucket still filling would pull every one of them down
	settledMs := settledEnd.UnixMilli()
	stepMs := b.Step * 1000
	var vals []float64
	for i, v := range b.Total {
		if stepMs > 0 && b.Times[i]+stepMs > settledMs {
			continue
		}
		// nor buckets from before there was any data: zeros that are
		// "not recorded", not "idle", would drag the percentile down
		if b.DataFrom > 0 && b.Times[i]+stepMs <= b.DataFrom {
			continue
		}
		vals = append(vals, v)
		if v > out.PeakBps {
			out.PeakBps, out.PeakAt = v, b.Times[i]
		}
	}
	if len(vals) > 0 {
		out.P95Bps, out.P95Step = percentile95(vals), b.Step
	}
	return out, nil
}

// resolveKindObject resolves the (kind, object) pair a read-only
// account is bound to, or a creation asks for.
func (s *Server) resolveKindObject(kind, object string) (billingObject, error) {
	switch kind {
	case "product":
		return s.resolveBillingObject(map[string]string{"product": object})
	case "vip":
		return s.resolveBillingObject(map[string]string{"vip": object})
	}
	return billingObject{}, errors.New("bad_kind")
}

// billMaxSpan is the widest window one billing query may cover: a
// calendar month, and a day over for the viewer's zone. A bill is written
// per month; a wider query over a many-VIP line or a wide subnet would
// only cost the database time every other page is waiting on.
const billMaxSpan = 32 * 24 * time.Hour

// billingRange reads an explicit from/to window for a billing query.
// Unlike the chart pages it does not round the window to whole hours:
// a bill is written for the window that was asked for, to the minute the
// records are kept in.
func billingRange(r *http.Request) (store.Query, error) {
	v := r.URL.Query()
	f, t := v.Get("from"), v.Get("to")
	if f == "" || t == "" {
		// a whole calendar month, this one or the one named by month=,
		// cut on the viewer's clock
		zo := zoneOffset(v.Get("tz"))
		zone := time.FixedZone("", int(zo.Seconds()))
		now := time.Now().In(zone)
		y, m := now.Year(), now.Month()
		if mv := v.Get("month"); mv != "" {
			tm, err := time.Parse("2006-01", mv)
			if err != nil {
				return store.Query{}, errBadRange
			}
			y, m = tm.Year(), tm.Month()
		}
		from := time.Date(y, m, 1, 0, 0, 0, 0, zone)
		return store.Query{From: from.UTC(), To: from.AddDate(0, 1, 0).UTC(), ZoneOffset: zo}, nil
	}
	fm, err1 := strconv.ParseInt(f, 10, 64)
	tm, err2 := strconv.ParseInt(t, 10, 64)
	if d := tm - fm; err1 != nil || err2 != nil || d <= 0 || d > billMaxSpan.Milliseconds() {
		return store.Query{}, errBadRange
	}
	return store.Query{From: time.UnixMilli(fm).UTC(), To: time.UnixMilli(tm).UTC(),
		ZoneOffset: zoneOffset(v.Get("tz"))}, nil
}

// billingRequest settles what a billing call asks for: the object and
// the window, with the read-only caller's narrower rules applied — its
// bound object only, a window of at most fourteen days, and its rate
// limit — after the registration check, which comes first: the whole
// page is a registered feature. It writes the refusal itself and
// reports false.
func (s *Server) billingRequest(w http.ResponseWriter, r *http.Request) (billingObject, store.Query, bool) {
	if !s.registeredOnly(w, "billing") {
		return billingObject{}, store.Query{}, false
	}
	v := r.URL.Query()
	params := map[string]string{"product": v.Get("product"), "vip": v.Get("vip"), "subnet": v.Get("subnet")}
	ro, isRO := s.roRole(userOf(r))
	if isRO {
		if params["subnet"] != "" {
			// a subnet is not an object a read-only account can be
			// bound to, so asking for one is outside its scope,
			// whatever the installation's registration state
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "readonly_scope"})
			return billingObject{}, store.Query{}, false
		}
		bound, err := s.resolveKindObject(ro.Kind, ro.Object)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return billingObject{}, store.Query{}, false
		}
		// The scope is settled on what was asked for, before any of it
		// is resolved: another object is a refusal, whether or not it
		// exists, and only the bound object may pass.
		if p := params["product"]; p != "" && (bound.Kind != "product" || p != bound.Name) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "readonly_scope"})
			return billingObject{}, store.Query{}, false
		}
		if a := params["vip"]; a != "" {
			// a read-only account is bound to one address; a list is
			// never its object, whatever it holds
			if strings.ContainsAny(a, ",， ") {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "readonly_scope"})
				return billingObject{}, store.Query{}, false
			}
			addr, perr := netip.ParseAddr(a)
			if perr == nil && (bound.Kind != "vip" || addr.Unmap().String() != bound.Name) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "readonly_scope"})
				return billingObject{}, store.Query{}, false
			}
		}
		if params["product"] == "" && params["vip"] == "" {
			// asking for nothing in particular is asking for the bound
			// object: it is the only thing this account can mean
			params[bound.Kind] = bound.Name
		}
	}
	obj, err := s.resolveBillingObject(params)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return billingObject{}, store.Query{}, false
	}
	if isRO && (obj.Kind != ro.Kind || obj.Name != ro.Object) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "readonly_scope"})
		return billingObject{}, store.Query{}, false
	}
	q, err := billingRange(r)
	if err != nil {
		fail(w, err)
		return billingObject{}, store.Query{}, false
	}
	if isRO {
		if q.To.Sub(q.From) > roMaxSpan {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_range"})
			return billingObject{}, store.Query{}, false
		}
		if ok, retry := s.roRL.allow(ro.User, clientIP(r), time.Now()); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limited"})
			return billingObject{}, store.Query{}, false
		}
		s.audit("billing query user=%s from=%s for %s %s", ro.User, clientIP(r), obj.Kind, obj.Name)
	}
	return obj, q, true
}

// billing answers GET /api/billing?product=…|vip=…|subnet=…&from&to (or &month=).
func (s *Server) billing(w http.ResponseWriter, r *http.Request) {
	obj, q, ok := s.billingRequest(w, r)
	if !ok {
		return
	}
	out, err := s.billingFor(obj, q)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// billingExport answers GET /api/billing/export with the same selection
// as the statistics page, as a spreadsheet file: one row per bucket with
// the same field names the page shows, and a final "total" row carrying
// the settled figures the page headlines.
func (s *Server) billingExport(w http.ResponseWriter, r *http.Request) {
	obj, q, ok := s.billingRequest(w, r)
	if !ok {
		return
	}
	out, err := s.billingFor(obj, q)
	if err != nil {
		fail(w, err)
		return
	}
	zone := time.FixedZone("", int(q.ZoneOffset.Seconds()))
	name := "traffic66-billing-" + q.From.In(zone).Format("20060102") + "_" + q.To.In(zone).Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	cw.Write([]string{"bucket_start", "bucket_end", "in_bytes", "out_bytes", "bytes",
		"in_bps", "out_bps", "total_bps"})
	step := time.Duration(out.Step) * time.Second
	row := func(i int) []string {
		start := time.UnixMilli(out.Times[i]).In(zone)
		secs := step.Seconds()
		ib := uint64(math.Round(out.In[i] * secs / 8))
		ob := uint64(math.Round(out.Out[i] * secs / 8))
		tb := uint64(math.Round(out.Total[i] * secs / 8))
		return []string{start.Format(time.RFC3339), start.Add(step).Format(time.RFC3339),
			strconv.FormatUint(ib, 10), strconv.FormatUint(ob, 10), strconv.FormatUint(tb, 10),
			strconv.FormatFloat(out.In[i], 'f', 0, 64), strconv.FormatFloat(out.Out[i], 'f', 0, 64),
			strconv.FormatFloat(out.Total[i], 'f', 0, 64)}
	}
	for i := range out.Times {
		cw.Write(row(i))
	}
	cw.Write([]string{"total", "", strconv.FormatUint(out.InBytes, 10), strconv.FormatUint(out.OutBytes, 10),
		strconv.FormatUint(out.Bytes, 10), "", "", ""})
	cw.Flush()
}

// maxVIPs is how many addresses one ad-hoc VIP list may name.
const maxVIPs = 64

// parseVIPs reads the vip parameter: one address, or several merged into
// one bill, separated by commas (a full-width comma or spaces work too,
// as they are typed). Repeats are dropped, the order is kept.
func parseVIPs(s string) ([]netip.Addr, error) {
	f := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' || r == ' ' || r == '\t' })
	if len(f) == 0 {
		return nil, errors.New("bad_vip")
	}
	seen := map[netip.Addr]bool{}
	var out []netip.Addr
	for _, x := range f {
		a, err := netip.ParseAddr(x)
		if err != nil {
			return nil, errors.New("bad_vip")
		}
		a = a.Unmap()
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	if len(out) > maxVIPs {
		return nil, errors.New("bad_vip")
	}
	return out, nil
}

// percentile95 is the nearest-rank 95th percentile: sorted, the top 5%
// of the samples are the ones left out, and the next one is the figure.
// It sorts vals in place.
func percentile95(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	k := int(math.Ceil(0.95*float64(len(vals)))) - 1
	return vals[max(k, 0)]
}
