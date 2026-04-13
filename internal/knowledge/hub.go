package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	cachedBrainDir string
	brainDirOnce   sync.Once
	brainDirMu     sync.Mutex
)

type Hub struct {
	dir string
}

func Open(brainDir string) (*Hub, error) {
	if brainDir == "" {
		return nil, fmt.Errorf("brainDir must not be empty")
	}
	return &Hub{dir: brainDir}, nil
}

func FindBrainDir() (string, error) {
	brainDirMu.Lock()
	defer brainDirMu.Unlock()
	var err error
	brainDirOnce.Do(func() {
		cachedBrainDir, err = findBrainDirFromUncached()
	})
	return cachedBrainDir, err
}

func FindBrainDirFrom(cwd string) (string, error) {
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".brain")
		info, err := os.Lstat(candidate)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf(".brain at %s is a symlink — not allowed for security.\nWhat to do: remove the symlink and run 'brain init' again", dir)
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

func findBrainDirFromUncached() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindBrainDirFrom(cwd)
}

func ResetCache() {
	brainDirMu.Lock()
	defer brainDirMu.Unlock()
	brainDirOnce = sync.Once{}
	cachedBrainDir = ""
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

var topicFiles = map[string]string{
	"memory":       "MEMORY.md",
	"gotchas":      "gotchas.md",
	"patterns":     "patterns.md",
	"decisions":    "decisions.md",
	"architecture": "architecture.md",
}

func AvailableTopics() []string {
	return []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
}

func (h *Hub) topicFilePath(topic string) (string, error) {
	filename, ok := topicFiles[strings.ToLower(topic)]
	if !ok {
		return "", fmt.Errorf("unknown topic '%s'. Available: memory, gotchas, patterns, decisions, architecture.\nWhat to do: use one of the listed topic names.", topic)
	}
	return filepath.Join(h.dir, filename), nil
}

func TopicFilePathForDir(topic, brainDir string) (string, error) {
	filename, ok := topicFiles[strings.ToLower(topic)]
	if !ok {
		return "", fmt.Errorf("unknown topic '%s'. Available: memory, gotchas, patterns, decisions, architecture.\nWhat to do: use one of the listed topic names.", topic)
	}
	return filepath.Join(brainDir, filename), nil
}

func (h *Hub) Dir() string {
	return h.dir
}

func (h *Hub) PendingDir() string {
	return filepath.Join(h.dir, "pending")
}

func EnsureBrainDir(cwd string) error {
	brainDir := filepath.Join(cwd, ".brain")
	dirs := []string{
		brainDir,
		filepath.Join(brainDir, ".queue", "done"),
		filepath.Join(brainDir, ".queue", "failed"),
		filepath.Join(brainDir, ".queue", "flagged"),
		filepath.Join(brainDir, "sessions"),
		filepath.Join(brainDir, "archived"),
		filepath.Join(brainDir, "pending"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
