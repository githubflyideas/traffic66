package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/embed"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Links for other platforms: /embed/vip?ip=<address>&k=<key> is one
// address's bandwidth, in and out, and nothing else, for a button beside
// that address on the platform. The links are set up on the settings
// page; this is for registered installations.

// embedRanges are the periods the page offers; a day unless asked.
var embedRanges = map[string]time.Duration{"1h": time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour}

func (s *Server) embedRegistered() bool {
	return s.License != nil && s.License.State().Registered()
}

func (s *Server) getEmbeds(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"registered": s.embedRegistered(), "links": []embed.Source{}, "per_minute": embed.PerMinute}
	if s.Embeds != nil {
		out["links"] = s.Embeds.List()
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) putEmbed(w http.ResponseWriter, r *http.Request) {
	if !s.embedReady(w) {
		return
	}
	var in embed.Source
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	key, err := s.Embeds.Put(embed.Source{ID: in.ID, Name: in.Name, Ranges: in.Ranges, Origins: in.Origins, Clients: in.Clients})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "key": key})
}

func (s *Server) rotateEmbed(w http.ResponseWriter, r *http.Request) {
	if !s.embedReady(w) {
		return
	}
	var in struct{ ID string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	key, err := s.Embeds.Rotate(in.ID)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "key": key})
}

func (s *Server) deleteEmbed(w http.ResponseWriter, r *http.Request) {
	if s.Embeds == nil {
		fail(w, errors.New("links are not available here"))
		return
	}
	// withdrawing is allowed without a licence: a link must be stoppable
	// whatever happened to the licence since it was made
	if err := s.Embeds.Delete(r.URL.Query().Get("id")); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// embedReady answers the request unless links can be made here.
func (s *Server) embedReady(w http.ResponseWriter) bool {
	if s.Embeds == nil {
		fail(w, errors.New("links are not available here"))
		return false
	}
	if !s.embedRegistered() {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_registered"})
		return false
	}
	return true
}

// embedCheck decides one request to the page or its data. It answers the
// request itself when refusing, and says why in plain words: the person
// who set the platform up needs to tell a typo from a block.
func (s *Server) embedCheck(w http.ResponseWriter, r *http.Request, page bool) (netip.Addr, embed.Source, bool) {
	refuse := func(code int, err error) (netip.Addr, embed.Source, bool) {
		if page {
			embedRefusal(w, code, err.Error())
		} else {
			writeJSON(w, code, map[string]string{"error": err.Error()})
		}
		return netip.Addr{}, embed.Source{}, false
	}
	if s.Embeds == nil || !s.embedRegistered() {
		return refuse(http.StatusForbidden, errors.New("links to traffic are for registered installations"))
	}
	q := r.URL.Query()
	a, err := netip.ParseAddr(q.Get("ip"))
	if err != nil {
		return refuse(http.StatusBadRequest, errors.New("the link has no address to show (ip=)"))
	}
	viewer, _ := netip.ParseAddr(clientIP(r))
	src, err := s.Embeds.Check(embed.Request{Key: q.Get("k"), Addr: a.Unmap(), Viewer: viewer, Referer: r.Referer(), Page: page})
	switch {
	case errors.Is(err, embed.ErrRate):
		return refuse(http.StatusTooManyRequests, err)
	case err != nil:
		return refuse(http.StatusForbidden, err)
	}
	return a.Unmap(), src, true
}

// embedPage is the page a platform opens or frames.
func (s *Server) embedPage(w http.ResponseWriter, r *http.Request) {
	a, src, ok := s.embedCheck(w, r, true)
	if !ok {
		return
	}
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; "+embed.FrameAncestors(src.Origins))
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s · traffic66</title><link rel="icon" href="data:,"><link rel="stylesheet" href="/embed.css"></head>
<body><main id="em" data-ip="%s"></main><script src="/embed.js"></script></body></html>`, html.EscapeString(a.String()), html.EscapeString(a.String()))
}

// embedData is the bandwidth behind the page: bits per second to the
// address and from it, over the period asked.
func (s *Server) embedData(w http.ResponseWriter, r *http.Request) {
	a, _, ok := s.embedCheck(w, r, false)
	if !ok {
		return
	}
	d, ok := embedRanges[r.URL.Query().Get("range")]
	if !ok {
		d = 24 * time.Hour
	}
	now := time.Now().UTC()
	q := store.Query{From: now.Add(-d), To: now, ZoneOffset: zoneOffset(r.URL.Query().Get("tz"))}
	if d > longRange {
		q.From = q.From.Truncate(time.Hour)
	}
	series := func(field string) (*store.Series, error) {
		qq := q
		qq.Filters = []store.Filter{{Field: field, Value: a.String()}}
		return s.Store.TotalSeries(qq)
	}
	in, err := series("dst")
	if err != nil {
		fail(w, err)
		return
	}
	out, err := series("src")
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ip": a.String(), "step": in.Step, "times": in.Times,
		"in": in.Values[0], "out": out.Values[0], "settled_to": in.SettledTo, "data_from": in.DataFrom,
	})
}

// embedRefusal is the page shown instead, small enough to sit in a frame.
func embedRefusal(w http.ResponseWriter, code int, why string) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>traffic66</title></head>
<body style="font:14px system-ui,sans-serif;background:#000;color:#ccc;margin:0;padding:24px">
<p style="margin:0 0 6px;color:#fff">traffic66</p><p style="margin:0">%s</p></body></html>`, html.EscapeString(why))
}
