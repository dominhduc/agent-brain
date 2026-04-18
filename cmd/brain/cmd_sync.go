package main

import (
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdSync() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain sync [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --push       Propose pushing unique project entries to global store")
		fmt.Println("  --apply      Actually push to global (use with --push)")
		fmt.Println()
		fmt.Println("What to do: run 'brain sync' to pull relevant global entries,")
		fmt.Println("or 'brain sync --push' to propose pushing project entries to global.")
		os.Exit(1)
	}

	pushFlag := hasFlag("--push")
	applyFlag := hasFlag("--apply")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	if pushFlag {
		syncPush(brainDir, applyFlag)
	} else {
		syncPull(brainDir)
	}
}

func syncPull(brainDir string) {
	globalEntries, err := knowledge.LoadGlobalEntriesForMerge()
	if err != nil {
		fmt.Println("No global knowledge store found.")
		return
	}

	if len(globalEntries) == 0 {
		fmt.Println("Global knowledge store is empty.")
		return
	}

	fmt.Printf("Global store has %d entries.\n", len(globalEntries))
	fmt.Println("Run 'brain get all' to see merged project + global knowledge.")
}

func syncPush(brainDir string, applyFlag bool) {
	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	idx, err := hub.LoadIndex()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	type candidate struct {
		topic   string
		timestamp string
		message string
	}
	var candidates []candidate
	for _, topic := range knowledge.AvailableTopics() {
		entries, err := knowledge.GetTopicEntriesForDir(topic, brainDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if ie, found := idx.Get(topic, e.Timestamp); found {
				if ie.Strength > 0.8 && ie.RetrievalCount > 5 {
					candidates = append(candidates, candidate{topic, e.Timestamp, e.Message})
				}
			}
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No entries qualify for promotion (need strength > 0.8 and retrieval count > 5).")
		return
	}

	fmt.Printf("Found %d entries that may be worth sharing globally:\n\n", len(candidates))
	for i, c := range candidates {
		fmt.Printf("%d. [%s] %s\n", i+1, c.timestamp, c.message)
	}

	if !applyFlag {
		fmt.Println("\nRun 'brain sync --push --apply' to actually push these entries.")
	} else {
		pushed := 0
		for _, c := range candidates {
			if _, err := knowledge.AddGlobalEntry(c.topic, c.message); err != nil {
				continue
			}
			pushed++
		}
		fmt.Printf("\nPushed %d entries to global store.\n", pushed)
	}
}
