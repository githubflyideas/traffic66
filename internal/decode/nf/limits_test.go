package nf

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// dataOnly is an IPFIX message with a data set whose template was never sent.
func dataOnly() []byte {
	m := encode.NewIPFIX()
	m.AddData(ipfixData.ID, []encode.Values{ipfixRec(1000)})
	return m.IPFIX(uint32(base.Unix()), 0, 1)
}

func TestSessionCapAndExpiry(t *testing.T) {
	d := NewDecoder()
	d.MaxSessions = 2
	for i, want := range []error{nil, nil, ErrTooManyExporters} {
		a := netip.AddrFrom4([4]byte{192, 0, 2, byte(i + 1)})
		if _, err := d.Decode(dataOnly(), a, base); !errors.Is(err, want) && !(want == nil && err == nil) {
			t.Fatalf("exporter %d: %v, want %v", i, err, want)
		}
	}
	// held sets expire even when nothing waits for a sampling rate
	d.Flush(base.Add(d.HoldFor + time.Second))
	var dropped uint64
	for _, s := range d.Stats() {
		dropped += s.DroppedSets
	}
	if dropped != 2 {
		t.Fatalf("dropped held sets %d, want 2", dropped)
	}
	// silent exporters are forgotten after a day, making room for new ones
	d.Flush(base.Add(sessionIdle + time.Hour))
	if n := len(d.Stats()); n != 0 {
		t.Fatalf("%d sessions left after a day of silence", n)
	}
	if _, err := d.Decode(dataOnly(), netip.MustParseAddr("192.0.2.9"), base.Add(sessionIdle+time.Hour)); err != nil {
		t.Fatal(err)
	}
}
