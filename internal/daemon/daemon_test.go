package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func TestProcessItem_ValidItem(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	queueDir := filepath.Join(brainDir, ".queue")
	os.MkdirAll(filepath.Join(queueDir, "done"), 0755)
	os.MkdirAll(filepath.Join(brainDir, "sessions"), 0755)
	for _, name := range []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"} {
		os.WriteFile(filepath.Join(brainDir, name), []byte("# "+name+"\n"), 0600)
	}

	item := QueueItem{
		Timestamp: "20260402T143000",
		Repo:      tmpDir,
		DiffStat:  "1 file changed",
		Files:     "A\tmain.go",
		Attempts:  0,
	}

	itemData, _ := json.Marshal(item)
	itemPath := filepath.Join(queueDir, "commit-test.json")
	processingPath := itemPath + ".processing"
	os.WriteFile(itemPath, itemData, 0600)

	err := os.Rename(itemPath, processingPath)
	if err != nil {
		t.Fatal(err)
	}

	mockDiff := func(repo string) (string, error) {
		return "diff --git a/main.go b/main.go\n+func main() {}", nil
	}
	mockAnalyze := func(req AnalyzeRequest) (Finding, error) {
		return Finding{
			Gotchas:    []string{"test gotcha"},
			Confidence: "HIGH",
		}, nil
	}
	processed, err := ProcessItemWithDeps(context.Background(), processingPath, queueDir, brainDir, tmpDir, 3, mockDiff, mockAnalyze)
	if err != nil {
		t.Fatalf("ProcessItem failed: %v", err)
	}
	if !processed {
		t.Error("expected item to be processed")
	}

	if _, err := os.Stat(processingPath); !os.IsNotExist(err) {
		t.Error("expected processing file to be moved away")
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entries, err := knowledge.LoadPendingEntries(pendingDir)
	if err != nil {
		t.Fatalf("loading pending entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 pending entry, got %d", len(entries))
	}
	if entries[0].Topic != "gotchas" {
		t.Errorf("expected topic gotchas, got %s", entries[0].Topic)
	}
	if entries[0].Content != "test gotcha" {
		t.Errorf("expected content 'test gotcha', got %q", entries[0].Content)
	}
	if entries[0].Confidence != "HIGH" {
		t.Errorf("expected confidence HIGH, got %s", entries[0].Confidence)
	}
	if entries[0].Source != "daemon" {
		t.Errorf("expected source daemon, got %s", entries[0].Source)
	}
}

func TestProcessItem_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	queueDir := filepath.Join(tmpDir, ".queue")
	os.MkdirAll(filepath.Join(queueDir, "failed"), 0755)

	processingPath := filepath.Join(queueDir, "commit-bad.json.processing")
	os.WriteFile(processingPath, []byte("not json"), 0600)

	processed, err := ProcessItemWithDeps(context.Background(), processingPath, queueDir, tmpDir, tmpDir, 3, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if processed {
		t.Error("expected item to NOT be processed")
	}
}

func TestProcessItem_WrongRepo(t *testing.T) {
	tmpDir := t.TempDir()
	queueDir := filepath.Join(tmpDir, ".queue")
	os.MkdirAll(filepath.Join(queueDir, "failed"), 0755)

	item := QueueItem{
		Timestamp: "20260402T143000",
		Repo:      "/some/other/path",
	}
	itemData, _ := json.Marshal(item)
	processingPath := filepath.Join(queueDir, "commit-wrong.json.processing")
	os.WriteFile(processingPath, itemData, 0600)

	processed, _ := ProcessItemWithDeps(context.Background(), processingPath, queueDir, tmpDir, tmpDir, 3, nil, nil)
	if processed {
		t.Error("expected item to be rejected for wrong repo")
	}
}

func TestProcessItem_EmptyTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	queueDir := filepath.Join(tmpDir, ".queue")
	os.MkdirAll(filepath.Join(queueDir, "failed"), 0755)

	item := QueueItem{
		Timestamp: "",
		Repo:      tmpDir,
	}
	itemData, _ := json.Marshal(item)
	processingPath := filepath.Join(queueDir, "commit-empty.json.processing")
	os.WriteFile(processingPath, itemData, 0600)

	processed, _ := ProcessItemWithDeps(context.Background(), processingPath, queueDir, tmpDir, tmpDir, 3, nil, nil)
	if processed {
		t.Error("expected item to be rejected for empty timestamp")
	}
}

func TestParsePollInterval(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"5s", 5 * time.Second},
		{"30s", 30 * time.Second},
		{"1m", 1 * time.Minute},
		{"0s", 5 * time.Second},
		{"not-a-duration", 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParsePollInterval(tt.input)
			if got != tt.want {
				t.Errorf("ParsePollInterval(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalcBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 5 * time.Second},
		{2, 20 * time.Second},
		{3, 45 * time.Second},
	}

	for _, tt := range tests {
		got := CalcBackoff(tt.attempt)
		if got != tt.want {
			t.Errorf("CalcBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}
