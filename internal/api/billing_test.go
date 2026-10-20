package api

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

func ms(t time.Time) string { return strconv.FormatInt(t.UnixMilli(), 10) }

func billingOf(t *testing.T, s *Server, q string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	s.billing(w, httptest.NewRequest("GET", "/api/billing?"+q, nil))
	if w.Code != 200 {
		t.Fatalf("%s: %d %s", q, w.Code, w.Body)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// A month is given as an interval, and the interval is used as given.
// parseQuery rounds any range over a day to the hour, which would move the
// first midnight of a month and take the wrong samples with it; billing
// therefore reads from and to itself, and this is the test that says so.
func TestBillingKeepsTheExactPeriod(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{Store: st}
	// A month that starts at 00:00 in a zone 30 minutes off UTC: the
	// boundary the browser sends is half past the hour.
	from := time.Date(2026, 3, 1, 18, 30, 0, 0, time.UTC)
	for i := 0; i < 288; i++ {
		rate := 100e6
		if i == 0 {
			rate = 900e6 // only inside the period if the boundary is kept
		}
		if _, err := st.DB.Exec(`INSERT INTO ifr VALUES (?, '10.0.0.1', 3, ?, 0, 10)`,
			from.Add(time.Duration(i)*5*time.Minute), rate); err != nil {
			t.Fatal(err)
		}
	}
	out := billingOf(t, s, "from="+ms(from)+"&to="+ms(from.Add(24*time.Hour)))
	if got := int64(out["from"].(float64)); got != from.UnixMilli() {
		t.Errorf("from came back as %v, want the half-hour boundary unchanged", time.UnixMilli(got).UTC())
	}
	bills := out["bills"].([]any)
	if len(bills) != 1 {
		t.Fatalf("%d interfaces, want 1: %v", len(bills), out)
	}
	b := bills[0].(map[string]any)
	if b["buckets"].(float64) != 288 || b["coverage"].(float64) != 1 {
		t.Errorf("buckets %v coverage %v: the period lost samples at its edge", b["buckets"], b["coverage"])
	}
	if b["out_max"].(float64) != 0 || b["in_max"].(float64) != 900e6 {
		t.Errorf("in_max %v out_max %v, want the first bucket inside the period", b["in_max"], b["out_max"])
	}
	if b["dir"].(string) != "in" || b["billed"].(float64) != 100e6 {
		t.Errorf("billed %v on %v, want 100000000 on in", b["billed"], b["dir"])
	}
	if out["retention_days"].(float64) != 400 {
		t.Errorf("retention_days = %v, want 400: it is what bounds how far back a bill can go", out["retention_days"])
	}
}

func TestBillingRejectsMissingPeriod(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{Store: st}
	for _, q := range []string{"", "from=0", "to=0", "from=x&to=1", "from=2&to=1"} {
		w := httptest.NewRecorder()
		s.billing(w, httptest.NewRequest("GET", "/api/billing?"+q, nil))
		if w.Code == 200 {
			t.Errorf("%q was accepted: %s", q, w.Body)
		}
	}
}

// A switch reports dozens of interfaces and a bill is about two or three
// of them, so marked interfaces win. With none marked the page shows
// everything, because an empty page on the first visit teaches nobody
// anything — and the count of marks is reported so the page can say which
// of the two it is doing.
func TestBillingShowsTheMarkedInterfaces(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	inv := enrich.NewInventory()
	s := &Server{Store: st, Inv: inv}
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for _, idx := range []int{3, 4, 5} {
		for i := 0; i < 288; i++ {
			if _, err := st.DB.Exec(`INSERT INTO ifr VALUES (?, '10.0.0.1', ?, ?, 0, 10)`,
				from.Add(time.Duration(i)*5*time.Minute), idx, float64(idx)*100e6); err != nil {
				t.Fatal(err)
			}
		}
	}
	q := "from=" + ms(from) + "&to=" + ms(from.Add(24*time.Hour))

	out := billingOf(t, s, q)
	if n := len(out["bills"].([]any)); n != 3 || out["marked"].(float64) != 0 {
		t.Fatalf("nothing marked: %d interfaces, marked=%v, want all three", n, out["marked"])
	}

	// mark one, and one that has no data at all
	if err := inv.Parse("iface 10.0.0.1 4 ISP-A billed\niface 10.0.0.1 9 ISP-B billed\n"); err != nil {
		t.Fatal(err)
	}
	out = billingOf(t, s, q)
	if out["marked"].(float64) != 2 {
		t.Errorf("marked = %v, want 2", out["marked"])
	}
	rows := out["bills"].([]any)
	got := map[float64]map[string]any{}
	for _, r := range rows {
		m := r.(map[string]any)
		got[m["ifindex"].(float64)] = m
	}
	if len(rows) != 2 || got[4] == nil || got[9] == nil {
		t.Fatalf("%d rows %v, want only the two marked interfaces", len(rows), got)
	}
	if got[4]["marked"] != true || got[4]["billed"] == nil {
		t.Errorf("the marked interface with data lost its figure: %v", got[4])
	}
	// the marked interface that reported nothing is still listed, saying so
	if got[9]["buckets"].(float64) != 0 || got[9]["coverage"].(float64) != 0 {
		t.Errorf("a marked interface with no data should be a row with nothing in it: %v", got[9])
	}
	if got[9]["name"] != "ISP-B" {
		t.Errorf("name = %v, want the name from the names file", got[9]["name"])
	}
}

// The chart under the table is drawn from the same buckets the figure is
// taken from: every bucket that holds a measurement comes back, the ones
// that hold none are absent rather than zero, and the percentile the chart
// draws its line at is the one the table charges.
func TestBillingSeries(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{Store: st}
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	// six hours of five-minute buckets, with an hour in the middle the
	// device reported nothing for
	for i := 0; i < 72; i++ {
		if i >= 30 && i < 42 {
			continue
		}
		rate := 100e6
		if i == 50 {
			rate = 900e6
		}
		if _, err := st.DB.Exec(`INSERT INTO ifr VALUES (?, '10.0.0.1', 3, ?, ?, 10)`,
			from.Add(time.Duration(i)*5*time.Minute), rate, rate/2); err != nil {
			t.Fatal(err)
		}
	}
	get := func(q string) (int, map[string]any) {
		w := httptest.NewRecorder()
		s.billingSeries(w, httptest.NewRequest("GET", "/api/billing/series?"+q, nil))
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, out := get("from=" + ms(from) + "&to=" + ms(from.Add(6*time.Hour)) + "&if=10.0.0.1/3")
	if code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	pts := out["points"].([]any)
	if len(pts) != 60 {
		t.Fatalf("%d buckets, want the 60 that hold a measurement and not the 12 that do not", len(pts))
	}
	first := pts[0].(map[string]any)
	if first["in"].(float64) != 100e6 || first["out"].(float64) != 50e6 || first["samples"].(float64) != 10 {
		t.Errorf("first bucket %v", first)
	}
	b := out["bill"].(map[string]any)
	if b["billed"].(float64) != 100e6 || b["in_max"].(float64) != 900e6 {
		t.Errorf("bill %v, want the table's own figure", b)
	}
	if cov := out["coverage"].(float64); cov < 0.8 || cov > 0.9 {
		t.Errorf("coverage %v, want the 60 buckets of 72 the period could hold", cov)
	}
	// the interface has to be named, and named the way the store expects
	for _, q := range []string{"", "if=10.0.0.1/3", "from=" + ms(from) + "&to=" + ms(from.Add(time.Hour)),
		"from=" + ms(from) + "&to=" + ms(from.Add(time.Hour)) + "&if=eth0"} {
		if code, _ := get(q); code == 200 {
			t.Errorf("%q was accepted", q)
		}
	}
	// an interface nobody reported counters for answers with no buckets
	// rather than an error: the page draws the empty state
	code, out = get("from=" + ms(from) + "&to=" + ms(from.Add(6*time.Hour)) + "&if=10.0.0.9/1")
	if code != 200 || len(out["points"].([]any)) != 0 || out["bill"] != nil {
		t.Errorf("unknown interface: %d %v", code, out)
	}
}
