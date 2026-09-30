package enrich

import (
	"fmt"

	"github.com/githubflyideas/traffic66/internal/flow"
)

type svcKey struct {
	proto uint8
	port  uint16
}

var services = map[svcKey]string{}

func init() {
	tcp := map[uint16]string{
		20: "FTP-data", 21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP", 53: "DNS", 80: "HTTP", 88: "Kerberos",
		110: "POP3", 111: "RPC", 135: "MS-RPC", 139: "NetBIOS", 143: "IMAP", 179: "BGP", 389: "LDAP",
		443: "HTTPS", 445: "SMB", 465: "SMTPS", 514: "Syslog", 548: "AFP", 587: "SMTP-submission", 636: "LDAPS",
		873: "rsync", 993: "IMAPS", 995: "POP3S", 1080: "SOCKS", 1194: "OpenVPN", 1433: "MS-SQL", 1521: "Oracle",
		1723: "PPTP", 1883: "MQTT", 2049: "NFS", 2375: "Docker", 2376: "Docker", 3128: "HTTP-proxy",
		3260: "iSCSI", 3306: "MySQL", 3389: "RDP", 5060: "SIP", 5061: "SIP-TLS", 5222: "XMPP", 5432: "PostgreSQL",
		5671: "AMQPS", 5672: "AMQP", 5900: "VNC", 5985: "WinRM", 5986: "WinRM", 6379: "Redis", 6443: "Kubernetes",
		8080: "HTTP-alt", 8443: "HTTPS-alt", 8883: "MQTTS", 9000: "HTTP-alt", 9092: "Kafka", 9100: "Printer",
		9200: "Elasticsearch", 11211: "Memcached", 27017: "MongoDB",
	}
	udp := map[uint16]string{
		53: "DNS", 67: "DHCP", 68: "DHCP", 69: "TFTP", 123: "NTP", 137: "NetBIOS", 138: "NetBIOS", 161: "SNMP",
		162: "SNMP-trap", 443: "QUIC", 500: "IPsec", 514: "Syslog", 520: "RIP", 1194: "OpenVPN", 1812: "RADIUS",
		1813: "RADIUS", 1900: "SSDP", 2055: "NetFlow", 3478: "STUN", 3479: "STUN", 4500: "IPsec-NAT",
		4739: "IPFIX", 4789: "VXLAN", 5060: "SIP", 5353: "mDNS", 6081: "GENEVE", 6343: "sFlow", 8801: "Zoom",
		3480: "Teams", 51820: "WireGuard",
	}
	for p, n := range tcp {
		services[svcKey{6, p}] = n
	}
	for p, n := range udp {
		services[svcKey{17, p}] = n
	}
}

var protoNames = map[uint8]string{1: "ICMP", 2: "IGMP", 4: "IPIP", 6: "TCP", 17: "UDP", 41: "IPv6", 47: "GRE", 50: "ESP", 51: "AH", 58: "ICMPv6", 89: "OSPF", 103: "PIM", 112: "VRRP", 132: "SCTP"}

// ProtoName returns the IANA protocol name or its number.
func ProtoName(p uint8) string {
	if n, ok := protoNames[p]; ok {
		return n
	}
	return fmt.Sprintf("%d", p)
}

func known(proto uint8, port uint16) bool {
	_, ok := services[svcKey{proto, port}]
	return ok
}

// Role decides which side of a flow is the client. It returns true when the
// source is the client.
func Role(r *flow.Record) bool {
	switch r.Proto {
	case 6:
		if r.TCPFlags&0x12 == 0x02 {
			return true // SYN without ACK: source opened the connection
		}
		if r.TCPFlags&0x12 == 0x12 {
			return false // SYN+ACK: source is answering
		}
	case 17, 132:
	default:
		return true
	}
	sk, dk := known(r.Proto, r.SrcPort), known(r.Proto, r.DstPort)
	switch {
	case dk && !sk:
		return true
	case sk && !dk:
		return false
	}
	if r.DstPort == r.SrcPort {
		return true
	}
	return r.DstPort < r.SrcPort
}

// App names the service of a flow given its server port.
func App(proto uint8, port uint16) string {
	if proto != 6 && proto != 17 && proto != 132 {
		return ProtoName(proto)
	}
	if n, ok := services[svcKey{proto, port}]; ok {
		return n
	}
	return fmt.Sprintf("%s/%d", ProtoName(proto), port)
}
