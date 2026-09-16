package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hitraa/readline"
)

func main() {
	ed, err := readline.New(readline.Config{
		Prompt: "timed (5s limit)> ",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Context Cancellation Example: You have 5 seconds to answer.")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	line, err := ed.ReadLineContext(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Timed out! Moving on...")
			return
		}
		fmt.Printf("Encountered error: %v\n", err)
		return
	}

	fmt.Printf("Answer submitted in time: %s\n", line)
}
