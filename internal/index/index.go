package index

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
)

type IndexEntry struct {
	Strength        float64   `json:"strength"`
	RetrievalCount  int       `json:"retrieval_count"`
	LastRetrieved   time.Time `json:"last_retrieved"`
	HalfLifeDays    int       `json:"half_life_days"`
	Confidence      string    `json:"confidence"`
}

type Index struct {
	Version     int                    `json:"version"`
	LastRebuild time.Time              `json:"last_rebuild"`
	Entries     map[string]IndexEntry  `json:"entries"`
}

const currentVersion = 1
const indexFilename = "index.json"

func IndexFilePath(brainDir string) string {
	return filepath.Join(brainDir, indexFilename)
}

func MakeKey(topic, timestamp string) string {
	return topic + ":" + timestamp
}

func Load(brainDir string) (*Index, error) {
	path := IndexFilePath(brainDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newEmptyIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return newEmptyIndex(), nil
	}

	if idx.Entries == nil {
		idx.Entries = make(map[string]IndexEntry)
	}
	if idx.Version != currentVersion {
		return newEmptyIndex(), nil
	}

	return &idx, nil
}

func (idx *Index) Save(brainDir string) error {
	idx.Version = currentVersion
	idx.LastRebuild = time.Now()

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	path := IndexFilePath(brainDir)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename index: %w", err)
	}

	return nil
}

func (idx *Index) Get(topic, timestamp string) (IndexEntry, bool) {
	key := MakeKey(topic, timestamp)
	entry, ok := idx.Entries[key]
	return entry, ok
}

func (idx *Index) Set(topic, timestamp string, entry IndexEntry) {
	key := MakeKey(topic, timestamp)
	idx.Entries[key] = entry
}

func (idx *Index) GetByRawKey(key string) (IndexEntry, bool) {
	entry, ok := idx.Entries[key]
	return entry, ok
}

func (idx *Index) SetByRawKey(key string, entry IndexEntry) {
	idx.Entries[key] = entry
}

func newEmptyIndex() *Index {
	return &Index{
		Version: currentVersion,
		Entries: make(map[string]IndexEntry),
	}
}

var entryHeaderRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

func Rebuild(brainDir string) (*Index, error) {
	idx := newEmptyIndex()

	for _, topic := range brain.AvailableTopics() {
		path, err := brain.TopicFilePathForDir(topic, brainDir)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			matches := entryHeaderRe.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			timestamp := matches[1]
			key := MakeKey(topic, timestamp)

			if _, exists := idx.Entries[key]; exists {
				continue
			}

			halfLife := 7
			if topic == "gotchas" {
				halfLife = 14
			}

			idx.Entries[key] = IndexEntry{
				Strength:       1.0,
				RetrievalCount: 0,
				LastRetrieved:  time.Now(),
				HalfLifeDays:   halfLife,
				Confidence:     "observed",
			}
		}
	}

	idx.LastRebuild = time.Now()
	return idx, nil
}

func CalculateStrength(e IndexEntry, now time.Time) float64 {
	halfLife := float64(e.HalfLifeDays)
	if halfLife <= 0 {
		halfLife = 7
	}

	effectiveHalfLife := halfLife + float64(e.RetrievalCount)*2

	var ageDays float64
	if !e.LastRetrieved.IsZero() {
		ageDays = now.Sub(e.LastRetrieved).Hours() / 24
	}

	strength := math.Pow(0.5, ageDays/effectiveHalfLife)

	switch strings.ToLower(e.Confidence) {
	case "verified":
		strength *= 1.2
	case "stale":
		strength *= 0.1
	}

	if strength > 1.0 {
		strength = 1.0
	}
	if strength < 0 {
		strength = 0
	}

	return strength
}
