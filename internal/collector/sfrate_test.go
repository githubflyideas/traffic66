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

// Each sample counts for the packets the pool advanced by since the newest
// sample before it: samples lost on the way are made up for at once, an
// older sample arriving late counts for nothing, a restart starts over, and
// a pool that does not count packets falls back to the configured rate.
func TestSampleWeights(t *testing.T) {
	now := time.Now()
	for name, c := range map[string]struct {
		rate uint32
		in   []wIn
	}{
		// the first sample and the ten that check the pool count at the rate
		"a loss": {1000, append(steady(1000, 1000, 11, 0),
			wIn{12500, 12, 1500}, wIn{15500, 14, 3000}, // sample 13 was lost
		)},
		"late and duplicate": {1000, append(steady(1000, 1000, 11, 0),
			wIn{13000, 13, 2000}, wIn{12000, 12, 0}, wIn{12000, 12, 0}, wIn{14000, 14, 1000},
		)},
		"agent restart": {1000, append(steady(1000, 1000, 11, 50000),
			wIn{900, 1, 1000}, wIn{1900, 2, 1000}, wIn{2900, 3, 1000},
		)},
		"pool left at 0":        {1000, append(steady(1000, 0, 11, 0), wIn{0, 12, 1000}, wIn{0, 13, 1000})},
		"pool counting samples": {1000, append(steady(1000, 1, 11, 0), wIn{12, 12, 1000}, wIn{14, 14, 1000})},
		"one in one":            {1, append(steady(1, 1, 11, 0), wIn{19, 19, 8})},
		// thousands of samples lost in a row: the sequence number moved as far
		"one in one, a long run lost": {1, append(steady(1, 1, 11, 0), wIn{5011, 5011, 5000}, wIn{5012, 5012, 1})},
		"a pool that jumps alone":     {1000, append(steady(1000, 1000, 11, 0), wIn{11000 + 5e6, 12, 1000}, wIn{11000 + 5e6 + 1000, 13, 1000})},
	} {
		st := &sfState{evalAt: now}
		for i, x := range c.in {
			if got := st.update(&sflow.FlowSample{Rate: c.rate, Pool: x.pool, Seq: x.seq}, now); got != x.want {
				t.Errorf("%s: sample %d (pool %d): weight %v, want %v", name, i, x.pool, got, x.want)
			}
		}
	}
}

type wIn struct {
	pool, seq uint32
	want      float64
}

// steady is n samples whose pool advances by step each and whose sequence
// numbers start after seq0, counted at the configured rate while the pool
// is checked.
func steady(rate float64, step uint32, n int, seq0 uint32) []wIn {
	out := make([]wIn, n)
	for i := range out {
		out[i] = wIn{step * uint32(i+1), seq0 + uint32(i+1), rate}
	}
	return out
}
