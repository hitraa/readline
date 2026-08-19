# readline

A production-grade, zero-dependency, pure Go terminal line editor designed for building interactive command-line interfaces.

It is Linux, macOS, and Windows native, requiring **no CGO** and **no third-party dependencies**. It includes native UTF-8 (CJK character width) support, full history management, history prefix-search, and Emacs-style terminal key bindings.

---

## Features

- 📦 **Zero Dependencies**: Pure Go implementation using OS-specific termios/console mode system calls.
- ⚡ **No CGO**: Light binaries, fast builds, and seamless cross-compilation.
- 🎨 **Visual Styling Integration**: Features a companion subpackage [`format`](./format) for advanced foreground, background, and text decoration setups.
- 🇯🇵 **CJK Support**: Correctly handles multi-byte UTF-8 character metrics (such as Japanese, Chinese, and Korean characters) for visually aligned prompts and cursors.
- 🕒 **Persistent History**: Browsable command history with auto-deduplication, prefix-based search, and cross-session file saving/loading.
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
	}

	ed, err := readline.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing editor: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Interactive Shell. Press Ctrl+D to exit, Ctrl+C to cancel.")

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

## Advanced Visual Formatting

Use the built-in [`format`](./format) subpackage to easily style your prompt and output without manually writing ANSI escape codes:

```go
package main

import (
	"github.com/hitraa/readline"
	"github.com/hitraa/readline/format"
)

func main() {
	// Build a beautiful colored prompt
	prompt := format.StyleBoldGreen.Sprint("myapp") + format.StyleBold.Sprint(">") + " "

	config := readline.Config{
		Prompt: prompt,
	}

	ed, err := readline.New(config)
	// ... handling & defer close

	// Styling normal prints
	format.StyleBoldCyan.Println("--- Advanced Shell Active ---")
}
```

---

## Emacs Key Bindings

`readline` supports native keyboard shortcuts to streamline interactive typing:

### Cursor Movement

- **Ctrl + A** or **Home**: Move cursor to start of line.
- **Ctrl + E** or **End**: Move cursor to end of line.
- **Ctrl + B** or **Left Arrow**: Move cursor backward one character.
- **Ctrl + F** or **Right Arrow**: Move cursor forward one character.

### Command History

- **Ctrl + P** or **Up Arrow**: Browse backward through history.
- **Ctrl + N** or **Down Arrow**: Browse forward through history.
- **Page Up**: Search backward in history for entries starting with the current input prefix.
- **Page Down**: Search forward in history for entries starting with the current input prefix.

### Text Editing

- **Backspace**: Delete character before cursor.
- **Delete** or **Ctrl + D** (on non-empty line): Delete character at cursor.
- **Ctrl + D** (on empty line): Send End-Of-File (EOF) to close editor.
- **Ctrl + K**: Kill/delete all text from the cursor to the end of the line.
- **Ctrl + U**: Kill/delete all text from the cursor to the start of the line.
- **Ctrl + W**: Kill/delete the word immediately preceding the cursor.
- **Ctrl + C**: Interrupt current input session.

---

## Architecture and Design

The library is designed modularly to ensure safety and testability:

- **LineBuffer (`buffer.go`)**: Manages the UTF-8 rune buffer, tracks the edit cursor, and calculates the actual visual column width of runes (handling CJK characters).
- **History (`history.go`)**: An ring-buffer history management subsystem supporting standard browsing and prefix-based searching.
- **EscapeParser (`escape.go`)**: Converts raw byte streams from stdin into high-level terminal action events.
- **Renderer (`render.go`)**: Optimizes terminal redrawing by using ANSI control commands, wiping lines, and keeping cursor alignment.
- **Editor (`editor.go`)**: Integrates terminal state modifications (raw mode termios), OS signals, event reading loops, and binds actions together.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

