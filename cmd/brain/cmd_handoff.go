package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/handoff"
)

func cmdHandoff() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain handoff <subcommand>")
		fmt.Println("\nSubcommands:")
		fmt.Println("  create --summary \"...\" --next \"...\" [--session sess_id]")
		fmt.Println("  latest                                      Show latest handoff")
		fmt.Println("  show <id>                                   Show specific handoff")
		fmt.Println("  resume                                      Re-inject latest handoff")
		os.Exit(1)
	}

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sub := os.Args[2]
	switch sub {
	case "create":
		summary := ""
		next := ""
		session := ""
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--summary":
				if i+1 < len(os.Args) {
					summary = os.Args[i+1]
					i++
				}
			case "--next":
				if i+1 < len(os.Args) {
					next = os.Args[i+1]
					i++
				}
			case "--session":
				if i+1 < len(os.Args) {
					session = os.Args[i+1]
					i++
				}
			}
		}
		if summary == "" || next == "" {
			fmt.Println("Usage: brain handoff create --summary \"...\" --next \"...\"")
			os.Exit(1)
		}
		h, err := handoff.Create(brainDir, summary, next, session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Handoff created: %s\n", h.ID)
	case "latest":
		h, err := handoff.Latest(brainDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if h == nil {
			fmt.Println("No handoff found.")
			return
		}
		fmt.Printf("ID:      %s\n", h.ID)
		fmt.Printf("Summary: %s\n", h.Summary)
		fmt.Printf("Next:    %s\n", h.Next)
		if h.Session != "" {
			fmt.Printf("Session: %s\n", h.Session)
		}
		fmt.Printf("Time:    %s\n", h.Timestamp.Format("2006-01-02 15:04:05"))
	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: brain handoff show <id>")
			os.Exit(1)
		}
		h, err := handoff.Show(brainDir, os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if h == nil {
			fmt.Println("Handoff not found.")
			return
		}
		fmt.Printf("ID:      %s\n", h.ID)
		fmt.Printf("Summary: %s\n", h.Summary)
		fmt.Printf("Next:    %s\n", h.Next)
		fmt.Printf("Time:    %s\n", h.Timestamp.Format("2006-01-02 15:04:05"))
	case "resume":
		h, err := handoff.Resume(brainDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if h == nil {
			fmt.Println("No handoff found.")
			return
		}
		fmt.Println("=== LATEST HANDOFF ===")
		fmt.Printf("Summary: %s\n", h.Summary)
		fmt.Printf("Next:    %s\n", h.Next)
	default:
		fmt.Fprintf(os.Stderr, "Unknown handoff subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func parseFlagValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			val := args[i+1]
			if strings.HasPrefix(val, "--") {
				return ""
			}
			return val
		}
	}
	return ""
}
