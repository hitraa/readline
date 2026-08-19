package readline

// LineBuffer is a rune-indexed, editable line with an in-line cursor position.
// All operations are O(n) in the worst case — acceptable for interactive input.
type LineBuffer struct {
	data []rune
	pos  int // cursor: 0 ≤ pos ≤ len(data)
}

// NewLineBuffer returns an empty buffer.
func NewLineBuffer() *LineBuffer { return &LineBuffer{} }

// ── Mutation ─────────────────────────────────────────────────────────────────

// Insert inserts rune r at the cursor and advances the cursor by one.
func (b *LineBuffer) Insert(r rune) {
	tail := make([]rune, len(b.data[b.pos:]))
	copy(tail, b.data[b.pos:])
	b.data = append(b.data[:b.pos], r)
	b.data = append(b.data, tail...)
	b.pos++
}

// Backspace removes the rune immediately before the cursor.
// Returns false if the cursor is already at the start.
func (b *LineBuffer) Backspace() bool {
	if b.pos == 0 {
		return false
	}
	b.data = append(b.data[:b.pos-1], b.data[b.pos:]...)
	b.pos--
	return true
}

// Delete removes the rune at the cursor (the character after it).
// Returns false if the cursor is at the end.
func (b *LineBuffer) Delete() bool {
	if b.pos >= len(b.data) {
		return false
	}
	b.data = append(b.data[:b.pos], b.data[b.pos+1:]...)
	return true
}

// DeleteWordBefore (Ctrl+W) removes the word immediately before the cursor.
// A "word" is a maximal run of non-space characters preceded by optional spaces.
func (b *LineBuffer) DeleteWordBefore() bool {
	if b.pos == 0 {
		return false
	}
	end := b.pos
	for b.pos > 0 && b.data[b.pos-1] == ' ' {
		b.pos--
	}
	for b.pos > 0 && b.data[b.pos-1] != ' ' {
		b.pos--
	}
	b.data = append(b.data[:b.pos], b.data[end:]...)
	return true
}

// KillToEnd (Ctrl+K) removes everything from the cursor to the end.
func (b *LineBuffer) KillToEnd() bool {
	if b.pos >= len(b.data) {
		return false
	}
	b.data = b.data[:b.pos]
	return true
}

// KillToStart (Ctrl+U) removes everything from the start to the cursor.
func (b *LineBuffer) KillToStart() bool {
	if b.pos == 0 {
		return false
	}
	b.data = b.data[b.pos:]
	b.pos = 0
	return true
}

// Set replaces the buffer content with s and moves the cursor to the end.
func (b *LineBuffer) Set(s string) {
	b.data = []rune(s)
	b.pos = len(b.data)
}

// Clear resets the buffer to empty.
func (b *LineBuffer) Clear() {
	b.data = b.data[:0]
	b.pos = 0
}

// ── Movement ──────────────────────────────────────────────────────────────────

// MoveLeft moves the cursor one rune to the left. Returns false at start.
func (b *LineBuffer) MoveLeft() bool {
	if b.pos == 0 {
		return false
	}
	b.pos--
	return true
}

// MoveRight moves the cursor one rune to the right. Returns false at end.
func (b *LineBuffer) MoveRight() bool {
	if b.pos >= len(b.data) {
		return false
	}
	b.pos++
	return true
}

// MoveHome moves the cursor to position 0.
func (b *LineBuffer) MoveHome() { b.pos = 0 }

// MoveEnd moves the cursor past the last rune.
func (b *LineBuffer) MoveEnd() { b.pos = len(b.data) }

// ── Accessors ─────────────────────────────────────────────────────────────────

// String returns the current buffer content.
func (b *LineBuffer) String() string { return string(b.data) }

// Len returns the number of runes in the buffer.
func (b *LineBuffer) Len() int { return len(b.data) }

// Pos returns the current cursor position (rune index).
func (b *LineBuffer) Pos() int { return b.pos }

// ── Display width helpers ──────────────────────────────────────────────────────

// DisplayWidth returns the terminal column width of the first n runes.
func (b *LineBuffer) DisplayWidth(n int) int {
	w := 0
	for i := 0; i < n && i < len(b.data); i++ {
		w += runeDisplayWidth(b.data[i])
	}
	return w
}

// DisplayWidthTotal returns the terminal column width of the entire buffer.
func (b *LineBuffer) DisplayWidthTotal() int {
	return b.DisplayWidth(len(b.data))
}

// runeDisplayWidth returns 2 for East Asian wide characters, 1 otherwise.
func runeDisplayWidth(r rune) int {
	if r >= 0x1100 &&
		(r <= 0x115F ||
			r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) ||
			(r >= 0x3040 && r <= 0xA4CF) ||
			(r >= 0xA960 && r <= 0xA97F) ||
			(r >= 0xAC00 && r <= 0xD7FF) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1F300 && r <= 0x1F9FF) ||
			(r >= 0x20000 && r <= 0x3FFFD)) {
		return 2
	}
	return 1
}
