package api

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/license"
	"github.com/githubflyideas/traffic66/internal/syslogout"
)

// The feed is set from the findings page and in force at once; turning it
// on is for registered installations, reading or turning it off is not.
func TestSyslogSettings(t *testing.T) {
	dir := t.TempDir()
	c, priv := signed(t, dir, license.File{})
	s := &Server{License: c, DataDir: dir, Syslog: syslogout.New(syslogout.Config{})}

	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	on := `{"enabled":true,"address":"` + udp.LocalAddr().String() + `","network":"udp","format":"json","min_sev":2}`
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.putSyslog(w, httptest.NewRequest("POST", "/api/syslog", strings.NewReader(body)))
		return w
	}
	test := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.testSyslog(w, httptest.NewRequest("POST", "/api/syslog/test", nil))
		return w
	}

	if w := post(on); w.Code != 403 || !strings.Contains(w.Body.String(), "not_registered") {
		t.Fatalf("unregistered, switching on: %d %s", w.Code, w.Body)
	}
	if w := post(`{"enabled":false,"address":"","network":"udp","format":"cef","min_sev":1}`); w.Code != 200 {
		t.Fatalf("unregistered, switching off: %d %s", w.Code, w.Body)
	}
	if w := test(); w.Code != 403 {
		t.Fatalf("unregistered test: %d", w.Code)
	}

	install(t, s, sign(priv, license.File{Customer: "Example KK", Product: "traffic66",
		License: license.TypeSubscription, Tier: license.TierStandard, MaxUsers: 25, Features: []string{license.FeatSyslog}}))
	if w := post(`{"enabled":true,"address":"no port","network":"udp","format":"json","min_sev":1}`); w.Code != 400 {
		t.Fatalf("an address without a port was accepted: %d", w.Code)
	}
	if w := post(on); w.Code != 200 {
		t.Fatalf("registered, switching on: %d %s", w.Code, w.Body)
	}
	if w := test(); w.Code != 200 {
		t.Fatalf("test: %d %s", w.Code, w.Body)
	}
	buf := make([]byte, 4096)
	udp.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := udp.ReadFrom(buf)
	if err != nil || !strings.Contains(string(buf[:n]), "test") {
		t.Fatalf("the collector got %q, %v", buf[:n], err)
	}

	// kept for the next start, and shown with how the feed is going
	saved, ok, err := syslogout.LoadConfig(s.syslogFile())
	if err != nil || !ok || saved.Format != "json" || saved.MinSev != 2 {
		t.Fatalf("saved %+v %v %v", saved, ok, err)
	}
	w := httptest.NewRecorder()
	s.getSyslog(w, httptest.NewRequest("GET", "/api/syslog", nil))
	var out struct {
		Registered bool             `json:"registered"`
		Stats      syslogout.Stats  `json:"stats"`
		Config     syslogout.Config `json:"config"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if !out.Registered || out.Stats.Sent != 1 || out.Config.Address != udp.LocalAddr().String() {
		t.Fatalf("status %s", w.Body)
	}
}
