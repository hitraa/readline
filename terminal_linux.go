//go:build linux

package readline

import (
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// terminal holds per-platform terminal state.
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
	// Input flags: disable break, CR-to-NL, parity, strip, software flow control
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	// Output flags: disable post-processing
	raw.Oflag &^= syscall.OPOST
	// Control flags: 8-bit characters
	raw.Cflag |= syscall.CS8
	// Local flags: disable echo, canonical, and extended processing
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN
	if !enableSignals {
		raw.Lflag &^= syscall.ISIG
	}
	// Read returns after 1 byte, no timeout
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := tcSet(t.fd, &raw); err != nil {
		return nil, err
	}
	t.inRaw = true
	return t, nil
}

// enterRaw re-applies raw mode (no-op if already raw).
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

// leaveRaw restores the terminal to its state before entering raw mode.
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

// WatchResize starts a goroutine that calls fn whenever the terminal is resized.
// Call the returned stop function to stop watching.
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

// GetSize returns the current terminal dimensions.
func (t *terminal) GetSize() (cols, rows int, err error) {
	type winsize struct {
		Row, Col       uint16
		Xpixel, Ypixel uint16
	}
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}

// Close restores the original terminal settings.
func (t *terminal) Close() error {
	return t.leaveRaw()
}

// ── low-level termios wrappers ────────────────────────────────────────────────

func tcGet(fd int) (syscall.Termios, error) {
	var t syscall.Termios
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&t)),
		0, 0, 0,
	)
	if errno != 0 {
		return t, errno
	}
	return t, nil
}

func tcSet(fd int, t *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(t)),
		0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
