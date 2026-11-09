package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/flow"
)

type nullSink struct{}

func (nullSink) Submit([]flow.Record)             {}
func (nullSink) SubmitCounters([]flow.IfCounters) {}

// Blocking a source silences a device for everybody, so it is the
// administrator's, like every other install-wide write.
func TestBlockingIsTheAdministrators(t *testing.T) {
	dir := t.TempDir()
	bl, err := collector.NewBlocklist(dir)
	if err != nil {
		t.Fatal(err)
	}
	col := collector.New(nullSink{})
	col.Block = bl
	s := &Server{Col: col, sessions: map[string]session{}}
	mux := http.NewServeMux()
	mux.Handle("POST /api/sources/block", s.adminOnly(s.postBlock))
	mux.Handle("DELETE /api/sources/block", s.adminOnly(s.deleteBlock))
	call := func(method, path, body, user string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, user))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	// no Accounts at all: nobody is the administrator, so nobody blocks
	if w := call("POST", "/api/sources/block", `{"exporter":"192.0.2.5"}`, "alice"); w.Code != http.StatusForbidden {
		t.Errorf("without an administrator: got %d, want 403", w.Code)
	}
	if n := len(bl.List()); n != 0 {
		t.Fatalf("a refused block still blocked %d sources", n)
	}
}

// The block and unblock round trip, and say no to what they cannot do.
func TestBlockAndUnblock(t *testing.T) {
	dir := t.TempDir()
	bl, err := collector.NewBlocklist(dir)
	if err != nil {
		t.Fatal(err)
	}
	col := collector.New(nullSink{})
	col.Block = bl
	s := &Server{Col: col, LocalTok: "tok", sessions: map[string]session{}}
	mux := http.NewServeMux()
	// the local token stands in for the administrator, as everywhere else
	mux.Handle("POST /api/sources/block", s.auth(s.adminOnly(s.postBlock)))
	mux.Handle("DELETE /api/sources/block", s.auth(s.adminOnly(s.deleteBlock)))
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.1:1"
		r.Header.Set("Authorization", "Bearer tok")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := call("POST", "/api/sources/block", `{"exporter":"192.0.2.5","note":"noisy test rig"}`); w.Code != http.StatusOK {
		t.Fatalf("block: got %d %s", w.Code, w.Body)
	}
	if !bl.Has(mustAddr(t, "192.0.2.5")) {
		t.Error("the source is not blocked after a 200")
	}
	if w := call("POST", "/api/sources/block", `{"exporter":"not-an-address"}`); w.Code != http.StatusBadRequest {
		t.Errorf("a bad address: got %d, want 400", w.Code)
	}
	// blocking twice is not an error, and does not look like a new block
	first := bl.List()[0].Since
	if w := call("POST", "/api/sources/block", `{"exporter":"192.0.2.5"}`); w.Code != http.StatusOK {
		t.Errorf("blocking twice: got %d %s", w.Code, w.Body)
	}
	if got := bl.List()[0].Since; !got.Equal(first) {
		t.Errorf("a second block moved the time from %v to %v", first, got)
	}
	if w := call("DELETE", "/api/sources/block?exporter=192.0.2.9", ""); w.Code != http.StatusNotFound {
		t.Errorf("unblocking what is not blocked: got %d, want 404", w.Code)
	}
	if w := call("DELETE", "/api/sources/block?exporter=192.0.2.5", ""); w.Code != http.StatusOK {
		t.Fatalf("unblock: got %d %s", w.Code, w.Body)
	}
	if bl.Has(mustAddr(t, "192.0.2.5")) {
		t.Error("still blocked after a 200 unblock")
	}
}

// Reading a flow file offline has no collector, so there is nothing to
// block; saying so beats a nil dereference.
func TestBlockingWithoutACollector(t *testing.T) {
	s := &Server{sessions: map[string]session{}}
	mux := http.NewServeMux()
	mux.Handle("POST /api/sources/block", s.auth(s.adminOnly(s.postBlock)))
	r := httptest.NewRequest("POST", "/api/sources/block", strings.NewReader(`{"exporter":"192.0.2.5"}`))
	r.RemoteAddr = "192.0.2.1:1"
	r.Header.Set("Authorization", "Bearer tok")
	s.LocalTok = "tok"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Errorf("without a collector: got %d %s, want 409", w.Code, w.Body)
	}
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	a, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
