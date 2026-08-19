package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	readline "github.com/hitraa/readline"
	format "github.com/hitraa/readline/format"
)

func main() {
	// Enable color explicitly for the example if it was disabled by TTY check
	// (so that even in subagent/non-interactive test runs, we can see the codes or test functionality,
	// but normally it will auto-detect TTY). Let's make sure it's enabled for this demo.
	format.Enable()

	histFile := os.ExpandEnv("$HOME/.argos_history")

	// Create a beautiful prompt using the format package
	prompt := format.StyleBoldGreen.Sprint("argos") + format.StyleBold.Sprint(":") + " "

	config := readline.Config{
		Prompt:        prompt,
		HistoryFile:   histFile,
		MaxHistory:    1000,
		EnableSignals: true, // Ctrl+Z suspends normally
	}

	ed, err := readline.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", format.StyleBoldRed.Sprint("readline error:"), err)
		os.Exit(1)
	}
	defer ed.Close()

	// Optional: redraw on terminal resize.
	ed.WatchResize(func(cols, rows int) {
		// In a real app you might re-wrap output here.
		_ = cols
		_ = rows
	})

	// Print styled banners
	fmt.Printf("%s\r\n", format.New().Bold().Fg(format.Cyan).Sprint("Argos Interactive Shell"))
	fmt.Printf("%s\r\n", format.StyleFaint.Sprint("Ctrl+C to cancel, Ctrl+D to exit, Up/Down for history"))
	fmt.Println()

	for {
		line, err := ed.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("\r\n%s\r\n", format.StyleBoldGreen.Sprint("Goodbye!"))
				return
			}
			if errors.Is(err, readline.ErrInterrupt) {
				// Ctrl+C pressed — just prompt again.
				continue
			}
			fmt.Fprintf(os.Stderr, "%s %v\n", format.StyleBoldRed.Sprint("error:"), err)
			return
		}

		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch strings.ToLower(fields[0]) {
		case "exit", "quit", "close":
			fmt.Printf("%s\r\n", format.StyleBoldGreen.Sprint("Goodbye!"))
			return
		case "history":
			historyItems := ed.History()
			if len(historyItems) == 0 {
				fmt.Printf("%s\r\n", format.StyleItalic.Sprint("No history recorded yet."))
				continue
			}
			for i, h := range historyItems {
				fmt.Printf("%s %s\r\n", format.StyleFaint.Sprintf("%3d)", i+1), format.StyleBoldWhite.Sprint(h))
			}
		case "clear":
			fmt.Print("\033[2J\033[H")
		default:
			// Restore cooked mode for child-process output, then re-enter raw.
			ed.Close()

			// Highlight the executed command in light cyan/yellow
			cmdStyle := format.New().Bold().Fg(format.BrightYellow)
			fmt.Printf("Executed: %s\r\n", cmdStyle.Sprint(line))

			// Re-open the editor (reuses the same history file).
			ed, err = readline.New(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s %v\n", format.StyleBoldRed.Sprint("readline error:"), err)
				return
			}
		}
	}
}
