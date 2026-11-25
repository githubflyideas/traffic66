// Package wire is a bounds-checked big-endian reader for protocol decoding.
package wire

import (
	"encoding/binary"
	"errors"
	"net/netip"
)

// ErrShort is returned when a read runs past the end of the buffer.
var ErrShort = errors.New("wire: short buffer")

// Reader reads big-endian values; after the first short read every further
// read returns zero and Err reports ErrShort.
type Reader struct {
	b   []byte
	err error
}

func New(b []byte) *Reader { return &Reader{b: b} }

func (r *Reader) Err() error { return r.err }
func (r *Reader) Len() int   { return len(r.b) }
func (r *Reader) Rest() []byte {
	return r.b
}

func (r *Reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || len(r.b) < n {
		r.err = ErrShort
		r.b = nil
		return false
	}
	return true
}

func (r *Reader) U8() uint8 {
	if !r.need(1) {
		return 0
	}
	v := r.b[0]
	r.b = r.b[1:]
	return v
}

func (r *Reader) U16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := binary.BigEndian.Uint16(r.b)
	r.b = r.b[2:]
	return v
}

func (r *Reader) U32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := binary.BigEndian.Uint32(r.b)
	r.b = r.b[4:]
	return v
}

func (r *Reader) U64() uint64 {
	if !r.need(8) {
		return 0
	}
	v := binary.BigEndian.Uint64(r.b)
	r.b = r.b[8:]
	return v
}

// Bytes returns the next n bytes without copying.
func (r *Reader) Bytes(n int) []byte {
	if !r.need(n) {
		return nil
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

func (r *Reader) Skip(n int) { r.Bytes(n) }

func (r *Reader) IPv4() netip.Addr {
	b := r.Bytes(4)
	if b == nil {
		return netip.Addr{}
	}
	return netip.AddrFrom4([4]byte(b))
}

func (r *Reader) IPv6() netip.Addr {
	b := r.Bytes(16)
	if b == nil {
		return netip.Addr{}
	}
	return netip.AddrFrom16([16]byte(b))
}

// Uint reads an unsigned big-endian integer of 1..8 bytes.
func Uint(b []byte) uint64 {
	var v uint64
	if len(b) > 8 {
		b = b[len(b)-8:]
	}
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}
