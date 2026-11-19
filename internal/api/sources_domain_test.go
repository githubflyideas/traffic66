package api

import (
	"encoding/json"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/enrich"
)

// A device with two observation domains: the first carries records, the
// second never sent its template, so all of its data is lost. The second
// may not be left out of the list as a silent domain: it is the one to see.
func TestSourcesShowADomainWaitingForItsTemplate(t *testing.T) {
	col := collector.New(nullSink{})
	tmpl := encode.Template{ID: 256, Fields: []encode.Field{{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2},
		{ID: 4, Len: 1}, {ID: 1, Len: 4}, {ID: 2, Len: 4}, {ID: 22, Len: 4}, {ID: 21, Len: 4}}}
	rec := encode.Values{encode.A(netip.MustParseAddr("10.0.0.1")), encode.A(netip.MustParseAddr("192.0.2.9")), encode.U(50000, 2), encode.U(443, 2),
		encode.U(6, 1), encode.U(1500, 4), encode.U(1, 4), encode.U(90000, 4), encode.U(99000, 4)}
	dev := netip.MustParseAddr("192.0.2.61")
	now := time.Now()
	for i := uint32(1); i <= 5; i++ {
		m := encode.NewV9()
		m.AddTemplate(tmpl)
		m.AddData(256, []encode.Values{rec})
		col.Handle(m.V9(100000, uint32(now.Unix()), i, 1), dev, now)
		m = encode.NewV9()
		m.AddData(256, []encode.Values{rec, rec})
		col.Handle(m.V9(100000, uint32(now.Unix()), i, 2), dev, now)
	}
	s := &Server{Col: col, Inv: enrich.NewInventory(), sessions: map[string]session{}}
	w := httptest.NewRecorder()
	s.sources(w, httptest.NewRequest("GET", "/api/sources", nil))
	var out struct {
		Sources []struct {
			Domain     uint32 `json:"domain"`
			NoTemplate uint64 `json:"no_template"`
			Pending    int    `json:"pending"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err, w.Body.String())
	}
	for _, r := range out.Sources {
		if r.Domain == 2 {
			if r.NoTemplate == 0 && r.Pending == 0 {
				t.Errorf("domain 2 listed without its missing template: %+v", r)
			}
			return
		}
	}
	t.Errorf("domain 2, whose data all waits for a template, is not listed: %s", w.Body.String())
}
