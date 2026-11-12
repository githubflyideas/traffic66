package accuracy

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/store"
)

// A template with the flow's own start and end in milliseconds.
var ipfixTimed = encode.Template{ID: 310, Fields: []encode.Field{
	{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2}, {ID: 4, Len: 1},
	{ID: 1, Len: 8}, {ID: 2, Len: 8}, {ID: 152, Len: 8}, {ID: 153, Len: 8},
}}

func timedRecord(cli string, startMs, endMs uint64) encode.Values {
	return encode.Values{encode.A(netip.MustParseAddr(cli)), encode.A(netip.MustParseAddr("198.51.100.9")), encode.U(40000, 2), encode.U(443, 2),
		encode.U(6, 1), encode.U(5000, 8), encode.U(10, 8), encode.U(startMs, 8), encode.U(endMs, 8)}
}

func sendTimed(c interface {
	Handle([]byte, netip.Addr, time.Time) bool
}, exp netip.Addr, now time.Time, recs ...encode.Values) {
	m := encode.NewIPFIX()
	m.AddTemplate(ipfixTimed)
	m.AddData(ipfixTimed.ID, recs)
	c.Handle(m.IPFIX(uint32(now.Unix()), 0, 7), exp, now)
}

// One record with an end time no database can hold (2^55 ms) used to fail
// the whole minute's batch, other exporters' rows with it.
func TestImpossibleTimeDoesNotSinkOtherExporters(t *testing.T) {
	for _, live := range []bool{true, false} {
		c, p, st := setup(t)
		c.Live = live
		now := time.Now().UTC().Truncate(time.Second)
		ms := uint64(now.UnixMilli())
		good, bad := netip.MustParseAddr("10.7.0.1"), netip.MustParseAddr("10.7.0.2")
		sendTimed(c, good, now, timedRecord("10.1.1.1", ms-30000, ms-1000))
		sendTimed(c, bad, now, timedRecord("10.1.1.2", ms-30000, 1<<55))
		c.Tick(now.Add(6 * time.Minute)) // past the wait for a sampling rate
		got := wireTotal(t, p, st, good.String(), now.Add(-time.Hour), now.Add(time.Hour))
		if p.WriteErrs.Load() != 0 || got != 5000+10*18 {
			t.Fatalf("live=%v: write errors %d, good exporter wire %v, want 5180", live, p.WriteErrs.Load(), got)
		}
	}
}

// Microseconds in a millisecond field put a flow 56,000 years ahead. It
// must not move the duplicate window, or the same flow seen by two
// exporters is counted twice from then on.
func TestFutureTimeKeepsDuplicateCheck(t *testing.T) {
	c, p, st := setup(t)
	now := time.Now().UTC().Truncate(time.Second)
	ms := uint64(now.UnixMilli())
	e1, e2, e3 := netip.MustParseAddr("10.7.0.1"), netip.MustParseAddr("10.7.0.2"), netip.MustParseAddr("10.7.0.3")
	sendTimed(c, e3, now, timedRecord("10.1.1.9", ms*1000-30000, ms*1000))
	sendTimed(c, e1, now, timedRecord("10.1.1.1", ms-30000, ms-1000))
	sendTimed(c, e2, now, timedRecord("10.1.1.1", ms-30000, ms-1000))
	c.Tick(now.Add(6 * time.Minute)) // past the wait for a sampling rate
	p.FlushRows()
	p.FlushRollups()
	tot, err := st.Totals(storeQuery(now))
	if err != nil || tot.Wire != 5180 {
		t.Fatalf("wire %d %v, want 5180 (one flow, reported by two exporters)", tot.Wire, err)
	}
}

func storeQuery(now time.Time) store.Query {
	return store.Query{From: now.Add(-time.Hour), To: now.Add(time.Hour)}
}
