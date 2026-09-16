package readline

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Word operations and Transpose tests ──────────────────────────────────────

func TestBuffer_WordOperations(t *testing.T) {
	b := NewLineBuffer()
	b.Set("hello beautiful world")
	b.MoveHome()

	// MoveWordRight
	if !b.MoveWordRight() || b.Pos() != 5 {
		t.Fatalf("MoveWordRight 1: want pos=5 got %d", b.Pos())
	}
	if !b.MoveWordRight() || b.Pos() != 15 {
		t.Fatalf("MoveWordRight 2: want pos=15 got %d", b.Pos())
	}
	if !b.MoveWordRight() || b.Pos() != 21 {
		t.Fatalf("MoveWordRight 3: want pos=21 got %d", b.Pos())
	}
	if b.MoveWordRight() {
		t.Fatal("MoveWordRight at end should return false")
	}

	// MoveWordLeft
	if !b.MoveWordLeft() || b.Pos() != 16 {
		t.Fatalf("MoveWordLeft 1: want pos=16 got %d", b.Pos())
	}
	if !b.MoveWordLeft() || b.Pos() != 6 {
		t.Fatalf("MoveWordLeft 2: want pos=6 got %d", b.Pos())
	}
	if !b.MoveWordLeft() || b.Pos() != 0 {
		t.Fatalf("MoveWordLeft 3: want pos=0 got %d", b.Pos())
	}

	// DeleteWordAfter
	b.Set("foo bar baz")
	b.MoveHome()
	del, ok := b.DeleteWordAfterText()
	if !ok || del != "foo" || b.String() != " bar baz" {
		t.Fatalf("DeleteWordAfter: want 'foo' remaining ' bar baz', got %q %q", del, b.String())
	}

	// DeleteWordBefore
	b.MoveEnd()
	del, ok = b.DeleteWordBeforeText()
	if !ok || del != "baz" || b.String() != " bar " {
		t.Fatalf("DeleteWordBefore: want 'baz' remaining ' bar ', got %q %q", del, b.String())
	}
}

func TestBuffer_Transpose(t *testing.T) {
	b := NewLineBuffer()
	b.Set("ab")
	b.MoveEnd()
	if !b.Transpose() || b.String() != "ba" {
		t.Fatalf("Transpose at end: want 'ba', got %q", b.String())
	}

	b.Set("helo")
	b.SetPos(3) // at 'o'
	if !b.Transpose() || b.String() != "heol" {
		t.Fatalf("Transpose mid: want 'heol', got %q", b.String())
	}
}

// ── Unicode zero-width and wide characters ───────────────────────────────────

func TestBuffer_ZeroWidthRunes(t *testing.T) {
	b := NewLineBuffer()
	// 'e' (width 1) + combining acute accent U+0301 (width 0)
	b.Insert('e')
	b.Insert('\u0301')
	if b.Len() != 2 {
		t.Fatalf("want 2 runes, got %d", b.Len())
	}
	if w := b.DisplayWidthTotal(); w != 1 {
		t.Fatalf("combining accent should have width 1 total, got %d", w)
	}

	// Zero-width joiner
	b.Clear()
	b.Insert('\u200D')
	if w := b.DisplayWidthTotal(); w != 0 {
		t.Fatalf("ZWJ should have display width 0, got %d", w)
	}

	// Variation selector
	b.Clear()
	b.Insert('\uFE0F')
	if w := b.DisplayWidthTotal(); w != 0 {
		t.Fatalf("VS16 should have display width 0, got %d", w)
	}

	// Emoji
	b.Clear()
	b.Insert('🚀')
	if w := b.DisplayWidthTotal(); w != 2 {
		t.Fatalf("rocket emoji should have display width 2, got %d", w)
	}
}

// ── KillRing tests ───────────────────────────────────────────────────────────

func TestKillRing(t *testing.T) {
	kr := NewKillRing(3)
	if kr.Top() != "" {
		t.Fatal("empty kill ring should return empty Top")
	}

	kr.Push("kill1")
	kr.Push("kill2")
	kr.Push("kill3")
	kr.Push("kill4")

	if kr.Len() != 3 {
		t.Fatalf("want max 3 items, got %d", kr.Len())
	}
	if kr.Top() != "kill4" {
		t.Fatalf("want Top='kill4', got %q", kr.Top())
	}

	// YankPop
	if yp := kr.YankPop(); yp != "kill3" {
		t.Fatalf("want YankPop='kill3', got %q", yp)
	}
	if yp := kr.YankPop(); yp != "kill2" {
		t.Fatalf("want YankPop='kill2', got %q", yp)
	}
}

// ── UndoStack tests ──────────────────────────────────────────────────────────

