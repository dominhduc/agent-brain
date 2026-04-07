package main

import (
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/index"
)

func cmdIndex() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain index rebuild")
		fmt.Println("\nCommands:")
		fmt.Println("  rebuild    Rebuild the index from topic files")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "rebuild":
		cmdIndexRebuild()
	default:
		fmt.Fprintf(os.Stderr, "Unknown index subcommand: %s\n", os.Args[2])
		fmt.Println("Usage: brain index rebuild")
		os.Exit(1)
	}
}

func cmdIndexRebuild() {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	idx, err := index.Rebuild(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rebuilding index: %v\n", err)
		os.Exit(1)
	}

	if err := idx.Save(brainDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Index rebuilt: %d entries across %d topics\n", len(idx.Entries), len(brain.AvailableTopics()))
}
