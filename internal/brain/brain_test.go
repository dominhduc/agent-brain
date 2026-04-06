package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/review"
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
	ResetCache()
	t.Cleanup(func() {
		os.Chdir(originalDir)
		ResetCache()
	})
	return brainDir
}

func TestAvailableTopics(t *testing.T) {
	topics := AvailableTopics()
	if len(topics) != 5 {
		t.Errorf("expected 5 topics, got %d", len(topics))
	}
	expected := []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
	for i, exp := range expected {
		if topics[i] != exp {
			t.Errorf("topics[%d] = %q, want %q", i, topics[i], exp)
		}
	}
}

func TestTopicFilePath_ValidTopics(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	tests := []struct {
		topic    string
		filename string
	}{
		{"memory", "MEMORY.md"},
		{"gotchas", "gotchas.md"},
		{"patterns", "patterns.md"},
		{"decisions", "decisions.md"},
		{"architecture", "architecture.md"},
		{"MEMORY", "MEMORY.md"},
		{"Gotchas", "gotchas.md"},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			path, err := TopicFilePath(tt.topic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expected := filepath.Join(brainDir, tt.filename)
			if path != expected {
				t.Errorf("TopicFilePath(%q) = %q, want %q", tt.topic, path, expected)
			}
		})
	}
}

func TestTopicFilePath_InvalidTopic(t *testing.T) {
	setupTestBrainDir(t)

	_, err := TopicFilePath("nonexistent")
	if err == nil {
		t.Error("expected error for invalid topic")
	}
}

func TestBrainDirExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".brain"), 0755)

	if !BrainDirExists(tmpDir) {
		t.Error("expected BrainDirExists to return true")
	}
}

func TestBrainDirExists_False(t *testing.T) {
	tmpDir := t.TempDir()

	if BrainDirExists(tmpDir) {
		t.Error("expected BrainDirExists to return false for dir without .brain/")
	}
}

func TestBrainDirExists_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := t.TempDir()
	os.MkdirAll(filepath.Join(target, ".brain"), 0755)
	os.Symlink(filepath.Join(target, ".brain"), filepath.Join(tmpDir, ".brain"))

	if BrainDirExists(tmpDir) {
		t.Error("expected BrainDirExists to return false for symlinked .brain/")
	}
}

func TestFindBrainDir_Symlink(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer func() {
		os.Chdir(originalDir)
		ResetCache()
	}()

	target := t.TempDir()
	os.MkdirAll(filepath.Join(target, ".brain"), 0755)
	linkDir := t.TempDir()
	os.Symlink(filepath.Join(target, ".brain"), filepath.Join(linkDir, ".brain"))
	os.Chdir(linkDir)
	ResetCache()

	_, err := FindBrainDir()
	if err == nil {
		t.Error("expected error when .brain is a symlink")
	}
}

func TestFindBrainDir_Found(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	found, err := FindBrainDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != brainDir {
		t.Errorf("FindBrainDir() = %q, want %q", found, brainDir)
	}
}

func TestFindBrainDir_NotFound(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer func() {
		os.Chdir(originalDir)
		ResetCache()
	}()

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	ResetCache()

	_, err := FindBrainDir()
	if err == nil {
		t.Error("expected error when no .brain/ exists")
	}
}

func TestAddEntry_GetTopic_RoundTrip(t *testing.T) {
	setupTestBrainDir(t)

	msg := "Always use argon2id, NOT bcrypt"
	if err := AddEntry("gotchas", msg); err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	content, err := GetTopic("gotchas")
	if err != nil {
		t.Fatalf("GetTopic failed: %v", err)
	}
	if !strings.Contains(content, msg) {
		t.Errorf("expected content to contain %q, got %q", msg, content)
	}
	if !strings.Contains(content, "### [") {
		t.Error("expected timestamp header in entry")
	}
}

