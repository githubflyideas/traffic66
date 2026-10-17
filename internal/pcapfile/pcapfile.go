// Package pcapfile reads packet capture files: classic pcap (microsecond and
// nanosecond, either byte order) and pcapng, optionally gzip-compressed, as
// written by tcpdump, Wireshark and dumpcap.
package pcapfile

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// Link types handled by the flow builder.
const (
	LinkNull     = 0
	LinkEthernet = 1
	LinkRaw      = 101
	LinkLoop     = 108
	LinkSLL      = 113
	LinkIPv4     = 228
	LinkIPv6     = 229
	LinkSLL2     = 276
)

// Packet is one captured packet. Data is valid until the next call to Next.
type Packet struct {
	Time    time.Time
	Data    []byte // captured bytes (may be shorter than the packet)
	WireLen int    // length of the packet on the wire
	Link    int    // link type of the interface it was captured on
}

// ErrFormat is returned for files that are neither pcap nor pcapng.
var ErrFormat = errors.New("not a pcap or pcapng file")

// maxPacket bounds one packet record; larger lengths mean a damaged file.
const maxPacket = 1 << 18

// Reader reads packets from a capture file.
type Reader struct {
	r     *bufio.Reader
	ng    bool
	bo    binary.ByteOrder
	buf   []byte
	Count uint64

	// pcap
	link  int
	nanos bool

	// pcapng
	ifaces []iface
	lastTS time.Time
}

type iface struct {
	link int
	res  float64 // seconds per timestamp unit
}

// Format names the format of a file from its first bytes ("pcap",
// "pcapng") or returns "".
func Format(b []byte) string {
	if len(b) < 4 {
		return ""
	}
	switch binary.LittleEndian.Uint32(b) {
	case 0xa1b2c3d4, 0xd4c3b2a1, 0xa1b23c4d, 0x4d3cb2a1:
		return "pcap"
	case 0x0a0d0d0a:
		return "pcapng"
	}
	return ""
}

// NewReader reads the file header from r.
func NewReader(r io.Reader) (*Reader, error) {
	br := bufio.NewReaderSize(r, 1<<16)
	head, err := br.Peek(4)
	if err != nil {
		return nil, ErrFormat
	}
	rd := &Reader{r: br, buf: make([]byte, 0, 1<<16)}
	switch Format(head) {
	case "pcap":
		return rd, rd.pcapHeader()
	case "pcapng":
		rd.ng = true
		return rd, nil
	}
	return nil, ErrFormat
}

func (rd *Reader) read(n int) ([]byte, error) {
	if cap(rd.buf) < n {
		rd.buf = make([]byte, n)
	}
	b := rd.buf[:n]
	_, err := io.ReadFull(rd.r, b)
	return b, err
}

func (rd *Reader) pcapHeader() error {
	h, err := rd.read(24)
	if err != nil {
		return ErrFormat
	}
	switch binary.LittleEndian.Uint32(h) {
	case 0xa1b2c3d4:
		rd.bo = binary.LittleEndian
	case 0xa1b23c4d:
		rd.bo, rd.nanos = binary.LittleEndian, true
	case 0xd4c3b2a1:
		rd.bo = binary.BigEndian
	case 0x4d3cb2a1:
		rd.bo, rd.nanos = binary.BigEndian, true
	}
	// the low 16 bits are the link type; the high bits may carry an FCS flag
	rd.link = int(rd.bo.Uint32(h[20:]) & 0xffff)
	return nil
}

// Next returns the next packet, or io.EOF at the end of the file.
func (rd *Reader) Next(p *Packet) error {
	if rd.ng {
		return rd.nextNG(p)
	}
	h, err := rd.read(16)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return io.EOF // a capture cut off mid-header
		}
		return err
	}
	sec, frac := rd.bo.Uint32(h), rd.bo.Uint32(h[4:])
	incl, orig := rd.bo.Uint32(h[8:]), rd.bo.Uint32(h[12:])
	if incl > maxPacket {
		return fmt.Errorf("packet %d: damaged record (%d bytes)", rd.Count+1, incl)
	}
	ns := int64(frac) * 1000
	if rd.nanos {
		ns = int64(frac)
	}
	data, err := rd.read(int(incl))
	if err != nil {
		return io.EOF
	}
	rd.Count++
	*p = Packet{Time: time.Unix(int64(sec), ns).UTC(), Data: data, WireLen: int(max(orig, incl)), Link: rd.link}
	return nil
}

