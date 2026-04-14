package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SessionRetrieved struct {
	SessionID string    `json:"session_id"`
	Keys      []string  `json:"keys"`
	Started   time.Time `json:"started"`
}

func (h *Hub) sessionDir() string {
	return filepath.Join(h.dir, ".session")
}

func (h *Hub) retrievalPath() string {
	return filepath.Join(h.sessionDir(), "retrieved.json")
}

func (h *Hub) RecordRetrieval(keys []string) error {
	dir := h.sessionDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}

	sr := SessionRetrieved{
		SessionID: fmt.Sprintf("sess-%s", time.Now().Format("20060102-150405")),
		Keys:      keys,
		Started:   time.Now(),
	}

	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(h.retrievalPath(), data, 0600)
}

func (h *Hub) LoadRetrievals() ([]string, error) {
	data, err := os.ReadFile(h.retrievalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}
	var sr SessionRetrieved
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, nil
	}
	return sr.Keys, nil
}

func (h *Hub) ClearRetrievals() error {
	path := h.retrievalPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func RecordRetrieval(brainDir string, keys []string) error {
	h, err := Open(brainDir)
	if err != nil {
		return err
	}
	return h.RecordRetrieval(keys)
}

func LoadRetrievals(brainDir string) ([]string, error) {
	path := filepath.Join(brainDir, ".session", "retrieved.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}
	var sr SessionRetrieved
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, nil
	}
	return sr.Keys, nil
}

func ClearRetrievals(brainDir string) error {
	path := filepath.Join(brainDir, ".session", "retrieved.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
