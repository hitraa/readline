package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hitraa/readline"
)

func main() {
	ed, err := readline.New(readline.Config{
		Prompt: "basic> ",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Basic Readline Example. Type input and press Enter. Ctrl+D to exit.")

	for {
		line, err := ed.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\nBye!")
				break
			}
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}
		if line == "exit" {
			break
		}
		fmt.Printf("You entered: %s\n", line)
	}
}
