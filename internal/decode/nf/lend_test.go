package nf

import (
	"testing"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// sampler declares one sampler's rate in an options record.
func sampler(id, rate uint32) (encode.Template, encode.Values) {
	return encode.Template{ID: 410, Scope: []encode.Field{{ID: ieSamplerID, Len: 4}},
			Fields: []encode.Field{{ID: ieSamplingInterval, Len: 4}}},
		encode.Values{encode.U(uint64(id), 4), encode.U(uint64(rate), 4)}
}

// What one observation domain lends to another is the rate the device
// declares — when it declares one. A device sampling one interface at
// 1:100 and another at 1:1000 used to lend whichever arrived last, and the
// borrowing domain could not tell it was not its own: a tenfold error in
// every number that domain produced.
func TestADeviceWithTwoRatesLendsNeither(t *testing.T) {
	d := NewDecoder()
	opt, vals := sampler(1, 100)
	m := encode.NewIPFIX()
	m.AddTemplate(opt)
	m.AddData(opt.ID, []encode.Values{vals})
	if _, err := d.Decode(m.IPFIX(uint32(base.Unix()), 0, 9), exp, base); err != nil {
		t.Fatal(err)
	}
	if got := d.byAddr[exp]; got != 100 {
		t.Fatalf("one rate declared: byAddr = %d, want 100 — a device with one rate still lends it", got)
	}

	// the same device declares a second, different rate for another sampler
	_, vals2 := sampler(2, 1000)
	m = encode.NewIPFIX()
	m.AddTemplate(opt)
	m.AddData(opt.ID, []encode.Values{vals2})
	if _, err := d.Decode(m.IPFIX(uint32(base.Unix()), 0, 9), exp, base); err != nil {
		t.Fatal(err)
	}
	if got := d.byAddr[exp]; got == 1000 {
		t.Errorf("byAddr = %d: the last rate to arrive was lent, which is the bug", got)
	}

	// a domain of its own that declares nothing must not borrow either of
	// them: it waits for a rate instead of taking one that may be wrong
	if got := decodeOne(t, d, 11, nil, nil, ipfixFlow(1, 2)); got != 0 {
		t.Errorf("another domain took mult %v from a device with two rates", got)
	}
}

// A device that declares the same rate twice still lends it: the two
// agree, so there is nothing for the borrower to get wrong.
func TestADeviceWithOneRateStillLendsIt(t *testing.T) {
	d := NewDecoder()
	opt, _ := sampler(0, 0)
	for _, id := range []uint32{1, 2} {
		_, vals := sampler(id, 4096)
		m := encode.NewIPFIX()
		m.AddTemplate(opt)
		m.AddData(opt.ID, []encode.Values{vals})
		if _, err := d.Decode(m.IPFIX(uint32(base.Unix()), 0, 9), exp, base); err != nil {
			t.Fatal(err)
		}
	}
	if got := d.byAddr[exp]; got != 4096 {
		t.Errorf("byAddr = %d, want 4096", got)
	}
	if got := decodeOne(t, d, 11, nil, nil, ipfixFlow(1, 2)); got != 4096 {
		t.Errorf("another domain got mult %v, want 4096", got)
	}
}
