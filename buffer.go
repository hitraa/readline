package readline

import "unicode"

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

// InsertString inserts string s at the cursor position and advances the cursor.
func (b *LineBuffer) InsertString(s string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return
	}
	tail := make([]rune, len(b.data[b.pos:]))
	copy(tail, b.data[b.pos:])
	b.data = append(b.data[:b.pos], runes...)
	b.data = append(b.data, tail...)
	b.pos += len(runes)
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
	_, ok := b.DeleteWordBeforeText()
	return ok
}

// DeleteWordBeforeText removes the word before the cursor and returns the deleted text.
func (b *LineBuffer) DeleteWordBeforeText() (string, bool) {
	if b.pos == 0 {
		return "", false
	}
	end := b.pos
	for b.pos > 0 && b.data[b.pos-1] == ' ' {
		b.pos--
	}
	for b.pos > 0 && b.data[b.pos-1] != ' ' {
		b.pos--
	}
	deleted := string(b.data[b.pos:end])
	b.data = append(b.data[:b.pos], b.data[end:]...)
	return deleted, true
}

// DeleteWordAfter (Alt+D) removes the word immediately after the cursor.
func (b *LineBuffer) DeleteWordAfter() bool {
	_, ok := b.DeleteWordAfterText()
	return ok
}

// DeleteWordAfterText removes the word after the cursor and returns the deleted text.
func (b *LineBuffer) DeleteWordAfterText() (string, bool) {
	if b.pos >= len(b.data) {
		return "", false
	}
	start := b.pos
	idx := b.pos
	for idx < len(b.data) && b.data[idx] == ' ' {
		idx++
	}
	for idx < len(b.data) && b.data[idx] != ' ' {
		idx++
	}
	deleted := string(b.data[start:idx])
	b.data = append(b.data[:start], b.data[idx:]...)
	return deleted, true
}

// KillToEnd (Ctrl+K) removes everything from the cursor to the end.
func (b *LineBuffer) KillToEnd() bool {
	_, ok := b.KillToEndText()
	return ok
}

// KillToEndText removes everything from cursor to end and returns the removed text.
func (b *LineBuffer) KillToEndText() (string, bool) {
	if b.pos >= len(b.data) {
		return "", false
	}
	deleted := string(b.data[b.pos:])
	b.data = b.data[:b.pos]
	return deleted, true
}

// KillToStart (Ctrl+U) removes everything from the start to the cursor.
func (b *LineBuffer) KillToStart() bool {
	_, ok := b.KillToStartText()
	return ok
}

// KillToStartText removes everything from start to cursor and returns the removed text.
func (b *LineBuffer) KillToStartText() (string, bool) {
	if b.pos == 0 {
		return "", false
	}
	deleted := string(b.data[:b.pos])
	b.data = b.data[b.pos:]
	b.pos = 0
	return deleted, true
}

// Transpose (Ctrl+T) swaps the rune before cursor with the rune at cursor,
// or if cursor is at the end, swaps the last two runes.
func (b *LineBuffer) Transpose() bool {
	if len(b.data) < 2 {
		return false
	}
	if b.pos == len(b.data) {
		b.data[b.pos-2], b.data[b.pos-1] = b.data[b.pos-1], b.data[b.pos-2]
		return true
	}
	if b.pos > 0 {
		b.data[b.pos-1], b.data[b.pos] = b.data[b.pos], b.data[b.pos-1]
		b.pos++
		return true
	}
	return false
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

// MoveWordLeft (Alt+B / Ctrl+Left) moves cursor back to the start of current or previous word.
func (b *LineBuffer) MoveWordLeft() bool {
	if b.pos == 0 {
		return false
	}
	for b.pos > 0 && b.data[b.pos-1] == ' ' {
		b.pos--
	}
	for b.pos > 0 && b.data[b.pos-1] != ' ' {
		b.pos--
	}
	return true
}

// MoveWordRight (Alt+F / Ctrl+Right) moves cursor forward past the end of the next word.
func (b *LineBuffer) MoveWordRight() bool {
	if b.pos >= len(b.data) {
		return false
	}
	for b.pos < len(b.data) && b.data[b.pos] == ' ' {
		b.pos++
	}
	for b.pos < len(b.data) && b.data[b.pos] != ' ' {
		b.pos++
	}
	return true
}

// MoveHome moves the cursor to position 0.
func (b *LineBuffer) MoveHome() { b.pos = 0 }

// MoveEnd moves the cursor past the last rune.
func (b *LineBuffer) MoveEnd() { b.pos = len(b.data) }

// SetPos sets the cursor position clamped to [0, len(data)].
func (b *LineBuffer) SetPos(pos int) {
	if pos < 0 {
		b.pos = 0
	} else if pos > len(b.data) {
		b.pos = len(b.data)
	} else {
		b.pos = pos
	}
}

// ── Accessors ─────────────────────────────────────────────────────────────────

// String returns the current buffer content.
func (b *LineBuffer) String() string { return string(b.data) }

// Runes returns a copy of the current buffer runes.
func (b *LineBuffer) Runes() []rune {
	cp := make([]rune, len(b.data))
	copy(cp, b.data)
	return cp
}

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

// runeDisplayWidth returns the visual column width of rune r in terminal cells:
// - 0 for zero-width characters (combining marks, zero-width space/joiner, control codes)
// - 2 for East Asian wide characters and emojis
// - 1 for standard single-width characters
func runeDisplayWidth(r rune) int {
	if r == 0 {
		return 0
	}
	// Control codes have 0 display width in line calculations
	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return 0
	}
	// Zero-width spaces, joiners, formatting characters
	if r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF {
		return 0
	}
	// Variation selectors (VS1-VS16, VS17-VS256)
	if (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF) {
		return 0
	}
	// Combining diacritical marks and enclosing marks
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || (unicode.Is(unicode.Cf, r) && r != 0x00AD) {
		return 0
	}
	// East Asian wide characters and emojis
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
			(r >= 0x1F000 && r <= 0x1FAFF) ||
			(r >= 0x20000 && r <= 0x3FFFD)) {
		return 2
	}
	return 1
}
