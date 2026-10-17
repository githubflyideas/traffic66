package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The sponsor banner of unregistered installations. The browser first loads
// it straight from GitHub; when it cannot (intranet, proxy, blocked), the
// page asks this server, which keeps the last set it managed to download in
// <data>/sponsor and serves that. The download is a plain GET with no data
// about the user or the installation, and is skipped while registered.

var (
	sponsorBase     = "https://raw.githubusercontent.com/githubflyideas/ads-sponsor/main/traffic66/"
	sponsorEvery    = 6 * time.Hour
	sponsorTimeout  = 10 * time.Second
	sponsorClient   = &http.Client{Timeout: sponsorTimeout}
	sponsorImgTypes = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/gif": ".gif"}
	sponsorHTTPS    = true // images over https only; tests use plain http
	sponsorName     = regexp.MustCompile(`^[0-3]\.(png|jpg|webp|gif)$`)
)

const (
	sponsorMaxSlides   = 4
	sponsorMaxImage    = 150 << 10
	sponsorMaxManifest = 64 << 10
)

// SponsorSlide is one slide as kept on disk and served to the page.
type SponsorSlide struct {
	ID    string `json:"id,omitempty"`
	Image string `json:"image"`
	Link  string `json:"link,omitempty"`
	Alt   string `json:"alt"`
	Title string `json:"title,omitempty"`
}

type sponsorManifest struct {
	Version int            `json:"version"`
	Updated string         `json:"updated,omitempty"`
	Slides  []SponsorSlide `json:"slides"`
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func httpLink(v string, base *url.URL, httpsOnly bool) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil || v == "" {
		return ""
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if u.Scheme == "https" || (!httpsOnly && u.Scheme == "http") {
		if u.Host != "" {
			return u.String()
		}
	}
	return ""
}

// validSponsor checks a downloaded manifest: at most four slides, https
// images, http or https links (others dropped, the slide not clickable).
func validSponsor(b []byte, base string) ([]SponsorSlide, error) {
	var m sponsorManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("sponsor: manifest: %w", err)
	}
	bu, _ := url.Parse(base)
	var out []SponsorSlide
	for _, s := range m.Slides {
		if len(out) == sponsorMaxSlides {
			break
		}
		img := httpLink(s.Image, bu, sponsorHTTPS)
		if img == "" {
			continue
		}
		out = append(out, SponsorSlide{ID: clip(s.ID, 60), Image: img, Link: httpLink(s.Link, nil, false),
			Alt: clip(s.Alt, 120), Title: clip(s.Title, 80)})
	}
	if len(out) == 0 {
		return nil, errors.New("sponsor: no usable slides")
	}
	return out, nil
}

// sponsorImage checks one downloaded image and returns its file extension.
func sponsorImage(b []byte) (string, error) {
	if len(b) == 0 || len(b) > sponsorMaxImage {
		return "", fmt.Errorf("sponsor: image of %d bytes", len(b))
	}
	ext, ok := sponsorImgTypes[http.DetectContentType(b)]
	if !ok {
		return "", fmt.Errorf("sponsor: image type %s", http.DetectContentType(b))
	}
	return ext, nil
}

func sponsorGet(ctx context.Context, u string, max int64, ua string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	resp, err := sponsorClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("%s: larger than %d bytes", u, max)
	}
	return b, nil
}

// refreshSponsor downloads the manifest and its images into dir. The set on
// disk is replaced only by a complete new one with at least one image, so
// the last good set stays when GitHub cannot be reached.
func refreshSponsor(ctx context.Context, dir, ua string) error {
	b, err := sponsorGet(ctx, sponsorBase+"sponsors.json", sponsorMaxManifest, ua)
	if err != nil {
		return err
	}
	slides, err := validSponsor(b, sponsorBase+"sponsors.json")
	if err != nil {
		return err
	}
	tmp := dir + ".new"
	os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var keep []SponsorSlide
	for _, s := range slides {
		ib, err := sponsorGet(ctx, s.Image, sponsorMaxImage, ua)
		if err != nil {
			continue
		}
		ext, err := sponsorImage(ib)
		if err != nil {
			continue
		}
		name := fmt.Sprintf("%d%s", len(keep), ext)
		if err := os.WriteFile(filepath.Join(tmp, name), ib, 0o644); err != nil {
			return err
		}
		s.Image = name
		keep = append(keep, s)
	}
	if len(keep) == 0 {
		return errors.New("sponsor: no image could be downloaded")
	}
	mb, _ := json.Marshal(sponsorManifest{Version: 1, Updated: time.Now().UTC().Format(time.RFC3339), Slides: keep})
	if err := os.WriteFile(filepath.Join(tmp, "sponsors.json"), mb, 0o644); err != nil {
		return err
	}
	old := dir + ".old"
	os.RemoveAll(old)
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, old); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		os.Rename(old, dir)
		return err
	}
	os.RemoveAll(old)
	return nil
}

func (s *Server) sponsorDir() string { return filepath.Join(s.DataDir, "sponsor") }

func (s *Server) registered() bool { return s.License != nil && s.License.State().Registered() }

// RunSponsor keeps the sponsor cache fresh: at start and every six hours,
// and only while the installation is not registered.
func (s *Server) RunSponsor(ctx context.Context) {
	if s.DataDir == "" {
		return
	}
	ua := "traffic66/" + s.Version
	for {
		if !s.registered() {
			c, cancel := context.WithTimeout(ctx, time.Minute)
			if err := refreshSponsor(c, s.sponsorDir(), ua); err != nil {
				log.Printf("sponsor: %v (keeping the cached banner, if any)", err)
			}
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(sponsorEvery):
		}
	}
}

// getSponsor serves the cached manifest with the images pointing here.
func (s *Server) getSponsor(w http.ResponseWriter, r *http.Request) {
	if s.registered() || s.DataDir == "" {
		http.NotFound(w, r)
		return
	}
	b, err := os.ReadFile(filepath.Join(s.sponsorDir(), "sponsors.json"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var m sponsorManifest
	if json.Unmarshal(b, &m) != nil {
		http.NotFound(w, r)
		return
	}
	var out []SponsorSlide
	for _, sl := range m.Slides {
		if sponsorName.MatchString(sl.Image) && len(out) < sponsorMaxSlides {
			sl.Image = "/api/sponsor/img/" + sl.Image
			out = append(out, sl)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"slides": out})
}

func (s *Server) getSponsorImg(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if s.registered() || s.DataDir == "" || !sponsorName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	b, err := os.ReadFile(filepath.Join(s.sponsorDir(), name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ext, err := sponsorImage(b)
	if err != nil || ext != filepath.Ext(name) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(b))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(b)
}
