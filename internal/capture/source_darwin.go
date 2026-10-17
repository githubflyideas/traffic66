package capture

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// BSD packet filter ioctls (net/bpf.h).
const (
	biocGBLen     = 0x40044266
	biocSBLen     = 0xc0044266
	biocSetIf     = 0x8020426c
	biocImmediate = 0x80044270
	biocPromisc   = 0x20004269
	biocGStats    = 0x4008426f
	biocSSeeSent  = 0x80044277
)

type bpfDev struct {
	f     *os.File
	fd    uintptr
	blen  int
	drops uint64
}

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

func open(iface string) (source, error) {
	if _, err := net.InterfaceByName(iface); err != nil {
		return nil, err
	}
	var f *os.File
	var err error
	for i := 0; i < 256; i++ {
		f, err = os.OpenFile(fmt.Sprintf("/dev/bpf%d", i), os.O_RDONLY, 0)
		if err == nil {
			break
		}
		if !os.IsExist(err) && !isBusy(err) {
			if os.IsPermission(err) {
				return nil, fmt.Errorf("/dev/bpf: %w (run with sudo)", err)
			}
		}
	}
	if f == nil {
		return nil, fmt.Errorf("no free /dev/bpf device: %v", err)
	}
	fd := f.Fd()
	blen := uint32(1 << 20)
	ioctl(fd, biocSBLen, uintptr(unsafe.Pointer(&blen)))
	var ifr [32]byte
	copy(ifr[:16], iface)
	if err := ioctl(fd, biocSetIf, uintptr(unsafe.Pointer(&ifr[0]))); err != nil {
		f.Close()
		return nil, fmt.Errorf("BIOCSETIF %s: %w", iface, err)
	}
	one := uint32(1)
	ioctl(fd, biocImmediate, uintptr(unsafe.Pointer(&one)))
	ioctl(fd, biocSSeeSent, uintptr(unsafe.Pointer(&one)))
	ioctl(fd, biocPromisc, 0)
	var got uint32
	ioctl(fd, biocGBLen, uintptr(unsafe.Pointer(&got)))
	return &bpfDev{f: f, fd: fd, blen: int(got)}, nil
}

func isBusy(err error) bool {
	pe, ok := err.(*os.PathError)
	return ok && pe.Err == syscall.EBUSY
}

func (b *bpfDev) Method() string { return "BPF" }

func (b *bpfDev) Read(buf []byte, fn func([]byte, int)) error {
	if len(buf) < b.blen {
		buf = make([]byte, b.blen)
	}
	n, err := b.f.Read(buf[:b.blen])
	if err != nil {
		return err
	}
	p := buf[:n]
	for len(p) >= 18 {
		// struct bpf_hdr: timeval32 (8), caplen (4), datalen (4), hdrlen (2)
		caplen := int(binary.LittleEndian.Uint32(p[8:12]))
		datalen := int(binary.LittleEndian.Uint32(p[12:16]))
		hdrlen := int(binary.LittleEndian.Uint16(p[16:18]))
		if hdrlen+caplen > len(p) {
			break
		}
		fn(p[hdrlen:hdrlen+caplen], datalen)
		next := (hdrlen + caplen + 3) &^ 3
		if next > len(p) {
			break
		}
		p = p[next:]
	}
	return nil
}

func (b *bpfDev) Drops() uint64 {
	var st struct{ recv, drop uint32 }
	if ioctl(b.fd, biocGStats, uintptr(unsafe.Pointer(&st))) == nil {
		b.drops = uint64(st.drop)
	}
	return b.drops
}

func (b *bpfDev) Close() error { return b.f.Close() }

func Interfaces() ([]string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, i := range ifs {
		out = append(out, fmt.Sprintf("%-16s %s", i.Name, i.Flags))
	}
	return out, nil
}
