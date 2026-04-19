package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestHub(t *testing.T) *Hub {
	t.Helper()
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"} {
		if err := os.WriteFile(filepath.Join(brainDir, filename), []byte("# "+filename+"\n\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	hub, err := Open(brainDir)
	if err != nil {
		t.Fatal(err)
	}
	return hub
}

func TestOpen(t *testing.T) {
	hub := setupTestHub(t)
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.dir == "" {
		t.Fatal("expected hub.dir to be set")
	}
}

func TestAvailableTopics(t *testing.T) {
	topics := AvailableTopics()
	expected := []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
	if len(topics) != len(expected) {
		t.Fatalf("expected %d topics, got %d", len(expected), len(topics))
	}
	for i, topic := range topics {
		if topic != expected[i] {
			t.Errorf("expected topic[%d] = %s, got %s", i, expected[i], topic)
		}
	}
}

func TestAddAndGet(t *testing.T) {
	hub := setupTestHub(t)

	_, err := hub.Add("gotchas", "Test gotcha message")
	if err != nil {
		t.Fatal(err)
	}

	content, err := hub.Get("gotchas")
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}
	found := false
	for _, line := range splitLines(content) {
		if contains(line, "Test gotcha message") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find added entry in content")
	}
}

func TestAddDuplicate(t *testing.T) {
	hub := setupTestHub(t)

	_, err := hub.Add("gotchas", "Duplicate test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = hub.Add("gotchas", "Duplicate test")
	if err != nil {
		t.Fatal(err)
	}

	content, err := hub.Get("gotchas")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range splitLines(content) {
		if contains(line, "Duplicate test") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 occurrence, got %d", count)
	}
}

func TestAddEntryFuzzyDuplicate(t *testing.T) {
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		t.Fatal(err)
	}
	topicPath := filepath.Join(brainDir, "gotchas.md")
	content := "# gotchas.md\n\n### [2025-01-01 10:00:00] Go const cannot be overridden by ldflags - use var instead\n\n"
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chdir(originalDir)
		ResetCache()
	})

	dup, err := AddEntry("gotchas", "Go const cannot be overridden by ldflags - use var instead")
	if err != nil {
		t.Fatal(err)
	}
	if dup != true {
		t.Error("expected AddEntry to return true (duplicate) for near-duplicate entry")
	}

	data, _ := os.ReadFile(topicPath)
	count := strings.Count(string(data), "Go const cannot be overridden by ldflags")
	if count != 1 {
		t.Errorf("expected 1 occurrence of ldflags entry, got %d", count)
	}
}

func TestAddEntryNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		t.Fatal(err)
	}
	topicPath := filepath.Join(brainDir, "gotchas.md")
	content := "# gotchas.md\n\n### [2025-01-01 10:00:00] Completely different entry about something\n\n"
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chdir(originalDir)
		ResetCache()
	})

	dup, err := AddEntry("gotchas", "Go const cannot be overridden by ldflags - use var instead")
	if err != nil {
		t.Fatal(err)
	}
	if dup != false {
		t.Error("expected AddEntry to return false (new entry) for unrelated content")
	}
}

func TestGetAll(t *testing.T) {
	hub := setupTestHub(t)
	_, _ = hub.Add("gotchas", "Test gotcha")
	_, _ = hub.Add("patterns", "Test pattern")

	content, err := hub.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}
	if !contains(content, "Test gotcha") {
		t.Fatal("expected to find gotcha in all topics")
	}
	if !contains(content, "Test pattern") {
		t.Fatal("expected to find pattern in all topics")
	}
}

func TestGetSummary(t *testing.T) {
	hub := setupTestHub(t)
	_, _ = hub.Add("gotchas", "Entry one")
	_, _ = hub.Add("gotchas", "Entry two")

	summary, err := hub.GetSummary("gotchas")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Name != "gotchas" {
		t.Errorf("expected name=gotchas, got %s", summary.Name)
	}
	if summary.EntryCount != 2 {
		t.Errorf("expected 2 entries, got %d", summary.EntryCount)
	}
}

func TestGetAllSummaries(t *testing.T) {
	hub := setupTestHub(t)
	_, _ = hub.Add("gotchas", "A gotcha")
	_, _ = hub.Add("patterns", "A pattern")

	summaries, err := hub.GetAllSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 5 {
		t.Fatalf("expected 5 summaries, got %d", len(summaries))
	}
}

