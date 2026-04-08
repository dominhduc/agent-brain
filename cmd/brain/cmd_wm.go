package main

import (
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/wm"
)

func cmdWM() {
	fmt.Fprintln(os.Stderr, "Warning: 'brain wm' is deprecated. Use 'brain add --wm' instead.")

	if len(os.Args) < 3 {
		fmt.Println("Usage: brain wm <subcommand>")
		fmt.Println("\nSubcommands:")
		fmt.Println("  push \"<content>\" [--importance 0.5]   Add a working memory note")
		fmt.Println("  read                                     Show current working notes")
		fmt.Println("  clear                                    Clear working memory")
		fmt.Println("  flush                                    Flush (same as clear)")
		os.Exit(1)
	}

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sub := os.Args[2]
	switch sub {
	case "push":
		if len(os.Args) < 4 {
			fmt.Println("Usage: brain wm push \"<content>\" [--importance 0.9]")
			os.Exit(1)
		}
		content := os.Args[3]
		importance := 0.5
		for i := 4; i < len(os.Args); i++ {
			if os.Args[i] == "--importance" && i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%f", &importance)
			}
		}
		if err := wm.Push(brainDir, content, importance); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Working memory updated.")
	case "read":
		entries, err := wm.Read(brainDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(entries) == 0 {
			fmt.Println("Working memory is empty.")
			return
		}
		for _, e := range entries {
			fmt.Printf("• %s\n", e.Content)
		}
	case "clear":
		if err := wm.Clear(brainDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Working memory cleared.")
	case "flush":
		if err := wm.Flush(brainDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Working memory flushed.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown wm subcommand: %s\n", sub)
		os.Exit(1)
	}
}
