package api

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/assets"
	"github.com/githubflyideas/traffic66/internal/auth"
	"github.com/githubflyideas/traffic66/internal/license"
)

// licensed writes a licence into dir granting the given features and the
// given quotas, signed by a key only this test knows, and returns a
// checker that trusts that key. It never touches the release keyring, so
// the test says nothing about which keys we actually ship.
func licensed(t *testing.T, dir string, feats []string, sites, racks, devices int) *license.Checker {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	f := license.File{
		KeyID: license.KeyID(pub), Customer: "Test", Product: "traffic66",
		License: license.TypePerpetual, Tier: license.TierStandard,
		MaxUsers: 10, MaxSites: sites, MaxRacks: racks, MaxDevices: devices,
		Features: license.CanonFeatures(feats),
	}
	f.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, license.Message(f)))
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, license.FileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := license.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c.Ring = license.Keyring{Keys: []license.Key{{ID: license.KeyID(pub), Pub: pub}}}
	c.Check(time.Now())
	return c
}

// assetServer is a signed-in server with the ledger open.
func assetServer(t *testing.T, feats []string, sites, racks, devices int) (*Server, func(method, path, body, user string) *httptest.ResponseRecorder) {
	t.Helper()
	dir := t.TempDir()
	if err := auth.Set(dir, "admin", "s3cret-pass"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Set(dir, "viewer", "s3cret-pass"); err != nil {
		t.Fatal(err)
	}
	fc := auth.NewFileChecker(dir, nil)
	al, err := assets.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { al.Close() })
	s := &Server{Accounts: fc, DataDir: dir, Assets: al, sessions: map[string]session{}}
	if feats != nil {
		s.License = licensed(t, dir, feats, sites, racks, devices)
	}
	mux := http.NewServeMux()
	// the handlers under test, without the session middleware: the user is
	// put on the request directly so the test is about the ledger's own
	// two gates, not about signing in
	mux.HandleFunc("GET /api/assets", s.getAssets)
	mux.HandleFunc("GET /api/assets/ipam", s.getIPAM)
	mux.HandleFunc("GET /api/assets/addrs", s.getAddrs)
	mux.HandleFunc("GET /api/assets/free", s.getFreeAddrs)
	mux.HandleFunc("POST /api/assets/{kind}", s.putAsset)
	mux.HandleFunc("DELETE /api/assets/{kind}", s.deleteAsset)
	call := func(method, path, body, user string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, user))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	return s, call
}

// Without a licence granting it, the ledger is not there: every call is
// refused, and refused the same way whether the caller is the
// administrator or not.
func TestAssetsNeedLicence(t *testing.T) {
	_, call := assetServer(t, nil, 0, 0, 0)
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/api/assets", ""},
		{"GET", "/api/assets/ipam", ""},
		{"POST", "/api/assets/site", `{"name":"A"}`},
		{"DELETE", "/api/assets/site?id=1", ""},
	} {
		for _, u := range []string{"admin", "viewer"} {
			w := call(c.method, c.path, c.body, u)
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s %s as %s: %d %s", c.method, c.path, u, w.Code, w.Body)
			}
			if !strings.Contains(w.Body.String(), "not_registered") {
				t.Fatalf("%s %s: %s", c.method, c.path, w.Body)
			}
		}
	}
	// a licence granting something else is no licence for this one
	_, call = assetServer(t, []string{license.FeatSyslog}, 0, 0, 0)
	if w := call("GET", "/api/assets", "", "admin"); w.Code != http.StatusForbidden {
		t.Fatalf("with another feature only: %d %s", w.Code, w.Body)
	}
}

