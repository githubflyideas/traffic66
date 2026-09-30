// Package accuracy holds end-to-end tests: real export datagrams go through
// the collector, the pipeline and the store, and the stored link-layer
// totals are compared with the traffic that was generated.
package accuracy

import (
	"math"
	"math/rand/v2"
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/store"
)

// syncSink feeds the pipeline synchronously so the test is deterministic.
type syncSink struct{ p *pipeline.Pipeline }

func (s syncSink) Submit(r []flow.Record)             { s.p.Ingest(r) }
func (s syncSink) SubmitCounters(c []flow.IfCounters) { s.p.IngestCounters(c) }

func setup(t *testing.T) (*collector.Collector, *pipeline.Pipeline, *store.Store) {
	t.Helper()
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	p := pipeline.New(pipeline.Config{}, st, enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	c := collector.New(syncSink{p})
	c.Live = false
	return c, p, st
}

func wireTotal(t *testing.T, p *pipeline.Pipeline, st *store.Store, exporter string, from, to time.Time) float64 {
	t.Helper()
	p.FlushRows()
	p.FlushRollups()
	tot, err := st.Totals(store.Query{From: from, To: to, Filters: []store.Filter{{Field: "exporter", Value: exporter}}})
	if err != nil {
		t.Fatal(err)
	}
	return float64(tot.Wire)
}

func check(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	e := (got - want) / want
	t.Logf("%s: stored %.0f bytes, generated %.0f bytes, error %+.2f%%", what, got, want, e*100)
	if math.Abs(e) > tol {
		t.Errorf("%s: error %+.2f%% exceeds ±%.1f%%", what, e*100, tol*100)
	}
}

var (
	clients = []netip.Addr{netip.MustParseAddr("10.1.0.10"), netip.MustParseAddr("10.1.0.11"), netip.MustParseAddr("10.1.0.12")}
	servers = []netip.Addr{netip.MustParseAddr("198.51.100.20"), netip.MustParseAddr("203.0.113.30")}
)

// sFlow, 1:1000, random frame sizes, 3% of datagrams lost on the way.
func TestSFlowSampledWithDatagramLoss(t *testing.T) {
	c, p, st := setup(t)
	agent := netip.MustParseAddr("10.0.0.2")
	rng := rand.New(rand.NewPCG(1, 2))
	const rate = 1000
	start := time.Now().UTC().Truncate(time.Minute).Add(-20 * time.Minute)
	var truth float64
	var pool, seq, dseq uint32
	var batch []encode.SFlowSample
	now := start
	send := func() {
		dseq++
		if rng.Float64() >= 0.03 {
			c.Handle(encode.SFlow(agent, 0, dseq, 1000, batch, nil), agent, now)
		}
		batch = batch[:0]
	}
	const packets = 60_000_000 // ten minutes at 100 kpps
	for i := 0; i < packets; i++ {
		ip := 46 + rng.IntN(1500-46+1)
		truth += float64(ip + 18)
		pool++
		if i%rate != rate-1 {
			continue
		}
		now = start.Add(time.Duration(float64(i) / packets * float64(10*time.Minute)))
		seq++
		cl, sv := clients[rng.IntN(len(clients))], servers[rng.IntN(len(servers))]
		hdr, _ := encode.Frame(cl, sv, 40000, 443, 6, 0, ip, 0x18)
		batch = append(batch, encode.SFlowSample{Seq: seq, SourceID: 5, Rate: rate, Pool: pool, In: 5, Out: 6, FrameLen: uint32(ip + 18), Header: hdr})
		if len(batch) == 8 {
			send()
		}
	}
	if len(batch) > 0 {
		send()
	}
	got := wireTotal(t, p, st, agent.String(), start.Add(-time.Hour), start.Add(time.Hour))
	check(t, "sFlow 1:1000 with 3% datagram loss", got, truth, 0.01)
}

var v9Tmpl = encode.Template{ID: 256, Fields: []encode.Field{
	{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2}, {ID: 4, Len: 1},
	{ID: 1, Len: 4}, {ID: 2, Len: 4}, {ID: 10, Len: 2}, {ID: 14, Len: 2}, {ID: 48, Len: 1},
}}
var v9Opt = encode.Template{ID: 257, Scope: []encode.Field{{ID: 1, Len: 4}},
	Fields: []encode.Field{{ID: 48, Len: 1}, {ID: 49, Len: 1}, {ID: 50, Len: 4}}}

// NetFlow v9, 1:100 random sampling, IP-layer byte counts, 5% of export
// packets lost. The stored total must be the link-layer estimate.
func TestNetFlow9SampledWithPacketLoss(t *testing.T) {
	c, p, st := setup(t)
	exp := netip.MustParseAddr("10.0.0.1")
	rng := rand.New(rand.NewPCG(3, 4))
	const rate = 100
	start := time.Now().UTC().Truncate(time.Minute).Add(-20 * time.Minute)
	var truth float64
	var seq uint32
	for i := 0; i < 6000; i++ { // one export packet every 100 ms for 10 minutes
		now := start.Add(time.Duration(i) * 100 * time.Millisecond)
		m := encode.NewV9()
		if i%100 == 0 {
			m.AddTemplate(v9Tmpl)
			m.AddTemplate(v9Opt)
			m.AddData(v9Opt.ID, []encode.Values{{encode.U(0, 4), encode.U(1, 1), encode.U(2, 1), encode.U(rate, 4)}})
		}
		var recs []encode.Values
		for j := 0; j < 20; j++ {
			pk := 1 + rng.IntN(40)
			ipLen := 40 + rng.IntN(1460)
			truth += float64(pk*rate) * float64(ipLen+18)
			recs = append(recs, encode.Values{encode.A(clients[j%3]), encode.A(servers[j%2]), encode.U(40000, 2), encode.U(443, 2),
				encode.U(6, 1), encode.U(uint64(pk*ipLen), 4), encode.U(uint64(pk), 4), encode.U(1, 2), encode.U(2, 2), encode.U(1, 1)})
		}
		m.AddData(v9Tmpl.ID, recs)
		seq++
		if i%100 != 0 && rng.Float64() < 0.05 {
			continue // lost on the way
		}
		c.Handle(m.V9(uint32(i*100), uint32(now.Unix()), seq, 1), exp, now)
	}
	got := wireTotal(t, p, st, exp.String(), start.Add(-time.Hour), start.Add(time.Hour))
	check(t, "NetFlow v9 1:100 with 5% export loss", got, truth, 0.015)
}

var ipfixTmpl = encode.Template{ID: 300, Fields: []encode.Field{
	{ID: 8, Len: 4}, {ID: 12, Len: 4}, {ID: 7, Len: 2}, {ID: 11, Len: 2}, {ID: 4, Len: 1},
	{ID: 1, Len: 8}, {ID: 2, Len: 8}, {ID: 10, Len: 4}, {ID: 14, Len: 4},
}}
var ipfixOpt = encode.Template{ID: 301, Scope: []encode.Field{{ID: 149, Len: 4}},
	Fields: []encode.Field{{ID: 305, Len: 4}, {ID: 306, Len: 4}}}

// IPFIX whose sampling rate (1:10) is only announced two minutes after the
// data started: held records must be scaled, not counted 1:1.
func TestIPFIXLateSamplingRate(t *testing.T) {
	c, p, st := setup(t)
	exp := netip.MustParseAddr("10.0.0.3")
	rng := rand.New(rand.NewPCG(5, 6))
	start := time.Now().UTC().Truncate(time.Minute).Add(-20 * time.Minute)
	var truth float64
	var seq uint32
	for i := 0; i < 600; i++ { // one message per second for 10 minutes
		now := start.Add(time.Duration(i) * time.Second)
		m := encode.NewIPFIX()
		if i%30 == 0 {
			m.AddTemplate(ipfixTmpl)
			if i >= 120 {
				m.AddTemplate(ipfixOpt)
				m.AddData(ipfixOpt.ID, []encode.Values{{encode.U(9, 4), encode.U(1, 4), encode.U(9, 4)}})
			}
		}
		var recs []encode.Values
		for j := 0; j < 10; j++ {
			pk := 1 + rng.IntN(20)
			ipLen := 40 + rng.IntN(1460)
			truth += float64(pk*10) * float64(ipLen+18)
			recs = append(recs, encode.Values{encode.A(clients[j%3]), encode.A(servers[j%2]), encode.U(40000, 2), encode.U(443, 2),
				encode.U(6, 1), encode.U(uint64(pk*ipLen), 8), encode.U(uint64(pk), 8), encode.U(1, 4), encode.U(2, 4)})
		}
		m.AddData(ipfixTmpl.ID, recs)
		c.Handle(m.IPFIX(uint32(now.Unix()), seq, 9), exp, now)
		seq += uint32(m.DataRecords())
	}
	c.Tick(start.Add(11 * time.Minute))
	got := wireTotal(t, p, st, exp.String(), start.Add(-time.Hour), start.Add(time.Hour))
	check(t, "IPFIX with the sampling rate announced 2 min late", got, truth, 0.001)
}