func TestGetAllTopics(t *testing.T) {
	setupTestBrainDir(t)

	if err := AddEntry("gotchas", "test gotcha"); err != nil {
		t.Fatal(err)
	}
	if err := AddEntry("patterns", "test pattern"); err != nil {
		t.Fatal(err)
	}

	content, err := GetAllTopics()
	if err != nil {
		t.Fatalf("GetAllTopics failed: %v", err)
	}
	if !strings.Contains(content, "test gotcha") {
		t.Error("expected content to contain gotcha entry")
	}
	if !strings.Contains(content, "test pattern") {
		t.Error("expected content to contain pattern entry")
	}
	if !strings.Contains(content, "## MEMORY") {
		t.Error("expected MEMORY section header")
	}
}

func TestMemoryLineCount(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	os.WriteFile(filepath.Join(brainDir, "MEMORY.md"), []byte("line1\nline2\nline3\n"), 0600)

	count, err := MemoryLineCount()
	if err != nil {
		t.Fatalf("MemoryLineCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 lines, got %d", count)
	}
}

func TestMemoryLineCount_Empty(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	os.WriteFile(filepath.Join(brainDir, "MEMORY.md"), []byte(""), 0600)

	count, err := MemoryLineCount()
	if err != nil {
		t.Fatalf("MemoryLineCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 lines for empty file, got %d", count)
	}
}

func TestAddEntry_MultipleEntries(t *testing.T) {
	setupTestBrainDir(t)

	for i := 0; i < 5; i++ {
		if err := AddEntry("patterns", fmt.Sprintf("pattern entry %d", i)); err != nil {
			t.Fatal(err)
		}
	}

	content, err := GetTopic("patterns")
	if err != nil {
		t.Fatal(err)
	}
	occurrences := strings.Count(content, "pattern entry")
	if occurrences != 5 {
		t.Errorf("expected 5 occurrences, got %d", occurrences)
	}
}

func TestAddEntry_Deduplication(t *testing.T) {
	setupTestBrainDir(t)

	if err := AddEntry("gotchas", "Test deduplication entry"); err != nil {
		t.Fatal(err)
	}

	if err := AddEntry("gotchas", "Test deduplication entry"); err != nil {
		t.Fatal(err)
	}

	content, err := GetTopic("gotchas")
	if err != nil {
		t.Fatal(err)
	}
	occurrences := strings.Count(content, "Test deduplication entry")
	if occurrences != 1 {
		t.Errorf("expected 1 occurrence (deduplication), got %d", occurrences)
	}
}

func TestResetCache(t *testing.T) {
	setupTestBrainDir(t)

	_, err := FindBrainDir()
	if err != nil {
		t.Fatal(err)
	}

	ResetCache()

	_, err = FindBrainDir()
	if err != nil {
		t.Fatalf("FindBrainDir after ResetCache failed: %v", err)
	}
}

func TestPendingDir(t *testing.T) {
	tmpDir := t.TempDir()
	got := PendingDir(tmpDir)
	want := filepath.Join(tmpDir, ".brain", "pending")
	if got != want {
		t.Errorf("PendingDir() = %q, want %q", got, want)
	}
}

func TestAddPendingEntry(t *testing.T) {
	tmpDir := t.TempDir()

	entry := review.PendingEntry{
		ID:        "test-entry",
		Topic:     "gotchas",
		Content:   "Test pending entry",
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	if err := AddPendingEntry(tmpDir, entry); err != nil {
		t.Fatalf("AddPendingEntry failed: %v", err)
	}

	pendingDir := PendingDir(tmpDir)
	loaded, err := review.LoadPendingEntries(pendingDir)
	if err != nil {
		t.Fatalf("LoadPendingEntries failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	if loaded[0].ID != "test-entry" {
		t.Errorf("ID = %q, want %q", loaded[0].ID, "test-entry")
	}
	if loaded[0].Topic != "gotchas" {
		t.Errorf("Topic = %q, want %q", loaded[0].Topic, "gotchas")
	}
	if loaded[0].Content != "Test pending entry" {
		t.Errorf("Content = %q, want %q", loaded[0].Content, "Test pending entry")
	}
}

func TestResetCache_ConcurrentAccess(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			ResetCache()
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			found, err := FindBrainDir()
			if err == nil && found != brainDir {
				t.Errorf("FindBrainDir returned wrong path: %s", found)
			}
		}
		done <- true
	}()
	<-done
	<-done
}
