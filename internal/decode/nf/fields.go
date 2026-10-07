package nf

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/packet"
	"github.com/githubflyideas/traffic66/internal/wire"
)

// Information element numbers (IANA IPFIX registry; NetFlow v9 uses the
// same numbers for the shared range).
const (
	ieOctetDelta       = 1
	iePacketDelta      = 2
	ieProtocol         = 4
	ieTOS              = 5
	ieTCPFlags         = 6
	ieSrcPort          = 7
	ieSrcIPv4          = 8
	ieInIf             = 10
	ieDstPort          = 11
	ieDstIPv4          = 12
	ieOutIf            = 14
	ieSrcAS            = 16
	ieDstAS            = 17
	ieEndSysUp         = 21
	ieStartSysUp       = 22
	iePostOctetDelta   = 23
	iePostPacketDelta  = 24
	ieSrcIPv6          = 27
	ieDstIPv6          = 28
	ieICMPv4           = 32
	ieSamplingInterval = 34
	ieSamplerID        = 48
	ieSamplerRandom    = 50
	ieVLAN             = 58
	ieDirection        = 61
	ieOctetTotal       = 85
	iePacketTotal      = 86
	ieICMPv6           = 139
	ieExportingProcID  = 144
	ieObsDomainID      = 149
	ieStartSec         = 150
	ieEndSec           = 151
	ieStartMs          = 152
	ieEndMs            = 153
	ieStartUs          = 154
	ieEndUs            = 155
	ieStartNs          = 156
	ieEndNs            = 157
	ieStartDeltaUs     = 158
	ieEndDeltaUs       = 159
	ieSysInitMs        = 160
	ieInitiatorOctets  = 231
	ieResponderOctets  = 232
	ieDot1qVLAN        = 243
	ieInitiatorPackets = 298
	ieResponderPackets = 299
	ieSelectorID       = 302
	ieSamplingPktIntvl = 305
	ieSamplingPktSpace = 306
	ieSamplingSize     = 309
	ieSamplingPop      = 310
	ieSamplingProb     = 311 // samplingProbability, float64

	// NetFlow v9 option scope types (RFC 3954 section 6.1).
	nf9ScopeSystem    = 1
	nf9ScopeInterface = 2

	// RFC 5103 reverse information elements.
	penReverse = 29305

	ntpEpochOffset = 2208988800
)

type msgHeader struct {
	src        flow.Source
	exportSecs uint32
	uptime     uint32 // v9 only
}

type acc struct {
	rec flow.Record

	bytes, pkts         uint64
	haveDelta           bool
	totBytes, totPkts   uint64
	postBytes, postPkts uint64
	revBytes, revPkts   uint64
	initB, respB        uint64
	initP, respP        uint64

	startMs, endMs     int64
	startUp, endUp     uint32
	hasStartUp, hasEnd bool
	startDeltaUs       uint64
	endDeltaUs         uint64
	hasDeltaUs         bool

	interval, pktInterval, pktSpace uint32
	sampSize, sampPop               uint32
	prob                            float64
	samplerID                       uint32
	hasSampler                      bool

	// option scope
	scopeIf    uint32
	hasScopeIf bool
	sysInitMs  int64
}

func ntpToMs(v uint64) int64 {
	sec := int64(v>>32) - ntpEpochOffset
	frac := int64(v&0xFFFFFFFF) * 1000 >> 32
	return sec*1000 + frac
}

