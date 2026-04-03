package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPendingEntryJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := PendingEntry{
		ID:         "test-123",
		Topic:      "gotchas",
		Content:    "Always use filepath.Join",
		CommitSHA:  "abc123def",
		Timestamp:  now,
		Confidence: "high",
		Source:     "daemon",
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}

	var decoded PendingEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Topic != original.Topic {
		t.Errorf("Topic = %q, want %q", decoded.Topic, original.Topic)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, original.Content)
	}
	if decoded.CommitSHA != original.CommitSHA {
		t.Errorf("CommitSHA = %q, want %q", decoded.CommitSHA, original.CommitSHA)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if decoded.Confidence != original.Confidence {
		t.Errorf("Confidence = %q, want %q", decoded.Confidence, original.Confidence)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, original.Source)
	}
}

func TestLoadPendingEntries_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	entries, err := LoadPendingEntries(tmpDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadPendingEntries_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	pendingDir := filepath.Join(tmpDir, "nonexistent")

	entries, err := LoadPendingEntries(pendingDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil for nonexistent dir, got %v", entries)
	}
}

func TestLoadPendingEntries_SingleEntry(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	entry := PendingEntry{
		ID:        "entry-1",
		Topic:     "patterns",
		Content:   "Use TDD for internal packages",
		Timestamp: now,
	}

	if err := SavePendingEntry(tmpDir, entry); err != nil {
		t.Fatalf("SavePendingEntry failed: %v", err)
	}

	loaded, err := LoadPendingEntries(tmpDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	if loaded[0].ID != "entry-1" {
		t.Errorf("ID = %q, want %q", loaded[0].ID, "entry-1")
	}
	if loaded[0].Topic != "patterns" {
		t.Errorf("Topic = %q, want %q", loaded[0].Topic, "patterns")
	}
	if loaded[0].Content != "Use TDD for internal packages" {
		t.Errorf("Content = %q, want %q", loaded[0].Content, "Use TDD for internal packages")
	}
}

func TestLoadPendingEntries_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		entry := PendingEntry{
			ID:        "entry-" + strings.Repeat("a", i+1),
			Topic:     "gotchas",
			Content:   "Gotcha number " + string(rune('0'+i+1)),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		}
		if err := SavePendingEntry(tmpDir, entry); err != nil {
			t.Fatalf("SavePendingEntry failed: %v", err)
		}
	}

	loaded, err := LoadPendingEntries(tmpDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(loaded))
	}

	for i := 1; i < len(loaded); i++ {
		if loaded[i].Timestamp.Before(loaded[i-1].Timestamp) {
			t.Errorf("entries not sorted by timestamp: [%d] = %v before [%d] = %v",
				i-1, loaded[i-1].Timestamp, i, loaded[i].Timestamp)
		}
	}
}

func TestLoadPendingEntries_IgnoresNonJSON(t *testing.T) {
	tmpDir := t.TempDir()

	if err := SavePendingEntry(tmpDir, PendingEntry{
		ID:        "valid-1",
		Topic:     "patterns",
		Content:   "valid entry",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(tmpDir, "not-json.txt"), []byte("not json"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("{invalid json}"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "empty-id.json"), []byte(`{"id":"","topic":"x"}`), 0600)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	loaded, err := LoadPendingEntries(tmpDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 valid entry, got %d", len(loaded))
	}
	if loaded[0].ID != "valid-1" {
		t.Errorf("ID = %q, want %q", loaded[0].ID, "valid-1")
	}
}

func TestSavePendingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	pendingDir := filepath.Join(tmpDir, "pending")

	entry := PendingEntry{
		ID:        "save-1",
		Topic:     "decisions",
		Content:   "Use stdlib only",
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	if err := SavePendingEntry(pendingDir, entry); err != nil {
		t.Fatalf("SavePendingEntry failed: %v", err)
	}

	expectedPath := filepath.Join(pendingDir, "save-1.json")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("file not created at expected path: %v", err)
	}

	var loaded PendingEntry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved file: %v", err)
	}
	if loaded.ID != "save-1" {
		t.Errorf("ID = %q, want %q", loaded.ID, "save-1")
	}
}

func TestRemovePendingEntry(t *testing.T) {
	tmpDir := t.TempDir()

	entry := PendingEntry{
		ID:        "remove-1",
		Topic:     "gotchas",
		Content:   "to be removed",
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}
	if err := SavePendingEntry(tmpDir, entry); err != nil {
		t.Fatal(err)
	}

	if err := RemovePendingEntry(tmpDir, "remove-1"); err != nil {
		t.Fatalf("RemovePendingEntry failed: %v", err)
	}

	loaded, err := LoadPendingEntries(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 entries after removal, got %d", len(loaded))
	}
}

func TestRemovePendingEntry_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	err := RemovePendingEntry(tmpDir, "nonexistent")
	if err == nil {
		t.Error("expected error when removing nonexistent entry")
	}
}

func TestFingerprint_SameContent(t *testing.T) {
	e1 := PendingEntry{Topic: "gotchas", Content: "Always use filepath.Join"}
	e2 := PendingEntry{Topic: "gotchas", Content: "  always use filepath.join  "}

	if e1.Fingerprint() != e2.Fingerprint() {
		t.Errorf("expected same fingerprint for equivalent content, got %q vs %q",
			e1.Fingerprint(), e2.Fingerprint())
	}
}

