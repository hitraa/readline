package readline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── LineBuffer tests ──────────────────────────────────────────────────────────

func TestBuffer_InsertAndString(t *testing.T) {
	b := NewLineBuffer()
	for _, r := range "hello" {
		b.Insert(r)
	}
	if got := b.String(); got != "hello" {
		t.Fatalf("want %q got %q", "hello", got)
	}
	if b.Pos() != 5 {
		t.Fatalf("want pos=5 got %d", b.Pos())
	}
}

func TestBuffer_InsertAtMiddle(t *testing.T) {
	b := NewLineBuffer()
	b.Set("helo")
	b.MoveHome()
	b.MoveRight()
	b.MoveRight()
	b.Insert('l')
	if b.String() != "hello" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBuffer_Backspace(t *testing.T) {
	b := NewLineBuffer()
	b.Set("hello")
	b.Backspace()
	if b.String() != "hell" || b.Pos() != 4 {
		t.Fatalf("got %q pos=%d", b.String(), b.Pos())
	}
	// Backspace at pos 0 is a no-op.
	b.MoveHome()
	if b.Backspace() {
		t.Fatal("expected false from Backspace at pos 0")
	}
}

func TestBuffer_Delete(t *testing.T) {
	b := NewLineBuffer()
	b.Set("hello")
	b.MoveHome()
	b.Delete()
	if b.String() != "ello" || b.Pos() != 0 {
		t.Fatalf("got %q pos=%d", b.String(), b.Pos())
	}
}

func TestBuffer_DeleteWordBefore(t *testing.T) {
	b := NewLineBuffer()
	b.Set("foo bar baz")
	// cursor is at the end
	b.DeleteWordBefore()
	if b.String() != "foo bar " {
		t.Fatalf("got %q", b.String())
	}
	b.DeleteWordBefore()
	if b.String() != "foo " {
		t.Fatalf("got %q", b.String())
	}
}

func TestBuffer_KillToEnd(t *testing.T) {
	b := NewLineBuffer()
	b.Set("hello world")
	b.MoveHome()
	b.MoveRight() // pos=1
	b.MoveRight() // pos=2
	b.MoveRight() // pos=3
	b.MoveRight() // pos=4
	b.MoveRight() // pos=5  ("hello")
	b.KillToEnd()
	if b.String() != "hello" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBuffer_KillToStart(t *testing.T) {
	b := NewLineBuffer()
	b.Set("hello world")
	// cursor at end; kill to start
	b.KillToStart()
	if b.String() != "" || b.Pos() != 0 {
		t.Fatalf("got %q pos=%d", b.String(), b.Pos())
	}
}

func TestBuffer_UTF8(t *testing.T) {
	b := NewLineBuffer()
	// Insert Japanese characters
	for _, r := range "日本語" {
		b.Insert(r)
	}
	if b.String() != "日本語" {
		t.Fatalf("got %q", b.String())
	}
	if b.Len() != 3 {
		t.Fatalf("want 3 runes got %d", b.Len())
	}
}

func TestBuffer_DisplayWidth_WideChars(t *testing.T) {
	b := NewLineBuffer()
	for _, r := range "日本" { // each CJK char is width 2
		b.Insert(r)
	}
	if w := b.DisplayWidthTotal(); w != 4 {
		t.Fatalf("want display width 4 got %d", w)
	}
}

// ── History tests ─────────────────────────────────────────────────────────────

func TestHistory_PushAndUp(t *testing.T) {
	h := NewHistory(10)
	h.Push("foo")
	h.Push("bar")
	h.Push("baz")

	got, ok := h.Up()
	if !ok || got != "baz" {
		t.Fatalf("want baz ok=true, got %q ok=%v", got, ok)
	}
	got, ok = h.Up()
	if !ok || got != "bar" {
		t.Fatalf("want bar ok=true, got %q ok=%v", got, ok)
	}
}

func TestHistory_DownRestoresPending(t *testing.T) {
	h := NewHistory(10)
	h.Push("foo")
	h.Push("bar")
	h.SetPending("partial")

	h.Up() // go to "bar"
	got, ok := h.Down()
	if !ok || got != "partial" {
		t.Fatalf("want partial ok=true, got %q ok=%v", got, ok)
	}
}

func TestHistory_DeduplicatesConsecutive(t *testing.T) {
	h := NewHistory(10)
	h.Push("foo")
	h.Push("foo")
	if h.Len() != 1 {
		t.Fatalf("want 1 entry, got %d", h.Len())
	}
}

func TestHistory_MaxSize(t *testing.T) {
	h := NewHistory(3)
	for _, s := range []string{"a", "b", "c", "d"} {
		h.Push(s)
	}
	if h.Len() != 3 {
		t.Fatalf("want 3, got %d", h.Len())
	}
	entries := h.Entries()
	if entries[0] != "b" {
		t.Fatalf("oldest should be 'b', got %q", entries[0])
	}
}

func TestHistory_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hist.txt")

	h1 := NewHistory(100)
	h1.Push("cmd1")
	h1.Push("cmd2")
	if err := h1.SaveFile(path, h1.Entries()); err != nil {
		t.Fatal(err)
	}

	h2 := NewHistory(100)
	if err := h2.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h2.Len() != 2 {
		t.Fatalf("want 2, got %d", h2.Len())
	}
	entries := h2.Entries()
	if entries[0] != "cmd1" || entries[1] != "cmd2" {
		t.Fatalf("unexpected entries: %v", entries)
	}
}

func TestHistory_LoadMissingFile(t *testing.T) {
	h := NewHistory(10)
	// Should not error on missing file.
	if err := h.LoadFile("/nonexistent/file/path"); err != nil {
		t.Fatal(err)
	}
}

func TestHistory_SearchUp_PrefixMatch(t *testing.T) {
	h := NewHistory(10)
	for _, s := range []string{"git add .", "git commit -m init", "go build", "git push"} {
		h.Push(s)
	}
	h.SetPending("git")

	got, ok := h.SearchUp("git")
	if !ok || got != "git push" {
		t.Fatalf("1st SearchUp: want 'git push' ok=true, got %q ok=%v", got, ok)
	}
	got, ok = h.SearchUp("git")
	if !ok || got != "git commit -m init" {
		t.Fatalf("2nd SearchUp: want 'git commit -m init' ok=true, got %q ok=%v", got, ok)
	}
	got, ok = h.SearchUp("git")
	if !ok || got != "git add ." {
		t.Fatalf("3rd SearchUp: want 'git add .' ok=true, got %q ok=%v", got, ok)
	}
	// No more "git" entries above.
	_, ok = h.SearchUp("git")
	if ok {
		t.Fatal("expected ok=false when no more matches")
	}
}

func TestHistory_SearchDown_PrefixMatch(t *testing.T) {
	h := NewHistory(10)
	for _, s := range []string{"git add .", "go build", "git commit -m v2"} {
		h.Push(s)
	}
	h.SetPending("git")
	// Start from beginning (idx=0 after SearchUp exhausts)
	h.SearchUp("git") // → "git commit -m v2", idx=2
	h.SearchUp("git") // → "git add .",        idx=0

	got, ok := h.SearchDown("git")
	if !ok || got != "git commit -m v2" {
		t.Fatalf("SearchDown: want 'git commit -m v2' ok=true, got %q ok=%v", got, ok)
	}
}

func TestHistory_SearchDown_Exhausted_RestoresPending(t *testing.T) {
	h := NewHistory(10)
	h.Push("git add .")
	h.SetPending("git")
	h.SearchUp("git") // → "git add .", idx=0

	// SearchDown from idx=0 — no "git" entry forward.
	got, ok := h.SearchDown("git")
	if ok {
		t.Fatal("expected ok=false when search exhausted")
	}
	if got != "git" {
		t.Fatalf("expected pending 'git' restored, got %q", got)
	}
}

func TestHistory_SearchUp_NoMatch(t *testing.T) {
	h := NewHistory(10)
	h.Push("go test")
	h.Push("go build")

	_, ok := h.SearchUp("git")
	if ok {
		t.Fatal("expected ok=false for no-match prefix")
	}
}

func TestHistory_ResetSearch(t *testing.T) {
	h := NewHistory(10)
	h.Push("git add .")
	h.SetPending("git")
	h.SearchUp("git") // idx moves to 0

	h.ResetSearch()

	if h.pending != "" {
		t.Fatalf("expected pending cleared, got %q", h.pending)
	}
	if h.idx != h.Len() {
		t.Fatalf("expected idx at end (%d), got %d", h.Len(), h.idx)
	}
}

// TestHistory_StalePrefixAfterErase covers the bug where erasing the typed
// prefix and pressing PageUp again continued searching with the old locked
// prefix instead of starting fresh from the new (empty or different) buffer.
func TestHistory_StalePrefixAfterErase(t *testing.T) {
	h := NewHistory(20)
	for _, s := range []string{"go build", "git status", "git add ."} {
		h.Push(s)
	}

	// Session 1: search with "git"
	h.SetPending("git")
	got, ok := h.SearchUp("git")
	if !ok || got != "git add ." {
		t.Fatalf("session1 PageUp: want 'git add .', got %q ok=%v", got, ok)
	}

	// User erases everything → buffer is now "".
	// Editor detects HasPrefix("", "git") == false and calls ResetSearch.
	h.ResetSearch()

	// Session 2: fresh search from "" (user presses PageUp on empty buffer).
	// Should start from the newest entry, not continue the "git" search.
	h.SetPending("")
	// Empty prefix → Up() used instead of SearchUp; verify idx is at end.
	if h.idx != h.Len() {
		t.Fatalf("after ResetSearch idx should be %d, got %d", h.Len(), h.idx)
	}
	got, ok = h.Up()
	if !ok || got != "git add ." {
		t.Fatalf("session2 Up: want 'git add .' (newest), got %q ok=%v", got, ok)
	}

	// Session 3: user types "go" (different prefix).
	h.ResetSearch()
	h.SetPending("go")
	got, ok = h.SearchUp("go")
	if !ok || got != "go build" {
		t.Fatalf("session3 SearchUp('go'): want 'go build', got %q ok=%v", got, ok)
	}
}

// TestHistory_SearchUp_MultiplePageUp is the regression test for the bug where
// successive PageUp presses only showed the most recent match because the
// prefix was re-read from the buffer (which now held the full history entry)
// instead of from the originally typed text locked in pending.
//
// Scenario: history = ["git add .", "git commit -m \"\"", "git status", "clear"]
// User types "git", presses PageUp three times → must cycle through all three
// "git" entries newest-first, then PageDown must bring them back.
func TestHistory_SearchUp_MultiplePageUp_RegressBug(t *testing.T) {
	h := NewHistory(20)
	for _, s := range []string{`git add .`, `git commit -m ""`, `git status`, `clear`} {
		h.Push(s)
	}

	// Simulate: user types "git", first PageUp — prefix locked in as pending.
	prefix := "git"
	h.SetPending(prefix)

	got, ok := h.SearchUp(prefix)
	if !ok || got != "git status" {
		t.Fatalf("1st PageUp: want 'git status', got %q ok=%v", got, ok)
	}

	// BUG was here: the old code did `prefix = buf.String()` which would now
	// return "git status", calling SearchUp("git status") → no match.
	// Correct code reuses h.pending = "git".
	got, ok = h.SearchUp(prefix) // prefix is still "git"
	if !ok || got != `git commit -m ""` {
		t.Fatalf(`2nd PageUp: want 'git commit -m ""', got %q ok=%v`, got, ok)
	}

	got, ok = h.SearchUp(prefix)
	if !ok || got != "git add ." {
		t.Fatalf("3rd PageUp: want 'git add .', got %q ok=%v", got, ok)
	}

	// No more "git" entries — further PageUp should return false (no change).
	_, ok = h.SearchUp(prefix)
	if ok {
		t.Fatal("4th PageUp: expected no more matches")
	}

	// PageDown: walk back forward through matches.
	got, ok = h.SearchDown(prefix)
	if !ok || got != `git commit -m ""` {
		t.Fatalf(`1st PageDown: want 'git commit -m ""', got %q ok=%v`, got, ok)
	}

	got, ok = h.SearchDown(prefix)
	if !ok || got != "git status" {
		t.Fatalf("2nd PageDown: want 'git status', got %q ok=%v", got, ok)
	}

	// One more PageDown exhausts forward matches; pending ("git") is restored.
	got, ok = h.SearchDown(prefix)
	if ok {
		t.Fatal("3rd PageDown: expected exhausted (ok=false)")
	}
	if got != "git" {
		t.Fatalf("3rd PageDown: want pending 'git' restored, got %q", got)
	}
}

// ── EscapeParser tests ────────────────────────────────────────────────────────

func feedAll(p *EscapeParser, bs []byte) (Key, rune, bool) {
	var key Key
	var r rune
	var complete bool
	for _, b := range bs {
		key, r, complete = p.Feed(b)
		if complete {
			return key, r, true
		}
	}
	return key, r, complete
}

func TestEscapeParser_ArrowKeys(t *testing.T) {
	tests := []struct {
		seq []byte
		key Key
	}{
		{[]byte{'\x1b', '[', 'A'}, KeyArrowUp},
		{[]byte{'\x1b', '[', 'B'}, KeyArrowDown},
		{[]byte{'\x1b', '[', 'C'}, KeyArrowRight},
		{[]byte{'\x1b', '[', 'D'}, KeyArrowLeft},
	}
	for _, tt := range tests {
		var p EscapeParser
		got, _, ok := feedAll(&p, tt.seq)
		if !ok || got != tt.key {
			t.Errorf("seq %q: want key %d ok=true, got key=%d ok=%v", tt.seq, tt.key, got, ok)
		}
	}
}

func TestEscapeParser_HomeEnd(t *testing.T) {
	tests := []struct {
		seq []byte
		key Key
	}{
		{[]byte{'\x1b', '[', 'H'}, KeyHome},
		{[]byte{'\x1b', '[', 'F'}, KeyEnd},
		{[]byte{'\x1b', 'O', 'H'}, KeyHome},
		{[]byte{'\x1b', 'O', 'F'}, KeyEnd},
		{[]byte{'\x1b', '[', '1', '~'}, KeyHome},
		{[]byte{'\x1b', '[', '4', '~'}, KeyEnd},
	}
	for _, tt := range tests {
		var p EscapeParser
		got, _, ok := feedAll(&p, tt.seq)
		if !ok || got != tt.key {
			t.Errorf("seq %q: want key %d ok=true, got key=%d ok=%v", tt.seq, tt.key, got, ok)
		}
	}
}

func TestEscapeParser_Delete(t *testing.T) {
	var p EscapeParser
	got, _, ok := feedAll(&p, []byte{'\x1b', '[', '3', '~'})
	if !ok || got != KeyDelete {
		t.Fatalf("want KeyDelete, got %d ok=%v", got, ok)
	}
}

func TestEscapeParser_PageUpDown(t *testing.T) {
	tests := []struct {
		seq []byte
		key Key
	}{
		{[]byte{'\x1b', '[', '5', '~'}, KeyPageUp},
		{[]byte{'\x1b', '[', '6', '~'}, KeyPageDown},
	}
	for _, tt := range tests {
		var p EscapeParser
		got, _, ok := feedAll(&p, tt.seq)
		if !ok || got != tt.key {
			t.Errorf("seq %q: want key %d ok=true, got key=%d ok=%v", tt.seq, tt.key, got, ok)
		}
	}
}

func TestEscapeParser_CtrlKeys(t *testing.T) {
	tests := []struct {
		b   byte
		key Key
	}{
		{0x01, KeyCtrlA},
		{0x03, KeyCtrlC},
		{0x04, KeyCtrlD},
		{0x05, KeyCtrlE},
		{0x17, KeyCtrlW},
	}
	for _, tt := range tests {
		var p EscapeParser
		got, _, ok := p.Feed(tt.b)
		if !ok || got != tt.key {
			t.Errorf("byte 0x%02x: want key %d ok=true, got key=%d ok=%v", tt.b, tt.key, got, ok)
		}
	}
}

func TestEscapeParser_PrintableRune(t *testing.T) {
	var p EscapeParser
	got, r, ok := p.Feed('A')
	if !ok || got != KeyRune || r != 'A' {
		t.Fatalf("want KeyRune 'A', got key=%d rune=%c ok=%v", got, r, ok)
	}
}

// ── UTF-8 decoder test ────────────────────────────────────────────────────────

func TestDecodeUTF8Rune(t *testing.T) {
	// Encode "日" (U+65E5) as UTF-8: 0xE6 0x97 0xA5
	encoded := []byte("日")
	idx := 1
	readByte := func() (byte, error) {
		b := encoded[idx]
		idx++
		return b, nil
	}

	r, n, err := decodeUTF8Rune(encoded[0], readByte)
	if err != nil {
		t.Fatal(err)
	}
	if r != '日' || n != 3 {
		t.Fatalf("want rune '日' n=3, got rune=%c n=%d", r, n)
	}
}

// ── Renderer test (using a buffer as io.Writer) ───────────────────────────────

func TestRenderer_PromptDisplayLen_StripANSI(t *testing.T) {
	rend := NewRenderer(os.Stdout, "\033[1;32mtest\033[0m> ")
	// "test" = 4 chars, "> " = 2 chars → 6
	if got := rend.PromptDisplayLen(); got != 6 {
		t.Fatalf("want 6, got %d", got)
	}
}

func TestRenderer_RedrawOutput(t *testing.T) {
	var sb strings.Builder
	w := &testWriter{sb: &sb}
	rend := NewRenderer(w, "> ")
	buf := NewLineBuffer()
	buf.Set("hello")
	rend.Redraw(buf)
	out := sb.String()
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected 'hello' in output, got %q", out)
	}
	if !strings.HasPrefix(out, "\r> \033[K") {
		t.Fatalf("expected redraw prefix, got %q", out)
	}
}

