package readline

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// ── inputReader and EscapeParser Coverage ────────────────────────────────────

func TestInputReader_AllEvents(t *testing.T) {
	// Construct a byte stream with various keys:
	// - "hi"
	// - ArrowUp (\x1b[A), ArrowDown (\x1b[B), ArrowRight (\x1b[C), ArrowLeft (\x1b[D)
	// - Home (\x1b[H), End (\x1b[F)
	// - Insert (\x1b[2~), Delete (\x1b[3~), PageUp (\x1b[5~), PageDown (\x1b[6~)
	// - Ctrl+Left (\x1b[1;5D), Ctrl+Right (\x1b[1;5C)
	// - BackTab (\x1b[Z)
	// - Alt+B (\x1bb), Alt+F (\x1bf), Alt+D (\x1bd), Alt+Y (\x1by), Alt+Backspace (\x1b\x7f)
	// - Bracketed paste (\x1b[200~hello paste\x1b[201~)
	// - UTF-8 character "日"
	// - Lone ESC followed by 'x' (testing pending byte pop)
	var input bytes.Buffer
	input.WriteString("hi")
	input.WriteString("\x1b[A\x1b[B\x1b[C\x1b[D")
	input.WriteString("\x1b[H\x1b[F")
	input.WriteString("\x1b[2~\x1b[3~\x1b[5~\x1b[6~")
	input.WriteString("\x1b[1;5D\x1b[1;5C")
	input.WriteString("\x1b[Z")
	input.WriteString("\x1bb\x1bf\x1bd\x1by\x1b\x7f")
	input.WriteString("\x1b[200~hello paste\x1b[201~")
	input.WriteString("日")
	input.WriteString("\x1bx")

	ir := newInputReader(&input)

	expectedKeys := []Key{
		KeyRune, KeyRune, // 'h', 'i'
		KeyArrowUp, KeyArrowDown, KeyArrowRight, KeyArrowLeft,
		KeyHome, KeyEnd,
		KeyInsert, KeyDelete, KeyPageUp, KeyPageDown,
		KeyCtrlLeft, KeyCtrlRight,
		KeyBackTab,
		KeyAltB, KeyAltF, KeyAltD, KeyAltY, KeyAltBackspace,
		KeyPasteStart,
		KeyRune, // '日'
		KeyEsc, KeyRune, // lone ESC, then 'x'
	}

	for i, wantKey := range expectedKeys {
		evt, err := ir.ReadEvent()
		if err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
		if evt.Key != wantKey {
			t.Fatalf("step %d: want key %d, got %d", i, wantKey, evt.Key)
		}
		if wantKey == KeyPasteStart && evt.Paste != "hello paste" {
			t.Fatalf("step %d: want paste 'hello paste', got %q", i, evt.Paste)
		}
	}

	// EOF check
	_, err := ir.ReadEvent()
	if err != io.EOF {
		t.Fatalf("expected EOF at stream end, got %v", err)
	}
}

func TestEscapeParser_Pending(t *testing.T) {
	var p EscapeParser
	if p.HasPending() {
		t.Fatal("new parser should not have pending")
	}
	_, _, ok := p.PopPending()
	if ok {
		t.Fatal("PopPending on empty should return false")
	}

	// Feed ESC then 'z'
	p.Feed(0x1B)
	k, _, ok := p.Feed('z')
	if !ok || k != KeyEsc {
		t.Fatalf("want KeyEsc, got %d ok=%v", k, ok)
	}
	if !p.HasPending() {
		t.Fatal("expected pending byte 'z'")
	}
	k2, r2, ok2 := p.PopPending()
	if !ok2 || k2 != KeyRune || r2 != 'z' {
		t.Fatalf("want 'z', got %d %c", k2, r2)
	}
}

// ── Editor handleEvent Full Coverage ─────────────────────────────────────────

