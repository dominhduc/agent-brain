package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrigrams(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 3},
		{"ab", 0},
		{"abc", 1},
		{"abcd", 2},
		{"", 0},
		{"  hello  ", 3},
	}

	for _, tc := range tests {
		ts := trigrams(tc.input)
		if tc.expected == 0 {
			if ts != nil {
				t.Errorf("trigrams(%q): expected nil, got %d trigrams", tc.input, len(ts))
			}
		} else {
			if len(ts) != tc.expected {
				t.Errorf("trigrams(%q): expected %d trigrams, got %d", tc.input, tc.expected, len(ts))
			}
		}
	}
}

func TestTrigramJaccard(t *testing.T) {
	tests := []struct {
		a, b     string
		minScore float64
		maxScore float64
	}{
		{"unified api key configuration key renamed from nested llm.api_key to flat api-key",
			"unified api key configuration key renamed from nested llm.api_key to flat api-key.",
			0.95, 1.0},
		{"unified api key configuration key renamed from nested llm.api_key to flat api-key",
			"unified api key configuration renamed from nested structure to flat api-key",
			0.55, 0.80},
		{"version bump in code matches entry in changelog.md",
			"version bump in code matches changelog.md",
			0.75, 1.0},
		{"unified api key configuration key renamed from nested llm.api_key to flat api-key",
			"goreleaser names binaries as brain_Linux_x86_64",
			0.0, 0.05},
		{"hello", "hello", 1.0, 1.0},
		{"", "", 0.0, 0.0},
	}

	for _, tc := range tests {
		score := trigramJaccard(tc.a, tc.b)
		if score < tc.minScore || score > tc.maxScore {
			t.Errorf("trigramJaccard(%q, %q): expected %.2f-%.2f, got %.3f",
				tc.a, tc.b, tc.minScore, tc.maxScore, score)
		}
	}
}

func TestFindFuzzyDuplicates(t *testing.T) {
	dir := t.TempDir()
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `### [2025-01-01 10:00:00] Unified API key configuration renamed from nested structure to flat api-key.
### [2025-01-01 11:00:00] Unified API key configuration key renamed from nested llm.api_key to flat api-key.
### [2025-01-01 12:00:00] Completely different entry about something else.
### [2025-01-01 13:00:00] Version bump in code matches entry in changelog.md.
### [2025-01-01 14:00:00] Version bump in code matches changelog.md.
`

	topicPath := filepath.Join(dir, "decisions.md")
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	groups, err := h.FindFuzzyDuplicates(0.5)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) < 2 {
		t.Errorf("expected at least 2 fuzzy duplicate groups, got %d", len(groups))
	}

	for _, g := range groups {
		if len(g.Duplicates) == 0 {
			t.Error("expected non-empty Duplicates in each group")
		}
	}
}

func TestFindFuzzyDuplicatesNoMatches(t *testing.T) {
	dir := t.TempDir()
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `### [2025-01-01 10:00:00] First unique entry.
### [2025-01-01 11:00:00] Second completely different entry.
### [2025-01-01 12:00:00] Third unrelated entry.
`

	topicPath := filepath.Join(dir, "decisions.md")
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	groups, err := h.FindFuzzyDuplicates(0.9)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 0 {
		t.Errorf("expected 0 fuzzy duplicate groups at threshold 0.9, got %d", len(groups))
	}
}

func TestRunFuzzyDedup(t *testing.T) {
	dir := t.TempDir()
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `### [2025-01-01 10:00:00] Unified API key configuration renamed from nested structure to flat api-key.
### [2025-01-01 11:00:00] Unified API key configuration key renamed from nested llm.api_key to flat api-key.
### [2025-01-01 12:00:00] Completely different entry about something else.
`

	topicPath := filepath.Join(dir, "decisions.md")
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	report, err := h.RunFuzzyDedup(false, 0.5)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Groups) == 0 {
		t.Error("expected at least 1 fuzzy duplicate group")
	}

	if report.TotalRemoved == 0 {
		t.Error("expected at least 1 entry removed")
	}

	data, err := os.ReadFile(topicPath)
	if err != nil {
		t.Fatal(err)
	}

	lines := 0
	for _, line := range string(data) {
		if line == '\n' {
			lines++
		}
	}

	originalLines := 0
	for _, line := range content {
		if line == '\n' {
			originalLines++
		}
	}

	if lines >= originalLines {
		t.Error("expected fewer lines after dedup")
	}
}

func TestRunFuzzyDedupDryRun(t *testing.T) {
	dir := t.TempDir()
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	content := `### [2025-01-01 10:00:00] Unified API key configuration renamed from nested structure to flat api-key.
### [2025-01-01 11:00:00] Unified API key configuration key renamed from nested llm.api_key to flat api-key.
### [2025-01-01 12:00:00] Completely different entry about something else.
`

	topicPath := filepath.Join(dir, "decisions.md")
	if err := os.WriteFile(topicPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	report, err := h.RunFuzzyDedup(true, 0.5)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Groups) == 0 {
		t.Error("expected at least 1 fuzzy duplicate group")
	}

	data, err := os.ReadFile(topicPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != content {
		t.Error("dry run should not modify files")
	}
}
