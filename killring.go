package readline

// KillRing manages a circular buffer of killed text for Emacs-style kill/yank operations.
type KillRing struct {
	entries []string
	pos     int
	max     int
}

// NewKillRing creates a KillRing with given maximum capacity.
func NewKillRing(max int) *KillRing {
	if max <= 0 {
		max = 60
	}
	return &KillRing{max: max}
}

// Push adds a non-empty string to the kill ring.
func (kr *KillRing) Push(s string) {
	if s == "" {
		return
	}
	kr.entries = append(kr.entries, s)
	if len(kr.entries) > kr.max {
		kr.entries = kr.entries[len(kr.entries)-kr.max:]
	}
	kr.pos = len(kr.entries) - 1
}

// Top returns the most recently killed text, or "" if empty.
func (kr *KillRing) Top() string {
	if len(kr.entries) == 0 {
		return ""
	}
	return kr.entries[len(kr.entries)-1]
}

// YankPop moves back to the previous entry in the kill ring and returns it.
func (kr *KillRing) YankPop() string {
	if len(kr.entries) == 0 {
		return ""
	}
	kr.pos--
	if kr.pos < 0 {
		kr.pos = len(kr.entries) - 1
	}
	return kr.entries[kr.pos]
}

// Len returns the count of items in the kill ring.
func (kr *KillRing) Len() int {
	return len(kr.entries)
}
