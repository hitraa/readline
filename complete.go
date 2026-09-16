package readline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Completion represents a candidate completion item.
type Completion struct {
	Value       string // replacement text to insert
	Display     string // display label (if empty, defaults to Value)
	Description string // optional description / help text
}

// Completer computes completions for the given input line and cursor position.
// Returns candidate completions, the length of the prefix to be replaced, and an optional error.
type Completer interface {
	Complete(ctx context.Context, line string, pos int) (completions []Completion, prefixLen int, err error)
}

// CompleterFunc adapts a function into a Completer.
type CompleterFunc func(ctx context.Context, line string, pos int) ([]Completion, int, error)

// Complete executes the underlying function.
func (f CompleterFunc) Complete(ctx context.Context, line string, pos int) ([]Completion, int, error) {
	return f(ctx, line, pos)
}

// PrefixCompleter creates a Completer from a static list of candidate strings.
func PrefixCompleter(words ...string) Completer {
	return CompleterFunc(func(_ context.Context, line string, pos int) ([]Completion, int, error) {
		word, wordStart, _ := FindWordAtCursor(line, pos)
		prefixLen := pos - wordStart
		var matches []Completion
		for _, w := range words {
			if strings.HasPrefix(w, word) {
				matches = append(matches, Completion{Value: w, Display: w})
			}
		}
		return matches, prefixLen, nil
	})
}

// PathCompleter completes local filesystem paths based on the word before cursor.
func PathCompleter() Completer {
	return CompleterFunc(func(_ context.Context, line string, pos int) ([]Completion, int, error) {
		word, wordStart, _ := FindWordAtCursor(line, pos)
		prefixLen := pos - wordStart

		dir, file := filepath.Split(word)
		lookupDir := dir
		if lookupDir == "" {
			lookupDir = "."
		}
		entries, err := os.ReadDir(lookupDir)
		if err != nil {
			return nil, 0, nil
		}
		var matches []Completion
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, file) {
				val := filepath.Join(dir, name)
				if e.IsDir() {
					val += "/"
				}
				matches = append(matches, Completion{
					Value:   val,
					Display: name,
				})
			}
		}
		return matches, prefixLen, nil
	})
}

// FormatCompletionGrid formats candidate completions into columns fitting within maxCols.
func FormatCompletionGrid(items []Completion, maxCols int) []string {
	if len(items) == 0 {
		return nil
	}
	if maxCols <= 0 {
		maxCols = 80
	}

	hasDesc := false
	maxDisplayW := 0
	for _, it := range items {
		disp := it.Display
		if disp == "" {
			disp = it.Value
		}
		w := stringDisplayWidth(disp)
		if w > maxDisplayW {
			maxDisplayW = w
		}
		if it.Description != "" {
			hasDesc = true
		}
	}

	// If single column or descriptions are present, display 1 item per line with description
	colWidth := maxDisplayW + 2
	numCols := maxCols / colWidth
	if numCols <= 1 || hasDesc {
		var lines []string
		for _, it := range items {
			disp := it.Display
			if disp == "" {
				disp = it.Value
			}
			if it.Description != "" {
				lines = append(lines, fmt.Sprintf("%-*s  (%s)", maxDisplayW, disp, it.Description))
			} else {
				lines = append(lines, disp)
			}
		}
		return lines
	}

	numRows := (len(items) + numCols - 1) / numCols
	var lines []string
	for r := 0; r < numRows; r++ {
		var line strings.Builder
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx < len(items) {
				disp := items[idx].Display
				if disp == "" {
					disp = items[idx].Value
				}
				if c == numCols-1 {
					line.WriteString(disp)
				} else {
					dispW := stringDisplayWidth(disp)
					padding := colWidth - dispW
					if padding < 1 {
						padding = 1
					}
					line.WriteString(disp)
					line.WriteString(strings.Repeat(" ", padding))
				}
			}
		}
		lines = append(lines, line.String())
	}
	return lines
}
