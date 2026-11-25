package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
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
func signed(t *testing.T, dir string, f license.File) (*license.Checker, *ecdsa.PrivateKey) {
	t.Helper()
	c, err := license.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	c.Ring = license.SingleKey(&priv.PublicKey)
	return c, priv
}

func sign(priv *ecdsa.PrivateKey, f license.File) string {
	f.KeyID = license.KeyID(&priv.PublicKey)
	digest := sha512.Sum512(license.Message(f))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		panic(err)
	}
	f.Signature = base64.StdEncoding.EncodeToString(sig)
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
	wrongKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	past := "2020-01-01"
	elsewhere := "ZZZZZZZZ"
	for _, tc := range []struct{ name, text, want string }{
		{"not json", "{nonsense", "lic_unreadable"},
		{"unsigned", `{"customer":"X","product":"traffic66"}`, "lic_unsigned"},
		{"another key", sign(wrongKey, license.File{Customer: "X", Product: "traffic66"}), "lic_"},
		{"expired", sign(priv, license.File{Customer: "X", Product: "traffic66", Expires: &past}), "lic_expired"},
		{"another installation", sign(priv, license.File{Customer: "X", Product: "traffic66", InstallationID: elsewhere}), "lic_other_install"},
	} {
		w := install(t, s, tc.text)
		if w.Code == 200 {
			t.Errorf("%s: accepted", tc.name)
			continue
		}
		// a code the page says in the reader's language, and the
		// checker's sentence for support
		var j struct{ Error, Detail string }
		json.Unmarshal(w.Body.Bytes(), &j)
		if !strings.HasPrefix(j.Error, tc.want) || j.Error == "lic_invalid" || j.Detail == "" {
			t.Errorf("%s: %s, want code %q and a detail", tc.name, strings.TrimSpace(w.Body.String()), tc.want)
		}
		if c.State().Registered() {
			t.Fatalf("%s: the installation became registered", tc.name)
		}
		if _, err := license.Load(c.Path); err == nil {
			t.Fatalf("%s: a refused licence was written to disk", tc.name)
		}
	}
}
