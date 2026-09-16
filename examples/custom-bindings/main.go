package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hitraa/readline"
)

func main() {
	km := make(readline.KeyMap)

	// Bind Ctrl+X to clear the current line and insert a default timestamp
	km.Bind(readline.KeyCtrlX, func(e *readline.Editor, buf *readline.LineBuffer) (bool, string, error) {
		buf.Clear()
		buf.InsertString("timestamp: 2026-09-16T22:00:00Z ")
		return false, "", nil
	})

	// Bind Ctrl+O to instantly submit with a canned command
	km.Bind(readline.KeyCtrlO, func(e *readline.Editor, buf *readline.LineBuffer) (bool, string, error) {
		return true, "canned-shortcut-command", nil
	})

	ed, err := readline.New(readline.Config{
		Prompt: "custom> ",
		KeyMap: km,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Custom Key Bindings Example:")
	fmt.Println(" - Press Ctrl+X to insert a canned timestamp into the line.")
	fmt.Println(" - Press Ctrl+O to immediately execute 'canned-shortcut-command'.")
	fmt.Println(" - Press Ctrl+D to exit.")

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
		fmt.Printf("Received: %s\n", line)
	}
}
