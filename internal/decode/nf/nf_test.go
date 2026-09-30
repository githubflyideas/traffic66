package nf

import (
	"net/netip"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/encode"
	"github.com/githubflyideas/traffic66/internal/flow"
)

var (
	exp  = netip.MustParseAddr("10.0.0.1")
	cli  = netip.MustParseAddr("10.10.3.27")
	srv  = netip.MustParseAddr("203.0.113.66")
	base = time.Unix(1700000000, 0)
)

var v9Data = encode.Template{ID: 256, Fields: []encode.Field{
	{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
	{ID: ieProtocol, Len: 1}, {ID: ieOctetDelta, Len: 4}, {ID: iePacketDelta, Len: 4},
	{ID: ieStartSysUp, Len: 4}, {ID: ieEndSysUp, Len: 4}, {ID: ieInIf, Len: 2}, {ID: ieSamplerID, Len: 1},
}}

var v9Opt = encode.Template{ID: 257,
	Scope:  []encode.Field{{ID: nf9ScopeSystem, Len: 4}},
	Fields: []encode.Field{{ID: ieSamplerID, Len: 1}, {ID: ieSamplerRandom, Len: 4}}}

func v9Rec(bytes uint32) encode.Values {
	return encode.Values{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2), encode.U(6, 1),
		encode.U(uint64(bytes), 4), encode.U(10, 4), encode.U(40000, 4), encode.U(59000, 4), encode.U(3, 2), encode.U(1, 1)}
}

func TestV9DataBeforeTemplateAndLateSampling(t *testing.T) {
	d := NewDecoder()
	// 1. data arrives before any template: held.
	m := encode.NewV9()
	m.AddData(256, []encode.Values{v9Rec(1500)})
	res, err := d.Decode(m.V9(60000, uint32(base.Unix()), 1, 0), exp, base)
	if err != nil || len(res.Records) != 0 {
		t.Fatalf("expected held, got %v %v", res.Records, err)
	}
	// 2. template arrives: held data decodes, but waits for sampling.
	m = encode.NewV9()
	m.AddTemplate(v9Data)
	res, _ = d.Decode(m.V9(61000, uint32(base.Unix())+1, 2, 0), exp, base)
	if len(res.Records) != 0 {
		t.Fatalf("expected pending, got %d", len(res.Records))
	}
	st := d.Stats()[0]
	if st.Pending != 1 || st.SamplingState != "waiting" || st.NoTemplate != 1 {
		t.Fatalf("stats %+v", st)
	}
	// 3. options template + data declare sampler 1 = 1:1000: record released.
	m = encode.NewV9()
	m.AddTemplate(v9Opt)
	m.AddData(257, []encode.Values{{encode.U(0, 4), encode.U(1, 1), encode.U(1000, 4)}})
	res, _ = d.Decode(m.V9(62000, uint32(base.Unix())+2, 3, 0), exp, base)
	if len(res.Records) != 1 {
		t.Fatalf("expected release, got %d", len(res.Records))
	}
	r := res.Records[0]
	if r.Mult != 1000 || !r.SamplingKnown || r.Bytes != 1500 || r.Src != cli || r.DstPort != 443 || r.InIf != 3 {
		t.Fatalf("record %+v", r)
	}
	// Start/End from sysUptime: export at 60000ms uptime => end 1s before export.
	if want := base.Add(-1 * time.Second); !r.End.Equal(want) {
		t.Fatalf("end %v want %v", r.End, want)
	}
	if want := base.Add(-20 * time.Second); !r.Start.Equal(want) {
		t.Fatalf("start %v want %v", r.Start, want)
	}
	// 4. sequence gap: packet 5 after 3 => 1 lost packet.
	m = encode.NewV9()
	m.AddData(256, []encode.Values{v9Rec(100)})
	res, _ = d.Decode(m.V9(63000, uint32(base.Unix())+3, 5, 0), exp, base)
	if len(res.Records) != 1 || res.Records[0].Mult != 1000 {
		t.Fatalf("got %+v", res.Records)
	}
	if st := d.Stats()[0]; st.LostPackets != 1 {
		t.Fatalf("lost %+v", st)
	}
}

func TestHoldTimeoutAssumesUnsampled(t *testing.T) {
	d := NewDecoder()
	m := encode.NewV9()
	m.AddTemplate(v9Data)
	m.AddData(256, []encode.Values{v9Rec(700)})
	d.Decode(m.V9(60000, uint32(base.Unix()), 1, 0), exp, base)
	res := d.Flush(base.Add(10 * time.Minute))
	if len(res.Records) != 1 || res.Records[0].Mult != 1 || res.Records[0].SamplingKnown {
		t.Fatalf("got %+v", res.Records)
	}
	if st := d.Stats()[0]; st.SamplingState != "assumed1" {
		t.Fatalf("state %s", st.SamplingState)
	}
}

