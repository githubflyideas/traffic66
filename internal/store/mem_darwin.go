package store

import (
	"encoding/binary"
	"syscall"
)

func totalMemory() uint64 {
	s, err := syscall.Sysctl("hw.memsize")
	if err != nil || len(s) < 4 {
		return 4 << 30
	}
	b := []byte(s)
	for len(b) < 8 {
		b = append(b, 0)
	}
	return binary.LittleEndian.Uint64(b[:8])
}
