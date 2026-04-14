package main

import (
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdOutcome() {
	fmt.Fprintln(os.Stderr, "Warning: 'brain outcome' is deprecated. Use 'brain eval --good/--bad' instead.")

	good := hasFlag("--good")
	bad := hasFlag("--bad")

	if !good && !bad {
		fmt.Println("Usage: brain outcome --good    (memories helped)")
		fmt.Println("       brain outcome --bad     (memories were irrelevant)")
		os.Exit(1)
	}

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	keys, err := knowledge.LoadRetrievals(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading session: %v\n", err)
		os.Exit(1)
	}

	if len(keys) == 0 {
		fmt.Println("No retrieved entries in current session.")
		return
	}

	idx, err := knowledge.LoadIndex(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	var adjusted int
	for _, key := range keys {
		entry, ok := idx.GetByRawKey(key)
		if !ok {
			continue
		}
		if good {
			entry.HalfLifeDays += 5
		} else if bad {
			entry.HalfLifeDays = max(1, entry.HalfLifeDays-3)
		}
		idx.SetByRawKey(key, entry)
		adjusted++
	}

	if err := idx.Save(brainDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	knowledge.ClearRetrievals(brainDir)

	fmt.Printf("Applied %s outcome to %d entries\n", map[bool]string{true: "positive", false: "negative"}[good], adjusted)
}