// With the licence, everyone signed in reads the ledger and only the
// administrator writes it.
func TestAssetsAdminWrites(t *testing.T) {
	_, call := assetServer(t, []string{license.FeatDCIM}, 0, 0, 0)
	if w := call("GET", "/api/assets", "", "viewer"); w.Code != 200 {
		t.Fatalf("viewer reading: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/assets/site", `{"name":"Site A"}`, "viewer"); w.Code != http.StatusForbidden {
		t.Fatalf("viewer writing: %d %s", w.Code, w.Body)
	}
	w := call("POST", "/api/assets/site", `{"name":"Site A"}`, "admin")
	if w.Code != 200 {
		t.Fatalf("admin writing: %d %s", w.Code, w.Body)
	}
	var site assets.Site
	json.Unmarshal(w.Body.Bytes(), &site)
	if site.ID == 0 {
		t.Fatalf("no id came back: %s", w.Body)
	}
	if w := call("DELETE", fmt.Sprintf("/api/assets/site?id=%d", site.ID), "", "viewer"); w.Code != http.StatusForbidden {
		t.Fatalf("viewer deleting: %d %s", w.Code, w.Body)
	}
	if w := call("DELETE", fmt.Sprintf("/api/assets/site?id=%d", site.ID), "", "admin"); w.Code != 200 {
		t.Fatalf("admin deleting: %d %s", w.Code, w.Body)
	}
}

// Hidden records are the administrator's: nobody else sees them, nor the
// ports and cables that hang off them.
func TestAssetsHidden(t *testing.T) {
	s, call := assetServer(t, []string{license.FeatDCIM}, 0, 0, 0)
	open, err := s.Assets.SaveDevice(assets.Device{Name: "open"})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := s.Assets.SaveDevice(assets.Device{Name: "secret", Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	p1, _ := s.Assets.SavePort(assets.Port{DeviceID: open.ID, Name: "a"})
	p2, _ := s.Assets.SavePort(assets.Port{DeviceID: secret.ID, Name: "b"})
	if _, err := s.Assets.SaveLink(assets.Link{APort: p1.ID, BPort: p2.ID}); err != nil {
		t.Fatal(err)
	}
	read := func(user string) (devices []assets.Device, ports []assets.Port, links []assets.Link) {
		w := call("GET", "/api/assets", "", user)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", user, w.Code, w.Body)
		}
		var out struct {
			Devices []assets.Device `json:"devices"`
			Ports   []assets.Port   `json:"ports"`
			Links   []assets.Link   `json:"links"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Devices, out.Ports, out.Links
	}
	if d, p, l := read("admin"); len(d) != 2 || len(p) != 2 || len(l) != 1 {
		t.Fatalf("admin sees %d devices, %d ports, %d links", len(d), len(p), len(l))
	}
	d, p, l := read("viewer")
	if len(d) != 1 || d[0].Name != "open" {
		t.Fatalf("viewer sees devices %+v", d)
	}
	if len(p) != 1 || p[0].ID != p1.ID {
		t.Fatalf("viewer sees ports %+v", p)
	}
	if len(l) != 0 {
		t.Fatalf("viewer sees a cable into a hidden device: %+v", l)
	}
}

// The quotas stop a new record, never an edit of one that is already
// there: a quota that blocked a correction would be a trap.
func TestAssetsQuota(t *testing.T) {
	_, call := assetServer(t, []string{license.FeatDCIM}, 1, 0, 2)
	w := call("POST", "/api/assets/site", `{"name":"Site A"}`, "admin")
	if w.Code != 200 {
		t.Fatalf("first site: %d %s", w.Code, w.Body)
	}
	var site assets.Site
	json.Unmarshal(w.Body.Bytes(), &site)
	if w := call("POST", "/api/assets/site", `{"name":"Site B"}`, "admin"); w.Code != http.StatusForbidden {
		t.Fatalf("second site past a quota of 1: %d %s", w.Code, w.Body)
	} else if !strings.Contains(w.Body.String(), "quota") {
		t.Fatalf("second site: %s", w.Body)
	}
	// the one that exists can still be corrected
	if w := call("POST", "/api/assets/site", fmt.Sprintf(`{"id":%d,"name":"Site A2"}`, site.ID), "admin"); w.Code != 200 {
		t.Fatalf("editing inside the quota: %d %s", w.Code, w.Body)
	}
	// racks are not limited here (0), devices are
	if w := call("POST", "/api/assets/rack", fmt.Sprintf(`{"site_id":%d,"name":"R1","units":42}`, site.ID), "admin"); w.Code != 200 {
		t.Fatalf("rack with no limit: %d %s", w.Code, w.Body)
	}
	for i, want := range []int{200, 200, http.StatusForbidden} {
		w := call("POST", "/api/assets/device", fmt.Sprintf(`{"name":"d%d"}`, i), "admin")
		if w.Code != want {
			t.Fatalf("device %d: %d %s", i, w.Code, w.Body)
		}
	}
}

// The IPAM pages answer with what the ledger holds.
func TestAssetsIPAM(t *testing.T) {
	_, call := assetServer(t, []string{license.FeatDCIM}, 0, 0, 0)
	if w := call("POST", "/api/assets/subnet", `{"prefix":"10.9.9.0/29","gateway":"10.9.9.1"}`, "admin"); w.Code != 200 {
		t.Fatalf("subnet: %d %s", w.Code, w.Body)
	}
	var sn assets.Subnet
	w := call("GET", "/api/assets/ipam", "", "viewer")
	if w.Code != 200 {
		t.Fatalf("ipam: %d %s", w.Code, w.Body)
	}
	var out struct {
		Subnets []assets.SubnetUse `json:"subnets"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Subnets) != 1 || out.Subnets[0].Hosts != 6 {
		t.Fatalf("ipam: %s", w.Body)
	}
	sn = out.Subnets[0].Subnet
	if w := call("POST", "/api/assets/addr", `{"ip":"10.9.9.2","name":"printer"}`, "admin"); w.Code != 200 {
		t.Fatalf("addr: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/api/assets/addr", `{"ip":"10.9.9.2","name":"someone else"}`, "admin"); w.Code == 200 {
		t.Fatal("the address was assigned twice")
	}
	w = call("GET", fmt.Sprintf("/api/assets/free?subnet=%d&n=3", sn.ID), "", "viewer")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"10.9.9.3"`) || strings.Contains(w.Body.String(), `"10.9.9.2"`) {
		t.Fatalf("free: %d %s", w.Code, w.Body)
	}
	w = call("GET", fmt.Sprintf("/api/assets/addrs?subnet=%d", sn.ID), "", "viewer")
	if !strings.Contains(w.Body.String(), "printer") {
		t.Fatalf("addrs: %s", w.Body)
	}
	if w := call("POST", "/api/assets/subnet", `{"prefix":"not a network"}`, "admin"); w.Code == 200 {
		t.Fatal("rubbish was accepted as a network")
	}
}
