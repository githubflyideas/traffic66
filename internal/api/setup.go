package api

import (
	"net"
	"sort"
	"strings"
)

// ListenInfo is one UDP collector: its name (sflow, netflow, ipfix …) and port.
type ListenInfo struct {
	Name string `json:"name"`
	Port string `json:"port"`
}

// LocalIPs lists this machine's IPv4 addresses that a device could send
// to: up interfaces, no loopback or link-local, at most four, private
// ones first.
func LocalIPs() []string {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []net.IP
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 || virtualIface(i.Name) {
			continue
		}
		as, _ := i.Addrs()
		for _, a := range as {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := n.IP.To4()
			if ip == nil || ip.IsLinkLocalUnicast() || ip.IsLoopback() {
				continue
			}
			out = append(out, ip)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].IsPrivate() && !out[b].IsPrivate() })
	if len(out) > 4 {
		out = out[:4]
	}
	s := make([]string, len(out))
	for i, ip := range out {
		s[i] = ip.String()
	}
	return s
}

// virtualIface tells container and VM bridges, which no device sends to.
func virtualIface(name string) bool {
	for _, p := range []string{"docker", "br-", "veth", "virbr", "cni", "flannel", "vmnet", "vEthernet"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
