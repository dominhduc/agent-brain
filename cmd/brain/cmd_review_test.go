package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/review"
)

func TestCmdReview_NoPendingEntries(t *testing.T) {
	setupTestProject(t)

	oldArgs := os.Args
	os.Args = []string{"brain", "review"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdReview(false) })

	if output == "" {
		t.Log("cmdReview returned empty output (expected for no pending entries)")
	}
}

func TestCmdReview_WithPendingEntries(t *testing.T) {
	tmpDir := setupTestProject(t)

	cfg := config.DefaultConfig()
	cfg.Review.Profile = "agent"
	config.Save(cfg)

	entry := review.PendingEntry{
		ID:         "test-review-001",
		Topic:      "gotchas",
		Content:    "Test gotcha for review",
		CommitSHA:  "abc123",
		Timestamp:  time.Now(),
		Confidence: "HIGH",
		Source:     "daemon",
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(filepath.Join(tmpDir, ".brain", "pending", entry.ID+".json"), data, 0600)

	oldArgs := os.Args
	os.Args = []string{"brain", "review"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdReview(false) })

	if output == "" {
		t.Fatal("expected output from cmdReview with pending entries")
	}
}
