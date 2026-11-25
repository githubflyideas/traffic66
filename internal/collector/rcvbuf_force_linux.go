package collector

import (
	"net"
	"syscall"
)

// forceReadBuf asks for want bytes past net.core.rmem_max, which a process
// with CAP_NET_ADMIN (root, or the service) may do. Elsewhere it does
// nothing, and the size stays what SetReadBuffer got.
func forceReadBuf(c *net.UDPConn, want int) {
	rc, err := c.SyscallConn()
	if err != nil {
		return
	}
	rc.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUFFORCE, want)
	})
}
