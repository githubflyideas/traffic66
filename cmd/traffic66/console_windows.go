package main

import (
	"syscall"
	"unsafe"
)

// startedByDoubleClick reports whether this process has a console window
// of its own, which is the case when it was started from Explorer rather
// than from a terminal.
func startedByDoubleClick() bool {
	p := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")
	var ids [4]uint32
	n, _, _ := p.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	return n == 1
}
