package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/secrets"
)

func cmdAdd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: brain add <topic> \"<message>\"")
		fmt.Println("Topics: gotcha, pattern, decision, architecture, memory")
		fmt.Println("What to do: provide a topic and a message to add.")
		os.Exit(1)
	}

	topic := os.Args[2]
	message := strings.Join(os.Args[3:], " ")

	if len(message) > maxMessageLen {
		fmt.Fprintf(os.Stderr, "Error: message too long (%d bytes, max %d).\nWhat to do: shorten your message or split it into multiple entries.\n", len(message), maxMessageLen)
		os.Exit(1)
	}

	if secrets.HasSecrets(message) {
		findings := secrets.Scan(message)
		fmt.Fprintf(os.Stderr, "Error: your message may contain a secret (detected: %s).\n", findings[0].Type)
		fmt.Fprintln(os.Stderr, "What to do: redact the sensitive value and try again.")
		os.Exit(1)
	}

	topicMap := map[string]string{
		"gotcha":       "gotchas",
		"pattern":      "patterns",
		"decision":     "decisions",
		"architecture": "architecture",
		"memory":       "memory",
	}

	normalized, ok := topicMap[strings.ToLower(topic)]
	if !ok {
		fmt.Printf("Unknown topic '%s'. Available topics: gotcha, pattern, decision, architecture, memory\n", topic)
		fmt.Println("What to do: use one of the listed topic names.")
		os.Exit(1)
	}

	if err := brain.AddEntry(normalized, message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: make sure you are in a project with .brain/ initialized.\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added to %s\n", normalized)
}
