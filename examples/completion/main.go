package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hitraa/readline"
)

func main() {
	commands := []string{
		"deploy",
		"destroy",
		"diagnose",
		"diff",
		"doctor",
		"down",
		"drain",
		"dump",
		"help",
		"status",
		"version",
		"exit",
	}

	ed, err := readline.New(readline.Config{
		Prompt:    "cli> ",
		Completer: readline.PrefixCompleter(commands...),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Autocompletion Example. Type 'd' and press Tab to see the completion grid or cycle candidates.")
	fmt.Println("Press Tab to cycle forward, Shift+Tab to cycle backward.")

	for {
		line, err := ed.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, readline.ErrInterrupt) {
				break
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}
		if line == "exit" {
			break
		}
		fmt.Printf("Command received: %s\n", line)
	}
}