func TestUndoRedo(t *testing.T) {
	b := NewLineBuffer()
	u := NewUndoStack(10)

	b.Set("initial")
	u.Save(b)

	b.Set("second")
	u.Save(b)

	b.Set("third")

	// Undo to second
	if !u.Undo(b) || b.String() != "second" {
		t.Fatalf("1st undo: want 'second', got %q", b.String())
	}

	// Undo to initial
	if !u.Undo(b) || b.String() != "initial" {
		t.Fatalf("2nd undo: want 'initial', got %q", b.String())
	}

	// No more undos
	if u.Undo(b) {
		t.Fatal("expected Undo to return false when stack exhausted")
	}

	// Redo to second
	if !u.Redo(b) || b.String() != "second" {
		t.Fatalf("1st redo: want 'second', got %q", b.String())
	}
}

// ── Token and Quote parsing tests ────────────────────────────────────────────

func TestTokens_QuotedAndEscaped(t *testing.T) {
	line := `docker run -v "/path with spaces:/data" 'flag with single' plain\ text`
	tokens := ParseTokens(line)
	expected := []string{"docker", "run", "-v", "/path with spaces:/data", "flag with single", "plain text"}
	if len(tokens) != len(expected) {
		t.Fatalf("want %d tokens, got %d: %+v", len(expected), len(tokens), tokens)
	}
	for i, want := range expected {
		if tokens[i].Value != want {
			t.Errorf("token %d: want %q, got %q", i, want, tokens[i].Value)
		}
	}
}

func TestFindWordAtCursor(t *testing.T) {
	line := "git checkout -b feature"
	word, start, isQuoted := FindWordAtCursor(line, len(line))
	if word != "feature" || start != 16 || isQuoted {
		t.Fatalf("want word='feature' start=16 isQuoted=false, got word=%q start=%d isQuoted=%v", word, start, isQuoted)
	}

	// Cursor right after space
	word, start, _ = FindWordAtCursor(line+" ", len(line)+1)
	if word != "" || start != len(line)+1 {
		t.Fatalf("want empty word after space, got %q start=%d", word, start)
	}
}

// ── Completion tests ─────────────────────────────────────────────────────────

func TestCompletion_PrefixCompleter(t *testing.T) {
	completer := PrefixCompleter("docker", "doctor", "down", "git")
	comps, pLen, err := completer.Complete(context.Background(), "doc", 3)
	if err != nil {
		t.Fatal(err)
	}
	if pLen != 3 {
		t.Fatalf("want prefixLen=3, got %d", pLen)
	}
	if len(comps) != 2 {
		t.Fatalf("want 2 completions, got %d", len(comps))
	}
	if comps[0].Value != "docker" || comps[1].Value != "doctor" {
		t.Fatalf("unexpected completions: %v", comps)
	}
}

func TestCompletion_PathCompleter(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "alpine.sh"), []byte("b"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "beta.go"), []byte("c"), 0600)

	completer := PathCompleter()
	input := filepath.Join(dir, "alp")
	comps, _, err := completer.Complete(context.Background(), input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) != 2 {
		t.Fatalf("want 2 completions, got %d", len(comps))
	}
}

func TestCompletion_FormatGrid(t *testing.T) {
	items := []Completion{
		{Value: "apple"},
		{Value: "banana"},
		{Value: "cherry"},
		{Value: "date"},
	}
	grid := FormatCompletionGrid(items, 80)
	if len(grid) == 0 {
		t.Fatal("expected formatted grid rows")
	}

	// With descriptions
	itemsWithDesc := []Completion{
		{Value: "start", Description: "start the engine"},
		{Value: "stop", Description: "stop the engine"},
	}
	gridDesc := FormatCompletionGrid(itemsWithDesc, 80)
	if len(gridDesc) != 2 || !strings.Contains(gridDesc[0], "start the engine") {
		t.Fatalf("unexpected grid with desc: %v", gridDesc)
	}
}

// ── SearchMode (Ctrl+R / Ctrl+S) tests ───────────────────────────────────────

