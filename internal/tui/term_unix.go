//go:build linux || darwin

package tui

import (
	"os"
	"syscall"
	"unsafe"
)

type rawState struct{ t syscall.Termios }

func makeRaw() (*rawState, error) {
	fd := os.Stdin.Fd()
	var old syscall.Termios
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, ioctlGet, uintptr(unsafe.Pointer(&old))); e != 0 {
		return nil, e
	}
	t := old
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	t.Oflag &^= syscall.OPOST
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB
	t.Cflag |= syscall.CS8
	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 0
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, ioctlSet, uintptr(unsafe.Pointer(&t))); e != 0 {
		return nil, e
	}
	return &rawState{old}, nil
}

func (s *rawState) restore() {
	syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), ioctlSet, uintptr(unsafe.Pointer(&s.t)))
}

func size() (int, int) { return termSize(int(os.Stdout.Fd())) }

func isTerminal() bool {
	var t syscall.Termios
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), ioctlGet, uintptr(unsafe.Pointer(&t)))
	return e == 0
}
