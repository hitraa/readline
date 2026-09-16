package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hitraa/readline"
)

func main() {
	histPath := filepath.Join(os.TempDir(), "readline_example_history.txt")

	ed, err := readline.New(readline.Config{
		Prompt:      "hist> ",
		HistoryFile: histPath,
		MaxHistory:  200,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Printf("History saved to %s\n", histPath)
	fmt.Println("Use Up/Down arrows or Ctrl+P/Ctrl+N to navigate history.")
	fmt.Println("Use PageUp/PageDown to prefix-search history.")
	fmt.Println("Use Ctrl+R to start reverse incremental search.")

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
		if line == "history" {
			for i, entry := range ed.History() {
				fmt.Printf("%3d: %s\n", i+1, entry)
			}
			continue
		}
		fmt.Printf("Executed: %s\n", line)
	}
}
