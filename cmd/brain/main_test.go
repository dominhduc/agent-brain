package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git", "hooks"), 0755)
	brainDir := filepath.Join(tmpDir, ".brain")
	os.MkdirAll(filepath.Join(brainDir, ".queue", "done"), 0755)
	os.MkdirAll(filepath.Join(brainDir, "pending"), 0755)
	os.MkdirAll(filepath.Join(brainDir, "sessions"), 0755)
	for _, name := range []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"} {
		os.WriteFile(filepath.Join(brainDir, name), []byte("# "+name+"\n"), 0600)
	}
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() {
		os.Chdir(originalDir)
		knowledge.ResetCache()
	})
	return tmpDir
}

func captureStdout(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	data := make([]byte, 4096)
	for {
		n, err := r.Read(data)
		if n > 0 {
			buf.Write(data[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String()
}

func TestCmdDoctor_WithPendingEntries(t *testing.T) {
	setupTestProject(t)

	for i := 0; i < 3; i++ {
		entry := knowledge.PendingEntry{
			ID:         "test-00" + string(rune('0'+i)),
			Topic:      "gotchas",
			Content:    "Test gotcha " + string(rune('0'+i)),
			CommitSHA:  "abc123",
			Timestamp:  time.Now(),
			Confidence: "HIGH",
			Source:     "daemon",
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(filepath.Join(".brain", "pending", entry.ID+".json"), data, 0600)
	}

	oldArgs := os.Args
	os.Args = []string{"brain", "doctor", "--json"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdDoctor(true, false) })

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, output)
	}
	count, ok := status["pending_entries"].(float64)
	if !ok || int(count) != 3 {
		t.Errorf("expected 3 pending entries, got %v", status["pending_entries"])
	}
}

func TestCmdDoctor_NoPendingEntries(t *testing.T) {
	setupTestProject(t)

	oldArgs := os.Args
	os.Args = []string{"brain", "doctor", "--json"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdDoctor(true, false) })

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, output)
	}
	count, ok := status["pending_entries"].(float64)
	if !ok || int(count) != 0 {
		t.Errorf("expected 0 pending entries, got %v", status["pending_entries"])
	}
}

func TestCmdGet_FocusWithoutTopic(t *testing.T) {
	setupTestProject(t)

	oldArgs := os.Args
	os.Args = []string{"brain", "get", "--focus", "security"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdGet(false, false, false, false, false) })

	if strings.Contains(output, "No results for") {
		t.Error("should not search for '--focus'; should treat as focus flag with implicit 'all' topic")
	}
}

func TestCmdGet_WorkingMemory(t *testing.T) {
	tmpDir := setupTestProject(t)
	brainDir := filepath.Join(tmpDir, ".brain")

	knowledge.PushWM(brainDir, "test wm entry", 0.7)

	oldArgs := os.Args
	os.Args = []string{"brain", "get", "wm"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdGet(false, false, false, false, false) })

	if !strings.Contains(output, "test wm entry") {
		t.Errorf("expected output to contain working memory entry, got: %s", output)
	}
}
