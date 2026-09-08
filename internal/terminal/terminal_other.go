//go:build !windows

package terminal

import (
	"os"
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
