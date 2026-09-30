// Package sim generates realistic flow export traffic for demos and tests:
// a small company with an office LAN, a server zone, guest Wi-Fi, a DMZ and
// a branch office, seen by a NetFlow v9 core router, an sFlow switch, an
// IPFIX firewall and a NetFlow v5 branch router. All external addresses
// come from documentation and benchmarking ranges.
package sim

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/netip"
	"time"
)

var (
	CoreRouter = netip.MustParseAddr("10.0.0.1")
	Switch     = netip.MustParseAddr("10.0.0.2")
	Firewall   = netip.MustParseAddr("10.0.0.3")
	Branch     = netip.MustParseAddr("10.0.0.4")
)

// Interface indexes used by the exporters.
const (
	ifCoreISPA  = 3 // core router uplink A
	ifCoreISPB  = 4 // core router uplink B
	ifCoreLAN   = 10
	ifSwAccess  = 1 // switch: office access aggregate
	ifSwUplink  = 3 // switch: uplink to core
	ifSwServers = 7 // switch: server zone
	ifFwOutside = 1
	ifFwDMZ     = 2
	ifBranchWAN = 1
	ifBranchLAN = 2
)

type path uint8

const (
	viaCore path = 1 << iota
	viaSwitchUp
	viaSwitchSrv
	viaFirewall
	viaBranch
)

type ext struct {
	addr netip.Addr
	cc   string
	asn  uint32
	org  string
}

// External endpoints (documentation / benchmarking address space).
var (
	extBackup  = ext{netip.MustParseAddr("198.51.100.10"), "JP", 64601, "Demo Backup DC"}
	extCDN     = ext{netip.MustParseAddr("198.51.100.130"), "US", 64602, "Demo CDN"}
	extVideo   = ext{netip.MustParseAddr("198.18.1.10"), "JP", 64603, "Demo Video Meetings"}
	extCode    = ext{netip.MustParseAddr("198.18.2.20"), "US", 64604, "Demo Code Hosting"}
	extMirror  = ext{netip.MustParseAddr("198.18.3.30"), "CN", 64605, "Demo Mirror"}
	extSaaS    = ext{netip.MustParseAddr("198.18.4.40"), "SG", 64606, "Demo SaaS"}
	extUpdates = ext{netip.MustParseAddr("198.18.5.50"), "DE", 64607, "Demo Updates"}
	extDesign  = ext{netip.MustParseAddr("198.18.6.60"), "US", 64608, "Demo Design Cloud"}
	extC2      = ext{netip.MustParseAddr("203.0.113.200"), "RU", 64609, "Demo Hosting"}
	extScanner = ext{netip.MustParseAddr("192.0.2.66"), "NL", 64610, "Demo Scanner Net"}
	extVisitor = ext{netip.MustParseAddr("198.18.7.0"), "JP", 64611, "Demo Consumer ISP"}
	dmzWeb     = netip.MustParseAddr("203.0.113.5")
	dnsServer  = netip.MustParseAddr("10.20.0.53")
	fileServer = netip.MustParseAddr("10.20.0.15")
	backupHost = netip.MustParseAddr("10.20.0.8")
	financePC  = netip.MustParseAddr("10.10.3.27")
	designNAS  = netip.MustParseAddr("10.10.5.40")
)

var externals = []ext{extBackup, extCDN, extVideo, extCode, extMirror, extSaaS, extUpdates, extDesign, extC2, extScanner, extVisitor}

// ASNTable returns a TSV table (range_start range_end asn cc org) covering
// the demo's external addresses.
func ASNTable() string {
	s := ""
	for _, e := range externals {
		lo, hi := e.addr, e.addr
		if e == extVisitor {
			hi = netip.MustParseAddr("198.18.7.255")
		}
		s += fmt.Sprintf("%s\t%s\t%d\t%s\t%s\n", lo, hi, e.asn, e.cc, e.org)
	}
	return s
}

// ThreatList returns the demo threat list entries.
func ThreatList() (c2, scanners string) {
	return "# demo list\n" + extC2.addr.String() + "\n", "# demo list\n" + extScanner.addr.String() + "\n"
}

