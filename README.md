# readline

A production-grade, zero-dependency, pure Go terminal line editor designed for building interactive command-line interfaces.

It is Linux, macOS, and Windows native, requiring **no CGO** and **no third-party dependencies**. It includes native UTF-8 (CJK and zero-width character metrics) support, multi-line line wrapping, full history management with reverse incremental search, dynamic autocompletion, undo/redo, kill ring, password input, and Emacs-style terminal key bindings.

---

## Features

- 📦 **Zero Dependencies**: Pure Go implementation using OS-specific termios and Win32 console mode system calls.
- ⚡ **No CGO**: Light binaries, fast builds, and seamless cross-compilation.
- 🪟 **Cross-Platform**: Full native support for Linux, macOS, and Windows.
- 🔀 **Piped & Non-TTY Fallback**: Automatically falls back to standard buffered input when stdin/stdout are not terminals.
- 🔍 **Reverse Incremental Search (Ctrl+R)**: Interactive history search with dynamic matching and redisplay (`Ctrl+R` for older, `Ctrl+S` for newer, `Enter` to accept, `Esc` to cancel).
- 💡 **Dynamic Autocompletion**: Extensible `Completer` interface with Tab completion, candidate cycling, and terminal-width-aware column grid display. Built-in `PrefixCompleter` and `PathCompleter`.
- 📋 **Bracketed Paste**: Atomic insertion of pasted text without triggering premature execution.
- 🔑 **Password & Secret Mode**: `ReadPassword` for secret input without terminal echo.
- ⏱️ **Context Cancellation**: `ReadLineContext` for non-blocking cancellation and timeouts.
- 🔄 **Undo & Redo**: Multi-level undo stack (`Ctrl+_`) and redo support.
- ✂️ **Kill Ring & Yank**: Full kill ring storing deletions from `Ctrl+K`, `Ctrl+U`, `Ctrl+W`, and `Alt+D` with yank (`Ctrl+Y`) and yank-pop (`Alt+Y`).
- 🎨 **Visual Styling Integration**: Features a companion subpackage [`format`](./format) for advanced foreground, background, and text decoration setups.
- 🇯🇵 **Advanced Unicode Metrics**: Correctly handles multi-byte UTF-8, zero-width characters (combining marks, zero-width joiners, variation selectors), and CJK wide characters/emojis for visually aligned cursors.
- 🕒 **Persistent History**: Browsable command history with auto-deduplication, prefix-based search, cross-session file saving/loading, and compaction.
- 📟 **Responsive watch-resize**: Hook to handle terminal dimensions change gracefully.

---

## Installation

```bash
go get github.com/hitraa/readline
```

---

## Quick Start

Here is a minimal, complete example showing how to initialize the terminal line editor:

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hitraa/readline"
)

func main() {
	config := readline.Config{
		Prompt:        "> ",
		HistoryFile:   os.ExpandEnv("$HOME/.my_app_history"),
		MaxHistory:    1000,
		EnableSignals: true, // Allow Ctrl+Z to suspend the process
		Completer:     readline.PrefixCompleter("help", "exit", "status", "version"),
	}

	ed, err := readline.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing editor: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Interactive Shell. Press Tab for completion, Ctrl+R for search, Ctrl+D to exit.")

	for {
		line, err := ed.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\nGoodbye!")
				break
			}
			if errors.Is(err, readline.ErrInterrupt) {
				// Ctrl+C pressed; clear prompt and continue
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}

		if line == "" {
			continue
		}

		fmt.Printf("Received input: %s\n", line)
	}
}
```

---

## Key Bindings

`readline` supports native keyboard shortcuts to streamline interactive typing:

### Cursor Movement

- **Ctrl + A** or **Home**: Move cursor to start of line.
- **Ctrl + E** or **End**: Move cursor to end of line.
- **Ctrl + B** or **Left Arrow**: Move cursor backward one character.
- **Ctrl + F** or **Right Arrow**: Move cursor forward one character.
- **Alt + B** or **Ctrl + Left**: Move cursor backward one word.
- **Alt + F** or **Ctrl + Right**: Move cursor forward one word.

### Command History & Search

- **Ctrl + P** or **Up Arrow**: Browse backward through history.
- **Ctrl + N** or **Down Arrow**: Browse forward through history.
- **Ctrl + R**: Start reverse incremental search. Repeated `Ctrl+R` moves to older matches; `Ctrl+S` moves to newer matches; `Enter` accepts; `Esc` or `Ctrl+G` cancels.
- **Page Up**: Search backward in history for entries starting with the current input prefix.
- **Page Down**: Search forward in history for entries starting with the current input prefix.

### Text Editing & Kill Ring

- **Backspace** or **Ctrl + H**: Delete character before cursor.
- **Delete** or **Ctrl + D** (on non-empty line): Delete character at cursor.
- **Ctrl + D** (on empty line): Send End-Of-File (EOF) to close editor.
- **Ctrl + K**: Kill all text from cursor to end of line (saved to kill ring).
- **Ctrl + U**: Kill all text from start of line to cursor (saved to kill ring).
- **Ctrl + W** or **Alt + Backspace**: Kill word before cursor (saved to kill ring).
- **Alt + D**: Kill word after cursor (saved to kill ring).
- **Ctrl + Y**: Yank (paste) the most recently killed text.
- **Alt + Y**: Yank-pop: cycle through previous killed texts in the kill ring.
- **Ctrl + T**: Transpose adjacent characters.
- **Ctrl + _**: Undo last modification.

### Autocompletion

- **Tab**: Trigger autocompletion or cycle through available candidates.
- **Shift + Tab**: Cycle backward through candidates.

### Screen & Control

- **Ctrl + L**: Clear terminal screen and redraw line.
- **Ctrl + C**: Interrupt current input session.

---

## Architecture and Design

The library is designed modularly to ensure safety and testability:

- **LineBuffer (`buffer.go`)**: Manages the UTF-8 rune buffer, edit cursor, word operations, and visual column width metrics.
- **Renderer (`render.go`)**: Optimizes terminal redrawing, managing multi-row wrapping, screen clearing, and cursor tracking.
- **Completer (`complete.go`)**: Provides flexible completion interfaces and adaptive column grid rendering.
- **Quote Parser (`quote.go`)**: Lexes quotes (`"..."`, `'...'`) and backslash escapes (`\ `) for word tokenization.
- **SearchMode (`search.go`)**: Manages the reverse incremental history search state machine.
- **KillRing (`killring.go`)**: Implements an Emacs-style kill ring buffer.
- **UndoStack (`undo.go`)**: Manages multi-level undo and redo operations.
- **Terminal Drivers (`terminal_linux.go`, `terminal_darwin.go`, `terminal_windows.go`)**: Implements pure Go OS-level raw terminal and console modes with non-TTY fallbacks.
- **Editor (`editor.go`)**: Coordinates terminal state, events, signal watchers, history, and key dispatching.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
