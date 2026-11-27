package capture

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/flow"
)

func TestOpener(t *testing.T) {
	for _, tc := range []struct {
		flags uint8
		want  int8
	}{
		{0x02, 1},  // SYN alone opens
		{0x12, -1}, // SYN+ACK answers
		{0x10, 0},  // plain ACK says nothing
		{0x04, 0},  // RST says nothing
		{0x42, 1},  // SYN with ECE still opens (only SYN/ACK bits count)
		{0x00, 0},
	} {
		if got := opener(tc.flags); got != tc.want {
			t.Errorf("opener(%#x) = %d, want %d", tc.flags, got, tc.want)
		}
	}
}

type sliceSink struct{ recs []flow.Record }

func (s *sliceSink) Submit(rs []flow.Record) { s.recs = append(s.recs, rs...) }

var (
	capA = netip.MustParseAddr("10.0.0.1")
	capB = netip.MustParseAddr("10.0.0.2")
)

func TestRecordShape(t *testing.T) {
	c := &Capture{domain: 7}
	k := key{src: capA, dst: capB, sport: 1111, dport: 443, proto: 6, vlan: 9}
	first := time.Now().Add(-time.Minute)
	e := &entry{first: first, last: first.Add(30 * time.Second), pkts: 5, bytes: 4000, flags: 0x1B, opener: 1}
	r := c.record(k, e)
	if r.Start != first || r.End != e.last || r.Src != capA || r.Dst != capB ||
		r.SrcPort != 1111 || r.DstPort != 443 || r.Proto != 6 || r.VLAN != 9 {
		t.Errorf("record identity wrong: %+v", r)
	}
	if r.Bytes != 4000 || r.Packets != 5 || r.TCPFlags != 0x1B || r.Opener != 1 {
		t.Errorf("record counters wrong: %+v", r)
	}
	if !r.L2 || r.Sampling != 1 || r.Mult != 1 || !r.SamplingKnown ||
		r.Exporter != Exporter || r.Domain != 7 || r.Source != flow.SrcCapture || r.Direction != 255 {
		t.Errorf("record capture marks wrong: %+v", r)
	}
}

func TestFlushLifecycle(t *testing.T) {
	now := time.Now()
	sink := &sliceSink{}
	c := &Capture{sink: sink, table: map[key]*entry{}}
	kIdle := key{src: capA, dst: capB, sport: 1, dport: 2, proto: 6}
	kFresh := key{src: capA, dst: capB, sport: 3, dport: 4, proto: 17}
	kEmpty := key{src: capB, dst: capA, sport: 9, dport: 9, proto: 6}
	c.table[kIdle] = &entry{first: now.Add(-time.Minute), last: now.Add(-idleTimeout - time.Second), pkts: 3, bytes: 300}
	c.table[kFresh] = &entry{first: now, last: now, pkts: 1, bytes: 60}
	c.table[kEmpty] = &entry{first: now.Add(-time.Minute), last: now.Add(-idleTimeout - time.Second)}

	c.flush(now, false)
	if len(sink.recs) != 1 || sink.recs[0].Packets != 3 {
		t.Fatalf("idle flush exported %+v", sink.recs)
	}
	if _, ok := c.table[kIdle]; ok {
		t.Error("idle entry still in the table after its export")
	}
	if _, ok := c.table[kEmpty]; ok {
		t.Error("packet-less idle entry was kept")
	}
	if _, ok := c.table[kFresh]; !ok {
		t.Error("fresh entry was flushed early")
	}
	if c.Active() != 1 {
		t.Errorf("Active = %d", c.Active())
	}

	// a full flush exports what is left and empties the table
	c.flush(now, true)
	if len(sink.recs) != 2 || len(c.table) != 0 {
		t.Errorf("after full flush: %d records, %d entries", len(sink.recs), len(c.table))
	}
}

func TestFlushInfersOpenerFromReverse(t *testing.T) {
	now := time.Now()
	sink := &sliceSink{}
	c := &Capture{sink: sink, table: map[key]*entry{}}
	// the SYN was only seen in the B->A direction (opener 1 there);
	// the A->B entry never learned its own opener and must inherit
	// the negation: it was opened by the other side (-1)... the entry
	// in the SYN's direction keeps its own mark.
	kAB := key{src: capA, dst: capB, sport: 5000, dport: 443, proto: 6}
	kBA := key{src: capB, dst: capA, sport: 443, dport: 5000, proto: 6}
	c.table[kAB] = &entry{first: now.Add(-time.Minute), last: now.Add(-idleTimeout - time.Second), pkts: 2, bytes: 200, opener: 0}
	c.table[kBA] = &entry{first: now.Add(-time.Minute), last: now.Add(-idleTimeout - time.Second), pkts: 2, bytes: 200, opener: 1}
	c.flush(now, false)
	bySrc := map[netip.Addr]int8{}
	for _, r := range sink.recs {
		bySrc[r.Src] = r.Opener
	}
	if bySrc[capB] != 1 {
		t.Errorf("B->A opener = %d, want its own 1", bySrc[capB])
	}
	if bySrc[capA] != -1 {
		t.Errorf("A->B opener = %d, want -1 inferred from the reverse entry", bySrc[capA])
	}
}

