package readline

import (
	"fmt"
	"io"
)

// Renderer handles all terminal output for the line editor.
// It writes ANSI escape sequences to an io.Writer and knows the prompt string
// so it can correctly position the cursor after redraws.
type Renderer struct {
	out    io.Writer
	prompt string
}

// NewRenderer creates a Renderer that writes to out using the given prompt.
func NewRenderer(out io.Writer, prompt string) *Renderer {
	return &Renderer{out: out, prompt: prompt}
}

// SetPrompt replaces the current prompt string.
func (r *Renderer) SetPrompt(p string) { r.prompt = p }

// Prompt returns the current prompt string.
func (r *Renderer) Prompt() string { return r.prompt }

// PromptDisplayLen returns the visible column width of the prompt, stripping
// any embedded ANSI escape sequences.
func (r *Renderer) PromptDisplayLen() int {
	w, inEsc := 0, false
	for _, c := range r.prompt {
		if c == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		w++
	}
	return w
}

// Redraw erases the current terminal line and redraws prompt + buffer,
// then repositions the cursor to match buf.Pos().
func (r *Renderer) Redraw(buf *LineBuffer) {
	content := buf.String()
	totalW := buf.DisplayWidthTotal()
	cursorW := buf.DisplayWidth(buf.Pos())
	retreat := totalW - cursorW

	// \r        — move to column 0
	// \033[K    — erase to end of line
	fmt.Fprintf(r.out, "\r%s\033[K%s", r.prompt, content)
	if retreat > 0 {
		fmt.Fprintf(r.out, "\033[%dD", retreat)
	}
}

// ClearScreen clears the entire terminal and redraws the current prompt+buffer.
func (r *Renderer) ClearScreen(buf *LineBuffer) {
	fmt.Fprint(r.out, "\033[2J\033[H")
	r.Redraw(buf)
}

// NewLine writes a CR+LF, used after the user presses Enter.
func (r *Renderer) NewLine() {
	fmt.Fprint(r.out, "\r\n")
}

// PrintBanner writes a multi-line message followed by a CR+LF.  Use this to
// print banners or help text before the first prompt.
func (r *Renderer) PrintBanner(msg string) {
	fmt.Fprintf(r.out, "%s\r\n", msg)
}

// Write implements io.Writer so callers can use fmt.Fprintf(renderer, …).
func (r *Renderer) Write(p []byte) (int, error) {
	return r.out.Write(p)
}
