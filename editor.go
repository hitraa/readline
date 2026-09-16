// Package readline provides a production-grade terminal line editor for
// interactive CLI applications. It is Linux, macOS, and Windows native (no CGO, no
// external dependencies) with UTF-8 support, history, autocompletion, and a rich set of
// Emacs-style key bindings.
//
// # Quick start
//
//	ed, err := readline.New(readline.Config{Prompt: "myapp> "})
//	if err != nil { log.Fatal(err) }
//	defer ed.Close()
//
//	for {
//	    line, err := ed.ReadLine()
//	    if err == io.EOF { break }                   // Ctrl+D
//	    if errors.Is(err, readline.ErrInterrupt) { continue } // Ctrl+C
//	    fmt.Println("got:", line)
//	}
package readline

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

// ErrInterrupt is returned by ReadLine when the user presses Ctrl+C.
var ErrInterrupt = errors.New("readline: interrupted")

// ErrNotSupported is returned on platforms where raw terminal mode is not implemented.
var ErrNotSupported = errors.New("readline: raw terminal mode is not supported on this platform")

// ErrClosed is returned when an operation is attempted on a closed editor.
var ErrClosed = errors.New("readline: editor is closed")

// Config holds the configuration for a new Editor.
type Config struct {
	// Prompt is displayed before each line (may contain ANSI colour codes).
	Prompt string

	// HistoryFile is an optional file path for cross-session persistence.
	// History is saved after each line when this is set.
	HistoryFile string

	// MaxHistory caps the number of in-memory entries (default 500).
	MaxHistory int

	// Stdin / Stdout override the default os.Stdin / os.Stdout.
	Stdin  *os.File
	Stdout *os.File

	// EnableSignals controls whether ISIG is set in the raw termios.
	// When true (the default) Ctrl+Z suspends the process via SIGTSTP and
	// Ctrl+\ sends SIGQUIT. When false those keystrokes have no effect.
	EnableSignals bool

	// Completer provides dynamic autocompletion on Tab.
	Completer Completer

	// KeyMap allows overriding or extending key bindings.
	KeyMap KeyMap

	// Mask character for password/secret input (default 0 = silent, e.g. '*' for stars).
	Mask rune
}

// Editor is the public terminal line editor.
type Editor struct {
	cfg        Config
	history    *History
	renderer   *Renderer
	reader     *inputReader
	term       *terminal
	sessionNew []string // entries added during this session (for file append)
	stopResize func()   // cancels SIGWINCH watcher if started
	sigCloseCh chan struct{}
	closeOnce  sync.Once
	killRing   *KillRing
	undoStack  *UndoStack
	searchMode *SearchMode
	keymap     KeyMap

	// Non-TTY buffered reader fallback
	nonTTYReader *bufio.Reader

	// Active completion state
	completing     bool
	compCandidates []Completion
	compIdx        int
	compPrefixLen  int
	compOrigWord   string
	compOrigPos    int
	lastWasYank    bool
}

// New initialises a new Editor and prepares terminal input.
// Call Close() when done to restore terminal state.
func New(cfg Config) (*Editor, error) {
	if cfg.Prompt == "" {
		cfg.Prompt = "> "
	}
	if cfg.MaxHistory <= 0 {
		cfg.MaxHistory = 500
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}

	term, err := newTerminal(cfg.Stdin, cfg.EnableSignals)
	if err != nil {
		return nil, err
	}

	cols, _, _ := term.GetSize()
	renderer := NewRenderer(cfg.Stdout, cfg.Prompt)
	renderer.SetColumns(cols)

	hist := NewHistory(cfg.MaxHistory)
	if cfg.HistoryFile != "" {
		_ = hist.LoadFile(cfg.HistoryFile) // non-fatal
	}

	km := make(KeyMap)
	if cfg.KeyMap != nil {
		km = cfg.KeyMap.Clone()
	}

	e := &Editor{
		cfg:        cfg,
		history:    hist,
		renderer:   renderer,
		reader:     newInputReader(cfg.Stdin),
		term:       term,
		sigCloseCh: make(chan struct{}),
		killRing:   NewKillRing(60),
		undoStack:  NewUndoStack(100),
		searchMode: newSearchMode(hist),
		keymap:     km,
	}

	if !term.IsTerminal() {
		e.nonTTYReader = bufio.NewReader(cfg.Stdin)
	} else {
		// Hook resize to renderer
		e.WatchResize(func(c, _ int) {
			e.renderer.SetColumns(c)
		})
		// Handle SIGTERM / SIGHUP to restore terminal state
		go e.watchOSSignals()
	}

	return e, nil
}

