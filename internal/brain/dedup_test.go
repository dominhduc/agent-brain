package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentFingerprint(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		same bool
	}{
		{
			name: "same content different case",
			a:    "Use filepath.Join for path construction",
			b:    "use filepath.join for path construction",
			same: true,
		},
		{
			name: "same content extra whitespace",
			a:    "Use filepath.Join  for  path construction",
			b:    "Use filepath.Join for path construction",
			same: true,
		},
		{
			name: "different content",
			a:    "Use filepath.Join",
			b:    "Use os.Open",
			same: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fpA := contentFingerprint(tc.a)
			fpB := contentFingerprint(tc.b)
			if tc.same {
				if fpA != fpB {
					t.Errorf("expected same fingerprint, got %s and %s", fpA, fpB)
				}
			} else {
				if fpA == fpB {
					t.Errorf("expected different fingerprint, both got %s", fpA)
				}
			}
		})
	}
}

func TestFindDuplicates_NoDuplicates(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	topics := []string{"gotchas", "patterns", "decisions", "architecture", "memory"}
	for _, topic := range topics {
		path := filepath.Join(brainDir, topic+".md")
		content := "### [2026-01-01 00:00:00] Unique entry for " + topic + "\n"
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	groups, err := FindDuplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestFindDuplicates_WithinTopic(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	content := `### [2026-01-01 00:00:00] Use filepath.Join

### [2026-01-01 00:00:01] Use os.Open

### [2026-01-01 00:00:02] use filepath.join
`
	path := filepath.Join(brainDir, "gotchas.md")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	for _, topic := range []string{"patterns", "decisions", "architecture", "memory"} {
		os.WriteFile(filepath.Join(brainDir, topic+".md"), []byte(""), 0600)
	}

	groups, err := FindDuplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Duplicates) != 1 {
		t.Errorf("expected 1 duplicate, got %d", len(groups[0].Duplicates))
	}
	if groups[0].Kept.LineNum != 1 {
		t.Errorf("expected kept at line 1, got %d", groups[0].Kept.LineNum)
	}
	if groups[0].IsCrossTopic {
		t.Error("expected not cross-topic")
	}
}

func TestFindDuplicates_CrossTopic(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	gotchasContent := "### [2026-01-01 00:00:00] Use argon2 NOT bcrypt\n"
	patternsContent := "### [2026-01-01 00:00:00] use argon2 not bcrypt\n"

	os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(gotchasContent), 0600)
	os.WriteFile(filepath.Join(brainDir, "patterns.md"), []byte(patternsContent), 0600)
	for _, topic := range []string{"decisions", "architecture", "memory"} {
		os.WriteFile(filepath.Join(brainDir, topic+".md"), []byte(""), 0600)
	}

	groups, err := FindDuplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if !groups[0].IsCrossTopic {
		t.Error("expected cross-topic duplicate")
	}
	if groups[0].Kept.Topic != "gotchas" {
		t.Errorf("expected kept in gotchas (alphabetically first), got %s", groups[0].Kept.Topic)
	}
}

func TestRunDedup_DryRun(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	content := `### [2026-01-01 00:00:00] Use filepath.Join

### [2026-01-01 00:00:01] Duplicate entry

### [2026-01-01 00:00:02] duplicate entry
`
	path := filepath.Join(brainDir, "gotchas.md")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{"patterns", "decisions", "architecture", "memory"} {
		os.WriteFile(filepath.Join(brainDir, topic+".md"), []byte(""), 0600)
	}

	report, err := RunDedup(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(report.Groups))
	}
	if report.TotalRemoved != 1 {
		t.Errorf("expected 1 removed, got %d", report.TotalRemoved)
	}
	if report.ArchivedPath != "" {
		t.Error("dry run should not create archive")
	}

	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Error("dry run should not modify files")
	}
}

func TestRunDedup_ActualDedup(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	content := "### [2026-01-01 00:00:00] Use filepath.Join\n\n### [2026-01-01 00:00:01] duplicate entry one\n\n### [2026-01-01 00:00:02] duplicate entry one\n"
	path := filepath.Join(brainDir, "gotchas.md")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{"patterns", "decisions", "architecture", "memory"} {
		os.WriteFile(filepath.Join(brainDir, topic+".md"), []byte(""), 0600)
	}

	report, err := RunDedup(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(report.Groups))
	}
	if report.TotalRemoved != 1 {
		t.Errorf("expected 1 removed (one duplicate of the same entry), got %d", report.TotalRemoved)
	}
	if report.ArchivedPath == "" {
		t.Error("expected archive path")
	}

	if _, err := os.Stat(report.ArchivedPath); os.IsNotExist(err) {
		t.Errorf("archive file not created: %s", report.ArchivedPath)
	}

	data, _ := os.ReadFile(path)
	count := strings.Count(string(data), "duplicate entry one")
	if count != 1 {
		t.Errorf("expected 1 occurrence of duplicate entry (the kept one), got %d", count)
	}
}

func TestRunDedup_EmptyFiles(t *testing.T) {
	brainDir := setupTestBrainDir(t)

	for _, topic := range []string{"gotchas", "patterns", "decisions", "architecture", "memory"} {
		os.WriteFile(filepath.Join(brainDir, topic+".md"), []byte(""), 0600)
	}

	report, err := RunDedup(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(report.Groups))
	}
}
