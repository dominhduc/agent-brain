package index

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func setupTestBrainDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"} {
		if err := os.WriteFile(filepath.Join(brainDir, name), []byte("# "+name+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	brain.ResetCache()
	t.Cleanup(func() {
		os.Chdir(originalDir)
		brain.ResetCache()
	})
	return brainDir
}

func TestIndexLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	os.MkdirAll(brainDir, 0755)

	idx := newEmptyIndex()
	now := time.Now().UTC().Truncate(time.Second)
	idx.Set("gotchas", "2026-04-07 10:00:00", IndexEntry{
		Strength:       0.8,
		RetrievalCount: 3,
		LastRetrieved:  now,
		HalfLifeDays:   9,
		Confidence:     "verified",
	})

	if err := idx.Save(brainDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(brainDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	entry, ok := loaded.Get("gotchas", "2026-04-07 10:00:00")
	if !ok {
		t.Fatal("entry not found after load")
	}
	if entry.Strength != 0.8 {
		t.Errorf("strength = %v, want 0.8", entry.Strength)
	}
	if entry.RetrievalCount != 3 {
		t.Errorf("retrieval_count = %v, want 3", entry.RetrievalCount)
	}
	if entry.HalfLifeDays != 9 {
		t.Errorf("half_life_days = %v, want 9", entry.HalfLifeDays)
	}
	if entry.Confidence != "verified" {
		t.Errorf("confidence = %v, want verified", entry.Confidence)
	}
}

func TestLoad_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty index, got %d entries", len(idx.Entries))
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "index.json"), []byte("not valid json"), 0600)
	idx, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty index for corrupt file, got %d entries", len(idx.Entries))
	}
}

func TestLoad_WrongVersion(t *testing.T) {
	tmpDir := t.TempDir()
	data, _ := json.Marshal(map[string]any{"version": 999, "entries": map[string]any{}})
	os.WriteFile(filepath.Join(tmpDir, "index.json"), data, 0600)
	idx, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty index for wrong version, got %d entries", len(idx.Entries))
	}
}

func TestRebuild(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(
		"# Gotchas\n\n"+
			"### [2026-04-07 10:00:00] First gotcha\n\n"+
			"### [2026-04-07 11:00:00] Second gotcha\n\n",
	), 0600)

	os.WriteFile(filepath.Join(brainDir, "patterns.md"), []byte(
		"# Patterns\n\n"+
			"### [2026-04-06 09:00:00] A pattern\n\n",
	), 0600)

	idx, err := Rebuild(brainDir)
	if err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}

	if len(idx.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(idx.Entries))
	}

	e1, ok := idx.Get("gotchas", "2026-04-07 10:00:00")
	if !ok {
		t.Error("expected to find gotchas:2026-04-07 10:00:00")
	} else if e1.HalfLifeDays != 14 {
		t.Errorf("gotcha half_life = %d, want 14", e1.HalfLifeDays)
	}

	e2, ok := idx.Get("patterns", "2026-04-06 09:00:00")
	if !ok {
		t.Error("expected to find patterns:2026-04-06 09:00:00")
	} else if e2.HalfLifeDays != 7 {
		t.Errorf("pattern half_life = %d, want 7", e2.HalfLifeDays)
	}
}

func TestCalculateStrength(t *testing.T) {
	now := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		entry    IndexEntry
		expected float64
	}{
		{
			name:     "fresh entry (age=0)",
			entry:    IndexEntry{HalfLifeDays: 7, RetrievalCount: 0},
			expected: 1.0,
		},
		{
			name: "7 days old, no retrieval",
			entry: IndexEntry{
				HalfLifeDays:   7,
				RetrievalCount: 0,
				LastRetrieved:  now.Add(-7 * 24 * time.Hour),
			},
			expected: 0.5,
		},
		{
			name: "14 days old, no retrieval",
			entry: IndexEntry{
				HalfLifeDays:   7,
				RetrievalCount: 0,
				LastRetrieved:  now.Add(-14 * 24 * time.Hour),
			},
			expected: 0.25,
		},
		{
			name: "retrieved 3 times, 5 days old",
			entry: IndexEntry{
				HalfLifeDays:   7,
				RetrievalCount: 3,
				LastRetrieved:  now.Add(-5 * 24 * time.Hour),
			},
			expected: 0.77,
		},
		{
			name: "verified confidence boost",
			entry: IndexEntry{
				HalfLifeDays:   7,
				RetrievalCount: 0,
				LastRetrieved:  now,
				Confidence:     "verified",
			},
			expected: 1.0,
		},
		{
			name: "stale confidence penalty",
			entry: IndexEntry{
				HalfLifeDays:   7,
				RetrievalCount: 0,
				LastRetrieved:  now,
				Confidence:     "stale",
			},
			expected: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateStrength(tt.entry, now)
			if math.Abs(got-tt.expected) > 0.02 {
				t.Errorf("CalculateStrength = %v, want ~%v", got, tt.expected)
			}
		})
	}
}

func TestMakeKey(t *testing.T) {
	key := MakeKey("gotchas", "2026-04-07 10:00:00")
	if key != "gotchas:2026-04-07 10:00:00" {
		t.Errorf("MakeKey = %q, want gotchas:2026-04-07 10:00:00", key)
	}
}

func TestIndexFilePath(t *testing.T) {
	path := IndexFilePath("/some/.brain")
	if path != "/some/.brain/index.json" {
		t.Errorf("IndexFilePath = %q, want /some/.brain/index.json", path)
	}
}
