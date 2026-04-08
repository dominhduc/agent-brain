package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/analyzer"
	"github.com/dominhduc/agent-brain/internal/review"
	"github.com/dominhduc/agent-brain/internal/secrets"
)

const maxDiffSize = 2000 * 100

type QueueItem struct {
	Timestamp   string `json:"timestamp"`
	Repo        string `json:"repo"`
	DiffStat    string `json:"diff_stat"`
	Files       string `json:"files"`
	Attempts    int    `json:"attempts"`
	ErrorReason string `json:"error_reason,omitempty"`
}

type DiffGetter func(repo string) (string, error)

type AnalyzeFunc func(req analyzer.AnalyzeRequest) (analyzer.Finding, error)

func ProcessItemWithDeps(processingPath, queueDir, brainDir, projectRoot string, maxRetries int, getDiff DiffGetter, analyzeFn AnalyzeFunc) (bool, error) {
	data, err := os.ReadFile(processingPath)
	if err != nil {
		moveToFailed(processingPath, queueDir, fmt.Sprintf("reading queue item: %v", err))
		return false, fmt.Errorf("reading queue item: %w", err)
	}

	var item QueueItem
	if err := json.Unmarshal(data, &item); err != nil {
		moveToFailed(processingPath, queueDir, fmt.Sprintf("parsing queue item: %v", err))
		return false, fmt.Errorf("parsing queue item: %w", err)
	}

	if item.Timestamp == "" || item.Repo == "" {
		moveToFailed(processingPath, queueDir, "invalid queue item: missing timestamp or repo")
		return false, fmt.Errorf("invalid queue item: missing timestamp or repo")
	}

	if len(item.Timestamp) > 20 || len(item.Repo) > 4096 {
		moveToFailed(processingPath, queueDir, "invalid queue item: field too long")
		return false, fmt.Errorf("invalid queue item: field too long")
	}

	absRepo, _ := filepath.Abs(item.Repo)
	absRoot, _ := filepath.Abs(projectRoot)
	if absRepo != absRoot {
		moveToFailed(processingPath, queueDir, fmt.Sprintf("security: repo %q does not match project root %q", absRepo, absRoot))
		return false, fmt.Errorf("security: queue item repo %q does not match project root %q", absRepo, absRoot)
	}

	var diff string
	if getDiff != nil {
		diff, err = getDiff(item.Repo)
	} else {
		out, e := exec.Command("git", "-C", item.Repo, "diff", "HEAD~1").CombinedOutput()
		if e != nil {
			diff = ""
			err = e
		} else {
			diff = string(out)
		}
	}

	if err != nil || diff == "" {
		item.Attempts++
		if item.Attempts >= maxRetries {
			moveToFailed(processingPath, queueDir, fmt.Sprintf("getting diff failed after %d attempts: %v", item.Attempts, err))
			return false, fmt.Errorf("getting diff failed after %d attempts", item.Attempts)
		}
		saveAndRequeue(processingPath, itemPath(processingPath), item)
		return false, fmt.Errorf("getting diff: %w", err)
	}

	if len(diff) > maxDiffSize {
		diff = diff[:maxDiffSize]
	}

	if findings := secrets.ScanDiff(diff); len(findings) > 0 {
		flaggedDir := filepath.Join(queueDir, "flagged")
		os.MkdirAll(flaggedDir, 0755)
		os.Rename(processingPath, filepath.Join(flaggedDir, filepath.Base(processingPath)))
		return false, fmt.Errorf("secret detected in diff (type: %s)", findings[0].Type)
	}

	var finding analyzer.Finding
	if analyzeFn != nil {
		finding, err = analyzeFn(analyzer.AnalyzeRequest{Diff: diff})
	} else {
		return false, fmt.Errorf("no analyze function provided")
	}

	if err != nil {
		item.Attempts++
		if item.Attempts >= maxRetries {
			moveToFailed(processingPath, queueDir, fmt.Sprintf("LLM analysis failed after %d attempts: %v", item.Attempts, err))
			return false, fmt.Errorf("LLM analysis failed after %d attempts: %w", item.Attempts, err)
		}
		saveAndRequeue(processingPath, itemPath(processingPath), item)
		return false, fmt.Errorf("LLM analysis: %w", err)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entryCount := 0

	writePending := func(topic, content string, topics []string) {
		if content == "" {
			return
		}
		entryCount++
		entry := review.PendingEntry{
			ID:         fmt.Sprintf("%s-%s-%d", item.Timestamp, topic, entryCount),
			Topic:      topic,
			Content:    content,
			CommitSHA:  "",
			Timestamp:  time.Now(),
			Confidence: finding.Confidence,
			Source:     "daemon",
			Topics:     topics,
		}
		review.SavePendingEntry(pendingDir, entry)
	}

	for _, g := range finding.Gotchas {
		writePending("gotchas", g, finding.Topics)
	}
	for _, p := range finding.Patterns {
		writePending("patterns", p, finding.Topics)
	}
	for _, d := range finding.Decisions {
		writePending("decisions", d, finding.Topics)
	}
	for _, a := range finding.Architecture {
		writePending("architecture", a, finding.Topics)
	}

	moveToDone(processingPath, queueDir)
	return true, nil
}

func ParsePollInterval(input string) time.Duration {
	d, err := time.ParseDuration(input)
	if err != nil || d < time.Second {
		return 5 * time.Second
	}
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

func CalcBackoff(attempt int) time.Duration {
	return time.Duration(attempt*attempt) * 5 * time.Second
}

func RecoverStaleProcessing(brainDir string) {
	if brainDir == "" {
		return
	}
	queueDir := filepath.Join(brainDir, ".queue")
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		return
	}
	recovered := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".processing") {
			oldPath := filepath.Join(queueDir, e.Name())
			newName := strings.TrimSuffix(e.Name(), ".processing")
			newPath := filepath.Join(queueDir, newName)
			if err := os.Rename(oldPath, newPath); err == nil {
				recovered++
			}
		}
	}
	if recovered > 0 {
		fmt.Printf("Recovered %d stale processing item(s)\n", recovered)
	}
}

func moveToFailed(itemPath, queueDir string, reason string) {
	failedDir := filepath.Join(queueDir, "failed")
	os.MkdirAll(failedDir, 0755)

	data, err := os.ReadFile(itemPath)
	if err == nil {
		var item QueueItem
		if json.Unmarshal(data, &item) == nil {
			item.ErrorReason = reason
			itemData, _ := json.Marshal(item)
			os.WriteFile(itemPath, itemData, 0600)
		}
	}

	destName := strings.TrimSuffix(filepath.Base(itemPath), ".processing")
	destPath := filepath.Join(failedDir, destName)
	os.Rename(itemPath, destPath)
}

func moveToDone(itemPath, queueDir string) {
	doneDir := filepath.Join(queueDir, "done")
	os.MkdirAll(doneDir, 0755)
	os.Rename(itemPath, filepath.Join(doneDir, filepath.Base(itemPath)))
}

func saveAndRequeue(processingPath, originalPath string, item QueueItem) {
	itemData, _ := json.Marshal(item)
	os.WriteFile(processingPath, itemData, 0600)
	os.Rename(processingPath, originalPath)
}

func itemPath(processingPath string) string {
	return strings.TrimSuffix(processingPath, ".processing")
}
