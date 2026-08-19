package format

import (
	"bytes"
	"os"
	"testing"
)

func TestHexParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected Color
		valid    bool
	}{
		{"#FF5733", RGB(255, 87, 51), true},
		{"FF5733", RGB(255, 87, 51), true},
		{"#F53", RGB(255, 85, 51), true}, // F53 translates to FF5533
		{"F53", RGB(255, 85, 51), true},
		{"#ff5733", RGB(255, 87, 51), true}, // lowercase hex
		{"ff5733", RGB(255, 87, 51), true},  // lowercase hex without hash
		{"#f53", RGB(255, 85, 51), true},    // lowercase short hex
		{"#invalid", Color{mode: modeNone}, false},
		{"", Color{mode: modeNone}, false},
		{"#FF573", Color{mode: modeNone}, false},
		{"FF5733F", Color{mode: modeNone}, false},
		{"#G00000", Color{mode: modeNone}, false},
	}

	for _, tc := range tests {
		got := Hex(tc.input)
		if tc.valid {
			if got.mode != modeRGB || got.r != tc.expected.r || got.g != tc.expected.g || got.b != tc.expected.b {
				t.Errorf("Hex(%q) = RGB(%d,%d,%d); want RGB(%d,%d,%d)", tc.input, got.r, got.g, got.b, tc.expected.r, tc.expected.g, tc.expected.b)
			}
			if got.IsNone() {
				t.Errorf("expected IsNone() to be false for valid hex %q", tc.input)
			}
		} else {
			if got.mode != modeNone {
				t.Errorf("Hex(%q) got mode %v; want modeNone", tc.input, got.mode)
			}
			if !got.IsNone() {
				t.Errorf("expected IsNone() to be true for invalid hex %q", tc.input)
			}
		}
	}
}

func TestColorSequences(t *testing.T) {
	// Standard
	if seq := Red.Sequence(false); seq != "31" {
		t.Errorf("Red.Sequence(false) = %q; want %q", seq, "31")
	}
	if seq := Red.Sequence(true); seq != "41" {
		t.Errorf("Red.Sequence(true) = %q; want %q", seq, "41")
	}
	if seq := BrightBlue.Sequence(false); seq != "94" {
		t.Errorf("BrightBlue.Sequence(false) = %q; want %q", seq, "94")
	}
	if seq := BrightBlue.Sequence(true); seq != "104" {
		t.Errorf("BrightBlue.Sequence(true) = %q; want %q", seq, "104")
	}

	// 256
	c256 := Color256(128)
	if seq := c256.Sequence(false); seq != "38;5;128" {
		t.Errorf("c256.Sequence(false) = %q; want %q", seq, "38;5;128")
	}
	if seq := c256.Sequence(true); seq != "48;5;128" {
		t.Errorf("c256.Sequence(true) = %q; want %q", seq, "48;5;128")
	}

	// RGB
	cRGB := RGB(10, 20, 30)
	if seq := cRGB.Sequence(false); seq != "38;2;10;20;30" {
		t.Errorf("cRGB.Sequence(false) = %q; want %q", seq, "38;2;10;20;30")
	}
	if seq := cRGB.Sequence(true); seq != "48;2;10;20;30" {
		t.Errorf("cRGB.Sequence(true) = %q; want %q", seq, "48;2;10;20;30")
	}

	// Empty color
	cNone := Color{mode: modeNone}
	if seq := cNone.Sequence(false); seq != "" {
		t.Errorf("cNone.Sequence(false) = %q; want empty string", seq)
	}
}

func TestStyleSequence(t *testing.T) {
	s := New().Bold().Italic().Fg(Red).Bg(Black)
	expected := "\033[1;3;31;40m"
	if got := s.escapeSequence(); got != expected {
		t.Errorf("style.escapeSequence() = %q; want %q", got, expected)
	}

	// Check order of attributes and background
	s2 := New().Bg(BrightWhite).Fg(BrightBlue).Underline().CrossOut()
	expected2 := "\033[4;9;94;107m"
	if got := s2.escapeSequence(); got != expected2 {
		t.Errorf("style.escapeSequence() = %q; want %q", got, expected2)
	}

	// Test empty style
	if got := New().escapeSequence(); got != "" {
		t.Errorf("empty style escapeSequence = %q; want empty string", got)
	}

	// Test all attributes
	sAll := New().Bold().Faint().Italic().Underline().Blink().Reverse().Conceal().CrossOut()
	expectedAll := "\033[1;2;3;4;5;7;8;9m"
	if got := sAll.escapeSequence(); got != expectedAll {
		t.Errorf("sAll.escapeSequence() = %q; want %q", got, expectedAll)
	}
}

