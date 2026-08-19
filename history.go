package readline

import (
	"bufio"
	"os"
	"strings"
)

// History manages in-session and persistent command history.
type History struct {
	entries []string
	idx     int    // points past entries when at "live" position
	pending string // live (unsubmitted) line saved while browsing
	maxSize int
}

// NewHistory creates a History capped at maxSize entries.
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &History{maxSize: maxSize}
}

// Push records a non-empty entry, deduplicating consecutive identical lines.
// Resets the browse index to the end.
func (h *History) Push(s string) {
	if s == "" {
		return
	}
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == s {
		h.resetIdx()
		return
	}
	h.entries = append(h.entries, s)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
	}
	h.resetIdx()
}

// SetPending saves the current (unsaved) line before the user starts browsing.
func (h *History) SetPending(s string) { h.pending = s }

// resetIdx moves the browse cursor past the last entry and clears any saved
// pending text so a fresh ReadLine session never inherits a stale prefix.
func (h *History) resetIdx() {
	h.idx = len(h.entries)
	h.pending = ""
}

// Reset is the public form of resetIdx.
func (h *History) Reset() { h.resetIdx() }

// ResetSearch abandons any in-progress prefix search: it clears pending and
// resets the browse index to the end so the next PageUp starts fresh.
func (h *History) ResetSearch() {
	h.pending = ""
	h.idx = len(h.entries)
}

// Up moves one step earlier in history.
// Returns ("", false) when already at the oldest entry.
func (h *History) Up() (string, bool) {
	if h.idx == 0 {
		return "", false
	}
	h.idx--
	return h.entries[h.idx], true
}

// Down moves one step later in history.
// Returns (pending, false) when the browse cursor is already past the last entry.
func (h *History) Down() (string, bool) {
	if h.idx >= len(h.entries) {
		return h.pending, false
	}
	h.idx++
	if h.idx == len(h.entries) {
		return h.pending, true
	}
	return h.entries[h.idx], true
}

// SearchUp walks backwards from the current index looking for an entry that
// starts with prefix.  If found, the index is moved to that entry and the
// entry is returned.  Returns ("", false) when no earlier match exists.
func (h *History) SearchUp(prefix string) (string, bool) {
	for i := h.idx - 1; i >= 0; i-- {
		if strings.HasPrefix(h.entries[i], prefix) {
			h.idx = i
			return h.entries[i], true
		}
	}
	return "", false
}

// SearchDown walks forwards from the current index looking for an entry that
// starts with prefix.  If found, the index is moved to that entry and the
// entry is returned.  When no later match exists the index is reset to the end
// and (pending, false) is returned so the caller can restore the typed text.
func (h *History) SearchDown(prefix string) (string, bool) {
	for i := h.idx + 1; i < len(h.entries); i++ {
		if strings.HasPrefix(h.entries[i], prefix) {
			h.idx = i
			return h.entries[i], true
		}
	}
	h.idx = len(h.entries)
	return h.pending, false
}

// Entries returns a copy of all history entries in order.
func (h *History) Entries() []string {
	cp := make([]string, len(h.entries))
	copy(cp, h.entries)
	return cp
}

// Len returns the number of stored entries.
func (h *History) Len() int { return len(h.entries) }

// ── File persistence ──────────────────────────────────────────────────────────

// LoadFile reads plain newline-delimited history from path.
// Missing files are silently ignored.
func (h *History) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line != "" {
			h.entries = append(h.entries, line)
		}
	}
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
	}
	h.resetIdx()
	return sc.Err()
}

// SaveFile appends entries to path, creating the file with mode 0600 if needed.
func (h *History) SaveFile(path string, entries []string) error {
	if len(entries) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range entries {
		if _, err := w.WriteString(e + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}
