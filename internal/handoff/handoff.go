package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Handoff struct {
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	Next      string    `json:"next"`
	Session   string    `json:"session"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	handoffDirName = "handoffs"
	latestFile     = "latest.json"
)

func handoffDir(brainDir string) string {
	return filepath.Join(brainDir, handoffDirName)
}

func latestPath(brainDir string) string {
	return filepath.Join(handoffDir(brainDir), latestFile)
}

func Create(brainDir, summary, next, session string) (*Handoff, error) {
	dir := handoffDir(brainDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating handoff dir: %w", err)
	}

	h := &Handoff{
		ID:        fmt.Sprintf("handoff-%s", time.Now().Format("20060102-150405")),
		Summary:   summary,
		Next:      next,
		Session:   session,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling handoff: %w", err)
	}

	if err := os.WriteFile(latestPath(brainDir), data, 0600); err != nil {
		return nil, fmt.Errorf("writing handoff: %w", err)
	}

	return h, nil
}

func Latest(brainDir string) (*Handoff, error) {
	path := latestPath(brainDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading handoff: %w", err)
	}

	var h Handoff
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, nil
	}

	return &h, nil
}

func Show(brainDir, id string) (*Handoff, error) {
	h, err := Latest(brainDir)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, nil
	}
	if h.ID != id {
		return nil, nil
	}
	return h, nil
}

func Resume(brainDir string) (*Handoff, error) {
	return Latest(brainDir)
}