// Inventory returns the demo inventory text.
func Inventory() string {
	return `# traffic66 demo inventory
net 10.10.0.0/16 Office LAN
net 10.20.0.0/24 Server zone
net 10.30.0.0/16 Guest Wi-Fi
net 10.40.0.0/16 Branch office
net 203.0.113.0/28 DMZ
device 10.0.0.1 Core router
device 10.0.0.2 Aggregation switch
device 10.0.0.3 Edge firewall
device 10.0.0.4 Branch router
iface 10.0.0.1 3 ISP-A uplink speed=1000000000
iface 10.0.0.1 4 ISP-B uplink speed=500000000
iface 10.0.0.2 1 Office access speed=10000000000
iface 10.0.0.2 3 Uplink to core speed=10000000000
iface 10.0.0.2 7 Server zone speed=10000000000
host 10.10.3.27 Finance PC
host 10.10.5.40 Design NAS
host 10.20.0.8 Backup server
host 10.20.0.15 File server
host 10.20.0.53 DNS server
host 203.0.113.5 Public website
# interface counters of the NetFlow/IPFIX devices, read over SNMP
snmp 10.0.0.1 demo 127.0.0.1:16161
snmp 10.0.0.3 demo 127.0.0.1:16163
snmp 10.0.0.4 demo 127.0.0.1:16164
`
}

// conv is a class of traffic between a pool of clients and one server.
type conv struct {
	name     string
	clients  func(r *rand.Rand) netip.Addr
	server   netip.Addr
	proto    uint8
	port     uint16
	mbps     func(h float64, today bool) float64 // mean total rate
	upShare  float64                             // share of bytes client->server
	ipSize   int                                 // mean IP packet size
	via      path
	sessions int // concurrent sessions at full rate
	coreIf   uint32
}

func bell(h, center, width float64) float64 { return math.Exp(-math.Pow((h-center)/width, 2)) }
func workday(h float64) float64             { return bell(h, 10.8, 2.6) + 0.9*bell(h, 15.8, 2.3) }

func pool(prefix string, n int) func(r *rand.Rand) netip.Addr {
	p := netip.MustParsePrefix(prefix)
	base := p.Addr().As4()
	return func(r *rand.Rand) netip.Addr {
		i := r.IntN(n) + 2
		b := base
		b[2] += byte(i / 250)
		b[3] = byte(i%250 + 1)
		return netip.AddrFrom4(b)
	}
}

func one(a netip.Addr) func(*rand.Rand) netip.Addr { return func(*rand.Rand) netip.Addr { return a } }

// incident window relative to "now" so it always falls in the last day.
type model struct {
	convs         []conv
	incidentStart time.Time
	incidentEnd   time.Time
}

