package api

import (
	"os"
	"regexp"
	"testing"
)

// The route table (apiRouteTable) is written twice: once where the routes
// are registered and once as path templates for the 405 and OpenAPI
// answers. A route added to one and not the other would answer 404 where
// it should answer 405, or be missing from /api/openapi.json. The two are
// compared on the source, so adding a route to only one side fails here.
func TestRouteTableMatchesRegistrations(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	reg := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?:\bapi|\badminAPI|mux\.Handle(?:Func)?)\("(GET|POST|PUT|DELETE) (/[^"]*)"`).FindAllStringSubmatch(string(src), -1) {
		reg[m[1]+" "+m[2]] = true
	}
	// not part of the API: served as pages or files, outside the table
	for _, k := range []string{"GET /auto", "GET /healthz", "GET /logo", "GET /tls.crt"} {
		if !reg[k] {
			t.Errorf("expected the non-API route %q to be registered", k)
		}
		delete(reg, k)
	}
	tbl := map[string]bool{}
	for _, rt := range apiRouteTable {
		tbl[rt.method+" "+rt.pat] = true
	}
	for k := range reg {
		if !tbl[k] {
			t.Errorf("registered but not in apiRouteTable: %s", k)
		}
	}
	for k := range tbl {
		if !reg[k] {
			t.Errorf("in apiRouteTable but not registered: %s", k)
		}
	}
	if len(reg) < 40 {
		t.Errorf("only %d routes found: the pattern no longer matches the registrations", len(reg))
	}
}
