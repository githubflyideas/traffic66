package api

import (
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
)

type srcList []collector.SourceInfo

// settleFor stands in for the installation's settle delay.
const settleFor = 2 * time.Minute

func TestWaitingNothingReceived(t *testing.T) {
	if w := waitingFor(nil, settleFor); w.Reason != WaitNothing {
		t.Errorf("reason %q, want %q", w.Reason, WaitNothing)
	}
}

// Templates come first: until one arrives the records cannot be read at
// all, so that is what somebody is actually waiting for.
func TestWaitingForTemplates(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 0, NoTemplate: 42}}, settleFor)
	if w.Reason != WaitTemplates {
		t.Errorf("reason %q, want %q", w.Reason, WaitTemplates)
	}
	if w.Records != 42 {
		t.Errorf("records %d, want 42", w.Records)
	}
}

// Once the templates are in, the sampling rate is the next thing held for.
func TestWaitingForSampling(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 3, SamplingState: "waiting", Pending: 7}}, settleFor)
	if w.Reason != WaitSampling || w.Records != 7 {
		t.Errorf("got %+v, want sampling with 7 records", w)
	}
}

// A source that has its templates and its rate is waiting only for the
// recent minutes to fill in.
func TestWaitingForSettling(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 3, SamplingState: "known"}}, settleFor)
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
	}, settleFor)
	if w.Reason != WaitTemplates {
		t.Errorf("reason %q, want %q", w.Reason, WaitTemplates)
	}
}

// A source with templates is not "waiting for templates" merely because it
// once lacked them: NoTemplate counts what was held, not what still is.
func TestTemplatesArrivedIsNotStillWaiting(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 2, NoTemplate: 100, SamplingState: "known"}}, settleFor)
	if w.Reason == WaitTemplates {
		t.Error("still reported as waiting for templates after they arrived")
	}
}

// Nothing to wait for, and no settle delay: an empty page then means what
// it says, and the caller falls back to "no data in this period".
func TestWaitingForNothingAtAll(t *testing.T) {
	w := waitingFor(srcList{{Exporter: "10.0.0.1", Templates: 2, SamplingState: "known"}}, 0)
	if w.Reason != "" {
		t.Errorf("reason %q, want none", w.Reason)
	}
}