func TestSearchMode_CtrlR_CtrlS(t *testing.T) {
	h := NewHistory(10)
	h.Push("git clone https://github.com")
	h.Push("cargo build")
	h.Push("git pull")
	h.Push("go test ./...")
	h.Push("git push origin main")

	sm := newSearchMode(h)
	buf := NewLineBuffer()
	buf.Set("my typing")

	sm.Start(buf)
	if !sm.Active() {
		t.Fatal("expected search mode to be active")
	}

	// Type 'g', 'i', 't'
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 'g'}, buf)
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 'i'}, buf)
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 't'}, buf)

	if sm.Match() != "git push origin main" {
		t.Fatalf("want 'git push origin main', got %q", sm.Match())
	}

	// Ctrl+R for older match
	sm.HandleKey(InputEvent{Key: KeyCtrlR}, buf)
	if sm.Match() != "git pull" {
		t.Fatalf("1st Ctrl+R: want 'git pull', got %q", sm.Match())
	}

	sm.HandleKey(InputEvent{Key: KeyCtrlR}, buf)
	if sm.Match() != "git clone https://github.com" {
		t.Fatalf("2nd Ctrl+R: want 'git clone...', got %q", sm.Match())
	}

	// Ctrl+S for newer match
	sm.HandleKey(InputEvent{Key: KeyCtrlS}, buf)
	if sm.Match() != "git pull" {
		t.Fatalf("Ctrl+S: want 'git pull', got %q", sm.Match())
	}

	// Backspace in query
	sm.HandleKey(InputEvent{Key: KeyBackspace}, buf) // query is now "gi"
	if string(sm.query) != "gi" {
		t.Fatalf("want query 'gi', got %q", string(sm.query))
	}

	// Esc cancels and restores original buffer
	sm.HandleKey(InputEvent{Key: KeyEsc}, buf)
	if sm.Active() {
		t.Fatal("expected search mode to be inactive after Esc")
	}
	if buf.String() != "my typing" {
		t.Fatalf("expected original buffer restored, got %q", buf.String())
	}
}

// ── Non-TTY fallback test ───────────────────────────────────────────────────

func TestEditor_NonTTY_Fallback(t *testing.T) {
	// Create an os.Pipe (not a TTY)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	cfg := Config{
		Stdin:  r,
		Stdout: os.Stdout,
		Prompt: "> ",
	}
	ed, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed on pipe input: %v", err)
	}
	defer ed.Close()

	// Write simulated piped input
	go func() {
		_, _ = w.WriteString("echo piped input\nsecond line\n")
		_ = w.Close()
	}()

	line1, err := ed.ReadLine()
	if err != nil || line1 != "echo piped input" {
		t.Fatalf("line 1: want 'echo piped input' err=nil, got %q err=%v", line1, err)
	}

	line2, err := ed.ReadLine()
	if err != nil || line2 != "second line" {
		t.Fatalf("line 2: want 'second line' err=nil, got %q err=%v", line2, err)
	}

	_, err = ed.ReadLine()
	if err != io.EOF {
		t.Fatalf("expected EOF after piped input closed, got %v", err)
	}
}

// ── Password mode test ───────────────────────────────────────────────────────

func TestEditor_Password_Mode(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	cfg := Config{
		Stdin:  r,
		Stdout: os.Stdout,
		Prompt: "Password: ",
	}
	ed, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ed.Close()

	go func() {
		_, _ = w.WriteString("supersecret123\n")
		_ = w.Close()
	}()

	pass, err := ed.ReadPassword()
	if err != nil || pass != "supersecret123" {
		t.Fatalf("ReadPassword: want 'supersecret123' err=nil, got %q err=%v", pass, err)
	}

	// Verify password is NEVER recorded in history
	entries := ed.History()
	for _, e := range entries {
		if strings.Contains(e, "supersecret") {
			t.Fatal("password found in history!")
		}
	}
}

// ── Context cancellation test ───────────────────────────────────────────────

func TestEditor_ContextCancellation(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	cfg := Config{
		Stdin:  r,
		Stdout: os.Stdout,
	}
	ed, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ed.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = ed.ReadLineContext(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("took too long to cancel")
	}
}

// ── Custom KeyMap test ───────────────────────────────────────────────────────

func TestEditor_CustomKeyMap(t *testing.T) {
	km := make(KeyMap)
	customTriggered := false
	km.Bind(KeyCtrlX, func(e *Editor, buf *LineBuffer) (bool, string, error) {
		customTriggered = true
		buf.Set("custom action executed")
		return false, "", nil
	})

	cfg := Config{
		KeyMap: km,
	}
	ed, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ed.Close()

	buf := NewLineBuffer()
	ed.handleEvent(InputEvent{Key: KeyCtrlX}, buf)
	if !customTriggered || buf.String() != "custom action executed" {
		t.Fatalf("custom keymap not triggered: %v %q", customTriggered, buf.String())
	}
}

// ── History compaction test ─────────────────────────────────────────────────

func TestHistory_CompactFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compact_hist.txt")

	h := NewHistory(3)
	for _, s := range []string{"1", "2", "3", "4", "5"} {
		h.Push(s)
	}
	if err := h.CompactFile(path); err != nil {
		t.Fatal(err)
	}

	h2 := NewHistory(10)
	if err := h2.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h2.Len() != 3 {
		t.Fatalf("want 3 entries after compact, got %d", h2.Len())
	}
}
