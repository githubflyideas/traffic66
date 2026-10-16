package assets

import (
	"net/netip"
	"sort"
)

// sortSubnets puts the narrowest network first, so that looking up which
// network an address belongs to finds the most specific one, and the list
// reads from the smallest to the largest.
func sortSubnets(v []Subnet) {
	key := func(s Subnet) (netip.Prefix, bool) {
		p, err := netip.ParsePrefix(s.Prefix)
		return p, err == nil
	}
	sort.SliceStable(v, func(i, j int) bool {
		a, aok := key(v[i])
		b, bok := key(v[j])
		if !aok || !bok {
			return aok
		}
		if a.Bits() != b.Bits() {
			return a.Bits() > b.Bits()
		}
		return a.Addr().Less(b.Addr())
	})
}

// sortAddrs sorts by address, not by the text of it, so that .9 comes
// before .10.
func sortAddrs(v []Addr) {
	sort.SliceStable(v, func(i, j int) bool {
		a, aerr := netip.ParseAddr(v[i].IP)
		b, berr := netip.ParseAddr(v[j].IP)
		if aerr != nil || berr != nil {
			return v[i].IP < v[j].IP
		}
		return a.Less(b)
	})
}
