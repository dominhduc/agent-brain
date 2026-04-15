package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdClean(dryRun, fuzzy, patternsOnly, duplicatesOnly, decayOnly, rebuildOnly bool) {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	ranAnything := false

	if rebuildOnly || (!patternsOnly && !duplicatesOnly && !decayOnly && !rebuildOnly) {
		ranAnything = true
		cmdCleanRebuild(brainDir)
	}

	if patternsOnly || (!duplicatesOnly && !decayOnly && !rebuildOnly) {
		ranAnything = true
		cmdCleanPrune(brainDir, dryRun)
	}

	if duplicatesOnly || (!patternsOnly && !decayOnly && !rebuildOnly) {
		ranAnything = true
		cmdCleanDedup(brainDir, dryRun, fuzzy)
	}

	if decayOnly || (!patternsOnly && !duplicatesOnly && !rebuildOnly) {
		ranAnything = true
		cmdCleanSleep(brainDir, dryRun)
	}

	if !ranAnything {
		fmt.Println("Nothing to do.")
	}
}

func cmdCleanRebuild(brainDir string) {
	idx, err := knowledge.RebuildIndex(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rebuilding index: %v\n", err)
		os.Exit(1)
	}

	if err := idx.Save(brainDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Index rebuilt: %d entries across %d topics\n", len(idx.Entries), len(knowledge.AvailableTopics()))
}

func cmdCleanPrune(brainDir string, dryRun bool) {
	prunePath := filepath.Join(filepath.Dir(brainDir), ".brainprune")
	data, err := os.ReadFile(prunePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No .brainprune file found. Create one to enable pattern-based pruning:")
			fmt.Println(`    echo "# Stale patterns" > .brainprune`)
			fmt.Println(`    echo "v0." >> .brainprune`)
			fmt.Println("    Then run: brain clean --patterns")
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading .brainprune: %v\n", err)
		os.Exit(1)
	}

	patterns := parsePrunePatterns(string(data))
	if len(patterns) == 0 {
		fmt.Println("No prune patterns defined. Skipping prune.")
		return
	}

	topicFiles := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	totalRemoved := 0

	for _, topicFile := range topicFiles {
		filePath := filepath.Join(brainDir, topicFile)
		content, err := os.ReadFile(filePath)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", topicFile, err)
			}
			continue
		}

		lines := strings.Split(string(content), "\n")
		var kept, removed []string
		var currentEntry strings.Builder

		for _, line := range lines {
			if strings.HasPrefix(line, "### [") {
				if currentEntry.Len() > 0 {
					entryText := currentEntry.String()
					if shouldPrune(entryText, patterns) {
						removed = append(removed, strings.TrimSpace(entryText))
					} else {
						kept = append(kept, entryText)
					}
					currentEntry.Reset()
				}
				currentEntry.WriteString(line + "\n")
			} else {
				currentEntry.WriteString(line + "\n")
			}
		}

		if currentEntry.Len() > 0 {
			entryText := currentEntry.String()
			if shouldPrune(entryText, patterns) {
				removed = append(removed, strings.TrimSpace(entryText))
			} else {
				kept = append(kept, entryText)
			}
		}

		if len(removed) > 0 {
			if dryRun {
				fmt.Printf("  Would remove %d entries from %s:\n", len(removed), topicFile)
				for _, r := range removed {
					fmt.Printf("    - %s\n", truncate(r, 80))
				}
			} else {
				archivePath := filepath.Join(brainDir, "archived", fmt.Sprintf("%s.%s.md", strings.TrimSuffix(topicFile, ".md"), time.Now().Format("20060102")))
				if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "Error creating archive dir: %v\n", err)
					continue
				}
				if err := os.WriteFile(archivePath, []byte(strings.Join(removed, "\n")), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error archiving %s: %v\n", topicFile, err)
					continue
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(kept, "\n")), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", topicFile, err)
					continue
				}
				fmt.Printf("  Removed %d entries from %s (archived to %s)\n", len(removed), topicFile, filepath.Base(archivePath))
			}
			totalRemoved += len(removed)
		}
	}

	if totalRemoved == 0 && !dryRun {
		fmt.Println("  No entries matched prune patterns.")
	} else if totalRemoved > 0 && dryRun {
		fmt.Printf("  Would remove %d total entries\n", totalRemoved)
	} else if totalRemoved > 0 {
		fmt.Printf("  Removed %d total entries\n", totalRemoved)
	}
}