func (a *acc) set(f fieldSpec, v []byte, nf9Scope bool) {
	if f.scope && nf9Scope {
		switch f.id {
		case nf9ScopeInterface:
			a.scopeIf, a.hasScopeIf = uint32(wire.Uint(v)), true
		}
		return
	}
	if f.ent == penReverse {
		switch f.id {
		case ieOctetDelta:
			a.revBytes = wire.Uint(v)
		case iePacketDelta:
			a.revPkts = wire.Uint(v)
		}
		return
	}
	if f.ent != 0 {
		return
	}
	r := &a.rec
	switch f.id {
	case ieOctetDelta:
		a.bytes, a.haveDelta = wire.Uint(v), true
	case iePacketDelta:
		a.pkts = wire.Uint(v)
	case ieOctetTotal:
		a.totBytes = wire.Uint(v)
	case iePacketTotal:
		a.totPkts = wire.Uint(v)
	case iePostOctetDelta:
		a.postBytes = wire.Uint(v)
	case iePostPacketDelta:
		a.postPkts = wire.Uint(v)
	case ieInitiatorOctets:
		a.initB = wire.Uint(v)
	case ieResponderOctets:
		a.respB = wire.Uint(v)
	case ieInitiatorPackets:
		a.initP = wire.Uint(v)
	case ieResponderPackets:
		a.respP = wire.Uint(v)
	case ieProtocol:
		r.Proto = uint8(wire.Uint(v))
	case ieTOS:
		r.TOS = uint8(wire.Uint(v))
	case ieTCPFlags:
		r.TCPFlags = uint8(wire.Uint(v))
	case ieSrcPort:
		r.SrcPort = uint16(wire.Uint(v))
	case ieDstPort:
		r.DstPort = uint16(wire.Uint(v))
	case ieSrcIPv4:
		if len(v) == 4 {
			r.Src = netip.AddrFrom4([4]byte(v))
		}
	case ieDstIPv4:
		if len(v) == 4 {
			r.Dst = netip.AddrFrom4([4]byte(v))
		}
	case ieSrcIPv6:
		if len(v) == 16 {
			r.Src = netip.AddrFrom16([16]byte(v))
		}
	case ieDstIPv6:
		if len(v) == 16 {
			r.Dst = netip.AddrFrom16([16]byte(v))
		}
	case ieInIf:
		r.InIf = uint32(wire.Uint(v))
		if f.scope {
			a.scopeIf, a.hasScopeIf = r.InIf, true
		}
	case ieOutIf:
		r.OutIf = uint32(wire.Uint(v))
	case ieSrcAS:
		r.SrcAS = uint32(wire.Uint(v))
	case ieDstAS:
		r.DstAS = uint32(wire.Uint(v))
	case ieVLAN, ieDot1qVLAN:
		if r.VLAN == 0 {
			r.VLAN = uint16(wire.Uint(v)) & 0x0FFF
		}
	case ieDirection:
		r.Direction = uint8(wire.Uint(v))
	case ieICMPv4, ieICMPv6:
		if r.DstPort == 0 {
			r.DstPort = uint16(wire.Uint(v))
		}
	case ieStartSysUp:
		a.startUp, a.hasStartUp = uint32(wire.Uint(v)), true
	case ieEndSysUp:
		a.endUp, a.hasEnd = uint32(wire.Uint(v)), true
	case ieStartSec:
		a.startMs = int64(wire.Uint(v)) * 1000
	case ieEndSec:
		a.endMs = int64(wire.Uint(v)) * 1000
	case ieStartMs:
		a.startMs = int64(wire.Uint(v))
	case ieEndMs:
		a.endMs = int64(wire.Uint(v))
	case ieStartUs, ieStartNs:
		a.startMs = ntpToMs(wire.Uint(v))
	case ieEndUs, ieEndNs:
		a.endMs = ntpToMs(wire.Uint(v))
	case ieStartDeltaUs:
		a.startDeltaUs, a.hasDeltaUs = wire.Uint(v), true
	case ieEndDeltaUs:
		a.endDeltaUs, a.hasDeltaUs = wire.Uint(v), true
	case ieSysInitMs:
		a.sysInitMs = int64(wire.Uint(v))
	case ieSamplingInterval, ieSamplerRandom:
		// both may be present; a zero in one must not hide the other
		if x := uint32(wire.Uint(v)); x > 0 {
			a.interval = x
		}
	case ieSamplingProb:
		if len(v) == 8 {
			if p := math.Float64frombits(binary.BigEndian.Uint64(v)); p > 0 && p <= 1 {
				a.prob = p
			}
		}
	case ieSamplingPktIntvl:
		a.pktInterval = uint32(wire.Uint(v))
	case ieSamplingPktSpace:
		a.pktSpace = uint32(wire.Uint(v))
	case ieSamplingSize:
		a.sampSize = uint32(wire.Uint(v))
	case ieSamplingPop:
		a.sampPop = uint32(wire.Uint(v))
	case ieSamplerID, ieSelectorID:
		a.samplerID, a.hasSampler = uint32(wire.Uint(v)), true
	}
}

// rate derives a 1-in-N rate from whichever sampling fields were present.
func (a *acc) rate() uint32 {
	switch {
	case a.pktInterval > 0:
		return (a.pktInterval + a.pktSpace) / a.pktInterval
	case a.sampSize > 0 && a.sampPop > 0:
		return a.sampPop / a.sampSize
	case a.interval > 0:
		return a.interval
	case a.prob > 0:
		return uint32(math.Round(1 / a.prob))
	}
	return 0
}

