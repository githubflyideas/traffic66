package api

import (
	"syscall"
	"unsafe"
)

func diskFree(dir string) uint64 {
	k, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return 0
	}
	p, err := k.FindProc("GetDiskFreeSpaceExW")
	if err != nil {
		return 0
	}
	d, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0
	}
	var free uint64
	if r, _, _ := p.Call(uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&free)), 0, 0); r == 0 {
		return 0
	}
	return free
}
