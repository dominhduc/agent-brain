package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdUpdateEntry() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: brain edit <topic> <timestamp-prefix> --message \"<new message>\"")
		fmt.Println()
		fmt.Println("Example: brain edit gotchas \"2026-04-03 08\" --message \"New message\"")
		fmt.Println()
		fmt.Println("What to do: specify topic, timestamp prefix, and new message.")
		os.Exit(1)
	}

	topic := os.Args[2]
	timestampPrefix := os.Args[3]

	messageIdx := -1
	for i := 4; i < len(os.Args); i++ {
		if os.Args[i] == "--message" && i+1 < len(os.Args) {
			messageIdx = i + 1
			break
		}
	}
	if messageIdx < 0 {
		fmt.Fprintln(os.Stderr, "Error: --message flag is required")
		fmt.Fprintln(os.Stderr, "What to do: brain update gotchas \"2026-04-03 08\" --message \"new text\"")
		os.Exit(1)
	}

	newMessage := os.Args[messageIdx]

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := hub.UpdateEntry(topic, timestampPrefix, newMessage); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s entry. Previous version archived.\n", topic)
}

func cmdSupersedeEntry() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: brain supersede <topic> <old-timestamp> <new-timestamp>")
		fmt.Println()
		fmt.Println("Example: brain supersede gotchas \"2026-04-03 08:42:10\" \"2026-04-18 10:00:00\"")
		fmt.Println()
		fmt.Println("What to do: specify topic and both timestamps.")
		os.Exit(1)
	}

	topic := os.Args[2]
	oldTS := strings.Trim(os.Args[3], "\"")
	newTS := strings.Trim(os.Args[4], "\"")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := hub.SupersedeEntry(topic, oldTS, newTS); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Superseded entry. Old entry marked as superseded.\n")
}
