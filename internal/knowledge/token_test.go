package knowledge

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minTokens int
		maxTokens int
	}{
		{"empty", "", 0, 0},
		{"single word", "hello", 1, 2},
		{"short phrase", "Use filepath.Join", 3, 5},
		{"code snippet", "func (h *Hub) SaveIndex(idx *Index) error {", 8, 12},
		{"prose sentence", "This is a test sentence with about ten words in it.", 12, 16},
		{"mixed code and prose", "Use filepath.Join for paths. Always check the error return value.", 13, 18},
		{"markdown entry", "### [2026-04-03 08:42:10] Use filepath.Join, not string concatenation", 10, 14},
		{"long paragraph", strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10), 98, 130},
		{"numbers and symbols", "JWT returns 401 on expiry. Use bcrypt for passwords.", 10, 14},
		{"underscores and dashes", "use snake_case variables and kebab-case flags", 7, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want [%d, %d]",
					tt.input, result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimateTokens_Proportional(t *testing.T) {
	short := "the quick brown fox jumps over the lazy dog"
	long := strings.Repeat(short+" ", 10)

	shortTokens := EstimateTokens(short)
	longTokens := EstimateTokens(long)

	if longTokens < shortTokens*9 || longTokens > shortTokens*11 {
		t.Errorf("Expected ~10x ratio, got %.2f (%d vs %d)", float64(longTokens)/float64(shortTokens), longTokens, shortTokens)
	}
}

func TestEstimateTokens_ZeroForWhitespace(t *testing.T) {
	result := EstimateTokens("   \n\t  ")
	if result != 0 {
		t.Errorf("EstimateTokens(whitespace) = %d, want 0", result)
	}
}
