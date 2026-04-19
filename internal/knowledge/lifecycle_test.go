package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateEntry_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	timestamp := "2026-04-15 10:00:00"
	content := "# Gotchas\n\n### [" + timestamp + "] Old message\n"
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	idx.Entries[MakeKey("gotchas", timestamp)] = IndexEntry{
		Strength:       1.0,
		RetrievalCount: 0,
		LastRetrieved:  time.Now(),
		HalfLifeDays:   14,
		Confidence:     "observed",
		Topics:         []string{"general"},
		Version:        1,
	}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	if err := hub.UpdateEntry("gotchas", timestamp, "New message"); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(brainDir, "gotchas.md"))
	if !strings.Contains(string(data), "New message") {
		t.Error("expected new message in file")
	}
	if strings.Contains(string(data), "Old message") {
		t.Error("old message should be removed")
	}

	updatedIdx, _ := LoadIndex(brainDir)
	entry := updatedIdx.Entries[MakeKey("gotchas", timestamp)]
	if entry.Version != 2 {
		t.Errorf("expected version 2, got %d", entry.Version)
	}
	if entry.Confidence != "verified" {
		t.Errorf("expected confidence 'verified', got %q", entry.Confidence)
	}

	archiveDir := filepath.Join(brainDir, "archived", "versions")
	entries, _ := os.ReadDir(archiveDir)
	if len(entries) == 0 {
		t.Error("expected archived version file")
	}
}

func TestUpdateEntry_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := "# Gotchas\n\n### [2026-04-15 10:00:00] Some message\n"
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, _ := Open(brainDir)
	err := hub.UpdateEntry("gotchas", "nonexistent", "New message")
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
}

func TestSupersedeEntry_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	oldTS := "2026-04-15 10:00:00"
	newTS := "2026-04-18 12:00:00"
	content := "# Gotchas\n\n### [" + oldTS + "] Old entry\n\n### [" + newTS + "] New entry\n"
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	idx.Entries[MakeKey("gotchas", oldTS)] = IndexEntry{
		Strength:       1.0,
		RetrievalCount: 0,
		LastRetrieved:  time.Now(),
		HalfLifeDays:   14,
		Confidence:     "observed",
		Topics:         []string{"general"},
	}
	idx.Entries[MakeKey("gotchas", newTS)] = IndexEntry{
		Strength:       1.0,
		RetrievalCount: 0,
		LastRetrieved:  time.Now(),
		HalfLifeDays:   14,
		Confidence:     "observed",
		Topics:         []string{"general"},
	}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, _ := Open(brainDir)
	if err := hub.SupersedeEntry("gotchas", oldTS, newTS); err != nil {
		t.Fatalf("SupersedeEntry failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(brainDir, "gotchas.md"))
	if !strings.Contains(string(data), "~~Old entry~~") {
		t.Error("expected strikethrough on old entry")
	}

	updatedIdx, _ := LoadIndex(brainDir)
	oldEntry := updatedIdx.Entries[MakeKey("gotchas", oldTS)]
	if oldEntry.Confidence != "superseded" {
		t.Errorf("expected superseded confidence, got %q", oldEntry.Confidence)
	}
	if oldEntry.SupersededBy != MakeKey("gotchas", newTS) {
		t.Errorf("expected SupersededBy link, got %q", oldEntry.SupersededBy)
	}

	newEntry := updatedIdx.Entries[MakeKey("gotchas", newTS)]
	if newEntry.Supersedes != MakeKey("gotchas", oldTS) {
		t.Errorf("expected Supersedes link, got %q", newEntry.Supersedes)
	}
}

func TestFindConflicts_DetectsOpposingSentiment(t *testing.T) {
	tests := []struct {
		a, b       string
		expectConf bool
	}{
		{"Always use X", "Never use X", true},
		{"Use filepath.Join", "Avoid filepath.Join", true},
		{"Enable caching", "Disable caching", true},
		{"Use X for paths", "Use X for strings", false},
		{"Always do Y", "Always do Z", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			result := hasOpposingSentiment(tt.a, tt.b)
			if result != tt.expectConf {
				t.Errorf("hasOpposingSentiment(%q, %q) = %v, want %v",
					tt.a, tt.b, result, tt.expectConf)
			}
		})
	}
}

func TestParseEntriesFromContent_SkipsSuperseded(t *testing.T) {
	content := `# Gotchas

### [2026-04-15 10:00:00] Active entry

### [2026-04-16 11:00:00] ~~Superseded entry~~ (superseded)

### [2026-04-17 12:00:00] Another active entry
`

	entries, err := parseEntriesFromContent(content)
	if err != nil {
		t.Fatalf("parseEntriesFromContent failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries (superseded skipped), got %d", len(entries))
	}

	for _, e := range entries {
		if strings.Contains(e.Message, "~~") {
			t.Errorf("superseded entry should be skipped: %s", e.Message)
		}
	}
}

func TestIndexMigration_V1toV2(t *testing.T) {
	ResetIndexCache()
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	v1Index := `{
		"version": 1,
		"last_rebuild": "2026-04-15T10:00:00Z",
		"entries": {
			"gotchas:2026-04-15 10:00:00": {
				"strength": 1.0,
				"retrieval_count": 0,
				"last_retrieved": "2026-04-15T10:00:00Z",
				"half_life_days": 14,
				"confidence": "observed",
				"topics": ["general"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(brainDir, "index.json"), []byte(v1Index), 0600); err != nil {
		t.Fatal(err)
	}

	idx, err := LoadIndex(brainDir)
	if err != nil {
		t.Fatalf("failed to load index: %v", err)
	}

	entry := idx.Entries["gotchas:2026-04-15 10:00:00"]
	if entry.Version != 1 {
		t.Errorf("expected migrated version=1, got %d", entry.Version)
	}
}

func TestGetTopicEntriesForDir(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := "# Gotchas\n\n### [2026-04-15 10:00:00] Entry A\n\n### [2026-04-16 11:00:00] Entry B\n"
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := GetTopicEntriesForDir("gotchas", brainDir)
	if err != nil {
		t.Fatalf("GetTopicEntriesForDir failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}
