//go:build windows

package readline

import (
	"os"
)

// terminal is a stub for Windows.  Raw mode via the Win32 Console API is not
// yet implemented.  The Editor will return ErrNotSupported from New().
type terminal struct {
	fd int
}

func newTerminal(f *os.File, _ bool) (*terminal, error) {
	return nil, ErrNotSupported
}

func (t *terminal) enterRaw() error                      { return ErrNotSupported }
func (t *terminal) leaveRaw() error                      { return nil }
func (t *terminal) Close() error                         { return nil }
func (t *terminal) GetSize() (int, int, error)           { return 80, 24, nil }
func (t *terminal) WatchResize(func(int, int)) func()    { return func() {} }
