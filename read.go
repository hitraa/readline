package readline

import (
	"bytes"
	"io"
)

// InputEvent is a fully parsed terminal input event.
type InputEvent struct {
	Key   Key
	Rune  rune   // valid when Key == KeyRune
	Paste string // valid when Key == KeyPasteStart
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
	// First drain any pending events buffered in the escape parser
	if ir.parser.HasPending() {
		if k, r, ok := ir.parser.PopPending(); ok {
			return InputEvent{Key: k, Rune: r}, nil
		}
	}

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
			if key == KeyPasteStart {
				// Read until KeyPasteEnd is reached
				pasteStr, pErr := ir.readPaste()
				if pErr != nil {
					return InputEvent{}, pErr
				}
				return InputEvent{Key: KeyPasteStart, Paste: pasteStr}, nil
			}
			return InputEvent{Key: key, Rune: r}, nil
		}
	}
}

// readPaste collects all characters inside bracketed paste mode until KeyPasteEnd is reached.
func (ir *inputReader) readPaste() (string, error) {
	var pasteBuf bytes.Buffer
	for {
		b, err := ir.readByte()
		if err != nil {
			return pasteBuf.String(), err
		}
		key, r, complete := ir.parser.Feed(b)
		if complete {
			if key == KeyPasteEnd {
				break
			}
			if key == KeyRune {
				pasteBuf.WriteRune(r)
			} else if key == KeyEnter {
				pasteBuf.WriteByte('\n')
			} else if key < 0x20 || key == KeyBackspace {
				// Pass control characters like Tab or standard chars
				if key == KeyTab {
					pasteBuf.WriteByte('\t')
				}
			}
		}
	}
	return pasteBuf.String(), nil
}
