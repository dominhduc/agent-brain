package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/index"
)

var entryHeaderRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

func cmdSleep(dryRun bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	idx, err := index.Load(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading index: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	const threshold = 0.05

	var archived, decayed int
	var toRemove []string

	for key, entry := range idx.Entries {
		strength := index.CalculateStrength(entry, now)
		if strength < threshold {
			toRemove = append(toRemove, key)
			decayed++
		}
	}

	if dryRun {
		if decayed == 0 {
			fmt.Println("No entries below decay threshold. Knowledge base is healthy.")
			return
		}
		fmt.Printf("Dry run — would archive %d entries below strength threshold (%.2f):\n\n", decayed, threshold)
		for _, key := range toRemove {
			fmt.Printf("  - %s\n", key)
		}
		return
	}

	archivedDir := filepath.Join(brainDir, "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archived dir: %v\n", err)
		os.Exit(1)
	}

	for _, key := range toRemove {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		topic := parts[0]
		timestamp := parts[1]

		entryText := extractEntryText(brainDir, topic, timestamp)
		if entryText != "" {
			archiveFile := filepath.Join(archivedDir, topic+"-"+strings.ReplaceAll(timestamp, ":", "-")+".md")
			os.WriteFile(archiveFile, []byte(entryText+"\n"), 0600)
			archived++
		}

		delete(idx.Entries, key)
	}

	if err := removeArchivedFromTopicFiles(brainDir, toRemove); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not remove archived entries from topic files: %v\n", err)
	}

	idx.LastRebuild = now
	if err := idx.Save(brainDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sleep complete: %d entries decayed, %d archived\n", decayed, archived)
}

func extractEntryText(brainDir, topic, timestamp string) string {
	path, err := brain.TopicFilePathForDir(topic, brainDir)
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		matches := entryHeaderRe.FindStringSubmatch(line)
		if matches != nil && matches[1] == timestamp {
			return line
		}
	}
	return ""
}

func removeArchivedFromTopicFiles(brainDir string, keys []string) error {
	topicEntries := make(map[string][]string)
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			topicEntries[parts[0]] = append(topicEntries[parts[0]], parts[1])
		}
	}

	for topic, timestamps := range topicEntries {
		path, err := brain.TopicFilePathForDir(topic, brainDir)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		tsMap := make(map[string]bool)
		for _, ts := range timestamps {
			tsMap[ts] = true
		}

		var kept []string
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			matches := entryHeaderRe.FindStringSubmatch(line)
			if matches != nil && tsMap[matches[1]] {
				continue
			}
			kept = append(kept, line)
		}

		os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0600)
	}

	return nil
}
