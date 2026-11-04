package collector

import (
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/decode/sflow"
)

// Two samplers under one source id, each with its own pool and sequence
// number, sending in turn: every packet either of them stood for counts.
func TestSamplersSharingASource(t *testing.T) {
	at := time.Unix(1700000000, 0)
	for _, c := range []struct {
		name               string
		poolA, poolB       uint32
		seqA, seqB         uint32
		loseA, loseB, swap int // lose every nth sample of A/B; swap a pair of A every nth
	}{
		{"b behind", 5000, 3000, 10, 7, 0, 0, 0},
		{"b ahead", 3000, 900000, 7, 9000, 0, 0, 0},
		{"far apart", 9000000, 100, 400000, 3, 0, 0, 0},
		{"losses", 9000000, 100, 400000, 3, 7, 5, 0},
		{"reordered", 9000000, 100, 400000, 3, 0, 0, 9},
	} {
		// datagram numbers known, as from a real agent; a lost sample is
		// in a lost datagram
		st := &sfState{evalAt: at, dknown: true, dper: 1}
		var got, want float64
		type smp struct{ pool, seq uint32 }
		var a, b []smp
		for i := uint32(1); i <= 400; i++ {
			a = append(a, smp{c.poolA + i*100, c.seqA + i})
			b = append(b, smp{c.poolB + i*100, c.seqB + i})
		}
		want = 800 * 100
		if c.swap > 0 {
			for i := c.swap; i+1 < len(a); i += c.swap {
				a[i], a[i+1] = a[i+1], a[i]
			}
		}
		for i := range a {
			for k, x := range []smp{a[i], b[i]} {
				lose := c.loseA
				if k == 1 {
					lose = c.loseB
				}
				if lose > 0 && i > 20 && i%lose == 0 {
					st.dlost++
					continue
				}
				w, _ := st.update(&sflow.FlowSample{Rate: 100, Pool: x.pool, Seq: x.seq}, at)
				got += w
			}
		}
		if r := got / want; r < 0.99 || r > 1.01 {
			t.Errorf("%s: counted %.1f%% of the packets", c.name, r*100)
		}
		if c.loseA == 0 && c.loseB == 0 && c.swap == 0 && st.lost != 0 {
			t.Errorf("%s: %d samples reported lost, none were", c.name, st.lost)
		}
	}
}

// One delivered twice adds nothing, on either sampler.
func TestRepeatedSampleAddsNothing(t *testing.T) {
	st := &sfState{evalAt: time.Unix(1700000000, 0), dknown: true, dper: 1}
	for i := uint32(1); i <= 20; i++ {
		st.update(&sflow.FlowSample{Rate: 100, Pool: 5000 + i*100, Seq: 10 + i}, time.Time{})
		st.update(&sflow.FlowSample{Rate: 100, Pool: 3000 + i*100, Seq: 7 + i}, time.Time{})
	}
	for _, s := range []sflow.FlowSample{{Rate: 100, Pool: 7000, Seq: 30}, {Rate: 100, Pool: 4500, Seq: 22}} {
		if w, _ := st.update(&s, time.Time{}); w != 0 {
			t.Errorf("repeated pool=%d seq=%d counted %g", s.Pool, s.Seq, w)
		}
	}
}