func newModel(now time.Time) *model {
	m := &model{incidentStart: now.Add(-5 * time.Hour).Truncate(time.Minute), incidentEnd: now.Add(-3*time.Hour - 13*time.Minute).Truncate(time.Minute)}
	office := pool("10.10.0.0/16", 900)
	guest := pool("10.30.0.0/16", 200)
	branch := pool("10.40.0.0/16", 120)
	visitors := pool("198.18.7.0/24", 240)
	m.convs = []conv{
		{name: "web", clients: office, server: extCDN.addr, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 12 + 260*workday(h) },
			upShare: 0.06, ipSize: 1180, via: viaCore | viaSwitchUp, sessions: 900, coreIf: ifCoreISPA},
		{name: "saas", clients: office, server: extSaaS.addr, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 4 + 70*workday(h) },
			upShare: 0.2, ipSize: 900, via: viaCore | viaSwitchUp, sessions: 400, coreIf: ifCoreISPB},
		{name: "mirror", clients: office, server: extMirror.addr, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 2 + 40*workday(h) },
			upShare: 0.03, ipSize: 1400, via: viaCore | viaSwitchUp, sessions: 120, coreIf: ifCoreISPA},
		{name: "video", clients: office, server: extVideo.addr, proto: 17, port: 8801, mbps: func(h float64, _ bool) float64 {
			return 55*bell(h, 10, 1.1) + 70*bell(h, 14, 1.4) + 45*bell(h, 16.5, 0.9)
		}, upShare: 0.45, ipSize: 1050, via: viaCore | viaSwitchUp, sessions: 60, coreIf: ifCoreISPB},
		{name: "code", clients: pool("10.10.2.0/24", 40), server: extCode.addr, proto: 6, port: 22, mbps: func(h float64, _ bool) float64 { return 1 + 22*bell(h, 13, 4) },
			upShare: 0.35, ipSize: 700, via: viaCore | viaSwitchUp, sessions: 50, coreIf: ifCoreISPA},
		{name: "updates", clients: office, server: extUpdates.addr, proto: 6, port: 80, mbps: func(h float64, _ bool) float64 { return 3 + 25*bell(h, 12.5, 1.5) },
			upShare: 0.02, ipSize: 1450, via: viaCore | viaSwitchUp, sessions: 80, coreIf: ifCoreISPA},
		{name: "backup", clients: one(backupHost), server: extBackup.addr, proto: 6, port: 873, mbps: func(h float64, _ bool) float64 { return 240*bell(h, 2.5, 0.9) + 2 },
			upShare: 0.97, ipSize: 1440, via: viaCore | viaSwitchSrv, sessions: 4, coreIf: ifCoreISPA},
		{name: "smb", clients: office, server: fileServer, proto: 6, port: 445, mbps: func(h float64, _ bool) float64 { return 5 + 75*workday(h) },
			upShare: 0.35, ipSize: 1300, via: viaSwitchSrv, sessions: 300},
		{name: "dns", clients: office, server: dnsServer, proto: 17, port: 53, mbps: func(h float64, _ bool) float64 { return 0.3 + 2*workday(h) },
			upShare: 0.4, ipSize: 110, via: viaSwitchSrv, sessions: 600},
		{name: "guest", clients: guest, server: extCDN.addr, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 3 + 45*workday(h) },
			upShare: 0.05, ipSize: 1200, via: viaCore, sessions: 200, coreIf: ifCoreISPB},
		{name: "dmz", clients: visitors, server: dmzWeb, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 6 + 30*bell(h, 20, 3) },
			upShare: 0.08, ipSize: 1100, via: viaFirewall, sessions: 150},
		{name: "scan", clients: one(extScanner.addr), server: dmzWeb, proto: 6, port: 22, mbps: func(h float64, _ bool) float64 { return 0.05 },
			upShare: 0.6, ipSize: 60, via: viaFirewall, sessions: 40},
		{name: "branch", clients: branch, server: extSaaS.addr, proto: 6, port: 443, mbps: func(h float64, _ bool) float64 { return 2 + 35*workday(h) },
			upShare: 0.18, ipSize: 950, via: viaBranch, sessions: 90},
		// grows this week: the design NAS started syncing with a design cloud
		{name: "design", clients: one(designNAS), server: extDesign.addr, proto: 6, port: 443, mbps: func(h float64, today bool) float64 {
			if !today {
				return 1
			}
			return 1 + 52*workday(h)
		}, upShare: 0.6, ipSize: 1350, via: viaCore | viaSwitchUp, sessions: 6, coreIf: ifCoreISPA},
	}
	return m
}

// incident: the finance PC uploads to the C2 address.
func (m *model) incidentMbps(t time.Time) float64 {
	if t.Before(m.incidentStart) || !t.Before(m.incidentEnd) {
		return 0
	}
	x := t.Sub(m.incidentStart).Seconds() / m.incidentEnd.Sub(m.incidentStart).Seconds()
	return 150 * math.Sin(math.Pi*x) * (0.85 + 0.3*x)
}

func hourOf(t time.Time) float64 {
	l := t.Local()
	return float64(l.Hour()) + float64(l.Minute())/60 + float64(l.Second())/3600
}

// SNMPAgents maps each simulated device to its demo agent address.
var SNMPAgents = map[netip.Addr]string{CoreRouter: "127.0.0.1:16161", Firewall: "127.0.0.1:16163", Branch: "127.0.0.1:16164"}

// IfAlias is the interface description a demo device reports.
func IfAlias(dev netip.Addr, idx uint32) (name, alias string) {
	switch dev {
	case CoreRouter:
		return fmt.Sprintf("Gi0/0/%d", idx), map[uint32]string{ifCoreISPA: "ISP-A uplink", ifCoreISPB: "ISP-B uplink", ifCoreLAN: "Core LAN"}[idx]
	case Firewall:
		return fmt.Sprintf("port%d", idx), map[uint32]string{ifFwOutside: "Outside", ifFwDMZ: "DMZ"}[idx]
	case Branch:
		return fmt.Sprintf("Fa0/%d", idx), map[uint32]string{ifBranchWAN: "Branch WAN", ifBranchLAN: "Branch LAN"}[idx]
	}
	return fmt.Sprintf("if%d", idx), ""
}
