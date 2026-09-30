package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// Interfaces lists Npcap devices.
func Interfaces() ([]string, error) {
	if err := loadNpcap(); err != nil {
		return nil, err
	}
	var all *pcapIf
	errbuf := make([]byte, 256)
	if r, _, _ := pFindAll.Call(uintptr(unsafe.Pointer(&all)), uintptr(unsafe.Pointer(&errbuf[0]))); int32(r) != 0 {
		return nil, fmt.Errorf("pcap_findalldevs: %s", cstr(&errbuf[0]))
	}
	defer pFreeAll.Call(uintptr(unsafe.Pointer(all)))
	var out []string
	for d := all; d != nil; d = d.next {
		out = append(out, fmt.Sprintf("%s  %s", cstr(d.name), cstr(d.description)))
	}
	return out, nil
}

type npcap struct {
	h     uintptr
	drops uint64
}

func open(iface string) (source, error) {
	if err := loadNpcap(); err != nil {
		return nil, err
	}
	name := append([]byte(iface), 0)
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
