package tui

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32   = syscall.NewLazyDLL("kernel32.dll")
	getMode    = kernel32.NewProc("GetConsoleMode")
	setMode    = kernel32.NewProc("SetConsoleMode")
	getBufInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	setCP      = kernel32.NewProc("SetConsoleOutputCP")
	setInCP    = kernel32.NewProc("SetConsoleCP")
)

const (
	enableProcessedInput  = 0x0001
	enableLineInput       = 0x0002
	enableEchoInput       = 0x0004
	enableVTInput         = 0x0200
	enableVTProcessing    = 0x0004
	disableNewlineAutoRet = 0x0008
)

type rawState struct{ in, out uint32 }

func makeRaw() (*rawState, error) {
	var in, out uint32
	hin, hout := os.Stdin.Fd(), os.Stdout.Fd()
	if r, _, e := getMode.Call(hin, uintptr(unsafe.Pointer(&in))); r == 0 {
		return nil, e
	}
	getMode.Call(hout, uintptr(unsafe.Pointer(&out)))
	st := &rawState{in, out}
	setMode.Call(hin, uintptr(in&^(enableProcessedInput|enableLineInput|enableEchoInput)|enableVTInput))
	setMode.Call(hout, uintptr(out|enableVTProcessing|disableNewlineAutoRet))
	setCP.Call(65001)
	setInCP.Call(65001)
	return st, nil
}

func (s *rawState) restore() {
	setMode.Call(os.Stdin.Fd(), uintptr(s.in))
	setMode.Call(os.Stdout.Fd(), uintptr(s.out))
}

type coord struct{ x, y int16 }
type smallRect struct{ left, top, right, bottom int16 }
type bufInfo struct {
	size       coord
	cursor     coord
	attrs      uint16
	window     smallRect
	maxWinSize coord
}

func size() (int, int) {
	var bi bufInfo
	if r, _, _ := getBufInfo.Call(os.Stdout.Fd(), uintptr(unsafe.Pointer(&bi))); r == 0 {
		return 100, 32
	}
	return int(bi.window.right-bi.window.left) + 1, int(bi.window.bottom-bi.window.top) + 1
}

func isTerminal() bool {
	var m uint32
	r, _, _ := getMode.Call(os.Stdin.Fd(), uintptr(unsafe.Pointer(&m)))
	return r != 0
}
