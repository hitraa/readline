package readline

// Key represents a parsed key event.
// Positive values ≤ 0x7F are ASCII control codes or the KeyBackspace byte.
// Values ≥ 0x1000 are synthetic keys produced by escape sequences.
// KeyRune (-1) means a printable Unicode rune was read; inspect InputEvent.Rune.
type Key int32

const (
	// ── ASCII control keys ────────────────────────────────────────────────────
	KeyCtrlA          Key = 0x01
	KeyCtrlB          Key = 0x02
	KeyCtrlC          Key = 0x03
	KeyCtrlD          Key = 0x04
	KeyCtrlE          Key = 0x05
	KeyCtrlF          Key = 0x06
	KeyCtrlG          Key = 0x07 // Cancel / Abort
	KeyCtrlH          Key = 0x08 // Backspace on some terminals
	KeyTab            Key = 0x09
	KeyCtrlK          Key = 0x0B
	KeyCtrlL          Key = 0x0C
	KeyEnter          Key = 0x0D
	KeyCtrlN          Key = 0x0E
	KeyCtrlO          Key = 0x0F
	KeyCtrlP          Key = 0x10
	KeyCtrlQ          Key = 0x11
	KeyCtrlR          Key = 0x12 // Reverse incremental search
	KeyCtrlS          Key = 0x13 // Forward incremental search
	KeyCtrlT          Key = 0x14 // Transpose characters
	KeyCtrlU          Key = 0x15
	KeyCtrlV          Key = 0x16
	KeyCtrlW          Key = 0x17
	KeyCtrlX          Key = 0x18
	KeyCtrlY          Key = 0x19 // Yank from kill ring
	KeyCtrlZ          Key = 0x1A
	KeyEsc            Key = 0x1B
	KeyCtrlUnderscore Key = 0x1F // Undo (Ctrl+_)
	KeyBackspace      Key = 0x7F // DEL byte, standard backspace

	// ── Synthetic keys (escape sequences) ────────────────────────────────────
	KeyArrowUp Key = 0x1000 + iota
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyDelete
	KeyPageUp
	KeyPageDown
	KeyInsert
	KeyAltB         // Alt+B (word left)
	KeyAltF         // Alt+F (word right)
	KeyAltD         // Alt+D (delete word forward)
	KeyAltBackspace // Alt+Backspace (delete word backward)
	KeyAltY         // Alt+Y (yank-pop)
	KeyCtrlLeft     // Ctrl+Left (word left)
	KeyCtrlRight    // Ctrl+Right (word right)
	KeyBackTab      // Shift+Tab
	KeyPasteStart   // Bracketed paste begin
	KeyPasteEnd     // Bracketed paste end

	// ── Special sentinel ──────────────────────────────────────────────────────
	// KeyRune signals that InputEvent.Rune holds a printable Unicode character.
	KeyRune Key = -1
)
