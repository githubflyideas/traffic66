package syslogout

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Formats of the message body. The envelope is RFC 5424 either way; this
// is what goes in it, and it is a choice because which one parses by
// itself depends entirely on what the customer already runs.
type Format string

const (
	// FormatCEF is ArcSight's Common Event Format, and the one with the
	// widest out-of-the-box support: ArcSight, QRadar, Splunk, Graylog,
	// Elastic and Wazuh all read it without anyone writing a parser.
	FormatCEF Format = "cef"
	// FormatLEEF is IBM's, read natively by QRadar.
	FormatLEEF Format = "leef"
	// FormatJSON uses Elastic Common Schema field names, so an Elastic or
	// Splunk pipeline needs no mapping written for it.
	FormatJSON Format = "json"
	// FormatText is for a person reading the file.
	FormatText Format = "text"
)

// Formats lists what can be chosen, for the settings page and the flag.
var Formats = []Format{FormatCEF, FormatLEEF, FormatJSON, FormatText}

// Valid reports whether f is a format this knows.
func (f Format) Valid() bool {
	for _, k := range Formats {
		if f == k {
			return true
		}
	}
	return false
}

// Vendor, Product and the enterprise number identify traffic66 in the
// formats that ask. 32473 is the number IANA reserves for examples and
// private use, which is honest: we have not registered one.
const (
	Vendor     = "traffic66"
	Product    = "traffic66"
	Enterprise = "32473"
)

// severity turns our three levels into the 0–10 the formats expect.
func severity(sev int) int {
	switch sev {
	case 3:
		return 8
	case 2:
		return 5
	case 1:
		return 3
	}
	return 1
}

// priority is the syslog PRI: facility times eight plus the level. High
// findings go out as warnings, the rest as notices, because a finding is
// something to look at, not something that is on fire.
func priority(facility, sev int) int {
	level := 5 // notice
	if sev >= 3 {
		level = 4 // warning
	}
	return facility*8 + level
}

// Message builds the whole line, envelope and body.
func Message(cfg Config, e Event, now time.Time) string {
	host := cfg.Hostname
	if host == "" {
		host, _ = os.Hostname()
	}
	if host == "" {
		host = "-"
	}
	body := body(cfg.Format, e)
	if cfg.ASCII {
		body, host = ascii(body), ascii(host)
	}
	// RFC 5424: <PRI>1 TIMESTAMP HOSTNAME APP PROCID MSGID [SD] MSG
	sd := structured(e)
	return fmt.Sprintf("<%d>1 %s %s %s %d %s %s %s",
		priority(cfg.Facility, e.Severity),
		now.UTC().Format("2006-01-02T15:04:05.000Z"),
		nilable(host), Product, os.Getpid(), nilable(e.Kind), sd, body)
}

// structured is the RFC 5424 structured data: the few fields worth having
// even when the body is plain text, so a collector that understands only
// the envelope still gets the identity of the event and the sampling rate.
func structured(e Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, `[traffic66@%s id="%s" kind="%s" phase="%s"`, Enterprise, sdEsc(e.ID), sdEsc(e.Kind), sdEsc(string(e.Phase)))
	if e.Sampling != "" {
		fmt.Fprintf(&b, ` sampling="%s"`, sdEsc(e.Sampling))
	}
	b.WriteString("]")
	return b.String()
}

func body(f Format, e Event) string {
	switch f {
	case FormatLEEF:
		return leef(e)
	case FormatJSON:
		return asJSON(e)
	case FormatText:
		return text(e)
	}
	return cef(e)
}

