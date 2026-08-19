// Package readline provides a production-grade terminal line editor for
// interactive CLI applications.  It is Linux and macOS native (no CGO, no
// external dependencies) with UTF-8 support, history, and a rich set of
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
//	    if err == io.EOF { break }          // Ctrl+D
//	    if err == readline.ErrInterrupt { continue } // Ctrl+C
//	    fmt.Println("got:", line)
//	}
package readline

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// ErrInterrupt is returned by ReadLine when the user presses Ctrl+C.
var ErrInterrupt = errors.New("readline: interrupted")

// ErrNotSupported is returned on platforms where raw terminal mode is not implemented.
var ErrNotSupported = errors.New("readline: raw terminal mode is not supported on this platform")

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
	// Ctrl+\ sends SIGQUIT.  When false those keystrokes have no effect.
	EnableSignals bool
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
}

// New initialises a new Editor and puts stdin into raw mode.
// Call Close() when done to restore terminal state.
func New(cfg Config) (*Editor, error) {
	if cfg.Prompt == "" {
		cfg.Prompt = "> "
	}
	if cfg.MaxHistory == 0 {
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

	hist := NewHistory(cfg.MaxHistory)
	if cfg.HistoryFile != "" {
		_ = hist.LoadFile(cfg.HistoryFile) // non-fatal
	}

	e := &Editor{
		cfg:      cfg,
		history:  hist,
		renderer: NewRenderer(cfg.Stdout, cfg.Prompt),
		reader:   newInputReader(cfg.Stdin),
		term:     term,
	}

	// Handle SIGTERM / os.Interrupt so the terminal is always restored.
	go e.watchOSSignals()

	return e, nil
}

// watchOSSignals restores the terminal on SIGTERM/SIGINT so the shell is left
// in a usable state even when the process is killed.
func (e *Editor) watchOSSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	_ = e.term.leaveRaw()
	e.Close()
	os.Exit(1)
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
// deregister. Can be called at most once; subsequent calls are ignored.
func (e *Editor) WatchResize(fn func(cols, rows int)) {
	if e.stopResize != nil {
		return
	}
	e.stopResize = e.term.WatchResize(fn)
}

// ReadLine reads one line of input from the terminal.
//
// An optional prompt string may be supplied to override the editor's current
// prompt for this call and all subsequent calls (equivalent to calling
// SetPrompt before ReadLine).  If omitted, the previously configured prompt
// is used unchanged.
//
// Return values:
//   - (line, nil)          — user submitted a line (Enter)
//   - ("", ErrInterrupt)   — user pressed Ctrl+C
//   - ("", io.EOF)         — user pressed Ctrl+D on an empty line
//   - ("", err)            — underlying I/O error
func (e *Editor) ReadLine(prompt ...string) (string, error) {
	// Optional per-call prompt override.
	if len(prompt) > 0 && prompt[0] != "" {
		e.SetPrompt(prompt[0])
	}

	// Ensure we're in raw mode (idempotent if already raw).
	if err := e.term.enterRaw(); err != nil {
		return "", err
	}

	buf := NewLineBuffer()
	e.history.Reset()

	// Print the prompt (no trailing newline).
	e.renderer.SetPrompt(e.cfg.Prompt)
	fmt_fprint(e.renderer, e.cfg.Prompt)

	for {
		evt, err := e.reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				e.renderer.NewLine()
			}
			return "", err
		}

		if handled, line, rerr := e.handleEvent(evt, buf); handled {
			return line, rerr
		}
	}
}

// handleEvent processes a single InputEvent, mutating buf and the renderer.
// Returns (true, line, err) when the line is complete (Enter/Ctrl+C/Ctrl+D).
// Returns (false, "", nil) when more input is needed.
func (e *Editor) handleEvent(evt InputEvent, buf *LineBuffer) (done bool, line string, err error) {
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
		// Ctrl+D on a non-empty line deletes the character under the cursor.
		if buf.Delete() {
			e.renderer.Redraw(buf)
		}

	// ── Deletion ──────────────────────────────────────────────────────────────
	case KeyBackspace, KeyCtrlH:
		if buf.Backspace() {
			e.renderer.Redraw(buf)
		}

	case KeyDelete:
		if buf.Delete() {
			e.renderer.Redraw(buf)
		}

	case KeyCtrlW:
		if buf.DeleteWordBefore() {
			e.renderer.Redraw(buf)
		}

	case KeyCtrlK:
		if buf.KillToEnd() {
			e.renderer.Redraw(buf)
		}

	case KeyCtrlU:
		if buf.KillToStart() {
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

	case KeyPageUp:
		bufContent := buf.String()

		// ── Stale-prefix guard ────────────────────────────────────────────────
		// If a search is already in progress (pending != "") but the buffer no
		// longer starts with the locked prefix, the user has edited the input
		// (erased characters or typed a new prefix).  Discard the old session so
		// the next search starts fresh from the newest history entry.
		if e.history.pending != "" && !strings.HasPrefix(bufContent, e.history.pending) {
			e.history.ResetSearch()
		}

		// Determine prefix: reuse pending when mid-search, otherwise snapshot
		// the current buffer and lock it in for this session.
		prefix := e.history.pending
		if prefix == "" {
			prefix = bufContent
			e.history.SetPending(prefix)
		}

		if prefix == "" {
			// Empty buffer: jump to the oldest history entry (original behaviour).
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
			// Non-empty prefix: walk backwards to the next matching entry.
			if entry, ok := e.history.SearchUp(prefix); ok {
				buf.Set(entry)
				e.renderer.Redraw(buf)
			}
		}

	case KeyPageDown:
		bufContent := buf.String()

		// Same stale-prefix guard: if the buffer diverged from pending, abandon
		// the search so PageDown doesn't continue with the wrong prefix.
		if e.history.pending != "" && !strings.HasPrefix(bufContent, e.history.pending) {
			e.history.ResetSearch()
		}

		prefix := e.history.pending
		if prefix == "" {
			// No active search: clear the buffer and return to live input.
			e.history.Reset()
			buf.Set("")
			e.renderer.Redraw(buf)
		} else {
			// Walk forward; SearchDown restores pending text when exhausted.
			entry, _ := e.history.SearchDown(prefix)
			buf.Set(entry)
			e.renderer.Redraw(buf)
		}

	// ── Screen ────────────────────────────────────────────────────────────────
	case KeyCtrlL:
		e.renderer.ClearScreen(buf)

	// ── Printable input ───────────────────────────────────────────────────────
	case KeyRune:
		buf.Insert(evt.Rune)
		e.renderer.Redraw(buf)

		// Tab / other keys: extensible — no-op for now.
	}

	return false, "", nil
}

// Close restores the terminal to its original state and flushes pending history.
// It is safe to call Close multiple times.
func (e *Editor) Close() error {
	if e.stopResize != nil {
		e.stopResize()
		e.stopResize = nil
	}
	return e.term.Close()
}

// fmt_fprint is a tiny helper to avoid importing "fmt" just for Fprint.
func fmt_fprint(w interface{ Write([]byte) (int, error) }, s string) {
	_, _ = w.Write([]byte(s))
}
