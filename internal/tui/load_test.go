package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stubAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer t66_test" && r.Header.Get("Authorization") == "" {
			// still allow unauthenticated for the error-path test below
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/topn"):
			dim := r.URL.Query().Get("dim")
			var rows any
			switch dim {
			case "dir":
				rows = []map[string]any{
					{"key": "2", "wire": 1.5e7, "pkts": 1000},
					{"key": "1", "wire": 5e6, "pkts": 400},
					{"key": "3", "wire": 1e6, "pkts": 80},
					{"key": "0", "wire": 2e5, "pkts": 10},
				}
			default:
				rows = []map[string]any{
					{"key": "10.0.0.1", "wire": 8e6, "pkts": 600, "flows": 12, "extra": "edge"},
					{"key": "10.0.0.2", "wire": 3e6, "pkts": 200, "flows": 4},
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"rows": rows})
		case strings.HasPrefix(r.URL.Path, "/api/ifaces"):
			json.NewEncoder(w).Encode(map[string]any{"ifaces": []map[string]any{{
				"exporter": "192.168.1.1", "name": "uplink", "device": "edge-1",
				"ifindex": 1, "speed": 1e9, "in_wire": 2e7, "out_wire": 8e6,
				"in_dev": 2.1e7, "out_dev": 8.1e6, "has_counters": true,
			}}})
		case strings.HasPrefix(r.URL.Path, "/api/sources"):
			json.NewEncoder(w).Encode(map[string]any{"sources": []map[string]any{{
				"exporter": "192.168.1.1", "proto": "netflow", "name": "edge-1",
				"sampling": "1:1", "status": "ok",
			}}})
		case strings.HasPrefix(r.URL.Path, "/api/records"):
			json.NewEncoder(w).Encode(map[string]any{"rows": []map[string]any{{
				"ts": time.Now().UnixMilli(), "client": "10.0.0.8", "server": "1.1.1.1",
				"app": "dns", "cc": "US", "exporter": "192.168.1.1",
				"port": 53, "proto": 17, "wire": 1500, "pkts": 2, "dir": 1,
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestLoadPagesFromAPI(t *testing.T) {
	ts := stubAPI(t)
	defer ts.Close()
	a := newApp(Options{URL: ts.URL, Lang: "en", Token: "t66_test"})
	top := a.load(a.state())
	if top.err != "" {
		t.Fatalf("top: %s", top.err)
	}
	if len(top.top) == 0 || len(top.sections) == 0 || len(top.sections[0].rows) != 2 {
		t.Fatalf("top page: top=%d sections=%d", len(top.top), len(top.sections))
	}
	plain := top.top[0].plain()
	if !strings.Contains(plain, "Mbps") && !strings.Contains(plain, "bps") {
		t.Errorf("top header has no rate: %q", plain)
	}
	if top.sections[0].rows[0].key != "10.0.0.1" || top.sections[0].rows[0].target == nil {
		t.Errorf("top row: %+v", top.sections[0].rows[0])
	}

	a.setPage(1)
	a.w, a.h = 120, 36
	ifaces := a.load(a.state())
	if ifaces.err != "" || len(ifaces.sections) == 0 || len(ifaces.sections[0].rows) == 0 {
		t.Fatalf("ifaces: err=%s sections=%d", ifaces.err, len(ifaces.sections))
	}

	a.setPage(2)
	rec := a.load(a.state())
	if rec.err != "" || len(rec.sections) == 0 || len(rec.sections[0].rows) != 1 {
		t.Fatalf("records: err=%s rows=%v", rec.err, rec.sections)
	}
	got := rec.sections[0].rows[0]
	if got.target == nil || got.target.value != "1.1.1.1" {
		t.Errorf("record target: %+v", got.target)
	}

	// a failing API surfaces the status in the page error
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized","code":"unauthorized"}`))
	}))
	defer bad.Close()
	b := newApp(Options{URL: bad.URL, Lang: "en"})
	p := b.load(b.state())
	if p.err == "" || !strings.Contains(p.err, "401") {
		t.Fatalf("unauthorized load error: %q", p.err)
	}

	// reload and a short-lived fetchLoop
	a.setPage(0)
	a.reload()
	if a.data.err != "" || a.rowCount() != 2 {
		t.Fatalf("reload: err=%s count=%d", a.data.err, a.rowCount())
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.fetchLoop(ctx); close(done) }()
	a.kick()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fetchLoop did not exit")
	}
}
