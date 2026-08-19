//go:build darwin

package readline

import (
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// Darwin termios ioctl numbers differ from Linux.
const (
	ioctlReadTermios  = 0x40487413 // TIOCGETA
	ioctlWriteTermios = 0x80487414 // TIOCSETA
	ioctlGetWinSize   = 0x40087468 // TIOCGWINSZ
)

type terminal struct {
	fd       int
	orig     syscall.Termios
	inRaw    bool
	sigWinCh chan os.Signal
	onResize func(cols, rows int)
}

func newTerminal(f *os.File, enableSignals bool) (*terminal, error) {
	t := &terminal{fd: int(f.Fd())}
	orig, err := tcGet(t.fd)
	if err != nil {
		return nil, err
	}
	t.orig = orig

	raw := orig
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN
	if !enableSignals {
		raw.Lflag &^= syscall.ISIG
	}
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := tcSet(t.fd, &raw); err != nil {
		return nil, err
	}
	t.inRaw = true
	return t, nil
}

func (t *terminal) enterRaw() error {
	if t.inRaw {
		return nil
	}
	orig, err := tcGet(t.fd)
	if err != nil {
		return err
	}
	t.orig = orig

	raw := orig
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := tcSet(t.fd, &raw); err != nil {
		return err
	}
	t.inRaw = true
	return nil
}

func (t *terminal) leaveRaw() error {
	if !t.inRaw {
		return nil
	}
	if err := tcSet(t.fd, &t.orig); err != nil {
		return err
	}
	t.inRaw = false
	return nil
}

func (t *terminal) WatchResize(fn func(cols, rows int)) func() {
	t.sigWinCh = make(chan os.Signal, 1)
	t.onResize = fn
	signal.Notify(t.sigWinCh, syscall.SIGWINCH)
	go func() {
		for range t.sigWinCh {
			cols, rows, err := t.GetSize()
			if err == nil && fn != nil {
				fn(cols, rows)
			}
		}
	}()
	return func() {
		signal.Stop(t.sigWinCh)
		close(t.sigWinCh)
	}
}

func (t *terminal) GetSize() (cols, rows int, err error) {
	type winsize struct {
		Row, Col        uint16
		Xpixel, Ypixel uint16
	}
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		uintptr(ioctlGetWinSize),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}

func (t *terminal) Close() error { return t.leaveRaw() }

func tcGet(fd int) (syscall.Termios, error) {
	var term syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctlReadTermios),
		uintptr(unsafe.Pointer(&term)),
	)
	if errno != 0 {
		return term, errno
	}
	return term, nil
}

func tcSet(fd int, term *syscall.Termios) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctlWriteTermios),
		uintptr(unsafe.Pointer(term)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
