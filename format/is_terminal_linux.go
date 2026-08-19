//go:build linux

package format

import (
	"os"
	"syscall"
	"unsafe"
)

func isTerminal(f *os.File) bool {
	var term syscall.Termios
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&term)),
		0, 0, 0,
	)
	return errno == 0
}
