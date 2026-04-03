package review

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PendingEntry struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	Content   string    `json:"content"`
	CommitSHA string    `json:"commit_sha"`
	Timestamp time.Time `json:"timestamp"`
	Confidence string   `json:"confidence"`
	Source    string    `json:"source"`
}

func (e PendingEntry) Fingerprint() string {
	normalized := strings.ToLower(strings.TrimSpace(e.Content))
	h := sha256.Sum256([]byte(e.Topic + ":" + normalized))
	return fmt.Sprintf("%x", h[:8])
}

func (e PendingEntry) DisplayTime() string {
	return e.Timestamp.Format("2006-01-02 15:04")
}

func LoadPendingEntries(pendingDir string) ([]PendingEntry, error) {
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading pending directory: %w", err)
	}

	var result []PendingEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(pendingDir, e.Name()))
		if err != nil {
			continue
		}
		var entry PendingEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.ID == "" {
			continue
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

func SavePendingEntry(pendingDir string, entry PendingEntry) error {
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return fmt.Errorf("creating pending directory: %w", err)
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	path := filepath.Join(pendingDir, entry.ID+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing entry file: %w", err)
	}
	return nil
}

func RemovePendingEntry(pendingDir, id string) error {
	path := filepath.Join(pendingDir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing entry %s: %w", id, err)
	}
	return nil
}

func GroupByTopic(entries []PendingEntry) map[string][]PendingEntry {
	groups := make(map[string][]PendingEntry)
	for _, e := range entries {
		groups[e.Topic] = append(groups[e.Topic], e)
	}
	return groups
}

func CountByTopic(entries []PendingEntry) map[string]int {
	counts := make(map[string]int)
	for _, e := range entries {
		counts[e.Topic]++
	}
	return counts
}

type DedupGroup struct {
	Fingerprint    string
	Entries        []PendingEntry
	Representative string
}

func FindDuplicateGroups(entries []PendingEntry) []DedupGroup {
	fingerprints := make(map[string][]PendingEntry)
	for _, e := range entries {
		fp := e.Fingerprint()
		fingerprints[fp] = append(fingerprints[fp], e)
	}

	var groups []DedupGroup
	for fp, entries := range fingerprints {
		if len(entries) > 1 {
			groups = append(groups, DedupGroup{
				Fingerprint:    fp,
				Entries:        entries,
				Representative: entries[0].Content,
			})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Entries) > len(groups[j].Entries)
	})
	return groups
}
