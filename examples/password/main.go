package main

import (
	"fmt"
	"os"

	"github.com/hitraa/readline"
)

func main() {
	ed, err := readline.New(readline.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer ed.Close()

	fmt.Println("Password Input Example (passwords are never saved to history)")

	// 1. Silent secret input
	pass1, err := ed.ReadPassword("Enter password (silent): ")
	if err != nil {
		fmt.Printf("Cancelled: %v\n", err)
		return
	}
	fmt.Printf("Received %d character password\n", len(pass1))

	// 2. Masked secret input
	edMasked, _ := readline.New(readline.Config{Mask: '*'})
	defer edMasked.Close()

	pass2, err := edMasked.ReadPassword("Enter token (masked with *): ")
	if err != nil {
		fmt.Printf("Cancelled: %v\n", err)
		return
	}
	fmt.Printf("Received %d character token\n", len(pass2))
}
