//go:build !linux && !darwin && !windows

package readline

import (
	"os"
)

// terminal is a stub for unsupported platforms.
type terminal struct {
	fd     int
	isTerm bool
}

func newTerminal(f *os.File, _ bool) (*terminal, error) {
	return &terminal{fd: int(f.Fd()), isTerm: false}, nil
}

func (t *terminal) IsTerminal() bool                  { return false }
func (t *terminal) enterRaw() error                   { return nil }
func (t *terminal) leaveRaw() error                   { return nil }
func (t *terminal) Close() error                      { return nil }
func (t *terminal) GetSize() (int, int, error)        { return 80, 24, nil }
func (t *terminal) WatchResize(func(int, int)) func() { return func() {} }
