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
)

var (
	cachedBrainDir string
	brainDirOnce   sync.Once
)

func FindBrainDir() (string, error) {
	var err error
	brainDirOnce.Do(func() {
		cachedBrainDir, err = findBrainDirUncached()
	})
	return cachedBrainDir, err
}

func ResetCache() {
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
