package store

import (
	"syscall"
	"unsafe"
)

type memStatusEx struct {
	length       uint32
	memoryLoad   uint32
	totalPhys    uint64
	availPhys    uint64
	totalPage    uint64
	availPage    uint64
	totalVirtual uint64
	availVirtual uint64
	availExt     uint64
}

func totalMemory() uint64 {
	k, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return 4 << 30
	}
	p, err := k.FindProc("GlobalMemoryStatusEx")
	if err != nil {
		return 4 << 30
	}
	var m memStatusEx
	m.length = uint32(unsafe.Sizeof(m))
	if r, _, _ := p.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 4 << 30
	}
	return m.totalPhys
}
