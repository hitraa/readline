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
	format.Enable()

	histFile := os.ExpandEnv("$HOME/.my_input_history")

	// Create a beautiful prompt using the format package
	prompt := format.StyleBoldGreen.Sprint("my_input") + format.StyleBold.Sprint(":") + " "

	// Configure built-in commands for autocompletion
	commands := []string{"help", "exit", "quit", "history", "clear", "password", "echo", "status"}

	config := readline.Config{
		Prompt:        prompt,
		HistoryFile:   histFile,
		MaxHistory:    1000,
		EnableSignals: true, // Allow Ctrl+Z to suspend normally
		Completer:     readline.PrefixCompleter(commands...),
	}

	ed, err := readline.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", format.StyleBoldRed.Sprint("readline error:"), err)
		os.Exit(1)
	}
	defer ed.Close()

	// Optional: hook terminal resize
	ed.WatchResize(func(cols, rows int) {
		_ = cols
		_ = rows
	})

	// Print styled banners
	fmt.Printf("%s\r\n", format.New().Bold().Fg(format.Cyan).Sprint("My Interactive Shell"))
	fmt.Printf("%s\r\n", format.StyleFaint.Sprint("Ctrl+R: reverse search | Tab: autocompletion | Ctrl+C: cancel | Ctrl+D: exit"))
	fmt.Println()

	for {
		line, err := ed.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Printf("\r\n%s\r\n", format.StyleBoldGreen.Sprint("Goodbye!"))
				return
			}
			if errors.Is(err, readline.ErrInterrupt) {
				// Ctrl+C pressed — just prompt again.
				continue
			}
			fmt.Fprintf(os.Stderr, "%s %v\r\n", format.StyleBoldRed.Sprint("error:"), err)
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
			ed.Close()
			fmt.Printf("%s\r\n", format.StyleBoldGreen.Sprint("Goodbye!"))
			return

		case "help":
			fmt.Printf("%s\r\n", format.StyleBold.Sprint("Available commands:"))
			for _, cmd := range commands {
				fmt.Printf("  - %s\r\n", format.StyleCyan.Sprint(cmd))
			}

		case "password":
			maskRune := '*'
			if len(fields) > 1 {
				arg := fields[1]
				if arg == "silent" || arg == "none" {
					maskRune = 0
				} else {
					maskRune = []rune(arg)[0]
				}
			}
			prompt := fmt.Sprintf("Enter secret (mask: %c, or use 'password silent'): ", maskRune)
			if maskRune == 0 {
				prompt = "Enter secret (silent, no echo): "
			}
			pw, err := ed.ReadPasswordWithMask(maskRune, prompt)
			if err != nil {
				fmt.Printf("Password entry cancelled (%v)\r\n", err)
			} else {
				fmt.Printf("Received token length: %d characters (not persisted to history)\r\n", len(pw))
			}

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
			ed.ClearScreen()

		default:
			cmdStyle := format.New().Bold().Fg(format.BrightYellow)
			fmt.Printf("Executed: %s\r\n", cmdStyle.Sprint(line))
		}
	}
}
