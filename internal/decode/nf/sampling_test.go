package nf

import (
	"encoding/binary"
	"math"
	"net/netip"
	"testing"

	"github.com/githubflyideas/traffic66/internal/encode"
)

// a minimal IPFIX data template: addresses, ports, interfaces, counts
var ipfixFlowTmpl = encode.Template{ID: 400, Fields: []encode.Field{
	{ID: ieSrcIPv4, Len: 4}, {ID: ieDstIPv4, Len: 4}, {ID: ieSrcPort, Len: 2}, {ID: ieDstPort, Len: 2},
	{ID: ieProtocol, Len: 1}, {ID: ieInIf, Len: 4}, {ID: ieOutIf, Len: 4}, {ID: ieOctetDelta, Len: 8}, {ID: iePacketDelta, Len: 8},
}}

func ipfixFlow(in, out uint32) encode.Values {
	return encode.Values{encode.A(cli), encode.A(srv), encode.U(50522, 2), encode.U(443, 2), encode.U(6, 1),
		encode.U(uint64(in), 4), encode.U(uint64(out), 4), encode.U(1500, 8), encode.U(1, 8)}
}

func f64(v float64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(v))
	return b
}

// decodeOne sends options (if any) and one data record, returns its multiplier
func decodeOne(t *testing.T, d *Decoder, domain uint32, opt *encode.Template, optVals encode.Values, rec encode.Values) float64 {
	t.Helper()
	m := encode.NewIPFIX()
	m.AddTemplate(ipfixFlowTmpl)
	if opt != nil {
		m.AddTemplate(*opt)
		m.AddData(opt.ID, []encode.Values{optVals})
	}
	m.AddData(400, []encode.Values{rec})
	res, err := d.Decode(m.IPFIX(uint32(base.Unix()), 0, domain), exp, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Records) != 1 {
		return 0 // held, waiting for a rate
	}
	return res.Records[0].Mult
}

func TestSamplingForms(t *testing.T) {
	// samplingProbability (311) as a float
	opt := encode.Template{ID: 401, Scope: []encode.Field{{ID: ieObsDomainID, Len: 4}}, Fields: []encode.Field{{ID: ieSamplingProb, Len: 8}}}
	if m := decodeOne(t, NewDecoder(), 1, &opt, encode.Values{encode.U(1, 4), f64(0.001)}, ipfixFlow(1, 2)); m != 1000 {
		t.Fatalf("probability: mult %v", m)
	}
	// rate per interface, configured on the egress interface of the flow
	opt = encode.Template{ID: 402, Scope: []encode.Field{{ID: ieInIf, Len: 4}}, Fields: []encode.Field{{ID: ieSamplingInterval, Len: 4}}}
	if m := decodeOne(t, NewDecoder(), 1, &opt, encode.Values{encode.U(5, 4), encode.U(1000, 4)}, ipfixFlow(7, 5)); m != 1000 {
		t.Fatalf("egress interface: mult %v", m)
	}
	// a sampler declared with an id the records never name: the only rate
	opt = encode.Template{ID: 403, Scope: []encode.Field{{ID: ieObsDomainID, Len: 4}},
		Fields: []encode.Field{{ID: ieSelectorID, Len: 4}, {ID: ieSamplingPktIntvl, Len: 4}, {ID: ieSamplingPktSpace, Len: 4}}}
	if m := decodeOne(t, NewDecoder(), 1, &opt, encode.Values{encode.U(1, 4), encode.U(9, 4), encode.U(1, 4), encode.U(511, 4)}, ipfixFlow(1, 2)); m != 512 {
		t.Fatalf("only sampler: mult %v", m)
	}
	// a zero samplingInterval must not hide samplerRandomInterval
	opt = encode.Template{ID: 404, Scope: []encode.Field{{ID: ieObsDomainID, Len: 4}},
		Fields: []encode.Field{{ID: ieSamplerRandom, Len: 4}, {ID: ieSamplingInterval, Len: 4}}}
	if m := decodeOne(t, NewDecoder(), 1, &opt, encode.Values{encode.U(1, 4), encode.U(100, 4), encode.U(0, 4)}, ipfixFlow(1, 2)); m != 100 {
		t.Fatalf("zero interval: mult %v", m)
	}
}

func TestSamplingOtherDomainAndManual(t *testing.T) {
	// the rate is declared in one observation domain, the flows come in another
	d := NewDecoder()
	opt := encode.Template{ID: 405, Scope: []encode.Field{{ID: ieObsDomainID, Len: 4}}, Fields: []encode.Field{{ID: ieSamplingInterval, Len: 4}}}
	m := encode.NewIPFIX()
	m.AddTemplate(opt)
	m.AddData(405, []encode.Values{{encode.U(9, 4), encode.U(2000, 4)}})
	d.Decode(m.IPFIX(uint32(base.Unix()), 0, 9), exp, base)
	if got := decodeOne(t, d, 10, nil, nil, ipfixFlow(1, 2)); got != 2000 {
		t.Fatalf("other domain: mult %v", got)
	}

	// nothing declared: held; a rate set by hand releases it
	d = NewDecoder()
	if got := decodeOne(t, d, 1, nil, nil, ipfixFlow(1, 2)); got != 0 {
		t.Fatalf("expected held, mult %v", got)
	}
	res := d.SetManual(map[netip.Addr]uint32{exp: 1000})
	if len(res.Records) != 1 || res.Records[0].Mult != 1000 {
		t.Fatalf("manual: %+v", res.Records)
	}
	st := d.Stats()[0]
	if st.SamplingState != "manual" || st.Sampling != 1000 || len(st.TemplateInfo) != 1 {
		t.Fatalf("stats %+v", st)
	}
	// a rate the device declares wins over the hand-set one
	if got := decodeOne(t, d, 1, &opt, encode.Values{encode.U(1, 4), encode.U(500, 4)}, ipfixFlow(1, 2)); got != 500 {
		t.Fatalf("declared over manual: mult %v", got)
	}
}
