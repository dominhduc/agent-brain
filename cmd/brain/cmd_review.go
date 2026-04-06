package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/profile"
	"github.com/dominhduc/agent-brain/internal/review"
	"github.com/dominhduc/agent-brain/internal/tui"
)

func cmdReview(allFlag, yesFlag, ttyFlag bool) {
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

	fmt.Printf("Reviewing %d pending entries (profile: %s)\n\n", len(entries), prof.Name)

	canUseTTY := tui.CanUseRawMode()
	useTUI := !prof.AutoAccept && !yesFlag && !ttyFlag && canUseTTY

	if useTUI {
		doInteractiveReview(entries, prof, pendingDir)
	} else if !prof.AutoAccept && !yesFlag {
		doLineBufferedReview(entries, prof, pendingDir)
	} else {
		doAutoAccept(entries, prof, pendingDir, yesFlag || !canUseTTY)
	}
}

func doAutoAccept(entries []review.PendingEntry, prof profile.Profile, pendingDir string, nonInteractive bool) {
	accepted, rejectedIDs := autoAcceptEntries(entries, prof)
	writeAccepted(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	msg := "Auto-accepted %d entries (auto-dedup: %v)\n"
	if nonInteractive {
		msg = "Auto-accepted %d entries in non-interactive mode (auto-dedup: %v)\n"
	}
	fmt.Printf(msg, len(accepted), prof.AutoDedup)
}

func doInteractiveReview(entries []review.PendingEntry, prof profile.Profile, pendingDir string) {
	accepted, rejectedIDs, err := tui.RunReview(entries, prof.Name, os.Stdout)
	if err != nil {
		if accepted == nil && rejectedIDs == nil {
			fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered review.\n\n")
			doLineBufferedReview(entries, prof, pendingDir)
			return
		}
		fmt.Fprintf(os.Stderr, "Interactive review failed: %v\nFalling back to auto-accept.\n", err)
		doAutoAccept(entries, prof, pendingDir, true)
		return
	}

	if accepted == nil && rejectedIDs == nil {
		fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered review.\n\n")
		doLineBufferedReview(entries, prof, pendingDir)
		return
	}

	writeAccepted(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)
	fmt.Printf("\nApplied %d entries, rejected %d.\n", len(accepted), len(rejectedIDs))
}

func doLineBufferedReview(entries []review.PendingEntry, prof profile.Profile, pendingDir string) {
	reader := bufio.NewReader(os.Stdin)

	var accepted []review.PendingEntry
	var rejectedIDs []string

	for i, e := range entries {
		fmt.Printf("Entry %d/%d [%s]\n", i+1, len(entries), e.Topic)
		fmt.Printf("  %s\n", truncateForPrompt(e.Content, 60))
		fmt.Print("Accept? (y/n/q/a): ")

		input, err := reader.ReadString('\n')
		isEOF := err != nil && strings.Contains(err.Error(), "EOF")

		if err != nil && !isEOF {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			input = "y"
		}

		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" || isEOF {
			input = "y"
		}

		switch input {
		case "y":
			accepted = append(accepted, e)
		case "n":
			rejectedIDs = append(rejectedIDs, e.ID)
		case "a":
			remaining := entries[i:]
			for _, rem := range remaining {
				accepted = append(accepted, rem)
			}
			rejectedIDs = append(rejectedIDs, collectIDs(entries[i+1:])...)
			goto done
		case "q":
			rejectedIDs = append(rejectedIDs, collectIDs(entries[i+1:])...)
			goto done
		default:
			accepted = append(accepted, e)
		}
	}

done:
	writeAccepted(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	acceptedCount := len(accepted)
	rejectedCount := len(rejectedIDs)

	if rejectedCount > 0 {
		fmt.Printf("\nAccepted %d entries, rejected %d.\n", acceptedCount, rejectedCount)
	} else {
		fmt.Printf("\nAccepted %d entries.\n", acceptedCount)
	}
}

func collectIDs(entries []review.PendingEntry) []string {
	var ids []string
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return ids
}

func truncateForPrompt(s string, maxLen int) string {
	lines := strings.Split(s, "\n")
	firstLine := lines[0]
	if len(firstLine) > maxLen {
		return firstLine[:maxLen-3] + "..."
	}
	return firstLine
}

func autoAcceptEntries(entries []review.PendingEntry, prof profile.Profile) ([]review.PendingEntry, []string) {
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
	return accepted, rejectedIDs
}

func writeAccepted(accepted []review.PendingEntry, pendingDir string) {
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
}

func removeEntries(accepted []review.PendingEntry, rejectedIDs []string, pendingDir string) {
	for _, id := range rejectedIDs {
		review.RemovePendingEntry(pendingDir, id)
	}
	for _, e := range accepted {
		review.RemovePendingEntry(pendingDir, e.ID)
	}
}
