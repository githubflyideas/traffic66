package sandbox

import (
	"encoding/binary"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/packet"
	"github.com/githubflyideas/traffic66/internal/pcapfile"
)

// builder turns packets into flow records the way local capture does
// (60 s active, 15 s idle timeout), but on the packets' own timestamps.
type builder struct {
	exporter netip.Addr
	domain   uint32
	table    map[fkey]*fentry
	swept    time.Time
	emit     func([]flow.Record)
	skipped  uint64 // packets without an IP header
}

type fkey struct {
	src, dst     netip.Addr
	sport, dport uint16
	proto        uint8
	vlan         uint16
	encap        flow.Encap
	l2           bool
}

type fentry struct {
	first, last time.Time
	pkts, bytes uint64
	flags       uint8
}

const (
	activeTimeout = 60 * time.Second
	idleTimeout   = 15 * time.Second
	maxEntries    = 500000
)

func newBuilder(exporter netip.Addr, domain uint32, emit func([]flow.Record)) *builder {
	return &builder{exporter: exporter, domain: domain, table: map[fkey]*fentry{}, emit: emit}
}

// decode reads the headers of a packet captured on link type link.
func decode(p *pcapfile.Packet, r *flow.Record) (l2 bool, ok bool) {
	b := p.Data
	switch p.Link {
	case pcapfile.LinkEthernet:
		return true, packet.Ethernet(b, r)
	case pcapfile.LinkRaw, pcapfile.LinkIPv4, pcapfile.LinkIPv6, 12, 14:
		return false, packet.IP(b, r)
	case pcapfile.LinkNull, pcapfile.LinkLoop:
		if len(b) < 4 {
			return false, false
		}
		return false, packet.IP(b[4:], r)
	case pcapfile.LinkSLL:
		if len(b) < 16 {
			return false, false
		}
		return false, ipType(binary.BigEndian.Uint16(b[14:])) && packet.IP(b[16:], r)
	case pcapfile.LinkSLL2:
		if len(b) < 20 {
			return false, false
		}
		return false, ipType(binary.BigEndian.Uint16(b)) && packet.IP(b[20:], r)
	}
	return false, false
}

func ipType(t uint16) bool { return t == 0x0800 || t == 0x86dd }

func (b *builder) add(p *pcapfile.Packet) {
	var r flow.Record
	l2, ok := decode(p, &r)
	if !ok {
		b.skipped++
		return
	}
	k := fkey{r.Src, r.Dst, r.SrcPort, r.DstPort, r.Proto, r.VLAN, r.Encap, l2}
	e := b.table[k]
	if e == nil {
		if len(b.table) >= maxEntries {
			b.sweep(p.Time, true)
		}
		e = &fentry{first: p.Time}
		b.table[k] = e
	}
	if p.Time.After(e.last) {
		e.last = p.Time
	}
	e.pkts++
	n := uint64(p.WireLen)
	if l2 {
		n += 4 // captured frames exclude the FCS
	}
	e.bytes += n
	e.flags |= r.TCPFlags
	if p.Time.Sub(b.swept) >= time.Second {
		b.sweep(p.Time, false)
	}
}

// sweep emits flows that ended (idle) or ran long enough (active); all
// emits everything.
func (b *builder) sweep(now time.Time, all bool) {
	b.swept = now
	var out []flow.Record
	for k, e := range b.table {
		idle := now.Sub(e.last) >= idleTimeout
		active := now.Sub(e.first) >= activeTimeout
		if !all && !idle && !active {
			continue
		}
		if e.pkts > 0 {
			out = append(out, flow.Record{Start: e.first, End: e.last, Src: k.src, Dst: k.dst, SrcPort: k.sport, DstPort: k.dport,
				Proto: k.proto, VLAN: k.vlan, Encap: k.encap, TCPFlags: e.flags, Bytes: e.bytes, Packets: e.pkts, L2: k.l2,
				Sampling: 1, Mult: 1, SamplingKnown: true, Exporter: b.exporter, Domain: b.domain, Source: flow.SrcCapture, Direction: 255})
		}
		if idle || all {
			delete(b.table, k)
		} else {
			*e = fentry{first: now, last: now}
		}
	}
	if len(out) > 0 {
		b.emit(out)
	}
}
