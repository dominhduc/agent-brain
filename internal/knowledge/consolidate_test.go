package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClusterEntries_Basic(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use filepath.Join for joining file paths, not string concatenation with slashes"},
		{Timestamp: "2026-04-16 11:00:00", Message: "Use filepath.Join for file paths instead of string concatenation with slashes"},
		{Timestamp: "2026-04-17 12:00:00", Message: "Always use JWT for authentication"},
	}

	clusters := ClusterEntries(entries, "gotchas")

	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster (2 filepath entries), got %d", len(clusters))
	}

	if len(clusters) > 0 && len(clusters[0].Members) != 2 {
		t.Errorf("expected cluster with 2 members, got %d", len(clusters[0].Members))
	}
}

func TestClusterEntries_NoClusters(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use JWT for authentication"},
		{Timestamp: "2026-04-16 11:00:00", Message: "Deploy to production on Fridays"},
		{Timestamp: "2026-04-17 12:00:00", Message: "Use React for UI components"},
	}

	clusters := ClusterEntries(entries, "gotchas")

	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestClusterEntries_AllCluster(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use filepath.Join for joining file paths, not string concatenation"},
		{Timestamp: "2026-04-16 11:00:00", Message: "Use filepath.Join for file paths instead of string concatenation"},
		{Timestamp: "2026-04-17 12:00:00", Message: "Always use filepath.Join for paths, never use string concatenation"},
	}

	clusters := ClusterEntries(entries, "gotchas")

	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	if len(clusters) > 0 && len(clusters[0].Members) != 3 {
		t.Errorf("expected cluster with 3 members, got %d", len(clusters[0].Members))
	}
}

func TestConsolidateCluster_Basic(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use filepath.Join for paths. Always check errors."},
		{Timestamp: "2026-04-16 11:00:00", Message: "Never use string concatenation."},
	}

	result := ConsolidateCluster(entries)

	if !strings.Contains(result, "filepath.Join") {
		t.Error("consolidated message should contain filepath.Join")
	}
	if !strings.Contains(result, "string concatenation") {
		t.Error("consolidated message should contain string concatenation")
	}
}

func TestConsolidateCluster_DeduplicatesSentences(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use filepath.Join for paths."},
		{Timestamp: "2026-04-16 11:00:00", Message: "Use filepath.Join for paths."},
	}

	result := ConsolidateCluster(entries)

	count := strings.Count(result, "filepath.Join")
	if count != 1 {
		t.Errorf("expected 1 occurrence of filepath.Join, got %d", count)
	}
}

func TestConsolidateCluster_Empty(t *testing.T) {
	result := ConsolidateCluster(nil)
	if result != "" {
		t.Errorf("expected empty string for nil input, got %q", result)
	}
}

func TestExtractSentences(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"One sentence", 1},
		{"Two sentences. Three sentences.", 2},
		{"Question? Yes!", 2},
		{"", 0},
		{"   ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sentences := extractSentences(tt.input)
			if len(sentences) != tt.expected {
				t.Errorf("extractSentences(%q) = %d sentences, want %d", tt.input, len(sentences), tt.expected)
			}
		})
	}
}

func TestIsSentenceDuplicate(t *testing.T) {
	existing := []string{
		"Use filepath.Join for paths.",
		"Always check error returns.",
	}

	tests := []struct {
		sentence string
		expected bool
	}{
		{"Use filepath.Join for paths.", true},
		{"use filepath.join for paths", true},
		{"Always check error returns.", true},
		{"Use bcrypt for passwords.", false},
	}

	for _, tt := range tests {
		t.Run(tt.sentence, func(t *testing.T) {
			result := isSentenceDuplicate(tt.sentence, existing)
			if result != tt.expected {
				t.Errorf("isSentenceDuplicate(%q) = %v, want %v", tt.sentence, result, tt.expected)
			}
		})
	}
}

