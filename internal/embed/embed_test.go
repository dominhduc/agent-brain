package embed

import (
	"testing"
)

func TestNoneProvider(t *testing.T) {
	p := &NoneProvider{}

	_, err := p.Embed("test")
	if err == nil {
		t.Error("NoneProvider.Embed should return error")
	}

	_, err = p.EmbedBatch([]string{"test"})
	if err == nil {
		t.Error("NoneProvider.EmbedBatch should return error")
	}

	if p.Dimensions() != 0 {
		t.Errorf("NoneProvider.Dimensions() = %d, want 0", p.Dimensions())
	}

	if p.Name() != "none" {
		t.Errorf("NoneProvider.Name() = %q, want 'none'", p.Name())
	}
}

func TestRRFScore(t *testing.T) {
	keywordResults := []string{"a", "b", "c"}
	vectorResults := []SearchResult{
		{Key: "b"},
		{Key: "d"},
		{Key: "a"},
	}

	scoreA := RRFScore(keywordResults, vectorResults, "a")
	scoreB := RRFScore(keywordResults, vectorResults, "b")
	scoreX := RRFScore(keywordResults, vectorResults, "x")

	if scoreB <= scoreA {
		t.Errorf("expected scoreB > scoreA (b ranks higher in both), got %.4f vs %.4f", scoreB, scoreA)
	}

	if scoreX != 0 {
		t.Errorf("expected scoreX = 0 (not in any results), got %.4f", scoreX)
	}
}

func TestHybridSearch(t *testing.T) {
	keywordResults := []string{"a", "b", "c", "d"}
	vectorResults := []SearchResult{
		{Key: "c"},
		{Key: "a"},
		{Key: "e"},
	}

	results := HybridSearch("test", keywordResults, vectorResults, 10)

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	if len(results) > 0 && results[0].Key != "a" && results[0].Key != "c" {
		t.Errorf("expected 'a' or 'c' as top result (high in both), got %s", results[0].Key)
	}
}

func TestHybridSearch_TopKLimits(t *testing.T) {
	keywordResults := []string{"a", "b", "c", "d", "e", "f"}
	vectorResults := []SearchResult{
		{Key: "a"},
		{Key: "b"},
		{Key: "c"},
		{Key: "d"},
		{Key: "e"},
		{Key: "f"},
	}

	results := HybridSearch("test", keywordResults, vectorResults, 3)

	if len(results) != 3 {
		t.Errorf("expected 3 results (topK=3), got %d", len(results))
	}
}
