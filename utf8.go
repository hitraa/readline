package readline

import "unicode/utf8"

// decodeUTF8Rune completes a multi-byte UTF-8 sequence whose first byte has
// already been read.  readByte is called to supply each continuation byte.
// Returns (rune, byteCount, error).
func decodeUTF8Rune(first byte, readByte func() (byte, error)) (rune, int, error) {
	// Single-byte (ASCII): caller should not call us, but handle gracefully.
	if first < 0x80 {
		return rune(first), 1, nil
	}

	// Determine sequence length from the lead byte.
	var size int
	switch {
	case first&0xE0 == 0xC0:
		size = 2
	case first&0xF0 == 0xE0:
		size = 3
	case first&0xF8 == 0xF0:
		size = 4
	default:
		return utf8.RuneError, 1, nil
	}

	buf := make([]byte, size)
	buf[0] = first
	for i := 1; i < size; i++ {
		b, err := readByte()
		if err != nil {
			return utf8.RuneError, i, err
		}
		if b&0xC0 != 0x80 { // not a valid continuation byte
			return utf8.RuneError, i, nil
		}
		buf[i] = b
	}

	r, n := utf8.DecodeRune(buf)
	return r, n, nil
}