type testWriter struct{ sb *strings.Builder }

func (w *testWriter) Write(p []byte) (int, error) {
	return w.sb.Write(p)
}

func TestHistory_Reset(t *testing.T) {
	h := NewHistory(10)
	h.Push("cmd1")
	h.SetPending("pending cmd")
	h.Reset()
	if h.pending != "" {
		t.Fatalf("expected pending to be empty after Reset, got %q", h.pending)
	}
	if h.idx != h.Len() {
		t.Fatalf("expected idx to be %d, got %d", h.Len(), h.idx)
	}
}

func TestEscapeParser_SequenceControls(t *testing.T) {
	var p EscapeParser
	if p.InSequence() {
		t.Fatal("expected InSequence to be false initially")
	}

	// Start sequence
	_, _, ok := p.Feed(0x1B) // ESC
	if ok {
		t.Fatal("expected escape sequence to be incomplete")
	}
	if !p.InSequence() {
		t.Fatal("expected InSequence to be true after ESC")
	}

	// Reset sequence
	p.Reset()
	if p.InSequence() {
		t.Fatal("expected InSequence to be false after Reset")
	}

	// Escape parser cancellation (ESC followed by non-control)
	_, _, ok = p.Feed(0x1B) // ESC
	if ok {
		t.Fatal("expected incomplete")
	}
	got, _, ok := p.Feed('x') // invalid escape char 'x'
	if !ok || got != KeyEsc {
		t.Fatalf("expected KeyEsc on cancellation, got key=%d ok=%v", got, ok)
	}
	if p.InSequence() {
		t.Fatal("expected InSequence to be false after cancellation")
	}
}

