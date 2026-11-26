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
