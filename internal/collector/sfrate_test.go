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
	if st.winMiss != 0 {
		t.Errorf("window holds %v missing samples from the old rate", st.winMiss)
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

// run sends n samples at rate, each with the given pool growth and gap in
// sequence numbers (1 = nothing lost), one a second, then one more after
// the 30-second window so the rate is measured; it returns the rate.
func run(st *sfState, rate, poolStep, seqStep uint32, n int, pool, seq *uint32, at time.Time) float64 {
	for i := 0; i < n; i++ {
		*pool += poolStep
		*seq += seqStep
		st.update(&sflow.FlowSample{Rate: rate, Pool: *pool, Seq: *seq}, at)
		at = at.Add(time.Second)
	}
	*pool += poolStep
	*seq += seqStep
	st.update(&sflow.FlowSample{Rate: rate, Pool: *pool, Seq: *seq}, at.Add(31*time.Second))
	return st.eff
}

// Whatever rate the agent declares (1 in 1024, 10240 or 65535), that is
// the rate used when nothing is lost, whatever its sample_pool says: an H3C
// port at 1:10240 whose pool grew a fifth more per sample was scaled by
// about 1:12200, a fifth more traffic than the interface carried.
func TestRateIsTheDeclaredOneWhateverThePool(t *testing.T) {
	for _, rate := range []uint32{1024, 10240, 65535} {
		st := &sfState{evalAt: time.Now()}
		var pool, seq uint32
		if got := run(st, rate, rate*6/5, 1, 40, &pool, &seq, time.Now()); got != float64(rate) {
			t.Errorf("1:%d, pool a fifth larger, nothing lost: rate %v, want %d", rate, got, rate)
		}
	}
}

// Samples that never arrived raise the rate by exactly what is missing:
// with every fifth sample lost, each received one stands for 5/4.
func TestRateCountsLostSamples(t *testing.T) {
	for _, rate := range []uint32{1024, 10240, 65535} {
		st := &sfState{evalAt: time.Now()}
		var pool, seq uint32
		seq = 1
		st.update(&sflow.FlowSample{Rate: rate, Pool: pool, Seq: seq}, time.Now())
		// alternate gaps of 1 and 2: 4 received for every 5 sent, on average
		now := time.Now()
		for i := 0; i < 80; i++ {
			step := uint32(1)
			if i%4 == 3 {
				step = 2
			}
			seq += step
			st.update(&sflow.FlowSample{Rate: rate, Pool: pool, Seq: seq}, now.Add(time.Duration(i)*time.Second))
		}
		want := float64(rate) * 5 / 4
		if got := st.eff; got < want*0.98 || got > want*1.02 {
			t.Errorf("1:%d with a fifth of the samples lost: rate %v, want about %v", rate, got, want)
		}
	}
}

// Changing the rate on the switch (1:1024 to 1:65535) is followed at once,
// and nothing measured against the old rate is carried over.
func TestRateFollowsAChangeOnTheSwitch(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	var pool, seq uint32
	if got := run(st, 1024, 1024, 1, 40, &pool, &seq, time.Now()); got != 1024 {
		t.Fatalf("before the change: %v", got)
	}
	seq++
	st.update(&sflow.FlowSample{Rate: 65535, Pool: pool, Seq: seq}, time.Now())
	if st.eff != 0 || st.rate != 65535 {
		t.Errorf("after the change: measured rate %v kept, declared %d", st.eff, st.rate)
	}
	if got := run(st, 65535, 65535, 1, 40, &pool, &seq, time.Now()); got != 65535 {
		t.Errorf("measured again after the change: %v, want 65535", got)
	}
}
