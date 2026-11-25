package pcapfile

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"
)

func readAll(t *testing.T, b []byte) []Packet {
	t.Helper()
	rd, err := NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var out []Packet
	for {
		var p Packet
		err := rd.Next(&p)
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		p.Data = append([]byte(nil), p.Data...)
		out = append(out, p)
	}
}

func TestPcapRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w, _ := NewWriter(&buf, LinkEthernet)
	t0 := time.Date(2026, 9, 1, 10, 0, 0, 123456000, time.UTC)
	w.Write(t0, []byte{1, 2, 3}, 60)
	w.Write(t0.Add(time.Second), []byte{4, 5}, 2)
	for _, b := range [][]byte{buf.Bytes()} {
		ps := readAll(t, b)
		if len(ps) != 2 || !ps[0].Time.Equal(t0) || ps[0].WireLen != 60 || ps[1].Link != LinkEthernet || !bytes.Equal(ps[1].Data, []byte{4, 5}) {
			t.Fatalf("%+v", ps)
		}
	}
	// cut off in the middle of a packet: the complete ones are kept
	if ps := readAll(t, buf.Bytes()[:buf.Len()-1]); len(ps) != 1 {
		t.Fatalf("truncated: %d packets", len(ps))
	}
}

// pcapng written by hand: section header, two interfaces (the second with
// nanosecond resolution), enhanced and simple packet blocks, an unknown block.
func TestPcapng(t *testing.T) {
	le := binary.LittleEndian
	var b bytes.Buffer
	block := func(typ uint32, body []byte) {
		for len(body)%4 != 0 {
			body = append(body, 0)
		}
		n := uint32(12 + len(body))
		b.Write(le.AppendUint32(le.AppendUint32(nil, typ), n))
		b.Write(body)
		b.Write(le.AppendUint32(nil, n))
	}
	shb := le.AppendUint32(nil, 0x1a2b3c4d)
	shb = append(shb, 1, 0, 0, 0)
	shb = append(shb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)
	block(0x0a0d0d0a, shb)
	idb := func(link uint16, opts []byte) []byte {
		x := le.AppendUint16(nil, link)
		x = append(x, 0, 0)
		x = le.AppendUint32(x, 0)
		return append(x, opts...)
	}
	block(1, idb(LinkEthernet, nil))
	block(1, idb(LinkRaw, []byte{9, 0, 1, 0, 9, 0, 0, 0, 0, 0, 0, 0})) // if_tsresol=9, end
	epb := func(id uint32, ts uint64, data []byte, orig uint32) []byte {
		x := le.AppendUint32(nil, id)
		x = le.AppendUint32(x, uint32(ts>>32))
		x = le.AppendUint32(x, uint32(ts))
		x = le.AppendUint32(x, uint32(len(data)))
		x = le.AppendUint32(x, orig)
		return append(x, data...)
	}
	t0 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	block(6, epb(0, uint64(t0.UnixMicro()), []byte{1, 2, 3}, 100))
	block(99, []byte{1, 2, 3, 4})
	block(6, epb(1, uint64(t0.Add(time.Second+7).UnixNano()), []byte{0x45, 0}, 2))
	block(3, append(le.AppendUint32(nil, 5), 9, 9, 9, 9, 9, 9, 9))
	ps := readAll(t, b.Bytes())
	if len(ps) != 3 {
		t.Fatalf("%d packets", len(ps))
	}
	if !ps[0].Time.Equal(t0) || ps[0].WireLen != 100 || ps[0].Link != LinkEthernet {
		t.Errorf("first %+v", ps[0])
	}
	if !ps[1].Time.Equal(t0.Add(time.Second+7)) || ps[1].Link != LinkRaw {
		t.Errorf("second %+v", ps[1])
	}
	if len(ps[2].Data) != 5 || !ps[2].Time.Equal(ps[1].Time) {
		t.Errorf("simple %+v", ps[2])
	}
}

func TestNotACapture(t *testing.T) {
	gz := []byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 3}
	for _, b := range [][]byte{[]byte("hello world, this is text"), gz, []byte("ustar\x00")} {
		if _, err := NewReader(bytes.NewReader(b)); err != ErrFormat {
			t.Fatalf("%q: %v", b, err)
		}
	}
}
