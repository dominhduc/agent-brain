package wm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Entry struct {
	Content    string    `json:"content"`
	Importance float64   `json:"importance"`
	Timestamp  time.Time `json:"timestamp"`
}

const (
	MaxEntries    = 20
	bufferDirName = "buffer"
	bufferFile    = "wm.json"
)

func bufferDir(brainDir string) string {
	return filepath.Join(brainDir, bufferDirName)
}

func bufferPath(brainDir string) string {
	return filepath.Join(bufferDir(brainDir), bufferFile)
}

func Push(brainDir, content string, importance float64) error {
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	dir := bufferDir(brainDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating buffer dir: %w", err)
	}

	entries, _ := load(brainDir)

	entries = append(entries, Entry{
		Content:    content,
		Importance: importance,
		Timestamp:  time.Now(),
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Importance > entries[j].Importance
	})

	if len(entries) > MaxEntries {
		entries = entries[:MaxEntries]
	}

	return save(brainDir, entries)
}

func Read(brainDir string) ([]Entry, error) {
	return load(brainDir)
}

func Clear(brainDir string) error {
	path := bufferPath(brainDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Flush(brainDir string) error {
	return Clear(brainDir)
}

func load(brainDir string) ([]Entry, error) {
	path := bufferPath(brainDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading buffer: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, nil
	}

	return entries, nil
}

func save(brainDir string, entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling buffer: %w", err)
	}

	path := bufferPath(brainDir)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing buffer: %w", err)
	}

	return nil
}
