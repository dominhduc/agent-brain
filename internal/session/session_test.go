package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestSession(t *testing.T) *Session {
	t.Helper()
	dir := t.TempDir()
	brainDir := filepath.Join(dir, ".brain")
	os.MkdirAll(brainDir, 0755)
	os.MkdirAll(filepath.Join(brainDir, "sessions"), 0755)
	os.MkdirAll(filepath.Join(brainDir, "handoffs"), 0755)
	return Open(brainDir)
}

func TestOpen(t *testing.T) {
	s := setupTestSession(t)
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.Dir() == "" {
		t.Fatal("expected non-empty dir")
	}
}

func TestCreateHandoff(t *testing.T) {
	s := setupTestSession(t)

	h, err := s.CreateHandoff("test summary", "next steps", "sess-001", "backend")
	if err != nil {
		t.Fatal(err)
	}
	if h == nil {
		t.Fatal("expected non-nil handoff")
	}
	if h.Summary != "test summary" {
		t.Errorf("expected 'test summary', got %s", h.Summary)
	}
	if h.Topic != "backend" {
		t.Errorf("expected 'backend', got %s", h.Topic)
	}
}

func TestLatestHandoff(t *testing.T) {
	s := setupTestSession(t)

	h, err := s.LatestHandoff()
	if err != nil {
		t.Fatal(err)
	}
	if h != nil {
		t.Fatal("expected nil when no handoff exists")
	}

	s.CreateHandoff("first", "next", "sess-001", "general")
	h, err = s.LatestHandoff()
	if err != nil {
		t.Fatal(err)
	}
	if h == nil {
		t.Fatal("expected handoff")
	}
	if h.Summary != "first" {
		t.Errorf("expected 'first', got %s", h.Summary)
	}
}

func TestResumeHandoff(t *testing.T) {
	s := setupTestSession(t)
	s.CreateHandoff("resume me", "do stuff", "sess-001", "testing")

	h, err := s.ResumeHandoff()
	if err != nil {
		t.Fatal(err)
	}
	if h == nil || h.Summary != "resume me" {
		t.Fatal("ResumeHandoff should return latest handoff")
	}
}

func TestOverwriteLatestHandoff(t *testing.T) {
	s := setupTestSession(t)

	s.CreateHandoff("first", "next", "sess-001", "general")
	s.CreateHandoff("second", "next2", "sess-002", "backend")

	h, _ := s.LatestHandoff()
	if h.Summary != "second" {
		t.Errorf("expected latest to be 'second', got %s", h.Summary)
	}
}

func TestCreateSessionFile(t *testing.T) {
	s := setupTestSession(t)

	stats := &GitStats{
		ShortStat: "2 files changed",
		Log:       "abc123 test commit",
		Created:   []string{"foo.go"},
		Modified:  []string{"bar.go"},
		Deleted:   []string{},
	}

	path, err := s.CreateSessionFile(stats)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected non-empty session path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "foo.go") {
		t.Error("expected session file to contain created file")
	}
	if !strings.Contains(content, "bar.go") {
		t.Error("expected session file to contain modified file")
	}
}

func TestFormatList(t *testing.T) {
	if formatList(nil) != "none" {
		t.Error("expected 'none' for nil list")
	}
	if formatList([]string{}) != "none" {
		t.Error("expected 'none' for empty list")
	}
	result := formatList([]string{"a.go", "b.go"})
	if !strings.Contains(result, "a.go") || !strings.Contains(result, "b.go") {
		t.Errorf("expected both files in output, got %s", result)
	}
}
