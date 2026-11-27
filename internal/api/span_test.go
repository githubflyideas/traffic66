package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// One query covers at most thirty days; more is a bad_range, in the
// shape bad ranges already have. The counter download is exempt.
func TestParseQuerySpanCap(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	url := func(span time.Duration) string {
		return fmt.Sprintf("/api/topn?from=%d&to=%d", base.UnixMilli(), base.Add(span).UnixMilli())
	}
	if _, err := parseQuery(httptest.NewRequest("GET", url(31*24*time.Hour), nil)); !errors.Is(err, errBadRange) {
		t.Fatalf("31 days: %v, want bad_range", err)
	}
	if _, err := parseQuery(httptest.NewRequest("GET", url(30*24*time.Hour), nil)); err != nil {
		t.Fatalf("30 days: %v", err)
	}
	if _, err := parseQuery(httptest.NewRequest("GET", "/api/topn?range=30d", nil)); err != nil {
		t.Fatalf("range=30d: %v", err)
	}
	if _, err := parseQuerySpan(httptest.NewRequest("GET", url(90*24*time.Hour), nil), 0); err != nil {
		t.Fatalf("uncapped: %v", err)
	}
	// the refusal carries the bad_range shape
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{Store: st}
	w := httptest.NewRecorder()
	s.topn(w, httptest.NewRequest("GET", url(31*24*time.Hour)+"&dim=app", nil))
	if w.Code != 400 || w.Body.String() != "{\"code\":\"bad_range\",\"error\":\"bad range\"}\n" {
		t.Fatalf("31 days over the handler: %d %s", w.Code, w.Body)
	}
}

// Top answers say where they came from; records say when they start
// before the records do, and that a range wholly before them is
// answered by the summaries only.
func TestTopnSourceAndRecordsDetailFrom(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	row := store.Row{TS: base, Src: "198.51.100.20", Dst: "203.0.113.10", Client: "198.51.100.20", Server: "203.0.113.10",
		Cli4: 0xC6336414, Srv4: 0xCB00710A, SvcPort: 443, Proto: 6, App: "HTTPS", Bytes: 750, Wire: 1000, Pkts: 10, Flows: 1,
		Exporter: "10.0.0.1", InIf: 1, OutIf: 2, Dir: store.DirOutbound, PeerCC: "US", CliInt: true}
	if err := st.AppendRows([]store.Row{row}); err != nil {
		t.Fatal(err)
	}
	s := &Server{Store: st}
	get := func(handler func(http.ResponseWriter, *http.Request), url string) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", url, w.Code, w.Body)
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	rng := func(from, to time.Time) string {
		return fmt.Sprintf("from=%d&to=%d", from.UnixMilli(), to.UnixMilli())
	}
	out := get(s.topn, "/api/topn?dim=conv&"+rng(base.Add(-time.Hour), base.Add(time.Hour)))
	if out["source"] != "detail" || out["rollup"] != false {
		t.Fatalf("topn over the records: %v", out)
	}
	if _, ok := out["rolled_from"]; ok {
		t.Fatalf("topn over the records names a rolled range: %v", out)
	}
	out = get(s.records, "/api/records?"+rng(base.Add(-2*time.Hour), base.Add(-time.Hour)))
	if out["source"] != "rollup_only" {
		t.Fatalf("records before the records: %v", out)
	}
	if rows, ok := out["rows"].([]any); !ok || len(rows) != 0 {
		t.Fatalf("records before the records: rows %v", out["rows"])
	}
	if out["detail_from"] != float64(base.UnixMilli()) {
		t.Fatalf("detail_from %v, want %d", out["detail_from"], base.UnixMilli())
	}
	out = get(s.records, "/api/records?"+rng(base.Add(-time.Hour), base.Add(time.Hour)))
	if _, ok := out["source"]; ok {
		t.Fatalf("records straddling the start claim rollup_only: %v", out)
	}
	if out["detail_from"] != float64(base.UnixMilli()) {
		t.Fatalf("straddling records: detail_from %v", out["detail_from"])
	}
	out = get(s.records, "/api/records?"+rng(base, base.Add(time.Hour)))
	if _, ok := out["detail_from"]; ok {
		t.Fatalf("records inside the detail name a detail_from: %v", out)
	}
}
