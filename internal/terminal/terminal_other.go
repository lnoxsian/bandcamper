//go:build !windows

package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

func enableWindowsVT() bool {
	return false
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTerminalSize() (int, int) {
	var ws winsize
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if err == 0 && ws.Row > 0 && ws.Col > 0 {
		return int(ws.Col), int(ws.Row)
	}
	return 80, 24
}
