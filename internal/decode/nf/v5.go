package nf

import (
	"errors"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/packet"
	"github.com/githubflyideas/traffic66/internal/wire"
)

// NetFlow v5 layout: 24-byte header, 48-byte records.
const (
	v5HeaderLen = 24
	v5RecordLen = 48
)

func (d *Decoder) decodeV5(b []byte, addr netip.Addr, now time.Time) (Result, error) {
	r := wire.New(b)
	r.U16() // version
	count := r.U16()
	uptime := r.U32()
	secs := r.U32()
	nsecs := r.U32()
	seq := r.U32()
	engType := r.U8()
	engID := r.U8()
	sampling := r.U16()
	if r.Err() != nil {
		return Result{}, errors.New("nf: v5 header truncated")
	}
	if len(b) < v5HeaderLen+int(count)*v5RecordLen {
		return Result{}, errors.New("nf: v5 records truncated")
	}
	rate := uint32(sampling & 0x3FFF)
	if rate == 0 {
		rate = 1
	}
	exportMs := int64(secs)*1000 + int64(nsecs)/1e6
	at := func(up uint32) time.Time {
		return time.UnixMilli(exportMs - int64(uptime-up))
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	s := d.session(key{addr, uint32(engType)<<8 | uint32(engID), flow.SrcNetFlow5})
	s.touch(now, time.Unix(int64(secs), int64(nsecs)))
	s.seq(seq, uint32(count), uint32(count), now)
	s.def = rate
	s.stats.Records += uint64(count)

	res := Result{Records: make([]flow.Record, 0, count)}
	for i := 0; i < int(count); i++ {
		rr := wire.New(b[v5HeaderLen+i*v5RecordLen:])
		rec := flow.Record{Exporter: addr, Domain: uint32(engType)<<8 | uint32(engID), Source: flow.SrcNetFlow5, Direction: 0}
		rec.Src = rr.IPv4()
		rec.Dst = rr.IPv4()
		rr.Skip(4) // next hop
		rec.InIf = uint32(rr.U16())
		rec.OutIf = uint32(rr.U16())
		rec.Packets = uint64(rr.U32())
		rec.Bytes = uint64(rr.U32())
		first := rr.U32()
		last := rr.U32()
		rec.SrcPort = rr.U16()
		rec.DstPort = rr.U16()
		rr.U8()
		rec.TCPFlags = rr.U8()
		rec.Proto = rr.U8()
		rec.TOS = rr.U8()
		rec.SrcAS = uint32(rr.U16())
		rec.DstAS = uint32(rr.U16())
		rec.Start, rec.End = at(first), at(last)
		rec.Sampling, rec.Mult, rec.SamplingKnown = rate, float64(rate)*s.comp, true
		packet.EncapFromTuple(&rec)
		res.Records = append(res.Records, rec)
	}
	return res, nil
}