func (rd *Reader) nextNG(p *Packet) error {
	for {
		h, err := rd.read(8)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				return io.EOF
			}
			return err
		}
		typ := binary.LittleEndian.Uint32(h)
		if typ == 0x0a0d0d0a { // section header: learn the byte order
			if err := rd.section(h); err != nil {
				return err
			}
			continue
		}
		if rd.bo == nil {
			return ErrFormat
		}
		typ = rd.bo.Uint32(h)
		blen := rd.bo.Uint32(h[4:])
		if blen < 12 || blen > maxPacket+64 || blen%4 != 0 {
			return fmt.Errorf("block after packet %d: damaged (%d bytes)", rd.Count, blen)
		}
		body, err := rd.read(int(blen - 8))
		if err != nil {
			return io.EOF
		}
		body = body[:len(body)-4] // trailing length
		switch typ {
		case 1: // interface description
			if len(body) < 8 {
				continue
			}
			ifc := iface{link: int(rd.bo.Uint16(body)), res: 1e-6}
			rd.options(body[8:], func(code uint16, v []byte) {
				if code == 9 && len(v) >= 1 { // if_tsresol
					if v[0]&0x80 != 0 {
						ifc.res = math.Pow(2, -float64(v[0]&0x7f))
					} else {
						ifc.res = math.Pow(10, -float64(v[0]))
					}
				}
			})
			rd.ifaces = append(rd.ifaces, ifc)
		case 6, 2: // enhanced packet, obsolete packet
			if len(body) < 20 {
				continue
			}
			id := rd.bo.Uint32(body)
			if typ == 2 {
				id = uint32(rd.bo.Uint16(body))
			}
			ts := uint64(rd.bo.Uint32(body[4:]))<<32 | uint64(rd.bo.Uint32(body[8:]))
			incl, orig := rd.bo.Uint32(body[12:]), rd.bo.Uint32(body[16:])
			if int(id) >= len(rd.ifaces) || 20+int(incl) > len(body) {
				continue
			}
			ifc := rd.ifaces[id]
			t := tsTime(ts, ifc.res)
			rd.lastTS = t
			rd.Count++
			*p = Packet{Time: t, Data: body[20 : 20+incl], WireLen: int(max(orig, incl)), Link: ifc.link}
			return nil
		case 3: // simple packet: no timestamp, interface 0
			if len(body) < 4 || len(rd.ifaces) == 0 {
				continue
			}
			orig := rd.bo.Uint32(body)
			data := body[4:]
			if int(orig) < len(data) {
				data = data[:orig]
			}
			rd.Count++
			*p = Packet{Time: rd.lastTS, Data: data, WireLen: int(max(orig, uint32(len(data)))), Link: rd.ifaces[0].link}
			return nil
		}
	}
}

func (rd *Reader) section(h []byte) error {
	blen := h[4:8]
	bom, err := rd.r.Peek(4)
	if err != nil {
		return ErrFormat
	}
	switch binary.LittleEndian.Uint32(bom) {
	case 0x1a2b3c4d:
		rd.bo = binary.LittleEndian
	case 0x4d3c2b1a:
		rd.bo = binary.BigEndian
	default:
		return ErrFormat
	}
	n := rd.bo.Uint32(blen)
	if n < 28 || n > 1<<20 {
		return ErrFormat
	}
	if _, err := rd.read(int(n - 8)); err != nil {
		return ErrFormat
	}
	rd.ifaces = nil // interfaces are per section
	return nil
}

func (rd *Reader) options(b []byte, fn func(code uint16, v []byte)) {
	for len(b) >= 4 {
		code, n := rd.bo.Uint16(b), int(rd.bo.Uint16(b[2:]))
		if code == 0 || 4+n > len(b) {
			return
		}
		fn(code, b[4:4+n])
		b = b[4+(n+3)&^3:]
	}
}

func tsTime(ts uint64, res float64) time.Time {
	if res == 1e-6 {
		return time.UnixMicro(int64(ts)).UTC()
	}
	if res == 1e-9 {
		return time.Unix(0, int64(ts)).UTC()
	}
	sec := float64(ts) * res
	s := math.Floor(sec)
	return time.Unix(int64(s), int64((sec-s)*1e9)).UTC()
}
