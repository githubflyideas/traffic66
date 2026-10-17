package syslogout

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

var when = time.Date(2026, 10, 14, 2, 59, 0, 0, time.UTC)

func sample() Event {
	e := Event{
		Phase: PhaseStart, ID: "41", Kind: "brute", Severity: 3,
		Src: "10.10.3.27", Dst: "10.20.0.15", Port: "3389", Proto: "TCP",
		First: when, Last: when.Add(15 * time.Minute), Hits: 22,
		Bytes: 3740, Packets: 90112, Sampling: "1:4096",
		Exporter: "192.0.2.1", IfIndex: 3, Country: "JP", ASN: "AS64500", Org: "Example Telecom",
		Evidence: "https://traffic66.example.com/#v=findings&finding=41",
		Extra:    map[string]string{"conns": "22"},
	}
	e.Text = Describe(e)
	return e
}

func lineFor(t *testing.T, f Format, e Event) string {
	t.Helper()
	cfg := Config{Enabled: true, Address: "collector:514", Format: f, Hostname: "sensor1"}
	cfg.defaults()
	return Message(cfg, e, when)
}

// Whatever the body, the envelope carries the identity of the event and
// the sampling rate, so a collector that understands only RFC 5424 still
// gets the two things it cannot work out for itself.
func TestEnvelope(t *testing.T) {
	for _, f := range Formats {
		line := lineFor(t, f, sample())
		if !strings.HasPrefix(line, "<108>1 2026-10-14T02:59:00.000Z sensor1 traffic66 ") {
			t.Fatalf("%s: envelope is %.60q", f, line)
		}
		for _, want := range []string{`traffic66@32473`, `id="41"`, `kind="brute"`, `phase="start"`, `sampling="1:4096"`} {
			if !strings.Contains(line, want) {
				t.Errorf("%s: missing %s", f, want)
			}
		}
		if strings.Count(line, "\n") != 0 {
			t.Errorf("%s: a message must be one line", f)
		}
	}
}

func TestCEF(t *testing.T) {
	line := lineFor(t, FormatCEF, sample())
	body := line[strings.Index(line, "CEF:"):]
	head := strings.SplitN(body, "|", 8)
	if len(head) != 8 {
		t.Fatalf("CEF header has %d fields: %q", len(head), body)
	}
	if head[0] != "CEF:0" || head[1] != "traffic66" || head[4] != "brute" {
		t.Fatalf("header %v", head[:5])
	}
	if head[6] != "8" {
		t.Fatalf("a high finding should be severity 8, got %s", head[6])
	}
	for _, want := range []string{
		"src=10.10.3.27", "dst=10.20.0.15", "dpt=3389", "proto=TCP",
		"out=3740", "cnt=22", "externalId=41", "act=start",
		"start=" + strconv.FormatInt(when.UnixMilli(), 10),
		"end=" + strconv.FormatInt(when.Add(15*time.Minute).UnixMilli(), 10),
		"cs1Label=sampling cs1=1:4096",
		"cs3Label=exporter cs3=192.0.2.1/3",
		"cs5Label=peerCountry cs5=JP",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in %s", want, body)
		}
	}
}

