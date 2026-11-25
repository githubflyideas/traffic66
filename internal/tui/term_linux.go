package tui

import (
	"syscall"
	"unsafe"
)

const (
	ioctlGet = syscall.TCGETS
	ioctlSet = syscall.TCSETS
)

type winsize struct{ rows, cols, x, y uint16 }

func termSize(fd int) (int, int) {
	var ws winsize
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws))); e != 0 || ws.cols == 0 {
		return 100, 32
	}
	return int(ws.cols), int(ws.rows)
}
