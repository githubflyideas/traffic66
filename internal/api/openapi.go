package api

import (
	"net/http"
	"sort"
	"strings"
)

// openapi serves the machine-readable description of the HTTP API, built
// from the same route table the 405 fallback answers from, so the
// document cannot drift from what the server actually routes. The
// parameters it lists are the ones parseQuery reads for every query;
// path templates ({name}) are single-segment parameters.
func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	type op struct {
		Summary   string                `json:"summary"`
		Params    []map[string]any      `json:"parameters,omitempty"`
		Responses map[string]any        `json:"responses"`
		Security  []map[string][]string `json:"security"`
	}
	paths := map[string]map[string]op{}
	queryParams := []map[string]any{
		{"name": "range", "in": "query", "schema": map[string]any{"type": "string"}, "description": "window: 5m, 15m, 1h, 6h, 12h, 24h, 7d, 30d; an unknown value is an error (bad_range)"},
		{"name": "from", "in": "query", "schema": map[string]any{"type": "string", "format": "date-time"}},
		{"name": "to", "in": "query", "schema": map[string]any{"type": "string", "format": "date-time"}},
		{"name": "tz", "in": "query", "schema": map[string]any{"type": "integer"}, "description": "viewer zone offset, seconds east of UTC"},
		{"name": "f", "in": "query", "schema": map[string]any{"type": "string"}, "description": "filters as a JSON array of {f, v, neg}"},
	}
	errResp := map[string]any{"description": "error", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{
		"error":   map[string]any{"type": "string", "description": "short human-readable message"},
		"code":    map[string]any{"type": "string", "description": "stable machine-readable code, snake_case; programs branch on this"},
		"message": map[string]any{"type": "string", "description": "only where a handler adds detail of its own"},
		"kind":    map[string]any{"type": "string", "description": "only for database errors: memory or storage"},
	}}}}}
	responses := map[string]any{
		"200": map[string]any{"description": "ok"},
		"400": errResp, "401": errResp, "403": errResp, "404": errResp, "405": errResp,
	}
	seen := map[string]bool{}
	for _, rt := range apiRouteTable {
		key := rt.method + " " + rt.pat
		if seen[key] {
			continue
		}
		seen[key] = true
		var params []map[string]any
		for _, seg := range strings.Split(strings.Trim(rt.pat, "/"), "/") {
			if strings.HasPrefix(seg, "{") {
				params = append(params, map[string]any{
					"name": strings.Trim(seg, "{}"), "in": "path", "required": true,
					"schema": map[string]any{"type": "string"},
				})
			}
		}
		if rt.method == "GET" {
			params = append(params, queryParams...)
		}
		m := paths[rt.pat]
		if m == nil {
			m = map[string]op{}
			paths[rt.pat] = m
		}
		m[strings.ToLower(rt.method)] = op{
			Summary:   rt.method + " " + rt.pat,
			Params:    params,
			Responses: responses,
			Security:  []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
		}
	}
	// The utility paths outside /api/: served by this same program, so a
	// client reading only this document should still learn they exist.
	// They are not in apiRouteTable — that table is the API's, and the
	// route-table test keeps the two apart on purpose. All four are
	// public (security: []): a health probe, the one-click sign-in, the
	// logo and the TLS certificate are handed to anyone who asks, the
	// certificate because somebody who does not trust it yet cannot
	// sign in to fetch it.
	public := []map[string][]string{}
	for _, u := range []struct{ path, summary string }{
		{"/healthz", "health probe: 200 while storage answers, 503 otherwise"},
		{"/auto", "one-click sign-in with this machine's token (the TUI's door)"},
		{"/logo", "the installation's logo image"},
		{"/tls.crt", "the self-signed TLS certificate the web UI is served with, for trust stores"},
	} {
		paths[u.path] = map[string]op{"get": {
			Summary:   "GET " + u.path + " — " + u.summary,
			Responses: responses,
			Security:  public,
		}}
	}
	names := make([]string, 0, len(paths))
	for p := range paths {
		names = append(names, p)
	}
	sort.Strings(names)
	ordered := make(map[string]any, len(paths))
	for _, p := range names {
		ordered[p] = paths[p]
	}
	writeJSON(w, 200, map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "traffic66 API",
			"version":     apiVersion,
			"description": "Every route also answers under /api/v1/…; every /api response carries an API-Version header. Errors are JSON {\"error\": message, \"code\": code} — a short sentence for a person, a stable snake_case code for a program; a wrong method on a known path is 405 with an Allow header. This document itself is public; the routes it describes need a session cookie or a scoped API token (Authorization: Bearer t66_…; read tokens may only GET), except the four utility paths marked security: [].",
		},
		"paths": ordered,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "description": "API token from the account page or `traffic66 gen-token`"},
				"cookieAuth": map[string]any{"type": "apiKey", "in": "cookie", "name": "t66"},
			},
		},
	})
}
