package api

import (
	"net/http"
	"sort"
)

// rings: traffic by an inner dimension and, within each, by an outer one,
// for the two-ring charts on Traffic details.
func (s *Server) rings(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	v := r.URL.Query()
	rs, err := s.Store.Rings(q, v.Get("inner"), v.Get("outer"), 8, 12)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

// geoLines: traffic between the countries of the internal networks (set
// in the names) and remote countries, for lines on the world map.
func (s *Server) geoLines(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		fail(w, err)
		return
	}
	segs := s.Inv.SegmentCountries()
	type line struct {
		From string  `json:"from"`
		To   string  `json:"to"`
		Wire float64 `json:"v"`
	}
	origins := map[string]bool{}
	for _, cc := range segs {
		origins[cc] = true
	}
	out := map[[2]string]float64{}
	if len(segs) > 0 {
		rows, err := s.Store.SegmentCountries(q)
		if err != nil {
			fail(w, err)
			return
		}
		for _, x := range rows {
			from := segs[x.Segment]
			if from == "" || from == x.Country {
				continue
			}
			out[[2]string{from, x.Country}] += x.Wire
		}
	}
	lines := []line{}
	for k, v := range out {
		lines = append(lines, line{k[0], k[1], v})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].Wire > lines[j].Wire })
	var os []string
	for cc := range origins {
		os = append(os, cc)
	}
	sort.Strings(os)
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "origins": os})
}