// times fills Start/End from whichever timestamp fields were present.
func (a *acc) times(h msgHeader, sysInitMs int64) {
	exportMs := int64(h.exportSecs) * 1000
	r := &a.rec
	switch {
	case a.endMs > 0:
		r.End = time.UnixMilli(a.endMs)
		if a.startMs > 0 {
			r.Start = time.UnixMilli(a.startMs)
		} else {
			r.Start = r.End
		}
	case a.hasDeltaUs:
		r.End = time.UnixMilli(exportMs - int64(a.endDeltaUs/1000))
		r.Start = time.UnixMilli(exportMs - int64(a.startDeltaUs/1000))
	case a.hasEnd && h.src == flow.SrcNetFlow9:
		r.End = time.UnixMilli(exportMs - int64(h.uptime-a.endUp))
		if a.hasStartUp {
			r.Start = time.UnixMilli(exportMs - int64(h.uptime-a.startUp))
		} else {
			r.Start = r.End
		}
	case a.hasEnd && sysInitMs > 0:
		r.End = time.UnixMilli(sysInitMs + int64(a.endUp))
		if a.hasStartUp {
			r.Start = time.UnixMilli(sysInitMs + int64(a.startUp))
		} else {
			r.Start = r.End
		}
	case a.hasEnd && a.hasStartUp:
		r.End = time.UnixMilli(exportMs)
		r.Start = r.End.Add(-time.Duration(a.endUp-a.startUp) * time.Millisecond)
	default:
		r.End = time.UnixMilli(exportMs)
		r.Start = r.End
	}
	if r.Start.After(r.End) {
		r.Start = r.End
	}
}

// parseTemplateSet reads (options) template records into s.
func parseTemplateSet(s *session, b []byte, src flow.Source, options bool) []uint16 {
	var ids []uint16
	r := wire.New(b)
	for r.Len() >= 4 {
		id := r.U16()
		if src == flow.SrcNetFlow9 && options {
			scopeLen := r.U16()
			optLen := r.U16()
			if r.Err() != nil || id < 256 {
				break
			}
			t := &template{options: true, nf9Scope: true}
			for i := 0; i < int(scopeLen)/4; i++ {
				t.fields = append(t.fields, fieldSpec{id: r.U16(), len: r.U16(), scope: true})
			}
			for i := 0; i < int(optLen)/4; i++ {
				t.fields = append(t.fields, fieldSpec{id: r.U16(), len: r.U16()})
			}
			if r.Err() != nil {
				break
			}
			s.templates[id] = t
			s.describe(id, t)
			ids = append(ids, id)
			// v9 options template records are padded to 4 bytes per set,
			// not per record; one record per set is the common case.
			if r.Len() < 4 {
				break
			}
			continue
		}
		count := r.U16()
		if r.Err() != nil {
			break
		}
		if id < 256 {
			break
		}
		if count == 0 { // IPFIX template withdrawal
			delete(s.templates, id)
			continue
		}
		scopeCount := uint16(0)
		if options {
			scopeCount = r.U16()
		}
		t := &template{options: options}
		for i := 0; i < int(count); i++ {
			fid := r.U16()
			flen := r.U16()
			var ent uint32
			if src == flow.SrcIPFIX && fid&0x8000 != 0 {
				fid &= 0x7FFF
				ent = r.U32()
			}
			t.fields = append(t.fields, fieldSpec{id: fid, ent: ent, len: flen, scope: options && i < int(scopeCount)})
		}
		if r.Err() != nil {
			break
		}
		s.templates[id] = t
		s.describe(id, t)
		ids = append(ids, id)
	}
	return ids
}

// decodeData decodes every record in one data set.
func (d *Decoder) decodeData(s *session, t *template, body []byte, h msgHeader, addr netip.Addr, domain uint32, now time.Time, out *[]flow.Record) int {
	minLen := 0
	for _, f := range t.fields {
		if f.len == 0xFFFF {
			minLen++
		} else {
			minLen += int(f.len)
		}
	}
	if minLen == 0 {
		return 0
	}
	n := 0
	sampleChanged := false
	for len(body) >= minLen {
		a := acc{}
		a.rec.Direction = 255
		ok := true
		for _, f := range t.fields {
			l := int(f.len)
			if f.len == 0xFFFF {
				if len(body) < 1 {
					ok = false
					break
				}
				l = int(body[0])
				body = body[1:]
				if l == 255 {
					if len(body) < 2 {
						ok = false
						break
					}
					l = int(binary.BigEndian.Uint16(body))
					body = body[2:]
				}
			}
			if len(body) < l {
				ok = false
				break
			}
			a.set(f, body[:l], t.nf9Scope)
			body = body[l:]
		}
		if !ok {
			break
		}
		n++
		if t.options {
			if a.sysInitMs > 0 {
				s.sysInitMs(a.sysInitMs)
			}
			rate := a.rate()
			if rate == 0 {
				continue
			}
			switch {
			case a.hasSampler:
				s.bySampler[a.samplerID] = rate
			case a.hasScopeIf:
				s.byIf[a.scopeIf] = rate
			default:
				s.def = rate
			}
			d.byAddr[addr] = rate
			sampleChanged = true
			continue
		}
		d.emit(s, &a, h, addr, domain, now, out)
	}
	if sampleChanged {
		d.retryPending(s, out)
	}
	return n
}

