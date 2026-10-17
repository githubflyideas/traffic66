//go:build linux || darwin || freebsd

package collector

import (
	"net"
	"runtime"
	"syscall"
)

func readBufSize(c *net.UDPConn) int {
	rc, err := c.SyscallConn()
	if err != nil {
		return 0
	}
	v := 0
	rc.Control(func(fd uintptr) {
		v, _ = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	})
	if runtime.GOOS == "linux" {
		v /= 2 // Linux reports double the usable size
	}
	return v
}