func TestLEEF(t *testing.T) {
	line := lineFor(t, FormatLEEF, sample())
	body := line[strings.Index(line, "LEEF:"):]
	if !strings.HasPrefix(body, "LEEF:2.0|traffic66|traffic66|") {
		t.Fatalf("header %.40q", body)
	}
	head := strings.SplitN(body, "|", 7)
	if head[4] != "brute" {
		t.Fatalf("event id %q", head[4])
	}
	if head[5] != "\t" {
		t.Fatalf("version 2.0 names its delimiter, got %q", head[5])
	}
	for _, want := range []string{"src=10.10.3.27", "dstPort=3389", "sev=8", "findingId=41", "sampling=1:4096"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestJSONIsECS(t *testing.T) {
	line := lineFor(t, FormatJSON, sample())
	body := line[strings.Index(line, "{"):]
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("%v in %s", err, body)
	}
	ev := m["event"].(map[string]any)
	if ev["kind"] != "alert" || ev["action"] != "brute" {
		t.Fatalf("event %v", ev)
	}
	if m["source"].(map[string]any)["ip"] != "10.10.3.27" {
		t.Fatal("source.ip")
	}
	dst := m["destination"].(map[string]any)
	if dst["port"].(float64) != 3389 {
		t.Fatalf("destination %v", dst)
	}
	if m["source"].(map[string]any)["bytes"].(float64) != 3740 {
		t.Fatalf("what the source sent belongs on the source: %v", m["source"])
	}
	if dst["geo"].(map[string]any)["country_iso_code"] != "JP" {
		t.Fatal("destination.geo")
	}
	if dst["as"].(map[string]any)["number"].(float64) != 64500 {
		t.Fatalf("destination.as %v", dst["as"])
	}
	own := m["traffic66"].(map[string]any)
	if own["sampling"] != "1:4096" || own["estimated"] != true {
		t.Fatalf("sampled counts must be marked as estimates: %v", own)
	}
	obs := m["observer"].(map[string]any)
	if obs["ip"] != "192.0.2.1" {
		t.Fatalf("observer %v", obs)
	}
}

// Someone reading the file should see the sampling warning in words, not
// have to know what 1:4096 means for the numbers next to it.
func TestTextSaysTheCountsAreEstimates(t *testing.T) {
	line := lineFor(t, FormatText, sample())
	if !strings.Contains(line, "counts are estimates") {
		t.Fatalf("got %s", line)
	}
}

// A value carrying the separator of its own format would split one event
// into two, or swallow the rest of it.
func TestSeparatorsAreEscaped(t *testing.T) {
	e := sample()
	e.Org = "Pipe | Equals = Back\\slash"
	e.Text = "a|b=c"
	e.Extra = map[string]string{"note": "tab\there"}

	cef := lineFor(t, FormatCEF, e)
	if strings.Contains(cef, "Pipe | Equals = ") {
		t.Errorf("CEF did not escape: %s", cef)
	}
	if n := unescapedPipes(cef[strings.Index(cef, "CEF:"):]); n != 7 {
		t.Errorf("CEF should have exactly 7 separating pipes, got %d in %s", n, cef)
	}
	leef := lineFor(t, FormatLEEF, e)
	fields := strings.Split(leef[strings.Index(leef, "LEEF:"):], "\t")
	for _, f := range fields[1:] {
		if f != "" && !strings.Contains(f, "=") {
			t.Errorf("LEEF field %q has no key, so a tab leaked through", f)
		}
	}
}

// Older collectors mangle UTF-8, and a Japanese host name should not be
// the reason a feed stops being readable.
// unescapedPipes counts the pipes that separate CEF header fields: one
// written in a value is escaped and does not.
func unescapedPipes(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '|' {
			n++
		}
	}
	return n
}

func TestASCIIOption(t *testing.T) {
	e := sample()
	e.Text = "株式会社サンプル"
	cfg := Config{Enabled: true, Address: "c:514", Format: FormatCEF, Hostname: "センサー", ASCII: true}
	cfg.defaults()
	line := Message(cfg, e, when)
	for _, r := range line {
		if r > 0x7e {
			t.Fatalf("%q is not ASCII: %s", r, line)
		}
	}
	if !strings.Contains(line, "\\u682a") {
		t.Fatalf("non-ASCII should be escaped, not dropped: %s", line)
	}
}

func TestSeverityAndPriority(t *testing.T) {
	for _, c := range []struct{ sev, cef, pri int }{{3, 8, 108}, {2, 5, 109}, {1, 3, 109}} {
		if got := severity(c.sev); got != c.cef {
			t.Errorf("severity(%d) = %d, want %d", c.sev, got, c.cef)
		}
		if got := priority(13, c.sev); got != c.pri {
			t.Errorf("priority(13, %d) = %d, want %d", c.sev, got, c.pri)
		}
	}
}

// The event type is what a dashboard groups by, so it must not change
// with the wording.
func TestKindIsTheStableKey(t *testing.T) {
	e := sample()
	e.Text = "完全に違う文章"
	line := lineFor(t, FormatCEF, e)
	if !strings.Contains(line, `kind="brute"`) || !strings.Contains(line, "|brute|") {
		t.Fatalf("the event type should survive any wording: %s", line)
	}
}

