package capture

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Npcap (wpcap.dll) is loaded at run time; it is not bundled.
var (
	npcapOnce                                              sync.Once
	npcapErr                                               error
	pOpenLive, pNextEx, pClose, pFindAll, pFreeAll, pStats *syscall.Proc
)

func loadNpcap() error {
	npcapOnce.Do(func() {
		dir := filepath.Join(os.Getenv("SystemRoot"), "System32", "Npcap")
		if k, err := syscall.LoadDLL("kernel32.dll"); err == nil {
			if p, err := k.FindProc("SetDllDirectoryW"); err == nil {
				d, _ := syscall.UTF16PtrFromString(dir)
				p.Call(uintptr(unsafe.Pointer(d)))
			}
		}
		dll, err := syscall.LoadDLL(filepath.Join(dir, "wpcap.dll"))
		if err != nil {
			dll, err = syscall.LoadDLL("wpcap.dll")
		}
		if err != nil {
			npcapErr = errors.New("Npcap is not installed; get it from https://npcap.com")
			return
		}
		find := func(n string) *syscall.Proc {
			p, e := dll.FindProc(n)
			if e != nil && npcapErr == nil {
				npcapErr = e
			}
			return p
		}
		pOpenLive, pNextEx, pClose = find("pcap_open_live"), find("pcap_next_ex"), find("pcap_close")
		pFindAll, pFreeAll, pStats = find("pcap_findalldevs"), find("pcap_freealldevs"), find("pcap_stats")
	})
	return npcapErr
}

func cstr(p *byte) string {
	if p == nil {
		return ""
	}
	var b []byte
	for i := 0; i < 4096; i++ {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(p), i))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}

type pcapIf struct {
	next        *pcapIf
	name        *byte
	description *byte
	addresses   unsafe.Pointer
	flags       uint32
}

// devices lists Npcap devices with the Windows connection name ("Wi-Fi",
// "Ethernet") and IPv4 address of each, matched by the adapter GUID.
func devices() ([]Device, error) {
	if err := loadNpcap(); err != nil {
		return nil, err
	}
	var all *pcapIf
	errbuf := make([]byte, 256)
	if r, _, _ := pFindAll.Call(uintptr(unsafe.Pointer(&all)), uintptr(unsafe.Pointer(&errbuf[0]))); int32(r) != 0 {
		return nil, fmt.Errorf("pcap_findalldevs: %s", cstr(&errbuf[0]))
	}
	defer pFreeAll.Call(uintptr(unsafe.Pointer(all)))
	byGUID := adaptersByGUID()
	var out []Device
	for d := all; d != nil; d = d.next {
		dev := Device{Num: len(out) + 1, Name: cstr(d.name), Desc: cstr(d.description)}
		if i := strings.Index(dev.Name, "{"); i >= 0 {
			if a, ok := byGUID[strings.ToUpper(dev.Name[i:])]; ok {
				dev.Friendly, dev.Addrs = a.name, a.addrs
			}
		}
		if dev.Friendly == "" {
			if strings.HasSuffix(dev.Name, "Loopback") {
				dev.Friendly = "Loopback"
			} else {
				dev.Friendly = dev.Name
			}
		}
		out = append(out, dev)
	}
	return out, nil
}

type adapter struct {
	name  string
	addrs []string
}

// adaptersByGUID maps "{GUID}" to the connection name and IPv4 addresses.
func adaptersByGUID() map[string]adapter {
	out := map[string]adapter{}
	size := uint32(16 << 10)
	var buf []byte
	for i := 0; i < 3; i++ {
		buf = make([]byte, size)
		err := syscall.GetAdaptersInfo((*syscall.IpAdapterInfo)(unsafe.Pointer(&buf[0])), &size)
		if err == nil {
			break
		}
		if err != syscall.ERROR_BUFFER_OVERFLOW {
			return out
		}
		buf = nil
	}
	if buf == nil {
		return out
	}
	ifs, _ := net.Interfaces()
	byIndex := map[int]net.Interface{}
	for _, i := range ifs {
		byIndex[i.Index] = i
	}
	for ai := (*syscall.IpAdapterInfo)(unsafe.Pointer(&buf[0])); ai != nil; ai = ai.Next {
		guid := strings.ToUpper(cstr(&ai.AdapterName[0]))
		a := adapter{}
		if ni, ok := byIndex[int(ai.Index)]; ok {
			a.name = ni.Name
			if addrs, err := ni.Addrs(); err == nil {
				for _, ad := range addrs {
					if ipn, ok := ad.(*net.IPNet); ok && ipn.IP.To4() != nil {
						a.addrs = append(a.addrs, ipn.IP.String())
					}
				}
			}
		}
		for ip := &ai.IpAddressList; ip != nil && len(a.addrs) == 0; ip = ip.Next {
			if s := cstr(&ip.IpAddress.String[0]); s != "" && s != "0.0.0.0" {
				a.addrs = append(a.addrs, s)
			}
		}
		if a.name == "" {
			a.name = cstr(&ai.Description[0])
		}
		out[guid] = a
	}
	return out
}

// Interfaces lists Npcap devices for "traffic66 interfaces".
func Interfaces() ([]string, error) {
	ds, err := devices()
	if err != nil {
		return nil, err
	}
	return formatDevices(ds), nil
}

type npcap struct {
	h     uintptr
	drops uint64
}

func open(iface string) (source, error) {
	if err := loadNpcap(); err != nil {
		return nil, err
	}
	ds, err := devices()
	if err != nil {
		return nil, err
	}
	dev, err := resolveDevice(iface, ds)
	if err != nil {
		return nil, err
	}
	name := append([]byte(dev), 0)
	errbuf := make([]byte, 256)
	h, _, _ := pOpenLive.Call(uintptr(unsafe.Pointer(&name[0])), 256, 1, 200, uintptr(unsafe.Pointer(&errbuf[0])))
	if h == 0 {
		return nil, fmt.Errorf("pcap_open_live %s: %s (use a name from 'traffic66 interfaces')", iface, cstr(&errbuf[0]))
	}
	return &npcap{h: h}, nil
}

func (n *npcap) Method() string { return "Npcap" }

// struct pcap_pkthdr on Windows: timeval (2 x 32-bit), caplen, len.
type pkthdr struct {
	sec, usec    int32
	caplen, wlen uint32
}

func (n *npcap) Read(buf []byte, fn func([]byte, int)) error {
	for i := 0; i < 4096; i++ {
		var hdr *pkthdr
		var data *byte
		r, _, _ := pNextEx.Call(n.h, uintptr(unsafe.Pointer(&hdr)), uintptr(unsafe.Pointer(&data)))
		switch int32(r) {
		case 1:
			c := int(hdr.caplen)
			if c > len(buf) {
				c = len(buf)
			}
			copy(buf, unsafe.Slice(data, c))
			fn(buf[:c], int(hdr.wlen))
		case 0:
			return nil
		default:
			return errors.New("pcap_next_ex failed")
		}
	}
	return nil
}

func (n *npcap) Drops() uint64 {
	var st struct{ recv, drop, ifdrop uint32 }
	if r, _, _ := pStats.Call(n.h, uintptr(unsafe.Pointer(&st))); int32(r) == 0 {
		n.drops = uint64(st.drop) + uint64(st.ifdrop)
	}
	return n.drops
}

func (n *npcap) Close() error {
	pClose.Call(n.h)
	return nil
}
