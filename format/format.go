package format

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// colorMode defines the type of color representation.
type colorMode int

const (
	modeNone colorMode = iota
	modeStandard
	mode256
	modeRGB
)

// Color represents a foreground or background color.
type Color struct {
	mode  colorMode
	value uint8
	r     uint8
	g     uint8
	b     uint8
}

// Standard creates a Color using standard 8/16 ANSI codes.
func Standard(code uint8) Color {
	return Color{mode: modeStandard, value: code}
}

// Color256 creates a 256-color palette Color.
func Color256(code uint8) Color {
	return Color{mode: mode256, value: code}
}

// RGB creates a 24-bit TrueColor Color.
func RGB(r, g, b uint8) Color {
	return Color{mode: modeRGB, r: r, g: g, b: b}
}

// Hex parses a hex color string (e.g., "#FF5733", "FF5733", "#F53", "F53") and returns a Color.
// If the string is invalid, it returns a blank Color (mode None).
func Hex(hex string) Color {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	var err error

	if len(hex) == 3 {
		var val uint64
		val, err = strconv.ParseUint(hex, 16, 16)
		if err == nil {
			r = uint8((val>>8)&0xF) * 17
			g = uint8((val>>4)&0xF) * 17
			b = uint8(val&0xF) * 17
		}
	} else if len(hex) == 6 {
		var val uint64
		val, err = strconv.ParseUint(hex, 16, 32)
		if err == nil {
			r = uint8((val >> 16) & 0xFF)
			g = uint8((val >> 8) & 0xFF)
			b = uint8(val & 0xFF)
		}
	} else {
		return Color{mode: modeNone}
	}

	if err != nil {
		return Color{mode: modeNone}
	}
	return RGB(r, g, b)
}

// Sequence returns the ANSI escape sequence parameter for this color.
// If isBackground is true, it returns the background sequence variant.
func (c Color) Sequence(isBackground bool) string {
	switch c.mode {
	case modeStandard:
		code := c.value
		if isBackground {
			code += 10
		}
		return strconv.Itoa(int(code))
	case mode256:
		prefix := "38;5;"
		if isBackground {
			prefix = "48;5;"
		}
		return prefix + strconv.Itoa(int(c.value))
	case modeRGB:
		prefix := "38;2;"
		if isBackground {
			prefix = "48;2;"
		}
		return prefix + strconv.Itoa(int(c.r)) + ";" + strconv.Itoa(int(c.g)) + ";" + strconv.Itoa(int(c.b))
	default:
		return ""
	}
}

// IsNone returns true if the color is unconfigured/empty.
func (c Color) IsNone() bool {
	return c.mode == modeNone
}

// Standard Foreground Colors
var (
	Black         = Standard(30)
	Red           = Standard(31)
	Green         = Standard(32)
	Yellow        = Standard(33)
	Blue          = Standard(34)
	Magenta       = Standard(35)
	Cyan          = Standard(36)
	White         = Standard(37)
	BrightBlack   = Standard(90)
	BrightRed     = Standard(91)
	BrightGreen   = Standard(92)
	BrightYellow  = Standard(93)
	BrightBlue    = Standard(94)
	BrightMagenta = Standard(95)
	BrightCyan    = Standard(96)
	BrightWhite   = Standard(97)
)

// Attribute represents a text decoration bitmask.
type Attribute uint16

const (
	Bold      Attribute = 1 << 0
	Faint     Attribute = 1 << 1
	Italic    Attribute = 1 << 2
	Underline Attribute = 1 << 3
	Blink     Attribute = 1 << 4
	Reverse   Attribute = 1 << 5
	Conceal   Attribute = 1 << 6
	CrossOut  Attribute = 1 << 7
)

// Style represents a combination of color and formatting attributes.
type Style struct {
	fg         Color
	bg         Color
	attrs      Attribute
	forceColor bool
}

// New creates an empty Style.
func New() Style {
	return Style{}
}

// Fg returns a new style with the foreground color set.
func (s Style) Fg(c Color) Style {
	s.fg = c
	return s
}

// Bg returns a new style with the background color set.
func (s Style) Bg(c Color) Style {
	s.bg = c
	return s
}

