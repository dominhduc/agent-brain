package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/otel"
	"github.com/dominhduc/agent-brain/internal/profile"
	"github.com/dominhduc/agent-brain/internal/tui"
	"go.opentelemetry.io/otel/trace"
)

func cmdReview(allFlag, yesFlag, ttyFlag bool) {
	ctx := context.Background()
	ctx, span := otel.StartSpan(ctx, "brain.review")
	defer otel.EndSpan(span, nil)

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entries, err := knowledge.LoadPendingEntries(pendingDir)
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
			count, err := knowledge.TopicEntriesToPending(topic, path, pendingDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not import from %s: %v\n", path, err)
				continue
			}
			totalImported += count
		}
		if totalImported > 0 {
			fmt.Printf("Imported %d existing entries into pending queue.\n", totalImported)
		}
		entries, _ = knowledge.LoadPendingEntries(pendingDir)
	}

	if len(entries) == 0 {
		fmt.Println("No pending entries to knowledge.")
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

	var accepted []knowledge.PendingEntry
	var rejectedIDs []string
	var mode string

	if useTUI {
		mode = "tui"
		accepted, rejectedIDs, err = tui.RunReview(entries, prof.Name, os.Stdout)
		if err != nil {
			if accepted == nil && rejectedIDs == nil {
				fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered knowledge.\n\n")
				mode = "line_buffered"
				accepted, rejectedIDs = doLineBufferedReview(ctx, span, entries, prof, pendingDir)
			} else {
				fmt.Fprintf(os.Stderr, "Interactive review failed: %v\nFalling back to auto-accept.\n", err)
				mode = "auto"
				doAutoAccept(ctx, span, entries, prof, pendingDir, true)
				return
			}
		} else if accepted == nil && rejectedIDs == nil {
			fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered knowledge.\n\n")
			mode = "line_buffered"
			accepted, rejectedIDs = doLineBufferedReview(ctx, span, entries, prof, pendingDir)
		} else {
			writeAccepted(ctx, span, accepted, pendingDir)
			removeEntries(accepted, rejectedIDs, pendingDir)
			fmt.Printf("\nApplied %d entries, rejected %d.\n", len(accepted), len(rejectedIDs))
		}
	} else if !prof.AutoAccept && !yesFlag {
		mode = "line_buffered"
		accepted, rejectedIDs = doLineBufferedReview(ctx, span, entries, prof, pendingDir)
	} else {
		mode = "auto"
		doAutoAccept(ctx, span, entries, prof, pendingDir, yesFlag || !canUseTTY)
		return
	}

	otel.SetAttributes(span,
		otel.BrainProfile.String(prof.Name),
		otel.BrainReviewMode.String(mode),
		otel.BrainReviewTotal.Int(len(entries)),
		otel.BrainReviewAccepted.Int(len(accepted)),
		otel.BrainReviewRejected.Int(len(rejectedIDs)),
		otel.BrainDurationMs.Int64(0),
	)
}