func TestEditor_HandleEvent_FullCoverage(t *testing.T) {
	var out bytes.Buffer
	cfg := Config{
		Prompt: "> ",
		Stdout: os.Stdout,
		Completer: PrefixCompleter("alpha", "alpine", "beta"),
	}
	ed, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ed.Close()
	ed.renderer.out = &out

	buf := NewLineBuffer()

	// 1. KeyRune insertion
	for _, r := range "hello world" {
		ed.handleEvent(InputEvent{Key: KeyRune, Rune: r}, buf)
	}
	if buf.String() != "hello world" {
		t.Fatalf("got %q", buf.String())
	}

	// 2. Transpose (Ctrl+T)
	ed.handleEvent(InputEvent{Key: KeyCtrlT}, buf)
	if buf.String() != "hello wordl" {
		t.Fatalf("transpose: got %q", buf.String())
	}

	// 3. MoveHome / MoveEnd
	ed.handleEvent(InputEvent{Key: KeyHome}, buf)
	if buf.Pos() != 0 {
		t.Fatalf("MoveHome pos=%d", buf.Pos())
	}
	ed.handleEvent(InputEvent{Key: KeyEnd}, buf)
	if buf.Pos() != buf.Len() {
		t.Fatalf("MoveEnd pos=%d", buf.Pos())
	}

	// 4. MoveLeft / MoveRight
	ed.handleEvent(InputEvent{Key: KeyArrowLeft}, buf)
	if buf.Pos() != buf.Len()-1 {
		t.Fatalf("ArrowLeft pos=%d", buf.Pos())
	}
	ed.handleEvent(InputEvent{Key: KeyArrowRight}, buf)
	if buf.Pos() != buf.Len() {
		t.Fatalf("ArrowRight pos=%d", buf.Pos())
	}

	// 5. MoveWordLeft / MoveWordRight
	ed.handleEvent(InputEvent{Key: KeyAltB}, buf)
	if buf.Pos() != 6 {
		t.Fatalf("AltB pos=%d", buf.Pos())
	}
	ed.handleEvent(InputEvent{Key: KeyAltF}, buf)
	if buf.Pos() != buf.Len() {
		t.Fatalf("AltF pos=%d", buf.Pos())
	}

	// 6. DeleteWordBefore (Ctrl+W)
	ed.handleEvent(InputEvent{Key: KeyCtrlW}, buf)
	if buf.String() != "hello " {
		t.Fatalf("Ctrl+W: got %q", buf.String())
	}
	if ed.killRing.Top() != "wordl" {
		t.Fatalf("killring top: got %q", ed.killRing.Top())
	}

	// 7. Yank (Ctrl+Y)
	ed.handleEvent(InputEvent{Key: KeyCtrlY}, buf)
	if buf.String() != "hello wordl" {
		t.Fatalf("Ctrl+Y: got %q", buf.String())
	}

	// 8. KillToEnd (Ctrl+K)
	buf.SetPos(5)
	ed.handleEvent(InputEvent{Key: KeyCtrlK}, buf)
	if buf.String() != "hello" {
		t.Fatalf("Ctrl+K: got %q", buf.String())
	}
	if ed.killRing.Top() != " wordl" {
		t.Fatalf("killring top: got %q", ed.killRing.Top())
	}

	// 9. KillToStart (Ctrl+U)
	ed.handleEvent(InputEvent{Key: KeyCtrlU}, buf)
	if buf.String() != "" {
		t.Fatalf("Ctrl+U: got %q", buf.String())
	}

	// 10. Undo (Ctrl+_)
	ed.handleEvent(InputEvent{Key: KeyCtrlUnderscore}, buf)
	if buf.String() != "hello" {
		t.Fatalf("Undo: got %q", buf.String())
	}

	// 11. DeleteWordAfter (Alt+D)
	buf.Set("one two three")
	buf.MoveHome()
	ed.handleEvent(InputEvent{Key: KeyAltD}, buf)
	if buf.String() != " two three" {
		t.Fatalf("Alt+D: got %q", buf.String())
	}

	// 12. YankPop (Alt+Y)
	ed.handleEvent(InputEvent{Key: KeyCtrlY}, buf) // yanks "one"
	ed.handleEvent(InputEvent{Key: KeyAltY}, buf)  // pops previous kill

	// 13. Backspace & Delete
	buf.Set("abc")
	buf.MoveEnd()
	ed.handleEvent(InputEvent{Key: KeyBackspace}, buf)
	if buf.String() != "ab" {
		t.Fatalf("Backspace: got %q", buf.String())
	}
	buf.MoveHome()
	ed.handleEvent(InputEvent{Key: KeyDelete}, buf)
	if buf.String() != "b" {
		t.Fatalf("Delete: got %q", buf.String())
	}

	// 14. Ctrl+D on empty line -> EOF
	buf.Clear()
	done, _, err := ed.handleEvent(InputEvent{Key: KeyCtrlD}, buf)
	if !done || err != io.EOF {
		t.Fatalf("Ctrl+D on empty: want done=true err=EOF, got %v %v", done, err)
	}

	// 15. Ctrl+C -> ErrInterrupt
	done, _, err = ed.handleEvent(InputEvent{Key: KeyCtrlC}, buf)
	if !done || !errors.Is(err, ErrInterrupt) {
		t.Fatalf("Ctrl+C: want ErrInterrupt, got %v %v", done, err)
	}

	// 16. Enter submit with non-empty line
	buf.Set("my command")
	done, line, err := ed.handleEvent(InputEvent{Key: KeyEnter}, buf)
	if !done || line != "my command" || err != nil {
		t.Fatalf("Enter submit: got done=%v line=%q err=%v", done, line, err)
	}

	// 17. Bracketed Paste
	buf.Clear()
	ed.handleEvent(InputEvent{Key: KeyPasteStart, Paste: "pasted content"}, buf)
	if buf.String() != "pasted content" {
		t.Fatalf("Bracketed paste: got %q", buf.String())
	}

	// 18. Tab completion: single candidate
	buf.Set("bet")
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if buf.String() != "beta" {
		t.Fatalf("Tab completion single: got %q", buf.String())
	}

	// 19. Tab completion: multiple candidates & cycling
	buf.Set("alp")
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if !ed.completing {
		t.Fatal("expected ed.completing to be true")
	}
	// Cycle with Tab
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	// Cycle back with BackTab (Shift+Tab)
	ed.handleEvent(InputEvent{Key: KeyBackTab}, buf)
	// Cancel completion with rune
	ed.handleEvent(InputEvent{Key: KeyRune, Rune: 'x'}, buf)
	if ed.completing {
		t.Fatal("expected ed.completing to be false after typing")
	}

	// 20. Clear Screen (Ctrl+L)
	ed.handleEvent(InputEvent{Key: KeyCtrlL}, buf)

	// 21. History navigation (Up/Down)
	ed.handleEvent(InputEvent{Key: KeyArrowUp}, buf)
	ed.handleEvent(InputEvent{Key: KeyArrowDown}, buf)
}

