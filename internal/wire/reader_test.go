package wire

import (
	"errors"
	"net/netip"
	"testing"
)

func TestReaderBigEndian(t *testing.T) {
	r := New([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09})
	if got := r.U8(); got != 0x01 {
		t.Errorf("U8 = %x", got)
	}
	if got := r.U16(); got != 0x0203 {
		t.Errorf("U16 = %x", got)
	}
	if got := r.U32(); got != 0x04050607 {
		t.Errorf("U32 = %x", got)
	}
	if r.Len() != 2 || r.Err() != nil {
		t.Errorf("after reads: Len %d Err %v", r.Len(), r.Err())
	}
	if got := r.U64(); got != 0 {
		t.Errorf("U64 past the end = %x, want 0", got)
	}
	if !errors.Is(r.Err(), ErrShort) {
		t.Errorf("Err = %v, want ErrShort", r.Err())
	}
}

func TestReaderShortIsSticky(t *testing.T) {
	r := New([]byte{0xAA})
	if got := r.U32(); got != 0 {
		t.Errorf("short U32 = %x", got)
	}
	// after the first short read, everything reads as zero and the
	// buffer is gone, however much is asked for
	if got := r.U8(); got != 0 {
		t.Errorf("U8 after short = %x", got)
	}
	if b := r.Bytes(1); b != nil {
		t.Errorf("Bytes after short = %v", b)
	}
	if ip := r.IPv4(); ip != (netip.Addr{}) {
		t.Errorf("IPv4 after short = %v", ip)
	}
	if r.Len() != 0 || len(r.Rest()) != 0 {
		t.Errorf("Len %d Rest %v, want both empty", r.Len(), r.Rest())
	}
}

func TestReaderBytesSkipAndRest(t *testing.T) {
	r := New([]byte{1, 2, 3, 4, 5, 6})
	if b := r.Bytes(2); len(b) != 2 || b[0] != 1 || b[1] != 2 {
		t.Fatalf("Bytes(2) = %v", b)
	}
	r.Skip(3)
	if rest := r.Rest(); len(rest) != 1 || rest[0] != 6 {
		t.Errorf("Rest = %v", rest)
	}
	r.Skip(-1) // a negative skip is a short read, not a panic
	if !errors.Is(r.Err(), ErrShort) {
		t.Errorf("Skip(-1): Err = %v", r.Err())
	}
}

func TestReaderAddresses(t *testing.T) {
	r := New([]byte{192, 0, 2, 1})
	if ip := r.IPv4(); ip != netip.MustParseAddr("192.0.2.1") {
		t.Errorf("IPv4 = %v", ip)
	}
	r = New([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	if ip := r.IPv6(); ip != netip.MustParseAddr("2001:db8::1") {
		t.Errorf("IPv6 = %v", ip)
	}
}

func TestUint(t *testing.T) {
	for _, tc := range []struct {
		b    []byte
		want uint64
	}{
		{nil, 0},
		{[]byte{0x05}, 5},
		{[]byte{0x01, 0x00}, 256},
		{[]byte{0xDE, 0xAD, 0xBE, 0xEF}, 0xDEADBEEF},
		// more than 8 bytes: the low 8 count, the rest is dropped
		{[]byte{0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 0x0102030405060708},
	} {
		if got := Uint(tc.b); got != tc.want {
			t.Errorf("Uint(%x) = %x, want %x", tc.b, got, tc.want)
		}
	}
}
