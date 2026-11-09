package api

import (
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A new installation is empty for its first several minutes, and the pages
// said only "No data in this period." That reads as a fault, and the first
// thing somebody does about a fault is start changing things — ports,
// firewall rules, the device's configuration — while the collector was
// doing exactly what it should and was nearly finished doing it.
//
// Nothing here is new behaviour. It is the waiting the collector already
// does, named so a page can say which of them it is in.
const (
	// WaitNothing: no datagram has arrived from anything. This one really
	// does need the operator: the flows are not being sent, or not here.
	WaitNothing = "nothing"
	// WaitTemplates: NetFlow v9 and IPFIX describe their records in
	// templates sent every few minutes. Until one arrives its data sets
	// cannot be read, so they are held rather than dropped.
	WaitTemplates = "templates"
	// WaitSampling: the records are readable but the device has not said
	// what rate it samples at. They are held rather than counted as 1:1,
	// which would under-report by the sampling rate.
	WaitSampling = "sampling"
	// WaitSettling: the data is in, but the most recent minutes are not
	// complete yet, because exporters report a flow after it ends.
	WaitSettling = "settling"
)

// Waiting says what the collector is still waiting for, for a page that has
// nothing to show. Reason is empty when it is waiting for nothing and an
// empty page means what it says.
type Waiting struct {
	Reason string `json:"reason,omitempty"`
	// Records is how many are being held, for the reasons that hold them.
	Records int `json:"records,omitempty"`
	// Seconds is roughly how much longer it is worth waiting.
	Seconds int `json:"seconds,omitempty"`
	// Since is when the earliest exporter still held at this stage first
	// sent anything, and ElapsedSec how long ago that was, so a page can
	// say "waiting x minutes" rather than leave somebody guessing.
	// A pointer, because omitempty never omits a time.Time struct: in the
	// settling state it went out as "0001-01-01T00:00:00Z".
	Since      *time.Time `json:"since,omitempty"`
	ElapsedSec int        `json:"elapsed_sec,omitempty"`
}

// waitingOn asks the collector what it has and works out which stage it is
// in.
func (s *Server) waitingOn() Waiting {
	if s.Col == nil {
		return Waiting{}
	}
	return waitingFor(s.Col.Sources(), store.Settle(), time.Now())
}

// waitingFor is the decision itself, over the sources as they are. The
// order matters: it is the order the stages happen in, so the answer is the
// earliest one still unfinished, which is the one somebody would otherwise
// be puzzled by.
func waitingFor(srcs []collector.SourceInfo, settle time.Duration, now time.Time) Waiting {
	if len(srcs) == 0 {
		return Waiting{Reason: WaitNothing}
	}
	var held, noTemplate int
	var tmplSince, sampSince time.Time
	earliest := func(cur *time.Time, t time.Time) {
		if !t.IsZero() && (cur.IsZero() || t.Before(*cur)) {
			*cur = t
		}
	}
	for _, si := range srcs {
		if si.Templates == 0 && si.NoTemplate > 0 {
			noTemplate += int(si.NoTemplate)
			earliest(&tmplSince, si.FirstSeen)
		}
		if si.SamplingState == "waiting" {
			held += si.Pending
			earliest(&sampSince, si.FirstSeen)
		}
	}
	withSince := func(w Waiting, since time.Time) Waiting {
		if !since.IsZero() {
			t := since
			w.Since = &t
			if d := now.Sub(since); d > 0 {
				w.ElapsedSec = int(d.Seconds())
			}
		}
		return w
	}
	if noTemplate > 0 {
		return withSince(Waiting{Reason: WaitTemplates, Records: noTemplate}, tmplSince)
	}
	if held > 0 {
		return withSince(Waiting{Reason: WaitSampling, Records: held}, sampSince)
	}
	// Everything is being read and scaled; what is left is the settle
	// delay, which is why the last few minutes of a chart are marked.
	if settle > 0 {
		return Waiting{Reason: WaitSettling, Seconds: int(settle.Seconds())}
	}
	return Waiting{}
}
