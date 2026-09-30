//go:build linux || darwin

package accuracy

import (
	"context"
	"math/rand/v2"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/collector"
	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/store"
)

func cpuTime() time.Duration {
	var ru syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

// TestLoadUDP sends IPFIX over UDP at the design rate and checks that
// nothing is dropped. Run with T66_LOAD=1 (optionally T66_LOAD_RATE, _SECS).
func TestLoadUDP(t *testing.T) {
	if os.Getenv("T66_LOAD") == "" {
		t.Skip("set T66_LOAD=1 to run the load test")
	}
	rate, _ := strconv.Atoi(os.Getenv("T66_LOAD_RATE"))
	if rate == 0 {
		rate = 5000
	}
	secs, _ := strconv.Atoi(os.Getenv("T66_LOAD_SECS"))
	if secs == 0 {
		secs = 60
	}
	st, err := store.Open(store.Options{Dir: t.TempDir(), RawDays: 30, MemoryFraction: 0.10})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	p := pipeline.New(pipeline.Config{}, st, enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	c := collector.New(p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)
	l, err := c.Listen("127.0.0.1:0", "ipfix")
	if err != nil {
		t.Fatal(err)
	}
	go c.Serve(ctx, l)
	conn, err := net.Dial("udp", l.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(7, 8))
	const perMsg = 30
	msgs := rate / perMsg
	cpu0 := cpuTime()
	start := time.Now()
	var seq uint32
	sent := 0
	tick := time.NewTicker(time.Second / time.Duration(msgs))
	defer tick.Stop()
	for i := 0; i < msgs*secs; i++ {
		<-tick.C
		m := encode.NewIPFIX()
		if i%100 == 0 {
			m.AddTemplate(ipfixTmpl)
			m.AddTemplate(ipfixOpt)
			m.AddData(ipfixOpt.ID, []encode.Values{{encode.U(9, 4), encode.U(1, 4), encode.U(0, 4)}})
		}
		var recs []encode.Values
		for j := 0; j < perMsg; j++ {
			cl := netip.AddrFrom4([4]byte{10, 1, byte(rng.IntN(20)), byte(rng.IntN(250))})
			sv := netip.AddrFrom4([4]byte{byte(1 + rng.IntN(220)), byte(rng.IntN(256)), byte(rng.IntN(256)), byte(1 + rng.IntN(250))})
			pk := 1 + rng.IntN(30)
			recs = append(recs, encode.Values{encode.A(cl), encode.A(sv), encode.U(uint64(1024+rng.IntN(60000)), 2), encode.U([]uint64{443, 80, 53, 22, 8801}[rng.IntN(5)], 2),
				encode.U(6, 1), encode.U(uint64(pk*(40+rng.IntN(1460))), 8), encode.U(uint64(pk), 8), encode.U(1, 4), encode.U(2, 4)})
		}
		m.AddData(ipfixTmpl.ID, recs)
		conn.Write(m.IPFIX(uint32(time.Now().Unix()), seq, 9))
		seq += uint32(m.DataRecords())
		sent += perMsg
	}
	time.Sleep(3 * time.Second)
	el := time.Since(start)
	p.FlushRows()
	cpu := cpuTime() - cpu0
	got := p.Records.Load()
	t.Logf("sent %d records in %s (%.0f/s); pipeline saw %d, dropped %d, undecoded datagrams %d, lost per sequence %d",
		sent, el.Round(time.Second), float64(sent)/el.Seconds(), got, p.Dropped.Load(), l.Undecoded.Load(), c.Sources()[0].LostRecords)
	t.Logf("CPU %.0f%% of one core (including the sender)", cpu.Seconds()/el.Seconds()*100)
	logPeakMemory(t)
	if p.Dropped.Load() > 0 || got < uint64(sent) {
		t.Errorf("records were lost: sent %d, processed %d, dropped %d", sent, got, p.Dropped.Load())
	}
}

// TestLoadQueries bulk-loads hours of flows at the design rate and times
// the queries the UI makes. Run with T66_LOAD=1 T66_LOAD_HOURS=n.
func TestLoadQueries(t *testing.T) {
	if os.Getenv("T66_LOAD") == "" {
		t.Skip("set T66_LOAD=1 to run the load test")
	}
	hours, _ := strconv.Atoi(os.Getenv("T66_LOAD_HOURS"))
	if hours == 0 {
		hours = 1
	}
	dir := os.Getenv("T66_LOAD_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	st, err := store.Open(store.Options{Dir: dir, RawDays: 30, MemoryFraction: 0.10})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	p := pipeline.New(pipeline.Config{}, st, enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	rng := rand.New(rand.NewPCG(9, 10))
	end := time.Now().UTC().Truncate(time.Hour)
	begin := end.Add(-time.Duration(hours) * time.Hour)
	exp := netip.MustParseAddr("10.0.0.1")
	servers := make([]netip.Addr, 50000)
	for i := range servers {
		servers[i] = netip.AddrFrom4([4]byte{byte(1 + rng.IntN(220)), byte(rng.IntN(256)), byte(rng.IntN(256)), byte(1 + rng.IntN(250))})
	}
	spoofed := os.Getenv("T66_LOAD_SPOOFED") != "" // every remote address different, as in a spoofed flood
	// Each client talks to its own ~100 servers per hour, drawn from the
	// popular ones, as on real networks. T66_LOAD_UNIQUE=1 makes every flow a
	// new conversation (worst case).
	unique := os.Getenv("T66_LOAD_UNIQUE") != ""
	peersOf := map[netip.Addr][]netip.Addr{}
	t0 := time.Now()
	var n int
	for m := begin; m.Before(end); m = m.Add(time.Minute) {
		recs := make([]flow.Record, 0, 5000*60)
		for i := 0; i < 5000*60; i++ {
			ts := m.Add(time.Duration(rng.IntN(60000)) * time.Millisecond)
			cl := netip.AddrFrom4([4]byte{10, 1, byte(rng.IntN(20)), byte(rng.IntN(250))})
			// popular servers get most flows, like real traffic
			sv := servers[int(float64(len(servers))*rng.Float64()*rng.Float64()*rng.Float64())]
			if !unique && !spoofed {
				ps := peersOf[cl]
				if len(ps) < 100 {
					peersOf[cl] = append(ps, sv)
				} else {
					sv = ps[rng.IntN(len(ps))]
				}
			}
			if spoofed {
				sv = netip.AddrFrom4([4]byte{byte(1 + rng.IntN(220)), byte(rng.IntN(256)), byte(rng.IntN(256)), byte(1 + rng.IntN(250))})
			}
			pk := uint64(1 + rng.IntN(30))
			recs = append(recs, flow.Record{Start: ts.Add(-time.Duration(rng.IntN(30)) * time.Second), End: ts, Src: cl, Dst: sv,
				SrcPort: uint16(1024 + rng.IntN(60000)), DstPort: []uint16{443, 80, 53, 22, 8801}[rng.IntN(5)], Proto: 6,
				Bytes: pk * uint64(40+rng.IntN(1460)), Packets: pk, Mult: 100, Sampling: 100, SamplingKnown: true,
				InIf: 1, OutIf: 2, Exporter: exp, Source: flow.SrcIPFIX})
		}
		p.Ingest(recs)
		p.FlushRows()
		p.FlushRollups() // the live pipeline flushes rollups every minute too
		n += len(recs)
		if m.Minute() == 59 {
			if err := st.Seal(m.Add(2 * time.Minute)); err != nil {
				t.Fatal(err)
			}
		}
	}
	p.FlushRollups()
	st.Seal(end.Add(time.Hour))
	load := time.Since(t0)
	u := st.Usage()
	logPeakMemory(t)
	t.Logf("loaded %d records (%d rows) over %d h in %s = %.0f records/s; disk %.2f GB (%.1f bytes/row)",
		n, p.Rows.Load(), hours, load.Round(time.Second), float64(n)/load.Seconds(), float64(u.DiskBytes)/1e9, float64(u.DiskBytes)/float64(p.Rows.Load()))

	timeIt := func(name string, f func() error) {
		t0 := time.Now()
		if err := f(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		t.Logf("%-40s %6d ms", name, time.Since(t0).Milliseconds())
	}
	full := store.Query{From: end.Add(-24 * time.Hour), To: end}
	last1h := store.Query{From: end.Add(-time.Hour), To: end}
	host := []store.Filter{{Field: "ip", Value: "10.1.3.7"}}
	timeIt("24h totals (rollup)", func() error { _, e := st.Totals(full); return e })
	timeIt("24h series by app (rollup)", func() error { _, e := st.SeriesByApp(full, 6); return e })
	timeIt("24h top 66 clients (rollup)", func() error { _, e := st.TopN(full, "client", 66); return e })
	timeIt("24h top 66 ports (rollup)", func() error { _, e := st.TopN(full, "port", 66); return e })
	timeIt("1h totals (detail)", func() error { _, e := st.Totals(last1h); return e })
	timeIt("1h top 66 conversations (detail)", func() error { _, e := st.TopN(last1h, "conv", 66); return e })
	timeIt("1h series by app (detail)", func() error { _, e := st.SeriesByApp(last1h, 6); return e })
	full.Filters, last1h.Filters = host, host
	timeIt("1h one host: totals (detail)", func() error { _, e := st.Totals(last1h); return e })
	timeIt("one host, all loaded data: top 66 servers", func() error { _, e := st.TopN(full, "server", 66); return e })
	timeIt("one host, all loaded data: series", func() error { _, e := st.SeriesByApp(full, 6); return e })
	timeIt("latest 200 flow records (detail)", func() error { _, e := st.Records(last1h, 200); return e })
}

func logPeakMemory(t *testing.T) {
	if b, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "VmHWM") {
				t.Logf("peak memory (resident): %s", strings.TrimSpace(strings.TrimPrefix(l, "VmHWM:")))
			}
		}
	}
}
