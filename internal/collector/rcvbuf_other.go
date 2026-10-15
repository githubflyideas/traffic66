//go:build !linux && !darwin && !freebsd && !windows

package collector

import "net"

func readBufSize(*net.UDPConn) int { return 0 }