// ── Multi-Row Line Wrapping Coverage ────────────────────────────────────────

func TestRenderer_MultiRowWrapping(t *testing.T) {
	var out bytes.Buffer
	rend := NewRenderer(&out, "prompt> ")
	rend.SetColumns(20)

	if rend.Columns() != 20 {
		t.Fatalf("want cols=20, got %d", rend.Columns())
	}
	rend.SetPrompt("new_prompt> ")
	if rend.Prompt() != "new_prompt> " {
		t.Fatalf("want 'new_prompt> ', got %q", rend.Prompt())
	}

	// Text longer than 20 columns to force multi-row wrapping
	buf := NewLineBuffer()
	buf.Set("This is a very long line that will definitely wrap across multiple rows in a narrow terminal.")
	rend.Redraw(buf)
	if rend.lastRows <= 1 {
		t.Fatalf("expected lastRows > 1 for wrapped line, got %d", rend.lastRows)
	}

	// Redraw with much shorter text to test line shrinking and clearing below
	buf.Set("short")
	rend.Redraw(buf)
	if rend.lastRows != 1 {
		t.Fatalf("expected lastRows = 1 after shrink, got %d", rend.lastRows)
	}
}

// ── KeyMap and UndoStack Coverage ────────────────────────────────────────────

func TestKeyMap_Unbind(t *testing.T) {
	km := make(KeyMap)
	km.Bind(KeyCtrlX, func(e *Editor, buf *LineBuffer) (bool, string, error) {
		return true, "done", nil
	})
	if km[KeyCtrlX] == nil {
		t.Fatal("binding not found")
	}
	km.Unbind(KeyCtrlX)
	if km[KeyCtrlX] != nil {
		t.Fatal("binding was not unbound")
	}
}

