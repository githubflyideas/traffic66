package api

import (
	"fmt"
	"net"
	"strings"
)

// TrustedProxies is the set of reverse proxies whose X-Forwarded-For
// header is believed: the direct peers named by -trusted-proxies, as
// single addresses or CIDR ranges. The zero value trusts nobody, and
// then X-Forwarded-For is ignored entirely — a client can write
// anything in it, so believing it from just anyone would let a caller
// pick the address its failed logins are charged to, and the address
// a read-only account is rate-limited by.
type TrustedProxies struct {
	ips  map[string]bool
	nets []*net.IPNet
}

// ParseTrustedProxies reads the -trusted-proxies value: proxy
// addresses and networks, comma separated, each an IP address or a
// CIDR range. Empty trusts nobody.
func ParseTrustedProxies(list string) (TrustedProxies, error) {
	var tp TrustedProxies
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, n, err := net.ParseCIDR(part)
			if err != nil {
				return tp, fmt.Errorf("-trusted-proxies %q: %v; give IP addresses or CIDR ranges, comma separated", part, err)
			}
			tp.nets = append(tp.nets, n)
			continue
		}
		ip := net.ParseIP(part)
		if ip == nil {
			return tp, fmt.Errorf("-trusted-proxies %q: not an IP address or CIDR range", part)
		}
		if tp.ips == nil {
			tp.ips = map[string]bool{}
		}
		tp.ips[ip.String()] = true
	}
	return tp, nil
}

// trusted reports whether the direct peer at ip is a named proxy.
func (tp TrustedProxies) trusted(ip string) bool {
	if tp.ips[ip] {
		return true
	}
	if len(tp.nets) == 0 {
		return false
	}
	p := net.ParseIP(ip)
	if p == nil {
		return false
	}
	for _, n := range tp.nets {
		if n.Contains(p) {
			return true
		}
	}
	return false
}
