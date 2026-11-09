package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/license"
)

// signed builds a licence the test's own key signs, and a checker that
// accepts that key: a release build does not ship it.
func signed(t *testing.T, dir string, f license.File) (*license.Checker, ed25519.PrivateKey) {
	t.Helper()
	c, err := license.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	c.Ring = license.SingleKey(priv.Public().(ed25519.PublicKey))
	return c, priv
}

func sign(priv ed25519.PrivateKey, f license.File) string {
	f.KeyID = license.KeyID(priv.Public().(ed25519.PublicKey))
	f.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, license.Message(f)))
	b, _ := json.Marshal(f)
	return string(b)
}

func install(t *testing.T, s *Server, text string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"text": text})
	s.putLicense(w, httptest.NewRequest("POST", "/api/license", strings.NewReader(string(body))))
	return w
}

// Installing a licence from the page is the one step that turns a paying
// customer into a registered installation, and it used to need a shell on
// the machine.
func TestLicenseInstalls(t *testing.T) {
	dir := t.TempDir()
	c, priv := signed(t, dir, license.File{})
	s := &Server{License: c}
	if c.State().Registered() {
		t.Fatal("a fresh data directory is registered?")
	}
	text := sign(priv, license.File{
		Customer: "Example KK", Product: "traffic66",
		License: license.TypeSubscription, Tier: license.TierStandard,
		MaxUsers: 25, Features: []string{license.FeatSyslog},
	})
	if w := install(t, s, text); w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	// in force at once, not at the next few-hourly check
	if !c.State().Registered() {
		t.Errorf("still unregistered after installing: %s", c.State().Reason)
	}
	if c.State().Watermark() {
		t.Error("a registered installation must not watermark")
	}
	// and on disk, so a restart keeps it
	again, err := license.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	again.Ring = c.Ring
	if st := again.Check(time.Now()); !st.Registered() {
		t.Errorf("not registered after a restart: %s", st.Reason)
	}
}

// Nothing is written until the licence has been checked the way the
// checker checks it at startup. A file that cannot be used must be refused
// with the reason, not saved and found useless later.
func TestLicenseRefusedIsNotWritten(t *testing.T) {
	dir := t.TempDir()
	c, priv := signed(t, dir, license.File{})
	s := &Server{License: c}
	other := make([]byte, ed25519.SeedSize)
	for i := range other {
		other[i] = byte(200 - i)
	}
	wrongKey := ed25519.NewKeyFromSeed(other)
	past := "2020-01-01"
	elsewhere := "ZZZZZZZZ"
	for _, tc := range []struct{ name, text, want string }{
		{"not json", "{nonsense", "license"},
		{"unsigned", `{"customer":"X","product":"traffic66"}`, "license"},
		{"another key", sign(wrongKey, license.File{Customer: "X", Product: "traffic66"}), ""},
		{"expired", sign(priv, license.File{Customer: "X", Product: "traffic66", Expires: &past}), "expired"},
		{"another installation", sign(priv, license.File{Customer: "X", Product: "traffic66", InstallationID: elsewhere}), "another installation"},
	} {
		w := install(t, s, tc.text)
		if w.Code == 200 {
			t.Errorf("%s: accepted", tc.name)
			continue
		}
		if tc.want != "" && !strings.Contains(w.Body.String(), tc.want) {
			t.Errorf("%s: %s, want a reason mentioning %q", tc.name, strings.TrimSpace(w.Body.String()), tc.want)
		}
		if c.State().Registered() {
			t.Fatalf("%s: the installation became registered", tc.name)
		}
		if _, err := license.Load(c.Path); err == nil {
			t.Fatalf("%s: a refused licence was written to disk", tc.name)
		}
	}
}