func TestUndoStack_Reset(t *testing.T) {
	u := NewUndoStack(10)
	b := NewLineBuffer()
	b.Set("text")
	u.Save(b)
	u.Reset()
	if u.Undo(b) {
		t.Fatal("undo should return false after reset")
	}
}

// ── SearchMode Additional Coverage ──────────────────────────────────────────

func TestSearchMode_AcceptAndCancel(t *testing.T) {
	h := NewHistory(10)
	h.Push("my test command")
	sm := newSearchMode(h)
	buf := NewLineBuffer()
	buf.Set("hello")

	sm.Start(buf)
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 't'}, buf)
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 'e'}, buf)
	sm.HandleKey(InputEvent{Key: KeyRune, Rune: 's'}, buf)

	promptStr := sm.Prompt()
	if !strings.Contains(promptStr, "reverse-i-search") {
		t.Fatalf("unexpected prompt: %q", promptStr)
	}

	// Accept via Enter
	handled, done, line, err := sm.HandleKey(InputEvent{Key: KeyEnter}, buf)
	if !handled || !done || line != "my test command" || err != nil {
		t.Fatalf("want accepted match, got %v %v %q %v", handled, done, line, err)
	}

	// Test Cancel via Ctrl+C
	sm.Start(buf)
	handled, done, _, err = sm.HandleKey(InputEvent{Key: KeyCtrlC}, buf)
	if !handled || !done || !errors.Is(err, ErrInterrupt) {
		t.Fatalf("want ErrInterrupt on Ctrl+C, got %v %v %v", handled, done, err)
	}
}

func TestEditor_TabCompletionCycling(t *testing.T) {
	commands := []string{"help", "exit", "quit", "history", "clear", "password", "echo", "status"}
	cfg := Config{
		Completer: PrefixCompleter(commands...),
	}
	ed := &Editor{
		cfg:       cfg,
		renderer:  NewRenderer(&bytes.Buffer{}, "> "),
		undoStack: NewUndoStack(50),
		history:   NewHistory(50),
	}

	buf := NewLineBuffer()
	buf.Set("hello ")
	buf.SetPos(6)

	// 1st Tab: enter completion mode, grid shown, buffer unchanged
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if !ed.completing {
		t.Fatal("expected completing=true after first Tab")
	}
	if buf.String() != "hello " {
		t.Fatalf("expected buffer to remain 'hello ', got %q", buf.String())
	}

	// 2nd Tab: cycle to candidate 0 ("help")
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if buf.String() != "hello help" {
		t.Fatalf("expected 'hello help', got %q", buf.String())
	}

	// 3rd Tab: cycle to candidate 1 ("exit") - must NOT concatenate!
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if buf.String() != "hello exit" {
		t.Fatalf("expected 'hello exit', got %q", buf.String())
	}

	// 4th Tab: cycle to candidate 2 ("quit")
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if buf.String() != "hello quit" {
		t.Fatalf("expected 'hello quit', got %q", buf.String())
	}

	// BackTab (Shift+Tab): cycle back to candidate 1 ("exit")
	ed.handleEvent(InputEvent{Key: KeyBackTab}, buf)
	if buf.String() != "hello exit" {
		t.Fatalf("expected 'hello exit' on Shift+Tab, got %q", buf.String())
	}

	// Esc: cancel completion, restores original buffer
	ed.handleEvent(InputEvent{Key: KeyEsc}, buf)
	if ed.completing {
		t.Fatal("expected completing=false after Esc")
	}
	if buf.String() != "hello " {
		t.Fatalf("expected 'hello ' restored on Esc, got %q", buf.String())
	}

	// Test single candidate auto-completion
	buf.Set("hello q")
	buf.SetPos(7)
	ed.handleEvent(InputEvent{Key: KeyTab}, buf)
	if buf.String() != "hello quit" {
		t.Fatalf("expected 'hello quit' on single match, got %q", buf.String())
	}
	if ed.completing {
		t.Fatal("expected completing=false for single match")
	}
}
