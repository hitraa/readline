package readline

import (
	"fmt"
	"io"
	"regexp"
)

var (
	ansiCSIEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
	ansiOSCEscape = regexp.MustCompile(`\x1b\].*?(\x07|\x1b\\)`)
)

// stripANSI removes ANSI CSI and OSC escape sequences from string s.
func stripANSI(s string) string {
	s = ansiCSIEscape.ReplaceAllString(s, "")
	s = ansiOSCEscape.ReplaceAllString(s, "")
	return s
}

// stringDisplayWidth computes the terminal column width of a string (ignoring ANSI escapes).
func stringDisplayWidth(s string) int {
	clean := stripANSI(s)
	w := 0
	for _, r := range clean {
		w += runeDisplayWidth(r)
	}
	return w
}

// Renderer handles all terminal output for the line editor.
// It writes ANSI escape sequences to an io.Writer and knows the prompt string
// so it can correctly position the cursor after redraws.
type Renderer struct {
	out      io.Writer
	prompt   string
	cols     int
	lastRows int
}

// NewRenderer creates a Renderer that writes to out using the given prompt.
func NewRenderer(out io.Writer, prompt string) *Renderer {
	return &Renderer{out: out, prompt: prompt, cols: 80, lastRows: 1}
}

// SetColumns updates the terminal width for wrapping calculations.
func (r *Renderer) SetColumns(cols int) {
	if cols > 0 {
		r.cols = cols
	}
}

// Columns returns the currently configured column width.
func (r *Renderer) Columns() int {
	if r.cols <= 0 {
		return 80
	}
	return r.cols
}

// SetPrompt replaces the current prompt string.
func (r *Renderer) SetPrompt(p string) { r.prompt = p }

// Prompt returns the current prompt string.
func (r *Renderer) Prompt() string { return r.prompt }

// PromptDisplayLen returns the visible column width of the prompt, stripping
// any embedded ANSI escape sequences.
func (r *Renderer) PromptDisplayLen() int {
	return stringDisplayWidth(r.prompt)
}

// Redraw erases the current terminal line and redraws prompt + buffer,
// then repositions the cursor to match buf.Pos().
func (r *Renderer) Redraw(buf *LineBuffer) {
	cols := r.Columns()
	promptLen := r.PromptDisplayLen()
	content := buf.String()
	totalW := buf.DisplayWidthTotal()
	cursorW := buf.DisplayWidth(buf.Pos())

	// Move cursor up if previous render occupied multiple rows
	if r.lastRows > 1 {
		fmt.Fprintf(r.out, "\033[%dA", r.lastRows-1)
	}

	// \r        — move to column 0
	// \033[K    — erase to end of line
	fmt.Fprintf(r.out, "\r%s\033[K%s", r.prompt, content)

	fullVisualLen := promptLen + totalW
	totalRows := fullVisualLen/cols + 1
	if fullVisualLen > 0 && fullVisualLen%cols == 0 {
		totalRows = fullVisualLen / cols
	}
	if totalRows < 1 {
		totalRows = 1
	}

	// If previously had more rows than current, clear remaining lines below
	if r.lastRows > totalRows {
		for i := totalRows; i < r.lastRows; i++ {
			fmt.Fprint(r.out, "\n\r\033[K")
		}
		fmt.Fprintf(r.out, "\033[%dA", r.lastRows-totalRows)
	}

	if totalRows <= 1 {
		// Single row: simple horizontal retreat
		retreat := totalW - cursorW
		if retreat > 0 {
			fmt.Fprintf(r.out, "\033[%dD", retreat)
		}
	} else {
		// Multi-row cursor positioning
		cursorVisualLen := promptLen + cursorW
		cursorRow := cursorVisualLen / cols
		cursorCol := cursorVisualLen % cols

		endRow := fullVisualLen / cols
		rowDiff := endRow - cursorRow
		if rowDiff > 0 {
			fmt.Fprintf(r.out, "\033[%dA", rowDiff)
		}
		fmt.Fprint(r.out, "\r")
		if cursorCol > 0 {
			fmt.Fprintf(r.out, "\033[%dC", cursorCol)
		}
	}

	r.lastRows = totalRows
}

// ClearScreen clears the entire terminal and redraws the current prompt+buffer.
func (r *Renderer) ClearScreen(buf *LineBuffer) {
	fmt.Fprint(r.out, "\033[2J\033[H")
	r.lastRows = 1
	r.Redraw(buf)
}

// NewLine writes a CR+LF, used after the user presses Enter.
func (r *Renderer) NewLine() {
	fmt.Fprint(r.out, "\r\n")
	r.lastRows = 1
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
