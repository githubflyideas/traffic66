// Package sflow decodes sFlow version 5 datagrams (sflow.org, "sFlow
// Version 5", July 2004). Only the structures traffic66 uses are decoded;
// everything else is skipped by its declared length.
package sflow

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/packet"
	"github.com/githubflyideas/traffic66/internal/wire"
)

// Sample formats (enterprise 0).
const (
	fmtFlowSample       = 1
	fmtCounterSample    = 2
	fmtFlowSampleExp    = 3
	fmtCounterSampleExp = 4
	recRawHeader        = 1
	recEthernetFrame    = 2
	recIPv4Data         = 3
	recIPv6Data         = 4
	recExtSwitch        = 1001
	recExtRouter        = 1002
	recExtGateway       = 1003
	ctrGenericInterface = 1
	headerProtoEthernet = 1
	headerProtoIPv4     = 11
	headerProtoIPv6     = 12
)

// FlowSample is one decoded flow sample with the fields needed to measure
// the effective sampling rate.
type FlowSample struct {
	SourceID uint32 // type<<24 | index
	Seq      uint32
	Rate     uint32
	Pool     uint32
	Drops    uint32
	Record   flow.Record
	Decoded  bool // a packet header was found and parsed
}

// Datagram is one decoded sFlow datagram.
type Datagram struct {
	Agent    netip.Addr
	SubAgent uint32
	Seq      uint32
	Uptime   uint32
	Flows    []FlowSample
	Counters []flow.IfCounters
}

var ErrVersion = errors.New("sflow: not version 5")

// Decode parses one UDP payload. now stamps records and counters, since
// sFlow carries no wall-clock time.
func Decode(b []byte, now time.Time) (*Datagram, error) {
	r := wire.New(b)
	if r.U32() != 5 {
		return nil, ErrVersion
	}
	d := &Datagram{}
	switch r.U32() {
	case 1:
		d.Agent = r.IPv4()
	case 2:
		d.Agent = r.IPv6()
	default:
		return nil, errors.New("sflow: bad agent address type")
	}
	d.SubAgent = r.U32()
	d.Seq = r.U32()
	d.Uptime = r.U32()
	n := r.U32()
	if r.Err() != nil {
		return nil, r.Err()
	}
	if n > 4096 {
		return nil, fmt.Errorf("sflow: implausible sample count %d", n)
	}
	for i := uint32(0); i < n; i++ {
		format := r.U32()
		length := r.U32()
		body := r.Bytes(int(length))
		if r.Err() != nil {
			return d, fmt.Errorf("sflow: sample %d truncated", i)
		}
		if format>>12 != 0 { // vendor enterprise
			continue
		}
		var err error
		switch format & 0xFFF {
		case fmtFlowSample:
			err = d.flowSample(body, false, now)
		case fmtFlowSampleExp:
			err = d.flowSample(body, true, now)
		case fmtCounterSample:
			err = d.counterSample(body, false, now)
		case fmtCounterSampleExp:
			err = d.counterSample(body, true, now)
		}
		if err != nil {
			return d, err
		}
	}
	return d, nil
}

func (d *Datagram) flowSample(b []byte, expanded bool, now time.Time) error {
	r := wire.New(b)
	s := FlowSample{}
	s.Seq = r.U32()
	if expanded {
		typ := r.U32()
		idx := r.U32()
		s.SourceID = typ<<24 | idx&0xFFFFFF
	} else {
		s.SourceID = r.U32()
	}
	s.Rate = r.U32()
	s.Pool = r.U32()
	s.Drops = r.U32()
	var in, out uint32
	if expanded {
		r.U32() // input format
		in = r.U32()
		r.U32() // output format
		out = r.U32()
	} else {
		in = r.U32() & 0x3FFFFFFF
		out = r.U32() & 0x3FFFFFFF
	}
	nrec := r.U32()
	if r.Err() != nil {
		return errors.New("sflow: flow sample truncated")
	}
	rec := &s.Record
	rec.Start, rec.End = now, now
	rec.InIf, rec.OutIf = in, out
	rec.Packets = 1
	rec.Sampling = s.Rate
	rec.SamplingKnown = s.Rate > 0
	rec.Mult = float64(s.Rate)
	rec.Exporter = d.Agent
	rec.Domain = d.SubAgent
	rec.Source = flow.SrcSFlow
	rec.Observation = s.SourceID
	rec.Direction = 255
	for i := uint32(0); i < nrec && r.Err() == nil; i++ {
		format := r.U32()
		length := r.U32()
		body := r.Bytes(int(length))
		if r.Err() != nil {
			return errors.New("sflow: flow record truncated")
		}
		if format>>12 != 0 {
			continue
		}
		switch format & 0xFFF {
		case recRawHeader:
			if rawHeader(body, rec) {
				s.Decoded = true
			}
		case recIPv4Data, recIPv6Data:
			if !s.Decoded && ipData(body, format&0xFFF == recIPv6Data, rec) {
				s.Decoded = true
			}
		case recExtSwitch:
			er := wire.New(body)
			srcVLAN := er.U32()
			if er.Err() == nil && rec.VLAN == 0 {
				rec.VLAN = uint16(srcVLAN & 0x0FFF)
			}
		case recExtGateway:
			gatewayAS(body, rec)
		}
	}
	d.Flows = append(d.Flows, s)
	return nil
}

