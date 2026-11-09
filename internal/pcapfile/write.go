package pcapfile

import (
	"encoding/binary"
	"io"
	"time"
)

// Writer writes a classic pcap file (microsecond timestamps, little endian).
type Writer struct {
	w   io.Writer
	hdr [16]byte
}

// NewWriter writes the file header for link type link.
func NewWriter(w io.Writer, link int) (*Writer, error) {
	var h [24]byte
	binary.LittleEndian.PutUint32(h[0:], 0xa1b2c3d4)
	binary.LittleEndian.PutUint16(h[4:], 2)
	binary.LittleEndian.PutUint16(h[6:], 4)
	binary.LittleEndian.PutUint32(h[16:], 65535)
	binary.LittleEndian.PutUint32(h[20:], uint32(link))
	_, err := w.Write(h[:])
	return &Writer{w: w}, err
}

// Write writes one packet of wireLen bytes on the wire, of which data was
// captured.
func (w *Writer) Write(t time.Time, data []byte, wireLen int) error {
	us := t.UnixMicro()
	binary.LittleEndian.PutUint32(w.hdr[0:], uint32(us/1e6))
	binary.LittleEndian.PutUint32(w.hdr[4:], uint32(us%1e6))
	binary.LittleEndian.PutUint32(w.hdr[8:], uint32(len(data)))
	binary.LittleEndian.PutUint32(w.hdr[12:], uint32(max(wireLen, len(data))))
	if _, err := w.w.Write(w.hdr[:]); err != nil {
		return err
	}
	_, err := w.w.Write(data)
	return err
}
