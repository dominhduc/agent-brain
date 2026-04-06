package brain

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dominhduc/agent-brain/internal/review"
)

var (
	cachedBrainDir string
	brainDirOnce   sync.Once
	brainDirMu     sync.Mutex
)

func FindBrainDir() (string, error) {
	brainDirMu.Lock()
	defer brainDirMu.Unlock()
	var err error
	brainDirOnce.Do(func() {
		cachedBrainDir, err = findBrainDirUncached()
	})
	return cachedBrainDir, err
}

func ResetCache() {
	brainDirMu.Lock()
	defer brainDirMu.Unlock()
	brainDirOnce = sync.Once{}
	cachedBrainDir = ""
}

func findBrainDirUncached() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".brain")
		info, err := os.Lstat(candidate)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf(".brain at %s is a symlink — this is not allowed for security.\nWhat to do: remove the symlink and run 'brain init' again", dir)
			}
			if info.IsDir() {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("knowledge hub not found.\nWhat to do: run \"brain init\" in your project directory first")
}

func BrainDirExists(cwd string) bool {
	candidate := filepath.Join(cwd, ".brain")
	info, err := os.Lstat(candidate)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return info.IsDir()
}

func IsGitRepo(cwd string) bool {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func EnsureBrainDir(cwd string) error {
	brainDir := filepath.Join(cwd, ".brain")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		return err
	}

	queueDir := filepath.Join(brainDir, ".queue", "done")
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		return err
	}

	failedDir := filepath.Join(brainDir, ".queue", "failed")
	if err := os.MkdirAll(failedDir, 0755); err != nil {
		return err
	}

	flaggedDir := filepath.Join(brainDir, ".queue", "flagged")
	if err := os.MkdirAll(flaggedDir, 0755); err != nil {
		return err
	}

	sessionsDir := filepath.Join(brainDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}

	archivedDir := filepath.Join(brainDir, "archived")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return err
	}

	pendingDir := filepath.Join(brainDir, "pending")
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return err
	}

	return nil
}

var topicFiles = map[string]string{
	"memory":       "MEMORY.md",
	"gotchas":      "gotchas.md",
	"patterns":     "patterns.md",
	"decisions":    "decisions.md",
	"architecture": "architecture.md",
}

func TopicFilePath(topic string) (string, error) {
	brainDir, err := FindBrainDir()
	if err != nil {
		return "", err
	}

	filename, ok := topicFiles[strings.ToLower(topic)]
	if !ok {
		return "", fmt.Errorf("unknown topic '%s'. Available topics: memory, gotchas, patterns, decisions, architecture.\nWhat to do: use one of the listed topic names.", topic)
	}

	return filepath.Join(brainDir, filename), nil
}

func AvailableTopics() []string {
	return []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
}

func GetTopic(name string) (string, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w.\nWhat to do: run 'brain init' to recreate missing files.", filepath.Base(path), err)
	}

	return string(data), nil
}

func GetAllTopics() (string, error) {
	var result strings.Builder
	for _, topic := range AvailableTopics() {
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), content))
	}
	return result.String(), nil
}

func AddEntry(topic string, message string) error {
	path, err := TopicFilePath(topic)
	if err != nil {
		return err
	}

	normalizedMsg := normalizeEntry(message)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	if data != nil {
		existing := string(data)
		lines := strings.Split(existing, "\n")
		for _, line := range lines {
			msg := extractMessageFromEntry(line)
			lineNormalized := normalizeEntry(msg)
			if lineNormalized == normalizedMsg && lineNormalized != "" {
				return nil
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w.\nWhat to do: check file permissions.", filepath.Base(path), err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filepath.Base(path), err)
	}

	if strings.ToLower(topic) == "memory" {
		lineCount, _ := MemoryLineCount()
		if lineCount > 200 {
			fmt.Fprintf(os.Stderr, "Warning: MEMORY.md is %d lines (recommended: under 200).\nWhat to do: move detailed entries to topic files (gotchas.md, patterns.md, etc.).\n", lineCount)
		}
	}

	return nil
}

func PendingDir(cwd string) string {
	return filepath.Join(cwd, ".brain", "pending")
}

func AddPendingEntry(cwd string, entry review.PendingEntry) error {
	pendingDir := PendingDir(cwd)
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return fmt.Errorf("creating pending directory: %w", err)
	}
	return review.SavePendingEntry(pendingDir, entry)
}

func MemoryLineCount() (int, error) {
	brainDir, err := FindBrainDir()
	if err != nil {
		return 0, err
	}

	path := filepath.Join(brainDir, "MEMORY.md")
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func normalizeEntry(message string) string {
	re := strings.NewReplacer(
		"\n", " ",
		"\r", "",
		"  ", " ",
	)
	normalized := re.Replace(message)
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	return normalized
}

func extractMessageFromEntry(line string) string {
	if strings.HasPrefix(line, "### [") {
		idx := strings.Index(line, "] ")
		if idx > 0 {
			return line[idx+2:]
		}
	}
	return line
}

type TopicSummary struct {
	Name          string `json:"name"`
	EntryCount    int    `json:"entry_count"`
	LineCount     int    `json:"line_count"`
	HasDuplicates bool   `json:"has_duplicates"`
}

func GetTopicSummary(name string) (TopicSummary, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return TopicSummary{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return TopicSummary{}, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	entryCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "### [") {
			entryCount++
		}
	}

	hasDupes := detectDuplicates(content)

	return TopicSummary{
		Name:          name,
		EntryCount:    entryCount,
		LineCount:     len(lines),
		HasDuplicates: hasDupes,
	}, nil
}

func detectDuplicates(content string) bool {
	lines := strings.Split(content, "\n")
	seen := make(map[string]bool)
	for _, line := range lines {
		if strings.HasPrefix(line, "### [") {
			normalized := strings.ToLower(strings.TrimSpace(line))
			if seen[normalized] {
				return true
			}
			seen[normalized] = true
		}
	}
	return false
}

func GetAllSummaries() ([]TopicSummary, error) {
	var summaries []TopicSummary
	for _, topic := range AvailableTopics() {
		summary, err := GetTopicSummary(topic)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func GetAllTopicsWithSummary() (string, error) {
	summaries, err := GetAllSummaries()
	if err != nil {
		return "", err
	}

	var result strings.Builder

	result.WriteString("# PROJECT MEMORY SUMMARY\n\n")
	result.WriteString("## Overview\n\n")
	for _, s := range summaries {
		result.WriteString(fmt.Sprintf("- **%s**: %d entries, %d lines", s.Name, s.EntryCount, s.LineCount))
		if s.HasDuplicates {
			result.WriteString(" (⚠️ duplicates detected)")
		}
		result.WriteString("\n")
	}
	result.WriteString("\n---\n\n")

	for _, topic := range AvailableTopics() {
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}
		deduped := deduplicateContent(content)
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), deduped))
	}
	return result.String(), nil
}

func deduplicateContent(content string) string {
	lines := strings.Split(content, "\n")
	seen := make(map[string]bool)
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, "### [") {
			normalized := strings.ToLower(strings.TrimSpace(line))
			if seen[normalized] {
				continue
			}
			seen[normalized] = true
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}