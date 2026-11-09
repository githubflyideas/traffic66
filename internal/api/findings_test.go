package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

func TestFindings(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{Store: st}
	now := time.Now().UTC().Truncate(time.Minute)
	scan := store.Finding{Kind: "scan", Sev: 3, Src: "10.1.1.1", Port: "445/tcp", First: now.Add(-20 * time.Minute), Last: now.Add(-10 * time.Minute),
		Ev: map[string]any{"targets": 40.0, "examples": []string{"10.1.2.3"}}}
	id, created, err := st.SaveFinding(scan, 30*time.Minute)
	if err != nil || !created {
		t.Fatal(created, err)
	}
	// the same scan ten minutes later extends the finding
	scan.First, scan.Last, scan.Ev = now.Add(-10*time.Minute), now, map[string]any{"targets": 90.0, "examples": []string{"10.1.2.9"}}
	if id2, created, err := st.SaveFinding(scan, 30*time.Minute); err != nil || created || id2 != id {
		t.Fatal(id2, created, err)
	}
	st.SaveFinding(store.Finding{Kind: "threat", Sev: 1, Src: "192.0.2.66", Dst: "203.0.113.5", Port: "scanner", First: now.Add(-time.Hour), Last: now}, 30*time.Minute)

	get := func(q string) map[string]any {
		w := httptest.NewRecorder()
		s.findings(w, httptest.NewRequest("GET", "/api/findings?range=24h"+q, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", q, w.Code, w.Body)
		}
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	out := get("")
	fs := out["findings"].([]any)
	if len(fs) != 2 || out["open"].(map[string]any)["high"].(float64) != 1 {
		t.Fatalf("findings %v", out)
	}
	first := fs[0].(map[string]any)
	ev := first["ev"].(map[string]any)
	if first["kind"] != "scan" || first["hits"].(float64) != 2 || ev["targets"].(float64) != 90 || len(ev["examples"].([]any)) != 2 {
		t.Fatalf("merged finding %v", first)
	}
	if fs := get("&ip=10.1.2.9")["findings"].([]any); len(fs) != 1 {
		t.Fatalf("by target: %v", fs)
	}

	post := func(body string) int {
		w := httptest.NewRecorder()
		s.setFindings(w, httptest.NewRequest("POST", "/api/findings", strings.NewReader(body)))
		return w.Code
	}
	if c := post(`{"ids":[` + jsonInt(id) + `],"status":"false"}`); c != 200 {
		t.Fatal(c)
	}
	if c := post(`{"ids":[1],"status":"bogus"}`); c != 400 {
		t.Fatal("bad status accepted", c)
	}
	if fs := get("")["findings"].([]any); len(fs) != 1 {
		t.Fatalf("open after marking false: %v", fs)
	}
	// a finding marked as not a problem is not reported again
	scan.First, scan.Last = now.Add(time.Hour), now.Add(70*time.Minute)
	if _, created, _ := st.SaveFinding(scan, 30*time.Minute); created {
		t.Fatal("suppressed finding reopened")
	}
}

func jsonInt(n int64) string { b, _ := json.Marshal(n); return string(b) }