func doAutoAccept(ctx context.Context, span trace.Span, entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string, nonInteractive bool) {
	accepted, rejectedIDs := autoAcceptEntries(entries, prof)
	writeAccepted(ctx, span, accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	msg := "Auto-accepted %d entries (auto-dedup: %v)\n"
	if nonInteractive {
		msg = "Auto-accepted %d entries in non-interactive mode (auto-dedup: %v)\n"
	}
	fmt.Printf(msg, len(accepted), prof.AutoDedup)
}

func doInteractiveReview(entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string) {
	accepted, rejectedIDs, err := tui.RunReview(entries, prof.Name, os.Stdout)
	if err != nil {
		if accepted == nil && rejectedIDs == nil {
			fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered knowledge.\n\n")
			doLineBufferedReviewFallback(entries, prof, pendingDir)
			return
		}
		fmt.Fprintf(os.Stderr, "Interactive review failed: %v\nFalling back to auto-accept.\n", err)
		doAutoAcceptFallback(entries, prof, pendingDir, true)
		return
	}

	if accepted == nil && rejectedIDs == nil {
		fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered knowledge.\n\n")
		doLineBufferedReviewFallback(entries, prof, pendingDir)
		return
	}

	writeAcceptedSimple(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)
	fmt.Printf("\nApplied %d entries, rejected %d.\n", len(accepted), len(rejectedIDs))
}

func doLineBufferedReview(ctx context.Context, span trace.Span, entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string) ([]knowledge.PendingEntry, []string) {
	reader := bufio.NewReader(os.Stdin)

	var accepted []knowledge.PendingEntry
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

		decision := "accepted"
		switch input {
		case "y":
			accepted = append(accepted, e)
		case "n":
			rejectedIDs = append(rejectedIDs, e.ID)
			decision = "rejected"
		case "a":
			remaining := entries[i:]
			for _, rem := range remaining {
				accepted = append(accepted, rem)
			}
			rejectedIDs = append(rejectedIDs, collectIDs(entries[i+1:])...)
			decision = "accept_all"
			goto done
		case "q":
			rejectedIDs = append(rejectedIDs, collectIDs(entries[i+1:])...)
			decision = "quit"
			goto done
		default:
			accepted = append(accepted, e)
		}

		otel.RecordEvent(span, "brain.knowledge.entry",
			otel.BrainEntryID.String(e.ID),
			otel.BrainEntryTopic.String(e.Topic),
			otel.BrainEntryDecision.String(decision),
		)
	}

done:
	writeAccepted(ctx, span, accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	acceptedCount := len(accepted)
	rejectedCount := len(rejectedIDs)

	if rejectedCount > 0 {
		fmt.Printf("\nAccepted %d entries, rejected %d.\n", acceptedCount, rejectedCount)
	} else {
		fmt.Printf("\nAccepted %d entries.\n", acceptedCount)
	}

	return accepted, rejectedIDs
}

func doLineBufferedReviewFallback(entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string) {
	reader := bufio.NewReader(os.Stdin)

	var accepted []knowledge.PendingEntry
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
	writeAcceptedSimple(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	acceptedCount := len(accepted)
	rejectedCount := len(rejectedIDs)

	if rejectedCount > 0 {
		fmt.Printf("\nAccepted %d entries, rejected %d.\n", acceptedCount, rejectedCount)
	} else {
		fmt.Printf("\nAccepted %d entries.\n", acceptedCount)
	}
}

func doAutoAcceptFallback(entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string, nonInteractive bool) {
	accepted, rejectedIDs := autoAcceptEntries(entries, prof)
	writeAcceptedSimple(accepted, pendingDir)
	removeEntries(accepted, rejectedIDs, pendingDir)

	msg := "Auto-accepted %d entries (auto-dedup: %v)\n"
	if nonInteractive {
		msg = "Auto-accepted %d entries in non-interactive mode (auto-dedup: %v)\n"
	}
	fmt.Printf(msg, len(accepted), prof.AutoDedup)
}

func collectIDs(entries []knowledge.PendingEntry) []string {
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

func autoAcceptEntries(entries []knowledge.PendingEntry, prof profile.Profile) ([]knowledge.PendingEntry, []string) {
	accepted := entries
	var rejectedIDs []string
	if prof.AutoDedup {
		seen := make(map[string]bool)
		var unique []knowledge.PendingEntry
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

func writeAccepted(ctx context.Context, span trace.Span, accepted []knowledge.PendingEntry, pendingDir string) {
	ctx, writeSpan := otel.StartSpan(ctx, "brain.knowledge.write")
	defer otel.EndSpan(writeSpan, nil)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	brainDir, _ := knowledge.FindBrainDir()
	idx, _ := knowledge.LoadIndex(brainDir)
	now := time.Now()

	for _, e := range accepted {
		path, err := knowledge.TopicFilePath(e.Topic)
		if err != nil {
			continue
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			continue
		}
		fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, e.Content)
		f.Close()

		idx.Set(e.Topic, timestamp, knowledge.IndexEntry{
			Strength:       1.0,
			RetrievalCount: 0,
			LastRetrieved:  now,
			HalfLifeDays:   7,
			Confidence:     e.Confidence,
			Topics:         e.Topics,
		})

		otel.RecordEvent(writeSpan, "brain.knowledge.entry_written",
			otel.BrainEntryID.String(e.ID),
			otel.BrainEntryTopic.String(e.Topic),
		)
	}
	idx.Save(brainDir)

	otel.SetAttributes(writeSpan,
		otel.BrainReviewAccepted.Int(len(accepted)),
	)
}

func writeAcceptedSimple(accepted []knowledge.PendingEntry, pendingDir string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	brainDir, _ := knowledge.FindBrainDir()
	idx, _ := knowledge.LoadIndex(brainDir)
	now := time.Now()

	for _, e := range accepted {
		path, err := knowledge.TopicFilePath(e.Topic)
		if err != nil {
			continue
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			continue
		}
		fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, e.Content)
		f.Close()

		idx.Set(e.Topic, timestamp, knowledge.IndexEntry{
			Strength:       1.0,
			RetrievalCount: 0,
			LastRetrieved:  now,
			HalfLifeDays:   7,
			Confidence:     e.Confidence,
			Topics:         e.Topics,
		})
	}
	idx.Save(brainDir)
}

func removeEntries(accepted []knowledge.PendingEntry, rejectedIDs []string, pendingDir string) {
	for _, id := range rejectedIDs {
		knowledge.RemovePendingEntry(pendingDir, id)
	}
	for _, e := range accepted {
		knowledge.RemovePendingEntry(pendingDir, e.ID)
	}
}