func TestMemoryLineCount(t *testing.T) {
	hub := setupTestHub(t)
	count, err := hub.MemoryLineCount()
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected non-zero line count for MEMORY.md with header")
	}
}

func TestWorkingMemory(t *testing.T) {
	hub := setupTestHub(t)

	err := hub.PushWM("temp note", 0.8)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := hub.ReadWM()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "temp note" {
		t.Errorf("expected 'temp note', got %s", entries[0].Content)
	}

	err = hub.ClearWM()
	if err != nil {
		t.Fatal(err)
	}

	entries, err = hub.ReadWM()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestWorkingMemoryMax(t *testing.T) {
	hub := setupTestHub(t)

	for i := 0; i < 25; i++ {
		err := hub.PushWM("note", float64(i)/25.0)
		if err != nil {
			t.Fatal(err)
		}
	}

	entries, err := hub.ReadWM()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 20 {
		t.Fatalf("expected max 20 entries, got %d", len(entries))
	}
}

func TestPendingEntries(t *testing.T) {
	hub := setupTestHub(t)

	entry := PendingEntry{
		ID:         "test-001",
		Topic:      "gotchas",
		Content:    "A pending gotcha",
		Timestamp:  time.Now(),
		Confidence: "HIGH",
		Source:     "daemon",
	}

	err := hub.AddPending(entry)
	if err != nil {
		t.Fatal(err)
	}

	pending, err := hub.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending entry, got %d", len(pending))
	}
	if pending[0].ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", pending[0].ID)
	}

	err = hub.RemovePending("test-001")
	if err != nil {
		t.Fatal(err)
	}

	pending, err = hub.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after removal, got %d", len(pending))
	}
}

func TestPendingFingerprint(t *testing.T) {
	e1 := PendingEntry{Topic: "gotchas", Content: "Test message"}
	e2 := PendingEntry{Topic: "gotchas", Content: "Test message"}
	e3 := PendingEntry{Topic: "patterns", Content: "Test message"}

	if e1.Fingerprint() != e2.Fingerprint() {
		t.Fatal("same topic+content should have same fingerprint")
	}
	if e1.Fingerprint() == e3.Fingerprint() {
		t.Fatal("different topics should have different fingerprints")
	}
}

func TestRetrievalTracking(t *testing.T) {
	hub := setupTestHub(t)

	keys := []string{"gotchas:2026-04-03 08:42:10", "patterns:2026-04-03 09:00:00"}
	err := hub.RecordRetrieval(keys)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := hub.LoadRetrievals()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(loaded))
	}

	err = hub.ClearRetrievals()
	if err != nil {
		t.Fatal(err)
	}

	loaded, err = hub.LoadRetrievals()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 after clear, got %d", len(loaded))
	}
}

func TestDedup(t *testing.T) {
	hub := setupTestHub(t)

	path, _ := hub.topicFilePath("gotchas")
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	f.WriteString("\n### [2026-04-03 10:00:00] Same message for dedup test\n\n")
	f.WriteString("\n### [2026-04-03 11:00:00] Same message for dedup test\n\n")
	f.Close()

	groups, err := hub.FindDuplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) == 0 {
		t.Fatal("expected to find duplicate groups")
	}

	report, err := hub.RunDedup(true)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRemoved == 0 {
		t.Fatal("expected non-zero removals in dry-run")
	}

	content, _ := hub.Get("gotchas")
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "Same message for dedup test") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("dry-run should not modify files, expected 2 occurrences, got %d", count)
	}

	report, err = hub.RunDedup(false)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRemoved == 0 {
		t.Fatal("expected non-zero removals in real run")
	}

	content, _ = hub.Get("gotchas")
	count = 0
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "Same message for dedup test") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("real run should deduplicate, expected 1 occurrence, got %d", count)
	}
}

func TestIndexLoadSave(t *testing.T) {
	ResetIndexCache()
	hub := setupTestHub(t)

	idx, err := hub.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Entries) != 0 {
		t.Fatalf("expected empty index, got %d entries", len(idx.Entries))
	}

	idx.Set("gotchas", "2026-04-03 08:42:10", IndexEntry{
		Strength:       0.9,
		RetrievalCount: 3,
		HalfLifeDays:   14,
		Confidence:     "HIGH",
	})

	err = hub.SaveIndex(idx)
	if err != nil {
		t.Fatal(err)
	}

	idx2, err := hub.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := idx2.Get("gotchas", "2026-04-03 08:42:10")
	if !ok {
		t.Fatal("expected to find saved entry")
	}
	if entry.Strength != 0.9 {
		t.Errorf("expected strength 0.9, got %f", entry.Strength)
	}
	if entry.RetrievalCount != 3 {
		t.Errorf("expected retrieval count 3, got %d", entry.RetrievalCount)
	}
}

