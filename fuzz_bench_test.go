package readline

import (
	"fmt"
	"strings"
	"testing"
)

// ── Fuzz Tests ───────────────────────────────────────────────────────────────

func FuzzEscapeParser(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x1B, '[', 'A'})
	f.Add([]byte{0x1B, '[', '3', '~'})
	f.Add([]byte{0x1B, 'b'})
	f.Add([]byte{0x1B, '[', '2', '0', '0', '~'})
	f.Add([]byte("hello world"))
	f.Add([]byte{0x00, 0xFF, 0x1B, 0x1B, 0x5B})

	f.Fuzz(func(t *testing.T, data []byte) {
		var p EscapeParser
		for _, b := range data {
			p.Feed(b)
		}
	})
}

func FuzzParseTokens(f *testing.F) {
	f.Add(`docker run -v "/foo bar:/data"`)
	f.Add(`simple text`)
	f.Add(`'unclosed single quote`)
	f.Add(`"unclosed double quote \`)
	f.Add(`escaped\ space\ here`)

	f.Fuzz(func(t *testing.T, line string) {
		tokens := ParseTokens(line)
		for _, tok := range tokens {
			if tok.Start < 0 || tok.End < tok.Start {
				t.Fatalf("invalid token boundaries: %+v", tok)
			}
		}
	})
}

func FuzzRuneDisplayWidth(f *testing.F) {
	f.Add(rune('a'))
	f.Add(rune('日'))
	f.Add(rune('🚀'))
	f.Add(rune('\u0301'))
	f.Add(rune(0))

	f.Fuzz(func(t *testing.T, r rune) {
		w := runeDisplayWidth(r)
		if w < 0 || w > 2 {
			t.Fatalf("unexpected width %d for rune %U", w, r)
		}
	})
}

// ── Benchmarks ───────────────────────────────────────────────────────────────

func BenchmarkBufferInsert(b *testing.B) {
	buf := NewLineBuffer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Insert('a')
		if buf.Len() > 1000 {
			buf.Clear()
		}
	}
}

func BenchmarkBufferDisplayWidth(b *testing.B) {
	buf := NewLineBuffer()
	buf.Set("Hello 世界 🚀 combining e\u0301 test line for display width benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.DisplayWidthTotal()
	}
}

func BenchmarkEscapeParser(b *testing.B) {
	seq := []byte{0x1B, '[', '1', ';', '5', 'C'}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p EscapeParser
		for _, bt := range seq {
			p.Feed(bt)
		}
	}
}

func BenchmarkHistorySearch(b *testing.B) {
	h := NewHistory(1000)
	for i := 0; i < 500; i++ {
		h.Push(fmt.Sprintf("command number %d with some text", i))
	}
	h.Push("git commit -m message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ResetSearch()
		_, _ = h.SearchUp("git")
	}
}

func BenchmarkParseTokens(b *testing.B) {
	line := `docker run --rm -v "/my storage/data:/app" -e DEBUG=true alpine:latest sh -c "echo 'hello world'"`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseTokens(line)
	}
}

func BenchmarkRendererRedraw(b *testing.B) {
	var sb strings.Builder
	rend := NewRenderer(&sb, "app> ")
	buf := NewLineBuffer()
	buf.Set("my long input command line for redrawing benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.Reset()
		rend.Redraw(buf)
	}
}
