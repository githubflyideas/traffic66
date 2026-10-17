package nf

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

var ipfixData = encode.Template{ID: 300, Fields: []encode.Field{
	{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
	{ID: ieProtocol, Len: 1}, {ID: ieOctetDelta, Len: 8}, {ID: iePacketDelta, Len: 8},
}}

func ipfixRec(bytes uint64) encode.Values {
	return encode.Values{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2), encode.U(6, 1),
		encode.U(bytes, 8), encode.U(1, 8)}
}

// ipfixStream sends msgs messages of perMsg records each, dropping every
// dropEvery-th message (0 = none), and returns the sum of Mult-scaled bytes
// seen after the compensation window has settled.
func ipfixStream(t *testing.T, d *Decoder, msgs, perMsg, dropEvery int, order func([]int) []int) (sent, scaled float64) {
	t.Helper()
	now := base
	seq := uint32(0)
	type msg struct {
		b  []byte
		at time.Time
	}
	var out []msg
	for i := 0; i < msgs; i++ {
		m := encode.NewIPFIX()
		if i%50 == 0 {
			m.AddTemplate(ipfixData)
		}
		var recs []encode.Values
		for j := 0; j < perMsg; j++ {
			recs = append(recs, ipfixRec(1000))
		}
		m.AddData(ipfixData.ID, recs)
		b := m.IPFIX(uint32(now.Unix()), seq, 1)
		seq += uint32(m.DataRecords())
		if dropEvery == 0 || i%dropEvery != dropEvery-1 || i%50 == 0 {
			out = append(out, msg{b, now})
		}
		now = now.Add(100 * time.Millisecond)
	}
	idx := make([]int, len(out))
	for i := range idx {
		idx[i] = i
	}
	if order != nil {
		idx = order(idx)
	}
	for n, i := range idx {
		res, err := d.Decode(out[i].b, exp, out[i].at)
		if err != nil {
			t.Fatal(err)
		}
		if n < len(idx)/2 { // let the window settle first
			continue
		}
		for _, r := range res.Records {
			scaled += float64(r.Bytes) * r.Mult
		}
	}
	d.Unsampled[exp] = true
	return float64((msgs-msgs/2)*perMsg) * 1000, scaled
}

func TestIPFIXLossIsCompensated(t *testing.T) {
	d := NewDecoder()
	d.Unsampled[exp] = true
	sent, scaled := ipfixStream(t, d, 2000, 10, 10, nil) // 10% of messages lost
	st := d.Stats()[0]
	if math.Abs(st.LossComp-1/0.9) > 0.02 {
		t.Fatalf("loss multiplier %.3f, want about %.3f", st.LossComp, 1/0.9)
	}
	if e := math.Abs(scaled-sent) / sent; e > 0.03 {
		t.Fatalf("compensated total %.0f vs sent %.0f (error %.1f%%)", scaled, sent, e*100)
	}
	if st.LostRecords == 0 {
		t.Fatal("loss not counted")
	}
}

func TestIPFIXReorderIsNotLoss(t *testing.T) {
	d := NewDecoder()
	d.Unsampled[exp] = true
	swapPairs := func(idx []int) []int {
		for i := 1; i+1 < len(idx); i += 7 {
			idx[i], idx[i+1] = idx[i+1], idx[i]
		}
		return idx
	}
	ipfixStream(t, d, 2000, 10, 0, swapPairs)
	st := d.Stats()[0]
	if st.LostRecords != 0 {
		t.Fatalf("reordering counted as %d lost records", st.LostRecords)
	}
	if st.LossComp != 1 {
		t.Fatalf("loss multiplier %.3f for a lossless stream", st.LossComp)
	}
}

// lossyRun sends dur worth of IPFIX messages (one every 10 ms, 10 to 30
// records each), loses each message with probability p, and returns the
// Mult-scaled bytes over the whole run divided by the bytes sent.
func lossyRun(t *testing.T, dur time.Duration, p float64) float64 {
	t.Helper()
	d := NewDecoder()
	d.Unsampled[exp] = true
	rng := rand.New(rand.NewSource(1))
	now := base
	seq := uint32(0)
	var sent, scaled float64
	for at := time.Duration(0); at < dur; at += 10 * time.Millisecond {
		m := encode.NewIPFIX()
		m.AddTemplate(ipfixData)
		n := 10 + rng.Intn(21)
		recs := make([]encode.Values, n)
		for j := range recs {
			recs[j] = ipfixRec(1000)
		}
		m.AddData(ipfixData.ID, recs)
		b := m.IPFIX(uint32(now.Unix()), seq, 1)
		seq += uint32(m.DataRecords())
		sent += float64(n) * 1000
		if rng.Float64() >= p {
			res, err := d.Decode(b, exp, now.Add(at))
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range res.Records {
				scaled += float64(r.Bytes) * r.Mult
			}
		}
	}
	return scaled / sent
}

// A session that is lossy from the start must be compensated from its
// first seconds, not only after a long warm-up window.
func TestLossCompensatedFromStart(t *testing.T) {
	for _, dur := range []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second, 300 * time.Second} {
		r := lossyRun(t, dur, 0.37)
		t.Logf("37%% loss, %v: compensated/true = %.3f", dur, r)
		if dur == 60*time.Second && r < 0.95 {
			t.Fatalf("60 s at 37%% loss: compensated/true %.3f, want >= 0.95", r)
		}
	}
	if r := lossyRun(t, 60*time.Second, 0); math.Abs(r-1) > 1e-9 {
		t.Fatalf("lossless run: compensated/true %.6f, want 1", r)
	}
}