func TestFindConsolidations(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := `# Gotchas

### [2026-04-15 10:00:00] Use filepath.Join for joining file paths, not string concatenation.

### [2026-04-16 11:00:00] Use filepath.Join for file paths instead of string concatenation.

### [2026-04-17 12:00:00] Use JWT for authentication.
`
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	idx.Entries["gotchas:2026-04-15 10:00:00"] = IndexEntry{Strength: 0.8, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed", Version: 1}
	idx.Entries["gotchas:2026-04-16 11:00:00"] = IndexEntry{Strength: 0.7, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed", Version: 1}
	idx.Entries["gotchas:2026-04-17 12:00:00"] = IndexEntry{Strength: 0.9, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed", Version: 1}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, _ := Open(brainDir)
	proposals, err := hub.FindConsolidations()
	if err != nil {
		t.Fatalf("FindConsolidations failed: %v", err)
	}

	if len(proposals) != 1 {
		t.Errorf("expected 1 proposal (2 filepath entries), got %d", len(proposals))
	}

	if len(proposals) > 0 {
		p := proposals[0]
		if len(p.Sources) != 2 {
			t.Errorf("expected 2 sources, got %d", len(p.Sources))
		}
		if p.Topic != "gotchas" {
			t.Errorf("expected topic 'gotchas', got %q", p.Topic)
		}
	}
}

func TestApplyConsolidation(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := `# Gotchas

### [2026-04-15 10:00:00] Use filepath.Join for joining file paths, not string concatenation.

### [2026-04-16 11:00:00] Always check errors.

### [2026-04-17 12:00:00] Use JWT for auth.
`
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	idx.Entries["gotchas:2026-04-15 10:00:00"] = IndexEntry{Strength: 0.8, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed"}
	idx.Entries["gotchas:2026-04-16 11:00:00"] = IndexEntry{Strength: 0.7, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed"}
	idx.Entries["gotchas:2026-04-17 12:00:00"] = IndexEntry{Strength: 0.9, LastRetrieved: time.Now(), HalfLifeDays: 14, Confidence: "observed"}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, _ := Open(brainDir)
	proposals, _ := hub.FindConsolidations()

	if len(proposals) > 0 {
		if err := hub.ApplyConsolidation(proposals[0]); err != nil {
			t.Fatalf("ApplyConsolidation failed: %v", err)
		}

		data, _ := os.ReadFile(filepath.Join(brainDir, "gotchas.md"))
		content := string(data)

		if !strings.Contains(content, "<!-- Source timeline:") {
			t.Error("expected timeline HTML comment")
		}

		updatedIdx, _ := LoadIndex(brainDir)
		hasConsolidated := false
		for key, entry := range updatedIdx.Entries {
			if strings.HasPrefix(key, "gotchas:") && entry.Confidence == "verified" {
				hasConsolidated = true
			}
		}
		if !hasConsolidated {
			t.Error("expected consolidated entry with verified confidence")
		}
	}
}

func TestClusterEntries_UnrelatedNotClustered(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-01-01 00:00:00", Message: "Go const cannot be overridden by ldflags"},
		{Timestamp: "2026-01-02 00:00:00", Message: "GitHub private repos require GITHUB_TOKEN for downloads"},
		{Timestamp: "2026-01-03 00:00:00", Message: "Always handle os.UserHomeDir errors explicitly"},
	}
	clusters := ClusterEntries(entries, "gotchas")
	if len(clusters) != 0 {
		t.Errorf("unrelated entries should not cluster, got %d clusters", len(clusters))
		for i, c := range clusters {
			t.Logf("  cluster %d: %d members", i, len(c.Members))
		}
	}
}

func TestClusterEntries_AvgStrengthPopulated(t *testing.T) {
	entries := []TopicEntry{
		{Timestamp: "2026-04-15 10:00:00", Message: "Use filepath.Join for joining file paths, not string concatenation with slashes"},
		{Timestamp: "2026-04-16 11:00:00", Message: "Use filepath.Join for file paths instead of string concatenation with slashes"},
	}

	clusters := ClusterEntries(entries, "gotchas")
	if len(clusters) == 0 {
		t.Fatal("expected at least 1 cluster")
	}
	if clusters[0].AvgStrength <= 0 {
		t.Errorf("expected positive AvgStrength, got %.2f", clusters[0].AvgStrength)
	}
}
