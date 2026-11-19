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

// When the configured rate changes, the measured compensation carries over
// to the new rate: nothing lost, so 1:1000 becomes exactly 1:100, not 1:1000
// (ten times too much) and not a pause in measuring.
func TestEffectiveRateFollowsAConfiguredRateChange(t *testing.T) {
	now := time.Now()
	st := &sfState{evalAt: now}
	var pool, seq uint32
	feed(st, 1000, 40, &pool, &seq, now)
	st.update(&sflow.FlowSample{Rate: 1000, Pool: pool + 1000, Seq: seq + 1}, now.Add(31*time.Second))
	if st.eff != 1000 {
		t.Fatalf("measured %v at 1:1000, want 1000", st.eff)
	}
	st.update(&sflow.FlowSample{Rate: 100, Pool: pool + 1100, Seq: seq + 2}, now.Add(32*time.Second))
	if st.eff != 100 || st.rate != 100 {
		t.Errorf("after the change to 1:100: rate %v (declared %d), want 100", st.eff, st.rate)
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
// with the share of lost samples kept: a fifth lost before the change is a
// fifth lost after it, not a pause of 30 seconds counted too low.
func TestRateChangeKeepsTheLossShare(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	var pool, seq uint32
	now := time.Now()
	send := func(rate uint32, n int) {
		for i := 0; i < n; i++ {
			pool += rate
			seq++
			if seq%5 == 0 {
				continue // lost on the way
			}
			st.update(&sflow.FlowSample{Rate: rate, Pool: pool, Seq: seq}, now)
			now = now.Add(time.Second / 2)
		}
	}
	send(1024, 100)
	before := st.eff
	if before < 1280*0.99 || before > 1280*1.01 {
		t.Fatalf("before the change: %v, want about 1280", before)
	}
	send(65535, 1)
	if want := before / 1024 * 65535; st.eff != want {
		t.Errorf("right after the change: %v, want %v", st.eff, want)
	}
}

// A sample that arrives after a later one, or twice, is neither a loss nor
// more traffic.
func TestReorderedAndRepeatedSamples(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	now := time.Now()
	var counted int
	order := []uint32{}
	for base := uint32(1); base < 400; base += 10 {
		// 1..10 with 3 and 4 swapped, and 6 delivered again after 7
		order = append(order, base, base+1, base+3, base+2, base+4, base+5, base+6, base+5, base+7, base+8, base+9)
	}
	for _, q := range order {
		if st.update(&sflow.FlowSample{Rate: 100, Pool: q * 100, Seq: q}, now) {
			counted++
		}
		now = now.Add(time.Second / 4)
	}
	if counted != 400 || st.lost != 0 || st.eff != 100 {
		t.Errorf("counted %d of 400, lost %d, rate %v; want 400, 0, 100", counted, st.lost, st.eff)
	}
}

// An agent numbering the samples of all its interfaces in one sequence:
// each interface sees two of three numbers missing, while its pool grows by
// the declared rate per sample. Nothing is lost, and nothing is tripled.
func TestSequenceSharedByInterfaces(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	now := time.Now()
	for i := uint32(1); i <= 200; i++ {
		st.update(&sflow.FlowSample{Rate: 1024, Pool: i * 1024, Seq: i * 3}, now)
		now = now.Add(time.Second / 4)
	}
	if st.eff != 1024 {
		t.Errorf("rate %v, want 1024", st.eff)
	}
	if st.lost > 0 {
		t.Errorf("%d samples reported lost", st.lost)
	}
}

// Four of five samples lost, with the pool agreeing: each received one
// stands for five, rather than compensation being switched off.
func TestHeavyLoss(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	now := time.Now()
	for i := uint32(1); i <= 1000; i++ {
		if i%5 == 0 {
			st.update(&sflow.FlowSample{Rate: 512, Pool: i * 512, Seq: i}, now)
		}
		now = now.Add(time.Second / 8)
	}
	if st.eff != 512*5 {
		t.Errorf("rate %v, want %d", st.eff, 512*5)
	}
}

// A restarted agent (numbers, drops and pool from zero) and an outage of
// minutes are not a loss rate for the samples that follow.
func TestRestartAndOutage(t *testing.T) {
	st := &sfState{evalAt: time.Now()}
	now := time.Now()
	seq := uint32(900000)
	step := func(n int) {
		for i := 0; i < n; i++ {
			seq++
			st.update(&sflow.FlowSample{Rate: 1000, Pool: seq * 1000, Seq: seq}, now)
			now = now.Add(time.Second / 4)
		}
	}
	step(150)
	seq = 0 // restart
	step(150)
	seq += 5000 // outage
	step(150)
	if st.eff != 1000 {
		t.Errorf("rate %v, want 1000", st.eff)
	}
}

// An interface that sees a sample every two seconds never fills a window
// of 30 samples in half a minute; its losses are still made up for where
// the pool shows them, and a shared sequence's gaps are still not.
func TestQuietInterface(t *testing.T) {
	for _, tc := range []struct {
		name   string
		shared bool
		want   float64
	}{{"lossy", false, 1280}, {"shared sequence", true, 1024}} {
		st := &sfState{}
		now := time.Unix(1e9, 0)
		var pool, seq uint32
		for i := 0; i < 60; i++ {
			seq++
			if tc.shared {
				seq += 2 // two other interfaces' samples in between
				pool += 1024
			} else {
				pool += 1024
				if i%5 == 4 {
					continue // lost on the way
				}
			}
			st.update(&sflow.FlowSample{Rate: 1024, Pool: pool, Seq: seq}, now)
			now = now.Add(2 * time.Second)
		}
		// a window holds some 15 samples: one loss more or less is 2%
		if st.eff < tc.want*0.97 || st.eff > tc.want*1.03 {
			t.Errorf("%s: %v, want %v", tc.name, st.eff, tc.want)
		}
	}
}
