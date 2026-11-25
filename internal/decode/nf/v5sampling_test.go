package nf

import (
	"net/netip"
	"testing"

	"github.com/githubflyideas/traffic66/internal/encode"
)

func v5Msg(t *testing.T, d *Decoder, headerRate uint16, seq uint32) flowRec {
	t.Helper()
	recs := []encode.V5Record{{Src: cli, Dst: srv, In: 3, Out: 4, Packets: 10, Bytes: 1500,
		First: 40000, Last: 59000, SrcPort: 50522, DstPort: 443, Proto: 6}}
	res, err := d.Decode(encode.V5(60000, uint32(base.Unix()), seq, headerRate, recs), exp, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Records) != 1 {
		t.Fatalf("got %d records, want 1", len(res.Records))
	}
	r := res.Records[0]
	return flowRec{mult: r.Mult, sampling: r.Sampling, known: r.SamplingKnown, bytes: r.Bytes}
}

type flowRec struct {
	mult     float64
	sampling uint32
	known    bool
	bytes    uint64
}

// A device sampling 1 in 1000 that does not say so in the v5 header: the
// operator supplies the rate by hand, and it has to be applied, or the
// traffic is recorded at a thousandth of what went past.
func TestV5ManualRateIsApplied(t *testing.T) {
	d := NewDecoder()
	d.Manual = map[netip.Addr]uint32{exp: 1000}
	got := v5Msg(t, d, 0, 1)
	if got.mult != 1000 {
		t.Errorf("multiplier %v, want 1000: the hand-set rate was not applied", got.mult)
	}
	if got.sampling != 1000 || !got.known {
		t.Errorf("sampling %d known %v, want 1000 and known", got.sampling, got.known)
	}
}

// A rate in the header wins over one set by hand: the device knows better.
func TestV5HeaderRateWinsOverManual(t *testing.T) {
	d := NewDecoder()
	d.Manual = map[netip.Addr]uint32{exp: 1000}
	got := v5Msg(t, d, 100, 1)
	if got.mult != 100 || got.sampling != 100 || !got.known {
		t.Errorf("got %+v, want the header's 100", got)
	}
}

// Nothing declared and nothing set: the counts are taken as they came, but
// the collector does not claim to know the rate. Saying it knows the rate
// is 1 is what stopped the interface comparison explaining the difference.
func TestV5NoRateIsNotClaimedKnown(t *testing.T) {
	d := NewDecoder()
	got := v5Msg(t, d, 0, 1)
	if got.mult != 1 {
		t.Errorf("multiplier %v, want 1", got.mult)
	}
	if got.known {
		t.Error("SamplingKnown is true, but no rate was declared or set")
	}
}

// An exporter known to export every packet: rate 1, and known.
func TestV5UnsampledIsKnown(t *testing.T) {
	d := NewDecoder()
	d.Unsampled = map[netip.Addr]bool{exp: true}
	got := v5Msg(t, d, 0, 1)
	if got.mult != 1 || !got.known {
		t.Errorf("got %+v, want rate 1 and known", got)
	}
}

// A rate that was never declared must not be reported as the exporter's own
// on the sources page, nor lent to another observation domain of the same
// device. Forcing it to 1 made it look declared.
func TestV5UndeclaredRateIsNotStoredAsDeclared(t *testing.T) {
	d := NewDecoder()
	d.Manual = map[netip.Addr]uint32{exp: 1000}
	v5Msg(t, d, 0, 1)
	st := d.Stats()[0]
	if !st.Manual {
		t.Error("the sources page does not show the rate as hand-set")
	}
	if st.Sampling != 1000 {
		t.Errorf("sources page shows rate %d, want 1000", st.Sampling)
	}
	if st.SamplingState != "manual" {
		t.Errorf("sampling state %q, want \"manual\"", st.SamplingState)
	}
}
