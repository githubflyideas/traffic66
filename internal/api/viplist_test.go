package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// A comma-separated VIP list bills its addresses together, the way a
// product line holding the same addresses does.
func TestBillingVIPList(t *testing.T) {
	s, ts := billingServer(t)
	get := func(q string) (int, billingOut, string) {
		w := httptest.NewRecorder()
		s.billing(w, httptest.NewRequest("GET", "/api/billing?"+q+"&"+billingWindow(ts), nil))
		var out billingOut
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out, w.Body.String()
	}
	_, line, _ := get("product=Shop")
	for _, q := range []string{"vip=203.0.113.10,203.0.113.11", "vip=203.0.113.11%EF%BC%8C203.0.113.10", "vip=203.0.113.10,%20203.0.113.11,203.0.113.10"} {
		code, out, body := get(q)
		if code != 200 || out.Bytes != line.Bytes || out.InBytes != line.InBytes || out.OutBytes != line.OutBytes {
			t.Errorf("%s: %d %s, want the product line's %d bytes", q, code, body, line.Bytes)
		}
	}
	for _, q := range []string{"vip=203.0.113.10,nope", "vip=,", "vip=" + strings.Repeat("10.0.0.1,", 0) + "x"} {
		if code, _, _ := get(q); code != 400 {
			t.Errorf("%s: %d, want 400", q, code)
		}
	}
}

// The 95th percentile, the peak and the daily volumes come from the
// settled buckets, and the days add up to the window's total.
func TestBillingP95AndDays(t *testing.T) {
	s, ts := billingServer(t)
	w := httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?product=Shop&"+billingWindow(ts), nil))
	var out billingOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if out.PeakBps <= 0 || out.P95Bps > out.PeakBps || out.P95Step != out.Step {
		t.Errorf("p95 %v (step %d), peak %v (step %d)", out.P95Bps, out.P95Step, out.PeakBps, out.Step)
	}
	var sum uint64
	for _, d := range out.Days {
		if len(d.Day) != 10 {
			t.Errorf("day %q", d.Day)
		}
		sum += d.Bytes
	}
	if len(out.Days) == 0 || sum != out.Bytes {
		t.Errorf("days %+v add up to %d, want the total %d", out.Days, sum, out.Bytes)
	}
}

func TestPercentile95(t *testing.T) {
	seq := func(n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			v[n-1-i] = float64(i + 1) // reversed: the sort is the function's
		}
		return v
	}
	for _, tc := range []struct {
		n    int
		want float64
	}{{1, 1}, {20, 19}, {100, 95}, {288, 274}, {8640, 8208}} {
		if got := percentile95(seq(tc.n)); got != tc.want {
			t.Errorf("1..%d: %v, want %v", tc.n, got, tc.want)
		}
	}
	if percentile95(nil) != 0 {
		t.Error("empty")
	}
}
