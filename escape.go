package readline

// parserState is the state of the escape-sequence state machine.
type parserState int

const (
	stateNormal parserState = iota
	stateESC                // received ESC (0x1B)
	stateCSI                // received ESC [
	stateESCO               // received ESC O  (application-mode arrows/home/end)
)

// EscapeParser is a byte-level state machine that converts raw terminal bytes
// into Key values.  Feed one byte at a time; when complete==true the returned
// key (and optional rune) represent the fully parsed event.
type EscapeParser struct {
	state  parserState
	params []byte // accumulates CSI parameter bytes
}

// Feed processes a single raw byte and returns the resulting event.
// If complete is false the machine is still accumulating an escape sequence and
// the caller should feed more bytes.
func (p *EscapeParser) Feed(b byte) (key Key, r rune, complete bool) {
	switch p.state {

	// ── Normal state ─────────────────────────────────────────────────────────
	case stateNormal:
		if b == 0x1B {
			p.state = stateESC
			p.params = p.params[:0]
			return 0, 0, false
		}
		return p.fromByte(b)

	// ── After ESC ────────────────────────────────────────────────────────────
	case stateESC:
		switch b {
		case '[':
			p.state = stateCSI
			p.params = p.params[:0]
			return 0, 0, false
		case 'O':
			p.state = stateESCO
			return 0, 0, false
		default:
			p.state = stateNormal
			return KeyEsc, 0, true
		}

	// ── CSI: ESC [ <params> <final> ──────────────────────────────────────────
	case stateCSI:
		// Parameter bytes: 0x30–0x3F  ('0'–'?')
		// Intermediate bytes: 0x20–0x2F (space, !, …)
		// Final byte: 0x40–0x7E
		if b >= 0x20 && b <= 0x3F {
			p.params = append(p.params, b)
			return 0, 0, false
		}
		p.state = stateNormal
		return p.dispatchCSI(b)

	// ── ESC O (application keypad / cursor keys) ──────────────────────────────
	case stateESCO:
		p.state = stateNormal
		switch b {
		case 'A':
			return KeyArrowUp, 0, true
		case 'B':
			return KeyArrowDown, 0, true
		case 'C':
			return KeyArrowRight, 0, true
		case 'D':
			return KeyArrowLeft, 0, true
		case 'H':
			return KeyHome, 0, true
		case 'F':
			return KeyEnd, 0, true
		}
		return KeyEsc, 0, true
	}
	return 0, 0, false
}

// Reset returns the parser to the normal state, discarding any partial sequence.
func (p *EscapeParser) Reset() {
	p.state = stateNormal
	p.params = p.params[:0]
}

// InSequence reports whether the parser is mid-escape-sequence.
func (p *EscapeParser) InSequence() bool {
	return p.state != stateNormal
}

// fromByte maps a plain (non-ESC) byte to a Key.
func (p *EscapeParser) fromByte(b byte) (Key, rune, bool) {
	switch {
	case b == 0x7F:
		return KeyBackspace, 0, true
	case b == 0x0D || b == 0x0A:
		return KeyEnter, 0, true
	case b < 0x20:
		return Key(b), 0, true
	default:
		// ASCII printable – caller handles UTF-8 lead bytes before calling Feed.
		return KeyRune, rune(b), true
	}
}

// dispatchCSI interprets a complete CSI sequence.
func (p *EscapeParser) dispatchCSI(final byte) (Key, rune, bool) {
	param := string(p.params)
	switch final {
	case 'A':
		return KeyArrowUp, 0, true
	case 'B':
		return KeyArrowDown, 0, true
	case 'C':
		return KeyArrowRight, 0, true
	case 'D':
		return KeyArrowLeft, 0, true
	case 'H':
		return KeyHome, 0, true
	case 'F':
		return KeyEnd, 0, true
	case '~':
		switch param {
		case "1", "7":
			return KeyHome, 0, true
		case "3":
			return KeyDelete, 0, true
		case "4", "8":
			return KeyEnd, 0, true
		case "5":
			return KeyPageUp, 0, true
		case "6":
			return KeyPageDown, 0, true
		}
	}
	return KeyEsc, 0, true
}
