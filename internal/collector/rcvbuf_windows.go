package collector

import (
	"net"
	"syscall"
	"unsafe"
)

func readBufSize(c *net.UDPConn) int {
	rc, err := c.SyscallConn()
	if err != nil {
		return 0
	}
	var v int32
	l := int32(unsafe.Sizeof(v))
	rc.Control(func(fd uintptr) {
		syscall.Getsockopt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, (*byte)(unsafe.Pointer(&v)), &l)
	})
	return int(v)
}
