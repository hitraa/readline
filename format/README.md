# format

A lightweight, zero-dependency, production-grade Go utility to style CLI output with colors (standard 16, 256-color, TrueColor RGB/HEX) and rich text attributes.

Specifically designed for developer comfort, it integrates natively with interactive terminals and readline utilities.

## Features

- ⚡ **Zero External Dependencies**: Implemented purely using Go's standard library and raw OS-specific syscalls.
- 🎨 **Complete Color Support**:
  - 16 Standard colors (e.g. `Red`, `BrightGreen`)
  - 256 ANSI color codes (`Color256(128)`)
  - 24-bit TrueColor RGB (`RGB(255, 87, 51)`)
  - Hex values (`Hex("#FF5733")` or `"FF5733"`)
- 📝 **Rich Typography**: Supports `Bold`, `Faint`, `Italic`, `Underline`, `Blink`, `Reverse`, `Conceal`, and `CrossOut`.
- ⚙️ **Smart Auto-Detection**:
  - Automatically respects the standard `NO_COLOR` environment variable (https://no-color.org/).
  - Disables colors if `TERM=dumb`.
  - Automatically detects if output is redirected/piped (non-TTY) and turns off formatting.
  - Custom overrides to force color via `.ForceColor(true)` or package-level settings.
- 🔗 **Fluent Chainable Builder**: Make formatting intuitive and clean.
- 🧽 **ANSI Stripper**: Cleanly strips ANSI sequences from any string (excellent for logging formatted outputs).

---

## Installation

Import the package directly from the parent repository module:

```go
import "github.com/hitraa/readline/format"
```

---

## Quick Start

### 1. Fluent Builder Style

```go
package main

import (
	"fmt"
	"github.com/hitraa/readline/format"
)

func main() {
	// Simple foreground + background + attributes
	styled := format.New().
		Bold().
		Italic().
		Fg(format.Hex("#FFA500")). // Orange foreground
		Bg(format.RGB(20, 20, 20)). // Dark background
		Sprint("Hello Styled World!")

	fmt.Println(styled)
}
```

### 2. Ready-to-Use Package Helpers

For quick styling, you can use predefined package-level variables and functions:

```go
// Predefined Styles
fmt.Println(format.StyleBoldRed.Sprint("Danger!"))
fmt.Println(format.StyleBoldGreen.Sprint("Success!"))

// Quick formatting helper functions
fmt.Println(format.SprintGreen("Success!"))
fmt.Println(format.SprintfCyan("Loading file: %s", filepath))
```

---

## API Reference

### Text Attributes

| Attribute   | Fluent Method  | Description                           |
| :---------- | :------------- | :------------------------------------ |
| `Bold`      | `.Bold()`      | Embolden text                         |
| `Faint`     | `.Faint()`     | Dim / faint color                     |
| `Italic`    | `.Italic()`    | Italicize text                        |
| `Underline` | `.Underline()` | Underlined text                       |
| `Blink`     | `.Blink()`     | Blinking text                         |
| `Reverse`   | `.Reverse()`   | Invert foreground & background colors |
| `Conceal`   | `.Conceal()`   | Hidden text                           |
| `CrossOut`  | `.CrossOut()`  | Strikethrough text                    |

### Foreground & Background Colors

Colors can be applied using the `.Fg(Color)` and `.Bg(Color)` builder methods.

```go
// Standard 16 colors: Black, Red, Green, Yellow, Blue, Magenta, Cyan, White
// Bright variants: BrightBlack, BrightRed, BrightGreen, BrightYellow, BrightBlue, BrightMagenta, BrightCyan, BrightWhite
style := format.New().Fg(format.BrightRed).Bg(format.Black)

// 256-color palette (0-255)
style := format.New().Fg(format.Color256(128))

// TrueColor RGB
style := format.New().Fg(format.RGB(255, 87, 51))

// Hex string parsing (Supports 3 or 6 digit formats, optional '#' prefix)
style := format.New().Fg(format.Hex("#FF5733")).Bg(format.Hex("000"))
```

### Methods on `Style`

Every `Style` supports the standard Go `fmt` print interfaces:

- `Sprint(a ...interface{}) string`
- `Sprintf(format string, a ...interface{}) string`
- `Sprintln(a ...interface{}) string`
- `Fprint(w io.Writer, a ...interface{}) (int, error)`
- `Fprintf(w io.Writer, format string, a ...interface{}) (int, error)`
- `Fprintln(w io.Writer, a ...interface{}) (int, error)`

### Environment Overrides

```go
// Disable colors globally
format.Disable()

// Enable colors globally
format.Enable()

// Check if colors are globally enabled (true unless TTY check failed or NO_COLOR/dumb terminal is set)
enabled := format.IsEnabled()

// Force formatting for a single style, bypassing NO_COLOR/TTY detection
style := format.New().Bold().Fg(format.Red).ForceColor(true)
```

### Stripping ANSI Sequences

Useful when writing to log files or processing input that contains styling:

```go
rawStr := format.Strip("\033[1;31mStyled Text\033[0m") // Returns "Styled Text"
```