func TestStyleSprint(t *testing.T) {
	oldNoColor := noColor
	defer func() { noColor = oldNoColor }()
	noColor = false

	s := New().Bold().Fg(Red)

	// Test Sprint
	got := s.Sprint("hello")
	expected := "\033[1;31mhello\033[0m"
	if got != expected {
		t.Errorf("Sprint() = %q; want %q", got, expected)
	}

	// Test Sprint with empty string
	if gotEmpty := s.Sprint(""); gotEmpty != "" {
		t.Errorf("Sprint(\"\") = %q; want empty string", gotEmpty)
	}

	// Test Sprintf
	gotf := s.Sprintf("hello %s", "world")
	expectedf := "\033[1;31mhello world\033[0m"
	if gotf != expectedf {
		t.Errorf("Sprintf() = %q; want %q", gotf, expectedf)
	}

	if gotfEmpty := s.Sprintf(""); gotfEmpty != "" {
		t.Errorf("Sprintf(\"\") = %q; want empty string", gotfEmpty)
	}

	// Test Sprintln
	gotln := s.Sprintln("hello")
	expectedln := "\033[1;31mhello\033[0m\n"
	if gotln != expectedln {
		t.Errorf("Sprintln() = %q; want %q", gotln, expectedln)
	}

	if gotlnEmpty := s.Sprintln(""); gotlnEmpty != "\n" {
		t.Errorf("Sprintln(\"\") = %q; want %q", gotlnEmpty, "\n")
	}
}

func TestGlobalNoColor(t *testing.T) {
	oldNoColor := noColor
	defer func() { noColor = oldNoColor }()

	noColor = true
	s := New().Bold().Fg(Red)
	if got := s.Sprint("hello"); got != "hello" {
		t.Errorf("Sprint() with noColor=true got %q; want %q", got, "hello")
	}
	if got := s.Sprintf("hello %s", "world"); got != "hello world" {
		t.Errorf("Sprintf() with noColor=true got %q; want %q", got, "hello world")
	}
	if got := s.Sprintln("hello"); got != "hello\n" {
		t.Errorf("Sprintln() with noColor=true got %q; want %q", got, "hello\n")
	}

	// Test ForceColor override
	sForce := s.ForceColor(true)
	expected := "\033[1;31mhello\033[0m"
	if got := sForce.Sprint("hello"); got != expected {
		t.Errorf("ForceColor(true) Sprint() got %q; want %q", got, expected)
	}
}

func TestFprint(t *testing.T) {
	oldNoColor := noColor
	defer func() { noColor = oldNoColor }()
	noColor = false

	s := New().Bold().Fg(Green)

	// Fprint
	var buf bytes.Buffer
	n, err := s.Fprint(&buf, "test")
	if err != nil {
		t.Fatalf("Fprint err: %v", err)
	}
	expected := "\033[1;32mtest\033[0m"
	if got := buf.String(); got != expected {
		t.Errorf("Fprint() wrote %q; want %q", got, expected)
	}
	if n != len(expected) {
		t.Errorf("Fprint() returned len %d; want %d", n, len(expected))
	}

	// Fprintf
	buf.Reset()
	n, err = s.Fprintf(&buf, "hello %d", 42)
	if err != nil {
		t.Fatalf("Fprintf err: %v", err)
	}
	expectedf := "\033[1;32mhello 42\033[0m"
	if got := buf.String(); got != expectedf {
		t.Errorf("Fprintf() wrote %q; want %q", got, expectedf)
	}

	// Fprintln
	buf.Reset()
	n, err = s.Fprintln(&buf, "hello")
	if err != nil {
		t.Fatalf("Fprintln err: %v", err)
	}
	expectedln := "\033[1;32mhello\033[0m\n"
	if got := buf.String(); got != expectedln {
		t.Errorf("Fprintln() wrote %q; want %q", got, expectedln)
	}
}

