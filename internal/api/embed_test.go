package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/embed"
	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A platform's button opens one VIP's bandwidth: in to it and out of it,
// for the addresses its key covers, from the sites it names.
func TestEmbedLinks(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ts := time.Now().UTC().Add(-20 * time.Minute)
	vip, user := "203.0.113.10", "198.51.100.7"
	base := store.Row{TS: ts, Client: user, Server: vip, Dir: store.DirTransit, Pkts: 1, Flows: 1, Exporter: "10.0.0.1"}
	req, resp := base, base
	req.Src, req.Dst, req.Wire = user, vip, 60_000
	resp.Src, resp.Dst, resp.Wire = vip, user, 600_000
	if err := st.AppendRows([]store.Row{req, resp}); err != nil {
		t.Fatal(err)
	}
	emb, _ := embed.Open(filepath.Join(dir, "embeds.json"))
	c, priv := signed(t, dir, license.File{})
	s := &Server{Store: st, License: c, Embeds: emb}

	page := func(url, referer string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", url, nil)
		r.Header.Set("Referer", referer)
		s.embedPage(w, r)
		return w
	}

	// links are for registered installations
	w := httptest.NewRecorder()
	s.putEmbed(w, httptest.NewRequest("POST", "/api/embeds", strings.NewReader(`{"name":"Assets","ranges":["203.0.113.0/24"],"origins":["assets.example.com"]}`)))
	if w.Code != 403 {
		t.Fatalf("unregistered: %d", w.Code)
	}
	install(t, s, sign(priv, license.File{Customer: "Example KK", Product: "traffic66",
		License: license.TypeSubscription, Tier: license.TierStandard, MaxUsers: 25, Features: []string{license.FeatSyslog}}))
	w = httptest.NewRecorder()
	s.putEmbed(w, httptest.NewRequest("POST", "/api/embeds", strings.NewReader(`{"name":"Assets","ranges":["203.0.113.0/24"],"origins":["assets.example.com"]}`)))
	var made struct{ Key string }
	json.Unmarshal(w.Body.Bytes(), &made)
	if w.Code != 200 || made.Key == "" {
		t.Fatalf("make: %d %s", w.Code, w.Body)
	}

	link := "/embed/vip?ip=" + vip + "&k=" + made.Key
	from := "https://assets.example.com/vip/42"
	if w := page(link, from); w.Code != 200 || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors assets.example.com") {
		t.Fatalf("page: %d %q", w.Code, w.Header().Get("Content-Security-Policy"))
	}
	if w := page(link, "https://elsewhere.example.org/"); w.Code != 403 {
		t.Errorf("another site: %d", w.Code)
	}
	if w := page("/embed/vip?ip=198.51.100.7&k="+made.Key, from); w.Code != 403 {
		t.Errorf("an address outside the key: %d", w.Code)
	}
	if w := page("/embed/vip?ip="+vip+"&k=t66e_wrong", from); w.Code != 403 {
		t.Errorf("a wrong key: %d", w.Code)
	}

	w = httptest.NewRecorder()
	s.embedData(w, httptest.NewRequest("GET", link+"&range=1h", nil))
	var out struct {
		IP      string
		In, Out []float64
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
		t.Fatalf("data: %d %s", w.Code, w.Body)
	}
	sum := func(v []float64) (n float64) {
		for _, x := range v {
			n += x
		}
		return
	}
	// in to the VIP and out of it, the download ten times the request
	if in, o := sum(out.In), sum(out.Out); out.IP != vip || in <= 0 || o/in < 9.99 || o/in > 10.01 {
		t.Fatalf("in %v out %v", sum(out.In), sum(out.Out))
	}
	// nothing but the bandwidth
	var raw map[string]any
	json.Unmarshal(w.Body.Bytes(), &raw)
	for k := range raw {
		switch k {
		case "ip", "step", "times", "in", "out", "settled_to", "data_from":
		default:
			t.Errorf("the data carries %q", k)
		}
	}
}
