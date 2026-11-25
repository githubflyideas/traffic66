package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// Read-only accounts count toward the user limit like any other
// account: they live in the same password file, and the limit is on
// how many accounts an installation may have. An unregistered
// installation allows three (FreeUsers): the administrator, one
// ordinary account and one read-only account fill it, and then both
// creation paths refuse — until a deletion frees a place again.
func TestReadonlyCountsTowardUserLimit(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	ck := apiLogin(t, h, pw)

	post := func(path, body string) (int, string) {
		w := apiCall(h, "POST", path, body, ck, "")
		return w.Code, w.Body.String()
	}
	users := func() (int, int) {
		w := apiCall(h, "GET", "/api/users", "", ck, "")
		var out struct {
			Users []string `json:"users"`
			Limit int      `json:"limit"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("users answer: %v (%s)", err, w.Body)
		}
		return len(out.Users), out.Limit
	}
	ro := func(name string) string {
		return fmt.Sprintf(`{"user":%q,"password":"a-long-enough-password","kind":"vip","object":"203.0.113.10"}`, name)
	}

	if code, body := post("/api/users", `{"user":"alice","password":"alice-password-1"}`); code != 200 {
		t.Fatalf("create ordinary: %d %s", code, body)
	}
	if code, body := post("/api/readonly", ro("shop")); code != 200 {
		t.Fatalf("create read-only: %d %s", code, body)
	}
	// admin + alice + shop = 3 of 3: the list and its limit agree
	if n, limit := users(); n != 3 || limit != 3 {
		t.Fatalf("users = %d of %d, want 3 of 3", n, limit)
	}
	// full: both paths refuse, with the same answer putUser has always given
	for _, tc := range []struct{ path, body string }{
		{"/api/readonly", ro("shop2")},
		{"/api/users", `{"user":"bob","password":"bob-password-123"}`},
	} {
		code, body := post(tc.path, tc.body)
		if code != http.StatusForbidden || !strings.Contains(body, `"user_limit"`) || !strings.Contains(body, `"limit":3`) {
			t.Errorf("full %s: %d %s, want 403 user_limit limit 3", tc.path, code, body)
		}
	}
	// deleting the read-only account frees its place
	w := apiCall(h, "DELETE", "/api/readonly?user=shop", "", ck, "")
	if w.Code != 200 {
		t.Fatalf("delete read-only: %d %s", w.Code, w.Body)
	}
	if n, _ := users(); n != 2 {
		t.Fatalf("users after delete = %d, want 2", n)
	}
	if code, body := post("/api/readonly", ro("shop2")); code != 200 {
		t.Fatalf("create read-only after delete: %d %s", code, body)
	}
	// and full again means full for the ordinary path too
	if code, body := post("/api/users", `{"user":"bob","password":"bob-password-123"}`); code != http.StatusForbidden || !strings.Contains(body, `"user_limit"`) {
		t.Errorf("refilled, create ordinary: %d %s, want 403 user_limit", code, body)
	}
}

// The two creation paths share the limit, so they share the lock:
// twenty parallel creations, half ordinary and half read-only, must
// land exactly the two places an unregistered installation has beside
// its administrator — never three, whichever path they took.
func TestUserLimitRaceMixedPaths(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	ck := apiLogin(t, h, pw)
	var wg sync.WaitGroup
	codes := make([]int, 20)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				body := fmt.Sprintf(`{"user":"mixuser%d","password":"a-long-enough-password"}`, i)
				codes[i] = apiCall(h, "POST", "/api/users", body, ck, "").Code
			} else {
				body := fmt.Sprintf(`{"user":"mixro%d","password":"a-long-enough-password","kind":"vip","object":"203.0.113.10"}`, i)
				codes[i] = apiCall(h, "POST", "/api/readonly", body, ck, "").Code
			}
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == 200 {
			ok++
		}
	}
	if ok != 2 {
		t.Fatalf("parallel mixed creates landed %d, want 2 (limit 3 incl. admin); codes %v", ok, codes)
	}
	w := apiCall(h, "GET", "/api/users", "", ck, "")
	var out struct {
		Users []string `json:"users"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || len(out.Users) != 3 {
		t.Fatalf("user list after mixed race: %d users (%v) %s", len(out.Users), err, w.Body)
	}
}
