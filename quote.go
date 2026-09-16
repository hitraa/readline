package readline

import (
	"strings"
	"unicode"
)

// Token represents a parsed lexical token from an input line.
type Token struct {
	Value      string // unescaped / unquoted value
	Raw        string // raw slice of the input string
	Start      int    // rune start index in line
	End        int    // rune end index in line
	IsQuoted   bool   // true if token was enclosed in ' or "
	QuoteChar  rune   // ' or " or 0
	Terminated bool   // true if closing quote was matched
}

// ParseTokens splits line into shell-like tokens with support for quotes (' and ") and backslash escapes (\ ).
func ParseTokens(line string) []Token {
	var tokens []Token
	runes := []rune(line)
	n := len(runes)
	i := 0

	for i < n {
		// skip leading spaces
		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}
		if i >= n {
			break
		}

		start := i
		var val strings.Builder
		isQuoted := false
		var quoteChar rune
		terminated := true

		for i < n {
			r := runes[i]
			if quoteChar != 0 {
				if r == quoteChar {
					quoteChar = 0
					terminated = true
					i++
					continue
				}
				if r == '\\' && quoteChar == '"' && i+1 < n && (runes[i+1] == '"' || runes[i+1] == '\\') {
					i++
					val.WriteRune(runes[i])
					i++
					continue
				}
				val.WriteRune(r)
				i++
			} else {
				if r == '\'' || r == '"' {
					isQuoted = true
					quoteChar = r
					terminated = false
					i++
				} else if r == '\\' && i+1 < n {
					i++
					val.WriteRune(runes[i])
					i++
				} else if unicode.IsSpace(r) {
					break
				} else {
					val.WriteRune(r)
					i++
				}
			}
		}

		tokens = append(tokens, Token{
			Value:      val.String(),
			Raw:        string(runes[start:i]),
			Start:      start,
			End:        i,
			IsQuoted:   isQuoted,
			QuoteChar:  quoteChar,
			Terminated: terminated,
		})
	}
	return tokens
}

// FindWordAtCursor finds the token and word prefix at cursor rune position `pos`.
// Returns (word, wordStart, isQuoted).
func FindWordAtCursor(line string, pos int) (word string, wordStart int, isQuoted bool) {
	runes := []rune(line)
	if pos > len(runes) {
		pos = len(runes)
	}
	if pos == 0 {
		return "", 0, false
	}
	tokens := ParseTokens(string(runes[:pos]))
	if len(tokens) == 0 {
		return "", pos, false
	}
	// Check if cursor is directly adjacent to or inside the last token
	last := tokens[len(tokens)-1]
	if last.End == pos {
		return last.Value, last.Start, last.IsQuoted
	}
	// Otherwise cursor is after whitespace
	return "", pos, false
}