// Bold returns a new style with the bold attribute enabled.
func (s Style) Bold() Style {
	s.attrs |= Bold
	return s
}

// Faint returns a new style with the faint/dim attribute enabled.
func (s Style) Faint() Style {
	s.attrs |= Faint
	return s
}

// Italic returns a new style with the italic attribute enabled.
func (s Style) Italic() Style {
	s.attrs |= Italic
	return s
}

// Underline returns a new style with the underline attribute enabled.
func (s Style) Underline() Style {
	s.attrs |= Underline
	return s
}

// Blink returns a new style with the blink attribute enabled.
func (s Style) Blink() Style {
	s.attrs |= Blink
	return s
}

// Reverse returns a new style with the reverse/inverse attribute enabled.
func (s Style) Reverse() Style {
	s.attrs |= Reverse
	return s
}

// Conceal returns a new style with the conceal/hidden attribute enabled.
func (s Style) Conceal() Style {
	s.attrs |= Conceal
	return s
}

// CrossOut returns a new style with the strike-through attribute enabled.
func (s Style) CrossOut() Style {
	s.attrs |= CrossOut
	return s
}

// ForceColor overrides the global NO_COLOR check for this style if force is true.
func (s Style) ForceColor(force bool) Style {
	s.forceColor = force
	return s
}

// isDisabled returns true if the style should output plain text (no styling).
func (s Style) isDisabled() bool {
	if s.forceColor {
		return false
	}
	return noColor
}

const resetSeq = "\033[0m"

// escapeSequence returns the raw ANSI escape sequence for this style.
func (s Style) escapeSequence() string {
	var parts []string

	if s.attrs&Bold != 0 {
		parts = append(parts, "1")
	}
	if s.attrs&Faint != 0 {
		parts = append(parts, "2")
	}
	if s.attrs&Italic != 0 {
		parts = append(parts, "3")
	}
	if s.attrs&Underline != 0 {
		parts = append(parts, "4")
	}
	if s.attrs&Blink != 0 {
		parts = append(parts, "5")
	}
	if s.attrs&Reverse != 0 {
		parts = append(parts, "7")
	}
	if s.attrs&Conceal != 0 {
		parts = append(parts, "8")
	}
	if s.attrs&CrossOut != 0 {
		parts = append(parts, "9")
	}

	if s.fg.mode != modeNone {
		parts = append(parts, s.fg.Sequence(false))
	}
	if s.bg.mode != modeNone {
		parts = append(parts, s.bg.Sequence(true))
	}

	if len(parts) == 0 {
		return ""
	}

	return "\033[" + strings.Join(parts, ";") + "m"
}

// Sprint formats using the default formats for its operands and returns the resulting string.
// If the style is disabled or empty, it returns the raw output without styling.
func (s Style) Sprint(a ...any) string {
	str := fmt.Sprint(a...)
	if str == "" {
		return ""
	}
	if s.isDisabled() {
		return str
	}
	return s.escapeSequence() + str + resetSeq
}

// Sprintf formats according to a format specifier and returns the resulting string.
// If the style is disabled or empty, it returns the raw output without styling.
func (s Style) Sprintf(formatStr string, a ...any) string {
	str := fmt.Sprintf(formatStr, a...)
	if str == "" {
		return ""
	}
	if s.isDisabled() {
		return str
	}
	return s.escapeSequence() + str + resetSeq
}

// Sprintln formats using the default formats for its operands and returns the resulting string.
// It ensures styling does not bleed past the final newline.
func (s Style) Sprintln(a ...any) string {
	str := fmt.Sprintln(a...)
	if str == "" {
		return ""
	}
	if str == "\n" {
		return "\n"
	}
	if s.isDisabled() {
		return str
	}
	if strings.HasSuffix(str, "\n") {
		return s.escapeSequence() + str[:len(str)-1] + resetSeq + "\n"
	}
	return s.escapeSequence() + str + resetSeq
}

// Fprint formats using the default formats for its operands and writes to w.
func (s Style) Fprint(w io.Writer, a ...any) (int, error) {
	return io.WriteString(w, s.Sprint(a...))
}

