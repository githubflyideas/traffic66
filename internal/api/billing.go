package api

import (
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// billingSeries answers GET /api/billing/series?from&to&if=exporter/ifindex
// with the five-minute buckets one interface's bill was computed from, and
// the percentile drawn across them. The table says what is charged; this
// says what it was charged from, which is what somebody disputing a bill
// asks to see.
func (s *Server) billingSeries(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	from, to, err := billPeriod(v)
	if err != nil {
		fail(w, err)
		return
	}
	key := v.Get("if")
	if err := checkIface(key); err != nil {
		fail(w, err)
		return
	}
	e, idxs, _ := strings.Cut(key, "/")
	idx, _ := strconv.ParseUint(idxs, 10, 32)
	pts, err := s.Store.BillSeries(from, to, e, uint32(idx))
	if err != nil {
		fail(w, err)
		return
	}
	out := map[string]any{
		"from": from.UnixMilli(), "to": to.UnixMilli(), "if": key,
		"bucket_minutes": 5, "points": pts,
	}
	// the figure the table shows, so the line across the chart and the
	// number beside it can never drift apart
	if bills, err := s.Store.Bills(from, to, e); err == nil {
		for _, b := range bills {
			if b.IfIndex == uint32(idx) {
				out["bill"] = b
				out["coverage"] = b.Coverage()
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// billPeriod reads the interval both billing endpoints take. The
// boundaries are used exactly as given: parseQuery rounds long ranges to
// the hour, which would quietly move the start of a month.
func billPeriod(v url.Values) (time.Time, time.Time, error) {
	ms := func(k string) (time.Time, error) {
		n, err := strconv.ParseInt(v.Get(k), 10, 64)
		if err != nil {
			return time.Time{}, errors.New(k + " must be a millisecond timestamp")
		}
		return time.UnixMilli(n).UTC(), nil
	}
	from, err := ms("from")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := ms("to")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}

// Billing reports the 95th percentile per interface over a period the
// caller gives as an interval. A calendar month, the last thirty days and a
// cycle that starts on the third are then the same request, and the month's
// boundaries are worked out where the user's time zone is known — in the
// browser — rather than guessed at here or pinned to a server flag.
func (s *Server) billing(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	from, to, err := billPeriod(v)
	if err != nil {
		fail(w, err)
		return
	}
	bills, err := s.Store.Bills(from, to, v.Get("exporter"))
	if err != nil {
		fail(w, err)
		return
	}
	type out struct {
		store.Bill
		Name     string  `json:"name,omitempty"`
		Device   string  `json:"device,omitempty"`
		Coverage float64 `json:"coverage"`
		// Not "billed": store.Bill already uses that for the figure, and an
		// outer field of the same name would shadow it out of the JSON.
		Marked bool `json:"marked,omitempty"`
	}
	// A device reports every interface it has; the two or three somebody
	// pays a carrier for are the ones a bill is about. Marked interfaces
	// win when there are any. When there are none the page shows
	// everything and says so, because an empty page on the first visit
	// teaches nobody anything.
	var billed map[string]bool
	if s.Inv != nil {
		billed = s.Inv.Billed()
	}
	key := func(e string, i uint32) string { return e + "/" + strconv.FormatUint(uint64(i), 10) }
	rows := make([]out, 0, len(bills))
	seen := map[string]bool{}
	for _, b := range bills {
		k := key(b.Exporter, b.IfIndex)
		if len(billed) > 0 && !billed[k] {
			continue
		}
		seen[k] = true
		o := out{Bill: b, Coverage: b.Coverage(), Marked: billed[k]}
		if a, err := netip.ParseAddr(b.Exporter); err == nil && s.Inv != nil {
			o.Name = s.Inv.Iface(a, b.IfIndex).Name
			o.Device = s.Inv.Device(a)
		}
		rows = append(rows, o)
	}
	// A marked interface with nothing in the period is the thing an
	// operator most needs to see: the circuit they are paying for stopped
	// reporting. It gets a row saying so rather than quietly vanishing.
	for k := range billed {
		if seen[k] {
			continue
		}
		e, idxs, ok := strings.Cut(k, "/")
		if !ok {
			continue
		}
		i, err := strconv.ParseUint(idxs, 10, 32)
		if err != nil {
			continue
		}
		o := out{Bill: store.Bill{Exporter: e, IfIndex: uint32(i), Possible: int(to.Sub(from) / (5 * time.Minute))}, Marked: true}
		if a, err := netip.ParseAddr(e); err == nil && s.Inv != nil {
			o.Name = s.Inv.Iface(a, uint32(i)).Name
			o.Device = s.Inv.Device(a)
		}
		rows = append(rows, o)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from.UnixMilli(), "to": to.UnixMilli(),
		"bucket_minutes": 5, "retention_days": s.Store.RollupRetentionDays(),
		"marked": len(billed), "bills": rows,
	})
}
