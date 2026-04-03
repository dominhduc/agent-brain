package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func cmdPrune(dryRun bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	prunePath := filepath.Join(filepath.Dir(brainDir), ".brainprune")
	data, err := os.ReadFile(prunePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No .brainprune file found. Nothing to prune.")
			fmt.Println("What to do: create a .brainprune file with patterns to match for removal.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading .brainprune: %v\nWhat to do: check file permissions.\n", err)
		os.Exit(1)
	}

	var activePatterns []string
	for _, p := range strings.Split(string(data), "\n") {
		p = strings.TrimSpace(p)
		if p != "" && !strings.HasPrefix(p, "#") {
			activePatterns = append(activePatterns, p)
		}
	}

	if len(activePatterns) == 0 {
		fmt.Println("No prune patterns defined in .brainprune. Nothing to prune.")
		return
	}

	topicFiles := []string{"gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	archivedDir := filepath.Join(brainDir, "archived")
	os.MkdirAll(archivedDir, 0755)

	var pruned []string

	for _, tf := range topicFiles {
		path := filepath.Join(brainDir, tf)
		topicData, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(topicData))
		var kept, removed []string
		for scanner.Scan() {
			line := scanner.Text()
			matched := false
			for _, pattern := range activePatterns {
				if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
					matched = true
					break
				}
			}
			if matched {
				removed = append(removed, line)
			} else {
				kept = append(kept, line)
			}
		}

		if len(removed) > 0 {
			pruned = append(pruned, fmt.Sprintf("%s: %d entries", tf, len(removed)))
			if !dryRun {
				if writeErr := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0600); writeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", tf, writeErr)
					continue
				}
				archivePath := filepath.Join(archivedDir, fmt.Sprintf("%s-%s.md", tf[:len(tf)-3], time.Now().Format("2006-01-02")))
				if archiveErr := os.WriteFile(archivePath, []byte(fmt.Sprintf("# Archived from %s — %s\n\n%s\n", tf, time.Now().Format("2006-01-02"), strings.Join(removed, "\n"))), 0600); archiveErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to archive to %s: %v\n", archivePath, archiveErr)
				}
			}
		}
	}

	if len(pruned) == 0 {
		fmt.Println("No entries matched prune patterns. Nothing to prune.")
		return
	}

	if dryRun {
		fmt.Println("Dry run — would prune:")
		for _, p := range pruned {
			fmt.Printf("  %s\n", p)
		}
	} else {
		fmt.Println("Pruned:")
		for _, p := range pruned {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println("\nArchived entries saved to .brain/archived/")
	}
}
