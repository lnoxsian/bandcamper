//go:build windows

package terminal

import (
	"os"
	"syscall"
)

const (
	// ENABLE_VIRTUAL_TERMINAL_PROCESSING enables VT100 / ANSI escape sequence processing
	enableVirtualTerminalProcessing uint32 = 0x0004
	// ENABLE_PROCESSED_OUTPUT enables control character processing (\r, \b, \t, etc.)
	enableProcessedOutput uint32 = 0x0001
	// CP_UTF8 sets console code page to UTF-8
	codePageUTF8 uintptr = 65001
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode     = kernel32.NewProc("SetConsoleMode")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
)

func setConsoleMode(h syscall.Handle, mode uint32) error {
	r1, _, err := procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	if r1 == 0 {
		if err != nil && err != syscall.Errno(0) {
			return err
		}
		return syscall.EINVAL
	}
	return nil
}

func enableWindowsVT() bool {
	// 1. Set Windows console input and output code pages to UTF-8 (CP 65001)
	// so Unicode titles and artist names render accurately without question marks.
	_, _, _ = procSetConsoleOutputCP.Call(codePageUTF8)
	_, _, _ = procSetConsoleCP.Call(codePageUTF8)

	// 2. Enable Virtual Terminal Processing on stdout and stderr
	stdoutOk := enableVTForHandle(syscall.STD_OUTPUT_HANDLE)
	stderrOk := enableVTForHandle(syscall.STD_ERROR_HANDLE)

	return stdoutOk || stderrOk
}

func enableVTForHandle(stdHandleID int) bool {
	h, err := syscall.GetStdHandle(stdHandleID)
	if err != nil || h == syscall.InvalidHandle {
		return false
	}

	var mode uint32
	if err := syscall.GetConsoleMode(h, &mode); err != nil {
		// Not a console handle (e.g. redirected to a file or pipe)
		return false
	}

	// Already enabled
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}

	// Request virtual terminal processing + processed output
	newMode := mode | enableVirtualTerminalProcessing | enableProcessedOutput
	if err := setConsoleMode(h, newMode); err != nil {
		return false
	}

	return true
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	var mode uint32
	err := syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode)
	return err == nil
}
