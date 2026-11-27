package flow

import "testing"

func TestSourceString(t *testing.T) {
	for _, tc := range []struct {
		s    Source
		want string
	}{
		{SrcSFlow, "sFlow v5"},
		{SrcNetFlow5, "NetFlow v5"},
		{SrcNetFlow9, "NetFlow v9"},
		{SrcIPFIX, "IPFIX"},
		{SrcCapture, "capture"},
		{SrcUnknown, "unknown"},
		{Source(99), "unknown"},
	} {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("Source(%d).String() = %q, want %q", uint8(tc.s), got, tc.want)
		}
	}
}

func TestEncapString(t *testing.T) {
	for _, tc := range []struct {
		e    Encap
		want string
	}{
		{EncapNone, ""},
		{EncapGRE, "GRE"},
		{EncapIPIP, "IPIP"},
		{Encap6in4, "6in4"},
		{EncapIP6in6, "IPv6-in-IPv6"},
		{EncapVXLAN, "VXLAN"},
		{EncapGENEVE, "GENEVE"},
		{EncapMPLS, "MPLS"},
		{Encap(200), ""}, // beyond the table: no name, not a panic
	} {
		if got := tc.e.String(); got != tc.want {
			t.Errorf("Encap(%d).String() = %q, want %q", uint8(tc.e), got, tc.want)
		}
	}
}