// fakeSource feeds crafted frames once, then blocks until closed.
type fakeSource struct {
	frames [][]byte
	wires  []int
	done   chan struct{}
	once   sync.Once
}

func (f *fakeSource) Read(buf []byte, fn func([]byte, int)) error {
	if len(f.frames) > 0 {
		fr, w := f.frames[0], f.wires[0]
		f.frames, f.wires = f.frames[1:], f.wires[1:]
		fn(fr, w)
		return nil
	}
	<-f.done
	return errors.New("closed")
}

func (f *fakeSource) Close() error {
	f.once.Do(func() { close(f.done) })
	return nil
}

func (f *fakeSource) Method() string { return "fake" }
func (f *fakeSource) Drops() uint64  { return 0 }

func TestReadLoopEndToEnd(t *testing.T) {
	syn, synLen := encode.Frame(capA, capB, 5000, 443, 6, 0, 40, 0x02)
	synack, ackLen := encode.Frame(capB, capA, 443, 5000, 6, 0, 40, 0x12)
	src := &fakeSource{frames: [][]byte{syn, synack}, wires: []int{int(synLen), int(ackLen)}, done: make(chan struct{})}
	sink := &sliceSink{}
	c := &Capture{src: src, sink: sink, table: map[key]*entry{}, domain: 3}
	ctx, cancel := context.WithCancel(context.Background())
	go c.readLoop(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && c.Packets.Load() < 2 {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	src.Close()
	if got := c.Packets.Load(); got != 2 {
		t.Fatalf("Packets = %d", got)
	}
	if c.Active() != 2 {
		t.Fatalf("Active = %d, want the two directions as two entries", c.Active())
	}
	c.flush(time.Now(), true)
	if len(sink.recs) != 2 {
		t.Fatalf("flushed %d records", len(sink.recs))
	}
	bySrc := map[netip.Addr]flow.Record{}
	for _, r := range sink.recs {
		bySrc[r.Src] = r
	}
	if r := bySrc[capA]; r.Opener != 1 || r.Packets != 1 || r.Bytes != uint64(synLen)+4 {
		t.Errorf("A->B record: opener %d packets %d bytes %d", r.Opener, r.Packets, r.Bytes)
	}
	if r := bySrc[capB]; r.Opener != -1 {
		t.Errorf("B->A opener = %d, want -1 (SYN+ACK)", r.Opener)
	}
}

func TestExpireLoopFlushesOnStop(t *testing.T) {
	sink := &sliceSink{}
	c := &Capture{sink: sink, table: map[key]*entry{}}
	k := key{src: capA, dst: capB, sport: 1, dport: 2, proto: 17}
	now := time.Now()
	c.table[k] = &entry{first: now, last: now, pkts: 1, bytes: 60}
	ctx, cancel := context.WithCancel(context.Background())
	go c.expireLoop(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(sink.recs) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if len(sink.recs) != 1 {
		t.Errorf("stopping the capture flushed %d records, want the open flow", len(sink.recs))
	}
}

func TestFormatDevices(t *testing.T) {
	ds := []Device{
		{Num: 1, Name: `\Device\NPF_{1}`, Friendly: "Wi-Fi", Desc: "Intel Wi-Fi", Addrs: []string{"192.168.1.5"}},
		{Num: 2, Name: "eth0", Friendly: "eth0"},
	}
	lines := formatDevices(ds)
	if len(lines) != 3 {
		t.Fatalf("%d lines", len(lines))
	}
	if !strings.Contains(lines[0], "Name") || !strings.Contains(lines[0], "Address") {
		t.Errorf("header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "Wi-Fi") || !strings.Contains(lines[1], "192.168.1.5") || !strings.Contains(lines[1], "Intel Wi-Fi") {
		t.Errorf("row 1: %q", lines[1])
	}
	// no address shows a dash; an empty description falls back to the name
	if !strings.Contains(lines[2], "  -  ") && !strings.Contains(lines[2], "-") {
		t.Errorf("row 2: %q", lines[2])
	}
	if !strings.HasSuffix(lines[2], "eth0") {
		t.Errorf("row 2 desc fallback: %q", lines[2])
	}
}
