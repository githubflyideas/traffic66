//go:build !linux

package collector

import "net"

func forceReadBuf(*net.UDPConn, int) {}
