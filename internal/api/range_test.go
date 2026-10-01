package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLongRangesStartOnTheHour(t *testing.T) {
	for _, tc := range []struct {
		r       string
		aligned bool
	}{{"15m", false}, {"1h", false}, {"6h", false}, {"24h", true}, {"7d", true}, {"30d", true}} {
		q, err := parseQuery(httptest.NewRequest("GET", "/api/totals?range="+tc.r, nil))
		if err != nil {
			t.Fatal(err)
		}
		if got := q.From.Equal(q.From.Truncate(time.Hour)); got != tc.aligned && !(got && !tc.aligned) {
			t.Errorf("%s: from %v, want aligned=%v", tc.r, q.From, tc.aligned)
		}
		if tc.aligned && (q.To.Sub(q.From) < ranges[tc.r] || q.To.Sub(q.From) > ranges[tc.r]+time.Hour) {
			t.Errorf("%s: window %v", tc.r, q.To.Sub(q.From))
		}
	}
	q, _ := parseQuery(httptest.NewRequest("GET", "/api/totals?from=1790751629590&to=1790838029590", nil))
	if !q.From.Equal(q.From.Truncate(time.Hour)) || !q.To.Equal(q.To.Truncate(time.Hour)) {
		t.Errorf("explicit long range not aligned: %v – %v", q.From, q.To)
	}
}
