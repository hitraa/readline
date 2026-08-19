package readline

import "io"

// InputEvent is a fully parsed terminal input event.
type InputEvent struct {
	Key  Key
	Rune rune // valid when Key == KeyRune
}

// inputReader reads one byte at a time from an io.Reader, running the bytes
// through the EscapeParser (and UTF-8 decoder for multi-byte runes) to produce
// InputEvents.
type inputReader struct {
	r      io.Reader
	parser EscapeParser
	buf    [1]byte
}

func newInputReader(r io.Reader) *inputReader {
	return &inputReader{r: r}
}

// readByte reads exactly one byte from the underlying reader.
func (ir *inputReader) readByte() (byte, error) {
	_, err := ir.r.Read(ir.buf[:])
	return ir.buf[0], err
}

// ReadEvent blocks until a complete key event is available and returns it.
// io.EOF is propagated as-is so the caller can distinguish Ctrl+D (EOF) from
// other errors.
func (ir *inputReader) ReadEvent() (InputEvent, error) {
	for {
		b, err := ir.readByte()
		if err != nil {
			return InputEvent{}, err
		}

		// Multi-byte UTF-8 lead byte while in normal parser state:
		// decode the full rune before touching the escape parser.
		if ir.parser.state == stateNormal && b >= 0x80 {
			r, _, decErr := decodeUTF8Rune(b, ir.readByte)
			if decErr != nil {
				return InputEvent{}, decErr
			}
			return InputEvent{Key: KeyRune, Rune: r}, nil
		}

		key, r, complete := ir.parser.Feed(b)
		if complete {
			return InputEvent{Key: key, Rune: r}, nil
		}
	}
}
