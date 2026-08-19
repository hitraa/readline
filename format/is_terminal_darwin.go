//go:build darwin

package format

import (
	"os"
	"syscall"
	"unsafe"
)

const ioctlReadTermios = 0x40487413 // TIOCGETA

func isTerminal(f *os.File) bool {
	var term syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(ioctlReadTermios),
		uintptr(unsafe.Pointer(&term)),
	)
	return errno == 0
}
