package nf

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"
)

// be builds a big-endian byte string from 16-bit words, which is what an
// IPFIX header and its sets are made of.
func be16(vals ...uint16) []byte {
	b := make([]byte, 2*len(vals))
	for i, v := range vals {
		binary.BigEndian.PutUint16(b[2*i:], v)
	}
	return b
}

// A message that registers a template and withdraws it again, with a data
// set already waiting for that template, used to take the process down: the
// withdrawal left the id in the list of templates to replay, the lookup
// came back nil, and the replay walked a nil template's fields.
func TestWithdrawnTemplateDoesNotCrashTheReplay(t *testing.T) {
	d := NewDecoder()
	addr := netip.MustParseAddr("10.0.0.1")
	now := time.Now()

	// A data set for template 300, which is not known yet, so it is held.
	data := append(be16(300, 12), make([]byte, 8)...)
	hdr := func(payload []byte) []byte {
		h := make([]byte, 16)
		binary.BigEndian.PutUint16(h[0:], 10)
		binary.BigEndian.PutUint16(h[2:], uint16(16+len(payload)))
		binary.BigEndian.PutUint32(h[4:], uint32(now.Unix()))
		return append(h, payload...)
	}
	if _, err := d.Decode(hdr(data), addr, now); err != nil {
		t.Fatalf("holding the data set failed: %v", err)
	}

	// One template set that first defines template 300 and then withdraws
	// it — count 0 is a withdrawal in IPFIX.
	tmpl := be16(300, 1, 1, 4) // id 300, one field: octetDeltaCount, 4 bytes
	tmpl = append(tmpl, be16(300, 0)...)
	set := append(be16(2, uint16(4+len(tmpl))), tmpl...)
	if _, err := d.Decode(hdr(set), addr, now); err != nil {
		t.Fatalf("withdrawing the template failed: %v", err)
	}
	st := d.Stats()
	if len(st) != 1 {
		t.Fatalf("%d sessions, want 1", len(st))
	}
	if st[0].DroppedSets != 1 {
		t.Errorf("DroppedSets = %d, want 1: the held set can never be decoded now, and saying so is the point of the counter", st[0].DroppedSets)
	}
	if st[0].Templates != 0 || len(st[0].TemplateInfo) != 0 {
		t.Errorf("%d templates and %v left after the withdrawal, want none: the sources page should stop listing a template the exporter has taken back",
			st[0].Templates, st[0].TemplateInfo)
	}
}

// NetFlow v9 has no withdrawal: a template that declares no fields is
// malformed. Treating it as a withdrawal threw away a template the
// exporter was still using, and every later data set for it with it.
func TestV9HasNoTemplateWithdrawal(t *testing.T) {
	d := NewDecoder()
	addr := netip.MustParseAddr("10.0.0.2")
	now := time.Now()
	hdr := func(payload []byte) []byte {
		h := make([]byte, 20)
		binary.BigEndian.PutUint16(h[0:], 9)
		binary.BigEndian.PutUint16(h[2:], 1) // one flowset
		binary.BigEndian.PutUint32(h[8:], uint32(now.Unix()))
		return append(h, payload...)
	}
	tmpl := be16(300, 1, 1, 4)           // template 300, one field
	tmpl = append(tmpl, be16(300, 0)...) // and a zero-field template after it
	set := append(be16(0, uint16(4+len(tmpl))), tmpl...)
	if _, err := d.Decode(hdr(set), addr, now); err != nil {
		t.Fatalf("v9 template set failed: %v", err)
	}
	st := d.Stats()
	if len(st) != 1 || st[0].Templates != 1 {
		t.Errorf("templates = %v, want template 300 still registered", st)
	}
}