// watchOSSignals restores the terminal on SIGTERM/SIGHUP so the shell is left
// in a usable state even if the process is killed.
func (e *Editor) watchOSSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(ch)

	select {
	case <-ch:
		_ = e.term.leaveRaw()
		_ = e.Close()
		os.Exit(1)
	case <-e.sigCloseCh:
		return
	}
}

// SetPrompt replaces the prompt for subsequent ReadLine calls.
func (e *Editor) SetPrompt(p string) {
	e.cfg.Prompt = p
	e.renderer.SetPrompt(p)
}

// History returns a copy of all in-session history entries.
func (e *Editor) History() []string { return e.history.Entries() }

// WatchResize registers a callback for terminal resize events (SIGWINCH).
// The callback receives (cols, rows). Call the returned stop function to
// deregister.
func (e *Editor) WatchResize(fn func(cols, rows int)) {
	if e.stopResize != nil {
		return
	}
	e.stopResize = e.term.WatchResize(func(cols, rows int) {
		e.renderer.SetColumns(cols)
		if fn != nil {
			fn(cols, rows)
		}
	})
}

// ReadLine reads one line of input from the terminal.
//
// An optional prompt string may be supplied to override the editor's current
// prompt for this call and all subsequent calls (equivalent to calling
// SetPrompt before ReadLine). If omitted, the previously configured prompt
// is used unchanged.
//
// Return values:
//   - (line, nil)          — user submitted a line (Enter)
//   - ("", ErrInterrupt)   — user pressed Ctrl+C
//   - ("", io.EOF)         — user pressed Ctrl+D on an empty line or EOF reached
//   - ("", err)            — underlying I/O error
func (e *Editor) ReadLine(prompt ...string) (string, error) {
	return e.ReadLineContext(context.Background(), prompt...)
}

