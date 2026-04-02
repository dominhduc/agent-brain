package brain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func FindBrainDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".brain")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("knowledge hub not found. Run \"brain init\" in your project directory first")
}

func BrainDirExists(cwd string) bool {
	candidate := filepath.Join(cwd, ".brain")
	info, err := os.Stat(candidate)
	return err == nil && info.IsDir()
}

func IsGitRepo(cwd string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = cwd
	return cmd.Run() == nil
}

func FriendlyError(msg string) string {
	return msg
}

func ValidateBrainDir(brainDir string) error {
	required := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	for _, f := range required {
		path := filepath.Join(brainDir, f)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("missing required file: %s", f)
		}
	}
	return nil
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

func TopicFilePath(topic string) (string, error) {
	brainDir, err := FindBrainDir()
	if err != nil {
		return "", err
	}

	topicFiles := map[string]string{
		"memory":       "MEMORY.md",
		"gotchas":      "gotchas.md",
		"patterns":     "patterns.md",
		"decisions":    "decisions.md",
		"architecture": "architecture.md",
	}

	filename, ok := topicFiles[strings.ToLower(topic)]
	if !ok {
		return "", fmt.Errorf("unknown topic '%s'. Available topics: memory, gotchas, patterns, decisions, architecture", topic)
	}

	return filepath.Join(brainDir, filename), nil
}
