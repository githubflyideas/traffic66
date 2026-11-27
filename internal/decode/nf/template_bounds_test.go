package nf

import (
	"encoding/binary"
	"runtime"
	"testing"
)

// hugeCountPacket is one IPFIX message, 24 bytes: a template set whose one
// template claims 65535 fields and carries none.
func hugeCountPacket() []byte {
	b := make([]byte, 0, 40)
	b = binary.BigEndian.AppendUint16(b, 10)                  // version
	b = binary.BigEndian.AppendUint16(b, 16+4+4)              // message length, patched below
	b = binary.BigEndian.AppendUint32(b, uint32(base.Unix())) // export time
	b = binary.BigEndian.AppendUint32(b, 0)                   // sequence
	b = binary.BigEndian.AppendUint32(b, 1)                   // domain
	b = binary.BigEndian.AppendUint16(b, 2)                   // template set
	b = binary.BigEndian.AppendUint16(b, 4+4)                 // set length
	b = binary.BigEndian.AppendUint16(b, 256)                 // template id
	b = binary.BigEndian.AppendUint16(b, 65535)               // field count
	binary.BigEndian.PutUint16(b[2:], uint16(len(b)))
	return b
}

// A template that claims more fields than its set has bytes for costs
// nothing: a 24-byte datagram claiming 65535 fields once made the
// collector allocate over 4 MB, on a port that takes datagrams from
// anyone. The count is held to what the bytes can carry.
func TestTemplateFieldCountIsBoundedByTheBytes(t *testing.T) {
	pkt := hugeCountPacket()
	d := NewDecoder()
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	const n = 2000
	for i := 0; i < n; i++ {
		d.Decode(pkt, exp, base)
	}
	runtime.ReadMemStats(&m1)
	if per := (m1.TotalAlloc - m0.TotalAlloc) / n; per > 4096 {
		t.Errorf("%d-byte packet allocated %d bytes: the field count is not bounded", len(pkt), per)
	}
	if len(d.Stats()) > 0 && d.Stats()[0].Templates != 0 {
		t.Error("a template that claimed fields it did not have was stored")
	}
}