func (s *session) sysInitMs(v int64) { s.initMs = v }

func (d *Decoder) emit(s *session, a *acc, h msgHeader, addr netip.Addr, domain uint32, now time.Time, out *[]flow.Record) {
	r := &a.rec
	if !r.Src.IsValid() || !r.Dst.IsValid() {
		return
	}
	r.Exporter, r.Domain, r.Source = addr, domain, h.src
	a.times(h, s.initMs)
	packet.EncapFromTuple(r)

	switch {
	case a.haveDelta:
		r.Bytes, r.Packets = a.bytes, a.pkts
	case a.initB > 0 || a.respB > 0:
		r.Bytes, r.Packets = a.initB, a.initP
		a.revBytes, a.revPkts = a.respB, a.respP
	case a.postBytes > 0:
		r.Bytes, r.Packets = a.postBytes, a.postPkts
	default:
		r.Bytes, r.Packets = a.totBytes, a.totPkts
	}
	ref := samplerRef{samplerID: a.samplerID, hasSampler: a.hasSampler, inIf: r.InIf, outIf: r.OutIf}
	inRec := a.rate()
	s.stats.Records++
	if r.Bytes > 0 || r.Packets > 0 {
		d.resolve(s, *r, ref, inRec, now, out)
	}
	if a.revBytes > 0 || a.revPkts > 0 {
		rev := *r
		rev.Src, rev.Dst = r.Dst, r.Src
		rev.SrcPort, rev.DstPort = r.DstPort, r.SrcPort
		rev.InIf, rev.OutIf = r.OutIf, r.InIf
		rev.SrcAS, rev.DstAS = r.DstAS, r.SrcAS
		rev.Bytes, rev.Packets = a.revBytes, a.revPkts
		d.resolve(s, rev, samplerRef{samplerID: a.samplerID, hasSampler: a.hasSampler, inIf: rev.InIf, outIf: rev.OutIf}, inRec, now, out)
	}
}

// holdSet keeps a data set whose template has not arrived yet.
func (s *session) holdSet(id uint16, body []byte, h msgHeader, now time.Time) {
	const maxHeld = 4 << 20
	if s.heldBytes+len(body) > maxHeld {
		s.stats.DroppedSets++
		return
	}
	cp := make([]byte, len(body))
	copy(cp, body)
	s.held[id] = append(s.held[id], heldSet{body: cp, hdr: h, at: now})
	s.heldBytes += len(cp)
	s.stats.NoTemplate++
}

// replayHeld decodes data sets that were waiting for template id.
func (d *Decoder) replayHeld(s *session, id uint16, addr netip.Addr, domain uint32, now time.Time, out *[]flow.Record) {
	hs := s.held[id]
	if len(hs) == 0 {
		return
	}
	delete(s.held, id)
	t := s.templates[id]
	for _, h := range hs {
		s.heldBytes -= len(h.body)
		d.decodeData(s, t, h.body, h.hdr, addr, domain, now, out)
	}
}

// describe keeps a short description of a template for the sources page:
// its id, whether it is an options template, and its field ids (scope
// fields marked with s, enterprise fields as pen:id).
func (s *session) describe(id uint16, t *template) {
	var b strings.Builder
	if t.options {
		fmt.Fprintf(&b, "options %d:", id)
	} else {
		fmt.Fprintf(&b, "%d:", id)
	}
	for _, f := range t.fields {
		b.WriteByte(' ')
		if f.scope {
			b.WriteByte('s')
		}
		if f.ent != 0 {
			fmt.Fprintf(&b, "%d:", f.ent)
		}
		fmt.Fprintf(&b, "%d", f.id)
	}
	if s.tmplDesc == nil {
		s.tmplDesc = map[uint16]string{}
	}
	s.tmplDesc[id] = b.String()
}
