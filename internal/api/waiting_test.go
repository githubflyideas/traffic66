package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
)

type srcList []collector.SourceInfo

// settleFor stands in for the installation's settle delay.
const settleFor = 2 * time.Minute

func TestWaitingNothingReceived(t *testing.T) {
	if w := waitingFor(nil, settleFor, time.Now()); w.Reason != WaitNothing {
		t.Errorf("reason %q, want %q", w.Reason, WaitNothing)
	}
}

// Templates come first: until one arrives the records cannot be read at
// all, so that is what somebody is actually waiting for.
func TestWaitingForTemplates(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 0, NoTemplate: 42}}, settleFor, time.Now())
	if w.Reason != WaitTemplates {
		t.Errorf("reason %q, want %q", w.Reason, WaitTemplates)
	}
	if w.Records != 42 {
		t.Errorf("records %d, want 42", w.Records)
	}
}

// Once the templates are in, the sampling rate is the next thing held for.
func TestWaitingForSampling(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 3, SamplingState: "waiting", Pending: 7}}, settleFor, time.Now())
	if w.Reason != WaitSampling || w.Records != 7 {
		t.Errorf("got %+v, want sampling with 7 records", w)
	}
}

// A source that has its templates and its rate is waiting only for the
// recent minutes to fill in.
func TestWaitingForSettling(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 3, SamplingState: "known"}}, settleFor, time.Now())
	if w.Reason != WaitSettling {
		t.Errorf("reason %q, want %q", w.Reason, WaitSettling)
	}
	if w.Seconds != int(settleFor.Seconds()) {
		t.Errorf("seconds %d, want %d", w.Seconds, int(settleFor.Seconds()))
	}
}

// The earliest unfinished stage wins: one source short of templates while
// another is short of a rate is a templates answer, because that is the one
// that has to finish first.
func TestWaitingReportsTheEarliestStage(t *testing.T) {
	w := waitingFor(srcList{
		{Exporter: "10.0.0.1", Templates: 3, SamplingState: "waiting", Pending: 5},
		{Exporter: "10.0.0.2", Templates: 0, NoTemplate: 9},
	}, settleFor, time.Now())
	if w.Reason != WaitTemplates {
		t.Errorf("reason %q, want %q", w.Reason, WaitTemplates)
	}
}

// A source with templates is not "waiting for templates" merely because it
// once lacked them: NoTemplate counts what was held, not what still is.
func TestTemplatesArrivedIsNotStillWaiting(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 2, NoTemplate: 100, SamplingState: "known"}}, settleFor, time.Now())
	if w.Reason == WaitTemplates {
		t.Error("still reported as waiting for templates after they arrived")
	}
}

// Nothing to wait for, and no settle delay: an empty page then means what
// it says, and the caller falls back to "no data in this period".
func TestWaitingForNothingAtAll(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 2, SamplingState: "known"}}, 0, time.Now())
	if w.Reason != "" {
		t.Errorf("reason %q, want none", w.Reason)
	}
}

// The page says how long it has been waiting, from the first datagram of
// the earliest exporter still held at that stage.
func TestWaitingReportsElapsed(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	w := waitingFor(srcList{
		{Exporter: "10.0.0.1", NoTemplate: 4, FirstSeen: now.Add(-3 * time.Minute)},
		{Exporter: "10.0.0.2", NoTemplate: 5, FirstSeen: now.Add(-7 * time.Minute)},
		{Exporter: "10.0.0.3", Templates: 2, SamplingState: "waiting", Pending: 1, FirstSeen: now.Add(-20 * time.Minute)},
	}, settleFor, now)
	if w.Reason != WaitTemplates || w.ElapsedSec != 420 || !w.Since.Equal(now.Add(-7*time.Minute)) {
		t.Errorf("got %+v, want templates, 420 s since -7m", w)
	}
	w = waitingFor(srcList{{Exporter: "10.0.0.3", Templates: 2, SamplingState: "waiting", Pending: 1, FirstSeen: now.Add(-90 * time.Second)}}, settleFor, now)
	if w.Reason != WaitSampling || w.ElapsedSec != 90 {
		t.Errorf("got %+v, want sampling, 90 s", w)
	}
	// settling and unknown start times carry no elapsed time
	if w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 3, SamplingState: "known", FirstSeen: now}}, settleFor, now); w.ElapsedSec != 0 || !w.Since.IsZero() {
		t.Errorf("settling got %+v", w)
	}
}

// The field is in the JSON the page reads.
func TestWaitingJSONFields(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	w := waitingFor(srcList{{Exporter: "10.0.0.1", NoTemplate: 1, FirstSeen: now.Add(-time.Minute)}}, settleFor, now)
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"elapsed_sec":60`) || !strings.Contains(string(b), `"since":"2026-10-16T09:59:00Z"`) {
		t.Errorf("json %s", b)
	}
}

func TestWaitStep(t *testing.T) {
	for _, c := range []struct {
		tm   int
		nt   uint64
		st   string
		want string
	}{{0, 3, "waiting", "templates"}, {2, 3, "waiting", "sampling"}, {2, 0, "known", ""}, {0, 0, "", ""}} {
		if g := collector.WaitStep(c.tm, c.nt, c.st); g != c.want {
			t.Errorf("WaitStep(%d,%d,%q) = %q, want %q", c.tm, c.nt, c.st, g, c.want)
		}
	}
}
