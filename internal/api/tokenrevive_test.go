package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A token belongs to an account, not to a name: deleting the account
// ends its tokens, and an account made later under the same name does not
// inherit them.
func TestDeletedUsersTokensDoNotReturn(t *testing.T) {
	_, h, pw := apiRulesServer(t)
	cookie := apiLogin(t, h, pw)
	call := func(method, path, body string) (int, string) {
		r := apiCall(h, method, path, body, cookie, "")
		return r.Code, r.Body.String()
	}
	if c, b := call("POST", "/api/users", `{"user":"bob","password":"bob-password-1"}`); c != 200 {
		t.Fatalf("create bob: %d %s", c, b)
	}
	c, b := call("POST", "/api/tokens", `{"name":"ci","scope":"read","user":"bob"}`)
	if c != 200 {
		t.Fatalf("token: %d %s", c, b)
	}
	var made struct{ Token string }
	json.Unmarshal([]byte(b), &made)
	use := func() int { return apiCall(h, "GET", "/api/me", "", "", made.Token).Code }
	if got := use(); got != http.StatusOK {
		t.Fatalf("token while bob exists: %d", got)
	}
	if c, b := call("DELETE", "/api/users?user=bob", ""); c != 200 {
		t.Fatalf("delete bob: %d %s", c, b)
	}
	if got := use(); got != http.StatusUnauthorized {
		t.Errorf("token after bob was deleted: %d, want 401", got)
	}
	if c, b := call("POST", "/api/users", `{"user":"bob","password":"another-pass-22"}`); c != 200 {
		t.Fatalf("recreate bob: %d %s", c, b)
	}
	if got := use(); got != http.StatusUnauthorized {
		t.Errorf("the old bob's token works for the new bob: %d, want 401", got)
	}
	if c, b := call("GET", "/api/tokens", ""); c != 200 || strings.Contains(b, `"bob"`) {
		t.Errorf("a deleted user's token is still listed: %d %s", c, b)
	}
}
