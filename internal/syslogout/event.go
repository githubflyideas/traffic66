// Package syslogout sends findings to a syslog collector.
//
// What makes this worth having is not that it can send a line of text: it
// is that the line can be parsed, counted and summed by whatever the
// customer already runs. Three things decide that, and they are the
// reason this package exists rather than a fmt.Fprintf somewhere:
//
//   - The event type is a stable word (scan, brute, exfil …) that never
//     changes with the wording, the translation or the version, so a
//     dashboard grouped by it does not break when a sentence is improved.
//   - A finding is one event with one id, sent when it starts and when it
//     ends, not once per detection run. Otherwise one night of password
//     guessing is counted as forty incidents and somebody turns the feed
//     off.
//   - The sampling rate travels with every event. traffic66 detects on
//     sampled data, so the byte and packet counts are estimates; a
//     collector that sums them without knowing that produces a number
//     nobody can defend.
package syslogout

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// Phase says what happened to a finding, which is what decides whether
// anything is sent at all.
type Phase string

const (
	PhaseStart Phase = "start" // first seen
	PhaseEnd   Phase = "end"   // closed, by a person or by time
	PhaseTest  Phase = "test"  // the "send test" button
)

// Event is one thing to report, already flattened out of a finding so the
// formats do not each have to understand the store.
type Event struct {
	Phase    Phase
	ID       string // stable for the life of one finding
	Kind     string // scan, portscan, brute, lateral, exfil, flood, threat
	Severity int    // 1 low, 2 medium, 3 high
	Text     string // one line a person can read

	Src, Dst string
	Port     string
	Proto    string

	First, Last time.Time
	Hits        int // detection runs that saw it

	Bytes, Packets uint64 // what the source sent; estimates when Sampling is set
	Down           uint64 // what came back, where the rule counted it
	Sampling       string // "1:4096", empty when the data was not sampled
	Exporter       string
	IfIndex        uint32
	Country        string
	ASN            string
	Org            string
	Evidence       string // link to the finding, empty when there is none

	Extra map[string]string // whatever else the rule recorded
}

// FromFinding flattens a finding. base is the address the web UI is
// reachable at, used to build the evidence link; empty leaves it out.
func FromFinding(f store.Finding, phase Phase, base string) Event {
	e := Event{
		Phase: phase, ID: strconv.FormatInt(f.ID, 10), Kind: f.Kind,
		Severity: f.Sev, Src: f.Src, Dst: f.Dst, Port: f.Port,
		First: f.First, Last: f.Last, Hits: f.Hits,
		Extra: map[string]string{},
	}
	if host, proto, ok := strings.Cut(f.Port, "/"); ok {
		e.Port, e.Proto = host, strings.ToUpper(proto)
	}
	if base != "" && f.ID != 0 {
		e.Evidence = strings.TrimRight(base, "/") + "/#v=findings&finding=" + e.ID
	}
	for k, v := range f.Ev {
		switch k {
		// The rules record what they actually counted: "up" is what the
		// source sent, which is the number that matters for an upload to
		// somewhere new, and "down" is what came back.
		case "up":
			e.Bytes = asUint(v)
		case "down":
			e.Down = asUint(v)
		case "pkts":
			e.Packets = asUint(v)
		case "sampling":
			if n := asUint(v); n > 1 {
				e.Sampling = "1:" + strconv.FormatUint(n, 10)
			}
		case "cc":
			e.Country = asString(v)
		case "org":
			e.Org = asString(v)
		case "asn":
			e.ASN = asString(v)
		case "examples":
			// a list of sample addresses; useful to a person, noise in a
			// field a collector has to index
		default:
			if s := asString(v); s != "" {
				e.Extra[k] = s
			}
		}
	}
	e.Text = Describe(e)
	return e
}

// Describe writes the one line a person reads, in English: a collector is
// indexed and searched in one language, and the translated wording lives
// in the web UI where a person is actually reading it.
func Describe(e Event) string {
	who := e.Src
	if who == "" {
		who = "an unknown source"
	}
	target := e.Dst
	if target != "" && e.Port != "" {
		target += ":" + e.Port
	}
	switch e.Kind {
	case "scan":
		return fmt.Sprintf("%s scanned the network on port %s", who, e.Port)
	case "portscan":
		return fmt.Sprintf("%s probed many ports of %s", who, e.Dst)
	case "brute":
		return fmt.Sprintf("%s tried to log in to %s again and again", who, target)
	case "lateral":
		return fmt.Sprintf("%s reached %s, a service never offered there before", who, target)
	case "exfil":
		return fmt.Sprintf("%s uploaded %s to %s, a destination it had not used before", who, bytes(e.Bytes), e.Dst)
	case "flood":
		return fmt.Sprintf("%s was flooded with packets", e.Dst)
	case "threat":
		return fmt.Sprintf("%s communicated with %s, which is on a threat list", who, e.Dst)
	case "test":
		return "test message from traffic66"
	}
	return fmt.Sprintf("%s: %s to %s", e.Kind, who, target)
}

func bytes(n uint64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f kB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func asUint(v any) uint64 {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return uint64(n)
		}
	case int:
		if n > 0 {
			return uint64(n)
		}
	case int64:
		if n > 0 {
			return uint64(n)
		}
	case uint64:
		return n
	case json.Number:
		if f, err := n.Float64(); err == nil && f > 0 {
			return uint64(f)
		}
	}
	return 0
}

func asString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case int:
		return strconv.Itoa(s)
	case int64:
		return strconv.FormatInt(s, 10)
	case bool:
		return strconv.FormatBool(s)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

// extraKeys lists the extra fields in a fixed order, so two messages about
// the same thing look the same and a collector's field list stays stable.
func (e Event) extraKeys() []string {
	out := make([]string, 0, len(e.Extra))
	for k := range e.Extra {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
