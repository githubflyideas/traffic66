package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/license"
)

// importServer is assetServer plus the two batch routes, which its mux
// does not carry.
func importServer(t *testing.T, devices int) (*Server, func(path, body string) *httptest.ResponseRecorder) {
	t.Helper()
	s, _ := assetServer(t, []string{license.FeatDCIM}, 0, 0, devices)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/assets/csv", s.postAssetCSV)
	mux.HandleFunc("POST /api/assets/paste", s.postPaste)
	call := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, "admin"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	return s, call
}

// deviceCSV is n new devices: an empty id column means "add".
func deviceCSV(from, n int) string {
	var b strings.Builder
	b.WriteString("id,name\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, ",dev%d\n", from+i)
	}
	return b.String()
}

func importCSV(call func(path, body string) *httptest.ResponseRecorder, csv string, apply bool) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{"kind": "device", "text": csv, "apply": apply})
	return call("/api/assets/csv", string(body))
}

// TestCSVImportObeysTheQuota. A spreadsheet used to be the way round
// every number the licence sells: postAssetCSV had no quota check at all,
// so a licence for five devices took an import of five hundred.
func TestCSVImportObeysTheQuota(t *testing.T) {
	s, call := importServer(t, 5)
	devs := func() int { return s.Assets.Counts().Devices }

	// a preview writes nothing, so it is never refused
	if w := importCSV(call, deviceCSV(1, 500), false); w.Code != http.StatusOK {
		t.Fatalf("preview of 500: got %d %s, want 200", w.Code, w.Body)
	}
	if n := devs(); n != 0 {
		t.Fatalf("a preview wrote %d devices", n)
	}

	// applying more than the licence holds is refused, and writes nothing
	w := importCSV(call, deviceCSV(1, 10), true)
	if w.Code != http.StatusForbidden {
		t.Errorf("importing 10 against a licence for 5: got %d %s, want 403", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "quota") {
		t.Errorf("expected a quota refusal, got %s", w.Body)
	}
	if n := devs(); n != 0 {
		t.Fatalf("a refused import wrote %d devices; it must write none", n)
	}

	// exactly the licence's worth goes in
	if w := importCSV(call, deviceCSV(1, 5), true); w.Code != http.StatusOK {
		t.Fatalf("importing 5 against a licence for 5: got %d %s", w.Code, w.Body)
	}
	if n := devs(); n != 5 {
		t.Fatalf("after importing 5 the ledger holds %d", n)
	}

	// and then nothing more
	if w := importCSV(call, deviceCSV(6, 1), true); w.Code != http.StatusForbidden {
		t.Errorf("one more at the limit: got %d %s, want 403", w.Code, w.Body)
	}
	if n := devs(); n != 5 {
		t.Errorf("the ledger holds %d after a refused import", n)
	}
}

// TestBatchCannotOvershootTheQuota is the off-by-a-batch the first
// version of the paste check had: it asked whether the ledger was already
// at its limit, which is false at four of five, so a batch of three went
// in and left seven.
func TestBatchCannotOvershootTheQuota(t *testing.T) {
	s, call := importServer(t, 5)
	if w := importCSV(call, deviceCSV(1, 4), true); w.Code != http.StatusOK {
		t.Fatalf("filling to 4: got %d %s", w.Code, w.Body)
	}
	if n := s.Assets.Counts().Devices; n != 4 {
		t.Fatalf("expected 4 devices, got %d", n)
	}
	// 4 + 3 > 5: refused as a batch, not one at a time
	if w := importCSV(call, deviceCSV(5, 3), true); w.Code != http.StatusForbidden {
		t.Errorf("3 more at 4 of 5: got %d %s, want 403", w.Code, w.Body)
	}
	if n := s.Assets.Counts().Devices; n != 4 {
		t.Errorf("the ledger overshot to %d; the licence holds 5 and it was at 4", n)
	}
	// the one that does fit still goes in
	if w := importCSV(call, deviceCSV(5, 1), true); w.Code != http.StatusOK {
		t.Errorf("the 5th device: got %d %s, want 200", w.Code, w.Body)
	}
	if n := s.Assets.Counts().Devices; n != 5 {
		t.Errorf("expected 5 devices, got %d", n)
	}
}

// An unlimited licence (0) is not a limit of zero.
func TestQuotaZeroIsUnlimited(t *testing.T) {
	s, call := importServer(t, 0)
	if w := importCSV(call, deviceCSV(1, 50), true); w.Code != http.StatusOK {
		t.Fatalf("50 devices on an unlimited licence: got %d %s", w.Code, w.Body)
	}
	if n := s.Assets.Counts().Devices; n != 50 {
		t.Errorf("expected 50 devices, got %d", n)
	}
}
