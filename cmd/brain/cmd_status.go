package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func cmdStatus(jsonFlag bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	topicFiles := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	var topicCount int
	var totalSize int64
	for _, f := range topicFiles {
		info, err := os.Stat(filepath.Join(brainDir, f))
		if err == nil {
			topicCount++
			totalSize += info.Size()
		}
	}

	sessionsDir := filepath.Join(brainDir, "sessions")
	sessionCount := 0
	if entries, err := os.ReadDir(sessionsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				sessionCount++
			}
		}
	}

	lineCount, _ := brain.MemoryLineCount()
	lineStatus := "OK"
	if lineCount > 200 {
		lineStatus = "OVER LIMIT"
	}

	queueDir := filepath.Join(brainDir, ".queue")
	queuePendingCount := 0
	doneCount := 0
	if entries, err := os.ReadDir(queueDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				queuePendingCount++
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(queueDir, "done")); err == nil {
		doneCount = len(entries)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	pendingCount := 0
	if entries, err := os.ReadDir(pendingDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				pendingCount++
			}
		}
	}

	if jsonFlag {
		status := map[string]interface{}{
			"memory_lines":    lineCount,
			"memory_status":   lineStatus,
			"topic_files":     topicCount,
			"session_files":   sessionCount,
			"total_size_kb":   totalSize / 1024,
			"queue_pending":   queuePendingCount,
			"queue_done":      doneCount,
			"pending_entries": pendingCount,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Knowledge Hub Status")
		fmt.Println("====================")
		limitHint := "OK"
		if lineCount > 200 {
			limitHint = "OVER LIMIT — run 'brain prune' or move entries to topic files"
		}
		fmt.Printf("MEMORY.md:       %d lines (%s)\n", lineCount, limitHint)
		fmt.Printf("Topic files:     %d files\n", topicCount)
		fmt.Printf("Session files:   %d sessions\n", sessionCount)
		fmt.Printf("Total size:      %d KB\n", totalSize/1024)
		fmt.Printf("Queue depth:     %d pending, %d done\n", queuePendingCount, doneCount)
		fmt.Printf("Pending entries: %d (run 'brain review' to approve)\n", pendingCount)
	}
}