func TestIndexRebuild(t *testing.T) {
	hub := setupTestHub(t)
	_, _ = hub.Add("gotchas", "First gotcha")
	_, _ = hub.Add("patterns", "First pattern")

	idx, err := hub.RebuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Entries) < 2 {
		t.Fatalf("expected at least 2 entries after rebuild, got %d", len(idx.Entries))
	}
}

func TestCalculateStrength(t *testing.T) {
	now := time.Now()

	fresh := IndexEntry{
		HalfLifeDays:   7,
		RetrievalCount: 5,
		LastRetrieved:  now,
		Confidence:     "HIGH",
	}
	s := CalculateStrength(fresh, now)
	if s <= 0 || s > 1 {
		t.Errorf("expected strength in (0,1], got %f", s)
	}

	old := IndexEntry{
		HalfLifeDays:   7,
		RetrievalCount: 0,
		LastRetrieved:  now.Add(-30 * 24 * time.Hour),
		Confidence:     "MEDIUM",
	}
	s2 := CalculateStrength(old, now)
	if s2 >= s {
		t.Errorf("old entry should have lower strength: old=%f, fresh=%f", s2, s)
	}
}

func TestDetectTopics(t *testing.T) {
	topics := DetectTopics("React component with Tailwind CSS styling")
	if !containsTopic(topics, "ui") {
		t.Errorf("expected 'ui' topic for React content, got %v", topics)
	}

	topics2 := DetectTopics("API handler with middleware chain")
	if !containsTopic(topics2, "backend") {
		t.Errorf("expected 'backend' topic for API content, got %v", topics2)
	}

	topics3 := DetectTopics("random unrelated text")
	if !containsTopic(topics3, "general") {
		t.Errorf("expected 'general' fallback, got %v", topics3)
	}
}

func TestGetAllWithSummary(t *testing.T) {
	hub := setupTestHub(t)
	_, _ = hub.Add("gotchas", "A gotcha entry")

	content, err := hub.GetAllWithSummary()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(content, "PROJECT MEMORY SUMMARY") {
		t.Fatal("expected summary header")
	}
	if !contains(content, "A gotcha entry") {
		t.Fatal("expected gotcha content in summary")
	}
}

func TestGroupByTopic(t *testing.T) {
	entries := []PendingEntry{
		{ID: "1", Topic: "gotchas", Content: "A"},
		{ID: "2", Topic: "patterns", Content: "B"},
		{ID: "3", Topic: "gotchas", Content: "C"},
	}
	groups := GroupByTopic(entries)
	if len(groups["gotchas"]) != 2 {
		t.Fatalf("expected 2 gotchas, got %d", len(groups["gotchas"]))
	}
	if len(groups["patterns"]) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(groups["patterns"]))
	}
}

func TestEnsureBrainDir(t *testing.T) {
	dir := t.TempDir()
	err := EnsureBrainDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !BrainDirExists(dir) {
		t.Fatal("expected .brain dir to exist")
	}
}

func TestFindBrainDir(t *testing.T) {
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	os.MkdirAll(brainDir, 0755)

	result, err := FindBrainDirFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result != brainDir {
		t.Errorf("expected %s, got %s", brainDir, result)
	}
}

func TestFindBrainDirWalksUp(t *testing.T) {
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	os.MkdirAll(brainDir, 0755)
	subDir := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(subDir, 0755)

	result, err := FindBrainDirFrom(subDir)
	if err != nil {
		t.Fatal(err)
	}
	if result != brainDir {
		t.Errorf("expected %s, got %s", brainDir, result)
	}
}

func TestFindBrainDirNotFound(t *testing.T) {
	_, err := FindBrainDirFrom(t.TempDir())
	if err == nil {
		t.Fatal("expected error when .brain not found")
	}
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsTopic(topics []string, target string) bool {
	for _, t := range topics {
		if t == target {
			return true
		}
	}
	return false
}

var _ = contains
var _ = containsSubstr
var _ = containsTopic
var _ = splitLines
