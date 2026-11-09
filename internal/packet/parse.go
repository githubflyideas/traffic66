// Package packet reads the outer L2/L3/L4 headers of a (possibly truncated)
// packet into a flow record and notes any tunnel encapsulation.
package packet

import (
	"encoding/binary"
	"net/netip"

	"github.com/githubflyideas/traffic66/internal/flow"
)

const (
	etherIPv4   = 0x0800
	etherIPv6   = 0x86DD
	etherVLAN   = 0x8100
	etherQinQ   = 0x88A8
	etherQinQ2  = 0x9100
	etherMPLS   = 0x8847
	etherMPLSMc = 0x8848

	protoICMP   = 1
	protoIPIP   = 4
	protoTCP    = 6
	protoUDP    = 17
	protoIPv6   = 41
	protoGRE    = 47
	protoICMPv6 = 58
	protoSCTP   = 132

	portVXLAN  = 4789
	portGENEVE = 6081
)

// Ethernet parses an Ethernet II frame starting at the destination MAC.
// It reports whether an IP header was found.
func Ethernet(b []byte, r *flow.Record) bool {
	if len(b) < 14 {
		return false
	}
	et := binary.BigEndian.Uint16(b[12:14])
	b = b[14:]
	for et == etherVLAN || et == etherQinQ || et == etherQinQ2 {
		if len(b) < 4 {
			return false
		}
		if r.VLAN == 0 {
			r.VLAN = binary.BigEndian.Uint16(b[0:2]) & 0x0FFF
		}
		et = binary.BigEndian.Uint16(b[2:4])
		b = b[4:]
	}
	switch et {
	case etherIPv4:
		return IPv4(b, r)
	case etherIPv6:
		return IPv6(b, r)
	case etherMPLS, etherMPLSMc:
		return mpls(b, r)
	}
	return false
}

func mpls(b []byte, r *flow.Record) bool {
	for {
		if len(b) < 4 {
			return false
		}
		bottom := b[2]&0x01 != 0
		b = b[4:]
		if bottom {
			break
		}
	}
	if r.Encap == flow.EncapNone {
		r.Encap = flow.EncapMPLS
	}
	return IP(b, r)
}

// IP parses an IPv4 or IPv6 header, choosing by the version nibble.
func IP(b []byte, r *flow.Record) bool {
	if len(b) < 1 {
		return false
	}
	switch b[0] >> 4 {
	case 4:
		return IPv4(b, r)
	case 6:
		return IPv6(b, r)
	}
	return false
}

// IPv4 parses an IPv4 header and the transport header after it.
func IPv4(b []byte, r *flow.Record) bool {
	if len(b) < 20 || b[0]>>4 != 4 {
		return false
	}
	ihl := int(b[0]&0x0F) * 4
	if ihl < 20 {
		return false
	}
	r.TOS = b[1]
	r.Proto = b[9]
	r.Src = netip.AddrFrom4([4]byte(b[12:16]))
	r.Dst = netip.AddrFrom4([4]byte(b[16:20]))
	fragOff := binary.BigEndian.Uint16(b[6:8]) & 0x1FFF
	if fragOff != 0 || len(b) < ihl {
		return true
	}
	transport(b[ihl:], r, false)
	return true
}

// IPv6 parses an IPv6 header, skipping common extension headers.
func IPv6(b []byte, r *flow.Record) bool {
	if len(b) < 40 || b[0]>>4 != 6 {
		return false
	}
	r.TOS = byte(binary.BigEndian.Uint16(b[0:2]) >> 4)
	r.Src = netip.AddrFrom16([16]byte(b[8:24]))
	r.Dst = netip.AddrFrom16([16]byte(b[24:40]))
	next := b[6]
	b = b[40:]
ext:
	for i := 0; i < 8; i++ {
		switch next {
		case 0, 43, 60: // hop-by-hop, routing, destination options
			if len(b) < 8 {
				r.Proto = next
				return true
			}
			l := (int(b[1]) + 1) * 8
			next = b[0]
			if len(b) < l {
				r.Proto = next
				return true
			}
			b = b[l:]
			continue
		case 44: // fragment
			if len(b) < 8 {
				r.Proto = next
				return true
			}
			off := binary.BigEndian.Uint16(b[2:4]) >> 3
			next = b[0]
			r.Proto = next
			if off != 0 {
				return true
			}
			b = b[8:]
			continue
		}
		break ext
	}
	r.Proto = next
	transport(b, r, true)
	return true
}

func transport(b []byte, r *flow.Record, outerV6 bool) {
	switch r.Proto {
	case protoTCP:
		if len(b) >= 4 {
			r.SrcPort = binary.BigEndian.Uint16(b[0:2])
			r.DstPort = binary.BigEndian.Uint16(b[2:4])
		}
		if len(b) >= 14 {
			r.TCPFlags |= b[13]
		}
	case protoUDP, protoSCTP:
		if len(b) >= 4 {
			r.SrcPort = binary.BigEndian.Uint16(b[0:2])
			r.DstPort = binary.BigEndian.Uint16(b[2:4])
		}
		if r.Proto == protoUDP && r.Encap == flow.EncapNone {
			switch r.DstPort {
			case portVXLAN:
				r.Encap = flow.EncapVXLAN
			case portGENEVE:
				r.Encap = flow.EncapGENEVE
			}
		}
	case protoICMP, protoICMPv6:
		if len(b) >= 2 {
			// type/code in the destination port, as NetFlow does.
			r.DstPort = uint16(b[0])<<8 | uint16(b[1])
		}
	case protoGRE:
		if r.Encap == flow.EncapNone {
			r.Encap = flow.EncapGRE
		}
	case protoIPIP:
		if r.Encap == flow.EncapNone {
			r.Encap = flow.EncapIPIP
		}
	case protoIPv6:
		if r.Encap == flow.EncapNone {
			if outerV6 {
				r.Encap = flow.EncapIP6in6
			} else {
				r.Encap = flow.Encap6in4
			}
		}
	}
}

// EncapFromTuple infers encapsulation from protocol and port when only the
// flow key is known (NetFlow/IPFIX records carry no packet bytes).
func EncapFromTuple(r *flow.Record) {
	if r.Encap != flow.EncapNone {
		return
	}
	switch r.Proto {
	case protoGRE:
		r.Encap = flow.EncapGRE
	case protoIPIP:
		r.Encap = flow.EncapIPIP
	case protoIPv6:
		if r.Src.Is6() && !r.Src.Is4In6() {
			r.Encap = flow.EncapIP6in6
		} else {
			r.Encap = flow.Encap6in4
		}
	case protoUDP:
		switch r.DstPort {
		case portVXLAN:
			r.Encap = flow.EncapVXLAN
		case portGENEVE:
			r.Encap = flow.EncapGENEVE
		}
	}
}
