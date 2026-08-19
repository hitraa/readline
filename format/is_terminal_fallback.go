//go:build !linux && !darwin && !windows

package format

import "os"

func isTerminal(f *os.File) bool {
	return os.Getenv("TERM") != "dumb"
}
