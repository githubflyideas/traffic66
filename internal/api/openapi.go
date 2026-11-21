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
		"error":   map[string]any{"type": "string", "description": "stable machine-readable code, snake_case"},
		"message": map[string]any{"type": "string"},
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
			"description": "Every route also answers under /api/v1/…; every /api response carries an Api-Version header. Errors are JSON {\"error\": code} with stable snake_case codes; a wrong method on a known path is 405 with an Allow header. Sign in with a session cookie, or with a scoped API token (Authorization: Bearer t66_…; read tokens may only GET).",
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