func TestStrip(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\033[1;31mhello\033[0m", "hello"},
		{"\033[2J\033[Hhello", "hello"},
		{"\x1b[1;32margos\x1b[0m\x1b[1m:\x1b[0m ", "argos: "},
		{"normal text", "normal text"},
		{"\x1b[m", ""},
		{"\x1b[?25h", ""},
	}

	for _, tc := range tests {
		if got := Strip(tc.input); got != tc.expected {
			t.Errorf("Strip(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestPackageHelpers(t *testing.T) {
	oldNoColor := noColor
	defer func() { noColor = oldNoColor }()
	noColor = false

	if got := SprintBold("bold"); got != "\033[1mbold\033[0m" {
		t.Errorf("SprintBold got %q", got)
	}
	if got := SprintRed("red"); got != "\033[31mred\033[0m" {
		t.Errorf("SprintRed got %q", got)
	}
	if got := SprintGreen("green"); got != "\033[32mgreen\033[0m" {
		t.Errorf("SprintGreen got %q", got)
	}
	if got := SprintYellow("yellow"); got != "\033[33myellow\033[0m" {
		t.Errorf("SprintYellow got %q", got)
	}
	if got := SprintBlue("blue"); got != "\033[34mblue\033[0m" {
		t.Errorf("SprintBlue got %q", got)
	}
	if got := SprintMagenta("magenta"); got != "\033[35mmagenta\033[0m" {
		t.Errorf("SprintMagenta got %q", got)
	}
	if got := SprintCyan("cyan"); got != "\033[36mcyan\033[0m" {
		t.Errorf("SprintCyan got %q", got)
	}
	if got := SprintWhite("white"); got != "\033[37mwhite\033[0m" {
		t.Errorf("SprintWhite got %q", got)
	}

	if got := SprintfBold("hello %s", "world"); got != "\033[1mhello world\033[0m" {
		t.Errorf("SprintfBold got %q", got)
	}
	if got := SprintfRed("hello %s", "world"); got != "\033[31mhello world\033[0m" {
		t.Errorf("SprintfRed got %q", got)
	}
	if got := SprintfGreen("hello %s", "world"); got != "\033[32mhello world\033[0m" {
		t.Errorf("SprintfGreen got %q", got)
	}
	if got := SprintfYellow("hello %s", "world"); got != "\033[33mhello world\033[0m" {
		t.Errorf("SprintfYellow got %q", got)
	}
	if got := SprintfBlue("hello %s", "world"); got != "\033[34mhello world\033[0m" {
		t.Errorf("SprintfBlue got %q", got)
	}
	if got := SprintfMagenta("hello %s", "world"); got != "\033[35mhello world\033[0m" {
		t.Errorf("SprintfMagenta got %q", got)
	}
	if got := SprintfCyan("hello %s", "world"); got != "\033[36mhello world\033[0m" {
		t.Errorf("SprintfCyan got %q", got)
	}
	if got := SprintfWhite("hello %s", "world"); got != "\033[37mhello world\033[0m" {
		t.Errorf("SprintfWhite got %q", got)
	}
}

func TestEnvironmentDetection(t *testing.T) {
	// Mock NO_COLOR environment variable.
	os.Setenv("NO_COLOR", "1")
	detectNoColor()
	if !noColor {
		t.Errorf("expected noColor to be true when NO_COLOR environment variable is set")
	}

	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "dumb")
	detectNoColor()
	if !noColor {
		t.Errorf("expected noColor to be true when TERM=dumb")
	}
}

func TestManualControlHelpers(t *testing.T) {
	oldNoColor := noColor
	defer func() { noColor = oldNoColor }()

	Disable()
	if IsEnabled() {
		t.Error("expected IsEnabled() to be false after Disable()")
	}
	if !noColor {
		t.Error("expected noColor to be true after Disable()")
	}

	Enable()
	if !IsEnabled() {
		t.Error("expected IsEnabled() to be true after Enable()")
	}
	if noColor {
		t.Error("expected noColor to be false after Enable()")
	}
}

func TestIsTerminalFallback(t *testing.T) {
	// The fallback implementation should run fine
	f, err := os.Open(os.DevNull)
	if err == nil {
		defer f.Close()
		// Running isTerminal should not crash on any platform
		_ = isTerminal(f)
	}
}

