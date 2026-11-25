package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailHidesDatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		err  string
		code int
		kind string
	}{
		{"bad port \"x\"", 400, ""},
		{"Out of Memory Error: failed to allocate data of size 64.0 MiB (256.0 MiB/256.0 MiB used)", 503, "memory"},
		{"IO Error: Could not write file \"tmp/x.tmp\": No space left on device", 503, "storage"},
		{"database/sql/driver: could not connect to database", 503, "storage"},
	} {
		w := httptest.NewRecorder()
		fail(w, errors.New(tc.err))
		var body map[string]string
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != tc.code || body["kind"] != tc.kind {
			t.Errorf("%q: %d %v", tc.err, w.Code, body)
		}
		if tc.kind != "" && strings.Contains(w.Body.String(), "Error:") {
			t.Errorf("%q: SQL error sent to the browser: %s", tc.err, w.Body.String())
		}
	}
}
