//go:build windows

package readline

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
)

func setConsoleMode(h syscall.Handle, mode uint32) error {
	r1, _, err := procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	if r1 == 0 {
		return err
	}
	return nil
}

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

const (
	enableProcessedInput       = 0x0001
	enableLineInput            = 0x0002
	enableEchoInput            = 0x0004
	enableWindowInput          = 0x0008
	enableVirtualTerminalInput = 0x0200

	enableProcessedOutput           = 0x0001
	enableWrapAtEolOutput           = 0x0002
	enableVirtualTerminalProcessing = 0x0004
)

// terminal implements Windows Console mode handling using Win32 Console API via standard library syscall.
type terminal struct {
	inHandle  syscall.Handle
	outHandle syscall.Handle
	origIn    uint32
	origOut   uint32
	inRaw     bool
	isTerm    bool
}

func newTerminal(f *os.File, _ bool) (*terminal, error) {
	inH := syscall.Handle(f.Fd())
	var inMode uint32
	if err := syscall.GetConsoleMode(inH, &inMode); err != nil {
		// Stdin is not a console (e.g. pipe or redirected file)
		return &terminal{inHandle: inH, isTerm: false}, nil
	}

	outH := syscall.Handle(os.Stdout.Fd())
	var outMode uint32
	_ = syscall.GetConsoleMode(outH, &outMode)

	t := &terminal{
		inHandle:  inH,
		outHandle: outH,
		origIn:    inMode,
		origOut:   outMode,
		isTerm:    true,
	}

	if err := t.enterRaw(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *terminal) IsTerminal() bool {
	return t.isTerm
}

func (t *terminal) enterRaw() error {
	if !t.isTerm || t.inRaw {
		return nil
	}
	rawIn := t.origIn &^ (enableLineInput | enableEchoInput | enableProcessedInput | enableWindowInput)
	rawIn |= enableVirtualTerminalInput
	_ = setConsoleMode(t.inHandle, rawIn)

	rawOut := t.origOut | enableVirtualTerminalProcessing | enableProcessedOutput
	_ = setConsoleMode(t.outHandle, rawOut)

	t.inRaw = true
	return nil
}

func (t *terminal) leaveRaw() error {
	if !t.isTerm || !t.inRaw {
		return nil
	}
	_ = setConsoleMode(t.inHandle, t.origIn)
	_ = setConsoleMode(t.outHandle, t.origOut)
	t.inRaw = false
	return nil
}

func (t *terminal) Close() error {
	return t.leaveRaw()
}

func (t *terminal) GetSize() (int, int, error) {
	if !t.isTerm {
		return 80, 24, nil
	}
	var csbi consoleScreenBufferInfo
	r1, _, err := procGetConsoleScreenBufferInfo.Call(
		uintptr(t.outHandle),
		uintptr(unsafe.Pointer(&csbi)),
	)
	if r1 == 0 {
		return 80, 24, err
	}
	cols := int(csbi.Window.Right - csbi.Window.Left + 1)
	rows := int(csbi.Window.Bottom - csbi.Window.Top + 1)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return cols, rows, nil
}

func (t *terminal) WatchResize(fn func(int, int)) func() {
	return func() {}
}
