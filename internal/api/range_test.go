package api

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestLongRangesStartOnTheHour(t *testing.T) {
	for _, tc := range []struct {
		r       string
		aligned bool
	}{{"15m", false}, {"30m", false}, {"1h", false}, {"6h", false}, {"12h", true}, {"24h", true}, {"7d", true}, {"30d", true}} {
		q, err := parseQuery(httptest.NewRequest("GET", "/api/totals?range="+tc.r, nil))
		if err != nil {
			t.Fatal(err)
		}
		if got := q.From.Equal(q.From.Truncate(time.Hour)); got != tc.aligned && !(got && !tc.aligned) {
			t.Errorf("%s: from %v, want aligned=%v", tc.r, q.From, tc.aligned)
		}
		if _, ok := ranges[tc.r]; !ok || (!tc.aligned && q.To.Sub(q.From) != ranges[tc.r]) {
			t.Errorf("%s: window %v, want exactly %v", tc.r, q.To.Sub(q.From), ranges[tc.r])
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

// A span of centuries used to overflow when it was turned into a
// Duration, come out negative and pass the thirty-day limit.
func TestHugeSpanIsRefused(t *testing.T) {
	for _, ft := range []string{
		"from=0&to=4102444800000",
		"from=0&to=9300000000000",
		"from=0&to=9223372036854775807",
		"from=-9223372036854775808&to=9223372036854775807",
		"from=-9000000000000000000&to=9000000000000000000",
	} {
		if _, err := parseQuery(httptest.NewRequest("GET", "/api/overview?"+ft, nil)); err != errBadRange {
			t.Errorf("%s: err %v, want bad_range", ft, err)
		}
	}
	if _, err := parseQuery(httptest.NewRequest("GET", "/api/overview?from=1790751629590&to=1790838029590", nil)); err != nil {
		t.Errorf("one day: %v", err)
	}
}

func TestSummaryQueriesAreCapped(t *testing.T) {
	half := "from=1767225600000&to=" + strconv.FormatInt(1767225600000+summarySpan.Milliseconds(), 10)
	over := "from=1767225600000&to=" + strconv.FormatInt(1767225600000+summarySpan.Milliseconds()+1, 10)
	if _, err := parseQuerySpan(httptest.NewRequest("GET", "/api/export/rollups?"+half, nil), summarySpan); err != nil {
		t.Errorf("export at the limit: %v", err)
	}
	for _, ft := range []string{over, "from=0&to=9300000000000"} {
		if _, err := parseQuerySpan(httptest.NewRequest("GET", "/api/export/rollups?"+ft, nil), summarySpan); err != errBadRange {
			t.Errorf("export %s: %v, want bad_range", ft, err)
		}
		if _, err := billingRange(httptest.NewRequest("GET", "/api/billing?"+ft, nil)); err != errBadRange {
			t.Errorf("billing %s: %v, want bad_range", ft, err)
		}
	}
	if _, err := billingRange(httptest.NewRequest("GET", "/api/billing?"+half, nil)); err != nil {
		t.Errorf("billing at the limit: %v", err)
	}
}
