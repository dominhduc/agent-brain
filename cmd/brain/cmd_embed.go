package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/embed"
	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdEmbed() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain embed [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --all      Embed all entries (re-embed everything)")
		fmt.Println("  --status   Show embedding coverage status")
		fmt.Println()
		fmt.Println("What to do: run 'brain embed' to embed new/stale entries.")
		os.Exit(1)
	}

	statusFlag := hasFlag("--status")
	_ = hasFlag("--all")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	if statusFlag {
		store, err := embed.NewStore(brainDir)
		if err != nil {
			fmt.Printf("Embedding not configured. Run 'brain config set embedding.provider ollama' to enable.\n")
			return
		}
		fmt.Printf("Vector store: %s\n", store.Dir())
		fmt.Println("Embedding: not configured (set embedding.provider to enable)")
		return
	}

	provider := &embed.NoneProvider{}

	entries, err := loadAllEntriesForEmbed(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading entries: %v\n", err)
		os.Exit(1)
	}

	store, err := embed.NewStore(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating vector store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Embedding %d entries...\n", len(entries))
	start := time.Now()

	if err := store.IndexEntries(entries, provider); err != nil {
		fmt.Fprintf(os.Stderr, "Error embedding entries: %v\n", err)
		fmt.Fprintln(os.Stderr, "What to do: configure an embedding provider (ollama or openai)")
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Printf("Embedded %d entries in %s.\n", len(entries), elapsed.Round(time.Millisecond))
}

func loadAllEntriesForEmbed(brainDir string) ([]embed.IndexEntry, error) {
	var entries []embed.IndexEntry
	for _, topic := range knowledge.AvailableTopics() {
		topicEntries, err := knowledge.GetTopicEntriesForDir(topic, brainDir)
		if err != nil {
			continue
		}
		for _, e := range topicEntries {
			entries = append(entries, embed.IndexEntry{
				Key:       knowledge.MakeKey(topic, e.Timestamp),
				Topic:     topic,
				Message:   e.Message,
				Timestamp: e.Timestamp,
			})
		}
	}
	return entries, nil
}
