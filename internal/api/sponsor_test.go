package api

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tinyPNG(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 1))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestValidSponsor(t *testing.T) {
	base := "https://example.com/t/sponsors.json"
	in := `{"slides":[
	 {"image":"a.png","link":"https://a.example/?ref=1","alt":"A"},
	 {"image":"http://insecure/b.png","link":"https://b","alt":"B"},
	 {"image":"c.png","link":"mailto:x@y","alt":"C"},
	 {"image":"d.png","link":"javascript:alert(1)","alt":"D"},
	 {"image":"e.png","alt":"E"},
	 {"image":"f.png","alt":"F"}]}`
	s, err := validSponsor([]byte(in), base)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 4 {
		t.Fatalf("want 4 slides, got %d", len(s))
	}
	if s[0].Image != "https://example.com/t/a.png" || s[0].Link != "https://a.example/?ref=1" {
		t.Fatalf("slide 0: %+v", s[0])
	}
	if s[1].Alt != "C" || s[1].Link != "" || s[2].Link != "" {
		t.Fatalf("bad links kept: %+v", s)
	}
	if _, err := validSponsor([]byte(`{"slides":[]}`), base); err == nil {
		t.Fatal("empty manifest accepted")
	}
	if _, err := validSponsor([]byte(`nope`), base); err == nil {
		t.Fatal("bad json accepted")
	}
}

func TestSponsorImage(t *testing.T) {
	if ext, err := sponsorImage(tinyPNG(t)); err != nil || ext != ".png" {
		t.Fatalf("png: %q %v", ext, err)
	}
	if _, err := sponsorImage([]byte("<svg xmlns='http://www.w3.org/2000/svg'/>")); err == nil {
		t.Fatal("svg accepted")
	}
	big := append(tinyPNG(t), make([]byte, sponsorMaxImage)...)
	if _, err := sponsorImage(big); err == nil {
		t.Fatal("oversized image accepted")
	}
}

func TestSponsorCacheFallback(t *testing.T) {
	pic := tinyPNG(t)
	up := true
	var ua string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		if !up {
			http.Error(w, "down", 503)
			return
		}
		switch r.URL.Path {
		case "/t/sponsors.json":
			w.Write([]byte(`{"slides":[{"image":"a.png","link":"https://a.example/","alt":"A"},{"image":"missing.png","alt":"M"}]}`))
		case "/t/a.png":
			w.Write(pic)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	old := sponsorBase
	sponsorBase = ts.URL + "/t/"
	sponsorHTTPS = false
	defer func() { sponsorBase, sponsorHTTPS = old, true }()

	data := t.TempDir()
	s := &Server{DataDir: data, Version: "9.9.9"}
	dir := s.sponsorDir()
	if err := refreshSponsor(context.Background(), dir, "traffic66/9.9.9"); err != nil {
		t.Fatal(err)
	}
	if ua != "traffic66/9.9.9" {
		t.Fatalf("user agent %q", ua)
	}
	// GitHub goes away: the last good set stays
	up = false
	if err := refreshSponsor(context.Background(), dir, "x"); err == nil {
		t.Fatal("refresh should fail while offline")
	}
	w := httptest.NewRecorder()
	s.getSponsor(w, httptest.NewRequest("GET", "/api/sponsor", nil))
	var got struct{ Slides []SponsorSlide }
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Slides) != 1 {
		t.Fatalf("cached manifest: %s %v", w.Body, err)
	}
	if got.Slides[0].Image != "/api/sponsor/img/0.png" || got.Slides[0].Link != "https://a.example/" {
		t.Fatalf("slide: %+v", got.Slides[0])
	}
	r := httptest.NewRequest("GET", "/api/sponsor/img/0.png", nil)
	r.SetPathValue("name", "0.png")
	w = httptest.NewRecorder()
	s.getSponsorImg(w, r)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), pic) || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("image: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	for _, bad := range []string{"../license.json", "sponsors.json", "9.png"} {
		r := httptest.NewRequest("GET", "/x", nil)
		r.SetPathValue("name", bad)
		w := httptest.NewRecorder()
		s.getSponsorImg(w, r)
		if w.Code != 404 {
			t.Fatalf("%s served: %d", bad, w.Code)
		}
	}
	// nothing cached at all: 404, the page shows its built-in text
	os.RemoveAll(dir)
	w = httptest.NewRecorder()
	s.getSponsor(w, httptest.NewRequest("GET", "/api/sponsor", nil))
	if w.Code != 404 {
		t.Fatalf("empty cache: %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(data, "sponsor.new")); err == nil || !strings.Contains(err.Error(), "no such") {
		t.Fatal("temporary directory left behind")
	}
}