func cmdCleanDedup(brainDir string, dryRun, fuzzy bool) {
	var report *knowledge.DedupReport
	var err error

	if fuzzy {
		report, err = knowledge.RunFuzzyDedup(brainDir, dryRun, 0.55)
	} else {
		report, err = knowledge.RunDedup(brainDir, dryRun)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during dedup: %v\n", err)
		os.Exit(1)
	}

	if len(report.Groups) == 0 {
		fmt.Println("  No duplicates found.")
		return
	}

	if dryRun {
		fmt.Printf("  Would remove %d duplicate entries:\n", report.TotalRemoved)
		for _, group := range report.Groups {
			fmt.Printf("    - %s\n", truncate(group.Message, 80))
			for _, dup := range group.Duplicates {
				fmt.Printf("      %s\n", truncate(dup.Line, 80))
			}
		}
	} else {
		fmt.Printf("  Removed %d duplicate entries (kept %d)\n", report.TotalRemoved, len(report.Groups))
	}
}

func cmdCleanSleep(brainDir string, dryRun bool) {
	idx, err := knowledge.LoadIndex(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	var archivedEntries []string
	strengthThreshold := 0.05

	for key, entry := range idx.Entries {
		strength := knowledge.CalculateStrength(entry, now)
		if strength < strengthThreshold {
			archivedEntries = append(archivedEntries, key)
		}
	}

	if len(archivedEntries) == 0 {
		fmt.Println("  No decayed entries to archive.")
		return
	}

	if dryRun {
		fmt.Printf("  Would archive %d decayed entries:\n", len(archivedEntries))
		for _, key := range archivedEntries[:min(len(archivedEntries), 10)] {
			fmt.Printf("    - %s\n", key)
		}
		if len(archivedEntries) > 10 {
			fmt.Printf("    ... and %d more\n", len(archivedEntries)-10)
		}
		return
	}

	topicFiles := map[string]string{
		"MEMORY.md":      "memory",
		"gotchas.md":     "gotchas",
		"patterns.md":    "patterns",
		"decisions.md":   "decisions",
		"architecture.md": "architecture",
	}

	entryHeaderRe := regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

	for _, key := range archivedEntries {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		topic := parts[0]

		var topicFile string
		for f, t := range topicFiles {
			if t == topic {
				topicFile = f
				break
			}
		}
		if topicFile == "" {
			continue
		}

		filePath := filepath.Join(brainDir, topicFile)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		var kept []string
		var currentEntry []string
		inTargetEntry := false

		for _, line := range lines {
			if entryHeaderRe.MatchString(line) {
				if inTargetEntry {
					continue
				}
				if currentEntry != nil {
					kept = append(kept, currentEntry...)
				}
				currentEntry = []string{line}
				if len(parts) > 1 && strings.Contains(key, parts[1]) {
					inTargetEntry = true
					currentEntry = nil
				}
			} else {
				if inTargetEntry {
					continue
				}
				if currentEntry != nil {
					currentEntry = append(currentEntry, line)
				} else {
					kept = append(kept, line)
				}
			}
		}

		if currentEntry != nil {
			kept = append(kept, currentEntry...)
		}

		os.WriteFile(filePath, []byte(strings.Join(kept, "\n")), 0644)
		delete(idx.Entries, key)
	}

	if err := idx.Save(brainDir); err == nil {
		fmt.Printf("  Archived %d decayed entries\n", len(archivedEntries))
	}
}

func parsePrunePatterns(data string) []string {
	var patterns []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, strings.ToLower(line))
	}
	return patterns
}

func shouldPrune(entry string, patterns []string) bool {
	entryLower := strings.ToLower(entry)
	for _, pattern := range patterns {
		if strings.Contains(entryLower, pattern) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
