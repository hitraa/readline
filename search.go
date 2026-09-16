package readline

import (
	"fmt"
	"strings"
)

// SearchMode manages reverse/forward incremental history search (Ctrl+R / Ctrl+S).
type SearchMode struct {
	active      bool
	query       []rune
	match       string
	matchIdx    int
	originalBuf string
	originalPos int
	history     *History
}

func newSearchMode(h *History) *SearchMode {
	return &SearchMode{history: h}
}

// Start initiates search mode, saving original buffer state.
func (sm *SearchMode) Start(buf *LineBuffer) {
	sm.active = true
	sm.query = sm.query[:0]
	sm.originalBuf = buf.String()
	sm.originalPos = buf.Pos()
	sm.match = ""
	sm.matchIdx = sm.history.Len()
	sm.findPrev()
}

// Active returns true if search mode is currently running.
func (sm *SearchMode) Active() bool {
	return sm.active
}

// Cancel terminates search mode and restores the original buffer state.
func (sm *SearchMode) Cancel(buf *LineBuffer) {
	sm.active = false
	buf.Set(sm.originalBuf)
	buf.SetPos(sm.originalPos)
}

// Accept terminates search mode, keeping the currently matched line.
func (sm *SearchMode) Accept(buf *LineBuffer) {
	sm.active = false
	if sm.match != "" {
		buf.Set(sm.match)
	}
}

// Prompt returns the search status prompt to display:
// (reverse-i-search)`query': match
func (sm *SearchMode) Prompt() string {
	q := string(sm.query)
	return fmt.Sprintf("(reverse-i-search)`%s': %s", q, sm.match)
}

// Match returns the current matched string.
func (sm *SearchMode) Match() string {
	return sm.match
}

func (sm *SearchMode) findPrev() {
	q := string(sm.query)
	entries := sm.history.Entries()
	start := sm.matchIdx - 1
	if start >= len(entries) {
		start = len(entries) - 1
	}
	for i := start; i >= 0; i-- {
		if strings.Contains(entries[i], q) {
			sm.match = entries[i]
			sm.matchIdx = i
			return
		}
	}
}

func (sm *SearchMode) findNext() {
	q := string(sm.query)
	entries := sm.history.Entries()
	for i := sm.matchIdx + 1; i < len(entries); i++ {
		if strings.Contains(entries[i], q) {
			sm.match = entries[i]
			sm.matchIdx = i
			return
		}
	}
}

// HandleKey processes an input event while in reverse search mode.
// Returns (handled, done, line, err).
func (sm *SearchMode) HandleKey(evt InputEvent, buf *LineBuffer) (handled bool, done bool, line string, err error) {
	switch evt.Key {
	case KeyCtrlR:
		sm.findPrev()
		return true, false, "", nil

	case KeyCtrlS:
		sm.findNext()
		return true, false, "", nil

	case KeyBackspace, KeyCtrlH:
		if len(sm.query) > 0 {
			sm.query = sm.query[:len(sm.query)-1]
			sm.matchIdx = sm.history.Len()
			sm.match = ""
			sm.findPrev()
		}
		return true, false, "", nil

	case KeyEnter:
		sm.Accept(buf)
		return true, true, buf.String(), nil

	case KeyEsc, KeyCtrlG:
		sm.Cancel(buf)
		return true, false, "", nil

	case KeyCtrlC:
		sm.Cancel(buf)
		return true, true, "", ErrInterrupt

	case KeyRune:
		sm.query = append(sm.query, evt.Rune)
		sm.matchIdx = sm.history.Len()
		sm.findPrev()
		return true, false, "", nil

	default:
		// For other keys (arrows, submit, etc.): accept the match and exit search mode,
		// allowing the caller to process the key normally on the matched buffer
		sm.Accept(buf)
		return false, false, "", nil
	}
}
