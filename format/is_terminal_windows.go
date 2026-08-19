//go:build windows

package format

import (
	"os"
	"syscall"
)

func isTerminal(f *os.File) bool {
	var mode uint32
	err := syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode)
	return err == nil
}
