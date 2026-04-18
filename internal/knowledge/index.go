package knowledge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type IndexEntry struct {
	Strength       float64   `json:"strength"`
	RetrievalCount int       `json:"retrieval_count"`
	LastRetrieved  time.Time `json:"last_retrieved"`
	HalfLifeDays   int       `json:"half_life_days"`
	Confidence     string    `json:"confidence"`
	Topics         []string  `json:"topics"`

	SupersededBy string    `json:"superseded_by,omitempty"`
	Supersedes   string    `json:"supersedes,omitempty"`
	Version      int       `json:"version"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	ConflictWith []string  `json:"conflict_with,omitempty"`
}

type Index struct {
	Version     int                   `json:"version"`
	LastRebuild time.Time             `json:"last_rebuild"`
	Entries     map[string]IndexEntry `json:"entries"`
}

const indexVersion = 2

var entryHeaderRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

func IndexFilePath(brainDir string) string {
	return filepath.Join(brainDir, "index.json")
}

func (h *Hub) indexPath() string {
	return IndexFilePath(h.dir)
}

func LoadIndex(brainDir string) (*Index, error) {
	data, err := os.ReadFile(IndexFilePath(brainDir))
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
	if idx.Version < 1 || idx.Version > indexVersion {
		return newEmptyIndex(), nil
	}
	if idx.Version < 2 {
		for key, entry := range idx.Entries {
			entry.Version = 1
			idx.Entries[key] = entry
		}
	}
	return &idx, nil
}

func (h *Hub) LoadIndex() (*Index, error) {
	return LoadIndex(h.dir)
}

func (idx *Index) Save(brainDir string) error {
	idx.Version = indexVersion
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

func (h *Hub) SaveIndex(idx *Index) error {
	return idx.Save(h.dir)
}

func RebuildIndex(brainDir string) (*Index, error) {
	idx := newEmptyIndex()

	for _, topic := range AvailableTopics() {
		path, err := TopicFilePathForDir(topic, brainDir)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		var currentEntry strings.Builder
		var currentKey string
		for _, line := range lines {
			matches := entryHeaderRe.FindStringSubmatch(line)
			if matches != nil {
				if currentEntry.Len() > 0 && currentKey != "" {
					if _, exists := idx.Entries[currentKey]; !exists {
						halfLife := 7
						if topic == "gotchas" {
							halfLife = 14
						}
						idx.Entries[currentKey] = IndexEntry{
							Strength:       1.0,
							RetrievalCount: 0,
							LastRetrieved:  time.Now(),
							HalfLifeDays:   halfLife,
							Confidence:     "observed",
							Topics:         DetectTopics(currentEntry.String()),
						}
					}
				}
				currentKey = topic + ":" + matches[1]
				currentEntry.Reset()
				currentEntry.WriteString(line)
				continue
			}
			if currentKey != "" {
				currentEntry.WriteString("\n")
				currentEntry.WriteString(line)
			}
		}
		if currentEntry.Len() > 0 && currentKey != "" {
			if _, exists := idx.Entries[currentKey]; !exists {
				halfLife := 7
				if topic == "gotchas" {
					halfLife = 14
				}
				idx.Entries[currentKey] = IndexEntry{
					Strength:       1.0,
					RetrievalCount: 0,
					LastRetrieved:  time.Now(),
					HalfLifeDays:   halfLife,
					Confidence:     "observed",
					Topics:         DetectTopics(currentEntry.String()),
				}
			}
		}
	}

	idx.LastRebuild = time.Now()
	return idx, nil
}

func (h *Hub) RebuildIndex() (*Index, error) {
	return RebuildIndex(h.dir)
}

func MakeKey(topic, timestamp string) string {
	return topic + ":" + timestamp
}

func (idx *Index) Get(topic, timestamp string) (IndexEntry, bool) {
	entry, ok := idx.Entries[MakeKey(topic, timestamp)]
	return entry, ok
}

func (idx *Index) Set(topic, timestamp string, entry IndexEntry) {
	idx.Entries[MakeKey(topic, timestamp)] = entry
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
		Version: indexVersion,
		Entries: make(map[string]IndexEntry),
	}
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

var topicKeywords = map[string][]string{
	"ui":            {"react", "css", "component", "style", "tailwind", "html", "frontend", "navbar", "button", "form", "responsive", "dark mode", "jsx", "typescript"},
	"backend":       {"api", "handler", "controller", "service", "middleware", "route", "endpoint", "grpc", "rest", "graphql", "http", "server"},
	"infrastructure": {"vps", "deploy", "docker", "ci", "cd", "kubernetes", "cloudflare", "nginx", "ssl", "domain", "dns", "server", "ubuntu", "fly.io", "render", "vercel"},
	"database":      {"sql", "migration", "schema", "query", "postgres", "mysql", "sqlite", "mongo", "redis", "index", "table", "gorm", "prisma"},
	"security":      {"auth", "secret", "token", "permission", "jwt", "oauth", "bcrypt", "argon2", "encrypt", "csrf", "cors", "password", "session"},
	"testing":       {"test", "spec", "mock", "assert", "vitest", "jest", "pytest", "coverage", "tdd", "suite", "fixture"},
	"architecture":  {"module", "layer", "package", "directory", "structure", "pattern", "abstraction", "dependency", "interface", "refactor"},
}

func DetectTopics(text string) []string {
	lower := strings.ToLower(text)
	var topics []string
	for topic, keywords := range topicKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				topics = append(topics, topic)
				break
			}
		}
	}
	if len(topics) == 0 {
		topics = []string{"general"}
	}
	return topics
}

func (h *Hub) TopEntries(n int) []IndexEntry {
	idx, err := h.LoadIndex()
	if err != nil {
		return nil
	}
	now := time.Now()

	type scored struct {
		key     string
		entry   IndexEntry
		score   float64
	}

	var entries []scored
	for key, entry := range idx.Entries {
		entries = append(entries, scored{key: key, entry: entry, score: CalculateStrength(entry, now)})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})
	if len(entries) > n {
		entries = entries[:n]
	}

	result := make([]IndexEntry, len(entries))
	for i, e := range entries {
		result[i] = e.entry
	}
	return result
}

func contentFingerprint(content string) string {
	normalized := normalizeEntry(content)
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h[:8])
}
