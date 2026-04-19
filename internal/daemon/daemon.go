package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/otel"
)

const maxDiffSize = 50000

var internalHostRe = regexp.MustCompile(`(?m)(?:https?://|ssh://|git@)([a-zA-Z0-9._-]+)`)

type QueueItem struct {
	Timestamp   string `json:"timestamp"`
	Repo        string `json:"repo"`
	DiffStat    string `json:"diff_stat"`
	Files       string `json:"files"`
	Attempts    int    `json:"attempts"`
	ErrorReason string `json:"error_reason,omitempty"`
}

type DiffGetter func(repo string) (string, error)

type AnalyzeFunc func(req AnalyzeRequest) (Finding, error)

func ProcessItemWithDeps(ctx context.Context, processingPath, queueDir, brainDir, projectRoot string, maxRetries int, getDiff DiffGetter, analyzeFn AnalyzeFunc) (bool, error) {
	ctx, processSpan := otel.StartSpan(ctx, "brain.daemon.process")
	defer otel.EndSpan(processSpan, nil)

	data, err := os.ReadFile(processingPath)
	if err != nil {
		otel.SetAttributes(processSpan, otel.BrainDaemonItem.String(processingPath), otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, fmt.Sprintf("reading queue item: %v", err))
		return false, fmt.Errorf("reading queue item: %w", err)
	}

	var item QueueItem
	if err := json.Unmarshal(data, &item); err != nil {
		otel.SetAttributes(processSpan, otel.BrainDaemonItem.String(processingPath), otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, fmt.Sprintf("parsing queue item: %v", err))
		return false, fmt.Errorf("parsing queue item: %w", err)
	}

	otel.SetAttributes(processSpan, otel.BrainDaemonItem.String(processingPath), otel.BrainDaemonRepo.String(item.Repo), otel.BrainDaemonAttempt.Int(item.Attempts))

	if item.Timestamp == "" || item.Repo == "" {
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, "invalid queue item: missing timestamp or repo")
		return false, fmt.Errorf("invalid queue item: missing timestamp or repo")
	}

	if len(item.Timestamp) > 20 || len(item.Repo) > 4096 {
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, "invalid queue item: field too long")
		return false, fmt.Errorf("invalid queue item: field too long")
	}

	absRepo, err := filepath.Abs(item.Repo)
	if err != nil {
		moveToFailed(processingPath, queueDir, fmt.Sprintf("invalid repo path: %v", err))
		return false, fmt.Errorf("invalid repo path: %w", err)
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		moveToFailed(processingPath, queueDir, fmt.Sprintf("invalid root path: %v", err))
		return false, fmt.Errorf("invalid root path: %w", err)
	}
	evalRepo, _ := filepath.EvalSymlinks(absRepo)
	evalRoot, _ := filepath.EvalSymlinks(absRoot)
	if evalRepo != "" && evalRoot != "" && evalRepo != evalRoot {
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, fmt.Sprintf("security: repo %q does not match project root %q (eval symlinks)", evalRepo, evalRoot))
		return false, fmt.Errorf("security: queue item repo %q does not match project root %q", evalRepo, evalRoot)
	} else if absRepo != absRoot {
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("error"))
		moveToFailed(processingPath, queueDir, fmt.Sprintf("security: repo %q does not match project root %q", absRepo, absRoot))
		return false, fmt.Errorf("security: queue item repo %q does not match project root %q", absRepo, absRoot)
	}

	var diff string
	_, diffSpan := otel.StartSpan(ctx, "brain.daemon.diff")
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
	otel.SetAttributes(diffSpan, otel.BrainDiffSize.Int(len(diff)), otel.BrainDiffFiles.Int(strings.Count(diff, "diff --git")))
	otel.EndSpan(diffSpan, err)

	if err != nil || diff == "" {
		item.Attempts++
		outcome := "retry"
		if item.Attempts >= maxRetries {
			outcome = "fail"
			moveToFailed(processingPath, queueDir, fmt.Sprintf("getting diff failed after %d attempts: %v", item.Attempts, err))
		} else {
			saveAndRequeue(processingPath, itemPath(processingPath), item)
		}
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String(outcome))
		return false, fmt.Errorf("getting diff: %w", err)
	}

	if len(diff) > maxDiffSize {
		diff = diff[:maxDiffSize]
	}

	diff = internalHostRe.ReplaceAllStringFunc(diff, func(match string) string {
		if looksInternal(match) {
			return "[REDACTED_HOST]"
		}
		return match
	})

	_, secretsSpan := otel.StartSpan(ctx, "brain.daemon.secrets")
	if findings := ScanDiffSecrets(diff); len(findings) > 0 {
		secretTypes := make([]string, len(findings))
		for i, f := range findings {
			secretTypes[i] = f.Type
		}
		otel.SetAttributes(secretsSpan, otel.BrainSecretsFound.Int(len(findings)), otel.BrainSecretsTypes.StringSlice(secretTypes))
		otel.EndSpan(secretsSpan, nil)
		flaggedDir := filepath.Join(queueDir, "flagged")
		os.MkdirAll(flaggedDir, 0755)
		os.Rename(processingPath, filepath.Join(flaggedDir, filepath.Base(processingPath)))
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("fail"))
		return false, fmt.Errorf("secret detected in diff (type: %s)", findings[0].Type)
	}
	otel.SetAttributes(secretsSpan, otel.BrainSecretsFound.Int(0))
	otel.EndSpan(secretsSpan, nil)

	_, analyzeSpan := otel.StartSpan(ctx, "brain.analyze")
	var finding Finding
	if analyzeFn != nil {
	analyzeStart := time.Now()
		finding, err = analyzeFn(AnalyzeRequest{Diff: diff})
		latencyMs := time.Since(analyzeStart).Milliseconds()
		otel.SetAttributes(analyzeSpan, otel.BrainLLMLatencyMs.Int64(latencyMs), otel.BrainLLMConfidence.String(finding.Confidence), otel.BrainFindingsGotchas.Int(len(finding.Gotchas)), otel.BrainFindingsPatterns.Int(len(finding.Patterns)), otel.BrainFindingsDecisions.Int(len(finding.Decisions)), otel.BrainFindingsArchitecture.Int(len(finding.Architecture)))
	} else {
		err = fmt.Errorf("no analyze function provided")
	}
	otel.EndSpan(analyzeSpan, err)

	if err != nil {
		item.Attempts++
		outcome := "retry"
		if item.Attempts >= maxRetries {
			outcome = "fail"
			moveToFailed(processingPath, queueDir, fmt.Sprintf("LLM analysis failed after %d attempts: %v", item.Attempts, err))
		} else {
			saveAndRequeue(processingPath, itemPath(processingPath), item)
		}
		otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String(outcome))
		return false, fmt.Errorf("LLM analysis: %w", err)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entryCount := 0

	writePending := func(topic, content string, topics []string) {
		if content == "" {
			return
		}
		entryCount++
		entry := knowledge.PendingEntry{
			ID:         fmt.Sprintf("%s-%s-%d", item.Timestamp, topic, entryCount),
			Topic:      topic,
			Content:    content,
			CommitSHA:  "",
			Timestamp:  time.Now(),
			Confidence: finding.Confidence,
			Source:     "daemon",
			Topics:     topics,
		}
		knowledge.SavePendingEntry(pendingDir, entry)
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
	otel.SetAttributes(processSpan, otel.BrainDaemonOutcome.String("success"))
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

func looksInternal(host string) bool {
	public := []string{"github.com", "gitlab.com", "bitbucket.org", "localhost", "127.0.0.1", "0.0.0.0"}
	lower := strings.ToLower(host)
	for _, p := range public {
		if strings.Contains(lower, p) {
			return false
		}
	}
	if strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") ||
		strings.HasSuffix(lower, ".corp") || strings.Contains(lower, "192.168.") ||
		strings.Contains(lower, "10.") || strings.Contains(lower, "172.16.") {
		return true
	}
	return false
}