func TestFromFinding(t *testing.T) {
	f := store.Finding{
		ID: 41, Kind: "exfil", Sev: 3, Src: "10.10.3.27", Dst: "203.0.113.200",
		Port: "443/tcp", First: when, Last: when.Add(time.Hour), Hits: 4,
		Ev: map[string]any{"up": float64(11_900_000_000), "down": float64(372_000_000), "sampling": float64(1000), "cc": "RU", "org": "Demo Hosting", "conns": float64(7)},
	}
	e := FromFinding(f, PhaseStart, "https://traffic66.example.com/")
	if e.Port != "443" || e.Proto != "TCP" {
		t.Fatalf("port %q proto %q", e.Port, e.Proto)
	}
	if e.Sampling != "1:1000" {
		t.Fatalf("sampling %q", e.Sampling)
	}
	if e.Bytes != 11_900_000_000 || e.Down != 372_000_000 || e.Country != "RU" || e.Org != "Demo Hosting" {
		t.Fatalf("%+v", e)
	}
	if e.Extra["conns"] != "7" {
		t.Fatalf("extra %v", e.Extra)
	}
	if !strings.Contains(e.Evidence, "finding=41") {
		t.Fatalf("evidence %q", e.Evidence)
	}
	if !strings.Contains(e.Text, "11.1 GB") {
		t.Fatalf("text %q", e.Text)
	}
}

// Sampling of 1 is not sampling, and saying "1:1" would make a reader
// think the counts were weighted.
func TestUnsampledSaysNothing(t *testing.T) {
	f := store.Finding{ID: 1, Kind: "scan", Sev: 1, Ev: map[string]any{"sampling": float64(1)}}
	if e := FromFinding(f, PhaseStart, ""); e.Sampling != "" {
		t.Fatalf("sampling %q", e.Sampling)
	}
}

func TestConfigCheck(t *testing.T) {
	ok := Config{Enabled: true, Address: "siem.example.com:514", Network: "tls", Format: FormatLEEF, MinSev: 2}
	if err := ok.Check(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		what string
		cfg  Config
	}{
		{"no address", Config{Enabled: true}},
		{"no port", Config{Enabled: true, Address: "siem.example.com"}},
		{"unknown transport", Config{Enabled: true, Address: "a:514", Network: "carrier-pigeon"}},
		{"unknown format", Config{Enabled: true, Address: "a:514", Format: "xml"}},
		{"severity out of range", Config{Enabled: true, Address: "a:514", MinSev: 9}},
	} {
		if err := c.cfg.Check(); err == nil {
			t.Errorf("%s was accepted", c.what)
		}
	}
	// Switched off, nothing else matters.
	if err := (Config{Address: "nonsense"}).Check(); err != nil {
		t.Fatalf("a feed that is off should not complain: %v", err)
	}
}

// A flood of findings must not become a flood of syslog, or the feed gets
// turned off and nobody hears about the next one either.
func TestCeiling(t *testing.T) {
	s := New(Config{Enabled: true, Address: "203.0.113.9:514", PerMin: 3})
	now := time.Now()
	for i := 0; i < 5; i++ {
		s.mu.Lock()
		got := s.allow(now)
		s.mu.Unlock()
		if want := i < 3; got != want {
			t.Fatalf("message %d: allowed=%v, want %v", i+1, got, want)
		}
	}
	s.mu.Lock()
	next := s.allow(now.Add(time.Minute))
	s.mu.Unlock()
	if !next {
		t.Fatal("the next minute starts again")
	}
}

// It has to actually reach a collector.
func TestSendsOverUDP(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	s := New(Config{Enabled: true, Address: pc.LocalAddr().String(), Format: FormatCEF})
	defer s.Close()
	s.Send(sample())

	buf := make([]byte, 4096)
	pc.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(buf[:n]); !strings.Contains(got, "CEF:0|traffic66") || !strings.Contains(got, "src=10.10.3.27") {
		t.Fatalf("got %s", got)
	}
}

func TestBelowSeverityIsNotSent(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	s := New(Config{Enabled: true, Address: pc.LocalAddr().String(), MinSev: 3})
	defer s.Close()
	low := sample()
	low.Severity = 1
	s.Send(low)
	pc.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := pc.ReadFrom(make([]byte, 512)); err == nil {
		t.Fatal("a finding below the chosen severity was sent anyway")
	}
}

// A stream needs to know where one message ends.
func TestTCPIsFramed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		b := make([]byte, 4096)
		n, _ := c.Read(b)
		got <- string(b[:n])
	}()
	s := New(Config{Enabled: true, Address: ln.Addr().String(), Network: "tcp", Format: FormatText})
	defer s.Close()
	s.Send(sample())
	select {
	case line := <-got:
		size, rest, ok := strings.Cut(line, " ")
		if !ok {
			t.Fatalf("no octet count: %q", line)
		}
		if len(rest) == 0 || size == "" {
			t.Fatalf("framing %q", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("nothing arrived")
	}
}