func TestFingerprint_DifferentContent(t *testing.T) {
	e1 := PendingEntry{Topic: "gotchas", Content: "Use filepath.Join"}
	e2 := PendingEntry{Topic: "gotchas", Content: "Use path.Join"}

	if e1.Fingerprint() == e2.Fingerprint() {
		t.Errorf("expected different fingerprints for different content")
	}
}

func TestFingerprint_DifferentTopics(t *testing.T) {
	e1 := PendingEntry{Topic: "gotchas", Content: "Same content"}
	e2 := PendingEntry{Topic: "patterns", Content: "Same content"}

	if e1.Fingerprint() == e2.Fingerprint() {
		t.Errorf("expected different fingerprints for different topics")
	}
}

func TestDisplayTime(t *testing.T) {
	now, _ := time.Parse("2006-01-02 15:04:05", "2026-04-03 14:30:00")
	e := PendingEntry{Timestamp: now}

	got := e.DisplayTime()
	want := "2026-04-03 14:30"
	if got != want {
		t.Errorf("DisplayTime() = %q, want %q", got, want)
	}
}

func TestGroupByTopic(t *testing.T) {
	now := time.Now().UTC()
	entries := []PendingEntry{
		{ID: "1", Topic: "gotchas", Content: "g1", Timestamp: now},
		{ID: "2", Topic: "patterns", Content: "p1", Timestamp: now},
		{ID: "3", Topic: "gotchas", Content: "g2", Timestamp: now},
		{ID: "4", Topic: "patterns", Content: "p2", Timestamp: now},
		{ID: "5", Topic: "patterns", Content: "p3", Timestamp: now},
	}

	groups := GroupByTopic(entries)

	if len(groups["gotchas"]) != 2 {
		t.Errorf("gotchas group = %d entries, want 2", len(groups["gotchas"]))
	}
	if len(groups["patterns"]) != 3 {
		t.Errorf("patterns group = %d entries, want 3", len(groups["patterns"]))
	}
	if _, ok := groups["decisions"]; ok {
		t.Error("decisions group should not exist")
	}
}

func TestCountByTopic(t *testing.T) {
	now := time.Now().UTC()
	entries := []PendingEntry{
		{ID: "1", Topic: "gotchas", Content: "g1", Timestamp: now},
		{ID: "2", Topic: "gotchas", Content: "g2", Timestamp: now},
		{ID: "3", Topic: "gotchas", Content: "g3", Timestamp: now},
		{ID: "4", Topic: "patterns", Content: "p1", Timestamp: now},
	}

	counts := CountByTopic(entries)

	if counts["gotchas"] != 3 {
		t.Errorf("gotchas count = %d, want 3", counts["gotchas"])
	}
	if counts["patterns"] != 1 {
		t.Errorf("patterns count = %d, want 1", counts["patterns"])
	}
}

func TestFindDuplicateGroups(t *testing.T) {
	now := time.Now().UTC()

	entries := []PendingEntry{
		{ID: "1", Topic: "gotchas", Content: "Use filepath.Join", Timestamp: now},
		{ID: "2", Topic: "gotchas", Content: "  use filepath.join  ", Timestamp: now},
		{ID: "3", Topic: "patterns", Content: "Use TDD", Timestamp: now},
		{ID: "4", Topic: "patterns", Content: "Use TDD", Timestamp: now},
		{ID: "5", Topic: "patterns", Content: "Use TDD", Timestamp: now},
		{ID: "6", Topic: "decisions", Content: "Unique item", Timestamp: now},
	}

	groups := FindDuplicateGroups(entries)

	if len(groups) != 2 {
		t.Fatalf("expected 2 duplicate groups, got %d", len(groups))
	}

	if len(groups[0].Entries) != 3 {
		t.Errorf("first group should have 3 entries, got %d", len(groups[0].Entries))
	}
	if groups[0].Fingerprint != entries[2].Fingerprint() {
		t.Errorf("first group fingerprint mismatch")
	}

	if len(groups[1].Entries) != 2 {
		t.Errorf("second group should have 2 entries, got %d", len(groups[1].Entries))
	}
	if groups[1].Representative != "Use filepath.Join" {
		t.Errorf("second group representative = %q, want %q", groups[1].Representative, "Use filepath.Join")
	}
}

func TestFindDuplicateGroups_NoDuplicates(t *testing.T) {
	now := time.Now().UTC()
	entries := []PendingEntry{
		{ID: "1", Topic: "gotchas", Content: "a", Timestamp: now},
		{ID: "2", Topic: "patterns", Content: "b", Timestamp: now},
		{ID: "3", Topic: "decisions", Content: "c", Timestamp: now},
	}

	groups := FindDuplicateGroups(entries)
	if len(groups) != 0 {
		t.Errorf("expected 0 duplicate groups, got %d", len(groups))
	}
}

func TestFindDuplicateGroups_Empty(t *testing.T) {
	groups := FindDuplicateGroups(nil)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for nil input, got %d", len(groups))
	}
}