// Fprintf formats according to a format specifier and writes to w.
func (s Style) Fprintf(w io.Writer, formatStr string, a ...any) (int, error) {
	return io.WriteString(w, s.Sprintf(formatStr, a...))
}

// Fprintln formats using the default formats for its operands and writes to w.
func (s Style) Fprintln(w io.Writer, a ...any) (int, error) {
	return io.WriteString(w, s.Sprintln(a...))
}

// Global configuration state
var noColor bool

func init() {
	detectNoColor()
}

func detectNoColor() {
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
		return
	}
	if os.Getenv("TERM") == "dumb" {
		noColor = true
		return
	}
	if !isTerminal(os.Stdout) {
		noColor = true
		return
	}
}

// Disable disables styling globally.
func Disable() {
	noColor = true
}

// Enable enables styling globally.
func Enable() {
	noColor = false
}

// IsEnabled returns true if styling is globally enabled.
func IsEnabled() bool {
	return !noColor
}

// Predefined styles
var (
	StyleBold      = New().Bold()
	StyleFaint     = New().Faint()
	StyleItalic    = New().Italic()
	StyleUnderline = New().Underline()
	StyleBlink     = New().Blink()
	StyleReverse   = New().Reverse()
	StyleConceal   = New().Conceal()
	StyleCrossOut  = New().CrossOut()

	StyleRed     = New().Fg(Red)
	StyleGreen   = New().Fg(Green)
	StyleYellow  = New().Fg(Yellow)
	StyleBlue    = New().Fg(Blue)
	StyleMagenta = New().Fg(Magenta)
	StyleCyan    = New().Fg(Cyan)
	StyleWhite   = New().Fg(White)
	StyleBlack   = New().Fg(Black)

	StyleBrightRed     = New().Fg(BrightRed)
	StyleBrightGreen   = New().Fg(BrightGreen)
	StyleBrightYellow  = New().Fg(BrightYellow)
	StyleBrightBlue    = New().Fg(BrightBlue)
	StyleBrightMagenta = New().Fg(BrightMagenta)
	StyleBrightCyan    = New().Fg(BrightCyan)
	StyleBrightWhite   = New().Fg(BrightWhite)

	StyleBoldRed     = StyleBold.Fg(Red)
	StyleBoldGreen   = StyleBold.Fg(Green)
	StyleBoldYellow  = StyleBold.Fg(Yellow)
	StyleBoldBlue    = StyleBold.Fg(Blue)
	StyleBoldMagenta = StyleBold.Fg(Magenta)
	StyleBoldCyan    = StyleBold.Fg(Cyan)
	StyleBoldWhite   = StyleBold.Fg(White)
)

// Helper functions for quick standard styling
func SprintBold(a ...any) string { return StyleBold.Sprint(a...) }
func SprintRed(a ...any) string  { return StyleRed.Sprint(a...) }
func SprintGreen(a ...any) string { return StyleGreen.Sprint(a...) }
func SprintYellow(a ...any) string { return StyleYellow.Sprint(a...) }
func SprintBlue(a ...any) string { return StyleBlue.Sprint(a...) }
func SprintMagenta(a ...any) string { return StyleMagenta.Sprint(a...) }
func SprintCyan(a ...any) string { return StyleCyan.Sprint(a...) }
func SprintWhite(a ...any) string { return StyleWhite.Sprint(a...) }

func SprintfBold(f string, a ...any) string { return StyleBold.Sprintf(f, a...) }
func SprintfRed(f string, a ...any) string  { return StyleRed.Sprintf(f, a...) }
func SprintfGreen(f string, a ...any) string { return StyleGreen.Sprintf(f, a...) }
func SprintfYellow(f string, a ...any) string { return StyleYellow.Sprintf(f, a...) }
func SprintfBlue(f string, a ...any) string { return StyleBlue.Sprintf(f, a...) }
func SprintfMagenta(f string, a ...any) string { return StyleMagenta.Sprintf(f, a...) }
func SprintfCyan(f string, a ...any) string { return StyleCyan.Sprintf(f, a...) }
func SprintfWhite(f string, a ...any) string { return StyleWhite.Sprintf(f, a...) }

// Standard ANSI sequence regex matching any standard terminal control character seq.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// Strip removes all ANSI escape sequences from the given string.
func Strip(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}
