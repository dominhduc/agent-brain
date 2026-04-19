package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestBrainWithEntries(t *testing.T, entries map[string][]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	for topic, messages := range entries {
		path := filepath.Join(brainDir, topic+".md")
		var content string
		for _, msg := range messages {
			content += "### [2026-04-15 10:00:00] " + msg + "\n"
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	idx := &Index{
		Version: indexVersion,
		Entries: make(map[string]IndexEntry),
	}
	for topic, messages := range entries {
		for _, msg := range messages {
			key := MakeKey(topic, "2026-04-15 10:00:00")
			idx.Entries[key] = IndexEntry{
				Strength:       1.0,
				RetrievalCount: 0,
				LastRetrieved:  idx.LastRebuild,
				HalfLifeDays:   14,
				Confidence:     "observed",
				Topics:         DetectTopics(msg),
			}
		}
	}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	return brainDir
}

func TestRetrieveWithBudget_Basic(t *testing.T) {
	brainDir := setupTestBrainWithEntries(t, map[string][]string{
		"gotchas":   {"Error A", "Error B", "Error C"},
		"patterns":  {"Pattern X", "Pattern Y"},
		"decisions": {"Decision 1"},
	})

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	budget := DefaultBudget()
	budget.MaxTokens = 500
	result, err := hub.RetrieveWithBudget(budget, nil)
	if err != nil {
		t.Fatalf("RetrieveWithBudget failed: %v", err)
	}

	if len(result.Entries) == 0 {
		t.Error("expected entries, got none")
	}
	if result.TotalTokens <= 0 {
		t.Error("expected positive token count")
	}
	if result.BudgetUsed <= 0 || result.BudgetUsed > 1 {
		t.Errorf("budget used out of range: %.2f", result.BudgetUsed)
	}
}

func TestRetrieveWithBudget_RespectsMaxTokens(t *testing.T) {
	brainDir := setupTestBrainWithEntries(t, map[string][]string{
		"gotchas":  {"Error A", "Error B", "Error C", "Error D", "Error E"},
		"patterns": {"Pattern X", "Pattern Y", "Pattern Z"},
	})

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	budget := DefaultBudget()
	budget.MaxTokens = 100
	result, err := hub.RetrieveWithBudget(budget, nil)
	if err != nil {
		t.Fatalf("RetrieveWithBudget failed: %v", err)
	}

	if result.TotalTokens > 100 {
		t.Errorf("exceeded token budget: got %d, want <= 100", result.TotalTokens)
	}
}

func TestRetrieveWithBudget_RespectsMaxEntries(t *testing.T) {
	brainDir := setupTestBrainWithEntries(t, map[string][]string{
		"gotchas":  {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"},
		"patterns": {"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T"},
	})

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	budget := DefaultBudget()
	budget.MaxTokens = 10000
	budget.MaxEntries = 5
	result, err := hub.RetrieveWithBudget(budget, nil)
	if err != nil {
		t.Fatalf("RetrieveWithBudget failed: %v", err)
	}

	if len(result.Entries) > 5 {
		t.Errorf("exceeded max entries: got %d, want <= 5", len(result.Entries))
	}
}

func TestRetrieveWithBudget_ContextBoost(t *testing.T) {
	brainDir := setupTestBrainWithEntries(t, map[string][]string{
		"gotchas":  {"JWT token expires without refresh", "filepath.Join errors on Windows"},
		"patterns": {"All handlers use middleware chain"},
	})

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	budget := DefaultBudget()
	budget.MaxTokens = 10000

	resultWithContext, err := hub.RetrieveWithBudget(budget, []string{"security"})
	if err != nil {
		t.Fatalf("RetrieveWithBudget failed: %v", err)
	}

	if len(resultWithContext.Entries) == 0 {
		t.Error("expected entries with context")
	}
}

func TestRetrieveWithBudget_MinStrength(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	if err := os.MkdirAll(brainDir, 0700); err != nil {
		t.Fatal(err)
	}

	oldTimestamp := "2020-01-01 10:00:00"
	oldTime, _ := time.Parse("2006-01-02 15:04:05", oldTimestamp)
	content := "### [" + oldTimestamp + "] High strength entry\n"
	if err := os.WriteFile(filepath.Join(brainDir, "gotchas.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	idx := &Index{Version: indexVersion, Entries: make(map[string]IndexEntry)}
	idx.Entries[MakeKey("gotchas", oldTimestamp)] = IndexEntry{
		Strength:       1.0,
		RetrievalCount: 0,
		LastRetrieved:  oldTime,
		HalfLifeDays:   14,
		Confidence:     "observed",
		Topics:         []string{"general"},
	}
	if err := idx.Save(brainDir); err != nil {
		t.Fatal(err)
	}

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatalf("failed to open hub: %v", err)
	}

	budget := DefaultBudget()
	budget.MinStrength = 0.99
	result, err := hub.RetrieveWithBudget(budget, nil)
	if err != nil {
		t.Fatalf("RetrieveWithBudget failed: %v", err)
	}

	if len(result.Entries) > 0 {
		t.Error("expected no entries with high min strength threshold on old entries")
	}
}

func TestBudgetResult_Summary(t *testing.T) {
	result := &BudgetResult{
		Entries: []ScoredEntry{
			{Topic: "gotchas", Tokens: 100},
			{Topic: "gotchas", Tokens: 150},
			{Topic: "patterns", Tokens: 200},
		},
		TotalTokens: 450,
		Skipped:     10,
	}

	summary := result.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(summary, "gotchas") {
		t.Error("summary should mention gotchas")
	}
	if !strings.Contains(summary, "patterns") {
		t.Error("summary should mention patterns")
	}
}

func TestDefaultBudget(t *testing.T) {
	budget := DefaultBudget()
	if budget.MaxTokens != 3000 {
		t.Errorf("expected MaxTokens=3000, got %d", budget.MaxTokens)
	}
	if budget.MinStrength != 0.15 {
		t.Errorf("expected MinStrength=0.15, got %.2f", budget.MinStrength)
	}
	if budget.MaxEntries != 50 {
		t.Errorf("expected MaxEntries=50, got %d", budget.MaxEntries)
	}
	if budget.IncludeRecentDays != 7 {
		t.Errorf("expected IncludeRecentDays=7, got %d", budget.IncludeRecentDays)
	}
	totalPct := budget.GotchasPct + budget.PatternsPct + budget.DecisionsPct + budget.ArchitecturePct + budget.RemainingPct
	if totalPct != 1.0 {
		t.Errorf("budget percentages don't sum to 1.0: got %.2f", totalPct)
	}
}

func TestRetrieveWithBudget_MultiTopic(t *testing.T) {
	tmpDir := t.TempDir()
	brainDir := filepath.Join(tmpDir, ".brain")
	os.MkdirAll(brainDir, 0755)

	for _, topic := range []string{"gotchas", "patterns", "decisions", "architecture"} {
		filename := topic + ".md"
		if topic == "memory" {
			filename = "MEMORY.md"
		}
		content := fmt.Sprintf("# %s\n\n### [2026-04-15 10:00:00] Test %s entry one\n\n### [2026-04-16 10:00:00] Test %s entry two\n", topic, topic, topic)
		os.WriteFile(filepath.Join(brainDir, filename), []byte(content), 0600)
	}

	hub, err := Open(brainDir)
	if err != nil {
		t.Fatal(err)
	}

	budget := DefaultBudget()
	budget.MaxTokens = 2000
	result, err := hub.RetrieveWithBudget(budget, nil)
	if err != nil {
		t.Fatal(err)
	}

	topicSet := make(map[string]bool)
	for _, e := range result.Entries {
		topicSet[e.Topic] = true
	}

	if len(topicSet) < 3 {
		t.Errorf("expected entries from at least 3 topics, got %d: %v", len(topicSet), topicSet)
	}
}
