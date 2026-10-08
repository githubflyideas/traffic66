// Package flow defines the record every input produces.
package flow

import (
	"net/netip"
	"time"
)

// Source identifies the protocol a record arrived with.
type Source uint8

const (
	SrcUnknown Source = iota
	SrcSFlow
	SrcNetFlow5
	SrcNetFlow9
	SrcIPFIX
	SrcCapture
)

func (s Source) String() string {
	switch s {
	case SrcSFlow:
		return "sFlow v5"
	case SrcNetFlow5:
		return "NetFlow v5"
	case SrcNetFlow9:
		return "NetFlow v9"
	case SrcIPFIX:
		return "IPFIX"
	case SrcCapture:
		return "capture"
	}
	return "unknown"
}

// Encap is the outer encapsulation seen on the packet, if any.
type Encap uint8

const (
	EncapNone Encap = iota
	EncapGRE
	EncapIPIP
	Encap6in4
	EncapIP6in6
	EncapVXLAN
	EncapGENEVE
	EncapMPLS
)

var encapNames = [...]string{"", "GRE", "IPIP", "6in4", "IPv6-in-IPv6", "VXLAN", "GENEVE", "MPLS"}

func (e Encap) String() string {
	if int(e) < len(encapNames) {
		return encapNames[e]
	}
	return ""
}

// Record is one flow observation. Bytes and Packets are the counts the
// exporter observed; multiply by Mult to estimate the real traffic.
type Record struct {
	Start, End time.Time

	Src, Dst         netip.Addr
	SrcPort, DstPort uint16
	Proto            uint8
	TCPFlags         uint8
	TOS              uint8

	Bytes   uint64
	Packets uint64
	// L2 is true when Bytes already counts link-layer frames (FCS included).
	L2 bool

	// Sampling is the configured 1-in-N rate reported by the exporter, 0 when
	// not yet known. Mult is the multiplier actually applied (may differ from
	// Sampling when the effective rate was measured).
	Sampling      uint32
	Mult          float64
	SamplingKnown bool

	InIf, OutIf uint32
	VLAN        uint16
	Encap       Encap
	// Direction as reported by the exporter: 0 ingress, 1 egress, 255 unknown.
	Direction uint8
	// Opener says who opened a TCP connection, when the packets showed it
	// (local capture and pcap files see the first packet of each
	// direction): 1 the source sent the SYN, -1 the source answered it with
	// SYN+ACK, 0 unknown. Exported records carry only the merged flags of
	// all packets, which cannot tell.
	Opener int8

	SrcAS, DstAS uint32

	Exporter netip.Addr
	// Domain is the NetFlow v9 source ID, IPFIX observation domain or sFlow
	// sub-agent, so two logical exporters on one address stay apart.
	Domain uint32
	// Observation is the sFlow data source (type<<24|index); 0 otherwise.
	Observation uint32
	Source      Source
}

// IfCounters is one interface counter snapshot (octets are cumulative).
type IfCounters struct {
	Time      time.Time
	Exporter  netip.Addr
	IfIndex   uint32
	Speed     uint64
	InOctets  uint64
	OutOctets uint64
	InPkts    uint64
	OutPkts   uint64
}
