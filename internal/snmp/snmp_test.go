package snmp

import (
	"bytes"
	"context"
	"math"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestIntegerEncoding(t *testing.T) {
	cases := map[int64][]byte{
		0: {0}, 127: {0x7f}, 128: {0, 0x80}, 256: {1, 0}, -1: {0xff}, -128: {0x80}, -129: {0xff, 0x7f},
		math.MaxInt32: {0x7f, 0xff, 0xff, 0xff}, math.MinInt32: {0x80, 0, 0, 0},
	}
	for v, want := range cases {
		got := encInt(v)
		if !bytes.Equal(got, want) {
			t.Errorf("encInt(%d) = % x, want % x", v, got, want)
		}
		back, err := decInt(got)
		if err != nil || back != v {
			t.Errorf("decInt(% x) = %d, %v", got, back, err)
		}
	}
	for _, v := range []uint64{0, 127, 128, 1 << 32, math.MaxUint64} {
		back, err := decUint(encUint(v))
		if err != nil || back != v {
			t.Errorf("uint %d round trip = %d, %v", v, back, err)
		}
	}
}

func TestOIDEncoding(t *testing.T) {
	// X.690 example: 2.999.3 encodes as 88 37 03
	b, err := encOID(OID{2, 999, 3})
	if err != nil || !bytes.Equal(b, []byte{0x88, 0x37, 0x03}) {
		t.Fatalf("encOID(2.999.3) = % x, %v", b, err)
	}
	o, err := decOID(b)
	if err != nil || o.String() != "2.999.3" {
		t.Fatalf("decOID = %s, %v", o, err)
	}
	o2, _ := decOID(mustEnc(OIDIfHCInOctets))
	if o2.Compare(OIDIfHCInOctets) != 0 {
		t.Fatalf("round trip %s", o2)
	}
}

func mustEnc(o OID) []byte { b, _ := encOID(o); return b }

func startAgent(t *testing.T, community string, rows []InterfaceRow) netip.AddrPort {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a := &Agent{Community: community, View: func() []VarBind { return InterfaceView(rows) }}
	go a.Serve(ctx, conn)
	return conn.LocalAddr().(*net.UDPAddr).AddrPort()
}

func TestPollInterfaces(t *testing.T) {
	var rows []InterfaceRow
	for i := uint32(1); i <= 60; i++ { // more rows than one GETBULK returns
		rows = append(rows, InterfaceRow{Index: i, Name: "Gi1/0/" + string(rune('0'+i%10)), SpeedMbps: 1000,
			InOctets: uint64(i) << 40, OutOctets: uint64(i) * 7, InPkts: uint64(i), OutPkts: 1})
	}
	addr := startAgent(t, "s3cret", rows)
	exp := netip.MustParseAddr("10.0.0.1")
	c := &Client{Addr: addr, Community: "s3cret", Timeout: time.Second}
	ctrs, ifs, err := c.PollInterfaces(context.Background(), exp, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(ctrs) != 60 || len(ifs) != 60 {
		t.Fatalf("got %d counters, %d names", len(ctrs), len(ifs))
	}
	for i, c := range ctrs {
		want := uint64(i+1) << 40
		if c.IfIndex != uint32(i+1) || c.InOctets != want || c.OutOctets != uint64(i+1)*7 || c.Speed != 1e9 || c.Exporter != exp {
			t.Fatalf("row %d = %+v", i, c)
		}
	}
	if ifs[0].Name != "Gi1/0/1" {
		t.Fatalf("name %q", ifs[0].Name)
	}
}

func TestWrongCommunityTimesOut(t *testing.T) {
	addr := startAgent(t, "right", []InterfaceRow{{Index: 1}})
	c := &Client{Addr: addr, Community: "wrong", Timeout: 150 * time.Millisecond}
	_, _, err := c.PollInterfaces(context.Background(), netip.MustParseAddr("10.0.0.1"), time.Now())
	if err != ErrTimeout {
		t.Fatalf("err = %v, want timeout", err)
	}
}
