package tui

import "testing"

func TestPortLabel(t *testing.T) {
	for k, want := range map[string]string{"443/tcp": "443/tcp", "0/1": "ICMP", "0/58": "ICMPv6", "0/99": "IP 99", "53/udp": "53/udp"} {
		if got := portLabel(k); got != want {
			t.Errorf("portLabel(%q) = %q, want %q", k, got, want)
		}
	}
}

// A right-aligned heading as wide as its column keeps a gap from the
// heading before it.
func TestLayoutHeadingGap(t *testing.T) {
	ws := layout([]column{{head: "Δ", width: 6, right: true}, {head: "التدفقات", width: 8, right: true}, {head: "Flows", width: 8, right: true}}, 80)
	if ws[0] != 6 || ws[1] != 9 || ws[2] != 8 {
		t.Fatalf("widths %v, want [6 9 8]", ws)
	}
}
