package collector

import (
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/decode/sflow"
)

// feed drives one source through n samples at the given configured rate,
// with pool advancing by that rate per sample, which is what an agent
// sampling at it would report.
func feed(st *sfState, rate uint32, n int, pool *uint32, seq *uint32, at time.Time) {
	for i := 0; i < n; i++ {
		*pool += rate
		*seq++
		st.update(&sflow.FlowSample{Rate: rate, Pool: *pool, Seq: *seq}, at)
		at = at.Add(time.Second)
	}
}

// After the measurement window has settled, the effective rate is the
// measured one; when the agent's configured rate then changes, the measured
// rate from the old setting must not be carried over.
func TestEffectiveRateDroppedWhenConfiguredRateChanges(t *testing.T) {
	now := time.Now()
	st := &sfState{evalAt: now}
	var pool, seq uint32
	// 40 samples at 1:1000, more than a 30-second window needs
	feed(st, 1000, 40, &pool, &seq, now)
	st.update(&sflow.FlowSample{Rate: 1000, Pool: pool + 1000, Seq: seq + 1}, now.Add(31*time.Second))
	if st.eff == 0 {
		t.Fatal("no effective rate was measured at 1:1000")
	}
	measured := st.eff
	if measured < 500 || measured > 2000 {
		t.Fatalf("measured %v, want about 1000", measured)
	}
	// the operator changes the agent to 1 in 100
	st.update(&sflow.FlowSample{Rate: 100, Pool: pool + 1100, Seq: seq + 2}, now.Add(32*time.Second))
	if st.eff != 0 {
		t.Errorf("effective rate is still %v after the configured rate changed to 100; "+
			"records would be scaled by the old rate (%.0fx too much)", st.eff, measured/100)
	}
	if st.rate != 100 {
		t.Errorf("configured rate %d, want 100", st.rate)
	}
}

// The window restarts on a change, so the next measurement is not a blend
// of the two settings.
func TestMeasurementWindowRestartsOnRateChange(t *testing.T) {
	now := time.Now()
	st := &sfState{evalAt: now}
	var pool, seq uint32
	feed(st, 1000, 20, &pool, &seq, now)
	if st.winRecv == 0 {
		t.Fatal("nothing accumulated")
	}
	st.update(&sflow.FlowSample{Rate: 100, Pool: pool + 100, Seq: seq + 1}, now.Add(time.Second))
	// only the sample that carried the new rate is in the window; the 20
	// measured against 1:1000 are gone
	if st.winRecv != 1 {
		t.Errorf("window holds %v samples, want only the one that carried the new rate", st.winRecv)
	}
	if st.winPool != 100 {
		t.Errorf("window holds %v pool, want only the new sample's 100", st.winPool)
	}
	// measuring again at the new rate gives the new rate, not a blend
	pool += 100
	seq++
	at := now.Add(2 * time.Second)
	feed(st, 100, 40, &pool, &seq, at)
	st.update(&sflow.FlowSample{Rate: 100, Pool: pool + 100, Seq: seq + 1}, at.Add(31*time.Second))
	if st.eff < 50 || st.eff > 200 {
		t.Errorf("measured %v after the change, want about 100", st.eff)
	}
}

// A rate that does not change must not reset anything: that would stop the
// effective rate from ever being measured.
func TestSteadyRateKeepsMeasuring(t *testing.T) {
	now := time.Now()
	st := &sfState{evalAt: now}
	var pool, seq uint32
	feed(st, 1000, 40, &pool, &seq, now)
	st.update(&sflow.FlowSample{Rate: 1000, Pool: pool + 1000, Seq: seq + 1}, now.Add(31*time.Second))
	if st.eff == 0 {
		t.Error("no effective rate measured although the rate never changed")
	}
}