// cef writes ArcSight's Common Event Format. The header's seven fields are
// pipe separated; everything else is key=value.
func cef(e Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CEF:0|%s|%s|%s|%s|%s|%d|", Vendor, Product, version(), cefHdr(e.Kind), cefHdr(e.Text), severity(e.Severity))
	kv := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%s=%s ", k, cefVal(v))
		}
	}
	kv("src", e.Src)
	kv("dst", e.Dst)
	kv("dpt", e.Port)
	kv("proto", e.Proto)
	if e.Bytes > 0 {
		kv("out", strconv.FormatUint(e.Bytes, 10))
	}
	if e.Down > 0 {
		kv("in", strconv.FormatUint(e.Down, 10))
	}
	if e.Packets > 0 {
		kv("cn1Label", "packets")
		kv("cn1", strconv.FormatUint(e.Packets, 10))
	}
	if e.Hits > 0 {
		kv("cnt", strconv.Itoa(e.Hits))
	}
	if !e.First.IsZero() {
		kv("start", strconv.FormatInt(e.First.UnixMilli(), 10))
	}
	if !e.Last.IsZero() {
		kv("end", strconv.FormatInt(e.Last.UnixMilli(), 10))
	}
	kv("externalId", e.ID)
	kv("act", string(e.Phase))
	label(&b, "cs1", "sampling", e.Sampling)
	label(&b, "cs2", "findingUrl", e.Evidence)
	label(&b, "cs3", "exporter", exporter(e))
	label(&b, "cs4", "peerOrg", strings.TrimSpace(e.Org+" "+e.ASN))
	if e.Country != "" {
		kv("cs5Label", "peerCountry")
		kv("cs5", e.Country)
	}
	for _, k := range e.extraKeys() {
		kv("traffic66"+title(k), e.Extra[k])
	}
	return strings.TrimRight(b.String(), " ")
}

func label(b *strings.Builder, key, name, v string) {
	if v == "" {
		return
	}
	fmt.Fprintf(b, "%sLabel=%s %s=%s ", key, name, key, cefVal(v))
}

// leef writes IBM's format, which QRadar reads natively. Version 2.0 names
// its own delimiter; a tab keeps values readable and never appears in one.
func leef(e Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "LEEF:2.0|%s|%s|%s|%s|\t|", Vendor, Product, version(), leefEsc(e.Kind))
	kv := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%s=%s\t", k, leefEsc(v))
		}
	}
	kv("cat", e.Kind)
	kv("sev", strconv.Itoa(severity(e.Severity)))
	kv("src", e.Src)
	kv("dst", e.Dst)
	kv("dstPort", e.Port)
	kv("proto", e.Proto)
	if e.Bytes > 0 {
		kv("srcBytes", strconv.FormatUint(e.Bytes, 10))
	}
	if e.Down > 0 {
		kv("dstBytes", strconv.FormatUint(e.Down, 10))
	}
	if e.Packets > 0 {
		kv("dstPackets", strconv.FormatUint(e.Packets, 10))
	}
	if !e.First.IsZero() {
		kv("devTimeFormat", "MMM dd yyyy HH:mm:ss")
		kv("devTime", e.First.UTC().Format("Jan 02 2006 15:04:05"))
	}
	kv("identSrc", e.Src)
	kv("findingId", e.ID)
	kv("phase", string(e.Phase))
	kv("hits", nonZero(e.Hits))
	kv("sampling", e.Sampling)
	kv("exporter", exporter(e))
	kv("peerCountry", e.Country)
	kv("peerOrg", strings.TrimSpace(e.Org+" "+e.ASN))
	kv("findingUrl", e.Evidence)
	kv("msg", e.Text)
	for _, k := range e.extraKeys() {
		kv(k, e.Extra[k])
	}
	return strings.TrimRight(b.String(), "\t")
}

