package outcome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const sessionDirName = ".session"
const retrievedFile = "retrieved.json"

type SessionRetrieved struct {
	SessionID string    `json:"session_id"`
	Keys      []string  `json:"keys"`
	Started   time.Time `json:"started"`
}

func sessionDir(brainDir string) string {
	return filepath.Join(brainDir, sessionDirName)
}

func retrievedPath(brainDir string) string {
	return filepath.Join(sessionDir(brainDir), retrievedFile)
}

func Track(brainDir string, keys []string) error {
	dir := sessionDir(brainDir)
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

	if err := os.WriteFile(retrievedPath(brainDir), data, 0600); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	return nil
}

func LoadKeys(brainDir string) ([]string, error) {
	path := retrievedPath(brainDir)
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

func Clear(brainDir string) error {
	path := retrievedPath(brainDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