func rawHeader(b []byte, rec *flow.Record) bool {
	r := wire.New(b)
	proto := r.U32()
	frameLen := r.U32()
	stripped := r.U32()
	hlen := r.U32()
	hdr := r.Bytes(int(hlen))
	if r.Err() != nil {
		return false
	}
	var ok bool
	switch proto {
	case headerProtoEthernet:
		ok = packet.Ethernet(hdr, rec)
		// For a layer-2 header, frame_length counts every octet received,
		// FCS included, so it matches interface octet counters directly.
		rec.Bytes = uint64(frameLen)
		rec.L2 = true
	case headerProtoIPv4:
		ok = packet.IPv4(hdr, rec)
		rec.Bytes = uint64(frameLen)
	case headerProtoIPv6:
		ok = packet.IPv6(hdr, rec)
		rec.Bytes = uint64(frameLen)
	default:
		rec.Bytes = uint64(frameLen)
	}
	_ = stripped
	return ok
}

func ipData(b []byte, v6 bool, rec *flow.Record) bool {
	r := wire.New(b)
	length := r.U32()
	proto := r.U32()
	var src, dst netip.Addr
	if v6 {
		src, dst = r.IPv6(), r.IPv6()
	} else {
		src, dst = r.IPv4(), r.IPv4()
	}
	sp := r.U32()
	dp := r.U32()
	flags := r.U32()
	tos := r.U32()
	if r.Err() != nil {
		return false
	}
	rec.Bytes = uint64(length)
	rec.Proto = uint8(proto)
	rec.Src, rec.Dst = src, dst
	rec.SrcPort, rec.DstPort = uint16(sp), uint16(dp)
	rec.TCPFlags = uint8(flags)
	rec.TOS = uint8(tos)
	packet.EncapFromTuple(rec)
	return true
}

func gatewayAS(b []byte, rec *flow.Record) {
	r := wire.New(b)
	switch r.U32() { // next hop address type
	case 1:
		r.Skip(4)
	case 2:
		r.Skip(16)
	default:
		return
	}
	as := r.U32()
	srcAS := r.U32()
	srcPeerAS := r.U32()
	_ = srcPeerAS
	if r.Err() != nil {
		return
	}
	if rec.SrcAS == 0 {
		rec.SrcAS = srcAS
	}
	// dst AS: last AS of the first path segment, when present.
	nseg := r.U32()
	var last uint32
	for i := uint32(0); i < nseg && i < 16 && r.Err() == nil; i++ {
		r.U32() // segment type
		n := r.U32()
		for j := uint32(0); j < n && j < 64; j++ {
			last = r.U32()
		}
	}
	if last != 0 {
		rec.DstAS = last
	} else if as != 0 {
		rec.DstAS = as
	}
}

func (d *Datagram) counterSample(b []byte, expanded bool, now time.Time) error {
	r := wire.New(b)
	r.U32() // sequence
	if expanded {
		r.U32()
		r.U32()
	} else {
		r.U32()
	}
	n := r.U32()
	for i := uint32(0); i < n && r.Err() == nil; i++ {
		format := r.U32()
		length := r.U32()
		body := r.Bytes(int(length))
		if r.Err() != nil {
			return errors.New("sflow: counter record truncated")
		}
		if format == ctrGenericInterface {
			c := wire.New(body)
			ifIndex := c.U32()
			c.U32() // ifType
			speed := c.U64()
			c.U32() // direction
			c.U32() // status
			inOct := c.U64()
			inU, inM, inB := c.U32(), c.U32(), c.U32()
			c.U32() // discards
			c.U32() // errors
			c.U32() // unknown protos
			outOct := c.U64()
			outU, outM, outB := c.U32(), c.U32(), c.U32()
			if c.Err() != nil {
				continue
			}
			d.Counters = append(d.Counters, flow.IfCounters{
				Time: now, Exporter: d.Agent, IfIndex: ifIndex, Speed: speed,
				InOctets: inOct, OutOctets: outOct,
				InPkts:  uint64(inU) + uint64(inM) + uint64(inB),
				OutPkts: uint64(outU) + uint64(outM) + uint64(outB),
			})
		}
	}
	return nil
}
