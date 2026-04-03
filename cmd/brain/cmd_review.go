package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/profile"
	"github.com/dominhduc/agent-brain/internal/review"
	"github.com/dominhduc/agent-brain/internal/tui"
)

func cmdReview(allFlag bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entries, err := review.LoadPendingEntries(pendingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading pending entries: %v\n", err)
		os.Exit(1)
	}

	if allFlag {
		topicFiles := map[string]string{
			"gotchas":      filepath.Join(brainDir, "gotchas.md"),
			"patterns":     filepath.Join(brainDir, "patterns.md"),
			"decisions":    filepath.Join(brainDir, "decisions.md"),
			"architecture": filepath.Join(brainDir, "architecture.md"),
		}
		totalImported := 0
		for topic, path := range topicFiles {
			count, err := review.TopicEntriesToPending(topic, path, pendingDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not import from %s: %v\n", path, err)
				continue
			}
			totalImported += count
		}
		if totalImported > 0 {
			fmt.Printf("Imported %d existing entries into pending queue.\n", totalImported)
		}
		entries, _ = review.LoadPendingEntries(pendingDir)
	}

	if len(entries) == 0 {
		fmt.Println("No pending entries to review.")
		fmt.Println("What to do: push some commits and let the daemon analyze them.")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	prof, err := profile.FromName(cfg.Review.Profile)
	if err != nil {
		prof = profile.DefaultProfile()
	}

	fmt.Printf("Reviewing %d pending entries (profile: %s)\n", len(entries), prof.Name)
	fmt.Println()

	if prof.AutoAccept {
		accepted := entries
		var rejectedIDs []string
		if prof.AutoDedup {
			seen := make(map[string]bool)
			var unique []review.PendingEntry
			for _, e := range entries {
				fp := e.Fingerprint()
				if seen[fp] {
					rejectedIDs = append(rejectedIDs, e.ID)
					continue
				}
				seen[fp] = true
				unique = append(unique, e)
			}
			accepted = unique
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		for _, e := range accepted {
			path, err := brain.TopicFilePath(e.Topic)
			if err != nil {
				continue
			}
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				continue
			}
			fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, e.Content)
			f.Close()
		}
		for _, e := range accepted {
			review.RemovePendingEntry(pendingDir, e.ID)
		}
		for _, id := range rejectedIDs {
			review.RemovePendingEntry(pendingDir, id)
		}
		fmt.Printf("Auto-accepted %d entries (profile: agent, auto-dedup: %v)\n", len(accepted), prof.AutoDedup)
		return
	}

	accepted, rejectedIDs, err := tui.RunReview(entries, prof.Name, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in review UI: %v\n", err)
		os.Exit(1)
	}

	if accepted == nil && rejectedIDs == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	for _, e := range accepted {
		path, err := brain.TopicFilePath(e.Topic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not find topic file for %s: %v\n", e.Topic, err)
			continue
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open %s: %v\n", path, err)
			continue
		}
		fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, e.Content)
		f.Close()
	}

	for _, id := range rejectedIDs {
		review.RemovePendingEntry(pendingDir, id)
	}

	for _, e := range accepted {
		review.RemovePendingEntry(pendingDir, e.ID)
	}

	fmt.Printf("\nApplied %d entries, rejected %d.\n", len(accepted), len(rejectedIDs))
}
