package capture

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

const ethPAll = 0x0003

func htons(v uint16) uint16 { return v<<8 | v>>8 }

type afPacket struct {
	fd    int
	drops uint64
}

type packetMreq struct {
	ifindex int32
	typ     uint16
	alen    uint16
	addr    [8]byte
}

type tpacketStats struct{ packets, drops uint32 }

func open(iface string) (source, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(ethPAll)))
	if err != nil {
		return nil, fmt.Errorf("AF_PACKET socket: %w (needs root or CAP_NET_RAW)", err)
	}
	if err := syscall.Bind(fd, &syscall.SockaddrLinklayer{Protocol: htons(ethPAll), Ifindex: ifi.Index}); err != nil {
		syscall.Close(fd)
		return nil, err
	}
	mreq := packetMreq{ifindex: int32(ifi.Index), typ: syscall.PACKET_MR_PROMISC}
	syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.SOL_PACKET, syscall.PACKET_ADD_MEMBERSHIP,
		uintptr(unsafe.Pointer(&mreq)), unsafe.Sizeof(mreq), 0)
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 16<<20)
	tv := syscall.Timeval{Sec: 0, Usec: 200000}
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)
	return &afPacket{fd: fd}, nil
}

func (a *afPacket) Method() string { return "AF_PACKET" }

func (a *afPacket) Read(buf []byte, fn func([]byte, int)) error {
	n, _, err := syscall.Recvfrom(a.fd, buf[:65536], syscall.MSG_TRUNC)
	if err != nil {
		if err == syscall.EAGAIN || err == syscall.EINTR {
			return nil
		}
		return err
	}
	c := n
	if c > 65536 {
		c = 65536
	}
	fn(buf[:c], n)
	return nil
}

func (a *afPacket) Drops() uint64 {
	var st tpacketStats
	l := uint32(unsafe.Sizeof(st))
	_, _, e := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(a.fd), syscall.SOL_PACKET, 6, /* PACKET_STATISTICS */
		uintptr(unsafe.Pointer(&st)), uintptr(unsafe.Pointer(&l)), 0)
	if e == 0 {
		a.drops += uint64(st.drops) // counters reset on read
	}
	return a.drops
}

func (a *afPacket) Close() error { return syscall.Close(a.fd) }

// Interfaces lists capture-capable interfaces.
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