// ReadLineContext reads one line of input, respecting context cancellation.
func (e *Editor) ReadLineContext(ctx context.Context, prompt ...string) (string, error) {
	// Optional per-call prompt override.
	if len(prompt) > 0 && prompt[0] != "" {
		e.SetPrompt(prompt[0])
	}

	// Non-interactive fallback when stdin is not a terminal
	if !e.term.IsTerminal() {
		return e.readLineNonTTY(ctx)
	}

	// Ensure we're in raw mode (idempotent if already raw).
	if err := e.term.enterRaw(); err != nil {
		return "", err
	}
	// Enable bracketed paste mode in terminal
	fmt.Fprint(e.renderer.out, "\033[?2004h")

	// Panic safety: always restore terminal mode even on panics
	defer func() {
		fmt.Fprint(e.renderer.out, "\033[?2004l")
		_ = e.term.leaveRaw()
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	buf := NewLineBuffer()
	e.history.Reset()
	e.undoStack.Reset()
	e.completing = false

	// Update columns
	if cols, _, err := e.term.GetSize(); err == nil && cols > 0 {
		e.renderer.SetColumns(cols)
	}

	// Print prompt
	e.renderer.SetPrompt(e.cfg.Prompt)
	fmt_fprint(e.renderer, e.cfg.Prompt)

	type eventResult struct {
		evt InputEvent
		err error
	}
	eventCh := make(chan eventResult, 1)

	for {
		go func() {
			evt, err := e.reader.ReadEvent()
			eventCh <- eventResult{evt: evt, err: err}
		}()

		select {
		case <-ctx.Done():
			e.renderer.NewLine()
			return "", ctx.Err()

		case res := <-eventCh:
			if res.err != nil {
				if res.err == io.EOF {
					e.renderer.NewLine()
				}
				return "", res.err
			}

			// Handle reverse incremental search if active
			if e.searchMode.Active() {
				handled, done, sline, serr := e.searchMode.HandleKey(res.evt, buf)
				if handled {
					if done {
						e.renderer.NewLine()
						return sline, serr
					}
					// Redraw search prompt
					fmt.Fprintf(e.renderer.out, "\r\033[K%s", e.searchMode.Prompt())
					continue
				}
				// If not handled by search, search mode accepted match; redraw normal prompt
				e.renderer.SetPrompt(e.cfg.Prompt)
				e.renderer.Redraw(buf)
			}

			if handled, line, rerr := e.handleEvent(res.evt, buf); handled {
				return line, rerr
			}
		}
	}
}

// readLineNonTTY reads a line without raw terminal manipulation for piped or redirected input,
// respecting context cancellation.
func (e *Editor) readLineNonTTY(ctx context.Context) (string, error) {
	if e.nonTTYReader == nil {
		e.nonTTYReader = bufio.NewReader(e.cfg.Stdin)
	}
	type res struct {
		line string
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		l, err := e.nonTTYReader.ReadString('\n')
		ch <- res{line: l, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		if r.err != nil && len(r.line) == 0 {
			return "", r.err
		}
		return strings.TrimRight(r.line, "\r\n"), nil
	}
}

// ReadPassword reads a secret line without echoing characters to the terminal.
// It does not record input into history.
func (e *Editor) ReadPassword(prompt ...string) (string, error) {
	if len(prompt) > 0 && prompt[0] != "" {
		e.SetPrompt(prompt[0])
	}

	if !e.term.IsTerminal() {
		return e.readLineNonTTY(context.Background())
	}

	if err := e.term.enterRaw(); err != nil {
		return "", err
	}
	defer func() {
		_ = e.term.leaveRaw()
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	fmt_fprint(e.renderer, e.cfg.Prompt)

	var runes []rune
	for {
		evt, err := e.reader.ReadEvent()
		if err != nil {
			e.renderer.NewLine()
			return "", err
		}

		switch evt.Key {
		case KeyEnter:
			e.renderer.NewLine()
			return string(runes), nil
		case KeyCtrlC:
			e.renderer.NewLine()
			return "", ErrInterrupt
		case KeyCtrlD:
			if len(runes) == 0 {
				e.renderer.NewLine()
				return "", io.EOF
			}
		case KeyBackspace, KeyCtrlH:
			if len(runes) > 0 {
				runes = runes[:len(runes)-1]
				if e.cfg.Mask != 0 {
					fmt.Fprint(e.renderer.out, "\b \b")
				}
			}
		case KeyCtrlU:
			if e.cfg.Mask != 0 {
				for range runes {
					fmt.Fprint(e.renderer.out, "\b \b")
				}
			}
			runes = runes[:0]
		case KeyRune:
			runes = append(runes, evt.Rune)
			if e.cfg.Mask != 0 {
				fmt.Fprintf(e.renderer.out, "%c", e.cfg.Mask)
			}
		}
	}
}

// handleEvent processes a single InputEvent, mutating buf and the renderer.
func (e *Editor) handleEvent(evt InputEvent, buf *LineBuffer) (done bool, line string, err error) {
	// Custom keymap dispatch
	if action, ok := e.keymap[evt.Key]; ok {
		e.undoStack.Save(buf)
		return action(e, buf)
	}

	// Completion handling: cancel completion state if another key is pressed
	if e.completing && evt.Key != KeyTab && evt.Key != KeyBackTab {
		e.completing = false
		e.renderer.Redraw(buf)
	}

	isYank := (evt.Key == KeyCtrlY || evt.Key == KeyAltY)
	if !isYank {
		e.lastWasYank = false
	}

	switch evt.Key {

	// ── Submit ────────────────────────────────────────────────────────────────
	case KeyEnter:
		e.renderer.NewLine()
		result := strings.TrimSpace(buf.String())
		if result != "" {
			e.history.Push(result)
			e.sessionNew = append(e.sessionNew, result)
			if e.cfg.HistoryFile != "" {
				_ = e.history.SaveFile(e.cfg.HistoryFile, []string{result})
			}
		}
		return true, result, nil

	// ── Interrupt / EOF ────────────────────────────────────────────────────────
	case KeyCtrlC:
		e.renderer.NewLine()
		return true, "", ErrInterrupt

	case KeyCtrlD:
		if buf.Len() == 0 {
			e.renderer.NewLine()
			return true, "", io.EOF
		}
		e.undoStack.Save(buf)
		if buf.Delete() {
			e.renderer.Redraw(buf)
		}

	// ── Undo / Redo ───────────────────────────────────────────────────────────
	case KeyCtrlUnderscore:
		if e.undoStack.Undo(buf) {
			e.renderer.Redraw(buf)
		}

	// ── Deletion ──────────────────────────────────────────────────────────────
	case KeyBackspace, KeyCtrlH:
		e.undoStack.Save(buf)
		if buf.Backspace() {
			e.renderer.Redraw(buf)
		}

	case KeyDelete:
		e.undoStack.Save(buf)
		if buf.Delete() {
			e.renderer.Redraw(buf)
		}

	case KeyCtrlW, KeyAltBackspace:
		e.undoStack.Save(buf)
		if deleted, ok := buf.DeleteWordBeforeText(); ok {
			e.killRing.Push(deleted)
			e.renderer.Redraw(buf)
		}

	case KeyAltD:
		e.undoStack.Save(buf)
		if deleted, ok := buf.DeleteWordAfterText(); ok {
			e.killRing.Push(deleted)
			e.renderer.Redraw(buf)
		}

	case KeyCtrlK:
		e.undoStack.Save(buf)
		if deleted, ok := buf.KillToEndText(); ok {
			e.killRing.Push(deleted)
			e.renderer.Redraw(buf)
		}

	case KeyCtrlU:
		e.undoStack.Save(buf)
		if deleted, ok := buf.KillToStartText(); ok {
			e.killRing.Push(deleted)
			e.renderer.Redraw(buf)
		}

	// ── Kill ring & Yank ──────────────────────────────────────────────────────
	case KeyCtrlY:
		top := e.killRing.Top()
		if top != "" {
			e.undoStack.Save(buf)
			buf.InsertString(top)
			e.renderer.Redraw(buf)
			e.lastWasYank = true
		}

	case KeyAltY:
		if e.lastWasYank && e.killRing.Len() > 1 {
			e.undoStack.Save(buf)
			old := e.killRing.Top()
			for range old {
				buf.Backspace()
			}
			next := e.killRing.YankPop()
			buf.InsertString(next)
			e.renderer.Redraw(buf)
		}

	// ── Transpose ─────────────────────────────────────────────────────────────
	case KeyCtrlT:
		e.undoStack.Save(buf)
		if buf.Transpose() {
			e.renderer.Redraw(buf)
		}

	// ── Movement ──────────────────────────────────────────────────────────────
	case KeyArrowLeft, KeyCtrlB:
		if buf.MoveLeft() {
			e.renderer.Redraw(buf)
		}

	case KeyArrowRight, KeyCtrlF:
		if buf.MoveRight() {
			e.renderer.Redraw(buf)
		}

	case KeyAltB, KeyCtrlLeft:
		if buf.MoveWordLeft() {
			e.renderer.Redraw(buf)
		}

	case KeyAltF, KeyCtrlRight:
		if buf.MoveWordRight() {
			e.renderer.Redraw(buf)
		}

	case KeyHome, KeyCtrlA:
		buf.MoveHome()
		e.renderer.Redraw(buf)

	case KeyEnd, KeyCtrlE:
		buf.MoveEnd()
		e.renderer.Redraw(buf)

	// ── History ───────────────────────────────────────────────────────────────
	case KeyArrowUp, KeyCtrlP:
		e.history.SetPending(buf.String())
		if entry, ok := e.history.Up(); ok {
			buf.Set(entry)
			e.renderer.Redraw(buf)
		}

	case KeyArrowDown, KeyCtrlN:
		if entry, ok := e.history.Down(); ok {
			buf.Set(entry)
		} else {
			buf.Set(e.history.pending)
		}
		e.renderer.Redraw(buf)

	case KeyCtrlR:
		e.searchMode.Start(buf)
		fmt.Fprintf(e.renderer.out, "\r\033[K%s", e.searchMode.Prompt())

	case KeyPageUp:
		bufContent := buf.String()
		if e.history.pending != "" && !strings.HasPrefix(bufContent, e.history.pending) {
			e.history.ResetSearch()
		}
		prefix := e.history.pending
		if prefix == "" {
			prefix = bufContent
			e.history.SetPending(prefix)
		}
		if prefix == "" {
			for {
				if _, ok := e.history.Up(); !ok {
					break
				}
			}
			if e.history.Len() > 0 {
				buf.Set(e.history.Entries()[0])
				e.renderer.Redraw(buf)
			}
		} else {
			if entry, ok := e.history.SearchUp(prefix); ok {
				buf.Set(entry)
				e.renderer.Redraw(buf)
			}
		}

	case KeyPageDown:
		bufContent := buf.String()
		if e.history.pending != "" && !strings.HasPrefix(bufContent, e.history.pending) {
			e.history.ResetSearch()
		}
		prefix := e.history.pending
		if prefix == "" {
			e.history.Reset()
			buf.Set("")
			e.renderer.Redraw(buf)
		} else {
			entry, _ := e.history.SearchDown(prefix)
			buf.Set(entry)
			e.renderer.Redraw(buf)
		}

	// ── Tab Completion ────────────────────────────────────────────────────────
	case KeyTab:
		if e.cfg.Completer == nil {
			break
		}
		if !e.completing {
			comps, pLen, err := e.cfg.Completer.Complete(context.Background(), buf.String(), buf.Pos())
			if err != nil || len(comps) == 0 {
				break
			}
			if len(comps) == 1 {
				e.undoStack.Save(buf)
				for i := 0; i < pLen; i++ {
					buf.Backspace()
				}
				buf.InsertString(comps[0].Value)
				e.renderer.Redraw(buf)
				break
			}
			// Multiple candidates: enter completion mode and display candidate grid
			e.completing = true
			e.compCandidates = comps
			e.compIdx = 0
			e.compPrefixLen = pLen
			e.compOrigPos = buf.Pos()

			// Render completion list below current prompt
			grid := FormatCompletionGrid(comps, e.renderer.Columns())
			if len(grid) > 0 {
				fmt.Fprint(e.renderer.out, "\r\n")
				for _, line := range grid {
					fmt.Fprintf(e.renderer.out, "%s\r\n", line)
				}
			}
			e.renderer.lastRows = 1
			e.renderer.Redraw(buf)
		} else {
			// Cycle to next candidate
			e.compIdx = (e.compIdx + 1) % len(e.compCandidates)
			cand := e.compCandidates[e.compIdx]
			buf.SetPos(e.compOrigPos)
			for i := 0; i < e.compPrefixLen; i++ {
				buf.Backspace()
			}
			buf.InsertString(cand.Value)
			e.renderer.Redraw(buf)
		}

	case KeyBackTab:
		if e.completing && len(e.compCandidates) > 0 {
			e.compIdx = (e.compIdx - 1 + len(e.compCandidates)) % len(e.compCandidates)
			cand := e.compCandidates[e.compIdx]
			buf.SetPos(e.compOrigPos)
			for i := 0; i < e.compPrefixLen; i++ {
				buf.Backspace()
			}
			buf.InsertString(cand.Value)
			e.renderer.Redraw(buf)
		}

	// ── Bracketed Paste ───────────────────────────────────────────────────────
	case KeyPasteStart:
		e.undoStack.Save(buf)
		buf.InsertString(evt.Paste)
		e.renderer.Redraw(buf)

	// ── Screen ────────────────────────────────────────────────────────────────
	case KeyCtrlL:
		e.renderer.ClearScreen(buf)

	// ── Printable input ───────────────────────────────────────────────────────
	case KeyRune:
		e.undoStack.Save(buf)
		buf.Insert(evt.Rune)
		e.renderer.Redraw(buf)
	}

	return false, "", nil
}

// Close restores the terminal to its original state and flushes pending history.
// It is safe to call Close multiple times.
func (e *Editor) Close() error {
	var err error
	e.closeOnce.Do(func() {
		close(e.sigCloseCh)
		if e.stopResize != nil {
			e.stopResize()
			e.stopResize = nil
		}
		err = e.term.Close()
	})
	return err
}

// fmt_fprint is a tiny helper to avoid importing "fmt" just for Fprint.
func fmt_fprint(w interface{ Write([]byte) (int, error) }, s string) {
	_, _ = w.Write([]byte(s))
}