func TestEscapeParser_CSIPagingAndHomeEnd(t *testing.T) {
	tests := []struct {
		seq []byte
		key Key
	}{
		{[]byte{'\x1b', '[', '7', '~'}, KeyHome},
		{[]byte{'\x1b', '[', '8', '~'}, KeyEnd},
		{[]byte{'\x1b', '[', '9', '9', 'x'}, KeyEsc}, // Unknown CSI sequence
	}
	for _, tt := range tests {
		var p EscapeParser
		got, _, ok := feedAll(&p, tt.seq)
		if !ok || got != tt.key {
			t.Errorf("seq %q: want key %d ok=true, got key=%d ok=%v", tt.seq, tt.key, got, ok)
		}
	}
}

func TestRenderer_ClearScreen(t *testing.T) {
	var sb strings.Builder
	w := &testWriter{sb: &sb}
	rend := NewRenderer(w, "> ")
	buf := NewLineBuffer()
	buf.Set("test")
	rend.ClearScreen(buf)
	out := sb.String()
	if !strings.Contains(out, "\033[2J\033[H") {
		t.Fatalf("expected clear screen escape code, got %q", out)
	}
	if !strings.Contains(out, "test") {
		t.Fatalf("expected 'test' redrawn, got %q", out)
	}
}

func TestRenderer_NewLine(t *testing.T) {
	var sb strings.Builder
	w := &testWriter{sb: &sb}
	rend := NewRenderer(w, "> ")
	rend.NewLine()
	out := sb.String()
	if out != "\r\n" {
		t.Fatalf("expected newline sequence, got %q", out)
	}
}

func TestRenderer_PrintBanner(t *testing.T) {
	var sb strings.Builder
	w := &testWriter{sb: &sb}
	rend := NewRenderer(w, "> ")
	rend.PrintBanner("Banner Text")
	out := sb.String()
	if out != "Banner Text\r\n" {
		t.Fatalf("expected banner with CR+LF, got %q", out)
	}
}

func TestRenderer_Write(t *testing.T) {
	var sb strings.Builder
	w := &testWriter{sb: &sb}
	rend := NewRenderer(w, "> ")
	n, err := rend.Write([]byte("raw text"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 8 {
		t.Fatalf("expected n=8, got %d", n)
	}
	if sb.String() != "raw text" {
		t.Fatalf("expected 'raw text', got %q", sb.String())
	}
}