// asJSON writes Elastic Common Schema names, so an Elastic or Splunk
// pipeline needs no mapping written for it and the SIEM views recognise
// the event on their own.
func asJSON(e Event) string {
	m := map[string]any{
		"@timestamp": e.Last.UTC().Format(time.RFC3339Nano),
		"message":    e.Text,
		"event": map[string]any{
			"kind":     "alert",
			"category": []string{"network", "intrusion_detection"},
			"action":   e.Kind,
			"id":       e.ID,
			"severity": severity(e.Severity),
			"count":    e.Hits,
			"start":    rfc(e.First),
			"end":      rfc(e.Last),
			"module":   Product,
			"provider": Vendor,
		},
		"rule":     map[string]any{"name": e.Kind, "id": e.Kind},
		"observer": observer(e),
	}
	if e.Evidence != "" {
		m["event"].(map[string]any)["reference"] = e.Evidence
	}
	if e.Src != "" {
		src := map[string]any{"ip": e.Src}
		if e.Bytes > 0 {
			src["bytes"] = e.Bytes
		}
		if e.Packets > 0 {
			src["packets"] = e.Packets
		}
		m["source"] = src
	}
	dst := map[string]any{}
	if e.Dst != "" {
		dst["ip"] = e.Dst
	}
	if p, err := strconv.Atoi(e.Port); err == nil {
		dst["port"] = p
	}
	if e.Down > 0 {
		dst["bytes"] = e.Down
	}
	if e.Country != "" {
		dst["geo"] = map[string]any{"country_iso_code": e.Country}
	}
	if e.ASN != "" || e.Org != "" {
		as := map[string]any{}
		if n, err := strconv.Atoi(strings.TrimPrefix(e.ASN, "AS")); err == nil {
			as["number"] = n
		}
		if e.Org != "" {
			as["organization"] = map[string]any{"name": e.Org}
		}
		dst["as"] = as
	}
	if len(dst) > 0 {
		m["destination"] = dst
	}
	if e.Proto != "" {
		m["network"] = map[string]any{"transport": strings.ToLower(e.Proto), "bytes": e.Bytes + e.Down}
	}
	var related []string
	for _, ip := range []string{e.Src, e.Dst} {
		if ip != "" {
			related = append(related, ip)
		}
	}
	if len(related) > 0 {
		m["related"] = map[string]any{"ip": related}
	}
	// What a collector cannot work out for itself, under our own name so
	// it never collides with a schema field.
	own := map[string]any{"phase": string(e.Phase)}
	if e.Sampling != "" {
		own["sampling"] = e.Sampling
		own["estimated"] = true // the byte and packet counts are weighted
	}
	for k, v := range e.Extra {
		own[k] = v
	}
	m["traffic66"] = own
	b, err := json.Marshal(m)
	if err != nil {
		return `{"message":"traffic66: could not encode this finding"}`
	}
	return string(b)
}

func observer(e Event) map[string]any {
	o := map[string]any{"vendor": Vendor, "product": Product, "type": "flow"}
	if e.Exporter != "" {
		o["ip"] = e.Exporter
	}
	if e.IfIndex != 0 {
		o["ingress"] = map[string]any{"interface": map[string]any{"id": strconv.FormatUint(uint64(e.IfIndex), 10)}}
	}
	return o
}

// text is for a person: everything that matters on one line, no escaping
// games, no schema.
func text(e Event) string {
	var parts []string
	add := func(f string, a ...any) { parts = append(parts, fmt.Sprintf(f, a...)) }
	add("%s", e.Text)
	add("kind=%s", e.Kind)
	add("severity=%d", e.Severity)
	add("id=%s", e.ID)
	add("phase=%s", e.Phase)
	if e.Sampling != "" {
		add("sampling=%s (counts are estimates)", e.Sampling)
	}
	if ex := exporter(e); ex != "" {
		add("exporter=%s", ex)
	}
	if e.Evidence != "" {
		add("%s", e.Evidence)
	}
	return strings.Join(parts, " · ")
}

func exporter(e Event) string {
	switch {
	case e.Exporter != "" && e.IfIndex != 0:
		return fmt.Sprintf("%s/%d", e.Exporter, e.IfIndex)
	case e.Exporter != "":
		return e.Exporter
	}
	return ""
}

func rfc(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func nonZero(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func nilable(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// Version is the traffic66 version put in the message headers; main sets
// it at startup.
var Version = "0.0.0"

func version() string { return cefHdr(Version) }

// cefHdr escapes a CEF header field, where a pipe would end it.
func cefHdr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", `\|`)
	return oneLine(s)
}

// cefVal escapes a CEF extension value. The specification asks only for
// the backslash and the equals sign, but parsers differ on a pipe or a tab
// inside a value — some split the header on every pipe they see — so both
// are escaped too. Nothing is lost by being stricter than the format
// requires, and a field that swallows the rest of an event is the kind of
// bug that is only ever found in production.
func cefVal(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "=", `\=`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return oneLine(s)
}

// leefEsc escapes the tab that separates LEEF fields.
func leefEsc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return oneLine(s)
}

// sdEsc escapes an RFC 5424 structured data value.
func sdEsc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "]", `\]`)
	return oneLine(s)
}

// oneLine keeps a message to one line: a newline in the middle would look
// like the start of another event to every collector there is.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

// ascii replaces anything outside ASCII. Plenty of older collectors
// mangle UTF-8, and a Japanese host name should not be the reason a feed
// stops being readable.
func ascii(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			fmt.Fprintf(&b, "\\u%04x", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