func TestIPFIXVarLenBiflowSampling(t *testing.T) {
	d := NewDecoder()
	data := encode.Template{ID: 300, Fields: []encode.Field{
		{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
		{ID: ieProtocol, Len: 1}, {ID: ieOctetDelta, Len: 8}, {ID: iePacketDelta, Len: 8},
		{ID: ieOctetDelta, Len: 8, Ent: penReverse}, {ID: iePacketDelta, Len: 8, Ent: penReverse},
		{ID: ieStartMs, Len: 8}, {ID: ieEndMs, Len: 8}, {ID: 82, Len: 0xFFFF}, // interfaceName, variable
	}}
	opt := encode.Template{ID: 301,
		Scope:  []encode.Field{{ID: ieObsDomainID, Len: 4}},
		Fields: []encode.Field{{ID: ieSamplingPktIntvl, Len: 4}, {ID: ieSamplingPktSpace, Len: 4}}}
	m := encode.NewIPFIX()
	m.AddTemplate(data)
	m.AddTemplate(opt)
	m.AddData(301, []encode.Values{{encode.U(7, 4), encode.U(1, 4), encode.U(99, 4)}})
	ms := uint64(base.UnixMilli())
	m.AddData(300, []encode.Values{{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2), encode.U(6, 1),
		encode.U(2000, 8), encode.U(20, 8), encode.U(90000, 8), encode.U(70, 8), encode.U(ms-30000, 8), encode.U(ms, 8),
		append([]byte{5}, "port1"...)}})
	res, err := d.Decode(m.IPFIX(uint32(base.Unix()), 0, 7), exp, base)
	if err != nil || len(res.Records) != 2 {
		t.Fatalf("got %d %v", len(res.Records), err)
	}
	fwd, rev := res.Records[0], res.Records[1]
	if fwd.Mult != 100 || fwd.Bytes != 2000 || fwd.Src != cli || fwd.Source != flow.SrcIPFIX || fwd.Domain != 7 {
		t.Fatalf("fwd %+v", fwd)
	}
	if rev.Bytes != 90000 || rev.Src != srv || rev.SrcPort != 443 || rev.DstPort != 50522 {
		t.Fatalf("rev %+v", rev)
	}
	if !fwd.End.Equal(base) || !fwd.Start.Equal(base.Add(-30*time.Second)) {
		t.Fatalf("times %v %v", fwd.Start, fwd.End)
	}
	// sequence: 2 data records sent so far; next message claims seq 5 => 3 lost.
	m = encode.NewIPFIX()
	m.AddData(301, []encode.Values{{encode.U(7, 4), encode.U(1, 4), encode.U(99, 4)}})
	d.Decode(m.IPFIX(uint32(base.Unix()), 5, 7), exp, base)
	if st := d.Stats()[0]; st.LostRecords != 3 {
		t.Fatalf("lost %+v", st)
	}
}

func TestV5(t *testing.T) {
	d := NewDecoder()
	pkt := encode.V5(100000, uint32(base.Unix()), 10, 512, []encode.V5Record{{Src: cli, Dst: srv, Packets: 3, Bytes: 4500, First: 70000, Last: 99000, SrcPort: 1, DstPort: 443, Proto: 6}})
	res, err := d.Decode(pkt, exp, base)
	if err != nil || len(res.Records) != 1 {
		t.Fatal(err)
	}
	r := res.Records[0]
	if r.Mult != 512 || r.Bytes != 4500 || !r.End.Equal(base.Add(-time.Second)) || !r.Start.Equal(base.Add(-30*time.Second)) {
		t.Fatalf("%+v", r)
	}
	// next seq should be 11; 14 means 3 records lost.
	d.Decode(encode.V5(100000, uint32(base.Unix()), 14, 512, nil), exp, base)
	if st := d.Stats()[0]; st.LostRecords != 3 {
		t.Fatalf("%+v", st)
	}
}

func FuzzDecode(f *testing.F) {
	m := encode.NewIPFIX()
	m.AddTemplate(encode.Template{ID: 300, Fields: []encode.Field{{ID: 8, Len: 4}, {ID: 82, Len: 0xFFFF}}})
	m.AddData(300, []encode.Values{{encode.A(cli), []byte{2, 'a', 'b'}}})
	f.Add(m.IPFIX(1, 0, 1))
	v := encode.NewV9()
	v.AddTemplate(v9Data)
	v.AddData(256, []encode.Values{v9Rec(1)})
	f.Add(v.V9(1, 1, 1, 0))
	f.Fuzz(func(t *testing.T, b []byte) {
		d := NewDecoder()
		d.Decode(b, exp, base)
		d.Decode(b, exp, base)
	})
}
