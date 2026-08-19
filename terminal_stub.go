//go:build !linux && !darwin && !windows

package readline

import (
	"os"
)

// terminal is a stub for unsupported platforms.
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
